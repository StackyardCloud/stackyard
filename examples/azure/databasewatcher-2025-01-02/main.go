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

const databaseWatcherAPIVersion = "2025-01-02"

type databaseWatcherClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type databaseWatcherSharedKeyPolicy struct {
	account         string
	subscriptionKey string
}

func (p *databaseWatcherSharedKeyPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newDatabaseWatcherClient(endpoint, account, subscriptionKey string) *databaseWatcherClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"databasewatcher-2025-01-02",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{
				&databaseWatcherSharedKeyPolicy{
					account:         account,
					subscriptionKey: subscriptionKey,
				},
			},
		},
		&policy.ClientOptions{},
	)

	return &databaseWatcherClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *databaseWatcherClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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
	account := getenv("STACKYARD_AZURE_DATABASEWATCHER_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_DATABASEWATCHER_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	resourceGroup := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-databasewatcher")
	location := getenv("STACKYARD_AZURE_LOCATION", "eastus2")
	watcherName := getenv("STACKYARD_AZURE_DATABASEWATCHER_NAME", "databasewatcher-example")
	targetName := getenv("STACKYARD_AZURE_DATABASEWATCHER_TARGET_NAME", "databasewatcher-target-example")
	alertRuleName := getenv("STACKYARD_AZURE_DATABASEWATCHER_ALERT_RULE_NAME", "databasewatcher-alert-example")
	healthValidationName := getenv("STACKYARD_AZURE_DATABASEWATCHER_HEALTH_VALIDATION_NAME", "databasewatcher-health-example")
	sharedPrivateLinkName := getenv("STACKYARD_AZURE_DATABASEWATCHER_SHARED_PRIVATE_LINK_NAME", "databasewatcher-spl-example")

	managedIdentityResourceID := getenv(
		"STACKYARD_AZURE_DATABASEWATCHER_MANAGED_IDENTITY_RESOURCE_ID",
		fmt.Sprintf(
			"/subscriptions/%s/resourceGroups/%s/providers/Microsoft.ManagedIdentity/userAssignedIdentities/databasewatcher-mi-example",
			subscriptionID,
			resourceGroup,
		),
	)
	adxClusterResourceID := getenv(
		"STACKYARD_AZURE_DATABASEWATCHER_ADX_CLUSTER_RESOURCE_ID",
		fmt.Sprintf(
			"/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Kusto/clusters/databasewatcher-adx-example",
			subscriptionID,
			resourceGroup,
		),
	)
	kustoClusterURI := getenv(
		"STACKYARD_AZURE_DATABASEWATCHER_KUSTO_CLUSTER_URI",
		"https://databasewatcher-adx-example.eastus2.kusto.windows.net",
	)
	kustoDataIngestionURI := getenv(
		"STACKYARD_AZURE_DATABASEWATCHER_KUSTO_INGEST_URI",
		"https://ingest-databasewatcher-adx-example.eastus2.kusto.windows.net",
	)
	kustoDatabaseName := getenv("STACKYARD_AZURE_DATABASEWATCHER_KUSTO_DB", "databasewatcher")
	targetSQLDBResourceID := getenv(
		"STACKYARD_AZURE_DATABASEWATCHER_SQL_DB_RESOURCE_ID",
		fmt.Sprintf(
			"/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Sql/servers/databasewatcher-sql/databases/monitoring",
			subscriptionID,
			resourceGroup,
		),
	)
	privateLinkResourceID := getenv(
		"STACKYARD_AZURE_DATABASEWATCHER_PRIVATE_LINK_RESOURCE_ID",
		fmt.Sprintf(
			"/subscriptions/%s/resourceGroups/%s/providers/Microsoft.KeyVault/vaults/databasewatcher-kv",
			subscriptionID,
			resourceGroup,
		),
	)

	client := newDatabaseWatcherClient(endpoint, account, subscriptionKey)

	watcherBase := fmt.Sprintf(
		"/azure/subscriptions/%s/resourceGroups/%s/providers/Microsoft.DatabaseWatcher/watchers/%s",
		subscriptionID,
		resourceGroup,
		watcherName,
	)
	targetBase := watcherBase + "/targets/" + targetName
	alertRuleBase := watcherBase + "/alertRuleResources/" + alertRuleName
	healthValidationBase := watcherBase + "/healthValidations/" + healthValidationName
	sharedPrivateLinkBase := watcherBase + "/sharedPrivateLinkResources/" + sharedPrivateLinkName

	calls := []struct {
		name     string
		method   string
		path     string
		payload  any
		statuses []int
	}{
		{
			name:     "ListProviderOperations",
			method:   http.MethodGet,
			path:     "/azure/providers/Microsoft.DatabaseWatcher/operations?api-version=" + databaseWatcherAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateOrUpdateWatcher",
			method: http.MethodPut,
			path:   watcherBase + "?api-version=" + databaseWatcherAPIVersion,
			payload: map[string]any{
				"location": location,
				"identity": map[string]any{
					"type": "SystemAssigned",
				},
				"tags": map[string]any{
					"environment": "local",
					"service":     "databasewatcher",
				},
				"properties": map[string]any{
					"defaultAlertRuleIdentityResourceId": managedIdentityResourceID,
					"datastore": map[string]any{
						"adxClusterResourceId":    adxClusterResourceID,
						"kustoClusterUri":         kustoClusterURI,
						"kustoClusterDisplayName": "databasewatcher-adx-example",
						"kustoDataIngestionUri":   kustoDataIngestionURI,
						"kustoDatabaseName":       kustoDatabaseName,
						"kustoManagementUrl":      "https://portal.azure.com",
						"kustoOfferingType":       "adx",
					},
				},
			},
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:     "GetWatcher",
			method:   http.MethodGet,
			path:     watcherBase + "?api-version=" + databaseWatcherAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListWatchersByResourceGroup",
			method:   http.MethodGet,
			path:     fmt.Sprintf("/azure/subscriptions/%s/resourceGroups/%s/providers/Microsoft.DatabaseWatcher/watchers?api-version=%s", subscriptionID, resourceGroup, databaseWatcherAPIVersion),
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListWatchersBySubscription",
			method:   http.MethodGet,
			path:     fmt.Sprintf("/azure/subscriptions/%s/providers/Microsoft.DatabaseWatcher/watchers?api-version=%s", subscriptionID, databaseWatcherAPIVersion),
			statuses: []int{http.StatusOK},
		},
		{
			name:   "UpdateWatcher",
			method: http.MethodPatch,
			path:   watcherBase + "?api-version=" + databaseWatcherAPIVersion,
			payload: map[string]any{
				"tags": map[string]any{
					"environment": "local-updated",
					"service":     "databasewatcher",
				},
				"properties": map[string]any{
					"defaultAlertRuleIdentityResourceId": managedIdentityResourceID,
					"datastore": map[string]any{
						"adxClusterResourceId":    adxClusterResourceID,
						"kustoClusterUri":         kustoClusterURI,
						"kustoClusterDisplayName": "databasewatcher-adx-example",
						"kustoDataIngestionUri":   kustoDataIngestionURI,
						"kustoDatabaseName":       kustoDatabaseName,
						"kustoManagementUrl":      "https://portal.azure.com",
						"kustoOfferingType":       "adx",
					},
				},
			},
			statuses: []int{http.StatusOK, http.StatusAccepted},
		},
		{
			name:     "StartWatcher",
			method:   http.MethodPost,
			path:     watcherBase + "/start?api-version=" + databaseWatcherAPIVersion,
			statuses: []int{http.StatusOK, http.StatusAccepted},
		},
		{
			name:     "StopWatcher",
			method:   http.MethodPost,
			path:     watcherBase + "/stop?api-version=" + databaseWatcherAPIVersion,
			statuses: []int{http.StatusOK, http.StatusAccepted},
		},
		{
			name:   "CreateOrUpdateTarget",
			method: http.MethodPut,
			path:   targetBase + "?api-version=" + databaseWatcherAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"targetType":               "SqlDb",
					"targetAuthenticationType": "Aad",
					"connectionServerName":     "databasewatcher-sql.database.windows.net",
					"sqlDbResourceId":          targetSQLDBResourceID,
				},
			},
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:     "GetTarget",
			method:   http.MethodGet,
			path:     targetBase + "?api-version=" + databaseWatcherAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListTargetsByWatcher",
			method:   http.MethodGet,
			path:     watcherBase + "/targets?api-version=" + databaseWatcherAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateOrUpdateAlertRuleResource",
			method: http.MethodPut,
			path:   alertRuleBase + "?api-version=" + databaseWatcherAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"alertRuleResourceId":      fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Insights/scheduledQueryRules/databasewatcher-alert-rule", subscriptionID, resourceGroup),
					"alertRuleTemplateId":      "databasewatcher-template",
					"alertRuleTemplateVersion": "1.0",
					"createdWithProperties":    "CreatedWithActionGroup",
					"creationTime":             "2026-01-01T00:00:00Z",
				},
			},
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:     "GetAlertRuleResource",
			method:   http.MethodGet,
			path:     alertRuleBase + "?api-version=" + databaseWatcherAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListAlertRuleResourcesByWatcher",
			method:   http.MethodGet,
			path:     watcherBase + "/alertRuleResources?api-version=" + databaseWatcherAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateSharedPrivateLinkResource",
			method: http.MethodPut,
			path:   sharedPrivateLinkBase + "?api-version=" + databaseWatcherAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"privateLinkResourceId": privateLinkResourceID,
					"groupId":               "vault",
					"requestMessage":        "approve private link for local database watcher emulation",
					"dnsZone":               "eastus2",
					"status":                "Pending",
				},
			},
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:     "GetSharedPrivateLinkResource",
			method:   http.MethodGet,
			path:     sharedPrivateLinkBase + "?api-version=" + databaseWatcherAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListSharedPrivateLinkResourcesByWatcher",
			method:   http.MethodGet,
			path:     watcherBase + "/sharedPrivateLinkResources?api-version=" + databaseWatcherAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "StartHealthValidation",
			method:   http.MethodPost,
			path:     healthValidationBase + "/startValidation?api-version=" + databaseWatcherAPIVersion,
			statuses: []int{http.StatusOK, http.StatusAccepted},
		},
		{
			name:     "GetHealthValidation",
			method:   http.MethodGet,
			path:     healthValidationBase + "?api-version=" + databaseWatcherAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListHealthValidationsByWatcher",
			method:   http.MethodGet,
			path:     watcherBase + "/healthValidations?api-version=" + databaseWatcherAPIVersion,
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

	cleanup := []struct {
		name     string
		method   string
		path     string
		statuses []int
	}{
		{
			name:     "DeleteSharedPrivateLinkResource",
			method:   http.MethodDelete,
			path:     sharedPrivateLinkBase + "?api-version=" + databaseWatcherAPIVersion,
			statuses: []int{http.StatusAccepted, http.StatusNoContent},
		},
		{
			name:     "DeleteTarget",
			method:   http.MethodDelete,
			path:     targetBase + "?api-version=" + databaseWatcherAPIVersion,
			statuses: []int{http.StatusOK, http.StatusNoContent},
		},
		{
			name:     "DeleteAlertRuleResource",
			method:   http.MethodDelete,
			path:     alertRuleBase + "?api-version=" + databaseWatcherAPIVersion,
			statuses: []int{http.StatusOK, http.StatusNoContent},
		},
		{
			name:     "DeleteWatcher",
			method:   http.MethodDelete,
			path:     watcherBase + "?api-version=" + databaseWatcherAPIVersion,
			statuses: []int{http.StatusAccepted, http.StatusNoContent},
		},
	}

	for _, call := range cleanup {
		payload, status, err := client.doJSON(ctx, call.method, call.path, nil, call.statuses...)
		if err != nil {
			fatalf("%s failed: %v", call.name, err)
		}
		printJSON(call.name, status, payload)
	}
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
