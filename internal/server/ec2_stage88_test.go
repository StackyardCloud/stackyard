package server

import (
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

func TestEC2Stage88SDKLifecycle(t *testing.T) {
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

	bundleOut, err := client.BundleInstance(ctx, &awsec2.BundleInstanceInput{
		InstanceId: aws.String("i-00000000000000088"),
		Storage: &awsec2types.Storage{
			S3: &awsec2types.S3Storage{
				Bucket: aws.String("stage88-bucket"),
			},
		},
	})
	if err != nil {
		t.Fatalf("bundle instance: %v", err)
	}
	if bundleOut.BundleTask == nil || aws.ToString(bundleOut.BundleTask.BundleId) == "" {
		t.Fatalf("expected bundle task with id")
	}

	cancelOut, err := client.CancelBundleTask(ctx, &awsec2.CancelBundleTaskInput{
		BundleId: bundleOut.BundleTask.BundleId,
	})
	if err != nil {
		t.Fatalf("cancel bundle task: %v", err)
	}
	if cancelOut.BundleTask == nil {
		t.Fatalf("expected bundle task in cancel response")
	}
	if aws.ToString(cancelOut.BundleTask.BundleId) != aws.ToString(bundleOut.BundleTask.BundleId) {
		t.Fatalf("unexpected bundle id in cancel response: %q", aws.ToString(cancelOut.BundleTask.BundleId))
	}
	if cancelOut.BundleTask.State != awsec2types.BundleTaskStateCancelling {
		t.Fatalf("unexpected cancel bundle task state: %q", cancelOut.BundleTask.State)
	}
}

func TestEC2Stage88ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"CancelBundleTask",
	}

	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, action := range implemented {
		params := map[string]string{
			"BundleId": "bun-00000088",
		}
		resp := ec2Request(t, ts, action, params)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action)
		}
	}
}
