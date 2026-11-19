package server

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

// NewRelic.Observability: Azure Native New Relic Service monitors with
// their tag rules and monitored subscriptions. Nothing is sent to New
// Relic: the monitor's tag rules decide which stored resources report as
// sending metrics and logs, and no agents are ever installed.
const azureNewRelicNamespace = "NewRelic.Observability"

const azureNewRelicMonitorType = azureNewRelicNamespace + "/monitors"

func init() {
	ns := azureNewRelicNamespace
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "monitors",
		Validate:            azureNewRelicMonitorValidate,
		PatchBody:           azureNewRelicMonitorPatch,
		WriteOnlyProperties: []string{"newRelicAccountProperties.accountInfo.ingestionKey"},
		ReadOnlyProperties: func(rec *azureARMResource) map[string]any {
			return map[string]any{
				"provisioningState": "Succeeded", "monitoringStatus": "Enabled", "marketplaceSubscriptionStatus": "Active",
				"liftrResourceCategory": "MonitorLogs", "marketplaceSubscriptionId": azureNewRelicID(rec.ID, "marketplace"),
			}
		},
		Reconcile: azureNewRelicMonitorReconcile,
		Linked:    azureNewRelicMonitorLinked,
		ServerActions: map[string]func(*Server, *azureARMResource, map[string]any) (int, any){
			"getmetricrules":                azureNewRelicMetricRules,
			"getmetricstatus":               azureNewRelicMetricStatus,
			"monitoredresources":            azureNewRelicMonitoredResources,
			"vmhostpayloads":                azureNewRelicVMHostPayload,
			"refreshingestionkey":           azureNewRelicRefreshKey,
			"switchbilling":                 azureNewRelicSwitchBilling,
			"resubscribe":                   azureNewRelicResubscribe,
			"linksaas":                      azureNewRelicLinkSaaS,
			"listlinkedresources":           azureNewRelicLinkedResources,
			"listconnectedpartnerresources": azureNewRelicConnectedResources,
		},
		StringBodyActions: map[string]bool{"listconnectedpartnerresources": true},
		Actions: map[string]azureARMAction{
			// ponytail: the emulator installs no New Relic agents, so no VM or
			// app service ever reports one.
			"listhosts":       azureNewRelicNoAgents("vmIds"),
			"listappservices": azureNewRelicNoAgents("azureResourceIds"),
			"latestlinkedsaas": func(rec *azureARMResource, _ map[string]any) (int, any) {
				out := map[string]any{"isHiddenSaaS": false}
				if id := azureNewRelicSaaS(rec); id != "" {
					out["saaSResourceId"] = id
				}
				return http.StatusOK, out
			},
			"getbillinginfo": azureNewRelicBillingInfo,
		},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "monitors/tagRules", Proxy: true,
		Validate: func(payload map[string]any) string {
			if msg := azureDataBoxObjects(payload, "", "properties"); msg != "" {
				return msg
			}
			return azureNewRelicTagRule(azureARMObject(payload["properties"]))
		},
		PatchBody: azureNewRelicTagRulePatch,
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "monitors/monitoredSubscriptions", Proxy: true,
		ValidateName: func(name string) string {
			if name != "default" {
				return "The configuration name must be 'default'."
			}
			return ""
		},
		Validate: azureNewRelicSubscriptionsValidate,
		PatchBody: func(payload map[string]any) (map[string]any, map[string]any, string) {
			for key := range payload {
				if key != "properties" {
					return nil, nil, key + " can't be updated."
				}
			}
			return payload, nil, azureNewRelicSubscriptionsValidate(payload)
		},
		Reconcile: azureNewRelicSubscriptionsReconcile,
	})
	registerAzureARMQueryView(ns, "accounts", azureNewRelicAccounts)
	registerAzureARMQueryView(ns, "organizations", azureNewRelicOrganizations)
	registerAzureARMQueryView(ns, "plans", azureNewRelicPlans)
}

// azureNewRelicID returns a stable GUID derived from a resource ID.
func azureNewRelicID(id, label string) string {
	return uuid.NewSHA1(uuid.NameSpaceURL, []byte("stackyard:newrelic-"+label+":"+azureARMKey(id))).String()
}

// azureNewRelicAccount returns a monitor's New Relic account and
// organization IDs.
func azureNewRelicAccount(rec *azureARMResource) (account, org string) {
	props := azureARMObject(rec.Properties["newRelicAccountProperties"])
	return azureARMString(azureARMObject(props["accountInfo"])["accountId"]), azureARMString(azureARMObject(props["organizationInfo"])["organizationId"])
}

func azureNewRelicSaaS(rec *azureARMResource) string {
	return azureARMString(azureARMObject(rec.Properties["saaSData"])["saaSResourceId"])
}

// azureNewRelicMonitorReconcile creates a new account and organization for
// monitors that don't link existing ones, and keeps the organization:
// Learn's FAQ says it can't change after creation.
func azureNewRelicMonitorReconcile(_ *Server, existing, rec *azureARMResource) (map[string]any, string) {
	props := azureARMObject(rec.Properties["newRelicAccountProperties"])
	accountInfo := azureARMMerge(azureARMObject(props["accountInfo"]), nil)
	account, org := azureNewRelicAccount(rec)
	if existing != nil {
		storedAccount, storedOrg := azureNewRelicAccount(existing)
		if org != "" && org != storedOrg {
			return nil, "The New Relic organization of a monitor can't be changed after creation."
		}
		org = storedOrg
		if account == "" {
			account = storedAccount
		}
	}
	if account == "" {
		sum := sha256.Sum256([]byte(azureARMKey(rec.ID) + "#account"))
		account = fmt.Sprint(1000000 + binary.BigEndian.Uint32(sum[:4])%9000000)
	}
	if org == "" {
		org = azureNewRelicID(rec.ID, "organization")
	}
	accountInfo["accountId"] = account
	if azureARMString(accountInfo["region"]) == "" {
		accountInfo["region"] = rec.Location
	}
	out := azureARMMerge(props, map[string]any{"accountInfo": accountInfo, "organizationInfo": map[string]any{"organizationId": org}})
	return map[string]any{"newRelicAccountProperties": out}, ""
}

func azureNewRelicKeyID(monitorID string) string { return monitorID + "/ingestionKey/default" }

// azureNewRelicMonitorLinked keeps the account's ingestion key, which GET
// never returns: the one the client sent, else the monitor's current one,
// else the one another monitor of the account uses, else a new one.
func azureNewRelicMonitorLinked(s *Server, rec *azureARMResource) (map[string]any, []*azureARMResource, string) {
	props := azureARMObject(rec.Properties["newRelicAccountProperties"])
	key := azureARMString(azureARMObject(props["accountInfo"])["ingestionKey"])
	if key == "" {
		key = s.azureNewRelicKeyLocked(rec.ID)
	}
	if key == "" {
		account, _ := azureNewRelicAccount(rec)
		for _, other := range s.azureNewRelicMonitorsLocked(func(m *azureARMResource) bool {
			a, _ := azureNewRelicAccount(m)
			return a == account
		}) {
			if key = s.azureNewRelicKeyLocked(other.ID); key != "" {
				break
			}
		}
	}
	if key == "" {
		key = azureARMDerivedKey(rec.ID, "ingestion")
	}
	return nil, []*azureARMResource{azureNewRelicKeyRecord(rec, key)}, ""
}

func azureNewRelicKeyRecord(monitor *azureARMResource, key string) *azureARMResource {
	return &azureARMResource{
		ID: azureNewRelicKeyID(monitor.ID), Type: azureNewRelicMonitorType + "/ingestionKey", Name: "default",
		SubscriptionID: monitor.SubscriptionID, ResourceGroup: monitor.ResourceGroup,
		Properties: map[string]any{"ingestionKey": key}, CreatedAt: monitor.CreatedAt, UpdatedAt: time.Now().UTC(),
	}
}

func (s *Server) azureNewRelicKeyLocked(monitorID string) string {
	if stored := s.azureARMResources[azureARMKey(azureNewRelicKeyID(monitorID))]; stored != nil {
		return azureARMString(stored.Properties["ingestionKey"])
	}
	return ""
}

// azureNewRelicMonitorsLocked returns matching monitors in every
// subscription, sorted by ID.
func (s *Server) azureNewRelicMonitorsLocked(match func(*azureARMResource) bool) []*azureARMResource {
	monitors := s.azureARMResourcesMatchingLocked(func(rec *azureARMResource) bool {
		return strings.EqualFold(rec.Type, azureNewRelicMonitorType) && match(rec)
	})
	sort.Slice(monitors, func(i, j int) bool { return azureARMKey(monitors[i].ID) < azureARMKey(monitors[j].ID) })
	return monitors
}

func (s *Server) azureNewRelicMonitors(match func(*azureARMResource) bool) []*azureARMResource {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	return s.azureNewRelicMonitorsLocked(match)
}

func azureNewRelicVMHostPayload(s *Server, rec *azureARMResource, _ map[string]any) (int, any) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	return http.StatusOK, map[string]any{"ingestionKey": s.azureNewRelicKeyLocked(rec.ID)}
}

// azureNewRelicRefreshKey gives every monitor linked to the monitor's
// account a new ingestion key.
func azureNewRelicRefreshKey(s *Server, rec *azureARMResource, _ map[string]any) (int, any) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	account, _ := azureNewRelicAccount(rec)
	key := azureARMDerivedKey(rec.ID, "ingestion#"+time.Now().UTC().Format(time.RFC3339Nano))
	for _, monitor := range s.azureNewRelicMonitorsLocked(func(m *azureARMResource) bool {
		a, _ := azureNewRelicAccount(m)
		return a == account
	}) {
		s.azureARMResources[azureARMKey(azureNewRelicKeyID(monitor.ID))] = azureNewRelicKeyRecord(monitor, key)
	}
	return http.StatusNoContent, nil
}

// azureNewRelicUpdate stores the monitor with props merged in and returns
// it, or 404 if it was deleted meanwhile.
func (s *Server) azureNewRelicUpdate(rec *azureARMResource, props map[string]any) (int, any) {
	typ, _ := lookupAzureARMResourceType(azureNewRelicNamespace, "monitors")
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	current := s.azureARMResources[azureARMKey(rec.ID)]
	if current == nil {
		return http.StatusNotFound, azureARMErrorBody("ResourceNotFound", "The resource '"+rec.ID+"' was not found.")
	}
	updated := *current
	updated.UpdatedAt = time.Now().UTC()
	updated.Properties = azureARMProperties(typ, &azureARMResource{ID: current.ID, Properties: azureARMMerge(current.Properties, props)})
	s.azureARMResources[azureARMKey(rec.ID)] = &updated
	return http.StatusOK, updated.fixture()
}

func azureNewRelicSwitchBilling(s *Server, rec *azureARMResource, payload map[string]any) (int, any) {
	if msg := azureNewRelicEmail(payload, "", true); msg != "" {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", msg)
	}
	if msg := azureNewRelicSameMonitor(rec, payload); msg != "" {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", msg)
	}
	if id, present := payload["azureResourceId"]; present && !strings.EqualFold(azureARMString(id), rec.ID) {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "azureResourceId must be this monitor's ID.")
	}
	if msg := azureDataBoxObjects(payload, "", "planData"); msg != "" {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", msg)
	}
	plan, present := payload["planData"].(map[string]any)
	if !present {
		return http.StatusNoContent, nil
	}
	if msg := azureNewRelicPlan(plan, "planData."); msg != "" {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", msg)
	}
	return s.azureNewRelicUpdate(rec, map[string]any{"planData": plan})
}

// azureNewRelicSameMonitor rejects an organizationId other than the
// monitor's, which can't change.
func azureNewRelicSameMonitor(rec *azureARMResource, payload map[string]any) string {
	_, org := azureNewRelicAccount(rec)
	if raw, present := payload["organizationId"]; present && azureARMString(raw) != org {
		return "organizationId must be the monitor's New Relic organization."
	}
	return ""
}

// azureNewRelicResubscribe renews the Marketplace subscription, taking the
// newly selected plan as the monitor's plan.
func azureNewRelicResubscribe(s *Server, rec *azureARMResource, payload map[string]any) (int, any) {
	for _, key := range []string{"offerId", "organizationId", "planId", "publisherId", "resourceGroup", "termId"} {
		if raw, present := payload[key]; present {
			if _, ok := raw.(string); !ok {
				return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", key+" must be a string.")
			}
		}
	}
	if msg := azureNewRelicSameMonitor(rec, payload); msg != "" {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", msg)
	}
	props := map[string]any{}
	if plan := azureARMString(payload["planId"]); plan != "" {
		props["planData"] = map[string]any{"planDetails": plan}
	}
	return s.azureNewRelicUpdate(rec, props)
}

func azureNewRelicLinkSaaS(s *Server, rec *azureARMResource, payload map[string]any) (int, any) {
	id := azureARMString(payload["saaSResourceId"])
	if !azureNewRelicIsSaaS(id) {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "saaSResourceId must be a Microsoft.SaaS/resources resource ID.")
	}
	return s.azureNewRelicUpdate(rec, map[string]any{"saaSData": map[string]any{"saaSResourceId": id}})
}

// azureNewRelicLinkedResources lists the monitors linked to the monitor's
// New Relic organization, in any subscription.
func azureNewRelicLinkedResources(s *Server, rec *azureARMResource, _ map[string]any) (int, any) {
	_, org := azureNewRelicAccount(rec)
	value := []any{}
	for _, monitor := range s.azureNewRelicMonitors(func(m *azureARMResource) bool { _, o := azureNewRelicAccount(m); return o == org }) {
		value = append(value, map[string]any{"id": monitor.ID})
	}
	return http.StatusOK, map[string]any{"value": value}
}

// azureNewRelicConnectedResources lists the monitors billed through the
// monitor's Marketplace SaaS resource; one without a SaaS resource lists
// itself. Learn's request body is a bare email string.
func azureNewRelicConnectedResources(s *Server, rec *azureARMResource, payload map[string]any) (int, any) {
	if msg := azureNewRelicEmail(payload, "body", false); msg != "" {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", msg)
	}
	saas := azureNewRelicSaaS(rec)
	value := []any{}
	for _, monitor := range s.azureNewRelicMonitors(func(m *azureARMResource) bool {
		if saas == "" {
			return strings.EqualFold(m.ID, rec.ID)
		}
		return strings.EqualFold(azureNewRelicSaaS(m), saas)
	}) {
		account, _ := azureNewRelicAccount(monitor)
		value = append(value, map[string]any{"properties": map[string]any{"accountId": account, "azureResourceId": monitor.ID, "location": monitor.Location}})
	}
	return http.StatusOK, map[string]any{"value": value}
}

func azureNewRelicBillingInfo(rec *azureARMResource, _ map[string]any) (int, any) {
	_, org := azureNewRelicAccount(rec)
	saas := map[string]any{"marketplaceSubscriptionId": rec.Properties["marketplaceSubscriptionId"], "billedAzureSubscriptionId": rec.SubscriptionID}
	if id := azureNewRelicSaaS(rec); id != "" {
		saas["marketplaceResourceId"] = id
	}
	return http.StatusOK, map[string]any{"marketplaceSaasInfo": saas, "partnerBillingEntity": map[string]any{"organizationId": org}}
}

func azureNewRelicNoAgents(idsKey string) azureARMAction {
	return func(_ *azureARMResource, payload map[string]any) (int, any) {
		if msg := azureNewRelicEmail(payload, "", true); msg != "" {
			return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", msg)
		}
		if msg := azureImpactStrings(payload, "", idsKey); msg != "" {
			return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", msg)
		}
		return http.StatusOK, map[string]any{"value": []any{}}
	}
}

// azureNewRelicTagged reports whether tags pass a rule's filtering tags:
// an Exclude match drops the resource, and Include tags, when there are
// any, must match. No filtering tags capture everything.
func azureNewRelicTagged(filtering []any, tags map[string]any) bool {
	hasInclude, included := false, false
	for _, raw := range filtering {
		ft := azureARMObject(raw)
		match := false
		for name, value := range tags {
			if strings.EqualFold(name, azureARMString(ft["name"])) && (ft["value"] == nil || azureARMString(ft["value"]) == azureARMString(value)) {
				match = true
			}
		}
		switch azureARMString(ft["action"]) {
		case "Exclude":
			if match {
				return false
			}
		case "Include":
			hasInclude = true
			included = included || match
		}
	}
	return !hasInclude || included
}

// azureNewRelicMonitored lists the stored top-level resources in the
// monitor's subscription and its monitored subscriptions that its tag
// rules capture for metrics (sendMetrics) or logs (sendActivityLogs).
func (s *Server) azureNewRelicMonitored(rec *azureARMResource) []map[string]any {
	tagRules, _ := lookupAzureARMResourceType(azureNewRelicNamespace, "monitors/tagRules")
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	rules := s.azureARMInlineChildrenLocked(rec.ID, tagRules)
	subs := map[string]bool{strings.ToLower(rec.SubscriptionID): true}
	if monitored := s.azureARMResources[azureARMKey(rec.ID+"/monitoredSubscriptions/default")]; monitored != nil {
		for _, item := range azureARMSlice(monitored.Properties["monitoredSubscriptionList"]) {
			subs[azureNewRelicSubscription(azureARMObject(item)["subscriptionId"])] = true
		}
	}
	sending := func(res *azureARMResource, rulesKey, flag string) string {
		for _, rule := range rules {
			r := azureARMObject(rule.Properties[rulesKey])
			if r[flag] == "Enabled" && azureNewRelicTagged(azureARMSlice(r["filteringTags"]), res.Tags) {
				return "Enabled"
			}
		}
		return "Disabled"
	}
	var out []map[string]any
	for _, res := range s.azureARMResourcesMatchingLocked(func(res *azureARMResource) bool {
		return res.ResourceGroup != "" && strings.Count(res.ID, "/") == 8 && subs[strings.ToLower(res.SubscriptionID)]
	}) {
		metrics, logs := sending(res, "metricRules", "sendMetrics"), sending(res, "logRules", "sendActivityLogs")
		if metrics == "Enabled" || logs == "Enabled" {
			out = append(out, map[string]any{"id": res.ID, "sendingMetrics": metrics, "sendingLogs": logs})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return azureARMKey(azureARMString(out[i]["id"])) < azureARMKey(azureARMString(out[j]["id"]))
	})
	return out
}

func azureNewRelicMonitoredResources(s *Server, rec *azureARMResource, _ map[string]any) (int, any) {
	value := []any{}
	for _, item := range s.azureNewRelicMonitored(rec) {
		value = append(value, item)
	}
	return http.StatusOK, map[string]any{"value": value}
}

// azureNewRelicMetricStatus returns which of the requested resources (all
// captured ones when none are named) send metrics.
func azureNewRelicMetricStatus(s *Server, rec *azureARMResource, payload map[string]any) (int, any) {
	if msg := azureNewRelicEmail(payload, "", true); msg != "" {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", msg)
	}
	if msg := azureImpactStrings(payload, "", "azureResourceIds"); msg != "" {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", msg)
	}
	requested, filter := payload["azureResourceIds"].([]any)
	ids := []any{}
	for _, item := range s.azureNewRelicMonitored(rec) {
		id := azureARMString(item["id"])
		if item["sendingMetrics"] == "Enabled" && (!filter || slices.ContainsFunc(requested, func(r any) bool { return strings.EqualFold(azureARMString(r), id) })) {
			ids = append(ids, id)
		}
	}
	return http.StatusOK, map[string]any{"azureResourceIds": ids}
}

// azureNewRelicMetricRules combines the metric rules of the monitor's tag
// rule sets.
func azureNewRelicMetricRules(s *Server, rec *azureARMResource, payload map[string]any) (int, any) {
	if msg := azureNewRelicEmail(payload, "", true); msg != "" {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", msg)
	}
	tagRules, _ := lookupAzureARMResourceType(azureNewRelicNamespace, "monitors/tagRules")
	s.providerStorageMu.Lock()
	rules := s.azureARMInlineChildrenLocked(rec.ID, tagRules)
	s.providerStorageMu.Unlock()
	filtering, send := []any{}, "Disabled"
	for _, rule := range rules {
		metrics := azureARMObject(rule.Properties["metricRules"])
		filtering = append(filtering, azureARMSlice(metrics["filteringTags"])...)
		if metrics["sendMetrics"] == "Enabled" {
			send = "Enabled"
		}
	}
	return http.StatusOK, map[string]any{"filteringTags": filtering, "sendMetrics": send, "userEmail": payload["userEmail"]}
}

// azureNewRelicSubscription normalizes a monitored subscription ID, sent
// bare or as /subscriptions/{id}, to the lower-cased GUID.
func azureNewRelicSubscription(raw any) string {
	return strings.TrimPrefix(strings.ToLower(azureARMString(raw)), "/subscriptions/")
}

// azureNewRelicSubscriptionsReconcile applies patchOperation to the stored
// list: AddBegin and AddComplete add the sent subscriptions, DeleteBegin
// and DeleteComplete remove them, and Active (or none) replaces the list.
// Every listed subscription is monitored. On PATCH the stored operation
// is always Active, so only the sent one counts.
func azureNewRelicSubscriptionsReconcile(_ *Server, existing, rec *azureARMResource) (map[string]any, string) {
	sent := azureARMSlice(rec.Properties["monitoredSubscriptionList"])
	var list []any
	if existing != nil {
		list = azureARMSlice(existing.Properties["monitoredSubscriptionList"])
	}
	inSent := func(item any) bool {
		id := azureNewRelicSubscription(azureARMObject(item)["subscriptionId"])
		return slices.ContainsFunc(sent, func(s any) bool { return azureNewRelicSubscription(azureARMObject(s)["subscriptionId"]) == id })
	}
	switch azureARMString(rec.Properties["patchOperation"]) {
	case "AddBegin", "AddComplete":
		list = append(slices.DeleteFunc(slices.Clone(list), inSent), sent...)
	case "DeleteBegin", "DeleteComplete":
		list = slices.DeleteFunc(slices.Clone(list), inSent)
	default:
		list = sent
	}
	out := make([]any, 0, len(list))
	seen := map[string]bool{}
	for _, item := range list {
		id := azureARMString(azureARMObject(item)["subscriptionId"])
		if !seen[azureNewRelicSubscription(id)] {
			seen[azureNewRelicSubscription(id)] = true
			out = append(out, map[string]any{"subscriptionId": id, "status": "Active"})
		}
	}
	// The operation is applied, so a later update without one replaces
	// the list rather than repeating it.
	return map[string]any{"monitoredSubscriptionList": out, "patchOperation": "Active"}, ""
}

// azureNewRelicMonitorsFor returns the monitors whose user email and
// location match the accounts and organizations query, which requires
// both.
func (s *Server) azureNewRelicMonitorsFor(query url.Values) []*azureARMResource {
	email, location := query.Get("userEmail"), azureNewRelicLocation(query.Get("location"))
	if email == "" || location == "" {
		return nil
	}
	return s.azureNewRelicMonitors(func(m *azureARMResource) bool {
		return strings.EqualFold(azureARMString(azureARMObject(m.Properties["userInfo"])["emailAddress"]), email) && azureNewRelicLocation(m.Location) == location
	})
}

func azureNewRelicLocation(location string) string {
	return strings.ToLower(strings.ReplaceAll(location, " ", ""))
}

func azureNewRelicAccounts(s *Server, scope string, query url.Values) []map[string]any {
	var out []map[string]any
	seen := map[string]bool{}
	for _, m := range s.azureNewRelicMonitorsFor(query) {
		account, org := azureNewRelicAccount(m)
		if !seen[account] {
			seen[account] = true
			region := azureARMObject(azureARMObject(m.Properties["newRelicAccountProperties"])["accountInfo"])["region"]
			out = append(out, map[string]any{
				"id": scope + "/providers/" + azureNewRelicNamespace + "/accounts/" + account, "name": account, "type": azureNewRelicNamespace + "/accounts",
				"properties": map[string]any{"accountId": account, "organizationId": org, "region": region},
			})
		}
	}
	return out
}

// azureNewRelicOrganizations lists the organizations; ones created in New
// Relic rather than through Azure are billed by New Relic (Learn FAQ).
func azureNewRelicOrganizations(s *Server, scope string, query url.Values) []map[string]any {
	var out []map[string]any
	seen := map[string]bool{}
	for _, m := range s.azureNewRelicMonitorsFor(query) {
		_, org := azureNewRelicAccount(m)
		if !seen[org] {
			seen[org] = true
			billing := "AZURE"
			if m.Properties["orgCreationSource"] == "NEWRELIC" {
				billing = "NEWRELIC"
			}
			out = append(out, map[string]any{
				"id": scope + "/providers/" + azureNewRelicNamespace + "/organizations/" + org, "name": org, "type": azureNewRelicNamespace + "/organizations",
				"properties": map[string]any{"organizationId": org, "billingSource": billing},
			})
		}
	}
	return out
}

// azureNewRelicPlans lists the plans of the subscription's monitors,
// optionally only those of one account or organization.
func azureNewRelicPlans(s *Server, scope string, query url.Values) []map[string]any {
	sub, ok := azureARMSubscriptionScope(scope)
	if !ok {
		return nil
	}
	var out []map[string]any
	for _, m := range s.azureNewRelicMonitors(func(m *azureARMResource) bool {
		account, org := azureNewRelicAccount(m)
		return strings.EqualFold(m.SubscriptionID, sub) && m.Properties["planData"] != nil &&
			(query.Get("accountId") == "" || query.Get("accountId") == account) && (query.Get("organizationId") == "" || query.Get("organizationId") == org)
	}) {
		props := map[string]any{"planData": m.Properties["planData"]}
		for _, key := range []string{"accountCreationSource", "orgCreationSource"} {
			if v, present := m.Properties[key]; present {
				props[key] = v
			}
		}
		out = append(out, map[string]any{
			"id": scope + "/providers/" + azureNewRelicNamespace + "/plans/" + m.Name, "name": m.Name, "type": azureNewRelicNamespace + "/plans", "properties": props,
		})
	}
	return out
}
