package server

import (
	"net/http"
	"strings"
	"testing"
)

const (
	azureAfdwTestAPI    = "?api-version=2025-11-01"
	azureAfdwTestPolicy = `{"location":"global","sku":{"name":"Premium_AzureFrontDoor"},"properties":{
		"policySettings":{"enabledState":"Enabled","mode":"Prevention","customBlockResponseStatusCode":403},
		"customRules":{"rules":[{"name":"BlockBadIPs","priority":1,"ruleType":"MatchRule","action":"Block",
			"matchConditions":[{"matchVariable":"RemoteAddr","operator":"IPMatch","matchValue":["192.0.2.0/24"]}]}]},
		"managedRules":{"managedRuleSets":[{"ruleSetType":"Microsoft_DefaultRuleSet","ruleSetVersion":"2.1","ruleSetAction":"Block"}]}}}`
)

func TestAzureFrontDoorWAFPolicyLifecycle(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-waf")
	policy := sub + "/resourceGroups/rg-waf/providers/Microsoft.Network/FrontDoorWebApplicationFirewallPolicies/edgepolicy"

	resp := do(http.MethodPut, policy+azureAfdwTestAPI, azureAfdwTestPolicy)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	body := providerContractJSONMap(t, resp)
	if body["properties"].(map[string]any)["resourceState"] != "Enabled" || body["etag"] == nil {
		t.Fatalf("unexpected policy: %#v", body)
	}
	resp = do(http.MethodGet, sub+"/providers/Microsoft.Network/FrontDoorWebApplicationFirewallPolicies"+azureAfdwTestAPI, "")
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if len(providerContractJSONMap(t, resp)["value"].([]any)) != 1 {
		t.Fatal("want the policy in the subscription list")
	}

	resp = do(http.MethodGet, sub+"/providers/Microsoft.Network/FrontDoorWebApplicationFirewallManagedRuleSets"+azureAfdwTestAPI, "")
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if got := len(providerContractJSONMap(t, resp)["value"].([]any)); got != len(azureAfdwManagedRuleSets) {
		t.Fatalf("listed %d managed rule sets, want %d", got, len(azureAfdwManagedRuleSets))
	}
	azureResourcesExpectStatus(t, do(http.MethodDelete, policy+azureAfdwTestAPI, ""), http.StatusOK)
}

func TestAzureFrontDoorWAFPolicyValidation(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-waf")
	base := sub + "/resourceGroups/rg-waf/providers/Microsoft.Network/FrontDoorWebApplicationFirewallPolicies/"
	azureResourcesExpectErrorCode(t, do(http.MethodPut, base+"1policy"+azureAfdwTestAPI, azureAfdwTestPolicy), http.StatusBadRequest, "InvalidResourceName")
	for name, body := range map[string]string{
		"bad mode":            strings.Replace(azureAfdwTestPolicy, `"Prevention"`, `"Enforce"`, 1),
		"bad status code":     strings.Replace(azureAfdwTestPolicy, `403`, `500`, 1),
		"no match conditions": strings.Replace(azureAfdwTestPolicy, `"matchConditions":[{"matchVariable":"RemoteAddr","operator":"IPMatch","matchValue":["192.0.2.0/24"]}]`, `"matchConditions":[]`, 1),
		"no match value":      strings.Replace(azureAfdwTestPolicy, `"matchValue":["192.0.2.0/24"]`, `"matchValue":[]`, 1),
		"unknown rule set":    strings.Replace(azureAfdwTestPolicy, `"2.1"`, `"9.9"`, 1),
		"bad sku":             strings.Replace(azureAfdwTestPolicy, `Premium_AzureFrontDoor`, `Basic`, 1),
	} {
		resp := do(http.MethodPut, base+"policy"+azureAfdwTestAPI, body)
		t.Run(name, func(t *testing.T) {
			azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, "InvalidRequestContent")
		})
	}
}

func TestAzureFrontDoorRulesEngine(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-waf")
	fd := sub + "/resourceGroups/rg-waf/providers/Microsoft.Network/frontDoors/contosofd"
	engine := `{"properties":{"rules":[{"name":"addHeader","priority":1,"action":{"requestHeaderActions":[{"headerActionType":"Append","headerName":"X-Edge","value":"1"}]},"matchProcessingBehavior":"Continue"}]}}`
	azureResourcesExpectErrorCode(t, do(http.MethodPut, fd+"/rulesEngines/engine1"+azureAfdwTestAPI, engine), http.StatusNotFound, "ParentResourceNotFound")
	azureResourcesExpectStatus(t, do(http.MethodPut, fd+azureAfdwTestAPI, azureAfdcTestBody(strings.TrimPrefix(fd, "/azure"))), http.StatusCreated)
	resp := do(http.MethodPut, fd+"/rulesEngines/engine1"+azureAfdwTestAPI, engine)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	if providerContractJSONMap(t, resp)["properties"].(map[string]any)["resourceState"] != "Enabled" {
		t.Fatal("rules engine not enabled")
	}
	azureResourcesExpectErrorCode(t, do(http.MethodPut, fd+"/rulesEngines/engine2"+azureAfdwTestAPI, `{"properties":{"rules":[{"name":"r","priority":1}]}}`), http.StatusBadRequest, "InvalidRequestContent")
}

func TestAzureNetworkExperiments(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-waf")
	profile := sub + "/resourceGroups/rg-waf/providers/Microsoft.Network/NetworkExperimentProfiles/latency"
	experiment := profile + "/Experiments/cdn-vs-afd"

	azureResourcesExpectStatus(t, do(http.MethodPut, profile+azureAfdwTestAPI, `{"location":"WestUS","properties":{"enabledState":"Enabled"}}`), http.StatusCreated)
	resp := do(http.MethodPut, experiment+azureAfdwTestAPI, `{"location":"WestUS","properties":{"endpointA":{"name":"a","endpoint":"a.contoso.com"},"endpointB":{"name":"b","endpoint":"b.contoso.com"},"enabledState":"Enabled"}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	if !strings.HasSuffix(providerContractJSONMap(t, resp)["properties"].(map[string]any)["scriptFileUri"].(string), ".js") {
		t.Fatal("experiment has no script URI")
	}
	azureResourcesExpectErrorCode(t, do(http.MethodPut, profile+"/Experiments/bad"+azureAfdwTestAPI, `{"location":"WestUS","properties":{"endpointA":{"endpoint":"a"}}}`), http.StatusBadRequest, "InvalidRequestContent")

	resp = do(http.MethodGet, profile+"/PreconfiguredEndpoints"+azureAfdwTestAPI, "")
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if len(providerContractJSONMap(t, resp)["value"].([]any)) == 0 {
		t.Fatal("no preconfigured endpoints")
	}

	resp = do(http.MethodGet, experiment+"/LatencyScorecard"+azureAfdwTestAPI+"&aggregationInterval=Daily", "")
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if providerContractJSONMap(t, resp)["properties"].(map[string]any)["endpointA"] != "a.contoso.com" {
		t.Fatal("scorecard does not name endpoint A")
	}
	azureResourcesExpectErrorCode(t, do(http.MethodGet, experiment+"/LatencyScorecard"+azureAfdwTestAPI+"&aggregationInterval=Yearly", ""), http.StatusBadRequest, "InvalidRequestContent")
	resp = do(http.MethodGet, experiment+"/Timeseries"+azureAfdwTestAPI+"&aggregationInterval=Hourly&timeseriesType=LatencyP50&startDateTimeUTC=2026-01-01T00:00:00Z&endDateTimeUTC=2026-01-01T03:00:00Z", "")
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if got := len(providerContractJSONMap(t, resp)["properties"].(map[string]any)["timeseriesData"].([]any)); got != 3 {
		t.Fatalf("timeseries has %d points, want 3", got)
	}
	azureResourcesExpectErrorCode(t, do(http.MethodGet, experiment+"/Timeseries"+azureAfdwTestAPI+"&aggregationInterval=Hourly&timeseriesType=LatencyP50", ""), http.StatusBadRequest, "InvalidRequestContent")
	azureResourcesExpectStatus(t, do(http.MethodGet, profile+"/Experiments/missing/LatencyScorecard"+azureAfdwTestAPI+"&aggregationInterval=Daily", ""), http.StatusNotFound)
}
