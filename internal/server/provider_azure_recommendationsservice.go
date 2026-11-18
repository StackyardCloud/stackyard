package server

import (
	"net/http"
	"strings"
	"time"
)

// Microsoft.RecommendationsService: Intelligent Recommendations accounts
// with their modeling and service endpoint resources. No models are
// trained: an account's status reports each modeling resource's stages
// from whether it has input data, and endpoints serve nothing.
const azureRecommendationsNamespace = "Microsoft.RecommendationsService"

const azureRecommendationsAccountType = azureRecommendationsNamespace + "/accounts"

// azureRecommendationsPairs maps a region (lower-cased, without spaces) to
// its paired region, from the Azure region pairs table on Learn.
var azureRecommendationsPairs = map[string]string{
	"australiacentral": "Australia Central 2", "australiaeast": "Australia Southeast", "australiasoutheast": "Australia East",
	"brazilsouth": "South Central US", "canadacentral": "Canada East", "canadaeast": "Canada Central",
	"centralindia": "South India", "centralus": "East US 2", "eastasia": "Southeast Asia", "eastus": "West US",
	"eastus2": "Central US", "francecentral": "France South", "germanywestcentral": "Germany North",
	"indiasouthcentral": "Central India", "japaneast": "Japan West", "japanwest": "Japan East",
	"koreacentral": "Korea South", "koreasouth": "Korea Central", "northcentralus": "South Central US",
	"northeurope": "West Europe", "norwayeast": "Norway West", "southafricanorth": "South Africa West",
	"southcentralus": "North Central US", "southeastasia": "East Asia", "southindia": "Central India",
	"swedencentral": "Sweden South", "switzerlandnorth": "Switzerland West", "uaenorth": "UAE Central",
	"uksouth": "UK West", "ukwest": "UK South", "westcentralus": "West US 2", "westeurope": "North Europe",
	"westindia": "South India", "westus": "East US", "westus2": "West Central US", "westus3": "East US",
	// The other halves of pairs whose second region is restricted.
	"australiacentral2": "Australia Central", "francesouth": "France Central", "germanynorth": "Germany West Central",
	"norwaywest": "Norway East", "southafricawest": "South Africa North", "swedensouth": "Sweden Central",
	"switzerlandwest": "Switzerland North", "uaecentral": "UAE North",
}

func init() {
	ns := azureRecommendationsNamespace
	succeeded := func(*azureARMResource) map[string]any { return map[string]any{"provisioningState": "Succeeded"} }
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "accounts",
		Validate:           azureRecommendationsAccountValidate,
		PatchBody:          azureRecommendationsAccountPatch,
		ReadOnlyProperties: succeeded,
		Reconcile:          azureRecommendationsAccountReconcile,
		ServerActions: map[string]func(*Server, *azureARMResource, map[string]any) (int, any){
			"status": azureRecommendationsStatus,
		},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "accounts/modeling",
		Validate:           azureRecommendationsModelingValidate,
		PatchBody:          azureRecommendationsModelingPatch,
		ReadOnlyProperties: succeeded,
		Reconcile:          azureRecommendationsFixed("features"),
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "accounts/serviceEndpoints",
		Validate: azureRecommendationsEndpointValidate,
		// ServiceEndpointPatchResource has tags only.
		PatchBody: func(payload map[string]any) (map[string]any, map[string]any, string) {
			if _, present := payload["properties"]; present {
				return nil, nil, "Only tags can be updated."
			}
			return payload, nil, ""
		},
		ReadOnlyProperties: azureRecommendationsEndpointProperties,
	})
	registerAzureARMScopeAction(ns, "checkNameAvailability", azureRecommendationsCheckName)
}

// azureRecommendationsFixed rejects a change to properties set only at
// creation; omitting one keeps the stored value.
func azureRecommendationsFixed(keys ...string) func(*Server, *azureARMResource, *azureARMResource) (map[string]any, string) {
	return func(_ *Server, existing, rec *azureARMResource) (map[string]any, string) {
		out := map[string]any{}
		if existing == nil {
			return out, ""
		}
		for _, key := range keys {
			stored, was := existing.Properties[key]
			sent, is := rec.Properties[key]
			switch {
			case is && (!was || azureARMString(sent) != azureARMString(stored)):
				return nil, "properties." + key + " can only be set at creation."
			case was:
				out[key] = stored
			}
		}
		return out, ""
	}
}

// azureRecommendationsAccountReconcile keeps account names unique across
// subscriptions, since they name the endpoints' hosts, and keeps the
// configuration set at creation.
func azureRecommendationsAccountReconcile(s *Server, existing, rec *azureARMResource) (map[string]any, string) {
	if existing == nil && len(s.azureARMResourcesMatchingLocked(func(other *azureARMResource) bool {
		return strings.EqualFold(other.Type, azureRecommendationsAccountType) && strings.EqualFold(other.Name, rec.Name)
	})) > 0 {
		return nil, "An Intelligent Recommendations account named '" + rec.Name + "' already exists."
	}
	return azureRecommendationsFixed("configuration")(s, existing, rec)
}

func azureRecommendationsCheckName(s *Server, scope string, payload map[string]any) (int, any) {
	if _, ok := azureARMSubscriptionScope(scope); !ok {
		return http.StatusBadRequest, azureARMErrorBody("InvalidScope", "Name availability is checked at subscription scope.")
	}
	if typ := azureARMString(payload["type"]); !strings.EqualFold(typ, azureRecommendationsAccountType) {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "type must be "+azureRecommendationsAccountType+".")
	}
	name := azureARMString(payload["name"])
	if strings.TrimSpace(name) == "" {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "name is required.")
	}
	if len(s.azureARMResourcesWhere(func(rec *azureARMResource) bool {
		return strings.EqualFold(rec.Type, azureRecommendationsAccountType) && strings.EqualFold(rec.Name, name)
	})) > 0 {
		return http.StatusOK, map[string]any{"nameAvailable": false, "reason": "AlreadyExists", "message": "Requested name is already in use. Please select a different name."}
	}
	return http.StatusOK, map[string]any{"nameAvailable": true}
}

// azureRecommendationsStatus reports each modeling resource's stages: with
// an input data connection string the data is read, validated and
// modeled; without one, data access fails and the later stages wait.
func azureRecommendationsStatus(s *Server, rec *azureARMResource, _ map[string]any) (int, any) {
	modeling, _ := lookupAzureARMResourceType(azureRecommendationsNamespace, "accounts/modeling")
	s.providerStorageMu.Lock()
	children := s.azureARMInlineChildrenLocked(rec.ID, modeling)
	s.providerStorageMu.Unlock()
	if len(children) == 0 {
		return http.StatusOK, map[string]any{}
	}
	var scopes []any
	for _, child := range children {
		results := []string{"Success", "Success", "Success"}
		if azureARMString(azureARMObject(child.Properties["inputData"])["connectionString"]) == "" {
			results = []string{"Failed", "Pending", "Pending"}
		}
		var statuses []any
		for i, stage := range []string{"DataAccess", "DataValidation", "Model"} {
			statuses = append(statuses, map[string]any{
				"stage": stage, "status": results[i],
				"time": child.UpdatedAt.Add(time.Duration(i) * time.Minute).Format(time.RFC3339Nano),
			})
		}
		scopes = append(scopes, map[string]any{"scope": child.Name, "statuses": statuses})
	}
	return http.StatusOK, map[string]any{"scopesStatuses": scopes}
}

// azureRecommendationsEndpointProperties derives an endpoint's host and
// paired region; regions without a pair have no pairedLocation.
func azureRecommendationsEndpointProperties(rec *azureARMResource) map[string]any {
	parts := strings.Split(rec.ID, "/")
	account := parts[len(parts)-3]
	out := map[string]any{
		"provisioningState": "Succeeded",
		"url":               strings.ToLower(account+"-"+rec.Name) + ".recommendations.stackyard.invalid",
	}
	if pair, ok := azureRecommendationsPairs[strings.ToLower(strings.ReplaceAll(rec.Location, " ", ""))]; ok {
		out["pairedLocation"] = pair
	}
	return out
}

func azureRecommendationsAccountValidate(payload map[string]any) string {
	if msg := azureDataBoxObjects(payload, "", "properties"); msg != "" {
		return msg
	}
	return azureRecommendationsAccount(azureARMObject(payload["properties"]))
}

func azureRecommendationsAccountPatch(payload map[string]any) (map[string]any, map[string]any, string) {
	if msg := azureDataBoxObjects(payload, "", "properties"); msg != "" {
		return nil, nil, msg
	}
	return payload, nil, azureRecommendationsAccount(azureARMObject(payload["properties"]))
}

func azureRecommendationsAccount(props map[string]any) string {
	if msg := azureLabEnum(props, "properties.", "configuration", []string{"Free", "Capacity"}); msg != "" {
		return msg
	}
	if raw, present := props["reportsConnectionString"]; present {
		if _, ok := raw.(string); !ok {
			return "properties.reportsConnectionString must be a string."
		}
	}
	if msg := azureRecommendationsItems(props, "endpointAuthentications", func(item map[string]any) string {
		for _, key := range []string{"aadTenantID", "principalID"} {
			if raw, present := item[key]; present {
				if _, ok := raw.(string); !ok {
					return "properties.endpointAuthentications[]." + key + " must be a string."
				}
			}
		}
		return azureLabEnum(item, "properties.endpointAuthentications[].", "principalType", []string{"Application", "User"})
	}); msg != "" {
		return msg
	}
	return azureRecommendationsItems(props, "cors", func(item map[string]any) string {
		for _, key := range []string{"allowedOrigins", "allowedMethods", "allowedHeaders", "exposedHeaders"} {
			if msg := azureImpactStrings(item, "properties.cors[].", key); msg != "" {
				return msg
			}
		}
		if raw, present := item["maxAgeInSeconds"]; present && !azureWholeNumber(raw, 0, 1<<31-1) {
			return "properties.cors[].maxAgeInSeconds must be a whole number of at least 0."
		}
		return ""
	})
}

// azureRecommendationsItems checks that props[key], when present, is an
// array of objects each passing check.
func azureRecommendationsItems(props map[string]any, key string, check func(map[string]any) string) string {
	raw, present := props[key]
	if !present {
		return ""
	}
	list, ok := raw.([]any)
	if !ok {
		return "properties." + key + " must be an array."
	}
	for _, rawItem := range list {
		item, ok := rawItem.(map[string]any)
		if !ok {
			return "properties." + key + " items must be objects."
		}
		if msg := check(item); msg != "" {
			return msg
		}
	}
	return ""
}

func azureRecommendationsModelingValidate(payload map[string]any) string {
	if msg := azureDataBoxObjects(payload, "", "properties"); msg != "" {
		return msg
	}
	return azureRecommendationsModeling(azureARMObject(payload["properties"]))
}

func azureRecommendationsModelingPatch(payload map[string]any) (map[string]any, map[string]any, string) {
	if msg := azureDataBoxObjects(payload, "", "properties"); msg != "" {
		return nil, nil, msg
	}
	return payload, nil, azureRecommendationsModeling(azureARMObject(payload["properties"]))
}

func azureRecommendationsModeling(props map[string]any) string {
	for key, values := range map[string][]string{
		"features": {"Basic", "Standard", "Premium"}, "frequency": {"Low", "Medium", "High"}, "size": {"Small", "Medium", "Large"},
	} {
		if msg := azureLabEnum(props, "properties.", key, values); msg != "" {
			return msg
		}
	}
	if msg := azureDataBoxObjects(props, "properties.", "inputData"); msg != "" {
		return msg
	}
	if raw, present := azureARMObject(props["inputData"])["connectionString"]; present {
		if _, ok := raw.(string); !ok {
			return "properties.inputData.connectionString must be a string."
		}
	}
	return ""
}

func azureRecommendationsEndpointValidate(payload map[string]any) string {
	if msg := azureDataBoxObjects(payload, "", "properties"); msg != "" {
		return msg
	}
	props := azureARMObject(payload["properties"])
	if raw, present := props["preAllocatedCapacity"]; present && !azureWholeNumber(raw, 0, 1<<31-1) {
		return "properties.preAllocatedCapacity must be a whole number of at least 0."
	}
	return ""
}
