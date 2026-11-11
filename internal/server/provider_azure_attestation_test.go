package server

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestAzureAttestationLifecycle(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	headers := map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
	}
	jsonHeaders := map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
		"Content-Type":              "application/json",
	}
	textHeaders := map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
		"Content-Type":              "text/plain",
	}

	resp := providerContractRequest(t, ts, http.MethodGet, "/azure/attestation/.well-known/openid-configuration?api-version=2025-06-01", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for metadata, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	metadata := providerContractJSONMap(t, resp)
	if metadata["issuer"] == "" || metadata["jwks_uri"] == "" {
		t.Fatalf("unexpected metadata payload: %#v", metadata)
	}

	jwkHeaders := map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
		"Accept":                    "application/jwk+json",
	}
	resp = providerContractRequest(t, ts, http.MethodGet, "/azure/attestation/certs?api-version=2025-06-01", nil, jwkHeaders)
	body := providerContractBody(t, resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for signing certs, got %d body=%s", resp.StatusCode, string(body))
	}
	if !strings.HasPrefix(resp.Header.Get("Content-Type"), "application/jwk+json") {
		t.Fatalf("expected jwk content type, got %q", resp.Header.Get("Content-Type"))
	}
	signingKeys := mustDecodeJSONMap(t, body)
	keys, ok := signingKeys["keys"].([]any)
	if !ok || len(keys) != 1 {
		t.Fatalf("expected one signing key, got %#v", signingKeys)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, "/azure/attestation/certificates?api-version=2025-06-01", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for policy certificates get, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	certificatePayload := providerContractJSONMap(t, resp)
	certificateClaims := azureAttestationDecodeTokenClaims(t, azureAttestationString(certificatePayload["token"]))
	if certificateClaims["CertificateCount"] != float64(1) {
		t.Fatalf("expected one policy certificate, got %#v", certificateClaims)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, "/azure/attestation/certificates:add?api-version=2025-06-01", []byte(`{"policyCertificateToAdd":"-----BEGIN CERTIFICATE-----\nU1RBQ0tZQVJELUFUVEVTVEFUSU9OLUNFUlQtMg==\n-----END CERTIFICATE-----"}`), jsonHeaders)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for policy certificate add, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	addClaims := azureAttestationDecodeTokenClaims(t, azureAttestationString(providerContractJSONMap(t, resp)["token"]))
	if result, ok := addClaims["x-ms-policycertificates-result"].(map[string]any); !ok || result["certificateResolution"] != "IsPresent" {
		t.Fatalf("unexpected add result payload: %#v", addClaims)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, "/azure/attestation/certificates?api-version=2025-06-01", nil, headers)
	updatedCertificates := azureAttestationDecodeTokenClaims(t, azureAttestationString(providerContractJSONMap(t, resp)["token"]))
	if updatedCertificates["CertificateCount"] != float64(2) {
		t.Fatalf("expected two policy certificates after add, got %#v", updatedCertificates)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, "/azure/attestation/policies/SgxEnclave?api-version=2025-06-01", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for policy get, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	policyClaims := azureAttestationDecodeTokenClaims(t, azureAttestationString(providerContractJSONMap(t, resp)["token"]))
	if policyClaims["AttestationType"] != "SgxEnclave" {
		t.Fatalf("unexpected policy token claims: %#v", policyClaims)
	}

	resp = providerContractRequest(t, ts, http.MethodPut, "/azure/attestation/policies/SgxEnclave?api-version=2025-06-01", []byte("version=1.0; authorizationrules { => permit(); };"), textHeaders)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for policy set, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	policyClaims = azureAttestationDecodeTokenClaims(t, azureAttestationString(providerContractJSONMap(t, resp)["token"]))
	if policyClaims["Action"] != "Set" {
		t.Fatalf("expected Action=Set, got %#v", policyClaims)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, "/azure/attestation/policies/SgxEnclave:reset?api-version=2025-06-01", []byte("eyJhbGciOiJub25lIn0.e30."), textHeaders)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for policy reset, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	resetClaims := azureAttestationDecodeTokenClaims(t, azureAttestationString(providerContractJSONMap(t, resp)["token"]))
	if resetClaims["Action"] != "Reset" {
		t.Fatalf("expected Action=Reset, got %#v", resetClaims)
	}

	attestCases := []struct {
		name            string
		path            string
		body            string
		wantType        string
		wantEvidenceKey string
	}{
		{"azure guest", "/azure/attestation/attest/AzureGuest?api-version=2025-06-01", `{"attestationInfo":"eyJ2bSI6Imd1ZXN0In0"}`, "AzureGuest", "attestationInfo"},
		{"open enclave", "/azure/attestation/attest/OpenEnclave?api-version=2025-06-01", `{"report":"b3BlbmVuY2xhdmUtcmVwb3J0","nonce":"nonce-oe"}`, "OpenEnclave", "report"},
		{"sev snp vm", "/azure/attestation/attest/SevSnpVm?api-version=2025-06-01", `{"report":"c2V2c25wLXJlcG9ydA=="}`, "SevSnpVm", "report"},
		{"sgx enclave", "/azure/attestation/attest/SgxEnclave?api-version=2025-06-01", `{"quote":"c2d4LXF1b3Rl","runtimeData":{"data":"cnVudGltZQ","dataType":"JSON"}}`, "SgxEnclave", "quote"},
		{"tdx vm", "/azure/attestation/attest/TdxVm?api-version=2025-06-01", `{"quote":"dGR4LXF1b3Rl","nonce":"nonce-tdx"}`, "TdxVm", "quote"},
		{"tpm", "/azure/attestation/attest/Tpm?api-version=2025-06-01", `{"data":"dHBtLWRhdGE","nonce":"nonce-tpm"}`, "Tpm", "data"},
	}
	for _, tt := range attestCases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			resp := providerContractRequest(t, ts, http.MethodPost, tt.path, []byte(tt.body), jsonHeaders)
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("expected 200 for %s, got %d body=%s", tt.path, resp.StatusCode, string(providerContractBody(t, resp)))
			}
			claims := azureAttestationDecodeTokenClaims(t, azureAttestationString(providerContractJSONMap(t, resp)["token"]))
			if claims["AttestationType"] != tt.wantType || claims["EvidenceType"] != tt.wantEvidenceKey {
				t.Fatalf("unexpected attestation claims: %#v", claims)
			}
		})
	}

	resp = providerContractRequest(t, ts, http.MethodPost, "/azure/attestation/certificates:remove?api-version=2025-06-01", []byte(`{"policyCertificateToRemove":"-----BEGIN CERTIFICATE-----\nU1RBQ0tZQVJELUFUVEVTVEFUSU9OLUNFUlQtMg==\n-----END CERTIFICATE-----"}`), jsonHeaders)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for policy certificate remove, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	removeClaims := azureAttestationDecodeTokenClaims(t, azureAttestationString(providerContractJSONMap(t, resp)["token"]))
	if result, ok := removeClaims["x-ms-policycertificates-result"].(map[string]any); !ok || result["certificateResolution"] != "IsAbsent" {
		t.Fatalf("unexpected remove result payload: %#v", removeClaims)
	}
}

func TestAzureAttestationValidation(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	headers := map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
		"Content-Type":              "application/json",
	}

	tests := []struct {
		name       string
		method     string
		path       string
		body       []byte
		headers    map[string]string
		wantStatus int
		wantCode   string
	}{
		{
			name:   "invalid api version",
			method: http.MethodGet,
			path:   "/azure/attestation/certs?api-version=",
			headers: map[string]string{
				"Authorization":             "SharedKey devstoreaccount1:signature",
				"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
			},
			wantStatus: http.StatusBadRequest,
			wantCode:   "InvalidRequest",
		},
		{
			name:       "add certificate missing field",
			method:     http.MethodPost,
			path:       "/azure/attestation/certificates:add?api-version=2025-06-01",
			body:       []byte(`{}`),
			headers:    headers,
			wantStatus: http.StatusBadRequest,
			wantCode:   "InvalidRequest",
		},
		{
			name:   "policy set missing body",
			method: http.MethodPut,
			path:   "/azure/attestation/policies/Tpm?api-version=2025-06-01",
			body:   []byte(""),
			headers: map[string]string{
				"Authorization":             "SharedKey devstoreaccount1:signature",
				"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
				"Content-Type":              "text/plain",
			},
			wantStatus: http.StatusBadRequest,
			wantCode:   "InvalidRequest",
		},
		{
			name:   "unknown attestation type",
			method: http.MethodGet,
			path:   "/azure/attestation/policies/Nope?api-version=2025-06-01",
			headers: map[string]string{
				"Authorization":             "SharedKey devstoreaccount1:signature",
				"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
			},
			wantStatus: http.StatusBadRequest,
			wantCode:   "InvalidRequest",
		},
		{
			name:       "attest tpm missing data",
			method:     http.MethodPost,
			path:       "/azure/attestation/attest/Tpm?api-version=2025-06-01",
			body:       []byte(`{"nonce":"missing-data"}`),
			headers:    headers,
			wantStatus: http.StatusBadRequest,
			wantCode:   "InvalidRequest",
		},
		{
			name:       "attest invalid json",
			method:     http.MethodPost,
			path:       "/azure/attestation/attest/AzureGuest?api-version=2025-06-01",
			body:       []byte(`{`),
			headers:    headers,
			wantStatus: http.StatusBadRequest,
			wantCode:   "InvalidRequest",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			resp := providerContractRequest(t, ts, tt.method, tt.path, tt.body, tt.headers)
			body := providerContractBody(t, resp)
			if resp.StatusCode != tt.wantStatus {
				t.Fatalf("expected %d, got %d body=%s", tt.wantStatus, resp.StatusCode, string(body))
			}
			payload := mustDecodeJSONMap(t, body)
			switch errPayload := payload["error"].(type) {
			case string:
				if errPayload != tt.wantCode {
					t.Fatalf("unexpected error payload: %#v", payload)
				}
			case map[string]any:
				if errPayload["code"] != tt.wantCode {
					t.Fatalf("unexpected error payload: %#v", payload)
				}
			default:
				t.Fatalf("unexpected error payload: %#v", payload)
			}
		})
	}

	resp := providerContractRequest(t, ts, http.MethodPost, "/azure/attestation/certificates:remove?api-version=2025-06-01", []byte(`{"policyCertificateToRemove":"-----BEGIN CERTIFICATE-----\nU1RBQ0tZQVJELUFUVEVTVEFUSU9OLUNFUlQtMQ==\n-----END CERTIFICATE-----"}`), headers)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 removing final certificate, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
}

func azureAttestationDecodeTokenClaims(t *testing.T, token string) map[string]any {
	t.Helper()
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("expected JWT token, got %q", token)
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatalf("decode JWT payload: %v", err)
	}
	var claims map[string]any
	if err := json.Unmarshal(payload, &claims); err != nil {
		t.Fatalf("unmarshal JWT payload: %v", err)
	}
	return claims
}
