package server

import (
	"net/http"
	"testing"
)

func TestAzureQuantumDataPlaneLifecycle(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	headers := map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
		"Content-Type":              "application/json",
	}

	base := azureQuantumDataPlanePrefix
	api := "?api-version=" + azureQuantumDataPlaneAPIVersion

	resp := providerContractRequest(t, ts, http.MethodGet, base+"/providers"+api, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing providers, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	providers := providerContractJSONMap(t, resp)
	if values, ok := providers["value"].([]any); !ok || len(values) == 0 {
		t.Fatalf("expected providers payload, got %#v", providers)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/quotas"+api, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing quotas, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"/sessions"+api, []byte(`{
		"name":"session-a",
		"providerId":"ionq",
		"target":"ionq.simulator",
		"jobFailurePolicy":"Continue"
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating session, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	session := providerContractJSONMap(t, resp)
	sessionID, _ := session["id"].(string)
	if sessionID == "" || session["status"] != "Executing" {
		t.Fatalf("unexpected session payload: %#v", session)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/sessions/"+sessionID+api, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting session, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/sessions"+api, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing sessions, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"/sessions/"+sessionID+"/jobs"+api, []byte(`{
		"name":"session-job-a",
		"providerId":"ionq",
		"target":"ionq.simulator",
		"inputDataFormat":"qir.v1",
		"outputDataFormat":"microsoft.quantum-results.v1",
		"inputParams":{"shots":100}
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating session job, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	sessionJob := providerContractJSONMap(t, resp)
	sessionJobID, _ := sessionJob["id"].(string)
	if sessionJobID == "" || sessionJob["sessionId"] != sessionID {
		t.Fatalf("unexpected session job payload: %#v", sessionJob)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/sessions/"+sessionID+"/jobs"+api, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing session jobs, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	sessionJobs := providerContractJSONMap(t, resp)
	if values, ok := sessionJobs["value"].([]any); !ok || len(values) != 1 {
		t.Fatalf("expected one session job, got %#v", sessionJobs)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"/jobs"+api, []byte(`{
		"name":"job-a",
		"providerId":"microsoft-qc",
		"target":"microsoft.estimator",
		"inputParams":{"entryPoint":"Main"}
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating top-level job, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	job := providerContractJSONMap(t, resp)
	jobID, _ := job["id"].(string)
	if jobID == "" || job["status"] != "Waiting" {
		t.Fatalf("unexpected top-level job payload: %#v", job)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/jobs/"+jobID+api, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting job, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/jobs"+api, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing jobs, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	jobs := providerContractJSONMap(t, resp)
	if values, ok := jobs["value"].([]any); !ok || len(values) != 1 {
		t.Fatalf("expected one top-level job, got %#v", jobs)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"/jobs/"+jobID+":cancel"+api, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 cancelling job, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	cancelled := providerContractJSONMap(t, resp)
	if cancelled["status"] != "Cancelled" {
		t.Fatalf("expected cancelled job payload, got %#v", cancelled)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"/storage/sasUri"+api, []byte(`{
		"containerName":"quantum-data",
		"blobName":"input.json"
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting storage SAS URI, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	sas := providerContractJSONMap(t, resp)
	if sas["sasUri"] == "" {
		t.Fatalf("expected sasUri payload, got %#v", sas)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/topLevelItems"+api, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing top level items, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	topLevel := providerContractJSONMap(t, resp)
	if values, ok := topLevel["value"].([]any); !ok || len(values) < 2 {
		t.Fatalf("expected mixed top level items, got %#v", topLevel)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"/sessions/"+sessionID+":close"+api, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 closing session, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	closed := providerContractJSONMap(t, resp)
	if closed["status"] != "Succeeded" {
		t.Fatalf("expected closed session payload, got %#v", closed)
	}

	resp = providerContractRequest(t, ts, http.MethodDelete, base+"/jobs/"+jobID+api, nil, headers)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204 deleting top-level job, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodDelete, base+"/sessions/"+sessionID+"/jobs/"+sessionJobID+api, nil, headers)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204 deleting session job, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodDelete, base+"/sessions/"+sessionID+api, nil, headers)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204 deleting session, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
}

func TestAzureQuantumDataPlaneValidation(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	headers := map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
		"Content-Type":              "application/json",
	}

	base := azureQuantumDataPlanePrefix

	tests := []struct {
		name       string
		method     string
		path       string
		body       []byte
		wantStatus int
	}{
		{
			name:       "invalid api version",
			method:     http.MethodGet,
			path:       base + "/providers?api-version=bad",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid json job body",
			method:     http.MethodPost,
			path:       base + "/jobs?api-version=" + azureQuantumDataPlaneAPIVersion,
			body:       []byte(`{`),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing job fields",
			method:     http.MethodPost,
			path:       base + "/jobs?api-version=" + azureQuantumDataPlaneAPIVersion,
			body:       []byte(`{"name":"job-a"}`),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing session parent",
			method:     http.MethodPost,
			path:       base + "/sessions/session-missing/jobs?api-version=" + azureQuantumDataPlaneAPIVersion,
			body:       []byte(`{"providerId":"ionq","target":"ionq.simulator"}`),
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "missing session",
			method:     http.MethodGet,
			path:       base + "/sessions/session-missing?api-version=" + azureQuantumDataPlaneAPIVersion,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "missing job",
			method:     http.MethodGet,
			path:       base + "/jobs/job-missing?api-version=" + azureQuantumDataPlaneAPIVersion,
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := providerContractRequest(t, ts, tt.method, tt.path, tt.body, headers)
			if resp.StatusCode != tt.wantStatus {
				t.Fatalf("expected %d, got %d body=%s", tt.wantStatus, resp.StatusCode, string(providerContractBody(t, resp)))
			}
		})
	}
}
