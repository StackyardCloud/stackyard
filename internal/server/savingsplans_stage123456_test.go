package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSavingsPlansStage12OfferingsAndReadSurfaces(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := savingsPlansRequest(t, ts, http.MethodPost, "/DescribeSavingsPlansOfferings", `{"maxResults":10}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "searchResults") {
		t.Fatalf("expected DescribeSavingsPlansOfferings response to include searchResults, got %q", body)
	}

	resp = savingsPlansRequest(t, ts, http.MethodPost, "/DescribeSavingsPlansOfferingRates", `{"maxResults":10}`)
	assertStatus(t, resp, http.StatusOK)

	resp = savingsPlansRequest(t, ts, http.MethodPost, "/DescribeSavingsPlans", `{"maxResults":10}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "savingsPlans") {
		t.Fatalf("expected DescribeSavingsPlans response to include savingsPlans, got %q", body)
	}

	resp = savingsPlansRequest(t, ts, http.MethodPost, "/DescribeSavingsPlanRates", `{"savingsPlanId":"sp-000001"}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "searchResults") {
		t.Fatalf("expected DescribeSavingsPlanRates response to include searchResults, got %q", body)
	}
}

func TestSavingsPlansStage34LifecycleMutations(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := savingsPlansRequest(
		t,
		ts,
		http.MethodPost,
		"/CreateSavingsPlan",
		`{"clientToken":"stage-create-token","savingsPlanOfferingId":"offering-000001","commitment":"0.010000","upfrontPaymentAmount":"0.0"}`,
	)
	assertStatus(t, resp, http.StatusOK)
	createPayload := decodeSavingsPlansPayload(t, resp)
	savingsPlanID := savingsPlansPayloadString(createPayload, "savingsPlanId")
	savingsPlanARN := savingsPlansPayloadString(createPayload, "savingsPlanArn")
	if strings.TrimSpace(savingsPlanID) == "" {
		t.Fatalf("expected CreateSavingsPlan to return savingsPlanId")
	}
	if strings.TrimSpace(savingsPlanARN) == "" {
		t.Fatalf("expected CreateSavingsPlan to return savingsPlanArn")
	}

	resp = savingsPlansRequest(
		t,
		ts,
		http.MethodPost,
		"/DescribeSavingsPlans",
		`{"savingsPlanIds":["`+savingsPlanID+`"]}`,
	)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, savingsPlanID) {
		t.Fatalf("expected DescribeSavingsPlans response to include %s, got %q", savingsPlanID, body)
	}

	resp = savingsPlansRequest(
		t,
		ts,
		http.MethodPost,
		"/DeleteQueuedSavingsPlan",
		`{"savingsPlanId":"`+savingsPlanID+`"}`,
	)
	assertStatus(t, resp, http.StatusOK)

	resp = savingsPlansRequest(
		t,
		ts,
		http.MethodPost,
		"/ReturnSavingsPlan",
		`{"savingsPlanId":"`+savingsPlanID+`"}`,
	)
	assertStatus(t, resp, http.StatusOK)
}

func TestSavingsPlansStage5Tagging(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resourceARN := "arn:aws:savingsplans:us-east-1:123456789012:savingsplan/sp-000001"

	resp := savingsPlansRequest(t, ts, http.MethodPost, "/TagResource", `{"resourceArn":"`+resourceARN+`","tags":{"env":"stage","owner":"qa"}}`)
	assertStatus(t, resp, http.StatusOK)

	resp = savingsPlansRequest(t, ts, http.MethodPost, "/ListTagsForResource", `{"resourceArn":"`+resourceARN+`"}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "owner") {
		t.Fatalf("expected ListTagsForResource response to include owner tag, got %q", body)
	}

	resp = savingsPlansRequest(t, ts, http.MethodPost, "/UntagResource", `{"resourceArn":"`+resourceARN+`","tagKeys":["owner"]}`)
	assertStatus(t, resp, http.StatusOK)

	resp = savingsPlansRequest(t, ts, http.MethodPost, "/ListTagsForResource", `{"resourceArn":"`+resourceARN+`"}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); strings.Contains(body, "owner") {
		t.Fatalf("expected owner tag to be removed, got %q", body)
	}
}

func TestSavingsPlansStage6ValidationAndIdempotency(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := savingsPlansRequest(t, ts, http.MethodPost, "/unknown-savingsplans-route", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException for unknown route, got %q", body)
	}

	resp = signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/CreateSavingsPlan",
		[]byte(`{"broken":`),
		map[string]string{"Content-Type": "application/json"},
		"savingsplans",
	)
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "invalid JSON body") {
		t.Fatalf("expected invalid JSON body error, got %q", body)
	}

	payload := `{"clientToken":"idempotent-create-token","savingsPlanOfferingId":"offering-000001"}`
	resp = savingsPlansRequest(t, ts, http.MethodPost, "/CreateSavingsPlan", payload)
	assertStatus(t, resp, http.StatusOK)
	first := decodeSavingsPlansPayload(t, resp)
	firstID := savingsPlansPayloadString(first, "savingsPlanId")
	if strings.TrimSpace(firstID) == "" {
		t.Fatalf("expected first CreateSavingsPlan response to include savingsPlanId")
	}

	resp = savingsPlansRequest(t, ts, http.MethodPost, "/CreateSavingsPlan", payload)
	assertStatus(t, resp, http.StatusOK)
	second := decodeSavingsPlansPayload(t, resp)
	secondID := savingsPlansPayloadString(second, "savingsPlanId")
	if firstID != secondID {
		t.Fatalf("expected idempotent CreateSavingsPlan IDs to match, got %q and %q", firstID, secondID)
	}
}

func decodeSavingsPlansPayload(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	payload := map[string]any{}
	if err := json.Unmarshal(mustBody(t, resp), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	return payload
}

func savingsPlansPayloadString(payload map[string]any, key string) string {
	raw, ok := payload[key]
	if !ok || raw == nil {
		return ""
	}
	value, ok := raw.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(value)
}
