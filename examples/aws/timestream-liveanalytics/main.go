// Command timestream-liveanalytics writes records to Amazon Timestream through
// Stackyard and reads them back with Timestream SQL.
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
}

func main() {
	c := client{
		endpoint: getenv("STACKYARD_ENDPOINT", "http://localhost:4566"),
		region:   getenv("AWS_REGION", "us-east-1"),
		creds: credentials.NewStaticCredentialsProvider(
			getenv("AWS_ACCESS_KEY_ID", "stackyard"), getenv("AWS_SECRET_ACCESS_KEY", "stackyard"), ""),
	}
	fmt.Printf("Stackyard Timestream client using %s\n", c.endpoint)

	db := fmt.Sprintf("example_%d", time.Now().UnixNano())
	c.call("CreateDatabase", map[string]any{"DatabaseName": db})
	c.call("CreateTable", map[string]any{"DatabaseName": db, "TableName": "cpu",
		"RetentionProperties": map[string]any{"MemoryStoreRetentionPeriodInHours": 24, "MagneticStoreRetentionPeriodInDays": 7}})
	fmt.Println("Created", db+".cpu")

	now := time.Now()
	var records []any
	for i, host := range []string{"web-1", "web-2", "web-1", "web-2"} {
		records = append(records, map[string]any{
			"Dimensions":  []any{map[string]any{"Name": "host", "Value": host}},
			"MeasureName": "usage", "MeasureValue": fmt.Sprint(20 + i*10), "MeasureValueType": "DOUBLE",
			"Time": fmt.Sprint(now.Add(-time.Duration(i) * time.Minute).UnixMilli()),
		})
	}
	out := c.call("WriteRecords", map[string]any{"DatabaseName": db, "TableName": "cpu", "Records": records})
	fmt.Println("Ingested:", out["RecordsIngested"])

	sql := fmt.Sprintf(`SELECT host, avg(measure_value::double) AS avg_usage, count(*) AS samples FROM "%s"."cpu" WHERE time > ago(1h) GROUP BY host ORDER BY host`, db)
	res := c.call("Query", map[string]any{"QueryString": sql})
	rows, _ := res["Rows"].([]any)
	if len(rows) != 2 {
		exitf("expected 2 rows, got %v", res)
	}
	for _, r := range rows {
		data := r.(map[string]any)["Data"].([]any)
		fmt.Printf("  %v avg=%v samples=%v\n", data[0].(map[string]any)["ScalarValue"], data[1].(map[string]any)["ScalarValue"], data[2].(map[string]any)["ScalarValue"])
	}

	c.call("DeleteTable", map[string]any{"DatabaseName": db, "TableName": "cpu"})
	c.call("DeleteDatabase", map[string]any{"DatabaseName": db})
	fmt.Println("Cleaned up", db)
}

// call sends one signed Timestream (JSON 1.0) request and exits unless it succeeds.
func (c client) call(op string, payload any) map[string]any {
	ctx := context.Background()
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint+"/", bytes.NewReader(body))
	if err != nil {
		exitf("%s: %v", op, err)
	}
	req.Header.Set("Content-Type", "application/x-amz-json-1.0")
	req.Header.Set("X-Amz-Target", "Timestream_20181101."+op)
	creds, err := c.creds.Retrieve(ctx)
	if err != nil {
		exitf("%s: %v", op, err)
	}
	sum := sha256.Sum256(body)
	if err := v4.NewSigner().SignHTTP(ctx, creds, req, hex.EncodeToString(sum[:]), "timestream", c.region, time.Now()); err != nil {
		exitf("%s: %v", op, err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		exitf("%s: %v", op, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		exitf("%s: %d %s", op, resp.StatusCode, raw)
	}
	out := map[string]any{}
	_ = json.Unmarshal(raw, &out)
	return out
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
