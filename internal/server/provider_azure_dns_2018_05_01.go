package server

import (
	"net/http"
	"strings"
)

const azureDNS20180501APIVersion = "2018-05-01"

// handleAzureDNS20180501Router is intentionally a stage-0 skeleton.
// The staged implementation plan lives in docs/dns-2018-05-01-plan.md.
func (s *Server) handleAzureDNS20180501Router(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	normalized := strings.ToLower(path)
	if !strings.Contains(normalized, "/providers/microsoft.network/dnszones") &&
		!strings.Contains(normalized, "/providers/microsoft.network/getdnsresourcereference") {
		return false
	}
	return false
}
