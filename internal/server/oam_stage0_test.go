package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func oamRequest(t *testing.T, ts *httptest.Server, method, path string, body []byte) *http.Response {
	t.Helper()
	headers := map[string]string{}
	if len(body) > 0 {
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "oam")
}

func oamPathForTest(template string) string {
	resourceARN := "arn:aws:oam:us-east-1:123456789012:sink/stackyard-sink"
	out := template
	out = strings.ReplaceAll(out, "{ResourceArn}", url.PathEscape(resourceARN))
	return out
}

func TestOAMStage0CatalogCoverage(t *testing.T) {
	if len(oamOperations) != 15 {
		t.Fatalf("expected 15 OAM operations from docs, got %d", len(oamOperations))
	}
	if len(oamOperationByName) != len(oamOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"CreateLink",
		"CreateSink",
		"GetLink",
		"ListSinks",
		"PutSinkPolicy",
		"TagResource",
	}
	for _, action := range requiredActions {
		if _, ok := oamOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(oamDataTypes) != 6 {
		t.Fatalf("expected 6 OAM data types from docs, got %d", len(oamDataTypes))
	}
	if len(oamDataTypeByName) != len(oamDataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"LinkConfiguration",
		"ListAttachedLinksItem",
		"ListLinksItem",
		"ListSinksItem",
		"LogGroupConfiguration",
		"MetricConfiguration",
	}
	for _, typeName := range requiredTypes {
		if _, ok := oamDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestOAMStage0UnknownRouteReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := oamRequest(t, ts, http.MethodPost, "/UnknownAction", []byte(`{}`))
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestOAMKnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := oamRequest(t, ts, http.MethodPost, "/ListSinks", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "Items") {
		t.Fatalf("expected ListSinks response body to include Items, got %q", body)
	}
}

func TestOAMAllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range oamOperations {
		path := oamPathForTest(op.URI)
		var body []byte
		if op.Method == http.MethodPost || op.Method == http.MethodPut || op.Method == http.MethodPatch {
			body = []byte(`{}`)
		}
		resp := oamRequest(t, ts, op.Method, path, body)
		respBody := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(respBody, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
	}
}
