package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
)

func gameliftStreamsRequest(t *testing.T, ts *httptest.Server, method, path string, body []byte) *http.Response {
	t.Helper()
	headers := map[string]string{}
	if len(body) > 0 {
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "gameliftstreams")
}

func TestGameLiftStreamsStage0CatalogCoverage(t *testing.T) {
	if len(gameliftStreamsOperations) != 24 {
		t.Fatalf("expected 24 GameLift Streams operations from docs, got %d", len(gameliftStreamsOperations))
	}
	if len(gameliftStreamsOperationByName) != len(gameliftStreamsOperations) {
		t.Fatalf("expected unique GameLift Streams operation names")
	}

	requiredActions := []string{
		"CreateApplication",
		"ListApplications",
		"CreateStreamGroup",
		"StartStreamSession",
		"CreateStreamSessionConnection",
		"ExportStreamSessionFiles",
		"TagResource",
		"ListTagsForResource",
	}
	for _, action := range requiredActions {
		if _, ok := gameliftStreamsOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(gameliftStreamsDataTypes) != 10 {
		t.Fatalf("expected 10 GameLift Streams data types from docs, got %d", len(gameliftStreamsDataTypes))
	}
	if len(gameliftStreamsDataTypeByName) != len(gameliftStreamsDataTypes) {
		t.Fatalf("expected unique GameLift Streams data type names")
	}

	requiredTypes := []string{
		"ApplicationSummary",
		"DefaultApplication",
		"LocationConfiguration",
		"PerformanceStatsConfiguration",
		"StreamSessionSummary",
	}
	for _, typeName := range requiredTypes {
		if _, ok := gameliftStreamsDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestGameLiftStreamsStage0UnknownRouteReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := gameliftStreamsRequest(t, ts, http.MethodPost, "/gameliftstreams-unknown", []byte(`{}`))
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestGameLiftStreamsStage0KnownActionReturnsListApplications(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := gameliftStreamsRequest(t, ts, http.MethodGet, "/applications?MaxResults=10", nil)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "items") {
		t.Fatalf("expected ListApplications response body to include items, got %q", body)
	}
}

func TestGameLiftStreamsStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	replacements := map[string]string{
		"identifier":              "sg-000001",
		"streamSessionIdentifier": "ss-000001",
		"resourceArn":             url.PathEscape("arn:aws:gameliftstreams:us-east-1:123456789012:streamgroup/sg-000001"),
	}
	placeholder := regexp.MustCompile(`\{([^}]+)\}`)

	for _, op := range gameliftStreamsOperations {
		path := placeholder.ReplaceAllStringFunc(op.URI, func(token string) string {
			name := strings.Trim(token, "{}")
			if value := replacements[name]; value != "" {
				return value
			}
			return "stackyard"
		})

		var body []byte
		if op.Method == http.MethodPost || op.Method == http.MethodPut || op.Method == http.MethodPatch {
			body = []byte(`{}`)
		}

		resp := gameliftStreamsRequest(t, ts, op.Method, path, body)
		respBody := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(respBody, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
		if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
	}
}
