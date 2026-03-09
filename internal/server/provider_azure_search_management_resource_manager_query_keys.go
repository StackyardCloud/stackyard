package server

import (
	"net/http"
	"strings"
)

const azureSearchManagementQueryKeysSubscriptionsBase = "/azure/subscriptions/"

func (s *Server) handleAzureSearchManagementResourceManagerQueryKeysRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, azureSearchManagementQueryKeysSubscriptionsBase) {
		return false
	}
	if hasAzureInvalidAPIVersion(r) {
		respondAzureInvalidRequest(w, path, "invalid api-version query parameter")
		return true
	}

	segments := splitPathSegments(strings.TrimPrefix(path, "/azure/"))
	if len(segments) < 9 {
		return false
	}

	if !strings.EqualFold(segments[0], "subscriptions") ||
		segments[1] == "" ||
		!strings.EqualFold(segments[2], "resourceGroups") ||
		segments[3] == "" ||
		!strings.EqualFold(segments[4], "providers") ||
		!strings.EqualFold(segments[5], "Microsoft.Search") ||
		!strings.EqualFold(segments[6], "searchServices") ||
		segments[7] == "" {
		return false
	}

	if len(segments) == 9 && segments[8] == "listQueryKeys" && r.Method == http.MethodPost {
		respondAzureImplemented(w, path)
		return true
	}

	if len(segments) == 10 && segments[8] == "createQueryKey" && segments[9] != "" && r.Method == http.MethodPost {
		respondAzureImplemented(w, path)
		return true
	}

	if len(segments) == 10 && segments[8] == "deleteQueryKey" && segments[9] != "" && r.Method == http.MethodDelete {
		respondAzureImplemented(w, path)
		return true
	}

	return false
}
