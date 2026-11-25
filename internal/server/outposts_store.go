package server

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
	"sync"
	"time"
)

const (
	opstAccount = "123456789012"
	opstRegion  = "us-east-1"
	opstARNBase = "arn:aws:outposts:" + opstRegion + ":" + opstAccount + ":"
	// ponytail: fixed caps keep stored counts bounded; AWS's own quotas are
	// lower for most of these.
	opstMaxSites       = 200
	opstMaxOutposts    = 200
	opstMaxQuotes      = 500
	opstMaxOrders      = 500
	opstMaxTasks       = 1000
	opstMaxConnections = 500
	opstMaxAssets      = 2000
	opstMaxTags        = 50
	opstMaxListItems   = 20 // capacities, constraints, line items, pools, VPCs, subnets
	opstMaxQuantity    = 10 // per line item
	opstMaxRenewals    = 20 // per outpost
)

type opstSite struct {
	id, arn, name, description, notes string
	tags                              map[string]string
	addresses                         map[string]map[string]any // by AddressType
	rack                              map[string]any
}

type opstOutpost struct {
	id, arn, siteID, name, description, az, azID, hardware, lifecycle string
	tags                                                              map[string]string
	privateConnectivity                                               map[string]any
	created                                                           time.Time
}

type opstStore struct {
	mu          sync.Mutex
	sites       map[string]*opstSite
	siteOrder   []string
	outposts    map[string]*opstOutpost
	outOrder    []string
	quotes      map[string]*opstQuote
	quoteOrder  []string
	orders      map[string]*opstOrder
	orderOrder  []string
	assets      map[string]*opstAsset // by asset id
	assetOrder  []string
	tasks       map[string]*opstTask
	taskOrder   []string
	connections map[string]*opstConnection
	connOrder   []string
	renewals    map[string][]opstSubscription // outpost id -> renewals, oldest first
}

func newOutpostsStore() *opstStore {
	return &opstStore{
		sites: map[string]*opstSite{}, outposts: map[string]*opstOutpost{}, quotes: map[string]*opstQuote{},
		orders: map[string]*opstOrder{}, assets: map[string]*opstAsset{}, tasks: map[string]*opstTask{},
		connections: map[string]*opstConnection{}, renewals: map[string][]opstSubscription{},
	}
}

func opstHex(n int) string {
	b := make([]byte, (n+1)/2)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)[:n]
}

func opstNotFound(kind, id string) *budErr {
	return opstErrorf("NotFoundException", "%s %s not found", kind, id)
}

func opstConflict(format string, args ...any) *budErr {
	return opstErrorf("ConflictException", format, args...)
}

func opstQuota(kind string, limit int) *budErr {
	return opstErrorf("ServiceQuotaExceededException", "you can have at most %d %s", limit, kind)
}

// opstStrs reads a list of strings (query filters arrive as []any).
func opstStrs(in map[string]any, key string) []string { return budStrings(in, key) }

func opstMatches(filter []string, v string) bool {
	if len(filter) == 0 {
		return true
	}
	for _, f := range filter {
		if f == v {
			return true
		}
	}
	return false
}

func opstRemove(order []string, id string) []string {
	for i, v := range order {
		if v == id {
			return append(order[:i:i], order[i+1:]...)
		}
	}
	return order
}

func opstTagsIn(in map[string]any) (map[string]string, *budErr) {
	out := map[string]string{}
	for k, v := range budMap(in, "Tags") {
		if strings.HasPrefix(strings.ToLower(k), "aws:") {
			return nil, opstInvalid("tag keys starting with aws: are reserved")
		}
		out[k], _ = v.(string)
	}
	if len(out) > opstMaxTags {
		return nil, opstInvalid("a resource can have at most %d tags", opstMaxTags)
	}
	return out, nil
}

func opstTagsOut(tags map[string]string) map[string]any {
	out := make(map[string]any, len(tags))
	for k, v := range tags {
		out[k] = v
	}
	return out
}

func (st *opstStore) dispatch(op string, in map[string]any) (map[string]any, *budErr) {
	h := opstHandlers[op]
	if h == nil {
		return nil, opstErrorf("UnknownOperationException", "unknown operation %s", op)
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	return h(st, in, time.Now().UTC())
}

type opstHandler func(st *opstStore, in map[string]any, now time.Time) (map[string]any, *budErr)

var opstHandlers map[string]opstHandler

func init() {
	opstHandlers = map[string]opstHandler{
		"CreateSite":                           (*opstStore).createSite,
		"GetSite":                              (*opstStore).getSite,
		"ListSites":                            (*opstStore).listSites,
		"UpdateSite":                           (*opstStore).updateSite,
		"DeleteSite":                           (*opstStore).deleteSite,
		"GetSiteAddress":                       (*opstStore).getSiteAddress,
		"UpdateSiteAddress":                    (*opstStore).updateSiteAddress,
		"UpdateSiteRackPhysicalProperties":     (*opstStore).updateRack,
		"CreateOutpost":                        (*opstStore).createOutpost,
		"GetOutpost":                           (*opstStore).getOutpost,
		"ListOutposts":                         (*opstStore).listOutposts,
		"UpdateOutpost":                        (*opstStore).updateOutpost,
		"DeleteOutpost":                        (*opstStore).deleteOutpost,
		"StartOutpostDecommission":             (*opstStore).decommission,
		"CreatePrivateConnectivityConfig":      (*opstStore).createPrivateConnectivity,
		"GetPrivateConnectivityConfig":         (*opstStore).getPrivateConnectivity,
		"TagResource":                          (*opstStore).tagResource,
		"UntagResource":                        (*opstStore).untagResource,
		"ListTagsForResource":                  (*opstStore).listTags,
		"GetCatalogItem":                       opstGetCatalogItem,
		"ListCatalogItems":                     opstListCatalogItems,
		"ListOrderableInstanceTypes":           opstListOrderableInstanceTypes,
		"CreateQuote":                          (*opstStore).createQuote,
		"GetQuote":                             (*opstStore).getQuote,
		"ListQuotes":                           (*opstStore).listQuotes,
		"UpdateQuote":                          (*opstStore).updateQuote,
		"DeleteQuote":                          (*opstStore).deleteQuote,
		"CreateOrder":                          (*opstStore).createOrder,
		"GetOrder":                             (*opstStore).getOrder,
		"ListOrders":                           (*opstStore).listOrders,
		"CancelOrder":                          (*opstStore).cancelOrder,
		"ListAssets":                           (*opstStore).listAssets,
		"ListAssetInstances":                   (*opstStore).listAssetInstances,
		"GetOutpostInstanceTypes":              (*opstStore).getOutpostInstanceTypes,
		"GetOutpostSupportedInstanceTypes":     (*opstStore).getSupportedInstanceTypes,
		"StartCapacityTask":                    (*opstStore).startCapacityTask,
		"GetCapacityTask":                      (*opstStore).getCapacityTask,
		"ListCapacityTasks":                    (*opstStore).listCapacityTasks,
		"CancelCapacityTask":                   (*opstStore).cancelCapacityTask,
		"ListBlockingInstancesForCapacityTask": (*opstStore).listBlockingInstances,
		"StartConnection":                      (*opstStore).startConnection,
		"GetConnection":                        (*opstStore).getConnection,
		"GetOutpostBillingInformation":         (*opstStore).billingInformation,
		"GetRenewalPricing":                    (*opstStore).renewalPricing,
		"CreateRenewal":                        (*opstStore).createRenewal,
	}
}

// ---- sites ----

func (st *opstStore) site(id string) (*opstSite, *budErr) {
	if i := strings.LastIndexByte(id, '/'); i >= 0 {
		id = id[i+1:]
	}
	if s := st.sites[id]; s != nil {
		return s, nil
	}
	return nil, opstNotFound("Site", id)
}

func (s *opstSite) out() map[string]any {
	out := map[string]any{
		"SiteId": s.id, "SiteArn": s.arn, "AccountId": opstAccount, "Name": s.name, "Tags": opstTagsOut(s.tags),
	}
	for k, v := range map[string]string{"Description": s.description, "Notes": s.notes} {
		if v != "" {
			out[k] = v
		}
	}
	if a := s.addresses["OPERATING_ADDRESS"]; a != nil {
		out["OperatingAddressCountryCode"] = a["CountryCode"]
		out["OperatingAddressStateOrRegion"] = a["StateOrRegion"]
		out["OperatingAddressCity"] = a["City"]
	}
	if len(s.rack) > 0 {
		out["RackPhysicalProperties"] = budCloneMap(s.rack)
	}
	return out
}

func (st *opstStore) createSite(in map[string]any, _ time.Time) (map[string]any, *budErr) {
	tags, err := opstTagsIn(in)
	if err != nil {
		return nil, err
	}
	if len(st.sites) >= opstMaxSites {
		return nil, opstQuota("sites", opstMaxSites)
	}
	name := budStr(in, "Name")
	for _, s := range st.sites {
		if s.name == name {
			return nil, opstConflict("a site named %s already exists", name)
		}
	}
	id := "os-" + opstHex(17)
	s := &opstSite{
		id: id, arn: opstARNBase + "site/" + id, name: name, description: budStr(in, "Description"), notes: budStr(in, "Notes"),
		tags: tags, addresses: map[string]map[string]any{}, rack: budCloneMap(budMap(in, "RackPhysicalProperties")),
	}
	for typ, key := range map[string]string{"OPERATING_ADDRESS": "OperatingAddress", "SHIPPING_ADDRESS": "ShippingAddress"} {
		if a := budMap(in, key); a != nil {
			s.addresses[typ] = budCloneMap(a)
		}
	}
	st.sites[id] = s
	st.siteOrder = append(st.siteOrder, id)
	return map[string]any{"Site": s.out()}, nil
}

func (st *opstStore) getSite(in map[string]any, _ time.Time) (map[string]any, *budErr) {
	s, err := st.site(budStr(in, "SiteId"))
	if err != nil {
		return nil, err
	}
	return map[string]any{"Site": s.out()}, nil
}

func (st *opstStore) listSites(in map[string]any, _ time.Time) (map[string]any, *budErr) {
	countries, states, cities := opstStrs(in, "OperatingAddressCountryCodeFilter"), opstStrs(in, "OperatingAddressStateOrRegionFilter"), opstStrs(in, "OperatingAddressCityFilter")
	var match []*opstSite
	for _, id := range st.siteOrder {
		s := st.sites[id]
		a := s.addresses["OPERATING_ADDRESS"]
		if (len(countries)+len(states)+len(cities) > 0) && a == nil {
			continue
		}
		if !opstMatches(countries, budStr(a, "CountryCode")) || !opstMatches(states, budStr(a, "StateOrRegion")) || !opstMatches(cities, budStr(a, "City")) {
			continue
		}
		match = append(match, s)
	}
	start, end, next, err := opstPageOf(in, len(match))
	if err != nil {
		return nil, err
	}
	list := make([]any, 0, end-start)
	for _, s := range match[start:end] {
		list = append(list, s.out())
	}
	return map[string]any{"Sites": list, "NextToken": next}, nil
}

func (st *opstStore) updateSite(in map[string]any, _ time.Time) (map[string]any, *budErr) {
	s, err := st.site(budStr(in, "SiteId"))
	if err != nil {
		return nil, err
	}
	if v := budStr(in, "Name"); v != "" && v != s.name {
		for _, other := range st.sites {
			if other.name == v {
				return nil, opstConflict("a site named %s already exists", v)
			}
		}
		s.name = v
	}
	if v := budStr(in, "Description"); v != "" {
		s.description = v
	}
	if v := budStr(in, "Notes"); v != "" {
		s.notes = v
	}
	return map[string]any{"Site": s.out()}, nil
}

func (st *opstStore) siteOutposts(siteID string) []*opstOutpost {
	var out []*opstOutpost
	for _, id := range st.outOrder {
		if o := st.outposts[id]; o.siteID == siteID {
			out = append(out, o)
		}
	}
	return out
}

// siteHardwareInTransit reports whether an order for an outpost at the site
// is shipping or being installed, which freezes its address and rack details.
func (st *opstStore) siteHardwareInTransit(siteID string) bool {
	for _, o := range st.siteOutposts(siteID) {
		for _, ord := range st.orders {
			if ord.outpostID == o.id && (ord.status() == "PROCESSING" || ord.status() == "INSTALLING") {
				return true
			}
		}
	}
	return false
}

func (st *opstStore) deleteSite(in map[string]any, _ time.Time) (map[string]any, *budErr) {
	s, err := st.site(budStr(in, "SiteId"))
	if err != nil {
		return nil, err
	}
	if len(st.siteOutposts(s.id)) > 0 {
		return nil, opstConflict("site %s still has outposts; delete them first", s.id)
	}
	delete(st.sites, s.id)
	st.siteOrder = opstRemove(st.siteOrder, s.id)
	return map[string]any{}, nil
}

func (st *opstStore) getSiteAddress(in map[string]any, _ time.Time) (map[string]any, *budErr) {
	s, err := st.site(budStr(in, "SiteId"))
	if err != nil {
		return nil, err
	}
	typ := budStr(in, "AddressType")
	a := s.addresses[typ]
	if a == nil {
		return nil, opstNotFound("Address "+typ+" of site", s.id)
	}
	return map[string]any{"SiteId": s.id, "AddressType": typ, "Address": budCloneMap(a)}, nil
}

func (st *opstStore) updateSiteAddress(in map[string]any, _ time.Time) (map[string]any, *budErr) {
	s, err := st.site(budStr(in, "SiteId"))
	if err != nil {
		return nil, err
	}
	if st.siteHardwareInTransit(s.id) {
		return nil, opstConflict("site %s has hardware shipping or being installed", s.id)
	}
	typ := budStr(in, "AddressType")
	s.addresses[typ] = budCloneMap(budMap(in, "Address"))
	return map[string]any{"AddressType": typ, "Address": budCloneMap(s.addresses[typ])}, nil
}

func (st *opstStore) updateRack(in map[string]any, _ time.Time) (map[string]any, *budErr) {
	s, err := st.site(budStr(in, "SiteId"))
	if err != nil {
		return nil, err
	}
	if st.siteHardwareInTransit(s.id) {
		return nil, opstConflict("site %s has hardware shipping or being installed", s.id)
	}
	rack := budCloneMap(s.rack)
	if rack == nil {
		rack = map[string]any{}
	}
	for k, v := range in {
		if k != "SiteId" {
			rack[k] = v
		}
	}
	s.rack = rack
	return map[string]any{"Site": s.out()}, nil
}

// ---- outposts ----

func (st *opstStore) outpost(id string) (*opstOutpost, *budErr) {
	if i := strings.LastIndexByte(id, '/'); i >= 0 {
		id = id[i+1:]
	}
	if o := st.outposts[id]; o != nil {
		return o, nil
	}
	return nil, opstNotFound("Outpost", id)
}

func (o *opstOutpost) out(st *opstStore) map[string]any {
	out := map[string]any{
		"OutpostId": o.id, "OutpostArn": o.arn, "OwnerId": opstAccount, "SiteId": o.siteID, "Name": o.name,
		"LifeCycleStatus": o.lifecycle, "AvailabilityZone": o.az, "AvailabilityZoneId": o.azID,
		"Tags": opstTagsOut(o.tags), "SupportedHardwareType": o.hardware, "Generation": "GENERATION_2", "RackScalingType": "SINGLE_RACK",
	}
	if s := st.sites[o.siteID]; s != nil {
		out["SiteArn"] = s.arn
	}
	if o.description != "" {
		out["Description"] = o.description
	}
	return out
}

func (st *opstStore) createOutpost(in map[string]any, now time.Time) (map[string]any, *budErr) {
	tags, err := opstTagsIn(in)
	if err != nil {
		return nil, err
	}
	s, err := st.site(budStr(in, "SiteId"))
	if err != nil {
		return nil, err
	}
	if len(st.outposts) >= opstMaxOutposts {
		return nil, opstQuota("outposts", opstMaxOutposts)
	}
	az, azID := budStr(in, "AvailabilityZone"), budStr(in, "AvailabilityZoneId")
	if az == "" {
		az = opstRegion + "a"
	}
	if azID == "" {
		azID = "use1-az1"
	}
	hw := budStr(in, "SupportedHardwareType")
	if hw == "" {
		hw = "RACK"
	}
	id := "op-" + opstHex(17)
	o := &opstOutpost{
		id: id, arn: opstARNBase + "outpost/" + id, siteID: s.id, name: budStr(in, "Name"), description: budStr(in, "Description"),
		az: az, azID: azID, hardware: hw, lifecycle: "ACTIVE", tags: tags, created: now,
	}
	st.outposts[id] = o
	st.outOrder = append(st.outOrder, id)
	return map[string]any{"Outpost": o.out(st)}, nil
}

func (st *opstStore) getOutpost(in map[string]any, _ time.Time) (map[string]any, *budErr) {
	o, err := st.outpost(budStr(in, "OutpostId"))
	if err != nil {
		return nil, err
	}
	return map[string]any{"Outpost": o.out(st)}, nil
}

func (st *opstStore) listOutposts(in map[string]any, _ time.Time) (map[string]any, *budErr) {
	life, azs, azIDs := opstStrs(in, "LifeCycleStatusFilter"), opstStrs(in, "AvailabilityZoneFilter"), opstStrs(in, "AvailabilityZoneIdFilter")
	var match []*opstOutpost
	for _, id := range st.outOrder {
		o := st.outposts[id]
		if opstMatches(life, o.lifecycle) && opstMatches(azs, o.az) && opstMatches(azIDs, o.azID) {
			match = append(match, o)
		}
	}
	start, end, next, err := opstPageOf(in, len(match))
	if err != nil {
		return nil, err
	}
	list := make([]any, 0, end-start)
	for _, o := range match[start:end] {
		list = append(list, o.out(st))
	}
	return map[string]any{"Outposts": list, "NextToken": next}, nil
}

func (st *opstStore) updateOutpost(in map[string]any, _ time.Time) (map[string]any, *budErr) {
	o, err := st.outpost(budStr(in, "OutpostId"))
	if err != nil {
		return nil, err
	}
	if hw := budStr(in, "SupportedHardwareType"); hw != "" && hw != o.hardware {
		for _, ord := range st.orders {
			if ord.outpostID == o.id && ord.status() != "CANCELLED" {
				return nil, opstConflict("the hardware type of outpost %s can't change once it has orders", o.id)
			}
		}
		o.hardware = hw
	}
	if v := budStr(in, "Name"); v != "" {
		o.name = v
	}
	if v, ok := in["Description"].(string); ok {
		o.description = v
	}
	return map[string]any{"Outpost": o.out(st)}, nil
}

func (st *opstStore) deleteOutpost(in map[string]any, _ time.Time) (map[string]any, *budErr) {
	o, err := st.outpost(budStr(in, "OutpostId"))
	if err != nil {
		return nil, err
	}
	if o.lifecycle != "RETIRING" {
		for _, ord := range st.orders {
			if ord.outpostID == o.id && ord.status() != "CANCELLED" {
				return nil, opstConflict("outpost %s has order %s; cancel it or decommission the outpost first", o.id, ord.id)
			}
		}
	}
	for _, t := range st.tasks {
		if t.outpostID == o.id && !t.terminal() {
			return nil, opstConflict("outpost %s has capacity task %s in progress", o.id, t.id)
		}
	}
	for _, aid := range append([]string(nil), st.assetOrder...) {
		if st.assets[aid].outpostID == o.id {
			st.dropAsset(aid)
		}
	}
	for _, id := range append([]string(nil), st.taskOrder...) {
		if st.tasks[id].outpostID == o.id {
			delete(st.tasks, id)
			st.taskOrder = opstRemove(st.taskOrder, id)
		}
	}
	for _, id := range append([]string(nil), st.orderOrder...) {
		if st.orders[id].outpostID == o.id {
			delete(st.orders, id)
			st.orderOrder = opstRemove(st.orderOrder, id)
		}
	}
	delete(st.outposts, o.id)
	delete(st.renewals, o.id)
	st.outOrder = opstRemove(st.outOrder, o.id)
	return map[string]any{}, nil
}

func (st *opstStore) decommission(in map[string]any, _ time.Time) (map[string]any, *budErr) {
	o, err := st.outpost(budStr(in, "OutpostIdentifier"))
	if err != nil {
		return nil, err
	}
	if o.lifecycle == "RETIRING" {
		return nil, opstConflict("outpost %s is already being decommissioned", o.id)
	}
	blocking := map[string]bool{}
	for _, ord := range st.orders {
		if ord.outpostID != o.id {
			continue
		}
		switch ord.status() {
		case "RECEIVED", "PENDING":
			blocking["OUTPOST_ORDER_CANCELLABLE"] = true
		case "PROCESSING", "INSTALLING":
			blocking["OUTPOST_ORDER_INTERVENTION_REQUIRED"] = true
		}
	}
	types := make([]any, 0, len(blocking))
	for _, t := range []string{"OUTPOST_ORDER_CANCELLABLE", "OUTPOST_ORDER_INTERVENTION_REQUIRED"} {
		if blocking[t] {
			types = append(types, t)
		}
	}
	switch {
	case len(types) > 0:
		return map[string]any{"Status": "BLOCKED", "BlockingResourceTypes": types}, nil
	case budBool(in, "ValidateOnly"):
		return map[string]any{"Status": "SKIPPED", "BlockingResourceTypes": types}, nil
	}
	o.lifecycle = "RETIRING"
	for _, a := range st.assets {
		if a.outpostID == o.id {
			a.state = "RETIRING"
		}
	}
	return map[string]any{"Status": "REQUESTED", "BlockingResourceTypes": types}, nil
}

func (st *opstStore) createPrivateConnectivity(in map[string]any, _ time.Time) (map[string]any, *budErr) {
	o, err := st.outpost(budStr(in, "OutpostId"))
	if err != nil {
		return nil, err
	}
	if o.privateConnectivity != nil {
		return nil, opstConflict("outpost %s already has a private connectivity configuration", o.id)
	}
	vpcs := budList(in, "VpcInformationList")
	if len(vpcs) == 0 || len(vpcs) > opstMaxListItems {
		return nil, opstInvalid("VpcInformationList must have 1 to %d entries", opstMaxListItems)
	}
	for _, v := range vpcs {
		if len(budList(v.(map[string]any), "SubnetIds")) > opstMaxListItems {
			return nil, opstInvalid("a VPC can list at most %d subnets", opstMaxListItems)
		}
	}
	role := "arn:aws:iam::" + opstAccount + ":role/aws-service-role/outposts.amazonaws.com/AWSServiceRoleForOutposts_" + o.id
	o.privateConnectivity = map[string]any{
		"RoleArn": role, "ProvisioningRoleArn": role, "PrivateConnectivityStatus": "ENABLED", "VpcInformationList": budClone(vpcs),
	}
	return map[string]any{"OutpostId": o.id, "PrivateConnectivityConfig": budCloneMap(o.privateConnectivity)}, nil
}

func (st *opstStore) getPrivateConnectivity(in map[string]any, _ time.Time) (map[string]any, *budErr) {
	o, err := st.outpost(budStr(in, "OutpostId"))
	if err != nil {
		return nil, err
	}
	if o.privateConnectivity == nil {
		return nil, opstNotFound("Private connectivity configuration of outpost", o.id)
	}
	return map[string]any{"PrivateConnectivityConfig": budCloneMap(o.privateConnectivity)}, nil
}

// ---- tags ----

func (st *opstStore) taggable(arn string) (*map[string]string, *budErr) {
	rest, ok := strings.CutPrefix(arn, opstARNBase)
	if ok {
		kind, id, _ := strings.Cut(rest, "/")
		switch kind {
		case "outpost":
			if o := st.outposts[id]; o != nil {
				return &o.tags, nil
			}
		case "site":
			if s := st.sites[id]; s != nil {
				return &s.tags, nil
			}
		}
	}
	return nil, opstNotFound("Resource", arn)
}

func (st *opstStore) tagResource(in map[string]any, _ time.Time) (map[string]any, *budErr) {
	add, err := opstTagsIn(in)
	if err != nil {
		return nil, err
	}
	tags, err := st.taggable(budStr(in, "ResourceArn"))
	if err != nil {
		return nil, err
	}
	merged, err := budMergeTags(*tags, add, opstMaxTags, "ValidationException")
	if err != nil {
		return nil, err
	}
	*tags = merged
	return map[string]any{}, nil
}

func (st *opstStore) untagResource(in map[string]any, _ time.Time) (map[string]any, *budErr) {
	tags, err := st.taggable(budStr(in, "ResourceArn"))
	if err != nil {
		return nil, err
	}
	*tags = budDropTags(*tags, budStrings(in, "TagKeys"))
	return map[string]any{}, nil
}

func (st *opstStore) listTags(in map[string]any, _ time.Time) (map[string]any, *budErr) {
	tags, err := st.taggable(budStr(in, "ResourceArn"))
	if err != nil {
		return nil, err
	}
	return map[string]any{"Tags": opstTagsOut(*tags)}, nil
}
