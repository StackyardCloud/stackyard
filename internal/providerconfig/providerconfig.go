package providerconfig

import (
	"fmt"
	"strings"
)

const (
	ProviderAWS   = "aws"
	ProviderGCP   = "gcp"
	ProviderAzure = "azure"
	ProviderOCI   = "oci"

	DefaultGCPAuthMode   = "emulator"
	DefaultAzureAuthMode = "shared_key_or_sas"
	DefaultOCIAuthMode   = "signature"
)

var providerOrder = []string{ProviderAWS, ProviderGCP, ProviderAzure, ProviderOCI}

var providerSet = map[string]struct{}{
	ProviderAWS:   {},
	ProviderGCP:   {},
	ProviderAzure: {},
	ProviderOCI:   {},
}

var gcpAuthModeOrder = []string{
	DefaultGCPAuthMode,
	"bearer_tolerant",
	"bearer_required",
}

var azureAuthModeOrder = []string{
	DefaultAzureAuthMode,
	"shared_key",
	"sas",
	"disabled",
}

var ociAuthModeOrder = []string{
	DefaultOCIAuthMode,
	"disabled",
}

func SupportedProviders() []string {
	out := make([]string, len(providerOrder))
	copy(out, providerOrder)
	return out
}

func SupportedProvidersText() string {
	return strings.Join(providerOrder, ", ")
}

func IsSupportedProvider(provider string) bool {
	_, ok := providerSet[strings.ToLower(strings.TrimSpace(provider))]
	return ok
}

func NormalizeEnabledProviders(raw []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		provider := strings.ToLower(strings.TrimSpace(item))
		if provider == "" {
			continue
		}
		if !IsSupportedProvider(provider) {
			continue
		}
		if _, ok := seen[provider]; ok {
			continue
		}
		seen[provider] = struct{}{}
		out = append(out, provider)
	}
	if len(out) == 0 {
		return []string{ProviderAWS}
	}
	return out
}

func ParseProvidersCSV(raw string) ([]string, error) {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for _, part := range parts {
		provider := strings.ToLower(strings.TrimSpace(part))
		if provider == "" {
			continue
		}
		if _, ok := seen[provider]; ok {
			continue
		}
		if !IsSupportedProvider(provider) {
			return nil, fmt.Errorf("unsupported provider %q (supported: %s)", provider, SupportedProvidersText())
		}
		seen[provider] = struct{}{}
		out = append(out, provider)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("at least one provider must be set (supported: %s)", SupportedProvidersText())
	}
	return out, nil
}

func GCPAuthModes() []string {
	out := make([]string, len(gcpAuthModeOrder))
	copy(out, gcpAuthModeOrder)
	return out
}

func AzureAuthModes() []string {
	out := make([]string, len(azureAuthModeOrder))
	copy(out, azureAuthModeOrder)
	return out
}

func OCIAuthModes() []string {
	out := make([]string, len(ociAuthModeOrder))
	copy(out, ociAuthModeOrder)
	return out
}

func GCPAuthModesText() string {
	return strings.Join(gcpAuthModeOrder, ", ")
}

func AzureAuthModesText() string {
	return strings.Join(azureAuthModeOrder, ", ")
}

func OCIAuthModesText() string {
	return strings.Join(ociAuthModeOrder, ", ")
}

func NormalizeGCPAuthMode(raw string) (string, bool) {
	mode := strings.ToLower(strings.TrimSpace(raw))
	switch mode {
	case "", DefaultGCPAuthMode:
		return DefaultGCPAuthMode, true
	case "bearer_tolerant", "bearer_required":
		return mode, true
	default:
		return DefaultGCPAuthMode, false
	}
}

func NormalizeAzureAuthMode(raw string) (string, bool) {
	mode := strings.ToLower(strings.TrimSpace(raw))
	switch mode {
	case "", DefaultAzureAuthMode:
		return DefaultAzureAuthMode, true
	case "shared_key", "sas", "disabled":
		return mode, true
	default:
		return DefaultAzureAuthMode, false
	}
}

func NormalizeOCIAuthMode(raw string) (string, bool) {
	mode := strings.ToLower(strings.TrimSpace(raw))
	switch mode {
	case "", DefaultOCIAuthMode:
		return DefaultOCIAuthMode, true
	case "disabled":
		return "disabled", true
	default:
		return DefaultOCIAuthMode, false
	}
}
