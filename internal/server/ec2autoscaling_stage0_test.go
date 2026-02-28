package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func ec2AutoScalingRequest(t *testing.T, ts *httptest.Server, action string, params url.Values) *http.Response {
	t.Helper()
	if params == nil {
		params = url.Values{}
	}
	params.Set("Action", action)
	params.Set("Version", "2011-01-01")
	return signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		[]byte(params.Encode()),
		map[string]string{
			"Content-Type": "application/x-www-form-urlencoded",
		},
		"autoscaling",
	)
}

func TestEC2AutoScalingStage0CatalogCoverage(t *testing.T) {
	if len(ec2AutoScalingOperations) != 66 {
		t.Fatalf("expected 66 EC2 Auto Scaling operations from docs, got %d", len(ec2AutoScalingOperations))
	}
	if len(ec2AutoScalingOperationByName) != len(ec2AutoScalingOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"CreateAutoScalingGroup",
		"DescribeAutoScalingGroups",
		"SetDesiredCapacity",
		"PutScalingPolicy",
		"StartInstanceRefresh",
		"DescribeScalingActivities",
	}
	for _, action := range requiredActions {
		if _, ok := ec2AutoScalingOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(ec2AutoScalingDataTypes) != 88 {
		t.Fatalf("expected 88 EC2 Auto Scaling data types from docs, got %d", len(ec2AutoScalingDataTypes))
	}
	if len(ec2AutoScalingDataTypeByName) != len(ec2AutoScalingDataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"AutoScalingGroup",
		"LaunchConfiguration",
		"ScalingPolicy",
		"InstanceRefresh",
		"ScheduledUpdateGroupAction",
	}
	for _, typeName := range requiredTypes {
		if _, ok := ec2AutoScalingDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestEC2AutoScalingStage0UnknownActionReturnsInvalidAction(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := ec2AutoScalingRequest(t, ts, "TotallyUnknownAction", nil)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "InvalidAction") {
		t.Fatalf("expected InvalidAction response body, got %q", body)
	}
}

func TestEC2AutoScalingKnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := ec2AutoScalingRequest(t, ts, "DescribeAutoScalingGroups", nil)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "DescribeAutoScalingGroupsResponse") {
		t.Fatalf("expected DescribeAutoScalingGroupsResponse in body, got %q", body)
	}
}

func TestEC2AutoScalingAllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range ec2AutoScalingOperations {
		resp := ec2AutoScalingRequest(t, ts, op.Name, nil)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
