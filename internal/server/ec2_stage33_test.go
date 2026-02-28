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

func TestEC2Stage33SDKLifecycle(t *testing.T) {
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

	createMulticastOneOut, err := client.CreateTransitGatewayMulticastDomain(ctx, &awsec2.CreateTransitGatewayMulticastDomainInput{
		TransitGatewayId: aws.String("tgw-00000033"),
	})
	if err != nil || createMulticastOneOut.TransitGatewayMulticastDomain == nil || createMulticastOneOut.TransitGatewayMulticastDomain.TransitGatewayMulticastDomainId == nil {
		t.Fatalf("create transit gateway multicast domain #1: %v", err)
	}
	multicastOneID := aws.ToString(createMulticastOneOut.TransitGatewayMulticastDomain.TransitGatewayMulticastDomainId)

	if _, err := client.CreateTransitGatewayMulticastDomain(ctx, &awsec2.CreateTransitGatewayMulticastDomainInput{
		TransitGatewayId: aws.String("tgw-00000034"),
	}); err != nil {
		t.Fatalf("create transit gateway multicast domain #2: %v", err)
	}

	describeMulticastPageOneOut, err := client.DescribeTransitGatewayMulticastDomains(ctx, &awsec2.DescribeTransitGatewayMulticastDomainsInput{
		MaxResults: aws.Int32(1),
	})
	if err != nil || len(describeMulticastPageOneOut.TransitGatewayMulticastDomains) != 1 || describeMulticastPageOneOut.NextToken == nil {
		t.Fatalf("describe transit gateway multicast domains page 1: %v", err)
	}
	describeMulticastPageTwoOut, err := client.DescribeTransitGatewayMulticastDomains(ctx, &awsec2.DescribeTransitGatewayMulticastDomainsInput{
		MaxResults: aws.Int32(1),
		NextToken:  describeMulticastPageOneOut.NextToken,
	})
	if err != nil || len(describeMulticastPageTwoOut.TransitGatewayMulticastDomains) != 1 {
		t.Fatalf("describe transit gateway multicast domains page 2: %v", err)
	}

	describeMulticastFilteredOut, err := client.DescribeTransitGatewayMulticastDomains(ctx, &awsec2.DescribeTransitGatewayMulticastDomainsInput{
		TransitGatewayMulticastDomainIds: []string{multicastOneID},
		Filters: []awsec2types.Filter{
			{Name: aws.String("state"), Values: []string{"available"}},
		},
	})
	if err != nil || len(describeMulticastFilteredOut.TransitGatewayMulticastDomains) != 1 || aws.ToString(describeMulticastFilteredOut.TransitGatewayMulticastDomains[0].TransitGatewayMulticastDomainId) != multicastOneID {
		t.Fatalf("describe transit gateway multicast domains filtered: %v", err)
	}

	createPolicyOut, err := client.CreateTransitGatewayPolicyTable(ctx, &awsec2.CreateTransitGatewayPolicyTableInput{
		TransitGatewayId: aws.String("tgw-00000033"),
	})
	if err != nil || createPolicyOut.TransitGatewayPolicyTable == nil || createPolicyOut.TransitGatewayPolicyTable.TransitGatewayPolicyTableId == nil {
		t.Fatalf("create transit gateway policy table: %v", err)
	}
	policyTableID := aws.ToString(createPolicyOut.TransitGatewayPolicyTable.TransitGatewayPolicyTableId)

	describePolicyOut, err := client.DescribeTransitGatewayPolicyTables(ctx, &awsec2.DescribeTransitGatewayPolicyTablesInput{
		TransitGatewayPolicyTableIds: []string{policyTableID},
		Filters: []awsec2types.Filter{
			{Name: aws.String("transit-gateway-id"), Values: []string{"tgw-00000033"}},
		},
	})
	if err != nil || len(describePolicyOut.TransitGatewayPolicyTables) != 1 || aws.ToString(describePolicyOut.TransitGatewayPolicyTables[0].TransitGatewayPolicyTableId) != policyTableID {
		t.Fatalf("describe transit gateway policy tables: %v", err)
	}

	createRouteOut, err := client.CreateTransitGatewayRouteTable(ctx, &awsec2.CreateTransitGatewayRouteTableInput{
		TransitGatewayId: aws.String("tgw-00000033"),
	})
	if err != nil || createRouteOut.TransitGatewayRouteTable == nil || createRouteOut.TransitGatewayRouteTable.TransitGatewayRouteTableId == nil {
		t.Fatalf("create transit gateway route table: %v", err)
	}
	routeTableID := aws.ToString(createRouteOut.TransitGatewayRouteTable.TransitGatewayRouteTableId)

	describeRouteOut, err := client.DescribeTransitGatewayRouteTables(ctx, &awsec2.DescribeTransitGatewayRouteTablesInput{
		TransitGatewayRouteTableIds: []string{routeTableID},
		Filters: []awsec2types.Filter{
			{Name: aws.String("default-association-route-table"), Values: []string{"false"}},
			{Name: aws.String("default-propagation-route-table"), Values: []string{"false"}},
			{Name: aws.String("state"), Values: []string{"available"}},
		},
	})
	if err != nil || len(describeRouteOut.TransitGatewayRouteTables) != 1 || aws.ToString(describeRouteOut.TransitGatewayRouteTables[0].TransitGatewayRouteTableId) != routeTableID {
		t.Fatalf("describe transit gateway route tables: %v", err)
	}
}

func TestEC2Stage33ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"DescribeTransitGatewayMulticastDomains",
		"DescribeTransitGatewayPolicyTables",
		"DescribeTransitGatewayRouteTables",
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
