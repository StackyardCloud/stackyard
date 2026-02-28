package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func elasticLoadBalancingRequest(t *testing.T, ts *httptest.Server, action string, params url.Values) *http.Response {
	t.Helper()
	if params == nil {
		params = url.Values{}
	}
	params.Set("Action", action)
	params.Set("Version", "2015-12-01")
	return signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		[]byte(params.Encode()),
		map[string]string{
			"Content-Type": "application/x-www-form-urlencoded",
		},
		"elasticloadbalancing",
	)
}

func TestElasticLoadBalancingStage0CatalogCoverage(t *testing.T) {
	if len(elasticLoadBalancingOperations) != 51 {
		t.Fatalf("expected 51 Elastic Load Balancing operations from docs, got %d", len(elasticLoadBalancingOperations))
	}
	if len(elasticLoadBalancingOperationByName) != len(elasticLoadBalancingOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"CreateLoadBalancer",
		"DescribeLoadBalancers",
		"CreateTargetGroup",
		"DescribeTargetHealth",
		"CreateListener",
		"DescribeRules",
	}
	for _, action := range requiredActions {
		if _, ok := elasticLoadBalancingOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(elasticLoadBalancingDataTypes) != 57 {
		t.Fatalf("expected 57 Elastic Load Balancing data types from docs, got %d", len(elasticLoadBalancingDataTypes))
	}
	if len(elasticLoadBalancingDataTypeByName) != len(elasticLoadBalancingDataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"LoadBalancer",
		"Listener",
		"Rule",
		"TargetGroup",
		"TargetHealthDescription",
		"TrustStore",
	}
	for _, typeName := range requiredTypes {
		if _, ok := elasticLoadBalancingDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestElasticLoadBalancingStage0UnknownActionReturnsInvalidAction(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := elasticLoadBalancingRequest(t, ts, "TotallyUnknownAction", nil)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "InvalidAction") {
		t.Fatalf("expected InvalidAction response body, got %q", body)
	}
}

func TestElasticLoadBalancingStage0KnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := elasticLoadBalancingRequest(t, ts, "DescribeLoadBalancers", nil)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "DescribeLoadBalancersResponse") {
		t.Fatalf("expected DescribeLoadBalancersResponse in body, got %q", body)
	}
}

func TestElasticLoadBalancingStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range elasticLoadBalancingOperations {
		resp := elasticLoadBalancingRequest(t, ts, op.Name, nil)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
