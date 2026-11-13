package server

import (
	"net/http"
	"testing"
)

const azureSiteRecoveryTestAPIVersion = "?api-version=2024-10-01"

func TestAzureSiteRecoveryLifecycle(t *testing.T) {
	t.Parallel()
	vault, do := newAzureRecoveryServicesVault(t)
	policy := vault + "/replicationPolicies/a2a-24h"
	fabric := vault + "/replicationFabrics/asr-eastus"
	container := fabric + "/replicationProtectionContainers/eastus-container"
	mapping := container + "/replicationProtectionContainerMappings/eastus-to-westus"

	resp := do(http.MethodPut, policy+azureSiteRecoveryTestAPIVersion, `{"properties":{"providerSpecificInput":{"instanceType":"A2A","recoveryPointHistory":1440,"appConsistentFrequencyInMinutes":240,"multiVmSyncStatus":"Enable"}}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	props := providerContractJSONMap(t, resp)["properties"].(map[string]any)
	if details, _ := props["providerSpecificDetails"].(map[string]any); details["instanceType"] != "A2A" || props["providerSpecificInput"] != nil || props["friendlyName"] != "a2a-24h" {
		t.Fatalf("unexpected policy: %#v", props)
	}
	// A PATCH without input keeps the details.
	resp = do(http.MethodPatch, policy+azureSiteRecoveryTestAPIVersion, `{"properties":{}}`)
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if details, _ := providerContractJSONMap(t, resp)["properties"].(map[string]any)["providerSpecificDetails"].(map[string]any); details["recoveryPointHistory"] != float64(1440) {
		t.Fatalf("PATCH lost the policy details: %#v", details)
	}

	resp = do(http.MethodPut, fabric+azureSiteRecoveryTestAPIVersion, `{"properties":{"customDetails":{"instanceType":"Azure","location":"eastus"}}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	if props := providerContractJSONMap(t, resp)["properties"].(map[string]any); props["fabricType"] != "Azure" || props["health"] != "Normal" || len(props["internalIdentifier"].(string)) != 32 {
		t.Fatalf("unexpected fabric: %#v", props)
	}
	azureResourcesExpectStatus(t, do(http.MethodPut, container+azureSiteRecoveryTestAPIVersion, `{"properties":{"providerSpecificInput":[{"instanceType":"A2A"}]}}`), http.StatusCreated)
	resp = do(http.MethodPut, mapping+azureSiteRecoveryTestAPIVersion, `{"properties":{"policyId":"`+policy[len("/azure"):]+`","targetProtectionContainerId":"Microsoft Azure","providerSpecificInput":{"instanceType":"A2A","agentAutoUpdateStatus":"Disabled"}}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	if props := providerContractJSONMap(t, resp)["properties"].(map[string]any); props["state"] != "Paired" || props["providerSpecificDetails"].(map[string]any)["instanceType"] != "A2A" {
		t.Fatalf("unexpected mapping: %#v", props)
	}

	resp = do(http.MethodGet, vault+"/replicationFabrics"+azureSiteRecoveryTestAPIVersion, "")
	if items, _ := providerContractJSONMap(t, resp)["value"].([]any); len(items) != 1 {
		t.Fatalf("expected one fabric, got %#v", items)
	}
	azureResourcesExpectStatus(t, do(http.MethodDelete, fabric+azureSiteRecoveryTestAPIVersion, ""), http.StatusOK)
	azureResourcesExpectErrorCode(t, do(http.MethodGet, mapping+azureSiteRecoveryTestAPIVersion, ""), http.StatusNotFound, "ParentResourceNotFound")
}

func TestAzureSiteRecoveryValidation(t *testing.T) {
	t.Parallel()
	vault, do := newAzureRecoveryServicesVault(t)
	fabric := vault + "/replicationFabrics/f1"
	cases := []struct{ name, path, body string }{
		{"policy without instance type", vault + "/replicationPolicies/p1", `{"properties":{"providerSpecificInput":{}}}`},
		{"unknown fabric type", fabric, `{"properties":{"customDetails":{"instanceType":"Mainframe"}}}`},
		{"azure fabric without location", fabric, `{"properties":{"customDetails":{"instanceType":"Azure"}}}`},
	}
	for _, tc := range cases {
		resp := do(http.MethodPut, tc.path+azureSiteRecoveryTestAPIVersion, tc.body)
		t.Run(tc.name, func(t *testing.T) {
			azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, "InvalidRequestContent")
		})
	}
	azureResourcesExpectStatus(t, do(http.MethodPut, fabric+azureSiteRecoveryTestAPIVersion, `{"properties":{"customDetails":{"instanceType":"VMwareV2"}}}`), http.StatusCreated)
	mapping := fabric + "/replicationProtectionContainers/c1/replicationProtectionContainerMappings/m1"
	azureResourcesExpectErrorCode(t, do(http.MethodPut, mapping+azureSiteRecoveryTestAPIVersion, `{"properties":{"policyId":"p","providerSpecificInput":{"instanceType":"A2A"}}}`), http.StatusNotFound, "ParentResourceNotFound")
	azureResourcesExpectStatus(t, do(http.MethodPut, fabric+"/replicationProtectionContainers/c1"+azureSiteRecoveryTestAPIVersion, `{"properties":{}}`), http.StatusCreated)
	azureResourcesExpectErrorCode(t, do(http.MethodPut, mapping+azureSiteRecoveryTestAPIVersion, `{"properties":{"providerSpecificInput":{"instanceType":"A2A"}}}`), http.StatusBadRequest, "InvalidRequestContent")
	azureResourcesExpectErrorCode(t, do(http.MethodPut, mapping+azureSiteRecoveryTestAPIVersion, `{"properties":{"policyId":"p"}}`), http.StatusBadRequest, "InvalidRequestContent")
}
