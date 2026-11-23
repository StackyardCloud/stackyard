package server

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	tsmMaxDatabases       = 500
	tsmMaxTablesPerDB     = 1000
	tsmMaxRecordsPerTable = 200_000
	tsmMaxRecords         = 500_000 // across all tables
	tsmMaxTags            = 200
	tsmMaxBatchFiles      = 100
	tsmMaxBatchBytes      = 16 << 20
	tsmMaxBatchRows       = 100_000
	tsmMaxTasks           = 1000
	tsmMaxScheduled       = 1000
	tsmMaxCursors         = 200
	tsmMaxCursorRows      = 500_000
	tsmFutureSkew         = 15 * time.Minute
)

type tsmStore struct {
	mu         sync.Mutex
	dbs        map[string]*tsmDatabase
	tags       map[string]map[string]string // by resource ARN
	records    int                          // stored records across all tables
	tasks      map[string]*tsmTask
	taskIDs    []string          // creation order
	tokens     map[string]string // batch load ClientToken -> TaskId
	sched      map[string]*tsmScheduled
	schedIDs   []string
	sqToken    map[string]string // scheduled query ClientToken -> ARN
	cursors    map[string]*tsmCursor
	cursorQ    []string
	cursorRows int // rows held across open cursors
	settings   tsmSettings
	seq        int
}

type tsmDatabase struct {
	name, kms        string
	created, updated time.Time
	tables           map[string]*tsmTable
}

type tsmTable struct {
	db, name         string
	memHours         int64
	magDays          int64
	magWrites        map[string]any // MagneticStoreWriteProperties as given
	schema           map[string]any
	created, updated time.Time
	recs             []*tsmRecord
	index            map[string]int // record key -> position in recs
}

type tsmMeasure struct{ name, typ, value string }

type tsmRecord struct {
	t       int64 // nanoseconds since the epoch
	dims    map[string]string
	measure string
	typ     string // scalar type or MULTI
	value   string
	multi   []tsmMeasure
	version int64
}

func newTimestreamStore() *tsmStore {
	return &tsmStore{
		dbs: map[string]*tsmDatabase{}, tags: map[string]map[string]string{},
		tasks: map[string]*tsmTask{}, tokens: map[string]string{},
		sched: map[string]*tsmScheduled{}, sqToken: map[string]string{},
		cursors:  map[string]*tsmCursor{},
		settings: tsmSettings{pricing: "BYTES_SCANNED", maxTCU: 200, mode: "ON_DEMAND"},
	}
}

func tsmDatabaseARN(db string) string {
	return fmt.Sprintf("arn:aws:timestream:%s:%s:database/%s", tsmRegion, tsmAccount, db)
}

func tsmTableARN(db, table string) string { return tsmDatabaseARN(db) + "/table/" + table }

// dispatch runs op under the store lock and shapes the result to the model's
// output while still holding it.
func (st *tsmStore) dispatch(op string, call *tsmCall) (any, *tsmError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	h := tsmHandlers[op]
	if h == nil {
		return nil, &tsmError{code: "InternalServerException", msg: "operation " + op + " has no handler"}
	}
	out, err := h(st, call)
	if err != nil {
		return nil, err
	}
	model := tsmModelFor(op)
	if shape := model.Operations[op].Output; shape != "" {
		return model.Conform(shape, out), nil
	}
	return map[string]any{}, nil
}

var tsmHandlers map[string]func(*tsmStore, *tsmCall) (map[string]any, *tsmError)

func init() {
	tsmHandlers = map[string]func(*tsmStore, *tsmCall) (map[string]any, *tsmError){
		"DescribeEndpoints":   (*tsmStore).describeEndpoints,
		"CreateDatabase":      (*tsmStore).createDatabase,
		"DescribeDatabase":    (*tsmStore).describeDatabase,
		"UpdateDatabase":      (*tsmStore).updateDatabase,
		"DeleteDatabase":      (*tsmStore).deleteDatabase,
		"ListDatabases":       (*tsmStore).listDatabases,
		"CreateTable":         (*tsmStore).createTable,
		"DescribeTable":       (*tsmStore).describeTable,
		"UpdateTable":         (*tsmStore).updateTable,
		"DeleteTable":         (*tsmStore).deleteTable,
		"ListTables":          (*tsmStore).listTables,
		"TagResource":         (*tsmStore).tagResource,
		"UntagResource":       (*tsmStore).untagResource,
		"ListTagsForResource": (*tsmStore).listTags,
		"WriteRecords":        (*tsmStore).writeRecords,

		"CreateBatchLoadTask":   (*tsmStore).createBatchLoadTask,
		"DescribeBatchLoadTask": (*tsmStore).describeBatchLoadTask,
		"ListBatchLoadTasks":    (*tsmStore).listBatchLoadTasks,
		"ResumeBatchLoadTask":   (*tsmStore).resumeBatchLoadTask,

		"Query":                   (*tsmStore).query,
		"PrepareQuery":            (*tsmStore).prepareQuery,
		"CancelQuery":             (*tsmStore).cancelQuery,
		"CreateScheduledQuery":    (*tsmStore).createScheduledQuery,
		"DescribeScheduledQuery":  (*tsmStore).describeScheduledQuery,
		"UpdateScheduledQuery":    (*tsmStore).updateScheduledQuery,
		"DeleteScheduledQuery":    (*tsmStore).deleteScheduledQuery,
		"ListScheduledQueries":    (*tsmStore).listScheduledQueries,
		"ExecuteScheduledQuery":   (*tsmStore).executeScheduledQuery,
		"DescribeAccountSettings": (*tsmStore).describeAccountSettings,
		"UpdateAccountSettings":   (*tsmStore).updateAccountSettings,
	}
}

func (st *tsmStore) describeEndpoints(call *tsmCall) (map[string]any, *tsmError) {
	host := call.host
	if host == "" {
		host = "localhost"
	}
	return map[string]any{"Endpoints": []any{map[string]any{"Address": host, "CachePeriodInMinutes": float64(1440)}}}, nil
}

// --- databases ---

func (db *tsmDatabase) out() map[string]any {
	m := map[string]any{
		"Arn": tsmDatabaseARN(db.name), "DatabaseName": db.name, "TableCount": float64(len(db.tables)),
		"CreationTime": tsmEpoch(db.created), "LastUpdatedTime": tsmEpoch(db.updated),
	}
	if db.kms != "" {
		m["KmsKeyId"] = db.kms
	}
	return m
}

func tsmCheckName(field, name string) *tsmError {
	if len(name) < 3 || len(name) > 256 {
		return tsmValidation("%s must be between 3 and 256 characters", field)
	}
	return nil
}

func (st *tsmStore) database(name string) (*tsmDatabase, *tsmError) {
	if db := st.dbs[name]; db != nil {
		return db, nil
	}
	return nil, tsmNotFound("The database %s does not exist.", name)
}

func (st *tsmStore) createDatabase(call *tsmCall) (map[string]any, *tsmError) {
	name := tsmStr(call.in, "DatabaseName")
	if err := tsmCheckName("DatabaseName", name); err != nil {
		return nil, err
	}
	if st.dbs[name] != nil {
		return nil, tsmConflict("The database %s already exists.", name)
	}
	if len(st.dbs) >= tsmMaxDatabases {
		return nil, tsmQuota("You have reached the maximum number of databases (%d).", tsmMaxDatabases)
	}
	tags, terr := tsmTagsIn(tsmList(call.in, "Tags"))
	if terr != nil {
		return nil, terr
	}
	kms := tsmStr(call.in, "KmsKeyId")
	if kms == "" {
		kms = fmt.Sprintf("arn:aws:kms:%s:%s:key/alias/aws/timestream", tsmRegion, tsmAccount)
	}
	db := &tsmDatabase{name: name, kms: kms, created: call.now, updated: call.now, tables: map[string]*tsmTable{}}
	st.dbs[name] = db
	if len(tags) > 0 {
		st.tags[tsmDatabaseARN(name)] = tags
	}
	return map[string]any{"Database": db.out()}, nil
}

func (st *tsmStore) describeDatabase(call *tsmCall) (map[string]any, *tsmError) {
	db, err := st.database(tsmStr(call.in, "DatabaseName"))
	if err != nil {
		return nil, err
	}
	return map[string]any{"Database": db.out()}, nil
}

func (st *tsmStore) updateDatabase(call *tsmCall) (map[string]any, *tsmError) {
	db, err := st.database(tsmStr(call.in, "DatabaseName"))
	if err != nil {
		return nil, err
	}
	db.kms, db.updated = tsmStr(call.in, "KmsKeyId"), call.now
	return map[string]any{"Database": db.out()}, nil
}

func (st *tsmStore) deleteDatabase(call *tsmCall) (map[string]any, *tsmError) {
	db, err := st.database(tsmStr(call.in, "DatabaseName"))
	if err != nil {
		return nil, err
	}
	if len(db.tables) > 0 {
		return nil, tsmValidation("Cannot delete database %s: it still has %d tables. Delete the tables first.", db.name, len(db.tables))
	}
	delete(st.dbs, db.name)
	delete(st.tags, tsmDatabaseARN(db.name))
	return map[string]any{}, nil
}

func (st *tsmStore) listDatabases(call *tsmCall) (map[string]any, *tsmError) {
	names := make([]string, 0, len(st.dbs))
	for n := range st.dbs {
		names = append(names, n)
	}
	sort.Strings(names)
	page, next, err := tsmPage(names, tsmStr(call.in, "NextToken"), tsmInt(call.in, "MaxResults", 0), 20)
	if err != nil {
		return nil, err
	}
	items := make([]any, 0, len(page))
	for _, n := range page {
		items = append(items, st.dbs[n].out())
	}
	out := map[string]any{"Databases": items}
	if next != "" {
		out["NextToken"] = next
	}
	return out, nil
}

// --- tables ---

func (t *tsmTable) out() map[string]any {
	m := map[string]any{
		"Arn": tsmTableARN(t.db, t.name), "TableName": t.name, "DatabaseName": t.db, "TableStatus": "ACTIVE",
		"RetentionProperties": map[string]any{
			"MemoryStoreRetentionPeriodInHours": float64(t.memHours), "MagneticStoreRetentionPeriodInDays": float64(t.magDays),
		},
		"MagneticStoreWriteProperties": t.magWrites,
		"CreationTime":                 tsmEpoch(t.created), "LastUpdatedTime": tsmEpoch(t.updated),
	}
	if t.schema != nil {
		m["Schema"] = t.schema
	}
	return m
}

func (st *tsmStore) table(dbName, name string) (*tsmTable, *tsmError) {
	db, err := st.database(dbName)
	if err != nil {
		return nil, err
	}
	if t := db.tables[name]; t != nil {
		return t, nil
	}
	return nil, tsmNotFound("The table %s does not exist in database %s.", name, dbName)
}

// tsmApplyTableProps validates and applies retention, magnetic-store writes and
// the composite partition key. Absent members keep their current values.
func tsmApplyTableProps(t *tsmTable, call *tsmCall) *tsmError {
	if rp := tsmMap(call.in, "RetentionProperties"); rp != nil {
		t.memHours = tsmInt(rp, "MemoryStoreRetentionPeriodInHours", t.memHours)
		t.magDays = tsmInt(rp, "MagneticStoreRetentionPeriodInDays", t.magDays)
	}
	if mw := tsmMap(call.in, "MagneticStoreWriteProperties"); mw != nil {
		if s3 := tsmPath(mw, "MagneticStoreRejectedDataLocation", "S3Configuration"); s3 != nil {
			bucket := tsmStr(s3, "BucketName")
			if bucket == "" {
				return tsmValidation("MagneticStoreRejectedDataLocation.S3Configuration.BucketName is required")
			}
			if !call.buckets[bucket] {
				return tsmValidation("The S3 bucket %s for rejected magnetic store data does not exist.", bucket)
			}
			if tsmStr(s3, "EncryptionOption") == "SSE_KMS" && tsmStr(s3, "KmsKeyId") == "" {
				return tsmValidation("KmsKeyId is required when EncryptionOption is SSE_KMS")
			}
		}
		t.magWrites = mw
	}
	if sc := tsmMap(call.in, "Schema"); sc != nil {
		keys := tsmList(sc, "CompositePartitionKey")
		if len(keys) != 1 {
			return tsmValidation("CompositePartitionKey must contain exactly one partition key")
		}
		k, _ := keys[0].(map[string]any)
		switch tsmStr(k, "Type") {
		case "DIMENSION":
			if tsmStr(k, "Name") == "" {
				return tsmValidation("A DIMENSION partition key needs a Name")
			}
		case "MEASURE":
			if tsmStr(k, "Name") != "" || tsmStr(k, "EnforcementInRecord") != "" {
				return tsmValidation("A MEASURE partition key takes no Name or EnforcementInRecord")
			}
		}
		t.schema = sc
	}
	return nil
}

func (st *tsmStore) createTable(call *tsmCall) (map[string]any, *tsmError) {
	db, err := st.database(tsmStr(call.in, "DatabaseName"))
	if err != nil {
		return nil, err
	}
	name := tsmStr(call.in, "TableName")
	if err := tsmCheckName("TableName", name); err != nil {
		return nil, err
	}
	if db.tables[name] != nil {
		return nil, tsmConflict("The table %s already exists in database %s.", name, db.name)
	}
	if len(db.tables) >= tsmMaxTablesPerDB {
		return nil, tsmQuota("Database %s has reached the maximum number of tables (%d).", db.name, tsmMaxTablesPerDB)
	}
	tags, terr := tsmTagsIn(tsmList(call.in, "Tags"))
	if terr != nil {
		return nil, terr
	}
	t := &tsmTable{
		db: db.name, name: name, memHours: 6, magDays: 73000,
		magWrites: map[string]any{"EnableMagneticStoreWrites": false},
		schema:    map[string]any{"CompositePartitionKey": []any{map[string]any{"Type": "MEASURE"}}},
		created:   call.now, updated: call.now, index: map[string]int{},
	}
	if err := tsmApplyTableProps(t, call); err != nil {
		return nil, err
	}
	db.tables[name] = t
	db.updated = call.now
	if len(tags) > 0 {
		st.tags[tsmTableARN(db.name, name)] = tags
	}
	return map[string]any{"Table": t.out()}, nil
}

func (st *tsmStore) describeTable(call *tsmCall) (map[string]any, *tsmError) {
	t, err := st.table(tsmStr(call.in, "DatabaseName"), tsmStr(call.in, "TableName"))
	if err != nil {
		return nil, err
	}
	return map[string]any{"Table": t.out()}, nil
}

func (st *tsmStore) updateTable(call *tsmCall) (map[string]any, *tsmError) {
	t, err := st.table(tsmStr(call.in, "DatabaseName"), tsmStr(call.in, "TableName"))
	if err != nil {
		return nil, err
	}
	next := *t // validate on a copy so a failed update changes nothing
	if err := tsmApplyTableProps(&next, call); err != nil {
		return nil, err
	}
	t.memHours, t.magDays, t.magWrites, t.schema, t.updated = next.memHours, next.magDays, next.magWrites, next.schema, call.now
	return map[string]any{"Table": t.out()}, nil
}

func (st *tsmStore) deleteTable(call *tsmCall) (map[string]any, *tsmError) {
	t, err := st.table(tsmStr(call.in, "DatabaseName"), tsmStr(call.in, "TableName"))
	if err != nil {
		return nil, err
	}
	st.records -= len(t.recs)
	db := st.dbs[t.db]
	delete(db.tables, t.name)
	db.updated = call.now
	delete(st.tags, tsmTableARN(t.db, t.name))
	return map[string]any{}, nil
}

func (st *tsmStore) listTables(call *tsmCall) (map[string]any, *tsmError) {
	var tables []*tsmTable
	if dbName := tsmStr(call.in, "DatabaseName"); dbName != "" {
		db, err := st.database(dbName)
		if err != nil {
			return nil, err
		}
		for _, t := range db.tables {
			tables = append(tables, t)
		}
	} else {
		for _, db := range st.dbs {
			for _, t := range db.tables {
				tables = append(tables, t)
			}
		}
	}
	sort.Slice(tables, func(i, j int) bool {
		if tables[i].db != tables[j].db {
			return tables[i].db < tables[j].db
		}
		return tables[i].name < tables[j].name
	})
	page, next, err := tsmPage(tables, tsmStr(call.in, "NextToken"), tsmInt(call.in, "MaxResults", 0), 20)
	if err != nil {
		return nil, err
	}
	items := make([]any, 0, len(page))
	for _, t := range page {
		items = append(items, t.out())
	}
	out := map[string]any{"Tables": items}
	if next != "" {
		out["NextToken"] = next
	}
	return out, nil
}

// --- tags ---

func tsmTagsIn(list []any) (map[string]string, *tsmError) {
	if len(list) > tsmMaxTags {
		return nil, tsmQuota("A resource can have at most %d tags", tsmMaxTags)
	}
	out := map[string]string{}
	for _, item := range list {
		tag, _ := item.(map[string]any)
		key := tsmStr(tag, "Key")
		if strings.HasPrefix(strings.ToLower(key), "aws:") {
			return nil, tsmValidation("Tag keys can't start with aws:")
		}
		out[key] = tsmStr(tag, "Value")
	}
	return out, nil
}

// resourceExists reports whether arn names a database, table or scheduled query.
func (st *tsmStore) resourceExists(arn string) bool {
	if st.sched[arn] != nil {
		return true
	}
	prefix := tsmDatabaseARN("")
	if !strings.HasPrefix(arn, prefix) {
		return false
	}
	parts := strings.Split(strings.TrimPrefix(arn, prefix), "/")
	db := st.dbs[parts[0]]
	switch {
	case db == nil:
		return false
	case len(parts) == 1:
		return true
	case len(parts) == 3 && parts[1] == "table":
		return db.tables[parts[2]] != nil
	}
	return false
}

func (st *tsmStore) tagResource(call *tsmCall) (map[string]any, *tsmError) {
	arn := tsmStr(call.in, "ResourceARN")
	if !st.resourceExists(arn) {
		return nil, tsmNotFound("The resource %s does not exist.", arn)
	}
	tags, err := tsmTagsIn(tsmList(call.in, "Tags"))
	if err != nil {
		return nil, err
	}
	merged := map[string]string{}
	for k, v := range st.tags[arn] {
		merged[k] = v
	}
	for k, v := range tags {
		merged[k] = v
	}
	if len(merged) > tsmMaxTags {
		return nil, tsmQuota("A resource can have at most %d tags", tsmMaxTags)
	}
	st.tags[arn] = merged
	return map[string]any{}, nil
}

func (st *tsmStore) untagResource(call *tsmCall) (map[string]any, *tsmError) {
	arn := tsmStr(call.in, "ResourceARN")
	if !st.resourceExists(arn) {
		return nil, tsmNotFound("The resource %s does not exist.", arn)
	}
	for _, k := range tsmList(call.in, "TagKeys") {
		key, _ := k.(string)
		delete(st.tags[arn], key)
	}
	return map[string]any{}, nil
}

func (st *tsmStore) listTags(call *tsmCall) (map[string]any, *tsmError) {
	arn := tsmStr(call.in, "ResourceARN")
	if !st.resourceExists(arn) {
		return nil, tsmNotFound("The resource %s does not exist.", arn)
	}
	keys := make([]string, 0, len(st.tags[arn]))
	for k := range st.tags[arn] {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	page, next, err := tsmPage(keys, tsmStr(call.in, "NextToken"), tsmInt(call.in, "MaxResults", 0), tsmMaxTags)
	if err != nil {
		return nil, err
	}
	tags := make([]any, 0, len(page))
	for _, k := range page {
		tags = append(tags, map[string]any{"Key": k, "Value": st.tags[arn][k]})
	}
	out := map[string]any{"Tags": tags}
	if next != "" {
		out["NextToken"] = next
	}
	return out, nil
}
