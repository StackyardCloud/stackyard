package server

import (
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
)

// Profile-level Azure Front Door operations. Profiles belong to the CDN
// router; it calls these for the Front Door actions it doesn't serve, and
// mirrors every profile into the ARM store so Front Door child resources
// have a parent there.

var azureAfdxKeyVaultSecret = regexp.MustCompile(`(?i)^/subscriptions/[^/]+/resourceGroups/[^/]+/providers/Microsoft\.KeyVault/vaults/[^/]+/secrets/[^/]+$`)

// azureAfdxProfileLimits are the per-profile resource limits usages report.
var azureAfdxProfileLimits = [][2]any{
	{"afdEndpoints", 25.0}, {"originGroups", 200.0}, {"customDomains", 500.0},
	{"ruleSets", 200.0}, {"secrets", 500.0}, {"securityPolicies", 200.0},
}

// azureAfdxMirrorProfileLocked stores the profile as an ARM record. Its type
// is not registered, so the CDN router keeps serving it. Caller holds
// providerStorageMu.
func (s *Server) azureAfdxMirrorProfileLocked(profile *azureCDNProfile) {
	id := azureCDNProfileID(profile.SubscriptionID, profile.ResourceGroup, profile.Name)
	tags := make(map[string]any, len(profile.Tags))
	for k, v := range profile.Tags {
		tags[k] = v
	}
	s.azureARMResources[azureARMKey(id)] = &azureARMResource{
		ID: id, SubscriptionID: profile.SubscriptionID, ResourceGroup: profile.ResourceGroup, Name: profile.Name,
		Type: azureAfdxNamespace + "/profiles", Location: profile.Location, Tags: tags,
		SKU: azureARMMerge(azureCDNProfileSKU(profile), nil), Properties: azureARMMerge(profile.Properties, nil),
		CreatedAt: profile.CreatedAt, UpdatedAt: profile.UpdatedAt,
	}
}

// handleAzureAfdxProfileAction serves {profile}/{action}, reporting whether
// action is a Front Door profile action.
func (s *Server) handleAzureAfdxProfileAction(w http.ResponseWriter, r *http.Request, sub, rg, name string, tail []string) bool {
	if len(tail) != 1 {
		return false
	}
	action := strings.ToLower(tail[0])
	post := map[string]func(*azureCDNProfile, map[string]any) (int, any){
		"checkendpointnameavailability": func(_ *azureCDNProfile, payload map[string]any) (int, any) {
			return s.azureAfdxCheckEndpointName(sub, rg, payload)
		},
		"checkhostnameavailability": s.azureAfdxCheckHostName,
		"usages":                    s.azureAfdxProfileUsages,
		"upgrade":                   s.azureAfdxUpgrade,
		"validatesecret":            azureAfdxValidateSecretRef,
	}
	get := map[string]func(*azureCDNProfile, url.Values) (int, any){
		"getloganalyticslocations":   azureAfdxLogLocations,
		"getloganalyticsmetrics":     azureAfdxLogMetrics,
		"getloganalyticsrankings":    azureAfdxLogRankings,
		"getloganalyticsresources":   s.azureAfdxLogResources,
		"getwafloganalyticsmetrics":  azureAfdxWafMetrics,
		"getwafloganalyticsrankings": azureAfdxWafRankings,
	}
	run, isPost := post[action]
	report, isGet := get[action]
	if !isPost && !isGet {
		return false
	}
	profileID := azureCDNProfileID(sub, rg, name)
	s.providerStorageMu.Lock()
	profile := s.azureCDNProfiles[profileID]
	s.providerStorageMu.Unlock()
	switch {
	case profile == nil:
		azureARMNotFound(w, "ResourceNotFound", profileID, "")
	case isPost:
		s.serveAzureARMPost(w, r, func(payload map[string]any) (int, any) { return run(profile, payload) })
	case r.Method != http.MethodGet:
		azureARMError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
	case hasAzureInvalidAPIVersion(r):
		azureARMError(w, http.StatusBadRequest, "InvalidApiVersionParameter", "invalid api-version query parameter")
	default:
		status, body := report(profile, r.URL.Query())
		respondAzureARMAction(w, status, body)
	}
	return true
}

func (s *Server) azureAfdxCheckEndpointName(sub, rg string, payload map[string]any) (int, any) {
	name := strings.TrimSpace(azureARMString(payload["name"]))
	if name == "" || !strings.EqualFold(azureARMString(payload["type"]), "Microsoft.Cdn/Profiles/AfdEndpoints") {
		return azureOracleBad("name and type 'Microsoft.Cdn/Profiles/AfdEndpoints' are required.")
	}
	host := strings.ToLower(name) + "-" + azureWorkloadsHash(sub+"/"+rg+"/"+name, 16) + ".z01.azurefd.net"
	if len(name) > 46 || !azureAfdxEndpointName.MatchString(name) {
		return http.StatusOK, map[string]any{"nameAvailable": false, "reason": "Invalid", "message": "The endpoint name is invalid.", "availableHostname": ""}
	}
	if len(s.azureARMResourcesWhere(func(rec *azureARMResource) bool {
		return strings.EqualFold(rec.Type, "Microsoft.Cdn/profiles/afdEndpoints") && strings.EqualFold(rec.Name, name) &&
			strings.EqualFold(rec.SubscriptionID, sub) && strings.EqualFold(rec.ResourceGroup, rg)
	})) > 0 {
		return http.StatusOK, map[string]any{"nameAvailable": false, "reason": "AlreadyExists", "message": "An endpoint named " + name + " already exists in the resource group.", "availableHostname": ""}
	}
	return http.StatusOK, map[string]any{"nameAvailable": true, "reason": "", "message": "", "availableHostname": host}
}

func (s *Server) azureAfdxCheckHostName(_ *azureCDNProfile, payload map[string]any) (int, any) {
	host := strings.TrimSpace(azureARMString(payload["hostName"]))
	if host == "" {
		return azureOracleBad("hostName is required.")
	}
	if len(s.azureARMResourcesWhere(func(rec *azureARMResource) bool {
		return strings.EqualFold(rec.Type, "Microsoft.Cdn/profiles/customDomains") && strings.EqualFold(azureARMString(rec.Properties["hostName"]), host)
	})) > 0 {
		return http.StatusOK, map[string]any{"nameAvailable": false, "reason": "AlreadyExists", "message": host + " is already a custom domain."}
	}
	return http.StatusOK, map[string]any{"nameAvailable": true, "reason": "", "message": ""}
}

func (s *Server) azureAfdxProfileUsages(profile *azureCDNProfile, _ map[string]any) (int, any) {
	id := azureCDNProfileID(profile.SubscriptionID, profile.ResourceGroup, profile.Name)
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	out := make([]any, 0, len(azureAfdxProfileLimits))
	for _, limit := range azureAfdxProfileLimits {
		kind := limit[0].(string)
		count := len(s.azureARMResourcesMatchingLocked(func(rec *azureARMResource) bool {
			return strings.EqualFold(rec.Type, azureAfdxNamespace+"/profiles/"+kind) && strings.HasPrefix(azureARMKey(rec.ID), azureARMKey(id+"/"))
		}))
		out = append(out, azureAfdxUsage(id, kind, count, limit[1].(float64)))
	}
	return http.StatusOK, map[string]any{"value": out}
}

// azureAfdxUpgrade moves a Standard profile to Premium, taking the WAF
// policy mappings the client sends.
func (s *Server) azureAfdxUpgrade(profile *azureCDNProfile, payload map[string]any) (int, any) {
	mappings := azureARMSlice(payload["wafMappingList"])
	if _, ok := payload["wafMappingList"].([]any); !ok || len(mappings) > azureAfdxMaxItems {
		return azureOracleBad("wafMappingList is required: list the WAF policies to upgrade (it may be empty).")
	}
	for i, raw := range mappings {
		m := azureARMObject(raw)
		if azureARMString(m["senderPolicyId"]) == "" || azureARMString(azureARMObject(m["upgradedPolicy"])["id"]) == "" {
			return azureOracleBad("wafMappingList[" + strconv.Itoa(i) + "] needs senderPolicyId and upgradedPolicy.id.")
		}
	}
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	stored := s.azureCDNProfiles[azureCDNProfileID(profile.SubscriptionID, profile.ResourceGroup, profile.Name)]
	if stored == nil {
		return http.StatusNotFound, azureARMErrorBody("ResourceNotFound", "The profile was not found.")
	}
	if azureARMString(stored.SKU["name"]) != "Standard_AzureFrontDoor" {
		return azureOracleBad("Only a Standard_AzureFrontDoor profile can be upgraded.")
	}
	stored.SKU = map[string]any{"name": "Premium_AzureFrontDoor"}
	stored.UpdatedAt = time.Now().UTC()
	s.azureAfdxMirrorProfileLocked(stored)
	return http.StatusOK, azureCDNProfileFixture(stored)
}

// azureAfdxValidateSecretRef checks the shape of a Key Vault secret
// reference; the vault is not read.
func azureAfdxValidateSecretRef(_ *azureCDNProfile, payload map[string]any) (int, any) {
	if msg := azureWorkloadsEnums(payload, "", []string{"secretType"}, map[string][]string{"secretType": azureAfdxSecretTypes}); msg != "" {
		return azureOracleBad(msg)
	}
	id := azureARMString(azureARMObject(payload["secretSource"])["id"])
	if id == "" {
		return azureOracleBad("secretSource.id is required.")
	}
	if !azureAfdxKeyVaultSecret.MatchString(id) {
		return http.StatusOK, map[string]any{"status": "Invalid", "message": "secretSource.id must be a Key Vault secret resource ID."}
	}
	return http.StatusOK, map[string]any{"status": "Valid", "message": ""}
}

// azureAfdxReportWindow checks a report's dateTimeBegin and dateTimeEnd and
// the other required query parameters.
func azureAfdxReportWindow(q url.Values, required ...string) (map[string]any, string) {
	begin, err1 := time.Parse(time.RFC3339, q.Get("dateTimeBegin"))
	end, err2 := time.Parse(time.RFC3339, q.Get("dateTimeEnd"))
	if err1 != nil || err2 != nil || !end.After(begin) {
		return nil, "dateTimeBegin and dateTimeEnd are required RFC 3339 times, with the end after the begin."
	}
	for _, key := range required {
		if len(q[key]) == 0 || q.Get(key) == "" {
			return nil, key + " is required."
		}
	}
	return map[string]any{"dateTimeBegin": begin.Format(time.RFC3339), "dateTimeEnd": end.Format(time.RFC3339)}, ""
}

// azureAfdxSeries reports a zero series per requested metric: no traffic
// flows through the emulator.
func azureAfdxSeries(q url.Values, out map[string]any) map[string]any {
	granularity := q.Get("granularity")
	if !slices.Contains([]string{"PT5M", "PT1H", "P1D"}, granularity) {
		return map[string]any{"error": "granularity is missing or invalid: use PT5M, PT1H or P1D."}
	}
	series := []any{}
	for _, metric := range q["metrics"] {
		series = append(series, map[string]any{"metric": metric, "unit": "count", "groups": []any{}, "data": []any{map[string]any{"dateTime": out["dateTimeBegin"], "value": 0}}})
	}
	return azureARMMerge(out, map[string]any{"granularity": granularity, "series": series})
}

func azureAfdxLogLocations(*azureCDNProfile, url.Values) (int, any) {
	return http.StatusOK, map[string]any{
		"continents":       []any{map[string]any{"id": "NA"}, map[string]any{"id": "EU"}, map[string]any{"id": "AS"}},
		"countryOrRegions": []any{map[string]any{"id": "US", "continentId": "NA"}, map[string]any{"id": "DE", "continentId": "EU"}, map[string]any{"id": "JP", "continentId": "AS"}},
	}
}

func azureAfdxLogMetrics(_ *azureCDNProfile, q url.Values) (int, any) {
	out, msg := azureAfdxReportWindow(q, "metrics")
	if msg != "" {
		return azureOracleBad(msg)
	}
	body := azureAfdxSeries(q, out)
	if body["error"] != nil {
		return azureOracleBad(azureARMString(body["error"]))
	}
	return http.StatusOK, body
}

func azureAfdxWafMetrics(p *azureCDNProfile, q url.Values) (int, any) {
	return azureAfdxLogMetrics(p, q)
}

func azureAfdxLogRankings(_ *azureCDNProfile, q url.Values) (int, any) {
	out, msg := azureAfdxReportWindow(q, "rankings", "metrics", "maxRanking")
	if msg != "" {
		return azureOracleBad(msg)
	}
	tables := []any{}
	for _, ranking := range q["rankings"] {
		tables = append(tables, map[string]any{"ranking": ranking, "data": []any{}})
	}
	return http.StatusOK, azureARMMerge(out, map[string]any{"tables": tables})
}

func azureAfdxWafRankings(_ *azureCDNProfile, q url.Values) (int, any) {
	out, msg := azureAfdxReportWindow(q, "rankings", "metrics", "maxRanking")
	if msg != "" {
		return azureOracleBad(msg)
	}
	return http.StatusOK, azureARMMerge(out, map[string]any{"groups": q["rankings"], "data": []any{}})
}

// azureAfdxLogResources lists the profile's endpoints and custom domains,
// the resources its reports can filter on.
func (s *Server) azureAfdxLogResources(profile *azureCDNProfile, _ url.Values) (int, any) {
	id := azureCDNProfileID(profile.SubscriptionID, profile.ResourceGroup, profile.Name)
	endpoints, domains := []any{}, []any{}
	for _, rec := range s.azureARMResourcesWhere(func(rec *azureARMResource) bool {
		return strings.HasPrefix(azureARMKey(rec.ID), azureARMKey(id+"/"))
	}) {
		switch {
		case strings.EqualFold(rec.Type, "Microsoft.Cdn/profiles/afdEndpoints"):
			endpoints = append(endpoints, map[string]any{"id": rec.ID, "name": azureARMString(rec.Properties["hostName"]), "history": []any{}, "customDomains": []any{}})
		case strings.EqualFold(rec.Type, "Microsoft.Cdn/profiles/customDomains"):
			domains = append(domains, map[string]any{"id": rec.ID, "name": azureARMString(rec.Properties["hostName"]), "endpointId": ""})
		}
	}
	return http.StatusOK, map[string]any{"endpoints": endpoints, "customDomains": domains}
}
