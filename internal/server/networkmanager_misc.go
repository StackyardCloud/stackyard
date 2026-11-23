package server

import (
	"encoding/json"
	"slices"
	"strconv"
	"strings"
)

func init() {
	for name, h := range map[string]nmHandler{
		"CreateContextStore": nmCreateContextStore, "GetContextStore": nmGetContextStore, "ListContextStores": nmListContextStores,
		"UpdateContextStore": nmUpdateContextStore, "DeleteContextStore": nmDeleteContextStore,
		"AssociateContextStore": nmAssociateContextStore, "DisassociateContextStore": nmDisassociateContextStore,
		"GetContextStoreAssociation": nmGetContextStoreAssociation, "ListContextStoreAssociations": nmListContextStoreAssociations,
		"GetContextVersion": nmGetContextVersion, "ListContextChanges": nmListContextChanges, "ListContexts": nmListContexts,
		"GetNetworkResources": nmGetNetworkResources, "GetNetworkResourceCounts": nmGetNetworkResourceCounts,
		"GetNetworkResourceRelationships": nmGetNetworkResourceRelationships, "GetNetworkTelemetry": nmGetNetworkTelemetry,
		"UpdateNetworkResourceMetadata": nmUpdateNetworkResourceMetadata, "GetNetworkRoutes": nmGetNetworkRoutes,
		"PutResourcePolicy": nmPutResourcePolicy, "GetResourcePolicy": nmGetResourcePolicy, "DeleteResourcePolicy": nmDeleteResourcePolicy,
		"StartOrganizationServiceAccessUpdate": nmStartOrgAccessUpdate, "ListOrganizationServiceAccessStatus": nmListOrgAccessStatus,
		"TagResource": nmTagResource, "UntagResource": nmUntagResource, "ListTagsForResource": nmListTagsForResource,
	} {
		nmHandlers[name] = h
	}
	nmPreChecks["AssociateContextStore"] = (*Server).nmCheckContextStoreTGW
}

// ---- context stores ----

func (st *nmStore) contextStore(id string) (*nmRec, *nmErr) {
	return st.get("context-store", "context-store", id)
}

func nmCreateContextStore(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	if id, ok := st.token("CreateContextStore", nmStr(in, "ClientToken")); ok {
		return map[string]any{"ContextStore": st.render(st.recs[id])}, nil
	}
	gnID := nmStr(in, "GlobalNetworkId")
	if _, err := st.globalNetwork(gnID); err != nil {
		return nil, err
	}
	if err := nmCheckCap(st, "context-store"); err != nil {
		return nil, err
	}
	if err := nmCheckTags(nil, nmTags(in)); err != nil {
		return nil, err
	}
	id := "context-store-" + nmHex(17)
	data := nmCopyFields(in, "GlobalNetworkId", "Description")
	data["ContextStoreId"], data["ContextStoreArn"] = id, nmArn("context-store/"+id)
	rec := st.newRec("context-store", id, nmArn("context-store/"+id), gnID, data, "PENDING", "AVAILABLE", nmTags(in))
	st.rememberToken("CreateContextStore", nmStr(in, "ClientToken"), id)
	return map[string]any{"ContextStore": st.render(rec)}, nil
}

func nmGetContextStore(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	rec, err := st.contextStore(nmStr(in, "ContextStoreId"))
	if err != nil {
		return nil, err
	}
	return map[string]any{"ContextStore": st.render(rec)}, nil
}

func nmListContextStores(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	gnID := nmStr(in, "GlobalNetworkId")
	recs := st.list("context-store", func(r *nmRec) bool { return gnID == "" || r.parent == gnID })
	page, next, err := nmPage(in, st.renderAll(recs))
	if err != nil {
		return nil, err
	}
	return map[string]any{"ContextStores": page, "NextToken": nmNonEmpty(next)}, nil
}

func nmUpdateContextStore(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	rec, err := st.contextStore(nmStr(in, "ContextStoreId"))
	if err != nil {
		return nil, err
	}
	nmApply(rec, in, "Description")
	return map[string]any{"ContextStore": st.render(rec)}, nil
}

func nmDeleteContextStore(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	rec, err := st.contextStore(nmStr(in, "ContextStoreId"))
	if err != nil {
		return nil, err
	}
	for _, a := range st.assocs {
		if a.kind == "ctx-assoc" && a.parent == rec.id {
			return nil, nmConflict("context-store", rec.id, "Context store %s still has associations; disassociate them first.", rec.id)
		}
	}
	rec.state, rec.settle = "DELETING", ""
	out := st.render(rec)
	delete(st.ctxVersions, rec.id)
	delete(st.ctxChanges, rec.id)
	st.deleteRec(rec)
	return map[string]any{"ContextStore": out}, nil
}

func (s *Server) nmCheckContextStoreTGW(in map[string]any) *nmErr {
	arn := nmStr(in, "TransitGatewayArn")
	if arn == "" {
		return nmValidation("TransitGatewayArn is required.")
	}
	return s.nmCheckTransitGateway(arn)
}

// recordContextChange bumps the store's version with one change entry.
func (st *nmStore) recordContextChange(storeID, change, tgwArn string) {
	st.ctxVersions[storeID]++
	changes := append(st.ctxChanges[storeID], nmContextChange{version: st.ctxVersions[storeID], change: change, tgwArn: tgwArn})
	if len(changes) > 1000 {
		changes = changes[len(changes)-1000:]
	}
	st.ctxChanges[storeID] = changes
}

func nmAssociateContextStore(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	storeID, arn := nmStr(in, "ContextStoreId"), nmStr(in, "TransitGatewayArn")
	if _, err := st.contextStore(storeID); err != nil {
		return nil, err
	}
	key := storeID + "|" + arn
	if st.assocs["ctx-assoc|"+key] != nil {
		return nil, nmConflict("context-store-association", key, "Transit gateway %s is already associated with context store %s.", arn, storeID)
	}
	if err := nmCheckCap(st, "context-store-association"); err != nil {
		return nil, err
	}
	rec := st.newAssoc("ctx-assoc", key, storeID, map[string]any{"ContextStoreId": storeID, "TransitGatewayArn": arn}, "State", "ASSOCIATING", "ASSOCIATED")
	st.recordContextChange(storeID, "ADDED", arn)
	return map[string]any{"ContextStoreAssociation": st.render(rec)}, nil
}

func nmDisassociateContextStore(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	storeID, arn := nmStr(in, "ContextStoreId"), nmStr(in, "TransitGatewayArn")
	rec := st.assocs["ctx-assoc|"+storeID+"|"+arn]
	if rec == nil {
		return nil, nmNotFound("context-store-association", storeID+"|"+arn)
	}
	rec.state, rec.settle = "DISASSOCIATING", ""
	out := st.render(rec)
	delete(st.assocs, "ctx-assoc|"+storeID+"|"+arn)
	st.recordContextChange(storeID, "REMOVED", arn)
	return map[string]any{"ContextStoreAssociation": out}, nil
}

func nmGetContextStoreAssociation(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	storeID, arn := nmStr(in, "ContextStoreId"), nmStr(in, "TransitGatewayArn")
	rec := st.assocs["ctx-assoc|"+storeID+"|"+arn]
	if rec == nil {
		return nil, nmNotFound("context-store-association", storeID+"|"+arn)
	}
	st.read(rec)
	return map[string]any{"ContextStoreAssociation": st.render(rec)}, nil
}

func nmListContextStoreAssociations(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	storeID := nmStr(in, "ContextStoreId")
	if _, err := st.contextStore(storeID); err != nil {
		return nil, err
	}
	page, next, err := nmPage(in, st.renderAll(st.listAssocs("ctx-assoc", func(r *nmRec) bool { return r.parent == storeID })))
	if err != nil {
		return nil, err
	}
	return map[string]any{"ContextStoreAssociations": page, "NextToken": nmNonEmpty(next)}, nil
}

func nmGetContextVersion(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	storeID := nmStr(in, "ContextStoreId")
	if _, err := st.contextStore(storeID); err != nil {
		return nil, err
	}
	return map[string]any{"VersionId": strconv.Itoa(st.ctxVersions[storeID])}, nil
}

func nmContextEntry(tgwArn string) map[string]any {
	_, region, _, _, id, _ := nmArnParts(tgwArn)
	return map[string]any{"ContextId": tgwArn, "ContextType": "TransitGateway", "PublicId": id, "Region": region}
}

func nmListContextChanges(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	storeID := nmStr(in, "ContextStoreId")
	if _, err := st.contextStore(storeID); err != nil {
		return nil, err
	}
	from, err := strconv.ParseInt(nmStr(in, "FromVersionId"), 10, 32)
	current := st.ctxVersions[storeID]
	if err != nil || from < 0 || int(from) > current {
		return nil, nmValidation("FromVersionId must be a version from 0 to %d.", current)
	}
	var items []any
	for _, c := range st.ctxChanges[storeID] {
		if c.version > int(from) {
			items = append(items, map[string]any{"ChangeType": c.change, "Context": nmContextEntry(c.tgwArn)})
		}
	}
	page, next, perr := nmPage(in, items)
	if perr != nil {
		return nil, perr
	}
	return map[string]any{"ContextChanges": page, "FromVersionId": nmStr(in, "FromVersionId"),
		"ToVersionId": strconv.Itoa(current), "NextToken": nmNonEmpty(next)}, nil
}

func nmListContexts(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	storeID := nmStr(in, "ContextStoreId")
	if _, err := st.contextStore(storeID); err != nil {
		return nil, err
	}
	var items []any
	for _, a := range st.listAssocs("ctx-assoc", func(r *nmRec) bool { return r.parent == storeID }) {
		items = append(items, nmContextEntry(nmStr(a.data, "TransitGatewayArn")))
	}
	page, next, err := nmPage(in, items)
	if err != nil {
		return nil, err
	}
	return map[string]any{"Contexts": page, "NextToken": nmNonEmpty(next), "VersionId": strconv.Itoa(st.ctxVersions[storeID])}, nil
}

// ---- network resources ----

// nmResource is one AWS resource a global network knows about: registered
// transit gateways, its core networks and their attachments, peerings and
// Connect peers.
type nmResource struct {
	arn, id, typ, core, gateway, region string
	rec                                 *nmRec
}

func (st *nmStore) networkResources(gnID string) []nmResource {
	var out []nmResource
	for _, a := range st.listAssocs("registration", func(r *nmRec) bool { return r.parent == gnID }) {
		arn := nmStr(a.data, "TransitGatewayArn")
		_, region, _, _, id, _ := nmArnParts(arn)
		out = append(out, nmResource{arn: arn, id: id, typ: "transit-gateway", gateway: arn, region: region, rec: a})
	}
	for _, core := range st.list("core-network", func(r *nmRec) bool { return r.parent == gnID }) {
		out = append(out, nmResource{arn: core.arn, id: core.id, typ: "core-network", core: core.id, rec: core})
		for _, kind := range []string{"attachment", "peering", "connect-peer"} {
			for _, rec := range st.list(kind, func(r *nmRec) bool { return r.parent == core.id }) {
				region := nmStr(rec.data, "EdgeLocation")
				out = append(out, nmResource{arn: rec.arn, id: rec.id, typ: kind, core: core.id, region: region, rec: rec})
			}
		}
	}
	return out
}

func (st *nmStore) filteredResources(in map[string]any) ([]nmResource, *nmErr) {
	gnID := nmStr(in, "GlobalNetworkId")
	if _, err := st.globalNetwork(gnID); err != nil {
		return nil, err
	}
	var out []nmResource
	for _, r := range st.networkResources(gnID) {
		if (nmStr(in, "CoreNetworkId") == "" || r.core == nmStr(in, "CoreNetworkId")) &&
			(nmStr(in, "RegisteredGatewayArn") == "" || r.gateway == nmStr(in, "RegisteredGatewayArn")) &&
			(nmStr(in, "AwsRegion") == "" || r.region == nmStr(in, "AwsRegion")) &&
			(nmStr(in, "AccountId") == "" || nmStr(in, "AccountId") == nmAccount) &&
			(nmStr(in, "ResourceType") == "" || r.typ == nmStr(in, "ResourceType")) &&
			(nmStr(in, "ResourceArn") == "" || r.arn == nmStr(in, "ResourceArn")) {
			out = append(out, r)
		}
	}
	return out, nil
}

func nmGetNetworkResources(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	resources, err := st.filteredResources(in)
	if err != nil {
		return nil, err
	}
	var items []any
	for _, r := range resources {
		def, _ := json.Marshal(st.render(r.rec))
		item := map[string]any{"ResourceArn": r.arn, "ResourceId": r.id, "ResourceType": r.typ, "AccountId": nmAccount,
			"Definition": string(def), "DefinitionTimestamp": nmTime(r.rec.updated), "Tags": nmTagList(r.rec.tags)}
		if len(item["Definition"].(string)) > 256 { // ConstrainedString caps definitions at 256 characters
			item["Definition"] = strings.ToValidUTF8(item["Definition"].(string)[:256], "")
		}
		for k, v := range map[string]string{"CoreNetworkId": r.core, "RegisteredGatewayArn": r.gateway, "AwsRegion": r.region} {
			if v != "" {
				item[k] = v
			}
		}
		if md := st.metadata[r.arn]; md != nil {
			item["Metadata"] = md
		}
		items = append(items, item)
	}
	page, next, err := nmPage(in, items)
	if err != nil {
		return nil, err
	}
	return map[string]any{"NetworkResources": page, "NextToken": nmNonEmpty(next)}, nil
}

func nmGetNetworkResourceCounts(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	resources, err := st.filteredResources(in)
	if err != nil {
		return nil, err
	}
	counts := map[string]int{}
	for _, r := range resources {
		counts[r.typ]++
	}
	var items []any
	for _, typ := range []string{"transit-gateway", "core-network", "attachment", "peering", "connect-peer"} {
		if n := counts[typ]; n > 0 {
			items = append(items, map[string]any{"ResourceType": typ, "Count": float64(n)})
		}
	}
	page, next, err := nmPage(in, items)
	if err != nil {
		return nil, err
	}
	return map[string]any{"NetworkResourceCounts": page, "NextToken": nmNonEmpty(next)}, nil
}

func nmGetNetworkResourceRelationships(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	resources, err := st.filteredResources(in)
	if err != nil {
		return nil, err
	}
	var items []any
	for _, r := range resources {
		if r.typ == "core-network" || r.typ == "transit-gateway" {
			continue
		}
		parent := st.recs[r.core]
		from := ""
		if parent != nil {
			from = parent.arn
		}
		if r.typ == "connect-peer" {
			if att := st.recs[nmStr(r.rec.data, "ConnectAttachmentId")]; att != nil {
				from = att.arn
			}
		}
		if from != "" {
			items = append(items, map[string]any{"From": from, "To": r.arn})
		}
	}
	page, next, err := nmPage(in, items)
	if err != nil {
		return nil, err
	}
	return map[string]any{"Relationships": page, "NextToken": nmNonEmpty(next)}, nil
}

func nmGetNetworkTelemetry(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	resources, err := st.filteredResources(in)
	if err != nil {
		return nil, err
	}
	var items []any
	for _, r := range resources {
		if r.typ != "attachment" && r.typ != "peering" && r.typ != "connect-peer" {
			continue
		}
		status := "DOWN"
		if r.rec.state == "AVAILABLE" {
			status = "UP"
		}
		items = append(items, map[string]any{"ResourceArn": r.arn, "ResourceId": r.id, "ResourceType": r.typ, "CoreNetworkId": r.core,
			"AwsRegion": r.region, "AccountId": nmAccount,
			"Health": map[string]any{"Type": "BGP", "Status": status, "Timestamp": nmTime(nmNow())}})
	}
	page, next, err := nmPage(in, items)
	if err != nil {
		return nil, err
	}
	return map[string]any{"NetworkTelemetry": page, "NextToken": nmNonEmpty(next)}, nil
}

func nmUpdateNetworkResourceMetadata(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	gnID, arn := nmStr(in, "GlobalNetworkId"), nmStr(in, "ResourceArn")
	if _, err := st.globalNetwork(gnID); err != nil {
		return nil, err
	}
	found := slices.ContainsFunc(st.networkResources(gnID), func(r nmResource) bool { return r.arn == arn })
	if !found {
		return nil, nmNotFound("network-resource", arn)
	}
	md := nmMap(in, "Metadata")
	if len(md) > 50 {
		return nil, nmValidation("Metadata can hold at most 50 entries.")
	}
	st.metadata[arn] = md
	return map[string]any{"ResourceArn": arn, "Metadata": md}, nil
}

func nmGetNetworkRoutes(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	gnID := nmStr(in, "GlobalNetworkId")
	if _, err := st.globalNetwork(gnID); err != nil {
		return nil, err
	}
	id := nmMap(in, "RouteTableIdentifier")
	set := 0
	for _, k := range []string{"TransitGatewayRouteTableArn", "CoreNetworkSegmentEdge", "CoreNetworkNetworkFunctionGroup"} {
		if id[k] != nil {
			set++
		}
	}
	if set != 1 {
		return nil, nmValidation("RouteTableIdentifier needs exactly one of TransitGatewayRouteTableArn, CoreNetworkSegmentEdge or CoreNetworkNetworkFunctionGroup.")
	}
	out := map[string]any{"RouteTableTimestamp": nmTime(nmNow()), "NetworkRoutes": []any{}}
	switch {
	case id["TransitGatewayRouteTableArn"] != nil:
		// ponytail: transit gateway route tables live in EC2; no routes are derived here.
		out["RouteTableType"], out["RouteTableArn"] = "TRANSIT_GATEWAY_ROUTE_TABLE", id["TransitGatewayRouteTableArn"]
	case id["CoreNetworkNetworkFunctionGroup"] != nil:
		out["RouteTableType"], out["CoreNetworkNetworkFunctionGroup"] = "NETWORK_FUNCTION_GROUP", id["CoreNetworkNetworkFunctionGroup"]
	default:
		edge := nmMap(id, "CoreNetworkSegmentEdge")
		coreID, segment, location := nmStr(edge, "CoreNetworkId"), nmStr(edge, "SegmentName"), nmStr(edge, "EdgeLocation")
		core, err := st.coreNetwork(coreID)
		if err != nil {
			return nil, err
		}
		if core.parent != gnID {
			return nil, nmNotFound("core-network", coreID)
		}
		exact := nmStrings(in, "ExactCidrMatches")
		routes := []any{}
		for _, rec := range st.nmSegmentRoutes(coreID, segment, location) {
			cidr := nmStr(rec.ext, "_vpcCidr")
			if len(exact) > 0 && !slices.Contains(exact, cidr) {
				continue
			}
			routes = append(routes, map[string]any{"DestinationCidrBlock": cidr, "State": "ACTIVE", "Type": "PROPAGATED",
				"Destinations": []any{map[string]any{"CoreNetworkAttachmentId": rec.id, "SegmentName": segment, "EdgeLocation": location,
					"ResourceType": "vpc", "ResourceId": rec.ext["_vpcId"]}}})
		}
		out["RouteTableType"], out["CoreNetworkSegmentEdge"], out["NetworkRoutes"] = "CORE_NETWORK_SEGMENT", edge, routes
		out["RouteTableArn"] = core.arn + "/segment/" + segment + "/" + location
	}
	return out, nil
}

// ---- resource policies, Organizations access, tags ----

func nmPutResourcePolicy(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	arn := nmStr(in, "ResourceArn")
	if rec := st.byArn[arn]; rec == nil || rec.kind != "core-network" {
		return nil, nmValidation("Resource policies can only be attached to core networks; %s isn't one.", arn)
	}
	var doc map[string]any
	if err := json.Unmarshal([]byte(nmStr(in, "PolicyDocument")), &doc); err != nil || doc == nil {
		return nil, nmValidation("PolicyDocument isn't a JSON object.")
	}
	if _, ok := doc["Statement"]; !ok {
		return nil, nmValidation("PolicyDocument needs a Statement.")
	}
	st.resPolicies[arn] = nmStr(in, "PolicyDocument")
	return map[string]any{}, nil
}

func nmGetResourcePolicy(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	if doc, ok := st.resPolicies[nmStr(in, "ResourceArn")]; ok {
		return map[string]any{"PolicyDocument": doc}, nil
	}
	return map[string]any{}, nil
}

func nmDeleteResourcePolicy(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	delete(st.resPolicies, nmStr(in, "ResourceArn"))
	return map[string]any{}, nil
}

func (st *nmStore) orgStatus() map[string]any {
	// ponytail: a fixed organization; the Organizations store isn't consulted.
	return map[string]any{"OrganizationId": "o-0123456789abcdef0", "OrganizationAwsServiceAccessStatus": st.orgAccess,
		"SLRDeploymentStatus": "COMPLETED", "AccountStatusList": []any{map[string]any{"AccountId": nmAccount, "SLRDeploymentStatus": "COMPLETED"}}}
}

func nmStartOrgAccessUpdate(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	switch nmStr(in, "Action") {
	case "ENABLE":
		st.orgAccess = "ENABLED"
	case "DISABLE":
		st.orgAccess = "DISABLED"
	default:
		return nil, nmValidation("Action must be ENABLE or DISABLE.")
	}
	return map[string]any{"OrganizationStatus": st.orgStatus()}, nil
}

func nmListOrgAccessStatus(st *nmStore, _ map[string]any) (map[string]any, *nmErr) {
	return map[string]any{"OrganizationStatus": st.orgStatus()}, nil
}

func (st *nmStore) taggable(arn string) (*nmRec, *nmErr) {
	rec := st.byArn[arn]
	if rec == nil || !strings.HasPrefix(arn, "arn:aws:networkmanager:") {
		return nil, nmNotFound("resource", arn)
	}
	return rec, nil
}

func nmTagResource(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	rec, err := st.taggable(nmStr(in, "ResourceArn"))
	if err != nil {
		return nil, err
	}
	if err := nmCheckTags(rec.tags, nmTags(in)); err != nil {
		return nil, err
	}
	nmSetTags(rec, nmTags(in))
	return map[string]any{}, nil
}

func nmUntagResource(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	rec, err := st.taggable(nmStr(in, "ResourceArn"))
	if err != nil {
		return nil, err
	}
	for _, k := range nmStrings(in, "TagKeys") {
		delete(rec.tags, k)
	}
	return map[string]any{}, nil
}

func nmListTagsForResource(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	rec, err := st.taggable(nmStr(in, "ResourceArn"))
	if err != nil {
		return nil, err
	}
	return map[string]any{"TagList": nmTagList(rec.tags)}, nil
}
