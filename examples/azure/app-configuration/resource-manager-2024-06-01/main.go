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

const appConfigurationResourceManagerAPIVersion = "2024-06-01"

type appConfigurationClient struct {
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

func newAppConfigurationClient(endpoint, account, subscriptionKey string) *appConfigurationClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"app-configuration-resource-manager-2024-06-01",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&sharedKeyAndSubscriptionPolicy{
				account:         account,
				subscriptionKey: subscriptionKey,
			}},
		},
		&policy.ClientOptions{},
	)

	return &appConfigurationClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *appConfigurationClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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
	account := getenv("STACKYARD_AZURE_APPCONFIG_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_APPCONFIG_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")

	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	resourceGroup := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-appcfg")
	location := getenv("STACKYARD_AZURE_REGION", "eastus")
	configStore := getenv("STACKYARD_AZURE_APPCONFIG_STORE", "cfg-store")
	keyValue := getenv("STACKYARD_AZURE_APPCONFIG_KEYVALUE", "featureA")
	privateEndpointConnection := getenv("STACKYARD_AZURE_APPCONFIG_PRIVATE_ENDPOINT_CONNECTION", "pec-a")
	privateLinkGroup := getenv("STACKYARD_AZURE_APPCONFIG_PRIVATE_LINK_GROUP", "configStore")
	replica := getenv("STACKYARD_AZURE_APPCONFIG_REPLICA", "replica-a")
	snapshot := getenv("STACKYARD_AZURE_APPCONFIG_SNAPSHOT", "snapshot-a")

	fmt.Printf("Stackyard Azure App Configuration Resource Manager (resource-manager-2024-06-01) example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newAppConfigurationClient(endpoint, account, subscriptionKey)

	providerScope := "/azure/providers/Microsoft.AppConfiguration"
	subscriptionScope := "/azure/subscriptions/" + subscriptionID + "/providers/Microsoft.AppConfiguration"
	storeScope := "/azure/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.AppConfiguration/configurationStores/" + configStore
	deletedScope := subscriptionScope + "/locations/" + location + "/deletedConfigurationStores/" + configStore
	keyValueScope := storeScope + "/keyValues/" + keyValue
	privateEndpointConnectionScope := storeScope + "/privateEndpointConnections/" + privateEndpointConnection
	privateLinkResourceScope := storeScope + "/privateLinkResources/" + privateLinkGroup
	replicaScope := storeScope + "/replicas/" + replica
	snapshotScope := storeScope + "/snapshots/" + snapshot

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
			path:     providerScope + "/operations?api-version=" + appConfigurationResourceManagerAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CheckNameAvailability",
			method: http.MethodPost,
			path:   subscriptionScope + "/checkNameAvailability?api-version=" + appConfigurationResourceManagerAPIVersion,
			payload: map[string]any{
				"name": configStore,
				"type": "Microsoft.AppConfiguration/configurationStores",
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:   "RegionalCheckNameAvailability",
			method: http.MethodPost,
			path:   subscriptionScope + "/locations/" + location + "/checkNameAvailability?api-version=" + appConfigurationResourceManagerAPIVersion,
			payload: map[string]any{
				"name": configStore,
				"type": "Microsoft.AppConfiguration/configurationStores",
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateConfigurationStore",
			method: http.MethodPut,
			path:   storeScope + "?api-version=" + appConfigurationResourceManagerAPIVersion,
			payload: map[string]any{
				"location": location,
				"sku": map[string]any{
					"name": "Standard",
				},
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GetConfigurationStore",
			method:   http.MethodGet,
			path:     storeScope + "?api-version=" + appConfigurationResourceManagerAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "UpdateConfigurationStore",
			method: http.MethodPatch,
			path:   storeScope + "?api-version=" + appConfigurationResourceManagerAPIVersion,
			payload: map[string]any{
				"tags": map[string]any{
					"env": "dev",
				},
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListConfigurationStoresByResourceGroup",
			method:   http.MethodGet,
			path:     "/azure/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.AppConfiguration/configurationStores?api-version=" + appConfigurationResourceManagerAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListConfigurationStoresBySubscription",
			method:   http.MethodGet,
			path:     subscriptionScope + "/configurationStores?api-version=" + appConfigurationResourceManagerAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListKeys",
			method:   http.MethodPost,
			path:     storeScope + "/listKeys?api-version=" + appConfigurationResourceManagerAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "RegenerateKey",
			method: http.MethodPost,
			path:   storeScope + "/regenerateKey?api-version=" + appConfigurationResourceManagerAPIVersion,
			payload: map[string]any{
				"id": "primary",
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListDeletedConfigurationStores",
			method:   http.MethodGet,
			path:     subscriptionScope + "/deletedConfigurationStores?api-version=" + appConfigurationResourceManagerAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GetDeletedConfigurationStore",
			method:   http.MethodGet,
			path:     deletedScope + "?api-version=" + appConfigurationResourceManagerAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "PurgeDeletedConfigurationStore",
			method:   http.MethodPost,
			path:     deletedScope + "/purge?api-version=" + appConfigurationResourceManagerAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateOrUpdateKeyValue",
			method: http.MethodPut,
			path:   keyValueScope + "?api-version=" + appConfigurationResourceManagerAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"value": "true",
				},
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GetKeyValue",
			method:   http.MethodGet,
			path:     keyValueScope + "?api-version=" + appConfigurationResourceManagerAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateOrUpdatePrivateEndpointConnection",
			method: http.MethodPut,
			path:   privateEndpointConnectionScope + "?api-version=" + appConfigurationResourceManagerAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"privateLinkServiceConnectionState": map[string]any{
						"status":      "Approved",
						"description": "approved by stackyard",
					},
				},
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GetPrivateEndpointConnection",
			method:   http.MethodGet,
			path:     privateEndpointConnectionScope + "?api-version=" + appConfigurationResourceManagerAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListPrivateEndpointConnections",
			method:   http.MethodGet,
			path:     storeScope + "/privateEndpointConnections?api-version=" + appConfigurationResourceManagerAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GetPrivateLinkResource",
			method:   http.MethodGet,
			path:     privateLinkResourceScope + "?api-version=" + appConfigurationResourceManagerAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListPrivateLinkResources",
			method:   http.MethodGet,
			path:     storeScope + "/privateLinkResources?api-version=" + appConfigurationResourceManagerAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateReplica",
			method: http.MethodPut,
			path:   replicaScope + "?api-version=" + appConfigurationResourceManagerAPIVersion,
			payload: map[string]any{
				"location": "westus2",
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GetReplica",
			method:   http.MethodGet,
			path:     replicaScope + "?api-version=" + appConfigurationResourceManagerAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListReplicas",
			method:   http.MethodGet,
			path:     storeScope + "/replicas?api-version=" + appConfigurationResourceManagerAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateSnapshot",
			method: http.MethodPut,
			path:   snapshotScope + "?api-version=" + appConfigurationResourceManagerAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"filters": []map[string]any{
						{"key": "*"},
					},
				},
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GetSnapshot",
			method:   http.MethodGet,
			path:     snapshotScope + "?api-version=" + appConfigurationResourceManagerAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "DeleteSnapshot",
			method:   http.MethodDelete,
			path:     snapshotScope + "?api-version=" + appConfigurationResourceManagerAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "DeleteReplica",
			method:   http.MethodDelete,
			path:     replicaScope + "?api-version=" + appConfigurationResourceManagerAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "DeletePrivateEndpointConnection",
			method:   http.MethodDelete,
			path:     privateEndpointConnectionScope + "?api-version=" + appConfigurationResourceManagerAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "DeleteKeyValue",
			method:   http.MethodDelete,
			path:     keyValueScope + "?api-version=" + appConfigurationResourceManagerAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "DeleteConfigurationStore",
			method:   http.MethodDelete,
			path:     storeScope + "?api-version=" + appConfigurationResourceManagerAPIVersion,
			statuses: []int{http.StatusOK},
		},
	}

	for _, call := range calls {
		_, status, err := client.doJSON(ctx, call.method, call.path, call.payload, call.statuses...)
		if err != nil {
			exitf("%s failed: %v", call.name, err)
		}
		fmt.Printf("%s: status=%d\n", call.name, status)
	}
}

func getenv(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
