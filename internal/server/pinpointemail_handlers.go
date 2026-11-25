package server

import (
	"net/http"
	"sort"
	"strings"
	"time"
)

var ppeHandlers map[string]ppeHandler

func init() {
	ppeHandlers = map[string]ppeHandler{
		"CreateConfigurationSet":               (*Server).ppeCreateConfigurationSet,
		"GetConfigurationSet":                  (*Server).ppeGetConfigurationSet,
		"DeleteConfigurationSet":               (*Server).ppeDeleteConfigurationSet,
		"ListConfigurationSets":                (*Server).ppeListConfigurationSets,
		"PutConfigurationSetDeliveryOptions":   (*Server).ppePutDeliveryOptions,
		"PutConfigurationSetReputationOptions": (*Server).ppePutReputationOptions,
		"PutConfigurationSetSendingOptions":    (*Server).ppePutSendingOptions,
		"PutConfigurationSetTrackingOptions":   (*Server).ppePutTrackingOptions,
		"CreateConfigurationSetEventDestination": func(s *Server, in map[string]any) (map[string]any, *ppeErr) {
			return s.ppePutEventDestination(in, true)
		},
		"UpdateConfigurationSetEventDestination": func(s *Server, in map[string]any) (map[string]any, *ppeErr) {
			return s.ppePutEventDestination(in, false)
		},
		"DeleteConfigurationSetEventDestination": (*Server).ppeDeleteEventDestination,
		"GetConfigurationSetEventDestinations":   (*Server).ppeGetEventDestinations,
		"CreateDedicatedIpPool":                  (*Server).ppeCreatePool,
		"DeleteDedicatedIpPool": func(s *Server, in map[string]any) (map[string]any, *ppeErr) {
			name := mmgrStr(in, "PoolName")
			if err := s.pinpointEmail.deletePool(name); err != nil {
				return nil, err
			}
			s.ppeClearTags(ppeARNBase + "dedicated-ip-pool/" + name)
			return nil, nil
		},
		"ListDedicatedIpPools": func(s *Server, in map[string]any) (map[string]any, *ppeErr) { return s.pinpointEmail.listPools(in) },
		"GetDedicatedIp": func(s *Server, in map[string]any) (map[string]any, *ppeErr) {
			return s.pinpointEmail.getIP(mmgrStr(in, "Ip"))
		},
		"GetDedicatedIps": func(s *Server, in map[string]any) (map[string]any, *ppeErr) { return s.pinpointEmail.listIPs(in) },
		"PutDedicatedIpInPool": func(s *Server, in map[string]any) (map[string]any, *ppeErr) {
			return nil, s.pinpointEmail.putIP(mmgrStr(in, "Ip"), mmgrStr(in, "DestinationPoolName"), -1)
		},
		"PutDedicatedIpWarmupAttributes": func(s *Server, in map[string]any) (map[string]any, *ppeErr) {
			pct := mmgrInt(in, "WarmupPercentage", -1)
			if pct < 0 || pct > 100 {
				return nil, ppeBadRequest("WarmupPercentage must be between 0 and 100.")
			}
			return nil, s.pinpointEmail.putIP(mmgrStr(in, "Ip"), "", pct)
		},
		"CreateEmailIdentity":                (*Server).ppeCreateEmailIdentity,
		"GetEmailIdentity":                   (*Server).ppeGetEmailIdentity,
		"DeleteEmailIdentity":                (*Server).ppeDeleteEmailIdentity,
		"ListEmailIdentities":                (*Server).ppeListEmailIdentities,
		"PutEmailIdentityDkimAttributes":     (*Server).ppePutDkim,
		"PutEmailIdentityFeedbackAttributes": (*Server).ppePutFeedback,
		"PutEmailIdentityMailFromAttributes": (*Server).ppePutMailFrom,
		"GetAccount":                         (*Server).ppeGetAccount,
		"PutAccountSendingAttributes": func(s *Server, in map[string]any) (map[string]any, *ppeErr) {
			enabled, _ := in["SendingEnabled"].(bool)
			s.sesv2.PutAccountSendingAttributes(enabled)
			return nil, nil
		},
		"PutAccountDedicatedIpWarmupAttributes": func(s *Server, in map[string]any) (map[string]any, *ppeErr) {
			st := s.pinpointEmail
			st.mu.Lock()
			st.autoWarmup, _ = in["AutoWarmupEnabled"].(bool)
			st.mu.Unlock()
			return nil, nil
		},
		"PutDeliverabilityDashboardOption":  (*Server).ppePutDashboard,
		"GetDeliverabilityDashboardOptions": (*Server).ppeGetDashboard,
		"GetBlacklistReports":               (*Server).ppeBlacklistReports,
		"GetDomainStatisticsReport":         (*Server).ppeDomainStatistics,
		"ListDomainDeliverabilityCampaigns": (*Server).ppeListCampaigns,
		"GetDomainDeliverabilityCampaign": func(s *Server, in map[string]any) (map[string]any, *ppeErr) {
			id := mmgrStr(in, "CampaignId")
			for _, c := range s.pinpointEmail.campaigns("", time.Time{}, time.Time{}) {
				if c["CampaignId"] == id {
					return map[string]any{"DomainDeliverabilityCampaign": c}, nil
				}
			}
			return nil, ppeNotFound("Campaign %s doesn't exist.", id)
		},
		"CreateDeliverabilityTestReport": (*Server).ppeCreateTestReport,
		"GetDeliverabilityTestReport":    (*Server).ppeGetTestReport,
		"ListDeliverabilityTestReports": func(s *Server, in map[string]any) (map[string]any, *ppeErr) {
			return s.pinpointEmail.listReports(in)
		},
		"SendEmail":           (*Server).ppeSendEmail,
		"TagResource":         (*Server).ppeTagResource,
		"UntagResource":       (*Server).ppeUntagResource,
		"ListTagsForResource": (*Server).ppeListTags,
	}
}

// ---- configuration sets ----

func (s *Server) ppeCreateConfigurationSet(in map[string]any) (map[string]any, *ppeErr) {
	name := mmgrStr(in, "ConfigurationSetName")
	tags, perr := ppeTagList(in)
	if perr != nil {
		return nil, perr
	}
	if pool := mmgrStr(mmgrMap(in, "DeliveryOptions"), "SendingPoolName"); pool != "" && !s.pinpointEmail.poolExists(pool) {
		return nil, ppeNotFound("Dedicated IP pool %s doesn't exist.", pool)
	}
	if len(s.sesv2.ListConfigurationSets()) >= ppeMaxConfigSets {
		return nil, ppeLimit("Stackyard keeps at most %d configuration sets.", ppeMaxConfigSets)
	}
	if err := ppeFromSESv2(s.sesv2.CreateConfigurationSet(name), "Configuration set "+name); err != nil {
		return nil, err
	}
	opts := ppeDefaultOpts()
	for _, k := range []string{"TrackingOptions", "DeliveryOptions", "ReputationOptions", "SendingOptions"} {
		if v, ok := in[k].(map[string]any); ok {
			opts[k] = v
		}
	}
	if mmgrMap(opts, "ReputationOptions")["ReputationMetricsEnabled"] == true {
		opts["ReputationOptions"] = map[string]any{"ReputationMetricsEnabled": true, "LastFreshStart": mmgrEpoch(time.Now().UTC())}
	}
	st := s.pinpointEmail
	st.mu.Lock()
	st.opts[name] = opts
	st.mu.Unlock()
	if len(tags) > 0 {
		_ = s.sesv2.TagResource(ppeARNBase+"configuration-set/"+name, tags)
	}
	return nil, nil
}

func (s *Server) ppeConfigSetExists(name string) *ppeErr {
	if _, ok := s.sesv2.GetConfigurationSet(name); !ok {
		return ppeNotFound("Configuration set %s doesn't exist.", name)
	}
	return nil
}

func (s *Server) ppeGetConfigurationSet(in map[string]any) (map[string]any, *ppeErr) {
	name := mmgrStr(in, "ConfigurationSetName")
	if err := s.ppeConfigSetExists(name); err != nil {
		return nil, err
	}
	tags := s.sesv2.ListTags(ppeARNBase + "configuration-set/" + name)
	st := s.pinpointEmail
	st.mu.Lock()
	out := st.optsLocked(name)
	st.mu.Unlock()
	out["ConfigurationSetName"] = name
	out["Tags"] = ppeTagsOut(tags)
	return out, nil
}

func (s *Server) ppeDeleteConfigurationSet(in map[string]any) (map[string]any, *ppeErr) {
	name := mmgrStr(in, "ConfigurationSetName")
	if !s.sesv2.DeleteConfigurationSet(name) {
		return nil, ppeNotFound("Configuration set %s doesn't exist.", name)
	}
	s.ppeClearTags(ppeARNBase + "configuration-set/" + name)
	st := s.pinpointEmail
	st.mu.Lock()
	delete(st.opts, name)
	st.mu.Unlock()
	return nil, nil
}

func (s *Server) ppeListConfigurationSets(in map[string]any) (map[string]any, *ppeErr) {
	var names []string
	for _, cfg := range s.sesv2.ListConfigurationSets() {
		names = append(names, cfg.Name)
	}
	return ppePage(names, in, "ConfigurationSets")
}

func (s *Server) ppePutDeliveryOptions(in map[string]any) (map[string]any, *ppeErr) {
	name := mmgrStr(in, "ConfigurationSetName")
	if err := s.ppeConfigSetExists(name); err != nil {
		return nil, err
	}
	opt := mmgrPick(in, "TlsPolicy", "SendingPoolName")
	if pool := mmgrStr(opt, "SendingPoolName"); pool != "" && !s.pinpointEmail.poolExists(pool) {
		return nil, ppeNotFound("Dedicated IP pool %s doesn't exist.", pool)
	}
	s.pinpointEmail.setOpt(name, "DeliveryOptions", opt)
	return nil, nil
}

func (s *Server) ppePutReputationOptions(in map[string]any) (map[string]any, *ppeErr) {
	name := mmgrStr(in, "ConfigurationSetName")
	if err := s.ppeConfigSetExists(name); err != nil {
		return nil, err
	}
	enabled, _ := in["ReputationMetricsEnabled"].(bool)
	st := s.pinpointEmail
	st.mu.Lock()
	prev := mmgrMap(st.optsLocked(name), "ReputationOptions")
	st.mu.Unlock()
	opt := map[string]any{"ReputationMetricsEnabled": enabled}
	if enabled {
		opt["LastFreshStart"] = prev["LastFreshStart"]
		if prev["ReputationMetricsEnabled"] != true {
			opt["LastFreshStart"] = mmgrEpoch(time.Now().UTC())
		}
	}
	st.setOpt(name, "ReputationOptions", opt)
	return nil, nil
}

func (s *Server) ppePutSendingOptions(in map[string]any) (map[string]any, *ppeErr) {
	name := mmgrStr(in, "ConfigurationSetName")
	if err := s.ppeConfigSetExists(name); err != nil {
		return nil, err
	}
	enabled, _ := in["SendingEnabled"].(bool)
	s.pinpointEmail.setOpt(name, "SendingOptions", map[string]any{"SendingEnabled": enabled})
	return nil, nil
}

func (s *Server) ppePutTrackingOptions(in map[string]any) (map[string]any, *ppeErr) {
	name := mmgrStr(in, "ConfigurationSetName")
	if err := s.ppeConfigSetExists(name); err != nil {
		return nil, err
	}
	var opt any
	if d := mmgrStr(in, "CustomRedirectDomain"); d != "" {
		opt = map[string]any{"CustomRedirectDomain": d}
	}
	s.pinpointEmail.setOpt(name, "TrackingOptions", opt)
	return nil, nil
}

// ---- event destinations (stored in SESv2's configuration sets) ----

func (s *Server) ppePutEventDestination(in map[string]any, create bool) (map[string]any, *ppeErr) {
	cfg, name := mmgrStr(in, "ConfigurationSetName"), mmgrStr(in, "EventDestinationName")
	def := mmgrMap(in, "EventDestination")
	kinds := 0
	for _, k := range []string{"KinesisFirehoseDestination", "CloudWatchDestination", "SnsDestination", "PinpointDestination"} {
		if def[k] != nil {
			kinds++
		}
	}
	if kinds != 1 {
		return nil, ppeBadRequest("An event destination sets exactly one of KinesisFirehoseDestination, CloudWatchDestination, SnsDestination and PinpointDestination.")
	}
	if len(mmgrList(def, "MatchingEventTypes")) == 0 {
		return nil, ppeBadRequest("MatchingEventTypes must list at least one event type.")
	}
	existing, ok := s.sesv2.GetConfigurationSetEventDestinations(cfg)
	if !ok {
		return nil, ppeNotFound("Configuration set %s doesn't exist.", cfg)
	}
	_, exists := existing[name]
	switch {
	case create && exists:
		return nil, ppeError(http.StatusBadRequest, "AlreadyExistsException", "Event destination %s already exists.", name)
	case !create && !exists:
		return nil, ppeNotFound("Event destination %s doesn't exist.", name)
	case create && len(existing) >= ppeMaxDests:
		return nil, ppeLimit("A configuration set can have at most %d event destinations.", ppeMaxDests)
	}
	return nil, ppeFromSESv2(s.sesv2.UpsertConfigurationSetEventDestination(cfg, name, map[string]any{"EventDestination": def}), "Configuration set "+cfg)
}

func (s *Server) ppeDeleteEventDestination(in map[string]any) (map[string]any, *ppeErr) {
	cfg, name := mmgrStr(in, "ConfigurationSetName"), mmgrStr(in, "EventDestinationName")
	if !s.sesv2.DeleteConfigurationSetEventDestination(cfg, name) {
		return nil, ppeNotFound("Event destination %s in configuration set %s doesn't exist.", name, cfg)
	}
	return nil, nil
}

func (s *Server) ppeGetEventDestinations(in map[string]any) (map[string]any, *ppeErr) {
	cfg := mmgrStr(in, "ConfigurationSetName")
	stored, ok := s.sesv2.GetConfigurationSetEventDestinations(cfg)
	if !ok {
		return nil, ppeNotFound("Configuration set %s doesn't exist.", cfg)
	}
	names := make([]string, 0, len(stored))
	for n := range stored {
		names = append(names, n)
	}
	sort.Strings(names)
	model := s.pinpointEmail.model
	out := make([]any, 0, len(names))
	for _, n := range names {
		def := stored[n]
		if wrapped := mmgrMap(def, "EventDestination"); wrapped != nil {
			def = wrapped
		}
		entry, _ := model.Conform("EventDestination", def).(map[string]any)
		if entry == nil {
			entry = map[string]any{}
		}
		entry["Name"] = n
		if entry["MatchingEventTypes"] == nil {
			entry["MatchingEventTypes"] = []any{}
		}
		if entry["Enabled"] == nil {
			entry["Enabled"] = false
		}
		out = append(out, entry)
	}
	return map[string]any{"EventDestinations": out}, nil
}

// ---- dedicated IP pools ----

func (s *Server) ppeCreatePool(in map[string]any) (map[string]any, *ppeErr) {
	name := mmgrStr(in, "PoolName")
	tags, perr := ppeTagList(in)
	if perr != nil {
		return nil, perr
	}
	if err := s.pinpointEmail.createPool(name); err != nil {
		return nil, err
	}
	if len(tags) > 0 {
		_ = s.sesv2.TagResource(ppeARNBase+"dedicated-ip-pool/"+name, tags)
	}
	return nil, nil
}

// ---- identities (SESv2's) ----

func ppeValidIdentity(id string) bool {
	if strings.ContainsAny(id, " \t\r\n/<>,;") || strings.HasPrefix(id, ".") || strings.HasSuffix(id, ".") {
		return false
	}
	if local, domain, ok := strings.Cut(id, "@"); ok {
		return local != "" && strings.Contains(domain, ".")
	}
	return strings.Contains(id, ".")
}

func (s *Server) ppeCreateEmailIdentity(in map[string]any) (map[string]any, *ppeErr) {
	id := strings.ToLower(strings.TrimSpace(mmgrStr(in, "EmailIdentity")))
	if !ppeValidIdentity(id) {
		return nil, ppeBadRequest("%s isn't an email address or domain.", id)
	}
	tags, perr := ppeTagList(in)
	if perr != nil {
		return nil, perr
	}
	if _, exists := s.sesv2.GetEmailIdentity(id); !exists && len(s.sesv2.ListEmailIdentities()) >= ppeMaxIdentities {
		return nil, ppeLimit("Stackyard keeps at most %d identities.", ppeMaxIdentities)
	}
	ident, err := s.sesv2.CreateEmailIdentity(id, "", nil)
	if err != nil {
		return nil, ppeFromSESv2(err, "Identity "+id)
	}
	if len(tags) > 0 {
		_ = s.sesv2.TagResource(ppeARNBase+"identity/"+ident.EmailIdentity, tags)
	}
	return map[string]any{
		"IdentityType": ident.IdentityType, "VerifiedForSendingStatus": ident.VerifiedForSendingStatus,
		"DkimAttributes": map[string]any{"SigningEnabled": ident.DkimSigningEnabled, "Status": ident.DkimStatus, "Tokens": mmgrAnyStrings(ident.DkimTokens)},
	}, nil
}

func (s *Server) ppeGetEmailIdentity(in map[string]any) (map[string]any, *ppeErr) {
	id := mmgrStr(in, "EmailIdentity")
	ident, ok := s.sesv2.GetEmailIdentity(id)
	if !ok {
		return nil, ppeNotFound("Identity %s doesn't exist.", id)
	}
	tags := map[string]string{}
	for k, v := range ident.Tags {
		tags[k] = v
	}
	for k, v := range s.sesv2.ListTags(ppeARNBase + "identity/" + ident.EmailIdentity) {
		tags[k] = v
	}
	return map[string]any{
		"IdentityType": ident.IdentityType, "FeedbackForwardingStatus": ident.FeedbackForwardingStatus,
		"VerifiedForSendingStatus": ident.VerifiedForSendingStatus,
		"DkimAttributes":           map[string]any{"SigningEnabled": ident.DkimSigningEnabled, "Status": ident.DkimStatus, "Tokens": mmgrAnyStrings(ident.DkimTokens)},
		"MailFromAttributes": map[string]any{
			"MailFromDomain": ident.MailFromDomain, "MailFromDomainStatus": ident.MailFromDomainStatus,
			"BehaviorOnMxFailure": ident.BehaviorOnMxFailure,
		},
		"Tags": ppeTagsOut(tags),
	}, nil
}

func (s *Server) ppeDeleteEmailIdentity(in map[string]any) (map[string]any, *ppeErr) {
	id := strings.ToLower(mmgrStr(in, "EmailIdentity"))
	if !s.sesv2.DeleteEmailIdentity(id) {
		return nil, ppeNotFound("Identity %s doesn't exist.", id)
	}
	s.ppeClearTags(ppeARNBase + "identity/" + id)
	return nil, nil
}

func (s *Server) ppeListEmailIdentities(in map[string]any) (map[string]any, *ppeErr) {
	idents := s.sesv2.ListEmailIdentities()
	start, end, next, ok := mmgrPage(len(idents), mmgrStr(in, "NextToken"), mmgrInt(in, "PageSize", 0), ppeDefaultPage)
	if !ok {
		return nil, ppeBadRequest("NextToken is not valid.")
	}
	items := make([]any, 0, end-start)
	for _, id := range idents[start:end] {
		items = append(items, map[string]any{"IdentityType": id.IdentityType, "IdentityName": id.EmailIdentity, "SendingEnabled": id.VerifiedForSendingStatus})
	}
	out := map[string]any{"EmailIdentities": items}
	if next != "" {
		out["NextToken"] = next
	}
	return out, nil
}

func (s *Server) ppePutDkim(in map[string]any) (map[string]any, *ppeErr) {
	enabled, _ := in["SigningEnabled"].(bool)
	id := mmgrStr(in, "EmailIdentity")
	return nil, ppeFromSESv2(s.sesv2.PutEmailIdentityDkim(id, enabled, nil), "Identity "+id)
}

func (s *Server) ppePutFeedback(in map[string]any) (map[string]any, *ppeErr) {
	enabled, _ := in["EmailForwardingEnabled"].(bool)
	id := mmgrStr(in, "EmailIdentity")
	return nil, ppeFromSESv2(s.sesv2.PutEmailIdentityFeedback(id, enabled), "Identity "+id)
}

func (s *Server) ppePutMailFrom(in map[string]any) (map[string]any, *ppeErr) {
	id := mmgrStr(in, "EmailIdentity")
	if _, ok := s.sesv2.GetEmailIdentity(id); !ok {
		return nil, ppeNotFound("Identity %s doesn't exist.", id)
	}
	domain := strings.ToLower(mmgrStr(in, "MailFromDomain"))
	if domain != "" && !strings.HasSuffix(domain, "."+ppeDomainOf(id)) {
		return nil, ppeBadRequest("MailFromDomain must be a subdomain of %s.", ppeDomainOf(id))
	}
	behavior := mmgrStr(in, "BehaviorOnMxFailure")
	if behavior == "" {
		behavior = "USE_DEFAULT_VALUE"
	}
	return nil, ppeFromSESv2(s.sesv2.PutEmailIdentityMailFrom(id, domain, behavior), "Identity "+id)
}

// ppeVerified reports whether address (or its domain) is a verified identity,
// and whether that identity signs with DKIM.
func (s *Server) ppeVerified(address string) (bool, bool) {
	for _, id := range []string{address, ppeDomainOf(address)} {
		if ident, ok := s.sesv2.GetEmailIdentity(id); ok && ident.VerifiedForSendingStatus {
			return true, ident.DkimSigningEnabled
		}
	}
	return false, false
}

// ---- account ----

func (s *Server) ppeGetAccount(map[string]any) (map[string]any, *ppeErr) {
	acct := s.sesv2.GetAccount()
	st := s.pinpointEmail
	st.mu.Lock()
	warmup := st.autoWarmup
	st.mu.Unlock()
	return map[string]any{
		"SendingEnabled": acct.SendingEnabled, "ProductionAccessEnabled": acct.ProductionAccessEnabled,
		"DedicatedIpAutoWarmupEnabled": warmup, "EnforcementStatus": "HEALTHY",
		"SendQuota": map[string]any{
			"Max24HourSend": acct.SendQuota.Max24HourSend, "MaxSendRate": acct.SendQuota.MaxSendRate,
			"SentLast24Hours": acct.SendQuota.SentLast24Hours,
		},
	}, nil
}

// ---- deliverability dashboard ----

func (s *Server) ppePutDashboard(in map[string]any) (map[string]any, *ppeErr) {
	enabled, _ := in["DashboardEnabled"].(bool)
	domains := mmgrList(in, "SubscribedDomains")
	if len(domains) > ppeMaxDomains {
		return nil, ppeLimit("The dashboard tracks at most %d domains.", ppeMaxDomains)
	}
	now := time.Now().UTC()
	next := make([]any, 0, len(domains))
	for _, item := range domains {
		d, _ := item.(map[string]any)
		domain := strings.ToLower(mmgrStr(d, "Domain"))
		if ident, ok := s.sesv2.GetEmailIdentity(domain); !ok || ident.IdentityType != "DOMAIN" {
			return nil, ppeNotFound("Domain %s isn't a verified identity.", domain)
		}
		entry := map[string]any{"Domain": domain, "InboxPlacementTrackingOption": d["InboxPlacementTrackingOption"], "SubscriptionStartDate": mmgrEpoch(now)}
		if d["SubscriptionStartDate"] != nil {
			entry["SubscriptionStartDate"] = d["SubscriptionStartDate"]
		}
		next = append(next, entry)
	}
	st := s.pinpointEmail
	st.mu.Lock()
	defer st.mu.Unlock()
	switch {
	case enabled:
		st.dashboard, st.dashboardOff, st.domains = true, time.Time{}, next
	case st.dashboard:
		st.dashboard, st.dashboardOff = false, now
	}
	return nil, nil
}

func (s *Server) ppeGetDashboard(map[string]any) (map[string]any, *ppeErr) {
	st := s.pinpointEmail
	st.mu.Lock()
	defer st.mu.Unlock()
	status, expiry := st.dashboardStatus(time.Now().UTC())
	out := map[string]any{"DashboardEnabled": st.dashboard, "AccountStatus": status}
	domains := st.model.Conform("DomainDeliverabilityTrackingOptions", append([]any(nil), st.domains...))
	switch status {
	case "ACTIVE":
		out["ActiveSubscribedDomains"] = domains
	case "PENDING_EXPIRATION":
		out["PendingExpirationSubscribedDomains"] = domains
		out["SubscriptionExpiryDate"] = mmgrEpoch(expiry)
	}
	return out, nil
}

func (s *Server) ppeBlacklistReports(in map[string]any) (map[string]any, *ppeErr) {
	report := map[string]any{}
	for _, name := range mmgrStrings(in["BlacklistItemNames"]) {
		if _, err := s.pinpointEmail.getIP(name); err != nil {
			if ident, ok := s.sesv2.GetEmailIdentity(name); !ok || ident.IdentityType != "DOMAIN" {
				return nil, ppeNotFound("%s is neither a dedicated IP nor a verified domain.", name)
			}
		}
		// Stackyard sends nothing to the internet, so nothing is ever listed.
		report[name] = []any{}
	}
	return map[string]any{"BlacklistReport": report}, nil
}

func ppeRange(in map[string]any, maxDays int) (time.Time, time.Time, *ppeErr) {
	from, ok1 := mmgrTime(in["StartDate"])
	to, ok2 := mmgrTime(in["EndDate"])
	if !ok1 || !ok2 || !from.Before(to) {
		return from, to, ppeBadRequest("StartDate must be before EndDate.")
	}
	if maxDays > 0 && to.Sub(from) > time.Duration(maxDays)*24*time.Hour {
		return from, to, ppeBadRequest("The range can span at most %d days.", maxDays)
	}
	return from, to, nil
}

func (s *Server) ppeDomainStatistics(in map[string]any) (map[string]any, *ppeErr) {
	domain := strings.ToLower(mmgrStr(in, "Domain"))
	if !s.pinpointEmail.subscribed(domain) {
		return nil, ppeNotFound("Domain %s isn't subscribed to the deliverability dashboard.", domain)
	}
	from, to, err := ppeRange(in, 31)
	if err != nil {
		return nil, err
	}
	st := s.pinpointEmail
	st.mu.Lock()
	perDay := map[string]int{}
	total := 0
	for _, m := range st.sent {
		if m.domain == domain && !m.at.Before(from) && m.at.Before(to) {
			perDay[m.at.Format("2006-01-02")] += m.recipients
			total += m.recipients
		}
	}
	st.mu.Unlock()
	volume := func(n int) map[string]any {
		return map[string]any{"InboxRawCount": n, "SpamRawCount": 0, "ProjectedInbox": n, "ProjectedSpam": 0}
	}
	var daily []any
	for day := from.Truncate(24 * time.Hour); day.Before(to); day = day.Add(24 * time.Hour) {
		daily = append(daily, map[string]any{"StartDate": mmgrEpoch(day), "VolumeStatistics": volume(perDay[day.Format("2006-01-02")]), "DomainIspPlacements": []any{}})
	}
	return map[string]any{
		"OverallVolume": map[string]any{"VolumeStatistics": volume(total), "ReadRatePercent": 0.0, "DomainIspPlacements": []any{}},
		"DailyVolumes":  daily,
	}, nil
}

func (s *Server) ppeListCampaigns(in map[string]any) (map[string]any, *ppeErr) {
	domain := strings.ToLower(mmgrStr(in, "SubscribedDomain"))
	if !s.pinpointEmail.subscribed(domain) {
		return nil, ppeNotFound("Domain %s isn't subscribed to the deliverability dashboard.", domain)
	}
	from, to, err := ppeRange(in, 0)
	if err != nil {
		return nil, err
	}
	all := s.pinpointEmail.campaigns(domain, from, to)
	start, end, next, ok := mmgrPage(len(all), mmgrStr(in, "NextToken"), mmgrInt(in, "PageSize", 0), ppeDefaultPage)
	if !ok {
		return nil, ppeBadRequest("NextToken is not valid.")
	}
	items := make([]any, 0, end-start)
	for _, c := range all[start:end] {
		items = append(items, c)
	}
	out := map[string]any{"DomainDeliverabilityCampaigns": items}
	if next != "" {
		out["NextToken"] = next
	}
	return out, nil
}

// ---- test reports ----

func (s *Server) ppeCreateTestReport(in map[string]any) (map[string]any, *ppeErr) {
	from := mmgrStr(in, "FromEmailAddress")
	tags, perr := ppeTagList(in)
	if perr != nil {
		return nil, perr
	}
	verified, dkim := s.ppeVerified(from)
	if !verified {
		return nil, ppeError(http.StatusBadRequest, "MessageRejected", "Email address is not verified. The following identities failed the check: %s", from)
	}
	m, err := s.ppeMessage(in)
	if err != nil {
		return nil, err
	}
	if len(m.raw) > ppeMaxReportMsg {
		return nil, ppeBadRequest("Stackyard keeps test messages of at most %d KiB.", ppeMaxReportMsg>>10)
	}
	r := &ppeReport{
		id: ppeNewReportID(), name: mmgrStr(in, "ReportName"), subject: m.subject, from: from,
		message: string(m.raw), status: "IN_PROGRESS", created: time.Now().UTC(), dkim: dkim,
	}
	if evicted := s.pinpointEmail.addReport(r); evicted != "" {
		s.ppeClearTags(ppeARNBase + "deliverability-test-report/" + evicted)
	}
	if len(tags) > 0 {
		_ = s.sesv2.TagResource(ppeARNBase+"deliverability-test-report/"+r.id, tags)
	}
	return map[string]any{"ReportId": r.id, "DeliverabilityTestStatus": "IN_PROGRESS"}, nil
}

func (s *Server) ppeGetTestReport(in map[string]any) (map[string]any, *ppeErr) {
	id := mmgrStr(in, "ReportId")
	summary, r, err := s.pinpointEmail.getReport(id)
	if err != nil {
		return nil, err
	}
	dkim := 0.0
	if r.dkim {
		dkim = 100
	}
	placement := map[string]any{"InboxPercentage": 100.0, "SpamPercentage": 0.0, "MissingPercentage": 0.0, "SpfPercentage": 100.0, "DkimPercentage": dkim}
	isps := make([]any, 0, len(ppeISPs))
	for _, isp := range ppeISPs {
		isps = append(isps, map[string]any{"IspName": isp, "PlacementStatistics": placement})
	}
	return map[string]any{
		"DeliverabilityTestReport": summary, "OverallPlacement": placement, "IspPlacements": isps, "Message": r.message,
		"Tags": ppeTagsOut(s.sesv2.ListTags(ppeARNBase + "deliverability-test-report/" + id)),
	}, nil
}

// ppeMessage renders EmailContent (exactly one of Simple, Raw and Template).
func (s *Server) ppeMessage(in map[string]any) (*mmgrMessage, *ppeErr) {
	content := mmgrMap(in, "Content")
	if len(content) != 1 {
		return nil, ppeBadRequest("Content sets exactly one of Simple, Raw and Template.")
	}
	if tpl := mmgrMap(content, "Template"); tpl != nil {
		arn := mmgrStr(tpl, "TemplateArn")
		if _, ok := s.sesv2.GetEmailTemplate(arn[strings.LastIndex(arn, "/")+1:]); !ok {
			return nil, ppeNotFound("Template %s doesn't exist.", arn)
		}
	}
	m := s.mmgrBuildMessage(in)
	if m == nil {
		return nil, ppeBadRequest("The message content can't be read.")
	}
	return m, nil
}

// ---- sending ----

func (s *Server) ppeSendEmail(in map[string]any) (map[string]any, *ppeErr) {
	m, err := s.ppeMessage(in)
	if err != nil {
		return nil, err
	}
	if m.from == "" {
		return nil, ppeBadRequest("FromEmailAddress is required.")
	}
	recipients := m.envelopeTo
	if len(recipients) == 0 {
		return nil, ppeBadRequest("The message has no recipients.")
	}
	if len(recipients) > ppeMaxRecipients {
		return nil, ppeError(http.StatusBadRequest, "MessageRejected", "A message can have at most %d recipients.", ppeMaxRecipients)
	}
	if ok, _ := s.ppeVerified(m.from); !ok {
		return nil, ppeError(http.StatusBadRequest, "MessageRejected", "Email address is not verified. The following identities failed the check: %s", m.from)
	}
	if cfg := mmgrStr(in, "ConfigurationSetName"); cfg != "" {
		if err := s.ppeConfigSetExists(cfg); err != nil {
			return nil, err
		}
		st := s.pinpointEmail
		st.mu.Lock()
		sending := mmgrMap(st.optsLocked(cfg), "SendingOptions")["SendingEnabled"]
		st.mu.Unlock()
		if sending == false {
			return nil, ppeError(http.StatusBadRequest, "SendingPausedException", "Sending is paused for configuration set %s.", cfg)
		}
	}
	acct := s.sesv2.GetAccount()
	if acct.SendQuota.SentLast24Hours >= acct.SendQuota.Max24HourSend { // SESv2 counts messages, not recipients
		return nil, ppeLimit("Daily message quota exceeded.")
	}
	if _, serr := s.sesv2.SendEmail(m.from, recipients); serr != nil {
		return nil, ppeFromSESv2(serr, "The message")
	}
	s.pinpointEmail.recordSent(m.from, m.subject, len(recipients))
	payload := map[string]any{"FromEmailAddress": m.from, "Destination": in["Destination"], "Content": in["Content"], "ConfigurationSetName": in["ConfigurationSetName"]}
	s.mmgrArchiveSESSend(payload, ppeARNBase+"identity/"+m.from)
	return map[string]any{"MessageId": m.id}, nil
}

// ---- tags (kept in SESv2's resource tags) ----

func (s *Server) ppeTaggable(arn string) *ppeErr {
	rest, ok := strings.CutPrefix(arn, ppeARNBase)
	kind, name, ok2 := strings.Cut(rest, "/")
	if !ok || !ok2 || name == "" {
		return ppeBadRequest("%s isn't a Pinpoint Email resource ARN.", arn)
	}
	found := false
	switch kind {
	case "configuration-set":
		_, found = s.sesv2.GetConfigurationSet(name)
	case "identity":
		_, found = s.sesv2.GetEmailIdentity(name)
	case "dedicated-ip-pool":
		found = s.pinpointEmail.poolExists(name)
	case "deliverability-test-report":
		_, _, err := s.pinpointEmail.getReport(name)
		found = err == nil
	}
	if !found {
		return ppeNotFound("Resource %s doesn't exist.", arn)
	}
	return nil
}

func (s *Server) ppeTagResource(in map[string]any) (map[string]any, *ppeErr) {
	arn := mmgrStr(in, "ResourceArn")
	if err := s.ppeTaggable(arn); err != nil {
		return nil, err
	}
	tags, err := ppeTagList(in)
	if err != nil {
		return nil, err
	}
	merged := s.sesv2.ListTags(arn)
	for k, v := range tags {
		merged[k] = v
	}
	if len(merged) > ppeMaxTags {
		return nil, ppeBadRequest("A resource can have at most %d tags.", ppeMaxTags)
	}
	return nil, ppeFromSESv2(s.sesv2.TagResource(arn, tags), arn)
}

func (s *Server) ppeUntagResource(in map[string]any) (map[string]any, *ppeErr) {
	arn := mmgrStr(in, "ResourceArn")
	if err := s.ppeTaggable(arn); err != nil {
		return nil, err
	}
	s.sesv2.UntagResource(arn, mmgrStrings(in["TagKeys"]))
	return nil, nil
}

func (s *Server) ppeListTags(in map[string]any) (map[string]any, *ppeErr) {
	arn := mmgrStr(in, "ResourceArn")
	if err := s.ppeTaggable(arn); err != nil {
		return nil, err
	}
	return map[string]any{"Tags": ppeTagsOut(s.sesv2.ListTags(arn))}, nil
}

func (s *Server) ppeClearTags(arn string) {
	tags := s.sesv2.ListTags(arn)
	keys := make([]string, 0, len(tags))
	for k := range tags {
		keys = append(keys, k)
	}
	s.sesv2.UntagResource(arn, keys)
}
