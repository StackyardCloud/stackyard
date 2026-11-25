// Command cost-optimization-hub runs AWS Cost Optimization Hub against
// Stackyard: enroll the account, set savings preferences, list the top
// recommendations, open one, and summarise savings by resource type.
//
// Stackyard serves a fixed, deterministic recommendation catalog while the
// account is enrolled. The SDK has no Cost Optimization Hub module in this
// repository's go.mod, so requests are signed by hand.
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
	signingName  = "cost-optimization-hub"
	targetPrefix = "CostOptimizationHubService."
	contentType  = "application/x-amz-json-1.0"
)

func main() {
	c := newClient()
	fmt.Printf("Stackyard Cost Optimization Hub client using %s\n", c.endpoint)

	fmt.Println("enrollment:", c.call("UpdateEnrollmentStatus", map[string]any{"status": "Active"})["status"])
	prefs := c.call("UpdatePreferences", map[string]any{"savingsEstimationMode": "AfterDiscounts"})
	fmt.Println("savings estimation mode:", prefs["savingsEstimationMode"])

	recs := c.call("ListRecommendations", map[string]any{
		"maxResults": 3, "orderBy": map[string]any{"dimension": "EstimatedMonthlySavings", "order": "Desc"},
	})["items"].([]any)
	for _, r := range recs {
		m := r.(map[string]any)
		fmt.Printf("%-28v %-22v %-26v saves $%.2f/month\n", m["recommendationId"], m["actionType"], m["currentResourceType"], m["estimatedMonthlySavings"])
	}

	rec := c.call("GetRecommendation", map[string]any{"recommendationId": "coh-ec2-rightsize-0001"})
	cur := rec["currentResourceDetails"].(map[string]any)["ec2Instance"].(map[string]any)["configuration"].(map[string]any)["instance"].(map[string]any)
	next := rec["recommendedResourceDetails"].(map[string]any)["ec2Instance"].(map[string]any)["configuration"].(map[string]any)["instance"].(map[string]any)
	fmt.Printf("rightsize %v: %v -> %v (restart needed: %v)\n", rec["resourceId"], cur["type"], next["type"], rec["restartNeeded"])

	sum := c.call("ListRecommendationSummaries", map[string]any{"groupBy": "ResourceType", "metrics": []any{"SavingsPercentage"}})
	for _, it := range sum["items"].([]any) {
		m := it.(map[string]any)
		fmt.Printf("%-22v %v recommendation(s), $%.2f/month\n", m["group"], m["recommendationCount"], m["estimatedMonthlySavings"])
	}
	fmt.Printf("total deduped savings: $%.2f/month (%v%% of spend)\n", sum["estimatedTotalDedupedSavings"], sum["metrics"].(map[string]any)["savingsPercentage"])
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

// call sends one awsJson operation and exits unless it returns 2xx.
func (c client) call(op string, payload any) map[string]any {
	body, err := json.Marshal(payload)
	if err != nil {
		exitf("%s: %v", op, err)
	}
	raw := c.send(op, http.MethodPost, signingName, c.endpoint+"/", body, map[string]string{
		"Content-Type": contentType, "X-Amz-Target": targetPrefix + op,
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
