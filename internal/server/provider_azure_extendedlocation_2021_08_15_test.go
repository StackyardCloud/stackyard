package server

import (
	"net/http"
	"strings"
	"testing"
)

const xlocTestAPI = "?api-version=" + azureExtendedLocation20210815APIVersion

func TestAzureCustomLocationLifecycle(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-xloc")
	loc := sub + "/resourceGroups/rg-xloc/providers/Microsoft.ExtendedLocation/customLocations/cl1"
	ext := strings.TrimPrefix(sub, "/azure") + "/resourceGroups/rg-xloc/providers/Microsoft.Kubernetes/connectedClusters/c1/providers/Microsoft.KubernetesConfiguration/extensions/arcdata"
	host := strings.TrimPrefix(sub, "/azure") + "/resourceGroups/rg-xloc/providers/Microsoft.Kubernetes/connectedClusters/c1"

	resp := do(http.MethodPut, loc+xlocTestAPI, `{"location":"eastus","properties":{"hostResourceId":"`+host+`","namespace":"arc","clusterExtensionIds":["`+ext+`"],"authentication":{"type":"KubeConfig","value":"secret"}}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	props := providerContractJSONMap(t, resp)["properties"].(map[string]any)
	if props["hostType"] != "Kubernetes" || props["displayName"] != "cl1" || azureARMObject(props["authentication"])["value"] != nil {
		t.Fatalf("unexpected custom location properties: %#v", props)
	}

	resp = do(http.MethodGet, loc+"/enabledResourceTypes"+xlocTestAPI, "")
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	types := providerContractJSONMap(t, resp)["value"].([]any)
	if len(types) != 1 || types[0].(map[string]any)["properties"].(map[string]any)["extensionType"] != "arcdata" {
		t.Fatalf("unexpected enabled resource types: %#v", types)
	}
	azureResourcesExpectErrorCode(t, do(http.MethodPatch, loc+xlocTestAPI, `{"properties":{"clusterExtensionIds":[]}}`), http.StatusBadRequest, "InvalidRequestContent")
	resp = do(http.MethodPatch, loc+xlocTestAPI, `{"tags":{"tier":"test"}}`)
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodGet, "/azure/providers/Microsoft.ExtendedLocation/operations"+xlocTestAPI, ""), http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodDelete, loc+xlocTestAPI, ""), http.StatusOK)
	azureResourcesExpectErrorCode(t, do(http.MethodGet, loc+"/enabledResourceTypes"+xlocTestAPI, ""), http.StatusNotFound, "ParentResourceNotFound")
}

func TestAzureCustomLocationValidation(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-xloc")
	loc := sub + "/resourceGroups/rg-xloc/providers/Microsoft.ExtendedLocation/customLocations/cl1"
	for name, body := range map[string]string{
		"no host":          `{"location":"eastus","properties":{"namespace":"arc","clusterExtensionIds":["/subscriptions/s/resourceGroups/r/providers/Microsoft.Kubernetes/connectedClusters/c/providers/Microsoft.KubernetesConfiguration/extensions/e"]}}`,
		"no extensions":    `{"location":"eastus","properties":{"hostResourceId":"/subscriptions/s/resourceGroups/r/providers/Microsoft.Kubernetes/connectedClusters/c","namespace":"arc"}}`,
		"not an extension": `{"location":"eastus","properties":{"hostResourceId":"/subscriptions/s/resourceGroups/r/providers/Microsoft.Kubernetes/connectedClusters/c","namespace":"arc","clusterExtensionIds":["/subscriptions/s/resourceGroups/r/providers/Microsoft.Compute/virtualMachines/v"]}}`,
		"bad host type":    `{"location":"eastus","properties":{"hostType":"VMware","hostResourceId":"/subscriptions/s/resourceGroups/r/providers/Microsoft.Kubernetes/connectedClusters/c","namespace":"arc","clusterExtensionIds":["/subscriptions/s/resourceGroups/r/providers/Microsoft.Kubernetes/connectedClusters/c/providers/Microsoft.KubernetesConfiguration/extensions/e"]}}`,
	} {
		t.Run(name, func(t *testing.T) {
			azureResourcesExpectErrorCode(t, do(http.MethodPut, loc+xlocTestAPI, body), http.StatusBadRequest, "InvalidRequestContent")
		})
	}
}
