package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

// A DynamoDB Streams walkthrough: create a table with a stream, write items,
// then read the INSERT / MODIFY / REMOVE change records back.
func main() {
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	region := getenv("AWS_REGION", "us-east-1")
	table := getenv("STACKYARD_TABLE_NAME", "orders-with-stream")
	ctx := context.Background()
	creds := credentials.NewStaticCredentialsProvider(
		getenv("AWS_ACCESS_KEY_ID", "stackyard"),
		getenv("AWS_SECRET_ACCESS_KEY", "stackyard"),
		"",
	)
	c := client{endpoint: endpoint, region: region, creds: creds}
	fmt.Printf("Stackyard DynamoDB Streams client using %s\n", endpoint)

	var created struct {
		TableDescription struct{ LatestStreamArn string }
	}
	c.call(ctx, "DynamoDB_20120810", "CreateTable", map[string]any{
		"TableName":            table,
		"AttributeDefinitions": []map[string]string{{"AttributeName": "id", "AttributeType": "S"}},
		"KeySchema":            []map[string]string{{"AttributeName": "id", "KeyType": "HASH"}},
		"BillingMode":          "PAY_PER_REQUEST",
		"StreamSpecification":  map[string]any{"StreamEnabled": true, "StreamViewType": "NEW_AND_OLD_IMAGES"},
	}, &created)
	streamArn := created.TableDescription.LatestStreamArn
	logf("table %s streams to %s", table, streamArn)

	key := map[string]any{"id": map[string]string{"S": "order-1"}}
	c.call(ctx, "DynamoDB_20120810", "PutItem", map[string]any{
		"TableName": table,
		"Item":      map[string]any{"id": map[string]string{"S": "order-1"}, "status": map[string]string{"S": "NEW"}},
	}, nil)
	c.call(ctx, "DynamoDB_20120810", "UpdateItem", map[string]any{
		"TableName":                 table,
		"Key":                       key,
		"UpdateExpression":          "SET #s = :s",
		"ExpressionAttributeNames":  map[string]string{"#s": "status"},
		"ExpressionAttributeValues": map[string]any{":s": map[string]string{"S": "SHIPPED"}},
	}, nil)
	c.call(ctx, "DynamoDB_20120810", "DeleteItem", map[string]any{"TableName": table, "Key": key}, nil)

	var listed struct{ Streams []struct{ StreamArn string } }
	c.call(ctx, "DynamoDBStreams_20120810", "ListStreams", map[string]any{"TableName": table}, &listed)
	if len(listed.Streams) != 1 || listed.Streams[0].StreamArn != streamArn {
		exitf("ListStreams: expected %s, got %+v", streamArn, listed.Streams)
	}

	var described struct {
		StreamDescription struct {
			StreamStatus string
			Shards       []struct{ ShardId string }
		}
	}
	c.call(ctx, "DynamoDBStreams_20120810", "DescribeStream", map[string]any{"StreamArn": streamArn}, &described)
	if len(described.StreamDescription.Shards) == 0 {
		exitf("DescribeStream returned no shards")
	}
	shardID := described.StreamDescription.Shards[0].ShardId
	logf("stream %s, shard %s", described.StreamDescription.StreamStatus, shardID)

	var iterator struct{ ShardIterator string }
	c.call(ctx, "DynamoDBStreams_20120810", "GetShardIterator", map[string]any{
		"StreamArn": streamArn, "ShardId": shardID, "ShardIteratorType": "TRIM_HORIZON",
	}, &iterator)

	var records struct {
		Records []struct {
			EventName string `json:"eventName"`
			Dynamodb  struct {
				SequenceNumber string
				NewImage       map[string]any
				OldImage       map[string]any
			} `json:"dynamodb"`
		}
	}
	c.call(ctx, "DynamoDBStreams_20120810", "GetRecords", map[string]any{"ShardIterator": iterator.ShardIterator}, &records)
	events := []string{}
	for _, rec := range records.Records {
		events = append(events, rec.EventName)
		logf("  %s seq=%s new=%v old=%v", rec.EventName, rec.Dynamodb.SequenceNumber, rec.Dynamodb.NewImage, rec.Dynamodb.OldImage)
	}
	if got := strings.Join(events, ","); got != "INSERT,MODIFY,REMOVE" {
		exitf("expected INSERT,MODIFY,REMOVE, got %s", got)
	}

	c.call(ctx, "DynamoDB_20120810", "DeleteTable", map[string]any{"TableName": table}, nil)
	logf("DynamoDB Streams example completed")
}

type client struct {
	endpoint, region string
	creds            aws.CredentialsProvider
}

// call sends one awsJson1_0 request signed for "dynamodb" (Streams shares the
// DynamoDB signing name), exits on a non-2xx reply, and decodes into out.
func (c client) call(ctx context.Context, targetPrefix, action string, payload, out any) {
	body, err := json.Marshal(payload)
	if err != nil {
		exitf("%s: %v", action, err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(c.endpoint, "/")+"/", bytes.NewReader(body))
	if err != nil {
		exitf("%s: %v", action, err)
	}
	req.Header.Set("Content-Type", "application/x-amz-json-1.0")
	req.Header.Set("X-Amz-Target", targetPrefix+"."+action)
	creds, err := c.creds.Retrieve(ctx)
	if err != nil {
		exitf("%s: %v", action, err)
	}
	sum := sha256.Sum256(body)
	if err := v4.NewSigner().SignHTTP(ctx, creds, req, hex.EncodeToString(sum[:]), "dynamodb", c.region, time.Now()); err != nil {
		exitf("%s: %v", action, err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		exitf("%s: %v", action, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		exitf("expected %s to return 2xx, got %d: %s", action, resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	logf("%s returned %d", action, resp.StatusCode)
	if out != nil {
		if err := json.Unmarshal(raw, out); err != nil {
			exitf("%s: decode %s: %v", action, raw, err)
		}
	}
}

func getenv(key, def string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return def
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

func logf(format string, args ...any) {
	fmt.Printf(format+"\n", args...)
}
