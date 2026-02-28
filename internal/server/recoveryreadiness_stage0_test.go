package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func recoveryReadinessRequest(t *testing.T, ts *httptest.Server, method, path, payload string) *http.Response {
	t.Helper()
	var body []byte
	headers := map[string]string{}
	if strings.TrimSpace(payload) != "" {
		body = []byte(payload)
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "route53-recovery-readiness")
}

func TestRecoveryReadinessStage0CatalogCoverage(t *testing.T) {
	if len(recoveryReadinessOperations) != 32 {
		t.Fatalf("expected 32 Recovery Readiness operations from docs, got %d", len(recoveryReadinessOperations))
	}
	if len(recoveryReadinessOperationByName) != len(recoveryReadinessOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"CreateCell",
		"GetCell",
		"ListCells",
		"CreateReadinessCheck",
		"GetReadinessCheckStatus",
		"ListRecoveryGroups",
		"ListRules",
		"TagResource",
		"UntagResource",
	}
	for _, action := range requiredActions {
		if _, ok := recoveryReadinessOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(recoveryReadinessDataTypes) != 22 {
		t.Fatalf("expected 22 Recovery Readiness data types from docs, got %d", len(recoveryReadinessDataTypes))
	}
	if len(recoveryReadinessDataTypeByName) != len(recoveryReadinessDataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"CellOutput",
		"ReadinessCheckOutput",
		"RecoveryGroupOutput",
		"ResourceSetOutput",
		"RuleResult",
		"Recommendation",
	}
	for _, typeName := range requiredTypes {
		if _, ok := recoveryReadinessDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestRecoveryReadinessStage0UnknownRouteReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := recoveryReadinessRequest(t, ts, http.MethodGet, "/cells/stackyard-cell/unknown", "")
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestRecoveryReadinessKnownRouteReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := recoveryReadinessRequest(t, ts, http.MethodGet, "/cells", "")
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "Cells") {
		t.Fatalf("expected ListCells response body to include Cells, got %q", body)
	}
}

func TestRecoveryReadinessStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	replacer := strings.NewReplacer(
		"{CellName}", "stackyard-cell",
		"{CrossAccountAuthorization}", "123456789012",
		"{ReadinessCheckName}", "stackyard-readiness-check",
		"{ResourceIdentifier}", "stackyard-resource",
		"{RecoveryGroupName}", "stackyard-recovery-group",
		"{ResourceSetName}", "stackyard-resource-set",
		"{ResourceArn}", "arn:aws:route53-recovery-readiness:us-east-1:123456789012:resource:stackyard",
	)

	for _, op := range recoveryReadinessOperations {
		path := replacer.Replace(op.URI)
		payload := ""
		if op.Method == http.MethodPost || op.Method == http.MethodPut {
			payload = `{}`
		}
		resp := recoveryReadinessRequest(t, ts, op.Method, path, payload)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
