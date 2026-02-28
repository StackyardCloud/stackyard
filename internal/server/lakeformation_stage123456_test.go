package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLakeFormationStage12ResourceTagAndPermissionSurfaces(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resourceARN := "arn:aws:s3:::stage-lakeformation-data"

	resp := lakeFormationRequest(t, ts, "RegisterResource", `{"ResourceArn":"`+resourceARN+`"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = lakeFormationRequest(t, ts, "DescribeResource", `{"ResourceArn":"`+resourceARN+`"}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, resourceARN) {
		t.Fatalf("expected DescribeResource to include %s, got %q", resourceARN, body)
	}

	resp = lakeFormationRequest(t, ts, "CreateLFTag", `{"TagKey":"env","TagValues":["stage","prod"]}`)
	assertStatus(t, resp, http.StatusOK)

	resp = lakeFormationRequest(
		t,
		ts,
		"AddLFTagsToResource",
		`{"Resource":{"DataLocation":{"ResourceArn":"`+resourceARN+`"}},"LFTags":[{"TagKey":"env","TagValues":["stage"]}]}`,
	)
	assertStatus(t, resp, http.StatusOK)

	resp = lakeFormationRequest(t, ts, "GetResourceLFTags", `{"Resource":{"DataLocation":{"ResourceArn":"`+resourceARN+`"}}}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "\"env\"") {
		t.Fatalf("expected GetResourceLFTags to include env tag, got %q", body)
	}

	resp = lakeFormationRequest(
		t,
		ts,
		"GrantPermissions",
		`{"Principal":{"DataLakePrincipalIdentifier":"arn:aws:iam::123456789012:role/stage"},"Resource":{"DataLocation":{"ResourceArn":"`+resourceARN+`"}},"Permissions":["DATA_LOCATION_ACCESS"]}`,
	)
	assertStatus(t, resp, http.StatusOK)

	resp = lakeFormationRequest(t, ts, "ListPermissions", `{}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "DATA_LOCATION_ACCESS") {
		t.Fatalf("expected ListPermissions to include DATA_LOCATION_ACCESS, got %q", body)
	}

	resp = lakeFormationRequest(
		t,
		ts,
		"CreateLFTagExpression",
		`{"Name":"stage-expression","Expression":[{"TagKey":"env","TagValues":["stage"]}]}`,
	)
	assertStatus(t, resp, http.StatusOK)
	resp = lakeFormationRequest(t, ts, "GetLFTagExpression", `{"Name":"stage-expression"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = lakeFormationRequest(t, ts, "ListLFTagExpressions", `{}`)
	assertStatus(t, resp, http.StatusOK)

	resp = lakeFormationRequest(
		t,
		ts,
		"RemoveLFTagsFromResource",
		`{"Resource":{"DataLocation":{"ResourceArn":"`+resourceARN+`"}},"LFTags":[{"TagKey":"env","TagValues":["stage"]}]}`,
	)
	assertStatus(t, resp, http.StatusOK)
	resp = lakeFormationRequest(
		t,
		ts,
		"RevokePermissions",
		`{"Principal":{"DataLakePrincipalIdentifier":"arn:aws:iam::123456789012:role/stage"},"Resource":{"DataLocation":{"ResourceArn":"`+resourceARN+`"}},"Permissions":["DATA_LOCATION_ACCESS"]}`,
	)
	assertStatus(t, resp, http.StatusOK)

	resp = lakeFormationRequest(t, ts, "DeleteLFTagExpression", `{"Name":"stage-expression"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = lakeFormationRequest(t, ts, "DeleteLFTag", `{"TagKey":"env","TagValues":["stage"]}`)
	assertStatus(t, resp, http.StatusOK)
	resp = lakeFormationRequest(t, ts, "DeregisterResource", `{"ResourceArn":"`+resourceARN+`"}`)
	assertStatus(t, resp, http.StatusOK)
}

func TestLakeFormationStage3DataCellsFiltersAndSearch(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := lakeFormationRequest(
		t,
		ts,
		"CreateDataCellsFilter",
		`{"TableCatalogId":"123456789012","DatabaseName":"stage_db","TableName":"stage_table","Name":"stage_filter","RowFilter":{"FilterExpression":"region = 'us-east-1'"}}`,
	)
	assertStatus(t, resp, http.StatusOK)

	resp = lakeFormationRequest(t, ts, "GetDataCellsFilter", `{"DatabaseName":"stage_db","TableName":"stage_table","Name":"stage_filter"}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "stage_filter") {
		t.Fatalf("expected GetDataCellsFilter to include stage_filter, got %q", body)
	}

	resp = lakeFormationRequest(t, ts, "ListDataCellsFilter", `{}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "stage_filter") {
		t.Fatalf("expected ListDataCellsFilter to include stage_filter, got %q", body)
	}

	resp = lakeFormationRequest(t, ts, "UpdateDataCellsFilter", `{"DatabaseName":"stage_db","TableName":"stage_table","Name":"stage_filter","RowFilter":{"FilterExpression":"TRUE"}}`)
	assertStatus(t, resp, http.StatusOK)

	resp = lakeFormationRequest(t, ts, "SearchDatabasesByLFTags", `{"Expression":[{"TagKey":"environment","TagValues":["dev"]}]}`)
	assertStatus(t, resp, http.StatusOK)
	resp = lakeFormationRequest(t, ts, "SearchTablesByLFTags", `{"Expression":[{"TagKey":"environment","TagValues":["dev"]}]}`)
	assertStatus(t, resp, http.StatusOK)
	resp = lakeFormationRequest(t, ts, "GetEffectivePermissionsForPath", `{"ResourceArn":"arn:aws:s3:::stage-lakeformation-data"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = lakeFormationRequest(t, ts, "DeleteDataCellsFilter", `{"DatabaseName":"stage_db","TableName":"stage_table","Name":"stage_filter"}`)
	assertStatus(t, resp, http.StatusOK)
}

func TestLakeFormationStage45TransactionsQueryCredentialsAndConfiguration(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := lakeFormationRequest(
		t,
		ts,
		"PutDataLakeSettings",
		`{"DataLakeSettings":{"DataLakeAdmins":[{"DataLakePrincipalIdentifier":"arn:aws:iam::123456789012:role/stage-admin"}]}}`,
	)
	assertStatus(t, resp, http.StatusOK)
	resp = lakeFormationRequest(t, ts, "GetDataLakeSettings", `{}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "stage-admin") {
		t.Fatalf("expected GetDataLakeSettings to include stage-admin, got %q", body)
	}

	resp = lakeFormationRequest(t, ts, "StartTransaction", `{"ClientRequestToken":"stage-lf-transaction-token-000001"}`)
	assertStatus(t, resp, http.StatusOK)
	tx := decodeLakeFormationPayload(t, resp)
	txID := lakeFormationTestString(tx, "TransactionId")
	if txID == "" {
		t.Fatalf("expected StartTransaction to include TransactionId")
	}

	resp = lakeFormationRequest(t, ts, "DescribeTransaction", `{"TransactionId":"`+txID+`"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = lakeFormationRequest(t, ts, "ExtendTransaction", `{"TransactionId":"`+txID+`"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = lakeFormationRequest(t, ts, "StartQueryPlanning", `{}`)
	assertStatus(t, resp, http.StatusOK)
	query := decodeLakeFormationPayload(t, resp)
	queryID := lakeFormationTestString(query, "QueryId")
	if queryID == "" {
		t.Fatalf("expected StartQueryPlanning to include QueryId")
	}

	resp = lakeFormationRequest(t, ts, "GetQueryState", `{"QueryId":"`+queryID+`"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = lakeFormationRequest(t, ts, "GetQueryStatistics", `{"QueryId":"`+queryID+`"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = lakeFormationRequest(t, ts, "GetWorkUnits", `{"QueryId":"`+queryID+`","PageSize":10}`)
	assertStatus(t, resp, http.StatusOK)
	resp = lakeFormationRequest(t, ts, "GetWorkUnitResults", `{"QueryId":"`+queryID+`","WorkUnitId":0}`)
	assertStatus(t, resp, http.StatusOK)

	resp = lakeFormationRequest(t, ts, "GetTemporaryDataLocationCredentials", `{"ResourceArn":"arn:aws:s3:::stage-lakeformation-data"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = lakeFormationRequest(t, ts, "GetTemporaryGluePartitionCredentials", `{"DatabaseName":"stage_db","TableName":"stage_table","Partition":{"Values":["2026","02","28"]}}`)
	assertStatus(t, resp, http.StatusOK)
	resp = lakeFormationRequest(t, ts, "GetTemporaryGlueTableCredentials", `{"DatabaseName":"stage_db","TableName":"stage_table"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = lakeFormationRequest(
		t,
		ts,
		"UpdateTableObjects",
		`{"DatabaseName":"stage_db","TableName":"stage_table","WriteOperations":[{"AddObject":{"Uri":"s3://stage-lakeformation-data/stage_db/stage_table/object-000001.parquet"}}]}`,
	)
	assertStatus(t, resp, http.StatusOK)
	resp = lakeFormationRequest(t, ts, "GetTableObjects", `{"DatabaseName":"stage_db","TableName":"stage_table"}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "object-000001.parquet") {
		t.Fatalf("expected GetTableObjects to include object-000001.parquet, got %q", body)
	}

	resp = lakeFormationRequest(
		t,
		ts,
		"UpdateTableStorageOptimizer",
		`{"DatabaseName":"stage_db","TableName":"stage_table","StorageOptimizerType":"COMPACTION","Config":{"RoleArn":"arn:aws:iam::123456789012:role/stage-admin"}}`,
	)
	assertStatus(t, resp, http.StatusOK)
	resp = lakeFormationRequest(t, ts, "ListTableStorageOptimizers", `{}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "COMPACTION") {
		t.Fatalf("expected ListTableStorageOptimizers to include COMPACTION, got %q", body)
	}

	resp = lakeFormationRequest(
		t,
		ts,
		"CreateLakeFormationOptIn",
		`{"Principal":{"DataLakePrincipalIdentifier":"arn:aws:iam::123456789012:role/stage-admin"},"Resource":{"Catalog":{}}}`,
	)
	assertStatus(t, resp, http.StatusOK)
	resp = lakeFormationRequest(t, ts, "ListLakeFormationOptIns", `{}`)
	assertStatus(t, resp, http.StatusOK)

	resp = lakeFormationRequest(
		t,
		ts,
		"CreateLakeFormationIdentityCenterConfiguration",
		`{"CatalogId":"123456789012","InstanceArn":"arn:aws:sso:::instance/ssoins-1234567890abcdef"}`,
	)
	assertStatus(t, resp, http.StatusOK)
	resp = lakeFormationRequest(t, ts, "DescribeLakeFormationIdentityCenterConfiguration", `{}`)
	assertStatus(t, resp, http.StatusOK)
	resp = lakeFormationRequest(t, ts, "UpdateLakeFormationIdentityCenterConfiguration", `{"ApplicationStatus":"ENABLED"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = lakeFormationRequest(t, ts, "DeleteLakeFormationIdentityCenterConfiguration", `{}`)
	assertStatus(t, resp, http.StatusOK)

	resp = lakeFormationRequest(t, ts, "AssumeDecoratedRoleWithSAML", `{}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "Credentials") {
		t.Fatalf("expected AssumeDecoratedRoleWithSAML to include Credentials, got %q", body)
	}

	resp = lakeFormationRequest(t, ts, "DeleteObjectsOnCancel", `{"TransactionId":"`+txID+`"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = lakeFormationRequest(t, ts, "CommitTransaction", `{"TransactionId":"`+txID+`"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = lakeFormationRequest(t, ts, "ListTransactions", `{}`)
	assertStatus(t, resp, http.StatusOK)
	resp = lakeFormationRequest(t, ts, "DeleteLakeFormationOptIn", `{}`)
	assertStatus(t, resp, http.StatusOK)
}

func TestLakeFormationStage6ValidationIdempotencyAndBatchSurfaces(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	token := "stage-lf-idempotent-start-transaction-token-000001"
	resp := lakeFormationRequest(t, ts, "StartTransaction", `{"ClientRequestToken":"`+token+`"}`)
	assertStatus(t, resp, http.StatusOK)
	first := decodeLakeFormationPayload(t, resp)
	firstID := lakeFormationTestString(first, "TransactionId")
	if firstID == "" {
		t.Fatalf("expected first StartTransaction response to include TransactionId")
	}

	resp = lakeFormationRequest(t, ts, "StartTransaction", `{"ClientRequestToken":"`+token+`"}`)
	assertStatus(t, resp, http.StatusOK)
	second := decodeLakeFormationPayload(t, resp)
	secondID := lakeFormationTestString(second, "TransactionId")
	if firstID != secondID {
		t.Fatalf("expected idempotent StartTransaction to return same id: %s != %s", firstID, secondID)
	}

	resp = lakeFormationRequest(
		t,
		ts,
		"BatchGrantPermissions",
		`{"Entries":[{"Id":"grant-1","Principal":{"DataLakePrincipalIdentifier":"arn:aws:iam::123456789012:role/stage-admin"},"Resource":{"Catalog":{}},"Permissions":["ALL"]}]}`,
	)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "Failures") {
		t.Fatalf("expected BatchGrantPermissions to include Failures, got %q", body)
	}

	resp = lakeFormationRequest(
		t,
		ts,
		"BatchRevokePermissions",
		`{"Entries":[{"Id":"revoke-1","Principal":{"DataLakePrincipalIdentifier":"arn:aws:iam::123456789012:role/stage-admin"},"Resource":{"Catalog":{}},"Permissions":["ALL"]}]}`,
	)
	assertStatus(t, resp, http.StatusOK)

	resp = signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/ListResources",
		[]byte(`{"broken":`),
		map[string]string{
			"Content-Type": "application/json",
		},
		"lakeformation",
	)
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "invalid JSON body") {
		t.Fatalf("expected invalid JSON body error, got %q", body)
	}
}

func decodeLakeFormationPayload(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	payload := map[string]any{}
	if err := json.Unmarshal(mustBody(t, resp), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	return payload
}

func lakeFormationTestString(payload map[string]any, key string) string {
	v, ok := payload[key]
	if !ok {
		return ""
	}
	if text, ok := v.(string); ok {
		return strings.TrimSpace(text)
	}
	return ""
}
