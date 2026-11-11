package server

import (
	"net/http"
	"testing"
)

func TestAzureCommunicationRoomsLifecycle(t *testing.T) {
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
	repeatHeaders := map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
		"Content-Type":              "application/json",
		"Repeatability-Request-ID":  "6f9619ff-8b86-d011-b42d-00cf4fc964ff",
		"Repeatability-First-Sent":  "Sun, 12 Apr 2026 09:00:00 GMT",
	}

	base := "/azure/communication-rooms/2025-03-13/rooms"
	createBody := []byte(`{
		"validFrom":"2026-04-12T09:00:00Z",
		"validUntil":"2026-04-13T09:00:00Z",
		"pstnDialOutEnabled":true,
		"participants":[
			{"rawId":"8:acs:user-a","role":"Presenter"},
			{"rawId":"8:acs:user-b","role":"Attendee"}
		]
	}`)

	resp := providerContractRequest(t, ts, http.MethodPost, base+"?api-version="+azureCommunicationRoomsAPIVersion, createBody, repeatHeaders)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 for create room, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	createPayload := providerContractJSONMap(t, resp)
	roomID := providerContractJSONString(t, createPayload["id"])
	if roomID == "" {
		t.Fatalf("expected room id, got %#v", createPayload)
	}
	if createdAt := providerContractJSONString(t, createPayload["createdAt"]); createdAt == "" {
		t.Fatalf("expected createdAt, got %#v", createPayload)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"?api-version="+azureCommunicationRoomsAPIVersion, createBody, repeatHeaders)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 for repeated create room, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	repeatPayload := providerContractJSONMap(t, resp)
	if providerContractJSONString(t, repeatPayload["id"]) != roomID {
		t.Fatalf("expected repeatable room id %q, got %#v", roomID, repeatPayload)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"?api-version="+azureCommunicationRoomsAPIVersion+"&maxPageSize=1", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for list rooms, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	listPayload := providerContractJSONMap(t, resp)
	if len(listPayload["value"].([]any)) != 1 {
		t.Fatalf("expected one room, got %#v", listPayload)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/"+roomID+"?api-version="+azureCommunicationRoomsAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for get room, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/"+roomID+"/participants?api-version="+azureCommunicationRoomsAPIVersion+"&maxPageSize=1", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for list participants, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	participantsPayload := providerContractJSONMap(t, resp)
	if len(participantsPayload["value"].([]any)) != 1 {
		t.Fatalf("expected one paged participant, got %#v", participantsPayload)
	}
	if participantsPayload["nextLink"] == nil {
		t.Fatalf("expected nextLink for paged participants, got %#v", participantsPayload)
	}

	resp = providerContractRequest(t, ts, http.MethodPatch, base+"/"+roomID+"/participants?api-version="+azureCommunicationRoomsAPIVersion, []byte(`{
		"participants":{
			"8:acs:user-c":{"role":"Consumer"},
			"8:acs:user-b":{"role":"Collaborator"},
			"8:acs:user-a":null
		}
	}`), jsonHeaders)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for patch participants, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/"+roomID+"/participants?api-version="+azureCommunicationRoomsAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for participant list after patch, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	participantsPayload = providerContractJSONMap(t, resp)
	value := participantsPayload["value"].([]any)
	if len(value) != 2 {
		t.Fatalf("expected two participants after patch, got %#v", participantsPayload)
	}

	resp = providerContractRequest(t, ts, http.MethodPatch, base+"/"+roomID+"?api-version="+azureCommunicationRoomsAPIVersion, []byte(`{
		"pstnDialOutEnabled":false,
		"validUntil":"2026-04-14T09:00:00Z"
	}`), jsonHeaders)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for update room, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	updatePayload := providerContractJSONMap(t, resp)
	if updatePayload["pstnDialOutEnabled"] != false {
		t.Fatalf("expected updated pstnDialOutEnabled=false, got %#v", updatePayload)
	}

	resp = providerContractRequest(t, ts, http.MethodDelete, base+"/"+roomID+"?api-version="+azureCommunicationRoomsAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204 for delete room, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/"+roomID+"?api-version="+azureCommunicationRoomsAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for deleted room, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
}

func TestAzureCommunicationRoomsDefaultsAndValidation(t *testing.T) {
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

	base := "/azure/communication-rooms/2025-03-13/rooms"

	resp := providerContractRequest(t, ts, http.MethodPost, base+"?api-version="+azureCommunicationRoomsAPIVersion, []byte(`{}`), jsonHeaders)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 for default create, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	defaultPayload := providerContractJSONMap(t, resp)
	if defaultPayload["pstnDialOutEnabled"] != false {
		t.Fatalf("expected default pstnDialOutEnabled=false, got %#v", defaultPayload)
	}
	if providerContractJSONString(t, defaultPayload["validFrom"]) == "" || providerContractJSONString(t, defaultPayload["validUntil"]) == "" {
		t.Fatalf("expected default validity timestamps, got %#v", defaultPayload)
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
			path:       base + "?api-version=2024-01-01",
			headers:    headers,
			wantStatus: http.StatusBadRequest,
			wantCode:   "InvalidApiVersion",
		},
		{
			name:       "invalid json",
			method:     http.MethodPost,
			path:       base + "?api-version=" + azureCommunicationRoomsAPIVersion,
			body:       []byte(`{`),
			headers:    jsonHeaders,
			wantStatus: http.StatusBadRequest,
			wantCode:   "InvalidRequest",
		},
		{
			name:       "invalid validFrom",
			method:     http.MethodPost,
			path:       base + "?api-version=" + azureCommunicationRoomsAPIVersion,
			body:       []byte(`{"validFrom":"bad-time"}`),
			headers:    jsonHeaders,
			wantStatus: http.StatusBadRequest,
			wantCode:   "InvalidRequest",
		},
		{
			name:       "validUntil before validFrom",
			method:     http.MethodPost,
			path:       base + "?api-version=" + azureCommunicationRoomsAPIVersion,
			body:       []byte(`{"validFrom":"2026-04-13T09:00:00Z","validUntil":"2026-04-12T09:00:00Z"}`),
			headers:    jsonHeaders,
			wantStatus: http.StatusBadRequest,
			wantCode:   "InvalidRequest",
		},
		{
			name:       "invalid participant role",
			method:     http.MethodPost,
			path:       base + "?api-version=" + azureCommunicationRoomsAPIVersion,
			body:       []byte(`{"participants":[{"rawId":"8:acs:user-a","role":"Owner"}]}`),
			headers:    jsonHeaders,
			wantStatus: http.StatusBadRequest,
			wantCode:   "InvalidRequest",
		},
		{
			name:       "invalid maxPageSize",
			method:     http.MethodGet,
			path:       base + "?api-version=" + azureCommunicationRoomsAPIVersion + "&maxPageSize=0",
			headers:    headers,
			wantStatus: http.StatusBadRequest,
			wantCode:   "InvalidRequest",
		},
		{
			name:       "missing room participants patch",
			method:     http.MethodPatch,
			path:       base + "/missing-room/participants?api-version=" + azureCommunicationRoomsAPIVersion,
			body:       []byte(`{"participants":{}}`),
			headers:    jsonHeaders,
			wantStatus: http.StatusNotFound,
			wantCode:   "RoomNotFound",
		},
		{
			name:       "unknown path",
			method:     http.MethodGet,
			path:       "/azure/communication-rooms/2025-03-13/unknown?api-version=" + azureCommunicationRoomsAPIVersion,
			headers:    headers,
			wantStatus: http.StatusNotFound,
			wantCode:   "PathNotFound",
		},
		{
			name:       "wrong method",
			method:     http.MethodPost,
			path:       base + "/room-a/participants?api-version=" + azureCommunicationRoomsAPIVersion,
			body:       []byte(`{}`),
			headers:    jsonHeaders,
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
			if providerContractJSONString(t, errPayload["code"]) != tc.wantCode {
				t.Fatalf("expected error code %q, got %#v", tc.wantCode, payload)
			}
		})
	}
}
