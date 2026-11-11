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

const defenderForCloudAPIVersion = "2025-06-01"

type defenderForCloudClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type defenderForCloudSharedKeyPolicy struct {
	account         string
	subscriptionKey string
}

func (p *defenderForCloudSharedKeyPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newDefenderForCloudClient(endpoint, account, subscriptionKey string) *defenderForCloudClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"defenderforcloud-2025-06-01",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{
				&defenderForCloudSharedKeyPolicy{
					account:         account,
					subscriptionKey: subscriptionKey,
				},
			},
		},
		&policy.ClientOptions{},
	)

	return &defenderForCloudClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *defenderForCloudClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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
	account := getenv("STACKYARD_AZURE_DEFENDERFORCLOUD_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_DEFENDERFORCLOUD_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	resourceID := getenv(
		"STACKYARD_AZURE_DEFENDERFORCLOUD_RESOURCE_ID",
		"/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/SampleRG/providers/Microsoft.Storage/storageAccounts/samplestorageaccount",
	)
	settingName := getenv("STACKYARD_AZURE_DEFENDERFORCLOUD_SETTING_NAME", "current")
	scanID := getenv("STACKYARD_AZURE_DEFENDERFORCLOUD_SCAN_ID", "latest")

	base := "/azure" + ensureLeadingSlash(resourceID) + "/providers/Microsoft.Security/defenderForStorageSettings/" + settingName
	settingPath := base + "?api-version=" + defenderForCloudAPIVersion
	startScanPath := base + "/startMalwareScan?api-version=" + defenderForCloudAPIVersion
	getScanPath := base + "/malwareScans/" + scanID + "?api-version=" + defenderForCloudAPIVersion
	cancelScanPath := base + "/malwareScans/" + scanID + "/cancelMalwareScan?api-version=" + defenderForCloudAPIVersion

	createPayload := map[string]any{
		"properties": map[string]any{
			"isEnabled": true,
			"malwareScanning": map[string]any{
				"onUpload": map[string]any{
					"isEnabled":     true,
					"capGBPerMonth": -1,
					"filters": map[string]any{
						"excludeBlobsWithPrefix": []any{"unscanned-container", "sample-container/logs"},
						"excludeBlobsWithSuffix": []any{".log", ".jpg"},
						"excludeBlobsLargerThan": 1024,
					},
				},
				"scanResultsEventGridTopicResourceId": "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/SampleRG/providers/Microsoft.EventGrid/topics/sampletopic",
			},
			"sensitiveDataDiscovery": map[string]any{
				"isEnabled": true,
			},
			"overrideSubscriptionLevelSettings": true,
		},
	}

	calls := []struct {
		name     string
		method   string
		path     string
		payload  any
		statuses []int
	}{
		{
			name:     "CreateDefenderForStorageSetting",
			method:   http.MethodPut,
			path:     settingPath,
			payload:  createPayload,
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:     "GetDefenderForStorageSetting",
			method:   http.MethodGet,
			path:     settingPath,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "StartMalwareScan",
			method:   http.MethodPost,
			path:     startScanPath,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GetMalwareScan",
			method:   http.MethodGet,
			path:     getScanPath,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "CancelMalwareScan",
			method:   http.MethodPost,
			path:     cancelScanPath,
			statuses: []int{http.StatusOK},
		},
	}

	client := newDefenderForCloudClient(endpoint, account, subscriptionKey)
	for _, call := range calls {
		payload, status, err := client.doJSON(ctx, call.method, call.path, call.payload, call.statuses...)
		if err != nil {
			fatalf("%s failed: %v", call.name, err)
		}
		printJSON(call.name, status, payload)
	}
}

func ensureLeadingSlash(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "/"
	}
	if strings.HasPrefix(trimmed, "/") {
		return trimmed
	}
	return "/" + trimmed
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
	formatted, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		fatalf("%s: marshal response: %v", name, err)
	}
	fmt.Printf("%s -> %d\n%s\n", name, status, string(formatted))
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
