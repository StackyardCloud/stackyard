package server

import (
	"net/http"
	"testing"
)

const aduRMTestAPIVersion = "?api-version=2023-07-01"

func TestAzureDeviceUpdateResourceManager20230701Lifecycle(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-adu")
	account := sub + "/resourceGroups/rg-adu/providers/Microsoft.DeviceUpdate/accounts/contoso-adu"

	resp := do(http.MethodPut, account+aduRMTestAPIVersion, `{"location":"eastus","identity":{"type":"SystemAssigned"},"properties":{"sku":"Standard","publicNetworkAccess":"Enabled"}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	if host := providerContractJSONMap(t, resp)["properties"].(map[string]any)["hostName"]; host != "contoso-adu.api.adu.microsoft.com" {
		t.Fatalf("unexpected hostName %v", host)
	}
	resp = do(http.MethodPut, account+"/instances/main"+aduRMTestAPIVersion, `{"location":"eastus","properties":{"iotHubs":[{"resourceId":"/subscriptions/s/resourceGroups/rg/providers/Microsoft.Devices/IotHubs/hub"}]}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	if name := providerContractJSONMap(t, resp)["properties"].(map[string]any)["accountName"]; name != "contoso-adu" {
		t.Fatalf("unexpected accountName %v", name)
	}
	azureResourcesExpectStatus(t, do(http.MethodPut, account+"/privateEndpointConnections/pe"+aduRMTestAPIVersion, `{"properties":{"privateLinkServiceConnectionState":{"status":"Approved","description":"ok"}}}`), http.StatusCreated)
	azureResourcesExpectStatus(t, do(http.MethodPut, account+"/privateEndpointConnectionProxies/proxy"+aduRMTestAPIVersion, `{"remotePrivateEndpoint":{"id":"/subscriptions/s/resourceGroups/rg/providers/Microsoft.Network/privateEndpoints/pe"}}`), http.StatusCreated)
	azureResourcesExpectStatus(t, do(http.MethodPost, account+"/privateEndpointConnectionProxies/proxy/validate"+aduRMTestAPIVersion, `{"remotePrivateEndpoint":{"id":"pe"}}`), http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodGet, account+"/privateLinkResources/DeviceUpdate"+aduRMTestAPIVersion, ""), http.StatusOK)

	resp = do(http.MethodPost, sub+"/providers/Microsoft.DeviceUpdate/checkNameAvailability"+aduRMTestAPIVersion, `{"name":"contoso-adu","type":"Microsoft.DeviceUpdate/accounts"}`)
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if providerContractJSONMap(t, resp)["nameAvailable"] != false {
		t.Fatal("expected the account name to be taken")
	}
	azureResourcesExpectStatus(t, do(http.MethodGet, "/azure/providers/Microsoft.DeviceUpdate/operations"+aduRMTestAPIVersion, ""), http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodDelete, account+aduRMTestAPIVersion, ""), http.StatusOK)
	azureResourcesExpectErrorCode(t, do(http.MethodGet, account+"/instances/main"+aduRMTestAPIVersion, ""), http.StatusNotFound, "ParentResourceNotFound")
}

func TestAzureDeviceUpdateResourceManager20230701Validation(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-adu")
	base := sub + "/resourceGroups/rg-adu/providers/Microsoft.DeviceUpdate/accounts/"
	azureResourcesExpectStatus(t, do(http.MethodPut, base+"acct"+aduRMTestAPIVersion, `{"location":"eastus"}`), http.StatusCreated)
	cases := []struct{ name, path, body string }{
		{"name", base + "a-", `{"location":"eastus"}`},
		{"sku", base + "acct2", `{"location":"eastus","properties":{"sku":"Premium"}}`},
		{"iot hub", base + "acct/instances/i", `{"location":"eastus","properties":{"iotHubs":[{}]}}`},
		{"connection state", base + "acct/privateEndpointConnections/pe", `{"properties":{"privateLinkServiceConnectionState":{"status":"Maybe"}}}`},
	}
	for _, tc := range cases {
		resp := do(http.MethodPut, tc.path+aduRMTestAPIVersion, tc.body)
		t.Run(tc.name, func(t *testing.T) { azureResourcesExpectStatus(t, resp, http.StatusBadRequest) })
	}
	azureResourcesExpectStatus(t, do(http.MethodPost, sub+"/providers/Microsoft.DeviceUpdate/checkNameAvailability"+aduRMTestAPIVersion, `{"name":"x"}`), http.StatusBadRequest)
}
