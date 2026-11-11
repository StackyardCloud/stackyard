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

const callAutomationAPIVersion = "2026-03-12"

type callAutomationClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type callAutomationSharedKeyPolicy struct {
	account         string
	subscriptionKey string
}

func (p *callAutomationSharedKeyPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newCallAutomationClient(endpoint, account, subscriptionKey string) *callAutomationClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"communication-callautomation-2026-03-12",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&callAutomationSharedKeyPolicy{account: account, subscriptionKey: subscriptionKey}},
		},
		&policy.ClientOptions{},
	)
	return &callAutomationClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *callAutomationClient) doJSON(ctx context.Context, method, path string, payload any, headers map[string]string, expectedStatuses ...int) (map[string]any, int, error) {
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
	account := getenv("STACKYARD_AZURE_COMMUNICATION_CALLAUTOMATION_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_COMMUNICATION_CALLAUTOMATION_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")

	fmt.Printf("Stackyard Azure Communication Call Automation example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newCallAutomationClient(endpoint, account, subscriptionKey)
	base := "/azure/communication-callautomation/2026-03-12/calling"

	createPayload, status, err := client.doJSON(ctx, http.MethodPost, base+"/callConnections?api-version="+callAutomationAPIVersion, map[string]any{
		"source": map[string]any{
			"rawId": "8:acs:source-a",
		},
		"targets": []map[string]any{
			{"rawId": "8:acs:target-a"},
			{"phoneNumber": map[string]any{"phoneNumber": "+15551230001"}},
		},
		"callbackUri":      "https://example.test/callbacks/outbound-a",
		"operationContext": "example-create-call",
	}, map[string]string{
		"Repeatability-Request-ID": "6f9619ff-8b86-d011-b42d-00cf4fc964ff",
		"Repeatability-First-Sent": "Sun, 12 Apr 2026 09:00:00 GMT",
	}, http.StatusCreated)
	if err != nil {
		fatalf("CreateCall failed: %v", err)
	}
	printJSON("CreateCall", status, createPayload)

	callID := fmt.Sprintf("%v", createPayload["callConnectionId"])

	callPayload, status, err := client.doJSON(ctx, http.MethodGet, base+"/callConnections/"+callID+"?api-version="+callAutomationAPIVersion, nil, nil, http.StatusOK)
	if err != nil {
		fatalf("GetCall failed: %v", err)
	}
	printJSON("GetCall", status, callPayload)

	addParticipantPayload, status, err := client.doJSON(ctx, http.MethodPost, base+"/callConnections/"+callID+"/participants:add?api-version="+callAutomationAPIVersion, map[string]any{
		"participant": map[string]any{
			"communicationUser": map[string]any{
				"id": "8:acs:participant-b",
			},
		},
	}, nil, http.StatusAccepted)
	if err != nil {
		fatalf("AddParticipant failed: %v", err)
	}
	printJSON("AddParticipant", status, addParticipantPayload)

	playPayload, status, err := client.doJSON(ctx, http.MethodPost, base+"/callConnections/"+callID+":play?api-version="+callAutomationAPIVersion, map[string]any{
		"playSource": map[string]any{
			"kind": "text",
			"text": "Hello from Stackyard",
		},
	}, nil, http.StatusAccepted)
	if err != nil {
		fatalf("Play failed: %v", err)
	}
	printJSON("Play", status, playPayload)

	recordingPayload, status, err := client.doJSON(ctx, http.MethodPost, base+"/recordings?api-version="+callAutomationAPIVersion, map[string]any{
		"callConnectionId":     callID,
		"recordingChannelType": "mixed",
	}, nil, http.StatusOK)
	if err != nil {
		fatalf("StartRecording failed: %v", err)
	}
	printJSON("StartRecording", status, recordingPayload)

	recordingID := fmt.Sprintf("%v", recordingPayload["recordingId"])
	getRecordingPayload, status, err := client.doJSON(ctx, http.MethodGet, base+"/recordings/"+recordingID+"?api-version="+callAutomationAPIVersion, nil, nil, http.StatusOK)
	if err != nil {
		fatalf("GetRecording failed: %v", err)
	}
	printJSON("GetRecording", status, getRecordingPayload)
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
