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

	resp = flinkRequest(t, ts, "ListApplicationOperations", `{"ApplicationName":"stage-flink-ops"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = flinkRequest(t, ts, "DescribeApplicationOperation", `{"ApplicationName":"stage-flink-ops"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = flinkRequest(t, ts, "CreateApplicationSnapshot", `{"ApplicationName":"stage-flink-ops","SnapshotName":"snap-001"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = flinkRequest(t, ts, "DescribeApplicationSnapshot", `{"ApplicationName":"stage-flink-ops","SnapshotName":"snap-001"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = flinkRequest(t, ts, "ListApplicationSnapshots", `{"ApplicationName":"stage-flink-ops"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = flinkRequest(t, ts, "DeleteApplicationSnapshot", `{"ApplicationName":"stage-flink-ops","SnapshotName":"snap-001"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = flinkRequest(t, ts, "StopApplication", `{"ApplicationName":"stage-flink-ops"}`)
	assertStatus(t, resp, http.StatusOK)
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
