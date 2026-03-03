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

	fmt.Printf("Stackyard Amazon Q Business advanced client using %s\n", endpoint)

	calls := []apiCall{
		{Name: "CreateApplication", Method: http.MethodPost, Path: "/applications", Body: []byte(`{"displayName":"example-qbusiness-app"}`)},
		{Name: "CreateIndex", Method: http.MethodPost, Path: "/applications/app-000001/indices", Body: []byte(`{"displayName":"example-index"}`)},
		{Name: "CreateDataSource", Method: http.MethodPost, Path: "/applications/app-000001/indices/idx-000001/datasources", Body: []byte(`{"displayName":"example-ds"}`)},
		{Name: "StartDataSourceSyncJob", Method: http.MethodPost, Path: "/applications/app-000001/indices/idx-000001/datasources/ds-000001/startsync", Body: []byte(`{}`)},
		{Name: "Chat", Method: http.MethodPost, Path: "/applications/app-000001/conversations?userId=user-000001", Body: []byte(`{"userMessage":"hello qbusiness"}`)},
		{Name: "ListConversations", Method: http.MethodGet, Path: "/applications/app-000001/conversations?userId=user-000001"},
		{Name: "ListChatResponseConfigurations", Method: http.MethodGet, Path: "/applications/app-000001/chatresponseconfigurations"},
		{Name: "ListSubscriptions", Method: http.MethodGet, Path: "/applications/app-000001/subscriptions"},
		{Name: "TagResource", Method: http.MethodPost, Path: "/v1/tags/arn%3Aaws%3Aqbusiness%3Aus-east-1%3A123456789012%3Aapplication%2Fapp-000001", Body: []byte(`{"tags":{"env":"example"}}`)},
		{Name: "ListTagsForResource", Method: http.MethodGet, Path: "/v1/tags/arn%3Aaws%3Aqbusiness%3Aus-east-1%3A123456789012%3Aapplication%2Fapp-000001"},
	}

	for _, call := range calls {
		status, body, err := qBusinessRequest(ctx, endpoint, region, creds, call.Method, call.Path, call.Body)
		if err != nil {
			exitf("%s request failed: %v", call.Name, err)
		}
		if status != http.StatusOK {
			exitf("%s expected 200, got %d: %s", call.Name, status, strings.TrimSpace(string(body)))
		}
		if strings.Contains(string(body), "NotImplemented") {
			exitf("%s returned NotImplemented: %s", call.Name, strings.TrimSpace(string(body)))
		}
		logf("%s returned %d", call.Name, status)
		trimmed := strings.TrimSpace(string(body))
		if trimmed != "" {
			fmt.Println(trimmed)
		}
	}

	fmt.Println("Done.")
}

func qBusinessRequest(
	ctx context.Context,
	endpoint, region string,
	creds aws.CredentialsProvider,
	method, path string,
	body []byte,
) (int, []byte, error) {
	base, err := url.Parse(strings.TrimRight(endpoint, "/"))
	if err != nil {
		return 0, nil, err
	}
	cleanPath, rawQuery, _ := strings.Cut(path, "?")
	base.Path = cleanPath
	base.RawQuery = rawQuery

	if body == nil {
		body = []byte{}
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
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(body), "qbusiness", region, time.Now()); err != nil {
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
