package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func mediaPackageRequest(t *testing.T, ts *httptest.Server, method, path, payload string) *http.Response {
	t.Helper()
	var body []byte
	headers := map[string]string{}
	if strings.TrimSpace(payload) != "" {
		body = []byte(payload)
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "mediapackagev2")
}

func mediaPackagePathForOperation(op mediaPackageOperation) string {
	path := op.URI
	replacements := map[string]string{
		"{ChannelGroupName}":   "channel-group-00000001",
		"{ChannelName}":        "channel-00000001",
		"{OriginEndpointName}": "origin-endpoint-00000001",
		"{HarvestJobName}":     "harvest-job-00000001",
		"{ResourceArn}":        url.PathEscape("arn:aws:mediapackagev2:us-east-1:123456789012:channelGroup/channel-group-00000001"),
	}
	for key, value := range replacements {
		path = strings.ReplaceAll(path, key, value)
	}
	return path
}

func TestMediaPackageStage0CatalogCoverage(t *testing.T) {
	if len(mediaPackageOperations) != 30 {
		t.Fatalf("expected 30 MediaPackage operations from docs, got %d", len(mediaPackageOperations))
	}
	if len(mediaPackageOperationByName) != len(mediaPackageOperations) {
		t.Fatalf("expected unique MediaPackage operation names")
	}

	requiredActions := []string{
		"CreateChannelGroup",
		"CreateChannel",
		"CreateOriginEndpoint",
		"CreateHarvestJob",
		"ResetChannelState",
		"ResetOriginEndpointState",
		"TagResource",
		"UntagResource",
		"ListTagsForResource",
	}
	for _, action := range requiredActions {
		if _, ok := mediaPackageOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(mediaPackageDataTypes) != 46 {
		t.Fatalf("expected 46 MediaPackage data types from docs, got %d", len(mediaPackageDataTypes))
	}
	if len(mediaPackageDataTypeByName) != len(mediaPackageDataTypes) {
		t.Fatalf("expected unique MediaPackage data type names")
	}
}

func TestMediaPackageStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := mediaPackageRequest(t, ts, http.MethodGet, "/channelGroup/not-real-channel-group/nope", "")
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestMediaPackageStage0KnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := mediaPackageRequest(t, ts, http.MethodGet, "/channelGroup", "")
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "Items") {
		t.Fatalf("expected ListChannelGroups response body to include Items, got %q", body)
	}
}

func TestMediaPackageStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range mediaPackageOperations {
		payload := `{}`
		if strings.EqualFold(op.Method, http.MethodGet) || strings.EqualFold(op.Method, http.MethodDelete) {
			payload = ""
		}
		resp := mediaPackageRequest(t, ts, op.Method, mediaPackagePathForOperation(op), payload)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
