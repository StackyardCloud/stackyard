package server

import (
	"net/http"
	"strings"
	"testing"
)

const dcadmTestAPIVersion = "?api-version=2024-02-01"

func TestAzureDevCenterAdministrator20240201Lifecycle(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-dc")
	base := sub + "/resourceGroups/rg-dc/providers/Microsoft.DevCenter/"
	dc := base + "devcenters/contoso"
	dcID := strings.TrimPrefix(dc, "/azure")

	resp := do(http.MethodPut, dc+dcadmTestAPIVersion, `{"location":"eastus"}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	if uri, _ := providerContractJSONMap(t, resp)["properties"].(map[string]any)["devCenterUri"].(string); !strings.Contains(uri, "devcenter.azure.com") {
		t.Fatalf("unexpected devCenterUri %q", uri)
	}
	resp = do(http.MethodGet, dc+"/galleries/Default/images"+dcadmTestAPIVersion, "")
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	image := providerContractJSONMap(t, resp)["value"].([]any)[0].(map[string]any)["id"].(string)
	azureResourcesExpectStatus(t, do(http.MethodGet, "/azure"+image+"/versions/1.0.0"+dcadmTestAPIVersion, ""), http.StatusOK)

	azureResourcesExpectStatus(t, do(http.MethodPut, dc+"/catalogs/cat"+dcadmTestAPIVersion, `{"properties":{"gitHub":{"uri":"https://github.com/contoso/envs.git","branch":"main","path":"/Environments"}}}`), http.StatusCreated)
	resp = do(http.MethodGet, dc+"/catalogs/cat/environmentDefinitions/Sandbox"+dcadmTestAPIVersion, "")
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodPost, dc+"/catalogs/cat/sync"+dcadmTestAPIVersion, ""), http.StatusAccepted)
	azureResourcesExpectStatus(t, do(http.MethodPost, dc+"/catalogs/cat/getSyncErrorDetails"+dcadmTestAPIVersion, ""), http.StatusOK)

	nc := base + "networkConnections/nc1"
	azureResourcesExpectStatus(t, do(http.MethodPut, nc+dcadmTestAPIVersion, `{"location":"eastus","properties":{"domainJoinType":"AzureADJoin","subnetId":"/subscriptions/s/resourceGroups/rg/providers/Microsoft.Network/virtualNetworks/v/subnets/s"}}`), http.StatusCreated)
	azureResourcesExpectStatus(t, do(http.MethodGet, nc+"/healthChecks/latest"+dcadmTestAPIVersion, ""), http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodPost, nc+"/runHealthChecks"+dcadmTestAPIVersion, ""), http.StatusAccepted)
	azureResourcesExpectStatus(t, do(http.MethodPut, dc+"/attachednetworks/net"+dcadmTestAPIVersion, `{"properties":{"networkConnectionId":"`+strings.TrimPrefix(nc, "/azure")+`"}}`), http.StatusCreated)
	azureResourcesExpectStatus(t, do(http.MethodPut, dc+"/devboxdefinitions/win11"+dcadmTestAPIVersion, `{"location":"eastus","properties":{"imageReference":{"id":"`+image+`"},"sku":{"name":"general_i_8c32gb256ssd_v2"}}}`), http.StatusCreated)
	azureResourcesExpectStatus(t, do(http.MethodPut, dc+"/environmentTypes/dev"+dcadmTestAPIVersion, `{"properties":{}}`), http.StatusCreated)

	project := base + "projects/web"
	azureResourcesExpectStatus(t, do(http.MethodPut, project+dcadmTestAPIVersion, `{"location":"eastus","properties":{"devCenterId":"`+dcID+`"}}`), http.StatusCreated)
	for _, view := range []string{"attachednetworks", "devboxdefinitions", "allowedEnvironmentTypes"} {
		resp = do(http.MethodGet, project+"/"+view+dcadmTestAPIVersion, "")
		azureResourcesExpectStatus(t, resp, http.StatusOK)
		if n := len(providerContractJSONMap(t, resp)["value"].([]any)); n != 1 {
			t.Fatalf("project %s listed %d items", view, n)
		}
	}
	azureResourcesExpectStatus(t, do(http.MethodPut, project+"/environmentTypes/dev"+dcadmTestAPIVersion, `{"identity":{"type":"SystemAssigned"},"properties":{"deploymentTargetId":"/subscriptions/s","status":"Enabled"}}`), http.StatusCreated)
	pool := project + "/pools/pool1"
	azureResourcesExpectStatus(t, do(http.MethodPut, pool+dcadmTestAPIVersion, `{"location":"eastus","properties":{"devBoxDefinitionName":"win11","networkConnectionName":"net","licenseType":"Windows_Client","localAdministrator":"Enabled"}}`), http.StatusCreated)
	azureResourcesExpectStatus(t, do(http.MethodPut, pool+"/schedules/default"+dcadmTestAPIVersion, `{"properties":{"type":"StopDevBox","frequency":"Daily","time":"19:00","timeZone":"America/Los_Angeles"}}`), http.StatusCreated)

	resp = do(http.MethodPost, sub+"/providers/Microsoft.DevCenter/checkNameAvailability"+dcadmTestAPIVersion, `{"name":"contoso","type":"Microsoft.DevCenter/devcenters"}`)
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if providerContractJSONMap(t, resp)["nameAvailable"] != false {
		t.Fatal("expected contoso to be taken")
	}
	azureResourcesExpectStatus(t, do(http.MethodGet, sub+"/providers/Microsoft.DevCenter/skus"+dcadmTestAPIVersion, ""), http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodGet, sub+"/providers/Microsoft.DevCenter/locations/eastus/usages"+dcadmTestAPIVersion, ""), http.StatusOK)

	azureResourcesExpectStatus(t, do(http.MethodDelete, dc+dcadmTestAPIVersion, ""), http.StatusOK)
	azureResourcesExpectErrorCode(t, do(http.MethodGet, dc+"/catalogs/cat"+dcadmTestAPIVersion, ""), http.StatusNotFound, "ParentResourceNotFound")
}

func TestAzureDevCenterAdministrator20240201Validation(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-dc")
	base := sub + "/resourceGroups/rg-dc/providers/Microsoft.DevCenter/"
	azureResourcesExpectStatus(t, do(http.MethodPut, base+"devcenters/dc1"+dcadmTestAPIVersion, `{"location":"eastus"}`), http.StatusCreated)
	azureResourcesExpectStatus(t, do(http.MethodPut, base+"projects/proj1"+dcadmTestAPIVersion, `{"location":"eastus","properties":{"devCenterId":"x"}}`), http.StatusCreated)
	cases := []struct{ name, path, body string }{
		{"short name", base + "devcenters/x", `{"location":"eastus"}`},
		{"project dev center", base + "projects/p", `{"location":"eastus","properties":{}}`},
		{"catalog source", base + "devcenters/dc1/catalogs/c", `{"properties":{}}`},
		{"hybrid domain", base + "networkConnections/n", `{"location":"eastus","properties":{"domainJoinType":"HybridAzureADJoin","subnetId":"s"}}`},
		{"pool license", base + "projects/proj1/pools/x", `{"location":"eastus","properties":{"devBoxDefinitionName":"d","networkConnectionName":"n","licenseType":"Server","localAdministrator":"Enabled"}}`},
		{"schedule time", base + "projects/proj1/pools/x/schedules/default", `{"properties":{"type":"StopDevBox","frequency":"Daily","time":"7pm","timeZone":"UTC"}}`},
	}
	for _, tc := range cases {
		resp := do(http.MethodPut, tc.path+dcadmTestAPIVersion, tc.body)
		t.Run(tc.name, func(t *testing.T) {
			if resp.StatusCode != http.StatusBadRequest && resp.StatusCode != http.StatusNotFound {
				t.Fatalf("expected 400 or 404, got %d", resp.StatusCode)
			}
		})
	}
	azureResourcesExpectStatus(t, do(http.MethodPost, sub+"/providers/Microsoft.DevCenter/checkNameAvailability"+dcadmTestAPIVersion, `{}`), http.StatusBadRequest)
}
