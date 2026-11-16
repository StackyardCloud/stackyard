package server

import (
	"net/http"
	"strings"
	"testing"
)

const azureCostManagementTestAPIVersion = "?api-version=2023-11-01"

func TestAzureCostManagementLifecycle(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-cost")
	storage := strings.TrimPrefix(sub, "/azure") + "/resourceGroups/rg-cost/providers/Microsoft.Storage/storageAccounts/costexports"
	cost := sub + "/providers/Microsoft.CostManagement"

	export := cost + "/exports/monthly"
	resp := do(http.MethodPut, export+azureCostManagementTestAPIVersion, `{"properties":{"definition":{"type":"ActualCost","timeframe":"MonthToDate","dataSet":{"granularity":"Daily"}},"deliveryInfo":{"destination":{"resourceId":"`+storage+`","container":"exports","rootFolderPath":"ad-hoc"}},"schedule":{"status":"Active","recurrence":"Daily","recurrencePeriod":{"from":"2026-12-01T00:00:00Z","to":"2027-12-01T00:00:00Z"}}}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	if got := providerContractJSONMap(t, resp)["properties"].(map[string]any)["format"]; got != "Csv" {
		t.Fatalf("expected format Csv, got %v", got)
	}
	azureResourcesExpectStatus(t, do(http.MethodPost, export+"/run"+azureCostManagementTestAPIVersion, ""), http.StatusOK)
	resp = do(http.MethodGet, cost+"/exports"+azureCostManagementTestAPIVersion, "")
	if got := providerContractJSONMap(t, resp)["value"].([]any); len(got) != 1 {
		t.Fatalf("expected one export, got %d", len(got))
	}

	group := sub + "/resourceGroups/rg-cost/providers/Microsoft.CostManagement"
	azureResourcesExpectStatus(t, do(http.MethodPut, group+"/views/by-service"+azureCostManagementTestAPIVersion, `{"properties":{"displayName":"By service","chart":"StackedColumn","accumulated":"false","metric":"ActualCost","query":{"type":"Usage","timeframe":"MonthToDate","dataSet":{"granularity":"Daily","aggregation":{"totalCost":{"name":"PreTaxCost","function":"Sum"}}}}}}`), http.StatusCreated)
	action := group + "/scheduledActions/daily-mail"
	azureResourcesExpectStatus(t, do(http.MethodPut, action+azureCostManagementTestAPIVersion, `{"kind":"Email","properties":{"displayName":"Daily cost","status":"Enabled","viewId":"`+strings.TrimPrefix(group, "/azure")+`/views/by-service","notification":{"to":["ops@example.com"],"subject":"Daily cost"},"schedule":{"frequency":"Daily","startDate":"2026-12-01T08:00:00Z","endDate":"2027-12-01T08:00:00Z"}}}`), http.StatusCreated)
	azureResourcesExpectStatus(t, do(http.MethodPost, action+"/execute"+azureCostManagementTestAPIVersion, ""), http.StatusOK)

	resp = do(http.MethodPost, cost+"/query"+azureCostManagementTestAPIVersion, `{"type":"ActualCost","timeframe":"Custom","timePeriod":{"from":"2026-10-01T00:00:00Z","to":"2026-10-31T00:00:00Z"},"dataset":{"granularity":"Daily","aggregation":{"totalCost":{"name":"Cost","function":"Sum"}},"grouping":[{"type":"Dimension","name":"ServiceName"}]}}`)
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	props := providerContractJSONMap(t, resp)["properties"].(map[string]any)
	if got := azureCostColumnNames(props); got != "totalCost,UsageDate,ServiceName,Currency" || len(props["rows"].([]any)) != 0 {
		t.Fatalf("unexpected query result: %#v", props)
	}
	resp = do(http.MethodPost, group+"/forecast"+azureCostManagementTestAPIVersion, `{"type":"Usage","timeframe":"MonthToDate","dataset":{"granularity":"Monthly"}}`)
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if got := azureCostColumnNames(providerContractJSONMap(t, resp)["properties"].(map[string]any)); got != "BillingMonth,CostStatus,Currency" {
		t.Fatalf("unexpected forecast columns: %s", got)
	}
	azureResourcesExpectErrorCode(t, do(http.MethodPost, sub+"/resourceGroups/missing/providers/Microsoft.CostManagement/query"+azureCostManagementTestAPIVersion, `{"type":"Usage","timeframe":"MonthToDate"}`), http.StatusNotFound, "ResourceGroupNotFound")
}

func azureCostColumnNames(props map[string]any) string {
	var names []string
	for _, column := range props["columns"].([]any) {
		names = append(names, column.(map[string]any)["name"].(string))
	}
	return strings.Join(names, ",")
}

func TestAzureCostManagementValidation(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-cost")
	storage := strings.TrimPrefix(sub, "/azure") + "/resourceGroups/rg-cost/providers/Microsoft.Storage/storageAccounts/costexports"
	cost := sub + "/providers/Microsoft.CostManagement"
	delivery := `"deliveryInfo":{"destination":{"resourceId":"` + storage + `","container":"exports"}}`
	schedule := `"notification":{"to":["ops@example.com"],"subject":"Cost"},"schedule":{"frequency":"Daily","startDate":"2026-12-01T08:00:00Z","endDate":"2027-12-01T08:00:00Z"}`
	cases := []struct{ name, method, path, body, code string }{
		{"export type", http.MethodPut, cost + "/exports/e1", `{"properties":{"definition":{"type":"Bill","timeframe":"MonthToDate"},` + delivery + `}}`, "InvalidRequestContent"},
		{"export custom period", http.MethodPut, cost + "/exports/e1", `{"properties":{"definition":{"type":"ActualCost","timeframe":"Custom"},` + delivery + `}}`, "InvalidRequestContent"},
		{"export destination", http.MethodPut, cost + "/exports/e1", `{"properties":{"definition":{"type":"ActualCost","timeframe":"MonthToDate"},"deliveryInfo":{"destination":{"resourceId":"x","container":"exports"}}}}`, "InvalidRequestContent"},
		{"export format", http.MethodPut, cost + "/exports/e1", `{"properties":{"format":"Json","definition":{"type":"ActualCost","timeframe":"MonthToDate"},` + delivery + `}}`, "InvalidRequestContent"},
		{"export recurrence", http.MethodPut, cost + "/exports/e1", `{"properties":{"definition":{"type":"ActualCost","timeframe":"MonthToDate"},` + delivery + `,"schedule":{"recurrence":"Hourly"}}}`, "InvalidRequestContent"},
		{"export container", http.MethodPut, cost + "/exports/e1", `{"properties":{"definition":{"type":"ActualCost","timeframe":"MonthToDate"},"deliveryInfo":{"destination":{"resourceId":"` + storage + `"}}}}`, "InvalidRequestContent"},
		{"view type", http.MethodPut, cost + "/views/v1", `{"properties":{"query":{"type":"ActualCost","timeframe":"MonthToDate"}}}`, "InvalidRequestContent"},
		{"view chart", http.MethodPut, cost + "/views/v1", `{"properties":{"chart":"Pie","query":{"type":"Usage","timeframe":"MonthToDate"}}}`, "InvalidRequestContent"},
		{"action kind", http.MethodPut, cost + "/scheduledActions/a1", `{"kind":"Sms","properties":{"displayName":"Cost","viewId":"v","status":"Enabled",` + schedule + `}}`, "InvalidRequestContent"},
		{"action status", http.MethodPut, cost + "/scheduledActions/a1", `{"kind":"Email","properties":{"displayName":"Cost","viewId":"v","status":"On",` + schedule + `}}`, "InvalidRequestContent"},
		{"action recipients", http.MethodPut, cost + "/scheduledActions/a1", `{"kind":"Email","properties":{"displayName":"Cost","viewId":"v","status":"Enabled","notification":{"to":[],"subject":"Cost"}}}`, "InvalidRequestContent"},
		{"action dates", http.MethodPut, cost + "/scheduledActions/a1", `{"kind":"Email","properties":{"displayName":"Cost","viewId":"v","status":"Enabled","notification":{"to":["ops@example.com"],"subject":"Cost"},"schedule":{"frequency":"Daily","startDate":"2027-12-01T08:00:00Z","endDate":"2026-12-01T08:00:00Z"}}}`, "InvalidRequestContent"},
		{"query timeframe", http.MethodPost, cost + "/query", `{"type":"Usage","timeframe":"Yesterday"}`, "BadRequest"},
		{"query granularity", http.MethodPost, cost + "/query", `{"type":"Usage","timeframe":"MonthToDate","dataset":{"granularity":"Hourly"}}`, "BadRequest"},
		{"forecast type", http.MethodPost, cost + "/forecast", `{"type":"Budget","timeframe":"MonthToDate"}`, "BadRequest"},
		{"query aggregation", http.MethodPost, cost + "/query", `{"type":"Usage","timeframe":"MonthToDate","dataset":{"aggregation":{"total":{"name":"Cost","function":"Avg"}}}}`, "BadRequest"},
		{"query grouping", http.MethodPost, cost + "/query", `{"type":"Usage","timeframe":"MonthToDate","dataset":{"grouping":[{"type":"Dimension","name":"A"},{"type":"Dimension","name":"B"},{"type":"Dimension","name":"C"}]}}`, "BadRequest"},
	}
	for _, tc := range cases {
		resp := do(tc.method, tc.path+azureCostManagementTestAPIVersion, tc.body)
		t.Run(tc.name, func(t *testing.T) {
			azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, tc.code)
		})
	}

	// Cost Management resources attach to subscriptions and resource groups,
	// not to other resources.
	vault := sub + "/resourceGroups/rg-cost/providers/Microsoft.MachineLearningServices/workspaces/costml"
	azureResourcesExpectStatus(t, do(http.MethodPut, vault+azureMLTestAPIVersion, azureMLWorkspaceTestBody(sub)), http.StatusCreated)
	azureResourcesExpectErrorCode(t, do(http.MethodPut, vault+"/providers/Microsoft.CostManagement/views/v1"+azureCostManagementTestAPIVersion, `{"properties":{"query":{"type":"Usage","timeframe":"MonthToDate"}}}`), http.StatusBadRequest, "InvalidScope")
	azureResourcesExpectErrorCode(t, do(http.MethodPut, vault+"/providers/Microsoft.CostManagement/exports/e1"+azureCostManagementTestAPIVersion, `{"properties":{}}`), http.StatusBadRequest, "InvalidScope")
	azureResourcesExpectErrorCode(t, do(http.MethodPost, vault+"/providers/Microsoft.CostManagement/query"+azureCostManagementTestAPIVersion, `{"type":"Usage","timeframe":"MonthToDate"}`), http.StatusBadRequest, "InvalidScope")
}
