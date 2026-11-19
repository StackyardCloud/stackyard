package server

import (
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

var (
	azureBBAliasPattern = regexp.MustCompile(`^[a-zA-Z0-9_\-.]+$`)
	// Savings plans run one or three years; reservations also one month
	// or five years.
	azureBBTermYears        = map[string]int{"P1Y": 1, "P3Y": 3}
	azureBBReservationTerms = []string{"P1M", "P1Y", "P3Y", "P5Y"}
	azureBBCurrency         = regexp.MustCompile(`^[A-Z]{3}$`)
	// billingScopeId is a subscription or an MCA/EA billing subscription.
	azureBBBillingScope  = regexp.MustCompile(`(?i)^(/providers/Microsoft\.Billing/billingAccounts/[^/]+/billingSubscriptions|/subscriptions)/([0-9a-f-]{36})$`)
	azureBBResourceGroup = regexp.MustCompile(`(?i)^/subscriptions/[0-9a-f-]{36}/resourceGroups/[^/]+$`)
	// After purchase only these change; commitment, term and billing
	// frequency are fixed.
	azureBBUpdatable = []string{"appliedScopeType", "appliedScopeProperties", "displayName", "renew", "renewProperties"}
	// One $filter comparison, optionally parenthesized, and the connective
	// after it. Only eq, and and or are supported.
	azureBBClause = regexp.MustCompile(`(?i)^\s*\(?\s*([A-Za-z/]+)\s+(\w+)\s+('(?:[^']|'')*'|[^\s')]+)\s*\)?\s*(?:(and|or)\s+|$)`)
	// azureBBListFields are the properties $filter and $orderby name.
	azureBBListFields = []string{"sku/name", "properties/appliedScopeType", "properties/archived", "properties/displayName",
		"properties/displayProvisioningState", "properties/effectiveDateTime", "properties/expiryDate", "properties/provisioningState",
		"properties/quantity", "properties/renew", "properties/reservedResourceType", "properties/term",
		"properties/userFriendlyAppliedScopeType", "properties/userFriendlyRenewState"}
)

func azureBBAliasName(name string) string {
	if !azureBBAliasPattern.MatchString(name) {
		return "Alias names may contain only letters, digits, underscores, hyphens and periods."
	}
	return ""
}

func azureBBGUID(s string) bool {
	_, err := uuid.Parse(s)
	return err == nil && len(s) == 36
}

// azureBBPurchaseValidate checks a savings plan purchase. partial checks
// only what's present, for renewal purchases, which inherit the rest from
// the expiring plan.
func azureBBPurchaseValidate(payload map[string]any, partial bool) string {
	props := azureARMObject(payload["properties"])
	if !partial && strings.TrimSpace(azureARMString(azureARMObject(payload["sku"])["name"])) == "" {
		return "sku.name is required, such as Compute_Savings_Plan."
	}
	if raw, present := props["term"]; present || !partial {
		if _, ok := azureBBTermYears[azureARMString(raw)]; !ok {
			return "properties.term must be P1Y or P3Y: savings plans run one or three years."
		}
	}
	if raw, present := props["billingPlan"]; present && raw != nil && raw != "P1M" {
		return "properties.billingPlan must be P1M for monthly payments, or omitted to pay up front."
	}
	if raw, present := props["billingScopeId"]; present || !partial {
		if msg := azureBBBillingScopeValidate(raw); msg != "" {
			return msg
		}
	}
	if raw, present := props["commitment"]; present || !partial {
		if msg := azureBBCommitmentValidate(raw); msg != "" {
			return msg
		}
	}
	if raw, present := props["appliedScopeType"]; present || !partial {
		if raw == nil {
			return "properties.appliedScopeType is required."
		}
		if msg := azureBBScopeValidate(props); msg != "" {
			return msg
		}
	}
	return azureBBCommonValidate(props)
}

// azureBBBillingScopeValidate checks the subscription a purchase bills.
func azureBBBillingScopeValidate(raw any) string {
	if m := azureBBBillingScope.FindStringSubmatch(azureARMString(raw)); m == nil || !azureBBGUID(m[2]) {
		return "properties.billingScopeId must be /subscriptions/{subscriptionId} or /providers/Microsoft.Billing/billingAccounts/{accountId}/billingSubscriptions/{subscriptionId}."
	}
	return ""
}

// azureBBCommitmentValidate checks a savings plan's hourly commitment.
func azureBBCommitmentValidate(raw any) string {
	c, ok := raw.(map[string]any)
	if !ok {
		return "properties.commitment is required."
	}
	if c["grain"] != "Hourly" {
		return "properties.commitment.grain must be Hourly: a savings plan commits to a fixed hourly spend."
	}
	if amount, ok := c["amount"].(float64); !ok || amount <= 0 {
		return "properties.commitment.amount must be a number greater than 0."
	}
	if !azureBBCurrency.MatchString(azureARMString(c["currencyCode"])) {
		return "properties.commitment.currencyCode must be an ISO 4217 code, such as USD."
	}
	return ""
}

// azureBBCommonValidate checks the fields purchases and updates share.
func azureBBCommonValidate(props map[string]any) string {
	if raw, present := props["displayName"]; present && strings.TrimSpace(azureARMString(raw)) == "" {
		return "properties.displayName must be a non-empty string."
	}
	if raw, present := props["renew"]; present {
		if _, ok := raw.(bool); !ok {
			return "properties.renew must be true or false."
		}
	}
	return ""
}

// azureBBScopeValidate checks that the applied scope type and its
// properties agree: Single needs a subscription or a resource group,
// ManagementGroup a management group and its tenant, and Shared neither.
func azureBBScopeValidate(props map[string]any) string {
	scope := azureARMObject(props["appliedScopeProperties"])
	sub, rg := azureARMString(scope["subscriptionId"]), azureARMString(scope["resourceGroupId"])
	if azureBBGUID(sub) {
		sub = "/subscriptions/" + sub
	}
	mg := azureARMString(scope["managementGroupId"])
	switch props["appliedScopeType"] {
	case "Single":
		if sub == "" && rg == "" {
			return "A Single scope needs appliedScopeProperties.subscriptionId or resourceGroupId."
		}
		if sub != "" && !azureBBGUID(strings.TrimPrefix(strings.ToLower(sub), "/subscriptions/")) {
			return "appliedScopeProperties.subscriptionId must be a subscription ID or /subscriptions/{subscriptionId}."
		}
		if rg != "" && !azureBBResourceGroup.MatchString(rg) {
			return "appliedScopeProperties.resourceGroupId must be /subscriptions/{subscriptionId}/resourceGroups/{name}."
		}
		if sub != "" && rg != "" && !strings.HasPrefix(strings.ToLower(rg), strings.ToLower(sub)+"/") {
			return "appliedScopeProperties.resourceGroupId must be in subscriptionId."
		}
	case "ManagementGroup":
		if !strings.HasPrefix(strings.ToLower(mg), "/providers/microsoft.management/managementgroups/") || !azureBBGUID(azureARMString(scope["tenantId"])) {
			return "A ManagementGroup scope needs appliedScopeProperties.managementGroupId (/providers/Microsoft.Management/managementGroups/{id}) and tenantId."
		}
	case "Shared":
		if sub != "" || rg != "" || mg != "" {
			return "A Shared scope takes no subscription, resource group or management group."
		}
	default:
		return "properties.appliedScopeType must be Single, Shared or ManagementGroup."
	}
	return ""
}

// azureBBUpdateValidate checks a savings plan update: only the scope,
// display name and renewal settings change after purchase.
func azureBBUpdateValidate(in map[string]any) string {
	for key := range in {
		if !slices.Contains(azureBBUpdatable, key) {
			return "properties." + key + " can't be updated: after purchase only the scope, display name and renewal settings change."
		}
	}
	if _, present := in["appliedScopeType"]; present {
		if msg := azureBBScopeValidate(in); msg != "" {
			return msg
		}
	} else if _, present := in["appliedScopeProperties"]; present {
		return "properties.appliedScopeType is required with appliedScopeProperties."
	}
	if raw, present := in["renewProperties"]; present && raw != nil {
		purchase, ok := azureARMObject(raw)["purchaseProperties"].(map[string]any)
		if !ok {
			return "properties.renewProperties.purchaseProperties is required."
		}
		if raw, present := in["renew"]; present && raw != true {
			return "properties.renewProperties needs renew set to true."
		}
		if msg := azureBBPurchaseValidate(purchase, true); msg != "" {
			return "renewProperties.purchaseProperties: " + msg
		}
	}
	return azureBBCommonValidate(in)
}

// azureBBPlanPatch accepts the savings plan update model. A new scope
// replaces the old one: PATCH merges nested objects, so every scope field
// the update doesn't set is cleared, and Shared clears them all.
func azureBBPlanPatch(payload map[string]any) (map[string]any, map[string]any, string) {
	in, ok := payload["properties"].(map[string]any)
	if raw, present := payload["properties"]; present && raw != nil && !ok {
		return nil, nil, "properties must be an object."
	}
	if msg := azureBBUpdateValidate(in); msg != "" {
		return nil, nil, msg
	}
	out := map[string]any{}
	for _, key := range []string{"displayName", "renew", "renewProperties"} {
		if v, present := in[key]; present {
			out[key] = v
		}
	}
	if scopeType, present := in["appliedScopeType"]; present {
		out["appliedScopeType"] = scopeType
		out["appliedScopeProperties"] = nil
		if scopeType != "Shared" {
			scope := map[string]any{"tenantId": nil, "managementGroupId": nil, "subscriptionId": nil, "resourceGroupId": nil, "displayName": nil, "benefitFilter": nil}
			for k, v := range azureARMObject(in["appliedScopeProperties"]) {
				scope[k] = v
			}
			out["appliedScopeProperties"] = scope
		}
	}
	return map[string]any{"properties": out}, nil, ""
}

// azureBBReservationValidate checks a reservation purchase.
func azureBBReservationValidate(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	switch {
	case strings.TrimSpace(azureARMString(azureARMObject(payload["sku"])["name"])) == "":
		return "sku.name is required, such as Standard_D2s_v5."
	case !slices.Contains(azureReservationResourceTypes, azureARMString(props["reservedResourceType"])):
		return "properties.reservedResourceType must be a reservable resource type, such as VirtualMachines."
	case !slices.Contains(azureBBReservationTerms, azureARMString(props["term"])):
		return "properties.term must be P1M, P1Y, P3Y or P5Y."
	case !azureWholeNumber(props["quantity"], 1, 2147483647):
		return "properties.quantity must be a whole number of at least 1."
	case azureBBBillingScopeValidate(props["billingScopeId"]) != "":
		return azureBBBillingScopeValidate(props["billingScopeId"])
	}
	if raw, present := props["billingPlan"]; present && raw != nil && raw != "P1M" {
		return "properties.billingPlan must be P1M for monthly payments, or omitted to pay up front."
	}
	if raw, present := props["renewAction"]; present && raw != "CreateNew" && raw != "RenewExisting" {
		return "properties.renewAction must be CreateNew or RenewExisting."
	}
	if raw, present := azureARMObject(props["reservedResourceProperties"])["instanceFlexibility"]; present && raw != "On" && raw != "Off" {
		return "properties.reservedResourceProperties.instanceFlexibility must be On or Off."
	}
	if props["appliedScopeType"] == nil {
		return "properties.appliedScopeType is required."
	}
	if msg := azureBBScopeValidate(props); msg != "" {
		return msg
	}
	return azureBBCommonValidate(props)
}

type azureBBCond struct{ field, value string }

// azureBBListOptions is a parsed tenant savings plan list query.
type azureBBListOptions struct {
	filter     [][]azureBBCond // OR of ANDs
	orderBy    string
	desc       bool
	skip, take int
	state      string
}

// azureBBMaxPageIndex bounds take and $skiptoken so paging math can't
// overflow.
const azureBBMaxPageIndex = 1_000_000

func azureBBListQuery(query url.Values) (azureBBListOptions, string) {
	var q azureBBListOptions
	var msg string
	if q.filter, msg = azureBBParseFilter(query.Get("$filter")); msg != "" {
		return q, msg
	}
	if raw := strings.TrimSpace(query.Get("$orderby")); raw != "" {
		field, dir, _ := strings.Cut(raw, " ")
		dir = strings.ToLower(strings.TrimSpace(dir))
		if !slices.Contains(azureBBListFields, field) || (dir != "" && dir != "asc" && dir != "desc") {
			return q, "$orderby must be a savings plan property, such as properties/displayName, then asc or desc."
		}
		q.orderBy, q.desc = field, dir == "desc"
	}
	for name, target := range map[string]*int{"$skiptoken": &q.skip, "take": &q.take} {
		if raw := query.Get(name); raw != "" {
			n, err := strconv.Atoi(raw)
			if err != nil || n < 0 || n > azureBBMaxPageIndex || (name == "take" && n < 1) {
				return q, name + " must be a whole number" + map[bool]string{true: " from 1", false: " from 0"}[name == "take"] + " to " + strconv.Itoa(azureBBMaxPageIndex) + "."
			}
			*target = n
		}
	}
	if raw := query.Get("refreshSummary"); raw != "" && !strings.EqualFold(raw, "true") && !strings.EqualFold(raw, "false") {
		return q, "refreshSummary must be true or false."
	}
	q.state = query.Get("selectedState")
	return q, ""
}

func azureBBParseFilter(raw string) ([][]azureBBCond, string) {
	if strings.TrimSpace(raw) == "" {
		return nil, ""
	}
	groups := [][]azureBBCond{{}}
	for rest := raw; strings.TrimSpace(rest) != ""; {
		m := azureBBClause.FindStringSubmatch(rest)
		switch {
		case m == nil:
			return nil, "$filter must be comparisons such as properties/displayName eq 'name' joined by and or or."
		case !strings.EqualFold(m[2], "eq"):
			return nil, "$filter supports only eq, and and or, not '" + m[2] + "'."
		case !slices.Contains(azureBBListFields, m[1]):
			return nil, "$filter can't use the property '" + m[1] + "'."
		case strings.Count(strings.Replace(m[0], m[3], "", 1), "(") != strings.Count(strings.Replace(m[0], m[3], "", 1), ")"):
			// ponytail: parentheses may enclose one comparison only; grouping
			// several needs a real expression parser.
			return nil, "$filter parentheses may enclose only a single comparison."
		}
		value := m[3]
		if strings.HasPrefix(value, "'") {
			value = strings.ReplaceAll(value[1:len(value)-1], "''", "'")
		}
		groups[len(groups)-1] = append(groups[len(groups)-1], azureBBCond{field: m[1], value: value})
		rest = rest[len(m[0]):]
		if strings.EqualFold(m[4], "or") {
			groups = append(groups, []azureBBCond{})
		}
		if m[4] != "" && strings.TrimSpace(rest) == "" {
			return nil, "$filter can't end with '" + m[4] + "'."
		}
	}
	return groups, ""
}

// azureBBField reads a list field of a savings plan. Savings plans are
// never archived.
func azureBBField(rec *azureARMResource, field string) any {
	if field == "sku/name" {
		return rec.SKU["name"]
	}
	name := strings.TrimPrefix(field, "properties/")
	switch name {
	case "expiryDate":
		name = "expiryDateTime"
	case "provisioningState":
		name = "displayProvisioningState"
	case "userFriendlyRenewState":
		return map[bool]string{true: "On", false: "Off"}[rec.Properties["renew"] == true]
	}
	if v, ok := rec.Properties[name]; ok {
		return v
	}
	if name == "archived" {
		return false
	}
	return nil
}

func (q azureBBListOptions) matches(rec *azureARMResource) bool {
	if len(q.filter) == 0 {
		return true
	}
	return slices.ContainsFunc(q.filter, func(group []azureBBCond) bool {
		for _, c := range group {
			if v := azureBBField(rec, c.field); v == nil || !azureMarketplaceEq(v, c.value) {
				return false
			}
		}
		return true
	})
}

// sort orders plans by $orderby, then by ID, so pages are stable.
func (q azureBBListOptions) sort(plans []*azureARMResource) {
	slices.SortStableFunc(plans, func(a, b *azureARMResource) int {
		if q.orderBy != "" {
			if c := strings.Compare(azureMarketplaceText(azureBBField(a, q.orderBy)), azureMarketplaceText(azureBBField(b, q.orderBy))); c != 0 {
				if q.desc {
					return -c
				}
				return c
			}
		}
		return strings.Compare(azureARMKey(a.ID), azureARMKey(b.ID))
	})
}
