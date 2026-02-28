package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
)

func tnbRequest(t *testing.T, ts *httptest.Server, method, path string, body []byte) *http.Response {
	t.Helper()
	headers := map[string]string{}
	if len(body) > 0 {
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "tnb")
}

func TestTNBStage0CatalogCoverage(t *testing.T) {
	if len(tnbOperations) != 33 {
		t.Fatalf("expected 33 TNB actions from docs, got %d", len(tnbOperations))
	}
	if len(tnbOperationByName) != len(tnbOperations) {
		t.Fatalf("expected unique TNB action names")
	}

	requiredActions := []string{
		"CreateSolNetworkPackage",
		"CreateSolFunctionPackage",
		"CreateSolNetworkInstance",
		"InstantiateSolNetworkInstance",
		"GetSolNetworkOperation",
		"ListSolNetworkPackages",
		"ListSolFunctionPackages",
		"ListSolNetworkInstances",
		"TagResource",
		"ListTagsForResource",
	}
	for _, action := range requiredActions {
		if _, ok := tnbOperationByName[action]; !ok {
			t.Fatalf("missing documented action %s", action)
		}
	}

	if len(tnbDataTypes) != 35 {
		t.Fatalf("expected 35 TNB data types from docs, got %d", len(tnbDataTypes))
	}
	if len(tnbDataTypeByName) != len(tnbDataTypes) {
		t.Fatalf("expected unique TNB data type names")
	}

	requiredTypes := []string{
		"LcmOperationInfo",
		"ListSolNetworkPackageInfo",
		"ListSolFunctionPackageInfo",
		"ListSolNetworkInstanceInfo",
		"GetSolNetworkOperationMetadata",
	}
	for _, typeName := range requiredTypes {
		if _, ok := tnbDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestTNBStage0UnknownRouteReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := tnbRequest(t, ts, http.MethodPost, "/tnb/unknown", []byte(`{}`))
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestTNBStage0KnownActionReturnsList(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := tnbRequest(t, ts, http.MethodGet, "/sol/nsd/v1/ns_descriptors?max_results=10", nil)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "networkPackages") {
		t.Fatalf("expected ListSolNetworkPackages response body to include networkPackages, got %q", body)
	}
}

func TestTNBStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	replacements := map[string]string{
		"nsLcmOpOccId":  "ns-lcm-op-000001",
		"vnfPkgId":      "vnfpkg-000001",
		"nsInstanceId":  "ns-instance-000001",
		"nsdInfoId":     "nsd-000001",
		"vnfInstanceId": "vnf-instance-000001",
		"resourceArn":   "arn:aws:tnb:us-east-1:123456789012:nsd/nsd-000001",
		"tagKeys":       "env",
		"maxResults":    "10",
		"nextToken":     "token-000001",
		"dryRun":        "false",
	}
	placeholder := regexp.MustCompile(`\{([^}]+)\}`)

	for _, op := range tnbOperations {
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

		resp := tnbRequest(t, ts, op.Method, path, body)
		respBody := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(respBody, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
	}
}
