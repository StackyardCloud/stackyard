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

func TestEC2Stage129SDKLifecycle(t *testing.T) {
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

	createIpamOut, err := client.CreateIpam(ctx, &awsec2.CreateIpamInput{
		Description: aws.String("stage129-ipam"),
		OperatingRegions: []awsec2types.AddIpamOperatingRegion{
			{RegionName: aws.String("us-east-1")},
		},
	})
	if err != nil || createIpamOut.Ipam == nil || createIpamOut.Ipam.IpamId == nil {
		t.Fatalf("create ipam: %v", err)
	}
	ipamID := aws.ToString(createIpamOut.Ipam.IpamId)

	modifyIpamOut, err := client.ModifyIpam(ctx, &awsec2.ModifyIpamInput{
		IpamId:         aws.String(ipamID),
		Description:    aws.String("stage129-ipam-updated"),
		MeteredAccount: awsec2types.IpamMeteredAccountIpamOwner,
		Tier:           awsec2types.IpamTierFree,
		AddOperatingRegions: []awsec2types.AddIpamOperatingRegion{
			{RegionName: aws.String("us-west-2")},
		},
	})
	if err != nil {
		t.Fatalf("modify ipam: %v", err)
	}
	if modifyIpamOut.Ipam == nil || aws.ToString(modifyIpamOut.Ipam.Description) != "stage129-ipam-updated" {
		t.Fatalf("unexpected modify ipam output: %#v", modifyIpamOut.Ipam)
	}

	createIpamScopeOut, err := client.CreateIpamScope(ctx, &awsec2.CreateIpamScopeInput{IpamId: aws.String(ipamID)})
	if err != nil || createIpamScopeOut.IpamScope == nil || createIpamScopeOut.IpamScope.IpamScopeId == nil {
		t.Fatalf("create ipam scope: %v", err)
	}
	ipamScopeID := aws.ToString(createIpamScopeOut.IpamScope.IpamScopeId)

	modifyIpamScopeOut, err := client.ModifyIpamScope(ctx, &awsec2.ModifyIpamScopeInput{
		IpamScopeId: aws.String(ipamScopeID),
		Description: aws.String("stage129-scope"),
	})
	if err != nil {
		t.Fatalf("modify ipam scope: %v", err)
	}
	if modifyIpamScopeOut.IpamScope == nil || aws.ToString(modifyIpamScopeOut.IpamScope.Description) != "stage129-scope" {
		t.Fatalf("unexpected modify ipam scope output: %#v", modifyIpamScopeOut.IpamScope)
	}

	createIpamPoolOut, err := client.CreateIpamPool(ctx, &awsec2.CreateIpamPoolInput{
		AddressFamily: awsec2types.AddressFamilyIpv4,
		IpamScopeId:   aws.String(ipamScopeID),
	})
	if err != nil || createIpamPoolOut.IpamPool == nil || createIpamPoolOut.IpamPool.IpamPoolId == nil {
		t.Fatalf("create ipam pool: %v", err)
	}
	ipamPoolID := aws.ToString(createIpamPoolOut.IpamPool.IpamPoolId)

	modifyIpamPoolOut, err := client.ModifyIpamPool(ctx, &awsec2.ModifyIpamPoolInput{
		IpamPoolId:  aws.String(ipamPoolID),
		Description: aws.String("stage129-pool"),
		AutoImport:  aws.Bool(true),
	})
	if err != nil {
		t.Fatalf("modify ipam pool: %v", err)
	}
	if modifyIpamPoolOut.IpamPool == nil || aws.ToString(modifyIpamPoolOut.IpamPool.Description) != "stage129-pool" {
		t.Fatalf("unexpected modify ipam pool output: %#v", modifyIpamPoolOut.IpamPool)
	}

	modifyIpamResourceCidrOut, err := client.ModifyIpamResourceCidr(ctx, &awsec2.ModifyIpamResourceCidrInput{
		CurrentIpamScopeId: aws.String(ipamScopeID),
		Monitored:          aws.Bool(true),
		ResourceCidr:       aws.String("10.0.0.0/24"),
		ResourceId:         aws.String("subnet-00000001"),
		ResourceRegion:     aws.String("us-east-1"),
	})
	if err != nil {
		t.Fatalf("modify ipam resource cidr: %v", err)
	}
	if modifyIpamResourceCidrOut.IpamResourceCidr == nil || aws.ToString(modifyIpamResourceCidrOut.IpamResourceCidr.ResourceId) != "subnet-00000001" {
		t.Fatalf("unexpected modify ipam resource cidr output: %#v", modifyIpamResourceCidrOut.IpamResourceCidr)
	}

	createIpamResourceDiscoveryOut, err := client.CreateIpamResourceDiscovery(ctx, &awsec2.CreateIpamResourceDiscoveryInput{
		Description: aws.String("stage129-discovery"),
		OperatingRegions: []awsec2types.AddIpamOperatingRegion{
			{RegionName: aws.String("us-east-1")},
		},
	})
	if err != nil || createIpamResourceDiscoveryOut.IpamResourceDiscovery == nil || createIpamResourceDiscoveryOut.IpamResourceDiscovery.IpamResourceDiscoveryId == nil {
		t.Fatalf("create ipam resource discovery: %v", err)
	}
	ipamResourceDiscoveryID := aws.ToString(createIpamResourceDiscoveryOut.IpamResourceDiscovery.IpamResourceDiscoveryId)

	modifyIpamResourceDiscoveryOut, err := client.ModifyIpamResourceDiscovery(ctx, &awsec2.ModifyIpamResourceDiscoveryInput{
		IpamResourceDiscoveryId: aws.String(ipamResourceDiscoveryID),
		Description:             aws.String("stage129-discovery-updated"),
		AddOperatingRegions: []awsec2types.AddIpamOperatingRegion{
			{RegionName: aws.String("us-west-2")},
		},
	})
	if err != nil {
		t.Fatalf("modify ipam resource discovery: %v", err)
	}
	if modifyIpamResourceDiscoveryOut.IpamResourceDiscovery == nil || aws.ToString(modifyIpamResourceDiscoveryOut.IpamResourceDiscovery.Description) != "stage129-discovery-updated" {
		t.Fatalf("unexpected modify ipam resource discovery output: %#v", modifyIpamResourceDiscoveryOut.IpamResourceDiscovery)
	}

	createLaunchTemplateOut, err := client.CreateLaunchTemplate(ctx, &awsec2.CreateLaunchTemplateInput{
		LaunchTemplateName: aws.String("stage129-template"),
		LaunchTemplateData: &awsec2types.RequestLaunchTemplateData{
			ImageId: aws.String("ami-00000000000000129"),
		},
	})
	if err != nil || createLaunchTemplateOut.LaunchTemplate == nil || createLaunchTemplateOut.LaunchTemplate.LaunchTemplateId == nil {
		t.Fatalf("create launch template: %v", err)
	}
	launchTemplateID := aws.ToString(createLaunchTemplateOut.LaunchTemplate.LaunchTemplateId)

	if _, err := client.CreateLaunchTemplateVersion(ctx, &awsec2.CreateLaunchTemplateVersionInput{
		LaunchTemplateId: aws.String(launchTemplateID),
		LaunchTemplateData: &awsec2types.RequestLaunchTemplateData{
			ImageId: aws.String("ami-00000000000000129"),
		},
	}); err != nil {
		t.Fatalf("create launch template version: %v", err)
	}

	modifyLaunchTemplateOut, err := client.ModifyLaunchTemplate(ctx, &awsec2.ModifyLaunchTemplateInput{
		LaunchTemplateId: aws.String(launchTemplateID),
		DefaultVersion:   aws.String("2"),
	})
	if err != nil {
		t.Fatalf("modify launch template: %v", err)
	}
	if modifyLaunchTemplateOut.LaunchTemplate == nil || aws.ToInt64(modifyLaunchTemplateOut.LaunchTemplate.DefaultVersionNumber) != 2 {
		t.Fatalf("unexpected modify launch template output: %#v", modifyLaunchTemplateOut.LaunchTemplate)
	}

	createLocalGatewayRouteTableOut, err := client.CreateLocalGatewayRouteTable(ctx, &awsec2.CreateLocalGatewayRouteTableInput{
		LocalGatewayId: aws.String("lgw-00000000000000129"),
	})
	if err != nil || createLocalGatewayRouteTableOut.LocalGatewayRouteTable == nil || createLocalGatewayRouteTableOut.LocalGatewayRouteTable.LocalGatewayRouteTableId == nil {
		t.Fatalf("create local gateway route table: %v", err)
	}
	localGatewayRouteTableID := aws.ToString(createLocalGatewayRouteTableOut.LocalGatewayRouteTable.LocalGatewayRouteTableId)

	if _, err := client.CreateLocalGatewayRoute(ctx, &awsec2.CreateLocalGatewayRouteInput{
		LocalGatewayRouteTableId: aws.String(localGatewayRouteTableID),
		DestinationCidrBlock:     aws.String("10.129.0.0/16"),
	}); err != nil {
		t.Fatalf("create local gateway route: %v", err)
	}

	modifyLocalGatewayRouteOut, err := client.ModifyLocalGatewayRoute(ctx, &awsec2.ModifyLocalGatewayRouteInput{
		LocalGatewayRouteTableId:            aws.String(localGatewayRouteTableID),
		DestinationCidrBlock:                aws.String("10.129.0.0/16"),
		LocalGatewayVirtualInterfaceGroupId: aws.String("lgw-vifgrp-00000000000000129"),
	})
	if err != nil {
		t.Fatalf("modify local gateway route: %v", err)
	}
	if modifyLocalGatewayRouteOut.Route == nil || aws.ToString(modifyLocalGatewayRouteOut.Route.LocalGatewayRouteTableId) != localGatewayRouteTableID {
		t.Fatalf("unexpected modify local gateway route output: %#v", modifyLocalGatewayRouteOut.Route)
	}

	createManagedPrefixListOut, err := client.CreateManagedPrefixList(ctx, &awsec2.CreateManagedPrefixListInput{
		AddressFamily:  aws.String("ipv4"),
		MaxEntries:     aws.Int32(10),
		PrefixListName: aws.String("stage129-prefix-list"),
	})
	if err != nil || createManagedPrefixListOut.PrefixList == nil || createManagedPrefixListOut.PrefixList.PrefixListId == nil {
		t.Fatalf("create managed prefix list: %v", err)
	}
	prefixListID := aws.ToString(createManagedPrefixListOut.PrefixList.PrefixListId)

	modifyManagedPrefixListOut, err := client.ModifyManagedPrefixList(ctx, &awsec2.ModifyManagedPrefixListInput{
		PrefixListId:   aws.String(prefixListID),
		CurrentVersion: aws.Int64(1),
		AddEntries: []awsec2types.AddPrefixListEntry{
			{Cidr: aws.String("10.129.1.0/24"), Description: aws.String("stage129")},
		},
	})
	if err != nil {
		t.Fatalf("modify managed prefix list: %v", err)
	}
	if modifyManagedPrefixListOut.PrefixList == nil || aws.ToInt64(modifyManagedPrefixListOut.PrefixList.Version) != 2 {
		t.Fatalf("unexpected modify managed prefix list output: %#v", modifyManagedPrefixListOut.PrefixList)
	}

	runInstancesOut, err := client.RunInstances(ctx, &awsec2.RunInstancesInput{
		ImageId:  aws.String("ami-00000000000000129"),
		MinCount: aws.Int32(1),
		MaxCount: aws.Int32(1),
		SubnetId: aws.String("subnet-00000001"),
	})
	if err != nil || len(runInstancesOut.Instances) != 1 || runInstancesOut.Instances[0].InstanceId == nil {
		t.Fatalf("run instances: %v", err)
	}
	instanceID := aws.ToString(runInstancesOut.Instances[0].InstanceId)

	modifyPrivateDnsNameOptionsOut, err := client.ModifyPrivateDnsNameOptions(ctx, &awsec2.ModifyPrivateDnsNameOptionsInput{
		InstanceId:                      aws.String(instanceID),
		EnableResourceNameDnsARecord:    aws.Bool(true),
		EnableResourceNameDnsAAAARecord: aws.Bool(false),
		PrivateDnsHostnameType:          awsec2types.HostnameTypeResourceName,
	})
	if err != nil {
		t.Fatalf("modify private dns name options: %v", err)
	}
	if modifyPrivateDnsNameOptionsOut.Return == nil || !aws.ToBool(modifyPrivateDnsNameOptionsOut.Return) {
		t.Fatalf("expected modify private dns name options return true")
	}

	createNetworkInterfaceOut, err := client.CreateNetworkInterface(ctx, &awsec2.CreateNetworkInterfaceInput{
		SubnetId: aws.String("subnet-00000001"),
	})
	if err != nil || createNetworkInterfaceOut.NetworkInterface == nil || createNetworkInterfaceOut.NetworkInterface.NetworkInterfaceId == nil {
		t.Fatalf("create network interface: %v", err)
	}
	networkInterfaceID := aws.ToString(createNetworkInterfaceOut.NetworkInterface.NetworkInterfaceId)

	modifyPublicIpDnsNameOptionsOut, err := client.ModifyPublicIpDnsNameOptions(ctx, &awsec2.ModifyPublicIpDnsNameOptionsInput{
		NetworkInterfaceId: aws.String(networkInterfaceID),
		HostnameType:       awsec2types.PublicIpDnsOptionPublicIpv4DnsName,
	})
	if err != nil {
		t.Fatalf("modify public ip dns name options: %v", err)
	}
	if modifyPublicIpDnsNameOptionsOut.Successful == nil || !aws.ToBool(modifyPublicIpDnsNameOptionsOut.Successful) {
		t.Fatalf("expected modify public ip dns name options successful true")
	}
}

func TestEC2Stage129ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"ModifyIpam",
		"ModifyIpamPool",
		"ModifyIpamResourceCidr",
		"ModifyIpamResourceDiscovery",
		"ModifyIpamScope",
		"ModifyLaunchTemplate",
		"ModifyLocalGatewayRoute",
		"ModifyManagedPrefixList",
		"ModifyPrivateDnsNameOptions",
		"ModifyPublicIpDnsNameOptions",
	}

	paramsByAction := map[string]map[string]string{
		"ModifyIpam": {
			"IpamId": "ipam-00000000129",
		},
		"ModifyIpamPool": {
			"IpamPoolId": "ipam-pool-00000000129",
		},
		"ModifyIpamResourceCidr": {
			"CurrentIpamScopeId": "ipam-scope-00000000129",
			"Monitored":          "true",
			"ResourceCidr":       "10.0.0.0/24",
			"ResourceId":         "subnet-00000001",
			"ResourceRegion":     "us-east-1",
		},
		"ModifyIpamResourceDiscovery": {
			"IpamResourceDiscoveryId": "ipam-rd-00000000129",
		},
		"ModifyIpamScope": {
			"IpamScopeId": "ipam-scope-00000000129",
		},
		"ModifyLaunchTemplate": {
			"LaunchTemplateName": "stage129-template",
		},
		"ModifyLocalGatewayRoute": {
			"LocalGatewayRouteTableId": "lgw-rtb-00000000129",
			"DestinationCidrBlock":     "10.129.0.0/16",
		},
		"ModifyManagedPrefixList": {
			"PrefixListId": "pl-00000000129",
		},
		"ModifyPrivateDnsNameOptions": {
			"InstanceId": "i-00000000129",
		},
		"ModifyPublicIpDnsNameOptions": {
			"NetworkInterfaceId": "eni-00000000129",
			"HostnameType":       "public-ipv4-dns-name",
		},
	}

	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, action := range implemented {
		resp := ec2Request(t, ts, action, paramsByAction[action])
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action)
		}
	}
}
