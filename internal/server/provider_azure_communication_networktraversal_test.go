package server

import (
	"net/http"
	"testing"
	"time"
)

func TestAzureCommunicationNetworkTraversalLifecycle(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	headers := map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
	}
	jsonHeaders := map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
		"Content-Type":              "application/json",
	}

	base := "/azure/communication-networktraversal/2022-03-01-preview/networkTraversal/:issueRelayConfiguration?api-version=" + azureCommunicationNetworkTraversalAPIVersion

	resp := providerContractRequest(t, ts, http.MethodPost, base, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for default relay issuance, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	payload := providerContractJSONMap(t, resp)
	servers := payload["iceServers"].([]any)
	if len(servers) != 2 {
		t.Fatalf("expected 2 iceServers for default route issuance, got %#v", payload)
	}
	expiresOn := providerContractJSONString(t, payload["expiresOn"])
	if _, err := time.Parse(time.RFC3339Nano, expiresOn); err != nil {
		t.Fatalf("expected RFC3339 expiresOn, got %q err=%v", expiresOn, err)
	}
	foundAny := false
	foundNearest := false
	for _, raw := range servers {
		server := providerContractJSONMapFromAny(t, raw)
		urls := server["urls"].([]any)
		if len(urls) == 0 {
			t.Fatalf("expected urls in ice server, got %#v", server)
		}
		if providerContractJSONString(t, server["username"]) == "" || providerContractJSONString(t, server["credential"]) == "" {
			t.Fatalf("expected non-empty relay credentials, got %#v", server)
		}
		switch providerContractJSONString(t, server["routeType"]) {
		case "any":
			foundAny = true
		case "nearest":
			foundNearest = true
		}
	}
	if !foundAny || !foundNearest {
		t.Fatalf("expected any and nearest route types, got %#v", payload)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base, []byte(`{
		"id":"8:acs:000000000000000000000001",
		"routeType":"nearest",
		"ttl":3600
	}`), jsonHeaders)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for nearest relay issuance, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	nearestPayload := providerContractJSONMap(t, resp)
	nearestServers := nearestPayload["iceServers"].([]any)
	if len(nearestServers) != 1 {
		t.Fatalf("expected 1 iceServer for explicit nearest route, got %#v", nearestPayload)
	}
	nearestServer := providerContractJSONMapFromAny(t, nearestServers[0])
	if providerContractJSONString(t, nearestServer["routeType"]) != "nearest" {
		t.Fatalf("expected nearest route type, got %#v", nearestServer)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base, []byte(`{
		"routeType":"any",
		"ttl":999999
	}`), jsonHeaders)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for clamped ttl issuance, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	clampedPayload := providerContractJSONMap(t, resp)
	clampedExpiresOn := providerContractJSONString(t, clampedPayload["expiresOn"])
	clampedTime, err := time.Parse(time.RFC3339Nano, clampedExpiresOn)
	if err != nil {
		t.Fatalf("expected RFC3339 expiresOn, got %q err=%v", clampedExpiresOn, err)
	}
	ttlSeconds := int(time.Until(clampedTime).Seconds())
	if ttlSeconds < azureCommunicationNetworkTraversalDefaultTTLSeconds-10 || ttlSeconds > azureCommunicationNetworkTraversalDefaultTTLSeconds+10 {
		t.Fatalf("expected ttl around %d seconds, got %d payload=%#v", azureCommunicationNetworkTraversalDefaultTTLSeconds, ttlSeconds, clampedPayload)
	}
}

func TestAzureCommunicationNetworkTraversalValidation(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	headers := map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
	}
	jsonHeaders := map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
		"Content-Type":              "application/json",
	}

	tests := []struct {
		name       string
		method     string
		path       string
		body       []byte
		headers    map[string]string
		wantStatus int
		wantCode   string
	}{
		{
			name:       "invalid api version",
			method:     http.MethodPost,
			path:       "/azure/communication-networktraversal/2022-03-01-preview/networkTraversal/:issueRelayConfiguration?api-version=2022-01-01",
			body:       []byte(`{}`),
			headers:    jsonHeaders,
			wantStatus: http.StatusBadRequest,
			wantCode:   "InvalidApiVersion",
		},
		{
			name:       "invalid json",
			method:     http.MethodPost,
			path:       "/azure/communication-networktraversal/2022-03-01-preview/networkTraversal/:issueRelayConfiguration?api-version=2022-03-01-preview",
			body:       []byte(`{`),
			headers:    jsonHeaders,
			wantStatus: http.StatusBadRequest,
			wantCode:   "InvalidRequest",
		},
		{
			name:       "invalid route type",
			method:     http.MethodPost,
			path:       "/azure/communication-networktraversal/2022-03-01-preview/networkTraversal/:issueRelayConfiguration?api-version=2022-03-01-preview",
			body:       []byte(`{"routeType":"furthest"}`),
			headers:    jsonHeaders,
			wantStatus: http.StatusBadRequest,
			wantCode:   "InvalidRequest",
		},
		{
			name:       "invalid ttl type",
			method:     http.MethodPost,
			path:       "/azure/communication-networktraversal/2022-03-01-preview/networkTraversal/:issueRelayConfiguration?api-version=2022-03-01-preview",
			body:       []byte(`{"ttl":"3600"}`),
			headers:    jsonHeaders,
			wantStatus: http.StatusBadRequest,
			wantCode:   "InvalidRequest",
		},
		{
			name:       "negative ttl",
			method:     http.MethodPost,
			path:       "/azure/communication-networktraversal/2022-03-01-preview/networkTraversal/:issueRelayConfiguration?api-version=2022-03-01-preview",
			body:       []byte(`{"ttl":-1}`),
			headers:    jsonHeaders,
			wantStatus: http.StatusBadRequest,
			wantCode:   "InvalidRequest",
		},
		{
			name:       "invalid identity id",
			method:     http.MethodPost,
			path:       "/azure/communication-networktraversal/2022-03-01-preview/networkTraversal/:issueRelayConfiguration?api-version=2022-03-01-preview",
			body:       []byte(`{"id":"not-an-acs-id"}`),
			headers:    jsonHeaders,
			wantStatus: http.StatusBadRequest,
			wantCode:   "InvalidRequest",
		},
		{
			name:       "unknown path",
			method:     http.MethodPost,
			path:       "/azure/communication-networktraversal/2022-03-01-preview/networkTraversal/unknown?api-version=2022-03-01-preview",
			body:       []byte(`{}`),
			headers:    jsonHeaders,
			wantStatus: http.StatusNotFound,
			wantCode:   "PathNotFound",
		},
		{
			name:       "wrong method",
			method:     http.MethodGet,
			path:       "/azure/communication-networktraversal/2022-03-01-preview/networkTraversal/:issueRelayConfiguration?api-version=2022-03-01-preview",
			headers:    headers,
			wantStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp := providerContractRequest(t, ts, tc.method, tc.path, tc.body, tc.headers)
			if resp.StatusCode != tc.wantStatus {
				t.Fatalf("expected %d, got %d body=%s", tc.wantStatus, resp.StatusCode, string(providerContractBody(t, resp)))
			}
			if tc.wantCode == "" {
				return
			}
			payload := providerContractJSONMap(t, resp)
			errPayload := providerContractJSONMapFromAny(t, payload["error"])
			if errPayload["code"] != tc.wantCode {
				t.Fatalf("expected error code %q, got %#v", tc.wantCode, payload)
			}
		})
	}
}

func providerContractJSONString(t *testing.T, value any) string {
	t.Helper()
	typed, ok := value.(string)
	if !ok {
		t.Fatalf("expected string, got %#v", value)
	}
	return typed
}
