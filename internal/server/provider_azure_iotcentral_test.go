package server

import (
	"net/http"
	"testing"
)

const azureIoTCentralTestAPIVersion = "?api-version=2021-06-01"

func TestAzureIoTCentralLifecycle(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-iotc")
	app := sub + "/resourceGroups/rg-iotc/providers/Microsoft.IoTCentral/iotApps/plant"

	resp := do(http.MethodPut, app+azureIoTCentralTestAPIVersion, `{"location":"eastus","sku":{"name":"ST2"},"identity":{"type":"SystemAssigned"},"properties":{"subdomain":"plant-floor","state":"suspended"}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	props := providerContractJSONMap(t, resp)["properties"].(map[string]any)
	if props["state"] != "created" || props["displayName"] != "plant" || props["template"] != "iotc-pnp-preview@1.0.0" || len(props["applicationId"].(string)) != 36 {
		t.Fatalf("unexpected app properties: %#v", props)
	}

	check := func(action, name string) map[string]any {
		resp := do(http.MethodPost, sub+"/providers/Microsoft.IoTCentral/"+action+azureIoTCentralTestAPIVersion, `{"name":"`+name+`","type":"IoTApps"}`)
		azureResourcesExpectStatus(t, resp, http.StatusOK)
		return providerContractJSONMap(t, resp)
	}
	for _, tc := range []struct {
		action, name string
		available    bool
		reason       any
	}{
		{"checkNameAvailability", "plant", false, "AlreadyExists"},
		{"checkNameAvailability", "other", true, nil},
		{"checkNameAvailability", "Bad_Name", false, "Invalid"},
		{"checkSubdomainAvailability", "plant-floor", false, "AlreadyExists"},
		{"checkSubdomainAvailability", "plant", true, nil},
	} {
		if got := check(tc.action, tc.name); got["nameAvailable"] != tc.available || got["reason"] != tc.reason {
			t.Fatalf("%s %s: unexpected %#v", tc.action, tc.name, got)
		}
	}

	azureResourcesExpectStatus(t, do(http.MethodDelete, app+azureIoTCentralTestAPIVersion, ""), http.StatusOK)
}

func TestAzureIoTCentralValidation(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-iotc")
	apps := sub + "/resourceGroups/rg-iotc/providers/Microsoft.IoTCentral/iotApps/"
	cases := []struct{ name, path, body, code string }{
		{"uppercase name", apps + "Plant", `{"location":"eastus","sku":{"name":"ST1"},"properties":{"subdomain":"plant"}}`, "InvalidResourceName"},
		{"bad sku", apps + "plant", `{"location":"eastus","sku":{"name":"F1"},"properties":{"subdomain":"plant"}}`, "InvalidRequestContent"},
		{"no subdomain", apps + "plant", `{"location":"eastus","sku":{"name":"ST1"},"properties":{}}`, "InvalidRequestContent"},
		{"bad identity", apps + "plant", `{"location":"eastus","sku":{"name":"ST1"},"identity":{"type":"UserAssigned"},"properties":{"subdomain":"plant"}}`, "InvalidRequestContent"},
	}
	for _, tc := range cases {
		resp := do(http.MethodPut, tc.path+azureIoTCentralTestAPIVersion, tc.body)
		t.Run(tc.name, func(t *testing.T) { azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, tc.code) })
	}
}
