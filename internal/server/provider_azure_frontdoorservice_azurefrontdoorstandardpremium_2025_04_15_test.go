package server

import (
	"net/http"
	"strings"
	"testing"
)

const azureAfdxTestAPI = "?api-version=2025-04-15"

func TestAzureFrontDoorStandardLifecycle(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-afd")
	profile := sub + "/resourceGroups/rg-afd/providers/Microsoft.Cdn/profiles/edge"
	id := strings.TrimPrefix(profile, "/azure")

	resp := do(http.MethodPut, profile+azureAfdxTestAPI, `{"location":"global","sku":{"name":"Standard_AzureFrontDoor"}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	if providerContractJSONMap(t, resp)["sku"].(map[string]any)["name"] != "Standard_AzureFrontDoor" {
		t.Fatal("profile lost its sku")
	}

	resp = do(http.MethodPut, profile+"/afdEndpoints/web"+azureAfdxTestAPI, `{"location":"global","properties":{"enabledState":"Enabled"}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	if host := providerContractJSONMap(t, resp)["properties"].(map[string]any)["hostName"].(string); !strings.HasSuffix(host, ".z01.azurefd.net") {
		t.Fatalf("endpoint hostName = %q", host)
	}
	azureResourcesExpectStatus(t, do(http.MethodPut, profile+"/originGroups/og"+azureAfdxTestAPI, `{"properties":{"healthProbeSettings":{"probeProtocol":"Https","probeRequestType":"HEAD","probeIntervalInSeconds":100},"sessionAffinityState":"Disabled"}}`), http.StatusCreated)
	azureResourcesExpectStatus(t, do(http.MethodPut, profile+"/originGroups/og/origins/app"+azureAfdxTestAPI, `{"properties":{"hostName":"app.contoso.com","httpsPort":443,"priority":1,"weight":1000}}`), http.StatusCreated)
	resp = do(http.MethodPut, profile+"/secrets/cert"+azureAfdxTestAPI, `{"properties":{"parameters":{"type":"CustomerCertificate","secretSource":{"id":"/subscriptions/s/resourceGroups/rg/providers/Microsoft.KeyVault/vaults/kv/secrets/cert"}}}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	if providerContractJSONMap(t, resp)["properties"].(map[string]any)["parameters"].(map[string]any)["thumbprint"] == "" {
		t.Fatal("customer certificate has no thumbprint")
	}
	resp = do(http.MethodPut, profile+"/customDomains/www"+azureAfdxTestAPI, `{"properties":{"hostName":"www.contoso.com","tlsSettings":{"certificateType":"CustomerCertificate","minimumTlsVersion":"TLS12","secret":{"id":"`+id+`/secrets/cert"}}}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	if providerContractJSONMap(t, resp)["properties"].(map[string]any)["domainValidationState"] != "Approved" {
		t.Fatal("custom domain not approved")
	}
	azureResourcesExpectStatus(t, do(http.MethodPost, profile+"/customDomains/www/refreshValidationToken"+azureAfdxTestAPI, ""), http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodPut, profile+"/ruleSets/rules1"+azureAfdxTestAPI, `{}`), http.StatusCreated)
	azureResourcesExpectStatus(t, do(http.MethodPut, profile+"/ruleSets/rules1/rules/cache"+azureAfdxTestAPI, `{"properties":{"order":1,"actions":[{"name":"CacheExpiration","parameters":{"cacheBehavior":"Override","cacheType":"All","cacheDuration":"01:00:00"}}],"matchProcessingBehavior":"Continue"}}`), http.StatusCreated)

	resp = do(http.MethodPut, profile+"/afdEndpoints/web/routes/default"+azureAfdxTestAPI, `{"properties":{"originGroup":{"id":"`+id+`/originGroups/og"},"customDomains":[{"id":"`+id+`/customDomains/www"}],"ruleSets":[{"id":"`+id+`/ruleSets/rules1"}],"httpsRedirect":"Enabled"}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	route := providerContractJSONMap(t, resp)["properties"].(map[string]any)
	if route["patternsToMatch"].([]any)[0] != "/*" || route["forwardingProtocol"] != "MatchRequest" || route["endpointName"] != "web" {
		t.Fatalf("route defaults missing: %#v", route)
	}
	azureResourcesExpectStatus(t, do(http.MethodPut, profile+"/securityPolicies/waf"+azureAfdxTestAPI, `{"properties":{"parameters":{"type":"WebApplicationFirewall","wafPolicy":{"id":"/subscriptions/s/resourceGroups/rg/providers/Microsoft.Network/FrontDoorWebApplicationFirewallPolicies/p"},"associations":[{"domains":[{"id":"`+id+`/afdEndpoints/web"},{"id":"`+id+`/customDomains/www"}],"patternsToMatch":["/*"]}]}}}`), http.StatusCreated)

	resp = do(http.MethodPost, profile+"/afdEndpoints/web/usages"+azureAfdxTestAPI, "")
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if got := providerContractJSONMap(t, resp)["value"].([]any)[0].(map[string]any)["currentValue"]; got != 1.0 {
		t.Fatalf("endpoint route usage = %v, want 1", got)
	}
	resp = do(http.MethodPost, profile+"/usages"+azureAfdxTestAPI, "")
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if !strings.Contains(string(mustBody(t, resp)), `"value":"afdEndpoints"`) {
		t.Fatal("profile usages miss afdEndpoints")
	}
	azureResourcesExpectStatus(t, do(http.MethodPost, profile+"/afdEndpoints/web/purge"+azureAfdxTestAPI, `{"contentPaths":["/*"]}`), http.StatusAccepted)

	resp = do(http.MethodPost, sub+"/resourceGroups/rg-afd/providers/Microsoft.Cdn/checkEndpointNameAvailability"+azureAfdxTestAPI, `{"name":"web","type":"Microsoft.Cdn/Profiles/AfdEndpoints"}`)
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if providerContractJSONMap(t, resp)["nameAvailable"] != false {
		t.Fatal("an existing endpoint name must be unavailable")
	}
	resp = do(http.MethodPost, profile+"/checkEndpointNameAvailability"+azureAfdxTestAPI, `{"name":"fresh","type":"Microsoft.Cdn/Profiles/AfdEndpoints"}`)
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if got := providerContractJSONMap(t, resp); got["nameAvailable"] != true || !strings.HasPrefix(got["availableHostname"].(string), "fresh-") {
		t.Fatalf("unexpected name check: %#v", got)
	}
	resp = do(http.MethodPost, profile+"/checkHostNameAvailability"+azureAfdxTestAPI, `{"hostName":"www.contoso.com"}`)
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if providerContractJSONMap(t, resp)["nameAvailable"] != false {
		t.Fatal("a used custom domain host must be unavailable")
	}
	resp = do(http.MethodPost, profile+"/validateSecret"+azureAfdxTestAPI, `{"secretType":"CustomerCertificate","secretSource":{"id":"/subscriptions/s/resourceGroups/rg/providers/Microsoft.KeyVault/vaults/kv/secrets/cert"}}`)
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if providerContractJSONMap(t, resp)["status"] != "Valid" {
		t.Fatal("a Key Vault secret reference must be valid")
	}

	resp = do(http.MethodGet, profile+"/getLogAnalyticsMetrics"+azureAfdxTestAPI+"&metrics=clientRequestCount&dateTimeBegin=2026-01-01T00:00:00Z&dateTimeEnd=2026-01-02T00:00:00Z&granularity=PT1H", "")
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if len(providerContractJSONMap(t, resp)["series"].([]any)) != 1 {
		t.Fatal("want one series")
	}
	azureResourcesExpectErrorCode(t, do(http.MethodGet, profile+"/getLogAnalyticsMetrics"+azureAfdxTestAPI+"&metrics=clientRequestCount&dateTimeBegin=2026-01-01T00:00:00Z&dateTimeEnd=2026-01-02T00:00:00Z", ""), http.StatusBadRequest, "InvalidRequestContent")
	resp = do(http.MethodGet, profile+"/getLogAnalyticsResources"+azureAfdxTestAPI, "")
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if len(providerContractJSONMap(t, resp)["endpoints"].([]any)) != 1 {
		t.Fatal("log analytics resources miss the endpoint")
	}
	azureResourcesExpectStatus(t, do(http.MethodGet, profile+"/getLogAnalyticsLocations"+azureAfdxTestAPI, ""), http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodGet, profile+"/getWafLogAnalyticsRankings"+azureAfdxTestAPI+"&metrics=clientRequestCount&rankings=ruleId&maxRanking=5&dateTimeBegin=2026-01-01T00:00:00Z&dateTimeEnd=2026-01-02T00:00:00Z", ""), http.StatusOK)

	resp = do(http.MethodPost, profile+"/upgrade"+azureAfdxTestAPI, `{"wafMappingList":[]}`)
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if providerContractJSONMap(t, resp)["sku"].(map[string]any)["name"] != "Premium_AzureFrontDoor" {
		t.Fatal("upgrade did not move the profile to Premium")
	}
	azureResourcesExpectErrorCode(t, do(http.MethodPost, profile+"/upgrade"+azureAfdxTestAPI, `{"wafMappingList":[]}`), http.StatusBadRequest, "InvalidRequestContent")

	azureResourcesExpectStatus(t, do(http.MethodDelete, profile+azureAfdxTestAPI, ""), http.StatusAccepted)
	azureResourcesExpectErrorCode(t, do(http.MethodGet, profile+"/afdEndpoints/web"+azureAfdxTestAPI, ""), http.StatusNotFound, "ParentResourceNotFound")
}

func TestAzureFrontDoorStandardValidation(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-afd")
	classic := sub + "/resourceGroups/rg-afd/providers/Microsoft.Cdn/profiles/classic"
	azureResourcesExpectStatus(t, do(http.MethodPut, classic+azureAfdxTestAPI, `{"location":"global","sku":{"name":"Standard_Microsoft"}}`), http.StatusCreated)
	azureResourcesExpectErrorCode(t, do(http.MethodPut, classic+"/afdEndpoints/web"+azureAfdxTestAPI, `{"location":"global"}`), http.StatusBadRequest, "InvalidRequestContent")
	azureResourcesExpectErrorCode(t, do(http.MethodPut, sub+"/resourceGroups/rg-afd/providers/Microsoft.Cdn/profiles/missing/afdEndpoints/web"+azureAfdxTestAPI, `{"location":"global"}`), http.StatusNotFound, "ParentResourceNotFound")

	profile := sub + "/resourceGroups/rg-afd/providers/Microsoft.Cdn/profiles/edge"
	id := strings.TrimPrefix(profile, "/azure")
	azureResourcesExpectStatus(t, do(http.MethodPut, profile+azureAfdxTestAPI, `{"location":"global","sku":{"name":"Premium_AzureFrontDoor"}}`), http.StatusCreated)
	azureResourcesExpectStatus(t, do(http.MethodPut, profile+"/afdEndpoints/web"+azureAfdxTestAPI, `{"location":"global"}`), http.StatusCreated)
	azureResourcesExpectStatus(t, do(http.MethodPut, profile+"/originGroups/og"+azureAfdxTestAPI, `{"properties":{}}`), http.StatusCreated)
	azureResourcesExpectStatus(t, do(http.MethodPut, profile+"/ruleSets/rs"+azureAfdxTestAPI, `{}`), http.StatusCreated)
	cases := []struct{ name, path, body string }{
		{"endpoint scope", "/afdEndpoints/e2", `{"location":"global","properties":{"autoGeneratedDomainNameLabelScope":"Global"}}`},
		{"origin without host", "/originGroups/og/origins/o1", `{"properties":{"httpsPort":443}}`},
		{"origin bad port", "/originGroups/og/origins/o1", `{"properties":{"hostName":"a.com","httpsPort":0}}`},
		{"probe interval", "/originGroups/og2", `{"properties":{"healthProbeSettings":{"probeIntervalInSeconds":300}}}`},
		{"route unknown origin group", "/afdEndpoints/web/routes/r1", `{"properties":{"originGroup":{"id":"` + id + `/originGroups/nope"}}}`},
		{"route bad pattern", "/afdEndpoints/web/routes/r1", `{"properties":{"originGroup":{"id":"` + id + `/originGroups/og"},"patternsToMatch":["images/*"]}}`},
		{"rule without actions", "/ruleSets/rs/rules/r1", `{"properties":{"order":1,"actions":[]}}`},
		{"domain needs secret", "/customDomains/d1", `{"properties":{"hostName":"a.contoso.com","tlsSettings":{"certificateType":"CustomerCertificate"}}}`},
		{"secret without source", "/secrets/s1", `{"properties":{"parameters":{"type":"CustomerCertificate"}}}`},
		{"policy unknown domain", "/securityPolicies/p1", `{"properties":{"parameters":{"type":"WebApplicationFirewall","wafPolicy":{"id":"/x"},"associations":[{"domains":[{"id":"` + id + `/customDomains/nope"}]}]}}}`},
	}
	for _, tc := range cases {
		resp := do(http.MethodPut, profile+tc.path+azureAfdxTestAPI, tc.body)
		t.Run(tc.name, func(t *testing.T) {
			azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, "InvalidRequestContent")
		})
	}
	azureResourcesExpectErrorCode(t, do(http.MethodPut, profile+"/ruleSets/bad-name"+azureAfdxTestAPI, `{}`), http.StatusBadRequest, "InvalidResourceName")
	azureResourcesExpectErrorCode(t, do(http.MethodPost, profile+"/validateSecret"+azureAfdxTestAPI, `{"secretType":"Password"}`), http.StatusBadRequest, "InvalidRequestContent")
}
