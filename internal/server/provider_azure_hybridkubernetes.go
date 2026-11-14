package server

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Microsoft.Kubernetes Arc-enabled Kubernetes clusters, served by the shared
// ARM handler. Clusters report Connected at once; no Arc agents run.
func init() {
	registerAzureARMResourceType(azureARMResourceType{
		Namespace:          "Microsoft.Kubernetes",
		Type:               "connectedClusters",
		ReadOnlyProperties: azureConnectedClusterProperties,
		Validate:           azureConnectedClusterValidate,
		Actions:            map[string]azureARMAction{"listclusterusercredential": azureConnectedClusterUserCredential},
	})
}

func azureConnectedClusterValidate(payload map[string]any) string {
	if identity := azureARMString(azureARMObject(payload["identity"])["type"]); identity != "SystemAssigned" && identity != "None" {
		return "identity.type must be SystemAssigned or None."
	}
	if strings.TrimSpace(azureARMString(azureARMObject(payload["properties"])["agentPublicKeyCertificate"])) == "" {
		return "properties.agentPublicKeyCertificate is required."
	}
	return azureEnumProperty("privateLinkState", "Enabled", "Disabled")(payload)
}

// azureConnectedClusterProperties reports the cluster as connected and fills
// in the distribution and infrastructure defaults.
func azureConnectedClusterProperties(rec *azureARMResource) map[string]any {
	props := map[string]any{
		"connectivityStatus":   "Connected",
		"lastConnectivityTime": rec.UpdatedAt.Format(time.RFC3339),
	}
	for key, value := range map[string]string{"distribution": "generic", "infrastructure": "generic", "privateLinkState": "Disabled"} {
		if _, present := rec.Properties[key]; !present {
			props[key] = value
		}
	}
	return props
}

// azureConnectedClusterUserCredential returns a kubeconfig for the cluster.
// No API server listens at its address.
func azureConnectedClusterUserCredential(rec *azureARMResource, payload map[string]any) (int, any) {
	if method := azureARMString(payload["authenticationMethod"]); method != "Token" && method != "AAD" {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "authenticationMethod must be Token or AAD.")
	}
	kubeconfig := fmt.Sprintf(`apiVersion: v1
kind: Config
clusters:
- name: %[1]s
  cluster:
    server: https://%[1]s.arc.stackyard.local
contexts:
- name: %[1]s
  context:
    cluster: %[1]s
    user: clusterUser_%[2]s_%[1]s
current-context: %[1]s
users:
- name: clusterUser_%[2]s_%[1]s
  user:
    token: stackyard
`, rec.Name, rec.ResourceGroup)
	return http.StatusOK, map[string]any{
		"kubeconfigs": []any{map[string]any{"name": "credentialUser", "value": base64.StdEncoding.EncodeToString([]byte(kubeconfig))}},
	}
}
