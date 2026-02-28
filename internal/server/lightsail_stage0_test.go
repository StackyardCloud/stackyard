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

func lightsailRequest(t *testing.T, ts *httptest.Server, action string, body []byte) *http.Response {
	t.Helper()
	return signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		body,
		map[string]string{
			"Content-Type": "application/x-amz-json-1.1",
			"X-Amz-Target": "Lightsail_20161128." + action,
		},
		"lightsail",
	)
}

func TestLightsailStage0Lifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := lightsailRequest(t, ts, "CreateInstances", []byte(`{"availabilityZone":"us-east-1a","blueprintId":"amazon_linux_2","bundleId":"micro_2_0","instanceNames":["demo-instance"],"tags":[{"key":"env","value":"test"}]}`))
	assertStatus(t, resp, http.StatusOK)
	var createOut struct {
		Operations []struct {
			ID string `json:"id"`
		} `json:"operations"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &createOut); err != nil {
		t.Fatalf("unmarshal CreateInstances: %v", err)
	}
	if len(createOut.Operations) != 1 || createOut.Operations[0].ID == "" {
		t.Fatalf("expected create operation id")
	}

	resp = lightsailRequest(t, ts, "GetInstance", []byte(`{"instanceName":"demo-instance"}`))
	assertStatus(t, resp, http.StatusOK)
	resp = lightsailRequest(t, ts, "GetInstances", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	resp = lightsailRequest(t, ts, "GetInstanceState", []byte(`{"instanceName":"demo-instance"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = lightsailRequest(t, ts, "StopInstance", []byte(`{"instanceName":"demo-instance"}`))
	assertStatus(t, resp, http.StatusOK)
	resp = lightsailRequest(t, ts, "StartInstance", []byte(`{"instanceName":"demo-instance"}`))
	assertStatus(t, resp, http.StatusOK)
	resp = lightsailRequest(t, ts, "RebootInstance", []byte(`{"instanceName":"demo-instance"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = lightsailRequest(t, ts, "CreateInstanceSnapshot", []byte(`{"instanceName":"demo-instance","instanceSnapshotName":"demo-snap"}`))
	assertStatus(t, resp, http.StatusOK)
	resp = lightsailRequest(t, ts, "GetInstanceSnapshot", []byte(`{"instanceSnapshotName":"demo-snap"}`))
	assertStatus(t, resp, http.StatusOK)
	resp = lightsailRequest(t, ts, "GetInstanceSnapshots", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)

	resp = lightsailRequest(t, ts, "AllocateStaticIp", []byte(`{"staticIpName":"demo-ip"}`))
	assertStatus(t, resp, http.StatusOK)
	resp = lightsailRequest(t, ts, "GetStaticIp", []byte(`{"staticIpName":"demo-ip"}`))
	assertStatus(t, resp, http.StatusOK)
	resp = lightsailRequest(t, ts, "GetStaticIps", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	resp = lightsailRequest(t, ts, "AttachStaticIp", []byte(`{"staticIpName":"demo-ip","instanceName":"demo-instance"}`))
	assertStatus(t, resp, http.StatusOK)
	resp = lightsailRequest(t, ts, "DetachStaticIp", []byte(`{"staticIpName":"demo-ip"}`))
	assertStatus(t, resp, http.StatusOK)
	resp = lightsailRequest(t, ts, "ReleaseStaticIp", []byte(`{"staticIpName":"demo-ip"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = lightsailRequest(t, ts, "TagResource", []byte(`{"resourceName":"demo-instance","tags":[{"key":"role","value":"web"}]}`))
	assertStatus(t, resp, http.StatusOK)
	resp = lightsailRequest(t, ts, "UntagResource", []byte(`{"resourceName":"demo-instance","tagKeys":["role"]}`))
	assertStatus(t, resp, http.StatusOK)

	resp = lightsailRequest(t, ts, "GetRegions", []byte(`{"includeAvailabilityZones":true,"includeRelationalDatabaseAvailabilityZones":true}`))
	assertStatus(t, resp, http.StatusOK)

	resp = lightsailRequest(t, ts, "GetOperationsForResource", []byte(`{"resourceName":"demo-instance"}`))
	assertStatus(t, resp, http.StatusOK)
	var forResourceOut struct {
		Operations []struct {
			ID string `json:"id"`
		} `json:"operations"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &forResourceOut); err != nil {
		t.Fatalf("unmarshal GetOperationsForResource: %v", err)
	}
	if len(forResourceOut.Operations) == 0 {
		t.Fatalf("expected operations for resource")
	}

	resp = lightsailRequest(t, ts, "GetOperation", []byte(`{"operationId":"`+forResourceOut.Operations[0].ID+`"}`))
	assertStatus(t, resp, http.StatusOK)
	resp = lightsailRequest(t, ts, "GetOperations", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)

	resp = lightsailRequest(t, ts, "DeleteInstanceSnapshot", []byte(`{"instanceSnapshotName":"demo-snap"}`))
	assertStatus(t, resp, http.StatusOK)
	resp = lightsailRequest(t, ts, "DeleteInstance", []byte(`{"instanceName":"demo-instance"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = lightsailRequest(t, ts, "GetSetupHistory", []byte(`{}`))
	assertStatus(t, resp, http.StatusBadRequest)
}

func TestLightsailStage0OperationCoverage(t *testing.T) {
	if len(lightsailOperations) != 161 {
		t.Fatalf("expected 161 Lightsail operations from docs, got %d", len(lightsailOperations))
	}
	if len(lightsailOperationByName) != len(lightsailOperations) {
		t.Fatalf("expected unique operation names")
	}
	required := []string{
		"CreateInstances",
		"GetInstance",
		"GetOperation",
		"GetOperations",
		"GetOperationsForResource",
		"GetRegions",
		"GetSetupHistory",
		"SetupInstanceHttps",
		"TagResource",
		"UntagResource",
	}
	for _, name := range required {
		if _, ok := lightsailOperationByName[name]; !ok {
			t.Fatalf("missing documented operation %s", name)
		}
	}
}

func TestLightsailStage0SDKClientLifecycle(t *testing.T) {
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

	createResp, err := client.CreateInstances(ctx, &awslightsail.CreateInstancesInput{
		AvailabilityZone: aws.String("us-east-1a"),
		BlueprintId:      aws.String("amazon_linux_2"),
		BundleId:         aws.String("micro_2_0"),
		InstanceNames:    []string{"sdk-instance"},
	})
	if err != nil {
		t.Fatalf("create instances: %v", err)
	}
	if len(createResp.Operations) != 1 || createResp.Operations[0].Id == nil {
		t.Fatalf("expected create operation id")
	}

	if _, err := client.GetInstance(ctx, &awslightsail.GetInstanceInput{InstanceName: aws.String("sdk-instance")}); err != nil {
		t.Fatalf("get instance: %v", err)
	}
	if _, err := client.GetInstances(ctx, &awslightsail.GetInstancesInput{}); err != nil {
		t.Fatalf("get instances: %v", err)
	}
	if _, err := client.GetInstanceState(ctx, &awslightsail.GetInstanceStateInput{InstanceName: aws.String("sdk-instance")}); err != nil {
		t.Fatalf("get instance state: %v", err)
	}

	if _, err := client.CreateInstanceSnapshot(ctx, &awslightsail.CreateInstanceSnapshotInput{
		InstanceName:         aws.String("sdk-instance"),
		InstanceSnapshotName: aws.String("sdk-snap"),
	}); err != nil {
		t.Fatalf("create instance snapshot: %v", err)
	}
	if _, err := client.GetInstanceSnapshot(ctx, &awslightsail.GetInstanceSnapshotInput{InstanceSnapshotName: aws.String("sdk-snap")}); err != nil {
		t.Fatalf("get instance snapshot: %v", err)
	}
	if _, err := client.GetInstanceSnapshots(ctx, &awslightsail.GetInstanceSnapshotsInput{}); err != nil {
		t.Fatalf("get instance snapshots: %v", err)
	}

	if _, err := client.AllocateStaticIp(ctx, &awslightsail.AllocateStaticIpInput{StaticIpName: aws.String("sdk-ip")}); err != nil {
		t.Fatalf("allocate static ip: %v", err)
	}
	if _, err := client.GetStaticIp(ctx, &awslightsail.GetStaticIpInput{StaticIpName: aws.String("sdk-ip")}); err != nil {
		t.Fatalf("get static ip: %v", err)
	}
	if _, err := client.AttachStaticIp(ctx, &awslightsail.AttachStaticIpInput{
		StaticIpName: aws.String("sdk-ip"),
		InstanceName: aws.String("sdk-instance"),
	}); err != nil {
		t.Fatalf("attach static ip: %v", err)
	}
	if _, err := client.DetachStaticIp(ctx, &awslightsail.DetachStaticIpInput{StaticIpName: aws.String("sdk-ip")}); err != nil {
		t.Fatalf("detach static ip: %v", err)
	}
	if _, err := client.ReleaseStaticIp(ctx, &awslightsail.ReleaseStaticIpInput{StaticIpName: aws.String("sdk-ip")}); err != nil {
		t.Fatalf("release static ip: %v", err)
	}

	if _, err := client.TagResource(ctx, &awslightsail.TagResourceInput{
		ResourceName: aws.String("sdk-instance"),
		Tags:         []awslightsailtypes.Tag{{Key: aws.String("env"), Value: aws.String("test")}},
	}); err != nil {
		t.Fatalf("tag resource: %v", err)
	}
	if _, err := client.UntagResource(ctx, &awslightsail.UntagResourceInput{
		ResourceName: aws.String("sdk-instance"),
		TagKeys:      []string{"env"},
	}); err != nil {
		t.Fatalf("untag resource: %v", err)
	}

	if _, err := client.GetRegions(ctx, &awslightsail.GetRegionsInput{
		IncludeAvailabilityZones:                   aws.Bool(true),
		IncludeRelationalDatabaseAvailabilityZones: aws.Bool(true),
	}); err != nil {
		t.Fatalf("get regions: %v", err)
	}

	if _, err := client.GetOperations(ctx, &awslightsail.GetOperationsInput{}); err != nil {
		t.Fatalf("get operations: %v", err)
	}
	if _, err := client.GetOperationsForResource(ctx, &awslightsail.GetOperationsForResourceInput{ResourceName: aws.String("sdk-instance")}); err != nil {
		t.Fatalf("get operations for resource: %v", err)
	}
	if _, err := client.GetOperation(ctx, &awslightsail.GetOperationInput{OperationId: createResp.Operations[0].Id}); err != nil {
		t.Fatalf("get operation: %v", err)
	}

	if _, err := client.DeleteInstanceSnapshot(ctx, &awslightsail.DeleteInstanceSnapshotInput{InstanceSnapshotName: aws.String("sdk-snap")}); err != nil {
		t.Fatalf("delete instance snapshot: %v", err)
	}
	if _, err := client.DeleteInstance(ctx, &awslightsail.DeleteInstanceInput{InstanceName: aws.String("sdk-instance")}); err != nil {
		t.Fatalf("delete instance: %v", err)
	}
}
