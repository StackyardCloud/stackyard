// Command osis walks an Amazon OpenSearch Ingestion lifecycle against
// Stackyard: validate a pipeline configuration, create a pipeline from a
// blueprint, scale it, stop and start it, tag it, then delete it.
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

const prefix = "/2022-01-01/osis/"

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
	fmt.Printf("Stackyard OpenSearch Ingestion client using %s\n", c.endpoint)

	blueprint := c.call("GetPipelineBlueprint", http.MethodGet, "getPipelineBlueprint/AWS-ApacheLogPipeline", nil)["Blueprint"].(map[string]any)
	config := blueprint["PipelineConfigurationBody"].(string)
	check := c.call("ValidatePipeline", http.MethodPost, "validatePipeline", map[string]any{"PipelineConfigurationBody": config})
	fmt.Printf("blueprint %s valid: %v\n", blueprint["BlueprintName"], check["isValid"])

	name := fmt.Sprintf("apache-%d", time.Now().UnixNano()%1_000_000_000)
	created := c.call("CreatePipeline", http.MethodPost, "createPipeline", map[string]any{
		"PipelineName": name, "MinUnits": 1, "MaxUnits": 2, "PipelineConfigurationBody": config,
		"Tags": []map[string]string{{"Key": "team", "Value": "logs"}},
	})["Pipeline"].(map[string]any)
	fmt.Printf("created %s (%v), ingest at %v\n", name, created["Status"], created["IngestEndpointUrls"])
	waitFor(c, name, "ACTIVE")

	c.call("UpdatePipeline", http.MethodPut, "updatePipeline/"+name, map[string]any{"MaxUnits": 4})
	scaled := waitFor(c, name, "ACTIVE")
	fmt.Printf("scaled to %v-%v units\n", scaled["MinUnits"], scaled["MaxUnits"])

	c.call("StopPipeline", http.MethodPut, "stopPipeline/"+name, nil)
	waitFor(c, name, "STOPPED")
	c.call("StartPipeline", http.MethodPut, "startPipeline/"+name, nil)
	waitFor(c, name, "ACTIVE")

	progress := c.call("GetPipelineChangeProgress", http.MethodGet, "getPipelineChangeProgress/"+name, nil)
	fmt.Printf("%d recorded change(s)\n", len(progress["ChangeProgressStatuses"].([]any)))

	arn := created["PipelineArn"].(string)
	q := "?arn=" + url.QueryEscape(arn)
	c.call("TagResource", http.MethodPost, "tagResource/"+q, map[string]any{"Tags": []map[string]string{{"Key": "env", "Value": "dev"}}})
	tags := c.call("ListTagsForResource", http.MethodGet, "listTagsForResource/"+q, nil)
	fmt.Printf("tags: %v\n", tags["Tags"])

	c.call("DeletePipeline", http.MethodDelete, "deletePipeline/"+name, nil)
	fmt.Println("OpenSearch Ingestion example completed")
}

// waitFor reads a pipeline until it reaches status (reads settle transitions).
func waitFor(c client, name, status string) map[string]any {
	for i := 0; i < 5; i++ {
		got := c.call("GetPipeline", http.MethodGet, "getPipeline/"+name, nil)["Pipeline"].(map[string]any)
		if got["Status"] == status {
			return got
		}
	}
	exitf("pipeline %s never reached %s", name, status)
	return nil
}

func (c client) call(op, method, path string, payload any) map[string]any {
	var body []byte
	if payload != nil {
		var err error
		if body, err = json.Marshal(payload); err != nil {
			exitf("%s: %v", op, err)
		}
	}
	raw := c.send(op, method, c.endpoint+prefix+path, body)
	out := map[string]any{}
	if len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			exitf("%s: decode: %v", op, err)
		}
	}
	return out
}

// send signs an osis request and exits unless it returns 2xx.
func (c client) send(op, method, target string, body []byte) []byte {
	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, method, target, bytes.NewReader(body))
	if err != nil {
		exitf("%s: %v", op, err)
	}
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	creds, err := c.creds.Retrieve(ctx)
	if err != nil {
		exitf("%s: %v", op, err)
	}
	sum := sha256.Sum256(body)
	if err := v4.NewSigner().SignHTTP(ctx, creds, req, hex.EncodeToString(sum[:]), "osis", c.region, time.Now()); err != nil {
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
