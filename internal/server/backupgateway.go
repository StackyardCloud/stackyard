package server

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

// AWS Backup gateway: awsJson1_0, X-Amz-Target BackupOnPremises_v20210101.*,
// signed backup-gateway (the CLI 2.42.1 model lists smithy-rpc-v2-cbor first,
// but the CLI sends JSON 1.0). Gateways associate with imported hypervisors;
// a hypervisor goes PENDING -> ONLINE on its next read, and a metadata sync
// (CREATED -> RUNNING -> SUCCEEDED, one step per GetHypervisor while ONLINE)
// discovers its virtual machines. VMs are AWS Backup resources: Backup's
// StartBackupJob resolves their names here, and LastBackupDate comes from
// Backup's completed recovery points. Lock order is always Backup's store
// before this one; this store never calls into Backup while holding its lock.
// Hypervisor credentials are never stored or returned.

const (
	bgwAccount        = "123456789012"
	bgwRegion         = "us-east-1"
	bgwARNBase        = "arn:aws:backup-gateway:" + bgwRegion + ":" + bgwAccount + ":"
	bgwMaxGateways    = 100
	bgwMaxHypervisors = 100
	bgwVMsPerHost     = 3
	bgwMaxTags        = 50
	bgwMaxIntervals   = 20
	bgwMaxMappings    = 100
	bgwSoftware       = "2.0.15"
	bgwNextSoftware   = "2.0.16"
)

var bgwAPI = &budJSONAPI{
	prefix: "BackupOnPremises_v20210101.", signing: "backup-gateway", contentType: "application/x-amz-json-1.0",
	invalidCode: "ValidationException", model: awsmodels.BackupGatewayModel, respondError: bgwRespondError,
}

// bgwRespondError writes Backup gateway's error shape, which carries the
// code in an ErrorCode member as well as __type.
func bgwRespondError(w http.ResponseWriter, e *budErr) {
	body, _ := json.Marshal(map[string]string{"__type": e.code, "ErrorCode": e.code, "Message": e.msg})
	w.Header().Set("Content-Type", "application/x-amz-json-1.0")
	w.Header().Set("X-Amzn-ErrorType", e.code)
	w.WriteHeader(e.status)
	_, _ = w.Write(body)
}

func init() {
	bgwAPI.handlers = map[string]budHandler{}
	for name, h := range map[string]bgwHandler{
		"CreateGateway":                    (*bgwStore).createGateway,
		"GetGateway":                       (*bgwStore).getGateway,
		"ListGateways":                     (*bgwStore).listGateways,
		"UpdateGatewayInformation":         (*bgwStore).updateGateway,
		"UpdateGatewaySoftwareNow":         (*bgwStore).updateSoftware,
		"DeleteGateway":                    (*bgwStore).deleteGateway,
		"AssociateGatewayToServer":         (*bgwStore).associate,
		"DisassociateGatewayFromServer":    (*bgwStore).disassociate,
		"PutMaintenanceStartTime":          (*bgwStore).putMaintenance,
		"PutBandwidthRateLimitSchedule":    (*bgwStore).putBandwidth,
		"GetBandwidthRateLimitSchedule":    (*bgwStore).getBandwidth,
		"TestHypervisorConfiguration":      (*bgwStore).testHypervisor,
		"ImportHypervisorConfiguration":    (*bgwStore).importHypervisor,
		"GetHypervisor":                    (*bgwStore).getHypervisor,
		"ListHypervisors":                  (*bgwStore).listHypervisors,
		"UpdateHypervisor":                 (*bgwStore).updateHypervisor,
		"DeleteHypervisor":                 (*bgwStore).deleteHypervisor,
		"PutHypervisorPropertyMappings":    (*bgwStore).putMappings,
		"GetHypervisorPropertyMappings":    (*bgwStore).getMappings,
		"StartVirtualMachinesMetadataSync": (*bgwStore).startSync,
		"GetVirtualMachine":                (*bgwStore).getVM,
		"ListVirtualMachines":              (*bgwStore).listVMs,
		"TagResource":                      (*bgwStore).tagResource,
		"UntagResource":                    (*bgwStore).untagResource,
		"ListTagsForResource":              (*bgwStore).listTags,
	} {
		bgwAPI.handlers[name] = bgwBind(name, h)
	}
}

func (s *Server) handleBackupGatewayJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	return s.budServe(w, r, bgwAPI, backupGatewayOperationByName)
}

type bgwHandler func(st *bgwStore, backups map[string]float64, in map[string]any, now time.Time) (map[string]any, *budErr)

// bgwBind runs a handler under this store's lock. VM reads first take a
// snapshot of Backup's last completed backups (taking and releasing its lock).
func bgwBind(name string, h bgwHandler) budHandler {
	return func(s *Server, in map[string]any) (map[string]any, *budErr) {
		var backups map[string]float64
		if name == "GetVirtualMachine" || name == "ListVirtualMachines" {
			backups = s.backup.lastVMBackups()
		}
		st := s.backupGateway
		st.mu.Lock()
		defer st.mu.Unlock()
		return h(st, backups, in, time.Now().UTC())
	}
}

type bgwGateway struct {
	arn, name, typ, hypervisor string // hypervisor ARN, "" when not associated
	tags                       map[string]string
	maintenance                map[string]any
	bandwidth                  []any
	software                   string
	nextUpdate, lastSeen       time.Time
}

type bgwHypervisor struct {
	arn, name, host, kms, logGroup, state, sync, syncMsg, role string
	tags                                                       map[string]string
	mappings                                                   []any
	lastSync                                                   time.Time
}

type bgwVM struct {
	arn, hypervisor, name, host, path string
}

type bgwStore struct {
	mu          sync.Mutex
	gateways    map[string]*bgwGateway
	gwOrder     []string
	hypervisors map[string]*bgwHypervisor
	hvOrder     []string
	vms         map[string]*bgwVM
	vmOrder     []string
}

func newBackupGatewayStore() *bgwStore {
	return &bgwStore{gateways: map[string]*bgwGateway{}, hypervisors: map[string]*bgwHypervisor{}, vms: map[string]*bgwVM{}}
}

func bgwErr(code, format string, args ...any) *budErr { return opstErrorf(code, format, args...) }

func bgwInvalid(format string, args ...any) *budErr {
	return bgwErr("ValidationException", format, args...)
}

func bgwNotFound(kind, arn string) *budErr {
	return bgwErr("ResourceNotFoundException", "%s %s not found", kind, arn)
}

func bgwRemove(order []string, id string) []string { return opstRemove(order, id) }

func bgwTagsIn(list []any) (map[string]string, *budErr) {
	tags, err := budTagsIn(list, "ValidationException")
	if err != nil {
		return nil, err
	}
	if len(tags) > bgwMaxTags {
		return nil, bgwInvalid("a resource can have at most %d tags", bgwMaxTags)
	}
	return tags, nil
}

func bgwPage(in map[string]any, total int) (int, int, string, *budErr) {
	return budPage(budStr(in, "NextToken"), total, budInt(in, "MaxResults", 100), "ValidationException")
}

func bgwID(arn string) string { return arn[strings.LastIndexByte(arn, '/')+1:] }

// ---- Backup integration ----

// lastVMBackups maps each backup-gateway VM ARN to the creation date of its
// latest completed recovery point.
func (st *backupStore) lastVMBackups() map[string]float64 {
	st.mu.Lock()
	defer st.mu.Unlock()
	out := map[string]float64{}
	for _, rp := range st.recoveryPoints {
		if bkStr(rp.rec, "ResourceType") != "VirtualMachine" || bkStr(rp.rec, "Status") != "COMPLETED" {
			continue
		}
		arn := bkStr(rp.rec, "ResourceArn")
		if at, _ := rp.rec["CreationDate"].(float64); at > out[arn] {
			out[arn] = at
		}
	}
	return out
}

// vmName returns the name of a discovered virtual machine. Backup calls it
// while holding its own lock (Backup before Backup gateway).
func (st *bgwStore) vmName(arn string) (string, bool) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if vm := st.vms[arn]; vm != nil {
		return vm.name, true
	}
	return "", false
}

// ---- gateways ----

func (st *bgwStore) gateway(arn string) (*bgwGateway, *budErr) {
	if g := st.gateways[arn]; g != nil {
		return g, nil
	}
	return nil, bgwNotFound("Gateway", arn)
}

func (st *bgwStore) createGateway(_ map[string]float64, in map[string]any, now time.Time) (map[string]any, *budErr) {
	tags, err := bgwTagsIn(budList(in, "Tags"))
	if err != nil {
		return nil, err
	}
	if len(st.gateways) >= bgwMaxGateways {
		return nil, bgwInvalid("you can have at most %d gateways", bgwMaxGateways)
	}
	// The activation key is spent on activation and never kept.
	arn := bgwARNBase + "gateway/bgw-" + opstHex(17)
	st.gateways[arn] = &bgwGateway{
		arn: arn, name: budStr(in, "GatewayDisplayName"), typ: budStr(in, "GatewayType"), tags: tags,
		maintenance: map[string]any{"HourOfDay": 2, "MinuteOfHour": 0, "DayOfWeek": 0},
		software:    bgwSoftware, nextUpdate: now, lastSeen: now,
	}
	st.gwOrder = append(st.gwOrder, arn)
	return map[string]any{"GatewayArn": arn}, nil
}

func (g *bgwGateway) summary() map[string]any {
	out := map[string]any{"GatewayArn": g.arn, "GatewayDisplayName": g.name, "GatewayType": g.typ, "LastSeenTime": budEpoch(g.lastSeen)}
	if g.hypervisor != "" {
		out["HypervisorId"] = bgwID(g.hypervisor)
	}
	return out
}

func (st *bgwStore) getGateway(_ map[string]float64, in map[string]any, _ time.Time) (map[string]any, *budErr) {
	g, err := st.gateway(budStr(in, "GatewayArn"))
	if err != nil {
		return nil, err
	}
	out := g.summary()
	out["MaintenanceStartTime"] = budCloneMap(g.maintenance)
	out["SoftwareVersion"] = g.software
	if !g.nextUpdate.IsZero() {
		out["NextUpdateAvailabilityTime"] = budEpoch(g.nextUpdate)
	}
	return map[string]any{"Gateway": out}, nil
}

func (st *bgwStore) listGateways(_ map[string]float64, in map[string]any, _ time.Time) (map[string]any, *budErr) {
	start, end, next, err := bgwPage(in, len(st.gwOrder))
	if err != nil {
		return nil, err
	}
	list := make([]any, 0, end-start)
	for _, arn := range st.gwOrder[start:end] {
		list = append(list, st.gateways[arn].summary())
	}
	return map[string]any{"Gateways": list, "NextToken": next}, nil
}

func (st *bgwStore) updateGateway(_ map[string]float64, in map[string]any, _ time.Time) (map[string]any, *budErr) {
	g, err := st.gateway(budStr(in, "GatewayArn"))
	if err != nil {
		return nil, err
	}
	if v := budStr(in, "GatewayDisplayName"); v != "" {
		g.name = v
	}
	return map[string]any{"GatewayArn": g.arn}, nil
}

func (st *bgwStore) updateSoftware(_ map[string]float64, in map[string]any, now time.Time) (map[string]any, *budErr) {
	g, err := st.gateway(budStr(in, "GatewayArn"))
	if err != nil {
		return nil, err
	}
	if !g.nextUpdate.IsZero() && !now.Before(g.nextUpdate) {
		g.software, g.nextUpdate, g.lastSeen = bgwNextSoftware, time.Time{}, now
	}
	return map[string]any{"GatewayArn": g.arn}, nil
}

func (st *bgwStore) deleteGateway(_ map[string]float64, in map[string]any, _ time.Time) (map[string]any, *budErr) {
	g, err := st.gateway(budStr(in, "GatewayArn"))
	if err != nil {
		return nil, err
	}
	delete(st.gateways, g.arn)
	st.gwOrder = bgwRemove(st.gwOrder, g.arn)
	return map[string]any{"GatewayArn": g.arn}, nil
}

func (st *bgwStore) associate(_ map[string]float64, in map[string]any, _ time.Time) (map[string]any, *budErr) {
	g, err := st.gateway(budStr(in, "GatewayArn"))
	if err != nil {
		return nil, err
	}
	h, err := st.hypervisor(budStr(in, "ServerArn"))
	if err != nil {
		return nil, err
	}
	if g.hypervisor != "" {
		return nil, bgwErr("ConflictException", "gateway %s is already associated with %s", g.arn, g.hypervisor)
	}
	g.hypervisor = h.arn
	return map[string]any{"GatewayArn": g.arn}, nil
}

func (st *bgwStore) disassociate(_ map[string]float64, in map[string]any, _ time.Time) (map[string]any, *budErr) {
	g, err := st.gateway(budStr(in, "GatewayArn"))
	if err != nil {
		return nil, err
	}
	if g.hypervisor == "" {
		return nil, bgwErr("ConflictException", "gateway %s is not associated with a hypervisor", g.arn)
	}
	g.hypervisor = ""
	return map[string]any{"GatewayArn": g.arn}, nil
}

func (st *bgwStore) putMaintenance(_ map[string]float64, in map[string]any, _ time.Time) (map[string]any, *budErr) {
	g, err := st.gateway(budStr(in, "GatewayArn"))
	if err != nil {
		return nil, err
	}
	_, week := in["DayOfWeek"]
	_, month := in["DayOfMonth"]
	if week == month {
		return nil, bgwInvalid("set exactly one of DayOfWeek (weekly) or DayOfMonth (monthly)")
	}
	m := map[string]any{"HourOfDay": in["HourOfDay"], "MinuteOfHour": in["MinuteOfHour"]}
	if week {
		m["DayOfWeek"] = in["DayOfWeek"]
	} else {
		m["DayOfMonth"] = in["DayOfMonth"]
	}
	g.maintenance = m
	return map[string]any{"GatewayArn": g.arn}, nil
}

// bgwCheckIntervals refuses empty or inverted intervals, repeated days and
// intervals that overlap on a day.
func bgwCheckIntervals(list []any) *budErr {
	if len(list) > bgwMaxIntervals {
		return bgwInvalid("at most %d bandwidth rate limit intervals", bgwMaxIntervals)
	}
	type span struct{ from, to int }
	byDay := map[int][]span{}
	for _, item := range list {
		m := item.(map[string]any)
		from := budInt(m, "StartHourOfDay", 0)*60 + budInt(m, "StartMinuteOfHour", 0)
		to := budInt(m, "EndHourOfDay", 0)*60 + budInt(m, "EndMinuteOfHour", 0)
		if to <= from {
			return bgwInvalid("an interval must end after it starts")
		}
		days := budList(m, "DaysOfWeek")
		if len(days) == 0 || len(days) > 7 {
			return bgwInvalid("DaysOfWeek must list 1 to 7 days")
		}
		seen := map[int]bool{}
		for _, d := range days {
			day := int(d.(float64))
			if seen[day] {
				return bgwInvalid("DaysOfWeek repeats day %d", day)
			}
			seen[day] = true
			for _, s := range byDay[day] {
				if from < s.to && s.from < to {
					return bgwInvalid("intervals overlap on day %d", day)
				}
			}
			byDay[day] = append(byDay[day], span{from, to})
		}
	}
	return nil
}

func (st *bgwStore) putBandwidth(_ map[string]float64, in map[string]any, _ time.Time) (map[string]any, *budErr) {
	g, err := st.gateway(budStr(in, "GatewayArn"))
	if err != nil {
		return nil, err
	}
	list := budList(in, "BandwidthRateLimitIntervals")
	if err := bgwCheckIntervals(list); err != nil {
		return nil, err
	}
	g.bandwidth = budClone(list).([]any)
	return map[string]any{"GatewayArn": g.arn}, nil
}

func (st *bgwStore) getBandwidth(_ map[string]float64, in map[string]any, _ time.Time) (map[string]any, *budErr) {
	g, err := st.gateway(budStr(in, "GatewayArn"))
	if err != nil {
		return nil, err
	}
	list, _ := budClone(g.bandwidth).([]any)
	if list == nil {
		list = []any{}
	}
	return map[string]any{"GatewayArn": g.arn, "BandwidthRateLimitIntervals": list}, nil
}

// ---- hypervisors ----

var bgwNameUnsafe = regexp.MustCompile(`[^a-zA-Z0-9-]+`)

func (st *bgwStore) hypervisor(arn string) (*bgwHypervisor, *budErr) {
	if h := st.hypervisors[arn]; h != nil {
		return h, nil
	}
	return nil, bgwNotFound("Hypervisor", arn)
}

func bgwCheckCredentials(in map[string]any) *budErr {
	if budStr(in, "Password") != "" && budStr(in, "Username") == "" {
		return bgwInvalid("a Password needs a Username")
	}
	return nil
}

func (st *bgwStore) hostTaken(host, except string) bool {
	for _, h := range st.hypervisors {
		if h.arn != except && strings.EqualFold(h.host, host) {
			return true
		}
	}
	return false
}

func (st *bgwStore) testHypervisor(_ map[string]float64, in map[string]any, _ time.Time) (map[string]any, *budErr) {
	if _, err := st.gateway(budStr(in, "GatewayArn")); err != nil {
		return nil, err
	}
	if err := bgwCheckCredentials(in); err != nil {
		return nil, err
	}
	return map[string]any{}, nil
}

func (st *bgwStore) importHypervisor(_ map[string]float64, in map[string]any, _ time.Time) (map[string]any, *budErr) {
	if err := bgwCheckCredentials(in); err != nil {
		return nil, err
	}
	tags, err := bgwTagsIn(budList(in, "Tags"))
	if err != nil {
		return nil, err
	}
	host := budStr(in, "Host")
	if st.hostTaken(host, "") {
		return nil, bgwErr("ConflictException", "a hypervisor for host %s is already imported", host)
	}
	if len(st.hypervisors) >= bgwMaxHypervisors {
		return nil, bgwInvalid("you can import at most %d hypervisors", bgwMaxHypervisors)
	}
	arn := bgwARNBase + "hypervisor/hype-" + opstHex(17)
	// Username and Password only reach the (absent) hypervisor; neither is kept.
	st.hypervisors[arn] = &bgwHypervisor{
		arn: arn, name: budStr(in, "Name"), host: host, kms: budStr(in, "KmsKeyArn"), state: "PENDING", sync: "CREATED", tags: tags,
	}
	st.hvOrder = append(st.hvOrder, arn)
	return map[string]any{"HypervisorArn": arn}, nil
}

// settle brings a hypervisor ONLINE and advances its metadata sync one step;
// a successful sync replaces the hypervisor's VMs with what it reports.
func (st *bgwStore) settle(h *bgwHypervisor, now time.Time) {
	if h.state == "PENDING" {
		h.state = "ONLINE"
		return
	}
	switch h.sync {
	case "CREATED":
		h.sync = "RUNNING"
	case "RUNNING":
		h.sync, h.lastSync, h.syncMsg = "SUCCEEDED", now, fmt.Sprintf("Discovered %d virtual machines.", bgwVMsPerHost)
		st.discover(h)
	}
}

func (st *bgwStore) dropVMs(hypervisor string) {
	for _, arn := range append([]string(nil), st.vmOrder...) {
		if st.vms[arn].hypervisor == hypervisor {
			delete(st.vms, arn)
			st.vmOrder = bgwRemove(st.vmOrder, arn)
		}
	}
}

func (st *bgwStore) discover(h *bgwHypervisor) {
	st.dropVMs(h.arn)
	base := strings.Trim(bgwNameUnsafe.ReplaceAllString(strings.SplitN(h.host, ".", 2)[0], "-"), "-")
	if base == "" || len(base) > 80 {
		base = "vm"
	}
	for i := 1; i <= bgwVMsPerHost; i++ {
		sum := sha256.Sum256([]byte(fmt.Sprintf("%s/%d", h.arn, i)))
		name := fmt.Sprintf("%s-vm-%d", base, i)
		vm := &bgwVM{arn: bgwARNBase + "vm/vm-" + hex.EncodeToString(sum[:])[:17], hypervisor: h.arn, name: name,
			host: h.host, path: "/Datacenter/vm/" + name}
		st.vms[vm.arn] = vm
		st.vmOrder = append(st.vmOrder, vm.arn)
	}
}

func (h *bgwHypervisor) summary() map[string]any {
	out := map[string]any{"HypervisorArn": h.arn, "Name": h.name, "Host": h.host, "State": h.state}
	if h.kms != "" {
		out["KmsKeyArn"] = h.kms
	}
	return out
}

func (st *bgwStore) getHypervisor(_ map[string]float64, in map[string]any, now time.Time) (map[string]any, *budErr) {
	h, err := st.hypervisor(budStr(in, "HypervisorArn"))
	if err != nil {
		return nil, err
	}
	st.settle(h, now)
	out := h.summary()
	out["LatestMetadataSyncStatus"] = h.sync
	if h.logGroup != "" {
		out["LogGroupArn"] = h.logGroup
	}
	if h.syncMsg != "" {
		out["LatestMetadataSyncStatusMessage"] = h.syncMsg
	}
	if !h.lastSync.IsZero() {
		out["LastSuccessfulMetadataSyncTime"] = budEpoch(h.lastSync)
	}
	return map[string]any{"Hypervisor": out}, nil
}

func (st *bgwStore) listHypervisors(_ map[string]float64, in map[string]any, _ time.Time) (map[string]any, *budErr) {
	start, end, next, err := bgwPage(in, len(st.hvOrder))
	if err != nil {
		return nil, err
	}
	list := make([]any, 0, end-start)
	for _, arn := range st.hvOrder[start:end] {
		list = append(list, st.hypervisors[arn].summary())
	}
	return map[string]any{"Hypervisors": list, "NextToken": next}, nil
}

func (st *bgwStore) updateHypervisor(_ map[string]float64, in map[string]any, _ time.Time) (map[string]any, *budErr) {
	h, err := st.hypervisor(budStr(in, "HypervisorArn"))
	if err != nil {
		return nil, err
	}
	if err := bgwCheckCredentials(in); err != nil {
		return nil, err
	}
	if host := budStr(in, "Host"); host != "" && !strings.EqualFold(host, h.host) {
		if st.hostTaken(host, h.arn) {
			return nil, bgwErr("ConflictException", "a hypervisor for host %s is already imported", host)
		}
		h.host, h.state, h.sync = host, "PENDING", "CREATED" // reconnect and re-sync
		st.dropVMs(h.arn)
	}
	if v := budStr(in, "Name"); v != "" {
		h.name = v
	}
	if v, ok := in["LogGroupArn"].(string); ok {
		h.logGroup = v
	}
	return map[string]any{"HypervisorArn": h.arn}, nil
}

func (st *bgwStore) deleteHypervisor(_ map[string]float64, in map[string]any, _ time.Time) (map[string]any, *budErr) {
	h, err := st.hypervisor(budStr(in, "HypervisorArn"))
	if err != nil {
		return nil, err
	}
	for _, g := range st.gateways {
		if g.hypervisor == h.arn {
			return nil, bgwErr("ConflictException", "gateway %s is associated with hypervisor %s; disassociate it first", g.arn, h.arn)
		}
	}
	st.dropVMs(h.arn)
	delete(st.hypervisors, h.arn)
	st.hvOrder = bgwRemove(st.hvOrder, h.arn)
	return map[string]any{"HypervisorArn": h.arn}, nil
}

func (st *bgwStore) putMappings(_ map[string]float64, in map[string]any, _ time.Time) (map[string]any, *budErr) {
	h, err := st.hypervisor(budStr(in, "HypervisorArn"))
	if err != nil {
		return nil, err
	}
	list := budList(in, "VmwareToAwsTagMappings")
	if len(list) > bgwMaxMappings {
		return nil, bgwInvalid("at most %d tag mappings", bgwMaxMappings)
	}
	h.mappings, h.role = budClone(list).([]any), budStr(in, "IamRoleArn")
	return map[string]any{"HypervisorArn": h.arn}, nil
}

func (st *bgwStore) getMappings(_ map[string]float64, in map[string]any, _ time.Time) (map[string]any, *budErr) {
	h, err := st.hypervisor(budStr(in, "HypervisorArn"))
	if err != nil {
		return nil, err
	}
	list, _ := budClone(h.mappings).([]any)
	if list == nil {
		list = []any{}
	}
	out := map[string]any{"HypervisorArn": h.arn, "VmwareToAwsTagMappings": list}
	if h.role != "" {
		out["IamRoleArn"] = h.role
	}
	return out, nil
}

func (st *bgwStore) startSync(_ map[string]float64, in map[string]any, _ time.Time) (map[string]any, *budErr) {
	h, err := st.hypervisor(budStr(in, "HypervisorArn"))
	if err != nil {
		return nil, err
	}
	h.sync, h.syncMsg = "CREATED", ""
	return map[string]any{"HypervisorArn": h.arn}, nil
}

// ---- virtual machines ----

// vmTags maps the hypervisor's VMware tag mappings onto a VM: every VM
// carries the mapped VMware tags.
func (st *bgwStore) vmTags(vm *bgwVM) []any {
	out := []any{}
	if h := st.hypervisors[vm.hypervisor]; h != nil {
		for _, m := range h.mappings {
			mm := m.(map[string]any)
			out = append(out, map[string]any{"VmwareCategory": mm["VmwareCategory"], "VmwareTagName": mm["VmwareTagName"], "VmwareTagDescription": ""})
		}
	}
	return out
}

func (vm *bgwVM) out(backups map[string]float64) map[string]any {
	out := map[string]any{"ResourceArn": vm.arn, "Name": vm.name, "HostName": vm.host, "HypervisorId": bgwID(vm.hypervisor), "Path": vm.path}
	if at := backups[vm.arn]; at > 0 {
		out["LastBackupDate"] = at
	}
	return out
}

func (st *bgwStore) getVM(backups map[string]float64, in map[string]any, _ time.Time) (map[string]any, *budErr) {
	vm := st.vms[budStr(in, "ResourceArn")]
	if vm == nil {
		return nil, bgwNotFound("Virtual machine", budStr(in, "ResourceArn"))
	}
	out := vm.out(backups)
	out["VmwareTags"] = st.vmTags(vm)
	return map[string]any{"VirtualMachine": out}, nil
}

func (st *bgwStore) listVMs(backups map[string]float64, in map[string]any, _ time.Time) (map[string]any, *budErr) {
	hv := budStr(in, "HypervisorArn")
	if hv != "" {
		if _, err := st.hypervisor(hv); err != nil {
			return nil, err
		}
	}
	var match []*bgwVM
	for _, arn := range st.vmOrder {
		if vm := st.vms[arn]; hv == "" || vm.hypervisor == hv {
			match = append(match, vm)
		}
	}
	start, end, next, err := bgwPage(in, len(match))
	if err != nil {
		return nil, err
	}
	list := make([]any, 0, end-start)
	for _, vm := range match[start:end] {
		list = append(list, vm.out(backups))
	}
	return map[string]any{"VirtualMachines": list, "NextToken": next}, nil
}

// ---- tags ----

func (st *bgwStore) tagsOf(arn string) (*map[string]string, *budErr) {
	if g := st.gateways[arn]; g != nil {
		return &g.tags, nil
	}
	if h := st.hypervisors[arn]; h != nil {
		return &h.tags, nil
	}
	return nil, bgwNotFound("Resource", arn)
}

func (st *bgwStore) tagResource(_ map[string]float64, in map[string]any, _ time.Time) (map[string]any, *budErr) {
	add, err := bgwTagsIn(budList(in, "Tags"))
	if err != nil {
		return nil, err
	}
	tags, err := st.tagsOf(budStr(in, "ResourceARN"))
	if err != nil {
		return nil, err
	}
	merged, err := budMergeTags(*tags, add, bgwMaxTags, "ValidationException")
	if err != nil {
		return nil, err
	}
	*tags = merged
	return map[string]any{"ResourceARN": budStr(in, "ResourceARN")}, nil
}

func (st *bgwStore) untagResource(_ map[string]float64, in map[string]any, _ time.Time) (map[string]any, *budErr) {
	tags, err := st.tagsOf(budStr(in, "ResourceARN"))
	if err != nil {
		return nil, err
	}
	*tags = budDropTags(*tags, budStrings(in, "TagKeys"))
	return map[string]any{"ResourceARN": budStr(in, "ResourceARN")}, nil
}

func (st *bgwStore) listTags(_ map[string]float64, in map[string]any, _ time.Time) (map[string]any, *budErr) {
	tags, err := st.tagsOf(budStr(in, "ResourceArn"))
	if err != nil {
		return nil, err
	}
	return map[string]any{"ResourceArn": budStr(in, "ResourceArn"), "Tags": budTagsOut(*tags)}, nil
}
