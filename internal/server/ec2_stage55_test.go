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

func TestEC2Stage55SDKLifecycle(t *testing.T) {
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

	out, err := client.CreateVpcEndpointServiceConfiguration(ctx, &awsec2.CreateVpcEndpointServiceConfigurationInput{
		AcceptanceRequired:      aws.Bool(true),
		ClientToken:             aws.String("stage55-token"),
		NetworkLoadBalancerArns: []string{"arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/net/stage55/1234567890abcdef"},
		PrivateDnsName:          aws.String("svc.stage55.internal"),
		SupportedIpAddressTypes: []string{"ipv4", "ipv6"},
		SupportedRegions:        []string{"us-east-1", "us-west-2"},
	})
	if err != nil {
		t.Fatalf("create vpc endpoint service configuration: %v", err)
	}
	if out.ServiceConfiguration == nil {
		t.Fatalf("expected service configuration in response")
	}
	if aws.ToString(out.ServiceConfiguration.ServiceId) == "" {
		t.Fatalf("expected non-empty service id")
	}
	if aws.ToString(out.ServiceConfiguration.ServiceName) == "" {
		t.Fatalf("expected non-empty service name")
	}
	if out.ServiceConfiguration.ServiceState != awsec2types.ServiceStateAvailable {
		t.Fatalf("unexpected service state: %q", out.ServiceConfiguration.ServiceState)
	}
	if aws.ToString(out.ClientToken) != "stage55-token" {
		t.Fatalf("unexpected client token: %q", aws.ToString(out.ClientToken))
	}
}

func TestEC2Stage55ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"CreateVpcEndpointServiceConfiguration",
	}

	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, action := range implemented {
		resp := ec2Request(t, ts, action, map[string]string{
			"NetworkLoadBalancerArn.1": "arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/net/stage55/1234567890abcdef",
			"SupportedIpAddressType.1": "ipv4",
		})
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action)
		}
	}
}
