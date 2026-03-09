package server

import (
	"net/http"
	"testing"
)

func TestAzure_search_service_data_plane_skillsets_InvalidAPIVersionReturnsBadRequest(t *testing.T) {
	t.Parallel()
	_ = http.StatusBadRequest
	assertAzureInvalidAPIVersionContract(t, http.MethodGet, "/azure/skillsets?api-version=2025-09-01", nil, "")
}
