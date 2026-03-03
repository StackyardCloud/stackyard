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

	fmt.Printf("Stackyard FinSpace advanced client using %s\n", endpoint)

	calls := []restCall{
		{Name: "ListDatasets", Method: http.MethodGet, Path: "/datasetsv2"},
		{Name: "CreateDataset", Method: http.MethodPost, Path: "/datasetsv2", Payload: map[string]any{"datasetTitle": "stackyard-finspace-dataset", "kind": "TABULAR", "description": "stackyard dataset"}},
		{Name: "GetDataset", Method: http.MethodGet, Path: "/datasetsv2/dataset-000001"},
		{Name: "ListUsers", Method: http.MethodGet, Path: "/user"},
		{Name: "GetUser", Method: http.MethodGet, Path: "/user/user-000001"},
		{Name: "ListPermissionGroups", Method: http.MethodGet, Path: "/permission-group"},
		{Name: "GetWorkingLocation", Method: http.MethodPost, Path: "/workingLocationV1", Payload: map[string]any{"locationType": "SAGEMAKER"}},
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

	status, respBody, err := finspaceRequest(ctx, endpoint, region, creds, call.Method, call.Path, body)
	if err != nil {
		return err
	}
	if status < 200 || status >= 300 {
		return fmt.Errorf("HTTP %d: %s", status, strings.TrimSpace(string(respBody)))
	}

	logf("%s returned %d", call.Name, status)
	return nil
}

func finspaceRequest(
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
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(body), "finspace-api", region, time.Now()); err != nil {
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
