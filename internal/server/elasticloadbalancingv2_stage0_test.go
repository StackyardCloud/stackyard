package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func elasticLoadBalancingV2Request(t *testing.T, ts *httptest.Server, action string, params url.Values) *http.Response {
	t.Helper()
	if params == nil {
		params = url.Values{}
	}
	params.Set("Action", action)
	params.Set("Version", "2012-06-01")
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

func TestElasticLoadBalancingV2Stage0CatalogCoverage(t *testing.T) {
	if len(elasticLoadBalancingV2Operations) != 29 {
		t.Fatalf("expected 29 Elastic Load Balancing v2 operations from docs, got %d", len(elasticLoadBalancingV2Operations))
	}
	if len(elasticLoadBalancingV2OperationByName) != len(elasticLoadBalancingV2Operations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"CreateLoadBalancer",
		"DescribeLoadBalancers",
		"ModifyLoadBalancerAttributes",
		"RegisterInstancesWithLoadBalancer",
		"DescribeInstanceHealth",
		"SetLoadBalancerPoliciesOfListener",
	}
	for _, action := range requiredActions {
		if _, ok := elasticLoadBalancingV2OperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(elasticLoadBalancingV2DataTypes) != 26 {
		t.Fatalf("expected 26 Elastic Load Balancing v2 data types from docs, got %d", len(elasticLoadBalancingV2DataTypes))
	}
	if len(elasticLoadBalancingV2DataTypeByName) != len(elasticLoadBalancingV2DataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"LoadBalancerDescription",
		"LoadBalancerAttributes",
		"ListenerDescription",
		"InstanceState",
		"PolicyDescription",
		"TagDescription",
	}
	for _, typeName := range requiredTypes {
		if _, ok := elasticLoadBalancingV2DataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestElasticLoadBalancingV2Stage0UnknownActionReturnsInvalidAction(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := elasticLoadBalancingV2Request(t, ts, "TotallyUnknownAction", nil)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "InvalidAction") {
		t.Fatalf("expected InvalidAction response body, got %q", body)
	}
}

func TestElasticLoadBalancingV2Stage0KnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := elasticLoadBalancingV2Request(t, ts, "DescribeLoadBalancers", nil)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "DescribeLoadBalancersResponse") {
		t.Fatalf("expected DescribeLoadBalancersResponse in body, got %q", body)
	}
}

func TestElasticLoadBalancingV2Stage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range elasticLoadBalancingV2Operations {
		resp := elasticLoadBalancingV2Request(t, ts, op.Name, nil)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
