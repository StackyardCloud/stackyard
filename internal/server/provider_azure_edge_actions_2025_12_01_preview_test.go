package server

import (
	"net/http"
	"strings"
	"testing"
)

const edgeaTestAPI = "?api-version=" + azureEdgeActions20251201PreviewAPIVersion

func TestAzureEdgeActionsLifecycle(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-ea")
	action := sub + "/resourceGroups/rg-ea/providers/Microsoft.Cdn/edgeActions/ea1"
	v1, v2 := action+"/versions/v1", action+"/versions/v2"

	resp := do(http.MethodPut, action+edgeaTestAPI, `{"location":"global","sku":{"name":"Standard","tier":"Standard"}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	if _, ok := providerContractJSONMap(t, resp)["properties"].(map[string]any)["attachments"]; !ok {
		t.Fatal("edge action has no attachments")
	}
	azureResourcesExpectStatus(t, do(http.MethodPut, v1+edgeaTestAPI, `{"location":"global","properties":{"deploymentType":"zip","isDefaultVersion":"True"}}`), http.StatusCreated)
	resp = do(http.MethodPut, v2+edgeaTestAPI, `{"location":"global","properties":{"deploymentType":"file","isDefaultVersion":"True"}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	if got := providerContractJSONMap(t, resp)["properties"].(map[string]any)["validationStatus"]; got != "Pending" {
		t.Fatalf("new version validationStatus = %v", got)
	}
	edgeaExpectDefault(t, do, v1, "False")

	azureResourcesExpectStatus(t, do(http.MethodPost, v1+"/swapDefault"+edgeaTestAPI, ""), http.StatusAccepted)
	edgeaExpectDefault(t, do, v1, "True")
	edgeaExpectDefault(t, do, v2, "False")

	azureResourcesExpectErrorCode(t, do(http.MethodPost, v1+"/getVersionCode"+edgeaTestAPI, ""), http.StatusNotFound, "NotFound")
	azureResourcesExpectErrorCode(t, do(http.MethodPost, v1+"/deployVersionCode"+edgeaTestAPI, `{"name":"code","content":"not base64!"}`), http.StatusBadRequest, "InvalidRequestContent")
	resp = do(http.MethodPost, v1+"/deployVersionCode"+edgeaTestAPI, `{"name":"code","content":"UEsDBA=="}`)
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if got := providerContractJSONMap(t, resp)["properties"].(map[string]any)["validationStatus"]; got != "Succeeded" {
		t.Fatalf("deployed validationStatus = %v", got)
	}
	resp = do(http.MethodPost, v1+"/getVersionCode"+edgeaTestAPI, "")
	if code := providerContractJSONMap(t, resp); code["name"] != "code" || code["content"] != "UEsDBA==" {
		t.Fatalf("unexpected code: %#v", code)
	}

	filter := action + "/executionFilters/f1"
	body := `{"location":"global","properties":{"versionId":"` + strings.TrimPrefix(v1, "/azure") + `","executionFilterIdentifierHeaderName":"x-ea","executionFilterIdentifierHeaderValue":"blue"}}`
	azureResourcesExpectStatus(t, do(http.MethodPut, filter+edgeaTestAPI, body), http.StatusCreated)
	azureResourcesExpectErrorCode(t, do(http.MethodPut, action+"/executionFilters/f2"+edgeaTestAPI, strings.Replace(body, "/versions/v1", "/versions/missing", 1)), http.StatusBadRequest, "InvalidRequestContent")
	azureResourcesExpectErrorCode(t, do(http.MethodDelete, v1+edgeaTestAPI, ""), http.StatusConflict, "Conflict")
	azureResourcesExpectStatus(t, do(http.MethodDelete, filter+edgeaTestAPI, ""), http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodDelete, v1+edgeaTestAPI, ""), http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodDelete, action+edgeaTestAPI, ""), http.StatusOK)
	azureResourcesExpectErrorCode(t, do(http.MethodGet, v2+edgeaTestAPI, ""), http.StatusNotFound, "ParentResourceNotFound")
}

func TestAzureEdgeActionsValidation(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-ea")
	action := sub + "/resourceGroups/rg-ea/providers/Microsoft.Cdn/edgeActions/ea1"
	azureResourcesExpectErrorCode(t, do(http.MethodPut, action+edgeaTestAPI, `{"location":"global"}`), http.StatusBadRequest, "InvalidRequestContent")
	azureResourcesExpectStatus(t, do(http.MethodPut, action+edgeaTestAPI, `{"location":"global","sku":{"name":"Standard","tier":"Standard"}}`), http.StatusCreated)
	azureResourcesExpectErrorCode(t, do(http.MethodPut, action+"/versions/v1"+edgeaTestAPI, `{"location":"global","properties":{"deploymentType":"tarball","isDefaultVersion":"True"}}`), http.StatusBadRequest, "InvalidRequestContent")
	azureResourcesExpectErrorCode(t, do(http.MethodPut, action+"/executionFilters/f1"+edgeaTestAPI, `{"location":"global","properties":{}}`), http.StatusBadRequest, "InvalidRequestContent")
}

func edgeaExpectDefault(t *testing.T, do func(method, path, body string) *http.Response, version, want string) {
	t.Helper()
	resp := do(http.MethodGet, version+edgeaTestAPI, "")
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if got := providerContractJSONMap(t, resp)["properties"].(map[string]any)["isDefaultVersion"]; got != want {
		t.Fatalf("%s isDefaultVersion = %v, want %s", version, got, want)
	}
}
