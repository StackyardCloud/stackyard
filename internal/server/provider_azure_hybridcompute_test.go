package server

import (
	"net/http"
	"testing"
	"time"
)

const azureHybridComputeTestAPIVersion = "?api-version=2024-07-10"

func TestAzureHybridComputeLifecycle(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-arc")
	machine := sub + "/resourceGroups/rg-arc/providers/Microsoft.HybridCompute/machines/web01"

	resp := do(http.MethodPut, machine+azureHybridComputeTestAPIVersion, `{"location":"eastus","identity":{"type":"SystemAssigned"},"properties":{"osType":"linux","status":"Disconnected"}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	props := providerContractJSONMap(t, resp)["properties"].(map[string]any)
	if props["status"] != "Connected" || len(props["vmId"].(string)) != 36 || props["osType"] != "linux" {
		t.Fatalf("unexpected machine properties: %#v", props)
	}

	// A PUT that updates the machine keeps its status change time.
	time.Sleep(1100 * time.Millisecond)
	resp = do(http.MethodPut, machine+azureHybridComputeTestAPIVersion, `{"location":"eastus","properties":{"osType":"Linux"}}`)
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if got := providerContractJSONMap(t, resp)["properties"].(map[string]any)["lastStatusChange"]; got != props["lastStatusChange"] {
		t.Fatalf("lastStatusChange moved on update: %v, was %v", got, props["lastStatusChange"])
	}

	ext := machine + "/extensions/monitor"
	resp = do(http.MethodPut, ext+azureHybridComputeTestAPIVersion, `{"location":"eastus","properties":{"publisher":"Microsoft.Azure.Monitor","type":"AzureMonitorLinuxAgent","settings":{"workspaceId":"w"},"protectedSettings":{"workspaceKey":"secret"}}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	if props := providerContractJSONMap(t, resp)["properties"].(map[string]any); props["protectedSettings"] != nil || props["settings"] == nil {
		t.Fatalf("expected protected settings stripped, got %#v", props)
	}

	azureResourcesExpectStatus(t, do(http.MethodDelete, machine+azureHybridComputeTestAPIVersion, ""), http.StatusOK)
	azureResourcesExpectErrorCode(t, do(http.MethodGet, ext+azureHybridComputeTestAPIVersion, ""), http.StatusNotFound, "ParentResourceNotFound")
}

func TestAzureHybridComputeValidation(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-arc")
	machine := sub + "/resourceGroups/rg-arc/providers/Microsoft.HybridCompute/machines/web01"
	cases := []struct{ name, path, body string }{
		{"bad kind", machine, `{"location":"eastus","kind":"Azure","properties":{}}`},
		{"bad os", machine, `{"location":"eastus","properties":{"osType":"macos"}}`},
		{"extension publisher", machine + "/extensions/x", `{"location":"eastus","properties":{"type":"CustomScript"}}`},
	}
	azureResourcesExpectStatus(t, do(http.MethodPut, sub+"/resourceGroups/rg-arc/providers/Microsoft.HybridCompute/machines/other"+azureHybridComputeTestAPIVersion, `{"location":"eastus","kind":"AWS","properties":{}}`), http.StatusCreated)
	azureResourcesExpectStatus(t, do(http.MethodPut, machine+azureHybridComputeTestAPIVersion, `{"location":"eastus","properties":{}}`), http.StatusCreated)
	for _, tc := range cases {
		resp := do(http.MethodPut, tc.path+azureHybridComputeTestAPIVersion, tc.body)
		t.Run(tc.name, func(t *testing.T) {
			azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, "InvalidRequestContent")
		})
	}
}
