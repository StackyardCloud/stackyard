package server

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"sort"
	"strings"
	"time"
)

// ---- availability, prices, suggestions ----

func (st *route53DomainsStore) availabilityLocked(name string) (string, r53dPrice, *r53dError) {
	label, p, err := r53dSplit(name)
	if err != nil {
		return "", p, err
	}
	name = label + "." + p.tld
	switch {
	case st.domains[name] != nil, r53dReserved[label]:
		return "UNAVAILABLE", p, nil
	case st.deleted[name] != nil:
		return "AVAILABLE_RESERVED", p, nil // in its redemption window
	}
	return "AVAILABLE", p, nil
}

func (st *route53DomainsStore) checkAvailability(_ *Server, in map[string]any) (map[string]any, *r53dError) {
	avail, p, err := st.availabilityLocked(r53dStr(in, "DomainName"))
	if err != nil {
		return nil, err
	}
	out := map[string]any{"Availability": avail}
	if avail == "AVAILABLE" {
		out["Price"] = r53dPriceOf(p.register)
	}
	return out, nil
}

func (st *route53DomainsStore) checkTransferability(_ *Server, in map[string]any) (map[string]any, *r53dError) {
	label, p, err := r53dSplit(r53dStr(in, "DomainName"))
	if err != nil {
		return nil, err
	}
	name := label + "." + p.tld
	state, msg := "TRANSFERABLE", "The domain can be transferred to Route 53."
	switch d := st.domains[name]; {
	case d != nil && d.transferTo != "":
		state, msg = "DOMAIN_IN_ANOTHER_ACCOUNT", "A transfer of this domain to another AWS account is pending."
	case d != nil:
		state, msg = "DOMAIN_IN_OWN_ACCOUNT", "The domain is already registered with Route 53 in this account."
	case r53dReserved[label] || p.transfer == 0:
		state, msg = "UNTRANSFERABLE", "The domain can't be transferred to Route 53."
	}
	out := map[string]any{"Transferability": map[string]any{"Transferable": state}, "Message": msg}
	if state == "TRANSFERABLE" {
		out["Price"] = r53dPriceOf(p.transfer)
	}
	return out, nil
}

func r53dDomainPrice(p r53dPrice) map[string]any {
	out := map[string]any{"Name": p.tld, "RegistrationPrice": r53dPriceOf(p.register), "RenewalPrice": r53dPriceOf(p.renew)}
	if p.transfer > 0 {
		out["TransferPrice"] = r53dPriceOf(p.transfer)
	}
	return out
}

func (st *route53DomainsStore) domainPrices(_ *Server, in map[string]any) (map[string]any, *r53dError) {
	_, p, err := r53dSplit(r53dStr(in, "DomainName"))
	if err != nil {
		return nil, err
	}
	return map[string]any{"DomainPrice": r53dDomainPrice(p)}, nil
}

func (st *route53DomainsStore) listPrices(_ *Server, in map[string]any) (map[string]any, *r53dError) {
	prices := r53dPrices
	if tld := strings.ToLower(strings.TrimPrefix(r53dStr(in, "Tld"), ".")); tld != "" {
		prices = nil
		for _, p := range r53dPrices {
			if p.tld == tld {
				prices = append(prices, p)
			}
		}
		if prices == nil {
			return nil, r53dErr("UnsupportedTLD", "Route 53 doesn't support the top-level domain .%s", tld)
		}
	}
	start, end, err := r53dPage(in, len(prices), 1000, 1000)
	if err != nil {
		return nil, err
	}
	list := []any{}
	for _, p := range prices[start:end] {
		list = append(list, r53dDomainPrice(p))
	}
	out := map[string]any{"Prices": list}
	r53dNext(out, end, len(prices))
	return out, nil
}

func (st *route53DomainsStore) suggestions(_ *Server, in map[string]any) (map[string]any, *r53dError) {
	label, _, err := r53dSplit(r53dStr(in, "DomainName"))
	if err != nil {
		return nil, err
	}
	count, _ := in["SuggestionCount"].(float64)
	if count < 1 || count > 50 {
		return nil, r53dInvalid("SuggestionCount must be between 1 and 50")
	}
	onlyAvailable, _ := r53dBool(in, "OnlyAvailable")
	list := []any{}
	for _, prefix := range []string{"", "get", "my", "the", "try"} {
		for _, p := range r53dPrices {
			if len(list) >= int(count) {
				break
			}
			name := prefix + label + "." + p.tld
			avail, _, aerr := st.availabilityLocked(name)
			if aerr != nil || (onlyAvailable && avail != "AVAILABLE") {
				continue
			}
			list = append(list, map[string]any{"DomainName": name, "Availability": avail, "Price": r53dPriceOf(p.register)})
		}
	}
	return map[string]any{"SuggestionsList": list}, nil
}

// ---- register, transfer in, renew, delete, restore ----

func (st *route53DomainsStore) register(s *Server, in map[string]any) (map[string]any, *r53dError) {
	avail, p, err := st.availabilityLocked(r53dStr(in, "DomainName"))
	if err != nil {
		return nil, err
	}
	name := strings.TrimSuffix(strings.ToLower(r53dStr(in, "DomainName")), ".")
	if avail != "AVAILABLE" {
		return nil, r53dInvalid("domain %s is not available (%s)", name, avail)
	}
	years, err := r53dYears(in, p, 1)
	if err != nil {
		return nil, err
	}
	if err := r53dCheckConsent(in, p.register*float64(years)); err != nil {
		return nil, err
	}
	if _, err := st.createDomainLocked(s, name, in, years, nil); err != nil {
		return nil, err
	}
	id, err := st.newOpLocked(name, "REGISTER_DOMAIN", "")
	if err != nil {
		return nil, err
	}
	st.billLocked(name, "REGISTER_DOMAIN", p.register*float64(years))
	return r53dOpOut(id), nil
}

func (st *route53DomainsStore) transferIn(s *Server, in map[string]any) (map[string]any, *r53dError) {
	label, p, err := r53dSplit(r53dStr(in, "DomainName"))
	if err != nil {
		return nil, err
	}
	name := label + "." + p.tld
	switch {
	case st.domains[name] != nil:
		return nil, r53dErr("DuplicateRequest", "domain %s is already registered with Route 53 in this account", name)
	case r53dReserved[label] || p.transfer == 0:
		return nil, r53dErr("TLDRulesViolation", "domain %s can't be transferred to Route 53", name)
	}
	years, err := r53dYears(in, p, 1)
	if err != nil {
		return nil, err
	}
	if err := r53dCheckConsent(in, p.transfer); err != nil {
		return nil, err
	}
	ns, _ := in["Nameservers"].([]any)
	if _, err := st.createDomainLocked(s, name, in, years, ns); err != nil {
		return nil, err
	}
	id, err := st.newOpLocked(name, "TRANSFER_IN_DOMAIN", "")
	if err != nil {
		return nil, err
	}
	st.billLocked(name, "TRANSFER_IN_DOMAIN", p.transfer)
	return r53dOpOut(id), nil
}

func (st *route53DomainsStore) renew(_ *Server, in map[string]any) (map[string]any, *r53dError) {
	d, err := st.domainLocked(in)
	if err != nil {
		return nil, err
	}
	_, p, _ := r53dSplit(d.name)
	if year, _ := in["CurrentExpiryYear"].(float64); int(year) != d.expiry.Year() {
		return nil, r53dInvalid("CurrentExpiryYear %d doesn't match the domain's expiry year %d", int(year), d.expiry.Year())
	}
	years, err := r53dYears(in, p, 1)
	if err != nil {
		return nil, err
	}
	if d.expiry.AddDate(years, 0, 0).After(time.Now().AddDate(p.maxYears, 0, 0)) {
		return nil, r53dErr("TLDRulesViolation", ".%s domains can't be registered more than %d years ahead", p.tld, p.maxYears)
	}
	if err := r53dCheckConsent(in, p.renew*float64(years)); err != nil {
		return nil, err
	}
	id, err := st.newOpLocked(d.name, "RENEW_DOMAIN", "")
	if err != nil {
		return nil, err
	}
	d.expiry = d.expiry.AddDate(years, 0, 0)
	d.updated = time.Now().UTC()
	st.billLocked(d.name, "RENEW_DOMAIN", p.renew*float64(years))
	return r53dOpOut(id), nil
}

func (st *route53DomainsStore) deleteDomain(_ *Server, in map[string]any) (map[string]any, *r53dError) {
	d, err := st.domainLocked(in)
	if err != nil {
		return nil, err
	}
	id, err := st.newOpLocked(d.name, "DELETE_DOMAIN", "")
	if err != nil {
		return nil, err
	}
	delete(st.domains, d.name)
	d.deleted = time.Now().UTC()
	st.deleted[d.name] = d
	return r53dOpOut(id), nil
}

func (st *route53DomainsStore) restore(_ *Server, in map[string]any) (map[string]any, *r53dError) {
	name := strings.TrimSuffix(strings.ToLower(r53dStr(in, "DomainName")), ".")
	d := st.deleted[name]
	if d == nil {
		return nil, r53dInvalid("domain %s isn't in its redemption period", name)
	}
	_, p, _ := r53dSplit(name)
	price := max(p.restore, p.renew)
	if err := r53dCheckConsent(in, price); err != nil {
		return nil, err
	}
	if len(st.domains) >= r53dMaxDomains {
		return nil, r53dErr("DomainLimitExceeded", "an account can hold at most %d domains here", r53dMaxDomains)
	}
	id, err := st.newOpLocked(name, "RESTORE_DOMAIN", "")
	if err != nil {
		return nil, err
	}
	delete(st.deleted, name)
	d.deleted = time.Time{}
	d.updated = time.Now().UTC()
	st.domains[name] = d
	st.billLocked(name, "RESTORE_DOMAIN", price)
	return r53dOpOut(id), nil
}

// ---- detail and listing ----

func (d *r53dDomain) statusList() []any {
	out := []any{"clientDeleteProhibited", "clientUpdateProhibited"}
	if d.transferLock {
		out = append(out, "clientTransferProhibited")
	}
	return out
}

func (st *route53DomainsStore) detail(_ *Server, in map[string]any) (map[string]any, *r53dError) {
	d, err := st.domainLocked(in)
	if err != nil {
		return nil, err
	}
	keys := []any{}
	for _, k := range d.keys {
		keys = append(keys, map[string]any{"Id": k.id, "Algorithm": k.algorithm, "Flags": k.flags, "PublicKey": k.publicKey, "DigestType": 2, "Digest": k.digest, "KeyTag": k.tag})
	}
	dnssec := "DISABLED"
	if len(d.keys) > 0 {
		dnssec = "ENABLED"
	}
	out := map[string]any{
		"DomainName": d.name, "Nameservers": d.nameservers, "AutoRenew": d.autoRenew,
		"AdminPrivacy": d.privacy["Admin"], "RegistrantPrivacy": d.privacy["Registrant"], "TechPrivacy": d.privacy["Tech"], "BillingPrivacy": d.privacy["Billing"],
		"RegistrarName": "Amazon Registrar, Inc.", "WhoIsServer": "whois.registrar.amazon.com", "RegistrarUrl": "https://registrar.amazon.com",
		"AbuseContactEmail": "abuse@registrar.amazon.com", "AbuseContactPhone": "+1.2062661000", "RegistryDomainId": "",
		"CreationDate": r53dEpoch(d.created), "UpdatedDate": r53dEpoch(d.updated), "ExpirationDate": r53dEpoch(d.expiry),
		"Reseller": "Amazon", "DnsSec": dnssec, "StatusList": d.statusList(), "DnssecKeys": keys,
	}
	for kind, c := range d.contacts {
		if c != nil {
			out[kind+"Contact"] = c
		}
	}
	return out, nil
}

func r53dMatches(d *r53dDomain, filters []any) bool {
	for _, f := range filters {
		fc, _ := f.(map[string]any)
		values, _ := fc["Values"].([]any)
		if len(values) == 0 {
			return false
		}
		v, _ := values[0].(string)
		switch r53dStr(fc, "Name") + "/" + r53dStr(fc, "Operator") {
		case "DomainName/BEGINS_WITH":
			if !strings.HasPrefix(d.name, strings.ToLower(v)) {
				return false
			}
		case "DomainName/LE":
			if d.name > strings.ToLower(v) {
				return false
			}
		case "DomainName/GE":
			if d.name < strings.ToLower(v) {
				return false
			}
		case "Expiry/LE", "Expiry/GE":
			t, ok := r53dTime(v)
			if !ok {
				return false
			}
			if r53dStr(fc, "Operator") == "LE" && d.expiry.After(t) || r53dStr(fc, "Operator") == "GE" && d.expiry.Before(t) {
				return false
			}
		default:
			return false
		}
	}
	return true
}

func (st *route53DomainsStore) listDomains(_ *Server, in map[string]any) (map[string]any, *r53dError) {
	filters, _ := in["FilterConditions"].([]any)
	for _, f := range filters {
		fc, _ := f.(map[string]any)
		if r53dStr(fc, "Name") == "DomainName" || r53dStr(fc, "Operator") != "BEGINS_WITH" {
			continue
		}
		return nil, r53dInvalid("BEGINS_WITH applies only to DomainName")
	}
	var list []*r53dDomain
	for _, name := range r53dSortedKeys(st.domains) {
		if d := st.domains[name]; r53dMatches(d, filters) {
			list = append(list, d)
		}
	}
	if sc, ok := in["SortCondition"].(map[string]any); ok {
		desc := r53dStr(sc, "SortOrder") == "DESC"
		byExpiry := r53dStr(sc, "Name") == "Expiry"
		sort.SliceStable(list, func(i, j int) bool {
			less := list[i].name < list[j].name
			if byExpiry {
				less = list[i].expiry.Before(list[j].expiry)
			}
			return less != desc
		})
	}
	start, end, err := r53dPage(in, len(list), r53dDefaultPage, 1000)
	if err != nil {
		return nil, err
	}
	domains := []any{}
	for _, d := range list[start:end] {
		domains = append(domains, map[string]any{"DomainName": d.name, "AutoRenew": d.autoRenew, "TransferLock": d.transferLock, "Expiry": r53dEpoch(d.expiry)})
	}
	out := map[string]any{"Domains": domains}
	r53dNext(out, end, len(list))
	return out, nil
}

// ---- updates ----

func (st *route53DomainsStore) updateContact(_ *Server, in map[string]any) (map[string]any, *r53dError) {
	d, err := st.domainLocked(in)
	if err != nil {
		return nil, err
	}
	changed := map[string]map[string]any{}
	for _, kind := range []string{"Admin", "Registrant", "Tech", "Billing"} {
		if c, ok := in[kind+"Contact"].(map[string]any); ok {
			changed[kind] = c
		}
	}
	if len(changed) == 0 {
		return nil, r53dInvalid("specify at least one contact to update")
	}
	typ := "UPDATE_DOMAIN_CONTACT"
	if _, ok := changed["Registrant"]; ok {
		typ = "CHANGE_DOMAIN_OWNER"
	}
	id, err := st.newOpLocked(d.name, typ, "")
	if err != nil {
		return nil, err
	}
	for kind, c := range changed {
		d.contacts[kind] = c
	}
	d.updated = time.Now().UTC()
	return r53dOpOut(id), nil
}

func (st *route53DomainsStore) updatePrivacy(_ *Server, in map[string]any) (map[string]any, *r53dError) {
	d, err := st.domainLocked(in)
	if err != nil {
		return nil, err
	}
	set := false
	for _, kind := range []string{"Admin", "Registrant", "Tech", "Billing"} {
		if v, ok := r53dBool(in, kind+"Privacy"); ok {
			d.privacy[kind] = v
			set = true
		}
	}
	if !set {
		return nil, r53dInvalid("specify at least one privacy setting to update")
	}
	d.updated = time.Now().UTC()
	id, err := st.newOpLocked(d.name, "CHANGE_PRIVACY_PROTECTION", "")
	if err != nil {
		return nil, err
	}
	return r53dOpOut(id), nil
}

func (st *route53DomainsStore) updateNameservers(_ *Server, in map[string]any) (map[string]any, *r53dError) {
	d, err := st.domainLocked(in)
	if err != nil {
		return nil, err
	}
	ns, _ := in["Nameservers"].([]any)
	seen := map[string]bool{}
	for _, n := range ns {
		name := strings.ToLower(r53dStr(n.(map[string]any), "Name"))
		if seen[name] {
			return nil, r53dInvalid("name server %s is listed twice", name)
		}
		seen[name] = true
		if glue, _ := n.(map[string]any)["GlueIps"].([]any); len(glue) > 0 && !strings.HasSuffix(name, "."+d.name) {
			return nil, r53dInvalid("glue IPs are only allowed for name servers inside %s", d.name)
		}
	}
	id, err := st.newOpLocked(d.name, "UPDATE_NAMESERVER", "")
	if err != nil {
		return nil, err
	}
	d.nameservers = ns
	d.updated = time.Now().UTC()
	return r53dOpOut(id), nil
}

func (st *route53DomainsStore) setAutoRenew(in map[string]any, on bool) (map[string]any, *r53dError) {
	d, err := st.domainLocked(in)
	if err != nil {
		return nil, err
	}
	d.autoRenew = on
	d.updated = time.Now().UTC()
	return map[string]any{}, nil
}

func (st *route53DomainsStore) setTransferLock(in map[string]any, on bool) (map[string]any, *r53dError) {
	d, err := st.domainLocked(in)
	if err != nil {
		return nil, err
	}
	id, err := st.newOpLocked(d.name, "DOMAIN_LOCK", "")
	if err != nil {
		return nil, err
	}
	d.transferLock = on
	d.updated = time.Now().UTC()
	return r53dOpOut(id), nil
}

// ---- DNSSEC ----

// r53dKeyTag is the RFC 4034 appendix B key tag over the DNSKEY RDATA.
func r53dKeyTag(rdata []byte) int {
	var ac uint32
	for i, b := range rdata {
		if i&1 == 1 {
			ac += uint32(b)
		} else {
			ac += uint32(b) << 8
		}
	}
	ac += ac >> 16 & 0xffff
	return int(ac & 0xffff)
}

// r53dWireName is a domain name in DNS wire format (lowercase, RFC 4034 §6.2).
func r53dWireName(name string) []byte {
	var out []byte
	for _, label := range strings.Split(strings.ToLower(name), ".") {
		out = append(out, byte(len(label)))
		out = append(out, label...)
	}
	return append(out, 0)
}

func (st *route53DomainsStore) associateDS(_ *Server, in map[string]any) (map[string]any, *r53dError) {
	d, err := st.domainLocked(in)
	if err != nil {
		return nil, err
	}
	attrs, _ := in["SigningAttributes"].(map[string]any)
	alg, okA := attrs["Algorithm"].(float64)
	flags, okF := attrs["Flags"].(float64)
	pub := r53dStr(attrs, "PublicKey")
	if !okA || !okF || pub == "" {
		return nil, r53dInvalid("SigningAttributes needs Algorithm, Flags and PublicKey")
	}
	if flags != 256 && flags != 257 {
		return nil, r53dInvalid("Flags must be 256 (ZSK) or 257 (KSK)")
	}
	key, derr := base64.StdEncoding.DecodeString(pub)
	if derr != nil || len(key) == 0 {
		return nil, r53dInvalid("PublicKey must be base64")
	}
	if len(d.keys) >= r53dMaxDNSSEC {
		return nil, r53dErr("DnssecLimitExceeded", "a domain can have at most %d DNSSEC keys here", r53dMaxDNSSEC)
	}
	rdata := append([]byte{byte(int(flags) >> 8), byte(int(flags)), 3, byte(int(alg))}, key...)
	for _, k := range d.keys {
		if k.publicKey == pub {
			return nil, r53dErr("DuplicateRequest", "the key is already associated with %s", d.name)
		}
	}
	digest := sha256.Sum256(append(r53dWireName(d.name), rdata...))
	id, err := st.newOpLocked(d.name, "ADD_DNSSEC", "")
	if err != nil {
		return nil, err
	}
	d.keys = append(d.keys, r53dKey{id: r53dRandom(8), publicKey: pub, digest: strings.ToUpper(hex.EncodeToString(digest[:])), algorithm: int(alg), flags: int(flags), tag: r53dKeyTag(rdata)})
	d.updated = time.Now().UTC()
	return r53dOpOut(id), nil
}

func (st *route53DomainsStore) disassociateDS(_ *Server, in map[string]any) (map[string]any, *r53dError) {
	d, err := st.domainLocked(in)
	if err != nil {
		return nil, err
	}
	for i, k := range d.keys {
		if k.id == r53dStr(in, "Id") {
			id, err := st.newOpLocked(d.name, "REMOVE_DNSSEC", "")
			if err != nil {
				return nil, err
			}
			d.keys = append(d.keys[:i:i], d.keys[i+1:]...)
			d.updated = time.Now().UTC()
			return r53dOpOut(id), nil
		}
	}
	return nil, r53dInvalid("domain %s has no DNSSEC key %s", d.name, r53dStr(in, "Id"))
}

// ---- transfers between accounts, push, auth code ----

func (st *route53DomainsStore) transferOut(_ *Server, in map[string]any) (map[string]any, *r53dError) {
	d, err := st.domainLocked(in)
	if err != nil {
		return nil, err
	}
	account := r53dStr(in, "AccountId")
	switch {
	case account == r53dAccountID:
		return nil, r53dInvalid("the domain is already in account %s", account)
	case d.transferTo != "":
		return nil, r53dErr("DuplicateRequest", "a transfer of %s to account %s is already pending", d.name, d.transferTo)
	}
	id, err := st.newOpLocked(d.name, "INTERNAL_TRANSFER_OUT_DOMAIN", "PENDING_ACCEPTANCE")
	if err != nil {
		return nil, err
	}
	d.transferTo, d.transferPassword = account, r53dRandom(12)
	return map[string]any{"OperationId": id, "Password": d.transferPassword}, nil
}

// settleTransferOpLocked finishes the domain's pending transfer-out operation.
func (st *route53DomainsStore) settleTransferOpLocked(domain, status string) {
	for _, id := range st.opOrder {
		if op := st.ops[id]; op != nil && op.domain == domain && op.typ == "INTERNAL_TRANSFER_OUT_DOMAIN" && op.flag != "" {
			op.flag, op.status, op.updated = "", status, time.Now().UTC()
		}
	}
}

// acceptTransfer is the receiving account accepting; with one account here
// the domain leaves this store, as it leaves the sending account in AWS.
func (st *route53DomainsStore) acceptTransfer(_ *Server, in map[string]any) (map[string]any, *r53dError) {
	d, err := st.domainLocked(in)
	if err != nil {
		return nil, err
	}
	if d.transferTo == "" || r53dStr(in, "Password") != d.transferPassword {
		return nil, r53dInvalid("no pending transfer of %s matches that password", d.name)
	}
	id, err := st.newOpLocked(d.name, "INTERNAL_TRANSFER_IN_DOMAIN", "")
	if err != nil {
		return nil, err
	}
	st.settleTransferOpLocked(d.name, "SUCCESSFUL")
	delete(st.domains, d.name)
	return r53dOpOut(id), nil
}

func (st *route53DomainsStore) endTransfer(_ *Server, in map[string]any) (map[string]any, *r53dError) {
	d, err := st.domainLocked(in)
	if err != nil {
		return nil, err
	}
	if d.transferTo == "" {
		return nil, r53dInvalid("domain %s has no pending transfer to another account", d.name)
	}
	st.settleTransferOpLocked(d.name, "FAILED")
	d.transferTo, d.transferPassword = "", ""
	id, err := st.newOpLocked(d.name, "INTERNAL_TRANSFER_OUT_DOMAIN", "")
	if err != nil {
		return nil, err
	}
	return r53dOpOut(id), nil
}

func (st *route53DomainsStore) authCode(_ *Server, in map[string]any) (map[string]any, *r53dError) {
	d, err := st.domainLocked(in)
	if err != nil {
		return nil, err
	}
	return map[string]any{"AuthCode": d.authCode}, nil
}

// push hands a .uk domain to another registrar (identified by its IPS tag):
// it leaves Route 53.
func (st *route53DomainsStore) push(_ *Server, in map[string]any) (map[string]any, *r53dError) {
	d, err := st.domainLocked(in)
	if err != nil {
		return nil, err
	}
	if !strings.HasSuffix(d.name, ".uk") {
		return nil, r53dErr("TLDRulesViolation", "only .uk domains can be pushed; %s isn't one", d.name)
	}
	if _, err := st.newOpLocked(d.name, "PUSH_DOMAIN", ""); err != nil {
		return nil, err
	}
	delete(st.domains, d.name)
	return map[string]any{}, nil
}

// ---- operations, reachability, billing, tags ----

func (st *route53DomainsStore) operationDetail(_ *Server, in map[string]any) (map[string]any, *r53dError) {
	op := st.ops[r53dStr(in, "OperationId")]
	if op == nil {
		return nil, r53dInvalid("operation %s doesn't exist", r53dStr(in, "OperationId"))
	}
	op.read()
	return op.summary(), nil
}

func (op *r53dOperation) summary() map[string]any {
	out := map[string]any{"OperationId": op.id, "Status": op.status, "DomainName": op.domain, "Type": op.typ,
		"SubmittedDate": r53dEpoch(op.submitted), "LastUpdatedDate": r53dEpoch(op.updated)}
	if op.flag != "" {
		out["StatusFlag"] = op.flag
	}
	return out
}

func r53dIn(list []any, v string) bool {
	if len(list) == 0 {
		return true
	}
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

func (st *route53DomainsStore) listOperations(_ *Server, in map[string]any) (map[string]any, *r53dError) {
	since, hasSince := r53dTime(in["SubmittedSince"])
	statuses, _ := in["Status"].([]any)
	types, _ := in["Type"].([]any)
	var list []*r53dOperation
	for _, id := range st.opOrder {
		op := st.ops[id]
		if (!hasSince || !op.submitted.Before(since)) && r53dIn(statuses, op.status) && r53dIn(types, op.typ) {
			list = append(list, op)
		}
	}
	if r53dStr(in, "SortOrder") == "DESC" {
		for i, j := 0, len(list)-1; i < j; i, j = i+1, j-1 {
			list[i], list[j] = list[j], list[i]
		}
	}
	start, end, err := r53dPage(in, len(list), r53dDefaultPage, 1000)
	if err != nil {
		return nil, err
	}
	ops := []any{}
	for _, op := range list[start:end] {
		ops = append(ops, op.summary())
	}
	out := map[string]any{"Operations": ops}
	r53dNext(out, end, len(list))
	return out, nil
}

func (st *route53DomainsStore) resendAuthorization(_ *Server, in map[string]any) (map[string]any, *r53dError) {
	if st.ops[r53dStr(in, "OperationId")] == nil {
		return nil, r53dInvalid("operation %s doesn't exist", r53dStr(in, "OperationId"))
	}
	return map[string]any{}, nil
}

func (st *route53DomainsStore) reachabilityDomainLocked(in map[string]any) (*r53dDomain, *r53dError) {
	return st.domainLocked(map[string]any{"DomainName": r53dStr(in, "domainName")})
}

func (st *route53DomainsStore) reachability(_ *Server, in map[string]any) (map[string]any, *r53dError) {
	d, err := st.reachabilityDomainLocked(in)
	if err != nil {
		return nil, err
	}
	return map[string]any{"domainName": d.name, "status": "DONE"}, nil
}

func (st *route53DomainsStore) resendReachability(_ *Server, in map[string]any) (map[string]any, *r53dError) {
	d, err := st.reachabilityDomainLocked(in)
	if err != nil {
		return nil, err
	}
	email := r53dStr(d.contacts["Registrant"], "Email")
	return map[string]any{"domainName": d.name, "emailAddress": email, "isAlreadyVerified": true}, nil
}

func (st *route53DomainsStore) viewBilling(_ *Server, in map[string]any) (map[string]any, *r53dError) {
	from, hasFrom := r53dTime(in["Start"])
	to, hasTo := r53dTime(in["End"])
	var list []r53dBilling
	for _, b := range st.billing {
		if (!hasFrom || !b.date.Before(from)) && (!hasTo || !b.date.After(to)) {
			list = append(list, b)
		}
	}
	start, end, err := r53dPage(in, len(list), r53dDefaultPage, 1000)
	if err != nil {
		return nil, err
	}
	records := []any{}
	for _, b := range list[start:end] {
		records = append(records, map[string]any{"DomainName": b.domain, "Operation": b.op, "InvoiceId": b.invoice, "BillDate": r53dEpoch(b.date), "Price": b.price})
	}
	out := map[string]any{"BillingRecords": records}
	r53dNext(out, end, len(list))
	return out, nil
}

func (st *route53DomainsStore) listTags(_ *Server, in map[string]any) (map[string]any, *r53dError) {
	d, err := st.domainLocked(in)
	if err != nil {
		return nil, err
	}
	tags := []any{}
	for _, k := range r53dSortedKeys(d.tags) {
		tags = append(tags, map[string]any{"Key": k, "Value": d.tags[k]})
	}
	return map[string]any{"TagList": tags}, nil
}

func (st *route53DomainsStore) updateTags(_ *Server, in map[string]any) (map[string]any, *r53dError) {
	d, err := st.domainLocked(in)
	if err != nil {
		return nil, err
	}
	list, _ := in["TagsToUpdate"].([]any)
	next := map[string]string{}
	for k, v := range d.tags {
		next[k] = v
	}
	for _, t := range list {
		tag, _ := t.(map[string]any)
		key := r53dStr(tag, "Key")
		if key == "" || strings.HasPrefix(strings.ToLower(key), "aws:") {
			return nil, r53dInvalid("tag key %q is not allowed", key)
		}
		next[key] = r53dStr(tag, "Value")
	}
	if len(next) > r53dMaxTags {
		return nil, r53dInvalid("a domain can have at most %d tags", r53dMaxTags)
	}
	d.tags = next
	return map[string]any{}, nil
}

func (st *route53DomainsStore) deleteTags(_ *Server, in map[string]any) (map[string]any, *r53dError) {
	d, err := st.domainLocked(in)
	if err != nil {
		return nil, err
	}
	keys, _ := in["TagsToDelete"].([]any)
	for _, k := range keys {
		key, _ := k.(string)
		delete(d.tags, key)
	}
	return map[string]any{}, nil
}
