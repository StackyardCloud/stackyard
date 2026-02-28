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

func TestEC2Stage130SDKLifecycle(t *testing.T) {
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

	modifyReservedInstancesOut, err := client.ModifyReservedInstances(ctx, &awsec2.ModifyReservedInstancesInput{
		ReservedInstancesIds: []string{"ri-stage1300001"},
		TargetConfigurations: []awsec2types.ReservedInstancesConfiguration{
			{
				InstanceCount: aws.Int32(1),
				InstanceType:  awsec2types.InstanceType("m5.large"),
				Scope:         awsec2types.Scope("Region"),
			},
		},
	})
	if err != nil {
		t.Fatalf("modify reserved instances: %v", err)
	}
	if aws.ToString(modifyReservedInstancesOut.ReservedInstancesModificationId) == "" {
		t.Fatalf("expected reserved instances modification id")
	}

	createVolumeOut, err := client.CreateVolume(ctx, &awsec2.CreateVolumeInput{
		AvailabilityZone: aws.String("us-east-1a"),
		Size:             aws.Int32(8),
	})
	if err != nil || createVolumeOut.VolumeId == nil {
		t.Fatalf("create volume: %v", err)
	}
	createSnapshotOut, err := client.CreateSnapshot(ctx, &awsec2.CreateSnapshotInput{
		VolumeId: createVolumeOut.VolumeId,
	})
	if err != nil || createSnapshotOut.SnapshotId == nil {
		t.Fatalf("create snapshot: %v", err)
	}
	snapshotID := aws.ToString(createSnapshotOut.SnapshotId)

	if _, err := client.ModifySnapshotAttribute(ctx, &awsec2.ModifySnapshotAttributeInput{
		SnapshotId:    aws.String(snapshotID),
		OperationType: awsec2types.OperationTypeAdd,
		GroupNames:    []string{"all"},
	}); err != nil {
		t.Fatalf("modify snapshot attribute: %v", err)
	}

	modifySnapshotTierOut, err := client.ModifySnapshotTier(ctx, &awsec2.ModifySnapshotTierInput{
		SnapshotId:  aws.String(snapshotID),
		StorageTier: awsec2types.TargetStorageTierArchive,
	})
	if err != nil {
		t.Fatalf("modify snapshot tier: %v", err)
	}
	if aws.ToString(modifySnapshotTierOut.SnapshotId) != snapshotID {
		t.Fatalf("unexpected modify snapshot tier output: %#v", modifySnapshotTierOut)
	}

	modifySpotFleetRequestOut, err := client.ModifySpotFleetRequest(ctx, &awsec2.ModifySpotFleetRequestInput{
		SpotFleetRequestId: aws.String("sfr-stage1300001"),
		TargetCapacity:     aws.Int32(1),
	})
	if err != nil {
		t.Fatalf("modify spot fleet request: %v", err)
	}
	if modifySpotFleetRequestOut.Return == nil || !aws.ToBool(modifySpotFleetRequestOut.Return) {
		t.Fatalf("expected modify spot fleet request return true")
	}

	createTrafficMirrorFilterOut, err := client.CreateTrafficMirrorFilter(ctx, &awsec2.CreateTrafficMirrorFilterInput{
		Description: aws.String("stage130 filter"),
	})
	if err != nil || createTrafficMirrorFilterOut.TrafficMirrorFilter == nil || createTrafficMirrorFilterOut.TrafficMirrorFilter.TrafficMirrorFilterId == nil {
		t.Fatalf("create traffic mirror filter: %v", err)
	}
	trafficMirrorFilterID := aws.ToString(createTrafficMirrorFilterOut.TrafficMirrorFilter.TrafficMirrorFilterId)

	modifyTrafficMirrorFilterNetworkServicesOut, err := client.ModifyTrafficMirrorFilterNetworkServices(ctx, &awsec2.ModifyTrafficMirrorFilterNetworkServicesInput{
		TrafficMirrorFilterId: aws.String(trafficMirrorFilterID),
		AddNetworkServices:    []awsec2types.TrafficMirrorNetworkService{awsec2types.TrafficMirrorNetworkServiceAmazonDns},
	})
	if err != nil {
		t.Fatalf("modify traffic mirror filter network services: %v", err)
	}
	if modifyTrafficMirrorFilterNetworkServicesOut.TrafficMirrorFilter == nil ||
		aws.ToString(modifyTrafficMirrorFilterNetworkServicesOut.TrafficMirrorFilter.TrafficMirrorFilterId) != trafficMirrorFilterID {
		t.Fatalf("unexpected modify traffic mirror filter network services output: %#v", modifyTrafficMirrorFilterNetworkServicesOut.TrafficMirrorFilter)
	}

	createTrafficMirrorFilterRuleOut, err := client.CreateTrafficMirrorFilterRule(ctx, &awsec2.CreateTrafficMirrorFilterRuleInput{
		DestinationCidrBlock:  aws.String("10.130.0.0/16"),
		RuleAction:            awsec2types.TrafficMirrorRuleActionAccept,
		RuleNumber:            aws.Int32(1),
		SourceCidrBlock:       aws.String("10.0.0.0/16"),
		TrafficDirection:      awsec2types.TrafficDirectionIngress,
		TrafficMirrorFilterId: aws.String(trafficMirrorFilterID),
	})
	if err != nil || createTrafficMirrorFilterRuleOut.TrafficMirrorFilterRule == nil || createTrafficMirrorFilterRuleOut.TrafficMirrorFilterRule.TrafficMirrorFilterRuleId == nil {
		t.Fatalf("create traffic mirror filter rule: %v", err)
	}
	trafficMirrorFilterRuleID := aws.ToString(createTrafficMirrorFilterRuleOut.TrafficMirrorFilterRule.TrafficMirrorFilterRuleId)

	modifyTrafficMirrorFilterRuleOut, err := client.ModifyTrafficMirrorFilterRule(ctx, &awsec2.ModifyTrafficMirrorFilterRuleInput{
		TrafficMirrorFilterRuleId: aws.String(trafficMirrorFilterRuleID),
		Description:               aws.String("stage130 rule updated"),
		RuleAction:                awsec2types.TrafficMirrorRuleActionReject,
	})
	if err != nil {
		t.Fatalf("modify traffic mirror filter rule: %v", err)
	}
	if modifyTrafficMirrorFilterRuleOut.TrafficMirrorFilterRule == nil ||
		aws.ToString(modifyTrafficMirrorFilterRuleOut.TrafficMirrorFilterRule.TrafficMirrorFilterRuleId) != trafficMirrorFilterRuleID {
		t.Fatalf("unexpected modify traffic mirror filter rule output: %#v", modifyTrafficMirrorFilterRuleOut.TrafficMirrorFilterRule)
	}

	createTrafficMirrorTargetOut, err := client.CreateTrafficMirrorTarget(ctx, &awsec2.CreateTrafficMirrorTargetInput{
		NetworkInterfaceId: aws.String("eni-stage1300001"),
	})
	if err != nil || createTrafficMirrorTargetOut.TrafficMirrorTarget == nil || createTrafficMirrorTargetOut.TrafficMirrorTarget.TrafficMirrorTargetId == nil {
		t.Fatalf("create traffic mirror target: %v", err)
	}
	trafficMirrorTargetID := aws.ToString(createTrafficMirrorTargetOut.TrafficMirrorTarget.TrafficMirrorTargetId)

	createTrafficMirrorSessionOut, err := client.CreateTrafficMirrorSession(ctx, &awsec2.CreateTrafficMirrorSessionInput{
		NetworkInterfaceId:    aws.String("eni-stage1300002"),
		SessionNumber:         aws.Int32(1),
		TrafficMirrorFilterId: aws.String(trafficMirrorFilterID),
		TrafficMirrorTargetId: aws.String(trafficMirrorTargetID),
	})
	if err != nil || createTrafficMirrorSessionOut.TrafficMirrorSession == nil || createTrafficMirrorSessionOut.TrafficMirrorSession.TrafficMirrorSessionId == nil {
		t.Fatalf("create traffic mirror session: %v", err)
	}
	trafficMirrorSessionID := aws.ToString(createTrafficMirrorSessionOut.TrafficMirrorSession.TrafficMirrorSessionId)

	modifyTrafficMirrorSessionOut, err := client.ModifyTrafficMirrorSession(ctx, &awsec2.ModifyTrafficMirrorSessionInput{
		TrafficMirrorSessionId: aws.String(trafficMirrorSessionID),
		PacketLength:           aws.Int32(128),
		Description:            aws.String("stage130 session updated"),
	})
	if err != nil {
		t.Fatalf("modify traffic mirror session: %v", err)
	}
	if modifyTrafficMirrorSessionOut.TrafficMirrorSession == nil ||
		aws.ToString(modifyTrafficMirrorSessionOut.TrafficMirrorSession.TrafficMirrorSessionId) != trafficMirrorSessionID {
		t.Fatalf("unexpected modify traffic mirror session output: %#v", modifyTrafficMirrorSessionOut.TrafficMirrorSession)
	}

	createIpamOut, err := client.CreateIpam(ctx, &awsec2.CreateIpamInput{
		OperatingRegions: []awsec2types.AddIpamOperatingRegion{
			{RegionName: aws.String("us-east-1")},
		},
	})
	if err != nil || createIpamOut.Ipam == nil || createIpamOut.Ipam.IpamId == nil {
		t.Fatalf("create ipam: %v", err)
	}
	ipamID := aws.ToString(createIpamOut.Ipam.IpamId)

	createIpamScopeOut, err := client.CreateIpamScope(ctx, &awsec2.CreateIpamScopeInput{
		IpamId: aws.String(ipamID),
	})
	if err != nil || createIpamScopeOut.IpamScope == nil || createIpamScopeOut.IpamScope.IpamScopeId == nil {
		t.Fatalf("create ipam scope: %v", err)
	}
	ipamScopeID := aws.ToString(createIpamScopeOut.IpamScope.IpamScopeId)

	createIpamPoolOut, err := client.CreateIpamPool(ctx, &awsec2.CreateIpamPoolInput{
		AddressFamily: awsec2types.AddressFamilyIpv4,
		IpamScopeId:   aws.String(ipamScopeID),
	})
	if err != nil || createIpamPoolOut.IpamPool == nil || createIpamPoolOut.IpamPool.IpamPoolId == nil {
		t.Fatalf("create ipam pool: %v", err)
	}
	ipamPoolID := aws.ToString(createIpamPoolOut.IpamPool.IpamPoolId)

	provisionByoipCidrOut, err := client.ProvisionByoipCidr(ctx, &awsec2.ProvisionByoipCidrInput{
		Cidr: aws.String("203.0.113.0/24"),
	})
	if err != nil {
		t.Fatalf("provision byoip cidr: %v", err)
	}
	if provisionByoipCidrOut.ByoipCidr == nil || aws.ToString(provisionByoipCidrOut.ByoipCidr.Cidr) != "203.0.113.0/24" {
		t.Fatalf("unexpected provision byoip cidr output: %#v", provisionByoipCidrOut.ByoipCidr)
	}

	moveByoipCidrToIpamOut, err := client.MoveByoipCidrToIpam(ctx, &awsec2.MoveByoipCidrToIpamInput{
		Cidr:          aws.String("203.0.113.0/24"),
		IpamPoolId:    aws.String(ipamPoolID),
		IpamPoolOwner: aws.String("123456789012"),
	})
	if err != nil {
		t.Fatalf("move byoip cidr to ipam: %v", err)
	}
	if moveByoipCidrToIpamOut.ByoipCidr == nil || aws.ToString(moveByoipCidrToIpamOut.ByoipCidr.Cidr) != "203.0.113.0/24" {
		t.Fatalf("unexpected move byoip cidr to ipam output: %#v", moveByoipCidrToIpamOut.ByoipCidr)
	}

	createSourceCapacityReservationOut, err := client.CreateCapacityReservation(ctx, &awsec2.CreateCapacityReservationInput{
		AvailabilityZone: aws.String("us-east-1a"),
		InstanceCount:    aws.Int32(3),
		InstancePlatform: awsec2types.CapacityReservationInstancePlatform("Linux/UNIX"),
		InstanceType:     aws.String("m5.large"),
	})
	if err != nil || createSourceCapacityReservationOut.CapacityReservation == nil || createSourceCapacityReservationOut.CapacityReservation.CapacityReservationId == nil {
		t.Fatalf("create source capacity reservation: %v", err)
	}
	sourceCapacityReservationID := aws.ToString(createSourceCapacityReservationOut.CapacityReservation.CapacityReservationId)

	createDestinationCapacityReservationOut, err := client.CreateCapacityReservation(ctx, &awsec2.CreateCapacityReservationInput{
		AvailabilityZone: aws.String("us-east-1a"),
		InstanceCount:    aws.Int32(1),
		InstancePlatform: awsec2types.CapacityReservationInstancePlatform("Linux/UNIX"),
		InstanceType:     aws.String("m5.large"),
	})
	if err != nil || createDestinationCapacityReservationOut.CapacityReservation == nil || createDestinationCapacityReservationOut.CapacityReservation.CapacityReservationId == nil {
		t.Fatalf("create destination capacity reservation: %v", err)
	}
	destinationCapacityReservationID := aws.ToString(createDestinationCapacityReservationOut.CapacityReservation.CapacityReservationId)

	moveCapacityReservationInstancesOut, err := client.MoveCapacityReservationInstances(ctx, &awsec2.MoveCapacityReservationInstancesInput{
		DestinationCapacityReservationId: aws.String(destinationCapacityReservationID),
		InstanceCount:                    aws.Int32(2),
		SourceCapacityReservationId:      aws.String(sourceCapacityReservationID),
	})
	if err != nil {
		t.Fatalf("move capacity reservation instances: %v", err)
	}
	if aws.ToInt32(moveCapacityReservationInstancesOut.InstanceCount) != 2 {
		t.Fatalf("unexpected moved instance count: %#v", moveCapacityReservationInstancesOut.InstanceCount)
	}
	if moveCapacityReservationInstancesOut.SourceCapacityReservation == nil ||
		aws.ToString(moveCapacityReservationInstancesOut.SourceCapacityReservation.CapacityReservationId) != sourceCapacityReservationID {
		t.Fatalf("unexpected source capacity reservation output: %#v", moveCapacityReservationInstancesOut.SourceCapacityReservation)
	}
	if moveCapacityReservationInstancesOut.DestinationCapacityReservation == nil ||
		aws.ToString(moveCapacityReservationInstancesOut.DestinationCapacityReservation.CapacityReservationId) != destinationCapacityReservationID {
		t.Fatalf("unexpected destination capacity reservation output: %#v", moveCapacityReservationInstancesOut.DestinationCapacityReservation)
	}
}

func TestEC2Stage130ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"ModifyReservedInstances",
		"ModifySnapshotAttribute",
		"ModifySnapshotTier",
		"ModifySpotFleetRequest",
		"ModifyTrafficMirrorFilterNetworkServices",
		"ModifyTrafficMirrorFilterRule",
		"ModifyTrafficMirrorSession",
		"MoveByoipCidrToIpam",
		"MoveCapacityReservationInstances",
		"ProvisionByoipCidr",
	}

	paramsByAction := map[string]map[string]string{
		"ModifyReservedInstances": {
			"ReservedInstancesId.1": "ri-00000000130",
			"ReservedInstancesConfigurationSetItemType.1.InstanceCount": "1",
		},
		"ModifySnapshotAttribute": {
			"SnapshotId":    "snap-00000000130",
			"OperationType": "add",
			"UserGroup.1":   "all",
			"Attribute":     "createVolumePermission",
		},
		"ModifySnapshotTier": {
			"SnapshotId":  "snap-00000000130",
			"StorageTier": "archive",
		},
		"ModifySpotFleetRequest": {
			"SpotFleetRequestId": "sfr-00000000130",
		},
		"ModifyTrafficMirrorFilterNetworkServices": {
			"TrafficMirrorFilterId": "tmf-00000000130",
		},
		"ModifyTrafficMirrorFilterRule": {
			"TrafficMirrorFilterRuleId": "tmfr-00000000130",
		},
		"ModifyTrafficMirrorSession": {
			"TrafficMirrorSessionId": "tms-00000000130",
		},
		"MoveByoipCidrToIpam": {
			"Cidr":          "203.0.113.0/24",
			"IpamPoolId":    "ipam-pool-00000000130",
			"IpamPoolOwner": "123456789012",
		},
		"MoveCapacityReservationInstances": {
			"DestinationCapacityReservationId": "cr-00000000130",
			"InstanceCount":                    "1",
			"SourceCapacityReservationId":      "cr-00000000131",
		},
		"ProvisionByoipCidr": {
			"Cidr": "203.0.114.0/24",
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
