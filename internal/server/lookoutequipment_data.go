package server

import (
	"encoding/csv"
	"fmt"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Ingestion reads sensor CSVs from Stackyard's S3 before the store lock is
// taken (the two stores' locks never nest) and summarises them: one sensor
// per non-timestamp column, with real missing, invalid and duplicate counts.

const lkeMaxSensors = 3000

type lkeSensor struct {
	component, name        string
	rows, missing, invalid int
	dups                   int
	start, end             float64
}

type lkeIngest struct {
	failure      string
	files        int
	size         int64
	sensors      []lkeSensor
	start, end   float64
	missing, bad int
	dupTimes     int
	badTimes     int
}

var lkeTimeLayouts = []string{time.RFC3339, "2006-01-02 15:04:05", "2006-01-02T15:04:05", "2006-01-02 15:04:05.000", "2006-01-02"}

func lkeParseTime(v string) (float64, bool) {
	v = strings.TrimSpace(v)
	for _, layout := range lkeTimeLayouts {
		if t, err := time.Parse(layout, v); err == nil {
			return fcEpoch(t), true
		}
	}
	if n, err := strconv.ParseFloat(v, 64); err == nil && n > 0 {
		return n, true
	}
	return 0, false
}

// lkeComponent names a file's component: its first folder under prefix, or
// its file name without extension.
func lkeComponent(key, prefix string) string {
	rel := strings.TrimPrefix(key, prefix)
	if dir, _, ok := strings.Cut(rel, "/"); ok && dir != "" {
		return dir
	}
	return strings.TrimSuffix(path.Base(rel), path.Ext(rel))
}

// lkeReadIngestion reads every CSV under s3://bucket/prefix.
func (s *Server) lkeReadIngestion(cfg map[string]any) lkeIngest {
	var res lkeIngest
	bucket, prefix := fcStr(cfg, "Bucket"), fcStr(cfg, "Prefix")
	files, err := s.pszReadS3("s3://" + bucket + "/" + prefix)
	if err != nil {
		res.failure = err.Error()
		return res
	}
	bySensor := map[string]*lkeSensor{}
	for _, f := range files {
		if !strings.HasSuffix(strings.ToLower(f.key), ".csv") {
			continue
		}
		res.files++
		res.size += int64(len(f.body))
		if msg := res.addCSV(bySensor, lkeComponent(f.key, prefix), string(f.body)); msg != "" {
			res.failure = fmt.Sprintf("s3://%s/%s: %s", bucket, f.key, msg)
			return res
		}
	}
	if res.files == 0 {
		res.failure = fmt.Sprintf("No CSV files found under s3://%s/%s.", bucket, prefix)
		return res
	}
	for _, sn := range bySensor {
		res.sensors = append(res.sensors, *sn)
	}
	sort.Slice(res.sensors, func(i, j int) bool {
		a, b := res.sensors[i], res.sensors[j]
		return a.component+"/"+a.name < b.component+"/"+b.name
	})
	return res
}

func (res *lkeIngest) addCSV(bySensor map[string]*lkeSensor, component, body string) string {
	rows, err := csv.NewReader(strings.NewReader(body)).ReadAll()
	if err != nil || len(rows) == 0 || len(rows[0]) < 2 {
		return "a CSV needs a header with a timestamp column and at least one sensor column"
	}
	header := rows[0]
	cols := make([]*lkeSensor, len(header))
	for i, name := range header[1:] {
		key := component + "/" + name
		if bySensor[key] == nil {
			if len(bySensor) >= lkeMaxSensors {
				return fmt.Sprintf("more than %d sensors", lkeMaxSensors)
			}
			bySensor[key] = &lkeSensor{component: component, name: name}
		}
		cols[i+1] = bySensor[key]
	}
	seen := make(map[float64]bool, len(rows))
	for _, row := range rows[1:] {
		ts, ok := lkeParseTime(row[0])
		if !ok {
			res.badTimes++
			continue
		}
		dup := seen[ts]
		if dup {
			res.dupTimes++
		}
		seen[ts] = true
		if res.start == 0 || ts < res.start {
			res.start = ts
		}
		res.end = max(res.end, ts)
		for i := 1; i < len(cols) && i < len(row); i++ {
			sn, v := cols[i], strings.TrimSpace(row[i])
			sn.rows++
			if sn.start == 0 || ts < sn.start {
				sn.start = ts
			}
			sn.end = max(sn.end, ts)
			if dup {
				sn.dups++
			}
			if v == "" {
				sn.missing++
				res.missing++
			} else if _, err := strconv.ParseFloat(v, 64); err != nil {
				sn.invalid++
				res.bad++
			}
		}
	}
	return ""
}

// lkeQualitySummary is the DataQualitySummary of an ingestion.
func lkeQualitySummary(in *lkeIngest) map[string]any {
	affected := func(f func(lkeSensor) int) int {
		n := 0
		for _, sn := range in.sensors {
			if f(sn) > 0 {
				n++
			}
		}
		return n
	}
	empty := 0
	for _, sn := range in.sensors {
		if sn.rows == 0 || sn.missing == sn.rows {
			empty++
		}
	}
	return map[string]any{
		"InsufficientSensorData": map[string]any{
			"MissingCompleteSensorData": map[string]any{"AffectedSensorCount": empty},
			"SensorsWithShortDateRange": map[string]any{"AffectedSensorCount": 0},
		},
		"MissingSensorData":     map[string]any{"AffectedSensorCount": affected(func(s lkeSensor) int { return s.missing }), "TotalNumberOfMissingValues": in.missing},
		"InvalidSensorData":     map[string]any{"AffectedSensorCount": affected(func(s lkeSensor) int { return s.invalid }), "TotalNumberOfInvalidValues": in.bad},
		"UnsupportedTimestamps": map[string]any{"TotalNumberOfUnsupportedTimestamps": in.badTimes},
		"DuplicateTimestamps":   map[string]any{"TotalNumberOfDuplicateTimestamps": in.dupTimes},
	}
}

func lkePct(n, total int) float64 {
	if total == 0 {
		return 0
	}
	return float64(n) * 100 / float64(total)
}

// lkeSensorSummary is one ListSensorStatistics entry.
func lkeSensorSummary(sn lkeSensor) map[string]any {
	none := map[string]any{"Status": "NO_ISSUE_DETECTED"}
	count := func(n int) map[string]any { return map[string]any{"Count": n, "Percentage": lkePct(n, sn.rows)} }
	return map[string]any{
		"ComponentName": sn.component, "SensorName": sn.name, "DataExists": sn.rows > sn.missing,
		"MissingValues": count(sn.missing), "InvalidValues": count(sn.invalid), "InvalidDateEntries": count(0),
		"DuplicateTimestamps": count(sn.dups), "CategoricalValues": none, "MultipleOperatingModes": none,
		"LargeTimestampGaps": none, "MonotonicValues": none,
		"DataStartTime": sn.start, "DataEndTime": sn.end,
	}
}
