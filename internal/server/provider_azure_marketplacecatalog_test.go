package server

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

const (
	azureMarketplaceProductsVersion = "?api-version=2025-05-01"
	azureMarketplaceKeysVersion     = "?api-version=2023-01-01-preview"
	azureMarketplaceSearchVersion   = "?api-version=2022-02-02"
)

// newAzureMarketplaceTestServer returns do, which sends the tenant header
// unless noTenant is set.
func newAzureMarketplaceTestServer(t *testing.T) (string, func(method, path, body string, noTenant ...bool) *http.Response) {
	t.Helper()
	ts := newAzureResourcesTestServer(t)
	sub := "/azure/subscriptions/" + azureResourcesTestSubscription
	return sub, func(method, path, body string, noTenant ...bool) *http.Response {
		headers := azureResourcesTestHeaders()
		if len(noTenant) == 0 {
			headers["x-ms-client-tenant-id"] = "72f988bf-86f1-41af-91ab-2d7cd011db47"
		}
		var payload []byte
		if body != "" {
			payload = []byte(body)
		}
		return providerContractRequest(t, ts, method, path, payload, headers)
	}
}

func azureMarketplaceIDs(t *testing.T, resp *http.Response) []string {
	t.Helper()
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	var ids []string
	for _, item := range providerContractJSONMap(t, resp)["value"].([]any) {
		ids = append(ids, azureARMString(item.(map[string]any)["uniqueProductId"]))
	}
	return ids
}

func TestAzureMarketplaceCatalogProducts(t *testing.T) {
	t.Parallel()
	sub, do := newAzureMarketplaceTestServer(t)
	billing := "/azure/providers/Microsoft.Billing/billingAccounts/1234:5678_2019-05-31"
	list := func(scope, query string) []string {
		return azureMarketplaceIDs(t, do(http.MethodGet, scope+"/providers/Microsoft.Marketplace/products"+azureMarketplaceProductsVersion+query, ""))
	}
	q := func(name, value string) string { return "&" + name + "=" + url.QueryEscape(value) }

	all := list(sub, "")
	if len(all) != 5 || all[0] != "securescan-enterprise" {
		t.Fatalf("all = %v", all)
	}
	// Every scope lists the catalog; billing profiles too.
	for _, scope := range []string{"/azure", billing, billing + "/billingProfiles/ABCD-EFGH"} {
		if got := list(scope, ""); len(got) != 5 {
			t.Fatalf("%s: %v", scope, got)
		}
	}

	// Learn's sample: a filter, a select and a search.
	resp := do(http.MethodGet, sub+"/providers/Microsoft.Marketplace/products"+azureMarketplaceProductsVersion+
		q("$filter", "productType eq 'SaaS'")+q("$select", "displayName,uniqueProductId,productType,publisherDisplayName"), "")
	body := providerContractJSONMap(t, resp)
	value := body["value"].([]any)
	first := value[0].(map[string]any)
	if len(value) != 2 || len(first) != 4 || first["publisherDisplayName"] != "Example Security, Inc." || body["nextLink"] != nil {
		t.Fatalf("select = %#v", body)
	}
	for _, tc := range []struct {
		query string
		want  []string
	}{
		{q("$filter", "productType eq 'SaaS' or productType eq 'VirtualMachine'"), []string{"securescan-enterprise", "microsoft.managedinstance_standard", "canonical.ubuntu-24_04-lts"}},
		{q("$filter", "operatingSystems eq 'Linux' and productType ne 'SaaS'"), []string{"canonical.ubuntu-24_04-lts", "fabrikam.inventory-api"}},
		{q("$filter", "plan.planId eq 'server-arm64'"), []string{"canonical.ubuntu-24_04-lts"}},
		{q("$filter", "plan.isPrivate eq true"), []string{"contoso.analytics-suite"}},
		{q("$filter", "popularity eq 4.9"), []string{"canonical.ubuntu-24_04-lts"}},
		{q("$filter", "popularity eq 4.90"), []string{"canonical.ubuntu-24_04-lts"}},
		// and binds tighter than or: (A or B) and C would list only the 4.6 product.
		{q("$filter", "productType eq 'SaaS' or productType eq 'VirtualMachine' and popularity eq 4.6"), []string{"securescan-enterprise", "microsoft.managedinstance_standard"}},
		{q("$filter", "displayName eq 'Contoso''s Analytics Suite'"), []string{"contoso.analytics-suite"}},
		// Hidden and stop-sold plans aren't listed, so they don't match.
		{q("$filter", "plan.planId eq 'basic' or plan.planId eq 'legacy'"), []string{}},
		{q("$search", "microsoft"), []string{"microsoft.managedinstance_standard"}},
		{q("$search", "CONTAINER"), []string{"fabrikam.inventory-api"}},
		{q("$orderBy", "displayName desc"), []string{"canonical.ubuntu-24_04-lts", "fabrikam.inventory-api", "securescan-enterprise", "contoso.analytics-suite", "microsoft.managedinstance_standard"}},
		{q("$orderBy", "lastModifiedDateTime"), []string{"contoso.analytics-suite", "fabrikam.inventory-api", "microsoft.managedinstance_standard", "securescan-enterprise", "canonical.ubuntu-24_04-lts"}},
		{q("excludePublicOffersAndPublicPlans", "true"), []string{"contoso.analytics-suite"}},
	} {
		got := list(sub, tc.query)
		if strings.Join(got, ",") != strings.Join(tc.want, ",") {
			t.Fatalf("%s: got %v, want %v", tc.query, got, tc.want)
		}
	}
	// Private plans only, and list plans are PlanSummary.
	private := providerContractJSONMap(t, do(http.MethodGet, sub+"/providers/Microsoft.Marketplace/products"+azureMarketplaceProductsVersion+q("excludePublicOffersAndPublicPlans", "true"), ""))
	plans := private["value"].([]any)[0].(map[string]any)["plans"].([]any)
	if len(plans) != 1 || plans[0].(map[string]any)["planId"] != "negotiated" || plans[0].(map[string]any)["isHidden"] != nil {
		t.Fatalf("private plans = %#v", plans)
	}
	// market (tenant) and locations (billing) narrow the list; neither
	// applies elsewhere.
	if got := list("/azure", q("market", "JP")); strings.Join(got, ",") != "canonical.ubuntu-24_04-lts" {
		t.Fatalf("market = %v", got)
	}
	if got := list(billing, q("locations", "westus2")); strings.Join(got, ",") != "canonical.ubuntu-24_04-lts" {
		t.Fatalf("locations = %v", got)
	}
	// locations is an array: repeated or comma-separated values match any.
	for _, query := range []string{q("locations", "westus2,global"), q("locations", "westus2") + q("locations", "global")} {
		if got := list(billing, query); strings.Join(got, ",") != "securescan-enterprise,canonical.ubuntu-24_04-lts" {
			t.Fatalf("locations %s = %v", query, got)
		}
	}
	if got := list(sub, q("locations", "westus2")+q("market", "JP")); len(got) != 5 {
		t.Fatalf("subscription ignores market and locations = %v", got)
	}

	for _, tc := range []struct{ query, want string }{
		{q("$filter", "plan.vmArchitectureType eq 'Arm64'"), "can't use the field 'plan.vmArchitectureType'"},
		{q("$filter", "contains(displayName, 'x')"), "must be comparisons"},
		{q("$filter", "productType eq 'SaaS' and"), "must be comparisons"},
		{q("$filter", "productType eq 'SaaS' or "), "can't end with 'or'"},
		{q("$filter", "productType eq 'SaaS' productType eq 'VirtualMachine'"), "must be comparisons"},
		{q("$filter", "productType gt 'SaaS'"), "must be comparisons"},
		{q("$filter", "productType eq SaaS"), "must be quoted"},
		{q("$select", "displayName,isHidden"), "$select can't name 'isHidden'"},
		{q("$orderBy", "displayName,productType"), "single field"},
		{q("$orderBy", "popularity"), "$orderBy must be one of"},
		{q("$orderBy", "displayName sideways"), "$orderBy must be one of"},
		{q("language", "xx"), "language must be one of"},
		{q("excludePublicOffersAndPublicPlans", "yes"), "must be true or false"},
	} {
		azureResourcesExpectErrorCode(t, do(http.MethodGet, sub+"/providers/Microsoft.Marketplace/products"+azureMarketplaceProductsVersion+tc.query, ""), http.StatusBadRequest, "BadRequest")
		if msg := azureARMString(providerContractJSONMap(t, do(http.MethodGet, sub+"/providers/Microsoft.Marketplace/products"+azureMarketplaceProductsVersion+tc.query, ""))["error"].(map[string]any)["message"]); !strings.Contains(msg, tc.want) {
			t.Fatalf("%s: message %q, want %q", tc.query, msg, tc.want)
		}
	}
	azureResourcesExpectErrorCode(t, do(http.MethodGet, "/azure/providers/Microsoft.Marketplace/products"+azureMarketplaceProductsVersion+q("market", "USA"), ""), http.StatusBadRequest, "BadRequest")
	azureResourcesExpectStatus(t, do(http.MethodPost, sub+"/providers/Microsoft.Marketplace/products"+azureMarketplaceProductsVersion, ""), http.StatusMethodNotAllowed)
	azureResourcesExpectErrorCode(t, do(http.MethodGet, sub+"/providers/Microsoft.Marketplace/products?api-version=", ""), http.StatusBadRequest, "InvalidApiVersionParameter")
}

func TestAzureMarketplaceCatalogProductGet(t *testing.T) {
	t.Parallel()
	sub, do := newAzureMarketplaceTestServer(t)
	get := func(scope, id, query string) *http.Response {
		return do(http.MethodGet, scope+"/providers/Microsoft.Marketplace/products/"+id+azureMarketplaceProductsVersion+query, "")
	}
	planIDs := func(resp *http.Response) string {
		azureResourcesExpectStatus(t, resp, http.StatusOK)
		var ids []string
		for _, plan := range providerContractJSONMap(t, resp)["plans"].([]any) {
			ids = append(ids, azureARMString(plan.(map[string]any)["planId"]))
		}
		return strings.Join(ids, ",")
	}

	product := providerContractJSONMap(t, get(sub, "SECURESCAN-enterprise", ""))
	if product["uniqueProductId"] != "securescan-enterprise" || product["markets"] != nil || product["serviceInstructionTemplates"] != nil || product["ratingCount"] != float64(245) {
		t.Fatalf("product = %#v", product)
	}
	// Hidden and stop-sold plans need asking for; plan and SKU narrow.
	for _, tc := range []struct{ query, want string }{
		{"", "pro"},
		{"&includeHiddenPlans=true", "pro,basic"},
		{"&includeStopSoldPlans=true", "pro,legacy"},
		{"&includeHiddenPlans=true&includeStopSoldPlans=true", "pro,basic,legacy"},
		{"&includeHiddenPlans=true&planId=basic", "basic"},
		{"&includeStopSoldPlans=true&skuId=legacy-001", "legacy"},
		{"&planId=missing", ""},
	} {
		if got := planIDs(get(sub, "securescan-enterprise", tc.query)); got != tc.want {
			t.Fatalf("%q: plans %q, want %q", tc.query, got, tc.want)
		}
	}
	if templates := providerContractJSONMap(t, get(sub, "securescan-enterprise", "&includeServiceInstructionTemplates=true"))["serviceInstructionTemplates"]; templates == nil {
		t.Fatal("templates not included")
	}
	billing := "/azure/providers/Microsoft.Billing/billingAccounts/1234:5678_2019-05-31"
	azureResourcesExpectStatus(t, get(billing, "canonical.ubuntu-24_04-lts", ""), http.StatusOK)
	azureResourcesExpectStatus(t, get(billing+"/billingProfiles/ABCD-EFGH", "canonical.ubuntu-24_04-lts", ""), http.StatusOK)

	// Tenant scope needs a market the product is sold in.
	tenant := providerContractJSONMap(t, get("/azure", "canonical.ubuntu-24_04-lts", "&market=JP"))
	if tenant["pricingAudience"] != "Direct-Commercial" {
		t.Fatalf("tenant = %#v", tenant)
	}
	if partner := providerContractJSONMap(t, get("/azure", "canonical.ubuntu-24_04-lts", "&market=US&pricingAudience=Partner-Commercial")); partner["pricingAudience"] != "Partner-Commercial" {
		t.Fatalf("partner = %#v", partner)
	}
	azureResourcesExpectErrorCode(t, get("/azure", "securescan-enterprise", "&market=JP"), http.StatusNotFound, "NotFound")
	azureResourcesExpectErrorCode(t, get("/azure", "securescan-enterprise", ""), http.StatusBadRequest, "BadRequest")
	azureResourcesExpectErrorCode(t, get("/azure", "securescan-enterprise", "&market=US&pricingAudience=Retail"), http.StatusBadRequest, "BadRequest")

	azureResourcesExpectErrorCode(t, get(sub, "missing-product", ""), http.StatusNotFound, "NotFound")
	for _, query := range []string{"&includeHiddenPlans=1", "&includeStopSoldPlans=no", "&includeServiceInstructionTemplates=maybe", "&lookUpOfferInTenantLevel=2", "&language=klingon"} {
		azureResourcesExpectErrorCode(t, get(sub, "securescan-enterprise", query), http.StatusBadRequest, "BadRequest")
	}
}

func TestAzureMarketplaceCatalogKeys(t *testing.T) {
	t.Parallel()
	sub, do := newAzureMarketplaceTestServer(t)
	keys := sub + "/providers/Microsoft.Marketplace/keys"
	list := func() []any {
		resp := do(http.MethodGet, keys+azureMarketplaceKeysVersion, "")
		azureResourcesExpectStatus(t, resp, http.StatusOK)
		return providerContractJSONMap(t, resp)["value"].([]any)
	}
	if got := list(); len(got) != 0 {
		t.Fatalf("initial keys = %#v", got)
	}
	created := providerContractJSONMap(t, do(http.MethodPost, keys+"/key-2/create"+azureMarketplaceKeysVersion, ""))
	if created["keyAlias"] != "key-2" || azureARMString(created["key"]) == "" {
		t.Fatalf("created = %#v", created)
	}
	first := providerContractJSONMap(t, do(http.MethodPost, keys+"/key1/create"+azureMarketplaceKeysVersion, ""))
	if first["key"] == created["key"] {
		t.Fatal("keys aren't distinct")
	}
	got := list()
	// By alias: '-' sorts before digits.
	if len(got) != 2 || got[1].(map[string]any)["keyAlias"] != "key1" || got[0].(map[string]any)["key"] != created["key"] {
		t.Fatalf("keys = %#v", got)
	}
	azureResourcesExpectErrorCode(t, do(http.MethodPost, keys+"/key1/create"+azureMarketplaceKeysVersion, ""), http.StatusConflict, "Conflict")
	for _, alias := range []string{"-key", "key-", "key--1", "key_1", "key.1"} {
		azureResourcesExpectErrorCode(t, do(http.MethodPost, keys+"/"+alias+"/create"+azureMarketplaceKeysVersion, ""), http.StatusBadRequest, "BadRequest")
	}

	// Revoking removes the key; the alias can then be reused.
	resp := do(http.MethodPost, keys+"/key1/revoke"+azureMarketplaceKeysVersion, "")
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if got := list(); len(got) != 1 || got[0].(map[string]any)["keyAlias"] != "key-2" {
		t.Fatalf("after revoke = %#v", got)
	}
	azureResourcesExpectErrorCode(t, do(http.MethodPost, keys+"/key1/revoke"+azureMarketplaceKeysVersion, ""), http.StatusNotFound, "NotFound")
	azureResourcesExpectStatus(t, do(http.MethodPost, keys+"/key1/create"+azureMarketplaceKeysVersion, ""), http.StatusOK)

	// Keys belong to their subscription.
	other := "/azure/subscriptions/11111111-1111-1111-1111-111111111111/providers/Microsoft.Marketplace/keys"
	if value := providerContractJSONMap(t, do(http.MethodGet, other+azureMarketplaceKeysVersion, ""))["value"].([]any); len(value) != 0 {
		t.Fatalf("other subscription keys = %#v", value)
	}
	azureResourcesExpectErrorCode(t, do(http.MethodPost, other+"/key-2/revoke"+azureMarketplaceKeysVersion, ""), http.StatusNotFound, "NotFound")

	// The tenant header is required.
	azureResourcesExpectErrorCode(t, do(http.MethodGet, keys+azureMarketplaceKeysVersion, "", true), http.StatusBadRequest, "BadRequest")
	azureResourcesExpectErrorCode(t, do(http.MethodPost, keys+"/key3/create"+azureMarketplaceKeysVersion, "", true), http.StatusBadRequest, "BadRequest")
	azureResourcesExpectErrorCode(t, do(http.MethodPost, keys+"/key-2/revoke"+azureMarketplaceKeysVersion, "", true), http.StatusBadRequest, "BadRequest")
	azureResourcesExpectStatus(t, do(http.MethodGet, keys+"/key-2/create"+azureMarketplaceKeysVersion, ""), http.StatusMethodNotAllowed)
	// Keys have no tenant form.
	azureResourcesExpectStatus(t, do(http.MethodGet, "/azure/providers/Microsoft.Marketplace/keys"+azureMarketplaceKeysVersion, ""), http.StatusNotFound)
}

func TestAzureMarketplaceCatalogSearch(t *testing.T) {
	t.Parallel()
	_, do := newAzureMarketplaceTestServer(t)
	search := func(body string) map[string]any {
		resp := do(http.MethodPost, "/azure/providers/Microsoft.Marketplace/search"+azureMarketplaceSearchVersion, body)
		azureResourcesExpectStatus(t, resp, http.StatusOK)
		return providerContractJSONMap(t, resp)
	}
	legacyIDs := func(got map[string]any) string {
		var ids []string
		for _, item := range got["results"].([]any) {
			ids = append(ids, azureARMString(item.(map[string]any)["legacyId"]))
		}
		return strings.Join(ids, ",")
	}

	// Learn's sample: take and select.
	got := search(`{"searchQuery":"a","take":2,"languages":["en"],"stores":["Ibiza"],"select":["legacyId","offerType"],"privateStoreAllowances":["AllowedByDefault"]}`)
	results := got["results"].([]any)
	if got["totalCount"] != float64(5) || len(results) != 2 || len(results[0].(map[string]any)) != 2 || results[0].(map[string]any)["offerType"] != "SaaS" {
		t.Fatalf("sample = %#v", got)
	}
	for _, tc := range []struct{ body, want string }{
		{`{"languages":["en"],"stores":["AMP"],"offerTypes":["VirtualMachine","ContainerApps"]}`, "canonical.ubuntu-24_04-lts,fabrikam.inventory-api"},
		{`{"languages":["en"],"stores":["AMP"],"publisherName":"microsoft"}`, "microsoft.managedinstance_standard"},
		{`{"languages":["en"],"stores":["AMP"],"pricingTypes":["Byol"],"isThirdParty":true}`, "contoso.analytics-suite,fabrikam.inventory-api"},
		{`{"languages":["en"],"stores":["AMP"],"isMicrosoftProduct":true}`, "microsoft.managedinstance_standard"},
		{`{"languages":["en"],"stores":["AMP"],"isThirdParty":false}`, "microsoft.managedinstance_standard"},
		{`{"languages":["en"],"stores":["AMP"],"isPreferredSolution":true,"isAzureBenefitEligible":true}`, "securescan-enterprise"},
		{`{"languages":["EN"],"stores":["AMP"],"searchQuery":"UBUNTU"}`, "canonical.ubuntu-24_04-lts"},
		{`{"languages":["en"],"stores":["AMP"],"azureCategories":["Databases"]}`, "microsoft.managedinstance_standard"},
		{`{"languages":["en"],"stores":["AMP"],"sortBy":["displayName desc"],"skip":3}`, "contoso.analytics-suite,microsoft.managedinstance_standard"},
		{`{"languages":["en"],"stores":["AMP"],"skip":10}`, ""},
		{`{"languages":["en"],"stores":["AMP"],"privateStoreAllowances":["NonAllowed"]}`, ""},
	} {
		if ids := legacyIDs(search(tc.body)); ids != tc.want {
			t.Fatalf("%s: got %q, want %q", tc.body, ids, tc.want)
		}
	}
	// Facets count each value; plans exclude hidden and stop-sold ones.
	got = search(`{"languages":["de"],"stores":["AMP"],"facets":["offerType"],"searchQuery":"scanner"}`)
	facet := got["facets"].([]any)[0].(map[string]any)
	item := got["results"].([]any)[0].(map[string]any)
	if facet["facetName"] != "offerType" || len(facet["facetValues"].([]any)) != 1 || facet["facetValues"].([]any)[0].(map[string]any)["count"] != float64(1) ||
		item["language"] != "de" || len(item["plans"].([]any)) != 1 || item["isMicrosoftProduct"] != false {
		t.Fatalf("facets = %#v", got)
	}
	counts := map[string]any{}
	for _, v := range search(`{"languages":["en"],"stores":["AMP"],"facets":["offerType"]}`)["facets"].([]any)[0].(map[string]any)["facetValues"].([]any) {
		counts[azureARMString(v.(map[string]any)["value"])] = v.(map[string]any)["count"]
	}
	if counts["SaaS"] != float64(2) || counts["VirtualMachine"] != float64(1) || len(counts) != 4 {
		t.Fatalf("facet counts = %#v", counts)
	}

	for _, tc := range []struct{ body, want string }{
		{`{"stores":["AMP"]}`, "languages is required"},
		{`{"languages":["en"]}`, "stores is required"},
		{`{"languages":["en"],"stores":["Shop"]}`, "stores can't include 'Shop'"},
		{`{"languages":["xx"],"stores":["AMP"]}`, "languages can't include 'xx'"},
		{`{"languages":["en"],"stores":["AMP"],"pricingTypes":["Cheap"]}`, "pricingTypes can't include"},
		{`{"languages":["en"],"stores":["AMP"],"privateStoreAllowances":["Maybe"]}`, "privateStoreAllowances can't include"},
		{`{"languages":["en"],"stores":["AMP"],"select":["secret"]}`, "select can't include"},
		{`{"languages":["en"],"stores":["AMP"],"facets":["secret"]}`, "facets can't include"},
		{`{"languages":["en"],"stores":["AMP"],"take":-1}`, "take must be a non-negative integer"},
		{`{"languages":["en"],"stores":["AMP"],"skip":1.5}`, "skip must be a non-negative integer"},
		{`{"languages":"en","stores":["AMP"]}`, "languages must be an array of strings"},
		{`{"languages":["en"],"stores":["AMP"],"isCoreVm":"yes"}`, "isCoreVm must be a boolean"},
		{`{"languages":["en"],"stores":["AMP"],"searchQuery":5}`, "searchQuery must be a string"},
		{`{"languages":["en"],"stores":["AMP"],"sortBy":["displayName up"]}`, "sortBy can't include"},
	} {
		resp := do(http.MethodPost, "/azure/providers/Microsoft.Marketplace/search"+azureMarketplaceSearchVersion, tc.body)
		body := providerContractJSONMap(t, resp)
		if resp.StatusCode != http.StatusBadRequest || !strings.Contains(azureARMString(body["error"].(map[string]any)["message"]), tc.want) {
			t.Fatalf("%s: %d %#v, want %q", tc.body, resp.StatusCode, body, tc.want)
		}
	}
	azureResourcesExpectErrorCode(t, do(http.MethodPost, "/azure/providers/Microsoft.Marketplace/search"+azureMarketplaceSearchVersion, `{"languages":["en"],"stores":["AMP"]}`, true), http.StatusBadRequest, "BadRequest")
	azureResourcesExpectStatus(t, do(http.MethodGet, "/azure/providers/Microsoft.Marketplace/search"+azureMarketplaceSearchVersion, ""), http.StatusMethodNotAllowed)
}
