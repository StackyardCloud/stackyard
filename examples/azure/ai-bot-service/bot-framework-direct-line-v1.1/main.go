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

type directLineClient struct {
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

func newDirectLineClient(endpoint, account string) *directLineClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"bot-framework-direct-line-v1.1",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&sharedKeyAuthPolicy{account: account}},
		},
		&policy.ClientOptions{},
	)
	return &directLineClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *directLineClient) doRequest(ctx context.Context, method, path string, payload any, expectedStatuses ...int) ([]byte, int, error) {
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

func (c *directLineClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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
	account := getenv("STACKYARD_AZURE_DIRECTLINE_ACCOUNT", "devstoreaccount1")
	userID := getenv("STACKYARD_AZURE_DIRECTLINE_USER_ID", "directline-user-1")

	fmt.Printf("Stackyard Azure AI Bot Service - Direct Line (bot-framework-direct-line-v1.1) example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newDirectLineClient(endpoint, account)

	const getTokensPath = "/azure/directline/v1.1/api/tokens"
	_, status, err := client.doJSON(ctx, http.MethodGet, getTokensPath, nil, http.StatusOK, http.StatusNotImplemented)
	if err != nil {
		exitf("GetTokens failed: %v", err)
	}
	if status == http.StatusNotImplemented {
		fmt.Println("Direct Line v1.1 routes are not implemented in this Stackyard build yet. Request envelope validated.")
		return
	}

	const generateTokenPath = "/azure/directline/v1.1/api/tokens/conversation"
	generateTokenResp, status, err := client.doJSON(ctx, http.MethodPost, generateTokenPath, map[string]any{
		"user": map[string]any{"id": userID},
	}, http.StatusOK, http.StatusNotImplemented)
	if err != nil {
		exitf("GenerateTokenForNewConversation failed: %v", err)
	}
	if status == http.StatusNotImplemented {
		notImplemented(generateTokenPath)
		return
	}

	token := asString(generateTokenResp["token"])

	const newConversationPath = "/azure/directline/v1.1/api/conversations"
	newConversationResp, status, err := client.doJSON(ctx, http.MethodPost, newConversationPath, map[string]any{
		"bot":  map[string]any{"id": "bot-1"},
		"user": map[string]any{"id": userID},
	}, http.StatusOK, http.StatusNotImplemented)
	if err != nil {
		exitf("NewConversation failed: %v", err)
	}
	if status == http.StatusNotImplemented {
		notImplemented(newConversationPath)
		return
	}

	conversationID := mustString(newConversationResp, "conversationId")

	renewTokenPath := "/azure/directline/v1.1/api/tokens/" + conversationID + "/renew"
	_, status, err = client.doJSON(ctx, http.MethodGet, renewTokenPath, nil, http.StatusOK, http.StatusNotImplemented)
	if err != nil {
		exitf("RenewToken failed: %v", err)
	}
	if status == http.StatusNotImplemented {
		notImplemented(renewTokenPath)
		return
	}

	postMessagePath := "/azure/directline/v1.1/api/conversations/" + conversationID + "/messages"
	postMessageResp, status, err := client.doJSON(ctx, http.MethodPost, postMessagePath, map[string]any{
		"text": "hello from stackyard direct line v1.1 sdk example",
		"from": userID,
	}, http.StatusOK, http.StatusNotImplemented)
	if err != nil {
		exitf("PostMessage failed: %v", err)
	}
	if status == http.StatusNotImplemented {
		notImplemented(postMessagePath)
		return
	}

	messageID := asString(postMessageResp["id"])
	getMessagesPath := postMessagePath + "?watermark=0"
	_, status, err = client.doJSON(ctx, http.MethodGet, getMessagesPath, nil, http.StatusOK, http.StatusNotImplemented)
	if err != nil {
		exitf("GetMessages failed: %v", err)
	}
	if status == http.StatusNotImplemented {
		notImplemented(getMessagesPath)
		return
	}

	if token != "" {
		fmt.Printf("Done. conversationId=%s token=%s messageId=%s\n", conversationID, token, messageID)
		return
	}
	fmt.Printf("Done. conversationId=%s messageId=%s\n", conversationID, messageID)
}

func notImplemented(path string) {
	fmt.Printf("Route is recognized but not implemented yet: %s\n", path)
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
