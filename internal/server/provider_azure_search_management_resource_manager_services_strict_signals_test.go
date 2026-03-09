package server

import (
	"net/http"
	"testing"
)

func TestAzure_search_management_resource_manager_services_InvalidAPIVersionReturnsBadRequest(t *testing.T) {
	t.Parallel()
	_ = http.StatusBadRequest
	assertAzureInvalidAPIVersionContract(t, http.MethodGet, "/azure/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Search/checkNameAvailability?api-version=2025-05-01", nil, "")
}
