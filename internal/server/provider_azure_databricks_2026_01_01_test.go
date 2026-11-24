package server

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

const azureDBXTestVersion = "?api-version=" + azureDatabricks20260101APIVersion

func TestAzureDatabricks20260101Lifecycle(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-dbx")
	workspace := sub + "/resourceGroups/rg-dbx/providers/Microsoft.Databricks/workspaces/ws1"

	resp := do(http.MethodPut, workspace+azureDBXTestVersion, `{"location":"eastus","sku":{"name":"premium"},"properties":{"managedResourceGroupId":"/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/databricks-rg-ws1"}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	props := providerContractJSONMap(t, resp)["properties"].(map[string]any)
	url, _ := props["workspaceUrl"].(string)
	if props["workspaceId"] == "" || !strings.HasPrefix(url, "adb-") || !strings.HasSuffix(url, ".azuredatabricks.net") {
		t.Fatalf("unexpected workspace properties: %#v", props)
	}

	resp = do(http.MethodPut, workspace+"/virtualNetworkPeerings/peer1"+azureDBXTestVersion, `{"properties":{"remoteVirtualNetwork":{"id":"/subscriptions/x/resourceGroups/rg/providers/Microsoft.Network/virtualNetworks/hub"}}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	if got := providerContractJSONMap(t, resp)["properties"].(map[string]any)["peeringState"]; got != "Connected" {
		t.Fatalf("peering state = %v", got)
	}
	azureResourcesExpectStatus(t, do(http.MethodPut, workspace+"/privateEndpointConnections/pe1"+azureDBXTestVersion, `{"properties":{"privateLinkServiceConnectionState":{"status":"Approved"}}}`), http.StatusCreated)
	resp = do(http.MethodGet, workspace+"/privateLinkResources"+azureDBXTestVersion, "")
	if items := providerContractJSONMap(t, resp)["value"].([]any); len(items) != 2 {
		t.Fatalf("want 2 private link resources, got %d", len(items))
	}
	azureResourcesExpectStatus(t, do(http.MethodGet, workspace+"/privateLinkResources/databricks_ui_api"+azureDBXTestVersion, ""), http.StatusOK)

	resp = do(http.MethodGet, workspace+"/outboundNetworkDependenciesEndpoints"+azureDBXTestVersion, "")
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	raw, _ := io.ReadAll(resp.Body)
	var groups []map[string]any
	if err := json.Unmarshal(raw, &groups); err != nil || len(groups) == 0 {
		t.Fatalf("outbound endpoints must be a JSON array: %s (%v)", raw, err)
	}

	connector := sub + "/resourceGroups/rg-dbx/providers/Microsoft.Databricks/accessConnectors/ac1"
	azureResourcesExpectStatus(t, do(http.MethodPut, connector+azureDBXTestVersion, `{"location":"eastus","identity":{"type":"SystemAssigned"}}`), http.StatusCreated)
	resp = do(http.MethodGet, sub+"/providers/Microsoft.Databricks/accessConnectors"+azureDBXTestVersion, "")
	if items := providerContractJSONMap(t, resp)["value"].([]any); len(items) != 1 {
		t.Fatalf("want 1 access connector, got %d", len(items))
	}
	resp = do(http.MethodGet, "/azure/providers/Microsoft.Databricks/operations"+azureDBXTestVersion, "")
	if ops := providerContractJSONMap(t, resp)["value"].([]any); len(ops) == 0 {
		t.Fatal("operations list is empty")
	}

	azureResourcesExpectStatus(t, do(http.MethodDelete, workspace+azureDBXTestVersion, ""), http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodGet, workspace+"/outboundNetworkDependenciesEndpoints"+azureDBXTestVersion, ""), http.StatusNotFound)
}

func TestAzureDatabricks20260101Validation(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-dbx")
	base := sub + "/resourceGroups/rg-dbx/providers/Microsoft.Databricks/"
	cases := []struct{ name, path, body, code string }{
		{"bad sku", "workspaces/ws2", `{"location":"eastus","sku":{"name":"gold"},"properties":{"managedResourceGroupId":"/subscriptions/x/resourceGroups/m"}}`, "InvalidRequestContent"},
		{"no managed group", "workspaces/ws2", `{"location":"eastus","sku":{"name":"standard"},"properties":{}}`, "InvalidRequestContent"},
		{"bad name", "workspaces/w", `{"location":"eastus","sku":{"name":"standard"},"properties":{"managedResourceGroupId":"/subscriptions/x/resourceGroups/m"}}`, "InvalidResourceName"},
		{"bad identity", "accessConnectors/ac2", `{"location":"eastus","identity":{"type":"Magic"}}`, "InvalidRequestContent"},
	}
	for _, tc := range cases {
		resp := do(http.MethodPut, base+tc.path+azureDBXTestVersion, tc.body)
		t.Run(tc.name, func(t *testing.T) { azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, tc.code) })
	}
	azureResourcesExpectStatus(t, do(http.MethodGet, base+"workspaces/missing/outboundNetworkDependenciesEndpoints"+azureDBXTestVersion, ""), http.StatusNotFound)
}
