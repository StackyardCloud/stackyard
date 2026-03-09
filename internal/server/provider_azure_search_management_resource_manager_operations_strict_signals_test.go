package server

import (
	"net/http"
	"testing"
)

func TestAzure_search_management_resource_manager_operations_InvalidAPIVersionReturnsBadRequest(t *testing.T) {
	t.Parallel()
	_ = http.StatusBadRequest
	assertAzureInvalidAPIVersionContract(t, http.MethodGet, "/azure/providers/Microsoft.Search/operations?api-version=2025-05-01", nil, "")
}
