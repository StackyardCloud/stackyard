package server

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func bcmdeExportJSON(name, bucket, query, format, compression string) string {
	return `{"Name":"` + name + `","DataQuery":{"QueryStatement":"` + query + `",` +
		`"TableConfigurations":{"COST_AND_USAGE_REPORT":{"INCLUDE_RESOURCES":"TRUE"}}},` +
		`"DestinationConfigurations":{"S3Destination":{"S3Bucket":"` + bucket + `","S3Prefix":"exports","S3Region":"us-east-1",` +
		`"S3OutputConfigurations":{"OutputType":"CUSTOM","Format":"` + format + `","Compression":"` + compression + `","Overwrite":"OVERWRITE_REPORT"}}},` +
		`"RefreshCadence":{"Frequency":"SYNCHRONOUS"}}`
}

func bcmdeMonth() string { return time.Now().UTC().Format("2006-01") }

const bcmdeQuery = "SELECT line_item_usage_account_id, line_item_unblended_cost, line_item_resource_id FROM COST_AND_USAGE_REPORT"

func TestBCMDataExportsStage1ExportLifecycle(t *testing.T) {
	srv, ts := newBudTestServer(t)
	if _, err := srv.s3.CreateBucket("billing"); err != nil {
		t.Fatal(err)
	}
	arn := budOKAPI(t, ts, bcmdeAPI, "CreateExport", `{"Export":`+bcmdeExportJSON("daily", "billing", bcmdeQuery, "TEXT_OR_CSV", "GZIP")+`,"ResourceTags":[{"Key":"team","Value":"finance"}]}`)["ExportArn"].(string)
	if !strings.HasPrefix(arn, "arn:aws:bcm-data-exports:us-east-1:123456789012:export/daily-") {
		t.Fatalf("arn %s", arn)
	}
	if code := budErrAPI(t, ts, bcmdeAPI, "CreateExport", `{"Export":`+bcmdeExportJSON("daily", "billing", bcmdeQuery, "TEXT_OR_CSV", "GZIP")+`}`, http.StatusBadRequest); code != "ValidationException" {
		t.Fatalf("duplicate name: %s", code)
	}
	got := budOKAPI(t, ts, bcmdeAPI, "GetExport", `{"ExportArn":"`+arn+`"}`)
	if got["ExportStatus"].(map[string]any)["StatusCode"] != "HEALTHY" || got["Export"].(map[string]any)["ExportArn"] != arn {
		t.Fatalf("export: %v", got)
	}

	// The execution delivered a header-only gzip CSV and a manifest.
	obj, err := srv.s3.GetObject("billing", "exports/daily/data/BILLING_PERIOD="+bcmdeMonth()+"/daily-00001.csv.gz")
	if err != nil {
		t.Fatal(err)
	}
	zr, err := gzip.NewReader(bytes.NewReader(obj.Body))
	if err != nil {
		t.Fatal(err)
	}
	header, _ := io.ReadAll(zr)
	if string(header) != "line_item_usage_account_id,line_item_unblended_cost,line_item_resource_id\n" {
		t.Fatalf("csv header %q", header)
	}
	man, err := srv.s3.GetObject("billing", "exports/daily/metadata/BILLING_PERIOD="+bcmdeMonth()+"/daily-Manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	var manifest map[string]any
	if err := json.Unmarshal(man.Body, &manifest); err != nil || manifest["exportArn"] != arn {
		t.Fatalf("manifest %s: %v", man.Body, err)
	}

	execs := budOKAPI(t, ts, bcmdeAPI, "ListExecutions", `{"ExportArn":"`+arn+`"}`)["Executions"].([]any)
	if len(execs) != 1 {
		t.Fatalf("executions: %v", execs)
	}
	id := execs[0].(map[string]any)["ExecutionId"].(string)
	ex := budOKAPI(t, ts, bcmdeAPI, "GetExecution", `{"ExportArn":"`+arn+`","ExecutionId":"`+id+`"}`)
	if ex["ExecutionStatus"].(map[string]any)["StatusCode"] != "DELIVERY_SUCCESS" {
		t.Fatalf("execution: %v", ex)
	}

	budOKAPI(t, ts, bcmdeAPI, "UpdateExport", `{"ExportArn":"`+arn+`","Export":`+bcmdeExportJSON("daily", "missing-bucket", bcmdeQuery, "TEXT_OR_CSV", "ZIP")+`}`)
	got = budOKAPI(t, ts, bcmdeAPI, "GetExport", `{"ExportArn":"`+arn+`"}`)
	if st := got["ExportStatus"].(map[string]any); st["StatusCode"] != "UNHEALTHY" || st["StatusReason"] != "INSUFFICIENT_PERMISSION" {
		t.Fatalf("missing bucket: %v", st)
	}
	execs = budOKAPI(t, ts, bcmdeAPI, "ListExecutions", `{"ExportArn":"`+arn+`"}`)["Executions"].([]any)
	if len(execs) != 2 || execs[0].(map[string]any)["ExecutionStatus"].(map[string]any)["StatusCode"] != "DELIVERY_FAILURE" {
		t.Fatalf("executions after update: %v", execs)
	}

	list := budOKAPI(t, ts, bcmdeAPI, "ListExports", `{}`)["Exports"].([]any)
	if len(list) != 1 || list[0].(map[string]any)["ExportName"] != "daily" {
		t.Fatalf("list: %v", list)
	}

	res := `"ResourceArn":"` + arn + `"`
	budOKAPI(t, ts, bcmdeAPI, "TagResource", `{`+res+`,"ResourceTags":[{"Key":"env","Value":"prod"}]}`)
	budOKAPI(t, ts, bcmdeAPI, "UntagResource", `{`+res+`,"ResourceTagKeys":["team"]}`)
	tags := budOKAPI(t, ts, bcmdeAPI, "ListTagsForResource", `{`+res+`}`)["ResourceTags"].([]any)
	if len(tags) != 1 || tags[0].(map[string]any)["Key"] != "env" {
		t.Fatalf("tags: %v", tags)
	}

	budOKAPI(t, ts, bcmdeAPI, "DeleteExport", `{"ExportArn":"`+arn+`"}`)
	if code := budErrAPI(t, ts, bcmdeAPI, "GetExport", `{"ExportArn":"`+arn+`"}`, http.StatusNotFound); code != "ResourceNotFoundException" {
		t.Fatalf("deleted: %s", code)
	}
}

func TestBCMDataExportsStage1Validation(t *testing.T) {
	_, ts := newBudTestServer(t)
	cases := map[string]string{
		"unknown table":            bcmdeExportJSON("a", "b", "SELECT x FROM NOPE", "TEXT_OR_CSV", "GZIP"),
		"unknown column":           bcmdeExportJSON("a", "b", "SELECT colour FROM COST_AND_USAGE_REPORT", "TEXT_OR_CSV", "GZIP"),
		"not a select":             bcmdeExportJSON("a", "b", "DROP TABLE COST_AND_USAGE_REPORT", "TEXT_OR_CSV", "GZIP"),
		"format mismatch":          bcmdeExportJSON("a", "b", bcmdeQuery, "PARQUET", "GZIP"),
		"bad name":                 bcmdeExportJSON("a b", "b", bcmdeQuery, "TEXT_OR_CSV", "GZIP"),
		"config for another table": bcmdeExportJSON("a", "b", "SELECT Tags FROM FOCUS_1_0_AWS", "TEXT_OR_CSV", "GZIP"),
	}
	for name, export := range cases {
		if code := budErrAPI(t, ts, bcmdeAPI, "CreateExport", `{"Export":`+export+`}`, http.StatusBadRequest); code != "ValidationException" {
			t.Errorf("%s: %s", name, code)
		}
	}
	// line_item_resource_id exists only with INCLUDE_RESOURCES=TRUE.
	noRes := strings.Replace(bcmdeExportJSON("a", "b", bcmdeQuery, "TEXT_OR_CSV", "GZIP"), `"INCLUDE_RESOURCES":"TRUE"`, `"INCLUDE_RESOURCES":"FALSE"`, 1)
	if code := budErrAPI(t, ts, bcmdeAPI, "CreateExport", `{"Export":`+noRes+`}`, http.StatusBadRequest); code != "ValidationException" {
		t.Errorf("resource column without INCLUDE_RESOURCES: %s", code)
	}
	if code := budErrAPI(t, ts, bcmdeAPI, "GetExport", `{"ExportArn":"arn:aws:bcm-data-exports:us-east-1:123456789012:export/none"}`, http.StatusNotFound); code != "ResourceNotFoundException" {
		t.Errorf("missing export: %s", code)
	}
}

func TestBCMDataExportsStage1Tables(t *testing.T) {
	_, ts := newBudTestServer(t)
	tables := budOKAPI(t, ts, bcmdeAPI, "ListTables", `{}`)["Tables"].([]any)
	if len(tables) != len(bcmdeTables) {
		t.Fatalf("tables: %d", len(tables))
	}
	without := budOKAPI(t, ts, bcmdeAPI, "GetTable", `{"TableName":"COST_AND_USAGE_REPORT"}`)
	with := budOKAPI(t, ts, bcmdeAPI, "GetTable", `{"TableName":"COST_AND_USAGE_REPORT","TableProperties":{"INCLUDE_RESOURCES":"TRUE"}}`)
	if len(with["Schema"].([]any)) != len(without["Schema"].([]any))+1 || with["TableProperties"].(map[string]any)["TIME_GRANULARITY"] != "HOURLY" {
		t.Fatalf("schema with resources: %v", with["TableProperties"])
	}
	if code := budErrAPI(t, ts, bcmdeAPI, "GetTable", `{"TableName":"COST_AND_USAGE_REPORT","TableProperties":{"TIME_GRANULARITY":"WEEKLY"}}`, http.StatusBadRequest); code != "ValidationException" {
		t.Fatalf("bad property: %s", code)
	}
}
