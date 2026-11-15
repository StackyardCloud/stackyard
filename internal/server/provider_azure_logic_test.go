package server

import (
	"net/http"
	"strings"
	"testing"
)

const azureLogicTestAPIVersion = "?api-version=2019-05-01"

const azureLogicTestWorkflow = `{"location":"eastus","properties":{"definition":{"$schema":"https://schema.management.azure.com/providers/Microsoft.Logic/schemas/2016-06-01/workflowdefinition.json#","contentVersion":"1.0.0.0","triggers":{"manual":{"type":"Request","kind":"Http"}},"actions":{}}}}`

func TestAzureLogicWorkflowLifecycle(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-logic")
	wf := sub + "/resourceGroups/rg-logic/providers/Microsoft.Logic/workflows/orders"

	resp := do(http.MethodPut, wf+azureLogicTestAPIVersion, azureLogicTestWorkflow)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	props := providerContractJSONMap(t, resp)["properties"].(map[string]any)
	if props["state"] != "Enabled" || !strings.HasPrefix(props["accessEndpoint"].(string), "https://prod-00.eastus.logic.azure.com:443/workflows/") || props["definition"] == nil {
		t.Fatalf("unexpected workflow properties: %#v", props)
	}

	for _, step := range []struct{ action, state string }{{"disable", "Disabled"}, {"enable", "Enabled"}} {
		azureResourcesExpectStatus(t, do(http.MethodPost, wf+"/"+step.action+azureLogicTestAPIVersion, ""), http.StatusOK)
		resp = do(http.MethodGet, wf+azureLogicTestAPIVersion, "")
		if got := providerContractJSONMap(t, resp)["properties"].(map[string]any)["state"]; got != step.state {
			t.Fatalf("after %s: expected %s, got %v", step.action, step.state, got)
		}
	}
	azureResourcesExpectStatus(t, do(http.MethodDelete, wf+azureLogicTestAPIVersion, ""), http.StatusOK)
}

func TestAzureLogicIntegrationAccountLifecycle(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-logic")
	account := sub + "/resourceGroups/rg-logic/providers/Microsoft.Logic/integrationAccounts/b2b"
	azureResourcesExpectStatus(t, do(http.MethodPut, account+azureLogicTestAPIVersion, `{"location":"eastus","sku":{"name":"Basic"},"properties":{}}`), http.StatusCreated)

	resp := do(http.MethodPost, account+"/listCallbackUrl"+azureLogicTestAPIVersion, `{"keyType":"Primary"}`)
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if url := providerContractJSONMap(t, resp)["value"].(string); !strings.Contains(url, "/integrationAccounts/b2b?") || !strings.Contains(url, "sig=") {
		t.Fatalf("unexpected callback URL %q", url)
	}

	// Schemas and maps take no location.
	resp = do(http.MethodPut, account+"/schemas/order"+azureLogicTestAPIVersion, `{"properties":{"schemaType":"Xml","content":"<xs:schema/>","contentType":"application/xml"}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	link := providerContractJSONMap(t, resp)["properties"].(map[string]any)["contentLink"].(map[string]any)
	if link["contentSize"] != float64(len("<xs:schema/>")) || !strings.Contains(link["uri"].(string), "/integrationAccounts/b2b/schemas/order/") {
		t.Fatalf("unexpected content link: %#v", link)
	}
	azureResourcesExpectStatus(t, do(http.MethodPut, account+"/maps/toInvoice"+azureLogicTestAPIVersion, `{"properties":{"mapType":"Liquid","content":"{{ content }}","contentType":"text/plain"}}`), http.StatusCreated)

	azureResourcesExpectStatus(t, do(http.MethodDelete, account+azureLogicTestAPIVersion, ""), http.StatusOK)
	azureResourcesExpectErrorCode(t, do(http.MethodGet, account+"/maps/toInvoice"+azureLogicTestAPIVersion, ""), http.StatusNotFound, "ParentResourceNotFound")
}

func TestAzureLogicValidation(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-logic")
	base := sub + "/resourceGroups/rg-logic/providers/Microsoft.Logic"
	azureResourcesExpectStatus(t, do(http.MethodPut, base+"/integrationAccounts/b2b"+azureLogicTestAPIVersion, `{"location":"eastus","sku":{"name":"Free"},"properties":{}}`), http.StatusCreated)
	cases := []struct{ name, path, body string }{
		{"bad state", base + "/workflows/w", `{"location":"eastus","properties":{"state":"Suspended"}}`},
		{"definition not an object", base + "/workflows/w", `{"location":"eastus","properties":{"definition":"x"}}`},
		{"triggers not an object", base + "/workflows/w", `{"location":"eastus","properties":{"definition":{"triggers":[]}}}`},
		{"bad sku", base + "/integrationAccounts/x", `{"location":"eastus","sku":{"name":"Premium"},"properties":{}}`},
		{"schema type", base + "/integrationAccounts/b2b/schemas/s", `{"properties":{"schemaType":"Json","content":"{}"}}`},
		{"map type", base + "/integrationAccounts/b2b/maps/m", `{"properties":{"content":"x"}}`},
	}
	for _, tc := range cases {
		resp := do(http.MethodPut, tc.path+azureLogicTestAPIVersion, tc.body)
		t.Run(tc.name, func(t *testing.T) {
			azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, "InvalidRequestContent")
		})
	}
}
