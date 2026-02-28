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

func TestEC2Stage35SDKLifecycle(t *testing.T) {
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

	createRouteOneOut, err := client.CreateTransitGatewayRouteTable(ctx, &awsec2.CreateTransitGatewayRouteTableInput{
		TransitGatewayId: aws.String("tgw-00000035"),
	})
	if err != nil || createRouteOneOut.TransitGatewayRouteTable == nil || createRouteOneOut.TransitGatewayRouteTable.TransitGatewayRouteTableId == nil {
		t.Fatalf("create transit gateway route table #1: %v", err)
	}
	routeTableOneID := aws.ToString(createRouteOneOut.TransitGatewayRouteTable.TransitGatewayRouteTableId)

	createRouteTwoOut, err := client.CreateTransitGatewayRouteTable(ctx, &awsec2.CreateTransitGatewayRouteTableInput{
		TransitGatewayId: aws.String("tgw-00000035"),
	})
	if err != nil || createRouteTwoOut.TransitGatewayRouteTable == nil || createRouteTwoOut.TransitGatewayRouteTable.TransitGatewayRouteTableId == nil {
		t.Fatalf("create transit gateway route table #2: %v", err)
	}
	routeTableTwoID := aws.ToString(createRouteTwoOut.TransitGatewayRouteTable.TransitGatewayRouteTableId)

	enablePropagationOut, err := client.EnableTransitGatewayRouteTablePropagation(ctx, &awsec2.EnableTransitGatewayRouteTablePropagationInput{
		TransitGatewayRouteTableId: aws.String(routeTableOneID),
		TransitGatewayAttachmentId: aws.String("tgw-attach-00000035"),
	})
	if err != nil || enablePropagationOut.Propagation == nil || enablePropagationOut.Propagation.State != awsec2types.TransitGatewayPropagationStateEnabled {
		t.Fatalf("enable transit gateway route table propagation #1: %v", err)
	}
	if _, err := client.EnableTransitGatewayRouteTablePropagation(ctx, &awsec2.EnableTransitGatewayRouteTablePropagationInput{
		TransitGatewayRouteTableId: aws.String(routeTableTwoID),
		TransitGatewayAttachmentId: aws.String("tgw-attach-00000035"),
	}); err != nil {
		t.Fatalf("enable transit gateway route table propagation #2: %v", err)
	}

	getAttachmentPageOneOut, err := client.GetTransitGatewayAttachmentPropagations(ctx, &awsec2.GetTransitGatewayAttachmentPropagationsInput{
		TransitGatewayAttachmentId: aws.String("tgw-attach-00000035"),
		MaxResults:                 aws.Int32(1),
	})
	if err != nil || len(getAttachmentPageOneOut.TransitGatewayAttachmentPropagations) != 1 || getAttachmentPageOneOut.NextToken == nil {
		t.Fatalf("get transit gateway attachment propagations page 1: %v", err)
	}
	getAttachmentPageTwoOut, err := client.GetTransitGatewayAttachmentPropagations(ctx, &awsec2.GetTransitGatewayAttachmentPropagationsInput{
		TransitGatewayAttachmentId: aws.String("tgw-attach-00000035"),
		MaxResults:                 aws.Int32(1),
		NextToken:                  getAttachmentPageOneOut.NextToken,
	})
	if err != nil || len(getAttachmentPageTwoOut.TransitGatewayAttachmentPropagations) != 1 {
		t.Fatalf("get transit gateway attachment propagations page 2: %v", err)
	}

	getRouteTablePropagationsOut, err := client.GetTransitGatewayRouteTablePropagations(ctx, &awsec2.GetTransitGatewayRouteTablePropagationsInput{
		TransitGatewayRouteTableId: aws.String(routeTableOneID),
		Filters: []awsec2types.Filter{
			{Name: aws.String("resource-id"), Values: []string{"tgw-attach-00000035"}},
			{Name: aws.String("resource-type"), Values: []string{"vpc"}},
		},
	})
	if err != nil || len(getRouteTablePropagationsOut.TransitGatewayRouteTablePropagations) != 1 || getRouteTablePropagationsOut.TransitGatewayRouteTablePropagations[0].State != awsec2types.TransitGatewayPropagationStateEnabled {
		t.Fatalf("get transit gateway route table propagations: %v", err)
	}

	disablePropagationOut, err := client.DisableTransitGatewayRouteTablePropagation(ctx, &awsec2.DisableTransitGatewayRouteTablePropagationInput{
		TransitGatewayRouteTableId: aws.String(routeTableOneID),
		TransitGatewayAttachmentId: aws.String("tgw-attach-00000035"),
	})
	if err != nil || disablePropagationOut.Propagation == nil || disablePropagationOut.Propagation.State != awsec2types.TransitGatewayPropagationStateDisabled {
		t.Fatalf("disable transit gateway route table propagation: %v", err)
	}
}

func TestEC2Stage35ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"EnableTransitGatewayRouteTablePropagation",
		"DisableTransitGatewayRouteTablePropagation",
		"GetTransitGatewayAttachmentPropagations",
		"GetTransitGatewayRouteTablePropagations",
	}

	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, action := range implemented {
		params := map[string]string{}
		switch action {
		case "EnableTransitGatewayRouteTablePropagation", "DisableTransitGatewayRouteTablePropagation":
			params["TransitGatewayRouteTableId"] = "tgw-rtb-00000035"
			params["TransitGatewayAttachmentId"] = "tgw-attach-00000035"
		case "GetTransitGatewayAttachmentPropagations":
			params["TransitGatewayAttachmentId"] = "tgw-attach-00000035"
		case "GetTransitGatewayRouteTablePropagations":
			params["TransitGatewayRouteTableId"] = "tgw-rtb-00000035"
		}
		resp := ec2Request(t, ts, action, params)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action)
		}
	}
}
