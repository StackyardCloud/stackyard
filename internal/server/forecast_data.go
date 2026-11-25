package server

import (
	"hash/fnv"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
)

// ponytail: no model is trained. Metrics and forecast values are synthetic
// but deterministic per resource ARN (and, for queries, per filter set), so
// repeated reads agree.

func fcSeed(parts ...string) uint64 {
	h := fnv.New64a()
	for _, p := range parts {
		_, _ = h.Write([]byte(p))
		_, _ = h.Write([]byte{0})
	}
	return h.Sum64()
}

// fcMetric is a deterministic error metric in [0.05, 0.35).
func fcMetric(arn, name string) float64 {
	return 0.05 + float64(fcSeed(arn, name)%3000)/10000
}

func fcRound(v float64) float64 { return math.Round(v*10000) / 10000 }

func (st *forecastStore) accuracyMetrics(arn string) (map[string]any, *fcErr) {
	p, err := st.get(fcKindByName["Predictor"], arn)
	if err != nil {
		return nil, err
	}
	if p.status != "ACTIVE" {
		return nil, fcInUse("predictor %s is %s, not ACTIVE", arn, p.status)
	}
	algos := fcStrs(p.doc, "AutoMLAlgorithmArns")
	if len(algos) == 0 {
		algos = []string{fcStr(p.doc, "AlgorithmArn")}
	}
	if fcBool(p.doc, "IsAutoPredictor") {
		algos = []string{fcAlgorithmArn("AutoPredictor")}
	}
	windows := 1
	if n, ok := fcNum(fcObj(p.doc, "EvaluationParameters"), "NumberOfBacktestWindows"); ok && n >= 1 && n <= 5 {
		windows = int(n)
	}
	end := p.created.Truncate(24 * time.Hour)
	results := make([]any, 0, len(algos))
	for _, algo := range algos {
		key := arn + "|" + algo
		tw := []any{fcWindow(key, "SUMMARY", p, end.AddDate(0, 0, -7*windows), end)}
		for i := 0; i < windows; i++ {
			s := end.AddDate(0, 0, -7*(i+1))
			tw = append(tw, fcWindow(key+strconv.Itoa(i), "COMPUTED", p, s, s.AddDate(0, 0, 7)))
		}
		results = append(results, map[string]any{"AlgorithmArn": algo, "TestWindows": tw})
	}
	out := map[string]any{"PredictorEvaluationResults": results, "IsAutoPredictor": fcBool(p.doc, "IsAutoPredictor")}
	if v := fcStr(p.doc, "OptimizationMetric"); v != "" {
		out["OptimizationMetric"] = v
	}
	if v := fcStr(p.doc, "AutoMLOverrideStrategy"); v != "" {
		out["AutoMLOverrideStrategy"] = v
	}
	return out, nil
}

func fcWindow(key, kind string, p *fcRes, start, end time.Time) map[string]any {
	var losses, errs []any
	sum := 0.0
	types := fcStrs(p.doc, "ForecastTypes")
	for _, t := range types {
		loss := fcRound(fcMetric(key, t))
		sum += loss
		if q, err := strconv.ParseFloat(t, 64); err == nil {
			losses = append(losses, map[string]any{"Quantile": q, "LossValue": loss})
		}
		errs = append(errs, map[string]any{"ForecastType": t, "WAPE": fcRound(fcMetric(key, "WAPE"+t)),
			"RMSE": fcRound(fcMetric(key, "RMSE"+t) * 100), "MASE": fcRound(fcMetric(key, "MASE"+t) * 4), "MAPE": fcRound(fcMetric(key, "MAPE"+t))})
	}
	metrics := map[string]any{"RMSE": fcRound(fcMetric(key, "RMSE") * 100), "WeightedQuantileLosses": losses, "ErrorMetrics": errs}
	if len(types) > 0 {
		metrics["AverageWeightedQuantileLoss"] = fcRound(sum / float64(len(types)))
	}
	return map[string]any{"TestWindowStart": fcEpoch(start), "TestWindowEnd": fcEpoch(end),
		"ItemCount": float64(10 + fcSeed(key)%90), "EvaluationType": kind, "Metrics": metrics}
}

// evaluations: a monitor evaluates its predictor each time new data finishes
// importing into the predictor's dataset group after the monitor was created.
func (st *forecastStore) evaluations(m *fcRes) []map[string]any {
	p := st.res[m.parent]
	if p == nil || m.status != "ACTIVE" && m.status != "ACTIVE_STOPPED" {
		return nil
	}
	var out []map[string]any
	for _, arn := range st.groupImportJobs(fcStr(p.doc, "DatasetGroupArn")) {
		job := st.res[arn]
		if job.created.Before(m.created) {
			continue
		}
		at := job.modified
		out = append(out, map[string]any{
			"ResourceArn": p.arn, "MonitorArn": m.arn, "EvaluationTime": fcEpoch(at), "EvaluationState": "SUCCESS",
			"WindowStartDatetime": fcEpoch(at.AddDate(0, 0, -7)), "WindowEndDatetime": fcEpoch(at),
			"MonitorDataSource": map[string]any{"DatasetImportJobArn": arn, "PredictorArn": p.arn},
			"NumItemsEvaluated": float64(10 + fcSeed(arn)%90),
			"MetricResults": []any{
				map[string]any{"MetricName": "WAPE", "MetricValue": fcRound(fcMetric(arn, "WAPE"))},
				map[string]any{"MetricName": "RMSE", "MetricValue": fcRound(fcMetric(arn, "RMSE") * 100)},
			},
		})
	}
	return out
}

func (st *forecastStore) monitorEvaluations(in map[string]any) (map[string]any, *fcErr) {
	m, err := st.get(fcKindByName["Monitor"], fcStr(in, "MonitorArn"))
	if err != nil {
		return nil, err
	}
	var items []any
	for _, raw := range fcObjList(in["Filters"]) {
		if fcStr(raw, "Key") != "EvaluationState" {
			return nil, fcInvalid("the only filter key is EvaluationState")
		}
	}
	for _, e := range st.evaluations(m) {
		keep := true
		for _, f := range fcObjList(in["Filters"]) {
			if (e["EvaluationState"] == fcStr(f, "Value")) != (fcStr(f, "Condition") == "IS") {
				keep = false
			}
		}
		if keep {
			items = append(items, e)
		}
	}
	page, next, ferr := fcPage(items, in, "NextToken", "MaxResults", 20)
	if ferr != nil {
		return nil, ferr
	}
	out := map[string]any{"PredictorMonitorEvaluations": page}
	if next != "" {
		out["NextToken"] = next
	}
	return out, nil
}

func fcObjList(raw any) []map[string]any {
	items, _ := raw.([]any)
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		if m, ok := item.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out
}

const fcqTimeLayout = "2006-01-02T15:04:05"

// fcStep advances t by one forecast period.
func fcStep(t time.Time, freq string) time.Time {
	switch freq {
	case "Y":
		return t.AddDate(1, 0, 0)
	case "M":
		return t.AddDate(0, 1, 0)
	case "W":
		return t.AddDate(0, 0, 7)
	case "H":
		return t.Add(time.Hour)
	case "D", "":
		return t.AddDate(0, 0, 1)
	}
	if n, err := strconv.ParseInt(strings.TrimSuffix(freq, "min"), 10, 32); err == nil && n > 0 {
		return t.Add(time.Duration(n) * time.Minute)
	}
	return t.AddDate(0, 0, 1)
}

// fcStatName is the Predictions key for a forecast type: mean, or p<percent>.
func fcStatName(t string) (string, float64) {
	q, err := strconv.ParseFloat(t, 64)
	if err != nil {
		return t, 0.5
	}
	return "p" + strconv.FormatFloat(math.Round(q*1000)/10, 'f', -1, 64), q
}

// query answers QueryForecast (whatIf false) and QueryWhatIfForecast.
func (st *forecastStore) query(arn string, whatIf bool, in map[string]any) (map[string]any, *fcErr) {
	kind := "Forecast"
	if whatIf {
		kind = "WhatIfForecast"
	}
	r, err := st.get(fcKindByName[kind], arn)
	if err != nil {
		return nil, err
	}
	if r.status != "ACTIVE" {
		return nil, fcInUse("%s %s is %s, not ACTIVE", kind, arn, r.status)
	}
	if fcStr(in, "NextToken") != "" {
		return nil, fcError("InvalidNextTokenException", "invalid NextToken")
	}
	f := r
	if whatIf {
		if a := st.res[r.parent]; a != nil {
			f = st.res[a.parent]
		}
	}
	if f == nil || st.res[f.parent] == nil {
		return nil, fcNotFound("the forecast's predictor no longer exists")
	}
	p := st.res[f.parent]
	filters := fcObj(in, "Filters")
	if ds, _ := st.groupTarget(fcStr(p.doc, "DatasetGroupArn")); ds != nil {
		attrs := fcSchemaAttrs(ds.doc)
		for key := range filters {
			if t, ok := attrs[key]; !ok || t == "timestamp" {
				return nil, fcInvalid("filter key %s is not an attribute of the target time series", key)
			}
		}
	}
	start, end, ferr := fcqWindow(in)
	if ferr != nil {
		return nil, ferr
	}
	keys := make([]string, 0, len(filters))
	for k := range filters {
		keys = append(keys, k+"="+fcStr(filters, k))
	}
	sort.Strings(keys)
	seed := fcSeed(append([]string{r.arn}, keys...)...)
	base := 20 + float64(seed%80)
	horizon, _ := fcNum(p.doc, "ForecastHorizon")
	freq := fcStr(p.doc, "ForecastFrequency")
	t0 := f.created.UTC().Truncate(24 * time.Hour)
	preds := map[string]any{}
	for _, typ := range fcStrs(r.doc, "ForecastTypes") {
		name, q := fcStatName(typ)
		points := []any{}
		t := t0
		for i := 0; i < int(min(horizon, fcMaxHorizon)); i++ {
			if (start.IsZero() || !t.Before(start)) && (end.IsZero() || !t.After(end)) {
				v := base * (1 + 0.1*math.Sin(float64(i)+float64(seed%7))) * (0.6 + 0.8*q)
				points = append(points, map[string]any{"Timestamp": t.Format(fcqTimeLayout), "Value": fcRound(v)})
			}
			t = fcStep(t, freq)
		}
		preds[name] = points
	}
	return map[string]any{"Forecast": map[string]any{"Predictions": preds}}, nil
}
