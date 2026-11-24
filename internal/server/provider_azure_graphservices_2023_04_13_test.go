package server

import (
	"net/http"
	"strings"
	"testing"
)

const azureGsvcTestAPI = "?api-version=2023-04-13"

func TestAzureGraphServicesAccountLifecycle(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-graph")
	account := sub + "/resourceGroups/rg-graph/providers/Microsoft.GraphServices/accounts/billing"

	resp := do(http.MethodPut, account+azureGsvcTestAPI, `{"location":"global","properties":{"appId":"0b3c4a6e-1f2d-4e5f-8a9b-0c1d2e3f4a5b"}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	body := providerContractJSONMap(t, resp)
	props := body["properties"].(map[string]any)
	if props["appId"] != "0b3c4a6e-1f2d-4e5f-8a9b-0c1d2e3f4a5b" || props["billingPlanId"] == "" || props["provisioningState"] != "Succeeded" {
		t.Fatalf("unexpected account: %#v", body)
	}

	resp = do(http.MethodPatch, account+azureGsvcTestAPI, `{"tags":{"env":"test"}}`)
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if got := providerContractJSONMap(t, resp)["properties"].(map[string]any)["billingPlanId"]; got != props["billingPlanId"] {
		t.Fatalf("billingPlanId changed on update: %v -> %v", props["billingPlanId"], got)
	}

	resp = do(http.MethodGet, sub+"/providers/Microsoft.GraphServices/accounts"+azureGsvcTestAPI, "")
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if items := providerContractJSONMap(t, resp)["value"].([]any); len(items) != 1 {
		t.Fatalf("want 1 account in the subscription, got %d", len(items))
	}

	resp = do(http.MethodGet, "/azure/providers/Microsoft.GraphServices/operations"+azureGsvcTestAPI, "")
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if !strings.Contains(string(mustBody(t, resp)), "Microsoft.GraphServices/accounts/read") {
		t.Fatal("operations list misses accounts/read")
	}

	azureResourcesExpectStatus(t, do(http.MethodDelete, account+azureGsvcTestAPI, ""), http.StatusOK)
	azureResourcesExpectErrorCode(t, do(http.MethodGet, account+azureGsvcTestAPI, ""), http.StatusNotFound, "ResourceNotFound")
}

func TestAzureGraphServicesAccountValidation(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-graph")
	account := sub + "/resourceGroups/rg-graph/providers/Microsoft.GraphServices/accounts/billing"
	for name, body := range map[string]string{
		"no appId":    `{"location":"global","properties":{}}`,
		"bad appId":   `{"location":"global","properties":{"appId":"not-a-guid"}}`,
		"no location": `{"properties":{"appId":"0b3c4a6e-1f2d-4e5f-8a9b-0c1d2e3f4a5b"}}`,
	} {
		resp := do(http.MethodPut, account+azureGsvcTestAPI, body)
		t.Run(name, func(t *testing.T) { azureResourcesExpectStatus(t, resp, http.StatusBadRequest) })
	}
}
