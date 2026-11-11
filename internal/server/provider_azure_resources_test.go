package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

const azureResourcesTestSubscription = "00000000-0000-0000-0000-000000000000"

func newAzureResourcesTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	return newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})
}

func azureResourcesTestHeaders() map[string]string {
	return map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
		"Content-Type":              "application/json",
	}
}

func azureResourcesExpectStatus(t *testing.T, resp *http.Response, want int) {
	t.Helper()
	if resp.StatusCode != want {
		t.Fatalf("expected %d, got %d body=%s", want, resp.StatusCode, string(providerContractBody(t, resp)))
	}
}

func azureResourcesExpectErrorCode(t *testing.T, resp *http.Response, status int, code string) {
	t.Helper()
	azureResourcesExpectStatus(t, resp, status)
	if got := resp.Header.Get("x-ms-error-code"); got != code {
		t.Fatalf("expected x-ms-error-code %q, got %q", code, got)
	}
	payload := providerContractJSONMap(t, resp)
	errObj, _ := payload["error"].(map[string]any)
	if errObj["code"] != code {
		t.Fatalf("expected error.code %q, got %#v", code, payload)
	}
}

func TestAzureResourceGroupLifecycle(t *testing.T) {
	t.Parallel()
	ts := newAzureResourcesTestServer(t)
	headers := azureResourcesTestHeaders()
	rgPath := "/azure/subscriptions/" + azureResourcesTestSubscription + "/resourcegroups/rg-life?api-version=2021-04-01"

	resp := providerContractRequest(t, ts, http.MethodPut, rgPath, []byte(`{"location":"eastus","tags":{"env":"dev"}}`), headers)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	created := providerContractJSONMap(t, resp)
	if created["id"] != "/subscriptions/"+azureResourcesTestSubscription+"/resourceGroups/rg-life" {
		t.Fatalf("unexpected id: %#v", created["id"])
	}
	if created["type"] != "Microsoft.Resources/resourceGroups" || created["location"] != "eastus" {
		t.Fatalf("unexpected resource group payload: %#v", created)
	}
	if props, _ := created["properties"].(map[string]any); props["provisioningState"] != "Succeeded" {
		t.Fatalf("expected provisioningState Succeeded, got %#v", created["properties"])
	}

	resp = providerContractRequest(t, ts, http.MethodPut, rgPath, []byte(`{"location":"eastus","tags":{"env":"prod"}}`), headers)
	azureResourcesExpectStatus(t, resp, http.StatusOK)

	resp = providerContractRequest(t, ts, http.MethodHead, rgPath, nil, headers)
	azureResourcesExpectStatus(t, resp, http.StatusNoContent)

	resp = providerContractRequest(t, ts, http.MethodPatch, rgPath, []byte(`{"tags":{"team":"core"}}`), headers)
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	patched := providerContractJSONMap(t, resp)
	if tags, _ := patched["tags"].(map[string]any); tags["team"] != "core" || tags["env"] != nil {
		t.Fatalf("expected PATCH to replace tags, got %#v", patched["tags"])
	}

	// ARM treats resource group names case-insensitively.
	resp = providerContractRequest(t, ts, http.MethodGet, "/azure/subscriptions/"+azureResourcesTestSubscription+"/resourceGroups/RG-LIFE?api-version=2021-04-01", nil, headers)
	azureResourcesExpectStatus(t, resp, http.StatusOK)

	resp = providerContractRequest(t, ts, http.MethodGet, "/azure/subscriptions/"+azureResourcesTestSubscription+"/resourcegroups?api-version=2021-04-01", nil, headers)
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if items, _ := providerContractJSONMap(t, resp)["value"].([]any); len(items) != 1 {
		t.Fatalf("expected one resource group in list, got %#v", items)
	}

	resp = providerContractRequest(t, ts, http.MethodDelete, rgPath, nil, headers)
	azureResourcesExpectStatus(t, resp, http.StatusOK)

	resp = providerContractRequest(t, ts, http.MethodGet, rgPath, nil, headers)
	azureResourcesExpectErrorCode(t, resp, http.StatusNotFound, "ResourceGroupNotFound")

	resp = providerContractRequest(t, ts, http.MethodHead, rgPath, nil, headers)
	azureResourcesExpectStatus(t, resp, http.StatusNotFound)

	resp = providerContractRequest(t, ts, http.MethodDelete, rgPath, nil, headers)
	azureResourcesExpectErrorCode(t, resp, http.StatusNotFound, "ResourceGroupNotFound")
}

func TestAzureResourceGroupValidation(t *testing.T) {
	t.Parallel()
	ts := newAzureResourcesTestServer(t)
	headers := azureResourcesTestHeaders()
	rgPath := "/azure/subscriptions/" + azureResourcesTestSubscription + "/resourcegroups/rg-bad?api-version=2021-04-01"

	tests := []struct {
		name   string
		method string
		path   string
		body   []byte
		status int
		code   string
	}{
		{name: "missing location", method: http.MethodPut, path: rgPath, body: []byte(`{"tags":{}}`), status: http.StatusBadRequest, code: "LocationRequired"},
		{name: "malformed json", method: http.MethodPut, path: rgPath, body: []byte(`{"location":`), status: http.StatusBadRequest, code: "InvalidRequestContent"},
		{name: "patch missing group", method: http.MethodPatch, path: rgPath, body: []byte(`{"tags":{}}`), status: http.StatusNotFound, code: "ResourceGroupNotFound"},
		{name: "unsupported method", method: http.MethodPost, path: rgPath, status: http.StatusMethodNotAllowed, code: "MethodNotAllowed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := providerContractRequest(t, ts, tt.method, tt.path, tt.body, headers)
			azureResourcesExpectErrorCode(t, resp, tt.status, tt.code)
		})
	}

	resp := providerContractRequest(t, ts, http.MethodGet, "/azure/subscriptions/"+azureResourcesTestSubscription+"/resourcegroups/rg-bad?api-version=", nil, headers)
	azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, "InvalidApiVersionParameter")

	// A resource group's location is fixed once created.
	resp = providerContractRequest(t, ts, http.MethodPut, rgPath, []byte(`{"location":"eastus"}`), headers)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	resp = providerContractRequest(t, ts, http.MethodPut, rgPath, []byte(`{"location":"westus"}`), headers)
	azureResourcesExpectErrorCode(t, resp, http.StatusConflict, "InvalidResourceGroupLocation")
}

func TestAzureSubscriptionsAndProviders(t *testing.T) {
	t.Parallel()
	ts := newAzureResourcesTestServer(t)
	headers := azureResourcesTestHeaders()
	sub := "/azure/subscriptions/" + azureResourcesTestSubscription

	resp := providerContractRequest(t, ts, http.MethodGet, "/azure/subscriptions?api-version=2022-12-01", nil, headers)
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if items, _ := providerContractJSONMap(t, resp)["value"].([]any); len(items) != 1 {
		t.Fatalf("expected default subscription in list, got %#v", items)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, sub+"?api-version=2022-12-01", nil, headers)
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if got := providerContractJSONMap(t, resp); got["subscriptionId"] != azureResourcesTestSubscription || got["state"] != "Enabled" {
		t.Fatalf("unexpected subscription payload: %#v", got)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, sub+"/providers?api-version=2021-04-01", nil, headers)
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if !azureResourcesListHasNamespace(providerContractJSONMap(t, resp), "Microsoft.ManagedIdentity") {
		t.Fatal("expected Microsoft.ManagedIdentity in provider list")
	}

	resp = providerContractRequest(t, ts, http.MethodPost, sub+"/providers/Microsoft.ManagedIdentity/register?api-version=2021-04-01", nil, headers)
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if got := providerContractJSONMap(t, resp); got["registrationState"] != "Registered" {
		t.Fatalf("expected Registered, got %#v", got)
	}

	// Every well-formed namespace reports Registered so tools like Terraform's
	// azurerm provider can complete their startup registration pass.
	resp = providerContractRequest(t, ts, http.MethodGet, sub+"/providers/Microsoft.Compute?api-version=2021-04-01", nil, headers)
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if got := providerContractJSONMap(t, resp); got["registrationState"] != "Registered" {
		t.Fatalf("expected Registered, got %#v", got)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, sub+"/providers/not-a-namespace?api-version=2021-04-01", nil, headers)
	azureResourcesExpectErrorCode(t, resp, http.StatusNotFound, "InvalidResourceNamespace")
}

func azureResourcesListHasNamespace(payload map[string]any, namespace string) bool {
	items, _ := payload["value"].([]any)
	for _, item := range items {
		if entry, _ := item.(map[string]any); entry["namespace"] == namespace {
			return true
		}
	}
	return false
}
