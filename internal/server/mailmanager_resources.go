package server

import (
	"sort"
	"strconv"
	"strings"
	"time"
)

// Mail Manager handlers: add-ons, address lists, archives, ingress points,
// relays, rule sets, traffic policies and tags. Searches, exports, import
// jobs and archived messages are in mailmanager_archive.go.

const (
	mmgrStatusActive   = "ACTIVE"
	mmgrStatusPending  = "PENDING_DELETION"
	mmgrStatusProvsn   = "PROVISIONING"
	mmgrStatusClosed   = "CLOSED"
	mmgrMaxListMembers = 10000
)

var mmgrHandlers map[string]mmgrHandler

func init() {
	mmgrHandlers = map[string]mmgrHandler{
		"CreateAddonSubscription":   mmgrCreate("addonSubscription", (*mmgrStore).prepAddonSubscription),
		"CreateAddonInstance":       mmgrCreate("addonInstance", (*mmgrStore).prepAddonInstance),
		"CreateAddressList":         mmgrCreate("addressList", (*mmgrStore).prepAddressList),
		"CreateArchive":             mmgrCreate("archive", (*mmgrStore).prepArchive),
		"CreateIngressPoint":        mmgrCreate("ingressPoint", (*mmgrStore).prepIngressPoint),
		"CreateRelay":               mmgrCreate("relay", (*mmgrStore).prepRelay),
		"CreateRuleSet":             mmgrCreate("ruleSet", (*mmgrStore).prepRuleSet),
		"CreateTrafficPolicy":       mmgrCreate("trafficPolicy", (*mmgrStore).prepTrafficPolicy),
		"UpdateArchive":             mmgrUpdate("archive", (*mmgrStore).prepArchive),
		"UpdateIngressPoint":        mmgrUpdate("ingressPoint", (*mmgrStore).prepIngressPoint),
		"UpdateRelay":               mmgrUpdate("relay", (*mmgrStore).prepRelay),
		"UpdateRuleSet":             mmgrUpdate("ruleSet", (*mmgrStore).prepRuleSet),
		"UpdateTrafficPolicy":       mmgrUpdate("trafficPolicy", (*mmgrStore).prepTrafficPolicy),
		"ListAddonInstances":        mmgrListOp("addonInstance", ""),
		"ListAddonSubscriptions":    mmgrListOp("addonSubscription", ""),
		"ListAddressLists":          mmgrListOp("addressList", ""),
		"ListArchives":              mmgrListOp("archive", ""),
		"ListIngressPoints":         mmgrListOp("ingressPoint", ""),
		"ListRelays":                mmgrListOp("relay", ""),
		"ListRuleSets":              mmgrListOp("ruleSet", ""),
		"ListTrafficPolicies":       mmgrListOp("trafficPolicy", ""),
		"ListAddressListImportJobs": mmgrListOp("importJob", "addressList:AddressListId"),
		"ListArchiveExports":        mmgrListOp("export", "archive:ArchiveId"),
		"ListArchiveSearches":       mmgrListOp("search", "archive:ArchiveId"),
		"GetIngressPoint": func(s *Server, q mmgrReq) (map[string]any, *mmgrErr) {
			out, err := s.mailmanager.getOne("ingressPoint", mmgrStr(q.in, "IngressPointId"))
			if err == nil && mmgrStr(q.in, "IncludeTrustStoreContents") != "INCLUDE" {
				if ts := mmgrMap(mmgrMap(mmgrMap(out, "IngressPointAuthConfiguration"), "TlsAuthConfiguration"), "TrustStore"); ts != nil {
					delete(ts, "CAContent")
					delete(ts, "CrlContent")
				}
			}
			return out, err
		},
		"DeleteAddonInstance":     mmgrDelete("addonInstance"),
		"DeleteAddonSubscription": mmgrDelete("addonSubscription"),
		"DeleteAddressList":       mmgrDelete("addressList"),
		"DeleteArchive":           mmgrDelete("archive"),
		"DeleteIngressPoint":      mmgrDelete("ingressPoint"),
		"DeleteRelay":             mmgrDelete("relay"),
		"DeleteRuleSet":           mmgrDelete("ruleSet"),
		"DeleteTrafficPolicy":     mmgrDelete("trafficPolicy"),
		"RegisterMemberToAddressList": func(s *Server, q mmgrReq) (map[string]any, *mmgrErr) {
			return nil, s.mailmanager.registerMember(mmgrStr(q.in, "AddressListId"), mmgrStr(q.in, "Address"))
		},
		"DeregisterMemberFromAddressList": func(s *Server, q mmgrReq) (map[string]any, *mmgrErr) {
			return nil, s.mailmanager.deregisterMember(mmgrStr(q.in, "AddressListId"), mmgrStr(q.in, "Address"))
		},
		"GetMemberOfAddressList": func(s *Server, q mmgrReq) (map[string]any, *mmgrErr) {
			return s.mailmanager.getMember(mmgrStr(q.in, "AddressListId"), mmgrStr(q.in, "Address"))
		},
		"ListMembersOfAddressList": func(s *Server, q mmgrReq) (map[string]any, *mmgrErr) {
			return s.mailmanager.listMembers(q.in)
		},
		"TagResource": func(s *Server, q mmgrReq) (map[string]any, *mmgrErr) {
			return nil, s.mailmanager.tagResource(mmgrStr(q.in, "ResourceArn"), q.in)
		},
		"UntagResource": func(s *Server, q mmgrReq) (map[string]any, *mmgrErr) {
			return nil, s.mailmanager.untagResource(mmgrStr(q.in, "ResourceArn"), mmgrStrings(q.in["TagKeys"]))
		},
		"ListTagsForResource": func(s *Server, q mmgrReq) (map[string]any, *mmgrErr) {
			return s.mailmanager.listTags(mmgrStr(q.in, "ResourceArn"))
		},
	}
	// Get* ops that are a plain view of one resource.
	for _, k := range mmgrKinds {
		if _, custom := mmgrHandlers[k.getOp]; custom {
			continue
		}
		k := k
		mmgrHandlers[k.getOp] = func(s *Server, q mmgrReq) (map[string]any, *mmgrErr) {
			return s.mailmanager.getOne(k.id, mmgrStr(q.in, k.idKey))
		}
	}
	for op, h := range mmgrArchiveHandlers {
		mmgrHandlers[op] = h
	}
}

// mmgrPrep checks a create or update request and returns the fields to store.
// existing is nil on create. Callers hold the lock.
type mmgrPrep func(st *mmgrStore, in map[string]any, existing *mmgrRes) (map[string]any, *mmgrErr)

// mmgrControlKeys are request members that aren't stored as resource fields.
var mmgrControlKeys = map[string]bool{"ClientToken": true, "Tags": true}

func mmgrCreate(kind string, prep mmgrPrep) mmgrHandler {
	k := mmgrKinds[kind]
	return func(s *Server, q mmgrReq) (map[string]any, *mmgrErr) {
		st := s.mailmanager
		st.mu.Lock()
		defer st.mu.Unlock()
		if r := st.byToken(kind, mmgrStr(q.in, "ClientToken")); r != nil {
			return map[string]any{k.idKey: r.id}, nil
		}
		if k.nameKey != "" {
			if err := st.nameTaken(kind, mmgrStr(q.in, k.nameKey), ""); err != nil {
				return nil, err
			}
		}
		fields, err := prep(st, q.in, nil)
		if err != nil {
			return nil, err
		}
		r, err := st.add(kind, q.in, fields, time.Now().UTC())
		if err != nil {
			return nil, err
		}
		st.created(r, q.in)
		return map[string]any{k.idKey: r.id}, nil
	}
}

// created finishes kind-specific setup of a new resource.
func (st *mmgrStore) created(r *mmgrRes, in map[string]any) {
	switch r.kind.id {
	case "addonInstance":
		r.parent = mmgrStr(in, "AddonSubscriptionId")
	case "addressList":
		r.members = map[string]time.Time{}
	case "archive":
		r.status = mmgrStatusActive
	case "ingressPoint":
		r.status = mmgrStatusProvsn
		r.fields["ARecord"] = r.id + ".mail-manager-smtp." + mmgrRegion + ".amazonaws.com"
		if mmgrStr(mmgrMap(in, "IngressPointConfiguration"), "SmtpPassword") != "" {
			r.pwVersion = 1
		}
	}
}

func mmgrUpdate(kind string, prep mmgrPrep) mmgrHandler {
	k := mmgrKinds[kind]
	return func(s *Server, q mmgrReq) (map[string]any, *mmgrErr) {
		st := s.mailmanager
		st.mu.Lock()
		defer st.mu.Unlock()
		r, err := st.get(kind, mmgrStr(q.in, k.idKey))
		if err != nil {
			return nil, err
		}
		now := time.Now().UTC()
		st.settle(r, now)
		if kind == "archive" && r.status != mmgrStatusActive {
			return nil, mmgrError("ConflictException", "Archive %s is %s.", r.id, r.status)
		}
		name := mmgrStr(q.in, k.nameKey)
		if name != "" {
			if err := st.nameTaken(kind, name, r.id); err != nil {
				return nil, err
			}
		}
		fields, err := prep(st, q.in, r)
		if err != nil {
			return nil, err
		}
		// Copy-on-write: responses built earlier never see this update.
		next := map[string]any{}
		for key, v := range r.fields {
			next[key] = v
		}
		for key, v := range fields {
			next[key] = v
		}
		r.fields = next
		if name != "" {
			r.name = name
		}
		if kind == "ingressPoint" {
			if mmgrStr(mmgrMap(q.in, "IngressPointConfiguration"), "SmtpPassword") != "" {
				if r.pwVersion > 0 {
					r.prevPwUntil = now.Add(24 * time.Hour)
				}
				r.pwVersion++
			}
			switch mmgrStr(q.in, "StatusToUpdate") {
			case mmgrStatusClosed:
				r.status = mmgrStatusClosed
			case mmgrStatusActive:
				r.status = mmgrStatusActive
			}
		}
		r.updated = now
		return nil, nil
	}
}

// mmgrPick copies the members of in named by keys that are present.
func mmgrPick(in map[string]any, keys ...string) map[string]any {
	out := map[string]any{}
	for _, k := range keys {
		if v, ok := in[k]; ok && !mmgrControlKeys[k] {
			out[k] = v
		}
	}
	return out
}

func (st *mmgrStore) prepAddonSubscription(in map[string]any, _ *mmgrRes) (map[string]any, *mmgrErr) {
	name := mmgrStr(in, "AddonName")
	for _, r := range st.res {
		if r.kind.id == "addonSubscription" && mmgrStr(r.fields, "AddonName") == name {
			return nil, mmgrError("ConflictException", "You are already subscribed to %s.", name)
		}
	}
	return map[string]any{"AddonName": name}, nil
}

func (st *mmgrStore) prepAddonInstance(in map[string]any, _ *mmgrRes) (map[string]any, *mmgrErr) {
	if _, err := st.get("addonSubscription", mmgrStr(in, "AddonSubscriptionId")); err != nil {
		return nil, err
	}
	return map[string]any{}, nil
}

func (st *mmgrStore) prepAddressList(map[string]any, *mmgrRes) (map[string]any, *mmgrErr) {
	return map[string]any{}, nil
}

func (st *mmgrStore) prepArchive(in map[string]any, existing *mmgrRes) (map[string]any, *mmgrErr) {
	out := mmgrPick(in, "Retention")
	if existing == nil {
		out = mmgrPick(in, "Retention", "KmsKeyArn")
		if out["Retention"] == nil {
			out["Retention"] = map[string]any{"RetentionPeriod": "THREE_MONTHS"}
		}
	}
	if ret, ok := out["Retention"].(map[string]any); ok && mmgrStr(ret, "RetentionPeriod") == "" {
		return nil, mmgrInvalid("Retention.RetentionPeriod is required.")
	}
	return out, nil
}

func (st *mmgrStore) prepIngressPoint(in map[string]any, existing *mmgrRes) (map[string]any, *mmgrErr) {
	out := mmgrPick(in, "RuleSetId", "TrafficPolicyId", "TlsPolicy")
	typ := mmgrStr(in, "Type")
	if existing != nil {
		typ = mmgrStr(existing.fields, "Type")
	} else {
		out["Type"] = typ
		out["NetworkConfiguration"] = in["NetworkConfiguration"]
		if out["NetworkConfiguration"] == nil {
			out["NetworkConfiguration"] = map[string]any{"PublicNetworkConfiguration": map[string]any{"IpType": "IPV4"}}
		}
		if out["TlsPolicy"] == nil {
			out["TlsPolicy"] = "OPTIONAL"
		}
	}
	if id := mmgrStr(in, "RuleSetId"); id != "" && st.resolve("ruleSet", id) == nil {
		return nil, mmgrInvalid("Rule set %s doesn't exist.", id)
	}
	if id := mmgrStr(in, "TrafficPolicyId"); id != "" && st.resolve("trafficPolicy", id) == nil {
		return nil, mmgrInvalid("Traffic policy %s doesn't exist.", id)
	}
	cfg, hasCfg := in["IngressPointConfiguration"].(map[string]any)
	if !hasCfg {
		if existing == nil && typ != "OPEN" {
			return nil, mmgrInvalid("A %s ingress point needs an IngressPointConfiguration.", typ)
		}
		return out, nil
	}
	password, secret, tls := mmgrStr(cfg, "SmtpPassword"), mmgrStr(cfg, "SecretArn"), mmgrMap(cfg, "TlsAuthConfiguration")
	set := 0
	for _, present := range []bool{password != "", secret != "", tls != nil} {
		if present {
			set++
		}
	}
	switch {
	case typ == "OPEN":
		return nil, mmgrInvalid("An OPEN ingress point takes no IngressPointConfiguration.")
	case set != 1:
		return nil, mmgrInvalid("Set exactly one of SmtpPassword, SecretArn and TlsAuthConfiguration.")
	case typ == "AUTH" && tls != nil:
		return nil, mmgrInvalid("An AUTH ingress point authenticates with SmtpPassword or SecretArn.")
	case typ == "MTLS" && tls == nil:
		return nil, mmgrInvalid("An MTLS ingress point needs TlsAuthConfiguration.")
	}
	// The SMTP password itself is never stored; only its version is reported.
	auth := map[string]any{}
	if secret != "" {
		auth["SecretArn"] = secret
	}
	if tls != nil {
		auth["TlsAuthConfiguration"] = tls
	}
	out["auth"] = auth
	return out, nil
}

func (st *mmgrStore) prepRelay(in map[string]any, existing *mmgrRes) (map[string]any, *mmgrErr) {
	out := mmgrPick(in, "ServerName", "ServerPort", "Authentication")
	if auth, ok := in["Authentication"].(map[string]any); ok {
		_, noAuth := auth["NoAuthentication"]
		if (mmgrStr(auth, "SecretArn") != "") == noAuth {
			return nil, mmgrInvalid("Set exactly one of Authentication.SecretArn and Authentication.NoAuthentication.")
		}
	}
	return out, nil
}

func (st *mmgrStore) prepRuleSet(in map[string]any, _ *mmgrRes) (map[string]any, *mmgrErr) {
	out := mmgrPick(in, "Rules")
	for _, item := range mmgrList(in, "Rules") {
		rule, _ := item.(map[string]any)
		for _, a := range mmgrList(rule, "Actions") {
			if action, _ := a.(map[string]any); len(action) != 1 {
				return nil, mmgrInvalid("Each rule action sets exactly one action type.")
			}
		}
	}
	return out, st.checkRefs(out)
}

func (st *mmgrStore) prepTrafficPolicy(in map[string]any, _ *mmgrRes) (map[string]any, *mmgrErr) {
	out := mmgrPick(in, "PolicyStatements", "DefaultAction", "MaxMessageSizeBytes")
	return out, st.checkRefs(out)
}

// viewKind adds what a kind reports beyond its stored fields.
func (st *mmgrStore) viewKind(r *mmgrRes, out map[string]any) {
	switch r.kind.id {
	case "addonInstance":
		out["AddonSubscriptionId"] = r.parent
		if sub := st.res[r.parent]; sub != nil {
			out["AddonName"] = sub.fields["AddonName"]
		}
	case "archive":
		out["ArchiveState"] = r.status
	case "ingressPoint":
		out["Status"] = r.status
		auth := map[string]any{}
		if stored, ok := r.fields["auth"].(map[string]any); ok {
			for k, v := range stored {
				auth[k] = v
			}
		}
		if r.pwVersion > 0 {
			pw := map[string]any{"SmtpPasswordVersion": mmgrVersion(r.pwVersion)}
			if r.pwVersion > 1 {
				pw["PreviousSmtpPasswordVersion"] = mmgrVersion(r.pwVersion - 1)
				pw["PreviousSmtpPasswordExpiryTimestamp"] = mmgrEpoch(r.prevPwUntil)
			}
			auth["IngressPointPasswordConfiguration"] = pw
		}
		// Conform deep-copies, so callers may trim the trust store.
		out["IngressPointAuthConfiguration"] = st.model.Conform("IngressPointAuthConfiguration", auth)
	default:
		st.viewJob(r, out)
	}
}

func mmgrVersion(n int) string { return "v" + strconv.Itoa(n) }

func mmgrListOp(kind, parent string) mmgrHandler {
	parentKind, parentKey, _ := strings.Cut(parent, ":")
	return func(s *Server, q mmgrReq) (map[string]any, *mmgrErr) {
		st := s.mailmanager
		id := ""
		if parentKey != "" {
			st.mu.Lock()
			p, err := st.get(parentKind, mmgrStr(q.in, parentKey))
			st.mu.Unlock()
			if err != nil {
				return nil, err
			}
			id = p.id
		}
		return st.list(kind, id, q.in)
	}
}

func mmgrDelete(kind string) mmgrHandler {
	k := mmgrKinds[kind]
	return func(s *Server, q mmgrReq) (map[string]any, *mmgrErr) {
		st := s.mailmanager
		st.mu.Lock()
		defer st.mu.Unlock()
		r, err := st.get(kind, mmgrStr(q.in, k.idKey))
		if err != nil {
			return nil, err
		}
		if ref := st.referencedBy(r); ref != "" {
			return nil, mmgrError("ConflictException", "%s is in use by %s.", r.id, ref)
		}
		switch kind {
		case "addonSubscription":
			for _, other := range st.res {
				if other.kind.id == "addonInstance" && other.parent == r.id {
					return nil, mmgrError("ConflictException", "Subscription %s still has add-on instance %s.", r.id, other.id)
				}
			}
		case "addressList":
			st.memberCount -= len(r.members)
			for id, other := range st.res {
				if other.kind.id == "importJob" && other.parent == r.id {
					st.uploadBytes -= len(other.upload)
					delete(st.res, id)
				}
			}
		case "archive":
			// Deletion is asynchronous: the archive stays PENDING_DELETION and
			// its messages become inaccessible at once.
			if r.status == mmgrStatusPending {
				return nil, mmgrError("ConflictException", "Archive %s is already %s.", r.id, mmgrStatusPending)
			}
			r.status, r.updated = mmgrStatusPending, time.Now().UTC()
			st.dropMessages(r.id)
			return nil, nil
		}
		delete(st.res, r.id)
		return nil, nil
	}
}

// ---- address list members ----

func mmgrAddress(a string) string { return strings.ToLower(strings.TrimSpace(a)) }

func mmgrValidAddress(a string) bool {
	local, domain, ok := strings.Cut(a, "@")
	return ok && local != "" && domain != "" && !strings.ContainsAny(a, " \t\r\n,;<>")
}

func (st *mmgrStore) registerMember(listID, address string) *mmgrErr {
	st.mu.Lock()
	defer st.mu.Unlock()
	r, err := st.get("addressList", listID)
	if err != nil {
		return err
	}
	address = mmgrAddress(address)
	if !mmgrValidAddress(address) {
		return mmgrInvalid("%s isn't an email address.", address)
	}
	if _, ok := r.members[address]; ok {
		return mmgrError("ConflictException", "%s is already a member of %s.", address, r.id)
	}
	if len(r.members) >= mmgrMaxListMembers || st.memberCount >= mmgrMaxMembers {
		return mmgrError("ServiceQuotaExceededException", "Stackyard keeps at most %d members per address list and %d in total.", mmgrMaxListMembers, mmgrMaxMembers)
	}
	now := time.Now().UTC()
	r.members[address] = now
	st.memberCount++
	r.updated = now
	return nil
}

func (st *mmgrStore) deregisterMember(listID, address string) *mmgrErr {
	st.mu.Lock()
	defer st.mu.Unlock()
	r, err := st.get("addressList", listID)
	if err != nil {
		return err
	}
	address = mmgrAddress(address)
	if _, ok := r.members[address]; !ok {
		return mmgrError("ResourceNotFoundException", "%s isn't a member of %s.", address, r.id)
	}
	delete(r.members, address)
	st.memberCount--
	r.updated = time.Now().UTC()
	return nil
}

func (st *mmgrStore) getMember(listID, address string) (map[string]any, *mmgrErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	r, err := st.get("addressList", listID)
	if err != nil {
		return nil, err
	}
	address = mmgrAddress(address)
	at, ok := r.members[address]
	if !ok {
		return nil, mmgrError("ResourceNotFoundException", "%s isn't a member of %s.", address, r.id)
	}
	return map[string]any{"Address": address, "CreatedTimestamp": mmgrEpoch(at)}, nil
}

func (st *mmgrStore) listMembers(in map[string]any) (map[string]any, *mmgrErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	r, err := st.get("addressList", mmgrStr(in, "AddressListId"))
	if err != nil {
		return nil, err
	}
	prefix := mmgrAddress(mmgrStr(mmgrMap(in, "Filter"), "AddressPrefix"))
	var addrs []string
	for a := range r.members {
		if strings.HasPrefix(a, prefix) {
			addrs = append(addrs, a)
		}
	}
	sort.Strings(addrs)
	start, end, next, ok := mmgrPage(len(addrs), mmgrStr(in, "NextToken"), mmgrInt(in, "PageSize", 0), mmgrDefaultPage)
	if !ok {
		return nil, mmgrInvalid("NextToken is not valid.")
	}
	items := make([]any, 0, end-start)
	for _, a := range addrs[start:end] {
		items = append(items, map[string]any{"Address": a, "CreatedTimestamp": mmgrEpoch(r.members[a])})
	}
	out := map[string]any{"Addresses": items}
	if next != "" {
		out["NextToken"] = next
	}
	return out, nil
}

// ---- tags ----

func (st *mmgrStore) byARN(arn string) (*mmgrRes, *mmgrErr) {
	if i := strings.LastIndex(arn, "/"); i >= 0 {
		if r := st.res[arn[i+1:]]; r != nil && r.arn == arn {
			return r, nil
		}
	}
	return nil, mmgrError("ResourceNotFoundException", "Resource %s was not found.", arn)
}

func (st *mmgrStore) tagResource(arn string, in map[string]any) *mmgrErr {
	tags, err := mmgrTags(in)
	if err != nil {
		return err
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	r, err := st.byARN(arn)
	if err != nil {
		return err
	}
	next := map[string]string{}
	for k, v := range r.tags {
		next[k] = v
	}
	for k, v := range tags {
		next[k] = v
	}
	if len(next) > mmgrMaxTags {
		return mmgrError("ServiceQuotaExceededException", "A resource can have at most %d tags.", mmgrMaxTags)
	}
	r.tags = next
	return nil
}

func (st *mmgrStore) untagResource(arn string, keys []string) *mmgrErr {
	st.mu.Lock()
	defer st.mu.Unlock()
	r, err := st.byARN(arn)
	if err != nil {
		return err
	}
	next := map[string]string{}
	for k, v := range r.tags {
		next[k] = v
	}
	for _, k := range keys {
		delete(next, k)
	}
	r.tags = next
	return nil
}

func (st *mmgrStore) listTags(arn string) (map[string]any, *mmgrErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	r, err := st.byARN(arn)
	if err != nil {
		return nil, err
	}
	keys := make([]string, 0, len(r.tags))
	for k := range r.tags {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	tags := make([]any, 0, len(keys))
	for _, k := range keys {
		tags = append(tags, map[string]any{"Key": k, "Value": r.tags[k]})
	}
	return map[string]any{"Tags": tags}, nil
}
