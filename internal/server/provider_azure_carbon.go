package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

const azureCarbonAPIVersion = "2025-04-01"

type azureCarbonEmissionRow struct {
	SubscriptionID   string
	Month            time.Time
	CarbonScope      string
	Location         string
	ResourceGroupURL string
	ResourceType     string
	ResourceID       string
	ResourceName     string
	Category         string
	KgCO2e           float64
}

type azureCarbonReportRequest struct {
	ReportType       string
	SubscriptionList []string
	CarbonScopeList  []string
	StartDate        time.Time
	EndDate          time.Time
	PageSize         int
	SkipOffset       int
	TopItems         int
	LocationList     []string
	ResourceGroupURL []string
	ResourceTypeList []string
}

func (s *Server) handleAzureCarbonRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, "/azure/providers/") {
		return false
	}

	segments := splitPathSegments(strings.TrimPrefix(path, "/azure/"))
	if len(segments) < 2 || !strings.EqualFold(segments[0], "providers") || !strings.EqualFold(segments[1], "Microsoft.Carbon") {
		return false
	}
	if hasAzureInvalidAPIVersion(r) {
		respondAzureInvalidRequest(w, path, "invalid api-version query parameter")
		return true
	}
	if !azureCarbonAllowedMethod(r.Method) {
		return false
	}

	tail := segments[2:]
	if len(tail) == 1 && strings.EqualFold(tail[0], "operations") {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return true
		}
		respondJSON(w, http.StatusOK, map[string]any{"value": azureCarbonOperationsFixture()})
		return true
	}
	if len(tail) == 1 && strings.EqualFold(tail[0], "carbonEmissionReports") {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return true
		}
		s.handleAzureCarbonQueryReports(w, r, path)
		return true
	}
	if len(tail) == 1 && strings.EqualFold(tail[0], "queryCarbonEmissionDataAvailableDateRange") {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return true
		}
		s.handleAzureCarbonQueryAvailableDateRange(w, r, path)
		return true
	}

	respondAzureImplemented(w, path)
	return true
}

func azureCarbonAllowedMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodPost:
		return true
	default:
		return false
	}
}

func (s *Server) handleAzureCarbonQueryAvailableDateRange(w http.ResponseWriter, r *http.Request, path string) {
	if _, _, ok := azureCarbonReadRequestBody(w, r, path); !ok {
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"startDate": "2024-01-01",
		"endDate":   "2026-03-31",
	})
}

func (s *Server) handleAzureCarbonQueryReports(w http.ResponseWriter, r *http.Request, path string) {
	_, payload, ok := azureCarbonReadRequestBody(w, r, path)
	if !ok {
		return
	}

	req, err := azureCarbonParseReportRequest(payload)
	if err != nil {
		respondAzureInvalidRequest(w, path, err.Error())
		return
	}

	rows := azureCarbonFilterRows(req)
	switch req.ReportType {
	case "Overall Summary Report":
		respondJSON(w, http.StatusOK, azureCarbonOverallSummaryResponse(req, rows))
	case "Monthly Summary Report":
		respondJSON(w, http.StatusOK, azureCarbonMonthlySummaryResponse(req, rows))
	case "Item Details Report":
		respondJSON(w, http.StatusOK, azureCarbonItemDetailsResponse(req, rows))
	case "Top Items Summary Report":
		respondJSON(w, http.StatusOK, azureCarbonTopItemsSummaryResponse(req, rows))
	case "Top Items Monthly Summary Report":
		respondJSON(w, http.StatusOK, azureCarbonTopItemsMonthlySummaryResponse(req, rows))
	default:
		respondAzureInvalidRequest(w, path, "unsupported reportType")
	}
}

func azureCarbonParseReportRequest(payload map[string]any) (azureCarbonReportRequest, error) {
	req := azureCarbonReportRequest{
		ReportType:       strings.TrimSpace(azureCarbonString(payload["reportType"])),
		SubscriptionList: azureCarbonUniqueStrings(payload["subscriptionList"]),
		CarbonScopeList:  azureCarbonUniqueStrings(payload["carbonScopeList"]),
		LocationList:     azureCarbonUniqueStrings(payload["locationList"]),
		ResourceGroupURL: azureCarbonUniqueStrings(payload["resourceGroupUrlList"]),
		ResourceTypeList: azureCarbonUniqueStrings(payload["resourceTypeList"]),
		PageSize:         100,
		TopItems:         5,
	}
	if req.ReportType == "" {
		return req, fmt.Errorf("reportType is required")
	}
	if len(req.SubscriptionList) == 0 {
		return req, fmt.Errorf("subscriptionList is required")
	}
	if len(req.SubscriptionList) > 100 {
		return req, fmt.Errorf("subscriptionList must contain at most 100 entries")
	}
	if len(req.CarbonScopeList) == 0 {
		return req, fmt.Errorf("carbonScopeList is required")
	}

	dateRange := azureCarbonObject(payload["dateRange"])
	if len(dateRange) == 0 {
		return req, fmt.Errorf("dateRange is required")
	}
	startDate, err := azureCarbonParseDate(azureCarbonString(dateRange["start"]))
	if err != nil {
		startDate, err = azureCarbonParseDate(azureCarbonString(dateRange["startDate"]))
		if err != nil {
			return req, fmt.Errorf("dateRange.startDate is required in YYYY-MM-DD format")
		}
	}
	endDate, err := azureCarbonParseDate(azureCarbonString(dateRange["end"]))
	if err != nil {
		endDate, err = azureCarbonParseDate(azureCarbonString(dateRange["endDate"]))
		if err != nil {
			return req, fmt.Errorf("dateRange.endDate is required in YYYY-MM-DD format")
		}
	}
	if endDate.Before(startDate) {
		return req, fmt.Errorf("dateRange end must be on or after start")
	}
	req.StartDate = startDate
	req.EndDate = endDate

	if rawPageSize := payload["pageSize"]; rawPageSize != nil {
		pageSize, ok := azureCarbonInt(rawPageSize)
		if !ok {
			return req, fmt.Errorf("pageSize must be an integer")
		}
		if pageSize < 1 || pageSize > 5000 {
			return req, fmt.Errorf("pageSize must be between 1 and 5000")
		}
		req.PageSize = pageSize
	}
	if rawTopItems := payload["topItems"]; rawTopItems != nil {
		topItems, ok := azureCarbonInt(rawTopItems)
		if !ok {
			return req, fmt.Errorf("topItems must be an integer")
		}
		if topItems < 1 || topItems > 10 {
			return req, fmt.Errorf("topItems must be between 1 and 10")
		}
		req.TopItems = topItems
	}
	if rawSkipToken := strings.TrimSpace(azureCarbonString(payload["skipToken"])); rawSkipToken != "" {
		offset, err := azureCarbonParseSkipToken(rawSkipToken)
		if err != nil {
			return req, err
		}
		req.SkipOffset = offset
	}

	switch req.ReportType {
	case "Overall Summary Report", "Monthly Summary Report", "Item Details Report", "Top Items Summary Report", "Top Items Monthly Summary Report":
	default:
		return req, fmt.Errorf("reportType is unsupported")
	}

	if (req.ReportType == "Item Details Report" || req.ReportType == "Top Items Summary Report") &&
		req.StartDate.Format("2006-01") != req.EndDate.Format("2006-01") {
		return req, fmt.Errorf("%s supports only one month per request", req.ReportType)
	}

	return req, nil
}

func azureCarbonFilterRows(req azureCarbonReportRequest) []azureCarbonEmissionRow {
	subscriptions := azureCarbonMembershipSet(req.SubscriptionList)
	scopes := azureCarbonMembershipSet(req.CarbonScopeList)
	locations := azureCarbonMembershipSet(req.LocationList)
	resourceGroups := azureCarbonMembershipSet(req.ResourceGroupURL)
	resourceTypes := azureCarbonMembershipSet(req.ResourceTypeList)

	filtered := make([]azureCarbonEmissionRow, 0, len(azureCarbonSeedRows))
	for _, row := range azureCarbonSeedRows {
		if !subscriptions[strings.ToLower(row.SubscriptionID)] {
			continue
		}
		if !scopes[strings.ToLower(row.CarbonScope)] {
			continue
		}
		if len(locations) > 0 && !locations[strings.ToLower(row.Location)] {
			continue
		}
		if len(resourceGroups) > 0 && !resourceGroups[strings.ToLower(row.ResourceGroupURL)] {
			continue
		}
		if len(resourceTypes) > 0 && !resourceTypes[strings.ToLower(row.ResourceType)] {
			continue
		}
		monthDate := row.Month.UTC()
		if monthDate.Before(azureCarbonMonthStart(req.StartDate)) || monthDate.After(azureCarbonMonthStart(req.EndDate)) {
			continue
		}
		filtered = append(filtered, row)
	}

	sort.Slice(filtered, func(i, j int) bool {
		if !filtered[i].Month.Equal(filtered[j].Month) {
			return filtered[i].Month.Before(filtered[j].Month)
		}
		if filtered[i].SubscriptionID != filtered[j].SubscriptionID {
			return filtered[i].SubscriptionID < filtered[j].SubscriptionID
		}
		if filtered[i].ResourceID != filtered[j].ResourceID {
			return filtered[i].ResourceID < filtered[j].ResourceID
		}
		return filtered[i].CarbonScope < filtered[j].CarbonScope
	})
	return filtered
}

func azureCarbonOverallSummaryResponse(req azureCarbonReportRequest, rows []azureCarbonEmissionRow) map[string]any {
	total := 0.0
	scopeTotals := map[string]float64{}
	for _, row := range rows {
		total += row.KgCO2e
		scopeTotals[row.CarbonScope] += row.KgCO2e
	}

	scopeKeys := make([]string, 0, len(scopeTotals))
	for key := range scopeTotals {
		scopeKeys = append(scopeKeys, key)
	}
	sort.Strings(scopeKeys)

	scopeValues := make([]map[string]any, 0, len(scopeKeys))
	for _, key := range scopeKeys {
		scopeValues = append(scopeValues, map[string]any{
			"carbonScope":         key,
			"totalCarbonEmission": azureCarbonRound(scopeTotals[key]),
			"unit":                "kgCO2e",
		})
	}

	return map[string]any{
		"value": []map[string]any{{
			"reportType":           req.ReportType,
			"unit":                 "kgCO2e",
			"totalCarbonEmission":  azureCarbonRound(total),
			"subscriptionCount":    len(req.SubscriptionList),
			"carbonScopeBreakdown": scopeValues,
			"dateRange": map[string]any{
				"startDate": req.StartDate.Format("2006-01-02"),
				"endDate":   req.EndDate.Format("2006-01-02"),
			},
		}},
		"count": 1,
	}
}

func azureCarbonMonthlySummaryResponse(req azureCarbonReportRequest, rows []azureCarbonEmissionRow) map[string]any {
	monthlyTotals := map[string]float64{}
	for _, row := range rows {
		month := row.Month.Format("2006-01")
		monthlyTotals[month] += row.KgCO2e
	}

	months := make([]string, 0, len(monthlyTotals))
	for key := range monthlyTotals {
		months = append(months, key)
	}
	sort.Strings(months)

	value := make([]map[string]any, 0, len(months))
	for _, month := range months {
		value = append(value, map[string]any{
			"month":               month,
			"totalCarbonEmission": azureCarbonRound(monthlyTotals[month]),
			"unit":                "kgCO2e",
		})
	}
	return map[string]any{
		"value": value,
		"count": len(value),
	}
}

func azureCarbonItemDetailsResponse(req azureCarbonReportRequest, rows []azureCarbonEmissionRow) map[string]any {
	count := len(rows)
	start := req.SkipOffset
	if start > count {
		start = count
	}
	end := start + req.PageSize
	if end > count {
		end = count
	}

	value := make([]map[string]any, 0, end-start)
	for _, row := range rows[start:end] {
		value = append(value, map[string]any{
			"month":               row.Month.Format("2006-01"),
			"subscriptionId":      row.SubscriptionID,
			"carbonScope":         row.CarbonScope,
			"location":            row.Location,
			"resourceGroupUrl":    row.ResourceGroupURL,
			"resourceType":        row.ResourceType,
			"resourceId":          row.ResourceID,
			"resourceName":        row.ResourceName,
			"category":            row.Category,
			"carbonEmissionValue": azureCarbonRound(row.KgCO2e),
			"unit":                "kgCO2e",
		})
	}

	response := map[string]any{
		"value":    value,
		"count":    count,
		"pageSize": req.PageSize,
	}
	if end < count {
		response["skipToken"] = fmt.Sprintf("offset:%d", end)
	}
	return response
}

func azureCarbonTopItemsSummaryResponse(req azureCarbonReportRequest, rows []azureCarbonEmissionRow) map[string]any {
	type item struct {
		ResourceID       string
		ResourceName     string
		ResourceType     string
		Location         string
		ResourceGroupURL string
		Category         string
		Total            float64
	}

	items := map[string]*item{}
	for _, row := range rows {
		current := items[row.ResourceID]
		if current == nil {
			current = &item{
				ResourceID:       row.ResourceID,
				ResourceName:     row.ResourceName,
				ResourceType:     row.ResourceType,
				Location:         row.Location,
				ResourceGroupURL: row.ResourceGroupURL,
				Category:         row.Category,
			}
			items[row.ResourceID] = current
		}
		current.Total += row.KgCO2e
	}

	ordered := make([]*item, 0, len(items))
	for _, current := range items {
		ordered = append(ordered, current)
	}
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].Total != ordered[j].Total {
			return ordered[i].Total > ordered[j].Total
		}
		return ordered[i].ResourceID < ordered[j].ResourceID
	})
	if len(ordered) > req.TopItems {
		ordered = ordered[:req.TopItems]
	}

	value := make([]map[string]any, 0, len(ordered))
	for _, current := range ordered {
		value = append(value, map[string]any{
			"resourceId":          current.ResourceID,
			"resourceName":        current.ResourceName,
			"resourceType":        current.ResourceType,
			"location":            current.Location,
			"resourceGroupUrl":    current.ResourceGroupURL,
			"category":            current.Category,
			"carbonEmissionValue": azureCarbonRound(current.Total),
			"unit":                "kgCO2e",
		})
	}

	return map[string]any{
		"value": value,
		"count": len(value),
	}
}

func azureCarbonTopItemsMonthlySummaryResponse(req azureCarbonReportRequest, rows []azureCarbonEmissionRow) map[string]any {
	monthGroups := map[string][]azureCarbonEmissionRow{}
	for _, row := range rows {
		month := row.Month.Format("2006-01")
		monthGroups[month] = append(monthGroups[month], row)
	}

	months := make([]string, 0, len(monthGroups))
	for month := range monthGroups {
		months = append(months, month)
	}
	sort.Strings(months)

	value := make([]map[string]any, 0, len(months))
	for _, month := range months {
		perMonth := azureCarbonTopItemsSummaryResponse(req, monthGroups[month])
		value = append(value, map[string]any{
			"month": month,
			"items": perMonth["value"],
		})
	}

	return map[string]any{
		"value": value,
		"count": len(value),
	}
}

func azureCarbonOperationsFixture() []map[string]any {
	return []map[string]any{
		{
			"name": "Microsoft.Carbon/operations/read",
			"display": map[string]any{
				"provider":  "Microsoft.Carbon",
				"resource":  "Operations",
				"operation": "List operations",
			},
		},
		{
			"name": "Microsoft.Carbon/carbonEmissionReports/action",
			"display": map[string]any{
				"provider":  "Microsoft.Carbon",
				"resource":  "Carbon Emission Reports",
				"operation": "Query carbon emission reports",
			},
		},
		{
			"name": "Microsoft.Carbon/queryCarbonEmissionDataAvailableDateRange/action",
			"display": map[string]any{
				"provider":  "Microsoft.Carbon",
				"resource":  "Carbon Emission Data Available Date Range",
				"operation": "Query carbon emission data available date range",
			},
		},
	}
}

func azureCarbonReadRequestBody(w http.ResponseWriter, r *http.Request, path string) ([]byte, map[string]any, bool) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		respondAzureInvalidRequest(w, path, "failed to read request body")
		return nil, nil, false
	}
	if len(strings.TrimSpace(string(body))) == 0 {
		return body, map[string]any{}, true
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		respondAzureInvalidRequest(w, path, "request body must be valid JSON")
		return nil, nil, false
	}
	if payload == nil {
		payload = map[string]any{}
	}
	return body, payload, true
}

func azureCarbonParseDate(value string) (time.Time, error) {
	return time.Parse("2006-01-02", strings.TrimSpace(value))
}

func azureCarbonMonthStart(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
}

func azureCarbonParseSkipToken(value string) (int, error) {
	if !strings.HasPrefix(value, "offset:") {
		return 0, fmt.Errorf("skipToken must use offset:<n> format")
	}
	offset, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(value, "offset:")))
	if err != nil || offset < 0 {
		return 0, fmt.Errorf("skipToken must use offset:<n> format")
	}
	return offset, nil
}

func azureCarbonString(value any) string {
	if s, ok := value.(string); ok {
		return s
	}
	return ""
}

func azureCarbonInt(value any) (int, bool) {
	switch typed := value.(type) {
	case int:
		return typed, true
	case int32:
		return int(typed), true
	case int64:
		return int(typed), true
	case float64:
		if typed != float64(int(typed)) {
			return 0, false
		}
		return int(typed), true
	case json.Number:
		parsed, err := typed.Int64()
		if err != nil {
			return 0, false
		}
		return int(parsed), true
	default:
		return 0, false
	}
}

func azureCarbonObject(value any) map[string]any {
	if object, ok := value.(map[string]any); ok {
		return object
	}
	return map[string]any{}
}

func azureCarbonUniqueStrings(value any) []string {
	items, ok := value.([]any)
	if !ok {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(items))
	for _, item := range items {
		raw := strings.TrimSpace(azureCarbonString(item))
		if raw == "" {
			continue
		}
		normalized := strings.ToLower(raw)
		if _, exists := seen[normalized]; exists {
			continue
		}
		seen[normalized] = struct{}{}
		out = append(out, raw)
	}
	return out
}

func azureCarbonMembershipSet(items []string) map[string]bool {
	out := make(map[string]bool, len(items))
	for _, item := range items {
		out[strings.ToLower(strings.TrimSpace(item))] = true
	}
	return out
}

func azureCarbonRound(value float64) float64 {
	return float64(int(value*100+0.5)) / 100
}

var azureCarbonSeedRows = []azureCarbonEmissionRow{
	{
		SubscriptionID:   "00000000-0000-0000-0000-000000000000",
		Month:            time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC),
		CarbonScope:      "Scope1",
		Location:         "eastus",
		ResourceGroupURL: "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-app",
		ResourceType:     "Microsoft.Compute/virtualMachines",
		ResourceID:       "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-app/providers/Microsoft.Compute/virtualMachines/vm-a",
		ResourceName:     "vm-a",
		Category:         "Compute",
		KgCO2e:           121.4,
	},
	{
		SubscriptionID:   "00000000-0000-0000-0000-000000000000",
		Month:            time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC),
		CarbonScope:      "Scope2",
		Location:         "eastus",
		ResourceGroupURL: "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-data",
		ResourceType:     "Microsoft.Storage/storageAccounts",
		ResourceID:       "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-data/providers/Microsoft.Storage/storageAccounts/staccta",
		ResourceName:     "staccta",
		Category:         "Storage",
		KgCO2e:           66.1,
	},
	{
		SubscriptionID:   "00000000-0000-0000-0000-000000000000",
		Month:            time.Date(2025, time.February, 1, 0, 0, 0, 0, time.UTC),
		CarbonScope:      "Scope1",
		Location:         "eastus2",
		ResourceGroupURL: "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-app",
		ResourceType:     "Microsoft.Compute/virtualMachines",
		ResourceID:       "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-app/providers/Microsoft.Compute/virtualMachines/vm-a",
		ResourceName:     "vm-a",
		Category:         "Compute",
		KgCO2e:           119.8,
	},
	{
		SubscriptionID:   "00000000-0000-0000-0000-000000000000",
		Month:            time.Date(2025, time.February, 1, 0, 0, 0, 0, time.UTC),
		CarbonScope:      "Scope3",
		Location:         "westus2",
		ResourceGroupURL: "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-analytics",
		ResourceType:     "Microsoft.Kusto/clusters",
		ResourceID:       "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-analytics/providers/Microsoft.Kusto/clusters/kusto-a",
		ResourceName:     "kusto-a",
		Category:         "Analytics",
		KgCO2e:           154.3,
	},
	{
		SubscriptionID:   "11111111-1111-1111-1111-111111111111",
		Month:            time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC),
		CarbonScope:      "Scope2",
		Location:         "westus2",
		ResourceGroupURL: "/subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/rg-edge",
		ResourceType:     "Microsoft.Web/sites",
		ResourceID:       "/subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/rg-edge/providers/Microsoft.Web/sites/web-a",
		ResourceName:     "web-a",
		Category:         "AppService",
		KgCO2e:           81.7,
	},
	{
		SubscriptionID:   "11111111-1111-1111-1111-111111111111",
		Month:            time.Date(2025, time.February, 1, 0, 0, 0, 0, time.UTC),
		CarbonScope:      "Scope3",
		Location:         "centralus",
		ResourceGroupURL: "/subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/rg-edge",
		ResourceType:     "Microsoft.ContainerService/managedClusters",
		ResourceID:       "/subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/rg-edge/providers/Microsoft.ContainerService/managedClusters/aks-a",
		ResourceName:     "aks-a",
		Category:         "Containers",
		KgCO2e:           143.9,
	},
}
