package server

import (
	"net/http"
	"testing"
)

const azureLockboxTestAPIVersion = "?api-version=2018-02-28-preview"

func TestAzureCustomerLockbox(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-lockbox")
	optIn := "/azure/providers/Microsoft.CustomerLockbox/tenantOptedIn/" + azureDefaultTenantID + azureLockboxTestAPIVersion
	for _, step := range []struct {
		action string
		want   bool
	}{{"", false}, {"enableLockbox", true}, {"disableLockbox", false}} {
		if step.action != "" {
			azureResourcesExpectStatus(t, do(http.MethodPost, "/azure/providers/Microsoft.CustomerLockbox/"+step.action+azureLockboxTestAPIVersion, ""), http.StatusOK)
		}
		resp := do(http.MethodGet, optIn, "")
		azureResourcesExpectStatus(t, resp, http.StatusOK)
		if got := providerContractJSONMap(t, resp)["isOptedIn"]; got != step.want {
			t.Fatalf("after %q: expected isOptedIn %v, got %v", step.action, step.want, got)
		}
	}
	resp := do(http.MethodGet, sub+"/providers/Microsoft.CustomerLockbox/requests"+azureLockboxTestAPIVersion, "")
	if got := providerContractJSONMap(t, resp)["value"].([]any); len(got) != 0 {
		t.Fatalf("expected no requests, got %d", len(got))
	}
	azureResourcesExpectErrorCode(t, do(http.MethodGet, sub+"/providers/Microsoft.CustomerLockbox/requests/a1b2"+azureLockboxTestAPIVersion, ""), http.StatusNotFound, "ResourceNotFound")
}
