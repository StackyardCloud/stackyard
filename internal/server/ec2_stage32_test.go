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

func TestEC2Stage32SDKLifecycle(t *testing.T) {
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

	createMulticastOut, err := client.CreateTransitGatewayMulticastDomain(ctx, &awsec2.CreateTransitGatewayMulticastDomainInput{
		TransitGatewayId: aws.String("tgw-00000032"),
		TagSpecifications: []awsec2types.TagSpecification{
			{
				ResourceType: awsec2types.ResourceTypeTransitGatewayMulticastDomain,
				Tags:         []awsec2types.Tag{{Key: aws.String("env"), Value: aws.String("test")}},
			},
		},
	})
	if err != nil || createMulticastOut.TransitGatewayMulticastDomain == nil || createMulticastOut.TransitGatewayMulticastDomain.TransitGatewayMulticastDomainId == nil {
		t.Fatalf("create transit gateway multicast domain: %v", err)
	}
	if createMulticastOut.TransitGatewayMulticastDomain.State != awsec2types.TransitGatewayMulticastDomainStateAvailable {
		t.Fatalf("unexpected transit gateway multicast domain state: %q", createMulticastOut.TransitGatewayMulticastDomain.State)
	}
	multicastDomainID := aws.ToString(createMulticastOut.TransitGatewayMulticastDomain.TransitGatewayMulticastDomainId)

	deleteMulticastOut, err := client.DeleteTransitGatewayMulticastDomain(ctx, &awsec2.DeleteTransitGatewayMulticastDomainInput{
		TransitGatewayMulticastDomainId: aws.String(multicastDomainID),
	})
	if err != nil || deleteMulticastOut.TransitGatewayMulticastDomain == nil || deleteMulticastOut.TransitGatewayMulticastDomain.State != awsec2types.TransitGatewayMulticastDomainStateDeleted {
		t.Fatalf("delete transit gateway multicast domain: %v", err)
	}

	createPolicyOut, err := client.CreateTransitGatewayPolicyTable(ctx, &awsec2.CreateTransitGatewayPolicyTableInput{
		TransitGatewayId: aws.String("tgw-00000032"),
		TagSpecifications: []awsec2types.TagSpecification{
			{
				ResourceType: awsec2types.ResourceTypeTransitGatewayPolicyTable,
				Tags:         []awsec2types.Tag{{Key: aws.String("team"), Value: aws.String("platform")}},
			},
		},
	})
	if err != nil || createPolicyOut.TransitGatewayPolicyTable == nil || createPolicyOut.TransitGatewayPolicyTable.TransitGatewayPolicyTableId == nil {
		t.Fatalf("create transit gateway policy table: %v", err)
	}
	if createPolicyOut.TransitGatewayPolicyTable.State != awsec2types.TransitGatewayPolicyTableStateAvailable {
		t.Fatalf("unexpected transit gateway policy table state: %q", createPolicyOut.TransitGatewayPolicyTable.State)
	}
	policyTableID := aws.ToString(createPolicyOut.TransitGatewayPolicyTable.TransitGatewayPolicyTableId)

	deletePolicyOut, err := client.DeleteTransitGatewayPolicyTable(ctx, &awsec2.DeleteTransitGatewayPolicyTableInput{
		TransitGatewayPolicyTableId: aws.String(policyTableID),
	})
	if err != nil || deletePolicyOut.TransitGatewayPolicyTable == nil || deletePolicyOut.TransitGatewayPolicyTable.State != awsec2types.TransitGatewayPolicyTableStateDeleted {
		t.Fatalf("delete transit gateway policy table: %v", err)
	}

	createRouteOut, err := client.CreateTransitGatewayRouteTable(ctx, &awsec2.CreateTransitGatewayRouteTableInput{
		TransitGatewayId: aws.String("tgw-00000032"),
		TagSpecifications: []awsec2types.TagSpecification{
			{
				ResourceType: awsec2types.ResourceTypeTransitGatewayRouteTable,
				Tags:         []awsec2types.Tag{{Key: aws.String("service"), Value: aws.String("edge")}},
			},
		},
	})
	if err != nil || createRouteOut.TransitGatewayRouteTable == nil || createRouteOut.TransitGatewayRouteTable.TransitGatewayRouteTableId == nil {
		t.Fatalf("create transit gateway route table: %v", err)
	}
	if createRouteOut.TransitGatewayRouteTable.State != awsec2types.TransitGatewayRouteTableStateAvailable {
		t.Fatalf("unexpected transit gateway route table state: %q", createRouteOut.TransitGatewayRouteTable.State)
	}
	routeTableID := aws.ToString(createRouteOut.TransitGatewayRouteTable.TransitGatewayRouteTableId)

	deleteRouteOut, err := client.DeleteTransitGatewayRouteTable(ctx, &awsec2.DeleteTransitGatewayRouteTableInput{
		TransitGatewayRouteTableId: aws.String(routeTableID),
	})
	if err != nil || deleteRouteOut.TransitGatewayRouteTable == nil || deleteRouteOut.TransitGatewayRouteTable.State != awsec2types.TransitGatewayRouteTableStateDeleted {
		t.Fatalf("delete transit gateway route table: %v", err)
	}
}

func TestEC2Stage32ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"CreateTransitGatewayMulticastDomain",
		"DeleteTransitGatewayMulticastDomain",
		"CreateTransitGatewayPolicyTable",
		"DeleteTransitGatewayPolicyTable",
		"CreateTransitGatewayRouteTable",
		"DeleteTransitGatewayRouteTable",
	}

	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, action := range implemented {
		params := map[string]string{}
		switch action {
		case "CreateTransitGatewayMulticastDomain", "CreateTransitGatewayPolicyTable", "CreateTransitGatewayRouteTable":
			params["TransitGatewayId"] = "tgw-00000032"
		case "DeleteTransitGatewayMulticastDomain":
			params["TransitGatewayMulticastDomainId"] = "tgw-mcast-domain-00000032"
		case "DeleteTransitGatewayPolicyTable":
			params["TransitGatewayPolicyTableId"] = "tgw-ptb-00000032"
		case "DeleteTransitGatewayRouteTable":
			params["TransitGatewayRouteTableId"] = "tgw-rtb-00000032"
		}
		resp := ec2Request(t, ts, action, params)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action)
		}
	}
}
