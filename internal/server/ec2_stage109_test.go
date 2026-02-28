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

func TestEC2Stage109SDKLifecycle(t *testing.T) {
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

	runOut, err := client.RunInstances(ctx, &awsec2.RunInstancesInput{
		ImageId:      aws.String("ami-00000000000000109"),
		InstanceType: awsec2types.InstanceTypeT3Micro,
		MinCount:     aws.Int32(1),
		MaxCount:     aws.Int32(1),
	})
	if err != nil {
		t.Fatalf("run instances: %v", err)
	}
	if len(runOut.Instances) == 0 {
		t.Fatalf("expected at least one instance")
	}
	instanceID := aws.ToString(runOut.Instances[0].InstanceId)
	if !strings.HasPrefix(instanceID, "i-") {
		t.Fatalf("unexpected instance id: %q", instanceID)
	}

	createLocalGatewayVirtualInterfaceGroupOut, err := client.CreateLocalGatewayVirtualInterfaceGroup(ctx, &awsec2.CreateLocalGatewayVirtualInterfaceGroupInput{
		LocalGatewayId: aws.String("lgw-00000000000000109"),
	})
	if err != nil {
		t.Fatalf("create local gateway virtual interface group: %v", err)
	}
	if createLocalGatewayVirtualInterfaceGroupOut.LocalGatewayVirtualInterfaceGroup == nil || !strings.HasPrefix(
		aws.ToString(createLocalGatewayVirtualInterfaceGroupOut.LocalGatewayVirtualInterfaceGroup.LocalGatewayVirtualInterfaceGroupId),
		"lgw-vifgrp-",
	) {
		t.Fatalf("unexpected local gateway virtual interface group output: %#v", createLocalGatewayVirtualInterfaceGroupOut.LocalGatewayVirtualInterfaceGroup)
	}

	createMacTaskOut, err := client.CreateMacSystemIntegrityProtectionModificationTask(ctx, &awsec2.CreateMacSystemIntegrityProtectionModificationTaskInput{
		InstanceId:                         aws.String(instanceID),
		MacSystemIntegrityProtectionStatus: awsec2types.MacSystemIntegrityProtectionSettingStatusEnabled,
	})
	if err != nil {
		t.Fatalf("create mac system integrity protection modification task: %v", err)
	}
	if createMacTaskOut.MacModificationTask == nil || !strings.HasPrefix(
		aws.ToString(createMacTaskOut.MacModificationTask.MacModificationTaskId),
		"mmt-",
	) {
		t.Fatalf("unexpected mac modification task output: %#v", createMacTaskOut.MacModificationTask)
	}

	createManagedPrefixListOut, err := client.CreateManagedPrefixList(ctx, &awsec2.CreateManagedPrefixListInput{
		AddressFamily:  aws.String("ipv4"),
		MaxEntries:     aws.Int32(10),
		PrefixListName: aws.String("stage109-prefix-list"),
		Entries: []awsec2types.AddPrefixListEntry{
			{Cidr: aws.String("10.109.0.0/16"), Description: aws.String("stage109")},
		},
	})
	if err != nil {
		t.Fatalf("create managed prefix list: %v", err)
	}
	if createManagedPrefixListOut.PrefixList == nil || !strings.HasPrefix(
		aws.ToString(createManagedPrefixListOut.PrefixList.PrefixListId),
		"pl-",
	) {
		t.Fatalf("unexpected managed prefix list output: %#v", createManagedPrefixListOut.PrefixList)
	}

	createNetworkInsightsAccessScopeOut, err := client.CreateNetworkInsightsAccessScope(ctx, &awsec2.CreateNetworkInsightsAccessScopeInput{
		ClientToken: aws.String("stage109-nias-token"),
	})
	if err != nil {
		t.Fatalf("create network insights access scope: %v", err)
	}
	if createNetworkInsightsAccessScopeOut.NetworkInsightsAccessScope == nil || !strings.HasPrefix(
		aws.ToString(createNetworkInsightsAccessScopeOut.NetworkInsightsAccessScope.NetworkInsightsAccessScopeId),
		"nias-",
	) {
		t.Fatalf("unexpected network insights access scope output: %#v", createNetworkInsightsAccessScopeOut.NetworkInsightsAccessScope)
	}

	createNetworkInsightsPathOut, err := client.CreateNetworkInsightsPath(ctx, &awsec2.CreateNetworkInsightsPathInput{
		ClientToken: aws.String("stage109-nip-token"),
		Protocol:    awsec2types.ProtocolTcp,
		Source:      aws.String(instanceID),
		Destination: aws.String("eni-00000000000000109"),
	})
	if err != nil {
		t.Fatalf("create network insights path: %v", err)
	}
	if createNetworkInsightsPathOut.NetworkInsightsPath == nil || !strings.HasPrefix(
		aws.ToString(createNetworkInsightsPathOut.NetworkInsightsPath.NetworkInsightsPathId),
		"nip-",
	) {
		t.Fatalf("unexpected network insights path output: %#v", createNetworkInsightsPathOut.NetworkInsightsPath)
	}

	createPublicIpv4PoolOut, err := client.CreatePublicIpv4Pool(ctx, &awsec2.CreatePublicIpv4PoolInput{})
	if err != nil {
		t.Fatalf("create public ipv4 pool: %v", err)
	}
	if !strings.HasPrefix(aws.ToString(createPublicIpv4PoolOut.PoolId), "ipv4pool-") {
		t.Fatalf("unexpected public ipv4 pool output: %#v", createPublicIpv4PoolOut.PoolId)
	}

	createReplaceRootVolumeTaskOut, err := client.CreateReplaceRootVolumeTask(ctx, &awsec2.CreateReplaceRootVolumeTaskInput{
		InstanceId: aws.String(instanceID),
	})
	if err != nil {
		t.Fatalf("create replace root volume task: %v", err)
	}
	if createReplaceRootVolumeTaskOut.ReplaceRootVolumeTask == nil || !strings.HasPrefix(
		aws.ToString(createReplaceRootVolumeTaskOut.ReplaceRootVolumeTask.ReplaceRootVolumeTaskId),
		"replacevol-",
	) {
		t.Fatalf("unexpected replace root volume task output: %#v", createReplaceRootVolumeTaskOut.ReplaceRootVolumeTask)
	}

	createReservedInstancesListingOut, err := client.CreateReservedInstancesListing(ctx, &awsec2.CreateReservedInstancesListingInput{
		ClientToken:         aws.String("stage109-ril-token"),
		InstanceCount:       aws.Int32(1),
		ReservedInstancesId: aws.String("ri-stage109"),
		PriceSchedules: []awsec2types.PriceScheduleSpecification{
			{Price: aws.Float64(10.5), Term: aws.Int64(1), CurrencyCode: awsec2types.CurrencyCodeValuesUsd},
		},
	})
	if err != nil {
		t.Fatalf("create reserved instances listing: %v", err)
	}
	if len(createReservedInstancesListingOut.ReservedInstancesListings) != 1 || !strings.HasPrefix(
		aws.ToString(createReservedInstancesListingOut.ReservedInstancesListings[0].ReservedInstancesListingId),
		"ril-",
	) {
		t.Fatalf("unexpected reserved instances listing output: %#v", createReservedInstancesListingOut.ReservedInstancesListings)
	}

	createRestoreImageTaskOut, err := client.CreateRestoreImageTask(ctx, &awsec2.CreateRestoreImageTaskInput{
		Bucket:    aws.String("stage109-bucket"),
		ObjectKey: aws.String("images/stage109.ova"),
	})
	if err != nil {
		t.Fatalf("create restore image task: %v", err)
	}
	if !strings.HasPrefix(aws.ToString(createRestoreImageTaskOut.ImageId), "ami-") {
		t.Fatalf("unexpected restore image task output: %#v", createRestoreImageTaskOut.ImageId)
	}

	createVolumeOut, err := client.CreateVolume(ctx, &awsec2.CreateVolumeInput{
		AvailabilityZone: runOut.Instances[0].Placement.AvailabilityZone,
		Size:             aws.Int32(8),
	})
	if err != nil {
		t.Fatalf("create volume: %v", err)
	}
	if createVolumeOut.VolumeId == nil {
		t.Fatalf("expected created volume id")
	}

	_, err = client.AttachVolume(ctx, &awsec2.AttachVolumeInput{
		Device:     aws.String("/dev/sdf"),
		InstanceId: aws.String(instanceID),
		VolumeId:   createVolumeOut.VolumeId,
	})
	if err != nil {
		t.Fatalf("attach volume: %v", err)
	}

	createSnapshotsOut, err := client.CreateSnapshots(ctx, &awsec2.CreateSnapshotsInput{
		Description: aws.String("stage109 snapshots"),
		InstanceSpecification: &awsec2types.InstanceSpecification{
			InstanceId: aws.String(instanceID),
		},
	})
	if err != nil {
		t.Fatalf("create snapshots: %v", err)
	}
	if len(createSnapshotsOut.Snapshots) == 0 || !strings.HasPrefix(aws.ToString(createSnapshotsOut.Snapshots[0].SnapshotId), "snap-") {
		t.Fatalf("unexpected create snapshots output: %#v", createSnapshotsOut.Snapshots)
	}
}

func TestEC2Stage109ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"CreateLocalGatewayVirtualInterfaceGroup",
		"CreateMacSystemIntegrityProtectionModificationTask",
		"CreateManagedPrefixList",
		"CreateNetworkInsightsAccessScope",
		"CreateNetworkInsightsPath",
		"CreatePublicIpv4Pool",
		"CreateReplaceRootVolumeTask",
		"CreateReservedInstancesListing",
		"CreateRestoreImageTask",
		"CreateSnapshots",
	}

	paramsByAction := map[string]map[string]string{
		"CreateLocalGatewayVirtualInterfaceGroup": {
			"LocalGatewayId": "lgw-00000000109",
		},
		"CreateMacSystemIntegrityProtectionModificationTask": {
			"InstanceId":                         "i-00000000109",
			"MacSystemIntegrityProtectionStatus": "enabled",
		},
		"CreateManagedPrefixList": {
			"AddressFamily":  "ipv4",
			"MaxEntries":     "5",
			"PrefixListName": "stage109-prefix-list",
		},
		"CreateNetworkInsightsAccessScope": {
			"ClientToken": "stage109-nias-token",
		},
		"CreateNetworkInsightsPath": {
			"ClientToken": "stage109-nip-token",
			"Protocol":    "tcp",
			"Source":      "i-00000000109",
		},
		"CreatePublicIpv4Pool": {},
		"CreateReplaceRootVolumeTask": {
			"InstanceId": "i-00000000109",
		},
		"CreateReservedInstancesListing": {
			"ClientToken":            "stage109-ril-token",
			"InstanceCount":          "1",
			"ReservedInstancesId":    "ri-stage109",
			"PriceSchedules.1.Price": "10.5",
			"PriceSchedules.1.Term":  "1",
		},
		"CreateRestoreImageTask": {
			"Bucket":    "stage109-bucket",
			"ObjectKey": "images/stage109.ova",
		},
		"CreateSnapshots": {
			"InstanceSpecification.InstanceId": "i-00000000109",
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
