package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestS3TablesStage10MigrationsAndLogging(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()
	bucketArn := "arn:aws:s3tables:us-east-1:123456789012:bucket/demo-bucket"
	jsonHdr := map[string]string{"Content-Type": "application/json"}
	call := func(method, path string, body string, want int) map[string]any {
		t.Helper()
		var b []byte
		if body != "" {
			b = []byte(body)
		}
		resp := signedRequestWithService(t, method, ts.URL+path, b, jsonHdr, "s3tables")
		assertStatus(t, resp, want)
		out := map[string]any{}
		_ = json.Unmarshal(mustBody(t, resp), &out)
		return out
	}
	start := `{"roleARN":"arn:aws:iam::123456789012:role/r","destinationTable":{"tableBucketARN":"` + bucketArn +
		`","namespace":"analytics","name":"moved"},"sourceType":"glueTable","source":{"glueTable":{"catalogId":"123456789012",` +
		`"databaseName":"db","tableName":"t"}},"reviewMigrationPlan":true,"migrationOutput":{"bucket":"reports"}}`

	started := call(http.MethodPost, "/start-table-migration", start, http.StatusOK)
	id, _ := started["migrationId"].(string)
	if id == "" || started["state"] != "pendingReview" {
		t.Fatalf("start: %v", started)
	}
	call(http.MethodPost, "/start-table-migration", start, http.StatusConflict)
	call(http.MethodPost, "/start-table-migration", `{"roleARN":"r"}`, http.StatusBadRequest)

	got := call(http.MethodGet, "/get-table-migration?migrationId="+id, "", http.StatusOK)
	tableARN, _ := got["tableARN"].(string)
	out, _ := got["migrationOutput"].(map[string]any)
	if tableARN == "" || got["type"] != "start" || out["bucket"] != "reports" || out["objectMigrationReport"] == nil {
		t.Fatalf("get: %v", got)
	}
	call(http.MethodGet, "/get-table-migration?migrationId=nope", "", http.StatusNotFound)

	upd := call(http.MethodPost, "/update-table-migration?migrationId="+id, `{"requestedState":"queued"}`, http.StatusOK)
	if upd["state"] != "queued" {
		t.Fatalf("update: %v", upd)
	}
	call(http.MethodPost, "/complete-table-migration?tableARN="+url.QueryEscape(tableARN), "", http.StatusOK)
	call(http.MethodPost, "/update-table-migration?migrationId="+id, `{"requestedState":"cancelled"}`, http.StatusConflict)

	sync := call(http.MethodPost, "/sync-table-migration",
		`{"roleARN":"r","tableARN":"`+tableARN+`","migrationOutput":{"bucket":"reports"}}`, http.StatusOK)
	if sync["state"] != "queued" {
		t.Fatalf("sync: %v", sync)
	}
	list := call(http.MethodGet, "/list-table-migrations?maxResults=1", "", http.StatusOK)
	if items, _ := list["migrations"].([]any); len(items) != 1 || list["nextToken"] != "1" {
		t.Fatalf("list: %v", list)
	}

	logPath := "/buckets/" + url.PathEscape(bucketArn) + "/logging"
	if st := call(http.MethodGet, logPath, "", http.StatusOK); st["tableBucketLoggingStatus"] == nil {
		t.Fatalf("logging default: %v", st)
	}
	call(http.MethodPut, logPath, `{"tableBucketLoggingStatus":{"loggingEnabled":{"targetBucket":"logs","targetPrefix":"p/",`+
		`"targetObjectKeyFormat":{"partitionedPrefix":{"partitionDateSource":"EventTime"}}}}}`, http.StatusOK)
	st := call(http.MethodGet, logPath, "", http.StatusOK)
	if le, _ := st["tableBucketLoggingStatus"].(map[string]any)["loggingEnabled"].(map[string]any); le["targetBucket"] != "logs" {
		t.Fatalf("logging: %v", st)
	}

	call(http.MethodPut, "/table-replication/disassociate?tableArn="+url.QueryEscape(tableARN), "", http.StatusOK)
	call(http.MethodPut, "/table-replication/disassociate?tableArn=missing", "", http.StatusNotFound)
}
