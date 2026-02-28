package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
)

func entityResolutionRequest(t *testing.T, ts *httptest.Server, method, path string, body []byte) *http.Response {
	t.Helper()
	headers := map[string]string{}
	if len(body) > 0 {
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "entityresolution")
}

func TestEntityResolutionStage0CatalogCoverage(t *testing.T) {
	if len(entityResolutionOperations) != 38 {
		t.Fatalf("expected 38 Entity Resolution actions from docs, got %d", len(entityResolutionOperations))
	}
	if len(entityResolutionOperationByName) != len(entityResolutionOperations) {
		t.Fatalf("expected unique Entity Resolution action names")
	}

	requiredActions := []string{
		"CreateIdNamespace",
		"CreateMatchingWorkflow",
		"StartMatchingJob",
		"GetMatchingJob",
		"PutPolicy",
		"TagResource",
		"ListTagsForResource",
		"UntagResource",
	}
	for _, action := range requiredActions {
		if _, ok := entityResolutionOperationByName[action]; !ok {
			t.Fatalf("missing documented action %s", action)
		}
	}

	if len(entityResolutionDataTypes) != 47 {
		t.Fatalf("expected 47 Entity Resolution data types from docs, got %d", len(entityResolutionDataTypes))
	}
	if len(entityResolutionDataTypeByName) != len(entityResolutionDataTypes) {
		t.Fatalf("expected unique Entity Resolution data type names")
	}

	requiredTypes := []string{
		"IdNamespaceSummary",
		"MatchingWorkflowSummary",
		"IdMappingWorkflowSummary",
		"ProviderServiceSummary",
		"Rule",
		"RuleBasedProperties",
	}
	for _, typeName := range requiredTypes {
		if _, ok := entityResolutionDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestEntityResolutionStage0UnknownRouteReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := entityResolutionRequest(t, ts, http.MethodGet, "/entityresolution/unknown", nil)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestEntityResolutionStage0KnownActionReturnsListMatchingWorkflows(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := entityResolutionRequest(t, ts, http.MethodGet, "/matchingworkflows", nil)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "matchingWorkflows") {
		t.Fatalf("expected ListMatchingWorkflows response body to include matchingWorkflows, got %q", body)
	}
}

func TestEntityResolutionStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	replacements := map[string]string{
		"arn":                 "arn:aws:entityresolution:us-east-1:123456789012:idnamespace/stackyard-id-namespace",
		"statementId":         "statement-000001",
		"workflowName":        "stackyard-matching-workflow",
		"schemaName":          "stackyard-schema",
		"idNamespaceName":     "stackyard-id-namespace",
		"jobId":               "job-000001",
		"providerName":        "default-provider",
		"providerServiceName": "default-service",
		"resourceArn":         "arn:aws:entityresolution:us-east-1:123456789012:idnamespace/stackyard-id-namespace",
		"tagKeys":             "env",
		"maxResults":          "10",
		"nextToken":           "token-000001",
	}
	placeholder := regexp.MustCompile(`\{([^}]+)\}`)

	for _, op := range entityResolutionOperations {
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
		case "CreateIdNamespace":
			body = []byte(`{"idNamespaceName":"stage-id-namespace"}`)
		case "CreateSchemaMapping":
			body = []byte(`{"schemaName":"stage-schema"}`)
		case "CreateMatchingWorkflow":
			body = []byte(`{"workflowName":"stage-matching-workflow"}`)
		case "CreateIdMappingWorkflow":
			body = []byte(`{"workflowName":"stage-idmapping-workflow"}`)
		case "TagResource":
			body = []byte(`{"tags":{"env":"stage0"}}`)
		case "PutPolicy":
			body = []byte(`{"policy":"{\"Version\":\"2012-10-17\",\"Statement\":[]}"}`)
		default:
			if op.Method == http.MethodPost || op.Method == http.MethodPut {
				body = []byte(`{}`)
			}
		}

		resp := entityResolutionRequest(t, ts, op.Method, path, body)
		respBody := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(respBody, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
	}
}
