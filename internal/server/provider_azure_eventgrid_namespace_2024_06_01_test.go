package server

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestAzureEventGridNamespacePullDelivery(t *testing.T) {
	t.Parallel()
	c := newAzureARMTestClientNoAuth(t, "rg-egnsdp")
	ns := c.sub() + "/resourceGroups/rg-egnsdp/providers/Microsoft.EventGrid/namespaces/fleet"
	c.expect(http.MethodPut, ns+egcpTestAPI, `{"location":"eastus","sku":{"name":"Standard"}}`, http.StatusCreated)
	c.expect(http.MethodPut, ns+"/topics/telemetry"+egcpTestAPI, `{}`, http.StatusCreated)
	c.expect(http.MethodPut, ns+"/topics/telemetry/eventSubscriptions/all"+egcpTestAPI, `{"properties":{"deliveryConfiguration":{"deliveryMode":"Queue","queue":{"maxDeliveryCount":2}}}}`, http.StatusCreated)
	c.expect(http.MethodPut, ns+"/topics/telemetry/eventSubscriptions/alerts"+egcpTestAPI,
		`{"properties":{"deliveryConfiguration":{"deliveryMode":"Queue"},"filtersConfiguration":{"includedEventTypes":["Vehicle.Alert"]}}}`, http.StatusCreated)
	keys := c.expect(http.MethodPost, ns+"/listKeys"+egcpTestAPI, "", http.StatusOK)

	headers := map[string]string{"Authorization": "SharedAccessKey " + keys["key1"].(string), "Content-Type": "application/cloudevents+json"}
	base := "/azure/eventgrid-namespace/2024-06-01/fleet.eastus-1.eventgrid.azure.net/topics/telemetry"
	call := func(path, body string, status int) map[string]any {
		t.Helper()
		resp := providerContractRequest(t, c.ts, http.MethodPost, base+path, []byte(body), headers)
		azureResourcesExpectStatus(t, resp, status)
		return providerContractJSONMap(t, resp)
	}
	call(":publish?api-version=2024-06-01", `{"id":"1","source":"/cars/1","type":"Vehicle.Moved","specversion":"1.0","data":{"km":3}}`, http.StatusOK)
	headers["Content-Type"] = "application/cloudevents-batch+json"
	call(":publish?api-version=2024-06-01", `[{"id":"2","source":"/cars/1","type":"Vehicle.Alert","specversion":"1.0"},{"id":"3","source":"/cars/2","type":"Vehicle.Moved","specversion":"1.0"}]`, http.StatusOK)
	headers["Content-Type"] = "application/json"

	received := azureTestList(t, call("/eventsubscriptions/all:receive?api-version=2024-06-01&maxEvents=2", "", http.StatusOK))
	if len(received) != 2 {
		t.Fatalf("expected two events, got %#v", received)
	}
	token := func(item any) string {
		return item.(map[string]any)["brokerProperties"].(map[string]any)["lockToken"].(string)
	}
	first, second := token(received[0]), token(received[1])
	lockTokens := func(tokens ...string) string {
		raw, _ := json.Marshal(map[string]any{"lockTokens": tokens})
		return string(raw)
	}
	acked := call("/eventsubscriptions/all:acknowledge?api-version=2024-06-01", lockTokens(first, "bogus"), http.StatusOK)
	if len(acked["succeededLockTokens"].([]any)) != 1 || len(acked["failedLockTokens"].([]any)) != 1 {
		t.Fatalf("unexpected acknowledge result: %#v", acked)
	}
	renewed := call("/eventsubscriptions/all:renewLock?api-version=2024-06-01", lockTokens(second), http.StatusOK)
	if len(renewed["succeededLockTokens"].([]any)) != 1 {
		t.Fatalf("unexpected renewLock result: %#v", renewed)
	}
	call("/eventsubscriptions/all:release?api-version=2024-06-01", lockTokens(second), http.StatusOK)

	// The released event and the third one are available again; the
	// acknowledged one is gone.
	again := azureTestList(t, call("/eventsubscriptions/all:receive?api-version=2024-06-01&maxEvents=10", "", http.StatusOK))
	if len(again) != 2 {
		t.Fatalf("expected the released and the remaining event, got %#v", again)
	}
	for _, item := range again {
		if item.(map[string]any)["event"].(map[string]any)["id"] == "1" {
			t.Fatalf("the acknowledged event was redelivered: %#v", again)
		}
	}
	rejected := call("/eventsubscriptions/all:reject?api-version=2024-06-01", lockTokens(token(again[0]), token(again[1])), http.StatusOK)
	if len(rejected["succeededLockTokens"].([]any)) != 2 {
		t.Fatalf("unexpected reject result: %#v", rejected)
	}
	if left := azureTestList(t, call("/eventsubscriptions/all:receive?api-version=2024-06-01&maxEvents=10", "", http.StatusOK)); len(left) != 0 {
		t.Fatalf("expected an empty queue, got %#v", left)
	}

	alerts := azureTestList(t, call("/eventsubscriptions/alerts:receive?api-version=2024-06-01&maxEvents=10", "", http.StatusOK))
	if len(alerts) != 1 || alerts[0].(map[string]any)["event"].(map[string]any)["type"] != "Vehicle.Alert" {
		t.Fatalf("the alerts subscription filters by event type, got %#v", alerts)
	}

	call("/eventsubscriptions/all:receive?api-version=2024-06-01&maxEvents=101", "", http.StatusBadRequest)
	call("/eventsubscriptions/all:release?api-version=2024-06-01&releaseDelayInSeconds=5", lockTokens("x"), http.StatusBadRequest)
	call("/eventsubscriptions/all:acknowledge?api-version=2024-06-01", `{"lockTokens":[]}`, http.StatusBadRequest)
	call("/eventsubscriptions/missing:receive?api-version=2024-06-01", "", http.StatusNotFound)
	call(":publish?api-version=2024-06-01", `{"id":"9","source":"/x","type":"t","specversion":"0.3"}`, http.StatusBadRequest)
	headers["Authorization"] = "SharedAccessKey wrong"
	call(":publish?api-version=2024-06-01", `{"id":"9","source":"/x","type":"t","specversion":"1.0"}`, http.StatusUnauthorized)
}
