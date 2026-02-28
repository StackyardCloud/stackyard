package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func artifactRequest(t *testing.T, ts *httptest.Server, method, path string, body []byte) *http.Response {
	t.Helper()
	headers := map[string]string{}
	if len(body) > 0 {
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "artifact")
}

func TestArtifactStage0CatalogCoverage(t *testing.T) {
	if len(artifactOperations) != 15 {
		t.Fatalf("expected 15 Artifact actions from docs, got %d", len(artifactOperations))
	}
	if len(artifactOperationByName) != len(artifactOperations) {
		t.Fatalf("expected unique Artifact action names")
	}

	requiredActions := []string{
		"ListAgreements",
		"AcceptAgreement",
		"GetAgreement",
		"AcceptNdaForAgreement",
		"GetNdaForAgreement",
		"ListCustomerAgreements",
		"GetCustomerAgreement",
		"ListReports",
		"ListReportVersions",
		"GetReport",
		"GetReportMetadata",
		"GetTermForReport",
		"GetAccountSettings",
		"PutAccountSettings",
		"TerminateAgreement",
	}
	for _, action := range requiredActions {
		if _, ok := artifactOperationByName[action]; !ok {
			t.Fatalf("missing documented action %s", action)
		}
	}

	if len(artifactDataTypes) != 7 {
		t.Fatalf("expected 7 Artifact data types from docs, got %d", len(artifactDataTypes))
	}
	if len(artifactDataTypeByName) != len(artifactDataTypes) {
		t.Fatalf("expected unique Artifact data type names")
	}

	requiredTypes := []string{
		"AccountSettings",
		"AgreementSummary",
		"CustomerAgreementSummary",
		"ReportDetail",
		"ReportSummary",
		"TerminateCustomerAgreementSummary",
		"ValidationExceptionField",
	}
	for _, typeName := range requiredTypes {
		if _, ok := artifactDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestArtifactStage0UnknownRouteReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := artifactRequest(t, ts, http.MethodGet, "/v1/artifact/unknown", nil)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestArtifactStage0KnownActionReturnsListReports(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := artifactRequest(t, ts, http.MethodGet, "/v1/report/list?maxResults=10", nil)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "reports") {
		t.Fatalf("expected ListReports response body to include reports, got %q", body)
	}
}

func TestArtifactStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range artifactOperations {
		path := op.URI
		var body []byte
		switch op.Name {
		case "GetAgreement", "GetNdaForAgreement":
			path += "?agreementId=agr-000001"
		case "GetCustomerAgreement":
			path += "?customerAgreementId=cagr-000001"
		case "GetReport", "GetReportMetadata", "GetTermForReport":
			path += "?reportId=rpt-000001"
		case "ListAgreements", "ListCustomerAgreements", "ListReports":
			path += "?maxResults=10"
		case "ListReportVersions":
			path += "?reportId=rpt-000001&maxResults=10"
		case "AcceptAgreement", "AcceptNdaForAgreement":
			body = []byte(`{"agreementId":"agr-000001"}`)
		case "TerminateAgreement":
			body = []byte(`{"customerAgreementId":"cagr-000001"}`)
		case "PutAccountSettings":
			body = []byte(`{"notificationsEnabled":true}`)
		}

		resp := artifactRequest(t, ts, op.Method, path, body)
		respBody := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(respBody, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
	}
}
