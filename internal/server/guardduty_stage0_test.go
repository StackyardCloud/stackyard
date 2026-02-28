package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
)

func guardDutyRequest(t *testing.T, ts *httptest.Server, method, path, payload string) *http.Response {
	t.Helper()
	var body []byte
	headers := map[string]string{}
	if strings.TrimSpace(payload) != "" {
		body = []byte(payload)
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "guardduty")
}

func guardDutyPathForOperation(op guardDutyOperation) string {
	re := regexp.MustCompile(`\{([^}]+)\}`)
	return re.ReplaceAllStringFunc(op.URI, func(match string) string {
		name := strings.Trim(match, "{}")
		switch name {
		case "detectorId":
			return "detector-00000000000000000000000000000001"
		case "filterName":
			return "stackyard-filter"
		case "ipSetId":
			return "ipset-00000001"
		case "destinationId":
			return "destination-00000001"
		case "threatEntitySetId":
			return "threatentityset-00000001"
		case "threatIntelSetId":
			return "threatintelset-00000001"
		case "trustedEntitySetId":
			return "trustedentityset-00000001"
		case "malwareProtectionPlanId":
			return "mp-00000001"
		case "scanId":
			return "scan-00000001"
		case "resourceArn":
			return url.PathEscape("arn:aws:guardduty:us-east-1:123456789012:detector/detector-00000000000000000000000000000001")
		default:
			return "stackyard"
		}
	})
}

func TestGuardDutyStage0CatalogCoverage(t *testing.T) {
	if len(guardDutyOperations) != 87 {
		t.Fatalf("expected 87 GuardDuty operations from docs, got %d", len(guardDutyOperations))
	}
	if len(guardDutyOperationByName) != len(guardDutyOperations) {
		t.Fatalf("expected unique GuardDuty operation names")
	}

	requiredActions := []string{
		"CreateDetector",
		"ListDetectors",
		"GetDetector",
		"CreateMembers",
		"ListFindings",
		"CreateIPSet",
		"CreateThreatIntelSet",
		"StartMalwareScan",
		"UpdateOrganizationConfiguration",
	}
	for _, action := range requiredActions {
		if _, ok := guardDutyOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(guardDutyDataTypes) != 251 {
		t.Fatalf("expected 251 GuardDuty data types from docs, got %d", len(guardDutyDataTypes))
	}
	if len(guardDutyDataTypeByName) != len(guardDutyDataTypes) {
		t.Fatalf("expected unique GuardDuty data type names")
	}
}

func TestGuardDutyStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := guardDutyRequest(t, ts, http.MethodGet, "/not-a-real-guardduty-route", "")
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestGuardDutyStage0KnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := guardDutyRequest(t, ts, http.MethodGet, "/detector", "")
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "DetectorIds") {
		t.Fatalf("expected ListDetectors response body to include DetectorIds, got %q", body)
	}
}

func TestGuardDutyStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range guardDutyOperations {
		payload := `{}`
		if strings.EqualFold(op.Method, http.MethodGet) || strings.EqualFold(op.Method, http.MethodDelete) {
			payload = ""
		}
		resp := guardDutyRequest(t, ts, op.Method, guardDutyPathForOperation(op), payload)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
