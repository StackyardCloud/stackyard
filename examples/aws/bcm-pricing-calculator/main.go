// Command bcm-pricing-calculator runs AWS Billing and Cost Management Pricing
// Calculator against Stackyard: build a bill scenario with EC2 usage and a
// Reserved Instance, estimate it, then price a workload of S3 storage.
//
// Prices come from Stackyard's small built-in us-east-1 catalog, so results
// are deterministic. The SDK has no Pricing Calculator module in this
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
	signingName  = "bcm-pricing-calculator"
	targetPrefix = "AWSBCMPricingCalculator."
	contentType  = "application/x-amz-json-1.0"
	account      = "123456789012"
	riOffering   = "7d1e2f3a-4b5c-4d6e-8f9a-0b1c2d3e4f5a" // m5.large, 1yr, No Upfront
)

func main() {
	c := newClient()
	fmt.Printf("Stackyard Pricing Calculator client using %s\n", c.endpoint)

	now := time.Now().UTC()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339)
	scenario := c.call("CreateBillScenario", map[string]any{"name": fmt.Sprintf("growth-%d", now.Unix())})["id"].(string)
	fmt.Println("bill scenario:", scenario)

	c.call("BatchCreateBillScenarioUsageModification", map[string]any{"billScenarioId": scenario, "usageModifications": []any{
		map[string]any{"key": "web", "serviceCode": "AmazonEC2", "usageType": "BoxUsage:m5.large", "operation": "RunInstances",
			"usageAccountId": account, "amounts": []any{map[string]any{"startHour": monthStart, "amount": 4}}},
	}})
	c.call("BatchCreateBillScenarioCommitmentModification", map[string]any{"billScenarioId": scenario, "commitmentModifications": []any{
		map[string]any{"key": "ri", "usageAccountId": account, "commitmentAction": map[string]any{
			"addReservedInstanceAction": map[string]any{"reservedInstancesOfferingId": riOffering, "instanceCount": 2}}},
	}})

	estimate := c.call("CreateBillEstimate", map[string]any{"billScenarioId": scenario, "name": "with-ri"})
	fmt.Printf("bill estimate %v: %v\n", estimate["id"], estimate["status"])
	got := c.call("GetBillEstimate", map[string]any{"identifier": estimate["id"]})
	total := got["costSummary"].(map[string]any)["totalCostDifference"].(map[string]any)["estimatedCost"].(map[string]any)
	fmt.Printf("estimate %v: %.2f %v\n", got["status"], total["amount"], total["currency"])
	lines := c.call("ListBillEstimateLineItems", map[string]any{"billEstimateId": estimate["id"]})["items"].([]any)
	for _, l := range lines {
		m := l.(map[string]any)
		fmt.Printf("  %-16v %-24v %10.2f\n", m["lineItemType"], m["usageType"], m["estimatedCost"].(map[string]any)["amount"])
	}

	workload := c.call("CreateWorkloadEstimate", map[string]any{"name": fmt.Sprintf("storage-%d", now.Unix())})["id"].(string)
	c.call("BatchCreateWorkloadEstimateUsage", map[string]any{"workloadEstimateId": workload, "usage": []any{
		map[string]any{"key": "s3", "serviceCode": "AmazonS3", "usageType": "TimedStorage-ByteHrs", "operation": "StandardStorage", "usageAccountId": account, "amount": 2048},
	}})
	w := c.call("GetWorkloadEstimate", map[string]any{"identifier": workload})
	fmt.Printf("workload estimate %v: %.2f %v/month\n", w["status"], w["totalCost"], w["costCurrency"])

	c.call("DeleteWorkloadEstimate", map[string]any{"identifier": workload})
	c.call("DeleteBillEstimate", map[string]any{"identifier": estimate["id"]})
	c.call("DeleteBillScenario", map[string]any{"identifier": scenario})
	fmt.Println("cleaned up")
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
