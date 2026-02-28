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

func TestEC2Stage66SDKLifecycle(t *testing.T) {
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

	createRouteServerOut, err := client.CreateRouteServer(ctx, &awsec2.CreateRouteServerInput{
		AmazonSideAsn:           aws.Int64(65000),
		PersistRoutes:           awsec2types.RouteServerPersistRoutesActionEnable,
		PersistRoutesDuration:   aws.Int64(2),
		SnsNotificationsEnabled: aws.Bool(true),
	})
	if err != nil {
		t.Fatalf("create route server: %v", err)
	}
	if createRouteServerOut.RouteServer == nil || createRouteServerOut.RouteServer.RouteServerId == nil {
		t.Fatalf("expected route server id")
	}
	routeServerID := aws.ToString(createRouteServerOut.RouteServer.RouteServerId)

	associateOut, err := client.AssociateRouteServer(ctx, &awsec2.AssociateRouteServerInput{
		RouteServerId: aws.String(routeServerID),
		VpcId:         aws.String("vpc-00000001"),
	})
	if err != nil {
		t.Fatalf("associate route server: %v", err)
	}
	if associateOut.RouteServerAssociation == nil || aws.ToString(associateOut.RouteServerAssociation.VpcId) != "vpc-00000001" {
		t.Fatalf("expected associated vpc")
	}

	associationsOut, err := client.GetRouteServerAssociations(ctx, &awsec2.GetRouteServerAssociationsInput{
		RouteServerId: aws.String(routeServerID),
	})
	if err != nil {
		t.Fatalf("get route server associations: %v", err)
	}
	if len(associationsOut.RouteServerAssociations) != 1 {
		t.Fatalf("expected one route server association, got %d", len(associationsOut.RouteServerAssociations))
	}

	enablePropagationOut, err := client.EnableRouteServerPropagation(ctx, &awsec2.EnableRouteServerPropagationInput{
		RouteServerId: aws.String(routeServerID),
		RouteTableId:  aws.String("rtb-00000001"),
	})
	if err != nil {
		t.Fatalf("enable route server propagation: %v", err)
	}
	if enablePropagationOut.RouteServerPropagation == nil || aws.ToString(enablePropagationOut.RouteServerPropagation.RouteTableId) != "rtb-00000001" {
		t.Fatalf("expected enabled route table propagation")
	}

	propagationsOut, err := client.GetRouteServerPropagations(ctx, &awsec2.GetRouteServerPropagationsInput{
		RouteServerId: aws.String(routeServerID),
	})
	if err != nil {
		t.Fatalf("get route server propagations: %v", err)
	}
	if len(propagationsOut.RouteServerPropagations) != 1 {
		t.Fatalf("expected one route server propagation, got %d", len(propagationsOut.RouteServerPropagations))
	}

	_, err = client.GetRouteServerPropagations(ctx, &awsec2.GetRouteServerPropagationsInput{
		RouteServerId: aws.String(routeServerID),
		RouteTableId:  aws.String("rtb-00000001"),
	})
	if err != nil {
		t.Fatalf("get route server propagations filtered: %v", err)
	}

	createEndpointOut, err := client.CreateRouteServerEndpoint(ctx, &awsec2.CreateRouteServerEndpointInput{
		RouteServerId: aws.String(routeServerID),
		SubnetId:      aws.String("subnet-00000001"),
	})
	if err != nil {
		t.Fatalf("create route server endpoint: %v", err)
	}
	if createEndpointOut.RouteServerEndpoint == nil || createEndpointOut.RouteServerEndpoint.RouteServerEndpointId == nil {
		t.Fatalf("expected route server endpoint id")
	}
	endpointID := aws.ToString(createEndpointOut.RouteServerEndpoint.RouteServerEndpointId)

	createPeerOneOut, err := client.CreateRouteServerPeer(ctx, &awsec2.CreateRouteServerPeerInput{
		RouteServerEndpointId: aws.String(endpointID),
		PeerAddress:           aws.String("169.254.10.1"),
		BgpOptions:            &awsec2types.RouteServerBgpOptionsRequest{PeerAsn: aws.Int64(65101)},
	})
	if err != nil {
		t.Fatalf("create route server peer one: %v", err)
	}
	if createPeerOneOut.RouteServerPeer == nil || createPeerOneOut.RouteServerPeer.RouteServerPeerId == nil {
		t.Fatalf("expected route server peer one id")
	}
	peerIDOne := aws.ToString(createPeerOneOut.RouteServerPeer.RouteServerPeerId)

	createPeerTwoOut, err := client.CreateRouteServerPeer(ctx, &awsec2.CreateRouteServerPeerInput{
		RouteServerEndpointId: aws.String(endpointID),
		PeerAddress:           aws.String("169.254.10.2"),
		BgpOptions:            &awsec2types.RouteServerBgpOptionsRequest{PeerAsn: aws.Int64(65102)},
	})
	if err != nil {
		t.Fatalf("create route server peer two: %v", err)
	}
	if createPeerTwoOut.RouteServerPeer == nil || createPeerTwoOut.RouteServerPeer.RouteServerPeerId == nil {
		t.Fatalf("expected route server peer two id")
	}
	peerIDTwo := aws.ToString(createPeerTwoOut.RouteServerPeer.RouteServerPeerId)

	routesPageOneOut, err := client.GetRouteServerRoutingDatabase(ctx, &awsec2.GetRouteServerRoutingDatabaseInput{
		RouteServerId: aws.String(routeServerID),
		MaxResults:    aws.Int32(1),
	})
	if err != nil {
		t.Fatalf("get route server routing database page one: %v", err)
	}
	if len(routesPageOneOut.Routes) != 1 {
		t.Fatalf("expected one route in first page, got %d", len(routesPageOneOut.Routes))
	}
	if routesPageOneOut.NextToken == nil {
		t.Fatalf("expected next token in first routing database page")
	}
	if !aws.ToBool(routesPageOneOut.AreRoutesPersisted) {
		t.Fatalf("expected routes persisted true")
	}

	routesPageTwoOut, err := client.GetRouteServerRoutingDatabase(ctx, &awsec2.GetRouteServerRoutingDatabaseInput{
		RouteServerId: aws.String(routeServerID),
		NextToken:     routesPageOneOut.NextToken,
	})
	if err != nil {
		t.Fatalf("get route server routing database page two: %v", err)
	}
	if len(routesPageTwoOut.Routes) == 0 {
		t.Fatalf("expected at least one route in second page")
	}

	routesFilteredOut, err := client.GetRouteServerRoutingDatabase(ctx, &awsec2.GetRouteServerRoutingDatabaseInput{
		RouteServerId: aws.String(routeServerID),
		Filters: []awsec2types.Filter{
			{Name: aws.String("route-server-peer-id"), Values: []string{peerIDOne}},
			{Name: aws.String("route-status"), Values: []string{"in-fib"}},
			{Name: aws.String("route-table-id"), Values: []string{"rtb-00000001"}},
		},
	})
	if err != nil {
		t.Fatalf("get route server routing database filtered: %v", err)
	}
	if len(routesFilteredOut.Routes) != 1 {
		t.Fatalf("expected one filtered route, got %d", len(routesFilteredOut.Routes))
	}
	if aws.ToString(routesFilteredOut.Routes[0].RouteServerPeerId) != peerIDOne {
		t.Fatalf("unexpected filtered route peer id: %q", aws.ToString(routesFilteredOut.Routes[0].RouteServerPeerId))
	}

	modifyRouteServerOut, err := client.ModifyRouteServer(ctx, &awsec2.ModifyRouteServerInput{
		RouteServerId:           aws.String(routeServerID),
		PersistRoutes:           awsec2types.RouteServerPersistRoutesActionDisable,
		SnsNotificationsEnabled: aws.Bool(false),
	})
	if err != nil {
		t.Fatalf("modify route server: %v", err)
	}
	if modifyRouteServerOut.RouteServer == nil {
		t.Fatalf("expected modified route server")
	}
	if modifyRouteServerOut.RouteServer.PersistRoutesState != awsec2types.RouteServerPersistRoutesStateDisabled {
		t.Fatalf("expected persist routes disabled, got %q", modifyRouteServerOut.RouteServer.PersistRoutesState)
	}

	disablePropagationOut, err := client.DisableRouteServerPropagation(ctx, &awsec2.DisableRouteServerPropagationInput{
		RouteServerId: aws.String(routeServerID),
		RouteTableId:  aws.String("rtb-00000001"),
	})
	if err != nil {
		t.Fatalf("disable route server propagation: %v", err)
	}
	if disablePropagationOut.RouteServerPropagation == nil || aws.ToString(disablePropagationOut.RouteServerPropagation.RouteTableId) != "rtb-00000001" {
		t.Fatalf("expected disabled route table propagation")
	}

	disassociateOut, err := client.DisassociateRouteServer(ctx, &awsec2.DisassociateRouteServerInput{
		RouteServerId: aws.String(routeServerID),
		VpcId:         aws.String("vpc-00000001"),
	})
	if err != nil {
		t.Fatalf("disassociate route server: %v", err)
	}
	if disassociateOut.RouteServerAssociation == nil || aws.ToString(disassociateOut.RouteServerAssociation.RouteServerId) != routeServerID {
		t.Fatalf("expected disassociated route server id")
	}

	if _, err := client.DeleteRouteServerPeer(ctx, &awsec2.DeleteRouteServerPeerInput{RouteServerPeerId: aws.String(peerIDOne)}); err != nil {
		t.Fatalf("delete route server peer one: %v", err)
	}
	if _, err := client.DeleteRouteServerPeer(ctx, &awsec2.DeleteRouteServerPeerInput{RouteServerPeerId: aws.String(peerIDTwo)}); err != nil {
		t.Fatalf("delete route server peer two: %v", err)
	}
	if _, err := client.DeleteRouteServerEndpoint(ctx, &awsec2.DeleteRouteServerEndpointInput{RouteServerEndpointId: aws.String(endpointID)}); err != nil {
		t.Fatalf("delete route server endpoint: %v", err)
	}
	if _, err := client.DeleteRouteServer(ctx, &awsec2.DeleteRouteServerInput{RouteServerId: aws.String(routeServerID)}); err != nil {
		t.Fatalf("delete route server: %v", err)
	}
}

func TestEC2Stage66ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"AssociateRouteServer",
		"DisassociateRouteServer",
		"GetRouteServerAssociations",
		"EnableRouteServerPropagation",
		"DisableRouteServerPropagation",
		"GetRouteServerPropagations",
		"ModifyRouteServer",
		"GetRouteServerRoutingDatabase",
	}

	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, action := range implemented {
		params := map[string]string{}
		switch action {
		case "AssociateRouteServer", "DisassociateRouteServer":
			params["RouteServerId"] = "rsrv-1234"
			params["VpcId"] = "vpc-00000001"
		case "GetRouteServerAssociations":
			params["RouteServerId"] = "rsrv-1234"
		case "EnableRouteServerPropagation", "DisableRouteServerPropagation":
			params["RouteServerId"] = "rsrv-1234"
			params["RouteTableId"] = "rtb-00000001"
		case "GetRouteServerPropagations":
			params["RouteServerId"] = "rsrv-1234"
		case "ModifyRouteServer":
			params["RouteServerId"] = "rsrv-1234"
		case "GetRouteServerRoutingDatabase":
			params["RouteServerId"] = "rsrv-1234"
		}
		resp := ec2Request(t, ts, action, params)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action)
		}
	}
}
