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

func TestEC2Stage39SDKLifecycle(t *testing.T) {
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
		Description: aws.String("stage-39-gateway-1"),
	})
	if err != nil || createGatewayOneOut.TransitGateway == nil || createGatewayOneOut.TransitGateway.TransitGatewayId == nil {
		t.Fatalf("create transit gateway 1: %v", err)
	}
	transitGatewayOneID := aws.ToString(createGatewayOneOut.TransitGateway.TransitGatewayId)

	createGatewayTwoOut, err := client.CreateTransitGateway(ctx, &awsec2.CreateTransitGatewayInput{
		Description: aws.String("stage-39-gateway-2"),
	})
	if err != nil || createGatewayTwoOut.TransitGateway == nil || createGatewayTwoOut.TransitGateway.TransitGatewayId == nil {
		t.Fatalf("create transit gateway 2: %v", err)
	}
	transitGatewayTwoID := aws.ToString(createGatewayTwoOut.TransitGateway.TransitGatewayId)

	createVpcAttachmentOut, err := client.CreateTransitGatewayVpcAttachment(ctx, &awsec2.CreateTransitGatewayVpcAttachmentInput{
		TransitGatewayId: aws.String(transitGatewayOneID),
		VpcId:            aws.String("vpc-00000001"),
		SubnetIds:        []string{"subnet-00000001"},
	})
	if err != nil || createVpcAttachmentOut.TransitGatewayVpcAttachment == nil || createVpcAttachmentOut.TransitGatewayVpcAttachment.TransitGatewayAttachmentId == nil {
		t.Fatalf("create transit gateway vpc attachment: %v", err)
	}
	vpcAttachmentID := aws.ToString(createVpcAttachmentOut.TransitGatewayVpcAttachment.TransitGatewayAttachmentId)

	acceptVpcAttachmentOut, err := client.AcceptTransitGatewayVpcAttachment(ctx, &awsec2.AcceptTransitGatewayVpcAttachmentInput{
		TransitGatewayAttachmentId: aws.String(vpcAttachmentID),
	})
	if err != nil || acceptVpcAttachmentOut.TransitGatewayVpcAttachment == nil || string(acceptVpcAttachmentOut.TransitGatewayVpcAttachment.State) != "available" {
		t.Fatalf("accept transit gateway vpc attachment: %v", err)
	}

	createPeeringOneOut, err := client.CreateTransitGatewayPeeringAttachment(ctx, &awsec2.CreateTransitGatewayPeeringAttachmentInput{
		TransitGatewayId:     aws.String(transitGatewayOneID),
		PeerTransitGatewayId: aws.String(transitGatewayTwoID),
		PeerAccountId:        aws.String("123456789012"),
		PeerRegion:           aws.String(testRegion),
	})
	if err != nil || createPeeringOneOut.TransitGatewayPeeringAttachment == nil || createPeeringOneOut.TransitGatewayPeeringAttachment.TransitGatewayAttachmentId == nil {
		t.Fatalf("create transit gateway peering attachment #1: %v", err)
	}
	peeringAttachmentOneID := aws.ToString(createPeeringOneOut.TransitGatewayPeeringAttachment.TransitGatewayAttachmentId)

	rejectPeeringOut, err := client.RejectTransitGatewayPeeringAttachment(ctx, &awsec2.RejectTransitGatewayPeeringAttachmentInput{
		TransitGatewayAttachmentId: aws.String(peeringAttachmentOneID),
	})
	if err != nil || rejectPeeringOut.TransitGatewayPeeringAttachment == nil || string(rejectPeeringOut.TransitGatewayPeeringAttachment.State) != "rejected" {
		t.Fatalf("reject transit gateway peering attachment: %v", err)
	}

	createPeeringTwoOut, err := client.CreateTransitGatewayPeeringAttachment(ctx, &awsec2.CreateTransitGatewayPeeringAttachmentInput{
		TransitGatewayId:     aws.String(transitGatewayOneID),
		PeerTransitGatewayId: aws.String(transitGatewayTwoID),
		PeerAccountId:        aws.String("123456789012"),
		PeerRegion:           aws.String(testRegion),
	})
	if err != nil || createPeeringTwoOut.TransitGatewayPeeringAttachment == nil || createPeeringTwoOut.TransitGatewayPeeringAttachment.TransitGatewayAttachmentId == nil {
		t.Fatalf("create transit gateway peering attachment #2: %v", err)
	}
	peeringAttachmentTwoID := aws.ToString(createPeeringTwoOut.TransitGatewayPeeringAttachment.TransitGatewayAttachmentId)

	acceptPeeringOut, err := client.AcceptTransitGatewayPeeringAttachment(ctx, &awsec2.AcceptTransitGatewayPeeringAttachmentInput{
		TransitGatewayAttachmentId: aws.String(peeringAttachmentTwoID),
	})
	if err != nil || acceptPeeringOut.TransitGatewayPeeringAttachment == nil || string(acceptPeeringOut.TransitGatewayPeeringAttachment.State) != "available" {
		t.Fatalf("accept transit gateway peering attachment: %v", err)
	}

	createMulticastDomainOut, err := client.CreateTransitGatewayMulticastDomain(ctx, &awsec2.CreateTransitGatewayMulticastDomainInput{
		TransitGatewayId: aws.String(transitGatewayOneID),
	})
	if err != nil || createMulticastDomainOut.TransitGatewayMulticastDomain == nil || createMulticastDomainOut.TransitGatewayMulticastDomain.TransitGatewayMulticastDomainId == nil {
		t.Fatalf("create transit gateway multicast domain: %v", err)
	}
	multicastDomainID := aws.ToString(createMulticastDomainOut.TransitGatewayMulticastDomain.TransitGatewayMulticastDomainId)

	if _, err := client.AssociateTransitGatewayMulticastDomain(ctx, &awsec2.AssociateTransitGatewayMulticastDomainInput{
		TransitGatewayMulticastDomainId: aws.String(multicastDomainID),
		TransitGatewayAttachmentId:      aws.String(vpcAttachmentID),
		SubnetIds:                       []string{"subnet-00000001"},
	}); err != nil {
		t.Fatalf("associate transit gateway multicast domain: %v", err)
	}

	acceptMulticastOut, err := client.AcceptTransitGatewayMulticastDomainAssociations(ctx, &awsec2.AcceptTransitGatewayMulticastDomainAssociationsInput{
		TransitGatewayMulticastDomainId: aws.String(multicastDomainID),
		TransitGatewayAttachmentId:      aws.String(vpcAttachmentID),
		SubnetIds:                       []string{"subnet-00000001"},
	})
	if err != nil || acceptMulticastOut.Associations == nil || len(acceptMulticastOut.Associations.Subnets) != 1 || string(acceptMulticastOut.Associations.Subnets[0].State) != "associated" {
		t.Fatalf("accept transit gateway multicast domain associations: %v", err)
	}

	rejectMulticastOut, err := client.RejectTransitGatewayMulticastDomainAssociations(ctx, &awsec2.RejectTransitGatewayMulticastDomainAssociationsInput{
		TransitGatewayMulticastDomainId: aws.String(multicastDomainID),
		TransitGatewayAttachmentId:      aws.String(vpcAttachmentID),
		SubnetIds:                       []string{"subnet-00000001"},
	})
	if err != nil || rejectMulticastOut.Associations == nil || len(rejectMulticastOut.Associations.Subnets) != 1 || string(rejectMulticastOut.Associations.Subnets[0].State) != "disassociated" {
		t.Fatalf("reject transit gateway multicast domain associations: %v", err)
	}

	describeAttachmentsPageOneOut, err := client.DescribeTransitGatewayAttachments(ctx, &awsec2.DescribeTransitGatewayAttachmentsInput{
		MaxResults: aws.Int32(1),
	})
	if err != nil || len(describeAttachmentsPageOneOut.TransitGatewayAttachments) != 1 || describeAttachmentsPageOneOut.NextToken == nil {
		t.Fatalf("describe transit gateway attachments page 1: %v", err)
	}
	describeAttachmentsPageTwoOut, err := client.DescribeTransitGatewayAttachments(ctx, &awsec2.DescribeTransitGatewayAttachmentsInput{
		MaxResults: aws.Int32(1),
		NextToken:  describeAttachmentsPageOneOut.NextToken,
	})
	if err != nil || len(describeAttachmentsPageTwoOut.TransitGatewayAttachments) != 1 {
		t.Fatalf("describe transit gateway attachments page 2: %v", err)
	}

	describeAttachmentsFilteredOut, err := client.DescribeTransitGatewayAttachments(ctx, &awsec2.DescribeTransitGatewayAttachmentsInput{
		Filters: []awsec2types.Filter{
			{Name: aws.String("resource-type"), Values: []string{"peering"}},
			{Name: aws.String("state"), Values: []string{"rejected"}},
		},
	})
	if err != nil || len(describeAttachmentsFilteredOut.TransitGatewayAttachments) != 1 || aws.ToString(describeAttachmentsFilteredOut.TransitGatewayAttachments[0].TransitGatewayAttachmentId) != peeringAttachmentOneID {
		t.Fatalf("describe transit gateway attachments filtered: %v", err)
	}
}

func TestEC2Stage39ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"AcceptTransitGatewayMulticastDomainAssociations",
		"RejectTransitGatewayMulticastDomainAssociations",
		"AcceptTransitGatewayVpcAttachment",
		"RejectTransitGatewayVpcAttachment",
		"AcceptTransitGatewayPeeringAttachment",
		"RejectTransitGatewayPeeringAttachment",
		"DescribeTransitGatewayAttachments",
	}

	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, action := range implemented {
		params := map[string]string{}
		switch action {
		case "AcceptTransitGatewayMulticastDomainAssociations", "RejectTransitGatewayMulticastDomainAssociations":
			params["TransitGatewayMulticastDomainId"] = "tgw-mcast-domain-00000039"
			params["TransitGatewayAttachmentId"] = "tgw-attach-00000039"
			params["SubnetIds.1"] = "subnet-00000001"
		case "AcceptTransitGatewayVpcAttachment", "RejectTransitGatewayVpcAttachment":
			params["TransitGatewayAttachmentId"] = "tgw-attach-00000039"
		case "AcceptTransitGatewayPeeringAttachment", "RejectTransitGatewayPeeringAttachment":
			params["TransitGatewayAttachmentId"] = "tgw-attach-00000040"
		}
		resp := ec2Request(t, ts, action, params)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action)
		}
	}
}
