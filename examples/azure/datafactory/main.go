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

const dataFactoryAPIVersion = "2018-06-01"

type dataFactoryClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type dataFactorySharedKeyPolicy struct {
	account         string
	subscriptionKey string
}

func (p *dataFactorySharedKeyPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newDataFactoryClient(endpoint, account, subscriptionKey string) *dataFactoryClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"datafactory",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{
				&dataFactorySharedKeyPolicy{
					account:         account,
					subscriptionKey: subscriptionKey,
				},
			},
		},
		&policy.ClientOptions{},
	)
	return &dataFactoryClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *dataFactoryClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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
	account := getenv("STACKYARD_AZURE_DATAFACTORY_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_DATAFACTORY_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	resourceGroup := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-datafactory")
	location := getenv("STACKYARD_AZURE_LOCATION", "eastus")
	factoryName := getenv("STACKYARD_AZURE_DATAFACTORY_NAME", "df-example")
	linkedServiceName := getenv("STACKYARD_AZURE_DATAFACTORY_LINKED_SERVICE_NAME", "ls-example")
	datasetName := getenv("STACKYARD_AZURE_DATAFACTORY_DATASET_NAME", "ds-example")
	pipelineName := getenv("STACKYARD_AZURE_DATAFACTORY_PIPELINE_NAME", "pl-example")
	triggerName := getenv("STACKYARD_AZURE_DATAFACTORY_TRIGGER_NAME", "trg-example")

	client := newDataFactoryClient(endpoint, account, subscriptionKey)

	factoryBase := fmt.Sprintf(
		"/azure/subscriptions/%s/resourceGroups/%s/providers/Microsoft.DataFactory/factories/%s",
		subscriptionID,
		resourceGroup,
		factoryName,
	)

	windowEnd := time.Now().UTC()
	windowStart := windowEnd.Add(-1 * time.Hour)

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
			path:     "/azure/providers/Microsoft.DataFactory/operations?api-version=" + dataFactoryAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateOrUpdateFactory",
			method: http.MethodPut,
			path:   factoryBase + "?api-version=" + dataFactoryAPIVersion,
			payload: map[string]any{
				"location": location,
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GetFactory",
			method:   http.MethodGet,
			path:     factoryBase + "?api-version=" + dataFactoryAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListFactoriesByResourceGroup",
			method:   http.MethodGet,
			path:     fmt.Sprintf("/azure/subscriptions/%s/resourceGroups/%s/providers/Microsoft.DataFactory/factories?api-version=%s", subscriptionID, resourceGroup, dataFactoryAPIVersion),
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateOrUpdateLinkedService",
			method: http.MethodPut,
			path:   factoryBase + "/linkedservices/" + linkedServiceName + "?api-version=" + dataFactoryAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"type": "AzureBlobStorage",
					"typeProperties": map[string]any{
						"connectionString": "DefaultEndpointsProtocol=https;AccountName=example;AccountKey=fake;EndpointSuffix=core.windows.net",
					},
				},
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateOrUpdateDataset",
			method: http.MethodPut,
			path:   factoryBase + "/datasets/" + datasetName + "?api-version=" + dataFactoryAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"type": "Binary",
					"linkedServiceName": map[string]any{
						"referenceName": linkedServiceName,
						"type":          "LinkedServiceReference",
					},
					"typeProperties": map[string]any{
						"location": map[string]any{
							"type":       "AzureBlobStorageLocation",
							"container":  "input",
							"folderPath": "raw",
						},
					},
				},
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateOrUpdatePipeline",
			method: http.MethodPut,
			path:   factoryBase + "/pipelines/" + pipelineName + "?api-version=" + dataFactoryAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"activities": []any{},
				},
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:     "CreateOrUpdateTrigger",
			method:   http.MethodPut,
			path:     factoryBase + "/triggers/" + triggerName + "?api-version=" + dataFactoryAPIVersion,
			statuses: []int{http.StatusOK},
			payload: map[string]any{
				"properties": map[string]any{
					"type": "ScheduleTrigger",
					"typeProperties": map[string]any{
						"recurrence": map[string]any{
							"frequency": "Day",
							"interval":  1,
							"startTime": "2026-01-01T00:00:00Z",
						},
					},
					"pipelines": []any{},
				},
			},
		},
		{
			name:     "StartTrigger",
			method:   http.MethodPost,
			path:     factoryBase + "/triggers/" + triggerName + "/start?api-version=" + dataFactoryAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GetTrigger",
			method:   http.MethodGet,
			path:     factoryBase + "/triggers/" + triggerName + "?api-version=" + dataFactoryAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "StopTrigger",
			method:   http.MethodPost,
			path:     factoryBase + "/triggers/" + triggerName + "/stop?api-version=" + dataFactoryAPIVersion,
			statuses: []int{http.StatusOK},
		},
	}

	var pipelineRunID string

	for _, call := range calls {
		payload, status, err := client.doJSON(ctx, call.method, call.path, call.payload, call.statuses...)
		if err != nil {
			fatalf("%s failed: %v", call.name, err)
		}
		printJSON(call.name, status, payload)
	}

	createRunPayload, status, err := client.doJSON(
		ctx,
		http.MethodPost,
		factoryBase+"/pipelines/"+pipelineName+"/createRun?api-version="+dataFactoryAPIVersion,
		map[string]any{},
		http.StatusOK,
	)
	if err != nil {
		fatalf("CreatePipelineRun failed: %v", err)
	}
	printJSON("CreatePipelineRun", status, createRunPayload)

	pipelineRunID = strings.TrimSpace(getString(createRunPayload, "runId"))
	if pipelineRunID == "" {
		pipelineRunID = "00000000-0000-0000-0000-000000000000"
	}

	queryRunsPayload := map[string]any{
		"lastUpdatedAfter":  windowStart.Format(time.RFC3339),
		"lastUpdatedBefore": windowEnd.Format(time.RFC3339),
	}
	payload, status, err := client.doJSON(
		ctx,
		http.MethodPost,
		factoryBase+"/pipelineruns/queryByFactory?api-version="+dataFactoryAPIVersion,
		queryRunsPayload,
		http.StatusOK,
	)
	if err != nil {
		fatalf("QueryPipelineRuns failed: %v", err)
	}
	printJSON("QueryPipelineRuns", status, payload)

	queryActivitiesPayload := map[string]any{
		"lastUpdatedAfter":  windowStart.Format(time.RFC3339),
		"lastUpdatedBefore": windowEnd.Format(time.RFC3339),
	}
	payload, status, err = client.doJSON(
		ctx,
		http.MethodPost,
		factoryBase+"/pipelineruns/"+pipelineRunID+"/queryActivityruns?api-version="+dataFactoryAPIVersion,
		queryActivitiesPayload,
		http.StatusOK,
	)
	if err != nil {
		fatalf("QueryActivityRuns failed: %v", err)
	}
	printJSON("QueryActivityRuns", status, payload)

	cleanup := []struct {
		name     string
		method   string
		path     string
		statuses []int
	}{
		{
			name:     "DeleteTrigger",
			method:   http.MethodDelete,
			path:     factoryBase + "/triggers/" + triggerName + "?api-version=" + dataFactoryAPIVersion,
			statuses: []int{http.StatusOK, http.StatusNoContent},
		},
		{
			name:     "DeletePipeline",
			method:   http.MethodDelete,
			path:     factoryBase + "/pipelines/" + pipelineName + "?api-version=" + dataFactoryAPIVersion,
			statuses: []int{http.StatusOK, http.StatusNoContent},
		},
		{
			name:     "DeleteDataset",
			method:   http.MethodDelete,
			path:     factoryBase + "/datasets/" + datasetName + "?api-version=" + dataFactoryAPIVersion,
			statuses: []int{http.StatusOK, http.StatusNoContent},
		},
		{
			name:     "DeleteLinkedService",
			method:   http.MethodDelete,
			path:     factoryBase + "/linkedservices/" + linkedServiceName + "?api-version=" + dataFactoryAPIVersion,
			statuses: []int{http.StatusOK, http.StatusNoContent},
		},
		{
			name:     "DeleteFactory",
			method:   http.MethodDelete,
			path:     factoryBase + "?api-version=" + dataFactoryAPIVersion,
			statuses: []int{http.StatusOK, http.StatusNoContent},
		},
	}

	for _, call := range cleanup {
		cleanupPayload, cleanupStatus, cleanupErr := client.doJSON(ctx, call.method, call.path, nil, call.statuses...)
		if cleanupErr != nil {
			fatalf("%s failed: %v", call.name, cleanupErr)
		}
		printJSON(call.name, cleanupStatus, cleanupPayload)
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

func getString(payload map[string]any, key string) string {
	if payload == nil {
		return ""
	}
	value, ok := payload[key]
	if !ok {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return typed
	default:
		return ""
	}
}

func printJSON(name string, status int, payload map[string]any) {
	body, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		fatalf("%s: marshal response: %v", name, err)
	}
	fmt.Printf("%s -> %d\n%s\n", name, status, string(body))
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
