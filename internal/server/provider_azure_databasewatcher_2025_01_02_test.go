package server

import (
	"net/http"
	"testing"
)

const azureDBWTestVersion = "?api-version=" + azureDatabaseWatcher20250102APIVersion

func TestAzureDatabaseWatcher20250102Lifecycle(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-dbw")
	watcher := sub + "/resourceGroups/rg-dbw/providers/Microsoft.DatabaseWatcher/watchers/w1"

	resp := do(http.MethodPut, watcher+azureDBWTestVersion, `{"location":"eastus","identity":{"type":"SystemAssigned"},"properties":{"datastore":{"kustoClusterUri":"https://k.kusto.windows.net","kustoDataIngestionUri":"https://ingest-k.kusto.windows.net","kustoDatabaseName":"db","kustoManagementUrl":"https://portal.azure.com","kustoOfferingType":"adx"},"status":"Running"}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	if got := providerContractJSONMap(t, resp)["properties"].(map[string]any)["status"]; got != "Stopped" {
		t.Fatalf("a new watcher must be Stopped whatever the client sends, got %v", got)
	}
	azureResourcesExpectStatus(t, do(http.MethodPost, watcher+"/start"+azureDBWTestVersion, ""), http.StatusOK)
	resp = do(http.MethodGet, watcher+azureDBWTestVersion, "")
	if got := providerContractJSONMap(t, resp)["properties"].(map[string]any)["status"]; got != "Running" {
		t.Fatalf("status after start = %v", got)
	}

	azureResourcesExpectStatus(t, do(http.MethodPut, watcher+"/targets/db1"+azureDBWTestVersion,
		`{"properties":{"targetType":"SqlDb","targetAuthenticationType":"Aad","connectionServerName":"srv.database.windows.net","sqlDbResourceId":"/subscriptions/x/resourceGroups/rg/providers/Microsoft.Sql/servers/srv/databases/db"}}`), http.StatusCreated)
	azureResourcesExpectStatus(t, do(http.MethodPut, watcher+"/alertRuleResources/cpu"+azureDBWTestVersion,
		`{"properties":{"alertRuleResourceId":"/subscriptions/x/resourceGroups/rg/providers/Microsoft.Insights/scheduledQueryRules/cpu","createdWithProperties":"None","creationTime":"2025-01-02T00:00:00Z","alertRuleTemplateId":"cpu","alertRuleTemplateVersion":"1.0"}}`), http.StatusCreated)
	resp = do(http.MethodPut, watcher+"/sharedPrivateLinkResources/spl"+azureDBWTestVersion,
		`{"properties":{"privateLinkResourceId":"/subscriptions/x/resourceGroups/rg/providers/Microsoft.Sql/servers/srv","groupId":"sqlServer","requestMessage":"please"}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	if got := providerContractJSONMap(t, resp)["properties"].(map[string]any)["status"]; got != "Pending" {
		t.Fatalf("shared private link status = %v", got)
	}

	resp = do(http.MethodPost, watcher+"/healthValidations/check1/startValidation"+azureDBWTestVersion, "")
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if got := providerContractJSONMap(t, resp)["properties"].(map[string]any)["status"]; got != "Succeeded" {
		t.Fatalf("health validation status = %v", got)
	}
	resp = do(http.MethodGet, watcher+"/healthValidations"+azureDBWTestVersion, "")
	if items := providerContractJSONMap(t, resp)["value"].([]any); len(items) != 1 {
		t.Fatalf("want 1 health validation, got %d", len(items))
	}
	azureResourcesExpectStatus(t, do(http.MethodPut, watcher+"/healthValidations/check1"+azureDBWTestVersion, `{}`), http.StatusMethodNotAllowed)
	resp = do(http.MethodGet, "/azure/providers/Microsoft.DatabaseWatcher/operations"+azureDBWTestVersion, "")
	if ops := providerContractJSONMap(t, resp)["value"].([]any); len(ops) == 0 {
		t.Fatal("operations list is empty")
	}

	azureResourcesExpectStatus(t, do(http.MethodDelete, watcher+azureDBWTestVersion, ""), http.StatusNoContent)
	azureResourcesExpectStatus(t, do(http.MethodGet, watcher+"/targets/db1"+azureDBWTestVersion, ""), http.StatusNotFound)
}

func TestAzureDatabaseWatcher20250102Validation(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-dbw")
	watcher := sub + "/resourceGroups/rg-dbw/providers/Microsoft.DatabaseWatcher/watchers/w2"
	azureResourcesExpectStatus(t, do(http.MethodPost, watcher+"/healthValidations/v/startValidation"+azureDBWTestVersion, ""), http.StatusNotFound)
	azureResourcesExpectErrorCode(t, do(http.MethodPut, watcher+azureDBWTestVersion, `{"location":"eastus","properties":{"datastore":{"kustoOfferingType":"adx"}}}`), http.StatusBadRequest, "InvalidRequestContent")
	azureResourcesExpectStatus(t, do(http.MethodPut, watcher+azureDBWTestVersion, `{"location":"eastus"}`), http.StatusCreated)
	cases := []struct{ name, path, body string }{
		{"target without resource ID", "/targets/t", `{"properties":{"targetType":"SqlMi","targetAuthenticationType":"Aad","connectionServerName":"mi"}}`},
		{"sql auth without vault", "/targets/t", `{"properties":{"targetType":"SqlDb","targetAuthenticationType":"Sql","connectionServerName":"s","sqlDbResourceId":"x"}}`},
		{"bad target type", "/targets/t", `{"properties":{"targetType":"Oracle","targetAuthenticationType":"Aad","connectionServerName":"s"}}`},
		{"alert rule missing template", "/alertRuleResources/a", `{"properties":{"alertRuleResourceId":"x","createdWithProperties":"None","creationTime":"2025-01-02T00:00:00Z"}}`},
		{"shared link missing group", "/sharedPrivateLinkResources/s", `{"properties":{"privateLinkResourceId":"x","requestMessage":"m"}}`},
	}
	for _, tc := range cases {
		resp := do(http.MethodPut, watcher+tc.path+azureDBWTestVersion, tc.body)
		t.Run(tc.name, func(t *testing.T) {
			azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, "InvalidRequestContent")
		})
	}
}
