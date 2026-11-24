package server

import (
	"net/http"
	"regexp"
	"slices"
	"strings"
)

// Microsoft.Fabric capacities (api-version 2023-11-01), served by the shared
// ARM handler. Suspend and resume finish at once; no compute is reserved.
const azureFabNamespace = "Microsoft.Fabric"

var (
	azureFabSKUs        = []string{"F2", "F4", "F8", "F16", "F32", "F64", "F128", "F256", "F512", "F1024", "F2048"}
	azureFabLocations   = []string{"eastus", "eastus2", "westus", "westus2", "westus3", "centralus", "northeurope", "westeurope", "uksouth", "southeastasia", "australiaeast", "japaneast"}
	azureFabNamePattern = regexp.MustCompile(`^[a-z][a-z0-9]{2,62}$`)
)

func init() {
	ns := azureFabNamespace
	registerAzureARMResourceType(azureARMResourceType{
		Namespace:       ns,
		Type:            "capacities",
		ValidateName:    azureFabValidateName,
		Validate:        azureFabValidateCapacity,
		StateProperties: map[string]any{"state": "Active"},
		Transitions: map[string]map[string]any{
			"suspend": {"state": "Paused"},
			"resume":  {"state": "Active"},
		},
	})
	registerAzureARMLocationAction(ns, "checkNameAvailability", azureFabCheckName)
	registerAzureARMView(ns, "skus", func(*Server, string) []map[string]any {
		out := make([]map[string]any, 0, len(azureFabSKUs))
		for _, sku := range azureFabSKUs {
			out = append(out, map[string]any{"name": sku, "resourceType": "Capacities", "locations": azureFabLocations})
		}
		return out
	})
	registerAzureARMView(ns, "capacities/skus", func(*Server, string) []map[string]any {
		out := make([]map[string]any, 0, len(azureFabSKUs))
		for _, sku := range azureFabSKUs {
			out = append(out, map[string]any{"resourceType": "Microsoft.Fabric/capacities", "sku": map[string]any{"name": sku, "tier": "Fabric"}})
		}
		return out
	})
	registerAzureARMView(ns, "operations", azureGsvcOperationsView(ns, "Microsoft Fabric", "capacities"))
}

func azureFabValidateName(name string) string {
	if !azureFabNamePattern.MatchString(name) {
		return "The capacity name is invalid: it must be 3 to 63 lowercase letters and digits, starting with a letter."
	}
	return ""
}

func azureFabValidateCapacity(payload map[string]any) string {
	sku := azureARMObject(payload["sku"])
	if !slices.Contains(azureFabSKUs, azureARMString(sku["name"])) {
		return "sku.name is missing or invalid: it must be one of " + strings.Join(azureFabSKUs, ", ") + "."
	}
	if azureARMString(sku["tier"]) != "Fabric" {
		return "sku.tier must be Fabric."
	}
	members := azureARMSlice(azureARMObject(azureARMObject(payload["properties"])["administration"])["members"])
	if len(members) == 0 {
		return "properties.administration.members is missing: a capacity needs at least one administrator."
	}
	for _, m := range members {
		if strings.TrimSpace(azureARMString(m)) == "" {
			return "properties.administration.members must hold non-empty user principal names."
		}
	}
	return ""
}

func azureFabCheckName(s *Server, _, _ string, payload map[string]any) (int, any) {
	name := strings.TrimSpace(azureARMString(payload["name"]))
	if name == "" || !strings.EqualFold(azureARMString(payload["type"]), "Microsoft.Fabric/capacities") {
		return azureOracleBad("name and type 'Microsoft.Fabric/capacities' are required.")
	}
	if msg := azureFabValidateName(name); msg != "" {
		return http.StatusOK, map[string]any{"nameAvailable": false, "reason": "Invalid", "message": msg}
	}
	if len(s.azureARMResourcesWhere(func(rec *azureARMResource) bool {
		return strings.EqualFold(rec.Type, "Microsoft.Fabric/capacities") && strings.EqualFold(rec.Name, name)
	})) > 0 {
		return http.StatusOK, map[string]any{"nameAvailable": false, "reason": "AlreadyExists", "message": "The capacity name " + name + " is already in use."}
	}
	return http.StatusOK, map[string]any{"nameAvailable": true}
}
