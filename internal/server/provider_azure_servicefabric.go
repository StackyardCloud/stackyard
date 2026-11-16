package server

import (
	"net/http"
	"slices"
	"strings"

	"github.com/google/uuid"
)

// azureSFClusterCodeVersion is the runtime version clusters report when the
// client sets none.
const azureSFClusterCodeVersion = "10.1.2493.9590"

// Microsoft.ServiceFabric classic clusters with their application types,
// versions, applications and services, served by the shared ARM handler.
// Clusters report Ready at once; no Service Fabric runtime runs and no
// application package is downloaded.
func init() {
	const ns = "Microsoft.ServiceFabric"
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "clusters",
		Validate:           azureSFClusterValidate,
		ReadOnlyProperties: azureSFClusterProperties,
		Actions: map[string]azureARMAction{
			"listupgradableversions": func(*azureARMResource, map[string]any) (int, any) {
				return http.StatusOK, map[string]any{"supportedPath": []any{azureSFClusterCodeVersion}}
			},
		},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "clusters/applicationTypes", Proxy: true,
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "clusters/applicationTypes/versions", Proxy: true,
		Validate: func(payload map[string]any) string {
			url := strings.ToLower(azureARMString(azureARMObject(payload["properties"])["appPackageUrl"]))
			if !strings.HasPrefix(url, "https://") && !strings.HasPrefix(url, "http://") {
				return "properties.appPackageUrl must be an http or https URL."
			}
			return ""
		},
		ReadOnlyProperties: func(*azureARMResource) map[string]any {
			return map[string]any{"defaultParameterList": map[string]any{}}
		},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "clusters/applications", Proxy: true,
		Validate: func(payload map[string]any) string {
			props := azureARMObject(payload["properties"])
			for _, key := range []string{"typeName", "typeVersion"} {
				if strings.TrimSpace(azureARMString(props[key])) == "" {
					return "properties." + key + " is required."
				}
			}
			return ""
		},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "clusters/applications/services", Proxy: true,
		Validate: azureSFServiceValidate,
	})
}

var azureSFReliabilityLevels = []string{"None", "Bronze", "Silver", "Gold", "Platinum"}

func azureSFClusterValidate(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	endpoint := strings.ToLower(azureARMString(props["managementEndpoint"]))
	if !strings.HasPrefix(endpoint, "https://") && !strings.HasPrefix(endpoint, "http://") {
		return "properties.managementEndpoint must be an http or https URL."
	}
	nodeTypes, _ := props["nodeTypes"].([]any)
	if len(nodeTypes) == 0 {
		return "properties.nodeTypes must list at least one node type."
	}
	primary := false
	for _, raw := range nodeTypes {
		nodeType := azureARMObject(raw)
		if strings.TrimSpace(azureARMString(nodeType["name"])) == "" {
			return "Each node type needs a name."
		}
		for _, key := range []string{"clientConnectionEndpointPort", "httpGatewayEndpointPort"} {
			if port, ok := nodeType[key].(float64); !ok || port != float64(int(port)) || port < 1 || port > 65535 {
				return "Node type " + key + " must be an integer from 1 to 65535."
			}
		}
		if count, ok := nodeType["vmInstanceCount"].(float64); !ok || count != float64(int(count)) || count < 1 {
			return "Node type vmInstanceCount must be a positive integer."
		}
		isPrimary, ok := nodeType["isPrimary"].(bool)
		if !ok {
			return "Node type isPrimary is required."
		}
		primary = primary || isPrimary
	}
	if !primary {
		return "properties.nodeTypes must include a primary node type."
	}
	if msg := azureEnumProperty("reliabilityLevel", azureSFReliabilityLevels...)(payload); msg != "" {
		return msg
	}
	return azureEnumProperty("upgradeMode", "Automatic", "Manual")(payload)
}

// azureSFClusterProperties reports the cluster as ready at its regional
// runtime endpoint, on the default runtime version unless the client pinned
// one.
func azureSFClusterProperties(rec *azureARMResource) map[string]any {
	clusterID := uuid.NewSHA1(uuid.NameSpaceURL, []byte(azureARMKey(rec.ID))).String()
	props := map[string]any{
		"clusterId":                clusterID,
		"clusterState":             "Ready",
		"clusterEndpoint":          "https://" + azureLocationHost(rec.Location) + ".servicefabric.azure.com/runtime/clusters/" + clusterID,
		"availableClusterVersions": []any{map[string]any{"codeVersion": azureSFClusterCodeVersion, "environment": "Windows"}},
	}
	for key, value := range map[string]any{"clusterCodeVersion": azureSFClusterCodeVersion, "upgradeMode": "Automatic", "reliabilityLevel": "Silver"} {
		if _, present := rec.Properties[key]; !present {
			props[key] = value
		}
	}
	return props
}

func azureSFServiceValidate(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	if strings.TrimSpace(azureARMString(props["serviceTypeName"])) == "" {
		return "properties.serviceTypeName is required."
	}
	scheme := azureARMString(azureARMObject(props["partitionDescription"])["partitionScheme"])
	if !slices.Contains([]string{"Singleton", "UniformInt64Range", "Named"}, scheme) {
		return "properties.partitionDescription.partitionScheme must be Singleton, UniformInt64Range or Named."
	}
	integer := func(key string, least float64) bool {
		raw, present := props[key]
		if !present {
			return true
		}
		n, ok := raw.(float64)
		return ok && n == float64(int(n)) && n >= least
	}
	switch azureARMString(props["serviceKind"]) {
	case "Stateless":
		if !integer("instanceCount", -1) {
			return "properties.instanceCount must be an integer of at least -1."
		}
	case "Stateful":
		if !integer("targetReplicaSetSize", 1) || !integer("minReplicaSetSize", 1) {
			return "properties.targetReplicaSetSize and minReplicaSetSize must be positive integers."
		}
	default:
		return "properties.serviceKind must be Stateless or Stateful."
	}
	return ""
}
