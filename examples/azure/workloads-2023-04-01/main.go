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
	workloadsAPIVersion = "2023-04-01"
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
		"workloads-2023-04-01",
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
	resourceGroup := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-workloads-example")
	region := getenv("STACKYARD_AZURE_REGION", "eastus")
	sid := getenv("STACKYARD_AZURE_SAP_SID", "X00")
	monitorName := getenv("STACKYARD_AZURE_SAP_MONITOR", "stackyard-sap-monitor")

	fmt.Printf("Stackyard Azure Workloads for SAP (workloads-2023-04-01) example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newARMClient(endpoint, account, subscriptionKey)
	base := "/azure/subscriptions/" + subscriptionID
	groupPath := base + "/resourcegroups/" + resourceGroup + "?api-version=" + resourcesAPIVersion
	providers := base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Workloads/"
	vis := providers + "sapVirtualInstances/" + sid
	monitor := providers + "monitors/" + monitorName
	metadata := base + "/providers/Microsoft.Workloads/locations/" + region + "/sapVirtualInstanceMetadata/default/"
	armBase := strings.TrimPrefix(base, "/azure")
	subnet := armBase + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/virtualNetworks/sap-vnet/subnets/sap"
	system := map[string]any{"appLocation": region, "environment": "NonProd", "sapProduct": "S4HANA", "databaseType": "HANA", "deploymentType": "ThreeTier"}
	server := func(count int) map[string]any {
		return map[string]any{"subnetId": subnet, "instanceCount": count, "virtualMachineConfiguration": map[string]any{
			"vmSize":         "Standard_E16ds_v4",
			"imageReference": map[string]any{"publisher": "RedHat", "offer": "RHEL-SAP", "sku": "84sapha-gen2", "version": "latest"},
			"osProfile": map[string]any{"adminUsername": "azureuser", "osConfiguration": map[string]any{
				"osType": "Linux", "disablePasswordAuthentication": true, "sshKeyPair": map[string]any{"publicKey": "ssh-rsa AAAAB3NzaC1yc2E stackyard"}}},
		}}
	}
	database := server(1)
	database["databaseType"] = "HANA"
	infrastructure := map[string]any{"deploymentType": "ThreeTier", "appResourceGroup": sid + "-RG", "centralServer": server(1), "applicationServer": server(2), "databaseServer": database}
	withSoftware := map[string]any{"configurationType": "Deployment", "appLocation": region, "infrastructureConfiguration": infrastructure,
		"softwareConfiguration": map[string]any{"softwareInstallationType": "ServiceInitiated", "bomUrl": "https://sapbits.blob.core.windows.net/sapbits/S4HANA_2021.yaml",
			"softwareVersion": "SAP S/4HANA 2021 ISS 00", "sapFqdn": "sap.contoso.com", "sapBitsStorageAccountId": armBase + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Storage/storageAccounts/sapbits"}}
	infraOnly := map[string]any{"configurationType": "Deployment", "appLocation": region, "infrastructureConfiguration": infrastructure}
	merge := func(extra map[string]any) map[string]any {
		out := map[string]any{}
		for k, v := range system {
			out[k] = v
		}
		for k, v := range extra {
			out[k] = v
		}
		return out
	}
	q := "?api-version=" + workloadsAPIVersion

	calls := []struct {
		name     string
		method   string
		path     string
		payload  any
		statuses []int
	}{
		{name: "CreateResourceGroup", method: http.MethodPut, path: groupPath, payload: map[string]any{"location": region}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "GetSizingRecommendations", method: http.MethodPost, path: metadata + "getSizingRecommendations" + q, payload: merge(map[string]any{"saps": 40000, "dbMemory": 512, "dbScaleMethod": "ScaleUp"}), statuses: []int{http.StatusOK}},
		{name: "GetSapSupportedSku", method: http.MethodPost, path: metadata + "getSapSupportedSku" + q, payload: system, statuses: []int{http.StatusOK}},
		{name: "GetDiskConfigurations", method: http.MethodPost, path: metadata + "getDiskConfigurations" + q, payload: merge(map[string]any{"dbVmSku": "Standard_M64ls"}), statuses: []int{http.StatusOK}},
		{name: "GetAvailabilityZoneDetails", method: http.MethodPost, path: metadata + "getAvailabilityZoneDetails" + q, payload: map[string]any{"appLocation": region, "sapProduct": "S4HANA", "databaseType": "HANA"}, statuses: []int{http.StatusOK}},
		{name: "DeployInfrastructure", method: http.MethodPut, path: vis + q, payload: map[string]any{"location": region, "properties": map[string]any{"environment": "NonProd", "sapProduct": "S4HANA", "configuration": infraOnly}}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "InstallSoftware", method: http.MethodPut, path: vis + q, payload: map[string]any{"location": region, "properties": map[string]any{"environment": "NonProd", "sapProduct": "S4HANA", "configuration": withSoftware}}, statuses: []int{http.StatusOK}},
		{name: "TagVirtualInstance", method: http.MethodPatch, path: vis + q, payload: map[string]any{"tags": map[string]any{"owner": "stackyard"}}, statuses: []int{http.StatusOK}},
		{name: "CreateCentralInstance", method: http.MethodPut, path: vis + "/centralInstances/centralServer" + q, payload: map[string]any{"location": region}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "CreateDatabaseInstance", method: http.MethodPut, path: vis + "/databaseInstances/databaseServer" + q, payload: map[string]any{"location": region}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "CreateApplicationInstance", method: http.MethodPut, path: vis + "/applicationInstances/app01" + q, payload: map[string]any{"location": region}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "ListApplicationInstances", method: http.MethodGet, path: vis + "/applicationInstances" + q, statuses: []int{http.StatusOK}},
		{name: "StopApplicationInstance", method: http.MethodPost, path: vis + "/applicationInstances/app01/stop" + q, payload: map[string]any{"softStopTimeoutSeconds": 0}, statuses: []int{http.StatusOK, http.StatusAccepted}},
		{name: "GetPartiallyRunningSystem", method: http.MethodGet, path: vis + q, statuses: []int{http.StatusOK}},
		{name: "StopSystem", method: http.MethodPost, path: vis + "/stop" + q, payload: map[string]any{"softStopTimeoutSeconds": 300}, statuses: []int{http.StatusOK, http.StatusAccepted}},
		{name: "StartSystem", method: http.MethodPost, path: vis + "/start" + q, statuses: []int{http.StatusOK, http.StatusAccepted}},
		{name: "ListVirtualInstances", method: http.MethodGet, path: base + "/providers/Microsoft.Workloads/sapVirtualInstances" + q, statuses: []int{http.StatusOK}},
		{name: "CreateMonitor", method: http.MethodPut, path: monitor + q, payload: map[string]any{"location": region, "properties": map[string]any{"monitorSubnet": subnet, "routingPreference": "RouteAll", "managedResourceGroupConfiguration": map[string]any{"name": monitorName + "-mrg"}}}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "CreateHanaProvider", method: http.MethodPut, path: monitor + "/providerInstances/hana" + q, payload: map[string]any{"properties": map[string]any{"providerSettings": map[string]any{"providerType": "SapHana", "hostname": "10.0.0.6", "dbUsername": "SYSTEM", "dbPasswordUri": "https://sap-kv.vault.azure.net/secrets/hana-system", "sapSid": sid, "instanceNumber": "00", "sslPreference": "Disabled"}}}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "CreateOSProvider", method: http.MethodPut, path: monitor + "/providerInstances/os" + q, payload: map[string]any{"properties": map[string]any{"providerSettings": map[string]any{"providerType": "PrometheusOS", "prometheusUrl": "http://10.0.0.6:9100/metrics", "sapSid": sid}}}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "ListProviders", method: http.MethodGet, path: monitor + "/providerInstances" + q, statuses: []int{http.StatusOK}},
		{name: "CreateLandscapeMonitor", method: http.MethodPut, path: monitor + "/sapLandscapeMonitor/default" + q, payload: map[string]any{"properties": map[string]any{"grouping": map[string]any{"landscape": []any{map[string]any{"name": "NonProd", "topSid": []any{sid}}}}, "topMetricsThresholds": []any{map[string]any{"name": "Instance Availability", "green": 90, "yellow": 75, "red": 50}}}}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "GetLandscapeMonitor", method: http.MethodGet, path: monitor + "/sapLandscapeMonitor/default" + q, statuses: []int{http.StatusOK}},
		{name: "DeleteLandscapeMonitor", method: http.MethodDelete, path: monitor + "/sapLandscapeMonitor/default" + q, statuses: []int{http.StatusOK, http.StatusNoContent}},
		{name: "DeleteOSProvider", method: http.MethodDelete, path: monitor + "/providerInstances/os" + q, statuses: []int{http.StatusOK, http.StatusAccepted, http.StatusNoContent}},
		{name: "DeleteMonitor", method: http.MethodDelete, path: monitor + q, statuses: []int{http.StatusOK, http.StatusAccepted, http.StatusNoContent}},
		{name: "DeleteApplicationInstance", method: http.MethodDelete, path: vis + "/applicationInstances/app01" + q, statuses: []int{http.StatusOK, http.StatusAccepted, http.StatusNoContent}},
		{name: "DeleteVirtualInstance", method: http.MethodDelete, path: vis + q, statuses: []int{http.StatusOK, http.StatusAccepted, http.StatusNoContent}},
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
