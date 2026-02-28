package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestWellArchitectedStage12WorkloadLifecycleAndReadSurfaces(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := wellArchitectedRequest(t, ts, http.MethodPost, "/workloads", []byte(`{"WorkloadName":"stage-workload","Description":"stage","Environment":"PREPRODUCTION","ReviewOwner":"qa","Lenses":["wellarchitected"],"AwsRegions":["us-east-1"]}`))
	assertStatus(t, resp, http.StatusOK)
	createPayload := decodeWellArchitectedPayload(t, resp)
	workloadID := wellArchitectedPayloadString(createPayload, "WorkloadId")
	if workloadID == "" {
		t.Fatalf("expected CreateWorkload to include WorkloadId")
	}

	resp = wellArchitectedRequest(t, ts, http.MethodPost, "/workloadsSummaries", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "WorkloadSummaries") {
		t.Fatalf("expected ListWorkloads to include WorkloadSummaries, got %q", body)
	}

	resp = wellArchitectedRequest(t, ts, http.MethodGet, "/workloads/"+url.PathEscape(workloadID), nil)
	assertStatus(t, resp, http.StatusOK)
	resp = wellArchitectedRequest(t, ts, http.MethodPatch, "/workloads/"+url.PathEscape(workloadID), []byte(`{"Description":"updated-description"}`))
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "updated-description") {
		t.Fatalf("expected UpdateWorkload response to include updated-description, got %q", body)
	}

	resp = wellArchitectedRequest(t, ts, http.MethodGet, "/lenses", nil)
	assertStatus(t, resp, http.StatusOK)
	resp = wellArchitectedRequest(t, ts, http.MethodGet, "/lenses/wellarchitected", nil)
	assertStatus(t, resp, http.StatusOK)
	resp = wellArchitectedRequest(t, ts, http.MethodPost, "/notifications", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	resp = wellArchitectedRequest(t, ts, http.MethodDelete, "/workloads/"+url.PathEscape(workloadID), nil)
	assertStatus(t, resp, http.StatusOK)
}

func TestWellArchitectedStage34TemplateProfileAndAnswerSurfaces(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := wellArchitectedRequest(t, ts, http.MethodPost, "/profiles", []byte(`{"ProfileName":"stage-profile"}`))
	assertStatus(t, resp, http.StatusOK)
	profilePayload := decodeWellArchitectedPayload(t, resp)
	profileARN := wellArchitectedPayloadString(profilePayload, "ProfileArn")
	if profileARN == "" {
		t.Fatalf("expected CreateProfile to include ProfileArn")
	}

	resp = wellArchitectedRequest(t, ts, http.MethodGet, "/profileSummaries", nil)
	assertStatus(t, resp, http.StatusOK)
	resp = wellArchitectedRequest(t, ts, http.MethodGet, "/profiles/"+url.PathEscape(profileARN), nil)
	assertStatus(t, resp, http.StatusOK)
	resp = wellArchitectedRequest(t, ts, http.MethodPatch, "/profiles/"+url.PathEscape(profileARN), []byte(`{"ProfileName":"updated-profile"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = wellArchitectedRequest(t, ts, http.MethodPost, "/reviewTemplates", []byte(`{"TemplateName":"stage-template","Description":"stage"}`))
	assertStatus(t, resp, http.StatusOK)
	templatePayload := decodeWellArchitectedPayload(t, resp)
	templateARN := wellArchitectedPayloadString(templatePayload, "TemplateArn")
	if templateARN == "" {
		t.Fatalf("expected CreateReviewTemplate to include TemplateArn")
	}

	resp = wellArchitectedRequest(t, ts, http.MethodGet, "/reviewTemplates", nil)
	assertStatus(t, resp, http.StatusOK)
	resp = wellArchitectedRequest(t, ts, http.MethodGet, "/reviewTemplates/"+url.PathEscape(templateARN), nil)
	assertStatus(t, resp, http.StatusOK)

	resp = wellArchitectedRequest(t, ts, http.MethodGet, "/workloads/workload-000001/lensReviews/wellarchitected/answers", nil)
	assertStatus(t, resp, http.StatusOK)
	resp = wellArchitectedRequest(t, ts, http.MethodGet, "/workloads/workload-000001/lensReviews/wellarchitected/answers/security_1", nil)
	assertStatus(t, resp, http.StatusOK)
	resp = wellArchitectedRequest(t, ts, http.MethodPatch, "/workloads/workload-000001/lensReviews/wellarchitected/answers/security_1", []byte(`{"Notes":"stage-note"}`))
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "stage-note") {
		t.Fatalf("expected UpdateAnswer response to include stage-note, got %q", body)
	}

	resp = wellArchitectedRequest(t, ts, http.MethodDelete, "/reviewTemplates/"+url.PathEscape(templateARN), nil)
	assertStatus(t, resp, http.StatusOK)
	resp = wellArchitectedRequest(t, ts, http.MethodDelete, "/profiles/"+url.PathEscape(profileARN), nil)
	assertStatus(t, resp, http.StatusOK)
}

func TestWellArchitectedStage56TaggingValidationAndIdempotency(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	workloadARN := "arn:aws:wellarchitected:us-east-1:123456789012:workload/workload-000001"

	resp := wellArchitectedRequest(t, ts, http.MethodPost, "/tags/"+url.PathEscape(workloadARN), []byte(`{"Tags":{"env":"stage","owner":"qa"}}`))
	assertStatus(t, resp, http.StatusOK)
	resp = wellArchitectedRequest(t, ts, http.MethodGet, "/tags/"+url.PathEscape(workloadARN), nil)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "owner") {
		t.Fatalf("expected ListTagsForResource to include owner tag, got %q", body)
	}

	resp = wellArchitectedRequest(t, ts, http.MethodDelete, "/tags/WorkloadArn?tagKeys=owner", nil)
	assertStatus(t, resp, http.StatusOK)
	resp = wellArchitectedRequest(t, ts, http.MethodDelete, "/tags/WorkloadArn?tagKeys=owner", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = wellArchitectedRequest(t, ts, http.MethodGet, "/wellarchitected/unknown", nil)
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException for unknown route, got %q", body)
	}

	resp = signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/workloads",
		[]byte(`{"broken":`),
		map[string]string{"Content-Type": "application/json"},
		"wellarchitected",
	)
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "invalid JSON body") {
		t.Fatalf("expected invalid JSON body error, got %q", body)
	}
}

func decodeWellArchitectedPayload(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	payload := map[string]any{}
	if err := json.Unmarshal(mustBody(t, resp), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	return payload
}

func wellArchitectedPayloadString(payload map[string]any, key string) string {
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
