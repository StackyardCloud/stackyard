package server

import (
	"net/http"
	"strings"
	"testing"
)

const azurePostgreSQLTestAPIVersion = "?api-version=2024-08-01"

const azurePostgreSQLTestServerBody = `{"location":"eastus","sku":{"name":"Standard_B1ms","tier":"Burstable"},"properties":{"administratorLogin":"pgadmin","administratorLoginPassword":"S3cret-Passw0rd","version":"16"}}`

func newAzureDatabaseTestServer(t *testing.T, rg string) (string, func(method, path, body string) *http.Response) {
	t.Helper()
	ts := newAzureResourcesTestServer(t)
	headers := azureResourcesTestHeaders()
	sub := "/azure/subscriptions/" + azureResourcesTestSubscription
	do := func(method, path, body string) *http.Response {
		var payload []byte
		if body != "" {
			payload = []byte(body)
		}
		return providerContractRequest(t, ts, method, path, payload, headers)
	}
	azureResourcesExpectStatus(t, do(http.MethodPut, sub+"/resourcegroups/"+rg+"?api-version=2021-04-01", `{"location":"eastus"}`), http.StatusCreated)
	return sub, do
}

func azureDatabaseTestState(t *testing.T, resp *http.Response) any {
	t.Helper()
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	props, _ := providerContractJSONMap(t, resp)["properties"].(map[string]any)
	return props["state"]
}

func TestAzurePostgreSQLFlexibleServerLifecycle(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-pg")
	server := sub + "/resourceGroups/rg-pg/providers/Microsoft.DBforPostgreSQL/flexibleServers/orders-db"

	resp := do(http.MethodPut, server+azurePostgreSQLTestAPIVersion, azurePostgreSQLTestServerBody)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	props, _ := providerContractJSONMap(t, resp)["properties"].(map[string]any)
	if props["fullyQualifiedDomainName"] != "orders-db.postgres.database.azure.com" || props["state"] != "Ready" || props["administratorLogin"] != "pgadmin" {
		t.Fatalf("unexpected server properties: %#v", props)
	}
	if _, leaked := props["administratorLoginPassword"]; leaked {
		t.Fatalf("password must not be returned: %#v", props)
	}

	resp = do(http.MethodGet, server+"/databases"+azurePostgreSQLTestAPIVersion, "")
	if items, _ := providerContractJSONMap(t, resp)["value"].([]any); len(items) != 3 {
		t.Fatalf("expected the three system databases, got %#v", items)
	}
	resp = do(http.MethodPut, server+"/databases/orders"+azurePostgreSQLTestAPIVersion, `{"properties":{}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	if dbProps, _ := providerContractJSONMap(t, resp)["properties"].(map[string]any); dbProps["charset"] != "UTF8" || dbProps["collation"] != "en_US.utf8" {
		t.Fatalf("unexpected database defaults: %#v", dbProps)
	}
	azureResourcesExpectStatus(t, do(http.MethodPut, server+"/firewallRules/office"+azurePostgreSQLTestAPIVersion, `{"properties":{"startIpAddress":"10.0.0.1","endIpAddress":"10.0.0.255"}}`), http.StatusCreated)

	// Stopped servers stay stopped across updates until started.
	azureResourcesExpectStatus(t, do(http.MethodPost, server+"/stop"+azurePostgreSQLTestAPIVersion, ""), http.StatusOK)
	if state := azureDatabaseTestState(t, do(http.MethodPatch, server+azurePostgreSQLTestAPIVersion, `{"tags":{"env":"test"},"properties":{"administratorLoginPassword":"Rotated-Passw0rd"}}`)); state != "Stopped" {
		t.Fatalf("expected Stopped after PATCH, got %v", state)
	}
	// Only start and stop change the state.
	if state := azureDatabaseTestState(t, do(http.MethodPatch, server+azurePostgreSQLTestAPIVersion, `{"properties":{"state":"Ready"}}`)); state != "Stopped" {
		t.Fatalf("a PATCH must not set the state, got %v", state)
	}
	resp = do(http.MethodGet, server+azurePostgreSQLTestAPIVersion, "")
	if got := providerContractJSONMap(t, resp)["properties"].(map[string]any); got["administratorLoginPassword"] != nil {
		t.Fatalf("PATCHed password must not be returned: %#v", got)
	}
	azureResourcesExpectStatus(t, do(http.MethodPost, server+"/start"+azurePostgreSQLTestAPIVersion, ""), http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodPost, server+"/restart"+azurePostgreSQLTestAPIVersion, `{"restartWithFailover":false}`), http.StatusOK)
	if state := azureDatabaseTestState(t, do(http.MethodGet, server+azurePostgreSQLTestAPIVersion, "")); state != "Ready" {
		t.Fatalf("expected Ready after start, got %v", state)
	}

	azureResourcesExpectStatus(t, do(http.MethodDelete, server+azurePostgreSQLTestAPIVersion, ""), http.StatusOK)
	azureResourcesExpectErrorCode(t, do(http.MethodGet, server+"/databases/orders"+azurePostgreSQLTestAPIVersion, ""), http.StatusNotFound, "ParentResourceNotFound")
	azureResourcesExpectErrorCode(t, do(http.MethodPost, server+"/start"+azurePostgreSQLTestAPIVersion, ""), http.StatusNotFound, "ResourceNotFound")
}

func TestAzurePostgreSQLValidation(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-pg")
	servers := sub + "/resourceGroups/rg-pg/providers/Microsoft.DBforPostgreSQL/flexibleServers/"
	sku := `"sku":{"name":"Standard_B1ms","tier":"Burstable"}`
	cases := []struct{ name, path, body, code string }{
		{"uppercase name", servers + "Orders", azurePostgreSQLTestServerBody, "InvalidResourceName"},
		{"trailing hyphen", servers + "orders-", azurePostgreSQLTestServerBody, "InvalidResourceName"},
		{"missing sku", servers + "orders", `{"location":"eastus","properties":{"administratorLogin":"pgadmin","administratorLoginPassword":"x"}}`, "InvalidRequestContent"},
		{"bad tier", servers + "orders", `{"location":"eastus","sku":{"name":"Standard_B1ms","tier":"Basic"},"properties":{"administratorLogin":"pgadmin","administratorLoginPassword":"x"}}`, "InvalidRequestContent"},
		{"bad version", servers + "orders", `{"location":"eastus",` + sku + `,"properties":{"version":"9.6","administratorLogin":"pgadmin","administratorLoginPassword":"x"}}`, "InvalidRequestContent"},
		{"missing password", servers + "orders", `{"location":"eastus",` + sku + `,"properties":{"administratorLogin":"pgadmin"}}`, "InvalidRequestContent"},
		{"reserved login", servers + "orders", `{"location":"eastus",` + sku + `,"properties":{"administratorLogin":"azure_superuser","administratorLoginPassword":"x"}}`, "InvalidRequestContent"},
		{"pg_ login", servers + "orders", `{"location":"eastus",` + sku + `,"properties":{"administratorLogin":"pg_owner","administratorLoginPassword":"x"}}`, "InvalidRequestContent"},
	}
	for _, tc := range cases {
		resp := do(http.MethodPut, tc.path+azurePostgreSQLTestAPIVersion, tc.body)
		t.Run(tc.name, func(t *testing.T) { azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, tc.code) })
	}
	// Entra-only servers and point-in-time restores need no password.
	azureResourcesExpectStatus(t, do(http.MethodPut, servers+"entra-only"+azurePostgreSQLTestAPIVersion, `{"location":"eastus",`+sku+`,"properties":{"authConfig":{"activeDirectoryAuth":"Enabled","passwordAuth":"Disabled"}}}`), http.StatusCreated)
	azureResourcesExpectStatus(t, do(http.MethodPut, servers+"restored"+azurePostgreSQLTestAPIVersion, `{"location":"eastus",`+sku+`,"properties":{"createMode":"PointInTimeRestore","sourceServerResourceId":"x"}}`), http.StatusCreated)

	azureResourcesExpectStatus(t, do(http.MethodPut, servers+"orders"+azurePostgreSQLTestAPIVersion, azurePostgreSQLTestServerBody), http.StatusCreated)
	for _, body := range []string{
		`{"properties":{"startIpAddress":"10.0.0.9","endIpAddress":"10.0.0.1"}}`,
		`{"properties":{"startIpAddress":"10.0.0.1","endIpAddress":"not-an-ip"}}`,
		`{"properties":{"startIpAddress":"::1","endIpAddress":"::1"}}`,
	} {
		azureResourcesExpectErrorCode(t, do(http.MethodPut, servers+"orders/firewallRules/bad"+azurePostgreSQLTestAPIVersion, body), http.StatusBadRequest, "InvalidRequestContent")
	}
}

func TestAzurePostgreSQLCheckNameAvailability(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-pg")
	check := func(path, name string) map[string]any {
		resp := do(http.MethodPost, sub+"/providers/Microsoft.DBforPostgreSQL/"+path+azurePostgreSQLTestAPIVersion, `{"name":"`+name+`","type":"Microsoft.DBforPostgreSQL/flexibleServers"}`)
		azureResourcesExpectStatus(t, resp, http.StatusOK)
		return providerContractJSONMap(t, resp)
	}
	if got := check("checkNameAvailability", "orders-db"); got["nameAvailable"] != true {
		t.Fatalf("expected available, got %#v", got)
	}
	azureResourcesExpectStatus(t, do(http.MethodPut, sub+"/resourceGroups/rg-pg/providers/Microsoft.DBforPostgreSQL/flexibleServers/orders-db"+azurePostgreSQLTestAPIVersion, strings.Replace(azurePostgreSQLTestServerBody, "eastus", "westus", 1)), http.StatusCreated)
	// Host names are global, so the location-scoped check sees a server in another region.
	for _, path := range []string{"checkNameAvailability", "locations/eastus/checkNameAvailability"} {
		if got := check(path, "orders-db"); got["nameAvailable"] != false || got["reason"] != "AlreadyExists" {
			t.Fatalf("%s: expected AlreadyExists, got %#v", path, got)
		}
	}
	if got := check("locations/eastus/checkNameAvailability", "Orders"); got["reason"] != "Invalid" {
		t.Fatalf("expected Invalid, got %#v", got)
	}
	// MySQL names live in their own DNS zone.
	resp := do(http.MethodPost, sub+"/providers/Microsoft.DBforMySQL/locations/eastus/checkNameAvailability?api-version=2023-12-30", `{"name":"orders-db","type":"Microsoft.DBforMySQL/flexibleServers"}`)
	if got := providerContractJSONMap(t, resp); got["nameAvailable"] != true {
		t.Fatalf("expected MySQL name to stay available, got %#v", got)
	}
	azureResourcesExpectErrorCode(t, do(http.MethodPost, sub+"/providers/Microsoft.DBforPostgreSQL/checkNameAvailability"+azurePostgreSQLTestAPIVersion, `{}`), http.StatusBadRequest, "InvalidRequestContent")
}
