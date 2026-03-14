package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFlinkStage12ApplicationLifecycleAndReadSurface(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := flinkRequest(t, ts, "CreateApplication", `{"ApplicationName":"stage-flink-app","RuntimeEnvironment":"FLINK-1_18"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = flinkRequest(t, ts, "DescribeApplication", `{"ApplicationName":"stage-flink-app"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = flinkRequest(t, ts, "ListApplications", `{}`)
	assertStatus(t, resp, http.StatusOK)

	resp = flinkRequest(t, ts, "UpdateApplication", `{"ApplicationName":"stage-flink-app"}`)
	assertStatus(t, resp, http.StatusOK)
	updateBody := decodeShard9JSONBody(t, resp)
	if _, ok := updateBody["OperationId"]; ok {
		t.Fatalf("expected UpdateApplication to omit OperationId, got %#v", updateBody["OperationId"])
	}

	resp = flinkRequest(t, ts, "DeleteApplication", `{"ApplicationName":"stage-flink-app"}`)
	assertStatus(t, resp, http.StatusOK)
}

func TestFlinkStage34OperationsAndSnapshots(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := flinkRequest(t, ts, "CreateApplication", `{"ApplicationName":"stage-flink-ops"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = flinkRequest(t, ts, "StartApplication", `{"ApplicationName":"stage-flink-ops"}`)
	assertStatus(t, resp, http.StatusOK)
	startBody := decodeShard9JSONBody(t, resp)
	if _, ok := startBody["OperationId"]; ok {
		t.Fatalf("expected StartApplication to omit OperationId, got %#v", startBody["OperationId"])
	}

	resp = flinkRequest(t, ts, "ListApplicationOperations", `{"ApplicationName":"stage-flink-ops"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = flinkRequest(t, ts, "DescribeApplicationOperation", `{"ApplicationName":"stage-flink-ops"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = flinkRequest(t, ts, "CreateApplicationSnapshot", `{"ApplicationName":"stage-flink-ops","SnapshotName":"snap-001"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = flinkRequest(t, ts, "DescribeApplicationSnapshot", `{"ApplicationName":"stage-flink-ops","SnapshotName":"snap-001"}`)
	assertStatus(t, resp, http.StatusOK)
	describeSnapshotBody := decodeShard9JSONBody(t, resp)
	snapshotDetails, _ := describeSnapshotBody["SnapshotDetails"].(map[string]any)
	if _, ok := snapshotDetails["RuntimeEnvironment"]; ok {
		t.Fatalf("expected DescribeApplicationSnapshot to omit RuntimeEnvironment, got %#v", snapshotDetails["RuntimeEnvironment"])
	}

	resp = flinkRequest(t, ts, "ListApplicationSnapshots", `{"ApplicationName":"stage-flink-ops"}`)
	assertStatus(t, resp, http.StatusOK)
	listSnapshotsBody := decodeShard9JSONBody(t, resp)
	snapshots, _ := listSnapshotsBody["SnapshotSummaries"].([]any)
	if len(snapshots) == 0 {
		t.Fatalf("expected SnapshotSummaries, got %#v", listSnapshotsBody["SnapshotSummaries"])
	}
	listedSnapshot, _ := snapshots[0].(map[string]any)
	if _, ok := listedSnapshot["RuntimeEnvironment"]; ok {
		t.Fatalf("expected ListApplicationSnapshots to omit RuntimeEnvironment, got %#v", listedSnapshot["RuntimeEnvironment"])
	}

	resp = flinkRequest(t, ts, "DeleteApplicationSnapshot", `{"ApplicationName":"stage-flink-ops","SnapshotName":"snap-001"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = flinkRequest(t, ts, "StopApplication", `{"ApplicationName":"stage-flink-ops"}`)
	assertStatus(t, resp, http.StatusOK)
	stopBody := decodeShard9JSONBody(t, resp)
	if _, ok := stopBody["OperationId"]; ok {
		t.Fatalf("expected StopApplication to omit OperationId, got %#v", stopBody["OperationId"])
	}
}

func TestFlinkMutationResponsesOmitLegacyOperationID(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := flinkRequest(t, ts, "CreateApplication", `{"ApplicationName":"stage-flink-mutations"}`)
	assertStatus(t, resp, http.StatusOK)

	for _, action := range []string{
		"AddApplicationCloudWatchLoggingOption",
		"AddApplicationVpcConfiguration",
		"DeleteApplicationCloudWatchLoggingOption",
		"DeleteApplicationVpcConfiguration",
		"RollbackApplication",
		"UpdateApplication",
	} {
		resp = flinkRequest(t, ts, action, `{"ApplicationName":"stage-flink-mutations"}`)
		assertStatus(t, resp, http.StatusOK)
		body := decodeShard9JSONBody(t, resp)
		if _, ok := body["OperationId"]; ok {
			t.Fatalf("expected %s to omit OperationId, got %#v", action, body["OperationId"])
		}
	}

	resp = flinkRequest(t, ts, "DescribeApplicationVersion", `{"ApplicationName":"stage-flink-mutations"}`)
	assertStatus(t, resp, http.StatusOK)
	versionBody := decodeShard9JSONBody(t, resp)
	versionDetail, _ := versionBody["ApplicationVersionDetail"].(map[string]any)
	if got, _ := versionDetail["RuntimeEnvironment"].(string); strings.TrimSpace(got) == "" {
		t.Fatalf("expected DescribeApplicationVersion to keep RuntimeEnvironment, got %#v", versionDetail["RuntimeEnvironment"])
	}
}

func TestFlinkStage56TaggingValidationAndIdempotency(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resourceARN := "arn:aws:kinesisanalytics:us-east-1:123456789012:application/stage-flink-tags"

	resp := flinkRequest(t, ts, "CreateApplication", `{"ApplicationName":"stage-flink-tags"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = flinkRequest(t, ts, "TagResource", `{"ResourceARN":"`+resourceARN+`","Tags":[{"Key":"env","Value":"stage"},{"Key":"team","Value":"qa"}]}`)
	assertStatus(t, resp, http.StatusOK)

	resp = flinkRequest(t, ts, "ListTagsForResource", `{"ResourceARN":"`+resourceARN+`"}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "stage") {
		t.Fatalf("expected ListTagsForResource to include stage tag value, got %q", body)
	}

	resp = flinkRequest(t, ts, "UntagResource", `{"ResourceARN":"`+resourceARN+`","TagKeys":["team"]}`)
	assertStatus(t, resp, http.StatusOK)

	resp = flinkRequest(t, ts, "CreateApplicationPresignedUrl", `{"ApplicationName":"stage-flink-tags"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = flinkRequest(t, ts, "DiscoverInputSchema", `{}`)
	assertStatus(t, resp, http.StatusOK)

	resp = flinkRequest(t, ts, "TotallyUnknownAction", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException for unknown action, got %q", body)
	}

	resp = signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		[]byte(`{"broken":`),
		map[string]string{
			"Content-Type": "application/x-amz-json-1.1",
			"X-Amz-Target": "KinesisAnalytics_20180523.ListApplications",
		},
		"kinesisanalytics",
	)
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "invalid JSON body") {
		t.Fatalf("expected invalid JSON body error, got %q", body)
	}
}
