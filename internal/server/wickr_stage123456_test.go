package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestWickrStage12NetworkAndUserLifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	networkID := "n-000001"
	userID := "u-000001"

	resp := wickrRequest(t, ts, http.MethodPost, "/networks", []byte(`{"name":"stage-wickr-network"}`))
	assertStatus(t, resp, http.StatusOK)
	createNetworkPayload := decodeWickrPayload(t, resp)
	if got := wickrPayloadString(createNetworkPayload, "Network", "NetworkId"); got == "" {
		t.Fatalf("expected CreateNetwork to include Network.NetworkId")
	}

	resp = wickrRequest(t, ts, http.MethodGet, "/networks?maxResults=10", nil)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "Networks") {
		t.Fatalf("expected ListNetworks to include Networks, got %q", body)
	}

	resp = wickrRequest(t, ts, http.MethodGet, "/networks/"+url.PathEscape(networkID), nil)
	assertStatus(t, resp, http.StatusOK)
	resp = wickrRequest(t, ts, http.MethodPatch, "/networks/"+url.PathEscape(networkID), []byte(`{"name":"updated-network"}`))
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "updated-network") {
		t.Fatalf("expected UpdateNetwork response to include updated-network, got %q", body)
	}

	resp = wickrRequest(t, ts, http.MethodPost, "/networks/"+url.PathEscape(networkID)+"/users", []byte(`{"users":[{"username":"stage-user"}]}`))
	assertStatus(t, resp, http.StatusOK)
	resp = wickrRequest(t, ts, http.MethodGet, "/networks/"+url.PathEscape(networkID)+"/users?maxResults=10", nil)
	assertStatus(t, resp, http.StatusOK)
	resp = wickrRequest(t, ts, http.MethodGet, "/networks/"+url.PathEscape(networkID)+"/users/"+url.PathEscape(userID), nil)
	assertStatus(t, resp, http.StatusOK)
	resp = wickrRequest(t, ts, http.MethodGet, "/networks/"+url.PathEscape(networkID)+"/users/count", nil)
	assertStatus(t, resp, http.StatusOK)
	resp = wickrRequest(t, ts, http.MethodPatch, "/networks/"+url.PathEscape(networkID)+"/users", []byte(`{"status":"SUSPENDED"}`))
	assertStatus(t, resp, http.StatusOK)
	resp = wickrRequest(t, ts, http.MethodGet, "/networks/"+url.PathEscape(networkID)+"/users/"+url.PathEscape(userID)+"/devices?maxResults=10", nil)
	assertStatus(t, resp, http.StatusOK)
}

func TestWickrStage34BotsSecurityGroupsAndGuests(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	networkID := "n-000001"
	botID := "b-000001"
	groupID := "g-000001"
	usernameHash := "uh-000001"

	resp := wickrRequest(t, ts, http.MethodPost, "/networks/"+url.PathEscape(networkID)+"/bots", []byte(`{"name":"stage-bot"}`))
	assertStatus(t, resp, http.StatusOK)
	resp = wickrRequest(t, ts, http.MethodGet, "/networks/"+url.PathEscape(networkID)+"/bots?maxResults=10", nil)
	assertStatus(t, resp, http.StatusOK)
	resp = wickrRequest(t, ts, http.MethodGet, "/networks/"+url.PathEscape(networkID)+"/bots/"+url.PathEscape(botID), nil)
	assertStatus(t, resp, http.StatusOK)
	resp = wickrRequest(t, ts, http.MethodPatch, "/networks/"+url.PathEscape(networkID)+"/bots/"+url.PathEscape(botID), []byte(`{"status":"INACTIVE"}`))
	assertStatus(t, resp, http.StatusOK)
	resp = wickrRequest(t, ts, http.MethodDelete, "/networks/"+url.PathEscape(networkID)+"/bots/"+url.PathEscape(botID), nil)
	assertStatus(t, resp, http.StatusOK)

	resp = wickrRequest(t, ts, http.MethodPost, "/networks/"+url.PathEscape(networkID)+"/security-groups", []byte(`{"name":"stage-group"}`))
	assertStatus(t, resp, http.StatusOK)
	resp = wickrRequest(t, ts, http.MethodGet, "/networks/"+url.PathEscape(networkID)+"/security-groups?maxResults=10", nil)
	assertStatus(t, resp, http.StatusOK)
	resp = wickrRequest(t, ts, http.MethodGet, "/networks/"+url.PathEscape(networkID)+"/security-groups/"+url.PathEscape(groupID), nil)
	assertStatus(t, resp, http.StatusOK)
	resp = wickrRequest(t, ts, http.MethodGet, "/networks/"+url.PathEscape(networkID)+"/security-groups/"+url.PathEscape(groupID)+"/users?maxResults=10", nil)
	assertStatus(t, resp, http.StatusOK)
	resp = wickrRequest(t, ts, http.MethodPatch, "/networks/"+url.PathEscape(networkID)+"/security-groups/"+url.PathEscape(groupID), []byte(`{"name":"updated-group"}`))
	assertStatus(t, resp, http.StatusOK)
	resp = wickrRequest(t, ts, http.MethodDelete, "/networks/"+url.PathEscape(networkID)+"/security-groups/"+url.PathEscape(groupID), nil)
	assertStatus(t, resp, http.StatusOK)

	resp = wickrRequest(t, ts, http.MethodGet, "/networks/"+url.PathEscape(networkID)+"/guest-users?maxResults=10", nil)
	assertStatus(t, resp, http.StatusOK)
	resp = wickrRequest(t, ts, http.MethodPatch, "/networks/"+url.PathEscape(networkID)+"/guest-users/"+url.PathEscape(usernameHash), []byte(`{"action":"BLOCK"}`))
	assertStatus(t, resp, http.StatusOK)
	resp = wickrRequest(t, ts, http.MethodGet, "/networks/"+url.PathEscape(networkID)+"/guest-users/blocklist?maxResults=10", nil)
	assertStatus(t, resp, http.StatusOK)
	resp = wickrRequest(t, ts, http.MethodGet, "/networks/"+url.PathEscape(networkID)+"/guest-users/count", nil)
	assertStatus(t, resp, http.StatusOK)
}

func TestWickrStage56OIDCValidationAndIdempotency(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	networkID := "n-000001"

	resp := wickrRequest(t, ts, http.MethodPost, "/networks/"+url.PathEscape(networkID)+"/oidc/save", []byte(`{"clientId":"updated-client-id","url":"https://idp.example.com"}`))
	assertStatus(t, resp, http.StatusOK)
	resp = wickrRequest(t, ts, http.MethodPost, "/networks/"+url.PathEscape(networkID)+"/oidc/test", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	resp = wickrRequest(t, ts, http.MethodGet, "/networks/"+url.PathEscape(networkID)+"/oidc", nil)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "OidcConfigInfo") {
		t.Fatalf("expected GetOidcInfo response to include OidcConfigInfo, got %q", body)
	}

	resp = wickrRequest(t, ts, http.MethodDelete, "/networks/"+url.PathEscape(networkID), nil)
	assertStatus(t, resp, http.StatusOK)
	resp = wickrRequest(t, ts, http.MethodDelete, "/networks/"+url.PathEscape(networkID), nil)
	assertStatus(t, resp, http.StatusOK)

	resp = wickrRequest(t, ts, http.MethodGet, "/wickr/unknown", nil)
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException for unknown route, got %q", body)
	}

	resp = signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/networks",
		[]byte(`{"broken":`),
		map[string]string{"Content-Type": "application/json"},
		"wickr",
	)
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "invalid JSON body") {
		t.Fatalf("expected invalid JSON body error, got %q", body)
	}
}

func decodeWickrPayload(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	payload := map[string]any{}
	if err := json.Unmarshal(mustBody(t, resp), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	return payload
}

func wickrPayloadString(payload map[string]any, keys ...string) string {
	current := any(payload)
	for _, key := range keys {
		object, ok := current.(map[string]any)
		if !ok {
			return ""
		}
		next, ok := object[key]
		if !ok || next == nil {
			return ""
		}
		current = next
	}

	value, ok := current.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(value)
}
