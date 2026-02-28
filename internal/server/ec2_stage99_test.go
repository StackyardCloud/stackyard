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

func TestEC2Stage99SDKLifecycle(t *testing.T) {
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

	out, err := client.CopyFpgaImage(ctx, &awsec2.CopyFpgaImageInput{
		SourceFpgaImageId: aws.String("afi-source-stage99"),
		SourceRegion:      aws.String("us-east-1"),
		Name:              aws.String("stage99-copy"),
		Description:       aws.String("stage99 copy"),
	})
	if err != nil {
		t.Fatalf("copy fpga image: %v", err)
	}
	if out.FpgaImageId == nil || strings.TrimSpace(aws.ToString(out.FpgaImageId)) == "" {
		t.Fatalf("expected fpga image id to be set")
	}
	if !strings.HasPrefix(aws.ToString(out.FpgaImageId), "afi-") {
		t.Fatalf("unexpected fpga image id: %q", aws.ToString(out.FpgaImageId))
	}
}

func TestEC2Stage99ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"CopyFpgaImage",
	}

	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, action := range implemented {
		params := map[string]string{
			"SourceFpgaImageId": "afi-source-stage99",
			"SourceRegion":      "us-east-1",
		}
		resp := ec2Request(t, ts, action, params)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action)
		}
	}
}
