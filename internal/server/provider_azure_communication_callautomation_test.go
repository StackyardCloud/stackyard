package server

import (
	"net/http"
	"testing"
)

func TestAzureCommunicationCallAutomationLifecycle(t *testing.T) {
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

	base := "/azure/communication-callautomation/2026-03-12/calling"

	createBody := []byte(`{
		"source":{"rawId":"8:acs:source-a"},
		"targets":[
			{"rawId":"8:acs:target-a"},
			{"phoneNumber":{"phoneNumber":"+15551230001"}}
		],
		"callbackUri":"https://example.test/callbacks/call-a",
		"operationContext":"create-call-a"
	}`)
	resp := providerContractRequest(t, ts, http.MethodPost, base+"/callConnections?api-version="+azureCommunicationCallAutomationAPIVersion, createBody, repeatHeaders)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 for create call, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	createPayload := providerContractJSONMap(t, resp)
	callID := createPayload["callConnectionId"].(string)
	if callID == "" {
		t.Fatalf("expected callConnectionId, got %#v", createPayload)
	}
	if len(createPayload["participants"].([]any)) != 2 {
		t.Fatalf("expected two seeded participants, got %#v", createPayload)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"/callConnections?api-version="+azureCommunicationCallAutomationAPIVersion, createBody, repeatHeaders)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 for repeated create call, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	repeatPayload := providerContractJSONMap(t, resp)
	if repeatPayload["callConnectionId"] != callID {
		t.Fatalf("expected repeatable call id %q, got %#v", callID, repeatPayload)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/callConnections/"+callID+"?api-version="+azureCommunicationCallAutomationAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for get call, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"/callConnections/"+callID+"/participants:add?api-version="+azureCommunicationCallAutomationAPIVersion, []byte(`{
		"participant":{"communicationUser":{"id":"8:acs:new-user"}}
	}`), jsonHeaders)
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202 for add participant, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/callConnections/"+callID+"/participants?api-version="+azureCommunicationCallAutomationAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for list participants, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	participantsPayload := providerContractJSONMap(t, resp)
	if len(participantsPayload["value"].([]any)) != 3 {
		t.Fatalf("expected three participants, got %#v", participantsPayload)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/callConnections/"+callID+"/participants/8:acs:new-user?api-version="+azureCommunicationCallAutomationAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for get participant, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"/callConnections/"+callID+"/participants:cancelAddParticipant?api-version="+azureCommunicationCallAutomationAPIVersion, []byte(`{"invitationId":"invite-a"}`), jsonHeaders)
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202 for cancel add participant, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"/callConnections/"+callID+"/participants:remove?api-version="+azureCommunicationCallAutomationAPIVersion, []byte(`{
		"participant":{"communicationUser":{"id":"8:acs:new-user"}}
	}`), jsonHeaders)
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202 for remove participant, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"/callConnections/"+callID+":transferToParticipant?api-version="+azureCommunicationCallAutomationAPIVersion, []byte(`{
		"targetParticipant":{"rawId":"8:acs:transfer-target"}
	}`), jsonHeaders)
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202 for transfer, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	for _, path := range []string{
		base + "/callConnections/" + callID + ":mute?api-version=" + azureCommunicationCallAutomationAPIVersion,
		base + "/callConnections/" + callID + ":hold?api-version=" + azureCommunicationCallAutomationAPIVersion,
		base + "/callConnections/" + callID + ":play?api-version=" + azureCommunicationCallAutomationAPIVersion,
		base + "/callConnections/" + callID + ":recognize?api-version=" + azureCommunicationCallAutomationAPIVersion,
		base + "/callConnections/" + callID + ":sendDtmfTones?api-version=" + azureCommunicationCallAutomationAPIVersion,
		base + "/callConnections/" + callID + ":startContinuousDtmfRecognition?api-version=" + azureCommunicationCallAutomationAPIVersion,
		base + "/callConnections/" + callID + ":stopContinuousDtmfRecognition?api-version=" + azureCommunicationCallAutomationAPIVersion,
		base + "/callConnections/" + callID + ":startMediaStreaming?api-version=" + azureCommunicationCallAutomationAPIVersion,
		base + "/callConnections/" + callID + ":startTranscription?api-version=" + azureCommunicationCallAutomationAPIVersion,
		base + "/callConnections/" + callID + ":updateTranscription?api-version=" + azureCommunicationCallAutomationAPIVersion,
		base + "/callConnections/" + callID + ":stopMediaStreaming?api-version=" + azureCommunicationCallAutomationAPIVersion,
		base + "/callConnections/" + callID + ":stopTranscription?api-version=" + azureCommunicationCallAutomationAPIVersion,
		base + "/callConnections/" + callID + ":cancelAllMediaOperations?api-version=" + azureCommunicationCallAutomationAPIVersion,
		base + "/callConnections/" + callID + ":unhold?api-version=" + azureCommunicationCallAutomationAPIVersion,
	} {
		resp = providerContractRequest(t, ts, http.MethodPost, path, []byte(`{}`), jsonHeaders)
		if resp.StatusCode != http.StatusAccepted {
			t.Fatalf("expected 202 for action %s, got %d body=%s", path, resp.StatusCode, string(providerContractBody(t, resp)))
		}
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"/recordings?api-version="+azureCommunicationCallAutomationAPIVersion, []byte(`{
		"callConnectionId":"`+callID+`",
		"recordingChannelType":"mixed"
	}`), jsonHeaders)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for start recording, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	recordingPayload := providerContractJSONMap(t, resp)
	recordingID := recordingPayload["recordingId"].(string)
	if recordingID == "" {
		t.Fatalf("expected recordingId, got %#v", recordingPayload)
	}

	for _, path := range []string{
		base + "/recordings/" + recordingID + "?api-version=" + azureCommunicationCallAutomationAPIVersion,
		base + "/recordings/" + recordingID + ":pause?api-version=" + azureCommunicationCallAutomationAPIVersion,
		base + "/recordings/" + recordingID + ":resume?api-version=" + azureCommunicationCallAutomationAPIVersion,
	} {
		method := http.MethodGet
		if path != base+"/recordings/"+recordingID+"?api-version="+azureCommunicationCallAutomationAPIVersion {
			method = http.MethodPost
		}
		resp = providerContractRequest(t, ts, method, path, []byte(`{}`), jsonHeaders)
		wantStatus := http.StatusOK
		if method == http.MethodPost {
			wantStatus = http.StatusAccepted
		}
		if resp.StatusCode != wantStatus {
			t.Fatalf("expected %d for recording path %s, got %d body=%s", wantStatus, path, resp.StatusCode, string(providerContractBody(t, resp)))
		}
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"/callConnections:answer?api-version="+azureCommunicationCallAutomationAPIVersion, []byte(`{
		"incomingCallContext":"incoming-context-a",
		"callbackUri":"https://example.test/callbacks/inbound-a",
		"answeredBy":{"rawId":"8:acs:agent-a"}
	}`), jsonHeaders)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for answer call, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"/callConnections:connect?api-version="+azureCommunicationCallAutomationAPIVersion, []byte(`{
		"callLocator":{"serverCallId":"server-connect-a"},
		"callbackUri":"https://example.test/callbacks/connect-a"
	}`), jsonHeaders)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for connect, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"/callConnections:redirect?api-version="+azureCommunicationCallAutomationAPIVersion, []byte(`{
		"incomingCallContext":"incoming-context-b",
		"targetParticipant":{"rawId":"8:acs:redirect-target"}
	}`), jsonHeaders)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204 for redirect, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"/callConnections:reject?api-version="+azureCommunicationCallAutomationAPIVersion, []byte(`{
		"incomingCallContext":"incoming-context-c",
		"callRejectReason":"busy"
	}`), jsonHeaders)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204 for reject, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"/callConnections/"+callID+":terminate?api-version="+azureCommunicationCallAutomationAPIVersion, []byte(`{}`), jsonHeaders)
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202 for terminate, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodDelete, base+"/recordings/"+recordingID+"?api-version="+azureCommunicationCallAutomationAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204 for delete recording, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodDelete, base+"/callConnections/"+callID+"?api-version="+azureCommunicationCallAutomationAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204 for delete call, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
}

func TestAzureCommunicationCallAutomationValidation(t *testing.T) {
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
			path:       "/azure/communication-callautomation/2026-03-12/calling/callConnections?api-version=2025-01-01",
			body:       []byte(`{}`),
			headers:    jsonHeaders,
			wantStatus: http.StatusBadRequest,
			wantCode:   "InvalidApiVersion",
		},
		{
			name:       "missing create callback",
			method:     http.MethodPost,
			path:       "/azure/communication-callautomation/2026-03-12/calling/callConnections?api-version=2026-03-12",
			body:       []byte(`{"targets":[{"rawId":"8:acs:target-a"}]}`),
			headers:    jsonHeaders,
			wantStatus: http.StatusBadRequest,
			wantCode:   "InvalidRequest",
		},
		{
			name:       "missing answer context",
			method:     http.MethodPost,
			path:       "/azure/communication-callautomation/2026-03-12/calling/callConnections:answer?api-version=2026-03-12",
			body:       []byte(`{"callbackUri":"https://example.test/callbacks"}`),
			headers:    jsonHeaders,
			wantStatus: http.StatusBadRequest,
			wantCode:   "InvalidRequest",
		},
		{
			name:       "missing recording locator",
			method:     http.MethodPost,
			path:       "/azure/communication-callautomation/2026-03-12/calling/recordings?api-version=2026-03-12",
			body:       []byte(`{"recordingChannelType":"mixed"}`),
			headers:    jsonHeaders,
			wantStatus: http.StatusBadRequest,
			wantCode:   "InvalidRequest",
		},
		{
			name:       "missing call",
			method:     http.MethodGet,
			path:       "/azure/communication-callautomation/2026-03-12/calling/callConnections/missing?api-version=2026-03-12",
			headers:    headers,
			wantStatus: http.StatusNotFound,
			wantCode:   "CallConnectionNotFound",
		},
		{
			name:       "missing recording",
			method:     http.MethodGet,
			path:       "/azure/communication-callautomation/2026-03-12/calling/recordings/missing?api-version=2026-03-12",
			headers:    headers,
			wantStatus: http.StatusNotFound,
			wantCode:   "RecordingNotFound",
		},
		{
			name:       "invalid json",
			method:     http.MethodPost,
			path:       "/azure/communication-callautomation/2026-03-12/calling/callConnections?api-version=2026-03-12",
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
