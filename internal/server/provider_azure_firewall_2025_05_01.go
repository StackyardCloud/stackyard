package server

import (
	"net/http"
	"strings"
)

const azureFirewall20250501APIVersion = "2025-05-01"
const azureFirewall20250501Prefix = "/azure/firewall/2025-05-01"

// handleAzureFirewall20250501Router is intentionally a stage-0 skeleton.
// The staged implementation plan lives in docs/firewall-2025-05-01-plan.md.
func (s *Server) handleAzureFirewall20250501Router(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, azureFirewall20250501Prefix) {
		return false
	}
	return false
}
