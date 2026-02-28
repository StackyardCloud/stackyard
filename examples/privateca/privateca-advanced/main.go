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
	certARN := getenv("STACKYARD_CERT_ARN", "arn:aws:acm-pca:us-east-1:123456789012:certificate-authority/stackyard/certificate/0000000001")

	ctx := context.Background()
	creds := credentials.NewStaticCredentialsProvider(
		getenv("AWS_ACCESS_KEY_ID", "stackyard"),
		getenv("AWS_SECRET_ACCESS_KEY", "stackyard"),
		"",
	)

	fmt.Printf("Stackyard Private CA advanced client using %s\n", endpoint)

	createStatus, createBody, err := privateCARequest(ctx, endpoint, region, creds, "CreateCertificateAuthority", map[string]any{
		"CertificateAuthorityConfiguration": map[string]any{
			"KeyAlgorithm":     "RSA_2048",
			"SigningAlgorithm": "SHA256WITHRSA",
			"Subject": map[string]any{
				"Country":      "US",
				"Organization": "Stackyard",
				"CommonName":   "privateca-advanced.stackyard.local",
			},
		},
		"CertificateAuthorityType": "ROOT",
		"IdempotencyToken":         "privatecaadvanced",
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
	logf("CreateCertificateAuthority succeeded")

	issueStatus, issueBody, err := privateCARequest(ctx, endpoint, region, creds, "IssueCertificate", map[string]any{
		"CertificateAuthorityArn": caARN,
		"Csr":                     "c3RhY2t5YXJk",
		"SigningAlgorithm":        "SHA256WITHRSA",
		"TemplateArn":             "arn:aws:acm-pca:::template/EndEntityCertificate/V1",
		"Validity": map[string]any{
			"Value": 365,
			"Type":  "DAYS",
		},
		"IdempotencyToken": "privatecaissue",
	})
	if err != nil {
		exitf("issue certificate: %v", err)
	} else if err := expectHTTPStatus("IssueCertificate", issueStatus, http.StatusOK, issueBody); err != nil {
		exitf("issue certificate: %v", err)
	}
	issuedARN := certARN
	if value, err := extractString(issueBody, "CertificateArn"); err == nil {
		issuedARN = value
	}
	logf("IssueCertificate succeeded")

	if status, body, err := privateCARequest(ctx, endpoint, region, creds, "GetCertificate", map[string]any{
		"CertificateAuthorityArn": caARN,
		"CertificateArn":          issuedARN,
	}); err != nil {
		exitf("get certificate: %v", err)
	} else if err := expectHTTPStatus("GetCertificate", status, http.StatusOK, body); err != nil {
		exitf("get certificate: %v", err)
	}
	logf("GetCertificate succeeded")

	if status, body, err := privateCARequest(ctx, endpoint, region, creds, "CreatePermission", map[string]any{
		"CertificateAuthorityArn": caARN,
		"Principal":               "123456789012",
		"SourceAccount":           "123456789012",
		"Actions":                 []string{"IssueCertificate", "GetCertificate", "ListPermissions"},
	}); err != nil {
		exitf("create permission: %v", err)
	} else if err := expectHTTPStatus("CreatePermission", status, http.StatusOK, body); err != nil {
		exitf("create permission: %v", err)
	}
	logf("CreatePermission succeeded")

	if status, body, err := privateCARequest(ctx, endpoint, region, creds, "ListPermissions", map[string]any{
		"CertificateAuthorityArn": caARN,
		"MaxResults":              10,
	}); err != nil {
		exitf("list permissions: %v", err)
	} else if err := expectHTTPStatus("ListPermissions", status, http.StatusOK, body); err != nil {
		exitf("list permissions: %v", err)
	}
	logf("ListPermissions succeeded")

	if status, body, err := privateCARequest(ctx, endpoint, region, creds, "DeletePermission", map[string]any{
		"CertificateAuthorityArn": caARN,
		"Principal":               "123456789012",
		"SourceAccount":           "123456789012",
	}); err != nil {
		exitf("delete permission: %v", err)
	} else if err := expectHTTPStatus("DeletePermission", status, http.StatusOK, body); err != nil {
		exitf("delete permission: %v", err)
	}
	logf("DeletePermission succeeded")

	policy := `{"Version":"2012-10-17","Statement":[]}`
	if status, body, err := privateCARequest(ctx, endpoint, region, creds, "PutPolicy", map[string]any{
		"CertificateAuthorityArn": caARN,
		"Policy":                  policy,
	}); err != nil {
		exitf("put policy: %v", err)
	} else if err := expectHTTPStatus("PutPolicy", status, http.StatusOK, body); err != nil {
		exitf("put policy: %v", err)
	}
	logf("PutPolicy succeeded")

	if status, body, err := privateCARequest(ctx, endpoint, region, creds, "GetPolicy", map[string]any{
		"CertificateAuthorityArn": caARN,
	}); err != nil {
		exitf("get policy: %v", err)
	} else if err := expectHTTPStatus("GetPolicy", status, http.StatusOK, body); err != nil {
		exitf("get policy: %v", err)
	}
	logf("GetPolicy succeeded")

	if status, body, err := privateCARequest(ctx, endpoint, region, creds, "DeletePolicy", map[string]any{
		"CertificateAuthorityArn": caARN,
	}); err != nil {
		exitf("delete policy: %v", err)
	} else if err := expectHTTPStatus("DeletePolicy", status, http.StatusOK, body); err != nil {
		exitf("delete policy: %v", err)
	}
	logf("DeletePolicy succeeded")

	reportStatus, reportBody, err := privateCARequest(ctx, endpoint, region, creds, "CreateCertificateAuthorityAuditReport", map[string]any{
		"CertificateAuthorityArn":   caARN,
		"S3BucketName":              "stackyard-privateca-reports",
		"AuditReportResponseFormat": "JSON",
	})
	if err != nil {
		exitf("create certificate authority audit report: %v", err)
	}
	if err := expectHTTPStatus("CreateCertificateAuthorityAuditReport", reportStatus, http.StatusOK, reportBody); err != nil {
		exitf("create certificate authority audit report: %v", err)
	}
	reportID, err := extractString(reportBody, "AuditReportId")
	if err != nil {
		exitf("create certificate authority audit report response parse: %v", err)
	}
	logf("CreateCertificateAuthorityAuditReport succeeded")

	if status, body, err := privateCARequest(ctx, endpoint, region, creds, "DescribeCertificateAuthorityAuditReport", map[string]any{
		"CertificateAuthorityArn": caARN,
		"AuditReportId":           reportID,
	}); err != nil {
		exitf("describe certificate authority audit report: %v", err)
	} else if err := expectHTTPStatus("DescribeCertificateAuthorityAuditReport", status, http.StatusOK, body); err != nil {
		exitf("describe certificate authority audit report: %v", err)
	}
	logf("DescribeCertificateAuthorityAuditReport succeeded")

	if status, body, err := privateCARequest(ctx, endpoint, region, creds, "UpdateCertificateAuthority", map[string]any{
		"CertificateAuthorityArn": caARN,
		"Status":                  "DISABLED",
	}); err != nil {
		exitf("update certificate authority: %v", err)
	} else if err := expectHTTPStatus("UpdateCertificateAuthority", status, http.StatusOK, body); err != nil {
		exitf("update certificate authority: %v", err)
	}
	logf("UpdateCertificateAuthority succeeded")

	if status, body, err := privateCARequest(ctx, endpoint, region, creds, "DeleteCertificateAuthority", map[string]any{
		"CertificateAuthorityArn":     caARN,
		"PermanentDeletionTimeInDays": 7,
	}); err != nil {
		exitf("delete certificate authority: %v", err)
	} else if err := expectHTTPStatus("DeleteCertificateAuthority", status, http.StatusOK, body); err != nil {
		exitf("delete certificate authority: %v", err)
	}
	logf("DeleteCertificateAuthority succeeded")

	if status, body, err := privateCARequest(ctx, endpoint, region, creds, "RestoreCertificateAuthority", map[string]any{
		"CertificateAuthorityArn": caARN,
	}); err != nil {
		exitf("restore certificate authority: %v", err)
	} else if err := expectHTTPStatus("RestoreCertificateAuthority", status, http.StatusOK, body); err != nil {
		exitf("restore certificate authority: %v", err)
	}
	logf("RestoreCertificateAuthority succeeded")

	if status, body, err := privateCARequest(ctx, endpoint, region, creds, "TagCertificateAuthority", map[string]any{
		"CertificateAuthorityArn": caARN,
		"Tags":                    []map[string]string{{"Key": "env", "Value": "dev"}},
	}); err != nil {
		exitf("tag certificate authority: %v", err)
	} else if err := expectHTTPStatus("TagCertificateAuthority", status, http.StatusOK, body); err != nil {
		exitf("tag certificate authority: %v", err)
	}
	logf("TagCertificateAuthority succeeded")

	if status, body, err := privateCARequest(ctx, endpoint, region, creds, "ListTags", map[string]any{
		"CertificateAuthorityArn": caARN,
		"MaxResults":              10,
	}); err != nil {
		exitf("list tags: %v", err)
	} else if err := expectHTTPStatus("ListTags", status, http.StatusOK, body); err != nil {
		exitf("list tags: %v", err)
	}
	logf("ListTags succeeded")

	if status, body, err := privateCARequest(ctx, endpoint, region, creds, "UntagCertificateAuthority", map[string]any{
		"CertificateAuthorityArn": caARN,
		"Tags":                    []map[string]string{{"Key": "env"}},
	}); err != nil {
		exitf("untag certificate authority: %v", err)
	} else if err := expectHTTPStatus("UntagCertificateAuthority", status, http.StatusOK, body); err != nil {
		exitf("untag certificate authority: %v", err)
	}
	logf("UntagCertificateAuthority succeeded")

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
