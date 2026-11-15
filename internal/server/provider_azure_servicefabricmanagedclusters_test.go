package server

import (
	"net/http"
	"testing"
)

const azureSFManagedClusterTestAPIVersion = "?api-version=2024-04-01"

func TestAzureSFManagedClusterLifecycle(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-sf")
	cluster := sub + "/resourceGroups/rg-sf/providers/Microsoft.ServiceFabric/managedClusters/apps"

	resp := do(http.MethodPut, cluster+azureSFManagedClusterTestAPIVersion, `{"location":"eastus","sku":{"name":"Standard"},"properties":{"dnsName":"Stackyard-Apps","adminUserName":"vmadmin","adminPassword":"Secret-123","clusterState":"Upgrading"}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	props := providerContractJSONMap(t, resp)["properties"].(map[string]any)
	if props["adminPassword"] != nil || props["clusterState"] != "Ready" || props["fqdn"] != "stackyard-apps.eastus.cloudapp.azure.com" || props["clientConnectionPort"] != float64(19000) {
		t.Fatalf("unexpected cluster properties: %#v", props)
	}

	nodeType := cluster + "/nodeTypes/primary"
	resp = do(http.MethodPut, nodeType+azureSFManagedClusterTestAPIVersion, `{"properties":{"isPrimary":true,"vmInstanceCount":5,"dataDiskSizeGB":100,"vmImagePublisher":"MicrosoftWindowsServer","vmImageOffer":"WindowsServer","vmImageSku":"2022-Datacenter","vmImageVersion":"latest"}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	if props := providerContractJSONMap(t, resp)["properties"].(map[string]any); props["vmSize"] != "Standard_D2s_v3" || props["dataDiskSizeGB"] != float64(100) {
		t.Fatalf("unexpected node type properties: %#v", props)
	}
	azureResourcesExpectStatus(t, do(http.MethodPost, nodeType+"/restart"+azureSFManagedClusterTestAPIVersion, `{"nodes":["primary_0"]}`), http.StatusAccepted)
	azureResourcesExpectErrorCode(t, do(http.MethodPost, nodeType+"/deleteNode"+azureSFManagedClusterTestAPIVersion, `{}`), http.StatusBadRequest, "InvalidRequestContent")

	azureResourcesExpectStatus(t, do(http.MethodDelete, cluster+azureSFManagedClusterTestAPIVersion, ""), http.StatusOK)
	azureResourcesExpectErrorCode(t, do(http.MethodGet, nodeType+azureSFManagedClusterTestAPIVersion, ""), http.StatusNotFound, "ParentResourceNotFound")
}

func TestAzureSFManagedClusterValidation(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-sf")
	cluster := sub + "/resourceGroups/rg-sf/providers/Microsoft.ServiceFabric/managedClusters/apps"
	azureResourcesExpectStatus(t, do(http.MethodPut, cluster+azureSFManagedClusterTestAPIVersion, `{"location":"eastus","sku":{"name":"Basic"},"properties":{"dnsName":"apps","adminUserName":"vmadmin"}}`), http.StatusCreated)
	cases := []struct{ name, path, body string }{
		{"bad sku", cluster, `{"location":"eastus","sku":{"name":"Premium"},"properties":{"dnsName":"apps","adminUserName":"vmadmin"}}`},
		{"bad dns name", cluster, `{"location":"eastus","sku":{"name":"Basic"},"properties":{"dnsName":"apps_1","adminUserName":"vmadmin"}}`},
		{"no admin", cluster, `{"location":"eastus","sku":{"name":"Basic"},"properties":{"dnsName":"apps"}}`},
		{"bad port", cluster, `{"location":"eastus","sku":{"name":"Basic"},"properties":{"dnsName":"apps","adminUserName":"vmadmin","clientConnectionPort":0}}`},
		{"node type primary", cluster + "/nodeTypes/nt", `{"properties":{"vmInstanceCount":3}}`},
		{"node type count", cluster + "/nodeTypes/nt", `{"properties":{"isPrimary":false,"vmInstanceCount":-2}}`},
		{"node type disk type", cluster + "/nodeTypes/nt", `{"properties":{"isPrimary":false,"vmInstanceCount":3,"dataDiskType":"Ultra"}}`},
	}
	for _, tc := range cases {
		resp := do(http.MethodPut, tc.path+azureSFManagedClusterTestAPIVersion, tc.body)
		t.Run(tc.name, func(t *testing.T) {
			azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, "InvalidRequestContent")
		})
	}
}
