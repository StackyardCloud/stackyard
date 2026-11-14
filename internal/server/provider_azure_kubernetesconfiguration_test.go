package server

import (
	"net/http"
	"testing"
)

const azureKubernetesConfigurationTestAPIVersion = "?api-version=2023-05-01"

func TestAzureKubernetesConfigurationLifecycle(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-arc")
	cluster := sub + "/resourceGroups/rg-arc/providers/Microsoft.Kubernetes/connectedClusters/edge"
	azureResourcesExpectStatus(t, do(http.MethodPut, cluster+azureConnectedClusterTestAPIVersion, azureConnectedClusterTestBody), http.StatusCreated)
	config := cluster + "/providers/Microsoft.KubernetesConfiguration"

	resp := do(http.MethodPut, config+"/extensions/flux"+azureKubernetesConfigurationTestAPIVersion, `{"properties":{"extensionType":"microsoft.flux","scope":{"cluster":{"releaseNamespace":"flux-system"}},"configurationSettings":{"multiTenancy.enforce":"false"},"configurationProtectedSettings":{"token":"secret"}}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	props := providerContractJSONMap(t, resp)["properties"].(map[string]any)
	if props["configurationProtectedSettings"] != nil || props["currentVersion"] != "1.0.0" || props["autoUpgradeMinorVersion"] != true || props["releaseTrain"] != "Stable" {
		t.Fatalf("unexpected extension properties: %#v", props)
	}

	flux := config + "/fluxConfigurations/apps"
	resp = do(http.MethodPut, flux+azureKubernetesConfigurationTestAPIVersion, `{"properties":{"namespace":"cluster-config","gitRepository":{"url":"https://github.com/Azure/gitops-flux2-kustomize-helm-mt","repositoryRef":{"branch":"main"}},"kustomizations":{"infra":{"path":"./infrastructure"}},"configurationProtectedSettings":{"sshPrivateKey":"secret"}}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	props = providerContractJSONMap(t, resp)["properties"].(map[string]any)
	if props["complianceState"] != "Compliant" || props["scope"] != "cluster" || props["sourceKind"] != "GitRepository" || props["configurationProtectedSettings"] != nil {
		t.Fatalf("unexpected flux properties: %#v", props)
	}
	resp = do(http.MethodPatch, flux+azureKubernetesConfigurationTestAPIVersion, `{"properties":{"suspend":true}}`)
	if got := providerContractJSONMap(t, resp)["properties"].(map[string]any)["suspend"]; got != true {
		t.Fatalf("expected suspend patched, got %v", got)
	}
	resp = do(http.MethodGet, config+"/fluxConfigurations"+azureKubernetesConfigurationTestAPIVersion, "")
	if got := providerContractJSONMap(t, resp)["value"].([]any); len(got) != 1 {
		t.Fatalf("expected one flux configuration, got %d", len(got))
	}

	blob := config + "/fluxConfigurations/blob"
	resp = do(http.MethodPut, blob+azureKubernetesConfigurationTestAPIVersion, `{"properties":{"sourceKind":"AzureBlob","azureBlob":{"url":"https://lake.blob.core.windows.net","containerName":"manifests","sasToken":"secret","servicePrincipal":{"clientId":"app","clientSecret":"secret"}}}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	source := providerContractJSONMap(t, resp)["properties"].(map[string]any)["azureBlob"].(map[string]any)
	if source["sasToken"] != nil || source["servicePrincipal"].(map[string]any)["clientSecret"] != nil || source["containerName"] != "manifests" {
		t.Fatalf("expected blob secrets stripped, got %#v", source)
	}

	// Deleting the cluster deletes what is attached to it.
	azureResourcesExpectStatus(t, do(http.MethodDelete, cluster+azureConnectedClusterTestAPIVersion, ""), http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodGet, flux+azureKubernetesConfigurationTestAPIVersion, ""), http.StatusNotFound)
}

func TestAzureKubernetesConfigurationValidation(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-arc")
	config := sub + "/resourceGroups/rg-arc/providers/Microsoft.ContainerService/managedClusters/aks/providers/Microsoft.KubernetesConfiguration"
	cases := []struct{ name, path, body string }{
		{"no extension type", config + "/extensions/x", `{"properties":{}}`},
		{"both scopes", config + "/extensions/x", `{"properties":{"extensionType":"microsoft.flux","scope":{"cluster":{},"namespace":{}}}}`},
		{"no git url", config + "/fluxConfigurations/x", `{"properties":{"gitRepository":{}}}`},
		{"bad source kind", config + "/fluxConfigurations/x", `{"properties":{"sourceKind":"Svn"}}`},
		{"bucket without url", config + "/fluxConfigurations/x", `{"properties":{"sourceKind":"Bucket","gitRepository":{"url":"https://example.com"}}}`},
		{"kustomizations not an object", config + "/fluxConfigurations/x", `{"properties":{"gitRepository":{"url":"https://example.com"},"kustomizations":[]}}`},
		{"bad scope", config + "/fluxConfigurations/x", `{"properties":{"scope":"tenant","gitRepository":{"url":"https://example.com"}}}`},
	}
	for _, tc := range cases {
		resp := do(http.MethodPut, tc.path+azureKubernetesConfigurationTestAPIVersion, tc.body)
		t.Run(tc.name, func(t *testing.T) {
			azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, "InvalidRequestContent")
		})
	}

	// AKS clusters and Arc resource bridges are valid scopes too.
	for _, scope := range []string{config, sub + "/resourceGroups/rg-arc/providers/Microsoft.ResourceConnector/appliances/bridge/providers/Microsoft.KubernetesConfiguration"} {
		azureResourcesExpectStatus(t, do(http.MethodPut, scope+"/extensions/x"+azureKubernetesConfigurationTestAPIVersion, `{"properties":{"extensionType":"microsoft.flux"}}`), http.StatusCreated)
	}

	// Configurations attach only to clusters.
	for _, scope := range []string{sub, sub + "/resourceGroups/rg-arc", sub + "/resourceGroups/rg-arc/providers/Microsoft.Compute/virtualMachines/vm"} {
		azureResourcesExpectErrorCode(t, do(http.MethodPut, scope+"/providers/Microsoft.KubernetesConfiguration/extensions/x"+azureKubernetesConfigurationTestAPIVersion, `{"properties":{"extensionType":"microsoft.flux"}}`), http.StatusBadRequest, "InvalidScope")
	}
}
