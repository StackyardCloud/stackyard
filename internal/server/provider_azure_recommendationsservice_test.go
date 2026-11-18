package server

import (
	"net/http"
	"testing"
)

const azureRecommendationsTestAPIVersion = "?api-version=2022-02-01"

func TestAzureRecommendationsAccounts(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-reco")
	account := sub + "/resourceGroups/rg-reco/providers/Microsoft.RecommendationsService/accounts/sampleAccount"
	resp := do(http.MethodPut, account+azureRecommendationsTestAPIVersion, `{"location":"West US","tags":{"Environment":"Prod"},
		"properties":{"configuration":"Capacity","endpointAuthentications":[{"aadTenantID":"tenant","principalID":"oid","principalType":"User"}]}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	if props := providerContractJSONMap(t, resp)["properties"].(map[string]any); props["provisioningState"] != "Succeeded" || props["configuration"] != "Capacity" {
		t.Fatalf("unexpected account: %#v", props)
	}

	// The configuration is set at creation only; omitting it keeps it.
	patched := providerContractJSONMap(t, do(http.MethodPatch, account+azureRecommendationsTestAPIVersion, `{"tags":{"Environment":"Test"},
		"properties":{"cors":[{"allowedOrigins":["http://siteA.com"],"allowedMethods":["GET","PUT"],"maxAgeInSeconds":60}],
		"reportsConnectionString":"https://storageAccount.blob.core.windows.net/container/root"}}`))
	if props := patched["properties"].(map[string]any); props["configuration"] != "Capacity" || props["cors"] == nil || patched["tags"].(map[string]any)["Environment"] != "Test" {
		t.Fatalf("unexpected patched account: %#v", patched)
	}
	azureSentinelExpectRejected(t, do(http.MethodPatch, account+azureRecommendationsTestAPIVersion, `{"properties":{"configuration":"Free"}}`), "configuration can only be set at creation")
	azureSentinelExpectRejected(t, do(http.MethodPut, account+azureRecommendationsTestAPIVersion, `{"location":"West US","properties":{"configuration":"Free"}}`), "configuration can only be set at creation")
	kept := providerContractJSONMap(t, do(http.MethodPut, account+azureRecommendationsTestAPIVersion, `{"location":"West US","properties":{}}`))
	if kept["properties"].(map[string]any)["configuration"] != "Capacity" {
		t.Fatalf("PUT without configuration should keep it: %#v", kept)
	}
	free := sub + "/resourceGroups/rg-reco/providers/Microsoft.RecommendationsService/accounts/freeAccount" + azureRecommendationsTestAPIVersion
	azureResourcesExpectStatus(t, do(http.MethodPut, free, `{"location":"West US","properties":{}}`), http.StatusCreated)
	azureSentinelExpectRejected(t, do(http.MethodPut, free, `{"location":"West US","properties":{"configuration":"Free"}}`), "configuration can only be set at creation")

	// Names are unique across resource groups and subscriptions.
	check := sub + "/providers/Microsoft.RecommendationsService/checkNameAvailability" + azureRecommendationsTestAPIVersion
	taken := providerContractJSONMap(t, do(http.MethodPost, check, `{"type":"Microsoft.RecommendationsService/accounts","name":"SAMPLEACCOUNT"}`))
	if taken["nameAvailable"] != false || taken["reason"] != "AlreadyExists" {
		t.Fatalf("taken = %#v", taken)
	}
	if open := providerContractJSONMap(t, do(http.MethodPost, check, `{"type":"Microsoft.RecommendationsService/accounts","name":"other"}`)); open["nameAvailable"] != true {
		t.Fatalf("open = %#v", open)
	}
	azureSentinelExpectRejected(t, do(http.MethodPost, check, `{"type":"Microsoft.Storage/storageAccounts","name":"other"}`), "type must be")
	azureSentinelExpectRejected(t, do(http.MethodPost, check, `{"type":"Microsoft.RecommendationsService/accounts"}`), "name is required")
	azureResourcesExpectStatus(t, do(http.MethodPut, sub+"/resourceGroups/rg-other"+azureRecommendationsTestAPIVersion, `{"location":"westus"}`), http.StatusCreated)
	azureSentinelExpectRejected(t, do(http.MethodPut, sub+"/resourceGroups/rg-other/providers/Microsoft.RecommendationsService/accounts/sampleaccount"+azureRecommendationsTestAPIVersion, `{"location":"West US"}`), "already exists")

	for _, tc := range []struct{ method, body, want string }{
		{http.MethodPut, `{"location":"West US","properties":{"configuration":"Premium"}}`, "properties.configuration"},
		{http.MethodPut, `{"location":"West US","properties":"x"}`, "properties must be an object"},
		{http.MethodPut, `{"location":"West US","properties":{"reportsConnectionString":1}}`, "reportsConnectionString must be a string"},
		{http.MethodPut, `{"location":"West US","properties":{"endpointAuthentications":{}}}`, "endpointAuthentications must be an array"},
		{http.MethodPut, `{"location":"West US","properties":{"endpointAuthentications":["oid"]}}`, "endpointAuthentications items must be objects"},
		{http.MethodPut, `{"location":"West US","properties":{"endpointAuthentications":[{"principalType":"Group"}]}}`, "principalType"},
		{http.MethodPut, `{"location":"West US","properties":{"endpointAuthentications":[{"principalID":5}]}}`, "principalID must be a string"},
		{http.MethodPatch, `{"properties":{"cors":[{"allowedOrigins":"*"}]}}`, "allowedOrigins must be an array of strings"},
		{http.MethodPatch, `{"properties":{"cors":[{"maxAgeInSeconds":-1}]}}`, "maxAgeInSeconds"},
		{http.MethodPatch, `{"properties":"x"}`, "properties must be an object"},
	} {
		azureSentinelExpectRejected(t, do(tc.method, account+azureRecommendationsTestAPIVersion, tc.body), tc.want)
	}
}

func TestAzureRecommendationsModelingAndEndpoints(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-reco")
	account := sub + "/resourceGroups/rg-reco/providers/Microsoft.RecommendationsService/accounts/acct"
	azureResourcesExpectStatus(t, do(http.MethodPut, account+azureRecommendationsTestAPIVersion, `{"location":"West US","properties":{"configuration":"Free"}}`), http.StatusCreated)

	// No modeling resources: an empty status.
	status := account + "/status" + azureRecommendationsTestAPIVersion
	if empty := providerContractJSONMap(t, do(http.MethodPost, status, "")); len(empty) != 0 {
		t.Fatalf("status = %#v", empty)
	}
	modeling := account + "/modeling"
	resp := do(http.MethodPut, modeling+"/c1"+azureRecommendationsTestAPIVersion, `{"location":"West US","properties":{"features":"Standard","frequency":"High","size":"Medium",
		"inputData":{"connectionString":"https://storageAccount.blob.core.windows.net/container/root"}}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	if props := providerContractJSONMap(t, resp)["properties"].(map[string]any); props["provisioningState"] != "Succeeded" {
		t.Fatalf("unexpected modeling: %#v", props)
	}
	azureResourcesExpectStatus(t, do(http.MethodPut, modeling+"/c2"+azureRecommendationsTestAPIVersion, `{"location":"West US","properties":{"features":"Basic"}}`), http.StatusCreated)
	scopes := providerContractJSONMap(t, do(http.MethodPost, status, ""))["scopesStatuses"].([]any)
	if len(scopes) != 2 {
		t.Fatalf("scopesStatuses = %#v", scopes)
	}
	for i, want := range [][]string{{"c1", "Success", "Success", "Success"}, {"c2", "Failed", "Pending", "Pending"}} {
		scope := scopes[i].(map[string]any)
		statuses := scope["statuses"].([]any)
		if scope["scope"] != want[0] || len(statuses) != 3 {
			t.Fatalf("scope %d = %#v", i, scope)
		}
		for j, stage := range []string{"DataAccess", "DataValidation", "Model"} {
			if s := statuses[j].(map[string]any); s["stage"] != stage || s["status"] != want[j+1] {
				t.Fatalf("scope %s stage %d = %#v", want[0], j, s)
			}
		}
	}

	// Features are set at creation only; frequency and input data change.
	azureSentinelExpectRejected(t, do(http.MethodPatch, modeling+"/c1"+azureRecommendationsTestAPIVersion, `{"properties":{"features":"Premium"}}`), "features can only be set at creation")
	patched := providerContractJSONMap(t, do(http.MethodPatch, modeling+"/c1"+azureRecommendationsTestAPIVersion, `{"properties":{"frequency":"Low"}}`))
	if p := patched["properties"].(map[string]any); p["frequency"] != "Low" || p["features"] != "Standard" {
		t.Fatalf("unexpected patched modeling: %#v", p)
	}
	for _, tc := range []struct{ method, body, want string }{
		{http.MethodPut, `{"location":"West US","properties":{"features":"Gold"}}`, "properties.features"},
		{http.MethodPut, `{"location":"West US","properties":{"frequency":"Hourly"}}`, "properties.frequency"},
		{http.MethodPut, `{"location":"West US","properties":{"size":"Huge"}}`, "properties.size"},
		{http.MethodPut, `{"location":"West US","properties":{"inputData":"x"}}`, "inputData must be an object"},
		{http.MethodPut, `{"location":"West US","properties":{"inputData":{"connectionString":1}}}`, "connectionString must be a string"},
		{http.MethodPatch, `{"properties":{"size":"Huge"}}`, "properties.size"},
	} {
		azureSentinelExpectRejected(t, do(tc.method, modeling+"/c3"+azureRecommendationsTestAPIVersion, tc.body), tc.want)
	}

	endpoints := account + "/serviceEndpoints"
	resp = do(http.MethodPut, endpoints+"/s1"+azureRecommendationsTestAPIVersion, `{"location":"West US","tags":{"Environment":"Prod"},"properties":{"preAllocatedCapacity":100,"url":"evil.example.com"}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	ep := providerContractJSONMap(t, resp)["properties"].(map[string]any)
	if ep["pairedLocation"] != "East US" || ep["url"] != "acct-s1.recommendations.stackyard.invalid" || ep["preAllocatedCapacity"] != float64(100) {
		t.Fatalf("unexpected endpoint: %#v", ep)
	}
	unpaired := providerContractJSONMap(t, do(http.MethodPut, endpoints+"/s2"+azureRecommendationsTestAPIVersion, `{"location":"polandcentral","properties":{}}`))
	if _, ok := unpaired["properties"].(map[string]any)["pairedLocation"]; ok {
		t.Fatalf("Poland Central has no pair: %#v", unpaired)
	}
	reverse := providerContractJSONMap(t, do(http.MethodPut, endpoints+"/s3"+azureRecommendationsTestAPIVersion, `{"location":"germanynorth","properties":{}}`))
	if reverse["properties"].(map[string]any)["pairedLocation"] != "Germany West Central" {
		t.Fatalf("Germany North pairs with Germany West Central: %#v", reverse)
	}
	tagged := providerContractJSONMap(t, do(http.MethodPatch, endpoints+"/s1"+azureRecommendationsTestAPIVersion, `{"tags":{"Environment":"Test"}}`))
	if tagged["tags"].(map[string]any)["Environment"] != "Test" {
		t.Fatalf("tags = %#v", tagged["tags"])
	}
	azureSentinelExpectRejected(t, do(http.MethodPatch, endpoints+"/s1"+azureRecommendationsTestAPIVersion, `{"properties":{"preAllocatedCapacity":5}}`), "Only tags can be updated")
	azureSentinelExpectRejected(t, do(http.MethodPut, endpoints+"/s3"+azureRecommendationsTestAPIVersion, `{"location":"West US","properties":{"preAllocatedCapacity":1.5}}`), "preAllocatedCapacity")
	azureSentinelExpectRejected(t, do(http.MethodPut, endpoints+"/s3"+azureRecommendationsTestAPIVersion, `{"location":"West US","properties":[]}`), "properties must be an object")
	if list := providerContractJSONMap(t, do(http.MethodGet, endpoints+azureRecommendationsTestAPIVersion, ""))["value"].([]any); len(list) != 3 {
		t.Fatalf("endpoints = %d", len(list))
	}

	// Deleting the account deletes its modeling and endpoint resources.
	azureResourcesExpectStatus(t, do(http.MethodDelete, account+azureRecommendationsTestAPIVersion, ""), http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodGet, endpoints+"/s1"+azureRecommendationsTestAPIVersion, ""), http.StatusNotFound)
	azureResourcesExpectStatus(t, do(http.MethodPost, sub+"/resourceGroups/rg-reco/providers/Microsoft.RecommendationsService/checkNameAvailability"+azureRecommendationsTestAPIVersion, `{"type":"Microsoft.RecommendationsService/accounts","name":"acct"}`), http.StatusBadRequest)
}
