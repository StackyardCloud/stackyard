package server

import (
	"net/http"
	"testing"
)

const azureLogAnalyticsTestAPIVersion = "?api-version=2022-10-01"

func TestAzureOperationalInsightsWorkspaceLifecycle(t *testing.T) {
	t.Parallel()
	ts := newAzureResourcesTestServer(t)
	headers := azureResourcesTestHeaders()
	sub := "/azure/subscriptions/" + azureResourcesTestSubscription
	azureResourcesExpectStatus(t, providerContractRequest(t, ts, http.MethodPut, sub+"/resourcegroups/rg-logs?api-version=2021-04-01", []byte(`{"location":"eastus"}`), headers), http.StatusCreated)
	workspace := sub + "/resourceGroups/rg-logs/providers/Microsoft.OperationalInsights/workspaces/logs-main"

	resp := providerContractRequest(t, ts, http.MethodPut, workspace+azureLogAnalyticsTestAPIVersion, []byte(`{"location":"eastus","properties":{"customerId":"spoofed"}}`), headers)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	props, _ := providerContractJSONMap(t, resp)["properties"].(map[string]any)
	customerID, _ := props["customerId"].(string)
	if len(customerID) != 36 || customerID == "spoofed" || azureARMString(azureARMObject(props["sku"])["name"]) != "PerGB2018" || props["retentionInDays"] != float64(30) {
		t.Fatalf("unexpected workspace properties: %#v", props)
	}

	// A chosen tier and retention survive, and the customerId stays stable.
	resp = providerContractRequest(t, ts, http.MethodPatch, workspace+azureLogAnalyticsTestAPIVersion, []byte(`{"properties":{"sku":{"name":"CapacityReservation"},"retentionInDays":90}}`), headers)
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	props, _ = providerContractJSONMap(t, resp)["properties"].(map[string]any)
	if props["customerId"] != customerID || azureARMString(azureARMObject(props["sku"])["name"]) != "CapacityReservation" || props["retentionInDays"] != float64(90) {
		t.Fatalf("unexpected patched workspace: %#v", props)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, workspace+"/sharedKeys"+azureLogAnalyticsTestAPIVersion, nil, headers)
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if keys := providerContractJSONMap(t, resp); len(azureARMString(keys["primarySharedKey"])) != 44 || keys["primarySharedKey"] == keys["secondarySharedKey"] {
		t.Fatalf("unexpected shared keys: %#v", keys)
	}

	children := []struct{ path, body, typ string }{
		{"/tables/AppEvents_CL", `{"properties":{"retentionInDays":60,"schema":{"name":"AppEvents_CL","columns":[{"name":"TimeGenerated","type":"datetime"}]}}}`, "Microsoft.OperationalInsights/workspaces/tables"},
		{"/savedSearches/errors", `{"properties":{"category":"Ops","displayName":"Errors","query":"AppTraces | where SeverityLevel > 3"}}`, "Microsoft.OperationalInsights/workspaces/savedSearches"},
		{"/dataExports/to-storage", `{"properties":{"destination":{"resourceId":"/subscriptions/x/resourceGroups/y/providers/Microsoft.Storage/storageAccounts/z"},"tableNames":["AppEvents_CL"]}}`, "Microsoft.OperationalInsights/workspaces/dataExports"},
	}
	for _, child := range children {
		resp = providerContractRequest(t, ts, http.MethodPut, workspace+child.path+azureLogAnalyticsTestAPIVersion, []byte(child.body), headers)
		azureResourcesExpectStatus(t, resp, http.StatusCreated)
		if got := providerContractJSONMap(t, resp); got["type"] != child.typ {
			t.Fatalf("unexpected child for %s: %#v", child.path, got)
		}
	}

	azureResourcesExpectStatus(t, providerContractRequest(t, ts, http.MethodDelete, workspace+azureLogAnalyticsTestAPIVersion, nil, headers), http.StatusOK)
	azureResourcesExpectErrorCode(t, providerContractRequest(t, ts, http.MethodGet, workspace+"/savedSearches/errors"+azureLogAnalyticsTestAPIVersion, nil, headers), http.StatusNotFound, "ParentResourceNotFound")
}

func TestAzureOperationalInsightsValidation(t *testing.T) {
	t.Parallel()
	ts := newAzureResourcesTestServer(t)
	headers := azureResourcesTestHeaders()
	sub := "/azure/subscriptions/" + azureResourcesTestSubscription
	azureResourcesExpectStatus(t, providerContractRequest(t, ts, http.MethodPut, sub+"/resourcegroups/rg-logs?api-version=2021-04-01", []byte(`{"location":"eastus"}`), headers), http.StatusCreated)
	workspaces := sub + "/resourceGroups/rg-logs/providers/Microsoft.OperationalInsights/workspaces/"

	cases := []struct{ name, path, body, code string }{
		{"name too short", workspaces + "abc", `{"location":"eastus"}`, "InvalidResourceName"},
		{"trailing hyphen", workspaces + "logs-", `{"location":"eastus"}`, "InvalidResourceName"},
		{"unknown sku", workspaces + "logs-main", `{"location":"eastus","properties":{"sku":{"name":"Basic"}}}`, "InvalidRequestContent"},
		{"retention too long", workspaces + "logs-main", `{"location":"eastus","properties":{"retentionInDays":731}}`, "InvalidRequestContent"},
		{"retention not a number", workspaces + "logs-main", `{"location":"eastus","properties":{"retentionInDays":"90"}}`, "InvalidRequestContent"},
	}
	for _, tc := range cases {
		resp := providerContractRequest(t, ts, http.MethodPut, tc.path+azureLogAnalyticsTestAPIVersion, []byte(tc.body), headers)
		t.Run(tc.name, func(t *testing.T) { azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, tc.code) })
	}

	azureResourcesExpectStatus(t, providerContractRequest(t, ts, http.MethodPut, workspaces+"logs-main"+azureLogAnalyticsTestAPIVersion, []byte(`{"location":"eastus"}`), headers), http.StatusCreated)
	childCases := []struct{ name, path, body string }{
		{"saved search without query", "/savedSearches/s1", `{"properties":{"category":"Ops","displayName":"Errors"}}`},
		{"export without tables", "/dataExports/e1", `{"properties":{"destination":{"resourceId":"/x"},"tableNames":[]}}`},
		{"export without destination", "/dataExports/e1", `{"properties":{"tableNames":["Heartbeat"]}}`},
	}
	for _, tc := range childCases {
		resp := providerContractRequest(t, ts, http.MethodPut, workspaces+"logs-main"+tc.path+azureLogAnalyticsTestAPIVersion, []byte(tc.body), headers)
		t.Run(tc.name, func(t *testing.T) {
			azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, "InvalidRequestContent")
		})
	}
	azureResourcesExpectErrorCode(t, providerContractRequest(t, ts, http.MethodPost, workspaces+"missing-ws/sharedKeys"+azureLogAnalyticsTestAPIVersion, nil, headers), http.StatusNotFound, "ResourceNotFound")
}
