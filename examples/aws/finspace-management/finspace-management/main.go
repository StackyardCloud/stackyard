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

	fmt.Printf("Stackyard FinSpace Management advanced client using %s\n", endpoint)

	resourceARN := url.PathEscape("arn:aws:finspace:us-east-1:123456789012:environment/env-000001")
	calls := []restCall{
		{Name: "ListEnvironments", Method: http.MethodGet, Path: "/environment?maxResults=10"},
		{Name: "CreateEnvironment", Method: http.MethodPost, Path: "/environment", Payload: map[string]any{"name": "stackyard-finspace-env", "description": "stackyard environment"}},
		{Name: "GetEnvironment", Method: http.MethodGet, Path: "/environment/env-000001"},
		{Name: "ListKxEnvironments", Method: http.MethodGet, Path: "/kx/environments?maxResults=10"},
		{Name: "GetKxEnvironment", Method: http.MethodGet, Path: "/kx/environments/env-000001"},
		{Name: "TagResource", Method: http.MethodPost, Path: "/tags/" + resourceARN, Payload: map[string]any{"tags": map[string]string{"env": "local"}}},
		{Name: "ListTagsForResource", Method: http.MethodGet, Path: "/tags/" + resourceARN},
	}

	for _, call := range calls {
		if err := runCall(ctx, endpoint, region, creds, call); err != nil {
			exitf("%s failed: %v", call.Name, err)
		}
	}

	fmt.Println("Done.")
}

func runCall(ctx context.Context, endpoint, region string, creds aws.CredentialsProvider, call restCall) error {
	var body []byte
	var err error
	if call.Payload != nil {
		body, err = json.Marshal(call.Payload)
		if err != nil {
			return err
		}
	}

	status, respBody, err := finspaceManagementRequest(ctx, endpoint, region, creds, call.Method, call.Path, body)
	if err != nil {
		return err
	}
	if status < 200 || status >= 300 {
		return fmt.Errorf("HTTP %d: %s", status, strings.TrimSpace(string(respBody)))
	}

	logf("%s returned %d", call.Name, status)
	return nil
}

func finspaceManagementRequest(
	ctx context.Context,
	endpoint, region string,
	creds aws.CredentialsProvider,
	method, path string,
	body []byte,
) (int, []byte, error) {
	if body == nil {
		body = []byte{}
	}

	req, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(endpoint, "/")+path, bytes.NewReader(body))
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
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(body), "finspace", region, time.Now()); err != nil {
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
