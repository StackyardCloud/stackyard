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

func TestEC2Stage42SDKLifecycle(t *testing.T) {
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

	createGatewayOut, err := client.CreateTransitGateway(ctx, &awsec2.CreateTransitGatewayInput{Description: aws.String("stage-42-gateway")})
	if err != nil || createGatewayOut.TransitGateway == nil || createGatewayOut.TransitGateway.TransitGatewayId == nil {
		t.Fatalf("create transit gateway: %v", err)
	}
	transitGatewayID := aws.ToString(createGatewayOut.TransitGateway.TransitGatewayId)

	createAttachmentOut, err := client.CreateTransitGatewayVpcAttachment(ctx, &awsec2.CreateTransitGatewayVpcAttachmentInput{
		TransitGatewayId: aws.String(transitGatewayID),
		VpcId:            aws.String("vpc-00000001"),
		SubnetIds:        []string{"subnet-00000001"},
	})
	if err != nil || createAttachmentOut.TransitGatewayVpcAttachment == nil || createAttachmentOut.TransitGatewayVpcAttachment.TransitGatewayAttachmentId == nil {
		t.Fatalf("create transit gateway vpc attachment: %v", err)
	}
	attachmentID := aws.ToString(createAttachmentOut.TransitGatewayVpcAttachment.TransitGatewayAttachmentId)

	createDomainOut, err := client.CreateTransitGatewayMulticastDomain(ctx, &awsec2.CreateTransitGatewayMulticastDomainInput{
		TransitGatewayId: aws.String(transitGatewayID),
	})
	if err != nil || createDomainOut.TransitGatewayMulticastDomain == nil || createDomainOut.TransitGatewayMulticastDomain.TransitGatewayMulticastDomainId == nil {
		t.Fatalf("create transit gateway multicast domain: %v", err)
	}
	domainID := aws.ToString(createDomainOut.TransitGatewayMulticastDomain.TransitGatewayMulticastDomainId)

	createENIOneOut, err := client.CreateNetworkInterface(ctx, &awsec2.CreateNetworkInterfaceInput{SubnetId: aws.String("subnet-00000001")})
	if err != nil || createENIOneOut.NetworkInterface == nil || createENIOneOut.NetworkInterface.NetworkInterfaceId == nil {
		t.Fatalf("create network interface #1: %v", err)
	}
	eniOneID := aws.ToString(createENIOneOut.NetworkInterface.NetworkInterfaceId)

	createENITwoOut, err := client.CreateNetworkInterface(ctx, &awsec2.CreateNetworkInterfaceInput{SubnetId: aws.String("subnet-00000001")})
	if err != nil || createENITwoOut.NetworkInterface == nil || createENITwoOut.NetworkInterface.NetworkInterfaceId == nil {
		t.Fatalf("create network interface #2: %v", err)
	}
	eniTwoID := aws.ToString(createENITwoOut.NetworkInterface.NetworkInterfaceId)

	registerMembersOut, err := client.RegisterTransitGatewayMulticastGroupMembers(ctx, &awsec2.RegisterTransitGatewayMulticastGroupMembersInput{
		TransitGatewayMulticastDomainId: aws.String(domainID),
		GroupIpAddress:                  aws.String("239.1.1.1"),
		NetworkInterfaceIds:             []string{eniOneID, eniTwoID},
	})
	if err != nil || registerMembersOut.RegisteredMulticastGroupMembers == nil || len(registerMembersOut.RegisteredMulticastGroupMembers.RegisteredNetworkInterfaceIds) != 2 {
		t.Fatalf("register transit gateway multicast group members: %v", err)
	}

	registerSourcesOut, err := client.RegisterTransitGatewayMulticastGroupSources(ctx, &awsec2.RegisterTransitGatewayMulticastGroupSourcesInput{
		TransitGatewayMulticastDomainId: aws.String(domainID),
		GroupIpAddress:                  aws.String("239.1.1.1"),
		NetworkInterfaceIds:             []string{eniOneID},
	})
	if err != nil || registerSourcesOut.RegisteredMulticastGroupSources == nil || len(registerSourcesOut.RegisteredMulticastGroupSources.RegisteredNetworkInterfaceIds) != 1 {
		t.Fatalf("register transit gateway multicast group sources: %v", err)
	}

	searchPageOneOut, err := client.SearchTransitGatewayMulticastGroups(ctx, &awsec2.SearchTransitGatewayMulticastGroupsInput{
		TransitGatewayMulticastDomainId: aws.String(domainID),
		MaxResults:                      aws.Int32(1),
		Filters: []awsec2types.Filter{
			{Name: aws.String("group-ip-address"), Values: []string{"239.1.1.1"}},
			{Name: aws.String("is-group-member"), Values: []string{"true"}},
		},
	})
	if err != nil || len(searchPageOneOut.MulticastGroups) != 1 || searchPageOneOut.NextToken == nil {
		t.Fatalf("search transit gateway multicast groups page 1: %v", err)
	}

	searchPageTwoOut, err := client.SearchTransitGatewayMulticastGroups(ctx, &awsec2.SearchTransitGatewayMulticastGroupsInput{
		TransitGatewayMulticastDomainId: aws.String(domainID),
		MaxResults:                      aws.Int32(1),
		NextToken:                       searchPageOneOut.NextToken,
		Filters: []awsec2types.Filter{
			{Name: aws.String("group-ip-address"), Values: []string{"239.1.1.1"}},
			{Name: aws.String("is-group-member"), Values: []string{"true"}},
		},
	})
	if err != nil || len(searchPageTwoOut.MulticastGroups) != 1 {
		t.Fatalf("search transit gateway multicast groups page 2: %v", err)
	}

	searchSourceOut, err := client.SearchTransitGatewayMulticastGroups(ctx, &awsec2.SearchTransitGatewayMulticastGroupsInput{
		TransitGatewayMulticastDomainId: aws.String(domainID),
		Filters: []awsec2types.Filter{
			{Name: aws.String("is-group-source"), Values: []string{"true"}},
			{Name: aws.String("member-type"), Values: []string{"static"}},
			{Name: aws.String("transit-gateway-attachment-id"), Values: []string{attachmentID}},
		},
	})
	if err != nil || len(searchSourceOut.MulticastGroups) != 1 || aws.ToString(searchSourceOut.MulticastGroups[0].NetworkInterfaceId) != eniOneID {
		t.Fatalf("search transit gateway multicast groups source filter: %v", err)
	}

	deregisterMembersOut, err := client.DeregisterTransitGatewayMulticastGroupMembers(ctx, &awsec2.DeregisterTransitGatewayMulticastGroupMembersInput{
		TransitGatewayMulticastDomainId: aws.String(domainID),
		GroupIpAddress:                  aws.String("239.1.1.1"),
		NetworkInterfaceIds:             []string{eniTwoID},
	})
	if err != nil || deregisterMembersOut.DeregisteredMulticastGroupMembers == nil || len(deregisterMembersOut.DeregisteredMulticastGroupMembers.DeregisteredNetworkInterfaceIds) != 1 {
		t.Fatalf("deregister transit gateway multicast group members: %v", err)
	}

	deregisterSourcesOut, err := client.DeregisterTransitGatewayMulticastGroupSources(ctx, &awsec2.DeregisterTransitGatewayMulticastGroupSourcesInput{
		TransitGatewayMulticastDomainId: aws.String(domainID),
		GroupIpAddress:                  aws.String("239.1.1.1"),
		NetworkInterfaceIds:             []string{eniOneID},
	})
	if err != nil || deregisterSourcesOut.DeregisteredMulticastGroupSources == nil || len(deregisterSourcesOut.DeregisteredMulticastGroupSources.DeregisteredNetworkInterfaceIds) != 1 {
		t.Fatalf("deregister transit gateway multicast group sources: %v", err)
	}

	searchNoSourcesOut, err := client.SearchTransitGatewayMulticastGroups(ctx, &awsec2.SearchTransitGatewayMulticastGroupsInput{
		TransitGatewayMulticastDomainId: aws.String(domainID),
		Filters: []awsec2types.Filter{{
			Name: aws.String("is-group-source"), Values: []string{"true"},
		}},
	})
	if err != nil || len(searchNoSourcesOut.MulticastGroups) != 0 {
		t.Fatalf("search transit gateway multicast groups after source deregister: %v", err)
	}
}

func TestEC2Stage42ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"RegisterTransitGatewayMulticastGroupMembers",
		"DeregisterTransitGatewayMulticastGroupMembers",
		"RegisterTransitGatewayMulticastGroupSources",
		"DeregisterTransitGatewayMulticastGroupSources",
		"SearchTransitGatewayMulticastGroups",
	}

	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, action := range implemented {
		params := map[string]string{}
		switch action {
		case "RegisterTransitGatewayMulticastGroupMembers", "DeregisterTransitGatewayMulticastGroupMembers", "RegisterTransitGatewayMulticastGroupSources", "DeregisterTransitGatewayMulticastGroupSources":
			params["TransitGatewayMulticastDomainId"] = "tgw-mcast-domain-00000042"
			params["GroupIpAddress"] = "239.1.1.1"
			params["NetworkInterfaceIds.1"] = "eni-00000042"
		case "SearchTransitGatewayMulticastGroups":
			params["TransitGatewayMulticastDomainId"] = "tgw-mcast-domain-00000042"
		}
		resp := ec2Request(t, ts, action, params)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action)
		}
	}
}
