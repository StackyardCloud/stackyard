package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPrivateCAStage45AuditTagsAndContracts(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	createBody := []byte(`{
		"CertificateAuthorityConfiguration": {
			"KeyAlgorithm": "RSA_2048",
			"SigningAlgorithm": "SHA256WITHRSA",
			"Subject": {
				"Country": "US",
				"Organization": "Stackyard",
				"CommonName": "stage45-privateca.stackyard.local"
			}
		},
		"CertificateAuthorityType": "ROOT",
		"IdempotencyToken": "privateca-stage45-create"
	}`)
	resp := privateCARequest(t, ts, "CreateCertificateAuthority", createBody)
	assertStatus(t, resp, http.StatusOK)
	var createOut struct {
		CertificateAuthorityArn string `json:"CertificateAuthorityArn"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &createOut); err != nil {
		t.Fatalf("unmarshal create output: %v", err)
	}
	if strings.TrimSpace(createOut.CertificateAuthorityArn) == "" {
		t.Fatalf("expected certificate authority arn")
	}

	// Stage 5 idempotency: same create token returns same CA ARN.
	resp = privateCARequest(t, ts, "CreateCertificateAuthority", createBody)
	assertStatus(t, resp, http.StatusOK)
	var createOut2 struct {
		CertificateAuthorityArn string `json:"CertificateAuthorityArn"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &createOut2); err != nil {
		t.Fatalf("unmarshal second create output: %v", err)
	}
	if createOut2.CertificateAuthorityArn != createOut.CertificateAuthorityArn {
		t.Fatalf("expected idempotent create arn %q, got %q", createOut.CertificateAuthorityArn, createOut2.CertificateAuthorityArn)
	}

	resp = privateCARequest(t, ts, "CreateCertificateAuthorityAuditReport", mustJSON(t, map[string]any{
		"CertificateAuthorityArn":   createOut.CertificateAuthorityArn,
		"S3BucketName":              "stackyard-privateca-reports",
		"AuditReportResponseFormat": "JSON",
	}))
	assertStatus(t, resp, http.StatusOK)
	var createReportOut struct {
		AuditReportId string `json:"AuditReportId"`
		S3Key         string `json:"S3Key"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &createReportOut); err != nil {
		t.Fatalf("unmarshal create audit report output: %v", err)
	}
	if strings.TrimSpace(createReportOut.AuditReportId) == "" || strings.TrimSpace(createReportOut.S3Key) == "" {
		t.Fatalf("expected audit report id and s3 key")
	}

	resp = privateCARequest(t, ts, "DescribeCertificateAuthorityAuditReport", mustJSON(t, map[string]any{
		"CertificateAuthorityArn": createOut.CertificateAuthorityArn,
		"AuditReportId":           createReportOut.AuditReportId,
	}))
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "\"AuditReportStatus\":\"SUCCESS\"") {
		t.Fatalf("expected successful audit report status, got %s", body)
	}

	resp = privateCARequest(t, ts, "TagCertificateAuthority", mustJSON(t, map[string]any{
		"CertificateAuthorityArn": createOut.CertificateAuthorityArn,
		"Tags": []map[string]string{
			{"Key": "env", "Value": "dev"},
			{"Key": "team", "Value": "platform"},
		},
	}))
	assertStatus(t, resp, http.StatusOK)

	resp = privateCARequest(t, ts, "ListTags", mustJSON(t, map[string]any{
		"CertificateAuthorityArn": createOut.CertificateAuthorityArn,
		"MaxResults":              1,
	}))
	assertStatus(t, resp, http.StatusOK)
	var listTagsOut struct {
		NextToken string `json:"NextToken"`
		Tags      []struct {
			Key   string `json:"Key"`
			Value string `json:"Value"`
		} `json:"Tags"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &listTagsOut); err != nil {
		t.Fatalf("unmarshal list tags output: %v", err)
	}
	if len(listTagsOut.Tags) != 1 {
		t.Fatalf("expected 1 tag on first page, got %d", len(listTagsOut.Tags))
	}
	if strings.TrimSpace(listTagsOut.NextToken) == "" {
		t.Fatalf("expected next token for paged tag response")
	}

	resp = privateCARequest(t, ts, "UntagCertificateAuthority", mustJSON(t, map[string]any{
		"CertificateAuthorityArn": createOut.CertificateAuthorityArn,
		"Tags": []map[string]string{
			{"Key": "team"},
		},
	}))
	assertStatus(t, resp, http.StatusOK)

	resp = privateCARequest(t, ts, "ListTags", mustJSON(t, map[string]any{
		"CertificateAuthorityArn": createOut.CertificateAuthorityArn,
		"MaxResults":              10,
	}))
	assertStatus(t, resp, http.StatusOK)
	body = string(mustBody(t, resp))
	if strings.Contains(body, `"Key":"team"`) {
		t.Fatalf("expected team tag to be removed")
	}
	if !strings.Contains(body, `"Key":"env"`) {
		t.Fatalf("expected env tag to remain")
	}

	resp = privateCARequest(t, ts, "ListPermissions", mustJSON(t, map[string]any{
		"CertificateAuthorityArn": createOut.CertificateAuthorityArn,
		"NextToken":               "not-a-number",
	}))
	assertStatus(t, resp, http.StatusBadRequest)
	if !strings.Contains(string(mustBody(t, resp)), "ValidationException") {
		t.Fatalf("expected validation exception for invalid next token")
	}

	resp = privateCARequest(t, ts, "DeleteCertificateAuthority", mustJSON(t, map[string]any{
		"CertificateAuthorityArn":     createOut.CertificateAuthorityArn,
		"PermanentDeletionTimeInDays": 7,
	}))
	assertStatus(t, resp, http.StatusOK)

	resp = privateCARequest(t, ts, "PutPolicy", mustJSON(t, map[string]any{
		"ResourceArn": createOut.CertificateAuthorityArn,
		"Policy":      `{"Version":"2012-10-17","Statement":[]}`,
	}))
	assertStatus(t, resp, http.StatusBadRequest)
	if !strings.Contains(string(mustBody(t, resp)), "InvalidStateException") {
		t.Fatalf("expected invalid state exception when putting policy on deleted CA")
	}
}

func TestPrivateCAStage4ActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := privateCARequest(t, ts, "CreateCertificateAuthority", []byte(`{
		"CertificateAuthorityConfiguration": {
			"KeyAlgorithm": "RSA_2048",
			"SigningAlgorithm": "SHA256WITHRSA",
			"Subject": {"Country": "US", "Organization": "Stackyard", "CommonName": "stage4-implemented"}
		},
		"CertificateAuthorityType": "ROOT"
	}`))
	assertStatus(t, resp, http.StatusOK)
	var createOut struct {
		CertificateAuthorityArn string `json:"CertificateAuthorityArn"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &createOut); err != nil {
		t.Fatalf("unmarshal create output: %v", err)
	}

	actions := []struct {
		Name string
		Body []byte
	}{
		{Name: "CreateCertificateAuthorityAuditReport", Body: mustJSON(t, map[string]any{"CertificateAuthorityArn": createOut.CertificateAuthorityArn, "S3BucketName": "stackyard-privateca-reports", "AuditReportResponseFormat": "JSON"})},
		{Name: "DescribeCertificateAuthorityAuditReport", Body: mustJSON(t, map[string]any{"CertificateAuthorityArn": createOut.CertificateAuthorityArn, "AuditReportId": "00000000-0000-0000-0000-000000000000"})},
		{Name: "TagCertificateAuthority", Body: mustJSON(t, map[string]any{"CertificateAuthorityArn": createOut.CertificateAuthorityArn, "Tags": []map[string]string{{"Key": "env", "Value": "dev"}}})},
		{Name: "ListTags", Body: mustJSON(t, map[string]any{"CertificateAuthorityArn": createOut.CertificateAuthorityArn, "MaxResults": 10})},
		{Name: "UntagCertificateAuthority", Body: mustJSON(t, map[string]any{"CertificateAuthorityArn": createOut.CertificateAuthorityArn, "Tags": []map[string]string{{"Key": "env"}}})},
	}

	for _, action := range actions {
		resp = privateCARequest(t, ts, action.Name, action.Body)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action.Name)
		}
	}
}
