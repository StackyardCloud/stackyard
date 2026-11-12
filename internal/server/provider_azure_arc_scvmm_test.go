package server

import (
	"net/http"
	"testing"
)

func TestAzureArcSCVMMLifecycle(t *testing.T) {
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
	resourceGroup := "rg-arc-scvmm"
	base := "/azure/subscriptions/" + subscriptionID
	vmmServerPath := base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.ScVmm/vmmServers/vmm-a?api-version=2025-03-13"
	cloudPath := base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.ScVmm/clouds/cloud-a?api-version=2025-03-13"
	inventoryPath := base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.ScVmm/vmmServers/vmm-a/inventoryItems/item-a?api-version=2025-03-13"
	resourceURI := "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.HybridCompute/machines/machine-a"
	vmPath := "/azure" + resourceURI + "/providers/Microsoft.ScVmm/virtualMachineInstances/default?api-version=2025-03-13"
	guestAgentPath := "/azure" + resourceURI + "/providers/Microsoft.ScVmm/virtualMachineInstances/default/guestAgents/default?api-version=2025-03-13"
	hybridMetadataListPath := "/azure" + resourceURI + "/providers/Microsoft.ScVmm/virtualMachineInstances/default/hybridIdentityMetadata?api-version=2025-03-13"
	hybridMetadataGetPath := "/azure" + resourceURI + "/providers/Microsoft.ScVmm/virtualMachineInstances/default/hybridIdentityMetadata/default?api-version=2025-03-13"

	resp := providerContractRequest(t, ts, http.MethodGet, "/azure/providers/Microsoft.ScVmm/operations?api-version=2025-03-13", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing operations, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	operations := providerContractJSONMap(t, resp)
	if values, ok := operations["value"].([]any); !ok || len(values) == 0 {
		t.Fatalf("expected operations payload to include values, got %#v", operations)
	}

	resp = providerContractRequest(t, ts, http.MethodPut, vmmServerPath, []byte(`{
		"location":"eastus",
		"extendedLocation":{"type":"CustomLocation","name":"custom-eastus"},
		"tags":{"env":"dev"},
		"properties":{
			"fqdn":"vmm-a.contoso.local",
			"port":8100,
			"credentials":{"username":"admin"}
		}
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating vmm server, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	vmmServer := providerContractJSONMap(t, resp)
	if vmmServer["type"] != "Microsoft.ScVmm/vmmServers" {
		t.Fatalf("unexpected vmm server payload: %#v", vmmServer)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, vmmServerPath, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting vmm server, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodHead, vmmServerPath, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 heading vmm server, got %d", resp.StatusCode)
	}

	resp = providerContractRequest(t, ts, http.MethodPut, cloudPath, []byte(`{
		"location":"eastus",
		"extendedLocation":{"type":"CustomLocation","name":"custom-eastus"},
		"properties":{"vmmServerId":"/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-arc-scvmm/providers/Microsoft.ScVmm/vmmServers/vmm-a"}
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating cloud, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	cloud := providerContractJSONMap(t, resp)
	if cloud["type"] != "Microsoft.ScVmm/clouds" {
		t.Fatalf("unexpected cloud payload: %#v", cloud)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/resourceGroups/"+resourceGroup+"/providers/Microsoft.ScVmm/clouds?api-version=2025-03-13", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing clouds by resource group, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/providers/Microsoft.ScVmm/clouds?api-version=2025-03-13", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing clouds by subscription, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPut, inventoryPath, []byte(`{
		"location":"eastus",
		"kind":"CloudInventoryItem",
		"properties":{"inventoryType":"Cloud","inventoryItemName":"item-a"}
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 creating inventory item, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/resourceGroups/"+resourceGroup+"/providers/Microsoft.ScVmm/vmmServers/vmm-a/inventoryItems?api-version=2025-03-13", nil, headers)
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
	if vmInstance["type"] != "Microsoft.ScVmm/virtualMachineInstances" {
		t.Fatalf("unexpected vm instance payload: %#v", vmInstance)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, "/azure"+resourceURI+"/providers/Microsoft.ScVmm/virtualMachineInstances?api-version=2025-03-13", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing vm instances, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	vmList := providerContractJSONMap(t, resp)
	if values, ok := vmList["value"].([]any); !ok || len(values) != 1 {
		t.Fatalf("expected one vm instance in list, got %#v", vmList)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, "/azure"+resourceURI+"/providers/Microsoft.ScVmm/virtualMachineInstances/default/start?api-version=2025-03-13", []byte(`{}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 starting vm instance, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	startResult := providerContractJSONMap(t, resp)
	if startResult["status"] != "Succeeded" {
		t.Fatalf("unexpected start payload: %#v", startResult)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, "/azure"+resourceURI+"/providers/Microsoft.ScVmm/virtualMachineInstances/default/createCheckpoint?api-version=2025-03-13", []byte(`{"checkpointName":"cp-a"}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 creating checkpoint, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, "/azure"+resourceURI+"/providers/Microsoft.ScVmm/virtualMachineInstances/default/restoreCheckpoint?api-version=2025-03-13", []byte(`{"checkpointName":"cp-a"}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 restoring checkpoint, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
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

	resp = providerContractRequest(t, ts, http.MethodGet, hybridMetadataListPath, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing hybrid identity metadata, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	hybridList := providerContractJSONMap(t, resp)
	if values, ok := hybridList["value"].([]any); !ok || len(values) != 1 {
		t.Fatalf("expected one hybrid metadata item, got %#v", hybridList)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, hybridMetadataGetPath, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting hybrid identity metadata, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	hybridMetadata := providerContractJSONMap(t, resp)
	if hybridMetadata["type"] != "Microsoft.ScVmm/virtualMachineInstances/hybridIdentityMetadata" {
		t.Fatalf("unexpected hybrid metadata payload: %#v", hybridMetadata)
	}

	resp = providerContractRequest(t, ts, http.MethodPatch, vmPath, []byte(`{
		"tags":{"role":"worker","env":"stage"},
		"properties":{"hardwareProfile":{"cpuCount":4}}
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 patching vm instance, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, "/azure"+resourceURI+"/providers/Microsoft.ScVmm/virtualMachineInstances/default/stop?api-version=2025-03-13", []byte(`{"skipShutdown":true}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 stopping vm instance, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, "/azure"+resourceURI+"/providers/Microsoft.ScVmm/virtualMachineInstances/default/deleteCheckpoint?api-version=2025-03-13", []byte(`{"checkpointName":"cp-a"}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 deleting checkpoint, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
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

	resp = providerContractRequest(t, ts, http.MethodDelete, cloudPath, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 deleting cloud, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodDelete, vmmServerPath, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 deleting vmm server, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
}

func TestAzureArcSCVMMTopLevelResourcesAndValidation(t *testing.T) {
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
	resourceGroup := "rg-arc-scvmm-validate"
	base := "/azure/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.ScVmm"
	vmmServerID := "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.ScVmm/vmmServers/vmm-b"

	resp := providerContractRequest(t, ts, http.MethodPut, base+"/vmmServers/vmm-b?api-version=2025-03-13", []byte(`{
		"location":"eastus",
		"extendedLocation":{"type":"CustomLocation","name":"custom-eastus"},
		"properties":{"fqdn":"vmm-b.contoso.local"}
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating vmm server, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	cases := []struct {
		name       string
		path       string
		body       string
		wantStatus int
	}{
		{
			name: "availability set",
			path: base + "/availabilitySets/as-a?api-version=2025-03-13",
			body: `{
				"location":"eastus",
				"extendedLocation":{"type":"CustomLocation","name":"custom-eastus"},
				"properties":{"platformUpdateDomainCount":5}
			}`,
			wantStatus: http.StatusCreated,
		},
		{
			name: "virtual network",
			path: base + "/virtualNetworks/vnet-a?api-version=2025-03-13",
			body: `{
				"location":"eastus",
				"extendedLocation":{"type":"CustomLocation","name":"custom-eastus"},
				"properties":{"vmmServerId":"` + vmmServerID + `"}
			}`,
			wantStatus: http.StatusCreated,
		},
		{
			name: "virtual machine template",
			path: base + "/virtualMachineTemplates/tmpl-a?api-version=2025-03-13",
			body: `{
				"location":"eastus",
				"extendedLocation":{"type":"CustomLocation","name":"custom-eastus"},
				"properties":{"vmmServerId":"` + vmmServerID + `"}
			}`,
			wantStatus: http.StatusCreated,
		},
	}

	for _, tc := range cases {
		resp := providerContractRequest(t, ts, http.MethodPut, tc.path, []byte(tc.body), headers)
		if resp.StatusCode != tc.wantStatus {
			t.Fatalf("expected %d creating %s, got %d body=%s", tc.wantStatus, tc.name, resp.StatusCode, string(providerContractBody(t, resp)))
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

	resp = providerContractRequest(t, ts, http.MethodPut, base+"/vmmServers/vmm-invalid?api-version=2025-03-13", []byte(`{
		"location":"eastus",
		"properties":{"fqdn":"vmm-invalid.contoso.local"}
	}`), headers)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing extendedLocation, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPut, base+"/clouds/cloud-invalid?api-version=2025-03-13", []byte(`{
		"location":"eastus",
		"extendedLocation":{"type":"CustomLocation","name":"custom-eastus"},
		"properties":{}
	}`), headers)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing vmmServerId, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPut, base+"/vmmServers/vmm-json?api-version=2025-03-13", []byte(`{"location":`), headers)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid JSON, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
}

func TestAzureArcSCVMMNotFoundAndInvalidAPIVersion(t *testing.T) {
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

	resp := providerContractRequest(t, ts, http.MethodGet, "/azure/providers/Microsoft.ScVmm/operations?api-version=", nil, headers)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid api-version, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-missing/providers/Microsoft.ScVmm/vmmServers/nope?api-version=2025-03-13", nil, headers)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for missing vmm server, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPut, "/azure"+resourceURI+"/providers/Microsoft.ScVmm/virtualMachineInstances/default/guestAgents/default?api-version=2025-03-13", []byte(`{
		"properties":{"credentials":{"username":"admin"}}
	}`), headers)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 creating guest agent without vm instance, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
}
