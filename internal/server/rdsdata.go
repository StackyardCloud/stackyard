package server

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	rdsDataTxIdleTimeout = 3 * time.Minute
	rdsDataTxMaxLifetime = 24 * time.Hour
)

// rdsDataStore holds the emulated databases behind the RDS Data API, keyed by
// cluster ARN and database name, plus open transactions.
// ponytail: one mutex for every database; per-cluster locks if throughput matters.
type rdsDataStore struct {
	mu  sync.Mutex
	dbs map[string]rdsDataDB
	txs map[string]*rdsDataTx
	now func() time.Time
}

// rdsDataTx runs statements against a private snapshot so it sees its own
// writes. Commit replays the recorded statements onto the live database.
type rdsDataTx struct {
	key         string
	resourceARN string
	snapshot    rdsDataDB
	statements  []rdsDataStatement
	started     time.Time
	lastUsed    time.Time
}

type rdsDataStatement struct {
	sql    string
	params map[string]any
}

func newRDSDataStore() *rdsDataStore {
	return &rdsDataStore{dbs: map[string]rdsDataDB{}, txs: map[string]*rdsDataTx{}, now: time.Now}
}

var (
	errRDSDataTxNotFound = errors.New("transaction not found")
	errRDSDataTxResource = errors.New("transaction belongs to another resource")
)

func rdsDataKey(resourceARN, database string) string {
	return resourceARN + "\x00" + database
}

// txLocked returns an open transaction, dropping it when it has been idle for
// three minutes or open for 24 hours.
func (st *rdsDataStore) txLocked(id, resourceARN string) (*rdsDataTx, error) {
	tx, ok := st.txs[id]
	if !ok {
		return nil, errRDSDataTxNotFound
	}
	now := st.now()
	if now.Sub(tx.lastUsed) > rdsDataTxIdleTimeout || now.Sub(tx.started) > rdsDataTxMaxLifetime {
		delete(st.txs, id)
		return nil, errRDSDataTxNotFound
	}
	if tx.resourceARN != resourceARN {
		return nil, errRDSDataTxResource
	}
	tx.lastUsed = now
	return tx, nil
}

// run executes statements atomically: inside a transaction against its
// snapshot, otherwise against a clone of the live database that replaces it
// only when every statement succeeds.
func (st *rdsDataStore) run(resourceARN, database, txID string, stmts []rdsDataStatement) ([]rdsDataResult, error) {
	st.mu.Lock()
	defer st.mu.Unlock()

	if txID != "" {
		tx, err := st.txLocked(txID, resourceARN)
		if err != nil {
			return nil, err
		}
		work := tx.snapshot.clone()
		results, err := work.execAll(stmts)
		if err != nil {
			return nil, err
		}
		tx.snapshot = work
		tx.statements = append(tx.statements, stmts...)
		return results, nil
	}
	key := rdsDataKey(resourceARN, database)
	work := st.dbs[key].clone()
	results, err := work.execAll(stmts)
	if err != nil {
		return nil, err
	}
	st.dbs[key] = work
	return results, nil
}

func (db rdsDataDB) execAll(stmts []rdsDataStatement) ([]rdsDataResult, error) {
	results := make([]rdsDataResult, 0, len(stmts))
	for _, stmt := range stmts {
		result, err := db.exec(stmt.sql, stmt.params)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	return results, nil
}

func (st *rdsDataStore) begin(resourceARN, database string) string {
	st.mu.Lock()
	defer st.mu.Unlock()

	raw := make([]byte, 48)
	_, _ = rand.Read(raw)
	id := base64.RawURLEncoding.EncodeToString(raw)
	key := rdsDataKey(resourceARN, database)
	now := st.now()
	st.txs[id] = &rdsDataTx{
		key:         key,
		resourceARN: resourceARN,
		snapshot:    st.dbs[key].clone(),
		started:     now,
		lastUsed:    now,
	}
	return id
}

// finish commits or rolls back a transaction. A commit whose replay fails
// leaves the live database untouched and the transaction gone.
func (st *rdsDataStore) finish(id, resourceARN string, commit bool) error {
	st.mu.Lock()
	defer st.mu.Unlock()

	tx, err := st.txLocked(id, resourceARN)
	if err != nil {
		return err
	}
	delete(st.txs, id)
	if !commit {
		return nil
	}
	work := st.dbs[tx.key].clone()
	if _, err := work.execAll(tx.statements); err != nil {
		return err
	}
	st.dbs[tx.key] = work
	return nil
}

func (s *Server) handleRDSDataRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isRDSDataRESTCandidate(r) {
		return false
	}
	if ok, status, code, msg, _ := s.validateSigV4WithService(r, "rds-data"); !ok {
		respondRDSDataError(w, status, code, msg)
		return true
	}
	op, ok := rdsdataOperationByPath[rawRequestPath(r)]
	if !ok || r.Method != http.MethodPost {
		respondRDSDataError(w, http.StatusNotFound, "UnknownOperationException", "unknown RDS Data API operation "+r.Method+" "+rawRequestPath(r))
		return true
	}
	switch op.Name {
	case "ExecuteStatement":
		s.handleRDSDataExecuteStatement(w, r)
	case "BatchExecuteStatement":
		s.handleRDSDataBatchExecuteStatement(w, r)
	case "BeginTransaction":
		s.handleRDSDataBeginTransaction(w, r)
	case "CommitTransaction", "RollbackTransaction":
		s.handleRDSDataFinishTransaction(w, r, op.Name == "CommitTransaction")
	case "ExecuteSql":
		s.handleRDSDataExecuteSQL(w, r)
	}
	return true
}

func isRDSDataRESTCandidate(r *http.Request) bool {
	if sigV4ServiceHint(r) == "rds-data" {
		return true
	}
	host := strings.ToLower(strings.TrimSpace(r.Host))
	return strings.HasPrefix(host, "rds-data.") || strings.Contains(host, ".rds-data.")
}

type rdsDataHTTPError struct {
	status int
	code   string
	msg    string
}

func (e *rdsDataHTTPError) Error() string { return e.msg }

func rdsDataBadRequest(msg string) error {
	return &rdsDataHTTPError{http.StatusBadRequest, "BadRequestException", msg}
}

// rdsDataTarget checks the shared resourceArn/secretArn/database inputs and
// returns the cluster ARN and the database the statements run against.
func (s *Server) rdsDataTarget(resourceARN, secretARN, database, schema string) (string, string, error) {
	resourceARN, secretARN = strings.TrimSpace(resourceARN), strings.TrimSpace(secretARN)
	switch {
	case resourceARN == "":
		return "", "", rdsDataBadRequest("resourceArn is required")
	case len(resourceARN) < 11 || len(resourceARN) > 570:
		return "", "", rdsDataBadRequest("resourceArn must be 11 to 570 characters")
	case secretARN == "":
		return "", "", rdsDataBadRequest("secretArn is required")
	case len(secretARN) < 11 || len(secretARN) > 570:
		return "", "", rdsDataBadRequest("secretArn must be 11 to 570 characters")
	case len(database) > 64 || len(schema) > 64:
		return "", "", rdsDataBadRequest("database and schema must be at most 64 characters")
	}
	cluster, err := s.rds.DescribeDBClusterByRef(resourceARN)
	if err != nil || cluster.ARN != resourceARN {
		return "", "", rdsDataBadRequest("DB cluster " + resourceARN + " not found")
	}
	if !cluster.HTTPEndpointEnabled {
		return "", "", &rdsDataHTTPError{http.StatusBadRequest, "HttpEndpointNotEnabledException", "HTTP endpoint is not enabled for " + resourceARN}
	}
	switch cluster.Status {
	case "stopped", "stopping", "starting":
		return "", "", &rdsDataHTTPError{http.StatusGatewayTimeout, "DatabaseUnavailableException", "DB cluster " + resourceARN + " is " + cluster.Status}
	}
	secret, err := s.secretsmgr.DescribeSecret(secretARN)
	if err != nil || secret.DeletedDate != nil {
		return "", "", &rdsDataHTTPError{http.StatusBadRequest, "SecretsErrorException", "secret " + secretARN + " can't be found or is scheduled for deletion"}
	}
	if database = strings.TrimSpace(database); database == "" {
		database = cluster.DatabaseName
	}
	return resourceARN, database, nil
}

func rdsDataSQL(sql string) error {
	switch {
	case strings.TrimSpace(sql) == "":
		return rdsDataBadRequest("sql is required")
	case len(sql) > 65536:
		return rdsDataBadRequest("sql must be at most 65536 characters")
	}
	return nil
}

func rdsDataTxID(id string) error {
	if len(id) > 192 {
		return rdsDataBadRequest("transactionId must be at most 192 characters")
	}
	return nil
}

func respondRDSDataJSON(w http.ResponseWriter, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(body)
}

func respondRDSDataError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Amzn-ErrorType", code)
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"message": msg})
}

// respondRDSDataErr maps engine, transaction and validation errors to the
// documented RDS Data API error codes.
func respondRDSDataErr(w http.ResponseWriter, err error, finishing bool) {
	var httpErr *rdsDataHTTPError
	var sqlErr *rdsDataSQLError
	switch {
	case errors.As(err, &httpErr):
		respondRDSDataError(w, httpErr.status, httpErr.code, httpErr.msg)
	case errors.As(err, &sqlErr) && sqlErr.Unsupported:
		respondRDSDataError(w, http.StatusBadRequest, "BadRequestException", sqlErr.Msg)
	case errors.As(err, &sqlErr):
		respondRDSDataError(w, http.StatusBadRequest, "DatabaseErrorException", sqlErr.Msg)
	case finishing && (errors.Is(err, errRDSDataTxNotFound) || errors.Is(err, errRDSDataTxResource)):
		respondRDSDataError(w, http.StatusNotFound, "NotFoundException", "transaction can't be found: "+err.Error())
	case errors.Is(err, errRDSDataTxNotFound), errors.Is(err, errRDSDataTxResource):
		respondRDSDataError(w, http.StatusNotFound, "TransactionNotFoundException", "transaction can't be found: "+err.Error())
	default:
		respondRDSDataError(w, http.StatusInternalServerError, "InternalServerErrorException", err.Error())
	}
}

func decodeRDSDataBody(w http.ResponseWriter, r *http.Request, out any) bool {
	body, err := readBodyBytes(r)
	if err == nil && len(strings.TrimSpace(string(body))) > 0 {
		err = json.Unmarshal(body, out)
	}
	if err != nil {
		respondRDSDataError(w, http.StatusBadRequest, "BadRequestException", "invalid JSON body")
		return false
	}
	return true
}
