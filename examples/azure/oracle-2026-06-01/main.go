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
	oracleAPIVersion    = "2026-06-01"
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
		"oracle-2026-06-01",
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
	resourceGroup := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-oracle-example")
	region := getenv("STACKYARD_AZURE_REGION", "eastus")
	infraName := getenv("STACKYARD_AZURE_ORACLE_INFRA", "stackyard-exadata")
	clusterName := getenv("STACKYARD_AZURE_ORACLE_VM_CLUSTER", "stackyard-vmcluster")
	adbName := getenv("STACKYARD_AZURE_ORACLE_ADB", "stackyardadb")

	fmt.Printf("Stackyard Oracle Database@Azure (oracle-2026-06-01) example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newARMClient(endpoint, account, subscriptionKey)
	base := "/azure/subscriptions/" + subscriptionID
	groupPath := base + "/resourcegroups/" + resourceGroup + "?api-version=" + resourcesAPIVersion
	providers := base + "/resourceGroups/" + resourceGroup + "/providers/Oracle.Database/"
	armProviders := strings.TrimPrefix(providers, "/azure")
	oracleSubscription := base + "/providers/Oracle.Database/oracleSubscriptions/default"
	location := base + "/providers/Oracle.Database/locations/" + region + "/"
	infra := providers + "cloudExadataInfrastructures/" + infraName
	cluster := providers + "cloudVmClusters/" + clusterName
	adb := providers + "autonomousDatabases/" + adbName
	vnet := strings.TrimPrefix(base, "/azure") + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/virtualNetworks/oracle-vnet"
	infraBody := map[string]any{"location": region, "zones": []any{"1"}, "properties": map[string]any{
		"displayName": "Exadata infrastructure", "shape": "Exadata.X9M", "computeCount": 2, "storageCount": 3,
		"customerContacts":  []any{map[string]any{"email": "dba@example.com"}},
		"maintenanceWindow": map[string]any{"preference": "CustomPreference", "hoursOfDay": []any{4}, "daysOfWeek": []any{map[string]any{"name": "Sunday"}}, "patchingMode": "Rolling"},
	}}
	clusterBody := map[string]any{"location": region, "properties": map[string]any{
		"cloudExadataInfrastructureId": armProviders + "cloudExadataInfrastructures/" + infraName, "displayName": "VM cluster",
		"giVersion": "19.0.0.0", "hostname": "exa", "clusterName": "exa-cl", "cpuCoreCount": 8, "memorySizeInGbs": 120,
		"sshPublicKeys": []any{"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC7 stackyard@example"},
		"vnetId":        vnet, "subnetId": vnet + "/subnets/oracle-delegated", "dataStoragePercentage": 80,
	}}
	adbBody := map[string]any{"location": region, "properties": map[string]any{
		"displayName": "Autonomous Database", "dataBaseType": "Regular", "adminPassword": "Stackyard-Admin-2026",
		"computeModel": "ECPU", "computeCount": 2, "dataStorageSizeInTbs": 1, "dbWorkload": "OLTP", "characterSet": "AL32UTF8",
	}}

	q := "?api-version=" + oracleAPIVersion

	calls := []struct {
		name     string
		method   string
		path     string
		payload  any
		statuses []int
	}{
		{name: "CreateResourceGroup", method: http.MethodPut, path: groupPath, payload: map[string]any{"location": region}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "CreateOracleSubscription", method: http.MethodPut, path: oracleSubscription + q, payload: map[string]any{"plan": map[string]any{"name": "pay-as-you-go", "product": "oracle-database-at-azure", "publisher": "oracle"}, "properties": map[string]any{"termUnit": "P1Y"}}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "GetOracleSubscription", method: http.MethodGet, path: oracleSubscription + q, statuses: []int{http.StatusOK}},
		{name: "ListActivationLinks", method: http.MethodPost, path: oracleSubscription + "/listActivationLinks" + q, statuses: []int{http.StatusOK}},
		{name: "ListCloudAccountDetails", method: http.MethodPost, path: oracleSubscription + "/listCloudAccountDetails" + q, statuses: []int{http.StatusOK}},
		{name: "ListSaasSubscriptionDetails", method: http.MethodPost, path: oracleSubscription + "/listSaasSubscriptionDetails" + q, statuses: []int{http.StatusOK}},
		{name: "AddAzureSubscriptions", method: http.MethodPost, path: oracleSubscription + "/addAzureSubscriptions" + q, payload: map[string]any{"azureSubscriptionIds": []any{"11111111-1111-1111-1111-111111111111"}}, statuses: []int{http.StatusAccepted, http.StatusOK}},
		{name: "UpdateOracleSubscription", method: http.MethodPatch, path: oracleSubscription + q, payload: map[string]any{"properties": map[string]any{"intent": "Retain"}}, statuses: []int{http.StatusOK}},
		{name: "ListDbSystemShapes", method: http.MethodGet, path: location + "dbSystemShapes" + q, statuses: []int{http.StatusOK}},
		{name: "ListGiVersions", method: http.MethodGet, path: location + "giVersions" + q + "&shape=Exadata.X9M", statuses: []int{http.StatusOK}},
		{name: "ListSystemVersions", method: http.MethodGet, path: location + "systemVersions" + q, statuses: []int{http.StatusOK}},
		{name: "ListCharacterSets", method: http.MethodGet, path: location + "autonomousDatabaseCharacterSets" + q, statuses: []int{http.StatusOK}},
		{name: "ListNationalCharacterSets", method: http.MethodGet, path: location + "autonomousDatabaseNationalCharacterSets" + q, statuses: []int{http.StatusOK}},
		{name: "ListAutonomousDbVersions", method: http.MethodGet, path: location + "autonomousDbVersions" + q, statuses: []int{http.StatusOK}},
		{name: "ListDnsPrivateViews", method: http.MethodGet, path: location + "dnsPrivateViews" + q, statuses: []int{http.StatusOK}},
		{name: "ListDnsPrivateZones", method: http.MethodGet, path: location + "dnsPrivateZones" + q, statuses: []int{http.StatusOK}},
		{name: "CreateExadataInfrastructure", method: http.MethodPut, path: infra + q, payload: infraBody, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "ScaleExadataStorage", method: http.MethodPatch, path: infra + q, payload: map[string]any{"properties": map[string]any{"storageCount": 4}}, statuses: []int{http.StatusOK}},
		{name: "AddStorageCapacity", method: http.MethodPost, path: infra + "/addStorageCapacity" + q, statuses: []int{http.StatusOK}},
		{name: "ListDbServers", method: http.MethodGet, path: infra + "/dbServers" + q, statuses: []int{http.StatusOK}},
		{name: "ListInfrastructures", method: http.MethodGet, path: providers + "cloudExadataInfrastructures" + q, statuses: []int{http.StatusOK}},
		{name: "CreateVmCluster", method: http.MethodPut, path: cluster + q, payload: clusterBody, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "GetVmCluster", method: http.MethodGet, path: cluster + q, statuses: []int{http.StatusOK}},
		{name: "ResizeVmCluster", method: http.MethodPatch, path: cluster + q, payload: map[string]any{"properties": map[string]any{"cpuCoreCount": 12, "displayName": "VM cluster (resized)"}}, statuses: []int{http.StatusOK}},
		{name: "ListDbNodes", method: http.MethodGet, path: cluster + "/dbNodes" + q, statuses: []int{http.StatusOK}},
		{name: "CreateVirtualNetworkAddress", method: http.MethodPut, path: cluster + "/virtualNetworkAddresses/app-vip" + q, payload: map[string]any{"properties": map[string]any{"ipAddress": "10.0.1.50"}}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "ListVirtualNetworkAddresses", method: http.MethodGet, path: cluster + "/virtualNetworkAddresses" + q, statuses: []int{http.StatusOK}},
		{name: "ListVmClustersBySubscription", method: http.MethodGet, path: base + "/providers/Oracle.Database/cloudVmClusters" + q, statuses: []int{http.StatusOK}},
		{name: "CreateAutonomousDatabase", method: http.MethodPut, path: adb + q, payload: adbBody, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "GetAutonomousDatabase", method: http.MethodGet, path: adb + q, statuses: []int{http.StatusOK}},
		{name: "UpdateAutonomousDatabase", method: http.MethodPatch, path: adb + q, payload: map[string]any{"properties": map[string]any{"computeCount": 4, "whitelistedIps": []any{"10.0.0.0/24"}}}, statuses: []int{http.StatusOK}},
		{name: "StopAutonomousDatabase", method: http.MethodPost, path: adb + "/action" + q, payload: map[string]any{"action": "Stop"}, statuses: []int{http.StatusOK}},
		{name: "StartAutonomousDatabase", method: http.MethodPost, path: adb + "/action" + q, payload: map[string]any{"action": "Start"}, statuses: []int{http.StatusOK}},
		{name: "GenerateWallet", method: http.MethodPost, path: adb + "/generateWallet" + q, payload: map[string]any{"password": "Wallet-Pass-2026", "generateType": "Single"}, statuses: []int{http.StatusOK}},
		{name: "ChangeDisasterRecoveryConfiguration", method: http.MethodPost, path: adb + "/changeDisasterRecoveryConfiguration" + q, payload: map[string]any{"disasterRecoveryType": "BackupBased", "isReplicateAutomaticBackups": true}, statuses: []int{http.StatusOK}},
		{name: "CreateLongTermBackup", method: http.MethodPut, path: adb + "/autonomousDatabaseBackups/yearly" + q, payload: map[string]any{"properties": map[string]any{"displayName": "yearly", "retentionPeriodInDays": 365}}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "ListBackups", method: http.MethodGet, path: adb + "/autonomousDatabaseBackups" + q, statuses: []int{http.StatusOK}},
		{name: "DeleteBackup", method: http.MethodDelete, path: adb + "/autonomousDatabaseBackups/yearly" + q, statuses: []int{http.StatusOK, http.StatusNoContent}},
		{name: "DeleteAutonomousDatabase", method: http.MethodDelete, path: adb + q, statuses: []int{http.StatusOK, http.StatusNoContent}},
		{name: "DeleteVirtualNetworkAddress", method: http.MethodDelete, path: cluster + "/virtualNetworkAddresses/app-vip" + q, statuses: []int{http.StatusOK, http.StatusNoContent}},
		{name: "DeleteVmCluster", method: http.MethodDelete, path: cluster + q, statuses: []int{http.StatusOK, http.StatusNoContent}},
		{name: "DeleteExadataInfrastructure", method: http.MethodDelete, path: infra + q, statuses: []int{http.StatusOK, http.StatusNoContent}},
		{name: "DeleteOracleSubscription", method: http.MethodDelete, path: oracleSubscription + q, statuses: []int{http.StatusOK, http.StatusNoContent}},
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
