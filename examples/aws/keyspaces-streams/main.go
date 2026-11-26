// Command keyspaces-streams runs Amazon Keyspaces Streams against Stackyard:
// create a table with change data capture enabled, find its stream, read the
// shard with a TRIM_HORIZON iterator, then disable CDC and see the shard
// close.
//
// Stackyard has no CQL endpoint, so no rows change and every shard is empty
// (GetRecords reports AT_TIP). The SDK has no Keyspaces Streams module in
// this repository's go.mod, so requests are signed by hand.
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

const (
	signingName = "cassandra"
	contentType = "application/x-amz-json-1.0"
)

func (c client) json(prefix, op string, payload map[string]any) map[string]any {
	return c.call(op, signingName, http.MethodPost, "/", mustJSON(payload), map[string]string{
		"Content-Type": contentType, "X-Amz-Target": prefix + op,
	})
}

func main() {
	c := newClient()
	fmt.Printf("Stackyard Keyspaces Streams client using %s\n", c.endpoint)
	keyspace := fmt.Sprintf("plant_%d", time.Now().UnixNano()%1e12)
	ks := func(op string, p map[string]any) map[string]any { return c.json("KeyspacesService.", op, p) }
	streams := func(op string, p map[string]any) map[string]any { return c.json("KeyspacesStreams.", op, p) }

	ks("CreateKeyspace", map[string]any{"keyspaceName": keyspace})
	ks("CreateTable", map[string]any{
		"keyspaceName": keyspace, "tableName": "readings",
		"schemaDefinition": map[string]any{
			"allColumns":    []any{map[string]any{"name": "sensor", "type": "text"}, map[string]any{"name": "value", "type": "int"}},
			"partitionKeys": []any{map[string]any{"name": "sensor"}},
		},
		"cdcSpecification": map[string]any{"status": "ENABLED", "viewType": "NEW_AND_OLD_IMAGES"},
	})
	table := ks("GetTable", map[string]any{"keyspaceName": keyspace, "tableName": "readings"})
	arn := table["latestStreamArn"].(string)
	fmt.Println("stream:", arn)

	for _, s := range streams("ListStreams", map[string]any{"keyspaceName": keyspace})["streams"].([]any) {
		fmt.Println("listed stream label", s.(map[string]any)["streamLabel"])
	}
	stream := streams("GetStream", map[string]any{"streamArn": arn})
	shard := stream["shards"].([]any)[0].(map[string]any)["shardId"].(string)
	fmt.Printf("stream %v (%v), shard %s\n", stream["streamStatus"], stream["streamViewType"], shard)
	it := streams("GetShardIterator", map[string]any{"streamArn": arn, "shardId": shard, "shardIteratorType": "TRIM_HORIZON"})["shardIterator"]
	recs := streams("GetRecords", map[string]any{"shardIterator": it, "maxResults": 100})
	fmt.Printf("%d records, position %v\n", len(recs["changeRecords"].([]any)), recs["iteratorDescription"].(map[string]any)["iteratorPosition"])

	ks("UpdateTable", map[string]any{"keyspaceName": keyspace, "tableName": "readings", "cdcSpecification": map[string]any{"status": "DISABLED"}})
	stream = streams("GetStream", map[string]any{"streamArn": arn})
	fmt.Println("after disabling CDC:", stream["streamStatus"])
	ks("DeleteKeyspace", map[string]any{"keyspaceName": keyspace})
}

type client struct {
	endpoint, region string
	creds            aws.CredentialsProvider
}

func newClient() client {
	return client{
		endpoint: strings.TrimRight(getenv("STACKYARD_ENDPOINT", "http://localhost:4566"), "/"),
		region:   getenv("AWS_REGION", "us-east-1"),
		creds: credentials.NewStaticCredentialsProvider(
			getenv("AWS_ACCESS_KEY_ID", "stackyard"), getenv("AWS_SECRET_ACCESS_KEY", "stackyard"), ""),
	}
}

// call sends one signed request and exits unless it returns 2xx; a JSON
// reply is decoded.
func (c client) call(op, service, method, path string, body []byte, headers map[string]string) map[string]any {
	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, method, c.endpoint+path, bytes.NewReader(body))
	if err != nil {
		exitf("%s: %v", op, err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	creds, err := c.creds.Retrieve(ctx)
	if err != nil {
		exitf("%s: %v", op, err)
	}
	sum := sha256.Sum256(body)
	payloadHash := hex.EncodeToString(sum[:])
	req.Header.Set("X-Amz-Content-Sha256", payloadHash)
	if err := v4.NewSigner().SignHTTP(ctx, creds, req, payloadHash, service, c.region, time.Now()); err != nil {
		exitf("%s: %v", op, err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		exitf("%s: %v", op, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		exitf("%s: HTTP %d: %s", op, resp.StatusCode, raw)
	}
	out := map[string]any{}
	if len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			exitf("%s: decode: %v", op, err)
		}
	}
	return out
}

func mustJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		exitf("marshal: %v", err)
	}
	return b
}

func getenv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
