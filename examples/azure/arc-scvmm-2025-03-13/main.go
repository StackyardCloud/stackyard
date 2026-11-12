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

const arcSCVMMAPIVersion = "2025-03-13"

type arcSCVMMClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type sharedKeyAndSubscriptionPolicy struct {
	account         string
	subscriptionKey string
}

func (p *sharedKeyAndSubscriptionPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newArcSCVMMClient(endpoint, account, subscriptionKey string) *arcSCVMMClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"arc-scvmm-2025-03-13",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&sharedKeyAndSubscriptionPolicy{account: account, subscriptionKey: subscriptionKey}},
		},
		&policy.ClientOptions{},
	)
	return &arcSCVMMClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *arcSCVMMClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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
		return nil, resp.StatusCode, fmt.Errorf("read response body %s %s: %w", method, path, readErr)
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
		return nil, resp.StatusCode, fmt.Errorf("decode JSON body for %s %s: %w", method, path, err)
	}
	if decoded == nil {
		decoded = map[string]any{}
	}
	return decoded, resp.StatusCode, nil
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	account := getenv("STACKYARD_AZURE_ARC_SCVMM_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_ARC_SCVMM_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	resourceGroup := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-arc-scvmm")
	region := getenv("STACKYARD_AZURE_REGION", "eastus")
	vmmServerName := getenv("STACKYARD_AZURE_ARC_SCVMM_VMM_SERVER", "vmm-a")
	cloudName := getenv("STACKYARD_AZURE_ARC_SCVMM_CLOUD", "cloud-a")
	vmName := getenv("STACKYARD_AZURE_ARC_SCVMM_VM", "machine-a")

	fmt.Printf("Stackyard Azure Arc-enabled SCVMM example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newArcSCVMMClient(endpoint, account, subscriptionKey)
	subscriptionBase := "/azure/subscriptions/" + subscriptionID
	resourceGroupBase := subscriptionBase + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.ScVmm"
	vmmServerID := "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.ScVmm/vmmServers/" + vmmServerName
	resourceURI := "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.HybridCompute/machines/" + vmName
	vmBase := "/azure" + resourceURI + "/providers/Microsoft.ScVmm/virtualMachineInstances/default"

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
			path:     "/azure/providers/Microsoft.ScVmm/operations?api-version=" + arcSCVMMAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateVMMServer",
			method: http.MethodPut,
			path:   resourceGroupBase + "/vmmServers/" + vmmServerName + "?api-version=" + arcSCVMMAPIVersion,
			payload: map[string]any{
				"location": region,
				"extendedLocation": map[string]any{
					"type": "CustomLocation",
					"name": "custom-eastus",
				},
				"properties": map[string]any{
					"fqdn": "vmm-a.contoso.local",
					"port": 8100,
					"credentials": map[string]any{
						"username": "admin",
					},
				},
			},
			statuses: []int{http.StatusCreated, http.StatusOK},
		},
		{
			name:   "CreateCloud",
			method: http.MethodPut,
			path:   resourceGroupBase + "/clouds/" + cloudName + "?api-version=" + arcSCVMMAPIVersion,
			payload: map[string]any{
				"location": region,
				"extendedLocation": map[string]any{
					"type": "CustomLocation",
					"name": "custom-eastus",
				},
				"properties": map[string]any{
					"vmmServerId": vmmServerID,
				},
			},
			statuses: []int{http.StatusCreated, http.StatusOK},
		},
		{
			name:   "CreateVirtualMachineInstance",
			method: http.MethodPut,
			path:   vmBase + "?api-version=" + arcSCVMMAPIVersion,
			payload: map[string]any{
				"location": region,
				"extendedLocation": map[string]any{
					"type": "CustomLocation",
					"name": "custom-eastus",
				},
				"properties": map[string]any{
					"infrastructureProfile": map[string]any{
						"templateId": "tmpl-a",
					},
					"osProfile": map[string]any{
						"computerName": vmName,
					},
				},
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:     "StartVirtualMachineInstance",
			method:   http.MethodPost,
			path:     vmBase + "/start?api-version=" + arcSCVMMAPIVersion,
			payload:  map[string]any{},
			statuses: []int{http.StatusOK},
		},
		{
			name:   "InstallGuestAgent",
			method: http.MethodPut,
			path:   vmBase + "/guestAgents/default?api-version=" + arcSCVMMAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"credentials": map[string]any{
						"username": "contoso\\admin",
					},
					"provisioningAction": "Install",
				},
			},
			statuses: []int{http.StatusCreated, http.StatusOK},
		},
		{
			name:     "ListHybridIdentityMetadata",
			method:   http.MethodGet,
			path:     vmBase + "/hybridIdentityMetadata?api-version=" + arcSCVMMAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "DeleteVirtualMachineInstance",
			method:   http.MethodDelete,
			path:     vmBase + "?api-version=" + arcSCVMMAPIVersion,
			statuses: []int{http.StatusOK, http.StatusNoContent},
		},
	}

	for _, call := range calls {
		resp, status, err := client.doJSON(ctx, call.method, call.path, call.payload, call.statuses...)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s failed: %v\n", call.name, err)
			os.Exit(1)
		}
		fmt.Printf("%s -> HTTP %d\n", call.name, status)
		if len(resp) != 0 {
			encoded, _ := json.MarshalIndent(resp, "", "  ")
			fmt.Println(string(encoded))
		}
	}
}

func getenv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
