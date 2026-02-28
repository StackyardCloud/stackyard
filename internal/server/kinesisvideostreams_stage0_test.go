package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func kinesisVideoStreamsRequest(t *testing.T, ts *httptest.Server, path string, body []byte) *http.Response {
	t.Helper()
	headers := map[string]string{}
	if len(body) > 0 {
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, http.MethodPost, ts.URL+path, body, headers, "kinesisvideo")
}

func TestKinesisVideoStreamsStage0CatalogCoverage(t *testing.T) {
	if len(kinesisVideoStreamsOperations) != 32 {
		t.Fatalf("expected 32 Kinesis Video Streams operations from docs, got %d", len(kinesisVideoStreamsOperations))
	}
	if len(kinesisVideoStreamsOperationByName) != len(kinesisVideoStreamsOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"CreateStream",
		"DescribeStream",
		"ListStreams",
		"GetDataEndpoint",
		"CreateSignalingChannel",
		"GetSignalingChannelEndpoint",
		"TagResource",
	}
	for _, action := range requiredActions {
		if _, ok := kinesisVideoStreamsOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(kinesisVideoStreamsDataTypes) != 26 {
		t.Fatalf("expected 26 Kinesis Video Streams data types from docs, got %d", len(kinesisVideoStreamsDataTypes))
	}
	if len(kinesisVideoStreamsDataTypeByName) != len(kinesisVideoStreamsDataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"StreamInfo",
		"ChannelInfo",
		"MediaStorageConfiguration",
		"ImageGenerationConfiguration",
		"NotificationConfiguration",
	}
	for _, typeName := range requiredTypes {
		if _, ok := kinesisVideoStreamsDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestKinesisVideoStreamsStage0UnknownRouteReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := kinesisVideoStreamsRequest(t, ts, "/unknown-kinesisvideo-action", []byte(`{}`))
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestKinesisVideoStreamsStage0KnownActionReturnsListStreams(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := kinesisVideoStreamsRequest(t, ts, "/listStreams", []byte(`{"MaxResults":10}`))
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "StreamInfoList") {
		t.Fatalf("expected ListStreams response body to include StreamInfoList, got %q", body)
	}
}

func TestKinesisVideoStreamsStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range kinesisVideoStreamsOperations {
		resp := kinesisVideoStreamsRequest(t, ts, op.URI, []byte(`{}`))
		respBody := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(respBody, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
	}
}
