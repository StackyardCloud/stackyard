package server

import (
	"net/http"
	"testing"
)

func TestAzure_search_management_resource_manager_network_security_perimeter_configurations_InvalidAPIVersionReturnsBadRequest(t *testing.T) {
	t.Parallel()
	_ = http.StatusBadRequest
	assertAzureInvalidAPIVersionContract(t, http.MethodGet, "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-search/providers/Microsoft.Search/searchServices/my-search/networkSecurityPerimeterConfigurations?api-version=2025-05-01", nil, "")
}
