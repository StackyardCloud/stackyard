package server

import (
	"encoding/base64"
	"net/http"
	"strings"
	"testing"
)

const azureFltTestAPI = "?api-version=2025-03-01"

func TestAzureFleetLifecycle(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-fleet")
	fleet := sub + "/resourceGroups/rg-fleet/providers/Microsoft.ContainerService/fleets/prod"
	cluster := "/subscriptions/" + azureResourcesTestSubscription + "/resourceGroups/rg-aks/providers/Microsoft.ContainerService/managedClusters/aks1"

	resp := do(http.MethodPut, fleet+azureFltTestAPI, `{"location":"eastus","properties":{"hubProfile":{"dnsPrefix":"prodhub"}}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	body := providerContractJSONMap(t, resp)
	hub := body["properties"].(map[string]any)["hubProfile"].(map[string]any)
	if !strings.HasPrefix(hub["fqdn"].(string), "prodhub-") || body["etag"] == nil {
		t.Fatalf("unexpected fleet: %#v", body)
	}
	resp = do(http.MethodPost, fleet+"/listCredentials"+azureFltTestAPI, "")
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	raw, _ := base64.StdEncoding.DecodeString(providerContractJSONMap(t, resp)["kubeconfigs"].([]any)[0].(map[string]any)["value"].(string))
	if !strings.Contains(string(raw), hub["fqdn"].(string)) {
		t.Fatalf("kubeconfig does not point at the hub: %s", raw)
	}

	azureResourcesExpectStatus(t, do(http.MethodPut, fleet+"/members/aks1"+azureFltTestAPI, `{"properties":{"clusterResourceId":"`+cluster+`","group":"canary"}}`), http.StatusCreated)
	azureResourcesExpectErrorCode(t, do(http.MethodPut, fleet+"/members/again"+azureFltTestAPI, `{"properties":{"clusterResourceId":"`+cluster+`"}}`), http.StatusConflict, "Conflict")

	strategy := fleet + "/updateStrategies/rollout"
	azureResourcesExpectStatus(t, do(http.MethodPut, strategy+azureFltTestAPI, `{"properties":{"strategy":{"stages":[{"name":"stage1","groups":[{"name":"canary"}],"afterStageWaitInSeconds":60}]}}}`), http.StatusCreated)

	run := fleet + "/updateRuns/run1"
	resp = do(http.MethodPut, run+azureFltTestAPI, `{"properties":{"updateStrategyId":"`+strings.TrimPrefix(strategy, "/azure")+`","managedClusterUpdate":{"upgrade":{"type":"Full","kubernetesVersion":"1.31.2"}}}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	props := providerContractJSONMap(t, resp)["properties"].(map[string]any)
	if props["strategy"] == nil || props["status"].(map[string]any)["status"].(map[string]any)["state"] != "NotStarted" {
		t.Fatalf("run did not snapshot its strategy or start NotStarted: %#v", props)
	}
	resp = do(http.MethodPost, run+"/skip"+azureFltTestAPI, `{"targets":[{"type":"Group","name":"canary"}]}`)
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if len(providerContractJSONMap(t, resp)["properties"].(map[string]any)["skippedTargets"].([]any)) != 1 {
		t.Fatal("skip did not record its target")
	}
	resp = do(http.MethodPost, run+"/start"+azureFltTestAPI, "")
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if got := providerContractJSONMap(t, resp)["properties"].(map[string]any)["status"].(map[string]any)["status"].(map[string]any)["state"]; got != "Completed" {
		t.Fatalf("state after start = %v, want Completed", got)
	}
	azureResourcesExpectErrorCode(t, do(http.MethodPost, run+"/start"+azureFltTestAPI, ""), http.StatusConflict, "Conflict")
	azureResourcesExpectErrorCode(t, do(http.MethodPost, run+"/stop"+azureFltTestAPI, ""), http.StatusConflict, "Conflict")
	azureResourcesExpectErrorCode(t, do(http.MethodPost, run+"/skip"+azureFltTestAPI, `{"targets":[{"type":"Group","name":"canary"}]}`), http.StatusConflict, "Conflict")

	profile := fleet + "/autoUpgradeProfiles/stable"
	azureResourcesExpectStatus(t, do(http.MethodPut, profile+azureFltTestAPI, `{"properties":{"channel":"Stable"}}`), http.StatusCreated)
	resp = do(http.MethodPost, profile+"/generateUpdateRun"+azureFltTestAPI, "")
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	generated := providerContractJSONMap(t, resp)["id"].(string)
	azureResourcesExpectStatus(t, do(http.MethodGet, "/azure"+generated+azureFltTestAPI, ""), http.StatusOK)

	azureResourcesExpectStatus(t, do(http.MethodDelete, fleet+azureFltTestAPI, ""), http.StatusOK)
	azureResourcesExpectErrorCode(t, do(http.MethodGet, run+azureFltTestAPI, ""), http.StatusNotFound, "ParentResourceNotFound")
}

func TestAzureFleetValidation(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-fleet")
	fleet := sub + "/resourceGroups/rg-fleet/providers/Microsoft.ContainerService/fleets/nohub"
	azureResourcesExpectStatus(t, do(http.MethodPut, fleet+azureFltTestAPI, `{"location":"eastus"}`), http.StatusCreated)
	azureResourcesExpectErrorCode(t, do(http.MethodPost, fleet+"/listCredentials"+azureFltTestAPI, ""), http.StatusBadRequest, "InvalidRequestContent")

	cases := []struct{ name, path, body string }{
		{"bad dns prefix", fleet, `{"location":"eastus","properties":{"hubProfile":{"dnsPrefix":"-bad-"}}}`},
		{"member without cluster", fleet + "/members/m1", `{"properties":{}}`},
		{"member with storage account", fleet + "/members/m1", `{"properties":{"clusterResourceId":"/subscriptions/x/resourceGroups/y/providers/Microsoft.Storage/storageAccounts/z"}}`},
		{"strategy without stages", fleet + "/updateStrategies/s1", `{"properties":{"strategy":{"stages":[]}}}`},
		{"duplicate stages", fleet + "/updateStrategies/s1", `{"properties":{"strategy":{"stages":[{"name":"a"},{"name":"a"}]}}}`},
		{"unknown upgrade type", fleet + "/updateRuns/r1", `{"properties":{"managedClusterUpdate":{"upgrade":{"type":"Partial"}}}}`},
		{"full upgrade without version", fleet + "/updateRuns/r1", `{"properties":{"managedClusterUpdate":{"upgrade":{"type":"Full"}}}}`},
		{"unknown channel", fleet + "/autoUpgradeProfiles/p1", `{"properties":{"channel":"Beta"}}`},
	}
	for _, tc := range cases {
		resp := do(http.MethodPut, tc.path+azureFltTestAPI, tc.body)
		t.Run(tc.name, func(t *testing.T) {
			azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, "InvalidRequestContent")
		})
	}
	azureResourcesExpectErrorCode(t, do(http.MethodPut, fleet+"/updateRuns/r2"+azureFltTestAPI,
		`{"properties":{"updateStrategyId":"/subscriptions/`+azureResourcesTestSubscription+`/resourceGroups/rg-fleet/providers/Microsoft.ContainerService/fleets/nohub/updateStrategies/missing","managedClusterUpdate":{"upgrade":{"type":"NodeImageOnly"}}}}`),
		http.StatusBadRequest, "InvalidRequestContent")
}
