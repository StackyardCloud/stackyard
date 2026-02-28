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

func main() {
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	region := getenv("AWS_REGION", "us-east-1")

	ctx := context.Background()
	creds := credentials.NewStaticCredentialsProvider(
		getenv("AWS_ACCESS_KEY_ID", "stackyard"),
		getenv("AWS_SECRET_ACCESS_KEY", "stackyard"),
		"",
	)

	fmt.Printf("Stackyard HealthImaging basic client using %s\n", endpoint)

	status, body, err := healthImagingRequest(
		ctx,
		endpoint,
		region,
		creds,
		http.MethodGet,
		"/datastore",
		nil,
		nil,
	)
	if err != nil {
		exitf("request failed: %v", err)
	}

	if status >= 200 && status < 300 {
		logf("ListDatastores returned %d", status)
		fmt.Println(strings.TrimSpace(string(body)))
		return
	}

	errType := extractErrorType(body)
	if isExpectedStagedError(errType) {
		logf("ListDatastores returned %d (%s): expected while staged plan is in progress", status, errType)
		return
	}

	exitf("ListDatastores returned HTTP %d: %s", status, strings.TrimSpace(string(body)))
}

func healthImagingRequest(
	ctx context.Context,
	endpoint, region string,
	creds aws.CredentialsProvider,
	method, path string,
	query map[string]string,
	payload map[string]any,
) (int, []byte, error) {
	base, err := url.Parse(strings.TrimRight(endpoint, "/"))
	if err != nil {
		return 0, nil, err
	}
	base.Path = path
	if len(query) > 0 {
		q := base.Query()
		for k, v := range query {
			q.Set(k, v)
		}
		base.RawQuery = q.Encode()
	}

	var bodyBytes []byte
	if payload != nil {
		bodyBytes, err = json.Marshal(payload)
		if err != nil {
			return 0, nil, err
		}
	} else {
		bodyBytes = []byte{}
	}

	req, err := http.NewRequestWithContext(ctx, method, base.String(), bytes.NewReader(bodyBytes))
	if err != nil {
		return 0, nil, err
	}
	if method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch {
		req.Header.Set("Content-Type", "application/json")
	}

	credValue, err := creds.Retrieve(ctx)
	if err != nil {
		return 0, nil, err
	}
	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(bodyBytes), "medical-imaging", region, time.Now()); err != nil {
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
	if v, ok := payload["error"].(string); ok {
		return strings.TrimSpace(v)
	}
	return ""
}

func isExpectedStagedError(errType string) bool {
	switch errType {
	case "NotImplemented", "NotImplementedException", "ResourceNotFoundException":
		return true
	default:
		return false
	}
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
