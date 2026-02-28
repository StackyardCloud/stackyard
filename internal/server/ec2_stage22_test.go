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

func TestEC2Stage22SDKLifecycle(t *testing.T) {
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
		IpAddress: aws.String("198.51.100.80"),
		BgpAsn:    aws.Int32(65080),
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

	if _, err := client.AttachVpnGateway(ctx, &awsec2.AttachVpnGatewayInput{
		VpcId:        aws.String("vpc-00000001"),
		VpnGatewayId: aws.String(vpnGatewayID),
	}); err != nil {
		t.Fatalf("attach vpn gateway: %v", err)
	}

	createVpnConnectionOut, err := client.CreateVpnConnection(ctx, &awsec2.CreateVpnConnectionInput{
		CustomerGatewayId: aws.String(customerGatewayID),
		Type:              aws.String("ipsec.1"),
		VpnGatewayId:      aws.String(vpnGatewayID),
	})
	if err != nil || createVpnConnectionOut.VpnConnection == nil || createVpnConnectionOut.VpnConnection.VpnConnectionId == nil {
		t.Fatalf("create vpn connection: %v", err)
	}
	vpnConnectionID := aws.ToString(createVpnConnectionOut.VpnConnection.VpnConnectionId)
	if len(createVpnConnectionOut.VpnConnection.VgwTelemetry) == 0 || createVpnConnectionOut.VpnConnection.VgwTelemetry[0].OutsideIpAddress == nil {
		t.Fatalf("expected vpn telemetry with outside ip address")
	}
	vpnTunnelOutsideIPAddress := aws.ToString(createVpnConnectionOut.VpnConnection.VgwTelemetry[0].OutsideIpAddress)

	if _, err := client.EnableVgwRoutePropagation(ctx, &awsec2.EnableVgwRoutePropagationInput{
		RouteTableId: aws.String("rtb-00000001"),
		GatewayId:    aws.String(vpnGatewayID),
	}); err != nil {
		t.Fatalf("enable vgw route propagation: %v", err)
	}

	getActiveVpnTunnelStatusOut, err := client.GetActiveVpnTunnelStatus(ctx, &awsec2.GetActiveVpnTunnelStatusInput{
		VpnConnectionId:           aws.String(vpnConnectionID),
		VpnTunnelOutsideIpAddress: aws.String(vpnTunnelOutsideIPAddress),
	})
	if err != nil || getActiveVpnTunnelStatusOut.ActiveVpnTunnelStatus == nil || getActiveVpnTunnelStatusOut.ActiveVpnTunnelStatus.IkeVersion == nil {
		t.Fatalf("get active vpn tunnel status: %v", err)
	}

	getVpnConnectionDeviceTypesOut, err := client.GetVpnConnectionDeviceTypes(ctx, &awsec2.GetVpnConnectionDeviceTypesInput{
		MaxResults: aws.Int32(1),
	})
	if err != nil || len(getVpnConnectionDeviceTypesOut.VpnConnectionDeviceTypes) != 1 || getVpnConnectionDeviceTypesOut.NextToken == nil {
		t.Fatalf("get vpn connection device types: %v", err)
	}
	deviceTypeID := aws.ToString(getVpnConnectionDeviceTypesOut.VpnConnectionDeviceTypes[0].VpnConnectionDeviceTypeId)

	getVpnConnectionDeviceSampleConfigurationOut, err := client.GetVpnConnectionDeviceSampleConfiguration(ctx, &awsec2.GetVpnConnectionDeviceSampleConfigurationInput{
		VpnConnectionId:            aws.String(vpnConnectionID),
		VpnConnectionDeviceTypeId:  aws.String(deviceTypeID),
		InternetKeyExchangeVersion: aws.String("ikev2"),
		SampleType:                 aws.String("recommended"),
	})
	if err != nil || getVpnConnectionDeviceSampleConfigurationOut.VpnConnectionDeviceSampleConfiguration == nil {
		t.Fatalf("get vpn connection device sample configuration: %v", err)
	}

	getVpnTunnelReplacementStatusOut, err := client.GetVpnTunnelReplacementStatus(ctx, &awsec2.GetVpnTunnelReplacementStatusInput{
		VpnConnectionId:           aws.String(vpnConnectionID),
		VpnTunnelOutsideIpAddress: aws.String(vpnTunnelOutsideIPAddress),
	})
	if err != nil || aws.ToString(getVpnTunnelReplacementStatusOut.VpnConnectionId) != vpnConnectionID {
		t.Fatalf("get vpn tunnel replacement status: %v", err)
	}

	modifyVpnTunnelCertificateOut, err := client.ModifyVpnTunnelCertificate(ctx, &awsec2.ModifyVpnTunnelCertificateInput{
		VpnConnectionId:           aws.String(vpnConnectionID),
		VpnTunnelOutsideIpAddress: aws.String(vpnTunnelOutsideIPAddress),
	})
	if err != nil || modifyVpnTunnelCertificateOut.VpnConnection == nil {
		t.Fatalf("modify vpn tunnel certificate: %v", err)
	}

	modifyVpnTunnelOptionsOut, err := client.ModifyVpnTunnelOptions(ctx, &awsec2.ModifyVpnTunnelOptionsInput{
		VpnConnectionId:           aws.String(vpnConnectionID),
		VpnTunnelOutsideIpAddress: aws.String(vpnTunnelOutsideIPAddress),
		TunnelOptions: &awsec2types.ModifyVpnTunnelOptionsSpecification{
			PreSharedKey:     aws.String("stackyardpsk"),
			TunnelInsideCidr: aws.String("169.254.22.0/30"),
		},
	})
	if err != nil || modifyVpnTunnelOptionsOut.VpnConnection == nil {
		t.Fatalf("modify vpn tunnel options: %v", err)
	}

	replaceVpnTunnelOut, err := client.ReplaceVpnTunnel(ctx, &awsec2.ReplaceVpnTunnelInput{
		VpnConnectionId:           aws.String(vpnConnectionID),
		VpnTunnelOutsideIpAddress: aws.String(vpnTunnelOutsideIPAddress),
		ApplyPendingMaintenance:   aws.Bool(true),
	})
	if err != nil || replaceVpnTunnelOut.Return == nil || !aws.ToBool(replaceVpnTunnelOut.Return) {
		t.Fatalf("replace vpn tunnel: %v", err)
	}

	if _, err := client.DisableVgwRoutePropagation(ctx, &awsec2.DisableVgwRoutePropagationInput{
		RouteTableId: aws.String("rtb-00000001"),
		GatewayId:    aws.String(vpnGatewayID),
	}); err != nil {
		t.Fatalf("disable vgw route propagation: %v", err)
	}
}

func TestEC2Stage22ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"DisableVgwRoutePropagation",
		"EnableVgwRoutePropagation",
		"GetActiveVpnTunnelStatus",
		"GetVpnConnectionDeviceSampleConfiguration",
		"GetVpnConnectionDeviceTypes",
		"GetVpnTunnelReplacementStatus",
		"ModifyVpnTunnelCertificate",
		"ModifyVpnTunnelOptions",
		"ReplaceVpnTunnel",
	}

	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, action := range implemented {
		params := map[string]string{}
		switch action {
		case "DisableVgwRoutePropagation", "EnableVgwRoutePropagation":
			params["RouteTableId"] = "rtb-00000001"
			params["GatewayId"] = "vgw-00000001"
		case "GetActiveVpnTunnelStatus", "GetVpnTunnelReplacementStatus", "ModifyVpnTunnelCertificate", "ReplaceVpnTunnel":
			params["VpnConnectionId"] = "vpn-00000001"
			params["VpnTunnelOutsideIpAddress"] = "198.51.100.80"
		case "GetVpnConnectionDeviceSampleConfiguration":
			params["VpnConnectionId"] = "vpn-00000001"
			params["VpnConnectionDeviceTypeId"] = "vpn-device-0001"
		case "GetVpnConnectionDeviceTypes":
			params["MaxResults"] = "1"
		case "ModifyVpnTunnelOptions":
			params["VpnConnectionId"] = "vpn-00000001"
			params["VpnTunnelOutsideIpAddress"] = "198.51.100.80"
			params["TunnelOptions.PreSharedKey"] = "stackyardpsk"
		}
		resp := ec2Request(t, ts, action, params)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action)
		}
	}
}
