package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestArtifactStage12AgreementAndNdaLifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := artifactRequest(t, ts, http.MethodGet, "/v1/agreement/list?maxResults=10", nil)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "agr-000001") {
		t.Fatalf("expected ListAgreements response to include agr-000001, got %q", body)
	}

	resp = artifactRequest(t, ts, http.MethodPost, "/v1/agreement/accept", []byte(`{"agreementId":"agr-000001"}`))
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "ACCEPTED") {
		t.Fatalf("expected AcceptAgreement response to include ACCEPTED, got %q", body)
	}

	resp = artifactRequest(t, ts, http.MethodPost, "/v1/agreement/acceptNdaForAgreement", []byte(`{"agreementId":"agr-000001"}`))
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "ndaAccepted") {
		t.Fatalf("expected AcceptNdaForAgreement response to include ndaAccepted, got %q", body)
	}

	resp = artifactRequest(t, ts, http.MethodGet, "/v1/agreement/get?agreementId=agr-000001", nil)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "accepted") {
		t.Fatalf("expected GetAgreement response to include accepted fields, got %q", body)
	}

	resp = artifactRequest(t, ts, http.MethodGet, "/v1/agreement/getNdaForAgreement?agreementId=agr-000001", nil)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "NDA terms") {
		t.Fatalf("expected GetNdaForAgreement response to include term content, got %q", body)
	}
}

func TestArtifactStage34ReportSurface(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := artifactRequest(t, ts, http.MethodGet, "/v1/report/list?maxResults=10", nil)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "rpt-000001") {
		t.Fatalf("expected ListReports response to include seeded report id, got %q", body)
	}

	resp = artifactRequest(t, ts, http.MethodGet, "/v1/report/listVersions?reportId=rpt-000001&maxResults=10", nil)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "reportVersions") {
		t.Fatalf("expected ListReportVersions response to include reportVersions, got %q", body)
	}

	resp = artifactRequest(t, ts, http.MethodGet, "/v1/report/getMetadata?reportId=rpt-000001", nil)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "latestVersion") {
		t.Fatalf("expected GetReportMetadata response to include latestVersion, got %q", body)
	}

	resp = artifactRequest(t, ts, http.MethodGet, "/v1/report/getTermForReport?reportId=rpt-000001", nil)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "termText") {
		t.Fatalf("expected GetTermForReport response to include termText, got %q", body)
	}

	resp = artifactRequest(t, ts, http.MethodGet, "/v1/report/get?reportId=rpt-000001", nil)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "downloadUrl") {
		t.Fatalf("expected GetReport response to include downloadUrl, got %q", body)
	}
}

func TestArtifactStage56CustomerAgreementAccountSettingsValidationAndIdempotency(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := artifactRequest(t, ts, http.MethodGet, "/v1/customer-agreement/list?maxResults=10", nil)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "cagr-000001") {
		t.Fatalf("expected ListCustomerAgreements response to include cagr-000001, got %q", body)
	}

	resp = artifactRequest(t, ts, http.MethodGet, "/v1/customer-agreement/get?customerAgreementId=cagr-000001", nil)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "ACTIVE") {
		t.Fatalf("expected GetCustomerAgreement response to include ACTIVE, got %q", body)
	}

	resp = artifactRequest(t, ts, http.MethodGet, "/v1/account-settings/get", nil)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "accountSettings") {
		t.Fatalf("expected GetAccountSettings response to include accountSettings, got %q", body)
	}

	first := artifactRequest(t, ts, http.MethodPut, "/v1/account-settings/put", []byte(`{"notificationsEnabled":false,"defaultReportFormat":"JSON"}`))
	assertStatus(t, first, http.StatusOK)
	firstBody := string(mustBody(t, first))

	second := artifactRequest(t, ts, http.MethodPut, "/v1/account-settings/put", []byte(`{"notificationsEnabled":false,"defaultReportFormat":"JSON"}`))
	assertStatus(t, second, http.StatusOK)
	secondBody := string(mustBody(t, second))
	if !strings.Contains(firstBody, "notificationsEnabled") || !strings.Contains(secondBody, "notificationsEnabled") {
		t.Fatalf("expected PutAccountSettings responses to include notificationsEnabled, got %q and %q", firstBody, secondBody)
	}

	resp = artifactRequest(t, ts, http.MethodPost, "/v1/customer-agreement/terminate", []byte(`{"customerAgreementId":"cagr-000001"}`))
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "TERMINATED") {
		t.Fatalf("expected TerminateAgreement response to include TERMINATED, got %q", body)
	}

	resp = artifactRequest(t, ts, http.MethodPost, "/v1/customer-agreement/terminate", []byte(`{"customerAgreementId":"cagr-000001"}`))
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "TERMINATED") {
		t.Fatalf("expected idempotent TerminateAgreement response to include TERMINATED, got %q", body)
	}

	resp = artifactRequest(t, ts, http.MethodGet, "/v1/artifact/does-not-exist", nil)
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException for unknown action, got %q", body)
	}

	resp = signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/v1/agreement/accept",
		[]byte(`{"broken":`),
		map[string]string{"Content-Type": "application/json"},
		"artifact",
	)
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "invalid JSON body") {
		t.Fatalf("expected invalid JSON body error, got %q", body)
	}
}
