package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type dynamodbstreamsRecord struct {
	EventName string `json:"eventName"`
	Dynamodb  struct {
		Keys           map[string]any
		NewImage       map[string]any
		OldImage       map[string]any
		SequenceNumber string
		SizeBytes      int64
		StreamViewType string
	} `json:"dynamodb"`
}

type dynamodbstreamsRecordsOutput struct {
	Records           []dynamodbstreamsRecord
	NextShardIterator *string
}

// dynamodbstreamsCall sends a Streams action, asserts the status and decodes
// the body into out (when out is non-nil). It returns the raw body.
func dynamodbstreamsCall(t *testing.T, ts *httptest.Server, action, body string, status int, out any) string {
	t.Helper()
	resp := dynamodbstreamsRequest(t, ts, action, body)
	raw := string(mustBody(t, resp))
	if resp.StatusCode != status {
		t.Fatalf("%s: expected %d, got %d: %s", action, status, resp.StatusCode, raw)
	}
	if out != nil {
		if err := json.Unmarshal([]byte(raw), out); err != nil {
			t.Fatalf("%s: decode %s: %v", action, raw, err)
		}
	}
	return raw
}

func dynamodbCall(t *testing.T, ts *httptest.Server, action, body string, status int) string {
	t.Helper()
	resp := dynamodbRequest(t, ts, action, []byte(body))
	raw := string(mustBody(t, resp))
	if resp.StatusCode != status {
		t.Fatalf("%s: expected %d, got %d: %s", action, status, resp.StatusCode, raw)
	}
	return raw
}

func dynamodbstreamsIterator(t *testing.T, ts *httptest.Server, arn, shard, kind, seq string) string {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"StreamArn": arn, "ShardId": shard, "ShardIteratorType": kind, "SequenceNumber": seq})
	if seq == "" {
		body, _ = json.Marshal(map[string]string{"StreamArn": arn, "ShardId": shard, "ShardIteratorType": kind})
	}
	var out struct{ ShardIterator string }
	dynamodbstreamsCall(t, ts, "GetShardIterator", string(body), http.StatusOK, &out)
	return out.ShardIterator
}

func dynamodbstreamsRead(t *testing.T, ts *httptest.Server, iterator string) dynamodbstreamsRecordsOutput {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"ShardIterator": iterator})
	var out dynamodbstreamsRecordsOutput
	dynamodbstreamsCall(t, ts, "GetRecords", string(body), http.StatusOK, &out)
	return out
}

const dynamodbstreamsTableBody = `{"TableName":"orders","AttributeDefinitions":[{"AttributeName":"pk","AttributeType":"S"},{"AttributeName":"sk","AttributeType":"S"}],"KeySchema":[{"AttributeName":"pk","KeyType":"HASH"},{"AttributeName":"sk","KeyType":"RANGE"}],"BillingMode":"PAY_PER_REQUEST","StreamSpecification":{"StreamEnabled":true,"StreamViewType":"NEW_AND_OLD_IMAGES"}}`

func TestDynamoDBStreamsStage1ChangeRecords(t *testing.T) {
	ts := newDynamoDBStreamsTestServer(t)

	var created struct {
		TableDescription struct {
			LatestStreamArn     string
			LatestStreamLabel   string
			StreamSpecification struct {
				StreamEnabled  bool
				StreamViewType string
			}
		}
	}
	if err := json.Unmarshal([]byte(dynamodbCall(t, ts, "CreateTable", dynamodbstreamsTableBody, http.StatusOK)), &created); err != nil {
		t.Fatal(err)
	}
	arn := created.TableDescription.LatestStreamArn
	if !strings.HasPrefix(arn, "arn:aws:dynamodb:us-east-1:123456789012:table/orders/stream/") || !strings.HasSuffix(arn, created.TableDescription.LatestStreamLabel) {
		t.Fatalf("unexpected stream ARN %q / label %q", arn, created.TableDescription.LatestStreamLabel)
	}
	if spec := created.TableDescription.StreamSpecification; !spec.StreamEnabled || spec.StreamViewType != "NEW_AND_OLD_IMAGES" {
		t.Fatalf("unexpected StreamSpecification %+v", spec)
	}
	if body := dynamodbCall(t, ts, "DescribeTable", `{"TableName":"orders"}`, http.StatusOK); !strings.Contains(body, arn) {
		t.Fatalf("DescribeTable is missing LatestStreamArn: %s", body)
	}

	var listed struct {
		Streams []struct{ StreamArn, StreamLabel, TableName string }
	}
	dynamodbstreamsCall(t, ts, "ListStreams", `{"TableName":"orders"}`, http.StatusOK, &listed)
	if len(listed.Streams) != 1 || listed.Streams[0].StreamArn != arn || listed.Streams[0].TableName != "orders" {
		t.Fatalf("unexpected ListStreams %+v", listed)
	}

	var described struct {
		StreamDescription struct {
			StreamStatus, StreamViewType, TableName string
			Shards                                  []struct {
				ShardId             string
				SequenceNumberRange struct{ StartingSequenceNumber, EndingSequenceNumber string }
			}
		}
	}
	dynamodbstreamsCall(t, ts, "DescribeStream", `{"StreamArn":"`+arn+`"}`, http.StatusOK, &described)
	desc := described.StreamDescription
	if desc.StreamStatus != "ENABLED" || desc.StreamViewType != "NEW_AND_OLD_IMAGES" || len(desc.Shards) != 1 || desc.Shards[0].SequenceNumberRange.EndingSequenceNumber != "" {
		t.Fatalf("unexpected DescribeStream %+v", desc)
	}
	shard := desc.Shards[0].ShardId
	start := desc.Shards[0].SequenceNumberRange.StartingSequenceNumber
	horizon := dynamodbstreamsIterator(t, ts, arn, shard, "TRIM_HORIZON", "")
	atStart := dynamodbstreamsIterator(t, ts, arn, shard, "AT_SEQUENCE_NUMBER", start)

	item := `{"TableName":"orders","Item":{"pk":{"S":"a"},"sk":{"S":"1"},"qty":{"N":"1"}}}`
	dynamodbCall(t, ts, "PutItem", item, http.StatusOK)
	dynamodbCall(t, ts, "PutItem", item, http.StatusOK) // unchanged: no record
	dynamodbCall(t, ts, "UpdateItem", `{"TableName":"orders","Key":{"pk":{"S":"a"},"sk":{"S":"1"}},"UpdateExpression":"SET qty = :q","ExpressionAttributeValues":{":q":{"N":"2"}}}`, http.StatusOK)
	dynamodbCall(t, ts, "DeleteItem", `{"TableName":"orders","Key":{"pk":{"S":"a"},"sk":{"S":"1"}}}`, http.StatusOK)
	dynamodbCall(t, ts, "DeleteItem", `{"TableName":"orders","Key":{"pk":{"S":"a"},"sk":{"S":"1"}}}`, http.StatusOK) // gone: no record
	dynamodbCall(t, ts, "BatchWriteItem", `{"RequestItems":{"orders":[{"PutRequest":{"Item":{"pk":{"S":"b"},"sk":{"S":"1"}}}}]}}`, http.StatusOK)
	dynamodbCall(t, ts, "TransactWriteItems", `{"TransactItems":[{"Delete":{"TableName":"orders","Key":{"pk":{"S":"b"},"sk":{"S":"1"}}}}]}`, http.StatusOK)

	got := dynamodbstreamsRead(t, ts, horizon)
	names := []string{}
	for _, rec := range got.Records {
		names = append(names, rec.EventName)
	}
	if strings.Join(names, ",") != "INSERT,MODIFY,REMOVE,INSERT,REMOVE" {
		t.Fatalf("unexpected events %v", names)
	}
	if got.NextShardIterator == nil {
		t.Fatalf("an open shard must return NextShardIterator")
	}
	insert, modify, remove := got.Records[0].Dynamodb, got.Records[1].Dynamodb, got.Records[2].Dynamodb
	if insert.OldImage != nil || insert.NewImage["qty"] == nil || len(insert.Keys) != 2 || insert.SizeBytes <= 0 {
		t.Fatalf("unexpected INSERT %+v", insert)
	}
	if modify.OldImage["qty"].(map[string]any)["N"] != "1" || modify.NewImage["qty"].(map[string]any)["N"] != "2" {
		t.Fatalf("unexpected MODIFY images %+v", modify)
	}
	if remove.NewImage != nil || remove.OldImage == nil {
		t.Fatalf("unexpected REMOVE images %+v", remove)
	}
	if !(start < insert.SequenceNumber && insert.SequenceNumber < modify.SequenceNumber) {
		t.Fatalf("sequence numbers must increase: start %s, %s, %s", start, insert.SequenceNumber, modify.SequenceNumber)
	}
	if len(dynamodbstreamsRead(t, ts, atStart).Records) != 5 {
		t.Fatalf("AT_SEQUENCE_NUMBER of the shard start must read every record")
	}

	at := dynamodbstreamsRead(t, ts, dynamodbstreamsIterator(t, ts, arn, shard, "AT_SEQUENCE_NUMBER", modify.SequenceNumber))
	after := dynamodbstreamsRead(t, ts, dynamodbstreamsIterator(t, ts, arn, shard, "AFTER_SEQUENCE_NUMBER", modify.SequenceNumber))
	if len(at.Records) != 4 || at.Records[0].EventName != "MODIFY" || len(after.Records) != 3 || after.Records[0].EventName != "REMOVE" {
		t.Fatalf("AT/AFTER_SEQUENCE_NUMBER positioned wrongly: at=%d after=%d", len(at.Records), len(after.Records))
	}

	latest := dynamodbstreamsIterator(t, ts, arn, shard, "LATEST", "")
	if len(dynamodbstreamsRead(t, ts, latest).Records) != 0 {
		t.Fatalf("LATEST must skip existing records")
	}
	dynamodbCall(t, ts, "PutItem", `{"TableName":"orders","Item":{"pk":{"S":"c"},"sk":{"S":"1"}}}`, http.StatusOK)
	limited := dynamodbstreamsCall(t, ts, "GetRecords", `{"ShardIterator":"`+horizon+`","Limit":2}`, http.StatusOK, nil)
	if strings.Count(limited, `"eventName"`) != 2 {
		t.Fatalf("Limit 2 must return two records: %s", limited)
	}
	if fresh := dynamodbstreamsRead(t, ts, latest); len(fresh.Records) != 1 || fresh.Records[0].EventName != "INSERT" {
		t.Fatalf("LATEST iterator must see later writes, got %+v", fresh.Records)
	}

	// Disabling closes the shard; reading to its end drops NextShardIterator.
	dynamodbCall(t, ts, "UpdateTable", `{"TableName":"orders","StreamSpecification":{"StreamEnabled":false}}`, http.StatusOK)
	dynamodbstreamsCall(t, ts, "DescribeStream", `{"StreamArn":"`+arn+`"}`, http.StatusOK, &described)
	if described.StreamDescription.StreamStatus != "DISABLED" || described.StreamDescription.Shards[0].SequenceNumberRange.EndingSequenceNumber == "" {
		t.Fatalf("expected a closed shard, got %+v", described.StreamDescription)
	}
	if drained := dynamodbstreamsRead(t, ts, horizon); len(drained.Records) != 6 || drained.NextShardIterator != nil {
		t.Fatalf("a drained closed shard must have no NextShardIterator: %d records, next %v", len(drained.Records), drained.NextShardIterator)
	}

	// Enabling again creates a new stream; the old one stays listed.
	var updated struct {
		TableDescription struct{ LatestStreamArn string }
	}
	if err := json.Unmarshal([]byte(dynamodbCall(t, ts, "UpdateTable", `{"TableName":"orders","StreamSpecification":{"StreamEnabled":true,"StreamViewType":"KEYS_ONLY"}}`, http.StatusOK)), &updated); err != nil {
		t.Fatal(err)
	}
	if updated.TableDescription.LatestStreamArn == "" || updated.TableDescription.LatestStreamArn == arn {
		t.Fatalf("re-enabling must create a new stream ARN, got %q", updated.TableDescription.LatestStreamArn)
	}
	dynamodbstreamsCall(t, ts, "ListStreams", `{}`, http.StatusOK, &listed)
	if len(listed.Streams) != 2 {
		t.Fatalf("expected both streams listed, got %+v", listed.Streams)
	}
	dynamodbstreamsCall(t, ts, "DescribeStream", `{"StreamArn":"`+updated.TableDescription.LatestStreamArn+`"}`, http.StatusOK, &described)
	keysOnly := dynamodbstreamsIterator(t, ts, updated.TableDescription.LatestStreamArn, described.StreamDescription.Shards[0].ShardId, "TRIM_HORIZON", "")
	dynamodbCall(t, ts, "PutItem", `{"TableName":"orders","Item":{"pk":{"S":"d"},"sk":{"S":"1"},"x":{"S":"y"}}}`, http.StatusOK)
	if recs := dynamodbstreamsRead(t, ts, keysOnly).Records; len(recs) != 1 || recs[0].Dynamodb.NewImage != nil || recs[0].Dynamodb.StreamViewType != "KEYS_ONLY" {
		t.Fatalf("KEYS_ONLY must carry keys only: %+v", recs)
	}

	// Deleting the table closes its stream but keeps it readable.
	dynamodbCall(t, ts, "DeleteTable", `{"TableName":"orders"}`, http.StatusOK)
	dynamodbstreamsCall(t, ts, "DescribeStream", `{"StreamArn":"`+updated.TableDescription.LatestStreamArn+`"}`, http.StatusOK, &described)
	if described.StreamDescription.StreamStatus != "DISABLED" {
		t.Fatalf("a deleted table's stream must be DISABLED, got %s", described.StreamDescription.StreamStatus)
	}
}

func TestDynamoDBStreamsStage1Validation(t *testing.T) {
	ts := newDynamoDBStreamsTestServer(t)
	dynamodbCall(t, ts, "CreateTable", strings.Replace(dynamodbstreamsTableBody, `"NEW_AND_OLD_IMAGES"`, `"EVERYTHING"`, 1), http.StatusBadRequest)
	dynamodbCall(t, ts, "CreateTable", strings.Replace(dynamodbstreamsTableBody, `"StreamEnabled":true,`, ``, 1), http.StatusBadRequest)
	if body := dynamodbCall(t, ts, "DescribeTable", `{"TableName":"orders"}`, http.StatusBadRequest); !strings.Contains(body, "ResourceNotFoundException") {
		t.Fatalf("a rejected StreamSpecification must not create the table: %s", body)
	}
	dynamodbCall(t, ts, "CreateTable", dynamodbstreamsTableBody, http.StatusOK)
	if body := dynamodbCall(t, ts, "UpdateTable", `{"TableName":"orders","StreamSpecification":{"StreamEnabled":true,"StreamViewType":"NEW_IMAGE"}}`, http.StatusBadRequest); !strings.Contains(body, "already has an enabled stream") {
		t.Fatalf("enabling twice must fail: %s", body)
	}
	dynamodbCall(t, ts, "UpdateTable", `{"TableName":"orders","StreamSpecification":{"StreamEnabled":false}}`, http.StatusOK)
	dynamodbCall(t, ts, "UpdateTable", `{"TableName":"orders","StreamSpecification":{"StreamEnabled":false}}`, http.StatusBadRequest)

	var listed struct{ Streams []struct{ StreamArn string } }
	dynamodbstreamsCall(t, ts, "ListStreams", `{}`, http.StatusOK, &listed)
	arn := listed.Streams[0].StreamArn

	cases := []struct{ action, body, code string }{
		{"ListStreams", `{"Limit":0}`, "ValidationException"},
		{"ListStreams", `{"Limit":101}`, "ValidationException"},
		{"ListStreams", `{"TableName":"missing"}`, "ResourceNotFoundException"},
		{"DescribeStream", `{}`, "ValidationException"},
		{"DescribeStream", `{"StreamArn":"arn:aws:dynamodb:us-east-1:123456789012:table/x/stream/2020"}`, "ResourceNotFoundException"},
		{"GetShardIterator", `{"StreamArn":"` + arn + `"}`, "ValidationException"},
		{"GetShardIterator", `{"StreamArn":"` + arn + `","ShardId":"shardId-nope","ShardIteratorType":"LATEST"}`, "ResourceNotFoundException"},
		{"GetRecords", `{"ShardIterator":"garbage"}`, "ValidationException"},
		{"GetRecords", `{}`, "ValidationException"},
	}
	for _, tc := range cases {
		if body := dynamodbstreamsCall(t, ts, tc.action, tc.body, http.StatusBadRequest, nil); !strings.Contains(body, tc.code) {
			t.Fatalf("%s %s: expected %s, got %s", tc.action, tc.body, tc.code, body)
		}
	}

	var described struct {
		StreamDescription struct {
			Shards []struct {
				ShardId             string
				SequenceNumberRange struct{ StartingSequenceNumber, EndingSequenceNumber string }
			}
		}
	}
	dynamodbstreamsCall(t, ts, "DescribeStream", `{"StreamArn":"`+arn+`"}`, http.StatusOK, &described)
	shard := described.StreamDescription.Shards[0]
	// Disabled before any write: the empty shard is still closed.
	if rng := shard.SequenceNumberRange; rng.EndingSequenceNumber == "" || rng.EndingSequenceNumber != rng.StartingSequenceNumber {
		t.Fatalf("an empty disabled shard must end at its start, got %+v", rng)
	}
	if empty := dynamodbstreamsRead(t, ts, dynamodbstreamsIterator(t, ts, arn, shard.ShardId, "TRIM_HORIZON", "")); empty.NextShardIterator != nil {
		t.Fatalf("an empty closed shard must have no NextShardIterator")
	}
	iteratorCases := []struct{ kind, seq, code string }{
		{"SIDEWAYS", "", "ValidationException"},
		{"AT_SEQUENCE_NUMBER", "", "ValidationException"},
		{"AT_SEQUENCE_NUMBER", "abc", "ValidationException"},
		{"AT_SEQUENCE_NUMBER", "000000000000000000000", "TrimmedDataAccessException"},
		{"AFTER_SEQUENCE_NUMBER", "999999999999999999999", "ValidationException"},
		{"LATEST", shard.SequenceNumberRange.StartingSequenceNumber, "ValidationException"},
	}
	for _, tc := range iteratorCases {
		body, _ := json.Marshal(map[string]string{"StreamArn": arn, "ShardId": shard.ShardId, "ShardIteratorType": tc.kind, "SequenceNumber": tc.seq})
		if raw := dynamodbstreamsCall(t, ts, "GetShardIterator", string(body), http.StatusBadRequest, nil); !strings.Contains(raw, tc.code) {
			t.Fatalf("%s %q: expected %s, got %s", tc.kind, tc.seq, tc.code, raw)
		}
	}
	iterator := dynamodbstreamsIterator(t, ts, arn, shard.ShardId, "TRIM_HORIZON", "")
	for _, limit := range []string{"0", "1001", `"x"`} {
		if raw := dynamodbstreamsCall(t, ts, "GetRecords", `{"ShardIterator":"`+iterator+`","Limit":`+limit+`}`, http.StatusBadRequest, nil); !strings.Contains(raw, "ValidationException") {
			t.Fatalf("Limit %s: expected ValidationException, got %s", limit, raw)
		}
	}
}

func TestDynamoDBStreamsStage1TableScopeAndRestore(t *testing.T) {
	ts := newDynamoDBStreamsTestServer(t)
	dynamodbCall(t, ts, "CreateTable", dynamodbstreamsTableBody, http.StatusOK)
	dynamodbCall(t, ts, "CreateTable", strings.Replace(dynamodbstreamsTableBody, `"orders"`, `"invoices"`, 1), http.StatusOK)

	var listed struct {
		Streams []struct{ StreamArn, TableName string }
	}
	dynamodbstreamsCall(t, ts, "ListStreams", `{"TableName":"invoices"}`, http.StatusOK, &listed)
	if len(listed.Streams) != 1 || listed.Streams[0].TableName != "invoices" {
		t.Fatalf("ListStreams must filter by TableName, got %+v", listed.Streams)
	}
	var page struct {
		Streams                []struct{ StreamArn string }
		LastEvaluatedStreamArn string
	}
	dynamodbstreamsCall(t, ts, "ListStreams", `{"Limit":1}`, http.StatusOK, &page)
	if len(page.Streams) != 1 || page.LastEvaluatedStreamArn != page.Streams[0].StreamArn {
		t.Fatalf("Limit 1 must return one stream and a LastEvaluatedStreamArn, got %+v", page)
	}
	first := page.Streams[0].StreamArn
	page.Streams, page.LastEvaluatedStreamArn = nil, "" // a missing member doesn't overwrite on decode
	dynamodbstreamsCall(t, ts, "ListStreams", `{"Limit":1,"ExclusiveStartStreamArn":"`+first+`"}`, http.StatusOK, &page)
	if len(page.Streams) != 1 || page.Streams[0].StreamArn == first || page.LastEvaluatedStreamArn != "" {
		t.Fatalf("the second page must hold the other stream and end the listing, got %+v", page)
	}

	// A table restored from a backup starts without the source table's stream.
	var backup struct{ BackupDetails struct{ BackupArn string } }
	if err := json.Unmarshal([]byte(dynamodbCall(t, ts, "CreateBackup", `{"TableName":"orders","BackupName":"b1"}`, http.StatusOK)), &backup); err != nil {
		t.Fatal(err)
	}
	restored := dynamodbCall(t, ts, "RestoreTableFromBackup", `{"TargetTableName":"orders-copy","BackupArn":"`+backup.BackupDetails.BackupArn+`"}`, http.StatusOK)
	if strings.Contains(restored, "LatestStreamArn") || strings.Contains(restored, "StreamSpecification") {
		t.Fatalf("a restored table must not inherit the source stream: %s", restored)
	}
}
