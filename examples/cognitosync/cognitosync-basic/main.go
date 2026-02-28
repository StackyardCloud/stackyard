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

type requestCase struct {
	Action string
	Method string
	Path   string
	Query  map[string]string
	Body   map[string]any
}

func main() {
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	region := getenv("AWS_REGION", "us-east-1")
	identityPoolID := getenv("STACKYARD_IDENTITY_POOL_ID", "us-east-1:stackyard-pool")
	identityID := getenv("STACKYARD_IDENTITY_ID", "us-east-1:stackyard-identity")
	datasetName := getenv("STACKYARD_DATASET_NAME", "profile")

	ctx := context.Background()
	creds := credentials.NewStaticCredentialsProvider(
		getenv("AWS_ACCESS_KEY_ID", "stackyard"),
		getenv("AWS_SECRET_ACCESS_KEY", "stackyard"),
		"",
	)

	fmt.Printf("Stackyard Cognito Sync basic client using %s\n", endpoint)

	requests := []requestCase{
		{
			Action: "DescribeIdentityPoolUsage",
			Method: http.MethodGet,
			Path:   fmt.Sprintf("/identitypools/%s", url.PathEscape(identityPoolID)),
		},
		{
			Action: "DescribeIdentityUsage",
			Method: http.MethodGet,
			Path: fmt.Sprintf(
				"/identitypools/%s/identities/%s",
				url.PathEscape(identityPoolID),
				url.PathEscape(identityID),
			),
		},
		{
			Action: "ListIdentityPoolUsage",
			Method: http.MethodGet,
			Path:   "/identitypools",
			Query:  map[string]string{"maxResults": "10"},
		},
		{
			Action: "ListDatasets",
			Method: http.MethodGet,
			Path: fmt.Sprintf(
				"/identitypools/%s/identities/%s/datasets",
				url.PathEscape(identityPoolID),
				url.PathEscape(identityID),
			),
		},
		{
			Action: "DescribeDataset",
			Method: http.MethodGet,
			Path: fmt.Sprintf(
				"/identitypools/%s/identities/%s/datasets/%s",
				url.PathEscape(identityPoolID),
				url.PathEscape(identityID),
				url.PathEscape(datasetName),
			),
		},
		{
			Action: "ListRecords",
			Method: http.MethodGet,
			Path: fmt.Sprintf(
				"/identitypools/%s/identities/%s/datasets/%s/records",
				url.PathEscape(identityPoolID),
				url.PathEscape(identityID),
				url.PathEscape(datasetName),
			),
			Query: map[string]string{"lastSyncCount": "0"},
		},
		{
			Action: "GetCognitoEvents",
			Method: http.MethodGet,
			Path: fmt.Sprintf(
				"/identitypools/%s/events",
				url.PathEscape(identityPoolID),
			),
		},
		{
			Action: "GetIdentityPoolConfiguration",
			Method: http.MethodGet,
			Path: fmt.Sprintf(
				"/identitypools/%s/configuration",
				url.PathEscape(identityPoolID),
			),
		},
		{
			Action: "GetBulkPublishDetails",
			Method: http.MethodPost,
			Path: fmt.Sprintf(
				"/identitypools/%s/getBulkPublishDetails",
				url.PathEscape(identityPoolID),
			),
		},
	}

	for _, request := range requests {
		status, body, err := callCognitoSyncAction(ctx, endpoint, region, creds, request)
		if err != nil {
			exitf("%s request failed: %v", request.Action, err)
		}
		if err := expectPlannedServiceResponse(request.Action, status, body); err != nil {
			exitf("%s response validation failed: %v", request.Action, err)
		}
		logf("%s -> %d %s", request.Action, status, trimForLog(body))
	}

	fmt.Println("Done.")
}

func callCognitoSyncAction(
	ctx context.Context,
	endpoint, region string,
	creds aws.CredentialsProvider,
	request requestCase,
) (int, []byte, error) {
	parsedURL, err := url.Parse(strings.TrimRight(endpoint, "/") + request.Path)
	if err != nil {
		return 0, nil, err
	}

	if len(request.Query) > 0 {
		query := parsedURL.Query()
		for key, value := range request.Query {
			query.Set(key, value)
		}
		parsedURL.RawQuery = query.Encode()
	}

	payload := []byte{}
	if request.Body != nil {
		payload, err = json.Marshal(request.Body)
		if err != nil {
			return 0, nil, err
		}
	}

	req, err := http.NewRequestWithContext(ctx, request.Method, parsedURL.String(), bytes.NewReader(payload))
	if err != nil {
		return 0, nil, err
	}
	if request.Body != nil {
		req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	}

	credentialValue, err := creds.Retrieve(ctx)
	if err != nil {
		return 0, nil, err
	}

	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, credentialValue, req, hashSHA256(payload), "cognito-sync", region, time.Now()); err != nil {
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

func expectPlannedServiceResponse(action string, status int, body []byte) error {
	if status >= http.StatusOK && status < http.StatusMultipleChoices {
		return nil
	}

	switch status {
	case http.StatusBadRequest,
		http.StatusUnauthorized,
		http.StatusForbidden,
		http.StatusNotFound,
		http.StatusMethodNotAllowed,
		http.StatusConflict,
		http.StatusPreconditionFailed,
		http.StatusInternalServerError,
		http.StatusNotImplemented:
		return nil
	}

	message := strings.ToLower(strings.TrimSpace(string(body)))
	if strings.Contains(message, "notimplemented") ||
		strings.Contains(message, "unknownoperation") ||
		strings.Contains(message, "unknown operation") ||
		strings.Contains(message, "not found") {
		return nil
	}

	return fmt.Errorf("unexpected status %d for %s: %s", status, action, trimForLog(body))
}

func trimForLog(body []byte) string {
	text := strings.TrimSpace(string(body))
	if text == "" {
		return "(empty body)"
	}
	if len(text) > 180 {
		return text[:180] + "..."
	}
	return text
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
