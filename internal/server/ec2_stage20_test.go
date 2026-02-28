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

func TestEC2Stage20SDKLifecycle(t *testing.T) {
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

	createCustomerGatewayOneOut, err := client.CreateCustomerGateway(ctx, &awsec2.CreateCustomerGatewayInput{
		Type:      awsec2types.GatewayTypeIpsec1,
		IpAddress: aws.String("198.51.100.60"),
		BgpAsn:    aws.Int32(65060),
	})
	if err != nil || createCustomerGatewayOneOut.CustomerGateway == nil || createCustomerGatewayOneOut.CustomerGateway.CustomerGatewayId == nil {
		t.Fatalf("create customer gateway #1: %v", err)
	}
	customerGatewayOneID := aws.ToString(createCustomerGatewayOneOut.CustomerGateway.CustomerGatewayId)

	createCustomerGatewayTwoOut, err := client.CreateCustomerGateway(ctx, &awsec2.CreateCustomerGatewayInput{
		Type:      awsec2types.GatewayTypeIpsec1,
		IpAddress: aws.String("198.51.100.61"),
		BgpAsn:    aws.Int32(65061),
	})
	if err != nil || createCustomerGatewayTwoOut.CustomerGateway == nil || createCustomerGatewayTwoOut.CustomerGateway.CustomerGatewayId == nil {
		t.Fatalf("create customer gateway #2: %v", err)
	}
	customerGatewayTwoID := aws.ToString(createCustomerGatewayTwoOut.CustomerGateway.CustomerGatewayId)

	createVpnGatewayOut, err := client.CreateVpnGateway(ctx, &awsec2.CreateVpnGatewayInput{
		Type: awsec2types.GatewayTypeIpsec1,
	})
	if err != nil || createVpnGatewayOut.VpnGateway == nil || createVpnGatewayOut.VpnGateway.VpnGatewayId == nil {
		t.Fatalf("create vpn gateway: %v", err)
	}
	vpnGatewayID := aws.ToString(createVpnGatewayOut.VpnGateway.VpnGatewayId)

	createVpnConnectionOut, err := client.CreateVpnConnection(ctx, &awsec2.CreateVpnConnectionInput{
		CustomerGatewayId: aws.String(customerGatewayOneID),
		Type:              aws.String("ipsec.1"),
		VpnGatewayId:      aws.String(vpnGatewayID),
	})
	if err != nil || createVpnConnectionOut.VpnConnection == nil || createVpnConnectionOut.VpnConnection.VpnConnectionId == nil {
		t.Fatalf("create vpn connection: %v", err)
	}
	vpnConnectionID := aws.ToString(createVpnConnectionOut.VpnConnection.VpnConnectionId)

	modifyConnectionOut, err := client.ModifyVpnConnection(ctx, &awsec2.ModifyVpnConnectionInput{
		VpnConnectionId:   aws.String(vpnConnectionID),
		CustomerGatewayId: aws.String(customerGatewayTwoID),
	})
	if err != nil || modifyConnectionOut.VpnConnection == nil || aws.ToString(modifyConnectionOut.VpnConnection.CustomerGatewayId) != customerGatewayTwoID {
		t.Fatalf("modify vpn connection: %v", err)
	}

	modifyConnectionOptionsOut, err := client.ModifyVpnConnectionOptions(ctx, &awsec2.ModifyVpnConnectionOptionsInput{
		VpnConnectionId:       aws.String(vpnConnectionID),
		LocalIpv4NetworkCidr:  aws.String("10.252.0.0/16"),
		RemoteIpv4NetworkCidr: aws.String("10.253.0.0/16"),
	})
	if err != nil || modifyConnectionOptionsOut.VpnConnection == nil || modifyConnectionOptionsOut.VpnConnection.Options == nil || aws.ToString(modifyConnectionOptionsOut.VpnConnection.Options.LocalIpv4NetworkCidr) != "10.252.0.0/16" || aws.ToString(modifyConnectionOptionsOut.VpnConnection.Options.RemoteIpv4NetworkCidr) != "10.253.0.0/16" {
		t.Fatalf("modify vpn connection options: %v", err)
	}
}

func TestEC2Stage20ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"ModifyVpnConnection",
		"ModifyVpnConnectionOptions",
	}

	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, action := range implemented {
		params := map[string]string{}
		switch action {
		case "ModifyVpnConnection":
			params["VpnConnectionId"] = "vpn-00000001"
			params["CustomerGatewayId"] = "cgw-00000001"
		case "ModifyVpnConnectionOptions":
			params["VpnConnectionId"] = "vpn-00000001"
			params["LocalIpv4NetworkCidr"] = "10.252.0.0/16"
		}
		resp := ec2Request(t, ts, action, params)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action)
		}
	}
}
