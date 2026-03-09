package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
)

type botConnectorClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type sharedKeyAuthPolicy struct {
	account string
}

func (p *sharedKeyAuthPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	return req.Next()
}

func newBotConnectorClient(endpoint, account string) *botConnectorClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"bot-framework-bot-connector-v3.1",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&sharedKeyAuthPolicy{account: account}},
		},
		&policy.ClientOptions{},
	)
	return &botConnectorClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *botConnectorClient) doRequest(ctx context.Context, method, path string, payload any, expectedStatuses ...int) ([]byte, int, error) {
	req, err := runtime.NewRequest(ctx, method, c.endpoint+path)
	if err != nil {
		return nil, 0, fmt.Errorf("create request %s %s: %w", method, path, err)
	}
	req.Raw().Header.Set("Accept", "application/json")
	if payload != nil {
		if err := runtime.MarshalAsJSON(req, payload); err != nil {
			return nil, 0, fmt.Errorf("marshal payload %s %s: %w", method, path, err)
		}
	}

	resp, err := c.pipeline.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("execute request %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, resp.StatusCode, fmt.Errorf("read response body %s %s: %w", method, path, readErr)
	}

	if len(expectedStatuses) == 0 {
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return body, resp.StatusCode, nil
		}
		return body, resp.StatusCode, fmt.Errorf("unexpected status %d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	for _, status := range expectedStatuses {
		if resp.StatusCode == status {
			return body, resp.StatusCode, nil
		}
	}
	return body, resp.StatusCode, fmt.Errorf("unexpected status %d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
}

func (c *botConnectorClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
	body, status, err := c.doRequest(ctx, method, path, payload, expectedStatuses...)
	if err != nil {
		return nil, status, err
	}
	if len(strings.TrimSpace(string(body))) == 0 {
		return map[string]any{}, status, nil
	}
	var decoded map[string]any
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, status, fmt.Errorf("decode JSON body for %s %s: %w", method, path, err)
	}
	if decoded == nil {
		decoded = map[string]any{}
	}
	return decoded, status, nil
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	account := getenv("STACKYARD_AZURE_BOT_ACCOUNT", "devstoreaccount1")
	botID := getenv("STACKYARD_AZURE_BOT_ID", "bot-1")
	userID := getenv("STACKYARD_AZURE_BOT_USER_ID", "user-1")

	fmt.Printf("Stackyard Azure AI Bot Service - Bot Connector (bot-framework-bot-connector-v3.1) example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newBotConnectorClient(endpoint, account)

	createConversationPayload := map[string]any{
		"bot": map[string]any{"id": botID, "name": "Stackyard Bot"},
		"members": []map[string]any{
			{"id": userID, "name": "Stackyard User"},
			{"id": "user-2", "name": "Stackyard User 2"},
		},
		"activity": map[string]any{"type": "message", "text": "hello from stackyard sdk example"},
	}
	createConversationResp, _, err := client.doJSON(ctx, http.MethodPost, "/azure/botframework/v3/conversations", createConversationPayload, http.StatusCreated)
	if err != nil {
		exitf("CreateConversation failed: %v", err)
	}
	conversationID := mustString(createConversationResp, "id")

	createActivityResp, _, err := client.doJSON(ctx, http.MethodPost, "/azure/botframework/v3/conversations/"+conversationID+"/activities", map[string]any{
		"type": "message",
		"text": "workflow message",
	}, http.StatusOK)
	if err != nil {
		exitf("CreateActivity failed: %v", err)
	}
	activityID := mustString(createActivityResp, "id")

	replyResp, _, err := client.doJSON(ctx, http.MethodPost, "/azure/botframework/v3/conversations/"+conversationID+"/activities/"+activityID, map[string]any{
		"type": "message",
		"text": "acknowledged",
	}, http.StatusOK)
	if err != nil {
		exitf("ReplyToActivity failed: %v", err)
	}
	replyID := mustString(replyResp, "id")

	membersResp, _, err := client.doJSON(ctx, http.MethodGet, "/azure/botframework/v3/conversations/"+conversationID+"/members", nil, http.StatusOK)
	if err != nil {
		exitf("ListConversationMembers failed: %v", err)
	}
	if !hasMembers(membersResp) {
		exitf("ListConversationMembers returned no members: %#v", membersResp)
	}

	pagedResp, _, err := client.doJSON(ctx, http.MethodGet, "/azure/botframework/v3/conversations/"+conversationID+"/pagedmembers?pageSize=1", nil, http.StatusOK)
	if err != nil {
		exitf("GetPagedMembers first page failed: %v", err)
	}
	nextToken := asString(pagedResp["continuationToken"])
	if nextToken == "" {
		exitf("GetPagedMembers expected continuationToken on first page: %#v", pagedResp)
	}

	_, _, err = client.doJSON(ctx, http.MethodGet, "/azure/botframework/v3/conversations/"+conversationID+"/pagedmembers?pageSize=1&continuationToken="+nextToken, nil, http.StatusOK)
	if err != nil {
		exitf("GetPagedMembers second page failed: %v", err)
	}

	_, _, err = client.doJSON(ctx, http.MethodGet, "/azure/botframework/v3/conversations/"+conversationID+"/activities/"+replyID+"/members", nil, http.StatusOK)
	if err != nil {
		exitf("GetActivityMembers failed: %v", err)
	}

	_, _, err = client.doJSON(ctx, http.MethodPut, "/azure/botframework/v3/conversations/"+conversationID+"/activities/"+activityID, map[string]any{
		"type": "message",
		"text": "workflow message updated",
	}, http.StatusOK)
	if err != nil {
		exitf("UpdateActivity failed: %v", err)
	}

	_, _, err = client.doRequest(ctx, http.MethodDelete, "/azure/botframework/v3/conversations/"+conversationID+"/activities/"+activityID, nil, http.StatusNoContent)
	if err != nil {
		exitf("DeleteActivity failed: %v", err)
	}

	fmt.Println("Done.")
}

func hasMembers(payload map[string]any) bool {
	members, ok := payload["members"].([]any)
	return ok && len(members) > 0
}

func mustString(payload map[string]any, key string) string {
	value := asString(payload[key])
	if value == "" {
		exitf("missing required field %q in response payload: %#v", key, payload)
	}
	return value
}

func asString(value any) string {
	text, _ := value.(string)
	return strings.TrimSpace(text)
}

func getenv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
