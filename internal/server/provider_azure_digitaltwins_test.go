package server

import (
	"net/http"
	"testing"
)

const azureDigitalTwinsTestAPIVersion = "?api-version=2023-01-31"

func TestAzureDigitalTwinsLifecycle(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-twins")
	instance := sub + "/resourceGroups/rg-twins/providers/Microsoft.DigitalTwins/digitalTwinsInstances/plant-twins"

	resp := do(http.MethodPut, instance+azureDigitalTwinsTestAPIVersion, `{"location":"eastus","properties":{"publicNetworkAccess":"Enabled"}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	props, _ := providerContractJSONMap(t, resp)["properties"].(map[string]any)
	if props["hostName"] != "plant-twins.api.eastus.digitaltwins.azure.net" || props["createdTime"] == nil || props["publicNetworkAccess"] != "Enabled" {
		t.Fatalf("unexpected instance properties: %#v", props)
	}

	endpoint := instance + "/endpoints/telemetry"
	resp = do(http.MethodPut, endpoint+azureDigitalTwinsTestAPIVersion, `{"properties":{"endpointType":"EventHub","authenticationType":"KeyBased","connectionStringPrimaryKey":"Endpoint=sb://hub.servicebus.windows.net/;SharedAccessKey=secret","connectionStringSecondaryKey":"other"}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	if got := providerContractJSONMap(t, resp)["properties"].(map[string]any); got["endpointType"] != "EventHub" || got["connectionStringPrimaryKey"] != nil || got["connectionStringSecondaryKey"] != nil {
		t.Fatalf("endpoint secrets must not be returned: %#v", got)
	}
	azureResourcesExpectStatus(t, do(http.MethodPut, instance+"/endpoints/events"+azureDigitalTwinsTestAPIVersion, `{"properties":{"endpointType":"ServiceBus","authenticationType":"IdentityBased","endpointUri":"sb://bus.servicebus.windows.net/","entityPath":"twins"}}`), http.StatusCreated)
	resp = do(http.MethodGet, instance+"/endpoints"+azureDigitalTwinsTestAPIVersion, "")
	if items, _ := providerContractJSONMap(t, resp)["value"].([]any); len(items) != 2 {
		t.Fatalf("expected two endpoints, got %#v", items)
	}

	azureResourcesExpectStatus(t, do(http.MethodDelete, instance+azureDigitalTwinsTestAPIVersion, ""), http.StatusOK)
	azureResourcesExpectErrorCode(t, do(http.MethodGet, endpoint+azureDigitalTwinsTestAPIVersion, ""), http.StatusNotFound, "ParentResourceNotFound")
}

func TestAzureDigitalTwinsValidation(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-twins")
	instances := sub + "/resourceGroups/rg-twins/providers/Microsoft.DigitalTwins/digitalTwinsInstances/"
	azureResourcesExpectErrorCode(t, do(http.MethodPut, instances+"ab"+azureDigitalTwinsTestAPIVersion, `{"location":"eastus"}`), http.StatusBadRequest, "InvalidResourceName")
	azureResourcesExpectStatus(t, do(http.MethodPut, instances+"plant-twins"+azureDigitalTwinsTestAPIVersion, `{"location":"eastus"}`), http.StatusCreated)

	endpoints := instances + "plant-twins/endpoints/"
	cases := []struct{ name, body string }{
		{"missing type", `{"properties":{}}`},
		{"unknown type", `{"properties":{"endpointType":"Kafka"}}`},
		{"event hub without key", `{"properties":{"endpointType":"EventHub"}}`},
		{"service bus without key", `{"properties":{"endpointType":"ServiceBus","authenticationType":"KeyBased"}}`},
		{"event grid without topic", `{"properties":{"endpointType":"EventGrid","accessKey1":"k"}}`},
		{"identity event grid", `{"properties":{"endpointType":"EventGrid","authenticationType":"IdentityBased","endpointUri":"x","entityPath":"y"}}`},
		{"identity without uri", `{"properties":{"endpointType":"EventHub","authenticationType":"IdentityBased"}}`},
		{"bad auth type", `{"properties":{"endpointType":"EventHub","authenticationType":"Certificate"}}`},
	}
	for _, tc := range cases {
		resp := do(http.MethodPut, endpoints+"bad"+azureDigitalTwinsTestAPIVersion, tc.body)
		t.Run(tc.name, func(t *testing.T) {
			azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, "InvalidRequestContent")
		})
	}

	check := func(name string) map[string]any {
		resp := do(http.MethodPost, sub+"/providers/Microsoft.DigitalTwins/locations/westus/checkNameAvailability"+azureDigitalTwinsTestAPIVersion, `{"name":"`+name+`","type":"Microsoft.DigitalTwins/digitalTwinsInstances"}`)
		azureResourcesExpectStatus(t, resp, http.StatusOK)
		return providerContractJSONMap(t, resp)
	}
	// Host names are global, so an instance in another region still collides.
	if got := check("plant-twins"); got["reason"] != "AlreadyExists" {
		t.Fatalf("expected AlreadyExists, got %#v", got)
	}
	if got := check("other-twins"); got["nameAvailable"] != true {
		t.Fatalf("expected available, got %#v", got)
	}
	if got := check("-bad"); got["reason"] != "Invalid" {
		t.Fatalf("expected Invalid, got %#v", got)
	}
}
