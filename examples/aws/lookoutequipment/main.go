// Command lookoutequipment runs Amazon Lookout for Equipment against
// Stackyard: upload sensor CSVs to S3, ingest them into a dataset and read
// the per-sensor statistics, label a fault, train a model, then run an
// inference scheduler and list its executions.
//
// Async work settles on the read after it is first seen. The SDK has no
// Lookout for Equipment module in this repository's go.mod, so requests are
// signed by hand.
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
	target = "AWSLookoutEquipmentFrontendService."
	role   = "arn:aws:iam::123456789012:role/lookout-equipment"
)

func main() {
	c := newClient()
	fmt.Printf("Stackyard Lookout for Equipment client using %s\n", c.endpoint)
	run := fmt.Sprint(time.Now().UnixNano() % 1_000_000_000)
	bucket := "lookout-example-" + run
	c.send(http.MethodPut, "/"+bucket, "", "", "s3", nil)
	c.send(http.MethodPut, "/"+bucket+"/sensors/pump1/data.csv", "text/csv", "", "s3", []byte(
		"Timestamp,vibration,temperature\n2024-01-01T00:00:00Z,0.12,61.5\n2024-01-01T00:05:00Z,0.15,\n2024-01-01T00:10:00Z,0.11,62.1\n"))

	dataset, model, labels := "pumps-"+run, "pump-model-"+run, "pump-faults-"+run
	c.call("CreateDataset", map[string]any{"DatasetName": dataset, "ClientToken": "ds-" + run})
	job := c.call("StartDataIngestionJob", map[string]any{"DatasetName": dataset, "RoleArn": role, "ClientToken": "ingest-" + run,
		"IngestionInputConfiguration": map[string]any{"S3InputConfiguration": map[string]any{"Bucket": bucket, "Prefix": "sensors/"}}})["JobId"]
	ingested := c.wait("DescribeDataIngestionJob", map[string]any{"JobId": job}, "SUCCESS")
	fmt.Printf("ingested %v bytes; data quality %v\n", ingested["IngestedDataSize"], ingested["DataQualitySummary"])
	for _, s := range c.call("ListSensorStatistics", map[string]any{"DatasetName": dataset})["SensorStatisticsSummaries"].([]any) {
		sensor := s.(map[string]any)
		fmt.Printf("  %v/%v missing %v\n", sensor["ComponentName"], sensor["SensorName"], sensor["MissingValues"])
	}

	c.call("CreateLabelGroup", map[string]any{"LabelGroupName": labels, "FaultCodes": []string{"bearing"}, "ClientToken": "lg-" + run})
	c.call("CreateLabel", map[string]any{"LabelGroupName": labels, "StartTime": 1704067200, "EndTime": 1704067800,
		"Rating": "ANOMALY", "FaultCode": "bearing", "ClientToken": "label-" + run})

	c.call("CreateModel", map[string]any{"ModelName": model, "DatasetName": dataset, "RoleArn": role, "ClientToken": "model-" + run,
		"LabelsInputConfiguration": map[string]any{"LabelGroupName": labels}})
	trained := c.wait("DescribeModel", map[string]any{"ModelName": model}, "SUCCESS")
	fmt.Printf("model %s trained; active version %v\n", model, trained["ActiveModelVersion"])

	sched := "pump-monitor-" + run
	c.call("CreateInferenceScheduler", map[string]any{"ModelName": model, "InferenceSchedulerName": sched, "DataUploadFrequency": "PT5M",
		"RoleArn": role, "ClientToken": "sched-" + run,
		"DataInputConfiguration":  map[string]any{"S3InputConfiguration": map[string]any{"Bucket": bucket, "Prefix": "inference/in/"}},
		"DataOutputConfiguration": map[string]any{"S3OutputConfiguration": map[string]any{"Bucket": bucket, "Prefix": "inference/out/"}}})
	c.wait("DescribeInferenceScheduler", map[string]any{"InferenceSchedulerName": sched}, "RUNNING")
	execs := c.call("ListInferenceExecutions", map[string]any{"InferenceSchedulerName": sched})["InferenceExecutionSummaries"].([]any)
	fmt.Printf("inference scheduler %s is RUNNING with %d execution(s)\n", sched, len(execs))

	c.call("StopInferenceScheduler", map[string]any{"InferenceSchedulerName": sched})
	c.wait("DescribeInferenceScheduler", map[string]any{"InferenceSchedulerName": sched}, "STOPPED")
	c.call("DeleteInferenceScheduler", map[string]any{"InferenceSchedulerName": sched})
	c.call("DeleteModel", map[string]any{"ModelName": model})
	c.call("DeleteLabelGroup", map[string]any{"LabelGroupName": labels})
	c.call("DeleteDataset", map[string]any{"DatasetName": dataset})
	fmt.Println("cleaned up")
}

func (c client) call(op string, payload any) map[string]any {
	return decode(c.send(http.MethodPost, "/", "application/x-amz-json-1.0", target+op, "lookoutequipment", mustJSON(payload)))
}

// wait describes a resource until its Status is want.
func (c client) wait(op string, payload map[string]any, want string) map[string]any {
	for i := 0; i < 20; i++ {
		out := c.call(op, payload)
		if out["Status"] == want {
			return out
		}
		if out["Status"] == "FAILED" {
			exitf("%s %v failed: %v", op, payload, out["FailedReason"])
		}
		time.Sleep(200 * time.Millisecond)
	}
	exitf("%s %v never reached %s", op, payload, want)
	return nil
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

// send signs a request for service and exits unless it returns 2xx.
func (c client) send(method, path, contentType, target, service string, body []byte) []byte {
	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, method, c.endpoint+path, bytes.NewReader(body))
	if err != nil {
		exitf("%s %s: %v", method, path, err)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if target != "" {
		req.Header.Set("X-Amz-Target", target)
	}
	creds, err := c.creds.Retrieve(ctx)
	if err != nil {
		exitf("credentials: %v", err)
	}
	sum := sha256.Sum256(body)
	payloadHash := hex.EncodeToString(sum[:])
	req.Header.Set("X-Amz-Content-Sha256", payloadHash)
	if err := v4.NewSigner().SignHTTP(ctx, creds, req, payloadHash, service, c.region, time.Now()); err != nil {
		exitf("sign: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		exitf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		exitf("%s %s %s: HTTP %d: %s", method, path, target, resp.StatusCode, raw)
	}
	return raw
}

func decode(raw []byte) map[string]any {
	out := map[string]any{}
	if len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			exitf("decode %q: %v", raw, err)
		}
	}
	return out
}

func mustJSON(v any) []byte {
	raw, err := json.Marshal(v)
	if err != nil {
		exitf("encode: %v", err)
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
