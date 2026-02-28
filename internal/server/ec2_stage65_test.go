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

func TestEC2Stage65SDKLifecycle(t *testing.T) {
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

	createRouteServerOneOut, err := client.CreateRouteServer(ctx, &awsec2.CreateRouteServerInput{
		AmazonSideAsn:           aws.Int64(64512),
		PersistRoutes:           awsec2types.RouteServerPersistRoutesActionEnable,
		PersistRoutesDuration:   aws.Int64(2),
		SnsNotificationsEnabled: aws.Bool(true),
		TagSpecifications: []awsec2types.TagSpecification{
			{
				ResourceType: awsec2types.ResourceTypeRouteServer,
				Tags:         []awsec2types.Tag{{Key: aws.String("env"), Value: aws.String("test")}},
			},
		},
	})
	if err != nil {
		t.Fatalf("create route server one: %v", err)
	}
	if createRouteServerOneOut.RouteServer == nil || createRouteServerOneOut.RouteServer.RouteServerId == nil {
		t.Fatalf("expected route server one id")
	}
	routeServerIDOne := aws.ToString(createRouteServerOneOut.RouteServer.RouteServerId)

	createRouteServerTwoOut, err := client.CreateRouteServer(ctx, &awsec2.CreateRouteServerInput{
		AmazonSideAsn: aws.Int64(64513),
		PersistRoutes: awsec2types.RouteServerPersistRoutesActionDisable,
	})
	if err != nil {
		t.Fatalf("create route server two: %v", err)
	}
	if createRouteServerTwoOut.RouteServer == nil || createRouteServerTwoOut.RouteServer.RouteServerId == nil {
		t.Fatalf("expected route server two id")
	}
	routeServerIDTwo := aws.ToString(createRouteServerTwoOut.RouteServer.RouteServerId)

	describeRouteServersPageOneOut, err := client.DescribeRouteServers(ctx, &awsec2.DescribeRouteServersInput{
		RouteServerIds: []string{routeServerIDOne, routeServerIDTwo},
		MaxResults:     aws.Int32(1),
	})
	if err != nil {
		t.Fatalf("describe route servers page one: %v", err)
	}
	if len(describeRouteServersPageOneOut.RouteServers) != 1 {
		t.Fatalf("expected one route server in page one, got %d", len(describeRouteServersPageOneOut.RouteServers))
	}
	if describeRouteServersPageOneOut.NextToken == nil {
		t.Fatalf("expected next token in route servers page one")
	}

	describeRouteServersPageTwoOut, err := client.DescribeRouteServers(ctx, &awsec2.DescribeRouteServersInput{
		RouteServerIds: []string{routeServerIDOne, routeServerIDTwo},
		NextToken:      describeRouteServersPageOneOut.NextToken,
	})
	if err != nil {
		t.Fatalf("describe route servers page two: %v", err)
	}
	if len(describeRouteServersPageTwoOut.RouteServers) == 0 {
		t.Fatalf("expected route server entries in page two")
	}

	describeRouteServersFilteredOut, err := client.DescribeRouteServers(ctx, &awsec2.DescribeRouteServersInput{
		Filters: []awsec2types.Filter{
			{Name: aws.String("route-server-id"), Values: []string{routeServerIDOne}},
			{Name: aws.String("persist-routes-state"), Values: []string{"enabled"}},
		},
	})
	if err != nil {
		t.Fatalf("describe route servers filtered: %v", err)
	}
	if len(describeRouteServersFilteredOut.RouteServers) != 1 {
		t.Fatalf("expected one filtered route server, got %d", len(describeRouteServersFilteredOut.RouteServers))
	}
	if aws.ToString(describeRouteServersFilteredOut.RouteServers[0].RouteServerId) != routeServerIDOne {
		t.Fatalf("unexpected filtered route server id: %q", aws.ToString(describeRouteServersFilteredOut.RouteServers[0].RouteServerId))
	}

	createEndpointOneOut, err := client.CreateRouteServerEndpoint(ctx, &awsec2.CreateRouteServerEndpointInput{
		RouteServerId: aws.String(routeServerIDOne),
		SubnetId:      aws.String("subnet-00000001"),
		TagSpecifications: []awsec2types.TagSpecification{
			{
				ResourceType: awsec2types.ResourceTypeRouteServerEndpoint,
				Tags:         []awsec2types.Tag{{Key: aws.String("team"), Value: aws.String("platform")}},
			},
		},
	})
	if err != nil {
		t.Fatalf("create route server endpoint one: %v", err)
	}
	if createEndpointOneOut.RouteServerEndpoint == nil || createEndpointOneOut.RouteServerEndpoint.RouteServerEndpointId == nil {
		t.Fatalf("expected route server endpoint one id")
	}
	endpointIDOne := aws.ToString(createEndpointOneOut.RouteServerEndpoint.RouteServerEndpointId)

	createEndpointTwoOut, err := client.CreateRouteServerEndpoint(ctx, &awsec2.CreateRouteServerEndpointInput{
		RouteServerId: aws.String(routeServerIDTwo),
		SubnetId:      aws.String("subnet-00000001"),
	})
	if err != nil {
		t.Fatalf("create route server endpoint two: %v", err)
	}
	if createEndpointTwoOut.RouteServerEndpoint == nil || createEndpointTwoOut.RouteServerEndpoint.RouteServerEndpointId == nil {
		t.Fatalf("expected route server endpoint two id")
	}
	endpointIDTwo := aws.ToString(createEndpointTwoOut.RouteServerEndpoint.RouteServerEndpointId)

	describeEndpointsPageOneOut, err := client.DescribeRouteServerEndpoints(ctx, &awsec2.DescribeRouteServerEndpointsInput{
		RouteServerEndpointIds: []string{endpointIDOne, endpointIDTwo},
		MaxResults:             aws.Int32(1),
	})
	if err != nil {
		t.Fatalf("describe route server endpoints page one: %v", err)
	}
	if len(describeEndpointsPageOneOut.RouteServerEndpoints) != 1 {
		t.Fatalf("expected one route server endpoint in page one, got %d", len(describeEndpointsPageOneOut.RouteServerEndpoints))
	}
	if describeEndpointsPageOneOut.NextToken == nil {
		t.Fatalf("expected next token in route server endpoints page one")
	}

	describeEndpointsPageTwoOut, err := client.DescribeRouteServerEndpoints(ctx, &awsec2.DescribeRouteServerEndpointsInput{
		RouteServerEndpointIds: []string{endpointIDOne, endpointIDTwo},
		NextToken:              describeEndpointsPageOneOut.NextToken,
	})
	if err != nil {
		t.Fatalf("describe route server endpoints page two: %v", err)
	}
	if len(describeEndpointsPageTwoOut.RouteServerEndpoints) == 0 {
		t.Fatalf("expected route server endpoint entries in page two")
	}

	describeEndpointsFilteredOut, err := client.DescribeRouteServerEndpoints(ctx, &awsec2.DescribeRouteServerEndpointsInput{
		Filters: []awsec2types.Filter{
			{Name: aws.String("route-server-id"), Values: []string{routeServerIDOne}},
			{Name: aws.String("subnet-id"), Values: []string{"subnet-00000001"}},
		},
	})
	if err != nil {
		t.Fatalf("describe route server endpoints filtered: %v", err)
	}
	if len(describeEndpointsFilteredOut.RouteServerEndpoints) != 1 {
		t.Fatalf("expected one filtered route server endpoint, got %d", len(describeEndpointsFilteredOut.RouteServerEndpoints))
	}
	if aws.ToString(describeEndpointsFilteredOut.RouteServerEndpoints[0].RouteServerEndpointId) != endpointIDOne {
		t.Fatalf("unexpected filtered endpoint id: %q", aws.ToString(describeEndpointsFilteredOut.RouteServerEndpoints[0].RouteServerEndpointId))
	}

	createPeerOneOut, err := client.CreateRouteServerPeer(ctx, &awsec2.CreateRouteServerPeerInput{
		RouteServerEndpointId: aws.String(endpointIDOne),
		PeerAddress:           aws.String("169.254.100.1"),
		BgpOptions: &awsec2types.RouteServerBgpOptionsRequest{
			PeerAsn:               aws.Int64(65010),
			PeerLivenessDetection: awsec2types.RouteServerPeerLivenessModeBfd,
		},
		TagSpecifications: []awsec2types.TagSpecification{
			{
				ResourceType: awsec2types.ResourceTypeRouteServerPeer,
				Tags:         []awsec2types.Tag{{Key: aws.String("purpose"), Value: aws.String("test")}},
			},
		},
	})
	if err != nil {
		t.Fatalf("create route server peer one: %v", err)
	}
	if createPeerOneOut.RouteServerPeer == nil || createPeerOneOut.RouteServerPeer.RouteServerPeerId == nil {
		t.Fatalf("expected route server peer one id")
	}
	peerIDOne := aws.ToString(createPeerOneOut.RouteServerPeer.RouteServerPeerId)

	createPeerTwoOut, err := client.CreateRouteServerPeer(ctx, &awsec2.CreateRouteServerPeerInput{
		RouteServerEndpointId: aws.String(endpointIDOne),
		PeerAddress:           aws.String("169.254.100.2"),
		BgpOptions: &awsec2types.RouteServerBgpOptionsRequest{
			PeerAsn:               aws.Int64(65011),
			PeerLivenessDetection: awsec2types.RouteServerPeerLivenessModeBgpKeepalive,
		},
	})
	if err != nil {
		t.Fatalf("create route server peer two: %v", err)
	}
	if createPeerTwoOut.RouteServerPeer == nil || createPeerTwoOut.RouteServerPeer.RouteServerPeerId == nil {
		t.Fatalf("expected route server peer two id")
	}
	peerIDTwo := aws.ToString(createPeerTwoOut.RouteServerPeer.RouteServerPeerId)

	describePeersPageOneOut, err := client.DescribeRouteServerPeers(ctx, &awsec2.DescribeRouteServerPeersInput{
		RouteServerPeerIds: []string{peerIDOne, peerIDTwo},
		MaxResults:         aws.Int32(1),
	})
	if err != nil {
		t.Fatalf("describe route server peers page one: %v", err)
	}
	if len(describePeersPageOneOut.RouteServerPeers) != 1 {
		t.Fatalf("expected one route server peer in page one, got %d", len(describePeersPageOneOut.RouteServerPeers))
	}
	if describePeersPageOneOut.NextToken == nil {
		t.Fatalf("expected next token in route server peers page one")
	}

	describePeersPageTwoOut, err := client.DescribeRouteServerPeers(ctx, &awsec2.DescribeRouteServerPeersInput{
		RouteServerPeerIds: []string{peerIDOne, peerIDTwo},
		NextToken:          describePeersPageOneOut.NextToken,
	})
	if err != nil {
		t.Fatalf("describe route server peers page two: %v", err)
	}
	if len(describePeersPageTwoOut.RouteServerPeers) == 0 {
		t.Fatalf("expected route server peer entries in page two")
	}

	describePeersFilteredOut, err := client.DescribeRouteServerPeers(ctx, &awsec2.DescribeRouteServerPeersInput{
		Filters: []awsec2types.Filter{
			{Name: aws.String("route-server-endpoint-id"), Values: []string{endpointIDOne}},
			{Name: aws.String("peer-address"), Values: []string{"169.254.100.1"}},
			{Name: aws.String("bgp-session-state"), Values: []string{"up"}},
		},
	})
	if err != nil {
		t.Fatalf("describe route server peers filtered: %v", err)
	}
	if len(describePeersFilteredOut.RouteServerPeers) != 1 {
		t.Fatalf("expected one filtered route server peer, got %d", len(describePeersFilteredOut.RouteServerPeers))
	}
	if aws.ToString(describePeersFilteredOut.RouteServerPeers[0].RouteServerPeerId) != peerIDOne {
		t.Fatalf("unexpected filtered peer id: %q", aws.ToString(describePeersFilteredOut.RouteServerPeers[0].RouteServerPeerId))
	}

	if _, err := client.DeleteRouteServerPeer(ctx, &awsec2.DeleteRouteServerPeerInput{RouteServerPeerId: aws.String(peerIDOne)}); err != nil {
		t.Fatalf("delete route server peer one: %v", err)
	}
	if _, err := client.DeleteRouteServerPeer(ctx, &awsec2.DeleteRouteServerPeerInput{RouteServerPeerId: aws.String(peerIDTwo)}); err != nil {
		t.Fatalf("delete route server peer two: %v", err)
	}
	if _, err := client.DeleteRouteServerEndpoint(ctx, &awsec2.DeleteRouteServerEndpointInput{RouteServerEndpointId: aws.String(endpointIDOne)}); err != nil {
		t.Fatalf("delete route server endpoint one: %v", err)
	}
	if _, err := client.DeleteRouteServerEndpoint(ctx, &awsec2.DeleteRouteServerEndpointInput{RouteServerEndpointId: aws.String(endpointIDTwo)}); err != nil {
		t.Fatalf("delete route server endpoint two: %v", err)
	}
	if _, err := client.DeleteRouteServer(ctx, &awsec2.DeleteRouteServerInput{RouteServerId: aws.String(routeServerIDOne)}); err != nil {
		t.Fatalf("delete route server one: %v", err)
	}
	if _, err := client.DeleteRouteServer(ctx, &awsec2.DeleteRouteServerInput{RouteServerId: aws.String(routeServerIDTwo)}); err != nil {
		t.Fatalf("delete route server two: %v", err)
	}
}

func TestEC2Stage65ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"CreateRouteServer",
		"DeleteRouteServer",
		"DescribeRouteServers",
		"CreateRouteServerEndpoint",
		"DeleteRouteServerEndpoint",
		"DescribeRouteServerEndpoints",
		"CreateRouteServerPeer",
		"DeleteRouteServerPeer",
		"DescribeRouteServerPeers",
	}

	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, action := range implemented {
		params := map[string]string{}
		switch action {
		case "CreateRouteServer":
			params["AmazonSideAsn"] = "64512"
		case "DeleteRouteServer":
			params["RouteServerId"] = "rsrv-1234"
		case "CreateRouteServerEndpoint":
			params["RouteServerId"] = "rsrv-1234"
			params["SubnetId"] = "subnet-00000001"
		case "DeleteRouteServerEndpoint":
			params["RouteServerEndpointId"] = "rse-1234"
		case "CreateRouteServerPeer":
			params["RouteServerEndpointId"] = "rse-1234"
			params["PeerAddress"] = "169.254.100.1"
			params["BgpOptions.PeerAsn"] = "65010"
		case "DeleteRouteServerPeer":
			params["RouteServerPeerId"] = "rsp-1234"
		}
		resp := ec2Request(t, ts, action, params)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action)
		}
	}
}
