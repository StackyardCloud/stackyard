package server

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

// AWS Snow Device Management: REST JSON, signed snow-device-management.
// Managed devices are the Snowball jobs that are not cancelled (read from the
// Snowball store, whose lock is released before this store's is taken).
// Tasks run reboot or unlock on their target devices; each execution goes
// QUEUED -> IN_PROGRESS -> SUCCEEDED one step per read, a successful unlock
// unlocks the device, and the task completes when every execution is done.

const (
	sdmAccount  = "123456789012"
	sdmRegion   = "us-east-1"
	sdmARNBase  = "arn:aws:snow-device-management:" + sdmRegion + ":" + sdmAccount + ":"
	sdmMaxTasks = 1000
	// ponytail: AWS caps targets per task well below this; the cap only
	// bounds per-task work.
	sdmMaxTargets   = 100
	sdmMaxDevices   = 1000 // devices tracked (and read from Snowball)
	sdmMaxTags      = 50
	sdmMaxInstances = 100 // instanceIds per DescribeDeviceEc2Instances
)

var sdmTokenRE = regexp.MustCompile(`^(\d+)$`)

var sdmAPI = &opstRESTAPI{
	signing: "snow-device-management", hostPrefix: "snow-device-management.", respondError: sdmWriteError,
	model:  awsmodels.SnowDeviceManagementModel,
	routes: opstCompileRoutes(snowDeviceManagementOperations),
	dispatch: func(s *Server, op string, in map[string]any) (map[string]any, *budErr) {
		return s.sdmDispatch(op, in)
	},
}

// sdmWriteError writes Snow Device Management's error shape (a lowercase
// message member).
func sdmWriteError(w http.ResponseWriter, e *budErr) {
	body, _ := json.Marshal(map[string]string{"message": e.msg})
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Amzn-ErrorType", e.code)
	w.WriteHeader(e.status)
	_, _ = w.Write(body)
}

func (s *Server) handleSnowDeviceManagementRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	return s.opstServe(w, r, sdmAPI)
}

type sdmDevice struct {
	id, arn, jobID, deviceType string
}

type sdmDeviceState struct {
	state               string // LOCKED, UNLOCKING, UNLOCKED
	tags                map[string]string
	updated, reachedOut time.Time
}

type sdmExecution struct {
	id, device, state string
	started, updated  time.Time
}

type sdmTask struct {
	id, arn, command, description, state, clientToken string
	targets                                           []string // device ids, in request order
	executions                                        map[string]*sdmExecution
	tags                                              map[string]string
	created, updated, completed                       time.Time
}

type sdmStore struct {
	mu        sync.Mutex
	devices   map[string]*sdmDeviceState
	tasks     map[string]*sdmTask
	taskOrder []string
}

func newSnowDeviceManagementStore() *sdmStore {
	return &sdmStore{devices: map[string]*sdmDeviceState{}, tasks: map[string]*sdmTask{}}
}

// sdmManagedDevices snapshots the Snowball jobs that are managed devices.
func (st *snowballStore) sdmManagedDevices() []sdmDevice {
	st.mu.Lock()
	defer st.mu.Unlock()
	ids := make([]string, 0, len(st.jobs))
	for id, job := range st.jobs {
		if state, _ := job["JobState"].(string); state != "Cancelled" {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	if len(ids) > sdmMaxDevices {
		ids = ids[:sdmMaxDevices]
	}
	out := make([]sdmDevice, 0, len(ids))
	for _, id := range ids {
		typ, _ := st.jobs[id]["SnowballType"].(string)
		sum := sha256.Sum256([]byte(id))
		dev := "smd-" + hex.EncodeToString(sum[:])[:17]
		out = append(out, sdmDevice{id: dev, arn: sdmARNBase + "managed-device/" + dev, jobID: id, deviceType: sdmDeviceType(typ)})
	}
	return out
}

func sdmDeviceType(snowballType string) string {
	switch {
	case strings.HasPrefix(snowballType, "SNC1"):
		return "SNOWCONE"
	case strings.HasPrefix(snowballType, "EDGE"), strings.HasPrefix(snowballType, "V3_"), strings.HasPrefix(snowballType, "RACK_"):
		return "SNOWBALL_EDGE"
	}
	return "SNOWBALL"
}

func sdmErr(code, format string, args ...any) *budErr { return opstErrorf(code, format, args...) }

func sdmNotFound(kind, id string) *budErr {
	return sdmErr("ResourceNotFoundException", "%s %s not found", kind, id)
}

func (s *Server) sdmDispatch(op string, in map[string]any) (map[string]any, *budErr) {
	h := sdmHandlers[op]
	if h == nil {
		return nil, sdmErr("UnknownOperationException", "unknown operation %s", op)
	}
	devices := s.snowball.sdmManagedDevices() // Snowball's lock is released here
	st := s.snowDeviceManagement
	st.mu.Lock()
	defer st.mu.Unlock()
	return h(st, devices, in, time.Now().UTC())
}

type sdmHandler func(st *sdmStore, devices []sdmDevice, in map[string]any, now time.Time) (map[string]any, *budErr)

var sdmHandlers map[string]sdmHandler

func init() {
	sdmHandlers = map[string]sdmHandler{
		"ListDevices":                (*sdmStore).listDevices,
		"DescribeDevice":             (*sdmStore).describeDevice,
		"ListDeviceResources":        (*sdmStore).listResources,
		"DescribeDeviceEc2Instances": (*sdmStore).describeInstances,
		"CreateTask":                 (*sdmStore).createTask,
		"DescribeTask":               (*sdmStore).describeTask,
		"ListTasks":                  (*sdmStore).listTasks,
		"CancelTask":                 (*sdmStore).cancelTask,
		"DescribeExecution":          (*sdmStore).describeExecution,
		"ListExecutions":             (*sdmStore).listExecutions,
		"TagResource":                (*sdmStore).tagResource,
		"UntagResource":              (*sdmStore).untagResource,
		"ListTagsForResource":        (*sdmStore).listTags,
	}
}

func sdmPage(in map[string]any, total int) (int, int, string, *budErr) {
	return opstPage(budStr(in, "nextToken"), total, budInt(in, "maxResults", 100), sdmTokenRE, "%d")
}

func sdmFind(devices []sdmDevice, idOrARN string) *sdmDevice {
	id := idOrARN
	if i := strings.LastIndexByte(id, '/'); i >= 0 {
		id = id[i+1:]
	}
	for i := range devices {
		if devices[i].id == id {
			return &devices[i]
		}
	}
	return nil
}

// state returns a device's tracked state, creating it (LOCKED) on first use.
func (st *sdmStore) state(id string, now time.Time) (*sdmDeviceState, *budErr) {
	if d := st.devices[id]; d != nil {
		return d, nil
	}
	if len(st.devices) >= sdmMaxDevices {
		return nil, sdmErr("ServiceQuotaExceededException", "at most %d devices can be managed", sdmMaxDevices)
	}
	d := &sdmDeviceState{state: "LOCKED", tags: map[string]string{}, updated: now, reachedOut: now}
	st.devices[id] = d
	return d, nil
}

// peek returns a device's state without tracking it.
func (st *sdmStore) peek(id string, now time.Time) *sdmDeviceState {
	if d := st.devices[id]; d != nil {
		return d
	}
	return &sdmDeviceState{state: "LOCKED", tags: map[string]string{}, updated: now, reachedOut: now}
}

func sdmTagsOut(tags map[string]string) map[string]any {
	out := make(map[string]any, len(tags))
	for k, v := range tags {
		out[k] = v
	}
	return out
}

func sdmTagsIn(in map[string]any) (map[string]string, *budErr) {
	out := map[string]string{}
	for k, v := range budMap(in, "tags") {
		if strings.HasPrefix(strings.ToLower(k), "aws:") {
			return nil, sdmErr("ValidationException", "tag keys starting with aws: are reserved")
		}
		out[k], _ = v.(string)
	}
	if len(out) > sdmMaxTags {
		return nil, sdmErr("ValidationException", "a resource can have at most %d tags", sdmMaxTags)
	}
	return out, nil
}

// ---- devices ----

func (st *sdmStore) listDevices(devices []sdmDevice, in map[string]any, now time.Time) (map[string]any, *budErr) {
	job := budStr(in, "jobId")
	var match []sdmDevice
	for _, d := range devices {
		if job == "" || d.jobID == job {
			match = append(match, d)
		}
	}
	start, end, next, err := sdmPage(in, len(match))
	if err != nil {
		return nil, err
	}
	list := make([]any, 0, end-start)
	for _, d := range match[start:end] {
		list = append(list, map[string]any{"managedDeviceId": d.id, "managedDeviceArn": d.arn, "associatedWithJob": d.jobID,
			"tags": sdmTagsOut(st.peek(d.id, now).tags)})
	}
	return map[string]any{"devices": list, "nextToken": next}, nil
}

func (st *sdmStore) device(devices []sdmDevice, in map[string]any) (*sdmDevice, *budErr) {
	d := sdmFind(devices, budStr(in, "managedDeviceId"))
	if d == nil {
		return nil, sdmNotFound("Managed device", budStr(in, "managedDeviceId"))
	}
	return d, nil
}

func (st *sdmStore) describeDevice(devices []sdmDevice, in map[string]any, now time.Time) (map[string]any, *budErr) {
	d, err := st.device(devices, in)
	if err != nil {
		return nil, err
	}
	ds := st.peek(d.id, now)
	const tb = int64(1) << 40
	used := int64(len(d.id)) << 30 // a few GiB, stable per device
	return map[string]any{
		"managedDeviceId": d.id, "managedDeviceArn": d.arn, "associatedWithJob": d.jobID, "deviceType": d.deviceType,
		"deviceState": ds.state, "lastUpdatedAt": budEpoch(ds.updated), "lastReachedOutAt": budEpoch(ds.reachedOut), "tags": sdmTagsOut(ds.tags),
		"deviceCapacities": []any{
			map[string]any{"name": "HDD Storage", "unit": "Byte", "total": 80 * tb / 1000, "used": used, "available": 80*tb/1000 - used},
			map[string]any{"name": "vCPU", "unit": "Number", "total": 104, "used": 2, "available": 102},
			map[string]any{"name": "Memory", "unit": "Byte", "total": int64(416) << 30, "used": int64(8) << 30, "available": int64(408) << 30},
		},
		"physicalNetworkInterfaces": []any{
			map[string]any{"physicalNetworkInterfaceId": "s.ni-" + d.id[4:], "physicalConnectorType": "RJ45", "ipAddressAssignment": "DHCP",
				"ipAddress": "192.168.1.10", "netmask": "255.255.255.0", "defaultGateway": "192.168.1.1", "macAddress": sdmMAC(d.id, 1)},
			map[string]any{"physicalNetworkInterfaceId": "s.ni-" + d.id[5:] + "a", "physicalConnectorType": "SFP_PLUS", "ipAddressAssignment": "STATIC",
				"ipAddress": "10.0.0.10", "netmask": "255.255.255.0", "defaultGateway": "10.0.0.1", "macAddress": sdmMAC(d.id, 2)},
		},
		"software": map[string]any{"installState": "INSTALLED", "installedVersion": "122"},
	}, nil
}

func sdmMAC(id string, n int) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s/%d", id, n)))
	return fmt.Sprintf("02:%02x:%02x:%02x:%02x:%02x", sum[0], sum[1], sum[2], sum[3], sum[4])
}

// sdmInstanceID is the one EC2 instance each device runs (its Snowball job's
// compute); the id is stable per device.
func sdmInstanceID(device string) string {
	sum := sha256.Sum256([]byte(device + "/ec2"))
	return "s.i-" + hex.EncodeToString(sum[:])[:17]
}

func (st *sdmStore) listResources(devices []sdmDevice, in map[string]any, _ time.Time) (map[string]any, *budErr) {
	d, err := st.device(devices, in)
	if err != nil {
		return nil, err
	}
	all := []map[string]any{{"resourceType": "AWS::EC2::Instance", "id": sdmInstanceID(d.id),
		"arn": "arn:aws:ec2:" + sdmRegion + ":" + sdmAccount + ":instance/" + sdmInstanceID(d.id)}}
	typ := budStr(in, "type")
	var match []any
	for _, r := range all {
		if typ == "" || r["resourceType"] == typ {
			match = append(match, r)
		}
	}
	start, end, next, err := sdmPage(in, len(match))
	if err != nil {
		return nil, err
	}
	return map[string]any{"resources": append([]any{}, match[start:end]...), "nextToken": next}, nil
}

func (st *sdmStore) describeInstances(devices []sdmDevice, in map[string]any, now time.Time) (map[string]any, *budErr) {
	d, err := st.device(devices, in)
	if err != nil {
		return nil, err
	}
	ids := budStrings(in, "instanceIds")
	if len(ids) > sdmMaxInstances {
		return nil, sdmErr("ValidationException", "at most %d instanceIds", sdmMaxInstances)
	}
	ds := st.peek(d.id, now)
	out := []any{}
	for _, id := range ids {
		if id != sdmInstanceID(d.id) {
			return nil, sdmNotFound("Instance", id)
		}
		out = append(out, map[string]any{"lastUpdatedAt": budEpoch(ds.updated), "instance": map[string]any{
			"instanceId": id, "instanceType": "sbe-c.large", "imageId": "s.ami-0123456789abcdef0", "amiLaunchIndex": 0,
			"cpuOptions": map[string]any{"coreCount": 1, "threadsPerCore": 2}, "createdAt": budEpoch(ds.reachedOut),
			"updatedAt": budEpoch(ds.updated), "privateIpAddress": "34.223.14.193", "rootDeviceName": "/dev/sda",
			"state":          map[string]any{"code": 16, "name": "RUNNING"},
			"securityGroups": []any{map[string]any{"groupId": "s.sg-0123456789abcdef0", "groupName": "default"}},
			"blockDeviceMappings": []any{map[string]any{"deviceName": "/dev/sda", "ebs": map[string]any{
				"attachTime": budEpoch(ds.reachedOut), "deleteOnTermination": true, "status": "ATTACHED", "volumeId": "s.vol-0123456789abcdef0"}}},
		}})
	}
	return map[string]any{"instances": out}, nil
}
