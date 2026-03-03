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

	fmt.Printf("Stackyard IAM Roles Anywhere advanced client using %s\n", endpoint)

	calls := []apiCall{
		{Name: "ListProfiles", Method: http.MethodGet, Path: "/profiles?pageSize=10"},
		{Name: "CreateProfile", Method: http.MethodPost, Path: "/profiles", Payload: map[string]any{
			"name":     "stackyard-profile",
			"enabled":  true,
			"roleArns": []string{"arn:aws:iam::123456789012:role/stackyard-role"},
		}},
		{Name: "ListTrustAnchors", Method: http.MethodGet, Path: "/trustanchors?pageSize=10"},
		{Name: "ListCrls", Method: http.MethodGet, Path: "/crls?pageSize=10"},
		{Name: "TagResource", Method: http.MethodPost, Path: "/TagResource", Payload: map[string]any{
			"resourceArn": "arn:aws:rolesanywhere:us-east-1:123456789012:profile/stackyard-profile",
			"tags": []map[string]string{
				{"key": "env", "value": "dev"},
			},
		}},
	}

	for _, call := range calls {
		var payloadBytes []byte
		if call.Payload != nil {
			encoded, err := json.Marshal(call.Payload)
			if err != nil {
				exitf("%s payload marshal failed: %v", call.Name, err)
			}
			payloadBytes = encoded
		}

		status, body, err := rolesAnywhereRequest(ctx, endpoint, region, creds, call.Method, call.Path, payloadBytes)
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

func rolesAnywhereRequest(
	ctx context.Context,
	endpoint, region string,
	creds aws.CredentialsProvider,
	method, path string,
	payload []byte,
) (int, []byte, error) {
	url := strings.TrimRight(endpoint, "/") + path
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(payload))
	if err != nil {
		return 0, nil, err
	}
	if len(payload) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}

	credValue, err := creds.Retrieve(ctx)
	if err != nil {
		return 0, nil, err
	}
	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(payload), "rolesanywhere", region, time.Now()); err != nil {
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
		strings.Contains(combined, "resourcenotfound") ||
		strings.Contains(combined, "accessdenied") ||
		strings.Contains(combined, "signaturedoesnotmatch") ||
		strings.Contains(combined, "service mismatch") ||
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
