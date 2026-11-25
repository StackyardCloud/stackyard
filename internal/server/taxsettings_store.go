package server

import (
	"sort"
	"sync"
	"time"
)

// taxsStore holds registrations and exemptions for organization accounts
// only (resolved before the lock), so per-account maps are bounded by the
// roster; documents, supplemental registrations and exemptions have caps.
type taxsStore struct {
	mu           sync.Mutex
	regs         map[string]*taxsReg
	docBytes     int // bytes of every stored document, charged on each put
	supplemental []map[string]any
	exemptions   map[string][]*taxsExemption
	heritage     string
}

type taxsReg struct {
	entry  map[string]any
	status string // Pending (documents to review) reported once, then Verified
	seen   bool
	docs   []taxsDoc
}

type taxsDoc struct {
	token, name string
	body        []byte // never mutated once stored
}

type taxsExemption struct {
	country, state, caseID string
	typ                    *taxsExemptionKind
	status                 string // Pending reported once, then Valid
	seen                   bool
	effective, expires     time.Time
}

func newTaxSettingsStore() *taxsStore {
	return &taxsStore{regs: map[string]*taxsReg{}, exemptions: map[string][]*taxsExemption{}, heritage: "OptOut"}
}

func taxsDocBytes(docs []taxsDoc) int {
	n := 0
	for _, d := range docs {
		n += len(d.body)
	}
	return n
}

// put stores account's registration, replacing any earlier one and
// re-charging document bytes against the store cap; callers hold st.mu.
func (st *taxsStore) put(account string, entry map[string]any, docs []taxsDoc) (*taxsReg, *budErr) {
	old := 0
	if prev := st.regs[account]; prev != nil {
		old = taxsDocBytes(prev.docs)
	}
	if st.docBytes-old+taxsDocBytes(docs) > taxsMaxDocBytes {
		return nil, taxsInvalid("Stored tax registration documents would exceed %d bytes.", taxsMaxDocBytes)
	}
	st.docBytes += taxsDocBytes(docs) - old
	status := "Verified"
	if len(docs) > 0 {
		status = "Pending"
	}
	reg := &taxsReg{entry: entry, status: status, docs: docs}
	st.regs[account] = reg
	return reg, nil
}

func (st *taxsStore) remove(account string) bool {
	reg := st.regs[account]
	if reg == nil {
		return false
	}
	st.docBytes -= taxsDocBytes(reg.docs)
	delete(st.regs, account)
	return true
}

func (st *taxsStore) document(token string) (taxsDoc, bool) {
	for _, reg := range st.regs {
		for _, d := range reg.docs {
			if d.token == token {
				return d, true
			}
		}
	}
	return taxsDoc{}, false
}

func (st *taxsStore) registeredAccounts() []string {
	out := make([]string, 0, len(st.regs))
	for id := range st.regs {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

// out shapes a registration (with its jurisdiction for listings), settling a
// Pending status that was already reported once.
func (r *taxsReg) out(listing bool) map[string]any {
	if r.status == "Pending" {
		if r.seen {
			r.status = "Verified"
		}
		r.seen = true
	}
	out := budCloneMap(r.entry)
	out["status"] = r.status
	if len(r.docs) > 0 {
		metas := make([]any, 0, len(r.docs))
		for _, d := range r.docs {
			metas = append(metas, map[string]any{"taxDocumentAccessToken": d.token, "taxDocumentName": d.name})
		}
		out["taxDocumentMetadatas"] = metas
	}
	if listing {
		addr := budMap(r.entry, "legalAddress")
		j := map[string]any{"countryCode": budStr(addr, "countryCode")}
		if state := budStr(addr, "stateOrRegion"); state != "" {
			j["stateOrRegion"] = state
		}
		out["jurisdiction"] = j
	}
	return out
}

// ---- exemptions ----

type taxsExemptionKind struct {
	name, description string
	country           string
	states            []string // empty: every state
}

var taxsExemptionTypes = []*taxsExemptionKind{
	{name: "Resale", description: "Purchases for resale to customers", country: "US", states: []string{"CA", "FL", "NY", "TX", "WA"}},
	{name: "Federal Government", description: "United States federal government agencies", country: "US"},
	{name: "State Government", description: "State and local government agencies", country: "US"},
	{name: "501(c)(3) Non-profit", description: "Charitable organizations exempt under IRC 501(c)(3)", country: "US"},
	{name: "Educational Institution", description: "Accredited schools, colleges and universities", country: "US"},
	{name: "Manufacturing", description: "Purchases used directly in manufacturing", country: "US", states: []string{"OH", "PA", "TX"}},
}

func taxsExemptionType(name string) *taxsExemptionKind {
	for _, t := range taxsExemptionTypes {
		if t.name == name {
			return t
		}
	}
	return nil
}

func (t *taxsExemptionKind) applies(country, state string) bool {
	if country != t.country {
		return false
	}
	return len(t.states) == 0 || budContains(t.states, state)
}

func (t *taxsExemptionKind) out() map[string]any {
	var juris []any
	if len(t.states) == 0 {
		juris = []any{map[string]any{"country": t.country}}
	}
	for _, s := range t.states {
		juris = append(juris, map[string]any{"country": t.country, "state": s})
	}
	return map[string]any{"displayName": t.name, "description": t.description, "applicableJurisdictions": juris}
}

// exemptionFits reports whether account can take an exemption for this
// authority (a replacement always fits); callers hold st.mu.
func (st *taxsStore) exemptionFits(account, country, state string) bool {
	list := st.exemptions[account]
	for _, e := range list {
		if e.country == country && e.state == state {
			return true
		}
	}
	return len(list) < taxsMaxExemption
}

// putExemption records (or replaces) account's exemption for one authority;
// callers hold st.mu and have checked exemptionFits.
func (st *taxsStore) putExemption(account, country, state string, typ *taxsExemptionKind, caseID string) {
	now := nowUTC()
	ex := &taxsExemption{country: country, state: state, caseID: caseID, typ: typ, status: "Pending", effective: now, expires: now.AddDate(3, 0, 0)}
	list := st.exemptions[account]
	for i, e := range list {
		if e.country == country && e.state == state {
			next := append([]*taxsExemption{}, list...)
			next[i] = ex
			st.exemptions[account] = next
			return
		}
	}
	st.exemptions[account] = append(append([]*taxsExemption{}, list...), ex)
}

// exemptAccounts lists accounts with exemptions, plus members inheriting the
// management account's.
func (st *taxsStore) exemptAccounts() []string {
	var out []string
	for _, a := range organizationsAccounts {
		if len(st.exemptions[a.ID]) > 0 || st.inherits(a) {
			out = append(out, a.ID)
		}
	}
	return out
}

func (st *taxsStore) inherits(a organizationsAccount) bool {
	return !a.Management && st.heritage == "OptIn" && len(st.exemptions[acctCaller]) > 0
}

func (st *taxsStore) exemptionDetails(account string) map[string]any {
	list := []any{}
	for _, e := range st.exemptions[account] {
		if e.status == "Pending" {
			if e.seen {
				e.status = "Valid"
			}
			e.seen = true
		}
		authority := map[string]any{"country": e.country}
		if e.state != "" {
			authority["state"] = e.state
		}
		list = append(list, map[string]any{
			"authority": authority, "taxExemptionType": e.typ.out(), "status": e.status,
			"effectiveDate": budEpoch(e.effective), "systemEffectiveDate": budEpoch(e.effective), "expirationDate": budEpoch(e.expires),
		})
	}
	out := map[string]any{"taxExemptions": list}
	for _, a := range organizationsAccounts {
		if a.ID == account && st.inherits(a) {
			out["heritageObtainedDetails"] = true
			out["heritageObtainedParentEntity"] = acctCaller
			out["heritageObtainedReason"] = "Inherited from the management account's tax settings"
		}
	}
	return out
}
