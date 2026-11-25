package server

import (
	"encoding/json"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

const (
	fcRegion     = "us-east-1"
	fcAccount    = "123456789012"
	fcMaxPerKind = 500 // resources of one kind
	fcMaxTags    = 200
	fcMaxFilters = 20
	fcMaxHorizon = 500
	fcMaxBytes   = 64 << 20 // stored request documents, store-wide
	// ponytail: async resources go ACTIVE on the read after the first
	// Describe, or once this long has passed, whichever comes first.
	fcSettle = 3 * time.Second
)

// fcKind is one Forecast resource type. Every kind shares the same
// Create/Describe/Delete/List shape: <Name>Name and <Name>Arn members, a
// list operation and an ARN of the form <arnType>/[<parent name>/]<name>.
type fcKind struct {
	name      string
	arnType   string
	parentKey string // create member naming the parent resource's ARN
	nested    bool   // the ARN embeds the parent's name
	async     bool
	listOp    string
}

var fcKinds = []*fcKind{
	{name: "Dataset", arnType: "dataset", listOp: "ListDatasets"},
	{name: "DatasetGroup", arnType: "dataset-group", listOp: "ListDatasetGroups"},
	{name: "DatasetImportJob", arnType: "dataset-import-job", parentKey: "DatasetArn", nested: true, async: true, listOp: "ListDatasetImportJobs"},
	{name: "Predictor", arnType: "predictor", async: true, listOp: "ListPredictors"},
	{name: "Forecast", arnType: "forecast", parentKey: "PredictorArn", async: true, listOp: "ListForecasts"},
	{name: "ForecastExportJob", arnType: "forecast-export-job", parentKey: "ForecastArn", nested: true, async: true, listOp: "ListForecastExportJobs"},
	{name: "PredictorBacktestExportJob", arnType: "predictor-backtest-export-job", parentKey: "PredictorArn", nested: true, async: true, listOp: "ListPredictorBacktestExportJobs"},
	{name: "Explainability", arnType: "explainability", parentKey: "ResourceArn", async: true, listOp: "ListExplainabilities"},
	{name: "ExplainabilityExport", arnType: "explainability-export", parentKey: "ExplainabilityArn", nested: true, async: true, listOp: "ListExplainabilityExports"},
	{name: "Monitor", arnType: "monitor", parentKey: "ResourceArn", async: true, listOp: "ListMonitors"},
	{name: "WhatIfAnalysis", arnType: "what-if-analysis", parentKey: "ForecastArn", async: true, listOp: "ListWhatIfAnalyses"},
	{name: "WhatIfForecast", arnType: "what-if-forecast", parentKey: "WhatIfAnalysisArn", nested: true, async: true, listOp: "ListWhatIfForecasts"},
	{name: "WhatIfForecastExport", arnType: "what-if-forecast-export", async: true, listOp: "ListWhatIfForecastExports"},
}

var fcKindByName, fcKindByListOp = func() (map[string]*fcKind, map[string]*fcKind) {
	byName, byList := map[string]*fcKind{}, map[string]*fcKind{}
	for _, k := range fcKinds {
		byName[k.name] = k
		byList[k.listOp] = k
	}
	return byName, byList
}()

var fcDefaultTypes = []string{"0.10", "0.50", "0.90"}

var fcAlgorithms = []string{"ARIMA", "CNN-QR", "Deep_AR_Plus", "ETS", "NPTS", "Prophet"}

// fcTargetField is the target attribute each domain's target time series needs.
var fcTargetField = map[string]string{
	"RETAIL": "demand", "CUSTOM": "target_value", "INVENTORY_PLANNING": "demand",
	"EC2_CAPACITY": "number_of_instances", "WORK_FORCE": "workforce_demand",
	"WEB_TRAFFIC": "value", "METRICS": "metric_value",
}

type fcRes struct {
	kind              *fcKind
	arn, name, parent string
	refs              []string       // other resources this one reads (what-if exports)
	doc               map[string]any // conformed create input plus derived fields
	status            string
	seen              bool
	seq, size         int
	created, modified time.Time
}

type forecastStore struct {
	mu    sync.Mutex
	res   map[string]*fcRes
	tags  map[string]map[string]string
	seq   int
	bytes int
}

func newForecastStore() *forecastStore {
	return &forecastStore{res: map[string]*fcRes{}, tags: map[string]map[string]string{}}
}

func fcAlgorithmArn(name string) string { return "arn:aws:forecast:::algorithm/" + name }

func fcEpoch(t time.Time) float64 { return float64(t.UnixMilli()) / 1000 }

// run executes op under the store lock and conforms its output there, so the
// response never shares maps with stored state.
func (st *forecastStore) run(op string, in map[string]any) (any, *fcErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	out, err := st.dispatch(op, in)
	if err != nil {
		return nil, err
	}
	if out == nil {
		out = map[string]any{}
	}
	model := awsmodels.ForecastModel()
	return model.Conform(model.Operations[op].Output, out), nil
}

func (st *forecastStore) dispatch(op string, in map[string]any) (map[string]any, *fcErr) {
	switch op {
	case "CreateAutoPredictor":
		return st.create(fcKindByName["Predictor"], in, true)
	case "DescribeAutoPredictor":
		return st.describe(fcKindByName["Predictor"], fcStr(in, "PredictorArn"))
	case "DeleteResourceTree":
		return nil, st.deleteTree(fcStr(in, "ResourceArn"))
	case "UpdateDatasetGroup":
		return nil, st.updateDatasetGroup(in)
	case "GetAccuracyMetrics":
		return st.accuracyMetrics(fcStr(in, "PredictorArn"))
	case "ListMonitorEvaluations":
		return st.monitorEvaluations(in)
	case "StopResource":
		return nil, st.stop(fcStr(in, "ResourceArn"))
	case "ResumeResource":
		return nil, st.resume(fcStr(in, "ResourceArn"))
	case "TagResource":
		return nil, st.tag(fcStr(in, "ResourceArn"), in["Tags"])
	case "UntagResource":
		return nil, st.untag(fcStr(in, "ResourceArn"), fcStrs(in, "TagKeys"))
	case "ListTagsForResource":
		return st.listTags(fcStr(in, "ResourceArn"))
	}
	if k := fcKindByListOp[op]; k != nil {
		return st.list(k, op, in)
	}
	for _, verb := range []string{"Create", "Describe", "Delete"} {
		k := fcKindByName[strings.TrimPrefix(op, verb)]
		if !strings.HasPrefix(op, verb) || k == nil {
			continue
		}
		switch verb {
		case "Create":
			return st.create(k, in, false)
		case "Describe":
			return st.describe(k, fcStr(in, k.name+"Arn"))
		default:
			return nil, st.delete(k, fcStr(in, k.name+"Arn"))
		}
	}
	return nil, fcError("UnknownOperationException", "unknown operation %s", op)
}

// settle moves an async resource to ACTIVE once it has been seen in progress
// or has been in progress long enough.
func (st *forecastStore) settle(r *fcRes) {
	if r.status == "CREATE_IN_PROGRESS" && (r.seen || time.Since(r.created) > fcSettle) {
		r.status = "ACTIVE"
		r.modified = time.Now().UTC()
	}
}

// get returns the resource of kind k (any kind when k is nil) at arn.
func (st *forecastStore) get(k *fcKind, arn string) (*fcRes, *fcErr) {
	r := st.res[arn]
	if r == nil || (k != nil && r.kind != k) {
		return nil, fcNotFound("No resource found with ARN %s", arn)
	}
	st.settle(r)
	return r, nil
}

// active returns the resource at arn, which must be ACTIVE.
func (st *forecastStore) active(k *fcKind, arn string) (*fcRes, *fcErr) {
	r, err := st.get(k, arn)
	if err != nil {
		return nil, err
	}
	if r.status != "ACTIVE" {
		return nil, fcInUse("%s %s is %s, not ACTIVE", k.name, arn, r.status)
	}
	return r, nil
}

func (st *forecastStore) create(k *fcKind, in map[string]any, auto bool) (map[string]any, *fcErr) {
	name := fcStr(in, k.name+"Name")
	doc := map[string]any{}
	for key, v := range in {
		if key != "Tags" {
			doc[key] = v
		}
	}
	tags, err := fcTagMap(nil, in["Tags"])
	if err != nil {
		return nil, err
	}
	parent, refs, err := st.prepare(k, doc, auto)
	if err != nil {
		return nil, err
	}
	arn := "arn:aws:forecast:" + fcRegion + ":" + fcAccount + ":" + k.arnType + "/"
	if k.nested {
		arn += st.res[parent].name + "/"
	}
	arn += name
	if st.res[arn] != nil {
		return nil, fcError("ResourceAlreadyExistsException", "A %s named %s already exists", k.name, name)
	}
	if st.count(k) >= fcMaxPerKind {
		return nil, fcError("LimitExceededException", "at most %d resources of type %s", fcMaxPerKind, k.name)
	}
	raw, _ := json.Marshal(doc)
	if st.bytes+len(raw) > fcMaxBytes {
		return nil, fcError("LimitExceededException", "stored resource definitions are limited to %d MiB", fcMaxBytes>>20)
	}
	r := st.insert(k, arn, name, parent, refs, doc, tags)
	r.size = len(raw)
	st.bytes += r.size
	if k.name == "Predictor" && auto {
		st.autoPredictorChildren(r, in)
	}
	return map[string]any{k.name + "Arn": arn}, nil
}

func (st *forecastStore) insert(k *fcKind, arn, name, parent string, refs []string, doc map[string]any, tags map[string]string) *fcRes {
	now := time.Now().UTC()
	st.seq++
	r := &fcRes{kind: k, arn: arn, name: name, parent: parent, refs: refs, doc: doc, status: "ACTIVE", seq: st.seq, created: now, modified: now}
	if k.async {
		r.status = "CREATE_IN_PROGRESS"
	}
	st.res[arn] = r
	st.tags[arn] = tags
	return r
}

// autoPredictorChildren creates the explainability and monitor an auto
// predictor asks for with ExplainPredictor and MonitorConfig.
func (st *forecastStore) autoPredictorChildren(p *fcRes, in map[string]any) {
	if fcBool(in, "ExplainPredictor") {
		k := fcKindByName["Explainability"]
		name := p.name + "_explainability"
		arn := "arn:aws:forecast:" + fcRegion + ":" + fcAccount + ":explainability/" + name
		if st.res[arn] == nil {
			doc := map[string]any{"ExplainabilityName": name, "ResourceArn": p.arn, "EnableVisualization": false,
				"ExplainabilityConfig": map[string]any{"TimeSeriesGranularity": "ALL", "TimePointGranularity": "ALL"}}
			st.insert(k, arn, name, p.arn, nil, doc, map[string]string{})
			p.doc["ExplainabilityArn"] = arn
		}
	}
	if mc := fcObj(in, "MonitorConfig"); mc != nil {
		k := fcKindByName["Monitor"]
		name := fcStr(mc, "MonitorName")
		arn := "arn:aws:forecast:" + fcRegion + ":" + fcAccount + ":monitor/" + name
		if st.res[arn] == nil {
			st.insert(k, arn, name, p.arn, nil, map[string]any{"MonitorName": name, "ResourceArn": p.arn}, map[string]string{})
			p.doc["MonitorArn"] = arn
		}
	}
}

func (st *forecastStore) count(k *fcKind) int {
	n := 0
	for _, r := range st.res {
		if r.kind == k {
			n++
		}
	}
	return n
}

// prepare checks a create request against the store, fills in defaults and
// returns the parent resource's ARN.
func (st *forecastStore) prepare(k *fcKind, doc map[string]any, auto bool) (string, []string, *fcErr) {
	parent := fcStr(doc, k.parentKey)
	switch k.name {
	case "Dataset":
		return "", nil, fcCheckDataset(doc)
	case "DatasetGroup":
		if _, ok := doc["DatasetArns"]; !ok {
			doc["DatasetArns"] = []any{}
		}
		return "", nil, st.checkGroupDatasets(fcStr(doc, "Domain"), fcStrs(doc, "DatasetArns"))
	case "DatasetImportJob":
		ds, err := st.get(fcKindByName["Dataset"], parent)
		if err != nil {
			return "", nil, err
		}
		if fcStr(doc, "TimeZone") != "" && fcBool(doc, "UseGeolocationForTimeZone") {
			return "", nil, fcInvalid("TimeZone and UseGeolocationForTimeZone can't both be set")
		}
		fcDefault(doc, "Format", "CSV")
		fcDefault(doc, "ImportMode", "FULL")
		if fcStr(ds.doc, "DatasetType") != "ITEM_METADATA" {
			fcDefault(doc, "TimestampFormat", "yyyy-MM-dd HH:mm:ss")
		}
		return parent, nil, nil
	case "Predictor":
		if auto {
			return st.prepareAutoPredictor(doc)
		}
		return st.preparePredictor(doc)
	case "Forecast":
		p, err := st.active(fcKindByName["Predictor"], parent)
		if err != nil {
			return "", nil, err
		}
		types := fcStrs(p.doc, "ForecastTypes")
		if _, ok := doc["ForecastTypes"]; !ok {
			doc["ForecastTypes"] = fcAnyList(types)
		}
		for _, t := range fcStrs(doc, "ForecastTypes") {
			if !fcContains(types, t) {
				return "", nil, fcInvalid("forecast type %s is not one of the predictor's forecast types", t)
			}
		}
		doc["DatasetGroupArn"] = fcStr(p.doc, "DatasetGroupArn")
		doc["CreatedUsingAutoPredictor"] = fcBool(p.doc, "IsAutoPredictor")
		return parent, nil, nil
	case "ForecastExportJob":
		fcDefault(doc, "Format", "CSV")
		_, err := st.active(fcKindByName["Forecast"], parent)
		return parent, nil, err
	case "PredictorBacktestExportJob":
		fcDefault(doc, "Format", "CSV")
		_, err := st.active(fcKindByName["Predictor"], parent)
		return parent, nil, err
	case "Explainability":
		return parent, nil, st.prepareExplainability(doc, parent)
	case "ExplainabilityExport":
		fcDefault(doc, "Format", "CSV")
		_, err := st.active(fcKindByName["Explainability"], parent)
		return parent, nil, err
	case "Monitor":
		p, err := st.active(fcKindByName["Predictor"], parent)
		if err != nil {
			return "", nil, err
		}
		if !fcBool(p.doc, "IsAutoPredictor") {
			return "", nil, fcInvalid("monitors can only be created for predictors created with CreateAutoPredictor")
		}
		for _, kid := range st.children(parent) {
			if kid.kind == k {
				return "", nil, fcError("ResourceAlreadyExistsException", "predictor %s already has monitor %s", parent, kid.arn)
			}
		}
		return parent, nil, nil
	case "WhatIfAnalysis":
		f, err := st.active(fcKindByName["Forecast"], parent)
		if err != nil {
			return "", nil, err
		}
		if !fcBool(f.doc, "CreatedUsingAutoPredictor") {
			return "", nil, fcInvalid("what-if analyses need a forecast created from an auto predictor")
		}
		return parent, nil, nil
	case "WhatIfForecast":
		a, err := st.active(fcKindByName["WhatIfAnalysis"], parent)
		if err != nil {
			return "", nil, err
		}
		if doc["TimeSeriesTransformations"] == nil && doc["TimeSeriesReplacementsDataSource"] == nil {
			return "", nil, fcInvalid("provide TimeSeriesTransformations or TimeSeriesReplacementsDataSource")
		}
		if f := st.res[a.parent]; f != nil {
			doc["ForecastTypes"] = f.doc["ForecastTypes"]
		}
		return parent, nil, nil
	case "WhatIfForecastExport":
		fcDefault(doc, "Format", "CSV")
		arns := fcStrs(doc, "WhatIfForecastArns")
		for _, arn := range arns {
			if _, err := st.active(fcKindByName["WhatIfForecast"], arn); err != nil {
				return "", nil, err
			}
		}
		return "", arns, nil
	}
	return parent, nil, nil
}

func fcDefault(doc map[string]any, key string, v any) {
	if _, ok := doc[key]; !ok {
		doc[key] = v
	}
}

func fcContains(list []string, v string) bool {
	for _, item := range list {
		if item == v {
			return true
		}
	}
	return false
}

func fcSchemaAttrs(doc map[string]any) map[string]string {
	out := map[string]string{}
	attrs, _ := fcObj(doc, "Schema")["Attributes"].([]any)
	for _, a := range attrs {
		if m, ok := a.(map[string]any); ok {
			out[fcStr(m, "AttributeName")] = fcStr(m, "AttributeType")
		}
	}
	return out
}

func fcCheckDataset(doc map[string]any) *fcErr {
	attrs, _ := fcObj(doc, "Schema")["Attributes"].([]any)
	seen := map[string]bool{}
	for _, a := range attrs {
		m, _ := a.(map[string]any)
		name := fcStr(m, "AttributeName")
		if name == "" || fcStr(m, "AttributeType") == "" {
			return fcInvalid("every schema attribute needs an AttributeName and AttributeType")
		}
		if seen[name] {
			return fcInvalid("schema attribute %s appears more than once", name)
		}
		seen[name] = true
	}
	types := fcSchemaAttrs(doc)
	if fcStr(doc, "DatasetType") == "ITEM_METADATA" {
		return nil
	}
	if types["timestamp"] != "timestamp" {
		return fcInvalid("a time series dataset needs a timestamp attribute of type timestamp")
	}
	if field := fcTargetField[fcStr(doc, "Domain")]; fcStr(doc, "DatasetType") == "TARGET_TIME_SERIES" && types[field] == "" {
		return fcInvalid("a %s target time series needs a %s attribute", fcStr(doc, "Domain"), field)
	}
	return nil
}

func (st *forecastStore) checkGroupDatasets(domain string, arns []string) *fcErr {
	byType := map[string]bool{}
	for _, arn := range arns {
		ds, err := st.get(fcKindByName["Dataset"], arn)
		if err != nil {
			return err
		}
		if d := fcStr(ds.doc, "Domain"); d != domain {
			return fcInvalid("dataset %s is in domain %s, not %s", arn, d, domain)
		}
		t := fcStr(ds.doc, "DatasetType")
		if byType[t] {
			return fcInvalid("a dataset group holds at most one %s dataset", t)
		}
		byType[t] = true
	}
	return nil
}

// groupTarget returns the dataset group's target time series dataset.
func (st *forecastStore) groupTarget(groupArn string) (*fcRes, *fcErr) {
	g, err := st.get(fcKindByName["DatasetGroup"], groupArn)
	if err != nil {
		return nil, err
	}
	for _, arn := range fcStrs(g.doc, "DatasetArns") {
		if ds := st.res[arn]; ds != nil && fcStr(ds.doc, "DatasetType") == "TARGET_TIME_SERIES" {
			return ds, nil
		}
	}
	return nil, fcInvalid("dataset group %s has no TARGET_TIME_SERIES dataset", groupArn)
}

func fcCheckHorizon(doc map[string]any) *fcErr {
	if h, _ := fcNum(doc, "ForecastHorizon"); h < 1 || h > fcMaxHorizon {
		return fcInvalid("ForecastHorizon must be between 1 and %d", fcMaxHorizon)
	}
	return nil
}

func (st *forecastStore) preparePredictor(doc map[string]any) (string, []string, *fcErr) {
	group := fcStr(fcObj(doc, "InputDataConfig"), "DatasetGroupArn")
	if _, err := st.groupTarget(group); err != nil {
		return "", nil, err
	}
	if err := fcCheckHorizon(doc); err != nil {
		return "", nil, err
	}
	algo := fcStr(doc, "AlgorithmArn")
	switch {
	case fcBool(doc, "PerformAutoML") && algo != "":
		return "", nil, fcInvalid("AlgorithmArn can't be set when PerformAutoML is true")
	case fcBool(doc, "PerformAutoML"):
		arns := make([]string, len(fcAlgorithms))
		for i, a := range fcAlgorithms {
			arns[i] = fcAlgorithmArn(a)
		}
		doc["AutoMLAlgorithmArns"] = fcAnyList(arns)
		doc["AlgorithmArn"] = fcAlgorithmArn("CNN-QR")
	case algo == "":
		return "", nil, fcInvalid("AlgorithmArn is required when PerformAutoML is false")
	case !fcContains(fcAlgorithms, strings.TrimPrefix(algo, "arn:aws:forecast:::algorithm/")):
		return "", nil, fcInvalid("unknown algorithm %s", algo)
	}
	horizon, _ := fcNum(doc, "ForecastHorizon")
	fcDefault(doc, "ForecastTypes", fcAnyList(fcDefaultTypes))
	fcDefault(doc, "PerformHPO", false)
	fcDefault(doc, "PerformAutoML", false)
	fcDefault(doc, "EvaluationParameters", map[string]any{"NumberOfBacktestWindows": 1.0, "BackTestWindowOffset": horizon})
	doc["IsAutoPredictor"] = false
	doc["DatasetGroupArn"] = group
	doc["ForecastFrequency"] = fcStr(fcObj(doc, "FeaturizationConfig"), "ForecastFrequency")
	doc["ForecastDimensions"] = fcObj(doc, "FeaturizationConfig")["ForecastDimensions"]
	return group, nil, nil
}

func (st *forecastStore) prepareAutoPredictor(doc map[string]any) (string, []string, *fcErr) {
	if ref := fcStr(doc, "ReferencePredictorArn"); ref != "" {
		for _, key := range []string{"DataConfig", "ForecastHorizon", "ForecastFrequency", "ForecastTypes", "ForecastDimensions"} {
			if doc[key] != nil {
				return "", nil, fcInvalid("%s can't be set when retraining with ReferencePredictorArn", key)
			}
		}
		p, err := st.active(fcKindByName["Predictor"], ref)
		if err != nil {
			return "", nil, err
		}
		if !fcBool(p.doc, "IsAutoPredictor") {
			return "", nil, fcInvalid("ReferencePredictorArn must name an auto predictor")
		}
		for _, key := range []string{"DataConfig", "ForecastHorizon", "ForecastFrequency", "ForecastTypes", "ForecastDimensions", "OptimizationMetric", "TimeAlignmentBoundary"} {
			if v, ok := p.doc[key]; ok {
				doc[key] = v
			}
		}
		doc["ReferencePredictorSummary"] = map[string]any{"Arn": ref, "State": "Active"}
	}
	for _, key := range []string{"DataConfig", "ForecastHorizon", "ForecastFrequency"} {
		if doc[key] == nil {
			return "", nil, fcInvalid("%s is required", key)
		}
	}
	if err := fcCheckHorizon(doc); err != nil {
		return "", nil, err
	}
	group := fcStr(fcObj(doc, "DataConfig"), "DatasetGroupArn")
	if _, err := st.groupTarget(group); err != nil {
		return "", nil, err
	}
	fcDefault(doc, "ForecastTypes", fcAnyList(fcDefaultTypes))
	fcDefault(doc, "OptimizationMetric", "AverageWeightedQuantileLoss")
	doc["IsAutoPredictor"] = true
	doc["DatasetGroupArn"] = group
	delete(doc, "MonitorConfig")
	delete(doc, "ExplainPredictor")
	return group, nil, nil
}

func (st *forecastStore) prepareExplainability(doc map[string]any, parent string) *fcErr {
	r, err := st.get(nil, parent)
	if err != nil {
		return err
	}
	switch r.kind.name {
	case "Predictor":
		if !fcBool(r.doc, "IsAutoPredictor") {
			return fcInvalid("explainability needs an auto predictor or a forecast")
		}
	case "Forecast":
	default:
		return fcInvalid("ResourceArn must name a predictor or a forecast")
	}
	if r.status != "ACTIVE" {
		return fcInUse("%s is %s, not ACTIVE", parent, r.status)
	}
	cfg := fcObj(doc, "ExplainabilityConfig")
	if fcStr(cfg, "TimePointGranularity") == "SPECIFIC" && (fcStr(doc, "StartDateTime") == "" || fcStr(doc, "EndDateTime") == "") {
		return fcInvalid("StartDateTime and EndDateTime are required when TimePointGranularity is SPECIFIC")
	}
	if fcStr(cfg, "TimeSeriesGranularity") == "SPECIFIC" && (doc["DataSource"] == nil || doc["Schema"] == nil) {
		return fcInvalid("DataSource and Schema are required when TimeSeriesGranularity is SPECIFIC")
	}
	fcDefault(doc, "EnableVisualization", false)
	return nil
}

// view is the resource as Describe and List report it.
func (st *forecastStore) view(r *fcRes) map[string]any {
	out := make(map[string]any, len(r.doc)+8)
	for k, v := range r.doc {
		out[k] = v
	}
	out[r.kind.name+"Name"] = r.name
	out[r.kind.name+"Arn"] = r.arn
	out["Status"] = r.status
	out["CreationTime"] = fcEpoch(r.created)
	out["LastModificationTime"] = fcEpoch(r.modified)
	if r.status == "CREATE_IN_PROGRESS" {
		out["EstimatedTimeRemainingInMinutes"] = 1.0
	}
	switch r.kind.name {
	case "Predictor":
		out["DatasetImportJobArns"] = fcAnyList(st.groupImportJobs(fcStr(r.doc, "DatasetGroupArn")))
		if e := st.res[fcStr(r.doc, "ExplainabilityArn")]; e != nil {
			st.settle(e)
			out["ExplainabilityInfo"] = map[string]any{"ExplainabilityArn": e.arn, "Status": e.status}
		}
		if m := st.res[fcStr(r.doc, "MonitorArn")]; m != nil {
			st.settle(m)
			out["MonitorInfo"] = map[string]any{"MonitorArn": m.arn, "Status": m.status}
		}
	case "Monitor":
		if evals := st.evaluations(r); len(evals) > 0 {
			last := evals[len(evals)-1]
			out["LastEvaluationTime"] = last["EvaluationTime"]
			out["LastEvaluationState"] = last["EvaluationState"]
		}
		out["Baseline"] = map[string]any{"PredictorBaseline": map[string]any{"BaselineMetrics": []any{
			map[string]any{"Name": "WAPE", "Value": fcMetric(r.parent, "WAPE")},
			map[string]any{"Name": "RMSE", "Value": fcMetric(r.parent, "RMSE") * 10},
		}}}
	}
	return out
}

// groupImportJobs lists the ACTIVE import jobs of a dataset group's datasets.
func (st *forecastStore) groupImportJobs(groupArn string) []string {
	g := st.res[groupArn]
	if g == nil {
		return nil
	}
	var out []string
	for _, r := range st.sorted(fcKindByName["DatasetImportJob"]) {
		if fcContains(fcStrs(g.doc, "DatasetArns"), r.parent) && r.status == "ACTIVE" {
			out = append(out, r.arn)
		}
	}
	return out
}

func (st *forecastStore) describe(k *fcKind, arn string) (map[string]any, *fcErr) {
	r, err := st.get(k, arn)
	if err != nil {
		return nil, err
	}
	out := st.view(r)
	r.seen = true
	return out, nil
}

// sorted returns the resources of kind k, oldest first, settled.
func (st *forecastStore) sorted(k *fcKind) []*fcRes {
	var out []*fcRes
	for _, r := range st.res {
		if r.kind == k {
			st.settle(r)
			out = append(out, r)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].seq < out[j].seq })
	return out
}

func (st *forecastStore) list(k *fcKind, op string, in map[string]any) (map[string]any, *fcErr) {
	model := awsmodels.ForecastModel()
	outShape := model.Shapes[model.Operations[op].Output]
	listKey, summary := "", ""
	for key, ref := range outShape.Members {
		if s := model.Shapes[ref.Shape]; s.Type == "list" {
			listKey, summary = key, s.Member.Shape
		}
	}
	filters, err := fcFilters(in, model.Shapes[summary])
	if err != nil {
		return nil, err
	}
	var items []any
	for _, r := range st.sorted(k) {
		v := st.view(r)
		if fcMatch(v, filters) {
			items = append(items, v)
		}
	}
	page, next, err := fcPage(items, in, "NextToken", "MaxResults", 100)
	if err != nil {
		return nil, err
	}
	out := map[string]any{listKey: page}
	if next != "" {
		out["NextToken"] = next
	}
	return out, nil
}

type fcFilter struct{ key, value, cond string }

// fcFilters reads Filters, whose keys must be members of the summary shape.
func fcFilters(in map[string]any, summary *awsmodels.ModelShape) ([]fcFilter, *fcErr) {
	raw, _ := in["Filters"].([]any)
	if len(raw) > fcMaxFilters {
		return nil, fcInvalid("at most %d filters", fcMaxFilters)
	}
	var out []fcFilter
	for _, f := range raw {
		m, _ := f.(map[string]any)
		key := fcStr(m, "Key")
		if _, ok := summary.Members[key]; !ok && !strings.HasSuffix(key, "Arn") {
			return nil, fcInvalid("unsupported filter key %s", key)
		}
		out = append(out, fcFilter{key: key, value: fcStr(m, "Value"), cond: fcStr(m, "Condition")})
	}
	return out, nil
}

func fcMatch(v map[string]any, filters []fcFilter) bool {
	for _, f := range filters {
		got, _ := v[f.key].(string)
		if (got == f.value) != (f.cond == "IS") {
			return false
		}
	}
	return true
}

// fcPage pages items with an offset NextToken.
func fcPage(items []any, in map[string]any, tokenKey, maxKey string, def int) ([]any, string, *fcErr) {
	start := 0
	if tok := fcStr(in, tokenKey); tok != "" {
		n, err := strconv.ParseInt(tok, 10, 32)
		if err != nil || n < 0 || int(n) > len(items) {
			return nil, "", fcError("InvalidNextTokenException", "invalid NextToken")
		}
		start = int(n)
	}
	limit := def
	if n, ok := fcNum(in, maxKey); ok && n >= 1 && n < float64(def) {
		limit = int(n)
	}
	end := min(start+limit, len(items))
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	page := items[start:end]
	if page == nil {
		page = []any{}
	}
	return page, next, nil
}

// children returns the resources whose parent (or references) is arn.
func (st *forecastStore) children(arn string) []*fcRes {
	var out []*fcRes
	for _, r := range st.res {
		if r.parent == arn || fcContains(r.refs, arn) {
			out = append(out, r)
		}
	}
	return out
}

func fcDeletable(status string) bool {
	switch status {
	case "ACTIVE", "CREATE_FAILED", "CREATE_STOPPED", "ACTIVE_STOPPED", "UPDATE_FAILED":
		return true
	}
	return false
}

func (st *forecastStore) delete(k *fcKind, arn string) *fcErr {
	r, err := st.get(k, arn)
	if err != nil {
		return err
	}
	if !fcDeletable(r.status) {
		return fcInUse("%s is %s and can't be deleted", arn, r.status)
	}
	if kids := st.children(arn); len(kids) > 0 {
		return fcInUse("%s is used by %s; delete it first or use DeleteResourceTree", arn, kids[0].arn)
	}
	st.remove(r)
	return nil
}

func (st *forecastStore) remove(r *fcRes) {
	st.bytes -= r.size
	delete(st.res, r.arn)
	delete(st.tags, r.arn)
}

func (st *forecastStore) deleteTree(arn string) *fcErr {
	root, err := st.get(nil, arn)
	if err != nil {
		return err
	}
	tree := []*fcRes{root}
	seen := map[string]bool{arn: true}
	for i := 0; i < len(tree); i++ { // bounded by the store size
		for _, kid := range st.children(tree[i].arn) {
			if !seen[kid.arn] {
				seen[kid.arn] = true
				st.settle(kid)
				tree = append(tree, kid)
			}
		}
	}
	for _, r := range tree {
		if !fcDeletable(r.status) {
			return fcInUse("%s is %s and can't be deleted", r.arn, r.status)
		}
	}
	for _, r := range tree {
		st.remove(r)
	}
	return nil
}

func (st *forecastStore) updateDatasetGroup(in map[string]any) *fcErr {
	g, err := st.get(fcKindByName["DatasetGroup"], fcStr(in, "DatasetGroupArn"))
	if err != nil {
		return err
	}
	arns := fcStrs(in, "DatasetArns")
	if err := st.checkGroupDatasets(fcStr(g.doc, "Domain"), arns); err != nil {
		return err
	}
	doc := make(map[string]any, len(g.doc))
	for k, v := range g.doc {
		doc[k] = v
	}
	doc["DatasetArns"] = fcAnyList(arns)
	g.doc = doc
	g.modified = time.Now().UTC()
	return nil
}

func (st *forecastStore) stop(arn string) *fcErr {
	r, err := st.get(nil, arn)
	if err != nil {
		return err
	}
	switch {
	case r.status == "CREATE_IN_PROGRESS":
		r.status = "CREATE_STOPPED"
	case r.kind.name == "Monitor" && r.status == "ACTIVE":
		r.status = "ACTIVE_STOPPED"
	default:
		return fcInvalid("%s is %s and can't be stopped", arn, r.status)
	}
	r.modified = time.Now().UTC()
	return nil
}

func (st *forecastStore) resume(arn string) *fcErr {
	r, err := st.get(nil, arn)
	if err != nil {
		return err
	}
	if r.kind.name != "Monitor" {
		return fcInvalid("only monitors can be resumed")
	}
	if r.status != "ACTIVE_STOPPED" {
		return fcInUse("monitor %s is %s, not ACTIVE_STOPPED", arn, r.status)
	}
	r.status = "ACTIVE"
	r.modified = time.Now().UTC()
	return nil
}

// fcTagMap merges a Tags list into a copy of base.
func fcTagMap(base map[string]string, raw any) (map[string]string, *fcErr) {
	out := make(map[string]string, len(base))
	for k, v := range base {
		out[k] = v
	}
	items, _ := raw.([]any)
	for _, item := range items {
		m, _ := item.(map[string]any)
		key := fcStr(m, "Key")
		if strings.HasPrefix(strings.ToLower(key), "aws:") {
			return nil, fcInvalid("tag keys can't start with aws:")
		}
		out[key] = fcStr(m, "Value")
	}
	if len(out) > fcMaxTags {
		return nil, fcError("LimitExceededException", "a resource can have at most %d tags", fcMaxTags)
	}
	return out, nil
}

func (st *forecastStore) tag(arn string, raw any) *fcErr {
	if _, err := st.get(nil, arn); err != nil {
		return err
	}
	tags, err := fcTagMap(st.tags[arn], raw)
	if err != nil {
		return err
	}
	st.tags[arn] = tags
	return nil
}

func (st *forecastStore) untag(arn string, keys []string) *fcErr {
	if _, err := st.get(nil, arn); err != nil {
		return err
	}
	tags, _ := fcTagMap(st.tags[arn], nil)
	for _, k := range keys {
		delete(tags, k)
	}
	st.tags[arn] = tags
	return nil
}

func (st *forecastStore) listTags(arn string) (map[string]any, *fcErr) {
	if _, err := st.get(nil, arn); err != nil {
		return nil, err
	}
	keys := make([]string, 0, len(st.tags[arn]))
	for k := range st.tags[arn] {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	tags := make([]any, len(keys))
	for i, k := range keys {
		tags[i] = map[string]any{"Key": k, "Value": st.tags[arn][k]}
	}
	return map[string]any{"Tags": tags}, nil
}
