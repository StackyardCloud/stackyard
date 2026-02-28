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

	fmt.Printf("Stackyard Cloud Directory advanced client using %s\n", endpoint)

	resourceARN := "arn:aws:clouddirectory:us-east-1:123456789012:directory/d-00000001"
	calls := []apiCall{
		{
			Name:   "GetDirectory",
			Method: http.MethodPost,
			Path:   "/amazonclouddirectory/2017-01-11/directory/get",
			Payload: map[string]any{
				"DirectoryArn": resourceARN,
			},
		},
		{
			Name:   "CreateFacet",
			Method: http.MethodPut,
			Path:   "/amazonclouddirectory/2017-01-11/facet/create",
			Payload: map[string]any{
				"SchemaArn": "arn:aws:clouddirectory:us-east-1:123456789012:schema/s-00000001",
				"Name":      "stackyard-facet-advanced",
			},
		},
		{
			Name:   "CreateObject",
			Method: http.MethodPut,
			Path:   "/amazonclouddirectory/2017-01-11/object",
			Payload: map[string]any{
				"DirectoryArn": resourceARN,
			},
		},
		{
			Name:   "BatchRead",
			Method: http.MethodPost,
			Path:   "/amazonclouddirectory/2017-01-11/batchread",
			Payload: map[string]any{
				"DirectoryArn":     resourceARN,
				"Operations":       []any{},
				"ConsistencyLevel": "SERIALIZABLE",
			},
		},
		{
			Name:   "TagResource",
			Method: http.MethodPut,
			Path:   "/amazonclouddirectory/2017-01-11/tags/add",
			Payload: map[string]any{
				"ResourceArn": resourceARN,
				"Tags": []any{
					map[string]any{"Key": "env", "Value": "advanced"},
				},
			},
		},
		{
			Name:   "ListTagsForResource",
			Method: http.MethodPost,
			Path:   "/amazonclouddirectory/2017-01-11/tags",
			Payload: map[string]any{
				"ResourceArn": resourceARN,
			},
		},
		{
			Name:   "UntagResource",
			Method: http.MethodPut,
			Path:   "/amazonclouddirectory/2017-01-11/tags/remove",
			Payload: map[string]any{
				"ResourceArn": resourceARN,
				"TagKeys":     []any{"env"},
			},
		},
	}

	for _, call := range calls {
		body, err := json.Marshal(call.Payload)
		if err != nil {
			exitf("%s payload marshal failed: %v", call.Name, err)
		}

		status, respBody, err := cloudDirectoryRequest(ctx, endpoint, region, creds, call.Method, call.Path, body)
		if err != nil {
			exitf("%s request failed: %v", call.Name, err)
		}
		if status < 200 || status >= 300 {
			exitf("%s returned HTTP %d: %s", call.Name, status, strings.TrimSpace(string(respBody)))
		}
		fmt.Printf("%s returned %d\n", call.Name, status)
	}

	fmt.Println("Done.")
}

func cloudDirectoryRequest(
	ctx context.Context,
	endpoint, region string,
	creds aws.CredentialsProvider,
	method, path string,
	payload []byte,
) (int, []byte, error) {
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
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(payload), "clouddirectory", region, time.Now()); err != nil {
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
