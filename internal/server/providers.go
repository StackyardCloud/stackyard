package server

import (
	"strings"

	"github.com/stackyard/stackyard/internal/providerconfig"
)

const (
	providerAWS   = providerconfig.ProviderAWS
	providerGCP   = providerconfig.ProviderGCP
	providerAzure = providerconfig.ProviderAzure
	providerOCI   = providerconfig.ProviderOCI
)

var supportedProviderSet = toProviderSet(providerconfig.SupportedProviders())

func supportedProviders() []string {
	return providerconfig.SupportedProviders()
}

func normalizeEnabledProviders(raw []string) []string {
	return providerconfig.NormalizeEnabledProviders(raw)
}

func toProviderSet(providers []string) map[string]struct{} {
	out := make(map[string]struct{}, len(providers))
	for _, provider := range providers {
		out[provider] = struct{}{}
	}
	return out
}

func providerFromPath(rawPath string) string {
	path := strings.TrimSpace(rawPath)
	switch {
	case strings.HasPrefix(path, "/maps."):
		return providerGCP
	case strings.HasPrefix(path, "/google."):
		return providerGCP
	case path == "/gcp" || strings.HasPrefix(path, "/gcp/"):
		return providerGCP
	case path == "/azure" || strings.HasPrefix(path, "/azure/"):
		return providerAzure
	case path == "/oci" || strings.HasPrefix(path, "/oci/"):
		return providerOCI
	default:
		return providerAWS
	}
}
