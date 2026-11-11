package server

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const azureConsumptionAPIVersion = "2024-08-01"

var azureConsumptionSubscriptionPattern = regexp.MustCompile(`^[0-9a-fA-F-]{36}$`)

type azureConsumptionBudget struct {
	Scope      string
	Name       string
	Properties map[string]any
	ETag       string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func (s *Server) handleAzureConsumptionRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, "/azure/providers/") && !strings.HasPrefix(path, "/azure/subscriptions/") {
		return false
	}

	segments := splitPathSegments(strings.TrimPrefix(path, "/azure/"))
	if len(segments) == 0 || !azureConsumptionIsPath(segments) {
		return false
	}
	if hasAzureInvalidAPIVersion(r) {
		respondAzureInvalidRequest(w, path, "invalid api-version query parameter")
		return true
	}
	if !azureConsumptionAllowedMethod(r.Method) {
		return false
	}

	if len(segments) == 3 &&
		strings.EqualFold(segments[0], "providers") &&
		strings.EqualFold(segments[1], "Microsoft.Consumption") &&
		strings.EqualFold(segments[2], "operations") {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return true
		}
		respondJSON(w, http.StatusOK, map[string]any{"value": azureConsumptionOperationsFixture()})
		return true
	}

	scopeSegments, tail, ok := azureConsumptionRouteContext(segments)
	if !ok || len(tail) == 0 {
		return false
	}
	scope := azureConsumptionScopeString(scopeSegments)
	if scope == "" {
		respondAzureImplemented(w, path)
		return true
	}

	billingPeriodName := ""
	if len(tail) >= 2 && strings.EqualFold(tail[0], "billingPeriods") {
		billingPeriodName = strings.TrimSpace(tail[1])
		tail = tail[2:]
		if billingPeriodName == "" || len(tail) == 0 {
			respondAzureInvalidRequest(w, path, "billingPeriodName is required")
			return true
		}
	}

	family := strings.ToLower(strings.TrimSpace(tail[0]))
	switch family {
	case "budgets":
		s.handleAzureConsumptionBudgets(w, r, path, scope, billingPeriodName, tail)
	case "charges", "marketplaces", "usagedetails", "reservationrecommendations", "reservationrecommendationdetails", "reservationdetails", "reservationsummaries", "reservationtransactions", "events", "lots":
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return true
		}
		s.handleAzureConsumptionListFamily(w, r, path, scope, billingPeriodName, family, tail)
	case "balances", "credits", "pricesheets", "tags", "aggregatedcost":
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return true
		}
		s.handleAzureConsumptionGetFamily(w, r, path, scope, billingPeriodName, family, tail)
	default:
		respondAzureImplemented(w, path)
	}
	return true
}

func azureConsumptionIsPath(segments []string) bool {
	if len(segments) == 3 &&
		strings.EqualFold(segments[0], "providers") &&
		strings.EqualFold(segments[1], "Microsoft.Consumption") &&
		strings.EqualFold(segments[2], "operations") {
		return true
	}
	_, tail, ok := azureConsumptionRouteContext(segments)
	if !ok || len(tail) == 0 {
		return false
	}
	if len(tail) >= 3 && strings.EqualFold(tail[0], "billingPeriods") {
		tail = tail[2:]
	}
	if len(tail) == 0 {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(tail[0])) {
	case "budgets", "charges", "marketplaces", "usagedetails", "balances", "credits", "events", "lots", "pricesheets", "tags", "aggregatedcost", "reservationrecommendations", "reservationrecommendationdetails", "reservationtransactions", "reservationdetails", "reservationsummaries":
		return true
	default:
		return false
	}
}

func azureConsumptionAllowedMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodPut, http.MethodDelete:
		return true
	default:
		return false
	}
}

func azureConsumptionRouteContext(segments []string) (scopeSegments []string, tail []string, ok bool) {
	for idx := len(segments) - 2; idx >= 0; idx-- {
		if strings.EqualFold(segments[idx], "providers") && strings.EqualFold(segments[idx+1], "Microsoft.Consumption") {
			return segments[:idx], segments[idx+2:], true
		}
	}
	return nil, nil, false
}

func (s *Server) handleAzureConsumptionBudgets(w http.ResponseWriter, r *http.Request, path, scope, billingPeriodName string, tail []string) {
	if billingPeriodName != "" {
		respondAzureInvalidRequest(w, path, "budgets do not support billingPeriodName in path")
		return
	}
	if len(tail) == 1 {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleAzureConsumptionListBudgets(w, scope)
		return
	}

	budgetName := strings.TrimSpace(tail[1])
	if budgetName == "" {
		respondAzureInvalidRequest(w, path, "budgetName is required")
		return
	}
	switch r.Method {
	case http.MethodGet:
		s.handleAzureConsumptionGetBudget(w, path, scope, budgetName, false)
	case http.MethodHead:
		s.handleAzureConsumptionGetBudget(w, path, scope, budgetName, true)
	case http.MethodPut:
		s.handleAzureConsumptionPutBudget(w, r, path, scope, budgetName)
	case http.MethodDelete:
		s.handleAzureConsumptionDeleteBudget(w, scope, budgetName)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleAzureConsumptionListBudgets(w http.ResponseWriter, scope string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	items := make([]map[string]any, 0)
	scopeKey := strings.ToLower(scope)
	for _, budget := range s.azureConsumptionBudgets {
		if strings.ToLower(budget.Scope) == scopeKey {
			items = append(items, azureConsumptionBudgetResponse(budget))
		}
	}
	sort.Slice(items, func(i, j int) bool {
		return strings.Compare(strings.ToLower(azureConsumptionString(items[i]["name"])), strings.ToLower(azureConsumptionString(items[j]["name"]))) < 0
	})
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) handleAzureConsumptionGetBudget(w http.ResponseWriter, path, scope, budgetName string, headOnly bool) {
	s.providerStorageMu.Lock()
	budget := s.azureConsumptionBudgets[azureConsumptionBudgetKey(scope, budgetName)]
	s.providerStorageMu.Unlock()
	if budget == nil {
		azureConsumptionRespondError(w, http.StatusNotFound, "ResourceNotFound", "budget was not found")
		return
	}
	if headOnly {
		w.WriteHeader(http.StatusOK)
		return
	}
	respondJSON(w, http.StatusOK, azureConsumptionBudgetResponse(budget))
}

func (s *Server) handleAzureConsumptionPutBudget(w http.ResponseWriter, r *http.Request, path, scope, budgetName string) {
	_, payload, ok := azureConsumptionReadRequestBody(w, r, path)
	if !ok {
		return
	}
	properties, _ := payload["properties"].(map[string]any)
	if _, ok := azureConsumptionNumber(properties["amount"]); !ok {
		respondAzureInvalidRequest(w, path, "properties.amount is required")
		return
	}
	if strings.TrimSpace(azureConsumptionString(properties["category"])) == "" {
		respondAzureInvalidRequest(w, path, "properties.category is required")
		return
	}
	if strings.TrimSpace(azureConsumptionString(properties["timeGrain"])) == "" {
		respondAzureInvalidRequest(w, path, "properties.timeGrain is required")
		return
	}
	timePeriod, _ := properties["timePeriod"].(map[string]any)
	if strings.TrimSpace(azureConsumptionString(timePeriod["startDate"])) == "" || strings.TrimSpace(azureConsumptionString(timePeriod["endDate"])) == "" {
		respondAzureInvalidRequest(w, path, "properties.timePeriod.startDate and properties.timePeriod.endDate are required")
		return
	}

	now := time.Now().UTC()
	key := azureConsumptionBudgetKey(scope, budgetName)
	s.providerStorageMu.Lock()
	budget := s.azureConsumptionBudgets[key]
	created := false
	if budget == nil {
		budget = &azureConsumptionBudget{
			Scope:     strings.ToLower(scope),
			Name:      budgetName,
			CreatedAt: now,
		}
		created = true
	}
	budget.Properties = azureConsumptionNormalizeBudgetProperties(scope, budgetName, properties)
	budget.UpdatedAt = now
	budget.ETag = azureConsumptionETag("budget", scope, budgetName, now.Format(time.RFC3339Nano))
	s.azureConsumptionBudgets[key] = budget
	s.providerStorageMu.Unlock()

	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	respondJSON(w, status, azureConsumptionBudgetResponse(budget))
}

func (s *Server) handleAzureConsumptionDeleteBudget(w http.ResponseWriter, scope, budgetName string) {
	s.providerStorageMu.Lock()
	delete(s.azureConsumptionBudgets, azureConsumptionBudgetKey(scope, budgetName))
	s.providerStorageMu.Unlock()
	respondJSON(w, http.StatusOK, map[string]any{})
}

func (s *Server) handleAzureConsumptionListFamily(w http.ResponseWriter, r *http.Request, path, scope, billingPeriodName, family string, tail []string) {
	if len(tail) != 1 {
		respondAzureImplemented(w, path)
		return
	}
	top, err := azureConsumptionPositiveIntQuery(r, "top")
	if err != nil {
		respondAzureInvalidRequest(w, path, err.Error())
		return
	}
	pageSize := top
	if pageSize == 0 {
		pageSize = 3
	}
	skiptoken := strings.TrimSpace(r.URL.Query().Get("skiptoken"))
	start := 0
	if skiptoken != "" {
		parsed, err := strconv.Atoi(skiptoken)
		if err != nil || parsed < 0 {
			respondAzureInvalidRequest(w, path, "skiptoken must be a non-negative integer")
			return
		}
		start = parsed
	}
	items := azureConsumptionListFamilyFixture(scope, billingPeriodName, family, r.URL.Query().Get("filter"), pageSize, start)
	body := map[string]any{"value": items}
	next := start + len(items)
	if next < azureConsumptionFamilyTotalCount(family) {
		body["nextLink"] = fmt.Sprintf("%s?api-version=%s&skiptoken=%d", rawRequestPath(r), azureConsumptionAPIVersion, next)
	}
	respondJSON(w, http.StatusOK, body)
}

func (s *Server) handleAzureConsumptionGetFamily(w http.ResponseWriter, r *http.Request, path, scope, billingPeriodName, family string, tail []string) {
	if family == "tags" && len(tail) == 2 {
		tagName := strings.TrimSpace(tail[1])
		if tagName == "" {
			respondAzureInvalidRequest(w, path, "tagName is required")
			return
		}
		payload := azureConsumptionTagDetailFixture(scope, billingPeriodName, tagName)
		if r.Method == http.MethodHead {
			w.WriteHeader(http.StatusOK)
			return
		}
		respondJSON(w, http.StatusOK, payload)
		return
	}
	if len(tail) != 1 {
		respondAzureImplemented(w, path)
		return
	}
	if r.Method == http.MethodHead {
		w.WriteHeader(http.StatusOK)
		return
	}
	respondJSON(w, http.StatusOK, azureConsumptionGetFamilyFixture(scope, billingPeriodName, family, r.URL.Query().Get("filter")))
}

func azureConsumptionOperationsFixture() []map[string]any {
	names := []string{
		"Microsoft.Consumption/budgets/read",
		"Microsoft.Consumption/budgets/write",
		"Microsoft.Consumption/budgets/delete",
		"Microsoft.Consumption/charges/read",
		"Microsoft.Consumption/marketplaces/read",
		"Microsoft.Consumption/usageDetails/read",
		"Microsoft.Consumption/tags/read",
		"Microsoft.Consumption/balances/read",
		"Microsoft.Consumption/credits/read",
		"Microsoft.Consumption/events/read",
		"Microsoft.Consumption/lots/read",
		"Microsoft.Consumption/pricesheets/read",
		"Microsoft.Consumption/aggregatedcost/read",
		"Microsoft.Consumption/reservationRecommendations/read",
		"Microsoft.Consumption/reservationRecommendationDetails/read",
		"Microsoft.Consumption/reservationTransactions/read",
		"Microsoft.Consumption/reservationDetails/read",
		"Microsoft.Consumption/reservationSummaries/read",
	}
	out := make([]map[string]any, 0, len(names))
	for _, name := range names {
		out = append(out, map[string]any{
			"id":     "/providers/Microsoft.Consumption/operations/" + strings.ReplaceAll(strings.ToLower(name), "/", "_"),
			"name":   name,
			"origin": "system",
			"display": map[string]any{
				"provider":  "Microsoft Consumption",
				"resource":  "Consumption",
				"operation": azureConsumptionOperationVerb(name),
			},
			"isDataAction": false,
		})
	}
	return out
}

func azureConsumptionBudgetResponse(budget *azureConsumptionBudget) map[string]any {
	return map[string]any{
		"id":         budget.Scope + "/providers/Microsoft.Consumption/budgets/" + budget.Name,
		"name":       budget.Name,
		"type":       "Microsoft.Consumption/budgets",
		"eTag":       budget.ETag,
		"properties": azureConsumptionCloneMap(budget.Properties),
	}
}

func azureConsumptionNormalizeBudgetProperties(scope, budgetName string, properties map[string]any) map[string]any {
	normalized := azureConsumptionCloneMap(properties)
	amount, _ := azureConsumptionNumber(properties["amount"])
	current := amount * 0.38
	forecast := amount * 0.81
	normalized["currentSpend"] = map[string]any{
		"amount": amountRound(current),
		"unit":   "USD",
	}
	normalized["forecastSpend"] = map[string]any{
		"amount": amountRound(forecast),
		"unit":   "USD",
	}
	normalized["notifications"] = azureConsumptionCloneMap(properties["notifications"])
	normalized["resourceGroupFilter"] = azureConsumptionScopeResourceGroup(scope)
	normalized["budgetScope"] = scope
	normalized["provisioningState"] = "Succeeded"
	normalized["resourceId"] = scope + "/providers/Microsoft.Consumption/budgets/" + budgetName
	return normalized
}

func azureConsumptionGetFamilyFixture(scope, billingPeriodName, family, filter string) map[string]any {
	id := scope + "/providers/Microsoft.Consumption/" + family
	if billingPeriodName != "" {
		id = scope + "/providers/Microsoft.Consumption/billingPeriods/" + billingPeriodName + "/" + family
	}
	switch family {
	case "balances":
		return map[string]any{
			"id":   id,
			"name": "default",
			"type": "Microsoft.Consumption/balances",
			"properties": map[string]any{
				"currency":         "USD",
				"beginningBalance": 10000.0,
				"newPurchases":     250.0,
				"adjustments":      -10.0,
				"currentBalance":   8240.5,
				"asOf":             time.Now().UTC().Format(time.RFC3339),
			},
		}
	case "credits":
		return map[string]any{
			"id":   id,
			"name": "default",
			"type": "Microsoft.Consumption/credits",
			"properties": map[string]any{
				"balanceSummary": []map[string]any{
					{"name": "Azure Credit", "amount": 500.0, "currency": "USD"},
					{"name": "Marketplace Credit", "amount": 75.5, "currency": "USD"},
				},
			},
		}
	case "pricesheets":
		return map[string]any{
			"id":   id,
			"name": "default",
			"type": "Microsoft.Consumption/pricesheets",
			"properties": map[string]any{
				"billingPeriod": azureConsumptionStringOr(billingPeriodName, "current"),
				"filter":        filter,
				"pricesheets": []map[string]any{
					{"meterName": "Virtual Machines D2s v5", "unitPrice": 0.096, "currencyCode": "USD"},
					{"meterName": "Premium SSD Managed Disks", "unitPrice": 0.135, "currencyCode": "USD"},
				},
			},
		}
	case "tags":
		return map[string]any{
			"id":   id,
			"name": "default",
			"type": "Microsoft.Consumption/tags",
			"properties": map[string]any{
				"tags": []map[string]any{
					{"key": "env", "value": "prod", "cost": 412.31, "currency": "USD"},
					{"key": "team", "value": "platform", "cost": 288.14, "currency": "USD"},
				},
			},
		}
	case "aggregatedcost":
		return map[string]any{
			"id":   id,
			"name": "default",
			"type": "Microsoft.Consumption/aggregatedcost",
			"properties": map[string]any{
				"currency":        "USD",
				"managementGroup": azureConsumptionScopeLeaf(scope),
				"billingPeriod":   azureConsumptionStringOr(billingPeriodName, "current"),
				"granularity":     "None",
				"rows": []map[string]any{
					{"name": "Compute", "cost": 1388.42},
					{"name": "Storage", "cost": 421.77},
				},
				"totalCost": 1810.19,
			},
		}
	default:
		return map[string]any{
			"id":   id,
			"name": "default",
			"type": "Microsoft.Consumption/" + family,
			"properties": map[string]any{
				"scope": scope,
			},
		}
	}
}

func azureConsumptionTagDetailFixture(scope, billingPeriodName, tagName string) map[string]any {
	return map[string]any{
		"id":   scope + "/providers/Microsoft.Consumption/tags/" + tagName,
		"name": tagName,
		"type": "Microsoft.Consumption/tags",
		"properties": map[string]any{
			"billingPeriod": azureConsumptionStringOr(billingPeriodName, "current"),
			"rows": []map[string]any{
				{"value": "prod", "cost": 412.31, "currency": "USD"},
				{"value": "stage", "cost": 94.05, "currency": "USD"},
			},
		},
	}
}

func azureConsumptionListFamilyFixture(scope, billingPeriodName, family, filter string, pageSize, start int) []map[string]any {
	total := azureConsumptionFamilyTotalCount(family)
	if start > total {
		return []map[string]any{}
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	items := make([]map[string]any, 0, end-start)
	for idx := start; idx < end; idx++ {
		items = append(items, azureConsumptionListItem(scope, billingPeriodName, family, filter, idx))
	}
	return items
}

func azureConsumptionListItem(scope, billingPeriodName, family, filter string, idx int) map[string]any {
	baseID := scope + "/providers/Microsoft.Consumption/" + family + "/" + fmt.Sprintf("%s-%03d", family, idx+1)
	if billingPeriodName != "" {
		baseID = scope + "/providers/Microsoft.Consumption/billingPeriods/" + billingPeriodName + "/" + family + "/" + fmt.Sprintf("%s-%03d", family, idx+1)
	}
	switch family {
	case "charges":
		return map[string]any{
			"id":   baseID,
			"name": fmt.Sprintf("charge-%03d", idx+1),
			"type": "Microsoft.Consumption/charges",
			"properties": map[string]any{
				"chargeType":    "Usage",
				"scope":         scope,
				"billingPeriod": azureConsumptionStringOr(billingPeriodName, "current"),
				"cost":          amountRound(25.0 + float64(idx)*7.25),
				"currency":      "USD",
				"filter":        filter,
			},
		}
	case "marketplaces":
		return map[string]any{
			"id":   baseID,
			"name": fmt.Sprintf("marketplace-%03d", idx+1),
			"type": "Microsoft.Consumption/marketplaces",
			"properties": map[string]any{
				"publisherName": "Contoso Marketplace",
				"offerName":     fmt.Sprintf("Managed Offer %d", idx+1),
				"cost":          amountRound(15.0 + float64(idx)*3.5),
				"currency":      "USD",
				"filter":        filter,
			},
		}
	case "usagedetails":
		return map[string]any{
			"id":   baseID,
			"name": fmt.Sprintf("usage-%03d", idx+1),
			"type": "Microsoft.Consumption/usageDetails",
			"properties": map[string]any{
				"instanceName":  fmt.Sprintf("vm-%03d", idx+1),
				"meterCategory": "Virtual Machines",
				"pretaxCost":    amountRound(12.5 + float64(idx)*1.75),
				"currency":      "USD",
				"usageDate":     time.Now().UTC().AddDate(0, 0, -idx).Format("2006-01-02"),
				"filter":        filter,
			},
		}
	case "reservationrecommendations":
		return map[string]any{
			"id":   baseID,
			"name": fmt.Sprintf("recommendation-%03d", idx+1),
			"type": "Microsoft.Consumption/reservationRecommendations",
			"properties": map[string]any{
				"scope":                    scope,
				"lookBackPeriod":           "Last7Days",
				"recommendedQuantity":      idx + 1,
				"resourceType":             "VirtualMachines",
				"term":                     "P1Y",
				"totalCostWithReservation": amountRound(340.0 + float64(idx)*12.0),
			},
		}
	case "reservationrecommendationdetails":
		return map[string]any{
			"id":   baseID,
			"name": fmt.Sprintf("recommendation-detail-%03d", idx+1),
			"type": "Microsoft.Consumption/reservationRecommendationDetails",
			"properties": map[string]any{
				"instanceFlexibilityGroup": "DSv5 Series",
				"region":                   "eastus",
				"term":                     "P1Y",
				"usage":                    amountRound(120.0 + float64(idx)*8.0),
			},
		}
	case "reservationtransactions":
		return map[string]any{
			"id":   baseID,
			"name": fmt.Sprintf("reservation-transaction-%03d", idx+1),
			"type": "Microsoft.Consumption/reservationTransactions",
			"properties": map[string]any{
				"eventDate":          time.Now().UTC().AddDate(0, 0, -idx).Format(time.RFC3339),
				"eventType":          "Purchase",
				"reservationOrderId": "ro-12345",
				"amount":             amountRound(500.0 + float64(idx)*50.0),
				"currency":           "USD",
			},
		}
	case "reservationdetails":
		return map[string]any{
			"id":   baseID,
			"name": fmt.Sprintf("reservation-detail-%03d", idx+1),
			"type": "Microsoft.Consumption/reservationDetails",
			"properties": map[string]any{
				"instanceId":         fmt.Sprintf("vm-res-%03d", idx+1),
				"reservedHours":      744,
				"usedHours":          701 - idx,
				"skuName":            "Standard_D4s_v5",
				"reservationOrderId": "ro-12345",
			},
		}
	case "reservationsummaries":
		return map[string]any{
			"id":   baseID,
			"name": fmt.Sprintf("reservation-summary-%03d", idx+1),
			"type": "Microsoft.Consumption/reservationSummaries",
			"properties": map[string]any{
				"avgUtilizationPercentage": amountRound(82.0 - float64(idx)*3.5),
				"reservationOrderId":       "ro-12345",
				"skuName":                  "Standard_D4s_v5",
			},
		}
	case "events":
		return map[string]any{
			"id":   baseID,
			"name": fmt.Sprintf("event-%03d", idx+1),
			"type": "Microsoft.Consumption/events",
			"properties": map[string]any{
				"eventType":  "InvoiceGenerated",
				"message":    fmt.Sprintf("Billing event %d for %s", idx+1, azureConsumptionScopeLeaf(scope)),
				"occurredAt": time.Now().UTC().AddDate(0, 0, -idx).Format(time.RFC3339),
			},
		}
	case "lots":
		return map[string]any{
			"id":   baseID,
			"name": fmt.Sprintf("lot-%03d", idx+1),
			"type": "Microsoft.Consumption/lots",
			"properties": map[string]any{
				"lotId":          fmt.Sprintf("lot-%03d", idx+1),
				"source":         "CreditPurchase",
				"originalAmount": amountRound(1000.0 + float64(idx)*150.0),
				"currency":       "USD",
			},
		}
	default:
		return map[string]any{
			"id":   baseID,
			"name": fmt.Sprintf("%s-%03d", family, idx+1),
			"type": "Microsoft.Consumption/" + family,
			"properties": map[string]any{
				"scope":  scope,
				"filter": filter,
			},
		}
	}
}

func azureConsumptionFamilyTotalCount(family string) int {
	switch family {
	case "reservationrecommendationdetails":
		return 2
	default:
		return 5
	}
}

func azureConsumptionOperationVerb(name string) string {
	switch {
	case strings.HasSuffix(name, "/write"):
		return "Write"
	case strings.HasSuffix(name, "/delete"):
		return "Delete"
	default:
		return "Read"
	}
}

func azureConsumptionReadRequestBody(w http.ResponseWriter, r *http.Request, path string) ([]byte, map[string]any, bool) {
	if r.Body == nil {
		return nil, map[string]any{}, true
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		respondAzureInvalidRequest(w, path, "unable to read request body")
		return nil, nil, false
	}
	if strings.TrimSpace(string(body)) == "" {
		return body, map[string]any{}, true
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		respondAzureInvalidRequest(w, path, "request body must be valid JSON")
		return body, nil, false
	}
	if payload == nil {
		payload = map[string]any{}
	}
	return body, payload, true
}

func azureConsumptionRespondError(w http.ResponseWriter, status int, code, message string) {
	respondJSON(w, status, map[string]any{
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
	})
}

func azureConsumptionBudgetKey(scope, name string) string {
	return strings.ToLower(strings.TrimSpace(scope)) + "::" + strings.ToLower(strings.TrimSpace(name))
}

func azureConsumptionScopeString(segments []string) string {
	if len(segments) == 0 {
		return ""
	}
	return "/" + strings.Join(segments, "/")
}

func azureConsumptionScopeLeaf(scope string) string {
	scope = strings.Trim(scope, "/")
	if scope == "" {
		return ""
	}
	parts := strings.Split(scope, "/")
	return parts[len(parts)-1]
}

func azureConsumptionScopeResourceGroup(scope string) string {
	segments := splitPathSegments(strings.Trim(scope, "/"))
	for idx := 0; idx+1 < len(segments); idx++ {
		if strings.EqualFold(segments[idx], "resourceGroups") {
			return segments[idx+1]
		}
	}
	return ""
}

func azureConsumptionPositiveIntQuery(r *http.Request, key string) (int, error) {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return 0, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 0 {
		return 0, fmt.Errorf("%s must be a non-negative integer", key)
	}
	return value, nil
}

func azureConsumptionCloneMap(value any) map[string]any {
	source, ok := value.(map[string]any)
	if !ok || len(source) == 0 {
		return map[string]any{}
	}
	cloned := make(map[string]any, len(source))
	for key, item := range source {
		cloned[key] = item
	}
	return cloned
}

func azureConsumptionString(value any) string {
	if s, ok := value.(string); ok {
		return strings.TrimSpace(s)
	}
	return ""
}

func azureConsumptionStringOr(value any, fallback string) string {
	if s := azureConsumptionString(value); s != "" {
		return s
	}
	return fallback
}

func azureConsumptionNumber(value any) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case json.Number:
		parsed, err := typed.Float64()
		if err != nil {
			return 0, false
		}
		return parsed, true
	default:
		return 0, false
	}
}

func azureConsumptionETag(parts ...string) string {
	h := sha1.New()
	_, _ = io.WriteString(h, strings.Join(parts, "|"))
	return `"` + hex.EncodeToString(h.Sum(nil))[:24] + `"`
}

func amountRound(value float64) float64 {
	return float64(int(value*100+0.5)) / 100
}
