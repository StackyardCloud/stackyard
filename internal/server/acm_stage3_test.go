package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func acmRequest(t *testing.T, ts *httptest.Server, action string, body []byte) *http.Response {
	t.Helper()
	return signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		body,
		map[string]string{
			"Content-Type": "application/x-amz-json-1.1",
			"X-Amz-Target": "CertificateManager." + action,
		},
		"acm",
	)
}

func TestACMOperationCoverage(t *testing.T) {
	if len(acmOperations) != 16 {
		t.Fatalf("expected 16 ACM operations from docs, got %d", len(acmOperations))
	}
	if len(acmOperationByName) != len(acmOperations) {
		t.Fatalf("expected unique operation names")
	}
	required := []string{
		"RequestCertificate",
		"ImportCertificate",
		"DescribeCertificate",
		"ListCertificates",
		"DeleteCertificate",
		"GetCertificate",
		"ExportCertificate",
		"RenewCertificate",
		"RevokeCertificate",
		"ResendValidationEmail",
		"UpdateCertificateOptions",
		"AddTagsToCertificate",
		"RemoveTagsFromCertificate",
		"ListTagsForCertificate",
		"GetAccountConfiguration",
		"PutAccountConfiguration",
	}
	for _, name := range required {
		if _, ok := acmOperationByName[name]; !ok {
			t.Fatalf("missing documented operation %s", name)
		}
	}
}

func TestACMStage123Lifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := acmRequest(t, ts, "RequestCertificate", []byte(`{
		"DomainName":"example.com",
		"ValidationMethod":"EMAIL",
		"IdempotencyToken":"stage123",
		"Options":{"CertificateTransparencyLoggingPreference":"ENABLED"}
	}`))
	assertStatus(t, resp, http.StatusOK)
	var requestOut struct {
		CertificateArn string `json:"CertificateArn"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &requestOut); err != nil {
		t.Fatalf("unmarshal request certificate response: %v", err)
	}
	if requestOut.CertificateArn == "" {
		t.Fatalf("expected certificate arn")
	}

	resp = acmRequest(t, ts, "DescribeCertificate", []byte(`{"CertificateArn":"`+requestOut.CertificateArn+`"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = acmRequest(t, ts, "ListCertificates", []byte(`{"MaxItems":10}`))
	assertStatus(t, resp, http.StatusOK)
	if !strings.Contains(string(mustBody(t, resp)), requestOut.CertificateArn) {
		t.Fatalf("expected list certificates to include request arn")
	}

	resp = acmRequest(t, ts, "ResendValidationEmail", []byte(`{
		"CertificateArn":"`+requestOut.CertificateArn+`",
		"Domain":"example.com",
		"ValidationDomain":"example.com"
	}`))
	assertStatus(t, resp, http.StatusOK)

	resp = acmRequest(t, ts, "UpdateCertificateOptions", []byte(`{
		"CertificateArn":"`+requestOut.CertificateArn+`",
		"Options":{"CertificateTransparencyLoggingPreference":"DISABLED"}
	}`))
	assertStatus(t, resp, http.StatusOK)

	resp = acmRequest(t, ts, "GetCertificate", []byte(`{"CertificateArn":"`+requestOut.CertificateArn+`"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = acmRequest(t, ts, "ImportCertificate", []byte(`{
		"Certificate":"-----BEGIN CERTIFICATE-----\\nimported\\n-----END CERTIFICATE-----",
		"PrivateKey":"-----BEGIN PRIVATE KEY-----\\nimported\\n-----END PRIVATE KEY-----",
		"CertificateChain":"-----BEGIN CERTIFICATE-----\\nchain\\n-----END CERTIFICATE-----"
	}`))
	assertStatus(t, resp, http.StatusOK)
	var importOut struct {
		CertificateArn string `json:"CertificateArn"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &importOut); err != nil {
		t.Fatalf("unmarshal import certificate response: %v", err)
	}
	if importOut.CertificateArn == "" {
		t.Fatalf("expected imported certificate arn")
	}

	resp = acmRequest(t, ts, "ExportCertificate", []byte(`{
		"CertificateArn":"`+importOut.CertificateArn+`",
		"Passphrase":"stackyard-passphrase"
	}`))
	assertStatus(t, resp, http.StatusOK)

	resp = acmRequest(t, ts, "RenewCertificate", []byte(`{"CertificateArn":"`+importOut.CertificateArn+`"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = acmRequest(t, ts, "RevokeCertificate", []byte(`{
		"CertificateArn":"`+importOut.CertificateArn+`",
		"RevocationReason":"KEY_COMPROMISE"
	}`))
	assertStatus(t, resp, http.StatusOK)

	resp = acmRequest(t, ts, "DeleteCertificate", []byte(`{"CertificateArn":"`+importOut.CertificateArn+`"}`))
	assertStatus(t, resp, http.StatusOK)
}

func TestACMStage4TagAndAccountConfig(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := acmRequest(t, ts, "ImportCertificate", []byte(`{
		"Certificate":"-----BEGIN CERTIFICATE-----\\nimported\\n-----END CERTIFICATE-----",
		"PrivateKey":"-----BEGIN PRIVATE KEY-----\\nimported\\n-----END PRIVATE KEY-----"
	}`))
	assertStatus(t, resp, http.StatusOK)
	var importOut struct {
		CertificateArn string `json:"CertificateArn"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &importOut); err != nil {
		t.Fatalf("unmarshal import certificate response: %v", err)
	}

	resp = acmRequest(t, ts, "AddTagsToCertificate", []byte(`{
		"CertificateArn":"`+importOut.CertificateArn+`",
		"Tags":[{"Key":"env","Value":"dev"},{"Key":"team","Value":"platform"}]
	}`))
	assertStatus(t, resp, http.StatusOK)

	resp = acmRequest(t, ts, "ListTagsForCertificate", []byte(`{"CertificateArn":"`+importOut.CertificateArn+`"}`))
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "\"Key\":\"env\"") || !strings.Contains(body, "\"Key\":\"team\"") {
		t.Fatalf("expected both tags in list output: %s", body)
	}

	resp = acmRequest(t, ts, "RemoveTagsFromCertificate", []byte(`{
		"CertificateArn":"`+importOut.CertificateArn+`",
		"Tags":["team"]
	}`))
	assertStatus(t, resp, http.StatusOK)

	resp = acmRequest(t, ts, "GetAccountConfiguration", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)

	resp = acmRequest(t, ts, "PutAccountConfiguration", []byte(`{
		"ExpiryEvents":{"DaysBeforeExpiry":30}
	}`))
	assertStatus(t, resp, http.StatusOK)

	resp = acmRequest(t, ts, "GetAccountConfiguration", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	if !strings.Contains(string(mustBody(t, resp)), "\"DaysBeforeExpiry\":30") {
		t.Fatalf("expected updated account configuration days")
	}
}

func TestACMStage5ContractAndInvariants(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := acmRequest(t, ts, "ListCertificates", []byte(`{"NextToken":"bad-token"}`))
	assertStatus(t, resp, http.StatusBadRequest)
	if !strings.Contains(string(mustBody(t, resp)), "ValidationException") {
		t.Fatalf("expected validation exception for invalid token")
	}

	resp = acmRequest(t, ts, "ListCertificates", []byte(`{"CertificateStatuses":["NOT_A_STATUS"]}`))
	assertStatus(t, resp, http.StatusBadRequest)
	if !strings.Contains(string(mustBody(t, resp)), "ValidationException") {
		t.Fatalf("expected validation exception for invalid status")
	}

	resp = acmRequest(t, ts, "RequestCertificate", []byte(`{
		"DomainName":"dns-only.example.com",
		"ValidationMethod":"DNS",
		"IdempotencyToken":"stage5dns"
	}`))
	assertStatus(t, resp, http.StatusOK)
	var requestOut struct {
		CertificateArn string `json:"CertificateArn"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &requestOut); err != nil {
		t.Fatalf("unmarshal request certificate response: %v", err)
	}

	resp = acmRequest(t, ts, "ResendValidationEmail", []byte(`{
		"CertificateArn":"`+requestOut.CertificateArn+`",
		"Domain":"dns-only.example.com",
		"ValidationDomain":"dns-only.example.com"
	}`))
	assertStatus(t, resp, http.StatusBadRequest)
	if !strings.Contains(string(mustBody(t, resp)), "InvalidStateException") {
		t.Fatalf("expected invalid state exception for DNS resend")
	}

	resp = acmRequest(t, ts, "ImportCertificate", []byte(`{
		"Certificate":"-----BEGIN CERTIFICATE-----\\nimported\\n-----END CERTIFICATE-----",
		"PrivateKey":"-----BEGIN PRIVATE KEY-----\\nimported\\n-----END PRIVATE KEY-----"
	}`))
	assertStatus(t, resp, http.StatusOK)
	var importOut struct {
		CertificateArn string `json:"CertificateArn"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &importOut); err != nil {
		t.Fatalf("unmarshal import response: %v", err)
	}

	resp = acmRequest(t, ts, "RevokeCertificate", []byte(`{"CertificateArn":"`+importOut.CertificateArn+`","RevocationReason":"KEY_COMPROMISE"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = acmRequest(t, ts, "ExportCertificate", []byte(`{"CertificateArn":"`+importOut.CertificateArn+`","Passphrase":"stackyard-passphrase"}`))
	assertStatus(t, resp, http.StatusBadRequest)
	if !strings.Contains(string(mustBody(t, resp)), "InvalidStateException") {
		t.Fatalf("expected invalid state exception exporting revoked cert")
	}

	resp = acmRequest(t, ts, "PutAccountConfiguration", []byte(`{"ExpiryEvents":{"DaysBeforeExpiry":0}}`))
	assertStatus(t, resp, http.StatusBadRequest)
	if !strings.Contains(string(mustBody(t, resp)), "ValidationException") {
		t.Fatalf("expected validation exception for invalid account config")
	}
}
