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

func TestEC2Stage34SDKLifecycle(t *testing.T) {
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

	createSubnetOut, err := client.CreateSubnet(ctx, &awsec2.CreateSubnetInput{
		VpcId:     aws.String("vpc-00000001"),
		CidrBlock: aws.String("10.0.2.0/24"),
	})
	if err != nil || createSubnetOut.Subnet == nil || createSubnetOut.Subnet.SubnetId == nil {
		t.Fatalf("create subnet: %v", err)
	}
	secondSubnetID := aws.ToString(createSubnetOut.Subnet.SubnetId)

	createMulticastOut, err := client.CreateTransitGatewayMulticastDomain(ctx, &awsec2.CreateTransitGatewayMulticastDomainInput{
		TransitGatewayId: aws.String("tgw-00000034"),
	})
	if err != nil || createMulticastOut.TransitGatewayMulticastDomain == nil || createMulticastOut.TransitGatewayMulticastDomain.TransitGatewayMulticastDomainId == nil {
		t.Fatalf("create transit gateway multicast domain: %v", err)
	}
	multicastDomainID := aws.ToString(createMulticastOut.TransitGatewayMulticastDomain.TransitGatewayMulticastDomainId)

	if _, err := client.AssociateTransitGatewayMulticastDomain(ctx, &awsec2.AssociateTransitGatewayMulticastDomainInput{
		TransitGatewayMulticastDomainId: aws.String(multicastDomainID),
		TransitGatewayAttachmentId:      aws.String("tgw-attach-00000034"),
		SubnetIds:                       []string{"subnet-00000001", secondSubnetID},
	}); err != nil {
		t.Fatalf("associate transit gateway multicast domain: %v", err)
	}

	getMulticastPageOneOut, err := client.GetTransitGatewayMulticastDomainAssociations(ctx, &awsec2.GetTransitGatewayMulticastDomainAssociationsInput{
		TransitGatewayMulticastDomainId: aws.String(multicastDomainID),
		MaxResults:                      aws.Int32(1),
	})
	if err != nil || len(getMulticastPageOneOut.MulticastDomainAssociations) != 1 || getMulticastPageOneOut.NextToken == nil {
		t.Fatalf("get transit gateway multicast domain associations page 1: %v", err)
	}
	if getMulticastPageOneOut.MulticastDomainAssociations[0].Subnet == nil || getMulticastPageOneOut.MulticastDomainAssociations[0].Subnet.State != awsec2types.TransitGatewayMulitcastDomainAssociationStateAssociated {
		t.Fatalf("unexpected multicast association page 1 payload: %+v", getMulticastPageOneOut.MulticastDomainAssociations[0])
	}

	getMulticastPageTwoOut, err := client.GetTransitGatewayMulticastDomainAssociations(ctx, &awsec2.GetTransitGatewayMulticastDomainAssociationsInput{
		TransitGatewayMulticastDomainId: aws.String(multicastDomainID),
		MaxResults:                      aws.Int32(1),
		NextToken:                       getMulticastPageOneOut.NextToken,
	})
	if err != nil || len(getMulticastPageTwoOut.MulticastDomainAssociations) != 1 {
		t.Fatalf("get transit gateway multicast domain associations page 2: %v", err)
	}

	getMulticastFilteredOut, err := client.GetTransitGatewayMulticastDomainAssociations(ctx, &awsec2.GetTransitGatewayMulticastDomainAssociationsInput{
		TransitGatewayMulticastDomainId: aws.String(multicastDomainID),
		Filters: []awsec2types.Filter{
			{Name: aws.String("subnet-id"), Values: []string{secondSubnetID}},
			{Name: aws.String("resource-type"), Values: []string{"vpc"}},
		},
	})
	if err != nil || len(getMulticastFilteredOut.MulticastDomainAssociations) != 1 || getMulticastFilteredOut.MulticastDomainAssociations[0].Subnet == nil || aws.ToString(getMulticastFilteredOut.MulticastDomainAssociations[0].Subnet.SubnetId) != secondSubnetID {
		t.Fatalf("get transit gateway multicast domain associations filtered: %v", err)
	}

	createPolicyOut, err := client.CreateTransitGatewayPolicyTable(ctx, &awsec2.CreateTransitGatewayPolicyTableInput{
		TransitGatewayId: aws.String("tgw-00000034"),
	})
	if err != nil || createPolicyOut.TransitGatewayPolicyTable == nil || createPolicyOut.TransitGatewayPolicyTable.TransitGatewayPolicyTableId == nil {
		t.Fatalf("create transit gateway policy table: %v", err)
	}
	policyTableID := aws.ToString(createPolicyOut.TransitGatewayPolicyTable.TransitGatewayPolicyTableId)

	if _, err := client.AssociateTransitGatewayPolicyTable(ctx, &awsec2.AssociateTransitGatewayPolicyTableInput{
		TransitGatewayPolicyTableId: aws.String(policyTableID),
		TransitGatewayAttachmentId:  aws.String("tgw-attach-00000035"),
	}); err != nil {
		t.Fatalf("associate transit gateway policy table: %v", err)
	}

	getPolicyOut, err := client.GetTransitGatewayPolicyTableAssociations(ctx, &awsec2.GetTransitGatewayPolicyTableAssociationsInput{
		TransitGatewayPolicyTableId: aws.String(policyTableID),
		Filters: []awsec2types.Filter{
			{Name: aws.String("state"), Values: []string{"associated"}},
		},
	})
	if err != nil || len(getPolicyOut.Associations) != 1 || getPolicyOut.Associations[0].State != awsec2types.TransitGatewayAssociationStateAssociated {
		t.Fatalf("get transit gateway policy table associations: %v", err)
	}

	createRouteOut, err := client.CreateTransitGatewayRouteTable(ctx, &awsec2.CreateTransitGatewayRouteTableInput{
		TransitGatewayId: aws.String("tgw-00000034"),
	})
	if err != nil || createRouteOut.TransitGatewayRouteTable == nil || createRouteOut.TransitGatewayRouteTable.TransitGatewayRouteTableId == nil {
		t.Fatalf("create transit gateway route table: %v", err)
	}
	routeTableID := aws.ToString(createRouteOut.TransitGatewayRouteTable.TransitGatewayRouteTableId)

	if _, err := client.AssociateTransitGatewayRouteTable(ctx, &awsec2.AssociateTransitGatewayRouteTableInput{
		TransitGatewayRouteTableId: aws.String(routeTableID),
		TransitGatewayAttachmentId: aws.String("tgw-attach-00000036"),
	}); err != nil {
		t.Fatalf("associate transit gateway route table: %v", err)
	}

	getRouteOut, err := client.GetTransitGatewayRouteTableAssociations(ctx, &awsec2.GetTransitGatewayRouteTableAssociationsInput{
		TransitGatewayRouteTableId: aws.String(routeTableID),
		Filters: []awsec2types.Filter{
			{Name: aws.String("transit-gateway-attachment-id"), Values: []string{"tgw-attach-00000036"}},
		},
	})
	if err != nil || len(getRouteOut.Associations) != 1 || getRouteOut.Associations[0].State != awsec2types.TransitGatewayAssociationStateAssociated {
		t.Fatalf("get transit gateway route table associations: %v", err)
	}
}

func TestEC2Stage34ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"GetTransitGatewayMulticastDomainAssociations",
		"GetTransitGatewayPolicyTableAssociations",
		"GetTransitGatewayRouteTableAssociations",
	}

	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, action := range implemented {
		params := map[string]string{}
		switch action {
		case "GetTransitGatewayMulticastDomainAssociations":
			params["TransitGatewayMulticastDomainId"] = "tgw-mcast-domain-00000034"
		case "GetTransitGatewayPolicyTableAssociations":
			params["TransitGatewayPolicyTableId"] = "tgw-ptb-00000034"
		case "GetTransitGatewayRouteTableAssociations":
			params["TransitGatewayRouteTableId"] = "tgw-rtb-00000034"
		}
		resp := ec2Request(t, ts, action, params)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action)
		}
	}
}
