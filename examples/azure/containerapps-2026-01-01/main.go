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

const containerAppsAPIVersion = "2026-01-01"

type containerAppsClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type containerAppsSharedKeyPolicy struct {
	account         string
	subscriptionKey string
}

func (p *containerAppsSharedKeyPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newContainerAppsClient(endpoint, account, subscriptionKey string) *containerAppsClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"containerapps-2026-01-01",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&containerAppsSharedKeyPolicy{account: account, subscriptionKey: subscriptionKey}},
		},
		&policy.ClientOptions{},
	)
	return &containerAppsClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *containerAppsClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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
	if decoded == nil {
		decoded = map[string]any{}
	}
	return decoded, resp.StatusCode, nil
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	account := getenv("STACKYARD_AZURE_CONTAINER_APPS_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_CONTAINER_APPS_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	resourceGroup := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-containerapps")
	region := getenv("STACKYARD_AZURE_REGION", "eastus")
	environmentName := getenv("STACKYARD_AZURE_CONTAINER_APPS_ENVIRONMENT", "env-a")
	appName := getenv("STACKYARD_AZURE_CONTAINER_APP_NAME", "app-a")
	jobName := getenv("STACKYARD_AZURE_CONTAINER_APPS_JOB_NAME", "job-a")

	fmt.Printf("Stackyard Azure Container Apps example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newContainerAppsClient(endpoint, account, subscriptionKey)
	subscriptionBase := "/azure/subscriptions/" + subscriptionID
	environmentID := "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.App/managedEnvironments/" + environmentName
	environmentBase := subscriptionBase + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.App/managedEnvironments/" + environmentName
	appBase := subscriptionBase + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.App/containerApps/" + appName
	jobBase := subscriptionBase + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.App/jobs/" + jobName

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
			path:     "/azure/providers/Microsoft.App/operations?api-version=" + containerAppsAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListWorkloadProfiles",
			method:   http.MethodGet,
			path:     subscriptionBase + "/locations/" + region + "/providers/Microsoft.App/availableManagedEnvironmentsWorkloadProfileTypes?api-version=" + containerAppsAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateManagedEnvironment",
			method: http.MethodPut,
			path:   environmentBase + "?api-version=" + containerAppsAPIVersion,
			payload: map[string]any{
				"location": region,
				"tags":     map[string]any{"env": "dev"},
				"properties": map[string]any{
					"workloadProfiles": []any{
						map[string]any{"name": "Consumption", "workloadProfileType": "Consumption"},
					},
				},
			},
			statuses: []int{http.StatusCreated, http.StatusOK},
		},
		{
			name:   "CreateContainerApp",
			method: http.MethodPut,
			path:   appBase + "?api-version=" + containerAppsAPIVersion,
			payload: map[string]any{
				"location": region,
				"tags":     map[string]any{"component": "web"},
				"properties": map[string]any{
					"environmentId": environmentID,
					"configuration": map[string]any{
						"ingress": map[string]any{"external": true, "targetPort": 8080},
					},
					"template": map[string]any{
						"containers": []any{
							map[string]any{"name": "web", "image": "nginx:latest"},
						},
					},
				},
			},
			statuses: []int{http.StatusCreated, http.StatusOK},
		},
		{
			name:     "GetContainerApp",
			method:   http.MethodGet,
			path:     appBase + "?api-version=" + containerAppsAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListRevisions",
			method:   http.MethodGet,
			path:     appBase + "/revisions?api-version=" + containerAppsAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateJob",
			method: http.MethodPut,
			path:   jobBase + "?api-version=" + containerAppsAPIVersion,
			payload: map[string]any{
				"location": region,
				"properties": map[string]any{
					"environmentId": environmentID,
					"configuration": map[string]any{"triggerType": "Manual"},
					"template": map[string]any{
						"containers": []any{
							map[string]any{"name": "job", "image": "busybox"},
						},
					},
				},
			},
			statuses: []int{http.StatusCreated, http.StatusOK},
		},
		{
			name:     "ListJobExecutions",
			method:   http.MethodGet,
			path:     jobBase + "/executions?api-version=" + containerAppsAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "PatchContainerApp",
			method: http.MethodPatch,
			path:   appBase + "?api-version=" + containerAppsAPIVersion,
			payload: map[string]any{
				"tags":       map[string]any{"component": "web", "owner": "stackyard"},
				"properties": map[string]any{"configuration": map[string]any{"maxInactiveRevisions": 5}},
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:     "DeleteJob",
			method:   http.MethodDelete,
			path:     jobBase + "?api-version=" + containerAppsAPIVersion,
			statuses: []int{http.StatusOK, http.StatusNoContent},
		},
		{
			name:     "DeleteContainerApp",
			method:   http.MethodDelete,
			path:     appBase + "?api-version=" + containerAppsAPIVersion,
			statuses: []int{http.StatusOK, http.StatusNoContent},
		},
		{
			name:     "DeleteManagedEnvironment",
			method:   http.MethodDelete,
			path:     environmentBase + "?api-version=" + containerAppsAPIVersion,
			statuses: []int{http.StatusOK, http.StatusNoContent},
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
