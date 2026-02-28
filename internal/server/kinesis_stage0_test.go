package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func kinesisRequest(t *testing.T, ts *httptest.Server, action, payload string) *http.Response {
	t.Helper()
	return signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		[]byte(payload),
		map[string]string{
			"Content-Type": "application/x-amz-json-1.1",
			"X-Amz-Target": "Kinesis_20131202." + action,
		},
		"kinesis",
	)
}

func TestKinesisStage0CatalogCoverage(t *testing.T) {
	if len(kinesisOperations) != 39 {
		t.Fatalf("expected 39 Kinesis operations from docs, got %d", len(kinesisOperations))
	}
	if len(kinesisOperationByName) != len(kinesisOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"CreateStream",
		"DescribeStreamSummary",
		"ListShards",
		"PutRecord",
		"PutRecords",
		"RegisterStreamConsumer",
		"UpdateStreamMode",
	}
	for _, action := range requiredActions {
		if _, ok := kinesisOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(kinesisDataTypes) != 23 {
		t.Fatalf("expected 23 Kinesis data types from docs, got %d", len(kinesisDataTypes))
	}
	if len(kinesisDataTypeByName) != len(kinesisDataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"Shard",
		"Record",
		"Consumer",
		"StreamDescriptionSummary",
		"WarmThroughputObject",
	}
	for _, typeName := range requiredTypes {
		if _, ok := kinesisDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestKinesisStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := kinesisRequest(t, ts, "TotallyUnknownAction", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestKinesisKnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := kinesisRequest(t, ts, "ListStreams", `{}`)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "StreamNames") {
		t.Fatalf("expected ListStreams response body to include StreamNames, got %q", body)
	}
}

func TestKinesisStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range kinesisOperations {
		resp := kinesisRequest(t, ts, op.Name, `{}`)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
