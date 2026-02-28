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

func TestEC2Stage43SDKLifecycle(t *testing.T) {
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

	createGatewayOut, err := client.CreateTransitGateway(ctx, &awsec2.CreateTransitGatewayInput{
		Description: aws.String("stage-43-gateway"),
		Options: &awsec2types.TransitGatewayRequestOptions{
			TransitGatewayCidrBlocks: []string{"10.43.0.0/24"},
		},
	})
	if err != nil || createGatewayOut.TransitGateway == nil || createGatewayOut.TransitGateway.TransitGatewayId == nil {
		t.Fatalf("create transit gateway: %v", err)
	}
	transitGatewayID := aws.ToString(createGatewayOut.TransitGateway.TransitGatewayId)

	createRouteTableOut, err := client.CreateTransitGatewayRouteTable(ctx, &awsec2.CreateTransitGatewayRouteTableInput{
		TransitGatewayId: aws.String(transitGatewayID),
	})
	if err != nil || createRouteTableOut.TransitGatewayRouteTable == nil || createRouteTableOut.TransitGatewayRouteTable.TransitGatewayRouteTableId == nil {
		t.Fatalf("create transit gateway route table: %v", err)
	}
	routeTableID := aws.ToString(createRouteTableOut.TransitGatewayRouteTable.TransitGatewayRouteTableId)

	modifyGatewayOut, err := client.ModifyTransitGateway(ctx, &awsec2.ModifyTransitGatewayInput{
		TransitGatewayId: aws.String(transitGatewayID),
		Description:      aws.String("stage-43-gateway-updated"),
		Options: &awsec2types.ModifyTransitGatewayOptions{
			AddTransitGatewayCidrBlocks:     []string{"10.43.1.0/24"},
			RemoveTransitGatewayCidrBlocks:  []string{"10.43.0.0/24"},
			AmazonSideAsn:                   aws.Int64(65043),
			AssociationDefaultRouteTableId:  aws.String(routeTableID),
			AutoAcceptSharedAttachments:     awsec2types.AutoAcceptSharedAttachmentsValueEnable,
			DefaultRouteTableAssociation:    awsec2types.DefaultRouteTableAssociationValueDisable,
			DefaultRouteTablePropagation:    awsec2types.DefaultRouteTablePropagationValueDisable,
			DnsSupport:                      awsec2types.DnsSupportValueDisable,
			PropagationDefaultRouteTableId:  aws.String(routeTableID),
			SecurityGroupReferencingSupport: awsec2types.SecurityGroupReferencingSupportValueEnable,
			VpnEcmpSupport:                  awsec2types.VpnEcmpSupportValueDisable,
		},
	})
	if err != nil || modifyGatewayOut.TransitGateway == nil || modifyGatewayOut.TransitGateway.Options == nil {
		t.Fatalf("modify transit gateway: %v", err)
	}
	if aws.ToString(modifyGatewayOut.TransitGateway.Description) != "stage-43-gateway-updated" {
		t.Fatalf("unexpected transit gateway description: %q", aws.ToString(modifyGatewayOut.TransitGateway.Description))
	}
	if aws.ToInt64(modifyGatewayOut.TransitGateway.Options.AmazonSideAsn) != 65043 {
		t.Fatalf("unexpected transit gateway Amazon-side ASN")
	}
	if aws.ToString(modifyGatewayOut.TransitGateway.Options.AssociationDefaultRouteTableId) != routeTableID {
		t.Fatalf("unexpected association default route table id")
	}
	if aws.ToString(modifyGatewayOut.TransitGateway.Options.PropagationDefaultRouteTableId) != routeTableID {
		t.Fatalf("unexpected propagation default route table id")
	}
	if modifyGatewayOut.TransitGateway.Options.AutoAcceptSharedAttachments != awsec2types.AutoAcceptSharedAttachmentsValueEnable {
		t.Fatalf("unexpected auto-accept setting: %q", modifyGatewayOut.TransitGateway.Options.AutoAcceptSharedAttachments)
	}
	if modifyGatewayOut.TransitGateway.Options.DefaultRouteTableAssociation != awsec2types.DefaultRouteTableAssociationValueDisable {
		t.Fatalf("unexpected default association setting: %q", modifyGatewayOut.TransitGateway.Options.DefaultRouteTableAssociation)
	}
	if modifyGatewayOut.TransitGateway.Options.DefaultRouteTablePropagation != awsec2types.DefaultRouteTablePropagationValueDisable {
		t.Fatalf("unexpected default propagation setting: %q", modifyGatewayOut.TransitGateway.Options.DefaultRouteTablePropagation)
	}
	if modifyGatewayOut.TransitGateway.Options.DnsSupport != awsec2types.DnsSupportValueDisable {
		t.Fatalf("unexpected dns support setting: %q", modifyGatewayOut.TransitGateway.Options.DnsSupport)
	}
	if modifyGatewayOut.TransitGateway.Options.SecurityGroupReferencingSupport != awsec2types.SecurityGroupReferencingSupportValueEnable {
		t.Fatalf("unexpected security-group-referencing setting: %q", modifyGatewayOut.TransitGateway.Options.SecurityGroupReferencingSupport)
	}
	if modifyGatewayOut.TransitGateway.Options.VpnEcmpSupport != awsec2types.VpnEcmpSupportValueDisable {
		t.Fatalf("unexpected vpn-ecmp setting: %q", modifyGatewayOut.TransitGateway.Options.VpnEcmpSupport)
	}
	if len(modifyGatewayOut.TransitGateway.Options.TransitGatewayCidrBlocks) != 1 || modifyGatewayOut.TransitGateway.Options.TransitGatewayCidrBlocks[0] != "10.43.1.0/24" {
		t.Fatalf("unexpected transit gateway CIDR blocks: %#v", modifyGatewayOut.TransitGateway.Options.TransitGatewayCidrBlocks)
	}

	createSubnetOut, err := client.CreateSubnet(ctx, &awsec2.CreateSubnetInput{
		VpcId:            aws.String("vpc-00000001"),
		CidrBlock:        aws.String("10.0.2.0/24"),
		AvailabilityZone: aws.String("us-east-1b"),
	})
	if err != nil || createSubnetOut.Subnet == nil || createSubnetOut.Subnet.SubnetId == nil {
		t.Fatalf("create subnet: %v", err)
	}
	subnetTwoID := aws.ToString(createSubnetOut.Subnet.SubnetId)

	createAttachmentOut, err := client.CreateTransitGatewayVpcAttachment(ctx, &awsec2.CreateTransitGatewayVpcAttachmentInput{
		TransitGatewayId: aws.String(transitGatewayID),
		VpcId:            aws.String("vpc-00000001"),
		SubnetIds:        []string{"subnet-00000001"},
	})
	if err != nil || createAttachmentOut.TransitGatewayVpcAttachment == nil || createAttachmentOut.TransitGatewayVpcAttachment.TransitGatewayAttachmentId == nil {
		t.Fatalf("create transit gateway vpc attachment: %v", err)
	}
	attachmentID := aws.ToString(createAttachmentOut.TransitGatewayVpcAttachment.TransitGatewayAttachmentId)

	modifyAttachmentOut, err := client.ModifyTransitGatewayVpcAttachment(ctx, &awsec2.ModifyTransitGatewayVpcAttachmentInput{
		TransitGatewayAttachmentId: aws.String(attachmentID),
		AddSubnetIds:               []string{subnetTwoID},
		RemoveSubnetIds:            []string{"subnet-00000001"},
		Options: &awsec2types.ModifyTransitGatewayVpcAttachmentRequestOptions{
			ApplianceModeSupport:            awsec2types.ApplianceModeSupportValueEnable,
			DnsSupport:                      awsec2types.DnsSupportValueDisable,
			Ipv6Support:                     awsec2types.Ipv6SupportValueEnable,
			SecurityGroupReferencingSupport: awsec2types.SecurityGroupReferencingSupportValueEnable,
		},
	})
	if err != nil || modifyAttachmentOut.TransitGatewayVpcAttachment == nil || modifyAttachmentOut.TransitGatewayVpcAttachment.Options == nil {
		t.Fatalf("modify transit gateway vpc attachment: %v", err)
	}
	if len(modifyAttachmentOut.TransitGatewayVpcAttachment.SubnetIds) != 1 || modifyAttachmentOut.TransitGatewayVpcAttachment.SubnetIds[0] != subnetTwoID {
		t.Fatalf("unexpected transit gateway vpc attachment subnet IDs: %#v", modifyAttachmentOut.TransitGatewayVpcAttachment.SubnetIds)
	}
	if modifyAttachmentOut.TransitGatewayVpcAttachment.Options.ApplianceModeSupport != awsec2types.ApplianceModeSupportValueEnable {
		t.Fatalf("unexpected appliance mode support setting: %q", modifyAttachmentOut.TransitGatewayVpcAttachment.Options.ApplianceModeSupport)
	}
	if modifyAttachmentOut.TransitGatewayVpcAttachment.Options.DnsSupport != awsec2types.DnsSupportValueDisable {
		t.Fatalf("unexpected attachment dns support setting: %q", modifyAttachmentOut.TransitGatewayVpcAttachment.Options.DnsSupport)
	}
	if modifyAttachmentOut.TransitGatewayVpcAttachment.Options.Ipv6Support != awsec2types.Ipv6SupportValueEnable {
		t.Fatalf("unexpected attachment ipv6 support setting: %q", modifyAttachmentOut.TransitGatewayVpcAttachment.Options.Ipv6Support)
	}
	if modifyAttachmentOut.TransitGatewayVpcAttachment.Options.SecurityGroupReferencingSupport != awsec2types.SecurityGroupReferencingSupportValueEnable {
		t.Fatalf("unexpected attachment security-group-referencing setting: %q", modifyAttachmentOut.TransitGatewayVpcAttachment.Options.SecurityGroupReferencingSupport)
	}

	if _, err := client.CreateTransitGatewayPrefixListReference(ctx, &awsec2.CreateTransitGatewayPrefixListReferenceInput{
		TransitGatewayRouteTableId: aws.String(routeTableID),
		PrefixListId:               aws.String("pl-00000043"),
		Blackhole:                  aws.Bool(true),
	}); err != nil {
		t.Fatalf("create transit gateway prefix list reference: %v", err)
	}

	modifyReferenceOut, err := client.ModifyTransitGatewayPrefixListReference(ctx, &awsec2.ModifyTransitGatewayPrefixListReferenceInput{
		TransitGatewayRouteTableId: aws.String(routeTableID),
		PrefixListId:               aws.String("pl-00000043"),
		Blackhole:                  aws.Bool(false),
		TransitGatewayAttachmentId: aws.String(attachmentID),
	})
	if err != nil || modifyReferenceOut.TransitGatewayPrefixListReference == nil {
		t.Fatalf("modify transit gateway prefix list reference: %v", err)
	}
	if aws.ToBool(modifyReferenceOut.TransitGatewayPrefixListReference.Blackhole) {
		t.Fatalf("expected non-blackhole prefix list reference")
	}
	if modifyReferenceOut.TransitGatewayPrefixListReference.TransitGatewayAttachment == nil || aws.ToString(modifyReferenceOut.TransitGatewayPrefixListReference.TransitGatewayAttachment.TransitGatewayAttachmentId) != attachmentID {
		t.Fatalf("unexpected prefix list reference attachment payload")
	}
	if modifyReferenceOut.TransitGatewayPrefixListReference.State != awsec2types.TransitGatewayPrefixListReferenceStateAvailable {
		t.Fatalf("unexpected prefix list reference state: %q", modifyReferenceOut.TransitGatewayPrefixListReference.State)
	}
}

func TestEC2Stage43ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"ModifyTransitGateway",
		"ModifyTransitGatewayPrefixListReference",
		"ModifyTransitGatewayVpcAttachment",
	}

	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, action := range implemented {
		params := map[string]string{}
		switch action {
		case "ModifyTransitGateway":
			params["TransitGatewayId"] = "tgw-00000043"
		case "ModifyTransitGatewayPrefixListReference":
			params["TransitGatewayRouteTableId"] = "tgw-rtb-00000043"
			params["PrefixListId"] = "pl-00000043"
		case "ModifyTransitGatewayVpcAttachment":
			params["TransitGatewayAttachmentId"] = "tgw-attach-00000043"
		}
		resp := ec2Request(t, ts, action, params)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action)
		}
	}
}
