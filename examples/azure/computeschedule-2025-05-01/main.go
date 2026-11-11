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

const computeScheduleAPIVersion = "2025-05-01"

type computeScheduleClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type computeScheduleSharedKeyPolicy struct {
	account         string
	subscriptionKey string
}

func (p *computeScheduleSharedKeyPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newComputeScheduleClient(endpoint, account, subscriptionKey string) *computeScheduleClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"computeschedule-2025-05-01",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&computeScheduleSharedKeyPolicy{
				account:         account,
				subscriptionKey: subscriptionKey,
			}},
		},
		&policy.ClientOptions{},
	)
	return &computeScheduleClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *computeScheduleClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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
	account := getenv("STACKYARD_AZURE_COMPUTE_SCHEDULE_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_COMPUTE_SCHEDULE_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	location := getenv("STACKYARD_AZURE_LOCATION", "eastus")
	vmID := getenv("STACKYARD_AZURE_VM_ID", "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-schedule/providers/Microsoft.Compute/virtualMachines/vm-a")

	client := newComputeScheduleClient(endpoint, account, subscriptionKey)
	base := "/azure/subscriptions/" + subscriptionID + "/providers/Microsoft.ComputeSchedule/locations/" + location

	operations, status, err := client.doJSON(ctx, http.MethodGet, "/azure/providers/Microsoft.ComputeSchedule/operations?api-version="+computeScheduleAPIVersion, nil, http.StatusOK)
	if err != nil {
		fatalf("ListOperations failed: %v", err)
	}
	printJSON("ListOperations", status, operations)

	submitPayload, status, err := client.doJSON(ctx, http.MethodPost, base+"/virtualMachinesSubmitStart?api-version="+computeScheduleAPIVersion, map[string]any{
		"correlationid": "example-submit-start",
		"resources": map[string]any{
			"ids": []string{vmID},
		},
		"executionParameters": map[string]any{
			"optimizationPreference": "Cost",
		},
		"schedule": map[string]any{
			"deadline":     "2026-04-13T15:04:05Z",
			"timezone":     "UTC",
			"deadlineType": "InitiateAt",
		},
	}, http.StatusOK)
	if err != nil {
		fatalf("SubmitStart failed: %v", err)
	}
	printJSON("SubmitStart", status, submitPayload)

	operationID := firstOperationID(submitPayload)
	if operationID == "" {
		fatalf("SubmitStart did not return an operationId")
	}

	statusPayload, status, err := client.doJSON(ctx, http.MethodPost, base+"/virtualMachinesGetOperationStatus?api-version="+computeScheduleAPIVersion, map[string]any{
		"correlationid": "example-submit-start",
		"operationIds":  []string{operationID},
	}, http.StatusOK)
	if err != nil {
		fatalf("GetOperationStatus failed: %v", err)
	}
	printJSON("GetOperationStatus", status, statusPayload)

	cancelPayload, status, err := client.doJSON(ctx, http.MethodPost, base+"/virtualMachinesCancelOperations?api-version="+computeScheduleAPIVersion, map[string]any{
		"correlationid": "example-submit-start",
		"operationIds":  []string{operationID},
	}, http.StatusOK)
	if err != nil {
		fatalf("CancelOperations failed: %v", err)
	}
	printJSON("CancelOperations", status, cancelPayload)
}

func firstOperationID(payload map[string]any) string {
	results, ok := payload["results"].([]any)
	if !ok || len(results) == 0 {
		return ""
	}
	first, ok := results[0].(map[string]any)
	if !ok {
		return ""
	}
	operation, ok := first["operation"].(map[string]any)
	if !ok {
		return ""
	}
	id, _ := operation["operationId"].(string)
	return strings.TrimSpace(id)
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
	body, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		fatalf("%s: marshal response: %v", name, err)
	}
	fmt.Printf("\n== %s (%d) ==\n%s\n", name, status, string(body))
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
