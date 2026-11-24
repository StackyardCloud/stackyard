package server

import (
	"net/http"
	"strings"
	"testing"
)

const adrTestAPIVersion = "?api-version=2026-04-01"

const adrTestEdge = `"extendedLocation":{"type":"CustomLocation","name":"/subscriptions/s/resourceGroups/rg/providers/Microsoft.ExtendedLocation/customLocations/edge"}`

func TestAzureDeviceRegistry20260401Lifecycle(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-adr")
	base := sub + "/resourceGroups/rg-adr/providers/Microsoft.DeviceRegistry/"

	resp := do(http.MethodPut, base+"assetEndpointProfiles/opc"+adrTestAPIVersion, `{"location":"eastus",`+adrTestEdge+`,"properties":{"endpointProfileType":"OpcUa","targetAddress":"opc.tcp://plc:4840"}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	if uuid, _ := providerContractJSONMap(t, resp)["properties"].(map[string]any)["uuid"].(string); len(uuid) != 36 {
		t.Fatalf("unexpected uuid %q", uuid)
	}
	azureResourcesExpectStatus(t, do(http.MethodPut, base+"assets/pump"+adrTestAPIVersion, `{"location":"eastus",`+adrTestEdge+`,"properties":{"assetEndpointProfileRef":"opc"}}`), http.StatusCreated)

	ns := base + "namespaces/plant"
	azureResourcesExpectStatus(t, do(http.MethodPut, ns+adrTestAPIVersion, `{"location":"eastus"}`), http.StatusCreated)
	azureResourcesExpectStatus(t, do(http.MethodPut, ns+"/devices/plc"+adrTestAPIVersion, `{"location":"eastus","properties":{"manufacturer":"Contoso"}}`), http.StatusCreated)
	asset := ns + "/assets/pump"
	azureResourcesExpectStatus(t, do(http.MethodPut, asset+adrTestAPIVersion, `{"location":"eastus",`+adrTestEdge+`,"properties":{"deviceRef":{"deviceName":"plc","endpointName":"opc"},
		"managementGroups":[{"name":"ops","actions":[{"name":"restart","targetUri":"ns=1;s=restart"}]}]}}`), http.StatusCreated)
	resp = do(http.MethodPost, asset+"/executeAction"+adrTestAPIVersion, `{"managementGroupName":"ops","managementActionName":"restart"}`)
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodPost, asset+"/executeAction"+adrTestAPIVersion, `{"managementGroupName":"ops","managementActionName":"stop"}`), http.StatusNotFound)
	azureResourcesExpectStatus(t, do(http.MethodPost, ns+"/migrate"+adrTestAPIVersion, `{"scope":"Resources","resourceIds":["`+strings.TrimPrefix(base, "/azure")+`assets/pump"]}`), http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodPut, ns+"/discoveredDevices/found"+adrTestAPIVersion, `{"location":"eastus",`+adrTestEdge+`,"properties":{"discoveryId":"d1","version":1}}`), http.StatusCreated)

	reg := base + "schemaRegistries/reg"
	azureResourcesExpectStatus(t, do(http.MethodPut, reg+adrTestAPIVersion, `{"location":"eastus","properties":{"namespace":"plant","storageAccountContainerUrl":"https://acct.blob.core.windows.net/schemas"}}`), http.StatusCreated)
	azureResourcesExpectStatus(t, do(http.MethodPut, reg+"/schemas/temp"+adrTestAPIVersion, `{"properties":{"format":"JsonSchema/draft-07","schemaType":"MessageSchema"}}`), http.StatusCreated)
	resp = do(http.MethodPut, reg+"/schemas/temp/schemaVersions/1"+adrTestAPIVersion, `{"properties":{"schemaContent":"{\"type\":\"object\"}"}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	if hash, _ := providerContractJSONMap(t, resp)["properties"].(map[string]any)["hash"].(string); hash == "" {
		t.Fatal("expected a schema hash")
	}
	resp = do(http.MethodGet, sub+"/providers/Microsoft.DeviceRegistry/billingContainers"+adrTestAPIVersion, "")
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodGet, "/azure/providers/Microsoft.DeviceRegistry/operations"+adrTestAPIVersion, ""), http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodDelete, ns+adrTestAPIVersion, ""), http.StatusOK)
	azureResourcesExpectErrorCode(t, do(http.MethodGet, asset+adrTestAPIVersion, ""), http.StatusNotFound, "ParentResourceNotFound")
}

func TestAzureDeviceRegistry20260401Validation(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-adr")
	base := sub + "/resourceGroups/rg-adr/providers/Microsoft.DeviceRegistry/"
	azureResourcesExpectStatus(t, do(http.MethodPut, base+"namespaces/n"+adrTestAPIVersion, `{"location":"eastus"}`), http.StatusCreated)
	azureResourcesExpectStatus(t, do(http.MethodPut, base+"schemaRegistries/r"+adrTestAPIVersion, `{"location":"eastus","properties":{"namespace":"n","storageAccountContainerUrl":"u"}}`), http.StatusCreated)
	cases := []struct{ name, path, body string }{
		{"profile edge location", base + "assetEndpointProfiles/p", `{"location":"eastus","properties":{"endpointProfileType":"OpcUa","targetAddress":"x"}}`},
		{"asset profile ref", base + "assets/a", `{"location":"eastus",` + adrTestEdge + `,"properties":{}}`},
		{"discovered asset", base + "namespaces/n/discoveredAssets/a", `{"location":"eastus",` + adrTestEdge + `,"properties":{"deviceRef":{}}}`},
		{"registry storage", base + "schemaRegistries/x", `{"location":"eastus","properties":{"namespace":"n"}}`},
		{"schema format", base + "schemaRegistries/r/schemas/s", `{"properties":{"format":"Avro","schemaType":"MessageSchema"}}`},
		{"schema content", base + "schemaRegistries/r/schemas/s/schemaVersions/1", `{"properties":{}}`},
	}
	for _, tc := range cases {
		resp := do(http.MethodPut, tc.path+adrTestAPIVersion, tc.body)
		t.Run(tc.name, func(t *testing.T) {
			if resp.StatusCode != http.StatusBadRequest && resp.StatusCode != http.StatusNotFound {
				t.Fatalf("expected 400 or 404, got %d", resp.StatusCode)
			}
		})
	}
	azureResourcesExpectStatus(t, do(http.MethodPost, base+"namespaces/n/migrate"+adrTestAPIVersion, `{"scope":"Resources","resourceIds":[]}`), http.StatusBadRequest)
}
