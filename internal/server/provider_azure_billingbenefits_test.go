package server

import (
	"math"
	"net/http"
	"strings"
	"testing"
	"time"
)

const (
	azureBBTestAPI  = "?api-version=2026-06-01"
	azureBBTestRoot = "/azure/providers/Microsoft.BillingBenefits"
	azureBBTestSub  = "/subscriptions/30000000-0000-0000-0000-000000000000"
)

// azureBBPurchase is Learn's SavingsPlanOrderAliasCreate sample with a
// display name.
func azureBBPurchase(name, plan string) string {
	billing := ""
	if plan != "" {
		billing = `"billingPlan":"` + plan + `",`
	}
	return `{"sku":{"name":"Compute_Savings_Plan"},"properties":{"displayName":"` + name + `","appliedScopeType":"Shared",` + billing +
		`"billingScopeId":"` + azureBBTestSub + `","commitment":{"amount":0.001,"currencyCode":"USD","grain":"Hourly"},"term":"P3Y"}}`
}

// azureBBBuy purchases a savings plan and returns its order and plan paths.
func azureBBBuy(t *testing.T, do func(method, path, body string) *http.Response, alias, name, plan string) (string, string) {
	t.Helper()
	resp := do(http.MethodPut, azureBBTestRoot+"/savingsPlanOrderAliases/"+alias+azureBBTestAPI, azureBBPurchase(name, plan))
	if resp.StatusCode != http.StatusCreated || resp.Header.Get("Azure-AsyncOperation") == "" || resp.Header.Get("Retry-After") != "5" {
		t.Fatalf("purchase: status %d, headers %v", resp.StatusCode, resp.Header)
	}
	body := providerContractJSONMap(t, resp)
	order := azureARMString(body["properties"].(map[string]any)["savingsPlanOrderId"])
	got := providerContractJSONMap(t, do(http.MethodGet, "/azure"+order+azureBBTestAPI, ""))
	return "/azure" + order, "/azure" + azureARMString(got["properties"].(map[string]any)["savingsPlans"].([]any)[0])
}

func TestAzureBillingBenefitsPurchase(t *testing.T) {
	t.Parallel()
	_, do := newAzureDatabaseTestServer(t, "rg-bb")
	aliasPath := azureBBTestRoot + "/savingsPlanOrderAliases/spAlias123" + azureBBTestAPI
	resp := do(http.MethodPut, aliasPath, azureBBPurchase("Compute_SavingsPlan", "P1M"))
	operation := resp.Header.Get("Azure-AsyncOperation")
	alias := providerContractJSONMap(t, resp)
	if alias["properties"].(map[string]any)["provisioningState"] != "Created" || alias["type"] != "Microsoft.BillingBenefits/savingsPlanOrderAliases" {
		t.Fatalf("unexpected alias: %#v", alias)
	}
	if status := providerContractJSONMap(t, do(http.MethodGet, operation, "")); status["status"] != "Succeeded" {
		t.Fatalf("the purchase operation should be done: %#v", status)
	}
	if got := providerContractJSONMap(t, do(http.MethodGet, aliasPath, "")); got["properties"].(map[string]any)["savingsPlanOrderId"] == nil {
		t.Fatalf("unexpected alias GET: %#v", got)
	}
	// An alias purchases once, and aliases are never deleted or patched.
	azureResourcesExpectErrorCode(t, do(http.MethodPut, aliasPath, azureBBPurchase("again", "")), http.StatusBadRequest, "InvalidRequestContent")
	azureResourcesExpectStatus(t, do(http.MethodDelete, aliasPath, ""), http.StatusMethodNotAllowed)

	order := alias["properties"].(map[string]any)["savingsPlanOrderId"].(string)
	got := providerContractJSONMap(t, do(http.MethodGet, "/azure"+order+azureBBTestAPI, ""))
	props := got["properties"].(map[string]any)
	if props["provisioningState"] != "Succeeded" || props["billingPlan"] != "P1M" || props["term"] != "P3Y" || props["planInformation"] != nil {
		t.Fatalf("unexpected order: %#v", got)
	}
	planPath := "/azure" + props["savingsPlans"].([]any)[0].(string)
	plan := providerContractJSONMap(t, do(http.MethodGet, planPath+azureBBTestAPI, ""))
	pp := plan["properties"].(map[string]any)
	start, _ := time.Parse(time.RFC3339, pp["effectiveDateTime"].(string))
	expiry, _ := time.Parse(time.RFC3339, pp["expiryDateTime"].(string))
	if !expiry.Equal(start.AddDate(3, 0, 0)) || pp["renew"] != false || pp["userFriendlyAppliedScopeType"] != "Shared" || plan["sku"].(map[string]any)["name"] != "Compute_Savings_Plan" {
		t.Fatalf("unexpected plan: %#v", plan)
	}
	if n := len(providerContractJSONMap(t, do(http.MethodGet, "/azure"+order+"/savingsPlans"+azureBBTestAPI, ""))["value"].([]any)); n != 1 {
		t.Fatalf("expected the order's one plan, got %d", n)
	}
	if n := len(providerContractJSONMap(t, do(http.MethodGet, azureBBTestRoot+"/savingsPlanOrders"+azureBBTestAPI, ""))["value"].([]any)); n != 1 {
		t.Fatalf("expected one order, got %d", n)
	}
	// Orders and plans come only from purchases.
	azureResourcesExpectErrorCode(t, do(http.MethodPut, "/azure"+order+azureBBTestAPI, azureBBPurchase("x", "")), http.StatusBadRequest, "InvalidRequestContent")
	azureResourcesExpectErrorCode(t, do(http.MethodPut, "/azure"+order+"/savingsPlans/extra"+azureBBTestAPI, `{"properties":{}}`), http.StatusBadRequest, "InvalidRequestContent")
	azureResourcesExpectStatus(t, do(http.MethodDelete, planPath+azureBBTestAPI, ""), http.StatusMethodNotAllowed)

	elevated := providerContractJSONMap(t, do(http.MethodPost, "/azure"+order+"/elevate"+azureBBTestAPI, ""))
	if ep := elevated["properties"].(map[string]any); ep["scope"] != order || !strings.HasSuffix(ep["roleDefinitionId"].(string), "/roleDefinitions/"+azureBBOwnerRole) {
		t.Fatalf("elevate should grant Owner on the order: %#v", elevated)
	}
	azureResourcesExpectStatus(t, do(http.MethodGet, azureBBTestRoot+"/savingsPlanOrders/00000000-0000-0000-0000-000000000009"+azureBBTestAPI+"&$expand=schedule", ""), http.StatusNotFound)
}

func TestAzureBillingBenefitsSchedule(t *testing.T) {
	t.Parallel()
	_, do := newAzureDatabaseTestServer(t, "rg-bb")
	monthly, _ := azureBBBuy(t, do, "monthly", "m", "P1M")
	upfront, _ := azureBBBuy(t, do, "upfront", "u", "")
	got := providerContractJSONMap(t, do(http.MethodGet, monthly+azureBBTestAPI+"&$expand=schedule", ""))
	info := got["properties"].(map[string]any)["planInformation"].(map[string]any)
	txs := info["transactions"].([]any)
	first, second := txs[0].(map[string]any), txs[1].(map[string]any)
	// Learn's sample: USD 0.001 an hour for three years is 26.28, or 36
	// payments of 0.73.
	if info["pricingCurrencyTotal"].(map[string]any)["amount"] != 26.28 || len(txs) != 36 || first["pricingCurrencyTotal"].(map[string]any)["amount"] != 0.73 {
		t.Fatalf("unexpected monthly schedule: %#v", info)
	}
	if first["status"] != "Succeeded" || first["paymentDate"] == nil || second["status"] != "Scheduled" || info["nextPaymentDueDate"] != second["dueDate"] {
		t.Fatalf("only the first payment is due at purchase: %#v %#v %#v", first, second, info["nextPaymentDueDate"])
	}
	sum := 0.0
	for _, tx := range txs {
		sum += tx.(map[string]any)["pricingCurrencyTotal"].(map[string]any)["amount"].(float64)
	}
	if math.Abs(sum-26.28) > 1e-9 || txs[35].(map[string]any)["pricingCurrencyTotal"].(map[string]any)["amount"] != 0.73 {
		t.Fatalf("payments must add up to the total, got %v", sum)
	}
	info = providerContractJSONMap(t, do(http.MethodGet, upfront+azureBBTestAPI+"&$expand=schedule", ""))["properties"].(map[string]any)["planInformation"].(map[string]any)
	if txs := info["transactions"].([]any); len(txs) != 1 || txs[0].(map[string]any)["pricingCurrencyTotal"].(map[string]any)["amount"] != 26.28 || info["nextPaymentDueDate"] != nil {
		t.Fatalf("an upfront plan is paid in one payment: %#v", info)
	}
}

func TestAzureBillingBenefitsScheduleMath(t *testing.T) {
	t.Parallel()
	// 0.0033 an hour for three years is 86.72; 35 payments of 2.40 and a
	// last one of 2.72 add up to it exactly.
	plan := &azureARMResource{CreatedAt: time.Date(2025, 1, 31, 0, 0, 0, 0, time.UTC), Properties: map[string]any{
		"commitment": map[string]any{"amount": 0.0033, "currencyCode": "USD"}, "term": "P3Y", "billingPlan": "P1M",
	}}
	info := azureBBSchedule(plan, plan.CreatedAt)
	txs := info["transactions"].([]any)
	var cents int64
	for _, tx := range txs {
		cents += int64(math.Round(tx.(map[string]any)["pricingCurrencyTotal"].(map[string]any)["amount"].(float64) * 100))
	}
	if info["pricingCurrencyTotal"].(map[string]any)["amount"] != 86.72 || cents != 8672 {
		t.Fatalf("total %v, payments add to %d cents", info["pricingCurrencyTotal"], cents)
	}
	// Month-end purchases pay on the last day of shorter months.
	if due := txs[1].(map[string]any)["dueDate"]; due != "2025-02-28" {
		t.Fatalf("Jan 31 + 1 month = %v, want 2025-02-28", due)
	}
	if due := txs[13].(map[string]any)["dueDate"]; due != "2026-02-28" {
		t.Fatalf("Jan 31 + 13 months = %v, want 2026-02-28", due)
	}
	if due := azureBBAddMonths(time.Date(2027, 12, 31, 0, 0, 0, 0, time.UTC), 2).Format(time.DateOnly); due != "2028-02-29" {
		t.Fatalf("leap February: %s", due)
	}
}

func TestAzureBillingBenefitsPurchaseValidation(t *testing.T) {
	t.Parallel()
	_, do := newAzureDatabaseTestServer(t, "rg-bb")
	valid := azureBBPurchase("v", "P1M")
	for name, edit := range map[string][2]string{
		"term":            {`"term":"P3Y"`, `"term":"P5Y"`},
		"grain":           {`"grain":"Hourly"`, `"grain":"FullTerm"`},
		"amount":          {`"amount":0.001`, `"amount":0`},
		"currency":        {`"currencyCode":"USD"`, `"currencyCode":"usd"`},
		"billing plan":    {`"billingPlan":"P1M"`, `"billingPlan":"Upfront"`},
		"billing scope":   {azureBBTestSub, "/subscriptions/not-a-guid"},
		"single no scope": {`"appliedScopeType":"Shared"`, `"appliedScopeType":"Single"`},
		"mg no tenant":    {`"appliedScopeType":"Shared"`, `"appliedScopeType":"ManagementGroup","appliedScopeProperties":{"managementGroupId":"/providers/Microsoft.Management/managementGroups/mg"}`},
		"shared with sub": {`"appliedScopeType":"Shared"`, `"appliedScopeType":"Shared","appliedScopeProperties":{"subscriptionId":"` + azureBBTestSub + `"}`},
		"scope type":      {`"appliedScopeType":"Shared"`, `"appliedScopeType":"Everywhere"`},
		"sku":             {`"name":"Compute_Savings_Plan"`, `"name":""`},
		"renew":           {`"term":"P3Y"`, `"term":"P3Y","renew":"yes"`},
	} {
		resp := do(http.MethodPut, azureBBTestRoot+"/savingsPlanOrderAliases/bad"+azureBBTestAPI, strings.Replace(valid, edit[0], edit[1], 1))
		t.Run(name, func(t *testing.T) {
			azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, "InvalidRequestContent")
		})
	}
	azureResourcesExpectErrorCode(t, do(http.MethodPut, azureBBTestRoot+"/savingsPlanOrderAliases/bad%20name"+azureBBTestAPI, valid), http.StatusBadRequest, "InvalidResourceName")
	billing := strings.Replace(valid, azureBBTestSub, "/providers/Microsoft.Billing/billingAccounts/1234567/billingSubscriptions/30000000-0000-0000-0000-000000000000", 1)
	alias := providerContractJSONMap(t, do(http.MethodPut, azureBBTestRoot+"/savingsPlanOrderAliases/billing"+azureBBTestAPI, billing))
	order := providerContractJSONMap(t, do(http.MethodGet, "/azure"+alias["properties"].(map[string]any)["savingsPlanOrderId"].(string)+azureBBTestAPI, ""))
	if order["properties"].(map[string]any)["billingAccountId"] != "/providers/Microsoft.Billing/billingAccounts/1234567" {
		t.Fatalf("a billing subscription scope names its billing account: %#v", order)
	}

	result := providerContractJSONMap(t, do(http.MethodPost, azureBBTestRoot+"/validate?api-version=2022-11-01", `{"benefits":[`+valid+`,`+strings.Replace(valid, `"term":"P3Y"`, `"term":"P1M"`, 1)+`]}`))
	benefits := result["benefits"].([]any)
	if benefits[0].(map[string]any)["valid"] != true || benefits[1].(map[string]any)["valid"] != false || benefits[1].(map[string]any)["reason"] == nil {
		t.Fatalf("unexpected validate result: %#v", result)
	}
	azureResourcesExpectErrorCode(t, do(http.MethodPost, azureBBTestRoot+"/validate"+azureBBTestAPI, `{"benefits":[]}`), http.StatusBadRequest, "InvalidRequestContent")
}

func TestAzureBillingBenefitsUpdate(t *testing.T) {
	t.Parallel()
	_, do := newAzureDatabaseTestServer(t, "rg-bb")
	_, plan := azureBBBuy(t, do, "upd", "before", "P1M")
	path := plan + azureBBTestAPI
	rg := azureBBTestSub + "/resourceGroups/testrg"
	patched := providerContractJSONMap(t, do(http.MethodPatch, path, `{"properties":{"displayName":"after","appliedScopeType":"Single","appliedScopeProperties":{"subscriptionId":"`+azureBBTestSub+`"}}}`))
	pp := patched["properties"].(map[string]any)
	if pp["displayName"] != "after" || pp["userFriendlyAppliedScopeType"] != "Single" || pp["commitment"].(map[string]any)["amount"] != 0.001 {
		t.Fatalf("unexpected patched plan: %#v", pp)
	}
	scope := providerContractJSONMap(t, do(http.MethodPatch, path, `{"properties":{"appliedScopeType":"Single","appliedScopeProperties":{"resourceGroupId":"`+rg+`"}}}`))["properties"].(map[string]any)["appliedScopeProperties"].(map[string]any)
	if scope["resourceGroupId"] != rg || scope["subscriptionId"] != nil {
		t.Fatalf("a new scope should replace the old one: %#v", scope)
	}
	// A bare subscription GUID pairs with a resource group in it.
	bare := `{"properties":{"appliedScopeType":"Single","appliedScopeProperties":{"subscriptionId":"30000000-0000-0000-0000-000000000000","resourceGroupId":"` + rg + `"}}}`
	azureResourcesExpectStatus(t, do(http.MethodPatch, path, bare), http.StatusOK)
	if got := providerContractJSONMap(t, do(http.MethodPatch, path, `{"properties":{"appliedScopeType":"Shared"}}`))["properties"].(map[string]any); got["appliedScopeProperties"] != nil {
		t.Fatalf("a Shared scope has no scope properties: %#v", got)
	}
	renewal := `{"purchaseProperties":{"sku":{"name":"Compute_Savings_Plan"},"properties":{"term":"P1Y","commitment":{"amount":15.23,"currencyCode":"USD","grain":"Hourly"}}}}`
	for name, payload := range map[string]string{
		"commitment":      `{"properties":{"commitment":{"amount":1,"currencyCode":"USD","grain":"Hourly"}}}`,
		"term":            `{"properties":{"term":"P1Y"}}`,
		"scope props":     `{"properties":{"appliedScopeProperties":{"subscriptionId":"` + azureBBTestSub + `"}}}`,
		"renew off":       `{"properties":{"renew":false,"renewProperties":` + renewal + `}}`,
		"renew implicit":  `{"properties":{"renewProperties":` + renewal + `}}`,
		"renew term":      `{"properties":{"renew":true,"renewProperties":` + strings.Replace(renewal, "P1Y", "P5Y", 1) + `}}`,
		"rg outside sub":  `{"properties":{"appliedScopeType":"Single","appliedScopeProperties":{"subscriptionId":"/subscriptions/10000000-0000-0000-0000-000000000000","resourceGroupId":"` + rg + `"}}}`,
		"display name":    `{"properties":{"displayName":""}}`,
		"properties type": `{"properties":"Shared"}`,
	} {
		resp := do(http.MethodPatch, path, payload)
		t.Run(name, func(t *testing.T) {
			azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, "InvalidRequestContent")
		})
	}
	renewed := providerContractJSONMap(t, do(http.MethodPatch, path, `{"properties":{"renew":true,"renewProperties":`+renewal+`}}`))["properties"].(map[string]any)
	if renewed["renew"] != true || renewed["renewProperties"] == nil {
		t.Fatalf("renewal settings should be stored: %#v", renewed)
	}
	// A later update that leaves renewal alone keeps it.
	if got := providerContractJSONMap(t, do(http.MethodPatch, path, `{"properties":{"displayName":"kept"}}`))["properties"].(map[string]any); got["renewProperties"] == nil {
		t.Fatalf("renewal settings should survive other updates: %#v", got)
	}
	if got := providerContractJSONMap(t, do(http.MethodPatch, path, `{"properties":{"renew":false}}`))["properties"].(map[string]any); got["renew"] != false || got["renewProperties"] != nil {
		t.Fatalf("turning renew off should drop renewal settings: %#v", got)
	}

	result := providerContractJSONMap(t, do(http.MethodPost, plan+"/validate"+azureBBTestAPI, `{"benefits":[
		{"appliedScopeType":"ManagementGroup","appliedScopeProperties":{"managementGroupId":"/providers/Microsoft.Management/managementGroups/MockMG","tenantId":"30000000-0000-0000-0000-000000000100"}},
		{"appliedScopeType":"ManagementGroup","appliedScopeProperties":{"managementGroupId":"/providers/Microsoft.Management/managementGroups/MockMG"}},
		{"term":"P1Y"},
		{"renew":false,"renewProperties":`+renewal+`}]}`))
	got := result["benefits"].([]any)
	if got[0].(map[string]any)["valid"] != true || got[1].(map[string]any)["valid"] != false || got[2].(map[string]any)["valid"] != false || got[3].(map[string]any)["valid"] != false {
		t.Fatalf("unexpected validate result: %#v", got)
	}
	azureResourcesExpectErrorCode(t, do(http.MethodPost, plan+"/validate"+azureBBTestAPI, `{}`), http.StatusBadRequest, "InvalidRequestContent")
}

func TestAzureBillingBenefitsSavingsPlanList(t *testing.T) {
	t.Parallel()
	_, do := newAzureDatabaseTestServer(t, "rg-bb")
	for _, name := range []string{"bravo", "alpha", "charlie"} {
		azureBBBuy(t, do, name, name, "")
	}
	list := func(query string) map[string]any {
		t.Helper()
		return providerContractJSONMap(t, do(http.MethodGet, azureBBTestRoot+"/savingsPlans"+azureBBTestAPI+query, ""))
	}
	names := func(body map[string]any) string {
		var out []string
		for _, item := range body["value"].([]any) {
			out = append(out, item.(map[string]any)["properties"].(map[string]any)["displayName"].(string))
		}
		return strings.Join(out, ",")
	}
	all := list("&$orderby=properties/displayName%20desc")
	summary := all["additionalProperties"].([]any)[0].(map[string]any)
	if names(all) != "charlie,bravo,alpha" || summary["name"] != "summary" || summary["value"].(map[string]any)["succeededCount"] != float64(3) || all["nextLink"] != nil {
		t.Fatalf("unexpected list: %#v", all)
	}
	filtered := list("&$orderby=properties/displayName&$filter=" + "(properties%2fdisplayName+eq+'alpha')+or+properties/displayName+eq+'charlie'+and+properties/archived+eq+false")
	if names(filtered) != "alpha,charlie" {
		t.Fatalf("unexpected filtered list: %s", names(filtered))
	}
	// and binds tighter than or: alpha or (charlie and bravo) is alpha;
	// (alpha or charlie) and bravo would be nothing.
	if got := names(list("&$filter=properties/displayName+eq+'alpha'+or+properties/displayName+eq+'charlie'+and+properties/displayName+eq+'bravo'")); got != "alpha" {
		t.Fatalf("and must bind tighter than or, got %q", got)
	}
	// Documented fields that plans store under other names.
	for _, query := range []string{"properties/provisioningState+eq+'Succeeded'", "properties/userFriendlyRenewState+eq+'Off'"} {
		if got := list("&$filter=" + query); len(got["value"].([]any)) != 3 {
			t.Fatalf("%s matched %d plans", query, len(got["value"].([]any)))
		}
	}
	expiry := list("&$orderby=properties/expiryDate")["value"].([]any)[0].(map[string]any)["properties"].(map[string]any)["expiryDateTime"]
	if got := list("&$filter=properties/expiryDate+eq+'" + expiry.(string) + "'"); len(got["value"].([]any)) == 0 {
		t.Fatalf("expiryDate filter matched nothing")
	}
	page := list("&$orderby=properties/displayName&take=2")
	if names(page) != "alpha,bravo" || !strings.Contains(page["nextLink"].(string), "skiptoken=2") {
		t.Fatalf("unexpected first page: %#v", page)
	}
	if rest := list("&$orderby=properties/displayName&take=2&$skiptoken=2"); names(rest) != "charlie" || rest["nextLink"] != nil {
		t.Fatalf("unexpected second page: %#v", rest)
	}
	if got := list("&selectedState=Cancelled"); len(got["value"].([]any)) != 0 || got["additionalProperties"].([]any)[0].(map[string]any)["value"].(map[string]any)["succeededCount"] != float64(3) {
		t.Fatalf("selectedState narrows the list, not the summary: %#v", got)
	}
	for name, query := range map[string]string{
		"ne":        "&$filter=properties/displayName+ne+'alpha'",
		"field":     "&$filter=properties/commitment+eq+'x'",
		"dangling":  "&$filter=properties/displayName+eq+'alpha'+and+",
		"group":     "&$filter=(properties/displayName+eq+'a)'+or+properties/displayName+eq+'b')",
		"take":      "&take=0",
		"skiptoken": "&$skiptoken=-1",
		"hugetake":  "&take=9223372036854775808&$skiptoken=1",
		"floattake": "&take=1e2",
		"orderby":   "&$orderby=properties/displayName%20sideways",
		"refresh":   "&refreshSummary=maybe",
	} {
		resp := do(http.MethodGet, azureBBTestRoot+"/savingsPlans"+azureBBTestAPI+query, "")
		t.Run(name, func(t *testing.T) { azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, "BadRequest") })
	}
}

func TestAzureBillingBenefitsReservationOrderAlias(t *testing.T) {
	t.Parallel()
	_, do := newAzureDatabaseTestServer(t, "rg-bb")
	valid := `{"location":"eastus","sku":{"name":"Standard_M64s_v2"},"properties":{"appliedScopeType":"Single","appliedScopeProperties":{"resourceGroupId":"` + azureBBTestSub + `/resourceGroups/testrg"},"billingPlan":"P1M","billingScopeId":"` + azureBBTestSub + `","displayName":"ReservationOrder","quantity":5,"renew":true,"renewAction":"RenewExisting","reservedResourceProperties":{"instanceFlexibility":"On"},"reservedResourceType":"VirtualMachines","term":"P1M"}}`
	path := azureBBTestRoot + "/reservationOrderAliases/reservationOrderAlias123" + azureBBTestAPI
	resp := do(http.MethodPut, path, valid)
	if resp.StatusCode != http.StatusCreated || resp.Header.Get("Azure-AsyncOperation") == "" {
		t.Fatalf("purchase: status %d", resp.StatusCode)
	}
	alias := providerContractJSONMap(t, resp)
	props := alias["properties"].(map[string]any)
	if alias["location"] != "eastus" || props["provisioningState"] != "Created" || !strings.HasPrefix(props["reservationOrderId"].(string), "/providers/Microsoft.Capacity/reservationOrders/") || props["quantity"] != float64(5) {
		t.Fatalf("unexpected alias: %#v", alias)
	}
	if got := providerContractJSONMap(t, do(http.MethodGet, path, "")); got["properties"].(map[string]any)["reservationOrderId"] != props["reservationOrderId"] {
		t.Fatalf("unexpected alias GET: %#v", got)
	}
	azureResourcesExpectErrorCode(t, do(http.MethodPut, path, valid), http.StatusBadRequest, "InvalidRequestContent")
	for name, edit := range map[string][2]string{
		"quantity":    {`"quantity":5`, `"quantity":0`},
		"type":        {`"VirtualMachines"`, `"Spaceships"`},
		"term":        {`"term":"P1M"`, `"term":"P2Y"`},
		"action":      {`"RenewExisting"`, `"Maybe"`},
		"flexibility": {`"instanceFlexibility":"On"`, `"instanceFlexibility":"Sometimes"`},
		"plan":        {`"billingPlan":"P1M"`, `"billingPlan":"Monthly"`},
		"scope":       {`"appliedScopeType":"Single"`, `"appliedScopeType":"ManagementGroup"`},
		"billing":     {`"billingScopeId":"` + azureBBTestSub + `"`, `"billingScopeId":"nope"`},
		"billing id":  {`"billingScopeId":"` + azureBBTestSub + `"`, `"billingScopeId":"/subscriptions/------------------------------------"`},
	} {
		resp := do(http.MethodPut, azureBBTestRoot+"/reservationOrderAliases/bad"+azureBBTestAPI, strings.Replace(valid, edit[0], edit[1], 1))
		t.Run(name, func(t *testing.T) {
			azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, "InvalidRequestContent")
		})
	}
}
