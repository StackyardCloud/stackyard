package server

import (
	"net/http"
	"testing"
)

func TestAzureCommunicationSipRoutingLifecycle(t *testing.T) {
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
	patchHeaders := map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
		"Content-Type":              "application/merge-patch+json",
	}

	base := "/azure/communication-siprouting/2023-03-01/sip?api-version=" + azureCommunicationSipRoutingAPIVersion

	resp := providerContractRequest(t, ts, http.MethodGet, base, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for default get, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	payload := providerContractJSONMap(t, resp)
	trunks := providerContractJSONMapFromAny(t, payload["trunks"])
	if len(trunks) != 2 {
		t.Fatalf("expected seeded trunks, got %#v", payload)
	}
	routes := payload["routes"].([]any)
	if len(routes) != 2 {
		t.Fatalf("expected seeded routes, got %#v", payload)
	}

	resp = providerContractRequest(t, ts, http.MethodPatch, base, []byte(`{
		"trunks": {
			"pstn.contoso.com": {"sipSignalingPort": 5067},
			"backup.contoso.com": {"sipSignalingPort": 5070}
		},
		"routes": [
			{
				"name": "contoso-primary",
				"description": "Primary PSTN route",
				"numberPattern": "^\\\\+1206\\\\d{7}$",
				"trunks": ["pstn.contoso.com"]
			},
			{
				"name": "contoso-fallback",
				"numberPattern": "^\\\\+1\\\\d+$",
				"trunks": ["pstn.contoso.com", "backup.contoso.com"]
			}
		]
	}`), patchHeaders)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for patch, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	patched := providerContractJSONMap(t, resp)
	patchedTrunks := providerContractJSONMapFromAny(t, patched["trunks"])
	if _, ok := patchedTrunks["pstn.contoso.com"]; !ok {
		t.Fatalf("expected patched trunk inventory, got %#v", patched)
	}
	patchedRoutes := patched["routes"].([]any)
	if len(patchedRoutes) != 2 {
		t.Fatalf("expected patched routes, got %#v", patched)
	}

	resp = providerContractRequest(t, ts, http.MethodPatch, base, []byte(`{
		"routes": [
			{
				"name": "contoso-regional",
				"numberPattern": "^\\\\+44\\\\d+$",
				"trunks": ["backup.contoso.com"]
			}
		]
	}`), patchHeaders)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for partial patch, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for get after patch, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	after := providerContractJSONMap(t, resp)
	afterTrunks := providerContractJSONMapFromAny(t, after["trunks"])
	if len(afterTrunks) != 2 {
		t.Fatalf("expected trunks to remain after partial patch, got %#v", after)
	}
	afterRoutes := after["routes"].([]any)
	if len(afterRoutes) != 1 {
		t.Fatalf("expected routes to be replaced by partial patch, got %#v", after)
	}
	route := providerContractJSONMapFromAny(t, afterRoutes[0])
	if route["name"] != "contoso-regional" {
		t.Fatalf("expected replaced route set, got %#v", after)
	}
}

func TestAzureCommunicationSipRoutingValidation(t *testing.T) {
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
	patchHeaders := map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
		"Content-Type":              "application/merge-patch+json",
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
			method:     http.MethodGet,
			path:       "/azure/communication-siprouting/2023-03-01/sip?api-version=2022-01-01",
			headers:    headers,
			wantStatus: http.StatusBadRequest,
			wantCode:   "InvalidApiVersion",
		},
		{
			name:       "unknown path",
			method:     http.MethodGet,
			path:       "/azure/communication-siprouting/2023-03-01/unknown?api-version=2023-03-01",
			headers:    headers,
			wantStatus: http.StatusNotFound,
			wantCode:   "PathNotFound",
		},
		{
			name:       "wrong method",
			method:     http.MethodPost,
			path:       "/azure/communication-siprouting/2023-03-01/sip?api-version=2023-03-01",
			headers:    headers,
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "wrong content type",
			method:     http.MethodPatch,
			path:       "/azure/communication-siprouting/2023-03-01/sip?api-version=2023-03-01",
			body:       []byte(`{}`),
			headers:    jsonHeaders,
			wantStatus: http.StatusUnsupportedMediaType,
			wantCode:   "UnsupportedMediaType",
		},
		{
			name:       "invalid json",
			method:     http.MethodPatch,
			path:       "/azure/communication-siprouting/2023-03-01/sip?api-version=2023-03-01",
			body:       []byte(`{`),
			headers:    patchHeaders,
			wantStatus: http.StatusBadRequest,
			wantCode:   "InvalidRequest",
		},
		{
			name:       "invalid trunk key",
			method:     http.MethodPatch,
			path:       "/azure/communication-siprouting/2023-03-01/sip?api-version=2023-03-01",
			body:       []byte(`{"trunks":{"bad host":{"sipSignalingPort":5061}}}`),
			headers:    patchHeaders,
			wantStatus: http.StatusBadRequest,
			wantCode:   "InvalidRequest",
		},
		{
			name:       "invalid port",
			method:     http.MethodPatch,
			path:       "/azure/communication-siprouting/2023-03-01/sip?api-version=2023-03-01",
			body:       []byte(`{"trunks":{"pstn.contoso.com":{"sipSignalingPort":70000}}}`),
			headers:    patchHeaders,
			wantStatus: http.StatusBadRequest,
			wantCode:   "InvalidRequest",
		},
		{
			name:       "unknown route trunk",
			method:     http.MethodPatch,
			path:       "/azure/communication-siprouting/2023-03-01/sip?api-version=2023-03-01",
			body:       []byte(`{"routes":[{"name":"route-a","numberPattern":"^\\\\+1\\\\d+$","trunks":["missing.contoso.com"]}]}`),
			headers:    patchHeaders,
			wantStatus: http.StatusBadRequest,
			wantCode:   "InvalidRequest",
		},
		{
			name:       "missing route name",
			method:     http.MethodPatch,
			path:       "/azure/communication-siprouting/2023-03-01/sip?api-version=2023-03-01",
			body:       []byte(`{"routes":[{"numberPattern":"^\\\\+1\\\\d+$"}]}`),
			headers:    patchHeaders,
			wantStatus: http.StatusBadRequest,
			wantCode:   "InvalidRequest",
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
