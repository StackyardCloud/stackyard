package server

import (
	"net/http"
	"testing"
)

const azureDFCTestAPI = "?api-version=2025-06-01"

func TestAzureDefenderForCloud20250601MalwareScans(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-dfc")
	setting := sub + "/resourceGroups/rg-dfc/providers/Microsoft.Storage/storageAccounts/samplesa/providers/Microsoft.Security/defenderForStorageSettings/current"

	azureResourcesExpectStatus(t, do(http.MethodGet, setting+"/malwareScans/latest"+azureDFCTestAPI, ""), http.StatusNotFound)
	resp := do(http.MethodPut, setting+azureDFCTestAPI, `{"properties":{"isEnabled":true,"malwareScanning":{"onUpload":{"isEnabled":true,"capGBPerMonth":-1}}}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	scanning := providerContractJSONMap(t, resp)["properties"].(map[string]any)["malwareScanning"].(map[string]any)
	if scanning["operationStatus"].(map[string]any)["code"] != "Succeeded" {
		t.Fatalf("unexpected malwareScanning %#v", scanning)
	}

	azureResourcesExpectStatus(t, do(http.MethodGet, setting+"/malwareScans/latest"+azureDFCTestAPI, ""), http.StatusNotFound)
	scan := providerContractJSONMap(t, do(http.MethodPost, setting+"/startMalwareScan"+azureDFCTestAPI, ""))
	scanID := scan["name"].(string)
	latest := providerContractJSONMap(t, do(http.MethodGet, setting+"/malwareScans/latest"+azureDFCTestAPI, ""))
	if latest["name"] != scanID || latest["properties"].(map[string]any)["scanStatus"] != "Running" {
		t.Fatalf("latest scan %#v", latest)
	}
	canceled := providerContractJSONMap(t, do(http.MethodPost, setting+"/malwareScans/"+scanID+"/cancelMalwareScan"+azureDFCTestAPI, ""))
	if canceled["properties"].(map[string]any)["scanStatus"] != "Canceled" {
		t.Fatalf("canceled scan %#v", canceled)
	}
	azureResourcesExpectStatus(t, do(http.MethodPost, setting+"/malwareScans/"+scanID+"/cancelMalwareScan"+azureDFCTestAPI, ""), http.StatusConflict)
	azureResourcesExpectStatus(t, do(http.MethodGet, setting+"/malwareScans/not-a-guid"+azureDFCTestAPI, ""), http.StatusBadRequest)
	azureResourcesExpectStatus(t, do(http.MethodDelete, setting+azureDFCTestAPI, ""), http.StatusMethodNotAllowed)
}

func TestAzureDefenderForCloud20250601Validation(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-dfc")
	account := sub + "/resourceGroups/rg-dfc/providers/Microsoft.Storage/storageAccounts/samplesa"
	setting := account + "/providers/Microsoft.Security/defenderForStorageSettings/"
	azureResourcesExpectErrorCode(t, do(http.MethodPut, setting+"other"+azureDFCTestAPI, `{"properties":{"isEnabled":true}}`), http.StatusBadRequest, "InvalidResourceName")
	azureResourcesExpectErrorCode(t, do(http.MethodPut, setting+"current"+azureDFCTestAPI, `{"properties":{"isEnabled":"yes"}}`), http.StatusBadRequest, "InvalidRequestContent")
	azureResourcesExpectErrorCode(t, do(http.MethodPut, sub+"/resourceGroups/rg-dfc/providers/Microsoft.Security/defenderForStorageSettings/current"+azureDFCTestAPI, `{"properties":{}}`), http.StatusBadRequest, "InvalidScope")
	azureResourcesExpectStatus(t, do(http.MethodPut, setting+"current"+azureDFCTestAPI, `{"properties":{"isEnabled":false}}`), http.StatusCreated)
	azureResourcesExpectStatus(t, do(http.MethodPost, setting+"current/startMalwareScan"+azureDFCTestAPI, ""), http.StatusConflict)
}
