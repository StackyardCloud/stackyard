package server

import (
	"net/http"
	"strings"
	"testing"
)

const azureCustomProvidersTestAPIVersion = "?api-version=2018-09-01-preview"

func TestAzureCustomProvidersLifecycle(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-crp")
	provider := sub + "/resourceGroups/rg-crp/providers/Microsoft.CustomProviders/resourceProviders/inventory"
	azureResourcesExpectStatus(t, do(http.MethodPut, provider+azureCustomProvidersTestAPIVersion, `{"location":"eastus","properties":{"actions":[{"name":"ping","endpoint":"https://inventory.contoso.com/api/ping","routingType":"Proxy"}],"resourceTypes":[{"name":"assets","endpoint":"https://inventory.contoso.com/api/assets","routingType":"Proxy,Cache"}],"validations":[{"validationType":"Swagger","specification":"https://raw.githubusercontent.com/contoso/inventory/main/swagger.json"}]}}`), http.StatusCreated)
	// Updates change tags only.
	resp := do(http.MethodPatch, provider+azureCustomProvidersTestAPIVersion, `{"tags":{"team":"inventory"},"properties":{"actions":[{"name":"ping","endpoint":"http://insecure"}]}}`)
	body := providerContractJSONMap(t, resp)
	endpoint := body["properties"].(map[string]any)["actions"].([]any)[0].(map[string]any)["endpoint"]
	if body["tags"].(map[string]any)["team"] != "inventory" || endpoint != "https://inventory.contoso.com/api/ping" {
		t.Fatalf("unexpected patched provider: %#v", body)
	}

	vm := sub + "/resourceGroups/rg-crp/providers/Microsoft.Compute/virtualMachines/web1"
	association := vm + "/providers/Microsoft.CustomProviders/associations/inventory"
	resp = do(http.MethodPut, association+azureCustomProvidersTestAPIVersion, `{"properties":{"targetResourceId":"`+strings.TrimPrefix(provider, "/azure")+`"}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	resp = do(http.MethodGet, vm+"/providers/Microsoft.CustomProviders/associations"+azureCustomProvidersTestAPIVersion, "")
	if got := providerContractJSONMap(t, resp)["value"].([]any); len(got) != 1 {
		t.Fatalf("expected one association, got %d", len(got))
	}
	azureResourcesExpectStatus(t, do(http.MethodDelete, association+azureCustomProvidersTestAPIVersion, ""), http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodDelete, provider+azureCustomProvidersTestAPIVersion, ""), http.StatusOK)
}

func TestAzureCustomProvidersValidation(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-crp")
	provider := sub + "/resourceGroups/rg-crp/providers/Microsoft.CustomProviders/resourceProviders/inventory" + azureCustomProvidersTestAPIVersion
	for name, body := range map[string]string{
		"endpoint":      `{"location":"eastus","properties":{"actions":[{"name":"ping","endpoint":"http://inventory.contoso.com/ping"}]}}`,
		"action name":   `{"location":"eastus","properties":{"actions":[{"endpoint":"https://inventory.contoso.com/ping"}]}}`,
		"duplicate":     `{"location":"eastus","properties":{"resourceTypes":[{"name":"assets","endpoint":"https://a.contoso.com"},{"name":"Assets","endpoint":"https://b.contoso.com"}]}}`,
		"action route":  `{"location":"eastus","properties":{"actions":[{"name":"ping","endpoint":"https://inventory.contoso.com/ping","routingType":"Proxy,Cache"}]}}`,
		"specification": `{"location":"eastus","properties":{"validations":[{"specification":"https://contoso.com/swagger.json"}]}}`,
	} {
		resp := do(http.MethodPut, provider, body)
		t.Run(name, func(t *testing.T) {
			azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, "InvalidRequestContent")
		})
	}
	azureResourcesExpectErrorCode(t, do(http.MethodPut, sub+"/resourceGroups/rg-crp/providers/Microsoft.CustomProviders/resourceProviders/ab"+azureCustomProvidersTestAPIVersion, `{"location":"eastus","properties":{}}`), http.StatusBadRequest, "InvalidResourceName")
	association := sub + "/resourceGroups/rg-crp/providers/Microsoft.CustomProviders/associations/a" + azureCustomProvidersTestAPIVersion
	missing := strings.TrimPrefix(sub, "/azure") + "/resourceGroups/rg-crp/providers/Microsoft.CustomProviders/resourceProviders/missing"
	azureResourcesExpectErrorCode(t, do(http.MethodPut, association, `{"properties":{"targetResourceId":"`+missing+`"}}`), http.StatusBadRequest, "InvalidRequestContent")
	azureResourcesExpectErrorCode(t, do(http.MethodPut, association, `{"properties":{"targetResourceId":"inventory"}}`), http.StatusBadRequest, "InvalidRequestContent")
}
