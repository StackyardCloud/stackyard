package server

import (
	"net/http"
	"testing"
)

func TestAzureCommunicationChatLifecycle(t *testing.T) {
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

	base := "/azure/communication-chat/2025-03-15/chat/threads"

	createBody := []byte(`{
		"topic":"stackyard-thread",
		"metadata":{"env":"test"},
		"participants":[
			{
				"communicationIdentifier":{"rawId":"8:acs:user-a"},
				"displayName":"User A"
			},
			{
				"communicationIdentifier":{"phoneNumber":{"phoneNumber":"+15551230001"}},
				"displayName":"User B"
			}
		]
	}`)
	resp := providerContractRequest(t, ts, http.MethodPost, base+"?api-version="+azureCommunicationChatAPIVersion, createBody, repeatHeaders)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 for create thread, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	createPayload := providerContractJSONMap(t, resp)
	thread := providerContractJSONMapFromAny(t, createPayload["chatThread"])
	threadID := thread["id"].(string)
	if threadID == "" {
		t.Fatalf("expected thread id, got %#v", createPayload)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"?api-version="+azureCommunicationChatAPIVersion, createBody, repeatHeaders)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 for repeated create thread, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	repeatPayload := providerContractJSONMap(t, resp)
	repeatThread := providerContractJSONMapFromAny(t, repeatPayload["chatThread"])
	if repeatThread["id"] != threadID {
		t.Fatalf("expected repeatable thread id %q, got %#v", threadID, repeatPayload)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"?api-version="+azureCommunicationChatAPIVersion+"&maxPageSize=1", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for list threads, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	listThreadsPayload := providerContractJSONMap(t, resp)
	if len(listThreadsPayload["value"].([]any)) != 1 {
		t.Fatalf("expected one thread, got %#v", listThreadsPayload)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/"+threadID+"?api-version="+azureCommunicationChatAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for get thread, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPatch, base+"/"+threadID+"?api-version="+azureCommunicationChatAPIVersion, []byte(`{
		"topic":"stackyard-thread-updated",
		"metadata":{"env":"stage"}
	}`), jsonHeaders)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204 for update thread, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"/"+threadID+"/participants/:add?api-version="+azureCommunicationChatAPIVersion, []byte(`{
		"participants":[
			{
				"communicationIdentifier":{"rawId":"8:acs:user-c"},
				"displayName":"User C",
				"shareHistoryTime":"2026-04-12T09:00:00Z"
			}
		]
	}`), jsonHeaders)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 for add participants, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/"+threadID+"/participants?api-version="+azureCommunicationChatAPIVersion+"&maxPageSize=2&skip=1", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for list participants, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	listParticipantsPayload := providerContractJSONMap(t, resp)
	if len(listParticipantsPayload["value"].([]any)) != 2 {
		t.Fatalf("expected two paged participants, got %#v", listParticipantsPayload)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"/"+threadID+"/messages?api-version="+azureCommunicationChatAPIVersion, []byte(`{
		"content":"hello from stackyard",
		"type":"text",
		"metadata":{"kind":"greeting"},
		"senderDisplayName":"Stackyard User"
	}`), jsonHeaders)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 for send message, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if resp.Header.Get("Location") == "" {
		t.Fatalf("expected location header for send message, got headers=%v", resp.Header)
	}
	sendPayload := providerContractJSONMap(t, resp)
	messageID := sendPayload["id"].(string)
	if messageID == "" {
		t.Fatalf("expected message id, got %#v", sendPayload)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/"+threadID+"/messages/"+messageID+"?api-version="+azureCommunicationChatAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for get message, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/"+threadID+"/messages?api-version="+azureCommunicationChatAPIVersion+"&maxPageSize=1", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for list messages, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	listMessagesPayload := providerContractJSONMap(t, resp)
	if len(listMessagesPayload["value"].([]any)) != 1 {
		t.Fatalf("expected one message, got %#v", listMessagesPayload)
	}

	resp = providerContractRequest(t, ts, http.MethodPatch, base+"/"+threadID+"/messages/"+messageID+"?api-version="+azureCommunicationChatAPIVersion, []byte(`{
		"content":"edited hello",
		"metadata":{"kind":"edited"}
	}`), jsonHeaders)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204 for update message, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"/"+threadID+"/readReceipts?api-version="+azureCommunicationChatAPIVersion, []byte(`{
		"chatMessageId":"`+messageID+`"
	}`), jsonHeaders)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for send read receipt, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/"+threadID+"/readReceipts?api-version="+azureCommunicationChatAPIVersion+"&maxPageSize=1", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for list read receipts, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	listReceiptsPayload := providerContractJSONMap(t, resp)
	if len(listReceiptsPayload["value"].([]any)) != 1 {
		t.Fatalf("expected one read receipt, got %#v", listReceiptsPayload)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"/"+threadID+"/typing?api-version="+azureCommunicationChatAPIVersion, []byte(`{}`), jsonHeaders)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for typing notification, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodDelete, base+"/"+threadID+"/messages/"+messageID+"?api-version="+azureCommunicationChatAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204 for delete message, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"/"+threadID+"/participants/:remove?api-version="+azureCommunicationChatAPIVersion, []byte(`{
		"communicationIdentifier":{"rawId":"8:acs:user-c"}
	}`), jsonHeaders)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204 for remove participant, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodDelete, base+"/"+threadID+"?api-version="+azureCommunicationChatAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204 for delete thread, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
}

func TestAzureCommunicationChatValidation(t *testing.T) {
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
			method:     http.MethodGet,
			path:       "/azure/communication-chat/2025-03-15/chat/threads?api-version=2024-01-01",
			headers:    headers,
			wantStatus: http.StatusBadRequest,
			wantCode:   "InvalidApiVersion",
		},
		{
			name:       "missing topic",
			method:     http.MethodPost,
			path:       "/azure/communication-chat/2025-03-15/chat/threads?api-version=2025-03-15",
			body:       []byte(`{"metadata":{"env":"test"}}`),
			headers:    jsonHeaders,
			wantStatus: http.StatusBadRequest,
			wantCode:   "InvalidRequest",
		},
		{
			name:       "invalid maxPageSize",
			method:     http.MethodGet,
			path:       "/azure/communication-chat/2025-03-15/chat/threads?api-version=2025-03-15&maxPageSize=0",
			headers:    headers,
			wantStatus: http.StatusBadRequest,
			wantCode:   "InvalidRequest",
		},
		{
			name:       "invalid startTime",
			method:     http.MethodGet,
			path:       "/azure/communication-chat/2025-03-15/chat/threads?api-version=2025-03-15&startTime=not-a-time",
			headers:    headers,
			wantStatus: http.StatusBadRequest,
			wantCode:   "InvalidRequest",
		},
		{
			name:       "missing message content",
			method:     http.MethodPost,
			path:       "/azure/communication-chat/2025-03-15/chat/threads/thread-a/messages?api-version=2025-03-15",
			body:       []byte(`{}`),
			headers:    jsonHeaders,
			wantStatus: http.StatusBadRequest,
			wantCode:   "InvalidRequest",
		},
		{
			name:       "missing read receipt message id",
			method:     http.MethodPost,
			path:       "/azure/communication-chat/2025-03-15/chat/threads/thread-a/readReceipts?api-version=2025-03-15",
			body:       []byte(`{}`),
			headers:    jsonHeaders,
			wantStatus: http.StatusBadRequest,
			wantCode:   "InvalidRequest",
		},
		{
			name:       "missing thread",
			method:     http.MethodGet,
			path:       "/azure/communication-chat/2025-03-15/chat/threads/missing?api-version=2025-03-15",
			headers:    headers,
			wantStatus: http.StatusNotFound,
			wantCode:   "ChatThreadNotFound",
		},
		{
			name:       "invalid json",
			method:     http.MethodPost,
			path:       "/azure/communication-chat/2025-03-15/chat/threads?api-version=2025-03-15",
			body:       []byte(`{`),
			headers:    jsonHeaders,
			wantStatus: http.StatusBadRequest,
			wantCode:   "InvalidRequest",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			resp := providerContractRequest(t, ts, tt.method, tt.path, tt.body, tt.headers)
			if resp.StatusCode != tt.wantStatus {
				t.Fatalf("expected %d, got %d body=%s", tt.wantStatus, resp.StatusCode, string(providerContractBody(t, resp)))
			}
			payload := providerContractJSONMap(t, resp)
			errorPayload, ok := payload["error"].(map[string]any)
			if !ok || errorPayload["code"] != tt.wantCode {
				t.Fatalf("expected error code %q, got %#v", tt.wantCode, payload)
			}
		})
	}
}
