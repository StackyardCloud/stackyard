package server

import (
	"fmt"
	"net/http"
	"testing"
)

// Data Catalog: databases, tables with versions, partitions, indexes,
// column statistics, functions, connections and the Athena mirror.

const glueTestTable = `{"DatabaseName":"sales","TableInput":{"Name":"orders","Description":"order facts",
 "StorageDescriptor":{"Location":"s3://bucket/orders/","Columns":[{"Name":"id","Type":"bigint"},{"Name":"amount","Type":"double"}]},
 "PartitionKeys":[{"Name":"year","Type":"string"},{"Name":"month","Type":"int"}]}}`

func TestGlueStage1DatabasesAndTables(t *testing.T) {
	srv, ts := newGlueTestServer(t)

	glueCall(t, ts, http.StatusOK, "CreateDatabase", `{"DatabaseInput":{"Name":"sales","Description":"sales data"},"Tags":{"team":"bi"}}`)
	if code := glueErrorType(t, ts, "CreateDatabase", `{"DatabaseInput":{"Name":"sales"}}`); code != "AlreadyExistsException" {
		t.Fatalf("duplicate database: %s", code)
	}
	db := glueCall(t, ts, http.StatusOK, "GetDatabase", `{"Name":"sales"}`)["Database"].(map[string]any)
	if db["Description"] != "sales data" || db["CatalogId"] != glueAccount || db["CreateTime"] == nil {
		t.Fatalf("database: %v", db)
	}
	if list := glueCall(t, ts, http.StatusOK, "GetDatabases", `{}`)["DatabaseList"].([]any); len(list) != 2 {
		t.Fatalf("databases (default + sales): %v", list)
	}

	if code := glueErrorType(t, ts, "CreateTable", `{"DatabaseName":"nope","TableInput":{"Name":"t"}}`); code != "EntityNotFoundException" {
		t.Fatalf("table in a missing database: %s", code)
	}
	glueCall(t, ts, http.StatusOK, "CreateTable", glueTestTable)
	table := glueCall(t, ts, http.StatusOK, "GetTable", `{"DatabaseName":"sales","Name":"orders"}`)["Table"].(map[string]any)
	if table["VersionId"] != "1" || table["DatabaseName"] != "sales" || table["StorageDescriptor"].(map[string]any)["Location"] != "s3://bucket/orders/" {
		t.Fatalf("table: %v", table)
	}

	// Athena's AwsDataCatalog sees the Glue table.
	athenaTable, err := srv.athena.GetTable(glueAthenaCatalog, "sales", "orders")
	if err != nil || len(athenaTable.Columns) != 2 {
		t.Fatalf("athena mirror: %v %v", athenaTable, err)
	}

	// Updates bump the version and archive the previous definition.
	glueCall(t, ts, http.StatusOK, "UpdateTable", `{"DatabaseName":"sales","TableInput":{"Name":"orders","Description":"v2"}}`)
	if code := glueErrorType(t, ts, "UpdateTable", `{"DatabaseName":"sales","VersionId":"1","TableInput":{"Name":"orders"}}`); code != "ConcurrentModificationException" {
		t.Fatalf("stale VersionId: %s", code)
	}
	versions := glueCall(t, ts, http.StatusOK, "GetTableVersions", `{"DatabaseName":"sales","TableName":"orders"}`)["TableVersions"].([]any)
	if len(versions) != 2 {
		t.Fatalf("versions: %v", versions)
	}
	v1 := glueCall(t, ts, http.StatusOK, "GetTableVersion", `{"DatabaseName":"sales","TableName":"orders","VersionId":"1"}`)["TableVersion"].(map[string]any)
	if v1["Table"].(map[string]any)["Description"] != "order facts" {
		t.Fatalf("version 1: %v", v1)
	}
	if code := glueErrorType(t, ts, "DeleteTableVersion", `{"DatabaseName":"sales","TableName":"orders","VersionId":"2"}`); code != "InvalidInputException" {
		t.Fatalf("deleting the current version: %s", code)
	}
	glueCall(t, ts, http.StatusOK, "DeleteTableVersion", `{"DatabaseName":"sales","TableName":"orders","VersionId":"1"}`)

	glueCall(t, ts, http.StatusOK, "CreateTable", `{"DatabaseName":"sales","TableInput":{"Name":"customers"}}`)
	names := glueCall(t, ts, http.StatusOK, "GetTables", `{"DatabaseName":"sales","Expression":"cust*"}`)["TableList"].([]any)
	if len(names) != 1 || names[0].(map[string]any)["Name"] != "customers" {
		t.Fatalf("expression filter: %v", names)
	}
	page := glueCall(t, ts, http.StatusOK, "GetTables", `{"DatabaseName":"sales","MaxResults":1}`)
	if len(page["TableList"].([]any)) != 1 || page["NextToken"] == nil {
		t.Fatalf("paging: %v", page)
	}
	if hits := glueCall(t, ts, http.StatusOK, "SearchTables", `{"SearchText":"order"}`)["TableList"].([]any); len(hits) != 1 {
		t.Fatalf("search: %v", hits)
	}
	errs := glueCall(t, ts, http.StatusOK, "BatchDeleteTable", `{"DatabaseName":"sales","TablesToDelete":["customers","ghost"]}`)["Errors"].([]any)
	if len(errs) != 1 || errs[0].(map[string]any)["TableName"] != "ghost" {
		t.Fatalf("batch delete errors: %v", errs)
	}

	// Deleting the database removes its tables, here and in Athena.
	glueCall(t, ts, http.StatusOK, "DeleteDatabase", `{"Name":"sales"}`)
	if code := glueErrorType(t, ts, "GetTable", `{"DatabaseName":"sales","Name":"orders"}`); code != "EntityNotFoundException" {
		t.Fatalf("table after database delete: %s", code)
	}
	if _, err := srv.athena.GetTable(glueAthenaCatalog, "sales", "orders"); err == nil {
		t.Fatal("athena still has the table")
	}
}

func TestGlueStage1PartitionsIndexesAndStatistics(t *testing.T) {
	_, ts := newGlueTestServer(t)
	glueCall(t, ts, http.StatusOK, "CreateDatabase", `{"DatabaseInput":{"Name":"sales"}}`)
	glueCall(t, ts, http.StatusOK, "CreateTable", glueTestTable)

	for _, ym := range [][2]string{{"2024", "1"}, {"2024", "6"}, {"2025", "3"}} {
		glueCall(t, ts, http.StatusOK, "CreatePartition", fmt.Sprintf(`{"DatabaseName":"sales","TableName":"orders","PartitionInput":{"Values":[%q,%q]}}`, ym[0], ym[1]))
	}
	if code := glueErrorType(t, ts, "CreatePartition", `{"DatabaseName":"sales","TableName":"orders","PartitionInput":{"Values":["2024","1"]}}`); code != "AlreadyExistsException" {
		t.Fatalf("duplicate partition: %s", code)
	}
	if code := glueErrorType(t, ts, "CreatePartition", `{"DatabaseName":"sales","TableName":"orders","PartitionInput":{"Values":["2024"]}}`); code != "InvalidInputException" {
		t.Fatalf("wrong number of values: %s", code)
	}
	parts := glueCall(t, ts, http.StatusOK, "GetPartitions", `{"DatabaseName":"sales","TableName":"orders","Expression":"year = '2024' AND month >= 6"}`)["Partitions"].([]any)
	if len(parts) != 1 || fmt.Sprint(parts[0].(map[string]any)["Values"]) != "[2024 6]" {
		t.Fatalf("expression: %v", parts)
	}
	if code := glueErrorType(t, ts, "GetPartitions", `{"DatabaseName":"sales","TableName":"orders","Expression":"day = 1"}`); code != "InvalidInputException" {
		t.Fatalf("unknown key in expression: %s", code)
	}

	errs := glueCall(t, ts, http.StatusOK, "BatchCreatePartition", `{"DatabaseName":"sales","TableName":"orders","PartitionInputList":[{"Values":["2025","4"]},{"Values":["2024","1"]}]}`)["Errors"].([]any)
	if len(errs) != 1 || errs[0].(map[string]any)["ErrorDetail"].(map[string]any)["ErrorCode"] != "AlreadyExistsException" {
		t.Fatalf("batch create: %v", errs)
	}
	got := glueCall(t, ts, http.StatusOK, "BatchGetPartition", `{"DatabaseName":"sales","TableName":"orders","PartitionsToGet":[{"Values":["2025","4"]},{"Values":["1999","1"]}]}`)["Partitions"].([]any)
	if len(got) != 1 {
		t.Fatalf("batch get: %v", got)
	}
	glueCall(t, ts, http.StatusOK, "UpdatePartition", `{"DatabaseName":"sales","TableName":"orders","PartitionValueList":["2025","4"],"PartitionInput":{"Values":["2025","4"],"Parameters":{"k":"v"}}}`)
	p := glueCall(t, ts, http.StatusOK, "GetPartition", `{"DatabaseName":"sales","TableName":"orders","PartitionValues":["2025","4"]}`)["Partition"].(map[string]any)
	if p["Parameters"].(map[string]any)["k"] != "v" {
		t.Fatalf("updated partition: %v", p)
	}
	glueCall(t, ts, http.StatusOK, "BatchDeletePartition", `{"DatabaseName":"sales","TableName":"orders","PartitionsToDelete":[{"Values":["2025","4"]}]}`)
	if code := glueErrorType(t, ts, "GetPartition", `{"DatabaseName":"sales","TableName":"orders","PartitionValues":["2025","4"]}`); code != "EntityNotFoundException" {
		t.Fatalf("deleted partition: %s", code)
	}

	// Partition indexes must use partition keys, at most three per table.
	if code := glueErrorType(t, ts, "CreatePartitionIndex", `{"DatabaseName":"sales","TableName":"orders","PartitionIndex":{"IndexName":"bad","Keys":["amount"]}}`); code != "InvalidInputException" {
		t.Fatalf("non-partition index key: %s", code)
	}
	glueCall(t, ts, http.StatusOK, "CreatePartitionIndex", `{"DatabaseName":"sales","TableName":"orders","PartitionIndex":{"IndexName":"by_year","Keys":["year"]}}`)
	idx := glueCall(t, ts, http.StatusOK, "GetPartitionIndexes", `{"DatabaseName":"sales","TableName":"orders"}`)["PartitionIndexDescriptorList"].([]any)
	if len(idx) != 1 || idx[0].(map[string]any)["IndexStatus"] != "ACTIVE" {
		t.Fatalf("indexes: %v", idx)
	}
	glueCall(t, ts, http.StatusOK, "DeletePartitionIndex", `{"DatabaseName":"sales","TableName":"orders","IndexName":"by_year"}`)
	if code := glueErrorType(t, ts, "DeletePartitionIndex", `{"DatabaseName":"sales","TableName":"orders","IndexName":"by_year"}`); code != "EntityNotFoundException" {
		t.Fatalf("deleting a missing index: %s", code)
	}

	glueCall(t, ts, http.StatusOK, "UpdateColumnStatisticsForTable", `{"DatabaseName":"sales","TableName":"orders","ColumnStatisticsList":[{"ColumnName":"amount","ColumnType":"double","AnalyzedTime":1700000000,"StatisticsData":{"Type":"DOUBLE","DoubleColumnStatisticsData":{"NumberOfNulls":0,"NumberOfDistinctValues":10}}}]}`)
	stats := glueCall(t, ts, http.StatusOK, "GetColumnStatisticsForTable", `{"DatabaseName":"sales","TableName":"orders","ColumnNames":["amount","id"]}`)
	if len(stats["ColumnStatisticsList"].([]any)) != 1 || len(stats["Errors"].([]any)) != 1 {
		t.Fatalf("column statistics: %v", stats)
	}
	glueCall(t, ts, http.StatusOK, "DeleteColumnStatisticsForTable", `{"DatabaseName":"sales","TableName":"orders","ColumnName":"amount"}`)
	if code := glueErrorType(t, ts, "DeleteColumnStatisticsForTable", `{"DatabaseName":"sales","TableName":"orders","ColumnName":"amount"}`); code != "EntityNotFoundException" {
		t.Fatalf("deleting missing statistics: %s", code)
	}
}

func TestGlueStage1FunctionsConnectionsAndClassifiers(t *testing.T) {
	_, ts := newGlueTestServer(t)
	glueCall(t, ts, http.StatusOK, "CreateDatabase", `{"DatabaseInput":{"Name":"udfs"}}`)
	glueCall(t, ts, http.StatusOK, "CreateUserDefinedFunction", `{"DatabaseName":"udfs","FunctionInput":{"FunctionName":"to_upper","ClassName":"com.example.Upper","OwnerName":"me","OwnerType":"USER"}}`)
	fn := glueCall(t, ts, http.StatusOK, "GetUserDefinedFunction", `{"DatabaseName":"udfs","FunctionName":"to_upper"}`)["UserDefinedFunction"].(map[string]any)
	if fn["ClassName"] != "com.example.Upper" {
		t.Fatalf("function: %v", fn)
	}
	if fns := glueCall(t, ts, http.StatusOK, "GetUserDefinedFunctions", `{"Pattern":"to_*"}`)["UserDefinedFunctions"].([]any); len(fns) != 1 {
		t.Fatalf("pattern: %v", fns)
	}

	if st := glueCall(t, ts, http.StatusOK, "CreateConnection", `{"ConnectionInput":{"Name":"pg","ConnectionType":"JDBC","ConnectionProperties":{"JDBC_CONNECTION_URL":"jdbc:postgresql://db/x","USERNAME":"u","PASSWORD":"secret"}}}`); st["CreateConnectionStatus"] != "READY" {
		t.Fatalf("create connection: %v", st)
	}
	conn := glueCall(t, ts, http.StatusOK, "GetConnection", `{"Name":"pg","HidePassword":true}`)["Connection"].(map[string]any)
	if _, leaked := conn["ConnectionProperties"].(map[string]any)["PASSWORD"]; leaked {
		t.Fatalf("HidePassword: %v", conn)
	}
	res := glueCall(t, ts, http.StatusOK, "BatchDeleteConnection", `{"ConnectionNameList":["pg","nope"]}`)
	if len(res["Succeeded"].([]any)) != 1 || res["Errors"].(map[string]any)["nope"] == nil {
		t.Fatalf("batch delete connections: %v", res)
	}
	if code := glueErrorType(t, ts, "TestConnection", `{"ConnectionName":"pg"}`); code != "EntityNotFoundException" {
		t.Fatalf("testing a deleted connection: %s", code)
	}

	if code := glueErrorType(t, ts, "CreateClassifier", `{}`); code != "InvalidInputException" {
		t.Fatalf("classifier without a type: %s", code)
	}
	glueCall(t, ts, http.StatusOK, "CreateClassifier", `{"CsvClassifier":{"Name":"csv","Delimiter":","}}`)
	glueCall(t, ts, http.StatusOK, "UpdateClassifier", `{"CsvClassifier":{"Name":"csv","Delimiter":";"}}`)
	cl := glueCall(t, ts, http.StatusOK, "GetClassifier", `{"Name":"csv"}`)["Classifier"].(map[string]any)["CsvClassifier"].(map[string]any)
	if cl["Delimiter"] != ";" || cl["Version"] != 2.0 {
		t.Fatalf("classifier: %v", cl)
	}
	if code := glueErrorType(t, ts, "UpdateClassifier", `{"GrokClassifier":{"Name":"csv","Classification":"x","GrokPattern":"%{WORD}"}}`); code != "InvalidInputException" {
		t.Fatalf("changing a classifier's type: %s", code)
	}
}
