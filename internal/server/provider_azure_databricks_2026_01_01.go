package server

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// Azure Databricks (Microsoft.Databricks 2026-01-01): workspaces with their
// virtual network peerings and private endpoint connections, and access
// connectors, served by the shared ARM handler, plus private link
// resources, outbound network dependencies and the operations list. No
// Databricks control plane runs behind a workspace URL.

const (
	azureDatabricks20260101APIVersion = "2026-01-01"
	azureDBXNamespace                 = "Microsoft.Databricks"
)

var (
	azureDBXName          = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{1,62}[a-zA-Z0-9]$`)
	azureDBXResourceGroup = regexp.MustCompile(`(?i)^/subscriptions/[^/]+/resourceGroups/[^/]+$`)
)

func init() {
	ns := azureDBXNamespace
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "workspaces",
		ValidateName:       azureDBXValidateName,
		Validate:           azureDBXWorkspaceValidate,
		ReadOnlyProperties: azureDBXWorkspaceReadOnly,
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "workspaces/virtualNetworkPeerings", Proxy: true,
		Validate: azureDataPlatRequired("remoteVirtualNetwork.id"),
		ReadOnlyProperties: func(*azureARMResource) map[string]any {
			return map[string]any{"peeringState": "Connected"}
		},
		NoPatch: true,
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "workspaces/privateEndpointConnections", Proxy: true, CreateStatus: http.StatusOK,
		Validate: azureDataPlatAll(
			azureDataPlatRequired("privateLinkServiceConnectionState.status"),
			azureDataPlatEnum("privateLinkServiceConnectionState.status", "Pending", "Approved", "Rejected", "Disconnected"),
		),
		NoPatch: true, DeleteStatus: http.StatusNoContent,
	})
	registerAzureARMView(ns, "workspaces/privateLinkResources", func(_ *Server, scope string) []map[string]any {
		var out []map[string]any
		for _, group := range []string{"databricks_ui_api", "browser_authentication"} {
			out = append(out, map[string]any{
				"id": scope + "/privateLinkResources/" + group, "name": group, "type": ns + "/workspaces/privateLinkResources",
				"properties": map[string]any{"groupId": group, "requiredMembers": []any{group}, "requiredZoneNames": []any{"privatelink.azuredatabricks.net"}},
			})
		}
		return out
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "accessConnectors",
		ValidateName: azureDBXValidateName,
		Validate: func(payload map[string]any) string {
			switch azureARMString(azureARMObject(payload["identity"])["type"]) {
			case "", "None", "SystemAssigned", "UserAssigned":
				return ""
			}
			return "identity.type must be None, SystemAssigned or UserAssigned."
		},
		ReadOnlyProperties: func(*azureARMResource) map[string]any { return map[string]any{"referedBy": []any{}} },
	})
	registerAzureARMView(ns, "operations", azureDataPlatOperations(ns, "Microsoft Databricks",
		"workspaces/read", "workspaces/write", "workspaces/delete",
		"workspaces/virtualNetworkPeerings/read", "workspaces/virtualNetworkPeerings/write", "workspaces/virtualNetworkPeerings/delete",
		"workspaces/privateEndpointConnections/read", "workspaces/privateEndpointConnections/write", "workspaces/privateEndpointConnections/delete",
		"workspaces/privateLinkResources/read", "workspaces/outboundNetworkDependenciesEndpoints/read",
		"accessConnectors/read", "accessConnectors/write", "accessConnectors/delete"))
}

func azureDBXValidateName(name string) string {
	if !azureDBXName.MatchString(name) {
		return "The name '" + name + "' must be 3 to 64 letters, digits, underscores and hyphens, starting and ending with a letter or digit."
	}
	return ""
}

func azureDBXWorkspaceValidate(payload map[string]any) string {
	switch azureARMString(azureARMObject(payload["sku"])["name"]) {
	case "standard", "premium", "trial":
	default:
		return "sku.name must be standard, premium or trial."
	}
	group := azureARMString(azureARMObject(payload["properties"])["managedResourceGroupId"])
	if !azureDBXResourceGroup.MatchString(group) {
		return "properties.managedResourceGroupId must be a resource group ID."
	}
	return azureDataPlatEnum("publicNetworkAccess", "Enabled", "Disabled")(payload)
}

// azureDBXWorkspaceReadOnly derives a stable numeric workspace ID and its
// per-workspace URL from the resource ID.
func azureDBXWorkspaceReadOnly(rec *azureARMResource) map[string]any {
	sum := sha256.Sum256([]byte(strings.ToLower(rec.ID)))
	id := binary.BigEndian.Uint64(sum[:8]) % 1_000_000_000_000_000
	return map[string]any{
		"workspaceId":     fmt.Sprintf("%d", id),
		"workspaceUrl":    fmt.Sprintf("adb-%d.%d.azuredatabricks.net", id, sum[8]%20),
		"createdDateTime": rec.CreatedAt.Format(time.RFC3339),
	}
}

// handleAzureDatabricks20260101Router serves GET .../workspaces/{name}/
// outboundNetworkDependenciesEndpoints, whose body is a bare array rather
// than the list envelope the shared handler's views return.
func (s *Server) handleAzureDatabricks20260101Router(w http.ResponseWriter, r *http.Request) bool {
	seg := splitPathSegments(strings.TrimPrefix(rawRequestPath(r), "/azure/"))
	if len(seg) != 9 || !strings.EqualFold(seg[0], "subscriptions") || !strings.EqualFold(seg[2], "resourceGroups") ||
		!strings.EqualFold(seg[5], azureDBXNamespace) || !strings.EqualFold(seg[6], "workspaces") ||
		!strings.EqualFold(seg[8], "outboundNetworkDependenciesEndpoints") {
		return false
	}
	if r.Method != http.MethodGet {
		azureARMError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
		return true
	}
	if hasAzureInvalidAPIVersion(r) {
		azureARMError(w, http.StatusBadRequest, "InvalidApiVersionParameter", "invalid api-version query parameter")
		return true
	}
	id := "/subscriptions/" + seg[1] + "/resourceGroups/" + seg[3] + "/providers/" + azureDBXNamespace + "/workspaces/" + seg[7]
	s.providerStorageMu.Lock()
	workspace := s.azureDataPlatStoredLocked(id, azureDBXNamespace+"/workspaces")
	s.providerStorageMu.Unlock()
	if workspace == nil {
		azureARMNotFound(w, "ResourceNotFound", id, "")
		return true
	}
	endpoint := func(host string, port int) map[string]any {
		return map[string]any{"domainName": host, "endpointDetails": []any{map[string]any{"ipAddress": "", "port": port, "latency": 0, "isAccessible": true}}}
	}
	respondJSON(w, http.StatusOK, []any{
		map[string]any{"category": "Webapp", "endpoints": []any{map[string]any{"endpoints": []any{endpoint(azureARMString(workspace.Properties["workspaceUrl"]), 443)}}}},
		map[string]any{"category": "Control Plane NAT", "endpoints": []any{map[string]any{"endpoints": []any{endpoint("tunnel.azuredatabricks.net", 443)}}}},
		map[string]any{"category": "Extended infrastructure", "endpoints": []any{map[string]any{"endpoints": []any{endpoint("dbartifactsprod.blob.core.windows.net", 443)}}}},
	})
	return true
}
