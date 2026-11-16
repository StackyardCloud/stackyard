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
	storageSyncAPIVersion = "2022-09-01"
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
		"storagesync-2022-09-01",
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
	resourceGroup := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-file-sync-example")
	region := getenv("STACKYARD_AZURE_REGION", "eastus")
	serviceName := getenv("STACKYARD_AZURE_SYNC_SERVICE", "stackyard-files")

	fmt.Printf("Stackyard Azure File Sync (storagesync-2022-09-01) example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newARMClient(endpoint, account, subscriptionKey)
	base := "/azure/subscriptions/" + subscriptionID
	groupPath := base + "/resourcegroups/" + resourceGroup + "?api-version=" + resourcesAPIVersion
	service := base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.StorageSync/storageSyncServices/" + serviceName
	serverID := "3a048283-338f-4002-a9dd-a50fdadcb392"
	serverResourceID := "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.StorageSync/storageSyncServices/" + serviceName + "/registeredServers/" + serverID
	accountID := "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Storage/storageAccounts/stackyardfiles"
	group := service + "/syncGroups/departments"
	q := "?api-version=" + storageSyncAPIVersion

	calls := []struct {
		name     string
		method   string
		path     string
		payload  any
		statuses []int
	}{
		{name: "CreateResourceGroup", method: http.MethodPut, path: groupPath, payload: map[string]any{"location": region}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "CheckNameAvailability", method: http.MethodPost, path: base + "/providers/Microsoft.StorageSync/locations/" + region + "/checkNameAvailability" + q, payload: map[string]any{"name": serviceName, "type": "Microsoft.StorageSync/storageSyncServices"}, statuses: []int{http.StatusOK}},
		{name: "CreateSyncService", method: http.MethodPut, path: service + q, payload: map[string]any{"location": region, "properties": map[string]any{"incomingTrafficPolicy": "AllowAllTraffic"}}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "RegisterServer", method: http.MethodPut, path: service + "/registeredServers/" + serverID + q, payload: map[string]any{"properties": map[string]any{"serverCertificate": "MIIDdzCCAl+gAwIBAgIQ", "agentVersion": "18.0.0.0", "serverOSVersion": "10.0.20348.0", "friendlyName": "fs01.contoso.com"}}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "CreateSyncGroup", method: http.MethodPut, path: group + q, payload: map[string]any{"properties": map[string]any{}}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "CreateCloudEndpoint", method: http.MethodPut, path: group + "/cloudEndpoints/departments-share" + q, payload: map[string]any{"properties": map[string]any{"storageAccountResourceId": accountID, "azureFileShareName": "departments"}}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "CreateServerEndpoint", method: http.MethodPut, path: group + "/serverEndpoints/fs01" + q, payload: map[string]any{"properties": map[string]any{"serverLocalPath": "D:\\Departments", "serverResourceId": serverResourceID, "cloudTiering": "on", "volumeFreeSpacePercent": 20, "tierFilesOlderThanDays": 30}}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "TriggerChangeDetection", method: http.MethodPost, path: group + "/cloudEndpoints/departments-share/triggerChangeDetection" + q, payload: map[string]any{"directoryPath": "/", "changeDetectionMode": "Recursive"}, statuses: []int{http.StatusAccepted}},
		{name: "ListServerEndpoints", method: http.MethodGet, path: group + "/serverEndpoints" + q, statuses: []int{http.StatusOK}},
		{name: "DeleteServerEndpoint", method: http.MethodDelete, path: group + "/serverEndpoints/fs01" + q, statuses: []int{http.StatusOK}},
		{name: "DeleteSyncService", method: http.MethodDelete, path: service + q, statuses: []int{http.StatusOK}},
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
