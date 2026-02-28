package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awsec2 "github.com/aws/aws-sdk-go-v2/service/ec2"
	awsec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

func TestEC2Stage40SDKLifecycle(t *testing.T) {
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
		Description: aws.String("stage-40-gateway"),
	})
	if err != nil || createGatewayOut.TransitGateway == nil || createGatewayOut.TransitGateway.TransitGatewayId == nil {
		t.Fatalf("create transit gateway: %v", err)
	}
	transitGatewayID := aws.ToString(createGatewayOut.TransitGateway.TransitGatewayId)

	createVpcAttachmentOut, err := client.CreateTransitGatewayVpcAttachment(ctx, &awsec2.CreateTransitGatewayVpcAttachmentInput{
		TransitGatewayId: aws.String(transitGatewayID),
		VpcId:            aws.String("vpc-00000001"),
		SubnetIds:        []string{"subnet-00000001"},
	})
	if err != nil || createVpcAttachmentOut.TransitGatewayVpcAttachment == nil || createVpcAttachmentOut.TransitGatewayVpcAttachment.TransitGatewayAttachmentId == nil {
		t.Fatalf("create transit gateway vpc attachment: %v", err)
	}
	attachmentID := aws.ToString(createVpcAttachmentOut.TransitGatewayVpcAttachment.TransitGatewayAttachmentId)

	createRouteTableOut, err := client.CreateTransitGatewayRouteTable(ctx, &awsec2.CreateTransitGatewayRouteTableInput{
		TransitGatewayId: aws.String(transitGatewayID),
	})
	if err != nil || createRouteTableOut.TransitGatewayRouteTable == nil || createRouteTableOut.TransitGatewayRouteTable.TransitGatewayRouteTableId == nil {
		t.Fatalf("create transit gateway route table: %v", err)
	}
	routeTableID := aws.ToString(createRouteTableOut.TransitGatewayRouteTable.TransitGatewayRouteTableId)

	createRouteOneOut, err := client.CreateTransitGatewayRoute(ctx, &awsec2.CreateTransitGatewayRouteInput{
		TransitGatewayRouteTableId: aws.String(routeTableID),
		DestinationCidrBlock:       aws.String("10.10.0.0/16"),
		TransitGatewayAttachmentId: aws.String(attachmentID),
	})
	if err != nil || createRouteOneOut.Route == nil || aws.ToString(createRouteOneOut.Route.DestinationCidrBlock) != "10.10.0.0/16" || string(createRouteOneOut.Route.State) != "active" {
		t.Fatalf("create transit gateway route #1: %v", err)
	}

	if _, err := client.CreateTransitGatewayRoute(ctx, &awsec2.CreateTransitGatewayRouteInput{
		TransitGatewayRouteTableId: aws.String(routeTableID),
		DestinationCidrBlock:       aws.String("10.20.0.0/16"),
		TransitGatewayAttachmentId: aws.String(attachmentID),
	}); err != nil {
		t.Fatalf("create transit gateway route #2: %v", err)
	}

	searchPageOut, err := client.SearchTransitGatewayRoutes(ctx, &awsec2.SearchTransitGatewayRoutesInput{
		TransitGatewayRouteTableId: aws.String(routeTableID),
		MaxResults:                 aws.Int32(1),
		Filters: []awsec2types.Filter{
			{Name: aws.String("type"), Values: []string{"static"}},
		},
	})
	if err != nil || len(searchPageOut.Routes) != 1 || searchPageOut.AdditionalRoutesAvailable == nil || !aws.ToBool(searchPageOut.AdditionalRoutesAvailable) {
		t.Fatalf("search transit gateway routes page: %v", err)
	}

	searchExactOut, err := client.SearchTransitGatewayRoutes(ctx, &awsec2.SearchTransitGatewayRoutesInput{
		TransitGatewayRouteTableId: aws.String(routeTableID),
		Filters: []awsec2types.Filter{
			{Name: aws.String("route-search.exact-match"), Values: []string{"10.10.0.0/16"}},
		},
	})
	if err != nil || len(searchExactOut.Routes) != 1 || aws.ToString(searchExactOut.Routes[0].DestinationCidrBlock) != "10.10.0.0/16" || string(searchExactOut.Routes[0].State) != "active" {
		t.Fatalf("search transit gateway routes exact: %v", err)
	}

	replaceRouteOut, err := client.ReplaceTransitGatewayRoute(ctx, &awsec2.ReplaceTransitGatewayRouteInput{
		TransitGatewayRouteTableId: aws.String(routeTableID),
		DestinationCidrBlock:       aws.String("10.10.0.0/16"),
		Blackhole:                  aws.Bool(true),
	})
	if err != nil || replaceRouteOut.Route == nil || string(replaceRouteOut.Route.State) != "blackhole" {
		t.Fatalf("replace transit gateway route: %v", err)
	}

	searchBlackholeOut, err := client.SearchTransitGatewayRoutes(ctx, &awsec2.SearchTransitGatewayRoutesInput{
		TransitGatewayRouteTableId: aws.String(routeTableID),
		Filters: []awsec2types.Filter{
			{Name: aws.String("state"), Values: []string{"blackhole"}},
			{Name: aws.String("route-search.exact-match"), Values: []string{"10.10.0.0/16"}},
		},
	})
	if err != nil || len(searchBlackholeOut.Routes) != 1 || string(searchBlackholeOut.Routes[0].State) != "blackhole" {
		t.Fatalf("search transit gateway routes blackhole: %v", err)
	}

	exportRoutesOut, err := client.ExportTransitGatewayRoutes(ctx, &awsec2.ExportTransitGatewayRoutesInput{
		TransitGatewayRouteTableId: aws.String(routeTableID),
		S3Bucket:                   aws.String("demo-export-bucket"),
		Filters: []awsec2types.Filter{
			{Name: aws.String("transit-gateway-route-destination-cidr-block"), Values: []string{"10.10.0.0/16"}},
		},
	})
	if err != nil || exportRoutesOut.S3Location == nil || !strings.Contains(aws.ToString(exportRoutesOut.S3Location), "demo-export-bucket") {
		t.Fatalf("export transit gateway routes: %v", err)
	}

	deleteRouteOut, err := client.DeleteTransitGatewayRoute(ctx, &awsec2.DeleteTransitGatewayRouteInput{
		TransitGatewayRouteTableId: aws.String(routeTableID),
		DestinationCidrBlock:       aws.String("10.10.0.0/16"),
	})
	if err != nil || deleteRouteOut.Route == nil || string(deleteRouteOut.Route.State) != "deleted" {
		t.Fatalf("delete transit gateway route: %v", err)
	}
}

func TestEC2Stage40ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"CreateTransitGatewayRoute",
		"DeleteTransitGatewayRoute",
		"ReplaceTransitGatewayRoute",
		"SearchTransitGatewayRoutes",
		"ExportTransitGatewayRoutes",
	}

	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, action := range implemented {
		params := map[string]string{}
		switch action {
		case "CreateTransitGatewayRoute":
			params["TransitGatewayRouteTableId"] = "tgw-rtb-00000040"
			params["DestinationCidrBlock"] = "10.40.0.0/16"
			params["Blackhole"] = "true"
		case "DeleteTransitGatewayRoute":
			params["TransitGatewayRouteTableId"] = "tgw-rtb-00000040"
			params["DestinationCidrBlock"] = "10.40.0.0/16"
		case "ReplaceTransitGatewayRoute":
			params["TransitGatewayRouteTableId"] = "tgw-rtb-00000040"
			params["DestinationCidrBlock"] = "10.40.0.0/16"
			params["Blackhole"] = "true"
		case "SearchTransitGatewayRoutes":
			params["TransitGatewayRouteTableId"] = "tgw-rtb-00000040"
			params["Filter.1.Name"] = "state"
			params["Filter.1.Value.1"] = "active"
		case "ExportTransitGatewayRoutes":
			params["TransitGatewayRouteTableId"] = "tgw-rtb-00000040"
			params["S3Bucket"] = "demo-export-bucket"
		}
		resp := ec2Request(t, ts, action, params)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action)
		}
	}
}
