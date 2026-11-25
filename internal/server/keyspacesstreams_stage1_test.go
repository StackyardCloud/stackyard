package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func kssKeyspaces(t *testing.T, ts *httptest.Server, op, body string) (int, map[string]any) {
	t.Helper()
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte(body), map[string]string{
		"Content-Type": kssContentType, "X-Amz-Target": "KeyspacesService." + op,
	}, "cassandra")
	out := map[string]any{}
	_ = json.Unmarshal(mustBody(t, resp), &out)
	return resp.StatusCode, out
}

func kssOK(t *testing.T, ts *httptest.Server, op, body string) map[string]any {
	t.Helper()
	status, out := kssCall(t, ts, op, body)
	if status != http.StatusOK {
		t.Fatalf("%s: %d %v", op, status, out)
	}
	return out
}

func kssErrCode(t *testing.T, ts *httptest.Server, op, body, wantType string) map[string]any {
	t.Helper()
	status, out := kssCall(t, ts, op, body)
	if status != http.StatusBadRequest || out["__type"] != wantType || out["message"] == "" {
		t.Fatalf("%s: got %d %v, want %s", op, status, out, wantType)
	}
	return out
}

const kssSchema = `"schemaDefinition":{"allColumns":[{"name":"id","type":"text"}],"partitionKeys":[{"name":"id"}]}`

func TestKeyspacesStreamsStage1CDCStreams(t *testing.T) {
	_, ts := iotdpServer(t)
	kssKeyspaces(t, ts, "CreateKeyspace", `{"keyspaceName":"plant"}`)
	if s, out := kssKeyspaces(t, ts, "CreateTable", `{"keyspaceName":"plant","tableName":"readings",`+kssSchema+`,"cdcSpecification":{"status":"ENABLED","viewType":"NEW_IMAGE"}}`); s != http.StatusOK {
		t.Fatalf("CreateTable: %d %v", s, out)
	}
	kssKeyspaces(t, ts, "CreateTable", `{"keyspaceName":"plant","tableName":"plain",`+kssSchema+`}`)
	if s, out := kssKeyspaces(t, ts, "CreateTable", `{"keyspaceName":"plant","tableName":"bad",`+kssSchema+`,"cdcSpecification":{"status":"ON"}}`); s != http.StatusBadRequest || out["__type"] != "ValidationException" {
		t.Fatalf("bad cdc status: %d %v", s, out)
	}
	_, table := kssKeyspaces(t, ts, "GetTable", `{"keyspaceName":"plant","tableName":"readings"}`)
	arn, _ := table["latestStreamArn"].(string)
	if !strings.HasPrefix(arn, "arn:aws:cassandra:us-east-1:123456789012:/keyspace/plant/table/readings/stream/") ||
		table["cdcSpecification"].(map[string]any)["viewType"] != "NEW_IMAGE" {
		t.Fatalf("GetTable: %v", table)
	}

	streams := kssOK(t, ts, "ListStreams", `{"keyspaceName":"plant"}`)["streams"].([]any)
	if len(streams) != 1 || streams[0].(map[string]any)["streamArn"] != arn {
		t.Fatalf("ListStreams: %v", streams)
	}
	stream := kssOK(t, ts, "GetStream", `{"streamArn":"`+arn+`"}`)
	if stream["streamStatus"] != "ENABLED" || stream["streamViewType"] != "NEW_IMAGE" || stream["tableName"] != "readings" {
		t.Fatalf("GetStream: %v", stream)
	}
	shard := stream["shards"].([]any)[0].(map[string]any)
	shardID := shard["shardId"].(string)
	if _, closed := shard["sequenceNumberRange"].(map[string]any)["endingSequenceNumber"]; closed {
		t.Fatalf("open shard has an end: %v", shard)
	}
	if kids := kssOK(t, ts, "GetStream", `{"streamArn":"`+arn+`","shardFilter":{"type":"CHILD_SHARDS","shardId":"`+shardID+`"}}`)["shards"].([]any); len(kids) != 0 {
		t.Fatalf("child shards: %v", kids)
	}
	kssErrCode(t, ts, "GetStream", `{"streamArn":"`+arn+`x"}`, "ResourceNotFoundException")

	it := kssOK(t, ts, "GetShardIterator", `{"streamArn":"`+arn+`","shardId":"`+shardID+`","shardIteratorType":"TRIM_HORIZON"}`)["shardIterator"].(string)
	recs := kssOK(t, ts, "GetRecords", `{"shardIterator":"`+it+`","maxResults":10}`)
	if len(recs["changeRecords"].([]any)) != 0 || recs["nextShardIterator"] == nil || recs["iteratorDescription"].(map[string]any)["iteratorPosition"] != "AT_TIP" {
		t.Fatalf("GetRecords: %v", recs)
	}
	start := shard["sequenceNumberRange"].(map[string]any)["startingSequenceNumber"].(string)
	kssOK(t, ts, "GetShardIterator", `{"streamArn":"`+arn+`","shardId":"`+shardID+`","shardIteratorType":"AT_SEQUENCE_NUMBER","sequenceNumber":"`+start+`"}`)
	if out := kssErrCode(t, ts, "GetShardIterator", `{"streamArn":"`+arn+`","shardId":"`+shardID+`","shardIteratorType":"AFTER_SEQUENCE_NUMBER","sequenceNumber":"000000000000000000001"}`, "ValidationException"); out["errorCode"] != "TrimmedDataAccess" {
		t.Fatalf("trimmed: %v", out)
	}
	kssErrCode(t, ts, "GetShardIterator", `{"streamArn":"`+arn+`","shardId":"shardId-00000000000000000000-00000000","shardIteratorType":"LATEST"}`, "ResourceNotFoundException")
	if out := kssErrCode(t, ts, "GetRecords", `{"shardIterator":"garbage!"}`, "ValidationException"); out["errorCode"] != "InvalidFormat" {
		t.Fatalf("bad iterator: %v", out)
	}
	expired := kssIterator{arn: arn, shard: shardID, position: start, expires: time.Now().Add(-time.Minute).Unix()}.encode()
	if out := kssErrCode(t, ts, "GetRecords", `{"shardIterator":"`+expired+`"}`, "ValidationException"); out["errorCode"] != "ExpiredIterator" {
		t.Fatalf("expired iterator: %v", out)
	}

	// The view type of an enabled stream can't change; disabling closes the shard.
	if s, out := kssKeyspaces(t, ts, "UpdateTable", `{"keyspaceName":"plant","tableName":"readings","cdcSpecification":{"status":"ENABLED","viewType":"KEYS_ONLY"}}`); s != http.StatusBadRequest || out["__type"] != "ValidationException" {
		t.Fatalf("view type change: %d %v", s, out)
	}
	kssKeyspaces(t, ts, "UpdateTable", `{"keyspaceName":"plant","tableName":"readings","cdcSpecification":{"status":"DISABLED"}}`)
	stream = kssOK(t, ts, "GetStream", `{"streamArn":"`+arn+`"}`)
	shard = stream["shards"].([]any)[0].(map[string]any)
	if stream["streamStatus"] != "DISABLED" || shard["sequenceNumberRange"].(map[string]any)["endingSequenceNumber"] == nil {
		t.Fatalf("disabled stream: %v", stream)
	}
	if recs := kssOK(t, ts, "GetRecords", `{"shardIterator":"`+it+`"}`); recs["nextShardIterator"] != nil {
		t.Fatalf("closed shard handed out an iterator: %v", recs)
	}
	// Re-enabling opens a second stream; both are listed.
	kssKeyspaces(t, ts, "UpdateTable", `{"keyspaceName":"plant","tableName":"readings","cdcSpecification":{"status":"ENABLED"}}`)
	_, table = kssKeyspaces(t, ts, "GetTable", `{"keyspaceName":"plant","tableName":"readings"}`)
	if table["latestStreamArn"] == arn || table["cdcSpecification"].(map[string]any)["viewType"] != "NEW_AND_OLD_IMAGES" {
		t.Fatalf("re-enabled: %v", table)
	}
	page := kssOK(t, ts, "ListStreams", `{"keyspaceName":"plant","tableName":"readings","maxResults":1}`)
	tok, _ := page["nextToken"].(string)
	if len(page["streams"].([]any)) != 1 || len(tok) < 80 {
		t.Fatalf("page 1: %v", page)
	}
	page = kssOK(t, ts, "ListStreams", `{"keyspaceName":"plant","tableName":"readings","nextToken":"`+tok+`"}`)
	if len(page["streams"].([]any)) != 1 || page["nextToken"] != nil {
		t.Fatalf("page 2: %v", page)
	}
	kssErrCode(t, ts, "ListStreams", `{"nextToken":"`+strings.Repeat("z", 80)+`"}`, "ValidationException")
	// Dropping the table drops its streams.
	kssKeyspaces(t, ts, "DeleteTable", `{"keyspaceName":"plant","tableName":"readings"}`)
	kssErrCode(t, ts, "GetStream", `{"streamArn":"`+arn+`"}`, "ResourceNotFoundException")
}
