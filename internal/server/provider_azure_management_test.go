package server

import (
	"net/http"
	"testing"
)

const azureManagementTestAPIVersion = "?api-version=2021-04-01"

func TestAzureManagementGroupHierarchy(t *testing.T) {
	t.Parallel()
	_, do := newAzureDatabaseTestServer(t, "rg-mg")
	groups := "/azure/providers/Microsoft.Management/managementGroups/"
	root := groups + azureDefaultTenantID

	resp := do(http.MethodGet, root+azureManagementTestAPIVersion, "")
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if got := providerContractJSONMap(t, resp)["properties"].(map[string]any)["displayName"]; got != "Tenant Root Group" {
		t.Fatalf("unexpected root group name %v", got)
	}

	// A new group goes under the root unless it names a parent.
	resp = do(http.MethodPut, groups+"platform"+azureManagementTestAPIVersion, `{"properties":{"displayName":"Platform"}}`)
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	parent := providerContractJSONMap(t, resp)["properties"].(map[string]any)["details"].(map[string]any)["parent"].(map[string]any)
	if parent["name"] != azureDefaultTenantID {
		t.Fatalf("expected the root as parent, got %#v", parent)
	}
	azureResourcesExpectStatus(t, do(http.MethodPut, groups+"connectivity"+azureManagementTestAPIVersion, `{"properties":{"displayName":"Connectivity","details":{"parent":{"id":"/providers/Microsoft.Management/managementGroups/platform"}}}}`), http.StatusOK)

	// A client's own tenant ID names the root.
	resp = do(http.MethodPut, groups+"sandbox"+azureManagementTestAPIVersion, `{"properties":{"details":{"parent":{"id":"/providers/Microsoft.Management/managementGroups/72f988bf-86f1-41af-91ab-2d7cd011db47"}}}}`)
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if got := providerContractJSONMap(t, resp)["properties"].(map[string]any)["details"].(map[string]any)["parent"].(map[string]any)["name"]; got != azureDefaultTenantID {
		t.Fatalf("expected the root as parent, got %v", got)
	}
	azureResourcesExpectStatus(t, do(http.MethodDelete, groups+"sandbox"+azureManagementTestAPIVersion, ""), http.StatusOK)

	sub := "11111111-2222-3333-4444-555555555555"
	resp = do(http.MethodPut, groups+"connectivity/subscriptions/"+sub+azureManagementTestAPIVersion, "")
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if got := providerContractJSONMap(t, resp)["properties"].(map[string]any)["parent"].(map[string]any)["id"]; got != "/providers/Microsoft.Management/managementGroups/connectivity" {
		t.Fatalf("unexpected subscription parent %v", got)
	}

	// $expand=children&$recurse=true shows the whole tree.
	resp = do(http.MethodGet, groups+"platform"+azureManagementTestAPIVersion+"&$expand=children&$recurse=true", "")
	children := providerContractJSONMap(t, resp)["properties"].(map[string]any)["children"].([]any)
	connectivity := children[0].(map[string]any)
	if len(children) != 1 || connectivity["name"] != "connectivity" {
		t.Fatalf("unexpected children: %#v", children)
	}
	if grand := connectivity["children"].([]any); len(grand) != 1 || grand[0].(map[string]any)["id"] != "/subscriptions/"+sub {
		t.Fatalf("unexpected grandchildren: %#v", grand)
	}
	resp = do(http.MethodGet, groups+"platform/descendants"+azureManagementTestAPIVersion, "")
	descendants := providerContractJSONMap(t, resp)["value"].([]any)
	if len(descendants) != 2 || descendants[1].(map[string]any)["type"] != "/subscriptions" {
		t.Fatalf("unexpected descendants: %#v", descendants)
	}

	// A group can't move under its own descendant, and one with children can't be deleted.
	azureResourcesExpectErrorCode(t, do(http.MethodPatch, groups+"platform"+azureManagementTestAPIVersion, `{"parentGroupId":"connectivity"}`), http.StatusBadRequest, "InvalidRequestContent")
	azureResourcesExpectErrorCode(t, do(http.MethodDelete, groups+"connectivity"+azureManagementTestAPIVersion, ""), http.StatusBadRequest, "InvalidRequestContent")
	azureResourcesExpectErrorCode(t, do(http.MethodDelete, root+azureManagementTestAPIVersion, ""), http.StatusBadRequest, "InvalidRequestContent")

	// Moving the subscription to platform takes it out of connectivity.
	azureResourcesExpectStatus(t, do(http.MethodPut, groups+"platform/subscriptions/"+sub+azureManagementTestAPIVersion, ""), http.StatusOK)
	azureResourcesExpectErrorCode(t, do(http.MethodGet, groups+"connectivity/subscriptions/"+sub+azureManagementTestAPIVersion, ""), http.StatusNotFound, "NotFound")
	azureResourcesExpectStatus(t, do(http.MethodDelete, groups+"connectivity"+azureManagementTestAPIVersion, ""), http.StatusOK)

	// Removing it from platform puts it back under the root.
	azureResourcesExpectStatus(t, do(http.MethodDelete, groups+"platform/subscriptions/"+sub+azureManagementTestAPIVersion, ""), http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodGet, root+"/subscriptions/"+sub+azureManagementTestAPIVersion, ""), http.StatusOK)

	resp = do(http.MethodPatch, groups+"platform"+azureManagementTestAPIVersion, `{"displayName":"Platform Team"}`)
	if props := providerContractJSONMap(t, resp)["properties"].(map[string]any); props["displayName"] != "Platform Team" || props["details"].(map[string]any)["version"] != float64(2) {
		t.Fatalf("unexpected PATCH result: %#v", props)
	}
	resp = do(http.MethodGet, "/azure/providers/Microsoft.Management/managementGroups"+azureManagementTestAPIVersion, "")
	if got := providerContractJSONMap(t, resp)["value"].([]any); len(got) != 2 {
		t.Fatalf("expected root and platform, got %#v", got)
	}
	azureResourcesExpectStatus(t, do(http.MethodDelete, groups+"platform"+azureManagementTestAPIVersion, ""), http.StatusOK)
	azureResourcesExpectErrorCode(t, do(http.MethodGet, groups+"platform"+azureManagementTestAPIVersion, ""), http.StatusNotFound, "NotFound")
}

func TestAzureManagementGroupValidation(t *testing.T) {
	t.Parallel()
	_, do := newAzureDatabaseTestServer(t, "rg-mg")
	groups := "/azure/providers/Microsoft.Management/managementGroups/"
	azureResourcesExpectErrorCode(t, do(http.MethodPut, groups+"bad.name."+azureManagementTestAPIVersion, `{}`), http.StatusBadRequest, "InvalidResourceName")
	azureResourcesExpectErrorCode(t, do(http.MethodPut, groups+"orphan"+azureManagementTestAPIVersion, `{"properties":{"details":{"parent":{"id":"/providers/Microsoft.Management/managementGroups/missing"}}}}`), http.StatusBadRequest, "InvalidRequestContent")
	azureResourcesExpectErrorCode(t, do(http.MethodPut, groups+"orphan"+azureManagementTestAPIVersion, `{"properties":{"details":{"parent":{"id":"/subscriptions/x"}}}}`), http.StatusBadRequest, "InvalidRequestContent")
	azureResourcesExpectErrorCode(t, do(http.MethodPatch, groups+"missing"+azureManagementTestAPIVersion, `{"displayName":"x"}`), http.StatusNotFound, "NotFound")
	azureResourcesExpectErrorCode(t, do(http.MethodPut, groups+azureDefaultTenantID+"/subscriptions/not-a-guid"+azureManagementTestAPIVersion, ""), http.StatusBadRequest, "InvalidSubscriptionId")
	azureResourcesExpectErrorCode(t, do(http.MethodPut, groups+"missing/subscriptions/11111111-2222-3333-4444-555555555555"+azureManagementTestAPIVersion, ""), http.StatusNotFound, "NotFound")

	azureResourcesExpectStatus(t, do(http.MethodPut, groups+"taken"+azureManagementTestAPIVersion, `{}`), http.StatusOK)
	for name, reason := range map[string]any{"taken": "AlreadyExists", "bad.": "Invalid", "free": nil} {
		resp := do(http.MethodPost, "/azure/providers/Microsoft.Management/checkNameAvailability"+azureManagementTestAPIVersion, `{"name":"`+name+`","type":"Microsoft.Management/managementGroups"}`)
		azureResourcesExpectStatus(t, resp, http.StatusOK)
		if got := providerContractJSONMap(t, resp); got["reason"] != reason || got["nameAvailable"] != (reason == nil) {
			t.Fatalf("%s: unexpected %#v", name, got)
		}
	}
}
