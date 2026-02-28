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
)

func TestEC2Stage28SDKLifecycle(t *testing.T) {
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

	createIfaceOut, err := client.CreateNetworkInterface(ctx, &awsec2.CreateNetworkInterfaceInput{
		SubnetId: aws.String("subnet-00000001"),
	})
	if err != nil || createIfaceOut.NetworkInterface == nil || createIfaceOut.NetworkInterface.NetworkInterfaceId == nil {
		t.Fatalf("create network interface: %v", err)
	}
	ifaceID := aws.ToString(createIfaceOut.NetworkInterface.NetworkInterfaceId)

	assignPrivateOut, err := client.AssignPrivateIpAddresses(ctx, &awsec2.AssignPrivateIpAddressesInput{
		NetworkInterfaceId: aws.String(ifaceID),
		PrivateIpAddresses: []string{"10.0.0.55"},
		Ipv4Prefixes:       []string{"10.0.16.0/28"},
	})
	if err != nil {
		t.Fatalf("assign private ip addresses: %v", err)
	}
	if aws.ToString(assignPrivateOut.NetworkInterfaceId) != ifaceID {
		t.Fatalf("expected network interface id %s, got %q", ifaceID, aws.ToString(assignPrivateOut.NetworkInterfaceId))
	}
	if len(assignPrivateOut.AssignedPrivateIpAddresses) != 1 || aws.ToString(assignPrivateOut.AssignedPrivateIpAddresses[0].PrivateIpAddress) != "10.0.0.55" {
		t.Fatalf("unexpected assigned private ip addresses: %+v", assignPrivateOut.AssignedPrivateIpAddresses)
	}
	if len(assignPrivateOut.AssignedIpv4Prefixes) != 1 || aws.ToString(assignPrivateOut.AssignedIpv4Prefixes[0].Ipv4Prefix) != "10.0.16.0/28" {
		t.Fatalf("unexpected assigned ipv4 prefixes: %+v", assignPrivateOut.AssignedIpv4Prefixes)
	}

	assignPrivateCountOut, err := client.AssignPrivateIpAddresses(ctx, &awsec2.AssignPrivateIpAddressesInput{
		NetworkInterfaceId:             aws.String(ifaceID),
		SecondaryPrivateIpAddressCount: aws.Int32(1),
	})
	if err != nil {
		t.Fatalf("assign private ip addresses by count: %v", err)
	}
	if len(assignPrivateCountOut.AssignedPrivateIpAddresses) != 1 {
		t.Fatalf("expected one auto-assigned private ip, got %d", len(assignPrivateCountOut.AssignedPrivateIpAddresses))
	}
	autoAssignedPrivateIP := aws.ToString(assignPrivateCountOut.AssignedPrivateIpAddresses[0].PrivateIpAddress)
	if autoAssignedPrivateIP == "" {
		t.Fatalf("expected auto-assigned private ip to be non-empty")
	}

	if _, err := client.UnassignPrivateIpAddresses(ctx, &awsec2.UnassignPrivateIpAddressesInput{
		NetworkInterfaceId: aws.String(ifaceID),
		PrivateIpAddresses: []string{"10.0.0.55", autoAssignedPrivateIP},
		Ipv4Prefixes:       []string{"10.0.16.0/28"},
	}); err != nil {
		t.Fatalf("unassign private ip addresses: %v", err)
	}

	assignIPv6Out, err := client.AssignIpv6Addresses(ctx, &awsec2.AssignIpv6AddressesInput{
		NetworkInterfaceId: aws.String(ifaceID),
		Ipv6Addresses:      []string{"2001:db8::55"},
		Ipv6Prefixes:       []string{"2001:db8:10::/80"},
	})
	if err != nil {
		t.Fatalf("assign ipv6 addresses: %v", err)
	}
	if aws.ToString(assignIPv6Out.NetworkInterfaceId) != ifaceID {
		t.Fatalf("expected network interface id %s, got %q", ifaceID, aws.ToString(assignIPv6Out.NetworkInterfaceId))
	}
	if len(assignIPv6Out.AssignedIpv6Addresses) != 1 || assignIPv6Out.AssignedIpv6Addresses[0] != "2001:db8::55" {
		t.Fatalf("unexpected assigned ipv6 addresses: %+v", assignIPv6Out.AssignedIpv6Addresses)
	}
	if len(assignIPv6Out.AssignedIpv6Prefixes) != 1 || assignIPv6Out.AssignedIpv6Prefixes[0] != "2001:db8:10::/80" {
		t.Fatalf("unexpected assigned ipv6 prefixes: %+v", assignIPv6Out.AssignedIpv6Prefixes)
	}

	assignIPv6CountOut, err := client.AssignIpv6Addresses(ctx, &awsec2.AssignIpv6AddressesInput{
		NetworkInterfaceId: aws.String(ifaceID),
		Ipv6AddressCount:   aws.Int32(1),
	})
	if err != nil {
		t.Fatalf("assign ipv6 addresses by count: %v", err)
	}
	if len(assignIPv6CountOut.AssignedIpv6Addresses) != 1 {
		t.Fatalf("expected one auto-assigned ipv6 address, got %d", len(assignIPv6CountOut.AssignedIpv6Addresses))
	}
	autoAssignedIPv6 := assignIPv6CountOut.AssignedIpv6Addresses[0]
	if autoAssignedIPv6 == "" {
		t.Fatalf("expected auto-assigned ipv6 to be non-empty")
	}

	unassignIPv6Out, err := client.UnassignIpv6Addresses(ctx, &awsec2.UnassignIpv6AddressesInput{
		NetworkInterfaceId: aws.String(ifaceID),
		Ipv6Addresses:      []string{"2001:db8::55", autoAssignedIPv6},
		Ipv6Prefixes:       []string{"2001:db8:10::/80"},
	})
	if err != nil {
		t.Fatalf("unassign ipv6 addresses: %v", err)
	}
	if aws.ToString(unassignIPv6Out.NetworkInterfaceId) != ifaceID {
		t.Fatalf("expected network interface id %s in unassign response, got %q", ifaceID, aws.ToString(unassignIPv6Out.NetworkInterfaceId))
	}
	if len(unassignIPv6Out.UnassignedIpv6Addresses) != 2 {
		t.Fatalf("expected two unassigned ipv6 addresses, got %d", len(unassignIPv6Out.UnassignedIpv6Addresses))
	}
	if len(unassignIPv6Out.UnassignedIpv6Prefixes) != 1 || unassignIPv6Out.UnassignedIpv6Prefixes[0] != "2001:db8:10::/80" {
		t.Fatalf("unexpected unassigned ipv6 prefixes: %+v", unassignIPv6Out.UnassignedIpv6Prefixes)
	}
}

func TestEC2Stage28ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"AssignIpv6Addresses",
		"AssignPrivateIpAddresses",
		"UnassignIpv6Addresses",
		"UnassignPrivateIpAddresses",
	}

	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, action := range implemented {
		params := map[string]string{"NetworkInterfaceId": "eni-00000001"}
		switch action {
		case "AssignPrivateIpAddresses":
			params["PrivateIpAddress.1"] = "10.0.0.55"
		case "UnassignPrivateIpAddresses":
			params["PrivateIpAddress.1"] = "10.0.0.55"
		case "AssignIpv6Addresses":
			params["Ipv6Addresses.1"] = "2001:db8::55"
		case "UnassignIpv6Addresses":
			params["Ipv6Addresses.1"] = "2001:db8::55"
		}
		resp := ec2Request(t, ts, action, params)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action)
		}
	}
}
