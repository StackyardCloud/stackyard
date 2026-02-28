package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func controlTowerRequest(t *testing.T, ts *httptest.Server, method, path string, body []byte) *http.Response {
	t.Helper()
	headers := map[string]string{}
	if len(body) > 0 {
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "controltower")
}

func controlTowerPathForTest(template string) string {
	resourceARN := "arn:aws:controltower:us-east-1:123456789012:landingzone/lz-000001"
	out := template
	out = strings.ReplaceAll(out, "{resourceArn}", url.PathEscape(resourceARN))
	return out
}

func TestControlTowerStage0CatalogCoverage(t *testing.T) {
	if len(controlTowerOperations) != 28 {
		t.Fatalf("expected 28 Control Tower operations from docs, got %d", len(controlTowerOperations))
	}
	if len(controlTowerOperationByName) != len(controlTowerOperations) {
		t.Fatalf("expected unique operation names")
	}
	if len(controlTowerDataTypes) != 31 {
		t.Fatalf("expected 31 Control Tower data types from docs, got %d", len(controlTowerDataTypes))
	}
	if len(controlTowerDataTypeByName) != len(controlTowerDataTypes) {
		t.Fatalf("expected unique data type names")
	}
}

func TestControlTowerUnknownRouteReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := controlTowerRequest(t, ts, http.MethodPost, "/controltower-unknown-action", []byte(`{}`))
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestControlTowerAllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range controlTowerOperations {
		path := controlTowerPathForTest(op.URI)
		var body []byte
		switch op.Name {
		case "TagResource":
			body = []byte(`{"tags":{"stackyard":"true"}}`)
		default:
			if op.Method == http.MethodPost || op.Method == http.MethodPatch || op.Method == http.MethodPut {
				body = []byte(`{}`)
			}
		}

		resp := controlTowerRequest(t, ts, op.Method, path, body)
		respBody := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(respBody, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
	}
}
