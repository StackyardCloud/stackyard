package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// dynamodbstreamsSetup creates tables with NEW_AND_OLD_IMAGES streams and
// returns each table's stream ARN and shard ID.
func dynamodbstreamsSetup(t *testing.T, ts *httptest.Server, tables ...string) map[string][2]string {
	t.Helper()
	out := map[string][2]string{}
	for _, name := range tables {
		var created struct {
			TableDescription struct{ LatestStreamArn string }
		}
		body := strings.Replace(dynamodbstreamsTableBody, `"orders"`, `"`+name+`"`, 1)
		if err := json.Unmarshal([]byte(dynamodbCall(t, ts, "CreateTable", body, http.StatusOK)), &created); err != nil {
			t.Fatal(err)
		}
		arn := created.TableDescription.LatestStreamArn
		var described struct {
			StreamDescription struct{ Shards []struct{ ShardId string } }
		}
		dynamodbstreamsCall(t, ts, "DescribeStream", `{"StreamArn":"`+arn+`"}`, http.StatusOK, &described)
		out[name] = [2]string{arn, described.StreamDescription.Shards[0].ShardId}
	}
	return out
}

func dynamodbstreamsAll(t *testing.T, ts *httptest.Server, stream [2]string) []dynamodbstreamsRecord {
	t.Helper()
	return dynamodbstreamsRead(t, ts, dynamodbstreamsIterator(t, ts, stream[0], stream[1], "TRIM_HORIZON", "")).Records
}

func dynamodbItemCount(t *testing.T, ts *httptest.Server, table string) int {
	t.Helper()
	var scanned struct{ Count int }
	if err := json.Unmarshal([]byte(dynamodbCall(t, ts, "Scan", `{"TableName":"`+table+`"}`, http.StatusOK)), &scanned); err != nil {
		t.Fatal(err)
	}
	return scanned.Count
}

func TestDynamoDBStreamsStage2FailedWritesLeaveNothing(t *testing.T) {
	ts := newDynamoDBStreamsTestServer(t)
	streams := dynamodbstreamsSetup(t, ts, "orders")

	put := func(pk string) string {
		return `{"Put":{"TableName":"orders","Item":{"pk":{"S":"` + pk + `"},"sk":{"S":"1"}}}}`
	}
	failures := []struct{ action, body, code string }{
		{"TransactWriteItems", `{"TransactItems":[` + put("a") + `,` + put("b") + `,{"ConditionCheck":{"TableName":"orders","Key":{"pk":{"S":"missing"},"sk":{"S":"1"}}}}]}`, "ConditionalCheckFailedException"},
		{"TransactWriteItems", `{"TransactItems":[` + put("a") + `,{"Bogus":{}}]}`, "ValidationException"},
		{"TransactWriteItems", `{"TransactItems":[` + put("a") + `,{"Put":{"TableName":"nope","Item":{"pk":{"S":"x"}}}}]}`, "ResourceNotFoundException"},
		{"TransactWriteItems", `{"TransactItems":[` + put("a") + `,{"Delete":{"TableName":"orders","Key":{"pk":{"S":"a"},"sk":{"S":"1"}}}}]}`, "multiple operations on one item"},
		{"BatchWriteItem", `{"RequestItems":{"orders":[{"PutRequest":{"Item":{"pk":{"S":"a"},"sk":{"S":"1"}}}},{"Bogus":{}}]}}`, "ValidationException"},
		{"BatchWriteItem", `{"RequestItems":{"orders":[{"PutRequest":{"Item":{"pk":{"S":"a"},"sk":{"S":"1"}}}}],"zzz-missing":[{"DeleteRequest":{"Key":{"pk":{"S":"x"}}}}]}}`, "ResourceNotFoundException"},
		{"BatchWriteItem", `{"RequestItems":{"orders":[{"PutRequest":{"Item":{"pk":{"S":"a"},"sk":{"S":"1"}}}},{"DeleteRequest":{"Key":{"pk":{"S":"a"},"sk":{"S":"1"}}}}]}}`, "contains duplicates"},
	}
	for _, tc := range failures {
		if body := dynamodbCall(t, ts, tc.action, tc.body, http.StatusBadRequest); !strings.Contains(body, tc.code) {
			t.Fatalf("%s: expected %s, got %s", tc.action, tc.code, body)
		}
		if n := dynamodbItemCount(t, ts, "orders"); n != 0 {
			t.Fatalf("%s %s: a failed request must write nothing, found %d items", tc.action, tc.code, n)
		}
		if recs := dynamodbstreamsAll(t, ts, streams["orders"]); len(recs) != 0 {
			t.Fatalf("%s %s: a failed request must record nothing, found %d records", tc.action, tc.code, len(recs))
		}
	}
}

func TestDynamoDBStreamsStage2NoOpWritesAndOrdering(t *testing.T) {
	ts := newDynamoDBStreamsTestServer(t)
	streams := dynamodbstreamsSetup(t, ts, "orders", "invoices")

	dynamodbCall(t, ts, "PutItem", `{"TableName":"orders","Item":{"pk":{"S":"a"},"sk":{"S":"1"},"tags":{"SS":["x","y"]},"qty":{"N":"1"}}}`, http.StatusOK)
	// Same item with the set reordered, and an update to the value it already has.
	dynamodbCall(t, ts, "PutItem", `{"TableName":"orders","Item":{"qty":{"N":"1"},"tags":{"SS":["y","x"]},"sk":{"S":"1"},"pk":{"S":"a"}}}`, http.StatusOK)
	dynamodbCall(t, ts, "UpdateItem", `{"TableName":"orders","Key":{"pk":{"S":"a"},"sk":{"S":"1"}},"UpdateExpression":"SET qty = :q","ExpressionAttributeValues":{":q":{"N":"1"}}}`, http.StatusOK)
	if recs := dynamodbstreamsAll(t, ts, streams["orders"]); len(recs) != 1 || recs[0].EventName != "INSERT" {
		t.Fatalf("no-op writes must not record, got %+v", recs)
	}

	// A transaction across two tables records in entry order.
	dynamodbCall(t, ts, "TransactWriteItems", `{"TransactItems":[
		{"Put":{"TableName":"invoices","Item":{"pk":{"S":"i1"},"sk":{"S":"1"}}}},
		{"Put":{"TableName":"orders","Item":{"pk":{"S":"b"},"sk":{"S":"1"}}}},
		{"Put":{"TableName":"invoices","Item":{"pk":{"S":"i2"},"sk":{"S":"1"}}}}]}`, http.StatusOK)
	inv, ord := dynamodbstreamsAll(t, ts, streams["invoices"]), dynamodbstreamsAll(t, ts, streams["orders"])
	if len(inv) != 2 || len(ord) != 2 {
		t.Fatalf("expected 2 invoice and 2 order records, got %d and %d", len(inv), len(ord))
	}
	if !(inv[0].Dynamodb.SequenceNumber < ord[1].Dynamodb.SequenceNumber && ord[1].Dynamodb.SequenceNumber < inv[1].Dynamodb.SequenceNumber) {
		t.Fatalf("sequence numbers must follow entry order: i1=%s b=%s i2=%s", inv[0].Dynamodb.SequenceNumber, ord[1].Dynamodb.SequenceNumber, inv[1].Dynamodb.SequenceNumber)
	}
}

func TestDynamoDBStreamsStage2IteratorsAndRetention(t *testing.T) {
	ts := newDynamoDBStreamsTestServer(t)
	stream := dynamodbstreamsSetup(t, ts, "orders")["orders"]
	write := func(pk string) {
		dynamodbCall(t, ts, "PutItem", `{"TableName":"orders","Item":{"pk":{"S":"`+pk+`"},"sk":{"S":"1"}}}`, http.StatusOK)
	}

	// Paging with Limit 1 while writes land between pages: no gaps, no repeats.
	write("p1")
	iterator := dynamodbstreamsIterator(t, ts, stream[0], stream[1], "TRIM_HORIZON", "")
	seen := []string{}
	for _, next := range []string{"p2", "p3", ""} {
		var page dynamodbstreamsRecordsOutput
		dynamodbstreamsCall(t, ts, "GetRecords", `{"ShardIterator":"`+iterator+`","Limit":1}`, http.StatusOK, &page)
		for _, rec := range page.Records {
			seen = append(seen, rec.Dynamodb.Keys["pk"].(map[string]any)["S"].(string))
		}
		if page.NextShardIterator == nil {
			t.Fatalf("an open shard must keep returning an iterator")
		}
		iterator = *page.NextShardIterator
		if next != "" {
			write(next)
		}
	}
	if strings.Join(seen, ",") != "p1,p2,p3" {
		t.Fatalf("paged reads must see every write once in order, got %v", seen)
	}

	// AFTER the last record: an empty batch with a usable iterator.
	recs := dynamodbstreamsAll(t, ts, stream)
	last := recs[len(recs)-1].Dynamodb.SequenceNumber
	after := dynamodbstreamsRead(t, ts, dynamodbstreamsIterator(t, ts, stream[0], stream[1], "AFTER_SEQUENCE_NUMBER", last))
	if len(after.Records) != 0 || after.NextShardIterator == nil {
		t.Fatalf("AFTER the last record must be empty with a NextShardIterator, got %+v", after)
	}
	write("p4")
	if more := dynamodbstreamsRead(t, ts, *after.NextShardIterator); len(more.Records) != 1 {
		t.Fatalf("the iterator after the last record must see the next write, got %+v", more.Records)
	}

	// After disabling, DescribeTable drops StreamSpecification but keeps the latest ARN.
	dynamodbCall(t, ts, "UpdateTable", `{"TableName":"orders","StreamSpecification":{"StreamEnabled":false}}`, http.StatusOK)
	if body := dynamodbCall(t, ts, "DescribeTable", `{"TableName":"orders"}`, http.StatusOK); strings.Contains(body, "StreamSpecification") || !strings.Contains(body, stream[0]) {
		t.Fatalf("a disabled stream keeps LatestStreamArn without StreamSpecification: %s", body)
	}

	// A deleted table's stream stays readable and listed; recreating gives a new stream.
	dynamodbCall(t, ts, "DeleteTable", `{"TableName":"orders"}`, http.StatusOK)
	dynamodbstreamsCall(t, ts, "DescribeStream", `{"StreamArn":"`+stream[0]+`"}`, http.StatusOK, nil)
	if got := dynamodbstreamsAll(t, ts, stream); len(got) != 4 {
		t.Fatalf("a deleted table's records stay readable, got %d", len(got))
	}
	recreated := dynamodbstreamsSetup(t, ts, "orders")["orders"]
	if recreated[0] == stream[0] {
		t.Fatalf("a recreated table must get a new stream ARN")
	}
	var listed struct{ Streams []struct{ StreamArn string } }
	dynamodbstreamsCall(t, ts, "ListStreams", `{"TableName":"orders"}`, http.StatusOK, &listed)
	if len(listed.Streams) != 2 {
		t.Fatalf("ListStreams must include the deleted table's stream, got %+v", listed.Streams)
	}
	dynamodbCall(t, ts, "DeleteTable", `{"TableName":"orders"}`, http.StatusOK)
	dynamodbstreamsCall(t, ts, "ListStreams", `{"TableName":"orders"}`, http.StatusOK, &listed)
	if len(listed.Streams) != 2 {
		t.Fatalf("ListStreams by the name of a deleted table must list its retained streams, got %+v", listed.Streams)
	}
}

func TestDynamoDBStreamsStage2PlainDynamoDBStillRouted(t *testing.T) {
	ts := newDynamoDBStreamsTestServer(t)
	dynamodbCall(t, ts, "CreateTable", strings.Replace(dynamodbstreamsTableBody, `,"StreamSpecification":{"StreamEnabled":true,"StreamViewType":"NEW_AND_OLD_IMAGES"}`, ``, 1), http.StatusOK)
	dynamodbCall(t, ts, "PutItem", `{"TableName":"orders","Item":{"pk":{"S":"a"},"sk":{"S":"1"}}}`, http.StatusOK)
	if body := dynamodbCall(t, ts, "GetItem", `{"TableName":"orders","Key":{"pk":{"S":"a"},"sk":{"S":"1"}}}`, http.StatusOK); !strings.Contains(body, `"Item"`) {
		t.Fatalf("DynamoDB_20120810.PutItem must reach the DynamoDB router: %s", body)
	}
	if body := dynamodbCall(t, ts, "DescribeTable", `{"TableName":"orders"}`, http.StatusOK); strings.Contains(body, "LatestStreamArn") {
		t.Fatalf("a table without a stream must not report one: %s", body)
	}
}
