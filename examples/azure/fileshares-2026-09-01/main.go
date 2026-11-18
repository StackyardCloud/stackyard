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
	resourcesAPIVersion  = "2021-04-01"
	fileSharesAPIVersion = "2026-09-01"
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
		"fileshares-2026-09-01",
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

func (c *armClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (any, int, error) {
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
		return nil, resp.StatusCode, nil
	}
	var decoded any
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, resp.StatusCode, fmt.Errorf("decode JSON body for %s %s: %w", method, path, err)
	}
	return decoded, resp.StatusCode, nil
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	account := getenv("STACKYARD_AZURE_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	resourceGroup := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-fileshares-example")
	region := getenv("STACKYARD_AZURE_REGION", "eastus")
	shareName := getenv("STACKYARD_AZURE_FILE_SHARE_NAME", "stackyard-share")

	fmt.Printf("Stackyard Azure File Shares (fileshares-2026-09-01) example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newARMClient(endpoint, account, subscriptionKey)
	base := "/azure/subscriptions/" + subscriptionID
	groupPath := base + "/resourcegroups/" + resourceGroup + "?api-version=" + resourcesAPIVersion
	ns := base + "/providers/Microsoft.FileShares"
	share := base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.FileShares/fileShares/" + shareName
	location := ns + "/locations/" + region
	q := "?api-version=" + fileSharesAPIVersion

	calls := []struct {
		name     string
		method   string
		path     string
		payload  any
		statuses []int
	}{
		{name: "CreateResourceGroup", method: http.MethodPut, path: groupPath, payload: map[string]any{"location": region}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "CheckNameAvailability", method: http.MethodPost, path: location + "/checkNameAvailability" + q, payload: map[string]any{"name": shareName, "type": "Microsoft.FileShares/fileShares"}, statuses: []int{http.StatusOK}},
		{name: "GetLimits", method: http.MethodPost, path: location + "/getLimits" + q, payload: map[string]any{"properties": map[string]any{"mediaTier": "SSD", "protocol": "NFS"}}, statuses: []int{http.StatusOK}},
		{name: "GetProvisioningRecommendation", method: http.MethodPost, path: location + "/getProvisioningRecommendation" + q, payload: map[string]any{"properties": map[string]any{"provisionedStorageGiB": 1024}}, statuses: []int{http.StatusOK}},
		{name: "CreateFileShare", method: http.MethodPut, path: share + q, payload: map[string]any{"location": region, "properties": map[string]any{"redundancy": "Local", "provisionedStorageGiB": 1024, "nfsProtocolProperties": map[string]any{"rootSquash": "RootSquash"}}}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "GetFileShare", method: http.MethodGet, path: share + q, statuses: []int{http.StatusOK}},
		{name: "UpdateFileShare", method: http.MethodPatch, path: share + q, payload: map[string]any{"properties": map[string]any{"provisionedStorageGiB": 2048, "provisionedIOPerSec": 8000}}, statuses: []int{http.StatusOK}},
		{name: "ListFileSharesBySubscription", method: http.MethodGet, path: ns + "/fileShares" + q, statuses: []int{http.StatusOK}},
		{name: "CreateSnapshot", method: http.MethodPut, path: share + "/fileShareSnapshots/nightly" + q, payload: map[string]any{"properties": map[string]any{"initiatorId": "stackyard-example", "metadata": map[string]any{"purpose": "example"}}}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "ListSnapshots", method: http.MethodGet, path: share + "/fileShareSnapshots" + q, statuses: []int{http.StatusOK}},
		{name: "ListPrivateLinkResources", method: http.MethodGet, path: share + "/privateLinkResources" + q, statuses: []int{http.StatusOK}},
		{name: "StartRedundancyConversion", method: http.MethodPost, path: share + "/startRedundancyConversion" + q, payload: map[string]any{"properties": map[string]any{"targetRedundancy": "Zone"}}, statuses: []int{http.StatusNoContent}},
		{name: "GetRedundancyConversionStatus", method: http.MethodPost, path: share + "/getRedundancyConversionStatus" + q, statuses: []int{http.StatusOK}},
		{name: "GetUsageData", method: http.MethodPost, path: location + "/getUsageData" + q, statuses: []int{http.StatusOK}},
		{name: "DeleteSnapshot", method: http.MethodDelete, path: share + "/fileShareSnapshots/nightly" + q, statuses: []int{http.StatusOK}},
		{name: "DeleteFileShare", method: http.MethodDelete, path: share + q, statuses: []int{http.StatusOK}},
		{name: "DeleteResourceGroup", method: http.MethodDelete, path: groupPath, statuses: []int{http.StatusOK}},
	}

	for _, call := range calls {
		resp, status, err := client.doJSON(ctx, call.method, call.path, call.payload, call.statuses...)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s failed: %v\n", call.name, err)
			os.Exit(1)
		}
		fmt.Printf("%s -> HTTP %d\n", call.name, status)
		if resp != nil {
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
