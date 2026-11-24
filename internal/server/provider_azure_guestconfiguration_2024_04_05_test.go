package server

import (
	"net/http"
	"strings"
	"testing"
)

const azureGcfgTestAPI = "?api-version=2024-04-05"

func TestAzureGuestConfigurationAssignmentLifecycle(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-gc")
	vm := sub + "/resourceGroups/rg-gc/providers/Microsoft.Compute/virtualMachines/vm1"
	assignment := vm + "/providers/Microsoft.GuestConfiguration/guestConfigurationAssignments/AzureWindowsBaseline"

	resp := do(http.MethodPut, assignment+azureGcfgTestAPI, `{"location":"eastus","properties":{"guestConfiguration":{"name":"AzureWindowsBaseline","version":"1.*","assignmentType":"ApplyAndMonitor","configurationParameter":[{"name":"MinimumPasswordLength","value":"14"}],"configurationProtectedParameter":[{"name":"AdminPassword","value":"hunter2-secret"}]}}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	raw := string(mustBody(t, resp))
	if strings.Contains(raw, "hunter2-secret") || !strings.Contains(raw, `"complianceStatus":"Compliant"`) || !strings.Contains(raw, `"resourceType":"Microsoft.Compute/virtualMachines"`) {
		t.Fatalf("unexpected assignment: %s", raw)
	}

	for scope, want := range map[string]int{vm: 1, sub + "/resourceGroups/rg-gc": 1, sub: 1} {
		resp := do(http.MethodGet, scope+"/providers/Microsoft.GuestConfiguration/guestConfigurationAssignments"+azureGcfgTestAPI, "")
		azureResourcesExpectStatus(t, resp, http.StatusOK)
		if got := len(providerContractJSONMap(t, resp)["value"].([]any)); got != want {
			t.Fatalf("list at %s returned %d assignments, want %d", scope, got, want)
		}
	}

	resp = do(http.MethodGet, assignment+"/reports"+azureGcfgTestAPI, "")
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	reports := providerContractJSONMap(t, resp)["value"].([]any)
	report := reports[0].(map[string]any)
	if len(reports) != 1 || report["properties"].(map[string]any)["complianceStatus"] != "Compliant" {
		t.Fatalf("unexpected reports: %#v", reports)
	}
	azureResourcesExpectStatus(t, do(http.MethodGet, assignment+"/reports/"+report["name"].(string)+azureGcfgTestAPI, ""), http.StatusOK)

	// Scale sets and Arc-enabled servers take assignments too.
	for _, scope := range []string{"Microsoft.Compute/virtualMachineScaleSets/vmss1", "Microsoft.HybridCompute/machines/arc1"} {
		azureResourcesExpectStatus(t, do(http.MethodPut, sub+"/resourceGroups/rg-gc/providers/"+scope+"/providers/Microsoft.GuestConfiguration/guestConfigurationAssignments/a1"+azureGcfgTestAPI,
			`{"properties":{"guestConfiguration":{"name":"AuditSecureProtocol"}}}`), http.StatusCreated)
	}

	azureResourcesExpectStatus(t, do(http.MethodDelete, assignment+azureGcfgTestAPI, ""), http.StatusOK)
	azureResourcesExpectErrorCode(t, do(http.MethodGet, assignment+"/reports"+azureGcfgTestAPI, ""), http.StatusNotFound, "ParentResourceNotFound")
}

func TestAzureGuestConfigurationAssignmentValidation(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-gc")
	vm := sub + "/resourceGroups/rg-gc/providers/Microsoft.Compute/virtualMachines/vm1/providers/Microsoft.GuestConfiguration/guestConfigurationAssignments/a1"
	azureResourcesExpectErrorCode(t, do(http.MethodPut, sub+"/resourceGroups/rg-gc/providers/Microsoft.Storage/storageAccounts/st1/providers/Microsoft.GuestConfiguration/guestConfigurationAssignments/a1"+azureGcfgTestAPI,
		`{"properties":{"guestConfiguration":{"name":"x"}}}`), http.StatusBadRequest, "InvalidScope")
	for name, body := range map[string]string{
		"no configuration name": `{"properties":{"guestConfiguration":{}}}`,
		"bad assignment type":   `{"properties":{"guestConfiguration":{"name":"x","assignmentType":"Enforce"}}}`,
		"unnamed parameter":     `{"properties":{"guestConfiguration":{"name":"x","configurationParameter":[{"value":"1"}]}}}`,
		"http content":          `{"properties":{"guestConfiguration":{"name":"x","contentUri":"http://example.com/p.zip"}}}`,
	} {
		resp := do(http.MethodPut, vm+azureGcfgTestAPI, body)
		t.Run(name, func(t *testing.T) {
			azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, "InvalidRequestContent")
		})
	}
	azureResourcesExpectStatus(t, do(http.MethodGet, vm+azureGcfgTestAPI, ""), http.StatusNotFound)
}
