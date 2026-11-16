package server

import (
	"fmt"
	"net/http"
	"slices"
	"strings"
)

// Microsoft.ServiceLinker linkers and dry runs on any source resource,
// served by the shared ARM handler. No network, identity or app setting
// changes are made: validation always succeeds and the configurations are
// computed from the target's resource type.
func init() {
	const ns = "Microsoft.ServiceLinker"
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "linkers", Extension: true,
		ValidateScope: azureServiceLinkerScope,
		ValidateName:  azureServiceLinkerName,
		Validate: func(payload map[string]any) string {
			return azureServiceLinkerValidate(azureARMObject(payload["properties"]), "properties")
		},
		WriteOnlyProperties: azureServiceLinkerSecrets("authInfo"),
		Actions: map[string]azureARMAction{
			"validatelinker":         azureServiceLinkerValidateLinker,
			"listconfigurations":     azureServiceLinkerConfigurations,
			"generateconfigurations": azureServiceLinkerConfigurations,
		},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "dryruns", Extension: true,
		ValidateScope:       azureServiceLinkerScope,
		ValidateName:        azureServiceLinkerName,
		Validate:            azureServiceLinkerDryrunValidate,
		WriteOnlyProperties: azureServiceLinkerSecrets("parameters.authInfo"),
		ReadOnlyProperties:  azureServiceLinkerDryrunProperties,
	})
}

var (
	azureServiceLinkerAuthTypes   = []string{"systemAssignedIdentity", "userAssignedIdentity", "servicePrincipalSecret", "servicePrincipalCertificate", "secret", "accessKey", "userAccount", "easyAuthMicrosoftEntraID"}
	azureServiceLinkerClientTypes = []string{"none", "dotnet", "java", "python", "go", "php", "ruby", "django", "nodejs", "springBoot", "kafka-springBoot", "jms-springBoot", "dapr"}
)

func azureServiceLinkerSecrets(auth string) []string {
	return []string{auth + ".secret", auth + ".certificate", auth + ".secretInfo.value"}
}

// azureServiceLinkerScope accepts only a resource as the connection source.
func azureServiceLinkerScope(scope string) string {
	if !strings.Contains(strings.ToLower(scope), "/providers/") {
		return "A connection's source must be a resource."
	}
	return ""
}

func azureServiceLinkerName(name string) string {
	for _, c := range name {
		if !(c == '.' || c == '_' || c >= '0' && c <= '9' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z') {
			return "Connection names may contain only letters, digits, periods and underscores."
		}
	}
	if len(name) > 60 {
		return "Connection names may be at most 60 characters."
	}
	return ""
}

// azureServiceLinkerValidate checks linker properties found at path.
func azureServiceLinkerValidate(props map[string]any, path string) string {
	target := azureARMObject(props["targetService"])
	switch azureARMString(target["type"]) {
	case "AzureResource":
		if !strings.Contains(strings.ToLower(azureARMString(target["id"])), "/providers/") {
			return path + ".targetService.id must be a resource ID."
		}
	case "ConfluentBootstrapServer", "ConfluentSchemaRegistry", "SelfHostedServer":
		if strings.TrimSpace(azureARMString(target["endpoint"])) == "" {
			return path + ".targetService.endpoint is required."
		}
	default:
		return path + ".targetService.type must be AzureResource, ConfluentBootstrapServer, ConfluentSchemaRegistry or SelfHostedServer."
	}
	if raw, present := props["clientType"]; present && !slices.Contains(azureServiceLinkerClientTypes, azureARMString(raw)) {
		return path + ".clientType must be one of " + strings.Join(azureServiceLinkerClientTypes, ", ") + "."
	}
	if raw, present := azureARMObject(props["vNetSolution"])["type"]; present && raw != nil && !slices.Contains([]string{"serviceEndpoint", "privateLink"}, azureARMString(raw)) {
		return path + ".vNetSolution.type must be serviceEndpoint or privateLink."
	}
	auth, present := props["authInfo"]
	if !present {
		return ""
	}
	return azureServiceLinkerAuthValidate(azureARMObject(auth), path+".authInfo")
}

func azureServiceLinkerAuthValidate(auth map[string]any, path string) string {
	authType := azureARMString(auth["authType"])
	if !slices.Contains(azureServiceLinkerAuthTypes, authType) {
		return path + ".authType must be one of " + strings.Join(azureServiceLinkerAuthTypes, ", ") + "."
	}
	required := map[string][]string{
		"servicePrincipalSecret":      {"clientId", "principalId", "secret"},
		"servicePrincipalCertificate": {"clientId", "principalId", "certificate"},
	}[authType]
	for _, key := range required {
		if strings.TrimSpace(azureARMString(auth[key])) == "" {
			return path + "." + key + " is required for " + authType + " authentication."
		}
	}
	if raw, present := azureARMObject(auth["secretInfo"])["secretType"]; present && !slices.Contains([]string{"rawValue", "keyVaultSecretUri", "keyVaultSecretReference"}, azureARMString(raw)) {
		return path + ".secretInfo.secretType must be rawValue, keyVaultSecretUri or keyVaultSecretReference."
	}
	return ""
}

func azureServiceLinkerDryrunValidate(payload map[string]any) string {
	params := azureARMObject(azureARMObject(payload["properties"])["parameters"])
	if params["actionName"] != "createOrUpdate" {
		return "properties.parameters.actionName must be createOrUpdate."
	}
	return azureServiceLinkerValidate(params, "properties.parameters")
}

// azureServiceLinkerSource returns the source resource ID of a linker or
// dry run record.
func azureServiceLinkerSource(rec *azureARMResource) string {
	return rec.ID[:strings.LastIndex(strings.ToLower(rec.ID), "/providers/microsoft.servicelinker/")]
}

// azureServiceLinkerDryrunProperties previews the one change a connection
// always makes: app settings on the source.
func azureServiceLinkerDryrunProperties(rec *azureARMResource) map[string]any {
	source := azureServiceLinkerSource(rec)
	return map[string]any{
		"prerequisiteResults": []any{},
		"operationPreviews": []any{map[string]any{
			"name": "configConnection", "operationType": "configConnection",
			"description": "Configure the connection's settings on the source resource",
			"action":      azureARMResourceTypeOf(source) + "/write",
			"scope":       source,
		}},
	}
}

// azureARMResourceTypeOf returns the full type of a resource ID, e.g.
// "Microsoft.Web/sites" or "Microsoft.Web/sites/slots".
func azureARMResourceTypeOf(id string) string {
	seg := strings.Split(strings.Trim(id, "/"), "/")
	i := len(seg) - 1
	for i >= 0 && !strings.EqualFold(seg[i], "providers") {
		i--
	}
	if i < 0 || i+1 >= len(seg) {
		return ""
	}
	types := []string{seg[i+1]}
	for j := i + 2; j < len(seg); j += 2 {
		types = append(types, seg[j])
	}
	return strings.Join(types, "/")
}

func azureServiceLinkerValidateLinker(rec *azureARMResource, _ map[string]any) (int, any) {
	target := azureARMObject(rec.Properties["targetService"])
	targetID := azureARMString(target["id"])
	if targetID == "" {
		targetID = azureARMString(target["endpoint"])
	}
	now := rec.UpdatedAt.Format("2006-01-02T15:04:05Z")
	return http.StatusOK, map[string]any{
		"resourceId": rec.ID,
		"status":     "Succeeded",
		"properties": map[string]any{
			"linkerName":            rec.Name,
			"isConnectionAvailable": true,
			"reportStartTimeUtc":    now,
			"reportEndTimeUtc":      now,
			"sourceId":              azureServiceLinkerSource(rec),
			"targetId":              targetID,
			"authType":              azureARMObject(rec.Properties["authInfo"])["authType"],
			"validationDetail": []any{
				map[string]any{"name": "TargetExistence", "description": "The target existence is validated", "result": "success"},
			},
		},
	}
}

// azureServiceLinkerTargets maps a target resource type to its settings
// prefix, the setting naming its endpoint, and that endpoint's format.
// ponytail: common targets only; others get AZURE_RESOURCE_ID.
var azureServiceLinkerTargets = map[string][3]string{
	"microsoft.storage/storageaccounts":              {"AZURE_STORAGEBLOB", "RESOURCEENDPOINT", "https://%s.blob.core.windows.net/"},
	"microsoft.keyvault/vaults":                      {"AZURE_KEYVAULT", "RESOURCEENDPOINT", "https://%s.vault.azure.net/"},
	"microsoft.servicebus/namespaces":                {"AZURE_SERVICEBUS", "FULLYQUALIFIEDNAMESPACE", "%s.servicebus.windows.net"},
	"microsoft.eventhub/namespaces":                  {"AZURE_EVENTHUB", "FULLYQUALIFIEDNAMESPACE", "%s.servicebus.windows.net"},
	"microsoft.appconfiguration/configurationstores": {"AZURE_APPCONFIGURATION", "ENDPOINT", "https://%s.azconfig.io"},
	"microsoft.documentdb/databaseaccounts":          {"AZURE_COSMOS", "RESOURCEENDPOINT", "https://%s.documents.azure.com:443/"},
	"microsoft.dbforpostgresql/flexibleservers":      {"AZURE_POSTGRESQL", "HOST", "%s.postgres.database.azure.com"},
	"microsoft.dbformysql/flexibleservers":           {"AZURE_MYSQL", "HOST", "%s.mysql.database.azure.com"},
	"microsoft.sql/servers":                          {"AZURE_SQL", "SERVER", "%s.database.windows.net"},
	"microsoft.cache/redis":                          {"AZURE_REDIS", "HOST", "%s.redis.cache.windows.net"},
}

// azureServiceLinkerConfigurations returns the settings the connection
// puts on the source: the target endpoint, plus the client ID for a
// user-assigned identity. Secret-based connections name a connection
// string instead, whose value is never returned. A generateConfigurations
// payload's customizedKeys rename settings.
func azureServiceLinkerConfigurations(rec *azureARMResource, payload map[string]any) (int, any) {
	target := azureARMObject(rec.Properties["targetService"])
	auth := azureARMObject(rec.Properties["authInfo"])
	prefix, key, value := "AZURE_RESOURCE", "ID", azureARMString(target["id"])
	if value == "" {
		prefix, key, value = "AZURE_ENDPOINT", "URL", azureARMString(target["endpoint"])
	}
	seg := strings.Split(strings.Trim(value, "/"), "/")
	for i := 0; i+3 < len(seg); i++ {
		if !strings.EqualFold(seg[i], "providers") {
			continue
		}
		if known, ok := azureServiceLinkerTargets[strings.ToLower(seg[i+1]+"/"+seg[i+2])]; ok {
			prefix, key, value = known[0], known[1], fmt.Sprintf(known[2], strings.ToLower(seg[i+3]))
		}
		break
	}
	settings := []map[string]any{}
	if authType := azureARMString(auth["authType"]); authType == "secret" || authType == "accessKey" {
		settings = append(settings, map[string]any{"name": prefix + "_CONNECTIONSTRING", "value": "", "configType": "Default"})
	} else {
		settings = append(settings, map[string]any{"name": prefix + "_" + key, "value": value, "configType": "Default"})
		if authType == "userAssignedIdentity" {
			settings = append(settings, map[string]any{"name": prefix + "_CLIENTID", "value": azureARMString(auth["clientId"]), "configType": "Default"})
		}
	}
	renames := azureARMMerge(azureARMObject(azureARMObject(rec.Properties["configurationInfo"])["customizedKeys"]), azureARMObject(payload["customizedKeys"]))
	out := make([]any, 0, len(settings))
	for _, setting := range settings {
		if renamed := azureARMString(renames[azureARMString(setting["name"])]); renamed != "" {
			setting["name"] = renamed
		}
		out = append(out, setting)
	}
	return http.StatusOK, map[string]any{"configurations": out}
}
