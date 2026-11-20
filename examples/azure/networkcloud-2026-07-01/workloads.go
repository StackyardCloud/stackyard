package main

import (
	"context"
	"fmt"
	"net/http"
)

// runWorkloads exercises the Network Cloud tenant workloads: networks, a
// volume, a virtual machine with its console and power actions, and a
// Kubernetes cluster with an agent pool and a feature. The resource group
// must exist; everything created here is deleted again.
func runWorkloads(ctx context.Context, c *armClient) error {
	sub := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	group := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-networkcloud-example")
	region := getenv("STACKYARD_AZURE_REGION", "eastus")
	arm := "/subscriptions/" + sub + "/resourceGroups/" + group + "/providers/Microsoft.NetworkCloud/"
	base := "/azure" + arm
	const q = "?api-version=2026-07-01"
	location := map[string]any{"type": "CustomLocation", "name": "/subscriptions/" + sub + "/resourceGroups/" + group + "/providers/Microsoft.ExtendedLocation/customLocations/nexus-cl"}
	body := func(props map[string]any) map[string]any {
		out := map[string]any{"location": region, "extendedLocation": location}
		if props != nil {
			out["properties"] = props
		}
		return out
	}
	domain := "/subscriptions/" + sub + "/resourceGroups/" + group + "/providers/Microsoft.ManagedNetworkFabric/l3IsolationDomains/l3d"
	created := []int{http.StatusOK, http.StatusCreated}

	steps := []struct {
		name, method, path string
		payload            any
		statuses           []int
	}{
		{"CreateCloudServicesNetwork", http.MethodPut, "cloudServicesNetworks/csn", body(map[string]any{"enableDefaultEgressEndpoints": "True"}), created},
		{"CreateL2Network", http.MethodPut, "l2Networks/l2", body(map[string]any{"l2IsolationDomainId": domain}), created},
		{"CreateL3Network", http.MethodPut, "l3Networks/l3", body(map[string]any{"l3IsolationDomainId": domain, "vlan": 100, "ipAllocationType": "IPV4", "ipv4ConnectedPrefix": "198.51.100.0/24"}), created},
		{"CreateTrunkedNetwork", http.MethodPut, "trunkedNetworks/trunk", body(map[string]any{"isolationDomainIds": []any{domain}, "vlans": []any{10, 11}}), created},
		{"CreateVolume", http.MethodPut, "volumes/data", body(map[string]any{"sizeMiB": 10240}), created},
		{"CreateVirtualMachine", http.MethodPut, "virtualMachines/vm1", body(map[string]any{
			"adminUsername":                  "azureuser",
			"cloudServicesNetworkAttachment": map[string]any{"attachedNetworkId": arm + "cloudServicesNetworks/csn", "ipAllocationMethod": "Dynamic"},
			"networkAttachments":             []any{map[string]any{"attachedNetworkId": arm + "l3Networks/l3", "ipAllocationMethod": "Static", "ipv4Address": "198.51.100.10", "defaultGateway": "True"}},
			"cpuCores":                       2, "memorySizeGB": 8, "vmImage": "registry.example.com/vm:1",
			"storageProfile": map[string]any{"osDisk": map[string]any{"diskSizeGB": 120}, "volumeAttachments": []any{arm + "volumes/data"}},
		}), created},
		{"CreateConsole", http.MethodPut, "virtualMachines/vm1/consoles/default", body(map[string]any{"enabled": "True", "sshPublicKey": map[string]any{"keyData": "ssh-rsa AAAAexample"}}), created},
		{"PowerOffVirtualMachine", http.MethodPost, "virtualMachines/vm1/powerOff", map[string]any{"skipShutdown": "False"}, []int{http.StatusAccepted}},
		{"StartVirtualMachine", http.MethodPost, "virtualMachines/vm1/start", nil, []int{http.StatusAccepted}},
		{"CreateKubernetesCluster", http.MethodPut, "kubernetesClusters/k8s1", body(map[string]any{
			"kubernetesVersion":             "1.32.7",
			"controlPlaneNodeConfiguration": map[string]any{"count": 1, "vmSkuName": "NC_G6_28_v1"},
			"initialAgentPoolConfigurations": []any{map[string]any{
				"name": "system", "count": 1, "mode": "System", "vmSkuName": "NC_P46_224_v1",
				"attachedNetworkConfiguration": map[string]any{"l2Networks": []any{map[string]any{"networkId": arm + "l2Networks/l2"}}},
			}},
			"networkConfiguration": map[string]any{"cloudServicesNetworkId": arm + "cloudServicesNetworks/csn", "cniNetworkId": arm + "l3Networks/l3"},
		}), created},
		{"CreateAgentPool", http.MethodPut, "kubernetesClusters/k8s1/agentPools/user", map[string]any{"location": region, "properties": map[string]any{"count": 2, "mode": "User", "vmSkuName": "NC_P46_224_v1"}}, created},
		{"CreateFeature", http.MethodPut, "kubernetesClusters/k8s1/features/metrics-server", map[string]any{"location": region, "properties": map[string]any{}}, created},
		{"UpgradeKubernetesCluster", http.MethodPatch, "kubernetesClusters/k8s1", map[string]any{"properties": map[string]any{"kubernetesVersion": "1.33.3"}}, []int{http.StatusOK}},
		{"RestartNode", http.MethodPost, "kubernetesClusters/k8s1/restartNode", map[string]any{"nodeName": "k8s1-control-plane-1"}, []int{http.StatusAccepted}},
		{"DeleteNetworkInUse", http.MethodDelete, "l2Networks/l2", nil, []int{http.StatusConflict}},
		{"DeleteKubernetesCluster", http.MethodDelete, "kubernetesClusters/k8s1", nil, []int{http.StatusOK, http.StatusNoContent}},
		{"DeleteVirtualMachine", http.MethodDelete, "virtualMachines/vm1", nil, []int{http.StatusOK, http.StatusNoContent}},
		{"DeleteVolume", http.MethodDelete, "volumes/data", nil, []int{http.StatusOK, http.StatusNoContent}},
		{"DeleteTrunkedNetwork", http.MethodDelete, "trunkedNetworks/trunk", nil, []int{http.StatusOK, http.StatusNoContent}},
		{"DeleteL3Network", http.MethodDelete, "l3Networks/l3", nil, []int{http.StatusOK, http.StatusNoContent}},
		{"DeleteL2Network", http.MethodDelete, "l2Networks/l2", nil, []int{http.StatusOK, http.StatusNoContent}},
		{"DeleteCloudServicesNetwork", http.MethodDelete, "cloudServicesNetworks/csn", nil, []int{http.StatusOK, http.StatusNoContent}},
	}
	for _, step := range steps {
		resp, status, err := c.doJSON(ctx, step.method, base+step.path+q, step.payload, step.statuses...)
		if err != nil {
			return fmt.Errorf("%s: %w", step.name, err)
		}
		props, _ := resp["properties"].(map[string]any)
		fmt.Printf("%s -> %d %v\n", step.name, status, props["detailedStatus"])
	}
	return nil
}
