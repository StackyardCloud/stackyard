package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestPaymentCryptographyDataStage0CatalogCoverage(t *testing.T) {
	if len(paymentCryptographyDataOperations) != 14 {
		t.Fatalf("expected 14 Payment Cryptography Data operations from docs, got %d", len(paymentCryptographyDataOperations))
	}
	if len(paymentCryptographyDataOperationByName) != len(paymentCryptographyDataOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"DecryptData",
		"EncryptData",
		"GenerateMac",
		"GeneratePinData",
		"VerifyPinData",
		"TranslateKeyMaterial",
		"GenerateAs2805KekValidation",
	}
	for _, action := range requiredActions {
		if _, ok := paymentCryptographyDataOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(paymentCryptographyDataPlaneTypes) != 69 {
		t.Fatalf("expected 69 Payment Cryptography Data types from docs, got %d", len(paymentCryptographyDataPlaneTypes))
	}
	if len(paymentCryptographyDataPlaneTypeByName) != len(paymentCryptographyDataPlaneTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"WrappedKey",
		"WrappedKeyMaterial",
		"EncryptionDecryptionAttributes",
		"PinData",
		"CardGenerationAttributes",
		"MacAttributes",
	}
	for _, typeName := range requiredTypes {
		if _, ok := paymentCryptographyDataPlaneTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func paymentCryptographyDataRequest(t *testing.T, ts *httptest.Server, path, payload string) *http.Response {
	t.Helper()
	headers := map[string]string{}
	body := []byte(payload)
	if strings.TrimSpace(payload) != "" {
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, http.MethodPost, ts.URL+path, body, headers, "payment-cryptography")
}

func paymentCryptographyDataPathForOperation(op paymentCryptographyDataOperation) string {
	path := op.URI
	replacements := map[string]string{
		"{KeyIdentifier}":         url.PathEscape("arn:aws:payment-cryptography:us-east-1:123456789012:key/stackyard-key"),
		"{IncomingKeyIdentifier}": url.PathEscape("arn:aws:payment-cryptography:us-east-1:123456789012:key/stackyard-key"),
	}
	for key, value := range replacements {
		path = strings.ReplaceAll(path, key, value)
	}
	return path
}

func TestPaymentCryptographyDataStage0UnknownRouteReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := paymentCryptographyDataRequest(t, ts, "/keys/stackyard-key/unknown", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestPaymentCryptographyDataStage0KnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := paymentCryptographyDataRequest(t, ts, "/mac/verify", `{}`)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "MacValid") {
		t.Fatalf("expected VerifyMac response body to include MacValid, got %q", body)
	}
}

func TestPaymentCryptographyDataStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range paymentCryptographyDataOperations {
		resp := paymentCryptographyDataRequest(t, ts, paymentCryptographyDataPathForOperation(op), `{}`)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
