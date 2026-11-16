package server

import (
	"net/http"
	"strings"
	"testing"
)

const azureServiceLinkerTestAPIVersion = "?api-version=2024-04-01"

func TestAzureServiceLinkerLifecycle(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-link")
	ids := strings.TrimPrefix(sub, "/azure") + "/resourceGroups/rg-link/providers"
	app := sub + "/resourceGroups/rg-link/providers/Microsoft.Web/sites/shop"
	linker := app + "/providers/Microsoft.ServiceLinker/linkers/storage_blob"

	resp := do(http.MethodPut, linker+azureServiceLinkerTestAPIVersion, `{"properties":{"targetService":{"type":"AzureResource","id":"`+ids+`/Microsoft.Storage/storageAccounts/ShopData/blobServices/default"},"authInfo":{"authType":"userAssignedIdentity","clientId":"7f6c6a2e-36b4-4e55-8b56-1c1d2b0e5f11","subscriptionId":"00000000-0000-0000-0000-000000000000"},"clientType":"python","configurationInfo":{"customizedKeys":{"AZURE_STORAGEBLOB_CLIENTID":"BLOB_CLIENT_ID"}}}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	resp = do(http.MethodPost, linker+"/listConfigurations"+azureServiceLinkerTestAPIVersion, "")
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	got := map[string]any{}
	for _, item := range providerContractJSONMap(t, resp)["configurations"].([]any) {
		setting := item.(map[string]any)
		got[setting["name"].(string)] = setting["value"]
	}
	if got["AZURE_STORAGEBLOB_RESOURCEENDPOINT"] != "https://shopdata.blob.core.windows.net/" || got["BLOB_CLIENT_ID"] != "7f6c6a2e-36b4-4e55-8b56-1c1d2b0e5f11" {
		t.Fatalf("unexpected configurations: %#v", got)
	}
	resp = do(http.MethodPost, linker+"/validateLinker"+azureServiceLinkerTestAPIVersion, "")
	props := providerContractJSONMap(t, resp)["properties"].(map[string]any)
	if props["isConnectionAvailable"] != true || props["sourceId"] != strings.TrimPrefix(app, "/azure") {
		t.Fatalf("unexpected validation result: %#v", props)
	}

	secret := app + "/providers/Microsoft.ServiceLinker/linkers/postgres"
	resp = do(http.MethodPut, secret+azureServiceLinkerTestAPIVersion, `{"properties":{"targetService":{"type":"AzureResource","id":"`+ids+`/Microsoft.DBforPostgreSQL/flexibleServers/db/databases/shop"},"authInfo":{"authType":"secret","name":"admin","secretInfo":{"secretType":"rawValue","value":"P@ssw0rd!"}}}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	if auth := providerContractJSONMap(t, resp)["properties"].(map[string]any)["authInfo"].(map[string]any); auth["secretInfo"].(map[string]any)["value"] != nil {
		t.Fatalf("the secret was returned: %#v", auth)
	}
	resp = do(http.MethodPost, secret+"/generateConfigurations"+azureServiceLinkerTestAPIVersion, `{"customizedKeys":{"AZURE_POSTGRESQL_CONNECTIONSTRING":"DATABASE_URL"}}`)
	if items := providerContractJSONMap(t, resp)["configurations"].([]any); items[0].(map[string]any)["name"] != "DATABASE_URL" {
		t.Fatalf("unexpected generated configurations: %#v", items)
	}
	resp = do(http.MethodGet, app+"/providers/Microsoft.ServiceLinker/linkers"+azureServiceLinkerTestAPIVersion, "")
	if items := providerContractJSONMap(t, resp)["value"].([]any); len(items) != 2 {
		t.Fatalf("expected two linkers, got %d", len(items))
	}

	dryrun := app + "/providers/Microsoft.ServiceLinker/dryruns/preview"
	resp = do(http.MethodPut, dryrun+azureServiceLinkerTestAPIVersion, `{"properties":{"parameters":{"actionName":"createOrUpdate","targetService":{"type":"AzureResource","id":"`+ids+`/Microsoft.KeyVault/vaults/shop"},"authInfo":{"authType":"systemAssignedIdentity"}}}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	preview := providerContractJSONMap(t, resp)["properties"].(map[string]any)["operationPreviews"].([]any)[0].(map[string]any)
	if preview["action"] != "Microsoft.Web/sites/write" {
		t.Fatalf("unexpected preview: %#v", preview)
	}
	azureResourcesExpectStatus(t, do(http.MethodDelete, linker+azureServiceLinkerTestAPIVersion, ""), http.StatusOK)
}

func TestAzureServiceLinkerValidation(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-link")
	app := sub + "/resourceGroups/rg-link/providers/Microsoft.Web/sites/shop/providers/Microsoft.ServiceLinker"
	target := `"targetService":{"type":"AzureResource","id":"/subscriptions/x/resourceGroups/rg/providers/Microsoft.KeyVault/vaults/v"}`
	cases := []struct{ name, path, body string }{
		{"target type", "/linkers/l", `{"properties":{"targetService":{"type":"Database"}}}`},
		{"target id", "/linkers/l", `{"properties":{"targetService":{"type":"AzureResource","id":"vault"}}}`},
		{"endpoint", "/linkers/l", `{"properties":{"targetService":{"type":"SelfHostedServer"}}}`},
		{"auth type", "/linkers/l", `{"properties":{` + target + `,"authInfo":{"authType":"password"}}}`},
		{"principal secret", "/linkers/l", `{"properties":{` + target + `,"authInfo":{"authType":"servicePrincipalSecret","clientId":"c","principalId":"p"}}}`},
		{"secret type", "/linkers/l", `{"properties":{` + target + `,"authInfo":{"authType":"secret","secretInfo":{"secretType":"plain"}}}}`},
		{"client type", "/linkers/l", `{"properties":{` + target + `,"clientType":"rust"}}`},
		{"vnet", "/linkers/l", `{"properties":{` + target + `,"vNetSolution":{"type":"vpn"}}}`},
		{"dryrun action", "/dryruns/d", `{"properties":{"parameters":{"actionName":"delete",` + target + `}}}`},
		{"dryrun target", "/dryruns/d", `{"properties":{"parameters":{"actionName":"createOrUpdate"}}}`},
	}
	for _, tc := range cases {
		resp := do(http.MethodPut, app+tc.path+azureServiceLinkerTestAPIVersion, tc.body)
		t.Run(tc.name, func(t *testing.T) {
			azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, "InvalidRequestContent")
		})
	}
	azureResourcesExpectErrorCode(t, do(http.MethodPut, app+"/linkers/my-link"+azureServiceLinkerTestAPIVersion, `{"properties":{`+target+`}}`), http.StatusBadRequest, "InvalidResourceName")
	azureResourcesExpectErrorCode(t, do(http.MethodPut, sub+"/resourceGroups/rg-link/providers/Microsoft.ServiceLinker/linkers/l"+azureServiceLinkerTestAPIVersion, `{"properties":{`+target+`}}`), http.StatusBadRequest, "InvalidScope")
}
