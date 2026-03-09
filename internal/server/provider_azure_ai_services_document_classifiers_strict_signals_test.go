package server

import (
	"net/http"
	"testing"
)

func TestAzure_ai_services_document_classifiers_InvalidAPIVersionReturnsBadRequest(t *testing.T) {
	t.Parallel()
	_ = http.StatusBadRequest
	assertAzureInvalidAPIVersionContract(t, http.MethodGet, "/azure/aiservices/documentintelligence/documentClassifiers:authorizeCopy?api-version=2024-11-30", nil, "")
}
