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
	awsec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

func TestEC2Stage105SDKLifecycle(t *testing.T) {
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

	out, err := client.CreateCarrierGateway(ctx, &awsec2.CreateCarrierGatewayInput{
		VpcId: aws.String("vpc-00000001"),
		TagSpecifications: []awsec2types.TagSpecification{
			{
				ResourceType: awsec2types.ResourceTypeCarrierGateway,
				Tags: []awsec2types.Tag{
					{Key: aws.String("env"), Value: aws.String("stage105")},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("create carrier gateway: %v", err)
	}
	if out.CarrierGateway == nil {
		t.Fatalf("expected carrier gateway in response")
	}
	if !strings.HasPrefix(aws.ToString(out.CarrierGateway.CarrierGatewayId), "cagw-") {
		t.Fatalf("unexpected carrier gateway id: %q", aws.ToString(out.CarrierGateway.CarrierGatewayId))
	}
	if aws.ToString(out.CarrierGateway.VpcId) != "vpc-00000001" {
		t.Fatalf("unexpected carrier gateway vpc id: %q", aws.ToString(out.CarrierGateway.VpcId))
	}
	if out.CarrierGateway.State != awsec2types.CarrierGatewayStateAvailable {
		t.Fatalf("unexpected carrier gateway state: %q", out.CarrierGateway.State)
	}
	if len(out.CarrierGateway.Tags) != 1 {
		t.Fatalf("expected 1 tag, got %d", len(out.CarrierGateway.Tags))
	}
}

func TestEC2Stage105ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"CreateCarrierGateway",
	}

	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, action := range implemented {
		params := map[string]string{
			"VpcId": "vpc-00000001",
		}
		resp := ec2Request(t, ts, action, params)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action)
		}
	}
}
