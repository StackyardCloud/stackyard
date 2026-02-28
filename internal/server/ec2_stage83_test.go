package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awsec2 "github.com/aws/aws-sdk-go-v2/service/ec2"
)

func TestEC2Stage83SDKLifecycle(t *testing.T) {
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

	certificateARN := "arn:aws:acm:us-east-1:123456789012:certificate/stage83"
	roleARN := "arn:aws:iam::123456789012:role/stage83"

	out, err := client.AssociateEnclaveCertificateIamRole(ctx, &awsec2.AssociateEnclaveCertificateIamRoleInput{
		CertificateArn: aws.String(certificateARN),
		RoleArn:        aws.String(roleARN),
	})
	if err != nil {
		t.Fatalf("associate enclave certificate iam role: %v", err)
	}
	if aws.ToString(out.CertificateS3BucketName) == "" {
		t.Fatalf("expected certificate s3 bucket name")
	}
	if !strings.Contains(aws.ToString(out.CertificateS3ObjectKey), roleARN) {
		t.Fatalf("expected certificate s3 object key to include role arn, got %q", aws.ToString(out.CertificateS3ObjectKey))
	}
	if aws.ToString(out.EncryptionKmsKeyId) == "" {
		t.Fatalf("expected encryption kms key id")
	}
}

func TestEC2Stage83ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"AssociateEnclaveCertificateIamRole",
	}

	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, action := range implemented {
		params := map[string]string{
			"CertificateArn": "arn:aws:acm:us-east-1:123456789012:certificate/stage83",
			"RoleArn":        "arn:aws:iam::123456789012:role/stage83",
		}
		resp := ec2Request(t, ts, action, params)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action)
		}
	}
}
