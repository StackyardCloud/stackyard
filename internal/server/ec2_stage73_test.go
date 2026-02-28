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

func TestEC2Stage73SDKLifecycle(t *testing.T) {
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
		ImageId:      aws.String("ami-stage73"),
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
		Name:       aws.String("stage73-image"),
	})
	if err != nil || createImageOut.ImageId == nil {
		t.Fatalf("create image: %v", err)
	}
	imageID := aws.ToString(createImageOut.ImageId)

	disableImageOut, err := client.DisableImage(ctx, &awsec2.DisableImageInput{
		ImageId: aws.String(imageID),
	})
	if err != nil {
		t.Fatalf("disable image: %v", err)
	}
	if !aws.ToBool(disableImageOut.Return) {
		t.Fatalf("expected disable image return true")
	}

	describeDisabledOut, err := client.DescribeImages(ctx, &awsec2.DescribeImagesInput{ImageIds: []string{imageID}})
	if err != nil {
		t.Fatalf("describe images after disable image: %v", err)
	}
	if len(describeDisabledOut.Images) != 1 {
		t.Fatalf("expected one image after disable, got %d", len(describeDisabledOut.Images))
	}
	if describeDisabledOut.Images[0].State != awsec2types.ImageStateDisabled {
		t.Fatalf("unexpected image state after disable: %q", describeDisabledOut.Images[0].State)
	}

	enableImageOut, err := client.EnableImage(ctx, &awsec2.EnableImageInput{
		ImageId: aws.String(imageID),
	})
	if err != nil {
		t.Fatalf("enable image: %v", err)
	}
	if !aws.ToBool(enableImageOut.Return) {
		t.Fatalf("expected enable image return true")
	}

	describeEnabledOut, err := client.DescribeImages(ctx, &awsec2.DescribeImagesInput{ImageIds: []string{imageID}})
	if err != nil {
		t.Fatalf("describe images after enable image: %v", err)
	}
	if len(describeEnabledOut.Images) != 1 {
		t.Fatalf("expected one image after enable, got %d", len(describeEnabledOut.Images))
	}
	if describeEnabledOut.Images[0].State != awsec2types.ImageStateAvailable {
		t.Fatalf("unexpected image state after enable: %q", describeEnabledOut.Images[0].State)
	}

	enableFastLaunchOut, err := client.EnableFastLaunch(ctx, &awsec2.EnableFastLaunchInput{
		ImageId:             aws.String(imageID),
		MaxParallelLaunches: aws.Int32(6),
		ResourceType:        aws.String("snapshot"),
		SnapshotConfiguration: &awsec2types.FastLaunchSnapshotConfigurationRequest{
			TargetResourceCount: aws.Int32(2),
		},
	})
	if err != nil {
		t.Fatalf("enable fast launch: %v", err)
	}
	if aws.ToString(enableFastLaunchOut.ImageId) != imageID {
		t.Fatalf("unexpected enable fast launch image id: %q", aws.ToString(enableFastLaunchOut.ImageId))
	}
	if enableFastLaunchOut.ResourceType != awsec2types.FastLaunchResourceTypeSnapshot {
		t.Fatalf("unexpected enable fast launch resource type: %q", enableFastLaunchOut.ResourceType)
	}
	if enableFastLaunchOut.State != awsec2types.FastLaunchStateCodeEnabled {
		t.Fatalf("unexpected enable fast launch state: %q", enableFastLaunchOut.State)
	}
	if enableFastLaunchOut.MaxParallelLaunches == nil || *enableFastLaunchOut.MaxParallelLaunches != 6 {
		t.Fatalf("unexpected enable fast launch max parallel launches: %v", enableFastLaunchOut.MaxParallelLaunches)
	}
	if enableFastLaunchOut.SnapshotConfiguration == nil || enableFastLaunchOut.SnapshotConfiguration.TargetResourceCount == nil || *enableFastLaunchOut.SnapshotConfiguration.TargetResourceCount != 2 {
		t.Fatalf("unexpected enable fast launch snapshot configuration: %+v", enableFastLaunchOut.SnapshotConfiguration)
	}

	disableFastLaunchOut, err := client.DisableFastLaunch(ctx, &awsec2.DisableFastLaunchInput{
		ImageId: aws.String(imageID),
		Force:   aws.Bool(true),
	})
	if err != nil {
		t.Fatalf("disable fast launch: %v", err)
	}
	if aws.ToString(disableFastLaunchOut.ImageId) != imageID {
		t.Fatalf("unexpected disable fast launch image id: %q", aws.ToString(disableFastLaunchOut.ImageId))
	}
	if disableFastLaunchOut.State != awsec2types.FastLaunchStateCodeDisabling {
		t.Fatalf("unexpected disable fast launch state: %q", disableFastLaunchOut.State)
	}

	createVolumeOut, err := client.CreateVolume(ctx, &awsec2.CreateVolumeInput{
		AvailabilityZone: aws.String("us-east-1a"),
		Size:             aws.Int32(8),
		VolumeType:       awsec2types.VolumeTypeGp3,
	})
	if err != nil || createVolumeOut.VolumeId == nil {
		t.Fatalf("create volume: %v", err)
	}
	volumeID := aws.ToString(createVolumeOut.VolumeId)

	createSnapshotOut, err := client.CreateSnapshot(ctx, &awsec2.CreateSnapshotInput{
		VolumeId: aws.String(volumeID),
	})
	if err != nil || createSnapshotOut.SnapshotId == nil {
		t.Fatalf("create snapshot: %v", err)
	}
	snapshotID := aws.ToString(createSnapshotOut.SnapshotId)

	enableFastSnapshotRestoresOut, err := client.EnableFastSnapshotRestores(ctx, &awsec2.EnableFastSnapshotRestoresInput{
		SourceSnapshotIds: []string{snapshotID},
		AvailabilityZones: []string{"us-east-1a"},
	})
	if err != nil {
		t.Fatalf("enable fast snapshot restores: %v", err)
	}
	if len(enableFastSnapshotRestoresOut.Successful) != 1 {
		t.Fatalf("expected one successful fast snapshot restore enable item, got %d", len(enableFastSnapshotRestoresOut.Successful))
	}
	if len(enableFastSnapshotRestoresOut.Unsuccessful) != 0 {
		t.Fatalf("expected zero unsuccessful fast snapshot restore enable items, got %d", len(enableFastSnapshotRestoresOut.Unsuccessful))
	}
	if aws.ToString(enableFastSnapshotRestoresOut.Successful[0].SnapshotId) != snapshotID {
		t.Fatalf("unexpected enabled fast snapshot restore snapshot id: %q", aws.ToString(enableFastSnapshotRestoresOut.Successful[0].SnapshotId))
	}
	if enableFastSnapshotRestoresOut.Successful[0].State != awsec2types.FastSnapshotRestoreStateCodeEnabled {
		t.Fatalf("unexpected enabled fast snapshot restore state: %q", enableFastSnapshotRestoresOut.Successful[0].State)
	}

	disableFastSnapshotRestoresOut, err := client.DisableFastSnapshotRestores(ctx, &awsec2.DisableFastSnapshotRestoresInput{
		SourceSnapshotIds: []string{snapshotID},
		AvailabilityZones: []string{"us-east-1a"},
	})
	if err != nil {
		t.Fatalf("disable fast snapshot restores: %v", err)
	}
	if len(disableFastSnapshotRestoresOut.Successful) != 1 {
		t.Fatalf("expected one successful fast snapshot restore disable item, got %d", len(disableFastSnapshotRestoresOut.Successful))
	}
	if len(disableFastSnapshotRestoresOut.Unsuccessful) != 0 {
		t.Fatalf("expected zero unsuccessful fast snapshot restore disable items, got %d", len(disableFastSnapshotRestoresOut.Unsuccessful))
	}
	if aws.ToString(disableFastSnapshotRestoresOut.Successful[0].SnapshotId) != snapshotID {
		t.Fatalf("unexpected disabled fast snapshot restore snapshot id: %q", aws.ToString(disableFastSnapshotRestoresOut.Successful[0].SnapshotId))
	}
	if disableFastSnapshotRestoresOut.Successful[0].State != awsec2types.FastSnapshotRestoreStateCodeDisabled {
		t.Fatalf("unexpected disabled fast snapshot restore state: %q", disableFastSnapshotRestoresOut.Successful[0].State)
	}
}

func TestEC2Stage73ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"DisableFastLaunch",
		"DisableFastSnapshotRestores",
		"DisableImage",
		"EnableFastLaunch",
		"EnableFastSnapshotRestores",
		"EnableImage",
	}

	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, action := range implemented {
		params := map[string]string{}
		switch action {
		case "DisableFastLaunch", "DisableImage", "EnableFastLaunch", "EnableImage":
			params["ImageId"] = "ami-00000000"
		case "DisableFastSnapshotRestores", "EnableFastSnapshotRestores":
			params["SourceSnapshotId.1"] = "snap-00000000"
			params["AvailabilityZone.1"] = "us-east-1a"
		}
		if action == "EnableFastLaunch" {
			params["ResourceType"] = "snapshot"
		}
		resp := ec2Request(t, ts, action, params)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action)
		}
	}
}
