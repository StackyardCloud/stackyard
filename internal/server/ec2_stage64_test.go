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

func TestEC2Stage64SDKLifecycle(t *testing.T) {
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

	createServiceOut, err := client.CreateVpcEndpointServiceConfiguration(ctx, &awsec2.CreateVpcEndpointServiceConfigurationInput{
		AcceptanceRequired:      aws.Bool(true),
		NetworkLoadBalancerArns: []string{"arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/net/stage64/1234567890abcdef"},
	})
	if err != nil {
		t.Fatalf("create vpc endpoint service configuration: %v", err)
	}
	if createServiceOut.ServiceConfiguration == nil || createServiceOut.ServiceConfiguration.ServiceId == nil || createServiceOut.ServiceConfiguration.ServiceName == nil {
		t.Fatalf("expected service id and service name")
	}
	serviceID := aws.ToString(createServiceOut.ServiceConfiguration.ServiceId)
	serviceName := aws.ToString(createServiceOut.ServiceConfiguration.ServiceName)

	createEndpointOneOut, err := client.CreateVpcEndpoint(ctx, &awsec2.CreateVpcEndpointInput{
		VpcId:             aws.String("vpc-00000001"),
		ServiceName:       aws.String(serviceName),
		VpcEndpointType:   awsec2types.VpcEndpointTypeInterface,
		SubnetIds:         []string{"subnet-00000001"},
		SecurityGroupIds:  []string{"sg-00000000"},
		PrivateDnsEnabled: aws.Bool(true),
	})
	if err != nil {
		t.Fatalf("create first vpc endpoint: %v", err)
	}
	if createEndpointOneOut.VpcEndpoint == nil || createEndpointOneOut.VpcEndpoint.VpcEndpointId == nil {
		t.Fatalf("expected first vpc endpoint id")
	}
	endpointIDOne := aws.ToString(createEndpointOneOut.VpcEndpoint.VpcEndpointId)

	createEndpointTwoOut, err := client.CreateVpcEndpoint(ctx, &awsec2.CreateVpcEndpointInput{
		VpcId:             aws.String("vpc-00000001"),
		ServiceName:       aws.String(serviceName),
		VpcEndpointType:   awsec2types.VpcEndpointTypeInterface,
		SubnetIds:         []string{"subnet-00000001"},
		SecurityGroupIds:  []string{"sg-00000000"},
		PrivateDnsEnabled: aws.Bool(true),
	})
	if err != nil {
		t.Fatalf("create second vpc endpoint: %v", err)
	}
	if createEndpointTwoOut.VpcEndpoint == nil || createEndpointTwoOut.VpcEndpoint.VpcEndpointId == nil {
		t.Fatalf("expected second vpc endpoint id")
	}
	endpointIDTwo := aws.ToString(createEndpointTwoOut.VpcEndpoint.VpcEndpointId)

	describeEndpointsPageOneOut, err := client.DescribeVpcEndpoints(ctx, &awsec2.DescribeVpcEndpointsInput{
		VpcEndpointIds: []string{endpointIDOne, endpointIDTwo},
		MaxResults:     aws.Int32(1),
	})
	if err != nil {
		t.Fatalf("describe vpc endpoints page one: %v", err)
	}
	if len(describeEndpointsPageOneOut.VpcEndpoints) != 1 {
		t.Fatalf("expected one endpoint in page one, got %d", len(describeEndpointsPageOneOut.VpcEndpoints))
	}
	if describeEndpointsPageOneOut.NextToken == nil {
		t.Fatalf("expected next token in page one")
	}

	describeEndpointsPageTwoOut, err := client.DescribeVpcEndpoints(ctx, &awsec2.DescribeVpcEndpointsInput{
		VpcEndpointIds: []string{endpointIDOne, endpointIDTwo},
		NextToken:      describeEndpointsPageOneOut.NextToken,
	})
	if err != nil {
		t.Fatalf("describe vpc endpoints page two: %v", err)
	}
	if len(describeEndpointsPageTwoOut.VpcEndpoints) == 0 {
		t.Fatalf("expected at least one endpoint in page two")
	}

	describeEndpointsFilteredOut, err := client.DescribeVpcEndpoints(ctx, &awsec2.DescribeVpcEndpointsInput{
		Filters: []awsec2types.Filter{
			{Name: aws.String("vpc-endpoint-id"), Values: []string{endpointIDOne}},
			{Name: aws.String("vpc-endpoint-state"), Values: []string{"available"}},
		},
	})
	if err != nil {
		t.Fatalf("describe filtered vpc endpoints: %v", err)
	}
	if len(describeEndpointsFilteredOut.VpcEndpoints) != 1 {
		t.Fatalf("expected one filtered endpoint, got %d", len(describeEndpointsFilteredOut.VpcEndpoints))
	}
	if aws.ToString(describeEndpointsFilteredOut.VpcEndpoints[0].VpcEndpointId) != endpointIDOne {
		t.Fatalf("unexpected filtered endpoint id: %q", aws.ToString(describeEndpointsFilteredOut.VpcEndpoints[0].VpcEndpointId))
	}

	describeConnectionsPageOneOut, err := client.DescribeVpcEndpointConnections(ctx, &awsec2.DescribeVpcEndpointConnectionsInput{
		Filters: []awsec2types.Filter{
			{Name: aws.String("service-id"), Values: []string{serviceID}},
		},
		MaxResults: aws.Int32(1),
	})
	if err != nil {
		t.Fatalf("describe vpc endpoint connections page one: %v", err)
	}
	if len(describeConnectionsPageOneOut.VpcEndpointConnections) != 1 {
		t.Fatalf("expected one connection in page one, got %d", len(describeConnectionsPageOneOut.VpcEndpointConnections))
	}
	if describeConnectionsPageOneOut.NextToken == nil {
		t.Fatalf("expected next token in describe connections page one")
	}

	describeConnectionsPageTwoOut, err := client.DescribeVpcEndpointConnections(ctx, &awsec2.DescribeVpcEndpointConnectionsInput{
		Filters: []awsec2types.Filter{
			{Name: aws.String("service-id"), Values: []string{serviceID}},
		},
		NextToken: describeConnectionsPageOneOut.NextToken,
	})
	if err != nil {
		t.Fatalf("describe vpc endpoint connections page two: %v", err)
	}
	if len(describeConnectionsPageTwoOut.VpcEndpointConnections) == 0 {
		t.Fatalf("expected at least one connection in page two")
	}

	describeConnectionsFilteredOut, err := client.DescribeVpcEndpointConnections(ctx, &awsec2.DescribeVpcEndpointConnectionsInput{
		Filters: []awsec2types.Filter{
			{Name: aws.String("service-id"), Values: []string{serviceID}},
			{Name: aws.String("vpc-endpoint-id"), Values: []string{endpointIDOne}},
		},
	})
	if err != nil {
		t.Fatalf("describe filtered vpc endpoint connections: %v", err)
	}
	if len(describeConnectionsFilteredOut.VpcEndpointConnections) != 1 {
		t.Fatalf("expected one filtered connection, got %d", len(describeConnectionsFilteredOut.VpcEndpointConnections))
	}
	if aws.ToString(describeConnectionsFilteredOut.VpcEndpointConnections[0].VpcEndpointId) != endpointIDOne {
		t.Fatalf("unexpected filtered connection endpoint id: %q", aws.ToString(describeConnectionsFilteredOut.VpcEndpointConnections[0].VpcEndpointId))
	}

	acceptOut, err := client.AcceptVpcEndpointConnections(ctx, &awsec2.AcceptVpcEndpointConnectionsInput{
		ServiceId:      aws.String(serviceID),
		VpcEndpointIds: []string{endpointIDOne},
	})
	if err != nil {
		t.Fatalf("accept vpc endpoint connection: %v", err)
	}
	if len(acceptOut.Unsuccessful) != 0 {
		t.Fatalf("expected no unsuccessful items when accepting, got %d", len(acceptOut.Unsuccessful))
	}

	rejectOut, err := client.RejectVpcEndpointConnections(ctx, &awsec2.RejectVpcEndpointConnectionsInput{
		ServiceId:      aws.String(serviceID),
		VpcEndpointIds: []string{endpointIDTwo},
	})
	if err != nil {
		t.Fatalf("reject vpc endpoint connection: %v", err)
	}
	if len(rejectOut.Unsuccessful) != 0 {
		t.Fatalf("expected no unsuccessful items when rejecting, got %d", len(rejectOut.Unsuccessful))
	}

	describeRejectedEndpointOut, err := client.DescribeVpcEndpoints(ctx, &awsec2.DescribeVpcEndpointsInput{
		VpcEndpointIds: []string{endpointIDTwo},
	})
	if err != nil {
		t.Fatalf("describe rejected endpoint: %v", err)
	}
	if len(describeRejectedEndpointOut.VpcEndpoints) != 1 {
		t.Fatalf("expected one rejected endpoint, got %d", len(describeRejectedEndpointOut.VpcEndpoints))
	}
	if !strings.EqualFold(string(describeRejectedEndpointOut.VpcEndpoints[0].State), "rejected") {
		t.Fatalf("expected endpoint state rejected, got %q", describeRejectedEndpointOut.VpcEndpoints[0].State)
	}

	describeAssociationsOut, err := client.DescribeVpcEndpointAssociations(ctx, &awsec2.DescribeVpcEndpointAssociationsInput{
		VpcEndpointIds: []string{endpointIDOne},
	})
	if err != nil {
		t.Fatalf("describe vpc endpoint associations: %v", err)
	}
	if len(describeAssociationsOut.VpcEndpointAssociations) != 1 {
		t.Fatalf("expected one endpoint association, got %d", len(describeAssociationsOut.VpcEndpointAssociations))
	}
	associationID := aws.ToString(describeAssociationsOut.VpcEndpointAssociations[0].Id)
	if associationID == "" {
		t.Fatalf("expected non-empty association id")
	}
	if aws.ToString(describeAssociationsOut.VpcEndpointAssociations[0].VpcEndpointId) != endpointIDOne {
		t.Fatalf("unexpected association endpoint id: %q", aws.ToString(describeAssociationsOut.VpcEndpointAssociations[0].VpcEndpointId))
	}

	describeAssociationsFilteredOut, err := client.DescribeVpcEndpointAssociations(ctx, &awsec2.DescribeVpcEndpointAssociationsInput{
		Filters: []awsec2types.Filter{
			{Name: aws.String("association-id"), Values: []string{associationID}},
			{Name: aws.String("vpc-endpoint-id"), Values: []string{endpointIDOne}},
		},
	})
	if err != nil {
		t.Fatalf("describe filtered vpc endpoint associations: %v", err)
	}
	if len(describeAssociationsFilteredOut.VpcEndpointAssociations) != 1 {
		t.Fatalf("expected one filtered association, got %d", len(describeAssociationsFilteredOut.VpcEndpointAssociations))
	}
}

func TestEC2Stage64ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"AcceptVpcEndpointConnections",
		"RejectVpcEndpointConnections",
		"DescribeVpcEndpoints",
		"DescribeVpcEndpointConnections",
		"DescribeVpcEndpointAssociations",
	}

	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, action := range implemented {
		params := map[string]string{}
		switch action {
		case "AcceptVpcEndpointConnections", "RejectVpcEndpointConnections":
			params["ServiceId"] = "vpce-svc-00000000"
			params["VpcEndpointId.1"] = "vpce-00000000"
		}
		resp := ec2Request(t, ts, action, params)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action)
		}
	}
}
