package server

import (
	"net/http"
	"testing"
)

func TestAzure_ai_services_miscellaneous_operations_InvalidAPIVersionReturnsBadRequest(t *testing.T) {
	t.Parallel()
	_ = http.StatusBadRequest
	assertAzureInvalidAPIVersionContract(t, http.MethodGet, "/azure/aiservices/documentintelligence/operations/op-123?_overload=getOperation&api-version=2024-11-30", nil, "")
}
