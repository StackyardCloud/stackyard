package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func partnerCentralSellingRequest(t *testing.T, ts *httptest.Server, action string, payload string) *http.Response {
	t.Helper()
	return signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		[]byte(payload),
		map[string]string{
			"Content-Type": "application/x-amz-json-1.0",
			"X-Amz-Target": "PartnerCentralSelling." + action,
		},
		"partnercentral-selling",
	)
}

func TestPartnerCentralSellingStage0CatalogCoverage(t *testing.T) {
	if len(partnerCentralSellingOperations) != 97 {
		t.Fatalf("expected 97 Partner Central actions from docs, got %d", len(partnerCentralSellingOperations))
	}
	if len(partnerCentralSellingOperationByName) != len(partnerCentralSellingOperations) {
		t.Fatalf("expected unique Partner Central action names")
	}

	requiredActions := []string{
		"CreateOpportunity",
		"GetOpportunity",
		"ListOpportunities",
		"UpdateOpportunity",
		"CreateEngagement",
		"GetEngagement",
		"CreateEngagementInvitation",
		"AcceptEngagementInvitation",
		"CreateResourceSnapshot",
		"ListResourceSnapshots",
		"GetSellingSystemSettings",
		"PutSellingSystemSettings",
	}
	for _, action := range requiredActions {
		if _, ok := partnerCentralSellingOperationByName[action]; !ok {
			t.Fatalf("missing documented action %s", action)
		}
	}

	if len(partnerCentralSellingDataTypes) != 153 {
		t.Fatalf("expected 153 Partner Central data types from docs, got %d", len(partnerCentralSellingDataTypes))
	}
	if len(partnerCentralSellingDataTypeByName) != len(partnerCentralSellingDataTypes) {
		t.Fatalf("expected unique Partner Central data type names")
	}

	requiredTypes := []string{
		"OpportunitySummary",
		"EngagementSummary",
		"EngagementInvitationSummary",
		"ResourceSnapshotSummary",
		"Tag",
		"ValidationError",
	}
	for _, typeName := range requiredTypes {
		if _, ok := partnerCentralSellingDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestPartnerCentralSellingStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := partnerCentralSellingRequest(t, ts, "TotallyUnknownAction", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestPartnerCentralSellingKnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := partnerCentralSellingRequest(t, ts, "ListOpportunities", `{"Catalog":"AWS"}`)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "OpportunitySummaries") {
		t.Fatalf("expected ListOpportunities response body to include OpportunitySummaries, got %q", body)
	}
}

func TestPartnerCentralSellingStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range partnerCentralSellingOperations {
		resp := partnerCentralSellingRequest(t, ts, op.Name, `{}`)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
