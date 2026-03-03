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

	ctx := context.Background()
	creds := credentials.NewStaticCredentialsProvider(
		getenv("AWS_ACCESS_KEY_ID", "stackyard"),
		getenv("AWS_SECRET_ACCESS_KEY", "stackyard"),
		"",
	)

	fmt.Printf("Stackyard Private CA basic client using %s\n", endpoint)

	createStatus, createBody, err := privateCARequest(ctx, endpoint, region, creds, "CreateCertificateAuthority", map[string]any{
		"CertificateAuthorityConfiguration": map[string]any{
			"KeyAlgorithm":     "RSA_2048",
			"SigningAlgorithm": "SHA256WITHRSA",
			"Subject": map[string]any{
				"Country":      "US",
				"Organization": "Stackyard",
				"CommonName":   "privateca-basic.stackyard.local",
			},
		},
		"CertificateAuthorityType": "ROOT",
		"IdempotencyToken":         "privatecabasic",
	})
	if err != nil {
		exitf("create certificate authority: %v", err)
	}
	if err := expectHTTPStatus("CreateCertificateAuthority", createStatus, http.StatusOK, createBody); err != nil {
		exitf("create certificate authority: %v", err)
	}
	caARN, err := extractString(createBody, "CertificateAuthorityArn")
	if err != nil {
		exitf("create certificate authority response parse: %v", err)
	}
	logf("created certificate authority: %s", caARN)

	describeStatus, describeBody, err := privateCARequest(ctx, endpoint, region, creds, "DescribeCertificateAuthority", map[string]any{
		"CertificateAuthorityArn": caARN,
	})
	if err != nil {
		exitf("describe certificate authority: %v", err)
	}
	if err := expectHTTPStatus("DescribeCertificateAuthority", describeStatus, http.StatusOK, describeBody); err != nil {
		exitf("describe certificate authority: %v", err)
	}
	logf("DescribeCertificateAuthority succeeded (%d)", describeStatus)

	listStatus, listBody, err := privateCARequest(ctx, endpoint, region, creds, "ListCertificateAuthorities", map[string]any{
		"MaxResults": 10,
	})
	if err != nil {
		exitf("list certificate authorities: %v", err)
	}
	if err := expectHTTPStatus("ListCertificateAuthorities", listStatus, http.StatusOK, listBody); err != nil {
		exitf("list certificate authorities: %v", err)
	}
	logf("ListCertificateAuthorities succeeded (%d)", listStatus)

	updateStatus, updateBody, err := privateCARequest(ctx, endpoint, region, creds, "UpdateCertificateAuthority", map[string]any{
		"CertificateAuthorityArn": caARN,
		"Status":                  "DISABLED",
	})
	if err != nil {
		exitf("update certificate authority: %v", err)
	}
	if err := expectHTTPStatus("UpdateCertificateAuthority", updateStatus, http.StatusOK, updateBody); err != nil {
		exitf("update certificate authority: %v", err)
	}
	logf("UpdateCertificateAuthority succeeded (%d)", updateStatus)

	deleteStatus, deleteBody, err := privateCARequest(ctx, endpoint, region, creds, "DeleteCertificateAuthority", map[string]any{
		"CertificateAuthorityArn":     caARN,
		"PermanentDeletionTimeInDays": 7,
	})
	if err != nil {
		exitf("delete certificate authority: %v", err)
	}
	if err := expectHTTPStatus("DeleteCertificateAuthority", deleteStatus, http.StatusOK, deleteBody); err != nil {
		exitf("delete certificate authority: %v", err)
	}
	logf("DeleteCertificateAuthority succeeded (%d)", deleteStatus)

	fmt.Println("Done.")
}

func privateCARequest(
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
	req.Header.Set("X-Amz-Target", "ACMPrivateCA."+action)

	credentialValue, err := creds.Retrieve(ctx)
	if err != nil {
		return 0, nil, err
	}
	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, credentialValue, req, hashSHA256(body), "acm-pca", region, time.Now()); err != nil {
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
