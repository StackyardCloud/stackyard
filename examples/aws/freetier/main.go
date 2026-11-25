// Command freetier runs AWS Free Tier against Stackyard: read the account
// plan and its credits, complete the AWS Budgets sign-up activity by creating
// a budget, list Always Free usage, and upgrade to the paid plan.
//
// The SDK has no Free Tier module in this repository's go.mod, so requests
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

func main() {
	c := newClient()
	fmt.Printf("Stackyard Free Tier client using %s\n", c.endpoint)

	plan := c.call("freetier", "AWSFreeTierService.", "application/x-amz-json-1.0", "GetAccountPlanState", map[string]any{})
	fmt.Printf("plan %v (%v), credits %v USD\n", plan["accountPlanType"], plan["accountPlanStatus"], plan["accountPlanRemainingCredits"].(map[string]any)["amount"])

	// Creating a budget completes the Budgets sign-up activity.
	c.call("budgets", "AWSBudgetServiceGateway.", "application/x-amz-json-1.1", "CreateBudget", map[string]any{
		"AccountId": "123456789012",
		"Budget": map[string]any{
			"BudgetName": fmt.Sprintf("free-tier-%d", time.Now().UnixNano()), "BudgetLimit": map[string]any{"Amount": "10", "Unit": "USD"},
			"TimeUnit": "MONTHLY", "BudgetType": "COST",
		},
	})
	for _, a := range c.call("freetier", "AWSFreeTierService.", "application/x-amz-json-1.0", "ListAccountActivities", map[string]any{})["activities"].([]any) {
		act := a.(map[string]any)
		fmt.Printf("activity %-58v %v\n", act["title"], act["status"])
	}

	usage := c.call("freetier", "AWSFreeTierService.", "application/x-amz-json-1.0", "GetFreeTierUsage", map[string]any{
		"filter": map[string]any{"Dimensions": map[string]any{"Key": "FREE_TIER_TYPE", "Values": []any{"Always Free"}, "MatchOptions": []any{"EQUALS"}}},
	})
	for _, u := range usage["freeTierUsages"].([]any) {
		item := u.(map[string]any)
		fmt.Printf("%-36v %v of %v %v\n", item["service"], item["actualUsageAmount"], item["limit"], item["unit"])
	}

	up := c.call("freetier", "AWSFreeTierService.", "application/x-amz-json-1.0", "UpgradeAccountPlan", map[string]any{"accountPlanType": "PAID"})
	plan = c.call("freetier", "AWSFreeTierService.", "application/x-amz-json-1.0", "GetAccountPlanState", map[string]any{})
	fmt.Printf("upgraded to %v; credits %v USD\n", up["accountPlanType"], plan["accountPlanRemainingCredits"].(map[string]any)["amount"])
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

func decode(op string, raw []byte) map[string]any {
	out := map[string]any{}
	if len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			exitf("%s: decode: %v", op, err)
		}
	}
	return out
}

// call sends one awsJson operation and exits unless it returns 2xx.
func (c client) call(service, prefix, contentType, op string, payload any) map[string]any {
	body, err := json.Marshal(payload)
	if err != nil {
		exitf("%s: %v", op, err)
	}
	return decode(op, c.send(op, http.MethodPost, service, c.endpoint+"/", body, map[string]string{
		"Content-Type": contentType, "X-Amz-Target": prefix + op,
	}))
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
