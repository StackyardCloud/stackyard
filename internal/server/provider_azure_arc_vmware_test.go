package server

import (
	"net/http"
	"testing"
)

func TestAzureArcVMwareLifecycle(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	headers := map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
		"Content-Type":              "application/json",
	}

	subscriptionID := "00000000-0000-0000-0000-000000000000"
	resourceGroup := "rg-arc-vmware"
	base := "/azure/subscriptions/" + subscriptionID
	vCenterPath := base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.ConnectedVMwarevSphere/vCenters/vcenter-a?api-version=2023-12-01"
	clusterPath := base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.ConnectedVMwarevSphere/clusters/cluster-a?api-version=2023-12-01"
	inventoryPath := base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.ConnectedVMwarevSphere/vCenters/vcenter-a/inventoryItems/item-a?api-version=2023-12-01"
	resourceURI := "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.HybridCompute/machines/machine-a"
	vmPath := "/azure" + resourceURI + "/providers/Microsoft.ConnectedVMwarevSphere/virtualMachineInstances/default?api-version=2023-12-01"
	guestAgentPath := "/azure" + resourceURI + "/providers/Microsoft.ConnectedVMwarevSphere/virtualMachineInstances/default/guestAgents/default?api-version=2023-12-01"
	hybridListPath := "/azure" + resourceURI + "/providers/Microsoft.ConnectedVMwarevSphere/virtualMachineInstances/default/hybridIdentityMetadata?api-version=2023-12-01"
	hybridGetPath := "/azure" + resourceURI + "/providers/Microsoft.ConnectedVMwarevSphere/virtualMachineInstances/default/hybridIdentityMetadata/default?api-version=2023-12-01"
	vCenterID := "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.ConnectedVMwarevSphere/vCenters/vcenter-a"

	resp := providerContractRequest(t, ts, http.MethodGet, "/azure/providers/Microsoft.ConnectedVMwarevSphere/operations?api-version=2023-12-01", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing operations, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	operations := providerContractJSONMap(t, resp)
	if values, ok := operations["value"].([]any); !ok || len(values) == 0 {
		t.Fatalf("expected operations payload to include values, got %#v", operations)
	}

	resp = providerContractRequest(t, ts, http.MethodPut, vCenterPath, []byte(`{
		"location":"eastus",
		"extendedLocation":{"type":"CustomLocation","name":"custom-eastus"},
		"tags":{"env":"dev"},
		"properties":{
			"fqdn":"vcenter-a.contoso.local",
			"port":443,
			"credentials":{"username":"administrator"}
		}
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating vcenter, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	vcenter := providerContractJSONMap(t, resp)
	if vcenter["type"] != "Microsoft.ConnectedVMwarevSphere/vCenters" {
		t.Fatalf("unexpected vcenter payload: %#v", vcenter)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, vCenterPath, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting vcenter, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodHead, vCenterPath, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 heading vcenter, got %d", resp.StatusCode)
	}

	resp = providerContractRequest(t, ts, http.MethodPut, clusterPath, []byte(`{
		"location":"eastus",
		"extendedLocation":{"type":"CustomLocation","name":"custom-eastus"},
		"properties":{"vCenterId":"`+vCenterID+`","moRefId":"domain-c7"}
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating cluster, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	cluster := providerContractJSONMap(t, resp)
	if cluster["type"] != "Microsoft.ConnectedVMwarevSphere/clusters" {
		t.Fatalf("unexpected cluster payload: %#v", cluster)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/resourceGroups/"+resourceGroup+"/providers/Microsoft.ConnectedVMwarevSphere/clusters?api-version=2023-12-01", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing clusters by resource group, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/providers/Microsoft.ConnectedVMwarevSphere/clusters?api-version=2023-12-01", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing clusters by subscription, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPut, inventoryPath, []byte(`{
		"location":"eastus",
		"kind":"VirtualMachineInventoryItem",
		"properties":{"inventoryType":"VirtualMachine","inventoryItemName":"item-a"}
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 creating inventory item, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/resourceGroups/"+resourceGroup+"/providers/Microsoft.ConnectedVMwarevSphere/vCenters/vcenter-a/inventoryItems?api-version=2023-12-01", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing inventory items, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPut, vmPath, []byte(`{
		"location":"eastus",
		"extendedLocation":{"type":"CustomLocation","name":"custom-eastus"},
		"tags":{"role":"app"},
		"properties":{
			"infrastructureProfile":{"templateId":"tmpl-a"},
			"osProfile":{"computerName":"machine-a"}
		}
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 creating vm instance, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	vmInstance := providerContractJSONMap(t, resp)
	if vmInstance["type"] != "Microsoft.ConnectedVMwarevSphere/virtualMachineInstances" {
		t.Fatalf("unexpected vm instance payload: %#v", vmInstance)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, "/azure"+resourceURI+"/providers/Microsoft.ConnectedVMwarevSphere/virtualMachineInstances?api-version=2023-12-01", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing vm instances, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	vmList := providerContractJSONMap(t, resp)
	if values, ok := vmList["value"].([]any); !ok || len(values) != 1 {
		t.Fatalf("expected one vm instance in list, got %#v", vmList)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, "/azure"+resourceURI+"/providers/Microsoft.ConnectedVMwarevSphere/virtualMachineInstances/default/start?api-version=2023-12-01", []byte(`{}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 starting vm instance, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	startResult := providerContractJSONMap(t, resp)
	if startResult["status"] != "Succeeded" {
		t.Fatalf("unexpected start payload: %#v", startResult)
	}

	resp = providerContractRequest(t, ts, http.MethodPut, guestAgentPath, []byte(`{
		"properties":{
			"credentials":{"username":"contoso\\admin"},
			"provisioningAction":"Install"
		}
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating guest agent, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, guestAgentPath, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting guest agent, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, hybridListPath, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing hybrid identity metadata, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	hybridList := providerContractJSONMap(t, resp)
	if values, ok := hybridList["value"].([]any); !ok || len(values) != 1 {
		t.Fatalf("expected one hybrid metadata item, got %#v", hybridList)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, hybridGetPath, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting hybrid identity metadata, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	hybridMetadata := providerContractJSONMap(t, resp)
	if hybridMetadata["type"] != "Microsoft.ConnectedVMwarevSphere/virtualMachineInstances/hybridIdentityMetadata" {
		t.Fatalf("unexpected hybrid metadata payload: %#v", hybridMetadata)
	}

	resp = providerContractRequest(t, ts, http.MethodPatch, vmPath, []byte(`{
		"tags":{"role":"worker","env":"stage"},
		"properties":{"hardwareProfile":{"cpuCount":4}}
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 patching vm instance, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, "/azure"+resourceURI+"/providers/Microsoft.ConnectedVMwarevSphere/virtualMachineInstances/default/restart?api-version=2023-12-01", []byte(`{}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 restarting vm instance, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, "/azure"+resourceURI+"/providers/Microsoft.ConnectedVMwarevSphere/virtualMachineInstances/default/stop?api-version=2023-12-01", []byte(`{}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 stopping vm instance, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodDelete, guestAgentPath, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 deleting guest agent, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodDelete, vmPath, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 deleting vm instance, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodDelete, inventoryPath, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 deleting inventory item, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodDelete, clusterPath, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 deleting cluster, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodDelete, vCenterPath, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 deleting vcenter, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
}

func TestAzureArcVMwareTopLevelResourcesAndValidation(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	headers := map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
		"Content-Type":              "application/json",
	}

	subscriptionID := "00000000-0000-0000-0000-000000000000"
	resourceGroup := "rg-arc-vmware-validate"
	base := "/azure/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.ConnectedVMwarevSphere"
	vCenterID := "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.ConnectedVMwarevSphere/vCenters/vcenter-b"

	resp := providerContractRequest(t, ts, http.MethodPut, base+"/vCenters/vcenter-b?api-version=2023-12-01", []byte(`{
		"location":"eastus",
		"extendedLocation":{"type":"CustomLocation","name":"custom-eastus"},
		"properties":{"fqdn":"vcenter-b.contoso.local"}
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating vcenter, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	cases := []struct {
		name       string
		path       string
		body       string
		wantStatus int
	}{
		{
			name: "host",
			path: base + "/hosts/host-a?api-version=2023-12-01",
			body: `{
				"location":"eastus",
				"properties":{"vCenterId":"` + vCenterID + `","moRefId":"host-42"}
			}`,
			wantStatus: http.StatusCreated,
		},
		{
			name: "resource pool",
			path: base + "/resourcePools/pool-a?api-version=2023-12-01",
			body: `{
				"location":"eastus",
				"properties":{"vCenterId":"` + vCenterID + `","moRefId":"resgroup-1"}
			}`,
			wantStatus: http.StatusCreated,
		},
		{
			name: "datastore",
			path: base + "/datastores/ds-a?api-version=2023-12-01",
			body: `{
				"location":"eastus",
				"properties":{"inventoryItemId":"` + vCenterID + `/inventoryItems/item-a"}
			}`,
			wantStatus: http.StatusCreated,
		},
		{
			name: "virtual network",
			path: base + "/virtualNetworks/vnet-a?api-version=2023-12-01",
			body: `{
				"location":"eastus",
				"properties":{"vCenterId":"` + vCenterID + `"}
			}`,
			wantStatus: http.StatusCreated,
		},
		{
			name: "virtual machine template",
			path: base + "/virtualMachineTemplates/tmpl-a?api-version=2023-12-01",
			body: `{
				"location":"eastus",
				"properties":{"inventoryItemId":"` + vCenterID + `/inventoryItems/item-a"}
			}`,
			wantStatus: http.StatusCreated,
		},
	}

	for _, tc := range cases {
		resp := providerContractRequest(t, ts, http.MethodPut, tc.path, []byte(tc.body), headers)
		if resp.StatusCode != tc.wantStatus {
			t.Fatalf("expected %d creating %s, got %d body=%s", tc.wantStatus, tc.name, resp.StatusCode, string(providerContractBody(t, resp)))
		}
		payload := providerContractJSONMap(t, resp)
		if payload["type"] == nil {
			t.Fatalf("expected typed payload for %s, got %#v", tc.name, payload)
		}
		resp = providerContractRequest(t, ts, http.MethodPatch, tc.path, []byte(`{"tags":{"env":"test"}}`), headers)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 patching %s, got %d body=%s", tc.name, resp.StatusCode, string(providerContractBody(t, resp)))
		}
		resp = providerContractRequest(t, ts, http.MethodGet, tc.path, nil, headers)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 getting %s, got %d body=%s", tc.name, resp.StatusCode, string(providerContractBody(t, resp)))
		}
	}

	resp = providerContractRequest(t, ts, http.MethodPut, base+"/vCenters/vcenter-invalid?api-version=2023-12-01", []byte(`{
		"location":"eastus",
		"properties":{"fqdn":"vcenter-invalid.contoso.local"}
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 for vcenter without extendedLocation, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPut, base+"/clusters/cluster-invalid?api-version=2023-12-01", []byte(`{
		"location":"eastus",
		"properties":{}
	}`), headers)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing vCenterId or inventoryItemId, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPut, base+"/vCenters/vcenter-json?api-version=2023-12-01", []byte(`{"location":`), headers)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid JSON, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
}

func TestAzureArcVMwareNotFoundAndInvalidAPIVersion(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	headers := map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
		"Content-Type":              "application/json",
	}

	resourceURI := "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-missing/providers/Microsoft.HybridCompute/machines/missing"

	resp := providerContractRequest(t, ts, http.MethodGet, "/azure/providers/Microsoft.ConnectedVMwarevSphere/operations?api-version=", nil, headers)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid api-version, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-missing/providers/Microsoft.ConnectedVMwarevSphere/vCenters/nope?api-version=2023-12-01", nil, headers)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for missing vcenter, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPut, "/azure"+resourceURI+"/providers/Microsoft.ConnectedVMwarevSphere/virtualMachineInstances/default/guestAgents/default?api-version=2023-12-01", []byte(`{
		"properties":{"credentials":{"username":"admin"}}
	}`), headers)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 creating guest agent without vm instance, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
}
