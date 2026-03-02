package server

import (
	"net/http"
	"strings"
)

const (
	defaultGCPAuthMode   = "emulator"
	defaultAzureAuthMode = "shared_key_or_sas"
	defaultOCIAuthMode   = "signature"
)

type awsRequestAuthValidator interface {
	ValidateWithService(r *http.Request, requiredService string) (bool, int, string, string, bool)
}

type providerRequestAuthValidator interface {
	Validate(r *http.Request) (bool, int, string, string)
}

type awsSigV4AuthAdapter struct {
	server *Server
}

func (a *awsSigV4AuthAdapter) ValidateWithService(r *http.Request, requiredService string) (bool, int, string, string, bool) {
	if a == nil || a.server == nil {
		return false, http.StatusInternalServerError, "InternalFailure", "aws auth adapter is not initialized", false
	}
	return a.server.validateSigV4WithServiceCore(r, requiredService)
}

type gcpAuthValidator struct {
	mode string
}

func (v gcpAuthValidator) Validate(r *http.Request) (bool, int, string, string) {
	mode := normalizeGCPAuthMode(v.mode)
	if mode == "emulator" {
		return true, 0, "", ""
	}
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	if auth == "" {
		if mode == "bearer_required" {
			return false, http.StatusUnauthorized, "Unauthorized", "missing Bearer token"
		}
		return true, 0, "", ""
	}
	if !strings.HasPrefix(strings.ToLower(auth), "bearer ") {
		return false, http.StatusUnauthorized, "Unauthorized", "unsupported authorization scheme; expected Bearer token"
	}
	token := strings.TrimSpace(auth[len("Bearer "):])
	if token == "" {
		return false, http.StatusUnauthorized, "Unauthorized", "empty Bearer token"
	}
	return true, 0, "", ""
}

type azureAuthValidator struct {
	mode string
}

func (v azureAuthValidator) Validate(r *http.Request) (bool, int, string, string) {
	mode := normalizeAzureAuthMode(v.mode)
	if mode == "disabled" {
		return true, 0, "", ""
	}
	hasSharedKey := hasAzureSharedKeyAuth(r)
	hasSAS := hasAzureSASAuth(r)
	switch mode {
	case "shared_key":
		if hasSharedKey {
			return true, 0, "", ""
		}
		return false, http.StatusUnauthorized, "AuthenticationFailed", "missing or invalid Azure Shared Key authorization"
	case "sas":
		if hasSAS {
			return true, 0, "", ""
		}
		return false, http.StatusUnauthorized, "AuthenticationFailed", "missing or invalid Azure SAS signature"
	default:
		if hasSharedKey || hasSAS {
			return true, 0, "", ""
		}
		return false, http.StatusUnauthorized, "AuthenticationFailed", "missing Azure Shared Key or SAS authentication"
	}
}

type ociAuthValidator struct {
	mode string
}

func (v ociAuthValidator) Validate(r *http.Request) (bool, int, string, string) {
	mode := normalizeOCIAuthMode(v.mode)
	if mode == "disabled" {
		return true, 0, "", ""
	}
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	if !strings.HasPrefix(strings.ToLower(auth), "signature ") {
		return false, http.StatusUnauthorized, "NotAuthenticated", "missing OCI Signature authorization header"
	}
	params := parseAuthParams(strings.TrimSpace(auth[len("Signature "):]))
	required := []string{"keyId", "algorithm", "signature", "headers"}
	for _, key := range required {
		if strings.TrimSpace(params[key]) == "" {
			return false, http.StatusUnauthorized, "NotAuthenticated", "invalid OCI Signature authorization header"
		}
	}
	date := strings.TrimSpace(r.Header.Get("Date"))
	xDate := strings.TrimSpace(r.Header.Get("X-Date"))
	if date == "" && xDate == "" {
		return false, http.StatusUnauthorized, "NotAuthenticated", "missing Date or X-Date header required by OCI signature validation"
	}
	return true, 0, "", ""
}

func newProviderAuthValidators(cfg Config) map[string]providerRequestAuthValidator {
	return map[string]providerRequestAuthValidator{
		providerGCP:   gcpAuthValidator{mode: normalizeGCPAuthMode(cfg.GCPAuthMode)},
		providerAzure: azureAuthValidator{mode: normalizeAzureAuthMode(cfg.AzureAuthMode)},
		providerOCI:   ociAuthValidator{mode: normalizeOCIAuthMode(cfg.OCIAuthMode)},
	}
}

func normalizeGCPAuthMode(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", defaultGCPAuthMode:
		return defaultGCPAuthMode
	case "bearer_tolerant", "bearer_required":
		return strings.ToLower(strings.TrimSpace(raw))
	default:
		return defaultGCPAuthMode
	}
}

func normalizeAzureAuthMode(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", defaultAzureAuthMode:
		return defaultAzureAuthMode
	case "shared_key", "sas", "disabled":
		return strings.ToLower(strings.TrimSpace(raw))
	default:
		return defaultAzureAuthMode
	}
}

func normalizeOCIAuthMode(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", defaultOCIAuthMode:
		return defaultOCIAuthMode
	case "disabled":
		return "disabled"
	default:
		return defaultOCIAuthMode
	}
}

func hasAzureSharedKeyAuth(r *http.Request) bool {
	if r == nil {
		return false
	}
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	if !strings.HasPrefix(strings.ToLower(auth), "sharedkey ") {
		return false
	}
	value := strings.TrimSpace(auth[len("SharedKey "):])
	account, signature, ok := strings.Cut(value, ":")
	return ok && strings.TrimSpace(account) != "" && strings.TrimSpace(signature) != ""
}

func hasAzureSASAuth(r *http.Request) bool {
	if r == nil || r.URL == nil {
		return false
	}
	return strings.TrimSpace(r.URL.Query().Get("sig")) != ""
}

func parseAuthParams(value string) map[string]string {
	parts := strings.Split(value, ",")
	out := make(map[string]string, len(parts))
	for _, part := range parts {
		key, val, ok := strings.Cut(strings.TrimSpace(part), "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(strings.Trim(val, `"`))
		if key == "" {
			continue
		}
		out[key] = val
	}
	return out
}
