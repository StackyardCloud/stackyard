package server

import (
	"math"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Microsoft.Capacity reservations: purchasing a reservation order creates
// its reservation, which can be updated, split, merged, archived and
// returned. Nothing is billed: prices come from a flat sample rate, and
// reservations never apply to running resources.
func init() {
	const ns = "Microsoft.Capacity"
	reservations := azureARMResourceType{
		Namespace: ns, Type: "reservationOrders/reservations", Tenant: true, NoDelete: true,
		// Reservations come only from purchases, splits and merges.
		Validate: func(map[string]any) string {
			return "Reservations are created by purchasing a reservation order."
		},
		ReadOnlyProperties: azureReservationProperties,
		PatchBody:          azureReservationPatch,
		StateProperties:    map[string]any{"archived": false},
		Transitions: map[string]map[string]any{
			"archive": {"archived": true}, "unarchive": {"archived": false},
		},
		Actions: map[string]azureARMAction{"availablescopes": azureReservationAvailableScopes},
	}
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "reservationOrders", Tenant: true, KeepLocation: true, NoDelete: true,
		// Purchases answer 200 (or 202), never 201.
		CreateStatus: http.StatusOK,
		ValidateName: func(name string) string {
			if _, err := uuid.Parse(name); err != nil {
				return "Reservation order IDs are GUIDs, such as the reservationOrderId that calculatePrice returns."
			}
			return ""
		},
		Validate:            azureReservationPurchaseValidate,
		WriteOnlyProperties: azureReservationPurchaseOnly,
		ReadOnlyProperties:  azureReservationOrderProperties,
		Linked: func(s *Server, rec *azureARMResource) (map[string]any, []*azureARMResource, string) {
			return azureReservationOrderLinked(s, rec, reservations)
		},
		ServerActions: map[string]func(*Server, *azureARMResource, map[string]any) (int, any){
			"split": func(s *Server, rec *azureARMResource, p map[string]any) (int, any) {
				return s.azureReservationSplit(rec, p, reservations)
			},
			"merge": func(s *Server, rec *azureARMResource, p map[string]any) (int, any) {
				return s.azureReservationMerge(rec, p, reservations)
			},
			"return":          (*Server).azureReservationReturn,
			"calculaterefund": (*Server).azureReservationCalculateRefund,
			"changedirectory": (*Server).azureReservationChangeDirectory,
		},
	})
	registerAzureARMResourceType(reservations)
	registerAzureARMView(ns, "reservations", func(s *Server, _ string) []map[string]any {
		var items []map[string]any
		for _, rec := range s.azureARMResourcesWhere(func(rec *azureARMResource) bool { return rec.Type == reservations.fullType() }) {
			items = append(items, rec.fixture())
		}
		return items
	})
	registerAzureARMView(ns, "catalogs", azureReservationCatalog)
	registerAzureARMProviderAction(ns, "calculatePrice", func(_ *Server, payload map[string]any) (int, any) {
		if msg := azureReservationPurchaseValidate(payload); msg != "" {
			return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", msg)
		}
		return http.StatusOK, azureReservationPrice(payload)
	})
}

// ponytail: every SKU costs a flat sample rate per unit per month; a price
// list per SKU and region is the upgrade if tests need realistic totals.
const azureReservationMonthlyPrice = 100.0

var (
	azureReservationResourceTypes = []string{"VirtualMachines", "SqlDatabases", "SuseLinux", "CosmosDb", "RedHat", "SqlDataWarehouse",
		"VMwareCloudSimple", "RedHatOsa", "Databricks", "AppService", "ManagedDisk", "BlockBlob", "RedisCache", "AzureDataExplorer", "MySql",
		"MariaDb", "PostgreSql", "DedicatedHost", "SapHana", "SqlAzureHybridBenefit", "AVS", "DataFactory", "NetAppStorage", "AzureFiles",
		"SqlEdge", "VirtualMachineSoftware"}
	azureReservationTermMonths = map[string]int{"P1Y": 12, "P3Y": 36, "P5Y": 60}
	// Purchase details a reservation order passes to its reservations
	// instead of returning them itself.
	azureReservationPurchaseOnly = []string{"reservedResourceType", "billingScopeId", "quantity", "appliedScopeType", "appliedScopes",
		"appliedScopeProperties", "renew", "reservedResourceProperties"}
)

func azureReservationPurchaseValidate(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	if strings.TrimSpace(azureARMString(azureARMObject(payload["sku"])["name"])) == "" {
		return "sku.name is required."
	}
	if !slices.Contains(azureReservationResourceTypes, azureARMString(props["reservedResourceType"])) {
		return "properties.reservedResourceType must be a reservable resource type, such as VirtualMachines."
	}
	billing := strings.ToLower(azureARMString(props["billingScopeId"]))
	if _, err := uuid.Parse(strings.TrimPrefix(billing, "/subscriptions/")); err != nil && !strings.HasPrefix(billing, "/providers/microsoft.billing/billingaccounts/") {
		return "properties.billingScopeId must be a subscription (/subscriptions/{id}) or billing profile ID."
	}
	if _, ok := azureReservationTermMonths[azureARMString(props["term"])]; !ok {
		return "properties.term must be P1Y, P3Y or P5Y."
	}
	if raw, present := props["billingPlan"]; present && raw != "Upfront" && raw != "Monthly" {
		return "properties.billingPlan must be Upfront or Monthly."
	}
	if !azureWholeNumber(props["quantity"], 1, 2147483647) {
		return "properties.quantity must be a whole number of at least 1."
	}
	if raw, present := azureARMObject(props["reservedResourceProperties"])["instanceFlexibility"]; present && raw != "On" && raw != "Off" {
		return "properties.reservedResourceProperties.instanceFlexibility must be On or Off."
	}
	if props["appliedScopeType"] == nil {
		return "properties.appliedScopeType is required."
	}
	return azureReservationScopeValidate(props)
}

// azureReservationScopeValidate checks that the scope type and scopes
// agree: Single needs one subscription or resource group, ManagementGroup
// a management group and its tenant, and Shared no scopes.
func azureReservationScopeValidate(props map[string]any) string {
	scopes := azureARMSlice(props["appliedScopes"])
	scopeProps := azureARMObject(props["appliedScopeProperties"])
	switch props["appliedScopeType"] {
	case nil:
	case "Single":
		single := len(scopes) == 1 && strings.HasPrefix(strings.ToLower(azureARMString(scopes[0])), "/subscriptions/")
		if !single && scopeProps["subscriptionId"] == nil && scopeProps["resourceGroupId"] == nil {
			return "A Single scope needs one subscription or resource group in appliedScopes or appliedScopeProperties."
		}
	case "Shared":
		if len(scopes) != 0 {
			return "A Shared scope takes no appliedScopes."
		}
	case "ManagementGroup":
		_, err := uuid.Parse(azureARMString(scopeProps["tenantId"]))
		if !strings.HasPrefix(strings.ToLower(azureARMString(scopeProps["managementGroupId"])), "/providers/microsoft.management/managementgroups/") || err != nil {
			return "A ManagementGroup scope needs appliedScopeProperties.managementGroupId and tenantId."
		}
	default:
		return "properties.appliedScopeType must be Single, Shared or ManagementGroup."
	}
	return ""
}

func azureReservationExpiry(start time.Time, term string) time.Time {
	return start.AddDate(0, azureReservationTermMonths[term], 0)
}

func azureReservationOrderProperties(rec *azureARMResource) map[string]any {
	term := azureARMString(rec.Properties["term"])
	expiry := azureReservationExpiry(rec.CreatedAt, term)
	plan := azureARMString(rec.Properties["billingPlan"])
	if plan == "" {
		plan = "Upfront"
	}
	return map[string]any{
		"requestDateTime": rec.CreatedAt.Format(time.RFC3339), "createdDateTime": rec.CreatedAt.Format(time.RFC3339),
		"benefitStartTime": rec.CreatedAt.Format(time.RFC3339), "expiryDate": expiry.Format(time.DateOnly),
		"expiryDateTime": expiry.Format(time.RFC3339), "originalQuantity": rec.Properties["quantity"], "billingPlan": plan,
	}
}

// azureReservationOrderLinked creates a new order's reservation from the
// purchase details. An order can be purchased only once. Caller holds
// providerStorageMu.
func azureReservationOrderLinked(s *Server, rec *azureARMResource, typ azureARMResourceType) (map[string]any, []*azureARMResource, string) {
	if s.azureARMResources[azureARMKey(rec.ID)] != nil {
		return nil, nil, "Reservation order " + rec.Name + " was already purchased."
	}
	p := rec.Properties
	flexibility := azureARMObject(p["reservedResourceProperties"])["instanceFlexibility"]
	if flexibility == nil && p["reservedResourceType"] == "VirtualMachines" {
		flexibility = "On"
	}
	expiry := azureReservationExpiry(rec.CreatedAt, azureARMString(p["term"]))
	props := map[string]any{
		"reservedResourceType": p["reservedResourceType"], "displayName": p["displayName"], "quantity": p["quantity"],
		"appliedScopeType": p["appliedScopeType"], "appliedScopes": p["appliedScopes"], "appliedScopeProperties": p["appliedScopeProperties"],
		"billingPlan": p["billingPlan"], "billingScopeId": p["billingScopeId"], "term": p["term"], "renew": p["renew"] == true,
		"instanceFlexibility": flexibility, "skuDescription": rec.SKU["name"],
		"effectiveDateTime": rec.CreatedAt.Format(time.RFC3339), "benefitStartTime": rec.CreatedAt.Format(time.RFC3339),
		"purchaseDate": rec.CreatedAt.Format(time.DateOnly), "purchaseDateTime": rec.CreatedAt.Format(time.RFC3339),
		"expiryDate": expiry.Format(time.DateOnly), "expiryDateTime": expiry.Format(time.RFC3339),
	}
	name := uuid.NewSHA1(uuid.NameSpaceURL, []byte(azureARMKey(rec.ID))).String()
	child := azureReservationChild(rec, typ, name, props)
	return map[string]any{"reservations": []any{map[string]any{"id": child.ID}}}, []*azureARMResource{child}, ""
}

// azureReservationChild builds a reservation of order with its order's
// sku and location.
func azureReservationChild(order *azureARMResource, typ azureARMResourceType, name string, props map[string]any) *azureARMResource {
	child := azureARMChild(order, typ, name, props)
	child.Location, child.Kind, child.SKU = order.Location, "Microsoft.Compute", azureARMMerge(order.SKU, nil)
	return child
}

func azureReservationProperties(rec *azureARMResource) map[string]any {
	state := azureARMString(rec.Properties["provisioningState"])
	if state == "" {
		state = "Succeeded"
	}
	return map[string]any{
		"provisioningState": state, "displayProvisioningState": state, "lastUpdatedDateTime": rec.UpdatedAt.Format(time.RFC3339),
		"userFriendlyAppliedScopeType": rec.Properties["appliedScopeType"], "userFriendlyRenewState": map[bool]string{true: "On", false: "Off"}[rec.Properties["renew"] == true],
	}
}

// azureReservationPatch accepts the update model: scope, instance
// flexibility, display name (as "name"), renewal and review date.
func azureReservationPatch(payload map[string]any) (map[string]any, map[string]any, string) {
	in := azureARMObject(payload["properties"])
	if msg := azureReservationScopeValidate(in); msg != "" {
		return nil, nil, msg
	}
	if raw, present := in["instanceFlexibility"]; present && raw != "On" && raw != "Off" {
		return nil, nil, "properties.instanceFlexibility must be On or Off."
	}
	out := map[string]any{}
	for _, key := range []string{"appliedScopeType", "appliedScopes", "appliedScopeProperties", "instanceFlexibility", "renew", "renewProperties", "reviewDateTime"} {
		if v, present := in[key]; present {
			out[key] = v
		}
	}
	if name, present := in["name"]; present {
		out["displayName"] = name
	}
	// A new scope replaces the old one. PATCH merges nested objects, so
	// clear every scope field the update doesn't set.
	if _, present := in["appliedScopeType"]; present {
		scopeProps := map[string]any{"tenantId": nil, "managementGroupId": nil, "subscriptionId": nil, "resourceGroupId": nil, "displayName": nil}
		for k, v := range azureARMObject(in["appliedScopeProperties"]) {
			scopeProps[k] = v
		}
		out["appliedScopes"], out["appliedScopeProperties"] = in["appliedScopes"], scopeProps
	}
	return map[string]any{"properties": out}, nil, ""
}

func azureReservationAvailableScopes(_ *azureARMResource, payload map[string]any) (int, any) {
	scopes := []any{}
	for _, raw := range azureARMSlice(azureARMObject(payload["properties"])["scopes"]) {
		scope := azureARMString(raw)
		scopes = append(scopes, map[string]any{"scope": scope, "valid": strings.HasPrefix(strings.ToLower(scope), "/subscriptions/")})
	}
	return http.StatusOK, map[string]any{"properties": map[string]any{"scopes": scopes}}
}

// azureReservationActiveLocked finds a reservation of order that is still
// in effect. Caller holds providerStorageMu.
func (s *Server) azureReservationActiveLocked(order *azureARMResource, id string) (*azureARMResource, string) {
	rec := s.azureARMResources[azureARMKey(id)]
	if rec == nil || !strings.HasPrefix(azureARMKey(id), azureARMKey(order.ID+"/reservations/")) {
		return nil, "Reservation " + id + " isn't in reservation order " + order.Name + "."
	}
	if state := rec.Properties["provisioningState"]; state != "Succeeded" {
		return nil, "Reservation " + rec.Name + " is " + azureARMString(state) + "."
	}
	return rec, ""
}

// azureReservationReplaceLocked stores rec with changes merged into its
// properties. Caller holds providerStorageMu.
func (s *Server) azureReservationReplaceLocked(rec *azureARMResource, changes map[string]any) *azureARMResource {
	updated := *rec
	updated.Properties = azureARMMerge(rec.Properties, changes)
	updated.UpdatedAt = time.Now().UTC()
	updated.Properties = azureARMMerge(updated.Properties, azureReservationProperties(&updated))
	s.azureARMResources[azureARMKey(rec.ID)] = &updated
	return &updated
}

// azureReservationListLocked rewrites the order's reservation list after a
// split or merge adds reservations. Caller holds providerStorageMu.
func (s *Server) azureReservationListLocked(order *azureARMResource, added ...*azureARMResource) {
	current := s.azureARMResources[azureARMKey(order.ID)]
	list := slices.Clone(azureARMSlice(current.Properties["reservations"]))
	for _, rec := range added {
		list = append(list, map[string]any{"id": rec.ID})
	}
	s.azureReservationReplaceOrderLocked(current, map[string]any{"reservations": list})
}

func (s *Server) azureReservationReplaceOrderLocked(order *azureARMResource, changes map[string]any) {
	updated := *order
	updated.Properties = azureARMMerge(order.Properties, changes)
	updated.UpdatedAt = time.Now().UTC()
	s.azureARMResources[azureARMKey(order.ID)] = &updated
}

func azureReservationFixtures(records ...*azureARMResource) []any {
	out := make([]any, 0, len(records))
	for _, rec := range records {
		out = append(out, rec.fixture())
	}
	return out
}

// azureReservationSplit splits a reservation into two whose quantities add
// up to its own. The source stays as a Split record.
func (s *Server) azureReservationSplit(order *azureARMResource, payload map[string]any, typ azureARMResourceType) (int, any) {
	props := azureARMObject(payload["properties"])
	quantities := azureARMSlice(props["quantities"])
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	source, msg := s.azureReservationActiveLocked(order, azureARMString(props["reservationId"]))
	if msg != "" {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", msg)
	}
	total, _ := source.Properties["quantity"].(float64)
	if len(quantities) != 2 || !azureWholeNumber(quantities[0], 1, total) || !azureWholeNumber(quantities[1], 1, total) || quantities[0].(float64)+quantities[1].(float64) != total {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "properties.quantities must be two whole numbers of at least 1 that add up to the reservation's quantity, "+strconv.Itoa(int(total))+".")
	}
	parts := make([]*azureARMResource, 0, 2)
	destinations := []any{}
	for i, q := range quantities {
		name := uuid.NewSHA1(uuid.NameSpaceURL, []byte(azureARMKey(source.ID)+"#split"+strconv.Itoa(i))).String()
		part := azureReservationChild(order, typ, name, azureARMMerge(azureReservationBase(source), map[string]any{
			"quantity": q, "splitProperties": map[string]any{"splitSource": source.ID}, "provisioningState": "Succeeded",
		}))
		s.azureARMResources[azureARMKey(part.ID)] = part
		parts = append(parts, part)
		destinations = append(destinations, part.ID)
	}
	source = s.azureReservationReplaceLocked(source, map[string]any{"provisioningState": "Split", "splitProperties": map[string]any{"splitDestinations": destinations}})
	s.azureReservationListLocked(order, parts...)
	return http.StatusOK, azureReservationFixtures(append(parts, source)...)
}

// azureReservationMerge merges reservations of the same SKU, location,
// term and billing plan into one. The sources stay as Merged records.
func (s *Server) azureReservationMerge(order *azureARMResource, payload map[string]any, typ azureARMResourceType) (int, any) {
	ids := azureARMSlice(azureARMObject(payload["properties"])["sources"])
	if len(ids) < 2 {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "properties.sources must name at least two reservations.")
	}
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	sources := make([]*azureARMResource, 0, len(ids))
	total := 0.0
	for _, raw := range ids {
		rec, msg := s.azureReservationActiveLocked(order, azureARMString(raw))
		if msg == "" && slices.Contains(sources, rec) {
			msg = "properties.sources names " + rec.Name + " twice."
		}
		if msg == "" && len(sources) > 0 && !azureReservationsMergeable(sources[0], rec) {
			msg = "Merged reservations need the same SKU, location, term and billing plan."
		}
		if msg != "" {
			return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", msg)
		}
		q, _ := rec.Properties["quantity"].(float64)
		total += q
		sources = append(sources, rec)
	}
	name := uuid.NewSHA1(uuid.NameSpaceURL, []byte(azureARMKey(sources[0].ID)+"#merge")).String()
	merged := azureReservationChild(order, typ, name, azureARMMerge(azureReservationBase(sources[0]), map[string]any{
		"quantity": total, "mergeProperties": map[string]any{"mergeSources": ids}, "provisioningState": "Succeeded",
	}))
	s.azureARMResources[azureARMKey(merged.ID)] = merged
	out := []*azureARMResource{merged}
	for _, rec := range sources {
		out = append(out, s.azureReservationReplaceLocked(rec, map[string]any{"provisioningState": "Merged", "mergeProperties": map[string]any{"mergeDestination": merged.ID}}))
	}
	s.azureReservationListLocked(order, merged)
	return http.StatusOK, azureReservationFixtures(out...)
}

// azureReservationBase copies a reservation's properties for a split or
// merge result, without the source's own split and merge history.
func azureReservationBase(rec *azureARMResource) map[string]any {
	props := azureARMMerge(rec.Properties, nil)
	delete(props, "splitProperties")
	delete(props, "mergeProperties")
	return props
}

func azureReservationsMergeable(a, b *azureARMResource) bool {
	return strings.EqualFold(azureARMString(a.SKU["name"]), azureARMString(b.SKU["name"])) && a.Location == b.Location &&
		a.Properties["term"] == b.Properties["term"] && a.Properties["billingPlan"] == b.Properties["billingPlan"]
}

// Refunds: cancelled commitments in a billing scope can't exceed USD 50,000
// in a rolling 12-month window, and Red Hat and SUSE Linux plans can't be
// refunded.
const azureReservationRefundLimit = 50000.0

var azureReservationNoRefund = []string{"RedHat", "RedHatOsa", "SuseLinux"}

// azureReservationRefundAmount prorates quantity units of rec by the whole
// days left in its term.
func azureReservationRefundAmount(rec *azureARMResource, quantity float64, now time.Time) float64 {
	term := azureARMString(rec.Properties["term"])
	day := func(t time.Time) time.Time { return t.UTC().Truncate(24 * time.Hour) }
	expiry := day(azureReservationExpiry(rec.CreatedAt, term))
	total := expiry.Sub(day(rec.CreatedAt)).Hours()
	left := max(0, expiry.Sub(day(now)).Hours())
	full := azureReservationMonthlyPrice * float64(azureReservationTermMonths[term]) * quantity
	return math.Round(full*left/total*100) / 100
}

// azureReservationLedgerID names the hidden record of a billing scope's
// refunds. Its type isn't registered, so no route serves it.
func azureReservationLedgerID(rec *azureARMResource) string {
	scope := strings.ToLower(azureARMString(rec.Properties["billingScopeId"]))
	return "/providers/Microsoft.Capacity/refundLedgers/" + uuid.NewSHA1(uuid.NameSpaceURL, []byte(scope)).String()
}

// azureReservationRefundedLocked sums the refunds in rec's billing scope in
// the 12 months before now. Caller holds providerStorageMu.
func (s *Server) azureReservationRefundedLocked(rec *azureARMResource, now time.Time) float64 {
	total := 0.0
	if ledger := s.azureARMResources[azureARMKey(azureReservationLedgerID(rec))]; ledger != nil {
		for _, raw := range azureARMSlice(ledger.Properties["refunds"]) {
			entry := azureARMObject(raw)
			at, _ := time.Parse(time.RFC3339, azureARMString(entry["at"]))
			if amount, _ := entry["amount"].(float64); at.After(now.AddDate(-1, 0, 0)) {
				total += amount
			}
		}
	}
	return total
}

// azureReservationRefund prices returning quantity units of a reservation
// and names the session a return must quote. Caller holds providerStorageMu.
func (s *Server) azureReservationRefund(order, rec *azureARMResource, quantity float64, now time.Time) map[string]any {
	usd := func(amount float64) map[string]any { return map[string]any{"currencyCode": "USD", "amount": amount} }
	session := uuid.NewSHA1(uuid.NameSpaceURL, []byte(azureARMKey(rec.ID)+"#"+strconv.Itoa(int(quantity)))).String()
	amount := azureReservationRefundAmount(rec, quantity, now)
	consumed := s.azureReservationRefundedLocked(rec, now)
	errors := []any{}
	if consumed+amount > azureReservationRefundLimit {
		errors = append(errors, map[string]any{"code": "RefundLimitExceeded", "message": "Refunds in this billing scope can't exceed USD 50,000 in 12 months."})
	}
	return map[string]any{
		"id": rec.ID,
		"properties": map[string]any{
			"sessionId": session, "quantity": quantity, "billingRefundAmount": usd(amount), "pricingRefundAmount": usd(amount),
			"policyResult": map[string]any{"properties": map[string]any{
				"consumedRefundsTotal": usd(consumed), "maxRefundLimit": usd(azureReservationRefundLimit), "policyErrors": errors,
			}},
			"billingInformation": map[string]any{
				"billingPlan": order.Properties["billingPlan"], "completedTransactions": 0,
				"totalTransactions": azureReservationTermMonths[azureARMString(rec.Properties["term"])],
			},
		},
	}
}

// azureReservationToReturn reads and checks the reservation and quantity a
// refund request names. Caller holds providerStorageMu.
func (s *Server) azureReservationToReturn(order *azureARMResource, props map[string]any) (*azureARMResource, float64, string) {
	if props["scope"] != "Reservation" {
		return nil, 0, "properties.scope must be Reservation."
	}
	target := azureARMObject(props["reservationToReturn"])
	rec, msg := s.azureReservationActiveLocked(order, azureARMString(target["reservationId"]))
	if msg != "" {
		return nil, 0, msg
	}
	if slices.Contains(azureReservationNoRefund, azureARMString(rec.Properties["reservedResourceType"])) {
		return nil, 0, "Red Hat and SUSE Linux plans can't be refunded."
	}
	held, _ := rec.Properties["quantity"].(float64)
	if !azureWholeNumber(target["quantity"], 1, held) {
		return nil, 0, "properties.reservationToReturn.quantity must be from 1 to the reservation's quantity, " + strconv.Itoa(int(held)) + "."
	}
	return rec, target["quantity"].(float64), ""
}

func (s *Server) azureReservationCalculateRefund(order *azureARMResource, payload map[string]any) (int, any) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	rec, quantity, msg := s.azureReservationToReturn(order, azureARMObject(payload["properties"]))
	if msg != "" {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", msg)
	}
	return http.StatusOK, s.azureReservationRefund(order, rec, quantity, time.Now().UTC())
}

// azureReservationReturn returns units of a reservation, quoting the
// session calculateRefund gave. Returning them all cancels it, and the
// order is cancelled once none of its reservations is in effect.
func (s *Server) azureReservationReturn(order *azureARMResource, payload map[string]any) (int, any) {
	props := azureARMObject(payload["properties"])
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	now := time.Now().UTC()
	rec, quantity, msg := s.azureReservationToReturn(order, props)
	var refund map[string]any
	if msg == "" {
		refund = azureARMObject(s.azureReservationRefund(order, rec, quantity, now)["properties"])
	}
	switch {
	case msg != "":
	case props["sessionId"] != refund["sessionId"]:
		msg = "properties.sessionId must be the sessionId calculateRefund returned for this return."
	case len(azureARMSlice(azureARMObject(azureARMObject(refund["policyResult"])["properties"])["policyErrors"])) > 0:
		msg = "Refunds in this billing scope can't exceed USD 50,000 in 12 months."
	}
	if msg != "" {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", msg)
	}
	s.azureReservationRecordRefundLocked(rec, azureARMObject(refund["billingRefundAmount"])["amount"].(float64), now)
	held, _ := rec.Properties["quantity"].(float64)
	if quantity == held {
		s.azureReservationReplaceLocked(rec, map[string]any{"provisioningState": "Cancelled"})
	} else {
		s.azureReservationReplaceLocked(rec, map[string]any{"quantity": held - quantity})
	}
	prefix := azureARMKey(order.ID + "/reservations/")
	active := false
	for key, other := range s.azureARMResources {
		active = active || (strings.HasPrefix(key, prefix) && other.Properties["provisioningState"] == "Succeeded")
	}
	if !active {
		s.azureReservationReplaceOrderLocked(s.azureARMResources[azureARMKey(order.ID)], map[string]any{"provisioningState": "Cancelled"})
	}
	return http.StatusOK, s.azureARMResources[azureARMKey(order.ID)].fixture()
}

// azureReservationRecordRefundLocked adds a refund to rec's billing scope
// ledger. Caller holds providerStorageMu.
func (s *Server) azureReservationRecordRefundLocked(rec *azureARMResource, amount float64, now time.Time) {
	id := azureReservationLedgerID(rec)
	var refunds []any
	if ledger := s.azureARMResources[azureARMKey(id)]; ledger != nil {
		refunds = slices.Clone(azureARMSlice(ledger.Properties["refunds"]))
	}
	refunds = append(refunds, map[string]any{"amount": amount, "at": now.Format(time.RFC3339)})
	s.azureARMResources[azureARMKey(id)] = &azureARMResource{
		ID: id, Type: "Microsoft.Capacity/refundLedgers", Properties: map[string]any{"refunds": refunds}, CreatedAt: now, UpdatedAt: now,
	}
}

// azureReservationChangeDirectory reports moving the order to another
// tenant. The emulator has one tenant, so nothing moves.
func (s *Server) azureReservationChangeDirectory(order *azureARMResource, payload map[string]any) (int, any) {
	if _, err := uuid.Parse(azureARMString(payload["destinationTenantId"])); err != nil {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "destinationTenantId must be a tenant GUID.")
	}
	prefix := azureARMKey(order.ID + "/reservations/")
	results := []any{}
	for _, rec := range s.azureARMResourcesWhere(func(rec *azureARMResource) bool { return strings.HasPrefix(azureARMKey(rec.ID), prefix) }) {
		results = append(results, map[string]any{"id": rec.ID, "name": rec.Name, "isSucceeded": true})
	}
	return http.StatusOK, map[string]any{
		"reservationOrder": map[string]any{"id": order.ID, "name": order.Name, "isSucceeded": true},
		"reservations":     results,
	}
}

// azureReservationPrice prices a purchase at the flat sample rate and
// gives the reservation order ID to purchase it under.
func azureReservationPrice(payload map[string]any) map[string]any {
	props := azureARMObject(payload["properties"])
	months := azureReservationTermMonths[azureARMString(props["term"])]
	quantity, _ := props["quantity"].(float64)
	total := azureReservationMonthlyPrice * float64(months) * quantity
	usd := map[string]any{"currencyCode": "USD", "amount": total}
	sku := azureARMString(azureARMObject(payload["sku"])["name"])
	return map[string]any{"properties": map[string]any{
		"billingCurrencyTotal": usd, "pricingCurrencyTotal": usd, "netTotal": total, "taxTotal": 0, "grandTotal": total,
		"isTaxIncluded": false, "isBillingPartnerManaged": false, "reservationOrderId": uuid.NewString(),
		"skuTitle": azureARMString(props["reservedResourceType"]) + " " + sku, "skuDescription": sku, "paymentSchedule": []any{},
	}}
}

func azureReservationCatalog(_ *Server, _ string) []map[string]any {
	items := []map[string]any{}
	for _, sku := range []string{"Standard_D2s_v5", "Standard_D4s_v5"} {
		usd := func(months int) map[string]any {
			return map[string]any{"currencyCode": "USD", "amount": azureReservationMonthlyPrice * float64(months)}
		}
		items = append(items, map[string]any{
			"id": "/providers/Microsoft.Capacity/catalogs/" + sku, "resourceType": "VirtualMachines", "name": sku,
			"billingPlans": map[string]any{"P1Y": []any{"Upfront", "Monthly"}, "P3Y": []any{"Upfront", "Monthly"}},
			"terms":        []any{"P1Y", "P3Y"}, "locations": []any{"eastus", "westus"},
			"msrp": map[string]any{"p1Y": usd(12), "p3Y": usd(36)},
		})
	}
	return items
}
