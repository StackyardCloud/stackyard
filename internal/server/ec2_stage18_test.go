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

func TestEC2Stage18SDKLifecycle(t *testing.T) {
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

	createOut, err := client.CreateVpnGateway(ctx, &awsec2.CreateVpnGatewayInput{
		Type:             awsec2types.GatewayTypeIpsec1,
		AmazonSideAsn:    aws.Int64(65020),
		AvailabilityZone: aws.String("us-east-1a"),
		TagSpecifications: []awsec2types.TagSpecification{
			{
				ResourceType: awsec2types.ResourceTypeVpnGateway,
				Tags:         []awsec2types.Tag{{Key: aws.String("env"), Value: aws.String("stage18")}},
			},
		},
	})
	if err != nil || createOut.VpnGateway == nil || aws.ToString(createOut.VpnGateway.VpnGatewayId) == "" || createOut.VpnGateway.State != awsec2types.VpnStateAvailable {
		t.Fatalf("create vpn gateway: %v", err)
	}
	vpnGatewayID := aws.ToString(createOut.VpnGateway.VpnGatewayId)

	attachOut, err := client.AttachVpnGateway(ctx, &awsec2.AttachVpnGatewayInput{
		VpnGatewayId: aws.String(vpnGatewayID),
		VpcId:        aws.String("vpc-00000001"),
	})
	if err != nil || attachOut.VpcAttachment == nil || aws.ToString(attachOut.VpcAttachment.VpcId) != "vpc-00000001" || attachOut.VpcAttachment.State != awsec2types.AttachmentStatusAttached {
		t.Fatalf("attach vpn gateway: %v", err)
	}

	describeOut, err := client.DescribeVpnGateways(ctx, &awsec2.DescribeVpnGatewaysInput{
		VpnGatewayIds: []string{vpnGatewayID},
		Filters: []awsec2types.Filter{
			{Name: aws.String("attachment.vpc-id"), Values: []string{"vpc-00000001"}},
			{Name: aws.String("attachment.state"), Values: []string{"attached"}},
			{Name: aws.String("state"), Values: []string{"available"}},
			{Name: aws.String("tag:env"), Values: []string{"stage18"}},
		},
	})
	if err != nil || len(describeOut.VpnGateways) != 1 || aws.ToString(describeOut.VpnGateways[0].VpnGatewayId) != vpnGatewayID {
		t.Fatalf("describe vpn gateways: %v", err)
	}

	if _, err := client.DetachVpnGateway(ctx, &awsec2.DetachVpnGatewayInput{
		VpnGatewayId: aws.String(vpnGatewayID),
		VpcId:        aws.String("vpc-00000001"),
	}); err != nil {
		t.Fatalf("detach vpn gateway: %v", err)
	}

	if _, err := client.DeleteVpnGateway(ctx, &awsec2.DeleteVpnGatewayInput{
		VpnGatewayId: aws.String(vpnGatewayID),
	}); err != nil {
		t.Fatalf("delete vpn gateway: %v", err)
	}

	describeAfterDeleteOut, err := client.DescribeVpnGateways(ctx, &awsec2.DescribeVpnGatewaysInput{
		VpnGatewayIds: []string{vpnGatewayID},
		Filters: []awsec2types.Filter{
			{Name: aws.String("state"), Values: []string{"deleted"}},
		},
	})
	if err != nil {
		t.Fatalf("describe vpn gateways after delete: %v", err)
	}
	if len(describeAfterDeleteOut.VpnGateways) != 1 || describeAfterDeleteOut.VpnGateways[0].State != awsec2types.VpnStateDeleted {
		t.Fatalf("expected deleted vpn gateway after delete")
	}
}

func TestEC2Stage18ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"CreateVpnGateway",
		"DescribeVpnGateways",
		"AttachVpnGateway",
		"DetachVpnGateway",
		"DeleteVpnGateway",
	}

	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, action := range implemented {
		params := map[string]string{}
		switch action {
		case "CreateVpnGateway":
			params["Type"] = "ipsec.1"
		case "DescribeVpnGateways":
			params["Filter.1.Name"] = "type"
			params["Filter.1.Value.1"] = "ipsec.1"
		case "AttachVpnGateway", "DetachVpnGateway":
			params["VpnGatewayId"] = "vgw-00000001"
			params["VpcId"] = "vpc-00000001"
		case "DeleteVpnGateway":
			params["VpnGatewayId"] = "vgw-00000001"
		}
		resp := ec2Request(t, ts, action, params)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action)
		}
	}
}
