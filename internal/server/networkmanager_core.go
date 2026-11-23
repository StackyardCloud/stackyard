package server

import (
	"net"
	"slices"
	"strconv"
	"strings"
	"time"
)

func init() {
	for name, h := range map[string]nmHandler{
		"CreateCoreNetwork": nmCreateCoreNetwork, "GetCoreNetwork": nmGetCoreNetwork, "ListCoreNetworks": nmListCoreNetworks,
		"UpdateCoreNetwork": nmUpdateCoreNetwork, "DeleteCoreNetwork": nmDeleteCoreNetwork,
		"PutCoreNetworkPolicy": nmPutCoreNetworkPolicy, "GetCoreNetworkPolicy": nmGetCoreNetworkPolicy,
		"ListCoreNetworkPolicyVersions": nmListCoreNetworkPolicyVersions, "DeleteCoreNetworkPolicyVersion": nmDeleteCoreNetworkPolicyVersion,
		"RestoreCoreNetworkPolicyVersion": nmRestoreCoreNetworkPolicyVersion, "ExecuteCoreNetworkChangeSet": nmExecuteCoreNetworkChangeSet,
		"GetCoreNetworkChangeSet": nmGetCoreNetworkChangeSet, "GetCoreNetworkChangeEvents": nmGetCoreNetworkChangeEvents,
		"CreateVpcAttachment": nmCreateVpcAttachment, "CreateConnectAttachment": nmCreateConnectAttachment,
		"CreateSiteToSiteVpnAttachment": nmCreateVpnAttachment, "CreateTransitGatewayRouteTableAttachment": nmCreateTGWRouteTableAttachment,
		"CreateDirectConnectGatewayAttachment": nmCreateDXAttachment,
		"GetVpcAttachment":                     nmGetAttachmentOf("VPC", "VpcAttachment"), "GetConnectAttachment": nmGetAttachmentOf("CONNECT", "ConnectAttachment"),
		"GetSiteToSiteVpnAttachment":            nmGetAttachmentOf("SITE_TO_SITE_VPN", "SiteToSiteVpnAttachment"),
		"GetTransitGatewayRouteTableAttachment": nmGetAttachmentOf("TRANSIT_GATEWAY_ROUTE_TABLE", "TransitGatewayRouteTableAttachment"),
		"GetDirectConnectGatewayAttachment":     nmGetAttachmentOf("DIRECT_CONNECT_GATEWAY", "DirectConnectGatewayAttachment"),
		"UpdateVpcAttachment":                   nmUpdateVpcAttachment, "UpdateDirectConnectGatewayAttachment": nmUpdateDXAttachment,
		"ListAttachments": nmListAttachments, "AcceptAttachment": nmAcceptAttachment, "RejectAttachment": nmRejectAttachment,
		"DeleteAttachment":  nmDeleteAttachment,
		"CreateConnectPeer": nmCreateConnectPeer, "GetConnectPeer": nmGetConnectPeer, "ListConnectPeers": nmListConnectPeers,
		"DeleteConnectPeer":           nmDeleteConnectPeer,
		"CreateTransitGatewayPeering": nmCreateTGWPeering, "GetTransitGatewayPeering": nmGetTGWPeering, "ListPeerings": nmListPeerings,
		"DeletePeering":                          nmDeletePeering,
		"CreateCoreNetworkPrefixListAssociation": nmCreatePrefixListAssociation, "DeleteCoreNetworkPrefixListAssociation": nmDeletePrefixListAssociation,
		"ListCoreNetworkPrefixListAssociations": nmListPrefixListAssociations,
		"PutAttachmentRoutingPolicyLabel":       nmPutRoutingPolicyLabel, "RemoveAttachmentRoutingPolicyLabel": nmRemoveRoutingPolicyLabel,
		"ListAttachmentRoutingPolicyAssociations": nmListRoutingPolicyAssociations,
		"ListAttachmentNatConfigurations":         nmListNatConfigurations, "UpdateAttachmentNatConfiguration": nmUpdateNatConfiguration,
		"ListCoreNetworkRoutingInformation": nmListCoreNetworkRoutingInformation,
	} {
		nmHandlers[name] = h
	}
	nmPreChecks["CreateVpcAttachment"] = (*Server).nmCheckVpcAttachment
	nmPreChecks["UpdateVpcAttachment"] = (*Server).nmCheckAddedSubnets
	nmPreChecks["CreateSiteToSiteVpnAttachment"] = (*Server).nmCheckVpnConnection
	nmPreChecks["CreateTransitGatewayPeering"] = (*Server).nmCheckTransitGatewayInput
}

// ---- core networks ----

func (st *nmStore) coreNetwork(id string) (*nmRec, *nmErr) {
	return st.get("core-network", "core-network", id)
}

// live returns the core network's executed (LIVE) policy, if any.
func (st *nmStore) live(coreID string) *nmPolicy {
	for _, p := range st.policies[coreID] {
		if p.alias == "LIVE" {
			return p
		}
	}
	return nil
}

func (st *nmStore) liveDoc(coreID string) *nmPolicyDoc {
	if p := st.live(coreID); p != nil {
		return p.parsed
	}
	return nil
}

func (st *nmStore) renderCore(rec *nmRec) map[string]any {
	out := st.render(rec)
	doc := st.liveDoc(rec.id)
	out["Segments"], out["Edges"], out["NetworkFunctionGroups"] = nmSegmentsOut(doc), nmEdgesOut(doc), []any{}
	out["OwnerAccountId"] = nmAccount
	return out
}

func nmCreateCoreNetwork(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	if id, ok := st.token("CreateCoreNetwork", nmStr(in, "ClientToken")); ok {
		return map[string]any{"CoreNetwork": st.renderCore(st.recs[id])}, nil
	}
	gnID := nmStr(in, "GlobalNetworkId")
	if _, err := st.globalNetwork(gnID); err != nil {
		return nil, err
	}
	var doc *nmPolicyDoc
	if raw, ok := nmOptStr(in, "PolicyDocument"); ok {
		var err *nmErr
		if doc, err = nmParsePolicy(raw); err != nil {
			return nil, err
		}
	}
	if err := nmCheckCap(st, "core-network"); err != nil {
		return nil, err
	}
	if err := nmCheckTags(nil, nmTags(in)); err != nil {
		return nil, err
	}
	id := "core-network-" + nmHex(17)
	data := nmCopyFields(in, "GlobalNetworkId", "Description")
	data["CoreNetworkId"], data["CoreNetworkArn"] = id, nmArn("core-network/"+id)
	rec := st.newRec("core-network", id, nmArn("core-network/"+id), gnID, data, "CREATING", "AVAILABLE", nmTags(in))
	if doc != nil {
		st.addPolicy(id, nmStr(in, "PolicyDocument"), "", doc)
	}
	st.rememberToken("CreateCoreNetwork", nmStr(in, "ClientToken"), id)
	return map[string]any{"CoreNetwork": st.renderCore(rec)}, nil
}

func nmGetCoreNetwork(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	rec, err := st.coreNetwork(nmStr(in, "CoreNetworkId"))
	if err != nil {
		return nil, err
	}
	return map[string]any{"CoreNetwork": st.renderCore(rec)}, nil
}

func nmListCoreNetworks(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	var items []any
	for _, rec := range st.list("core-network", nil) {
		items = append(items, st.renderCore(rec))
	}
	page, next, err := nmPage(in, items)
	if err != nil {
		return nil, err
	}
	return map[string]any{"CoreNetworks": page, "NextToken": nmNonEmpty(next)}, nil
}

func nmUpdateCoreNetwork(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	rec, err := st.coreNetwork(nmStr(in, "CoreNetworkId"))
	if err != nil {
		return nil, err
	}
	if d, ok := nmOptStr(in, "Description"); ok {
		rec.data["Description"] = d
	}
	rec.state, rec.settle, rec.updated = "UPDATING", "AVAILABLE", nmNow()
	return map[string]any{"CoreNetwork": st.renderCore(rec)}, nil
}

func nmDeleteCoreNetwork(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	rec, err := st.coreNetwork(nmStr(in, "CoreNetworkId"))
	if err != nil {
		return nil, err
	}
	for _, other := range st.recs {
		if (other.kind == "attachment" || other.kind == "peering" || other.kind == "connect-peer") && other.parent == rec.id {
			return nil, nmConflict("core-network", rec.id, "Core network %s still has %s %s; delete it first.", rec.id, other.kind, other.id)
		}
	}
	rec.state, rec.settle = "DELETING", ""
	out := st.renderCore(rec)
	delete(st.policies, rec.id)
	delete(st.prefixLists, rec.id)
	delete(st.resPolicies, rec.arn)
	st.deleteRec(rec)
	return map[string]any{"CoreNetwork": out}, nil
}

// ---- policies and change sets ----

func (st *nmStore) addPolicy(coreID, raw, description string, doc *nmPolicyDoc) *nmPolicy {
	// Trim first: it may delete from this core network's slice, so read the
	// versions only afterwards (a slice read before would be stale).
	st.trimPolicyBytes(len(raw))
	versions := st.policies[coreID]
	next := 1
	if n := len(versions); n > 0 {
		next = versions[n-1].version + 1
	}
	for _, p := range versions {
		if p.alias == "LATEST" {
			p.alias = ""
		}
	}
	p := &nmPolicy{version: next, alias: "LATEST", description: description, document: raw,
		changeSet: "PENDING_GENERATION", created: nmNow(), parsed: doc}
	versions = append(versions, p)
	if len(versions) > nmMaxVersion { // drop the oldest version that isn't LIVE
		for i, old := range versions {
			if old.alias != "LIVE" {
				versions = slices.Delete(versions, i, i+1)
				break
			}
		}
	}
	st.policies[coreID] = versions
	return p
}

// settlePolicy moves a freshly generated change set to READY_TO_EXECUTE on
// the first read after the version was created.
func nmSettlePolicy(p *nmPolicy) {
	if p.changeSet == "PENDING_GENERATION" {
		p.changeSet = "READY_TO_EXECUTE"
	}
}

func (st *nmStore) policyVersion(coreID string, version int) (*nmPolicy, *nmErr) {
	for _, p := range st.policies[coreID] {
		if p.version == version {
			nmSettlePolicy(p)
			return p, nil
		}
	}
	return nil, nmNotFound("core-network-policy", coreID+"/"+strconv.Itoa(version))
}

func nmPolicyOut(coreID string, p *nmPolicy, withDocument bool) map[string]any {
	out := map[string]any{"CoreNetworkId": coreID, "PolicyVersionId": float64(p.version), "Description": p.description,
		"CreatedAt": nmTime(p.created), "ChangeSetState": p.changeSet}
	if p.alias != "" {
		out["Alias"] = p.alias
	}
	if withDocument {
		out["PolicyDocument"], out["PolicyErrors"] = p.document, []any{}
	}
	return out
}

func nmPutCoreNetworkPolicy(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	coreID := nmStr(in, "CoreNetworkId")
	if _, err := st.coreNetwork(coreID); err != nil {
		return nil, err
	}
	versions := st.policies[coreID]
	if latest, ok := nmInt(in, "LatestVersionId"); ok {
		if n := len(versions); n == 0 || versions[n-1].version != latest {
			return nil, nmConflict("core-network-policy", coreID, "LatestVersionId %d isn't the latest policy version.", latest)
		}
	}
	doc, err := nmParsePolicy(nmStr(in, "PolicyDocument"))
	if err != nil {
		return nil, err
	}
	p := st.addPolicy(coreID, nmStr(in, "PolicyDocument"), nmStr(in, "Description"), doc)
	return map[string]any{"CoreNetworkPolicy": nmPolicyOut(coreID, p, true)}, nil
}

func nmGetCoreNetworkPolicy(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	coreID := nmStr(in, "CoreNetworkId")
	if _, err := st.coreNetwork(coreID); err != nil {
		return nil, err
	}
	if v, ok := nmInt(in, "PolicyVersionId"); ok {
		p, err := st.policyVersion(coreID, v)
		if err != nil {
			return nil, err
		}
		return map[string]any{"CoreNetworkPolicy": nmPolicyOut(coreID, p, true)}, nil
	}
	alias := "LIVE"
	if a := nmStr(in, "Alias"); a != "" {
		alias = a
	}
	versions := st.policies[coreID]
	var p *nmPolicy
	if alias == "LATEST" && len(versions) > 0 {
		p = versions[len(versions)-1]
	} else if alias == "LIVE" {
		p = st.live(coreID)
	}
	if p == nil {
		return nil, nmNotFound("core-network-policy", coreID+"/"+alias)
	}
	nmSettlePolicy(p)
	return map[string]any{"CoreNetworkPolicy": nmPolicyOut(coreID, p, true)}, nil
}

func nmListCoreNetworkPolicyVersions(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	coreID := nmStr(in, "CoreNetworkId")
	if _, err := st.coreNetwork(coreID); err != nil {
		return nil, err
	}
	var items []any
	for _, p := range st.policies[coreID] {
		nmSettlePolicy(p)
		items = append(items, nmPolicyOut(coreID, p, false))
	}
	page, next, err := nmPage(in, items)
	if err != nil {
		return nil, err
	}
	return map[string]any{"CoreNetworkPolicyVersions": page, "NextToken": nmNonEmpty(next)}, nil
}

func nmDeleteCoreNetworkPolicyVersion(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	coreID := nmStr(in, "CoreNetworkId")
	if _, err := st.coreNetwork(coreID); err != nil {
		return nil, err
	}
	v, _ := nmInt(in, "PolicyVersionId")
	p, err := st.policyVersion(coreID, v)
	if err != nil {
		return nil, err
	}
	if p.alias == "LIVE" {
		return nil, nmConflict("core-network-policy", coreID, "Policy version %d is LIVE and can't be deleted.", v)
	}
	st.policies[coreID] = slices.DeleteFunc(st.policies[coreID], func(x *nmPolicy) bool { return x == p })
	return map[string]any{"CoreNetworkPolicy": nmPolicyOut(coreID, p, true)}, nil
}

func nmRestoreCoreNetworkPolicyVersion(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	coreID := nmStr(in, "CoreNetworkId")
	if _, err := st.coreNetwork(coreID); err != nil {
		return nil, err
	}
	v, _ := nmInt(in, "PolicyVersionId")
	old, err := st.policyVersion(coreID, v)
	if err != nil {
		return nil, err
	}
	p := st.addPolicy(coreID, old.document, old.description, old.parsed)
	return map[string]any{"CoreNetworkPolicy": nmPolicyOut(coreID, p, true)}, nil
}

func nmExecuteCoreNetworkChangeSet(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	coreID := nmStr(in, "CoreNetworkId")
	core, err := st.coreNetwork(coreID)
	if err != nil {
		return nil, err
	}
	v, _ := nmInt(in, "PolicyVersionId")
	p, err := st.policyVersion(coreID, v)
	if err != nil {
		return nil, err
	}
	if p.changeSet != "READY_TO_EXECUTE" {
		return nil, nmValidation("The change set for policy version %d is %s, not READY_TO_EXECUTE.", v, p.changeSet)
	}
	prev := st.live(coreID)
	for _, other := range st.policies[coreID] {
		if other.alias == "LIVE" {
			other.alias = ""
		}
		if other != p && other.version < p.version && other.changeSet == "READY_TO_EXECUTE" {
			other.changeSet = "OUT_OF_DATE"
		}
	}
	if prev != nil {
		p.prevLive = prev.parsed
	}
	p.alias, p.changeSet, p.executedAt = "LIVE", "EXECUTION_SUCCEEDED", nmNow()
	core.state, core.settle, core.updated = "UPDATING", "AVAILABLE", nmNow()
	return map[string]any{}, nil
}

func nmGetCoreNetworkChangeSet(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	coreID := nmStr(in, "CoreNetworkId")
	if _, err := st.coreNetwork(coreID); err != nil {
		return nil, err
	}
	v, _ := nmInt(in, "PolicyVersionId")
	p, err := st.policyVersion(coreID, v)
	if err != nil {
		return nil, err
	}
	base := p.prevLive
	if p.alias != "LIVE" {
		base = st.liveDoc(coreID)
	}
	page, next, err := nmPage(in, nmChanges(base, p.parsed))
	if err != nil {
		return nil, err
	}
	return map[string]any{"CoreNetworkChanges": page, "NextToken": nmNonEmpty(next)}, nil
}

func nmGetCoreNetworkChangeEvents(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	coreID := nmStr(in, "CoreNetworkId")
	if _, err := st.coreNetwork(coreID); err != nil {
		return nil, err
	}
	v, _ := nmInt(in, "PolicyVersionId")
	p, err := st.policyVersion(coreID, v)
	if err != nil {
		return nil, err
	}
	var events []any
	if !p.executedAt.IsZero() {
		for _, c := range nmChanges(p.prevLive, p.parsed) {
			cm := c.(map[string]any)
			values := map[string]any{}
			if cm["Type"] == "CORE_NETWORK_SEGMENT" {
				values["SegmentName"] = cm["Identifier"]
			} else {
				values["EdgeLocation"] = cm["Identifier"]
			}
			events = append(events, map[string]any{"Type": cm["Type"], "Action": cm["Action"], "IdentifierPath": cm["IdentifierPath"],
				"EventTime": nmTime(p.executedAt), "Status": "COMPLETE", "Values": values})
		}
	}
	page, next, err := nmPage(in, events)
	if err != nil {
		return nil, err
	}
	return map[string]any{"CoreNetworkChangeEvents": page, "NextToken": nmNonEmpty(next)}, nil
}

// ---- attachments ----

var nmAttachmentKeys = map[string]string{
	"VPC": "VpcAttachment", "CONNECT": "ConnectAttachment", "SITE_TO_SITE_VPN": "SiteToSiteVpnAttachment",
	"TRANSIT_GATEWAY_ROUTE_TABLE": "TransitGatewayRouteTableAttachment", "DIRECT_CONNECT_GATEWAY": "DirectConnectGatewayAttachment",
}

var nmAttachmentTypeNames = map[string]string{
	"VPC": "vpc", "CONNECT": "connect", "SITE_TO_SITE_VPN": "site-to-site-vpn",
	"TRANSIT_GATEWAY_ROUTE_TABLE": "transit-gateway-route-table", "DIRECT_CONNECT_GATEWAY": "direct-connect-gateway",
}

func (st *nmStore) attachment(id string) (*nmRec, *nmErr) {
	return st.get("attachment", "attachment", id)
}

func (st *nmStore) attachmentBase(rec *nmRec) map[string]any {
	out := st.render(rec)
	out["UpdatedAt"] = nmTime(rec.updated)
	return out
}

func (st *nmStore) attachmentOut(rec *nmRec) map[string]any {
	out := make(map[string]any, len(rec.ext)+1)
	for k, v := range rec.ext {
		out[k] = v
	}
	out["Attachment"] = st.attachmentBase(rec)
	return map[string]any{nmAttachmentKeys[nmStr(rec.data, "AttachmentType")]: out}
}

// newAttachment validates the edge against the core network's LIVE policy,
// maps the attachment to a segment and stores it.
func (st *nmStore) newAttachment(op string, in map[string]any, attType, edge string, edges []string, resourceArn, resourceID string) (*nmRec, *nmErr) {
	coreID := nmStr(in, "CoreNetworkId")
	core, err := st.coreNetwork(coreID)
	if err != nil {
		return nil, err
	}
	doc := st.liveDoc(coreID)
	if doc == nil {
		return nil, nmValidation("Core network %s has no LIVE policy; execute a policy change set first.", coreID)
	}
	check := edges
	if edge != "" {
		check = []string{edge}
	}
	for _, e := range check {
		if doc.edge(e) == nil {
			return nil, nmValidation("Edge location %s isn't in the LIVE policy of core network %s.", e, coreID)
		}
	}
	if err := nmCheckCap(st, "attachment"); err != nil {
		return nil, err
	}
	if err := nmCheckTags(nil, nmTags(in)); err != nil {
		return nil, err
	}
	nat, rules, err := nmNatInput(in, nil)
	if err != nil {
		return nil, err
	}
	id := "attachment-" + nmHex(17)
	data := map[string]any{"CoreNetworkId": coreID, "CoreNetworkArn": core.arn, "AttachmentId": id, "OwnerAccountId": nmAccount,
		"AttachmentType": attType, "ResourceArn": resourceArn}
	if edge != "" {
		data["EdgeLocation"] = edge
	}
	if len(edges) > 0 {
		data["EdgeLocations"] = nmStringsAny(edges)
	}
	rec := st.newRec("attachment", id, nmArn("attachment/"+id), coreID, data, "CREATING", "AVAILABLE", nmTags(in))
	segment, rule, accept := doc.mapAttachment(nmAttachmentTypeNames[attType], edge, resourceID, rec.tags)
	if segment != "" {
		data["SegmentName"], data["AttachmentPolicyRuleNumber"] = segment, float64(rule)
	}
	if accept {
		rec.state, rec.settle = "PENDING_ATTACHMENT_ACCEPTANCE", ""
	}
	if nat != "" {
		data["NatGatewaySupport"] = nat
		rec.ext["NatGatewayRules"] = rules
	}
	if v, ok := in["NatSystemRouteSupport"].(bool); ok {
		data["NatSystemRouteSupport"] = v
	}
	if label := nmStr(in, "RoutingPolicyLabel"); label != "" {
		rec.ext["RoutingPolicyLabel"] = label
	}
	st.rememberToken(op, nmStr(in, "ClientToken"), id)
	return rec, nil
}

// replayAttachment answers a repeated request that carries a known client token.
func (st *nmStore) replayAttachment(op string, in map[string]any) (map[string]any, bool) {
	if id, ok := st.token(op, nmStr(in, "ClientToken")); ok {
		return st.attachmentOut(st.recs[id]), true
	}
	return nil, false
}

// nmNatInput turns NAT request members into stored form; prev holds the
// rules already on an attachment when updating.
func nmNatInput(in map[string]any, prev []any) (string, []any, *nmErr) {
	support := ""
	switch nmStr(in, "NatGatewaySupport") {
	case "ENABLE":
		support = "ENABLED"
	case "DISABLE":
		support = "DISABLED"
	}
	reqRules, given := in["NatGatewayRules"].([]any)
	if !given {
		return support, prev, nil
	}
	if len(reqRules) > 100 {
		return "", nil, nmValidation("An attachment can have at most 100 NAT rules.")
	}
	seen := map[float64]bool{}
	out := make([]any, 0, len(reqRules))
	for _, r := range reqRules {
		rm, _ := r.(map[string]any)
		n, _ := rm["RuleNumber"].(float64)
		if seen[n] {
			return "", nil, nmValidation("NAT rule number %v is used twice.", n)
		}
		seen[n] = true
		for _, k := range []string{"MatchSourceCidr", "NatCidr"} {
			if _, _, err := net.ParseCIDR(nmStr(rm, k)); err != nil {
				return "", nil, nmValidation("NAT rule %v %s isn't a CIDR block.", n, k)
			}
		}
		out = append(out, map[string]any{"RuleNumber": n, "MatchSourceCidr": rm["MatchSourceCidr"], "NatCidr": rm["NatCidr"], "State": "ACTIVATED"})
	}
	return support, out, nil
}

func (s *Server) nmCheckVpcAttachment(in map[string]any) *nmErr {
	_, region, _, _, vpcID, _ := nmArnParts(nmStr(in, "VpcArn"))
	vpcs := s.ec2.DescribeVpcs([]string{vpcID})
	if len(vpcs) == 0 {
		return nmNotFound("vpc", vpcID)
	}
	subnets, err := s.nmSubnets(nmStrings(in, "SubnetArns"))
	if err != nil {
		return err
	}
	zones := map[string]bool{}
	for _, sub := range subnets {
		if sub.vpc != vpcID {
			return nmValidation("Subnet %s isn't in VPC %s.", sub.id, vpcID)
		}
		if zones[sub.az] {
			return nmValidation("Only one subnet per Availability Zone can be attached; %s repeats %s.", sub.id, sub.az)
		}
		zones[sub.az] = true
	}
	azs := map[string]any{}
	for _, sub := range subnets {
		azs[sub.arn] = sub.az
	}
	in["_vpcId"], in["_vpcCidr"], in["_region"], in["_subnetAZs"] = vpcID, vpcs[0].CidrBlock, region, azs
	return nil
}

type nmSubnet struct{ id, vpc, az, arn string }

func (s *Server) nmSubnets(arns []string) ([]nmSubnet, *nmErr) {
	ids := make([]string, 0, len(arns))
	for _, arn := range arns {
		_, _, _, _, id, _ := nmArnParts(arn)
		ids = append(ids, id)
	}
	found := map[string]nmSubnet{}
	if len(ids) > 0 {
		for _, sub := range s.ec2.DescribeSubnets(ids, nil) {
			found[sub.ID] = nmSubnet{id: sub.ID, vpc: sub.VpcID, az: sub.AvailabilityZone}
		}
	}
	out := make([]nmSubnet, 0, len(ids))
	for i, id := range ids {
		sub, ok := found[id]
		if !ok {
			return nil, nmNotFound("subnet", id)
		}
		sub.arn = arns[i]
		out = append(out, sub)
	}
	return out, nil
}

func (s *Server) nmCheckAddedSubnets(in map[string]any) *nmErr {
	subnets, err := s.nmSubnets(nmStrings(in, "AddSubnetArns"))
	if err != nil {
		return err
	}
	added := map[string]any{}
	for _, sub := range subnets {
		added[sub.arn] = map[string]any{"vpc": sub.vpc, "az": sub.az}
	}
	in["_addSubnets"] = added
	return nil
}

func (s *Server) nmCheckVpnConnection(in map[string]any) *nmErr {
	_, _, _, _, id, _ := nmArnParts(nmStr(in, "VpnConnectionArn"))
	if len(s.ec2.DescribeVpnConnections([]string{id}, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)) == 0 {
		return nmNotFound("vpn-connection", id)
	}
	return nil
}

func nmCreateVpcAttachment(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	if out, ok := st.replayAttachment("CreateVpcAttachment", in); ok {
		return out, nil
	}
	rec, err := st.newAttachment("CreateVpcAttachment", in, "VPC", nmStr(in, "_region"), nil, nmStr(in, "VpcArn"), nmStr(in, "_vpcId"))
	if err != nil {
		return nil, err
	}
	rec.ext["SubnetArns"] = nmList(in, "SubnetArns")
	rec.ext["Options"] = nmMap(in, "Options")
	rec.ext["_vpcId"], rec.ext["_vpcCidr"] = nmStr(in, "_vpcId"), nmStr(in, "_vpcCidr")
	rec.ext["_azs"] = nmMap(in, "_subnetAZs") // subnet ARN → zone, for one-per-zone checks on update
	return st.attachmentOut(rec), nil
}

func nmUpdateVpcAttachment(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	rec, err := st.attachment(nmStr(in, "AttachmentId"))
	if err != nil {
		return nil, err
	}
	if rec.data["AttachmentType"] != "VPC" {
		return nil, nmNotFound("vpc-attachment", rec.id)
	}
	if rec.state != "AVAILABLE" {
		return nil, nmConflict("attachment", rec.id, "Attachment %s is %s; it can only be updated when AVAILABLE.", rec.id, rec.state)
	}
	current := nmAnyStrings(rec.ext["SubnetArns"])
	for _, arn := range nmStrings(in, "RemoveSubnetArns") {
		if !slices.Contains(current, arn) {
			return nil, nmValidation("Subnet %s isn't attached.", arn)
		}
		current = slices.DeleteFunc(current, func(x string) bool { return x == arn })
	}
	added, _ := in["_addSubnets"].(map[string]any)
	zones := map[string]any{}
	for arn, az := range nmMap(rec.ext, "_azs") {
		if slices.Contains(current, arn) {
			zones[arn] = az
		}
	}
	for _, arn := range nmStrings(in, "AddSubnetArns") {
		info := added[arn].(map[string]any)
		if info["vpc"] != rec.ext["_vpcId"] {
			return nil, nmValidation("Subnet %s isn't in the attached VPC.", arn)
		}
		if slices.Contains(current, arn) {
			return nil, nmValidation("Subnet %s is already attached.", arn)
		}
		for _, az := range zones {
			if az == info["az"] {
				return nil, nmValidation("Only one subnet per Availability Zone can be attached; %s repeats %v.", arn, az)
			}
		}
		zones[arn] = info["az"]
		current = append(current, arn)
	}
	if len(current) == 0 {
		return nil, nmValidation("A VPC attachment needs at least one subnet.")
	}
	rec.ext["SubnetArns"], rec.ext["_azs"] = nmStringsAny(current), zones
	if opts := nmMap(in, "Options"); opts != nil {
		rec.ext["Options"] = opts
	}
	rec.state, rec.settle, rec.updated = "UPDATING", "AVAILABLE", nmNow()
	return st.attachmentOut(rec), nil
}

func nmCreateConnectAttachment(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	if out, ok := st.replayAttachment("CreateConnectAttachment", in); ok {
		return out, nil
	}
	transport, err := st.attachment(nmStr(in, "TransportAttachmentId"))
	if err != nil {
		return nil, err
	}
	if transport.parent != nmStr(in, "CoreNetworkId") {
		return nil, nmValidation("Transport attachment %s isn't in core network %s.", transport.id, nmStr(in, "CoreNetworkId"))
	}
	if t := nmStr(transport.data, "AttachmentType"); t != "VPC" && t != "DIRECT_CONNECT_GATEWAY" {
		return nil, nmValidation("Transport attachment %s must be a VPC or Direct Connect gateway attachment.", transport.id)
	}
	if transport.state != "AVAILABLE" {
		return nil, nmConflict("attachment", transport.id, "Transport attachment %s is %s, not AVAILABLE.", transport.id, transport.state)
	}
	edge := nmStr(in, "EdgeLocation")
	if e, ok := transport.data["EdgeLocation"].(string); ok && e != edge {
		return nil, nmValidation("EdgeLocation %s must match the transport attachment's edge location %s.", edge, e)
	}
	rec, err := st.newAttachment("CreateConnectAttachment", in, "CONNECT", edge, nil, transport.arn, transport.id)
	if err != nil {
		return nil, err
	}
	rec.ext["TransportAttachmentId"], rec.ext["Options"] = transport.id, nmMap(in, "Options")
	return st.attachmentOut(rec), nil
}

func nmCreateVpnAttachment(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	if out, ok := st.replayAttachment("CreateSiteToSiteVpnAttachment", in); ok {
		return out, nil
	}
	arn := nmStr(in, "VpnConnectionArn")
	_, region, _, _, id, _ := nmArnParts(arn)
	rec, err := st.newAttachment("CreateSiteToSiteVpnAttachment", in, "SITE_TO_SITE_VPN", region, nil, arn, id)
	if err != nil {
		return nil, err
	}
	rec.ext["VpnConnectionArn"] = arn
	return st.attachmentOut(rec), nil
}

func nmCreateTGWRouteTableAttachment(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	if out, ok := st.replayAttachment("CreateTransitGatewayRouteTableAttachment", in); ok {
		return out, nil
	}
	peering, err := st.get("peering", "peering", nmStr(in, "PeeringId"))
	if err != nil {
		return nil, err
	}
	if peering.state != "AVAILABLE" {
		return nil, nmConflict("peering", peering.id, "Peering %s is %s, not AVAILABLE.", peering.id, peering.state)
	}
	arn := nmStr(in, "TransitGatewayRouteTableArn")
	svc, _, _, typ, rtb, ok := nmArnParts(arn)
	if !ok || svc != "ec2" || typ != "transit-gateway-route-table" || !strings.HasPrefix(rtb, "tgw-rtb-") {
		return nil, nmValidation("TransitGatewayRouteTableArn %s isn't a transit gateway route table ARN.", arn)
	}
	in["CoreNetworkId"] = peering.parent
	rec, err := st.newAttachment("CreateTransitGatewayRouteTableAttachment", in, "TRANSIT_GATEWAY_ROUTE_TABLE",
		nmStr(peering.data, "EdgeLocation"), nil, arn, rtb)
	if err != nil {
		return nil, err
	}
	rec.ext["PeeringId"], rec.ext["TransitGatewayRouteTableArn"] = peering.id, arn
	return st.attachmentOut(rec), nil
}

func nmCreateDXAttachment(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	if out, ok := st.replayAttachment("CreateDirectConnectGatewayAttachment", in); ok {
		return out, nil
	}
	arn := nmStr(in, "DirectConnectGatewayArn")
	_, _, _, _, id, _ := nmArnParts(arn)
	edges := nmStrings(in, "EdgeLocations")
	if len(edges) == 0 {
		return nil, nmValidation("EdgeLocations needs at least one edge location.")
	}
	rec, err := st.newAttachment("CreateDirectConnectGatewayAttachment", in, "DIRECT_CONNECT_GATEWAY", "", edges, arn, id)
	if err != nil {
		return nil, err
	}
	rec.ext["DirectConnectGatewayArn"] = arn
	return st.attachmentOut(rec), nil
}

func nmUpdateDXAttachment(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	rec, err := st.attachment(nmStr(in, "AttachmentId"))
	if err != nil {
		return nil, err
	}
	if rec.data["AttachmentType"] != "DIRECT_CONNECT_GATEWAY" {
		return nil, nmNotFound("direct-connect-gateway-attachment", rec.id)
	}
	if rec.state != "AVAILABLE" {
		return nil, nmConflict("attachment", rec.id, "Attachment %s is %s; it can only be updated when AVAILABLE.", rec.id, rec.state)
	}
	if edges := nmStrings(in, "EdgeLocations"); len(edges) > 0 {
		doc := st.liveDoc(rec.parent)
		for _, e := range edges {
			if doc == nil || doc.edge(e) == nil {
				return nil, nmValidation("Edge location %s isn't in the LIVE policy of core network %s.", e, rec.parent)
			}
		}
		rec.data["EdgeLocations"] = nmStringsAny(edges)
	}
	rec.state, rec.settle, rec.updated = "UPDATING", "AVAILABLE", nmNow()
	return st.attachmentOut(rec), nil
}

func nmGetAttachmentOf(attType, resourceType string) nmHandler {
	return func(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
		rec, err := st.attachment(nmStr(in, "AttachmentId"))
		if err != nil || rec.data["AttachmentType"] != attType {
			return nil, nmNotFound(resourceType, nmStr(in, "AttachmentId"))
		}
		return st.attachmentOut(rec), nil
	}
}

func nmListAttachments(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	core, typ, edge, state := nmStr(in, "CoreNetworkId"), nmStr(in, "AttachmentType"), nmStr(in, "EdgeLocation"), nmStr(in, "State")
	var items []any
	for _, rec := range st.list("attachment", func(r *nmRec) bool {
		return (core == "" || r.parent == core) && (typ == "" || r.data["AttachmentType"] == typ) &&
			(edge == "" || r.data["EdgeLocation"] == edge || slices.Contains(nmAnyStrings(r.data["EdgeLocations"]), edge))
	}) {
		if state == "" || rec.state == state {
			items = append(items, st.attachmentBase(rec))
		}
	}
	page, next, err := nmPage(in, items)
	if err != nil {
		return nil, err
	}
	return map[string]any{"Attachments": page, "NextToken": nmNonEmpty(next)}, nil
}

func nmAcceptAttachment(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	rec, err := st.attachment(nmStr(in, "AttachmentId"))
	if err != nil {
		return nil, err
	}
	if rec.state != "PENDING_ATTACHMENT_ACCEPTANCE" && rec.state != "PENDING_TAG_ACCEPTANCE" {
		return nil, nmConflict("attachment", rec.id, "Attachment %s is %s, not pending acceptance.", rec.id, rec.state)
	}
	rec.state, rec.settle, rec.updated = "CREATING", "AVAILABLE", nmNow()
	return map[string]any{"Attachment": st.attachmentBase(rec)}, nil
}

func nmRejectAttachment(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	rec, err := st.attachment(nmStr(in, "AttachmentId"))
	if err != nil {
		return nil, err
	}
	if rec.state != "PENDING_ATTACHMENT_ACCEPTANCE" && rec.state != "PENDING_TAG_ACCEPTANCE" {
		return nil, nmConflict("attachment", rec.id, "Attachment %s is %s, not pending acceptance.", rec.id, rec.state)
	}
	rec.state, rec.settle, rec.updated = "REJECTED", "", nmNow()
	return map[string]any{"Attachment": st.attachmentBase(rec)}, nil
}

func nmDeleteAttachment(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	rec, err := st.attachment(nmStr(in, "AttachmentId"))
	if err != nil {
		return nil, err
	}
	for _, other := range st.recs {
		if other.kind == "attachment" && other.ext["TransportAttachmentId"] == rec.id {
			return nil, nmConflict("attachment", rec.id, "Attachment %s is the transport for Connect attachment %s.", rec.id, other.id)
		}
		if other.kind == "connect-peer" && other.data["ConnectAttachmentId"] == rec.id {
			return nil, nmConflict("attachment", rec.id, "Attachment %s still has Connect peer %s.", rec.id, other.id)
		}
	}
	rec.state, rec.settle = "DELETING", ""
	out := st.attachmentBase(rec)
	st.deleteRec(rec)
	return map[string]any{"Attachment": out}, nil
}

// ---- connect peers ----

func nmCreateConnectPeer(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	if id, ok := st.token("CreateConnectPeer", nmStr(in, "ClientToken")); ok {
		return map[string]any{"ConnectPeer": st.render(st.recs[id])}, nil
	}
	att, err := st.attachment(nmStr(in, "ConnectAttachmentId"))
	if err != nil {
		return nil, err
	}
	if att.data["AttachmentType"] != "CONNECT" {
		return nil, nmValidation("Attachment %s isn't a Connect attachment.", att.id)
	}
	if att.state != "AVAILABLE" {
		return nil, nmConflict("attachment", att.id, "Connect attachment %s is %s, not AVAILABLE.", att.id, att.state)
	}
	peerAddr := nmStr(in, "PeerAddress")
	if net.ParseIP(peerAddr) == nil {
		return nil, nmValidation("PeerAddress %s isn't an IP address.", peerAddr)
	}
	coreAddr := nmStr(in, "CoreNetworkAddress")
	if coreAddr != "" && net.ParseIP(coreAddr) == nil {
		return nil, nmValidation("CoreNetworkAddress %s isn't an IP address.", coreAddr)
	}
	cidrs := nmStrings(in, "InsideCidrBlocks")
	if len(cidrs) > 2 {
		return nil, nmValidation("InsideCidrBlocks takes at most 2 blocks.")
	}
	for _, c := range cidrs {
		if _, _, err := net.ParseCIDR(c); err != nil {
			return nil, nmValidation("InsideCidrBlocks entry %s isn't a CIDR block.", c)
		}
	}
	protocol, _ := nmMap(att.ext, "Options")["Protocol"].(string)
	if protocol == "" {
		protocol = "GRE"
	}
	if protocol == "GRE" && len(cidrs) == 0 {
		return nil, nmValidation("A GRE Connect peer needs InsideCidrBlocks.")
	}
	if protocol == "NO_ENCAP" && nmStr(in, "SubnetArn") == "" {
		return nil, nmValidation("A NO_ENCAP Connect peer needs a SubnetArn.")
	}
	if coreAddr == "" && len(cidrs) > 0 {
		if ip, ipnet, err := net.ParseCIDR(cidrs[0]); err == nil && ip.To4() != nil {
			base := ipnet.IP.To4()
			coreAddr = net.IPv4(base[0], base[1], base[2], base[3]+1).String()
		}
	}
	if err := nmCheckCap(st, "connect-peer"); err != nil {
		return nil, err
	}
	if err := nmCheckTags(nil, nmTags(in)); err != nil {
		return nil, err
	}
	edge := nmStr(att.data, "EdgeLocation")
	var coreAsn float64
	if doc := st.liveDoc(att.parent); doc != nil {
		if e := doc.edge(edge); e != nil {
			coreAsn = float64(e.asn)
		}
	}
	bgp := map[string]any{"CoreNetworkAsn": coreAsn, "CoreNetworkAddress": coreAddr, "PeerAddress": peerAddr}
	if asn, ok := nmMap(in, "BgpOptions")["PeerAsn"].(float64); ok {
		bgp["PeerAsn"] = asn
	}
	id := "connect-peer-" + nmHex(17)
	data := map[string]any{"CoreNetworkId": att.parent, "ConnectAttachmentId": att.id, "ConnectPeerId": id, "EdgeLocation": edge,
		"Configuration": map[string]any{"CoreNetworkAddress": coreAddr, "PeerAddress": peerAddr, "InsideCidrBlocks": nmStringsAny(cidrs),
			"Protocol": protocol, "BgpConfigurations": []any{bgp}}}
	if sub := nmStr(in, "SubnetArn"); sub != "" {
		data["SubnetArn"] = sub
	}
	rec := st.newRec("connect-peer", id, nmArn("connect-peer/"+id), att.parent, data, "CREATING", "AVAILABLE", nmTags(in))
	st.rememberToken("CreateConnectPeer", nmStr(in, "ClientToken"), id)
	return map[string]any{"ConnectPeer": st.render(rec)}, nil
}

func nmGetConnectPeer(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	rec, err := st.get("connect-peer", "connect-peer", nmStr(in, "ConnectPeerId"))
	if err != nil {
		return nil, err
	}
	return map[string]any{"ConnectPeer": st.render(rec)}, nil
}

func nmListConnectPeers(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	core, att := nmStr(in, "CoreNetworkId"), nmStr(in, "ConnectAttachmentId")
	var items []any
	for _, rec := range st.list("connect-peer", func(r *nmRec) bool {
		return (core == "" || r.parent == core) && (att == "" || r.data["ConnectAttachmentId"] == att)
	}) {
		out := st.render(rec)
		out["ConnectPeerState"] = rec.state
		items = append(items, out)
	}
	page, next, err := nmPage(in, items)
	if err != nil {
		return nil, err
	}
	return map[string]any{"ConnectPeers": page, "NextToken": nmNonEmpty(next)}, nil
}

func nmDeleteConnectPeer(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	rec, err := st.get("connect-peer", "connect-peer", nmStr(in, "ConnectPeerId"))
	if err != nil {
		return nil, err
	}
	for _, a := range st.assocs {
		if a.kind == "cp-assoc" && a.data["ConnectPeerId"] == rec.id {
			return nil, nmConflict("connect-peer", rec.id, "Connect peer %s is still associated with a device.", rec.id)
		}
	}
	rec.state, rec.settle = "DELETING", ""
	out := st.render(rec)
	st.deleteRec(rec)
	return map[string]any{"ConnectPeer": out}, nil
}

// ---- peerings ----

func (st *nmStore) peeringOut(rec *nmRec) map[string]any {
	out := map[string]any{"Peering": st.render(rec)}
	for k, v := range rec.ext {
		out[k] = v
	}
	return map[string]any{"TransitGatewayPeering": out}
}

func nmCreateTGWPeering(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	if id, ok := st.token("CreateTransitGatewayPeering", nmStr(in, "ClientToken")); ok {
		return st.peeringOut(st.recs[id]), nil
	}
	coreID, arn := nmStr(in, "CoreNetworkId"), nmStr(in, "TransitGatewayArn")
	core, err := st.coreNetwork(coreID)
	if err != nil {
		return nil, err
	}
	_, region, _, _, _, _ := nmArnParts(arn)
	if doc := st.liveDoc(coreID); doc == nil || doc.edge(region) == nil {
		return nil, nmValidation("Core network %s has no LIVE edge location in %s.", coreID, region)
	}
	for _, rec := range st.recs {
		if rec.kind == "peering" && rec.parent == coreID && rec.ext["TransitGatewayArn"] == arn {
			return nil, nmConflict("peering", rec.id, "Transit gateway %s is already peered with core network %s.", arn, coreID)
		}
	}
	if err := nmCheckCap(st, "peering"); err != nil {
		return nil, err
	}
	if err := nmCheckTags(nil, nmTags(in)); err != nil {
		return nil, err
	}
	id := "peering-" + nmHex(17)
	data := map[string]any{"CoreNetworkId": coreID, "CoreNetworkArn": core.arn, "PeeringId": id, "OwnerAccountId": nmAccount,
		"PeeringType": "TRANSIT_GATEWAY", "EdgeLocation": region, "ResourceArn": arn}
	rec := st.newRec("peering", id, nmArn("peering/"+id), coreID, data, "CREATING", "AVAILABLE", nmTags(in))
	rec.ext["TransitGatewayArn"], rec.ext["TransitGatewayPeeringAttachmentId"] = arn, "tgw-attach-"+nmHex(17)
	st.rememberToken("CreateTransitGatewayPeering", nmStr(in, "ClientToken"), id)
	return st.peeringOut(rec), nil
}

func nmGetTGWPeering(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	rec, err := st.get("peering", "peering", nmStr(in, "PeeringId"))
	if err != nil {
		return nil, err
	}
	return st.peeringOut(rec), nil
}

func nmListPeerings(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	core, typ, edge, state := nmStr(in, "CoreNetworkId"), nmStr(in, "PeeringType"), nmStr(in, "EdgeLocation"), nmStr(in, "State")
	var items []any
	for _, rec := range st.list("peering", func(r *nmRec) bool {
		return (core == "" || r.parent == core) && (typ == "" || r.data["PeeringType"] == typ) && (edge == "" || r.data["EdgeLocation"] == edge)
	}) {
		if state == "" || rec.state == state {
			items = append(items, st.render(rec))
		}
	}
	page, next, err := nmPage(in, items)
	if err != nil {
		return nil, err
	}
	return map[string]any{"Peerings": page, "NextToken": nmNonEmpty(next)}, nil
}

func nmDeletePeering(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	rec, err := st.get("peering", "peering", nmStr(in, "PeeringId"))
	if err != nil {
		return nil, err
	}
	for _, other := range st.recs {
		if other.kind == "attachment" && other.ext["PeeringId"] == rec.id {
			return nil, nmConflict("peering", rec.id, "Peering %s still has route table attachment %s.", rec.id, other.id)
		}
	}
	rec.state, rec.settle = "DELETING", ""
	out := st.render(rec)
	st.deleteRec(rec)
	return map[string]any{"Peering": out}, nil
}

// ---- prefix lists, routing policy labels, NAT ----

func nmCreatePrefixListAssociation(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	coreID, arn := nmStr(in, "CoreNetworkId"), nmStr(in, "PrefixListArn")
	if _, err := st.coreNetwork(coreID); err != nil {
		return nil, err
	}
	if svc, _, _, typ, id, ok := nmArnParts(arn); !ok || svc != "ec2" || typ != "prefix-list" || !strings.HasPrefix(id, "pl-") {
		return nil, nmValidation("PrefixListArn %s isn't a prefix list ARN.", arn)
	}
	lists := st.prefixLists[coreID]
	if lists == nil {
		lists = map[string]string{}
		st.prefixLists[coreID] = lists
	}
	if _, ok := lists[arn]; ok {
		return nil, nmConflict("prefix-list-association", arn, "Prefix list %s is already associated with core network %s.", arn, coreID)
	}
	if len(lists) >= 100 {
		return nil, nmQuota("prefix-list-association", "A core network can have at most 100 prefix list associations.")
	}
	lists[arn] = nmStr(in, "PrefixListAlias")
	return map[string]any{"CoreNetworkId": coreID, "PrefixListArn": arn, "PrefixListAlias": lists[arn]}, nil
}

func nmDeletePrefixListAssociation(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	coreID, arn := nmStr(in, "CoreNetworkId"), nmStr(in, "PrefixListArn")
	if _, ok := st.prefixLists[coreID][arn]; !ok {
		return nil, nmNotFound("prefix-list-association", arn)
	}
	delete(st.prefixLists[coreID], arn)
	return map[string]any{"CoreNetworkId": coreID, "PrefixListArn": arn}, nil
}

func nmListPrefixListAssociations(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	coreID, filter := nmStr(in, "CoreNetworkId"), nmStr(in, "PrefixListArn")
	if _, err := st.coreNetwork(coreID); err != nil {
		return nil, err
	}
	var items []any
	for _, arn := range nmSortedKeys(st.prefixLists[coreID]) {
		if filter == "" || arn == filter {
			items = append(items, map[string]any{"CoreNetworkId": coreID, "PrefixListArn": arn, "PrefixListAlias": st.prefixLists[coreID][arn]})
		}
	}
	page, next, err := nmPage(in, items)
	if err != nil {
		return nil, err
	}
	return map[string]any{"PrefixListAssociations": page, "NextToken": nmNonEmpty(next)}, nil
}

func nmSortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

func (st *nmStore) coreAttachment(coreID, id string) (*nmRec, *nmErr) {
	if _, err := st.coreNetwork(coreID); err != nil {
		return nil, err
	}
	rec, err := st.attachment(id)
	if err != nil || rec.parent != coreID {
		return nil, nmNotFound("attachment", id)
	}
	return rec, nil
}

func nmPutRoutingPolicyLabel(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	rec, err := st.coreAttachment(nmStr(in, "CoreNetworkId"), nmStr(in, "AttachmentId"))
	if err != nil {
		return nil, err
	}
	rec.ext["RoutingPolicyLabel"] = nmStr(in, "RoutingPolicyLabel")
	return map[string]any{"CoreNetworkId": rec.parent, "AttachmentId": rec.id, "RoutingPolicyLabel": rec.ext["RoutingPolicyLabel"]}, nil
}

func nmRemoveRoutingPolicyLabel(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	rec, err := st.coreAttachment(nmStr(in, "CoreNetworkId"), nmStr(in, "AttachmentId"))
	if err != nil {
		return nil, err
	}
	label, ok := rec.ext["RoutingPolicyLabel"].(string)
	if !ok {
		return nil, nmNotFound("routing-policy-label", rec.id)
	}
	delete(rec.ext, "RoutingPolicyLabel")
	return map[string]any{"CoreNetworkId": rec.parent, "AttachmentId": rec.id, "RoutingPolicyLabel": label}, nil
}

func nmListRoutingPolicyAssociations(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	coreID, att := nmStr(in, "CoreNetworkId"), nmStr(in, "AttachmentId")
	if _, err := st.coreNetwork(coreID); err != nil {
		return nil, err
	}
	var items []any
	for _, rec := range st.list("attachment", func(r *nmRec) bool { return r.parent == coreID && (att == "" || r.id == att) }) {
		if label, ok := rec.ext["RoutingPolicyLabel"].(string); ok {
			items = append(items, map[string]any{"AttachmentId": rec.id, "RoutingPolicyLabel": label,
				"AssociatedRoutingPolicies": []any{}, "PendingRoutingPolicies": []any{}})
		}
	}
	page, next, err := nmPage(in, items)
	if err != nil {
		return nil, err
	}
	return map[string]any{"AttachmentRoutingPolicyAssociations": page, "NextToken": nmNonEmpty(next)}, nil
}

func nmNatOut(rec *nmRec) map[string]any {
	out := map[string]any{"AttachmentId": rec.id, "NatGatewaySupport": rec.data["NatGatewaySupport"],
		"NatGatewayRules": rec.ext["NatGatewayRules"]}
	if v, ok := rec.data["NatSystemRouteSupport"]; ok {
		out["NatSystemRouteSupport"] = v
	}
	return out
}

func nmListNatConfigurations(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	core, att := nmStr(in, "CoreNetworkId"), nmStr(in, "AttachmentId")
	if core != "" {
		if _, err := st.coreNetwork(core); err != nil {
			return nil, err
		}
	}
	items := []any{}
	for _, rec := range st.list("attachment", func(r *nmRec) bool {
		return (core == "" || r.parent == core) && (att == "" || r.id == att) && r.data["NatGatewaySupport"] != nil
	}) {
		items = append(items, nmNatOut(rec))
	}
	page, next, err := nmPage(in, items)
	if err != nil {
		return nil, err
	}
	return map[string]any{"AttachmentNatConfigurations": page, "NextToken": nmNonEmpty(next)}, nil
}

func nmUpdateNatConfiguration(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	rec, err := st.attachment(nmStr(in, "AttachmentId"))
	if err != nil {
		return nil, err
	}
	if rec.data["AttachmentType"] == "TRANSIT_GATEWAY_ROUTE_TABLE" {
		return nil, nmValidation("Transit gateway route table attachments don't support NAT.")
	}
	prev, _ := rec.ext["NatGatewayRules"].([]any)
	support, rules, err := nmNatInput(in, prev)
	if err != nil {
		return nil, err
	}
	if support != "" {
		rec.data["NatGatewaySupport"] = support
	} else if rec.data["NatGatewaySupport"] == nil {
		rec.data["NatGatewaySupport"] = "DISABLED"
	}
	rec.ext["NatGatewayRules"] = rules
	if v, ok := in["NatSystemRouteSupport"].(bool); ok {
		rec.data["NatSystemRouteSupport"] = v
	}
	rec.updated = nmNow()
	return nmNatOut(rec), nil
}

// ---- routing information ----

// nmSegmentRoutes lists the VPC CIDRs reachable on a core network segment at
// an edge: one route per AVAILABLE VPC attachment mapped to that segment.
func (st *nmStore) nmSegmentRoutes(coreID, segment, edge string) []*nmRec {
	return st.list("attachment", func(r *nmRec) bool {
		return r.parent == coreID && r.data["AttachmentType"] == "VPC" && r.data["SegmentName"] == segment &&
			r.data["EdgeLocation"] == edge && r.state == "AVAILABLE"
	})
}

func nmListCoreNetworkRoutingInformation(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	coreID, segment, edge := nmStr(in, "CoreNetworkId"), nmStr(in, "SegmentName"), nmStr(in, "EdgeLocation")
	if _, err := st.coreNetwork(coreID); err != nil {
		return nil, err
	}
	exact := nmStrings(in, "ExactPrefixMatches")
	var asn string
	if doc := st.liveDoc(coreID); doc != nil {
		if e := doc.edge(edge); e != nil {
			asn = strconv.FormatInt(e.asn, 10)
		}
	}
	var items []any
	for _, rec := range st.nmSegmentRoutes(coreID, segment, edge) {
		cidr := nmStr(rec.ext, "_vpcCidr")
		if len(exact) > 0 && !slices.Contains(exact, cidr) {
			continue
		}
		items = append(items, map[string]any{"Prefix": cidr, "LocalPreference": "100", "Med": "0",
			"AsPath": []any{asn}, "Communities": []any{},
			"NextHop": map[string]any{"CoreNetworkAttachmentId": rec.id, "ResourceId": rec.ext["_vpcId"], "ResourceType": "vpc",
				"SegmentName": segment, "EdgeLocation": edge}})
	}
	page, next, err := nmPage(in, items)
	if err != nil {
		return nil, err
	}
	return map[string]any{"CoreNetworkRoutingInformation": page, "NextToken": nmNonEmpty(next)}, nil
}

// trimPolicyBytes drops the oldest non-LIVE policy versions (across all core
// networks) until adding incoming bytes stays within nmMaxPolicyBytes.
func (st *nmStore) trimPolicyBytes(incoming int) {
	total := incoming
	for _, vs := range st.policies {
		for _, p := range vs {
			total += len(p.document)
		}
	}
	for total > nmMaxPolicyBytes {
		oldestCore, oldestIdx := "", -1
		var oldest time.Time
		for core, vs := range st.policies {
			for i, p := range vs {
				if p.alias == "LIVE" || p.alias == "LATEST" {
					continue
				}
				if oldestIdx < 0 || p.created.Before(oldest) {
					oldestCore, oldestIdx, oldest = core, i, p.created
				}
				break // versions are in creation order; the first eligible is the oldest
			}
		}
		if oldestIdx < 0 {
			return // only LIVE/LATEST versions remain
		}
		vs := st.policies[oldestCore]
		total -= len(vs[oldestIdx].document)
		st.policies[oldestCore] = slices.Delete(vs, oldestIdx, oldestIdx+1)
	}
}
