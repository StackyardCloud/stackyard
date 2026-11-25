package server

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"
)

// Outposts assets (installed by fulfilled orders), capacity tasks that
// re-slot instance capacity on them (REQUESTED -> IN_PROGRESS -> COMPLETED,
// one step per read) and service-link connections to assets. No EC2
// instances run on emulated assets, so instance listings are empty.

type opstPool struct {
	instanceType string
	count        int
}

type opstAsset struct {
	id, outpostID, orderID, lineID, rackID, hostID, family, state string
	maxVcpus                                                      int
	pools                                                         []opstPool
	macs                                                          []string
	elevation                                                     float64
}

type opstTask struct {
	id, outpostID, orderID, assetID, action, status, failType, failReason string
	pools                                                                 []opstPool
	exclude                                                               map[string]any
	dryRun                                                                bool
	created, modified, completed                                          time.Time
}

type opstConnection struct {
	id, assetID, clientKey, serverKey, underlay string
}

// opstAssetID returns a 10-digit asset id (the model's AssetIdInput).
func opstAssetID() string {
	b := make([]byte, 10)
	_, _ = rand.Read(b)
	for i := range b {
		b[i] = '0' + b[i]%10
	}
	return string(b)
}

func (st *opstStore) installAssets(o *opstOrder) {
	rack := "fr-" + opstHex(12)
	elevation := 1.0
	for _, l := range o.lines {
		c := opstCatalogItemByID(l.catalog)
		for i := 0; i < l.quantity && len(st.assets) < opstMaxAssets; i++ {
			a := &opstAsset{
				id: opstAssetID(), outpostID: o.outpostID, orderID: o.id, lineID: l.id, rackID: rack, hostID: "h-" + opstHex(17),
				family: c.family, state: "ACTIVE", maxVcpus: c.vcpus, elevation: elevation,
				pools: []opstPool{{c.family + ".xlarge", c.vcpus / 4}},
			}
			mac := opstHex(10)
			a.macs = []string{fmt.Sprintf("02:%s:%s:%s:%s:%s", mac[0:2], mac[2:4], mac[4:6], mac[6:8], mac[8:10])}
			elevation += 2
			st.assets[a.id] = a
			st.assetOrder = append(st.assetOrder, a.id)
		}
	}
}

func (st *opstStore) dropAsset(id string) {
	delete(st.assets, id)
	st.assetOrder = opstRemove(st.assetOrder, id)
	for cid, c := range st.connections {
		if c.assetID == id {
			delete(st.connections, cid)
			st.connOrder = opstRemove(st.connOrder, cid)
		}
	}
}

func opstPoolsOut(pools []opstPool) []any {
	out := make([]any, 0, len(pools))
	for _, p := range pools {
		out = append(out, map[string]any{"InstanceType": p.instanceType, "Count": p.count})
	}
	return out
}

func (a *opstAsset) out() map[string]any {
	return map[string]any{
		"AssetId": a.id, "RackId": a.rackID, "AssetType": "COMPUTE", "AssetLocation": map[string]any{"RackElevation": a.elevation},
		"ComputeAttributes": map[string]any{
			"HostId": a.hostID, "State": a.state, "InstanceFamilies": []any{a.family},
			"InstanceTypeCapacities": opstPoolsOut(a.pools), "MaxVcpus": a.maxVcpus,
		},
	}
}

func (st *opstStore) listAssets(in map[string]any, _ time.Time) (map[string]any, *budErr) {
	o, err := st.outpost(budStr(in, "OutpostIdentifier"))
	if err != nil {
		return nil, err
	}
	hosts, states, types := opstStrs(in, "HostIdFilter"), opstStrs(in, "StatusFilter"), opstStrs(in, "AssetTypeFilter")
	var match []any
	for _, id := range st.assetOrder {
		a := st.assets[id]
		if a.outpostID == o.id && opstMatches(hosts, a.hostID) && opstMatches(states, a.state) && opstMatches(types, "COMPUTE") {
			match = append(match, a.out())
		}
	}
	start, end, next, err := opstPageOf(in, len(match))
	if err != nil {
		return nil, err
	}
	return map[string]any{"Assets": append([]any{}, match[start:end]...), "NextToken": next}, nil
}

func (st *opstStore) listAssetInstances(in map[string]any, _ time.Time) (map[string]any, *budErr) {
	if _, err := st.outpost(budStr(in, "OutpostIdentifier")); err != nil {
		return nil, err
	}
	if _, _, _, err := opstPageOf(in, 0); err != nil {
		return nil, err
	}
	return map[string]any{"AssetInstances": []any{}}, nil
}

func (st *opstStore) families(outpostID string) map[string]bool {
	out := map[string]bool{}
	for _, a := range st.assets {
		if a.outpostID == outpostID && a.state == "ACTIVE" {
			out[a.family] = true
		}
	}
	return out
}

func (st *opstStore) getOutpostInstanceTypes(in map[string]any, _ time.Time) (map[string]any, *budErr) {
	o, err := st.outpost(budStr(in, "OutpostId"))
	if err != nil {
		return nil, err
	}
	types := opstTypesOf(st.families(o.id))
	start, end, next, err := opstPageOf(in, len(types))
	if err != nil {
		return nil, err
	}
	return map[string]any{"InstanceTypes": types[start:end], "NextToken": next, "OutpostId": o.id, "OutpostArn": o.arn}, nil
}

func (st *opstStore) getSupportedInstanceTypes(in map[string]any, _ time.Time) (map[string]any, *budErr) {
	o, err := st.outpost(budStr(in, "OutpostIdentifier"))
	if err != nil {
		return nil, err
	}
	families := map[string]bool{}
	switch {
	case budStr(in, "OrderId") != "":
		ord, err := st.order(budStr(in, "OrderId"))
		if err != nil || ord.outpostID != o.id {
			return nil, opstNotFound("Order", budStr(in, "OrderId"))
		}
		for _, l := range ord.lines {
			families[opstCatalogItemByID(l.catalog).family] = true
		}
	case budStr(in, "AssetId") != "":
		a := st.assets[budStr(in, "AssetId")]
		if a == nil || a.outpostID != o.id {
			return nil, opstNotFound("Asset", budStr(in, "AssetId"))
		}
		families[a.family] = true
	default:
		families = st.families(o.id)
	}
	types := opstTypesOf(families)
	start, end, next, err := opstPageOf(in, len(types))
	if err != nil {
		return nil, err
	}
	return map[string]any{"InstanceTypes": types[start:end], "NextToken": next}, nil
}

// ---- capacity tasks ----

func (t *opstTask) terminal() bool {
	return t.status == "COMPLETED" || t.status == "FAILED" || t.status == "CANCELLED"
}

func (t *opstTask) fail(typ, reason string, now time.Time) {
	t.status, t.failType, t.failReason, t.modified, t.completed = "FAILED", typ, reason, now, now
}

// scope returns the active compute assets a task re-slots.
func (st *opstStore) scope(t *opstTask) []*opstAsset {
	var out []*opstAsset
	for _, id := range st.assetOrder {
		a := st.assets[id]
		if a.outpostID != t.outpostID || a.state != "ACTIVE" {
			continue
		}
		if (t.assetID != "" && a.id != t.assetID) || (t.orderID != "" && a.orderID != t.orderID) {
			continue
		}
		out = append(out, a)
	}
	return out
}

// plan places the task's pools on its assets, filling each asset's vCPUs
// with pools of its family. It returns the new pools per asset, or a reason.
func (st *opstStore) plan(t *opstTask) (map[string][]opstPool, string) {
	assets := st.scope(t)
	if len(assets) == 0 {
		return nil, "no active compute assets are in the task's scope"
	}
	left := make([]int, len(t.pools))
	for i, p := range t.pools {
		left[i] = p.count
	}
	out := map[string][]opstPool{}
	for _, a := range assets {
		free := a.maxVcpus
		pools := []opstPool{}
		for i, p := range t.pools {
			it := opstInstanceTypeByName(p.instanceType)
			if it.family != a.family || left[i] == 0 {
				continue
			}
			n := min(left[i], free/it.vcpus)
			if n > 0 {
				pools = append(pools, opstPool{p.instanceType, n})
				left[i] -= n
				free -= n * it.vcpus
			}
		}
		out[a.id] = pools
	}
	for i, n := range left {
		if n > 0 {
			return nil, fmt.Sprintf("%d %s instances don't fit on the assets", n, t.pools[i].instanceType)
		}
	}
	return out, ""
}

// step advances a task one state; completion applies its plan.
func (st *opstStore) step(t *opstTask, now time.Time) {
	switch t.status {
	case "REQUESTED":
		t.status, t.modified = "IN_PROGRESS", now
	case "IN_PROGRESS":
		plan, reason := st.plan(t)
		if reason != "" {
			t.fail("UNSUPPORTED_CAPACITY_CONFIGURATION", reason, now)
			return
		}
		if !t.dryRun {
			for id, pools := range plan {
				st.assets[id].pools = pools
			}
		}
		t.status, t.modified, t.completed = "COMPLETED", now, now
	case "CANCELLATION_IN_PROGRESS":
		t.status, t.modified, t.completed = "CANCELLED", now, now
	}
}

func (st *opstStore) task(outpost, id string) (*opstTask, *budErr) {
	o, err := st.outpost(outpost)
	if err != nil {
		return nil, err
	}
	if t := st.tasks[id]; t != nil && t.outpostID == o.id {
		return t, nil
	}
	return nil, opstNotFound("Capacity task", id)
}

func (st *opstStore) startCapacityTask(in map[string]any, now time.Time) (map[string]any, *budErr) {
	o, err := st.outpost(budStr(in, "OutpostIdentifier"))
	if err != nil {
		return nil, err
	}
	if len(st.tasks) >= opstMaxTasks && !st.evictFinishedTask() {
		return nil, opstInvalid("at most %d capacity tasks are kept", opstMaxTasks)
	}
	for _, t := range st.tasks {
		if t.outpostID == o.id && !t.terminal() {
			return nil, opstConflict("capacity task %s is still in progress on outpost %s", t.id, o.id)
		}
	}
	if err := opstListMax(in, "InstancePools"); err != nil {
		return nil, err
	}
	t := &opstTask{
		id: "cap-" + opstHex(17), outpostID: o.id, orderID: budStr(in, "OrderId"), assetID: budStr(in, "AssetId"),
		action: budStr(in, "TaskActionOnBlockingInstances"), status: "REQUESTED", dryRun: budBool(in, "DryRun"),
		exclude: budCloneMap(budMap(in, "InstancesToExclude")), created: now, modified: now,
	}
	if t.action == "" {
		t.action = "WAIT_FOR_EVACUATION"
	}
	if t.orderID != "" {
		if ord := st.orders[t.orderID]; ord == nil || ord.outpostID != o.id {
			return nil, opstNotFound("Order", t.orderID)
		}
	}
	if t.assetID != "" {
		if a := st.assets[t.assetID]; a == nil || a.outpostID != o.id {
			return nil, opstNotFound("Asset", t.assetID)
		}
	}
	if err := opstListMax(t.exclude, "Instances", "AccountIds", "Services"); err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	unsupported := ""
	for _, p := range budList(in, "InstancePools") {
		m := p.(map[string]any)
		name := budStr(m, "InstanceType")
		if seen[name] {
			return nil, opstInvalid("instance type %s appears in more than one pool", name)
		}
		seen[name] = true
		if opstInstanceTypeByName(name) == nil && unsupported == "" {
			unsupported = name
		}
		t.pools = append(t.pools, opstPool{name, budInt(m, "Count", 0)})
	}
	if unsupported != "" {
		t.fail("UNSUPPORTED_CAPACITY_CONFIGURATION", "instance type "+unsupported+" is not supported on Outposts", now)
	} else if _, reason := st.plan(t); reason != "" {
		t.fail("UNSUPPORTED_CAPACITY_CONFIGURATION", reason, now)
	}
	st.tasks[t.id] = t
	st.taskOrder = append(st.taskOrder, t.id)
	return st.taskOut(t), nil
}

// evictFinishedTask drops the oldest finished capacity task to make room.
func (st *opstStore) evictFinishedTask() bool {
	for _, id := range st.taskOrder {
		if st.tasks[id].terminal() {
			delete(st.tasks, id)
			st.taskOrder = opstRemove(st.taskOrder, id)
			return true
		}
	}
	return false
}

func (st *opstStore) taskOut(t *opstTask) map[string]any {
	out := map[string]any{
		"CapacityTaskId": t.id, "OutpostId": t.outpostID, "RequestedInstancePools": opstPoolsOut(t.pools), "DryRun": t.dryRun,
		"CapacityTaskStatus": t.status, "CreationDate": budEpoch(t.created), "LastModifiedDate": budEpoch(t.modified),
		"TaskActionOnBlockingInstances": t.action,
	}
	if t.exclude != nil {
		out["InstancesToExclude"] = budCloneMap(t.exclude)
	}
	if t.orderID != "" {
		out["OrderId"] = t.orderID
	}
	if t.assetID != "" {
		out["AssetId"] = t.assetID
	}
	if !t.completed.IsZero() {
		out["CompletionDate"] = budEpoch(t.completed)
	}
	if t.status == "FAILED" {
		out["Failed"] = map[string]any{"Reason": t.failReason, "Type": t.failType}
	}
	return out
}

func (st *opstStore) getCapacityTask(in map[string]any, now time.Time) (map[string]any, *budErr) {
	t, err := st.task(budStr(in, "OutpostIdentifier"), budStr(in, "CapacityTaskId"))
	if err != nil {
		return nil, err
	}
	st.step(t, now)
	return st.taskOut(t), nil
}

func (st *opstStore) listCapacityTasks(in map[string]any, _ time.Time) (map[string]any, *budErr) {
	outpost := opstStripARN(budStr(in, "OutpostIdentifierFilter"))
	statuses := opstStrs(in, "CapacityTaskStatusFilter")
	var match []any
	for _, id := range st.taskOrder {
		t := st.tasks[id]
		if (outpost == "" || t.outpostID == outpost) && opstMatches(statuses, t.status) {
			full := st.taskOut(t)
			sum := map[string]any{}
			for _, k := range []string{"CapacityTaskId", "OutpostId", "OrderId", "AssetId", "CapacityTaskStatus", "CreationDate", "CompletionDate", "LastModifiedDate"} {
				if v, ok := full[k]; ok {
					sum[k] = v
				}
			}
			match = append(match, sum)
		}
	}
	start, end, next, err := opstPageOf(in, len(match))
	if err != nil {
		return nil, err
	}
	return map[string]any{"CapacityTasks": append([]any{}, match[start:end]...), "NextToken": next}, nil
}

func (st *opstStore) cancelCapacityTask(in map[string]any, now time.Time) (map[string]any, *budErr) {
	t, err := st.task(budStr(in, "OutpostIdentifier"), budStr(in, "CapacityTaskId"))
	if err != nil {
		return nil, err
	}
	if t.terminal() || t.status == "CANCELLATION_IN_PROGRESS" {
		return nil, opstConflict("capacity task %s is %s", t.id, t.status)
	}
	t.status, t.modified = "CANCELLATION_IN_PROGRESS", now
	return map[string]any{}, nil
}

func (st *opstStore) listBlockingInstances(in map[string]any, _ time.Time) (map[string]any, *budErr) {
	if _, err := st.task(budStr(in, "OutpostIdentifier"), budStr(in, "CapacityTaskId")); err != nil {
		return nil, err
	}
	if _, _, _, err := opstPageOf(in, 0); err != nil {
		return nil, err
	}
	return map[string]any{"BlockingInstances": []any{}}, nil
}

// ---- connections ----

func (st *opstStore) startConnection(in map[string]any, _ time.Time) (map[string]any, *budErr) {
	a := st.assets[budStr(in, "AssetId")]
	if a == nil {
		return nil, opstNotFound("Asset", budStr(in, "AssetId"))
	}
	if a.state != "ACTIVE" {
		return nil, opstInvalid("asset %s is %s", a.id, a.state)
	}
	if len(st.connections) >= opstMaxConnections {
		oldest := st.connOrder[0]
		delete(st.connections, oldest)
		st.connOrder = st.connOrder[1:]
	}
	key := make([]byte, 32)
	_, _ = rand.Read(key)
	n := len(st.connOrder) + 1
	c := &opstConnection{
		id: opstHex(32), assetID: a.id, clientKey: budStr(in, "ClientPublicKey"),
		serverKey: base64.StdEncoding.EncodeToString(key), underlay: fmt.Sprintf("10.0.%d.%d", n/250, n%250+1),
	}
	st.connections[c.id] = c
	st.connOrder = append(st.connOrder, c.id)
	return map[string]any{"ConnectionId": c.id, "UnderlayIpAddress": c.underlay}, nil
}

func (st *opstStore) getConnection(in map[string]any, _ time.Time) (map[string]any, *budErr) {
	c := st.connections[budStr(in, "ConnectionId")]
	if c == nil {
		return nil, opstNotFound("Connection", budStr(in, "ConnectionId"))
	}
	return map[string]any{"ConnectionId": c.id, "ConnectionDetails": map[string]any{
		"ClientPublicKey": c.clientKey, "ServerPublicKey": c.serverKey, "ServerEndpoint": "52.94.0.10:51820",
		"ClientTunnelAddress": "169.254.0.2/30", "ServerTunnelAddress": "169.254.0.1/30", "AllowedIps": []any{"169.254.0.0/30"},
	}}, nil
}
