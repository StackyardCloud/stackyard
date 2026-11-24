package server

import (
	"net/http"
	"strings"
	"testing"
)

const dtaskTestAPIVersion = "?api-version=2026-02-01"

func TestAzureDurableTaskLifecycle(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-dt")
	scheduler := sub + "/resourceGroups/rg-dt/providers/Microsoft.DurableTask/schedulers/Orchestrator"

	resp := do(http.MethodPut, scheduler+dtaskTestAPIVersion, `{"location":"East US","properties":{"ipAllowlist":["10.0.0.0/8"],"sku":{"name":"Dedicated","capacity":1}}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	if endpoint := providerContractJSONMap(t, resp)["properties"].(map[string]any)["endpoint"]; endpoint != "https://orchestrator.eastus.durabletask.io" {
		t.Fatalf("unexpected endpoint %v", endpoint)
	}

	resp = do(http.MethodPut, scheduler+"/taskHubs/default-hub"+dtaskTestAPIVersion, `{"properties":{}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	if url, _ := providerContractJSONMap(t, resp)["properties"].(map[string]any)["dashboardUrl"].(string); !strings.HasSuffix(url, "/schedulers/Orchestrator/taskhubs/default-hub") {
		t.Fatalf("unexpected dashboard URL %q", url)
	}
	azureResourcesExpectStatus(t, do(http.MethodPut, scheduler+"/retentionPolicies/default"+dtaskTestAPIVersion,
		`{"properties":{"retentionPolicies":[{"retentionPeriodInDays":7},{"retentionPeriodInDays":30,"orchestrationState":"Failed"}]}}`), http.StatusCreated)
	azureResourcesExpectStatus(t, do(http.MethodPatch, scheduler+dtaskTestAPIVersion, `{"tags":{"team":"core"}}`), http.StatusOK)

	if links := adnsTestValue(t, do(http.MethodGet, scheduler+"/privateLinkResources"+dtaskTestAPIVersion, "")); len(links) != 1 {
		t.Fatalf("unexpected private link resources: %#v", links)
	}
	azureResourcesExpectStatus(t, do(http.MethodPut, scheduler+"/privateEndpointConnections/pe1"+dtaskTestAPIVersion,
		`{"properties":{"privateLinkServiceConnectionState":{"status":"Approved"}}}`), http.StatusCreated)
	if ops := adnsTestValue(t, do(http.MethodGet, "/azure/providers/Microsoft.DurableTask/operations"+dtaskTestAPIVersion, "")); len(ops) == 0 {
		t.Fatal("expected provider operations")
	}
	if hubs := adnsTestValue(t, do(http.MethodGet, scheduler+"/taskHubs"+dtaskTestAPIVersion, "")); len(hubs) != 1 {
		t.Fatalf("expected one task hub, got %d", len(hubs))
	}

	azureResourcesExpectStatus(t, do(http.MethodDelete, scheduler+dtaskTestAPIVersion, ""), http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodGet, scheduler+"/taskHubs/default-hub"+dtaskTestAPIVersion, ""), http.StatusNotFound)
}

func TestAzureDurableTaskValidation(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-dt")
	schedulers := sub + "/resourceGroups/rg-dt/providers/Microsoft.DurableTask/schedulers/"

	cases := []struct{ name, body string }{
		{"missing allowlist", `{"location":"eastus","properties":{"sku":{"name":"Dedicated"}}}`},
		{"bad allowlist entry", `{"location":"eastus","properties":{"ipAllowlist":["not-an-ip"],"sku":{"name":"Dedicated"}}}`},
		{"unknown sku", `{"location":"eastus","properties":{"ipAllowlist":[],"sku":{"name":"Huge"}}}`},
		{"capacity out of range", `{"location":"eastus","properties":{"ipAllowlist":[],"sku":{"name":"Dedicated","capacity":9}}}`},
	}
	for _, tc := range cases {
		resp := do(http.MethodPut, schedulers+"s1"+dtaskTestAPIVersion, tc.body)
		t.Run(tc.name, func(t *testing.T) {
			azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, "InvalidRequestContent")
		})
	}
	scheduler := schedulers + "s1"
	azureResourcesExpectStatus(t, do(http.MethodPut, scheduler+dtaskTestAPIVersion, `{"location":"eastus","properties":{"ipAllowlist":["0.0.0.0/0"],"sku":{"name":"Consumption"}}}`), http.StatusCreated)
	azureResourcesExpectErrorCode(t, do(http.MethodPut, scheduler+"/retentionPolicies/custom"+dtaskTestAPIVersion, `{"properties":{"retentionPolicies":[]}}`), http.StatusBadRequest, "InvalidResourceName")
	azureResourcesExpectErrorCode(t, do(http.MethodPut, scheduler+"/retentionPolicies/default"+dtaskTestAPIVersion,
		`{"properties":{"retentionPolicies":[{"retentionPeriodInDays":7},{"retentionPeriodInDays":9}]}}`), http.StatusBadRequest, "InvalidRequestContent")
	azureResourcesExpectErrorCode(t, do(http.MethodPut, scheduler+"/retentionPolicies/default"+dtaskTestAPIVersion,
		`{"properties":{"retentionPolicies":[{"retentionPeriodInDays":365}]}}`), http.StatusBadRequest, "InvalidRequestContent")
	azureResourcesExpectStatus(t, do(http.MethodPatch, scheduler+"/taskHubs/hub"+dtaskTestAPIVersion, `{}`), http.StatusMethodNotAllowed)
	azureResourcesExpectErrorCode(t, do(http.MethodPut, schedulers+"absent/taskHubs/hub"+dtaskTestAPIVersion, `{"properties":{}}`), http.StatusNotFound, "ParentResourceNotFound")
}
