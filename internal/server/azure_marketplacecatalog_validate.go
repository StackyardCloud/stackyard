package server

import (
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// azureMarketplaceLanguages are the ISO 639-1 codes Learn lists for language.
var azureMarketplaceLanguages = []string{"en", "cs", "de", "es", "fr", "hu", "it", "ja", "ko", "nl", "pl", "pt-br", "pt-pt", "ru", "sv", "tr", "id", "zh-hans", "zh-hant"}

// azureMarketplaceFilterFields are the fields Learn lets $filter use.
var azureMarketplaceFilterFields = []string{
	"displayName", "productId", "popularity", "categoryIds", "industryIds", "publisherId", "uniqueProductId", "productType",
	"operatingSystems", "pricingTypes", "publisherDisplayName", "longSummary", "summary", "linkedAddinsTypes", "description",
	"supportedProducts", "applicableProducts", "lastModifiedDateTime", "plan.planId", "plan.displayName", "plan.cspState",
	"plan.altStackReference", "plan.stackType", "plan.categoryIds", "plan.hasProtectedArtifacts", "plan.pricingTypes",
	"plan.summary", "plan.description", "plan.skuId", "plan.displayRank", "plan.isPrivate",
}

// azureMarketplaceOrderFields are the fields Learn lets $orderBy use.
var azureMarketplaceOrderFields = []string{"lastModifiedDateTime", "uniqueProductId", "productType", "displayName", "publisherId"}

// azureMarketplaceSummaryFields are ProductSummary's properties, the ones
// a list returns and $select may name.
var azureMarketplaceSummaryFields = []string{
	"applicableProducts", "attributes", "azureBenefit", "badges", "categoryIds", "cspLegalTermsUri", "description", "displayName",
	"hasMarketplaceFootprint", "hasRiPlans", "industryIds", "lastModifiedDateTime", "locations", "longSummary", "offeringType",
	"operatingSystems", "plans", "popularity", "pricingTypes", "privacyPolicyUri", "productFamily", "productId", "productSubType",
	"productType", "publisherDisplayName", "publisherId", "publisherType", "publishingStage", "ratingAverage", "ratingBuckets",
	"ratingCount", "service", "serviceFamily", "smallIconUri", "summary", "supportedProducts", "uniqueProductId",
}

// azureMarketplacePlanSummaryFields are PlanSummary's properties.
var azureMarketplacePlanSummaryFields = []string{
	"altArchitectureReference", "altStackReference", "categoryIds", "cspState", "description", "displayName", "displayRank",
	"hasProtectedArtifacts", "hasRi", "isPrivate", "metadata", "planId", "planType", "pricingTypes", "skuId", "stackType",
	"summary", "uniquePlanId", "vmArchitectureType", "vmSecuritytypes",
}

var (
	azureMarketplaceMarket = regexp.MustCompile(`^[A-Za-z]{2}$`)
	// One comparison and the connective after it: field eq|ne value.
	azureMarketplaceClause = regexp.MustCompile(`(?i)^\s*([A-Za-z.]+)\s+(eq|ne)\s+('(?:[^']|'')*'|[^\s']+)\s*(?:(and|or)\s+|$)`)
)

type azureMarketplaceCond struct {
	field, value string
	negate       bool
}

// azureMarketplaceListOptions is a parsed list query.
type azureMarketplaceListOptions struct {
	filter        [][]azureMarketplaceCond // OR of ANDs
	selected      []string
	orderBy       string
	desc          bool
	search        string
	excludePublic bool
	market        string
	locations     []string
}

func azureMarketplaceLanguage(query url.Values) string {
	if lang := query.Get("language"); lang != "" && !slices.Contains(azureMarketplaceLanguages, strings.ToLower(lang)) {
		return "language must be one of " + strings.Join(azureMarketplaceLanguages, ", ") + "."
	}
	return ""
}

// azureMarketplaceBool reads an optional boolean query parameter.
func azureMarketplaceBool(query url.Values, name string) (bool, string) {
	raw := query.Get(name)
	if raw == "" {
		return false, ""
	}
	if !strings.EqualFold(raw, "true") && !strings.EqualFold(raw, "false") {
		return false, name + " must be true or false."
	}
	return strings.EqualFold(raw, "true"), ""
}

func azureMarketplaceListQuery(scope azureMarketplaceScope, query url.Values) (azureMarketplaceListOptions, string) {
	var q azureMarketplaceListOptions
	if msg := azureMarketplaceLanguage(query); msg != "" {
		return q, msg
	}
	var msg string
	if q.excludePublic, msg = azureMarketplaceBool(query, "excludePublicOffersAndPublicPlans"); msg != "" {
		return q, msg
	}
	if scope.kind == "tenant" {
		if q.market = query.Get("market"); q.market != "" && !azureMarketplaceMarket.MatchString(q.market) {
			return q, "market must be a two-letter market code, such as 'US'."
		}
	}
	if scope.kind == "billing" {
		for _, raw := range query["locations"] {
			for _, loc := range strings.Split(raw, ",") {
				if loc = strings.TrimSpace(loc); loc != "" {
					q.locations = append(q.locations, loc)
				}
			}
		}
	}
	if q.filter, msg = azureMarketplaceParseFilter(query.Get("$filter")); msg != "" {
		return q, msg
	}
	if raw := strings.TrimSpace(query.Get("$select")); raw != "" {
		for _, field := range strings.Split(raw, ",") {
			field = strings.TrimSpace(field)
			if !slices.Contains(azureMarketplaceSummaryFields, field) {
				return q, "$select can't name '" + field + "'."
			}
			q.selected = append(q.selected, field)
		}
	}
	if raw := strings.TrimSpace(query.Get("$orderBy")); raw != "" {
		if strings.Contains(raw, ",") {
			return q, "$orderBy supports ordering by a single field."
		}
		parts := strings.Fields(raw)
		if !slices.Contains(azureMarketplaceOrderFields, parts[0]) || len(parts) > 2 ||
			len(parts) == 2 && !strings.EqualFold(parts[1], "asc") && !strings.EqualFold(parts[1], "desc") {
			return q, "$orderBy must be one of " + strings.Join(azureMarketplaceOrderFields, ", ") + ", optionally followed by asc or desc."
		}
		q.orderBy, q.desc = parts[0], len(parts) == 2 && strings.EqualFold(parts[1], "desc")
	}
	q.search = strings.ToLower(strings.TrimSpace(query.Get("$search")))
	return q, ""
}

// azureMarketplaceParseFilter parses comparisons joined by and/or, with
// and binding tighter; parentheses and functions aren't supported.
func azureMarketplaceParseFilter(raw string) ([][]azureMarketplaceCond, string) {
	if strings.TrimSpace(raw) == "" {
		return nil, ""
	}
	groups := [][]azureMarketplaceCond{{}}
	for rest := raw; strings.TrimSpace(rest) != ""; {
		m := azureMarketplaceClause.FindStringSubmatch(rest)
		if m == nil {
			return nil, "$filter must be comparisons such as productType eq 'SaaS' joined by and or or."
		}
		if !slices.Contains(azureMarketplaceFilterFields, m[1]) {
			return nil, "$filter can't use the field '" + m[1] + "'."
		}
		value := m[3]
		if strings.HasPrefix(value, "'") {
			value = strings.ReplaceAll(value[1:len(value)-1], "''", "'")
		} else if _, err := strconv.ParseFloat(value, 64); err != nil && !strings.EqualFold(value, "true") && !strings.EqualFold(value, "false") {
			return nil, "$filter string literals must be quoted, as in productType eq 'SaaS'."
		}
		last := len(groups) - 1
		groups[last] = append(groups[last], azureMarketplaceCond{field: m[1], value: value, negate: strings.EqualFold(m[2], "ne")})
		// The clause pattern ends at a connective or the end of the filter.
		rest = rest[len(m[0]):]
		if strings.EqualFold(m[4], "or") {
			groups = append(groups, []azureMarketplaceCond{})
		}
		if m[4] != "" && strings.TrimSpace(rest) == "" {
			return nil, "$filter can't end with '" + m[4] + "'."
		}
	}
	return groups, ""
}

// azureMarketplaceText renders a JSON value as a $filter literal compares it.
func azureMarketplaceText(v any) string {
	switch v := v.(type) {
	case string:
		return v
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(v)
	}
	return ""
}

// azureMarketplaceHas reports whether value, a scalar or an array of
// them, is or holds want, ignoring case; numbers compare as numbers.
func azureMarketplaceHas(value any, want string) bool {
	if items, ok := value.([]any); ok {
		return slices.ContainsFunc(items, func(item any) bool { return azureMarketplaceEq(item, want) })
	}
	return value != nil && azureMarketplaceEq(value, want)
}

func azureMarketplaceEq(value any, want string) bool {
	if n, ok := value.(float64); ok {
		w, err := strconv.ParseFloat(want, 64)
		return err == nil && n == w
	}
	return strings.EqualFold(azureMarketplaceText(value), want)
}

func (c azureMarketplaceCond) match(product map[string]any) bool {
	hit := false
	if field, ok := strings.CutPrefix(c.field, "plan."); ok {
		hit = slices.ContainsFunc(azureARMSlice(product["plans"]), func(plan any) bool {
			return azureMarketplaceHas(azureARMObject(plan)[field], c.value)
		})
	} else {
		hit = azureMarketplaceHas(product[c.field], c.value)
	}
	return hit != c.negate
}

// azureMarketplaceProducts returns the catalog products q selects, with
// only the plans a list shows.
func azureMarketplaceProducts(q azureMarketplaceListOptions) []map[string]any {
	var out []map[string]any
	for _, product := range azureMarketplaceCatalog {
		plans := []any{}
		for _, plan := range azureARMSlice(product["plans"]) {
			p := azureARMObject(plan)
			if p["isHidden"] == true || p["isStopSell"] == true || q.excludePublic && p["isPrivate"] != true {
				continue
			}
			plans = append(plans, plan)
		}
		if q.excludePublic && len(plans) == 0 ||
			q.market != "" && !azureMarketplaceHas(product["markets"], q.market) ||
			len(q.locations) > 0 && !slices.ContainsFunc(q.locations, func(loc string) bool { return azureMarketplaceHas(product["locations"], loc) }) ||
			q.search != "" && !slices.ContainsFunc([]string{"displayName", "publisherDisplayName", "summary", "longSummary"}, func(k string) bool {
				return strings.Contains(strings.ToLower(azureARMString(product[k])), q.search)
			}) {
			continue
		}
		candidate := azureARMMerge(product, map[string]any{"plans": plans})
		if q.filter == nil || slices.ContainsFunc(q.filter, func(group []azureMarketplaceCond) bool {
			return !slices.ContainsFunc(group, func(c azureMarketplaceCond) bool { return !c.match(candidate) })
		}) {
			out = append(out, candidate)
		}
	}
	if q.orderBy != "" {
		slices.SortStableFunc(out, func(a, b map[string]any) int {
			c := strings.Compare(strings.ToLower(azureMarketplaceText(a[q.orderBy])), strings.ToLower(azureMarketplaceText(b[q.orderBy])))
			if q.desc {
				return -c
			}
			return c
		})
	}
	return out
}

// project returns the ProductSummary view of product, cut to $select.
func (q azureMarketplaceListOptions) project(product map[string]any) map[string]any {
	fields := q.selected
	if fields == nil {
		fields = azureMarketplaceSummaryFields
	}
	out := map[string]any{}
	for _, field := range fields {
		v, ok := product[field]
		if !ok {
			continue
		}
		if field == "plans" {
			plans := []any{}
			for _, plan := range azureARMSlice(v) {
				p := map[string]any{}
				for _, k := range azureMarketplacePlanSummaryFields {
					if pv, ok := azureARMObject(plan)[k]; ok {
						p[k] = pv
					}
				}
				plans = append(plans, p)
			}
			v = plans
		}
		out[field] = v
	}
	return out
}

// azureMarketplaceGetOptions is a parsed product get query.
type azureMarketplaceGetOptions struct {
	hidden, stopSold, templates bool
	planID, skuID               string
	market, pricingAudience     string
}

func (o azureMarketplaceGetOptions) keepPlan(plan map[string]any) bool {
	return (o.hidden || plan["isHidden"] != true) && (o.stopSold || plan["isStopSell"] != true) &&
		(o.planID == "" || strings.EqualFold(azureARMString(plan["planId"]), o.planID)) &&
		(o.skuID == "" || strings.EqualFold(azureARMString(plan["skuId"]), o.skuID))
}

func azureMarketplaceGetQuery(scope azureMarketplaceScope, query url.Values) (azureMarketplaceGetOptions, string) {
	var o azureMarketplaceGetOptions
	if msg := azureMarketplaceLanguage(query); msg != "" {
		return o, msg
	}
	var msg string
	for _, flag := range []struct {
		name string
		dst  *bool
	}{{"includeHiddenPlans", &o.hidden}, {"includeStopSoldPlans", &o.stopSold}, {"includeServiceInstructionTemplates", &o.templates}} {
		if *flag.dst, msg = azureMarketplaceBool(query, flag.name); msg != "" {
			return o, msg
		}
	}
	if _, msg = azureMarketplaceBool(query, "lookUpOfferInTenantLevel"); msg != "" {
		return o, msg
	}
	o.planID, o.skuID = query.Get("planId"), query.Get("skuId")
	if scope.kind == "tenant" {
		if o.market = query.Get("market"); !azureMarketplaceMarket.MatchString(o.market) {
			return o, "market is required and must be a two-letter market code, such as 'US'."
		}
		o.pricingAudience = query.Get("pricingAudience")
		if o.pricingAudience == "" {
			o.pricingAudience = "Direct-Commercial"
		}
		if o.pricingAudience != "Direct-Commercial" && o.pricingAudience != "Partner-Commercial" {
			return o, "pricingAudience must be Direct-Commercial or Partner-Commercial."
		}
	}
	return o, ""
}

// Search request enumerations from Learn.
var (
	azureMarketplaceStores        = []string{"Appsource", "AMP", "Ibiza", "Cosell", "DakotaDownstream"}
	azureMarketplacePricingTypes  = []string{"Free", "FreeTrial", "Byol", "Payg", "Ri"}
	azureMarketplaceAllowances    = []string{"AllowedByAdmin", "AllowedByDefault", "NonAllowed"}
	azureMarketplaceSearchFields  = []string{"legacyId", "displayName", "publisherDisplayName", "summary", "offerType", "pricingTypes", "operatingSystems", "plans", "language", "azureCategories", "azureIndustries", "isMicrosoftProduct", "isThirdParty", "isPreferredSolution", "isPowerBICertified", "isAdditionalPurchaseRequired", "isAzureBenefitEligible", "isPreview", "smallIconUri"}
	azureMarketplaceSearchStrings = []string{"appSourceCategories", "appSourceIndustries", "azureCategories", "azureIndustries", "azurePortalCategories", "facets", "languages", "offerTypes", "operatingSystems", "pricingTypes", "privateStoreAllowances", "select", "sortBy", "stores"}
	azureMarketplaceSearchFlags   = []string{"isAdditionalPurchaseRequired", "isAzureBenefitEligible", "isCoreVm", "isIndustryCloud", "isMicrosoftProduct", "isPowerBICertified", "isPreferredSolution", "isThirdParty"}
)

// azureMarketplaceSearchValidate checks a SearchRequest's types and values.
func azureMarketplaceSearchValidate(body map[string]any) string {
	for _, name := range azureMarketplaceSearchStrings {
		if v, ok := body[name]; ok && !azureMarketplaceStrings(v) {
			return name + " must be an array of strings."
		}
	}
	for _, name := range azureMarketplaceSearchFlags {
		if v, ok := body[name]; ok {
			if _, isBool := v.(bool); !isBool {
				return name + " must be a boolean."
			}
		}
	}
	for _, name := range []string{"searchQuery", "publisherName"} {
		if v, ok := body[name]; ok {
			if _, isString := v.(string); !isString {
				return name + " must be a string."
			}
		}
	}
	for _, name := range []string{"skip", "take"} {
		if v, ok := body[name]; ok {
			if n, isNum := v.(float64); !isNum || n < 0 || n != float64(int32(n)) {
				return name + " must be a non-negative integer."
			}
		}
	}
	checks := []struct {
		name    string
		allowed []string
	}{
		{"languages", azureMarketplaceLanguages}, {"stores", azureMarketplaceStores}, {"pricingTypes", azureMarketplacePricingTypes},
		{"privateStoreAllowances", azureMarketplaceAllowances}, {"select", azureMarketplaceSearchFields}, {"facets", azureMarketplaceSearchFields},
	}
	for _, c := range checks {
		for _, v := range azureARMSlice(body[c.name]) {
			// Languages ignore case, as the list endpoints' language does.
			if !slices.Contains(c.allowed, v.(string)) && !(c.name == "languages" && slices.Contains(c.allowed, strings.ToLower(v.(string)))) {
				return c.name + " can't include '" + v.(string) + "'; it must be one of " + strings.Join(c.allowed, ", ") + "."
			}
		}
	}
	for _, name := range []string{"languages", "stores"} {
		if len(azureARMSlice(body[name])) == 0 {
			return name + " is required."
		}
	}
	for _, v := range azureARMSlice(body["sortBy"]) {
		parts := strings.Fields(v.(string))
		if len(parts) == 0 || len(parts) > 2 || !slices.Contains(azureMarketplaceSearchFields, parts[0]) ||
			len(parts) == 2 && !strings.EqualFold(parts[1], "asc") && !strings.EqualFold(parts[1], "desc") {
			return "sortBy can't include '" + v.(string) + "'."
		}
	}
	return ""
}

func azureMarketplaceStrings(v any) bool {
	items, ok := v.([]any)
	return ok && !slices.ContainsFunc(items, func(item any) bool { _, isString := item.(string); return !isString })
}

// searchAzureMarketplace runs a SearchRequest over the catalog's public
// plans. ponytail: there is no private store, so every product is
// AllowedByDefault.
func searchAzureMarketplace(body map[string]any) (int, any) {
	if msg := azureMarketplaceSearchValidate(body); msg != "" {
		return http.StatusBadRequest, azureARMErrorBody("BadRequest", msg)
	}
	if allow := azureARMSlice(body["privateStoreAllowances"]); len(allow) > 0 && !slices.Contains(allow, any("AllowedByDefault")) {
		return http.StatusOK, map[string]any{"facets": []any{}, "results": []any{}, "totalCount": 0}
	}
	query := strings.ToLower(azureARMString(body["searchQuery"]))
	var items []map[string]any
	for _, product := range azureMarketplaceProducts(azureMarketplaceListOptions{}) {
		item := azureMarketplaceSearchItem(product, azureARMString(azureARMSlice(body["languages"])[0]))
		if azureMarketplaceSearchMatch(item, body, query) {
			items = append(items, item)
		}
	}
	for _, v := range slices.Backward(azureARMSlice(body["sortBy"])) {
		parts := strings.Fields(v.(string))
		slices.SortStableFunc(items, func(a, b map[string]any) int {
			c := strings.Compare(strings.ToLower(azureMarketplaceText(a[parts[0]])), strings.ToLower(azureMarketplaceText(b[parts[0]])))
			if len(parts) == 2 && strings.EqualFold(parts[1], "desc") {
				return -c
			}
			return c
		})
	}
	facets := []any{}
	for _, name := range azureARMSlice(body["facets"]) {
		facets = append(facets, azureMarketplaceFacet(items, name.(string)))
	}
	total := len(items)
	skip := min(int(azureMarketplaceNumber(body["skip"], 0)), total)
	take := int(azureMarketplaceNumber(body["take"], float64(total)))
	results := []any{}
	for _, item := range items[skip:min(skip+take, total)] {
		if sel := azureARMSlice(body["select"]); len(sel) > 0 {
			picked := map[string]any{}
			for _, f := range sel {
				if v, ok := item[f.(string)]; ok {
					picked[f.(string)] = v
				}
			}
			item = picked
		}
		results = append(results, item)
	}
	return http.StatusOK, map[string]any{"facets": facets, "results": results, "totalCount": total}
}

func azureMarketplaceNumber(v any, fallback float64) float64 {
	if n, ok := v.(float64); ok {
		return n
	}
	return fallback
}

// azureMarketplaceSearchItem maps a product to SearchItemDto.
func azureMarketplaceSearchItem(product map[string]any, language string) map[string]any {
	plans := []any{}
	for _, plan := range azureARMSlice(product["plans"]) {
		p := azureARMObject(plan)
		dto := map[string]any{"legacyPlanId": p["planId"], "displayName": p["displayName"], "cspState": p["cspState"]}
		if arch := p["vmArchitectureType"]; arch != nil {
			dto["architectureType"] = arch
		}
		if gen := azureARMObject(p["metadata"])["generation"]; gen != nil {
			dto["metadata"] = map[string]any{"generation": gen}
		}
		plans = append(plans, dto)
	}
	return map[string]any{
		"legacyId": product["uniqueProductId"], "displayName": product["displayName"], "publisherDisplayName": product["publisherDisplayName"],
		"summary": product["summary"], "offerType": product["productType"], "pricingTypes": product["pricingTypes"],
		"operatingSystems": product["operatingSystems"], "plans": plans, "language": language,
		"azureCategories": product["categoryIds"], "azureIndustries": product["industryIds"], "smallIconUri": product["smallIconUri"],
		"isMicrosoftProduct": product["publisherType"] == "Microsoft", "isThirdParty": product["publisherType"] == "ThirdParty",
		"isPreferredSolution":          azureMarketplaceHas(product["badges"], "PreferredSolution"),
		"isPowerBICertified":           azureMarketplaceHas(product["badges"], "PowerBICertified"),
		"isAdditionalPurchaseRequired": azureMarketplaceHas(product["badges"], "AdditionalPurchaseRequirement"),
		"isAzureBenefitEligible":       product["azureBenefit"] == "Eligible", "isPreview": product["publishingStage"] == "Preview",
	}
}

// azureMarketplaceSearchMatch applies the request's filters to an item.
func azureMarketplaceSearchMatch(item, body map[string]any, query string) bool {
	if query != "" && !slices.ContainsFunc([]string{"displayName", "publisherDisplayName", "summary"}, func(k string) bool {
		return strings.Contains(strings.ToLower(azureARMString(item[k])), query)
	}) {
		return false
	}
	if name := azureARMString(body["publisherName"]); name != "" && !strings.EqualFold(azureARMString(item["publisherDisplayName"]), name) {
		return false
	}
	for field, itemField := range map[string]string{"offerTypes": "offerType", "pricingTypes": "pricingTypes", "operatingSystems": "operatingSystems", "azureCategories": "azureCategories", "azureIndustries": "azureIndustries"} {
		if want := azureARMSlice(body[field]); len(want) > 0 && !slices.ContainsFunc(want, func(v any) bool { return azureMarketplaceHas(item[itemField], v.(string)) }) {
			return false
		}
	}
	for _, flag := range azureMarketplaceSearchFlags {
		if want, ok := body[flag].(bool); ok && item[flag] != nil && item[flag] != want {
			return false
		}
	}
	return true
}

// azureMarketplaceFacet counts the items by each value of field.
func azureMarketplaceFacet(items []map[string]any, field string) map[string]any {
	counts := map[string]int{}
	var order []string
	for _, item := range items {
		values := azureARMSlice(item[field])
		if values == nil && item[field] != nil {
			values = []any{item[field]}
		}
		for _, v := range values {
			text := azureMarketplaceText(v)
			if text == "" {
				continue
			}
			if counts[text] == 0 {
				order = append(order, text)
			}
			counts[text]++
		}
	}
	values := []any{}
	for _, v := range order {
		values = append(values, map[string]any{"value": v, "count": counts[v]})
	}
	return map[string]any{"facetName": field, "facetValues": values}
}
