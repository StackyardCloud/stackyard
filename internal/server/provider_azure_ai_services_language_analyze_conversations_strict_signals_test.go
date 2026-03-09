package server

import (
	"net/http"
	"testing"
)

func TestAzure_ai_services_language_analyze_conversations_InvalidAPIVersionReturnsBadRequest(t *testing.T) {
	t.Parallel()
	_ = http.StatusBadRequest
	assertAzureInvalidAPIVersionContract(t, http.MethodGet, "/azure/language/:analyze-conversations?api-version=2024-11-01&showStats=true", nil, "")
}
