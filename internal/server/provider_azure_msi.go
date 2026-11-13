package server

import "github.com/google/uuid"

// Microsoft.ManagedIdentity user-assigned identities, served by the shared ARM handler.
func init() {
	registerAzureARMResourceType(azureARMResourceType{
		Namespace:          "Microsoft.ManagedIdentity",
		Type:               "userAssignedIdentities",
		ReadOnlyProperties: azureMSIIdentityProperties,
	})
}

// azureMSIIdentityProperties derives stable IDs from the resource ID so an
// identity keeps its principal across updates and server restarts.
func azureMSIIdentityProperties(rec *azureARMResource) map[string]any {
	key := []byte(azureARMKey(rec.ID))
	return map[string]any{
		"tenantId":    azureDefaultTenantID,
		"principalId": uuid.NewSHA1(uuid.NameSpaceURL, append(key, "#principal"...)).String(),
		"clientId":    uuid.NewSHA1(uuid.NameSpaceURL, append(key, "#client"...)).String(),
	}
}
