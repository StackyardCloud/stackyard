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

const dataLakeAnalyticsAPIVersion = "2016-11-01"

type dataLakeAnalyticsClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type dataLakeAnalyticsSharedKeyPolicy struct {
	account         string
	subscriptionKey string
}

func (p *dataLakeAnalyticsSharedKeyPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newDataLakeAnalyticsClient(endpoint, account, subscriptionKey string) *dataLakeAnalyticsClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"datalakeanalytics-2016-11-01",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&dataLakeAnalyticsSharedKeyPolicy{
				account:         account,
				subscriptionKey: subscriptionKey,
			}},
		},
		&policy.ClientOptions{},
	)
	return &dataLakeAnalyticsClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *dataLakeAnalyticsClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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
	account := getenv("STACKYARD_AZURE_DATALAKEANALYTICS_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_DATALAKEANALYTICS_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	resourceGroup := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-datalakeanalytics")
	location := getenv("STACKYARD_AZURE_LOCATION", "eastus2")
	accountName := getenv("STACKYARD_AZURE_DATALAKEANALYTICS_ACCOUNT_NAME", "contosoadla")
	storageAccountName := getenv("STACKYARD_AZURE_DATALAKEANALYTICS_STORAGE_ACCOUNT_NAME", "teststorage")
	dataLakeStoreName := getenv("STACKYARD_AZURE_DATALAKEANALYTICS_DATALAKESTORE_NAME", "testadls")
	computePolicyName := getenv("STACKYARD_AZURE_DATALAKEANALYTICS_COMPUTE_POLICY_NAME", "test-policy")
	firewallRuleName := getenv("STACKYARD_AZURE_DATALAKEANALYTICS_FIREWALL_RULE_NAME", "test-rule")

	client := newDataLakeAnalyticsClient(endpoint, account, subscriptionKey)

	accountBase := fmt.Sprintf(
		"/azure/subscriptions/%s/resourceGroups/%s/providers/Microsoft.DataLakeAnalytics/accounts/%s",
		subscriptionID,
		resourceGroup,
		accountName,
	)

	calls := []struct {
		name     string
		method   string
		path     string
		payload  any
		statuses []int
	}{
		{
			name:   "CheckNameAvailability",
			method: http.MethodPost,
			path: fmt.Sprintf(
				"/azure/subscriptions/%s/providers/Microsoft.DataLakeAnalytics/locations/%s/checkNameAvailability?api-version=%s",
				subscriptionID,
				location,
				dataLakeAnalyticsAPIVersion,
			),
			payload: map[string]any{
				"name": accountName,
				"type": "Microsoft.DataLakeAnalytics/accounts",
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateOrUpdateAccount",
			method: http.MethodPut,
			path:   accountBase + "?api-version=" + dataLakeAnalyticsAPIVersion,
			payload: map[string]any{
				"location": location,
				"properties": map[string]any{
					"defaultDataLakeStoreAccount": dataLakeStoreName,
				},
			},
			statuses: []int{http.StatusOK, http.StatusCreated, http.StatusAccepted},
		},
		{
			name:     "GetAccount",
			method:   http.MethodGet,
			path:     accountBase + "?api-version=" + dataLakeAnalyticsAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "AddStorageAccount",
			method: http.MethodPut,
			path: accountBase + "/storageAccounts/" + storageAccountName +
				"?api-version=" + dataLakeAnalyticsAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"accessKey": "test-storage-access-key",
					"suffix":    "core.windows.net",
				},
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:   "AddDataLakeStoreAccount",
			method: http.MethodPut,
			path: accountBase + "/dataLakeStoreAccounts/" + dataLakeStoreName +
				"?api-version=" + dataLakeAnalyticsAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"suffix": "azuredatalakestore.net",
				},
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateOrUpdateComputePolicy",
			method: http.MethodPut,
			path: accountBase + "/computePolicies/" + computePolicyName +
				"?api-version=" + dataLakeAnalyticsAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"objectId":                     "776b9091-8916-4638-87f7-9c989a38da98",
					"objectType":                   "User",
					"maxDegreeOfParallelismPerJob": 4,
					"minPriorityPerJob":            50,
				},
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateOrUpdateFirewallRule",
			method: http.MethodPut,
			path: accountBase + "/firewallRules/" + firewallRuleName +
				"?api-version=" + dataLakeAnalyticsAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"startIpAddress": "1.1.1.1",
					"endIpAddress":   "2.2.2.2",
				},
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:   "ListAccounts",
			method: http.MethodGet,
			path: fmt.Sprintf(
				"/azure/subscriptions/%s/providers/Microsoft.DataLakeAnalytics/accounts?api-version=%s",
				subscriptionID,
				dataLakeAnalyticsAPIVersion,
			),
			statuses: []int{http.StatusOK},
		},
		{
			name:   "ListAccountsByResourceGroup",
			method: http.MethodGet,
			path: fmt.Sprintf(
				"/azure/subscriptions/%s/resourceGroups/%s/providers/Microsoft.DataLakeAnalytics/accounts?api-version=%s",
				subscriptionID,
				resourceGroup,
				dataLakeAnalyticsAPIVersion,
			),
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListStorageAccountsByAccount",
			method:   http.MethodGet,
			path:     accountBase + "/storageAccounts?api-version=" + dataLakeAnalyticsAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListDataLakeStoreAccountsByAccount",
			method:   http.MethodGet,
			path:     accountBase + "/dataLakeStoreAccounts?api-version=" + dataLakeAnalyticsAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListComputePoliciesByAccount",
			method:   http.MethodGet,
			path:     accountBase + "/computePolicies?api-version=" + dataLakeAnalyticsAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListFirewallRulesByAccount",
			method:   http.MethodGet,
			path:     accountBase + "/firewallRules?api-version=" + dataLakeAnalyticsAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListProviderOperations",
			method:   http.MethodGet,
			path:     "/azure/providers/Microsoft.DataLakeAnalytics/operations?api-version=" + dataLakeAnalyticsAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListJobs",
			method:   http.MethodGet,
			path:     "/azure/Jobs?api-version=" + dataLakeAnalyticsAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListPipelines",
			method:   http.MethodGet,
			path:     "/azure/pipelines?api-version=" + dataLakeAnalyticsAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListRecurrences",
			method:   http.MethodGet,
			path:     "/azure/recurrences?api-version=" + dataLakeAnalyticsAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListCatalogDatabases",
			method:   http.MethodGet,
			path:     "/azure/catalog/usql/databases?api-version=" + dataLakeAnalyticsAPIVersion,
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
	if len(expectedStatuses) == 0 {
		if status < 200 || status >= 300 {
			return fmt.Errorf("unexpected status %d body=%s", status, strings.TrimSpace(string(body)))
		}
		return nil
	}
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
