package server

import (
	"net/http"
	"testing"
)

const azureMySQLTestAPIVersion = "?api-version=2023-12-30"

// PostgreSQL tests cover the shared flexible server rules in depth; these
// cover what differs for MySQL.
func TestAzureMySQLFlexibleServerLifecycle(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-mysql")
	server := sub + "/resourceGroups/rg-mysql/providers/Microsoft.DBforMySQL/flexibleServers/shop-db"

	resp := do(http.MethodPut, server+azureMySQLTestAPIVersion, `{"location":"eastus","sku":{"name":"Standard_B1ms","tier":"Burstable"},"properties":{"administratorLogin":"shopadmin","administratorLoginPassword":"S3cret-Passw0rd"}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	props, _ := providerContractJSONMap(t, resp)["properties"].(map[string]any)
	if props["fullyQualifiedDomainName"] != "shop-db.mysql.database.azure.com" || props["version"] != "8.0.21" || props["administratorLoginPassword"] != nil {
		t.Fatalf("unexpected server properties: %#v", props)
	}

	resp = do(http.MethodGet, server+"/databases"+azureMySQLTestAPIVersion, "")
	if items, _ := providerContractJSONMap(t, resp)["value"].([]any); len(items) != 4 {
		t.Fatalf("expected the four system databases, got %#v", items)
	}
	resp = do(http.MethodPut, server+"/databases/orders"+azureMySQLTestAPIVersion, `{"properties":{"charset":"utf8mb3"}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	if dbProps, _ := providerContractJSONMap(t, resp)["properties"].(map[string]any); dbProps["charset"] != "utf8mb3" || dbProps["collation"] != "utf8mb4_0900_ai_ci" {
		t.Fatalf("unexpected database properties: %#v", dbProps)
	}

	azureResourcesExpectStatus(t, do(http.MethodPost, server+"/stop"+azureMySQLTestAPIVersion, ""), http.StatusOK)
	if state := azureDatabaseTestState(t, do(http.MethodGet, server+azureMySQLTestAPIVersion, "")); state != "Stopped" {
		t.Fatalf("expected Stopped, got %v", state)
	}

	cases := []struct{ name, body string }{
		{"postgres version", `{"location":"eastus","sku":{"name":"Standard_B1ms","tier":"Burstable"},"properties":{"version":"16","administratorLogin":"shopadmin","administratorLoginPassword":"x"}}`},
		{"reserved login", `{"location":"eastus","sku":{"name":"Standard_B1ms","tier":"Burstable"},"properties":{"administratorLogin":"sa","administratorLoginPassword":"x"}}`},
	}
	for _, tc := range cases {
		resp := do(http.MethodPut, sub+"/resourceGroups/rg-mysql/providers/Microsoft.DBforMySQL/flexibleServers/other-db"+azureMySQLTestAPIVersion, tc.body)
		t.Run(tc.name, func(t *testing.T) {
			azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, "InvalidRequestContent")
		})
	}

	resp = do(http.MethodPost, sub+"/providers/Microsoft.DBforMySQL/locations/eastus/checkNameAvailability"+azureMySQLTestAPIVersion, `{"name":"shop-db","type":"Microsoft.DBforMySQL/flexibleServers"}`)
	if got := providerContractJSONMap(t, resp); got["reason"] != "AlreadyExists" {
		t.Fatalf("expected AlreadyExists, got %#v", got)
	}
	azureResourcesExpectStatus(t, do(http.MethodDelete, sub+"/resourcegroups/rg-mysql?api-version=2021-04-01", ""), http.StatusOK)
	azureResourcesExpectErrorCode(t, do(http.MethodGet, server+azureMySQLTestAPIVersion, ""), http.StatusNotFound, "ResourceGroupNotFound")
}
