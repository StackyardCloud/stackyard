package server

import (
	"net/http"
	"testing"
)

func TestAzureWebPubSubDataPlaneLifecycle(t *testing.T) {
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

	base := "/azure/webpubsub-data-plane/2024-12-01"

	resp := providerContractRequest(t, ts, http.MethodHead, base+"/api/health?api-version=2024-12-01", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for health, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"/api/hubs/chat/:generateToken?api-version=2024-12-01&userId=user-a&roles=webpubsub.joinLeaveGroup&groups=group-a&minutesToExpire=30", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for generate token, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	tokenPayload := providerContractJSONMap(t, resp)
	if tokenPayload["token"] == "" {
		t.Fatalf("expected token response, got %#v", tokenPayload)
	}

	resp = providerContractRequest(t, ts, http.MethodPut, base+"/api/hubs/chat/groups/group-a/connections/conn-a?api-version=2024-12-01&userId=user-a", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for add connection to group, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPut, base+"/api/hubs/chat/groups/group-b/connections/conn-b?api-version=2024-12-01", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for add second connection, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPut, base+"/api/hubs/chat/users/user-a/groups/group-a?api-version=2024-12-01", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for add user to group, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodHead, base+"/api/hubs/chat/connections/conn-a?api-version=2024-12-01", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for connection exists, got %d", resp.StatusCode)
	}

	resp = providerContractRequest(t, ts, http.MethodHead, base+"/api/hubs/chat/groups/group-a?api-version=2024-12-01", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for group exists, got %d", resp.StatusCode)
	}

	resp = providerContractRequest(t, ts, http.MethodHead, base+"/api/hubs/chat/users/user-a?api-version=2024-12-01", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for user exists, got %d", resp.StatusCode)
	}

	resp = providerContractRequest(t, ts, http.MethodPut, base+"/api/hubs/chat/permissions/sendToGroup/connections/conn-a?api-version=2024-12-01&targetName=group-a", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for grant permission, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodHead, base+"/api/hubs/chat/permissions/sendToGroup/connections/conn-a?api-version=2024-12-01&targetName=group-a", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for check permission, got %d", resp.StatusCode)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"/api/hubs/chat/:addToGroups?api-version=2024-12-01", []byte(`{
		"groups":["group-b","group-c"],
		"filter":"connectionId eq 'conn-a'"
	}`), jsonHeaders)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204 for bulk add groups, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/api/hubs/chat/groups/group-b/connections?api-version=2024-12-01&maxpagesize=1", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for list group connections, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	firstPage := providerContractJSONMap(t, resp)
	firstValue, ok := firstPage["value"].([]any)
	if !ok || len(firstValue) != 1 {
		t.Fatalf("expected one item on first page, got %#v", firstPage)
	}
	if firstPage["nextLink"] == "" {
		t.Fatalf("expected nextLink on first page, got %#v", firstPage)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/api/hubs/chat/groups/group-b/connections?api-version=2024-12-01&maxpagesize=1&continuationToken=1", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for second page, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	secondPage := providerContractJSONMap(t, resp)
	secondValue, ok := secondPage["value"].([]any)
	if !ok || len(secondValue) != 1 {
		t.Fatalf("expected one item on second page, got %#v", secondPage)
	}

	for _, path := range []string{
		base + "/api/hubs/chat/:send?api-version=2024-12-01",
		base + "/api/hubs/chat/groups/group-a/:send?api-version=2024-12-01",
		base + "/api/hubs/chat/users/user-a/:send?api-version=2024-12-01",
		base + "/api/hubs/chat/connections/conn-a/:send?api-version=2024-12-01",
	} {
		resp = providerContractRequest(t, ts, http.MethodPost, path, []byte(`{"message":"hello"}`), jsonHeaders)
		if resp.StatusCode != http.StatusNoContent {
			t.Fatalf("expected 204 for send route %s, got %d body=%s", path, resp.StatusCode, string(providerContractBody(t, resp)))
		}
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"/api/hubs/chat/:removeFromGroups?api-version=2024-12-01", []byte(`{
		"groups":["group-c"],
		"filter":"connectionId eq 'conn-a'"
	}`), jsonHeaders)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204 for bulk remove groups, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodDelete, base+"/api/hubs/chat/users/user-a/groups/group-a?api-version=2024-12-01", nil, headers)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204 for remove user from group, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodDelete, base+"/api/hubs/chat/connections/conn-a/groups?api-version=2024-12-01", nil, headers)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204 for remove connection from all groups, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"/api/hubs/chat/users/user-a/:closeConnections?api-version=2024-12-01", nil, headers)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204 for close user connections, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodHead, base+"/api/hubs/chat/connections/conn-a?api-version=2024-12-01", nil, headers)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 after closing user connections, got %d", resp.StatusCode)
	}

	resp = providerContractRequest(t, ts, http.MethodDelete, base+"/api/hubs/chat/connections/conn-b?api-version=2024-12-01", nil, headers)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204 for delete connection, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"/api/hubs/chat/:closeConnections?api-version=2024-12-01", nil, headers)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204 for close all connections, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
}

func TestAzureWebPubSubDataPlaneValidation(t *testing.T) {
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
			method:     http.MethodHead,
			path:       "/azure/webpubsub-data-plane/2024-12-01/api/health?api-version=2024-01-01",
			headers:    headers,
			wantStatus: http.StatusBadRequest,
			wantCode:   "InvalidApiVersion",
		},
		{
			name:       "invalid hub",
			method:     http.MethodPost,
			path:       "/azure/webpubsub-data-plane/2024-12-01/api/hubs/1bad/:generateToken?api-version=2024-12-01",
			headers:    headers,
			wantStatus: http.StatusBadRequest,
			wantCode:   "InvalidRequest",
		},
		{
			name:       "invalid bulk filter",
			method:     http.MethodPost,
			path:       "/azure/webpubsub-data-plane/2024-12-01/api/hubs/chat/:addToGroups?api-version=2024-12-01",
			body:       []byte(`{"groups":["group-a"],"filter":"contains(userId,'a')"}`),
			headers:    jsonHeaders,
			wantStatus: http.StatusBadRequest,
			wantCode:   "InvalidRequest",
		},
		{
			name:       "permission missing target name",
			method:     http.MethodHead,
			path:       "/azure/webpubsub-data-plane/2024-12-01/api/hubs/chat/permissions/sendToGroup/connections/conn-a?api-version=2024-12-01",
			headers:    headers,
			wantStatus: http.StatusBadRequest,
			wantCode:   "InvalidRequest",
		},
		{
			name:       "missing group list",
			method:     http.MethodGet,
			path:       "/azure/webpubsub-data-plane/2024-12-01/api/hubs/chat/groups/group-a/connections?api-version=2024-12-01",
			headers:    headers,
			wantStatus: http.StatusNotFound,
			wantCode:   "GroupNotFound",
		},
		{
			name:       "missing connection send",
			method:     http.MethodPost,
			path:       "/azure/webpubsub-data-plane/2024-12-01/api/hubs/chat/connections/missing/:send?api-version=2024-12-01",
			body:       []byte(`{"message":"hello"}`),
			headers:    jsonHeaders,
			wantStatus: http.StatusNotFound,
			wantCode:   "ConnectionNotFound",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := providerContractRequest(t, ts, tt.method, tt.path, tt.body, tt.headers)
			if resp.StatusCode != tt.wantStatus {
				t.Fatalf("expected %d, got %d body=%s", tt.wantStatus, resp.StatusCode, string(providerContractBody(t, resp)))
			}
			if tt.method == http.MethodHead {
				if got := resp.Header.Get("x-ms-error-code"); got != tt.wantCode {
					t.Fatalf("expected x-ms-error-code %q, got %q", tt.wantCode, got)
				}
				return
			}
			payload := providerContractJSONMap(t, resp)
			errorPayload, ok := payload["error"].(map[string]any)
			if !ok {
				t.Fatalf("expected error payload, got %#v", payload)
			}
			if errorPayload["code"] != tt.wantCode {
				t.Fatalf("expected error code %q, got %#v", tt.wantCode, payload)
			}
		})
	}
}
