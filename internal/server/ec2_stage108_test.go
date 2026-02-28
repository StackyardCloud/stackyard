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

func TestEC2Stage108SDKLifecycle(t *testing.T) {
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
		Description: aws.String("stage108-ipam"),
	})
	if err != nil {
		t.Fatalf("create ipam: %v", err)
	}
	if createIpamOut.Ipam == nil || !strings.HasPrefix(aws.ToString(createIpamOut.Ipam.IpamId), "ipam-") {
		t.Fatalf("unexpected ipam output: %#v", createIpamOut.Ipam)
	}

	createIpamScopeOut, err := client.CreateIpamScope(ctx, &awsec2.CreateIpamScopeInput{
		IpamId: createIpamOut.Ipam.IpamId,
	})
	if err != nil {
		t.Fatalf("create ipam scope: %v", err)
	}
	if createIpamScopeOut.IpamScope == nil || !strings.HasPrefix(aws.ToString(createIpamScopeOut.IpamScope.IpamScopeId), "ipam-scope-") {
		t.Fatalf("unexpected ipam scope output: %#v", createIpamScopeOut.IpamScope)
	}

	createIpamPoolOut, err := client.CreateIpamPool(ctx, &awsec2.CreateIpamPoolInput{
		AddressFamily: awsec2types.AddressFamilyIpv4,
		IpamScopeId:   createIpamScopeOut.IpamScope.IpamScopeId,
	})
	if err != nil {
		t.Fatalf("create ipam pool: %v", err)
	}
	if createIpamPoolOut.IpamPool == nil || !strings.HasPrefix(aws.ToString(createIpamPoolOut.IpamPool.IpamPoolId), "ipam-pool-") {
		t.Fatalf("unexpected ipam pool output: %#v", createIpamPoolOut.IpamPool)
	}

	createIpamResourceDiscoveryOut, err := client.CreateIpamResourceDiscovery(ctx, &awsec2.CreateIpamResourceDiscoveryInput{
		Description: aws.String("stage108-discovery"),
		OperatingRegions: []awsec2types.AddIpamOperatingRegion{
			{RegionName: aws.String("us-east-1")},
		},
	})
	if err != nil {
		t.Fatalf("create ipam resource discovery: %v", err)
	}
	if createIpamResourceDiscoveryOut.IpamResourceDiscovery == nil || !strings.HasPrefix(
		aws.ToString(createIpamResourceDiscoveryOut.IpamResourceDiscovery.IpamResourceDiscoveryId),
		"ipam-rd-",
	) {
		t.Fatalf("unexpected ipam resource discovery output: %#v", createIpamResourceDiscoveryOut.IpamResourceDiscovery)
	}

	createLaunchTemplateOut, err := client.CreateLaunchTemplate(ctx, &awsec2.CreateLaunchTemplateInput{
		LaunchTemplateName: aws.String("stage108-template"),
		LaunchTemplateData: &awsec2types.RequestLaunchTemplateData{
			ImageId: aws.String("ami-00000000000000108"),
		},
	})
	if err != nil {
		t.Fatalf("create launch template: %v", err)
	}
	if createLaunchTemplateOut.LaunchTemplate == nil || !strings.HasPrefix(aws.ToString(createLaunchTemplateOut.LaunchTemplate.LaunchTemplateId), "lt-") {
		t.Fatalf("unexpected launch template output: %#v", createLaunchTemplateOut.LaunchTemplate)
	}

	createLaunchTemplateVersionOut, err := client.CreateLaunchTemplateVersion(ctx, &awsec2.CreateLaunchTemplateVersionInput{
		LaunchTemplateId: createLaunchTemplateOut.LaunchTemplate.LaunchTemplateId,
		LaunchTemplateData: &awsec2types.RequestLaunchTemplateData{
			InstanceType: awsec2types.InstanceTypeT3Micro,
		},
		VersionDescription: aws.String("stage108-v2"),
	})
	if err != nil {
		t.Fatalf("create launch template version: %v", err)
	}
	if createLaunchTemplateVersionOut.LaunchTemplateVersion == nil || aws.ToInt64(createLaunchTemplateVersionOut.LaunchTemplateVersion.VersionNumber) != 2 {
		t.Fatalf("unexpected launch template version output: %#v", createLaunchTemplateVersionOut.LaunchTemplateVersion)
	}

	createLocalGatewayRouteTableOut, err := client.CreateLocalGatewayRouteTable(ctx, &awsec2.CreateLocalGatewayRouteTableInput{
		LocalGatewayId: aws.String("lgw-00000000000000108"),
	})
	if err != nil {
		t.Fatalf("create local gateway route table: %v", err)
	}
	if createLocalGatewayRouteTableOut.LocalGatewayRouteTable == nil || !strings.HasPrefix(
		aws.ToString(createLocalGatewayRouteTableOut.LocalGatewayRouteTable.LocalGatewayRouteTableId),
		"lgw-rtb-",
	) {
		t.Fatalf("unexpected local gateway route table output: %#v", createLocalGatewayRouteTableOut.LocalGatewayRouteTable)
	}

	createLocalGatewayRouteOut, err := client.CreateLocalGatewayRoute(ctx, &awsec2.CreateLocalGatewayRouteInput{
		LocalGatewayRouteTableId: createLocalGatewayRouteTableOut.LocalGatewayRouteTable.LocalGatewayRouteTableId,
		DestinationCidrBlock:     aws.String("10.108.0.0/16"),
	})
	if err != nil {
		t.Fatalf("create local gateway route: %v", err)
	}
	if createLocalGatewayRouteOut.Route == nil || aws.ToString(createLocalGatewayRouteOut.Route.LocalGatewayRouteTableId) != aws.ToString(createLocalGatewayRouteTableOut.LocalGatewayRouteTable.LocalGatewayRouteTableId) {
		t.Fatalf("unexpected local gateway route output: %#v", createLocalGatewayRouteOut.Route)
	}

	createLocalGatewayRouteTableVifAssociationOut, err := client.CreateLocalGatewayRouteTableVirtualInterfaceGroupAssociation(ctx, &awsec2.CreateLocalGatewayRouteTableVirtualInterfaceGroupAssociationInput{
		LocalGatewayRouteTableId:            createLocalGatewayRouteTableOut.LocalGatewayRouteTable.LocalGatewayRouteTableId,
		LocalGatewayVirtualInterfaceGroupId: aws.String("lgw-vifgrp-00000000000000108"),
	})
	if err != nil {
		t.Fatalf("create local gateway route table virtual interface group association: %v", err)
	}
	if createLocalGatewayRouteTableVifAssociationOut.LocalGatewayRouteTableVirtualInterfaceGroupAssociation == nil || !strings.HasPrefix(
		aws.ToString(createLocalGatewayRouteTableVifAssociationOut.LocalGatewayRouteTableVirtualInterfaceGroupAssociation.LocalGatewayRouteTableVirtualInterfaceGroupAssociationId),
		"lgw-vif-assoc-",
	) {
		t.Fatalf(
			"unexpected local gateway route table virtual interface group association output: %#v",
			createLocalGatewayRouteTableVifAssociationOut.LocalGatewayRouteTableVirtualInterfaceGroupAssociation,
		)
	}

	createLocalGatewayRouteTableVpcAssociationOut, err := client.CreateLocalGatewayRouteTableVpcAssociation(ctx, &awsec2.CreateLocalGatewayRouteTableVpcAssociationInput{
		LocalGatewayRouteTableId: createLocalGatewayRouteTableOut.LocalGatewayRouteTable.LocalGatewayRouteTableId,
		VpcId:                    aws.String("vpc-00000001"),
	})
	if err != nil {
		t.Fatalf("create local gateway route table vpc association: %v", err)
	}
	if createLocalGatewayRouteTableVpcAssociationOut.LocalGatewayRouteTableVpcAssociation == nil || !strings.HasPrefix(
		aws.ToString(createLocalGatewayRouteTableVpcAssociationOut.LocalGatewayRouteTableVpcAssociation.LocalGatewayRouteTableVpcAssociationId),
		"lgw-vpc-assoc-",
	) {
		t.Fatalf("unexpected local gateway route table vpc association output: %#v", createLocalGatewayRouteTableVpcAssociationOut.LocalGatewayRouteTableVpcAssociation)
	}

	createLocalGatewayVirtualInterfaceOut, err := client.CreateLocalGatewayVirtualInterface(ctx, &awsec2.CreateLocalGatewayVirtualInterfaceInput{
		LocalAddress:                        aws.String("169.254.108.1"),
		LocalGatewayVirtualInterfaceGroupId: aws.String("lgw-vifgrp-00000000000000108"),
		OutpostLagId:                        aws.String("lag-00000000000000108"),
		PeerAddress:                         aws.String("169.254.108.2"),
		Vlan:                                aws.Int32(108),
	})
	if err != nil {
		t.Fatalf("create local gateway virtual interface: %v", err)
	}
	if createLocalGatewayVirtualInterfaceOut.LocalGatewayVirtualInterface == nil || !strings.HasPrefix(
		aws.ToString(createLocalGatewayVirtualInterfaceOut.LocalGatewayVirtualInterface.LocalGatewayVirtualInterfaceId),
		"lgw-vif-",
	) {
		t.Fatalf("unexpected local gateway virtual interface output: %#v", createLocalGatewayVirtualInterfaceOut.LocalGatewayVirtualInterface)
	}
}

func TestEC2Stage108ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"CreateIpamPool",
		"CreateIpamResourceDiscovery",
		"CreateIpamScope",
		"CreateLaunchTemplate",
		"CreateLaunchTemplateVersion",
		"CreateLocalGatewayRoute",
		"CreateLocalGatewayRouteTable",
		"CreateLocalGatewayRouteTableVirtualInterfaceGroupAssociation",
		"CreateLocalGatewayRouteTableVpcAssociation",
		"CreateLocalGatewayVirtualInterface",
	}

	paramsByAction := map[string]map[string]string{
		"CreateIpamPool": {
			"AddressFamily": "ipv4",
			"IpamScopeId":   "ipam-scope-00000000108",
		},
		"CreateIpamResourceDiscovery": {
			"Description": "stage108-discovery",
		},
		"CreateIpamScope": {
			"IpamId": "ipam-00000000108",
		},
		"CreateLaunchTemplate": {
			"LaunchTemplateName":         "stage108-template",
			"LaunchTemplateData.ImageId": "ami-00000000000000108",
		},
		"CreateLaunchTemplateVersion": {
			"LaunchTemplateName":         "stage108-template",
			"LaunchTemplateData.ImageId": "ami-00000000000000108",
		},
		"CreateLocalGatewayRoute": {
			"LocalGatewayRouteTableId": "lgw-rtb-00000000108",
		},
		"CreateLocalGatewayRouteTable": {
			"LocalGatewayId": "lgw-00000000108",
		},
		"CreateLocalGatewayRouteTableVirtualInterfaceGroupAssociation": {
			"LocalGatewayRouteTableId":            "lgw-rtb-00000000108",
			"LocalGatewayVirtualInterfaceGroupId": "lgw-vifgrp-00000000108",
		},
		"CreateLocalGatewayRouteTableVpcAssociation": {
			"LocalGatewayRouteTableId": "lgw-rtb-00000000108",
			"VpcId":                    "vpc-00000001",
		},
		"CreateLocalGatewayVirtualInterface": {
			"LocalAddress":                        "169.254.108.1",
			"LocalGatewayVirtualInterfaceGroupId": "lgw-vifgrp-00000000108",
			"OutpostLagId":                        "lag-00000000108",
			"PeerAddress":                         "169.254.108.2",
			"Vlan":                                "108",
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
