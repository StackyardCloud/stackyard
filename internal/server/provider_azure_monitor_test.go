package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func newAzureMonitorTestServer(t *testing.T) (*httptest.Server, string) {
	t.Helper()
	ts := newAzureResourcesTestServer(t)
	sub := "/azure/subscriptions/" + azureResourcesTestSubscription
	azureResourcesExpectStatus(t, providerContractRequest(t, ts, http.MethodPut, sub+"/resourcegroups/rg-mon?api-version=2021-04-01", []byte(`{"location":"eastus"}`), azureResourcesTestHeaders()), http.StatusCreated)
	return ts, sub
}

func TestAzureMonitorAlertingLifecycle(t *testing.T) {
	t.Parallel()
	ts, sub := newAzureMonitorTestServer(t)
	headers := azureResourcesTestHeaders()
	insights := sub + "/resourceGroups/rg-mon/providers/microsoft.insights/"
	scope := `"/subscriptions/` + azureResourcesTestSubscription + `/resourceGroups/rg-mon"`

	resources := []struct{ path, body, typ string }{
		{"actionGroups/oncall?api-version=2023-01-01", `{"location":"Global","properties":{"groupShortName":"oncall","enabled":true,"emailReceivers":[{"name":"ops","emailAddress":"ops@example.test"}]}}`, "Microsoft.Insights/actionGroups"},
		{"metricAlerts/cpu-high?api-version=2018-03-01", `{"location":"global","properties":{"severity":2,"enabled":true,"scopes":[` + scope + `],"evaluationFrequency":"PT1M","windowSize":"PT5M","criteria":{"odata.type":"Microsoft.Azure.Monitor.SingleResourceMultipleMetricCriteria","allOf":[]}}}`, "Microsoft.Insights/metricAlerts"},
		{"activityLogAlerts/deletes?api-version=2020-10-01", `{"location":"Global","properties":{"scopes":[` + scope + `],"condition":{"allOf":[{"field":"operationName","equals":"Microsoft.Resources/subscriptions/resourceGroups/delete"}]}}}`, "Microsoft.Insights/activityLogAlerts"},
		{"scheduledQueryRules/errors?api-version=2023-12-01", `{"location":"eastus","properties":{"scopes":[` + scope + `],"criteria":{"allOf":[{"query":"AppTraces | where SeverityLevel > 3","timeAggregation":"Count"}]}}}`, "Microsoft.Insights/scheduledQueryRules"},
		{"autoscalesettings/web-scale?api-version=2022-10-01", `{"location":"eastus","properties":{"targetResourceUri":"/subscriptions/x/resourceGroups/y/providers/Microsoft.Web/serverfarms/z","profiles":[{"name":"default","capacity":{"minimum":"1","maximum":"5","default":"1"},"rules":[]}]}}`, "Microsoft.Insights/autoscalesettings"},
	}
	for _, res := range resources {
		resp := providerContractRequest(t, ts, http.MethodPut, insights+res.path, []byte(res.body), headers)
		azureResourcesExpectStatus(t, resp, http.StatusCreated)
		if got := providerContractJSONMap(t, resp); got["type"] != res.typ {
			t.Fatalf("unexpected type for %s: %#v", res.path, got)
		}
	}
	resp := providerContractRequest(t, ts, http.MethodGet, sub+"/providers/Microsoft.Insights/metricAlerts?api-version=2018-03-01", nil, headers)
	if items, _ := providerContractJSONMap(t, resp)["value"].([]any); len(items) != 1 {
		t.Fatalf("expected one metric alert in the subscription, got %#v", items)
	}

	azureResourcesExpectStatus(t, providerContractRequest(t, ts, http.MethodDelete, sub+"/resourcegroups/rg-mon?api-version=2021-04-01", nil, headers), http.StatusOK)
	azureResourcesExpectErrorCode(t, providerContractRequest(t, ts, http.MethodGet, insights+"actionGroups/oncall?api-version=2023-01-01", nil, headers), http.StatusNotFound, "ResourceGroupNotFound")
}

func TestAzureMonitorDiagnosticSettingsAtEveryScope(t *testing.T) {
	t.Parallel()
	ts, sub := newAzureMonitorTestServer(t)
	headers := azureResourcesTestHeaders()
	body := []byte(`{"properties":{"workspaceId":"/subscriptions/x/resourceGroups/y/providers/Microsoft.OperationalInsights/workspaces/z","logs":[{"categoryGroup":"allLogs","enabled":true}]}}`)
	const version = "?api-version=2021-05-01-preview"

	// The target resource belongs to the shared handler, so the setting
	// should be deleted with it.
	registry := sub + "/resourceGroups/rg-mon/providers/Microsoft.ContainerRegistry/registries/acrmon01"
	azureResourcesExpectStatus(t, providerContractRequest(t, ts, http.MethodPut, registry+"?api-version=2023-07-01", []byte(`{"location":"eastus","sku":{"name":"Basic"}}`), headers), http.StatusCreated)
	scopes := map[string]string{
		"subscription":   sub,
		"resource group": sub + "/resourceGroups/rg-mon",
		"resource":       registry,
		// A resource from a service with its own store is a valid scope too.
		"external resource": sub + "/resourceGroups/rg-mon/providers/Microsoft.Storage/storageAccounts/logs01",
	}
	for label, scope := range scopes {
		setting := scope + "/providers/Microsoft.Insights/diagnosticSettings/to-workspace"
		resp := providerContractRequest(t, ts, http.MethodPut, setting+version, body, headers)
		azureResourcesExpectStatus(t, resp, http.StatusCreated)
		got := providerContractJSONMap(t, resp)
		if got["id"] != setting[len("/azure"):] || got["type"] != "Microsoft.Insights/diagnosticSettings" {
			t.Fatalf("unexpected %s setting: %#v", label, got)
		}
		if _, has := got["location"]; has {
			t.Fatalf("diagnostic settings carry no location: %#v", got)
		}
		resp = providerContractRequest(t, ts, http.MethodGet, scope+"/providers/Microsoft.Insights/diagnosticSettings"+version, nil, headers)
		// Each scope lists only its own setting, not those of resources under it.
		if items, _ := providerContractJSONMap(t, resp)["value"].([]any); len(items) != 1 {
			t.Fatalf("expected one setting at %s scope, got %#v", label, items)
		}
	}
	resp := providerContractRequest(t, ts, http.MethodGet, sub+"/resourcegroups/rg-mon/resources?api-version=2021-04-01", nil, headers)
	if items, _ := providerContractJSONMap(t, resp)["value"].([]any); len(items) != 1 {
		t.Fatalf("expected only the registry at group scope, got %#v", items)
	}

	azureResourcesExpectStatus(t, providerContractRequest(t, ts, http.MethodDelete, registry+"?api-version=2023-07-01", nil, headers), http.StatusOK)
	azureResourcesExpectErrorCode(t, providerContractRequest(t, ts, http.MethodGet, registry+"/providers/Microsoft.Insights/diagnosticSettings/to-workspace"+version, nil, headers), http.StatusNotFound, "ResourceNotFound")
	azureResourcesExpectStatus(t, providerContractRequest(t, ts, http.MethodGet, sub+"/providers/Microsoft.Insights/diagnosticSettings/to-workspace"+version, nil, headers), http.StatusOK)
}

func TestAzureMonitorValidation(t *testing.T) {
	t.Parallel()
	ts, sub := newAzureMonitorTestServer(t)
	headers := azureResourcesTestHeaders()
	insights := sub + "/resourceGroups/rg-mon/providers/Microsoft.Insights/"
	scope := `"/subscriptions/x"`
	cases := []struct{ name, path, body string }{
		{"long short name", "actionGroups/ag", `{"location":"Global","properties":{"groupShortName":"thirteen-char","enabled":true}}`},
		{"action group without enabled", "actionGroups/ag", `{"location":"Global","properties":{"groupShortName":"ops"}}`},
		{"severity 5", "metricAlerts/ma", `{"location":"global","properties":{"severity":5,"enabled":true,"scopes":[` + scope + `],"evaluationFrequency":"PT1M","windowSize":"PT5M","criteria":{"odata.type":"x"}}}`},
		{"window not a duration", "metricAlerts/ma", `{"location":"global","properties":{"severity":1,"enabled":true,"scopes":[` + scope + `],"evaluationFrequency":"PT1M","windowSize":"5 minutes","criteria":{"odata.type":"x"}}}`},
		{"empty duration", "metricAlerts/ma", `{"location":"global","properties":{"severity":1,"enabled":true,"scopes":[` + scope + `],"evaluationFrequency":"PT","windowSize":"PT5M","criteria":{"odata.type":"x"}}}`},
		{"metric alert without scopes", "metricAlerts/ma", `{"location":"global","properties":{"severity":1,"enabled":true,"scopes":[],"evaluationFrequency":"PT1M","windowSize":"PT5M","criteria":{"odata.type":"x"}}}`},
		{"metric alert without criteria", "metricAlerts/ma", `{"location":"global","properties":{"severity":1,"enabled":true,"scopes":[` + scope + `],"evaluationFrequency":"PT1M","windowSize":"PT5M"}}`},
		{"activity alert without conditions", "activityLogAlerts/al", `{"location":"Global","properties":{"scopes":[` + scope + `],"condition":{"allOf":[]}}}`},
		{"query rule without criteria", "scheduledQueryRules/qr", `{"location":"eastus","properties":{"scopes":[` + scope + `]}}`},
		{"autoscale without profiles", "autoscalesettings/as", `{"location":"eastus","properties":{"profiles":[]}}`},
	}
	for _, tc := range cases {
		resp := providerContractRequest(t, ts, http.MethodPut, insights+tc.path+"?api-version=2023-01-01", []byte(tc.body), headers)
		t.Run(tc.name, func(t *testing.T) {
			azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, "InvalidRequestContent")
		})
	}
	resp := providerContractRequest(t, ts, http.MethodPut, sub+"/providers/Microsoft.Insights/diagnosticSettings/nowhere?api-version=2021-05-01-preview", []byte(`{"properties":{"logs":[]}}`), headers)
	azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, "InvalidRequestContent")
	resp = providerContractRequest(t, ts, http.MethodPut, sub+"/resourceGroups/rg-missing/providers/Microsoft.Insights/diagnosticSettings/ds?api-version=2021-05-01-preview", []byte(`{"properties":{"workspaceId":"/w"}}`), headers)
	azureResourcesExpectErrorCode(t, resp, http.StatusNotFound, "ResourceGroupNotFound")
}
