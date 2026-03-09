package server

import (
	"net/http"
	"testing"
)

func TestAzure_search_service_data_plane_documents_InvalidAPIVersionReturnsBadRequest(t *testing.T) {
	t.Parallel()
	_ = http.StatusBadRequest
	assertAzureInvalidAPIVersionContract(t, http.MethodGet, "/azure/indexes('hotels')/docs/search.autocomplete?api-version=2025-09-01&search=washington&suggesterName=sg", nil, "")
}
