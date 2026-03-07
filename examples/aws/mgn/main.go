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
	Action  string
	Method  string
	Path    string
	Payload map[string]any
}

func main() {
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	region := getenv("AWS_REGION", "us-east-1")
	accountID := getenv("AWS_ACCOUNT_ID", "123456789012")

	ctx := context.Background()
	creds := credentials.NewStaticCredentialsProvider(
		getenv("AWS_ACCESS_KEY_ID", "stackyard"),
		getenv("AWS_SECRET_ACCESS_KEY", "stackyard"),
		"",
	)

	fmt.Printf("Stackyard MGN advanced client using %s\n", endpoint)

	status, body, err := apiRequest(ctx, endpoint, region, creds, http.MethodPost, "/CreateWave", map[string]any{
		"accountID":   accountID,
		"name":        "stackyard-wave",
		"description": "stackyard migration wave",
	})
	mustSuccess(status, body, err, "CreateWave")
	waveID := extractString(body, "waveID", "wave-00000001")

	status, body, err = apiRequest(ctx, endpoint, region, creds, http.MethodPost, "/CreateApplication", map[string]any{
		"accountID":   accountID,
		"name":        "stackyard-app",
		"description": "stackyard migration app",
	})
	mustSuccess(status, body, err, "CreateApplication")
	applicationID := extractString(body, "applicationID", "app-00000001")

	calls := []apiCall{
		{Action: "ListWaves", Method: http.MethodPost, Path: "/ListWaves", Payload: map[string]any{"accountID": accountID, "maxResults": 10}},
		{Action: "ListApplications", Method: http.MethodPost, Path: "/ListApplications", Payload: map[string]any{"accountID": accountID, "maxResults": 10}},
		{Action: "UpdateWave", Method: http.MethodPost, Path: "/UpdateWave", Payload: map[string]any{"accountID": accountID, "waveID": waveID, "name": "stackyard-wave-updated"}},
		{Action: "UpdateApplication", Method: http.MethodPost, Path: "/UpdateApplication", Payload: map[string]any{"accountID": accountID, "applicationID": applicationID, "name": "stackyard-app-updated"}},
		{Action: "ArchiveApplication", Method: http.MethodPost, Path: "/ArchiveApplication", Payload: map[string]any{"accountID": accountID, "applicationID": applicationID}},
		{Action: "UnarchiveApplication", Method: http.MethodPost, Path: "/UnarchiveApplication", Payload: map[string]any{"accountID": accountID, "applicationID": applicationID}},
		{Action: "DeleteApplication", Method: http.MethodPost, Path: "/DeleteApplication", Payload: map[string]any{"accountID": accountID, "applicationID": applicationID}},
		{Action: "DeleteWave", Method: http.MethodPost, Path: "/DeleteWave", Payload: map[string]any{"accountID": accountID, "waveID": waveID}},
	}

	for _, call := range calls {
		status, body, err = apiRequest(ctx, endpoint, region, creds, call.Method, call.Path, call.Payload)
		mustSuccess(status, body, err, call.Action)
		fmt.Printf("%s returned %d\n", call.Action, status)
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

func extractString(body []byte, key, fallback string) string {
	payload := map[string]any{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return fallback
	}
	if value, ok := payload[key].(string); ok && strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
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
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(body), "mgn", region, time.Now()); err != nil {
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
