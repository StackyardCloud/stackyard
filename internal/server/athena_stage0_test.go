package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func athenaRequest(t *testing.T, ts *httptest.Server, target string, body []byte) *http.Response {
	t.Helper()
	headers := map[string]string{
		"Content-Type": "application/x-amz-json-1.0",
		"X-Amz-Target": target,
	}
	return signedRequestWithService(t, http.MethodPost, ts.URL+"/", body, headers, "athena")
}

func TestAthenaStage0WorkGroupsAndQueries(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	createWG := []byte(`{"Name":"demo-wg","Description":"demo","State":"ENABLED"}`)
	resp := athenaRequest(t, ts, "AmazonAthena.CreateWorkGroup", createWG)
	assertStatus(t, resp, http.StatusOK)

	getWG := []byte(`{"WorkGroup":"demo-wg"}`)
	resp = athenaRequest(t, ts, "AmazonAthena.GetWorkGroup", getWG)
	assertStatus(t, resp, http.StatusOK)
	var getWGResp struct {
		WorkGroup struct {
			Name string `json:"Name"`
		} `json:"WorkGroup"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &getWGResp); err != nil {
		t.Fatalf("get workgroup unmarshal: %v", err)
	}
	if getWGResp.WorkGroup.Name != "demo-wg" {
		t.Fatalf("expected workgroup demo-wg, got %q", getWGResp.WorkGroup.Name)
	}

	resp = athenaRequest(t, ts, "AmazonAthena.ListWorkGroups", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	var listWGResp struct {
		WorkGroups []struct {
			Name string `json:"Name"`
		} `json:"WorkGroups"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &listWGResp); err != nil {
		t.Fatalf("list workgroups unmarshal: %v", err)
	}
	if len(listWGResp.WorkGroups) == 0 {
		t.Fatalf("expected workgroups")
	}

	startQuery := []byte(`{"QueryString":"SELECT 1","QueryExecutionContext":{"Database":"db1","Catalog":"AwsDataCatalog"},"WorkGroup":"demo-wg","ResultConfiguration":{"OutputLocation":"s3://demo/output/"}}`)
	resp = athenaRequest(t, ts, "AmazonAthena.StartQueryExecution", startQuery)
	assertStatus(t, resp, http.StatusOK)
	var startResp struct {
		QueryExecutionId string `json:"QueryExecutionId"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &startResp); err != nil {
		t.Fatalf("start query unmarshal: %v", err)
	}
	if startResp.QueryExecutionId == "" {
		t.Fatalf("expected query execution id")
	}

	getExec := []byte(`{"QueryExecutionId":"` + startResp.QueryExecutionId + `"}`)
	resp = athenaRequest(t, ts, "AmazonAthena.GetQueryExecution", getExec)
	assertStatus(t, resp, http.StatusOK)
	var getExecResp struct {
		QueryExecution struct {
			QueryExecutionId string `json:"QueryExecutionId"`
			Status           struct {
				State string `json:"State"`
			} `json:"Status"`
		} `json:"QueryExecution"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &getExecResp); err != nil {
		t.Fatalf("get query execution unmarshal: %v", err)
	}
	if getExecResp.QueryExecution.QueryExecutionId != startResp.QueryExecutionId {
		t.Fatalf("expected query execution id %q", startResp.QueryExecutionId)
	}

	resp = athenaRequest(t, ts, "AmazonAthena.ListQueryExecutions", []byte(`{"WorkGroup":"demo-wg"}`))
	assertStatus(t, resp, http.StatusOK)
	var listExecResp struct {
		QueryExecutionIds []string `json:"QueryExecutionIds"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &listExecResp); err != nil {
		t.Fatalf("list query executions unmarshal: %v", err)
	}
	if len(listExecResp.QueryExecutionIds) != 1 {
		t.Fatalf("expected 1 query execution")
	}

	resp = athenaRequest(t, ts, "AmazonAthena.GetQueryResults", getExec)
	assertStatus(t, resp, http.StatusOK)
	var resultsResp struct {
		ResultSet struct {
			Rows []struct {
				Data []struct {
					VarCharValue string `json:"VarCharValue"`
				} `json:"Data"`
			} `json:"Rows"`
		} `json:"ResultSet"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &resultsResp); err != nil {
		t.Fatalf("get query results unmarshal: %v", err)
	}
	if len(resultsResp.ResultSet.Rows) == 0 {
		t.Fatalf("expected query results")
	}

	resp = athenaRequest(t, ts, "AmazonAthena.BatchGetQueryExecution", []byte(`{"QueryExecutionIds":["`+startResp.QueryExecutionId+`"]}`))
	assertStatus(t, resp, http.StatusOK)

	resp = athenaRequest(t, ts, "AmazonAthena.StopQueryExecution", getExec)
	assertStatus(t, resp, http.StatusOK)

	resp = athenaRequest(t, ts, "AmazonAthena.DeleteWorkGroup", []byte(`{"WorkGroup":"demo-wg"}`))
	assertStatus(t, resp, http.StatusOK)
}

func TestAthenaStage0MetadataAndTags(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	createDB := []byte(`{"CatalogName":"AwsDataCatalog","DatabaseInput":{"Name":"demo-db","Description":"demo"}}`)
	resp := athenaRequest(t, ts, "AmazonAthena.CreateDatabase", createDB)
	assertStatus(t, resp, http.StatusOK)

	createTable := []byte(`{"CatalogName":"AwsDataCatalog","DatabaseName":"demo-db","TableInput":{"Name":"demo-table","StorageDescriptor":{"Columns":[{"Name":"col1"}]}}}`)
	resp = athenaRequest(t, ts, "AmazonAthena.CreateTable", createTable)
	assertStatus(t, resp, http.StatusOK)

	getTable := []byte(`{"CatalogName":"AwsDataCatalog","DatabaseName":"demo-db","TableName":"demo-table"}`)
	resp = athenaRequest(t, ts, "AmazonAthena.GetTableMetadata", getTable)
	assertStatus(t, resp, http.StatusOK)
	var getTableResp struct {
		TableMetadata struct {
			Name string `json:"Name"`
		} `json:"TableMetadata"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &getTableResp); err != nil {
		t.Fatalf("get table metadata unmarshal: %v", err)
	}
	if getTableResp.TableMetadata.Name != "demo-table" {
		t.Fatalf("expected table demo-table")
	}

	resp = athenaRequest(t, ts, "AmazonAthena.ListTableMetadata", []byte(`{"CatalogName":"AwsDataCatalog","DatabaseName":"demo-db"}`))
	assertStatus(t, resp, http.StatusOK)

	createNamedQuery := []byte(`{"Name":"demo-nq","Database":"demo-db","QueryString":"SELECT 1","WorkGroup":"primary"}`)
	resp = athenaRequest(t, ts, "AmazonAthena.CreateNamedQuery", createNamedQuery)
	assertStatus(t, resp, http.StatusOK)
	var namedQueryResp struct {
		NamedQueryId string `json:"NamedQueryId"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &namedQueryResp); err != nil {
		t.Fatalf("create named query unmarshal: %v", err)
	}
	if namedQueryResp.NamedQueryId == "" {
		t.Fatalf("expected named query id")
	}

	resp = athenaRequest(t, ts, "AmazonAthena.GetNamedQuery", []byte(`{"NamedQueryId":"`+namedQueryResp.NamedQueryId+`"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = athenaRequest(t, ts, "AmazonAthena.BatchGetNamedQuery", []byte(`{"NamedQueryIds":["`+namedQueryResp.NamedQueryId+`"]}`))
	assertStatus(t, resp, http.StatusOK)

	resp = athenaRequest(t, ts, "AmazonAthena.CreatePreparedStatement", []byte(`{"WorkGroup":"primary","StatementName":"demo-ps","QueryStatement":"SELECT 1"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = athenaRequest(t, ts, "AmazonAthena.GetPreparedStatement", []byte(`{"WorkGroup":"primary","StatementName":"demo-ps"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = athenaRequest(t, ts, "AmazonAthena.ListPreparedStatements", []byte(`{"WorkGroup":"primary"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = athenaRequest(t, ts, "AmazonAthena.UpdatePreparedStatement", []byte(`{"WorkGroup":"primary","StatementName":"demo-ps","QueryStatement":"SELECT 2"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = athenaRequest(t, ts, "AmazonAthena.DeletePreparedStatement", []byte(`{"WorkGroup":"primary","StatementName":"demo-ps"}`))
	assertStatus(t, resp, http.StatusOK)

	resourceARN := "arn:aws:athena:us-east-1:123456789012:workgroup/primary"
	resp = athenaRequest(t, ts, "AmazonAthena.TagResource", []byte(`{"ResourceARN":"`+resourceARN+`","Tags":[{"Key":"env","Value":"test"}]}`))
	assertStatus(t, resp, http.StatusOK)

	resp = athenaRequest(t, ts, "AmazonAthena.ListTagsForResource", []byte(`{"ResourceARN":"`+resourceARN+`"}`))
	assertStatus(t, resp, http.StatusOK)
	var tagsResp struct {
		Tags []struct {
			Key string `json:"Key"`
		} `json:"Tags"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &tagsResp); err != nil {
		t.Fatalf("list tags unmarshal: %v", err)
	}
	if len(tagsResp.Tags) != 1 || tagsResp.Tags[0].Key != "env" {
		t.Fatalf("expected env tag")
	}

	resp = athenaRequest(t, ts, "AmazonAthena.UntagResource", []byte(`{"ResourceARN":"`+resourceARN+`","TagKeys":["env"]}`))
	assertStatus(t, resp, http.StatusOK)

	resp = athenaRequest(t, ts, "AmazonAthena.DeleteNamedQuery", []byte(`{"NamedQueryId":"`+namedQueryResp.NamedQueryId+`"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = athenaRequest(t, ts, "AmazonAthena.DeleteTable", []byte(`{"CatalogName":"AwsDataCatalog","DatabaseName":"demo-db","TableName":"demo-table"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = athenaRequest(t, ts, "AmazonAthena.DeleteDatabase", []byte(`{"CatalogName":"AwsDataCatalog","DatabaseName":"demo-db"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = athenaRequest(t, ts, "AmazonAthena.GetQueryExecution", []byte(`{"QueryExecutionId":"missing"}`))
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for missing query execution, got %d", resp.StatusCode)
	}
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ResourceNotFoundException") {
		t.Fatalf("expected ResourceNotFoundException error, got %s", body)
	}
}
