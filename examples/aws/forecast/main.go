// Command forecast runs Amazon Forecast against Stackyard: a retail dataset
// with an import job, a dataset group, an auto predictor with its accuracy
// metrics, a forecast and an export job, then deletes the tree.
//
// The SDK has no Forecast module in this repository's go.mod, so requests
// are signed by hand.
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

type client struct {
	endpoint, region, service, target string
	creds                             aws.CredentialsProvider
}

func newClient(service, target string) client {
	return client{
		endpoint: strings.TrimRight(getenv("STACKYARD_ENDPOINT", "http://localhost:4566"), "/"),
		region:   getenv("AWS_REGION", "us-east-1"),
		service:  service,
		target:   target,
		creds: credentials.NewStaticCredentialsProvider(
			getenv("AWS_ACCESS_KEY_ID", "stackyard"), getenv("AWS_SECRET_ACCESS_KEY", "stackyard"), ""),
	}
}

func main() {
	c := newClient("forecast", "AmazonForecast.")
	fmt.Printf("Stackyard Forecast client using %s\n", c.endpoint)
	suffix := fmt.Sprint(time.Now().UnixNano() % 1_000_000_000)
	group, pred, fc := c.pipeline(suffix)
	fmt.Printf("forecast %s from predictor %s\n", fc, pred)

	metrics := c.call("GetAccuracyMetrics", map[string]any{"PredictorArn": pred})["PredictorEvaluationResults"].([]any)
	fmt.Printf("accuracy results: %d algorithm(s)\n", len(metrics))
	exp := c.call("CreateForecastExportJob", map[string]any{"ForecastExportJobName": "export_" + suffix, "ForecastArn": fc,
		"Destination": map[string]any{"S3Config": map[string]any{"Path": "s3://stackyard-forecast/exports/", "RoleArn": role}}})["ForecastExportJobArn"].(string)
	c.active("DescribeForecastExportJob", "ForecastExportJobArn", exp)
	fmt.Printf("exported to s3://stackyard-forecast/exports/\n")

	c.call("DeleteResourceTree", map[string]any{"ResourceArn": group})
	fmt.Printf("deleted the tree under %s; %d predictors left\n", group,
		len(c.call("ListPredictors", map[string]any{"Filters": []any{map[string]any{"Key": "DatasetGroupArn", "Value": group, "Condition": "IS"}}})["Predictors"].([]any)))
}

const role = "arn:aws:iam::123456789012:role/stackyard-forecast"

// active describes arn until it reports ACTIVE.
func (c client) active(describe, key, arn string) map[string]any {
	for i := 0; i < 10; i++ {
		out := c.call(describe, map[string]any{key: arn})
		if out["Status"] == "ACTIVE" {
			return out
		}
		time.Sleep(200 * time.Millisecond)
	}
	exitf("%s never became ACTIVE", arn)
	return nil
}

// pipeline creates a retail dataset with data, a dataset group, an auto
// predictor and a forecast, waiting for each to become ACTIVE.
func (c client) pipeline(suffix string) (string, string, string) {
	ds := c.call("CreateDataset", map[string]any{"DatasetName": "demand_" + suffix, "Domain": "RETAIL", "DatasetType": "TARGET_TIME_SERIES",
		"DataFrequency": "D", "Schema": map[string]any{"Attributes": []any{
			map[string]any{"AttributeName": "item_id", "AttributeType": "string"},
			map[string]any{"AttributeName": "timestamp", "AttributeType": "timestamp"},
			map[string]any{"AttributeName": "demand", "AttributeType": "float"},
		}}})["DatasetArn"].(string)
	job := c.call("CreateDatasetImportJob", map[string]any{"DatasetImportJobName": "load_" + suffix, "DatasetArn": ds,
		"DataSource": map[string]any{"S3Config": map[string]any{"Path": "s3://stackyard-forecast/demand.csv", "RoleArn": role}}})["DatasetImportJobArn"].(string)
	c.active("DescribeDatasetImportJob", "DatasetImportJobArn", job)
	group := c.call("CreateDatasetGroup", map[string]any{"DatasetGroupName": "retail_" + suffix, "Domain": "RETAIL", "DatasetArns": []any{ds}})["DatasetGroupArn"].(string)
	pred := c.call("CreateAutoPredictor", map[string]any{"PredictorName": "demand_" + suffix, "ForecastHorizon": 14, "ForecastFrequency": "D",
		"DataConfig": map[string]any{"DatasetGroupArn": group}})["PredictorArn"].(string)
	c.active("DescribeAutoPredictor", "PredictorArn", pred)
	fc := c.call("CreateForecast", map[string]any{"ForecastName": "demand_" + suffix, "PredictorArn": pred})["ForecastArn"].(string)
	c.active("DescribeForecast", "ForecastArn", fc)
	return group, pred, fc
}

func (c client) call(op string, payload any) map[string]any {
	body, err := json.Marshal(payload)
	if err != nil {
		exitf("%s: %v", op, err)
	}
	raw := c.send(op, body, map[string]string{
		"Content-Type": "application/x-amz-json-1.1", "X-Amz-Target": c.target + op,
	})
	out := map[string]any{}
	if len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			exitf("%s: decode: %v", op, err)
		}
	}
	return out
}

// send signs a request and exits unless it returns 2xx.
func (c client) send(op string, body []byte, headers map[string]string) []byte {
	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint+"/", bytes.NewReader(body))
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
	if err := v4.NewSigner().SignHTTP(ctx, creds, req, payloadHash, c.service, c.region, time.Now()); err != nil {
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
