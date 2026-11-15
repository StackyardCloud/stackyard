package server

import (
	"net/http"
	"path"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Microsoft.Logic workflows (Consumption Logic Apps) and integration accounts
// with their schemas and maps, served by the shared ARM handler. Workflows
// are stored but never run.
func init() {
	const ns = "Microsoft.Logic"
	registerAzureARMResourceType(azureARMResourceType{
		Namespace:          ns,
		Type:               "workflows",
		ReadOnlyProperties: azureLogicWorkflowProperties,
		Validate:           azureLogicValidateWorkflow,
		Transitions:        map[string]map[string]any{"enable": {"state": "Enabled"}, "disable": {"state": "Disabled"}},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace:          ns,
		Type:               "integrationAccounts",
		ReadOnlyProperties: azureLogicTimestamps,
		Validate: func(payload map[string]any) string {
			if sku := azureARMString(azureARMObject(payload["sku"])["name"]); sku != "Free" && sku != "Basic" && sku != "Standard" {
				return "sku.name must be Free, Basic or Standard."
			}
			return azureEnumProperty("state", "Enabled", "Disabled")(payload)
		},
		Actions: map[string]azureARMAction{"listcallbackurl": azureLogicIntegrationAccountCallbackURL},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "integrationAccounts/schemas", Proxy: true,
		ReadOnlyProperties: azureLogicArtifactProperties("schemas"),
		Validate: func(payload map[string]any) string {
			if azureARMString(azureARMObject(payload["properties"])["schemaType"]) != "Xml" {
				return "properties.schemaType must be Xml."
			}
			return ""
		},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "integrationAccounts/maps", Proxy: true,
		ReadOnlyProperties: azureLogicArtifactProperties("maps"),
		Validate: func(payload map[string]any) string {
			kinds := []string{"Xslt", "Xslt20", "Xslt30", "Liquid"}
			if !slices.Contains(kinds, azureARMString(azureARMObject(payload["properties"])["mapType"])) {
				return "properties.mapType must be one of " + strings.Join(kinds, ", ") + "."
			}
			return ""
		},
	})
}

func azureLogicValidateWorkflow(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	if msg := azureEnumProperty("state", "Enabled", "Disabled")(payload); msg != "" {
		return msg
	}
	raw, present := props["definition"]
	if !present {
		return ""
	}
	definition, ok := raw.(map[string]any)
	if !ok {
		return "properties.definition must be an object."
	}
	for _, key := range []string{"triggers", "actions", "parameters", "outputs"} {
		if value, present := definition[key]; present {
			if _, ok := value.(map[string]any); !ok {
				return "properties.definition." + key + " must be an object."
			}
		}
	}
	return ""
}

func azureLogicTimestamps(rec *azureARMResource) map[string]any {
	return map[string]any{
		"createdTime": rec.CreatedAt.Format(time.RFC3339),
		"changedTime": rec.UpdatedAt.Format(time.RFC3339),
	}
}

// azureLogicWorkflowProperties adds the workflow's access endpoint and
// version, and defaults the state to Enabled.
func azureLogicWorkflowProperties(rec *azureARMResource) map[string]any {
	workflowID := strings.ReplaceAll(uuid.NewSHA1(uuid.NameSpaceURL, []byte(azureARMKey(rec.ID))).String(), "-", "")
	props := azureLogicTimestamps(rec)
	props["accessEndpoint"] = "https://prod-00." + rec.Location + ".logic.azure.com:443/workflows/" + workflowID
	props["version"] = rec.UpdatedAt.Format("20060102150405")
	if _, present := rec.Properties["state"]; !present {
		props["state"] = "Enabled"
	}
	return props
}

// azureLogicArtifactProperties describes the stored content of a schema or
// map with a content link. Schemas and maps carry no location, so the link
// has no regional host.
func azureLogicArtifactProperties(collection string) func(rec *azureARMResource) map[string]any {
	return func(rec *azureARMResource) map[string]any {
		account := path.Base(path.Dir(path.Dir(rec.ID)))
		props := azureLogicTimestamps(rec)
		props["contentLink"] = map[string]any{
			"uri":         "https://prod-00.logic.azure.com:443/integrationAccounts/" + account + "/" + collection + "/" + rec.Name + "/contents/Value",
			"contentSize": len(azureARMString(rec.Properties["content"])),
		}
		return props
	}
}

// azureLogicIntegrationAccountCallbackURL returns a callback URL signed with
// a key derived from the account.
func azureLogicIntegrationAccountCallbackURL(rec *azureARMResource, _ map[string]any) (int, any) {
	sig := strings.NewReplacer("+", "-", "/", "_", "=", "").Replace(azureARMDerivedKey(rec.ID, "callback"))
	return http.StatusOK, map[string]any{
		"value": "https://prod-00." + rec.Location + ".logic.azure.com:443/integrationAccounts/" + rec.Name + "?api-version=2015-08-01-preview&sp=%2F%2F%2A&sv=1.0&sig=" + sig,
	}
}
