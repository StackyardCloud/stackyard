package server

import (
	"net/http"
	"regexp"
	"slices"
	"strings"
)

const azureDevCenterAdministrator20240201APIVersion = "2024-02-01"

const dcadmNamespace = "Microsoft.DevCenter"

// Microsoft.DevCenter management plane, served by the shared ARM handler.
// Nothing is provisioned: catalogs sync at once and expose one Sandbox
// environment definition, each dev center gets the built-in Default gallery
// of marketplace images, and network connection health checks pass.
func init() {
	ns := dcadmNamespace
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "devcenters", ValidateName: dcadmValidateName,
		DefaultChildren: func(parent *azureARMResource) []*azureARMResource {
			typ, _ := lookupAzureARMResourceType(ns, "devcenters/galleries")
			return []*azureARMResource{azureARMChild(parent, typ, "Default", map[string]any{"galleryResourceId": ""})}
		},
		ReadOnlyProperties: func(rec *azureARMResource) map[string]any {
			return map[string]any{"devCenterUri": "https://" + azureARMDerivedKey(rec.ID, "uri")[:12] + "-" + strings.ToLower(rec.Name) + "." + strings.ToLower(rec.Location) + ".devcenter.azure.com/"}
		},
	})
	for _, parent := range []string{"devcenters", "projects"} {
		registerAzureARMResourceType(azureARMResourceType{
			Namespace: ns, Type: parent + "/catalogs", Proxy: true, ValidateName: dcadmValidateName,
			Validate: dcadmValidateCatalog, ReadOnlyProperties: dcadmCatalogStatus,
			DefaultChildren: dcadmSandboxDefinition(parent + "/catalogs/environmentDefinitions"),
			Actions: map[string]azureARMAction{
				"sync":                func(*azureARMResource, map[string]any) (int, any) { return http.StatusAccepted, nil },
				"connect":             func(*azureARMResource, map[string]any) (int, any) { return http.StatusAccepted, nil },
				"getsyncerrordetails": dcadmNoErrors,
			},
		})
		registerAzureARMResourceType(azureARMResourceType{
			Namespace: ns, Type: parent + "/catalogs/environmentDefinitions", Proxy: true, NoPut: true, NoPatch: true, NoDelete: true,
			Actions: map[string]azureARMAction{"geterrordetails": dcadmNoErrors},
		})
		registerAzureARMResourceType(azureARMResourceType{
			Namespace: ns, Type: parent + "/environmentTypes", Proxy: true, ValidateName: dcadmValidateName,
			Validate: dcadmValidateEnvironmentType(parent == "projects"),
		})
	}
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "devcenters/attachednetworks", Proxy: true,
		Validate: func(payload map[string]any) string {
			return azureARMRequireProperties(azureARMObject(payload["properties"]), "networkConnectionId")
		},
		ReadOnlyProperties: func(rec *azureARMResource) map[string]any {
			return map[string]any{"domainJoinType": "AzureADJoin", "healthCheckStatus": "Passed", "networkConnectionLocation": "eastus"}
		},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "devcenters/devboxdefinitions", ValidateName: dcadmValidateName,
		Validate: dcadmValidateDevBoxDefinition,
		ReadOnlyProperties: func(rec *azureARMResource) map[string]any {
			return map[string]any{"imageValidationStatus": "Succeeded", "osStorageType": "ssd_256gb", "activeImageReference": rec.Properties["imageReference"]}
		},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "devcenters/galleries", Proxy: true, ValidateName: dcadmValidateName,
		Validate: func(payload map[string]any) string {
			return azureARMRequireProperties(azureARMObject(payload["properties"]), "galleryResourceId")
		},
	})
	registerAzureARMView(ns, "devcenters/galleries/images", dcadmImages)
	registerAzureARMView(ns, "devcenters/galleries/images/versions", dcadmImageVersions)
	registerAzureARMView(ns, "devcenters/images", func(s *Server, scope string) []map[string]any {
		return dcadmImages(s, scope+"/galleries/Default")
	})

	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "networkConnections", ValidateName: dcadmValidateName,
		Validate: dcadmValidateNetworkConnection,
		ReadOnlyProperties: func(*azureARMResource) map[string]any {
			return map[string]any{"healthCheckStatus": "Passed"}
		},
		Actions: map[string]azureARMAction{
			"runhealthchecks": func(*azureARMResource, map[string]any) (int, any) { return http.StatusAccepted, nil },
		},
	})
	registerAzureARMView(ns, "networkConnections/healthChecks", dcadmHealthChecks)
	registerAzureARMView(ns, "networkConnections/outboundNetworkDependenciesEndpoints", func(_ *Server, scope string) []map[string]any {
		return []map[string]any{{
			"category": "Dev Box Service", "endpoints": []any{map[string]any{
				"domainName": "*.devcenter.azure.com", "endpointDetails": []any{map[string]any{"port": 443}},
			}},
		}}
	})

	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "projects", ValidateName: dcadmValidateName,
		Validate: func(payload map[string]any) string {
			props := azureARMObject(payload["properties"])
			if msg := azureARMRequireProperties(props, "devCenterId"); msg != "" {
				return msg
			}
			if max, ok := props["maxDevBoxesPerUser"].(float64); ok && max < 0 {
				return "properties.maxDevBoxesPerUser must not be negative."
			}
			return ""
		},
		ReadOnlyProperties: func(rec *azureARMResource) map[string]any {
			return map[string]any{"devCenterUri": "https://" + strings.ToLower(rec.Name) + ".devcenter.azure.com/"}
		},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "projects/pools", ValidateName: dcadmValidateName,
		Validate: dcadmValidatePool,
		ReadOnlyProperties: func(*azureARMResource) map[string]any {
			return map[string]any{"healthStatus": "Healthy", "devBoxCount": 0}
		},
		Actions: map[string]azureARMAction{
			"runhealthchecks": func(*azureARMResource, map[string]any) (int, any) { return http.StatusAccepted, nil },
		},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "projects/pools/schedules", Proxy: true, Validate: dcadmValidateSchedule,
	})
	registerAzureARMView(ns, "projects/attachednetworks", dcadmFromDevCenter("attachednetworks", "projects/attachednetworks"))
	registerAzureARMView(ns, "projects/devboxdefinitions", dcadmFromDevCenter("devboxdefinitions", "projects/devboxdefinitions"))
	registerAzureARMView(ns, "projects/allowedEnvironmentTypes", dcadmFromDevCenter("environmentTypes", "projects/allowedEnvironmentTypes"))

	registerAzureARMProviderAction(ns, "checkNameAvailability", dcadmCheckName(false))
	registerAzureARMProviderAction(ns, "checkScopedNameAvailability", dcadmCheckName(true))
	registerAzureARMView(ns, "skus", func(_ *Server, scope string) []map[string]any {
		var out []map[string]any
		for _, size := range []string{"8c32gb", "16c64gb", "32c128gb"} {
			out = append(out, map[string]any{
				"name": "general_i_" + size + "256ssd_v2", "tier": "Standard", "resourceType": "projects/pools",
				"locations": []any{"eastus", "westus3", "westeurope"}, "capabilities": []any{map[string]any{"name": "vCPUs", "value": strings.Split(size, "c")[0]}},
			})
		}
		return out
	})
	registerAzureARMView(ns, "locations/usages", func(_ *Server, scope string) []map[string]any {
		return []map[string]any{{
			"id": scope + "/usages/devCenters", "currentValue": 0, "limit": 20, "unit": "Count",
			"name": map[string]any{"value": "devCenters", "localizedValue": "Dev Centers"},
		}}
	})
	registerAzureARMView(ns, "operations", azureARMCrudOperations(ns, "Microsoft Dev Center",
		"devcenters", "projects", "networkConnections", "devcenters/catalogs", "projects/pools"))
}

var dcadmNamePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9-_.]{2,62}$`)

func dcadmValidateName(name string) string {
	if !dcadmNamePattern.MatchString(name) {
		return "names must be 3-63 characters of letters, digits, '-', '_' or '.', starting with a letter or digit."
	}
	return ""
}

func dcadmNoErrors(*azureARMResource, map[string]any) (int, any) {
	return http.StatusOK, map[string]any{"conflicts": []any{}, "errors": []any{}}
}

func dcadmValidateCatalog(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	git := azureARMObject(props["gitHub"])
	if git == nil {
		git = azureARMObject(props["adoGit"])
	}
	if git == nil {
		return "properties.gitHub or properties.adoGit is required."
	}
	if strings.TrimSpace(azureARMString(git["uri"])) == "" {
		return "the catalog's uri is required."
	}
	if v, ok := props["syncType"]; ok && !slices.Contains([]string{"Manual", "Scheduled"}, azureARMString(v)) {
		return "properties.syncType must be Manual or Scheduled."
	}
	return ""
}

func dcadmCatalogStatus(rec *azureARMResource) map[string]any {
	stamp := rec.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z")
	return map[string]any{
		"syncState": "Succeeded", "connectionState": "Connected", "lastSyncTime": stamp, "lastConnectionTime": stamp,
		"lastSyncStats": map[string]any{"added": 1, "updated": 0, "unchanged": 0, "removed": 0, "validationErrors": 0, "synchronizationErrors": 0},
	}
}

func dcadmSandboxDefinition(childType string) func(*azureARMResource) []*azureARMResource {
	return func(parent *azureARMResource) []*azureARMResource {
		typ, _ := lookupAzureARMResourceType(dcadmNamespace, childType)
		return []*azureARMResource{azureARMChild(parent, typ, "Sandbox", map[string]any{
			"description": "Empty sandbox environment.", "templatePath": "Environments/Sandbox/environment.yaml",
			"parameters": []any{}, "validationStatus": "Succeeded",
		})}
	}
}

func dcadmValidateEnvironmentType(project bool) func(map[string]any) string {
	return func(payload map[string]any) string {
		if !project {
			return ""
		}
		props := azureARMObject(payload["properties"])
		if v, ok := props["status"]; ok && !slices.Contains([]string{"Enabled", "Disabled"}, azureARMString(v)) {
			return "properties.status must be Enabled or Disabled."
		}
		return azureARMRequireProperties(props, "deploymentTargetId")
	}
}

func dcadmValidateDevBoxDefinition(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	if strings.TrimSpace(azureARMString(azureARMObject(props["imageReference"])["id"])) == "" {
		return "properties.imageReference.id is required."
	}
	if strings.TrimSpace(azureARMString(azureARMObject(props["sku"])["name"])) == "" {
		return "properties.sku.name is required."
	}
	if v, ok := props["hibernateSupport"]; ok && !slices.Contains([]string{"Enabled", "Disabled"}, azureARMString(v)) {
		return "properties.hibernateSupport must be Enabled or Disabled."
	}
	return ""
}

func dcadmValidateNetworkConnection(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	if msg := azureARMRequireProperties(props, "subnetId"); msg != "" {
		return msg
	}
	join := azureARMString(props["domainJoinType"])
	if !slices.Contains([]string{"AzureADJoin", "HybridAzureADJoin", "None"}, join) {
		return "properties.domainJoinType must be AzureADJoin, HybridAzureADJoin or None."
	}
	if join == "HybridAzureADJoin" {
		return azureARMRequireProperties(props, "domainName")
	}
	return ""
}

func dcadmHealthChecks(_ *Server, scope string) []map[string]any {
	return []map[string]any{{
		"id": scope + "/healthChecks/latest", "name": "latest", "type": dcadmNamespace + "/networkConnections/healthChecks",
		"properties": map[string]any{"healthChecks": []any{
			map[string]any{"displayName": "Azure tenant readiness", "status": "Passed"},
			map[string]any{"displayName": "Virtual network", "status": "Passed"},
		}},
	}}
}

func dcadmValidatePool(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	if msg := azureARMRequireProperties(props, "devBoxDefinitionName", "networkConnectionName"); msg != "" {
		return msg
	}
	if azureARMString(props["licenseType"]) != "Windows_Client" {
		return "properties.licenseType must be Windows_Client."
	}
	if !slices.Contains([]string{"Enabled", "Disabled"}, azureARMString(props["localAdministrator"])) {
		return "properties.localAdministrator must be Enabled or Disabled."
	}
	return ""
}

var dcadmTimePattern = regexp.MustCompile(`^([01]\d|2[0-3]):[0-5]\d$`)

func dcadmValidateSchedule(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	if azureARMString(props["type"]) != "StopDevBox" || azureARMString(props["frequency"]) != "Daily" {
		return "properties.type must be StopDevBox and properties.frequency Daily."
	}
	if !dcadmTimePattern.MatchString(azureARMString(props["time"])) {
		return "properties.time must be HH:mm."
	}
	return azureARMRequireProperties(props, "timeZone")
}

var dcadmGalleryImages = []string{
	"microsoftwindowsdesktop_windows-ent-cpc_win11-24h2-ent-cpc",
	"microsoftvisualstudio_visualstudioplustools_vs-2022-ent-general-win11-m365-gen2",
}

func dcadmImages(_ *Server, scope string) []map[string]any {
	out := make([]map[string]any, 0, len(dcadmGalleryImages))
	for _, name := range dcadmGalleryImages {
		out = append(out, map[string]any{
			"id": scope + "/images/" + name, "name": name, "type": dcadmNamespace + "/devcenters/galleries/images",
			"properties": map[string]any{
				"description": name, "publisher": strings.Split(name, "_")[0], "offer": strings.Split(name, "_")[1],
				"recommendedMachineConfiguration": map[string]any{"memory": map[string]any{"min": 32, "max": 128}, "vCPUs": map[string]any{"min": 8, "max": 32}},
				"hibernateSupport":                "Enabled", "provisioningState": "Succeeded",
			},
		})
	}
	return out
}

func dcadmImageVersions(_ *Server, scope string) []map[string]any {
	return []map[string]any{{
		"id": scope + "/versions/1.0.0", "name": "1.0.0", "type": dcadmNamespace + "/devcenters/galleries/images/versions",
		"properties": map[string]any{"name": "1.0.0", "publishedDate": "2026-01-01T00:00:00Z", "excludeFromLatest": false, "osDiskImageSizeInGb": 64, "provisioningState": "Succeeded"},
	}}
}

// dcadmFromDevCenter lists the project's dev center's children of kind,
// re-homed under the project as typ.
func dcadmFromDevCenter(kind, typ string) azureARMView {
	return func(s *Server, scope string) []map[string]any {
		project := s.azureARMResourcesWhere(func(rec *azureARMResource) bool { return strings.EqualFold(rec.ID, scope) })
		if len(project) == 0 {
			return nil
		}
		prefix := strings.ToLower(azureARMString(project[0].Properties["devCenterId"])) + "/" + strings.ToLower(kind) + "/"
		recs := s.azureARMResourcesWhere(func(rec *azureARMResource) bool {
			rest, ok := strings.CutPrefix(strings.ToLower(rec.ID), prefix)
			return ok && !strings.Contains(rest, "/")
		})
		out := make([]map[string]any, 0, len(recs))
		for _, rec := range recs {
			item := rec.fixture()
			item["id"], item["type"] = scope+"/"+kind+"/"+rec.Name, dcadmNamespace+"/"+typ
			out = append(out, item)
		}
		return out
	}
}

func dcadmCheckName(scoped bool) azureARMProviderAction {
	return func(s *Server, payload map[string]any) (int, any) {
		name, typ := strings.TrimSpace(azureARMString(payload["name"])), azureARMString(payload["type"])
		if name == "" || typ == "" || (scoped && azureARMString(payload["scope"]) == "") {
			return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "name, type and (for scoped checks) scope are required.")
		}
		if msg := dcadmValidateName(name); msg != "" {
			return http.StatusOK, map[string]any{"nameAvailable": false, "reason": "Invalid", "message": msg}
		}
		scope := strings.ToLower(azureARMString(payload["scope"]))
		taken := len(s.azureARMResourcesWhere(func(rec *azureARMResource) bool {
			return strings.EqualFold(rec.Type, typ) && strings.EqualFold(rec.Name, name) && (!scoped || strings.HasPrefix(strings.ToLower(rec.ID), scope+"/"))
		})) > 0
		if taken {
			return http.StatusOK, map[string]any{"nameAvailable": false, "reason": "AlreadyExists", "message": name + " is already in use."}
		}
		return http.StatusOK, map[string]any{"nameAvailable": true}
	}
}
