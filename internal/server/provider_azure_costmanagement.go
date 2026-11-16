package server

import (
	"net/http"
	"slices"
	"sort"
	"strings"
	"time"
)

// Microsoft.CostManagement exports, saved views and scheduled actions,
// served by the shared ARM handler as extension resources on subscriptions
// and resource groups, plus the query and forecast APIs. Stackyard bills
// nothing, so queries return their columns with no rows, exports write no
// files and scheduled actions send no email.
func init() {
	const ns = "Microsoft.CostManagement"
	ok := func(*azureARMResource, map[string]any) (int, any) { return http.StatusOK, nil }
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "exports", Extension: true,
		ValidateScope:      azureCostManagementScope,
		Validate:           azureCostExportValidate,
		ReadOnlyProperties: azureCostExportProperties,
		Actions:            map[string]azureARMAction{"run": ok},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "views", Extension: true,
		ValidateScope: azureCostManagementScope,
		Validate:      azureCostViewValidate,
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "scheduledActions", Extension: true,
		ValidateScope: azureCostManagementScope,
		Validate:      azureCostScheduledActionValidate,
		Actions:       map[string]azureARMAction{"execute": ok},
	})
	for _, path := range []string{"query", "forecast"} {
		registerAzureARMScopeAction(ns, path, func(_ *Server, scope string, payload map[string]any) (int, any) {
			if msg := azureCostManagementScope(scope); msg != "" {
				return http.StatusBadRequest, azureARMErrorBody("InvalidScope", msg)
			}
			if msg := azureCostQueryValidate(payload, azureCostQueryTypes); msg != "" {
				return http.StatusBadRequest, azureARMErrorBody("BadRequest", msg)
			}
			return http.StatusOK, azureCostQueryResult(scope, path, payload)
		})
	}
}

var (
	azureCostQueryTypes      = []string{"Usage", "ActualCost", "AmortizedCost"}
	azureCostExportTypes     = []string{"Usage", "ActualCost", "AmortizedCost", "FocusCost", "PriceSheet", "ReservationDetails", "ReservationRecommendations", "ReservationTransactions"}
	azureCostTimeframes      = []string{"MonthToDate", "BillingMonthToDate", "TheLastMonth", "TheLastBillingMonth", "WeekToDate", "TheCurrentMonth", "Custom"}
	azureCostGranularities   = []string{"Daily", "Monthly"}
	azureCostScheduleStatus  = []string{"Enabled", "Disabled", "Expired"}
	azureCostFrequencies     = []string{"Daily", "Weekly", "Monthly"}
	azureCostExportRecurring = []string{"Daily", "Weekly", "Monthly", "Annually"}
)

// azureCostManagementScope accepts a subscription or resource group scope.
func azureCostManagementScope(scope string) string {
	seg := strings.Split(strings.Trim(scope, "/"), "/")
	if len(seg) == 2 || (len(seg) == 4 && strings.EqualFold(seg[2], "resourceGroups")) {
		return ""
	}
	return "Cost Management resources attach only to subscriptions and resource groups."
}

// azureCostQueryValidate checks a query definition: its type, timeframe,
// custom time period and dataset. Query requests spell the dataset
// "dataset"; exports and views spell it "dataSet".
func azureCostQueryValidate(def map[string]any, types []string) string {
	if !slices.Contains(types, azureARMString(def["type"])) {
		return "type must be one of " + strings.Join(types, ", ") + "."
	}
	timeframe := azureARMString(def["timeframe"])
	if !slices.Contains(azureCostTimeframes, timeframe) {
		return "timeframe must be one of " + strings.Join(azureCostTimeframes, ", ") + "."
	}
	if timeframe == "Custom" {
		period := azureARMObject(def["timePeriod"])
		from, errFrom := time.Parse(time.RFC3339, azureARMString(period["from"]))
		to, errTo := time.Parse(time.RFC3339, azureARMString(period["to"]))
		if errFrom != nil || errTo != nil || to.Before(from) {
			return "A Custom timeframe needs timePeriod.from and timePeriod.to as RFC 3339 times, from first."
		}
	}
	dataset := azureCostDataset(def)
	if g, present := dataset["granularity"]; present && !slices.Contains(azureCostGranularities, azureARMString(g)) {
		return "dataset.granularity must be Daily or Monthly."
	}
	for name, raw := range azureARMObject(dataset["aggregation"]) {
		if agg := azureARMObject(raw); azureARMString(agg["name"]) == "" || azureARMString(agg["function"]) != "Sum" {
			return "dataset.aggregation." + name + " needs a column name and function Sum."
		}
	}
	grouping, _ := dataset["grouping"].([]any)
	if len(grouping) > 2 {
		return "dataset.grouping allows at most 2 groupings."
	}
	for _, raw := range grouping {
		group := azureARMObject(raw)
		if !slices.Contains([]string{"Dimension", "TagKey"}, azureARMString(group["type"])) || azureARMString(group["name"]) == "" {
			return "Each dataset.grouping entry needs type Dimension or TagKey and a name."
		}
	}
	return ""
}

func azureCostDataset(def map[string]any) map[string]any {
	if dataset, ok := def["dataset"].(map[string]any); ok {
		return dataset
	}
	return azureARMObject(def["dataSet"])
}

// azureCostQueryResult returns the columns the query asks for, aggregations
// then the date then groupings then the currency, and no rows.
func azureCostQueryResult(scope, path string, payload map[string]any) map[string]any {
	dataset := azureCostDataset(payload)
	aggregations := azureARMObject(dataset["aggregation"])
	names := make([]string, 0, len(aggregations))
	for name := range aggregations {
		names = append(names, name)
	}
	sort.Strings(names)
	columns := []any{}
	for _, name := range names {
		columns = append(columns, map[string]any{"name": name, "type": "Number"})
	}
	switch azureARMString(dataset["granularity"]) {
	case "Daily":
		columns = append(columns, map[string]any{"name": "UsageDate", "type": "Number"})
	case "Monthly":
		columns = append(columns, map[string]any{"name": "BillingMonth", "type": "Datetime"})
	}
	grouping, _ := dataset["grouping"].([]any)
	for _, raw := range grouping {
		columns = append(columns, map[string]any{"name": azureARMString(azureARMObject(raw)["name"]), "type": "String"})
	}
	if path == "forecast" {
		columns = append(columns, map[string]any{"name": "CostStatus", "type": "String"})
	}
	columns = append(columns, map[string]any{"name": "Currency", "type": "String"})
	return map[string]any{
		"id":         scope + "/providers/Microsoft.CostManagement/" + path,
		"name":       path,
		"type":       "Microsoft.CostManagement/" + path,
		"properties": map[string]any{"nextLink": nil, "columns": columns, "rows": []any{}},
	}
}

func azureCostExportValidate(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	if msg := azureCostQueryValidate(azureARMObject(props["definition"]), azureCostExportTypes); msg != "" {
		return "properties.definition." + msg
	}
	destination := azureARMObject(azureARMObject(props["deliveryInfo"])["destination"])
	if !strings.Contains(strings.ToLower(azureARMString(destination["resourceId"])), "/providers/microsoft.storage/storageaccounts/") {
		return "properties.deliveryInfo.destination.resourceId must be a storage account ID."
	}
	if strings.TrimSpace(azureARMString(destination["container"])) == "" {
		return "properties.deliveryInfo.destination.container is required."
	}
	if msg := azureEnumProperty("format", "Csv", "Parquet")(payload); msg != "" {
		return msg
	}
	schedule, present := props["schedule"]
	if !present {
		return ""
	}
	sched := azureARMObject(schedule)
	if status, present := sched["status"]; present && !slices.Contains([]string{"Active", "Inactive"}, azureARMString(status)) {
		return "properties.schedule.status must be Active or Inactive."
	}
	if recurrence, present := sched["recurrence"]; present {
		if !slices.Contains(azureCostExportRecurring, azureARMString(recurrence)) {
			return "properties.schedule.recurrence must be one of " + strings.Join(azureCostExportRecurring, ", ") + "."
		}
		if _, err := time.Parse(time.RFC3339, azureARMString(azureARMObject(sched["recurrencePeriod"])["from"])); err != nil {
			return "properties.schedule.recurrencePeriod.from must be an RFC 3339 time."
		}
	}
	return ""
}

func azureCostExportProperties(rec *azureARMResource) map[string]any {
	if _, present := rec.Properties["format"]; present {
		return nil
	}
	return map[string]any{"format": "Csv"}
}

func azureCostViewValidate(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	if msg := azureCostQueryValidate(azureARMObject(props["query"]), []string{"Usage"}); msg != "" {
		return "properties.query." + msg
	}
	for key, values := range map[string][]string{
		"chart":       {"Area", "Line", "StackedColumn", "GroupedColumn", "Table"},
		"accumulated": {"true", "false"},
		"metric":      {"ActualCost", "AmortizedCost", "AHUB"},
	} {
		if msg := azureEnumProperty(key, values...)(payload); msg != "" {
			return msg
		}
	}
	return ""
}

func azureCostScheduledActionValidate(payload map[string]any) string {
	if kind, present := payload["kind"]; present && !slices.Contains([]string{"Email", "InsightAlert"}, azureARMString(kind)) {
		return "kind must be Email or InsightAlert."
	}
	props := azureARMObject(payload["properties"])
	for _, key := range []string{"displayName", "viewId"} {
		if strings.TrimSpace(azureARMString(props[key])) == "" {
			return "properties." + key + " is required."
		}
	}
	if !slices.Contains(azureCostScheduleStatus, azureARMString(props["status"])) {
		return "properties.status must be one of " + strings.Join(azureCostScheduleStatus, ", ") + "."
	}
	notification := azureARMObject(props["notification"])
	if to, _ := notification["to"].([]any); len(to) == 0 || strings.TrimSpace(azureARMString(notification["subject"])) == "" {
		return "properties.notification needs at least one recipient in to and a subject."
	}
	schedule := azureARMObject(props["schedule"])
	if !slices.Contains(azureCostFrequencies, azureARMString(schedule["frequency"])) {
		return "properties.schedule.frequency must be Daily, Weekly or Monthly."
	}
	start, errStart := time.Parse(time.RFC3339, azureARMString(schedule["startDate"]))
	end, errEnd := time.Parse(time.RFC3339, azureARMString(schedule["endDate"]))
	if errStart != nil || errEnd != nil || !end.After(start) {
		return "properties.schedule needs startDate and a later endDate as RFC 3339 times."
	}
	return ""
}
