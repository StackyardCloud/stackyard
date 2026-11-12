package server

import (
	"net/http"
	"strings"
)

const azureDNSDNSResolver20250501APIVersion = "2025-05-01"

// handleAzureDNSDNSResolver20250501Router is intentionally a stage-0 skeleton.
// The staged implementation plan lives in docs/dns-dnsresolver-2025-05-01-plan.md.
func (s *Server) handleAzureDNSDNSResolver20250501Router(w http.ResponseWriter, r *http.Request) bool {
	path := strings.ToLower(rawRequestPath(r))
	if !strings.Contains(path, "/providers/microsoft.network/dnsresolvers") &&
		!strings.Contains(path, "/providers/microsoft.network/dnsforwardingrulesets") &&
		!strings.Contains(path, "/providers/microsoft.network/dnsresolverdomainlists") &&
		!strings.Contains(path, "/providers/microsoft.network/dnsresolverpolicies") &&
		!strings.Contains(path, "/listdnsresolvers") &&
		!strings.Contains(path, "/listdnsforwardingrulesets") &&
		!strings.Contains(path, "/listdnsresolverpolicies") {
		return false
	}
	return false
}
