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
	"github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

type rpcCall struct {
	Name    string
	Action  string
	Payload map[string]any
}

func main() {
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	region := getenv("AWS_REGION", "us-east-1")

	ctx := context.Background()
	creds := credentials.NewStaticCredentialsProvider(
		getenv("AWS_ACCESS_KEY_ID", "stackyard"),
		getenv("AWS_SECRET_ACCESS_KEY", "stackyard"),
		"",
	)

	fmt.Printf("Stackyard AWS Cost Management advanced client using %s\n", endpoint)

	resourceARN := "arn:aws:awscostmanagement:us-east-1:123456789012:anomalymonitor/monitor-000001"
	calls := []rpcCall{
		{Name: "CreateAnomalyMonitor", Action: "CreateAnomalyMonitor", Payload: map[string]any{"monitorName": "stage-monitor"}},
		{Name: "CreateAnomalySubscription", Action: "CreateAnomalySubscription", Payload: map[string]any{"subscriptionName": "stage-subscription"}},
		{Name: "GetAnomalyMonitors", Action: "GetAnomalyMonitors", Payload: map[string]any{}},
		{Name: "GetAnomalySubscriptions", Action: "GetAnomalySubscriptions", Payload: map[string]any{}},
		{Name: "GetCostAndUsage", Action: "GetCostAndUsage", Payload: map[string]any{}},
		{Name: "GetCostForecast", Action: "GetCostForecast", Payload: map[string]any{}},
		{Name: "ListCostCategoryDefinitions", Action: "ListCostCategoryDefinitions", Payload: map[string]any{}},
		{Name: "budgets_CreateBudget", Action: "budgets_CreateBudget", Payload: map[string]any{"BudgetName": "stage-budget"}},
		{Name: "budgets_DescribeBudgets", Action: "budgets_DescribeBudgets", Payload: map[string]any{}},
		{Name: "cur_PutReportDefinition", Action: "cur_PutReportDefinition", Payload: map[string]any{"ReportName": "stage-report"}},
		{Name: "cur_DescribeReportDefinitions", Action: "cur_DescribeReportDefinitions", Payload: map[string]any{}},
		{Name: "pricing_GetProducts", Action: "pricing_GetProducts", Payload: map[string]any{}},
		{Name: "taxSettings_GetTaxRegistration", Action: "taxSettings_GetTaxRegistration", Payload: map[string]any{}},
		{Name: "invoicing_ListInvoiceUnits", Action: "invoicing_ListInvoiceUnits", Payload: map[string]any{}},
		{Name: "billing_ListBillingViews", Action: "billing_ListBillingViews", Payload: map[string]any{}},
		{Name: "DataExports_ListExports", Action: "DataExports_ListExports", Payload: map[string]any{}},
		{Name: "CostOptimizationHub_ListRecommendations", Action: "CostOptimizationHub_ListRecommendations", Payload: map[string]any{}},
		{Name: "TagResource", Action: "TagResource", Payload: map[string]any{"resourceArn": resourceARN, "tags": map[string]any{"env": "advanced", "owner": "qa"}}},
		{Name: "ListTagsForResource", Action: "ListTagsForResource", Payload: map[string]any{"resourceArn": resourceARN}},
		{Name: "UntagResource", Action: "UntagResource", Payload: map[string]any{"resourceArn": resourceARN, "tagKeys": []string{"owner"}}},
		{Name: "DeleteAnomalySubscription", Action: "DeleteAnomalySubscription", Payload: map[string]any{}},
		{Name: "DeleteAnomalyMonitor", Action: "DeleteAnomalyMonitor", Payload: map[string]any{}},
	}

	for _, call := range calls {
		if _, err := runCall(ctx, endpoint, region, creds, call); err != nil {
			exitf("%s failed: %v", call.Name, err)
		}
	}

	fmt.Println("Done.")
}

func runCall(ctx context.Context, endpoint, region string, creds aws.CredentialsProvider, call rpcCall) (map[string]any, error) {
	status, body, err := awsCostManagementRequest(ctx, endpoint, region, creds, call.Action, call.Payload)
	if err != nil {
		return nil, err
	}
	if status < 200 || status >= 300 {
		trimmed := strings.TrimSpace(string(body))
		if trimmed == "" {
			trimmed = "<empty body>"
		}
		return nil, fmt.Errorf("HTTP %d: %s", status, trimmed)
	}

	fmt.Printf("%s returned %d\n", call.Name, status)

	out := map[string]any{}
	if len(strings.TrimSpace(string(body))) == 0 {
		return out, nil
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("decode payload: %w", err)
	}
	return out, nil
}

func awsCostManagementRequest(
	ctx context.Context,
	endpoint, region string,
	creds aws.CredentialsProvider,
	action string,
	payload map[string]any,
) (int, []byte, error) {
	var body []byte
	var err error
	if payload == nil {
		body = []byte("{}")
	} else {
		body, err = json.Marshal(payload)
		if err != nil {
			return 0, nil, err
		}
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		strings.TrimRight(endpoint, "/")+"/",
		bytes.NewReader(body),
	)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "AWSBillingAndCostManagement."+action)

	credValue, err := creds.Retrieve(ctx)
	if err != nil {
		return 0, nil, err
	}
	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(body), "ce", region, time.Now()); err != nil {
		return 0, nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, err
	}
	return resp.StatusCode, respBody, nil
}

func hashSHA256(payload []byte) string {
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

func getenv(key, def string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return def
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
