package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func mediaTailorRequest(t *testing.T, ts *httptest.Server, method, path, payload string) *http.Response {
	t.Helper()
	var body []byte
	headers := map[string]string{}
	if strings.TrimSpace(payload) != "" {
		body = []byte(payload)
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "mediatailor")
}

func mediaTailorPathForOperation(op mediaTailorOperation) string {
	path := op.URI
	replacements := map[string]string{
		"{ChannelName}":               "channel-00000001",
		"{ProgramName}":               "program-00000001",
		"{SourceLocationName}":        "source-location-00000001",
		"{VodSourceName}":             "vod-source-00000001",
		"{LiveSourceName}":            "live-source-00000001",
		"{PlaybackConfigurationName}": "playback-config-00000001",
		"{Name}":                      "prefetch-00000001",
		"{ResourceArn}":               url.PathEscape("arn:aws:mediatailor:us-east-1:123456789012:channel/channel-00000001"),
	}
	for key, value := range replacements {
		path = strings.ReplaceAll(path, key, value)
	}
	return path
}

func TestMediaTailorStage0CatalogCoverage(t *testing.T) {
	if len(mediaTailorOperations) != 44 {
		t.Fatalf("expected 44 MediaTailor operations from docs, got %d", len(mediaTailorOperations))
	}
	if len(mediaTailorOperationByName) != len(mediaTailorOperations) {
		t.Fatalf("expected unique MediaTailor operation names")
	}

	requiredActions := []string{
		"CreateChannel",
		"DescribeChannel",
		"ListChannels",
		"UpdateChannel",
		"DeleteChannel",
		"PutPlaybackConfiguration",
		"GetPlaybackConfiguration",
		"ListPlaybackConfigurations",
		"TagResource",
		"UntagResource",
		"ListTagsForResource",
	}
	for _, action := range requiredActions {
		if _, ok := mediaTailorOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(mediaTailorDataTypes) != 56 {
		t.Fatalf("expected 56 MediaTailor data types from docs, got %d", len(mediaTailorDataTypes))
	}
	if len(mediaTailorDataTypeByName) != len(mediaTailorDataTypes) {
		t.Fatalf("expected unique MediaTailor data type names")
	}
}

func TestMediaTailorStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := mediaTailorRequest(t, ts, http.MethodGet, "/not-a-real-mediatailor-route", "")
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestMediaTailorStage0KnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := mediaTailorRequest(t, ts, http.MethodGet, "/channels", "")
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "Items") {
		t.Fatalf("expected ListChannels response body to include Items, got %q", body)
	}
}

func TestMediaTailorStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range mediaTailorOperations {
		payload := `{}`
		if strings.EqualFold(op.Method, http.MethodGet) || strings.EqualFold(op.Method, http.MethodDelete) {
			payload = ""
		}
		resp := mediaTailorRequest(t, ts, op.Method, mediaTailorPathForOperation(op), payload)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
