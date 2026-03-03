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
	domainName := getenv("STACKYARD_DOMAIN_NAME", "acm.example.com")

	ctx := context.Background()
	creds := credentials.NewStaticCredentialsProvider(
		getenv("AWS_ACCESS_KEY_ID", "stackyard"),
		getenv("AWS_SECRET_ACCESS_KEY", "stackyard"),
		"",
	)

	fmt.Printf("Stackyard ACM advanced client using %s\n", endpoint)

	requestStatus, requestBody, err := acmRequest(ctx, endpoint, region, creds, "RequestCertificate", map[string]any{
		"DomainName":       domainName,
		"ValidationMethod": "EMAIL",
		"IdempotencyToken": "acmadvanced",
	})
	if err != nil {
		exitf("request certificate: %v", err)
	}
	if err := expectHTTPStatus("RequestCertificate", requestStatus, http.StatusOK, requestBody); err != nil {
		exitf("request certificate: %v", err)
	}
	requestARN, err := extractString(requestBody, "CertificateArn")
	if err != nil {
		exitf("request certificate response parse: %v", err)
	}
	logf("requested certificate: %s", requestARN)

	if status, body, err := acmRequest(ctx, endpoint, region, creds, "ResendValidationEmail", map[string]any{
		"CertificateArn":   requestARN,
		"Domain":           domainName,
		"ValidationDomain": domainName,
	}); err != nil {
		exitf("resend validation email: %v", err)
	} else if err := expectHTTPStatus("ResendValidationEmail", status, http.StatusOK, body); err != nil {
		exitf("resend validation email: %v", err)
	}
	logf("ResendValidationEmail succeeded")

	if status, body, err := acmRequest(ctx, endpoint, region, creds, "UpdateCertificateOptions", map[string]any{
		"CertificateArn": requestARN,
		"Options": map[string]any{
			"CertificateTransparencyLoggingPreference": "DISABLED",
		},
	}); err != nil {
		exitf("update certificate options: %v", err)
	} else if err := expectHTTPStatus("UpdateCertificateOptions", status, http.StatusOK, body); err != nil {
		exitf("update certificate options: %v", err)
	}
	logf("UpdateCertificateOptions succeeded")

	if status, body, err := acmRequest(ctx, endpoint, region, creds, "AddTagsToCertificate", map[string]any{
		"CertificateArn": requestARN,
		"Tags": []map[string]string{
			{"Key": "env", "Value": "dev"},
			{"Key": "team", "Value": "platform"},
		},
	}); err != nil {
		exitf("add tags to certificate: %v", err)
	} else if err := expectHTTPStatus("AddTagsToCertificate", status, http.StatusOK, body); err != nil {
		exitf("add tags to certificate: %v", err)
	}
	logf("AddTagsToCertificate succeeded")

	if status, body, err := acmRequest(ctx, endpoint, region, creds, "ListTagsForCertificate", map[string]any{
		"CertificateArn": requestARN,
	}); err != nil {
		exitf("list tags for certificate: %v", err)
	} else if err := expectHTTPStatus("ListTagsForCertificate", status, http.StatusOK, body); err != nil {
		exitf("list tags for certificate: %v", err)
	}
	logf("ListTagsForCertificate succeeded")

	if status, body, err := acmRequest(ctx, endpoint, region, creds, "RemoveTagsFromCertificate", map[string]any{
		"CertificateArn": requestARN,
		"Tags": []map[string]string{
			{"Key": "team"},
		},
	}); err != nil {
		exitf("remove tags from certificate: %v", err)
	} else if err := expectHTTPStatus("RemoveTagsFromCertificate", status, http.StatusOK, body); err != nil {
		exitf("remove tags from certificate: %v", err)
	}
	logf("RemoveTagsFromCertificate succeeded")

	if status, body, err := acmRequest(ctx, endpoint, region, creds, "GetAccountConfiguration", map[string]any{}); err != nil {
		exitf("get account configuration: %v", err)
	} else if err := expectHTTPStatus("GetAccountConfiguration", status, http.StatusOK, body); err != nil {
		exitf("get account configuration: %v", err)
	}
	logf("GetAccountConfiguration succeeded")

	if status, body, err := acmRequest(ctx, endpoint, region, creds, "PutAccountConfiguration", map[string]any{
		"ExpiryEvents": map[string]any{
			"DaysBeforeExpiry": 30,
		},
	}); err != nil {
		exitf("put account configuration: %v", err)
	} else if err := expectHTTPStatus("PutAccountConfiguration", status, http.StatusOK, body); err != nil {
		exitf("put account configuration: %v", err)
	}
	logf("PutAccountConfiguration succeeded")

	importStatus, importBody, err := acmRequest(ctx, endpoint, region, creds, "ImportCertificate", map[string]any{
		"Certificate":      "-----BEGIN CERTIFICATE-----\nimported\n-----END CERTIFICATE-----",
		"PrivateKey":       "-----BEGIN PRIVATE KEY-----\nimported\n-----END PRIVATE KEY-----",
		"CertificateChain": "-----BEGIN CERTIFICATE-----\nchain\n-----END CERTIFICATE-----",
	})
	if err != nil {
		exitf("import certificate: %v", err)
	}
	if err := expectHTTPStatus("ImportCertificate", importStatus, http.StatusOK, importBody); err != nil {
		exitf("import certificate: %v", err)
	}
	importARN, err := extractString(importBody, "CertificateArn")
	if err != nil {
		exitf("import certificate response parse: %v", err)
	}
	logf("imported certificate: %s", importARN)

	if status, body, err := acmRequest(ctx, endpoint, region, creds, "GetCertificate", map[string]any{
		"CertificateArn": importARN,
	}); err != nil {
		exitf("get certificate: %v", err)
	} else if err := expectHTTPStatus("GetCertificate", status, http.StatusOK, body); err != nil {
		exitf("get certificate: %v", err)
	}
	logf("GetCertificate succeeded")

	if status, body, err := acmRequest(ctx, endpoint, region, creds, "ExportCertificate", map[string]any{
		"CertificateArn": importARN,
		"Passphrase":     "stackyard-passphrase",
	}); err != nil {
		exitf("export certificate: %v", err)
	} else if err := expectHTTPStatus("ExportCertificate", status, http.StatusOK, body); err != nil {
		exitf("export certificate: %v", err)
	}
	logf("ExportCertificate succeeded")

	if status, body, err := acmRequest(ctx, endpoint, region, creds, "RenewCertificate", map[string]any{
		"CertificateArn": importARN,
	}); err != nil {
		exitf("renew certificate: %v", err)
	} else if err := expectHTTPStatus("RenewCertificate", status, http.StatusOK, body); err != nil {
		exitf("renew certificate: %v", err)
	}
	logf("RenewCertificate succeeded")

	if status, body, err := acmRequest(ctx, endpoint, region, creds, "RevokeCertificate", map[string]any{
		"CertificateArn":   importARN,
		"RevocationReason": "KEY_COMPROMISE",
	}); err != nil {
		exitf("revoke certificate: %v", err)
	} else if err := expectHTTPStatus("RevokeCertificate", status, http.StatusOK, body); err != nil {
		exitf("revoke certificate: %v", err)
	}
	logf("RevokeCertificate succeeded")

	if status, body, err := acmRequest(ctx, endpoint, region, creds, "DeleteCertificate", map[string]any{
		"CertificateArn": requestARN,
	}); err != nil {
		exitf("delete requested certificate: %v", err)
	} else if err := expectHTTPStatus("DeleteCertificate", status, http.StatusOK, body); err != nil {
		exitf("delete requested certificate: %v", err)
	}

	if status, body, err := acmRequest(ctx, endpoint, region, creds, "DeleteCertificate", map[string]any{
		"CertificateArn": importARN,
	}); err != nil {
		exitf("delete imported certificate: %v", err)
	} else if err := expectHTTPStatus("DeleteCertificate", status, http.StatusOK, body); err != nil {
		exitf("delete imported certificate: %v", err)
	}
	logf("DeleteCertificate succeeded")

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
