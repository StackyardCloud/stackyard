package server

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	tsqMaxResultRows = 100_000
	tsqDefaultPage   = 1000
	tsqScheduledTime = "@scheduled_runtime"
)

type tsqCol struct{ name, typ string }

type tsqResult struct {
	cols    []tsqCol
	rows    [][]tsqVal
	scanned int
	q       *tsqQuery
	aliased []bool
}

type tsmCursor struct {
	sql  string
	cols []tsqCol
	rows [][]tsqVal
}

type tsmSettings struct {
	pricing string
	maxTCU  int64
	mode    string
	target  int64
	notify  map[string]any
	updated time.Time
}

func tsqQueryError(format string, args ...any) *tsmError {
	return &tsmError{code: "ValidationException", msg: fmt.Sprintf("Query not supported by the Stackyard Timestream engine: "+format, args...)}
}

// tableTypes returns the column types of a table: time, measure_name,
// dimensions and multi-measure attributes seen in its records.
func tsqTableTypes(t *tsmTable) (map[string]string, []string, []string, []string) {
	types := map[string]string{"time": "TIMESTAMP", "measure_name": "VARCHAR"}
	dims, attrs, scalars := map[string]bool{}, map[string]bool{}, map[string]bool{}
	for _, r := range t.recs {
		for d := range r.dims {
			if !dims[d] {
				dims[d] = true
				types[d] = "VARCHAR"
			}
		}
		if r.typ == "MULTI" {
			for _, m := range r.multi {
				if !attrs[m.name] {
					attrs[m.name] = true
					types[m.name] = m.typ
				}
			}
		} else {
			scalars[r.typ] = true
		}
	}
	keys := func(m map[string]bool) []string {
		out := make([]string, 0, len(m))
		for k := range m {
			out = append(out, k)
		}
		sort.Strings(out)
		return out
	}
	var scalarCols []string
	for _, typ := range []string{"DOUBLE", "BIGINT", "VARCHAR", "BOOLEAN", "TIMESTAMP"} {
		if scalars[typ] {
			name := "measure_value::" + strings.ToLower(typ)
			scalarCols = append(scalarCols, name)
			types[name] = typ
		}
	}
	return types, keys(dims), keys(attrs), scalarCols
}

// runQuery parses and evaluates sql against the store.
func (st *tsmStore) runQuery(sql string, params map[string]tsqVal, now time.Time) (*tsqResult, *tsmError) {
	q, perr := tsqParse(sql)
	if perr != nil {
		return nil, tsqQueryError("%v", perr)
	}
	for _, p := range q.params {
		if _, ok := params[p]; !ok {
			return nil, tsmValidation("The query parameter %s has no value; it is only bound in scheduled queries.", p)
		}
	}
	t, terr := st.table(q.db, q.tbl)
	if terr != nil {
		return nil, tsmValidation("%s", terr.msg)
	}
	types, dims, attrs, scalars := tsqTableTypes(t)
	env := &tsqEnv{now: now.UnixNano(), params: params, types: types}
	if q.star {
		if len(q.groupBy) > 0 {
			return nil, tsqQueryError("SELECT * can't be combined with GROUP BY")
		}
		var items []tsqItem
		for _, name := range append(append(append(dims, "measure_name", "time"), scalars...), attrs...) {
			items = append(items, tsqItem{expr: &tsqNode{op: "col", name: name, text: name}})
		}
		q.items = append(items, q.items...)
	}
	for _, it := range q.items {
		if err := env.checkColumns(it.expr); err != nil {
			return nil, tsmValidation("%v", err)
		}
	}
	if err := env.checkColumns(q.where); err != nil {
		return nil, tsmValidation("%v", err)
	}
	if q.where != nil && tsqHasAgg(q.where) {
		return nil, tsqQueryError("aggregate functions can't be used in WHERE")
	}
	// GROUP BY 1 refers to the first select item.
	for i, g := range q.groupBy {
		if g.op == "lit" && g.val.typ == "BIGINT" {
			if g.val.i < 1 || int(g.val.i) > len(q.items) {
				return nil, tsqQueryError("GROUP BY position %d is out of range", g.val.i)
			}
			q.groupBy[i] = q.items[g.val.i-1].expr
		}
		if err := env.checkColumns(q.groupBy[i]); err != nil {
			return nil, tsmValidation("%v", err)
		}
	}
	res := &tsqResult{q: q, scanned: len(t.recs)}
	for i, it := range q.items {
		name := it.alias
		if name == "" {
			if it.expr.op == "col" {
				name = env.resolve(it.expr.name)
			} else {
				name = "_col" + strconv.Itoa(i)
			}
		}
		res.cols = append(res.cols, tsqCol{name: name, typ: env.typeOf(it.expr)})
		res.aliased = append(res.aliased, it.alias != "")
	}
	var matched []*tsmRecord
	for _, r := range t.recs {
		if q.where == nil || tsqTruth(env.eval(q.where, r)) {
			matched = append(matched, r)
		}
	}
	if env.over {
		return nil, &tsmError{code: "QueryExecutionException", msg: "The query exceeded the emulator's compute budget for LIKE comparisons; narrow the WHERE clause."}
	}
	grouped := len(q.groupBy) > 0
	for _, it := range q.items {
		grouped = grouped || tsqHasAgg(it.expr)
	}
	type outRow struct {
		vals  []tsqVal
		src   *tsmRecord
		group []*tsmRecord
	}
	var rows []outRow
	if grouped {
		groupText := map[string]bool{}
		for _, g := range q.groupBy {
			groupText[g.text] = true
		}
		for _, it := range q.items {
			if !tsqHasAgg(it.expr) && !groupText[it.expr.text] && it.expr.op != "lit" {
				return nil, tsqQueryError("'%s' must be an aggregate expression or appear in GROUP BY", it.expr.text)
			}
		}
		order := []string{}
		groups := map[string][]*tsmRecord{}
		for _, r := range matched {
			var key strings.Builder
			for _, g := range q.groupBy {
				v := env.eval(g, r)
				key.WriteString(v.typ + ":" + v.format() + "\x00")
			}
			k := key.String()
			if _, ok := groups[k]; !ok {
				order = append(order, k)
			}
			groups[k] = append(groups[k], r)
		}
		if len(q.groupBy) == 0 && len(order) == 0 {
			order, groups[""] = []string{""}, nil // a global aggregate over no rows still returns one row
		}
		for _, k := range order {
			g := groups[k]
			vals := make([]tsqVal, len(q.items))
			for i, it := range q.items {
				vals[i] = env.aggregate(it.expr, g)
			}
			rows = append(rows, outRow{vals: vals, group: g})
		}
	} else {
		for _, r := range matched {
			vals := make([]tsqVal, len(q.items))
			for i, it := range q.items {
				vals[i] = env.eval(it.expr, r)
			}
			rows = append(rows, outRow{vals: vals, src: r})
		}
	}
	if len(q.orderBy) > 0 {
		keyOf := func(o tsqOrder, row outRow) tsqVal {
			if o.expr.op == "lit" && o.expr.val.typ == "BIGINT" && o.expr.val.i >= 1 && int(o.expr.val.i) <= len(q.items) {
				return row.vals[o.expr.val.i-1]
			}
			for i, it := range q.items {
				if (o.expr.op == "col" && strings.EqualFold(it.alias, o.expr.name)) || it.expr.text == o.expr.text {
					return row.vals[i]
				}
			}
			if row.src != nil {
				return env.eval(o.expr, row.src)
			}
			return env.aggregate(o.expr, row.group)
		}
		for _, o := range q.orderBy {
			if err := env.checkColumns(o.expr); err != nil && !tsqAliasRef(q, o.expr) {
				return nil, tsmValidation("%v", err)
			}
		}
		sort.SliceStable(rows, func(a, b int) bool {
			for _, o := range q.orderBy {
				x, y := keyOf(o, rows[a]), keyOf(o, rows[b])
				var c int
				switch {
				case x.null() && y.null():
					c = 0
				case x.null():
					c = 1 // NULLs sort last
				case y.null():
					c = -1
				default:
					c, _ = tsqCompare(x, y)
				}
				if o.desc && !x.null() && !y.null() {
					c = -c
				}
				if c != 0 {
					return c < 0
				}
			}
			return false
		})
	}
	if q.limit >= 0 && len(rows) > q.limit {
		rows = rows[:q.limit]
	}
	if len(rows) > tsqMaxResultRows {
		return nil, &tsmError{code: "QueryExecutionException", msg: fmt.Sprintf("The query returned more than %d rows; add a LIMIT or a narrower WHERE.", tsqMaxResultRows)}
	}
	for _, r := range rows {
		res.rows = append(res.rows, r.vals)
	}
	return res, nil
}

func tsqAliasRef(q *tsqQuery, n *tsqNode) bool {
	if n.op != "col" {
		return false
	}
	for _, it := range q.items {
		if strings.EqualFold(it.alias, n.name) {
			return true
		}
	}
	return false
}

func tsqColumnInfo(cols []tsqCol) []any {
	out := make([]any, len(cols))
	for i, c := range cols {
		out[i] = map[string]any{"Name": c.name, "Type": map[string]any{"ScalarType": c.typ}}
	}
	return out
}

func tsqRows(rows [][]tsqVal) []any {
	out := make([]any, len(rows))
	for i, r := range rows {
		data := make([]any, len(r))
		for j, v := range r {
			if v.null() {
				data[j] = map[string]any{"NullValue": true}
			} else {
				data[j] = map[string]any{"ScalarValue": v.format()}
			}
		}
		out[i] = map[string]any{"Data": data}
	}
	return out
}

func tsmQueryID() string {
	b := make([]byte, 13)
	_, _ = rand.Read(b)
	return strings.ToUpper(hex.EncodeToString(b))
}

func (st *tsmStore) query(call *tsmCall) (map[string]any, *tsmError) {
	sql := tsmStr(call.in, "QueryString")
	size := int(tsmInt(call.in, "MaxRows", 0))
	if size <= 0 {
		size = tsqDefaultPage
	}
	var cols []tsqCol
	var rows [][]tsqVal
	var qid string
	offset, scanned := 0, 0
	if tok := tsmStr(call.in, "NextToken"); tok != "" {
		id, off, ok := strings.Cut(tok, ":")
		cur := st.cursors[id]
		n, err := strconv.ParseInt(off, 10, 32)
		if !ok || cur == nil || err != nil || n < 0 || int(n) > len(cur.rows) {
			return nil, tsmValidation("The NextToken is invalid or has expired.")
		}
		if cur.sql != sql {
			return nil, tsmValidation("The QueryString must match the query that produced the NextToken.")
		}
		qid, cols, rows, offset = id, cur.cols, cur.rows, int(n)
	} else {
		res, err := st.runQuery(sql, map[string]tsqVal{}, call.now)
		if err != nil {
			return nil, err
		}
		qid, cols, rows, scanned = tsmQueryID(), res.cols, res.rows, res.scanned
	}
	end := min(offset+size, len(rows))
	out := map[string]any{
		"QueryId": qid, "ColumnInfo": tsqColumnInfo(cols), "Rows": tsqRows(rows[offset:end]),
		"QueryStatus": map[string]any{"ProgressPercentage": float64(100), "CumulativeBytesScanned": float64(scanned * 64), "CumulativeBytesMetered": float64(max(scanned*64, 10<<20))},
	}
	if end < len(rows) {
		if _, ok := st.cursors[qid]; !ok {
			st.cursors[qid] = &tsmCursor{sql: sql, cols: cols, rows: rows}
			st.cursorQ = append(st.cursorQ, qid)
			st.cursorRows += len(rows)
			// Bound both the number of open cursors and the rows they hold.
			for len(st.cursorQ) > tsmMaxCursors || (st.cursorRows > tsmMaxCursorRows && len(st.cursorQ) > 1) {
				st.dropCursor(st.cursorQ[0])
			}
		}
		out["NextToken"] = qid + ":" + strconv.Itoa(end)
	} else if _, ok := st.cursors[qid]; ok {
		st.dropCursor(qid)
	}
	return out, nil
}

func (st *tsmStore) dropCursor(id string) {
	if cur := st.cursors[id]; cur != nil {
		st.cursorRows -= len(cur.rows)
	}
	delete(st.cursors, id)
	for i, q := range st.cursorQ {
		if q == id {
			st.cursorQ = append(st.cursorQ[:i], st.cursorQ[i+1:]...)
			break
		}
	}
}

func (st *tsmStore) cancelQuery(call *tsmCall) (map[string]any, *tsmError) {
	st.dropCursor(tsmStr(call.in, "QueryId"))
	return map[string]any{"CancellationMessage": "Query " + tsmStr(call.in, "QueryId") + " has been cancelled."}, nil
}

func (st *tsmStore) prepareQuery(call *tsmCall) (map[string]any, *tsmError) {
	sql := tsmStr(call.in, "QueryString")
	q, perr := tsqParse(sql)
	if perr != nil {
		return nil, tsqQueryError("%v", perr)
	}
	params := map[string]tsqVal{}
	for _, p := range q.params {
		params[p] = tsqVal{typ: "TIMESTAMP", i: call.now.UnixNano()}
	}
	res, err := st.runQuery(sql, params, call.now)
	if err != nil {
		return nil, err
	}
	cols := make([]any, len(res.cols))
	for i, c := range res.cols {
		cols[i] = map[string]any{"Name": c.name, "Type": map[string]any{"ScalarType": c.typ},
			"DatabaseName": q.db, "TableName": q.tbl, "Aliased": res.aliased[i]}
	}
	ps := make([]any, 0, len(q.params))
	for _, p := range q.params {
		ps = append(ps, map[string]any{"Name": p, "Type": map[string]any{"ScalarType": "TIMESTAMP"}})
	}
	return map[string]any{"QueryString": sql, "Columns": cols, "Parameters": ps}, nil
}

// --- scheduled queries ---

type tsmScheduled struct {
	arn, name string
	in        map[string]any
	state     string
	created   time.Time
	prev      time.Time
	last      map[string]any
	failed    []any
	tokens    map[string]bool // ExecuteScheduledQuery ClientTokens already run
}

var tsmRateExpr = regexp.MustCompile(`^rate\((\d+) (minute|minutes|hour|hours|day|days)\)$`)

func (sq *tsmScheduled) next(now time.Time) (time.Time, bool) {
	m := tsmRateExpr.FindStringSubmatch(tsmStr(tsmMap(sq.in, "ScheduleConfiguration"), "ScheduleExpression"))
	if m == nil {
		return time.Time{}, false // ponytail: no next-run time for cron() schedules
	}
	n, _ := strconv.ParseInt(m[1], 10, 32)
	unit := map[string]time.Duration{"minute": time.Minute, "hour": time.Hour, "day": 24 * time.Hour}[strings.TrimSuffix(m[2], "s")]
	base := sq.created
	if !sq.prev.IsZero() {
		base = sq.prev
	}
	next := base.Add(time.Duration(n) * unit)
	for next.Before(now) {
		next = next.Add(time.Duration(n) * unit)
	}
	return next, true
}

func (sq *tsmScheduled) target() map[string]any {
	return tsmPath(sq.in, "TargetConfiguration", "TimestreamConfiguration")
}

func (sq *tsmScheduled) summary(now time.Time) map[string]any {
	m := map[string]any{"Arn": sq.arn, "Name": sq.name, "State": sq.state, "CreationTime": tsmEpoch(sq.created),
		"ErrorReportConfiguration": sq.in["ErrorReportConfiguration"]}
	if t := sq.target(); t != nil {
		m["TargetDestination"] = map[string]any{"TimestreamDestination": map[string]any{"DatabaseName": t["DatabaseName"], "TableName": t["TableName"]}}
	}
	if !sq.prev.IsZero() {
		m["PreviousInvocationTime"] = tsmEpoch(sq.prev)
	}
	if next, ok := sq.next(now); ok && sq.state == "ENABLED" {
		m["NextInvocationTime"] = tsmEpoch(next)
	}
	if sq.last != nil {
		m["LastRunStatus"] = sq.last["RunStatus"]
	}
	return m
}

func (sq *tsmScheduled) description(now time.Time) map[string]any {
	m := sq.summary(now)
	delete(m, "TargetDestination")
	delete(m, "LastRunStatus")
	for _, k := range []string{"QueryString", "ScheduleConfiguration", "NotificationConfiguration", "TargetConfiguration", "ScheduledQueryExecutionRoleArn", "KmsKeyId"} {
		if v, ok := sq.in[k]; ok {
			m[k] = v
		}
	}
	if sq.last != nil {
		m["LastRunSummary"] = sq.last
	}
	if len(sq.failed) > 0 {
		m["RecentlyFailedRuns"] = sq.failed
	}
	return m
}

func (st *tsmStore) scheduled(arn string) (*tsmScheduled, *tsmError) {
	if sq := st.sched[arn]; sq != nil {
		return sq, nil
	}
	return nil, tsmNotFound("The scheduled query %s does not exist.", arn)
}

func (st *tsmStore) createScheduledQuery(call *tsmCall) (map[string]any, *tsmError) {
	in := call.in
	if tok := tsmStr(in, "ClientToken"); tok != "" {
		if arn, ok := st.sqToken[tok]; ok {
			return map[string]any{"Arn": arn}, nil
		}
	}
	name := tsmStr(in, "Name")
	for _, sq := range st.sched {
		if sq.name == name {
			return nil, tsmConflict("A scheduled query named %s already exists.", name)
		}
	}
	if len(st.sched) >= tsmMaxScheduled {
		return nil, tsmQuota("You have reached the maximum number of scheduled queries (%d).", tsmMaxScheduled)
	}
	expr := tsmStr(tsmMap(in, "ScheduleConfiguration"), "ScheduleExpression")
	if !strings.HasPrefix(expr, "rate(") && !strings.HasPrefix(expr, "cron(") {
		return nil, tsmValidation("ScheduleExpression must be a rate() or cron() expression.")
	}
	if err := schedulerValidateExpression(expr, ""); err != nil {
		return nil, tsmValidation("%s", err.msg)
	}
	if !call.topicOK {
		return nil, tsmValidation("The SNS topic %s does not exist.", tsmStr(tsmPath(in, "NotificationConfiguration", "SnsConfiguration"), "TopicArn"))
	}
	s3 := tsmPath(in, "ErrorReportConfiguration", "S3Configuration")
	if !call.buckets[tsmStr(s3, "BucketName")] {
		return nil, tsmValidation("The error report S3 bucket %s does not exist.", tsmStr(s3, "BucketName"))
	}
	q, perr := tsqParse(tsmStr(in, "QueryString"))
	if perr != nil {
		return nil, tsqQueryError("%v", perr)
	}
	for _, p := range q.params {
		if p != tsqScheduledTime {
			return nil, tsmValidation("Scheduled queries only support the %s parameter, not %s.", tsqScheduledTime, p)
		}
	}
	if _, err := st.table(q.db, q.tbl); err != nil {
		return nil, tsmValidation("%s", err.msg)
	}
	if t := tsmPath(in, "TargetConfiguration", "TimestreamConfiguration"); t != nil {
		if _, err := st.table(tsmStr(t, "DatabaseName"), tsmStr(t, "TableName")); err != nil {
			return nil, tsmValidation("Target %s", err.msg)
		}
		if tsmMap(t, "MultiMeasureMappings") == nil && len(tsmList(t, "MixedMeasureMappings")) == 0 {
			return nil, tsmValidation("TimestreamConfiguration needs MultiMeasureMappings or MixedMeasureMappings.")
		}
	}
	tags, terr := tsmTagsIn(tsmList(in, "Tags"))
	if terr != nil {
		return nil, terr
	}
	suffix := make([]byte, 4)
	_, _ = rand.Read(suffix)
	sq := &tsmScheduled{
		arn:  fmt.Sprintf("arn:aws:timestream:%s:%s:scheduled-query/%s-%s", tsmRegion, tsmAccount, name, hex.EncodeToString(suffix)),
		name: name, in: in, state: "ENABLED", created: call.now, tokens: map[string]bool{},
	}
	st.sched[sq.arn] = sq
	st.schedIDs = append(st.schedIDs, sq.arn)
	if len(tags) > 0 {
		st.tags[sq.arn] = tags
	}
	if tok := tsmStr(in, "ClientToken"); tok != "" {
		st.sqToken[tok] = sq.arn
	}
	return map[string]any{"Arn": sq.arn}, nil
}

func (st *tsmStore) describeScheduledQuery(call *tsmCall) (map[string]any, *tsmError) {
	sq, err := st.scheduled(tsmStr(call.in, "ScheduledQueryArn"))
	if err != nil {
		return nil, err
	}
	return map[string]any{"ScheduledQuery": sq.description(call.now)}, nil
}

func (st *tsmStore) updateScheduledQuery(call *tsmCall) (map[string]any, *tsmError) {
	sq, err := st.scheduled(tsmStr(call.in, "ScheduledQueryArn"))
	if err != nil {
		return nil, err
	}
	sq.state = tsmStr(call.in, "State")
	return map[string]any{}, nil
}

func (st *tsmStore) deleteScheduledQuery(call *tsmCall) (map[string]any, *tsmError) {
	sq, err := st.scheduled(tsmStr(call.in, "ScheduledQueryArn"))
	if err != nil {
		return nil, err
	}
	delete(st.sched, sq.arn)
	delete(st.tags, sq.arn)
	for i, a := range st.schedIDs {
		if a == sq.arn {
			st.schedIDs = append(st.schedIDs[:i], st.schedIDs[i+1:]...)
			break
		}
	}
	return map[string]any{}, nil
}

func (st *tsmStore) listScheduledQueries(call *tsmCall) (map[string]any, *tsmError) {
	page, next, err := tsmPage(st.schedIDs, tsmStr(call.in, "NextToken"), tsmInt(call.in, "MaxResults", 0), 100)
	if err != nil {
		return nil, err
	}
	items := make([]any, 0, len(page))
	for _, arn := range page {
		items = append(items, st.sched[arn].summary(call.now))
	}
	out := map[string]any{"ScheduledQueries": items}
	if next != "" {
		out["NextToken"] = next
	}
	return out, nil
}

func (st *tsmStore) executeScheduledQuery(call *tsmCall) (map[string]any, *tsmError) {
	sq, err := st.scheduled(tsmStr(call.in, "ScheduledQueryArn"))
	if err != nil {
		return nil, err
	}
	if tok := tsmStr(call.in, "ClientToken"); tok != "" {
		if sq.tokens[tok] {
			return map[string]any{}, nil
		}
		if len(sq.tokens) < 1000 {
			sq.tokens[tok] = true
		}
	}
	f, _ := call.in["InvocationTime"].(float64)
	invocation := time.Unix(0, int64(f*1e9)).UTC()
	start := time.Now()
	res, qerr := st.runQuery(tsmStr(sq.in, "QueryString"), map[string]tsqVal{tsqScheduledTime: {typ: "TIMESTAMP", i: invocation.UnixNano()}}, call.now)
	ingested := 0
	var failure string
	if qerr != nil {
		failure = qerr.msg
	} else if t := sq.target(); t != nil {
		ingested, failure = st.writeQueryResults(t, res, call.now)
	}
	run := map[string]any{
		"InvocationTime": tsmEpoch(invocation), "TriggerTime": tsmEpoch(call.now), "RunStatus": "MANUAL_TRIGGER_SUCCESS",
		"ExecutionStats": map[string]any{"ExecutionTimeInMillis": float64(time.Since(start).Milliseconds()), "DataWrites": float64(ingested * 64),
			"BytesMetered": float64(10 << 20), "CumulativeBytesScanned": float64(tsqScanned(res) * 64), "RecordsIngested": float64(ingested), "QueryResultRows": float64(tsqRowCount(res))},
	}
	if failure != "" {
		run["RunStatus"], run["FailureReason"] = "MANUAL_TRIGGER_FAILURE", failure
		s3 := tsmPath(sq.in, "ErrorReportConfiguration", "S3Configuration")
		prefix := strings.TrimSuffix(tsmStr(s3, "ObjectKeyPrefix"), "/")
		if prefix != "" {
			prefix += "/"
		}
		key := fmt.Sprintf("%s%s/%d.json", prefix, sq.name, invocation.Unix())
		report, _ := json.Marshal(map[string]any{"scheduledQueryArn": sq.arn, "invocationTime": invocation.Format(time.RFC3339), "error": failure})
		call.putS3(tsmStr(s3, "BucketName"), key, report)
		run["ErrorReportLocation"] = map[string]any{"S3ReportLocation": map[string]any{"BucketName": tsmStr(s3, "BucketName"), "ObjectKey": key}}
		sq.failed = append([]any{run}, sq.failed...)
		if len(sq.failed) > 5 {
			sq.failed = sq.failed[:5]
		}
	}
	sq.last, sq.prev = run, invocation
	note, _ := json.Marshal(map[string]any{"type": run["RunStatus"], "arn": sq.arn, "nextInvocationEpochSecond": invocation.Unix(), "message": failure})
	call.publish(tsmStr(tsmPath(sq.in, "NotificationConfiguration", "SnsConfiguration"), "TopicArn"), string(note))
	return map[string]any{}, nil
}

func tsqScanned(r *tsqResult) int {
	if r == nil {
		return 0
	}
	return r.scanned
}

func tsqRowCount(r *tsqResult) int {
	if r == nil {
		return 0
	}
	return len(r.rows)
}

// writeQueryResults writes scheduled-query result rows into the target table.
func (st *tsmStore) writeQueryResults(t map[string]any, res *tsqResult, now time.Time) (int, string) {
	table, err := st.table(tsmStr(t, "DatabaseName"), tsmStr(t, "TableName"))
	if err != nil {
		return 0, err.msg
	}
	idx := map[string]int{}
	for i, c := range res.cols {
		idx[c.name] = i
	}
	get := func(row []tsqVal, name string) (tsqVal, bool) {
		i, ok := idx[name]
		if !ok {
			return tsqVal{}, false
		}
		return row[i], true
	}
	raw := func(v tsqVal) string {
		if v.typ == "TIMESTAMP" {
			return strconv.FormatInt(v.i/1_000_000, 10) // TIMESTAMP measures are epoch milliseconds
		}
		return v.format()
	}
	attrs := func(row []tsqVal, list []any) []any {
		var out []any
		for _, item := range list {
			m, _ := item.(map[string]any)
			v, ok := get(row, tsmStr(m, "SourceColumn"))
			if !ok || v.null() {
				continue
			}
			name := tsmStr(m, "TargetMultiMeasureAttributeName")
			if name == "" {
				name = tsmStr(m, "SourceColumn")
			}
			out = append(out, map[string]any{"Name": name, "Value": raw(v), "Type": tsmStr(m, "MeasureValueType")})
		}
		return out
	}
	written := 0
	for _, row := range res.rows {
		tv, ok := get(row, tsmStr(t, "TimeColumn"))
		if !ok || tv.typ != "TIMESTAMP" {
			return written, fmt.Sprintf("The time column %s is missing from the result or isn't a TIMESTAMP.", tsmStr(t, "TimeColumn"))
		}
		r := map[string]any{"Time": strconv.FormatInt(tv.i, 10), "TimeUnit": "NANOSECONDS"}
		var dims []any
		for _, item := range tsmList(t, "DimensionMappings") {
			m, _ := item.(map[string]any)
			if v, ok := get(row, tsmStr(m, "Name")); ok && !v.null() {
				dims = append(dims, map[string]any{"Name": tsmStr(m, "Name"), "Value": v.format()})
			}
		}
		r["Dimensions"] = dims
		var recs []map[string]any
		if mm := tsmMap(t, "MultiMeasureMappings"); mm != nil {
			name := tsmStr(mm, "TargetMultiMeasureName")
			if name == "" {
				v, _ := get(row, tsmStr(t, "MeasureNameColumn"))
				name = v.format()
			}
			r["MeasureName"], r["MeasureValueType"], r["MeasureValues"] = name, "MULTI", attrs(row, tsmList(mm, "MultiMeasureAttributeMappings"))
			recs = append(recs, r)
		}
		for _, item := range tsmList(t, "MixedMeasureMappings") {
			m, _ := item.(map[string]any)
			measure, _ := get(row, tsmStr(t, "MeasureNameColumn"))
			if want := tsmStr(m, "MeasureName"); want != "" && want != measure.format() {
				continue
			}
			rec := map[string]any{"Time": r["Time"], "TimeUnit": r["TimeUnit"], "Dimensions": dims}
			name := tsmStr(m, "TargetMeasureName")
			if name == "" {
				name = tsmStr(m, "SourceColumn")
			}
			if tsmStr(m, "MeasureValueType") == "MULTI" {
				rec["MeasureName"], rec["MeasureValueType"], rec["MeasureValues"] = name, "MULTI", attrs(row, tsmList(m, "MultiMeasureAttributeMappings"))
			} else {
				v, _ := get(row, tsmStr(m, "SourceColumn"))
				rec["MeasureName"], rec["MeasureValueType"], rec["MeasureValue"] = name, tsmStr(m, "MeasureValueType"), raw(v)
			}
			recs = append(recs, rec)
		}
		for _, rm := range recs {
			rec, reason := tsmBuildRecord(nil, rm)
			if reason == "" {
				_, reason, _ = st.ingest(table, rec, now)
			}
			if reason != "" {
				return written, "A result row could not be written to the target table: " + reason
			}
			written++
		}
	}
	return written, ""
}

// --- account settings ---

func (st *tsmStore) settingsOut() map[string]any {
	compute := map[string]any{"ComputeMode": st.settings.mode}
	if st.settings.mode == "PROVISIONED" {
		pc := map[string]any{"ActiveQueryTCU": float64(st.settings.target),
			"LastUpdate": map[string]any{"TargetQueryTCU": float64(st.settings.target), "Status": "SUCCEEDED"}}
		if st.settings.notify != nil {
			pc["NotificationConfiguration"] = st.settings.notify
		}
		compute["ProvisionedCapacity"] = pc
	}
	return map[string]any{"MaxQueryTCU": float64(st.settings.maxTCU), "QueryPricingModel": st.settings.pricing, "QueryCompute": compute}
}

func (st *tsmStore) describeAccountSettings(call *tsmCall) (map[string]any, *tsmError) {
	return st.settingsOut(), nil
}

func (st *tsmStore) updateAccountSettings(call *tsmCall) (map[string]any, *tsmError) {
	next := st.settings
	if p := tsmStr(call.in, "QueryPricingModel"); p != "" {
		next.pricing = p
	}
	if v, ok := call.in["MaxQueryTCU"].(float64); ok {
		if int64(v)%4 != 0 {
			return nil, tsmValidation("MaxQueryTCU must be a multiple of 4.")
		}
		next.maxTCU = int64(v)
	}
	if qc := tsmMap(call.in, "QueryCompute"); qc != nil {
		if mode := tsmStr(qc, "ComputeMode"); mode != "" {
			next.mode = mode
		}
		pc := tsmMap(qc, "ProvisionedCapacity")
		switch next.mode {
		case "PROVISIONED":
			if pc == nil {
				return nil, tsmValidation("ProvisionedCapacity is required when ComputeMode is PROVISIONED.")
			}
			next.target = tsmInt(pc, "TargetQueryTCU", 0)
			if next.target%4 != 0 {
				return nil, tsmValidation("TargetQueryTCU must be a multiple of 4.")
			}
			next.notify = tsmMap(pc, "NotificationConfiguration")
		default:
			if pc != nil {
				return nil, tsmValidation("ProvisionedCapacity can only be set when ComputeMode is PROVISIONED.")
			}
			next.target, next.notify = 0, nil
		}
	}
	if next.mode == "PROVISIONED" && next.pricing != "COMPUTE_UNITS" {
		return nil, tsmValidation("Provisioned query capacity requires QueryPricingModel COMPUTE_UNITS.")
	}
	next.updated = call.now
	st.settings = next
	return st.settingsOut(), nil
}
