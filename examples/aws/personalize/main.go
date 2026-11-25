// Command personalize runs Amazon Personalize against Stackyard: create a
// dataset group with Interactions and Items datasets, bulk-import
// interactions from S3, train a user-personalization solution, deploy it as a
// campaign, run a batch inference job and read its output back from S3.
//
// Async resources report CREATE IN_PROGRESS on their first Describe and
// settle on the next; waitActive polls the way a real client would. The SDK
// has no Personalize module in this repository's go.mod, so requests are
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

const role = "arn:aws:iam::123456789012:role/personalize"

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
	fmt.Printf("Stackyard Personalize client using %s\n", c.endpoint)
	run := fmt.Sprintf("%d", time.Now().UnixNano()%1_000_000_000)
	bucket := "personalize-example-" + run

	// Training data in S3.
	c.send(http.MethodPut, "/"+bucket, "", "s3", nil)
	c.send(http.MethodPut, "/"+bucket+"/interactions.csv", "", "s3", []byte(
		"USER_ID,ITEM_ID,EVENT_TYPE,TIMESTAMP\nu1,i1,click,1700000000\nu1,i2,click,1700000060\n"+
			"u2,i1,purchase,1700000120\nu2,i3,click,1700000180\nu3,i2,click,1700000240\n"))
	c.send(http.MethodPut, "/"+bucket+"/users.jsonl", "", "s3", []byte(`{"userId": "u1"}`+"\n"+`{"userId": "u3"}`+"\n"))

	group := c.call("CreateDatasetGroup", map[string]any{"name": "shop-" + run})["datasetGroupArn"].(string)
	c.waitActive("DatasetGroup", group)
	schema := c.call("CreateSchema", map[string]any{"name": "interactions-" + run, "schema": avro(
		"USER_ID", "string", "ITEM_ID", "string", "EVENT_TYPE", "string", "TIMESTAMP", "long")})["schemaArn"].(string)
	dataset := c.call("CreateDataset", map[string]any{"name": "interactions-" + run, "schemaArn": schema,
		"datasetGroupArn": group, "datasetType": "Interactions"})["datasetArn"].(string)
	c.waitActive("Dataset", dataset)

	job := c.call("CreateDatasetImportJob", map[string]any{"jobName": "import-" + run, "datasetArn": dataset, "roleArn": role,
		"dataSource": map[string]any{"dataLocation": "s3://" + bucket + "/interactions.csv"}})["datasetImportJobArn"].(string)
	fmt.Println("import job:", c.waitActive("DatasetImportJob", job)["status"])

	solution := c.call("CreateSolution", map[string]any{"name": "picks-" + run, "datasetGroupArn": group,
		"recipeArn": "arn:aws:personalize:::recipe/aws-user-personalization"})["solutionArn"].(string)
	c.waitActive("Solution", solution)
	version := c.call("CreateSolutionVersion", map[string]any{"solutionArn": solution})["solutionVersionArn"].(string)
	c.waitActive("SolutionVersion", version)
	fmt.Printf("solution metrics: %v\n", c.call("GetSolutionMetrics", map[string]any{"solutionVersionArn": version})["metrics"])

	campaign := c.call("CreateCampaign", map[string]any{"name": "home-" + run, "solutionVersionArn": version})["campaignArn"].(string)
	fmt.Println("campaign:", c.waitActive("Campaign", campaign)["status"])

	batch := c.call("CreateBatchInferenceJob", map[string]any{
		"jobName": "nightly-" + run, "solutionVersionArn": version, "numResults": 3, "roleArn": role,
		"jobInput":  map[string]any{"s3DataSource": map[string]any{"path": "s3://" + bucket + "/users.jsonl"}},
		"jobOutput": map[string]any{"s3DataDestination": map[string]any{"path": "s3://" + bucket + "/out/"}},
	})["batchInferenceJobArn"].(string)
	c.waitActive("BatchInferenceJob", batch)
	fmt.Printf("batch output:\n%s", c.send(http.MethodGet, "/"+bucket+"/out/users.jsonl.out", "", "s3", nil))

	fmt.Printf("campaigns for the solution: %d\n", len(c.call("ListCampaigns", map[string]any{"solutionArn": solution})["campaigns"].([]any)))
	c.call("DeleteCampaign", map[string]any{"campaignArn": campaign})
	c.call("DeleteSolution", map[string]any{"solutionArn": solution})
	c.call("DeleteDataset", map[string]any{"datasetArn": dataset})
	c.call("DeleteSchema", map[string]any{"schemaArn": schema})
	c.call("DeleteDatasetGroup", map[string]any{"datasetGroupArn": group})
	fmt.Println("deleted dataset group", group)
}

// avro builds an Avro record schema from name/type pairs.
func avro(pairs ...string) string {
	var fields []map[string]string
	for i := 0; i+1 < len(pairs); i += 2 {
		fields = append(fields, map[string]string{"name": pairs[i], "type": pairs[i+1]})
	}
	raw, _ := json.Marshal(map[string]any{"type": "record", "name": "Interactions",
		"namespace": "com.amazonaws.personalize.schema", "fields": fields, "version": "1.0"})
	return string(raw)
}

// waitActive describes kind until it is ACTIVE and returns it.
func (c client) waitActive(kind, arn string) map[string]any {
	key := strings.ToLower(kind[:1]) + kind[1:]
	for i := 0; i < 20; i++ {
		obj, _ := c.call("Describe"+kind, map[string]any{key + "Arn": arn})[key].(map[string]any)
		switch status, _ := obj["status"].(string); {
		case status == "ACTIVE":
			return obj
		case strings.HasSuffix(status, "FAILED"):
			exitf("%s %s failed: %v", kind, arn, obj["failureReason"])
		}
		time.Sleep(100 * time.Millisecond)
	}
	exitf("%s %s never became ACTIVE", kind, arn)
	return nil
}

func (c client) call(op string, payload any) map[string]any {
	body, err := json.Marshal(payload)
	if err != nil {
		exitf("%s: %v", op, err)
	}
	raw := c.send(http.MethodPost, "/", "AmazonPersonalize."+op, "personalize", body)
	out := map[string]any{}
	if len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			exitf("%s: decode: %v", op, err)
		}
	}
	return out
}

// send signs a request for service and exits unless it returns 2xx.
func (c client) send(method, path, target, service string, body []byte) []byte {
	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, method, c.endpoint+path, bytes.NewReader(body))
	if err != nil {
		exitf("%s %s: %v", method, path, err)
	}
	if target != "" {
		req.Header.Set("Content-Type", "application/x-amz-json-1.1")
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
