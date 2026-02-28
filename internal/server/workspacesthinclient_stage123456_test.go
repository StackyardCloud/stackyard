package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestWorkSpacesThinClientStage12EnvironmentAndDeviceLifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resourceID := "abcdefghijklmnopqrstuvwx"

	resp := workspacesThinClientRequest(t, ts, http.MethodPost, "/environments", []byte(`{"name":"stage-thin-client-environment"}`))
	assertStatus(t, resp, http.StatusOK)
	createEnv := decodeWorkSpacesThinClientPayload(t, resp)
	if got := workspacesThinClientPayloadString(createEnv, "id"); got == "" {
		t.Fatalf("expected CreateEnvironment to include id")
	}

	resp = workspacesThinClientRequest(t, ts, http.MethodGet, "/environments?maxResults=10", nil)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "environments") {
		t.Fatalf("expected ListEnvironments to include environments, got %q", body)
	}

	resp = workspacesThinClientRequest(t, ts, http.MethodGet, "/environments/"+url.PathEscape(resourceID), nil)
	assertStatus(t, resp, http.StatusOK)

	resp = workspacesThinClientRequest(t, ts, http.MethodPatch, "/environments/"+url.PathEscape(resourceID), []byte(`{"name":"updated-environment"}`))
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "updated-environment") {
		t.Fatalf("expected UpdateEnvironment response to include updated-environment, got %q", body)
	}

	resp = workspacesThinClientRequest(t, ts, http.MethodGet, "/devices?maxResults=10", nil)
	assertStatus(t, resp, http.StatusOK)
	resp = workspacesThinClientRequest(t, ts, http.MethodGet, "/devices/"+url.PathEscape(resourceID), nil)
	assertStatus(t, resp, http.StatusOK)
	resp = workspacesThinClientRequest(t, ts, http.MethodPatch, "/devices/"+url.PathEscape(resourceID), []byte(`{"name":"updated-device"}`))
	assertStatus(t, resp, http.StatusOK)
	resp = workspacesThinClientRequest(t, ts, http.MethodPost, "/deregister-device/"+url.PathEscape(resourceID), []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	resp = workspacesThinClientRequest(t, ts, http.MethodDelete, "/devices/"+url.PathEscape(resourceID)+"?clientToken=stage-token", nil)
	assertStatus(t, resp, http.StatusOK)
}

func TestWorkSpacesThinClientStage34SoftwareSetAndTaggingSurfaces(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resourceID := "abcdefghijklmnopqrstuvwx"
	resourceARN := "arn:aws:thinclient:us-east-1:123456789012:environment/" + resourceID

	resp := workspacesThinClientRequest(t, ts, http.MethodGet, "/softwaresets?maxResults=10", nil)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "softwareSets") {
		t.Fatalf("expected ListSoftwareSets to include softwareSets, got %q", body)
	}

	resp = workspacesThinClientRequest(t, ts, http.MethodGet, "/softwaresets/"+url.PathEscape(resourceID), nil)
	assertStatus(t, resp, http.StatusOK)

	resp = workspacesThinClientRequest(t, ts, http.MethodPatch, "/softwaresets/"+url.PathEscape(resourceID), []byte(`{"name":"updated-software-set"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = workspacesThinClientRequest(t, ts, http.MethodPost, "/tags/"+url.PathEscape(resourceARN), []byte(`{"tags":{"env":"stage","owner":"qa"}}`))
	assertStatus(t, resp, http.StatusOK)
	resp = workspacesThinClientRequest(t, ts, http.MethodGet, "/tags/"+url.PathEscape(resourceARN), nil)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "owner") {
		t.Fatalf("expected ListTagsForResource to include owner tag, got %q", body)
	}
	resp = workspacesThinClientRequest(t, ts, http.MethodDelete, "/tags/"+url.PathEscape(resourceARN)+"?tagKeys=owner", nil)
	assertStatus(t, resp, http.StatusOK)
	resp = workspacesThinClientRequest(t, ts, http.MethodDelete, "/environments/"+url.PathEscape(resourceID)+"?clientToken=stage-token", nil)
	assertStatus(t, resp, http.StatusOK)
}

func TestWorkSpacesThinClientStage56ValidationAndIdempotency(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resourceID := "abcdefghijklmnopqrstuvwx"

	resp := workspacesThinClientRequest(t, ts, http.MethodDelete, "/devices/"+url.PathEscape(resourceID)+"?clientToken=idempotent-token", nil)
	assertStatus(t, resp, http.StatusOK)
	resp = workspacesThinClientRequest(t, ts, http.MethodDelete, "/devices/"+url.PathEscape(resourceID)+"?clientToken=idempotent-token", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = workspacesThinClientRequest(t, ts, http.MethodPost, "/thinclient/unknown", []byte(`{}`))
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException for unknown route, got %q", body)
	}

	resp = signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/environments",
		[]byte(`{"broken":`),
		map[string]string{"Content-Type": "application/json"},
		"thinclient",
	)
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "invalid JSON body") {
		t.Fatalf("expected invalid JSON body error, got %q", body)
	}
}

func decodeWorkSpacesThinClientPayload(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	payload := map[string]any{}
	if err := json.Unmarshal(mustBody(t, resp), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	return payload
}

func workspacesThinClientPayloadString(payload map[string]any, key string) string {
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
