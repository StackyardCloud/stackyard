package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestFISStage12TemplateAndTargetAccountLifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := fisRequest(t, ts, http.MethodPost, "/experimentTemplates", []byte(`{"description":"stage-fis-template","clientToken":"stage-fis-create-template-token-000001"}`))
	assertStatus(t, resp, http.StatusOK)
	createPayload := decodeFISPayload(t, resp)
	template := fisMap(createPayload, "experimentTemplate")
	templateID := fisStringField(template, "id")
	if templateID == "" {
		t.Fatalf("expected CreateExperimentTemplate response to include id")
	}

	resp = fisRequest(t, ts, http.MethodGet, "/experimentTemplates/"+url.PathEscape(templateID), nil)
	assertStatus(t, resp, http.StatusOK)

	resp = fisRequest(t, ts, http.MethodPatch, "/experimentTemplates/"+url.PathEscape(templateID), []byte(`{"description":"stage-fis-template-updated"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = fisRequest(t, ts, http.MethodGet, "/experimentTemplates?maxResults=10", nil)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "experimentTemplates") {
		t.Fatalf("expected ListExperimentTemplates to include experimentTemplates, got %q", body)
	}

	resp = fisRequest(t, ts, http.MethodPost, "/experimentTemplates/"+url.PathEscape(templateID)+"/targetAccountConfigurations", []byte(`{"accountId":"111122223333","roleArn":"arn:aws:iam::111122223333:role/stackyard-fis-role"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = fisRequest(t, ts, http.MethodGet, "/experimentTemplates/"+url.PathEscape(templateID)+"/targetAccountConfigurations/111122223333", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = fisRequest(t, ts, http.MethodPatch, "/experimentTemplates/"+url.PathEscape(templateID)+"/targetAccountConfigurations/111122223333", []byte(`{"status":"DISABLED"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = fisRequest(t, ts, http.MethodGet, "/experimentTemplates/"+url.PathEscape(templateID)+"/targetAccountConfigurations?maxResults=10", nil)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "targetAccountConfigurations") {
		t.Fatalf("expected ListTargetAccountConfigurations to include targetAccountConfigurations, got %q", body)
	}

	resp = fisRequest(t, ts, http.MethodDelete, "/experimentTemplates/"+url.PathEscape(templateID)+"/targetAccountConfigurations/111122223333", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = fisRequest(t, ts, http.MethodDelete, "/experimentTemplates/"+url.PathEscape(templateID), nil)
	assertStatus(t, resp, http.StatusOK)
}

func TestFISStage34ExperimentLifecycleAndReadSurfaces(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := fisRequest(t, ts, http.MethodPost, "/experiments", []byte(`{"experimentTemplateId":"ext-000001","clientToken":"stage-fis-start-experiment-token-000001"}`))
	assertStatus(t, resp, http.StatusOK)
	startPayload := decodeFISPayload(t, resp)
	experiment := fisMap(startPayload, "experiment")
	experimentID := fisStringField(experiment, "id")
	if experimentID == "" {
		t.Fatalf("expected StartExperiment response to include id")
	}

	resp = fisRequest(t, ts, http.MethodGet, "/experiments/"+url.PathEscape(experimentID), nil)
	assertStatus(t, resp, http.StatusOK)

	resp = fisRequest(t, ts, http.MethodGet, "/experiments?experimentTemplateId=ext-000001&maxResults=10", nil)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "experiments") {
		t.Fatalf("expected ListExperiments to include experiments, got %q", body)
	}

	resp = fisRequest(t, ts, http.MethodGet, "/experiments/"+url.PathEscape(experimentID)+"/resolvedTargets?maxResults=10", nil)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "resolvedTargets") {
		t.Fatalf("expected ListExperimentResolvedTargets to include resolvedTargets, got %q", body)
	}

	resp = fisRequest(t, ts, http.MethodGet, "/experiments/"+url.PathEscape(experimentID)+"/targetAccountConfigurations?maxResults=10", nil)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "experimentTargetAccountConfigurations") {
		t.Fatalf("expected ListExperimentTargetAccountConfigurations to include experimentTargetAccountConfigurations, got %q", body)
	}

	resp = fisRequest(t, ts, http.MethodGet, "/experiments/"+url.PathEscape(experimentID)+"/targetAccountConfigurations/123456789012", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = fisRequest(t, ts, http.MethodDelete, "/experiments/"+url.PathEscape(experimentID), nil)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "stopped") {
		t.Fatalf("expected StopExperiment response to include stopped state, got %q", body)
	}
}

func TestFISStage56ReadTaggingValidationAndIdempotency(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := fisRequest(t, ts, http.MethodGet, "/actions/aws%3Aec2%3Astop-instances", nil)
	assertStatus(t, resp, http.StatusOK)
	resp = fisRequest(t, ts, http.MethodGet, "/actions?maxResults=10", nil)
	assertStatus(t, resp, http.StatusOK)
	resp = fisRequest(t, ts, http.MethodGet, "/targetResourceTypes/aws%3Aec2%3Ainstance", nil)
	assertStatus(t, resp, http.StatusOK)
	resp = fisRequest(t, ts, http.MethodGet, "/targetResourceTypes?maxResults=10", nil)
	assertStatus(t, resp, http.StatusOK)
	resp = fisRequest(t, ts, http.MethodGet, "/safetyLevers/default", nil)
	assertStatus(t, resp, http.StatusOK)
	resp = fisRequest(t, ts, http.MethodPatch, "/safetyLevers/default/state", []byte(`{"state":"DISABLED","reason":"stage-test"}`))
	assertStatus(t, resp, http.StatusOK)

	resourceARN := "arn:aws:fis:us-east-1:123456789012:experiment-template/ext-000001"
	escapedResourceARN := url.PathEscape(resourceARN)
	resp = fisRequest(t, ts, http.MethodPost, "/tags/"+escapedResourceARN, []byte(`{"tags":{"env":"stage","owner":"qa"}}`))
	assertStatus(t, resp, http.StatusOK)
	resp = fisRequest(t, ts, http.MethodGet, "/tags/"+escapedResourceARN, nil)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "owner") {
		t.Fatalf("expected ListTagsForResource to include owner tag, got %q", body)
	}
	resp = fisRequest(t, ts, http.MethodDelete, "/tags/"+escapedResourceARN+"?tagKeys=owner", nil)
	assertStatus(t, resp, http.StatusOK)

	createToken := "stage-fis-idempotent-create-template-token-000001"
	resp = fisRequest(t, ts, http.MethodPost, "/experimentTemplates", []byte(`{"clientToken":"`+createToken+`","description":"idempotent-1"}`))
	assertStatus(t, resp, http.StatusOK)
	firstCreate := decodeFISPayload(t, resp)
	firstTemplateID := fisStringField(fisMap(firstCreate, "experimentTemplate"), "id")
	resp = fisRequest(t, ts, http.MethodPost, "/experimentTemplates", []byte(`{"clientToken":"`+createToken+`","description":"idempotent-2"}`))
	assertStatus(t, resp, http.StatusOK)
	secondCreate := decodeFISPayload(t, resp)
	secondTemplateID := fisStringField(fisMap(secondCreate, "experimentTemplate"), "id")
	if firstTemplateID == "" || secondTemplateID == "" || firstTemplateID != secondTemplateID {
		t.Fatalf("expected idempotent CreateExperimentTemplate to return same id: %q != %q", firstTemplateID, secondTemplateID)
	}

	startToken := "stage-fis-idempotent-start-experiment-token-000001"
	resp = fisRequest(t, ts, http.MethodPost, "/experiments", []byte(`{"experimentTemplateId":"ext-000001","clientToken":"`+startToken+`"}`))
	assertStatus(t, resp, http.StatusOK)
	firstStart := decodeFISPayload(t, resp)
	firstExperimentID := fisStringField(fisMap(firstStart, "experiment"), "id")
	resp = fisRequest(t, ts, http.MethodPost, "/experiments", []byte(`{"experimentTemplateId":"ext-000001","clientToken":"`+startToken+`"}`))
	assertStatus(t, resp, http.StatusOK)
	secondStart := decodeFISPayload(t, resp)
	secondExperimentID := fisStringField(fisMap(secondStart, "experiment"), "id")
	if firstExperimentID == "" || secondExperimentID == "" || firstExperimentID != secondExperimentID {
		t.Fatalf("expected idempotent StartExperiment to return same id: %q != %q", firstExperimentID, secondExperimentID)
	}

	resp = fisRequest(t, ts, http.MethodPost, "/fis/unknown", []byte(`{}`))
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException for unknown route, got %q", body)
	}

	resp = signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/experimentTemplates",
		[]byte(`{"broken":`),
		map[string]string{"Content-Type": "application/json"},
		"fis",
	)
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "invalid JSON body") {
		t.Fatalf("expected invalid JSON body error, got %q", body)
	}
}

func decodeFISPayload(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	payload := map[string]any{}
	if err := json.Unmarshal(mustBody(t, resp), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	return payload
}

func fisMap(payload map[string]any, key string) map[string]any {
	if payload == nil {
		return map[string]any{}
	}
	if raw, ok := payload[key]; ok {
		if value, ok := raw.(map[string]any); ok {
			return value
		}
	}
	return map[string]any{}
}

func fisStringField(payload map[string]any, key string) string {
	if payload == nil {
		return ""
	}
	raw, ok := payload[key]
	if !ok {
		return ""
	}
	value, ok := raw.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(value)
}
