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
			name:     "CreateResourceGroup",
			method:   http.MethodPut,
			path:     "/azure/subscriptions/" + subscriptionID + "/resourcegroups/" + resourceGroup + "?api-version=2021-04-01",
			payload:  map[string]any{"location": location},
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
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
	}

	for _, call := range calls {
		payload, status, err := client.doJSON(ctx, call.method, call.path, call.payload, call.statuses...)
		if err != nil {
			fatalf("%s failed: %v", call.name, err)
		}
		printJSON(call.name, status, payload)
	}

	// Follow-ups that use the IDs the service hands back: the ad hoc backup's
	// job, the job export's operation (from its Location header) and, after
	// cleanup, the soft-deleted vault.
	q := "?api-version=" + dataProtectionAPIVersion
	run := func(name, method, path string, payload any, statuses ...int) map[string]any {
		body, status, err := client.doJSON(ctx, method, path, payload, statuses...)
		if err != nil {
			fatalf("%s failed: %v", name, err)
		}
		printJSON(name, status, body)
		return body
	}
	backup := run("TriggerAdhocBackup", http.MethodPost, backupInstanceBase+"/backup"+q,
		map[string]any{"backupRuleOptions": map[string]any{"ruleName": "Default"}}, http.StatusOK, http.StatusAccepted)
	jobID, _ := backup["jobId"].(string)
	run("ListBackupJobs", http.MethodGet, vaultBase+"/backupJobs"+q, nil, http.StatusOK)
	run("GetBackupJob", http.MethodGet, vaultBase+"/backupJobs/"+jobID+q, nil, http.StatusOK)

	operationLocation, err := client.postForLocation(ctx, vaultBase+"/exportBackupJobs"+q)
	if err != nil {
		fatalf("TriggerExportJobs failed: %v", err)
	}
	fmt.Printf("TriggerExportJobs -> 202 Location: %s\n", operationLocation)
	operationID, _, _ := strings.Cut(operationLocation[strings.LastIndex(operationLocation, "/")+1:], "?")
	providers := fmt.Sprintf("/azure/subscriptions/%s/providers/Microsoft.DataProtection/locations/%s", subscriptionID, location)
	groupProviders := fmt.Sprintf("/azure/subscriptions/%s/resourceGroups/%s/providers/Microsoft.DataProtection/locations/%s", subscriptionID, resourceGroup, location)
	run("GetExportJobsOperationResult", http.MethodGet, vaultBase+"/backupJobs/operations/"+operationID+q, nil, http.StatusOK)
	run("GetProviderOperationStatus", http.MethodGet, providers+"/operationStatus/"+operationID+q, nil, http.StatusOK)
	run("GetProviderOperationResult", http.MethodGet, providers+"/operationResults/"+operationID+q, nil, http.StatusOK, http.StatusAccepted)
	run("GetResourceGroupOperationStatus", http.MethodGet, groupProviders+"/operationStatus/"+operationID+q, nil, http.StatusOK)
	run("GetBackupVaultContextOperationStatus", http.MethodGet, vaultBase+"/operationStatus/"+operationID+q, nil, http.StatusOK)

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

	deleted := run("ListDeletedBackupVaults", http.MethodGet, providers+"/deletedVaults"+q, nil, http.StatusOK)
	if vaults, _ := deleted["value"].([]any); len(vaults) > 0 {
		name, _ := vaults[0].(map[string]any)["name"].(string)
		run("GetDeletedBackupVault", http.MethodGet, providers+"/deletedVaults/"+name+q, nil, http.StatusOK)
	}
}

// postForLocation POSTs an empty body, expects 202 Accepted and returns the
// Location header naming the operation.
func (c *dataProtectionClient) postForLocation(ctx context.Context, path string) (string, error) {
	req, err := runtime.NewRequest(ctx, http.MethodPost, c.endpoint+path)
	if err != nil {
		return "", fmt.Errorf("create request POST %s: %w", path, err)
	}
	resp, err := c.pipeline.Do(req)
	if err != nil {
		return "", fmt.Errorf("execute request POST %s: %w", path, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if err := ensureStatus(resp.StatusCode, body, http.StatusAccepted); err != nil {
		return "", err
	}
	location := resp.Header.Get("Location")
	if location == "" {
		return "", fmt.Errorf("POST %s: no Location header", path)
	}
	return location, nil
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
