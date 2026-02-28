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

	fmt.Printf("Stackyard Timestream for InfluxDB basic client using %s\n", endpoint)

	actions := []string{
		"ListDbClusters",
		"ListDbInstances",
		"ListDbParameterGroups",
	}

	for _, action := range actions {
		status, body, err := callTimestreamInfluxDBAction(ctx, endpoint, region, creds, action, map[string]any{})
		if err != nil {
			exitf("%s request failed: %v", action, err)
		}
		if err := expectListSuccess(action, status, body); err != nil {
			exitf("%s response validation failed: %v", action, err)
		}
		logf("%s returned implemented stage-1 response (%d)", action, status)
	}

	fmt.Println("Done.")
}

func callTimestreamInfluxDBAction(
	ctx context.Context,
	endpoint, region string,
	creds aws.CredentialsProvider,
	action string,
	payload map[string]any,
) (int, []byte, error) {
	encodedPayload, err := json.Marshal(payload)
	if err != nil {
		return 0, nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(endpoint, "/")+"/", bytes.NewReader(encodedPayload))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/x-amz-json-1.0")
	req.Header.Set("X-Amz-Target", "AmazonTimestreamInfluxDB."+action)

	credentialValue, err := creds.Retrieve(ctx)
	if err != nil {
		return 0, nil, err
	}
	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, credentialValue, req, hashSHA256(encodedPayload), "timestream-influxdb", region, time.Now()); err != nil {
		return 0, nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, err
	}
	return resp.StatusCode, body, nil
}

func expectListSuccess(action string, status int, body []byte) error {
	if status != http.StatusOK {
		return fmt.Errorf("expected %s to return %d, got %d: %s", action, http.StatusOK, status, strings.TrimSpace(string(body)))
	}

	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return fmt.Errorf("expected JSON response, got: %s", strings.TrimSpace(string(body)))
	}
	if _, ok := payload["items"]; !ok {
		return fmt.Errorf("expected items field in %s response: %s", action, strings.TrimSpace(string(body)))
	}
	return nil
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
