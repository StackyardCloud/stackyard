package server

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/netip"
	"regexp"
	"sort"
	"strings"

	"github.com/stackyard/stackyard/internal/services/ec2"
)

// Resolver endpoints, rules and their VPC associations, query logging,
// the per-VPC resolver/DNSSEC settings, Outposts resolvers and policies.

const (
	r53rMaxEndpoints     = 200
	r53rMaxEndpointIPs   = 20
	r53rMaxRules         = 1000
	r53rMaxTargetIPs     = 6
	r53rMaxQLCs          = 1000
	r53rMaxAssociations  = 2000
	r53rMaxOutposts      = 100
	r53rAutodefinedID    = "rslvr-autodefined-rr-internet-resolver"
	r53rMinOutpostCount  = 4
	r53rMaxOutpostCount  = 64
	r53rMaxSecurityGroup = 10
)

type r53rEndpoint struct {
	r53rRec
	ips []map[string]any // IpAddressResponse records
}

func (e *r53rEndpoint) out() map[string]any {
	m := e.view()
	m["IpAddressCount"] = len(e.ips)
	return m
}

type r53rRule struct {
	r53rRec
	autodefined bool
}

func (st *r53rStore) seedLocked() {
	st.rules[r53rAutodefinedID] = &r53rRule{autodefined: true, r53rRec: r53rRec{rec: map[string]any{
		"Id": r53rAutodefinedID, "Arn": "arn:aws:route53resolver:" + r53rRegion + "::autodefined-rule/" + r53rAutodefinedID,
		"DomainName": ".", "Name": "Internet Resolver", "RuleType": "RECURSIVE", "Status": "COMPLETE",
		"OwnerId": "Route 53 Resolver", "ShareStatus": "NOT_SHARED", "CreatorRequestId": "",
	}}}
	st.seedManagedDomainListsLocked()
}

// --- EC2 lookups ---------------------------------------------------------------

func (st *r53rStore) vpcExists(id string) bool {
	return id != "" && len(st.ec2.DescribeVpcs([]string{id})) == 1
}

func (st *r53rStore) subnet(id string) (ec2.Subnet, bool) {
	if id == "" {
		return ec2.Subnet{}, false
	}
	subs := st.ec2.DescribeSubnets([]string{id}, nil)
	if len(subs) != 1 {
		return ec2.Subnet{}, false
	}
	return subs[0], true
}

func (st *r53rStore) checkSecurityGroups(ids []string, vpc string) *r53rErr {
	if len(ids) == 0 || len(ids) > r53rMaxSecurityGroup {
		return r53rE("InvalidParameterException", "specify between 1 and %d security groups", r53rMaxSecurityGroup)
	}
	found := map[string]string{}
	for _, g := range st.ec2.DescribeSecurityGroups(ids, nil, "") {
		found[g.ID] = g.VpcID
	}
	for _, id := range ids {
		v, ok := found[id]
		if !ok {
			return r53rE("InvalidParameterException", "security group %s doesn't exist", id)
		}
		if v != vpc {
			return r53rE("InvalidParameterException", "security group %s isn't in VPC %s", id, vpc)
		}
	}
	return nil
}

// usedIPsLocked returns every address held by an endpoint.
func (st *r53rStore) usedIPsLocked() map[string]bool {
	used := map[string]bool{}
	for _, e := range st.endpoints {
		for _, ip := range e.ips {
			for _, k := range []string{"Ip", "Ipv6"} {
				if s, _ := ip[k].(string); s != "" {
					used[s] = true
				}
			}
		}
	}
	return used
}

// r53rUsable reports whether addr is a host address AWS hands out in the
// subnet: not one of the first four nor the broadcast address.
func r53rUsable(prefix netip.Prefix, addr netip.Addr) bool {
	if !prefix.Contains(addr) {
		return false
	}
	reserved := prefix.Masked().Addr()
	for i := 0; i < 4; i++ {
		if addr == reserved {
			return false
		}
		reserved = reserved.Next()
	}
	return prefix.Contains(addr.Next())
}

// r53rAllocIP returns the lowest free host address in the subnet.
func r53rAllocIP(prefix netip.Prefix, used map[string]bool) (string, bool) {
	a := prefix.Masked().Addr()
	for i := 0; i < 1<<16 && prefix.Contains(a); i, a = i+1, a.Next() {
		if r53rUsable(prefix, a) && !used[a.String()] {
			return a.String(), true
		}
	}
	return "", false
}

// buildIPsLocked turns IpAddressRequest entries into IpAddressResponse
// records, checking subnets (all in one VPC) and allocating free addresses.
func (st *r53rStore) buildIPsLocked(reqs []any, vpc *string, endpointType string, used map[string]bool) ([]map[string]any, *r53rErr) {
	var out []map[string]any
	for _, raw := range reqs {
		req, _ := raw.(map[string]any)
		sub, ok := st.subnet(r53rStr(req, "SubnetId"))
		if !ok {
			return nil, r53rE("InvalidParameterException", "subnet %s doesn't exist", r53rStr(req, "SubnetId"))
		}
		if *vpc == "" {
			*vpc = sub.VpcID
		} else if *vpc != sub.VpcID {
			return nil, r53rE("InvalidParameterException", "all IP addresses must be in subnets of one VPC")
		}
		ip, ipv6 := r53rStr(req, "Ip"), r53rStr(req, "Ipv6")
		if endpointType == "IPV6" && ip != "" {
			return nil, r53rE("InvalidParameterException", "an IPV6 endpoint can't have IPv4 addresses")
		}
		if (endpointType == "IPV6" || endpointType == "DUALSTACK") && ipv6 == "" {
			return nil, r53rE("InvalidParameterException", "an %s endpoint needs an IPv6 address for every IP", endpointType)
		}
		if endpointType == "IPV4" && ipv6 != "" {
			return nil, r53rE("InvalidParameterException", "an IPV4 endpoint can't have IPv6 addresses")
		}
		if endpointType != "IPV6" {
			prefix, err := netip.ParsePrefix(sub.CidrBlock)
			if err != nil {
				return nil, r53rE("InvalidParameterException", "subnet %s has no IPv4 CIDR", sub.ID)
			}
			if ip == "" {
				if ip, ok = r53rAllocIP(prefix, used); !ok {
					return nil, r53rE("InvalidRequestException", "subnet %s has no free IP addresses", sub.ID)
				}
			} else if addr, err := netip.ParseAddr(ip); err != nil || !addr.Is4() || !r53rUsable(prefix, addr) {
				return nil, r53rE("InvalidParameterException", "IP address %s isn't a usable address in subnet %s (%s)", ip, sub.ID, sub.CidrBlock)
			} else if used[ip] {
				return nil, r53rE("ResourceExistsException", "IP address %s is already in use", ip)
			}
			used[ip] = true
		}
		if ipv6 != "" {
			if addr, err := netip.ParseAddr(ipv6); err != nil || !addr.Is6() {
				return nil, r53rE("InvalidParameterException", "%s isn't an IPv6 address", ipv6)
			}
			if used[ipv6] {
				return nil, r53rE("ResourceExistsException", "IP address %s is already in use", ipv6)
			}
			used[ipv6] = true
		}
		now := r53rNow()
		rec := map[string]any{"IpId": r53rID("rni-"), "SubnetId": sub.ID, "Status": "ATTACHED", "CreationTime": now, "ModificationTime": now}
		if ip != "" {
			rec["Ip"] = ip
		}
		if ipv6 != "" {
			rec["Ipv6"] = ipv6
		}
		out = append(out, rec)
	}
	return out, nil
}

func r53rCheckProtocols(direction string, protos []string) *r53rErr {
	seen := map[string]bool{}
	for _, p := range protos {
		if seen[p] {
			return r53rE("InvalidParameterException", "protocol %s is listed twice", p)
		}
		seen[p] = true
		if p == "DoH-FIPS" && direction == "OUTBOUND" {
			return r53rE("InvalidParameterException", "DoH-FIPS is only supported on inbound endpoints")
		}
	}
	return nil
}

func (st *r53rStore) endpointLocked(id string) (*r53rEndpoint, *r53rErr) {
	if e := st.endpoints[id]; e != nil {
		return e, nil
	}
	return nil, r53rE("ResourceNotFoundException", "resolver endpoint %s doesn't exist", id)
}

// --- rules ---------------------------------------------------------------------

var r53rLabelRE = regexp.MustCompile(`^[a-z0-9_*-]{1,63}$`)

// r53rDomain normalises a rule domain name to lower case with a trailing dot.
func r53rDomain(name string) (string, bool) {
	n := strings.TrimSuffix(strings.ToLower(strings.TrimSpace(name)), ".")
	if n == "" {
		return ".", true
	}
	for _, label := range strings.Split(n, ".") {
		if !r53rLabelRE.MatchString(label) {
			return "", false
		}
	}
	return n + ".", true
}

func r53rCheckTargets(targets []any) ([]any, *r53rErr) {
	if len(targets) > r53rMaxTargetIPs {
		return nil, r53rE("InvalidParameterException", "a rule can have at most %d target IP addresses", r53rMaxTargetIPs)
	}
	out := make([]any, 0, len(targets))
	for _, raw := range targets {
		t, _ := raw.(map[string]any)
		ip, ipv6 := r53rStr(t, "Ip"), r53rStr(t, "Ipv6")
		if ip == "" && ipv6 == "" {
			return nil, r53rE("InvalidParameterException", "each target needs an Ip or Ipv6 address")
		}
		if a, err := netip.ParseAddr(ip); ip != "" && (err != nil || !a.Is4()) {
			return nil, r53rE("InvalidParameterException", "%s isn't an IPv4 address", ip)
		}
		if a, err := netip.ParseAddr(ipv6); ipv6 != "" && (err != nil || !a.Is6()) {
			return nil, r53rE("InvalidParameterException", "%s isn't an IPv6 address", ipv6)
		}
		rec := map[string]any{"Port": 53, "Protocol": "Do53"}
		r53rCopy(rec, t, "Ip", "Ipv6", "Port", "Protocol", "ServerNameIndication")
		if rec["Protocol"] == "DoH-FIPS" {
			return nil, r53rE("InvalidParameterException", "DoH-FIPS isn't supported for rule targets")
		}
		if r53rHas(t, "ServerNameIndication") && rec["Protocol"] != "DoH" {
			return nil, r53rE("InvalidParameterException", "ServerNameIndication needs the DoH protocol")
		}
		out = append(out, rec)
	}
	return out, nil
}

func (st *r53rStore) ruleLocked(id string) (*r53rRule, *r53rErr) {
	if r := st.rules[id]; r != nil {
		return r, nil
	}
	return nil, r53rE("ResourceNotFoundException", "resolver rule %s doesn't exist", id)
}

// checkRuleEndpointLocked requires an endpoint whose direction fits the rule.
func (st *r53rStore) checkRuleEndpointLocked(ruleType, endpointID string) *r53rErr {
	e, err := st.endpointLocked(endpointID)
	if err != nil {
		return err
	}
	want := "OUTBOUND"
	if ruleType == "DELEGATE" {
		want = "INBOUND_DELEGATION"
	}
	if e.rec["Direction"] != want {
		return r53rE("InvalidRequestException", "%s rules need an %s endpoint; %s is %v", ruleType, want, endpointID, e.rec["Direction"])
	}
	return nil
}

func (st *r53rStore) createRule(in map[string]any) (map[string]any, *r53rErr) {
	if e := r53rCheckName(in, "Name", "InvalidParameterException"); e != nil {
		return nil, e
	}
	for _, r := range st.rules {
		if r.rec["CreatorRequestId"] == r53rStr(in, "CreatorRequestId") {
			return nil, r53rE("ResourceExistsException", "a rule with CreatorRequestId %s already exists", r53rStr(in, "CreatorRequestId"))
		}
	}
	if len(st.rules) > r53rMaxRules {
		return nil, r53rE("LimitExceededException", "an account can have at most %d resolver rules", r53rMaxRules)
	}
	ruleType := r53rStr(in, "RuleType")
	domain, ok := r53rDomain(r53rStr(in, "DomainName"))
	if !ok {
		return nil, r53rE("InvalidParameterException", "%q isn't a valid domain name", r53rStr(in, "DomainName"))
	}
	targets, e := r53rCheckTargets(r53rList(in, "TargetIps"))
	if e != nil {
		return nil, e
	}
	endpointID := r53rStr(in, "ResolverEndpointId")
	switch ruleType {
	case "RECURSIVE":
		return nil, r53rE("InvalidRequestException", "RECURSIVE rules are defined by Resolver and can't be created")
	case "FORWARD":
		if !r53rHas(in, "DomainName") || len(targets) == 0 || endpointID == "" {
			return nil, r53rE("InvalidRequestException", "FORWARD rules need DomainName, TargetIps and ResolverEndpointId")
		}
	case "SYSTEM":
		if !r53rHas(in, "DomainName") || len(targets) > 0 || endpointID != "" {
			return nil, r53rE("InvalidRequestException", "SYSTEM rules take a DomainName and no TargetIps or ResolverEndpointId")
		}
	case "DELEGATE":
		if r53rStr(in, "DelegationRecord") == "" || endpointID == "" {
			return nil, r53rE("InvalidRequestException", "DELEGATE rules need DelegationRecord and ResolverEndpointId")
		}
	}
	if endpointID != "" {
		if e := st.checkRuleEndpointLocked(ruleType, endpointID); e != nil {
			return nil, e
		}
	}
	id := r53rID("rslvr-rr-")
	now := r53rNow()
	rec := map[string]any{
		"Id": id, "Arn": r53rARN("resolver-rule", id), "CreatorRequestId": r53rStr(in, "CreatorRequestId"),
		"RuleType": ruleType, "DomainName": domain, "Status": "COMPLETE", "StatusMessage": "Successfully created Resolver Rule",
		"OwnerId": r53rAccountID, "ShareStatus": "NOT_SHARED", "CreationTime": now, "ModificationTime": now,
	}
	r53rCopy(rec, in, "Name", "ResolverEndpointId", "DelegationRecord")
	if len(targets) > 0 {
		rec["TargetIps"] = targets
	}
	if e := st.tagsLocked(rec["Arn"].(string), r53rList(in, "Tags")); e != nil {
		return nil, e
	}
	st.rules[id] = &r53rRule{r53rRec: r53rRec{rec: rec}}
	return map[string]any{"ResolverRule": rec}, nil
}

// --- query log configs -------------------------------------------------------

var (
	r53rS3DestRE       = regexp.MustCompile(`^arn:aws:s3:::([a-z0-9][a-z0-9.-]{1,61}[a-z0-9])(/.*)?$`)
	r53rLogsDestRE     = regexp.MustCompile(`^arn:aws:logs:[a-z0-9-]+:\d{12}:log-group:[^:]+(:\*)?$`)
	r53rFirehoseDestRE = regexp.MustCompile(`^arn:aws:firehose:[a-z0-9-]+:\d{12}:deliverystream/[A-Za-z0-9_.-]+$`)
)

// destinationType checks a query-log destination and returns its kind. S3
// buckets must exist in Stackyard's S3 store; log groups and delivery streams
// are checked by ARN (ponytail: their stores create them on first use).
func (st *r53rStore) destinationType(arn string) (string, *r53rErr) {
	switch {
	case r53rS3DestRE.MatchString(arn):
		bucket := r53rS3DestRE.FindStringSubmatch(arn)[1]
		if _, err := st.s3.GetBucket(bucket); err != nil {
			return "", r53rE("InvalidParameterException", "S3 bucket %s doesn't exist", bucket)
		}
		return "S3", nil
	case r53rLogsDestRE.MatchString(arn):
		return "CloudWatchLogs", nil
	case r53rFirehoseDestRE.MatchString(arn):
		return "KinesisFirehose", nil
	}
	return "", r53rE("InvalidParameterException", "DestinationArn must be an S3 bucket, CloudWatch Logs log group or Firehose delivery stream ARN")
}

var r53rQLCFields = map[string]string{
	"CreationTime": "CreationTime", "CreatorRequestId": "CreatorRequestId", "DestinationArn": "DestinationArn",
	"DestinationType": "DestinationType", "Id": "Id", "Name": "Name", "OwnerId": "OwnerId", "ShareStatus": "ShareStatus",
	"Status": "Status", "Arn": "Arn", "AssociationCount": "AssociationCount",
}

var r53rQLCAssocFields = map[string]string{
	"CreationTime": "CreationTime", "Error": "Error", "Id": "Id", "ResolverQueryLogConfigId": "ResolverQueryLogConfigId",
	"ResourceId": "ResourceId", "Status": "Status",
}

// sortItems orders items by the request's SortBy/SortOrder.
func r53rSortItems(items []map[string]any, in map[string]any, fields map[string]string) *r53rErr {
	by := r53rStr(in, "SortBy")
	if by == "" {
		return nil
	}
	member, ok := fields[by]
	if !ok {
		return r53rE("InvalidParameterException", "can't sort by %q", by)
	}
	desc := r53rStr(in, "SortOrder") == "DESCENDING"
	sort.SliceStable(items, func(i, j int) bool {
		a, b := r53rSortKey(items[i][member]), r53rSortKey(items[j][member])
		if desc {
			return a > b
		}
		return a < b
	})
	return nil
}

func r53rSortKey(v any) string {
	if n, ok := v.(int); ok {
		return fmt.Sprintf("%012d", n)
	}
	s, _ := v.(string)
	return s
}

func (st *r53rStore) qlcLocked(id string) (*r53rRec, *r53rErr) {
	if q := st.qlcs[id]; q != nil {
		return q, nil
	}
	return nil, r53rE("ResourceNotFoundException", "query logging configuration %s doesn't exist", id)
}

// --- per-VPC settings ----------------------------------------------------------

type r53rSetting struct {
	store  func(*r53rStore) map[string]*r53rRec
	prefix string
	field  string
	def    string
}

var (
	r53rResolverConfig = r53rSetting{func(st *r53rStore) map[string]*r53rRec { return st.resolverConfigs }, "rslvr-rc-", "AutodefinedReverse", "ENABLED"}
	r53rDnssecConfig   = r53rSetting{func(st *r53rStore) map[string]*r53rRec { return st.dnssecConfigs }, "rslvr-rsdc-", "ValidationStatus", "DISABLED"}
	r53rFirewallConfig = r53rSetting{func(st *r53rStore) map[string]*r53rRec { return st.firewallConfigs }, "rslvr-fc-", "FirewallFailOpen", "DISABLED"}
)

// getLocked returns the VPC's setting record, creating the default one.
func (s r53rSetting) getLocked(st *r53rStore, vpc string) *r53rRec {
	m := s.store(st)
	if r := m[vpc]; r != nil {
		return r
	}
	sum := sha256.Sum256([]byte(s.prefix + vpc))
	r := &r53rRec{rec: map[string]any{"Id": s.prefix + hex.EncodeToString(sum[:])[:17], "ResourceId": vpc, "OwnerId": r53rAccountID, s.field: s.def}}
	m[vpc] = r
	return r
}

// view settles a transitional value (ENABLING → ENABLED, ...) on read.
func (s r53rSetting) view(r *r53rRec) map[string]any {
	if r.settle != "" {
		r.rec[s.field] = r.settle
		r.settle = ""
	}
	return r.rec
}

func (s r53rSetting) get(st *r53rStore, vpc string) (map[string]any, *r53rErr) {
	if !st.vpcExists(vpc) {
		return nil, r53rE("ResourceNotFoundException", "VPC %s doesn't exist", vpc)
	}
	return s.view(s.getLocked(st, vpc)), nil
}

// update applies ENABLE/DISABLE/USE_LOCAL_RESOURCE_SETTING, reporting the
// transitional status until the next read.
func (s r53rSetting) update(st *r53rStore, vpc, flag string) (map[string]any, *r53rErr) {
	if !st.vpcExists(vpc) {
		return nil, r53rE("ResourceNotFoundException", "VPC %s doesn't exist", vpc)
	}
	r := s.getLocked(st, vpc)
	transit, final := map[string][2]string{
		"ENABLE": {"ENABLING", "ENABLED"}, "DISABLE": {"DISABLING", "DISABLED"},
		"USE_LOCAL_RESOURCE_SETTING": {"UPDATING_TO_USE_LOCAL_RESOURCE_SETTING", "USE_LOCAL_RESOURCE_SETTING"},
	}[flag][0], ""
	final = map[string]string{"ENABLE": "ENABLED", "DISABLE": "DISABLED", "USE_LOCAL_RESOURCE_SETTING": "USE_LOCAL_RESOURCE_SETTING"}[flag]
	r.rec[s.field], r.settle = transit, final
	return r.rec, nil
}

func (s r53rSetting) list(st *r53rStore, in map[string]any) ([]any, string, *r53rErr) {
	var items []any
	for _, vpc := range st.ec2.DescribeVpcs(nil) {
		items = append(items, s.view(s.getLocked(st, vpc.ID)))
	}
	return r53rPage(items, in, "InvalidNextTokenException")
}

// --- handlers --------------------------------------------------------------------

func init() {
	h := r53rHandlers

	h["CreateResolverEndpoint"] = func(st *r53rStore, in map[string]any) (map[string]any, *r53rErr) {
		if e := r53rCheckName(in, "Name", "InvalidParameterException"); e != nil {
			return nil, e
		}
		for _, e := range st.endpoints {
			if e.rec["CreatorRequestId"] == r53rStr(in, "CreatorRequestId") {
				return nil, r53rE("ResourceExistsException", "an endpoint with CreatorRequestId %s already exists", r53rStr(in, "CreatorRequestId"))
			}
		}
		if len(st.endpoints) >= r53rMaxEndpoints {
			return nil, r53rE("LimitExceededException", "an account can have at most %d resolver endpoints", r53rMaxEndpoints)
		}
		direction := r53rStr(in, "Direction")
		endpointType := r53rStr(in, "ResolverEndpointType")
		if endpointType == "" {
			endpointType = "IPV4"
		}
		protos := r53rStrings(in, "Protocols")
		if len(protos) == 0 {
			protos = []string{"Do53"}
		}
		if e := r53rCheckProtocols(direction, protos); e != nil {
			return nil, e
		}
		if (r53rStr(in, "OutpostArn") == "") != (r53rStr(in, "PreferredInstanceType") == "") {
			return nil, r53rE("InvalidParameterException", "OutpostArn and PreferredInstanceType must be given together")
		}
		if arn := r53rStr(in, "OutpostArn"); arn != "" && st.outpostByARNLocked(arn) == nil {
			return nil, r53rE("ResourceNotFoundException", "no Outposts resolver exists on %s", arn)
		}
		vpc := ""
		ips, e := st.buildIPsLocked(r53rList(in, "IpAddresses"), &vpc, endpointType, st.usedIPsLocked())
		if e != nil {
			return nil, e
		}
		groups := r53rStrings(in, "SecurityGroupIds")
		if e := st.checkSecurityGroups(groups, vpc); e != nil {
			return nil, e
		}
		prefix := "rslvr-in-"
		if direction == "OUTBOUND" {
			prefix = "rslvr-out-"
		}
		id := r53rID(prefix)
		now := r53rNow()
		rec := map[string]any{
			"Id": id, "Arn": r53rARN("resolver-endpoint", id), "CreatorRequestId": r53rStr(in, "CreatorRequestId"),
			"Direction": direction, "HostVPCId": vpc, "SecurityGroupIds": r53rList(in, "SecurityGroupIds"),
			"Status": "CREATING", "StatusMessage": "[Trace id: stackyard] Creating the Resolver Endpoint",
			"ResolverEndpointType": endpointType, "Protocols": r53rStringsAny(protos), "CreationTime": now, "ModificationTime": now,
		}
		r53rCopy(rec, in, "Name", "OutpostArn", "PreferredInstanceType", "RniEnhancedMetricsEnabled", "TargetNameServerMetricsEnabled", "Dns64Enabled", "Ipv6InternetAccessEnabled")
		if e := st.tagsLocked(rec["Arn"].(string), r53rList(in, "Tags")); e != nil {
			return nil, e
		}
		ep := &r53rEndpoint{r53rRec: r53rRec{rec: rec, settle: "OPERATIONAL"}, ips: ips}
		st.endpoints[id] = ep
		out := map[string]any{}
		for k, v := range rec {
			out[k] = v
		}
		out["IpAddressCount"] = len(ips)
		return map[string]any{"ResolverEndpoint": out}, nil
	}
	h["GetResolverEndpoint"] = func(st *r53rStore, in map[string]any) (map[string]any, *r53rErr) {
		e, err := st.endpointLocked(r53rStr(in, "ResolverEndpointId"))
		if err != nil {
			return nil, err
		}
		return map[string]any{"ResolverEndpoint": e.out()}, nil
	}
	h["UpdateResolverEndpoint"] = func(st *r53rStore, in map[string]any) (map[string]any, *r53rErr) {
		e, err := st.endpointLocked(r53rStr(in, "ResolverEndpointId"))
		if err != nil {
			return nil, err
		}
		if err := r53rCheckName(in, "Name", "InvalidParameterException"); err != nil {
			return nil, err
		}
		endpointType, _ := e.rec["ResolverEndpointType"].(string)
		if t := r53rStr(in, "ResolverEndpointType"); t != "" {
			endpointType = t
		}
		ipv6 := map[string]string{}
		used := st.usedIPsLocked()
		for _, raw := range r53rList(in, "UpdateIpAddresses") {
			u, _ := raw.(map[string]any)
			v6 := r53rStr(u, "Ipv6")
			if a, perr := netip.ParseAddr(v6); perr != nil || !a.Is6() {
				return nil, r53rE("InvalidParameterException", "%s isn't an IPv6 address", v6)
			}
			if used[v6] {
				return nil, r53rE("ResourceExistsException", "IP address %s is already in use", v6)
			}
			ipv6[r53rStr(u, "IpId")] = v6
		}
		for id := range ipv6 {
			if !e.hasIP(id) {
				return nil, r53rE("ResourceNotFoundException", "IP address %s isn't on endpoint %v", id, e.rec["Id"])
			}
		}
		for _, ip := range e.ips {
			v6, _ := ip["Ipv6"].(string)
			if n := ipv6[ip["IpId"].(string)]; n != "" {
				v6 = n
			}
			v4, _ := ip["Ip"].(string)
			if (endpointType == "IPV6" || endpointType == "DUALSTACK") && v6 == "" {
				return nil, r53rE("InvalidParameterException", "every IP needs an IPv6 address for a %s endpoint; pass UpdateIpAddresses", endpointType)
			}
			if endpointType == "IPV6" && v4 != "" || endpointType == "IPV4" && v6 != "" {
				return nil, r53rE("InvalidParameterException", "the endpoint's IP addresses don't fit type %s", endpointType)
			}
		}
		protos := r53rStrings(in, "Protocols")
		if len(protos) > 0 {
			if err := r53rCheckProtocols(r53rStrS(e.rec["Direction"]), protos); err != nil {
				return nil, err
			}
			e.rec["Protocols"] = r53rStringsAny(protos)
		}
		now := r53rNow()
		for _, ip := range e.ips {
			if n := ipv6[ip["IpId"].(string)]; n != "" {
				ip["Ipv6"], ip["ModificationTime"] = n, now
			}
		}
		e.rec["ResolverEndpointType"] = endpointType
		r53rCopy(e.rec, in, "Name", "RniEnhancedMetricsEnabled", "TargetNameServerMetricsEnabled", "Dns64Enabled", "Ipv6InternetAccessEnabled")
		e.rec["Status"], e.settle, e.rec["ModificationTime"] = "UPDATING", "OPERATIONAL", now
		return map[string]any{"ResolverEndpoint": e.outRaw()}, nil
	}
	h["DeleteResolverEndpoint"] = func(st *r53rStore, in map[string]any) (map[string]any, *r53rErr) {
		e, err := st.endpointLocked(r53rStr(in, "ResolverEndpointId"))
		if err != nil {
			return nil, err
		}
		for _, r := range st.rules {
			if r.rec["ResolverEndpointId"] == e.rec["Id"] {
				return nil, r53rE("InvalidRequestException", "endpoint %v is used by resolver rule %v", e.rec["Id"], r.rec["Id"])
			}
		}
		out := e.outRaw()
		out["Status"] = "DELETING"
		delete(st.endpoints, e.rec["Id"].(string))
		st.forgetLocked(e.rec["Arn"].(string))
		return map[string]any{"ResolverEndpoint": out}, nil
	}
	h["ListResolverEndpoints"] = func(st *r53rStore, in map[string]any) (map[string]any, *r53rErr) {
		var items []map[string]any
		for _, e := range r53rSorted(st.endpoints, func(e *r53rEndpoint) string { return e.rec["Id"].(string) }) {
			items = append(items, e.out())
		}
		items, err := r53rFilter(items, r53rList(in, "Filters"), map[string]string{
			"CreatorRequestId": "CreatorRequestId", "Direction": "Direction", "HostVPCId": "HostVPCId", "IpAddressCount": "IpAddressCount",
			"Name": "Name", "ResolverEndpointType": "ResolverEndpointType", "SecurityGroupIds": "SecurityGroupIds", "Status": "Status",
		}, "InvalidParameterException")
		if err != nil {
			return nil, err
		}
		page, next, err := r53rPage(r53rAny(items), in, "InvalidNextTokenException")
		if err != nil {
			return nil, err
		}
		return map[string]any{"ResolverEndpoints": page, "NextToken": r53rOmit(next), "MaxResults": r53rMaxResults(in)}, nil
	}
	h["ListResolverEndpointIpAddresses"] = func(st *r53rStore, in map[string]any) (map[string]any, *r53rErr) {
		e, err := st.endpointLocked(r53rStr(in, "ResolverEndpointId"))
		if err != nil {
			return nil, err
		}
		page, next, err := r53rPage(r53rAny(e.ips), in, "InvalidNextTokenException")
		if err != nil {
			return nil, err
		}
		return map[string]any{"IpAddresses": page, "NextToken": r53rOmit(next), "MaxResults": r53rMaxResults(in)}, nil
	}
	h["AssociateResolverEndpointIpAddress"] = func(st *r53rStore, in map[string]any) (map[string]any, *r53rErr) {
		e, err := st.endpointLocked(r53rStr(in, "ResolverEndpointId"))
		if err != nil {
			return nil, err
		}
		if len(e.ips) >= r53rMaxEndpointIPs {
			return nil, r53rE("LimitExceededException", "an endpoint can have at most %d IP addresses", r53rMaxEndpointIPs)
		}
		vpc := r53rStrS(e.rec["HostVPCId"])
		ips, err := st.buildIPsLocked([]any{r53rMap(in, "IpAddress")}, &vpc, r53rStrS(e.rec["ResolverEndpointType"]), st.usedIPsLocked())
		if err != nil {
			return nil, err
		}
		e.ips = append(e.ips, ips...)
		e.rec["Status"], e.settle, e.rec["ModificationTime"] = "UPDATING", "OPERATIONAL", r53rNow()
		return map[string]any{"ResolverEndpoint": e.outRaw()}, nil
	}
	h["DisassociateResolverEndpointIpAddress"] = func(st *r53rStore, in map[string]any) (map[string]any, *r53rErr) {
		e, err := st.endpointLocked(r53rStr(in, "ResolverEndpointId"))
		if err != nil {
			return nil, err
		}
		target := r53rMap(in, "IpAddress")
		idx := -1
		for i, ip := range e.ips {
			if (r53rStr(target, "IpId") != "" && ip["IpId"] == r53rStr(target, "IpId")) ||
				(r53rStr(target, "Ip") != "" && ip["Ip"] == r53rStr(target, "Ip")) ||
				(r53rStr(target, "Ipv6") != "" && ip["Ipv6"] == r53rStr(target, "Ipv6")) {
				idx = i
				break
			}
		}
		if idx < 0 {
			return nil, r53rE("ResourceNotFoundException", "that IP address isn't on endpoint %v", e.rec["Id"])
		}
		if len(e.ips) <= 2 {
			return nil, r53rE("InvalidRequestException", "an endpoint must keep at least two IP addresses")
		}
		e.ips = append(e.ips[:idx:idx], e.ips[idx+1:]...)
		e.rec["Status"], e.settle, e.rec["ModificationTime"] = "UPDATING", "OPERATIONAL", r53rNow()
		return map[string]any{"ResolverEndpoint": e.outRaw()}, nil
	}

	h["CreateResolverRule"] = func(st *r53rStore, in map[string]any) (map[string]any, *r53rErr) { return st.createRule(in) }
	h["GetResolverRule"] = func(st *r53rStore, in map[string]any) (map[string]any, *r53rErr) {
		r, err := st.ruleLocked(r53rStr(in, "ResolverRuleId"))
		if err != nil {
			return nil, err
		}
		return map[string]any{"ResolverRule": r.view()}, nil
	}
	h["UpdateResolverRule"] = func(st *r53rStore, in map[string]any) (map[string]any, *r53rErr) {
		r, err := st.ruleLocked(r53rStr(in, "ResolverRuleId"))
		if err != nil {
			return nil, err
		}
		if r.autodefined {
			return nil, r53rE("InvalidRequestException", "the autodefined rule %s can't be changed", r53rAutodefinedID)
		}
		cfg := r53rMap(in, "Config")
		if err := r53rCheckName(cfg, "Name", "InvalidParameterException"); err != nil {
			return nil, err
		}
		ruleType := r53rStrS(r.rec["RuleType"])
		if ruleType == "SYSTEM" && (r53rHas(cfg, "TargetIps") || r53rHas(cfg, "ResolverEndpointId")) {
			return nil, r53rE("InvalidRequestException", "SYSTEM rules take no TargetIps or ResolverEndpointId")
		}
		var targets []any
		if r53rHas(cfg, "TargetIps") {
			if targets, err = r53rCheckTargets(r53rList(cfg, "TargetIps")); err != nil {
				return nil, err
			}
			if len(targets) == 0 {
				return nil, r53rE("InvalidRequestException", "a %s rule needs at least one target", ruleType)
			}
		}
		if id := r53rStr(cfg, "ResolverEndpointId"); id != "" {
			if err := st.checkRuleEndpointLocked(ruleType, id); err != nil {
				return nil, err
			}
		}
		if targets != nil {
			r.rec["TargetIps"] = targets
		}
		r53rCopy(r.rec, cfg, "Name", "ResolverEndpointId")
		r.rec["Status"], r.settle, r.rec["ModificationTime"] = "UPDATING", "COMPLETE", r53rNow()
		return map[string]any{"ResolverRule": r.rec}, nil
	}
	h["DeleteResolverRule"] = func(st *r53rStore, in map[string]any) (map[string]any, *r53rErr) {
		r, err := st.ruleLocked(r53rStr(in, "ResolverRuleId"))
		if err != nil {
			return nil, err
		}
		if r.autodefined {
			return nil, r53rE("InvalidRequestException", "the autodefined rule %s can't be deleted", r53rAutodefinedID)
		}
		for _, a := range st.ruleAssocs {
			if a.rec["ResolverRuleId"] == r.rec["Id"] {
				return nil, r53rE("ResourceInUseException", "rule %v is still associated with VPC %v", r.rec["Id"], a.rec["VPCId"])
			}
		}
		out := map[string]any{}
		for k, v := range r.rec {
			out[k] = v
		}
		out["Status"] = "DELETING"
		delete(st.rules, r.rec["Id"].(string))
		st.forgetLocked(r.rec["Arn"].(string))
		return map[string]any{"ResolverRule": out}, nil
	}
	h["ListResolverRules"] = func(st *r53rStore, in map[string]any) (map[string]any, *r53rErr) {
		var items []map[string]any
		for _, r := range r53rSorted(st.rules, func(r *r53rRule) string { return r.rec["Id"].(string) }) {
			items = append(items, r.view())
		}
		filters := r53rNormaliseDomainFilters(r53rList(in, "Filters"))
		items, err := r53rFilter(items, filters, map[string]string{
			"CreatorRequestId": "CreatorRequestId", "DomainName": "DomainName", "Name": "Name",
			"ResolverEndpointId": "ResolverEndpointId", "Status": "Status", "Type": "RuleType",
		}, "InvalidParameterException")
		if err != nil {
			return nil, err
		}
		page, next, err := r53rPage(r53rAny(items), in, "InvalidNextTokenException")
		if err != nil {
			return nil, err
		}
		return map[string]any{"ResolverRules": page, "NextToken": r53rOmit(next), "MaxResults": r53rMaxResults(in)}, nil
	}

	h["AssociateResolverRule"] = func(st *r53rStore, in map[string]any) (map[string]any, *r53rErr) {
		r, err := st.ruleLocked(r53rStr(in, "ResolverRuleId"))
		if err != nil {
			return nil, err
		}
		if err := r53rCheckName(in, "Name", "InvalidParameterException"); err != nil {
			return nil, err
		}
		vpc := r53rStr(in, "VPCId")
		if !st.vpcExists(vpc) {
			return nil, r53rE("InvalidParameterException", "VPC %s doesn't exist", vpc)
		}
		for _, a := range st.ruleAssocs {
			if a.rec["VPCId"] != vpc {
				continue
			}
			if a.rec["ResolverRuleId"] == r.rec["Id"] {
				return nil, r53rE("ResourceExistsException", "rule %v is already associated with VPC %s", r.rec["Id"], vpc)
			}
			if other := st.rules[r53rStrS(a.rec["ResolverRuleId"])]; other != nil && other.rec["DomainName"] == r.rec["DomainName"] {
				return nil, r53rE("InvalidRequestException", "VPC %s already has a rule for %v", vpc, r.rec["DomainName"])
			}
		}
		if len(st.ruleAssocs) >= r53rMaxAssociations {
			return nil, r53rE("LimitExceededException", "an account can have at most %d rule associations", r53rMaxAssociations)
		}
		id := r53rID("rslvr-rrassoc-")
		rec := map[string]any{"Id": id, "ResolverRuleId": r.rec["Id"], "VPCId": vpc, "Status": "CREATING", "StatusMessage": "[Trace id: stackyard] Creating Association"}
		r53rCopy(rec, in, "Name")
		st.ruleAssocs[id] = &r53rRec{rec: rec, settle: "COMPLETE"}
		return map[string]any{"ResolverRuleAssociation": rec}, nil
	}
	h["DisassociateResolverRule"] = func(st *r53rStore, in map[string]any) (map[string]any, *r53rErr) {
		for id, a := range st.ruleAssocs {
			if a.rec["VPCId"] == r53rStr(in, "VPCId") && a.rec["ResolverRuleId"] == r53rStr(in, "ResolverRuleId") {
				out := map[string]any{}
				for k, v := range a.rec {
					out[k] = v
				}
				out["Status"] = "DELETING"
				delete(st.ruleAssocs, id)
				return map[string]any{"ResolverRuleAssociation": out}, nil
			}
		}
		return nil, r53rE("ResourceNotFoundException", "rule %s isn't associated with VPC %s", r53rStr(in, "ResolverRuleId"), r53rStr(in, "VPCId"))
	}
	h["GetResolverRuleAssociation"] = func(st *r53rStore, in map[string]any) (map[string]any, *r53rErr) {
		a := st.ruleAssocs[r53rStr(in, "ResolverRuleAssociationId")]
		if a == nil {
			return nil, r53rE("ResourceNotFoundException", "rule association %s doesn't exist", r53rStr(in, "ResolverRuleAssociationId"))
		}
		return map[string]any{"ResolverRuleAssociation": a.view()}, nil
	}
	h["ListResolverRuleAssociations"] = func(st *r53rStore, in map[string]any) (map[string]any, *r53rErr) {
		items, err := r53rFilter(r53rViews(st.ruleAssocs), r53rList(in, "Filters"), map[string]string{
			"Name": "Name", "ResolverRuleId": "ResolverRuleId", "Status": "Status", "VPCId": "VPCId",
		}, "InvalidParameterException")
		if err != nil {
			return nil, err
		}
		page, next, err := r53rPage(r53rAny(items), in, "InvalidNextTokenException")
		if err != nil {
			return nil, err
		}
		return map[string]any{"ResolverRuleAssociations": page, "NextToken": r53rOmit(next), "MaxResults": r53rMaxResults(in)}, nil
	}

	h["CreateResolverQueryLogConfig"] = func(st *r53rStore, in map[string]any) (map[string]any, *r53rErr) {
		if err := r53rCheckName(in, "Name", "InvalidParameterException"); err != nil {
			return nil, err
		}
		for _, q := range st.qlcs {
			if q.rec["CreatorRequestId"] == r53rStr(in, "CreatorRequestId") {
				return nil, r53rE("ResourceExistsException", "a query logging configuration with CreatorRequestId %s already exists", r53rStr(in, "CreatorRequestId"))
			}
		}
		if len(st.qlcs) >= r53rMaxQLCs {
			return nil, r53rE("LimitExceededException", "an account can have at most %d query logging configurations", r53rMaxQLCs)
		}
		kind, err := st.destinationType(r53rStr(in, "DestinationArn"))
		if err != nil {
			return nil, err
		}
		id := r53rID("rslvr-rqlc-")
		rec := map[string]any{
			"Id": id, "Arn": r53rARN("resolver-query-log-config", id), "Name": r53rStr(in, "Name"),
			"DestinationArn": r53rStr(in, "DestinationArn"), "DestinationType": kind, "CreatorRequestId": r53rStr(in, "CreatorRequestId"),
			"OwnerId": r53rAccountID, "Status": "CREATING", "ShareStatus": "NOT_SHARED", "AssociationCount": 0, "CreationTime": r53rNow(),
		}
		if err := st.tagsLocked(rec["Arn"].(string), r53rList(in, "Tags")); err != nil {
			return nil, err
		}
		st.qlcs[id] = &r53rRec{rec: rec, settle: "CREATED"}
		return map[string]any{"ResolverQueryLogConfig": rec}, nil
	}
	h["GetResolverQueryLogConfig"] = func(st *r53rStore, in map[string]any) (map[string]any, *r53rErr) {
		q, err := st.qlcLocked(r53rStr(in, "ResolverQueryLogConfigId"))
		if err != nil {
			return nil, err
		}
		return map[string]any{"ResolverQueryLogConfig": q.view()}, nil
	}
	h["DeleteResolverQueryLogConfig"] = func(st *r53rStore, in map[string]any) (map[string]any, *r53rErr) {
		q, err := st.qlcLocked(r53rStr(in, "ResolverQueryLogConfigId"))
		if err != nil {
			return nil, err
		}
		for _, a := range st.qlcAssocs {
			if a.rec["ResolverQueryLogConfigId"] == q.rec["Id"] {
				return nil, r53rE("InvalidRequestException", "configuration %v is still associated with %v", q.rec["Id"], a.rec["ResourceId"])
			}
		}
		out := map[string]any{}
		for k, v := range q.view() {
			out[k] = v
		}
		out["Status"] = "DELETING"
		delete(st.qlcs, q.rec["Id"].(string))
		st.forgetLocked(q.rec["Arn"].(string))
		return map[string]any{"ResolverQueryLogConfig": out}, nil
	}
	h["ListResolverQueryLogConfigs"] = func(st *r53rStore, in map[string]any) (map[string]any, *r53rErr) {
		all := r53rViews(st.qlcs)
		items, err := r53rFilter(all, r53rList(in, "Filters"), r53rQLCFields, "InvalidParameterException")
		if err != nil {
			return nil, err
		}
		if err := r53rSortItems(items, in, r53rQLCFields); err != nil {
			return nil, err
		}
		page, next, err := r53rPage(r53rAny(items), in, "InvalidNextTokenException")
		if err != nil {
			return nil, err
		}
		return map[string]any{"ResolverQueryLogConfigs": page, "NextToken": r53rOmit(next), "TotalCount": len(all), "TotalFilteredCount": len(items)}, nil
	}
	h["AssociateResolverQueryLogConfig"] = func(st *r53rStore, in map[string]any) (map[string]any, *r53rErr) {
		q, err := st.qlcLocked(r53rStr(in, "ResolverQueryLogConfigId"))
		if err != nil {
			return nil, err
		}
		vpc := r53rStr(in, "ResourceId")
		if !st.vpcExists(vpc) {
			return nil, r53rE("InvalidParameterException", "VPC %s doesn't exist", vpc)
		}
		for _, a := range st.qlcAssocs {
			if a.rec["ResolverQueryLogConfigId"] == q.rec["Id"] && a.rec["ResourceId"] == vpc {
				return nil, r53rE("ResourceExistsException", "configuration %v is already associated with %s", q.rec["Id"], vpc)
			}
		}
		if len(st.qlcAssocs) >= r53rMaxAssociations {
			return nil, r53rE("LimitExceededException", "an account can have at most %d query logging associations", r53rMaxAssociations)
		}
		id := r53rID("rslvr-qlcassoc-")
		rec := map[string]any{"Id": id, "ResolverQueryLogConfigId": q.rec["Id"], "ResourceId": vpc, "Status": "CREATING", "Error": "NONE", "ErrorMessage": "", "CreationTime": r53rNow()}
		st.qlcAssocs[id] = &r53rRec{rec: rec, settle: "ACTIVE"}
		q.rec["AssociationCount"] = r53rCount(st.qlcAssocs, q.rec["Id"])
		return map[string]any{"ResolverQueryLogConfigAssociation": rec}, nil
	}
	h["DisassociateResolverQueryLogConfig"] = func(st *r53rStore, in map[string]any) (map[string]any, *r53rErr) {
		for id, a := range st.qlcAssocs {
			if a.rec["ResolverQueryLogConfigId"] == r53rStr(in, "ResolverQueryLogConfigId") && a.rec["ResourceId"] == r53rStr(in, "ResourceId") {
				out := map[string]any{}
				for k, v := range a.view() {
					out[k] = v
				}
				out["Status"] = "DELETING"
				delete(st.qlcAssocs, id)
				if q := st.qlcs[r53rStr(in, "ResolverQueryLogConfigId")]; q != nil {
					q.rec["AssociationCount"] = r53rCount(st.qlcAssocs, q.rec["Id"])
				}
				return map[string]any{"ResolverQueryLogConfigAssociation": out}, nil
			}
		}
		return nil, r53rE("ResourceNotFoundException", "configuration %s isn't associated with %s", r53rStr(in, "ResolverQueryLogConfigId"), r53rStr(in, "ResourceId"))
	}
	h["GetResolverQueryLogConfigAssociation"] = func(st *r53rStore, in map[string]any) (map[string]any, *r53rErr) {
		a := st.qlcAssocs[r53rStr(in, "ResolverQueryLogConfigAssociationId")]
		if a == nil {
			return nil, r53rE("ResourceNotFoundException", "query logging association %s doesn't exist", r53rStr(in, "ResolverQueryLogConfigAssociationId"))
		}
		return map[string]any{"ResolverQueryLogConfigAssociation": a.view()}, nil
	}
	h["ListResolverQueryLogConfigAssociations"] = func(st *r53rStore, in map[string]any) (map[string]any, *r53rErr) {
		all := r53rViews(st.qlcAssocs)
		items, err := r53rFilter(all, r53rList(in, "Filters"), r53rQLCAssocFields, "InvalidParameterException")
		if err != nil {
			return nil, err
		}
		if err := r53rSortItems(items, in, r53rQLCAssocFields); err != nil {
			return nil, err
		}
		page, next, err := r53rPage(r53rAny(items), in, "InvalidParameterException")
		if err != nil {
			return nil, err
		}
		return map[string]any{"ResolverQueryLogConfigAssociations": page, "NextToken": r53rOmit(next), "TotalCount": len(all), "TotalFilteredCount": len(items)}, nil
	}

	h["GetResolverConfig"] = func(st *r53rStore, in map[string]any) (map[string]any, *r53rErr) {
		rec, err := r53rResolverConfig.get(st, r53rStr(in, "ResourceId"))
		return map[string]any{"ResolverConfig": rec}, err
	}
	h["UpdateResolverConfig"] = func(st *r53rStore, in map[string]any) (map[string]any, *r53rErr) {
		rec, err := r53rResolverConfig.update(st, r53rStr(in, "ResourceId"), r53rStr(in, "AutodefinedReverseFlag"))
		return map[string]any{"ResolverConfig": rec}, err
	}
	h["ListResolverConfigs"] = func(st *r53rStore, in map[string]any) (map[string]any, *r53rErr) {
		page, next, err := r53rResolverConfig.list(st, in)
		return map[string]any{"ResolverConfigs": page, "NextToken": r53rOmit(next)}, err
	}
	h["GetResolverDnssecConfig"] = func(st *r53rStore, in map[string]any) (map[string]any, *r53rErr) {
		rec, err := r53rDnssecConfig.get(st, r53rStr(in, "ResourceId"))
		return map[string]any{"ResolverDNSSECConfig": rec}, err
	}
	h["UpdateResolverDnssecConfig"] = func(st *r53rStore, in map[string]any) (map[string]any, *r53rErr) {
		rec, err := r53rDnssecConfig.update(st, r53rStr(in, "ResourceId"), r53rStr(in, "Validation"))
		return map[string]any{"ResolverDNSSECConfig": rec}, err
	}
	h["ListResolverDnssecConfigs"] = func(st *r53rStore, in map[string]any) (map[string]any, *r53rErr) {
		var all []map[string]any
		for _, vpc := range st.ec2.DescribeVpcs(nil) {
			all = append(all, r53rDnssecConfig.view(r53rDnssecConfig.getLocked(st, vpc.ID)))
		}
		items, err := r53rFilter(all, r53rList(in, "Filters"), map[string]string{
			"Id": "Id", "OwnerId": "OwnerId", "ResourceId": "ResourceId", "ValidationStatus": "ValidationStatus",
		}, "InvalidParameterException")
		if err != nil {
			return nil, err
		}
		page, next, err := r53rPage(r53rAny(items), in, "InvalidNextTokenException")
		return map[string]any{"ResolverDnssecConfigs": page, "NextToken": r53rOmit(next)}, err
	}

	h["CreateOutpostResolver"] = func(st *r53rStore, in map[string]any) (map[string]any, *r53rErr) {
		for _, o := range st.outposts {
			if o.rec["CreatorRequestId"] == r53rStr(in, "CreatorRequestId") {
				return map[string]any{"OutpostResolver": o.view()}, nil // the request id makes retries safe
			}
		}
		if len(st.outposts) >= r53rMaxOutposts {
			return nil, r53rE("ServiceQuotaExceededException", "an account can have at most %d Outposts resolvers", r53rMaxOutposts)
		}
		count := r53rMinOutpostCount
		if n, ok := r53rInt(in, "InstanceCount"); ok {
			count = n
		}
		if count < r53rMinOutpostCount || count > r53rMaxOutpostCount {
			return nil, r53rE("ValidationException", "InstanceCount must be between %d and %d", r53rMinOutpostCount, r53rMaxOutpostCount)
		}
		id := r53rID("rslvr-or-")
		now := r53rNow()
		rec := map[string]any{
			"Id": id, "Arn": r53rARN("outpost-resolver", id), "CreatorRequestId": r53rStr(in, "CreatorRequestId"),
			"Name": r53rStr(in, "Name"), "InstanceCount": count, "PreferredInstanceType": r53rStr(in, "PreferredInstanceType"),
			"OutpostArn": r53rStr(in, "OutpostArn"), "Status": "CREATING", "StatusMessage": "Creating the Outposts resolver",
			"CreationTime": now, "ModificationTime": now,
		}
		if err := st.tagsLocked(rec["Arn"].(string), r53rList(in, "Tags")); err != nil {
			return nil, err
		}
		st.outposts[id] = &r53rRec{rec: rec, settle: "OPERATIONAL"}
		return map[string]any{"OutpostResolver": rec}, nil
	}
	h["GetOutpostResolver"] = func(st *r53rStore, in map[string]any) (map[string]any, *r53rErr) {
		o, err := st.outpostLocked(r53rStr(in, "Id"))
		if err != nil {
			return nil, err
		}
		return map[string]any{"OutpostResolver": o.view()}, nil
	}
	h["UpdateOutpostResolver"] = func(st *r53rStore, in map[string]any) (map[string]any, *r53rErr) {
		o, err := st.outpostLocked(r53rStr(in, "Id"))
		if err != nil {
			return nil, err
		}
		if n, ok := r53rInt(in, "InstanceCount"); ok && (n < r53rMinOutpostCount || n > r53rMaxOutpostCount) {
			return nil, r53rE("ValidationException", "InstanceCount must be between %d and %d", r53rMinOutpostCount, r53rMaxOutpostCount)
		}
		r53rCopy(o.rec, in, "Name", "InstanceCount", "PreferredInstanceType")
		o.rec["Status"], o.settle, o.rec["ModificationTime"] = "UPDATING", "OPERATIONAL", r53rNow()
		return map[string]any{"OutpostResolver": o.rec}, nil
	}
	h["DeleteOutpostResolver"] = func(st *r53rStore, in map[string]any) (map[string]any, *r53rErr) {
		o, err := st.outpostLocked(r53rStr(in, "Id"))
		if err != nil {
			return nil, err
		}
		for _, e := range st.endpoints {
			if e.rec["OutpostArn"] == o.rec["OutpostArn"] && st.outpostsOnLocked(r53rStrS(o.rec["OutpostArn"])) == 1 {
				return nil, r53rE("ConflictException", "endpoint %v still runs on this Outposts resolver", e.rec["Id"])
			}
		}
		out := map[string]any{}
		for k, v := range o.view() {
			out[k] = v
		}
		out["Status"] = "DELETING"
		delete(st.outposts, o.rec["Id"].(string))
		st.forgetLocked(o.rec["Arn"].(string))
		return map[string]any{"OutpostResolver": out}, nil
	}
	h["ListOutpostResolvers"] = func(st *r53rStore, in map[string]any) (map[string]any, *r53rErr) {
		var items []any
		for _, v := range r53rViews(st.outposts) {
			if arn := r53rStr(in, "OutpostArn"); arn == "" || v["OutpostArn"] == arn {
				items = append(items, v)
			}
		}
		page, next, err := r53rPage(items, in, "ValidationException")
		return map[string]any{"OutpostResolvers": page, "NextToken": r53rOmit(next)}, err
	}

	r53rPolicyHandlers(h, "PutResolverRulePolicy", "GetResolverRulePolicy", "ResolverRulePolicy",
		func(st *r53rStore, arn string) bool {
			for _, r := range st.rules {
				if r.rec["Arn"] == arn && !r.autodefined {
					return true
				}
			}
			return false
		}, "UnknownResourceException", "InvalidPolicyDocument")
	r53rPolicyHandlers(h, "PutResolverQueryLogConfigPolicy", "GetResolverQueryLogConfigPolicy", "ResolverQueryLogConfigPolicy",
		func(st *r53rStore, arn string) bool {
			for _, q := range st.qlcs {
				if q.rec["Arn"] == arn {
					return true
				}
			}
			return false
		}, "UnknownResourceException", "InvalidPolicyDocument")
}

// r53rPolicyHandlers registers a Put/Get resource-policy pair.
func r53rPolicyHandlers(h map[string]r53rHandler, put, get, member string, exists func(*r53rStore, string) bool, missing, badDoc string) {
	h[put] = func(st *r53rStore, in map[string]any) (map[string]any, *r53rErr) {
		arn := r53rStr(in, "Arn")
		if !exists(st, arn) {
			return nil, r53rE(missing, "resource %s doesn't exist", arn)
		}
		if !r53rPolicyJSON(r53rStr(in, member)) {
			return nil, r53rE(badDoc, "the policy must be a JSON document with a Statement")
		}
		st.policies[arn] = r53rStr(in, member)
		return map[string]any{"ReturnValue": true}, nil
	}
	h[get] = func(st *r53rStore, in map[string]any) (map[string]any, *r53rErr) {
		arn := r53rStr(in, "Arn")
		if !exists(st, arn) {
			return nil, r53rE(missing, "resource %s doesn't exist", arn)
		}
		return map[string]any{member: r53rOmit(st.policies[arn])}, nil
	}
}

func (e *r53rEndpoint) hasIP(id string) bool {
	for _, ip := range e.ips {
		if ip["IpId"] == id {
			return true
		}
	}
	return false
}

// outRaw is out without settling a just-set transitional status.
func (e *r53rEndpoint) outRaw() map[string]any {
	m := map[string]any{}
	for k, v := range e.rec {
		m[k] = v
	}
	m["IpAddressCount"] = len(e.ips)
	return m
}

func (st *r53rStore) outpostLocked(id string) (*r53rRec, *r53rErr) {
	if o := st.outposts[id]; o != nil {
		return o, nil
	}
	return nil, r53rE("ResourceNotFoundException", "Outposts resolver %s doesn't exist", id)
}

func (st *r53rStore) outpostByARNLocked(arn string) *r53rRec {
	for _, o := range st.outposts {
		if o.rec["OutpostArn"] == arn {
			return o
		}
	}
	return nil
}

func (st *r53rStore) outpostsOnLocked(arn string) int {
	n := 0
	for _, o := range st.outposts {
		if o.rec["OutpostArn"] == arn {
			n++
		}
	}
	return n
}

// r53rViews returns the settled records of m, sorted by Id.
func r53rViews(m map[string]*r53rRec) []map[string]any {
	var out []map[string]any
	for _, r := range r53rSorted(m, func(r *r53rRec) string { return r53rStrS(r.rec["Id"]) }) {
		out = append(out, r.view())
	}
	return out
}

func r53rCount(m map[string]*r53rRec, configID any) int {
	n := 0
	for _, a := range m {
		if a.rec["ResolverQueryLogConfigId"] == configID {
			n++
		}
	}
	return n
}

func r53rStrS(v any) string {
	s, _ := v.(string)
	return s
}

func r53rStringsAny(ss []string) []any {
	out := make([]any, len(ss))
	for i, s := range ss {
		out[i] = s
	}
	return out
}

func r53rMaxResults(in map[string]any) int {
	if n, ok := r53rInt(in, "MaxResults"); ok && n > 0 {
		return n
	}
	return r53rDefaultPage
}

// r53rNormaliseDomainFilters adds the trailing dot rule domains are stored with.
func r53rNormaliseDomainFilters(filters []any) []any {
	out := make([]any, 0, len(filters))
	for _, f := range filters {
		fm, _ := f.(map[string]any)
		if r53rStr(fm, "Name") != "DomainName" {
			out = append(out, f)
			continue
		}
		var values []any
		for _, v := range r53rStrings(fm, "Values") {
			if d, ok := r53rDomain(v); ok {
				v = d
			}
			values = append(values, v)
		}
		out = append(out, map[string]any{"Name": "DomainName", "Values": values})
	}
	return out
}
