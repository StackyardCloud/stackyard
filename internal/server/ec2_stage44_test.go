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

func TestEC2Stage44SDKLifecycle(t *testing.T) {
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
		VolumeType:       awsec2types.VolumeTypeGp3,
	})
	if err != nil || createVolumeOut.VolumeId == nil {
		t.Fatalf("create volume: %v", err)
	}
	volumeID := aws.ToString(createVolumeOut.VolumeId)

	modifyVolumeOut, err := client.ModifyVolume(ctx, &awsec2.ModifyVolumeInput{
		VolumeId:           aws.String(volumeID),
		Size:               aws.Int32(16),
		VolumeType:         awsec2types.VolumeTypeGp3,
		Iops:               aws.Int32(4000),
		Throughput:         aws.Int32(200),
		MultiAttachEnabled: aws.Bool(true),
	})
	if err != nil || modifyVolumeOut.VolumeModification == nil {
		t.Fatalf("modify volume: %v", err)
	}
	if aws.ToString(modifyVolumeOut.VolumeModification.VolumeId) != volumeID {
		t.Fatalf("unexpected modified volume id: %q", aws.ToString(modifyVolumeOut.VolumeModification.VolumeId))
	}
	if modifyVolumeOut.VolumeModification.ModificationState != awsec2types.VolumeModificationStateCompleted {
		t.Fatalf("unexpected volume modification state: %q", modifyVolumeOut.VolumeModification.ModificationState)
	}
	if aws.ToInt32(modifyVolumeOut.VolumeModification.OriginalSize) != 8 || aws.ToInt32(modifyVolumeOut.VolumeModification.TargetSize) != 16 {
		t.Fatalf("unexpected volume modification size payload")
	}
	if modifyVolumeOut.VolumeModification.TargetVolumeType != awsec2types.VolumeTypeGp3 {
		t.Fatalf("unexpected target volume type: %q", modifyVolumeOut.VolumeModification.TargetVolumeType)
	}
	if aws.ToInt32(modifyVolumeOut.VolumeModification.TargetIops) != 4000 {
		t.Fatalf("unexpected target iops")
	}
	if aws.ToInt32(modifyVolumeOut.VolumeModification.TargetThroughput) != 200 {
		t.Fatalf("unexpected target throughput")
	}
	if !aws.ToBool(modifyVolumeOut.VolumeModification.TargetMultiAttachEnabled) {
		t.Fatalf("unexpected target multi-attach setting")
	}

	describeVolumeOut, err := client.DescribeVolumes(ctx, &awsec2.DescribeVolumesInput{
		VolumeIds: []string{volumeID},
	})
	if err != nil || len(describeVolumeOut.Volumes) != 1 {
		t.Fatalf("describe volumes: %v", err)
	}
	if aws.ToInt32(describeVolumeOut.Volumes[0].Size) != 16 {
		t.Fatalf("unexpected volume size after modify")
	}

	if _, err := client.ModifyVolumeAttribute(ctx, &awsec2.ModifyVolumeAttributeInput{
		VolumeId:     aws.String(volumeID),
		AutoEnableIO: &awsec2types.AttributeBooleanValue{Value: aws.Bool(false)},
	}); err != nil {
		t.Fatalf("modify volume attribute: %v", err)
	}
}

func TestEC2Stage44ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"ModifyVolume",
		"ModifyVolumeAttribute",
	}

	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, action := range implemented {
		params := map[string]string{}
		switch action {
		case "ModifyVolume", "ModifyVolumeAttribute":
			params["VolumeId"] = "vol-00000044"
		}
		resp := ec2Request(t, ts, action, params)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action)
		}
	}
}
