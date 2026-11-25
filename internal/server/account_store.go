package server

import (
	"sort"
	"sync"
	"time"
)

// acctDefaultRegions are enabled for every account; the rest of
// acctRegionNames are opt-in Regions that start DISABLED.
var acctDefaultRegions = map[string]bool{
	"ap-northeast-1": true, "ap-northeast-2": true, "ap-northeast-3": true, "ap-south-1": true,
	"ap-southeast-1": true, "ap-southeast-2": true, "ca-central-1": true, "eu-central-1": true,
	"eu-north-1": true, "eu-west-1": true, "eu-west-2": true, "eu-west-3": true, "sa-east-1": true,
	"us-east-1": true, "us-east-2": true, "us-west-1": true, "us-west-2": true,
}

var acctOptInRegions = []string{
	"af-south-1", "ap-east-1", "ap-east-2", "ap-south-2", "ap-southeast-3", "ap-southeast-4",
	"ap-southeast-5", "ap-southeast-6", "ap-southeast-7", "ca-west-1", "eu-central-2", "eu-south-1",
	"eu-south-2", "il-central-1", "me-central-1", "me-south-1", "mx-central-1",
}

var acctRegionNames = func() []string {
	out := append([]string{}, acctOptInRegions...)
	for name := range acctDefaultRegions {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}()

// acctStore holds one record per organization account. Records exist only
// for accounts the Organizations store knows, and every field is fixed-size
// or bounded by the model (three alternate contacts, one contact record, the
// fixed Region list), so the store needs no further caps.
type acctStore struct {
	mu       sync.Mutex
	accounts map[string]*acctAccount
}

type acctAccount struct {
	id, name, email string
	created         time.Time
	contacts        map[string]map[string]any // AlternateContactType -> contact
	info            map[string]any            // ContactInformation
	phoneStatus     string
	regions         map[string]*acctRegion // opt-in Regions only
	emailUpdate     *acctEmailUpdate
}

// acctRegion is an opt-in Region. ENABLING and DISABLING are reported on the
// first read after the change and settle on the next one.
type acctRegion struct {
	status string
	seen   bool
}

type acctEmailUpdate struct {
	email, status string
	updated       time.Time
	seen          bool
}

func newAccountStore() *acctStore { return &acctStore{accounts: map[string]*acctAccount{}} }

func nowUTC() time.Time { return time.Now().UTC() }

// get returns the record for a, creating it from the organization's view of
// the account; callers hold st.mu.
func (st *acctStore) get(a organizationsAccount) *acctAccount {
	if acc := st.accounts[a.ID]; acc != nil {
		return acc
	}
	acc := &acctAccount{
		id: a.ID, name: a.Name, email: a.Email, created: a.Joined,
		contacts: map[string]map[string]any{}, phoneStatus: "UNVERIFIED", regions: map[string]*acctRegion{},
	}
	for _, name := range acctOptInRegions {
		acc.regions[name] = &acctRegion{status: "DISABLED"}
	}
	st.accounts[a.ID] = acc
	return acc
}

// emailInUse reports whether any organization account already uses email as
// its primary address; callers hold st.mu.
func (st *acctStore) emailInUse(email string) bool {
	for _, a := range organizationsAccounts {
		current := a.Email
		if acc := st.accounts[a.ID]; acc != nil {
			current = acc.email
		}
		if current == email {
			return true
		}
	}
	return false
}

// observeRegion returns a Region's status as a read reports it, advancing a
// transition that was already reported once.
func (a *acctAccount) observeRegion(name string) (string, bool) {
	if acctDefaultRegions[name] {
		return "ENABLED_BY_DEFAULT", true
	}
	r := a.regions[name]
	if r == nil {
		return "", false
	}
	if r.status == "ENABLING" || r.status == "DISABLING" {
		if r.seen {
			r.status = map[string]string{"ENABLING": "ENABLED", "DISABLING": "DISABLED"}[r.status]
		} else {
			r.seen = true
		}
	}
	return r.status, true
}
