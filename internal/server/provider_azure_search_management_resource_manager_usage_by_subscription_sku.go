package server

import (
	"net/http"
	"strings"
)

const azureSearchManagementUsageBySubscriptionSKUSubscriptionsBase = "/azure/subscriptions/"

func (s *Server) handleAzureSearchManagementResourceManagerUsageBySubscriptionSKURouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, azureSearchManagementUsageBySubscriptionSKUSubscriptionsBase) {
		return false
	}
	if hasAzureInvalidAPIVersion(r) {
		respondAzureInvalidRequest(w, path, "invalid api-version query parameter")
		return true
	}

	segments := splitPathSegments(strings.TrimPrefix(path, "/azure/"))
	if len(segments) != 8 {
		return false
	}

	if !strings.EqualFold(segments[0], "subscriptions") ||
		segments[1] == "" ||
		!strings.EqualFold(segments[2], "providers") ||
		!strings.EqualFold(segments[3], "Microsoft.Search") ||
		!strings.EqualFold(segments[4], "locations") ||
		segments[5] == "" ||
		!strings.EqualFold(segments[6], "usages") {
		return false
	}
	if segments[7] == "" {
		return false
	}
	if r.Method == http.MethodGet {
		respondAzureImplemented(w, path)
		return true
	}
	return false
}
