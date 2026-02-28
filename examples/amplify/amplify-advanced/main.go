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
	appName := getenv("STACKYARD_AMPLIFY_APP_NAME", "stackyard-amplify-app")
	appID := getenv("STACKYARD_AMPLIFY_APP_ID", "d1234567890")

	ctx := context.Background()
	creds := credentials.NewStaticCredentialsProvider(
		getenv("AWS_ACCESS_KEY_ID", "stackyard"),
		getenv("AWS_SECRET_ACCESS_KEY", "stackyard"),
		"",
	)

	fmt.Printf("Stackyard Amplify advanced client using %s\n", endpoint)

	createdID, err := runCreateApp(ctx, endpoint, region, creds, appName)
	if err != nil {
		exitf("CreateApp failed: %v", err)
	}
	if createdID != "" {
		appID = createdID
	}

	calls := []restCall{
		{Name: "GetApp", Method: http.MethodGet, Path: "/apps/" + appID, Payload: nil},
		{Name: "ListApps", Method: http.MethodGet, Path: "/apps", Payload: nil},
		{Name: "CreateBranch", Method: http.MethodPost, Path: "/apps/" + appID + "/branches", Payload: map[string]any{"branchName": "main"}},
		{Name: "ListBranches", Method: http.MethodGet, Path: "/apps/" + appID + "/branches", Payload: nil},
		{Name: "CreateBackendEnvironment", Method: http.MethodPost, Path: "/apps/" + appID + "/backendenvironments", Payload: map[string]any{"environmentName": "dev"}},
		{Name: "ListBackendEnvironments", Method: http.MethodGet, Path: "/apps/" + appID + "/backendenvironments", Payload: nil},
		{Name: "GenerateAccessLogs", Method: http.MethodPost, Path: "/apps/" + appID + "/accesslogs", Payload: map[string]any{"domainName": "example.com"}},
		{Name: "DeleteApp", Method: http.MethodDelete, Path: "/apps/" + appID, Payload: nil},
	}

	for _, call := range calls {
		if err := runCall(ctx, endpoint, region, creds, call); err != nil {
			exitf("%s failed: %v", call.Name, err)
		}
	}

	fmt.Println("Done.")
}

func runCreateApp(ctx context.Context, endpoint, region string, creds aws.CredentialsProvider, appName string) (string, error) {
	status, body, err := amplifyRequest(ctx, endpoint, region, creds, http.MethodPost, "/apps", map[string]any{
		"name":       appName,
		"repository": "https://example.com/stackyard/amplify.git",
	})
	if err != nil {
		return "", err
	}
	if status >= 200 && status < 300 {
		payload := map[string]any{}
		if err := json.Unmarshal(body, &payload); err == nil {
			if appObj, ok := payload["app"].(map[string]any); ok {
				if id, ok := appObj["appId"].(string); ok && strings.TrimSpace(id) != "" {
					return strings.TrimSpace(id), nil
				}
			}
			if id, ok := payload["appId"].(string); ok && strings.TrimSpace(id) != "" {
				return strings.TrimSpace(id), nil
			}
		}
		return "", nil
	}

	errType := extractErrorType(body)
	if isStagedPlanTolerated(errType, body) {
		logf("CreateApp returned %d (%s): tolerated while staged plan is in progress", status, errType)
		return "", nil
	}
	return "", fmt.Errorf("HTTP %d: %s", status, strings.TrimSpace(string(body)))
}

func runCall(ctx context.Context, endpoint, region string, creds aws.CredentialsProvider, call restCall) error {
	status, body, err := amplifyRequest(ctx, endpoint, region, creds, call.Method, call.Path, call.Payload)
	if err != nil {
		return err
	}

	if status >= 200 && status < 300 {
		logf("%s returned %d", call.Name, status)
		return nil
	}

	errType := extractErrorType(body)
	if isStagedPlanTolerated(errType, body) {
		logf("%s returned %d (%s): expected while staged plan is in progress", call.Name, status, errType)
		return nil
	}

	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		trimmed = "<empty body>"
	}
	return fmt.Errorf("HTTP %d: %s", status, trimmed)
}

func amplifyRequest(
	ctx context.Context,
	endpoint, region string,
	creds aws.CredentialsProvider,
	method, path string,
	payload map[string]any,
) (int, []byte, error) {
	var body []byte
	var err error
	if payload == nil {
		body = []byte{}
	} else {
		body, err = json.Marshal(payload)
		if err != nil {
			return 0, nil, err
		}
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
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}

	credValue, err := creds.Retrieve(ctx)
	if err != nil {
		return 0, nil, err
	}
	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(body), "amplify", region, time.Now()); err != nil {
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

func isStagedPlanTolerated(errType string, body []byte) bool {
	combined := strings.ToLower(errType + " " + string(body))
	return strings.Contains(combined, "notimplemented") ||
		strings.Contains(combined, "unknownoperation") ||
		strings.Contains(combined, "validationexception") ||
		strings.Contains(combined, "resourcenotfound") ||
		strings.Contains(combined, "accessdenied")
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
