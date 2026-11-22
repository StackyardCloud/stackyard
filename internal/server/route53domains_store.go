package server

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/stackyard/stackyard/internal/awsmodels"
)

const (
	r53dAccountID   = "123456789012"
	r53dMaxDomains  = 1000
	r53dMaxOps      = 5000
	r53dMaxBilling  = 5000
	r53dMaxTags     = 50
	r53dMaxDNSSEC   = 6 // ponytail: AWS documents no per-domain DNSSEC key limit; 6 bounds the list.
	r53dDefaultPage = 20
	r53dDeletedTTL  = 30 * 24 * time.Hour // redemption window for RestoreDomain
)

type r53dError struct{ code, msg string }

func r53dErr(code, format string, args ...any) *r53dError {
	return &r53dError{code: code, msg: fmt.Sprintf(format, args...)}
}

func r53dInvalid(format string, args ...any) *r53dError {
	return r53dErr("InvalidInput", format, args...)
}

// r53dPrice is one TLD's yearly prices in USD.
type r53dPrice struct {
	tld                                       string
	register, renew, transfer, owner, restore float64
	maxYears                                  int
}

// ponytail: a fixed price list covering common TLDs; extend as needed.
var r53dPrices = []r53dPrice{
	{"app", 16, 16, 16, 0, 0, 10}, {"co", 32, 32, 32, 0, 0, 5}, {"co.uk", 9, 9, 0, 0, 0, 10},
	{"com", 15, 15, 15, 0, 0, 10}, {"dev", 14, 14, 14, 0, 0, 10}, {"info", 30, 30, 30, 0, 0, 10},
	{"io", 71, 71, 71, 0, 0, 10}, {"me", 20, 20, 20, 0, 0, 10}, {"net", 17, 17, 17, 0, 0, 10},
	{"org", 15, 15, 15, 0, 0, 10}, {"uk", 9, 9, 0, 0, 0, 10}, {"us", 10, 10, 10, 0, 0, 10},
}

// r53dReserved names are never available (the registry holds them).
var r53dReserved = map[string]bool{"example": true, "amazon": true, "aws": true, "google": true, "localhost": true, "test": true, "invalid": true}

type r53dKey struct {
	id, publicKey, digest string
	algorithm, flags, tag int
}

type r53dDomain struct {
	name, zoneID, authCode       string
	contacts                     map[string]map[string]any // Admin, Registrant, Tech, Billing
	privacy                      map[string]bool
	autoRenew, transferLock      bool
	nameservers                  []any
	created, updated, expiry     time.Time
	keys                         []r53dKey
	tags                         map[string]string
	transferTo, transferPassword string // pending transfer to another account
	deleted                      time.Time
}

type r53dOperation struct {
	id, domain, typ, status, flag string
	submitted, updated            time.Time
	reads                         int
}

type r53dBilling struct {
	domain, op, invoice string
	date                time.Time
	price               float64
}

type route53DomainsStore struct {
	mu      sync.Mutex
	domains map[string]*r53dDomain
	deleted map[string]*r53dDomain
	ops     map[string]*r53dOperation
	opOrder []string
	billing []r53dBilling
}

func newRoute53DomainsStore() *route53DomainsStore {
	return &route53DomainsStore{domains: map[string]*r53dDomain{}, deleted: map[string]*r53dDomain{}, ops: map[string]*r53dOperation{}}
}

type r53dHandler func(st *route53DomainsStore, s *Server, in map[string]any) (map[string]any, *r53dError)

var r53dHandlers map[string]r53dHandler

func init() {
	r53dHandlers = map[string]r53dHandler{
		"CheckDomainAvailability":    (*route53DomainsStore).checkAvailability,
		"CheckDomainTransferability": (*route53DomainsStore).checkTransferability,
		"RegisterDomain":             (*route53DomainsStore).register,
		"TransferDomain":             (*route53DomainsStore).transferIn,
		"RenewDomain":                (*route53DomainsStore).renew,
		"DeleteDomain":               (*route53DomainsStore).deleteDomain,
		"RestoreDomain":              (*route53DomainsStore).restore,
		"GetDomainDetail":            (*route53DomainsStore).detail,
		"ListDomains":                (*route53DomainsStore).listDomains,
		"GetDomainPrices":            (*route53DomainsStore).domainPrices,
		"ListPrices":                 (*route53DomainsStore).listPrices,
		"GetDomainSuggestions":       (*route53DomainsStore).suggestions,
		"UpdateDomainContact":        (*route53DomainsStore).updateContact,
		"UpdateDomainContactPrivacy": (*route53DomainsStore).updatePrivacy,
		"UpdateDomainNameservers":    (*route53DomainsStore).updateNameservers,
		"EnableDomainAutoRenew": func(st *route53DomainsStore, s *Server, in map[string]any) (map[string]any, *r53dError) {
			return st.setAutoRenew(in, true)
		},
		"DisableDomainAutoRenew": func(st *route53DomainsStore, s *Server, in map[string]any) (map[string]any, *r53dError) {
			return st.setAutoRenew(in, false)
		},
		"EnableDomainTransferLock": func(st *route53DomainsStore, s *Server, in map[string]any) (map[string]any, *r53dError) {
			return st.setTransferLock(in, true)
		},
		"DisableDomainTransferLock": func(st *route53DomainsStore, s *Server, in map[string]any) (map[string]any, *r53dError) {
			return st.setTransferLock(in, false)
		},
		"AssociateDelegationSignerToDomain":         (*route53DomainsStore).associateDS,
		"DisassociateDelegationSignerFromDomain":    (*route53DomainsStore).disassociateDS,
		"TransferDomainToAnotherAwsAccount":         (*route53DomainsStore).transferOut,
		"AcceptDomainTransferFromAnotherAwsAccount": (*route53DomainsStore).acceptTransfer,
		"RejectDomainTransferFromAnotherAwsAccount": (*route53DomainsStore).endTransfer,
		"CancelDomainTransferToAnotherAwsAccount":   (*route53DomainsStore).endTransfer,
		"RetrieveDomainAuthCode":                    (*route53DomainsStore).authCode,
		"PushDomain":                                (*route53DomainsStore).push,
		"GetOperationDetail":                        (*route53DomainsStore).operationDetail,
		"ListOperations":                            (*route53DomainsStore).listOperations,
		"ResendOperationAuthorization":              (*route53DomainsStore).resendAuthorization,
		"GetContactReachabilityStatus":              (*route53DomainsStore).reachability,
		"ResendContactReachabilityEmail":            (*route53DomainsStore).resendReachability,
		"ViewBilling":                               (*route53DomainsStore).viewBilling,
		"ListTagsForDomain":                         (*route53DomainsStore).listTags,
		"UpdateTagsForDomain":                       (*route53DomainsStore).updateTags,
		"DeleteTagsForDomain":                       (*route53DomainsStore).deleteTags,
	}
}

// handle runs op under the store lock and shapes the result to the output
// model before unlocking, so nothing stored is read after unlock.
func (st *route53DomainsStore) handle(s *Server, op, outShape string, in map[string]any) (any, *r53dError) {
	h := r53dHandlers[op]
	if h == nil {
		return nil, r53dInvalid("operation %s has no handler", op)
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	st.purgeDeletedLocked()
	out, err := h(st, s, in)
	if err != nil {
		return nil, err
	}
	if outShape == "" {
		return map[string]any{}, nil
	}
	return awsmodels.Route53DomainsModel().Conform(outShape, out), nil
}

// ---- helpers ----

func r53dStr(in map[string]any, key string) string { v, _ := in[key].(string); return v }

func r53dBool(in map[string]any, key string) (bool, bool) { v, ok := in[key].(bool); return v, ok }

func r53dEpoch(t time.Time) float64 { return float64(t.UnixMilli()) / 1000 }

func r53dTime(v any) (time.Time, bool) {
	switch t := v.(type) {
	case float64:
		return time.UnixMilli(int64(t * 1000)).UTC(), true
	case string:
		if p, err := time.Parse(time.RFC3339, t); err == nil {
			return p, true
		}
		if f, err := strconv.ParseFloat(t, 64); err == nil {
			return time.UnixMilli(int64(f * 1000)).UTC(), true
		}
	}
	return time.Time{}, false
}

func r53dRandom(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// r53dSplit validates a domain name and returns its label and price entry
// (the longest matching TLD in the price list).
func r53dSplit(name string) (string, r53dPrice, *r53dError) {
	name = strings.TrimSuffix(strings.ToLower(name), ".")
	if len(name) > 255 || !strings.Contains(name, ".") {
		return "", r53dPrice{}, r53dInvalid("domain name %q is not valid", name)
	}
	for _, label := range strings.Split(name, ".") {
		if label == "" || len(label) > 63 || strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
			return "", r53dPrice{}, r53dInvalid("domain name %q is not valid", name)
		}
		for _, c := range label {
			if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-') {
				return "", r53dPrice{}, r53dInvalid("domain name %q is not valid", name)
			}
		}
	}
	best := -1
	for i, p := range r53dPrices {
		if strings.HasSuffix(name, "."+p.tld) && (best < 0 || len(p.tld) > len(r53dPrices[best].tld)) {
			best = i
		}
	}
	if best < 0 {
		return "", r53dPrice{}, r53dErr("UnsupportedTLD", "Route 53 doesn't support the top-level domain of %s", name)
	}
	p := r53dPrices[best]
	label := strings.TrimSuffix(name, "."+p.tld)
	if strings.Contains(label, ".") {
		return "", r53dPrice{}, r53dInvalid("%s is a subdomain; register the second-level domain instead", name)
	}
	return label, p, nil
}

func r53dPriceOf(amount float64) map[string]any {
	return map[string]any{"Price": amount, "Currency": "USD"}
}

func (st *route53DomainsStore) domainLocked(in map[string]any) (*r53dDomain, *r53dError) {
	name := strings.TrimSuffix(strings.ToLower(r53dStr(in, "DomainName")), ".")
	d := st.domains[name]
	if d == nil {
		return nil, r53dInvalid("domain %s isn't registered with Route 53 in this account", name)
	}
	return d, nil
}

func (st *route53DomainsStore) purgeDeletedLocked() {
	for name, d := range st.deleted {
		if time.Since(d.deleted) > r53dDeletedTTL {
			delete(st.deleted, name)
		}
	}
}

// newOpLocked records an operation; it reports SUBMITTED, then IN_PROGRESS,
// then SUCCESSFUL as it is read.
func (st *route53DomainsStore) newOpLocked(domain, typ, flag string) (string, *r53dError) {
	if len(st.ops) >= r53dMaxOps {
		return "", r53dErr("OperationLimitExceeded", "too many operations; at most %d are kept", r53dMaxOps)
	}
	now := time.Now().UTC()
	op := &r53dOperation{id: uuid.NewString(), domain: domain, typ: typ, status: "SUBMITTED", flag: flag, submitted: now, updated: now}
	st.ops[op.id] = op
	st.opOrder = append(st.opOrder, op.id)
	return op.id, nil
}

func (op *r53dOperation) read() {
	if op.flag != "" || op.status == "SUCCESSFUL" || op.status == "FAILED" {
		return
	}
	op.reads++
	if op.reads == 1 {
		op.status = "IN_PROGRESS"
	} else {
		op.status = "SUCCESSFUL"
	}
	op.updated = time.Now().UTC()
}

func (st *route53DomainsStore) billLocked(domain, op string, price float64) {
	if price <= 0 {
		return
	}
	if len(st.billing) >= r53dMaxBilling {
		st.billing = st.billing[1:]
	}
	st.billing = append(st.billing, r53dBilling{domain: domain, op: op, invoice: "stackyard-" + r53dRandom(6), date: time.Now().UTC(), price: price})
}

func r53dCheckConsent(in map[string]any, price float64) *r53dError {
	c, ok := in["Consent"].(map[string]any)
	if !ok {
		return nil
	}
	if r53dStr(c, "Currency") != "USD" {
		return r53dInvalid("Consent.Currency must be USD")
	}
	if max, _ := c["MaxPrice"].(float64); max < price {
		return r53dInvalid("the price %.2f USD is above Consent.MaxPrice %.2f", price, max)
	}
	return nil
}

func r53dYears(in map[string]any, p r53dPrice, def int) (int, *r53dError) {
	years := def
	if v, ok := in["DurationInYears"].(float64); ok {
		years = int(v)
	}
	if years < 1 || years > p.maxYears {
		return 0, r53dErr("TLDRulesViolation", ".%s domains can be registered for 1 to %d years", p.tld, p.maxYears)
	}
	return years, nil
}

func r53dContacts(in map[string]any) map[string]map[string]any {
	out := map[string]map[string]any{}
	for _, kind := range []string{"Admin", "Registrant", "Tech", "Billing"} {
		if c, ok := in[kind+"Contact"].(map[string]any); ok {
			out[kind] = c
		}
	}
	if _, ok := out["Billing"]; !ok {
		out["Billing"] = out["Registrant"]
	}
	return out
}

// r53dPrivacy reads per-contact privacy flags; key builds the member name
// (RegisterDomain: PrivacyProtectAdminContact, UpdateDomainContactPrivacy:
// AdminPrivacy). Privacy protection is on unless set false.
func r53dPrivacy(in map[string]any, key func(kind string) string) map[string]bool {
	out := map[string]bool{}
	for _, kind := range []string{"Admin", "Registrant", "Tech", "Billing"} {
		v, ok := r53dBool(in, key(kind))
		out[kind] = !ok || v
	}
	return out
}

func r53dProtectKey(kind string) string { return "PrivacyProtect" + kind + "Contact" }

// createDomainLocked adds a domain with a new hosted zone (unless nameservers
// are given) and returns it.
func (st *route53DomainsStore) createDomainLocked(s *Server, name string, in map[string]any, years int, nameservers []any) (*r53dDomain, *r53dError) {
	if len(st.domains) >= r53dMaxDomains {
		return nil, r53dErr("DomainLimitExceeded", "an account can hold at most %d domains here", r53dMaxDomains)
	}
	now := time.Now().UTC()
	d := &r53dDomain{
		name: name, authCode: r53dRandom(8), contacts: r53dContacts(in), privacy: r53dPrivacy(in, r53dProtectKey),
		autoRenew: true, transferLock: true, created: now, updated: now, expiry: now.AddDate(years, 0, 0), tags: map[string]string{},
	}
	if v, ok := r53dBool(in, "AutoRenew"); ok {
		d.autoRenew = v
	}
	if len(nameservers) > 0 {
		d.nameservers = nameservers
	} else {
		zone, ns := s.route53.CreateZoneForDomain(name)
		d.zoneID = zone
		for _, n := range ns {
			d.nameservers = append(d.nameservers, map[string]any{"Name": n})
		}
	}
	st.domains[name] = d
	delete(st.deleted, name)
	return d, nil
}

func r53dOpOut(id string) map[string]any { return map[string]any{"OperationId": id} }

func r53dPage(in map[string]any, total, def, max int) (int, int, *r53dError) {
	start := 0
	if m := r53dStr(in, "Marker"); m != "" {
		n, err := strconv.ParseInt(m, 10, 32)
		if err != nil || n < 0 || int(n) > total {
			return 0, 0, r53dInvalid("Marker is not valid")
		}
		start = int(n)
	}
	size := def
	if v, ok := in["MaxItems"].(float64); ok {
		size = int(v)
	}
	if size < 1 || size > max {
		return 0, 0, r53dInvalid("MaxItems must be between 1 and %d", max)
	}
	end := min(start+size, total)
	return start, end, nil
}

func r53dNext(out map[string]any, end, total int) {
	if end < total {
		out["NextPageMarker"] = strconv.Itoa(end)
	}
}

func r53dSortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
