package server

import (
	"net/http"
	"testing"
)

const azureDLSTestVersion = "?api-version=" + azureDataLakeStore20161101APIVersion

func TestAzureDataLakeStore20161101Lifecycle(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-dls")
	account := sub + "/resourceGroups/rg-dls/providers/Microsoft.DataLakeStore/accounts/lake1"

	resp := do(http.MethodPut, account+azureDLSTestVersion, `{"location":"eastus2","properties":{"firewallState":"Enabled","firewallRules":[{"name":"office","properties":{"startIpAddress":"10.0.0.1","endIpAddress":"10.0.0.9"}}]}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	props := providerContractJSONMap(t, resp)["properties"].(map[string]any)
	if props["endpoint"] != "lake1.azuredatalakestore.net" || props["currentTier"] != "Consumption" || props["encryptionState"] != "Enabled" {
		t.Fatalf("unexpected account properties: %#v", props)
	}
	if rules, _ := props["firewallRules"].([]any); len(rules) != 1 {
		t.Fatalf("inline firewall rule missing: %#v", props["firewallRules"])
	}

	resp = do(http.MethodGet, account+"/firewallRules/office"+azureDLSTestVersion, "")
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if got := providerContractJSONMap(t, resp)["properties"].(map[string]any)["startIpAddress"]; got != "10.0.0.1" {
		t.Fatalf("firewall rule start = %v", got)
	}
	azureResourcesExpectStatus(t, do(http.MethodPut, account+"/trustedIdProviders/aad"+azureDLSTestVersion, `{"properties":{"idProvider":"https://sts.windows.net/00000000-0000-0000-0000-000000000000/"}}`), http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodPut, account+"/virtualNetworkRules/vnet"+azureDLSTestVersion, `{"properties":{"subnetId":"/subscriptions/x/resourceGroups/rg/providers/Microsoft.Network/virtualNetworks/v/subnets/s"}}`), http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodPost, account+"/enableKeyVault"+azureDLSTestVersion, ""), http.StatusOK)

	resp = do(http.MethodPost, sub+"/providers/Microsoft.DataLakeStore/locations/eastus2/checkNameAvailability"+azureDLSTestVersion, `{"name":"lake1","type":"Microsoft.DataLakeStore/accounts"}`)
	if body := providerContractJSONMap(t, resp); body["nameAvailable"] != false || body["reason"] != "AlreadyExists" {
		t.Fatalf("unexpected name check: %#v", body)
	}
	resp = do(http.MethodGet, sub+"/providers/Microsoft.DataLakeStore/locations/eastus2/capability"+azureDLSTestVersion, "")
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if got := providerContractJSONMap(t, resp)["accountCount"]; got != float64(1) {
		t.Fatalf("capability accountCount = %v", got)
	}
	resp = do(http.MethodGet, sub+"/providers/Microsoft.DataLakeStore/locations/eastus2/usages"+azureDLSTestVersion, "")
	if usages := providerContractJSONMap(t, resp)["value"].([]any); len(usages) != 1 || usages[0].(map[string]any)["currentValue"] != float64(1) {
		t.Fatalf("unexpected usages: %#v", usages)
	}
	resp = do(http.MethodGet, "/azure/providers/Microsoft.DataLakeStore/operations"+azureDLSTestVersion, "")
	if ops := providerContractJSONMap(t, resp)["value"].([]any); len(ops) == 0 {
		t.Fatal("operations list is empty")
	}

	azureResourcesExpectStatus(t, do(http.MethodDelete, account+azureDLSTestVersion, ""), http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodGet, account+"/firewallRules/office"+azureDLSTestVersion, ""), http.StatusNotFound)
}

func TestAzureDataLakeStore20161101Validation(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-dls")
	account := sub + "/resourceGroups/rg-dls/providers/Microsoft.DataLakeStore/accounts/"
	cases := []struct{ name, path, body, code string }{
		{"bad name", account + "Bad_Name", `{"location":"eastus"}`, "InvalidResourceName"},
		{"bad tier", account + "lake2", `{"location":"eastus","properties":{"newTier":"Huge"}}`, "InvalidRequestContent"},
		{"user managed without vault", account + "lake2", `{"location":"eastus","properties":{"encryptionConfig":{"type":"UserManaged"}}}`, "InvalidRequestContent"},
		{"bad inline rule", account + "lake2", `{"location":"eastus","properties":{"firewallRules":[{"name":"r","properties":{"startIpAddress":"10.0.0.9","endIpAddress":"10.0.0.1"}}]}}`, "InvalidRequestContent"},
	}
	for _, tc := range cases {
		resp := do(http.MethodPut, tc.path+azureDLSTestVersion, tc.body)
		t.Run(tc.name, func(t *testing.T) { azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, tc.code) })
	}
	azureResourcesExpectStatus(t, do(http.MethodGet, account+"missing"+azureDLSTestVersion, ""), http.StatusNotFound)
	resp := do(http.MethodPost, sub+"/providers/Microsoft.DataLakeStore/locations/eastus/checkNameAvailability"+azureDLSTestVersion, `{"name":"x","type":"Microsoft.Storage/storageAccounts"}`)
	azureResourcesExpectStatus(t, resp, http.StatusBadRequest)
}
