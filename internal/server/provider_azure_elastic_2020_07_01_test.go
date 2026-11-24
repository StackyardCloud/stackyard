package server

import (
	"net/http"
	"testing"
)

const elasTestAPI = "?api-version=" + azureElastic20200701APIVersion

func TestAzureElasticMonitorLifecycle(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-elastic")
	monitor := sub + "/resourceGroups/rg-elastic/providers/Microsoft.Elastic/monitors/m1"

	resp := do(http.MethodPut, monitor+elasTestAPI, `{"location":"eastus","sku":{"name":"free_Monthly"},"properties":{"userInfo":{"emailAddress":"a@b.c"}}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	props := providerContractJSONMap(t, resp)["properties"].(map[string]any)
	if props["monitoringStatus"] != "Enabled" || props["userInfo"] != nil || props["elasticProperties"] == nil {
		t.Fatalf("unexpected monitor properties: %#v", props)
	}

	// A tagged resource in the subscription is monitored unless excluded.
	azureResourcesExpectStatus(t, do(http.MethodPut, sub+"/resourceGroups/rg-elastic/providers/Microsoft.ElasticSan/elasticSans/s1"+esanTestAPI,
		`{"location":"eastus","tags":{"env":"prod"},"properties":{"sku":{"name":"Premium_LRS"},"baseSizeTiB":1,"extendedCapacitySizeTiB":0}}`), http.StatusCreated)
	azureResourcesExpectStatus(t, do(http.MethodPut, monitor+"/tagRules/default"+elasTestAPI,
		`{"properties":{"logRules":{"sendSubscriptionLogs":true,"filteringTags":[{"name":"env","value":"prod","action":"Include"}]}}}`), http.StatusCreated)
	resp = do(http.MethodPost, monitor+"/listMonitoredResources"+elasTestAPI, "")
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	value := providerContractJSONMap(t, resp)["value"].([]any)
	if len(value) != 1 || value[0].(map[string]any)["sendingLogs"] != "True" {
		t.Fatalf("unexpected monitored resources: %#v", value)
	}
	azureResourcesExpectStatus(t, do(http.MethodPut, monitor+"/tagRules/default"+elasTestAPI,
		`{"properties":{"logRules":{"filteringTags":[{"name":"env","value":"prod","action":"Exclude"}]}}}`), http.StatusOK)
	resp = do(http.MethodPost, monitor+"/listMonitoredResources"+elasTestAPI, "")
	if value := providerContractJSONMap(t, resp)["value"].([]any); len(value) != 0 {
		t.Fatalf("excluded resource still monitored: %#v", value)
	}

	vm := `/subscriptions/x/resourceGroups/y/providers/Microsoft.Compute/virtualMachines/vm1`
	azureResourcesExpectStatus(t, do(http.MethodPost, monitor+"/vmCollectionUpdate"+elasTestAPI, `{"operationName":"Add","vmResourceId":"`+vm+`"}`), http.StatusOK)
	resp = do(http.MethodPost, monitor+"/listVMHost"+elasTestAPI, "")
	if hosts := providerContractJSONMap(t, resp)["value"].([]any); len(hosts) != 1 || hosts[0].(map[string]any)["vmResourceId"] != vm {
		t.Fatalf("unexpected VM hosts: %#v", hosts)
	}
	azureResourcesExpectStatus(t, do(http.MethodPost, monitor+"/vmCollectionUpdate"+elasTestAPI, `{"operationName":"Delete","vmResourceId":"`+vm+`"}`), http.StatusOK)
	resp = do(http.MethodPost, monitor+"/listVMHost"+elasTestAPI, "")
	if hosts := providerContractJSONMap(t, resp)["value"].([]any); len(hosts) != 0 {
		t.Fatalf("VM not removed: %#v", hosts)
	}
	azureResourcesExpectErrorCode(t, do(http.MethodPost, monitor+"/vmCollectionUpdate"+elasTestAPI, `{"operationName":"Move","vmResourceId":"`+vm+`"}`), http.StatusBadRequest, "InvalidRequestContent")

	for _, action := range []string{"listDeploymentInfo", "vmIngestionDetails"} {
		azureResourcesExpectStatus(t, do(http.MethodPost, monitor+"/"+action+elasTestAPI, ""), http.StatusOK)
	}
	azureResourcesExpectStatus(t, do(http.MethodGet, "/azure/providers/Microsoft.Elastic/operations"+elasTestAPI, ""), http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodDelete, monitor+elasTestAPI, ""), http.StatusOK)
	azureResourcesExpectErrorCode(t, do(http.MethodGet, monitor+"/tagRules/default"+elasTestAPI, ""), http.StatusNotFound, "ParentResourceNotFound")
}

func TestAzureElasticValidation(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-elastic")
	monitor := sub + "/resourceGroups/rg-elastic/providers/Microsoft.Elastic/monitors/m1"
	azureResourcesExpectErrorCode(t, do(http.MethodPut, monitor+elasTestAPI, `{"location":"eastus","properties":{"monitoringStatus":"On"}}`), http.StatusBadRequest, "InvalidRequestContent")
	azureResourcesExpectErrorCode(t, do(http.MethodPut, monitor+elasTestAPI, `{"location":"eastus","properties":{"userInfo":{"emailAddress":"nope"}}}`), http.StatusBadRequest, "InvalidRequestContent")
	azureResourcesExpectStatus(t, do(http.MethodPut, monitor+elasTestAPI, `{"location":"eastus"}`), http.StatusCreated)
	azureResourcesExpectErrorCode(t, do(http.MethodPut, monitor+"/tagRules/custom"+elasTestAPI, `{}`), http.StatusBadRequest, "InvalidResourceName")
	azureResourcesExpectErrorCode(t, do(http.MethodPut, monitor+"/tagRules/default"+elasTestAPI,
		`{"properties":{"logRules":{"filteringTags":[{"name":"env","action":"Maybe"}]}}}`), http.StatusBadRequest, "InvalidRequestContent")
	azureResourcesExpectErrorCode(t, do(http.MethodPost, sub+"/resourceGroups/rg-elastic/providers/Microsoft.Elastic/monitors/missing/listVMHost"+elasTestAPI, ""), http.StatusNotFound, "ResourceNotFound")
}
