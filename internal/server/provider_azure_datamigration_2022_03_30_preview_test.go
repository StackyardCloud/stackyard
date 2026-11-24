package server

import (
	"net/http"
	"testing"
)

const azureDMSTestAPI = "?api-version=2022-03-30-preview"

func TestAzureDataMigration20220330PreviewLifecycle(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-dms")
	service := sub + "/resourceGroups/rg-dms/providers/Microsoft.DataMigration/services/dms1"
	project := service + "/projects/proj1"

	azureResourcesExpectStatus(t, do(http.MethodPut, service+azureDMSTestAPI, `{"location":"eastus","sku":{"name":"Standard_1vCores"}}`), http.StatusCreated)
	resp := do(http.MethodPost, sub+"/providers/Microsoft.DataMigration/locations/eastus/checkNameAvailability"+azureDMSTestAPI, `{"name":"dms1","type":"Microsoft.DataMigration/services"}`)
	if got := providerContractJSONMap(t, resp)["nameAvailable"]; got != false {
		t.Fatalf("service name should be taken, got %v", got)
	}
	azureResourcesExpectStatus(t, do(http.MethodPost, service+"/stop"+azureDMSTestAPI, ""), http.StatusOK)
	if got := providerContractJSONMap(t, do(http.MethodPost, service+"/checkStatus"+azureDMSTestAPI, ""))["status"]; got != "Stopped" {
		t.Fatalf("stopped service should report Stopped, got %v", got)
	}
	azureResourcesExpectStatus(t, do(http.MethodPost, service+"/start"+azureDMSTestAPI, ""), http.StatusOK)

	azureResourcesExpectStatus(t, do(http.MethodPut, project+azureDMSTestAPI, `{"location":"eastus","properties":{"sourcePlatform":"SQL","targetPlatform":"SQLDB"}}`), http.StatusCreated)
	resp = do(http.MethodPost, service+"/checkNameAvailability"+azureDMSTestAPI, `{"name":"proj1","type":"projects"}`)
	if got := providerContractJSONMap(t, resp)["nameAvailable"]; got != false {
		t.Fatalf("project name should be taken, got %v", got)
	}

	task := project + "/tasks/t1"
	resp = do(http.MethodPut, task+azureDMSTestAPI, `{"properties":{"taskType":"Migrate.SqlServer.SqlDb"}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	if state := providerContractJSONMap(t, resp)["properties"].(map[string]any)["state"]; state != "Queued" {
		t.Fatalf("new task state = %v, want Queued", state)
	}
	azureResourcesExpectErrorCode(t, do(http.MethodPost, task+"/command"+azureDMSTestAPI, `{"commandType":"explode"}`), http.StatusBadRequest, "InvalidRequestContent")
	azureResourcesExpectStatus(t, do(http.MethodPost, task+"/command"+azureDMSTestAPI, `{"commandType":"cancel"}`), http.StatusOK)
	props := providerContractJSONMap(t, do(http.MethodGet, task+azureDMSTestAPI, ""))["properties"].(map[string]any)
	if props["state"] != "Canceled" || len(props["commands"].([]any)) != 1 {
		t.Fatalf("cancel command should cancel the task and be recorded: %#v", props)
	}

	file := project + "/files/f1"
	azureResourcesExpectStatus(t, do(http.MethodPut, file+azureDMSTestAPI, `{"properties":{"extension":"json"}}`), http.StatusCreated)
	if uri := providerContractJSONMap(t, do(http.MethodPost, file+"/read"+azureDMSTestAPI, ""))["uri"]; uri == "" || uri == nil {
		t.Fatal("file read should return a URI")
	}

	skus := providerContractJSONMap(t, do(http.MethodGet, sub+"/providers/Microsoft.DataMigration/skus"+azureDMSTestAPI, ""))["value"].([]any)
	if len(skus) != 4 {
		t.Fatalf("expected 4 SKUs, got %d", len(skus))
	}
	azureResourcesExpectStatus(t, do(http.MethodGet, "/azure/providers/Microsoft.DataMigration/operations"+azureDMSTestAPI, ""), http.StatusOK)
	azureResourcesExpectErrorCode(t, do(http.MethodGet, service+"/bogus"+azureDMSTestAPI, ""), http.StatusNotFound, "InvalidResourceType")
	azureResourcesExpectStatus(t, do(http.MethodDelete, service+azureDMSTestAPI, ""), http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodGet, task+azureDMSTestAPI, ""), http.StatusNotFound)
}

func TestAzureDataMigration20220330PreviewValidation(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-dms")
	service := sub + "/resourceGroups/rg-dms/providers/Microsoft.DataMigration/services/dms1"
	for _, tc := range []struct{ name, path, body string }{
		{"bad sku", service, `{"location":"eastus","sku":{"name":"Standard_1vCore"}}`},
		{"bad subnet", service, `{"location":"eastus","sku":{"name":"Standard_1vCores"},"properties":{"virtualSubnetId":"/nope"}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			azureResourcesExpectErrorCode(t, do(http.MethodPut, tc.path+azureDMSTestAPI, tc.body), http.StatusBadRequest, "InvalidRequestContent")
		})
	}
	azureResourcesExpectStatus(t, do(http.MethodPut, service+azureDMSTestAPI, `{"location":"eastus","sku":{"name":"Premium_4vCores"}}`), http.StatusCreated)
	azureResourcesExpectErrorCode(t, do(http.MethodPut, service+"/projects/p"+azureDMSTestAPI, `{"location":"eastus","properties":{"sourcePlatform":"Oracle","targetPlatform":"SQLDB"}}`), http.StatusBadRequest, "InvalidRequestContent")
	azureResourcesExpectErrorCode(t, do(http.MethodPut, service+"/serviceTasks/st"+azureDMSTestAPI, `{"properties":{}}`), http.StatusBadRequest, "InvalidRequestContent")
	azureResourcesExpectErrorCode(t, do(http.MethodPost, service+"/checkNameAvailability"+azureDMSTestAPI, `{"name":"x","type":"widgets"}`), http.StatusBadRequest, "InvalidRequestContent")
}
