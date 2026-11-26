package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

const opstAddress = `{"ContactName":"Ops","ContactPhoneNumber":"+15555550100","AddressLine1":"1 Main St","City":"Seattle","StateOrRegion":"WA","PostalCode":"98101","CountryCode":"US"}`

func opstOK(t *testing.T, ts *httptest.Server, method, path, body string) map[string]any {
	t.Helper()
	return edgeOK(t, ts, "outposts", method, path, body)
}

func opstSiteAndOutpost(t *testing.T, ts *httptest.Server, name string) (string, string) {
	t.Helper()
	site := opstOK(t, ts, http.MethodPost, "/sites", `{"Name":"`+name+`","OperatingAddress":`+opstAddress+`,"ShippingAddress":`+opstAddress+`,"Tags":{"team":"edge"}}`)["Site"].(map[string]any)
	out := opstOK(t, ts, http.MethodPost, "/outposts", `{"Name":"`+name+`-op","SiteId":"`+site["SiteId"].(string)+`"}`)["Outpost"].(map[string]any)
	return site["SiteId"].(string), out["OutpostId"].(string)
}

// opstFulfil orders one m5 rack for the outpost and reads it to FULFILLED.
func opstFulfil(t *testing.T, ts *httptest.Server, outpost string) string {
	t.Helper()
	ord := opstOK(t, ts, http.MethodPost, "/orders", `{"OutpostIdentifier":"`+outpost+`","PaymentOption":"ALL_UPFRONT","LineItems":[{"CatalogItemId":"OR-RACKM5A","Quantity":1}]}`)["Order"].(map[string]any)
	id := ord["OrderId"].(string)
	var statuses []string
	for i := 0; i < 4; i++ {
		statuses = append(statuses, opstOK(t, ts, http.MethodGet, "/orders/"+id, "")["Order"].(map[string]any)["Status"].(string))
	}
	if strings.Join(statuses, ",") != "PENDING,PROCESSING,INSTALLING,FULFILLED" {
		t.Fatalf("order statuses: %v", statuses)
	}
	return id
}

func TestOutpostsStage1SitesOutpostsAndOrders(t *testing.T) {
	_, ts := newBudTestServer(t)
	siteID, opID := opstSiteAndOutpost(t, ts, "seattle")
	if code := edgeErr(t, ts, "outposts", http.MethodPost, "/sites", `{"Name":"seattle"}`, http.StatusConflict); code != "ConflictException" {
		t.Fatalf("duplicate site: %s", code)
	}
	site := opstOK(t, ts, http.MethodGet, "/sites/"+siteID, "")["Site"].(map[string]any)
	if site["OperatingAddressCity"] != "Seattle" {
		t.Fatalf("site: %v", site)
	}
	addr := opstOK(t, ts, http.MethodGet, "/sites/"+siteID+"/address?AddressType=SHIPPING_ADDRESS", "")
	if addr["Address"].(map[string]any)["PostalCode"] != "98101" {
		t.Fatalf("address: %v", addr)
	}
	if n := len(opstOK(t, ts, http.MethodGet, "/sites?OperatingAddressCityFilter=Portland", "")["Sites"].([]any)); n != 0 {
		t.Fatalf("city filter: %d", n)
	}
	// A site with outposts can't be deleted.
	if code := edgeErr(t, ts, "outposts", http.MethodDelete, "/sites/"+siteID, "", http.StatusConflict); code != "ConflictException" {
		t.Fatalf("delete busy site: %s", code)
	}

	// Ordering hardware of the wrong form factor is refused.
	if code := edgeErr(t, ts, "outposts", http.MethodPost, "/orders", `{"OutpostIdentifier":"`+opID+`","PaymentOption":"ALL_UPFRONT","LineItems":[{"CatalogItemId":"OR-SRVC6GD","Quantity":1}]}`, http.StatusBadRequest); code != "ValidationException" {
		t.Fatalf("server item on rack outpost: %s", code)
	}
	orderID := opstFulfil(t, ts, opID)
	if code := edgeErr(t, ts, "outposts", http.MethodPost, "/orders/"+orderID+"/cancel", "", http.StatusConflict); code != "ConflictException" {
		t.Fatalf("cancel fulfilled order: %s", code)
	}
	assets := opstOK(t, ts, http.MethodGet, "/outposts/"+opID+"/assets", "")["Assets"].([]any)
	if len(assets) != 1 {
		t.Fatalf("assets: %v", assets)
	}
	asset := assets[0].(map[string]any)
	if asset["ComputeAttributes"].(map[string]any)["MaxVcpus"].(float64) != 96 {
		t.Fatalf("asset: %v", asset)
	}
	types := opstOK(t, ts, http.MethodGet, "/outposts/"+opID+"/instanceTypes", "")["InstanceTypes"].([]any)
	if len(types) != 4 || !strings.HasPrefix(types[0].(map[string]any)["InstanceType"].(string), "m5.") {
		t.Fatalf("instance types: %v", types)
	}

	// Billing: the order's subscription is ACTIVE; a renewal starts at the contract end.
	bill := opstOK(t, ts, http.MethodGet, "/outpost/"+opID+"/billing-information", "")
	subs := bill["Subscriptions"].([]any)
	if len(subs) != 1 || subs[0].(map[string]any)["SubscriptionStatus"] != "ACTIVE" || bill["ContractEndDate"] == nil {
		t.Fatalf("billing: %v", bill)
	}
	if p := opstOK(t, ts, http.MethodGet, "/outpost/"+opID+"/renewal-pricing", ""); p["PricingResult"] != "PRICED" || len(p["PricingOptions"].([]any)) != 9 {
		t.Fatalf("renewal pricing: %v", p)
	}
	renewal := `{"OutpostIdentifier":"` + opID + `","PaymentOption":"NO_UPFRONT","PaymentTerm":"ONE_YEAR","ClientToken":"r1"}`
	r1 := opstOK(t, ts, http.MethodPost, "/renewals", renewal)
	if r1["UpfrontPrice"].(float64) != 0 || r1["MonthlyRecurringPrice"].(float64) != 7000 {
		t.Fatalf("renewal: %v", r1)
	}
	opstOK(t, ts, http.MethodPost, "/renewals", renewal) // idempotent
	if subs := opstOK(t, ts, http.MethodGet, "/outpost/"+opID+"/billing-information", "")["Subscriptions"].([]any); len(subs) != 2 || subs[1].(map[string]any)["SubscriptionStatus"] != "PENDING" {
		t.Fatalf("renewal subscription: %v", subs)
	}

	// Hardware shipping freezes the site's address.
	ord := opstOK(t, ts, http.MethodPost, "/orders", `{"OutpostIdentifier":"`+opID+`","PaymentOption":"ALL_UPFRONT","LineItems":[{"CatalogItemId":"OR-RACKC5A","Quantity":1}]}`)["Order"].(map[string]any)
	second := ord["OrderId"].(string)
	opstOK(t, ts, http.MethodGet, "/orders/"+second, "")
	opstOK(t, ts, http.MethodGet, "/orders/"+second, "") // PROCESSING
	if code := edgeErr(t, ts, "outposts", http.MethodPut, "/sites/"+siteID+"/address", `{"AddressType":"OPERATING_ADDRESS","Address":`+opstAddress+`}`, http.StatusConflict); code != "ConflictException" {
		t.Fatalf("address change while shipping: %s", code)
	}
	if d := opstOK(t, ts, http.MethodPost, "/outposts/"+opID+"/decommission", `{"ValidateOnly":true}`); d["Status"] != "BLOCKED" {
		t.Fatalf("decommission with an order in flight: %v", d)
	}
	opstOK(t, ts, http.MethodGet, "/orders/"+second, "")
	opstOK(t, ts, http.MethodGet, "/orders/"+second, "") // FULFILLED
	if d := opstOK(t, ts, http.MethodPost, "/outposts/"+opID+"/decommission", `{"ValidateOnly":true}`); d["Status"] != "SKIPPED" {
		t.Fatalf("validate-only decommission: %v", d)
	}
	if d := opstOK(t, ts, http.MethodPost, "/outposts/"+opID+"/decommission", `{}`); d["Status"] != "REQUESTED" {
		t.Fatalf("decommission: %v", d)
	}
	if o := opstOK(t, ts, http.MethodGet, "/outposts/"+opID, "")["Outpost"].(map[string]any); o["LifeCycleStatus"] != "RETIRING" {
		t.Fatalf("retiring outpost: %v", o)
	}
	opstOK(t, ts, http.MethodDelete, "/outposts/"+opID, "")
	opstOK(t, ts, http.MethodDelete, "/sites/"+siteID, "")
	if code := edgeErr(t, ts, "outposts", http.MethodGet, "/sites/"+siteID, "", http.StatusNotFound); code != "NotFoundException" {
		t.Fatalf("deleted site: %s", code)
	}
}

func TestOutpostsStage1QuotesAndCapacityTasks(t *testing.T) {
	_, ts := newBudTestServer(t)
	_, opID := opstSiteAndOutpost(t, ts, "quotes")
	// A quote without an outpost fails its ordering requirement.
	loose := opstOK(t, ts, http.MethodPost, "/quotes", `{"CountryCode":"US","RequestedCapacities":[{"QuoteCapacityType":"EC2","Unit":"c5","Quantity":150}]}`)["Quote"].(map[string]any)
	if reqs := loose["OrderingRequirements"].([]any); reqs[0].(map[string]any)["Status"] != "FAIL" {
		t.Fatalf("requirements: %v", reqs)
	}
	opt := loose["QuoteOptions"].([]any)[0].(map[string]any)
	if specs := opt["Specifications"].([]any); len(specs) != 2 {
		t.Fatalf("150 c5 vCPUs need two racks: %v", specs)
	}
	updated := opstOK(t, ts, http.MethodPatch, "/quotes/"+loose["QuoteId"].(string), `{"OutpostIdentifier":"`+opID+`","RequestedPaymentTerms":["ONE_YEAR","THREE_YEARS"]}`)["Quote"].(map[string]any)
	for _, r := range updated["OrderingRequirements"].([]any) {
		if r.(map[string]any)["Status"] != "PASS" {
			t.Fatalf("requirements after adding the outpost: %v", updated["OrderingRequirements"])
		}
	}
	optID := updated["QuoteOptions"].([]any)[0].(map[string]any)["QuoteOptionIdentifier"].(string)
	if n := len(updated["QuoteOptions"].([]any)[0].(map[string]any)["PricingOptions"].([]any)); n != 2 {
		t.Fatalf("pricing options: %d", n)
	}
	if code := edgeErr(t, ts, "outposts", http.MethodPost, "/quotes", `{"CountryCode":"US","RequestedCapacities":[{"QuoteCapacityType":"EC2","Unit":"z9","Quantity":1}]}`, http.StatusBadRequest); code != "ValidationException" {
		t.Fatalf("unknown family: %s", code)
	}
	qid := updated["QuoteId"].(string)
	ord := opstOK(t, ts, http.MethodPost, "/orders", fmt.Sprintf(`{"OutpostIdentifier":%q,"QuoteIdentifier":%q,"QuoteOptionIdentifier":%q,"PaymentOption":"ALL_UPFRONT","PaymentTerm":"ONE_YEAR"}`, opID, qid, optID))["Order"].(map[string]any)
	if lines := ord["LineItems"].([]any); len(lines) != 1 || lines[0].(map[string]any)["Quantity"].(float64) != 2 {
		t.Fatalf("order from quote: %v", ord)
	}
	if q := opstOK(t, ts, http.MethodGet, "/quotes/"+qid, "")["Quote"].(map[string]any); q["QuoteStatus"] != "ORDER_SUBMITTED" || q["SubmittedOrderId"] != ord["OrderId"] {
		t.Fatalf("quote after order: %v", q)
	}
	if code := edgeErr(t, ts, "outposts", http.MethodDelete, "/quotes/"+qid, "", http.StatusBadRequest); code != "ValidationException" {
		t.Fatalf("delete submitted quote: %s", code)
	}
	for i := 0; i < 4; i++ {
		opstOK(t, ts, http.MethodGet, "/orders/"+ord["OrderId"].(string), "")
	}

	// Capacity: two c5 racks = 192 vCPUs.
	base := "/outposts/" + opID + "/capacity"
	tooBig := opstOK(t, ts, http.MethodPost, base, `{"InstancePools":[{"InstanceType":"c5.4xlarge","Count":13}]}`)
	if tooBig["CapacityTaskStatus"] != "FAILED" || tooBig["Failed"].(map[string]any)["Type"] != "UNSUPPORTED_CAPACITY_CONFIGURATION" {
		t.Fatalf("oversized task: %v", tooBig)
	}
	task := opstOK(t, ts, http.MethodPost, base, `{"InstancePools":[{"InstanceType":"c5.4xlarge","Count":10},{"InstanceType":"c5.large","Count":16}]}`)
	taskID := task["CapacityTaskId"].(string)
	if code := edgeErr(t, ts, "outposts", http.MethodPost, base, `{"InstancePools":[{"InstanceType":"c5.large","Count":1}]}`, http.StatusConflict); code != "ConflictException" {
		t.Fatalf("second task in flight: %s", code)
	}
	var statuses []string
	for i := 0; i < 2; i++ {
		statuses = append(statuses, opstOK(t, ts, http.MethodGet, base+"/"+taskID, "")["CapacityTaskStatus"].(string))
	}
	if strings.Join(statuses, ",") != "IN_PROGRESS,COMPLETED" {
		t.Fatalf("task statuses: %v", statuses)
	}
	total := 0.0
	for _, a := range opstOK(t, ts, http.MethodGet, "/outposts/"+opID+"/assets", "")["Assets"].([]any) {
		for _, p := range a.(map[string]any)["ComputeAttributes"].(map[string]any)["InstanceTypeCapacities"].([]any) {
			total += p.(map[string]any)["Count"].(float64)
		}
	}
	if total != 26 {
		t.Fatalf("applied pools: %v instances", total)
	}
	cancel := opstOK(t, ts, http.MethodPost, base, `{"InstancePools":[{"InstanceType":"c5.large","Count":1}]}`)["CapacityTaskId"].(string)
	opstOK(t, ts, http.MethodPost, base+"/"+cancel, "")
	if s := opstOK(t, ts, http.MethodGet, base+"/"+cancel, "")["CapacityTaskStatus"]; s != "CANCELLED" {
		t.Fatalf("cancelled task: %v", s)
	}
	if n := len(opstOK(t, ts, http.MethodGet, "/capacity/tasks?CapacityTaskStatusFilter=COMPLETED", "")["CapacityTasks"].([]any)); n != 1 {
		t.Fatalf("completed tasks: %d", n)
	}

	// Connections to an installed asset.
	assetID := opstOK(t, ts, http.MethodGet, "/outposts/"+opID+"/assets", "")["Assets"].([]any)[0].(map[string]any)["AssetId"].(string)
	conn := opstOK(t, ts, http.MethodPost, "/connections", `{"AssetId":"`+assetID+`","ClientPublicKey":"`+strings.Repeat("A", 43)+`=","NetworkInterfaceDeviceIndex":0}`)
	details := opstOK(t, ts, http.MethodGet, "/connections/"+conn["ConnectionId"].(string), "")["ConnectionDetails"].(map[string]any)
	if len(details["ServerPublicKey"].(string)) != 44 {
		t.Fatalf("connection: %v", details)
	}

	// Paging tokens use the Outposts format and are checked.
	page := opstOK(t, ts, http.MethodGet, "/catalog/items?MaxResults=2", "")
	if tok := page["NextToken"].(string); tok != "2##page" {
		t.Fatalf("token: %v", tok)
	}
	if code := edgeErr(t, ts, "outposts", http.MethodGet, "/catalog/items?NextToken=99%23%23page", "", http.StatusBadRequest); code != "ValidationException" {
		t.Fatalf("bad token: %s", code)
	}

	// Tags on outposts and sites.
	arn := url.PathEscape("arn:aws:outposts:us-east-1:123456789012:outpost/" + opID)
	opstOK(t, ts, http.MethodPost, "/tags/"+arn, `{"Tags":{"env":"dev"}}`)
	opstOK(t, ts, http.MethodDelete, "/tags/"+arn+"?tagKeys=env", "")
	if tags := opstOK(t, ts, http.MethodGet, "/tags/"+arn, "")["Tags"].(map[string]any); len(tags) != 0 {
		t.Fatalf("tags: %v", tags)
	}
}

// TestOutpostsStage1Bounds checks caps on update paths as well as creates.
func TestOutpostsStage1Bounds(t *testing.T) {
	srv, ts := newBudTestServer(t)
	_, opID := opstSiteAndOutpost(t, ts, "bounds")
	tags := make([]string, 0, opstMaxTags+1)
	for i := 0; i <= opstMaxTags; i++ {
		tags = append(tags, fmt.Sprintf(`"k%d":"v"`, i))
	}
	arn := url.PathEscape("arn:aws:outposts:us-east-1:123456789012:outpost/" + opID)
	if code := edgeErr(t, ts, "outposts", http.MethodPost, "/tags/"+arn, `{"Tags":{`+strings.Join(tags, ",")+`}}`, http.StatusBadRequest); code != "ValidationException" {
		t.Fatalf("too many tags: %s", code)
	}
	if code := edgeErr(t, ts, "outposts", http.MethodPost, "/tags/"+arn, `{"Tags":{"aws:owner":"x"}}`, http.StatusBadRequest); code != "ValidationException" {
		t.Fatalf("reserved tag key: %s", code)
	}
	// A rack outpost can't be quoted server-only families.
	if code := edgeErr(t, ts, "outposts", http.MethodPost, "/quotes", `{"OutpostIdentifier":"`+opID+`","CountryCode":"US","RequestedCapacities":[{"QuoteCapacityType":"EC2","Unit":"c6gd","Quantity":8}]}`, http.StatusBadRequest); code != "ValidationException" {
		t.Fatalf("form factor mismatch: %s", code)
	}
	caps := make([]string, 0, opstMaxListItems+1)
	for i := 0; i <= opstMaxListItems; i++ {
		caps = append(caps, `{"QuoteCapacityType":"EBS","Unit":"TB","Quantity":1}`)
	}
	q := opstOK(t, ts, http.MethodPost, "/quotes", `{"CountryCode":"US","RequestedCapacities":[{"QuoteCapacityType":"EBS","Unit":"TB","Quantity":1}]}`)["Quote"].(map[string]any)
	if code := edgeErr(t, ts, "outposts", http.MethodPatch, "/quotes/"+q["QuoteId"].(string), `{"RequestedCapacities":[`+strings.Join(caps, ",")+`]}`, http.StatusBadRequest); code != "ValidationException" {
		t.Fatalf("too many capacities on update: %s", code)
	}
	if code := edgeErr(t, ts, "outposts", http.MethodPost, "/orders", `{"OutpostIdentifier":"`+opID+`","PaymentOption":"ALL_UPFRONT","LineItems":[{"CatalogItemId":"OR-RACKM5A","Quantity":11}]}`, http.StatusBadRequest); code != "ValidationException" {
		t.Fatalf("oversized line item: %s", code)
	}
	if code := edgeErr(t, ts, "outposts", http.MethodPost, "/sites", `{"Name":"`+strings.Repeat("x", opstMaxBody)+`"}`, http.StatusBadRequest); code != "ValidationException" {
		t.Fatalf("oversized body: %s", code)
	}
	srv.outposts.mu.Lock()
	for i := len(srv.outposts.sites); i < opstMaxSites; i++ {
		id := fmt.Sprintf("os-%017x", i)
		srv.outposts.sites[id] = &opstSite{id: id, name: id}
	}
	srv.outposts.mu.Unlock()
	if code := edgeErr(t, ts, "outposts", http.MethodPost, "/sites", `{"Name":"one-too-many"}`, http.StatusPaymentRequired); code != "ServiceQuotaExceededException" {
		t.Fatalf("site quota: %s", code)
	}
}
