package server

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
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
	mu      sync.Mutex
	dbs     map[string]rdsDataDB
	txs     map[string]*rdsDataTx
	version int64
	now     func() time.Time
}

// rdsDataTx copies each table the first time it touches it and records that
// table's version. Commit fails if any touched table changed since then, and
// otherwise swaps the transaction's copies in, so the client keeps exactly the
// ids and row counts it saw.
type rdsDataTx struct {
	key         string
	resourceARN string
	tables      map[string]*rdsDataTable // private copies; nil means absent
	versions    map[string]int64         // live version when first touched
	dirty       map[string]bool
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

func (t *rdsDataTable) version() int64 {
	if t == nil {
		return 0
	}
	return t.Version
}

// sweepLocked drops transactions idle for three minutes or open for 24 hours.
func (st *rdsDataStore) sweepLocked() {
	now := st.now()
	for id, tx := range st.txs {
		if now.Sub(tx.lastUsed) > rdsDataTxIdleTimeout || now.Sub(tx.started) > rdsDataTxMaxLifetime {
			delete(st.txs, id)
		}
	}
}

func (st *rdsDataStore) txLocked(id, resourceARN string) (*rdsDataTx, error) {
	tx, ok := st.txs[id]
	if !ok {
		return nil, errRDSDataTxNotFound
	}
	if tx.resourceARN != resourceARN {
		return nil, errRDSDataTxResource
	}
	tx.lastUsed = st.now()
	return tx, nil
}

func (st *rdsDataStore) liveLocked(key string) rdsDataDB {
	db := st.dbs[key]
	if db == nil {
		db = rdsDataDB{}
		st.dbs[key] = db
	}
	return db
}

// run executes statements atomically. Only the tables a statement touches are
// loaded; writes work on copies that replace the originals when every
// statement succeeds, and reads use the tables directly.
func (st *rdsDataStore) run(resourceARN, database, txID string, stmts []rdsDataStatement) ([]rdsDataResult, error) {
	prepared := make([]rdsDataStmt, 0, len(stmts))
	written := map[string]bool{}
	for _, stmt := range stmts {
		p, err := rdsDataPrepare(stmt.sql, stmt.params)
		if err != nil {
			return nil, err
		}
		prepared = append(prepared, p)
		if p.write {
			written[p.table] = true
		}
	}

	st.mu.Lock()
	defer st.mu.Unlock()
	st.sweepLocked()

	key := rdsDataKey(resourceARN, database)
	var tx *rdsDataTx
	if txID != "" {
		var err error
		if tx, err = st.txLocked(txID, resourceARN); err != nil {
			return nil, err
		}
		key = tx.key
	}
	source := func(name string) *rdsDataTable {
		if tx == nil {
			return st.dbs[key][name]
		}
		if _, seen := tx.versions[name]; !seen {
			live := st.dbs[key][name]
			tx.versions[name] = live.version()
			tx.tables[name] = live.clone()
		}
		return tx.tables[name]
	}

	work := rdsDataDB{}
	loaded := map[string]bool{}
	for _, p := range prepared {
		if p.table == "" || loaded[p.table] {
			continue
		}
		loaded[p.table] = true
		t := source(p.table)
		if written[p.table] {
			t = t.clone()
		}
		if t != nil {
			work[p.table] = t
		}
	}
	results, err := work.execAll(prepared)
	if err != nil {
		return nil, err
	}
	for name := range written {
		t := work[name]
		if tx != nil {
			tx.tables[name] = t
			tx.dirty[name] = true
			continue
		}
		live := st.liveLocked(key)
		if t == nil {
			delete(live, name)
			continue
		}
		st.version++
		t.Version = st.version
		live[name] = t
	}
	return results, nil
}

func (db rdsDataDB) execAll(stmts []rdsDataStmt) ([]rdsDataResult, error) {
	results := make([]rdsDataResult, 0, len(stmts))
	for _, stmt := range stmts {
		result, err := db.exec(stmt)
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
	st.sweepLocked()

	raw := make([]byte, 48)
	_, _ = rand.Read(raw)
	id := base64.RawURLEncoding.EncodeToString(raw)
	now := st.now()
	st.txs[id] = &rdsDataTx{
		key:         rdsDataKey(resourceARN, database),
		resourceARN: resourceARN,
		tables:      map[string]*rdsDataTable{},
		versions:    map[string]int64{},
		dirty:       map[string]bool{},
		started:     now,
		lastUsed:    now,
	}
	return id
}

// finish commits or rolls back a transaction; either way it is gone afterwards.
// A commit fails with a serialization error when another write changed a table
// the transaction touched.
func (st *rdsDataStore) finish(id, resourceARN string, commit bool) error {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.sweepLocked()

	tx, err := st.txLocked(id, resourceARN)
	if err != nil {
		return err
	}
	delete(st.txs, id)
	if !commit {
		return nil
	}
	live := st.liveLocked(tx.key)
	for name, v := range tx.versions {
		if live[name].version() != v {
			return rdsDataDBErr("could not serialize access: table %q changed after the transaction read it; the transaction was rolled back", name)
		}
	}
	for name := range tx.dirty {
		t := tx.tables[name]
		if t == nil {
			delete(live, name)
			continue
		}
		st.version++
		t.Version = st.version
		live[name] = t
	}
	return nil
}

// dropCluster removes a deleted cluster's databases and open transactions, so a
// cluster recreated under the same name starts empty.
func (st *rdsDataStore) dropCluster(resourceARN string) {
	st.mu.Lock()
	defer st.mu.Unlock()

	prefix := resourceARN + "\x00"
	for key := range st.dbs {
		if strings.HasPrefix(key, prefix) {
			delete(st.dbs, key)
		}
	}
	for id, tx := range st.txs {
		if tx.resourceARN == resourceARN {
			delete(st.txs, id)
		}
	}
}
