package server

import (
	"net/http"
	"regexp"
	"slices"
	"strings"

	"github.com/google/uuid"
)

// Microsoft.PowerBIDedicated: Power BI Embedded capacities, which suspend
// and resume, and the auto scale v-cores that add capacity to them. No
// reports run: a capacity is only its SKU, administrators and state.
const azurePowerBINamespace = "Microsoft.PowerBIDedicated"

var (
	azurePowerBICapacityName = regexp.MustCompile(`^[a-z][a-z0-9]{2,62}$`)
	azurePowerBIVCoreName    = regexp.MustCompile(`^[-a-zA-Z0-9_]{3,63}$`)
	// ponytail: the A SKUs listSkus returns; EM and P SKUs, bought in
	// Microsoft 365, aren't offered here.
	azurePowerBISKUs = []string{"A1", "A2", "A3", "A4", "A5", "A6"}
)

func init() {
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: azurePowerBINamespace, Type: "capacities",
		ValidateName: func(name string) string {
			if !azurePowerBICapacityName.MatchString(name) {
				return "Capacity names are 3 to 63 lowercase letters and digits, starting with a letter."
			}
			return ""
		},
		Validate: func(payload map[string]any) string {
			if msg := azurePowerBISKUValidate(payload["sku"], true); msg != "" {
				return msg
			}
			if msg := azureDataBoxObjects(payload, "", "properties"); msg != "" {
				return msg
			}
			return azurePowerBICapacityPropertiesValidate(azureARMObject(payload["properties"]))
		},
		ReadOnlySKU: func(map[string]any) map[string]any { return map[string]any{"tier": "PBIE_Azure"} },
		ReadOnlyProperties: func(rec *azureARMResource) map[string]any {
			if rec.Properties["mode"] == nil {
				return map[string]any{"mode": "Gen2"}
			}
			return nil
		},
		// Only suspend and resume change the state, which provisioningState
		// mirrors.
		StateProperties: map[string]any{"state": "Succeeded", "provisioningState": "Succeeded"},
		PatchBody: func(payload map[string]any) (map[string]any, map[string]any, string) {
			if raw, present := payload["sku"]; present {
				if msg := azurePowerBISKUValidate(raw, true); msg != "" {
					return nil, nil, msg
				}
				payload = azureARMMerge(payload, map[string]any{"sku": map[string]any{"tier": "PBIE_Azure"}})
			}
			if msg := azureDataBoxObjects(payload, "", "properties"); msg != "" {
				return nil, nil, msg
			}
			props := azureARMObject(payload["properties"])
			for key := range props {
				if key != "administration" && key != "mode" {
					return nil, nil, "properties." + key + " can't be updated."
				}
			}
			return payload, nil, azurePowerBICapacityPropertiesValidate(props)
		},
		Transitions: map[string]map[string]any{
			"suspend": {"state": "Paused", "provisioningState": "Paused"},
			"resume":  {"state": "Succeeded", "provisioningState": "Succeeded"},
		},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: azurePowerBINamespace, Type: "autoScaleVCores",
		ReadOnlySKU: func(map[string]any) map[string]any { return map[string]any{"tier": "AutoScale"} },
		ValidateName: func(name string) string {
			if !azurePowerBIVCoreName.MatchString(name) {
				return "Auto scale v-core names are 3 to 63 letters, digits, hyphens or underscores."
			}
			return ""
		},
		Validate: func(payload map[string]any) string {
			if msg := azurePowerBIVCoreSKUValidate(payload["sku"]); msg != "" {
				return msg
			}
			if msg := azureDataBoxObjects(payload, "", "properties"); msg != "" {
				return msg
			}
			props := azureARMObject(payload["properties"])
			if raw, present := props["capacityObjectId"]; present {
				if _, err := uuid.Parse(azureARMString(raw)); err != nil {
					return "properties.capacityObjectId must be the capacity's object ID (a GUID)."
				}
			}
			return azurePowerBILimitValidate(props)
		},
		PatchBody: func(payload map[string]any) (map[string]any, map[string]any, string) {
			if raw, present := payload["sku"]; present {
				if msg := azurePowerBIVCoreSKUValidate(raw); msg != "" {
					return nil, nil, msg
				}
				payload = azureARMMerge(payload, map[string]any{"sku": map[string]any{"tier": "AutoScale"}})
			}
			if msg := azureDataBoxObjects(payload, "", "properties"); msg != "" {
				return nil, nil, msg
			}
			props := azureARMObject(payload["properties"])
			for key := range props {
				if key != "capacityLimit" {
					return nil, nil, "properties." + key + " can't be updated."
				}
			}
			return payload, nil, azurePowerBILimitValidate(props)
		},
	})
	registerAzureARMView(azurePowerBINamespace, "skus", func(*Server, string) []map[string]any {
		items := []map[string]any{}
		for _, name := range azurePowerBISKUs {
			items = append(items, map[string]any{"name": name})
		}
		return items
	})
	registerAzureARMView(azurePowerBINamespace, "capacities/skus", azurePowerBICapacitySKUs)
	registerAzureARMProviderAction(azurePowerBINamespace, "locations/checkNameAvailability", azurePowerBICheckName)
}

func azurePowerBISKUValidate(raw any, required bool) string {
	sku, ok := raw.(map[string]any)
	if !ok {
		if required {
			return "sku must be an object with a name, one of " + strings.Join(azurePowerBISKUs, ", ") + "."
		}
		return ""
	}
	if !slices.Contains(azurePowerBISKUs, azureARMString(sku["name"])) {
		return "sku.name must be one of " + strings.Join(azurePowerBISKUs, ", ") + "."
	}
	if tier, present := sku["tier"]; present && tier != "PBIE_Azure" {
		return "sku.tier must be PBIE_Azure for A SKUs."
	}
	return ""
}

func azurePowerBICapacityPropertiesValidate(props map[string]any) string {
	if raw, present := props["mode"]; present && raw != "Gen1" && raw != "Gen2" {
		return "properties.mode must be Gen1 or Gen2."
	}
	if raw, present := props["administration"]; present {
		members, ok := azureARMObject(raw)["members"].([]any)
		if !ok || len(members) == 0 {
			return "properties.administration.members needs at least one administrator."
		}
		for _, member := range members {
			if strings.TrimSpace(azureARMString(member)) == "" {
				return "properties.administration.members must hold user or service principal identities."
			}
		}
	}
	return ""
}

func azurePowerBIVCoreSKUValidate(raw any) string {
	sku, ok := raw.(map[string]any)
	if !ok || sku["name"] != "AutoScale" {
		return "sku.name must be AutoScale."
	}
	if tier, present := sku["tier"]; present && tier != "AutoScale" {
		return "sku.tier must be AutoScale."
	}
	if raw, present := sku["capacity"]; present && !azureWholeNumber(raw, 0, 2147483647) {
		return "sku.capacity must be a whole number of at least 0."
	}
	return ""
}

func azurePowerBILimitValidate(props map[string]any) string {
	if raw, present := props["capacityLimit"]; present && !azureWholeNumber(raw, 0, 2147483647) {
		return "properties.capacityLimit must be a whole number of at least 0."
	}
	return ""
}

// azurePowerBICapacitySKUs lists the SKUs a capacity can scale to: every A
// SKU but its own.
func azurePowerBICapacitySKUs(s *Server, scope string) []map[string]any {
	s.providerStorageMu.Lock()
	current := ""
	if rec := s.azureARMResources[azureARMKey(scope)]; rec != nil {
		current = azureARMString(rec.SKU["name"])
	}
	s.providerStorageMu.Unlock()
	items := []map[string]any{}
	for _, name := range azurePowerBISKUs {
		if name != current {
			items = append(items, map[string]any{"sku": map[string]any{"name": name, "tier": "PBIE_Azure"}})
		}
	}
	return items
}

// azurePowerBICheckName reports whether a capacity name is valid and not
// taken in any subscription.
func azurePowerBICheckName(s *Server, payload map[string]any) (int, any) {
	name := azureARMString(payload["name"])
	if raw, present := payload["type"]; present && !strings.EqualFold(azureARMString(raw), azurePowerBINamespace+"/capacities") {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "type must be Microsoft.PowerBIDedicated/capacities.")
	}
	if !azurePowerBICapacityName.MatchString(name) {
		return http.StatusOK, map[string]any{"nameAvailable": false, "reason": "Invalid",
			"message": "Capacity names are 3 to 63 lowercase letters and digits, starting with a letter."}
	}
	taken := len(s.azureARMResourcesWhere(func(rec *azureARMResource) bool {
		return strings.EqualFold(rec.Type, azurePowerBINamespace+"/capacities") && strings.EqualFold(rec.Name, name)
	})) > 0
	if taken {
		return http.StatusOK, map[string]any{"nameAvailable": false, "reason": "Unavailable", "message": "Capacity name " + name + " is already in use."}
	}
	return http.StatusOK, map[string]any{"nameAvailable": true}
}
