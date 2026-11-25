package server

import (
	"encoding/json"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

const (
	amlMaxPerKind = 1000     // resources of one kind, DELETED ones included
	amlMaxBytes   = 32 << 20 // stored documents, store-wide
	amlMaxTags    = 10       // Amazon ML's per-resource tag limit
	amlIAMUser    = "arn:aws:iam::" + fcAccount + ":user/stackyard"
	// ponytail: resources move PENDING -> INPROGRESS -> COMPLETED one step per
	// read, or settle outright once this long has passed.
	amlSettle = 3 * time.Second
)

var amlIDMember = map[string]string{
	"DataSource": "DataSourceId", "MLModel": "MLModelId", "Evaluation": "EvaluationId", "BatchPrediction": "BatchPredictionId",
}

// amlFilterFields maps each kind's FilterVariable values to record fields.
var amlFilterFields = map[string]map[string]string{
	"DataSource": {"CreatedAt": "CreatedAt", "LastUpdatedAt": "LastUpdatedAt", "Status": "Status", "Name": "Name",
		"DataLocationS3": "DataLocationS3", "IAMUser": "CreatedByIamUser"},
	"MLModel": {"CreatedAt": "CreatedAt", "LastUpdatedAt": "LastUpdatedAt", "Status": "Status", "Name": "Name",
		"IAMUser": "CreatedByIamUser", "TrainingDataSourceId": "TrainingDataSourceId", "RealtimeEndpointStatus": "EndpointStatus",
		"MLModelType": "MLModelType", "Algorithm": "Algorithm", "TrainingDataURI": "InputDataLocationS3"},
	"Evaluation": {"CreatedAt": "CreatedAt", "LastUpdatedAt": "LastUpdatedAt", "Status": "Status", "Name": "Name",
		"IAMUser": "CreatedByIamUser", "MLModelId": "MLModelId", "DataSourceId": "EvaluationDataSourceId", "DataURI": "InputDataLocationS3"},
	"BatchPrediction": {"CreatedAt": "CreatedAt", "LastUpdatedAt": "LastUpdatedAt", "Status": "Status", "Name": "Name",
		"IAMUser": "CreatedByIamUser", "MLModelId": "MLModelId", "DataSourceId": "BatchPredictionDataSourceId", "DataURI": "InputDataLocationS3"},
}

type amlRes struct {
	kind, id       string
	create         map[string]any // conformed create input, for idempotent retries
	rec            map[string]any // output fields
	dsID, modelID  string         // resources this one reads
	failure        string         // settles FAILED with this message
	header         bool           // data sources: files start with a header line
	threshold      float64        // models
	created        time.Time
	tags           [][2]string
	size           int
	endpointSeenAt time.Time // models: when the endpoint went UPDATING
}

type machineLearningStore struct {
	mu    sync.Mutex
	res   map[string]map[string]*amlRes
	bytes int
}

func newMachineLearningStore() *machineLearningStore {
	st := &machineLearningStore{res: map[string]map[string]*amlRes{}}
	for k := range amlIDMember {
		st.res[k] = map[string]*amlRes{}
	}
	return st
}

func amlNotFound(kind, id string) *fcErr {
	return fcNotFound("%s %s does not exist", kind, id)
}

// run executes op under the lock and conforms its output there. write
// reports that a new batch prediction was created, so its output may go to S3.
func (st *machineLearningStore) run(op string, in map[string]any, base string, pre *amlPre) (any, bool, *fcErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	out, write, err := st.dispatch(op, in, base, pre)
	if err != nil {
		return nil, false, err
	}
	m := awsmodels.MachineLearningModel()
	return m.Conform(m.Operations[op].Output, out), write, nil
}

func (st *machineLearningStore) dispatch(op string, in map[string]any, base string, pre *amlPre) (map[string]any, bool, *fcErr) {
	switch op {
	case "CreateDataSourceFromS3", "CreateDataSourceFromRDS", "CreateDataSourceFromRedshift":
		out, err := st.createDataSource(op, in, pre)
		return out, false, err
	case "CreateMLModel":
		out, err := st.createModel(in)
		return out, false, err
	case "CreateEvaluation", "CreateBatchPrediction":
		return st.createJob(strings.TrimPrefix(op, "Create"), in, pre)
	case "CreateRealtimeEndpoint", "DeleteRealtimeEndpoint":
		out, err := st.realtimeEndpoint(op == "CreateRealtimeEndpoint", fcStr(in, "MLModelId"), base)
		return out, false, err
	case "Predict":
		out, err := st.predict(in)
		return out, false, err
	case "AddTags", "DeleteTags", "DescribeTags":
		out, err := st.tagOp(op, in)
		return out, false, err
	}
	for _, verb := range []string{"Get", "Delete", "Update", "Describe"} {
		kind, ok := strings.CutPrefix(op, verb)
		if !ok {
			continue
		}
		if verb == "Describe" {
			kind = strings.TrimSuffix(kind, "s")
		}
		if _, known := amlIDMember[kind]; !known {
			continue
		}
		var out map[string]any
		var err *fcErr
		switch verb {
		case "Get":
			out, err = st.getOut(kind, in)
		case "Delete":
			out, err = st.markDeleted(kind, fcStr(in, amlIDMember[kind]))
		case "Update":
			out, err = st.update(kind, in)
		default:
			out, err = st.describe(kind, in)
		}
		return out, false, err
	}
	return nil, false, fcError("UnknownOperationException", "unknown operation %s", op)
}

// get returns a live (not DELETED) resource, advancing its status.
func (st *machineLearningStore) get(kind, id string) (*amlRes, *fcErr) {
	r := st.res[kind][id]
	if r == nil || r.rec["Status"] == "DELETED" {
		return nil, amlNotFound(kind, id)
	}
	st.advance(r)
	return r, nil
}

// advance moves r one status step (or to its final status once amlSettle
// has passed). A resource waits in INPROGRESS until what it reads is done.
func (st *machineLearningStore) advance(r *amlRes) {
	now := time.Now().UTC()
	if ep, _ := r.rec["EndpointInfo"].(map[string]any); ep != nil && ep["EndpointStatus"] == "UPDATING" {
		ep["EndpointStatus"] = "READY"
	}
	settle := now.Sub(r.created) > amlSettle
	for {
		switch r.rec["Status"] {
		case "PENDING":
			r.rec["Status"], r.rec["StartedAt"], r.rec["LastUpdatedAt"] = "INPROGRESS", fcEpoch(now), fcEpoch(now)
		case "INPROGRESS":
			failure, ready := st.depsState(r)
			if !ready {
				return
			}
			if failure == "" {
				failure = r.failure
			}
			status := "COMPLETED"
			if failure != "" {
				status, r.rec["Message"] = "FAILED", failure
			}
			r.rec["Status"], r.rec["FinishedAt"], r.rec["LastUpdatedAt"] = status, fcEpoch(now), fcEpoch(now)
			r.rec["ComputeTime"] = int64(now.Sub(r.created).Milliseconds())
			st.finish(r)
		default:
			return
		}
		if !settle {
			return
		}
	}
}

// depsState reports whether r's inputs are final, and why r fails if one failed.
func (st *machineLearningStore) depsState(r *amlRes) (string, bool) {
	for _, dep := range []struct{ kind, id string }{{"MLModel", r.modelID}, {"DataSource", r.dsID}} {
		if dep.id == "" {
			continue
		}
		d := st.res[dep.kind][dep.id]
		if d == nil || d.rec["Status"] == "DELETED" {
			return dep.kind + " " + dep.id + " was deleted.", true
		}
		if d != r {
			st.advance(d)
		}
		switch d.rec["Status"] {
		case "FAILED":
			return dep.kind + " " + dep.id + " failed.", true
		case "COMPLETED":
		default:
			return "", false
		}
	}
	return "", true
}

// finish fills in what a completed resource reports.
func (st *machineLearningStore) finish(r *amlRes) {
	if r.rec["Status"] != "COMPLETED" {
		return
	}
	switch r.kind {
	case "MLModel":
		r.rec["SizeInBytes"] = int64(2048 + len(fcStr(r.rec, "Recipe")))
		if ds := st.res["DataSource"][r.dsID]; ds != nil {
			r.rec["Schema"] = ds.rec["DataSourceSchema"]
		}
	case "Evaluation":
		model := st.res["MLModel"][r.modelID]
		metric := map[string]string{"BINARY": "BinaryAUC", "REGRESSION": "RegressionRMSE", "MULTICLASS": "MulticlassAvgFScore"}[fcStr(model.rec, "MLModelType")]
		value := 0.7 + 0.29*amlScore(r.modelID+"|"+r.dsID)
		if metric == "RegressionRMSE" {
			value = 10 * amlScore(r.modelID+"|"+r.dsID)
		}
		r.rec["PerformanceMetrics"] = map[string]any{"Properties": map[string]any{metric: amlFmt(value)}}
	}
}

func amlSize(v ...any) int {
	n := 0
	for _, item := range v {
		raw, _ := json.Marshal(item)
		n += len(raw)
	}
	return n
}

// admit checks a new resource against the count and byte caps, evicting
// one DELETED resource of the kind when the kind is full.
func (st *machineLearningStore) admit(kind string, size int) *fcErr {
	if st.bytes+size > amlMaxBytes {
		return fcError("LimitExceededException", "Stackyard holds at most %d MiB of Amazon ML resources", amlMaxBytes>>20)
	}
	if len(st.res[kind]) < amlMaxPerKind {
		return nil
	}
	for id, r := range st.res[kind] {
		if r.rec["Status"] == "DELETED" {
			st.bytes -= r.size
			delete(st.res[kind], id)
			return nil
		}
	}
	return fcError("LimitExceededException", "Stackyard holds at most %d %s resources", amlMaxPerKind, kind)
}

// existing handles a create whose ID is taken: the same request again is an
// idempotent retry, anything else a mismatch.
func (st *machineLearningStore) existing(kind, id string, in map[string]any) (map[string]any, bool, *fcErr) {
	r := st.res[kind][id]
	if r == nil {
		return nil, false, nil
	}
	if r.rec["Status"] != "DELETED" && reflect.DeepEqual(r.create, in) {
		return map[string]any{amlIDMember[kind]: id}, true, nil
	}
	return nil, true, fcError("IdempotentParameterMismatchException", "%s %s already exists with different parameters", kind, id)
}

func (st *machineLearningStore) add(kind, id string, in, rec map[string]any, r *amlRes) (map[string]any, *fcErr) {
	now := time.Now().UTC()
	r.kind, r.id, r.create, r.rec, r.created = kind, id, in, rec, now
	rec[amlIDMember[kind]] = id
	rec["CreatedByIamUser"], rec["CreatedAt"], rec["LastUpdatedAt"], rec["Status"] = amlIAMUser, fcEpoch(now), fcEpoch(now), "PENDING"
	r.size = amlSize(in, rec)
	if err := st.admit(kind, r.size); err != nil {
		return nil, err
	}
	st.res[kind][id] = r
	st.bytes += r.size
	return map[string]any{amlIDMember[kind]: id}, nil
}

func (st *machineLearningStore) createDataSource(op string, in map[string]any, pre *amlPre) (map[string]any, *fcErr) {
	id := fcStr(in, "DataSourceId")
	if out, taken, err := st.existing("DataSource", id, in); taken {
		return out, err
	}
	rec := map[string]any{"Name": fcStr(in, "DataSourceName"), "ComputeStatistics": fcBool(in, "ComputeStatistics")}
	r := &amlRes{failure: pre.failure, header: pre.header}
	var spec map[string]any
	switch op {
	case "CreateDataSourceFromS3":
		spec = fcObj(in, "DataSpec")
		rec["DataLocationS3"] = fcStr(spec, "DataLocationS3")
		rec["DataSizeInBytes"], rec["NumberOfFiles"], rec["DataSourceSchema"] = pre.size, int64(pre.files), pre.schema
	case "CreateDataSourceFromRDS":
		spec = fcObj(in, "RDSData")
		rec["DataLocationS3"], rec["RoleARN"] = fcStr(spec, "S3StagingLocation"), fcStr(in, "RoleARN")
		rec["RDSMetadata"] = map[string]any{"Database": fcObj(spec, "DatabaseInformation"),
			"DatabaseUserName": fcStr(fcObj(spec, "DatabaseCredentials"), "Username"), "SelectSqlQuery": fcStr(spec, "SelectSqlQuery"),
			"ResourceRole": fcStr(spec, "ResourceRole"), "ServiceRole": fcStr(spec, "ServiceRole"),
			"DataPipelineId": "df-" + strings.ToUpper(strconv.FormatUint(uint64(amlScore(id)*1e12), 36))}
	default:
		spec = fcObj(in, "DataSpec")
		rec["DataLocationS3"], rec["RoleARN"] = fcStr(spec, "S3StagingLocation"), fcStr(in, "RoleARN")
		rec["RedshiftMetadata"] = map[string]any{"RedshiftDatabase": fcObj(spec, "DatabaseInformation"),
			"DatabaseUserName": fcStr(fcObj(spec, "DatabaseCredentials"), "Username"), "SelectSqlQuery": fcStr(spec, "SelectSqlQuery")}
	}
	if op != "CreateDataSourceFromS3" {
		// Database sources are staged by the query; their schema is optional.
		if schema := fcStr(spec, "DataSchema"); schema != "" {
			sc, err := amlParseSchema(schema)
			if err != nil {
				return nil, fcInvalid("%s", err.Error())
			}
			rec["DataSourceSchema"], r.header = schema, sc.DataFileContainsHeader
		}
		rec["DataSizeInBytes"], rec["NumberOfFiles"] = int64(0), int64(0)
	}
	if v := fcStr(spec, "DataRearrangement"); v != "" {
		rec["DataRearrangement"] = v
	}
	return st.add("DataSource", id, in, rec, r)
}

var amlDefaultParameters = map[string]string{
	"sgd.maxMLModelSizeInBytes": "104857600", "sgd.maxPasses": "10", "sgd.shuffleType": "auto",
	"sgd.l2RegularizationAmount": "1e-6",
}

func (st *machineLearningStore) createModel(in map[string]any) (map[string]any, *fcErr) {
	id := fcStr(in, "MLModelId")
	if out, taken, err := st.existing("MLModel", id, in); taken {
		return out, err
	}
	ds, err := st.get("DataSource", fcStr(in, "TrainingDataSourceId"))
	if err != nil {
		return nil, err
	}
	params := map[string]any{}
	for k, v := range amlDefaultParameters {
		params[k] = v
	}
	for k, v := range fcObj(in, "Parameters") {
		if _, known := amlDefaultParameters[k]; !known {
			return nil, fcInvalid("unknown training parameter %s", k)
		}
		params[k] = v
	}
	recipe := fcStr(in, "Recipe")
	if recipe == "" {
		recipe = `{"groups":{},"assignments":{},"outputs":["ALL_INPUTS"]}`
	}
	rec := map[string]any{"Name": fcStr(in, "MLModelName"), "TrainingDataSourceId": ds.id, "MLModelType": fcStr(in, "MLModelType"),
		"TrainingParameters": params, "InputDataLocationS3": ds.rec["DataLocationS3"], "Algorithm": "sgd", "Recipe": recipe,
		"EndpointInfo": map[string]any{"PeakRequestsPerSecond": 0, "EndpointStatus": "NONE"}}
	r := &amlRes{dsID: ds.id}
	if rec["MLModelType"] == "BINARY" {
		r.threshold, rec["ScoreThreshold"] = 0.5, 0.5
	}
	return st.add("MLModel", id, in, rec, r)
}

// createJob creates an evaluation or batch prediction of a model over a
// data source.
func (st *machineLearningStore) createJob(kind string, in map[string]any, pre *amlPre) (map[string]any, bool, *fcErr) {
	id := fcStr(in, amlIDMember[kind])
	if out, taken, err := st.existing(kind, id, in); taken {
		return out, false, err
	}
	model, err := st.get("MLModel", fcStr(in, "MLModelId"))
	if err != nil {
		return nil, false, err
	}
	ds, err := st.get("DataSource", fcStr(in, kind+"DataSourceId"))
	if err != nil {
		return nil, false, err
	}
	rec := map[string]any{"Name": fcStr(in, kind+"Name"), "MLModelId": model.id, kind + "DataSourceId": ds.id,
		"InputDataLocationS3": ds.rec["DataLocationS3"]}
	r := &amlRes{modelID: model.id, dsID: ds.id}
	if kind == "BatchPrediction" {
		rec["Name"], rec["OutputUri"] = fcStr(in, "BatchPredictionName"), fcStr(in, "OutputUri")
		rec["TotalRecordCount"], rec["InvalidRecordCount"] = int64(pre.records), int64(0)
		r.failure = pre.failure
	}
	out, err := st.add(kind, id, in, rec, r)
	return out, err == nil && kind == "BatchPrediction", err
}

// batchSnapshot reads what a batch prediction needs before S3 is read.
func (st *machineLearningStore) batchSnapshot(modelID, dsID string) (amlBatchSnapshot, bool) {
	st.mu.Lock()
	defer st.mu.Unlock()
	model, ds := st.res["MLModel"][modelID], st.res["DataSource"][dsID]
	if model == nil || ds == nil {
		return amlBatchSnapshot{}, false
	}
	return amlBatchSnapshot{location: fcStr(ds.rec, "DataLocationS3"), header: ds.header,
		modelType: fcStr(model.rec, "MLModelType"), threshold: model.threshold}, true
}

func (st *machineLearningStore) getOut(kind string, in map[string]any) (map[string]any, *fcErr) {
	r := st.res[kind][fcStr(in, amlIDMember[kind])]
	if r == nil {
		return nil, amlNotFound(kind, fcStr(in, amlIDMember[kind]))
	}
	st.advance(r)
	out := make(map[string]any, len(r.rec)+1)
	for k, v := range r.rec {
		out[k] = v
	}
	out["LogUri"] = "https://eml-prod-emr.s3.amazonaws.com/" + fcAccount + "/" + strings.ToLower(kind) + "/" + r.id + "/log.txt"
	if !fcBool(in, "Verbose") {
		delete(out, "DataSourceSchema")
		delete(out, "Recipe")
		delete(out, "Schema")
	}
	return out, nil
}

func (st *machineLearningStore) markDeleted(kind, id string) (map[string]any, *fcErr) {
	r, err := st.get(kind, id)
	if err != nil {
		return nil, err
	}
	r.rec["Status"], r.rec["LastUpdatedAt"] = "DELETED", fcEpoch(time.Now().UTC())
	if ep, _ := r.rec["EndpointInfo"].(map[string]any); ep != nil {
		ep["EndpointStatus"] = "NONE"
	}
	return map[string]any{amlIDMember[kind]: id}, nil
}

// update renames a resource (and moves a model's score threshold),
// re-charging its size against the byte cap.
func (st *machineLearningStore) update(kind string, in map[string]any) (map[string]any, *fcErr) {
	id := fcStr(in, amlIDMember[kind])
	r, err := st.get(kind, id)
	if err != nil {
		return nil, err
	}
	rec := make(map[string]any, len(r.rec))
	for k, v := range r.rec {
		rec[k] = v
	}
	if name, ok := in[kind+"Name"].(string); ok {
		rec["Name"] = name
	}
	threshold := r.threshold
	if v, ok := fcNum(in, "ScoreThreshold"); ok {
		if rec["MLModelType"] != "BINARY" {
			return nil, fcInvalid("ScoreThreshold applies only to BINARY models")
		}
		now := fcEpoch(time.Now().UTC())
		threshold, rec["ScoreThreshold"], rec["ScoreThresholdLastUpdatedAt"] = v, v, now
	}
	rec["LastUpdatedAt"] = fcEpoch(time.Now().UTC())
	size := amlSize(r.create, rec)
	if st.bytes-r.size+size > amlMaxBytes {
		return nil, fcError("LimitExceededException", "Stackyard holds at most %d MiB of Amazon ML resources", amlMaxBytes>>20)
	}
	st.bytes += size - r.size
	r.rec, r.size, r.threshold = rec, size, threshold
	return map[string]any{amlIDMember[kind]: id}, nil
}

// amlCompare orders two field values; timestamps compare numerically.
func amlCompare(field string, a, b any) int {
	if field == "CreatedAt" || field == "LastUpdatedAt" {
		x, y := amlTime(a), amlTime(b)
		switch {
		case x < y:
			return -1
		case x > y:
			return 1
		}
		return 0
	}
	return strings.Compare(amlString(a), amlString(b))
}

func amlString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// amlTime reads an epoch number or a filter operand (epoch seconds or RFC 3339).
func amlTime(v any) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case string:
		if f, err := strconv.ParseFloat(t, 64); err == nil {
			return f
		}
		if ts, err := time.Parse(time.RFC3339, t); err == nil {
			return fcEpoch(ts)
		}
	}
	return 0
}

func amlFieldValue(r *amlRes, field string) any {
	if field == "EndpointStatus" {
		ep, _ := r.rec["EndpointInfo"].(map[string]any)
		return ep["EndpointStatus"]
	}
	return r.rec[field]
}

func amlMatches(field string, v any, in map[string]any) bool {
	for _, op := range []string{"EQ", "NE", "GT", "LT", "GE", "LE", "Prefix"} {
		operand, ok := in[op].(string)
		if !ok {
			continue
		}
		c := amlCompare(field, v, operand)
		pass := map[string]bool{"EQ": c == 0, "NE": c != 0, "GT": c > 0, "LT": c < 0, "GE": c >= 0, "LE": c <= 0,
			"Prefix": strings.HasPrefix(amlString(v), operand)}[op]
		if !pass {
			return false
		}
	}
	return true
}

// describe filters, sorts and pages one kind (at most amlMaxPerKind items).
func (st *machineLearningStore) describe(kind string, in map[string]any) (map[string]any, *fcErr) {
	field := "CreatedAt"
	if fv := fcStr(in, "FilterVariable"); fv != "" {
		mapped, ok := amlFilterFields[kind][fv]
		if !ok {
			return nil, fcInvalid("FilterVariable %s does not apply to %s", fv, kind)
		}
		field = mapped
	}
	items := make([]*amlRes, 0, len(st.res[kind]))
	for _, r := range st.res[kind] {
		st.advance(r)
		if amlMatches(field, amlFieldValue(r, field), in) {
			items = append(items, r)
		}
	}
	desc := fcStr(in, "SortOrder") == "dsc"
	sort.Slice(items, func(i, j int) bool {
		c := amlCompare(field, amlFieldValue(items[i], field), amlFieldValue(items[j], field))
		if c == 0 {
			c = strings.Compare(items[i].id, items[j].id)
		}
		return (c < 0) != desc
	})
	start := 0
	if tok := fcStr(in, "NextToken"); tok != "" {
		n, err := strconv.ParseInt(tok, 10, 32)
		if err != nil || n < 0 || int(n) > len(items) {
			return nil, fcInvalid("invalid NextToken")
		}
		start = int(n)
	}
	limit := 100
	if v, ok := fcNum(in, "Limit"); ok {
		limit = int(v) // the model bounds Limit to 1..100
	}
	end := min(start+limit, len(items))
	results := make([]any, 0, end-start)
	for _, r := range items[start:end] {
		results = append(results, r.rec)
	}
	out := map[string]any{"Results": results}
	if end < len(items) {
		out["NextToken"] = strconv.Itoa(end)
	}
	return out, nil
}

func (st *machineLearningStore) realtimeEndpoint(create bool, id, base string) (map[string]any, *fcErr) {
	r, err := st.get("MLModel", id)
	if err != nil {
		return nil, err
	}
	ep := map[string]any{"PeakRequestsPerSecond": 0, "EndpointStatus": "NONE"}
	if create {
		ep = map[string]any{"PeakRequestsPerSecond": 200, "CreatedAt": fcEpoch(time.Now().UTC()),
			"EndpointUrl": base, "EndpointStatus": "UPDATING"}
	}
	r.rec["EndpointInfo"] = ep
	return map[string]any{"MLModelId": id, "RealtimeEndpointInfo": ep}, nil
}

func (st *machineLearningStore) predict(in map[string]any) (map[string]any, *fcErr) {
	r, err := st.get("MLModel", fcStr(in, "MLModelId"))
	if err != nil {
		return nil, err
	}
	ep, _ := r.rec["EndpointInfo"].(map[string]any)
	if r.rec["Status"] != "COMPLETED" || ep["EndpointStatus"] != "READY" {
		return nil, fcError("PredictorNotMountedException", "MLModel %s has no READY realtime endpoint", r.id)
	}
	return map[string]any{"Prediction": amlPredict(fcStr(r.rec, "MLModelType"), r.threshold, fcObj(in, "Record"))}, nil
}

func (st *machineLearningStore) tagOp(op string, in map[string]any) (map[string]any, *fcErr) {
	kind, id := fcStr(in, "ResourceType"), fcStr(in, "ResourceId")
	r := st.res[kind][id]
	if r == nil {
		return nil, amlNotFound(kind, id)
	}
	out := map[string]any{"ResourceId": id, "ResourceType": kind}
	switch op {
	case "AddTags":
		tags := r.tags
		tagsIn, _ := in["Tags"].([]any)
		for _, t := range tagsIn {
			tag, _ := t.(map[string]any)
			key, value := fcStr(tag, "Key"), fcStr(tag, "Value")
			if key == "" || strings.HasPrefix(key, "aws:") {
				return nil, fcError("InvalidTagException", "tag key %q is not allowed", key)
			}
			tags = amlSetTag(tags, key, value)
			if len(tags) > amlMaxTags {
				return nil, fcError("TagLimitExceededException", "a resource can have at most %d tags", amlMaxTags)
			}
		}
		return out, st.retag(r, tags)
	case "DeleteTags":
		drop := map[string]bool{}
		for _, k := range fcStrs(in, "TagKeys") {
			drop[k] = true
		}
		var tags [][2]string
		for _, t := range r.tags {
			if !drop[t[0]] {
				tags = append(tags, t)
			}
		}
		return out, st.retag(r, tags)
	}
	list := make([]any, 0, len(r.tags))
	for _, t := range r.tags {
		list = append(list, map[string]any{"Key": t[0], "Value": t[1]})
	}
	out["Tags"] = list
	return out, nil
}

// amlSetTag returns tags with key set to value, never mutating tags.
func amlSetTag(tags [][2]string, key, value string) [][2]string {
	out := make([][2]string, 0, len(tags)+1)
	replaced := false
	for _, t := range tags {
		if t[0] == key {
			t[1], replaced = value, true
		}
		out = append(out, t)
	}
	if !replaced {
		out = append(out, [2]string{key, value})
	}
	return out
}

// retag stores new tags, re-charging the resource's size.
func (st *machineLearningStore) retag(r *amlRes, tags [][2]string) *fcErr {
	size := amlSize(r.create, r.rec)
	for _, t := range tags {
		size += len(t[0]) + len(t[1])
	}
	if st.bytes-r.size+size > amlMaxBytes {
		return fcError("LimitExceededException", "Stackyard holds at most %d MiB of Amazon ML resources", amlMaxBytes>>20)
	}
	st.bytes += size - r.size
	r.tags, r.size = tags, size
	return nil
}
