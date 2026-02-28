package server

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awsec2 "github.com/aws/aws-sdk-go-v2/service/ec2"
	awsec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

func TestEC2Stage87SDKLifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(testRegion),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(testAccessKey, testSecretKey, "")),
	)
	if err != nil {
		t.Fatalf("load aws config: %v", err)
	}
	client := awsec2.NewFromConfig(cfg, func(o *awsec2.Options) {
		o.BaseEndpoint = aws.String(ts.URL)
	})

	out, err := client.BundleInstance(ctx, &awsec2.BundleInstanceInput{
		InstanceId: aws.String("i-00000000000000087"),
		Storage: &awsec2types.Storage{
			S3: &awsec2types.S3Storage{
				AWSAccessKeyId:        aws.String(testAccessKey),
				Bucket:                aws.String("stage87-bucket"),
				Prefix:                aws.String("stage87-prefix"),
				UploadPolicy:          []byte("stage87-policy"),
				UploadPolicySignature: aws.String("stage87-signature"),
			},
		},
	})
	if err != nil {
		t.Fatalf("bundle instance: %v", err)
	}
	if out.BundleTask == nil {
		t.Fatalf("expected bundle task in response")
	}
	if aws.ToString(out.BundleTask.BundleId) == "" {
		t.Fatalf("expected bundle id")
	}
	if aws.ToString(out.BundleTask.InstanceId) != "i-00000000000000087" {
		t.Fatalf("unexpected instance id: %q", aws.ToString(out.BundleTask.InstanceId))
	}
	if out.BundleTask.State != awsec2types.BundleTaskStatePending {
		t.Fatalf("unexpected bundle task state: %q", out.BundleTask.State)
	}
	if out.BundleTask.Storage == nil || out.BundleTask.Storage.S3 == nil {
		t.Fatalf("expected storage.s3 in bundle task")
	}
	if aws.ToString(out.BundleTask.Storage.S3.Bucket) != "stage87-bucket" {
		t.Fatalf("unexpected s3 bucket: %q", aws.ToString(out.BundleTask.Storage.S3.Bucket))
	}
	if aws.ToString(out.BundleTask.Storage.S3.Prefix) != "stage87-prefix" {
		t.Fatalf("unexpected s3 prefix: %q", aws.ToString(out.BundleTask.Storage.S3.Prefix))
	}
	if !bytes.Equal(out.BundleTask.Storage.S3.UploadPolicy, []byte("stage87-policy")) {
		t.Fatalf("unexpected upload policy: %q", string(out.BundleTask.Storage.S3.UploadPolicy))
	}
}

func TestEC2Stage87ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"BundleInstance",
	}

	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, action := range implemented {
		params := map[string]string{
			"InstanceId":                       "i-00000000000000087",
			"Storage.S3.Bucket":                "stage87-bucket",
			"Storage.S3.Prefix":                "stage87-prefix",
			"Storage.S3.AWSAccessKeyId":        testAccessKey,
			"Storage.S3.UploadPolicy":          "c3RhZ2U4Ny1wb2xpY3k=",
			"Storage.S3.UploadPolicySignature": "stage87-signature",
		}
		resp := ec2Request(t, ts, action, params)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action)
		}
	}
}
