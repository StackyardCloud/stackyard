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

func TestEC2Stage30SDKLifecycle(t *testing.T) {
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

	associateMulticastOut, err := client.AssociateTransitGatewayMulticastDomain(ctx, &awsec2.AssociateTransitGatewayMulticastDomainInput{
		SubnetIds:                       []string{"subnet-00000001"},
		TransitGatewayAttachmentId:      aws.String("tgw-attach-00000001"),
		TransitGatewayMulticastDomainId: aws.String("tgw-mcast-domain-00000001"),
	})
	if err != nil || associateMulticastOut.Associations == nil {
		t.Fatalf("associate transit gateway multicast domain: %v", err)
	}
	if aws.ToString(associateMulticastOut.Associations.TransitGatewayAttachmentId) != "tgw-attach-00000001" {
		t.Fatalf("unexpected transit gateway attachment id: %q", aws.ToString(associateMulticastOut.Associations.TransitGatewayAttachmentId))
	}
	if len(associateMulticastOut.Associations.Subnets) != 1 || aws.ToString(associateMulticastOut.Associations.Subnets[0].SubnetId) != "subnet-00000001" {
		t.Fatalf("unexpected multicast association subnets: %+v", associateMulticastOut.Associations.Subnets)
	}

	associatePolicyOut, err := client.AssociateTransitGatewayPolicyTable(ctx, &awsec2.AssociateTransitGatewayPolicyTableInput{
		TransitGatewayAttachmentId:  aws.String("tgw-attach-00000001"),
		TransitGatewayPolicyTableId: aws.String("tgw-ptb-00000001"),
	})
	if err != nil || associatePolicyOut.Association == nil {
		t.Fatalf("associate transit gateway policy table: %v", err)
	}
	if associatePolicyOut.Association.State != awsec2types.TransitGatewayAssociationStateAssociated {
		t.Fatalf("unexpected policy table association state: %q", associatePolicyOut.Association.State)
	}

	associateRouteOut, err := client.AssociateTransitGatewayRouteTable(ctx, &awsec2.AssociateTransitGatewayRouteTableInput{
		TransitGatewayAttachmentId: aws.String("tgw-attach-00000001"),
		TransitGatewayRouteTableId: aws.String("tgw-rtb-00000001"),
	})
	if err != nil || associateRouteOut.Association == nil {
		t.Fatalf("associate transit gateway route table: %v", err)
	}
	if associateRouteOut.Association.State != awsec2types.TransitGatewayAssociationStateAssociated {
		t.Fatalf("unexpected route table association state: %q", associateRouteOut.Association.State)
	}

	createTrunkOut, err := client.CreateNetworkInterface(ctx, &awsec2.CreateNetworkInterfaceInput{SubnetId: aws.String("subnet-00000001")})
	if err != nil || createTrunkOut.NetworkInterface == nil || createTrunkOut.NetworkInterface.NetworkInterfaceId == nil {
		t.Fatalf("create trunk network interface: %v", err)
	}
	trunkID := aws.ToString(createTrunkOut.NetworkInterface.NetworkInterfaceId)

	createBranchOut, err := client.CreateNetworkInterface(ctx, &awsec2.CreateNetworkInterfaceInput{SubnetId: aws.String("subnet-00000001")})
	if err != nil || createBranchOut.NetworkInterface == nil || createBranchOut.NetworkInterface.NetworkInterfaceId == nil {
		t.Fatalf("create branch network interface: %v", err)
	}
	branchID := aws.ToString(createBranchOut.NetworkInterface.NetworkInterfaceId)

	associateTrunkOut, err := client.AssociateTrunkInterface(ctx, &awsec2.AssociateTrunkInterfaceInput{
		BranchInterfaceId: aws.String(branchID),
		TrunkInterfaceId:  aws.String(trunkID),
		VlanId:            aws.Int32(101),
		ClientToken:       aws.String("stage30-associate-token"),
	})
	if err != nil || associateTrunkOut.InterfaceAssociation == nil || associateTrunkOut.InterfaceAssociation.AssociationId == nil {
		t.Fatalf("associate trunk interface: %v", err)
	}
	if associateTrunkOut.InterfaceAssociation.InterfaceProtocol != awsec2types.InterfaceProtocolTypeVlan {
		t.Fatalf("unexpected interface protocol: %q", associateTrunkOut.InterfaceAssociation.InterfaceProtocol)
	}
	if associateTrunkOut.InterfaceAssociation.VlanId == nil || aws.ToInt32(associateTrunkOut.InterfaceAssociation.VlanId) != 101 {
		t.Fatalf("unexpected vlan id: %+v", associateTrunkOut.InterfaceAssociation.VlanId)
	}
	associationID := aws.ToString(associateTrunkOut.InterfaceAssociation.AssociationId)

	disassociateTrunkOut, err := client.DisassociateTrunkInterface(ctx, &awsec2.DisassociateTrunkInterfaceInput{
		AssociationId: aws.String(associationID),
		ClientToken:   aws.String("stage30-disassociate-token"),
	})
	if err != nil || disassociateTrunkOut.Return == nil || !aws.ToBool(disassociateTrunkOut.Return) {
		t.Fatalf("disassociate trunk interface: %v", err)
	}
	if aws.ToString(disassociateTrunkOut.ClientToken) != "stage30-disassociate-token" {
		t.Fatalf("unexpected disassociate client token: %q", aws.ToString(disassociateTrunkOut.ClientToken))
	}
}

func TestEC2Stage30ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"AssociateTransitGatewayMulticastDomain",
		"AssociateTransitGatewayPolicyTable",
		"AssociateTransitGatewayRouteTable",
		"AssociateTrunkInterface",
		"DisassociateTrunkInterface",
	}

	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, action := range implemented {
		params := map[string]string{}
		switch action {
		case "AssociateTransitGatewayMulticastDomain":
			params["TransitGatewayMulticastDomainId"] = "tgw-mcast-domain-00000001"
			params["TransitGatewayAttachmentId"] = "tgw-attach-00000001"
			params["SubnetIds.1"] = "subnet-00000001"
		case "AssociateTransitGatewayPolicyTable":
			params["TransitGatewayPolicyTableId"] = "tgw-ptb-00000001"
			params["TransitGatewayAttachmentId"] = "tgw-attach-00000001"
		case "AssociateTransitGatewayRouteTable":
			params["TransitGatewayRouteTableId"] = "tgw-rtb-00000001"
			params["TransitGatewayAttachmentId"] = "tgw-attach-00000001"
		case "AssociateTrunkInterface":
			params["BranchInterfaceId"] = "eni-00000001"
			params["TrunkInterfaceId"] = "eni-00000002"
			params["VlanId"] = "101"
		case "DisassociateTrunkInterface":
			params["AssociationId"] = "trunk-assoc-00000001"
		}
		resp := ec2Request(t, ts, action, params)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action)
		}
	}
}
