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

const communicationChatAPIVersion = "2025-03-15"

type communicationChatClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type communicationChatSharedKeyPolicy struct {
	account         string
	subscriptionKey string
}

func (p *communicationChatSharedKeyPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newCommunicationChatClient(endpoint, account, subscriptionKey string) *communicationChatClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"communication-chat-2025-03-15",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&communicationChatSharedKeyPolicy{
				account:         account,
				subscriptionKey: subscriptionKey,
			}},
		},
		&policy.ClientOptions{},
	)
	return &communicationChatClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *communicationChatClient) doJSON(ctx context.Context, method, path string, payload any, headers map[string]string, expectedStatuses ...int) (map[string]any, int, http.Header, error) {
	req, err := runtime.NewRequest(ctx, method, c.endpoint+path)
	if err != nil {
		return nil, 0, nil, fmt.Errorf("create request %s %s: %w", method, path, err)
	}
	req.Raw().Header.Set("Accept", "application/json")
	for key, value := range headers {
		req.Raw().Header.Set(key, value)
	}
	if payload != nil {
		req.Raw().Header.Set("Content-Type", "application/json")
		if err := runtime.MarshalAsJSON(req, payload); err != nil {
			return nil, 0, nil, fmt.Errorf("marshal payload %s %s: %w", method, path, err)
		}
	}

	resp, err := c.pipeline.Do(req)
	if err != nil {
		return nil, 0, nil, fmt.Errorf("execute request %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, resp.Header, fmt.Errorf("read response %s %s: %w", method, path, err)
	}
	if err := ensureStatus(resp.StatusCode, body, expectedStatuses...); err != nil {
		return nil, resp.StatusCode, resp.Header, err
	}
	if strings.TrimSpace(string(body)) == "" {
		return map[string]any{}, resp.StatusCode, resp.Header, nil
	}

	var decoded map[string]any
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, resp.StatusCode, resp.Header, fmt.Errorf("decode response %s %s: %w", method, path, err)
	}
	if decoded == nil {
		decoded = map[string]any{}
	}
	return decoded, resp.StatusCode, resp.Header, nil
}

func ensureStatus(status int, body []byte, expectedStatuses ...int) error {
	for _, expected := range expectedStatuses {
		if status == expected {
			return nil
		}
	}
	return fmt.Errorf("unexpected status %d body=%s", status, strings.TrimSpace(string(body)))
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	account := getenv("STACKYARD_AZURE_COMMUNICATION_CHAT_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_COMMUNICATION_CHAT_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")

	fmt.Printf("Stackyard Azure Communication Chat example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newCommunicationChatClient(endpoint, account, subscriptionKey)
	base := "/azure/communication-chat/2025-03-15/chat/threads"

	createPayload, status, _, err := client.doJSON(ctx, http.MethodPost, base+"?api-version="+communicationChatAPIVersion, map[string]any{
		"topic": "stackyard-thread",
		"metadata": map[string]any{
			"env": "example",
		},
		"participants": []map[string]any{
			{
				"communicationIdentifier": map[string]any{
					"rawId": "8:acs:user-a",
				},
				"displayName": "User A",
			},
		},
	}, map[string]string{
		"Repeatability-Request-ID": "6f9619ff-8b86-d011-b42d-00cf4fc964ff",
		"Repeatability-First-Sent": "Sun, 12 Apr 2026 09:00:00 GMT",
	}, http.StatusCreated)
	if err != nil {
		fatalf("CreateChatThread failed: %v", err)
	}
	printJSON("CreateChatThread", status, createPayload)

	thread := createPayload["chatThread"].(map[string]any)
	threadID := fmt.Sprintf("%v", thread["id"])

	listThreads, status, _, err := client.doJSON(ctx, http.MethodGet, base+"?api-version="+communicationChatAPIVersion, nil, nil, http.StatusOK)
	if err != nil {
		fatalf("ListChatThreads failed: %v", err)
	}
	printJSON("ListChatThreads", status, listThreads)

	addParticipants, status, _, err := client.doJSON(ctx, http.MethodPost, base+"/"+threadID+"/participants/:add?api-version="+communicationChatAPIVersion, map[string]any{
		"participants": []map[string]any{
			{
				"communicationIdentifier": map[string]any{
					"rawId": "8:acs:user-b",
				},
				"displayName": "User B",
			},
		},
	}, nil, http.StatusCreated)
	if err != nil {
		fatalf("AddChatParticipants failed: %v", err)
	}
	printJSON("AddChatParticipants", status, addParticipants)

	sendMessage, status, sendHeaders, err := client.doJSON(ctx, http.MethodPost, base+"/"+threadID+"/messages?api-version="+communicationChatAPIVersion, map[string]any{
		"content":           "hello from stackyard",
		"type":              "text",
		"senderDisplayName": "Stackyard User",
		"metadata": map[string]any{
			"kind": "greeting",
		},
	}, nil, http.StatusCreated)
	if err != nil {
		fatalf("SendChatMessage failed: %v", err)
	}
	fmt.Printf("\nSendChatMessage Location: %s\n", sendHeaders.Get("Location"))
	printJSON("SendChatMessage", status, sendMessage)

	messageID := fmt.Sprintf("%v", sendMessage["id"])

	getMessage, status, _, err := client.doJSON(ctx, http.MethodGet, base+"/"+threadID+"/messages/"+messageID+"?api-version="+communicationChatAPIVersion, nil, nil, http.StatusOK)
	if err != nil {
		fatalf("GetChatMessage failed: %v", err)
	}
	printJSON("GetChatMessage", status, getMessage)

	_, status, _, err = client.doJSON(ctx, http.MethodPost, base+"/"+threadID+"/readReceipts?api-version="+communicationChatAPIVersion, map[string]any{
		"chatMessageId": messageID,
	}, nil, http.StatusOK)
	if err != nil {
		fatalf("SendChatReadReceipt failed: %v", err)
	}
	fmt.Printf("\n== SendChatReadReceipt (%d) ==\n{}\n", status)

	readReceipts, status, _, err := client.doJSON(ctx, http.MethodGet, base+"/"+threadID+"/readReceipts?api-version="+communicationChatAPIVersion, nil, nil, http.StatusOK)
	if err != nil {
		fatalf("ListChatReadReceipts failed: %v", err)
	}
	printJSON("ListChatReadReceipts", status, readReceipts)
}

func printJSON(name string, status int, payload map[string]any) {
	formatted, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		fatalf("format %s response: %v", name, err)
	}
	fmt.Printf("\n== %s (%d) ==\n%s\n", name, status, string(formatted))
}

func getenv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
