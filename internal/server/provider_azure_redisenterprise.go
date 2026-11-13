package server

import (
	"regexp"
	"strings"
)

// Microsoft.Cache/redisEnterprise clusters (Azure Managed Redis and the
// Enterprise tiers) and their databases, served by the shared ARM handler.
func init() {
	const ns = "Microsoft.Cache"
	registerAzureARMResourceType(azureARMResourceType{
		Namespace:          ns,
		Type:               "redisEnterprise",
		ReadOnlyProperties: azureRedisEnterpriseClusterProperties,
		ValidateName:       azureRedisEnterpriseValidateName,
		Validate:           azureRedisEnterpriseValidateCluster,
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace:          ns,
		Type:               "redisEnterprise/databases",
		Proxy:              true,
		ReadOnlyProperties: azureRedisEnterpriseDatabaseProperties,
		ValidateName: func(name string) string {
			if name != "default" {
				return "The only supported database name is 'default'."
			}
			return ""
		},
		Validate: azureRedisEnterpriseValidateDatabase,
		Actions:  map[string]azureARMAction{"listkeys": azureRedisListKeys},
	})
}

var (
	azureRedisEnterpriseNamePattern = regexp.MustCompile(`^[a-zA-Z0-9]+(-[a-zA-Z0-9]+)*$`)
	azureRedisEnterpriseSKUPattern  = regexp.MustCompile(`^(Enterprise_E|EnterpriseFlash_F|Balanced_B|MemoryOptimized_M|ComputeOptimized_X|FlashOptimized_A)[0-9]+$`)
)

func azureRedisEnterpriseValidateName(name string) string {
	if len(name) > 60 || !azureRedisEnterpriseNamePattern.MatchString(name) {
		return "Cluster names must be 1-60 letters, digits or single hyphens, and start and end with a letter or digit."
	}
	return ""
}

func azureRedisEnterpriseValidateCluster(payload map[string]any) string {
	if !azureRedisEnterpriseSKUPattern.MatchString(azureARMString(azureARMObject(payload["sku"])["name"])) {
		return "sku.name must be an Azure Managed Redis or Enterprise SKU, for example Balanced_B5 or Enterprise_E10."
	}
	return ""
}

func azureRedisEnterpriseValidateDatabase(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	allowed := map[string][]string{
		"clientProtocol":   {"Encrypted", "Plaintext"},
		"clusteringPolicy": {"EnterpriseCluster", "OSSCluster", "NoCluster"},
	}
	for field, values := range allowed {
		value := azureARMString(props[field])
		if value == "" {
			continue
		}
		valid := false
		for _, v := range values {
			valid = valid || value == v
		}
		if !valid {
			return "properties." + field + " must be one of " + strings.Join(values, ", ") + "."
		}
	}
	return ""
}

// azureRedisEnterpriseClusterProperties builds the regional host name. The
// Enterprise tiers keep the older redisenterprise.cache.azure.net domain.
func azureRedisEnterpriseClusterProperties(rec *azureARMResource) map[string]any {
	domain := ".redis.azure.net"
	if strings.HasPrefix(azureARMString(rec.SKU["name"]), "Enterprise") {
		domain = ".redisenterprise.cache.azure.net"
	}
	region := strings.ToLower(strings.ReplaceAll(rec.Location, " ", ""))
	return map[string]any{
		"hostName":      strings.ToLower(rec.Name) + "." + region + domain,
		"resourceState": "Running",
	}
}

// azureRedisEnterpriseDatabaseProperties defaults the port to 10000.
// ponytail: Azure only accepts a port at creation; PATCH can change it here.
func azureRedisEnterpriseDatabaseProperties(rec *azureARMResource) map[string]any {
	port, isNumber := rec.Properties["port"].(float64)
	if !isNumber {
		port = 10000
	}
	return map[string]any{"port": port, "resourceState": "Running"}
}
