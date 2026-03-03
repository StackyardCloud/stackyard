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

type call struct {
	Name   string
	Method string
	Path   string
	Body   []byte
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

	fmt.Printf("Stackyard IoT Events advanced client using %s\n", endpoint)

	calls := []call{
		{Name: "ListInputs", Method: http.MethodGet, Path: "/inputs?maxResults=10"},
		{Name: "ListDetectorModels", Method: http.MethodGet, Path: "/detector-models?maxResults=10"},
		{Name: "ListAlarmModels", Method: http.MethodGet, Path: "/alarm-models?maxResults=10"},
		{Name: "DescribeLoggingOptions", Method: http.MethodGet, Path: "/logging"},
	}

	for _, c := range calls {
		status, body, err := iotEventsRequest(ctx, endpoint, region, creds, c.Method, c.Path, c.Body)
		if err != nil {
			exitf("%s failed: %v", c.Name, err)
		}
		if err := expectOK(c.Name, status, body); err != nil {
			exitf("%v", err)
		}
	}

	fmt.Println("Done.")
}

func iotEventsRequest(
	ctx context.Context,
	endpoint, region string,
	creds aws.CredentialsProvider,
	method, path string,
	payload []byte,
) (int, []byte, error) {
	if payload == nil {
		payload = []byte("{}")
	}

	url := strings.TrimRight(endpoint, "/") + path
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(payload))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	credValue, err := creds.Retrieve(ctx)
	if err != nil {
		return 0, nil, err
	}
	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(payload), "iotevents", region, time.Now()); err != nil {
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

func expectOK(action string, status int, body []byte) error {
	if status >= 200 && status < 300 {
		logf("%s returned %d", action, status)
		return nil
	}

	bodyText := strings.TrimSpace(string(body))
	if bodyText == "" {
		bodyText = "<empty body>"
	}
	return fmt.Errorf("%s returned HTTP %d: %s", action, status, bodyText)
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
