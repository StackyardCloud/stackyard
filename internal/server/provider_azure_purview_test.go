package server

import (
	"net/http"
	"strings"
	"testing"
)

const azurePurviewTestAPIVersion = "?api-version=2021-12-01"

func TestAzurePurviewLifecycle(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-purview")
	account := sub + "/resourceGroups/rg-purview/providers/Microsoft.Purview/accounts/governance"

	resp := do(http.MethodPut, account+azurePurviewTestAPIVersion, `{"location":"eastus","identity":{"type":"SystemAssigned"},"properties":{"managedResourceGroupName":"purview-managed","endpoints":{"catalog":"https://forged"}}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	props := providerContractJSONMap(t, resp)["properties"].(map[string]any)
	endpoints := props["endpoints"].(map[string]any)
	if endpoints["catalog"] != "https://governance.purview.azure.com/catalog" || props["managedResourceGroupName"] != "purview-managed" || props["publicNetworkAccess"] != "Enabled" {
		t.Fatalf("unexpected account properties: %#v", props)
	}
	if storage := props["managedResources"].(map[string]any)["storageAccount"].(string); len(storage[strings.LastIndex(storage, "/")+1:]) > 24 {
		t.Fatalf("managed storage account name too long: %s", storage)
	}

	resp = do(http.MethodPost, account+"/listkeys"+azurePurviewTestAPIVersion, "")
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if keys := providerContractJSONMap(t, resp); !strings.HasPrefix(keys["atlasKafkaPrimaryEndpoint"].(string), "Endpoint=sb://atlas-") || keys["atlasKafkaPrimaryEndpoint"] == keys["atlasKafkaSecondaryEndpoint"] {
		t.Fatalf("unexpected keys: %#v", keys)
	}
	azureResourcesExpectStatus(t, do(http.MethodPost, account+"/addRootCollectionAdmin"+azurePurviewTestAPIVersion, `{"objectId":"7e8de0e7-2bfc-4e1f-9659-2a5785e4356f"}`), http.StatusOK)
	azureResourcesExpectErrorCode(t, do(http.MethodPost, account+"/addRootCollectionAdmin"+azurePurviewTestAPIVersion, `{}`), http.StatusBadRequest, "InvalidRequestContent")

	check := func(name string) map[string]any {
		resp := do(http.MethodPost, sub+"/providers/Microsoft.Purview/checkNameAvailability"+azurePurviewTestAPIVersion, `{"name":"`+name+`","type":"Microsoft.Purview/accounts"}`)
		azureResourcesExpectStatus(t, resp, http.StatusOK)
		return providerContractJSONMap(t, resp)
	}
	if got := check("governance"); got["nameAvailable"] != false || got["reason"] != "AlreadyExists" {
		t.Fatalf("expected AlreadyExists, got %#v", got)
	}
	if got := check("other"); got["nameAvailable"] != true {
		t.Fatalf("expected available, got %#v", got)
	}

	azureResourcesExpectStatus(t, do(http.MethodDelete, account+azurePurviewTestAPIVersion, ""), http.StatusOK)
}

func TestAzurePurviewValidation(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-purview")
	accounts := sub + "/resourceGroups/rg-purview/providers/Microsoft.Purview/accounts/"
	cases := []struct{ name, path, body, code string }{
		{"short name", accounts + "ab", `{"location":"eastus"}`, "InvalidResourceName"},
		{"bad identity", accounts + "governance", `{"location":"eastus","identity":{"type":"Both"}}`, "InvalidRequestContent"},
		{"bad network access", accounts + "governance", `{"location":"eastus","properties":{"publicNetworkAccess":"Open"}}`, "InvalidRequestContent"},
	}
	for _, tc := range cases {
		resp := do(http.MethodPut, tc.path+azurePurviewTestAPIVersion, tc.body)
		t.Run(tc.name, func(t *testing.T) { azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, tc.code) })
	}
}
