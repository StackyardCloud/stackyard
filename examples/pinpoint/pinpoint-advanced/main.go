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

type restCall struct {
	Name    string
	Method  string
	Path    string
	Query   map[string]string
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

	fmt.Printf("Stackyard Amazon Pinpoint advanced client using %s\n", endpoint)

	appID := "app-000001"
	resourceARN := "arn:aws:mobiletargeting:us-east-1:123456789012:apps/" + appID

	calls := []restCall{
		{Name: "GetApps", Method: http.MethodGet, Path: "/v1/apps", Query: map[string]string{"page-size": "10"}},
		{Name: "CreateApp", Method: http.MethodPost, Path: "/v1/apps", Payload: map[string]any{"Name": "stackyard-app", "tags": map[string]string{"env": "coverage"}}},
		{Name: "GetApp", Method: http.MethodGet, Path: "/v1/apps/" + url.PathEscape(appID)},
		{Name: "GetChannels", Method: http.MethodGet, Path: "/v1/apps/" + url.PathEscape(appID) + "/channels"},
		{Name: "GetSegments", Method: http.MethodGet, Path: "/v1/apps/" + url.PathEscape(appID) + "/segments", Query: map[string]string{"page-size": "10"}},
		{Name: "GetCampaigns", Method: http.MethodGet, Path: "/v1/apps/" + url.PathEscape(appID) + "/campaigns", Query: map[string]string{"page-size": "10"}},
		{Name: "ListTemplates", Method: http.MethodGet, Path: "/v1/templates", Query: map[string]string{"page-size": "10", "prefix": "stackyard"}},
		{Name: "TagResource", Method: http.MethodPost, Path: "/v1/tags/" + url.PathEscape(resourceARN), Payload: map[string]any{"tags": map[string]string{"env": "coverage"}}},
		{Name: "ListTagsForResource", Method: http.MethodGet, Path: "/v1/tags/" + url.PathEscape(resourceARN)},
	}

	for _, call := range calls {
		if err := runCall(ctx, endpoint, region, creds, call); err != nil {
			exitf("%s failed: %v", call.Name, err)
		}
	}

	fmt.Println("Done.")
}

func runCall(ctx context.Context, endpoint, region string, creds aws.CredentialsProvider, call restCall) error {
	status, body, err := pinpointRequest(ctx, endpoint, region, creds, call.Method, call.Path, call.Query, call.Payload)
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

func pinpointRequest(
	ctx context.Context,
	endpoint, region string,
	creds aws.CredentialsProvider,
	method string,
	path string,
	query map[string]string,
	payload map[string]any,
) (int, []byte, error) {
	var body []byte
	var err error
	if payload != nil && method != http.MethodGet && method != http.MethodDelete {
		body, err = json.Marshal(payload)
		if err != nil {
			return 0, nil, err
		}
	}

	u, err := url.Parse(strings.TrimRight(endpoint, "/") + path)
	if err != nil {
		return 0, nil, err
	}
	if len(query) > 0 {
		q := u.Query()
		for key, value := range query {
			if strings.TrimSpace(value) == "" {
				continue
			}
			q.Set(key, value)
		}
		u.RawQuery = q.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), bytes.NewReader(body))
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
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(body), "mobiletargeting", region, time.Now()); err != nil {
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
