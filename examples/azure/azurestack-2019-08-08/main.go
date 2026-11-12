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

const azureStackAdminAPIVersion = "2019-08-08-preview"

type azureStackAdminClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type azureStackAdminSharedKeyPolicy struct {
	account         string
	subscriptionKey string
}

func (p *azureStackAdminSharedKeyPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newAzureStackAdminClient(endpoint, account, subscriptionKey string) *azureStackAdminClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"azurestack-2019-08-08",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&azureStackAdminSharedKeyPolicy{account: account, subscriptionKey: subscriptionKey}},
		},
		&policy.ClientOptions{},
	)
	return &azureStackAdminClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *azureStackAdminClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, *http.Response, error) {
	req, err := runtime.NewRequest(ctx, method, c.endpoint+path)
	if err != nil {
		return nil, nil, fmt.Errorf("create request %s %s: %w", method, path, err)
	}
	req.Raw().Header.Set("Accept", "application/json")
	if payload != nil {
		req.Raw().Header.Set("Content-Type", "application/json")
		if err := runtime.MarshalAsJSON(req, payload); err != nil {
			return nil, nil, fmt.Errorf("marshal payload %s %s: %w", method, path, err)
		}
	}

	resp, err := c.pipeline.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("execute request %s %s: %w", method, path, err)
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, resp, fmt.Errorf("read response %s %s: %w", method, path, err)
	}
	if err := ensureStatus(resp.StatusCode, body, expectedStatuses...); err != nil {
		return nil, resp, err
	}
	if strings.TrimSpace(string(body)) == "" {
		return map[string]any{}, resp, nil
	}
	var decoded map[string]any
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, resp, fmt.Errorf("decode response %s %s: %w", method, path, err)
	}
	if decoded == nil {
		decoded = map[string]any{}
	}
	return decoded, resp, nil
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
	account := getenv("STACKYARD_AZURE_STACK_ADMIN_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_STACK_ADMIN_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	location := getenv("STACKYARD_AZURE_REGION", "eastus")

	fmt.Printf("Stackyard Azure Stack Admin example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newAzureStackAdminClient(endpoint, account, subscriptionKey)
	base := "/azure/subscriptions/" + subscriptionID + "/providers/Microsoft.Storage.Admin"
	settingsPath := base + "/locations/" + location + "/settings?api-version=" + azureStackAdminAPIVersion
	deletedAccountBase := base + "/locations/" + location + "/storageAccounts/account-deleted-001"
	activeAccountBase := base + "/locations/" + location + "/storageAccounts/account-active-001"

	calls := []struct {
		name     string
		method   string
		path     string
		payload  any
		statuses []int
	}{
		{"ListOperations", http.MethodGet, "/azure/providers/Microsoft.Storage.Admin/operations?api-version=" + azureStackAdminAPIVersion, nil, []int{http.StatusOK}},
		{"ListStorageServices", http.MethodGet, base + "/storageServices?api-version=" + azureStackAdminAPIVersion, nil, []int{http.StatusOK}},
		{"GetSettings", http.MethodGet, settingsPath, nil, []int{http.StatusOK}},
		{"UpdateSettings", http.MethodPut, settingsPath, map[string]any{"properties": map[string]any{"retentionPeriodForDeletedStorageAccountsInDays": 14}}, []int{http.StatusOK}},
		{"ListStorageAccounts", http.MethodGet, base + "/locations/" + location + "/storageAccounts?api-version=" + azureStackAdminAPIVersion, nil, []int{http.StatusOK}},
		{"GetDeletedStorageAccount", http.MethodGet, deletedAccountBase + "?api-version=" + azureStackAdminAPIVersion, nil, []int{http.StatusOK}},
		{"ListAcquisitions", http.MethodGet, base + "/locations/" + location + "/acquisitions?api-version=" + azureStackAdminAPIVersion, nil, []int{http.StatusOK}},
	}

	for _, call := range calls {
		payload, _, err := client.doJSON(ctx, call.method, call.path, call.payload, call.statuses...)
		if err != nil {
			fatalf("%s failed: %v", call.name, err)
		}
		printJSON(call.name, payload)
	}

	undeletePayload, undeleteResp, err := client.doJSON(ctx, http.MethodPost, deletedAccountBase+"/undelete?api-version="+azureStackAdminAPIVersion, map[string]any{
		"newAccountName": "tenantrestored001",
	}, http.StatusAccepted)
	if err != nil {
		fatalf("UndeleteStorageAccount failed: %v", err)
	}
	printJSON("UndeleteStorageAccount", undeletePayload)
	undeleteAsync := undeleteResp.Header.Get("Azure-AsyncOperation")
	if undeleteAsync == "" {
		fatalf("UndeleteStorageAccount missing Azure-AsyncOperation header")
	}
	undeleteAsyncPayload, _, err := client.doJSON(ctx, http.MethodGet, undeleteAsync, nil, http.StatusOK)
	if err != nil {
		fatalf("GetUndeleteAsyncOperation failed: %v", err)
	}
	printJSON("GetUndeleteAsyncOperation", undeleteAsyncPayload)

	reclaimPayload, reclaimResp, err := client.doJSON(ctx, http.MethodPost, activeAccountBase+"/reclaimStorageCapacity?api-version="+azureStackAdminAPIVersion, map[string]any{}, http.StatusAccepted)
	if err != nil {
		fatalf("ReclaimStorageCapacity failed: %v", err)
	}
	printJSON("ReclaimStorageCapacity", reclaimPayload)
	reclaimAsync := reclaimResp.Header.Get("Azure-AsyncOperation")
	if reclaimAsync == "" {
		fatalf("ReclaimStorageCapacity missing Azure-AsyncOperation header")
	}
	reclaimAsyncPayload, _, err := client.doJSON(ctx, http.MethodGet, reclaimAsync, nil, http.StatusOK)
	if err != nil {
		fatalf("GetReclaimAsyncOperation failed: %v", err)
	}
	printJSON("GetReclaimAsyncOperation", reclaimAsyncPayload)
}

func printJSON(name string, payload map[string]any) {
	body, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		fatalf("marshal %s response: %v", name, err)
	}
	fmt.Printf("\n== %s ==\n%s\n", name, string(body))
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
