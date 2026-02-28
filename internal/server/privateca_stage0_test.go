package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPrivateCAStage0OperationCoverage(t *testing.T) {
	if len(privateCAOperations) != 23 {
		t.Fatalf("expected 23 Private CA operations from docs, got %d", len(privateCAOperations))
	}
	if len(privateCAOperationByName) != len(privateCAOperations) {
		t.Fatalf("expected unique operation names")
	}
	required := []string{
		"CreateCertificateAuthority",
		"DescribeCertificateAuthority",
		"IssueCertificate",
		"GetCertificate",
		"RevokeCertificate",
		"UpdateCertificateAuthority",
	}
	for _, name := range required {
		if _, ok := privateCAOperationByName[name]; !ok {
			t.Fatalf("missing documented operation %s", name)
		}
	}
}

func TestPrivateCACatalogKnownActionIsHandled(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		[]byte(`{"CertificateAuthorityArn":"arn:aws:acm-pca:us-east-1:123456789012:certificate-authority/stackyard"}`),
		map[string]string{
			"Content-Type": "application/x-amz-json-1.1",
			"X-Amz-Target": "ACMPrivateCA.ListCertificateAuthorities",
		},
		"acm-pca",
	)
	if resp.StatusCode == http.StatusNotImplemented {
		t.Fatalf("expected implemented action to avoid %d", http.StatusNotImplemented)
	}
}

func TestPrivateCAStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		[]byte(`{}`),
		map[string]string{
			"Content-Type": "application/x-amz-json-1.1",
			"X-Amz-Target": "ACMPrivateCA.TotallyUnknownAction",
		},
		"acm-pca",
	)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, resp.StatusCode)
	}
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}
