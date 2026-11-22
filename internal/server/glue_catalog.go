package server

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Data Catalog behaviour beyond plain CRUD: Athena mirroring, table versions,
// partition batches and expressions, partition indexes, column statistics,
// connection and classifier specifics.

const (
	glueAthenaCatalog       = "AwsDataCatalog"
	glueMaxPartitionIndexes = 3
	glueMaxColumnStatistics = 1000
)

func init() {
	glueKinds["tableversion"] = glueKind{fields: []string{"DatabaseName", "TableName", "VersionId"}, parent: "table", parentFields: []string{"DatabaseName", "TableName"}}
	registerGlueHandlers(map[string]glueHandler{
		"CreateDatabase":          glueCreateDatabase,
		"DeleteDatabase":          glueDeleteDatabase,
		"CreateTable":             glueCreateTable,
		"UpdateTable":             glueUpdateTable,
		"DeleteTable":             glueDeleteTable,
		"GetTables":               glueGetTables,
		"BatchDeleteTable":        glueBatchDeleteTable,
		"SearchTables":            glueSearchTables,
		"GetTableVersion":         glueGetTableVersion,
		"GetTableVersions":        glueGetTableVersions,
		"DeleteTableVersion":      glueDeleteTableVersion,
		"BatchDeleteTableVersion": glueBatchDeleteTableVersion,
		"GetPartitions":           glueGetPartitions,
		"CreatePartition": func(st *glueStore, in glueRecord) (glueRecord, *glueError) {
			input, _ := in["PartitionInput"].(map[string]any)
			return glueRecord{}, st.createPartition(in["DatabaseName"], in["TableName"], input)
		},
		"BatchCreatePartition":               glueBatchCreatePartition,
		"BatchDeletePartition":               glueBatchDeletePartition,
		"BatchGetPartition":                  glueBatchGetPartition,
		"BatchUpdatePartition":               glueBatchUpdatePartition,
		"CreatePartitionIndex":               glueCreatePartitionIndex,
		"GetPartitionIndexes":                glueGetPartitionIndexes,
		"DeletePartitionIndex":               glueDeletePartitionIndex,
		"UpdateColumnStatisticsForTable":     glueUpdateColumnStats(false),
		"UpdateColumnStatisticsForPartition": glueUpdateColumnStats(true),
		"GetColumnStatisticsForTable":        glueGetColumnStats(false),
		"GetColumnStatisticsForPartition":    glueGetColumnStats(true),
		"DeleteColumnStatisticsForTable":     glueDeleteColumnStats(false),
		"DeleteColumnStatisticsForPartition": glueDeleteColumnStats(true),
		"GetUserDefinedFunctions":            glueGetUserDefinedFunctions,
		"CreateConnection":                   glueCreateConnection,
		"GetConnection":                      glueGetConnection,
		"BatchDeleteConnection":              glueBatchDeleteConnection,
		"TestConnection":                     glueTestConnection,
		"CreateClassifier":                   glueCreateClassifier,
		"UpdateClassifier":                   glueUpdateClassifier,
		"GetUnfilteredTableMetadata":         glueGetUnfilteredTable,
		"GetUnfilteredPartitionMetadata":     glueGetUnfilteredPartition,
		"GetUnfilteredPartitionsMetadata":    glueGetUnfilteredPartitions,
		"PutDataCatalogEncryptionSettings":   glueSinglePut("encryption", "DataCatalogEncryptionSettings"),
		"GetDataCatalogEncryptionSettings":   glueSingleGet("encryption", "DataCatalogEncryptionSettings"),
		"PutDataCatalogExportConfiguration":  glueSinglePut("export", ""),
		"GetDataCatalogExportConfiguration":  glueSingleGet("export", ""),
		"ImportCatalogToGlue": func(st *glueStore, in glueRecord) (glueRecord, *glueError) {
			st.singles["import"] = glueRecord{"ImportCompleted": true, "ImportTime": glueEpoch(time.Now()), "ImportedBy": "stackyard"}
			return glueRecord{}, nil
		},
		"GetCatalogImportStatus": func(st *glueStore, in glueRecord) (glueRecord, *glueError) {
			return glueRecord{"ImportStatus": glueFirstRecord(st.singles["import"], glueRecord{"ImportCompleted": false})}, nil
		},
	})
}

func glueCreateDatabase(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	rec, err := st.createFrom(glueOps["CreateDatabase"], in)
	if err != nil {
		return nil, err
	}
	rec["CatalogId"] = glueAccount
	if st.athena != nil {
		_ = st.athena.CreateDatabase(glueAthenaCatalog, glueStr(rec, "Name"), glueStr(rec, "Description"), glueTags(rec["Parameters"]))
	}
	return glueRecord{}, nil
}

func glueDeleteDatabase(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	rec, err := st.find("database", []any{in["Name"]})
	if err != nil {
		return nil, err
	}
	if st.athena != nil {
		for _, t := range st.childTables(glueStr(rec, "Name")) {
			_ = st.athena.DeleteTable(glueAthenaCatalog, glueStr(rec, "Name"), glueStr(t, "Name"))
		}
		_ = st.athena.DeleteDatabase(glueAthenaCatalog, glueStr(rec, "Name"))
	}
	st.remove("database", rec)
	return glueRecord{}, nil
}

func (st *glueStore) childTables(db string) []glueRecord {
	var out []glueRecord
	for _, t := range st.all("table") {
		if glueStr(t, "DatabaseName") == db {
			out = append(out, t)
		}
	}
	return out
}

func glueCreateTable(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	input, _ := in["TableInput"].(map[string]any)
	name := glueFirst(glueStr(in, "Name"), glueStr(input, "Name"))
	if name == "" {
		return nil, glueInvalid("TableInput.Name is required")
	}
	if _, ok := in["OpenTableFormatInput"]; ok && input == nil {
		input = map[string]any{"TableType": "EXTERNAL_TABLE", "Parameters": map[string]any{"table_type": "ICEBERG"}}
	}
	rec := glueRecord{"DatabaseName": in["DatabaseName"], "Name": name, "CatalogId": glueAccount, "VersionId": "1", "IsRegisteredWithLakeFormation": false}
	for k, v := range input {
		if k != "Name" {
			rec[k] = v
		}
	}
	var indexes []any
	for _, raw := range glueList(in["PartitionIndexes"]) {
		idx, _ := raw.(map[string]any)
		desc, err := gluePartitionIndexDescriptor(rec, idx)
		if err != nil {
			return nil, err
		}
		indexes = append(indexes, desc)
	}
	if len(indexes) > glueMaxPartitionIndexes {
		return nil, glueErr("ResourceNumberLimitExceededException", "a table can have at most %d partition indexes", glueMaxPartitionIndexes)
	}
	rec["_indexes"] = indexes
	created, err := st.create("table", rec, nil)
	if err != nil {
		return nil, err
	}
	st.archiveTable(created)
	st.mirrorTable(created, false)
	return glueRecord{}, nil
}

// archiveTable stores the table's current definition as a TableVersion.
func (st *glueStore) archiveTable(rec glueRecord) {
	snapshot := glueRecord{}
	for k, v := range rec {
		if !strings.HasPrefix(k, "_") {
			snapshot[k] = v
		}
	}
	_, _ = st.create("tableversion", glueRecord{
		"DatabaseName": rec["DatabaseName"], "TableName": rec["Name"], "VersionId": rec["VersionId"], "Table": snapshot,
	}, nil)
}

func (st *glueStore) mirrorTable(rec glueRecord, update bool) {
	if st.athena == nil {
		return
	}
	var cols []string
	if sd, ok := rec["StorageDescriptor"].(map[string]any); ok {
		for _, c := range glueList(sd["Columns"]) {
			if m, ok := c.(map[string]any); ok {
				cols = append(cols, glueStr(m, "Name"))
			}
		}
	}
	db, name, desc, params := glueStr(rec, "DatabaseName"), glueStr(rec, "Name"), glueStr(rec, "Description"), glueTags(rec["Parameters"])
	if update {
		_ = st.athena.UpdateTable(glueAthenaCatalog, db, name, desc, cols, params)
		return
	}
	_ = st.athena.CreateTable(glueAthenaCatalog, db, name, desc, cols, params)
}

func glueUpdateTable(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	input, _ := in["TableInput"].(map[string]any)
	name := glueFirst(glueStr(in, "Name"), glueStr(input, "Name"))
	rec, err := st.find("table", []any{in["DatabaseName"], name})
	if err != nil {
		return nil, err
	}
	if want := glueStr(in, "VersionId"); want != "" && want != glueStr(rec, "VersionId") {
		return nil, glueErr("ConcurrentModificationException", "table %s is at version %s, not %s", name, glueStr(rec, "VersionId"), want)
	}
	for k, v := range input {
		if k != "Name" {
			rec[k] = v
		}
	}
	n, _ := strconv.Atoi(glueStr(rec, "VersionId"))
	rec["VersionId"] = strconv.Itoa(n + 1)
	glueTouch(rec)
	if skip, _ := in["SkipArchive"].(bool); !skip {
		st.archiveTable(rec)
	}
	st.mirrorTable(rec, true)
	return glueRecord{}, nil
}

func glueDeleteTable(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	rec, err := st.find("table", []any{in["DatabaseName"], in["Name"]})
	if err != nil {
		return nil, err
	}
	st.remove("table", rec)
	if st.athena != nil {
		_ = st.athena.DeleteTable(glueAthenaCatalog, glueStr(rec, "DatabaseName"), glueStr(rec, "Name"))
	}
	return glueRecord{}, nil
}

func glueGetTables(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	if _, err := st.find("database", []any{in["DatabaseName"]}); err != nil {
		return nil, err
	}
	match, err := glueNameMatcher(glueStr(in, "Expression"))
	if err != nil {
		return nil, err
	}
	var tables []glueRecord
	for _, t := range st.childTables(glueStr(in, "DatabaseName")) {
		if match(glueStr(t, "Name")) {
			tables = append(tables, t)
		}
	}
	return gluePageRecords(tables, in, "TableList")
}

// glueNameMatcher implements Glue's name Expression / Pattern: a regex where
// "*" means any run of characters, matched against the whole name.
func glueNameMatcher(expr string) (func(string) bool, *glueError) {
	if strings.TrimSpace(expr) == "" {
		return func(string) bool { return true }, nil
	}
	pattern := strings.ReplaceAll(expr, "*", ".*")
	pattern = strings.ReplaceAll(pattern, "..*", ".*")
	re, err := regexp.Compile("(?i)^(" + pattern + ")$")
	if err != nil {
		return nil, glueInvalid("invalid name expression %q", expr)
	}
	return re.MatchString, nil
}

func gluePageRecords(recs []glueRecord, in glueRecord, member string) (glueRecord, *glueError) {
	keys := make([]string, len(recs))
	for i := range recs {
		keys[i] = fmt.Sprintf("%08d", i)
	}
	page, next, err := gluePage(keys, in)
	if err != nil {
		return nil, err
	}
	items := make([]any, 0, len(page))
	for _, k := range page {
		i, _ := strconv.Atoi(k)
		items = append(items, recs[i])
	}
	out := glueRecord{member: items}
	if next != "" {
		out["NextToken"] = next
	}
	return out, nil
}

func glueBatchDeleteTable(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	var errs []any
	for _, name := range glueList(in["TablesToDelete"]) {
		if _, err := glueDeleteTable(st, glueRecord{"DatabaseName": in["DatabaseName"], "Name": name}); err != nil {
			errs = append(errs, glueRecord{"TableName": name, "ErrorDetail": glueErrorDetail(err)})
		}
	}
	return glueRecord{"Errors": glueNonNil(errs)}, nil
}

func glueSearchTables(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	text := strings.ToLower(glueStr(in, "SearchText"))
	var out []glueRecord
	for _, t := range st.all("table") {
		if text == "" || strings.Contains(strings.ToLower(glueStr(t, "Name")), text) || strings.Contains(strings.ToLower(glueStr(t, "Description")), text) {
			out = append(out, t)
		}
	}
	return gluePageRecords(out, in, "TableList")
}

func glueGetTableVersion(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	table, err := st.find("table", []any{in["DatabaseName"], in["TableName"]})
	if err != nil {
		return nil, err
	}
	version := glueFirst(glueStr(in, "VersionId"), glueStr(table, "VersionId"))
	rec, err := st.find("tableversion", []any{in["DatabaseName"], in["TableName"], version})
	if err != nil {
		return nil, err
	}
	return glueRecord{"TableVersion": rec}, nil
}

func glueGetTableVersions(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	if _, err := st.find("table", []any{in["DatabaseName"], in["TableName"]}); err != nil {
		return nil, err
	}
	var out []glueRecord
	for _, v := range st.all("tableversion") {
		if v["DatabaseName"] == in["DatabaseName"] && v["TableName"] == in["TableName"] {
			out = append(out, v)
		}
	}
	return gluePageRecords(out, in, "TableVersions")
}

func glueDeleteTableVersion(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	table, err := st.find("table", []any{in["DatabaseName"], in["TableName"]})
	if err != nil {
		return nil, err
	}
	if glueStr(in, "VersionId") == glueStr(table, "VersionId") {
		return nil, glueInvalid("version %s is the table's current version", glueStr(in, "VersionId"))
	}
	rec, err := st.find("tableversion", []any{in["DatabaseName"], in["TableName"], in["VersionId"]})
	if err != nil {
		return nil, err
	}
	st.remove("tableversion", rec)
	return glueRecord{}, nil
}

func glueBatchDeleteTableVersion(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	var errs []any
	for _, v := range glueList(in["VersionIds"]) {
		if _, err := glueDeleteTableVersion(st, glueRecord{"DatabaseName": in["DatabaseName"], "TableName": in["TableName"], "VersionId": v}); err != nil {
			errs = append(errs, glueRecord{"TableName": in["TableName"], "VersionId": v, "ErrorDetail": glueErrorDetail(err)})
		}
	}
	return glueRecord{"Errors": glueNonNil(errs)}, nil
}

// Partitions.

func (st *glueStore) tablePartitions(db, table any) []glueRecord {
	var out []glueRecord
	for _, p := range st.all("partition") {
		if p["DatabaseName"] == db && p["TableName"] == table {
			out = append(out, p)
		}
	}
	return out
}

func glueGetPartitions(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	table, err := st.find("table", []any{in["DatabaseName"], in["TableName"]})
	if err != nil {
		return nil, err
	}
	match, gerr := gluePartitionExpression(table, glueStr(in, "Expression"))
	if gerr != nil {
		return nil, gerr
	}
	var out []glueRecord
	for _, p := range st.tablePartitions(in["DatabaseName"], in["TableName"]) {
		if match(glueList(p["Values"])) {
			out = append(out, p)
		}
	}
	return gluePageRecords(out, in, "Partitions")
}

var gluePartitionTerm = regexp.MustCompile(`^\s*([A-Za-z_][A-Za-z0-9_]*)\s*(=|<>|!=|>=|<=|>|<)\s*('(?:[^']*)'|-?[0-9.]+)\s*$`)

// gluePartitionExpression supports conjunctions of comparisons between a
// partition key and a literal, e.g. year = '2024' AND month >= 6.
func gluePartitionExpression(table glueRecord, expr string) (func([]any) bool, *glueError) {
	if strings.TrimSpace(expr) == "" {
		return func([]any) bool { return true }, nil
	}
	keys := map[string]int{}
	for i, k := range glueList(table["PartitionKeys"]) {
		if m, ok := k.(map[string]any); ok {
			keys[strings.ToLower(glueStr(m, "Name"))] = i
		}
	}
	type term struct {
		idx     int
		op, lit string
	}
	var terms []term
	for _, part := range regexp.MustCompile(`(?i)\s+AND\s+`).Split(expr, -1) {
		m := gluePartitionTerm.FindStringSubmatch(part)
		if m == nil {
			return nil, glueInvalid("unsupported partition expression %q", part)
		}
		idx, ok := keys[strings.ToLower(m[1])]
		if !ok {
			return nil, glueInvalid("%s is not a partition key", m[1])
		}
		terms = append(terms, term{idx, m[2], strings.Trim(m[3], "'")})
	}
	return func(values []any) bool {
		for _, t := range terms {
			if t.idx >= len(values) {
				return false
			}
			v := fmt.Sprint(values[t.idx])
			if !glueCompare(v, t.op, t.lit) {
				return false
			}
		}
		return true
	}, nil
}

func glueCompare(a, op, b string) bool {
	cmp := strings.Compare(a, b)
	if fa, err1 := strconv.ParseFloat(a, 64); err1 == nil {
		if fb, err2 := strconv.ParseFloat(b, 64); err2 == nil {
			cmp = 0
			if fa < fb {
				cmp = -1
			} else if fa > fb {
				cmp = 1
			}
		}
	}
	switch op {
	case "=":
		return cmp == 0
	case "<>", "!=":
		return cmp != 0
	case ">":
		return cmp > 0
	case "<":
		return cmp < 0
	case ">=":
		return cmp >= 0
	case "<=":
		return cmp <= 0
	}
	return false
}

func (st *glueStore) createPartition(db, table any, input map[string]any) *glueError {
	rec := glueRecord{"DatabaseName": db, "TableName": table, "CatalogId": glueAccount}
	for k, v := range input {
		rec[k] = v
	}
	tbl, err := st.find("table", []any{db, table})
	if err != nil {
		return err
	}
	if n := len(glueList(tbl["PartitionKeys"])); n > 0 && len(glueList(rec["Values"])) != n {
		return glueInvalid("the table has %d partition keys but %d values were given", n, len(glueList(rec["Values"])))
	}
	_, err = st.create("partition", rec, nil)
	return err
}

func glueBatchCreatePartition(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	if _, err := st.find("table", []any{in["DatabaseName"], in["TableName"]}); err != nil {
		return nil, err
	}
	var errs []any
	for _, raw := range glueList(in["PartitionInputList"]) {
		input, _ := raw.(map[string]any)
		if err := st.createPartition(in["DatabaseName"], in["TableName"], input); err != nil {
			errs = append(errs, glueRecord{"PartitionValues": input["Values"], "ErrorDetail": glueErrorDetail(err)})
		}
	}
	return glueRecord{"Errors": glueNonNil(errs)}, nil
}

func glueBatchDeletePartition(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	if _, err := st.find("table", []any{in["DatabaseName"], in["TableName"]}); err != nil {
		return nil, err
	}
	var errs []any
	for _, raw := range glueList(in["PartitionsToDelete"]) {
		values := glueAt(glueRecord{"v": raw}, "v.Values")
		rec, err := st.find("partition", []any{in["DatabaseName"], in["TableName"], values})
		if err != nil {
			errs = append(errs, glueRecord{"PartitionValues": values, "ErrorDetail": glueErrorDetail(err)})
			continue
		}
		st.remove("partition", rec)
	}
	return glueRecord{"Errors": glueNonNil(errs)}, nil
}

func glueBatchGetPartition(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	if _, err := st.find("table", []any{in["DatabaseName"], in["TableName"]}); err != nil {
		return nil, err
	}
	var found []any
	for _, raw := range glueList(in["PartitionsToGet"]) {
		values := glueAt(glueRecord{"v": raw}, "v.Values")
		if rec, err := st.find("partition", []any{in["DatabaseName"], in["TableName"], values}); err == nil {
			found = append(found, rec)
		}
	}
	return glueRecord{"Partitions": glueNonNil(found), "UnprocessedKeys": []any{}}, nil
}

func glueBatchUpdatePartition(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	if _, err := st.find("table", []any{in["DatabaseName"], in["TableName"]}); err != nil {
		return nil, err
	}
	var errs []any
	for _, raw := range glueList(in["Entries"]) {
		entry, _ := raw.(map[string]any)
		rec, err := st.find("partition", []any{in["DatabaseName"], in["TableName"], entry["PartitionValueList"]})
		if err != nil {
			errs = append(errs, glueRecord{"PartitionValueList": entry["PartitionValueList"], "ErrorDetail": glueErrorDetail(err)})
			continue
		}
		if input, ok := entry["PartitionInput"].(map[string]any); ok {
			for k, v := range input {
				if k != "Values" {
					rec[k] = v
				}
			}
			glueTouch(rec)
		}
	}
	return glueRecord{"Errors": glueNonNil(errs)}, nil
}

// Partition indexes live on the table record.

func gluePartitionIndexDescriptor(table glueRecord, idx map[string]any) (glueRecord, *glueError) {
	types := map[string]string{}
	for _, k := range glueList(table["PartitionKeys"]) {
		if m, ok := k.(map[string]any); ok {
			types[glueStr(m, "Name")] = glueStr(m, "Type")
		}
	}
	var keys []any
	for _, k := range glueList(idx["Keys"]) {
		name, _ := k.(string)
		typ, ok := types[name]
		if !ok {
			return nil, glueInvalid("index key %s is not a partition key of the table", name)
		}
		keys = append(keys, glueRecord{"Name": name, "Type": typ})
	}
	return glueRecord{"IndexName": idx["IndexName"], "Keys": keys, "IndexStatus": "ACTIVE"}, nil
}

func glueCreatePartitionIndex(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	table, err := st.find("table", []any{in["DatabaseName"], in["TableName"]})
	if err != nil {
		return nil, err
	}
	idx, _ := in["PartitionIndex"].(map[string]any)
	existing := glueList(table["_indexes"])
	for _, e := range existing {
		if glueStr(e.(glueRecord), "IndexName") == glueStr(idx, "IndexName") {
			return nil, glueErr("AlreadyExistsException", "partition index %s already exists", glueStr(idx, "IndexName"))
		}
	}
	if len(existing) >= glueMaxPartitionIndexes {
		return nil, glueErr("ResourceNumberLimitExceededException", "a table can have at most %d partition indexes", glueMaxPartitionIndexes)
	}
	desc, err := gluePartitionIndexDescriptor(table, idx)
	if err != nil {
		return nil, err
	}
	table["_indexes"] = append(existing, desc)
	return glueRecord{}, nil
}

func glueGetPartitionIndexes(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	table, err := st.find("table", []any{in["DatabaseName"], in["TableName"]})
	if err != nil {
		return nil, err
	}
	return glueRecord{"PartitionIndexDescriptorList": glueNonNil(glueList(table["_indexes"]))}, nil
}

func glueDeletePartitionIndex(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	table, err := st.find("table", []any{in["DatabaseName"], in["TableName"]})
	if err != nil {
		return nil, err
	}
	var kept []any
	found := false
	for _, e := range glueList(table["_indexes"]) {
		if glueStr(e.(glueRecord), "IndexName") == glueStr(in, "IndexName") {
			found = true
			continue
		}
		kept = append(kept, e)
	}
	if !found {
		return nil, glueNotFound("partition index %s not found", glueStr(in, "IndexName"))
	}
	table["_indexes"] = kept
	return glueRecord{}, nil
}

// Column statistics live on the table or partition record under "_colstats".

func (st *glueStore) statsHolder(in glueRecord, partition bool) (glueRecord, *glueError) {
	if partition {
		return st.find("partition", []any{in["DatabaseName"], in["TableName"], in["PartitionValues"]})
	}
	return st.find("table", []any{in["DatabaseName"], in["TableName"]})
}

func glueUpdateColumnStats(partition bool) glueHandler {
	return func(st *glueStore, in glueRecord) (glueRecord, *glueError) {
		holder, err := st.statsHolder(in, partition)
		if err != nil {
			return nil, err
		}
		stats, _ := holder["_colstats"].(map[string]any)
		if stats == nil {
			stats = map[string]any{}
		}
		for _, raw := range glueList(in["ColumnStatisticsList"]) {
			m, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			if _, exists := stats[glueStr(m, "ColumnName")]; !exists && len(stats) >= glueMaxColumnStatistics {
				return nil, glueErr("ResourceNumberLimitExceededException", "at most %d columns can have statistics", glueMaxColumnStatistics)
			}
			stats[glueStr(m, "ColumnName")] = m
		}
		holder["_colstats"] = stats
		return glueRecord{"Errors": []any{}}, nil
	}
}

func glueGetColumnStats(partition bool) glueHandler {
	return func(st *glueStore, in glueRecord) (glueRecord, *glueError) {
		holder, err := st.statsHolder(in, partition)
		if err != nil {
			return nil, err
		}
		stats, _ := holder["_colstats"].(map[string]any)
		var found, errs []any
		for _, c := range glueList(in["ColumnNames"]) {
			name, _ := c.(string)
			if s, ok := stats[name]; ok {
				found = append(found, s)
				continue
			}
			errs = append(errs, glueRecord{"ColumnName": name, "Error": glueErrorDetail(glueNotFound("no statistics for column %s", name))})
		}
		return glueRecord{"ColumnStatisticsList": glueNonNil(found), "Errors": glueNonNil(errs)}, nil
	}
}

func glueDeleteColumnStats(partition bool) glueHandler {
	return func(st *glueStore, in glueRecord) (glueRecord, *glueError) {
		holder, err := st.statsHolder(in, partition)
		if err != nil {
			return nil, err
		}
		stats, _ := holder["_colstats"].(map[string]any)
		name := glueStr(in, "ColumnName")
		if _, ok := stats[name]; !ok {
			return nil, glueNotFound("no statistics for column %s", name)
		}
		delete(stats, name)
		return glueRecord{}, nil
	}
}

func glueGetUserDefinedFunctions(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	match, err := glueNameMatcher(glueStr(in, "Pattern"))
	if err != nil {
		return nil, err
	}
	db := glueStr(in, "DatabaseName")
	if db != "" {
		if _, err := st.find("database", []any{db}); err != nil {
			return nil, err
		}
	}
	var out []glueRecord
	for _, f := range st.all("udf") {
		if (db == "" || glueStr(f, "DatabaseName") == db) && match(glueStr(f, "FunctionName")) {
			out = append(out, f)
		}
	}
	return gluePageRecords(out, in, "UserDefinedFunctions")
}

func glueCreateConnection(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	rec, err := st.createFrom(glueOps["CreateConnection"], in)
	if err != nil {
		return nil, err
	}
	rec["Status"] = "READY"
	return glueRecord{"CreateConnectionStatus": "READY"}, nil
}

func glueGetConnection(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	rec, err := st.find("connection", []any{in["Name"]})
	if err != nil {
		return nil, err
	}
	out := glueWrap(rec, "")
	if hide, _ := in["HidePassword"].(bool); hide {
		props := map[string]any{}
		for k, v := range glueTags(rec["ConnectionProperties"]) {
			if k != "PASSWORD" {
				props[k] = v
			}
		}
		out["ConnectionProperties"] = props
	}
	return glueRecord{"Connection": out}, nil
}

func glueBatchDeleteConnection(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	var ok []any
	errs := map[string]any{}
	for _, n := range glueList(in["ConnectionNameList"]) {
		rec, err := st.find("connection", []any{n})
		if err != nil {
			errs[fmt.Sprint(n)] = glueErrorDetail(err)
			continue
		}
		st.remove("connection", rec)
		ok = append(ok, n)
	}
	return glueRecord{"Succeeded": glueNonNil(ok), "Errors": errs}, nil
}

func glueTestConnection(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	if name := glueStr(in, "ConnectionName"); name != "" {
		if _, err := st.find("connection", []any{name}); err != nil {
			return nil, err
		}
	} else if in["TestConnectionInput"] == nil {
		return nil, glueInvalid("ConnectionName or TestConnectionInput is required")
	}
	return glueRecord{}, nil
}

var glueClassifierTypes = []string{"GrokClassifier", "XMLClassifier", "JsonClassifier", "CsvClassifier"}

func glueClassifierInput(in glueRecord) (string, map[string]any, *glueError) {
	var typ string
	var body map[string]any
	for _, t := range glueClassifierTypes {
		if m, ok := in[t].(map[string]any); ok {
			if typ != "" {
				return "", nil, glueInvalid("specify exactly one classifier type")
			}
			typ, body = t, m
		}
	}
	if typ == "" {
		return "", nil, glueInvalid("one of GrokClassifier, XMLClassifier, JsonClassifier or CsvClassifier is required")
	}
	return typ, body, nil
}

func glueCreateClassifier(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	typ, body, err := glueClassifierInput(in)
	if err != nil {
		return nil, err
	}
	now := glueEpoch(time.Now())
	spec := glueRecord{"CreationTime": now, "LastUpdated": now, "Version": float64(1)}
	for k, v := range body {
		spec[k] = v
	}
	_, err = st.create("classifier", glueRecord{"Name": body["Name"], typ: spec}, nil)
	return glueRecord{}, err
}

func glueUpdateClassifier(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	typ, body, err := glueClassifierInput(in)
	if err != nil {
		return nil, err
	}
	rec, err := st.find("classifier", []any{body["Name"]})
	if err != nil {
		return nil, err
	}
	spec, ok := rec[typ].(glueRecord)
	if !ok {
		return nil, glueInvalid("classifier %v is not a %s", body["Name"], typ)
	}
	for k, v := range body {
		spec[k] = v
	}
	spec["LastUpdated"] = glueEpoch(time.Now())
	version, _ := spec["Version"].(float64)
	spec["Version"] = version + 1
	return glueRecord{}, nil
}

// Lake Formation-style unfiltered reads return the stored definitions.

func glueGetUnfilteredTable(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	rec, err := st.find("table", []any{in["DatabaseName"], in["Name"]})
	if err != nil {
		return nil, err
	}
	return glueRecord{"Table": rec, "AuthorizedColumns": glueColumnNames(rec), "IsRegisteredWithLakeFormation": false, "ResourceArn": st.arn("table", rec)}, nil
}

func glueGetUnfilteredPartition(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	table, err := st.find("table", []any{in["DatabaseName"], in["TableName"]})
	if err != nil {
		return nil, err
	}
	rec, err := st.find("partition", []any{in["DatabaseName"], in["TableName"], in["PartitionValues"]})
	if err != nil {
		return nil, err
	}
	return glueRecord{"Partition": rec, "AuthorizedColumns": glueColumnNames(table), "IsRegisteredWithLakeFormation": false}, nil
}

func glueGetUnfilteredPartitions(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	page, err := glueGetPartitions(st, in)
	if err != nil {
		return nil, err
	}
	table, _ := st.find("table", []any{in["DatabaseName"], in["TableName"]})
	var out []any
	for _, p := range glueList(page["Partitions"]) {
		out = append(out, glueRecord{"Partition": p, "AuthorizedColumns": glueColumnNames(table), "IsRegisteredWithLakeFormation": false})
	}
	result := glueRecord{"UnfilteredPartitions": glueNonNil(out)}
	if next, ok := page["NextToken"]; ok {
		result["NextToken"] = next
	}
	return result, nil
}

func glueColumnNames(table glueRecord) []any {
	var out []any
	if sd, ok := table["StorageDescriptor"].(map[string]any); ok {
		for _, c := range glueList(sd["Columns"]) {
			if m, ok := c.(map[string]any); ok {
				out = append(out, m["Name"])
			}
		}
	}
	return glueNonNil(out)
}

// Singleton settings (one per account).

func glueSinglePut(name, member string) glueHandler {
	return func(st *glueStore, in glueRecord) (glueRecord, *glueError) {
		rec := glueRecord{}
		src := in
		if member != "" {
			src, _ = in[member].(map[string]any)
		}
		for k, v := range src {
			if k != "ClientToken" && k != "CatalogId" {
				rec[k] = v
			}
		}
		now := glueEpoch(time.Now())
		if prev, ok := st.singles[name]; ok && prev["CreatedAt"] != nil {
			rec["CreatedAt"] = prev["CreatedAt"]
		} else {
			rec["CreatedAt"] = now
		}
		rec["UpdatedAt"], rec["Status"] = now, "ACTIVE"
		st.singles[name] = rec
		return rec, nil
	}
}

func glueSingleGet(name, member string) glueHandler {
	return func(st *glueStore, in glueRecord) (glueRecord, *glueError) {
		rec, ok := st.singles[name]
		if !ok {
			if member == "" {
				return nil, glueNotFound("%s is not configured", name)
			}
			rec = glueRecord{}
		}
		if member != "" {
			return glueRecord{member: rec}, nil
		}
		return rec, nil
	}
}

// Shared helpers.

func glueList(v any) []any {
	l, _ := v.([]any)
	return l
}

func glueNonNil(l []any) []any {
	if l == nil {
		return []any{}
	}
	return l
}

func glueErrorDetail(err *glueError) glueRecord {
	return glueRecord{"ErrorCode": err.code, "ErrorMessage": err.msg}
}

func glueFirstRecord(recs ...glueRecord) glueRecord {
	for _, r := range recs {
		if r != nil {
			return r
		}
	}
	return glueRecord{}
}
