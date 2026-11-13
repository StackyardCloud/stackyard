package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

const azureACRTestAPIVersion = "?api-version=2023-07-01"

// newAzureACRTestRegistry starts a server with resource group rg-acr and
// returns it with the subscription prefix and the registry path (no query).
func newAzureACRTestRegistry(t *testing.T) (*httptest.Server, string, string) {
	t.Helper()
	ts := newAzureResourcesTestServer(t)
	sub := "/azure/subscriptions/" + azureResourcesTestSubscription
	resp := providerContractRequest(t, ts, http.MethodPut, sub+"/resourcegroups/rg-acr?api-version=2021-04-01", []byte(`{"location":"eastus"}`), azureResourcesTestHeaders())
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	return ts, sub, sub + "/resourceGroups/rg-acr/providers/Microsoft.ContainerRegistry/registries/acrDemo01"
}

func TestAzureContainerRegistryLifecycle(t *testing.T) {
	t.Parallel()
	ts, sub, registry := newAzureACRTestRegistry(t)
	headers := azureResourcesTestHeaders()

	resp := providerContractRequest(t, ts, http.MethodPut, registry+azureACRTestAPIVersion, []byte(`{"location":"eastus","sku":{"name":"Premium"},"properties":{"adminUserEnabled":true,"loginServer":"spoofed"}}`), headers)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	created := providerContractJSONMap(t, resp)
	props, _ := created["properties"].(map[string]any)
	if created["type"] != "Microsoft.ContainerRegistry/registries" || props["loginServer"] != "acrdemo01.azurecr.io" || props["provisioningState"] != "Succeeded" {
		t.Fatalf("unexpected registry payload: %#v", created)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, registry+"/listCredentials"+azureACRTestAPIVersion, nil, headers)
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	creds := providerContractJSONMap(t, resp)
	passwords, _ := creds["passwords"].([]any)
	if creds["username"] != "acrDemo01" || len(passwords) != 2 {
		t.Fatalf("unexpected credentials: %#v", creds)
	}

	children := []struct{ path, body, typ string }{
		{"/replications/westus", `{"location":"westus"}`, "Microsoft.ContainerRegistry/registries/replications"},
		{"/webhooks/hook01", `{"location":"eastus","properties":{"serviceUri":"https://example.test/hook","actions":["push"]}}`, "Microsoft.ContainerRegistry/registries/webhooks"},
		{"/scopeMaps/pullers", `{"properties":{"actions":["repositories/app/content/read"]}}`, "Microsoft.ContainerRegistry/registries/scopeMaps"},
		{"/tokens/ci", `{"properties":{"scopeMapId":"` + registry[len("/azure"):] + `/scopeMaps/pullers"}}`, "Microsoft.ContainerRegistry/registries/tokens"},
	}
	for _, child := range children {
		resp = providerContractRequest(t, ts, http.MethodPut, registry+child.path+azureACRTestAPIVersion, []byte(child.body), headers)
		azureResourcesExpectStatus(t, resp, http.StatusCreated)
		got := providerContractJSONMap(t, resp)
		if got["type"] != child.typ || got["id"] != registry[len("/azure"):]+child.path {
			t.Fatalf("unexpected child payload for %s: %#v", child.path, got)
		}
		resp = providerContractRequest(t, ts, http.MethodGet, registry+child.path[:len(child.path)-len(got["name"].(string))-1]+azureACRTestAPIVersion, nil, headers)
		azureResourcesExpectStatus(t, resp, http.StatusOK)
		if items, _ := providerContractJSONMap(t, resp)["value"].([]any); len(items) != 1 {
			t.Fatalf("expected one child listed for %s, got %#v", child.path, items)
		}
	}
	// Proxy resources carry no location.
	resp = providerContractRequest(t, ts, http.MethodGet, registry+"/scopeMaps/pullers"+azureACRTestAPIVersion, nil, headers)
	if _, has := providerContractJSONMap(t, resp)["location"]; has {
		t.Fatal("scope map should not have a location")
	}

	// Child resources are not listed at resource group scope.
	resp = providerContractRequest(t, ts, http.MethodGet, sub+"/resourcegroups/rg-acr/resources?api-version=2021-04-01", nil, headers)
	if items, _ := providerContractJSONMap(t, resp)["value"].([]any); len(items) != 1 {
		t.Fatalf("expected only the registry at group scope, got %#v", items)
	}

	// Deleting the registry removes its children.
	resp = providerContractRequest(t, ts, http.MethodDelete, registry+azureACRTestAPIVersion, nil, headers)
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	resp = providerContractRequest(t, ts, http.MethodPut, registry+azureACRTestAPIVersion, []byte(`{"location":"eastus","sku":{"name":"Basic"}}`), headers)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	resp = providerContractRequest(t, ts, http.MethodGet, registry+"/webhooks/hook01"+azureACRTestAPIVersion, nil, headers)
	azureResourcesExpectErrorCode(t, resp, http.StatusNotFound, "ResourceNotFound")
}

func TestAzureContainerRegistryValidation(t *testing.T) {
	t.Parallel()
	ts, _, registry := newAzureACRTestRegistry(t)
	headers := azureResourcesTestHeaders()
	rgProviders := registry[:len(registry)-len("/registries/acrDemo01")]

	cases := []struct {
		name, method, path, body, code string
		status                         int
	}{
		{"short name", http.MethodPut, rgProviders + "/registries/abc", `{"location":"eastus","sku":{"name":"Basic"}}`, "InvalidResourceName", http.StatusBadRequest},
		{"dash in name", http.MethodPut, rgProviders + "/registries/acr-demo", `{"location":"eastus","sku":{"name":"Basic"}}`, "InvalidResourceName", http.StatusBadRequest},
		{"missing sku", http.MethodPut, registry, `{"location":"eastus"}`, "InvalidRequestContent", http.StatusBadRequest},
		{"bad sku", http.MethodPut, registry, `{"location":"eastus","sku":{"name":"Gold"}}`, "InvalidRequestContent", http.StatusBadRequest},
		{"missing location", http.MethodPut, registry, `{"sku":{"name":"Basic"}}`, "LocationRequired", http.StatusBadRequest},
		{"child without parent", http.MethodPut, registry + "/replications/westus", `{"location":"westus"}`, "ParentResourceNotFound", http.StatusNotFound},
		{"list children without parent", http.MethodGet, registry + "/webhooks", "", "ParentResourceNotFound", http.StatusNotFound},
		{"credentials without registry", http.MethodPost, registry + "/listCredentials", "", "ResourceNotFound", http.StatusNotFound},
	}
	for _, tc := range cases {
		var body []byte
		if tc.body != "" {
			body = []byte(tc.body)
		}
		resp := providerContractRequest(t, ts, tc.method, tc.path+azureACRTestAPIVersion, body, headers)
		t.Run(tc.name, func(t *testing.T) { azureResourcesExpectErrorCode(t, resp, tc.status, tc.code) })
	}

	resp := providerContractRequest(t, ts, http.MethodPut, registry+azureACRTestAPIVersion, []byte(`{"location":"eastus","sku":{"name":"Basic"}}`), headers)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	childCases := []struct{ name, path, body string }{
		{"webhook without serviceUri", "/webhooks/hook01", `{"location":"eastus","properties":{"actions":["push"]}}`},
		{"scope map without actions", "/scopeMaps/pullers", `{"properties":{}}`},
	}
	for _, tc := range childCases {
		resp = providerContractRequest(t, ts, http.MethodPut, registry+tc.path+azureACRTestAPIVersion, []byte(tc.body), headers)
		t.Run(tc.name, func(t *testing.T) { azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, "InvalidRequestContent") })
	}

	// The admin user is disabled by default, so no credentials are issued.
	resp = providerContractRequest(t, ts, http.MethodPost, registry+"/listCredentials"+azureACRTestAPIVersion, nil, headers)
	azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, "AdminUserDisabled")
}

func TestAzureContainerRegistryCheckNameAvailability(t *testing.T) {
	t.Parallel()
	ts, sub, registry := newAzureACRTestRegistry(t)
	headers := azureResourcesTestHeaders()
	checkPath := sub + "/providers/Microsoft.ContainerRegistry/checkNameAvailability" + azureACRTestAPIVersion
	check := func(name string) map[string]any {
		resp := providerContractRequest(t, ts, http.MethodPost, checkPath, []byte(`{"name":"`+name+`","type":"Microsoft.ContainerRegistry/registries"}`), headers)
		azureResourcesExpectStatus(t, resp, http.StatusOK)
		return providerContractJSONMap(t, resp)
	}

	if got := check("acrDemo01"); got["nameAvailable"] != true {
		t.Fatalf("expected name to be available, got %#v", got)
	}
	resp := providerContractRequest(t, ts, http.MethodPut, registry+azureACRTestAPIVersion, []byte(`{"location":"eastus","sku":{"name":"Basic"}}`), headers)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	// Registry names are global, so case differences still collide.
	if got := check("ACRDEMO01"); got["nameAvailable"] != false || got["reason"] != "AlreadyExists" {
		t.Fatalf("expected AlreadyExists, got %#v", got)
	}
	if got := check("ab"); got["nameAvailable"] != false || got["reason"] != "Invalid" {
		t.Fatalf("expected Invalid, got %#v", got)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, checkPath, []byte(`{"name":"acrDemo02","type":"Microsoft.ContainerRegistry/unknown"}`), headers)
	azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, "InvalidRequestContent")
}
