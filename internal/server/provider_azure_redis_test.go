package server

import (
	"net/http"
	"testing"
)

const azureRedisTestAPIVersion = "?api-version=2024-11-01"

func TestAzureRedisLifecycle(t *testing.T) {
	t.Parallel()
	ts := newAzureResourcesTestServer(t)
	headers := azureResourcesTestHeaders()
	sub := "/azure/subscriptions/" + azureResourcesTestSubscription
	resp := providerContractRequest(t, ts, http.MethodPut, sub+"/resourcegroups/rg-redis?api-version=2021-04-01", []byte(`{"location":"eastus"}`), headers)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	cache := sub + "/resourceGroups/rg-redis/providers/Microsoft.Cache/redis/Cache-A"

	resp = providerContractRequest(t, ts, http.MethodPut, cache+azureRedisTestAPIVersion, []byte(`{"location":"eastus","properties":{"sku":{"name":"Standard","family":"C","capacity":1},"hostName":"spoofed"}}`), headers)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	created := providerContractJSONMap(t, resp)
	props, _ := created["properties"].(map[string]any)
	if created["type"] != "Microsoft.Cache/redis" || props["hostName"] != "cache-a.redis.cache.windows.net" || props["sslPort"] != float64(6380) {
		t.Fatalf("unexpected cache payload: %#v", created)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, cache+"/listKeys"+azureRedisTestAPIVersion, nil, headers)
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	keys := providerContractJSONMap(t, resp)
	if primary, _ := keys["primaryKey"].(string); len(primary) != 44 || keys["primaryKey"] == keys["secondaryKey"] {
		t.Fatalf("unexpected keys: %#v", keys)
	}

	rule := cache + "/firewallRules/office"
	resp = providerContractRequest(t, ts, http.MethodPut, rule+azureRedisTestAPIVersion, []byte(`{"properties":{"startIP":"10.0.0.1","endIP":"10.0.0.9"}}`), headers)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	if got := providerContractJSONMap(t, resp); got["type"] != "Microsoft.Cache/redis/firewallRules" {
		t.Fatalf("unexpected firewall rule: %#v", got)
	}
	resp = providerContractRequest(t, ts, http.MethodGet, cache+"/firewallRules"+azureRedisTestAPIVersion, nil, headers)
	if items, _ := providerContractJSONMap(t, resp)["value"].([]any); len(items) != 1 {
		t.Fatalf("expected one firewall rule, got %#v", items)
	}

	resp = providerContractRequest(t, ts, http.MethodDelete, cache+azureRedisTestAPIVersion, nil, headers)
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	resp = providerContractRequest(t, ts, http.MethodGet, rule+azureRedisTestAPIVersion, nil, headers)
	azureResourcesExpectErrorCode(t, resp, http.StatusNotFound, "ParentResourceNotFound")
}

func TestAzureRedisValidation(t *testing.T) {
	t.Parallel()
	ts := newAzureResourcesTestServer(t)
	headers := azureResourcesTestHeaders()
	sub := "/azure/subscriptions/" + azureResourcesTestSubscription
	resp := providerContractRequest(t, ts, http.MethodPut, sub+"/resourcegroups/rg-redis?api-version=2021-04-01", []byte(`{"location":"eastus"}`), headers)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	caches := sub + "/resourceGroups/rg-redis/providers/Microsoft.Cache/redis/"

	cases := []struct{ name, path, body, code string }{
		{"double hyphen", caches + "cache--a", `{"location":"eastus","properties":{"sku":{"name":"Basic","family":"C","capacity":0}}}`, "InvalidResourceName"},
		{"leading hyphen", caches + "-cache", `{"location":"eastus","properties":{"sku":{"name":"Basic","family":"C","capacity":0}}}`, "InvalidResourceName"},
		{"missing sku", caches + "cache-a", `{"location":"eastus"}`, "InvalidRequestContent"},
		{"top-level sku only", caches + "cache-a", `{"location":"eastus","sku":{"name":"Basic","family":"C","capacity":0}}`, "InvalidRequestContent"},
		{"premium with family C", caches + "cache-a", `{"location":"eastus","properties":{"sku":{"name":"Premium","family":"C","capacity":1}}}`, "InvalidRequestContent"},
		{"standard capacity 7", caches + "cache-a", `{"location":"eastus","properties":{"sku":{"name":"Standard","family":"C","capacity":7}}}`, "InvalidRequestContent"},
		{"missing capacity", caches + "cache-a", `{"location":"eastus","properties":{"sku":{"name":"Basic","family":"C"}}}`, "InvalidRequestContent"},
	}
	for _, tc := range cases {
		resp := providerContractRequest(t, ts, http.MethodPut, tc.path+azureRedisTestAPIVersion, []byte(tc.body), headers)
		t.Run(tc.name, func(t *testing.T) { azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, tc.code) })
	}

	resp = providerContractRequest(t, ts, http.MethodPut, caches+"cache-a"+azureRedisTestAPIVersion, []byte(`{"location":"eastus","properties":{"sku":{"name":"Premium","family":"P","capacity":1}}}`), headers)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	for _, body := range []string{`{"properties":{"startIP":"10.0.0.1"}}`, `{"properties":{"startIP":"::1","endIP":"::2"}}`} {
		resp = providerContractRequest(t, ts, http.MethodPut, caches+"cache-a/firewallRules/bad"+azureRedisTestAPIVersion, []byte(body), headers)
		azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, "InvalidRequestContent")
	}
	resp = providerContractRequest(t, ts, http.MethodPost, caches+"missing-cache/listKeys"+azureRedisTestAPIVersion, nil, headers)
	azureResourcesExpectErrorCode(t, resp, http.StatusNotFound, "ResourceNotFound")
}
