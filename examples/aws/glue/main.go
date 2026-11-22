// Command glue walks a small AWS Glue workflow against Stackyard: a catalog
// database and partitioned table, a crawler run, a job run, and a schema in
// the Schema Registry. It signs requests with SigV4 and exits non-zero on the
// first unexpected response.
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
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

type client struct {
	endpoint, region string
	creds            aws.CredentialsProvider
	signer           *v4.Signer
}

func main() {
	c := client{
		endpoint: getenv("STACKYARD_ENDPOINT", "http://localhost:4566"),
		region:   getenv("AWS_REGION", "us-east-1"),
		creds:    credentials.NewStaticCredentialsProvider(getenv("AWS_ACCESS_KEY_ID", "stackyard"), getenv("AWS_SECRET_ACCESS_KEY", "stackyard"), ""),
		signer:   v4.NewSigner(),
	}
	suffix := fmt.Sprint(time.Now().UnixNano())
	db, job, crawler := "example_"+suffix, "example-job-"+suffix, "example-crawler-"+suffix
	fmt.Printf("Stackyard AWS Glue example using %s\n", c.endpoint)

	c.call("CreateDatabase", map[string]any{"DatabaseInput": map[string]any{"Name": db, "Description": "example database"}})
	c.call("CreateTable", map[string]any{"DatabaseName": db, "TableInput": map[string]any{
		"Name": "events",
		"StorageDescriptor": map[string]any{
			"Location": "s3://example-bucket/events/",
			"Columns":  []any{map[string]any{"Name": "id", "Type": "string"}, map[string]any{"Name": "value", "Type": "double"}},
		},
		"PartitionKeys": []any{map[string]any{"Name": "day", "Type": "string"}},
	}})
	c.call("BatchCreatePartition", map[string]any{"DatabaseName": db, "TableName": "events", "PartitionInputList": []any{
		map[string]any{"Values": []any{"2024-01-01"}}, map[string]any{"Values": []any{"2024-01-02"}},
	}})
	parts := c.call("GetPartitions", map[string]any{"DatabaseName": db, "TableName": "events", "Expression": "day >= '2024-01-02'"})
	if n := len(parts["Partitions"].([]any)); n != 1 {
		exitf("GetPartitions with an expression returned %d partitions, want 1", n)
	}
	logf("table %s.events has partitions; the expression matched one", db)

	c.call("CreateCrawler", map[string]any{"Name": crawler, "Role": "arn:aws:iam::123456789012:role/glue", "DatabaseName": db,
		"Targets": map[string]any{"S3Targets": []any{map[string]any{"Path": "s3://example-bucket/clicks/"}}}})
	c.call("StartCrawler", map[string]any{"Name": crawler})
	state := c.waitFor("GetCrawler", map[string]any{"Name": crawler}, func(out map[string]any) string {
		return out["Crawler"].(map[string]any)["State"].(string)
	}, "READY")
	c.call("GetTable", map[string]any{"DatabaseName": db, "Name": "clicks"})
	logf("crawler %s finished (%s) and registered table clicks", crawler, state)

	c.call("CreateJob", map[string]any{"Name": job, "Role": "arn:aws:iam::123456789012:role/glue",
		"Command": map[string]any{"Name": "glueetl", "ScriptLocation": "s3://example-bucket/etl.py"}})
	runID := c.call("StartJobRun", map[string]any{"JobName": job})["JobRunId"].(string)
	final := c.waitFor("GetJobRun", map[string]any{"JobName": job, "RunId": runID}, func(out map[string]any) string {
		return out["JobRun"].(map[string]any)["JobRunState"].(string)
	}, "SUCCEEDED")
	logf("job run %s is %s", runID, final)

	schema := `{"type":"record","name":"Event","fields":[{"name":"id","type":"string"}]}`
	created := c.call("CreateSchema", map[string]any{"SchemaName": "event-" + suffix, "DataFormat": "AVRO", "SchemaDefinition": schema})
	logf("schema %s version status %v", created["SchemaArn"], created["SchemaVersionStatus"])

	c.call("DeleteCrawler", map[string]any{"Name": crawler})
	c.call("DeleteJob", map[string]any{"JobName": job})
	c.call("DeleteDatabase", map[string]any{"Name": db})
	fmt.Println("AWS Glue example completed")
}

// waitFor polls op until extract returns want (runs settle on the next read).
func (c client) waitFor(op string, in map[string]any, extract func(map[string]any) string, want string) string {
	var got string
	for i := 0; i < 10; i++ {
		if got = extract(c.call(op, in)); got == want {
			return got
		}
	}
	exitf("%s never reached %s (last %s)", op, want, got)
	return ""
}

func (c client) call(op string, in map[string]any) map[string]any {
	body, _ := json.Marshal(in)
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, c.endpoint+"/", bytes.NewReader(body))
	if err != nil {
		exitf("%s: %v", op, err)
	}
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "AWSGlue."+op)
	sum := sha256.Sum256(body)
	creds, err := c.creds.Retrieve(context.Background())
	if err != nil {
		exitf("credentials: %v", err)
	}
	if err := c.signer.SignHTTP(context.Background(), creds, req, hex.EncodeToString(sum[:]), "glue", c.region, time.Now()); err != nil {
		exitf("%s: sign: %v", op, err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		exitf("%s: %v", op, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode/100 != 2 {
		exitf("%s: HTTP %d: %s", op, resp.StatusCode, raw)
	}
	out := map[string]any{}
	if err := json.Unmarshal(raw, &out); err != nil {
		exitf("%s: invalid JSON: %v", op, err)
	}
	return out
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func logf(format string, args ...any) { fmt.Printf("  "+format+"\n", args...) }

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", args...)
	os.Exit(1)
}
