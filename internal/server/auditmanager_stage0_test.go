package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
)

func auditManagerRequest(t *testing.T, ts *httptest.Server, method, path, payload string) *http.Response {
	t.Helper()
	var body []byte
	headers := map[string]string{}
	if strings.TrimSpace(payload) != "" {
		body = []byte(payload)
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "auditmanager")
}

func auditManagerPathForOperation(op auditManagerOperation) string {
	path := strings.TrimSpace(strings.SplitN(op.URI, "?", 2)[0])
	re := regexp.MustCompile(`\{([^}]+)\}`)
	return re.ReplaceAllStringFunc(path, func(match string) string {
		name := strings.Trim(match, "{}")
		switch name {
		case "assessmentId":
			return "assessment-000001"
		case "frameworkId":
			return "framework-000001"
		case "requestId":
			return "request-000001"
		case "assessmentReportId":
			return "report-000001"
		case "controlId":
			return "control-000001"
		case "controlSetId":
			return "controlset-000001"
		case "evidenceFolderId":
			return "evidence-folder-000001"
		case "evidenceId":
			return "evidence-000001"
		case "attribute":
			return "all"
		case "resourceArn":
			return url.PathEscape("arn:aws:auditmanager:us-east-1:123456789012:assessment/assessment-000001")
		default:
			return "stackyard"
		}
	})
}

func TestAuditManagerStage0CatalogCoverage(t *testing.T) {
	if len(auditManagerOperations) != 62 {
		t.Fatalf("expected 62 Audit Manager operations from docs, got %d", len(auditManagerOperations))
	}
	if len(auditManagerOperationByName) != len(auditManagerOperations) {
		t.Fatalf("expected unique Audit Manager operation names")
	}

	requiredActions := []string{
		"GetAccountStatus",
		"ListAssessments",
		"CreateAssessment",
		"ListControls",
		"TagResource",
		"ValidateAssessmentReportIntegrity",
	}
	for _, action := range requiredActions {
		if _, ok := auditManagerOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(auditManagerDataTypes) != 54 {
		t.Fatalf("expected 54 Audit Manager data types from docs, got %d", len(auditManagerDataTypes))
	}
	if len(auditManagerDataTypeByName) != len(auditManagerDataTypes) {
		t.Fatalf("expected unique Audit Manager data type names")
	}
}

func TestAuditManagerStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := auditManagerRequest(t, ts, http.MethodGet, "/definitely-not-auditmanager-route", "")
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestAuditManagerStage0KnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := auditManagerRequest(t, ts, http.MethodGet, "/account/status", "")
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "status") {
		t.Fatalf("expected GetAccountStatus response body to include status, got %q", body)
	}
}

func TestAuditManagerStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range auditManagerOperations {
		payload := `{}`
		if strings.EqualFold(op.Method, http.MethodGet) || strings.EqualFold(op.Method, http.MethodDelete) {
			payload = ""
		}
		resp := auditManagerRequest(t, ts, op.Method, auditManagerPathForOperation(op), payload)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
