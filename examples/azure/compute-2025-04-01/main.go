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

const computeAPIVersion = "2025-04-01"

type computeClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type computeSharedKeyPolicy struct {
	account         string
	subscriptionKey string
}

func (p *computeSharedKeyPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newComputeClient(endpoint, account, subscriptionKey string) *computeClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"compute-2025-04-01",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&computeSharedKeyPolicy{
				account:         account,
				subscriptionKey: subscriptionKey,
			}},
		},
		&policy.ClientOptions{},
	)
	return &computeClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *computeClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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
	account := getenv("STACKYARD_AZURE_COMPUTE_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_COMPUTE_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	resourceGroup := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-compute")
	location := getenv("STACKYARD_AZURE_LOCATION", "eastus")

	client := newComputeClient(endpoint, account, subscriptionKey)
	subscriptionBase := "/azure/subscriptions/" + subscriptionID
	computeBase := subscriptionBase + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Compute"
	availabilitySetBase := computeBase + "/availabilitySets/aset-a"
	diskBase := computeBase + "/disks/disk-a"
	vmBase := computeBase + "/virtualMachines/vm-a"

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
			path:     "/azure/providers/Microsoft.Compute/operations?api-version=" + computeAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListSKUs",
			method:   http.MethodGet,
			path:     subscriptionBase + "/providers/Microsoft.Compute/skus?api-version=" + computeAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "CreateAvailabilitySet",
			method:   http.MethodPut,
			path:     availabilitySetBase + "?api-version=" + computeAPIVersion,
			statuses: []int{http.StatusCreated, http.StatusOK},
			payload: map[string]any{
				"location": location,
				"properties": map[string]any{
					"platformFaultDomainCount":  2,
					"platformUpdateDomainCount": 5,
				},
			},
		},
		{
			name:     "CreateDisk",
			method:   http.MethodPut,
			path:     diskBase + "?api-version=" + computeAPIVersion,
			statuses: []int{http.StatusCreated, http.StatusOK},
			payload: map[string]any{
				"location": location,
				"properties": map[string]any{
					"creationData": map[string]any{"createOption": "Empty"},
					"diskSizeGB":   64,
				},
			},
		},
		{
			name:     "CreateVirtualMachine",
			method:   http.MethodPut,
			path:     vmBase + "?api-version=" + computeAPIVersion,
			statuses: []int{http.StatusCreated, http.StatusOK},
			payload: map[string]any{
				"location": location,
				"properties": map[string]any{
					"hardwareProfile": map[string]any{"vmSize": "Standard_DS1_v2"},
					"storageProfile": map[string]any{
						"osDisk": map[string]any{"managedDisk": map[string]any{"id": diskBase}},
					},
					"osProfile":      map[string]any{"computerName": "vm-a", "adminUsername": "stackyard"},
					"networkProfile": map[string]any{"networkInterfaces": []any{}},
				},
			},
		},
		{
			name:     "StartVirtualMachine",
			method:   http.MethodPost,
			path:     vmBase + "/start?api-version=" + computeAPIVersion,
			statuses: []int{http.StatusAccepted},
			payload:  map[string]any{},
		},
		{
			name:     "GetVirtualMachineInstanceView",
			method:   http.MethodGet,
			path:     vmBase + "/instanceView?api-version=" + computeAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListVirtualMachines",
			method:   http.MethodGet,
			path:     computeBase + "/virtualMachines?api-version=" + computeAPIVersion,
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
