package server

import (
	"net/http"
	"strings"
)

const azureDNSPrivateDNS20180901APIVersion = "2018-09-01"

// handleAzureDNSPrivateDNS20180901Router is intentionally a stage-0 skeleton.
// The staged implementation plan lives in docs/dns-privatedns-2018-09-01-plan.md.
func (s *Server) handleAzureDNSPrivateDNS20180901Router(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.Contains(strings.ToLower(path), "/providers/microsoft.network/privatednszones") {
		return false
	}
	return false
}
