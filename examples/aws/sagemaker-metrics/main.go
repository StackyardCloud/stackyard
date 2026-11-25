// Command sagemaker-metrics runs Amazon SageMaker Metrics against Stackyard: log
// training metrics to a trial component and to a training job's trial
// component, query them by iteration and by minute, and read the summary the
// control plane keeps.
//
// The SDK has no SageMaker Metrics module in this repository's go.mod, so requests are
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

func main() {
	c := newClient()
	fmt.Printf("Stackyard SageMaker Metrics client using %s\n", c.endpoint)
	run := fmt.Sprintf("%d", time.Now().UnixNano()%1_000_000_000)
	component := "experiment-run-" + run
	c.sagemaker("CreateTrialComponent", map[string]any{"TrialComponentName": component,
		"Parameters": map[string]any{"learning_rate": map[string]any{"NumberValue": 0.01}}})

	start := float64(time.Now().Add(-time.Hour).Unix())
	var data []any
	for step := 0; step < 6; step++ {
		data = append(data, map[string]any{"MetricName": "train:loss", "Timestamp": start + float64(step*20),
			"Step": step, "Value": 1.0 / float64(step+1)})
	}
	if out := c.restOK(http.MethodPut, "/BatchPutMetrics", map[string]any{"TrialComponentName": component, "MetricData": data}); len(out) != 0 {
		exitf("BatchPutMetrics reported errors: %v", out)
	}
	logf("logged %d loss values to %s", len(data), component)

	arn := "arn:aws:sagemaker:us-east-1:123456789012:experiment-trial-component/" + component
	query := func(stat, period, axis string) map[string]any {
		out := c.restOK(http.MethodPost, "/BatchGetMetrics", map[string]any{"MetricQueries": []any{map[string]any{
			"MetricName": "train:loss", "ResourceArn": arn, "MetricStat": stat, "Period": period, "XAxisType": axis}}})
		return out["MetricQueryResults"].([]any)[0].(map[string]any)
	}
	byStep := query("Last", "IterationNumber", "IterationNumber")
	logf("loss by iteration: x=%v y=%v", byStep["XAxisValues"], byStep["MetricValues"])
	byMinute := query("Avg", "OneMinute", "Timestamp")
	logf("average loss per minute: x=%v y=%v", byMinute["XAxisValues"], byMinute["MetricValues"])
	if len(byStep["MetricValues"].([]any)) != 6 {
		exitf("expected 6 iterations, got %v", byStep)
	}

	// A training job's metrics land on its "<job>-aws-training-job" component.
	job := "train-" + run
	c.sagemaker("CreateTrainingJob", map[string]any{"TrainingJobName": job})
	c.restOK(http.MethodPut, "/BatchPutMetrics", map[string]any{"TrialComponentName": job + "-aws-training-job",
		"MetricData": []any{map[string]any{"MetricName": "validation:accuracy", "Timestamp": start, "Step": 1, "Value": 0.93}}})
	summary := c.sagemaker("DescribeTrialComponent", map[string]any{"TrialComponentName": job + "-aws-training-job"})
	logf("training job component source=%v metrics=%v", summary["Source"], summary["Metrics"])

	summary = c.sagemaker("DescribeTrialComponent", map[string]any{"TrialComponentName": component})
	logf("summary of %s: %v", component, summary["Metrics"])
	c.sagemaker("DeleteTrialComponent", map[string]any{"TrialComponentName": component})
	c.sagemaker("DeleteTrialComponent", map[string]any{"TrialComponentName": job + "-aws-training-job"})
	fmt.Println("SageMaker Metrics example completed")
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

// sagemaker calls the SageMaker control plane (JSON 1.1).
func (c client) sagemaker(action string, payload any) map[string]any {
	status, body := c.do(http.MethodPost, "/", payload, map[string]string{
		"Content-Type": "application/x-amz-json-1.1", "X-Amz-Target": "SageMaker." + action,
	})
	return expectOK(action, status, body)
}

// rest calls a REST JSON SageMaker data plane; payload may be nil.
func (c client) rest(method, path string, payload any) (int, map[string]any) {
	status, body := c.do(method, path, payload, map[string]string{"Content-Type": "application/json"})
	out := map[string]any{}
	if len(bytes.TrimSpace(body)) > 0 {
		if err := json.Unmarshal(body, &out); err != nil {
			exitf("%s %s: decode %s: %v", method, path, body, err)
		}
	}
	return status, out
}

func (c client) restOK(method, path string, payload any) map[string]any {
	status, out := c.rest(method, path, payload)
	if status != http.StatusOK {
		exitf("%s %s failed: status=%d %v", method, path, status, out)
	}
	return out
}

func (c client) do(method, path string, payload any, headers map[string]string) (int, []byte) {
	var body []byte
	if payload != nil {
		var err error
		if body, err = json.Marshal(payload); err != nil {
			exitf("%s %s: %v", method, path, err)
		}
	}
	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, method, c.endpoint+path, bytes.NewReader(body))
	if err != nil {
		exitf("%s %s: %v", method, path, err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	creds, err := c.creds.Retrieve(ctx)
	if err != nil {
		exitf("credentials: %v", err)
	}
	sum := sha256.Sum256(body)
	hash := hex.EncodeToString(sum[:])
	req.Header.Set("X-Amz-Content-Sha256", hash)
	if err := v4.NewSigner().SignHTTP(ctx, creds, req, hash, "sagemaker", c.region, time.Now()); err != nil {
		exitf("sign: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		exitf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	out, err := io.ReadAll(resp.Body)
	if err != nil {
		exitf("read %s %s: %v", method, path, err)
	}
	return resp.StatusCode, out
}

func expectOK(action string, status int, body []byte) map[string]any {
	if status != http.StatusOK {
		exitf("%s failed: status=%d body=%s", action, status, strings.TrimSpace(string(body)))
	}
	out := map[string]any{}
	if len(bytes.TrimSpace(body)) > 0 {
		_ = json.Unmarshal(body, &out)
	}
	return out
}

func getenv(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

func logf(format string, args ...any) {
	fmt.Printf("  "+format+"\n", args...)
}
