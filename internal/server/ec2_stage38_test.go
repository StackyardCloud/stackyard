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

func TestEC2Stage38SDKLifecycle(t *testing.T) {
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

	createGatewayOneOut, err := client.CreateTransitGateway(ctx, &awsec2.CreateTransitGatewayInput{
		Description: aws.String("stage-38-gateway-1"),
		TagSpecifications: []awsec2types.TagSpecification{
			{
				ResourceType: awsec2types.ResourceTypeTransitGateway,
				Tags:         []awsec2types.Tag{{Key: aws.String("env"), Value: aws.String("test")}},
			},
		},
	})
	if err != nil || createGatewayOneOut.TransitGateway == nil || createGatewayOneOut.TransitGateway.TransitGatewayId == nil {
		t.Fatalf("create transit gateway 1: %v", err)
	}
	transitGatewayOneID := aws.ToString(createGatewayOneOut.TransitGateway.TransitGatewayId)

	createGatewayTwoOut, err := client.CreateTransitGateway(ctx, &awsec2.CreateTransitGatewayInput{
		Description: aws.String("stage-38-gateway-2"),
	})
	if err != nil || createGatewayTwoOut.TransitGateway == nil || createGatewayTwoOut.TransitGateway.TransitGatewayId == nil {
		t.Fatalf("create transit gateway 2: %v", err)
	}
	transitGatewayTwoID := aws.ToString(createGatewayTwoOut.TransitGateway.TransitGatewayId)

	describeGatewaysPageOneOut, err := client.DescribeTransitGateways(ctx, &awsec2.DescribeTransitGatewaysInput{
		MaxResults: aws.Int32(1),
	})
	if err != nil || len(describeGatewaysPageOneOut.TransitGateways) != 1 || describeGatewaysPageOneOut.NextToken == nil {
		t.Fatalf("describe transit gateways page 1: %v", err)
	}
	describeGatewaysPageTwoOut, err := client.DescribeTransitGateways(ctx, &awsec2.DescribeTransitGatewaysInput{
		MaxResults: aws.Int32(1),
		NextToken:  describeGatewaysPageOneOut.NextToken,
	})
	if err != nil || len(describeGatewaysPageTwoOut.TransitGateways) != 1 {
		t.Fatalf("describe transit gateways page 2: %v", err)
	}
	describeGatewaysFilteredOut, err := client.DescribeTransitGateways(ctx, &awsec2.DescribeTransitGatewaysInput{
		Filters: []awsec2types.Filter{
			{Name: aws.String("state"), Values: []string{"available"}},
			{Name: aws.String("tag:env"), Values: []string{"test"}},
		},
	})
	if err != nil || len(describeGatewaysFilteredOut.TransitGateways) == 0 {
		t.Fatalf("describe transit gateways filtered: %v", err)
	}

	createVpcAttachmentOut, err := client.CreateTransitGatewayVpcAttachment(ctx, &awsec2.CreateTransitGatewayVpcAttachmentInput{
		TransitGatewayId: aws.String(transitGatewayOneID),
		VpcId:            aws.String("vpc-00000001"),
		SubnetIds:        []string{"subnet-00000001"},
	})
	if err != nil || createVpcAttachmentOut.TransitGatewayVpcAttachment == nil || createVpcAttachmentOut.TransitGatewayVpcAttachment.TransitGatewayAttachmentId == nil {
		t.Fatalf("create transit gateway vpc attachment: %v", err)
	}
	vpcAttachmentID := aws.ToString(createVpcAttachmentOut.TransitGatewayVpcAttachment.TransitGatewayAttachmentId)

	describeVpcAttachmentsOut, err := client.DescribeTransitGatewayVpcAttachments(ctx, &awsec2.DescribeTransitGatewayVpcAttachmentsInput{
		TransitGatewayAttachmentIds: []string{vpcAttachmentID},
	})
	if err != nil || len(describeVpcAttachmentsOut.TransitGatewayVpcAttachments) != 1 {
		t.Fatalf("describe transit gateway vpc attachments: %v", err)
	}

	createPeeringAttachmentOut, err := client.CreateTransitGatewayPeeringAttachment(ctx, &awsec2.CreateTransitGatewayPeeringAttachmentInput{
		TransitGatewayId:     aws.String(transitGatewayOneID),
		PeerTransitGatewayId: aws.String(transitGatewayTwoID),
		PeerAccountId:        aws.String("123456789012"),
		PeerRegion:           aws.String(testRegion),
	})
	if err != nil || createPeeringAttachmentOut.TransitGatewayPeeringAttachment == nil || createPeeringAttachmentOut.TransitGatewayPeeringAttachment.TransitGatewayAttachmentId == nil {
		t.Fatalf("create transit gateway peering attachment: %v", err)
	}
	peeringAttachmentID := aws.ToString(createPeeringAttachmentOut.TransitGatewayPeeringAttachment.TransitGatewayAttachmentId)

	describePeeringAttachmentsOut, err := client.DescribeTransitGatewayPeeringAttachments(ctx, &awsec2.DescribeTransitGatewayPeeringAttachmentsInput{
		TransitGatewayAttachmentIds: []string{peeringAttachmentID},
	})
	if err != nil || len(describePeeringAttachmentsOut.TransitGatewayPeeringAttachments) != 1 {
		t.Fatalf("describe transit gateway peering attachments: %v", err)
	}

	createConnectOut, err := client.CreateTransitGatewayConnect(ctx, &awsec2.CreateTransitGatewayConnectInput{
		TransportTransitGatewayAttachmentId: aws.String(vpcAttachmentID),
		Options: &awsec2types.CreateTransitGatewayConnectRequestOptions{
			Protocol: awsec2types.ProtocolValueGre,
		},
	})
	if err != nil || createConnectOut.TransitGatewayConnect == nil || createConnectOut.TransitGatewayConnect.TransitGatewayAttachmentId == nil {
		t.Fatalf("create transit gateway connect: %v", err)
	}
	connectID := aws.ToString(createConnectOut.TransitGatewayConnect.TransitGatewayAttachmentId)

	describeConnectsOut, err := client.DescribeTransitGatewayConnects(ctx, &awsec2.DescribeTransitGatewayConnectsInput{
		TransitGatewayAttachmentIds: []string{connectID},
		Filters: []awsec2types.Filter{
			{Name: aws.String("options.protocol"), Values: []string{"gre"}},
		},
	})
	if err != nil || len(describeConnectsOut.TransitGatewayConnects) != 1 {
		t.Fatalf("describe transit gateway connects: %v", err)
	}

	createConnectPeerOut, err := client.CreateTransitGatewayConnectPeer(ctx, &awsec2.CreateTransitGatewayConnectPeerInput{
		TransitGatewayAttachmentId: aws.String(connectID),
		InsideCidrBlocks:           []string{"169.254.110.0/29"},
		PeerAddress:                aws.String("169.254.110.2"),
		BgpOptions:                 &awsec2types.TransitGatewayConnectRequestBgpOptions{PeerAsn: aws.Int64(65010)},
	})
	if err != nil || createConnectPeerOut.TransitGatewayConnectPeer == nil || createConnectPeerOut.TransitGatewayConnectPeer.TransitGatewayConnectPeerId == nil {
		t.Fatalf("create transit gateway connect peer: %v", err)
	}
	connectPeerID := aws.ToString(createConnectPeerOut.TransitGatewayConnectPeer.TransitGatewayConnectPeerId)

	describeConnectPeersOut, err := client.DescribeTransitGatewayConnectPeers(ctx, &awsec2.DescribeTransitGatewayConnectPeersInput{
		TransitGatewayConnectPeerIds: []string{connectPeerID},
	})
	if err != nil || len(describeConnectPeersOut.TransitGatewayConnectPeers) != 1 {
		t.Fatalf("describe transit gateway connect peers: %v", err)
	}

	deleteConnectPeerOut, err := client.DeleteTransitGatewayConnectPeer(ctx, &awsec2.DeleteTransitGatewayConnectPeerInput{
		TransitGatewayConnectPeerId: aws.String(connectPeerID),
	})
	if err != nil || deleteConnectPeerOut.TransitGatewayConnectPeer == nil || deleteConnectPeerOut.TransitGatewayConnectPeer.State != awsec2types.TransitGatewayConnectPeerStateDeleted {
		t.Fatalf("delete transit gateway connect peer: %v", err)
	}

	deleteConnectOut, err := client.DeleteTransitGatewayConnect(ctx, &awsec2.DeleteTransitGatewayConnectInput{
		TransitGatewayAttachmentId: aws.String(connectID),
	})
	if err != nil || deleteConnectOut.TransitGatewayConnect == nil || deleteConnectOut.TransitGatewayConnect.State != awsec2types.TransitGatewayAttachmentStateDeleted {
		t.Fatalf("delete transit gateway connect: %v", err)
	}

	deletePeeringOut, err := client.DeleteTransitGatewayPeeringAttachment(ctx, &awsec2.DeleteTransitGatewayPeeringAttachmentInput{
		TransitGatewayAttachmentId: aws.String(peeringAttachmentID),
	})
	if err != nil || deletePeeringOut.TransitGatewayPeeringAttachment == nil || deletePeeringOut.TransitGatewayPeeringAttachment.State != awsec2types.TransitGatewayAttachmentStateDeleted {
		t.Fatalf("delete transit gateway peering attachment: %v", err)
	}

	deleteVpcAttachmentOut, err := client.DeleteTransitGatewayVpcAttachment(ctx, &awsec2.DeleteTransitGatewayVpcAttachmentInput{
		TransitGatewayAttachmentId: aws.String(vpcAttachmentID),
	})
	if err != nil || deleteVpcAttachmentOut.TransitGatewayVpcAttachment == nil || deleteVpcAttachmentOut.TransitGatewayVpcAttachment.State != awsec2types.TransitGatewayAttachmentStateDeleted {
		t.Fatalf("delete transit gateway vpc attachment: %v", err)
	}

	deleteGatewayOneOut, err := client.DeleteTransitGateway(ctx, &awsec2.DeleteTransitGatewayInput{
		TransitGatewayId: aws.String(transitGatewayOneID),
	})
	if err != nil || deleteGatewayOneOut.TransitGateway == nil || deleteGatewayOneOut.TransitGateway.State != awsec2types.TransitGatewayStateDeleted {
		t.Fatalf("delete transit gateway 1: %v", err)
	}

	deleteGatewayTwoOut, err := client.DeleteTransitGateway(ctx, &awsec2.DeleteTransitGatewayInput{
		TransitGatewayId: aws.String(transitGatewayTwoID),
	})
	if err != nil || deleteGatewayTwoOut.TransitGateway == nil || deleteGatewayTwoOut.TransitGateway.State != awsec2types.TransitGatewayStateDeleted {
		t.Fatalf("delete transit gateway 2: %v", err)
	}
}

func TestEC2Stage38ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"CreateTransitGateway",
		"DeleteTransitGateway",
		"DescribeTransitGateways",
		"CreateTransitGatewayVpcAttachment",
		"DeleteTransitGatewayVpcAttachment",
		"DescribeTransitGatewayVpcAttachments",
		"CreateTransitGatewayPeeringAttachment",
		"DeleteTransitGatewayPeeringAttachment",
		"DescribeTransitGatewayPeeringAttachments",
		"CreateTransitGatewayConnect",
		"DeleteTransitGatewayConnect",
		"DescribeTransitGatewayConnects",
		"CreateTransitGatewayConnectPeer",
		"DeleteTransitGatewayConnectPeer",
		"DescribeTransitGatewayConnectPeers",
	}

	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, action := range implemented {
		params := map[string]string{}
		switch action {
		case "DeleteTransitGateway":
			params["TransitGatewayId"] = "tgw-00000038"
		case "CreateTransitGatewayVpcAttachment":
			params["TransitGatewayId"] = "tgw-00000038"
			params["VpcId"] = "vpc-00000001"
			params["SubnetIds.1"] = "subnet-00000001"
		case "DeleteTransitGatewayVpcAttachment":
			params["TransitGatewayAttachmentId"] = "tgw-attach-00000038"
		case "CreateTransitGatewayPeeringAttachment":
			params["TransitGatewayId"] = "tgw-00000038"
			params["PeerTransitGatewayId"] = "tgw-00000039"
			params["PeerAccountId"] = "123456789012"
			params["PeerRegion"] = testRegion
		case "DeleteTransitGatewayPeeringAttachment":
			params["TransitGatewayAttachmentId"] = "tgw-attach-00000039"
		case "CreateTransitGatewayConnect":
			params["TransportTransitGatewayAttachmentId"] = "tgw-attach-00000038"
		case "DeleteTransitGatewayConnect":
			params["TransitGatewayAttachmentId"] = "tgw-attach-00000040"
		case "CreateTransitGatewayConnectPeer":
			params["TransitGatewayAttachmentId"] = "tgw-attach-00000040"
			params["InsideCidrBlocks.1"] = "169.254.110.0/29"
			params["PeerAddress"] = "169.254.110.2"
		case "DeleteTransitGatewayConnectPeer":
			params["TransitGatewayConnectPeerId"] = "tgw-connect-peer-00000038"
		}
		resp := ec2Request(t, ts, action, params)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action)
		}
	}
}
