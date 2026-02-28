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

func TestEC2Stage63SDKLifecycle(t *testing.T) {
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

	createOne, err := client.CreateVpcEndpointServiceConfiguration(ctx, &awsec2.CreateVpcEndpointServiceConfigurationInput{
		NetworkLoadBalancerArns: []string{"arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/net/stage63-a/1234567890abcdef"},
		SupportedIpAddressTypes: []string{"ipv4"},
		SupportedRegions:        []string{"us-east-1"},
	})
	if err != nil {
		t.Fatalf("create service configuration one: %v", err)
	}
	if createOne.ServiceConfiguration == nil || createOne.ServiceConfiguration.ServiceName == nil {
		t.Fatalf("expected service configuration one name")
	}
	serviceNameOne := aws.ToString(createOne.ServiceConfiguration.ServiceName)

	createTwo, err := client.CreateVpcEndpointServiceConfiguration(ctx, &awsec2.CreateVpcEndpointServiceConfigurationInput{
		GatewayLoadBalancerArns: []string{"arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/gwy/stage63-b/abcdef1234567890"},
		SupportedIpAddressTypes: []string{"ipv6"},
		SupportedRegions:        []string{"us-west-2"},
	})
	if err != nil {
		t.Fatalf("create service configuration two: %v", err)
	}
	if createTwo.ServiceConfiguration == nil || createTwo.ServiceConfiguration.ServiceName == nil || createTwo.ServiceConfiguration.ServiceId == nil {
		t.Fatalf("expected service configuration two identifiers")
	}
	serviceNameTwo := aws.ToString(createTwo.ServiceConfiguration.ServiceName)
	serviceIDTwo := aws.ToString(createTwo.ServiceConfiguration.ServiceId)

	pageOne, err := client.DescribeVpcEndpointServices(ctx, &awsec2.DescribeVpcEndpointServicesInput{
		ServiceNames: []string{serviceNameOne, serviceNameTwo},
		MaxResults:   aws.Int32(1),
	})
	if err != nil {
		t.Fatalf("describe vpc endpoint services page one: %v", err)
	}
	if len(pageOne.ServiceDetails) != 1 {
		t.Fatalf("expected one service detail in first page, got %d", len(pageOne.ServiceDetails))
	}
	if len(pageOne.ServiceNames) != 1 {
		t.Fatalf("expected one service name in first page, got %d", len(pageOne.ServiceNames))
	}
	if pageOne.NextToken == nil {
		t.Fatalf("expected next token in first page")
	}

	pageTwo, err := client.DescribeVpcEndpointServices(ctx, &awsec2.DescribeVpcEndpointServicesInput{
		ServiceNames: []string{serviceNameOne, serviceNameTwo},
		NextToken:    pageOne.NextToken,
	})
	if err != nil {
		t.Fatalf("describe vpc endpoint services page two: %v", err)
	}
	if len(pageTwo.ServiceDetails) != 1 {
		t.Fatalf("expected one service detail in second page, got %d", len(pageTwo.ServiceDetails))
	}

	filtered, err := client.DescribeVpcEndpointServices(ctx, &awsec2.DescribeVpcEndpointServicesInput{
		Filters: []awsec2types.Filter{
			{Name: aws.String("service-type"), Values: []string{"GatewayLoadBalancer"}},
			{Name: aws.String("supported-ip-address-types"), Values: []string{"ipv6"}},
			{Name: aws.String("service-region"), Values: []string{"us-west-2"}},
		},
	})
	if err != nil {
		t.Fatalf("describe vpc endpoint services filtered: %v", err)
	}
	if len(filtered.ServiceDetails) != 1 {
		t.Fatalf("expected one filtered service detail, got %d", len(filtered.ServiceDetails))
	}
	if aws.ToString(filtered.ServiceDetails[0].ServiceId) != serviceIDTwo {
		t.Fatalf("unexpected filtered service id: %q", aws.ToString(filtered.ServiceDetails[0].ServiceId))
	}
	if len(filtered.ServiceDetails[0].ServiceType) != 1 || filtered.ServiceDetails[0].ServiceType[0].ServiceType != awsec2types.ServiceTypeGatewayLoadBalancer {
		t.Fatalf("unexpected filtered service type: %+v", filtered.ServiceDetails[0].ServiceType)
	}
}

func TestEC2Stage63ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"DescribeVpcEndpointServices",
	}

	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, action := range implemented {
		resp := ec2Request(t, ts, action, nil)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action)
		}
	}
}
