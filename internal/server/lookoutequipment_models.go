package server

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// Models, model versions, retraining schedulers and inference schedulers.
// Training and imports go IN_PROGRESS -> SUCCESS; schedulers go PENDING ->
// RUNNING and STOPPING -> STOPPED. Inference executions are derived from the
// scheduler's RUNNING spells at its upload frequency, so they appear and
// accumulate in real time.

const (
	lkeMaxVersions   = 50  // per model
	lkeMaxRuns       = 20  // RUNNING spells remembered per scheduler
	lkeMaxExecutions = 500 // executions derived per request
)

var lkeFrequencies = map[string]time.Duration{
	"PT5M": 5 * time.Minute, "PT10M": 10 * time.Minute, "PT15M": 15 * time.Minute, "PT30M": 30 * time.Minute, "PT1H": time.Hour,
}

// lkeModelFields are what a model reports from its active (or latest) version.
var lkeModelFields = []string{"DatasetName", "DatasetArn", "Schema", "LabelsInputConfiguration", "TrainingDataStartTime",
	"TrainingDataEndTime", "EvaluationDataStartTime", "EvaluationDataEndTime", "RoleArn", "DataPreProcessingConfiguration",
	"ServerSideKmsKeyId", "OffCondition", "ModelDiagnosticsOutputConfiguration", "TrainingExecutionStartTime",
	"TrainingExecutionEndTime", "ModelMetrics", "SourceModelVersionArn", "ImportJobStartTime", "ImportJobEndTime", "ModelQuality"}

func (st *lkeStore) model(name string) (*lkeEnt, *fcErr) {
	m := st.models[name]
	if m == nil {
		return nil, fcNotFound("Model %s does not exist", name)
	}
	st.advanceModel(m)
	return m, nil
}

func lkeVersionBusy(v *lkeEnt) bool {
	s := v.doc["Status"]
	return s == "IN_PROGRESS" || s == "IMPORT_IN_PROGRESS"
}

// advanceModel finishes the model's latest version once it has settled and
// makes it the active version.
func (st *lkeStore) advanceModel(m *lkeEnt) {
	if r := m.retrainer; r != nil {
		st.advanceSwitch(r)
		m.doc["RetrainingSchedulerStatus"] = r.doc["Status"]
	}
	if len(m.versions) == 0 {
		return
	}
	v := m.versions[len(m.versions)-1]
	if !lkeVersionBusy(v) || !lkeSettled(v) {
		return
	}
	now := lkeNow()
	if v.doc["Status"] == "IMPORT_IN_PROGRESS" {
		v.doc["ImportJobEndTime"] = now
	} else {
		v.doc["TrainingExecutionEndTime"] = now
	}
	v.doc["Status"], v.doc["LastUpdatedTime"], v.doc["ModelQuality"] = "SUCCESS", now, "QUALITY_THRESHOLD_MET"
	if v.doc["ModelMetrics"] == nil {
		v.doc["ModelMetrics"] = st.modelMetrics(v)
	}
	st.activate(m, v)
	if m.doc["Status"] != "SUCCESS" {
		m.doc["Status"] = "SUCCESS"
	}
}

// modelMetrics is a placeholder evaluation over the version's labels.
func (st *lkeStore) modelMetrics(v *lkeEnt) string {
	labeled := []any{}
	if g := st.groups[fcStr(fcObj(v.doc, "LabelsInputConfiguration"), "LabelGroupName")]; g != nil {
		for _, l := range g.labels {
			if len(labeled) >= 100 {
				break
			}
			labeled = append(labeled, map[string]any{"start": l.doc["StartTime"], "end": l.doc["EndTime"]})
		}
	}
	raw, _ := json.Marshal(map[string]any{"labels_count": len(labeled), "labeled_ranges": labeled, "predicted_ranges": []any{},
		"unknown_sensors": []any{}, "latent_variables": []any{}})
	return string(raw)
}

// activate makes v the model's active version.
func (st *lkeStore) activate(m, v *lkeEnt) {
	now := lkeNow()
	if m.doc["ActiveModelVersion"] != nil && m.doc["ActiveModelVersion"] != v.doc["ModelVersion"] {
		m.doc["PreviousActiveModelVersion"], m.doc["PreviousActiveModelVersionArn"] = m.doc["ActiveModelVersion"], m.doc["ActiveModelVersionArn"]
		m.doc["PreviousModelVersionActivatedAt"] = m.doc["ModelVersionActivatedAt"]
		m.doc["PriorModelMetrics"] = m.doc["ModelMetrics"]
	}
	m.doc["ActiveModelVersion"], m.doc["ActiveModelVersionArn"], m.doc["ModelVersionActivatedAt"] = v.doc["ModelVersion"], v.arn, now
	for _, k := range lkeModelFields {
		if val, ok := v.doc[k]; ok {
			m.doc[k] = val
		} else {
			delete(m.doc, k)
		}
	}
	m.doc["LastUpdatedTime"] = now
}

// addVersion appends a version to m.
func (st *lkeStore) addVersion(m *lkeEnt, doc map[string]any) (*lkeEnt, *fcErr) {
	if len(m.versions) >= lkeMaxVersions {
		return nil, lkeQuota("a model can have at most %d versions", lkeMaxVersions)
	}
	n := len(m.versions) + 1
	arn := fcStr(m.doc, "ModelArn") + "/model-version/" + strconv.Itoa(n)
	now := lkeNow()
	doc["ModelName"], doc["ModelArn"], doc["ModelVersion"], doc["ModelVersionArn"] = m.doc["ModelName"], m.doc["ModelArn"], n, arn
	doc["CreatedAt"], doc["LastUpdatedTime"] = now, now
	v := &lkeEnt{kind: "version", arn: arn, doc: doc, model: m}
	if err := st.add(v, nil); err != nil {
		return nil, err
	}
	m.versions = append(m.versions, v)
	return v, nil
}

func (st *lkeStore) newModel(name string, tags any) (*lkeEnt, *fcErr) {
	if st.models[name] != nil {
		return nil, lkeConflict("Model %s already exists", name)
	}
	if len(st.models) >= lkeMaxPerKind {
		return nil, lkeQuota("at most %d models", lkeMaxPerKind)
	}
	arn := lkeArn("model", name)
	now := lkeNow()
	m := &lkeEnt{kind: "model", arn: arn, doc: map[string]any{"ModelName": name, "ModelArn": arn, "CreatedAt": now, "LastUpdatedTime": now}}
	if err := st.add(m, tags); err != nil {
		return nil, err
	}
	st.models[name] = m
	return m, nil
}

func (st *lkeStore) dropModel(m *lkeEnt) {
	for _, v := range m.versions {
		st.release(v)
	}
	if m.retrainer != nil {
		st.release(m.retrainer)
	}
	st.release(m)
	delete(st.models, fcStr(m.doc, "ModelName"))
}

// activeDataset returns the named dataset, which must hold ingested data.
func (st *lkeStore) activeDataset(name string) (*lkeEnt, *fcErr) {
	d, err := st.dataset(name)
	if err != nil {
		return nil, err
	}
	if d.doc["Status"] != "ACTIVE" {
		return nil, lkeInvalid("Dataset %s has no ingested data (status %s)", name, d.doc["Status"])
	}
	return d, nil
}

func (st *lkeStore) checkLabels(cfg map[string]any) *fcErr {
	if name := fcStr(cfg, "LabelGroupName"); name != "" {
		if _, err := st.group(name); err != nil {
			return err
		}
	}
	return nil
}

func (st *lkeStore) createModel(in map[string]any) (map[string]any, *fcErr) {
	d, err := st.activeDataset(fcStr(in, "DatasetName"))
	if err != nil {
		return nil, err
	}
	if err := st.checkLabels(fcObj(in, "LabelsInputConfiguration")); err != nil {
		return nil, err
	}
	for _, pair := range [][2]string{{"TrainingDataStartTime", "TrainingDataEndTime"}, {"EvaluationDataStartTime", "EvaluationDataEndTime"}} {
		if a, b := amlTime(in[pair[0]]), amlTime(in[pair[1]]); a > 0 && b > 0 && b <= a {
			return nil, lkeInvalid("%s must be after %s", pair[1], pair[0])
		}
	}
	schema := fcStr(fcObj(in, "DatasetSchema"), "InlineDataSchema")
	if schema == "" {
		schema = fcStr(d.doc, "Schema")
	}
	m, err := st.newModel(fcStr(in, "ModelName"), in["Tags"])
	if err != nil {
		return nil, err
	}
	doc := lkePick(in, "LabelsInputConfiguration", "TrainingDataStartTime", "TrainingDataEndTime", "EvaluationDataStartTime",
		"EvaluationDataEndTime", "RoleArn", "DataPreProcessingConfiguration", "ServerSideKmsKeyId", "OffCondition",
		"ModelDiagnosticsOutputConfiguration")
	doc["DatasetName"], doc["DatasetArn"], doc["Status"], doc["SourceType"] = d.doc["DatasetName"], d.doc["DatasetArn"], "IN_PROGRESS", "TRAINING"
	doc["TrainingExecutionStartTime"] = lkeNow()
	if schema != "" {
		doc["Schema"] = schema
	}
	if _, err := st.addVersion(m, doc); err != nil {
		st.dropModel(m)
		return nil, err
	}
	for _, k := range lkeModelFields {
		if v, ok := doc[k]; ok {
			m.doc[k] = v
		}
	}
	m.doc["Status"] = "IN_PROGRESS"
	return map[string]any{"ModelArn": m.arn, "Status": "IN_PROGRESS"}, nil
}

func (st *lkeStore) describeModel(in map[string]any) (map[string]any, *fcErr) {
	m, err := st.model(fcStr(in, "ModelName"))
	if err != nil {
		return nil, err
	}
	return m.doc, nil
}

func (st *lkeStore) updateModel(in map[string]any) *fcErr {
	m, err := st.model(fcStr(in, "ModelName"))
	if err != nil {
		return err
	}
	if err := st.checkLabels(fcObj(in, "LabelsInputConfiguration")); err != nil {
		return err
	}
	doc := lkeCopy(m.doc)
	for _, k := range []string{"LabelsInputConfiguration", "RoleArn", "ModelDiagnosticsOutputConfiguration"} {
		if v, ok := in[k]; ok {
			doc[k] = v
		}
	}
	doc["LastUpdatedTime"] = lkeNow()
	return st.replace(m, doc)
}

func (st *lkeStore) deleteModel(in map[string]any) *fcErr {
	m, err := st.model(fcStr(in, "ModelName"))
	if err != nil {
		return err
	}
	for _, s := range st.schedulers {
		if s.model == m {
			return lkeConflict("Model %s has inference scheduler %s", fcStr(m.doc, "ModelName"), fcStr(s.doc, "InferenceSchedulerName"))
		}
	}
	st.dropModel(m)
	return nil
}

func (st *lkeStore) listModels(in map[string]any) (map[string]any, *fcErr) {
	prefix, dsPrefix, status := fcStr(in, "ModelNameBeginsWith"), fcStr(in, "DatasetNameBeginsWith"), fcStr(in, "Status")
	var items []map[string]any
	for name, m := range st.models {
		st.advanceModel(m)
		if !strings.HasPrefix(name, prefix) || !strings.HasPrefix(fcStr(m.doc, "DatasetName"), dsPrefix) || (status != "" && m.doc["Status"] != status) {
			continue
		}
		items = append(items, lkePick(m.doc, "ModelName", "ModelArn", "DatasetName", "DatasetArn", "Status", "CreatedAt",
			"ActiveModelVersion", "ActiveModelVersionArn", "RetrainingSchedulerStatus", "NextScheduledRetrainingStartDate",
			"ModelDiagnosticsOutputConfiguration", "ModelQuality"))
	}
	return lkePage(items, in, "ModelName", "ModelSummaries")
}

// version returns version n of m; n is clamped against the versions held.
func lkeVersion(m *lkeEnt, n float64) (*lkeEnt, *fcErr) {
	if n < 1 || n > float64(len(m.versions)) || n != math.Trunc(n) {
		return nil, fcNotFound("Model %s has no version %v", fcStr(m.doc, "ModelName"), n)
	}
	return m.versions[int(n)-1], nil
}

func (st *lkeStore) describeVersion(in map[string]any) (map[string]any, *fcErr) {
	m, err := st.model(fcStr(in, "ModelName"))
	if err != nil {
		return nil, err
	}
	n, _ := fcNum(in, "ModelVersion")
	v, err := lkeVersion(m, n)
	if err != nil {
		return nil, err
	}
	return v.doc, nil
}

func (st *lkeStore) listVersions(in map[string]any) (map[string]any, *fcErr) {
	m, err := st.model(fcStr(in, "ModelName"))
	if err != nil {
		return nil, err
	}
	status, source := fcStr(in, "Status"), fcStr(in, "SourceType")
	from, to := amlTime(in["CreatedAtStartTime"]), amlTime(in["CreatedAtEndTime"])
	lo, hasLo := fcNum(in, "MinModelVersion")
	hi, hasHi := fcNum(in, "MaxModelVersion")
	var items []map[string]any
	for _, v := range m.versions {
		n, _ := v.doc["ModelVersion"].(int)
		created := amlTime(v.doc["CreatedAt"])
		if (status != "" && v.doc["Status"] != status) || (source != "" && v.doc["SourceType"] != source) ||
			(from > 0 && created < from) || (to > 0 && created > to) || (hasLo && float64(n) < lo) || (hasHi && float64(n) > hi) {
			continue
		}
		item := lkePick(v.doc, "ModelName", "ModelArn", "ModelVersion", "ModelVersionArn", "CreatedAt", "Status", "SourceType", "ModelQuality")
		item["_key"] = fmt.Sprintf("%06d", lkeMaxVersions-n) // newest first
		items = append(items, item)
	}
	return lkePage(items, in, "_key", "ModelVersionSummaries")
}

func (st *lkeStore) importVersion(in map[string]any) (map[string]any, *fcErr) {
	src := st.byArn[fcStr(in, "SourceModelVersionArn")]
	if src == nil || src.kind != "version" {
		return nil, fcNotFound("Model version %s does not exist", fcStr(in, "SourceModelVersionArn"))
	}
	if src.doc["Status"] != "SUCCESS" {
		return nil, lkeConflict("Model version %s is %s, not SUCCESS", src.arn, src.doc["Status"])
	}
	d, err := st.activeDataset(fcStr(in, "DatasetName"))
	if err != nil {
		return nil, err
	}
	if err := st.checkLabels(fcObj(in, "LabelsInputConfiguration")); err != nil {
		return nil, err
	}
	name := fcStr(in, "ModelName")
	if name == "" {
		name = fcStr(src.doc, "ModelName")
	}
	m := st.models[name]
	created := m == nil
	if created {
		if m, err = st.newModel(name, in["Tags"]); err != nil {
			return nil, err
		}
		m.doc["Status"] = "IMPORT_IN_PROGRESS"
	} else {
		st.advanceModel(m)
		if n := len(m.versions); n > 0 && lkeVersionBusy(m.versions[n-1]) {
			return nil, lkeConflict("Model %s already has a version in progress", name)
		}
	}
	doc := lkePick(src.doc, "Schema", "TrainingDataStartTime", "TrainingDataEndTime", "EvaluationDataStartTime",
		"EvaluationDataEndTime", "DataPreProcessingConfiguration", "OffCondition", "ModelMetrics")
	for _, k := range []string{"LabelsInputConfiguration", "RoleArn", "ServerSideKmsKeyId"} {
		if v, ok := in[k]; ok {
			doc[k] = v
		}
	}
	doc["DatasetName"], doc["DatasetArn"], doc["SourceModelVersionArn"] = d.doc["DatasetName"], d.doc["DatasetArn"], src.arn
	doc["Status"], doc["SourceType"], doc["ImportJobStartTime"] = "IMPORT_IN_PROGRESS", "IMPORT", lkeNow()
	doc["ImportedDataSizeInBytes"] = int64(src.size)
	v, err := st.addVersion(m, doc)
	if err != nil {
		if created {
			st.dropModel(m)
		}
		return nil, err
	}
	return map[string]any{"ModelName": name, "ModelArn": m.arn, "ModelVersionArn": v.arn, "ModelVersion": v.doc["ModelVersion"], "Status": "IMPORT_IN_PROGRESS"}, nil
}

func (st *lkeStore) activateVersion(in map[string]any) (map[string]any, *fcErr) {
	m, err := st.model(fcStr(in, "ModelName"))
	if err != nil {
		return nil, err
	}
	n, _ := fcNum(in, "ModelVersion")
	v, err := lkeVersion(m, n)
	if err != nil {
		return nil, err
	}
	if v.doc["Status"] != "SUCCESS" {
		return nil, lkeInvalid("Model version %v is %s, not SUCCESS", n, v.doc["Status"])
	}
	prev, prevArn := m.doc["ActiveModelVersion"], m.doc["ActiveModelVersionArn"]
	st.activate(m, v)
	out := map[string]any{"ModelName": m.doc["ModelName"], "ModelArn": m.arn, "CurrentActiveVersion": v.doc["ModelVersion"], "CurrentActiveVersionArn": v.arn}
	if prev != nil {
		out["PreviousActiveVersion"], out["PreviousActiveVersionArn"] = prev, prevArn
	}
	return out, nil
}

// ---- schedulers (retraining and inference) ----

// advanceSwitch settles a scheduler's PENDING or STOPPING status.
func (st *lkeStore) advanceSwitch(e *lkeEnt) {
	next := map[any]string{"PENDING": "RUNNING", "STOPPING": "STOPPED"}[e.doc["Status"]]
	if next == "" || !lkeSettled(e) {
		return
	}
	now := time.Now().UTC()
	e.doc["Status"], e.doc["UpdatedAt"] = next, fcEpoch(now)
	if e.kind != "scheduler" {
		return
	}
	if next == "RUNNING" {
		e.runs = append(e.runs, [2]time.Time{now})
		if len(e.runs) > lkeMaxRuns {
			e.runs = e.runs[len(e.runs)-lkeMaxRuns:]
		}
	} else if n := len(e.runs); n > 0 && e.runs[n-1][1].IsZero() {
		e.runs[n-1][1] = now
	}
}

// flip starts or stops a scheduler.
func (st *lkeStore) flip(e *lkeEnt, start bool, what string) *fcErr {
	status := e.doc["Status"]
	switch {
	case start && status != "STOPPED":
		return lkeConflict("%s is %s; only a STOPPED scheduler can start", what, status)
	case !start && status != "RUNNING" && status != "PENDING":
		return lkeConflict("%s is %s; only a RUNNING or PENDING scheduler can stop", what, status)
	}
	if !start && status == "PENDING" && e.kind == "scheduler" {
		e.runs = append(e.runs, [2]time.Time{time.Now().UTC()})
	}
	next := "PENDING"
	if !start {
		next = "STOPPING"
	}
	e.doc["Status"], e.doc["UpdatedAt"] = next, lkeNow()
	lkeBegin(e)
	return nil
}

func (st *lkeStore) retrainer(name string) (*lkeEnt, *lkeEnt, *fcErr) {
	m, err := st.model(name)
	if err != nil {
		return nil, nil, err
	}
	if m.retrainer == nil {
		return nil, nil, fcNotFound("Model %s has no retraining scheduler", name)
	}
	return m, m.retrainer, nil
}

func (st *lkeStore) createRetrainer(in map[string]any) (map[string]any, *fcErr) {
	m, err := st.model(fcStr(in, "ModelName"))
	if err != nil {
		return nil, err
	}
	if m.retrainer != nil {
		return nil, lkeConflict("Model %s already has a retraining scheduler", fcStr(m.doc, "ModelName"))
	}
	if m.doc["Status"] != "SUCCESS" {
		return nil, lkeInvalid("Model %s is %s; it must finish training first", fcStr(m.doc, "ModelName"), m.doc["Status"])
	}
	now := lkeNow()
	doc := lkePick(in, "RetrainingStartDate", "RetrainingFrequency", "LookbackWindow", "PromoteMode")
	if doc["PromoteMode"] == nil {
		doc["PromoteMode"] = "MANAGED"
	}
	doc["ModelName"], doc["ModelArn"], doc["Status"], doc["CreatedAt"], doc["UpdatedAt"] = m.doc["ModelName"], m.arn, "PENDING", now, now
	r := &lkeEnt{kind: "retrainer", doc: doc, model: m}
	if err := st.add(r, nil); err != nil {
		return nil, err
	}
	m.retrainer = r
	next := amlTime(doc["RetrainingStartDate"])
	if next == 0 {
		next = now + 86400
	}
	m.doc["RetrainingSchedulerStatus"], m.doc["NextScheduledRetrainingStartDate"] = "PENDING", next
	return lkePick(doc, "ModelName", "ModelArn", "Status"), nil
}

func (st *lkeStore) describeRetrainer(in map[string]any) (map[string]any, *fcErr) {
	_, r, err := st.retrainer(fcStr(in, "ModelName"))
	if err != nil {
		return nil, err
	}
	return r.doc, nil
}

func (st *lkeStore) updateRetrainer(in map[string]any) *fcErr {
	_, r, err := st.retrainer(fcStr(in, "ModelName"))
	if err != nil {
		return err
	}
	doc := lkeCopy(r.doc)
	for _, k := range []string{"RetrainingStartDate", "RetrainingFrequency", "LookbackWindow", "PromoteMode"} {
		if v, ok := in[k]; ok {
			doc[k] = v
		}
	}
	doc["UpdatedAt"] = lkeNow()
	return st.replace(r, doc)
}

func (st *lkeStore) deleteRetrainer(in map[string]any) *fcErr {
	m, r, err := st.retrainer(fcStr(in, "ModelName"))
	if err != nil {
		return err
	}
	if r.doc["Status"] != "STOPPED" {
		return lkeConflict("the retraining scheduler is %s; stop it before deleting it", r.doc["Status"])
	}
	st.release(r)
	m.retrainer = nil
	delete(m.doc, "RetrainingSchedulerStatus")
	delete(m.doc, "NextScheduledRetrainingStartDate")
	return nil
}

func (st *lkeStore) switchRetrainer(in map[string]any, start bool) (map[string]any, *fcErr) {
	m, r, err := st.retrainer(fcStr(in, "ModelName"))
	if err != nil {
		return nil, err
	}
	if err := st.flip(r, start, "the retraining scheduler"); err != nil {
		return nil, err
	}
	m.doc["RetrainingSchedulerStatus"] = r.doc["Status"]
	return lkePick(r.doc, "ModelName", "ModelArn", "Status"), nil
}

func (st *lkeStore) listRetrainers(in map[string]any) (map[string]any, *fcErr) {
	prefix, status := fcStr(in, "ModelNameBeginsWith"), fcStr(in, "Status")
	var items []map[string]any
	for name, m := range st.models {
		if m.retrainer == nil || !strings.HasPrefix(name, prefix) {
			continue
		}
		st.advanceModel(m)
		if status == "" || m.retrainer.doc["Status"] == status {
			items = append(items, lkePick(m.retrainer.doc, "ModelName", "ModelArn", "Status", "RetrainingStartDate", "RetrainingFrequency", "LookbackWindow"))
		}
	}
	return lkePage(items, in, "ModelName", "RetrainingSchedulerSummaries")
}

func (st *lkeStore) scheduler(name string) (*lkeEnt, *fcErr) {
	s := st.schedulers[name]
	if s == nil {
		return nil, fcNotFound("Inference scheduler %s does not exist", name)
	}
	st.advanceSwitch(s)
	if execs := st.executions(s, 1); len(execs) > 0 {
		s.doc["LatestInferenceResult"] = lkeResult(s, execs[0].tick)
	}
	return s, nil
}

func (st *lkeStore) createScheduler(in map[string]any) (map[string]any, *fcErr) {
	name := fcStr(in, "InferenceSchedulerName")
	if st.schedulers[name] != nil {
		return nil, lkeConflict("Inference scheduler %s already exists", name)
	}
	if len(st.schedulers) >= lkeMaxPerKind {
		return nil, lkeQuota("at most %d inference schedulers", lkeMaxPerKind)
	}
	m, err := st.model(fcStr(in, "ModelName"))
	if err != nil {
		return nil, err
	}
	if m.doc["ActiveModelVersion"] == nil {
		return nil, lkeInvalid("Model %s has no trained version yet", fcStr(m.doc, "ModelName"))
	}
	now := lkeNow()
	arn := lkeArn("inference-scheduler", name)
	doc := lkePick(in, "DataDelayOffsetInMinutes", "DataUploadFrequency", "DataInputConfiguration", "DataOutputConfiguration", "RoleArn", "ServerSideKmsKeyId")
	if doc["DataDelayOffsetInMinutes"] == nil {
		doc["DataDelayOffsetInMinutes"] = 0
	}
	doc["ModelName"], doc["ModelArn"], doc["InferenceSchedulerName"], doc["InferenceSchedulerArn"] = m.doc["ModelName"], m.arn, name, arn
	doc["Status"], doc["CreatedAt"], doc["UpdatedAt"] = "PENDING", now, now
	s := &lkeEnt{kind: "scheduler", arn: arn, doc: doc, model: m}
	if err := st.add(s, in["Tags"]); err != nil {
		return nil, err
	}
	st.schedulers[name] = s
	out := lkePick(doc, "InferenceSchedulerArn", "InferenceSchedulerName", "Status")
	out["ModelQuality"] = m.doc["ModelQuality"]
	return out, nil
}

func (st *lkeStore) describeScheduler(in map[string]any) (map[string]any, *fcErr) {
	s, err := st.scheduler(fcStr(in, "InferenceSchedulerName"))
	if err != nil {
		return nil, err
	}
	return s.doc, nil
}

func (st *lkeStore) updateScheduler(in map[string]any) *fcErr {
	s, err := st.scheduler(fcStr(in, "InferenceSchedulerName"))
	if err != nil {
		return err
	}
	if s.doc["Status"] == "STOPPING" {
		return lkeConflict("Inference scheduler %s is STOPPING", fcStr(s.doc, "InferenceSchedulerName"))
	}
	doc := lkeCopy(s.doc)
	for _, k := range []string{"DataDelayOffsetInMinutes", "DataUploadFrequency", "DataInputConfiguration", "DataOutputConfiguration", "RoleArn"} {
		if v, ok := in[k]; ok {
			doc[k] = v
		}
	}
	doc["UpdatedAt"] = lkeNow()
	return st.replace(s, doc)
}

func (st *lkeStore) deleteScheduler(in map[string]any) *fcErr {
	s, err := st.scheduler(fcStr(in, "InferenceSchedulerName"))
	if err != nil {
		return err
	}
	if status := s.doc["Status"]; status == "RUNNING" || status == "PENDING" {
		return lkeConflict("Inference scheduler %s is %s; stop it before deleting it", fcStr(s.doc, "InferenceSchedulerName"), status)
	}
	st.release(s)
	delete(st.schedulers, fcStr(s.doc, "InferenceSchedulerName"))
	return nil
}

func (st *lkeStore) switchScheduler(in map[string]any, start bool) (map[string]any, *fcErr) {
	s, err := st.scheduler(fcStr(in, "InferenceSchedulerName"))
	if err != nil {
		return nil, err
	}
	if err := st.flip(s, start, "inference scheduler "+fcStr(s.doc, "InferenceSchedulerName")); err != nil {
		return nil, err
	}
	return lkePick(s.doc, "ModelArn", "ModelName", "InferenceSchedulerName", "InferenceSchedulerArn", "Status"), nil
}

func (st *lkeStore) listSchedulers(in map[string]any) (map[string]any, *fcErr) {
	prefix, model, status := fcStr(in, "InferenceSchedulerNameBeginsWith"), fcStr(in, "ModelName"), fcStr(in, "Status")
	var items []map[string]any
	for name := range st.schedulers {
		if !strings.HasPrefix(name, prefix) {
			continue
		}
		s, _ := st.scheduler(name)
		if (model != "" && s.doc["ModelName"] != model) || (status != "" && s.doc["Status"] != status) {
			continue
		}
		items = append(items, lkePick(s.doc, "ModelName", "ModelArn", "InferenceSchedulerName", "InferenceSchedulerArn", "Status",
			"DataDelayOffsetInMinutes", "DataUploadFrequency", "LatestInferenceResult"))
	}
	return lkePage(items, in, "InferenceSchedulerName", "InferenceSchedulerSummaries")
}

type lkeExecution struct {
	tick, dataStart, dataEnd time.Time
}

// executions derives up to limit executions, newest first: one per upload
// interval of every RUNNING spell, the first at the spell's start.
func (st *lkeStore) executions(s *lkeEnt, limit int) []lkeExecution {
	freq := lkeFrequencies[fcStr(s.doc, "DataUploadFrequency")]
	if freq == 0 {
		freq = 5 * time.Minute
	}
	delayMin, _ := fcNum(s.doc, "DataDelayOffsetInMinutes")
	delay := time.Duration(delayMin) * time.Minute
	now := time.Now().UTC()
	var out []lkeExecution
	for i := len(s.runs) - 1; i >= 0 && len(out) < limit; i-- {
		start, end := s.runs[i][0], s.runs[i][1]
		if end.IsZero() {
			end = now
		}
		last := int(end.Sub(start) / freq)
		for k := last; k >= 0 && len(out) < limit; k-- {
			tick := start.Add(time.Duration(k) * freq)
			out = append(out, lkeExecution{tick: tick, dataStart: tick.Add(-freq - delay), dataEnd: tick.Add(-delay)})
		}
	}
	return out
}

// lkeResult is the placeholder verdict of one execution: a stable hash of
// the scheduler and tick flags about one interval in five as anomalous.
func lkeResult(s *lkeEnt, tick time.Time) string {
	if amlScore(s.arn+"|"+tick.Format(time.RFC3339)) < 0.2 {
		return "ANOMALOUS"
	}
	return "NORMAL"
}

func (st *lkeStore) listExecutions(in map[string]any) (map[string]any, *fcErr) {
	s, err := st.scheduler(fcStr(in, "InferenceSchedulerName"))
	if err != nil {
		return nil, err
	}
	after, before := amlTime(in["DataStartTimeAfter"]), amlTime(in["DataEndTimeBefore"])
	if status := fcStr(in, "Status"); status != "" && status != "SUCCESS" {
		return map[string]any{"InferenceExecutionSummaries": []any{}}, nil
	}
	m := s.model
	out := fcObj(fcObj(s.doc, "DataOutputConfiguration"), "S3OutputConfiguration")
	var items []map[string]any
	for _, e := range st.executions(s, lkeMaxExecutions) {
		if (after > 0 && fcEpoch(e.dataStart) < after) || (before > 0 && fcEpoch(e.dataEnd) > before) {
			continue
		}
		item := lkePick(s.doc, "ModelName", "ModelArn", "InferenceSchedulerName", "InferenceSchedulerArn", "DataInputConfiguration", "DataOutputConfiguration")
		item["ScheduledStartTime"], item["DataStartTime"], item["DataEndTime"] = fcEpoch(e.tick), fcEpoch(e.dataStart), fcEpoch(e.dataEnd)
		item["Status"], item["ModelVersion"], item["ModelVersionArn"] = "SUCCESS", m.doc["ActiveModelVersion"], m.doc["ActiveModelVersionArn"]
		item["CustomerResultObject"] = map[string]any{"Bucket": fcStr(out, "Bucket"),
			"Key": fcStr(out, "Prefix") + fcStr(s.doc, "InferenceSchedulerName") + "/" + e.tick.Format("2006-01-02-15-04-05") + "/results.jsonl"}
		item["_key"] = fmt.Sprintf("%012d", int64(1e11)-e.tick.Unix()) // newest first
		items = append(items, item)
	}
	return lkePage(items, in, "_key", "InferenceExecutionSummaries")
}

func (st *lkeStore) listEvents(in map[string]any) (map[string]any, *fcErr) {
	s, err := st.scheduler(fcStr(in, "InferenceSchedulerName"))
	if err != nil {
		return nil, err
	}
	from, to := amlTime(in["IntervalStartTime"]), amlTime(in["IntervalEndTime"])
	if to <= from {
		return nil, lkeInvalid("IntervalEndTime must be after IntervalStartTime")
	}
	diag := []any{}
	if d := st.datasets[fcStr(s.model.doc, "DatasetName")]; d != nil {
		for i, sn := range d.sensors {
			if i >= 5 {
				break
			}
			diag = append(diag, map[string]any{"name": sn.component + "\\" + sn.name, "value": math.Round(amlScore(s.arn+sn.name)*1e4) / 1e4})
		}
	}
	diagnostics, _ := json.Marshal(diag)
	var items []map[string]any
	for _, e := range st.executions(s, lkeMaxExecutions) {
		start, end := fcEpoch(e.dataStart), fcEpoch(e.dataEnd)
		if end < from || start > to || lkeResult(s, e.tick) != "ANOMALOUS" {
			continue
		}
		items = append(items, map[string]any{"InferenceSchedulerArn": s.arn, "InferenceSchedulerName": s.doc["InferenceSchedulerName"],
			"EventStartTime": start, "EventEndTime": end, "Diagnostics": string(diagnostics),
			"EventDurationInSeconds": int64(e.dataEnd.Sub(e.dataStart).Seconds()), "_key": fmt.Sprintf("%012d", int64(start))})
	}
	return lkePage(items, in, "_key", "InferenceEventSummaries")
}
