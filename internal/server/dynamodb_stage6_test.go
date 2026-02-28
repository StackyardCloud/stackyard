package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDynamoDBStage6DataMobilityAndStreamingCore(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := dynamodbRequest(t, ts, "CreateTable", []byte(`{
		"TableName":"stage6-table",
		"AttributeDefinitions":[{"AttributeName":"pk","AttributeType":"S"}],
		"KeySchema":[{"AttributeName":"pk","KeyType":"HASH"}],
		"BillingMode":"PAY_PER_REQUEST"
	}`))
	assertStatus(t, resp, http.StatusOK)

	resp = dynamodbRequest(t, ts, "DescribeTable", []byte(`{"TableName":"stage6-table"}`))
	assertStatus(t, resp, http.StatusOK)
	var describeTableOut struct {
		Table struct {
			TableArn string `json:"TableArn"`
		} `json:"Table"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &describeTableOut); err != nil {
		t.Fatalf("unmarshal describe table response: %v", err)
	}
	tableArn := describeTableOut.Table.TableArn
	if tableArn == "" {
		t.Fatalf("expected table arn")
	}

	resp = dynamodbRequest(t, ts, "ExportTableToPointInTime", []byte(`{
		"TableArn":"`+tableArn+`",
		"S3Bucket":"stackyard-stage6",
		"ExportFormat":"DYNAMODB_JSON",
		"ClientToken":"stage6-export"
	}`))
	assertStatus(t, resp, http.StatusOK)
	var exportOut struct {
		ExportDescription struct {
			ExportArn string `json:"ExportArn"`
		} `json:"ExportDescription"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &exportOut); err != nil {
		t.Fatalf("unmarshal export response: %v", err)
	}
	if exportOut.ExportDescription.ExportArn == "" {
		t.Fatalf("expected export arn")
	}

	resp = dynamodbRequest(t, ts, "DescribeExport", []byte(`{"ExportArn":"`+exportOut.ExportDescription.ExportArn+`"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = dynamodbRequest(t, ts, "ListExports", []byte(`{"TableArn":"`+tableArn+`","Limit":10}`))
	assertStatus(t, resp, http.StatusOK)

	resp = dynamodbRequest(t, ts, "ImportTable", []byte(`{
		"ClientToken":"stage6-import",
		"InputFormat":"DYNAMODB_JSON",
		"TableCreationParameters":{
			"TableName":"stage6-imported"
		}
	}`))
	assertStatus(t, resp, http.StatusOK)
	var importOut struct {
		ImportTableDescription struct {
			ImportArn string `json:"ImportArn"`
			TableArn  string `json:"TableArn"`
		} `json:"ImportTableDescription"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &importOut); err != nil {
		t.Fatalf("unmarshal import response: %v", err)
	}
	if importOut.ImportTableDescription.ImportArn == "" {
		t.Fatalf("expected import arn")
	}

	resp = dynamodbRequest(t, ts, "DescribeImport", []byte(`{"ImportArn":"`+importOut.ImportTableDescription.ImportArn+`"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = dynamodbRequest(t, ts, "ListImports", []byte(`{"TableArn":"`+importOut.ImportTableDescription.TableArn+`","Limit":10}`))
	assertStatus(t, resp, http.StatusOK)

	streamArn := "arn:aws:kinesis:us-east-1:123456789012:stream/stage6"
	resp = dynamodbRequest(t, ts, "EnableKinesisStreamingDestination", []byte(`{"TableName":"stage6-table","StreamArn":"`+streamArn+`"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = dynamodbRequest(t, ts, "DescribeKinesisStreamingDestination", []byte(`{"TableName":"stage6-table"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = dynamodbRequest(t, ts, "UpdateKinesisStreamingDestination", []byte(`{"TableName":"stage6-table","StreamArn":"`+streamArn+`-updated"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = dynamodbRequest(t, ts, "DisableKinesisStreamingDestination", []byte(`{"TableName":"stage6-table","StreamArn":"`+streamArn+`-updated"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = dynamodbRequest(t, ts, "DescribeEndpoints", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
}

func TestDynamoDBStage6ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := dynamodbRequest(t, ts, "CreateTable", []byte(`{
		"TableName":"stage6-implemented",
		"AttributeDefinitions":[{"AttributeName":"pk","AttributeType":"S"}],
		"KeySchema":[{"AttributeName":"pk","KeyType":"HASH"}],
		"BillingMode":"PAY_PER_REQUEST"
	}`))
	assertStatus(t, resp, http.StatusOK)

	resp = dynamodbRequest(t, ts, "DescribeTable", []byte(`{"TableName":"stage6-implemented"}`))
	assertStatus(t, resp, http.StatusOK)
	var describeOut struct {
		Table struct {
			TableArn string `json:"TableArn"`
		} `json:"Table"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &describeOut); err != nil {
		t.Fatalf("unmarshal describe response: %v", err)
	}

	resp = dynamodbRequest(t, ts, "ExportTableToPointInTime", []byte(`{"TableArn":"`+describeOut.Table.TableArn+`","S3Bucket":"b","ClientToken":"c"}`))
	assertStatus(t, resp, http.StatusOK)
	var exportOut struct {
		ExportDescription struct {
			ExportArn string `json:"ExportArn"`
		} `json:"ExportDescription"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &exportOut); err != nil {
		t.Fatalf("unmarshal export response: %v", err)
	}

	resp = dynamodbRequest(t, ts, "ImportTable", []byte(`{"TableCreationParameters":{"TableName":"stage6-implemented-import"},"InputFormat":"DYNAMODB_JSON","ClientToken":"i"}`))
	assertStatus(t, resp, http.StatusOK)
	var importOut struct {
		ImportTableDescription struct {
			ImportArn string `json:"ImportArn"`
			TableArn  string `json:"TableArn"`
		} `json:"ImportTableDescription"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &importOut); err != nil {
		t.Fatalf("unmarshal import response: %v", err)
	}

	actions := []struct {
		action string
		body   []byte
	}{
		{action: "ExportTableToPointInTime", body: []byte(`{"TableArn":"` + describeOut.Table.TableArn + `","S3Bucket":"b2","ClientToken":"c2"}`)},
		{action: "DescribeExport", body: []byte(`{"ExportArn":"` + exportOut.ExportDescription.ExportArn + `"}`)},
		{action: "ListExports", body: []byte(`{"TableArn":"` + describeOut.Table.TableArn + `","Limit":10}`)},
		{action: "ImportTable", body: []byte(`{"TableCreationParameters":{"TableName":"stage6-implemented-import-2"},"InputFormat":"CSV","ClientToken":"i2"}`)},
		{action: "DescribeImport", body: []byte(`{"ImportArn":"` + importOut.ImportTableDescription.ImportArn + `"}`)},
		{action: "ListImports", body: []byte(`{"TableArn":"` + importOut.ImportTableDescription.TableArn + `","Limit":10}`)},
		{action: "EnableKinesisStreamingDestination", body: []byte(`{"TableName":"stage6-implemented","StreamArn":"arn:aws:kinesis:us-east-1:123456789012:stream/s6"}`)},
		{action: "DisableKinesisStreamingDestination", body: []byte(`{"TableName":"stage6-implemented","StreamArn":"arn:aws:kinesis:us-east-1:123456789012:stream/s6"}`)},
		{action: "UpdateKinesisStreamingDestination", body: []byte(`{"TableName":"stage6-implemented","StreamArn":"arn:aws:kinesis:us-east-1:123456789012:stream/s6-updated"}`)},
		{action: "DescribeKinesisStreamingDestination", body: []byte(`{"TableName":"stage6-implemented"}`)},
		{action: "DescribeEndpoints", body: []byte(`{}`)},
	}

	for _, tc := range actions {
		resp := dynamodbRequest(t, ts, tc.action, tc.body)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("%s returned NotImplemented", tc.action)
		}
	}
}
