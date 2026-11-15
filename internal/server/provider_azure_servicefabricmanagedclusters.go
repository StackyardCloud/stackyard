package server

import (
	"net/http"
	"regexp"
	"strings"

	"github.com/google/uuid"
)

// Microsoft.ServiceFabric managed clusters and their node types, served by
// the shared ARM handler. Clusters report Ready at once; no Service Fabric
// runtime or VM scale sets run.
func init() {
	const ns = "Microsoft.ServiceFabric"
	registerAzureARMResourceType(azureARMResourceType{
		Namespace:           ns,
		Type:                "managedClusters",
		ReadOnlyProperties:  azureSFManagedClusterProperties,
		Validate:            azureSFManagedClusterValidate,
		WriteOnlyProperties: []string{"adminPassword"},
	})
	nodes := func(_ *azureARMResource, payload map[string]any) (int, any) {
		if list, _ := payload["nodes"].([]any); len(list) == 0 {
			return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "nodes must list at least one node.")
		}
		return http.StatusAccepted, nil
	}
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "managedClusters/nodeTypes", Proxy: true,
		ReadOnlyProperties: azureSFNodeTypeProperties,
		Validate:           azureSFNodeTypeValidate,
		Actions:            map[string]azureARMAction{"restart": nodes, "reimage": nodes, "deletenode": nodes},
	})
}

var azureSFDNSNamePattern = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?$`)

func azureSFManagedClusterValidate(payload map[string]any) string {
	if sku := azureARMString(azureARMObject(payload["sku"])["name"]); sku != "Basic" && sku != "Standard" {
		return "sku.name must be Basic or Standard."
	}
	props := azureARMObject(payload["properties"])
	if !azureSFDNSNamePattern.MatchString(azureARMString(props["dnsName"])) {
		return "properties.dnsName must be 1-63 letters, digits or hyphens."
	}
	if strings.TrimSpace(azureARMString(props["adminUserName"])) == "" {
		return "properties.adminUserName is required."
	}
	for _, key := range []string{"clientConnectionPort", "httpGatewayConnectionPort"} {
		if raw, present := props[key]; present {
			if port, ok := raw.(float64); !ok || port != float64(int(port)) || port < 1 || port > 65535 {
				return "properties." + key + " must be an integer from 1 to 65535."
			}
		}
	}
	return azureEnumProperty("clusterUpgradeMode", "Automatic", "Manual")(payload)
}

// azureSFManagedClusterProperties reports the cluster as ready at its
// regional FQDN, with the port and upgrade defaults.
func azureSFManagedClusterProperties(rec *azureARMResource) map[string]any {
	props := map[string]any{
		"clusterId":                     uuid.NewSHA1(uuid.NameSpaceURL, []byte(azureARMKey(rec.ID))).String(),
		"clusterState":                  "Ready",
		"fqdn":                          strings.ToLower(azureARMString(rec.Properties["dnsName"])) + "." + rec.Location + ".cloudapp.azure.com",
		"clusterCertificateThumbprints": []any{},
	}
	defaults := map[string]any{"clientConnectionPort": 19000, "httpGatewayConnectionPort": 19080, "clusterUpgradeMode": "Automatic", "clusterUpgradeCadence": "Wave0"}
	for key, value := range defaults {
		if _, present := rec.Properties[key]; !present {
			props[key] = value
		}
	}
	return props
}

func azureSFNodeTypeValidate(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	if _, ok := props["isPrimary"].(bool); !ok {
		return "properties.isPrimary is required."
	}
	count, ok := props["vmInstanceCount"].(float64)
	if !ok || count != float64(int(count)) || count < -1 {
		return "properties.vmInstanceCount must be an integer of at least -1."
	}
	if raw, present := props["dataDiskSizeGB"]; present {
		if size, ok := raw.(float64); !ok || size != float64(int(size)) || size < 1 {
			return "properties.dataDiskSizeGB must be a positive integer."
		}
	}
	return azureEnumProperty("dataDiskType", "Standard_LRS", "StandardSSD_LRS", "Premium_LRS", "PremiumV2_LRS", "StandardSSD_ZRS", "Premium_ZRS")(payload)
}

func azureSFNodeTypeProperties(rec *azureARMResource) map[string]any {
	props := map[string]any{}
	for key, value := range map[string]any{"vmSize": "Standard_D2s_v3", "dataDiskType": "StandardSSD_LRS", "dataDiskSizeGB": 256} {
		if _, present := rec.Properties[key]; !present {
			props[key] = value
		}
	}
	return props
}
