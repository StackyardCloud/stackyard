package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func mpaRequest(t *testing.T, ts *httptest.Server, method, path string, body []byte) *http.Response {
	t.Helper()
	headers := map[string]string{}
	if len(body) > 0 {
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "mpa")
}

func mpaPathForTest(template string) string {
	approvalTeamArn := "arn:aws:mpa:us-east-1:123456789012:approval-team/stackyard"
	identitySourceArn := "arn:aws:mpa:us-east-1:123456789012:identity-source/stackyard"
	sessionArn := "arn:aws:mpa:us-east-1:123456789012:session/stackyard"
	policyArn := "arn:aws:mpa:us-east-1:123456789012:policy/stackyard"
	policyVersionArn := "arn:aws:mpa:us-east-1:123456789012:policy-version/stackyard/1"
	resourceArn := "arn:aws:mpa:us-east-1:123456789012:resource/stackyard"

	out := template
	out = strings.ReplaceAll(out, "{SessionArn}", url.PathEscape(sessionArn))
	out = strings.ReplaceAll(out, "{IdentitySourceArn}", url.PathEscape(identitySourceArn))
	out = strings.ReplaceAll(out, "{Arn}", url.PathEscape(approvalTeamArn))
	out = strings.ReplaceAll(out, "{VersionId}", "1")
	out = strings.ReplaceAll(out, "{PolicyVersionArn}", url.PathEscape(policyVersionArn))
	out = strings.ReplaceAll(out, "{PolicyArn}", url.PathEscape(policyArn))
	out = strings.ReplaceAll(out, "{ResourceArn}", url.PathEscape(resourceArn))
	out = strings.ReplaceAll(out, "{ApprovalTeamArn}", url.PathEscape(approvalTeamArn))
	return out
}

func TestMPAStage0CatalogCoverage(t *testing.T) {
	if len(mpaOperations) != 21 {
		t.Fatalf("expected 21 MPA operations from docs, got %d", len(mpaOperations))
	}
	if len(mpaOperationByName) != len(mpaOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"CreateApprovalTeam",
		"ListApprovalTeams",
		"GetSession",
		"ListPolicyVersions",
		"GetResourcePolicy",
		"TagResource",
	}
	for _, action := range requiredActions {
		if _, ok := mpaOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(mpaDataTypes) != 23 {
		t.Fatalf("expected 23 MPA data types from docs, got %d", len(mpaDataTypes))
	}
	if len(mpaDataTypeByName) != len(mpaDataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"ApprovalStrategy",
		"ListApprovalTeamsResponseApprovalTeam",
		"ListSessionsResponseSession",
		"PolicyVersion",
		"UpdateApprovalTeam",
	}
	for _, typeName := range requiredTypes {
		if _, ok := mpaDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestMPAStage0UnknownRouteReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := mpaRequest(t, ts, http.MethodGet, "/mpa/UnknownAction", nil)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestMPAKnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := mpaRequest(t, ts, http.MethodPost, "/approval-teams/", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "ApprovalTeams") {
		t.Fatalf("expected ListApprovalTeams response body to include ApprovalTeams, got %q", body)
	}
}

func TestMPAAllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range mpaOperations {
		path := mpaPathForTest(op.URI)
		var body []byte
		if op.Method == http.MethodPost || op.Method == http.MethodPut || op.Method == http.MethodPatch {
			body = []byte(`{}`)
		}
		resp := mpaRequest(t, ts, op.Method, path, body)
		respBody := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(respBody, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
	}
}
