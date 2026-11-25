package server

import (
	"encoding/json"
	"math"
	"net/http"
	"sort"
	"time"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

// Amazon SageMaker Metrics (metrics.sagemaker, REST JSON, signed
// "sagemaker"). Metrics are stored on SageMaker trial components
// (sagemaker_trials.go) — including the "<job>-aws-training-job" component of
// a training job — and DescribeTrialComponent summarises them.
//
// API reference:
// https://docs.aws.amazon.com/sagemaker/latest/APIReference/API_Operations_Amazon_SageMaker_Metrics_Service.html

const (
	smmMaxBody = 1 << 20
	smmHost    = "metrics.sagemaker."
	// smmMaxResultPoints caps the points of one query result ("Truncated").
	smmMaxResultPoints = 10000
)

var smmPeriods = map[string]float64{"OneMinute": 60, "FiveMinute": 300, "OneHour": 3600}

func (s *Server) handleSageMakerMetricsRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !sagemakerDataPlaneClaims(r, smmHost) {
		return false
	}
	op := ""
	switch path := rawRequestPath(r); {
	case r.Method == http.MethodPut && path == "/BatchPutMetrics":
		op = "BatchPutMetrics"
	case r.Method == http.MethodPost && path == "/BatchGetMetrics":
		op = "BatchGetMetrics"
	default:
		return false
	}
	body, ok := s.runSageMakerDataPlane(w, r, awsmodels.SageMakerMetricsModel(), op, smmMaxBody, func(raw []byte) (any, *sagemakerDataError) {
		if op == "BatchPutMetrics" {
			var req smmPutReq
			if err := json.Unmarshal(raw, &req); err != nil {
				return nil, smfsInvalid("invalid request: %v", err)
			}
			return s.sagemaker.smmPut(req, time.Now().UTC())
		}
		var req smmGetReq
		if err := json.Unmarshal(raw, &req); err != nil {
			return nil, smfsInvalid("invalid request: %v", err)
		}
		return s.sagemaker.smmGet(req, time.Now().UTC()), nil
	})
	if ok {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}
	return true
}

type smmPutReq struct {
	TrialComponentName string
	MetricData         []struct {
		MetricName string
		Timestamp  sagemakerTimestamp
		Step       *int64
		Value      float64
	}
}

func (s *sagemakerStore) smmPut(req smmPutReq, now time.Time) (any, *sagemakerDataError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.initResourcesLocked()
	c, err := s.smmComponentLocked(req.TrialComponentName, now)
	if err != nil {
		return nil, smfsInvalid("%s", err.msg)
	}
	errs := []any{}
	for i, d := range req.MetricData {
		series := c.metrics[d.MetricName]
		if series == nil && len(c.metricOrder) >= smmMaxMetricsPerResource ||
			series != nil && len(series.points) >= smmMaxPointsPerMetric || s.smmPoints >= smmMaxPoints {
			errs = append(errs, map[string]any{"Code": "METRIC_LIMIT_EXCEEDED", "MetricIndex": i})
			continue
		}
		if series == nil {
			series = &smmSeries{}
			c.metrics[d.MetricName] = series
			c.metricOrder = append(c.metricOrder, d.MetricName)
		}
		p := smmPoint{at: sagemakerEpoch(d.Timestamp.Time), value: d.Value}
		if d.Step != nil {
			p.step, p.hasStep = *d.Step, true
		}
		series.add(p)
		s.smmPoints++
	}
	c.Modified = now
	if len(errs) == 0 {
		return nil, nil
	}
	return map[string]any{"Errors": errs}, nil
}

type smmQuery struct {
	MetricName, ResourceArn, MetricStat, Period, XAxisType string
	Start, End                                             *int64
}

type smmGetReq struct {
	MetricQueries []smmQuery
}

// smmGet snapshots each query's points under the lock (stored points are
// append-only, so points[:n] stays valid) and aggregates after releasing it.
func (s *sagemakerStore) smmGet(req smmGetReq, now time.Time) any {
	snaps := make([][]smmPoint, len(req.MetricQueries))
	problems := make([]string, len(req.MetricQueries))
	s.mu.Lock()
	s.initResourcesLocked()
	for i, q := range req.MetricQueries {
		c, err := s.smmComponentLocked(q.ResourceArn, now)
		switch {
		case err != nil:
			problems[i] = err.msg
		case c.metrics[q.MetricName] == nil:
			problems[i] = "Metric " + q.MetricName + " has no data for " + q.ResourceArn + "."
		default:
			pts := c.metrics[q.MetricName].points
			snaps[i] = pts[:len(pts):len(pts)]
		}
	}
	s.mu.Unlock()
	results := make([]any, 0, len(req.MetricQueries))
	for i, q := range req.MetricQueries {
		if problems[i] == "" && q.Period == "IterationNumber" && q.XAxisType == "Timestamp" {
			problems[i] = "Period IterationNumber needs XAxisType IterationNumber."
		}
		if problems[i] != "" {
			results = append(results, map[string]any{"Status": "ValidationError", "Message": problems[i],
				"XAxisValues": []int64{}, "MetricValues": []float64{}})
			continue
		}
		results = append(results, smmAggregate(q, snaps[i]))
	}
	return map[string]any{"MetricQueryResults": results}
}

type smmBucket struct {
	x                    int64
	count                int
	sum, sumSq, min, max float64
	last, lastAt         float64
}

func (b *smmBucket) stat(name string) float64 {
	mean := b.sum / float64(b.count)
	switch name {
	case "Min":
		return b.min
	case "Max":
		return b.max
	case "Count":
		return float64(b.count)
	case "StdDev":
		return math.Sqrt(math.Max(0, b.sumSq/float64(b.count)-mean*mean))
	case "Last":
		return b.last
	}
	return mean
}

// smmAggregate groups points by Period (a time window or an iteration) and
// reports MetricStat per group, keyed on the XAxisType value.
func smmAggregate(q smmQuery, pts []smmPoint) map[string]any {
	period := smmPeriods[q.Period]
	byStep := q.XAxisType == "IterationNumber"
	buckets := map[int64]*smmBucket{}
	for _, p := range pts {
		if (period == 0 || byStep) && !p.hasStep {
			continue
		}
		x := int64(p.at)
		if byStep {
			x = p.step
		}
		if (q.Start != nil && x < *q.Start) || (q.End != nil && x > *q.End) {
			continue
		}
		key := p.step
		if period > 0 {
			key = int64(math.Floor(p.at/period)) * int64(period)
		}
		b := buckets[key]
		if b == nil {
			b = &smmBucket{x: x, min: p.value, max: p.value}
			if !byStep && period > 0 {
				b.x = key
			}
			buckets[key] = b
		}
		if x > b.x && (byStep || period == 0) {
			b.x = x
		}
		b.count++
		b.sum += p.value
		b.sumSq += p.value * p.value
		b.min, b.max = math.Min(b.min, p.value), math.Max(b.max, p.value)
		if b.count == 1 || p.at >= b.lastAt {
			b.last, b.lastAt = p.value, p.at
		}
	}
	keys := make([]int64, 0, len(buckets))
	for k := range buckets {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	status := "Complete"
	if len(keys) > smmMaxResultPoints {
		keys, status = keys[:smmMaxResultPoints], "Truncated"
	}
	xs, vs := make([]int64, 0, len(keys)), make([]float64, 0, len(keys))
	for _, k := range keys {
		xs = append(xs, buckets[k].x)
		vs = append(vs, buckets[k].stat(q.MetricStat))
	}
	return map[string]any{"Status": status, "XAxisValues": xs, "MetricValues": vs}
}
