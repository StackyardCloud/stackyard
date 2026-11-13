package server

import (
	"net/http"
	"strings"
	"testing"
)

const azureMongoClusterTestAPIVersion = "?api-version=2024-07-01"

const azureMongoClusterTestBody = `{"location":"eastus","properties":{"administrator":{"userName":"mongoadmin","password":"S3cret-Passw0rd"},"serverVersion":"7.0","compute":{"tier":"M30"},"storage":{"sizeGb":128},"sharding":{"shardCount":1},"highAvailability":{"targetMode":"Disabled"}}}`

func TestAzureMongoClusterLifecycle(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-mongo")
	cluster := sub + "/resourceGroups/rg-mongo/providers/Microsoft.DocumentDB/mongoClusters/catalog"

	resp := do(http.MethodPut, cluster+azureMongoClusterTestAPIVersion, azureMongoClusterTestBody)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	props, _ := providerContractJSONMap(t, resp)["properties"].(map[string]any)
	admin, _ := props["administrator"].(map[string]any)
	if admin["userName"] != "mongoadmin" || admin["password"] != nil || props["clusterStatus"] != "Ready" {
		t.Fatalf("unexpected cluster properties: %#v", props)
	}
	if conn, _ := props["connectionString"].(string); !strings.HasPrefix(conn, "mongodb+srv://<user>:<password>@catalog.global.mongocluster.cosmos.azure.com/") {
		t.Fatalf("unexpected connection string: %q", conn)
	}

	resp = do(http.MethodPost, cluster+"/listConnectionStrings"+azureMongoClusterTestAPIVersion, "")
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if items, _ := providerContractJSONMap(t, resp)["connectionStrings"].([]any); len(items) != 1 || items[0].(map[string]any)["name"] != "GlobalReadWrite" {
		t.Fatalf("unexpected connection strings: %#v", items)
	}

	rule := cluster + "/firewallRules/office"
	azureResourcesExpectErrorCode(t, do(http.MethodPut, rule+azureMongoClusterTestAPIVersion, `{"properties":{"startIpAddress":"10.0.0.9","endIpAddress":"10.0.0.1"}}`), http.StatusBadRequest, "InvalidRequestContent")
	azureResourcesExpectStatus(t, do(http.MethodPut, rule+azureMongoClusterTestAPIVersion, `{"properties":{"startIpAddress":"10.0.0.1","endIpAddress":"10.0.0.255"}}`), http.StatusCreated)

	// PATCH merges into the nested administrator object without storing the password.
	resp = do(http.MethodPatch, cluster+azureMongoClusterTestAPIVersion, `{"properties":{"administrator":{"password":"Rotated-Passw0rd"},"compute":{"tier":"M40"}}}`)
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	props, _ = providerContractJSONMap(t, resp)["properties"].(map[string]any)
	if admin, _ := props["administrator"].(map[string]any); admin["userName"] != "mongoadmin" || admin["password"] != nil {
		t.Fatalf("unexpected administrator after PATCH: %#v", admin)
	}

	azureResourcesExpectStatus(t, do(http.MethodDelete, cluster+azureMongoClusterTestAPIVersion, ""), http.StatusOK)
	azureResourcesExpectErrorCode(t, do(http.MethodGet, rule+azureMongoClusterTestAPIVersion, ""), http.StatusNotFound, "ParentResourceNotFound")
}

func TestAzureMongoClusterValidation(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-mongo")
	clusters := sub + "/resourceGroups/rg-mongo/providers/Microsoft.DocumentDB/mongoClusters/"
	cases := []struct{ name, path, body string }{
		{"name too long", clusters + strings.Repeat("a", 41), azureMongoClusterTestBody},
		{"missing password", clusters + "catalog", `{"location":"eastus","properties":{"administrator":{"userName":"mongoadmin"}}}`},
		{"bad server version", clusters + "catalog", strings.Replace(azureMongoClusterTestBody, `"7.0"`, `"4.2"`, 1)},
		{"bad ha mode", clusters + "catalog", strings.Replace(azureMongoClusterTestBody, `"Disabled"`, `"Always"`, 1)},
	}
	for _, tc := range cases {
		resp := do(http.MethodPut, tc.path+azureMongoClusterTestAPIVersion, tc.body)
		code := "InvalidRequestContent"
		if tc.name == "name too long" {
			code = "InvalidResourceName"
		}
		t.Run(tc.name, func(t *testing.T) { azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, code) })
	}

	check := func(name string) map[string]any {
		resp := do(http.MethodPost, sub+"/providers/Microsoft.DocumentDB/locations/eastus/checkMongoClusterNameAvailability"+azureMongoClusterTestAPIVersion, `{"name":"`+name+`","type":"Microsoft.DocumentDB/mongoClusters"}`)
		azureResourcesExpectStatus(t, resp, http.StatusOK)
		return providerContractJSONMap(t, resp)
	}
	if got := check("catalog"); got["nameAvailable"] != true {
		t.Fatalf("expected available, got %#v", got)
	}
	azureResourcesExpectStatus(t, do(http.MethodPut, clusters+"catalog"+azureMongoClusterTestAPIVersion, azureMongoClusterTestBody), http.StatusCreated)
	if got := check("catalog"); got["reason"] != "AlreadyExists" {
		t.Fatalf("expected AlreadyExists, got %#v", got)
	}
	if got := check("-bad"); got["reason"] != "Invalid" {
		t.Fatalf("expected Invalid, got %#v", got)
	}
}
