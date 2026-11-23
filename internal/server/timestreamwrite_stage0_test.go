package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

func newTimestreamTestServer(t *testing.T) (*Server, *httptest.Server) {
	t.Helper()
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return srv, ts
}

// tsmCallAPI sends one signed Timestream call and decodes the reply.
func tsmCallAPI(t *testing.T, ts *httptest.Server, op, body string) (int, map[string]any) {
	t.Helper()
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte(body), map[string]string{
		"Content-Type": "application/x-amz-json-1.0", "X-Amz-Target": "Timestream_20181101." + op,
	}, "timestream")
	raw := mustBody(t, resp)
	out := map[string]any{}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			t.Fatalf("%s: decode %s: %v", op, raw, err)
		}
	}
	return resp.StatusCode, out
}

func tsmOK(t *testing.T, ts *httptest.Server, op, body string) map[string]any {
	t.Helper()
	status, out := tsmCallAPI(t, ts, op, body)
	if status != http.StatusOK {
		t.Fatalf("%s %s: %d %v", op, body, status, out)
	}
	return out
}

// tsmFail expects a 400 and returns the error type.
func tsmFail(t *testing.T, ts *httptest.Server, op, body string) (string, map[string]any) {
	t.Helper()
	status, out := tsmCallAPI(t, ts, op, body)
	if status != http.StatusBadRequest {
		t.Fatalf("%s %s: status %d (want 400) %v", op, body, status, out)
	}
	return out["__type"].(string), out
}

func tsmNowMillis() string { return fmt.Sprint(time.Now().UnixMilli()) }

func TestTimestreamWriteStage0CatalogMatchesModels(t *testing.T) {
	for name, pair := range map[string]struct {
		ops   []tsmOperation
		model *awsmodels.Model
		want  int
	}{
		"write": {timestreamWriteOperations, awsmodels.TimestreamWriteModel(), 19},
		"query": {timestreamQueryOperations, awsmodels.TimestreamQueryModel(), 15},
	} {
		if len(pair.ops) != pair.want || len(pair.model.Operations) != pair.want {
			t.Fatalf("%s: %d ops, model %d, want %d", name, len(pair.ops), len(pair.model.Operations), pair.want)
		}
		for _, op := range pair.ops {
			if _, ok := pair.model.Operations[op.Name]; !ok {
				t.Errorf("%s: %s not in model", name, op.Name)
			}
			if tsmHandlers[op.Name] == nil {
				t.Errorf("%s: %s has no handler", name, op.Name)
			}
		}
	}
}

func TestTimestreamWriteStage0RoutingLeavesInfluxDBAndOthersAlone(t *testing.T) {
	_, ts := newTimestreamTestServer(t)
	// Timestream for InfluxDB keeps its own target prefix.
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte(`{}`), map[string]string{
		"Content-Type": "application/x-amz-json-1.0", "X-Amz-Target": "AmazonTimestreamInfluxDB.ListDbInstances",
	}, "timestream-influxdb")
	body := mustBody(t, resp)
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), "items") {
		t.Fatalf("influxdb ListDbInstances: %d %s", resp.StatusCode, body)
	}
	// A request signed for another service isn't claimed by Timestream.
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte(`{}`), map[string]string{
		"Content-Type": "application/x-amz-json-1.0", "X-Amz-Target": "Timestream_20181101.ListDatabases",
	}, "s3")
	body = mustBody(t, resp)
	if strings.Contains(string(body), "Databases") {
		t.Fatalf("an s3-signed request reached Timestream: %s", body)
	}
	// Chunked bodies of unknown length are refused before signing.
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/", io.MultiReader(strings.NewReader(`{}`)))
	req.Header.Set("X-Amz-Target", "Timestream_20181101.ListDatabases")
	// Scoped to timestream so this router claims it; the length check runs before signing.
	req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential="+testAccessKey+"/20260101/"+testRegion+"/timestream/aws4_request, SignedHeaders=host, Signature=00")
	resp2, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp2.Body.Close()
	if resp2.StatusCode != http.StatusBadRequest {
		t.Fatalf("chunked body: %d", resp2.StatusCode)
	}
	// Both APIs answer DescribeEndpoints with the Stackyard host.
	out := tsmOK(t, ts, "DescribeEndpoints", `{}`)
	eps, _ := out["Endpoints"].([]any)
	if len(eps) != 1 || eps[0].(map[string]any)["Address"] != strings.TrimPrefix(ts.URL, "http://") {
		t.Fatalf("DescribeEndpoints: %v", out)
	}
}

func TestTimestreamWriteStage0DatabasesAndTables(t *testing.T) {
	_, ts := newTimestreamTestServer(t)
	tsmOK(t, ts, "CreateDatabase", `{"DatabaseName":"db1","Tags":[{"Key":"env","Value":"dev"}]}`)
	if code, _ := tsmFail(t, ts, "CreateDatabase", `{"DatabaseName":"db1"}`); code != "ConflictException" {
		t.Fatalf("duplicate database: %s", code)
	}
	if code, _ := tsmFail(t, ts, "DescribeDatabase", `{"DatabaseName":"nope"}`); code != "ResourceNotFoundException" {
		t.Fatalf("missing database: %s", code)
	}
	tbl := tsmOK(t, ts, "CreateTable", `{"DatabaseName":"db1","TableName":"tbl1"}`)["Table"].(map[string]any)
	rp := tbl["RetentionProperties"].(map[string]any)
	if rp["MemoryStoreRetentionPeriodInHours"] != float64(6) || tbl["TableStatus"] != "ACTIVE" {
		t.Fatalf("table defaults: %v", tbl)
	}
	if code, _ := tsmFail(t, ts, "CreateTable", `{"DatabaseName":"db1","TableName":"tbl2","MagneticStoreWriteProperties":{"EnableMagneticStoreWrites":true,"MagneticStoreRejectedDataLocation":{"S3Configuration":{"BucketName":"no-such-bucket"}}}}`); code != "ValidationException" {
		t.Fatalf("missing rejected-data bucket: %s", code)
	}
	if code, _ := tsmFail(t, ts, "UpdateTable", `{"DatabaseName":"db1","TableName":"tbl1","Schema":{"CompositePartitionKey":[{"Type":"MEASURE","Name":"x"}]}}`); code != "ValidationException" {
		t.Fatalf("MEASURE key with a name: %s", code)
	}
	if code, _ := tsmFail(t, ts, "DeleteDatabase", `{"DatabaseName":"db1"}`); code != "ValidationException" {
		t.Fatalf("delete database with tables: %s", code)
	}
	if got := tsmOK(t, ts, "DescribeDatabase", `{"DatabaseName":"db1"}`)["Database"].(map[string]any)["TableCount"]; got != float64(1) {
		t.Fatalf("TableCount: %v", got)
	}
	tags := tsmOK(t, ts, "ListTagsForResource", `{"ResourceARN":"arn:aws:timestream:us-east-1:123456789012:database/db1"}`)["Tags"].([]any)
	if len(tags) != 1 {
		t.Fatalf("tags: %v", tags)
	}
	if code, _ := tsmFail(t, ts, "TagResource", `{"ResourceARN":"arn:aws:timestream:us-east-1:123456789012:database/nope","Tags":[{"Key":"a","Value":"b"}]}`); code != "ResourceNotFoundException" {
		t.Fatalf("tag missing resource: %s", code)
	}
	tsmOK(t, ts, "DeleteTable", `{"DatabaseName":"db1","TableName":"tbl1"}`)
	tsmOK(t, ts, "DeleteDatabase", `{"DatabaseName":"db1"}`)
}

func TestTimestreamWriteStage0WriteRecords(t *testing.T) {
	_, ts := newTimestreamTestServer(t)
	tsmOK(t, ts, "CreateDatabase", `{"DatabaseName":"db1"}`)
	tsmOK(t, ts, "CreateTable", `{"DatabaseName":"db1","TableName":"tbl1"}`)
	now := tsmNowMillis()
	write := func(records string) (int, map[string]any) {
		return tsmCallAPI(t, ts, "WriteRecords", `{"DatabaseName":"db1","TableName":"tbl1","CommonAttributes":{"Dimensions":[{"Name":"host","Value":"h1"}]},"Records":`+records+`}`)
	}
	status, out := write(`[{"MeasureName":"cpu","MeasureValue":"1.5","Time":"` + now + `"},{"MeasureName":"mem","MeasureValueType":"MULTI","MeasureValues":[{"Name":"used","Value":"10","Type":"BIGINT"}],"Time":"` + now + `"}]`)
	if status != http.StatusOK || out["RecordsIngested"].(map[string]any)["Total"] != float64(2) {
		t.Fatalf("write: %d %v", status, out)
	}
	// Rewriting the same record is idempotent; a different value needs a higher version.
	if status, _ := write(`[{"MeasureName":"cpu","MeasureValue":"1.5","Time":"` + now + `"}]`); status != http.StatusOK {
		t.Fatalf("identical rewrite: %d", status)
	}
	_, rej := tsmFail(t, ts, "WriteRecords", `{"DatabaseName":"db1","TableName":"tbl1","Records":[{"Dimensions":[{"Name":"host","Value":"h1"}],"MeasureName":"cpu","MeasureValue":"9","Time":"`+now+`"},{"Dimensions":[{"Name":"host","Value":"h1"}],"MeasureName":"cpu","MeasureValue":"abc","Time":"`+now+`"},{"Dimensions":[{"Name":"host","Value":"h1"}],"MeasureName":"cpu","MeasureValue":"1","Time":"1000"},{"Dimensions":[{"Name":"host","Value":"h2"}],"MeasureName":"cpu","MeasureValue":"3","Time":"`+now+`"}]}`)
	if rej["__type"] != "RejectedRecordsException" {
		t.Fatalf("rejections: %v", rej)
	}
	records := rej["RejectedRecords"].([]any)
	if len(records) != 3 || records[0].(map[string]any)["ExistingVersion"] != float64(1) {
		t.Fatalf("rejected records: %v", records)
	}
	if status, _ := write(`[{"MeasureName":"cpu","MeasureValue":"9","Version":2,"Time":"` + now + `"}]`); status != http.StatusOK {
		t.Fatalf("versioned upsert: %d", status)
	}
	// The valid record in the rejected batch (host h2) was still written.
	rows := tsmOK(t, ts, "Query", `{"QueryString":"SELECT host, measure_value::double FROM \"db1\".\"tbl1\" WHERE measure_name = 'cpu' ORDER BY host"}`)["Rows"].([]any)
	if len(rows) != 2 || tsmDatum(rows[0], 1) != "9.0" || tsmDatum(rows[1], 0) != "h2" {
		t.Fatalf("rows after upsert: %v", rows)
	}
}

func tsmDatum(row any, i int) any {
	d := row.(map[string]any)["Data"].([]any)[i].(map[string]any)
	if v, ok := d["ScalarValue"]; ok {
		return v
	}
	return nil
}

func TestTimestreamWriteStage0PartitionKeyAndMagneticStore(t *testing.T) {
	srv, ts := newTimestreamTestServer(t)
	if _, err := srv.s3.CreateBucket("rejects"); err != nil {
		t.Fatal(err)
	}
	tsmOK(t, ts, "CreateDatabase", `{"DatabaseName":"db1"}`)
	tsmOK(t, ts, "CreateTable", `{"DatabaseName":"db1","TableName":"tbl1","Schema":{"CompositePartitionKey":[{"Type":"DIMENSION","Name":"host","EnforcementInRecord":"REQUIRED"}]},"MagneticStoreWriteProperties":{"EnableMagneticStoreWrites":true,"MagneticStoreRejectedDataLocation":{"S3Configuration":{"BucketName":"rejects"}}}}`)
	old := fmt.Sprint(time.Now().Add(-48 * time.Hour).UnixMilli())
	out := tsmOK(t, ts, "WriteRecords", `{"DatabaseName":"db1","TableName":"tbl1","Records":[{"Dimensions":[{"Name":"host","Value":"h1"}],"MeasureName":"cpu","MeasureValue":"1","Time":"`+old+`"}]}`)
	if out["RecordsIngested"].(map[string]any)["MagneticStore"] != float64(1) {
		t.Fatalf("magnetic write: %v", out)
	}
	_, rej := tsmFail(t, ts, "WriteRecords", `{"DatabaseName":"db1","TableName":"tbl1","Records":[{"Dimensions":[{"Name":"region","Value":"us"}],"MeasureName":"cpu","MeasureValue":"1","Time":"`+tsmNowMillis()+`"}]}`)
	if !strings.Contains(fmt.Sprint(rej["RejectedRecords"]), "partition key") {
		t.Fatalf("missing required partition key: %v", rej)
	}
}
