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

func TestEC2Stage61SDKLifecycle(t *testing.T) {
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
		NetworkLoadBalancerArns: []string{"arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/net/stage61-a/1234567890abcdef"},
		SupportedIpAddressTypes: []string{"ipv4"},
	})
	if err != nil {
		t.Fatalf("create service configuration one: %v", err)
	}
	if createOne.ServiceConfiguration == nil || createOne.ServiceConfiguration.ServiceId == nil {
		t.Fatalf("expected service configuration one id")
	}

	createTwo, err := client.CreateVpcEndpointServiceConfiguration(ctx, &awsec2.CreateVpcEndpointServiceConfigurationInput{
		NetworkLoadBalancerArns: []string{"arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/net/stage61-b/abcdef1234567890"},
		SupportedIpAddressTypes: []string{"ipv6"},
	})
	if err != nil {
		t.Fatalf("create service configuration two: %v", err)
	}
	if createTwo.ServiceConfiguration == nil || createTwo.ServiceConfiguration.ServiceId == nil || createTwo.ServiceConfiguration.ServiceName == nil {
		t.Fatalf("expected service configuration two identifiers")
	}
	serviceIDTwo := aws.ToString(createTwo.ServiceConfiguration.ServiceId)
	serviceNameTwo := aws.ToString(createTwo.ServiceConfiguration.ServiceName)

	pageOne, err := client.DescribeVpcEndpointServiceConfigurations(ctx, &awsec2.DescribeVpcEndpointServiceConfigurationsInput{
		MaxResults: aws.Int32(1),
	})
	if err != nil {
		t.Fatalf("describe service configurations page one: %v", err)
	}
	if len(pageOne.ServiceConfigurations) != 1 {
		t.Fatalf("expected one service configuration in first page, got %d", len(pageOne.ServiceConfigurations))
	}
	if pageOne.NextToken == nil {
		t.Fatalf("expected next token in first page")
	}

	pageTwo, err := client.DescribeVpcEndpointServiceConfigurations(ctx, &awsec2.DescribeVpcEndpointServiceConfigurationsInput{
		NextToken: pageOne.NextToken,
	})
	if err != nil {
		t.Fatalf("describe service configurations page two: %v", err)
	}
	if len(pageTwo.ServiceConfigurations) == 0 {
		t.Fatalf("expected at least one service configuration in second page")
	}

	byID, err := client.DescribeVpcEndpointServiceConfigurations(ctx, &awsec2.DescribeVpcEndpointServiceConfigurationsInput{
		ServiceIds: []string{serviceIDTwo},
	})
	if err != nil {
		t.Fatalf("describe service configurations by id: %v", err)
	}
	if len(byID.ServiceConfigurations) != 1 {
		t.Fatalf("expected one service configuration by id, got %d", len(byID.ServiceConfigurations))
	}
	if aws.ToString(byID.ServiceConfigurations[0].ServiceId) != serviceIDTwo {
		t.Fatalf("unexpected service configuration id: %q", aws.ToString(byID.ServiceConfigurations[0].ServiceId))
	}

	filteredByName, err := client.DescribeVpcEndpointServiceConfigurations(ctx, &awsec2.DescribeVpcEndpointServiceConfigurationsInput{
		Filters: []awsec2types.Filter{
			{Name: aws.String("service-name"), Values: []string{serviceNameTwo}},
			{Name: aws.String("supported-ip-address-types"), Values: []string{"ipv6"}},
		},
	})
	if err != nil {
		t.Fatalf("describe service configurations by filters: %v", err)
	}
	if len(filteredByName.ServiceConfigurations) != 1 {
		t.Fatalf("expected one service configuration by filters, got %d", len(filteredByName.ServiceConfigurations))
	}
	if aws.ToString(filteredByName.ServiceConfigurations[0].ServiceId) != serviceIDTwo {
		t.Fatalf("unexpected filtered service configuration id: %q", aws.ToString(filteredByName.ServiceConfigurations[0].ServiceId))
	}
}

func TestEC2Stage61ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"DescribeVpcEndpointServiceConfigurations",
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
