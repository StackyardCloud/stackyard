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

func TestEC2Stage101SDKLifecycle(t *testing.T) {
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

	createSnapshotOut, err := client.CreateSnapshot(ctx, &awsec2.CreateSnapshotInput{
		VolumeId: aws.String(volumeID),
	})
	if err != nil || createSnapshotOut.SnapshotId == nil {
		t.Fatalf("create snapshot: %v", err)
	}
	sourceSnapshotID := aws.ToString(createSnapshotOut.SnapshotId)

	copyOut, err := client.CopySnapshot(ctx, &awsec2.CopySnapshotInput{
		SourceRegion:     aws.String("us-east-1"),
		SourceSnapshotId: aws.String(sourceSnapshotID),
		Description:      aws.String("stage101 snapshot copy"),
		TagSpecifications: []awsec2types.TagSpecification{
			{
				ResourceType: awsec2types.ResourceTypeSnapshot,
				Tags: []awsec2types.Tag{
					{Key: aws.String("env"), Value: aws.String("stage101")},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("copy snapshot: %v", err)
	}
	if copyOut.SnapshotId == nil || strings.TrimSpace(aws.ToString(copyOut.SnapshotId)) == "" {
		t.Fatalf("expected snapshot id")
	}
	if !strings.HasPrefix(aws.ToString(copyOut.SnapshotId), "snap-") {
		t.Fatalf("unexpected copied snapshot id: %q", aws.ToString(copyOut.SnapshotId))
	}
	if len(copyOut.Tags) != 1 {
		t.Fatalf("expected 1 copied snapshot tag, got %d", len(copyOut.Tags))
	}
	if aws.ToString(copyOut.Tags[0].Key) != "env" || aws.ToString(copyOut.Tags[0].Value) != "stage101" {
		t.Fatalf("unexpected copied snapshot tag: %+v", copyOut.Tags[0])
	}
}

func TestEC2Stage101ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"CopySnapshot",
	}

	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, action := range implemented {
		params := map[string]string{
			"SourceRegion":     "us-east-1",
			"SourceSnapshotId": "snap-00000000000000101",
		}
		resp := ec2Request(t, ts, action, params)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action)
		}
	}
}
