package server

import (
	"net/http"
	"testing"
)

func TestAzure_search_management_resource_manager_shared_private_link_resources_InvalidAPIVersionReturnsBadRequest(t *testing.T) {
	t.Parallel()
	_ = http.StatusBadRequest
	assertAzureInvalidAPIVersionContract(t, http.MethodGet, "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-search/providers/Microsoft.Search/searchServices/my-search/sharedPrivateLinkResources/sql?api-version=2025-05-01", nil, "")
}
