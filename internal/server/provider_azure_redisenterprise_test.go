package server

import (
	"net/http"
	"testing"
)

const azureRedisEnterpriseTestAPIVersion = "?api-version=2024-10-01"

func TestAzureRedisEnterpriseLifecycle(t *testing.T) {
	t.Parallel()
	ts := newAzureResourcesTestServer(t)
	headers := azureResourcesTestHeaders()
	sub := "/azure/subscriptions/" + azureResourcesTestSubscription
	azureResourcesExpectStatus(t, providerContractRequest(t, ts, http.MethodPut, sub+"/resourcegroups/rg-amr?api-version=2021-04-01", []byte(`{"location":"eastus"}`), headers), http.StatusCreated)
	clusters := sub + "/resourceGroups/rg-amr/providers/Microsoft.Cache/redisEnterprise/"

	hostCases := []struct{ name, sku, location, host string }{
		{"Managed-A", "Balanced_B5", "West Europe", "managed-a.westeurope.redis.azure.net"},
		{"enterprise-b", "Enterprise_E10", "eastus", "enterprise-b.eastus.redisenterprise.cache.azure.net"},
	}
	for _, tc := range hostCases {
		resp := providerContractRequest(t, ts, http.MethodPut, clusters+tc.name+azureRedisEnterpriseTestAPIVersion, []byte(`{"location":"`+tc.location+`","sku":{"name":"`+tc.sku+`"}}`), headers)
		azureResourcesExpectStatus(t, resp, http.StatusCreated)
		if props, _ := providerContractJSONMap(t, resp)["properties"].(map[string]any); props["hostName"] != tc.host || props["resourceState"] != "Running" {
			t.Fatalf("unexpected cluster properties for %s: %#v", tc.sku, props)
		}
	}

	database := clusters + "Managed-A/databases/default"
	resp := providerContractRequest(t, ts, http.MethodPut, database+azureRedisEnterpriseTestAPIVersion, []byte(`{"properties":{"clientProtocol":"Encrypted","clusteringPolicy":"OSSCluster"}}`), headers)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	if props, _ := providerContractJSONMap(t, resp)["properties"].(map[string]any); props["port"] != float64(10000) {
		t.Fatalf("expected default port 10000, got %#v", props)
	}
	resp = providerContractRequest(t, ts, http.MethodPut, clusters+"enterprise-b/databases/default"+azureRedisEnterpriseTestAPIVersion, []byte(`{"properties":{"port":10001}}`), headers)
	if props, _ := providerContractJSONMap(t, resp)["properties"].(map[string]any); props["port"] != float64(10001) {
		t.Fatalf("expected chosen port 10001, got %#v", props)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, database+"/listKeys"+azureRedisEnterpriseTestAPIVersion, nil, headers)
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if keys := providerContractJSONMap(t, resp); keys["primaryKey"] == "" || keys["primaryKey"] == keys["secondaryKey"] {
		t.Fatalf("unexpected keys: %#v", keys)
	}

	azureResourcesExpectStatus(t, providerContractRequest(t, ts, http.MethodDelete, clusters+"Managed-A"+azureRedisEnterpriseTestAPIVersion, nil, headers), http.StatusOK)
	azureResourcesExpectErrorCode(t, providerContractRequest(t, ts, http.MethodGet, database+azureRedisEnterpriseTestAPIVersion, nil, headers), http.StatusNotFound, "ParentResourceNotFound")
}

func TestAzureRedisEnterpriseValidation(t *testing.T) {
	t.Parallel()
	ts := newAzureResourcesTestServer(t)
	headers := azureResourcesTestHeaders()
	sub := "/azure/subscriptions/" + azureResourcesTestSubscription
	azureResourcesExpectStatus(t, providerContractRequest(t, ts, http.MethodPut, sub+"/resourcegroups/rg-amr?api-version=2021-04-01", []byte(`{"location":"eastus"}`), headers), http.StatusCreated)
	clusters := sub + "/resourceGroups/rg-amr/providers/Microsoft.Cache/redisEnterprise/"

	cases := []struct{ name, path, body, code string }{
		{"double hyphen", clusters + "amr--a", `{"location":"eastus","sku":{"name":"Balanced_B5"}}`, "InvalidResourceName"},
		{"classic redis sku", clusters + "amr-a", `{"location":"eastus","sku":{"name":"Premium"}}`, "InvalidRequestContent"},
		{"sku without size", clusters + "amr-a", `{"location":"eastus","sku":{"name":"Balanced_B"}}`, "InvalidRequestContent"},
	}
	for _, tc := range cases {
		resp := providerContractRequest(t, ts, http.MethodPut, tc.path+azureRedisEnterpriseTestAPIVersion, []byte(tc.body), headers)
		t.Run(tc.name, func(t *testing.T) { azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, tc.code) })
	}

	azureResourcesExpectStatus(t, providerContractRequest(t, ts, http.MethodPut, clusters+"amr-a"+azureRedisEnterpriseTestAPIVersion, []byte(`{"location":"eastus","sku":{"name":"Balanced_B5"}}`), headers), http.StatusCreated)
	dbCases := []struct{ name, path, body, code string }{
		{"non-default name", "/databases/db1", `{}`, "InvalidResourceName"},
		{"bad protocol", "/databases/default", `{"properties":{"clientProtocol":"TLS"}}`, "InvalidRequestContent"},
		{"bad clustering", "/databases/default", `{"properties":{"clusteringPolicy":"Sharded"}}`, "InvalidRequestContent"},
	}
	for _, tc := range dbCases {
		resp := providerContractRequest(t, ts, http.MethodPut, clusters+"amr-a"+tc.path+azureRedisEnterpriseTestAPIVersion, []byte(tc.body), headers)
		t.Run(tc.name, func(t *testing.T) { azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, tc.code) })
	}
	azureResourcesExpectErrorCode(t, providerContractRequest(t, ts, http.MethodPost, clusters+"amr-a/databases/default/listKeys"+azureRedisEnterpriseTestAPIVersion, nil, headers), http.StatusNotFound, "ResourceNotFound")
}
