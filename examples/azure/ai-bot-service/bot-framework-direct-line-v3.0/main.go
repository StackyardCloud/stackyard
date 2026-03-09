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
		"bot-framework-direct-line-v3.0",
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

	fmt.Printf("Stackyard Azure AI Bot Service - Direct Line (bot-framework-direct-line-v3.0) example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newDirectLineClient(endpoint, account)

	const tokensGeneratePath = "/azure/directline/v3/directline/tokens/generate"
	tokenResp, status, err := client.doJSON(ctx, http.MethodPost, tokensGeneratePath, map[string]any{
		"user": map[string]any{"id": userID},
	}, http.StatusOK, http.StatusNotImplemented)
	if err != nil {
		exitf("GenerateToken failed: %v", err)
	}
	if status == http.StatusNotImplemented {
		fmt.Println("Direct Line routes are not implemented in this Stackyard build yet. Request envelope validated.")
		return
	}
	token := asString(tokenResp["token"])

	const startConversationPath = "/azure/directline/v3/directline/conversations"
	conversationResp, status, err := client.doJSON(ctx, http.MethodPost, startConversationPath, map[string]any{
		"locale": "en-US",
	}, http.StatusOK, http.StatusNotImplemented)
	if err != nil {
		exitf("StartConversation failed: %v", err)
	}
	if status == http.StatusNotImplemented {
		notImplemented(startConversationPath)
		return
	}

	conversationID := mustString(conversationResp, "conversationId")
	if token == "" {
		token = asString(conversationResp["token"])
	}

	postActivityPath := "/azure/directline/v3/directline/conversations/" + conversationID + "/activities"
	postActivityResp, status, err := client.doJSON(ctx, http.MethodPost, postActivityPath, map[string]any{
		"type": "message",
		"from": map[string]any{"id": userID},
		"text": "hello from stackyard direct line sdk example",
	}, http.StatusOK, http.StatusNotImplemented)
	if err != nil {
		exitf("PostActivity failed: %v", err)
	}
	if status == http.StatusNotImplemented {
		notImplemented(postActivityPath)
		return
	}
	activityID := asString(postActivityResp["id"])

	getActivitiesPath := "/azure/directline/v3/directline/conversations/" + conversationID + "/activities"
	_, status, err = client.doJSON(ctx, http.MethodGet, getActivitiesPath, nil, http.StatusOK, http.StatusNotImplemented)
	if err != nil {
		exitf("GetActivities failed: %v", err)
	}
	if status == http.StatusNotImplemented {
		notImplemented(getActivitiesPath)
		return
	}

	reconnectPath := "/azure/directline/v3/directline/conversations/" + conversationID
	_, status, err = client.doJSON(ctx, http.MethodGet, reconnectPath, nil, http.StatusOK, http.StatusNotImplemented)
	if err != nil {
		exitf("ReconnectToConversation failed: %v", err)
	}
	if status == http.StatusNotImplemented {
		notImplemented(reconnectPath)
		return
	}

	if token != "" {
		const refreshTokenPath = "/azure/directline/v3/directline/tokens/refresh"
		_, status, err = client.doJSON(ctx, http.MethodPost, refreshTokenPath, map[string]any{"token": token}, http.StatusOK, http.StatusNotImplemented)
		if err != nil {
			exitf("RefreshToken failed: %v", err)
		}
		if status == http.StatusNotImplemented {
			notImplemented(refreshTokenPath)
			return
		}
	}

	const getSessionPath = "/azure/directline/v3/directline/session/getsessionid"
	_, status, err = client.doJSON(ctx, http.MethodGet, getSessionPath, nil, http.StatusOK, http.StatusNotImplemented)
	if err != nil {
		exitf("GetSessionId failed: %v", err)
	}
	if status == http.StatusNotImplemented {
		notImplemented(getSessionPath)
		return
	}

	if activityID != "" {
		fmt.Printf("Done. conversationId=%s activityId=%s\n", conversationID, activityID)
		return
	}
	fmt.Printf("Done. conversationId=%s\n", conversationID)
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
