package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func cloudFormationRequest(t *testing.T, ts *httptest.Server, action string, params url.Values) *http.Response {
	t.Helper()
	if params == nil {
		params = url.Values{}
	}
	params.Set("Action", action)
	params.Set("Version", "2010-05-15")
	return signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		[]byte(params.Encode()),
		map[string]string{
			"Content-Type": "application/x-www-form-urlencoded",
		},
		"cloudformation",
	)
}

func TestCloudFormationStage0CatalogCoverage(t *testing.T) {
	if len(cloudFormationOperations) != 90 {
		t.Fatalf("expected 90 CloudFormation operations from docs, got %d", len(cloudFormationOperations))
	}
	if len(cloudFormationOperationByName) != len(cloudFormationOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"CreateStack",
		"UpdateStack",
		"DeleteStack",
		"DescribeStacks",
		"CreateStackSet",
		"ListTypes",
	}
	for _, action := range requiredActions {
		if _, ok := cloudFormationOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(cloudFormationDataTypes) != 88 {
		t.Fatalf("expected 88 CloudFormation data types from docs, got %d", len(cloudFormationDataTypes))
	}
	if len(cloudFormationDataTypeByName) != len(cloudFormationDataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"Stack",
		"StackEvent",
		"StackSet",
		"StackInstance",
		"TypeSummary",
		"TemplateSummary",
	}
	for _, typeName := range requiredTypes {
		if _, ok := cloudFormationDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestCloudFormationStage0UnknownActionReturnsInvalidAction(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := cloudFormationRequest(t, ts, "TotallyUnknownAction", nil)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "InvalidAction") {
		t.Fatalf("expected InvalidAction response body, got %q", body)
	}
}

func TestCloudFormationKnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := cloudFormationRequest(t, ts, "DescribeAccountLimits", nil)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "AccountLimits") {
		t.Fatalf("expected DescribeAccountLimits response body to include AccountLimits, got %q", body)
	}
}

func TestCloudFormationStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range cloudFormationOperations {
		resp := cloudFormationRequest(t, ts, op.Name, nil)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
