package server

import (
	"net/http"
	"slices"
	"strings"
)

const azureDeviceRegistry20260401APIVersion = "2026-04-01"

const adrNamespace = "Microsoft.DeviceRegistry"

// Microsoft.DeviceRegistry (Azure Device Registry for Azure IoT Operations),
// served by the shared ARM handler. No IoT Operations instance runs: asset
// management actions complete at once, and namespace migration only checks
// that the listed assets exist.
func init() {
	ns := adrNamespace
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "assetEndpointProfiles",
		Validate: adrRequire(true, "endpointProfileType", "targetAddress"),
		ReadOnlyProperties: func(rec *azureARMResource) map[string]any {
			return map[string]any{"uuid": adrUUID(rec), "status": map[string]any{"errors": []any{}}}
		},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "assets",
		Validate:           adrRequire(true, "assetEndpointProfileRef"),
		ReadOnlyProperties: adrAssetStatus,
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "namespaces",
		ReadOnlyProperties: func(rec *azureARMResource) map[string]any {
			return map[string]any{"uuid": adrUUID(rec)}
		},
		ServerActions: map[string]func(*Server, *azureARMResource, map[string]any) (int, any){"migrate": adrMigrate},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "namespaces/assets",
		Validate:           adrRequire(true, "deviceRef"),
		ReadOnlyProperties: adrAssetStatus,
		Actions:            map[string]azureARMAction{"executeaction": adrExecuteAction},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "namespaces/devices",
		ReadOnlyProperties: func(rec *azureARMResource) map[string]any {
			return map[string]any{"uuid": adrUUID(rec), "etag": rec.etag(), "lastTransitionTime": rec.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z")}
		},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "namespaces/discoveredAssets",
		Validate: adrRequire(true, "deviceRef", "discoveryId", "version"),
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "namespaces/discoveredDevices",
		Validate: adrRequire(true, "discoveryId", "version"),
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "schemaRegistries",
		Validate: adrRequire(false, "namespace", "storageAccountContainerUrl"),
		ReadOnlyProperties: func(rec *azureARMResource) map[string]any {
			return map[string]any{"uuid": adrUUID(rec)}
		},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "schemaRegistries/schemas", Proxy: true,
		Validate: adrValidateSchema,
		ReadOnlyProperties: func(rec *azureARMResource) map[string]any {
			return map[string]any{"uuid": adrUUID(rec)}
		},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "schemaRegistries/schemas/schemaVersions", Proxy: true, NoPatch: true,
		Validate: func(payload map[string]any) string {
			return azureARMRequireProperties(azureARMObject(payload["properties"]), "schemaContent")
		},
		ReadOnlyProperties: func(rec *azureARMResource) map[string]any {
			return map[string]any{"uuid": adrUUID(rec), "hash": azureARMDerivedKey(rec.ID, azureARMString(rec.Properties["schemaContent"]))}
		},
	})
	registerAzureARMView(ns, "billingContainers", func(_ *Server, scope string) []map[string]any {
		return []map[string]any{{
			"id": scope + "/providers/" + ns + "/billingContainers/default", "name": "default", "type": ns + "/billingContainers",
			"properties": map[string]any{"provisioningState": "Succeeded"},
		}}
	})
	registerAzureARMView(ns, "operations", azureARMCrudOperations(ns, "Microsoft Device Registry",
		"assetEndpointProfiles", "assets", "namespaces", "namespaces/assets", "namespaces/devices", "schemaRegistries"))
}

func adrUUID(rec *azureARMResource) string {
	key := azureARMDerivedKey(rec.ID, "uuid")
	return key[0:8] + "-" + key[8:12] + "-" + key[12:16] + "-" + key[16:20] + "-" + key[20:32]
}

// adrRequire checks required properties and, for edge resources, the custom
// location they run in.
func adrRequire(extendedLocation bool, keys ...string) func(map[string]any) string {
	return func(payload map[string]any) string {
		if extendedLocation {
			loc := azureARMObject(payload["extendedLocation"])
			if strings.TrimSpace(azureARMString(loc["name"])) == "" || strings.TrimSpace(azureARMString(loc["type"])) == "" {
				return "extendedLocation with name and type is required."
			}
		}
		return azureARMRequireProperties(azureARMObject(payload["properties"]), keys...)
	}
}

func adrAssetStatus(rec *azureARMResource) map[string]any {
	return map[string]any{"uuid": adrUUID(rec), "version": 1, "status": map[string]any{"errors": []any{}, "version": 1}}
}

func adrValidateSchema(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	if !slices.Contains([]string{"JsonSchema/draft-07", "Delta/1.0"}, azureARMString(props["format"])) {
		return "properties.format must be JsonSchema/draft-07 or Delta/1.0."
	}
	if azureARMString(props["schemaType"]) != "MessageSchema" {
		return "properties.schemaType must be MessageSchema."
	}
	return ""
}

func adrExecuteAction(rec *azureARMResource, payload map[string]any) (int, any) {
	group, action := azureARMString(payload["managementGroupName"]), azureARMString(payload["managementActionName"])
	if group == "" || action == "" {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "managementGroupName and managementActionName are required.")
	}
	groups, _ := rec.Properties["managementGroups"].([]any)
	found := slices.ContainsFunc(groups, func(g any) bool {
		grp := azureARMObject(g)
		actions, _ := grp["actions"].([]any)
		return azureARMString(grp["name"]) == group && slices.ContainsFunc(actions, func(a any) bool { return azureARMString(azureARMObject(a)["name"]) == action })
	})
	if !found {
		return http.StatusNotFound, azureARMErrorBody("NotFound", "management action "+group+"/"+action+" is not defined on the asset.")
	}
	return http.StatusOK, map[string]any{"status": "Succeeded", "statusCode": 200, "payload": map[string]any{}}
}

// adrMigrate checks that each listed asset exists; nothing else moves.
func adrMigrate(s *Server, rec *azureARMResource, payload map[string]any) (int, any) {
	if azureARMString(payload["scope"]) != "Resources" {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "scope must be Resources.")
	}
	ids, _ := payload["resourceIds"].([]any)
	if len(ids) == 0 || len(ids) > 1000 {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "resourceIds must list 1 to 1000 assets.")
	}
	for _, id := range ids {
		if len(s.azureARMResourcesWhere(func(r *azureARMResource) bool { return strings.EqualFold(r.ID, azureARMString(id)) })) == 0 {
			return http.StatusNotFound, azureARMErrorBody("ResourceNotFound", "asset "+azureARMString(id)+" was not found.")
		}
	}
	return http.StatusOK, nil
}
