package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
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

type apiCall struct {
	Name   string
	Method string
	Path   string
	Query  map[string]string
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

	fmt.Printf("Stackyard CodeGuru Profiler advanced client using %s\n", endpoint)

	calls := []apiCall{
		{Name: "ListProfilingGroups", Method: http.MethodGet, Path: "/profilingGroups"},
		{Name: "DescribeProfilingGroup", Method: http.MethodGet, Path: "/profilingGroups/stackyard-profiling-group"},
		{Name: "ListProfileTimes", Method: http.MethodGet, Path: "/profilingGroups/stackyard-profiling-group/profileTimes"},
		{Name: "GetRecommendations", Method: http.MethodGet, Path: "/internal/profilingGroups/stackyard-profiling-group/recommendations"},
	}

	for _, call := range calls {
		status, body, err := codeGuruProfilerRequest(ctx, endpoint, region, creds, call.Method, call.Path, call.Query, call.Body)
		if err != nil {
			exitf("%s request failed: %v", call.Name, err)
		}
		logf("%s returned %d", call.Name, status)
		trimmed := strings.TrimSpace(string(body))
		if trimmed != "" {
			fmt.Println(trimmed)
		}
	}

	fmt.Println("Done.")
}

func codeGuruProfilerRequest(
	ctx context.Context,
	endpoint, region string,
	creds aws.CredentialsProvider,
	method, path string,
	query map[string]string,
	body []byte,
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

	req, err := http.NewRequestWithContext(ctx, method, base.String(), bytes.NewReader(body))
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
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(body), "codeguru-profiler", region, time.Now()); err != nil {
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

func logf(format string, args ...any) {
	fmt.Printf(format+"\n", args...)
}
