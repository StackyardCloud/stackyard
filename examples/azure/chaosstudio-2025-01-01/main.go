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

const chaosStudioAPIVersion = "2025-01-01"

type chaosStudioClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type chaosStudioSharedKeyPolicy struct {
	account         string
	subscriptionKey string
}

func (p *chaosStudioSharedKeyPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newChaosStudioClient(endpoint, account, subscriptionKey string) *chaosStudioClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"chaosstudio-2025-01-01",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&chaosStudioSharedKeyPolicy{account: account, subscriptionKey: subscriptionKey}},
		},
		&policy.ClientOptions{},
	)
	return &chaosStudioClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *chaosStudioClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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
	account := getenv("STACKYARD_AZURE_CHAOSSTUDIO_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_CHAOSSTUDIO_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	resourceGroup := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-chaos")
	location := getenv("STACKYARD_AZURE_LOCATION", "eastus")
	experimentName := getenv("STACKYARD_AZURE_CHAOSSTUDIO_EXPERIMENT", "exp-a")
	targetName := getenv("STACKYARD_AZURE_CHAOSSTUDIO_TARGET", "microsoft-agent")
	parentResourceID := getenv("STACKYARD_AZURE_CHAOSSTUDIO_PARENT_RESOURCE_ID", "/subscriptions/"+subscriptionID+"/resourceGroups/rg-vm/providers/Microsoft.Compute/virtualMachines/vm-chaos")

	fmt.Printf("Stackyard Azure Chaos Studio example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newChaosStudioClient(endpoint, account, subscriptionKey)
	locationBase := "/azure/subscriptions/" + subscriptionID + "/providers/Microsoft.Chaos/locations/" + location
	targetBase := "/azure" + parentResourceID + "/providers/Microsoft.Chaos/targets/" + targetName
	experimentBase := "/azure/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Chaos/experiments/" + experimentName

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
			path:     "/azure/providers/Microsoft.Chaos/operations?api-version=" + chaosStudioAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListTargetTypes",
			method:   http.MethodGet,
			path:     locationBase + "/targetTypes?api-version=" + chaosStudioAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListCapabilityTypes",
			method:   http.MethodGet,
			path:     locationBase + "/targetTypes/Microsoft-VirtualMachine/capabilityTypes?api-version=" + chaosStudioAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateTarget",
			method: http.MethodPut,
			path:   targetBase + "?api-version=" + chaosStudioAPIVersion,
			payload: map[string]any{
				"location": location,
				"properties": map[string]any{
					"targetType":  "Microsoft-VirtualMachine",
					"displayName": "VM Chaos Agent",
				},
			},
			statuses: []int{http.StatusCreated, http.StatusOK},
		},
		{
			name:   "CreateCapability",
			method: http.MethodPut,
			path:   targetBase + "/capabilities/Shutdown-1.0?api-version=" + chaosStudioAPIVersion,
			payload: map[string]any{
				"location": location,
				"properties": map[string]any{
					"publisher": "stackyard",
					"version":   "1.0",
				},
			},
			statuses: []int{http.StatusCreated, http.StatusOK},
		},
		{
			name:   "CreateExperiment",
			method: http.MethodPut,
			path:   experimentBase + "?api-version=" + chaosStudioAPIVersion,
			payload: map[string]any{
				"location": location,
				"properties": map[string]any{
					"selectors": []any{
						map[string]any{
							"id":   "Selector1",
							"type": "List",
							"targets": []any{
								map[string]any{"type": "ChaosTarget", "id": targetBase},
							},
						},
					},
					"steps": []any{
						map[string]any{
							"name": "Step1",
							"branches": []any{
								map[string]any{
									"name": "Branch1",
									"actions": []any{
										map[string]any{
											"name":       "Action1",
											"type":       "continuous",
											"selectorId": "Selector1",
											"duration":   "PT1M",
										},
									},
								},
							},
						},
					},
				},
			},
			statuses: []int{http.StatusCreated, http.StatusOK},
		},
		{
			name:     "StartExperiment",
			method:   http.MethodPost,
			path:     experimentBase + "/start?api-version=" + chaosStudioAPIVersion,
			payload:  map[string]any{},
			statuses: []int{http.StatusAccepted},
		},
	}

	var startPayload map[string]any
	for _, call := range calls {
		payload, status, err := client.doJSON(ctx, call.method, call.path, call.payload, call.statuses...)
		if err != nil {
			fatalf("%s failed: %v", call.name, err)
		}
		printJSON(call.name, status, payload)
		if call.name == "StartExperiment" {
			startPayload = payload
		}
	}

	operationID, _ := startPayload["operationId"].(string)
	executionID, _ := startPayload["executionId"].(string)
	if operationID == "" || executionID == "" {
		fatalf("StartExperiment response missing operationId or executionId: %#v", startPayload)
	}

	followUps := []struct {
		name     string
		method   string
		path     string
		payload  any
		statuses []int
	}{
		{
			name:     "GetOperationStatus",
			method:   http.MethodGet,
			path:     locationBase + "/operationStatuses/" + operationID + "?api-version=" + chaosStudioAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListExecutions",
			method:   http.MethodGet,
			path:     experimentBase + "/executions?api-version=" + chaosStudioAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GetExecutionDetails",
			method:   http.MethodPost,
			path:     experimentBase + "/executions/" + executionID + "/getExecutionDetails?api-version=" + chaosStudioAPIVersion,
			payload:  map[string]any{},
			statuses: []int{http.StatusOK},
		},
	}

	for _, call := range followUps {
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
