package server

import (
	"strings"

	"github.com/google/uuid"
)

// Microsoft.GraphServices accounts (api-version 2023-04-13), served by the
// shared ARM handler. An account links an Entra application to an Azure
// billing plan; no Microsoft Graph usage is metered.
func init() {
	const ns = "Microsoft.GraphServices"
	registerAzureARMResourceType(azureARMResourceType{
		Namespace:          ns,
		Type:               "accounts",
		Validate:           azureGsvcValidateAccount,
		ReadOnlyProperties: azureGsvcAccountProperties,
	})
	registerAzureARMView(ns, "operations", azureGsvcOperationsView(ns, "Microsoft Graph Services", "accounts"))
}

func azureGsvcValidateAccount(payload map[string]any) string {
	appID := strings.TrimSpace(azureARMString(azureARMObject(payload["properties"])["appId"]))
	if appID == "" {
		return "properties.appId is required."
	}
	if _, err := uuid.Parse(appID); err != nil {
		return "properties.appId is invalid: it must be a GUID."
	}
	return ""
}

// azureGsvcAccountProperties derives the billing plan from the account ID,
// so it stays the same across updates.
func azureGsvcAccountProperties(rec *azureARMResource) map[string]any {
	return map[string]any{"billingPlanId": uuid.NewSHA1(uuid.NameSpaceURL, []byte(strings.ToLower(rec.ID))).String()}
}

// azureGsvcOperationsView lists read, write and delete operations for each
// resource type of a namespace, for GET /providers/{ns}/operations.
func azureGsvcOperationsView(ns, provider string, types ...string) azureARMView {
	return func(*Server, string) []map[string]any {
		ops := make([]map[string]any, 0, 3*len(types))
		for _, typ := range types {
			for _, verb := range []string{"read", "write", "delete"} {
				ops = append(ops, map[string]any{
					"name": ns + "/" + typ + "/" + verb, "isDataAction": false, "origin": "user,system",
					"display": map[string]any{"provider": provider, "resource": typ, "operation": verb + " " + typ, "description": strings.ToUpper(verb[:1]) + verb[1:] + " " + typ + "."},
				})
			}
		}
		return ops
	}
}
