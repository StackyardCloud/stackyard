package server

import (
	"net/http"
	"testing"
)

const azureBackupTestAPIVersion = "?api-version=2024-10-01"

func TestAzureBackupDefaultsAndPolicies(t *testing.T) {
	t.Parallel()
	vault, do := newAzureRecoveryServicesVault(t)

	resp := do(http.MethodGet, vault+"/backupPolicies"+azureBackupTestAPIVersion, "")
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if items, _ := providerContractJSONMap(t, resp)["value"].([]any); len(items) != 3 {
		t.Fatalf("expected the three built-in policies, got %#v", items)
	}
	resp = do(http.MethodGet, vault+"/backupconfig/vaultconfig"+azureBackupTestAPIVersion, "")
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if props := providerContractJSONMap(t, resp)["properties"].(map[string]any); props["softDeleteFeatureState"] != "Enabled" {
		t.Fatalf("unexpected vault config: %#v", props)
	}
	resp = do(http.MethodPatch, vault+"/backupstorageconfig/vaultstorageconfig"+azureBackupTestAPIVersion, `{"properties":{"storageModelType":"LocallyRedundant"}}`)
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if props := providerContractJSONMap(t, resp)["properties"].(map[string]any); props["storageModelType"] != "LocallyRedundant" || props["storageTypeState"] != "Unlocked" {
		t.Fatalf("unexpected storage config: %#v", props)
	}

	policy := vault + "/backupPolicies/nightly"
	resp = do(http.MethodPut, policy+azureBackupTestAPIVersion, `{"properties":{"backupManagementType":"AzureStorage","workLoadType":"AzureFileShare","timeZone":"UTC"}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	if props := providerContractJSONMap(t, resp)["properties"].(map[string]any); props["protectedItemsCount"] != float64(0) {
		t.Fatalf("unexpected policy: %#v", props)
	}
	azureResourcesExpectStatus(t, do(http.MethodDelete, policy+azureBackupTestAPIVersion, ""), http.StatusOK)

	cases := []struct{ name, path, body string }{
		{"bad management type", policy, `{"properties":{"backupManagementType":"Tape"}}`},
		{"bad soft delete", vault + "/backupconfig/vaultconfig", `{"properties":{"softDeleteFeatureState":"Maybe"}}`},
		{"bad storage", vault + "/backupstorageconfig/vaultstorageconfig", `{"properties":{"storageModelType":"Everywhere"}}`},
	}
	for _, tc := range cases {
		resp := do(http.MethodPut, tc.path+azureBackupTestAPIVersion, tc.body)
		t.Run(tc.name, func(t *testing.T) { azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, "InvalidRequestContent") })
	}
	azureResourcesExpectErrorCode(t, do(http.MethodPut, vault+"/backupconfig/other"+azureBackupTestAPIVersion, `{"properties":{}}`), http.StatusBadRequest, "InvalidResourceName")
}

func TestAzureBackupProtectedItems(t *testing.T) {
	t.Parallel()
	vault, do := newAzureRecoveryServicesVault(t)
	container := vault + "/backupFabrics/Azure/protectionContainers/StorageContainer;storage;rg-rsv;files"
	item := container + "/protectedItems/AzureFileShare;share1"
	policyID := "/subscriptions/" + azureResourcesTestSubscription + "/resourceGroups/rg-rsv/providers/Microsoft.RecoveryServices/vaults/backups/backupPolicies/DefaultPolicy"

	// Items need a registered container.
	azureResourcesExpectErrorCode(t, do(http.MethodPut, item+azureBackupTestAPIVersion, `{"properties":{"protectedItemType":"AzureFileShareProtectedItem","policyId":"`+policyID+`"}}`), http.StatusNotFound, "ParentResourceNotFound")
	resp := do(http.MethodPut, container+azureBackupTestAPIVersion, `{"properties":{"containerType":"StorageContainer","backupManagementType":"AzureStorage"}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	if props := providerContractJSONMap(t, resp)["properties"].(map[string]any); props["registrationStatus"] != "Registered" {
		t.Fatalf("unexpected container: %#v", props)
	}

	azureResourcesExpectErrorCode(t, do(http.MethodPut, item+azureBackupTestAPIVersion, `{"properties":{"protectedItemType":"AzureFileShareProtectedItem"}}`), http.StatusBadRequest, "InvalidRequestContent")
	azureResourcesExpectErrorCode(t, do(http.MethodPut, item+azureBackupTestAPIVersion, `{"properties":{"policyId":"`+policyID+`"}}`), http.StatusBadRequest, "InvalidRequestContent")
	resp = do(http.MethodPut, item+azureBackupTestAPIVersion, `{"properties":{"protectedItemType":"AzureFileShareProtectedItem","policyId":"`+policyID+`"}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	if props := providerContractJSONMap(t, resp)["properties"].(map[string]any); props["protectionState"] != "IRPending" || props["protectionStatus"] != "Healthy" {
		t.Fatalf("unexpected protected item: %#v", props)
	}
	azureResourcesExpectStatus(t, do(http.MethodPost, item+"/backup"+azureBackupTestAPIVersion, `{"properties":{"objectType":"AzureFileShareBackupRequest"}}`), http.StatusAccepted)

	// Stopping protection drops the policy and keeps the client's state.
	resp = do(http.MethodPut, item+azureBackupTestAPIVersion, `{"properties":{"protectedItemType":"AzureFileShareProtectedItem","protectionState":"ProtectionStopped"}}`)
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if props := providerContractJSONMap(t, resp)["properties"].(map[string]any); props["protectionState"] != "ProtectionStopped" {
		t.Fatalf("expected ProtectionStopped, got %#v", props)
	}
	azureResourcesExpectErrorCode(t, do(http.MethodPut, vault+"/backupFabrics/OnPrem"+azureBackupTestAPIVersion, `{}`), http.StatusBadRequest, "InvalidResourceName")
}
