package server

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// Microsoft.ManagedServices (Azure Lighthouse) registration definitions and
// assignments at subscription and resource group scope, served by the
// shared ARM handler. No access is delegated: an assignment only records
// that the managing tenant would get the definition's roles. Marketplace
// offers are never published, so that list is always empty.
func init() {
	const ns = "Microsoft.ManagedServices"
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "registrationDefinitions", Extension: true,
		ValidateScope:      azureLighthouseScope,
		ValidateName:       azureLighthouseName,
		Validate:           azureLighthouseDefinitionValidate,
		ReadOnlyProperties: azureLighthouseDefinitionProperties,
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "registrationAssignments", Extension: true,
		ValidateScope: azureLighthouseScope,
		ValidateName:  azureLighthouseName,
		Validate: func(payload map[string]any) string {
			id := strings.ToLower(azureARMString(azureARMObject(payload["properties"])["registrationDefinitionId"]))
			if !strings.Contains(id, "/providers/microsoft.managedservices/registrationdefinitions/") {
				return "properties.registrationDefinitionId must be a registration definition ID."
			}
			return ""
		},
		Linked: azureLighthouseAssignmentLinked,
	})
	registerAzureARMView(ns, "marketplaceRegistrationDefinitions", func(*Server, string) []map[string]any { return nil })
}

const (
	azureLighthouseOwnerRole                   = "8e3af657-a8ff-443c-a75c-2fe8c4bcb635"
	azureLighthouseUserAccessAdministratorRole = "18d7d88d-d35e-4fb5-a5c3-7773c20a72d9"
)

// azureLighthouseScope accepts a subscription or resource group.
func azureLighthouseScope(scope string) string {
	seg := strings.Split(strings.Trim(scope, "/"), "/")
	if (len(seg) == 2 || len(seg) == 4 && strings.EqualFold(seg[2], "resourceGroups")) && strings.EqualFold(seg[0], "subscriptions") {
		return ""
	}
	return "Lighthouse delegations apply to a subscription or resource group."
}

func azureLighthouseName(name string) string {
	if _, err := uuid.Parse(name); err != nil {
		return "Registration names must be GUIDs."
	}
	return ""
}

func azureLighthouseDefinitionValidate(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	tenant := azureARMString(props["managedByTenantId"])
	if _, err := uuid.Parse(tenant); err != nil {
		return "properties.managedByTenantId must be a tenant GUID."
	}
	if strings.EqualFold(tenant, azureDefaultTenantID) {
		return "properties.managedByTenantId must be a different tenant than the customer's."
	}
	authorizations := azureARMSlice(props["authorizations"])
	if len(authorizations) == 0 {
		return "properties.authorizations must list at least one authorization."
	}
	for i, raw := range authorizations {
		if msg := azureLighthouseAuthorizationValidate(azureARMObject(raw), fmt.Sprintf("properties.authorizations[%d]", i)); msg != "" {
			return msg
		}
	}
	for i, raw := range azureARMSlice(props["eligibleAuthorizations"]) {
		at := fmt.Sprintf("properties.eligibleAuthorizations[%d]", i)
		auth := azureARMObject(raw)
		if msg := azureLighthouseAuthorizationValidate(auth, at); msg != "" {
			return msg
		}
		policy, present := auth["justInTimeAccessPolicy"]
		if provider := azureARMObject(policy)["multiFactorAuthProvider"]; present && provider != "Azure" && provider != "None" {
			return at + ".justInTimeAccessPolicy.multiFactorAuthProvider must be Azure or None."
		}
	}
	return ""
}

// azureLighthouseAuthorizationValidate checks one authorization: a principal
// and a role by GUID, never Owner, and delegated roles only for User Access
// Administrator.
func azureLighthouseAuthorizationValidate(auth map[string]any, at string) string {
	role := azureARMString(auth["roleDefinitionId"])
	for key, value := range map[string]string{"principalId": azureARMString(auth["principalId"]), "roleDefinitionId": role} {
		if _, err := uuid.Parse(value); err != nil {
			return at + "." + key + " must be a GUID."
		}
	}
	if strings.EqualFold(role, azureLighthouseOwnerRole) {
		return at + ": the Owner role can't be delegated with Azure Lighthouse."
	}
	if delegated := azureARMSlice(auth["delegatedRoleDefinitionIds"]); len(delegated) > 0 && !strings.EqualFold(role, azureLighthouseUserAccessAdministratorRole) {
		return at + ".delegatedRoleDefinitionIds is only allowed with the User Access Administrator role."
	}
	return ""
}

func azureLighthouseDefinitionProperties(*azureARMResource) map[string]any {
	return map[string]any{"manageeTenantId": azureDefaultTenantID, "manageeTenantName": "Stackyard"}
}

// azureLighthouseAssignmentLinked requires the assigned definition to exist.
func azureLighthouseAssignmentLinked(s *Server, rec *azureARMResource) (map[string]any, []*azureARMResource, string) {
	id := azureARMString(rec.Properties["registrationDefinitionId"])
	if s.azureARMResources[azureARMKey(id)] == nil {
		return nil, nil, "Registration definition " + id + " was not found."
	}
	return nil, nil, ""
}
