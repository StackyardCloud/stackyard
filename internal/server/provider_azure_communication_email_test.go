package server

import (
	"net/http"
	"testing"
)

func TestAzureCommunicationEmailLifecycle(t *testing.T) {
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
	idempotentHeaders := map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
		"Content-Type":              "application/json",
		"Operation-Id":              "2a5f4d10-9b4d-4a70-98ef-7d93b42c1b70",
	}

	base := "/azure/communication-email/2025-09-01"
	sendBody := []byte(`{
		"senderAddress":"DoNotReply@contoso.com",
		"content":{
			"subject":"Stackyard hello",
			"plainText":"hello from stackyard",
			"html":"<html><body><p>hello from stackyard</p></body></html>"
		},
		"recipients":{
			"to":[{"address":"alice@example.com","displayName":"Alice"}],
			"cc":[{"address":"cc@example.com"}]
		},
		"attachments":[
			{
				"name":"note.txt",
				"contentType":"text/plain",
				"contentInBase64":"aGVsbG8="
			}
		],
		"replyTo":[{"address":"reply@example.com"}]
	}`)

	resp := providerContractRequest(t, ts, http.MethodPost, base+"/emails:send?api-version="+azureCommunicationEmailAPIVersion, sendBody, idempotentHeaders)
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202 for send email, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if got := resp.Header.Get("Operation-Location"); got == "" {
		t.Fatalf("expected Operation-Location header, got headers=%v", resp.Header)
	}
	if got := resp.Header.Get("Retry-After"); got == "" {
		t.Fatalf("expected Retry-After header, got headers=%v", resp.Header)
	}
	sendPayload := providerContractJSONMap(t, resp)
	if sendPayload["id"] != idempotentHeaders["Operation-Id"] {
		t.Fatalf("expected operation id %q, got %#v", idempotentHeaders["Operation-Id"], sendPayload)
	}
	if sendPayload["status"] != "Running" {
		t.Fatalf("expected Running send status, got %#v", sendPayload)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"/emails:send?api-version="+azureCommunicationEmailAPIVersion, sendBody, idempotentHeaders)
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202 for replay send email, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	replayPayload := providerContractJSONMap(t, resp)
	if replayPayload["id"] != idempotentHeaders["Operation-Id"] {
		t.Fatalf("expected replay operation id %q, got %#v", idempotentHeaders["Operation-Id"], replayPayload)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/emails/operations/"+idempotentHeaders["Operation-Id"]+"?api-version="+azureCommunicationEmailAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for first poll, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	firstPoll := providerContractJSONMap(t, resp)
	if firstPoll["status"] != "Running" {
		t.Fatalf("expected first poll Running, got %#v", firstPoll)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/emails/operations/"+idempotentHeaders["Operation-Id"]+"?api-version="+azureCommunicationEmailAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for second poll, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	secondPoll := providerContractJSONMap(t, resp)
	if secondPoll["status"] != "Succeeded" {
		t.Fatalf("expected second poll Succeeded, got %#v", secondPoll)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"/emails:send?api-version="+azureCommunicationEmailAPIVersion, sendBody, jsonHeaders)
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202 for generated operation id send, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	generatedPayload := providerContractJSONMap(t, resp)
	generatedID, _ := generatedPayload["id"].(string)
	if generatedID == "" {
		t.Fatalf("expected generated operation id, got %#v", generatedPayload)
	}
}

func TestAzureCommunicationEmailFailureAndValidation(t *testing.T) {
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

	resp := providerContractRequest(t, ts, http.MethodPost, "/azure/communication-email/2025-09-01/emails:send?api-version="+azureCommunicationEmailAPIVersion, []byte(`{
		"senderAddress":"DoNotReply@contoso.com",
		"content":{"subject":"force failure"},
		"recipients":{"to":[{"address":"alice@example.com"}]},
		"headers":{"x-stackyard-operation-status":"Failed"}
	}`), jsonHeaders)
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202 for seeded failure send, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	failedID := providerContractJSONMap(t, resp)["id"].(string)

	resp = providerContractRequest(t, ts, http.MethodGet, "/azure/communication-email/2025-09-01/emails/operations/"+failedID+"?api-version="+azureCommunicationEmailAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for first failure poll, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	resp = providerContractRequest(t, ts, http.MethodGet, "/azure/communication-email/2025-09-01/emails/operations/"+failedID+"?api-version="+azureCommunicationEmailAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for second failure poll, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	failedPayload := providerContractJSONMap(t, resp)
	if failedPayload["status"] != "Failed" {
		t.Fatalf("expected Failed status, got %#v", failedPayload)
	}
	if _, ok := failedPayload["error"].(map[string]any); !ok {
		t.Fatalf("expected error payload, got %#v", failedPayload)
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
			path:       "/azure/communication-email/2025-09-01/emails:send?api-version=2024-01-01",
			body:       []byte(`{}`),
			headers:    jsonHeaders,
			wantStatus: http.StatusBadRequest,
			wantCode:   "InvalidApiVersion",
		},
		{
			name:       "missing sender address",
			method:     http.MethodPost,
			path:       "/azure/communication-email/2025-09-01/emails:send?api-version=2025-09-01",
			body:       []byte(`{"content":{"subject":"hi"},"recipients":{"to":[{"address":"a@example.com"}]}}`),
			headers:    jsonHeaders,
			wantStatus: http.StatusBadRequest,
			wantCode:   "InvalidRequest",
		},
		{
			name:       "missing subject",
			method:     http.MethodPost,
			path:       "/azure/communication-email/2025-09-01/emails:send?api-version=2025-09-01",
			body:       []byte(`{"senderAddress":"DoNotReply@contoso.com","content":{},"recipients":{"to":[{"address":"a@example.com"}]}}`),
			headers:    jsonHeaders,
			wantStatus: http.StatusBadRequest,
			wantCode:   "InvalidRequest",
		},
		{
			name:       "missing to recipients",
			method:     http.MethodPost,
			path:       "/azure/communication-email/2025-09-01/emails:send?api-version=2025-09-01",
			body:       []byte(`{"senderAddress":"DoNotReply@contoso.com","content":{"subject":"hi"},"recipients":{"to":[]}}`),
			headers:    jsonHeaders,
			wantStatus: http.StatusBadRequest,
			wantCode:   "InvalidRequest",
		},
		{
			name:       "invalid attachment",
			method:     http.MethodPost,
			path:       "/azure/communication-email/2025-09-01/emails:send?api-version=2025-09-01",
			body:       []byte(`{"senderAddress":"DoNotReply@contoso.com","content":{"subject":"hi"},"recipients":{"to":[{"address":"a@example.com"}]},"attachments":[{"contentType":"text/plain","contentInBase64":"aGVsbG8="}]}`),
			headers:    jsonHeaders,
			wantStatus: http.StatusBadRequest,
			wantCode:   "InvalidRequest",
		},
		{
			name:       "invalid json",
			method:     http.MethodPost,
			path:       "/azure/communication-email/2025-09-01/emails:send?api-version=2025-09-01",
			body:       []byte(`{`),
			headers:    jsonHeaders,
			wantStatus: http.StatusBadRequest,
			wantCode:   "InvalidRequest",
		},
		{
			name:       "unknown operation",
			method:     http.MethodGet,
			path:       "/azure/communication-email/2025-09-01/emails/operations/does-not-exist?api-version=2025-09-01",
			headers:    headers,
			wantStatus: http.StatusNotFound,
			wantCode:   "EmailOperationNotFound",
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
