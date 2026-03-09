package server

import (
	"net/http"
	"testing"
)

func TestAzure_ai_services_language_analyze_text_authoring_InvalidAPIVersionReturnsBadRequest(t *testing.T) {
	t.Parallel()
	_ = http.StatusBadRequest
	assertAzureInvalidAPIVersionContract(t, http.MethodGet, "/azure/language/authoring/analyze-text/projects/proj-a/supportedLanguages?api-version=2023-04-01", nil, "")
}
