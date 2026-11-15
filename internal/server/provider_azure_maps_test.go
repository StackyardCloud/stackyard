package server

import (
	"net/http"
	"strings"
	"testing"
)

const azureMapsTestAPIVersion = "?api-version=2023-06-01"

func TestAzureMapsAccountLifecycle(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-maps")
	account := sub + "/resourceGroups/rg-maps/providers/Microsoft.Maps/accounts/fleet"

	resp := do(http.MethodPut, account+azureMapsTestAPIVersion, `{"location":"eastus","sku":{"name":"G2"},"kind":"Gen2","properties":{"disableLocalAuth":false}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	if id := providerContractJSONMap(t, resp)["properties"].(map[string]any)["uniqueId"]; len(id.(string)) != 36 {
		t.Fatalf("unexpected uniqueId %v", id)
	}

	resp = do(http.MethodPost, account+"/listKeys"+azureMapsTestAPIVersion, "")
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	keys := providerContractJSONMap(t, resp)
	if primary := keys["primaryKey"].(string); primary == "" || primary == keys["secondaryKey"] || strings.ContainsAny(primary, "+/=") {
		t.Fatalf("unexpected keys: %#v", keys)
	}

	sas := `{"signingKey":"primaryKey","principalId":"e917f87b-324d-4c3e-8d0d-1c8d2b8d1c4b","maxRatePerSecond":500,"start":"2026-01-01T00:00:00Z","expiry":"2026-01-02T00:00:00Z"}`
	resp = do(http.MethodPost, account+"/listSas"+azureMapsTestAPIVersion, sas)
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if token := providerContractJSONMap(t, resp)["accountSasToken"].(string); strings.Count(token, ".") != 1 {
		t.Fatalf("unexpected SAS token %q", token)
	}
	for name, body := range map[string]string{
		"bad signing key": strings.Replace(sas, `"primaryKey"`, `"tertiaryKey"`, 1),
		"rate too high":   strings.Replace(sas, `500`, `501`, 1),
		"expiry first":    strings.Replace(sas, `2026-01-02`, `2025-12-31`, 1),
	} {
		resp := do(http.MethodPost, account+"/listSas"+azureMapsTestAPIVersion, body)
		t.Run(name, func(t *testing.T) {
			azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, "InvalidRequestContent")
		})
	}

	azureResourcesExpectErrorCode(t, do(http.MethodPut, sub+"/resourceGroups/rg-maps/providers/Microsoft.Maps/accounts/bad"+azureMapsTestAPIVersion, `{"location":"eastus","sku":{"name":"S2"}}`), http.StatusBadRequest, "InvalidRequestContent")
	azureResourcesExpectErrorCode(t, do(http.MethodPut, sub+"/resourceGroups/rg-maps/providers/Microsoft.Maps/accounts/bad"+azureMapsTestAPIVersion, `{"location":"eastus","sku":{"name":"G2"},"kind":"Gen3"}`), http.StatusBadRequest, "InvalidRequestContent")
	azureResourcesExpectStatus(t, do(http.MethodDelete, account+azureMapsTestAPIVersion, ""), http.StatusOK)
}
