package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPrivateCAStage23IssuanceAndPolicyFlows(t *testing.T) {
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
				"CommonName": "stage23-privateca.stackyard.local"
			}
		},
		"CertificateAuthorityType": "ROOT",
		"IdempotencyToken": "privateca-stage23"
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

	resp = privateCARequest(t, ts, "IssueCertificate", mustJSON(t, map[string]any{
		"CertificateAuthorityArn": createOut.CertificateAuthorityArn,
		"Csr":                     "c3RhY2t5YXJk",
		"SigningAlgorithm":        "SHA256WITHRSA",
		"TemplateArn":             "arn:aws:acm-pca:::template/EndEntityCertificate/V1",
		"Validity":                map[string]any{"Type": "DAYS", "Value": 365},
		"IdempotencyToken":        "privateca-stage23-issue",
	}))
	assertStatus(t, resp, http.StatusOK)
	var issueOut struct {
		CertificateArn string `json:"CertificateArn"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &issueOut); err != nil {
		t.Fatalf("unmarshal issue output: %v", err)
	}
	if strings.TrimSpace(issueOut.CertificateArn) == "" {
		t.Fatalf("expected certificate arn")
	}

	resp = privateCARequest(t, ts, "GetCertificate", mustJSON(t, map[string]any{
		"CertificateAuthorityArn": createOut.CertificateAuthorityArn,
		"CertificateArn":          issueOut.CertificateArn,
	}))
	assertStatus(t, resp, http.StatusOK)
	if !strings.Contains(string(mustBody(t, resp)), "BEGIN CERTIFICATE") {
		t.Fatalf("expected certificate payload")
	}

	resp = privateCARequest(t, ts, "GetCertificateAuthorityCsr", mustJSON(t, map[string]any{
		"CertificateAuthorityArn": createOut.CertificateAuthorityArn,
	}))
	assertStatus(t, resp, http.StatusOK)

	resp = privateCARequest(t, ts, "ImportCertificateAuthorityCertificate", mustJSON(t, map[string]any{
		"CertificateAuthorityArn": createOut.CertificateAuthorityArn,
		"Certificate":             "imported-cert",
		"CertificateChain":        "imported-chain",
	}))
	assertStatus(t, resp, http.StatusOK)

	resp = privateCARequest(t, ts, "GetCertificateAuthorityCertificate", mustJSON(t, map[string]any{
		"CertificateAuthorityArn": createOut.CertificateAuthorityArn,
	}))
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "imported-cert") {
		t.Fatalf("expected imported certificate in payload")
	}

	resp = privateCARequest(t, ts, "RevokeCertificate", mustJSON(t, map[string]any{
		"CertificateAuthorityArn": createOut.CertificateAuthorityArn,
		"CertificateSerial":       "01",
		"RevocationReason":        "KEY_COMPROMISE",
	}))
	assertStatus(t, resp, http.StatusOK)

	resp = privateCARequest(t, ts, "CreatePermission", mustJSON(t, map[string]any{
		"CertificateAuthorityArn": createOut.CertificateAuthorityArn,
		"Principal":               "123456789012",
		"SourceAccount":           "123456789012",
		"Actions":                 []string{"IssueCertificate", "GetCertificate", "ListPermissions"},
	}))
	assertStatus(t, resp, http.StatusOK)

	resp = privateCARequest(t, ts, "ListPermissions", mustJSON(t, map[string]any{
		"CertificateAuthorityArn": createOut.CertificateAuthorityArn,
		"MaxResults":              10,
	}))
	assertStatus(t, resp, http.StatusOK)
	if !strings.Contains(string(mustBody(t, resp)), "ListPermissions") {
		t.Fatalf("expected listed permission actions")
	}

	resp = privateCARequest(t, ts, "DeletePermission", mustJSON(t, map[string]any{
		"CertificateAuthorityArn": createOut.CertificateAuthorityArn,
		"Principal":               "123456789012",
		"SourceAccount":           "123456789012",
	}))
	assertStatus(t, resp, http.StatusOK)

	policy := `{"Version":"2012-10-17","Statement":[]}`
	resp = privateCARequest(t, ts, "PutPolicy", mustJSON(t, map[string]any{
		"CertificateAuthorityArn": createOut.CertificateAuthorityArn,
		"Policy":                  policy,
	}))
	assertStatus(t, resp, http.StatusOK)

	resp = privateCARequest(t, ts, "GetPolicy", mustJSON(t, map[string]any{
		"CertificateAuthorityArn": createOut.CertificateAuthorityArn,
	}))
	assertStatus(t, resp, http.StatusOK)
	var getPolicyOut struct {
		Policy string `json:"Policy"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &getPolicyOut); err != nil {
		t.Fatalf("unmarshal get policy output: %v", err)
	}
	if getPolicyOut.Policy != policy {
		t.Fatalf("expected policy %q, got %q", policy, getPolicyOut.Policy)
	}

	resp = privateCARequest(t, ts, "DeletePolicy", mustJSON(t, map[string]any{
		"CertificateAuthorityArn": createOut.CertificateAuthorityArn,
	}))
	assertStatus(t, resp, http.StatusOK)
}

func TestPrivateCAStage23ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
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
				"CommonName": "stage23-privateca.stackyard.local"
			}
		},
		"CertificateAuthorityType": "ROOT"
	}`)
	resp := privateCARequest(t, ts, "CreateCertificateAuthority", createBody)
	assertStatus(t, resp, http.StatusOK)
	var createOut struct {
		CertificateAuthorityArn string `json:"CertificateAuthorityArn"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &createOut); err != nil {
		t.Fatalf("unmarshal create output: %v", err)
	}

	resp = privateCARequest(t, ts, "IssueCertificate", mustJSON(t, map[string]any{
		"CertificateAuthorityArn": createOut.CertificateAuthorityArn,
		"Csr":                     "c3RhY2t5YXJk",
		"SigningAlgorithm":        "SHA256WITHRSA",
		"TemplateArn":             "arn:aws:acm-pca:::template/EndEntityCertificate/V1",
		"Validity":                map[string]any{"Type": "DAYS", "Value": 365},
		"IdempotencyToken":        "privateca-stage23-implemented",
	}))
	assertStatus(t, resp, http.StatusOK)
	var issueOut struct {
		CertificateArn string `json:"CertificateArn"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &issueOut); err != nil {
		t.Fatalf("unmarshal issue output: %v", err)
	}

	actions := []struct {
		Name string
		Body []byte
	}{
		{Name: "GetCertificate", Body: mustJSON(t, map[string]any{"CertificateAuthorityArn": createOut.CertificateAuthorityArn, "CertificateArn": issueOut.CertificateArn})},
		{Name: "GetCertificateAuthorityCsr", Body: mustJSON(t, map[string]any{"CertificateAuthorityArn": createOut.CertificateAuthorityArn})},
		{Name: "ImportCertificateAuthorityCertificate", Body: mustJSON(t, map[string]any{"CertificateAuthorityArn": createOut.CertificateAuthorityArn, "Certificate": "imported-cert", "CertificateChain": "imported-chain"})},
		{Name: "GetCertificateAuthorityCertificate", Body: mustJSON(t, map[string]any{"CertificateAuthorityArn": createOut.CertificateAuthorityArn})},
		{Name: "RevokeCertificate", Body: mustJSON(t, map[string]any{"CertificateAuthorityArn": createOut.CertificateAuthorityArn, "CertificateSerial": "01", "RevocationReason": "KEY_COMPROMISE"})},
		{Name: "CreatePermission", Body: mustJSON(t, map[string]any{"CertificateAuthorityArn": createOut.CertificateAuthorityArn, "Principal": "123456789012", "SourceAccount": "123456789012", "Actions": []string{"IssueCertificate", "GetCertificate", "ListPermissions"}})},
		{Name: "ListPermissions", Body: mustJSON(t, map[string]any{"CertificateAuthorityArn": createOut.CertificateAuthorityArn, "MaxResults": 10})},
		{Name: "DeletePermission", Body: mustJSON(t, map[string]any{"CertificateAuthorityArn": createOut.CertificateAuthorityArn, "Principal": "123456789012", "SourceAccount": "123456789012"})},
		{Name: "PutPolicy", Body: mustJSON(t, map[string]any{"CertificateAuthorityArn": createOut.CertificateAuthorityArn, "Policy": `{"Version":"2012-10-17","Statement":[]}`})},
		{Name: "GetPolicy", Body: mustJSON(t, map[string]any{"CertificateAuthorityArn": createOut.CertificateAuthorityArn})},
		{Name: "DeletePolicy", Body: mustJSON(t, map[string]any{"CertificateAuthorityArn": createOut.CertificateAuthorityArn})},
	}

	for _, action := range actions {
		resp = privateCARequest(t, ts, action.Name, action.Body)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action.Name)
		}
	}
}
