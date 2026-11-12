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

const migrateProjectsAPIVersion = "2018-09-01-preview"

type migrateProjectsClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type migrateProjectsSharedKeyPolicy struct {
	account         string
	subscriptionKey string
}

func (p *migrateProjectsSharedKeyPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newMigrateProjectsClient(endpoint, account, subscriptionKey string) *migrateProjectsClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"migrate-projects-2018-09-01-preview",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&migrateProjectsSharedKeyPolicy{account: account, subscriptionKey: subscriptionKey}},
		},
		&policy.ClientOptions{},
	)
	return &migrateProjectsClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *migrateProjectsClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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
	if len(expectedStatuses) > 0 {
		ok := false
		for _, status := range expectedStatuses {
			if resp.StatusCode == status {
				ok = true
				break
			}
		}
		if !ok {
			return nil, resp.StatusCode, fmt.Errorf("unexpected status %d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
		}
	} else if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, resp.StatusCode, fmt.Errorf("unexpected status %d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	if strings.TrimSpace(string(body)) == "" {
		return map[string]any{}, resp.StatusCode, nil
	}
	var decoded map[string]any
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, resp.StatusCode, fmt.Errorf("decode response %s %s: %w", method, path, err)
	}
	return decoded, resp.StatusCode, nil
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	account := getenv("STACKYARD_AZURE_MIGRATE_PROJECTS_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_MIGRATE_PROJECTS_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	resourceGroup := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-migrate-projects")
	projectName := getenv("STACKYARD_AZURE_MIGRATE_PROJECTS_NAME", "project-a")

	fmt.Printf("Stackyard Azure Migrate Projects example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newMigrateProjectsClient(endpoint, account, subscriptionKey)
	base := "/azure/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Migrate/migrateProjects/" + projectName

	calls := []struct {
		name     string
		method   string
		path     string
		payload  any
		statuses []int
	}{
		{
			name:     "ListOperations",
			method:   http.MethodGet,
			path:     "/azure/providers/Microsoft.Migrate/operations?api-version=" + migrateProjectsAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateProject",
			method: http.MethodPut,
			path:   base + "?api-version=" + migrateProjectsAPIVersion,
			payload: map[string]any{
				"location": "eastus",
				"tags":     map[string]any{"env": "dev"},
			},
			statuses: []int{http.StatusCreated, http.StatusOK},
		},
		{
			name:   "RegisterTool",
			method: http.MethodPost,
			path:   base + "/registerTool?api-version=" + migrateProjectsAPIVersion,
			payload: map[string]any{
				"tool": "ServerDiscovery",
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:   "RefreshSummary",
			method: http.MethodPost,
			path:   base + "/refreshSummary?api-version=" + migrateProjectsAPIVersion,
			payload: map[string]any{
				"goal": "Servers",
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GetProject",
			method:   http.MethodGet,
			path:     base + "?api-version=" + migrateProjectsAPIVersion,
			statuses: []int{http.StatusOK},
		},
	}

	for _, call := range calls {
		payload, status, err := client.doJSON(ctx, call.method, call.path, call.payload, call.statuses...)
		if err != nil {
			fatalf("%s failed: %v", call.name, err)
		}
		printJSON(call.name, status, payload)
	}
}

func printJSON(name string, status int, payload map[string]any) {
	formatted, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		fatalf("%s: marshal response: %v", name, err)
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
