package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/stackyard/stackyard/internal/services/secretsmanager"
)

// Amazon Redshift Data API (awsJson1_1, target prefix RedshiftData, SigV4
// name redshift-data). Statements target a provisioned cluster from the
// Redshift emulation or a workgroup from Redshift Serverless and run on the
// small SQL engine in redshiftdata_sql*.go.

const (
	rsdTargetPrefix   = "RedshiftData."
	rsdRetention      = 24 * time.Hour // results, statements and sessions are kept for 24 hours
	rsdResultPageRows = 1000
	rsdMaxStatements  = 1000
)

var (
	rsdUUIDRE      = regexp.MustCompile(`^[a-z0-9]{8}(-[a-z0-9]{4}){3}-[a-z0-9]{12}(:\d{0,2})?$`)
	rsdClusterRE   = regexp.MustCompile(`^[a-z][a-z0-9]*(-[a-z0-9]+)*$`)
	rsdWorkgroupRE = regexp.MustCompile(`^([a-z0-9-]{3,63}|arn:(aws(-[a-z]+)*):redshift-serverless:([a-z]{2}(-gov|(-iso[a-z]?))?|eusc-[a-z]+)-[a-z]+-\d{1}:\d{12}:workgroup/([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}|[a-z0-9-]{1,63}/[a-z0-9-]{1,63}))$`)
	rsdParamNameRE = regexp.MustCompile(`^[0-9a-zA-Z_]+$`)
)

type rsdErr struct {
	status          int
	code, msg, rsID string
}

func rsdBad(format string, args ...any) *rsdErr {
	return &rsdErr{status: http.StatusBadRequest, code: "ValidationException", msg: fmt.Sprintf(format, args...)}
}

func rsdNotFound(id, format string, args ...any) *rsdErr {
	return &rsdErr{status: http.StatusBadRequest, code: "ResourceNotFoundException", msg: fmt.Sprintf(format, args...), rsID: id}
}

func respondRedshiftDataJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondRedshiftDataError(w http.ResponseWriter, e *rsdErr) {
	body := map[string]any{"__type": e.code, "message": e.msg}
	if e.rsID != "" {
		body["ResourceId"] = e.rsID
	}
	w.Header().Set("X-Amzn-ErrorType", e.code)
	respondRedshiftDataJSON(w, e.status, body)
}

func (s *Server) handleRedshiftDataJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	target := strings.TrimSpace(r.Header.Get("X-Amz-Target"))
	if r.Method != http.MethodPost || !strings.HasPrefix(target, rsdTargetPrefix) {
		return false
	}
	if ok, status, code, msg, _ := s.validateSigV4WithService(r, "redshift-data"); !ok {
		respondRedshiftDataError(w, &rsdErr{status: status, code: code, msg: msg})
		return true
	}
	action := strings.TrimPrefix(target, rsdTargetPrefix)
	if _, ok := redshiftdataOperationByName[action]; !ok {
		respondRedshiftDataError(w, &rsdErr{status: http.StatusBadRequest, code: "UnknownOperationException", msg: "unknown operation " + action})
		return true
	}
	body, err := readBodyBytes(r)
	if err != nil {
		respondRedshiftDataError(w, &rsdErr{status: http.StatusBadRequest, code: "SerializationException", msg: "unable to read request body"})
		return true
	}
	var req rsdRequest
	if len(bytes.TrimSpace(body)) > 0 {
		if err := json.Unmarshal(body, &req); err != nil {
			respondRedshiftDataError(w, &rsdErr{status: http.StatusBadRequest, code: "SerializationException", msg: "invalid request body: " + err.Error()})
			return true
		}
	}
	out, rerr := s.redshiftDataDispatch(action, &req, body)
	if rerr != nil {
		respondRedshiftDataError(w, rerr)
		return true
	}
	respondRedshiftDataJSON(w, http.StatusOK, out)
	return true
}

type rsdParam struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type rsdRequest struct {
	Sql                     string     `json:"Sql"`
	Sqls                    []string   `json:"Sqls"`
	ClusterIdentifier       string     `json:"ClusterIdentifier"`
	WorkgroupName           string     `json:"WorkgroupName"`
	SecretArn               string     `json:"SecretArn"`
	DbUser                  string     `json:"DbUser"`
	Database                string     `json:"Database"`
	ConnectedDatabase       string     `json:"ConnectedDatabase"`
	WithEvent               *bool      `json:"WithEvent"`
	StatementName           *string    `json:"StatementName"`
	Parameters              []rsdParam `json:"Parameters"`
	ClientToken             *string    `json:"ClientToken"`
	ResultFormat            string     `json:"ResultFormat"`
	ExecutionMode           string     `json:"ExecutionMode"`
	SessionKeepAliveSeconds *int       `json:"SessionKeepAliveSeconds"`
	SessionId               string     `json:"SessionId"`
	WaitTimeSeconds         *int       `json:"WaitTimeSeconds"`
	Id                      string     `json:"Id"`
	NextToken               string     `json:"NextToken"`
	MaxResults              *int       `json:"MaxResults"`
	Status                  string     `json:"Status"`
	RoleLevel               *bool      `json:"RoleLevel"`
	Schema                  string     `json:"Schema"`
	Table                   string     `json:"Table"`
	SchemaPattern           string     `json:"SchemaPattern"`
	TablePattern            string     `json:"TablePattern"`
}

// rsdTarget is a resolved compute target plus the credentials used.
type rsdTarget struct {
	key          string // kind:name:creation time, so a recreated cluster starts empty
	cluster      string
	workgroup    string
	workgroupARN string
	defaultDBs   []string
	dbUser       string
	secretArn    string
}

type rsdTargetData struct {
	dbs map[string]*rsdDatabase
	pid int64
}

type rsdStatement struct {
	id, name, sql, sessionID   string
	subSQLs                    []string // batch only
	batch                      bool
	params                     []rsdParam
	target                     rsdTarget
	database, format, execMode string
	status, errText            string
	created, updated           time.Time
	duration                   int64
	hasResultSet               bool
	resultRows, resultSize     int64
	queryID                    int64
	queryUUID                  string
	pid                        int64
	result                     *rsdResult
	subs                       []*rsdStatement
	parent                     *rsdStatement
}

type rsdSession struct {
	id               string
	target           rsdTarget
	conn             rsdRequest // what to re-resolve on reuse
	database         string
	keepAlive        int
	created, updated time.Time
	ttl              time.Time
	lastStatement    string
}

type rsdClientToken struct {
	statementID, fingerprint string
	created                  time.Time
}

type redshiftDataStore struct {
	mu          sync.Mutex
	now         func() time.Time
	targets     map[string]*rsdTargetData
	statements  map[string]*rsdStatement // top-level statements by Id
	order       []*rsdStatement
	pending     *rsdStatement
	sessions    map[string]*rsdSession
	tokens      map[string]rsdClientToken
	nextQueryID int64
	nextPid     int64
}

func newRedshiftDataStore() *redshiftDataStore {
	return &redshiftDataStore{
		now:         func() time.Time { return time.Now().UTC() },
		targets:     map[string]*rsdTargetData{},
		statements:  map[string]*rsdStatement{},
		sessions:    map[string]*rsdSession{},
		tokens:      map[string]rsdClientToken{},
		nextQueryID: 1000,
		nextPid:     1073741824,
	}
}

func (st *redshiftDataStore) targetLocked(t rsdTarget) *rsdTargetData {
	td, ok := st.targets[t.key]
	if !ok {
		td = &rsdTargetData{dbs: map[string]*rsdDatabase{}, pid: st.nextPid}
		st.nextPid++
		for _, name := range t.defaultDBs {
			if name != "" {
				td.dbs[name] = newRSDDatabase()
			}
		}
		st.targets[t.key] = td
	}
	return td
}

// resolveRedshiftDataTarget validates the connection parameters and looks the
// compute target and secret up in their emulations. It never holds the Redshift
// Data lock, so it can't deadlock with those stores.
func (s *Server) resolveRedshiftDataTarget(req *rsdRequest) (rsdTarget, *rsdErr) {
	var t rsdTarget
	switch {
	case req.ClusterIdentifier != "" && req.WorkgroupName != "":
		return t, rsdBad("ClusterIdentifier and WorkgroupName can't both be specified")
	case req.ClusterIdentifier == "" && req.WorkgroupName == "":
		return t, rsdBad("Either ClusterIdentifier or WorkgroupName must be specified")
	case req.ClusterIdentifier != "":
		if len(req.ClusterIdentifier) > 63 || !rsdClusterRE.MatchString(req.ClusterIdentifier) {
			return t, rsdBad("ClusterIdentifier %q doesn't match the required pattern", req.ClusterIdentifier)
		}
		s.redshiftMu.Lock()
		cluster, ok := s.redshift.clusters[req.ClusterIdentifier]
		var created time.Time
		var status, dbName string
		if ok {
			created, status, dbName = cluster.CreatedAt, cluster.Status, cluster.DBName
		}
		s.redshiftMu.Unlock()
		if !ok {
			return t, rsdBad("Redshift endpoint doesn't exist in this region: cluster %s was not found", req.ClusterIdentifier)
		}
		if status == "paused" {
			return t, rsdBad("Cluster %s is paused", req.ClusterIdentifier)
		}
		t.cluster = req.ClusterIdentifier
		t.key = rsdClusterKey(t.cluster, created)
		t.defaultDBs = []string{"dev", dbName}
	default:
		if len(req.WorkgroupName) < 3 || len(req.WorkgroupName) > 128 || !rsdWorkgroupRE.MatchString(req.WorkgroupName) {
			return t, rsdBad("WorkgroupName %q doesn't match the required pattern", req.WorkgroupName)
		}
		store := s.redshiftserverless
		store.mu.Lock()
		var wg *redshiftServerlessWorkgroup
		for _, candidate := range store.workgroups {
			if candidate.Name == req.WorkgroupName || candidate.ARN == req.WorkgroupName {
				wg = candidate
				break
			}
		}
		var name, arn, created, dbName string
		found := wg != nil && wg.Status != "DELETING"
		if found {
			name, arn, created = wg.Name, wg.ARN, wg.CreatedAtRFC
			if ns, ok := store.namespaces[wg.Namespace]; ok {
				dbName = ns.DBName
			}
		}
		store.mu.Unlock()
		if !found {
			return t, rsdBad("Redshift endpoint doesn't exist in this region: workgroup %s was not found", req.WorkgroupName)
		}
		t.workgroup, t.workgroupARN = name, arn
		t.key = rsdWorkgroupKey(name, created)
		t.defaultDBs = []string{"dev", dbName}
	}
	t.dbUser = req.DbUser
	if req.SecretArn != "" {
		t.secretArn = req.SecretArn
		secret, err := s.secretsmgr.DescribeSecret(req.SecretArn)
		if err != nil || secret.DeletedDate != nil {
			return t, rsdBad("Secrets Manager secret %s was not found", req.SecretArn)
		}
		value, err := s.secretsmgr.GetSecretValue(secretsmanager.GetSecretValueInput{SecretID: req.SecretArn})
		var creds map[string]any
		if err == nil && json.Unmarshal([]byte(value.SecretString), &creds) == nil {
			if id, _ := creds["dbClusterIdentifier"].(string); id != "" && t.cluster != "" && id != t.cluster {
				return t, rsdBad("ClusterIdentifier %s doesn't match the dbClusterIdentifier %s stored in the secret", t.cluster, id)
			}
			if user, _ := creds["username"].(string); user != "" && t.dbUser == "" {
				t.dbUser = user
			}
		}
	}
	if t.dbUser == "" {
		t.dbUser = "IAM:" + s.accessKey // ponytail: the IAM user name is the access key, there's no IAM identity lookup
	}
	return t, nil
}

func (st *redshiftDataStore) newIDLocked() string { return uuid.NewString() }

func rsdClusterKey(id string, created time.Time) string {
	return fmt.Sprintf("cluster:%s:%d", id, created.UnixNano())
}

func rsdWorkgroupKey(name, created string) string { return "workgroup:" + name + ":" + created }

// pruneRedshiftDataTargets drops the data of clusters and workgroups that no
// longer exist (or were recreated, which gives them a new key).
func (s *Server) pruneRedshiftDataTargets() {
	live := map[string]bool{}
	s.redshiftMu.Lock()
	for id, c := range s.redshift.clusters {
		live[rsdClusterKey(id, c.CreatedAt)] = true
	}
	s.redshiftMu.Unlock()
	s.redshiftserverless.mu.Lock()
	for _, wg := range s.redshiftserverless.workgroups {
		if wg.Status != "DELETING" {
			live[rsdWorkgroupKey(wg.Name, wg.CreatedAtRFC)] = true
		}
	}
	s.redshiftserverless.mu.Unlock()
	st := s.redshiftData
	st.mu.Lock()
	defer st.mu.Unlock()
	for key := range st.targets {
		if !live[key] && (st.pending == nil || st.pending.target.key != key) {
			delete(st.targets, key)
		}
	}
}

// settleLocked runs the pending statement, if any. Statements are executed
// lazily, on the next API call, so a CancelStatement right after submission can
// still abort them, as with a real queued query.
//
// ponytail: at most one statement is ever pending, because every other call
// settles it first; ActiveStatementsExceededException can't occur.
func (st *redshiftDataStore) settleLocked() {
	now := st.now()
	if p := st.pending; p != nil {
		st.pending = nil
		st.runSafeLocked(p)
	}
	cutoff := now.Add(-rsdRetention)
	kept := st.order[:0]
	for i, stmt := range st.order {
		// ponytail: only the newest rsdMaxStatements are kept, besides the 24-hour retention
		if stmt.created.Before(cutoff) || len(st.order)-i > rsdMaxStatements {
			delete(st.statements, stmt.id)
			continue
		}
		kept = append(kept, stmt)
	}
	st.order = kept
	for k, tok := range st.tokens {
		if tok.created.Before(cutoff) {
			delete(st.tokens, k)
		}
	}
	for k, sess := range st.sessions {
		if sess.ttl.Before(cutoff) {
			delete(st.sessions, k)
		}
	}
}

// runSafeLocked turns a panic in the SQL engine into a FAILED statement
// instead of crashing the request.
func (st *redshiftDataStore) runSafeLocked(stmt *rsdStatement) {
	defer func() {
		if r := recover(); r != nil {
			stmt.status, stmt.errText, stmt.result, stmt.hasResultSet = "FAILED", fmt.Sprintf("ERROR: internal error: %v", r), nil, false
			stmt.updated = st.now()
			if stmt.execMode == "TRANSACTION" {
				st.rollbackSubs(stmt)
			}
			for _, sub := range stmt.subs {
				if sub != nil && sub.status == "SUBMITTED" { // the handler itself must never panic
					sub.status, sub.updated = "ABORTED", stmt.updated
				}
			}
		}
	}()
	st.runLocked(stmt)
}

// rollbackSubs marks the sub-statements of a rolled-back transaction ABORTED
// and drops their results, since none of their work survives.
func (st *redshiftDataStore) rollbackSubs(stmt *rsdStatement) {
	for _, sub := range stmt.subs {
		if sub != nil && sub.status == "FINISHED" {
			sub.status, sub.result, sub.hasResultSet, sub.resultRows, sub.updated = "ABORTED", nil, false, -1, st.now()
		}
	}
	stmt.hasResultSet, stmt.resultRows = false, -1
}

func (st *redshiftDataStore) runLocked(stmt *rsdStatement) {
	start := time.Now()
	if stmt.sessionID != "" {
		if sess, ok := st.sessions[stmt.sessionID]; !ok || !st.now().Before(sess.ttl) {
			stmt.status, stmt.errText, stmt.updated = "FAILED", "ERROR: session "+stmt.sessionID+" is closed", st.now()
			for _, sub := range stmt.subs {
				sub.status, sub.updated = "ABORTED", stmt.updated
			}
			return
		}
	}
	td := st.targetLocked(stmt.target)
	params := make(map[string]string, len(stmt.params))
	for _, p := range stmt.params {
		params[p.Name] = p.Value
	}
	if !stmt.batch {
		st.finishOne(stmt, td, params, false, start)
	} else {
		work := td
		if stmt.execMode == "TRANSACTION" {
			work = &rsdTargetData{dbs: make(map[string]*rsdDatabase, len(td.dbs)), pid: td.pid}
			for k, v := range td.dbs { // copy-on-write: each write swaps in a cloned database
				work.dbs[k] = v
			}
		}
		failed := ""
		for _, sub := range stmt.subs {
			if failed != "" {
				sub.status, sub.updated = "ABORTED", st.now()
				continue
			}
			st.finishOne(sub, work, params, stmt.execMode == "TRANSACTION", time.Now())
			stmt.duration += sub.duration
			stmt.resultRows += max(sub.resultRows, 0)
			stmt.resultSize += sub.resultSize
			stmt.hasResultSet = stmt.hasResultSet || sub.hasResultSet
			if sub.status == "FAILED" {
				failed = sub.errText
			}
		}
		switch {
		case failed == "":
			stmt.status = "FINISHED"
			td.dbs = work.dbs
		case stmt.execMode == "TRANSACTION":
			stmt.status, stmt.errText = "FAILED", failed
			st.rollbackSubs(stmt)
		default:
			stmt.status, stmt.errText = "FAILED", failed
		}
	}
	stmt.updated = st.now()
	if stmt.sessionID != "" {
		if sess, ok := st.sessions[stmt.sessionID]; ok {
			sess.updated = stmt.updated
			sess.ttl = minTime(stmt.updated.Add(time.Duration(sess.keepAlive)*time.Second), sess.created.Add(rsdRetention))
		}
	}
}

func minTime(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}

func (st *redshiftDataStore) finishOne(stmt *rsdStatement, td *rsdTargetData, params map[string]string, inTx bool, start time.Time) {
	st.nextQueryID++
	stmt.queryID, stmt.queryUUID, stmt.pid = st.nextQueryID, uuid.NewString(), td.pid
	res, err := rsdRunSQL(td, stmt.database, stmt.sql, params, inTx)
	stmt.duration = max(time.Since(start).Nanoseconds(), 1)
	stmt.updated = st.now()
	if err != nil {
		stmt.status, stmt.errText, stmt.resultRows = "FAILED", "ERROR: "+err.Error(), -1
		return
	}
	stmt.status, stmt.result, stmt.hasResultSet = "FINISHED", &res, res.hasResultSet
	stmt.resultRows = res.affected
	if res.hasResultSet {
		stmt.resultRows = int64(len(res.rows))
		for _, row := range res.rows {
			for _, v := range row {
				stmt.resultSize += int64(len(rsdCSVValue(v)))
			}
		}
	}
}

func rsdRunSQL(td *rsdTargetData, dbName, sql string, params map[string]string, inTx bool) (rsdResult, error) {
	db, ok := td.dbs[dbName]
	if !ok {
		return rsdResult{}, fmt.Errorf("database \"%s\" does not exist", dbName)
	}
	tokens, err := rsdTokenize(sql)
	if err != nil {
		return rsdResult{}, err
	}
	stmts := rsdSplitStatements(tokens)
	switch {
	case len(stmts) == 0:
		return rsdResult{}, fmt.Errorf("syntax error at end of input")
	case len(stmts) > 1:
		return rsdResult{}, fmt.Errorf("cannot insert multiple commands into a prepared statement; use BatchExecuteStatement")
	}
	st, err := rsdParse(stmts[0])
	if err != nil {
		return rsdResult{}, err
	}
	switch st.verb {
	case "create_database":
		if inTx {
			return rsdResult{}, fmt.Errorf("CREATE DATABASE cannot run inside a transaction block")
		}
		if _, ok := td.dbs[st.table]; ok {
			return rsdResult{}, fmt.Errorf("database \"%s\" already exists", st.table)
		}
		td.dbs[st.table] = newRSDDatabase()
		return rsdResult{}, nil
	case "select":
		return rsdExec(db, st, params) // reads don't clone
	}
	cp := db.clone() // ponytail: O(database size) per write; clone only the touched table if data gets large
	res, err := rsdExec(cp, st, params)
	if err == nil {
		td.dbs[dbName] = cp
	}
	return res, err
}
