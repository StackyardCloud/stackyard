package server

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
)

const azureAttestationPrefix = "/azure/attestation"

var azureAttestationTypes = map[string]string{
	"azureguest":  "AzureGuest",
	"openenclave": "OpenEnclave",
	"sevsnpvm":    "SevSnpVm",
	"sgxenclave":  "SgxEnclave",
	"tdxvm":       "TdxVm",
	"tpm":         "Tpm",
}

var azureAttestationDefaultPolicies = map[string]string{
	"AzureGuest":  "version=1.0; authorizationrules { => permit(); }; issuancerules { c:[type=\"x-ms-isolation-tee\"] => issue(type=\"x-ms-ver\", value=\"azureguest\"); };",
	"OpenEnclave": "version=1.0; authorizationrules { => permit(); }; issuancerules { c:[type=\"x-ms-isolation-tee\"] => issue(type=\"x-ms-ver\", value=\"openenclave\"); };",
	"SevSnpVm":    "version=1.0; authorizationrules { => permit(); }; issuancerules { c:[type=\"x-ms-isolation-tee\"] => issue(type=\"x-ms-ver\", value=\"sevsnpvm\"); };",
	"SgxEnclave":  "version=1.0; authorizationrules { => permit(); }; issuancerules { c:[type=\"x-ms-isolation-tee\"] => issue(type=\"x-ms-ver\", value=\"sgxenclave\"); };",
	"TdxVm":       "version=1.0; authorizationrules { => permit(); }; issuancerules { c:[type=\"x-ms-isolation-tee\"] => issue(type=\"x-ms-ver\", value=\"tdxvm\"); };",
	"Tpm":         "version=1.0; authorizationrules { => permit(); }; issuancerules { c:[type=\"x-ms-isolation-tee\"] => issue(type=\"x-ms-ver\", value=\"tpm\"); };",
}

var azureAttestationDefaultPolicyCertificates = []string{
	"-----BEGIN CERTIFICATE-----\nU1RBQ0tZQVJELUFUVEVTVEFUSU9OLUNFUlQtMQ==\n-----END CERTIFICATE-----",
}

func (s *Server) handleAzureAttestationRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, azureAttestationPrefix) {
		return false
	}
	if hasAzureInvalidAPIVersion(r) {
		respondAzureInvalidRequest(w, path, "invalid api-version query parameter")
		return true
	}

	relative := strings.TrimPrefix(path, azureAttestationPrefix)
	normalized := strings.Trim(strings.TrimSpace(relative), "/")
	if normalized == "" {
		return false
	}
	segments := splitPathSegments(normalized)
	if len(segments) == 0 {
		return false
	}

	switch {
	case len(segments) == 2 && segments[0] == ".well-known" && strings.EqualFold(segments[1], "openid-configuration"):
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return true
		}
		respondJSON(w, http.StatusOK, azureAttestationOpenIDConfigurationFixture(r))
		return true
	case len(segments) == 1 && strings.EqualFold(segments[0], "certs"):
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return true
		}
		contentType := "application/json"
		if accept := strings.ToLower(strings.TrimSpace(r.Header.Get("Accept"))); strings.Contains(accept, "application/jwk+json") {
			contentType = "application/jwk+json"
		}
		azureAttestationRespondJSON(w, http.StatusOK, contentType, azureAttestationSigningCertificatesFixture())
		return true
	case len(segments) == 1 && strings.EqualFold(segments[0], "certificates"):
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return true
		}
		respondJSON(w, http.StatusOK, map[string]any{
			"token": s.azureAttestationPolicyCertificatesToken("", "Get"),
		})
		return true
	case len(segments) == 1 && strings.EqualFold(segments[0], "certificates:add"):
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return true
		}
		s.handleAzureAttestationPolicyCertificateAdd(w, r, path)
		return true
	case len(segments) == 1 && strings.EqualFold(segments[0], "certificates:remove"):
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return true
		}
		s.handleAzureAttestationPolicyCertificateRemove(w, r, path)
		return true
	case len(segments) == 2 && strings.EqualFold(segments[0], "policies"):
		s.handleAzureAttestationPolicies(w, r, path, segments[1])
		return true
	case len(segments) == 2 && strings.EqualFold(segments[0], "attest"):
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return true
		}
		s.handleAzureAttestationAttest(w, r, path, segments[1])
		return true
	default:
		azureAttestationError(w, http.StatusNotFound, path, "PathNotFound", "route not found")
		return true
	}
}

func (s *Server) handleAzureAttestationPolicyCertificateAdd(w http.ResponseWriter, r *http.Request, path string) {
	certificate, ok := azureAttestationReadPolicyCertificateMutationBody(w, r, path, "policyCertificateToAdd")
	if !ok {
		return
	}
	if certificate == "" {
		azureAttestationError(w, http.StatusBadRequest, path, "InvalidRequest", "policyCertificateToAdd is required")
		return
	}

	result := "Unchanged"
	s.providerStorageMu.Lock()
	found := false
	for _, existing := range s.azureAttestationPolicyCertificates {
		if existing == certificate {
			found = true
			break
		}
	}
	if !found {
		s.azureAttestationPolicyCertificates = append(s.azureAttestationPolicyCertificates, certificate)
		result = "IsPresent"
	}
	s.providerStorageMu.Unlock()

	respondJSON(w, http.StatusOK, map[string]any{
		"token": s.azureAttestationPolicyCertificatesToken(certificate, result),
	})
}

func (s *Server) handleAzureAttestationPolicyCertificateRemove(w http.ResponseWriter, r *http.Request, path string) {
	certificate, ok := azureAttestationReadPolicyCertificateMutationBody(w, r, path, "policyCertificateToRemove")
	if !ok {
		return
	}
	if certificate == "" {
		azureAttestationError(w, http.StatusBadRequest, path, "InvalidRequest", "policyCertificateToRemove is required")
		return
	}

	result := "IsAbsent"
	s.providerStorageMu.Lock()
	if len(s.azureAttestationPolicyCertificates) <= 1 {
		s.providerStorageMu.Unlock()
		azureAttestationError(w, http.StatusBadRequest, path, "InvalidRequest", "the final policy management certificate cannot be removed")
		return
	}
	filtered := make([]string, 0, len(s.azureAttestationPolicyCertificates))
	removed := false
	for _, existing := range s.azureAttestationPolicyCertificates {
		if !removed && existing == certificate {
			removed = true
			continue
		}
		filtered = append(filtered, existing)
	}
	if !removed {
		result = "Unchanged"
	} else {
		s.azureAttestationPolicyCertificates = filtered
	}
	s.providerStorageMu.Unlock()

	respondJSON(w, http.StatusOK, map[string]any{
		"token": s.azureAttestationPolicyCertificatesToken(certificate, result),
	})
}

func (s *Server) handleAzureAttestationPolicies(w http.ResponseWriter, r *http.Request, path, rawType string) {
	reset := false
	if base, ok := strings.CutSuffix(rawType, ":reset"); ok {
		rawType = base
		reset = true
	}
	attestationType := azureAttestationCanonicalType(rawType)
	if attestationType == "" {
		azureAttestationError(w, http.StatusBadRequest, path, "InvalidRequest", "unsupported attestationType")
		return
	}

	switch {
	case reset && r.Method == http.MethodPost:
		body, ok := azureAttestationReadPlainTextBody(w, r, path, "policyJws is required")
		if !ok {
			return
		}
		s.providerStorageMu.Lock()
		s.azureAttestationPolicies[attestationType] = azureAttestationDefaultPolicy(attestationType)
		s.providerStorageMu.Unlock()
		respondJSON(w, http.StatusOK, map[string]any{
			"token": s.azureAttestationPolicyToken(attestationType, "Reset", body),
		})
		return
	case reset:
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	case !reset && r.Method == http.MethodGet:
		respondJSON(w, http.StatusOK, map[string]any{
			"token": s.azureAttestationPolicyToken(attestationType, "Get", ""),
		})
		return
	case !reset && r.Method == http.MethodPut:
		body, ok := azureAttestationReadPlainTextBody(w, r, path, "newAttestationPolicy is required")
		if !ok {
			return
		}
		s.providerStorageMu.Lock()
		s.azureAttestationPolicies[attestationType] = body
		s.providerStorageMu.Unlock()
		respondJSON(w, http.StatusOK, map[string]any{
			"token": s.azureAttestationPolicyToken(attestationType, "Set", body),
		})
		return
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleAzureAttestationAttest(w http.ResponseWriter, r *http.Request, path, rawType string) {
	attestationType := azureAttestationCanonicalType(rawType)
	if attestationType == "" {
		azureAttestationError(w, http.StatusBadRequest, path, "InvalidRequest", "unsupported attestationType")
		return
	}

	_, payload, ok := azureAttestationReadJSONBody(w, r, path)
	if !ok {
		return
	}
	if message := azureAttestationValidateAttestationRequest(attestationType, payload); message != "" {
		azureAttestationError(w, http.StatusBadRequest, path, "InvalidRequest", message)
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"token": s.azureAttestationAttestationToken(attestationType, payload),
	})
}

func (s *Server) azureAttestationPolicyCertificatesToken(certificate, resolution string) string {
	s.providerStorageMu.Lock()
	certificates := append([]string(nil), s.azureAttestationPolicyCertificates...)
	s.providerStorageMu.Unlock()
	sort.Strings(certificates)

	return azureAttestationJWT(map[string]any{
		"AttestationType":             "PolicyCertificates",
		"CertificateCount":            len(certificates),
		"Certificates":                certificates,
		"x-ms-certificate-thumbprint": azureAttestationThumbprint(certificate),
		"x-ms-policycertificates-result": map[string]any{
			"certificateResolution": azureAttestationDefaultString(resolution, "Unchanged"),
		},
		"iss": "stackyard-attestation",
	})
}

func (s *Server) azureAttestationPolicyToken(attestationType, action, rawBody string) string {
	s.providerStorageMu.Lock()
	policy := s.azureAttestationPolicies[attestationType]
	s.providerStorageMu.Unlock()
	if strings.TrimSpace(policy) == "" {
		policy = azureAttestationDefaultPolicy(attestationType)
	}
	return azureAttestationJWT(map[string]any{
		"AttestationPolicy": base64.RawURLEncoding.EncodeToString([]byte(policy)),
		"AttestationType":   attestationType,
		"Action":            action,
		"InputDigest":       azureAttestationDigest(rawBody),
		"iss":               "stackyard-attestation",
	})
}

func (s *Server) azureAttestationAttestationToken(attestationType string, payload map[string]any) string {
	s.providerStorageMu.Lock()
	policy := s.azureAttestationPolicies[attestationType]
	s.providerStorageMu.Unlock()
	if strings.TrimSpace(policy) == "" {
		policy = azureAttestationDefaultPolicy(attestationType)
	}

	claims := map[string]any{
		"AttestationType": attestationType,
		"PolicyHash":      azureAttestationDigest(policy),
		"EvidenceDigest":  azureAttestationDigest(azureAttestationJSONString(payload)),
		"iss":             "stackyard-attestation",
		"x-ms-ver":        "1.0",
	}
	if nonce := strings.TrimSpace(azureAttestationString(payload["nonce"])); nonce != "" {
		claims["nonce"] = nonce
	}
	if summary := azureAttestationEvidenceSummary(attestationType, payload); summary != "" {
		claims["EvidenceType"] = summary
	}
	return azureAttestationJWT(claims)
}

func azureAttestationOpenIDConfigurationFixture(r *http.Request) map[string]any {
	baseURL := azureAttestationBaseURL(r)
	return map[string]any{
		"issuer":                                baseURL,
		"jwks_uri":                              baseURL + "/certs",
		"revocation_endpoint":                   baseURL + "/certificates",
		"claims_supported":                      []string{"iss", "jti", "iat", "nbf", "exp", "nonce", "x-ms-ver"},
		"id_token_signing_alg_values_supported": []string{"RS256"},
		"response_types_supported":              []string{"token"},
	}
}

func azureAttestationSigningCertificatesFixture() map[string]any {
	return map[string]any{
		"keys": []map[string]any{
			{
				"kty": "RSA",
				"use": "sig",
				"kid": "stackyard-attestation-key",
				"alg": "RS256",
				"n":   "sXchQXR0ZXN0YXRpb25TdGFja3lhcmRTaWduaW5nS2V5",
				"e":   "AQAB",
				"x5c": []string{"U1RBQ0tZQVJELUFUVEVTVEFUSU9OLVNJR05JTkctQ0VSVA"},
			},
		},
	}
}

func azureAttestationBaseURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")); forwarded != "" {
		scheme = forwarded
	}
	host := strings.TrimSpace(r.Host)
	if host == "" {
		host = "localhost:4566"
	}
	return fmt.Sprintf("%s://%s%s", scheme, host, azureAttestationPrefix)
}

func azureAttestationValidateAttestationRequest(attestationType string, payload map[string]any) string {
	switch attestationType {
	case "AzureGuest":
		if strings.TrimSpace(azureAttestationString(payload["attestationInfo"])) == "" {
			return "attestationInfo is required"
		}
	case "Tpm":
		if strings.TrimSpace(azureAttestationString(payload["data"])) == "" {
			return "data is required"
		}
	default:
		if len(payload) == 0 {
			return "request body must contain attestation evidence"
		}
	}
	return ""
}

func azureAttestationCanonicalType(raw string) string {
	value := strings.ToLower(strings.TrimSpace(raw))
	return azureAttestationTypes[value]
}

func azureAttestationDefaultPolicy(attestationType string) string {
	if value := strings.TrimSpace(azureAttestationDefaultPolicies[attestationType]); value != "" {
		return value
	}
	return "version=1.0; authorizationrules { => permit(); };"
}

func azureAttestationReadJSONBody(w http.ResponseWriter, r *http.Request, path string) ([]byte, map[string]any, bool) {
	body, err := readBodyBytes(r)
	if err != nil {
		azureAttestationError(w, http.StatusBadRequest, path, "InvalidRequest", "unable to read request body")
		return nil, nil, false
	}
	if strings.TrimSpace(string(body)) == "" {
		azureAttestationError(w, http.StatusBadRequest, path, "InvalidRequest", "request body is required")
		return body, nil, false
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		azureAttestationError(w, http.StatusBadRequest, path, "InvalidRequest", "request body must be valid JSON")
		return body, nil, false
	}
	return body, payload, true
}

func azureAttestationReadPlainTextBody(w http.ResponseWriter, r *http.Request, path, emptyMessage string) (string, bool) {
	body, err := readBodyBytes(r)
	if err != nil {
		azureAttestationError(w, http.StatusBadRequest, path, "InvalidRequest", "unable to read request body")
		return "", false
	}
	value := strings.TrimSpace(string(body))
	if value == "" {
		azureAttestationError(w, http.StatusBadRequest, path, "InvalidRequest", azureAttestationDefaultString(emptyMessage, "request body is required"))
		return "", false
	}
	var decoded string
	if json.Unmarshal(body, &decoded) == nil && strings.TrimSpace(decoded) != "" {
		return strings.TrimSpace(decoded), true
	}
	return value, true
}

func azureAttestationReadPolicyCertificateMutationBody(w http.ResponseWriter, r *http.Request, path, field string) (string, bool) {
	body, err := readBodyBytes(r)
	if err != nil {
		azureAttestationError(w, http.StatusBadRequest, path, "InvalidRequest", "unable to read request body")
		return "", false
	}
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		azureAttestationError(w, http.StatusBadRequest, path, "InvalidRequest", field+" is required")
		return "", false
	}

	var payload map[string]any
	if json.Unmarshal(body, &payload) == nil {
		if value := strings.TrimSpace(azureAttestationString(payload[field])); value != "" {
			return value, true
		}
		azureAttestationError(w, http.StatusBadRequest, path, "InvalidRequest", field+" is required")
		return "", false
	}

	var decoded string
	if json.Unmarshal(body, &decoded) == nil {
		if value := strings.TrimSpace(decoded); value != "" {
			return value, true
		}
	}

	return trimmed, true
}

func azureAttestationError(w http.ResponseWriter, status int, path, code, message string) {
	if strings.TrimSpace(code) == "" {
		code = "InvalidRequest"
	}
	if strings.TrimSpace(message) == "" {
		message = "invalid request"
	}
	w.Header().Set("x-ms-error-code", code)
	respondJSON(w, status, map[string]any{
		"error": map[string]any{
			"code":    code,
			"message": message,
			"details": []map[string]any{
				{
					"code":    code,
					"message": message,
					"target":  path,
				},
			},
		},
		"provider": providerAzure,
		"path":     path,
	})
}

func azureAttestationRespondJSON(w http.ResponseWriter, status int, contentType string, body any) {
	if strings.TrimSpace(contentType) == "" {
		contentType = "application/json"
	}
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func azureAttestationJWT(claims map[string]any) string {
	header := map[string]any{
		"alg": "RS256",
		"kid": "stackyard-attestation-key",
		"typ": "JWT",
	}
	headerJSON, _ := json.Marshal(header)
	claimsJSON, _ := json.Marshal(claims)
	signature := base64.RawURLEncoding.EncodeToString([]byte("stackyard-signature"))
	return base64.RawURLEncoding.EncodeToString(headerJSON) + "." + base64.RawURLEncoding.EncodeToString(claimsJSON) + "." + signature
}

func azureAttestationDigest(value string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(value)))
	return hex.EncodeToString(sum[:8])
}

func azureAttestationThumbprint(value string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(value))
	return strings.ToUpper(hex.EncodeToString(sum[:8]))
}

func azureAttestationEvidenceSummary(attestationType string, payload map[string]any) string {
	switch attestationType {
	case "AzureGuest":
		if strings.TrimSpace(azureAttestationString(payload["attestationInfo"])) != "" {
			return "attestationInfo"
		}
	case "OpenEnclave":
		if strings.TrimSpace(azureAttestationString(payload["report"])) != "" {
			return "report"
		}
	case "SgxEnclave":
		if strings.TrimSpace(azureAttestationString(payload["quote"])) != "" {
			return "quote"
		}
	case "SevSnpVm":
		if strings.TrimSpace(azureAttestationString(payload["report"])) != "" {
			return "report"
		}
	case "TdxVm":
		if strings.TrimSpace(azureAttestationString(payload["quote"])) != "" {
			return "quote"
		}
	case "Tpm":
		if strings.TrimSpace(azureAttestationString(payload["data"])) != "" {
			return "data"
		}
	}
	return "payload"
}

func azureAttestationJSONString(payload map[string]any) string {
	encoded, _ := json.Marshal(payload)
	return string(encoded)
}

func azureAttestationString(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	default:
		return ""
	}
}

func azureAttestationDefaultString(value, fallback string) string {
	if strings.TrimSpace(value) != "" {
		return value
	}
	return fallback
}
