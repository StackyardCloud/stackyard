package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awsec2 "github.com/aws/aws-sdk-go-v2/service/ec2"
	awsec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

func TestEC2Stage112SDKLifecycle(t *testing.T) {
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

	createLaunchTemplateOut, err := client.CreateLaunchTemplate(ctx, &awsec2.CreateLaunchTemplateInput{
		LaunchTemplateName: aws.String("stage112-template"),
		LaunchTemplateData: &awsec2types.RequestLaunchTemplateData{
			ImageId: aws.String("ami-00000000000000112"),
		},
	})
	if err != nil {
		t.Fatalf("create launch template: %v", err)
	}
	createLaunchTemplateVersionOut, err := client.CreateLaunchTemplateVersion(ctx, &awsec2.CreateLaunchTemplateVersionInput{
		LaunchTemplateId: createLaunchTemplateOut.LaunchTemplate.LaunchTemplateId,
		LaunchTemplateData: &awsec2types.RequestLaunchTemplateData{
			InstanceType: awsec2types.InstanceTypeT3Micro,
		},
	})
	if err != nil {
		t.Fatalf("create launch template version: %v", err)
	}
	deleteLaunchTemplateVersionsOut, err := client.DeleteLaunchTemplateVersions(ctx, &awsec2.DeleteLaunchTemplateVersionsInput{
		LaunchTemplateId: createLaunchTemplateOut.LaunchTemplate.LaunchTemplateId,
		Versions:         []string{strconv.FormatInt(aws.ToInt64(createLaunchTemplateVersionOut.LaunchTemplateVersion.VersionNumber), 10)},
	})
	if err != nil {
		t.Fatalf("delete launch template versions: %v", err)
	}
	if len(deleteLaunchTemplateVersionsOut.SuccessfullyDeletedLaunchTemplateVersions) != 1 ||
		aws.ToInt64(deleteLaunchTemplateVersionsOut.SuccessfullyDeletedLaunchTemplateVersions[0].VersionNumber) != aws.ToInt64(createLaunchTemplateVersionOut.LaunchTemplateVersion.VersionNumber) {
		t.Fatalf("unexpected delete launch template versions output: %#v", deleteLaunchTemplateVersionsOut)
	}

	createLocalGatewayRouteTableOut, err := client.CreateLocalGatewayRouteTable(ctx, &awsec2.CreateLocalGatewayRouteTableInput{
		LocalGatewayId: aws.String("lgw-00000000000000112"),
	})
	if err != nil {
		t.Fatalf("create local gateway route table: %v", err)
	}

	createLocalGatewayVirtualInterfaceGroupOut, err := client.CreateLocalGatewayVirtualInterfaceGroup(ctx, &awsec2.CreateLocalGatewayVirtualInterfaceGroupInput{
		LocalGatewayId: aws.String("lgw-00000000000000112"),
	})
	if err != nil {
		t.Fatalf("create local gateway virtual interface group: %v", err)
	}

	createLocalGatewayRouteOut, err := client.CreateLocalGatewayRoute(ctx, &awsec2.CreateLocalGatewayRouteInput{
		LocalGatewayRouteTableId: createLocalGatewayRouteTableOut.LocalGatewayRouteTable.LocalGatewayRouteTableId,
		DestinationCidrBlock:     aws.String("10.112.0.0/16"),
	})
	if err != nil {
		t.Fatalf("create local gateway route: %v", err)
	}
	if createLocalGatewayRouteOut.Route == nil {
		t.Fatalf("expected local gateway route output")
	}

	createLocalGatewayRouteTableVirtualInterfaceGroupAssociationOut, err := client.CreateLocalGatewayRouteTableVirtualInterfaceGroupAssociation(ctx, &awsec2.CreateLocalGatewayRouteTableVirtualInterfaceGroupAssociationInput{
		LocalGatewayRouteTableId:            createLocalGatewayRouteTableOut.LocalGatewayRouteTable.LocalGatewayRouteTableId,
		LocalGatewayVirtualInterfaceGroupId: createLocalGatewayVirtualInterfaceGroupOut.LocalGatewayVirtualInterfaceGroup.LocalGatewayVirtualInterfaceGroupId,
	})
	if err != nil {
		t.Fatalf("create local gateway route table virtual interface group association: %v", err)
	}

	createLocalGatewayRouteTableVpcAssociationOut, err := client.CreateLocalGatewayRouteTableVpcAssociation(ctx, &awsec2.CreateLocalGatewayRouteTableVpcAssociationInput{
		LocalGatewayRouteTableId: createLocalGatewayRouteTableOut.LocalGatewayRouteTable.LocalGatewayRouteTableId,
		VpcId:                    aws.String("vpc-00000001"),
	})
	if err != nil {
		t.Fatalf("create local gateway route table vpc association: %v", err)
	}

	createLocalGatewayVirtualInterfaceOut, err := client.CreateLocalGatewayVirtualInterface(ctx, &awsec2.CreateLocalGatewayVirtualInterfaceInput{
		LocalAddress:                        aws.String("169.254.112.1"),
		LocalGatewayVirtualInterfaceGroupId: createLocalGatewayVirtualInterfaceGroupOut.LocalGatewayVirtualInterfaceGroup.LocalGatewayVirtualInterfaceGroupId,
		OutpostLagId:                        aws.String("lag-00000000000000112"),
		PeerAddress:                         aws.String("169.254.112.2"),
		Vlan:                                aws.Int32(112),
	})
	if err != nil {
		t.Fatalf("create local gateway virtual interface: %v", err)
	}

	createManagedPrefixListOut, err := client.CreateManagedPrefixList(ctx, &awsec2.CreateManagedPrefixListInput{
		AddressFamily:  aws.String("ipv4"),
		MaxEntries:     aws.Int32(5),
		PrefixListName: aws.String("stage112-prefix-list"),
	})
	if err != nil {
		t.Fatalf("create managed prefix list: %v", err)
	}

	createNetworkInsightsAccessScopeOut, err := client.CreateNetworkInsightsAccessScope(ctx, &awsec2.CreateNetworkInsightsAccessScopeInput{
		ClientToken: aws.String("stage112-nias-token"),
	})
	if err != nil {
		t.Fatalf("create network insights access scope: %v", err)
	}

	deleteLocalGatewayRouteOut, err := client.DeleteLocalGatewayRoute(ctx, &awsec2.DeleteLocalGatewayRouteInput{
		LocalGatewayRouteTableId: createLocalGatewayRouteTableOut.LocalGatewayRouteTable.LocalGatewayRouteTableId,
		DestinationCidrBlock:     aws.String("10.112.0.0/16"),
	})
	if err != nil {
		t.Fatalf("delete local gateway route: %v", err)
	}
	if deleteLocalGatewayRouteOut.Route == nil || aws.ToString(deleteLocalGatewayRouteOut.Route.LocalGatewayRouteTableId) != aws.ToString(createLocalGatewayRouteTableOut.LocalGatewayRouteTable.LocalGatewayRouteTableId) {
		t.Fatalf("unexpected delete local gateway route output: %#v", deleteLocalGatewayRouteOut.Route)
	}

	deleteLocalGatewayRouteTableVirtualInterfaceGroupAssociationOut, err := client.DeleteLocalGatewayRouteTableVirtualInterfaceGroupAssociation(ctx, &awsec2.DeleteLocalGatewayRouteTableVirtualInterfaceGroupAssociationInput{
		LocalGatewayRouteTableVirtualInterfaceGroupAssociationId: createLocalGatewayRouteTableVirtualInterfaceGroupAssociationOut.LocalGatewayRouteTableVirtualInterfaceGroupAssociation.LocalGatewayRouteTableVirtualInterfaceGroupAssociationId,
	})
	if err != nil {
		t.Fatalf("delete local gateway route table virtual interface group association: %v", err)
	}
	if deleteLocalGatewayRouteTableVirtualInterfaceGroupAssociationOut.LocalGatewayRouteTableVirtualInterfaceGroupAssociation == nil ||
		aws.ToString(deleteLocalGatewayRouteTableVirtualInterfaceGroupAssociationOut.LocalGatewayRouteTableVirtualInterfaceGroupAssociation.LocalGatewayRouteTableVirtualInterfaceGroupAssociationId) != aws.ToString(createLocalGatewayRouteTableVirtualInterfaceGroupAssociationOut.LocalGatewayRouteTableVirtualInterfaceGroupAssociation.LocalGatewayRouteTableVirtualInterfaceGroupAssociationId) {
		t.Fatalf("unexpected delete local gateway route table virtual interface group association output: %#v", deleteLocalGatewayRouteTableVirtualInterfaceGroupAssociationOut.LocalGatewayRouteTableVirtualInterfaceGroupAssociation)
	}

	deleteLocalGatewayRouteTableVpcAssociationOut, err := client.DeleteLocalGatewayRouteTableVpcAssociation(ctx, &awsec2.DeleteLocalGatewayRouteTableVpcAssociationInput{
		LocalGatewayRouteTableVpcAssociationId: createLocalGatewayRouteTableVpcAssociationOut.LocalGatewayRouteTableVpcAssociation.LocalGatewayRouteTableVpcAssociationId,
	})
	if err != nil {
		t.Fatalf("delete local gateway route table vpc association: %v", err)
	}
	if deleteLocalGatewayRouteTableVpcAssociationOut.LocalGatewayRouteTableVpcAssociation == nil ||
		aws.ToString(deleteLocalGatewayRouteTableVpcAssociationOut.LocalGatewayRouteTableVpcAssociation.LocalGatewayRouteTableVpcAssociationId) != aws.ToString(createLocalGatewayRouteTableVpcAssociationOut.LocalGatewayRouteTableVpcAssociation.LocalGatewayRouteTableVpcAssociationId) {
		t.Fatalf("unexpected delete local gateway route table vpc association output: %#v", deleteLocalGatewayRouteTableVpcAssociationOut.LocalGatewayRouteTableVpcAssociation)
	}

	deleteLocalGatewayVirtualInterfaceOut, err := client.DeleteLocalGatewayVirtualInterface(ctx, &awsec2.DeleteLocalGatewayVirtualInterfaceInput{
		LocalGatewayVirtualInterfaceId: createLocalGatewayVirtualInterfaceOut.LocalGatewayVirtualInterface.LocalGatewayVirtualInterfaceId,
	})
	if err != nil {
		t.Fatalf("delete local gateway virtual interface: %v", err)
	}
	if deleteLocalGatewayVirtualInterfaceOut.LocalGatewayVirtualInterface == nil ||
		aws.ToString(deleteLocalGatewayVirtualInterfaceOut.LocalGatewayVirtualInterface.LocalGatewayVirtualInterfaceId) != aws.ToString(createLocalGatewayVirtualInterfaceOut.LocalGatewayVirtualInterface.LocalGatewayVirtualInterfaceId) {
		t.Fatalf("unexpected delete local gateway virtual interface output: %#v", deleteLocalGatewayVirtualInterfaceOut.LocalGatewayVirtualInterface)
	}

	deleteLocalGatewayVirtualInterfaceGroupOut, err := client.DeleteLocalGatewayVirtualInterfaceGroup(ctx, &awsec2.DeleteLocalGatewayVirtualInterfaceGroupInput{
		LocalGatewayVirtualInterfaceGroupId: createLocalGatewayVirtualInterfaceGroupOut.LocalGatewayVirtualInterfaceGroup.LocalGatewayVirtualInterfaceGroupId,
	})
	if err != nil {
		t.Fatalf("delete local gateway virtual interface group: %v", err)
	}
	if deleteLocalGatewayVirtualInterfaceGroupOut.LocalGatewayVirtualInterfaceGroup == nil ||
		aws.ToString(deleteLocalGatewayVirtualInterfaceGroupOut.LocalGatewayVirtualInterfaceGroup.LocalGatewayVirtualInterfaceGroupId) != aws.ToString(createLocalGatewayVirtualInterfaceGroupOut.LocalGatewayVirtualInterfaceGroup.LocalGatewayVirtualInterfaceGroupId) {
		t.Fatalf("unexpected delete local gateway virtual interface group output: %#v", deleteLocalGatewayVirtualInterfaceGroupOut.LocalGatewayVirtualInterfaceGroup)
	}

	deleteManagedPrefixListOut, err := client.DeleteManagedPrefixList(ctx, &awsec2.DeleteManagedPrefixListInput{
		PrefixListId: createManagedPrefixListOut.PrefixList.PrefixListId,
	})
	if err != nil {
		t.Fatalf("delete managed prefix list: %v", err)
	}
	if deleteManagedPrefixListOut.PrefixList == nil || aws.ToString(deleteManagedPrefixListOut.PrefixList.PrefixListId) != aws.ToString(createManagedPrefixListOut.PrefixList.PrefixListId) {
		t.Fatalf("unexpected delete managed prefix list output: %#v", deleteManagedPrefixListOut.PrefixList)
	}

	deleteNetworkInsightsAccessScopeOut, err := client.DeleteNetworkInsightsAccessScope(ctx, &awsec2.DeleteNetworkInsightsAccessScopeInput{
		NetworkInsightsAccessScopeId: createNetworkInsightsAccessScopeOut.NetworkInsightsAccessScope.NetworkInsightsAccessScopeId,
	})
	if err != nil {
		t.Fatalf("delete network insights access scope: %v", err)
	}
	if aws.ToString(deleteNetworkInsightsAccessScopeOut.NetworkInsightsAccessScopeId) != aws.ToString(createNetworkInsightsAccessScopeOut.NetworkInsightsAccessScope.NetworkInsightsAccessScopeId) {
		t.Fatalf("unexpected delete network insights access scope output: %#v", deleteNetworkInsightsAccessScopeOut.NetworkInsightsAccessScopeId)
	}

	deleteNetworkInsightsAccessScopeAnalysisOut, err := client.DeleteNetworkInsightsAccessScopeAnalysis(ctx, &awsec2.DeleteNetworkInsightsAccessScopeAnalysisInput{
		NetworkInsightsAccessScopeAnalysisId: aws.String("niasa-00000000112"),
	})
	if err != nil {
		t.Fatalf("delete network insights access scope analysis: %v", err)
	}
	if aws.ToString(deleteNetworkInsightsAccessScopeAnalysisOut.NetworkInsightsAccessScopeAnalysisId) != "niasa-00000000112" {
		t.Fatalf("unexpected delete network insights access scope analysis output: %#v", deleteNetworkInsightsAccessScopeAnalysisOut.NetworkInsightsAccessScopeAnalysisId)
	}

	deleteLocalGatewayRouteTableOut, err := client.DeleteLocalGatewayRouteTable(ctx, &awsec2.DeleteLocalGatewayRouteTableInput{
		LocalGatewayRouteTableId: createLocalGatewayRouteTableOut.LocalGatewayRouteTable.LocalGatewayRouteTableId,
	})
	if err != nil {
		t.Fatalf("delete local gateway route table: %v", err)
	}
	if deleteLocalGatewayRouteTableOut.LocalGatewayRouteTable == nil ||
		aws.ToString(deleteLocalGatewayRouteTableOut.LocalGatewayRouteTable.LocalGatewayRouteTableId) != aws.ToString(createLocalGatewayRouteTableOut.LocalGatewayRouteTable.LocalGatewayRouteTableId) {
		t.Fatalf("unexpected delete local gateway route table output: %#v", deleteLocalGatewayRouteTableOut.LocalGatewayRouteTable)
	}
}

func TestEC2Stage112ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"DeleteLaunchTemplateVersions",
		"DeleteLocalGatewayRoute",
		"DeleteLocalGatewayRouteTable",
		"DeleteLocalGatewayRouteTableVirtualInterfaceGroupAssociation",
		"DeleteLocalGatewayRouteTableVpcAssociation",
		"DeleteLocalGatewayVirtualInterface",
		"DeleteLocalGatewayVirtualInterfaceGroup",
		"DeleteManagedPrefixList",
		"DeleteNetworkInsightsAccessScope",
		"DeleteNetworkInsightsAccessScopeAnalysis",
	}

	paramsByAction := map[string]map[string]string{
		"DeleteLaunchTemplateVersions": {
			"LaunchTemplateId":        "lt-00000000112",
			"LaunchTemplateVersion.1": "2",
		},
		"DeleteLocalGatewayRoute": {
			"LocalGatewayRouteTableId": "lgw-rtb-00000000112",
			"DestinationCidrBlock":     "10.112.0.0/16",
		},
		"DeleteLocalGatewayRouteTable": {
			"LocalGatewayRouteTableId": "lgw-rtb-00000000112",
		},
		"DeleteLocalGatewayRouteTableVirtualInterfaceGroupAssociation": {
			"LocalGatewayRouteTableVirtualInterfaceGroupAssociationId": "lgw-vif-assoc-00000000112",
		},
		"DeleteLocalGatewayRouteTableVpcAssociation": {
			"LocalGatewayRouteTableVpcAssociationId": "lgw-vpc-assoc-00000000112",
		},
		"DeleteLocalGatewayVirtualInterface": {
			"LocalGatewayVirtualInterfaceId": "lgw-vif-00000000112",
		},
		"DeleteLocalGatewayVirtualInterfaceGroup": {
			"LocalGatewayVirtualInterfaceGroupId": "lgw-vifgrp-00000000112",
		},
		"DeleteManagedPrefixList": {
			"PrefixListId": "pl-00000000112",
		},
		"DeleteNetworkInsightsAccessScope": {
			"NetworkInsightsAccessScopeId": "nias-00000000112",
		},
		"DeleteNetworkInsightsAccessScopeAnalysis": {
			"NetworkInsightsAccessScopeAnalysisId": "niasa-00000000112",
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
