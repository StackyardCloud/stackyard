package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestAmplifyUIBuilderStage123ResourceLifecycles(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := amplifyUIBuilderRequest(t, ts, http.MethodPost, "/app/d1234567890/environment/dev/components?clientToken=token-1", []byte(`{"name":"component-stage-001"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = amplifyUIBuilderRequest(t, ts, http.MethodGet, "/app/d1234567890/environment/dev/components/component-stage-001", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = amplifyUIBuilderRequest(t, ts, http.MethodPatch, "/app/d1234567890/environment/dev/components/component-stage-001?clientToken=token-2", []byte(`{"name":"component-stage-001-updated"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = amplifyUIBuilderRequest(t, ts, http.MethodDelete, "/app/d1234567890/environment/dev/components/component-stage-001", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = amplifyUIBuilderRequest(t, ts, http.MethodPost, "/app/d1234567890/environment/dev/forms?clientToken=token-3", []byte(`{"name":"form-stage-001"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = amplifyUIBuilderRequest(t, ts, http.MethodGet, "/app/d1234567890/environment/dev/forms/form-stage-001", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = amplifyUIBuilderRequest(t, ts, http.MethodPatch, "/app/d1234567890/environment/dev/forms/form-stage-001?clientToken=token-4", []byte(`{"name":"form-stage-001-updated"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = amplifyUIBuilderRequest(t, ts, http.MethodDelete, "/app/d1234567890/environment/dev/forms/form-stage-001", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = amplifyUIBuilderRequest(t, ts, http.MethodPost, "/app/d1234567890/environment/dev/themes?clientToken=token-5", []byte(`{"name":"theme-stage-001"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = amplifyUIBuilderRequest(t, ts, http.MethodGet, "/app/d1234567890/environment/dev/themes/theme-stage-001", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = amplifyUIBuilderRequest(t, ts, http.MethodPatch, "/app/d1234567890/environment/dev/themes/theme-stage-001?clientToken=token-6", []byte(`{"name":"theme-stage-001-updated"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = amplifyUIBuilderRequest(t, ts, http.MethodDelete, "/app/d1234567890/environment/dev/themes/theme-stage-001", nil)
	assertStatus(t, resp, http.StatusOK)
}

func TestAmplifyUIBuilderStage456MetadataCodegenTokenAndTagging(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := amplifyUIBuilderRequest(t, ts, http.MethodGet, "/app/d1234567890/environment/dev/metadata", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = amplifyUIBuilderRequest(t, ts, http.MethodPut, "/app/d1234567890/environment/dev/metadata/features/isRelationshipSupported", []byte(`{"newValue":false}`))
	assertStatus(t, resp, http.StatusOK)

	resp = amplifyUIBuilderRequest(t, ts, http.MethodPost, "/app/d1234567890/environment/dev/codegen-jobs?clientToken=token-7", []byte(`{"codegenJobToCreate":{}}`))
	assertStatus(t, resp, http.StatusOK)
	payload := decodeAmplifyUIBuilderPayload(t, resp)
	entity, _ := payload["entity"].(map[string]any)
	jobID, _ := entity["id"].(string)
	if jobID == "" {
		t.Fatalf("expected StartCodegenJob to return entity.id")
	}

	resp = amplifyUIBuilderRequest(t, ts, http.MethodGet, "/app/d1234567890/environment/dev/codegen-jobs/"+jobID, nil)
	assertStatus(t, resp, http.StatusOK)

	resp = amplifyUIBuilderRequest(t, ts, http.MethodPost, "/tokens/figma", []byte(`{"code":"abc"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = amplifyUIBuilderRequest(t, ts, http.MethodPost, "/tokens/figma/refresh", []byte(`{"refreshToken":"refresh"}`))
	assertStatus(t, resp, http.StatusOK)

	resourceARN := url.PathEscape("arn:aws:amplify:us-east-1:123456789012:apps/d1234567890")
	resp = amplifyUIBuilderRequest(t, ts, http.MethodPost, "/tags/"+resourceARN, []byte(`{"tags":{"owner":"qa","env":"test"}}`))
	assertStatus(t, resp, http.StatusOK)

	resp = amplifyUIBuilderRequest(t, ts, http.MethodGet, "/tags/"+resourceARN, nil)
	assertStatus(t, resp, http.StatusOK)
	payload = decodeAmplifyUIBuilderPayload(t, resp)
	tags, _ := payload["tags"].(map[string]any)
	if got, _ := tags["owner"].(string); got != "qa" {
		t.Fatalf("expected owner tag qa, got %q", got)
	}

	resp = amplifyUIBuilderRequest(t, ts, http.MethodDelete, "/tags/"+resourceARN+"?tagKeys=owner", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = amplifyUIBuilderRequest(t, ts, http.MethodGet, "/tags/"+resourceARN, nil)
	assertStatus(t, resp, http.StatusOK)
	payload = decodeAmplifyUIBuilderPayload(t, resp)
	tags, _ = payload["tags"].(map[string]any)
	if _, exists := tags["owner"]; exists {
		t.Fatalf("expected owner tag removed")
	}
}

func decodeAmplifyUIBuilderPayload(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	payload := map[string]any{}
	if err := json.Unmarshal(mustBody(t, resp), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	return payload
}
