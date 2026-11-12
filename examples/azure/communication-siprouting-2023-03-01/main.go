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

const communicationSipRoutingAPIVersion = "2023-03-01"

type communicationSipRoutingClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type communicationSipRoutingSharedKeyPolicy struct {
	account         string
	subscriptionKey string
}

func (p *communicationSipRoutingSharedKeyPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newCommunicationSipRoutingClient(endpoint, account, subscriptionKey string) *communicationSipRoutingClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"communication-siprouting-2023-03-01",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&communicationSipRoutingSharedKeyPolicy{
				account:         account,
				subscriptionKey: subscriptionKey,
			}},
		},
		&policy.ClientOptions{},
	)

	return &communicationSipRoutingClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *communicationSipRoutingClient) doJSON(ctx context.Context, method, path, contentType string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
	req, err := runtime.NewRequest(ctx, method, c.endpoint+path)
	if err != nil {
		return nil, 0, fmt.Errorf("create request %s %s: %w", method, path, err)
	}
	req.Raw().Header.Set("Accept", "application/json")
	if payload != nil {
		if strings.TrimSpace(contentType) != "" {
			req.Raw().Header.Set("Content-Type", contentType)
		}
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

	var decoded map[string]any
	if len(body) != 0 {
		if err := json.Unmarshal(body, &decoded); err != nil {
			return nil, resp.StatusCode, fmt.Errorf("decode response %s %s: %w", method, path, err)
		}
	}
	if decoded == nil {
		decoded = map[string]any{}
	}
	return decoded, resp.StatusCode, nil
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	account := getenv("STACKYARD_AZURE_COMMUNICATION_SIPROUTING_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_COMMUNICATION_SIPROUTING_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")

	fmt.Printf("Stackyard Azure Communication Sip Routing example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newCommunicationSipRoutingClient(endpoint, account, subscriptionKey)
	path := "/azure/communication-siprouting/2023-03-01/sip?api-version=" + communicationSipRoutingAPIVersion

	initial, status, err := client.doJSON(ctx, http.MethodGet, path, "", nil, http.StatusOK)
	if err != nil {
		fatalf("Get sip configuration failed: %v", err)
	}
	printJSON("Get sip configuration", status, initial)

	updated, status, err := client.doJSON(ctx, http.MethodPatch, path, "application/merge-patch+json", map[string]any{
		"trunks": map[string]any{
			"pstn.contoso.com":   map[string]any{"sipSignalingPort": 5067},
			"backup.contoso.com": map[string]any{"sipSignalingPort": 5070},
		},
		"routes": []any{
			map[string]any{
				"name":          "contoso-primary",
				"description":   "Primary PSTN route",
				"numberPattern": "^\\+1206\\d{7}$",
				"trunks":        []any{"pstn.contoso.com"},
			},
			map[string]any{
				"name":          "contoso-fallback",
				"numberPattern": "^\\+1\\d+$",
				"trunks":        []any{"pstn.contoso.com", "backup.contoso.com"},
			},
		},
	}, http.StatusOK)
	if err != nil {
		fatalf("Patch sip configuration failed: %v", err)
	}
	printJSON("Patch sip configuration", status, updated)

	finalPayload, status, err := client.doJSON(ctx, http.MethodGet, path, "", nil, http.StatusOK)
	if err != nil {
		fatalf("Get patched sip configuration failed: %v", err)
	}
	printJSON("Get patched sip configuration", status, finalPayload)
}

func ensureStatus(status int, body []byte, expectedStatuses ...int) error {
	for _, expected := range expectedStatuses {
		if status == expected {
			return nil
		}
	}
	return fmt.Errorf("unexpected status %d body=%s", status, strings.TrimSpace(string(body)))
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
