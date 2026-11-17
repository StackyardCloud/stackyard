package server

import (
	"net/http"
	"testing"
)

const azureManagementPartnerTestAPIVersion = "?api-version=2018-02-01"

func TestAzureManagementPartnerLink(t *testing.T) {
	t.Parallel()
	_, do := newAzureDatabaseTestServer(t, "rg-partner")
	partners := "/azure/providers/Microsoft.ManagementPartner/partners"
	azureResourcesExpectErrorCode(t, do(http.MethodGet, partners+azureManagementPartnerTestAPIVersion, ""), http.StatusNotFound, "ResourceNotFound")
	azureResourcesExpectErrorCode(t, do(http.MethodPatch, partners+"/123456"+azureManagementPartnerTestAPIVersion, ""), http.StatusNotFound, "ResourceNotFound")

	body := providerContractJSONMap(t, do(http.MethodPut, partners+"/123456"+azureManagementPartnerTestAPIVersion, ""))
	props := body["properties"].(map[string]any)
	if body["id"] != "/providers/Microsoft.ManagementPartner/partners/123456" || body["type"] != "Microsoft.ManagementPartner/partner" || props["partnerId"] != "123456" || props["state"] != "Active" || props["version"] != float64(1) {
		t.Fatalf("unexpected partner: %#v", body)
	}
	// Linking the same ID again is a no-op; another ID needs PATCH.
	azureResourcesExpectStatus(t, do(http.MethodPut, partners+"/123456"+azureManagementPartnerTestAPIVersion, ""), http.StatusOK)
	azureResourcesExpectErrorCode(t, do(http.MethodPut, partners+"/654321"+azureManagementPartnerTestAPIVersion, ""), http.StatusConflict, "Conflict")
	azureResourcesExpectErrorCode(t, do(http.MethodPut, partners+"/abc"+azureManagementPartnerTestAPIVersion, ""), http.StatusBadRequest, "BadRequest")

	updated := providerContractJSONMap(t, do(http.MethodPatch, partners+"/654321"+azureManagementPartnerTestAPIVersion, ""))
	if p := updated["properties"].(map[string]any); p["partnerId"] != "654321" || p["version"] != float64(2) || updated["etag"] != float64(2) {
		t.Fatalf("PATCH should move the link to the new ID: %#v", updated)
	}
	if got := providerContractJSONMap(t, do(http.MethodGet, partners+azureManagementPartnerTestAPIVersion, ""))["name"]; got != "654321" {
		t.Fatalf("GET /partners should return the one link, got %v", got)
	}
	azureResourcesExpectErrorCode(t, do(http.MethodGet, partners+"/123456"+azureManagementPartnerTestAPIVersion, ""), http.StatusNotFound, "ResourceNotFound")
	azureResourcesExpectErrorCode(t, do(http.MethodDelete, partners+"/123456"+azureManagementPartnerTestAPIVersion, ""), http.StatusNotFound, "ResourceNotFound")
	azureResourcesExpectStatus(t, do(http.MethodDelete, partners+"/654321"+azureManagementPartnerTestAPIVersion, ""), http.StatusOK)
	azureResourcesExpectErrorCode(t, do(http.MethodGet, partners+azureManagementPartnerTestAPIVersion, ""), http.StatusNotFound, "ResourceNotFound")
	azureResourcesExpectStatus(t, do(http.MethodPost, partners+azureManagementPartnerTestAPIVersion, ""), http.StatusMethodNotAllowed)
}
