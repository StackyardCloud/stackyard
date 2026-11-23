package server

import (
	"encoding/json"
	"net/http"
	"slices"
	"strings"
)

type nmHandler func(st *nmStore, in map[string]any) (map[string]any, *nmErr)

// nmHandlers maps every operation to its handler; handlers run under the
// store lock and the dispatcher conforms their output to the model.
var nmHandlers = map[string]nmHandler{}

// nmPreChecks run before the store lock for operations that consult other
// stores (EC2). They may record what they learned under "_"-prefixed keys:
// input members can't start with "_" (the model rejects unknown members and
// labels/query only fill modelled ones), so callers can't forge them.
var nmPreChecks = map[string]func(s *Server, in map[string]any) *nmErr{}

func (s *Server) nmDispatch(op string, in map[string]any) (map[string]any, *nmErr) {
	if pre := nmPreChecks[op]; pre != nil {
		if err := pre(s, in); err != nil {
			return nil, err
		}
	}
	h := nmHandlers[op]
	if h == nil {
		return nil, &nmErr{status: http.StatusNotImplemented, code: "NotImplementedException", msg: op + " is not implemented"}
	}
	st := s.networkManager
	st.mu.Lock()
	defer st.mu.Unlock()
	out, err := h(st, in)
	if err != nil {
		return nil, err
	}
	return nmConformOutput(op, out), nil
}

func init() {
	for name, h := range map[string]nmHandler{
		"CreateGlobalNetwork": nmCreateGlobalNetwork, "DescribeGlobalNetworks": nmDescribeGlobalNetworks,
		"UpdateGlobalNetwork": nmUpdateGlobalNetwork, "DeleteGlobalNetwork": nmDeleteGlobalNetwork,
		"CreateSite": nmCreateSite, "GetSites": nmGetSites, "UpdateSite": nmUpdateSite, "DeleteSite": nmDeleteSite,
		"CreateDevice": nmCreateDevice, "GetDevices": nmGetDevices, "UpdateDevice": nmUpdateDevice, "DeleteDevice": nmDeleteDevice,
		"CreateLink": nmCreateLink, "GetLinks": nmGetLinks, "UpdateLink": nmUpdateLink, "DeleteLink": nmDeleteLink,
		"CreateConnection": nmCreateConnection, "GetConnections": nmGetConnections,
		"UpdateConnection": nmUpdateConnection, "DeleteConnection": nmDeleteConnection,
		"AssociateLink": nmAssociateLink, "GetLinkAssociations": nmGetLinkAssociations, "DisassociateLink": nmDisassociateLink,
		"AssociateCustomerGateway": nmAssociateCustomerGateway, "GetCustomerGatewayAssociations": nmGetCustomerGatewayAssociations,
		"DisassociateCustomerGateway":        nmDisassociateCustomerGateway,
		"AssociateTransitGatewayConnectPeer": nmAssociateTGWConnectPeer, "GetTransitGatewayConnectPeerAssociations": nmGetTGWConnectPeerAssociations,
		"DisassociateTransitGatewayConnectPeer": nmDisassociateTGWConnectPeer,
		"AssociateConnectPeer":                  nmAssociateConnectPeer, "GetConnectPeerAssociations": nmGetConnectPeerAssociations,
		"DisassociateConnectPeer": nmDisassociateConnectPeer,
		"RegisterTransitGateway":  nmRegisterTransitGateway, "GetTransitGatewayRegistrations": nmGetTransitGatewayRegistrations,
		"DeregisterTransitGateway": nmDeregisterTransitGateway,
		"StartRouteAnalysis":       nmStartRouteAnalysis, "GetRouteAnalysis": nmGetRouteAnalysis,
	} {
		nmHandlers[name] = h
	}
	nmPreChecks["RegisterTransitGateway"] = (*Server).nmCheckTransitGatewayInput
	nmPreChecks["AssociateCustomerGateway"] = (*Server).nmCheckCustomerGateway
}

// ---- global networks ----

func (st *nmStore) globalNetwork(id string) (*nmRec, *nmErr) {
	return st.get("global-network", "global-network", id)
}

func nmCreateGlobalNetwork(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	if err := nmCheckCap(st, "global-network"); err != nil {
		return nil, err
	}
	if err := nmCheckTags(nil, nmTags(in)); err != nil {
		return nil, err
	}
	id := "global-network-" + nmHex(17)
	data := nmCopyFields(in, "Description")
	data["GlobalNetworkId"], data["GlobalNetworkArn"] = id, nmArn("global-network/"+id)
	rec := st.newRec("global-network", id, nmArn("global-network/"+id), "", data, "PENDING", "AVAILABLE", nmTags(in))
	return map[string]any{"GlobalNetwork": st.render(rec)}, nil
}

func nmDescribeGlobalNetworks(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	ids := nmStrings(in, "GlobalNetworkIds")
	recs := st.list("global-network", func(r *nmRec) bool { return len(ids) == 0 || slices.Contains(ids, r.id) })
	page, next, err := nmPage(in, st.renderAll(recs))
	if err != nil {
		return nil, err
	}
	return map[string]any{"GlobalNetworks": page, "NextToken": nmNonEmpty(next)}, nil
}

func nmUpdateGlobalNetwork(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	gn, err := st.globalNetwork(nmStr(in, "GlobalNetworkId"))
	if err != nil {
		return nil, err
	}
	if d, ok := nmOptStr(in, "Description"); ok {
		gn.data["Description"] = d
	}
	gn.state, gn.settle, gn.updated = "UPDATING", "AVAILABLE", nmNow()
	return map[string]any{"GlobalNetwork": st.render(gn)}, nil
}

func nmDeleteGlobalNetwork(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	gn, err := st.globalNetwork(nmStr(in, "GlobalNetworkId"))
	if err != nil {
		return nil, err
	}
	for _, rec := range st.recs {
		if rec.parent == gn.id && rec.kind != "route-analysis" {
			return nil, nmConflict("global-network", gn.id, "Global network %s still has %s %s; delete it first.", gn.id, rec.kind, rec.id)
		}
	}
	for _, rec := range st.recs {
		if rec.parent == gn.id { // route analyses go with their global network
			st.deleteRec(rec)
		}
	}
	for _, rec := range st.assocs {
		if rec.kind == "registration" && rec.parent == gn.id {
			return nil, nmConflict("global-network", gn.id, "Global network %s still has registered transit gateways; deregister them first.", gn.id)
		}
	}
	gn.state, gn.settle = "DELETING", ""
	out := st.render(gn)
	st.deleteRec(gn)
	return map[string]any{"GlobalNetwork": out}, nil
}

// ---- sites, devices, links ----

// child returns a site, device, link or connection of a global network.
func (st *nmStore) child(kind, gnID, id string) (*nmRec, *nmErr) {
	if _, err := st.globalNetwork(gnID); err != nil {
		return nil, err
	}
	rec, err := st.get(kind, kind, id)
	if err != nil || rec.parent != gnID {
		return nil, nmNotFound(kind, id)
	}
	return rec, nil
}

func (st *nmStore) newChild(kind string, in map[string]any, fields ...string) (*nmRec, *nmErr) {
	gnID := nmStr(in, "GlobalNetworkId")
	if _, err := st.globalNetwork(gnID); err != nil {
		return nil, err
	}
	if err := nmCheckCap(st, kind); err != nil {
		return nil, err
	}
	if err := nmCheckTags(nil, nmTags(in)); err != nil {
		return nil, err
	}
	id := kind + "-" + nmHex(17)
	arn := nmArn(kind + "/" + gnID + "/" + id)
	data := nmCopyFields(in, fields...)
	data["GlobalNetworkId"] = gnID
	idKey := map[string]string{"site": "SiteId", "device": "DeviceId", "link": "LinkId", "connection": "ConnectionId"}[kind]
	arnKey := map[string]string{"site": "SiteArn", "device": "DeviceArn", "link": "LinkArn", "connection": "ConnectionArn"}[kind]
	data[idKey], data[arnKey] = id, arn
	return st.newRec(kind, id, arn, gnID, data, "PENDING", "AVAILABLE", nmTags(in)), nil
}

func (st *nmStore) requireSite(gnID, siteID string) *nmErr {
	if siteID == "" {
		return nil
	}
	_, err := st.child("site", gnID, siteID)
	return err
}

func nmCreateSite(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	rec, err := st.newChild("site", in, "Description", "Location")
	if err != nil {
		return nil, err
	}
	return map[string]any{"Site": st.render(rec)}, nil
}

func nmGetSites(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	return st.listChildren(in, "site", "Sites", nmFilterIDs(nmStrings(in, "SiteIds")))
}

func nmUpdateSite(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	rec, err := st.child("site", nmStr(in, "GlobalNetworkId"), nmStr(in, "SiteId"))
	if err != nil {
		return nil, err
	}
	nmApply(rec, in, "Description", "Location")
	return map[string]any{"Site": st.render(rec)}, nil
}

func nmDeleteSite(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	rec, err := st.child("site", nmStr(in, "GlobalNetworkId"), nmStr(in, "SiteId"))
	if err != nil {
		return nil, err
	}
	for _, other := range st.recs {
		if (other.kind == "device" || other.kind == "link") && other.data["SiteId"] == rec.id {
			return nil, nmConflict("site", rec.id, "Site %s is still used by %s %s.", rec.id, other.kind, other.id)
		}
	}
	return st.deleteChild(rec, "Site")
}

func nmCreateDevice(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	if err := st.requireSite(nmStr(in, "GlobalNetworkId"), nmStr(in, "SiteId")); err != nil {
		return nil, err
	}
	rec, err := st.newChild("device", in, "AWSLocation", "Description", "Type", "Vendor", "Model", "SerialNumber", "Location", "SiteId")
	if err != nil {
		return nil, err
	}
	return map[string]any{"Device": st.render(rec)}, nil
}

func nmGetDevices(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	ids, site := nmStrings(in, "DeviceIds"), nmStr(in, "SiteId")
	return st.listChildren(in, "device", "Devices", func(r *nmRec) bool {
		return nmFilterIDs(ids)(r) && (site == "" || r.data["SiteId"] == site)
	})
}

func nmUpdateDevice(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	gnID := nmStr(in, "GlobalNetworkId")
	rec, err := st.child("device", gnID, nmStr(in, "DeviceId"))
	if err != nil {
		return nil, err
	}
	if err := st.requireSite(gnID, nmStr(in, "SiteId")); err != nil {
		return nil, err
	}
	nmApply(rec, in, "AWSLocation", "Description", "Type", "Vendor", "Model", "SerialNumber", "Location", "SiteId")
	return map[string]any{"Device": st.render(rec)}, nil
}

func nmDeleteDevice(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	rec, err := st.child("device", nmStr(in, "GlobalNetworkId"), nmStr(in, "DeviceId"))
	if err != nil {
		return nil, err
	}
	for _, other := range st.recs {
		if other.kind == "connection" && (other.data["DeviceId"] == rec.id || other.data["ConnectedDeviceId"] == rec.id) {
			return nil, nmConflict("device", rec.id, "Device %s is still used by connection %s.", rec.id, other.id)
		}
	}
	for _, a := range st.assocs {
		if a.parent == rec.parent && a.data["DeviceId"] == rec.id {
			return nil, nmConflict("device", rec.id, "Device %s still has associations; disassociate them first.", rec.id)
		}
	}
	return st.deleteChild(rec, "Device")
}

func nmCreateLink(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	if err := st.requireSite(nmStr(in, "GlobalNetworkId"), nmStr(in, "SiteId")); err != nil {
		return nil, err
	}
	if err := nmCheckBandwidth(nmMap(in, "Bandwidth")); err != nil {
		return nil, err
	}
	rec, err := st.newChild("link", in, "Description", "Type", "Bandwidth", "Provider", "SiteId")
	if err != nil {
		return nil, err
	}
	return map[string]any{"Link": st.render(rec)}, nil
}

func nmCheckBandwidth(bw map[string]any) *nmErr {
	for _, k := range []string{"UploadSpeed", "DownloadSpeed"} {
		if v, ok := bw[k].(float64); ok && v < 0 {
			return nmValidation("Bandwidth %s can't be negative.", k)
		}
	}
	return nil
}

func nmGetLinks(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	ids := nmStrings(in, "LinkIds")
	site, typ, provider := nmStr(in, "SiteId"), nmStr(in, "Type"), nmStr(in, "Provider")
	return st.listChildren(in, "link", "Links", func(r *nmRec) bool {
		return nmFilterIDs(ids)(r) && (site == "" || r.data["SiteId"] == site) &&
			(typ == "" || r.data["Type"] == typ) && (provider == "" || r.data["Provider"] == provider)
	})
}

func nmUpdateLink(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	rec, err := st.child("link", nmStr(in, "GlobalNetworkId"), nmStr(in, "LinkId"))
	if err != nil {
		return nil, err
	}
	if err := nmCheckBandwidth(nmMap(in, "Bandwidth")); err != nil {
		return nil, err
	}
	nmApply(rec, in, "Description", "Type", "Bandwidth", "Provider")
	return map[string]any{"Link": st.render(rec)}, nil
}

func nmDeleteLink(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	rec, err := st.child("link", nmStr(in, "GlobalNetworkId"), nmStr(in, "LinkId"))
	if err != nil {
		return nil, err
	}
	for _, a := range st.assocs {
		if a.parent == rec.parent && a.data["LinkId"] == rec.id {
			return nil, nmConflict("link", rec.id, "Link %s still has associations; disassociate them first.", rec.id)
		}
	}
	for _, other := range st.recs {
		if other.kind == "connection" && (other.data["LinkId"] == rec.id || other.data["ConnectedLinkId"] == rec.id) {
			return nil, nmConflict("link", rec.id, "Link %s is still used by connection %s.", rec.id, other.id)
		}
	}
	return st.deleteChild(rec, "Link")
}

// ---- connections ----

func (st *nmStore) checkConnectionEnds(gnID string, in map[string]any) *nmErr {
	for _, k := range []string{"DeviceId", "ConnectedDeviceId"} {
		if v := nmStr(in, k); v != "" {
			if _, err := st.child("device", gnID, v); err != nil {
				return err
			}
		}
	}
	for _, k := range []string{"LinkId", "ConnectedLinkId"} {
		if v := nmStr(in, k); v != "" {
			if _, err := st.child("link", gnID, v); err != nil {
				return err
			}
		}
	}
	return nil
}

func nmCreateConnection(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	gnID := nmStr(in, "GlobalNetworkId")
	if nmStr(in, "DeviceId") == nmStr(in, "ConnectedDeviceId") {
		return nil, nmValidation("A connection must join two different devices.")
	}
	if err := st.checkConnectionEnds(gnID, in); err != nil {
		return nil, err
	}
	rec, err := st.newChild("connection", in, "DeviceId", "ConnectedDeviceId", "LinkId", "ConnectedLinkId", "Description")
	if err != nil {
		return nil, err
	}
	return map[string]any{"Connection": st.render(rec)}, nil
}

func nmGetConnections(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	ids, device := nmStrings(in, "ConnectionIds"), nmStr(in, "DeviceId")
	return st.listChildren(in, "connection", "Connections", func(r *nmRec) bool {
		return nmFilterIDs(ids)(r) && (device == "" || r.data["DeviceId"] == device || r.data["ConnectedDeviceId"] == device)
	})
}

func nmUpdateConnection(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	gnID := nmStr(in, "GlobalNetworkId")
	rec, err := st.child("connection", gnID, nmStr(in, "ConnectionId"))
	if err != nil {
		return nil, err
	}
	if err := st.checkConnectionEnds(gnID, in); err != nil {
		return nil, err
	}
	nmApply(rec, in, "LinkId", "ConnectedLinkId", "Description")
	return map[string]any{"Connection": st.render(rec)}, nil
}

func nmDeleteConnection(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	rec, err := st.child("connection", nmStr(in, "GlobalNetworkId"), nmStr(in, "ConnectionId"))
	if err != nil {
		return nil, err
	}
	return st.deleteChild(rec, "Connection")
}

func (st *nmStore) listChildren(in map[string]any, kind, key string, keep func(*nmRec) bool) (map[string]any, *nmErr) {
	gnID := nmStr(in, "GlobalNetworkId")
	if _, err := st.globalNetwork(gnID); err != nil {
		return nil, err
	}
	recs := st.list(kind, func(r *nmRec) bool { return r.parent == gnID && keep(r) })
	page, next, err := nmPage(in, st.renderAll(recs))
	if err != nil {
		return nil, err
	}
	return map[string]any{key: page, "NextToken": nmNonEmpty(next)}, nil
}

func (st *nmStore) deleteChild(rec *nmRec, key string) (map[string]any, *nmErr) {
	rec.state, rec.settle = "DELETING", ""
	out := st.render(rec)
	st.deleteRec(rec)
	return map[string]any{key: out}, nil
}

// nmApply copies the given members of an update request onto a record.
func nmApply(rec *nmRec, in map[string]any, keys ...string) {
	for _, k := range keys {
		if v, ok := in[k]; ok {
			rec.data[k] = v
		}
	}
	rec.state, rec.settle, rec.updated = "UPDATING", "AVAILABLE", nmNow()
}

func nmFilterIDs(ids []string) func(*nmRec) bool {
	return func(r *nmRec) bool { return len(ids) == 0 || slices.Contains(ids, r.id) }
}

func nmNonEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// ---- associations (link, customer gateway, connect peers) ----

func (st *nmStore) checkDeviceLink(gnID string, in map[string]any) *nmErr {
	if _, err := st.child("device", gnID, nmStr(in, "DeviceId")); err != nil {
		return err
	}
	if link := nmStr(in, "LinkId"); link != "" {
		if _, err := st.child("link", gnID, link); err != nil {
			return err
		}
	}
	return nil
}

func nmAssociateLink(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	gnID, device, link := nmStr(in, "GlobalNetworkId"), nmStr(in, "DeviceId"), nmStr(in, "LinkId")
	if err := st.checkDeviceLink(gnID, in); err != nil {
		return nil, err
	}
	if link == "" {
		return nil, nmValidation("LinkId is required.")
	}
	d, err := st.child("device", gnID, device)
	if err != nil {
		return nil, err
	}
	l, err := st.child("link", gnID, link)
	if err != nil {
		return nil, err
	}
	if ds, ls := d.data["SiteId"], l.data["SiteId"]; ds != nil && ls != nil && ds != ls {
		return nil, nmValidation("Device %s and link %s must be in the same site.", device, link)
	}
	key := gnID + "|" + device + "|" + link
	if st.assocs["link-assoc|"+key] != nil {
		return nil, nmConflict("link-association", key, "Link %s is already associated with device %s.", link, device)
	}
	if err := nmCheckCap(st, "link-association"); err != nil {
		return nil, err
	}
	rec := st.newAssoc("link-assoc", key, gnID, nmCopyFields(in, "GlobalNetworkId", "DeviceId", "LinkId"), "LinkAssociationState", "PENDING", "AVAILABLE")
	return map[string]any{"LinkAssociation": st.render(rec)}, nil
}

func nmGetLinkAssociations(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	device, link := nmStr(in, "DeviceId"), nmStr(in, "LinkId")
	return st.listAssocPage(in, "link-assoc", "LinkAssociations", func(r *nmRec) bool {
		return (device == "" || r.data["DeviceId"] == device) && (link == "" || r.data["LinkId"] == link)
	})
}

func nmDisassociateLink(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	key := nmStr(in, "GlobalNetworkId") + "|" + nmStr(in, "DeviceId") + "|" + nmStr(in, "LinkId")
	return st.disassociate("link-assoc", key, "link-association", "LinkAssociation")
}

func nmAssociateCustomerGateway(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	gnID, arn := nmStr(in, "GlobalNetworkId"), nmStr(in, "CustomerGatewayArn")
	if err := st.checkDeviceLink(gnID, in); err != nil {
		return nil, err
	}
	return st.associate(in, "cgw-assoc", gnID+"|"+arn, "customer-gateway-association", "CustomerGatewayAssociation",
		"GlobalNetworkId", "CustomerGatewayArn", "DeviceId", "LinkId")
}

func nmGetCustomerGatewayAssociations(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	arns := nmStrings(in, "CustomerGatewayArns")
	return st.listAssocPage(in, "cgw-assoc", "CustomerGatewayAssociations", func(r *nmRec) bool {
		return len(arns) == 0 || slices.Contains(arns, nmStr(r.data, "CustomerGatewayArn"))
	})
}

func nmDisassociateCustomerGateway(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	return st.disassociate("cgw-assoc", nmStr(in, "GlobalNetworkId")+"|"+nmStr(in, "CustomerGatewayArn"),
		"customer-gateway-association", "CustomerGatewayAssociation")
}

func nmAssociateTGWConnectPeer(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	gnID, arn := nmStr(in, "GlobalNetworkId"), nmStr(in, "TransitGatewayConnectPeerArn")
	// ponytail: connect peers are checked by ARN shape; the EC2 store doesn't key them by ARN.
	if svc, _, _, typ, id, ok := nmArnParts(arn); !ok || svc != "ec2" || typ != "transit-gateway-connect-peer" || !strings.HasPrefix(id, "tgw-connect-peer-") {
		return nil, nmValidation("TransitGatewayConnectPeerArn %s isn't a transit gateway Connect peer ARN.", arn)
	}
	if err := st.checkDeviceLink(gnID, in); err != nil {
		return nil, err
	}
	return st.associate(in, "tgw-cp-assoc", gnID+"|"+arn, "transit-gateway-connect-peer-association",
		"TransitGatewayConnectPeerAssociation", "GlobalNetworkId", "TransitGatewayConnectPeerArn", "DeviceId", "LinkId")
}

func nmGetTGWConnectPeerAssociations(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	arns := nmStrings(in, "TransitGatewayConnectPeerArns")
	return st.listAssocPage(in, "tgw-cp-assoc", "TransitGatewayConnectPeerAssociations", func(r *nmRec) bool {
		return len(arns) == 0 || slices.Contains(arns, nmStr(r.data, "TransitGatewayConnectPeerArn"))
	})
}

func nmDisassociateTGWConnectPeer(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	return st.disassociate("tgw-cp-assoc", nmStr(in, "GlobalNetworkId")+"|"+nmStr(in, "TransitGatewayConnectPeerArn"),
		"transit-gateway-connect-peer-association", "TransitGatewayConnectPeerAssociation")
}

func nmAssociateConnectPeer(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	gnID, peer := nmStr(in, "GlobalNetworkId"), nmStr(in, "ConnectPeerId")
	cp, err := st.get("connect-peer", "connect-peer", peer)
	if err != nil {
		return nil, err
	}
	if core := st.recs[nmStr(cp.data, "CoreNetworkId")]; core == nil || core.parent != gnID {
		return nil, nmValidation("Connect peer %s isn't in global network %s.", peer, gnID)
	}
	if err := st.checkDeviceLink(gnID, in); err != nil {
		return nil, err
	}
	return st.associate(in, "cp-assoc", gnID+"|"+peer, "connect-peer-association", "ConnectPeerAssociation",
		"GlobalNetworkId", "ConnectPeerId", "DeviceId", "LinkId")
}

func nmGetConnectPeerAssociations(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	ids := nmStrings(in, "ConnectPeerIds")
	return st.listAssocPage(in, "cp-assoc", "ConnectPeerAssociations", func(r *nmRec) bool {
		return len(ids) == 0 || slices.Contains(ids, nmStr(r.data, "ConnectPeerId"))
	})
}

func nmDisassociateConnectPeer(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	return st.disassociate("cp-assoc", nmStr(in, "GlobalNetworkId")+"|"+nmStr(in, "ConnectPeerId"),
		"connect-peer-association", "ConnectPeerAssociation")
}

func (st *nmStore) associate(in map[string]any, kind, key, resourceType, outKey string, fields ...string) (map[string]any, *nmErr) {
	if st.assocs[kind+"|"+key] != nil {
		return nil, nmConflict(resourceType, key, "The %s %s already exists.", resourceType, key)
	}
	if err := nmCheckCap(st, resourceType); err != nil {
		return nil, err
	}
	rec := st.newAssoc(kind, key, nmStr(in, "GlobalNetworkId"), nmCopyFields(in, fields...), "State", "PENDING", "AVAILABLE")
	return map[string]any{outKey: st.render(rec)}, nil
}

func (st *nmStore) disassociate(kind, key, resourceType, outKey string) (map[string]any, *nmErr) {
	rec := st.assocs[kind+"|"+key]
	if rec == nil {
		return nil, nmNotFound(resourceType, key)
	}
	rec.state, rec.settle = "DELETING", ""
	out := st.render(rec)
	delete(st.assocs, kind+"|"+key)
	return map[string]any{outKey: out}, nil
}

func (st *nmStore) listAssocPage(in map[string]any, kind, key string, keep func(*nmRec) bool) (map[string]any, *nmErr) {
	gnID := nmStr(in, "GlobalNetworkId")
	if _, err := st.globalNetwork(gnID); err != nil {
		return nil, err
	}
	recs := st.listAssocs(kind, func(r *nmRec) bool { return r.parent == gnID && keep(r) })
	page, next, err := nmPage(in, st.renderAll(recs))
	if err != nil {
		return nil, err
	}
	return map[string]any{key: page, "NextToken": nmNonEmpty(next)}, nil
}

// ---- transit gateway registrations ----

// nmCheckTransitGateway requires a transit gateway ARN whose gateway exists
// in the EC2 store; it runs before the Network Manager lock.
func (s *Server) nmCheckTransitGateway(arn string) *nmErr {
	svc, region, _, typ, id, ok := nmArnParts(arn)
	if !ok || svc != "ec2" || typ != "transit-gateway" || !strings.HasPrefix(id, "tgw-") || region == "" {
		return nmValidation("TransitGatewayArn %s isn't a transit gateway ARN.", arn)
	}
	if tgws, _, err := s.ec2.DescribeTransitGateways([]string{id}, nil, nil, nil); err != nil || len(tgws) == 0 {
		return nmNotFound("transit-gateway", arn)
	}
	return nil
}

func (s *Server) nmCheckTransitGatewayInput(in map[string]any) *nmErr {
	return s.nmCheckTransitGateway(nmStr(in, "TransitGatewayArn"))
}

func (s *Server) nmCheckCustomerGateway(in map[string]any) *nmErr {
	arn := nmStr(in, "CustomerGatewayArn")
	svc, _, _, typ, id, ok := nmArnParts(arn)
	if !ok || svc != "ec2" || typ != "customer-gateway" || !strings.HasPrefix(id, "cgw-") {
		return nmValidation("CustomerGatewayArn %s isn't a customer gateway ARN.", arn)
	}
	if len(s.ec2.DescribeCustomerGateways([]string{id}, nil, nil, nil, nil, nil, nil, nil)) == 0 {
		return nmNotFound("customer-gateway", arn)
	}
	return nil
}

func nmRegisterTransitGateway(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	gnID, arn := nmStr(in, "GlobalNetworkId"), nmStr(in, "TransitGatewayArn")
	if _, err := st.globalNetwork(gnID); err != nil {
		return nil, err
	}
	for _, rec := range st.assocs {
		if rec.kind == "registration" && rec.data["TransitGatewayArn"] == arn {
			return nil, nmConflict("transit-gateway-registration", arn, "Transit gateway %s is already registered with global network %s.", arn, rec.parent)
		}
	}
	if err := nmCheckCap(st, "transit-gateway-registration"); err != nil {
		return nil, err
	}
	rec := st.newAssoc("registration", gnID+"|"+arn, gnID, nmCopyFields(in, "GlobalNetworkId", "TransitGatewayArn"), "RegistrationState", "PENDING", "AVAILABLE")
	return map[string]any{"TransitGatewayRegistration": st.render(rec)}, nil
}

func nmGetTransitGatewayRegistrations(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	arns := nmStrings(in, "TransitGatewayArns")
	return st.listAssocPage(in, "registration", "TransitGatewayRegistrations", func(r *nmRec) bool {
		return len(arns) == 0 || slices.Contains(arns, nmStr(r.data, "TransitGatewayArn"))
	})
}

func nmDeregisterTransitGateway(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	return st.disassociate("registration", nmStr(in, "GlobalNetworkId")+"|"+nmStr(in, "TransitGatewayArn"),
		"transit-gateway-registration", "TransitGatewayRegistration")
}

// ---- route analysis ----

func nmStartRouteAnalysis(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	gnID := nmStr(in, "GlobalNetworkId")
	if _, err := st.globalNetwork(gnID); err != nil {
		return nil, err
	}
	if err := nmCheckCap(st, "route-analysis"); err != nil {
		return nil, err
	}
	src, dst := nmMap(in, "Source"), nmMap(in, "Destination")
	for name, ep := range map[string]map[string]any{"Source": src, "Destination": dst} {
		if nmStr(ep, "TransitGatewayAttachmentArn") == "" && nmStr(ep, "IpAddress") == "" {
			return nil, nmValidation("%s needs a TransitGatewayAttachmentArn or an IpAddress.", name)
		}
	}
	id := nmHex(8) + "-" + nmHex(4) + "-" + nmHex(4) + "-" + nmHex(4) + "-" + nmHex(12)
	// ponytail: no transit gateway routing is modelled, so every analysis
	// completes NOT_CONNECTED with ROUTE_NOT_FOUND.
	path := map[string]any{"CompletionStatus": map[string]any{"ResultCode": "NOT_CONNECTED", "ReasonCode": "ROUTE_NOT_FOUND"}, "Path": []any{}}
	data := map[string]any{
		"GlobalNetworkId": gnID, "OwnerAccountId": nmAccount, "RouteAnalysisId": id,
		"Source": src, "Destination": dst, "IncludeReturnPath": in["IncludeReturnPath"] == true,
		"UseMiddleboxes": in["UseMiddleboxes"] == true, "ForwardPath": path,
	}
	if in["IncludeReturnPath"] == true {
		data["ReturnPath"] = path
	}
	rec := st.newRec("route-analysis", id, "", gnID, data, "RUNNING", "COMPLETED", nil)
	rec.stateKey = "Status"
	return map[string]any{"RouteAnalysis": nmRouteAnalysisOut(st, rec)}, nil
}

func nmRouteAnalysisOut(st *nmStore, rec *nmRec) map[string]any {
	out := st.render(rec)
	out["StartTimestamp"] = nmTime(rec.created)
	return out
}

func nmGetRouteAnalysis(st *nmStore, in map[string]any) (map[string]any, *nmErr) {
	rec, err := st.child("route-analysis", nmStr(in, "GlobalNetworkId"), nmStr(in, "RouteAnalysisId"))
	if err != nil {
		return nil, err
	}
	return map[string]any{"RouteAnalysis": nmRouteAnalysisOut(st, rec)}, nil
}

// nmJSON renders a record definition for network resources.
func nmJSON(v any) string {
	raw, _ := json.Marshal(v)
	return string(raw)
}
