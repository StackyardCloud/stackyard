package server

import (
	"net/http"
	"strings"
	"testing"
)

const azureServiceBusTestAPIVersion = "?api-version=2021-11-01"

func TestAzureServiceBusLifecycle(t *testing.T) {
	t.Parallel()
	ts := newAzureResourcesTestServer(t)
	headers := azureResourcesTestHeaders()
	sub := "/azure/subscriptions/" + azureResourcesTestSubscription
	resp := providerContractRequest(t, ts, http.MethodPut, sub+"/resourcegroups/rg-sb?api-version=2021-04-01", []byte(`{"location":"eastus"}`), headers)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	namespace := sub + "/resourceGroups/rg-sb/providers/Microsoft.ServiceBus/namespaces/Orders-NS"

	resp = providerContractRequest(t, ts, http.MethodPut, namespace+azureServiceBusTestAPIVersion, []byte(`{"location":"eastus","sku":{"name":"Standard"}}`), headers)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	if props, _ := providerContractJSONMap(t, resp)["properties"].(map[string]any); props["serviceBusEndpoint"] != "https://orders-ns.servicebus.windows.net:443/" {
		t.Fatalf("unexpected namespace properties: %#v", props)
	}

	// Real namespaces come with a RootManageSharedAccessKey rule.
	resp = providerContractRequest(t, ts, http.MethodPost, namespace+"/authorizationRules/RootManageSharedAccessKey/listKeys"+azureServiceBusTestAPIVersion, nil, headers)
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	keys := providerContractJSONMap(t, resp)
	conn, _ := keys["primaryConnectionString"].(string)
	if !strings.HasPrefix(conn, "Endpoint=sb://orders-ns.servicebus.windows.net/;SharedAccessKeyName=RootManageSharedAccessKey;SharedAccessKey=") {
		t.Fatalf("unexpected connection string: %q", conn)
	}

	children := []struct{ path, body, typ string }{
		{"/queues/invoices", `{"properties":{"maxDeliveryCount":5}}`, "Microsoft.ServiceBus/namespaces/queues"},
		{"/topics/events", `{}`, "Microsoft.ServiceBus/namespaces/topics"},
		{"/topics/events/subscriptions/audit", `{}`, "Microsoft.ServiceBus/namespaces/topics/subscriptions"},
		{"/authorizationRules/sender", `{"properties":{"rights":["Send"]}}`, "Microsoft.ServiceBus/namespaces/authorizationRules"},
	}
	for _, child := range children {
		resp = providerContractRequest(t, ts, http.MethodPut, namespace+child.path+azureServiceBusTestAPIVersion, []byte(child.body), headers)
		azureResourcesExpectStatus(t, resp, http.StatusCreated)
		if got := providerContractJSONMap(t, resp); got["type"] != child.typ || got["id"] != namespace[len("/azure"):]+child.path {
			t.Fatalf("unexpected child payload for %s: %#v", child.path, got)
		}
	}
	resp = providerContractRequest(t, ts, http.MethodGet, namespace+"/authorizationRules"+azureServiceBusTestAPIVersion, nil, headers)
	if items, _ := providerContractJSONMap(t, resp)["value"].([]any); len(items) != 2 {
		t.Fatalf("expected default and sender rules, got %#v", items)
	}
	resp = providerContractRequest(t, ts, http.MethodGet, namespace+"/topics/events/subscriptions"+azureServiceBusTestAPIVersion, nil, headers)
	if items, _ := providerContractJSONMap(t, resp)["value"].([]any); len(items) != 1 {
		t.Fatalf("expected one subscription, got %#v", items)
	}

	// Deleting a topic removes its subscriptions; deleting the namespace removes the rest.
	resp = providerContractRequest(t, ts, http.MethodDelete, namespace+"/topics/events"+azureServiceBusTestAPIVersion, nil, headers)
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	resp = providerContractRequest(t, ts, http.MethodGet, namespace+"/topics/events/subscriptions/audit"+azureServiceBusTestAPIVersion, nil, headers)
	azureResourcesExpectErrorCode(t, resp, http.StatusNotFound, "ParentResourceNotFound")
	resp = providerContractRequest(t, ts, http.MethodDelete, namespace+azureServiceBusTestAPIVersion, nil, headers)
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	resp = providerContractRequest(t, ts, http.MethodGet, namespace+"/queues/invoices"+azureServiceBusTestAPIVersion, nil, headers)
	azureResourcesExpectErrorCode(t, resp, http.StatusNotFound, "ParentResourceNotFound")
}

func TestAzureServiceBusValidation(t *testing.T) {
	t.Parallel()
	ts := newAzureResourcesTestServer(t)
	headers := azureResourcesTestHeaders()
	sub := "/azure/subscriptions/" + azureResourcesTestSubscription
	resp := providerContractRequest(t, ts, http.MethodPut, sub+"/resourcegroups/rg-sb?api-version=2021-04-01", []byte(`{"location":"eastus"}`), headers)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	namespaces := sub + "/resourceGroups/rg-sb/providers/Microsoft.ServiceBus/namespaces/"

	nameCases := []string{"short", "1starts-with-digit", "ends-with-", "orders-sb", "orders-mgmt"}
	for _, name := range nameCases {
		resp = providerContractRequest(t, ts, http.MethodPut, namespaces+name+azureServiceBusTestAPIVersion, []byte(`{"location":"eastus","sku":{"name":"Basic"}}`), headers)
		t.Run("name "+name, func(t *testing.T) {
			azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, "InvalidResourceName")
		})
	}
	resp = providerContractRequest(t, ts, http.MethodPut, namespaces+"orders-ns"+azureServiceBusTestAPIVersion, []byte(`{"location":"eastus","sku":{"name":"Gold"}}`), headers)
	azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, "InvalidRequestContent")

	resp = providerContractRequest(t, ts, http.MethodPut, namespaces+"orders-ns"+azureServiceBusTestAPIVersion, []byte(`{"location":"eastus","sku":{"name":"Basic"}}`), headers)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	for _, rights := range []string{`[]`, `["Read"]`, `["Manage","Send"]`} {
		resp = providerContractRequest(t, ts, http.MethodPut, namespaces+"orders-ns/authorizationRules/bad"+azureServiceBusTestAPIVersion, []byte(`{"properties":{"rights":`+rights+`}}`), headers)
		t.Run("rights "+rights, func(t *testing.T) {
			azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, "InvalidRequestContent")
		})
	}
	resp = providerContractRequest(t, ts, http.MethodPut, namespaces+"missing-ns/queues/q1"+azureServiceBusTestAPIVersion, []byte(`{}`), headers)
	azureResourcesExpectErrorCode(t, resp, http.StatusNotFound, "ParentResourceNotFound")
}

func TestAzureServiceBusCheckNameAvailability(t *testing.T) {
	t.Parallel()
	ts := newAzureResourcesTestServer(t)
	headers := azureResourcesTestHeaders()
	sub := "/azure/subscriptions/" + azureResourcesTestSubscription
	check := func(body string) map[string]any {
		resp := providerContractRequest(t, ts, http.MethodPost, sub+"/providers/Microsoft.ServiceBus/CheckNameAvailability"+azureServiceBusTestAPIVersion, []byte(body), headers)
		azureResourcesExpectStatus(t, resp, http.StatusOK)
		return providerContractJSONMap(t, resp)
	}
	if got := check(`{"name":"orders-ns"}`); got["nameAvailable"] != true || got["reason"] != "None" {
		t.Fatalf("expected available, got %#v", got)
	}
	resp := providerContractRequest(t, ts, http.MethodPut, sub+"/resourcegroups/rg-sb?api-version=2021-04-01", []byte(`{"location":"eastus"}`), headers)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	resp = providerContractRequest(t, ts, http.MethodPut, sub+"/resourceGroups/rg-sb/providers/Microsoft.ServiceBus/namespaces/orders-ns"+azureServiceBusTestAPIVersion, []byte(`{"location":"eastus","sku":{"name":"Basic"}}`), headers)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	if got := check(`{"name":"ORDERS-NS"}`); got["nameAvailable"] != false || got["reason"] != "NameInUse" {
		t.Fatalf("expected NameInUse, got %#v", got)
	}
	if got := check(`{"name":"x"}`); got["reason"] != "InvalidName" {
		t.Fatalf("expected InvalidName, got %#v", got)
	}
	resp = providerContractRequest(t, ts, http.MethodPost, sub+"/providers/Microsoft.ServiceBus/CheckNameAvailability"+azureServiceBusTestAPIVersion, []byte(`{}`), headers)
	azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, "InvalidRequestContent")
}
