package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPWorkstationsRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPWorkstationsContractServer(t)
	location := "/gcp/v1/projects/stackyard/locations/us-central1"
	cluster := location + "/workstationClusters/cluster-1"
	config := cluster + "/workstationConfigs/config-1"
	workstationRunning := config + "/workstations/workstation-running"
	workstationStopped := config + "/workstations/workstation-stopped"

	assertGCPWorkstationsSuccess(t, ts, http.MethodGet, location+"/workstationClusters?pageSize=1", nil, `"workstationClusters"`)
	assertGCPWorkstationsSuccess(t, ts, http.MethodGet, cluster, nil, "workstationClusters/cluster-1")
	assertGCPWorkstationsSuccess(t, ts, http.MethodPost, location+"/workstationClusters?workstationClusterId=cluster-1", []byte(`{
		"workstationCluster": {
			"name": "projects/stackyard/locations/us-central1/workstationClusters/cluster-1",
			"network": "projects/stackyard/global/networks/default"
		}
	}`), "/operations/createWorkstationCluster.cluster-1")
	assertGCPWorkstationsSuccess(t, ts, http.MethodPatch, cluster+"?updateMask=displayName,network", []byte(`{
		"workstationCluster": {
			"name": "projects/stackyard/locations/us-central1/workstationClusters/cluster-1",
			"network": "projects/stackyard/global/networks/default"
		}
	}`), "/operations/updateWorkstationCluster.cluster-1")
	assertGCPWorkstationsSuccess(t, ts, http.MethodDelete, cluster, nil, "/operations/deleteWorkstationCluster.cluster-1")

	assertGCPWorkstationsSuccess(t, ts, http.MethodGet, cluster+"/workstationConfigs?pageSize=1", nil, `"workstationConfigs"`)
	assertGCPWorkstationsSuccess(t, ts, http.MethodGet, cluster+"/workstationConfigs:listUsable?pageSize=1", nil, `"workstationConfigs"`)
	assertGCPWorkstationsSuccess(t, ts, http.MethodGet, config, nil, "workstationConfigs/config-1")
	assertGCPWorkstationsSuccess(t, ts, http.MethodPost, cluster+"/workstationConfigs?workstationConfigId=config-1", []byte(`{
		"workstationConfig": {
			"name": "projects/stackyard/locations/us-central1/workstationClusters/cluster-1/workstationConfigs/config-1",
			"host": {"gceInstance":{"machineType":"e2-standard-4"}}
		}
	}`), "/operations/createWorkstationConfig.config-1")
	assertGCPWorkstationsSuccess(t, ts, http.MethodPatch, config+"?updateMask=displayName,host", []byte(`{
		"workstationConfig": {
			"name": "projects/stackyard/locations/us-central1/workstationClusters/cluster-1/workstationConfigs/config-1",
			"host": {"gceInstance":{"machineType":"e2-standard-4"}}
		}
	}`), "/operations/updateWorkstationConfig.config-1")
	assertGCPWorkstationsSuccess(t, ts, http.MethodDelete, config, nil, "/operations/deleteWorkstationConfig.config-1")

	assertGCPWorkstationsSuccess(t, ts, http.MethodGet, config+"/workstations?pageSize=1", nil, `"workstations"`)
	assertGCPWorkstationsSuccess(t, ts, http.MethodGet, config+"/workstations:listUsable?pageSize=1", nil, `"workstations"`)
	assertGCPWorkstationsSuccess(t, ts, http.MethodGet, workstationRunning, nil, "workstations/workstation-running")
	assertGCPWorkstationsSuccess(t, ts, http.MethodPost, config+"/workstations?workstationId=workstation-new", []byte(`{
		"workstation": {
			"name": "projects/stackyard/locations/us-central1/workstationClusters/cluster-1/workstationConfigs/config-1/workstations/workstation-new"
		}
	}`), "/operations/createWorkstation.workstation-new")
	assertGCPWorkstationsSuccess(t, ts, http.MethodPatch, workstationRunning+"?updateMask=displayName", []byte(`{
		"workstation": {
			"name": "projects/stackyard/locations/us-central1/workstationClusters/cluster-1/workstationConfigs/config-1/workstations/workstation-running"
		}
	}`), "/operations/updateWorkstation.workstation-running")
	assertGCPWorkstationsSuccess(t, ts, http.MethodDelete, workstationRunning, nil, "/operations/deleteWorkstation.workstation-running")
	assertGCPWorkstationsSuccess(t, ts, http.MethodPost, workstationStopped+":start", []byte(`{}`), "/operations/startWorkstation.workstation-stopped")
	assertGCPWorkstationsSuccess(t, ts, http.MethodPost, workstationRunning+":stop", []byte(`{}`), "/operations/stopWorkstation.workstation-running")
	assertGCPWorkstationsSuccess(t, ts, http.MethodPost, workstationRunning+":generateAccessToken", []byte(`{
		"workstation": "projects/stackyard/locations/us-central1/workstationClusters/cluster-1/workstationConfigs/config-1/workstations/workstation-running"
	}`), `"accessToken"`)

	assertGCPWorkstationsSuccess(t, ts, http.MethodGet, location+"/operations?pageSize=1", nil, `"operations"`)
	assertGCPWorkstationsSuccess(t, ts, http.MethodGet, location+"/operations/op-done", nil, "/operations/op-done")
	assertGCPWorkstationsSuccess(t, ts, http.MethodPost, location+"/operations/op-1:cancel", []byte(`{}`), `{}`)
	assertGCPWorkstationsSuccess(t, ts, http.MethodDelete, location+"/operations/op-1", nil, `{}`)

	assertGCPWorkstationsSuccess(t, ts, http.MethodGet, cluster+":getIamPolicy", nil, `"bindings"`)
	assertGCPWorkstationsSuccess(t, ts, http.MethodPost, cluster+":setIamPolicy", []byte(`{
		"policy": {
			"version": 1,
			"bindings": [
				{"role":"roles/workstations.user","members":["user:stackyard@example.com"]}
			]
		}
	}`), `"resource"`)
	assertGCPWorkstationsSuccess(t, ts, http.MethodPost, cluster+":testIamPermissions", []byte(`{
		"permissions": ["workstations.workstationClusters.get"]
	}`), `"permissions"`)
}

func TestGCPWorkstationsRouter_RequestValidationFailures(t *testing.T) {
	t.Parallel()

	ts := newGCPWorkstationsContractServer(t)
	location := "/gcp/v1/projects/stackyard/locations/us-central1"
	cluster := location + "/workstationClusters/cluster-1"
	config := cluster + "/workstationConfigs/config-1"
	workstationRunning := config + "/workstations/workstation-running"
	workstationStopped := config + "/workstations/workstation-stopped"

	cases := []struct {
		name     string
		method   string
		path     string
		payload  []byte
		wantCode int
		wantErr  string
	}{
		{
			name:     "list clusters invalid page size",
			method:   http.MethodGet,
			path:     location + "/workstationClusters?pageSize=bad",
			wantCode: http.StatusBadRequest,
			wantErr:  `"error":"InvalidArgument"`,
		},
		{
			name:   "create cluster missing id",
			method: http.MethodPost,
			path:   location + "/workstationClusters",
			payload: []byte(`{
				"workstationCluster": {"network":"projects/stackyard/global/networks/default"}
			}`),
			wantCode: http.StatusBadRequest,
			wantErr:  `"error":"InvalidArgument"`,
		},
		{
			name:     "update cluster missing update mask",
			method:   http.MethodPatch,
			path:     cluster,
			payload:  []byte(`{"workstationCluster":{"name":"projects/stackyard/locations/us-central1/workstationClusters/cluster-1"}}`),
			wantCode: http.StatusBadRequest,
			wantErr:  `"error":"InvalidArgument"`,
		},
		{
			name:     "delete in-use cluster without force",
			method:   http.MethodDelete,
			path:     location + "/workstationClusters/cluster-inuse",
			wantCode: http.StatusBadRequest,
			wantErr:  `"error":"FailedPrecondition"`,
		},
		{
			name:     "delete cluster invalid force",
			method:   http.MethodDelete,
			path:     cluster + "?force=notabool",
			wantCode: http.StatusBadRequest,
			wantErr:  `"error":"InvalidArgument"`,
		},
		{
			name:     "start running workstation fails precondition",
			method:   http.MethodPost,
			path:     workstationRunning + ":start",
			payload:  []byte(`{}`),
			wantCode: http.StatusBadRequest,
			wantErr:  `"error":"FailedPrecondition"`,
		},
		{
			name:     "stop stopped workstation fails precondition",
			method:   http.MethodPost,
			path:     workstationStopped + ":stop",
			payload:  []byte(`{}`),
			wantCode: http.StatusBadRequest,
			wantErr:  `"error":"FailedPrecondition"`,
		},
		{
			name:     "access token requires running workstation",
			method:   http.MethodPost,
			path:     workstationStopped + ":generateAccessToken",
			payload:  []byte(`{}`),
			wantCode: http.StatusBadRequest,
			wantErr:  `"error":"FailedPrecondition"`,
		},
		{
			name:     "set iam policy requires policy",
			method:   http.MethodPost,
			path:     cluster + ":setIamPolicy",
			payload:  []byte(`{}`),
			wantCode: http.StatusBadRequest,
			wantErr:  `"error":"InvalidArgument"`,
		},
		{
			name:     "test iam permissions requires permissions",
			method:   http.MethodPost,
			path:     cluster + ":testIamPermissions",
			payload:  []byte(`{}`),
			wantCode: http.StatusBadRequest,
			wantErr:  `"error":"InvalidArgument"`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			headers := map[string]string{
				"X-Stackyard-GCP-Service": "workstations",
			}
			if tc.payload != nil {
				headers["Content-Type"] = "application/json"
			}
			resp := providerContractRequest(t, ts, tc.method, tc.path, tc.payload, headers)
			if resp.StatusCode != tc.wantCode {
				t.Fatalf("expected %d from gcp workstations router, got %d body=%s", tc.wantCode, resp.StatusCode, string(providerContractBody(t, resp)))
			}
			if body := string(providerContractBody(t, resp)); !strings.Contains(body, tc.wantErr) {
				t.Fatalf("unexpected response body: %s", body)
			}
		})
	}
}

func TestGCPWorkstationsRouter_OutputShapeAssertions(t *testing.T) {
	t.Parallel()

	ts := newGCPWorkstationsContractServer(t)
	headers := map[string]string{
		"X-Stackyard-GCP-Service": "workstations",
	}

	clusterResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/workstationClusters/cluster-1", nil, headers)
	if clusterResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from get cluster, got %d body=%s", clusterResp.StatusCode, string(providerContractBody(t, clusterResp)))
	}
	clusterBody := providerContractJSONMap(t, clusterResp)
	if _, ok := clusterBody["name"].(string); !ok {
		t.Fatalf("expected cluster.name string, got %#v", clusterBody["name"])
	}
	if _, ok := clusterBody["network"].(string); !ok {
		t.Fatalf("expected cluster.network string, got %#v", clusterBody["network"])
	}
	if _, ok := clusterBody["annotations"].(map[string]any); !ok {
		t.Fatalf("expected cluster.annotations object, got %#v", clusterBody["annotations"])
	}
	if _, ok := clusterBody["createTime"].(string); !ok {
		t.Fatalf("expected cluster.createTime string, got %#v", clusterBody["createTime"])
	}

	configResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/workstationClusters/cluster-1/workstationConfigs/config-1", nil, headers)
	if configResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from get config, got %d body=%s", configResp.StatusCode, string(providerContractBody(t, configResp)))
	}
	configBody := providerContractJSONMap(t, configResp)
	if _, ok := configBody["host"].(map[string]any); !ok {
		t.Fatalf("expected config.host object, got %#v", configBody["host"])
	}
	if _, ok := configBody["container"].(map[string]any); !ok {
		t.Fatalf("expected config.container object, got %#v", configBody["container"])
	}
	if _, ok := configBody["idleTimeout"].(string); !ok {
		t.Fatalf("expected config.idleTimeout string, got %#v", configBody["idleTimeout"])
	}

	workstationResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/workstationClusters/cluster-1/workstationConfigs/config-1/workstations/workstation-running", nil, headers)
	if workstationResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from get workstation, got %d body=%s", workstationResp.StatusCode, string(providerContractBody(t, workstationResp)))
	}
	workstationBody := providerContractJSONMap(t, workstationResp)
	if got, _ := workstationBody["state"].(string); got != "STATE_RUNNING" {
		t.Fatalf("expected workstation.state STATE_RUNNING, got %#v", workstationBody["state"])
	}
	if _, ok := workstationBody["host"].(string); !ok {
		t.Fatalf("expected workstation.host string, got %#v", workstationBody["host"])
	}

	listResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/workstationClusters?pageSize=1", nil, headers)
	if listResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from list clusters, got %d body=%s", listResp.StatusCode, string(providerContractBody(t, listResp)))
	}
	listBody := providerContractJSONMap(t, listResp)
	clusters, ok := listBody["workstationClusters"].([]any)
	if !ok || len(clusters) != 1 {
		t.Fatalf("expected workstationClusters array size 1, got %#v", listBody["workstationClusters"])
	}
	if _, ok := listBody["nextPageToken"].(string); !ok {
		t.Fatalf("expected nextPageToken string, got %#v", listBody["nextPageToken"])
	}

	tokenResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/workstationClusters/cluster-1/workstationConfigs/config-1/workstations/workstation-running:generateAccessToken", []byte(`{}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "workstations",
	})
	if tokenResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from generate access token, got %d body=%s", tokenResp.StatusCode, string(providerContractBody(t, tokenResp)))
	}
	tokenBody := providerContractJSONMap(t, tokenResp)
	if _, ok := tokenBody["accessToken"].(string); !ok {
		t.Fatalf("expected accessToken string, got %#v", tokenBody["accessToken"])
	}
	if _, ok := tokenBody["expireTime"].(string); !ok {
		t.Fatalf("expected expireTime string, got %#v", tokenBody["expireTime"])
	}

	operationResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/operations/op-done", nil, headers)
	if operationResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from get operation, got %d body=%s", operationResp.StatusCode, string(providerContractBody(t, operationResp)))
	}
	operationBody := providerContractJSONMap(t, operationResp)
	if _, ok := operationBody["name"].(string); !ok {
		t.Fatalf("expected operation.name string, got %#v", operationBody["name"])
	}
	metadata, ok := operationBody["metadata"].(map[string]any)
	if !ok {
		t.Fatalf("expected operation.metadata object, got %#v", operationBody["metadata"])
	}
	if _, ok := metadata["target"].(string); !ok {
		t.Fatalf("expected operation.metadata.target string, got %#v", metadata["target"])
	}
	if _, ok := metadata["verb"].(string); !ok {
		t.Fatalf("expected operation.metadata.verb string, got %#v", metadata["verb"])
	}

	policyResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/workstationClusters/cluster-1:getIamPolicy", nil, headers)
	if policyResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from get iam policy, got %d body=%s", policyResp.StatusCode, string(providerContractBody(t, policyResp)))
	}
	policyBody := providerContractJSONMap(t, policyResp)
	if _, ok := policyBody["version"].(float64); !ok {
		t.Fatalf("expected policy.version number, got %#v", policyBody["version"])
	}
	if _, ok := policyBody["bindings"].([]any); !ok {
		t.Fatalf("expected policy.bindings array, got %#v", policyBody["bindings"])
	}
}

func TestGCPWorkstationsRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/workstations?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp workstations contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "workstations" {
		t.Fatalf("expected service=workstations, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}

func newGCPWorkstationsContractServer(t *testing.T) *httptest.Server {
	t.Helper()

	return newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})
}

func assertGCPWorkstationsSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{
		"X-Stackyard-GCP-Service": "workstations",
	}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp workstations router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}
