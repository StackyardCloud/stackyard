package server

import (
	"regexp"
	"strings"
)

// Microsoft Discovery (Microsoft.Discovery, api-version 2026-06-01) on the
// shared ARM handler: workspaces with projects and chat model deployments,
// bookshelves, storage containers with storage assets, supercomputers with
// node pools, tools, private endpoint connections and private link resources
// for workspaces and bookshelves, and the operations list. Nothing is
// computed or indexed; endpoints are derived from resource names.
const (
	discNS       = "Microsoft.Discovery"
	discMaxRefs  = 50
	discMaxNodes = 1000
)

func init() {
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: discNS, Type: "workspaces",
		Validate: discAll(discRefs("supercomputerIds", "supercomputers"), discPublicAccess),
		ReadOnlyProperties: func(rec *azureARMResource) map[string]any {
			host := strings.ToLower(rec.Name) + "." + discRegion(rec) + ".discovery.azure.com"
			return map[string]any{"workspaceApiUri": "https://" + host, "workspaceUiUri": "https://ui." + host}
		},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: discNS, Type: "workspaces/projects",
		Validate: discRefs("storageContainerIds", "storageContainers"),
		ReadOnlyProperties: func(rec *azureARMResource) map[string]any {
			return map[string]any{"foundryProjectEndpoint": "https://" + strings.ToLower(rec.Name) + "." + discRegion(rec) + ".projects.discovery.azure.com"}
		},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: discNS, Type: "workspaces/chatModelDeployments",
		Validate: discAll(discRequire("modelFormat"), discRequire("modelName")),
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: discNS, Type: "bookshelves",
		Validate: discPublicAccess,
		ReadOnlyProperties: func(rec *azureARMResource) map[string]any {
			return map[string]any{"bookshelfUri": "https://" + strings.ToLower(rec.Name) + "." + discRegion(rec) + ".bookshelf.discovery.azure.com"}
		},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: discNS, Type: "storageContainers",
		Validate: discValidateStorageStore,
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: discNS, Type: "storageContainers/storageAssets",
		Validate: discRequire("path"),
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: discNS, Type: "supercomputers",
		Validate: discRequireSubnet("subnetId"),
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: discNS, Type: "supercomputers/nodePools",
		Validate: discValidateNodePool,
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: discNS, Type: "tools",
		Validate: func(payload map[string]any) string {
			if _, ok := azureARMObject(payload["properties"])["definitionContent"].(map[string]any); !ok {
				return "properties.definitionContent is missing or is not an object."
			}
			return ""
		},
	})
	registerAzureARMPrivateLink(discNS, "workspaces", "workspace", "privatelink.discovery.azure.com")
	registerAzureARMPrivateLink(discNS, "bookshelves", "bookshelf", "privatelink.bookshelf.discovery.azure.com")
	registerAzureARMOperations(discNS, "Microsoft Discovery", [][2]string{
		{"workspaces", "workspaces"}, {"workspaces/projects", "projects"}, {"workspaces/chatModelDeployments", "chat model deployments"},
		{"bookshelves", "bookshelves"}, {"storageContainers", "storage containers"}, {"storageContainers/storageAssets", "storage assets"},
		{"supercomputers", "supercomputers"}, {"supercomputers/nodePools", "node pools"}, {"tools", "tools"},
	})
}

var discSubnetID = regexp.MustCompile(`(?i)^/subscriptions/[^/]+/resourcegroups/[^/]+/providers/microsoft\.network/virtualnetworks/[^/]+/subnets/[^/]+$`)

func discRegion(rec *azureARMResource) string {
	return strings.ToLower(strings.ReplaceAll(rec.Location, " ", ""))
}

func discAll(checks ...func(map[string]any) string) func(map[string]any) string {
	return func(payload map[string]any) string {
		for _, check := range checks {
			if msg := check(payload); msg != "" {
				return msg
			}
		}
		return ""
	}
}

func discRequire(field string) func(map[string]any) string {
	return func(payload map[string]any) string {
		if strings.TrimSpace(azureARMString(azureARMObject(payload["properties"])[field])) == "" {
			return "properties." + field + " is missing."
		}
		return ""
	}
}

func discRequireSubnet(field string) func(map[string]any) string {
	return func(payload map[string]any) string {
		if !discSubnetID.MatchString(azureARMString(azureARMObject(payload["properties"])[field])) {
			return "properties." + field + " is missing or is not a subnet ID."
		}
		return ""
	}
}

// discRefs checks an optional array of Microsoft.Discovery resource IDs of
// the given top-level type.
func discRefs(field, typ string) func(map[string]any) string {
	marker := "/providers/microsoft.discovery/" + strings.ToLower(typ) + "/"
	return func(payload map[string]any) string {
		raw, present := azureARMObject(payload["properties"])[field]
		if !present || raw == nil {
			return ""
		}
		ids, ok := raw.([]any)
		if !ok || len(ids) > discMaxRefs {
			return "properties." + field + " must be an array of at most 50 IDs."
		}
		for _, item := range ids {
			if id, _ := item.(string); !strings.Contains(strings.ToLower(id), marker) {
				return "properties." + field + " holds an invalid " + typ + " ID."
			}
		}
		return ""
	}
}

func discPublicAccess(payload map[string]any) string {
	switch azureARMString(azureARMObject(payload["properties"])["publicNetworkAccess"]) {
	case "", "Enabled", "Disabled":
		return ""
	}
	return "properties.publicNetworkAccess must be Enabled or Disabled."
}

func discValidateStorageStore(payload map[string]any) string {
	store := azureARMObject(azureARMObject(payload["properties"])["storageStore"])
	switch azureARMString(store["kind"]) {
	case "AzureStorageBlob":
		if !strings.Contains(strings.ToLower(azureARMString(store["storageAccountId"])), "/providers/microsoft.storage/storageaccounts/") {
			return "properties.storageStore.storageAccountId must be a storage account ID."
		}
	case "AzureNetApp":
		if azureARMString(store["netAppVolumeId"]) == "" {
			return "properties.storageStore.netAppVolumeId is missing."
		}
	default:
		return "properties.storageStore.kind must be AzureStorageBlob or AzureNetApp."
	}
	return ""
}

func discValidateNodePool(payload map[string]any) string {
	if msg := discAll(discRequireSubnet("subnetId"), discRequire("vmSize"))(payload); msg != "" {
		return msg
	}
	props := azureARMObject(payload["properties"])
	maxNodes, ok := props["maxNodeCount"].(float64)
	if !ok || maxNodes < 1 || maxNodes > discMaxNodes || maxNodes != float64(int64(maxNodes)) {
		return "properties.maxNodeCount must be an integer between 1 and 1000."
	}
	if raw, present := props["minNodeCount"]; present && raw != nil {
		minNodes, ok := raw.(float64)
		if !ok || minNodes < 0 || minNodes > maxNodes || minNodes != float64(int64(minNodes)) {
			return "properties.minNodeCount must be an integer between 0 and maxNodeCount."
		}
	}
	switch azureARMString(props["scaleSetPriority"]) {
	case "", "Regular", "Spot":
		return ""
	}
	return "properties.scaleSetPriority must be Regular or Spot."
}
