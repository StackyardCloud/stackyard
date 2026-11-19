package server

import (
	"math"
	"net/http"
	"net/url"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Azure Billing Benefits (Microsoft.BillingBenefits): savings plans bought
// through savings plan order aliases, their orders, the tenant savings plan
// list, purchase and update validation, and reservation order aliases.
// Nothing is billed: a purchase only stores the order and its plan.

const (
	azureBBNamespace = "Microsoft.BillingBenefits"
	azureBBOrderType = azureBBNamespace + "/savingsPlanOrders"
	azureBBPlanType  = azureBBOrderType + "/savingsPlans"
	// azureBBOwnerRole is the built-in Owner role, which elevate grants.
	azureBBOwnerRole = "8e3af657-a8ff-443c-a75c-2fe8c4bcb635"
	// A savings plan costs its hourly commitment for every hour of its
	// term, counted as 8,760 hours a year (Learn's sample order: USD 0.001
	// an hour for three years is 26.28).
	azureBBHoursPerYear = 8760
)

// azureBBCaller is the principal elevate grants Owner to; the emulator has
// one caller.
var azureBBCaller = uuid.NewSHA1(uuid.NameSpaceURL, []byte("stackyard/caller")).String()

func init() {
	ns := azureBBNamespace
	created := func(*azureARMResource) map[string]any { return map[string]any{"provisioningState": "Created"} }
	plans := azureARMResourceType{
		Namespace: ns, Type: "savingsPlanOrders/savingsPlans", Tenant: true, NoDelete: true,
		Validate: func(map[string]any) string {
			return "Savings plans are created by purchasing a savings plan order alias."
		},
		PatchBody: azureBBPlanPatch,
		Reconcile: azureBBPlanReconcile,
		Actions:   map[string]azureARMAction{"validate": azureBBValidateUpdate},
	}
	orders := azureARMResourceType{
		Namespace: ns, Type: "savingsPlanOrders", Tenant: true, NoDelete: true, NoPatch: true,
		Validate: func(map[string]any) string {
			return "Savings plan orders are created by purchasing a savings plan order alias."
		},
		Actions: map[string]azureARMAction{"elevate": azureBBElevate},
	}
	registerAzureARMResourceType(orders)
	registerAzureARMResourceType(plans)
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "savingsPlanOrderAliases", Tenant: true, NoDelete: true, NoPatch: true,
		ValidateName:       azureBBAliasName,
		Validate:           func(payload map[string]any) string { return azureBBPurchaseValidate(payload, false) },
		ReadOnlyProperties: created,
		Reconcile:          azureBBPurchaseOnce,
		Linked: func(_ *Server, rec *azureARMResource) (map[string]any, []*azureARMResource, string) {
			return azureBBSavingsPlanLinked(rec, orders, plans)
		},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "reservationOrderAliases", Tenant: true, NoDelete: true, NoPatch: true, KeepLocation: true,
		ValidateName: azureBBAliasName,
		Validate:     azureBBReservationValidate,
		ReadOnlyProperties: func(rec *azureARMResource) map[string]any {
			return map[string]any{"provisioningState": "Created", "reservationOrderId": "/providers/Microsoft.Capacity/reservationOrders/" + azureBBID(rec.ID, "")}
		},
		Reconcile: azureBBPurchaseOnce,
	})
	registerAzureARMProviderAction(ns, "validate", func(_ *Server, payload map[string]any) (int, any) {
		return azureBBValidateEach(payload, func(item map[string]any) string { return azureBBPurchaseValidate(item, false) })
	})
	registerAzureARMView(ns, "operationResults", azureBBOperationResults)
}

// azureBBID derives a stable GUID from a resource ID and a suffix.
func azureBBID(id, suffix string) string {
	return uuid.NewSHA1(uuid.NameSpaceURL, []byte(azureARMKey(id)+suffix)).String()
}

// azureBBPurchaseOnce rejects a second PUT to an alias: each one purchases
// once. Caller holds providerStorageMu.
func azureBBPurchaseOnce(_ *Server, existing, rec *azureARMResource) (map[string]any, string) {
	if existing != nil {
		order := azureARMString(existing.Properties["savingsPlanOrderId"]) + azureARMString(existing.Properties["reservationOrderId"])
		return nil, "Alias " + rec.Name + " was already used to purchase " + order + "; each alias purchases once."
	}
	return nil, ""
}

// azureBBSavingsPlanLinked creates the order and savings plan an alias
// purchases.
func azureBBSavingsPlanLinked(rec *azureARMResource, orderType, planType azureARMResourceType) (map[string]any, []*azureARMResource, string) {
	p := rec.Properties
	start := rec.CreatedAt
	years := azureBBTermYears[azureARMString(p["term"])]
	times := map[string]any{
		"benefitStartTime": start.Format(time.RFC3339), "expiryDateTime": start.AddDate(years, 0, 0).Format(time.RFC3339),
	}
	common := azureARMMerge(times, map[string]any{
		"billingScopeId": p["billingScopeId"], "billingPlan": p["billingPlan"], "displayName": p["displayName"], "term": p["term"],
	})
	if account := azureBBBillingAccount(azureARMString(p["billingScopeId"])); account != "" {
		common["billingAccountId"] = account
	}
	orderName := azureBBID(rec.ID, "")
	order := &azureARMResource{
		ID: "/providers/" + orderType.fullType() + "/" + orderName, Name: orderName, Type: orderType.fullType(),
		SKU: azureARMMerge(rec.SKU, nil), CreatedAt: start, UpdatedAt: start,
	}
	planProps := azureARMMerge(common, map[string]any{
		"appliedScopeType": p["appliedScopeType"], "appliedScopeProperties": p["appliedScopeProperties"], "commitment": p["commitment"],
		"renew": p["renew"] == true, "effectiveDateTime": start.Format(time.RFC3339), "purchaseDateTime": start.Format(time.RFC3339),
		"displayProvisioningState": "Succeeded", "userFriendlyAppliedScopeType": p["appliedScopeType"],
		"utilization": map[string]any{"trend": "", "aggregates": []any{
			azureBBUtilization(1), azureBBUtilization(7), azureBBUtilization(30),
		}},
	})
	plan := azureARMChild(order, planType, azureBBID(rec.ID, "#plan"), planProps)
	plan.SKU = azureARMMerge(rec.SKU, nil)
	order.Properties = azureARMMerge(common, map[string]any{"provisioningState": "Succeeded", "savingsPlans": []any{plan.ID}})
	return map[string]any{"savingsPlanOrderId": order.ID}, []*azureARMResource{order, plan}, ""
}

// azureBBUtilization is a new plan's utilization over days: nothing has
// used it yet.
func azureBBUtilization(days int) map[string]any {
	return map[string]any{"grain": days, "grainUnit": "days", "value": 0, "valueUnit": "percentage"}
}

// azureBBBillingAccount returns the billing account of a billing
// subscription scope, or "" for a subscription.
func azureBBBillingAccount(scope string) string {
	if i := strings.Index(strings.ToLower(scope), "/billingsubscriptions/"); i > 0 {
		return scope[:i]
	}
	return ""
}

// azureBBPlanReconcile keeps a patched plan's derived properties in step:
// its display scope type, and renewal settings only while renew is on.
// Turning renew off drops the stored renewal settings; sending new ones
// without renew on is rejected.
func azureBBPlanReconcile(_ *Server, existing, rec *azureARMResource) (map[string]any, string) {
	props := map[string]any{"userFriendlyAppliedScopeType": rec.Properties["appliedScopeType"]}
	if rec.Properties["renew"] != true {
		if renewal := rec.Properties["renewProperties"]; renewal != nil && (existing == nil || !reflect.DeepEqual(renewal, existing.Properties["renewProperties"])) {
			return nil, "properties.renewProperties needs renew set to true."
		}
		props["renewProperties"] = nil
	}
	return props, ""
}

func azureBBElevate(order *azureARMResource, _ map[string]any) (int, any) {
	name := azureBBID(order.ID, "#elevate")
	return http.StatusOK, map[string]any{
		"name": name,
		"id":   order.ID + "/providers/Microsoft.Authorization/roleAssignments/" + name,
		"properties": map[string]any{
			"principalId":      azureBBCaller,
			"roleDefinitionId": order.ID + "/providers/Microsoft.Authorization/roleDefinitions/" + azureBBOwnerRole,
			"scope":            order.ID,
		},
	}
}

// azureBBValidateUpdate checks each proposed update to a savings plan.
func azureBBValidateUpdate(_ *azureARMResource, payload map[string]any) (int, any) {
	return azureBBValidateEach(payload, azureBBUpdateValidate)
}

// azureBBValidateEach answers a validate request: one result per item of
// benefits, invalid ones with the reason.
func azureBBValidateEach(payload map[string]any, check func(map[string]any) string) (int, any) {
	items, ok := payload["benefits"].([]any)
	if !ok || len(items) == 0 {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "benefits must be a non-empty array.")
	}
	out := make([]any, 0, len(items))
	for _, raw := range items {
		item, ok := raw.(map[string]any)
		msg := "Each benefit must be an object."
		if ok {
			msg = check(item)
		}
		if msg == "" {
			out = append(out, map[string]any{"valid": true})
		} else {
			out = append(out, map[string]any{"valid": false, "reason": msg, "reasonCode": "InvalidRequestContent"})
		}
	}
	return http.StatusOK, map[string]any{"benefits": out}
}

// azureBBOperationResults lists the finished purchase operations that
// alias PUTs name in Azure-AsyncOperation.
func azureBBOperationResults(s *Server, scope string) []map[string]any {
	if scope != "" {
		return nil
	}
	var items []map[string]any
	for _, rec := range s.azureARMResourcesWhere(func(rec *azureARMResource) bool {
		return rec.Type == azureBBNamespace+"/savingsPlanOrderAliases" || rec.Type == azureBBNamespace+"/reservationOrderAliases"
	}) {
		name := azureBBID(rec.ID, "#operation")
		items = append(items, map[string]any{
			"id": "/providers/" + azureBBNamespace + "/operationResults/" + name, "name": name, "status": "Succeeded",
			"startTime": rec.CreatedAt.Format(time.RFC3339), "endTime": rec.CreatedAt.Format(time.RFC3339),
		})
	}
	return items
}

// azureBBAsyncWriter adds the purchase's Azure-AsyncOperation and
// Retry-After headers when an alias PUT answers 201.
type azureBBAsyncWriter struct {
	http.ResponseWriter
	operation string
}

func (w azureBBAsyncWriter) WriteHeader(code int) {
	if code == http.StatusCreated {
		w.Header().Set("Azure-AsyncOperation", w.operation)
		w.Header().Set("Retry-After", "5")
	}
	w.ResponseWriter.WriteHeader(code)
}

// handleAzureBillingBenefitsRouter serves what the shared ARM handler
// can't: the async headers of alias purchases, an order's payment schedule
// ($expand=schedule) and the tenant savings plan list with its summary.
// Everything else falls through to the shared handler.
func (s *Server) handleAzureBillingBenefitsRouter(w http.ResponseWriter, r *http.Request) bool {
	seg := splitPathSegments(strings.TrimPrefix(rawRequestPath(r), "/azure/"))
	if len(seg) < 3 || !strings.EqualFold(seg[0], "providers") || !strings.EqualFold(seg[1], azureBBNamespace) {
		return false
	}
	switch {
	case len(seg) == 4 && r.Method == http.MethodPut &&
		(strings.EqualFold(seg[2], "savingsPlanOrderAliases") || strings.EqualFold(seg[2], "reservationOrderAliases")):
		id := "/providers/" + azureBBNamespace + "/" + seg[2] + "/" + seg[3]
		operation := "/azure/providers/" + azureBBNamespace + "/operationResults/" + azureBBID(id, "#operation") + "?api-version=" + url.QueryEscape(r.URL.Query().Get("api-version"))
		return s.handleAzureARMRouter(azureBBAsyncWriter{ResponseWriter: w, operation: operation}, r)
	case len(seg) == 3 && r.Method == http.MethodGet && strings.EqualFold(seg[2], "savingsPlans"):
		s.listAzureBBSavingsPlans(w, r)
		return true
	case len(seg) == 4 && r.Method == http.MethodGet && strings.EqualFold(seg[2], "savingsPlanOrders") && azureBBExpands(r, "schedule"):
		return s.getAzureBBOrderWithSchedule(w, r, "/providers/"+azureBBOrderType+"/"+seg[3])
	}
	return false
}

func azureBBExpands(r *http.Request, what string) bool {
	return slices.ContainsFunc(strings.Split(r.URL.Query().Get("$expand"), ","), func(v string) bool {
		return strings.EqualFold(strings.TrimSpace(v), what)
	})
}

// getAzureBBOrderWithSchedule returns an order with its billing plan
// information, or leaves a missing order to the shared handler's 404.
func (s *Server) getAzureBBOrderWithSchedule(w http.ResponseWriter, r *http.Request, id string) bool {
	if hasAzureInvalidAPIVersion(r) {
		azureARMError(w, http.StatusBadRequest, "InvalidApiVersionParameter", "invalid api-version query parameter")
		return true
	}
	s.providerStorageMu.Lock()
	order := s.azureARMResources[azureARMKey(id)]
	var plan *azureARMResource
	if order != nil {
		if ids := azureARMSlice(order.Properties["savingsPlans"]); len(ids) > 0 {
			plan = s.azureARMResources[azureARMKey(azureARMString(ids[0]))]
		}
	}
	s.providerStorageMu.Unlock()
	if order == nil || plan == nil {
		return false
	}
	out := order.fixture()
	out["properties"] = azureARMMerge(order.Properties, map[string]any{"planInformation": azureBBSchedule(plan, time.Now().UTC())})
	respondJSON(w, http.StatusOK, out)
	return true
}

// azureBBSchedule is a plan's payments: the full cost up front, or for
// monthly billing (P1M) the same total in 12 or 36 equal payments, one a
// month from purchase. Payments due by now have been made.
func azureBBSchedule(plan *azureARMResource, now time.Time) map[string]any {
	p := plan.Properties
	commitment := azureARMObject(p["commitment"])
	currency := commitment["currencyCode"]
	amount, _ := commitment["amount"].(float64)
	years := azureBBTermYears[azureARMString(p["term"])]
	// Money in whole cents; the last payment takes the rounding remainder
	// so the payments add up to the total.
	cents := int64(math.Round(amount * azureBBHoursPerYear * float64(years) * 100))
	payments := int64(1)
	if p["billingPlan"] == "P1M" {
		payments = int64(12 * years)
	}
	each := cents / payments
	start := plan.CreatedAt
	transactions := make([]any, 0, payments)
	var next any
	for i := range payments {
		due := azureBBAddMonths(start, int(i))
		share := each
		if i == payments-1 {
			share = cents - each*(payments-1)
		}
		price := map[string]any{"amount": float64(share) / 100, "currencyCode": currency}
		tx := map[string]any{"dueDate": due.Format(time.DateOnly), "pricingCurrencyTotal": price, "status": "Scheduled"}
		if due.After(now) {
			if next == nil {
				next = due.Format(time.DateOnly)
			}
		} else {
			tx["status"], tx["paymentDate"], tx["billingCurrencyTotal"] = "Succeeded", due.Format(time.DateOnly), price
		}
		transactions = append(transactions, tx)
	}
	return map[string]any{
		"startDate": start.Format(time.DateOnly), "nextPaymentDueDate": next,
		"pricingCurrencyTotal": map[string]any{"amount": float64(cents) / 100, "currencyCode": currency}, "transactions": transactions,
	}
}

// azureBBAddMonths adds months, clamping to the month's last day: a plan
// bought on Jan 31 pays on Feb 28 (or 29), not Mar 3.
func azureBBAddMonths(t time.Time, months int) time.Time {
	first := time.Date(t.Year(), t.Month()+time.Month(months), 1, t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), t.Location())
	last := first.AddDate(0, 1, -1).Day()
	return first.AddDate(0, 0, min(t.Day(), last)-1)
}

// azureBBSummaryStates are the display states the list summary counts.
var azureBBSummaryStates = []string{"Succeeded", "Cancelled", "Expired", "Expiring", "Failed", "NoBenefit", "Pending", "Processing", "Warning"}

// listAzureBBSavingsPlans serves the tenant savings plan list: $filter,
// $orderby, take and $skiptoken paging, selectedState, and the roll-up
// counts by display state of the plans $filter selects.
func (s *Server) listAzureBBSavingsPlans(w http.ResponseWriter, r *http.Request) {
	if hasAzureInvalidAPIVersion(r) {
		azureARMError(w, http.StatusBadRequest, "InvalidApiVersionParameter", "invalid api-version query parameter")
		return
	}
	q, msg := azureBBListQuery(r.URL.Query())
	if msg != "" {
		azureARMError(w, http.StatusBadRequest, "BadRequest", msg)
		return
	}
	plans := s.azureARMResourcesWhere(func(rec *azureARMResource) bool {
		return rec.Type == azureBBPlanType && q.matches(rec)
	})
	q.sort(plans)
	counts := map[string]any{}
	for _, state := range azureBBSummaryStates {
		counts[azureBBCountKey(state)] = 0
	}
	var selected []*azureARMResource
	for _, rec := range plans {
		state := azureARMString(rec.Properties["displayProvisioningState"])
		if key := azureBBCountKey(state); counts[key] != nil {
			counts[key] = counts[key].(int) + 1
		}
		if q.state == "" || strings.EqualFold(state, q.state) {
			selected = append(selected, rec)
		}
	}
	start := min(q.skip, len(selected))
	end := len(selected)
	if q.take > 0 {
		end = start + min(q.take, end-start)
	}
	value := []any{}
	for _, rec := range selected[start:end] {
		value = append(value, rec.fixture())
	}
	body := map[string]any{"value": value, "additionalProperties": []any{map[string]any{"name": "summary", "value": counts}}}
	if end < len(selected) {
		next := r.URL.Query()
		next.Set("$skiptoken", strconv.Itoa(end))
		body["nextLink"] = rawRequestPath(r) + "?" + next.Encode()
	}
	respondJSON(w, http.StatusOK, body)
}

// azureBBCountKey is the summary key of a display state: Succeeded counts
// as succeededCount.
func azureBBCountKey(state string) string {
	if state == "" {
		return ""
	}
	return strings.ToLower(state[:1]) + state[1:] + "Count"
}
