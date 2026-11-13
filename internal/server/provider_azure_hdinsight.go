package server

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
)

const azureHDInsightClusterType = "Microsoft.HDInsight/clusters"

// Microsoft.HDInsight clusters, served by the shared ARM handler. Clusters
// report Running at once and run no Hadoop, Spark or Kafka services.
func init() {
	registerAzureARMResourceType(azureARMResourceType{
		Namespace:          "Microsoft.HDInsight",
		Type:               "clusters",
		ReadOnlyProperties: azureHDInsightClusterProperties,
		ValidateName:       azureHDInsightValidateName,
		Validate:           azureHDInsightValidateCluster,
		// Credentials are never returned; the gateway's live in configurations.
		WriteOnlyProperties: []string{
			"clusterDefinition.configurations",
			"computeProfile.roles[].osProfile.linuxOperatingSystemProfile.password",
			"securityProfile.domainUserPassword",
			"storageProfile.storageaccounts[].key",
		},
	})
	registerAzureARMProviderAction("Microsoft.HDInsight", "locations/checkNameAvailability", func(s *Server, payload map[string]any) (int, any) {
		return s.azureIoTNameAvailability(payload, azureHDInsightClusterType, azureHDInsightValidateName)
	})
}

var (
	azureHDInsightNamePattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9-]{1,57}[a-zA-Z0-9]$`)
	azureHDInsightKinds       = []string{"hadoop", "hbase", "spark", "kafka", "interactivehive"}
	azureHDInsightRoles       = []string{"headnode", "workernode", "zookeepernode", "edgenode"}
)

func azureHDInsightValidateName(name string) string {
	if !azureHDInsightNamePattern.MatchString(name) {
		return "Cluster names must be 3-59 letters, digits or hyphens, start with a letter and end with a letter or digit."
	}
	return ""
}

func azureHDInsightValidateCluster(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	if kind := strings.ToLower(azureARMString(azureARMObject(props["clusterDefinition"])["kind"])); !slices.Contains(azureHDInsightKinds, kind) {
		return "properties.clusterDefinition.kind must be one of " + strings.Join(azureHDInsightKinds, ", ") + "."
	}
	if msg := azureEnumProperty("tier", "Standard", "Premium")(payload); msg != "" {
		return msg
	}
	if msg := azureEnumProperty("osType", "Linux")(payload); msg != "" {
		return msg
	}
	roles, _ := azureARMObject(props["computeProfile"])["roles"].([]any)
	seen := map[string]bool{}
	for i, raw := range roles {
		role := azureARMObject(raw)
		name := azureARMString(role["name"])
		if !slices.Contains(azureHDInsightRoles, name) {
			return fmt.Sprintf("properties.computeProfile.roles[%d].name must be one of %s.", i, strings.Join(azureHDInsightRoles, ", "))
		}
		if n, ok := role["targetInstanceCount"].(float64); !ok || n < 1 || n != float64(int(n)) {
			return fmt.Sprintf("properties.computeProfile.roles[%d].targetInstanceCount must be a positive integer.", i)
		}
		seen[name] = true
	}
	if !seen["headnode"] || !seen["workernode"] {
		return "properties.computeProfile.roles must include headnode and workernode."
	}
	return ""
}

// azureHDInsightClusterProperties sets the state, endpoints, ID and
// version and tier defaults.
func azureHDInsightClusterProperties(rec *azureARMResource) map[string]any {
	host := strings.ToLower(rec.Name)
	props := map[string]any{
		"clusterState": "Running",
		"clusterId":    strings.ReplaceAll(uuid.NewSHA1(uuid.NameSpaceURL, []byte(azureARMKey(rec.ID))).String(), "-", ""),
		"createdDate":  rec.CreatedAt.Format(time.RFC3339),
		"connectivityEndpoints": []any{
			map[string]any{"name": "HTTPS", "protocol": "TCP", "location": host + ".azurehdinsight.net", "port": 443},
			map[string]any{"name": "SSH", "protocol": "TCP", "location": host + "-ssh.azurehdinsight.net", "port": 22},
		},
	}
	if _, present := rec.Properties["clusterVersion"]; !present {
		props["clusterVersion"] = "5.1"
	}
	if _, present := rec.Properties["tier"]; !present {
		props["tier"] = "Standard"
	}
	return props
}
