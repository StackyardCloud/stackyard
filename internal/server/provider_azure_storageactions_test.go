package server

import (
	"net/http"
	"strings"
	"testing"
)

const (
	azureStorageTaskTestAPIVersion = "?api-version=2023-01-01"
	azureStorageTaskTestBody       = `{"location":"eastus","identity":{"type":"SystemAssigned"},"properties":{"enabled":true,"description":"Archive old logs","action":{"if":{"condition":"[[and(endsWith(Name, '.log'), less(Creation-Time, '2025-01-01T00:00:00'))]]","operations":[{"name":"SetBlobTier","parameters":{"tier":"Archive"},"onSuccess":"continue","onFailure":"break"}]},"else":{"operations":[{"name":"SetBlobTags","parameters":{"reviewed":"true"}}]}}}}`
)

func TestAzureStorageTaskLifecycle(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-tasks")
	task := sub + "/resourceGroups/rg-tasks/providers/Microsoft.StorageActions/storageTasks/archivelogs"

	resp := do(http.MethodPut, task+azureStorageTaskTestAPIVersion, azureStorageTaskTestBody)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	if props := providerContractJSONMap(t, resp)["properties"].(map[string]any); props["taskVersion"] != float64(1) || props["creationTimeInUtc"] == nil {
		t.Fatalf("unexpected storage task properties: %#v", props)
	}
	azureResourcesExpectStatus(t, do(http.MethodPut, task+azureStorageTaskTestAPIVersion, strings.Replace(azureStorageTaskTestBody, `"else":{"operations":[{"name":"SetBlobTags","parameters":{"reviewed":"true"}}]}`, `"else":null`, 1)), http.StatusOK)
	resp = do(http.MethodPatch, task+azureStorageTaskTestAPIVersion, `{"properties":{"enabled":false}}`)
	if got := providerContractJSONMap(t, resp)["properties"].(map[string]any)["enabled"]; got != false {
		t.Fatalf("expected the task disabled, got %v", got)
	}
	resp = do(http.MethodGet, sub+"/providers/Microsoft.StorageActions/storageTasks"+azureStorageTaskTestAPIVersion, "")
	if got := providerContractJSONMap(t, resp)["value"].([]any); len(got) != 1 {
		t.Fatalf("expected one task, got %d", len(got))
	}
	azureResourcesExpectStatus(t, do(http.MethodDelete, task+azureStorageTaskTestAPIVersion, ""), http.StatusOK)
}

func TestAzureStorageTaskValidation(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-tasks")
	task := sub + "/resourceGroups/rg-tasks/providers/Microsoft.StorageActions/storageTasks/archivelogs"
	cases := []struct{ name, old, new string }{
		{"identity", `"identity":{"type":"SystemAssigned"},`, ``},
		{"enabled", `"enabled":true,`, ``},
		{"description", `"description":"Archive old logs",`, ``},
		{"condition", `"condition":"[[and`, `"condition":"and`},
		{"operation", `"name":"SetBlobTier"`, `"name":"CopyBlob"`},
		{"on success", `"onSuccess":"continue"`, `"onSuccess":"break"`},
		{"else operations", `"else":{"operations":[{"name":"SetBlobTags","parameters":{"reviewed":"true"}}]}`, `"else":{"operations":[]}`},
	}
	for _, tc := range cases {
		body := strings.Replace(azureStorageTaskTestBody, tc.old, tc.new, 1)
		if body == azureStorageTaskTestBody {
			t.Fatalf("case %s did not change the body", tc.name)
		}
		resp := do(http.MethodPut, task+azureStorageTaskTestAPIVersion, body)
		t.Run(tc.name, func(t *testing.T) {
			azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, "InvalidRequestContent")
		})
	}
	azureResourcesExpectErrorCode(t, do(http.MethodPut, sub+"/resourceGroups/rg-tasks/providers/Microsoft.StorageActions/storageTasks/Archive-Logs"+azureStorageTaskTestAPIVersion, azureStorageTaskTestBody), http.StatusBadRequest, "InvalidResourceName")
}
