package server

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestAzureAuthorizationLifecycle(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	headers := map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
		"Content-Type":              "application/json",
	}

	subscriptionID := "00000000-0000-0000-0000-000000000000"
	resourceGroup := "rg-authz"
	resourceProvider := "Microsoft.Storage"
	resourceType := "storageAccounts"
	resourceName := "saauth"
	subscriptionScope := "/subscriptions/" + subscriptionID
	resourceGroupScope := subscriptionScope + "/resourceGroups/" + resourceGroup
	resourceScope := resourceGroupScope + "/providers/" + resourceProvider + "/" + resourceType + "/" + resourceName

	roleDefinitionName := "rd-storage-reader"
	roleDefinitionPath := "/azure" + subscriptionScope + "/providers/Microsoft.Authorization/roleDefinitions/" + roleDefinitionName + "?api-version=2022-04-01"
	resp := providerContractRequest(t, ts, http.MethodPut, roleDefinitionPath, []byte(`{
		"properties":{
			"roleName":"Storage Reader",
			"description":"Read access to one storage account",
			"type":"CustomRole",
			"permissions":[{"actions":["Microsoft.Storage/storageAccounts/read"],"notActions":[]}],
			"assignableScopes":["/subscriptions/00000000-0000-0000-0000-000000000000","/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-authz"]
		}
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating role definition, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	roleDefinition := providerContractJSONMap(t, resp)
	if roleDefinition["type"] != "Microsoft.Authorization/roleDefinitions" {
		t.Fatalf("unexpected role definition payload: %#v", roleDefinition)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, roleDefinitionPath, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting role definition, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, "/azure"+subscriptionScope+"/providers/Microsoft.Authorization/roleDefinitions?api-version=2022-04-01&$filter=roleName%20eq%20'S%74orage%20Reader'", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing role definitions, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	var roleDefinitionList struct {
		Value []map[string]any `json:"value"`
	}
	if err := json.Unmarshal(providerContractBody(t, resp), &roleDefinitionList); err != nil {
		t.Fatalf("unmarshal role definition list: %v", err)
	}
	if len(roleDefinitionList.Value) != 1 {
		t.Fatalf("expected one role definition, got %#v", roleDefinitionList.Value)
	}

	roleDefinitionID := subscriptionScope + "/providers/Microsoft.Authorization/roleDefinitions/" + roleDefinitionName

	roleAssignmentName := "ra-storage-reader"
	roleAssignmentPath := "/azure" + resourceScope + "/providers/Microsoft.Authorization/roleAssignments/" + roleAssignmentName + "?api-version=2022-04-01"
	resp = providerContractRequest(t, ts, http.MethodPut, roleAssignmentPath, []byte(`{
		"properties":{
			"principalId":"11111111-1111-1111-1111-111111111111",
			"roleDefinitionId":"`+roleDefinitionID+`"
		}
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating role assignment, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, roleAssignmentPath, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting role assignment, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	roleAssignmentByIDName := "ra-subscription-reader"
	roleAssignmentByIDPath := "/azure" + subscriptionScope + "/providers/Microsoft.Authorization/roleAssignments/" + roleAssignmentByIDName + "?api-version=2022-04-01"
	resp = providerContractRequest(t, ts, http.MethodPut, roleAssignmentByIDPath, []byte(`{
		"properties":{
			"principalId":"22222222-2222-2222-2222-222222222222",
			"roleDefinitionId":"`+roleDefinitionID+`"
		}
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating role assignment by id path, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, "/azure"+resourceScope+"/providers/Microsoft.Authorization/roleAssignments?api-version=2022-04-01", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing role assignments for resource, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, "/azure"+resourceGroupScope+"/providers/Microsoft.Authorization/roleAssignments?api-version=2022-04-01&$filter=principalId%20eq%20'11111111-1111-1111-1111-111111111111'", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing role assignments for resource group, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, "/azure"+subscriptionScope+"/providers/Microsoft.Authorization/roleAssignments?api-version=2022-04-01", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing role assignments for subscription, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, "/azure"+resourceScope+"/providers/Microsoft.Authorization/permissions?api-version=2022-04-01", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing permissions for resource, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, "/azure"+resourceGroupScope+"/providers/Microsoft.Authorization/permissions?api-version=2022-04-01", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing permissions for resource group, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodDelete, roleAssignmentByIDPath, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 deleting role assignment by id path, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodDelete, roleAssignmentPath, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 deleting role assignment, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodDelete, roleDefinitionPath, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 deleting role definition, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
}

func TestAzureAuthorizationValidationAndNotFound(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	headers := map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
		"Content-Type":              "application/json",
	}

	subscriptionScope := "/subscriptions/00000000-0000-0000-0000-000000000000"
	roleDefinitionPath := "/azure" + subscriptionScope + "/providers/Microsoft.Authorization/roleDefinitions/rd-invalid?api-version=2022-04-01"
	roleAssignmentPath := "/azure" + subscriptionScope + "/providers/Microsoft.Authorization/roleAssignments/ra-invalid?api-version=2022-04-01"

	resp := providerContractRequest(t, ts, http.MethodPut, roleDefinitionPath, []byte(`{"properties":{"permissions":[{}],"assignableScopes":["/subscriptions/00000000-0000-0000-0000-000000000000"]}}`), headers)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing roleName, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPut, roleAssignmentPath, []byte(`{"properties":{"roleDefinitionId":"rd"}}`), headers)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing principalId, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPut, roleAssignmentPath, []byte(`{"properties":`), headers)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid JSON, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, roleAssignmentPath, nil, headers)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 getting missing role assignment, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, roleDefinitionPath, nil, headers)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 getting missing role definition, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodDelete, roleAssignmentPath, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 deleting missing role assignment, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodDelete, roleDefinitionPath, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 deleting missing role definition, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
}

func TestAzureAuthorizationInvalidAPIVersionReturnsBadRequest(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/azure/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Authorization/roleAssignments?api-version=", nil, map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid api-version, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
}
