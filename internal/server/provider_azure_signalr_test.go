package server

import (
	"net/http"
	"strings"
	"testing"
)

const azureSignalRTestAPIVersion = "?api-version=2024-03-01"

func TestAzureSignalRLifecycle(t *testing.T) {
	t.Parallel()
	ts := newAzureResourcesTestServer(t)
	headers := azureResourcesTestHeaders()
	sub := "/azure/subscriptions/" + azureResourcesTestSubscription
	azureResourcesExpectStatus(t, providerContractRequest(t, ts, http.MethodPut, sub+"/resourcegroups/rg-signalr?api-version=2021-04-01", []byte(`{"location":"eastus"}`), headers), http.StatusCreated)
	// SDKs use the "signalR" path segment; responses use the canonical type.
	service := sub + "/resourceGroups/rg-signalr/providers/Microsoft.SignalRService/signalR/Chat-Hub"

	resp := providerContractRequest(t, ts, http.MethodPut, service+azureSignalRTestAPIVersion, []byte(`{"location":"eastus","sku":{"name":"Premium_P1","capacity":1},"kind":"SignalR","properties":{"hostName":"spoofed"}}`), headers)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	created := providerContractJSONMap(t, resp)
	props, _ := created["properties"].(map[string]any)
	if created["type"] != "Microsoft.SignalRService/SignalR" || props["hostName"] != "chat-hub.service.signalr.net" || props["publicPort"] != float64(443) {
		t.Fatalf("unexpected service payload: %#v", created)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, service+"/listKeys"+azureSignalRTestAPIVersion, nil, headers)
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	keys := providerContractJSONMap(t, resp)
	if conn, _ := keys["primaryConnectionString"].(string); !strings.HasPrefix(conn, "Endpoint=https://chat-hub.service.signalr.net;AccessKey=") || !strings.HasSuffix(conn, ";Version=1.0;") {
		t.Fatalf("unexpected connection string: %q", conn)
	}

	replica := service + "/replicas/chat-hub-westus"
	azureResourcesExpectErrorCode(t, providerContractRequest(t, ts, http.MethodPut, replica+azureSignalRTestAPIVersion, []byte(`{"sku":{"name":"Premium_P1"}}`), headers), http.StatusBadRequest, "LocationRequired")
	resp = providerContractRequest(t, ts, http.MethodPut, replica+azureSignalRTestAPIVersion, []byte(`{"location":"westus","sku":{"name":"Premium_P1","capacity":1}}`), headers)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	if got := providerContractJSONMap(t, resp); got["type"] != "Microsoft.SignalRService/SignalR/replicas" || got["location"] != "westus" {
		t.Fatalf("unexpected replica: %#v", got)
	}

	azureResourcesExpectStatus(t, providerContractRequest(t, ts, http.MethodDelete, service+azureSignalRTestAPIVersion, nil, headers), http.StatusOK)
	azureResourcesExpectErrorCode(t, providerContractRequest(t, ts, http.MethodGet, replica+azureSignalRTestAPIVersion, nil, headers), http.StatusNotFound, "ParentResourceNotFound")
}

func TestAzureSignalRValidation(t *testing.T) {
	t.Parallel()
	ts := newAzureResourcesTestServer(t)
	headers := azureResourcesTestHeaders()
	sub := "/azure/subscriptions/" + azureResourcesTestSubscription
	azureResourcesExpectStatus(t, providerContractRequest(t, ts, http.MethodPut, sub+"/resourcegroups/rg-signalr?api-version=2021-04-01", []byte(`{"location":"eastus"}`), headers), http.StatusCreated)
	services := sub + "/resourceGroups/rg-signalr/providers/Microsoft.SignalRService/SignalR/"
	cases := []struct{ name, path, body, code string }{
		{"starts with digit", services + "1chat", `{"location":"eastus","sku":{"name":"Free_F1"}}`, "InvalidResourceName"},
		{"too short", services + "ab", `{"location":"eastus","sku":{"name":"Free_F1"}}`, "InvalidResourceName"},
		{"missing sku", services + "chat-hub", `{"location":"eastus"}`, "InvalidRequestContent"},
		{"web pubsub sku", services + "chat-hub", `{"location":"eastus","sku":{"name":"Standard_S2"}}`, "InvalidRequestContent"},
		{"bad kind", services + "chat-hub", `{"location":"eastus","sku":{"name":"Free_F1"},"kind":"Socket"}`, "InvalidRequestContent"},
	}
	for _, tc := range cases {
		resp := providerContractRequest(t, ts, http.MethodPut, tc.path+azureSignalRTestAPIVersion, []byte(tc.body), headers)
		t.Run(tc.name, func(t *testing.T) { azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, tc.code) })
	}
	azureResourcesExpectErrorCode(t, providerContractRequest(t, ts, http.MethodPost, services+"missing/listKeys"+azureSignalRTestAPIVersion, nil, headers), http.StatusNotFound, "ResourceNotFound")
}

func TestAzureSignalRCheckNameAvailabilitySharesWebPubSubRoute(t *testing.T) {
	t.Parallel()
	ts := newAzureResourcesTestServer(t)
	headers := azureResourcesTestHeaders()
	sub := "/azure/subscriptions/" + azureResourcesTestSubscription
	checkPath := sub + "/providers/Microsoft.SignalRService/locations/eastus/checkNameAvailability" + azureSignalRTestAPIVersion
	check := func(body string) map[string]any {
		resp := providerContractRequest(t, ts, http.MethodPost, checkPath, []byte(body), headers)
		azureResourcesExpectStatus(t, resp, http.StatusOK)
		return providerContractJSONMap(t, resp)
	}
	signalR := func(name string) string { return `{"name":"` + name + `","type":"Microsoft.SignalRService/SignalR"}` }

	if got := check(signalR("chat-hub")); got["nameAvailable"] != true {
		t.Fatalf("expected available, got %#v", got)
	}
	azureResourcesExpectStatus(t, providerContractRequest(t, ts, http.MethodPut, sub+"/resourcegroups/rg-signalr?api-version=2021-04-01", []byte(`{"location":"eastus"}`), headers), http.StatusCreated)
	azureResourcesExpectStatus(t, providerContractRequest(t, ts, http.MethodPut, sub+"/resourceGroups/rg-signalr/providers/Microsoft.SignalRService/SignalR/chat-hub"+azureSignalRTestAPIVersion, []byte(`{"location":"westus","sku":{"name":"Free_F1"}}`), headers), http.StatusCreated)
	// Host names are global, so a service in another region still collides.
	if got := check(signalR("CHAT-HUB")); got["nameAvailable"] != false || got["reason"] != "AlreadyExists" {
		t.Fatalf("expected AlreadyExists, got %#v", got)
	}
	if got := check(signalR("9chat")); got["reason"] != "Invalid" {
		t.Fatalf("expected Invalid, got %#v", got)
	}
	// Web PubSub names on the same route are unaffected by SignalR services.
	if got := check(`{"name":"chat-hub","type":"Microsoft.SignalRService/WebPubSub"}`); got["nameAvailable"] != true {
		t.Fatalf("expected Web PubSub name to stay available, got %#v", got)
	}
}
