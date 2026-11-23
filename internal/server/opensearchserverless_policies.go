package server

import (
	"encoding/json"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Security (encryption / network), data access and lifecycle (retention)
// policies, security configs and VPC endpoints for OpenSearch Serverless.

type aossPolicy struct {
	kind, ptype, name, description, version string
	raw                                     string // policy JSON as sent
	doc                                     any    // parsed policy; replaced on update, never mutated
	created, modified                       int64
}

// aossPolicyFields maps a policy kind to its response member names.
var aossPolicyFields = map[string]struct{ detail, summaries string }{
	"security":  {"securityPolicyDetail", "securityPolicySummaries"},
	"access":    {"accessPolicyDetail", "accessPolicySummaries"},
	"lifecycle": {"lifecyclePolicyDetail", "lifecyclePolicySummaries"},
}

func aossPolicyKey(kind, ptype, name string) string { return kind + "/" + ptype + "/" + name }

func (p *aossPolicy) detail(withDoc bool) map[string]any {
	out := map[string]any{
		"type": p.ptype, "name": p.name, "policyVersion": p.version, "description": p.description,
		"createdDate": p.created, "lastModifiedDate": p.modified,
	}
	if withDoc {
		out["policy"] = p.doc
	}
	return out
}

var aossRetention = regexp.MustCompile(`^[1-9][0-9]*[dh]$`)

// aossParsePolicy validates a policy document for its type and returns it parsed.
func aossParsePolicy(ptype, raw string) (any, *aossErr) {
	var doc any
	if err := json.Unmarshal([]byte(raw), &doc); err != nil {
		return nil, aossValidation("Policy json is invalid: %v", err)
	}
	bad := func(format string, args ...any) (any, *aossErr) {
		return nil, aossValidation("Policy json is invalid: "+format, args...)
	}
	switch ptype {
	case "encryption":
		obj, ok := doc.(map[string]any)
		if !ok {
			return bad("an encryption policy is a JSON object")
		}
		if err := aossCheckRules(obj, map[string]string{"collection": "collection/"}, false); err != "" {
			return bad("%s", err)
		}
		owned, _ := obj["AWSOwnedKey"].(bool)
		kms := aossStr(obj, "KmsARN")
		if owned == (kms != "") {
			return bad("set exactly one of AWSOwnedKey: true or KmsARN")
		}
	case "network", "data":
		items, ok := doc.([]any)
		if !ok || len(items) == 0 {
			return bad("a %s policy is a non-empty JSON array of statements", ptype)
		}
		for i, item := range items {
			obj, ok := item.(map[string]any)
			if !ok {
				return bad("statement %d is not an object", i)
			}
			if ptype == "network" {
				if err := aossCheckRules(obj, map[string]string{"collection": "collection/", "dashboard": "collection/"}, false); err != "" {
					return bad("statement %d: %s", i, err)
				}
				public, hasPublic := obj["AllowFromPublic"].(bool)
				if !hasPublic && len(aossList(obj, "SourceVPCEs")) == 0 && len(aossList(obj, "SourceServices")) == 0 {
					return bad("statement %d needs AllowFromPublic, SourceVPCEs or SourceServices", i)
				}
				if public && len(aossList(obj, "SourceVPCEs")) > 0 {
					return bad("statement %d can't combine AllowFromPublic with SourceVPCEs", i)
				}
				continue
			}
			if err := aossCheckRules(obj, map[string]string{"collection": "collection/", "index": "index/", "model": "model/"}, true); err != "" {
				return bad("statement %d: %s", i, err)
			}
			principals := aossStrings(obj, "Principal")
			if len(principals) == 0 {
				return bad("statement %d needs at least one Principal", i)
			}
		}
	case "retention":
		obj, ok := doc.(map[string]any)
		if !ok {
			return bad("a retention policy is a JSON object")
		}
		if err := aossCheckRules(obj, map[string]string{"index": "index/"}, false); err != "" {
			return bad("%s", err)
		}
		for i, r := range aossList(obj, "Rules") {
			rule := r.(map[string]any)
			minRet := aossStr(rule, "MinIndexRetention")
			noMin, _ := rule["NoMinIndexRetention"].(bool)
			if (minRet != "") == noMin {
				return bad("rule %d needs exactly one of MinIndexRetention or NoMinIndexRetention: true", i)
			}
			if minRet != "" && !aossRetention.MatchString(minRet) {
				return bad("rule %d MinIndexRetention must look like 15d or 24h", i)
			}
		}
	}
	return doc, nil
}

// aossCheckRules checks obj.Rules: each rule's ResourceType must be one of
// types and every Resource must start with that type's prefix.
func aossCheckRules(obj map[string]any, types map[string]string, permissions bool) string {
	rules := aossList(obj, "Rules")
	if len(rules) == 0 {
		return "Rules must list at least one rule"
	}
	for i, r := range rules {
		rule, ok := r.(map[string]any)
		if !ok {
			return "each rule must be an object"
		}
		prefix, ok := types[aossStr(rule, "ResourceType")]
		if !ok {
			return "rule " + strconv.Itoa(i) + " has an unsupported ResourceType"
		}
		resources := aossStrings(rule, "Resource")
		if len(resources) == 0 {
			return "rule " + strconv.Itoa(i) + " needs at least one Resource"
		}
		for _, res := range resources {
			if !strings.HasPrefix(res, prefix) || len(res) == len(prefix) {
				return "rule " + strconv.Itoa(i) + " Resource " + res + " must start with " + prefix
			}
		}
		if permissions {
			perms := aossStrings(rule, "Permission")
			if len(perms) == 0 {
				return "rule " + strconv.Itoa(i) + " needs at least one Permission"
			}
			for _, p := range perms {
				if !strings.HasPrefix(p, "aoss:") {
					return "rule " + strconv.Itoa(i) + " Permission " + p + " must start with aoss:"
				}
			}
		}
	}
	return ""
}

// aossMatch reports whether a policy resource pattern (with * wildcards)
// covers resource.
func aossMatch(pattern, resource string) bool {
	if pattern == resource {
		return true
	}
	ok, err := path.Match(pattern, resource)
	return err == nil && ok
}

// aossPolicyResources lists every Resource pattern in a policy document.
func aossPolicyResources(doc any) []string {
	var stmts []any
	switch d := doc.(type) {
	case map[string]any:
		stmts = []any{d}
	case []any:
		stmts = d
	}
	var out []string
	for _, s := range stmts {
		obj, _ := s.(map[string]any)
		for _, r := range aossList(obj, "Rules") {
			rule, _ := r.(map[string]any)
			out = append(out, aossStrings(rule, "Resource")...)
		}
	}
	return out
}

func aossPolicyCovers(doc any, resource string) bool {
	for _, pattern := range aossPolicyResources(doc) {
		if aossMatch(pattern, resource) {
			return true
		}
	}
	return false
}

func (st *openSearchServerlessStore) encryptionPolicyForLocked(collection string) *aossPolicy {
	var names []string
	for key, p := range st.policies {
		if p.kind == "security" && p.ptype == "encryption" && aossPolicyCovers(p.doc, "collection/"+collection) {
			names = append(names, key)
		}
	}
	if len(names) == 0 {
		return nil
	}
	sort.Strings(names) // deterministic when several match
	return st.policies[names[0]]
}

func aossCreatePolicy(kind string) aossHandler {
	return func(st *openSearchServerlessStore, in map[string]any, _ string) (map[string]any, *aossErr) {
		ptype, name := aossStr(in, "type"), aossStr(in, "name")
		key := aossPolicyKey(kind, ptype, name)
		if st.policies[key] != nil {
			return nil, aossConflict("Policy with name %s and type %s already exists", name, ptype)
		}
		if len(st.policies) >= aossMaxPerKind*3 {
			return nil, aossQuota("the account has too many policies")
		}
		doc, err := aossParsePolicy(ptype, aossStr(in, "policy"))
		if err != nil {
			return nil, err
		}
		now := aossNowMillis()
		p := &aossPolicy{kind: kind, ptype: ptype, name: name, description: aossStr(in, "description"),
			version: aossVersion(), raw: aossStr(in, "policy"), doc: doc, created: now, modified: now}
		st.policies[key] = p
		return map[string]any{aossPolicyFields[kind].detail: p.detail(true)}, nil
	}
}

func (st *openSearchServerlessStore) policyLocked(kind string, in map[string]any) (*aossPolicy, *aossErr) {
	p := st.policies[aossPolicyKey(kind, aossStr(in, "type"), aossStr(in, "name"))]
	if p == nil {
		return nil, aossNotFound("Policy with name %s and type %s does not exist", aossStr(in, "name"), aossStr(in, "type"))
	}
	return p, nil
}

func aossGetPolicy(kind string) aossHandler {
	return func(st *openSearchServerlessStore, in map[string]any, _ string) (map[string]any, *aossErr) {
		p, err := st.policyLocked(kind, in)
		if err != nil {
			return nil, err
		}
		return map[string]any{aossPolicyFields[kind].detail: p.detail(true)}, nil
	}
}

func aossUpdatePolicy(kind string) aossHandler {
	return func(st *openSearchServerlessStore, in map[string]any, _ string) (map[string]any, *aossErr) {
		p, err := st.policyLocked(kind, in)
		if err != nil {
			return nil, err
		}
		if aossStr(in, "policyVersion") != p.version {
			return nil, aossConflict("Policy version %s is out of date; the current version is %s", aossStr(in, "policyVersion"), p.version)
		}
		next := *p
		if raw, ok := in["policy"].(string); ok {
			doc, err := aossParsePolicy(p.ptype, raw)
			if err != nil {
				return nil, err
			}
			next.raw, next.doc = raw, doc
		}
		if v, ok := in["description"].(string); ok {
			next.description = v
		}
		next.version, next.modified = aossVersion(), aossNowMillis()
		*p = next
		return map[string]any{aossPolicyFields[kind].detail: p.detail(true)}, nil
	}
}

func aossDeletePolicy(kind string) aossHandler {
	return func(st *openSearchServerlessStore, in map[string]any, _ string) (map[string]any, *aossErr) {
		p, err := st.policyLocked(kind, in)
		if err != nil {
			return nil, err
		}
		delete(st.policies, aossPolicyKey(kind, p.ptype, p.name))
		return map[string]any{}, nil
	}
}

// aossResourceFilter normalises a list-call resource filter: bare collection
// names become collection/<name>.
func aossResourceFilter(in map[string]any, key string) []string {
	var out []string
	for _, r := range aossStrings(in, key) {
		if !strings.Contains(r, "/") {
			r = "collection/" + r
		}
		out = append(out, r)
	}
	return out
}

func aossListPolicies(kind string) aossHandler {
	return func(st *openSearchServerlessStore, in map[string]any, _ string) (map[string]any, *aossErr) {
		filterKey := "resource"
		if kind == "lifecycle" {
			filterKey = "resources"
		}
		filters := aossResourceFilter(in, filterKey)
		var matched []*aossPolicy
		for _, p := range st.policies {
			if p.kind != kind || p.ptype != aossStr(in, "type") {
				continue
			}
			if len(filters) > 0 {
				hit := false
				for _, f := range filters {
					if aossPolicyCovers(p.doc, f) {
						hit = true
						break
					}
				}
				if !hit {
					continue
				}
			}
			matched = append(matched, p)
		}
		sort.Slice(matched, func(i, j int) bool { return matched[i].name < matched[j].name })
		page, next, err := aossPage(matched, in)
		if err != nil {
			return nil, err
		}
		out := []any{}
		for _, p := range page {
			out = append(out, p.detail(false))
		}
		return map[string]any{aossPolicyFields[kind].summaries: out, "nextToken": next}, nil
	}
}

func (st *openSearchServerlessStore) batchGetLifecyclePolicy(in map[string]any, _ string) (map[string]any, *aossErr) {
	details, errs := []any{}, []any{}
	for _, item := range aossList(in, "identifiers") {
		id, _ := item.(map[string]any)
		if p := st.policies[aossPolicyKey("lifecycle", aossStr(id, "type"), aossStr(id, "name"))]; p != nil {
			details = append(details, p.detail(true))
		} else {
			errs = append(errs, map[string]any{"type": aossStr(id, "type"), "name": aossStr(id, "name"), "errorCode": "NOT_FOUND", "errorMessage": "Lifecycle policy not found"})
		}
	}
	return map[string]any{"lifecyclePolicyDetails": details, "lifecyclePolicyErrorDetails": errs}, nil
}

// batchGetEffectiveLifecyclePolicy finds, for each index, the retention rule
// that applies to it (the first matching policy by name).
func (st *openSearchServerlessStore) batchGetEffectiveLifecyclePolicy(in map[string]any, _ string) (map[string]any, *aossErr) {
	var policies []*aossPolicy
	for _, p := range st.policies {
		if p.kind == "lifecycle" {
			policies = append(policies, p)
		}
	}
	sort.Slice(policies, func(i, j int) bool { return policies[i].name < policies[j].name })
	details, errs := []any{}, []any{}
	for _, item := range aossList(in, "resourceIdentifiers") {
		id, _ := item.(map[string]any)
		ptype, resource := aossStr(id, "type"), aossStr(id, "resource")
		found := false
		for _, p := range policies {
			if p.ptype != ptype {
				continue
			}
			for _, r := range aossList(p.doc.(map[string]any), "Rules") {
				rule := r.(map[string]any)
				if !aossRuleCovers(rule, resource) {
					continue
				}
				d := map[string]any{"type": ptype, "resource": resource, "policyName": p.name, "resourceType": "index"}
				if v := aossStr(rule, "MinIndexRetention"); v != "" {
					d["retentionPeriod"] = v
				} else {
					d["noMinRetentionPeriod"] = true
				}
				details, found = append(details, d), true
				break
			}
			if found {
				break
			}
		}
		if !found {
			errs = append(errs, map[string]any{"type": ptype, "resource": resource, "errorCode": "NOT_FOUND", "errorMessage": "No lifecycle policy applies to this resource"})
		}
	}
	return map[string]any{"effectiveLifecyclePolicyDetails": details, "effectiveLifecyclePolicyErrorDetails": errs}, nil
}

func aossRuleCovers(rule map[string]any, resource string) bool {
	for _, pattern := range aossStrings(rule, "Resource") {
		if aossMatch(pattern, resource) {
			return true
		}
	}
	return false
}

// --- security configs ---------------------------------------------------------------

type aossConfig struct {
	id, ctype, name, description, version string
	saml, idc, federation                 map[string]any
	created, modified                     int64
}

func (c *aossConfig) detail() map[string]any {
	out := map[string]any{"id": c.id, "type": c.ctype, "configVersion": c.version, "createdDate": c.created, "lastModifiedDate": c.modified}
	if c.description != "" {
		out["description"] = c.description
	}
	for key, v := range map[string]map[string]any{"samlOptions": c.saml, "iamIdentityCenterOptions": c.idc, "iamFederationOptions": c.federation} {
		if v != nil {
			out[key] = v
		}
	}
	return out
}

// aossConfigOptions checks that only the options for ctype were sent and,
// on create, that they were sent.
func aossConfigOptions(ctype string, in map[string]any, idcKey string, required bool) *aossErr {
	want := map[string]string{"saml": "samlOptions", "iamidentitycenter": idcKey, "iamfederation": "iamFederationOptions"}
	for t, key := range want {
		_, present := in[key]
		if t == ctype && !present && required {
			return aossValidation("A %s security config needs %s", ctype, key)
		}
		if t != ctype && present {
			return aossValidation("%s can't be set on a %s security config", key, ctype)
		}
	}
	return nil
}

func (st *openSearchServerlessStore) createConfig(in map[string]any, _ string) (map[string]any, *aossErr) {
	ctype, name := aossStr(in, "type"), aossStr(in, "name")
	id := ctype + "/" + aossAccountID + "/" + name
	if st.configs[id] != nil {
		return nil, aossConflict("Security config %s already exists", id)
	}
	if len(st.configs) >= aossMaxPerKind {
		return nil, aossQuota("the account has too many security configs")
	}
	if err := aossConfigOptions(ctype, in, "iamIdentityCenterOptions", true); err != nil {
		return nil, err
	}
	now := aossNowMillis()
	c := &aossConfig{id: id, ctype: ctype, name: name, description: aossStr(in, "description"), version: aossVersion(), created: now, modified: now}
	switch ctype {
	case "saml":
		c.saml = aossMap(in, "samlOptions")
		if _, ok := c.saml["sessionTimeout"]; !ok {
			c.saml["sessionTimeout"] = float64(60)
		}
		c.saml["openSearchServerlessEntityId"] = "aws:opensearch:" + aossAccountID
	case "iamidentitycenter":
		opts := aossMap(in, "iamIdentityCenterOptions")
		c.idc = map[string]any{
			"instanceArn":            aossStr(opts, "instanceArn"),
			"applicationArn":         "arn:aws:sso::" + aossAccountID + ":application/ssoins-" + aossRandID(16) + "/apl-" + aossRandID(16),
			"applicationName":        "Amazon OpenSearch Serverless - " + name,
			"applicationDescription": "IAM Identity Center application for OpenSearch Serverless",
			"userAttribute":          aossOr(aossStr(opts, "userAttribute"), "UserId"),
			"groupAttribute":         aossOr(aossStr(opts, "groupAttribute"), "GroupId"),
		}
	case "iamfederation":
		c.federation = aossMap(in, "iamFederationOptions")
	}
	st.configs[id] = c
	return map[string]any{"securityConfigDetail": c.detail()}, nil
}

func (st *openSearchServerlessStore) configLocked(id string) (*aossConfig, *aossErr) {
	c := st.configs[id]
	if c == nil {
		return nil, aossNotFound("Security config %s does not exist", id)
	}
	return c, nil
}

func (st *openSearchServerlessStore) getConfig(in map[string]any, _ string) (map[string]any, *aossErr) {
	c, err := st.configLocked(aossStr(in, "id"))
	if err != nil {
		return nil, err
	}
	return map[string]any{"securityConfigDetail": c.detail()}, nil
}

func (st *openSearchServerlessStore) updateConfig(in map[string]any, _ string) (map[string]any, *aossErr) {
	c, err := st.configLocked(aossStr(in, "id"))
	if err != nil {
		return nil, err
	}
	if aossStr(in, "configVersion") != c.version {
		return nil, aossConflict("Config version %s is out of date; the current version is %s", aossStr(in, "configVersion"), c.version)
	}
	if err := aossConfigOptions(c.ctype, in, "iamIdentityCenterOptionsUpdates", false); err != nil {
		return nil, err
	}
	next := *c
	if v, ok := in["description"].(string); ok {
		next.description = v
	}
	if v := aossMap(in, "samlOptions"); v != nil {
		v["openSearchServerlessEntityId"] = "aws:opensearch:" + aossAccountID
		next.saml = v
	}
	if v := aossMap(in, "iamFederationOptions"); v != nil {
		next.federation = v
	}
	if v := aossMap(in, "iamIdentityCenterOptionsUpdates"); v != nil {
		idc := map[string]any{}
		for k, x := range c.idc {
			idc[k] = x
		}
		for k, x := range v {
			idc[k] = x
		}
		next.idc = idc
	}
	next.version, next.modified = aossVersion(), aossNowMillis()
	*c = next
	return map[string]any{"securityConfigDetail": c.detail()}, nil
}

func (st *openSearchServerlessStore) deleteConfig(in map[string]any, _ string) (map[string]any, *aossErr) {
	c, err := st.configLocked(aossStr(in, "id"))
	if err != nil {
		return nil, err
	}
	delete(st.configs, c.id)
	return map[string]any{}, nil
}

func (st *openSearchServerlessStore) listConfigs(in map[string]any, _ string) (map[string]any, *aossErr) {
	var matched []*aossConfig
	for _, c := range st.configs {
		if c.ctype == aossStr(in, "type") {
			matched = append(matched, c)
		}
	}
	sort.Slice(matched, func(i, j int) bool { return matched[i].id < matched[j].id })
	page, next, err := aossPage(matched, in)
	if err != nil {
		return nil, err
	}
	out := []any{}
	for _, c := range page {
		out = append(out, c.detail())
	}
	return map[string]any{"securityConfigSummaries": out, "nextToken": next}, nil
}

// --- VPC endpoints -----------------------------------------------------------------

type aossEndpoint struct {
	id, name, vpcID, status string
	subnets, groups         []string
	created, modified       int64
}

func (e *aossEndpoint) detail() map[string]any {
	return map[string]any{
		"id": e.id, "name": e.name, "vpcId": e.vpcID, "status": e.status, "createdDate": e.created,
		"lastModifiedDate": e.modified, "subnetIds": aossAnyList(e.subnets), "securityGroupIds": aossAnyList(e.groups),
	}
}

func aossAnyList(items []string) []any {
	out := make([]any, 0, len(items))
	for _, s := range items {
		out = append(out, s)
	}
	return out
}

func (e *aossEndpoint) settle() {
	if e.status == "PENDING" {
		e.status = "ACTIVE"
	}
}

// aossEC2Snapshot is what a VPC endpoint call needs from the EC2 store, read
// before the OpenSearch Serverless lock is taken.
type aossEC2Snapshot struct {
	vpcExists    bool
	subnetVPC    map[string]string // subnet id -> vpc id
	groupVPC     map[string]string // security group id -> vpc id
	defaultGroup string
}

func (st *openSearchServerlessStore) snapshotEC2(in map[string]any) *aossEC2Snapshot {
	snap := &aossEC2Snapshot{subnetVPC: map[string]string{}, groupVPC: map[string]string{}}
	if st.ec2 == nil {
		return snap
	}
	vpcID := aossStr(in, "vpcId")
	if vpcID == "" { // UpdateVpcEndpoint: the endpoint's VPC is read later under our lock
		st.mu.Lock()
		if e := st.endpoints[aossStr(in, "id")]; e != nil {
			vpcID = e.vpcID
		}
		st.mu.Unlock()
	}
	if vpcID != "" {
		snap.vpcExists = len(st.ec2.DescribeVpcs([]string{vpcID})) > 0
		for _, g := range st.ec2.DescribeSecurityGroups(nil, []string{"default"}, vpcID) {
			snap.defaultGroup, snap.groupVPC[g.ID] = g.ID, g.VpcID
			break
		}
	}
	subnets := append(aossStrings(in, "subnetIds"), aossStrings(in, "addSubnetIds")...)
	if len(subnets) > 0 {
		for _, s := range st.ec2.DescribeSubnets(subnets, nil) {
			snap.subnetVPC[s.ID] = s.VpcID
		}
	}
	groups := append(aossStrings(in, "securityGroupIds"), aossStrings(in, "addSecurityGroupIds")...)
	if len(groups) > 0 {
		for _, g := range st.ec2.DescribeSecurityGroups(groups, nil, "") {
			snap.groupVPC[g.ID] = g.VpcID
		}
	}
	return snap
}

func aossCheckNetwork(snap *aossEC2Snapshot, vpcID string, subnets, groups []string) *aossErr {
	for _, s := range subnets {
		if snap.subnetVPC[s] != vpcID {
			return aossValidation("Subnet %s does not exist in VPC %s", s, vpcID)
		}
	}
	for _, g := range groups {
		if snap.groupVPC[g] != vpcID {
			return aossValidation("Security group %s does not exist in VPC %s", g, vpcID)
		}
	}
	return nil
}

func (st *openSearchServerlessStore) createEndpoint(in map[string]any, _ string) (map[string]any, *aossErr) {
	snap, _ := in["__ec2"].(*aossEC2Snapshot)
	name, vpcID := aossStr(in, "name"), aossStr(in, "vpcId")
	for _, e := range st.endpoints {
		if e.name == name {
			return nil, aossConflict("A VPC endpoint named %s already exists", name)
		}
	}
	if len(st.endpoints) >= aossMaxPerKind {
		return nil, aossQuota("the account has too many VPC endpoints")
	}
	if snap == nil || !snap.vpcExists {
		return nil, aossValidation("VPC %s does not exist", vpcID)
	}
	subnets, groups := aossDedupe(aossStrings(in, "subnetIds")), aossDedupe(aossStrings(in, "securityGroupIds"))
	if len(groups) == 0 && snap.defaultGroup != "" {
		groups = []string{snap.defaultGroup}
	}
	if err := aossCheckNetwork(snap, vpcID, subnets, groups); err != nil {
		return nil, err
	}
	now := aossNowMillis()
	e := &aossEndpoint{id: "vpce-" + aossRandID(17), name: name, vpcID: vpcID, status: "PENDING", subnets: subnets, groups: groups, created: now, modified: now}
	st.endpoints[e.id] = e
	return map[string]any{"createVpcEndpointDetail": e.detail()}, nil
}

func aossDedupe(items []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range items {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

func (st *openSearchServerlessStore) batchGetEndpoint(in map[string]any, _ string) (map[string]any, *aossErr) {
	details, errs := []any{}, []any{}
	for _, id := range aossStrings(in, "ids") {
		if e := st.endpoints[id]; e != nil {
			e.settle()
			details = append(details, e.detail())
		} else {
			errs = append(errs, map[string]any{"id": id, "errorCode": "NOT_FOUND", "errorMessage": "VPC endpoint not found"})
		}
	}
	return map[string]any{"vpcEndpointDetails": details, "vpcEndpointErrorDetails": errs}, nil
}

func (st *openSearchServerlessStore) listEndpoints(in map[string]any, _ string) (map[string]any, *aossErr) {
	status := aossStr(aossMap(in, "vpcEndpointFilters"), "status")
	var matched []*aossEndpoint
	for _, e := range st.endpoints {
		e.settle()
		if status == "" || e.status == status {
			matched = append(matched, e)
		}
	}
	sort.Slice(matched, func(i, j int) bool { return matched[i].name < matched[j].name })
	page, next, err := aossPage(matched, in)
	if err != nil {
		return nil, err
	}
	out := []any{}
	for _, e := range page {
		out = append(out, e.detail())
	}
	return map[string]any{"vpcEndpointSummaries": out, "nextToken": next}, nil
}

func (st *openSearchServerlessStore) updateEndpoint(in map[string]any, _ string) (map[string]any, *aossErr) {
	snap, _ := in["__ec2"].(*aossEC2Snapshot)
	e := st.endpoints[aossStr(in, "id")]
	if e == nil {
		return nil, aossValidation("VPC endpoint %s does not exist", aossStr(in, "id"))
	}
	if e.status != "ACTIVE" {
		return nil, aossConflict("VPC endpoint %s is %s; it must be ACTIVE to update", e.id, e.status)
	}
	adds, addGroups := aossStrings(in, "addSubnetIds"), aossStrings(in, "addSecurityGroupIds")
	if snap == nil {
		snap = &aossEC2Snapshot{}
	}
	if err := aossCheckNetwork(snap, e.vpcID, adds, addGroups); err != nil {
		return nil, err
	}
	subnets := aossApply(e.subnets, adds, aossStrings(in, "removeSubnetIds"))
	groups := aossApply(e.groups, addGroups, aossStrings(in, "removeSecurityGroupIds"))
	if len(subnets) == 0 {
		return nil, aossValidation("A VPC endpoint needs at least one subnet")
	}
	if len(groups) > 5 {
		return nil, aossValidation("A VPC endpoint can have at most 5 security groups")
	}
	e.subnets, e.groups, e.status, e.modified = subnets, groups, "PENDING", aossNowMillis()
	return map[string]any{"UpdateVpcEndpointDetail": e.detail()}, nil
}

// aossApply returns cur plus add minus remove, as a fresh slice.
func aossApply(cur, add, remove []string) []string {
	drop := map[string]bool{}
	for _, r := range remove {
		drop[r] = true
	}
	var out []string
	for _, s := range aossDedupe(append(append([]string{}, cur...), add...)) {
		if !drop[s] {
			out = append(out, s)
		}
	}
	return out
}

func (st *openSearchServerlessStore) deleteEndpoint(in map[string]any, _ string) (map[string]any, *aossErr) {
	e := st.endpoints[aossStr(in, "id")]
	if e == nil {
		return nil, aossNotFound("VPC endpoint %s does not exist", aossStr(in, "id"))
	}
	delete(st.endpoints, e.id)
	return map[string]any{"deleteVpcEndpointDetail": map[string]any{"id": e.id, "name": e.name, "status": "DELETING"}}, nil
}
