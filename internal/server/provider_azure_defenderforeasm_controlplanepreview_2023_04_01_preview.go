package server

import (
	"net/http"
	"regexp"
	"strings"
)

const (
	azureDefenderForEASMControlPlanePreview20230401APIVersion = "2023-04-01-preview"
	azureEASMCNamespace                                       = "Microsoft.Easm"
)

// Defender External Attack Surface Management control plane
// (Microsoft.Easm 2023-04-01-preview) on the shared ARM handler: workspaces,
// their labels, task lookups and the operations list. Workspace tasks are
// the data-plane tasks (provider_azure_defenderforeasm_dataplanepreview_*),
// which emulate a single workspace.
var (
	azureEASMCNameRE   = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9_-]{0,61}[a-zA-Z0-9])?$`)
	azureEASMCColorsRE = regexp.MustCompile(`^[A-Za-z]+$|^#[0-9A-Fa-f]{6}$`)
)

func init() {
	ns := azureEASMCNamespace
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "workspaces",
		ValidateName: azureEASMCValidateName,
		DeleteStatus: http.StatusAccepted,
		ReadOnlyProperties: func(rec *azureARMResource) map[string]any {
			return map[string]any{"dataPlaneEndpoint": "https://" + rec.Location + ".easm.defender.microsoft.com/subscriptions/" + rec.SubscriptionID + "/resourceGroups/" + rec.ResourceGroup + "/workspaces/" + rec.Name}
		},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "workspaces/labels", Proxy: true,
		ValidateName: azureEASMCValidateName,
		Validate:     azureEASMCLabelValidate,
		DeleteStatus: http.StatusAccepted,
	})
	registerAzureARMView(ns, "workspaces/tasks", func(s *Server, scope string) []map[string]any {
		tasks := s.azureEASMDState().taskList()
		out := make([]map[string]any, 0, len(tasks))
		for _, task := range tasks {
			out = append(out, map[string]any{
				"id": scope + "/tasks/" + azureARMString(task["id"]), "name": task["id"], "type": ns + "/workspaces/tasks",
				"properties": map[string]any{
					"startedAt": task["startedAt"], "completedAt": task["completedAt"], "lastPolledAt": task["lastPolledAt"],
					"state": task["state"], "phase": task["phase"], "reason": task["reason"], "metadata": task["metadata"],
				},
			})
		}
		return out
	})
	azureDMSOperationsView(ns, "Microsoft Defender EASM", [2]string{"workspaces", "workspaces"}, [2]string{"workspaces/labels", "labels"})
}

// handleAzureDefenderForEasmControlPlanePreview20230401Router answers paths
// under the provider that the shared ARM handler did not serve.
func (s *Server) handleAzureDefenderForEasmControlPlanePreview20230401Router(w http.ResponseWriter, r *http.Request) bool {
	return azureDMSUnknownRoute(w, r, azureEASMCNamespace)
}

func azureEASMCValidateName(name string) string {
	if !azureEASMCNameRE.MatchString(name) {
		return "Names must be 1-63 letters, digits, '-' or '_', starting and ending with a letter or digit."
	}
	return ""
}

func azureEASMCLabelValidate(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	if display, present := props["displayName"]; present {
		if text, ok := display.(string); !ok || len(text) > 256 {
			return "properties.displayName must be a string of at most 256 characters."
		}
	}
	if color := azureARMString(props["color"]); props["color"] != nil && !azureEASMCColorsRE.MatchString(strings.TrimSpace(color)) {
		return "properties.color must be a color name or #RRGGBB value."
	}
	return ""
}
