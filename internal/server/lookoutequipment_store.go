package server

import (
	"encoding/json"
	"strings"
	"sync"
	"time"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

const (
	lkeMaxPerKind = 500      // datasets, label groups, models, schedulers
	lkeMaxLabels  = 5000     // labels across all groups
	lkeMaxJobs    = 2000     // ingestion jobs
	lkeMaxTags    = 200      // per resource
	lkeMaxBytes   = 64 << 20 // stored documents, store-wide
	// ponytail: async work advances one status per read, or settles outright
	// once this long has passed.
	lkeSettle = 3 * time.Second
)

// lkeEnt is one stored resource. doc holds its output fields; the rest is
// bookkeeping callers never see.
type lkeEnt struct {
	kind    string
	arn     string // taggable resources and model versions
	doc     map[string]any
	created time.Time
	size    int
	tags    [][2]string
	sensors []lkeSensor // datasets and ingestion jobs
	failure string      // async work settles FAILED with this reason
	dataset *lkeEnt     // ingestion jobs
	job     *lkeEnt     // datasets: the latest ingestion or import job
	// async transitions: when the current one began and whether a read saw it
	since time.Time
	seen  bool
	model *lkeEnt // versions, schedulers
	// models
	versions  []*lkeEnt
	retrainer *lkeEnt
	policy    map[string]any
	// label groups
	labels map[string]*lkeEnt
	// inference schedulers: when each RUNNING spell started and stopped
	runs [][2]time.Time
}

type lkeStore struct {
	mu         sync.Mutex
	datasets   map[string]*lkeEnt // by name
	jobs       map[string]*lkeEnt // by job ID
	groups     map[string]*lkeEnt
	models     map[string]*lkeEnt
	schedulers map[string]*lkeEnt
	byArn      map[string]*lkeEnt // taggable resources and model versions
	labelCount int
	bytes      int
}

func newLookoutEquipmentStore() *lkeStore {
	return &lkeStore{datasets: map[string]*lkeEnt{}, jobs: map[string]*lkeEnt{}, groups: map[string]*lkeEnt{},
		models: map[string]*lkeEnt{}, schedulers: map[string]*lkeEnt{}, byArn: map[string]*lkeEnt{}}
}

func lkeEntSize(e *lkeEnt) int {
	n := amlSize(e.doc, e.policy) + len(e.sensors)*96
	for _, t := range e.tags {
		n += len(t[0]) + len(t[1])
	}
	return n
}

// charge stores e's new size, refusing growth past the byte cap.
func (st *lkeStore) charge(e *lkeEnt) *fcErr {
	size := lkeEntSize(e)
	if st.bytes-e.size+size > lkeMaxBytes {
		return lkeQuota("Stackyard holds at most %d MiB of Lookout for Equipment resources", lkeMaxBytes>>20)
	}
	st.bytes += size - e.size
	e.size = size
	return nil
}

// replace swaps in a new doc (built from a copy), re-charging its size.
func (st *lkeStore) replace(e *lkeEnt, doc map[string]any) *fcErr {
	old := e.doc
	e.doc = doc
	if err := st.charge(e); err != nil {
		e.doc = old
		return err
	}
	return nil
}

func (st *lkeStore) release(e *lkeEnt) {
	st.bytes -= e.size
	if e.arn != "" && st.byArn[e.arn] == e {
		delete(st.byArn, e.arn)
	}
}

// add registers a new resource with its create-time tags.
func (st *lkeStore) add(e *lkeEnt, tags any) *fcErr {
	e.created = time.Now().UTC()
	e.since = e.created
	if list, _ := tags.([]any); len(list) > 0 {
		e.tags = lkeMergeTags(nil, list)
		if len(e.tags) > lkeMaxTags {
			return lkeQuota("a resource can have at most %d tags", lkeMaxTags)
		}
	}
	if err := st.charge(e); err != nil {
		return err
	}
	if e.arn != "" {
		st.byArn[e.arn] = e
	}
	return nil
}

func lkeMergeTags(tags [][2]string, list []any) [][2]string {
	for _, t := range list {
		tag, _ := t.(map[string]any)
		tags = amlSetTag(tags, fcStr(tag, "Key"), fcStr(tag, "Value"))
	}
	return tags
}

func (st *lkeStore) run(op string, in map[string]any, pre *lkeIngest) (any, *fcErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	out, err := st.dispatch(op, in, pre)
	if err != nil {
		return nil, err
	}
	if out == nil {
		out = map[string]any{}
	}
	m := awsmodels.LookoutEquipmentModel()
	return m.Conform(m.Operations[op].Output, out), nil
}

func (s *Server) lkeRun(op string, in map[string]any) (any, *fcErr) {
	var pre *lkeIngest
	if op == "StartDataIngestionJob" {
		res := s.lkeReadIngestion(fcObj(fcObj(in, "IngestionInputConfiguration"), "S3InputConfiguration"))
		pre = &res
	}
	return s.lookoutEquipment.run(op, in, pre)
}

var lkeHandlers = map[string]func(st *lkeStore, in map[string]any, pre *lkeIngest) (map[string]any, *fcErr){
	"CreateDataset": func(st *lkeStore, in map[string]any, _ *lkeIngest) (map[string]any, *fcErr) {
		return st.createDataset(in)
	},
	"DescribeDataset": func(st *lkeStore, in map[string]any, _ *lkeIngest) (map[string]any, *fcErr) {
		return st.describeDataset(in)
	},
	"DeleteDataset": func(st *lkeStore, in map[string]any, _ *lkeIngest) (map[string]any, *fcErr) {
		return nil, st.deleteDataset(in)
	},
	"ListDatasets": func(st *lkeStore, in map[string]any, _ *lkeIngest) (map[string]any, *fcErr) {
		return st.listDatasets(in)
	},
	"ImportDataset": func(st *lkeStore, in map[string]any, _ *lkeIngest) (map[string]any, *fcErr) {
		return st.importDataset(in)
	},
	"StartDataIngestionJob": func(st *lkeStore, in map[string]any, pre *lkeIngest) (map[string]any, *fcErr) {
		return st.startIngestion(in, pre)
	},
	"DescribeDataIngestionJob": func(st *lkeStore, in map[string]any, _ *lkeIngest) (map[string]any, *fcErr) {
		return st.describeJob(in)
	},
	"ListDataIngestionJobs": func(st *lkeStore, in map[string]any, _ *lkeIngest) (map[string]any, *fcErr) { return st.listJobs(in) },
	"ListSensorStatistics": func(st *lkeStore, in map[string]any, _ *lkeIngest) (map[string]any, *fcErr) {
		return st.sensorStatistics(in)
	},
	"CreateLabelGroup": func(st *lkeStore, in map[string]any, _ *lkeIngest) (map[string]any, *fcErr) {
		return st.createLabelGroup(in)
	},
	"DescribeLabelGroup": func(st *lkeStore, in map[string]any, _ *lkeIngest) (map[string]any, *fcErr) {
		return st.describeLabelGroup(in)
	},
	"UpdateLabelGroup": func(st *lkeStore, in map[string]any, _ *lkeIngest) (map[string]any, *fcErr) {
		return nil, st.updateLabelGroup(in)
	},
	"DeleteLabelGroup": func(st *lkeStore, in map[string]any, _ *lkeIngest) (map[string]any, *fcErr) {
		return nil, st.deleteLabelGroup(in)
	},
	"ListLabelGroups": func(st *lkeStore, in map[string]any, _ *lkeIngest) (map[string]any, *fcErr) {
		return st.listLabelGroups(in)
	},
	"CreateLabel": func(st *lkeStore, in map[string]any, _ *lkeIngest) (map[string]any, *fcErr) {
		return st.createLabel(in)
	},
	"DescribeLabel": func(st *lkeStore, in map[string]any, _ *lkeIngest) (map[string]any, *fcErr) {
		return st.describeLabel(in)
	},
	"DeleteLabel": func(st *lkeStore, in map[string]any, _ *lkeIngest) (map[string]any, *fcErr) {
		return nil, st.deleteLabel(in)
	},
	"ListLabels": func(st *lkeStore, in map[string]any, _ *lkeIngest) (map[string]any, *fcErr) { return st.listLabels(in) },
	"CreateModel": func(st *lkeStore, in map[string]any, _ *lkeIngest) (map[string]any, *fcErr) {
		return st.createModel(in)
	},
	"DescribeModel": func(st *lkeStore, in map[string]any, _ *lkeIngest) (map[string]any, *fcErr) {
		return st.describeModel(in)
	},
	"UpdateModel": func(st *lkeStore, in map[string]any, _ *lkeIngest) (map[string]any, *fcErr) {
		return nil, st.updateModel(in)
	},
	"DeleteModel": func(st *lkeStore, in map[string]any, _ *lkeIngest) (map[string]any, *fcErr) {
		return nil, st.deleteModel(in)
	},
	"ListModels": func(st *lkeStore, in map[string]any, _ *lkeIngest) (map[string]any, *fcErr) { return st.listModels(in) },
	"DescribeModelVersion": func(st *lkeStore, in map[string]any, _ *lkeIngest) (map[string]any, *fcErr) {
		return st.describeVersion(in)
	},
	"ListModelVersions": func(st *lkeStore, in map[string]any, _ *lkeIngest) (map[string]any, *fcErr) {
		return st.listVersions(in)
	},
	"ImportModelVersion": func(st *lkeStore, in map[string]any, _ *lkeIngest) (map[string]any, *fcErr) {
		return st.importVersion(in)
	},
	"UpdateActiveModelVersion": func(st *lkeStore, in map[string]any, _ *lkeIngest) (map[string]any, *fcErr) {
		return st.activateVersion(in)
	},
	"CreateRetrainingScheduler": func(st *lkeStore, in map[string]any, _ *lkeIngest) (map[string]any, *fcErr) {
		return st.createRetrainer(in)
	},
	"DescribeRetrainingScheduler": func(st *lkeStore, in map[string]any, _ *lkeIngest) (map[string]any, *fcErr) {
		return st.describeRetrainer(in)
	},
	"UpdateRetrainingScheduler": func(st *lkeStore, in map[string]any, _ *lkeIngest) (map[string]any, *fcErr) {
		return nil, st.updateRetrainer(in)
	},
	"DeleteRetrainingScheduler": func(st *lkeStore, in map[string]any, _ *lkeIngest) (map[string]any, *fcErr) {
		return nil, st.deleteRetrainer(in)
	},
	"StartRetrainingScheduler": func(st *lkeStore, in map[string]any, _ *lkeIngest) (map[string]any, *fcErr) {
		return st.switchRetrainer(in, true)
	},
	"StopRetrainingScheduler": func(st *lkeStore, in map[string]any, _ *lkeIngest) (map[string]any, *fcErr) {
		return st.switchRetrainer(in, false)
	},
	"ListRetrainingSchedulers": func(st *lkeStore, in map[string]any, _ *lkeIngest) (map[string]any, *fcErr) {
		return st.listRetrainers(in)
	},
	"CreateInferenceScheduler": func(st *lkeStore, in map[string]any, _ *lkeIngest) (map[string]any, *fcErr) {
		return st.createScheduler(in)
	},
	"DescribeInferenceScheduler": func(st *lkeStore, in map[string]any, _ *lkeIngest) (map[string]any, *fcErr) {
		return st.describeScheduler(in)
	},
	"UpdateInferenceScheduler": func(st *lkeStore, in map[string]any, _ *lkeIngest) (map[string]any, *fcErr) {
		return nil, st.updateScheduler(in)
	},
	"DeleteInferenceScheduler": func(st *lkeStore, in map[string]any, _ *lkeIngest) (map[string]any, *fcErr) {
		return nil, st.deleteScheduler(in)
	},
	"StartInferenceScheduler": func(st *lkeStore, in map[string]any, _ *lkeIngest) (map[string]any, *fcErr) {
		return st.switchScheduler(in, true)
	},
	"StopInferenceScheduler": func(st *lkeStore, in map[string]any, _ *lkeIngest) (map[string]any, *fcErr) {
		return st.switchScheduler(in, false)
	},
	"ListInferenceSchedulers": func(st *lkeStore, in map[string]any, _ *lkeIngest) (map[string]any, *fcErr) {
		return st.listSchedulers(in)
	},
	"ListInferenceExecutions": func(st *lkeStore, in map[string]any, _ *lkeIngest) (map[string]any, *fcErr) {
		return st.listExecutions(in)
	},
	"ListInferenceEvents": func(st *lkeStore, in map[string]any, _ *lkeIngest) (map[string]any, *fcErr) { return st.listEvents(in) },
	"PutResourcePolicy":   func(st *lkeStore, in map[string]any, _ *lkeIngest) (map[string]any, *fcErr) { return st.putPolicy(in) },
	"DescribeResourcePolicy": func(st *lkeStore, in map[string]any, _ *lkeIngest) (map[string]any, *fcErr) {
		return st.describePolicy(in)
	},
	"DeleteResourcePolicy": func(st *lkeStore, in map[string]any, _ *lkeIngest) (map[string]any, *fcErr) {
		return nil, st.deletePolicy(in)
	},
	"TagResource":         func(st *lkeStore, in map[string]any, _ *lkeIngest) (map[string]any, *fcErr) { return nil, st.tag(in) },
	"UntagResource":       func(st *lkeStore, in map[string]any, _ *lkeIngest) (map[string]any, *fcErr) { return nil, st.untag(in) },
	"ListTagsForResource": func(st *lkeStore, in map[string]any, _ *lkeIngest) (map[string]any, *fcErr) { return st.listTags(in) },
}

func (st *lkeStore) dispatch(op string, in map[string]any, pre *lkeIngest) (map[string]any, *fcErr) {
	h := lkeHandlers[op]
	if h == nil {
		return nil, fcError("UnknownOperationException", "unknown operation %s", op)
	}
	return h(st, in, pre)
}

func lkeNow() float64 { return fcEpoch(time.Now().UTC()) }

// settled reports whether e's current async transition should finish now:
// on the read after the one that first saw it, or once lkeSettle has passed.
func lkeSettled(e *lkeEnt) bool {
	if time.Since(e.since) > lkeSettle || e.seen {
		return true
	}
	e.seen = true
	return false
}

// begin starts a new async transition on e.
func lkeBegin(e *lkeEnt) {
	e.since, e.seen = time.Now().UTC(), false
}

// ---- datasets and ingestion ----

func (st *lkeStore) dataset(name string) (*lkeEnt, *fcErr) {
	d := st.datasets[name]
	if d == nil {
		return nil, fcNotFound("Dataset %s does not exist", name)
	}
	if d.job != nil {
		st.advanceJob(d.job)
	}
	return d, nil
}

func lkeJobBusy(j *lkeEnt) bool {
	s := j.doc["Status"]
	return s == "IN_PROGRESS" || s == "IMPORT_IN_PROGRESS"
}

func (st *lkeStore) createDataset(in map[string]any) (map[string]any, *fcErr) {
	name := fcStr(in, "DatasetName")
	if st.datasets[name] != nil {
		return nil, lkeConflict("Dataset %s already exists", name)
	}
	if len(st.datasets) >= lkeMaxPerKind {
		return nil, lkeQuota("at most %d datasets", lkeMaxPerKind)
	}
	schema := fcStr(fcObj(in, "DatasetSchema"), "InlineDataSchema")
	if schema != "" && !json.Valid([]byte(schema)) {
		return nil, lkeInvalid("InlineDataSchema is not valid JSON")
	}
	now := lkeNow()
	doc := map[string]any{"DatasetName": name, "DatasetArn": lkeArn("dataset", name), "Status": "CREATED",
		"CreatedAt": now, "LastUpdatedAt": now}
	if schema != "" {
		doc["Schema"] = schema
	}
	if v := fcStr(in, "ServerSideKmsKeyId"); v != "" {
		doc["ServerSideKmsKeyId"] = v
	}
	d := &lkeEnt{kind: "dataset", arn: fcStr(doc, "DatasetArn"), doc: doc}
	if err := st.add(d, in["Tags"]); err != nil {
		return nil, err
	}
	st.datasets[name] = d
	return lkePick(doc, "DatasetName", "DatasetArn", "Status"), nil
}

func (st *lkeStore) describeDataset(in map[string]any) (map[string]any, *fcErr) {
	d, err := st.dataset(fcStr(in, "DatasetName"))
	if err != nil {
		return nil, err
	}
	return d.doc, nil
}

func (st *lkeStore) deleteDataset(in map[string]any) *fcErr {
	d, err := st.dataset(fcStr(in, "DatasetName"))
	if err != nil {
		return err
	}
	if j := d.job; j != nil && lkeJobBusy(j) {
		return lkeConflict("Dataset %s has an ingestion job in progress", fcStr(d.doc, "DatasetName"))
	}
	for _, m := range st.models {
		if m.doc["DatasetArn"] == d.doc["DatasetArn"] && m.doc["Status"] == "IN_PROGRESS" {
			return lkeConflict("Model %s is training on dataset %s", fcStr(m.doc, "ModelName"), fcStr(d.doc, "DatasetName"))
		}
	}
	st.release(d)
	delete(st.datasets, fcStr(d.doc, "DatasetName"))
	return nil
}

func (st *lkeStore) listDatasets(in map[string]any) (map[string]any, *fcErr) {
	prefix := fcStr(in, "DatasetNameBeginsWith")
	var items []map[string]any
	for name := range st.datasets {
		if strings.HasPrefix(name, prefix) {
			d, _ := st.dataset(name)
			items = append(items, lkePick(d.doc, "DatasetName", "DatasetArn", "Status", "CreatedAt"))
		}
	}
	return lkePage(items, in, "DatasetName", "DatasetSummaries")
}

func (st *lkeStore) newJob(d *lkeEnt, doc map[string]any) (*lkeEnt, *fcErr) {
	if len(st.jobs) >= lkeMaxJobs {
		// Drop the oldest finished job to make room.
		var oldest *lkeEnt
		for _, j := range st.jobs {
			if !lkeJobBusy(j) && (oldest == nil || j.created.Before(oldest.created)) {
				oldest = j
			}
		}
		if oldest == nil {
			return nil, lkeQuota("at most %d ingestion jobs", lkeMaxJobs)
		}
		st.release(oldest)
		delete(st.jobs, fcStr(oldest.doc, "JobId"))
	}
	id := lkeHex(16)
	doc["JobId"], doc["DatasetArn"], doc["CreatedAt"] = id, d.doc["DatasetArn"], lkeNow()
	j := &lkeEnt{kind: "job", doc: doc, dataset: d}
	if err := st.add(j, nil); err != nil {
		return nil, err
	}
	st.jobs[id] = j
	d.job = j
	return j, nil
}

func (st *lkeStore) startIngestion(in map[string]any, pre *lkeIngest) (map[string]any, *fcErr) {
	d, err := st.dataset(fcStr(in, "DatasetName"))
	if err != nil {
		return nil, err
	}
	if s := d.doc["Status"]; s == "INGESTION_IN_PROGRESS" || s == "IMPORT_IN_PROGRESS" {
		return nil, lkeConflict("Dataset %s is %s", fcStr(d.doc, "DatasetName"), s)
	}
	cfg := fcObj(in, "IngestionInputConfiguration")
	j, err := st.newJob(d, map[string]any{"IngestionInputConfiguration": cfg, "RoleArn": fcStr(in, "RoleArn"), "Status": "IN_PROGRESS"})
	if err != nil {
		return nil, err
	}
	if pre != nil {
		j.failure, j.sensors = pre.failure, pre.sensors
		if pre.failure == "" {
			j.doc["IngestedFilesSummary"] = map[string]any{"TotalNumberOfFiles": pre.files, "IngestedNumberOfFiles": pre.files, "DiscardedFiles": []any{}}
			j.doc["IngestedDataSize"], j.doc["DataQualitySummary"] = pre.size, lkeQualitySummary(pre)
			j.doc["DataStartTime"], j.doc["DataEndTime"] = pre.start, pre.end
		}
		if err := st.charge(j); err != nil {
			st.release(j)
			delete(st.jobs, fcStr(j.doc, "JobId"))
			return nil, err
		}
	}
	d.doc["Status"], d.doc["LastUpdatedAt"] = "INGESTION_IN_PROGRESS", lkeNow()
	return lkePick(j.doc, "JobId", "Status"), nil
}

// advanceJob finishes an ingestion or import job and updates its dataset.
func (st *lkeStore) advanceJob(j *lkeEnt) {
	if !lkeJobBusy(j) || !lkeSettled(j) {
		return
	}
	d := j.dataset
	if j.failure != "" {
		j.doc["Status"], j.doc["FailedReason"] = "FAILED", j.failure
		if st.datasets[fcStr(d.doc, "DatasetName")] == d && d.doc["Status"] != "IMPORT_IN_PROGRESS" {
			status := "CREATED"
			if d.doc["IngestionInputConfiguration"] != nil {
				status = "ACTIVE" // keeps its earlier data
			}
			d.doc["Status"] = status
		}
		return
	}
	j.doc["Status"] = "SUCCESS"
	if st.datasets[fcStr(d.doc, "DatasetName")] != d {
		return // the dataset was deleted meanwhile
	}
	for _, k := range []string{"IngestionInputConfiguration", "DataQualitySummary", "IngestedFilesSummary", "RoleArn", "DataStartTime", "DataEndTime"} {
		if v, ok := j.doc[k]; ok {
			d.doc[k] = v
		}
	}
	d.sensors = j.sensors
	d.doc["Status"], d.doc["LastUpdatedAt"] = "ACTIVE", lkeNow()
	_ = st.charge(d) // the job's sensors were already charged once
}

func (st *lkeStore) describeJob(in map[string]any) (map[string]any, *fcErr) {
	j := st.jobs[fcStr(in, "JobId")]
	if j == nil {
		return nil, fcNotFound("Ingestion job %s does not exist", fcStr(in, "JobId"))
	}
	st.advanceJob(j)
	return j.doc, nil
}

func (st *lkeStore) listJobs(in map[string]any) (map[string]any, *fcErr) {
	name, status := fcStr(in, "DatasetName"), fcStr(in, "Status")
	var items []map[string]any
	for _, j := range st.jobs {
		st.advanceJob(j)
		dsName := fcStr(j.dataset.doc, "DatasetName")
		if (name == "" || name == dsName) && (status == "" || status == j.doc["Status"]) {
			item := lkePick(j.doc, "JobId", "DatasetArn", "IngestionInputConfiguration", "Status")
			item["DatasetName"] = dsName
			items = append(items, item)
		}
	}
	return lkePage(items, in, "JobId", "DataIngestionJobSummaries")
}

func (st *lkeStore) sensorStatistics(in map[string]any) (map[string]any, *fcErr) {
	d, err := st.dataset(fcStr(in, "DatasetName"))
	if err != nil {
		return nil, err
	}
	sensors := d.sensors
	if id := fcStr(in, "IngestionJobId"); id != "" {
		j := st.jobs[id]
		if j == nil || j.dataset != d {
			return nil, fcNotFound("Ingestion job %s does not exist for dataset %s", id, fcStr(d.doc, "DatasetName"))
		}
		st.advanceJob(j)
		sensors = j.sensors
	}
	items := make([]map[string]any, 0, len(sensors))
	for _, sn := range sensors {
		item := lkeSensorSummary(sn)
		item["_key"] = sn.component + "/" + sn.name
		items = append(items, item)
	}
	return lkePage(items, in, "_key", "SensorStatisticsSummaries")
}

func (st *lkeStore) importDataset(in map[string]any) (map[string]any, *fcErr) {
	src := st.byArn[fcStr(in, "SourceDatasetArn")]
	if src == nil || src.kind != "dataset" {
		return nil, fcNotFound("Dataset %s does not exist", fcStr(in, "SourceDatasetArn"))
	}
	if src.doc["Status"] != "ACTIVE" {
		return nil, lkeConflict("Dataset %s has no ingested data to import", fcStr(src.doc, "DatasetName"))
	}
	name := fcStr(in, "DatasetName")
	if name == "" {
		name = fcStr(src.doc, "DatasetName")
	}
	if st.datasets[name] != nil {
		return nil, lkeConflict("Dataset %s already exists", name)
	}
	if len(st.datasets) >= lkeMaxPerKind {
		return nil, lkeQuota("at most %d datasets", lkeMaxPerKind)
	}
	now := lkeNow()
	doc := map[string]any{"DatasetName": name, "DatasetArn": lkeArn("dataset", name), "Status": "IMPORT_IN_PROGRESS",
		"CreatedAt": now, "LastUpdatedAt": now, "SourceDatasetArn": src.doc["DatasetArn"]}
	if v, ok := src.doc["Schema"]; ok {
		doc["Schema"] = v
	}
	d := &lkeEnt{kind: "dataset", arn: fcStr(doc, "DatasetArn"), doc: doc}
	if err := st.add(d, in["Tags"]); err != nil {
		return nil, err
	}
	st.datasets[name] = d
	job := lkePick(src.doc, "IngestionInputConfiguration", "DataQualitySummary", "IngestedFilesSummary", "DataStartTime", "DataEndTime")
	job["Status"], job["SourceDatasetArn"] = "IMPORT_IN_PROGRESS", src.doc["DatasetArn"]
	j, err := st.newJob(d, job)
	if err != nil {
		st.release(d)
		delete(st.datasets, name)
		return nil, err
	}
	j.sensors = src.sensors
	out := lkePick(doc, "DatasetName", "DatasetArn", "Status")
	out["JobId"] = j.doc["JobId"]
	return out, nil
}

// ---- label groups and labels ----

func (st *lkeStore) group(name string) (*lkeEnt, *fcErr) {
	g := st.groups[name]
	if g == nil {
		return nil, fcNotFound("Label group %s does not exist", name)
	}
	return g, nil
}

func (st *lkeStore) createLabelGroup(in map[string]any) (map[string]any, *fcErr) {
	name := fcStr(in, "LabelGroupName")
	if st.groups[name] != nil {
		return nil, lkeConflict("Label group %s already exists", name)
	}
	if len(st.groups) >= lkeMaxPerKind {
		return nil, lkeQuota("at most %d label groups", lkeMaxPerKind)
	}
	now := lkeNow()
	doc := map[string]any{"LabelGroupName": name, "LabelGroupArn": lkeArn("label-group", name), "CreatedAt": now, "UpdatedAt": now,
		"FaultCodes": fcAnyList(fcStrs(in, "FaultCodes"))}
	g := &lkeEnt{kind: "labelGroup", arn: fcStr(doc, "LabelGroupArn"), doc: doc, labels: map[string]*lkeEnt{}}
	if err := st.add(g, in["Tags"]); err != nil {
		return nil, err
	}
	st.groups[name] = g
	return lkePick(doc, "LabelGroupName", "LabelGroupArn"), nil
}

func (st *lkeStore) describeLabelGroup(in map[string]any) (map[string]any, *fcErr) {
	g, err := st.group(fcStr(in, "LabelGroupName"))
	if err != nil {
		return nil, err
	}
	return g.doc, nil
}

func (st *lkeStore) updateLabelGroup(in map[string]any) *fcErr {
	g, err := st.group(fcStr(in, "LabelGroupName"))
	if err != nil {
		return err
	}
	doc := lkeCopy(g.doc)
	if _, ok := in["FaultCodes"]; ok {
		doc["FaultCodes"] = fcAnyList(fcStrs(in, "FaultCodes"))
	}
	doc["UpdatedAt"] = lkeNow()
	return st.replace(g, doc)
}

func (st *lkeStore) deleteLabelGroup(in map[string]any) *fcErr {
	g, err := st.group(fcStr(in, "LabelGroupName"))
	if err != nil {
		return err
	}
	for _, l := range g.labels {
		st.bytes -= l.size
	}
	st.labelCount -= len(g.labels)
	st.release(g)
	delete(st.groups, fcStr(g.doc, "LabelGroupName"))
	return nil
}

func (st *lkeStore) listLabelGroups(in map[string]any) (map[string]any, *fcErr) {
	prefix := fcStr(in, "LabelGroupNameBeginsWith")
	var items []map[string]any
	for name, g := range st.groups {
		if strings.HasPrefix(name, prefix) {
			items = append(items, lkePick(g.doc, "LabelGroupName", "LabelGroupArn", "CreatedAt", "UpdatedAt"))
		}
	}
	return lkePage(items, in, "LabelGroupName", "LabelGroupSummaries")
}

func (st *lkeStore) createLabel(in map[string]any) (map[string]any, *fcErr) {
	g, err := st.group(fcStr(in, "LabelGroupName"))
	if err != nil {
		return nil, err
	}
	start, end := amlTime(in["StartTime"]), amlTime(in["EndTime"])
	if start <= 0 || end <= start {
		return nil, lkeInvalid("EndTime must be after StartTime")
	}
	if code := fcStr(in, "FaultCode"); code != "" {
		codes, _ := g.doc["FaultCodes"].([]any)
		if len(codes) > 0 && !fcContainsAny(codes, code) {
			return nil, lkeInvalid("FaultCode %s is not one of the label group's fault codes", code)
		}
	}
	if st.labelCount >= lkeMaxLabels {
		return nil, lkeQuota("at most %d labels", lkeMaxLabels)
	}
	id := lkeHex(16)
	doc := lkePick(in, "Rating", "FaultCode", "Notes", "Equipment")
	doc["LabelGroupName"], doc["LabelGroupArn"], doc["LabelId"] = g.doc["LabelGroupName"], g.doc["LabelGroupArn"], id
	doc["StartTime"], doc["EndTime"], doc["CreatedAt"] = start, end, lkeNow()
	l := &lkeEnt{kind: "label", doc: doc}
	if err := st.add(l, nil); err != nil {
		return nil, err
	}
	g.labels[id] = l
	st.labelCount++
	return map[string]any{"LabelId": id}, nil
}

func fcContainsAny(items []any, v string) bool {
	for _, item := range items {
		if item == v {
			return true
		}
	}
	return false
}

func (st *lkeStore) label(in map[string]any) (*lkeEnt, *lkeEnt, *fcErr) {
	g, err := st.group(fcStr(in, "LabelGroupName"))
	if err != nil {
		return nil, nil, err
	}
	l := g.labels[fcStr(in, "LabelId")]
	if l == nil {
		return nil, nil, fcNotFound("Label %s does not exist", fcStr(in, "LabelId"))
	}
	return g, l, nil
}

func (st *lkeStore) describeLabel(in map[string]any) (map[string]any, *fcErr) {
	_, l, err := st.label(in)
	if err != nil {
		return nil, err
	}
	return l.doc, nil
}

func (st *lkeStore) deleteLabel(in map[string]any) *fcErr {
	g, l, err := st.label(in)
	if err != nil {
		return err
	}
	st.bytes -= l.size
	st.labelCount--
	delete(g.labels, fcStr(l.doc, "LabelId"))
	return nil
}

func (st *lkeStore) listLabels(in map[string]any) (map[string]any, *fcErr) {
	g, err := st.group(fcStr(in, "LabelGroupName"))
	if err != nil {
		return nil, err
	}
	from, to := amlTime(in["IntervalStartTime"]), amlTime(in["IntervalEndTime"])
	code, equipment := fcStr(in, "FaultCode"), fcStr(in, "Equipment")
	var items []map[string]any
	for _, l := range g.labels {
		start, end := amlTime(l.doc["StartTime"]), amlTime(l.doc["EndTime"])
		if (from > 0 && end < from) || (to > 0 && start > to) || (code != "" && l.doc["FaultCode"] != code) ||
			(equipment != "" && l.doc["Equipment"] != equipment) {
			continue
		}
		items = append(items, lkePick(l.doc, "LabelGroupName", "LabelId", "LabelGroupArn", "StartTime", "EndTime", "Rating", "FaultCode", "Equipment", "CreatedAt"))
	}
	return lkePage(items, in, "LabelId", "LabelSummaries")
}

// ---- tags and resource policies ----

func (st *lkeStore) taggable(arn string) (*lkeEnt, *fcErr) {
	e := st.byArn[arn]
	if e == nil || e.kind == "version" {
		return nil, fcNotFound("Resource %s does not exist", arn)
	}
	return e, nil
}

func (st *lkeStore) tag(in map[string]any) *fcErr {
	e, err := st.taggable(fcStr(in, "ResourceArn"))
	if err != nil {
		return err
	}
	list, _ := in["Tags"].([]any)
	tags := lkeMergeTags(e.tags, list)
	if len(tags) > lkeMaxTags {
		return lkeQuota("a resource can have at most %d tags", lkeMaxTags)
	}
	old := e.tags
	e.tags = tags
	if err := st.charge(e); err != nil {
		e.tags = old
		return err
	}
	return nil
}

func (st *lkeStore) untag(in map[string]any) *fcErr {
	e, err := st.taggable(fcStr(in, "ResourceArn"))
	if err != nil {
		return err
	}
	drop := map[string]bool{}
	for _, k := range fcStrs(in, "TagKeys") {
		drop[k] = true
	}
	var tags [][2]string
	for _, t := range e.tags {
		if !drop[t[0]] {
			tags = append(tags, t)
		}
	}
	e.tags = tags
	return st.charge(e)
}

func (st *lkeStore) listTags(in map[string]any) (map[string]any, *fcErr) {
	e, err := st.taggable(fcStr(in, "ResourceArn"))
	if err != nil {
		return nil, err
	}
	list := make([]any, 0, len(e.tags))
	for _, t := range e.tags {
		list = append(list, map[string]any{"Key": t[0], "Value": t[1]})
	}
	return map[string]any{"Tags": list}, nil
}

// policyTarget is the model or model version a resource policy lives on.
func (st *lkeStore) policyTarget(arn string) (*lkeEnt, *fcErr) {
	e := st.byArn[arn]
	if e == nil || (e.kind != "model" && e.kind != "version") {
		return nil, fcNotFound("Model or model version %s does not exist", arn)
	}
	return e, nil
}

func (st *lkeStore) putPolicy(in map[string]any) (map[string]any, *fcErr) {
	e, err := st.policyTarget(fcStr(in, "ResourceArn"))
	if err != nil {
		return nil, err
	}
	doc := fcStr(in, "ResourcePolicy")
	if !json.Valid([]byte(doc)) {
		return nil, lkeInvalid("ResourcePolicy is not valid JSON")
	}
	rev, hasRev := in["PolicyRevisionId"].(string)
	switch {
	case e.policy != nil && hasRev && rev != e.policy["PolicyRevisionId"]:
		return nil, lkeConflict("PolicyRevisionId %s is not the current revision", rev)
	case e.policy == nil && hasRev:
		return nil, lkeConflict("Resource %s has no policy revision %s", fcStr(in, "ResourceArn"), rev)
	}
	now := lkeNow()
	policy := map[string]any{"ResourcePolicy": doc, "PolicyRevisionId": lkeHex(16), "CreationTime": now, "LastModifiedTime": now}
	if e.policy != nil {
		policy["CreationTime"] = e.policy["CreationTime"]
	}
	old := e.policy
	e.policy = policy
	if err := st.charge(e); err != nil {
		e.policy = old
		return nil, err
	}
	return map[string]any{"ResourceArn": fcStr(in, "ResourceArn"), "PolicyRevisionId": policy["PolicyRevisionId"]}, nil
}

func (st *lkeStore) describePolicy(in map[string]any) (map[string]any, *fcErr) {
	e, err := st.policyTarget(fcStr(in, "ResourceArn"))
	if err != nil {
		return nil, err
	}
	if e.policy == nil {
		return nil, fcNotFound("Resource %s has no resource policy", fcStr(in, "ResourceArn"))
	}
	return e.policy, nil
}

func (st *lkeStore) deletePolicy(in map[string]any) *fcErr {
	e, err := st.policyTarget(fcStr(in, "ResourceArn"))
	if err != nil {
		return err
	}
	if e.policy == nil {
		return fcNotFound("Resource %s has no resource policy", fcStr(in, "ResourceArn"))
	}
	e.policy = nil
	return st.charge(e)
}
