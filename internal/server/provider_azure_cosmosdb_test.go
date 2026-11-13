package server

import (
	"net/http"
	"strings"
	"testing"
)

const azureCosmosDBTestAPIVersion = "?api-version=2024-11-15"

const azureCosmosDBTestAccountBody = `{"location":"eastus","kind":"GlobalDocumentDB","properties":{"databaseAccountOfferType":"Standard","locations":[{"locationName":"East US","failoverPriority":0},{"locationName":"West US","failoverPriority":1,"isZoneRedundant":true}]}}`

func TestAzureCosmosDBAccountLifecycle(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-cosmos")
	account := sub + "/resourceGroups/rg-cosmos/providers/Microsoft.DocumentDB/databaseAccounts/orders-cosmos"

	resp := do(http.MethodPut, account+azureCosmosDBTestAPIVersion, azureCosmosDBTestAccountBody)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	props, _ := providerContractJSONMap(t, resp)["properties"].(map[string]any)
	writes, _ := props["writeLocations"].([]any)
	reads, _ := props["readLocations"].([]any)
	if props["documentEndpoint"] != "https://orders-cosmos.documents.azure.com:443/" || len(writes) != 1 || len(reads) != 2 {
		t.Fatalf("unexpected account properties: %#v", props)
	}
	if write := writes[0].(map[string]any); write["id"] != "orders-cosmos-eastus" || write["documentEndpoint"] != "https://orders-cosmos-eastus.documents.azure.com:443/" {
		t.Fatalf("unexpected write location: %#v", write)
	}
	if policy, _ := props["consistencyPolicy"].(map[string]any); policy["defaultConsistencyLevel"] != "Session" {
		t.Fatalf("expected Session consistency by default, got %#v", policy)
	}

	resp = do(http.MethodPost, account+"/listKeys"+azureCosmosDBTestAPIVersion, "")
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	keys := providerContractJSONMap(t, resp)
	primary, _ := keys["primaryMasterKey"].(string)
	if len(primary) != 88 || keys["primaryMasterKey"] == keys["secondaryMasterKey"] {
		t.Fatalf("unexpected keys: %#v", keys)
	}
	resp = do(http.MethodPost, account+"/readonlykeys"+azureCosmosDBTestAPIVersion, "")
	if ro := providerContractJSONMap(t, resp); ro["primaryReadonlyMasterKey"] != keys["primaryReadonlyMasterKey"] || ro["primaryMasterKey"] != nil {
		t.Fatalf("unexpected read-only keys: %#v", ro)
	}
	resp = do(http.MethodPost, account+"/listConnectionStrings"+azureCosmosDBTestAPIVersion, "")
	conns, _ := providerContractJSONMap(t, resp)["connectionStrings"].([]any)
	if len(conns) != 4 || conns[0].(map[string]any)["connectionString"] != "AccountEndpoint=https://orders-cosmos.documents.azure.com:443/;AccountKey="+primary+";" {
		t.Fatalf("unexpected connection strings: %#v", conns)
	}

	database := account + "/sqlDatabases/orders"
	azureResourcesExpectStatus(t, do(http.MethodPut, database+azureCosmosDBTestAPIVersion, `{"properties":{"resource":{"id":"orders"}}}`), http.StatusCreated)
	container := database + "/containers/items"
	azureResourcesExpectErrorCode(t, do(http.MethodPut, container+azureCosmosDBTestAPIVersion, `{"properties":{"resource":{"id":"items"}}}`), http.StatusBadRequest, "InvalidRequestContent")
	azureResourcesExpectErrorCode(t, do(http.MethodPut, container+azureCosmosDBTestAPIVersion, `{"properties":{"resource":{"id":"items","partitionKey":{"paths":["customerId"]}}}}`), http.StatusBadRequest, "InvalidRequestContent")
	azureResourcesExpectStatus(t, do(http.MethodPut, container+azureCosmosDBTestAPIVersion, `{"properties":{"resource":{"id":"items","partitionKey":{"paths":["/customerId"],"kind":"Hash"}}}}`), http.StatusCreated)
	azureResourcesExpectStatus(t, do(http.MethodPut, account+"/mongodbDatabases/catalog"+azureCosmosDBTestAPIVersion, `{"properties":{"resource":{"id":"catalog"}}}`), http.StatusCreated)
	azureResourcesExpectErrorCode(t, do(http.MethodPut, account+"/mongodbDatabases/catalog/collections/products"+azureCosmosDBTestAPIVersion, `{"properties":{"resource":{}}}`), http.StatusBadRequest, "InvalidRequestContent")

	azureResourcesExpectStatus(t, do(http.MethodDelete, account+azureCosmosDBTestAPIVersion, ""), http.StatusOK)
	azureResourcesExpectErrorCode(t, do(http.MethodGet, container+azureCosmosDBTestAPIVersion, ""), http.StatusNotFound, "ParentResourceNotFound")
}

func TestAzureCosmosDBValidation(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-cosmos")
	accounts := sub + "/resourceGroups/rg-cosmos/providers/Microsoft.DocumentDB/databaseAccounts/"
	locations := `"locations":[{"locationName":"East US","failoverPriority":0}]`
	cases := []struct{ name, path, body, code string }{
		{"uppercase name", accounts + "Orders", azureCosmosDBTestAccountBody, "InvalidResourceName"},
		{"name too long", accounts + strings.Repeat("a", 45), azureCosmosDBTestAccountBody, "InvalidResourceName"},
		{"missing offer type", accounts + "orders", `{"location":"eastus","properties":{` + locations + `}}`, "InvalidRequestContent"},
		{"no locations", accounts + "orders", `{"location":"eastus","properties":{"databaseAccountOfferType":"Standard"}}`, "InvalidRequestContent"},
		{"duplicate priority", accounts + "orders", `{"location":"eastus","properties":{"databaseAccountOfferType":"Standard","locations":[{"locationName":"East US","failoverPriority":0},{"locationName":"West US","failoverPriority":0}]}}`, "InvalidRequestContent"},
		{"no write region", accounts + "orders", `{"location":"eastus","properties":{"databaseAccountOfferType":"Standard","locations":[{"locationName":"East US","failoverPriority":1}]}}`, "InvalidRequestContent"},
		{"bad consistency", accounts + "orders", `{"location":"eastus","properties":{"databaseAccountOfferType":"Standard",` + locations + `,"consistencyPolicy":{"defaultConsistencyLevel":"Linearizable"}}}`, "InvalidRequestContent"},
		{"bad kind", accounts + "orders", `{"location":"eastus","kind":"Cassandra","properties":{"databaseAccountOfferType":"Standard",` + locations + `}}`, "InvalidRequestContent"},
	}
	for _, tc := range cases {
		resp := do(http.MethodPut, tc.path+azureCosmosDBTestAPIVersion, tc.body)
		t.Run(tc.name, func(t *testing.T) { azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, tc.code) })
	}
}

func TestAzureCosmosDBCheckNameExists(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-cosmos")
	check := "/azure/providers/Microsoft.DocumentDB/databaseAccountNames/orders-cosmos" + azureCosmosDBTestAPIVersion
	azureResourcesExpectStatus(t, do(http.MethodHead, check, ""), http.StatusNotFound)
	azureResourcesExpectStatus(t, do(http.MethodPut, sub+"/resourceGroups/rg-cosmos/providers/Microsoft.DocumentDB/databaseAccounts/orders-cosmos"+azureCosmosDBTestAPIVersion, azureCosmosDBTestAccountBody), http.StatusCreated)
	azureResourcesExpectStatus(t, do(http.MethodHead, check, ""), http.StatusOK)
	// Mongo vCore clusters share the namespace but not the account name check.
	azureResourcesExpectStatus(t, do(http.MethodHead, "/azure/providers/Microsoft.DocumentDB/databaseAccountNames/other"+azureCosmosDBTestAPIVersion, ""), http.StatusNotFound)
	azureResourcesExpectErrorCode(t, do(http.MethodPost, check, ""), http.StatusMethodNotAllowed, "MethodNotAllowed")
}
