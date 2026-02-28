package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestManagedServicesCMStage12ChangeTypesAndRFCLifecycleReads(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := managedServicesCMRequest(t, ts, "ListChangeTypeCategories", `{}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "Infrastructure") {
		t.Fatalf("expected ListChangeTypeCategories to include Infrastructure, got %q", body)
	}

	resp = managedServicesCMRequest(t, ts, "ListChangeTypeVersionSummaries", `{}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "ct-ec2-patch") {
		t.Fatalf("expected ListChangeTypeVersionSummaries to include ct-ec2-patch, got %q", body)
	}

	resp = managedServicesCMRequest(t, ts, "GetChangeTypeVersion", `{"ChangeTypeId":"ct-ec2-patch","Version":"1.0"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = managedServicesCMRequest(t, ts, "CreateRfc", `{"Title":"stage-rfc","ChangeTypeId":"ct-ec2-patch","Version":"1.0","ClientToken":"stage-managedservicescm-create-rfc-token-000001"}`)
	assertStatus(t, resp, http.StatusOK)
	createBody := string(mustBody(t, resp))
	if !strings.Contains(createBody, "RfcId") {
		t.Fatalf("expected CreateRfc to include RfcId, got %q", createBody)
	}

	resp = managedServicesCMRequest(t, ts, "ListRfcSummaries", `{}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "stage-rfc") {
		t.Fatalf("expected ListRfcSummaries to include stage-rfc, got %q", body)
	}
}

func TestManagedServicesCMStage345RFCMutationsAttachmentsCorrespondenceAndRestrictedTimes(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := managedServicesCMRequest(t, ts, "CreateRfc", `{"Title":"stage-lifecycle-rfc","ClientToken":"stage-managedservicescm-lifecycle-token-000001"}`)
	assertStatus(t, resp, http.StatusOK)
	createBody := string(mustBody(t, resp))
	rfcID := extractJSONField(createBody, "RfcId")
	if rfcID == "" {
		t.Fatalf("expected CreateRfc to include RfcId, got %q", createBody)
	}

	resp = managedServicesCMRequest(t, ts, "UpdateRfc", `{"RfcId":"`+rfcID+`","Title":"stage-lifecycle-rfc-updated","Impact":"MEDIUM"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = managedServicesCMRequest(t, ts, "SubmitRfc", `{"RfcId":"`+rfcID+`"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = managedServicesCMRequest(t, ts, "ApproveRfc", `{"RfcId":"`+rfcID+`"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = managedServicesCMRequest(t, ts, "GetRfc", `{"RfcId":"`+rfcID+`"}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "Approved") {
		t.Fatalf("expected GetRfc to reflect Approved status, got %q", body)
	}

	resp = managedServicesCMRequest(t, ts, "CreateRfcAttachment", `{"RfcId":"`+rfcID+`","FileName":"change-plan.txt","Description":"stage attachment"}`)
	assertStatus(t, resp, http.StatusOK)
	attachmentID := extractJSONField(string(mustBody(t, resp)), "AttachmentId")
	if attachmentID == "" {
		t.Fatalf("expected CreateRfcAttachment to include AttachmentId")
	}
	resp = managedServicesCMRequest(t, ts, "GetRfcAttachment", `{"RfcId":"`+rfcID+`","AttachmentId":"`+attachmentID+`"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = managedServicesCMRequest(t, ts, "ListRfcAttachmentSummaries", `{"RfcId":"`+rfcID+`"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = managedServicesCMRequest(t, ts, "CreateRfcCorrespondence", `{"RfcId":"`+rfcID+`","Message":"please schedule this change"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = managedServicesCMRequest(t, ts, "ListRfcCorrespondences", `{"RfcId":"`+rfcID+`"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = managedServicesCMRequest(t, ts, "ListRestrictedExecutionTimes", `{}`)
	assertStatus(t, resp, http.StatusOK)
	resp = managedServicesCMRequest(t, ts, "UpdateRestrictedExecutionTimes", `{"RestrictedExecutionTimes":[{"StartTime":"2026-01-01T00:00:00Z","EndTime":"2026-01-01T01:00:00Z","Reason":"stage window"}]}`)
	assertStatus(t, resp, http.StatusOK)
	resp = managedServicesCMRequest(t, ts, "ListRestrictedExecutionTimes", `{}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "stage window") {
		t.Fatalf("expected updated restricted execution reason, got %q", body)
	}

	resp = managedServicesCMRequest(t, ts, "RejectRfc", `{"RfcId":"`+rfcID+`"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = managedServicesCMRequest(t, ts, "CancelRfc", `{"RfcId":"`+rfcID+`"}`)
	assertStatus(t, resp, http.StatusOK)
}

func TestManagedServicesCMStage6ValidationIdempotencyAndMalformedJSON(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	token := "stage-managedservicescm-idempotency-token-000001"

	resp := managedServicesCMRequest(t, ts, "CreateRfc", `{"Title":"idempotent-rfc-1","ClientToken":"`+token+`"}`)
	assertStatus(t, resp, http.StatusOK)
	firstID := extractJSONField(string(mustBody(t, resp)), "RfcId")
	if firstID == "" {
		t.Fatalf("expected first CreateRfc response to include RfcId")
	}

	resp = managedServicesCMRequest(t, ts, "CreateRfc", `{"Title":"idempotent-rfc-2","ClientToken":"`+token+`"}`)
	assertStatus(t, resp, http.StatusOK)
	secondID := extractJSONField(string(mustBody(t, resp)), "RfcId")
	if secondID == "" || firstID != secondID {
		t.Fatalf("expected idempotent CreateRfc to return same RfcId, got %q and %q", firstID, secondID)
	}

	resp = managedServicesCMRequest(t, ts, "TotallyUnknownAction", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException for unknown action, got %q", body)
	}

	resp = signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		[]byte(`{"broken":`),
		map[string]string{
			"Content-Type": "application/x-amz-json-1.1",
			"X-Amz-Target": "AWSManagedServicesChangeManagement.ListRfcSummaries",
		},
		"amscm",
	)
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "invalid JSON body") {
		t.Fatalf("expected invalid JSON body error, got %q", body)
	}
}

func extractJSONField(body, key string) string {
	if strings.TrimSpace(body) == "" || strings.TrimSpace(key) == "" {
		return ""
	}
	parsed := map[string]any{}
	if err := json.Unmarshal([]byte(body), &parsed); err != nil {
		return ""
	}
	value, _ := parsed[key].(string)
	return strings.TrimSpace(value)
}
