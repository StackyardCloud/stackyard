package server

import (
	"net/http"
	"strings"
	"testing"
)

const azureImageBuilderTestAPIVersion = "?api-version=2024-02-01"

const azureImageBuilderTestTemplate = `{"location":"eastus","identity":{"type":"UserAssigned","userAssignedIdentities":{"/subscriptions/s/resourceGroups/rg/providers/Microsoft.ManagedIdentity/userAssignedIdentities/builder":{}}},"properties":{"source":{"type":"PlatformImage","publisher":"Canonical","offer":"0001-com-ubuntu-server-jammy","sku":"22_04-lts","version":"latest"},"customize":[{"type":"Shell","name":"update","inline":["sudo apt-get update"]}],"distribute":[{"type":"ManagedImage","imageId":"/subscriptions/s/resourceGroups/rg/providers/Microsoft.Compute/images/golden","location":"eastus","runOutputName":"golden"}],"vmProfile":{"vmSize":"Standard_D2s_v3"},"lastRunStatus":{"runState":"Failed"}}}`

func TestAzureImageBuilderLifecycle(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-aib")
	template := sub + "/resourceGroups/rg-aib/providers/Microsoft.VirtualMachineImages/imageTemplates/golden"

	resp := do(http.MethodPut, template+azureImageBuilderTestAPIVersion, azureImageBuilderTestTemplate)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	// A new template has no run, and clients can't set one.
	if props := providerContractJSONMap(t, resp)["properties"].(map[string]any); props["lastRunStatus"] != nil {
		t.Fatalf("expected no lastRunStatus, got %#v", props["lastRunStatus"])
	}

	azureResourcesExpectStatus(t, do(http.MethodPost, template+"/run"+azureImageBuilderTestAPIVersion, ""), http.StatusOK)
	resp = do(http.MethodPatch, template+azureImageBuilderTestAPIVersion, `{"tags":{"env":"dev"},"properties":{"lastRunStatus":{"runState":"Failed"}}}`)
	if got := providerContractJSONMap(t, resp)["properties"].(map[string]any)["lastRunStatus"].(map[string]any); got["runState"] != "Succeeded" {
		t.Fatalf("expected the run to stay Succeeded, got %#v", got)
	}
	azureResourcesExpectStatus(t, do(http.MethodDelete, template+azureImageBuilderTestAPIVersion, ""), http.StatusOK)
}

func TestAzureImageBuilderValidation(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-aib")
	template := sub + "/resourceGroups/rg-aib/providers/Microsoft.VirtualMachineImages/imageTemplates/golden"
	cases := map[string][2]string{
		"system identity":     {`"identity":{"type":"UserAssigned"`, `"identity":{"type":"SystemAssigned"`},
		"bad source":          {`"type":"PlatformImage"`, `"type":"Iso"`},
		"bad distribute type": {`"type":"ManagedImage"`, `"type":"Blob"`},
		"no run output":       {`"runOutputName":"golden"`, `"runOutputName":""`},
		"no image id":         {`"imageId":"/subscriptions/s/resourceGroups/rg/providers/Microsoft.Compute/images/golden",`, ``},
		"bad customizer":      {`"type":"Shell","name":"update"`, `"type":"Ansible","name":"update"`},
		"no distribute":       {`"distribute":[{"type":"ManagedImage","imageId":"/subscriptions/s/resourceGroups/rg/providers/Microsoft.Compute/images/golden","location":"eastus","runOutputName":"golden"}]`, `"distribute":[]`},
		"bad timeout":         {`"vmProfile"`, `"buildTimeoutInMinutes":961,"vmProfile"`},
	}
	for name, edit := range cases {
		body := strings.Replace(azureImageBuilderTestTemplate, edit[0], edit[1], 1)
		if body == azureImageBuilderTestTemplate {
			t.Fatalf("%s: edit did not apply", name)
		}
		resp := do(http.MethodPut, template+azureImageBuilderTestAPIVersion, body)
		t.Run(name, func(t *testing.T) {
			azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, "InvalidRequestContent")
		})
	}
}
