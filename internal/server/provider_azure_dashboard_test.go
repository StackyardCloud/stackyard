package server

import (
	"net/http"
	"strings"
	"testing"
)

const (
	azureGrafanaTestAPIVersion = "?api-version=2025-08-01"
	azureGrafanaTestBody       = `{"location":"eastus","sku":{"name":"Standard"},"identity":{"type":"SystemAssigned"},"properties":{"apiKey":"Enabled","deterministicOutboundIP":"Enabled","grafanaMajorVersion":"11","grafanaConfigurations":{"smtp":{"enabled":true,"host":"smtp.contoso.com:587","user":"grafana","password":"S3cret!","fromAddress":"grafana@contoso.com","startTLSPolicy":"MandatoryStartTLS"}}}}`
)

func TestAzureGrafanaLifecycle(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-grafana")
	ids := strings.TrimPrefix(sub, "/azure") + "/resourceGroups/rg-grafana/providers"
	grafana := sub + "/resourceGroups/rg-grafana/providers/Microsoft.Dashboard/grafana/ops-grafana"

	resp := do(http.MethodPut, grafana+azureGrafanaTestAPIVersion, azureGrafanaTestBody)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	props := providerContractJSONMap(t, resp)["properties"].(map[string]any)
	if !strings.HasPrefix(props["endpoint"].(string), "https://ops-grafana-") || props["grafanaVersion"] != "11.6.3" || len(props["outboundIPs"].([]any)) != 2 || props["publicNetworkAccess"] != "Enabled" {
		t.Fatalf("unexpected workspace properties: %#v", props)
	}
	if smtp := props["grafanaConfigurations"].(map[string]any)["smtp"].(map[string]any); smtp["password"] != nil || smtp["user"] != "grafana" {
		t.Fatalf("the SMTP password was returned: %#v", smtp)
	}
	resp = do(http.MethodGet, grafana+azureGrafanaTestAPIVersion, "")
	if smtp := providerContractJSONMap(t, resp)["properties"].(map[string]any)["grafanaConfigurations"].(map[string]any)["smtp"].(map[string]any); smtp["password"] != nil {
		t.Fatal("the stored SMTP password was returned")
	}
	resp = do(http.MethodPatch, grafana+azureGrafanaTestAPIVersion, `{"properties":{"grafanaMajorVersion":"12"}}`)
	if got := providerContractJSONMap(t, resp)["properties"].(map[string]any)["grafanaVersion"]; got != "12.0.2" {
		t.Fatalf("expected the upgraded version, got %v", got)
	}
	resp = do(http.MethodPost, grafana+"/fetchAvailablePlugins"+azureGrafanaTestAPIVersion, "")
	if got := providerContractJSONMap(t, resp)["value"].([]any); len(got) == 0 {
		t.Fatal("expected available plugins")
	}

	mpe := grafana + "/managedPrivateEndpoints/sql"
	resp = do(http.MethodPut, mpe+azureGrafanaTestAPIVersion, `{"location":"eastus","properties":{"privateLinkResourceId":"`+ids+`/Microsoft.Sql/servers/orders","groupIds":["sqlServer"],"requestMessage":"Grafana dashboards"}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	if got := providerContractJSONMap(t, resp)["properties"].(map[string]any)["connectionState"].(map[string]any)["status"]; got != "Pending" {
		t.Fatalf("expected a pending connection, got %v", got)
	}
	azureResourcesExpectStatus(t, do(http.MethodPost, grafana+"/refreshManagedPrivateEndpoints"+azureGrafanaTestAPIVersion, ""), http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodPut, grafana+"/integrationFabrics/aks"+azureGrafanaTestAPIVersion, `{"location":"eastus","properties":{"targetResourceId":"`+ids+`/Microsoft.ContainerService/managedClusters/web","dataSourceResourceId":"`+ids+`/Microsoft.Monitor/accounts/prom","scenarios":["ContainerInsights"]}}`), http.StatusCreated)
	resp = do(http.MethodGet, grafana+"/privateLinkResources/grafana"+azureGrafanaTestAPIVersion, "")
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodPut, sub+"/resourceGroups/rg-grafana/providers/Microsoft.Dashboard/dashboards/team-board"+azureGrafanaTestAPIVersion, `{"location":"eastus","properties":{}}`), http.StatusCreated)

	azureResourcesExpectStatus(t, do(http.MethodDelete, grafana+azureGrafanaTestAPIVersion, ""), http.StatusOK)
	azureResourcesExpectErrorCode(t, do(http.MethodGet, mpe+azureGrafanaTestAPIVersion, ""), http.StatusNotFound, "ParentResourceNotFound")
}

func TestAzureGrafanaValidation(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-grafana")
	group := sub + "/resourceGroups/rg-grafana/providers/Microsoft.Dashboard/grafana"
	cases := []struct{ name, old, new string }{
		{"sku", `"sku":{"name":"Standard"}`, `"sku":{"name":"Premium"}`},
		{"size", `"sku":{"name":"Standard"}`, `"sku":{"name":"Essential","size":"X1"}`},
		{"api key", `"apiKey":"Enabled"`, `"apiKey":"On"`},
		{"major version", `"grafanaMajorVersion":"11"`, `"grafanaMajorVersion":"9"`},
		{"smtp host", `"host":"smtp.contoso.com:587",`, ``},
		{"tls policy", `"startTLSPolicy":"MandatoryStartTLS"`, `"startTLSPolicy":"Always"`},
		{"monitor workspace", `"apiKey":"Enabled"`, `"apiKey":"Enabled","grafanaIntegrations":{"azureMonitorWorkspaceIntegrations":[{"azureMonitorWorkspaceResourceId":"prom"}]}`},
	}
	for _, tc := range cases {
		body := strings.Replace(azureGrafanaTestBody, tc.old, tc.new, 1)
		if body == azureGrafanaTestBody {
			t.Fatalf("case %s did not change the body", tc.name)
		}
		resp := do(http.MethodPut, group+"/g1"+azureGrafanaTestAPIVersion, body)
		t.Run(tc.name, func(t *testing.T) {
			azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, "InvalidRequestContent")
		})
	}
	azureResourcesExpectErrorCode(t, do(http.MethodPut, group+"/1grafana"+azureGrafanaTestAPIVersion, azureGrafanaTestBody), http.StatusBadRequest, "InvalidResourceName")
	azureResourcesExpectStatus(t, do(http.MethodPut, group+"/g1"+azureGrafanaTestAPIVersion, azureGrafanaTestBody), http.StatusCreated)
	children := []struct{ name, path, body string }{
		{"mpe target", "/g1/managedPrivateEndpoints/m1", `{"location":"eastus","properties":{"privateLinkResourceId":"sql","groupIds":["sqlServer"]}}`},
		{"mpe groups", "/g1/managedPrivateEndpoints/m1", `{"location":"eastus","properties":{"privateLinkResourceId":"/subscriptions/x/resourceGroups/rg/providers/Microsoft.Sql/servers/s"}}`},
		{"fabric scenarios", "/g1/integrationFabrics/f1", `{"location":"eastus","properties":{"targetResourceId":"/subscriptions/x/resourceGroups/rg/providers/Microsoft.ContainerService/managedClusters/a","dataSourceResourceId":"/subscriptions/x/resourceGroups/rg/providers/Microsoft.Monitor/accounts/p"}}`},
	}
	for _, tc := range children {
		resp := do(http.MethodPut, group+tc.path+azureGrafanaTestAPIVersion, tc.body)
		t.Run(tc.name, func(t *testing.T) {
			azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, "InvalidRequestContent")
		})
	}
}
