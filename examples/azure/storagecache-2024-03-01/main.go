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

const (
	resourcesAPIVersion = "2021-04-01"
	storageCacheAPIVersion = "2024-03-01"
)

type armClient struct {
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

func newARMClient(endpoint, account, subscriptionKey string) *armClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"storagecache-2024-03-01",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&sharedKeyAndSubscriptionPolicy{account: account, subscriptionKey: subscriptionKey}},
		},
		&policy.ClientOptions{},
	)
	return &armClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *armClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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
	account := getenv("STACKYARD_AZURE_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	resourceGroup := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-hpc-cache-example")
	region := getenv("STACKYARD_AZURE_REGION", "eastus")
	cacheName := getenv("STACKYARD_AZURE_HPC_CACHE", "stackyard-render")

	fmt.Printf("Stackyard Azure HPC Cache and Managed Lustre (storagecache-2024-03-01) example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newARMClient(endpoint, account, subscriptionKey)
	base := "/azure/subscriptions/" + subscriptionID
	groupPath := base + "/resourcegroups/" + resourceGroup + "?api-version=" + resourcesAPIVersion
	caches := base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.StorageCache"
	ids := "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers"
	subnet := ids + "/Microsoft.Network/virtualNetworks/hpc/subnets/cache"
	container := ids + "/Microsoft.Storage/storageAccounts/stackyardhpc/blobServices/default/containers/assets"
	cache := caches + "/caches/" + cacheName
	q := "?api-version=" + storageCacheAPIVersion

	calls := []struct {
		name     string
		method   string
		path     string
		payload  any
		statuses []int
	}{
		{name: "CreateResourceGroup", method: http.MethodPut, path: groupPath, payload: map[string]any{"location": region}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "CreateCache", method: http.MethodPut, path: cache + q, payload: map[string]any{"location": region, "sku": map[string]any{"name": "Standard_2G"}, "properties": map[string]any{"cacheSizeGB": 3072, "subnet": subnet}}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "AddBlobNfsTarget", method: http.MethodPut, path: cache + "/storageTargets/assets" + q, payload: map[string]any{"properties": map[string]any{"targetType": "blobNfs", "blobNfs": map[string]any{"target": container, "usageModel": "WRITE_AROUND"}, "junctions": []any{map[string]any{"namespacePath": "/assets", "targetPath": "/"}}}}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "FlushCache", method: http.MethodPost, path: cache + "/flush" + q, statuses: []int{http.StatusAccepted}},
		{name: "StopCache", method: http.MethodPost, path: cache + "/stop" + q, statuses: []int{http.StatusOK}},
		{name: "GetSubnetSize", method: http.MethodPost, path: base + "/providers/Microsoft.StorageCache/getRequiredAmlFSSubnetsSize" + q, payload: map[string]any{"sku": map[string]any{"name": "AMLFS-Durable-Premium-250"}, "storageCapacityTiB": 16}, statuses: []int{http.StatusOK}},
		{name: "CreateLustreFilesystem", method: http.MethodPut, path: caches + "/amlFilesystems/scratch" + q, payload: map[string]any{"location": region, "zones": []any{"1"}, "sku": map[string]any{"name": "AMLFS-Durable-Premium-250"}, "properties": map[string]any{"storageCapacityTiB": 16, "filesystemSubnet": subnet, "maintenanceWindow": map[string]any{"dayOfWeek": "Saturday", "timeOfDayUTC": "22:00"}}}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "DeleteLustreFilesystem", method: http.MethodDelete, path: caches + "/amlFilesystems/scratch" + q, statuses: []int{http.StatusOK}},
		{name: "DeleteCache", method: http.MethodDelete, path: cache + q, statuses: []int{http.StatusOK}},
		{name: "DeleteResourceGroup", method: http.MethodDelete, path: groupPath, statuses: []int{http.StatusOK}},
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
