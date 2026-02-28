package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
)

func simSpaceWeaverRequest(t *testing.T, ts *httptest.Server, method, path string, body []byte) *http.Response {
	t.Helper()
	headers := map[string]string{}
	if len(body) > 0 {
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "simspaceweaver")
}

func TestSimSpaceWeaverStage0CatalogCoverage(t *testing.T) {
	if len(simSpaceWeaverOperations) != 16 {
		t.Fatalf("expected 16 SimSpace Weaver actions from docs, got %d", len(simSpaceWeaverOperations))
	}
	if len(simSpaceWeaverOperationByName) != len(simSpaceWeaverOperations) {
		t.Fatalf("expected unique SimSpace Weaver action names")
	}

	requiredActions := []string{
		"CreateSnapshot",
		"DeleteApp",
		"DeleteSimulation",
		"DescribeApp",
		"DescribeSimulation",
		"ListApps",
		"ListSimulations",
		"ListTagsForResource",
		"StartApp",
		"StartClock",
		"StartSimulation",
		"StopApp",
		"StopClock",
		"StopSimulation",
		"TagResource",
		"UntagResource",
	}
	for _, action := range requiredActions {
		if _, ok := simSpaceWeaverOperationByName[action]; !ok {
			t.Fatalf("missing documented action %s", action)
		}
	}

	if len(simSpaceWeaverDataTypes) != 13 {
		t.Fatalf("expected 13 SimSpace Weaver data types from docs, got %d", len(simSpaceWeaverDataTypes))
	}
	if len(simSpaceWeaverDataTypeByName) != len(simSpaceWeaverDataTypes) {
		t.Fatalf("expected unique SimSpace Weaver data type names")
	}

	requiredTypes := []string{
		"Domain",
		"LiveSimulationState",
		"LoggingConfiguration",
		"S3Location",
		"SimulationAppEndpointInfo",
		"SimulationMetadata",
	}
	for _, typeName := range requiredTypes {
		if _, ok := simSpaceWeaverDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestSimSpaceWeaverStage0UnknownRouteReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := simSpaceWeaverRequest(t, ts, http.MethodPost, "/simspaceweaver/unknown", []byte(`{}`))
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestSimSpaceWeaverStage0KnownActionReturnsListSimulations(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := simSpaceWeaverRequest(t, ts, http.MethodGet, "/listsimulations?maxResults=10", nil)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "simulations") {
		t.Fatalf("expected ListSimulations response body to include simulations, got %q", body)
	}
}

func TestSimSpaceWeaverStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	replacements := map[string]string{
		"app":         "stackyard-app-0001",
		"domain":      "stackyard-domain",
		"simulation":  "stackyard-sim-0001",
		"ResourceArn": "arn:aws:simspaceweaver:us-east-1:123456789012:simulation/stackyard-sim-0001",
		"tagKeys":     "env",
		"maxResults":  "10",
		"nextToken":   "token-000001",
	}
	placeholder := regexp.MustCompile(`\{([^}]+)\}`)

	for _, op := range simSpaceWeaverOperations {
		path := placeholder.ReplaceAllStringFunc(op.URI, func(token string) string {
			name := strings.Trim(token, "{}")
			value := replacements[name]
			if value == "" {
				value = "stackyard"
			}
			return url.PathEscape(value)
		})

		var body []byte
		if op.Method == http.MethodPost || op.Method == http.MethodPut || op.Method == http.MethodPatch {
			body = []byte(`{}`)
		}

		resp := simSpaceWeaverRequest(t, ts, op.Method, path, body)
		respBody := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(respBody, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
	}
}
