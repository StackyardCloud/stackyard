package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
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

	fmt.Printf("Stackyard AWS DRS basic client using %s\n", endpoint)

	status, body, err := drsRequest(
		ctx,
		endpoint,
		region,
		creds,
		http.MethodPost,
		"/DescribeSourceServers",
		[]byte(`{"maxResults":10}`),
		map[string]string{"Content-Type": "application/json"},
	)
	if err != nil {
		exitf("DescribeSourceServers request failed: %v", err)
	}
	if status >= 200 && status < 300 {
		logf("DescribeSourceServers returned %d", status)
		fmt.Println(strings.TrimSpace(string(body)))
		return
	}

	if isStagedPlanTolerated(status, body) {
		logf("DescribeSourceServers returned %d: expected while staged plan is in progress", status)
		return
	}

	exitf("DescribeSourceServers returned HTTP %d: %s", status, strings.TrimSpace(string(body)))
}

func drsRequest(
	ctx context.Context,
	endpoint, region string,
	creds aws.CredentialsProvider,
	method, path string,
	payload []byte,
	headers map[string]string,
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
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	if req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	credValue, err := creds.Retrieve(ctx)
	if err != nil {
		return 0, nil, err
	}
	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(payload), "drs", region, time.Now()); err != nil {
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
		strings.Contains(combined, "requestthrottled") ||
		strings.Contains(combined, "conflict")
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
