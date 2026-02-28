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

func TestEC2Stage31SDKLifecycle(t *testing.T) {
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

	if _, err := client.AssociateTransitGatewayMulticastDomain(ctx, &awsec2.AssociateTransitGatewayMulticastDomainInput{
		SubnetIds:                       []string{"subnet-00000001"},
		TransitGatewayAttachmentId:      aws.String("tgw-attach-00000011"),
		TransitGatewayMulticastDomainId: aws.String("tgw-mcast-domain-00000011"),
	}); err != nil {
		t.Fatalf("associate transit gateway multicast domain: %v", err)
	}
	disassociateMulticastOut, err := client.DisassociateTransitGatewayMulticastDomain(ctx, &awsec2.DisassociateTransitGatewayMulticastDomainInput{
		SubnetIds:                       []string{"subnet-00000001"},
		TransitGatewayAttachmentId:      aws.String("tgw-attach-00000011"),
		TransitGatewayMulticastDomainId: aws.String("tgw-mcast-domain-00000011"),
	})
	if err != nil || disassociateMulticastOut.Associations == nil {
		t.Fatalf("disassociate transit gateway multicast domain: %v", err)
	}
	if len(disassociateMulticastOut.Associations.Subnets) != 1 || disassociateMulticastOut.Associations.Subnets[0].State != awsec2types.TransitGatewayMulitcastDomainAssociationStateDisassociated {
		t.Fatalf("unexpected multicast disassociate subnets: %+v", disassociateMulticastOut.Associations.Subnets)
	}

	if _, err := client.AssociateTransitGatewayPolicyTable(ctx, &awsec2.AssociateTransitGatewayPolicyTableInput{
		TransitGatewayAttachmentId:  aws.String("tgw-attach-00000011"),
		TransitGatewayPolicyTableId: aws.String("tgw-ptb-00000011"),
	}); err != nil {
		t.Fatalf("associate transit gateway policy table: %v", err)
	}
	disassociatePolicyOut, err := client.DisassociateTransitGatewayPolicyTable(ctx, &awsec2.DisassociateTransitGatewayPolicyTableInput{
		TransitGatewayAttachmentId:  aws.String("tgw-attach-00000011"),
		TransitGatewayPolicyTableId: aws.String("tgw-ptb-00000011"),
	})
	if err != nil || disassociatePolicyOut.Association == nil || disassociatePolicyOut.Association.State != awsec2types.TransitGatewayAssociationStateDisassociated {
		t.Fatalf("disassociate transit gateway policy table: %v", err)
	}

	if _, err := client.AssociateTransitGatewayRouteTable(ctx, &awsec2.AssociateTransitGatewayRouteTableInput{
		TransitGatewayAttachmentId: aws.String("tgw-attach-00000011"),
		TransitGatewayRouteTableId: aws.String("tgw-rtb-00000011"),
	}); err != nil {
		t.Fatalf("associate transit gateway route table: %v", err)
	}
	disassociateRouteOut, err := client.DisassociateTransitGatewayRouteTable(ctx, &awsec2.DisassociateTransitGatewayRouteTableInput{
		TransitGatewayAttachmentId: aws.String("tgw-attach-00000011"),
		TransitGatewayRouteTableId: aws.String("tgw-rtb-00000011"),
	})
	if err != nil || disassociateRouteOut.Association == nil || disassociateRouteOut.Association.State != awsec2types.TransitGatewayAssociationStateDisassociated {
		t.Fatalf("disassociate transit gateway route table: %v", err)
	}
}

func TestEC2Stage31ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"DisassociateTransitGatewayMulticastDomain",
		"DisassociateTransitGatewayPolicyTable",
		"DisassociateTransitGatewayRouteTable",
	}

	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, action := range implemented {
		params := map[string]string{}
		switch action {
		case "DisassociateTransitGatewayMulticastDomain":
			params["TransitGatewayMulticastDomainId"] = "tgw-mcast-domain-00000011"
			params["TransitGatewayAttachmentId"] = "tgw-attach-00000011"
			params["SubnetIds.1"] = "subnet-00000001"
		case "DisassociateTransitGatewayPolicyTable":
			params["TransitGatewayPolicyTableId"] = "tgw-ptb-00000011"
			params["TransitGatewayAttachmentId"] = "tgw-attach-00000011"
		case "DisassociateTransitGatewayRouteTable":
			params["TransitGatewayRouteTableId"] = "tgw-rtb-00000011"
			params["TransitGatewayAttachmentId"] = "tgw-attach-00000011"
		}
		resp := ec2Request(t, ts, action, params)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action)
		}
	}
}
