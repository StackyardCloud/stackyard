package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func rtbFabricRequest(t *testing.T, ts *httptest.Server, method, path, payload string) *http.Response {
	t.Helper()
	var body []byte
	if strings.TrimSpace(payload) != "" {
		body = []byte(payload)
	}
	headers := map[string]string{}
	if len(body) > 0 {
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "rtb-fabric")
}

func TestRTBFabricStage0CatalogCoverage(t *testing.T) {
	if len(rtbFabricOperations) != 27 {
		t.Fatalf("expected 27 RTB Fabric operations from docs, got %d", len(rtbFabricOperations))
	}
	if len(rtbFabricOperationByName) != len(rtbFabricOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"CreateRequesterGateway",
		"CreateResponderGateway",
		"CreateLink",
		"AcceptLink",
		"ListRequesterGateways",
		"ListResponderGateways",
		"ListTagsForResource",
	}
	for _, action := range requiredActions {
		if _, ok := rtbFabricOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(rtbFabricDataTypes) != 21 {
		t.Fatalf("expected 21 RTB Fabric data types from docs, got %d", len(rtbFabricDataTypes))
	}
	if len(rtbFabricDataTypeByName) != len(rtbFabricDataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"Action",
		"LinkAttributes",
		"LinkLogSettings",
		"ManagedEndpointConfiguration",
		"TrustStoreConfiguration",
	}
	for _, typeName := range requiredTypes {
		if _, ok := rtbFabricDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestRTBFabricStage0UnknownRouteReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := rtbFabricRequest(t, ts, http.MethodGet, "/gateway/stackyard-gateway/unknown", "")
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestRTBFabricKnownRouteReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := rtbFabricRequest(t, ts, http.MethodGet, "/requester-gateways?maxResults=5", "")
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "gateways") {
		t.Fatalf("expected ListRequesterGateways response body to include gateways, got %q", body)
	}
}

func TestRTBFabricStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	escapedARN := url.PathEscape("arn:aws:rtb-fabric:us-east-1:123456789012:requester-gateway/stackyard-gateway")
	replacer := strings.NewReplacer(
		"{gatewayId}", "stackyard-gateway",
		"{linkId}", "stackyard-link",
		"{resourceArn}", escapedARN,
		"{maxResults}", "10",
		"{nextToken}", "token",
		"{tagKeys}", "env",
	)

	for _, op := range rtbFabricOperations {
		path := replacer.Replace(op.URI)
		payload := ""
		if op.Method == http.MethodPost || op.Method == http.MethodPatch {
			payload = `{}`
		}
		resp := rtbFabricRequest(t, ts, op.Method, path, payload)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
