package server

import (
	"net/http"
	"testing"
	"time"
)

func TestAzurePlaywrightDataPlaneLifecycle(t *testing.T) {
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
		"Content-Type":              "application/json",
		"x-ms-client-request-id":    "req-123",
	}

	workspaceID := "11111111-1111-1111-1111-111111111111"
	accessTokenID := "token-abc"
	runID := "22222222-2222-2222-2222-222222222222"
	base := azurePlaywrightDataPlanePrefix + "/playwrightworkspaces/" + workspaceID
	expiryAt := time.Now().UTC().Add(24 * time.Hour).Format(time.RFC3339)

	resp := providerContractRequest(t, ts, http.MethodPut, base+"/access-tokens/"+accessTokenID+"?api-version="+azurePlaywrightDataPlaneAPIVersion, []byte(`{
		"name":"token-a",
		"expiryAt":"`+expiryAt+`"
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating access token, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	created := providerContractJSONMap(t, resp)
	if created["name"] != "token-a" || created["jwtToken"] == "" {
		t.Fatalf("unexpected create token payload: %#v", created)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/access-tokens/"+accessTokenID+"?api-version="+azurePlaywrightDataPlaneAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting access token, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/access-tokens?api-version="+azurePlaywrightDataPlaneAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing access tokens, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	tokenList := providerContractJSONMap(t, resp)
	if values, ok := tokenList["value"].([]any); !ok || len(values) != 1 {
		t.Fatalf("expected one token, got %#v", tokenList)
	}

	resp = providerContractRequest(t, ts, http.MethodPatch, base+"/test-runs/"+runID+"?api-version="+azurePlaywrightDataPlaneAPIVersion, []byte(`{
		"displayName":"Stackyard Run",
		"config":{"sdkLanguage":"TYPESCRIPT","maxWorkers":2},
		"ciConfig":{"providerName":"GitHubActions","branch":"main"}
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating test run, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	run := providerContractJSONMap(t, resp)
	if run["displayName"] != "Stackyard Run" {
		t.Fatalf("unexpected run payload: %#v", run)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/test-runs?api-version="+azurePlaywrightDataPlaneAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing test runs, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	runList := providerContractJSONMap(t, resp)
	if values, ok := runList["value"].([]any); !ok || len(values) != 1 {
		t.Fatalf("expected one run, got %#v", runList)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/browsers?api-version="+azurePlaywrightDataPlaneAPIVersion+"&os=linux&runId="+runID, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting browsers bootstrap result, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	browsers := providerContractJSONMap(t, resp)
	if browsers["browserConnectionUrl"] == "" {
		t.Fatalf("expected browser connection payload, got %#v", browsers)
	}
	if echoed := resp.Header.Get("x-ms-client-request-id"); echoed != "req-123" {
		t.Fatalf("expected echoed client request id, got %q", echoed)
	}

	resp = providerContractRequest(t, ts, http.MethodDelete, base+"/access-tokens/"+accessTokenID+"?api-version="+azurePlaywrightDataPlaneAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204 deleting access token, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
}

func TestAzurePlaywrightDataPlaneValidation(t *testing.T) {
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
		"Content-Type":              "application/json",
	}

	tests := []struct {
		name       string
		method     string
		path       string
		body       []byte
		wantStatus int
	}{
		{
			name:       "invalid api version",
			method:     http.MethodGet,
			path:       azurePlaywrightDataPlanePrefix + "/playwrightworkspaces/11111111-1111-1111-1111-111111111111/access-tokens?api-version=bad",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid workspace id",
			method:     http.MethodGet,
			path:       azurePlaywrightDataPlanePrefix + "/playwrightworkspaces/not-a-guid/access-tokens?api-version=" + azurePlaywrightDataPlaneAPIVersion,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid token name",
			method:     http.MethodPut,
			path:       azurePlaywrightDataPlanePrefix + "/playwrightworkspaces/11111111-1111-1111-1111-111111111111/access-tokens/token-abc?api-version=" + azurePlaywrightDataPlaneAPIVersion,
			body:       []byte(`{"name":"bad_name","expiryAt":"2026-04-06T00:00:00Z"}`),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing browser os",
			method:     http.MethodGet,
			path:       azurePlaywrightDataPlanePrefix + "/playwrightworkspaces/11111111-1111-1111-1111-111111111111/browsers?api-version=" + azurePlaywrightDataPlaneAPIVersion,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing run for browser bootstrap",
			method:     http.MethodGet,
			path:       azurePlaywrightDataPlanePrefix + "/playwrightworkspaces/11111111-1111-1111-1111-111111111111/browsers?api-version=" + azurePlaywrightDataPlaneAPIVersion + "&os=linux&runId=22222222-2222-2222-2222-222222222222",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := providerContractRequest(t, ts, tt.method, tt.path, tt.body, headers)
			if resp.StatusCode != tt.wantStatus {
				t.Fatalf("expected %d, got %d body=%s", tt.wantStatus, resp.StatusCode, string(providerContractBody(t, resp)))
			}
		})
	}
}
