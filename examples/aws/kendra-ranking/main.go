// Command kendra-ranking runs Amazon Kendra Intelligent Ranking against
// Stackyard: create a rescore execution plan, wait for it to become ACTIVE,
// rescore search results from another engine, tag the plan and delete it.
//
// Stackyard ranks documents by query-term overlap (titles count double;
// documents sharing a GroupId share the group's best score). Requests are
// signed by hand (service kendra-ranking, JSON 1.0).
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
	fmt.Printf("Stackyard Kendra Intelligent Ranking client using %s\n", c.endpoint)

	plan := c.call("CreateRescoreExecutionPlan", map[string]any{
		"Name": "docs-reranker", "CapacityUnits": map[string]any{"RescoreCapacityUnits": 1},
		"Tags": []any{map[string]any{"Key": "team", "Value": "search"}},
	})
	id, arn := plan["Id"].(string), plan["Arn"].(string)
	fmt.Println("plan:", arn)
	for {
		status := c.call("DescribeRescoreExecutionPlan", map[string]any{"Id": id})["Status"]
		fmt.Println("status:", status)
		if status == "ACTIVE" {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}

	out := c.call("Rescore", map[string]any{
		"RescoreExecutionPlanId": id,
		"SearchQuery":            "emulate cloud search locally",
		"Documents": []any{
			map[string]any{"Id": "blog", "Title": "Company picnic", "Body": "Photos from the summer picnic", "OriginalScore": 12.5},
			map[string]any{"Id": "guide", "Title": "Emulate cloud search locally", "Body": "Run CloudSearch on your laptop", "OriginalScore": 3.1},
			map[string]any{"Id": "faq", "Body": "How do I search the cloud console?", "OriginalScore": 7.0},
		},
	})
	for i, item := range out["ResultItems"].([]any) {
		r := item.(map[string]any)
		fmt.Printf("%d. %v (%.4f)\n", i+1, r["DocumentId"], r["Score"])
	}

	c.call("TagResource", map[string]any{"ResourceARN": arn, "Tags": []any{map[string]any{"Key": "env", "Value": "dev"}}})
	fmt.Println("tags:", c.call("ListTagsForResource", map[string]any{"ResourceARN": arn})["Tags"])
	c.call("DeleteRescoreExecutionPlan", map[string]any{"Id": id})
	fmt.Println("deleted", id)
}

func (c client) call(op string, payload any) map[string]any {
	body, err := json.Marshal(payload)
	if err != nil {
		exitf("%s: %v", op, err)
	}
	raw := c.send(op, http.MethodPost, "kendra-ranking", c.endpoint+"/", body, map[string]string{
		"Content-Type": "application/x-amz-json-1.0", "X-Amz-Target": "AWSKendraRerankingFrontendService." + op,
	})
	out := map[string]any{}
	if len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			exitf("%s: decode: %v", op, err)
		}
	}
	return out
}

// send signs a request for service and exits unless it returns 2xx.
func (c client) send(op, method, service, target string, body []byte, headers map[string]string) []byte {
	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, method, target, bytes.NewReader(body))
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

func newClient() client {
	return client{
		endpoint: strings.TrimRight(getenv("STACKYARD_ENDPOINT", "http://localhost:4566"), "/"),
		region:   getenv("AWS_REGION", "us-east-1"),
		creds: credentials.NewStaticCredentialsProvider(
			getenv("AWS_ACCESS_KEY_ID", "stackyard"), getenv("AWS_SECRET_ACCESS_KEY", "stackyard"), ""),
	}
}

type client struct {
	endpoint, region string
	creds            aws.CredentialsProvider
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
