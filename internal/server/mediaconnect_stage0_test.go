package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func mediaConnectRequest(t *testing.T, ts *httptest.Server, method, path, payload string) *http.Response {
	t.Helper()
	var body []byte
	headers := map[string]string{}
	if strings.TrimSpace(payload) != "" {
		body = []byte(payload)
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "mediaconnect")
}

func mediaConnectPathForOperation(op mediaConnectOperation) string {
	path := op.URI
	replacements := map[string]string{
		"{FlowArn}":            url.PathEscape("arn:aws:mediaconnect:us-east-1:123456789012:flow/flow-00000001"),
		"{BridgeArn}":          url.PathEscape("arn:aws:mediaconnect:us-east-1:123456789012:bridge/bridge-00000001"),
		"{GatewayArn}":         url.PathEscape("arn:aws:mediaconnect:us-east-1:123456789012:gateway/gateway-00000001"),
		"{GatewayInstanceArn}": url.PathEscape("arn:aws:mediaconnect:us-east-1:123456789012:gateway-instance/gateway-instance-00000001"),
		"{RouterInputArn}":     url.PathEscape("arn:aws:mediaconnect:us-east-1:123456789012:router-input/router-input-00000001"),
		"{RouterOutputArn}":    url.PathEscape("arn:aws:mediaconnect:us-east-1:123456789012:router-output/router-output-00000001"),
		"{EntitlementArn}":     url.PathEscape("arn:aws:mediaconnect:us-east-1:123456789012:entitlement/entitlement-00000001"),
		"{OfferingArn}":        url.PathEscape("arn:aws:mediaconnect:us-east-1:123456789012:offering/offering-00000001"),
		"{ReservationArn}":     url.PathEscape("arn:aws:mediaconnect:us-east-1:123456789012:reservation/reservation-00000001"),
		"{ResourceArn}":        url.PathEscape("arn:aws:mediaconnect:us-east-1:123456789012:flow/flow-00000001"),
		"{Arn}":                url.PathEscape("arn:aws:mediaconnect:us-east-1:123456789012:router-interface/router-interface-00000001"),
		"{SourceName}":         "primary",
		"{OutputArn}":          url.PathEscape("arn:aws:mediaconnect:us-east-1:123456789012:flow-output/output-00000001"),
		"{OutputName}":         "output-00000001",
		"{SourceArn}":          url.PathEscape("arn:aws:mediaconnect:us-east-1:123456789012:flow-source/source-00000001"),
		"{MediaStreamName}":    "stream-00000001",
		"{VpcInterfaceName}":   "vpcif-00000001",
	}
	for key, value := range replacements {
		path = strings.ReplaceAll(path, key, value)
	}
	return path
}

func TestMediaConnectStage0CatalogCoverage(t *testing.T) {
	if len(mediaConnectOperations) != 82 {
		t.Fatalf("expected 82 MediaConnect operations from docs, got %d", len(mediaConnectOperations))
	}
	if len(mediaConnectOperationByName) != len(mediaConnectOperations) {
		t.Fatalf("expected unique MediaConnect operation names")
	}

	requiredActions := []string{
		"CreateFlow",
		"DescribeFlow",
		"ListFlows",
		"UpdateFlow",
		"DeleteFlow",
		"StartFlow",
		"StopFlow",
		"TagResource",
		"UntagResource",
		"ListTagsForResource",
	}
	for _, action := range requiredActions {
		if _, ok := mediaConnectOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(mediaConnectDataTypes) != 163 {
		t.Fatalf("expected 163 MediaConnect data types from docs, got %d", len(mediaConnectDataTypes))
	}
	if len(mediaConnectDataTypeByName) != len(mediaConnectDataTypes) {
		t.Fatalf("expected unique MediaConnect data type names")
	}
}

func TestMediaConnectStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := mediaConnectRequest(t, ts, http.MethodGet, "/v1/not-a-real-mediaconnect-route", "")
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestMediaConnectStage0KnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := mediaConnectRequest(t, ts, http.MethodGet, "/v1/flows", "")
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "Flows") {
		t.Fatalf("expected ListFlows response body to include Flows, got %q", body)
	}
}

func TestMediaConnectStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range mediaConnectOperations {
		payload := `{}`
		if strings.EqualFold(op.Method, http.MethodGet) || strings.EqualFold(op.Method, http.MethodDelete) {
			payload = ""
		}
		resp := mediaConnectRequest(t, ts, op.Method, mediaConnectPathForOperation(op), payload)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
