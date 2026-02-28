package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
)

func supplyChainRequest(t *testing.T, ts *httptest.Server, method, path string, body []byte) *http.Response {
	t.Helper()
	headers := map[string]string{}
	if len(body) > 0 {
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "scn")
}

func TestSupplyChainStage0CatalogCoverage(t *testing.T) {
	if len(supplyChainOperations) != 30 {
		t.Fatalf("expected 30 Supply Chain actions from docs, got %d", len(supplyChainOperations))
	}
	if len(supplyChainOperationByName) != len(supplyChainOperations) {
		t.Fatalf("expected unique Supply Chain action names")
	}

	requiredActions := []string{
		"CreateInstance",
		"GetInstance",
		"ListInstances",
		"UpdateInstance",
		"CreateDataLakeNamespace",
		"CreateDataLakeDataset",
		"CreateDataIntegrationFlow",
		"SendDataIntegrationEvent",
		"CreateBillOfMaterialsImportJob",
		"ListTagsForResource",
	}
	for _, action := range requiredActions {
		if _, ok := supplyChainOperationByName[action]; !ok {
			t.Fatalf("missing documented action %s", action)
		}
	}

	if len(supplyChainDataTypes) != 33 {
		t.Fatalf("expected 33 Supply Chain data types from docs, got %d", len(supplyChainDataTypes))
	}
	if len(supplyChainDataTypeByName) != len(supplyChainDataTypes) {
		t.Fatalf("expected unique Supply Chain data type names")
	}

	requiredTypes := []string{
		"BillOfMaterialsImportJob",
		"DataIntegrationEvent",
		"DataIntegrationFlow",
		"DataLakeDataset",
		"DataLakeNamespace",
		"Instance",
	}
	for _, typeName := range requiredTypes {
		if _, ok := supplyChainDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestSupplyChainStage0UnknownRouteReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := supplyChainRequest(t, ts, http.MethodPost, "/supplychain/unknown", []byte(`{}`))
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestSupplyChainStage0KnownActionReturnsListInstances(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := supplyChainRequest(t, ts, http.MethodGet, "/api/instance?maxResults=10", nil)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "instances") {
		t.Fatalf("expected ListInstances response body to include instances, got %q", body)
	}
}

func TestSupplyChainStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	replacements := map[string]string{
		"instanceId":          "scn-instance-000001",
		"name":                "stackyard-resource",
		"namespace":           "default",
		"jobId":               "bom-000001",
		"eventId":             "event-000001",
		"flowName":            "orders-flow",
		"executionId":         "exec-000001",
		"eventType":           "DataSetLoad",
		"instanceNameFilter":  "stackyard",
		"instanceStateFilter": "Active",
		"maxResults":          "10",
		"nextToken":           "token-000001",
		"resourceArn":         "arn:aws:scn:us-east-1:123456789012:instance/scn-instance-000001",
		"tagKeys":             "env",
	}
	placeholder := regexp.MustCompile(`\{([^}]+)\}`)

	for _, op := range supplyChainOperations {
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

		resp := supplyChainRequest(t, ts, op.Method, path, body)
		respBody := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(respBody, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
	}
}
