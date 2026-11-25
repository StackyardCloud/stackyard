// Command budgets runs AWS Budgets against Stackyard: create a monthly cost
// budget with an 80% alert, add a subscriber, attach a budget action that
// applies an IAM deny policy, approve and reverse it, and read its history.
//
// Stackyard records no billing data, so actual spend is always zero. The SDK
// has no Budgets module in this repository's go.mod, so requests are signed by
// hand.
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
	signingName  = "budgets"
	targetPrefix = "AWSBudgetServiceGateway."
	contentType  = "application/x-amz-json-1.1"
	account      = "123456789012"
)

func main() {
	c := newClient()
	fmt.Printf("Stackyard Budgets client using %s\n", c.endpoint)

	name := fmt.Sprintf("team-monthly-%d", time.Now().UnixNano())
	alert := map[string]any{"NotificationType": "ACTUAL", "ComparisonOperator": "GREATER_THAN", "Threshold": 80, "ThresholdType": "PERCENTAGE"}
	email := map[string]any{"SubscriptionType": "EMAIL", "Address": "finance@example.com"}
	c.call("CreateBudget", map[string]any{
		"AccountId": account,
		"Budget": map[string]any{
			"BudgetName": name, "BudgetLimit": map[string]any{"Amount": "500", "Unit": "USD"},
			"TimeUnit": "MONTHLY", "BudgetType": "COST",
		},
		"NotificationsWithSubscribers": []any{map[string]any{"Notification": alert, "Subscribers": []any{email}}},
		"ResourceTags":                 []any{map[string]any{"Key": "team", "Value": "payments"}},
	})
	budget := c.call("DescribeBudget", map[string]any{"AccountId": account, "BudgetName": name})["Budget"].(map[string]any)
	spend := budget["CalculatedSpend"].(map[string]any)["ActualSpend"].(map[string]any)
	fmt.Printf("budget %s: limit %v USD, actual spend %v %v\n", name, budget["BudgetLimit"].(map[string]any)["Amount"], spend["Amount"], spend["Unit"])

	c.call("CreateSubscriber", map[string]any{
		"AccountId": account, "BudgetName": name, "Notification": alert,
		"Subscriber": map[string]any{"SubscriptionType": "SNS", "Address": "arn:aws:sns:us-east-1:" + account + ":budget-alerts"},
	})
	subs := c.call("DescribeSubscribersForNotification", map[string]any{"AccountId": account, "BudgetName": name, "Notification": alert})["Subscribers"].([]any)
	for _, s := range subs {
		fmt.Printf("subscriber: %v %v\n", s.(map[string]any)["SubscriptionType"], s.(map[string]any)["Address"])
	}

	id := c.call("CreateBudgetAction", map[string]any{
		"AccountId": account, "BudgetName": name, "NotificationType": "ACTUAL", "ActionType": "APPLY_IAM_POLICY",
		"ActionThreshold": map[string]any{"ActionThresholdValue": 100, "ActionThresholdType": "PERCENTAGE"},
		"Definition": map[string]any{"IamActionDefinition": map[string]any{
			"PolicyArn": "arn:aws:iam::aws:policy/AWSDenyAll", "Roles": []any{"payments-dev"},
		}},
		"ExecutionRoleArn": "arn:aws:iam::" + account + ":role/budgets-actions", "ApprovalModel": "MANUAL",
		"Subscribers": []any{email},
	})["ActionId"].(string)
	ref := map[string]any{"AccountId": account, "BudgetName": name, "ActionId": id}
	for _, exec := range []string{"APPROVE_BUDGET_ACTION", "REVERSE_BUDGET_ACTION"} {
		c.call("ExecuteBudgetAction", with(ref, map[string]any{"ExecutionType": exec}))
		action := c.call("DescribeBudgetAction", ref)["Action"].(map[string]any)
		fmt.Printf("%s -> %v\n", exec, action["Status"])
	}
	for _, h := range c.call("DescribeBudgetActionHistories", ref)["ActionHistories"].([]any) {
		fmt.Printf("history: %v %v\n", h.(map[string]any)["EventType"], h.(map[string]any)["Status"])
	}

	c.call("DeleteBudget", map[string]any{"AccountId": account, "BudgetName": name})
	fmt.Println("deleted budget", name)
}

func with(base, extra map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range base {
		out[k] = v
	}
	for k, v := range extra {
		out[k] = v
	}
	return out
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
