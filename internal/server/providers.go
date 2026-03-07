package server

import "strings"

const (
	providerAWS   = "aws"
	providerGCP   = "gcp"
	providerAzure = "azure"
	providerOCI   = "oci"
)

var supportedProviderSet = map[string]struct{}{
	providerAWS:   {},
	providerGCP:   {},
	providerAzure: {},
	providerOCI:   {},
}

func supportedProviders() []string {
	return []string{providerAWS, providerGCP, providerAzure, providerOCI}
}

func normalizeEnabledProviders(raw []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		provider := strings.ToLower(strings.TrimSpace(item))
		if provider == "" {
			continue
		}
		if _, ok := supportedProviderSet[provider]; !ok {
			continue
		}
		if _, ok := seen[provider]; ok {
			continue
		}
		seen[provider] = struct{}{}
		out = append(out, provider)
	}
	if len(out) == 0 {
		return []string{providerAWS}
	}
	return out
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
