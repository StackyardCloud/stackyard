package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
)

func elementalInferenceRequest(t *testing.T, ts *httptest.Server, method, path string, body []byte) *http.Response {
	t.Helper()
	headers := map[string]string{}
	if len(body) > 0 {
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "elemental-inference")
}

func TestElementalInferenceStage0CatalogCoverage(t *testing.T) {
	if len(elementalInferenceOperations) != 10 {
		t.Fatalf("expected 10 Elemental Inference actions from docs, got %d", len(elementalInferenceOperations))
	}
	if len(elementalInferenceOperationByName) != len(elementalInferenceOperations) {
		t.Fatalf("expected unique Elemental Inference action names")
	}

	requiredActions := []string{
		"AssociateFeed",
		"CreateFeed",
		"DeleteFeed",
		"DisassociateFeed",
		"GetFeed",
		"ListFeeds",
		"ListTagsForResource",
		"TagResource",
		"UntagResource",
		"UpdateFeed",
	}
	for _, action := range requiredActions {
		if _, ok := elementalInferenceOperationByName[action]; !ok {
			t.Fatalf("missing documented action %s", action)
		}
	}

	if len(elementalInferenceDataTypes) != 9 {
		t.Fatalf("expected 9 Elemental Inference data types from docs, got %d", len(elementalInferenceDataTypes))
	}
	if len(elementalInferenceDataTypeByName) != len(elementalInferenceDataTypes) {
		t.Fatalf("expected unique Elemental Inference data type names")
	}

	requiredTypes := []string{
		"CreateOutput",
		"FeedAssociation",
		"FeedSummary",
		"GetOutput",
		"OutputConfig",
		"UpdateOutput",
	}
	for _, typeName := range requiredTypes {
		if _, ok := elementalInferenceDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestElementalInferenceStage0UnknownRouteReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := elementalInferenceRequest(t, ts, http.MethodPost, "/v1/unknown", []byte(`{}`))
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestElementalInferenceStage0KnownActionReturnsListFeeds(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := elementalInferenceRequest(t, ts, http.MethodGet, "/v1/feeds", nil)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "feeds") {
		t.Fatalf("expected ListFeeds response body to include feeds, got %q", body)
	}
}

func TestElementalInferenceStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	replacements := map[string]string{
		"id":          "feed-000001",
		"resourceArn": "arn:aws:elemental-inference:us-east-1:123456789012:feed/feed-000001",
	}
	placeholder := regexp.MustCompile(`\{([^}]+)\}`)

	for _, op := range elementalInferenceOperations {
		path := placeholder.ReplaceAllStringFunc(op.URI, func(token string) string {
			key := strings.Trim(token, "{}")
			value := replacements[key]
			if value == "" {
				value = "stackyard"
			}
			return url.PathEscape(value)
		})

		var body []byte
		switch op.Name {
		case "CreateFeed":
			body = []byte(`{"name":"stage-feed-stage0"}`)
		case "UpdateFeed":
			body = []byte(`{"name":"stage-feed-updated-stage0"}`)
		case "AssociateFeed":
			body = []byte(`{"flowArn":"arn:aws:mediaconnect:us-east-1:123456789012:flow/flow-000001"}`)
		case "DisassociateFeed":
			body = []byte(`{}`)
		case "TagResource":
			body = []byte(`{"tags":{"env":"stage0"}}`)
		default:
			if op.Method == http.MethodPost || op.Method == http.MethodPut {
				body = []byte(`{}`)
			}
		}

		resp := elementalInferenceRequest(t, ts, op.Method, path, body)
		respBody := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(respBody, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
	}
}
