package server

import (
	"net/http"
	"strings"
	"testing"
)

const azureNewRelicTestAPIVersion = "?api-version=2026-06-01"

const azureNewRelicTestMonitorBody = `{"location":"West US 2","properties":{"accountCreationSource":"LIFTR","orgCreationSource":"LIFTR",
	"planData":{"billingCycle":"Monthly","effectiveDate":"2022-12-05T14:11:37.786Z","planDetails":"newrelic-pay-as-you-go","usageType":"PAYG"},
	"userInfo":{"emailAddress":"alice@contoso.com","firstName":"Alice","lastName":"Smith","country":"US"}}}`

func newAzureNewRelicTestMonitor(t *testing.T, rg string) (string, string, func(method, path, body string) *http.Response) {
	t.Helper()
	sub, do := newAzureDatabaseTestServer(t, rg)
	monitor := sub + "/resourceGroups/" + rg + "/providers/NewRelic.Observability/monitors/mon1"
	azureResourcesExpectStatus(t, do(http.MethodPut, monitor+azureNewRelicTestAPIVersion, azureNewRelicTestMonitorBody), http.StatusCreated)
	return sub, monitor, do
}

func TestAzureNewRelicMonitors(t *testing.T) {
	t.Parallel()
	sub, monitor, do := newAzureNewRelicTestMonitor(t, "rg-nr")
	got := providerContractJSONMap(t, do(http.MethodGet, monitor+azureNewRelicTestAPIVersion, ""))
	props := got["properties"].(map[string]any)
	account := props["newRelicAccountProperties"].(map[string]any)
	accountInfo := account["accountInfo"].(map[string]any)
	org := account["organizationInfo"].(map[string]any)["organizationId"]
	if props["provisioningState"] != "Succeeded" || props["monitoringStatus"] != "Enabled" || props["marketplaceSubscriptionStatus"] != "Active" ||
		len(azureARMString(accountInfo["accountId"])) != 7 || accountInfo["region"] != "West US 2" || azureARMString(org) == "" || props["marketplaceSubscriptionId"] == nil {
		t.Fatalf("unexpected monitor: %#v", props)
	}

	// The ingestion key is never returned, but the VM host payload has it.
	key := azureARMString(providerContractJSONMap(t, do(http.MethodPost, monitor+"/vmHostPayloads"+azureNewRelicTestAPIVersion, ""))["ingestionKey"])
	if key == "" || accountInfo["ingestionKey"] != nil {
		t.Fatalf("key = %q, accountInfo = %#v", key, accountInfo)
	}

	// The organization can't change; omitting it keeps it.
	azureSentinelExpectRejected(t, do(http.MethodPatch, monitor+azureNewRelicTestAPIVersion, `{"properties":{"newRelicAccountProperties":{"organizationInfo":{"organizationId":"other"}}}}`), "can't be changed after creation")
	azureSentinelExpectRejected(t, do(http.MethodPut, monitor+azureNewRelicTestAPIVersion, `{"location":"West US 2","properties":{"newRelicAccountProperties":{"organizationInfo":{"organizationId":"other"}}}}`), "can't be changed after creation")
	patched := providerContractJSONMap(t, do(http.MethodPatch, monitor+azureNewRelicTestAPIVersion, `{"tags":{"env":"prod"},"properties":{"planData":{"usageType":"COMMITTED"},
		"newRelicAccountProperties":{"accountInfo":{"ingestionKey":"ignored"}}}}`))
	pp := patched["properties"].(map[string]any)
	pa := pp["newRelicAccountProperties"].(map[string]any)
	if pp["planData"].(map[string]any)["usageType"] != "COMMITTED" || pa["organizationInfo"].(map[string]any)["organizationId"] != org ||
		pa["accountInfo"].(map[string]any)["accountId"] != accountInfo["accountId"] || patched["tags"].(map[string]any)["env"] != "prod" {
		t.Fatalf("unexpected patched monitor: %#v", patched)
	}

	// A PUT that omits the account and organization keeps them.
	kept := providerContractJSONMap(t, do(http.MethodPut, monitor+azureNewRelicTestAPIVersion, `{"location":"West US 2","properties":{"planData":{"usageType":"COMMITTED"}}}`))
	if ka := kept["properties"].(map[string]any)["newRelicAccountProperties"].(map[string]any); ka["accountInfo"].(map[string]any)["accountId"] != accountInfo["accountId"] || ka["organizationInfo"].(map[string]any)["organizationId"] != org {
		t.Fatalf("PUT without account = %#v", kept)
	}

	// A second monitor linked to the same account and organization shares
	// the key; refreshing it rotates the key for both.
	other := sub + "/resourceGroups/rg-nr/providers/NewRelic.Observability/monitors/mon2"
	azureResourcesExpectStatus(t, do(http.MethodPut, other+azureNewRelicTestAPIVersion, `{"location":"westus2","properties":{"accountCreationSource":"NEWRELIC","orgCreationSource":"NEWRELIC",
		"newRelicAccountProperties":{"accountInfo":{"accountId":"`+azureARMString(accountInfo["accountId"])+`"},"organizationInfo":{"organizationId":"`+azureARMString(org)+`"}},
		"saaSData":{"saaSResourceId":"`+sub[len("/azure"):]+`/resourceGroups/rg-nr/providers/Microsoft.SaaS/resources/abcd"}}}`), http.StatusCreated)
	otherKey := func() string {
		return azureARMString(providerContractJSONMap(t, do(http.MethodPost, other+"/vmHostPayloads"+azureNewRelicTestAPIVersion, ""))["ingestionKey"])
	}
	if otherKey() != key {
		t.Fatalf("linked monitor key = %q, want %q", otherKey(), key)
	}
	azureResourcesExpectStatus(t, do(http.MethodPost, monitor+"/refreshIngestionKey"+azureNewRelicTestAPIVersion, ""), http.StatusNoContent)
	rotated := azureARMString(providerContractJSONMap(t, do(http.MethodPost, monitor+"/vmHostPayloads"+azureNewRelicTestAPIVersion, ""))["ingestionKey"])
	if rotated == key || otherKey() != rotated {
		t.Fatalf("rotated = %q, other = %q, old = %q", rotated, otherKey(), key)
	}

	// A key sent for one monitor of the account doesn't replace the key
	// another monitor already has.
	first := sub + "/resourceGroups/rg-nr/providers/NewRelic.Observability/monitors/a-first"
	azureResourcesExpectStatus(t, do(http.MethodPut, first+azureNewRelicTestAPIVersion, `{"location":"westus2","properties":{"accountCreationSource":"NEWRELIC",
		"newRelicAccountProperties":{"accountInfo":{"accountId":"`+azureARMString(accountInfo["accountId"])+`","ingestionKey":"custom-key"},"organizationInfo":{"organizationId":"`+azureARMString(org)+`"}}}}`), http.StatusCreated)
	azureResourcesExpectStatus(t, do(http.MethodPut, monitor+azureNewRelicTestAPIVersion, azureNewRelicTestMonitorBody), http.StatusOK)
	if kept := azureARMString(providerContractJSONMap(t, do(http.MethodPost, monitor+"/vmHostPayloads"+azureNewRelicTestAPIVersion, ""))["ingestionKey"]); kept != rotated {
		t.Fatalf("key after PUT = %q, want %q", kept, rotated)
	}
	azureResourcesExpectStatus(t, do(http.MethodDelete, first+azureNewRelicTestAPIVersion, ""), http.StatusOK)

	for _, tc := range []struct{ method, body, want string }{
		{http.MethodPut, `{"location":"westus","properties":{"accountCreationSource":"PORTAL"}}`, "accountCreationSource"},
		{http.MethodPut, `{"location":"westus","properties":{"accountCreationSource":"NEWRELIC"}}`, "accountId is required"},
		{http.MethodPut, `{"location":"westus","properties":{"orgCreationSource":"NEWRELIC"}}`, "organizationId is required"},
		{http.MethodPut, `{"location":"westus","properties":{"planData":{"usageType":"FREE"}}}`, "usageType"},
		{http.MethodPut, `{"location":"westus","properties":{"planData":{"effectiveDate":"soon"}}}`, "effectiveDate"},
		{http.MethodPut, `{"location":"westus","properties":{"planData":{"planDetails":1}}}`, "planDetails must be a string"},
		{http.MethodPut, `{"location":"westus","properties":{"userInfo":{"emailAddress":"nobody"}}}`, "emailAddress must be an email address"},
		{http.MethodPut, `{"location":"westus","properties":{"userInfo":{"firstName":"` + strings.Repeat("a", 51) + `"}}}`, "firstName can be at most 50"},
		{http.MethodPut, `{"location":"westus","properties":{"userInfo":{"phoneNumber":"` + strings.Repeat("1", 41) + `"}}}`, "phoneNumber can be at most 40"},
		{http.MethodPut, `{"location":"westus","properties":{"saaSData":{"saaSResourceId":"abcd"}}}`, "saaSResourceId"},
		{http.MethodPut, `{"location":"westus","properties":{"newRelicAccountProperties":{"singleSignOnProperties":{"singleSignOnState":"On"}}}}`, "singleSignOnState"},
		{http.MethodPut, `{"location":"westus","properties":{"newRelicAccountProperties":{"accountInfo":{"region":5}}}}`, "region must be a string"},
		{http.MethodPut, `{"location":"westus","properties":{"subscriptionState":5}}`, "subscriptionState must be a string"},
		{http.MethodPut, `{"location":"westus","identity":{"type":"Managed"},"properties":{}}`, "identity.type"},
		{http.MethodPut, `{"location":"westus","identity":{"type":"UserAssigned"},"properties":{}}`, "userAssignedIdentities is required"},
		{http.MethodPut, `{"location":"westus","identity":{"type":"UserAssigned","userAssignedIdentities":{"key1":{}}},"properties":{}}`, "keys must be user-assigned identity IDs"},
		{http.MethodPatch, `{"properties":{"monitoringStatus":"Disabled"}}`, "monitoringStatus can't be updated"},
		{http.MethodPatch, `{"location":"eastus"}`, "location can't be updated"},
		{http.MethodPatch, `{"properties":{"userInfo":{"emailAddress":"x@y"}}}`, "emailAddress"},
	} {
		target := sub + "/resourceGroups/rg-nr/providers/NewRelic.Observability/monitors/bad"
		if tc.method == http.MethodPatch {
			target = monitor
		}
		azureSentinelExpectRejected(t, do(tc.method, target+azureNewRelicTestAPIVersion, tc.body), tc.want)
	}
	// None needs no identities; Learn's sample sends placeholder keys.
	mon3 := sub + "/resourceGroups/rg-nr/providers/NewRelic.Observability/monitors/mon3"
	mon3Body := `{"location":"westus","identity":{"type":"None","userAssignedIdentities":{"key8903":{}}},"properties":{}}`
	azureResourcesExpectStatus(t, do(http.MethodPut, mon3+azureNewRelicTestAPIVersion, mon3Body), http.StatusCreated)

	// mon1 and mon2 share the organization; mon3 has its own.
	linked := providerContractJSONMap(t, do(http.MethodPost, monitor+"/listLinkedResources"+azureNewRelicTestAPIVersion, ""))["value"].([]any)
	if len(linked) != 2 {
		t.Fatalf("linked = %#v", linked)
	}

	// Deleting a monitor deletes its key, tag rules and monitored
	// subscriptions: recreated, mon3 has none of them.
	mon3Key := func() string {
		return azureARMString(providerContractJSONMap(t, do(http.MethodPost, mon3+"/vmHostPayloads"+azureNewRelicTestAPIVersion, ""))["ingestionKey"])
	}
	original := mon3Key()
	if original == rotated {
		t.Fatalf("mon3, of another account, took mon1's key %q", rotated)
	}
	azureResourcesExpectStatus(t, do(http.MethodPost, mon3+"/refreshIngestionKey"+azureNewRelicTestAPIVersion, ""), http.StatusNoContent)
	azureResourcesExpectStatus(t, do(http.MethodPut, mon3+"/tagRules/default"+azureNewRelicTestAPIVersion, `{"properties":{}}`), http.StatusCreated)
	azureResourcesExpectStatus(t, do(http.MethodPut, mon3+"/monitoredSubscriptions/default"+azureNewRelicTestAPIVersion,
		`{"properties":{"monitoredSubscriptionList":[{"subscriptionId":"00000000-0000-0000-0000-000000000001"}]}}`), http.StatusCreated)
	azureSentinelExpectRejected(t, do(http.MethodPatch, mon3+"/monitoredSubscriptions/default"+azureNewRelicTestAPIVersion,
		`{"properties":{"patchOperation":"DeleteComplete"}}`), "monitoredSubscriptionList is required")
	azureResourcesExpectStatus(t, do(http.MethodDelete, mon3+azureNewRelicTestAPIVersion, ""), http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodPut, mon3+azureNewRelicTestAPIVersion, mon3Body), http.StatusCreated)
	if key := mon3Key(); key != original {
		t.Fatalf("recreated key = %q, want the new monitor's %q", key, original)
	}
	if rules := providerContractJSONMap(t, do(http.MethodGet, mon3+"/tagRules"+azureNewRelicTestAPIVersion, ""))["value"].([]any); len(rules) != 0 {
		t.Fatalf("recreated tag rules = %#v", rules)
	}
	azureResourcesExpectStatus(t, do(http.MethodGet, mon3+"/monitoredSubscriptions/default"+azureNewRelicTestAPIVersion, ""), http.StatusNotFound)
}

func TestAzureNewRelicBilling(t *testing.T) {
	t.Parallel()
	sub, monitor, do := newAzureNewRelicTestMonitor(t, "rg-nr")
	saas := sub[len("/azure"):] + "/resourceGroups/rg-nr/providers/Microsoft.SaaS/resources/abcd"
	post := func(action, body string) *http.Response {
		return do(http.MethodPost, monitor+"/"+action+azureNewRelicTestAPIVersion, body)
	}
	if latest := providerContractJSONMap(t, post("latestLinkedSaaS", "")); latest["isHiddenSaaS"] != false || latest["saaSResourceId"] != nil {
		t.Fatalf("latest = %#v", latest)
	}
	// Without a SaaS resource a monitor is connected only to itself.
	azureResourcesExpectStatus(t, do(http.MethodPut, sub+"/resourceGroups/rg-nr/providers/NewRelic.Observability/monitors/unbilled"+azureNewRelicTestAPIVersion, `{"location":"westus","properties":{}}`), http.StatusCreated)
	if connected := providerContractJSONMap(t, post("listConnectedPartnerResources", ""))["value"].([]any); len(connected) != 1 {
		t.Fatalf("connected without SaaS = %#v", connected)
	}
	azureSentinelExpectRejected(t, post("linkSaaS", `{"saaSResourceId":"abcd"}`), "saaSResourceId")
	// Only listConnectedPartnerResources takes a bare string, and never null.
	azureSentinelExpectRejected(t, post("linkSaaS", `"abcd"`), "could not be deserialized")
	azureSentinelExpectRejected(t, post("listConnectedPartnerResources", `null`), "could not be deserialized")
	linked := providerContractJSONMap(t, post("linkSaaS", `{"saaSResourceId":"`+saas+`"}`))
	if linked["properties"].(map[string]any)["saaSData"].(map[string]any)["saaSResourceId"] != saas {
		t.Fatalf("linked = %#v", linked)
	}
	if latest := providerContractJSONMap(t, post("latestLinkedSaaS", "")); latest["saaSResourceId"] != saas {
		t.Fatalf("latest = %#v", latest)
	}
	billing := providerContractJSONMap(t, post("getBillingInfo", ""))
	if info := billing["marketplaceSaasInfo"].(map[string]any); info["marketplaceResourceId"] != saas || info["billedAzureSubscriptionId"] != azureResourcesTestSubscription || info["marketplaceSubscriptionId"] == nil {
		t.Fatalf("billing = %#v", billing)
	}
	org := billing["partnerBillingEntity"].(map[string]any)["organizationId"]

	// Bare email body, as Learn's reference defines it.
	connected := providerContractJSONMap(t, post("listConnectedPartnerResources", `"alice@contoso.com"`))["value"].([]any)
	if len(connected) != 1 || connected[0].(map[string]any)["properties"].(map[string]any)["azureResourceId"] != monitor[len("/azure"):] {
		t.Fatalf("connected = %#v", connected)
	}
	azureSentinelExpectRejected(t, post("listConnectedPartnerResources", `"nobody"`), "body must be an email address")

	switched := providerContractJSONMap(t, post("switchBilling", `{"userEmail":"alice@contoso.com","organizationId":"`+azureARMString(org)+`","planData":{"usageType":"COMMITTED","planDetails":"annual"}}`))
	if plan := switched["properties"].(map[string]any)["planData"].(map[string]any); plan["usageType"] != "COMMITTED" || plan["planDetails"] != "annual" {
		t.Fatalf("switched = %#v", switched)
	}
	azureResourcesExpectStatus(t, post("switchBilling", `{"userEmail":"alice@contoso.com"}`), http.StatusNoContent)
	for _, tc := range []struct{ body, want string }{
		{`{}`, "userEmail is required"},
		{`{"userEmail":"alice@contoso.com","organizationId":"other"}`, "organizationId must be"},
		{`{"userEmail":"alice@contoso.com","azureResourceId":"/subscriptions/x"}`, "azureResourceId must be"},
		{`{"userEmail":"alice@contoso.com","planData":{"usageType":"FREE"}}`, "usageType"},
		{`{"userEmail":"alice@contoso.com","planData":"x"}`, "planData must be an object"},
	} {
		azureSentinelExpectRejected(t, post("switchBilling", tc.body), tc.want)
	}
	resubscribed := providerContractJSONMap(t, post("resubscribe", `{"planId":"newrelic-annual","termId":"P1Y","organizationId":"`+azureARMString(org)+`"}`))
	if resubscribed["properties"].(map[string]any)["planData"].(map[string]any)["planDetails"] != "newrelic-annual" {
		t.Fatalf("resubscribed = %#v", resubscribed)
	}
	azureSentinelExpectRejected(t, post("resubscribe", `{"planId":5}`), "planId must be a string")
	azureSentinelExpectRejected(t, post("resubscribe", `{"organizationId":"other"}`), "organizationId must be")

	// Plans, accounts and organizations come from the monitors.
	plans := providerContractJSONMap(t, do(http.MethodGet, sub+"/providers/NewRelic.Observability/plans"+azureNewRelicTestAPIVersion+"&organizationId="+azureARMString(org), ""))["value"].([]any)
	if len(plans) != 1 || plans[0].(map[string]any)["properties"].(map[string]any)["orgCreationSource"] != "LIFTR" {
		t.Fatalf("plans = %#v", plans)
	}
	if none := providerContractJSONMap(t, do(http.MethodGet, sub+"/providers/NewRelic.Observability/plans"+azureNewRelicTestAPIVersion+"&accountId=0", ""))["value"].([]any); len(none) != 0 {
		t.Fatalf("plans for another account = %#v", none)
	}
	query := "&userEmail=ALICE@contoso.com&location=westus2"
	accounts := providerContractJSONMap(t, do(http.MethodGet, sub+"/providers/NewRelic.Observability/accounts"+azureNewRelicTestAPIVersion+query, ""))["value"].([]any)
	if len(accounts) != 1 || accounts[0].(map[string]any)["properties"].(map[string]any)["organizationId"] != org {
		t.Fatalf("accounts = %#v", accounts)
	}
	orgs := providerContractJSONMap(t, do(http.MethodGet, sub+"/providers/NewRelic.Observability/organizations"+azureNewRelicTestAPIVersion+query, ""))["value"].([]any)
	if len(orgs) != 1 || orgs[0].(map[string]any)["properties"].(map[string]any)["billingSource"] != "AZURE" {
		t.Fatalf("organizations = %#v", orgs)
	}
	// An organization created in New Relic is billed by New Relic.
	azureResourcesExpectStatus(t, do(http.MethodPut, sub+"/resourceGroups/rg-nr/providers/NewRelic.Observability/monitors/existing"+azureNewRelicTestAPIVersion, `{"location":"westus",
		"properties":{"orgCreationSource":"NEWRELIC","userInfo":{"emailAddress":"bob@contoso.com"},"newRelicAccountProperties":{"organizationInfo":{"organizationId":"nr-org"}}}}`), http.StatusCreated)
	existingOrgs := providerContractJSONMap(t, do(http.MethodGet, sub+"/providers/NewRelic.Observability/organizations"+azureNewRelicTestAPIVersion+"&userEmail=bob@contoso.com&location=westus", ""))["value"].([]any)
	if len(existingOrgs) != 1 || existingOrgs[0].(map[string]any)["properties"].(map[string]any)["billingSource"] != "NEWRELIC" || existingOrgs[0].(map[string]any)["name"] != "nr-org" {
		t.Fatalf("existing organizations = %#v", existingOrgs)
	}
	for _, q := range []string{"&userEmail=bob@contoso.com&location=westus2", "&userEmail=alice@contoso.com&location=eastus", "&location=westus2",
		// The unbilled monitor has no email, but the query needs one.
		"&location=westus"} {
		if none := providerContractJSONMap(t, do(http.MethodGet, sub+"/providers/NewRelic.Observability/accounts"+azureNewRelicTestAPIVersion+q, ""))["value"].([]any); len(none) != 0 {
			t.Fatalf("accounts%s = %#v", q, none)
		}
	}
}

func TestAzureNewRelicTagRulesAndMonitoring(t *testing.T) {
	t.Parallel()
	sub, monitor, do := newAzureNewRelicTestMonitor(t, "rg-nr")
	post := func(action, body string) map[string]any {
		return providerContractJSONMap(t, do(http.MethodPost, monitor+"/"+action+azureNewRelicTestAPIVersion, body))
	}
	vault := func(name, tags string) string {
		id := sub + "/resourceGroups/rg-nr/providers/NewRelic.Observability/monitors/" + name
		azureResourcesExpectStatus(t, do(http.MethodPut, id+azureNewRelicTestAPIVersion, `{"location":"westus","tags":`+tags+`,"properties":{}}`), http.StatusCreated)
		return id[len("/azure"):]
	}
	prod, dev := vault("prod1", `{"env":"prod"}`), vault("dev1", `{"env":"dev"}`)

	// Without tag rules nothing is monitored.
	if value := post("monitoredResources", "")["value"].([]any); len(value) != 0 {
		t.Fatalf("monitored = %#v", value)
	}
	rule := monitor + "/tagRules/default" + azureNewRelicTestAPIVersion
	resp := do(http.MethodPut, rule, `{"properties":{"logRules":{"sendActivityLogs":"Enabled","filteringTags":[{"name":"env","value":"dev","action":"Exclude"}]},
		"metricRules":{"sendMetrics":"Enabled","userEmail":"alice@contoso.com","filteringTags":[{"name":"env","value":"prod","action":"Include"}]}}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	monitored := post("monitoredResources", "")["value"].([]any)
	status := map[string][2]any{}
	for _, raw := range monitored {
		item := raw.(map[string]any)
		status[azureARMString(item["id"])] = [2]any{item["sendingMetrics"], item["sendingLogs"]}
	}
	// Only top-level resources count, not the monitor's tag rules or key.
	if len(monitored) != 2 || status[prod] != [2]any{"Enabled", "Enabled"} || status[dev] != ([2]any{}) || status[monitor[len("/azure"):]] != [2]any{"Disabled", "Enabled"} {
		t.Fatalf("monitored = %#v", monitored)
	}
	if ids := post("getMetricStatus", `{"userEmail":"alice@contoso.com"}`)["azureResourceIds"].([]any); len(ids) != 1 || ids[0] != prod {
		t.Fatalf("metric status = %#v", ids)
	}
	if ids := post("getMetricStatus", `{"userEmail":"alice@contoso.com","azureResourceIds":["`+dev+`"]}`)["azureResourceIds"].([]any); len(ids) != 0 {
		t.Fatalf("metric status for dev = %#v", ids)
	}
	rules := post("getMetricRules", `{"userEmail":"alice@contoso.com"}`)
	if rules["sendMetrics"] != "Enabled" || len(rules["filteringTags"].([]any)) != 1 || rules["userEmail"] != "alice@contoso.com" {
		t.Fatalf("metric rules = %#v", rules)
	}
	for _, action := range []string{"getMetricRules", "getMetricStatus", "listHosts", "listAppServices"} {
		azureSentinelExpectRejected(t, do(http.MethodPost, monitor+"/"+action+azureNewRelicTestAPIVersion, `{}`), "userEmail is required")
		azureSentinelExpectRejected(t, do(http.MethodPost, monitor+"/"+action+azureNewRelicTestAPIVersion, `{"userEmail":"bad"}`), "userEmail must be an email address")
	}
	if hosts := post("listHosts", `{"userEmail":"alice@contoso.com","vmIds":["vm1"]}`)["value"].([]any); len(hosts) != 0 {
		t.Fatalf("hosts = %#v", hosts)
	}
	azureSentinelExpectRejected(t, do(http.MethodPost, monitor+"/listAppServices"+azureNewRelicTestAPIVersion, `{"userEmail":"alice@contoso.com","azureResourceIds":"x"}`), "azureResourceIds must be an array of strings")

	// PATCH swaps the metric rules; metrics stop when disabled.
	patched := providerContractJSONMap(t, do(http.MethodPatch, rule, `{"properties":{"metricRules":{"sendMetrics":"Disabled"}}}`))
	if patched["properties"].(map[string]any)["metricRules"].(map[string]any)["sendMetrics"] != "Disabled" {
		t.Fatalf("patched = %#v", patched)
	}
	if rules := post("getMetricRules", `{"userEmail":"alice@contoso.com"}`); rules["sendMetrics"] != "Disabled" {
		t.Fatalf("metric rules = %#v", rules)
	}
	if ids := post("getMetricStatus", `{"userEmail":"alice@contoso.com"}`)["azureResourceIds"].([]any); len(ids) != 0 {
		t.Fatalf("metric status with metrics disabled = %#v", ids)
	}

	// Monitored subscriptions extend the scope to other subscriptions.
	otherSub := "11111111-1111-1111-1111-111111111111"
	azureResourcesExpectStatus(t, do(http.MethodPut, "/azure/subscriptions/"+otherSub+"/resourceGroups/rg-o"+azureNewRelicTestAPIVersion, `{"location":"westus"}`), http.StatusCreated)
	remote := "/subscriptions/" + otherSub + "/resourceGroups/rg-o/providers/NewRelic.Observability/monitors/remote"
	azureResourcesExpectStatus(t, do(http.MethodPut, "/azure"+remote+azureNewRelicTestAPIVersion, `{"location":"westus","properties":{}}`), http.StatusCreated)
	has := func(id string) bool {
		for _, raw := range post("monitoredResources", "")["value"].([]any) {
			if strings.EqualFold(azureARMString(raw.(map[string]any)["id"]), id) {
				return true
			}
		}
		return false
	}
	if has(remote) {
		t.Fatal("unmonitored subscription's resource listed")
	}
	subs := monitor + "/monitoredSubscriptions/default" + azureNewRelicTestAPIVersion
	created := providerContractJSONMap(t, do(http.MethodPut, subs, `{"properties":{"patchOperation":"AddBegin","monitoredSubscriptionList":[{"subscriptionId":"/subscriptions/`+otherSub+`"}]}}`))
	list := created["properties"].(map[string]any)["monitoredSubscriptionList"].([]any)
	if len(list) != 1 || list[0].(map[string]any)["status"] != "Active" || !has(remote) {
		t.Fatalf("monitored subscriptions = %#v", created)
	}
	third := "22222222-2222-2222-2222-222222222222"
	added := providerContractJSONMap(t, do(http.MethodPatch, subs, `{"properties":{"patchOperation":"AddComplete","monitoredSubscriptionList":[{"subscriptionId":"`+third+`"}]}}`))
	if list := added["properties"].(map[string]any)["monitoredSubscriptionList"].([]any); len(list) != 2 {
		t.Fatalf("added = %#v", added)
	}
	// Adding listed subscriptions again, in any form, keeps one of each.
	readded := providerContractJSONMap(t, do(http.MethodPatch, subs, `{"properties":{"patchOperation":"AddBegin","monitoredSubscriptionList":[{"subscriptionId":"`+strings.ToUpper(otherSub)+`"},{"subscriptionId":"/subscriptions/`+otherSub+`"}]}}`))
	if list := readded["properties"].(map[string]any)["monitoredSubscriptionList"].([]any); len(list) != 2 {
		t.Fatalf("re-added = %#v", readded)
	}
	removed := providerContractJSONMap(t, do(http.MethodPatch, subs, `{"properties":{"patchOperation":"DeleteComplete","monitoredSubscriptionList":[{"subscriptionId":"/subscriptions/`+otherSub+`"}]}}`))
	if list := removed["properties"].(map[string]any)["monitoredSubscriptionList"].([]any); len(list) != 1 || list[0].(map[string]any)["subscriptionId"] != third || has(remote) {
		t.Fatalf("removed = %#v", removed)
	}
	replaced := providerContractJSONMap(t, do(http.MethodPatch, subs, `{"properties":{"monitoredSubscriptionList":[{"subscriptionId":"`+otherSub+`"}]}}`))
	// The last DeleteComplete was applied, so no operation replaces the list.
	if props := replaced["properties"].(map[string]any); len(props["monitoredSubscriptionList"].([]any)) != 1 || props["patchOperation"] != "Active" || !has(remote) {
		t.Fatalf("replaced = %#v", replaced)
	}
	if listed := providerContractJSONMap(t, do(http.MethodGet, monitor+"/monitoredSubscriptions"+azureNewRelicTestAPIVersion, ""))["value"].([]any); len(listed) != 1 {
		t.Fatalf("listed = %#v", listed)
	}
	resp = do(http.MethodPut, monitor+"/monitoredSubscriptions/other"+azureNewRelicTestAPIVersion, `{"properties":{}}`)
	azureResourcesExpectStatus(t, resp, http.StatusBadRequest)
	for _, tc := range []struct{ method, path, body, want string }{
		{http.MethodPut, subs, `{"properties":{"patchOperation":"Replace"}}`, "patchOperation"},
		{http.MethodPut, subs, `{"properties":{"monitoredSubscriptionList":[{"subscriptionId":"abc"}]}}`, "subscriptionId must be a subscription ID"},
		{http.MethodPut, subs, `{"properties":{"monitoredSubscriptionList":"x"}}`, "monitoredSubscriptionList must be an array"},
		{http.MethodPatch, subs, `{"tags":{}}`, "tags can't be updated"},
		{http.MethodPatch, subs, `{"properties":{"monitoredSubscriptionList":[{}]}}`, "subscriptionId must be a subscription ID"},
		{http.MethodPut, rule, `{"properties":{"logRules":{"sendAadLogs":"On"}}}`, "sendAadLogs"},
		{http.MethodPut, rule, `{"properties":{"metricRules":{"sendMetrics":"On"}}}`, "sendMetrics"},
		{http.MethodPut, rule, `{"properties":{"metricRules":{"userEmail":"x"}}}`, "metricRules.userEmail must be an email address"},
		{http.MethodPut, rule, `{"properties":{"logRules":{"filteringTags":{}}}}`, "logRules.filteringTags must be an array"},
		{http.MethodPut, rule, `{"properties":{"logRules":{"filteringTags":["env"]}}}`, "filteringTags items must be objects"},
		{http.MethodPut, rule, `{"properties":{"metricRules":{"filteringTags":[{"name":1}]}}}`, "filteringTags[].name must be a string"},
		{http.MethodPut, rule, `{"properties":{"metricRules":{"filteringTags":[{"action":"Keep"}]}}}`, "action"},
		{http.MethodPut, rule, `{"properties":{"logRules":"x"}}`, "logRules must be an object"},
		{http.MethodPatch, rule, `{"properties":{"provisioningState":"Failed"}}`, "provisioningState can't be updated"},
		{http.MethodPatch, rule, `{"properties":{"logRules":{"sendActivityLogs":"Yes"}}}`, "sendActivityLogs"},
	} {
		azureSentinelExpectRejected(t, do(tc.method, tc.path, tc.body), tc.want)
	}
}

func TestAzureNewRelicTagged(t *testing.T) {
	t.Parallel()
	tags := map[string]any{"Env": "prod", "team": "a"}
	for _, tc := range []struct {
		filtering []any
		want      bool
	}{
		{nil, true},
		{[]any{map[string]any{"name": "env", "value": "prod", "action": "Include"}}, true},
		{[]any{map[string]any{"name": "env", "value": "dev", "action": "Include"}}, false},
		{[]any{map[string]any{"name": "env", "action": "Include"}}, true},
		{[]any{map[string]any{"name": "team", "value": "a", "action": "Exclude"}}, false},
		{[]any{map[string]any{"name": "team", "value": "b", "action": "Exclude"}}, true},
		// Exclusion wins over a matching inclusion.
		{[]any{map[string]any{"name": "env", "value": "prod", "action": "Include"}, map[string]any{"name": "team", "value": "a", "action": "Exclude"}}, false},
	} {
		if got := azureNewRelicTagged(tc.filtering, tags); got != tc.want {
			t.Errorf("azureNewRelicTagged(%v) = %v, want %v", tc.filtering, got, tc.want)
		}
	}
}
