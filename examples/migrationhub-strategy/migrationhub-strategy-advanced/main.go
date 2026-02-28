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

	fmt.Printf("Stackyard Migration Hub Strategy advanced client using %s\n", endpoint)

	status, body, err := apiRequest(ctx, endpoint, region, creds, http.MethodPost, "/start-assessment", map[string]any{})
	if err != nil {
		exitf("StartAssessment request failed: %v", err)
	}
	if status < 200 || status >= 300 {
		exitf("StartAssessment returned HTTP %d: %s", status, strings.TrimSpace(string(body)))
	}
	assessmentID := extractID(body, "assessment-00000001")

	calls := []apiCall{
		{Method: http.MethodGet, Path: "/get-assessment/" + assessmentID},
		{Method: http.MethodGet, Path: "/get-latest-assessment-id"},
		{Method: http.MethodPost, Path: "/list-servers", Payload: map[string]any{"maxResults": 10}},
		{Method: http.MethodGet, Path: "/get-server-details/server-00000001"},
		{Method: http.MethodGet, Path: "/get-server-strategies/server-00000001"},
		{Method: http.MethodPost, Path: "/put-portfolio-preferences", Payload: map[string]any{"applicationMode": "ALL"}},
		{Method: http.MethodGet, Path: "/get-portfolio-summary"},
		{Method: http.MethodPost, Path: "/start-import-file-task", Payload: map[string]any{}},
		{Method: http.MethodGet, Path: "/list-import-file-task"},
		{Method: http.MethodPost, Path: "/start-recommendation-report-generation", Payload: map[string]any{}},
	}

	for _, call := range calls {
		status, body, err := apiRequest(ctx, endpoint, region, creds, call.Method, call.Path, call.Payload)
		if err != nil {
			exitf("%s %s request failed: %v", call.Method, call.Path, err)
		}
		if status < 200 || status >= 300 {
			exitf("%s %s returned HTTP %d: %s", call.Method, call.Path, status, strings.TrimSpace(string(body)))
		}
		fmt.Printf("%s %s returned %d\n", call.Method, call.Path, status)
	}

	fmt.Println("Done.")
}

func extractID(body []byte, fallback string) string {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return fallback
	}
	if v, ok := payload["id"].(string); ok && strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}
	return fallback
}

func apiRequest(
	ctx context.Context,
	endpoint, region string,
	creds aws.CredentialsProvider,
	method, requestPath string,
	payload map[string]any,
) (int, []byte, error) {
	var body []byte
	var err error
	if payload != nil {
		body, err = json.Marshal(payload)
		if err != nil {
			return 0, nil, err
		}
	}

	url := strings.TrimRight(endpoint, "/") + requestPath
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Accept", "application/json")
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}

	credValue, err := creds.Retrieve(ctx)
	if err != nil {
		return 0, nil, err
	}
	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(body), "migrationhub-strategy", region, time.Now()); err != nil {
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
