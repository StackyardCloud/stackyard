package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
)

func vpcLatticeRequest(t *testing.T, ts *httptest.Server, method, path string, body []byte) *http.Response {
	t.Helper()
	headers := map[string]string{}
	if len(body) > 0 {
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "vpc-lattice")
}

func vpcLatticePathForTest(template string) string {
	resourceARN := url.PathEscape("arn:aws:vpc-lattice:us-east-1:123456789012:service/svc-00000000000000001")
	path := strings.NewReplacer(
		"{accessLogSubscriptionIdentifier}", "als-00000000000000001",
		"{domainVerificationIdentifier}", "dv-00000000000000001",
		"{listenerIdentifier}", "listener-00000000000000001",
		"{maxResults}", "25",
		"{nextToken}", "",
		"{resourceArn}", resourceARN,
		"{resourceConfigurationGroupIdentifier}", "rcfg-group-00000000000000001",
		"{resourceConfigurationIdentifier}", "rcfg-00000000000000001",
		"{resourceEndpointAssociationIdentifier}", "rea-00000000000000001",
		"{resourceGatewayIdentifier}", "rgw-00000000000000001",
		"{resourceIdentifier}", "svc-00000000000000001",
		"{ruleIdentifier}", "rule-00000000000000001",
		"{serviceIdentifier}", "svc-00000000000000001",
		"{serviceNetworkIdentifier}", "sn-00000000000000001",
		"{serviceNetworkResourceAssociationIdentifier}", "snra-00000000000000001",
		"{serviceNetworkServiceAssociationIdentifier}", "snsa-00000000000000001",
		"{serviceNetworkVpcAssociationIdentifier}", "snva-00000000000000001",
		"{tagKeys}", "env",
		"{targetGroupIdentifier}", "tg-00000000000000001",
		"{targetGroupType}", "IP",
		"{vpcEndpointId}", "vpce-00000000000000001",
		"{vpcEndpointOwner}", "123456789012",
		"{vpcIdentifier}", "vpc-00000000000000001",
		"{includeChildren}", "false",
	).Replace(template)
	if strings.Contains(path, "{") {
		path = regexp.MustCompile(`\{[^}]+\}`).ReplaceAllString(path, "stackyard")
	}
	return path
}

func TestVPCLatticeStage0CatalogCoverage(t *testing.T) {
	if len(vpcLatticeOperations) != 73 {
		t.Fatalf("expected 73 VPC Lattice operations from docs, got %d", len(vpcLatticeOperations))
	}
	if len(vpcLatticeOperationByName) != len(vpcLatticeOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"CreateService",
		"GetService",
		"ListServices",
		"CreateTargetGroup",
		"ListTargets",
		"PutAuthPolicy",
		"TagResource",
	}
	for _, action := range requiredActions {
		if _, ok := vpcLatticeOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(vpcLatticeDataTypes) != 42 {
		t.Fatalf("expected 42 VPC Lattice data types from docs, got %d", len(vpcLatticeDataTypes))
	}
	if len(vpcLatticeDataTypeByName) != len(vpcLatticeDataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"ServiceSummary",
		"ListenerSummary",
		"RuleSummary",
		"TargetGroupSummary",
		"ResourceGatewaySummary",
		"ServiceNetworkSummary",
	}
	for _, typeName := range requiredTypes {
		if _, ok := vpcLatticeDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestVPCLatticeStage0UnknownRouteReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := vpcLatticeRequest(t, ts, http.MethodGet, "/services/svc-00000000000000001/unknown", nil)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestVPCLatticeKnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := vpcLatticeRequest(t, ts, http.MethodGet, "/services", nil)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "items") {
		t.Fatalf("expected ListServices response body to include items, got %q", body)
	}
}

func TestVPCLatticeStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range vpcLatticeOperations {
		path := vpcLatticePathForTest(op.URI)
		var body []byte
		if op.Method == http.MethodPost || op.Method == http.MethodPatch || op.Method == http.MethodPut {
			body = []byte(`{}`)
		}
		resp := vpcLatticeRequest(t, ts, op.Method, path, body)
		respBody := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(respBody, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
	}
}
