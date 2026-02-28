package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awslightsail "github.com/aws/aws-sdk-go-v2/service/lightsail"
	awslightsailtypes "github.com/aws/aws-sdk-go-v2/service/lightsail/types"
)

func TestLightsailStage8DiskSnapshotsAndMigration(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := lightsailRequest(t, ts, "CreateInstances", []byte(`{"availabilityZone":"us-east-1a","blueprintId":"amazon_linux_2","bundleId":"micro_2_0","instanceNames":["stage8-instance"]}`))
	assertStatus(t, resp, http.StatusOK)
	resp = lightsailRequest(t, ts, "CreateDisk", []byte(`{"availabilityZone":"us-east-1a","diskName":"stage8-disk","sizeInGb":64}`))
	assertStatus(t, resp, http.StatusOK)

	resp = lightsailRequest(t, ts, "CreateDiskSnapshot", []byte(`{"diskName":"stage8-disk","diskSnapshotName":"stage8-snap"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = lightsailRequest(t, ts, "GetDiskSnapshot", []byte(`{"diskSnapshotName":"stage8-snap"}`))
	assertStatus(t, resp, http.StatusOK)
	var getDiskSnapshotOut struct {
		DiskSnapshot struct {
			Name         string `json:"name"`
			FromDiskName string `json:"fromDiskName"`
			SizeInGb     int32  `json:"sizeInGb"`
			State        string `json:"state"`
		} `json:"diskSnapshot"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &getDiskSnapshotOut); err != nil {
		t.Fatalf("unmarshal GetDiskSnapshot: %v", err)
	}
	if getDiskSnapshotOut.DiskSnapshot.Name != "stage8-snap" || getDiskSnapshotOut.DiskSnapshot.FromDiskName != "stage8-disk" || getDiskSnapshotOut.DiskSnapshot.SizeInGb != 64 {
		t.Fatalf("unexpected GetDiskSnapshot output: %+v", getDiskSnapshotOut)
	}

	resp = lightsailRequest(t, ts, "GetDiskSnapshots", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	var getDiskSnapshotsOut struct {
		DiskSnapshots []struct {
			Name string `json:"name"`
		} `json:"diskSnapshots"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &getDiskSnapshotsOut); err != nil {
		t.Fatalf("unmarshal GetDiskSnapshots: %v", err)
	}
	if len(getDiskSnapshotsOut.DiskSnapshots) != 1 || getDiskSnapshotsOut.DiskSnapshots[0].Name != "stage8-snap" {
		t.Fatalf("unexpected GetDiskSnapshots output: %+v", getDiskSnapshotsOut)
	}

	resp = lightsailRequest(t, ts, "CreateDiskFromSnapshot", []byte(`{"availabilityZone":"us-east-1a","diskName":"stage8-restored","diskSnapshotName":"stage8-snap","sizeInGb":64}`))
	assertStatus(t, resp, http.StatusOK)
	resp = lightsailRequest(t, ts, "GetDisk", []byte(`{"diskName":"stage8-restored"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = lightsailRequest(t, ts, "CopySnapshot", []byte(`{"sourceRegion":"us-east-1","sourceSnapshotName":"stage8-snap","targetSnapshotName":"stage8-snap-copy"}`))
	assertStatus(t, resp, http.StatusOK)
	resp = lightsailRequest(t, ts, "GetDiskSnapshot", []byte(`{"diskSnapshotName":"stage8-snap-copy"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = lightsailRequest(t, ts, "ExportSnapshot", []byte(`{"sourceSnapshotName":"stage8-snap"}`))
	assertStatus(t, resp, http.StatusOK)
	resp = lightsailRequest(t, ts, "GetExportSnapshotRecords", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	var exportRecordsOut struct {
		ExportSnapshotRecords []struct {
			Name string `json:"name"`
		} `json:"exportSnapshotRecords"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &exportRecordsOut); err != nil {
		t.Fatalf("unmarshal GetExportSnapshotRecords: %v", err)
	}
	if len(exportRecordsOut.ExportSnapshotRecords) != 1 {
		t.Fatalf("expected one export record, got %d", len(exportRecordsOut.ExportSnapshotRecords))
	}

	resp = lightsailRequest(t, ts, "DeleteDiskSnapshot", []byte(`{"diskSnapshotName":"stage8-snap-copy"}`))
	assertStatus(t, resp, http.StatusOK)
	resp = lightsailRequest(t, ts, "DeleteDiskSnapshot", []byte(`{"diskSnapshotName":"stage8-snap"}`))
	assertStatus(t, resp, http.StatusOK)
}

func TestLightsailStage8SDKClientDiskSnapshotsAndMigration(t *testing.T) {
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

	client := awslightsail.NewFromConfig(cfg, func(o *awslightsail.Options) {
		o.BaseEndpoint = aws.String(ts.URL)
	})

	if _, err := client.CreateInstances(ctx, &awslightsail.CreateInstancesInput{
		AvailabilityZone: aws.String("us-east-1a"),
		BlueprintId:      aws.String("amazon_linux_2"),
		BundleId:         aws.String("micro_2_0"),
		InstanceNames:    []string{"sdk-stage8-instance"},
	}); err != nil {
		t.Fatalf("create instance: %v", err)
	}
	if _, err := client.CreateDisk(ctx, &awslightsail.CreateDiskInput{
		AvailabilityZone: aws.String("us-east-1a"),
		DiskName:         aws.String("sdk-stage8-disk"),
		SizeInGb:         aws.Int32(64),
	}); err != nil {
		t.Fatalf("create disk: %v", err)
	}

	if _, err := client.CreateDiskSnapshot(ctx, &awslightsail.CreateDiskSnapshotInput{
		DiskName:         aws.String("sdk-stage8-disk"),
		DiskSnapshotName: aws.String("sdk-stage8-snap"),
	}); err != nil {
		t.Fatalf("create disk snapshot: %v", err)
	}

	getSnapshotOut, err := client.GetDiskSnapshot(ctx, &awslightsail.GetDiskSnapshotInput{
		DiskSnapshotName: aws.String("sdk-stage8-snap"),
	})
	if err != nil {
		t.Fatalf("get disk snapshot: %v", err)
	}
	if getSnapshotOut.DiskSnapshot == nil || aws.ToString(getSnapshotOut.DiskSnapshot.Name) != "sdk-stage8-snap" {
		t.Fatalf("unexpected get disk snapshot output")
	}

	listSnapshotsOut, err := client.GetDiskSnapshots(ctx, &awslightsail.GetDiskSnapshotsInput{})
	if err != nil {
		t.Fatalf("get disk snapshots: %v", err)
	}
	if len(listSnapshotsOut.DiskSnapshots) == 0 {
		t.Fatalf("expected at least one disk snapshot")
	}

	if _, err := client.CreateDiskFromSnapshot(ctx, &awslightsail.CreateDiskFromSnapshotInput{
		AvailabilityZone: aws.String("us-east-1a"),
		DiskName:         aws.String("sdk-stage8-restored"),
		DiskSnapshotName: aws.String("sdk-stage8-snap"),
		SizeInGb:         aws.Int32(64),
	}); err != nil {
		t.Fatalf("create disk from snapshot: %v", err)
	}

	if _, err := client.CopySnapshot(ctx, &awslightsail.CopySnapshotInput{
		SourceRegion:       awslightsailtypes.RegionName("us-east-1"),
		SourceSnapshotName: aws.String("sdk-stage8-snap"),
		TargetSnapshotName: aws.String("sdk-stage8-snap-copy"),
	}); err != nil {
		t.Fatalf("copy snapshot: %v", err)
	}

	if _, err := client.ExportSnapshot(ctx, &awslightsail.ExportSnapshotInput{
		SourceSnapshotName: aws.String("sdk-stage8-snap"),
	}); err != nil {
		t.Fatalf("export snapshot: %v", err)
	}

	exportRecordsOut, err := client.GetExportSnapshotRecords(ctx, &awslightsail.GetExportSnapshotRecordsInput{})
	if err != nil {
		t.Fatalf("get export snapshot records: %v", err)
	}
	if len(exportRecordsOut.ExportSnapshotRecords) != 1 {
		t.Fatalf("expected one export snapshot record, got %d", len(exportRecordsOut.ExportSnapshotRecords))
	}

	if _, err := client.DeleteDiskSnapshot(ctx, &awslightsail.DeleteDiskSnapshotInput{
		DiskSnapshotName: aws.String("sdk-stage8-snap-copy"),
	}); err != nil {
		t.Fatalf("delete copied snapshot: %v", err)
	}
	if _, err := client.DeleteDiskSnapshot(ctx, &awslightsail.DeleteDiskSnapshotInput{
		DiskSnapshotName: aws.String("sdk-stage8-snap"),
	}); err != nil {
		t.Fatalf("delete source snapshot: %v", err)
	}
}
