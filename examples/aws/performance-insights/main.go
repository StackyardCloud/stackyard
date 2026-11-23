// Command performance-insights runs Amazon RDS Performance Insights against
// Stackyard: create an RDS instance, read its database load grouped by wait
// event, list the top SQL, fetch a statement's full text, and run an analysis
// report.
//
// Stackyard's metric data is synthetic but deterministic, and the instance's
// DbiResourceId is "db-" plus its identifier, upper-cased without dashes. The
// SDK has no Performance Insights module in this repository's go.mod, so
// requests are signed by hand.
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
	"net/url"
	"os"
	"strings"
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
		endpoint: strings.TrimRight(getenv("STACKYARD_ENDPOINT", "http://localhost:4566"), "/"),
		region:   getenv("AWS_REGION", "us-east-1"),
		creds: credentials.NewStaticCredentialsProvider(
			getenv("AWS_ACCESS_KEY_ID", "stackyard"), getenv("AWS_SECRET_ACCESS_KEY", "stackyard"), ""),
	}
	fmt.Printf("Stackyard Performance Insights client using %s\n", c.endpoint)

	name := fmt.Sprintf("orders-%d", time.Now().UnixNano())
	c.send("CreateDBInstance", "rds", c.endpoint+"/", []byte(url.Values{
		"Action": {"CreateDBInstance"}, "Version": {"2014-10-31"}, "DBInstanceIdentifier": {name}, "Engine": {"postgres"},
		"DBInstanceClass": {"db.t3.micro"}, "AllocatedStorage": {"20"}, "MasterUsername": {"admin"}, "MasterUserPassword": {"stackyard123"},
	}.Encode()), map[string]string{"Content-Type": "application/x-www-form-urlencoded"})
	id := "db-" + strings.ToUpper(strings.ReplaceAll(name, "-", ""))
	fmt.Println("instance:", name, "resource id:", id)

	end := time.Now().UTC().Truncate(time.Minute)
	start := end.Add(-time.Hour)
	base := map[string]any{"ServiceType": "RDS", "Identifier": id, "StartTime": start.Unix(), "EndTime": end.Unix()}

	metrics := c.call("GetResourceMetrics", with(base, map[string]any{
		"PeriodInSeconds": 300,
		"MetricQueries":   []any{map[string]any{"Metric": "db.load.avg", "GroupBy": map[string]any{"Group": "db.wait_event", "Limit": 3}}},
	}))
	for _, m := range metrics["MetricList"].([]any) {
		series := m.(map[string]any)
		key := series["Key"].(map[string]any)
		fmt.Printf("%v %v: %d points\n", key["Metric"], key["Dimensions"], len(series["DataPoints"].([]any)))
	}

	keys := c.call("DescribeDimensionKeys", with(base, map[string]any{
		"Metric": "db.load.avg", "GroupBy": map[string]any{"Group": "db.sql", "Limit": 3},
	}))
	var topID string
	for i, k := range keys["Keys"].([]any) {
		key := k.(map[string]any)
		dims := key["Dimensions"].(map[string]any)
		if i == 0 {
			topID, _ = dims["db.sql.id"].(string)
		}
		fmt.Printf("top SQL %d: %.3f avg sessions — %v\n", i+1, key["Total"], dims["db.sql.statement"])
	}
	detail := c.call("GetDimensionKeyDetails", map[string]any{"ServiceType": "RDS", "Identifier": id, "Group": "db.sql", "GroupIdentifier": topID})
	fmt.Println("full text:", detail["Dimensions"].([]any)[0].(map[string]any)["Value"])

	report := c.call("CreatePerformanceAnalysisReport", with(base, nil))["AnalysisReportId"].(string)
	get := map[string]any{"ServiceType": "RDS", "Identifier": id, "AnalysisReportId": report, "TextFormat": "PLAIN_TEXT"}
	for range 2 {
		r := c.call("GetPerformanceAnalysisReport", get)["AnalysisReport"].(map[string]any)
		fmt.Println("report", report, r["Status"])
		if insights, ok := r["Insights"].([]any); ok {
			fmt.Println("insight:", insights[0].(map[string]any)["Description"])
		}
	}
	c.call("DeletePerformanceAnalysisReport", map[string]any{"ServiceType": "RDS", "Identifier": id, "AnalysisReportId": report})
	fmt.Println("deleted report", report)
}

func with(base, extra map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range base {
		out[k] = v
	}
	for k, v := range extra {
		out[k] = v
	}
	return out
}

func (c client) call(op string, payload any) map[string]any {
	body, err := json.Marshal(payload)
	if err != nil {
		exitf("%s: %v", op, err)
	}
	raw := c.send(op, "pi", c.endpoint+"/", body, map[string]string{
		"Content-Type": "application/x-amz-json-1.1", "X-Amz-Target": "PerformanceInsightsv20180227." + op,
	})
	out := map[string]any{}
	if len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			exitf("%s: decode: %v", op, err)
		}
	}
	return out
}

// send signs a POST for service and exits unless it returns 2xx.
func (c client) send(op, service, target string, body []byte, headers map[string]string) []byte {
	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, target, bytes.NewReader(body))
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
	return raw
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
