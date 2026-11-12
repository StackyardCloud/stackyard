package server

import (
	"net/http"
	"strings"
	"testing"
)

func TestAzureCommunicationAdvancedMessagingLifecycle(t *testing.T) {
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

	base := "/azure/communication-advancedmessaging/2024-08-30/messages"
	channelID := "11111111-1111-4111-8111-111111111111"

	repeatHeaders := map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
		"Content-Type":              "application/json",
		"Repeatability-Request-ID":  "6f9619ff-8b86-d011-b42d-00cf4fc964ff",
		"Repeatability-First-Sent":  "Mon, 12 Apr 2026 09:00:00 GMT",
	}

	textBody := []byte(`{
		"kind":"text",
		"channelRegistrationId":"channel-registration-a",
		"to":["+15551230001","+15551230002"],
		"content":"hello from stackyard"
	}`)
	resp := providerContractRequest(t, ts, http.MethodPost, base+"/notifications:send?api-version="+azureCommunicationAdvancedMessagingAPIVersion, textBody, repeatHeaders)
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202 for text send, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	textPayload := providerContractJSONMap(t, resp)
	receipts, ok := textPayload["receipts"].([]any)
	if !ok || len(receipts) != 2 {
		t.Fatalf("expected two receipts, got %#v", textPayload)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"/notifications:send?api-version="+azureCommunicationAdvancedMessagingAPIVersion, textBody, repeatHeaders)
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202 for repeated text send, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	repeatPayload := providerContractJSONMap(t, resp)
	if strings.TrimSpace(providerContractJSONMapFromAny(t, receipts[0])["messageId"].(string)) != strings.TrimSpace(providerContractJSONMapFromAny(t, repeatPayload["receipts"].([]any)[0])["messageId"].(string)) {
		t.Fatalf("expected repeatable receipts, got first=%#v repeat=%#v", textPayload, repeatPayload)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"/notifications:send?api-version="+azureCommunicationAdvancedMessagingAPIVersion, []byte(`{
		"kind":"template",
		"channelRegistrationId":"channel-registration-a",
		"to":["+15551230003"],
		"template":{
			"name":"order_update",
			"language":{"code":"en_US"},
			"values":{"1":"order-123"}
		}
	}`), jsonHeaders)
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202 for template send, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"/notifications:send?api-version="+azureCommunicationAdvancedMessagingAPIVersion, []byte(`{
		"kind":"image",
		"channelRegistrationId":"channel-registration-a",
		"to":["+15551230004"],
		"mediaUri":"https://example.test/media/image-a.jpg",
		"caption":"hello"
	}`), jsonHeaders)
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202 for media send, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/channels/"+channelID+"/templates?api-version="+azureCommunicationAdvancedMessagingAPIVersion+"&maxpagesize=1", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for list templates, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	templatePayload := providerContractJSONMap(t, resp)
	value, ok := templatePayload["value"].([]any)
	if !ok || len(value) != 1 {
		t.Fatalf("expected one template on first page, got %#v", templatePayload)
	}
	if templatePayload["nextLink"] == nil || templatePayload["nextLink"] == "" {
		t.Fatalf("expected nextLink for paged templates, got %#v", templatePayload)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/streams/stream-a?api-version="+azureCommunicationAdvancedMessagingAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for get media, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractBody(t, resp)
	if resp.Header.Get("Content-Type") != "application/octet-stream" || string(body) != "stackyard-advanced-messaging-stream-a" {
		t.Fatalf("unexpected media response: status=%d content-type=%q body=%q", resp.StatusCode, resp.Header.Get("Content-Type"), string(body))
	}
}

func TestAzureCommunicationAdvancedMessagingValidation(t *testing.T) {
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
			path:       "/azure/communication-advancedmessaging/2024-08-30/messages/channels/11111111-1111-4111-8111-111111111111/templates?api-version=",
			headers:    headers,
			wantStatus: http.StatusBadRequest,
			wantCode:   "InvalidApiVersion",
		},
		{
			name:       "invalid channel uuid",
			method:     http.MethodGet,
			path:       "/azure/communication-advancedmessaging/2024-08-30/messages/channels/not-a-uuid/templates?api-version=2024-08-30",
			headers:    headers,
			wantStatus: http.StatusBadRequest,
			wantCode:   "InvalidRequest",
		},
		{
			name:       "invalid send kind",
			method:     http.MethodPost,
			path:       "/azure/communication-advancedmessaging/2024-08-30/messages/notifications:send?api-version=2024-08-30",
			body:       []byte(`{"kind":"unknown","channelRegistrationId":"chan","to":["+15551230001"]}`),
			headers:    jsonHeaders,
			wantStatus: http.StatusBadRequest,
			wantCode:   "InvalidRequest",
		},
		{
			name:       "missing template language",
			method:     http.MethodPost,
			path:       "/azure/communication-advancedmessaging/2024-08-30/messages/notifications:send?api-version=2024-08-30",
			body:       []byte(`{"kind":"template","channelRegistrationId":"chan","to":["+15551230001"],"template":{"name":"order_update"}}`),
			headers:    jsonHeaders,
			wantStatus: http.StatusBadRequest,
			wantCode:   "InvalidRequest",
		},
		{
			name:       "invalid json",
			method:     http.MethodPost,
			path:       "/azure/communication-advancedmessaging/2024-08-30/messages/notifications:send?api-version=2024-08-30",
			body:       []byte(`{`),
			headers:    jsonHeaders,
			wantStatus: http.StatusBadRequest,
			wantCode:   "InvalidRequest",
		},
		{
			name:       "missing stream",
			method:     http.MethodGet,
			path:       "/azure/communication-advancedmessaging/2024-08-30/messages/streams/missing?api-version=2024-08-30",
			headers:    headers,
			wantStatus: http.StatusNotFound,
			wantCode:   "StreamNotFound",
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
			if tt.wantCode != "" {
				errorPayload, ok := payload["error"].(map[string]any)
				if !ok || errorPayload["code"] != tt.wantCode {
					t.Fatalf("expected error code %q, got %#v", tt.wantCode, payload)
				}
			}
		})
	}
}

func providerContractJSONMapFromAny(t *testing.T, value any) map[string]any {
	t.Helper()
	object, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("expected map payload, got %#v", value)
	}
	return object
}
