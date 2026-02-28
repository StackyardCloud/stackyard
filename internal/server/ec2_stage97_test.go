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

func TestEC2Stage97SDKLifecycle(t *testing.T) {
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

	out, err := client.CancelSpotInstanceRequests(ctx, &awsec2.CancelSpotInstanceRequestsInput{
		SpotInstanceRequestIds: []string{"sir-00000000000000097", "sir-00000000000000098"},
	})
	if err != nil {
		t.Fatalf("cancel spot instance requests: %v", err)
	}
	if len(out.CancelledSpotInstanceRequests) != 2 {
		t.Fatalf("expected 2 cancelled spot instance requests, got %d", len(out.CancelledSpotInstanceRequests))
	}

	first := out.CancelledSpotInstanceRequests[0]
	if aws.ToString(first.SpotInstanceRequestId) != "sir-00000000000000097" {
		t.Fatalf("unexpected first spot instance request id: %q", aws.ToString(first.SpotInstanceRequestId))
	}
	if first.State != awsec2types.CancelSpotInstanceRequestStateCancelled {
		t.Fatalf("unexpected first spot instance request state: %q", first.State)
	}
}

func TestEC2Stage97ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"CancelSpotInstanceRequests",
	}

	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, action := range implemented {
		params := map[string]string{
			"SpotInstanceRequestId.1": "sir-00000000000000097",
		}
		resp := ec2Request(t, ts, action, params)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action)
		}
	}
}
