package server

import (
	"net/http"
	"testing"
)

func TestAzureMSIUserAssignedIdentityLifecycle(t *testing.T) {
	t.Parallel()
	ts := newAzureResourcesTestServer(t)
	headers := azureResourcesTestHeaders()
	sub := "/azure/subscriptions/" + azureResourcesTestSubscription
	idPath := sub + "/resourceGroups/rg-msi/providers/Microsoft.ManagedIdentity/userAssignedIdentities/id-a?api-version=2024-11-30"

	// Creating into a missing resource group fails like real ARM.
	resp := providerContractRequest(t, ts, http.MethodPut, idPath, []byte(`{"location":"eastus"}`), headers)
	azureResourcesExpectErrorCode(t, resp, http.StatusNotFound, "ResourceGroupNotFound")

	resp = providerContractRequest(t, ts, http.MethodPut, sub+"/resourcegroups/rg-msi?api-version=2021-04-01", []byte(`{"location":"eastus"}`), headers)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)

	resp = providerContractRequest(t, ts, http.MethodPut, idPath, []byte(`{"location":"eastus","tags":{"env":"dev"}}`), headers)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	created := providerContractJSONMap(t, resp)
	if created["type"] != "Microsoft.ManagedIdentity/userAssignedIdentities" || created["name"] != "id-a" {
		t.Fatalf("unexpected identity payload: %#v", created)
	}
	props, _ := created["properties"].(map[string]any)
	for _, field := range []string{"principalId", "clientId", "tenantId"} {
		if s, _ := props[field].(string); s == "" {
			t.Fatalf("expected generated %s, got %#v", field, props)
		}
	}

	resp = providerContractRequest(t, ts, http.MethodPut, idPath, []byte(`{"location":"eastus"}`), headers)
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if again, _ := providerContractJSONMap(t, resp)["properties"].(map[string]any); again["principalId"] != props["principalId"] {
		t.Fatalf("expected principalId to stay stable across updates, got %#v vs %#v", again["principalId"], props["principalId"])
	}

	resp = providerContractRequest(t, ts, http.MethodHead, idPath, nil, headers)
	azureResourcesExpectStatus(t, resp, http.StatusNoContent)

	resp = providerContractRequest(t, ts, http.MethodPatch, idPath, []byte(`{"tags":{"team":"core"}}`), headers)
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if tags, _ := providerContractJSONMap(t, resp)["tags"].(map[string]any); tags["team"] != "core" {
		t.Fatalf("expected patched tags, got %#v", tags)
	}

	for _, listPath := range []string{
		sub + "/resourceGroups/rg-msi/providers/Microsoft.ManagedIdentity/userAssignedIdentities?api-version=2024-11-30",
		sub + "/providers/Microsoft.ManagedIdentity/userAssignedIdentities?api-version=2024-11-30",
		sub + "/resourcegroups/rg-msi/resources?api-version=2021-04-01",
	} {
		resp = providerContractRequest(t, ts, http.MethodGet, listPath, nil, headers)
		azureResourcesExpectStatus(t, resp, http.StatusOK)
		if items, _ := providerContractJSONMap(t, resp)["value"].([]any); len(items) != 1 {
			t.Fatalf("expected one identity from %s, got %#v", listPath, items)
		}
	}

	// Deleting the resource group cascades to its resources.
	resp = providerContractRequest(t, ts, http.MethodDelete, sub+"/resourcegroups/rg-msi?api-version=2021-04-01", nil, headers)
	azureResourcesExpectStatus(t, resp, http.StatusOK)

	resp = providerContractRequest(t, ts, http.MethodGet, idPath, nil, headers)
	azureResourcesExpectErrorCode(t, resp, http.StatusNotFound, "ResourceGroupNotFound")
}

func TestAzureMSIServerOwnedFieldsAndCasing(t *testing.T) {
	t.Parallel()
	ts := newAzureResourcesTestServer(t)
	headers := azureResourcesTestHeaders()
	sub := "/azure/subscriptions/" + azureResourcesTestSubscription
	resp := providerContractRequest(t, ts, http.MethodPut, sub+"/resourcegroups/rg-Case?api-version=2021-04-01", []byte(`{"location":"eastus"}`), headers)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)

	idPath := sub + "/resourceGroups/rg-Case/providers/Microsoft.ManagedIdentity/userAssignedIdentities/id-Case?api-version=2024-11-30"
	resp = providerContractRequest(t, ts, http.MethodPut, idPath, []byte(`{"location":"eastus"}`), headers)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	original, _ := providerContractJSONMap(t, resp)["properties"].(map[string]any)
	wantID := "/subscriptions/" + azureResourcesTestSubscription + "/resourceGroups/rg-Case/providers/Microsoft.ManagedIdentity/userAssignedIdentities/id-Case"

	// Server-owned properties survive a PATCH that tries to overwrite them.
	resp = providerContractRequest(t, ts, http.MethodPatch, idPath, []byte(`{"properties":{"principalId":"evil","provisioningState":"Failed"}}`), headers)
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	patched, _ := providerContractJSONMap(t, resp)["properties"].(map[string]any)
	if patched["principalId"] != original["principalId"] || patched["provisioningState"] != "Succeeded" {
		t.Fatalf("expected server-owned properties to be preserved, got %#v", patched)
	}

	// A PUT with different casing updates the same resource and keeps its canonical ID.
	upper := sub + "/resourceGroups/RG-CASE/providers/Microsoft.ManagedIdentity/userAssignedIdentities/ID-CASE?api-version=2024-11-30"
	resp = providerContractRequest(t, ts, http.MethodPut, upper, []byte(`{"location":"eastus"}`), headers)
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if got := providerContractJSONMap(t, resp); got["id"] != wantID || got["name"] != "id-Case" {
		t.Fatalf("expected canonical id %q and name id-Case, got %#v / %#v", wantID, got["id"], got["name"])
	}

	// Lists match subscriptions case-insensitively, like GET does.
	resp = providerContractRequest(t, ts, http.MethodGet, "/azure/subscriptions/"+azureResourcesTestSubscription+"/providers/Microsoft.ManagedIdentity/userAssignedIdentities?api-version=2024-11-30", nil, headers)
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if items, _ := providerContractJSONMap(t, resp)["value"].([]any); len(items) != 1 {
		t.Fatalf("expected one identity, got %#v", items)
	}

	// A resource's location is fixed once created.
	resp = providerContractRequest(t, ts, http.MethodPut, idPath, []byte(`{"location":"westus"}`), headers)
	azureResourcesExpectErrorCode(t, resp, http.StatusConflict, "InvalidResourceLocation")
}

func TestRegisterAzureARMResourceTypeRejectsDuplicates(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected duplicate registration to panic")
		}
	}()
	registerAzureARMResourceType(azureARMResourceType{Namespace: "microsoft.managedidentity", Type: "UserAssignedIdentities"})
}

func TestAzureMSIValidation(t *testing.T) {
	t.Parallel()
	ts := newAzureResourcesTestServer(t)
	headers := azureResourcesTestHeaders()
	sub := "/azure/subscriptions/" + azureResourcesTestSubscription
	resp := providerContractRequest(t, ts, http.MethodPut, sub+"/resourcegroups/rg-msi-bad?api-version=2021-04-01", []byte(`{"location":"eastus"}`), headers)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	idPath := sub + "/resourceGroups/rg-msi-bad/providers/Microsoft.ManagedIdentity/userAssignedIdentities/id-x?api-version=2024-11-30"

	tests := []struct {
		name   string
		method string
		body   []byte
		status int
		code   string
	}{
		{name: "missing location", method: http.MethodPut, body: []byte(`{}`), status: http.StatusBadRequest, code: "LocationRequired"},
		{name: "malformed json", method: http.MethodPut, body: []byte(`not-json`), status: http.StatusBadRequest, code: "InvalidRequestContent"},
		{name: "get missing", method: http.MethodGet, status: http.StatusNotFound, code: "ResourceNotFound"},
		{name: "patch missing", method: http.MethodPatch, body: []byte(`{"tags":{}}`), status: http.StatusNotFound, code: "ResourceNotFound"},
		{name: "unsupported method", method: http.MethodPost, status: http.StatusMethodNotAllowed, code: "MethodNotAllowed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := providerContractRequest(t, ts, tt.method, idPath, tt.body, headers)
			azureResourcesExpectErrorCode(t, resp, tt.status, tt.code)
		})
	}

	resp = providerContractRequest(t, ts, http.MethodDelete, idPath, nil, headers)
	azureResourcesExpectStatus(t, resp, http.StatusNoContent)
}
