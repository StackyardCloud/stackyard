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
)

func TestEC2Stage106SDKLifecycle(t *testing.T) {
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

	out, err := client.CreateCoipCidr(ctx, &awsec2.CreateCoipCidrInput{
		Cidr:       aws.String("10.0.0.0/24"),
		CoipPoolId: aws.String("coip-pool-00000001"),
	})
	if err != nil {
		t.Fatalf("create coip cidr: %v", err)
	}
	if out.CoipCidr == nil {
		t.Fatalf("expected coip cidr in response")
	}
	if aws.ToString(out.CoipCidr.Cidr) != "10.0.0.0/24" {
		t.Fatalf("unexpected coip cidr: %q", aws.ToString(out.CoipCidr.Cidr))
	}
	if aws.ToString(out.CoipCidr.CoipPoolId) != "coip-pool-00000001" {
		t.Fatalf("unexpected coip pool id: %q", aws.ToString(out.CoipCidr.CoipPoolId))
	}
}

func TestEC2Stage106ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"CreateCoipCidr",
	}

	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, action := range implemented {
		resp := ec2Request(t, ts, action, map[string]string{
			"Cidr":       "10.0.0.0/24",
			"CoipPoolId": "coip-pool-00000001",
		})
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action)
		}
	}
}
