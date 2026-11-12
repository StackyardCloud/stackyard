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

const advancedMessagingAPIVersion = "2024-08-30"

type advancedMessagingClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type advancedMessagingSharedKeyPolicy struct {
	account         string
	subscriptionKey string
}

func (p *advancedMessagingSharedKeyPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newAdvancedMessagingClient(endpoint, account, subscriptionKey string) *advancedMessagingClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"communication-advancedmessaging-2024-08-30",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&advancedMessagingSharedKeyPolicy{account: account, subscriptionKey: subscriptionKey}},
		},
		&policy.ClientOptions{},
	)
	return &advancedMessagingClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *advancedMessagingClient) doJSON(ctx context.Context, method, path string, payload any, headers map[string]string, expectedStatuses ...int) (map[string]any, int, error) {
	req, err := runtime.NewRequest(ctx, method, c.endpoint+path)
	if err != nil {
		return nil, 0, fmt.Errorf("create request %s %s: %w", method, path, err)
	}
	req.Raw().Header.Set("Accept", "application/json")
	for key, value := range headers {
		req.Raw().Header.Set(key, value)
	}
	if payload != nil {
		req.Raw().Header.Set("Content-Type", "application/json")
		if err := runtime.MarshalAsJSON(req, payload); err != nil {
			return nil, 0, fmt.Errorf("marshal payload %s %s: %w", method, path, err)
		}
	}

	resp, err := c.pipeline.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("execute request %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("read response %s %s: %w", method, path, err)
	}
	if err := ensureStatus(resp.StatusCode, body, expectedStatuses...); err != nil {
		return nil, resp.StatusCode, err
	}
	if strings.TrimSpace(string(body)) == "" {
		return map[string]any{}, resp.StatusCode, nil
	}

	var decoded map[string]any
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, resp.StatusCode, fmt.Errorf("decode response %s %s: %w", method, path, err)
	}
	if decoded == nil {
		decoded = map[string]any{}
	}
	return decoded, resp.StatusCode, nil
}

func (c *advancedMessagingClient) doBytes(ctx context.Context, method, path string, expectedStatuses ...int) ([]byte, string, int, error) {
	req, err := runtime.NewRequest(ctx, method, c.endpoint+path)
	if err != nil {
		return nil, "", 0, fmt.Errorf("create request %s %s: %w", method, path, err)
	}
	resp, err := c.pipeline.Do(req)
	if err != nil {
		return nil, "", 0, fmt.Errorf("execute request %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.Header.Get("Content-Type"), resp.StatusCode, fmt.Errorf("read response %s %s: %w", method, path, err)
	}
	if err := ensureStatus(resp.StatusCode, body, expectedStatuses...); err != nil {
		return nil, resp.Header.Get("Content-Type"), resp.StatusCode, err
	}
	return body, resp.Header.Get("Content-Type"), resp.StatusCode, nil
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
	account := getenv("STACKYARD_AZURE_COMMUNICATION_ADVANCEDMESSAGING_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_COMMUNICATION_ADVANCEDMESSAGING_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	channelID := getenv("STACKYARD_AZURE_COMMUNICATION_ADVANCEDMESSAGING_CHANNEL_ID", "11111111-1111-4111-8111-111111111111")

	fmt.Printf("Stackyard Azure Communication Advanced Messaging example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newAdvancedMessagingClient(endpoint, account, subscriptionKey)
	base := "/azure/communication-advancedmessaging/2024-08-30/messages"

	textPayload, status, err := client.doJSON(ctx, http.MethodPost, base+"/notifications:send?api-version="+advancedMessagingAPIVersion, map[string]any{
		"kind":                  "text",
		"channelRegistrationId": "channel-registration-a",
		"to":                    []string{"+15551230001"},
		"content":               "hello from stackyard",
	}, map[string]string{
		"Repeatability-Request-ID": "6f9619ff-8b86-d011-b42d-00cf4fc964ff",
		"Repeatability-First-Sent": "Mon, 12 Apr 2026 09:00:00 GMT",
	}, http.StatusAccepted)
	if err != nil {
		fatalf("SendText failed: %v", err)
	}
	printJSON("SendText", status, textPayload)

	templatePayload, status, err := client.doJSON(ctx, http.MethodPost, base+"/notifications:send?api-version="+advancedMessagingAPIVersion, map[string]any{
		"kind":                  "template",
		"channelRegistrationId": "channel-registration-a",
		"to":                    []string{"+15551230002"},
		"template": map[string]any{
			"name": "order_update",
			"language": map[string]any{
				"code": "en_US",
			},
			"values": map[string]any{
				"1": "order-123",
			},
		},
	}, nil, http.StatusAccepted)
	if err != nil {
		fatalf("SendTemplate failed: %v", err)
	}
	printJSON("SendTemplate", status, templatePayload)

	templates, status, err := client.doJSON(ctx, http.MethodGet, base+"/channels/"+channelID+"/templates?api-version="+advancedMessagingAPIVersion+"&maxpagesize=10", nil, nil, http.StatusOK)
	if err != nil {
		fatalf("ListTemplates failed: %v", err)
	}
	printJSON("ListTemplates", status, templates)

	streamBody, contentType, status, err := client.doBytes(ctx, http.MethodGet, base+"/streams/stream-a?api-version="+advancedMessagingAPIVersion, http.StatusOK)
	if err != nil {
		fatalf("GetMedia failed: %v", err)
	}
	fmt.Printf("\n== GetMedia (%d) ==\ncontent-type: %s\nbody: %s\n", status, contentType, string(streamBody))
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
