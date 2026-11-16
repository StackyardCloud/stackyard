package server

import (
	"fmt"
	"net/http"
	"regexp"
)

// Microsoft.Portal shared dashboards and the tenant configuration, served by
// the shared ARM handler. Dashboards are stored as sent; nothing renders
// them, so no tenant configuration violations are ever found.
func init() {
	const ns = "Microsoft.Portal"
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "dashboards",
		ValidateName: func(name string) string {
			if !azurePortalDashboardName.MatchString(name) {
				return "Dashboard names must be 3 to 24 letters, digits or hyphens."
			}
			return ""
		},
		Validate: azurePortalDashboardValidate,
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "tenantConfigurations", Tenant: true, NoPatch: true,
		ValidateName: azureSingletonName("default"),
		Validate: func(payload map[string]any) string {
			raw, present := azureARMObject(payload["properties"])["enforcePrivateMarkdownStorage"]
			if _, ok := raw.(bool); present && !ok {
				return "properties.enforcePrivateMarkdownStorage must be a boolean."
			}
			return ""
		},
	})
	registerAzureARMProviderAction(ns, "listTenantConfigurationViolations", func(*Server, map[string]any) (int, any) {
		return http.StatusOK, map[string]any{"value": []any{}}
	})
}

var azurePortalDashboardName = regexp.MustCompile(`^[a-zA-Z0-9-]{3,24}$`)

// azurePortalDashboardValidate checks each lens's order and parts and each
// part's grid position. Lenses (and parts) may be an array or, as older
// clients send them, an object keyed by index.
func azurePortalDashboardValidate(payload map[string]any) string {
	lenses, ok := azurePortalItems(azureARMObject(payload["properties"])["lenses"])
	if !ok {
		return "properties.lenses must be an array or an object of lenses."
	}
	for i, raw := range lenses {
		lens := azureARMObject(raw)
		if !azureWholeNumber(lens["order"], 0, 1<<31-1) {
			return fmt.Sprintf("lens %d needs an integer order.", i)
		}
		parts, ok := azurePortalItems(lens["parts"])
		if !ok || lens["parts"] == nil {
			return fmt.Sprintf("lens %d needs parts.", i)
		}
		for j, part := range parts {
			position := azureARMObject(azureARMObject(part)["position"])
			for _, key := range []string{"x", "y", "rowSpan", "colSpan"} {
				if !azureWholeNumber(position[key], 0, 1<<31-1) {
					return fmt.Sprintf("lens %d part %d needs an integer position.%s.", i, j, key)
				}
			}
		}
	}
	return ""
}

// azurePortalItems returns the items of an array, or the values of an
// object keyed by index; a missing value has none.
func azurePortalItems(raw any) ([]any, bool) {
	switch items := raw.(type) {
	case nil:
		return nil, true
	case []any:
		return items, true
	case map[string]any:
		out := make([]any, 0, len(items))
		for _, item := range items {
			out = append(out, item)
		}
		return out, true
	}
	return nil, false
}
