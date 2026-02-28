package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func resourceExplorer2Request(t *testing.T, ts *httptest.Server, method, path string, body []byte) *http.Response {
	t.Helper()
	headers := map[string]string{}
	if len(body) > 0 {
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "resource-explorer-2")
}

func resourceExplorer2PathForTest(template string) string {
	resourceARN := "arn:aws:resource-explorer-2:us-east-1:123456789012:view/stackyard-default/00000000-0000-0000-0000-000000000000"
	out := template
	out = strings.ReplaceAll(out, "{resourceArn}", url.PathEscape(resourceARN))
	return out
}

func TestResourceExplorer2Stage0CatalogCoverage(t *testing.T) {
	if len(resourceExplorer2Operations) != 32 {
		t.Fatalf("expected 32 Resource Explorer operations from docs, got %d", len(resourceExplorer2Operations))
	}
	if len(resourceExplorer2OperationByName) != len(resourceExplorer2Operations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"CreateIndex",
		"GetIndex",
		"ListViews",
		"Search",
		"TagResource",
		"UntagResource",
	}
	for _, action := range requiredActions {
		if _, ok := resourceExplorer2OperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(resourceExplorer2DataTypes) != 20 {
		t.Fatalf("expected 20 Resource Explorer data types from docs, got %d", len(resourceExplorer2DataTypes))
	}
	if len(resourceExplorer2DataTypeByName) != len(resourceExplorer2DataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"Index",
		"View",
		"ManagedView",
		"Resource",
		"SearchFilter",
	}
	for _, typeName := range requiredTypes {
		if _, ok := resourceExplorer2DataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestResourceExplorer2UnknownRouteReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := resourceExplorer2Request(t, ts, http.MethodPost, "/UnknownAction", []byte(`{}`))
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestResourceExplorer2KnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := resourceExplorer2Request(t, ts, http.MethodPost, "/ListViews", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "Views") {
		t.Fatalf("expected ListViews response body to include Views, got %q", body)
	}
}

func TestResourceExplorer2AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range resourceExplorer2Operations {
		path := resourceExplorer2PathForTest(op.URI)
		var body []byte
		if op.Method == http.MethodPost {
			if op.Name == "TagResource" {
				body = []byte(`{"Tags":{"stackyard":"true"}}`)
			} else {
				body = []byte(`{}`)
			}
		}

		resp := resourceExplorer2Request(t, ts, op.Method, path, body)
		respBody := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(respBody, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
	}
}
