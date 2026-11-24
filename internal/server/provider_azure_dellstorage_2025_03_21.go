package server

import (
	"net/http"
	"regexp"
	"strings"
)

const azureDellStorage20250321APIVersion = "2025-03-21"

const dellNamespace = "Dell.Storage"

// Dell.Storage file systems (Dell PowerScale for Azure), served by the shared
// ARM handler. Nothing is provisioned: a file system reports Succeeded at once
// with a deterministic SmartConnect FQDN and file system ID.
func init() {
	registerAzureARMResourceType(azureARMResourceType{
		Namespace:    dellNamespace,
		Type:         "filesystems",
		ValidateName: dellValidateName,
		Validate:     dellValidateFileSystem,
		// The SmartConnect password and the managed-identity secret never come back.
		WriteOnlyProperties: []string{"smartConnectFqdnPassword"},
		ReadOnlyProperties: func(rec *azureARMResource) map[string]any {
			return map[string]any{
				"fileSystemId":     azureARMDerivedKey(rec.ID, "filesystem"),
				"smartConnectFqdn": strings.ToLower(rec.Name) + ".dell.storage.azure.example",
				"oneFsUrl":         "https://" + strings.ToLower(rec.Name) + ".onefs.dell.storage.azure.example",
			}
		},
	})
	registerAzureARMView(dellNamespace, "operations", azureARMCrudOperations(dellNamespace, "Dell Storage", "filesystems"))
}

// handleAzureDellStorage20250321Router keeps the http.go dispatch slot; the
// shared ARM handler (registered above) serves every Dell.Storage route first.
func (s *Server) handleAzureDellStorage20250321Router(http.ResponseWriter, *http.Request) bool {
	return false
}

var (
	dellNamePattern = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,64}$`)
	dellCIDRPattern = regexp.MustCompile(`^(\d{1,3}\.){3}\d{1,3}/(16|24)$`)
)

func dellValidateName(name string) string {
	if !dellNamePattern.MatchString(name) {
		return "filesystemName must match ^[a-zA-Z0-9_-]*$ and be at most 64 characters."
	}
	return ""
}

func dellValidateFileSystem(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	if msg := azureARMRequireProperties(props, "delegatedSubnetId", "delegatedSubnetCidr", "dellReferenceNumber"); msg != "" {
		return msg
	}
	if !dellCIDRPattern.MatchString(azureARMString(props["delegatedSubnetCidr"])) {
		return "properties.delegatedSubnetCidr must be an IPv4 CIDR with a /16 or /24 prefix."
	}
	market := azureARMObject(props["marketplace"])
	for _, key := range []string{"planId", "offerId", "planName"} {
		if strings.TrimSpace(azureARMString(market[key])) == "" {
			return "properties.marketplace." + key + " is required."
		}
	}
	if strings.TrimSpace(azureARMString(azureARMObject(props["user"])["email"])) == "" {
		return "properties.user.email is required."
	}
	switch azureARMString(azureARMObject(props["encryption"])["encryptionType"]) {
	case "Microsoft-managed keys (MMK)", "Customer-managed keys (CMK)":
	default:
		return "properties.encryption.encryptionType must be 'Microsoft-managed keys (MMK)' or 'Customer-managed keys (CMK)'."
	}
	return ""
}

// azureARMRequireProperties returns a message naming the first of keys that
// is missing or blank in props, or "" when all are present.
func azureARMRequireProperties(props map[string]any, keys ...string) string {
	for _, key := range keys {
		v, ok := props[key]
		if !ok || v == nil {
			return "properties." + key + " is required."
		}
		if str, isString := v.(string); isString && strings.TrimSpace(str) == "" {
			return "properties." + key + " is required."
		}
	}
	return ""
}

// azureARMCrudOperations lists read, write and delete operations for each
// resource collection of a provider, for its /providers/{ns}/operations view.
func azureARMCrudOperations(ns, provider string, collections ...string) azureARMView {
	return func(*Server, string) []map[string]any {
		var ops []map[string]any
		for _, collection := range collections {
			for _, verb := range []string{"read", "write", "delete"} {
				ops = append(ops, map[string]any{
					"name": ns + "/" + collection + "/" + verb, "isDataAction": false, "origin": "user,system",
					"display": map[string]any{"provider": provider, "resource": collection, "operation": verb + " " + collection, "description": strings.ToUpper(verb[:1]) + verb[1:] + " " + collection + "."},
				})
			}
		}
		return ops
	}
}
