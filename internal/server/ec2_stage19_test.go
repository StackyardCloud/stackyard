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

func TestEC2Stage19SDKLifecycle(t *testing.T) {
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

	createCustomerGatewayOut, err := client.CreateCustomerGateway(ctx, &awsec2.CreateCustomerGatewayInput{
		Type:      awsec2types.GatewayTypeIpsec1,
		IpAddress: aws.String("198.51.100.40"),
		BgpAsn:    aws.Int32(65040),
	})
	if err != nil || createCustomerGatewayOut.CustomerGateway == nil || createCustomerGatewayOut.CustomerGateway.CustomerGatewayId == nil {
		t.Fatalf("create customer gateway: %v", err)
	}
	customerGatewayID := aws.ToString(createCustomerGatewayOut.CustomerGateway.CustomerGatewayId)

	createVpnGatewayOut, err := client.CreateVpnGateway(ctx, &awsec2.CreateVpnGatewayInput{
		Type: awsec2types.GatewayTypeIpsec1,
	})
	if err != nil || createVpnGatewayOut.VpnGateway == nil || createVpnGatewayOut.VpnGateway.VpnGatewayId == nil {
		t.Fatalf("create vpn gateway: %v", err)
	}
	vpnGatewayID := aws.ToString(createVpnGatewayOut.VpnGateway.VpnGatewayId)

	createVpnConnectionOut, err := client.CreateVpnConnection(ctx, &awsec2.CreateVpnConnectionInput{
		CustomerGatewayId: aws.String(customerGatewayID),
		Type:              aws.String("ipsec.1"),
		VpnGatewayId:      aws.String(vpnGatewayID),
		Options: &awsec2types.VpnConnectionOptionsSpecification{
			StaticRoutesOnly: aws.Bool(true),
		},
		TagSpecifications: []awsec2types.TagSpecification{
			{
				ResourceType: awsec2types.ResourceTypeVpnConnection,
				Tags:         []awsec2types.Tag{{Key: aws.String("env"), Value: aws.String("stage19")}},
			},
		},
	})
	if err != nil || createVpnConnectionOut.VpnConnection == nil || createVpnConnectionOut.VpnConnection.VpnConnectionId == nil || createVpnConnectionOut.VpnConnection.Options == nil || createVpnConnectionOut.VpnConnection.Options.StaticRoutesOnly == nil || !aws.ToBool(createVpnConnectionOut.VpnConnection.Options.StaticRoutesOnly) {
		t.Fatalf("create vpn connection: %v", err)
	}
	vpnConnectionID := aws.ToString(createVpnConnectionOut.VpnConnection.VpnConnectionId)

	if _, err := client.CreateVpnConnectionRoute(ctx, &awsec2.CreateVpnConnectionRouteInput{
		VpnConnectionId:      aws.String(vpnConnectionID),
		DestinationCidrBlock: aws.String("10.210.0.0/16"),
	}); err != nil {
		t.Fatalf("create vpn connection route: %v", err)
	}

	describeOut, err := client.DescribeVpnConnections(ctx, &awsec2.DescribeVpnConnectionsInput{
		VpnConnectionIds: []string{vpnConnectionID},
		Filters: []awsec2types.Filter{
			{Name: aws.String("customer-gateway-id"), Values: []string{customerGatewayID}},
			{Name: aws.String("vpn-gateway-id"), Values: []string{vpnGatewayID}},
			{Name: aws.String("option.static-routes-only"), Values: []string{"true"}},
			{Name: aws.String("route.destination-cidr-block"), Values: []string{"10.210.0.0/16"}},
			{Name: aws.String("tag:env"), Values: []string{"stage19"}},
		},
	})
	if err != nil || len(describeOut.VpnConnections) != 1 || aws.ToString(describeOut.VpnConnections[0].VpnConnectionId) != vpnConnectionID {
		t.Fatalf("describe vpn connections: %v", err)
	}
	if len(describeOut.VpnConnections[0].Routes) != 1 || aws.ToString(describeOut.VpnConnections[0].Routes[0].DestinationCidrBlock) != "10.210.0.0/16" {
		t.Fatalf("expected one static route in describe output")
	}

	if _, err := client.DeleteVpnConnectionRoute(ctx, &awsec2.DeleteVpnConnectionRouteInput{
		VpnConnectionId:      aws.String(vpnConnectionID),
		DestinationCidrBlock: aws.String("10.210.0.0/16"),
	}); err != nil {
		t.Fatalf("delete vpn connection route: %v", err)
	}

	if _, err := client.DeleteVpnConnection(ctx, &awsec2.DeleteVpnConnectionInput{
		VpnConnectionId: aws.String(vpnConnectionID),
	}); err != nil {
		t.Fatalf("delete vpn connection: %v", err)
	}

	describeAfterDeleteOut, err := client.DescribeVpnConnections(ctx, &awsec2.DescribeVpnConnectionsInput{
		VpnConnectionIds: []string{vpnConnectionID},
		Filters: []awsec2types.Filter{
			{Name: aws.String("state"), Values: []string{"deleted"}},
		},
	})
	if err != nil {
		t.Fatalf("describe vpn connections after delete: %v", err)
	}
	if len(describeAfterDeleteOut.VpnConnections) != 1 || describeAfterDeleteOut.VpnConnections[0].State != awsec2types.VpnStateDeleted {
		t.Fatalf("expected deleted vpn connection after delete")
	}
}

func TestEC2Stage19ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"CreateVpnConnection",
		"DescribeVpnConnections",
		"CreateVpnConnectionRoute",
		"DeleteVpnConnectionRoute",
		"DeleteVpnConnection",
	}

	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, action := range implemented {
		params := map[string]string{}
		switch action {
		case "CreateVpnConnection":
			params["CustomerGatewayId"] = "cgw-00000001"
			params["Type"] = "ipsec.1"
			params["VpnGatewayId"] = "vgw-00000001"
		case "DescribeVpnConnections":
			params["Filter.1.Name"] = "type"
			params["Filter.1.Value.1"] = "ipsec.1"
		case "CreateVpnConnectionRoute", "DeleteVpnConnectionRoute":
			params["VpnConnectionId"] = "vpn-00000001"
			params["DestinationCidrBlock"] = "10.210.0.0/16"
		case "DeleteVpnConnection":
			params["VpnConnectionId"] = "vpn-00000001"
		}
		resp := ec2Request(t, ts, action, params)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action)
		}
	}
}
