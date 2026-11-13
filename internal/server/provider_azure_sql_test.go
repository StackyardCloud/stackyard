package server

import (
	"net/http"
	"testing"
)

const azureSQLTestAPIVersion = "?api-version=2021-11-01"

const azureSQLTestServerBody = `{"location":"eastus","properties":{"administratorLogin":"sqladmin","administratorLoginPassword":"S3cret-Passw0rd","minimalTlsVersion":"1.2"}}`

func TestAzureSQLLifecycle(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-sql")
	server := sub + "/resourceGroups/rg-sql/providers/Microsoft.Sql/servers/orders-sql"

	resp := do(http.MethodPut, server+azureSQLTestAPIVersion, azureSQLTestServerBody)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	props, _ := providerContractJSONMap(t, resp)["properties"].(map[string]any)
	if props["fullyQualifiedDomainName"] != "orders-sql.database.windows.net" || props["version"] != "12.0" || props["state"] != "Ready" || props["administratorLoginPassword"] != nil {
		t.Fatalf("unexpected server properties: %#v", props)
	}

	// Every server has a master database in its region.
	resp = do(http.MethodGet, server+"/databases/master"+azureSQLTestAPIVersion, "")
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if master := providerContractJSONMap(t, resp); master["location"] != "eastus" || master["sku"].(map[string]any)["tier"] != "System" {
		t.Fatalf("unexpected master database: %#v", master)
	}

	database := server + "/databases/orders"
	azureResourcesExpectErrorCode(t, do(http.MethodPut, database+azureSQLTestAPIVersion, `{"sku":{"name":"S0"}}`), http.StatusBadRequest, "LocationRequired")
	resp = do(http.MethodPut, database+azureSQLTestAPIVersion, `{"location":"eastus","sku":{"name":"S0","tier":"Standard"}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	dbProps, _ := providerContractJSONMap(t, resp)["properties"].(map[string]any)
	databaseID, _ := dbProps["databaseId"].(string)
	if dbProps["status"] != "Online" || dbProps["currentServiceObjectiveName"] != "S0" || dbProps["collation"] != "SQL_Latin1_General_CP1_CI_AS" || len(databaseID) != 36 {
		t.Fatalf("unexpected database properties: %#v", dbProps)
	}
	resp = do(http.MethodPut, database+azureSQLTestAPIVersion, `{"location":"eastus","sku":{"name":"S1","tier":"Standard"}}`)
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if got := providerContractJSONMap(t, resp)["properties"].(map[string]any); got["databaseId"] != databaseID || got["currentServiceObjectiveName"] != "S1" {
		t.Fatalf("databaseId must be stable and the objective follow the SKU: %#v", got)
	}

	resp = do(http.MethodPut, server+"/elasticPools/shared"+azureSQLTestAPIVersion, `{"location":"eastus","sku":{"name":"StandardPool","tier":"Standard","capacity":50}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	if got := providerContractJSONMap(t, resp)["properties"].(map[string]any); got["state"] != "Ready" {
		t.Fatalf("unexpected pool: %#v", got)
	}
	azureResourcesExpectStatus(t, do(http.MethodPut, server+"/firewallRules/AllowAllWindowsAzureIps"+azureSQLTestAPIVersion, `{"properties":{"startIpAddress":"0.0.0.0","endIpAddress":"0.0.0.0"}}`), http.StatusCreated)

	resp = do(http.MethodGet, server+"/databases"+azureSQLTestAPIVersion, "")
	if items, _ := providerContractJSONMap(t, resp)["value"].([]any); len(items) != 2 {
		t.Fatalf("expected master and orders, got %#v", items)
	}
	azureResourcesExpectStatus(t, do(http.MethodDelete, server+azureSQLTestAPIVersion, ""), http.StatusOK)
	azureResourcesExpectErrorCode(t, do(http.MethodGet, database+azureSQLTestAPIVersion, ""), http.StatusNotFound, "ParentResourceNotFound")
}

func TestAzureSQLValidation(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-sql")
	servers := sub + "/resourceGroups/rg-sql/providers/Microsoft.Sql/servers/"
	cases := []struct{ name, path, body, code string }{
		{"uppercase server", servers + "Orders", azureSQLTestServerBody, "InvalidResourceName"},
		{"leading hyphen", servers + "-orders", azureSQLTestServerBody, "InvalidResourceName"},
		{"missing password", servers + "orders", `{"location":"eastus","properties":{"administratorLogin":"sqladmin"}}`, "InvalidRequestContent"},
		{"reserved login", servers + "orders", `{"location":"eastus","properties":{"administratorLogin":"SA","administratorLoginPassword":"x"}}`, "InvalidRequestContent"},
		{"bad tls", servers + "orders", `{"location":"eastus","properties":{"administratorLogin":"sqladmin","administratorLoginPassword":"x","minimalTlsVersion":"2.0"}}`, "InvalidRequestContent"},
	}
	for _, tc := range cases {
		resp := do(http.MethodPut, tc.path+azureSQLTestAPIVersion, tc.body)
		t.Run(tc.name, func(t *testing.T) { azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, tc.code) })
	}
	// Entra-only servers need no SQL administrator.
	azureResourcesExpectStatus(t, do(http.MethodPut, servers+"entra-only"+azureSQLTestAPIVersion, `{"location":"eastus","properties":{"administrators":{"azureADOnlyAuthentication":true,"login":"dba-group","sid":"00000000-0000-0000-0000-000000000001"}}}`), http.StatusCreated)
	for _, name := range []string{"bad%25name", "bad*name", "trailing."} {
		azureResourcesExpectErrorCode(t, do(http.MethodPut, servers+"entra-only/databases/"+name+azureSQLTestAPIVersion, `{"location":"eastus"}`), http.StatusBadRequest, "InvalidResourceName")
	}
	azureResourcesExpectErrorCode(t, do(http.MethodPut, servers+"entra-only/firewallRules/bad"+azureSQLTestAPIVersion, `{"properties":{"startIpAddress":"10.0.0.2","endIpAddress":"10.0.0.1"}}`), http.StatusBadRequest, "InvalidRequestContent")
}

func TestAzureSQLCheckNameAvailability(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-sql")
	check := func(name string) map[string]any {
		resp := do(http.MethodPost, sub+"/providers/Microsoft.Sql/checkNameAvailability"+azureSQLTestAPIVersion, `{"name":"`+name+`","type":"Microsoft.Sql/servers"}`)
		azureResourcesExpectStatus(t, resp, http.StatusOK)
		return providerContractJSONMap(t, resp)
	}
	if got := check("orders-sql"); got["available"] != true {
		t.Fatalf("expected available, got %#v", got)
	}
	azureResourcesExpectStatus(t, do(http.MethodPut, sub+"/resourceGroups/rg-sql/providers/Microsoft.Sql/servers/orders-sql"+azureSQLTestAPIVersion, azureSQLTestServerBody), http.StatusCreated)
	if got := check("orders-sql"); got["available"] != false || got["reason"] != "AlreadyExists" {
		t.Fatalf("expected AlreadyExists, got %#v", got)
	}
	if got := check("Orders_SQL"); got["reason"] != "Invalid" {
		t.Fatalf("expected Invalid, got %#v", got)
	}
	azureResourcesExpectErrorCode(t, do(http.MethodPost, sub+"/providers/Microsoft.Sql/checkNameAvailability"+azureSQLTestAPIVersion, `{}`), http.StatusBadRequest, "InvalidRequestContent")
}
