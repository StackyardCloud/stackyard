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

func TestEC2Stage41SDKLifecycle(t *testing.T) {
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
		Description: aws.String("stage-41-gateway"),
	})
	if err != nil || createGatewayOut.TransitGateway == nil || createGatewayOut.TransitGateway.TransitGatewayId == nil {
		t.Fatalf("create transit gateway: %v", err)
	}
	transitGatewayID := aws.ToString(createGatewayOut.TransitGateway.TransitGatewayId)

	createPeerGatewayOut, err := client.CreateTransitGateway(ctx, &awsec2.CreateTransitGatewayInput{
		Description: aws.String("stage-41-peer-gateway"),
	})
	if err != nil || createPeerGatewayOut.TransitGateway == nil || createPeerGatewayOut.TransitGateway.TransitGatewayId == nil {
		t.Fatalf("create peer transit gateway: %v", err)
	}
	peerTransitGatewayID := aws.ToString(createPeerGatewayOut.TransitGateway.TransitGatewayId)

	createRouteTableOut, err := client.CreateTransitGatewayRouteTable(ctx, &awsec2.CreateTransitGatewayRouteTableInput{
		TransitGatewayId: aws.String(transitGatewayID),
	})
	if err != nil || createRouteTableOut.TransitGatewayRouteTable == nil || createRouteTableOut.TransitGatewayRouteTable.TransitGatewayRouteTableId == nil {
		t.Fatalf("create transit gateway route table: %v", err)
	}
	routeTableID := aws.ToString(createRouteTableOut.TransitGatewayRouteTable.TransitGatewayRouteTableId)

	createPeeringOneOut, err := client.CreateTransitGatewayPeeringAttachment(ctx, &awsec2.CreateTransitGatewayPeeringAttachmentInput{
		TransitGatewayId:     aws.String(transitGatewayID),
		PeerTransitGatewayId: aws.String(peerTransitGatewayID),
		PeerAccountId:        aws.String("123456789012"),
		PeerRegion:           aws.String(testRegion),
	})
	if err != nil || createPeeringOneOut.TransitGatewayPeeringAttachment == nil || createPeeringOneOut.TransitGatewayPeeringAttachment.TransitGatewayAttachmentId == nil {
		t.Fatalf("create transit gateway peering attachment #1: %v", err)
	}
	peeringAttachmentOneID := aws.ToString(createPeeringOneOut.TransitGatewayPeeringAttachment.TransitGatewayAttachmentId)

	createAnnouncementOneOut, err := client.CreateTransitGatewayRouteTableAnnouncement(ctx, &awsec2.CreateTransitGatewayRouteTableAnnouncementInput{
		TransitGatewayRouteTableId: aws.String(routeTableID),
		PeeringAttachmentId:        aws.String(peeringAttachmentOneID),
		TagSpecifications: []awsec2types.TagSpecification{{
			ResourceType: awsec2types.ResourceTypeTransitGatewayRouteTableAnnouncement,
			Tags:         []awsec2types.Tag{{Key: aws.String("env"), Value: aws.String("stage41")}},
		}},
	})
	if err != nil || createAnnouncementOneOut.TransitGatewayRouteTableAnnouncement == nil || createAnnouncementOneOut.TransitGatewayRouteTableAnnouncement.TransitGatewayRouteTableAnnouncementId == nil {
		t.Fatalf("create transit gateway route table announcement #1: %v", err)
	}
	announcementOneID := aws.ToString(createAnnouncementOneOut.TransitGatewayRouteTableAnnouncement.TransitGatewayRouteTableAnnouncementId)
	if createAnnouncementOneOut.TransitGatewayRouteTableAnnouncement.State != awsec2types.TransitGatewayRouteTableAnnouncementStateAvailable {
		t.Fatalf("unexpected announcement #1 state: %q", createAnnouncementOneOut.TransitGatewayRouteTableAnnouncement.State)
	}
	if createAnnouncementOneOut.TransitGatewayRouteTableAnnouncement.AnnouncementDirection != awsec2types.TransitGatewayRouteTableAnnouncementDirectionOutgoing {
		t.Fatalf("unexpected announcement #1 direction: %q", createAnnouncementOneOut.TransitGatewayRouteTableAnnouncement.AnnouncementDirection)
	}

	createPeeringTwoOut, err := client.CreateTransitGatewayPeeringAttachment(ctx, &awsec2.CreateTransitGatewayPeeringAttachmentInput{
		TransitGatewayId:     aws.String(transitGatewayID),
		PeerTransitGatewayId: aws.String(peerTransitGatewayID),
		PeerAccountId:        aws.String("123456789012"),
		PeerRegion:           aws.String(testRegion),
	})
	if err != nil || createPeeringTwoOut.TransitGatewayPeeringAttachment == nil || createPeeringTwoOut.TransitGatewayPeeringAttachment.TransitGatewayAttachmentId == nil {
		t.Fatalf("create transit gateway peering attachment #2: %v", err)
	}
	peeringAttachmentTwoID := aws.ToString(createPeeringTwoOut.TransitGatewayPeeringAttachment.TransitGatewayAttachmentId)

	createAnnouncementTwoOut, err := client.CreateTransitGatewayRouteTableAnnouncement(ctx, &awsec2.CreateTransitGatewayRouteTableAnnouncementInput{
		TransitGatewayRouteTableId: aws.String(routeTableID),
		PeeringAttachmentId:        aws.String(peeringAttachmentTwoID),
	})
	if err != nil || createAnnouncementTwoOut.TransitGatewayRouteTableAnnouncement == nil || createAnnouncementTwoOut.TransitGatewayRouteTableAnnouncement.TransitGatewayRouteTableAnnouncementId == nil {
		t.Fatalf("create transit gateway route table announcement #2: %v", err)
	}

	describePageOneOut, err := client.DescribeTransitGatewayRouteTableAnnouncements(ctx, &awsec2.DescribeTransitGatewayRouteTableAnnouncementsInput{
		MaxResults: aws.Int32(1),
	})
	if err != nil || len(describePageOneOut.TransitGatewayRouteTableAnnouncements) != 1 || describePageOneOut.NextToken == nil {
		t.Fatalf("describe transit gateway route table announcements page 1: %v", err)
	}
	describePageTwoOut, err := client.DescribeTransitGatewayRouteTableAnnouncements(ctx, &awsec2.DescribeTransitGatewayRouteTableAnnouncementsInput{
		MaxResults: aws.Int32(1),
		NextToken:  describePageOneOut.NextToken,
	})
	if err != nil || len(describePageTwoOut.TransitGatewayRouteTableAnnouncements) != 1 {
		t.Fatalf("describe transit gateway route table announcements page 2: %v", err)
	}

	describeFilteredOut, err := client.DescribeTransitGatewayRouteTableAnnouncements(ctx, &awsec2.DescribeTransitGatewayRouteTableAnnouncementsInput{
		Filters: []awsec2types.Filter{
			{Name: aws.String("peering-attachment-id"), Values: []string{peeringAttachmentOneID}},
			{Name: aws.String("state"), Values: []string{"available"}},
			{Name: aws.String("announcement-direction"), Values: []string{"outgoing"}},
			{Name: aws.String("tag:env"), Values: []string{"stage41"}},
		},
	})
	if err != nil || len(describeFilteredOut.TransitGatewayRouteTableAnnouncements) != 1 || aws.ToString(describeFilteredOut.TransitGatewayRouteTableAnnouncements[0].TransitGatewayRouteTableAnnouncementId) != announcementOneID {
		t.Fatalf("describe transit gateway route table announcements filtered: %v", err)
	}

	deleteAnnouncementOut, err := client.DeleteTransitGatewayRouteTableAnnouncement(ctx, &awsec2.DeleteTransitGatewayRouteTableAnnouncementInput{
		TransitGatewayRouteTableAnnouncementId: aws.String(announcementOneID),
	})
	if err != nil || deleteAnnouncementOut.TransitGatewayRouteTableAnnouncement == nil || deleteAnnouncementOut.TransitGatewayRouteTableAnnouncement.State != awsec2types.TransitGatewayRouteTableAnnouncementStateDeleted {
		t.Fatalf("delete transit gateway route table announcement: %v", err)
	}

	describeDeletedOut, err := client.DescribeTransitGatewayRouteTableAnnouncements(ctx, &awsec2.DescribeTransitGatewayRouteTableAnnouncementsInput{
		TransitGatewayRouteTableAnnouncementIds: []string{announcementOneID},
	})
	if err != nil || len(describeDeletedOut.TransitGatewayRouteTableAnnouncements) != 0 {
		t.Fatalf("describe deleted transit gateway route table announcement: %v", err)
	}
}

func TestEC2Stage41ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"CreateTransitGatewayRouteTableAnnouncement",
		"DeleteTransitGatewayRouteTableAnnouncement",
		"DescribeTransitGatewayRouteTableAnnouncements",
	}

	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, action := range implemented {
		params := map[string]string{}
		switch action {
		case "CreateTransitGatewayRouteTableAnnouncement":
			params["TransitGatewayRouteTableId"] = "tgw-rtb-00000041"
			params["PeeringAttachmentId"] = "tgw-attach-00000041"
		case "DeleteTransitGatewayRouteTableAnnouncement":
			params["TransitGatewayRouteTableAnnouncementId"] = "tgw-rtb-announcement-00000041"
		}
		resp := ec2Request(t, ts, action, params)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action)
		}
	}
}
