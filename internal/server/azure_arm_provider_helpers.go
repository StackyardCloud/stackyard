package server

import "strings"

// Registration helpers for the provider-wide pieces many ARM services share:
// the operations list and a resource's private link surface.

// registerAzureARMOperations serves GET /providers/{ns}/operations with
// read/write/delete operations for each [type path, display name] pair.
func registerAzureARMOperations(ns, provider string, resources [][2]string) {
	registerAzureARMView(ns, "operations", func(*Server, string) []map[string]any {
		ops := make([]map[string]any, 0, len(resources)*3)
		for _, res := range resources {
			for _, verb := range []string{"read", "write", "delete"} {
				what := strings.ToUpper(verb[:1]) + verb[1:] + " " + res[1]
				ops = append(ops, map[string]any{
					"name": ns + "/" + res[0] + "/" + verb, "isDataAction": false, "origin": "user,system",
					"display": map[string]any{"provider": provider, "resource": res[1], "operation": what, "description": what + "."},
				})
			}
		}
		return ops
	})
}

// registerAzureARMPrivateLink registers {parent}/privateEndpointConnections
// as a child type and {parent}/privateLinkResources as a view holding one
// resource for groupID, whose DNS zone is zoneName.
func registerAzureARMPrivateLink(ns, parent, groupID, zoneName string) {
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: parent + "/privateEndpointConnections", Proxy: true,
		Validate: azureFileShareConnectionValidate,
		ReadOnlyProperties: func(*azureARMResource) map[string]any {
			return map[string]any{"groupIds": []any{groupID}}
		},
	})
	registerAzureARMView(ns, parent+"/privateLinkResources", func(_ *Server, scope string) []map[string]any {
		return []map[string]any{{
			"id": scope + "/privateLinkResources/" + groupID, "name": groupID, "type": ns + "/" + parent + "/privateLinkResources",
			"properties": map[string]any{"groupId": groupID, "requiredMembers": []any{groupID}, "requiredZoneNames": []any{zoneName}},
		}}
	})
}
