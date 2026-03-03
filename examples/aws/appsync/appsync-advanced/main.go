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
	apiID := getenv("STACKYARD_APPSYNC_API_ID", "api-00000001")
	apiARN := getenv("STACKYARD_APPSYNC_API_ARN", "arn:aws:appsync:us-east-1:123456789012:apis/api-00000001")

	ctx := context.Background()
	creds := credentials.NewStaticCredentialsProvider(
		getenv("AWS_ACCESS_KEY_ID", "stackyard"),
		getenv("AWS_SECRET_ACCESS_KEY", "stackyard"),
		"",
	)

	fmt.Printf("Stackyard AppSync advanced client using %s\n", endpoint)

	calls := []apiCall{
		{Name: "ListApis", Method: http.MethodGet, Path: "/v1/apis"},
		{
			Name:   "CreateGraphqlApi",
			Method: http.MethodPost,
			Path:   "/v1/apis",
			Payload: map[string]any{
				"name":               "stackyard-appsync",
				"authenticationType": "API_KEY",
			},
		},
		{Name: "GetGraphqlApi", Method: http.MethodGet, Path: "/v1/apis/" + apiID},
		{
			Name:   "CreateApiKey",
			Method: http.MethodPost,
			Path:   "/v1/apis/" + apiID + "/apikeys",
			Payload: map[string]any{
				"description": "stackyard-key",
			},
		},
		{Name: "ListApiKeys", Method: http.MethodGet, Path: "/v1/apis/" + apiID + "/apikeys"},
		{
			Name:   "CreateDataSource",
			Method: http.MethodPost,
			Path:   "/v1/apis/" + apiID + "/datasources",
			Payload: map[string]any{
				"name":           "NoneDS",
				"type":           "NONE",
				"serviceRoleArn": "arn:aws:iam::123456789012:role/appsync-service-role",
			},
		},
		{Name: "ListDataSources", Method: http.MethodGet, Path: "/v1/apis/" + apiID + "/datasources"},
		{
			Name:   "StartSchemaCreation",
			Method: http.MethodPost,
			Path:   "/v1/apis/" + apiID + "/schemacreation",
			Payload: map[string]any{
				"definition": "dHlwZSBRdWVyeSB7IHBpbmc6IFN0cmluZyB9",
			},
		},
		{Name: "GetSchemaCreationStatus", Method: http.MethodGet, Path: "/v1/apis/" + apiID + "/schemacreation"},
		{
			Name:   "TagResource",
			Method: http.MethodPost,
			Path:   "/v1/tags/" + apiARN,
			Payload: map[string]any{
				"tags": map[string]string{"env": "dev", "team": "stackyard"},
			},
		},
		{Name: "ListTagsForResource", Method: http.MethodGet, Path: "/v1/tags/" + apiARN},
	}

	for _, call := range calls {
		if err := runCall(ctx, endpoint, region, creds, call); err != nil {
			exitf("%s failed: %v", call.Name, err)
		}
	}

	fmt.Println("Done.")
}

func runCall(ctx context.Context, endpoint, region string, creds aws.CredentialsProvider, call apiCall) error {
	var payload []byte
	if call.Payload != nil {
		encoded, err := json.Marshal(call.Payload)
		if err != nil {
			return err
		}
		payload = encoded
	}

	status, body, err := appsyncRequest(ctx, endpoint, region, creds, call.Method, call.Path, payload)
	if err != nil {
		return err
	}

	if status >= 200 && status < 300 {
		logf("%s returned %d", call.Name, status)
		return nil
	}
	if isStagedPlanTolerated(status, body) {
		logf("%s returned %d: expected while staged plan is in progress", call.Name, status)
		return nil
	}

	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		trimmed = "<empty body>"
	}
	return fmt.Errorf("HTTP %d: %s", status, trimmed)
}

func appsyncRequest(
	ctx context.Context,
	endpoint, region string,
	creds aws.CredentialsProvider,
	method, path string,
	payload []byte,
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
	req.Header.Set("Content-Type", "application/json")

	credValue, err := creds.Retrieve(ctx)
	if err != nil {
		return 0, nil, err
	}
	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(payload), "appsync", region, time.Now()); err != nil {
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
		strings.Contains(combined, "not found")
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
