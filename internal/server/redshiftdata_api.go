package server

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

func (s *Server) redshiftDataDispatch(action string, req *rsdRequest, body []byte) (any, *rsdErr) {
	if req.WaitTimeSeconds != nil && (*req.WaitTimeSeconds < 1 || *req.WaitTimeSeconds > 30) {
		return nil, rsdBad("WaitTimeSeconds must be between 1 and 30")
	}
	s.pruneRedshiftDataTargets()
	switch action {
	case "ExecuteStatement":
		return s.redshiftDataSubmit(req, body, false)
	case "BatchExecuteStatement":
		return s.redshiftDataSubmit(req, body, true)
	case "DescribeStatement", "GetStatementResult", "GetStatementResultV2", "CancelStatement":
		if !rsdUUIDRE.MatchString(req.Id) {
			return nil, rsdBad("Id %q doesn't match the required pattern", req.Id)
		}
		return s.redshiftDataStatementCall(action, req)
	case "ListStatements":
		return s.redshiftDataListStatements(req)
	case "ListSessions":
		return s.redshiftDataListSessions(req)
	}
	return s.redshiftDataMetadata(action, req)
}

func rsdTS(t time.Time) float64 { return float64(t.UnixMilli()) / 1000 }

func rsdRange(v *int, name string, lo, hi int) *rsdErr {
	if v != nil && (*v < lo || *v > hi) {
		return rsdBad("%s must be between %d and %d", name, lo, hi)
	}
	return nil
}

// rsdPage slices n items starting at NextToken (an offset) into a page of at
// most limit items.
func rsdPage(n int, token string, maxResults *int, def int) (int, int, string, *rsdErr) {
	start := 0
	if token != "" {
		v, err := strconv.Atoi(token)
		if err != nil || v < 0 || v > n {
			return 0, 0, "", rsdBad("NextToken %q is invalid", token)
		}
		start = v
	}
	limit := def
	if maxResults != nil && *maxResults > 0 {
		limit = *maxResults
	}
	end := min(n, start+limit)
	next := ""
	if end < n {
		next = strconv.Itoa(end)
	}
	return start, end, next, nil
}

func (s *Server) redshiftDataSubmit(req *rsdRequest, body []byte, batch bool) (any, *rsdErr) {
	sqls := []string{req.Sql}
	if batch {
		if len(req.Sqls) < 1 || len(req.Sqls) > 40 {
			return nil, rsdBad("Sqls must contain between 1 and 40 SQL statements")
		}
		sqls = req.Sqls
	}
	for _, sql := range sqls {
		if strings.TrimSpace(sql) == "" {
			return nil, rsdBad("SQL statements must not be empty")
		}
		if len(sql) > rsdMaxSQLLength {
			return nil, rsdBad("SQL statements must be at most %d bytes", rsdMaxSQLLength)
		}
	}
	if req.Parameters != nil && len(req.Parameters) == 0 {
		return nil, rsdBad("Parameters must contain at least one item")
	}
	seen := map[string]bool{}
	for _, p := range req.Parameters {
		if !rsdParamNameRE.MatchString(p.Name) || p.Value == "" {
			return nil, rsdBad("parameter %q needs a name matching [0-9a-zA-Z_]+ and a non-empty value", p.Name)
		}
		if seen[p.Name] {
			return nil, rsdBad("parameter %q is specified more than once", p.Name)
		}
		seen[p.Name] = true
	}
	format := req.ResultFormat
	if format == "" {
		format = "JSON"
	}
	if format != "JSON" && format != "CSV" {
		return nil, rsdBad("ResultFormat must be JSON or CSV")
	}
	mode := ""
	if batch {
		mode = req.ExecutionMode
		if mode == "" {
			mode = "TRANSACTION"
		}
		if mode != "TRANSACTION" && mode != "AUTO_COMMIT" {
			return nil, rsdBad("ExecutionMode must be TRANSACTION or AUTO_COMMIT")
		}
	}
	if req.ClientToken != nil && (len(*req.ClientToken) < 1 || len(*req.ClientToken) > 64) {
		return nil, rsdBad("ClientToken must be 1 to 64 characters")
	}
	if req.StatementName != nil && len(*req.StatementName) > 2048 {
		return nil, rsdBad("StatementName must be at most 2048 characters")
	}
	if e := rsdRange(req.SessionKeepAliveSeconds, "SessionKeepAliveSeconds", 0, 86400); e != nil {
		return nil, e
	}

	st := s.redshiftData
	conn, database := *req, req.Database
	if req.SessionId != "" {
		if !rsdUUIDRE.MatchString(req.SessionId) {
			return nil, rsdBad("SessionId %q doesn't match the required pattern", req.SessionId)
		}
		st.mu.Lock()
		st.settleLocked()
		sess, ok := st.sessions[req.SessionId]
		var copied rsdSession
		if ok {
			copied = *sess
		}
		st.mu.Unlock()
		if !ok {
			return nil, rsdNotFound(req.SessionId, "Session %s does not exist", req.SessionId)
		}
		for _, pair := range [][2]string{{req.ClusterIdentifier, copied.conn.ClusterIdentifier}, {req.WorkgroupName, copied.conn.WorkgroupName}, {req.Database, copied.database}} {
			if pair[0] != "" && pair[0] != pair[1] {
				return nil, rsdBad("ClusterIdentifier, WorkgroupName and Database must match the session's when SessionId is specified")
			}
		}
		conn, database = copied.conn, copied.database
	} else if database == "" {
		return nil, rsdBad("Database is required")
	}
	target, rerr := s.resolveRedshiftDataTarget(&conn)
	if rerr != nil {
		return nil, rerr
	}
	fp := rsdFingerprint(req, batch)

	st.mu.Lock()
	defer st.mu.Unlock()
	st.settleLocked()
	if req.ClientToken != nil {
		if prior, ok := st.tokens[*req.ClientToken]; ok {
			stmt, live := st.statements[prior.statementID]
			switch {
			case live && prior.fingerprint == fp:
				return st.submitOutputLocked(stmt), nil
			case live:
				return nil, rsdBad("ClientToken %s was already used with different parameters", *req.ClientToken)
			}
			delete(st.tokens, *req.ClientToken) // its statement was pruned: treat as a new request
		}
	}
	now := st.now()
	stmt := &rsdStatement{
		id: st.newIDLocked(), sql: sqls[0], batch: batch, params: req.Parameters, target: target,
		database: database, format: format, execMode: mode, status: "SUBMITTED", created: now, updated: now, resultRows: -1,
	}
	if req.StatementName != nil {
		stmt.name = *req.StatementName
	}
	if batch {
		stmt.sql = ""
		stmt.subSQLs = sqls
		for i, sql := range sqls {
			stmt.subs = append(stmt.subs, &rsdStatement{
				id: stmt.id + ":" + strconv.Itoa(i+1), sql: sql, target: target, database: database, format: format,
				status: "SUBMITTED", created: now, updated: now, resultRows: -1, parent: stmt,
			})
		}
	}
	switch {
	case req.SessionId != "":
		sess, ok := st.sessions[req.SessionId]
		if !ok || !now.Before(sess.ttl) {
			return nil, rsdBad("Session %s is closed", req.SessionId)
		}
		if req.SessionKeepAliveSeconds != nil {
			sess.keepAlive = *req.SessionKeepAliveSeconds
		}
		stmt.sessionID, sess.lastStatement, sess.updated = sess.id, stmt.id, now
	case req.SessionKeepAliveSeconds != nil && *req.SessionKeepAliveSeconds > 0:
		saved := conn
		saved.SessionId, saved.Sql, saved.Sqls, saved.Parameters = "", "", nil, nil
		sess := &rsdSession{
			id: st.newIDLocked(), target: target, conn: saved, database: database, keepAlive: *req.SessionKeepAliveSeconds,
			created: now, updated: now, lastStatement: stmt.id,
		}
		sess.ttl = now.Add(time.Duration(sess.keepAlive) * time.Second)
		st.sessions[sess.id] = sess
		stmt.sessionID = sess.id
	}
	st.statements[stmt.id] = stmt
	st.order = append(st.order, stmt)
	st.pending = stmt
	if req.ClientToken != nil {
		st.tokens[*req.ClientToken] = rsdClientToken{statementID: stmt.id, fingerprint: fp, created: now}
	}
	if req.WaitTimeSeconds != nil {
		st.settleLocked()
	}
	return st.submitOutputLocked(stmt), nil
}

func rsdFingerprint(req *rsdRequest, batch bool) string {
	cp := *req
	cp.ClientToken, cp.WaitTimeSeconds = nil, nil
	b, _ := json.Marshal(cp)
	return strconv.FormatBool(batch) + string(b)
}

func rsdTargetFields(out map[string]any, t rsdTarget, database string) {
	out["Database"] = database
	out["DbUser"] = t.dbUser
	if t.cluster != "" {
		out["ClusterIdentifier"] = t.cluster
	} else {
		out["WorkgroupName"] = t.workgroup
	}
	if t.secretArn != "" {
		out["SecretArn"] = t.secretArn
	}
}

func (st *redshiftDataStore) submitOutputLocked(stmt *rsdStatement) map[string]any {
	out := map[string]any{"Id": stmt.id, "CreatedAt": rsdTS(stmt.created), "Status": stmt.status, "RedshiftPid": st.targetLocked(stmt.target).pid}
	rsdTargetFields(out, stmt.target, stmt.database)
	if stmt.sessionID != "" {
		out["SessionId"] = stmt.sessionID
	}
	if stmt.status != "SUBMITTED" {
		out["HasResultSet"] = stmt.hasResultSet
	}
	return out
}

func (st *redshiftDataStore) findLocked(id string) *rsdStatement {
	parentID, idx, isSub := strings.Cut(id, ":")
	parent := st.statements[parentID]
	if !isSub || parent == nil {
		return parent
	}
	n, err := strconv.Atoi(idx)
	if !parent.batch || err != nil || n < 1 || n > len(parent.subs) {
		return nil
	}
	return parent.subs[n-1]
}

func (s *Server) redshiftDataStatementCall(action string, req *rsdRequest) (any, *rsdErr) {
	st := s.redshiftData
	st.mu.Lock()
	defer st.mu.Unlock()
	if action == "CancelStatement" {
		stmt := st.findLocked(req.Id)
		if stmt == nil {
			return nil, rsdNotFound(req.Id, "Query does not exist.")
		}
		if stmt.parent != nil {
			stmt = stmt.parent
		}
		if st.pending == stmt {
			st.pending = nil
			now := st.now()
			stmt.status, stmt.updated = "ABORTED", now
			for _, sub := range stmt.subs {
				sub.status, sub.updated = "ABORTED", now
			}
			return map[string]any{"Status": true}, nil
		}
		st.settleLocked()
		return nil, rsdBad("Could not cancel a query that is already in %s status", stmt.status)
	}
	st.settleLocked()
	stmt := st.findLocked(req.Id)
	if stmt == nil {
		return nil, rsdNotFound(req.Id, "Query does not exist.")
	}
	if action == "DescribeStatement" {
		return st.describeLocked(stmt), nil
	}
	if stmt.batch {
		return nil, rsdBad("%s is a batch statement; fetch results with a sub-statement Id such as %s:1", stmt.id, stmt.id)
	}
	if !stmt.hasResultSet || stmt.result == nil {
		return nil, rsdNotFound(req.Id, "Query does not have result. Please check query status with DescribeStatement.")
	}
	csvOut := action == "GetStatementResultV2"
	if csvOut && stmt.format != "CSV" {
		return nil, rsdBad("Statement %s was run with ResultFormat JSON; use GetStatementResult", req.Id)
	}
	if !csvOut && stmt.format != "JSON" {
		return nil, rsdBad("Statement %s was run with ResultFormat CSV; use GetStatementResultV2", req.Id)
	}
	res := stmt.result
	start, end, next, perr := rsdPage(len(res.rows), req.NextToken, nil, rsdResultPageRows)
	if perr != nil {
		return nil, perr
	}
	meta := make([]map[string]any, 0, len(res.cols))
	for _, c := range res.cols {
		meta = append(meta, rsdColumnMeta(c))
	}
	out := map[string]any{"ColumnMetadata": meta, "TotalNumRows": len(res.rows)}
	if next != "" {
		out["NextToken"] = next
	}
	if csvOut {
		var buf bytes.Buffer
		if start == 0 {
			header := make([]any, len(res.cols))
			for i, c := range res.cols {
				header[i] = c.name
			}
			rsdWriteCSVRow(&buf, header)
		}
		for _, row := range res.rows[start:end] {
			rsdWriteCSVRow(&buf, row)
		}
		out["Records"] = []map[string]any{{"CSVRecords": buf.String()}}
		out["ResultFormat"] = "CSV"
		return out, nil
	}
	records := make([][]map[string]any, 0, end-start)
	for _, row := range res.rows[start:end] {
		fields := make([]map[string]any, len(row))
		for i, v := range row {
			fields[i] = rsdField(v)
		}
		records = append(records, fields)
	}
	out["Records"] = records
	return out, nil
}

func (st *redshiftDataStore) describeLocked(stmt *rsdStatement) map[string]any {
	top := stmt
	if stmt.parent != nil {
		top = stmt.parent
	}
	out := map[string]any{
		"Id": stmt.id, "Status": stmt.status, "CreatedAt": rsdTS(stmt.created), "UpdatedAt": rsdTS(stmt.updated),
		"Duration": stmt.duration, "HasResultSet": stmt.hasResultSet, "ResultRows": stmt.resultRows, "ResultSize": stmt.resultSize,
		"RedshiftPid": st.targetLocked(top.target).pid, "ResultFormat": top.format,
	}
	rsdTargetFields(out, top.target, top.database)
	if stmt.errText != "" {
		out["Error"] = stmt.errText
	}
	if stmt.sql != "" {
		out["QueryString"] = stmt.sql
	}
	if stmt.queryID != 0 {
		out["RedshiftQueryId"], out["RedshiftQueryUuid"] = stmt.queryID, stmt.queryUUID
	}
	if len(top.params) > 0 {
		out["QueryParameters"] = top.params
	}
	if top.sessionID != "" {
		out["SessionId"] = top.sessionID
	}
	if stmt.batch {
		out["ExecutionMode"] = stmt.execMode
		subs := make([]map[string]any, 0, len(stmt.subs))
		for _, sub := range stmt.subs {
			item := map[string]any{
				"Id": sub.id, "Status": sub.status, "QueryString": sub.sql, "CreatedAt": rsdTS(sub.created), "UpdatedAt": rsdTS(sub.updated),
				"Duration": sub.duration, "HasResultSet": sub.hasResultSet, "ResultRows": sub.resultRows, "ResultSize": sub.resultSize,
			}
			if sub.errText != "" {
				item["Error"] = sub.errText
			}
			if sub.queryID != 0 {
				item["RedshiftQueryId"], item["RedshiftQueryUuid"] = sub.queryID, sub.queryUUID
			}
			subs = append(subs, item)
		}
		out["SubStatements"] = subs
	}
	return out
}

func rsdStatusOK(status string, allowed ...string) bool {
	for _, a := range allowed {
		if status == a {
			return true
		}
	}
	return false
}

func (s *Server) redshiftDataListStatements(req *rsdRequest) (any, *rsdErr) {
	if req.ClusterIdentifier != "" && req.WorkgroupName != "" {
		return nil, rsdBad("ClusterIdentifier and WorkgroupName can't both be specified")
	}
	if e := rsdRange(req.MaxResults, "MaxResults", 0, 100); e != nil {
		return nil, e
	}
	status := req.Status
	if status == "" {
		status = "FINISHED" // "By default, only finished statements are shown."
	}
	if !rsdStatusOK(status, "SUBMITTED", "PICKED", "STARTED", "FINISHED", "ABORTED", "FAILED", "ALL") {
		return nil, rsdBad("Status %q is not valid", req.Status)
	}
	if req.StatementName != nil && len(*req.StatementName) > 2048 {
		return nil, rsdBad("StatementName must be at most 2048 characters")
	}
	st := s.redshiftData
	st.mu.Lock()
	defer st.mu.Unlock()
	st.settleLocked()
	var matched []*rsdStatement
	for i := len(st.order) - 1; i >= 0; i-- { // newest first
		stmt := st.order[i]
		t := stmt.target
		switch {
		case status != "ALL" && stmt.status != status,
			req.StatementName != nil && !strings.HasPrefix(stmt.name, *req.StatementName),
			req.ClusterIdentifier != "" && t.cluster != req.ClusterIdentifier,
			req.WorkgroupName != "" && t.workgroup != req.WorkgroupName && t.workgroupARN != req.WorkgroupName,
			req.Database != "" && stmt.database != req.Database:
			continue
		}
		matched = append(matched, stmt)
	}
	start, end, next, perr := rsdPage(len(matched), req.NextToken, req.MaxResults, 100)
	if perr != nil {
		return nil, perr
	}
	items := make([]map[string]any, 0, end-start)
	for _, stmt := range matched[start:end] {
		item := map[string]any{
			"Id": stmt.id, "Status": stmt.status, "CreatedAt": rsdTS(stmt.created), "UpdatedAt": rsdTS(stmt.updated),
			"IsBatchStatement": stmt.batch, "ResultFormat": stmt.format,
		}
		if stmt.batch {
			item["QueryStrings"] = stmt.subSQLs
		} else {
			item["QueryString"] = stmt.sql
		}
		if stmt.name != "" {
			item["StatementName"] = stmt.name
		}
		if stmt.target.secretArn != "" {
			item["SecretArn"] = stmt.target.secretArn
		}
		if len(stmt.params) > 0 {
			item["QueryParameters"] = stmt.params
		}
		if stmt.sessionID != "" {
			item["SessionId"] = stmt.sessionID
		}
		items = append(items, item)
	}
	out := map[string]any{"Statements": items}
	if next != "" {
		out["NextToken"] = next
	}
	return out, nil
}

func (s *Server) redshiftDataListSessions(req *rsdRequest) (any, *rsdErr) {
	if req.SessionId != "" && (req.Status != "" || req.ClusterIdentifier != "" || req.WorkgroupName != "" || req.Database != "") {
		return nil, rsdBad("SessionId can't be combined with Status, ClusterIdentifier, WorkgroupName or Database")
	}
	if req.ClusterIdentifier != "" && req.WorkgroupName != "" {
		return nil, rsdBad("ClusterIdentifier and WorkgroupName can't both be specified")
	}
	if req.Status != "" && !rsdStatusOK(req.Status, "AVAILABLE", "BUSY", "CLOSED") {
		return nil, rsdBad("Status %q is not valid", req.Status)
	}
	if e := rsdRange(req.MaxResults, "MaxResults", 0, 100); e != nil {
		return nil, e
	}
	st := s.redshiftData
	st.mu.Lock()
	defer st.mu.Unlock()
	now := st.now() // no settle here: a still-pending statement keeps its session BUSY
	sessions := make([]*rsdSession, 0, len(st.sessions))
	for _, sess := range st.sessions {
		sessions = append(sessions, sess)
	}
	sort.Slice(sessions, func(i, j int) bool { return sessions[i].created.After(sessions[j].created) })
	var items []map[string]any
	for _, sess := range sessions {
		status := "AVAILABLE"
		if !now.Before(sess.ttl) {
			status = "CLOSED"
		} else if st.pending != nil && st.pending.sessionID == sess.id {
			status = "BUSY"
		}
		t := sess.target
		switch {
		case req.SessionId != "" && sess.id != req.SessionId,
			req.SessionId == "" && req.Status == "" && status == "CLOSED",
			req.Status != "" && status != req.Status,
			req.ClusterIdentifier != "" && t.cluster != req.ClusterIdentifier,
			req.WorkgroupName != "" && t.workgroup != req.WorkgroupName && t.workgroupARN != req.WorkgroupName,
			req.Database != "" && sess.database != req.Database,
			sess.created.Before(now.Add(-rsdRetention)):
			continue
		}
		item := map[string]any{
			"SessionId": sess.id, "Status": status, "CreatedAt": rsdTS(sess.created), "UpdatedAt": rsdTS(sess.updated),
			"Database": sess.database, "DbUser": t.dbUser, "SessionAliveSeconds": sess.keepAlive, "SessionTtl": rsdTS(sess.ttl),
		}
		if t.cluster != "" {
			item["ClusterIdentifier"] = t.cluster
		} else {
			item["WorkgroupName"] = t.workgroup
		}
		if status == "BUSY" {
			item["CurrentStatementId"] = sess.lastStatement
		}
		items = append(items, item)
	}
	start, end, next, perr := rsdPage(len(items), req.NextToken, req.MaxResults, 100)
	if perr != nil {
		return nil, perr
	}
	out := map[string]any{"Sessions": append([]map[string]any{}, items[start:end]...)}
	if next != "" {
		out["NextToken"] = next
	}
	return out, nil
}

// rsdLike compiles a SQL LIKE pattern: % and _ are wildcards, \% and \_
// match them literally.
func rsdLike(pattern string) *regexp.Regexp {
	if pattern == "" {
		return nil
	}
	var b strings.Builder
	b.WriteString("(?s)^")
	for i := 0; i < len(pattern); i++ {
		switch c := pattern[i]; {
		case c == '\\' && i+1 < len(pattern):
			i++
			b.WriteString(regexp.QuoteMeta(pattern[i : i+1]))
		case c == '%':
			b.WriteString(".*")
		case c == '_':
			b.WriteString(".")
		default:
			b.WriteString(regexp.QuoteMeta(pattern[i : i+1]))
		}
	}
	b.WriteString("$")
	return regexp.MustCompile(b.String())
}

func (s *Server) redshiftDataMetadata(action string, req *rsdRequest) (any, *rsdErr) {
	if req.SessionId != "" {
		return nil, rsdBad("%s doesn't accept SessionId", action)
	}
	if req.Database == "" {
		return nil, rsdBad("Database is required")
	}
	if e := rsdRange(req.MaxResults, "MaxResults", 0, 1000); e != nil {
		return nil, e
	}
	target, rerr := s.resolveRedshiftDataTarget(req)
	if rerr != nil {
		return nil, rerr
	}
	st := s.redshiftData
	st.mu.Lock()
	defer st.mu.Unlock()
	st.settleLocked()
	td := st.targetLocked(target)
	connect := req.ConnectedDatabase
	if connect == "" {
		connect = req.Database
	}
	if td.dbs[connect] == nil {
		return nil, &rsdErr{status: http.StatusInternalServerError, code: "DatabaseConnectionException", msg: "FATAL: database \"" + connect + "\" does not exist"}
	}
	db := td.dbs[req.Database]
	if db == nil {
		db = newRSDDatabase()
		db.schemas = map[string]bool{}
	}
	schemaRE, tableRE := rsdLike(req.SchemaPattern), rsdLike(req.TablePattern)
	var listKey string
	var items []any
	switch action {
	case "ListDatabases":
		listKey = "Databases"
		for name := range td.dbs {
			items = append(items, name)
		}
	case "ListSchemas":
		listKey = "Schemas"
		names := map[string]bool{"information_schema": true, "pg_catalog": true}
		for name := range db.schemas {
			names[name] = true
		}
		for name := range names {
			if schemaRE == nil || schemaRE.MatchString(name) {
				items = append(items, name)
			}
		}
	case "ListTables":
		listKey = "Tables"
		for _, t := range db.tables {
			if (schemaRE == nil || schemaRE.MatchString(t.schema)) && (tableRE == nil || tableRE.MatchString(t.name)) {
				items = append(items, map[string]any{"name": t.name, "schema": t.schema, "type": "TABLE"})
			}
		}
	default: // DescribeTable
		listKey = "ColumnList"
		var tables []*rsdTable
		for _, t := range db.tables {
			if (req.Schema == "" || t.schema == req.Schema) && (req.Table == "" || t.name == req.Table) {
				tables = append(tables, t)
			}
		}
		sort.Slice(tables, func(i, j int) bool { return tables[i].schema+"."+tables[i].name < tables[j].schema+"."+tables[j].name })
		for _, t := range tables {
			for i := range t.cols {
				items = append(items, rsdColumnMeta(t.outCol(i)))
			}
		}
	}
	if action != "DescribeTable" {
		sort.Slice(items, func(i, j int) bool { return rsdSortKey(items[i]) < rsdSortKey(items[j]) })
	}
	start, end, next, perr := rsdPage(len(items), req.NextToken, req.MaxResults, 1000)
	if perr != nil {
		return nil, perr
	}
	out := map[string]any{listKey: append([]any{}, items[start:end]...)}
	if action == "DescribeTable" && req.Table != "" {
		out["TableName"] = req.Table
	}
	if next != "" {
		out["NextToken"] = next
	}
	return out, nil
}

func rsdSortKey(v any) string {
	if m, ok := v.(map[string]any); ok {
		return m["schema"].(string) + "." + m["name"].(string)
	}
	return v.(string)
}

func rsdField(v any) map[string]any {
	switch x := v.(type) {
	case nil:
		return map[string]any{"isNull": true}
	case int64:
		return map[string]any{"longValue": x}
	case float64:
		return map[string]any{"doubleValue": x}
	case bool:
		return map[string]any{"booleanValue": x}
	case []byte:
		return map[string]any{"blobValue": x} // encoding/json base64-encodes []byte, as the Blob shape expects
	}
	return map[string]any{"stringValue": v.(string)}
}

func rsdWriteCSVRow(buf *bytes.Buffer, row []any) {
	for i, v := range row {
		if i > 0 {
			buf.WriteByte(',')
		}
		if v == nil {
			continue // NULL is an empty, unquoted field
		}
		text := rsdCSVValue(v)
		if text == "" || strings.ContainsAny(text, ",\"\r\n") || strings.TrimSpace(text) != text {
			text = `"` + strings.ReplaceAll(text, `"`, `""`) + `"`
		}
		buf.WriteString(text)
	}
	buf.WriteByte('\n')
}

func rsdCSVValue(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case int64:
		return strconv.FormatInt(x, 10)
	case float64:
		return strconv.FormatFloat(x, 'g', -1, 64)
	case bool:
		if x {
			return "t"
		}
		return "f"
	case []byte:
		return hex.EncodeToString(x)
	}
	return v.(string)
}

func rsdColumnMeta(c rsdOutCol) map[string]any {
	nullable := 0
	if c.nullable {
		nullable = 1
	}
	k := c.typ.kind
	out := map[string]any{
		"name": c.name, "label": c.name, "typeName": c.typ.name, "nullable": nullable,
		"isCaseSensitive": k == rsdKindString, "isCurrency": false,
		"isSigned":  k == rsdKindLong || k == rsdKindDouble || k == rsdKindNumeric,
		"precision": c.typ.precision, "scale": c.typ.scale, "length": c.typ.length,
		"schemaName": c.schema, "tableName": c.table,
	}
	if c.def != "" {
		out["columnDefault"] = c.def
	}
	return out
}
