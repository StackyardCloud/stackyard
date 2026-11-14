package server

import (
	"slices"
	"strings"
	"time"
)

// Microsoft.KubernetesConfiguration cluster extensions and Flux (GitOps)
// configurations, served by the shared ARM handler as extension resources on
// AKS and Arc-enabled clusters. Nothing is installed and no repository is
// synced: extensions report Succeeded and Flux configurations Compliant.
func init() {
	const ns = "Microsoft.KubernetesConfiguration"
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "extensions", Extension: true,
		ValidateScope:       azureKubernetesConfigurationScope,
		Validate:            azureKubernetesExtensionValidate,
		ReadOnlyProperties:  azureKubernetesExtensionProperties,
		WriteOnlyProperties: []string{"configurationProtectedSettings"},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "fluxConfigurations", Extension: true,
		ValidateScope:      azureKubernetesConfigurationScope,
		Validate:           azureFluxConfigurationValidate,
		ReadOnlyProperties: azureFluxConfigurationProperties,
		WriteOnlyProperties: []string{
			"configurationProtectedSettings",
			"azureBlob.accountKey",
			"azureBlob.sasToken",
			"azureBlob.servicePrincipal.clientSecret",
			"azureBlob.servicePrincipal.clientCertificate",
			"azureBlob.servicePrincipal.clientCertificatePassword",
		},
	})
}

var azureKubernetesConfigurationClusterTypes = []string{
	"microsoft.containerservice/managedclusters",
	"microsoft.kubernetes/connectedclusters",
	"microsoft.hybridcontainerservice/provisionedclusters",
	"microsoft.resourceconnector/appliances",
}

// azureKubernetesConfigurationScope accepts only a cluster resource group
// scope: .../providers/{clusterRp}/{clusterResourceName}/{clusterName}.
func azureKubernetesConfigurationScope(scope string) string {
	seg := strings.Split(strings.Trim(scope, "/"), "/")
	n := len(seg)
	if n == 8 && strings.EqualFold(seg[2], "resourceGroups") && strings.EqualFold(seg[4], "providers") &&
		slices.Contains(azureKubernetesConfigurationClusterTypes, strings.ToLower(seg[5]+"/"+seg[6])) {
		return ""
	}
	return "Kubernetes configuration resources attach only to managedClusters, connectedClusters, provisionedClusters or appliances."
}

func azureKubernetesExtensionValidate(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	if strings.TrimSpace(azureARMString(props["extensionType"])) == "" {
		return "properties.extensionType is required."
	}
	scope := azureARMObject(props["scope"])
	if _, cluster := scope["cluster"]; cluster {
		if _, namespace := scope["namespace"]; namespace {
			return "properties.scope must set only one of cluster or namespace."
		}
	}
	return ""
}

// azureKubernetesExtensionProperties reports the requested version, or a
// fixed one, as installed, with the auto-upgrade and release train defaults.
func azureKubernetesExtensionProperties(rec *azureARMResource) map[string]any {
	version := azureARMString(rec.Properties["version"])
	if version == "" {
		version = "1.0.0"
	}
	props := map[string]any{"currentVersion": version, "isSystemExtension": false}
	if _, present := rec.Properties["autoUpgradeMinorVersion"]; !present {
		props["autoUpgradeMinorVersion"] = true
	}
	if _, present := rec.Properties["releaseTrain"]; !present {
		props["releaseTrain"] = "Stable"
	}
	return props
}

var azureFluxSourceKinds = map[string]string{
	"GitRepository": "gitRepository",
	"Bucket":        "bucket",
	"AzureBlob":     "azureBlob",
	"OCIRepository": "ociRepository",
}

func azureFluxConfigurationValidate(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	if msg := azureEnumProperty("scope", "cluster", "namespace")(payload); msg != "" {
		return msg
	}
	kind := azureARMString(props["sourceKind"])
	if kind == "" {
		kind = "GitRepository"
	}
	source, ok := azureFluxSourceKinds[kind]
	if !ok {
		return "properties.sourceKind must be GitRepository, Bucket, AzureBlob or OCIRepository."
	}
	if strings.TrimSpace(azureARMString(azureARMObject(props[source])["url"])) == "" {
		return "properties." + source + ".url is required."
	}
	kustomizations, ok := props["kustomizations"].(map[string]any)
	if _, present := props["kustomizations"]; present && !ok {
		return "properties.kustomizations must be an object."
	}
	for name, raw := range kustomizations {
		if _, ok := raw.(map[string]any); !ok {
			return "properties.kustomizations." + name + " must be an object."
		}
	}
	return ""
}

// azureFluxConfigurationProperties reports the configuration as compliant,
// with the scope, namespace and source kind defaults.
func azureFluxConfigurationProperties(rec *azureARMResource) map[string]any {
	props := map[string]any{
		"complianceState": "Compliant",
		"sourceUpdatedAt": rec.UpdatedAt.Format(time.RFC3339),
		"statuses":        []any{},
	}
	for key, value := range map[string]string{"scope": "cluster", "namespace": "default", "sourceKind": "GitRepository"} {
		if _, present := rec.Properties[key]; !present {
			props[key] = value
		}
	}
	return props
}
