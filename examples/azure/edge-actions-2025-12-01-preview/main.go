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

const edgeActionsAPIVersion = "2025-12-01-preview"

type edgeActionsClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type edgeActionsSharedKeyPolicy struct {
	account         string
	subscriptionKey string
}

func (p *edgeActionsSharedKeyPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newEdgeActionsClient(endpoint, account, subscriptionKey string) *edgeActionsClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"edge-actions-2025-12-01-preview",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{
				&edgeActionsSharedKeyPolicy{account: account, subscriptionKey: subscriptionKey},
			},
		},
		&policy.ClientOptions{},
	)
	return &edgeActionsClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *edgeActionsClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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
	account := getenv("STACKYARD_AZURE_EDGE_ACTIONS_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_EDGE_ACTIONS_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	resourceGroup := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-edge-actions")
	location := getenv("STACKYARD_AZURE_LOCATION", "global")

	edgeActionName := getenv("STACKYARD_AZURE_EDGE_ACTION_NAME", "edgeAction1")
	version := getenv("STACKYARD_AZURE_EDGE_ACTION_VERSION", "version1")
	executionFilter := getenv("STACKYARD_AZURE_EDGE_ACTION_EXECUTION_FILTER", "executionFilter1")
	versionCodeName := getenv("STACKYARD_AZURE_EDGE_ACTION_VERSION_CODE_NAME", "zippedFile")
	versionCodeContent := getenv("STACKYARD_AZURE_EDGE_ACTION_VERSION_CODE_CONTENT", "UEsDBBQAAAAIAI1NzkQAAAAABQAAAA==")
	headerName := getenv("STACKYARD_AZURE_EDGE_ACTION_HEADER_NAME", "x-edge-filter")
	headerValue := getenv("STACKYARD_AZURE_EDGE_ACTION_HEADER_VALUE", "blue")

	client := newEdgeActionsClient(endpoint, account, subscriptionKey)

	subscriptionBase := "/azure/subscriptions/" + subscriptionID
	resourceBase := subscriptionBase + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Cdn"
	edgeActionsResourceBase := resourceBase + "/edgeActions"
	edgeActionBase := edgeActionsResourceBase + "/" + edgeActionName
	versionBase := edgeActionBase + "/versions/" + version
	executionFilterBase := edgeActionBase + "/executionFilters/" + executionFilter

	armVersionID := fmt.Sprintf(
		"/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Cdn/EdgeActions/%s/versions/%s",
		subscriptionID,
		resourceGroup,
		edgeActionName,
		version,
	)

	calls := []struct {
		name     string
		method   string
		path     string
		payload  any
		statuses []int
	}{
		{
			name:   "EdgeActionsCreate",
			method: http.MethodPut,
			path:   edgeActionBase + "?api-version=" + edgeActionsAPIVersion,
			payload: map[string]any{
				"location": location,
				"sku": map[string]any{
					"name": "Standard",
					"tier": "Standard",
				},
				"tags": map[string]any{
					"environment": "local",
					"service":     "edge-actions",
				},
			},
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:     "EdgeActionsGet",
			method:   http.MethodGet,
			path:     edgeActionBase + "?api-version=" + edgeActionsAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "EdgeActionsListByResourceGroup",
			method:   http.MethodGet,
			path:     edgeActionsResourceBase + "?api-version=" + edgeActionsAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "EdgeActionsListBySubscription",
			method:   http.MethodGet,
			path:     subscriptionBase + "/providers/Microsoft.Cdn/edgeActions?api-version=" + edgeActionsAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "EdgeActionsUpdate",
			method: http.MethodPatch,
			path:   edgeActionBase + "?api-version=" + edgeActionsAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{},
				"sku": map[string]any{
					"name": "Standard",
					"tier": "Standard",
				},
				"tags": map[string]any{
					"environment": "local",
					"updated":     "true",
				},
			},
			statuses: []int{http.StatusOK, http.StatusAccepted},
		},
		{
			name:   "EdgeActionVersionsCreate",
			method: http.MethodPut,
			path:   versionBase + "?api-version=" + edgeActionsAPIVersion,
			payload: map[string]any{
				"location": location,
				"properties": map[string]any{
					"deploymentType":   "zip",
					"isDefaultVersion": "True",
				},
				"tags": map[string]any{
					"environment": "local",
				},
			},
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:     "EdgeActionVersionsGet",
			method:   http.MethodGet,
			path:     versionBase + "?api-version=" + edgeActionsAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "EdgeActionVersionsListByEdgeAction",
			method:   http.MethodGet,
			path:     edgeActionBase + "/versions?api-version=" + edgeActionsAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "EdgeActionVersionsDeployVersionCode",
			method: http.MethodPost,
			path:   versionBase + "/deployVersionCode?api-version=" + edgeActionsAPIVersion,
			payload: map[string]any{
				"name":    versionCodeName,
				"content": versionCodeContent,
			},
			statuses: []int{http.StatusOK, http.StatusAccepted},
		},
		{
			name:     "EdgeActionVersionsGetVersionCode",
			method:   http.MethodPost,
			path:     versionBase + "/getVersionCode?api-version=" + edgeActionsAPIVersion,
			statuses: []int{http.StatusOK, http.StatusAccepted},
		},
		{
			name:   "EdgeActionExecutionFiltersCreate",
			method: http.MethodPut,
			path:   executionFilterBase + "?api-version=" + edgeActionsAPIVersion,
			payload: map[string]any{
				"location": location,
				"properties": map[string]any{
					"versionId":                            armVersionID,
					"executionFilterIdentifierHeaderName":  headerName,
					"executionFilterIdentifierHeaderValue": headerValue,
				},
				"tags": map[string]any{
					"environment": "local",
				},
			},
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:     "EdgeActionExecutionFiltersGet",
			method:   http.MethodGet,
			path:     executionFilterBase + "?api-version=" + edgeActionsAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "EdgeActionExecutionFiltersListByEdgeAction",
			method:   http.MethodGet,
			path:     edgeActionBase + "/executionFilters?api-version=" + edgeActionsAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "EdgeActionExecutionFiltersUpdate",
			method: http.MethodPatch,
			path:   executionFilterBase + "?api-version=" + edgeActionsAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"versionId":                            armVersionID,
					"executionFilterIdentifierHeaderName":  headerName,
					"executionFilterIdentifierHeaderValue": "green",
				},
				"tags": map[string]any{
					"environment": "local",
					"updated":     "true",
				},
			},
			statuses: []int{http.StatusOK, http.StatusAccepted},
		},
		{
			name:     "EdgeActionExecutionFiltersDelete",
			method:   http.MethodDelete,
			path:     executionFilterBase + "?api-version=" + edgeActionsAPIVersion,
			statuses: []int{http.StatusOK, http.StatusAccepted, http.StatusNoContent},
		},
		{
			name:   "EdgeActionVersionsUpdate",
			method: http.MethodPatch,
			path:   versionBase + "?api-version=" + edgeActionsAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"deploymentType":   "file",
					"isDefaultVersion": "False",
				},
				"tags": map[string]any{
					"environment": "local",
					"updated":     "true",
				},
			},
			statuses: []int{http.StatusOK, http.StatusAccepted},
		},
		{
			name:     "EdgeActionVersionsSwapDefault",
			method:   http.MethodPost,
			path:     versionBase + "/swapDefault?api-version=" + edgeActionsAPIVersion,
			statuses: []int{http.StatusAccepted, http.StatusNoContent},
		},
		{
			name:     "EdgeActionVersionsDelete",
			method:   http.MethodDelete,
			path:     versionBase + "?api-version=" + edgeActionsAPIVersion,
			statuses: []int{http.StatusOK, http.StatusAccepted, http.StatusNoContent},
		},
		{
			name:     "EdgeActionsDelete",
			method:   http.MethodDelete,
			path:     edgeActionBase + "?api-version=" + edgeActionsAPIVersion,
			statuses: []int{http.StatusOK, http.StatusAccepted, http.StatusNoContent},
		},
	}

	for _, call := range calls {
		response, status, err := client.doJSON(ctx, call.method, call.path, call.payload, call.statuses...)
		if err != nil {
			fatalf("%s failed: %v", call.name, err)
		}
		printJSON(call.name, status, response)
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
