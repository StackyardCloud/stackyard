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

func TestEC2Stage127SDKLifecycle(t *testing.T) {
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

	createVolumeOut, err := client.CreateVolume(ctx, &awsec2.CreateVolumeInput{
		AvailabilityZone: aws.String("us-east-1a"),
		Size:             aws.Int32(8),
	})
	if err != nil || createVolumeOut.VolumeId == nil {
		t.Fatalf("create volume: %v", err)
	}
	createSnapshotOut, err := client.CreateSnapshot(ctx, &awsec2.CreateSnapshotInput{
		VolumeId:    createVolumeOut.VolumeId,
		Description: aws.String("stage127 snapshot"),
	})
	if err != nil || createSnapshotOut.SnapshotId == nil {
		t.Fatalf("create snapshot: %v", err)
	}
	snapshotID := aws.ToString(createSnapshotOut.SnapshotId)

	listSnapshotsInRecycleBinOut, err := client.ListSnapshotsInRecycleBin(ctx, &awsec2.ListSnapshotsInRecycleBinInput{
		SnapshotIds: []string{snapshotID},
		MaxResults:  aws.Int32(10),
	})
	if err != nil {
		t.Fatalf("list snapshots in recycle bin: %v", err)
	}
	if len(listSnapshotsInRecycleBinOut.Snapshots) == 0 || aws.ToString(listSnapshotsInRecycleBinOut.Snapshots[0].SnapshotId) != snapshotID {
		t.Fatalf("unexpected snapshots in recycle bin output: %#v", listSnapshotsInRecycleBinOut.Snapshots)
	}

	lockSnapshotOut, err := client.LockSnapshot(ctx, &awsec2.LockSnapshotInput{
		SnapshotId:   aws.String(snapshotID),
		LockMode:     awsec2types.LockModeCompliance,
		LockDuration: aws.Int32(2),
	})
	if err != nil {
		t.Fatalf("lock snapshot: %v", err)
	}
	if aws.ToString(lockSnapshotOut.SnapshotId) != snapshotID {
		t.Fatalf("unexpected lock snapshot id: %q", aws.ToString(lockSnapshotOut.SnapshotId))
	}
	if lockSnapshotOut.LockState == "" {
		t.Fatalf("expected lock state")
	}

	modifyAvailabilityZoneGroupOut, err := client.ModifyAvailabilityZoneGroup(ctx, &awsec2.ModifyAvailabilityZoneGroupInput{
		GroupName:   aws.String("us-east-1-lax-1"),
		OptInStatus: awsec2types.ModifyAvailabilityZoneOptInStatusOptedIn,
	})
	if err != nil {
		t.Fatalf("modify availability zone group: %v", err)
	}
	if modifyAvailabilityZoneGroupOut.Return == nil || !aws.ToBool(modifyAvailabilityZoneGroupOut.Return) {
		t.Fatalf("expected modify availability zone group return true")
	}

	createCapacityReservationOut, err := client.CreateCapacityReservation(ctx, &awsec2.CreateCapacityReservationInput{
		AvailabilityZone:      aws.String("us-east-1a"),
		InstanceCount:         aws.Int32(1),
		InstancePlatform:      awsec2types.CapacityReservationInstancePlatformLinuxUnix,
		InstanceType:          aws.String("m5.large"),
		InstanceMatchCriteria: awsec2types.InstanceMatchCriteriaOpen,
	})
	if err != nil || createCapacityReservationOut.CapacityReservation == nil || createCapacityReservationOut.CapacityReservation.CapacityReservationId == nil {
		t.Fatalf("create capacity reservation: %v", err)
	}
	capacityReservationID := aws.ToString(createCapacityReservationOut.CapacityReservation.CapacityReservationId)

	modifyCapacityReservationOut, err := client.ModifyCapacityReservation(ctx, &awsec2.ModifyCapacityReservationInput{
		CapacityReservationId: aws.String(capacityReservationID),
		InstanceCount:         aws.Int32(2),
		InstanceMatchCriteria: awsec2types.InstanceMatchCriteriaTargeted,
	})
	if err != nil {
		t.Fatalf("modify capacity reservation: %v", err)
	}
	if modifyCapacityReservationOut.Return == nil || !aws.ToBool(modifyCapacityReservationOut.Return) {
		t.Fatalf("expected modify capacity reservation return true")
	}

	describeCapacityReservationsOut, err := client.DescribeCapacityReservations(ctx, &awsec2.DescribeCapacityReservationsInput{
		CapacityReservationIds: []string{capacityReservationID},
	})
	if err != nil {
		t.Fatalf("describe capacity reservations: %v", err)
	}
	if len(describeCapacityReservationsOut.CapacityReservations) != 1 || aws.ToInt32(describeCapacityReservationsOut.CapacityReservations[0].TotalInstanceCount) != 2 {
		t.Fatalf("unexpected described capacity reservation output: %#v", describeCapacityReservationsOut.CapacityReservations)
	}

	createCapacityReservationFleetOut, err := client.CreateCapacityReservationFleet(ctx, &awsec2.CreateCapacityReservationFleetInput{
		InstanceTypeSpecifications: []awsec2types.ReservationFleetInstanceSpecification{
			{
				AvailabilityZone: aws.String("us-east-1a"),
				InstancePlatform: awsec2types.CapacityReservationInstancePlatformLinuxUnix,
				InstanceType:     awsec2types.InstanceTypeM5Large,
			},
		},
		TotalTargetCapacity: aws.Int32(1),
	})
	if err != nil || createCapacityReservationFleetOut.CapacityReservationFleetId == nil {
		t.Fatalf("create capacity reservation fleet: %v", err)
	}
	capacityReservationFleetID := aws.ToString(createCapacityReservationFleetOut.CapacityReservationFleetId)

	modifyCapacityReservationFleetOut, err := client.ModifyCapacityReservationFleet(ctx, &awsec2.ModifyCapacityReservationFleetInput{
		CapacityReservationFleetId: aws.String(capacityReservationFleetID),
		TotalTargetCapacity:        aws.Int32(2),
	})
	if err != nil {
		t.Fatalf("modify capacity reservation fleet: %v", err)
	}
	if modifyCapacityReservationFleetOut.Return == nil || !aws.ToBool(modifyCapacityReservationFleetOut.Return) {
		t.Fatalf("expected modify capacity reservation fleet return true")
	}

	modifyDefaultCreditSpecificationOut, err := client.ModifyDefaultCreditSpecification(ctx, &awsec2.ModifyDefaultCreditSpecificationInput{
		InstanceFamily: awsec2types.UnlimitedSupportedInstanceFamilyT3,
		CpuCredits:     aws.String("unlimited"),
	})
	if err != nil {
		t.Fatalf("modify default credit specification: %v", err)
	}
	if modifyDefaultCreditSpecificationOut.InstanceFamilyCreditSpecification == nil ||
		aws.ToString(modifyDefaultCreditSpecificationOut.InstanceFamilyCreditSpecification.CpuCredits) != "unlimited" {
		t.Fatalf("unexpected modify default credit specification output: %#v", modifyDefaultCreditSpecificationOut.InstanceFamilyCreditSpecification)
	}

	getDefaultCreditSpecificationOut, err := client.GetDefaultCreditSpecification(ctx, &awsec2.GetDefaultCreditSpecificationInput{
		InstanceFamily: awsec2types.UnlimitedSupportedInstanceFamilyT3,
	})
	if err != nil {
		t.Fatalf("get default credit specification: %v", err)
	}
	if getDefaultCreditSpecificationOut.InstanceFamilyCreditSpecification == nil ||
		aws.ToString(getDefaultCreditSpecificationOut.InstanceFamilyCreditSpecification.CpuCredits) != "unlimited" {
		t.Fatalf("unexpected get default credit specification output: %#v", getDefaultCreditSpecificationOut.InstanceFamilyCreditSpecification)
	}

	createFleetOut, err := client.CreateFleet(ctx, &awsec2.CreateFleetInput{
		LaunchTemplateConfigs: []awsec2types.FleetLaunchTemplateConfigRequest{
			{
				LaunchTemplateSpecification: &awsec2types.FleetLaunchTemplateSpecificationRequest{
					LaunchTemplateId: aws.String("lt-00000000000000127"),
					Version:          aws.String("1"),
				},
			},
		},
		TargetCapacitySpecification: &awsec2types.TargetCapacitySpecificationRequest{
			TotalTargetCapacity: aws.Int32(1),
		},
	})
	if err != nil || createFleetOut.FleetId == nil {
		t.Fatalf("create fleet: %v", err)
	}

	modifyFleetOut, err := client.ModifyFleet(ctx, &awsec2.ModifyFleetInput{
		FleetId: createFleetOut.FleetId,
		TargetCapacitySpecification: &awsec2types.TargetCapacitySpecificationRequest{
			TotalTargetCapacity: aws.Int32(2),
		},
	})
	if err != nil {
		t.Fatalf("modify fleet: %v", err)
	}
	if modifyFleetOut.Return == nil || !aws.ToBool(modifyFleetOut.Return) {
		t.Fatalf("expected modify fleet return true")
	}

	createFpgaImageOut, err := client.CreateFpgaImage(ctx, &awsec2.CreateFpgaImageInput{
		InputStorageLocation: &awsec2types.StorageLocation{
			Bucket: aws.String("stage127-bucket"),
			Key:    aws.String("stage127/input.xclbin"),
		},
	})
	if err != nil || createFpgaImageOut.FpgaImageId == nil {
		t.Fatalf("create fpga image: %v", err)
	}
	fpgaImageID := aws.ToString(createFpgaImageOut.FpgaImageId)

	modifyFpgaImageAttributeOut, err := client.ModifyFpgaImageAttribute(ctx, &awsec2.ModifyFpgaImageAttributeInput{
		FpgaImageId:   aws.String(fpgaImageID),
		Attribute:     awsec2types.FpgaImageAttributeNameName,
		OperationType: awsec2types.OperationTypeAdd,
		Name:          aws.String("stage127-fpga"),
		UserIds:       []string{"123456789012"},
	})
	if err != nil {
		t.Fatalf("modify fpga image attribute: %v", err)
	}
	if modifyFpgaImageAttributeOut.FpgaImageAttribute == nil ||
		aws.ToString(modifyFpgaImageAttributeOut.FpgaImageAttribute.FpgaImageId) != fpgaImageID {
		t.Fatalf("unexpected modify fpga image attribute output: %#v", modifyFpgaImageAttributeOut.FpgaImageAttribute)
	}

	allocateHostsOut, err := client.AllocateHosts(ctx, &awsec2.AllocateHostsInput{
		AvailabilityZone: aws.String("us-east-1a"),
		InstanceType:     aws.String("m5.large"),
		Quantity:         aws.Int32(1),
	})
	if err != nil || len(allocateHostsOut.HostIds) != 1 {
		t.Fatalf("allocate hosts: %v", err)
	}

	modifyHostsOut, err := client.ModifyHosts(ctx, &awsec2.ModifyHostsInput{
		HostIds:         allocateHostsOut.HostIds,
		AutoPlacement:   awsec2types.AutoPlacementOn,
		HostRecovery:    awsec2types.HostRecoveryOn,
		HostMaintenance: awsec2types.HostMaintenanceOn,
	})
	if err != nil {
		t.Fatalf("modify hosts: %v", err)
	}
	if len(modifyHostsOut.Successful) != 1 || modifyHostsOut.Successful[0] != allocateHostsOut.HostIds[0] {
		t.Fatalf("unexpected modify hosts successful output: %#v", modifyHostsOut.Successful)
	}
	if len(modifyHostsOut.Unsuccessful) != 0 {
		t.Fatalf("expected zero unsuccessful modify hosts items, got %#v", modifyHostsOut.Unsuccessful)
	}

	runInstancesOut, err := client.RunInstances(ctx, &awsec2.RunInstancesInput{
		ImageId:  aws.String("ami-00000000000000127"),
		MinCount: aws.Int32(1),
		MaxCount: aws.Int32(1),
		SubnetId: aws.String("subnet-00000001"),
	})
	if err != nil || len(runInstancesOut.Instances) != 1 || runInstancesOut.Instances[0].InstanceId == nil {
		t.Fatalf("run instances: %v", err)
	}
	instanceID := aws.ToString(runInstancesOut.Instances[0].InstanceId)

	modifyInstanceCapacityReservationAttributesOut, err := client.ModifyInstanceCapacityReservationAttributes(ctx, &awsec2.ModifyInstanceCapacityReservationAttributesInput{
		InstanceId: aws.String(instanceID),
		CapacityReservationSpecification: &awsec2types.CapacityReservationSpecification{
			CapacityReservationPreference: awsec2types.CapacityReservationPreferenceOpen,
		},
	})
	if err != nil {
		t.Fatalf("modify instance capacity reservation attributes: %v", err)
	}
	if modifyInstanceCapacityReservationAttributesOut.Return == nil || !aws.ToBool(modifyInstanceCapacityReservationAttributesOut.Return) {
		t.Fatalf("expected modify instance capacity reservation attributes return true")
	}
}

func TestEC2Stage127ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"ListSnapshotsInRecycleBin",
		"LockSnapshot",
		"ModifyAvailabilityZoneGroup",
		"ModifyCapacityReservation",
		"ModifyCapacityReservationFleet",
		"ModifyDefaultCreditSpecification",
		"ModifyFleet",
		"ModifyFpgaImageAttribute",
		"ModifyHosts",
		"ModifyInstanceCapacityReservationAttributes",
	}

	paramsByAction := map[string]map[string]string{
		"ListSnapshotsInRecycleBin": {},
		"LockSnapshot": {
			"SnapshotId": "snap-00000000000000127",
			"LockMode":   "compliance",
		},
		"ModifyAvailabilityZoneGroup": {
			"GroupName":   "us-east-1-lax-1",
			"OptInStatus": "opted-in",
		},
		"ModifyCapacityReservation": {
			"CapacityReservationId": "cr-00000000000000127",
		},
		"ModifyCapacityReservationFleet": {
			"CapacityReservationFleetId": "crf-00000000000000127",
		},
		"ModifyDefaultCreditSpecification": {
			"InstanceFamily": "t3",
			"CpuCredits":     "unlimited",
		},
		"ModifyFleet": {
			"FleetId": "fleet-00000000000000127",
		},
		"ModifyFpgaImageAttribute": {
			"FpgaImageId": "afi-00000000000000127",
		},
		"ModifyHosts": {
			"HostId.1": "h-00000000000000127",
		},
		"ModifyInstanceCapacityReservationAttributes": {
			"InstanceId": "i-00000000000000127",
			"CapacityReservationSpecification.CapacityReservationPreference": "open",
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
