package server

import (
	"net/http"
	"strings"
	"testing"
)

const (
	azureLighthouseTestAPIVersion = "?api-version=2022-10-01"
	azureLighthouseTestDefinition = "f1c3f8a1-2a6b-4c3e-9d6b-0b1c2d3e4f50"
	azureLighthouseTestBody       = `{"properties":{"registrationDefinitionName":"Contoso ops","description":"Managed monitoring","managedByTenantId":"2c1f5d3e-8a7b-4c6d-9e0f-1a2b3c4d5e6f","authorizations":[{"principalId":"6f3e2d1c-0b9a-4876-a543-210fedcba987","principalIdDisplayName":"Ops engineers","roleDefinitionId":"acdd72a7-3385-48ef-bd42-f606fba81ae7"}],"eligibleAuthorizations":[{"principalId":"6f3e2d1c-0b9a-4876-a543-210fedcba987","roleDefinitionId":"b24988ac-6180-42a0-ab88-20f7382dd24c","justInTimeAccessPolicy":{"multiFactorAuthProvider":"Azure","maximumActivationDuration":"PT8H"}}]}}`
)

func TestAzureLighthouseLifecycle(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-lh")
	definition := sub + "/providers/Microsoft.ManagedServices/registrationDefinitions/" + azureLighthouseTestDefinition
	resp := do(http.MethodPut, definition+azureLighthouseTestAPIVersion, azureLighthouseTestBody)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	if got := providerContractJSONMap(t, resp)["properties"].(map[string]any)["manageeTenantId"]; got != azureDefaultTenantID {
		t.Fatalf("unexpected managee tenant: %v", got)
	}
	definitionID := strings.TrimPrefix(definition, "/azure")
	for _, scope := range []string{sub, sub + "/resourceGroups/rg-lh"} {
		assignment := scope + "/providers/Microsoft.ManagedServices/registrationAssignments/0b4c8d2e-5f6a-4b7c-8d9e-0f1a2b3c4d5e"
		azureResourcesExpectStatus(t, do(http.MethodPut, assignment+azureLighthouseTestAPIVersion, `{"properties":{"registrationDefinitionId":"`+definitionID+`"}}`), http.StatusCreated)
		resp = do(http.MethodGet, scope+"/providers/Microsoft.ManagedServices/registrationAssignments"+azureLighthouseTestAPIVersion, "")
		if got := providerContractJSONMap(t, resp)["value"].([]any); len(got) != 1 {
			t.Fatalf("expected one assignment at %s, got %d", scope, len(got))
		}
		azureResourcesExpectErrorCode(t, do(http.MethodPatch, assignment+azureLighthouseTestAPIVersion, `{}`), http.StatusMethodNotAllowed, "MethodNotAllowed")
		azureResourcesExpectStatus(t, do(http.MethodDelete, assignment+azureLighthouseTestAPIVersion, ""), http.StatusOK)
	}
	resp = do(http.MethodGet, "/azure/providers/Microsoft.ManagedServices/marketplaceRegistrationDefinitions"+azureLighthouseTestAPIVersion, "")
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodDelete, definition+azureLighthouseTestAPIVersion, ""), http.StatusOK)
}

func TestAzureLighthouseValidation(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-lh")
	definition := sub + "/providers/Microsoft.ManagedServices/registrationDefinitions/" + azureLighthouseTestDefinition + azureLighthouseTestAPIVersion
	cases := []struct{ name, old, new string }{
		{"tenant", `"managedByTenantId":"2c1f5d3e-8a7b-4c6d-9e0f-1a2b3c4d5e6f"`, `"managedByTenantId":"contoso"`},
		{"own tenant", `"managedByTenantId":"2c1f5d3e-8a7b-4c6d-9e0f-1a2b3c4d5e6f"`, `"managedByTenantId":"` + azureDefaultTenantID + `"`},
		{"authorizations", `"authorizations":[{"principalId":"6f3e2d1c-0b9a-4876-a543-210fedcba987","principalIdDisplayName":"Ops engineers","roleDefinitionId":"acdd72a7-3385-48ef-bd42-f606fba81ae7"}]`, `"authorizations":[]`},
		{"owner", `"roleDefinitionId":"acdd72a7-3385-48ef-bd42-f606fba81ae7"`, `"roleDefinitionId":"` + azureLighthouseOwnerRole + `"`},
		{"role id", `"roleDefinitionId":"acdd72a7-3385-48ef-bd42-f606fba81ae7"`, `"roleDefinitionId":"Reader"`},
		{"delegated roles", `"roleDefinitionId":"acdd72a7-3385-48ef-bd42-f606fba81ae7"`, `"roleDefinitionId":"acdd72a7-3385-48ef-bd42-f606fba81ae7","delegatedRoleDefinitionIds":["b24988ac-6180-42a0-ab88-20f7382dd24c"]`},
		{"mfa", `"multiFactorAuthProvider":"Azure"`, `"multiFactorAuthProvider":"Okta"`},
	}
	for _, tc := range cases {
		body := strings.Replace(azureLighthouseTestBody, tc.old, tc.new, 1)
		if body == azureLighthouseTestBody {
			t.Fatalf("case %s did not change the body", tc.name)
		}
		resp := do(http.MethodPut, definition, body)
		t.Run(tc.name, func(t *testing.T) {
			azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, "InvalidRequestContent")
		})
	}
	assignment := sub + "/providers/Microsoft.ManagedServices/registrationAssignments/0b4c8d2e-5f6a-4b7c-8d9e-0f1a2b3c4d5e" + azureLighthouseTestAPIVersion
	missing := strings.TrimPrefix(sub, "/azure") + "/providers/Microsoft.ManagedServices/registrationDefinitions/" + azureLighthouseTestDefinition
	azureResourcesExpectErrorCode(t, do(http.MethodPut, assignment, `{"properties":{"registrationDefinitionId":"`+missing+`"}}`), http.StatusBadRequest, "InvalidRequestContent")
	azureResourcesExpectErrorCode(t, do(http.MethodPut, assignment, `{"properties":{"registrationDefinitionId":"definition"}}`), http.StatusBadRequest, "InvalidRequestContent")
	azureResourcesExpectErrorCode(t, do(http.MethodPut, sub+"/providers/Microsoft.ManagedServices/registrationDefinitions/contoso"+azureLighthouseTestAPIVersion, azureLighthouseTestBody), http.StatusBadRequest, "InvalidResourceName")
	azureResourcesExpectErrorCode(t, do(http.MethodPut, sub+"/resourceGroups/rg-lh/providers/Microsoft.Compute/virtualMachines/vm/providers/Microsoft.ManagedServices/registrationDefinitions/"+azureLighthouseTestDefinition+azureLighthouseTestAPIVersion, azureLighthouseTestBody), http.StatusBadRequest, "InvalidScope")
}
