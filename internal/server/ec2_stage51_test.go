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

func TestEC2Stage51SDKLifecycle(t *testing.T) {
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

	modifyOut, err := client.ModifyVpcEndpointServiceConfiguration(ctx, &awsec2.ModifyVpcEndpointServiceConfigurationInput{
		ServiceId:                  aws.String("vpce-svc-00000000"),
		AcceptanceRequired:         aws.Bool(true),
		AddNetworkLoadBalancerArns: []string{"arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/net/demo/1234567890abcdef"},
		AddSupportedIpAddressTypes: []string{"ipv6"},
		AddSupportedRegions:        []string{"us-west-2"},
		PrivateDnsName:             aws.String("service.internal"),
	})
	if err != nil {
		t.Fatalf("modify vpc endpoint service configuration: %v", err)
	}
	if !aws.ToBool(modifyOut.Return) {
		t.Fatalf("expected modify vpc endpoint service configuration return value true")
	}

	removeOut, err := client.ModifyVpcEndpointServiceConfiguration(ctx, &awsec2.ModifyVpcEndpointServiceConfigurationInput{
		ServiceId:                     aws.String("vpce-svc-00000000"),
		RemoveNetworkLoadBalancerArns: []string{"arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/net/demo/1234567890abcdef"},
		RemoveSupportedIpAddressTypes: []string{"ipv6"},
		RemoveSupportedRegions:        []string{"us-west-2"},
		RemovePrivateDnsName:          aws.Bool(true),
	})
	if err != nil {
		t.Fatalf("modify vpc endpoint service configuration remove values: %v", err)
	}
	if !aws.ToBool(removeOut.Return) {
		t.Fatalf("expected modify vpc endpoint service configuration remove values return value true")
	}
}

func TestEC2Stage51ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"ModifyVpcEndpointServiceConfiguration",
	}

	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, action := range implemented {
		resp := ec2Request(t, ts, action, map[string]string{
			"ServiceId":                    "vpce-svc-00000000",
			"AcceptanceRequired":           "true",
			"AddSupportedIpAddressTypes.1": "ipv6",
		})
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action)
		}
	}
}
