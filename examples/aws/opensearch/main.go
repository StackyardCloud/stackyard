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
	Action  string
	Method  string
	Path    string
	Payload map[string]any
}

func main() {
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	region := getenv("AWS_REGION", "us-east-1")
	domainName := getenv("STACKYARD_DOMAIN_NAME", "opensearch-domain")
	domainArn := getenv("STACKYARD_DOMAIN_ARN", "arn:aws:es:us-east-1:123456789012:domain/opensearch-domain")

	ctx := context.Background()
	creds := credentials.NewStaticCredentialsProvider(
		getenv("AWS_ACCESS_KEY_ID", "stackyard"),
		getenv("AWS_SECRET_ACCESS_KEY", "stackyard"),
		"",
	)

	fmt.Printf("Stackyard OpenSearch advanced client using %s\n", endpoint)

	requests := []requestCase{
		{
			Action: "CreateDomain",
			Method: http.MethodPost,
			Path:   "/2021-01-01/opensearch/domain",
			Payload: map[string]any{
				"DomainName": domainName,
			},
		},
		{
			Action: "UpdateDomainConfig",
			Method: http.MethodPost,
			Path:   "/2021-01-01/opensearch/domain/" + domainName + "/config",
			Payload: map[string]any{
				"ClusterConfig": map[string]any{
					"InstanceType":  "r6g.large.search",
					"InstanceCount": 1,
				},
			},
		},
		{
			Action: "AddTags",
			Method: http.MethodPost,
			Path:   "/2021-01-01/tags",
			Payload: map[string]any{
				"ARN": domainArn,
				"TagList": []map[string]string{
					{"Key": "env", "Value": "dev"},
				},
			},
		},
		{
			Action: "ListTags",
			Method: http.MethodGet,
			Path:   "/2021-01-01/tags?arn=" + url.QueryEscape(domainArn),
		},
		{
			Action: "DeleteDomain",
			Method: http.MethodDelete,
			Path:   "/2021-01-01/opensearch/domain/" + domainName,
		},
	}

	for _, reqCase := range requests {
		status, body, err := opensearchRequest(ctx, endpoint, region, creds, reqCase.Method, reqCase.Path, reqCase.Payload)
		if err != nil {
			exitf("%s request failed: %v", reqCase.Action, err)
		}
		if err := expectSuccess(reqCase.Action, status, body); err != nil {
			exitf("%s response validation failed: %v", reqCase.Action, err)
		}
		logf("%s returned implemented response (%d)", reqCase.Action, status)
	}

	fmt.Println("Done.")
}

func opensearchRequest(
	ctx context.Context,
	endpoint, region string,
	creds aws.CredentialsProvider,
	method, path string,
	payload map[string]any,
) (int, []byte, error) {
	var body []byte
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return 0, nil, err
		}
		body = encoded
	}

	requestURL := strings.TrimRight(endpoint, "/") + path
	req, err := http.NewRequestWithContext(ctx, method, requestURL, bytes.NewReader(body))
	if err != nil {
		return 0, nil, err
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	credentialValue, err := creds.Retrieve(ctx)
	if err != nil {
		return 0, nil, err
	}
	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, credentialValue, req, hashSHA256(body), "es", region, time.Now()); err != nil {
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

func expectSuccess(action string, status int, body []byte) error {
	if status < http.StatusOK || status >= http.StatusMultipleChoices {
		return fmt.Errorf("expected %s to return 2xx, got %d: %s", action, status, strings.TrimSpace(string(body)))
	}

	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return nil
	}

	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return fmt.Errorf("expected JSON response, got: %s", trimmed)
	}

	for _, key := range []string{"__type", "Type", "code", "Code"} {
		raw, ok := payload[key]
		if !ok {
			continue
		}
		value, ok := raw.(string)
		if !ok {
			continue
		}
		if strings.TrimSpace(value) != "" {
			return fmt.Errorf("expected success payload for %s, got error marker %s=%q", action, key, value)
		}
	}

	return nil
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
