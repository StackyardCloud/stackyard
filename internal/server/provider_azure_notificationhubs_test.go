package server

import (
	"net/http"
	"strings"
	"testing"
)

const azureNotificationHubsTestAPIVersion = "?api-version=2023-09-01"

func newAzureNotificationHubsTestServer(t *testing.T) (string, func(method, path, body string) *http.Response) {
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
	azureResourcesExpectStatus(t, do(http.MethodPut, sub+"/resourcegroups/rg-nh?api-version=2021-04-01", `{"location":"eastus"}`), http.StatusCreated)
	return sub, do
}

func TestAzureNotificationHubsLifecycle(t *testing.T) {
	t.Parallel()
	sub, do := newAzureNotificationHubsTestServer(t)
	namespace := sub + "/resourceGroups/rg-nh/providers/Microsoft.NotificationHubs/namespaces/push-ns"

	resp := do(http.MethodPut, namespace+azureNotificationHubsTestAPIVersion, `{"location":"eastus","sku":{"name":"Standard"}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	if props, _ := providerContractJSONMap(t, resp)["properties"].(map[string]any); props["namespaceType"] != "NotificationHub" || props["serviceBusEndpoint"] != "https://push-ns.servicebus.windows.net:443/" {
		t.Fatalf("unexpected namespace properties: %#v", props)
	}

	hub := namespace + "/notificationHubs/orders"
	azureResourcesExpectErrorCode(t, do(http.MethodPut, hub+azureNotificationHubsTestAPIVersion, `{}`), http.StatusBadRequest, "LocationRequired")
	resp = do(http.MethodPut, hub+azureNotificationHubsTestAPIVersion, `{"location":"eastus","properties":{"name":"orders"}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	if got := providerContractJSONMap(t, resp); got["type"] != "Microsoft.NotificationHubs/namespaces/notificationHubs" || got["location"] != "eastus" {
		t.Fatalf("unexpected hub: %#v", got)
	}

	azureResourcesExpectStatus(t, do(http.MethodPut, hub+"/authorizationRules/DefaultListenSharedAccessSignature"+azureNotificationHubsTestAPIVersion, `{"properties":{"rights":["Listen"]}}`), http.StatusCreated)
	resp = do(http.MethodPost, hub+"/authorizationRules/DefaultListenSharedAccessSignature/listKeys"+azureNotificationHubsTestAPIVersion, "")
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if conn, _ := providerContractJSONMap(t, resp)["primaryConnectionString"].(string); strings.Contains(conn, "EntityPath") ||
		!strings.HasPrefix(conn, "Endpoint=sb://push-ns.servicebus.windows.net/;SharedAccessKeyName=DefaultListenSharedAccessSignature;") {
		t.Fatalf("unexpected hub connection string: %q", conn)
	}

	resp = do(http.MethodGet, namespace+"/notificationHubs"+azureNotificationHubsTestAPIVersion, "")
	if items, _ := providerContractJSONMap(t, resp)["value"].([]any); len(items) != 1 {
		t.Fatalf("expected one hub, got %#v", items)
	}
	azureResourcesExpectStatus(t, do(http.MethodDelete, namespace+azureNotificationHubsTestAPIVersion, ""), http.StatusOK)
	azureResourcesExpectErrorCode(t, do(http.MethodGet, hub+azureNotificationHubsTestAPIVersion, ""), http.StatusNotFound, "ParentResourceNotFound")
}

func TestAzureNotificationHubsValidation(t *testing.T) {
	t.Parallel()
	sub, do := newAzureNotificationHubsTestServer(t)
	namespaces := sub + "/resourceGroups/rg-nh/providers/Microsoft.NotificationHubs/namespaces/"
	cases := []struct{ name, path, body, code string }{
		{"name ends with -sb", namespaces + "push-ns-sb", `{"location":"eastus","sku":{"name":"Free"}}`, "InvalidResourceName"},
		{"premium sku", namespaces + "push-ns", `{"location":"eastus","sku":{"name":"Premium"}}`, "InvalidRequestContent"},
		{"bad namespace type", namespaces + "push-ns", `{"location":"eastus","sku":{"name":"Free"},"properties":{"namespaceType":"Queue"}}`, "InvalidRequestContent"},
		{"hub without namespace", namespaces + "missing-ns/notificationHubs/orders", `{"location":"eastus"}`, "ParentResourceNotFound"},
	}
	for _, tc := range cases {
		resp := do(http.MethodPut, tc.path+azureNotificationHubsTestAPIVersion, tc.body)
		status := http.StatusBadRequest
		if tc.code == "ParentResourceNotFound" {
			status = http.StatusNotFound
		}
		t.Run(tc.name, func(t *testing.T) { azureResourcesExpectErrorCode(t, resp, status, tc.code) })
	}
}

func TestAzureNotificationHubsCheckNamespaceAvailability(t *testing.T) {
	t.Parallel()
	sub, do := newAzureNotificationHubsTestServer(t)
	check := func(name string) map[string]any {
		resp := do(http.MethodPost, sub+"/providers/Microsoft.NotificationHubs/checkNamespaceAvailability"+azureNotificationHubsTestAPIVersion, `{"name":"`+name+`"}`)
		azureResourcesExpectStatus(t, resp, http.StatusOK)
		return providerContractJSONMap(t, resp)
	}
	if got := check("push-ns"); got["isAvailiable"] != true {
		t.Fatalf("expected available, got %#v", got)
	}
	azureResourcesExpectStatus(t, do(http.MethodPut, sub+"/resourceGroups/rg-nh/providers/Microsoft.NotificationHubs/namespaces/push-ns"+azureNotificationHubsTestAPIVersion, `{"location":"eastus","sku":{"name":"Free"}}`), http.StatusCreated)
	if got := check("PUSH-NS"); got["isAvailiable"] != false {
		t.Fatalf("expected taken, got %#v", got)
	}
	if got := check("bad"); got["isAvailiable"] != false {
		t.Fatalf("expected invalid name to be unavailable, got %#v", got)
	}
	azureResourcesExpectErrorCode(t, do(http.MethodPost, sub+"/providers/Microsoft.NotificationHubs/checkNamespaceAvailability"+azureNotificationHubsTestAPIVersion, `{}`), http.StatusBadRequest, "InvalidRequestContent")
}
