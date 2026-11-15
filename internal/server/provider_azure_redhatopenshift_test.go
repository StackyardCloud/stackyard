package server

import (
	"encoding/base64"
	"net/http"
	"strings"
	"testing"
)

const azureOpenShiftTestAPIVersion = "?api-version=2023-11-22"

// azureOpenShiftTestCluster takes the subscription request path; resource
// IDs in the body drop its /azure prefix.
func azureOpenShiftTestCluster(sub string) string {
	sub = strings.TrimPrefix(sub, "/azure")
	subnet := sub + "/resourceGroups/rg-aro/providers/Microsoft.Network/virtualNetworks/aro-vnet/subnets/"
	return `{"location":"eastus","properties":{` +
		`"clusterProfile":{"domain":"stackyard","resourceGroupId":"` + sub + `/resourceGroups/aro-infra","pullSecret":"{\"auths\":{}}"},` +
		`"servicePrincipalProfile":{"clientId":"11111111-1111-1111-1111-111111111111","clientSecret":"secret"},` +
		`"masterProfile":{"vmSize":"Standard_D8s_v3","subnetId":"` + subnet + `master"},` +
		`"workerProfiles":[{"name":"worker","vmSize":"Standard_D4s_v3","diskSizeGB":128,"count":3,"subnetId":"` + subnet + `worker"}],` +
		`"apiserverProfile":{"visibility":"Public","url":"https://forged"},"ingressProfiles":[{"name":"default","visibility":"Public"}]}}`
}

func TestAzureOpenShiftLifecycle(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-aro")
	cluster := sub + "/resourceGroups/rg-aro/providers/Microsoft.RedHatOpenShift/openShiftClusters/aro"

	resp := do(http.MethodPut, cluster+azureOpenShiftTestAPIVersion, azureOpenShiftTestCluster(sub))
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	props := providerContractJSONMap(t, resp)["properties"].(map[string]any)
	profile := props["clusterProfile"].(map[string]any)
	api := props["apiserverProfile"].(map[string]any)
	if profile["pullSecret"] != nil || props["servicePrincipalProfile"].(map[string]any)["clientSecret"] != nil {
		t.Fatalf("expected secrets stripped, got %#v", props)
	}
	if api["url"] != "https://api.stackyard.eastus.aroapp.io:6443/" || api["visibility"] != "Public" || profile["version"] != "4.14.16" || props["networkProfile"].(map[string]any)["podCidr"] != "10.128.0.0/14" {
		t.Fatalf("unexpected cluster properties: %#v", props)
	}

	// A PATCH that re-sends the secrets doesn't return them either.
	resp = do(http.MethodPatch, cluster+azureOpenShiftTestAPIVersion, `{"properties":{"clusterProfile":{"pullSecret":"again"},"servicePrincipalProfile":{"clientSecret":"again"}}}`)
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	props = providerContractJSONMap(t, resp)["properties"].(map[string]any)
	if props["clusterProfile"].(map[string]any)["pullSecret"] != nil || props["servicePrincipalProfile"].(map[string]any)["clientSecret"] != nil {
		t.Fatalf("expected secrets stripped after PATCH, got %#v", props)
	}
	azureResourcesExpectErrorCode(t, do(http.MethodPost, cluster+"x/listCredentials"+azureOpenShiftTestAPIVersion, ""), http.StatusNotFound, "ResourceNotFound")

	resp = do(http.MethodPost, cluster+"/listCredentials"+azureOpenShiftTestAPIVersion, "")
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if creds := providerContractJSONMap(t, resp); creds["kubeadminUsername"] != "kubeadmin" || len(creds["kubeadminPassword"].(string)) != 23 {
		t.Fatalf("unexpected credentials: %#v", creds)
	}
	resp = do(http.MethodPost, cluster+"/listAdminCredentials"+azureOpenShiftTestAPIVersion, "")
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	raw, err := base64.StdEncoding.DecodeString(providerContractJSONMap(t, resp)["kubeconfig"].(string))
	if err != nil || !strings.Contains(string(raw), "server: https://api.stackyard.eastus.aroapp.io:6443\n") {
		t.Fatalf("unexpected kubeconfig %q: %v", raw, err)
	}

	azureResourcesExpectStatus(t, do(http.MethodDelete, cluster+azureOpenShiftTestAPIVersion, ""), http.StatusOK)
}

func TestAzureOpenShiftValidation(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-aro")
	cluster := sub + "/resourceGroups/rg-aro/providers/Microsoft.RedHatOpenShift/openShiftClusters/aro"
	valid := azureOpenShiftTestCluster(sub)
	cases := map[string][2]string{
		"bad domain":         {`"domain":"stackyard"`, `"domain":"Stack_Yard"`},
		"bad resource group": {`"resourceGroupId":"` + strings.TrimPrefix(sub, "/azure") + `/resourceGroups/aro-infra"`, `"resourceGroupId":"aro-infra"`},
		"no client id":       {`"clientId":"11111111-1111-1111-1111-111111111111",`, ``},
		"bad master subnet":  {`/subnets/master"`, `"`},
		"too few workers":    {`"count":3`, `"count":2`},
		"no workers":         {`"workerProfiles":[{"name":"worker","vmSize":"Standard_D4s_v3","diskSizeGB":128,"count":3,"subnetId":"` + strings.TrimPrefix(sub, "/azure") + `/resourceGroups/rg-aro/providers/Microsoft.Network/virtualNetworks/aro-vnet/subnets/worker"}]`, `"workerProfiles":[]`},
		"bad visibility":     {`"name":"default","visibility":"Public"`, `"name":"default","visibility":"Internal"`},
	}
	for name, edit := range cases {
		body := strings.Replace(valid, edit[0], edit[1], 1)
		if body == valid {
			t.Fatalf("%s: edit did not apply", name)
		}
		resp := do(http.MethodPut, cluster+azureOpenShiftTestAPIVersion, body)
		t.Run(name, func(t *testing.T) {
			azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, "InvalidRequestContent")
		})
	}
}
