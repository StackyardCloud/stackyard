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
	"github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

type apiCall struct {
	Name    string
	Method  string
	Path    string
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

	fmt.Printf("Stackyard Billing Conductor advanced client using %s\n", endpoint)

	pricingRuleOut, err := run(ctx, endpoint, region, creds, apiCall{
		Name:    "CreatePricingRule",
		Method:  http.MethodPost,
		Path:    "/create-pricing-rule",
		Payload: map[string]any{"name": "advanced-pricing-rule"},
	})
	if err != nil {
		exitf("CreatePricingRule failed: %v", err)
	}
	pricingRuleARN := stringField(pricingRuleOut, "PricingRuleArn", "arn:aws:billingconductor:us-east-1:123456789012:pricingrule/pr-000001")

	pricingPlanOut, err := run(ctx, endpoint, region, creds, apiCall{
		Name:    "CreatePricingPlan",
		Method:  http.MethodPost,
		Path:    "/create-pricing-plan",
		Payload: map[string]any{"name": "advanced-pricing-plan"},
	})
	if err != nil {
		exitf("CreatePricingPlan failed: %v", err)
	}
	pricingPlanARN := stringField(pricingPlanOut, "PricingPlanArn", "arn:aws:billingconductor:us-east-1:123456789012:pricingplan/pp-000001")

	billingGroupOut, err := run(ctx, endpoint, region, creds, apiCall{
		Name:    "CreateBillingGroup",
		Method:  http.MethodPost,
		Path:    "/create-billing-group",
		Payload: map[string]any{"name": "advanced-billing-group"},
	})
	if err != nil {
		exitf("CreateBillingGroup failed: %v", err)
	}
	billingGroupARN := stringField(billingGroupOut, "BillingGroupArn", "arn:aws:billingconductor:us-east-1:123456789012:billinggroup/bg-000001")

	customLineItemOut, err := run(ctx, endpoint, region, creds, apiCall{
		Name:    "CreateCustomLineItem",
		Method:  http.MethodPost,
		Path:    "/create-custom-line-item",
		Payload: map[string]any{"name": "advanced-custom-line-item"},
	})
	if err != nil {
		exitf("CreateCustomLineItem failed: %v", err)
	}
	customLineItemARN := stringField(customLineItemOut, "CustomLineItemArn", "arn:aws:billingconductor:us-east-1:123456789012:customlineitem/cli-000001")

	tagPath := "/tags/" + url.PathEscape(billingGroupARN)

	calls := []apiCall{
		{Name: "AssociatePricingRules", Method: http.MethodPut, Path: "/associate-pricing-rules", Payload: map[string]any{"pricingPlanArn": pricingPlanARN, "pricingRuleArns": []string{pricingRuleARN}}},
		{Name: "AssociateAccounts", Method: http.MethodPost, Path: "/associate-accounts", Payload: map[string]any{"billingGroupArn": billingGroupARN, "accountIds": []string{"111122223333"}}},
		{Name: "BatchAssociateResourcesToCustomLineItem", Method: http.MethodPut, Path: "/batch-associate-resources-to-custom-line-item", Payload: map[string]any{"customLineItemArn": customLineItemARN, "resourceArns": []string{"arn:aws:ec2:us-east-1:123456789012:instance/i-1234567890abcdef0"}}},
		{Name: "GetBillingGroupCostReport", Method: http.MethodPost, Path: "/get-billing-group-cost-report", Payload: map[string]any{"billingGroupArn": billingGroupARN}},
		{Name: "ListBillingGroups", Method: http.MethodPost, Path: "/list-billing-groups", Payload: map[string]any{}},
		{Name: "ListPricingPlans", Method: http.MethodPost, Path: "/list-pricing-plans", Payload: map[string]any{}},
		{Name: "ListPricingRules", Method: http.MethodPost, Path: "/list-pricing-rules", Payload: map[string]any{}},
		{Name: "ListCustomLineItems", Method: http.MethodPost, Path: "/list-custom-line-items", Payload: map[string]any{}},
		{Name: "ListResourcesAssociatedToCustomLineItem", Method: http.MethodPost, Path: "/list-resources-associated-to-custom-line-item", Payload: map[string]any{"customLineItemArn": customLineItemARN}},
		{Name: "TagResource", Method: http.MethodPost, Path: tagPath, Payload: map[string]any{"tags": map[string]any{"env": "advanced", "owner": "qa"}}},
		{Name: "ListTagsForResource", Method: http.MethodGet, Path: tagPath, Payload: nil},
		{Name: "UntagResource", Method: http.MethodDelete, Path: tagPath, Payload: map[string]any{"tagKeys": []string{"owner"}}},
		{Name: "DeleteCustomLineItem", Method: http.MethodPost, Path: "/delete-custom-line-item", Payload: map[string]any{"arn": customLineItemARN}},
		{Name: "DeleteBillingGroup", Method: http.MethodPost, Path: "/delete-billing-group", Payload: map[string]any{"arn": billingGroupARN}},
		{Name: "DeletePricingPlan", Method: http.MethodPost, Path: "/delete-pricing-plan", Payload: map[string]any{"arn": pricingPlanARN}},
		{Name: "DeletePricingRule", Method: http.MethodPost, Path: "/delete-pricing-rule", Payload: map[string]any{"arn": pricingRuleARN}},
	}

	for _, call := range calls {
		if _, err := run(ctx, endpoint, region, creds, call); err != nil {
			exitf("%s failed: %v", call.Name, err)
		}
	}

	fmt.Println("Done.")
}

func run(ctx context.Context, endpoint, region string, creds aws.CredentialsProvider, call apiCall) (map[string]any, error) {
	status, body, err := invoke(ctx, endpoint, region, creds, call.Method, call.Path, call.Payload)
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
	if len(strings.TrimSpace(string(body))) > 0 {
		if err := json.Unmarshal(body, &out); err != nil {
			return nil, fmt.Errorf("decode payload: %w", err)
		}
	}
	return out, nil
}

func invoke(ctx context.Context, endpoint, region string, creds aws.CredentialsProvider, method, path string, payload map[string]any) (int, []byte, error) {
	body := []byte{}
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return 0, nil, err
		}
		body = encoded
	}

	req, err := http.NewRequestWithContext(
		ctx,
		method,
		strings.TrimRight(endpoint, "/")+path,
		bytes.NewReader(body),
	)
	if err != nil {
		return 0, nil, err
	}
	if method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch || method == http.MethodDelete {
		req.Header.Set("Content-Type", "application/json")
	}

	credValue, err := creds.Retrieve(ctx)
	if err != nil {
		return 0, nil, err
	}
	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(body), "billingconductor", region, time.Now()); err != nil {
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

func stringField(payload map[string]any, key, def string) string {
	value, _ := payload[key].(string)
	value = strings.TrimSpace(value)
	if value == "" {
		return def
	}
	return value
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
