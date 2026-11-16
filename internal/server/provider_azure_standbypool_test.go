package server

import (
	"net/http"
	"strings"
	"testing"
)

const azureStandbyPoolTestAPIVersion = "?api-version=2026-04-01"

func TestAzureStandbyPoolLifecycle(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-standby")
	ids := strings.TrimPrefix(sub, "/azure") + "/resourceGroups/rg-standby/providers"
	pool := sub + "/resourceGroups/rg-standby/providers/Microsoft.StandbyPool/standbyVirtualMachinePools/web-pool"

	azureResourcesExpectStatus(t, do(http.MethodPut, pool+azureStandbyPoolTestAPIVersion, `{"location":"eastus","properties":{"elasticityProfile":{"maxReadyCapacity":4,"minReadyCapacity":2},"virtualMachineState":"Mix","vmStateDistribution":{"runningPercent":50,"deallocatedPercent":25,"hibernatedPercent":25},"attachedVirtualMachineScaleSetId":"`+ids+`/Microsoft.Compute/virtualMachineScaleSets/web"}}`), http.StatusCreated)
	resp := do(http.MethodGet, pool+"/runtimeViews/latest"+azureStandbyPoolTestAPIVersion, "")
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	counts := providerContractJSONMap(t, resp)["properties"].(map[string]any)["instanceCountSummary"].([]any)[0].(map[string]any)["instanceCountsByState"].([]any)
	byState := map[string]any{}
	for _, c := range counts {
		byState[c.(map[string]any)["state"].(string)] = c.(map[string]any)["count"]
	}
	if byState["Running"] != float64(2) || byState["Deallocated"] != float64(1) || byState["Hibernated"] != float64(1) {
		t.Fatalf("unexpected instance counts: %#v", byState)
	}
	resp = do(http.MethodGet, pool+"/standbyVirtualMachines"+azureStandbyPoolTestAPIVersion, "")
	vms := providerContractJSONMap(t, resp)["value"].([]any)
	if len(vms) != 4 {
		t.Fatalf("expected four standby machines, got %d", len(vms))
	}
	resp = do(http.MethodGet, pool+"/standbyVirtualMachines/web_3"+azureStandbyPoolTestAPIVersion, "")
	if got := providerContractJSONMap(t, resp)["properties"].(map[string]any)["virtualMachineResourceId"]; got != ids+"/Microsoft.Compute/virtualMachines/web_3" {
		t.Fatalf("unexpected machine ID: %v", got)
	}
	// PATCH skips validation, so bad stored values must not break the views.
	azureResourcesExpectStatus(t, do(http.MethodPatch, pool+azureStandbyPoolTestAPIVersion, `{"properties":{"vmStateDistribution":{"deallocatedPercent":1e300,"hibernatedPercent":-5}}}`), http.StatusOK)
	resp = do(http.MethodGet, pool+"/runtimeViews/latest"+azureStandbyPoolTestAPIVersion, "")
	counts = providerContractJSONMap(t, resp)["properties"].(map[string]any)["instanceCountSummary"].([]any)[0].(map[string]any)["instanceCountsByState"].([]any)
	if got := counts[0].(map[string]any); got["state"] != "Running" || got["count"] != float64(0) {
		t.Fatalf("expected every machine deallocated, got %#v", counts)
	}
	for _, patch := range []string{`{"properties":{"elasticityProfile":{"maxReadyCapacity":"lots"}}}`, `{"properties":{"elasticityProfile":{"maxReadyCapacity":4},"attachedVirtualMachineScaleSetId":"web"}}`} {
		azureResourcesExpectStatus(t, do(http.MethodPatch, pool+azureStandbyPoolTestAPIVersion, patch), http.StatusOK)
		resp = do(http.MethodGet, pool+"/standbyVirtualMachines"+azureStandbyPoolTestAPIVersion, "")
		if got := providerContractJSONMap(t, resp)["value"].([]any); len(got) != 0 {
			t.Fatalf("expected no machines after %s, got %d", patch, len(got))
		}
	}

	group := sub + "/resourceGroups/rg-standby/providers/Microsoft.StandbyPool/standbyContainerGroupPools/jobs"
	azureResourcesExpectStatus(t, do(http.MethodPut, group+azureStandbyPoolTestAPIVersion, `{"location":"eastus","properties":{"elasticityProfile":{"maxReadyCapacity":10,"refillPolicy":"always"},"containerGroupProperties":{"containerGroupProfile":{"id":"`+ids+`/Microsoft.ContainerInstance/containerGroupProfiles/worker","revision":1},"subnetIds":[{"id":"`+ids+`/Microsoft.Network/virtualNetworks/v/subnets/aci"}]},"zones":["1","2"]}}`), http.StatusCreated)
	resp = do(http.MethodGet, group+"/runtimeViews"+azureStandbyPoolTestAPIVersion, "")
	views := providerContractJSONMap(t, resp)["value"].([]any)
	if len(views) != 1 || views[0].(map[string]any)["name"] != "latest" {
		t.Fatalf("unexpected runtime views: %#v", views)
	}

	azureResourcesExpectStatus(t, do(http.MethodDelete, pool+azureStandbyPoolTestAPIVersion, ""), http.StatusOK)
	azureResourcesExpectErrorCode(t, do(http.MethodGet, pool+"/runtimeViews/latest"+azureStandbyPoolTestAPIVersion, ""), http.StatusNotFound, "ParentResourceNotFound")
	azureResourcesExpectErrorCode(t, do(http.MethodPut, group+"/runtimeViews/latest"+azureStandbyPoolTestAPIVersion, `{}`), http.StatusMethodNotAllowed, "MethodNotAllowed")
}

func TestAzureStandbyPoolValidation(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-standby")
	pools := sub + "/resourceGroups/rg-standby/providers/Microsoft.StandbyPool"
	profile := `"containerGroupProperties":{"containerGroupProfile":{"id":"/subscriptions/x/resourceGroups/rg/providers/Microsoft.ContainerInstance/containerGroupProfiles/p"}}`
	cases := []struct{ name, path, body string }{
		{"elasticity", "/standbyVirtualMachinePools/pool1", `{"location":"eastus","properties":{"virtualMachineState":"Running"}}`},
		{"max capacity", "/standbyVirtualMachinePools/pool1", `{"location":"eastus","properties":{"elasticityProfile":{"maxReadyCapacity":2001},"virtualMachineState":"Running"}}`},
		{"min capacity", "/standbyVirtualMachinePools/pool1", `{"location":"eastus","properties":{"elasticityProfile":{"maxReadyCapacity":2,"minReadyCapacity":3},"virtualMachineState":"Running"}}`},
		{"state", "/standbyVirtualMachinePools/pool1", `{"location":"eastus","properties":{"elasticityProfile":{"maxReadyCapacity":2},"virtualMachineState":"Stopped"}}`},
		{"scale set", "/standbyVirtualMachinePools/pool1", `{"location":"eastus","properties":{"elasticityProfile":{"maxReadyCapacity":2},"virtualMachineState":"Running","attachedVirtualMachineScaleSetId":"web"}}`},
		{"mix total", "/standbyVirtualMachinePools/pool1", `{"location":"eastus","properties":{"elasticityProfile":{"maxReadyCapacity":2},"virtualMachineState":"Mix","vmStateDistribution":{"runningPercent":50}}}`},
		{"refill", "/standbyContainerGroupPools/pool1", `{"location":"eastus","properties":{"elasticityProfile":{"maxReadyCapacity":2,"refillPolicy":"never"},` + profile + `}}`},
		{"profile", "/standbyContainerGroupPools/pool1", `{"location":"eastus","properties":{"elasticityProfile":{"maxReadyCapacity":2}}}`},
		{"subnet", "/standbyContainerGroupPools/pool1", `{"location":"eastus","properties":{"elasticityProfile":{"maxReadyCapacity":2},"containerGroupProperties":{"containerGroupProfile":{"id":"/subscriptions/x/resourceGroups/rg/providers/Microsoft.ContainerInstance/containerGroupProfiles/p"},"subnetIds":[{"id":"vnet"}]}}}`},
	}
	for _, tc := range cases {
		resp := do(http.MethodPut, pools+tc.path+azureStandbyPoolTestAPIVersion, tc.body)
		t.Run(tc.name, func(t *testing.T) {
			azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, "InvalidRequestContent")
		})
	}
	azureResourcesExpectErrorCode(t, do(http.MethodPut, pools+"/standbyVirtualMachinePools/p_1"+azureStandbyPoolTestAPIVersion, `{"location":"eastus","properties":{"elasticityProfile":{"maxReadyCapacity":2},"virtualMachineState":"Running"}}`), http.StatusBadRequest, "InvalidResourceName")
}
