package server

import (
	"net/http"
	"strings"
	"testing"
)

func TestAzureBotFrameworkConversationActivityAndMembersLifecycle(t *testing.T) {
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
		"Authorization": "SharedKey devstoreaccount1:signature",
		"Content-Type":  "application/json",
	}

	createBody := []byte(`{
		"bot": {"id": "bot-1", "name": "Stackyard Bot"},
		"members": [{"id": "user-1", "name": "Stackyard User"}],
		"activity": {"type": "message", "text": "hello"}
	}`)
	resp := providerContractRequest(t, ts, http.MethodPost, "/azure/botframework/v3/conversations", createBody, headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating conversation, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	createPayload := providerContractJSONMap(t, resp)
	conversationID, _ := createPayload["id"].(string)
	if conversationID == "" {
		t.Fatalf("expected conversation id in create response, got %#v", createPayload)
	}
	if createPayload["activityId"] == nil {
		t.Fatalf("expected activityId in create response, got %#v", createPayload)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, "/azure/botframework/v3/conversations/"+conversationID+"/activities", []byte(`{"type":"message","text":"task created"}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 creating activity, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	activityPayload := providerContractJSONMap(t, resp)
	activityID, _ := activityPayload["id"].(string)
	if activityID == "" {
		t.Fatalf("expected activity id in create response, got %#v", activityPayload)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, "/azure/botframework/v3/conversations/"+conversationID+"/activities/"+activityID, []byte(`{"type":"message","text":"ack"}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 replying to activity, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	replyPayload := providerContractJSONMap(t, resp)
	replyID, _ := replyPayload["id"].(string)
	if replyID == "" {
		t.Fatalf("expected reply id in response, got %#v", replyPayload)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, "/azure/botframework/v3/conversations/"+conversationID+"/members", nil, map[string]string{
		"Authorization": "SharedKey devstoreaccount1:signature",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing members, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	membersPayload := providerContractJSONMap(t, resp)
	members, ok := membersPayload["members"].([]any)
	if !ok || len(members) == 0 {
		t.Fatalf("expected members in payload, got %#v", membersPayload)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, "/azure/botframework/v3/conversations/"+conversationID+"/activities/"+replyID+"/members", nil, map[string]string{
		"Authorization": "SharedKey devstoreaccount1:signature",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing activity members, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	activityMembers := providerContractJSONMap(t, resp)
	if _, ok := activityMembers["members"].([]any); !ok {
		t.Fatalf("expected members list in activity payload, got %#v", activityMembers)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, "/azure/botframework/v3/conversations/"+conversationID+"/pagedmembers?pageSize=1", nil, map[string]string{
		"Authorization": "SharedKey devstoreaccount1:signature",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing paged members, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	paged := providerContractJSONMap(t, resp)
	pageMembers, ok := paged["members"].([]any)
	if !ok || len(pageMembers) != 1 {
		t.Fatalf("expected one member on first page, got %#v", paged)
	}
	token, _ := paged["continuationToken"].(string)
	if token == "" {
		t.Fatalf("expected continuation token from first page, got %#v", paged)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, "/azure/botframework/v3/conversations/"+conversationID+"/pagedmembers?pageSize=1&continuationToken="+token, nil, map[string]string{
		"Authorization": "SharedKey devstoreaccount1:signature",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing paged members second page, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPut, "/azure/botframework/v3/conversations/"+conversationID+"/activities/"+activityID, []byte(`{"type":"message","text":"task updated"}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 updating activity, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodDelete, "/azure/botframework/v3/conversations/"+conversationID+"/activities/"+activityID, nil, map[string]string{
		"Authorization": "SharedKey devstoreaccount1:signature",
	})
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204 deleting activity, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
}

func TestAzureBotFrameworkValidationAndNotFound(t *testing.T) {
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
		"Authorization": "SharedKey devstoreaccount1:signature",
		"Content-Type":  "application/json",
	}

	resp := providerContractRequest(t, ts, http.MethodPost, "/azure/botframework/v3/conversations", []byte(`{"bot":`), headers)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid JSON, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	invalidJSONBody := providerContractJSONMap(t, resp)
	if invalidJSONBody["error"] != "InvalidRequest" {
		t.Fatalf("expected InvalidRequest error, got %#v", invalidJSONBody)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, "/azure/botframework/v3/conversations/missing/activities", []byte(`{"type":"message","text":"hello"}`), headers)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 creating activity in missing conversation, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	notFound := providerContractJSONMap(t, resp)
	if notFound["error"] != "ConversationNotFound" {
		t.Fatalf("expected ConversationNotFound error, got %#v", notFound)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, "/azure/botframework/v3/conversations/seed", []byte(`{"members":[{"id":"user-1"}]}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating explicit conversation id, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	_ = providerContractBody(t, resp)

	resp = providerContractRequest(t, ts, http.MethodPost, "/azure/botframework/v3/conversations/seed/activities", []byte(`{}`), headers)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty activity payload, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	invalidActivity := providerContractJSONMap(t, resp)
	if invalidActivity["error"] != "InvalidRequest" {
		t.Fatalf("expected InvalidRequest for empty activity payload, got %#v", invalidActivity)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, "/azure/botframework/v3/conversations/seed/pagedmembers?pageSize=0", nil, map[string]string{
		"Authorization": "SharedKey devstoreaccount1:signature",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid pageSize, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	pagedErr := providerContractJSONMap(t, resp)
	if pagedErr["error"] != "InvalidRequest" {
		t.Fatalf("expected InvalidRequest for invalid pageSize, got %#v", pagedErr)
	}
}

func TestAzureBotFrameworkRouterUnsupportedRoutesReturnNotImplemented(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/azure/botframework/v3/channels", nil, map[string]string{
		"Authorization": "SharedKey devstoreaccount1:signature",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for unsupported botframework route, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"azure"`) || !strings.Contains(body, "/azure/botframework/v3/channels") {
		t.Fatalf("unexpected not implemented body: %s", body)
	}
}
