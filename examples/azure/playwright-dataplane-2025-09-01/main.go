package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
)

const playwrightDataPlaneAPIVersion = "2025-09-01"

type playwrightDataPlaneClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type playwrightDataPlaneSharedKeyPolicy struct {
	account         string
	subscriptionKey string
}

func (p *playwrightDataPlaneSharedKeyPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newPlaywrightDataPlaneClient(endpoint, account, subscriptionKey string) *playwrightDataPlaneClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"playwright-dataplane-2025-09-01",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&playwrightDataPlaneSharedKeyPolicy{account: account, subscriptionKey: subscriptionKey}},
		},
		&policy.ClientOptions{},
	)
	return &playwrightDataPlaneClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *playwrightDataPlaneClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, resp.StatusCode, fmt.Errorf("read response %s %s: %w", method, path, readErr)
	}
	if err := ensurePlaywrightDataPlaneStatus(resp.StatusCode, body, expectedStatuses...); err != nil {
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

func ensurePlaywrightDataPlaneStatus(status int, body []byte, expectedStatuses ...int) error {
	if len(expectedStatuses) > 0 {
		for _, expected := range expectedStatuses {
			if status == expected {
				return nil
			}
		}
		return fmt.Errorf("unexpected status %d body=%s", status, strings.TrimSpace(string(body)))
	}
	if status < 200 || status >= 300 {
		return fmt.Errorf("unexpected status %d body=%s", status, strings.TrimSpace(string(body)))
	}
	return nil
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	account := getenv("STACKYARD_AZURE_PLAYWRIGHT_DATA_PLANE_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_PLAYWRIGHT_DATA_PLANE_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	workspaceID := getenv("STACKYARD_AZURE_PLAYWRIGHT_DATA_PLANE_WORKSPACE_ID", "11111111-1111-1111-1111-111111111111")
	accessTokenID := getenv("STACKYARD_AZURE_PLAYWRIGHT_DATA_PLANE_ACCESS_TOKEN_ID", "token-abc")
	runID := getenv("STACKYARD_AZURE_PLAYWRIGHT_DATA_PLANE_RUN_ID", "22222222-2222-2222-2222-222222222222")

	fmt.Printf("Stackyard Azure Playwright Data Plane example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newPlaywrightDataPlaneClient(endpoint, account, subscriptionKey)
	base := "/azure/playwright-data-plane/2025-09-01/playwrightworkspaces/" + workspaceID
	expiryAt := time.Now().UTC().Add(24 * time.Hour).Format(time.RFC3339)

	calls := []struct {
		name     string
		method   string
		path     string
		payload  any
		statuses []int
	}{
		{
			name:   "CreateAccessToken",
			method: http.MethodPut,
			path:   base + "/access-tokens/" + accessTokenID + "?api-version=" + playwrightDataPlaneAPIVersion,
			payload: map[string]any{
				"name":     "token-a",
				"expiryAt": expiryAt,
			},
			statuses: []int{http.StatusCreated, http.StatusOK},
		},
		{
			name:     "ListAccessTokens",
			method:   http.MethodGet,
			path:     base + "/access-tokens?api-version=" + playwrightDataPlaneAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GetAccessToken",
			method:   http.MethodGet,
			path:     base + "/access-tokens/" + accessTokenID + "?api-version=" + playwrightDataPlaneAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateTestRun",
			method: http.MethodPatch,
			path:   base + "/test-runs/" + runID + "?api-version=" + playwrightDataPlaneAPIVersion,
			payload: map[string]any{
				"displayName": "Stackyard Run",
				"config": map[string]any{
					"sdkLanguage": "TYPESCRIPT",
					"maxWorkers":  2,
				},
				"ciConfig": map[string]any{
					"providerName": "GitHubActions",
				},
			},
			statuses: []int{http.StatusCreated, http.StatusOK},
		},
		{
			name:     "ListTestRuns",
			method:   http.MethodGet,
			path:     base + "/test-runs?api-version=" + playwrightDataPlaneAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "DeleteAccessToken",
			method:   http.MethodDelete,
			path:     base + "/access-tokens/" + accessTokenID + "?api-version=" + playwrightDataPlaneAPIVersion,
			statuses: []int{http.StatusNoContent},
		},
	}

	for _, call := range calls {
		payload, status, err := client.doJSON(ctx, call.method, call.path, call.payload, call.statuses...)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s failed: %v\n", call.name, err)
			os.Exit(1)
		}
		encoded, _ := json.MarshalIndent(payload, "", "  ")
		if len(encoded) == 0 || string(encoded) == "null" {
			encoded = []byte("{}")
		}
		fmt.Printf("%s -> %d\n%s\n", call.name, status, string(encoded))
	}
}

func getenv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
