package server

import (
	"hash/fnv"
	"math"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Performance Insights metric data is synthetic but deterministic: every value
// is a function of the identifier, metric, dimension key and timestamp, so
// repeated calls over the same window agree.

var piPeriods = []int{1, 60, 300, 3600, 86400}

const (
	piMaxPoints   = 1440                 // data points per series
	piMaxRange    = 731 * 24 * time.Hour // the longest retention period AWS offers
	piTargetPts   = 200                  // automatic period choice aims at or below this
	piDefaultKeys = 10                   // DimensionGroup.Limit default
)

type piWindow struct {
	start, end int64 // aligned epoch seconds
	period     int
	endAligned bool // timestamps mark the end of each period
}

func (w piWindow) points() int { return int((w.end - w.start) / int64(w.period)) }

func (w piWindow) stamp(i int) int64 {
	t := w.start + int64(i)*int64(w.period)
	if w.endAligned {
		t += int64(w.period)
	}
	return t
}

func piWindowOf(in map[string]any) (piWindow, *piErr) {
	start, _, err := piTime(in, "StartTime")
	if err != nil {
		return piWindow{}, err
	}
	end, _, err := piTime(in, "EndTime")
	if err != nil {
		return piWindow{}, err
	}
	if !end.After(start) {
		return piWindow{}, piInvalid("StartTime must be earlier than EndTime.")
	}
	if end.Sub(start) > piMaxRange {
		return piWindow{}, piInvalid("The time range can't be longer than %d days.", int(piMaxRange.Hours()/24))
	}
	span := int64(end.Sub(start).Seconds())
	period := piInt(in, "PeriodInSeconds", 0)
	switch {
	case period == 0:
		period = piPeriods[len(piPeriods)-1]
		for _, p := range piPeriods {
			if span/int64(p) <= piTargetPts {
				period = p
				break
			}
		}
	case !slices.Contains(piPeriods, period):
		return piWindow{}, piInvalid("PeriodInSeconds must be one of 1, 60, 300, 3600 or 86400.")
	}
	p := int64(period)
	w := piWindow{start: start.Unix() / p * p, end: (end.Unix() + p - 1) / p * p, period: period, endAligned: piStr(in, "PeriodAlignment") != "START_TIME"}
	if w.end == w.start {
		w.end += p
	}
	if w.points() > piMaxPoints {
		return piWindow{}, piInvalid("The request covers %d data points; use a larger PeriodInSeconds (at most %d points).", w.points(), piMaxPoints)
	}
	return w, nil
}

func piHash(parts ...string) uint64 {
	h := fnv.New64a()
	h.Write([]byte(strings.Join(parts, "|")))
	return h.Sum64()
}

// piValue is the metric's value at epoch second t, per stat.
func piValue(id string, def piMetricDef, stat string, t int64, period int) float64 {
	h := piHash(id, def.name)
	base := def.scale * (0.5 + float64(h%1000)/1000)
	phase := float64((h>>10)%6283) / 1000
	daily := math.Sin(2*math.Pi*float64(t)/86400 + phase)
	noise := float64(piHash(id, def.name, strconv.FormatInt(t/int64(period), 10))%1000)/1000 - 0.5
	v := max(0, base*(1+0.3*daily+0.2*noise))
	switch stat {
	case "min":
		v *= 0.8
	case "max":
		v *= 1.25
	case "sum":
		v *= float64(period)
	}
	if def.unit == "Percent" {
		v = min(v, 100)
	}
	return math.Round(v*1000) / 1000
}

// piShares splits a total across n keys: a fixed, descending share per key.
func piShares(id, group string, n int) []float64 {
	w := make([]float64, n)
	sum := 0.0
	for i := range w {
		w[i] = float64(n-i) + float64(piHash(id, group, strconv.Itoa(i))%100)/200
		sum += w[i]
	}
	for i := range w {
		w[i] /= sum
	}
	return w
}

// piKey is one dimension key of a group with its share of the load.
type piKey struct {
	values []string
	share  float64
}

// piKeysOf returns the group's keys that match the filter, by descending share.
func piKeysOf(id, name string, g piGroup, filter map[string]string) []piKey {
	shares := piShares(id, name, len(g.keys))
	var out []piKey
	for i, vals := range g.keys {
		match := true
		for d, want := range filter {
			if j := slices.Index(g.dims, d); j >= 0 && vals[j] != want {
				match = false
			}
		}
		if match {
			out = append(out, piKey{values: vals, share: shares[i]})
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].share > out[j].share })
	return out
}

// piGroupSpec validates a DimensionGroup and returns the group, the dimensions
// to report, and its key limit.
func piGroupSpec(serviceType string, spec map[string]any) (string, piGroup, []string, int, *piErr) {
	name := piStr(spec, "Group")
	g, ok := piGroups(serviceType)[name]
	if !ok {
		return "", piGroup{}, nil, 0, piInvalid("%s is not a valid dimension group for %s.", name, serviceType)
	}
	dims := piList(spec, "Dimensions")
	for _, d := range dims {
		if !slices.Contains(g.dims, d) {
			return "", piGroup{}, nil, 0, piInvalid("%s is not a dimension of %s.", d, name)
		}
	}
	if len(dims) == 0 {
		dims = g.dims
	}
	return name, g, dims, piInt(spec, "Limit", piDefaultKeys), nil
}

// piFilter checks that every filter dimension belongs to one of the groups.
func piFilter(in map[string]any, groups ...piGroup) (map[string]string, *piErr) {
	out := map[string]string{}
	for d, v := range piMap(in, "Filter") {
		known := false
		for _, g := range groups {
			known = known || slices.Contains(g.dims, d)
		}
		if !known {
			return nil, piInvalid("Filter dimension %s isn't part of the GroupBy or PartitionBy group.", d)
		}
		out[d], _ = v.(string)
	}
	return out, nil
}

func piDimensions(g piGroup, dims []string, values []string) map[string]any {
	out := map[string]any{}
	for _, d := range dims {
		out[d] = values[slices.Index(g.dims, d)]
	}
	return out
}

// piMean is the metric's average over the window, the Total of a dimension key.
func piMean(id string, def piMetricDef, stat string, w piWindow) float64 {
	sum := 0.0
	for i := 0; i < w.points(); i++ {
		sum += piValue(id, def, stat, w.stamp(i), w.period)
	}
	return sum / float64(w.points())
}

func piRound(v float64) float64 { return math.Round(v*1000) / 1000 }

func (s *Server) piDescribeDimensionKeys(in map[string]any) (map[string]any, *piErr) {
	serviceType, id, err := s.piResolve(in)
	if err != nil {
		return nil, err
	}
	metric := piStr(in, "Metric")
	def, load, ok := piMetric(metric)
	if !ok || !load {
		return nil, piInvalid("Metric must be db.load.avg or db.sampledload.avg.")
	}
	w, err := piWindowOf(in)
	if err != nil {
		return nil, err
	}
	stat := metric[strings.LastIndex(metric, ".")+1:]
	gname, g, dims, limit, err := piGroupSpec(serviceType, piMap(in, "GroupBy"))
	if err != nil {
		return nil, err
	}
	groups := []piGroup{g}
	var pname string
	var pg piGroup
	var pdims []string
	plimit := 0
	if pb := piMap(in, "PartitionBy"); pb != nil {
		if pname, pg, pdims, plimit, err = piGroupSpec(serviceType, pb); err != nil {
			return nil, err
		}
		groups = append(groups, pg)
	}
	filter, err := piFilter(in, groups...)
	if err != nil {
		return nil, err
	}
	for _, m := range piList(in, "AdditionalMetrics") {
		if !strings.HasPrefix(m, "db.") {
			return nil, piInvalid("%s is not a valid additional metric.", m)
		}
	}
	total := piMean(id, def, stat, w)
	keys := piKeysOf(id, gname, g, filter)
	keys = keys[:min(len(keys), limit)]
	var parts []piKey
	if pname != "" {
		parts = piKeysOf(id, pname, pg, filter)
		parts = parts[:min(len(parts), plimit)]
	}
	start, end, next, err := piPage(in, len(keys), piDefaultKeys)
	if err != nil {
		return nil, err
	}
	out := []any{}
	for _, k := range keys[start:end] {
		desc := map[string]any{"Dimensions": piDimensions(g, dims, k.values), "Total": piRound(total * k.share)}
		if extra := piList(in, "AdditionalMetrics"); len(extra) > 0 {
			m := map[string]any{}
			for _, name := range extra {
				m[name] = piRound(float64(piHash(id, name, strings.Join(k.values, ","))%10000) / 100)
			}
			desc["AdditionalMetrics"] = m
		}
		if len(parts) > 0 {
			vals := []any{}
			for _, p := range parts {
				vals = append(vals, piRound(total*k.share*p.share))
			}
			desc["Partitions"] = vals
		}
		out = append(out, desc)
	}
	resp := map[string]any{"AlignedStartTime": float64(w.start), "AlignedEndTime": float64(w.end), "Keys": out}
	if len(parts) > 0 {
		pkeys := []any{}
		for _, p := range parts {
			pkeys = append(pkeys, map[string]any{"Dimensions": piDimensions(pg, pdims, p.values)})
		}
		resp["PartitionKeys"] = pkeys
	}
	if next != "" {
		resp["NextToken"] = next
	}
	return resp, nil
}

// piSeriesSpec is one MetricKeyDataPoints entry before its points are built.
type piSeriesSpec struct {
	metric, stat string
	def          piMetricDef
	dims         map[string]any
	share        float64
}

func (s *Server) piGetResourceMetrics(in map[string]any) (map[string]any, *piErr) {
	serviceType, id, err := s.piResolve(in)
	if err != nil {
		return nil, err
	}
	w, err := piWindowOf(in)
	if err != nil {
		return nil, err
	}
	var specs []piSeriesSpec
	for _, q := range piAnyList(in["MetricQueries"]) {
		query, _ := q.(map[string]any)
		metric := piStr(query, "Metric")
		def, load, ok := piMetric(metric)
		if !ok {
			return nil, piInvalid("%s is not a valid metric.", metric)
		}
		stat := metric[strings.LastIndex(metric, ".")+1:]
		specs = append(specs, piSeriesSpec{metric: metric, stat: stat, def: def, share: 1})
		gb := piMap(query, "GroupBy")
		if gb == nil {
			if len(piMap(query, "Filter")) > 0 {
				return nil, piInvalid("Filter requires a GroupBy on %s.", metric)
			}
			continue
		}
		if !load {
			return nil, piInvalid("GroupBy is only supported for db.load and db.sampledload metrics.")
		}
		gname, g, dims, limit, err := piGroupSpec(serviceType, gb)
		if err != nil {
			return nil, err
		}
		filter, err := piFilter(query, g)
		if err != nil {
			return nil, err
		}
		keys := piKeysOf(id, gname, g, filter)
		for _, k := range keys[:min(len(keys), limit)] {
			specs = append(specs, piSeriesSpec{metric: metric, stat: stat, def: def, dims: piDimensions(g, dims, k.values), share: k.share})
		}
	}
	start, end, next, err := piPage(in, len(specs), 25)
	if err != nil {
		return nil, err
	}
	list := []any{}
	for _, sp := range specs[start:end] {
		points := make([]any, 0, w.points())
		for i := 0; i < w.points(); i++ {
			t := w.stamp(i)
			points = append(points, map[string]any{"Timestamp": float64(t), "Value": piRound(piValue(id, sp.def, sp.stat, t, w.period) * sp.share)})
		}
		key := map[string]any{"Metric": sp.metric}
		if sp.dims != nil {
			key["Dimensions"] = sp.dims
		}
		list = append(list, map[string]any{"Key": key, "DataPoints": points})
	}
	resp := map[string]any{"AlignedStartTime": float64(w.start), "AlignedEndTime": float64(w.end), "Identifier": id, "MetricList": list}
	if next != "" {
		resp["NextToken"] = next
	}
	return resp, nil
}

func (s *Server) piGetDimensionKeyDetails(in map[string]any) (map[string]any, *piErr) {
	serviceType, _, err := s.piResolve(in)
	if err != nil {
		return nil, err
	}
	name := piStr(in, "Group")
	detail, ok := piDetailGroups[name]
	g, inService := piGroups(serviceType)[name]
	if !ok || !inService {
		return nil, piInvalid("GetDimensionKeyDetails doesn't support the %s group for %s.", name, serviceType)
	}
	requested := piList(in, "RequestedDimensions")
	if len(requested) == 0 {
		requested = []string{detail[1]}
	}
	for _, d := range requested {
		if !slices.Contains(g.dims, d) {
			return nil, piInvalid("%s is not a dimension of %s.", d, name)
		}
	}
	idCol := slices.Index(g.dims, detail[0])
	for _, vals := range g.keys {
		if vals[idCol] != piStr(in, "GroupIdentifier") {
			continue
		}
		out := []any{}
		for _, d := range requested {
			out = append(out, map[string]any{"Dimension": d, "Value": vals[slices.Index(g.dims, d)], "Status": "AVAILABLE"})
		}
		return map[string]any{"Dimensions": out}, nil
	}
	return nil, piInvalid("No %s key has the identifier %s.", name, piStr(in, "GroupIdentifier"))
}

func (s *Server) piGetResourceMetadata(in map[string]any) (map[string]any, *piErr) {
	serviceType, id, err := s.piResolve(in)
	if err != nil {
		return nil, err
	}
	status := "ENABLED"
	if serviceType == "DOCDB" {
		status = "UNSUPPORTED"
	}
	return map[string]any{"Identifier": id, "Features": map[string]any{"PER_DB_COUNTERS": map[string]any{"Status": status}}}, nil
}

func (s *Server) piListAvailableResourceDimensions(in map[string]any) (map[string]any, *piErr) {
	serviceType, _, err := s.piResolve(in)
	if err != nil {
		return nil, err
	}
	groups := piGroups(serviceType)
	names := make([]string, 0, len(groups))
	for n := range groups {
		names = append(names, n)
	}
	sort.Strings(names)
	var all []any
	for _, metric := range piList(in, "Metrics") {
		root := strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(metric, ".avg"), ".min"), ".max"), ".sum")
		if _, ok := piLoadMetrics[root]; !ok {
			return nil, piInvalid("%s has no dimensions; use db.load or db.sampledload.", metric)
		}
		detail := []any{}
		for _, n := range names {
			dims := []any{}
			for _, d := range groups[n].dims {
				dims = append(dims, map[string]any{"Identifier": d})
			}
			detail = append(detail, map[string]any{"Group": n, "Dimensions": dims})
		}
		all = append(all, map[string]any{"Metric": metric, "Groups": detail})
	}
	start, end, next, err := piPage(in, len(all), 25)
	if err != nil {
		return nil, err
	}
	resp := map[string]any{"MetricDimensions": append([]any{}, all[start:end]...)}
	if next != "" {
		resp["NextToken"] = next
	}
	return resp, nil
}

func (s *Server) piListAvailableResourceMetrics(in map[string]any) (map[string]any, *piErr) {
	if _, _, err := s.piResolve(in); err != nil {
		return nil, err
	}
	var all []any
	for _, t := range piList(in, "MetricTypes") {
		defs, ok := piCounterMetrics[t]
		if !ok {
			return nil, piInvalid("%s is not a valid metric type; use os or db.", t)
		}
		for _, d := range defs {
			all = append(all, map[string]any{"Metric": d.name, "Description": d.desc, "Unit": d.unit})
		}
	}
	start, end, next, err := piPage(in, len(all), 25)
	if err != nil {
		return nil, err
	}
	resp := map[string]any{"Metrics": append([]any{}, all[start:end]...)}
	if next != "" {
		resp["NextToken"] = next
	}
	return resp, nil
}
