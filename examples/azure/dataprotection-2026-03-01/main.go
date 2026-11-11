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

const dataProtectionAPIVersion = "2026-03-01"

type dataProtectionClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type dataProtectionSharedKeyPolicy struct {
	account         string
	subscriptionKey string
}

func (p *dataProtectionSharedKeyPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newDataProtectionClient(endpoint, account, subscriptionKey string) *dataProtectionClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"dataprotection-2026-03-01",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&dataProtectionSharedKeyPolicy{
				account:         account,
				subscriptionKey: subscriptionKey,
			}},
		},
		&policy.ClientOptions{},
	)
	return &dataProtectionClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *dataProtectionClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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
	account := getenv("STACKYARD_AZURE_DATAPROTECTION_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_DATAPROTECTION_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	resourceGroup := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-dataprotection")
	location := getenv("STACKYARD_AZURE_LOCATION", "westus")
	vaultName := getenv("STACKYARD_AZURE_DATAPROTECTION_VAULT_NAME", "dp-vault-example")
	policyName := getenv("STACKYARD_AZURE_DATAPROTECTION_POLICY_NAME", "dp-policy-example")
	backupInstanceName := getenv("STACKYARD_AZURE_DATAPROTECTION_BACKUP_INSTANCE_NAME", "dp-bi-example")
	jobID := getenv("STACKYARD_AZURE_DATAPROTECTION_JOB_ID", "00000000-0000-0000-0000-000000000001")
	operationID := getenv("STACKYARD_AZURE_DATAPROTECTION_OPERATION_ID", "00000000-0000-0000-0000-000000000002")
	deletedVaultName := getenv("STACKYARD_AZURE_DATAPROTECTION_DELETED_VAULT_NAME", "deleted-vault-01")

	client := newDataProtectionClient(endpoint, account, subscriptionKey)

	vaultBase := fmt.Sprintf(
		"/azure/subscriptions/%s/resourceGroups/%s/providers/Microsoft.DataProtection/backupVaults/%s",
		subscriptionID,
		resourceGroup,
		vaultName,
	)
	policyBase := vaultBase + "/backupPolicies/" + policyName
	backupInstanceBase := vaultBase + "/backupInstances/" + backupInstanceName

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
			path:     "/azure/providers/Microsoft.DataProtection/operations?api-version=" + dataProtectionAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CheckFeatureSupport",
			method: http.MethodPost,
			path: fmt.Sprintf(
				"/azure/subscriptions/%s/providers/Microsoft.DataProtection/locations/%s/checkFeatureSupport?api-version=%s",
				subscriptionID,
				location,
				dataProtectionAPIVersion,
			),
			payload: map[string]any{
				"objectType":  "FeatureValidationRequest",
				"featureType": "DataSourceType",
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CheckBackupVaultNameAvailability",
			method: http.MethodPost,
			path: fmt.Sprintf(
				"/azure/subscriptions/%s/resourceGroups/%s/providers/Microsoft.DataProtection/locations/%s/checkNameAvailability?api-version=%s",
				subscriptionID,
				resourceGroup,
				location,
				dataProtectionAPIVersion,
			),
			payload: map[string]any{
				"name": vaultName,
				"type": "Microsoft.DataProtection/BackupVaults",
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateOrUpdateBackupVault",
			method: http.MethodPut,
			path:   vaultBase + "?api-version=" + dataProtectionAPIVersion,
			payload: map[string]any{
				"location": location,
				"properties": map[string]any{
					"storageSettings": []any{
						map[string]any{
							"type":          "LocallyRedundant",
							"datastoreType": "VaultStore",
						},
					},
				},
				"tags": map[string]any{
					"environment": "local",
				},
			},
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:     "GetBackupVault",
			method:   http.MethodGet,
			path:     vaultBase + "?api-version=" + dataProtectionAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListBackupVaultsInResourceGroup",
			method:   http.MethodGet,
			path:     fmt.Sprintf("/azure/subscriptions/%s/resourceGroups/%s/providers/Microsoft.DataProtection/backupVaults?api-version=%s", subscriptionID, resourceGroup, dataProtectionAPIVersion),
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListBackupVaultsInSubscription",
			method:   http.MethodGet,
			path:     fmt.Sprintf("/azure/subscriptions/%s/providers/Microsoft.DataProtection/backupVaults?api-version=%s", subscriptionID, dataProtectionAPIVersion),
			statuses: []int{http.StatusOK},
		},
		{
			name:   "UpdateBackupVault",
			method: http.MethodPatch,
			path:   vaultBase + "?api-version=" + dataProtectionAPIVersion,
			payload: map[string]any{
				"tags": map[string]any{
					"environment": "local-updated",
				},
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateOrUpdateBackupPolicy",
			method: http.MethodPut,
			path:   policyBase + "?api-version=" + dataProtectionAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"objectType":      "BackupPolicy",
					"datasourceTypes": []any{"Microsoft.Compute/disks"},
					"policyRules":     []any{},
				},
			},
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:     "GetBackupPolicy",
			method:   http.MethodGet,
			path:     policyBase + "?api-version=" + dataProtectionAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListBackupPolicies",
			method:   http.MethodGet,
			path:     vaultBase + "/backupPolicies?api-version=" + dataProtectionAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateOrUpdateBackupInstance",
			method: http.MethodPut,
			path:   backupInstanceBase + "?api-version=" + dataProtectionAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"objectType":   "BackupInstance",
					"friendlyName": "disk-example",
					"dataSourceInfo": map[string]any{
						"objectType":       "Datasource",
						"datasourceType":   "Microsoft.Compute/disks",
						"resourceID":       fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Compute/disks/example-disk", subscriptionID, resourceGroup),
						"resourceName":     "example-disk",
						"resourceType":     "Microsoft.Compute/disks",
						"resourceLocation": location,
					},
					"policyInfo": map[string]any{
						"policyId": fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.DataProtection/backupVaults/%s/backupPolicies/%s", subscriptionID, resourceGroup, vaultName, policyName),
					},
				},
			},
			statuses: []int{http.StatusOK, http.StatusCreated, http.StatusAccepted},
		},
		{
			name:     "GetBackupInstance",
			method:   http.MethodGet,
			path:     backupInstanceBase + "?api-version=" + dataProtectionAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListBackupInstances",
			method:   http.MethodGet,
			path:     vaultBase + "/backupInstances?api-version=" + dataProtectionAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "TriggerAdhocBackup",
			method: http.MethodPost,
			path:   backupInstanceBase + "/backup?api-version=" + dataProtectionAPIVersion,
			payload: map[string]any{
				"backupRuleOptions": map[string]any{
					"ruleName": "Default",
				},
			},
			statuses: []int{http.StatusOK, http.StatusAccepted},
		},
		{
			name:     "ListBackupJobs",
			method:   http.MethodGet,
			path:     vaultBase + "/backupJobs?api-version=" + dataProtectionAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GetBackupJob",
			method:   http.MethodGet,
			path:     vaultBase + "/backupJobs/" + jobID + "?api-version=" + dataProtectionAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "TriggerExportJobs",
			method:   http.MethodPost,
			path:     vaultBase + "/exportBackupJobs?api-version=" + dataProtectionAPIVersion,
			statuses: []int{http.StatusAccepted, http.StatusNoContent},
		},
		{
			name:     "GetExportJobsOperationResult",
			method:   http.MethodGet,
			path:     vaultBase + "/backupJobs/operations/" + operationID + "?api-version=" + dataProtectionAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GetProviderOperationStatus",
			method:   http.MethodGet,
			path:     fmt.Sprintf("/azure/subscriptions/%s/providers/Microsoft.DataProtection/locations/%s/operationStatus/%s?api-version=%s", subscriptionID, location, operationID, dataProtectionAPIVersion),
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GetProviderOperationResult",
			method:   http.MethodGet,
			path:     fmt.Sprintf("/azure/subscriptions/%s/providers/Microsoft.DataProtection/locations/%s/operationResults/%s?api-version=%s", subscriptionID, location, operationID, dataProtectionAPIVersion),
			statuses: []int{http.StatusOK, http.StatusAccepted},
		},
		{
			name:     "GetResourceGroupOperationStatus",
			method:   http.MethodGet,
			path:     fmt.Sprintf("/azure/subscriptions/%s/resourceGroups/%s/providers/Microsoft.DataProtection/locations/%s/operationStatus/%s?api-version=%s", subscriptionID, resourceGroup, location, operationID, dataProtectionAPIVersion),
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GetBackupVaultContextOperationStatus",
			method:   http.MethodGet,
			path:     vaultBase + "/operationStatus/" + operationID + "?api-version=" + dataProtectionAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GetDeletedBackupVault",
			method:   http.MethodGet,
			path:     fmt.Sprintf("/azure/subscriptions/%s/providers/Microsoft.DataProtection/locations/%s/deletedVaults/%s?api-version=%s", subscriptionID, location, deletedVaultName, dataProtectionAPIVersion),
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
			name:     "DeleteBackupInstance",
			method:   http.MethodDelete,
			path:     backupInstanceBase + "?api-version=" + dataProtectionAPIVersion,
			statuses: []int{http.StatusOK, http.StatusAccepted, http.StatusNoContent},
		},
		{
			name:     "DeleteBackupPolicy",
			method:   http.MethodDelete,
			path:     policyBase + "?api-version=" + dataProtectionAPIVersion,
			statuses: []int{http.StatusOK, http.StatusAccepted, http.StatusNoContent},
		},
		{
			name:     "DeleteBackupVault",
			method:   http.MethodDelete,
			path:     vaultBase + "?api-version=" + dataProtectionAPIVersion,
			statuses: []int{http.StatusOK, http.StatusAccepted, http.StatusNoContent},
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
