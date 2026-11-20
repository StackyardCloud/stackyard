package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
)

const (
	resourcesAPIVersion    = "2021-04-01"
	networkCloudAPIVersion = "2026-07-01"
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
		"networkcloud-2026-07-01",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&sharedKeyAndSubscriptionPolicy{account: account, subscriptionKey: subscriptionKey}},
		},
		&policy.ClientOptions{},
	)
	return &armClient{endpoint: strings.TrimRight(endpoint, "/"), pipeline: pipeline}
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
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("read response body %s %s: %w", method, path, err)
	}
	ok := len(expectedStatuses) == 0 && resp.StatusCode >= 200 && resp.StatusCode < 300
	for _, status := range expectedStatuses {
		ok = ok || resp.StatusCode == status
	}
	if !ok {
		return nil, resp.StatusCode, fmt.Errorf("unexpected status %d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	decoded := map[string]any{}
	if strings.TrimSpace(string(body)) != "" {
		if err := json.Unmarshal(body, &decoded); err != nil {
			return nil, resp.StatusCode, fmt.Errorf("decode JSON body for %s %s: %w", method, path, err)
		}
	}
	return decoded, resp.StatusCode, nil
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	account := getenv("STACKYARD_AZURE_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	resourceGroup := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-networkcloud-example")
	region := getenv("STACKYARD_AZURE_REGION", "eastus")

	fmt.Printf("Stackyard Azure Operator Nexus (networkcloud-2026-07-01) example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newARMClient(endpoint, account, subscriptionKey)
	base := "/azure/subscriptions/" + subscriptionID
	groupPath := base + "/resourcegroups/" + resourceGroup + "?api-version=" + resourcesAPIVersion
	providers := base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.NetworkCloud/"
	fabric := "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.ManagedNetworkFabric/"
	manager := providers + "clusterManagers/stackyard-cm"
	cluster := providers + "clusters/stackyard-cluster"
	q := "?api-version=" + networkCloudAPIVersion

	run := func(name, method, path string, payload any, statuses ...int) map[string]any {
		resp, status, err := client.doJSON(ctx, method, path, payload, statuses...)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s failed: %v\n", name, err)
			os.Exit(1)
		}
		fmt.Printf("%s -> HTTP %d\n", name, status)
		if len(resp) != 0 {
			encoded, _ := json.MarshalIndent(resp, "", "  ")
			fmt.Println(string(encoded))
		}
		return resp
	}
	props := func(resp map[string]any) map[string]any { p, _ := resp["properties"].(map[string]any); return p }
	created := []int{http.StatusCreated, http.StatusOK}
	deleted := []int{http.StatusOK, http.StatusNoContent}

	run("CreateResourceGroup", http.MethodPut, groupPath, map[string]any{"location": region}, created...)
	run("ListOperations", http.MethodGet, "/azure/providers/Microsoft.NetworkCloud/operations"+q, nil, http.StatusOK)
	run("ListRackSkus", http.MethodGet, base+"/providers/Microsoft.NetworkCloud/rackSkus"+q, nil, http.StatusOK)
	sku := "/subscriptions/" + subscriptionID + "/providers/Microsoft.NetworkCloud/rackSkus/NearEdge_Single_R750"

	cm := run("CreateClusterManager", http.MethodPut, manager+q, map[string]any{"location": region, "identity": map[string]any{"type": "SystemAssigned"}, "properties": map[string]any{
		"fabricControllerId": fabric + "networkFabricControllers/stackyard-nfc", "vmSize": "Standard_D8s_v3",
	}}, created...)
	managerLocation := props(cm)["managerExtendedLocation"]
	run("TagClusterManager", http.MethodPatch, manager+q, map[string]any{"tags": map[string]any{"env": "dev"}}, http.StatusOK)

	machine := func(name string, slot int) map[string]any {
		return map[string]any{
			"machineName": name, "rackSlot": slot, "serialNumber": "SN-" + name, "machineDetails": "PowerEdge R750",
			"bmcMacAddress": fmt.Sprintf("AA:BB:CC:DD:EE:%02d", slot), "bootMacAddress": fmt.Sprintf("00:BB:CC:DD:EE:%02d", slot),
			"bmcCredentials": map[string]any{"username": "root", "password": "Stackyard-Bmc-2026"},
		}
	}
	c := run("CreateCluster", http.MethodPut, cluster+q, map[string]any{"location": region, "extendedLocation": managerLocation, "properties": map[string]any{
		"clusterType": "SingleRack", "clusterVersion": "4.1.0", "networkFabricId": fabric + "networkFabrics/stackyard-fabric",
		"aggregatorOrSingleRackDefinition": map[string]any{
			"networkRackId": fabric + "networkRacks/stackyard-rack", "rackSkuId": sku, "rackSerialNumber": "RACK-0001", "rackLocation": "Lab 1",
			"bareMetalMachineConfigurationData": []any{machine("controller1", 1), machine("compute1", 3)},
			"storageApplianceConfigurationData": []any{map[string]any{
				"rackSlot": 11, "serialNumber": "SA-0001", "storageApplianceName": "stackyard-sa",
				"adminCredentials": map[string]any{"username": "pureuser", "password": "Stackyard-Storage-2026"},
			}},
		},
		"clusterServicePrincipal": map[string]any{"applicationId": "app", "password": "Stackyard-Sp-2026", "principalId": "principal", "tenantId": "tenant"},
		"commandOutputSettings":   map[string]any{"containerUrl": "https://stackyardoutput.blob.core.windows.net/nexus"},
	}}, created...)
	clusterLocation := props(c)["clusterExtendedLocation"]
	run("ListRacks", http.MethodGet, providers+"racks"+q, nil, http.StatusOK)
	run("DeployCluster", http.MethodPost, cluster+"/deploy"+q, map[string]any{}, http.StatusOK)
	run("GetCluster", http.MethodGet, cluster+q, nil, http.StatusOK)

	bmm := providers + "bareMetalMachines/compute1"
	run("CordonMachine", http.MethodPost, bmm+"/cordon"+q, map[string]any{"evacuate": "True"}, http.StatusOK)
	run("RunReadCommands", http.MethodPost, bmm+"/runReadCommands"+q, map[string]any{"commands": []any{map[string]any{"command": "hostname"}}, "limitTimeSeconds": 60}, http.StatusOK)
	run("UncordonMachine", http.MethodPost, bmm+"/uncordon"+q, map[string]any{}, http.StatusOK)
	run("PatchMachine", http.MethodPatch, bmm+q, map[string]any{"properties": map[string]any{"machineDetails": "rack slot 3"}}, http.StatusOK)
	run("EnableVendorManagement", http.MethodPost, providers+"storageAppliances/stackyard-sa/enableRemoteVendorManagement"+q, map[string]any{"supportEndpoints": []any{"10.0.0.0/24"}}, http.StatusOK)

	expiry := time.Now().Add(30 * 24 * time.Hour).UTC().Format(time.RFC3339)
	user := map[string]any{"azureUserName": "operator", "sshPublicKey": map[string]any{"keyData": "ssh-ed25519 AAAAC3Nza stackyard"}}
	run("CreateBareMetalMachineKeySet", http.MethodPut, cluster+"/bareMetalMachineKeySets/operators"+q, map[string]any{"location": region, "extendedLocation": clusterLocation, "properties": map[string]any{
		"azureGroupId": "operators", "expiration": expiry, "privilegeLevel": "Standard", "jumpHostsAllowed": []any{"10.0.0.4"}, "userList": []any{user},
	}}, created...)
	run("CreateBmcKeySet", http.MethodPut, cluster+"/bmcKeySets/operators"+q, map[string]any{"location": region, "extendedLocation": clusterLocation, "properties": map[string]any{
		"azureGroupId": "operators", "expiration": expiry, "privilegeLevel": "ReadOnly", "userList": []any{user},
	}}, created...)
	run("CreateMetricsConfiguration", http.MethodPut, cluster+"/metricsConfigurations/default"+q, map[string]any{"location": region, "extendedLocation": clusterLocation, "properties": map[string]any{
		"collectionInterval": 15, "enabledMetrics": []any{"node_cpu_frequency_hertz"},
	}}, created...)
	run("CreateKubernetesVersions", http.MethodPut, providers+"kubernetesVersions/default"+q, map[string]any{"location": region, "extendedLocation": clusterLocation}, created...)
	run("CreateAccessBridge", http.MethodPut, providers+"accessBridges/stackyard-bridge"+q, map[string]any{"location": region, "extendedLocation": clusterLocation, "properties": map[string]any{
		"networkId": fabric + "l3IsolationDomains/l3/internalNetworks/mgmt", "ipv4ConnectedPrefix": "198.51.100.0/24",
		"securityRules": []any{map[string]any{"direction": "Inbound", "port": "22", "ipv4Addresses": []any{"10.0.0.0/8"}}},
	}}, created...)

	if err := runWorkloads(ctx, client); err != nil {
		fmt.Fprintf(os.Stderr, "workloads failed: %v\n", err)
		os.Exit(1)
	}

	run("UpdateClusterVersion", http.MethodPost, cluster+"/updateVersion"+q, map[string]any{"targetClusterVersion": "4.2.0"}, http.StatusOK)
	run("RotateCredential", http.MethodPost, cluster+"/rotateCredential"+q, map[string]any{"credentials": []any{"BMC Credential"}}, http.StatusOK)
	run("DeleteAccessBridge", http.MethodDelete, providers+"accessBridges/stackyard-bridge"+q, nil, deleted...)
	run("DeleteKubernetesVersions", http.MethodDelete, providers+"kubernetesVersions/default"+q, nil, deleted...)
	run("DeleteCluster", http.MethodDelete, cluster+q, nil, deleted...)
	run("DeleteClusterManager", http.MethodDelete, manager+q, nil, deleted...)
	run("DeleteResourceGroup", http.MethodDelete, groupPath, nil, http.StatusOK)
}

func getenv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
