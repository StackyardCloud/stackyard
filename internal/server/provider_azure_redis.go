package server

import (
	"net"
	"net/http"
	"regexp"
	"strings"
)

// Microsoft.Cache Azure Cache for Redis caches and firewall rules, served by
// the shared ARM handler.
func init() {
	const ns = "Microsoft.Cache"
	registerAzureARMResourceType(azureARMResourceType{
		Namespace:          ns,
		Type:               "redis",
		ReadOnlyProperties: azureRedisProperties,
		ValidateName:       azureRedisValidateName,
		Validate:           azureRedisValidate,
		Actions:            map[string]azureARMAction{"listkeys": azureRedisListKeys},
	})
	registerAzureARMResourceType(azureARMResourceType{Namespace: ns, Type: "redis/firewallRules", Proxy: true, Validate: azureRedisValidateFirewallRule})
}

var azureRedisNamePattern = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?$`)

func azureRedisValidateName(name string) string {
	if !azureRedisNamePattern.MatchString(name) || strings.Contains(name, "--") {
		return "Cache names must be 1-63 letters, digits or single hyphens, and start and end with a letter or digit."
	}
	return ""
}

// azureRedisValidate checks properties.sku, where Redis (unlike most ARM
// types) carries its SKU: Basic/Standard use family C at capacity 0-6,
// Premium uses family P at capacity 1-5.
func azureRedisValidate(payload map[string]any) string {
	sku := azureARMObject(azureARMObject(payload["properties"])["sku"])
	capacity, isNumber := sku["capacity"].(float64)
	family := azureARMString(sku["family"])
	switch name := azureARMString(sku["name"]); {
	case name != "Basic" && name != "Standard" && name != "Premium":
		return "properties.sku.name must be one of Basic, Standard, Premium."
	case !isNumber || capacity != float64(int(capacity)):
		return "properties.sku.capacity is required."
	case name == "Premium" && (family != "P" || capacity < 1 || capacity > 5):
		return "Premium caches require properties.sku.family P and capacity 1-5."
	case name != "Premium" && (family != "C" || capacity < 0 || capacity > 6):
		return "Basic and Standard caches require properties.sku.family C and capacity 0-6."
	}
	return ""
}

func azureRedisValidateFirewallRule(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	for _, field := range []string{"startIP", "endIP"} {
		if ip := net.ParseIP(azureARMString(props[field])); ip == nil || ip.To4() == nil {
			return "properties.startIP and properties.endIP must be IPv4 addresses."
		}
	}
	return ""
}

func azureRedisProperties(rec *azureARMResource) map[string]any {
	return map[string]any{
		"hostName": strings.ToLower(rec.Name) + ".redis.cache.windows.net",
		"port":     6379,
		"sslPort":  6380,
	}
}

func azureRedisListKeys(record *azureARMResource, _ map[string]any) (int, any) {
	return http.StatusOK, map[string]any{
		"primaryKey":   azureARMDerivedKey(record.ID, "primary"),
		"secondaryKey": azureARMDerivedKey(record.ID, "secondary"),
	}
}
