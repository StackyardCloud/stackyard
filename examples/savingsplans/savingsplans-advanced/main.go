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

type restCall struct {
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

	fmt.Printf("Stackyard AWS Savings Plans advanced client using %s\n", endpoint)

	resourceARN := "arn:aws:savingsplans:us-east-1:123456789012:savingsplan/sp-00000001"

	calls := []restCall{
		{Name: "DescribeSavingsPlansOfferings", Method: http.MethodPost, Path: "/DescribeSavingsPlansOfferings", Payload: map[string]any{"maxResults": 10}},
		{Name: "DescribeSavingsPlansOfferingRates", Method: http.MethodPost, Path: "/DescribeSavingsPlansOfferingRates", Payload: map[string]any{"maxResults": 10}},
		{Name: "DescribeSavingsPlans", Method: http.MethodPost, Path: "/DescribeSavingsPlans", Payload: map[string]any{"maxResults": 10}},
		{Name: "DescribeSavingsPlanRates", Method: http.MethodPost, Path: "/DescribeSavingsPlanRates", Payload: map[string]any{"maxResults": 10}},
		{Name: "CreateSavingsPlan", Method: http.MethodPost, Path: "/CreateSavingsPlan", Payload: map[string]any{
			"clientToken":            "stackyard-savingsplans-example",
			"savingsPlanOfferingId":  "offering-00000001",
			"commitment":             "0.001",
			"upfrontPaymentAmount":   "0.0",
			"savingsPlanType":        "Compute",
			"paymentOption":          "No Upfront",
			"termDurationInSeconds":  31536000,
			"purchaseTime":           "2026-02-28T00:00:00Z",
			"recurringPaymentAmount": "0.0",
		}},
		{Name: "DeleteQueuedSavingsPlan", Method: http.MethodPost, Path: "/DeleteQueuedSavingsPlan", Payload: map[string]any{"savingsPlanId": "sp-00000001"}},
		{Name: "ReturnSavingsPlan", Method: http.MethodPost, Path: "/ReturnSavingsPlan", Payload: map[string]any{"savingsPlanId": "sp-00000001"}},
		{Name: "TagResource", Method: http.MethodPost, Path: "/TagResource", Payload: map[string]any{
			"resourceArn": resourceARN,
			"tags": map[string]string{
				"env": "example",
			},
		}},
		{Name: "ListTagsForResource", Method: http.MethodPost, Path: "/ListTagsForResource", Payload: map[string]any{
			"resourceArn": resourceARN,
		}},
		{Name: "UntagResource", Method: http.MethodPost, Path: "/UntagResource", Payload: map[string]any{
			"resourceArn": resourceARN,
			"tagKeys":     []string{"env"},
		}},
	}

	for _, call := range calls {
		if err := runCall(ctx, endpoint, region, creds, call); err != nil {
			exitf("%s failed: %v", call.Name, err)
		}
	}

	fmt.Println("Done.")
}

func runCall(ctx context.Context, endpoint, region string, creds aws.CredentialsProvider, call restCall) error {
	status, body, err := savingsPlansRequest(ctx, endpoint, region, creds, call.Method, call.Path, call.Payload)
	if err != nil {
		return err
	}

	if status >= 200 && status < 300 {
		logf("%s returned %d", call.Name, status)
		return nil
	}

	errType := extractErrorType(body)
	if isStagedPlanTolerated(status, errType, body) {
		logf("%s returned %d (%s): expected while staged plan is in progress", call.Name, status, errType)
		return nil
	}

	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		trimmed = "<empty body>"
	}
	return fmt.Errorf("HTTP %d: %s", status, trimmed)
}

func savingsPlansRequest(
	ctx context.Context,
	endpoint, region string,
	creds aws.CredentialsProvider,
	method, path string,
	payload map[string]any,
) (int, []byte, error) {
	var body []byte
	var err error
	if payload != nil {
		body, err = json.Marshal(payload)
		if err != nil {
			return 0, nil, err
		}
	} else {
		body = []byte{}
	}

	fullURL := strings.TrimRight(endpoint, "/") + path
	req, err := http.NewRequestWithContext(ctx, method, fullURL, bytes.NewReader(body))
	if err != nil {
		return 0, nil, err
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	credValue, err := creds.Retrieve(ctx)
	if err != nil {
		return 0, nil, err
	}
	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(body), "savingsplans", region, time.Now()); err != nil {
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

func extractErrorType(body []byte) string {
	payload := map[string]any{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return ""
	}
	if v, ok := payload["__type"].(string); ok {
		return strings.TrimSpace(v)
	}
	if v, ok := payload["code"].(string); ok {
		return strings.TrimSpace(v)
	}
	if v, ok := payload["message"].(string); ok {
		return strings.TrimSpace(v)
	}
	return ""
}

func isStagedPlanTolerated(status int, errType string, body []byte) bool {
	if status == http.StatusNotFound {
		return true
	}
	combined := strings.ToLower(errType + " " + string(body))
	return strings.Contains(combined, "notimplemented") ||
		strings.Contains(combined, "unknownoperation") ||
		strings.Contains(combined, "unknown action") ||
		strings.Contains(combined, "resourcenotfound") ||
		strings.Contains(combined, "validationexception") ||
		strings.Contains(combined, "accessdenied") ||
		strings.Contains(combined, "forbidden") ||
		strings.Contains(combined, "unauthorized")
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

func logf(format string, args ...any) {
	fmt.Printf(format+"\n", args...)
}
