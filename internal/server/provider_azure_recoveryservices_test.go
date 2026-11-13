package server

import (
	"net/http"
	"testing"
)

const azureRecoveryServicesTestAPIVersion = "?api-version=2024-04-01"

// newAzureRecoveryServicesVault creates vault "backups" in rg-rsv and
// returns its path.
func newAzureRecoveryServicesVault(t *testing.T) (string, func(method, path, body string) *http.Response) {
	t.Helper()
	sub, do := newAzureDatabaseTestServer(t, "rg-rsv")
	vault := sub + "/resourceGroups/rg-rsv/providers/Microsoft.RecoveryServices/vaults/backups"
	azureResourcesExpectStatus(t, do(http.MethodPut, vault+azureRecoveryServicesTestAPIVersion, `{"location":"eastus","sku":{"name":"RS0","tier":"Standard"},"properties":{"publicNetworkAccess":"Disabled"}}`), http.StatusCreated)
	return vault, do
}

func TestAzureRecoveryServicesVaultLifecycle(t *testing.T) {
	t.Parallel()
	vault, do := newAzureRecoveryServicesVault(t)

	resp := do(http.MethodGet, vault+azureRecoveryServicesTestAPIVersion, "")
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	props, _ := providerContractJSONMap(t, resp)["properties"].(map[string]any)
	redundancy, _ := props["redundancySettings"].(map[string]any)
	if props["publicNetworkAccess"] != "Disabled" || props["privateEndpointStateForBackup"] != "None" || redundancy["standardTierStorageRedundancy"] != "GeoRedundant" {
		t.Fatalf("unexpected vault properties: %#v", props)
	}

	resp = do(http.MethodPatch, vault+azureRecoveryServicesTestAPIVersion, `{"properties":{"redundancySettings":{"crossRegionRestore":"Enabled"}}}`)
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	redundancy, _ = providerContractJSONMap(t, resp)["properties"].(map[string]any)["redundancySettings"].(map[string]any)
	if redundancy["crossRegionRestore"] != "Enabled" || redundancy["standardTierStorageRedundancy"] != "GeoRedundant" {
		t.Fatalf("unexpected redundancy after PATCH: %#v", redundancy)
	}

	azureResourcesExpectStatus(t, do(http.MethodDelete, vault+azureRecoveryServicesTestAPIVersion, ""), http.StatusOK)
	// The default backup children went with the vault.
	azureResourcesExpectErrorCode(t, do(http.MethodGet, vault+"/backupPolicies/DefaultPolicy"+azureRecoveryServicesTestAPIVersion, ""), http.StatusNotFound, "ParentResourceNotFound")
}

func TestAzureRecoveryServicesValidation(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-rsv")
	vaults := sub + "/resourceGroups/rg-rsv/providers/Microsoft.RecoveryServices/vaults/"
	cases := []struct{ name, path, body, code string }{
		{"leading digit", vaults + "1vault", `{"location":"eastus","sku":{"name":"Standard"}}`, "InvalidResourceName"},
		{"missing sku", vaults + "vault", `{"location":"eastus"}`, "InvalidRequestContent"},
		{"bad sku", vaults + "vault", `{"location":"eastus","sku":{"name":"Premium"}}`, "InvalidRequestContent"},
		{"bad redundancy", vaults + "vault", `{"location":"eastus","sku":{"name":"Standard"},"properties":{"redundancySettings":{"standardTierStorageRedundancy":"Everywhere"}}}`, "InvalidRequestContent"},
	}
	for _, tc := range cases {
		resp := do(http.MethodPut, tc.path+azureRecoveryServicesTestAPIVersion, tc.body)
		t.Run(tc.name, func(t *testing.T) { azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, tc.code) })
	}

	check := func(rg, body string) *http.Response {
		return do(http.MethodPost, sub+"/resourceGroups/"+rg+"/providers/Microsoft.RecoveryServices/locations/eastus/checkNameAvailability"+azureRecoveryServicesTestAPIVersion, body)
	}
	const vaultCheck = `{"type":"Microsoft.RecoveryServices/Vaults","name":"Vault"}`
	resp := check("rg-rsv", vaultCheck)
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if got := providerContractJSONMap(t, resp); got["nameAvailable"] != true {
		t.Fatalf("expected available, got %#v", got)
	}
	azureResourcesExpectStatus(t, do(http.MethodPut, vaults+"vault"+azureRecoveryServicesTestAPIVersion, `{"location":"eastus","sku":{"name":"Standard"}}`), http.StatusCreated)
	resp = check("rg-rsv", vaultCheck)
	if got := providerContractJSONMap(t, resp); got["reason"] != "AlreadyExists" {
		t.Fatalf("expected AlreadyExists, got %#v", got)
	}
	// Names are only taken within the resource group.
	azureResourcesExpectStatus(t, do(http.MethodPut, sub+"/resourcegroups/rg-other?api-version=2021-04-01", `{"location":"eastus"}`), http.StatusCreated)
	if got := providerContractJSONMap(t, check("rg-other", vaultCheck)); got["nameAvailable"] != true {
		t.Fatalf("expected available in another group, got %#v", got)
	}
	azureResourcesExpectErrorCode(t, check("rg-missing", vaultCheck), http.StatusNotFound, "ResourceGroupNotFound")
	azureResourcesExpectErrorCode(t, check("rg-rsv", `{"type":"Microsoft.Storage/storageAccounts","name":"vault"}`), http.StatusBadRequest, "InvalidRequestContent")
}
