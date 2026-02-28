package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func detectiveRequest(t *testing.T, ts *httptest.Server, method, path, payload string) *http.Response {
	t.Helper()
	var body []byte
	headers := map[string]string{}
	if strings.TrimSpace(payload) != "" {
		body = []byte(payload)
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "detective")
}

func detectivePathForTest(template string) string {
	resourceARN := "arn:aws:detective:us-east-1:123456789012:graph:graph-00000001"
	out := template
	out = strings.ReplaceAll(out, "{ResourceArn}", url.PathEscape(resourceARN))
	return out
}

func TestDetectiveStage0CatalogCoverage(t *testing.T) {
	if len(detectiveOperations) != 29 {
		t.Fatalf("expected 29 Detective operations from docs, got %d", len(detectiveOperations))
	}
	if len(detectiveOperationByName) != len(detectiveOperations) {
		t.Fatalf("expected unique Detective operation names")
	}

	requiredActions := []string{
		"CreateGraph",
		"ListGraphs",
		"CreateMembers",
		"ListMembers",
		"StartInvestigation",
		"TagResource",
		"UntagResource",
	}
	for _, action := range requiredActions {
		if _, ok := detectiveOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(detectiveDataTypes) != 26 {
		t.Fatalf("expected 26 Detective data types from docs, got %d", len(detectiveDataTypes))
	}
	if len(detectiveDataTypeByName) != len(detectiveDataTypes) {
		t.Fatalf("expected unique Detective data type names")
	}
}

func TestDetectiveStage0UnknownRouteReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := detectiveRequest(t, ts, http.MethodPost, "/detective/not-a-real-route", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestDetectiveKnownRouteReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := detectiveRequest(t, ts, http.MethodPost, "/graphs/list", `{}`)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "GraphList") {
		t.Fatalf("expected ListGraphs response body to include GraphList, got %q", body)
	}
}

func TestDetectiveStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range detectiveOperations {
		path := detectivePathForTest(op.URI)
		payload := `{}`
		if op.Method == http.MethodGet || op.Method == http.MethodDelete {
			payload = ""
		}
		resp := detectiveRequest(t, ts, op.Method, path, payload)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
