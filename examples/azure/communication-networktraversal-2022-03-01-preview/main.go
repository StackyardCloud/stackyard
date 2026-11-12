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

const communicationNetworkTraversalAPIVersion = "2022-03-01-preview"

type communicationNetworkTraversalClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type communicationNetworkTraversalSharedKeyPolicy struct {
	account         string
	subscriptionKey string
}

func (p *communicationNetworkTraversalSharedKeyPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newCommunicationNetworkTraversalClient(endpoint, account, subscriptionKey string) *communicationNetworkTraversalClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"communication-networktraversal-2022-03-01-preview",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&communicationNetworkTraversalSharedKeyPolicy{
				account:         account,
				subscriptionKey: subscriptionKey,
			}},
		},
		&policy.ClientOptions{},
	)
	return &communicationNetworkTraversalClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *communicationNetworkTraversalClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
	req, err := runtime.NewRequest(ctx, method, c.endpoint+path)
	if err != nil {
		return nil, 0, fmt.Errorf("create request %s %s: %w", method, path, err)
	}
	req.Raw().Header.Set("Accept", "application/json")
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
	account := getenv("STACKYARD_AZURE_COMMUNICATION_NETWORKTRAVERSAL_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_COMMUNICATION_NETWORKTRAVERSAL_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")

	fmt.Printf("Stackyard Azure Communication Network Traversal example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newCommunicationNetworkTraversalClient(endpoint, account, subscriptionKey)
	path := "/azure/communication-networktraversal/2022-03-01-preview/networkTraversal/:issueRelayConfiguration?api-version=" + communicationNetworkTraversalAPIVersion

	payload, status, err := client.doJSON(ctx, http.MethodPost, path, map[string]any{
		"id":        "8:acs:000000000000000000000001",
		"routeType": "nearest",
		"ttl":       3600,
	}, http.StatusOK)
	if err != nil {
		fatalf("IssueRelayConfiguration failed: %v", err)
	}

	printJSON("IssueRelayConfiguration", status, payload)
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
