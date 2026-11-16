package server

import (
	"net/http"
	"slices"
	"strings"

	"github.com/google/uuid"
)

// Microsoft.Subscription aliases and subscription lifecycle actions, served
// by the shared ARM handler. Every subscription ID already exists in
// Stackyard, so an alias only names one: creating it bills nothing and
// cancel, rename and enable change nothing.
func init() {
	const ns = "Microsoft.Subscription"
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "aliases", Tenant: true,
		Validate:           azureSubscriptionAliasValidate,
		ReadOnlyProperties: azureSubscriptionAliasProperties,
	})
	for _, action := range []string{"cancel", "rename", "enable"} {
		registerAzureARMScopeAction(ns, action, func(_ *Server, scope string, payload map[string]any) (int, any) {
			seg := strings.Split(strings.Trim(scope, "/"), "/")
			if len(seg) != 2 {
				return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "Subscription actions apply only at subscription scope.")
			}
			if action == "rename" && strings.TrimSpace(azureARMString(payload["subscriptionName"])) == "" {
				return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "subscriptionName is required.")
			}
			return http.StatusOK, map[string]any{"subscriptionId": seg[1]}
		})
	}
}

func azureSubscriptionAliasValidate(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	if id, present := props["subscriptionId"]; present {
		if _, err := uuid.Parse(azureARMString(id)); err != nil {
			return "properties.subscriptionId must be a GUID."
		}
		return ""
	}
	// A new subscription needs a name, a workload and where to bill it.
	if strings.TrimSpace(azureARMString(props["displayName"])) == "" {
		return "properties.displayName is required to create a subscription."
	}
	if !slices.Contains([]string{"Production", "DevTest"}, azureARMString(props["workload"])) {
		return "properties.workload must be Production or DevTest."
	}
	if !strings.HasPrefix(strings.ToLower(azureARMString(props["billingScope"])), "/providers/microsoft.billing/billingaccounts/") {
		return "properties.billingScope must be a billing account scope."
	}
	return ""
}

// azureSubscriptionAliasProperties names the subscription the alias stands
// for: the one the client gave, or a stable new one.
func azureSubscriptionAliasProperties(rec *azureARMResource) map[string]any {
	if _, present := rec.Properties["subscriptionId"]; present {
		return nil
	}
	return map[string]any{"subscriptionId": uuid.NewSHA1(uuid.NameSpaceURL, []byte(azureARMKey(rec.ID))).String()}
}
