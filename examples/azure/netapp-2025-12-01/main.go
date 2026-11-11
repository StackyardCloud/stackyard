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

const netAppAPIVersion = "2025-12-01"

type netAppClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type netAppSharedKeyPolicy struct {
	account         string
	subscriptionKey string
}

func (p *netAppSharedKeyPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newNetAppClient(endpoint, account, subscriptionKey string) *netAppClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"netapp-2025-12-01",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&netAppSharedKeyPolicy{account: account, subscriptionKey: subscriptionKey}},
		},
		&policy.ClientOptions{},
	)
	return &netAppClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *netAppClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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
	account := getenv("STACKYARD_AZURE_NETAPP_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_NETAPP_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	resourceGroup := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-netapp")
	location := getenv("STACKYARD_AZURE_REGION", "eastus")
	accountName := getenv("STACKYARD_AZURE_NETAPP_ACCOUNT_NAME", "accounta")
	poolName := getenv("STACKYARD_AZURE_NETAPP_POOL_NAME", "poola")
	volumeName := getenv("STACKYARD_AZURE_NETAPP_VOLUME_NAME", "volumea")

	client := newNetAppClient(endpoint, account, subscriptionKey)
	subscriptionBase := "/azure/subscriptions/" + subscriptionID
	accountBase := subscriptionBase + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.NetApp/netAppAccounts/" + accountName
	poolBase := accountBase + "/capacityPools/" + poolName
	volumeBase := poolBase + "/volumes/" + volumeName

	calls := []struct {
		name     string
		method   string
		path     string
		payload  any
		statuses []int
	}{
		{"ListOperations", http.MethodGet, "/azure/providers/Microsoft.NetApp/operations?api-version=" + netAppAPIVersion, nil, []int{http.StatusOK}},
		{"CheckNameAvailability", http.MethodPost, subscriptionBase + "/providers/Microsoft.NetApp/locations/" + location + "/checkNameAvailability?api-version=" + netAppAPIVersion, map[string]any{"name": accountName}, []int{http.StatusOK}},
		{"CreateAccount", http.MethodPut, accountBase + "?api-version=" + netAppAPIVersion, map[string]any{"location": location, "tags": map[string]any{"env": "dev"}, "properties": map[string]any{"nfsV4IDDomain": "example.local"}}, []int{http.StatusCreated, http.StatusOK}},
		{"CreatePool", http.MethodPut, poolBase + "?api-version=" + netAppAPIVersion, map[string]any{"location": location, "properties": map[string]any{"size": 4398046511104, "serviceLevel": "Premium", "qosType": "Auto"}}, []int{http.StatusCreated, http.StatusOK}},
		{"CreateVolume", http.MethodPut, volumeBase + "?api-version=" + netAppAPIVersion, map[string]any{"location": location, "properties": map[string]any{"creationToken": "vol-a", "serviceLevel": "Premium", "usageThreshold": 1073741824, "protocolTypes": []any{"NFSv4.1"}, "subnetId": "/subscriptions/" + subscriptionID + "/resourceGroups/rg-network/providers/Microsoft.Network/virtualNetworks/vnet-a/subnets/netapp"}}, []int{http.StatusCreated, http.StatusOK}},
		{"ListSnapshots", http.MethodGet, volumeBase + "/snapshots?api-version=" + netAppAPIVersion, nil, []int{http.StatusOK}},
		{"ListAccountBackups", http.MethodGet, accountBase + "/backups?api-version=" + netAppAPIVersion, nil, []int{http.StatusOK}},
		{"PatchVolume", http.MethodPatch, volumeBase + "?api-version=" + netAppAPIVersion, map[string]any{"tags": map[string]any{"env": "stage", "owner": "stackyard"}, "properties": map[string]any{"serviceLevel": "Ultra"}}, []int{http.StatusOK}},
		{"ListQuotaLimits", http.MethodGet, subscriptionBase + "/providers/Microsoft.NetApp/locations/" + location + "/quotaLimits?api-version=" + netAppAPIVersion, nil, []int{http.StatusOK}},
		{"DeleteAccount", http.MethodDelete, accountBase + "?api-version=" + netAppAPIVersion, nil, []int{http.StatusAccepted, http.StatusNoContent}},
	}

	for _, call := range calls {
		payload, status, err := client.doJSON(ctx, call.method, call.path, call.payload, call.statuses...)
		if err != nil {
			fatalf("%s failed: %v", call.name, err)
		}
		printJSON(call.name, status, payload)
	}
}

func printJSON(name string, status int, payload map[string]any) {
	body, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		fatalf("marshal %s response: %v", name, err)
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
