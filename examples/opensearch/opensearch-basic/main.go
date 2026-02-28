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

func main() {
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	region := getenv("AWS_REGION", "us-east-1")
	domainName := getenv("STACKYARD_DOMAIN_NAME", "opensearch-basic-domain")

	ctx := context.Background()
	creds := credentials.NewStaticCredentialsProvider(
		getenv("AWS_ACCESS_KEY_ID", "stackyard"),
		getenv("AWS_SECRET_ACCESS_KEY", "stackyard"),
		"",
	)

	fmt.Printf("Stackyard OpenSearch basic client using %s\n", endpoint)

	listStatus, listBody, err := opensearchRequest(ctx, endpoint, region, creds, http.MethodGet, "/2021-01-01/domain", nil)
	if err != nil {
		exitf("list domain names: %v", err)
	}
	if err := expectNotImplemented("ListDomainNames", listStatus, listBody); err != nil {
		exitf("list domain names: %v", err)
	}
	logf("ListDomainNames returned stage-0 baseline response (%d)", listStatus)

	describePath := "/2021-01-01/opensearch/domain/" + domainName
	describeStatus, describeBody, err := opensearchRequest(ctx, endpoint, region, creds, http.MethodGet, describePath, nil)
	if err != nil {
		exitf("describe domain: %v", err)
	}
	if err := expectNotImplemented("DescribeDomain", describeStatus, describeBody); err != nil {
		exitf("describe domain: %v", err)
	}
	logf("DescribeDomain returned stage-0 baseline response (%d)", describeStatus)

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

	url := strings.TrimRight(endpoint, "/") + path
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
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

func expectNotImplemented(action string, status int, body []byte) error {
	if status != http.StatusNotImplemented {
		return fmt.Errorf("expected %s to return %d, got %d: %s", action, http.StatusNotImplemented, status, strings.TrimSpace(string(body)))
	}

	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		if strings.Contains(string(body), "NotImplemented") {
			return nil
		}
		return fmt.Errorf("expected NotImplemented response, got non-JSON payload: %s", strings.TrimSpace(string(body)))
	}

	for _, key := range []string{"__type", "Type"} {
		if raw, ok := payload[key]; ok {
			if value, ok := raw.(string); ok && strings.Contains(value, "NotImplemented") {
				return nil
			}
		}
	}
	return fmt.Errorf("expected NotImplemented marker in %s response: %s", action, strings.TrimSpace(string(body)))
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
