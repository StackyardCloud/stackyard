package server

import (
	"net/http"
	"strings"
	"testing"
)

const avdTestAPIVersion = "?api-version=2025-10-10"

func TestAzureDesktopVirtualization20251010Lifecycle(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-avd")
	base := sub + "/resourceGroups/rg-avd/providers/Microsoft.DesktopVirtualization/"
	hostPool := base + "hostPools/hp1"
	hostPoolID := strings.TrimPrefix(hostPool, "/azure")

	resp := do(http.MethodPut, hostPool+avdTestAPIVersion, `{"location":"eastus","properties":{"hostPoolType":"Pooled","loadBalancerType":"BreadthFirst","preferredAppGroupType":"Desktop","maxSessionLimit":10,
		"registrationInfo":{"registrationTokenOperation":"Update","expirationTime":"2030-01-01T00:00:00Z","token":"client"}}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	if info := providerContractJSONMap(t, resp)["properties"].(map[string]any)["registrationInfo"].(map[string]any); info["token"] != nil {
		t.Fatalf("client token echoed: %#v", info)
	}
	resp = do(http.MethodPost, hostPool+"/retrieveRegistrationToken"+avdTestAPIVersion, "")
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if token, _ := providerContractJSONMap(t, resp)["token"].(string); !strings.HasPrefix(token, "avd-") {
		t.Fatalf("unexpected registration token %q", token)
	}
	resp = do(http.MethodPost, hostPool+"/listRegistrationTokens"+avdTestAPIVersion, "")
	if n := len(providerContractJSONMap(t, resp)["value"].([]any)); n != 1 {
		t.Fatalf("listed %d registration tokens", n)
	}
	resp = do(http.MethodPost, hostPool+"/importAppAttachPackageInfo"+avdTestAPIVersion, `{"path":"\\\\share\\apps\\Notepad.vhdx","packageArchitecture":"x64"}`)
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if !strings.Contains(providerContractJSONMap(t, resp)["value"].([]any)[0].(map[string]any)["name"].(string), "notepad") {
		t.Fatal("import did not describe the package")
	}

	group := base + "applicationGroups/desktop"
	azureResourcesExpectStatus(t, do(http.MethodPut, group+avdTestAPIVersion, `{"location":"eastus","properties":{"hostPoolArmPath":"`+hostPoolID+`","applicationGroupType":"Desktop"}}`), http.StatusCreated)
	azureResourcesExpectStatus(t, do(http.MethodGet, group+"/desktops/SessionDesktop"+avdTestAPIVersion, ""), http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodPatch, group+"/desktops/SessionDesktop"+avdTestAPIVersion, `{"properties":{"friendlyName":"Work"}}`), http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodDelete, group+"/desktops/SessionDesktop"+avdTestAPIVersion, ""), http.StatusMethodNotAllowed)
	azureResourcesExpectStatus(t, do(http.MethodPut, group+"/applications/word"+avdTestAPIVersion, `{"properties":{"commandLineSetting":"DoNotAllow","filePath":"C:\\word.exe"}}`), http.StatusCreated)
	resp = do(http.MethodGet, group+"/startMenuItems"+avdTestAPIVersion, "")
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if n := len(providerContractJSONMap(t, resp)["value"].([]any)); n != 1 {
		t.Fatalf("listed %d start menu items", n)
	}

	azureResourcesExpectStatus(t, do(http.MethodPut, base+"workspaces/ws"+avdTestAPIVersion, `{"location":"eastus","properties":{"applicationGroupReferences":["`+strings.TrimPrefix(group, "/azure")+`"]}}`), http.StatusCreated)
	resp = do(http.MethodGet, base+"workspaces/ws/privateLinkResources"+avdTestAPIVersion, "")
	azureResourcesExpectStatus(t, resp, http.StatusOK)

	plan := base + "scalingPlans/plan"
	azureResourcesExpectStatus(t, do(http.MethodPut, plan+avdTestAPIVersion, `{"location":"eastus","properties":{"timeZone":"UTC","hostPoolType":"Pooled","hostPoolReferences":[{"hostPoolArmPath":"`+hostPoolID+`","scalingPlanEnabled":true}]}}`), http.StatusCreated)
	azureResourcesExpectStatus(t, do(http.MethodPut, plan+"/pooledSchedules/weekdays"+avdTestAPIVersion, `{"properties":{"daysOfWeek":["Monday","Friday"]}}`), http.StatusCreated)
	resp = do(http.MethodGet, hostPool+"/scalingPlans"+avdTestAPIVersion, "")
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if n := len(providerContractJSONMap(t, resp)["value"].([]any)); n != 1 {
		t.Fatalf("host pool lists %d scaling plans", n)
	}
	resp = do(http.MethodGet, hostPool+"/userSessions"+avdTestAPIVersion, "")
	azureResourcesExpectStatus(t, resp, http.StatusOK)

	azureResourcesExpectStatus(t, do(http.MethodGet, "/azure/providers/Microsoft.DesktopVirtualization/operations"+avdTestAPIVersion, ""), http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodDelete, hostPool+avdTestAPIVersion, ""), http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodGet, hostPool+avdTestAPIVersion, ""), http.StatusNotFound)
}

func TestAzureDesktopVirtualization20251010Validation(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-avd")
	base := sub + "/resourceGroups/rg-avd/providers/Microsoft.DesktopVirtualization/"
	cases := []struct{ name, path, body string }{
		{"pool type", base + "hostPools/x", `{"location":"eastus","properties":{"hostPoolType":"Shared","loadBalancerType":"BreadthFirst","preferredAppGroupType":"Desktop"}}`},
		{"persistent pooled", base + "hostPools/x", `{"location":"eastus","properties":{"hostPoolType":"Pooled","loadBalancerType":"Persistent","preferredAppGroupType":"Desktop"}}`},
		{"group host pool", base + "applicationGroups/x", `{"location":"eastus","properties":{"applicationGroupType":"Desktop"}}`},
		{"scaling time zone", base + "scalingPlans/x", `{"location":"eastus","properties":{}}`},
		{"app attach image", base + "appAttachPackages/x", `{"location":"eastus","properties":{}}`},
	}
	for _, tc := range cases {
		resp := do(http.MethodPut, tc.path+avdTestAPIVersion, tc.body)
		t.Run(tc.name, func(t *testing.T) {
			azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, "InvalidRequestContent")
		})
	}
	azureResourcesExpectStatus(t, do(http.MethodPut, base+"hostPools/hp/sessionHosts/vm1"+avdTestAPIVersion, `{}`), http.StatusMethodNotAllowed)
	azureResourcesExpectStatus(t, do(http.MethodGet, base+"hostPools/missing/scalingPlans"+avdTestAPIVersion, ""), http.StatusNotFound)
}
