package server

import (
	"net/http"
	"testing"
)

const (
	azureAlertProcessingTestAPIVersion = "?api-version=2021-08-08"
	azurePrometheusRulesTestAPIVersion = "?api-version=2023-03-01"
)

func TestAzureAlertProcessingRules(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-alerts")
	rule := sub + "/resourceGroups/rg-alerts/providers/Microsoft.AlertsManagement/actionRules/"
	scope := `"scopes":["/subscriptions/` + azureResourcesTestSubscription + `"]`

	resp := do(http.MethodPut, rule+"weekend-quiet"+azureAlertProcessingTestAPIVersion, `{"location":"Global","properties":{`+scope+`,"actions":[{"actionType":"RemoveAllActionGroups"}],"schedule":{"recurrences":[{"recurrenceType":"Weekly","daysOfWeek":["Saturday","Sunday"]}]}}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	if props := providerContractJSONMap(t, resp)["properties"].(map[string]any); props["enabled"] != true {
		t.Fatalf("expected the rule enabled by default, got %#v", props)
	}
	resp = do(http.MethodPatch, rule+"weekend-quiet"+azureAlertProcessingTestAPIVersion, `{"properties":{"enabled":false}}`)
	if props := providerContractJSONMap(t, resp)["properties"].(map[string]any); props["enabled"] != false {
		t.Fatalf("PATCH did not disable the rule: %#v", props)
	}

	cases := []struct{ name, body string }{
		{"no scopes", `{"location":"Global","properties":{"actions":[{"actionType":"RemoveAllActionGroups"}]}}`},
		{"no actions", `{"location":"Global","properties":{` + scope + `}}`},
		{"bad action", `{"location":"Global","properties":{` + scope + `,"actions":[{"actionType":"Page"}]}}`},
		{"add without groups", `{"location":"Global","properties":{` + scope + `,"actions":[{"actionType":"AddActionGroups"}]}}`},
	}
	for _, tc := range cases {
		resp := do(http.MethodPut, rule+"bad"+azureAlertProcessingTestAPIVersion, tc.body)
		t.Run(tc.name, func(t *testing.T) {
			azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, "InvalidRequestContent")
		})
	}

	resp = do(http.MethodGet, sub+"/providers/Microsoft.AlertsManagement/alerts?api-version=2019-05-05-preview", "")
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if items, ok := providerContractJSONMap(t, resp)["value"].([]any); !ok || len(items) != 0 {
		t.Fatalf("expected no fired alerts, got %#v", items)
	}
}

func TestAzurePrometheusRuleGroups(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-alerts")
	group := sub + "/resourceGroups/rg-alerts/providers/Microsoft.AlertsManagement/prometheusRuleGroups/node-rules"
	scope := `"scopes":["/subscriptions/` + azureResourcesTestSubscription + `/resourceGroups/rg-alerts/providers/Microsoft.Monitor/accounts/prom"]`

	resp := do(http.MethodPut, group+azurePrometheusRulesTestAPIVersion, `{"location":"eastus","properties":{`+scope+`,"interval":"PT1M","rules":[{"record":"job:cpu:avg","expression":"avg(rate(cpu[5m]))"},{"alert":"HighCPU","expression":"job:cpu:avg > 0.9","severity":3,"for":"PT5M"}]}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	if props := providerContractJSONMap(t, resp)["properties"].(map[string]any); props["enabled"] != true || len(props["rules"].([]any)) != 2 {
		t.Fatalf("unexpected rule group: %#v", props)
	}

	cases := []struct{ name, body string }{
		{"no scopes", `{"location":"eastus","properties":{"rules":[{"record":"r","expression":"up"}]}}`},
		{"no rules", `{"location":"eastus","properties":{` + scope + `}}`},
		{"record and alert", `{"location":"eastus","properties":{` + scope + `,"rules":[{"record":"r","alert":"a","expression":"up"}]}}`},
		{"no expression", `{"location":"eastus","properties":{` + scope + `,"rules":[{"alert":"a"}]}}`},
	}
	for _, tc := range cases {
		resp := do(http.MethodPut, group+azurePrometheusRulesTestAPIVersion, tc.body)
		t.Run(tc.name, func(t *testing.T) {
			azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, "InvalidRequestContent")
		})
	}
}
