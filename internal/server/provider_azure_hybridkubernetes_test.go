package server

import (
	"encoding/base64"
	"net/http"
	"strings"
	"testing"
)

const azureConnectedClusterTestAPIVersion = "?api-version=2024-01-01"

const azureConnectedClusterTestBody = `{"location":"eastus","identity":{"type":"SystemAssigned"},"properties":{"agentPublicKeyCertificate":"MIICYzCCAcygAwIBAgIBADANBgkqhkiG9w0BAQUFADAu","connectivityStatus":"Offline"}}`

func TestAzureConnectedClusterLifecycle(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-arc")
	cluster := sub + "/resourceGroups/rg-arc/providers/Microsoft.Kubernetes/connectedClusters/edge"

	resp := do(http.MethodPut, cluster+azureConnectedClusterTestAPIVersion, azureConnectedClusterTestBody)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	props := providerContractJSONMap(t, resp)["properties"].(map[string]any)
	if props["connectivityStatus"] != "Connected" || props["distribution"] != "generic" || props["privateLinkState"] != "Disabled" {
		t.Fatalf("unexpected cluster properties: %#v", props)
	}

	resp = do(http.MethodPatch, cluster+azureConnectedClusterTestAPIVersion, `{"properties":{"connectivityStatus":"Offline"}}`)
	if got := providerContractJSONMap(t, resp)["properties"].(map[string]any)["connectivityStatus"]; got != "Connected" {
		t.Fatalf("a PATCH must not set connectivityStatus, got %v", got)
	}

	resp = do(http.MethodPost, cluster+"/listClusterUserCredential"+azureConnectedClusterTestAPIVersion, `{"authenticationMethod":"Token","clientProxy":false}`)
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	kubeconfigs := providerContractJSONMap(t, resp)["kubeconfigs"].([]any)
	raw, err := base64.StdEncoding.DecodeString(kubeconfigs[0].(map[string]any)["value"].(string))
	if err != nil || !strings.Contains(string(raw), "server: https://edge.") {
		t.Fatalf("unexpected kubeconfig %q: %v", raw, err)
	}
	azureResourcesExpectErrorCode(t, do(http.MethodPost, cluster+"/listClusterUserCredential"+azureConnectedClusterTestAPIVersion, `{"authenticationMethod":"Password"}`), http.StatusBadRequest, "InvalidRequestContent")

	azureResourcesExpectStatus(t, do(http.MethodDelete, cluster+azureConnectedClusterTestAPIVersion, ""), http.StatusOK)
}

func TestAzureConnectedClusterValidation(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-arc")
	cluster := sub + "/resourceGroups/rg-arc/providers/Microsoft.Kubernetes/connectedClusters/edge"
	cases := []struct{ name, body string }{
		{"no identity", `{"location":"eastus","properties":{"agentPublicKeyCertificate":"abc"}}`},
		{"no certificate", `{"location":"eastus","identity":{"type":"SystemAssigned"},"properties":{}}`},
		{"bad private link state", `{"location":"eastus","identity":{"type":"SystemAssigned"},"properties":{"agentPublicKeyCertificate":"abc","privateLinkState":"On"}}`},
	}
	for _, tc := range cases {
		resp := do(http.MethodPut, cluster+azureConnectedClusterTestAPIVersion, tc.body)
		t.Run(tc.name, func(t *testing.T) {
			azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, "InvalidRequestContent")
		})
	}
}
