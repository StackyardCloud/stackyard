package server

import (
	"hash/fnv"
	"slices"
	"strings"
	"time"
)

// SageMaker Edge Manager control plane: device fleets, the devices registered
// to them and edge deployment plans. SageMaker Edge (sagemakeredge.go) reads
// the deployments a device should run and records its heartbeats and
// deployment results on the same structs.

const (
	smeMaxFleets          = 100
	smeMaxDevices         = 1000
	smeMaxDevicesPerCall  = 100
	smeMaxPlans           = 100
	smeMaxStages          = 10
	smeMaxModelConfigs    = 10
	smeMaxSelectedDevices = 100
)

type smeOutputConfig struct {
	S3OutputLocation       string `json:"S3OutputLocation"`
	KmsKeyId               string `json:"KmsKeyId,omitempty"`
	PresetDeploymentType   string `json:"PresetDeploymentType,omitempty"`
	PresetDeploymentConfig string `json:"PresetDeploymentConfig,omitempty"`
}

type smeFleet struct {
	Name, ARN, RoleArn, Description, IotRoleAlias string
	Output                                        smeOutputConfig
	Created, Modified                             time.Time
}

type smeEdgeModel struct {
	ModelName        string   `json:"ModelName"`
	ModelVersion     string   `json:"ModelVersion"`
	LatestSampleTime *float64 `json:"LatestSampleTime,omitempty"`
	LatestInference  *float64 `json:"LatestInference,omitempty"`
}

type smeDevice struct {
	Name, Fleet, ARN, Description, IotThingName, AgentVersion string
	Registered, Heartbeat                                     time.Time
	Models                                                    []smeEdgeModel
}

type smeSelection struct {
	DeviceSubsetType   string   `json:"DeviceSubsetType"`
	Percentage         *int     `json:"Percentage,omitempty"`
	DeviceNames        []string `json:"DeviceNames,omitempty"`
	DeviceNameContains string   `json:"DeviceNameContains,omitempty"`
}

// selects reports whether the stage targets device.
func (sel smeSelection) selects(device string) bool {
	switch sel.DeviceSubsetType {
	case "SELECTION":
		return slices.Contains(sel.DeviceNames, device)
	case "NAMECONTAINS":
		return strings.Contains(device, sel.DeviceNameContains)
	}
	// ponytail: PERCENTAGE picks a stable hash-based share of the fleet, not an
	// exact count; sort the fleet and take the first N if exact shares matter.
	h := fnv.New32a()
	_, _ = h.Write([]byte(device))
	return sel.Percentage != nil && int(h.Sum32()%100) < *sel.Percentage
}

type smeDeploymentConfig struct {
	FailureHandlingPolicy string `json:"FailureHandlingPolicy"`
}

type smeStageDef struct {
	StageName             string               `json:"StageName"`
	DeviceSelectionConfig smeSelection         `json:"DeviceSelectionConfig"`
	DeploymentConfig      *smeDeploymentConfig `json:"DeploymentConfig,omitempty"`
}

type smeStage struct {
	smeStageDef
	status  string
	started time.Time
}

type smeModelConfig struct {
	ModelHandle          string `json:"ModelHandle"`
	EdgePackagingJobName string `json:"EdgePackagingJobName"`
}

type smePlan struct {
	Name, ARN, Fleet  string
	ModelConfigs      []smeModelConfig
	Stages            []*smeStage
	Created, Modified time.Time
	// results maps a device to the SUCCESS or FAIL it reported for this plan.
	results map[string]string
}

// ---- device fleets ----

type smeFleetReq struct {
	DeviceFleetName    string
	RoleArn            string
	Description        string
	OutputConfig       smeOutputConfig
	EnableIotRoleAlias *bool
	Tags               []sagemakerTag
}

func (f *smeFleet) apply(req smeFleetReq) {
	f.RoleArn, f.Description, f.Output = req.RoleArn, req.Description, req.OutputConfig
	if req.EnableIotRoleAlias != nil {
		f.IotRoleAlias = ""
		if *req.EnableIotRoleAlias {
			f.IotRoleAlias = "arn:aws:iot:us-east-1:" + sagemakerAccountID + ":rolealias/SageMakerEdge-" + f.Name
		}
	}
}

func smeCreateDeviceFleet(st *sagemakerStore, req smeFleetReq, now time.Time) (any, *sagemakerResourceError) {
	if st.deviceFleets[req.DeviceFleetName] != nil {
		return nil, sagemakerResErr(sagemakerErrInUse, "Device fleet %s already exists.", req.DeviceFleetName)
	}
	if len(st.deviceFleets) >= smeMaxFleets {
		return nil, sagemakerResErr(sagemakerErrLimit, "Stackyard keeps at most %d device fleets.", smeMaxFleets)
	}
	f := &smeFleet{Name: req.DeviceFleetName, ARN: sagemakerResourceARN("device-fleet", req.DeviceFleetName), Created: now, Modified: now}
	f.apply(req)
	st.deviceFleets[f.Name] = f
	st.setTagsLocked(f.ARN, req.Tags)
	return nil, nil
}

func (s *sagemakerStore) smeFleetLocked(name string) (*smeFleet, *sagemakerResourceError) {
	if f := s.deviceFleets[name]; f != nil {
		return f, nil
	}
	return nil, sagemakerResErr(sagemakerErrNotFound, "Device fleet %s does not exist.", name)
}

type smeFleetNameReq struct {
	DeviceFleetName string
}

func smeDescribeDeviceFleet(st *sagemakerStore, req smeFleetNameReq, _ time.Time) (any, *sagemakerResourceError) {
	f, err := st.smeFleetLocked(req.DeviceFleetName)
	if err != nil {
		return nil, err
	}
	out := map[string]any{"DeviceFleetName": f.Name, "DeviceFleetArn": f.ARN, "OutputConfig": f.Output,
		"CreationTime": sagemakerEpoch(f.Created), "LastModifiedTime": sagemakerEpoch(f.Modified)}
	for k, v := range map[string]string{"Description": f.Description, "RoleArn": f.RoleArn, "IotRoleAlias": f.IotRoleAlias} {
		if v != "" {
			out[k] = v
		}
	}
	return out, nil
}

func smeUpdateDeviceFleet(st *sagemakerStore, req smeFleetReq, now time.Time) (any, *sagemakerResourceError) {
	f, err := st.smeFleetLocked(req.DeviceFleetName)
	if err != nil {
		return nil, err
	}
	f.apply(req)
	f.Modified = now
	return nil, nil
}

func smeDeleteDeviceFleet(st *sagemakerStore, req smeFleetNameReq, _ time.Time) (any, *sagemakerResourceError) {
	f, err := st.smeFleetLocked(req.DeviceFleetName)
	if err != nil {
		return nil, err
	}
	for _, d := range st.devices {
		if d.Fleet == f.Name {
			return nil, sagemakerResErr(sagemakerErrInUse, "Device fleet %s still has registered devices; deregister them first.", f.Name)
		}
	}
	for _, p := range st.edgePlans {
		if p.Fleet == f.Name {
			return nil, sagemakerResErr(sagemakerErrInUse, "Edge deployment plan %s uses device fleet %s.", p.Name, f.Name)
		}
	}
	delete(st.deviceFleets, f.Name)
	delete(st.tags, f.ARN)
	return nil, nil
}

type smeListFleetsReq struct {
	NextToken    string
	MaxResults   int
	NameContains string
	SortBy       string
	SortOrder    string
}

// ponytail: the Creation/LastModifiedTime filters are not applied.
func smeListDeviceFleets(st *sagemakerStore, req smeListFleetsReq, _ time.Time) (any, *sagemakerResourceError) {
	desc := req.SortOrder != "Ascending"
	var names []string
	switch req.SortBy {
	case "NAME":
		names = sagemakerSortNames(st.deviceFleets, false, desc, nil)
	case "LAST_MODIFIED_TIME":
		names = sagemakerSortNames(st.deviceFleets, true, desc, func(f *smeFleet) time.Time { return f.Modified })
	default:
		names = sagemakerSortNames(st.deviceFleets, true, desc, func(f *smeFleet) time.Time { return f.Created })
	}
	var keep []*smeFleet
	for _, name := range names {
		if req.NameContains == "" || strings.Contains(name, req.NameContains) {
			keep = append(keep, st.deviceFleets[name])
		}
	}
	start, end, next, err := sagemakerPage(len(keep), req.NextToken, req.MaxResults)
	if err != nil {
		return nil, err
	}
	items := []any{}
	for _, f := range keep[start:end] {
		items = append(items, map[string]any{"DeviceFleetArn": f.ARN, "DeviceFleetName": f.Name,
			"CreationTime": sagemakerEpoch(f.Created), "LastModifiedTime": sagemakerEpoch(f.Modified)})
	}
	out := map[string]any{"DeviceFleetSummaries": items}
	if next != "" {
		out["NextToken"] = next
	}
	return out, nil
}

// ---- devices ----

type smeDeviceDef struct {
	DeviceName   string
	Description  string
	IotThingName string
}

type smeDevicesReq struct {
	DeviceFleetName string
	Devices         []smeDeviceDef
	Tags            []sagemakerTag
}

func smeRegisterDevices(st *sagemakerStore, req smeDevicesReq, now time.Time) (any, *sagemakerResourceError) {
	f, err := st.smeFleetLocked(req.DeviceFleetName)
	if err != nil {
		return nil, err
	}
	if len(req.Devices) > smeMaxDevicesPerCall {
		return nil, sagemakerResErr(sagemakerErrValidation, "Register at most %d devices per call.", smeMaxDevicesPerCall)
	}
	added := map[string]bool{}
	for _, d := range req.Devices {
		if old := st.devices[d.DeviceName]; old != nil && old.Fleet != f.Name {
			return nil, sagemakerResErr(sagemakerErrValidation, "Device %s is registered to device fleet %s.", d.DeviceName, old.Fleet)
		} else if old == nil {
			added[d.DeviceName] = true
		}
	}
	if len(st.devices)+len(added) > smeMaxDevices {
		return nil, sagemakerResErr(sagemakerErrLimit, "Stackyard keeps at most %d devices.", smeMaxDevices)
	}
	for _, d := range req.Devices {
		dev := st.devices[d.DeviceName]
		if dev == nil {
			dev = &smeDevice{Name: d.DeviceName, Fleet: f.Name, Registered: now,
				ARN: sagemakerResourceARN("device-fleet", f.Name+"/device/"+d.DeviceName)}
			st.devices[d.DeviceName] = dev
		}
		dev.Description, dev.IotThingName = d.Description, d.IotThingName
		st.setTagsLocked(dev.ARN, req.Tags)
	}
	return nil, nil
}

type smeDeregisterReq struct {
	DeviceFleetName string
	DeviceNames     []string
}

func smeDeregisterDevices(st *sagemakerStore, req smeDeregisterReq, _ time.Time) (any, *sagemakerResourceError) {
	f, err := st.smeFleetLocked(req.DeviceFleetName)
	if err != nil {
		return nil, err
	}
	if len(req.DeviceNames) > smeMaxDevicesPerCall {
		return nil, sagemakerResErr(sagemakerErrValidation, "Deregister at most %d devices per call.", smeMaxDevicesPerCall)
	}
	for _, name := range req.DeviceNames {
		dev := st.devices[name]
		if dev == nil || dev.Fleet != f.Name {
			continue // deregistering an unknown device is a no-op
		}
		delete(st.devices, name)
		delete(st.tags, dev.ARN)
		for _, p := range st.edgePlans {
			delete(p.results, name)
		}
	}
	return nil, nil
}

func smeUpdateDevices(st *sagemakerStore, req smeDevicesReq, _ time.Time) (any, *sagemakerResourceError) {
	f, err := st.smeFleetLocked(req.DeviceFleetName)
	if err != nil {
		return nil, err
	}
	if len(req.Devices) > smeMaxDevicesPerCall {
		return nil, sagemakerResErr(sagemakerErrValidation, "Update at most %d devices per call.", smeMaxDevicesPerCall)
	}
	for _, d := range req.Devices {
		if dev := st.devices[d.DeviceName]; dev == nil || dev.Fleet != f.Name {
			return nil, sagemakerResErr(sagemakerErrValidation, "Device %s is not registered to device fleet %s.", d.DeviceName, f.Name)
		}
	}
	for _, d := range req.Devices {
		dev := st.devices[d.DeviceName]
		dev.Description, dev.IotThingName = d.Description, d.IotThingName
	}
	return nil, nil
}

// smeDeviceLocked returns a device registered to fleet.
func (s *sagemakerStore) smeDeviceLocked(name, fleet string) (*smeDevice, *sagemakerResourceError) {
	if d := s.devices[name]; d != nil && d.Fleet == fleet {
		return d, nil
	}
	return nil, sagemakerResErr(sagemakerErrNotFound, "Device %s is not registered to device fleet %s.", name, fleet)
}

type smeDeviceReq struct {
	DeviceName      string
	DeviceFleetName string
}

func smeDescribeDevice(st *sagemakerStore, req smeDeviceReq, _ time.Time) (any, *sagemakerResourceError) {
	d, err := st.smeDeviceLocked(req.DeviceName, req.DeviceFleetName)
	if err != nil {
		return nil, err
	}
	out := d.summary()
	if len(d.Models) > 0 {
		out["Models"] = d.Models
		out["MaxModels"] = len(d.Models)
	}
	return out, nil
}

func (d *smeDevice) summary() map[string]any {
	out := map[string]any{"DeviceName": d.Name, "DeviceArn": d.ARN, "DeviceFleetName": d.Fleet,
		"RegistrationTime": sagemakerEpoch(d.Registered)}
	for k, v := range map[string]string{"Description": d.Description, "IotThingName": d.IotThingName, "AgentVersion": d.AgentVersion} {
		if v != "" {
			out[k] = v
		}
	}
	if !d.Heartbeat.IsZero() {
		out["LatestHeartbeat"] = sagemakerEpoch(d.Heartbeat)
	}
	return out
}

type smeListDevicesReq struct {
	NextToken            string
	MaxResults           int
	LatestHeartbeatAfter *sagemakerTimestamp
	ModelName            string
	DeviceFleetName      string
}

func smeListDevices(st *sagemakerStore, req smeListDevicesReq, _ time.Time) (any, *sagemakerResourceError) {
	var keep []*smeDevice
	for _, name := range sagemakerSortNames(st.devices, false, false, nil) {
		d := st.devices[name]
		runs := req.ModelName == "" || slices.ContainsFunc(d.Models, func(m smeEdgeModel) bool { return m.ModelName == req.ModelName })
		if (req.DeviceFleetName != "" && d.Fleet != req.DeviceFleetName) || !runs ||
			(req.LatestHeartbeatAfter != nil && !d.Heartbeat.After(req.LatestHeartbeatAfter.Time)) {
			continue
		}
		keep = append(keep, d)
	}
	start, end, next, err := sagemakerPage(len(keep), req.NextToken, req.MaxResults)
	if err != nil {
		return nil, err
	}
	items := []any{}
	for _, d := range keep[start:end] {
		item := d.summary()
		if len(d.Models) > 0 {
			models := make([]any, 0, len(d.Models))
			for _, m := range d.Models {
				models = append(models, map[string]any{"ModelName": m.ModelName, "ModelVersion": m.ModelVersion})
			}
			item["Models"] = models
		}
		items = append(items, item)
	}
	out := map[string]any{"DeviceSummaries": items}
	if next != "" {
		out["NextToken"] = next
	}
	return out, nil
}

// ---- edge deployment plans ----

type smeCreatePlanReq struct {
	EdgeDeploymentPlanName string
	ModelConfigs           []smeModelConfig
	DeviceFleetName        string
	Stages                 []smeStageDef
	Tags                   []sagemakerTag
}

func smeCreatePlan(st *sagemakerStore, req smeCreatePlanReq, now time.Time) (any, *sagemakerResourceError) {
	if st.edgePlans[req.EdgeDeploymentPlanName] != nil {
		return nil, sagemakerResErr(sagemakerErrInUse, "Edge deployment plan %s already exists.", req.EdgeDeploymentPlanName)
	}
	if len(st.edgePlans) >= smeMaxPlans {
		return nil, sagemakerResErr(sagemakerErrLimit, "Stackyard keeps at most %d edge deployment plans.", smeMaxPlans)
	}
	if _, err := st.smeFleetLocked(req.DeviceFleetName); err != nil {
		return nil, err
	}
	if len(req.ModelConfigs) == 0 || len(req.ModelConfigs) > smeMaxModelConfigs {
		return nil, sagemakerResErr(sagemakerErrValidation, "An edge deployment plan needs 1 to %d ModelConfigs.", smeMaxModelConfigs)
	}
	if len(req.Stages) > smeMaxStages {
		return nil, sagemakerResErr(sagemakerErrValidation, "An edge deployment plan has at most %d stages.", smeMaxStages)
	}
	p := &smePlan{Name: req.EdgeDeploymentPlanName, ARN: sagemakerResourceARN("edge-deployment", req.EdgeDeploymentPlanName),
		Fleet: req.DeviceFleetName, ModelConfigs: req.ModelConfigs, Created: now, Modified: now, results: map[string]string{}}
	for _, def := range req.Stages {
		if err := p.addStage(def); err != nil {
			return nil, err
		}
	}
	st.edgePlans[p.Name] = p
	st.setTagsLocked(p.ARN, req.Tags)
	return map[string]any{"EdgeDeploymentPlanArn": p.ARN}, nil
}

func (p *smePlan) addStage(def smeStageDef) *sagemakerResourceError {
	if p.stage(def.StageName) != nil {
		return sagemakerResErr(sagemakerErrValidation, "Stage %s is defined more than once.", def.StageName)
	}
	sel := def.DeviceSelectionConfig
	switch {
	case sel.DeviceSubsetType == "PERCENTAGE" && (sel.Percentage == nil || *sel.Percentage < 0):
		return sagemakerResErr(sagemakerErrValidation, "Stage %s: PERCENTAGE needs a Percentage between 0 and 100.", def.StageName)
	case sel.DeviceSubsetType == "SELECTION" && (len(sel.DeviceNames) == 0 || len(sel.DeviceNames) > smeMaxSelectedDevices):
		return sagemakerResErr(sagemakerErrValidation, "Stage %s: SELECTION needs 1 to %d DeviceNames.", def.StageName, smeMaxSelectedDevices)
	case sel.DeviceSubsetType == "NAMECONTAINS" && sel.DeviceNameContains == "":
		return sagemakerResErr(sagemakerErrValidation, "Stage %s: NAMECONTAINS needs DeviceNameContains.", def.StageName)
	}
	if def.DeploymentConfig == nil {
		def.DeploymentConfig = &smeDeploymentConfig{FailureHandlingPolicy: "ROLLBACK_ON_FAILURE"}
	}
	p.Stages = append(p.Stages, &smeStage{smeStageDef: def, status: "READYTODEPLOY"})
	return nil
}

func (p *smePlan) stage(name string) *smeStage {
	for _, s := range p.Stages {
		if s.StageName == name {
			return s
		}
	}
	return nil
}

func (s *sagemakerStore) smePlanLocked(name string) (*smePlan, *sagemakerResourceError) {
	if p := s.edgePlans[name]; p != nil {
		return p, nil
	}
	return nil, sagemakerResErr(sagemakerErrNotFound, "Edge deployment plan %s does not exist.", name)
}

type smePlanReq struct {
	EdgeDeploymentPlanName string
	StageName              string
	NextToken              string
	MaxResults             int
}

func smeDescribePlan(st *sagemakerStore, req smePlanReq, _ time.Time) (any, *sagemakerResourceError) {
	p, err := st.smePlanLocked(req.EdgeDeploymentPlanName)
	if err != nil {
		return nil, err
	}
	start, end, next, perr := sagemakerPage(len(p.Stages), req.NextToken, req.MaxResults)
	if perr != nil {
		return nil, perr
	}
	// One pass over the fleet's devices (bounded by smeMaxDevices) per stage.
	var fleet []string
	for name, d := range st.devices {
		if d.Fleet == p.Fleet {
			fleet = append(fleet, name)
		}
	}
	totals := [3]int{}
	stages := []any{}
	for i, s := range p.Stages {
		counts := [3]int{} // success, pending, failed
		selected := 0
		for _, name := range fleet {
			if !s.DeviceSelectionConfig.selects(name) {
				continue
			}
			selected++
			switch p.results[name] {
			case "SUCCESS":
				counts[0]++
			case "FAIL":
				counts[2]++
			default:
				counts[1]++
			}
		}
		if s.status == "INPROGRESS" && selected > 0 && counts[1] == 0 {
			s.status = "DEPLOYED"
			if counts[2] > 0 {
				s.status = "FAILED"
			}
		}
		for j := range totals {
			totals[j] += counts[j]
		}
		if i < start || i >= end {
			continue
		}
		status := map[string]any{"StageStatus": s.status, "EdgeDeploymentSuccessInStage": counts[0],
			"EdgeDeploymentPendingInStage": counts[1], "EdgeDeploymentFailedInStage": counts[2]}
		if !s.started.IsZero() {
			status["EdgeDeploymentStageStartTime"] = sagemakerEpoch(s.started)
		}
		stages = append(stages, map[string]any{"StageName": s.StageName, "DeviceSelectionConfig": s.DeviceSelectionConfig,
			"DeploymentConfig": s.DeploymentConfig, "DeploymentStatus": status})
	}
	out := map[string]any{"EdgeDeploymentPlanArn": p.ARN, "EdgeDeploymentPlanName": p.Name,
		"ModelConfigs": p.ModelConfigs, "DeviceFleetName": p.Fleet, "Stages": stages,
		"EdgeDeploymentSuccess": totals[0], "EdgeDeploymentPending": totals[1], "EdgeDeploymentFailed": totals[2],
		"CreationTime": sagemakerEpoch(p.Created), "LastModifiedTime": sagemakerEpoch(p.Modified)}
	if next != "" {
		out["NextToken"] = next
	}
	return out, nil
}

func smeDeletePlan(st *sagemakerStore, req smePlanReq, now time.Time) (any, *sagemakerResourceError) {
	p, err := st.smePlanLocked(req.EdgeDeploymentPlanName)
	if err != nil {
		return nil, err
	}
	// Describing settles stages whose devices have all reported.
	if _, err := smeDescribePlan(st, smePlanReq{EdgeDeploymentPlanName: p.Name}, now); err != nil {
		return nil, err
	}
	for _, s := range p.Stages {
		if s.status == "INPROGRESS" {
			return nil, sagemakerResErr(sagemakerErrInUse, "Stage %s of edge deployment plan %s is in progress; stop it first.", s.StageName, p.Name)
		}
	}
	delete(st.edgePlans, p.Name)
	delete(st.tags, p.ARN)
	return nil, nil
}

func (s *sagemakerStore) smeStageLocked(req smePlanReq) (*smePlan, *smeStage, *sagemakerResourceError) {
	p, err := s.smePlanLocked(req.EdgeDeploymentPlanName)
	if err != nil {
		return nil, nil, err
	}
	stage := p.stage(req.StageName)
	if stage == nil {
		return nil, nil, sagemakerResErr(sagemakerErrNotFound, "Edge deployment plan %s has no stage %s.", p.Name, req.StageName)
	}
	return p, stage, nil
}

func smeStartStage(st *sagemakerStore, req smePlanReq, now time.Time) (any, *sagemakerResourceError) {
	p, stage, err := st.smeStageLocked(req)
	if err != nil {
		return nil, err
	}
	if stage.status == "INPROGRESS" {
		return nil, sagemakerResErr(sagemakerErrValidation, "Stage %s is already in progress.", stage.StageName)
	}
	// A new rollout: the stage's devices report afresh.
	for name := range p.results {
		if stage.DeviceSelectionConfig.selects(name) {
			delete(p.results, name)
		}
	}
	stage.status, stage.started, p.Modified = "INPROGRESS", now, now
	return nil, nil
}

func smeStopStage(st *sagemakerStore, req smePlanReq, now time.Time) (any, *sagemakerResourceError) {
	p, stage, err := st.smeStageLocked(req)
	if err != nil {
		return nil, err
	}
	if stage.status != "INPROGRESS" {
		return nil, sagemakerResErr(sagemakerErrValidation, "Stage %s is %s, not INPROGRESS.", stage.StageName, stage.status)
	}
	stage.status, p.Modified = "STOPPED", now
	return nil, nil
}
