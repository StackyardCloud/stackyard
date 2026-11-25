// Command machinelearning runs Amazon Machine Learning against Stackyard:
// upload training data to S3, create a data source over it, train a binary
// model, evaluate it, run a batch prediction whose results land back in S3,
// and score one record through a realtime endpoint.
//
// Resources go PENDING -> INPROGRESS -> COMPLETED as they are read. The SDK
// has no Machine Learning module in this repository's go.mod, so requests
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

const target = "AmazonML_20141212."

func main() {
	c := newClient()
	fmt.Printf("Stackyard Machine Learning client using %s\n", c.endpoint)
	run := fmt.Sprint(time.Now().UnixNano() % 1_000_000_000)
	bucket := "ml-example-" + run
	c.send(http.MethodPut, "/"+bucket, "", "", "s3", nil)
	c.send(http.MethodPut, "/"+bucket+"/train.csv", "text/csv", "", "s3", []byte("amount,fraud\n12,0\n950,1\n30,0\n1200,1\n"))

	schema := `{"version":"1.0","targetAttributeName":"fraud","dataFormat":"CSV","dataFileContainsHeader":true,` +
		`"attributes":[{"attributeName":"amount","attributeType":"NUMERIC"},{"attributeName":"fraud","attributeType":"BINARY"}]}`
	ds, model := "ds-"+run, "model-"+run
	c.call("CreateDataSourceFromS3", map[string]any{"DataSourceId": ds, "DataSourceName": "transactions", "ComputeStatistics": true,
		"DataSpec": map[string]any{"DataLocationS3": "s3://" + bucket + "/train.csv", "DataSchema": schema}})
	src := c.wait("DataSource", ds)
	fmt.Printf("data source %s: %v bytes in %v file(s)\n", ds, src["DataSizeInBytes"], src["NumberOfFiles"])

	c.call("CreateMLModel", map[string]any{"MLModelId": model, "MLModelType": "BINARY", "TrainingDataSourceId": ds})
	c.wait("MLModel", model)
	c.call("CreateEvaluation", map[string]any{"EvaluationId": "eval-" + run, "MLModelId": model, "EvaluationDataSourceId": ds})
	eval := c.wait("Evaluation", "eval-"+run)
	fmt.Printf("evaluation metrics: %v\n", eval["PerformanceMetrics"])

	c.call("CreateBatchPrediction", map[string]any{"BatchPredictionId": "batch-" + run, "MLModelId": model,
		"BatchPredictionDataSourceId": ds, "OutputUri": "s3://" + bucket + "/out/"})
	batch := c.wait("BatchPrediction", "batch-"+run)
	fmt.Printf("batch prediction scored %v records into s3://%s/out/\n", batch["TotalRecordCount"], bucket)

	ep := c.call("CreateRealtimeEndpoint", map[string]any{"MLModelId": model})["RealtimeEndpointInfo"].(map[string]any)
	pred := c.call("Predict", map[string]any{"MLModelId": model, "Record": map[string]any{"amount": "700"},
		"PredictEndpoint": ep["EndpointUrl"]})["Prediction"].(map[string]any)
	fmt.Printf("realtime prediction: label %v scores %v\n", pred["predictedLabel"], pred["predictedScores"])

	c.call("DeleteRealtimeEndpoint", map[string]any{"MLModelId": model})
	c.call("DeleteMLModel", map[string]any{"MLModelId": model})
	fmt.Printf("model %s is now %v\n", model, c.call("GetMLModel", map[string]any{"MLModelId": model})["Status"])
}

func (c client) call(op string, payload any) map[string]any {
	return decode(c.send(http.MethodPost, "/", "application/x-amz-json-1.1", target+op, "machinelearning", mustJSON(payload)))
}

// wait reads a resource until it reports COMPLETED.
func (c client) wait(kind, id string) map[string]any {
	for i := 0; i < 20; i++ {
		out := c.call("Get"+kind, map[string]any{kind + "Id": id})
		switch out["Status"] {
		case "COMPLETED":
			return out
		case "FAILED":
			exitf("%s %s failed: %v", kind, id, out["Message"])
		}
		time.Sleep(200 * time.Millisecond)
	}
	exitf("%s %s never completed", kind, id)
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
