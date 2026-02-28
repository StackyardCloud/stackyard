package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awsec2 "github.com/aws/aws-sdk-go-v2/service/ec2"
	awsec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

func TestEC2Stage132SDKLifecycle(t *testing.T) {
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

	createCapacityReservationOut, err := client.CreateCapacityReservation(ctx, &awsec2.CreateCapacityReservationInput{
		AvailabilityZone: aws.String("us-east-1a"),
		InstanceCount:    aws.Int32(1),
		InstancePlatform: awsec2types.CapacityReservationInstancePlatform("Linux/UNIX"),
		InstanceType:     aws.String("m5.large"),
	})
	if err != nil || createCapacityReservationOut.CapacityReservation == nil || createCapacityReservationOut.CapacityReservation.CapacityReservationId == nil {
		t.Fatalf("create capacity reservation: %v", err)
	}
	capacityReservationID := aws.ToString(createCapacityReservationOut.CapacityReservation.CapacityReservationId)

	rejectCapacityReservationBillingOwnershipOut, err := client.RejectCapacityReservationBillingOwnership(ctx, &awsec2.RejectCapacityReservationBillingOwnershipInput{
		CapacityReservationId: aws.String(capacityReservationID),
	})
	if err != nil {
		t.Fatalf("reject capacity reservation billing ownership: %v", err)
	}
	if !aws.ToBool(rejectCapacityReservationBillingOwnershipOut.Return) {
		t.Fatalf("expected reject capacity reservation billing ownership return true")
	}

	allocateHostsOut, err := client.AllocateHosts(ctx, &awsec2.AllocateHostsInput{
		Quantity: aws.Int32(1),
	})
	if err != nil || len(allocateHostsOut.HostIds) == 0 {
		t.Fatalf("allocate hosts: %v", err)
	}
	hostID := allocateHostsOut.HostIds[0]

	releaseHostsOut, err := client.ReleaseHosts(ctx, &awsec2.ReleaseHostsInput{
		HostIds: []string{hostID},
	})
	if err != nil {
		t.Fatalf("release hosts: %v", err)
	}
	if len(releaseHostsOut.Successful) == 0 || releaseHostsOut.Successful[0] != hostID {
		t.Fatalf("unexpected release hosts output: %#v", releaseHostsOut)
	}

	createIpamOut, err := client.CreateIpam(ctx, &awsec2.CreateIpamInput{})
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

	allocateIpamPoolCidrOut, err := client.AllocateIpamPoolCidr(ctx, &awsec2.AllocateIpamPoolCidrInput{
		IpamPoolId: aws.String(ipamPoolID),
		Cidr:       aws.String("10.132.0.0/24"),
	})
	if err != nil || allocateIpamPoolCidrOut.IpamPoolAllocation == nil || allocateIpamPoolCidrOut.IpamPoolAllocation.IpamPoolAllocationId == nil {
		t.Fatalf("allocate ipam pool cidr: %v", err)
	}
	ipamPoolAllocationID := aws.ToString(allocateIpamPoolCidrOut.IpamPoolAllocation.IpamPoolAllocationId)

	releaseIpamPoolAllocationOut, err := client.ReleaseIpamPoolAllocation(ctx, &awsec2.ReleaseIpamPoolAllocationInput{
		IpamPoolId:           aws.String(ipamPoolID),
		Cidr:                 aws.String("10.132.0.0/24"),
		IpamPoolAllocationId: aws.String(ipamPoolAllocationID),
	})
	if err != nil {
		t.Fatalf("release ipam pool allocation: %v", err)
	}
	if !aws.ToBool(releaseIpamPoolAllocationOut.Success) {
		t.Fatalf("expected release ipam pool allocation success true")
	}

	runInstancesOut, err := client.RunInstances(ctx, &awsec2.RunInstancesInput{
		ImageId:      aws.String("ami-00000000132"),
		InstanceType: awsec2types.InstanceTypeT3Micro,
		MinCount:     aws.Int32(1),
		MaxCount:     aws.Int32(1),
	})
	if err != nil || len(runInstancesOut.Instances) == 0 || runInstancesOut.Instances[0].InstanceId == nil {
		t.Fatalf("run instances: %v", err)
	}
	instanceID := aws.ToString(runInstancesOut.Instances[0].InstanceId)

	if _, err := client.ReportInstanceStatus(ctx, &awsec2.ReportInstanceStatusInput{
		Instances:   []string{instanceID},
		Status:      awsec2types.ReportStatusTypeImpaired,
		ReasonCodes: []awsec2types.ReportInstanceReasonCodes{awsec2types.ReportInstanceReasonCodesOther},
		StartTime:   aws.Time(time.Now().UTC().Add(-5 * time.Minute)),
		EndTime:     aws.Time(time.Now().UTC().Add(5 * time.Minute)),
	}); err != nil {
		t.Fatalf("report instance status: %v", err)
	}

	requestSpotFleetOut, err := client.RequestSpotFleet(ctx, &awsec2.RequestSpotFleetInput{
		SpotFleetRequestConfig: &awsec2types.SpotFleetRequestConfigData{
			IamFleetRole:   aws.String("arn:aws:iam::123456789012:role/aws-ec2-spot-fleet-tagging-role"),
			TargetCapacity: aws.Int32(1),
		},
	})
	if err != nil {
		t.Fatalf("request spot fleet: %v", err)
	}
	if aws.ToString(requestSpotFleetOut.SpotFleetRequestId) == "" {
		t.Fatalf("expected spot fleet request id")
	}

	requestSpotInstancesOut, err := client.RequestSpotInstances(ctx, &awsec2.RequestSpotInstancesInput{
		InstanceCount: aws.Int32(1),
		Type:          awsec2types.SpotInstanceTypeOneTime,
	})
	if err != nil {
		t.Fatalf("request spot instances: %v", err)
	}
	if len(requestSpotInstancesOut.SpotInstanceRequests) == 0 || requestSpotInstancesOut.SpotInstanceRequests[0].SpotInstanceRequestId == nil {
		t.Fatalf("unexpected request spot instances output: %#v", requestSpotInstancesOut.SpotInstanceRequests)
	}

	createFpgaImageOut, err := client.CreateFpgaImage(ctx, &awsec2.CreateFpgaImageInput{
		InputStorageLocation: &awsec2types.StorageLocation{
			Bucket: aws.String("stage132-bucket"),
			Key:    aws.String("stage132/input.xclbin"),
		},
	})
	if err != nil || createFpgaImageOut.FpgaImageId == nil {
		t.Fatalf("create fpga image: %v", err)
	}
	fpgaImageID := aws.ToString(createFpgaImageOut.FpgaImageId)

	resetFpgaImageAttributeOut, err := client.ResetFpgaImageAttribute(ctx, &awsec2.ResetFpgaImageAttributeInput{
		FpgaImageId: aws.String(fpgaImageID),
		Attribute:   awsec2types.ResetFpgaImageAttributeNameLoadPermission,
	})
	if err != nil {
		t.Fatalf("reset fpga image attribute: %v", err)
	}
	if !aws.ToBool(resetFpgaImageAttributeOut.Return) {
		t.Fatalf("expected reset fpga image attribute return true")
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

	if _, err := client.ResetSnapshotAttribute(ctx, &awsec2.ResetSnapshotAttributeInput{
		SnapshotId: aws.String(snapshotID),
		Attribute:  awsec2types.SnapshotAttributeNameCreateVolumePermission,
	}); err != nil {
		t.Fatalf("reset snapshot attribute: %v", err)
	}

	registerImageOut, err := client.RegisterImage(ctx, &awsec2.RegisterImageInput{
		Name: aws.String("stage132-image"),
	})
	if err != nil || registerImageOut.ImageId == nil {
		t.Fatalf("register image: %v", err)
	}
	imageID := aws.ToString(registerImageOut.ImageId)

	restoreImageFromRecycleBinOut, err := client.RestoreImageFromRecycleBin(ctx, &awsec2.RestoreImageFromRecycleBinInput{
		ImageId: aws.String(imageID),
	})
	if err != nil {
		t.Fatalf("restore image from recycle bin: %v", err)
	}
	if !aws.ToBool(restoreImageFromRecycleBinOut.Return) {
		t.Fatalf("expected restore image from recycle bin return true")
	}

	createManagedPrefixListOut, err := client.CreateManagedPrefixList(ctx, &awsec2.CreateManagedPrefixListInput{
		AddressFamily:  aws.String("ipv4"),
		MaxEntries:     aws.Int32(10),
		PrefixListName: aws.String("stage132-prefix-list"),
		Entries: []awsec2types.AddPrefixListEntry{
			{Cidr: aws.String("10.132.0.0/16"), Description: aws.String("stage132")},
		},
	})
	if err != nil || createManagedPrefixListOut.PrefixList == nil || createManagedPrefixListOut.PrefixList.PrefixListId == nil || createManagedPrefixListOut.PrefixList.Version == nil {
		t.Fatalf("create managed prefix list: %v", err)
	}
	prefixListID := aws.ToString(createManagedPrefixListOut.PrefixList.PrefixListId)
	initialVersion := aws.ToInt64(createManagedPrefixListOut.PrefixList.Version)

	modifyManagedPrefixListOut, err := client.ModifyManagedPrefixList(ctx, &awsec2.ModifyManagedPrefixListInput{
		PrefixListId:   aws.String(prefixListID),
		CurrentVersion: aws.Int64(initialVersion),
		AddEntries: []awsec2types.AddPrefixListEntry{
			{Cidr: aws.String("10.132.1.0/24"), Description: aws.String("stage132-v2")},
		},
	})
	if err != nil || modifyManagedPrefixListOut.PrefixList == nil || modifyManagedPrefixListOut.PrefixList.Version == nil {
		t.Fatalf("modify managed prefix list: %v", err)
	}
	currentVersion := aws.ToInt64(modifyManagedPrefixListOut.PrefixList.Version)

	restoreManagedPrefixListVersionOut, err := client.RestoreManagedPrefixListVersion(ctx, &awsec2.RestoreManagedPrefixListVersionInput{
		PrefixListId:    aws.String(prefixListID),
		PreviousVersion: aws.Int64(initialVersion),
		CurrentVersion:  aws.Int64(currentVersion),
	})
	if err != nil {
		t.Fatalf("restore managed prefix list version: %v", err)
	}
	if restoreManagedPrefixListVersionOut.PrefixList == nil || aws.ToString(restoreManagedPrefixListVersionOut.PrefixList.PrefixListId) != prefixListID {
		t.Fatalf("unexpected restore managed prefix list version output: %#v", restoreManagedPrefixListVersionOut.PrefixList)
	}
}

func TestEC2Stage132ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"RejectCapacityReservationBillingOwnership",
		"ReleaseHosts",
		"ReleaseIpamPoolAllocation",
		"ReportInstanceStatus",
		"RequestSpotFleet",
		"RequestSpotInstances",
		"ResetFpgaImageAttribute",
		"ResetSnapshotAttribute",
		"RestoreImageFromRecycleBin",
		"RestoreManagedPrefixListVersion",
	}

	paramsByAction := map[string]map[string]string{
		"RejectCapacityReservationBillingOwnership": {
			"CapacityReservationId": "cr-00000000132",
		},
		"ReleaseHosts": {
			"HostId.1": "h-00000000132",
		},
		"ReleaseIpamPoolAllocation": {
			"IpamPoolId":           "ipam-pool-00000000132",
			"Cidr":                 "10.132.0.0/24",
			"IpamPoolAllocationId": "ipam-pool-alloc-00000000132",
		},
		"ReportInstanceStatus": {
			"InstanceId.1": "i-00000000132",
			"Status":       "impaired",
			"ReasonCode.1": "other",
		},
		"RequestSpotFleet": {
			"SpotFleetRequestConfig.IamFleetRole":   "arn:aws:iam::123456789012:role/aws-ec2-spot-fleet-tagging-role",
			"SpotFleetRequestConfig.TargetCapacity": "1",
		},
		"RequestSpotInstances": {
			"InstanceCount": "1",
			"Type":          "one-time",
		},
		"ResetFpgaImageAttribute": {
			"FpgaImageId": "afi-00000000132",
			"Attribute":   "loadPermission",
		},
		"ResetSnapshotAttribute": {
			"SnapshotId": "snap-00000000132",
			"Attribute":  "createVolumePermission",
		},
		"RestoreImageFromRecycleBin": {
			"ImageId": "ami-00000000132",
		},
		"RestoreManagedPrefixListVersion": {
			"PrefixListId":    "pl-00000000132",
			"PreviousVersion": "1",
			"CurrentVersion":  "2",
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
