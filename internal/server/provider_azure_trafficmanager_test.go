package server

import (
	"net/http"
	"testing"
)

const azureTrafficManagerTestAPIVersion = "?api-version=2022-04-01"

func TestAzureTrafficManagerLifecycle(t *testing.T) {
	t.Parallel()
	ts := newAzureResourcesTestServer(t)
	headers := azureResourcesTestHeaders()
	sub := "/azure/subscriptions/" + azureResourcesTestSubscription
	resp := providerContractRequest(t, ts, http.MethodPut, sub+"/resourcegroups/rg-tm?api-version=2021-04-01", []byte(`{"location":"eastus"}`), headers)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	// SDKs use the lower-case path segment; responses use the canonical type.
	profile := sub + "/resourceGroups/rg-tm/providers/Microsoft.Network/trafficmanagerprofiles/web"

	resp = providerContractRequest(t, ts, http.MethodPut, profile+azureTrafficManagerTestAPIVersion, []byte(`{"location":"global","properties":{"trafficRoutingMethod":"Priority","dnsConfig":{"relativeName":"Web-App","ttl":30,"fqdn":"spoofed"}}}`), headers)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	created := providerContractJSONMap(t, resp)
	dns, _ := azureARMObject(created["properties"])["dnsConfig"].(map[string]any)
	if created["type"] != "Microsoft.Network/trafficManagerProfiles" || dns["fqdn"] != "web-app.trafficmanager.net" || dns["ttl"] != float64(30) {
		t.Fatalf("unexpected profile payload: %#v", created)
	}

	// Changing the relative name through PATCH moves the FQDN with it.
	resp = providerContractRequest(t, ts, http.MethodPatch, profile+azureTrafficManagerTestAPIVersion, []byte(`{"properties":{"dnsConfig":{"relativeName":"web-v2"}}}`), headers)
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if dns, _ := azureARMObject(providerContractJSONMap(t, resp)["properties"])["dnsConfig"].(map[string]any); dns["fqdn"] != "web-v2.trafficmanager.net" {
		t.Fatalf("expected FQDN to follow relativeName, got %#v", dns)
	}

	endpoints := []struct{ path, body string }{
		{"/externalEndpoints/primary", `{"properties":{"target":"primary.example.test","priority":1}}`},
		{"/azureEndpoints/backup", `{"properties":{"targetResourceId":"/subscriptions/x/resourceGroups/y/providers/Microsoft.Web/sites/z","priority":2}}`},
	}
	for _, endpoint := range endpoints {
		resp = providerContractRequest(t, ts, http.MethodPut, profile+endpoint.path+azureTrafficManagerTestAPIVersion, []byte(endpoint.body), headers)
		azureResourcesExpectStatus(t, resp, http.StatusCreated)
	}
	resp = providerContractRequest(t, ts, http.MethodGet, profile+"/externalEndpoints/primary"+azureTrafficManagerTestAPIVersion, nil, headers)
	if got := providerContractJSONMap(t, resp); got["type"] != "Microsoft.Network/trafficManagerProfiles/externalEndpoints" {
		t.Fatalf("unexpected endpoint: %#v", got)
	}

	resp = providerContractRequest(t, ts, http.MethodDelete, profile+azureTrafficManagerTestAPIVersion, nil, headers)
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	resp = providerContractRequest(t, ts, http.MethodGet, profile+"/azureEndpoints/backup"+azureTrafficManagerTestAPIVersion, nil, headers)
	azureResourcesExpectErrorCode(t, resp, http.StatusNotFound, "ParentResourceNotFound")
}

func TestAzureTrafficManagerValidation(t *testing.T) {
	t.Parallel()
	ts := newAzureResourcesTestServer(t)
	headers := azureResourcesTestHeaders()
	sub := "/azure/subscriptions/" + azureResourcesTestSubscription
	resp := providerContractRequest(t, ts, http.MethodPut, sub+"/resourcegroups/rg-tm?api-version=2021-04-01", []byte(`{"location":"eastus"}`), headers)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	profile := sub + "/resourceGroups/rg-tm/providers/Microsoft.Network/trafficManagerProfiles/web"

	cases := []struct{ name, body string }{
		{"regional location", `{"location":"eastus","properties":{"trafficRoutingMethod":"Priority","dnsConfig":{"relativeName":"web"}}}`},
		{"bad routing method", `{"location":"global","properties":{"trafficRoutingMethod":"Random","dnsConfig":{"relativeName":"web"}}}`},
		{"missing relative name", `{"location":"global","properties":{"trafficRoutingMethod":"Priority"}}`},
	}
	for _, tc := range cases {
		resp = providerContractRequest(t, ts, http.MethodPut, profile+azureTrafficManagerTestAPIVersion, []byte(tc.body), headers)
		t.Run(tc.name, func(t *testing.T) {
			azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, "InvalidRequestContent")
		})
	}

	resp = providerContractRequest(t, ts, http.MethodPut, profile+azureTrafficManagerTestAPIVersion, []byte(`{"location":"global","properties":{"trafficRoutingMethod":"Weighted","dnsConfig":{"relativeName":"web"}}}`), headers)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	for _, path := range []string{"/externalEndpoints/a", "/azureEndpoints/b", "/nestedEndpoints/c"} {
		resp = providerContractRequest(t, ts, http.MethodPut, profile+path+azureTrafficManagerTestAPIVersion, []byte(`{"properties":{}}`), headers)
		t.Run("endpoint "+path, func(t *testing.T) {
			azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, "InvalidRequestContent")
		})
	}
}

func TestAzureTrafficManagerCheckNameAvailability(t *testing.T) {
	t.Parallel()
	ts := newAzureResourcesTestServer(t)
	headers := azureResourcesTestHeaders()
	sub := "/azure/subscriptions/" + azureResourcesTestSubscription
	paths := []string{
		sub + "/providers/Microsoft.Network/checkTrafficManagerNameAvailabilityV2" + azureTrafficManagerTestAPIVersion,
		"/azure/providers/Microsoft.Network/checkTrafficManagerNameAvailability" + azureTrafficManagerTestAPIVersion,
	}
	body := []byte(`{"name":"WEB","type":"Microsoft.Network/trafficManagerProfiles"}`)
	for _, path := range paths {
		resp := providerContractRequest(t, ts, http.MethodPost, path, body, headers)
		if got := providerContractJSONMap(t, resp); got["nameAvailable"] != true {
			t.Fatalf("expected available from %s, got %#v", path, got)
		}
	}

	resp := providerContractRequest(t, ts, http.MethodPut, sub+"/resourcegroups/rg-tm?api-version=2021-04-01", []byte(`{"location":"eastus"}`), headers)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	resp = providerContractRequest(t, ts, http.MethodPut, sub+"/resourceGroups/rg-tm/providers/Microsoft.Network/trafficManagerProfiles/profile-a"+azureTrafficManagerTestAPIVersion, []byte(`{"location":"global","properties":{"trafficRoutingMethod":"Priority","dnsConfig":{"relativeName":"web"}}}`), headers)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	for _, path := range paths {
		resp = providerContractRequest(t, ts, http.MethodPost, path, body, headers)
		azureResourcesExpectStatus(t, resp, http.StatusOK)
		if got := providerContractJSONMap(t, resp); got["nameAvailable"] != false || got["reason"] != "AlreadyExists" {
			t.Fatalf("expected AlreadyExists from %s, got %#v", path, got)
		}
	}
	resp = providerContractRequest(t, ts, http.MethodPost, paths[0], []byte(`{}`), headers)
	azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, "InvalidRequestContent")
}
