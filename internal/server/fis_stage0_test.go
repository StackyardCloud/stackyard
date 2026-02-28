package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
)

func fisRequest(t *testing.T, ts *httptest.Server, method, path string, body []byte) *http.Response {
	t.Helper()
	headers := map[string]string{}
	if len(body) > 0 {
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "fis")
}

func TestFISStage0CatalogCoverage(t *testing.T) {
	if len(fisOperations) != 26 {
		t.Fatalf("expected 26 FIS actions from docs, got %d", len(fisOperations))
	}
	if len(fisOperationByName) != len(fisOperations) {
		t.Fatalf("expected unique FIS action names")
	}

	requiredActions := []string{
		"CreateExperimentTemplate",
		"UpdateExperimentTemplate",
		"DeleteExperimentTemplate",
		"StartExperiment",
		"StopExperiment",
		"GetAction",
		"ListTargetResourceTypes",
		"GetSafetyLever",
		"TagResource",
		"UntagResource",
	}
	for _, action := range requiredActions {
		if _, ok := fisOperationByName[action]; !ok {
			t.Fatalf("missing documented action %s", action)
		}
	}

	if len(fisDataTypes) != 73 {
		t.Fatalf("expected 73 FIS data types from docs, got %d", len(fisDataTypes))
	}
	if len(fisDataTypeByName) != len(fisDataTypes) {
		t.Fatalf("expected unique FIS data type names")
	}

	requiredTypes := []string{
		"Action",
		"CreateExperimentTemplateActionInput",
		"Experiment",
		"ExperimentTemplate",
		"SafetyLever",
		"TargetAccountConfiguration",
		"TargetResourceType",
		"UpdateSafetyLeverStateInput",
	}
	for _, typeName := range requiredTypes {
		if _, ok := fisDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestFISStage0UnknownRouteReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := fisRequest(t, ts, http.MethodPost, "/fis/unknown", []byte(`{}`))
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestFISStage0KnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := fisRequest(t, ts, http.MethodGet, "/actions/aws%3Aec2%3Astop-instances", nil)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "action") {
		t.Fatalf("expected GetAction response to include action, got %q", body)
	}
}

func TestFISStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	replacements := map[string]string{
		"id":                   "ext-000001",
		"experimentTemplateId": "ext-000001",
		"experimentId":         "exp-000001",
		"accountId":            "123456789012",
		"resourceArn":          "arn:aws:fis:us-east-1:123456789012:experiment-template/ext-000001",
		"resourceArn+":         "arn:aws:fis:us-east-1:123456789012:experiment-template/ext-000001",
		"maxResults":           "10",
		"nextToken":            "token-000001",
		"tagKeys":              "env",
	}
	placeholder := regexp.MustCompile(`\{([^}]+)\}`)

	for _, op := range fisOperations {
		path := placeholder.ReplaceAllStringFunc(op.URI, func(token string) string {
			name := strings.Trim(token, "{}")
			value := replacements[name]
			if value == "" {
				value = replacements[strings.TrimSuffix(name, "+")]
			}
			if value == "" {
				value = "stackyard"
			}
			return url.PathEscape(value)
		})

		var body []byte
		if op.Method == http.MethodPost || op.Method == http.MethodPatch {
			body = []byte(`{}`)
		}

		resp := fisRequest(t, ts, op.Method, path, body)
		respBody := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(respBody, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
	}
}
