package server

import (
	"net/http"
	"strings"
	"testing"
)

func TestAzureEventGridDataPlanePublish(t *testing.T) {
	t.Parallel()
	c := newAzureARMTestClientNoAuth(t, "rg-egdp")
	topic := c.sub() + "/resourceGroups/rg-egdp/providers/Microsoft.EventGrid/topics/orders"
	c.expect(http.MethodPut, topic+egcpTestAPI, `{"location":"westus2"}`, http.StatusCreated)
	keys := c.expect(http.MethodPost, topic+"/listKeys"+egcpTestAPI, "", http.StatusOK)

	publish := func(overload, body string, headers map[string]string) *http.Response {
		t.Helper()
		merged := map[string]string{"Content-Type": "application/json"}
		for k, v := range headers {
			merged[k] = v
		}
		return providerContractRequest(t, c.ts, http.MethodPost, "/azure/eventgrid-dataplane/2018-01-01/orders.westus2-1.eventgrid.azure.net/api/events?overload="+overload+"&api-version=2018-01-01", []byte(body), merged)
	}
	withKey := map[string]string{"aeg-sas-key": keys["key1"].(string)}

	event := `[{"id":"1","subject":"orders/1","eventType":"Order.Created","eventTime":"2026-11-23T10:00:00Z","dataVersion":"1.0","data":{"total":5}}]`
	azureResourcesExpectStatus(t, publish("EventGridEvent", event, withKey), http.StatusOK)
	azureResourcesExpectStatus(t, publish("cloudEvent", `[{"id":"2","source":"/orders","type":"Order.Shipped","specversion":"1.0"}]`, map[string]string{"aeg-sas-token": "r=orders&e=1&s=sig"}), http.StatusOK)
	azureResourcesExpectStatus(t, publish("customEvent", `[{"anything":"goes"}]`, withKey), http.StatusOK)

	azureResourcesExpectErrorCode(t, publish("EventGridEvent", event, map[string]string{"aeg-sas-key": "wrong"}), http.StatusUnauthorized, "Unauthorized")
	azureResourcesExpectErrorCode(t, publish("EventGridEvent", event, nil), http.StatusUnauthorized, "Unauthorized")
	resp := publish("EventGridEvent", `[{"id":"1"}]`, withKey)
	azureResourcesExpectStatus(t, resp, http.StatusBadRequest)
	if msg, _ := providerContractJSONMap(t, resp)["error"].(map[string]any)["message"].(string); !strings.Contains(msg, "Event 0: subject is required.") {
		t.Fatalf("the error must name the event and field, got %q", msg)
	}
	azureResourcesExpectErrorCode(t, publish("cloudEvent", `[{"id":"2","source":"/o","type":"t","specversion":"0.3"}]`, withKey), http.StatusBadRequest, "InvalidRequest")
	azureResourcesExpectErrorCode(t, publish("Avro", event, withKey), http.StatusBadRequest, "InvalidRequest")
	azureResourcesExpectErrorCode(t, publish("EventGridEvent", `[]`, withKey), http.StatusBadRequest, "InvalidRequest")

	resp = providerContractRequest(t, c.ts, http.MethodPost, "/azure/eventgrid-dataplane/2018-01-01/missing.westus2-1.eventgrid.azure.net?overload=EventGridEvent&api-version=2018-01-01", []byte(event), withKey)
	azureResourcesExpectErrorCode(t, resp, http.StatusNotFound, "ResourceNotFound")
	resp = providerContractRequest(t, c.ts, http.MethodPost, "/azure/eventgrid-dataplane/2018-01-01/orders.westus2-1.eventgrid.azure.net?overload=EventGridEvent&api-version=2017-01-01", []byte(event), withKey)
	azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, "InvalidApiVersionParameter")
}
