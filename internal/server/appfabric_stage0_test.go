package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
)

func appFabricRequest(t *testing.T, ts *httptest.Server, method, path string, body []byte) *http.Response {
	t.Helper()
	headers := map[string]string{}
	if len(body) > 0 {
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "appfabric")
}

func TestAppFabricStage0CatalogCoverage(t *testing.T) {
	if len(appFabricOperations) != 26 {
		t.Fatalf("expected 26 AppFabric actions from docs, got %d", len(appFabricOperations))
	}
	if len(appFabricOperationByName) != len(appFabricOperations) {
		t.Fatalf("expected unique AppFabric action names")
	}

	requiredActions := []string{
		"CreateAppBundle",
		"GetAppBundle",
		"ListAppBundles",
		"CreateAppAuthorization",
		"ConnectAppAuthorization",
		"CreateIngestion",
		"CreateIngestionDestination",
		"StartUserAccessTasks",
		"BatchGetUserAccessTasks",
		"TagResource",
		"ListTagsForResource",
		"UntagResource",
	}
	for _, action := range requiredActions {
		if _, ok := appFabricOperationByName[action]; !ok {
			t.Fatalf("missing documented action %s", action)
		}
	}

	if len(appFabricDataTypes) != 26 {
		t.Fatalf("expected 26 AppFabric data types from docs, got %d", len(appFabricDataTypes))
	}
	if len(appFabricDataTypeByName) != len(appFabricDataTypes) {
		t.Fatalf("expected unique AppFabric data type names")
	}

	requiredTypes := []string{
		"AppBundle",
		"AppAuthorization",
		"Ingestion",
		"IngestionDestination",
		"UserAccessResultItem",
		"ValidationExceptionField",
	}
	for _, typeName := range requiredTypes {
		if _, ok := appFabricDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestAppFabricStage0UnknownRouteReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := appFabricRequest(t, ts, http.MethodGet, "/appfabric/unknown", nil)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestAppFabricStage0KnownActionReturnsListAppBundles(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := appFabricRequest(t, ts, http.MethodGet, "/appbundles", nil)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "appBundles") {
		t.Fatalf("expected ListAppBundles response body to include appBundles, got %q", body)
	}
}

func TestAppFabricStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	replacements := map[string]string{
		"appBundleIdentifier":            "ab-000001",
		"appAuthorizationIdentifier":     "auth-000001",
		"ingestionIdentifier":            "ing-000001",
		"ingestionDestinationIdentifier": "dest-000001",
		"resourceArn":                    "arn:aws:appfabric:us-east-1:123456789012:appbundle/ab-000001",
	}
	placeholder := regexp.MustCompile(`\{([^}]+)\}`)

	for _, op := range appFabricOperations {
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
		case "CreateAppBundle":
			body = []byte(`{"appBundleIdentifier":"ab-stage0-001"}`)
		case "CreateAppAuthorization":
			body = []byte(`{"appAuthorizationIdentifier":"auth-stage0-001"}`)
		case "CreateIngestion":
			body = []byte(`{"ingestionIdentifier":"ing-stage0-001"}`)
		case "CreateIngestionDestination":
			body = []byte(`{"ingestionDestinationIdentifier":"dest-stage0-001"}`)
		case "TagResource":
			body = []byte(`{"tags":{"env":"stage0"}}`)
		case "UntagResource":
			path += "?tagKeys=env"
		default:
			if op.Method == http.MethodPost || op.Method == http.MethodPatch || op.Method == http.MethodPut {
				body = []byte(`{}`)
			}
		}

		resp := appFabricRequest(t, ts, op.Method, path, body)
		respBody := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(respBody, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
	}
}
