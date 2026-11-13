package server

import (
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"regexp"
	"slices"
	"strings"
)

// Microsoft.DocumentDB databaseAccounts (Azure Cosmos DB) with their SQL
// and MongoDB databases, containers and collections, served by the shared
// ARM handler. Mongo vCore clusters in the same namespace are in
// provider_azure_mongocluster.go.
func init() {
	const ns = "Microsoft.DocumentDB"
	registerAzureARMResourceType(azureARMResourceType{
		Namespace:          ns,
		Type:               "databaseAccounts",
		ReadOnlyProperties: azureCosmosDBAccountProperties,
		ValidateName:       azureCosmosDBValidateName,
		Validate:           azureCosmosDBValidateAccount,
		Actions: map[string]azureARMAction{
			"listkeys":              azureCosmosDBListKeys,
			"readonlykeys":          azureCosmosDBReadOnlyKeys,
			"listconnectionstrings": azureCosmosDBListConnectionStrings,
		},
	})
	for _, child := range []struct{ typ, key string }{
		{"databaseAccounts/sqlDatabases", ""},
		{"databaseAccounts/sqlDatabases/containers", "partitionKey"},
		{"databaseAccounts/mongodbDatabases", ""},
		{"databaseAccounts/mongodbDatabases/collections", ""},
	} {
		registerAzureARMResourceType(azureARMResourceType{Namespace: ns, Type: child.typ, Proxy: true, Validate: azureCosmosDBValidateChild(child.key)})
	}
	registerAzureARMNameCheck(ns, "databaseAccountNames", "databaseAccounts")
}

var (
	azureCosmosDBNamePattern  = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,42}[a-z0-9]$`)
	azureCosmosDBConsistency  = []string{"Eventual", "Session", "BoundedStaleness", "Strong", "ConsistentPrefix"}
	azureCosmosDBAccountKinds = []string{"", "GlobalDocumentDB", "MongoDB", "Parse"}
)

func azureCosmosDBValidateName(name string) string {
	if !azureCosmosDBNamePattern.MatchString(name) {
		return "Account names must be 3-44 lowercase letters, digits or hyphens, and cannot start or end with a hyphen."
	}
	return ""
}

func azureCosmosDBValidateAccount(payload map[string]any) string {
	if !slices.Contains(azureCosmosDBAccountKinds, azureARMString(payload["kind"])) {
		return "kind must be GlobalDocumentDB, MongoDB or Parse."
	}
	props := azureARMObject(payload["properties"])
	if azureARMString(props["databaseAccountOfferType"]) != "Standard" {
		return "properties.databaseAccountOfferType must be Standard."
	}
	if level := azureARMString(azureARMObject(props["consistencyPolicy"])["defaultConsistencyLevel"]); level != "" && !slices.Contains(azureCosmosDBConsistency, level) {
		return "properties.consistencyPolicy.defaultConsistencyLevel must be one of " + strings.Join(azureCosmosDBConsistency, ", ") + "."
	}
	locations, _ := props["locations"].([]any)
	if len(locations) == 0 {
		return "properties.locations must name at least one region."
	}
	seen := map[float64]bool{}
	for _, raw := range locations {
		loc := azureARMObject(raw)
		priority, _ := loc["failoverPriority"].(float64)
		if azureARMString(loc["locationName"]) == "" || seen[priority] {
			return "each properties.locations entry needs a locationName and a unique failoverPriority."
		}
		seen[priority] = true
	}
	if !seen[0] {
		return "properties.locations needs a write region with failoverPriority 0."
	}
	return ""
}

// azureCosmosDBValidateChild requires properties.resource.id, plus a
// partition key with /-rooted paths for containers.
func azureCosmosDBValidateChild(key string) func(map[string]any) string {
	return func(payload map[string]any) string {
		resource := azureARMObject(azureARMObject(payload["properties"])["resource"])
		if strings.TrimSpace(azureARMString(resource["id"])) == "" {
			return "properties.resource.id is required."
		}
		if key == "" {
			return ""
		}
		paths, _ := azureARMObject(resource[key])["paths"].([]any)
		if len(paths) == 0 {
			return "properties.resource.partitionKey.paths is required."
		}
		for _, path := range paths {
			if !strings.HasPrefix(azureARMString(path), "/") {
				return "properties.resource.partitionKey.paths must start with '/'."
			}
		}
		return ""
	}
}

func azureCosmosDBEndpoint(host string) string {
	return "https://" + host + ".documents.azure.com:443/"
}

// azureCosmosDBAccountProperties derives the endpoints and region lists from
// the requested locations: failover priority 0 is the write region.
func azureCosmosDBAccountProperties(rec *azureARMResource) map[string]any {
	var readLocations, writeLocations []any
	locations, _ := rec.Properties["locations"].([]any)
	for _, raw := range locations {
		loc := azureARMObject(raw)
		region := strings.ToLower(strings.ReplaceAll(azureARMString(loc["locationName"]), " ", ""))
		entry := map[string]any{
			"id":                rec.Name + "-" + region,
			"locationName":      azureARMString(loc["locationName"]),
			"documentEndpoint":  azureCosmosDBEndpoint(rec.Name + "-" + region),
			"provisioningState": "Succeeded",
			"failoverPriority":  loc["failoverPriority"],
			"isZoneRedundant":   loc["isZoneRedundant"] == true,
		}
		readLocations = append(readLocations, entry)
		if priority, _ := loc["failoverPriority"].(float64); priority == 0 {
			writeLocations = append(writeLocations, entry)
		}
	}
	props := map[string]any{
		"documentEndpoint": azureCosmosDBEndpoint(rec.Name),
		"readLocations":    readLocations,
		"writeLocations":   writeLocations,
	}
	if azureARMString(azureARMObject(rec.Properties["consistencyPolicy"])["defaultConsistencyLevel"]) == "" {
		props["consistencyPolicy"] = map[string]any{"defaultConsistencyLevel": "Session", "maxIntervalInSeconds": 5, "maxStalenessPrefix": 100}
	}
	return props
}

func azureCosmosDBKeys(record *azureARMResource) map[string]any {
	keys := map[string]any{}
	for _, label := range []string{"primaryMasterKey", "secondaryMasterKey", "primaryReadonlyMasterKey", "secondaryReadonlyMasterKey"} {
		// Cosmos DB keys are 64 bytes (88 base64 characters), derived from the ID.
		first := sha256.Sum256([]byte(azureARMKey(record.ID) + "#" + label + "#1"))
		second := sha256.Sum256([]byte(azureARMKey(record.ID) + "#" + label + "#2"))
		keys[label] = base64.StdEncoding.EncodeToString(append(first[:], second[:]...))
	}
	return keys
}

func azureCosmosDBListKeys(record *azureARMResource, _ map[string]any) (int, any) {
	return http.StatusOK, azureCosmosDBKeys(record)
}

func azureCosmosDBReadOnlyKeys(record *azureARMResource, _ map[string]any) (int, any) {
	keys := azureCosmosDBKeys(record)
	return http.StatusOK, map[string]any{"primaryReadonlyMasterKey": keys["primaryReadonlyMasterKey"], "secondaryReadonlyMasterKey": keys["secondaryReadonlyMasterKey"]}
}

func azureCosmosDBListConnectionStrings(record *azureARMResource, _ map[string]any) (int, any) {
	keys := azureCosmosDBKeys(record)
	endpoint := azureARMString(record.Properties["documentEndpoint"])
	var out []any
	for _, k := range []struct{ label, description string }{
		{"primaryMasterKey", "Primary SQL Connection String"},
		{"secondaryMasterKey", "Secondary SQL Connection String"},
		{"primaryReadonlyMasterKey", "Primary Read-Only SQL Connection String"},
		{"secondaryReadonlyMasterKey", "Secondary Read-Only SQL Connection String"},
	} {
		out = append(out, map[string]any{
			"connectionString": "AccountEndpoint=" + endpoint + ";AccountKey=" + azureARMString(keys[k.label]) + ";",
			"description":      k.description,
		})
	}
	return http.StatusOK, map[string]any{"connectionStrings": out}
}
