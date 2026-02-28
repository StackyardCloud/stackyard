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
	secretName := getenv("STACKYARD_SECRET_NAME", "stackyard-basic-secret")

	ctx := context.Background()
	creds := credentials.NewStaticCredentialsProvider(
		getenv("AWS_ACCESS_KEY_ID", "stackyard"),
		getenv("AWS_SECRET_ACCESS_KEY", "stackyard"),
		"",
	)

	fmt.Printf("Stackyard Secrets Manager basic client using %s\n", endpoint)

	createStatus, createBody, err := secretsManagerRequest(ctx, endpoint, region, creds, "CreateSecret", map[string]any{
		"Name":               secretName,
		"ClientRequestToken": "secretsmanager-basic-create",
		"SecretString":       "{\"username\":\"stackyard\",\"password\":\"basic\"}",
		"Tags":               []map[string]string{{"Key": "env", "Value": "dev"}},
	})
	if err != nil {
		exitf("create secret: %v", err)
	}
	if err := expectHTTPStatus("CreateSecret", createStatus, http.StatusOK, createBody); err != nil {
		exitf("create secret: %v", err)
	}

	secretARN, err := extractString(createBody, "ARN")
	if err != nil {
		exitf("extract secret arn: %v", err)
	}
	logf("CreateSecret succeeded (%d)", createStatus)

	describeStatus, describeBody, err := secretsManagerRequest(ctx, endpoint, region, creds, "DescribeSecret", map[string]any{
		"SecretId": secretARN,
	})
	if err != nil {
		exitf("describe secret: %v", err)
	}
	if err := expectHTTPStatus("DescribeSecret", describeStatus, http.StatusOK, describeBody); err != nil {
		exitf("describe secret: %v", err)
	}
	logf("DescribeSecret succeeded (%d)", describeStatus)

	getStatus, getBody, err := secretsManagerRequest(ctx, endpoint, region, creds, "GetSecretValue", map[string]any{
		"SecretId": secretARN,
	})
	if err != nil {
		exitf("get secret value: %v", err)
	}
	if err := expectHTTPStatus("GetSecretValue", getStatus, http.StatusOK, getBody); err != nil {
		exitf("get secret value: %v", err)
	}
	logf("GetSecretValue succeeded (%d)", getStatus)

	listStatus, listBody, err := secretsManagerRequest(ctx, endpoint, region, creds, "ListSecrets", map[string]any{
		"MaxResults": 10,
	})
	if err != nil {
		exitf("list secrets: %v", err)
	}
	if err := expectHTTPStatus("ListSecrets", listStatus, http.StatusOK, listBody); err != nil {
		exitf("list secrets: %v", err)
	}
	logf("ListSecrets succeeded (%d)", listStatus)

	fmt.Println("Done.")
}

func secretsManagerRequest(
	ctx context.Context,
	endpoint, region string,
	creds aws.CredentialsProvider,
	action string,
	payload map[string]any,
) (int, []byte, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return 0, nil, err
	}

	requestURL := strings.TrimRight(endpoint, "/") + "/"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, requestURL, bytes.NewReader(body))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "secretsmanager."+action)

	credentialValue, err := creds.Retrieve(ctx)
	if err != nil {
		return 0, nil, err
	}
	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, credentialValue, req, hashSHA256(body), "secretsmanager", region, time.Now()); err != nil {
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

func expectHTTPStatus(action string, status, expected int, body []byte) error {
	if status != expected {
		return fmt.Errorf("expected %s to return %d, got %d: %s", action, expected, status, strings.TrimSpace(string(body)))
	}
	return nil
}

func extractString(body []byte, key string) (string, error) {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", err
	}
	raw, ok := payload[key]
	if !ok {
		return "", fmt.Errorf("missing key %s", key)
	}
	value, ok := raw.(string)
	if !ok || strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("invalid key %s", key)
	}
	return value, nil
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
