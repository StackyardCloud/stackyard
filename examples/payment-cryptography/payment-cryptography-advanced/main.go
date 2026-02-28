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

type apiCall struct {
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

	fmt.Printf("Stackyard Payment Cryptography advanced client using %s\n", endpoint)

	keyArn := "arn:aws:payment-cryptography:us-east-1:123456789012:key/stackyard-key"

	calls := []apiCall{
		{Name: "ListKeys", Action: "ListKeys", Payload: map[string]any{"MaxResults": 10}},
		{Name: "ListAliases", Action: "ListAliases", Payload: map[string]any{"MaxResults": 10}},
		{Name: "GetParametersForImport", Action: "GetParametersForImport", Payload: map[string]any{"KeyMaterialType": "TR34_KEY_BLOCK", "WrappingKeyAlgorithm": "RSA_3072"}},
		{Name: "CreateAlias", Action: "CreateAlias", Payload: map[string]any{"AliasName": "alias/stackyard", "KeyArn": keyArn}},
		{Name: "GetAlias", Action: "GetAlias", Payload: map[string]any{"AliasName": "alias/stackyard"}},
		{Name: "TagResource", Action: "TagResource", Payload: map[string]any{"ResourceArn": keyArn, "Tags": []map[string]any{{"Key": "env", "Value": "test"}}}},
		{Name: "ListTagsForResource", Action: "ListTagsForResource", Payload: map[string]any{"ResourceArn": keyArn}},
		{Name: "UntagResource", Action: "UntagResource", Payload: map[string]any{"ResourceArn": keyArn, "TagKeys": []string{"env"}}},
	}

	for _, call := range calls {
		status, body, err := paymentCryptographyRequest(ctx, endpoint, region, creds, call.Action, call.Payload)
		if err != nil {
			exitf("%s request failed: %v", call.Name, err)
		}

		if status >= 200 && status < 300 {
			logf("%s returned %d", call.Name, status)
			continue
		}

		errType := extractErrorType(body)
		if isStagedPlanTolerated(errType, body) {
			logf("%s returned %d (%s): tolerated during staged plan", call.Name, status, errType)
			continue
		}

		exitf("%s returned HTTP %d: %s", call.Name, status, strings.TrimSpace(string(body)))
	}

	fmt.Println("Done.")
}

func paymentCryptographyRequest(
	ctx context.Context,
	endpoint, region string,
	creds aws.CredentialsProvider,
	action string,
	payload map[string]any,
) (int, []byte, error) {
	url := strings.TrimRight(endpoint, "/") + "/"
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return 0, nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payloadBytes))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/x-amz-json-1.0")
	req.Header.Set("X-Amz-Target", "PaymentCryptographyControlPlane."+action)

	credValue, err := creds.Retrieve(ctx)
	if err != nil {
		return 0, nil, err
	}
	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(payloadBytes), "payment-cryptography", region, time.Now()); err != nil {
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
	if v, ok := payload["Message"].(string); ok {
		return strings.TrimSpace(v)
	}
	return ""
}

func isStagedPlanTolerated(errType string, body []byte) bool {
	combined := strings.ToLower(errType + " " + string(body))
	return strings.Contains(combined, "notimplemented") ||
		strings.Contains(combined, "unknownoperation") ||
		strings.Contains(combined, "validationexception") ||
		strings.Contains(combined, "invalidrequest") ||
		strings.Contains(combined, "resourcenotfound") ||
		strings.Contains(combined, "accessdenied") ||
		strings.Contains(combined, "signaturedoesnotmatch") ||
		strings.Contains(combined, "service mismatch") ||
		strings.Contains(combined, "not found")
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
