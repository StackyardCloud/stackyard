package server

import (
	"net/http"
	"testing"
)

func TestAzure_ai_services_language_question_answering_InvalidAPIVersionReturnsBadRequest(t *testing.T) {
	t.Parallel()
	_ = http.StatusBadRequest
	assertAzureInvalidAPIVersionContract(t, http.MethodGet, "/azure/language/:query-knowledgebases?api-version=2023-04-01&projectName=proj-a&deploymentName=production", nil, "")
}
