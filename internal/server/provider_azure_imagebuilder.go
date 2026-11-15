package server

import (
	"fmt"
	"slices"
	"strings"
)

// Microsoft.VirtualMachineImages image templates, served by the shared ARM
// handler. A run builds and distributes nothing; it only records a
// succeeded run.
func init() {
	registerAzureARMResourceType(azureARMResourceType{
		Namespace:       "Microsoft.VirtualMachineImages",
		Type:            "imageTemplates",
		Validate:        azureImageBuilderValidateTemplate,
		StateProperties: map[string]any{"lastRunStatus": nil},
		Transitions: map[string]map[string]any{
			"run": {"lastRunStatus": map[string]any{"runState": "Succeeded", "runSubState": "Distributing", "message": ""}},
		},
	})
}

var (
	azureImageBuilderSourceTypes      = []string{"PlatformImage", "ManagedImage", "SharedImageVersion"}
	azureImageBuilderDistributeTypes  = []string{"ManagedImage", "SharedImage", "VHD"}
	azureImageBuilderCustomizerTypes  = []string{"Shell", "PowerShell", "WindowsRestart", "WindowsUpdate", "File"}
	azureImageBuilderValidationTypes  = []string{"Shell", "PowerShell", "File"}
	azureImageBuilderDistributeFields = map[string]string{"ManagedImage": "imageId", "SharedImage": "galleryImageId"}
)

func azureImageBuilderValidateTemplate(payload map[string]any) string {
	identity := azureARMObject(payload["identity"])
	if azureARMString(identity["type"]) != "UserAssigned" || len(azureARMObject(identity["userAssignedIdentities"])) != 1 {
		return "identity must be UserAssigned with exactly one user-assigned identity."
	}
	props := azureARMObject(payload["properties"])
	if !slices.Contains(azureImageBuilderSourceTypes, azureARMString(azureARMObject(props["source"])["type"])) {
		return "properties.source.type must be one of " + strings.Join(azureImageBuilderSourceTypes, ", ") + "."
	}
	distribute, _ := props["distribute"].([]any)
	if len(distribute) == 0 {
		return "properties.distribute must list at least one target."
	}
	for i, raw := range distribute {
		target := azureARMObject(raw)
		kind := azureARMString(target["type"])
		if !slices.Contains(azureImageBuilderDistributeTypes, kind) {
			return fmt.Sprintf("properties.distribute[%d].type must be one of %s.", i, strings.Join(azureImageBuilderDistributeTypes, ", "))
		}
		if azureARMString(target["runOutputName"]) == "" {
			return fmt.Sprintf("properties.distribute[%d].runOutputName is required.", i)
		}
		if field := azureImageBuilderDistributeFields[kind]; field != "" && azureARMString(target[field]) == "" {
			return fmt.Sprintf("properties.distribute[%d].%s is required.", i, field)
		}
	}
	for _, list := range []struct {
		key   string
		types []string
	}{{"customize", azureImageBuilderCustomizerTypes}, {"validate.inVMValidations", azureImageBuilderValidationTypes}} {
		raw := props[list.key]
		if key, nested, ok := strings.Cut(list.key, "."); ok {
			raw = azureARMObject(props[key])[nested]
		}
		steps, _ := raw.([]any)
		for i, step := range steps {
			if !slices.Contains(list.types, azureARMString(azureARMObject(step)["type"])) {
				return fmt.Sprintf("properties.%s[%d].type must be one of %s.", list.key, i, strings.Join(list.types, ", "))
			}
		}
	}
	if raw, present := props["buildTimeoutInMinutes"]; present {
		if n, ok := raw.(float64); !ok || n != float64(int(n)) || n < 0 || n > 960 {
			return "properties.buildTimeoutInMinutes must be an integer from 0 to 960."
		}
	}
	return ""
}
