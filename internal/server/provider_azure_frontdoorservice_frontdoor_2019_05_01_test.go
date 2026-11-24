package server

import (
	"net/http"
	"strings"
	"testing"
)

const azureAfdcTestAPI = "?api-version=2019-05-01"

// azureAfdcTestBody is a front door with one frontend, pool and rule;
// fd is its resource ID without the /azure prefix.
func azureAfdcTestBody(fd string) string {
	return `{"location":"global","properties":{
		"frontendEndpoints":[{"name":"fe1","properties":{"hostName":"contosofd.azurefd.net","sessionAffinityEnabledState":"Disabled"}}],
		"loadBalancingSettings":[{"name":"lb1","properties":{"sampleSize":4,"successfulSamplesRequired":2}}],
		"healthProbeSettings":[{"name":"hp1","properties":{"path":"/","protocol":"Https","intervalInSeconds":120}}],
		"backendPools":[{"name":"pool1","properties":{"backends":[{"address":"www.contoso.com","httpPort":80,"httpsPort":443,"priority":1,"weight":50}],
			"loadBalancingSettings":{"id":"` + fd + `/loadBalancingSettings/lb1"},"healthProbeSettings":{"id":"` + fd + `/healthProbeSettings/hp1"}}}],
		"routingRules":[{"name":"rule1","properties":{"frontendEndpoints":[{"id":"` + fd + `/frontendEndpoints/fe1"}],"acceptedProtocols":["Https"],"patternsToMatch":["/*"],
			"routeConfiguration":{"@odata.type":"#Microsoft.Azure.FrontDoor.Models.FrontdoorForwardingConfiguration","backendPool":{"id":"` + fd + `/backendPools/pool1"}}}}]}}`
}

func TestAzureFrontDoorClassicLifecycle(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-fd")
	fd := sub + "/resourceGroups/rg-fd/providers/Microsoft.Network/frontDoors/contosofd"
	id := strings.TrimPrefix(fd, "/azure")

	resp := do(http.MethodPut, fd+azureAfdcTestAPI, azureAfdcTestBody(id))
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	props := providerContractJSONMap(t, resp)["properties"].(map[string]any)
	fe := props["frontendEndpoints"].([]any)[0].(map[string]any)
	if props["cname"] != "contosofd.azurefd.net" || fe["id"] != id+"/frontendEndpoints/fe1" ||
		fe["properties"].(map[string]any)["customHttpsProvisioningState"] != "Disabled" {
		t.Fatalf("unexpected front door: %#v", props)
	}

	resp = do(http.MethodGet, fd+"/frontendEndpoints"+azureAfdcTestAPI, "")
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if len(providerContractJSONMap(t, resp)["value"].([]any)) != 1 {
		t.Fatal("want one frontend endpoint")
	}
	azureResourcesExpectStatus(t, do(http.MethodPost, fd+"/frontendEndpoints/fe1/enableHttps"+azureAfdcTestAPI, `{"certificateSource":"FrontDoor","protocolType":"ServerNameIndication","minimumTlsVersion":"1.2"}`), http.StatusAccepted)
	resp = do(http.MethodGet, fd+"/frontendEndpoints/fe1"+azureAfdcTestAPI, "")
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if got := providerContractJSONMap(t, resp)["properties"].(map[string]any)["customHttpsProvisioningState"]; got != "Enabled" {
		t.Fatalf("customHttpsProvisioningState after enableHttps = %v", got)
	}
	azureResourcesExpectErrorCode(t, do(http.MethodPost, fd+"/frontendEndpoints/fe1/enableHttps"+azureAfdcTestAPI, `{"certificateSource":"Self"}`), http.StatusBadRequest, "InvalidRequestContent")
	azureResourcesExpectStatus(t, do(http.MethodPost, fd+"/frontendEndpoints/missing/disableHttps"+azureAfdcTestAPI, ""), http.StatusNotFound)

	azureResourcesExpectStatus(t, do(http.MethodPost, fd+"/purge"+azureAfdcTestAPI, `{"contentPaths":["/images/*"]}`), http.StatusAccepted)
	azureResourcesExpectErrorCode(t, do(http.MethodPost, fd+"/purge"+azureAfdcTestAPI, `{"contentPaths":[]}`), http.StatusBadRequest, "InvalidRequestContent")
	resp = do(http.MethodPost, fd+"/validateCustomDomain"+azureAfdcTestAPI, `{"hostName":"www.contoso.com"}`)
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if providerContractJSONMap(t, resp)["customDomainValidated"] != true {
		t.Fatal("custom domain not validated")
	}

	for _, path := range []string{"/azure/providers/Microsoft.Network/checkFrontDoorNameAvailability", sub + "/providers/Microsoft.Network/checkFrontDoorNameAvailability"} {
		resp = do(http.MethodPost, path+azureAfdcTestAPI, `{"name":"contosofd","type":"Microsoft.Network/frontDoors"}`)
		azureResourcesExpectStatus(t, resp, http.StatusOK)
		if got := providerContractJSONMap(t, resp)["nameAvailability"]; got != "Unavailable" {
			t.Fatalf("%s: nameAvailability = %v, want Unavailable", path, got)
		}
	}

	azureResourcesExpectStatus(t, do(http.MethodDelete, fd+azureAfdcTestAPI, ""), http.StatusOK)
	azureResourcesExpectErrorCode(t, do(http.MethodGet, fd+"/frontendEndpoints"+azureAfdcTestAPI, ""), http.StatusNotFound, "ParentResourceNotFound")
}

func TestAzureFrontDoorClassicValidation(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-fd")
	fd := sub + "/resourceGroups/rg-fd/providers/Microsoft.Network/frontDoors/contosofd"
	id := strings.TrimPrefix(fd, "/azure")
	valid := azureAfdcTestBody(id)
	azureResourcesExpectErrorCode(t, do(http.MethodPut, sub+"/resourceGroups/rg-fd/providers/Microsoft.Network/frontDoors/ab"+azureAfdcTestAPI, valid), http.StatusBadRequest, "InvalidResourceName")
	for name, body := range map[string]string{
		"no routing rules":  strings.Replace(valid, `"routingRules"`, `"unusedRules"`, 1),
		"no host name":      strings.Replace(valid, `"hostName":"contosofd.azurefd.net",`, "", 1),
		"port out of range": strings.Replace(valid, `"httpPort":80`, `"httpPort":70000`, 1),
		"unknown pool":      strings.Replace(valid, `/backendPools/pool1"}}}}]`, `/backendPools/other"}}}}]`, 1),
		"bad route type":    strings.Replace(valid, "FrontdoorForwardingConfiguration", "Unknown", 1),
	} {
		resp := do(http.MethodPut, fd+azureAfdcTestAPI, body)
		t.Run(name, func(t *testing.T) {
			azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, "InvalidRequestContent")
		})
	}
	azureResourcesExpectErrorCode(t, do(http.MethodPost, "/azure/providers/Microsoft.Network/checkFrontDoorNameAvailability"+azureAfdcTestAPI, `{"name":"x"}`), http.StatusBadRequest, "InvalidRequestContent")
}
