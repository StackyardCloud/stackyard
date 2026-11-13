package server

import (
	"net/http"
	"strings"
	"testing"
)

const azureRelayTestAPIVersion = "?api-version=2021-11-01"

func newAzureRelayTestNamespace(t *testing.T) (string, func(method, path, body string) *http.Response) {
	t.Helper()
	ts := newAzureResourcesTestServer(t)
	headers := azureResourcesTestHeaders()
	sub := "/azure/subscriptions/" + azureResourcesTestSubscription
	do := func(method, path, body string) *http.Response {
		var payload []byte
		if body != "" {
			payload = []byte(body)
		}
		return providerContractRequest(t, ts, method, path, payload, headers)
	}
	azureResourcesExpectStatus(t, do(http.MethodPut, sub+"/resourcegroups/rg-relay?api-version=2021-04-01", `{"location":"eastus"}`), http.StatusCreated)
	return sub, do
}

func TestAzureRelayLifecycle(t *testing.T) {
	t.Parallel()
	sub, do := newAzureRelayTestNamespace(t)
	namespace := sub + "/resourceGroups/rg-relay/providers/Microsoft.Relay/namespaces/Bridge-NS"

	resp := do(http.MethodPut, namespace+azureRelayTestAPIVersion, `{"location":"eastus","sku":{"name":"Standard"}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	if props, _ := providerContractJSONMap(t, resp)["properties"].(map[string]any); props["serviceBusEndpoint"] != "https://bridge-ns.servicebus.windows.net:443/" {
		t.Fatalf("unexpected namespace properties: %#v", props)
	}

	resp = do(http.MethodPost, namespace+"/authorizationRules/RootManageSharedAccessKey/listKeys"+azureRelayTestAPIVersion, "")
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if conn, _ := providerContractJSONMap(t, resp)["primaryConnectionString"].(string); strings.Contains(conn, "EntityPath") ||
		!strings.HasPrefix(conn, "Endpoint=sb://bridge-ns.servicebus.windows.net/;SharedAccessKeyName=RootManageSharedAccessKey;") {
		t.Fatalf("unexpected namespace connection string: %q", conn)
	}

	azureResourcesExpectStatus(t, do(http.MethodPut, namespace+"/hybridConnections/onprem"+azureRelayTestAPIVersion, `{"properties":{"requiresClientAuthorization":true}}`), http.StatusCreated)
	azureResourcesExpectStatus(t, do(http.MethodPut, namespace+"/hybridConnections/onprem/authorizationRules/listener"+azureRelayTestAPIVersion, `{"properties":{"rights":["Listen"]}}`), http.StatusCreated)
	// Entity-level rules carry the entity in the connection string.
	resp = do(http.MethodPost, namespace+"/hybridConnections/onprem/authorizationRules/listener/listKeys"+azureRelayTestAPIVersion, "")
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if conn, _ := providerContractJSONMap(t, resp)["primaryConnectionString"].(string); !strings.HasSuffix(conn, ";EntityPath=onprem") {
		t.Fatalf("expected EntityPath in hybrid connection string, got %q", conn)
	}

	resp = do(http.MethodPut, namespace+"/wcfRelays/legacy"+azureRelayTestAPIVersion, `{"properties":{"relayType":"NetTcp"}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	if got := providerContractJSONMap(t, resp); got["type"] != "Microsoft.Relay/namespaces/wcfRelays" {
		t.Fatalf("unexpected WCF relay: %#v", got)
	}

	azureResourcesExpectStatus(t, do(http.MethodDelete, namespace+azureRelayTestAPIVersion, ""), http.StatusOK)
	azureResourcesExpectErrorCode(t, do(http.MethodGet, namespace+"/hybridConnections/onprem/authorizationRules/listener"+azureRelayTestAPIVersion, ""), http.StatusNotFound, "ParentResourceNotFound")
}

func TestAzureRelayValidation(t *testing.T) {
	t.Parallel()
	sub, do := newAzureRelayTestNamespace(t)
	namespaces := sub + "/resourceGroups/rg-relay/providers/Microsoft.Relay/namespaces/"

	cases := []struct{ name, method, path, body, code string }{
		{"name too short", http.MethodPut, namespaces + "short", `{"location":"eastus","sku":{"name":"Standard"}}`, "InvalidResourceName"},
		{"basic sku", http.MethodPut, namespaces + "bridge-ns", `{"location":"eastus","sku":{"name":"Basic"}}`, "InvalidRequestContent"},
	}
	for _, tc := range cases {
		resp := do(tc.method, tc.path+azureRelayTestAPIVersion, tc.body)
		t.Run(tc.name, func(t *testing.T) { azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, tc.code) })
	}

	azureResourcesExpectStatus(t, do(http.MethodPut, namespaces+"bridge-ns"+azureRelayTestAPIVersion, `{"location":"eastus","sku":{"name":"Standard"}}`), http.StatusCreated)
	childCases := []struct{ name, path, body string }{
		{"wcf relay without type", "/wcfRelays/legacy", `{"properties":{}}`},
		{"wcf relay bad type", "/wcfRelays/legacy", `{"properties":{"relayType":"Udp"}}`},
		{"rule without rights", "/authorizationRules/empty", `{"properties":{"rights":[]}}`},
		{"manage without send", "/authorizationRules/admin", `{"properties":{"rights":["Manage","Listen"]}}`},
	}
	for _, tc := range childCases {
		resp := do(http.MethodPut, namespaces+"bridge-ns"+tc.path+azureRelayTestAPIVersion, tc.body)
		t.Run(tc.name, func(t *testing.T) {
			azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, "InvalidRequestContent")
		})
	}
	azureResourcesExpectErrorCode(t, do(http.MethodPut, namespaces+"bridge-ns/hybridConnections/missing/authorizationRules/r1"+azureRelayTestAPIVersion, `{"properties":{"rights":["Send"]}}`), http.StatusNotFound, "ParentResourceNotFound")
}

func TestAzureRelayCheckNameAvailabilitySharesServiceBusNames(t *testing.T) {
	t.Parallel()
	sub, do := newAzureRelayTestNamespace(t)
	check := func(name string) map[string]any {
		resp := do(http.MethodPost, sub+"/providers/Microsoft.Relay/checkNameAvailability"+azureRelayTestAPIVersion, `{"name":"`+name+`"}`)
		azureResourcesExpectStatus(t, resp, http.StatusOK)
		return providerContractJSONMap(t, resp)
	}
	if got := check("shared-ns"); got["nameAvailable"] != true {
		t.Fatalf("expected available, got %#v", got)
	}
	// Relay and Service Bus namespaces share servicebus.windows.net.
	azureResourcesExpectStatus(t, do(http.MethodPut, sub+"/resourceGroups/rg-relay/providers/Microsoft.ServiceBus/namespaces/shared-ns?api-version=2021-11-01", `{"location":"eastus","sku":{"name":"Basic"}}`), http.StatusCreated)
	if got := check("Shared-NS"); got["nameAvailable"] != false || got["reason"] != "NameInUse" {
		t.Fatalf("expected NameInUse, got %#v", got)
	}
	if got := check("bad"); got["reason"] != "InvalidName" {
		t.Fatalf("expected InvalidName, got %#v", got)
	}
	azureResourcesExpectErrorCode(t, do(http.MethodPost, sub+"/providers/Microsoft.Relay/checkNameAvailability"+azureRelayTestAPIVersion, `{}`), http.StatusBadRequest, "InvalidRequestContent")
}
