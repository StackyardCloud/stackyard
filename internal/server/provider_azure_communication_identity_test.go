package server

import (
	"net/http"
	"testing"
)

func TestAzureCommunicationIdentityLifecycle(t *testing.T) {
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

	base := "/azure/communication-identity/2025-06-30"

	resp := providerContractRequest(t, ts, http.MethodPost, base+"/identities?api-version="+azureCommunicationIdentityAPIVersion, []byte(`{}`), jsonHeaders)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 for create identity, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	createPayload := providerContractJSONMap(t, resp)
	identityObj := providerContractJSONMapFromAny(t, createPayload["identity"])
	identityID := identityObj["id"].(string)
	if identityID == "" {
		t.Fatalf("expected identity id, got %#v", createPayload)
	}
	if _, ok := createPayload["accessToken"]; ok {
		t.Fatalf("did not expect access token on empty create payload, got %#v", createPayload)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"/identities?api-version="+azureCommunicationIdentityAPIVersion, []byte(`{
		"createTokenWithScopes":["chat","voip"],
		"expiresInMinutes":120
	}`), jsonHeaders)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 for create identity with token, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	createWithTokenPayload := providerContractJSONMap(t, resp)
	createAccessToken := providerContractJSONMapFromAny(t, createWithTokenPayload["accessToken"])
	if createAccessToken["token"] == "" {
		t.Fatalf("expected create token, got %#v", createWithTokenPayload)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"/identities/"+identityID+"/:issueAccessToken?api-version="+azureCommunicationIdentityAPIVersion, []byte(`{
		"scopes":["chat.join","voip.join"],
		"expiresInMinutes":180
	}`), jsonHeaders)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for issue token, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	issuePayload := providerContractJSONMap(t, resp)
	if issuePayload["token"] == "" {
		t.Fatalf("expected issued token, got %#v", issuePayload)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"/identities/"+identityID+"/:revokeAccessTokens?api-version="+azureCommunicationIdentityAPIVersion, []byte(`{}`), jsonHeaders)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204 for revoke tokens, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"/teamsUser/:exchangeAccessToken?api-version="+azureCommunicationIdentityAPIVersion, []byte(`{
		"token":"teams-user-token",
		"appId":"app-123",
		"userId":"user-123"
	}`), jsonHeaders)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for teams user exchange, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	teamsUserPayload := providerContractJSONMap(t, resp)
	if teamsUserPayload["token"] == "" {
		t.Fatalf("expected teams user exchange token, got %#v", teamsUserPayload)
	}

	resp = providerContractRequest(t, ts, http.MethodPut, base+"/access/teamsExtension/tenants/tenant-123/assignments/object-123?api-version="+azureCommunicationIdentityAPIVersion, []byte(`{
		"principalType":"resourceAccount",
		"clientIds":["client-a","client-b"]
	}`), jsonHeaders)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 for assignment create, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/access/teamsExtension/tenants/tenant-123/assignments/object-123?api-version="+azureCommunicationIdentityAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for assignment get, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	getAssignmentPayload := providerContractJSONMap(t, resp)
	if getAssignmentPayload["principalType"] != "resourceAccount" {
		t.Fatalf("expected principalType resourceAccount, got %#v", getAssignmentPayload)
	}

	resp = providerContractRequest(t, ts, http.MethodPut, base+"/access/teamsExtension/tenants/tenant-123/assignments/object-123?api-version="+azureCommunicationIdentityAPIVersion, []byte(`{
		"principalType":"user",
		"clientIds":["client-c"]
	}`), jsonHeaders)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for assignment update, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"/access/teamsExtension/:exchangeAccessToken?api-version="+azureCommunicationIdentityAPIVersion, []byte(`{}`), jsonHeaders)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for teams extension exchange, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	teamsExtensionPayload := providerContractJSONMap(t, resp)
	teamsExtensionIdentity := providerContractJSONMapFromAny(t, teamsExtensionPayload["identity"])
	if teamsExtensionIdentity["id"] == "" {
		t.Fatalf("expected teams extension identity, got %#v", teamsExtensionPayload)
	}

	resp = providerContractRequest(t, ts, http.MethodDelete, base+"/access/teamsExtension/tenants/tenant-123/assignments/object-123?api-version="+azureCommunicationIdentityAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204 for assignment delete, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodDelete, base+"/identities/"+identityID+"?api-version="+azureCommunicationIdentityAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204 for delete identity, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
}

func TestAzureCommunicationIdentityValidation(t *testing.T) {
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
			path:       "/azure/communication-identity/2025-06-30/identities?api-version=2024-01-01",
			body:       []byte(`{}`),
			headers:    jsonHeaders,
			wantStatus: http.StatusBadRequest,
			wantCode:   "InvalidApiVersion",
		},
		{
			name:       "invalid create scopes",
			method:     http.MethodPost,
			path:       "/azure/communication-identity/2025-06-30/identities?api-version=2025-06-30",
			body:       []byte(`{"createTokenWithScopes":["bad.scope"]}`),
			headers:    jsonHeaders,
			wantStatus: http.StatusBadRequest,
			wantCode:   "InvalidRequest",
		},
		{
			name:       "invalid expires in minutes",
			method:     http.MethodPost,
			path:       "/azure/communication-identity/2025-06-30/identities?api-version=2025-06-30",
			body:       []byte(`{"createTokenWithScopes":["chat"],"expiresInMinutes":30}`),
			headers:    jsonHeaders,
			wantStatus: http.StatusBadRequest,
			wantCode:   "InvalidRequest",
		},
		{
			name:       "invalid json",
			method:     http.MethodPost,
			path:       "/azure/communication-identity/2025-06-30/identities?api-version=2025-06-30",
			body:       []byte(`{`),
			headers:    jsonHeaders,
			wantStatus: http.StatusBadRequest,
			wantCode:   "InvalidRequest",
		},
		{
			name:       "issue token missing scopes",
			method:     http.MethodPost,
			path:       "/azure/communication-identity/2025-06-30/identities/id-123/:issueAccessToken?api-version=2025-06-30",
			body:       []byte(`{}`),
			headers:    jsonHeaders,
			wantStatus: http.StatusBadRequest,
			wantCode:   "InvalidRequest",
		},
		{
			name:       "issue token unknown identity",
			method:     http.MethodPost,
			path:       "/azure/communication-identity/2025-06-30/identities/id-123/:issueAccessToken?api-version=2025-06-30",
			body:       []byte(`{"scopes":["chat"]}`),
			headers:    jsonHeaders,
			wantStatus: http.StatusNotFound,
			wantCode:   "IdentityNotFound",
		},
		{
			name:       "revoke unknown identity",
			method:     http.MethodPost,
			path:       "/azure/communication-identity/2025-06-30/identities/id-123/:revokeAccessTokens?api-version=2025-06-30",
			body:       []byte(`{}`),
			headers:    jsonHeaders,
			wantStatus: http.StatusNotFound,
			wantCode:   "IdentityNotFound",
		},
		{
			name:       "teams user missing fields",
			method:     http.MethodPost,
			path:       "/azure/communication-identity/2025-06-30/teamsUser/:exchangeAccessToken?api-version=2025-06-30",
			body:       []byte(`{"token":"a"}`),
			headers:    jsonHeaders,
			wantStatus: http.StatusBadRequest,
			wantCode:   "InvalidRequest",
		},
		{
			name:       "assignment missing principal type",
			method:     http.MethodPut,
			path:       "/azure/communication-identity/2025-06-30/access/teamsExtension/tenants/tenant-123/assignments/object-123?api-version=2025-06-30",
			body:       []byte(`{}`),
			headers:    jsonHeaders,
			wantStatus: http.StatusBadRequest,
			wantCode:   "InvalidRequest",
		},
		{
			name:       "assignment invalid principal type",
			method:     http.MethodPut,
			path:       "/azure/communication-identity/2025-06-30/access/teamsExtension/tenants/tenant-123/assignments/object-123?api-version=2025-06-30",
			body:       []byte(`{"principalType":"bot"}`),
			headers:    jsonHeaders,
			wantStatus: http.StatusBadRequest,
			wantCode:   "InvalidRequest",
		},
		{
			name:       "assignment unknown get",
			method:     http.MethodGet,
			path:       "/azure/communication-identity/2025-06-30/access/teamsExtension/tenants/tenant-123/assignments/object-123?api-version=2025-06-30",
			headers:    headers,
			wantStatus: http.StatusNotFound,
			wantCode:   "TeamsExtensionAssignmentNotFound",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp := providerContractRequest(t, ts, tc.method, tc.path, tc.body, tc.headers)
			if resp.StatusCode != tc.wantStatus {
				t.Fatalf("expected %d, got %d body=%s", tc.wantStatus, resp.StatusCode, string(providerContractBody(t, resp)))
			}
			payload := providerContractJSONMap(t, resp)
			errPayload := providerContractJSONMapFromAny(t, payload["error"])
			if errPayload["code"] != tc.wantCode {
				t.Fatalf("expected error code %q, got %#v", tc.wantCode, payload)
			}
		})
	}
}
