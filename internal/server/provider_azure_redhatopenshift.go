package server

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

// Microsoft.RedHatOpenShift clusters, served by the shared ARM handler.
// Clusters report their console and API server addresses at once; no
// OpenShift control plane or workers run.
func init() {
	registerAzureARMResourceType(azureARMResourceType{
		Namespace:          "Microsoft.RedHatOpenShift",
		Type:               "openShiftClusters",
		ReadOnlyProperties: azureOpenShiftClusterProperties,
		Validate:           azureOpenShiftValidateCluster,
		Actions: map[string]azureARMAction{
			"listcredentials":      azureOpenShiftListCredentials,
			"listadmincredentials": azureOpenShiftListAdminCredentials,
		},
		// The Red Hat pull secret and the service principal secret are never returned.
		WriteOnlyProperties: []string{"clusterProfile.pullSecret", "servicePrincipalProfile.clientSecret"},
	})
}

var azureOpenShiftDomainPattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?(\.[a-z0-9]([a-z0-9-]*[a-z0-9])?)*$`)

func azureOpenShiftValidateCluster(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	cluster := azureARMObject(props["clusterProfile"])
	if !azureOpenShiftDomainPattern.MatchString(azureARMString(cluster["domain"])) {
		return "properties.clusterProfile.domain must be a lowercase DNS name."
	}
	if seg := strings.Split(strings.Trim(azureARMString(cluster["resourceGroupId"]), "/"), "/"); len(seg) != 4 || !strings.EqualFold(seg[0], "subscriptions") || !strings.EqualFold(seg[2], "resourceGroups") {
		return "properties.clusterProfile.resourceGroupId must be a resource group ID."
	}
	if strings.TrimSpace(azureARMString(azureARMObject(props["servicePrincipalProfile"])["clientId"])) == "" {
		return "properties.servicePrincipalProfile.clientId is required."
	}
	if !azureOpenShiftIsSubnet(azureARMObject(props["masterProfile"])["subnetId"]) {
		return "properties.masterProfile.subnetId must be a subnet ID."
	}
	workers, _ := props["workerProfiles"].([]any)
	if len(workers) == 0 {
		return "properties.workerProfiles must list at least one profile."
	}
	for i, raw := range workers {
		worker := azureARMObject(raw)
		if !azureOpenShiftIsSubnet(worker["subnetId"]) {
			return fmt.Sprintf("properties.workerProfiles[%d].subnetId must be a subnet ID.", i)
		}
		if count, ok := worker["count"].(float64); !ok || count != float64(int(count)) || count < 3 {
			return fmt.Sprintf("properties.workerProfiles[%d].count must be an integer of at least 3.", i)
		}
	}
	for _, path := range []string{"apiserverProfile", "ingressProfiles"} {
		profiles := []any{props[path]}
		if list, ok := props[path].([]any); ok {
			profiles = list
		}
		for _, profile := range profiles {
			if visibility, present := azureARMObject(profile)["visibility"]; present && visibility != "Public" && visibility != "Private" {
				return "properties." + path + " visibility must be Public or Private."
			}
		}
	}
	return ""
}

func azureOpenShiftIsSubnet(id any) bool {
	return strings.Contains(strings.ToLower(azureARMString(id)), "/providers/microsoft.network/virtualnetworks/") &&
		strings.Contains(strings.ToLower(azureARMString(id)), "/subnets/")
}

// azureOpenShiftClusterProperties reports the console and API server
// addresses under the cluster domain and the default version and network.
func azureOpenShiftClusterProperties(rec *azureARMResource) map[string]any {
	domain := azureARMString(azureARMObject(rec.Properties["clusterProfile"])["domain"])
	if !strings.Contains(domain, ".") {
		// A short domain is a prefix of the region's managed domain.
		domain += "." + rec.Location + ".aroapp.io"
	}
	props := map[string]any{
		"consoleProfile":   map[string]any{"url": "https://console-openshift-console.apps." + domain + "/"},
		"apiserverProfile": map[string]any{"url": "https://api." + domain + ":6443/"},
	}
	cluster := azureARMObject(rec.Properties["clusterProfile"])
	if _, present := cluster["version"]; !present {
		props["clusterProfile"] = map[string]any{"version": "4.14.16"}
	}
	network := azureARMObject(rec.Properties["networkProfile"])
	defaults := map[string]any{}
	for key, value := range map[string]any{"podCidr": "10.128.0.0/14", "serviceCidr": "172.30.0.0/16", "outboundType": "Loadbalancer"} {
		if _, present := network[key]; !present {
			defaults[key] = value
		}
	}
	if len(defaults) > 0 {
		props["networkProfile"] = defaults
	}
	return props
}

func azureOpenShiftListCredentials(rec *azureARMResource, _ map[string]any) (int, any) {
	password := strings.NewReplacer("+", "", "/", "", "=", "").Replace(azureARMDerivedKey(rec.ID, "kubeadmin"))[:23]
	return http.StatusOK, map[string]any{"kubeadminUsername": "kubeadmin", "kubeadminPassword": password}
}

// azureOpenShiftListAdminCredentials returns an admin kubeconfig for the
// cluster's API server address, at which nothing listens.
func azureOpenShiftListAdminCredentials(rec *azureARMResource, _ map[string]any) (int, any) {
	server := azureARMString(azureARMObject(azureOpenShiftClusterProperties(rec)["apiserverProfile"])["url"])
	kubeconfig := fmt.Sprintf(`apiVersion: v1
kind: Config
clusters:
- name: %[1]s
  cluster:
    server: %[2]s
contexts:
- name: admin
  context:
    cluster: %[1]s
    user: admin
current-context: admin
users:
- name: admin
  user:
    token: stackyard
`, rec.Name, strings.TrimSuffix(server, "/"))
	return http.StatusOK, map[string]any{"kubeconfig": base64.StdEncoding.EncodeToString([]byte(kubeconfig))}
}
