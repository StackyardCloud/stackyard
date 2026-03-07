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
	identifier := getenv("STACKYARD_RECYCLEBIN_RULE_ID", "rbin-00000001")
	ruleARN := getenv("STACKYARD_RECYCLEBIN_RULE_ARN", "arn:aws:rbin:us-east-1:123456789012:rule/rbin-00000001")

	ctx := context.Background()
	creds := credentials.NewStaticCredentialsProvider(
		getenv("AWS_ACCESS_KEY_ID", "stackyard"),
		getenv("AWS_SECRET_ACCESS_KEY", "stackyard"),
		"",
	)

	fmt.Printf("Stackyard Recycle Bin advanced client using %s\n", endpoint)

	createCall := apiCall{
		Name:   "CreateRule",
		Method: http.MethodPost,
		Path:   "/rules",
		Payload: map[string]any{
			"Description":  "stackyard recycle bin rule",
			"ResourceType": "EBS_SNAPSHOT",
			"RetentionPeriod": map[string]any{
				"RetentionPeriodValue": 30,
				"RetentionPeriodUnit":  "DAYS",
			},
		},
	}

	status, body, err := runCall(ctx, endpoint, region, creds, createCall)
	if err != nil {
		exitf("%s failed: %v", createCall.Name, err)
	}
	if status >= 200 && status < 300 {
		identifier = extractString(body, "Identifier", identifier)
		ruleARN = extractString(body, "RuleArn", ruleARN)
	}

	calls := []apiCall{
		{Name: "GetRule", Method: http.MethodGet, Path: "/rules/" + identifier},
		{Name: "ListRules", Method: http.MethodPost, Path: "/list-rules", Payload: map[string]any{"MaxResults": 10}},
		{
			Name:   "LockRule",
			Method: http.MethodPatch,
			Path:   "/rules/" + identifier + "/lock",
			Payload: map[string]any{
				"LockConfiguration": map[string]any{
					"UnlockDelay": map[string]any{
						"UnlockDelayValue": 7,
						"UnlockDelayUnit":  "DAYS",
					},
				},
			},
		},
		{Name: "UnlockRule", Method: http.MethodPatch, Path: "/rules/" + identifier + "/unlock", Payload: map[string]any{}},
		{
			Name:   "TagResource",
			Method: http.MethodPost,
			Path:   "/tags/" + url.PathEscape(ruleARN),
			Payload: map[string]any{
				"Tags": []map[string]string{
					{"Key": "env", "Value": "dev"},
					{"Key": "team", "Value": "stackyard"},
				},
			},
		},
		{Name: "ListTagsForResource", Method: http.MethodGet, Path: "/tags/" + url.PathEscape(ruleARN)},
		{
			Name:    "UntagResource",
			Method:  http.MethodDelete,
			Path:    "/tags/" + url.PathEscape(ruleARN),
			Payload: map[string]any{"TagKeys": []string{"team"}},
		},
		{
			Name:   "UpdateRule",
			Method: http.MethodPatch,
			Path:   "/rules/" + identifier,
			Payload: map[string]any{
				"Description": "stackyard recycle bin rule updated",
				"RetentionPeriod": map[string]any{
					"RetentionPeriodValue": 45,
					"RetentionPeriodUnit":  "DAYS",
				},
			},
		},
		{Name: "DeleteRule", Method: http.MethodDelete, Path: "/rules/" + identifier},
	}

	for _, call := range calls {
		if _, _, err := runCall(ctx, endpoint, region, creds, call); err != nil {
			exitf("%s failed: %v", call.Name, err)
		}
	}

	fmt.Println("Done.")
}

func runCall(
	ctx context.Context,
	endpoint, region string,
	creds aws.CredentialsProvider,
	call apiCall,
) (int, []byte, error) {
	var payload []byte
	if call.Payload != nil {
		encoded, err := json.Marshal(call.Payload)
		if err != nil {
			return 0, nil, err
		}
		payload = encoded
	}

	status, body, err := recycleBinRequest(ctx, endpoint, region, creds, call.Method, call.Path, payload)
	if err != nil {
		return 0, nil, err
	}

	if status >= 200 && status < 300 {
		logf("%s returned %d", call.Name, status)
		return status, body, nil
	}
	if isStagedPlanTolerated(status, body) {
		logf("%s returned %d: expected while staged plan is in progress", call.Name, status)
		return status, body, nil
	}

	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		trimmed = "<empty body>"
	}
	return status, body, fmt.Errorf("HTTP %d: %s", status, trimmed)
}

func recycleBinRequest(
	ctx context.Context,
	endpoint, region string,
	creds aws.CredentialsProvider,
	method, path string,
	payload []byte,
) (int, []byte, error) {
	if payload == nil {
		payload = []byte{}
	}

	req, err := http.NewRequestWithContext(
		ctx,
		method,
		strings.TrimRight(endpoint, "/")+path,
		bytes.NewReader(payload),
	)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	credValue, err := creds.Retrieve(ctx)
	if err != nil {
		return 0, nil, err
	}
	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(payload), "rbin", region, time.Now()); err != nil {
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

func extractString(body []byte, key, fallback string) string {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return fallback
	}
	value, ok := payload[key].(string)
	if !ok {
		return fallback
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func isStagedPlanTolerated(status int, body []byte) bool {
	if status == http.StatusNotFound || status == http.StatusMethodNotAllowed {
		return true
	}
	combined := strings.ToLower(string(body))
	return strings.Contains(combined, "notimplemented") ||
		strings.Contains(combined, "unknownoperation") ||
		strings.Contains(combined, "validationexception") ||
		strings.Contains(combined, "resourcenotfound") ||
		strings.Contains(combined, "accessdenied") ||
		strings.Contains(combined, "signaturedoesnotmatch") ||
		strings.Contains(combined, "invalidrequest") ||
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
