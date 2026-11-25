package server

import (
	"bytes"
	"crypto/rand"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
)

// --- WriteRecords ---

func tsmTimeNanos(raw, unit string) (int64, bool) {
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, false
	}
	scale := int64(1_000_000) // MILLISECONDS is the default unit
	switch unit {
	case "SECONDS":
		scale = 1_000_000_000
	case "MICROSECONDS":
		scale = 1_000
	case "NANOSECONDS":
		scale = 1
	}
	if n > math.MaxInt64/scale || n < math.MinInt64/scale {
		return 0, false
	}
	return n * scale, true
}

// tsmCheckValue validates a scalar measure value for its type.
func tsmCheckValue(typ, v string) string {
	switch typ {
	case "DOUBLE":
		if f, err := strconv.ParseFloat(v, 64); err != nil || math.IsNaN(f) || math.IsInf(f, 0) {
			return fmt.Sprintf("Measure value %q is not a valid DOUBLE.", v)
		}
	case "BIGINT", "TIMESTAMP":
		if _, err := strconv.ParseInt(v, 10, 64); err != nil {
			return fmt.Sprintf("Measure value %q is not a valid %s.", v, typ)
		}
	case "BOOLEAN":
		if v != "true" && v != "false" {
			return fmt.Sprintf("Measure value %q is not a valid BOOLEAN.", v)
		}
	case "VARCHAR":
	default:
		return fmt.Sprintf("Unsupported measure value type %s.", typ)
	}
	return ""
}

// tsmBuildRecord merges the common attributes into one record and validates
// it. It returns the record or the reason it is rejected.
func tsmBuildRecord(common, r map[string]any) (*tsmRecord, string) {
	pick := func(key string) string {
		if v := tsmStr(r, key); v != "" {
			return v
		}
		return tsmStr(common, key)
	}
	rec := &tsmRecord{dims: map[string]string{}, measure: pick("MeasureName"), typ: pick("MeasureValueType"), value: pick("MeasureValue")}
	for _, src := range [][]any{tsmList(common, "Dimensions"), tsmList(r, "Dimensions")} {
		for _, d := range src {
			dim, _ := d.(map[string]any)
			name := tsmStr(dim, "Name")
			if _, dup := rec.dims[name]; dup {
				return nil, fmt.Sprintf("Duplicate dimension %s.", name)
			}
			rec.dims[name] = tsmStr(dim, "Value")
		}
	}
	if rec.measure == "" {
		return nil, "MeasureName is required."
	}
	if _, clash := rec.dims[rec.measure]; clash {
		return nil, fmt.Sprintf("Measure name %s is also used as a dimension name.", rec.measure)
	}
	if rec.typ == "" {
		rec.typ = "DOUBLE"
	}
	raw := pick("Time")
	if raw == "" {
		return nil, "Time is required."
	}
	t, ok := tsmTimeNanos(raw, pick("TimeUnit"))
	if !ok {
		return nil, fmt.Sprintf("Time %q is not a valid timestamp.", raw)
	}
	rec.t = t
	rec.version = tsmInt(r, "Version", tsmInt(common, "Version", 1))
	values := tsmList(r, "MeasureValues")
	if len(values) == 0 {
		values = tsmList(common, "MeasureValues")
	}
	if rec.typ == "MULTI" {
		if rec.value != "" || len(values) == 0 {
			return nil, "A MULTI record needs MeasureValues and no MeasureValue."
		}
		seen := map[string]bool{}
		for _, item := range values {
			mv, _ := item.(map[string]any)
			m := tsmMeasure{name: tsmStr(mv, "Name"), typ: tsmStr(mv, "Type"), value: tsmStr(mv, "Value")}
			if seen[m.name] {
				return nil, fmt.Sprintf("Duplicate measure value name %s.", m.name)
			}
			if _, clash := rec.dims[m.name]; clash {
				return nil, fmt.Sprintf("Measure value name %s is also used as a dimension name.", m.name)
			}
			seen[m.name] = true
			if m.typ == "MULTI" {
				return nil, "MeasureValues can't have type MULTI."
			}
			if reason := tsmCheckValue(m.typ, m.value); reason != "" {
				return nil, reason
			}
			rec.multi = append(rec.multi, m)
		}
		sort.Slice(rec.multi, func(i, j int) bool { return rec.multi[i].name < rec.multi[j].name })
		return rec, ""
	}
	if len(values) > 0 {
		return nil, "MeasureValues can only be used with MeasureValueType MULTI."
	}
	if rec.value == "" {
		return nil, "MeasureValue is required."
	}
	if reason := tsmCheckValue(rec.typ, rec.value); reason != "" {
		return nil, reason
	}
	return rec, ""
}

func (r *tsmRecord) key() string {
	names := make([]string, 0, len(r.dims))
	for n := range r.dims {
		names = append(names, n)
	}
	sort.Strings(names)
	var b strings.Builder
	b.WriteString(r.measure)
	b.WriteByte(0)
	b.WriteString(strconv.FormatInt(r.t, 10))
	for _, n := range names {
		b.WriteByte(0)
		b.WriteString(n)
		b.WriteByte('=')
		b.WriteString(r.dims[n])
	}
	return b.String()
}

func (r *tsmRecord) sameValue(o *tsmRecord) bool {
	if r.typ != o.typ || r.value != o.value || len(r.multi) != len(o.multi) {
		return false
	}
	for i := range r.multi {
		if r.multi[i] != o.multi[i] {
			return false
		}
	}
	return true
}

// ingest stores one record, returning where it went ("memory"/"magnetic") or
// the rejection reason and any existing version.
func (st *tsmStore) ingest(t *tsmTable, rec *tsmRecord, now time.Time) (string, string, int64) {
	if pk := tsmList(t.schema, "CompositePartitionKey"); len(pk) == 1 {
		k, _ := pk[0].(map[string]any)
		if tsmStr(k, "Type") == "DIMENSION" && tsmStr(k, "EnforcementInRecord") == "REQUIRED" {
			if _, ok := rec.dims[tsmStr(k, "Name")]; !ok {
				return "", fmt.Sprintf("The record is missing the required partition key dimension %s.", tsmStr(k, "Name")), 0
			}
		}
	}
	memStart := now.Add(-time.Duration(t.memHours) * time.Hour).UnixNano()
	end := now.Add(tsmFutureSkew).UnixNano()
	store := "memory"
	if rec.t >= end || rec.t < memStart {
		magStart := now.AddDate(0, 0, -int(t.magDays)).UnixNano()
		enabled, _ := tsmBool(t.magWrites, "EnableMagneticStoreWrites")
		if rec.t >= end || !enabled || rec.t < magStart {
			return "", fmt.Sprintf("The record timestamp is outside the time range [%s, %s) of the data ingestion window.",
				time.Unix(0, memStart).UTC().Format(time.RFC3339), time.Unix(0, end).UTC().Format(time.RFC3339)), 0
		}
		store = "magnetic"
	}
	key := rec.key()
	if i, ok := t.index[key]; ok {
		old := t.recs[i]
		if old.sameValue(rec) && rec.version <= old.version {
			return store, "", 0 // an identical write is idempotent
		}
		if rec.version <= old.version {
			return "", "A record with the same time, dimensions and measure name already exists with a different value or a higher version.", old.version
		}
		t.recs[i] = rec
		return store, "", 0
	}
	if len(t.recs) >= tsmMaxRecordsPerTable || st.records >= tsmMaxRecords {
		return "", "The table has reached the emulator's record limit.", 0
	}
	t.index[key] = len(t.recs)
	t.recs = append(t.recs, rec)
	st.records++
	return store, "", 0
}

func (st *tsmStore) writeRecords(call *tsmCall) (map[string]any, *tsmError) {
	t, err := st.table(tsmStr(call.in, "DatabaseName"), tsmStr(call.in, "TableName"))
	if err != nil {
		return nil, err
	}
	common := tsmMap(call.in, "CommonAttributes")
	var rejected []any
	mem, mag := 0, 0
	for i, item := range tsmList(call.in, "Records") {
		r, _ := item.(map[string]any)
		rec, reason := tsmBuildRecord(common, r)
		var existing int64
		where := ""
		if reason == "" {
			where, reason, existing = st.ingest(t, rec, call.now)
		}
		if reason != "" {
			rr := map[string]any{"RecordIndex": float64(i), "Reason": reason}
			if existing != 0 {
				rr["ExistingVersion"] = float64(existing)
			}
			rejected = append(rejected, rr)
			continue
		}
		if where == "magnetic" {
			mag++
		} else {
			mem++
		}
	}
	if len(rejected) > 0 {
		// Valid records in the same request are still written, as in AWS.
		return nil, &tsmError{code: "RejectedRecordsException", msg: "One or more records have been rejected. See RejectedRecords for details.",
			extra: map[string]any{"RejectedRecords": rejected}}
	}
	return map[string]any{"RecordsIngested": map[string]any{"Total": float64(mem + mag), "MemoryStore": float64(mem), "MagneticStore": float64(mag)}}, nil
}

// --- batch load ---

type tsmTask struct {
	id, db, table  string
	status, errMsg string
	in             map[string]any // the create request, as conformed
	progress       map[string]any
	created        time.Time
	updated        time.Time
	reads          int
}

func tsmNewTaskID() string {
	b := make([]byte, 10)
	_, _ = rand.Read(b)
	return strings.ToUpper(hex.EncodeToString(b))
}

func (t *tsmTask) out() map[string]any {
	m := map[string]any{
		"TaskId": t.id, "TaskStatus": t.status, "TargetDatabaseName": t.db, "TargetTableName": t.table,
		"DataSourceConfiguration": t.in["DataSourceConfiguration"], "ReportConfiguration": t.in["ReportConfiguration"],
		"DataModelConfiguration": t.in["DataModelConfiguration"], "ProgressReport": t.progress,
		"CreationTime": tsmEpoch(t.created), "LastUpdatedTime": tsmEpoch(t.updated),
		"ResumableUntil": tsmEpoch(t.created.Add(15 * 24 * time.Hour)),
	}
	if v, ok := t.in["RecordVersion"]; ok {
		m["RecordVersion"] = v
	}
	if t.errMsg != "" {
		m["ErrorMessage"] = t.errMsg
	}
	return m
}

// tsmDataModel returns the data model from the request or from S3.
func tsmDataModel(call *tsmCall) (map[string]any, *tsmError) {
	cfg := tsmMap(call.in, "DataModelConfiguration")
	if dm := tsmMap(cfg, "DataModel"); dm != nil {
		return dm, nil
	}
	if s3 := tsmMap(cfg, "DataModelS3Configuration"); s3 != nil {
		var dm map[string]any
		if call.model == nil || json.Unmarshal(call.model, &dm) != nil {
			return nil, tsmValidation("The data model object s3://%s/%s could not be read as JSON.", tsmStr(s3, "BucketName"), tsmStr(s3, "ObjectKey"))
		}
		if inner := tsmMap(dm, "DataModel"); inner != nil {
			dm = inner
		}
		return dm, nil
	}
	return nil, tsmValidation("DataModelConfiguration needs a DataModel or a DataModelS3Configuration")
}

func (st *tsmStore) createBatchLoadTask(call *tsmCall) (map[string]any, *tsmError) {
	if tok := tsmStr(call.in, "ClientToken"); tok != "" {
		if id, ok := st.tokens[tok]; ok {
			return map[string]any{"TaskId": id}, nil
		}
	}
	t, err := st.table(tsmStr(call.in, "TargetDatabaseName"), tsmStr(call.in, "TargetTableName"))
	if err != nil {
		return nil, err
	}
	if len(st.tasks) >= tsmMaxTasks {
		return nil, tsmQuota("You have reached the maximum number of batch load tasks (%d).", tsmMaxTasks)
	}
	src := tsmPath(call.in, "DataSourceConfiguration", "DataSourceS3Configuration")
	if !call.buckets[tsmStr(src, "BucketName")] {
		return nil, tsmValidation("The source S3 bucket %s does not exist.", tsmStr(src, "BucketName"))
	}
	report := tsmPath(call.in, "ReportConfiguration", "ReportS3Configuration")
	if !call.buckets[tsmStr(report, "BucketName")] {
		return nil, tsmValidation("The report S3 bucket %s does not exist.", tsmStr(report, "BucketName"))
	}
	if call.denied != "" {
		return nil, &tsmError{code: "AccessDeniedException", msg: call.denied}
	}
	dm, derr := tsmDataModel(call)
	if derr != nil {
		return nil, derr
	}
	task := &tsmTask{id: tsmNewTaskID(), db: t.db, table: t.name, status: "CREATED", in: call.in, created: call.now, updated: call.now}
	// ponytail: the load runs at creation; the status only settles on later reads.
	st.runBatchLoad(task, t, dm, call)
	st.tasks[task.id] = task
	st.taskIDs = append(st.taskIDs, task.id)
	if tok := tsmStr(call.in, "ClientToken"); tok != "" {
		st.tokens[tok] = task.id
	}
	return map[string]any{"TaskId": task.id}, nil
}

// runBatchLoad parses the CSV objects with the data model and ingests the rows.
func (st *tsmStore) runBatchLoad(task *tsmTask, t *tsmTable, dm map[string]any, call *tsmCall) {
	var processed, ingested, parseFail, ingestFail, fileFail, bytesMetered int64
	var report []string
	if len(call.objects) == 0 {
		task.errMsg = "No objects were found under the data source S3 location."
	}
	keys := make([]string, 0, len(call.objects))
	for k := range call.objects {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	csvCfg := tsmPath(call.in, "DataSourceConfiguration", "CsvConfiguration")
	version := tsmInt(call.in, "RecordVersion", 1)
	for _, key := range keys {
		body := call.objects[key]
		bytesMetered += int64(len(body))
		rows, header, perr := tsmReadCSV(body, csvCfg)
		if perr != nil {
			fileFail++
			report = append(report, fmt.Sprintf("%s,0,%q", key, perr.Error()))
			continue
		}
		for i, row := range rows {
			if processed >= tsmMaxBatchRows {
				break
			}
			processed++
			rec, reason := tsmRowRecord(header, row, dm, csvCfg, version)
			if reason != "" {
				parseFail++
				report = append(report, fmt.Sprintf("%s,%d,%q", key, i+2, reason))
				continue
			}
			if _, reason, _ = st.ingest(t, rec, call.now); reason != "" {
				ingestFail++
				report = append(report, fmt.Sprintf("%s,%d,%q", key, i+2, reason))
				continue
			}
			ingested++
		}
	}
	task.progress = map[string]any{
		"RecordsProcessed": float64(processed), "RecordsIngested": float64(ingested), "ParseFailures": float64(parseFail),
		"RecordIngestionFailures": float64(ingestFail), "FileFailures": float64(fileFail), "BytesMetered": float64(bytesMetered),
	}
	if len(report) > 0 {
		rc := tsmPath(call.in, "ReportConfiguration", "ReportS3Configuration")
		prefix := strings.TrimSuffix(tsmStr(rc, "ObjectKeyPrefix"), "/")
		if prefix != "" {
			prefix += "/"
		}
		body := "object_key,line,error\n" + strings.Join(report, "\n") + "\n"
		call.putS3(tsmStr(rc, "BucketName"), prefix+task.id+"/error-report.csv", []byte(body))
	}
}

func tsmReadCSV(body []byte, cfg map[string]any) ([][]string, []string, error) {
	r := csv.NewReader(bytes.NewReader(body))
	if sep := tsmStr(cfg, "ColumnSeparator"); sep != "" {
		r.Comma = []rune(sep)[0]
	}
	r.FieldsPerRecord = -1
	if trim, _ := tsmBool(cfg, "TrimWhiteSpace"); trim {
		r.TrimLeadingSpace = true
	}
	header, err := r.Read()
	if err != nil {
		return nil, nil, fmt.Errorf("the file has no header row")
	}
	var rows [][]string
	for len(rows) < tsmMaxBatchRows {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, nil, err
		}
		rows = append(rows, row)
	}
	return rows, header, nil
}

// tsmRowRecord maps one CSV row to a record using the data model.
func tsmRowRecord(header, row []string, dm, cfg map[string]any, version int64) (*tsmRecord, string) {
	null := tsmStr(cfg, "NullValue")
	col := map[string]string{}
	for i, h := range header {
		if i < len(row) && row[i] != null {
			col[strings.TrimSpace(h)] = strings.TrimSpace(row[i])
		}
	}
	timeCol := tsmStr(dm, "TimeColumn")
	if timeCol == "" {
		timeCol = "time"
	}
	r := map[string]any{"Time": col[timeCol], "TimeUnit": tsmStr(dm, "TimeUnit"), "Version": float64(version)}
	var dims []any
	for _, item := range tsmList(dm, "DimensionMappings") {
		m, _ := item.(map[string]any)
		src := tsmStr(m, "SourceColumn")
		dst := tsmStr(m, "DestinationColumn")
		if dst == "" {
			dst = src
		}
		if v, ok := col[src]; ok && v != "" {
			dims = append(dims, map[string]any{"Name": dst, "Value": v})
		}
	}
	r["Dimensions"] = dims
	attrs := func(list []any) []any {
		var out []any
		for _, item := range list {
			m, _ := item.(map[string]any)
			src := tsmStr(m, "SourceColumn")
			if v, ok := col[src]; ok && v != "" {
				name := tsmStr(m, "TargetMultiMeasureAttributeName")
				if name == "" {
					name = src
				}
				typ := tsmStr(m, "MeasureValueType")
				if typ == "" {
					typ = "DOUBLE"
				}
				out = append(out, map[string]any{"Name": name, "Value": v, "Type": typ})
			}
		}
		return out
	}
	switch {
	case tsmMap(dm, "MultiMeasureMappings") != nil:
		mm := tsmMap(dm, "MultiMeasureMappings")
		name := tsmStr(mm, "TargetMultiMeasureName")
		if name == "" {
			name = col[tsmStr(dm, "MeasureNameColumn")]
		}
		r["MeasureName"], r["MeasureValueType"], r["MeasureValues"] = name, "MULTI", attrs(tsmList(mm, "MultiMeasureAttributeMappings"))
	case len(tsmList(dm, "MixedMeasureMappings")) > 0:
		// ponytail: the first mixed mapping whose measure applies to this row wins.
		measureName := col[tsmStr(dm, "MeasureNameColumn")]
		for _, item := range tsmList(dm, "MixedMeasureMappings") {
			m, _ := item.(map[string]any)
			if want := tsmStr(m, "MeasureName"); want != "" && want != measureName {
				continue
			}
			name := tsmStr(m, "TargetMeasureName")
			if name == "" {
				name = measureName
			}
			if tsmStr(m, "MeasureValueType") == "MULTI" {
				r["MeasureName"], r["MeasureValueType"], r["MeasureValues"] = name, "MULTI", attrs(tsmList(m, "MultiMeasureAttributeMappings"))
			} else {
				r["MeasureName"], r["MeasureValueType"], r["MeasureValue"] = name, tsmStr(m, "MeasureValueType"), col[tsmStr(m, "SourceColumn")]
			}
			break
		}
	default:
		return nil, "The data model needs MultiMeasureMappings or MixedMeasureMappings."
	}
	return tsmBuildRecord(nil, r)
}

func (st *tsmStore) task(id string) (*tsmTask, *tsmError) {
	if t := st.tasks[id]; t != nil {
		return t, nil
	}
	return nil, tsmNotFound("The batch load task %s does not exist.", id)
}

// settle moves a task one step on per read: CREATED -> IN_PROGRESS -> final.
func (t *tsmTask) settle(now time.Time) {
	switch t.status {
	case "CREATED":
		t.status = "IN_PROGRESS"
	case "IN_PROGRESS":
		switch {
		case t.errMsg != "":
			t.status = "FAILED"
		case tsmInt(t.progress, "RecordIngestionFailures", 0) > 0:
			t.status = "PROGRESS_STOPPED"
		default:
			t.status = "SUCCEEDED"
		}
	default:
		return
	}
	t.updated = now
}

func (st *tsmStore) describeBatchLoadTask(call *tsmCall) (map[string]any, *tsmError) {
	t, err := st.task(tsmStr(call.in, "TaskId"))
	if err != nil {
		return nil, err
	}
	t.settle(call.now)
	return map[string]any{"BatchLoadTaskDescription": t.out()}, nil
}

func (st *tsmStore) listBatchLoadTasks(call *tsmCall) (map[string]any, *tsmError) {
	want := tsmStr(call.in, "TaskStatus")
	var items []*tsmTask
	for _, id := range st.taskIDs {
		if t := st.tasks[id]; t != nil && (want == "" || t.status == want) {
			items = append(items, t)
		}
	}
	page, next, err := tsmPage(items, tsmStr(call.in, "NextToken"), tsmInt(call.in, "MaxResults", 0), 100)
	if err != nil {
		return nil, err
	}
	out := make([]any, 0, len(page))
	for _, t := range page {
		out = append(out, map[string]any{
			"TaskId": t.id, "TaskStatus": t.status, "DatabaseName": t.db, "TableName": t.table,
			"CreationTime": tsmEpoch(t.created), "LastUpdatedTime": tsmEpoch(t.updated), "ResumableUntil": tsmEpoch(t.created.Add(15 * 24 * time.Hour)),
		})
	}
	res := map[string]any{"BatchLoadTasks": out}
	if next != "" {
		res["NextToken"] = next
	}
	return res, nil
}

func (st *tsmStore) resumeBatchLoadTask(call *tsmCall) (map[string]any, *tsmError) {
	t, err := st.task(tsmStr(call.in, "TaskId"))
	if err != nil {
		return nil, err
	}
	if t.status != "PROGRESS_STOPPED" {
		return nil, tsmValidation("Batch load task %s is %s; only a PROGRESS_STOPPED task can be resumed.", t.id, t.status)
	}
	if call.now.After(t.created.Add(15 * 24 * time.Hour)) {
		return nil, tsmValidation("Batch load task %s can no longer be resumed.", t.id)
	}
	// ponytail: a resume retries nothing new here; it completes the task.
	t.status, t.updated = "SUCCEEDED", call.now
	return map[string]any{}, nil
}
