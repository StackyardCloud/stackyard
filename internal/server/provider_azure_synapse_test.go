package server

import (
	"net/http"
	"testing"
)

const azureSynapseTestAPIVersion = "?api-version=2021-06-01"

const azureSynapseTestWorkspace = `{"location":"eastus","identity":{"type":"SystemAssigned"},"properties":{"defaultDataLakeStorage":{"accountUrl":"https://lake.dfs.core.windows.net","filesystem":"synapse"},"sqlAdministratorLogin":"sqladmin","sqlAdministratorLoginPassword":"Secret-123"}}`

func TestAzureSynapseLifecycle(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-syn")
	ws := sub + "/resourceGroups/rg-syn/providers/Microsoft.Synapse/workspaces/analytics"

	resp := do(http.MethodPut, ws+azureSynapseTestAPIVersion, azureSynapseTestWorkspace)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	props := providerContractJSONMap(t, resp)["properties"].(map[string]any)
	endpoints, _ := props["connectivityEndpoints"].(map[string]any)
	if endpoints["sql"] != "analytics.sql.azuresynapse.net" || endpoints["sqlOnDemand"] != "analytics-ondemand.sql.azuresynapse.net" || props["sqlAdministratorLoginPassword"] != nil || props["workspaceUID"] == nil {
		t.Fatalf("unexpected workspace properties: %#v", props)
	}

	azureResourcesExpectStatus(t, do(http.MethodPut, ws+"/firewallRules/allowAll"+azureSynapseTestAPIVersion, `{"properties":{"startIpAddress":"0.0.0.0","endIpAddress":"255.255.255.255"}}`), http.StatusCreated)
	azureResourcesExpectErrorCode(t, do(http.MethodPut, ws+"/firewallRules/bad"+azureSynapseTestAPIVersion, `{"properties":{"startIpAddress":"10.0.0.9","endIpAddress":"10.0.0.1"}}`), http.StatusBadRequest, "InvalidRequestContent")

	pool := ws + "/sqlPools/dw"
	resp = do(http.MethodPut, pool+azureSynapseTestAPIVersion, `{"location":"eastus","sku":{"name":"DW100c"}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	if props := providerContractJSONMap(t, resp)["properties"].(map[string]any); props["status"] != "Online" || props["collation"] != "SQL_Latin1_General_CP1_CI_AS" {
		t.Fatalf("unexpected SQL pool: %#v", props)
	}
	azureResourcesExpectStatus(t, do(http.MethodPost, pool+"/pause"+azureSynapseTestAPIVersion, ""), http.StatusOK)
	// A PATCH keeps the pool paused.
	resp = do(http.MethodPatch, pool+azureSynapseTestAPIVersion, `{"tags":{"env":"dev"}}`)
	if got := providerContractJSONMap(t, resp)["properties"].(map[string]any)["status"]; got != "Paused" {
		t.Fatalf("expected Paused, got %v", got)
	}
	// Neither a PATCH nor a PUT can set the status.
	resp = do(http.MethodPatch, pool+azureSynapseTestAPIVersion, `{"properties":{"status":"Online"}}`)
	if got := providerContractJSONMap(t, resp)["properties"].(map[string]any)["status"]; got != "Paused" {
		t.Fatalf("a PATCH must not resume the pool, got %v", got)
	}
	resp = do(http.MethodPut, pool+azureSynapseTestAPIVersion, `{"location":"eastus","sku":{"name":"DW200c"}}`)
	if got := providerContractJSONMap(t, resp)["properties"].(map[string]any)["status"]; got != "Paused" {
		t.Fatalf("a PUT must not resume the pool, got %v", got)
	}
	azureResourcesExpectStatus(t, do(http.MethodPost, pool+"/resume"+azureSynapseTestAPIVersion, ""), http.StatusOK)
	resp = do(http.MethodGet, pool+azureSynapseTestAPIVersion, "")
	if got := providerContractJSONMap(t, resp)["properties"].(map[string]any)["status"]; got != "Online" {
		t.Fatalf("expected Online after resume, got %v", got)
	}

	spark := ws + "/bigDataPools/spark"
	azureResourcesExpectStatus(t, do(http.MethodPut, spark+azureSynapseTestAPIVersion, `{"location":"eastus","properties":{"nodeSize":"Small","nodeSizeFamily":"MemoryOptimized","sparkVersion":"3.4","nodeCount":3,"autoPause":{"enabled":true,"delayInMinutes":15}}}`), http.StatusCreated)

	azureResourcesExpectStatus(t, do(http.MethodDelete, ws+azureSynapseTestAPIVersion, ""), http.StatusOK)
	azureResourcesExpectErrorCode(t, do(http.MethodGet, spark+azureSynapseTestAPIVersion, ""), http.StatusNotFound, "ParentResourceNotFound")
}

func TestAzureSynapseValidation(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-syn")
	workspaces := sub + "/resourceGroups/rg-syn/providers/Microsoft.Synapse/workspaces/"
	cases := []struct{ name, path, body, code string }{
		{"uppercase name", workspaces + "Analytics", azureSynapseTestWorkspace, "InvalidResourceName"},
		{"ondemand suffix", workspaces + "lake-ondemand", azureSynapseTestWorkspace, "InvalidResourceName"},
		{"no storage", workspaces + "analytics", `{"location":"eastus","properties":{}}`, "InvalidRequestContent"},
	}
	for _, tc := range cases {
		resp := do(http.MethodPut, tc.path+azureSynapseTestAPIVersion, tc.body)
		t.Run(tc.name, func(t *testing.T) { azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, tc.code) })
	}

	azureResourcesExpectStatus(t, do(http.MethodPut, workspaces+"analytics"+azureSynapseTestAPIVersion, azureSynapseTestWorkspace), http.StatusCreated)
	children := []struct{ name, path, body string }{
		{"sql pool sku", workspaces + "analytics/sqlPools/dw", `{"location":"eastus","sku":{"name":"Basic"}}`},
		{"spark node size", workspaces + "analytics/bigDataPools/spark", `{"location":"eastus","properties":{"nodeSize":"Tiny"}}`},
		{"spark version", workspaces + "analytics/bigDataPools/spark", `{"location":"eastus","properties":{"sparkVersion":"2.4"}}`},
		{"spark node count", workspaces + "analytics/bigDataPools/spark", `{"location":"eastus","properties":{"nodeCount":2}}`},
	}
	for _, tc := range children {
		resp := do(http.MethodPut, tc.path+azureSynapseTestAPIVersion, tc.body)
		t.Run(tc.name, func(t *testing.T) {
			azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, "InvalidRequestContent")
		})
	}

	check := func(name string) map[string]any {
		resp := do(http.MethodPost, sub+"/providers/Microsoft.Synapse/checkNameAvailability"+azureSynapseTestAPIVersion, `{"name":"`+name+`","type":"Microsoft.Synapse/workspaces"}`)
		azureResourcesExpectStatus(t, resp, http.StatusOK)
		return providerContractJSONMap(t, resp)
	}
	if got := check("analytics"); got["available"] != false || got["reason"] != "AlreadyExists" {
		t.Fatalf("expected AlreadyExists, got %#v", got)
	}
	if got := check("other"); got["available"] != true {
		t.Fatalf("expected available, got %#v", got)
	}
	if got := check("Bad_Name"); got["reason"] != "Invalid" {
		t.Fatalf("expected Invalid, got %#v", got)
	}
}
