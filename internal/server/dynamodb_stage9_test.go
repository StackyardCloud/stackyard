package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDynamoDBStage9IdempotencyAndCrossActionInvariants(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := dynamodbRequest(t, ts, "CreateTable", []byte(`{
		"TableName":"stage9-table",
		"AttributeDefinitions":[{"AttributeName":"pk","AttributeType":"S"}],
		"KeySchema":[{"AttributeName":"pk","KeyType":"HASH"}],
		"BillingMode":"PAY_PER_REQUEST"
	}`))
	assertStatus(t, resp, http.StatusOK)

	resp = dynamodbRequest(t, ts, "DescribeTable", []byte(`{"TableName":"stage9-table"}`))
	assertStatus(t, resp, http.StatusOK)
	var describeOut struct {
		Table struct {
			TableArn string `json:"TableArn"`
		} `json:"Table"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &describeOut); err != nil {
		t.Fatalf("unmarshal describe table response: %v", err)
	}
	tableArn := describeOut.Table.TableArn

	exportPayload := []byte(`{
		"TableArn":"` + tableArn + `",
		"S3Bucket":"stackyard-stage9",
		"ExportFormat":"DYNAMODB_JSON",
		"ClientToken":"stage9-export-token"
	}`)
	resp = dynamodbRequest(t, ts, "ExportTableToPointInTime", exportPayload)
	assertStatus(t, resp, http.StatusOK)
	var firstExport struct {
		ExportDescription struct {
			ExportArn string `json:"ExportArn"`
		} `json:"ExportDescription"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &firstExport); err != nil {
		t.Fatalf("unmarshal first export response: %v", err)
	}

	resp = dynamodbRequest(t, ts, "ExportTableToPointInTime", exportPayload)
	assertStatus(t, resp, http.StatusOK)
	var secondExport struct {
		ExportDescription struct {
			ExportArn string `json:"ExportArn"`
		} `json:"ExportDescription"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &secondExport); err != nil {
		t.Fatalf("unmarshal second export response: %v", err)
	}
	if firstExport.ExportDescription.ExportArn != secondExport.ExportDescription.ExportArn {
		t.Fatalf("expected idempotent export arn, got %q then %q", firstExport.ExportDescription.ExportArn, secondExport.ExportDescription.ExportArn)
	}

	importPayload := []byte(`{
		"ClientToken":"stage9-import-token",
		"InputFormat":"DYNAMODB_JSON",
		"TableCreationParameters":{"TableName":"stage9-imported"}
	}`)
	resp = dynamodbRequest(t, ts, "ImportTable", importPayload)
	assertStatus(t, resp, http.StatusOK)
	var firstImport struct {
		ImportTableDescription struct {
			ImportArn string `json:"ImportArn"`
		} `json:"ImportTableDescription"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &firstImport); err != nil {
		t.Fatalf("unmarshal first import response: %v", err)
	}

	resp = dynamodbRequest(t, ts, "ImportTable", importPayload)
	assertStatus(t, resp, http.StatusOK)
	var secondImport struct {
		ImportTableDescription struct {
			ImportArn string `json:"ImportArn"`
		} `json:"ImportTableDescription"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &secondImport); err != nil {
		t.Fatalf("unmarshal second import response: %v", err)
	}
	if firstImport.ImportTableDescription.ImportArn != secondImport.ImportTableDescription.ImportArn {
		t.Fatalf("expected idempotent import arn, got %q then %q", firstImport.ImportTableDescription.ImportArn, secondImport.ImportTableDescription.ImportArn)
	}

	resp = dynamodbRequest(t, ts, "DeleteTable", []byte(`{"TableName":"stage9-table"}`))
	assertStatus(t, resp, http.StatusOK)
	resp = dynamodbRequest(t, ts, "DescribeTimeToLive", []byte(`{"TableName":"stage9-table"}`))
	assertStatus(t, resp, http.StatusBadRequest)
	if !strings.Contains(string(mustBody(t, resp)), "ResourceNotFoundException") {
		t.Fatalf("expected ResourceNotFoundException for deleted table ttl lookup")
	}
}

func TestDynamoDBStage9AllCatalogActionsRouteWithoutNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range dynamodbOperations {
		resp := dynamodbRequest(t, ts, op.Name, []byte(`{}`))
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("%s returned NotImplemented", op.Name)
		}
	}
}
