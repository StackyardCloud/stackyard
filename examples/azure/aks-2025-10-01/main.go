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

const aksAPIVersion = "2025-10-01"

type aksClient struct {
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

func newAKSClient(endpoint, account, subscriptionKey string) *aksClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"aks-2025-10-01",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&sharedKeyAndSubscriptionPolicy{account: account, subscriptionKey: subscriptionKey}},
		},
		&policy.ClientOptions{},
	)
	return &aksClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *aksClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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

	if len(expectedStatuses) == 0 {
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, resp.StatusCode, fmt.Errorf("unexpected status %d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
		}
	} else {
		matched := false
		for _, status := range expectedStatuses {
			if resp.StatusCode == status {
				matched = true
				break
			}
		}
		if !matched {
			return nil, resp.StatusCode, fmt.Errorf("unexpected status %d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
		}
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
	account := getenv("STACKYARD_AZURE_AISERVICES_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_AISERVICES_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	resourceGroup := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-aks")
	location := getenv("STACKYARD_AZURE_REGION", "eastus")
	clusterName := getenv("STACKYARD_AZURE_AKS_CLUSTER", "cluster-a")
	agentPoolName := getenv("STACKYARD_AZURE_AKS_AGENT_POOL", "system")
	snapshotName := getenv("STACKYARD_AZURE_AKS_SNAPSHOT", "snapshot-a")
	managedNamespace := getenv("STACKYARD_AZURE_AKS_MANAGED_NAMESPACE", "ns-a")

	fmt.Printf("Stackyard Azure AKS (aks-2025-10-01) example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newAKSClient(endpoint, account, subscriptionKey)
	scope := "/azure/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.ContainerService/managedClusters/" + clusterName

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
			path:     "/azure/providers/Microsoft.ContainerService/operations?api-version=" + aksAPIVersion,
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:   "CreateManagedCluster",
			method: http.MethodPut,
			path:   scope + "?api-version=" + aksAPIVersion,
			payload: map[string]any{
				"location": location,
				"identity": map[string]any{
					"type": "SystemAssigned",
				},
				"properties": map[string]any{
					"dnsPrefix": "stackyardaks",
				},
			},
			statuses: []int{http.StatusOK, http.StatusCreated, http.StatusAccepted, http.StatusNotImplemented},
		},
		{
			name:     "ListManagedClustersBySubscription",
			method:   http.MethodGet,
			path:     "/azure/subscriptions/" + subscriptionID + "/providers/Microsoft.ContainerService/managedClusters?api-version=" + aksAPIVersion,
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:   "CreateAgentPool",
			method: http.MethodPut,
			path:   scope + "/agentPools/" + agentPoolName + "?api-version=" + aksAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"count":  1,
					"vmSize": "Standard_DS2_v2",
					"mode":   "System",
				},
			},
			statuses: []int{http.StatusOK, http.StatusCreated, http.StatusAccepted, http.StatusNotImplemented},
		},
		{
			name:     "ListMachines",
			method:   http.MethodGet,
			path:     scope + "/agentPools/" + agentPoolName + "/machines?api-version=" + aksAPIVersion,
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:   "CreateMaintenanceConfiguration",
			method: http.MethodPut,
			path:   scope + "/maintenanceConfigurations/default?api-version=" + aksAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"maintenanceWindow": map[string]any{
						"schedule": map[string]any{
							"weekly": map[string]any{
								"dayOfWeek":     "Monday",
								"intervalWeeks": 1,
							},
						},
					},
				},
			},
			statuses: []int{http.StatusOK, http.StatusCreated, http.StatusAccepted, http.StatusNotImplemented},
		},
		{
			name:   "CreateManagedNamespace",
			method: http.MethodPut,
			path:   scope + "/managedNamespaces/" + managedNamespace + "?api-version=" + aksAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"namespaceLabels": map[string]any{"env": "dev"},
				},
			},
			statuses: []int{http.StatusOK, http.StatusCreated, http.StatusAccepted, http.StatusNotImplemented},
		},
		{
			name:     "ListPrivateEndpointConnections",
			method:   http.MethodGet,
			path:     scope + "/privateEndpointConnections?api-version=" + aksAPIVersion,
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:     "ListPrivateLinkResources",
			method:   http.MethodGet,
			path:     scope + "/privateLinkResources?api-version=" + aksAPIVersion,
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:   "ResolvePrivateLinkServiceID",
			method: http.MethodPost,
			path:   scope + "/resolvePrivateLinkServiceId?api-version=" + aksAPIVersion,
			payload: map[string]any{
				"name": clusterName,
				"privateLinkServiceConnectionState": map[string]any{
					"status":      "Approved",
					"description": "approved by stackyard",
				},
			},
			statuses: []int{http.StatusOK, http.StatusCreated, http.StatusAccepted, http.StatusNotImplemented},
		},
		{
			name:   "CreateSnapshot",
			method: http.MethodPut,
			path:   "/azure/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.ContainerService/snapshots/" + snapshotName + "?api-version=" + aksAPIVersion,
			payload: map[string]any{
				"location": location,
				"properties": map[string]any{
					"snapshotType": "NodePool",
				},
			},
			statuses: []int{http.StatusOK, http.StatusCreated, http.StatusAccepted, http.StatusNotImplemented},
		},
		{
			name:     "ListTrustedAccessRoles",
			method:   http.MethodGet,
			path:     "/azure/subscriptions/" + subscriptionID + "/providers/Microsoft.ContainerService/locations/" + location + "/trustedAccessRoles?api-version=" + aksAPIVersion,
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:   "CreateTrustedAccessRoleBinding",
			method: http.MethodPut,
			path:   scope + "/trustedAccessRoleBindings/binding-a?api-version=" + aksAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"sourceResourceId": "/subscriptions/" + subscriptionID + "/resourceGroups/rg-app/providers/Microsoft.App/containerApps/app-a",
					"roles": []string{
						"Microsoft.ContainerService/managedClusters/trustedAccessRoleBindings/read",
					},
				},
			},
			statuses: []int{http.StatusOK, http.StatusCreated, http.StatusAccepted, http.StatusNotImplemented},
		},
	}

	notImplementedCount := 0
	for _, call := range calls {
		_, status, err := client.doJSON(ctx, call.method, call.path, call.payload, call.statuses...)
		if err != nil {
			exitf("%s failed: %v", call.name, err)
		}
		if status == http.StatusNotImplemented {
			notImplementedCount++
			fmt.Printf("Route is recognized but not implemented yet: %s\n", call.path)
			continue
		}
		fmt.Printf("%s: status=%d\n", call.name, status)
	}

	if notImplementedCount == len(calls) {
		fmt.Println("All AKS routes are staged in this Stackyard build.")
		return
	}
	fmt.Println("Done.")
}

func getenv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
