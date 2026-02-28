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

func TestLightsailStage7DisksCore(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := lightsailRequest(t, ts, "CreateInstances", []byte(`{"availabilityZone":"us-east-1a","blueprintId":"amazon_linux_2","bundleId":"micro_2_0","instanceNames":["stage7-instance"]}`))
	assertStatus(t, resp, http.StatusOK)

	resp = lightsailRequest(t, ts, "CreateDisk", []byte(`{"availabilityZone":"us-east-1a","diskName":"stage7-disk","sizeInGb":32,"tags":[{"key":"env","value":"test"}]}`))
	assertStatus(t, resp, http.StatusOK)

	resp = lightsailRequest(t, ts, "GetDisk", []byte(`{"diskName":"stage7-disk"}`))
	assertStatus(t, resp, http.StatusOK)
	var getDiskOut struct {
		Disk struct {
			Name     string `json:"name"`
			SizeInGb int32  `json:"sizeInGb"`
			State    string `json:"state"`
		} `json:"disk"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &getDiskOut); err != nil {
		t.Fatalf("unmarshal GetDisk: %v", err)
	}
	if getDiskOut.Disk.Name != "stage7-disk" || getDiskOut.Disk.SizeInGb != 32 || getDiskOut.Disk.State != "available" {
		t.Fatalf("unexpected GetDisk output: %+v", getDiskOut)
	}

	resp = lightsailRequest(t, ts, "GetDisks", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	var getDisksOut struct {
		Disks []struct {
			Name string `json:"name"`
		} `json:"disks"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &getDisksOut); err != nil {
		t.Fatalf("unmarshal GetDisks: %v", err)
	}
	if len(getDisksOut.Disks) != 1 || getDisksOut.Disks[0].Name != "stage7-disk" {
		t.Fatalf("unexpected GetDisks output: %+v", getDisksOut)
	}

	resp = lightsailRequest(t, ts, "AttachDisk", []byte(`{"diskName":"stage7-disk","diskPath":"/dev/xvdf","instanceName":"stage7-instance","autoMounting":true}`))
	assertStatus(t, resp, http.StatusOK)

	resp = lightsailRequest(t, ts, "GetDisk", []byte(`{"diskName":"stage7-disk"}`))
	assertStatus(t, resp, http.StatusOK)
	var attachedDiskOut struct {
		Disk struct {
			IsAttached      bool   `json:"isAttached"`
			AttachedTo      string `json:"attachedTo"`
			Path            string `json:"path"`
			State           string `json:"state"`
			AutoMountStatus string `json:"autoMountStatus"`
		} `json:"disk"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &attachedDiskOut); err != nil {
		t.Fatalf("unmarshal attached GetDisk: %v", err)
	}
	if !attachedDiskOut.Disk.IsAttached || attachedDiskOut.Disk.AttachedTo != "stage7-instance" || attachedDiskOut.Disk.Path != "/dev/xvdf" || attachedDiskOut.Disk.State != "in-use" {
		t.Fatalf("unexpected attached disk output: %+v", attachedDiskOut)
	}
	if attachedDiskOut.Disk.AutoMountStatus != "Mounted" {
		t.Fatalf("expected mounted auto mount status, got %q", attachedDiskOut.Disk.AutoMountStatus)
	}

	resp = lightsailRequest(t, ts, "DetachDisk", []byte(`{"diskName":"stage7-disk"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = lightsailRequest(t, ts, "DeleteDisk", []byte(`{"diskName":"stage7-disk"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = lightsailRequest(t, ts, "GetDisk", []byte(`{"diskName":"stage7-disk"}`))
	assertStatus(t, resp, http.StatusNotFound)
}

func TestLightsailStage7SDKClientDisksCore(t *testing.T) {
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
		InstanceNames:    []string{"sdk-stage7-instance"},
	}); err != nil {
		t.Fatalf("create instance: %v", err)
	}

	createOut, err := client.CreateDisk(ctx, &awslightsail.CreateDiskInput{
		AvailabilityZone: aws.String("us-east-1a"),
		DiskName:         aws.String("sdk-stage7-disk"),
		SizeInGb:         aws.Int32(32),
		Tags: []awslightsailtypes.Tag{
			{Key: aws.String("env"), Value: aws.String("test")},
		},
	})
	if err != nil {
		t.Fatalf("create disk: %v", err)
	}
	if len(createOut.Operations) != 1 {
		t.Fatalf("expected one create disk operation")
	}

	getOut, err := client.GetDisk(ctx, &awslightsail.GetDiskInput{DiskName: aws.String("sdk-stage7-disk")})
	if err != nil {
		t.Fatalf("get disk: %v", err)
	}
	if getOut.Disk == nil || aws.ToString(getOut.Disk.Name) != "sdk-stage7-disk" || aws.ToInt32(getOut.Disk.SizeInGb) != 32 {
		t.Fatalf("unexpected GetDisk output")
	}

	listOut, err := client.GetDisks(ctx, &awslightsail.GetDisksInput{})
	if err != nil {
		t.Fatalf("get disks: %v", err)
	}
	if len(listOut.Disks) == 0 {
		t.Fatalf("expected at least one disk")
	}

	if _, err := client.AttachDisk(ctx, &awslightsail.AttachDiskInput{
		DiskName:     aws.String("sdk-stage7-disk"),
		DiskPath:     aws.String("/dev/xvdf"),
		InstanceName: aws.String("sdk-stage7-instance"),
		AutoMounting: aws.Bool(true),
	}); err != nil {
		t.Fatalf("attach disk: %v", err)
	}

	getOut, err = client.GetDisk(ctx, &awslightsail.GetDiskInput{DiskName: aws.String("sdk-stage7-disk")})
	if err != nil {
		t.Fatalf("get disk after attach: %v", err)
	}
	if getOut.Disk == nil || !aws.ToBool(getOut.Disk.IsAttached) || aws.ToString(getOut.Disk.AttachedTo) != "sdk-stage7-instance" {
		t.Fatalf("unexpected attached disk output")
	}

	if _, err := client.DetachDisk(ctx, &awslightsail.DetachDiskInput{DiskName: aws.String("sdk-stage7-disk")}); err != nil {
		t.Fatalf("detach disk: %v", err)
	}
	if _, err := client.DeleteDisk(ctx, &awslightsail.DeleteDiskInput{DiskName: aws.String("sdk-stage7-disk")}); err != nil {
		t.Fatalf("delete disk: %v", err)
	}
}
