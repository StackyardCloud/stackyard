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

const communicationEmailAPIVersion = "2025-09-01"

type communicationEmailClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type communicationEmailSharedKeyPolicy struct {
	account         string
	subscriptionKey string
}

func (p *communicationEmailSharedKeyPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newCommunicationEmailClient(endpoint, account, subscriptionKey string) *communicationEmailClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"communication-email-2025-09-01",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&communicationEmailSharedKeyPolicy{
				account:         account,
				subscriptionKey: subscriptionKey,
			}},
		},
		&policy.ClientOptions{},
	)
	return &communicationEmailClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *communicationEmailClient) doJSON(ctx context.Context, method, path string, payload any, headers map[string]string, expectedStatuses ...int) (map[string]any, int, http.Header, error) {
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

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	account := getenv("STACKYARD_AZURE_COMMUNICATION_EMAIL_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_COMMUNICATION_EMAIL_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")

	fmt.Printf("Stackyard Azure Communication Email example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newCommunicationEmailClient(endpoint, account, subscriptionKey)
	base := "/azure/communication-email/2025-09-01"
	operationID := "2a5f4d10-9b4d-4a70-98ef-7d93b42c1b70"

	sendPayload, status, sendHeaders, err := client.doJSON(ctx, http.MethodPost, base+"/emails:send?api-version="+communicationEmailAPIVersion, map[string]any{
		"senderAddress": "DoNotReply@contoso.com",
		"content": map[string]any{
			"subject":   "Stackyard hello",
			"plainText": "hello from stackyard",
			"html":      "<html><body><p>hello from stackyard</p></body></html>",
		},
		"recipients": map[string]any{
			"to": []map[string]any{
				{
					"address":     "alice@example.com",
					"displayName": "Alice",
				},
			},
			"cc": []map[string]any{
				{
					"address": "cc@example.com",
				},
			},
		},
		"attachments": []map[string]any{
			{
				"name":            "note.txt",
				"contentType":     "text/plain",
				"contentInBase64": "aGVsbG8=",
			},
		},
	}, map[string]string{
		"Operation-Id": operationID,
	}, http.StatusAccepted)
	if err != nil {
		fatalf("SendEmail failed: %v", err)
	}
	printJSON("SendEmail", status, sendPayload)

	operationLocation := sendHeaders.Get("Operation-Location")
	if strings.TrimSpace(operationLocation) == "" {
		operationLocation = base + "/emails/operations/" + operationID + "?api-version=" + communicationEmailAPIVersion
	}
	fmt.Printf("\nOperation-Location: %s\n", operationLocation)

	for attempt := 1; attempt <= 3; attempt++ {
		pollPayload, pollStatus, _, err := client.doJSON(ctx, http.MethodGet, operationLocation, nil, nil, http.StatusOK)
		if err != nil {
			fatalf("GetSendResult failed: %v", err)
		}
		printJSON(fmt.Sprintf("GetSendResult attempt %d", attempt), pollStatus, pollPayload)
		if strings.TrimSpace(stringValue(pollPayload["status"])) != "Running" {
			break
		}
	}
}

func ensureStatus(status int, body []byte, expectedStatuses ...int) error {
	for _, expected := range expectedStatuses {
		if status == expected {
			return nil
		}
	}
	return fmt.Errorf("unexpected status %d body=%s", status, strings.TrimSpace(string(body)))
}

func stringValue(value any) string {
	if s, ok := value.(string); ok {
		return s
	}
	return ""
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
