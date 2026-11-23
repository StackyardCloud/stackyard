package server

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
)

const (
	dbwMaxInputBytes = 32 << 20 // dataset bytes read by one run or preview
	dbwMaxRows       = 100_000  // rows read by one run
	dbwMaxColumns    = 1_000
	// Rows are padded to the column count and steps can grow values, so the
	// whole table is bounded too, not just what was read.
	dbwMaxCells      = 5_000_000
	dbwMaxTableBytes = 2 * dbwMaxInputBytes
	dbwMaxSteps      = 100
	dbwProfileRows   = 20_000 // default profile sample (AWS's default is 20,000 rows)
)

// dbwTable is a dataset loaded as strings; "" is a missing value.
type dbwTable struct {
	cols []string
	rows [][]string
	size int // bytes held in cells
}

// grow accounts for n more cell bytes and enforces the table budgets.
func (t *dbwTable) grow(n int) error {
	t.size += n
	if t.size > dbwMaxTableBytes {
		return fmt.Errorf("the table would exceed %d MiB", dbwMaxTableBytes>>20)
	}
	if len(t.rows)*len(t.cols) > dbwMaxCells {
		return fmt.Errorf("the table would exceed %d cells", dbwMaxCells)
	}
	return nil
}

// dbwLoadTable reads up to limit rows (the last ones with fromEnd) of an S3
// dataset in CSV or JSON. Other sources and formats aren't executed.
func (s *Server) dbwLoadTable(dataset map[string]any, limit int, fromEnd bool) (*dbwTable, error) {
	if dbwStr(dataset, "Source") != "S3" {
		return nil, fmt.Errorf("only S3 datasets are read by Stackyard (dataset %s is %s)", dbwStr(dataset, "Name"), dbwStr(dataset, "Source"))
	}
	loc := dbwMap(dbwMap(dataset, "Input"), "S3InputDefinition")
	bucket, key := dbwStr(loc, "Bucket"), dbwStr(loc, "Key")
	var keys []string
	if obj, err := s.s3.GetObject(bucket, key); err == nil && key != "" && !strings.HasSuffix(key, "/") {
		keys = []string{obj.Key}
	} else {
		objs, err := s.s3.ListObjects(bucket, key)
		if err != nil {
			return nil, fmt.Errorf("s3://%s/%s can't be listed", bucket, key)
		}
		for _, o := range objs {
			if !strings.HasSuffix(o.Key, "/") {
				keys = append(keys, o.Key)
			}
		}
		sort.Strings(keys)
	}
	if len(keys) == 0 {
		return nil, fmt.Errorf("s3://%s/%s has no objects", bucket, key)
	}
	format := strings.ToUpper(dbwStr(dataset, "Format"))
	if format == "" {
		format = "CSV"
		if strings.HasSuffix(strings.ToLower(keys[0]), ".json") {
			format = "JSON"
		}
	}
	if format != "CSV" && format != "JSON" {
		return nil, fmt.Errorf("Stackyard reads CSV and JSON datasets, not %s", format)
	}
	t := &dbwTable{}
	total := 0
	for _, k := range keys {
		obj, err := s.s3.GetObject(bucket, k)
		if err != nil {
			return nil, fmt.Errorf("s3://%s/%s can't be read", bucket, k)
		}
		if total += len(obj.Body); total > dbwMaxInputBytes {
			return nil, fmt.Errorf("the dataset is larger than the %d MiB Stackyard reads", dbwMaxInputBytes>>20)
		}
		if format == "CSV" {
			err = t.readCSV(obj.Body, dbwMap(dbwMap(dataset, "FormatOptions"), "Csv"))
		} else {
			err = t.readJSON(obj.Body)
		}
		if err != nil {
			return nil, fmt.Errorf("s3://%s/%s: %v", bucket, k, err)
		}
	}
	if fromEnd && len(t.rows) > limit {
		t.rows = t.rows[len(t.rows)-limit:]
	} else if len(t.rows) > limit {
		t.rows = t.rows[:limit]
	}
	return t, nil
}

func (t *dbwTable) readCSV(body []byte, opts map[string]any) error {
	r := csv.NewReader(bytes.NewReader(body))
	r.FieldsPerRecord = -1
	r.LazyQuotes = true
	if d := []rune(dbwStr(opts, "Delimiter")); len(d) == 1 {
		r.Comma = d[0]
	}
	header := true
	if v, ok := opts["HeaderRow"].(bool); ok {
		header = v
	}
	first := true
	for len(t.rows) < dbwMaxRows {
		rec, err := r.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("CSV can't be parsed: %v", err)
		}
		if len(rec) > dbwMaxColumns {
			return fmt.Errorf("more than %d columns", dbwMaxColumns)
		}
		if first {
			first = false
			if t.cols == nil {
				if header {
					t.cols = append([]string{}, rec...)
					continue
				}
				for i := range rec {
					t.cols = append(t.cols, "Column_"+strconv.Itoa(i+1))
				}
			} else if header {
				continue // later objects repeat the header
			}
		}
		if err := t.addRow(rec); err != nil {
			return err
		}
	}
	return nil
}

// readJSON reads JSON lines or one JSON array of objects. Columns appear in
// first-seen order; non-string values keep their JSON text.
func (t *dbwTable) readJSON(body []byte) error {
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()
	var records []map[string]any
	for len(t.rows)+len(records) < dbwMaxRows {
		var v any
		if err := dec.Decode(&v); errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return fmt.Errorf("JSON can't be parsed: %v", err)
		}
		switch x := v.(type) {
		case map[string]any:
			records = append(records, x)
		case []any:
			for _, item := range x {
				if len(t.rows)+len(records) >= dbwMaxRows {
					break
				}
				if obj, ok := item.(map[string]any); ok {
					records = append(records, obj)
				}
			}
		}
	}
	index := map[string]int{}
	for i, c := range t.cols {
		index[c] = i
	}
	for _, rec := range records {
		for _, k := range dbwSortedKeys(rec) {
			if _, ok := index[k]; !ok {
				if len(t.cols) >= dbwMaxColumns {
					return fmt.Errorf("more than %d columns", dbwMaxColumns)
				}
				index[k] = len(t.cols)
				t.cols = append(t.cols, k)
			}
		}
	}
	for _, rec := range records {
		row := make([]string, len(t.cols))
		for k, v := range rec {
			switch x := v.(type) {
			case string:
				row[index[k]] = x
			case nil:
			default:
				raw, _ := json.Marshal(x)
				row[index[k]] = string(raw)
			}
		}
		if err := t.addRow(row); err != nil {
			return err
		}
	}
	return nil
}

func (t *dbwTable) addRow(rec []string) error {
	if (len(t.rows)+1)*len(t.cols) > dbwMaxCells {
		return fmt.Errorf("the table would exceed %d cells", dbwMaxCells)
	}
	row := make([]string, len(t.cols))
	copy(row, rec)
	t.rows = append(t.rows, row)
	n := 0
	for _, v := range row {
		n += len(v)
	}
	return t.grow(n)
}

func (t *dbwTable) col(name string) (int, error) {
	for i, c := range t.cols {
		if c == name {
			return i, nil
		}
	}
	return 0, fmt.Errorf("column %q doesn't exist", name)
}

// dbwSupportedOperations lists the recipe actions Stackyard applies to data;
// any other operation fails the preview or job run with a clear message.
var dbwSupportedOperations = []string{
	"ADD_PREFIX", "ADD_SUFFIX", "CAPITAL_CASE", "DELETE", "DUPLICATE", "FILL_WITH_CUSTOM",
	"FILL_WITH_EMPTY", "LOWER_CASE", "REMOVE_MISSING", "RENAME", "SENTENCE_CASE", "UPPER_CASE",
}

// apply runs recipe steps in order.
func (t *dbwTable) apply(steps []any) error {
	if len(steps) > dbwMaxSteps {
		return fmt.Errorf("a recipe can apply at most %d steps", dbwMaxSteps)
	}
	for i, raw := range steps {
		step, _ := raw.(map[string]any)
		if len(dbwList(step, "ConditionExpressions")) > 0 {
			return fmt.Errorf("step %d: condition expressions aren't evaluated by Stackyard", i+1)
		}
		action := dbwMap(step, "Action")
		params := map[string]string{}
		for k, v := range dbwMap(action, "Parameters") {
			if s, ok := v.(string); ok {
				params[k] = s
			}
		}
		if err := t.applyOne(dbwStr(action, "Operation"), params); err != nil {
			return fmt.Errorf("step %d (%s): %v", i+1, dbwStr(action, "Operation"), err)
		}
	}
	return nil
}

func (t *dbwTable) mapColumn(name string, f func(string) string) error {
	c, err := t.col(name)
	if err != nil {
		return err
	}
	for _, row := range t.rows {
		before := len(row[c])
		row[c] = f(row[c])
		if err := t.grow(len(row[c]) - before); err != nil {
			return err
		}
	}
	return nil
}

func (t *dbwTable) applyOne(op string, p map[string]string) error {
	src := p["sourceColumn"]
	switch op {
	case "UPPER_CASE":
		return t.mapColumn(src, strings.ToUpper)
	case "LOWER_CASE":
		return t.mapColumn(src, strings.ToLower)
	case "CAPITAL_CASE":
		return t.mapColumn(src, func(v string) string {
			words := strings.Fields(strings.ToLower(v))
			for i, w := range words {
				r := []rune(w)
				r[0] = unicode.ToUpper(r[0])
				words[i] = string(r)
			}
			return strings.Join(words, " ")
		})
	case "SENTENCE_CASE":
		return t.mapColumn(src, func(v string) string {
			r := []rune(strings.ToLower(v))
			if len(r) > 0 {
				r[0] = unicode.ToUpper(r[0])
			}
			return string(r)
		})
	case "ADD_PREFIX", "ADD_SUFFIX":
		pattern := p["pattern"]
		return t.mapColumn(src, func(v string) string {
			if op == "ADD_PREFIX" {
				return pattern + v
			}
			return v + pattern
		})
	case "FILL_WITH_CUSTOM":
		value := p["value"]
		return t.mapColumn(src, func(v string) string {
			if v == "" {
				return value
			}
			return v
		})
	case "FILL_WITH_EMPTY":
		// Missing values are already "" here; check the column exists.
		_, err := t.col(src)
		return err
	case "REMOVE_MISSING":
		c, err := t.col(src)
		if err != nil {
			return err
		}
		kept := t.rows[:0]
		for _, row := range t.rows {
			if row[c] != "" {
				kept = append(kept, row)
			}
		}
		t.rows = kept
		return nil
	case "RENAME":
		c, err := t.col(src)
		if err != nil {
			return err
		}
		target := p["targetColumn"]
		if target == "" {
			return errors.New("targetColumn is required")
		}
		if _, err := t.col(target); err == nil {
			return fmt.Errorf("column %q already exists", target)
		}
		t.cols[c] = target
		return nil
	case "DUPLICATE":
		c, err := t.col(src)
		if err != nil {
			return err
		}
		target := p["targetColumn"]
		if target == "" {
			return errors.New("targetColumn is required")
		}
		if _, err := t.col(target); err == nil {
			return fmt.Errorf("column %q already exists", target)
		}
		if len(t.cols) >= dbwMaxColumns {
			return fmt.Errorf("more than %d columns", dbwMaxColumns)
		}
		if len(t.rows)*(len(t.cols)+1) > dbwMaxCells {
			return fmt.Errorf("the table would exceed %d cells", dbwMaxCells)
		}
		added := 0
		for _, row := range t.rows {
			added += len(row[c])
		}
		t.cols = append(t.cols, target)
		for i, row := range t.rows {
			t.rows[i] = append(row, row[c])
		}
		return t.grow(added)
	case "DELETE":
		var names []string
		if err := json.Unmarshal([]byte(p["sourceColumns"]), &names); err != nil || len(names) == 0 {
			return errors.New(`sourceColumns must be a JSON array of column names, like ["a","b"]`)
		}
		drop := map[int]bool{}
		for _, n := range names {
			c, err := t.col(n)
			if err != nil {
				return err
			}
			drop[c] = true
		}
		keep := func(vals []string) []string {
			out := make([]string, 0, len(vals)-len(drop))
			for i, v := range vals {
				if !drop[i] {
					out = append(out, v)
				}
			}
			return out
		}
		for i, row := range t.rows {
			t.rows[i] = keep(row)
		}
		t.cols = keep(t.cols)
		return nil
	}
	return fmt.Errorf("operation %s isn't supported by Stackyard (supported: %s)", op, strings.Join(dbwSupportedOperations, ", "))
}

func (t *dbwTable) csvBytes() []byte {
	var b bytes.Buffer
	w := csv.NewWriter(&b)
	_ = w.Write(t.cols)
	_ = w.WriteAll(t.rows)
	return b.Bytes()
}

func (t *dbwTable) jsonLines() []byte {
	var b bytes.Buffer
	for _, row := range t.rows {
		obj := make(map[string]string, len(t.cols))
		for i, c := range t.cols {
			obj[c] = row[i]
		}
		raw, _ := json.Marshal(obj)
		b.Write(raw)
		b.WriteByte('\n')
	}
	return b.Bytes()
}

// profile summarizes each column: counts, distinct values and, for numeric
// columns, min/max/mean.
func (t *dbwTable) profile(dataset string) map[string]any {
	cols := make([]any, 0, len(t.cols))
	for c, name := range t.cols {
		distinct := map[string]bool{}
		missing, numeric := 0, true
		minV, maxV, sum, n := math.Inf(1), math.Inf(-1), 0.0, 0
		for _, row := range t.rows {
			v := row[c]
			if v == "" {
				missing++
				continue
			}
			distinct[v] = true
			f, err := strconv.ParseFloat(v, 64)
			if err != nil || math.IsNaN(f) || math.IsInf(f, 0) {
				numeric = false
				continue
			}
			minV, maxV, sum, n = math.Min(minV, f), math.Max(maxV, f), sum+f, n+1
		}
		col := map[string]any{
			"name": name, "type": "string", "rowsCount": len(t.rows), "missingValuesCount": missing,
			"distinctValuesCount": len(distinct), "uniqueValuesCount": len(distinct),
		}
		if numeric && n > 0 {
			col["type"], col["min"], col["max"], col["mean"] = "double", minV, maxV, sum/float64(n)
		}
		cols = append(cols, col)
	}
	return map[string]any{"datasetName": dataset, "sampleSize": len(t.rows), "columnsCount": len(t.cols), "columns": cols}
}

// dbwRunJob does a run's work: load the dataset, apply the recipe (RECIPE) or
// profile it (PROFILE), and write the result to S3. A returned error fails the
// run with that message.
func (s *Server) dbwRunJob(spec *dbwJobSpec, runID string, started time.Time) error {
	job := spec.job
	limit := dbwMaxRows
	if dbwStr(job, "Type") == "PROFILE" {
		limit = dbwProfileRows
	}
	if js := dbwMap(job, "JobSample"); dbwStr(js, "Mode") == "CUSTOM_ROWS" {
		if n, ok := js["Size"].(float64); ok && n >= 1 && n < float64(limit) {
			limit = int(n)
		}
	} else if dbwStr(js, "Mode") == "FULL_DATASET" {
		limit = dbwMaxRows
	}
	table, err := s.dbwLoadTable(spec.dataset, limit, false)
	if err != nil {
		return err
	}
	name := dbwStr(job, "Name")
	if dbwStr(job, "Type") == "PROFILE" {
		loc := dbwMap(dbwList(job, "Outputs")[0], "Location")
		body, _ := json.MarshalIndent(table.profile(dbwStr(job, "DatasetName")), "", "  ")
		key := dbwPrefix(dbwStr(loc, "Key")) + dbwStr(job, "DatasetName") + "_" + runID + ".json"
		_, perr := s.s3.PutObject(dbwStr(loc, "Bucket"), key, "application/json", body, nil, "", "")
		return perr
	}
	if err := table.apply(spec.steps); err != nil {
		return err
	}
	stamp := started.Format("02Jan2006_150405")
	// The job's maps are shared with the store, so only read them here.
	type target struct {
		loc    map[string]any
		format string
	}
	var targets []target
	for _, o := range dbwList(job, "Outputs") {
		targets = append(targets, target{dbwMap(o, "Location"), dbwFirst(dbwStr(o, "Format"), "CSV")})
	}
	for _, o := range dbwList(job, "DataCatalogOutputs") {
		if loc := dbwMap(dbwMap(o, "S3Options"), "Location"); loc != nil {
			targets = append(targets, target{loc, "CSV"})
		}
	}
	// ponytail: DatabaseOutputs (JDBC) aren't written; compression and
	// partitioning options are ignored and output is written uncompressed.
	for _, o := range targets {
		loc := o.loc
		var body []byte
		var ext, ctype string
		switch o.format {
		case "CSV":
			body, ext, ctype = table.csvBytes(), "csv", "text/csv"
		case "JSON":
			body, ext, ctype = table.jsonLines(), "json", "application/json"
		default:
			return fmt.Errorf("output format %s isn't written by Stackyard (use CSV or JSON)", o.format)
		}
		key := dbwPrefix(dbwStr(loc, "Key")) + name + "_" + stamp + "_part00000." + ext
		if _, perr := s.s3.PutObject(dbwStr(loc, "Bucket"), key, ctype, body, nil, "", ""); perr != nil {
			return perr
		}
	}
	return nil
}

func dbwPrefix(key string) string {
	if key == "" || strings.HasSuffix(key, "/") {
		return key
	}
	return key + "/"
}
