package server

import (
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Microsoft.HybridCompute Arc-enabled servers and their VM extensions, served
// by the shared ARM handler. Machines report Connected at once; no Connected
// Machine agent runs and extensions install nothing.
func init() {
	const ns = "Microsoft.HybridCompute"
	registerAzureARMResourceType(azureARMResourceType{
		Namespace:          ns,
		Type:               "machines",
		ReadOnlyProperties: azureHybridComputeMachineProperties,
		Validate:           azureHybridComputeValidateMachine,
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "machines/extensions",
		Validate: azureHybridComputeValidateExtension,
		// Protected settings hold secrets and are never returned.
		WriteOnlyProperties: []string{"protectedSettings"},
	})
}

var azureHybridComputeMachineKinds = []string{"AVS", "HCI", "SCVMM", "VMware", "EPS", "GCP", "AWS"}

func azureHybridComputeValidateMachine(payload map[string]any) string {
	if kind, present := payload["kind"]; present && !slices.Contains(azureHybridComputeMachineKinds, azureARMString(kind)) {
		return "kind must be one of " + strings.Join(azureHybridComputeMachineKinds, ", ") + "."
	}
	if os, present := azureARMObject(payload["properties"])["osType"]; present {
		if lower := strings.ToLower(azureARMString(os)); lower != "windows" && lower != "linux" {
			return "properties.osType must be Windows or Linux."
		}
	}
	return ""
}

func azureHybridComputeValidateExtension(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	for _, key := range []string{"publisher", "type"} {
		if strings.TrimSpace(azureARMString(props[key])) == "" {
			return "properties." + key + " is required."
		}
	}
	return ""
}

// azureHybridComputeMachineProperties reports the machine as connected, with
// a stable VM ID.
func azureHybridComputeMachineProperties(rec *azureARMResource) map[string]any {
	return map[string]any{
		"status":           "Connected",
		"vmId":             uuid.NewSHA1(uuid.NameSpaceURL, []byte(azureARMKey(rec.ID))).String(),
		"lastStatusChange": rec.CreatedAt.Format(time.RFC3339),
	}
}
