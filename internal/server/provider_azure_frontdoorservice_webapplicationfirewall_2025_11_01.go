package server

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"time"
)

// Front Door web application firewall and network experiments
// (Microsoft.Network, api-version 2025-11-01), served by the shared ARM
// handler: WAF policies, the managed rule set catalog, rules engines on
// front doors, and network experiment profiles with their experiments and
// reports. Front doors and their frontend endpoints, which this version
// also covers, are registered once in the classic Front Door file. No
// request is inspected and no latency is measured: reports are synthetic.

var (
	azureAfdwPolicyName = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9]{0,127}$`)
	azureAfdwRuleName   = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9]{0,127}$`)
	azureAfdwActions    = []string{"Allow", "Block", "Log", "Redirect", "AnomalyScoring", "JSChallenge", "CAPTCHA"}
	azureAfdwVariables  = []string{"RemoteAddr", "RequestMethod", "QueryString", "PostArgs", "RequestUri", "RequestHeader", "RequestBody", "Cookies", "SocketAddr"}
	azureAfdwOperators  = []string{"Any", "IPMatch", "GeoMatch", "Equal", "Contains", "LessThan", "GreaterThan", "LessThanOrEqual", "GreaterThanOrEqual", "BeginsWith", "EndsWith", "RegEx", "ServiceTagMatch"}
	// azureAfdwManagedRuleSets is the catalog: rule set type, version and
	// rule group names.
	azureAfdwManagedRuleSets = []struct {
		typ, version string
		groups       []string
	}{
		{"DefaultRuleSet", "1.0", []string{"SQLI", "XSS", "RCE", "LFI", "RFI", "PHP", "JAVA", "PROTOCOL-ATTACK", "METHOD-ENFORCEMENT"}},
		{"Microsoft_DefaultRuleSet", "2.1", []string{"SQLI", "XSS", "RCE", "LFI", "RFI", "PHP", "JAVA", "PROTOCOL", "MS-ThreatIntel-WebShells", "General"}},
		{"Microsoft_BotManagerRuleSet", "1.1", []string{"BadBots", "GoodBots", "UnknownBots"}},
	}
)

// azureAfdwMaxRules bounds custom rules, match conditions, rules engine
// rules and managed rule sets.
const azureAfdwMaxRules = 100

func init() {
	ns := azureAfdcNamespace
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "FrontDoorWebApplicationFirewallPolicies", ETag: true,
		ValidateName: func(name string) string {
			if !azureAfdwPolicyName.MatchString(name) {
				return "The policy name is invalid: use up to 128 letters and digits, starting with a letter."
			}
			return ""
		},
		Validate: azureAfdwValidatePolicy,
		ReadOnlyProperties: func(*azureARMResource) map[string]any {
			// ponytail: links stay empty; derive them from front doors if clients read them.
			return map[string]any{"resourceState": "Enabled", "frontendEndpointLinks": []any{}, "routingRuleLinks": []any{}, "securityPolicyLinks": []any{}}
		},
	})
	registerAzureARMView(ns, "FrontDoorWebApplicationFirewallManagedRuleSets", azureAfdwManagedRuleSetView)
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "frontDoors/rulesEngines", Proxy: true, NoPatch: true,
		Validate:           azureAfdwValidateRulesEngine,
		ReadOnlyProperties: func(*azureARMResource) map[string]any { return map[string]any{"resourceState": "Enabled"} },
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "NetworkExperimentProfiles",
		Validate: func(payload map[string]any) string {
			return azureWorkloadsEnums(azureARMObject(payload["properties"]), "properties.", nil, map[string][]string{"enabledState": {"Enabled", "Disabled"}})
		},
		ReadOnlyProperties: func(*azureARMResource) map[string]any { return map[string]any{"resourceState": "Enabled"} },
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "NetworkExperimentProfiles/Experiments",
		Validate:           azureAfdwValidateExperiment,
		ReadOnlyProperties: azureAfdwExperimentProperties,
	})
	registerAzureARMView(ns, "NetworkExperimentProfiles/PreconfiguredEndpoints", func(_ *Server, scope string) []map[string]any {
		var out []map[string]any
		for i, region := range []string{"eastus", "westeurope", "southeastasia"} {
			name := fmt.Sprintf("endpoint-%d", i+1)
			out = append(out, map[string]any{
				"id": scope + "/PreconfiguredEndpoints/" + name, "name": name, "type": "Microsoft.Network/NetworkExperimentProfiles/PreconfiguredEndpoints",
				"properties": map[string]any{"description": "Azure " + region, "endpoint": region + ".probe.stackyard.local", "endpointType": "AFD", "backend": "Azure " + region},
			})
		}
		return out
	})
}

func azureAfdwValidatePolicy(payload map[string]any) string {
	if sku := azureARMObject(payload["sku"]); len(sku) > 0 {
		if msg := azureWorkloadsEnums(sku, "sku.", []string{"name"}, map[string][]string{"name": {"Classic_AzureFrontDoor", "Standard_AzureFrontDoor", "Premium_AzureFrontDoor"}}); msg != "" {
			return msg
		}
	}
	props := azureARMObject(payload["properties"])
	settings := azureARMObject(props["policySettings"])
	if msg := azureWorkloadsEnums(settings, "properties.policySettings.", nil, map[string][]string{
		"enabledState": {"Enabled", "Disabled"}, "mode": {"Prevention", "Detection"}, "requestBodyCheck": {"Enabled", "Disabled"},
	}); msg != "" {
		return msg
	}
	if raw, ok := settings["customBlockResponseStatusCode"]; ok && !slices.Contains([]any{200.0, 403.0, 405.0, 406.0, 429.0}, raw) {
		return "properties.policySettings.customBlockResponseStatusCode is invalid: use 200, 403, 405, 406 or 429."
	}
	rules := azureARMSlice(azureARMObject(props["customRules"])["rules"])
	if len(rules) > azureAfdwMaxRules {
		return fmt.Sprintf("properties.customRules.rules is invalid: at most %d rules.", azureAfdwMaxRules)
	}
	seen := map[string]bool{}
	for i, raw := range rules {
		if msg := azureAfdwValidateCustomRule(azureARMObject(raw), fmt.Sprintf("properties.customRules.rules[%d].", i), seen); msg != "" {
			return msg
		}
	}
	sets := azureARMSlice(azureARMObject(props["managedRules"])["managedRuleSets"])
	if len(sets) > azureAfdwMaxRules {
		return fmt.Sprintf("properties.managedRules.managedRuleSets is invalid: at most %d rule sets.", azureAfdwMaxRules)
	}
	for i, raw := range sets {
		set := azureARMObject(raw)
		at := fmt.Sprintf("properties.managedRules.managedRuleSets[%d].", i)
		if msg := azureWorkloadsRequired(set, at, "ruleSetType", "ruleSetVersion"); msg != "" {
			return msg
		}
		if !slices.ContainsFunc(azureAfdwManagedRuleSets, func(m struct {
			typ, version string
			groups       []string
		}) bool {
			return m.typ == set["ruleSetType"] && m.version == set["ruleSetVersion"]
		}) {
			return at + "ruleSetType and ruleSetVersion name no managed rule set; list them with FrontDoorWebApplicationFirewallManagedRuleSets."
		}
		if msg := azureWorkloadsEnums(set, at, nil, map[string][]string{"ruleSetAction": {"Block", "Log", "Redirect"}}); msg != "" {
			return msg
		}
	}
	return ""
}

func azureAfdwValidateCustomRule(rule map[string]any, at string, seen map[string]bool) string {
	name := azureARMString(rule["name"])
	if !azureAfdwRuleName.MatchString(name) || seen[strings.ToLower(name)] {
		return at + "name is missing, invalid or duplicated: use letters and digits, starting with a letter."
	}
	seen[strings.ToLower(name)] = true
	if n, ok := rule["priority"].(float64); !ok || n < 0 || n != float64(int(n)) {
		return at + "priority is required: a non-negative integer."
	}
	if msg := azureWorkloadsEnums(rule, at, []string{"ruleType", "action"}, map[string][]string{
		"ruleType": {"MatchRule", "RateLimitRule"}, "action": azureAfdwActions, "enabledState": {"Enabled", "Disabled"},
	}); msg != "" {
		return msg
	}
	if rule["ruleType"] == "RateLimitRule" {
		if n, ok := rule["rateLimitThreshold"].(float64); !ok || n < 0 {
			return at + "rateLimitThreshold is required for a RateLimitRule."
		}
	}
	conditions := azureARMSlice(rule["matchConditions"])
	if len(conditions) == 0 || len(conditions) > azureAfdwMaxRules {
		return fmt.Sprintf("%smatchConditions is missing or invalid: list 1 to %d conditions.", at, azureAfdwMaxRules)
	}
	for j, raw := range conditions {
		cond := azureARMObject(raw)
		cat := fmt.Sprintf("%smatchConditions[%d].", at, j)
		if msg := azureWorkloadsEnums(cond, cat, []string{"matchVariable", "operator"}, map[string][]string{"matchVariable": azureAfdwVariables, "operator": azureAfdwOperators}); msg != "" {
			return msg
		}
		if cond["operator"] != "Any" && len(azureARMSlice(cond["matchValue"])) == 0 {
			return cat + "matchValue is required unless the operator is Any."
		}
	}
	return ""
}

func azureAfdwManagedRuleSetView(_ *Server, scope string) []map[string]any {
	out := make([]map[string]any, 0, len(azureAfdwManagedRuleSets))
	for _, set := range azureAfdwManagedRuleSets {
		groups := make([]any, 0, len(set.groups))
		for _, g := range set.groups {
			groups = append(groups, map[string]any{"ruleGroupName": g, "description": g + " rules", "rules": []any{}})
		}
		name := set.typ + "_" + set.version
		out = append(out, map[string]any{
			"id": scope + "/providers/Microsoft.Network/FrontDoorWebApplicationFirewallManagedRuleSets/" + name, "name": name,
			"type": "Microsoft.Network/FrontDoorWebApplicationFirewallManagedRuleSets",
			"properties": map[string]any{
				"provisioningState": "Succeeded", "ruleSetId": azureWorkloadsHash(name, 32), "ruleSetType": set.typ, "ruleSetVersion": set.version, "ruleGroups": groups,
			},
		})
	}
	return out
}

func azureAfdwValidateRulesEngine(payload map[string]any) string {
	rules := azureARMSlice(azureARMObject(payload["properties"])["rules"])
	if len(rules) > azureAfdwMaxRules {
		return fmt.Sprintf("properties.rules is invalid: at most %d rules.", azureAfdwMaxRules)
	}
	seen := map[string]bool{}
	for i, raw := range rules {
		rule := azureARMObject(raw)
		at := fmt.Sprintf("properties.rules[%d].", i)
		name := strings.ToLower(azureARMString(rule["name"]))
		if name == "" || seen[name] {
			return at + "name is missing or duplicated."
		}
		seen[name] = true
		if n, ok := rule["priority"].(float64); !ok || n < 0 || n != float64(int(n)) {
			return at + "priority is required: a non-negative integer."
		}
		if _, ok := rule["action"].(map[string]any); !ok {
			return at + "action is required."
		}
		if msg := azureWorkloadsEnums(rule, at, nil, map[string][]string{"matchProcessingBehavior": {"Continue", "Stop"}}); msg != "" {
			return msg
		}
	}
	return ""
}

func azureAfdwValidateExperiment(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	for _, key := range []string{"endpointA", "endpointB"} {
		if strings.TrimSpace(azureARMString(azureARMObject(props[key])["endpoint"])) == "" {
			return "properties." + key + ".endpoint is required."
		}
	}
	return azureWorkloadsEnums(props, "properties.", nil, map[string][]string{"enabledState": {"Enabled", "Disabled"}})
}

func azureAfdwExperimentProperties(rec *azureARMResource) map[string]any {
	status := "Enabled"
	if rec.Properties["enabledState"] == "Disabled" {
		status = "Disabled"
	}
	return map[string]any{
		"resourceState": status,
		"status":        "The experiment is " + strings.ToLower(status) + ".",
		"scriptFileUri": "https://stackyardexperiments.blob.core.windows.net/scripts/" + azureWorkloadsHash(rec.ID, 16) + ".js",
	}
}

// handleAzureFrontDoorServiceWebApplicationFirewall20251101Router serves the
// experiment reports, LatencyScorecard and Timeseries, which return one
// object rather than a list; everything else is the shared ARM handler's.
func (s *Server) handleAzureFrontDoorServiceWebApplicationFirewall20251101Router(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, "/azure/subscriptions/") {
		return false
	}
	seg := splitPathSegments(strings.TrimPrefix(path, "/azure/"))
	if len(seg) != 11 || !strings.EqualFold(seg[2], "resourceGroups") || !strings.EqualFold(seg[5], azureAfdcNamespace) ||
		!strings.EqualFold(seg[6], "NetworkExperimentProfiles") || !strings.EqualFold(seg[8], "Experiments") {
		return false
	}
	report := strings.ToLower(seg[10])
	if report != "latencyscorecard" && report != "timeseries" {
		return false
	}
	if r.Method != http.MethodGet {
		azureARMError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
		return true
	}
	if hasAzureInvalidAPIVersion(r) {
		azureARMError(w, http.StatusBadRequest, "InvalidApiVersionParameter", "invalid api-version query parameter")
		return true
	}
	expID := "/subscriptions/" + seg[1] + "/resourceGroups/" + seg[3] + "/providers/" + azureAfdcNamespace + "/NetworkExperimentProfiles/" + seg[7] + "/Experiments/" + seg[9]
	exp, code := s.getAzureARMResource(expID, seg[1], seg[3], strings.TrimSuffix(expID, "/Experiments/"+seg[9]))
	if exp == nil {
		azureARMNotFound(w, code, expID, "")
		return true
	}
	q := r.URL.Query()
	if report == "latencyscorecard" {
		if msg := azureAfdwQueryEnum(q.Get("aggregationInterval"), "aggregationInterval", "Daily", "Weekly", "Monthly"); msg != "" {
			azureARMError(w, http.StatusBadRequest, "InvalidRequestContent", msg)
			return true
		}
		respondJSON(w, http.StatusOK, azureAfdwScorecard(exp, q.Get("aggregationInterval")))
		return true
	}
	if msg := azureAfdwQueryEnum(q.Get("aggregationInterval"), "aggregationInterval", "Hourly", "Daily"); msg != "" {
		azureARMError(w, http.StatusBadRequest, "InvalidRequestContent", msg)
		return true
	}
	if msg := azureAfdwQueryEnum(q.Get("timeseriesType"), "timeseriesType", "MeasurementCounts", "LatencyP50", "LatencyP75", "LatencyP95"); msg != "" {
		azureARMError(w, http.StatusBadRequest, "InvalidRequestContent", msg)
		return true
	}
	start, err1 := time.Parse(time.RFC3339, q.Get("startDateTimeUTC"))
	end, err2 := time.Parse(time.RFC3339, q.Get("endDateTimeUTC"))
	if err1 != nil || err2 != nil || !end.After(start) {
		azureARMError(w, http.StatusBadRequest, "InvalidRequestContent", "startDateTimeUTC and endDateTimeUTC are required RFC 3339 times, with end after start.")
		return true
	}
	respondJSON(w, http.StatusOK, azureAfdwTimeseries(exp, q, start, end))
	return true
}

func azureAfdwQueryEnum(value, name string, allowed ...string) string {
	if !slices.Contains(allowed, value) {
		return name + " is missing or invalid: use " + strings.Join(allowed, ", ") + "."
	}
	return ""
}

func azureAfdwScorecard(exp *azureARMResource, interval string) map[string]any {
	now := time.Now().UTC().Truncate(time.Hour)
	a := azureARMString(azureARMObject(exp.Properties["endpointA"])["endpoint"])
	b := azureARMString(azureARMObject(exp.Properties["endpointB"])["endpoint"])
	return map[string]any{
		"id": exp.ID + "/LatencyScorecard", "name": "LatencyScorecard", "type": "Microsoft.Network/NetworkExperimentProfiles/Experiments/LatencyScorecard",
		"properties": map[string]any{
			"id": azureWorkloadsHash(exp.ID+interval, 32), "name": exp.Name, "description": "Latency scorecard for " + exp.Name,
			"endpointA": a, "endpointB": b, "startDateTimeUTC": now.Add(-24 * time.Hour).Format(time.RFC3339), "endDateTimeUTC": now.Format(time.RFC3339),
			"country": "", "latencyMetrics": []any{
				map[string]any{"name": "LatencyP50", "endDateTimeUTC": now.Format(time.RFC3339), "aValue": 42.0, "bValue": 47.0, "delta": -5.0, "deltaPercent": -10.6, "aCLower95CI": 40.0, "aHUpper95CI": 44.0, "bCLower95CI": 45.0, "bUpper95CI": 49.0},
			},
		},
	}
}

func azureAfdwTimeseries(exp *azureARMResource, q url.Values, start, end time.Time) map[string]any {
	step := time.Hour
	if q.Get("aggregationInterval") == "Daily" {
		step = 24 * time.Hour
	}
	// ponytail: at most 744 points (a month of hours); longer ranges are cut off.
	var points []any
	for t := start; t.Before(end) && len(points) < 744; t = t.Add(step) {
		points = append(points, map[string]any{"dateTimeUTC": t.Format(time.RFC3339), "value": 40.0 + float64(len(points)%10)})
	}
	return map[string]any{
		"id": exp.ID + "/Timeseries", "name": "Timeseries", "type": "Microsoft.Network/NetworkExperimentProfiles/Experiments/Timeseries",
		"properties": map[string]any{
			"endpoint": q.Get("endpoint"), "startDateTimeUTC": start.Format(time.RFC3339), "endDateTimeUTC": end.Format(time.RFC3339),
			"aggregationInterval": q.Get("aggregationInterval"), "timeseriesType": q.Get("timeseriesType"), "country": q.Get("country"),
			"timeseriesData": points,
		},
	}
}
