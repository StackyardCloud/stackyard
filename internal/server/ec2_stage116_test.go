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

func TestEC2Stage116SDKLifecycle(t *testing.T) {
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
		ImageId:      aws.String("ami-stage116"),
		InstanceType: awsec2types.InstanceTypeT3Micro,
		MinCount:     aws.Int32(1),
		MaxCount:     aws.Int32(1),
	})
	if err != nil || len(runOut.Instances) == 0 || runOut.Instances[0].InstanceId == nil {
		t.Fatalf("run instances: %v", err)
	}
	instanceID := aws.ToString(runOut.Instances[0].InstanceId)

	createImageOut, err := client.CreateImage(ctx, &awsec2.CreateImageInput{
		InstanceId: aws.String(instanceID),
		Name:       aws.String("stage116-image"),
	})
	if err != nil || createImageOut.ImageId == nil {
		t.Fatalf("create image: %v", err)
	}
	imageID := aws.ToString(createImageOut.ImageId)

	if _, err := client.EnableFastLaunch(ctx, &awsec2.EnableFastLaunchInput{
		ImageId:      aws.String(imageID),
		ResourceType: aws.String("snapshot"),
	}); err != nil {
		t.Fatalf("enable fast launch: %v", err)
	}

	describeFastLaunchImagesOut, err := client.DescribeFastLaunchImages(ctx, &awsec2.DescribeFastLaunchImagesInput{
		ImageIds:   []string{imageID},
		MaxResults: aws.Int32(10),
	})
	if err != nil {
		t.Fatalf("describe fast launch images: %v", err)
	}
	if len(describeFastLaunchImagesOut.FastLaunchImages) != 1 {
		t.Fatalf("expected 1 fast launch image, got %d", len(describeFastLaunchImagesOut.FastLaunchImages))
	}
	if aws.ToString(describeFastLaunchImagesOut.FastLaunchImages[0].ImageId) != imageID {
		t.Fatalf("unexpected fast launch image id: %q", aws.ToString(describeFastLaunchImagesOut.FastLaunchImages[0].ImageId))
	}

	createVolumeOut, err := client.CreateVolume(ctx, &awsec2.CreateVolumeInput{
		AvailabilityZone: aws.String("us-east-1a"),
		Size:             aws.Int32(8),
		VolumeType:       awsec2types.VolumeTypeGp3,
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

	if _, err := client.EnableFastSnapshotRestores(ctx, &awsec2.EnableFastSnapshotRestoresInput{
		SourceSnapshotIds: []string{snapshotID},
		AvailabilityZones: []string{"us-east-1a"},
	}); err != nil {
		t.Fatalf("enable fast snapshot restores: %v", err)
	}

	describeFastSnapshotRestoresOut, err := client.DescribeFastSnapshotRestores(ctx, &awsec2.DescribeFastSnapshotRestoresInput{
		Filters: []awsec2types.Filter{
			{Name: aws.String("snapshot-id"), Values: []string{snapshotID}},
		},
		MaxResults: aws.Int32(10),
	})
	if err != nil {
		t.Fatalf("describe fast snapshot restores: %v", err)
	}
	if len(describeFastSnapshotRestoresOut.FastSnapshotRestores) == 0 {
		t.Fatalf("expected fast snapshot restore entries")
	}

	createFleetOut, err := client.CreateFleet(ctx, &awsec2.CreateFleetInput{
		LaunchTemplateConfigs: []awsec2types.FleetLaunchTemplateConfigRequest{
			{
				LaunchTemplateSpecification: &awsec2types.FleetLaunchTemplateSpecificationRequest{
					LaunchTemplateId: aws.String("lt-00000000000000116"),
					Version:          aws.String("1"),
				},
			},
		},
		TargetCapacitySpecification: &awsec2types.TargetCapacitySpecificationRequest{
			DefaultTargetCapacityType: awsec2types.DefaultTargetCapacityTypeOnDemand,
			TotalTargetCapacity:       aws.Int32(2),
		},
	})
	if err != nil {
		t.Fatalf("create fleet: %v", err)
	}
	fleetID := aws.ToString(createFleetOut.FleetId)
	if fleetID == "" {
		t.Fatalf("expected fleet id")
	}

	describeFleetHistoryOut, err := client.DescribeFleetHistory(ctx, &awsec2.DescribeFleetHistoryInput{
		FleetId:    aws.String(fleetID),
		StartTime:  aws.Time(time.Now().UTC().Add(-1 * time.Hour)),
		MaxResults: aws.Int32(10),
	})
	if err != nil {
		t.Fatalf("describe fleet history: %v", err)
	}
	if aws.ToString(describeFleetHistoryOut.FleetId) != fleetID {
		t.Fatalf("unexpected fleet id in history output: %q", aws.ToString(describeFleetHistoryOut.FleetId))
	}
	if len(describeFleetHistoryOut.HistoryRecords) == 0 {
		t.Fatalf("expected fleet history records")
	}

	describeFleetInstancesOut, err := client.DescribeFleetInstances(ctx, &awsec2.DescribeFleetInstancesInput{
		FleetId:    aws.String(fleetID),
		MaxResults: aws.Int32(10),
	})
	if err != nil {
		t.Fatalf("describe fleet instances: %v", err)
	}
	if aws.ToString(describeFleetInstancesOut.FleetId) != fleetID {
		t.Fatalf("unexpected fleet id in instances output: %q", aws.ToString(describeFleetInstancesOut.FleetId))
	}
	if len(describeFleetInstancesOut.ActiveInstances) == 0 {
		t.Fatalf("expected active instances")
	}

	describeFleetsOut, err := client.DescribeFleets(ctx, &awsec2.DescribeFleetsInput{
		FleetIds:   []string{fleetID},
		MaxResults: aws.Int32(10),
	})
	if err != nil {
		t.Fatalf("describe fleets: %v", err)
	}
	if len(describeFleetsOut.Fleets) != 1 {
		t.Fatalf("expected 1 fleet, got %d", len(describeFleetsOut.Fleets))
	}
	if aws.ToString(describeFleetsOut.Fleets[0].FleetId) != fleetID {
		t.Fatalf("unexpected described fleet id: %q", aws.ToString(describeFleetsOut.Fleets[0].FleetId))
	}

	createFlowLogsOut, err := client.CreateFlowLogs(ctx, &awsec2.CreateFlowLogsInput{
		ResourceIds:  []string{"vpc-00000001"},
		ResourceType: awsec2types.FlowLogsResourceTypeVpc,
		TrafficType:  awsec2types.TrafficTypeAll,
	})
	if err != nil || len(createFlowLogsOut.FlowLogIds) == 0 {
		t.Fatalf("create flow logs: %v", err)
	}
	flowLogID := createFlowLogsOut.FlowLogIds[0]

	describeFlowLogsOut, err := client.DescribeFlowLogs(ctx, &awsec2.DescribeFlowLogsInput{
		FlowLogIds: []string{flowLogID},
		MaxResults: aws.Int32(10),
	})
	if err != nil {
		t.Fatalf("describe flow logs: %v", err)
	}
	if len(describeFlowLogsOut.FlowLogs) != 1 {
		t.Fatalf("expected 1 flow log, got %d", len(describeFlowLogsOut.FlowLogs))
	}
	if aws.ToString(describeFlowLogsOut.FlowLogs[0].FlowLogId) != flowLogID {
		t.Fatalf("unexpected flow log id: %q", aws.ToString(describeFlowLogsOut.FlowLogs[0].FlowLogId))
	}

	createFpgaImageOut, err := client.CreateFpgaImage(ctx, &awsec2.CreateFpgaImageInput{
		Name:        aws.String("stage116-fpga"),
		Description: aws.String("stage116-description"),
		InputStorageLocation: &awsec2types.StorageLocation{
			Bucket: aws.String("stage116-bucket"),
			Key:    aws.String("input.xclbin"),
		},
	})
	if err != nil || createFpgaImageOut.FpgaImageId == nil {
		t.Fatalf("create fpga image: %v", err)
	}
	fpgaImageID := aws.ToString(createFpgaImageOut.FpgaImageId)

	describeFpgaImageAttributeOut, err := client.DescribeFpgaImageAttribute(ctx, &awsec2.DescribeFpgaImageAttributeInput{
		FpgaImageId: createFpgaImageOut.FpgaImageId,
		Attribute:   awsec2types.FpgaImageAttributeNameName,
	})
	if err != nil {
		t.Fatalf("describe fpga image attribute: %v", err)
	}
	if describeFpgaImageAttributeOut.FpgaImageAttribute == nil {
		t.Fatalf("expected fpga image attribute output")
	}
	if aws.ToString(describeFpgaImageAttributeOut.FpgaImageAttribute.FpgaImageId) != fpgaImageID {
		t.Fatalf("unexpected fpga image attribute image id: %q", aws.ToString(describeFpgaImageAttributeOut.FpgaImageAttribute.FpgaImageId))
	}

	describeFpgaImagesOut, err := client.DescribeFpgaImages(ctx, &awsec2.DescribeFpgaImagesInput{
		FpgaImageIds: []string{fpgaImageID},
		MaxResults:   aws.Int32(10),
	})
	if err != nil {
		t.Fatalf("describe fpga images: %v", err)
	}
	if len(describeFpgaImagesOut.FpgaImages) != 1 {
		t.Fatalf("expected 1 fpga image, got %d", len(describeFpgaImagesOut.FpgaImages))
	}
	if aws.ToString(describeFpgaImagesOut.FpgaImages[0].FpgaImageId) != fpgaImageID {
		t.Fatalf("unexpected described fpga image id: %q", aws.ToString(describeFpgaImagesOut.FpgaImages[0].FpgaImageId))
	}

	allocateHostsOut, err := client.AllocateHosts(ctx, &awsec2.AllocateHostsInput{
		AvailabilityZone: aws.String("us-east-1a"),
		InstanceFamily:   aws.String("m5"),
		Quantity:         aws.Int32(1),
	})
	if err != nil || len(allocateHostsOut.HostIds) == 0 {
		t.Fatalf("allocate hosts: %v", err)
	}

	describeHostReservationOfferingsOut, err := client.DescribeHostReservationOfferings(ctx, &awsec2.DescribeHostReservationOfferingsInput{
		MaxResults: aws.Int32(10),
	})
	if err != nil {
		t.Fatalf("describe host reservation offerings: %v", err)
	}
	if len(describeHostReservationOfferingsOut.OfferingSet) == 0 {
		t.Fatalf("expected host reservation offerings")
	}

	describeHostReservationsOut, err := client.DescribeHostReservations(ctx, &awsec2.DescribeHostReservationsInput{
		MaxResults: aws.Int32(10),
	})
	if err != nil {
		t.Fatalf("describe host reservations: %v", err)
	}
	if len(describeHostReservationsOut.HostReservationSet) == 0 {
		t.Fatalf("expected host reservations")
	}
	if len(describeHostReservationsOut.HostReservationSet[0].HostIdSet) == 0 {
		t.Fatalf("expected host reservation host ids")
	}
}

func TestEC2Stage116ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"DescribeFastLaunchImages",
		"DescribeFastSnapshotRestores",
		"DescribeFleetHistory",
		"DescribeFleetInstances",
		"DescribeFleets",
		"DescribeFlowLogs",
		"DescribeFpgaImageAttribute",
		"DescribeFpgaImages",
		"DescribeHostReservationOfferings",
		"DescribeHostReservations",
	}

	paramsByAction := map[string]map[string]string{
		"DescribeFastLaunchImages":         {},
		"DescribeFastSnapshotRestores":     {},
		"DescribeFleetHistory":             {"FleetId": "fleet-00000000116", "StartTime": time.Now().UTC().Format(time.RFC3339)},
		"DescribeFleetInstances":           {"FleetId": "fleet-00000000116"},
		"DescribeFleets":                   {},
		"DescribeFlowLogs":                 {},
		"DescribeFpgaImageAttribute":       {"FpgaImageId": "afi-00000000116", "Attribute": "name"},
		"DescribeFpgaImages":               {},
		"DescribeHostReservationOfferings": {},
		"DescribeHostReservations":         {},
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
