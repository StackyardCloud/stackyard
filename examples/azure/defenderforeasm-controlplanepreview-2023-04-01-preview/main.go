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

const defenderForEASMControlPlanePreviewAPIVersion = "2023-04-01-preview"

type defenderForEASMClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type defenderForEASMSharedKeyPolicy struct {
	account         string
	subscriptionKey string
}

func (p *defenderForEASMSharedKeyPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newDefenderForEASMClient(endpoint, account, subscriptionKey string) *defenderForEASMClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"defenderforeasm-controlplanepreview-2023-04-01-preview",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{
				&defenderForEASMSharedKeyPolicy{
					account:         account,
					subscriptionKey: subscriptionKey,
				},
			},
		},
		&policy.ClientOptions{},
	)

	return &defenderForEASMClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *defenderForEASMClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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
	account := getenv("STACKYARD_AZURE_DEFENDERFOREASM_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_DEFENDERFOREASM_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	resourceGroup := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-defenderforeasm")
	location := getenv("STACKYARD_AZURE_LOCATION", "westus")
	workspaceName := getenv("STACKYARD_AZURE_DEFENDERFOREASM_WORKSPACE_NAME", "ThisisaWorkspace")
	labelName := getenv("STACKYARD_AZURE_DEFENDERFOREASM_LABEL_NAME", "ThisisaLabel")
	taskID := getenv("STACKYARD_AZURE_DEFENDERFOREASM_TASK_ID", "ThisisaTaskId")

	workspaceBase := fmt.Sprintf(
		"/azure/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Easm/workspaces/%s",
		subscriptionID,
		resourceGroup,
		workspaceName,
	)
	labelBase := workspaceBase + "/labels/" + labelName
	taskBase := workspaceBase + "/tasks/" + taskID

	calls := []struct {
		name     string
		method   string
		path     string
		payload  any
		statuses []int
	}{
		{
			name:     "ListProviderOperations",
			method:   http.MethodGet,
			path:     "/azure/providers/Microsoft.Easm/operations?api-version=" + defenderForEASMControlPlanePreviewAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateOrUpdateWorkspace",
			method: http.MethodPut,
			path:   workspaceBase + "?api-version=" + defenderForEASMControlPlanePreviewAPIVersion,
			payload: map[string]any{
				"location": location,
				"tags": map[string]any{
					"Environment": "local",
					"Service":     "defenderforeasm",
				},
				"properties": map[string]any{},
			},
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:     "GetWorkspace",
			method:   http.MethodGet,
			path:     workspaceBase + "?api-version=" + defenderForEASMControlPlanePreviewAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "ListWorkspacesByResourceGroup",
			method: http.MethodGet,
			path: fmt.Sprintf(
				"/azure/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Easm/workspaces?api-version=%s",
				subscriptionID,
				resourceGroup,
				defenderForEASMControlPlanePreviewAPIVersion,
			),
			statuses: []int{http.StatusOK},
		},
		{
			name:   "ListWorkspacesBySubscription",
			method: http.MethodGet,
			path: fmt.Sprintf(
				"/azure/subscriptions/%s/providers/Microsoft.Easm/workspaces?api-version=%s",
				subscriptionID,
				defenderForEASMControlPlanePreviewAPIVersion,
			),
			statuses: []int{http.StatusOK},
		},
		{
			name:   "UpdateWorkspace",
			method: http.MethodPatch,
			path:   workspaceBase + "?api-version=" + defenderForEASMControlPlanePreviewAPIVersion,
			payload: map[string]any{
				"tags": map[string]any{
					"Environment": "local-updated",
					"Service":     "defenderforeasm",
				},
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateOrUpdateLabel",
			method: http.MethodPut,
			path:   labelBase + "?api-version=" + defenderForEASMControlPlanePreviewAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"color":       "blue",
					"displayName": "This is a label",
				},
			},
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:     "GetLabelByWorkspace",
			method:   http.MethodGet,
			path:     labelBase + "?api-version=" + defenderForEASMControlPlanePreviewAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListLabelsByWorkspace",
			method:   http.MethodGet,
			path:     workspaceBase + "/labels?api-version=" + defenderForEASMControlPlanePreviewAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "UpdateLabel",
			method: http.MethodPatch,
			path:   labelBase + "?api-version=" + defenderForEASMControlPlanePreviewAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"color":       "green",
					"displayName": "This is an updated label",
				},
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GetTaskByWorkspace",
			method:   http.MethodGet,
			path:     taskBase + "?api-version=" + defenderForEASMControlPlanePreviewAPIVersion,
			statuses: []int{http.StatusOK, http.StatusNotFound},
		},
	}

	client := newDefenderForEASMClient(endpoint, account, subscriptionKey)

	for _, call := range calls {
		payload, status, err := client.doJSON(ctx, call.method, call.path, call.payload, call.statuses...)
		if err != nil {
			fatalf("%s failed: %v", call.name, err)
		}
		printJSON(call.name, status, payload)
	}

	cleanup := []struct {
		name     string
		method   string
		path     string
		statuses []int
	}{
		{
			name:     "DeleteLabel",
			method:   http.MethodDelete,
			path:     labelBase + "?api-version=" + defenderForEASMControlPlanePreviewAPIVersion,
			statuses: []int{http.StatusAccepted, http.StatusNoContent},
		},
		{
			name:     "DeleteWorkspace",
			method:   http.MethodDelete,
			path:     workspaceBase + "?api-version=" + defenderForEASMControlPlanePreviewAPIVersion,
			statuses: []int{http.StatusAccepted, http.StatusNoContent},
		},
	}

	for _, call := range cleanup {
		payload, status, err := client.doJSON(ctx, call.method, call.path, nil, call.statuses...)
		if err != nil {
			fatalf("%s failed: %v", call.name, err)
		}
		printJSON(call.name, status, payload)
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

func printJSON(name string, status int, payload map[string]any) {
	formatted, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		fatalf("%s: marshal response: %v", name, err)
	}
	fmt.Printf("%s -> %d\n%s\n", name, status, string(formatted))
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
