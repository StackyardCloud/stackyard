package server

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/stackyard/stackyard/internal/awsmodels"
)

// Pinpoint Email state that SESv2 doesn't keep. Callers finish any SESv2
// service call before taking ppeStore's lock.

const (
	ppeARNBase       = "arn:aws:ses:us-east-1:123456789012:"
	ppeMaxConfigSets = 10000
	ppeMaxIdentities = 10000
	ppeMaxPools      = 50
	ppeMaxDests      = 10 // event destinations per configuration set
	ppeMaxReports    = 50 // test reports kept; the oldest are dropped
	ppeMaxReportMsg  = 256 << 10
	ppeMaxSent       = 1000 // send log for dashboard statistics; the oldest are dropped
	ppeMaxDomains    = 20
	ppeMaxTags       = 50
	ppeMaxRecipients = 50
	ppeDefaultPage   = 100
)

// ppeSeedIPs are the account's dedicated IPs (TEST-NET-2 addresses).
var ppeSeedIPs = []string{"198.51.100.10", "198.51.100.11", "198.51.100.12", "198.51.100.13"}

var ppeISPs = []string{"Gmail", "Hotmail", "Yahoo", "AOL", "Comcast"}

type ppeIP struct {
	pool string
	pct  int
}

type ppeReport struct {
	id, name, subject, from, message, status string
	created                                  time.Time
	dkim                                     bool
}

type ppeSent struct {
	from, domain, subject string
	recipients            int
	at                    time.Time
}

type ppeStore struct {
	mu           sync.Mutex
	model        *awsmodels.Model
	opts         map[string]map[string]any // configuration set -> conformed options
	pools        map[string]time.Time
	ips          map[string]*ppeIP
	autoWarmup   bool
	dashboard    bool
	dashboardOff time.Time
	domains      []any // conformed DomainDeliverabilityTrackingOption
	reports      map[string]*ppeReport
	reportOrder  []string
	sent         []ppeSent
}

func newPinpointEmailStore() *ppeStore {
	st := &ppeStore{
		model: awsmodels.PinpointEmailModel(), opts: map[string]map[string]any{}, pools: map[string]time.Time{},
		ips: map[string]*ppeIP{}, reports: map[string]*ppeReport{},
	}
	for _, ip := range ppeSeedIPs {
		st.ips[ip] = &ppeIP{pct: 100}
	}
	return st
}

func ppeDefaultOpts() map[string]any {
	return map[string]any{
		"SendingOptions":    map[string]any{"SendingEnabled": true},
		"ReputationOptions": map[string]any{"ReputationMetricsEnabled": false},
		"DeliveryOptions":   map[string]any{"TlsPolicy": "OPTIONAL"},
	}
}

// optsLocked returns a copy of a configuration set's options.
func (st *ppeStore) optsLocked(name string) map[string]any {
	out := ppeDefaultOpts()
	for k, v := range st.opts[name] {
		out[k] = v
	}
	return out
}

func (st *ppeStore) setOpt(name, key string, v any) {
	st.mu.Lock()
	defer st.mu.Unlock()
	next := st.optsLocked(name)
	if v == nil {
		delete(next, key)
	} else {
		next[key] = v
	}
	st.opts[name] = next
}

func (st *ppeStore) poolExists(name string) bool {
	st.mu.Lock()
	defer st.mu.Unlock()
	_, ok := st.pools[name]
	return ok
}

func ppeTagList(in map[string]any) (map[string]string, *ppeErr) {
	out := map[string]string{}
	for _, item := range mmgrList(in, "Tags") {
		t, _ := item.(map[string]any)
		key := mmgrStr(t, "Key")
		if strings.HasPrefix(strings.ToLower(key), "aws:") {
			return nil, ppeBadRequest("Tag keys can't start with aws:.")
		}
		out[key] = mmgrStr(t, "Value")
	}
	if len(out) > ppeMaxTags {
		return nil, ppeBadRequest("A resource can have at most %d tags.", ppeMaxTags)
	}
	return out, nil
}

func ppeTagsOut(tags map[string]string) []any {
	keys := make([]string, 0, len(tags))
	for k := range tags {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, k := range keys {
		out = append(out, map[string]any{"Key": k, "Value": tags[k]})
	}
	return out
}

// ppePage pages sorted names.
func ppePage(names []string, in map[string]any, key string) (map[string]any, *ppeErr) {
	start, end, next, ok := mmgrPage(len(names), mmgrStr(in, "NextToken"), mmgrInt(in, "PageSize", 0), ppeDefaultPage)
	if !ok {
		return nil, ppeBadRequest("NextToken is not valid.")
	}
	items := make([]any, 0, end-start)
	for _, n := range names[start:end] {
		items = append(items, n)
	}
	out := map[string]any{key: items}
	if next != "" {
		out["NextToken"] = next
	}
	return out, nil
}

func ppeDomainOf(address string) string {
	if _, domain, ok := strings.Cut(address, "@"); ok {
		return strings.ToLower(domain)
	}
	return strings.ToLower(address)
}

// ---- dedicated IPs and pools ----

func (st *ppeStore) ipView(ip string, v *ppeIP) map[string]any {
	status := "DONE"
	if v.pct < 100 {
		status = "IN_PROGRESS"
	}
	out := map[string]any{"Ip": ip, "WarmupStatus": status, "WarmupPercentage": v.pct}
	if v.pool != "" {
		out["PoolName"] = v.pool
	}
	return out
}

func (st *ppeStore) createPool(name string) *ppeErr {
	st.mu.Lock()
	defer st.mu.Unlock()
	if _, ok := st.pools[name]; ok {
		return ppeError(400, "AlreadyExistsException", "Dedicated IP pool %s already exists.", name)
	}
	if len(st.pools) >= ppeMaxPools {
		return ppeLimit("Stackyard keeps at most %d dedicated IP pools.", ppeMaxPools)
	}
	st.pools[name] = time.Now().UTC()
	return nil
}

func (st *ppeStore) deletePool(name string) *ppeErr {
	st.mu.Lock()
	defer st.mu.Unlock()
	if _, ok := st.pools[name]; !ok {
		return ppeNotFound("Dedicated IP pool %s doesn't exist.", name)
	}
	delete(st.pools, name)
	for _, ip := range st.ips {
		if ip.pool == name {
			ip.pool = ""
		}
	}
	for cfg, opts := range st.opts {
		if mmgrStr(mmgrMap(opts, "DeliveryOptions"), "SendingPoolName") == name {
			next := st.optsLocked(cfg)
			next["DeliveryOptions"] = map[string]any{"TlsPolicy": mmgrStr(mmgrMap(opts, "DeliveryOptions"), "TlsPolicy")}
			st.opts[cfg] = next
		}
	}
	return nil
}

func (st *ppeStore) listPools(in map[string]any) (map[string]any, *ppeErr) {
	st.mu.Lock()
	names := make([]string, 0, len(st.pools))
	for n := range st.pools {
		names = append(names, n)
	}
	st.mu.Unlock()
	sort.Strings(names)
	return ppePage(names, in, "DedicatedIpPools")
}

func (st *ppeStore) getIP(ip string) (map[string]any, *ppeErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	v := st.ips[ip]
	if v == nil {
		return nil, ppeNotFound("Dedicated IP %s doesn't exist.", ip)
	}
	return map[string]any{"DedicatedIp": st.ipView(ip, v)}, nil
}

func (st *ppeStore) listIPs(in map[string]any) (map[string]any, *ppeErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	pool := mmgrStr(in, "PoolName")
	if _, ok := st.pools[pool]; pool != "" && !ok {
		return nil, ppeNotFound("Dedicated IP pool %s doesn't exist.", pool)
	}
	var ips []string
	for ip, v := range st.ips {
		if pool == "" || v.pool == pool {
			ips = append(ips, ip)
		}
	}
	sort.Strings(ips)
	start, end, next, ok := mmgrPage(len(ips), mmgrStr(in, "NextToken"), mmgrInt(in, "PageSize", 0), ppeDefaultPage)
	if !ok {
		return nil, ppeBadRequest("NextToken is not valid.")
	}
	items := make([]any, 0, end-start)
	for _, ip := range ips[start:end] {
		items = append(items, st.ipView(ip, st.ips[ip]))
	}
	out := map[string]any{"DedicatedIps": items}
	if next != "" {
		out["NextToken"] = next
	}
	return out, nil
}

func (st *ppeStore) putIP(ip, pool string, pct int) *ppeErr {
	st.mu.Lock()
	defer st.mu.Unlock()
	v := st.ips[ip]
	if v == nil {
		return ppeNotFound("Dedicated IP %s doesn't exist.", ip)
	}
	if pool != "" {
		if _, ok := st.pools[pool]; !ok {
			return ppeNotFound("Dedicated IP pool %s doesn't exist.", pool)
		}
		v.pool = pool
	}
	if pct >= 0 {
		v.pct = pct
	}
	return nil
}

// ---- deliverability dashboard ----

// dashboardStatus reports the account status and, while a disabled
// subscription runs to the end of its month, its expiry.
func (st *ppeStore) dashboardStatus(now time.Time) (string, time.Time) {
	if st.dashboard {
		return "ACTIVE", time.Time{}
	}
	if !st.dashboardOff.IsZero() {
		y, m, _ := st.dashboardOff.Date()
		expiry := time.Date(y, m+1, 1, 0, 0, 0, 0, time.UTC)
		if now.Before(expiry) {
			return "PENDING_EXPIRATION", expiry
		}
	}
	return "DISABLED", time.Time{}
}

func (st *ppeStore) subscribed(domain string) bool {
	st.mu.Lock()
	defer st.mu.Unlock()
	if status, _ := st.dashboardStatus(time.Now().UTC()); status == "DISABLED" {
		return false
	}
	for _, d := range st.domains {
		if strings.EqualFold(mmgrStr(d.(map[string]any), "Domain"), domain) {
			return true
		}
	}
	return false
}

// campaigns groups the send log by sending domain and subject.
func (st *ppeStore) campaigns(domain string, from, to time.Time) []map[string]any {
	st.mu.Lock()
	defer st.mu.Unlock()
	byID := map[string]map[string]any{}
	var order []string
	for _, m := range st.sent {
		if (domain != "" && m.domain != domain) || (!from.IsZero() && (m.at.Before(from) || !m.at.Before(to))) {
			continue
		}
		sum := sha256.Sum256([]byte(m.domain + "\x00" + m.subject))
		id := hex.EncodeToString(sum[:10])
		c := byID[id]
		if c == nil {
			c = map[string]any{
				"CampaignId": id, "Subject": m.subject, "FromAddress": m.from, "FirstSeenDateTime": mmgrEpoch(m.at),
				"SendingIps": []any{}, "Esps": []any{}, "InboxCount": 0, "SpamCount": 0, "ProjectedVolume": 0,
				"ReadRate": 0.0, "DeleteRate": 0.0, "ReadDeleteRate": 0.0,
			}
			byID[id] = c
			order = append(order, id)
		}
		c["LastSeenDateTime"] = mmgrEpoch(m.at)
		c["InboxCount"] = c["InboxCount"].(int) + m.recipients
		c["ProjectedVolume"] = c["InboxCount"]
	}
	out := make([]map[string]any, 0, len(order))
	for _, id := range order {
		out = append(out, byID[id])
	}
	return out
}

func (st *ppeStore) recordSent(from, subject string, recipients int) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if len(subject) > 256 {
		subject = subject[:256]
	}
	st.sent = append(st.sent, ppeSent{from: from, domain: ppeDomainOf(from), subject: subject, recipients: recipients, at: time.Now().UTC()})
	if len(st.sent) > ppeMaxSent {
		st.sent = append([]ppeSent(nil), st.sent[len(st.sent)-ppeMaxSent:]...)
	}
}

// ---- test reports ----

func (st *ppeStore) addReport(r *ppeReport) (evicted string) {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.reports[r.id] = r
	st.reportOrder = append(st.reportOrder, r.id)
	if len(st.reportOrder) > ppeMaxReports {
		evicted = st.reportOrder[0]
		delete(st.reports, evicted)
		st.reportOrder = st.reportOrder[1:]
	}
	return evicted
}

// reportView settles a report (IN_PROGRESS reports complete on read) and
// shapes its summary; callers hold the lock.
func (st *ppeStore) reportView(r *ppeReport) map[string]any {
	r.status = "COMPLETED"
	return map[string]any{
		"ReportId": r.id, "ReportName": r.name, "Subject": r.subject, "FromEmailAddress": r.from,
		"CreateDate": mmgrEpoch(r.created), "DeliverabilityTestStatus": r.status,
	}
}

func (st *ppeStore) getReport(id string) (map[string]any, *ppeReport, *ppeErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	r := st.reports[id]
	if r == nil {
		return nil, nil, ppeNotFound("Deliverability test report %s doesn't exist.", id)
	}
	cp := *r
	return st.reportView(r), &cp, nil
}

func (st *ppeStore) listReports(in map[string]any) (map[string]any, *ppeErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	start, end, next, ok := mmgrPage(len(st.reportOrder), mmgrStr(in, "NextToken"), mmgrInt(in, "PageSize", 0), ppeDefaultPage)
	if !ok {
		return nil, ppeBadRequest("NextToken is not valid.")
	}
	items := make([]any, 0, end-start)
	for _, id := range st.reportOrder[start:end] {
		items = append(items, st.reportView(st.reports[id]))
	}
	out := map[string]any{"DeliverabilityTestReports": items}
	if next != "" {
		out["NextToken"] = next
	}
	return out, nil
}

func ppeNewReportID() string { return strings.ReplaceAll(uuid.NewString(), "-", "") }
