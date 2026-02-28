package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestWorkSpacesWebStage12PortalAndSettingsLifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := workspacesWebRequest(t, ts, http.MethodPost, "/portals", []byte(`{"displayName":"stage-portal"}`))
	assertStatus(t, resp, http.StatusOK)
	createPortal := decodeWorkSpacesWebPayload(t, resp)
	portalARN := workspacesWebPayloadString(createPortal, "portalArn")
	if portalARN == "" {
		t.Fatalf("expected CreatePortal to return portalArn")
	}

	resp = workspacesWebRequest(t, ts, http.MethodGet, "/portals/"+url.PathEscape(portalARN), nil)
	assertStatus(t, resp, http.StatusOK)

	resp = workspacesWebRequest(t, ts, http.MethodPut, "/portals/"+url.PathEscape(portalARN), []byte(`{"portalDisplayName":"stage-portal-updated"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = workspacesWebRequest(t, ts, http.MethodGet, "/portals?maxResults=10", nil)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "portals") {
		t.Fatalf("expected ListPortals to include portals, got %q", body)
	}

	resp = workspacesWebRequest(t, ts, http.MethodPost, "/browserSettings", []byte(`{"displayName":"stage-browser"}`))
	assertStatus(t, resp, http.StatusOK)
	createBrowser := decodeWorkSpacesWebPayload(t, resp)
	browserARN := workspacesWebPayloadString(createBrowser, "browserSettingsArn")
	if browserARN == "" {
		t.Fatalf("expected CreateBrowserSettings to return browserSettingsArn")
	}

	resp = workspacesWebRequest(t, ts, http.MethodGet, "/browserSettings/"+url.PathEscape(browserARN), nil)
	assertStatus(t, resp, http.StatusOK)

	resp = workspacesWebRequest(t, ts, http.MethodPatch, "/browserSettings/"+url.PathEscape(browserARN), []byte(`{"description":"updated"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = workspacesWebRequest(t, ts, http.MethodDelete, "/browserSettings/"+url.PathEscape(browserARN), nil)
	assertStatus(t, resp, http.StatusOK)

	resp = workspacesWebRequest(t, ts, http.MethodDelete, "/portals/"+url.PathEscape(portalARN), nil)
	assertStatus(t, resp, http.StatusOK)
}

func TestWorkSpacesWebStage34AssociationsSessionAndTrustStoreSurface(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	portalARN := "arn:aws:workspaces-web:us-east-1:123456789012:portal/p-000001"
	browserARN := "arn:aws:workspaces-web:us-east-1:123456789012:browserSettings/bs-000001"
	trustStoreARN := "arn:aws:workspaces-web:us-east-1:123456789012:trustStore/ts-000001"

	resp := workspacesWebRequest(t, ts, http.MethodPut, "/portals/"+url.PathEscape(portalARN)+"/browserSettings?browserSettingsArn="+url.QueryEscape(browserARN), nil)
	assertStatus(t, resp, http.StatusOK)
	resp = workspacesWebRequest(t, ts, http.MethodDelete, "/portals/"+url.PathEscape(portalARN)+"/browserSettings", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = workspacesWebRequest(t, ts, http.MethodGet, "/portals/"+url.PathEscape(portalARN)+"/identityProviders?maxResults=10", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = workspacesWebRequest(t, ts, http.MethodGet, "/portals/p-000001/sessions?maxResults=10&status=ACTIVE", nil)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "sessions") {
		t.Fatalf("expected ListSessions to include sessions, got %q", body)
	}

	resp = workspacesWebRequest(t, ts, http.MethodDelete, "/portals/p-000001/sessions/s-000001", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = workspacesWebRequest(t, ts, http.MethodGet, "/portals/p-000001/sessions/s-000001", nil)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "EXPIRED") {
		t.Fatalf("expected GetSession to include EXPIRED status, got %q", body)
	}

	resp = workspacesWebRequest(t, ts, http.MethodGet, "/trustStores/"+url.PathEscape(trustStoreARN)+"/certificate?thumbprint=aa", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = workspacesWebRequest(t, ts, http.MethodGet, "/trustStores/"+url.PathEscape(trustStoreARN)+"/certificates?maxResults=10", nil)
	assertStatus(t, resp, http.StatusOK)
}

func TestWorkSpacesWebStage56TaggingValidationAndIdempotency(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resourceARN := "arn:aws:workspaces-web:us-east-1:123456789012:portal/p-000001"

	resp := workspacesWebRequest(t, ts, http.MethodPost, "/tags/"+url.PathEscape(resourceARN), []byte(`{"tags":{"env":"stage","owner":"qa"}}`))
	assertStatus(t, resp, http.StatusOK)

	resp = workspacesWebRequest(t, ts, http.MethodGet, "/tags/"+url.PathEscape(resourceARN), nil)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "owner") {
		t.Fatalf("expected ListTagsForResource to include owner tag, got %q", body)
	}

	resp = workspacesWebRequest(t, ts, http.MethodDelete, "/tags/"+url.PathEscape(resourceARN)+"?tagKeys=owner", nil)
	assertStatus(t, resp, http.StatusOK)

	browserARN := "arn:aws:workspaces-web:us-east-1:123456789012:browserSettings/bs-idempotent"
	resp = workspacesWebRequest(t, ts, http.MethodDelete, "/browserSettings/"+url.PathEscape(browserARN), nil)
	assertStatus(t, resp, http.StatusOK)
	resp = workspacesWebRequest(t, ts, http.MethodDelete, "/browserSettings/"+url.PathEscape(browserARN), nil)
	assertStatus(t, resp, http.StatusOK)

	resp = workspacesWebRequest(t, ts, http.MethodPost, "/workspaces-web/unknown", []byte(`{}`))
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException for unknown route, got %q", body)
	}

	resp = signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/portals",
		[]byte(`{"broken":`),
		map[string]string{"Content-Type": "application/json"},
		"workspaces-web",
	)
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "invalid JSON body") {
		t.Fatalf("expected invalid JSON body error, got %q", body)
	}
}

func decodeWorkSpacesWebPayload(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	payload := map[string]any{}
	if err := json.Unmarshal(mustBody(t, resp), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	return payload
}

func workspacesWebPayloadString(payload map[string]any, key string) string {
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
