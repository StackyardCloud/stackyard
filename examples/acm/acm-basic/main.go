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
	domainName := getenv("STACKYARD_DOMAIN_NAME", "acm-basic.example.com")

	ctx := context.Background()
	creds := credentials.NewStaticCredentialsProvider(
		getenv("AWS_ACCESS_KEY_ID", "stackyard"),
		getenv("AWS_SECRET_ACCESS_KEY", "stackyard"),
		"",
	)

	fmt.Printf("Stackyard ACM basic client using %s\n", endpoint)

	requestStatus, requestBody, err := acmRequest(ctx, endpoint, region, creds, "RequestCertificate", map[string]any{
		"DomainName":       domainName,
		"ValidationMethod": "DNS",
		"IdempotencyToken": "acmbasic",
	})
	if err != nil {
		exitf("request certificate: %v", err)
	}
	if err := expectHTTPStatus("RequestCertificate", requestStatus, http.StatusOK, requestBody); err != nil {
		exitf("request certificate: %v", err)
	}
	certificateARN, err := extractString(requestBody, "CertificateArn")
	if err != nil {
		exitf("request certificate response parse: %v", err)
	}
	logf("requested certificate: %s", certificateARN)

	listStatus, listBody, err := acmRequest(ctx, endpoint, region, creds, "ListCertificates", map[string]any{"MaxItems": 10})
	if err != nil {
		exitf("list certificates: %v", err)
	}
	if err := expectHTTPStatus("ListCertificates", listStatus, http.StatusOK, listBody); err != nil {
		exitf("list certificates: %v", err)
	}
	logf("ListCertificates succeeded (%d)", listStatus)

	describeStatus, describeBody, err := acmRequest(ctx, endpoint, region, creds, "DescribeCertificate", map[string]any{
		"CertificateArn": certificateARN,
	})
	if err != nil {
		exitf("describe certificate: %v", err)
	}
	if err := expectHTTPStatus("DescribeCertificate", describeStatus, http.StatusOK, describeBody); err != nil {
		exitf("describe certificate: %v", err)
	}
	logf("DescribeCertificate succeeded (%d)", describeStatus)

	deleteStatus, deleteBody, err := acmRequest(ctx, endpoint, region, creds, "DeleteCertificate", map[string]any{
		"CertificateArn": certificateARN,
	})
	if err != nil {
		exitf("delete certificate: %v", err)
	}
	if err := expectHTTPStatus("DeleteCertificate", deleteStatus, http.StatusOK, deleteBody); err != nil {
		exitf("delete certificate: %v", err)
	}
	logf("DeleteCertificate succeeded (%d)", deleteStatus)

	fmt.Println("Done.")
}

func acmRequest(
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
	req.Header.Set("X-Amz-Target", "CertificateManager."+action)

	credentialValue, err := creds.Retrieve(ctx)
	if err != nil {
		return 0, nil, err
	}
	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, credentialValue, req, hashSHA256(body), "acm", region, time.Now()); err != nil {
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
