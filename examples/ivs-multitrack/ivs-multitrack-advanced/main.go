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

func main() {
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	region := getenv("AWS_REGION", "us-east-1")

	ctx := context.Background()
	creds := credentials.NewStaticCredentialsProvider(
		getenv("AWS_ACCESS_KEY_ID", "stackyard"),
		getenv("AWS_SECRET_ACCESS_KEY", "stackyard"),
		"",
	)

	fmt.Printf("Stackyard IVS multitrack advanced client using %s\n", endpoint)

	configPayload := map[string]any{
		"client": map[string]any{
			"id":       "client-advanced-0001",
			"name":     "stackyard-broadcast-suite",
			"version":  "2.1.0",
			"platform": "linux",
		},
		"preferences": map[string]any{
			"maxBitrateKbps": 5000,
			"latencyMode":    "LOW",
		},
	}

	status, body, err := apiRequest(ctx, endpoint, region, creds, http.MethodPost, "/api/v3/GetClientConfiguration", configPayload)
	mustSuccess(status, body, err, "GetClientConfiguration")
	fmt.Printf("POST /api/v3/GetClientConfiguration returned %d\n", status)

	status, body, err = apiRequest(ctx, endpoint, region, creds, http.MethodGet, "/api/v2/FindIngest?clientId=client-advanced-0001", nil)
	mustSuccess(status, body, err, "FindIngest")
	fmt.Printf("GET /api/v2/FindIngest returned %d\n", status)

	status, body, err = apiRequest(ctx, endpoint, region, creds, http.MethodPost, "/api/v3/GetClientConfiguration", map[string]any{"clientId": "client-advanced-0001"})
	mustSuccess(status, body, err, "GetClientConfiguration (repeat)")

	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		exitf("unable to parse response JSON: %v", err)
	}
	if _, ok := payload["clientConfigurationStatus"]; !ok {
		exitf("response missing clientConfigurationStatus: %s", strings.TrimSpace(string(body)))
	}
	if _, ok := payload["ingestEndpoint"]; !ok {
		exitf("response missing ingestEndpoint: %s", strings.TrimSpace(string(body)))
	}

	fmt.Println("Done.")
}

func mustSuccess(status int, body []byte, err error, action string) {
	if err != nil {
		exitf("%s request failed: %v", action, err)
	}
	if status < 200 || status >= 300 {
		exitf("%s returned HTTP %d: %s", action, status, strings.TrimSpace(string(body)))
	}
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

	requestURL := strings.TrimRight(endpoint, "/") + requestPath
	req, err := http.NewRequestWithContext(ctx, method, requestURL, bytes.NewReader(body))
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
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(body), "ivs", region, time.Now()); err != nil {
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
