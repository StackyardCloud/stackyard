package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func resilienceHubRequest(t *testing.T, ts *httptest.Server, method, path string, body []byte) *http.Response {
	t.Helper()
	headers := map[string]string{}
	if len(body) > 0 {
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "resiliencehub")
}

func resilienceHubPathForTest(template string) string {
	resourceARN := "arn:aws:resiliencehub:us-east-1:123456789012:app/stackyard-app"
	out := template
	out = strings.ReplaceAll(out, "{resourceArn}", url.PathEscape(resourceARN))
	return out
}

func TestResilienceHubStage0CatalogCoverage(t *testing.T) {
	if len(resilienceHubOperations) != 63 {
		t.Fatalf("expected 63 Resilience Hub operations from docs, got %d", len(resilienceHubOperations))
	}
	if len(resilienceHubOperationByName) != len(resilienceHubOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"CreateApp",
		"DescribeApp",
		"ListApps",
		"CreateResiliencyPolicy",
		"DescribeResiliencyPolicy",
		"ListTagsForResource",
	}
	for _, action := range requiredActions {
		if _, ok := resilienceHubOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(resilienceHubDataTypes) != 57 {
		t.Fatalf("expected 57 Resilience Hub data types from docs, got %d", len(resilienceHubDataTypes))
	}
	if len(resilienceHubDataTypeByName) != len(resilienceHubDataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"App",
		"AppAssessment",
		"ResiliencyPolicy",
		"RecommendationTemplate",
		"ResourceMapping",
	}
	for _, typeName := range requiredTypes {
		if _, ok := resilienceHubDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestResilienceHubUnknownRouteReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := resilienceHubRequest(t, ts, http.MethodPost, "/resiliencehub-unknown-action", []byte(`{}`))
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestResilienceHubKnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := resilienceHubRequest(t, ts, http.MethodGet, "/list-apps", nil)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "appSummaries") {
		t.Fatalf("expected ListApps response body to include appSummaries, got %q", body)
	}
}

func TestResilienceHubAllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range resilienceHubOperations {
		path := resilienceHubPathForTest(op.URI)
		var body []byte
		if op.Method == http.MethodPost {
			if op.Name == "TagResource" {
				body = []byte(`{"tags":{"stackyard":"true"}}`)
			} else {
				body = []byte(`{}`)
			}
		}

		resp := resilienceHubRequest(t, ts, op.Method, path, body)
		respBody := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(respBody, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
	}
}
