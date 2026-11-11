package server

import (
	"fmt"
	"net/http"
	"testing"
)

func TestAzureCommunicationJobRouterLifecycle(t *testing.T) {
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
	}
	jsonHeaders := map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
		"Content-Type":              "application/merge-patch+json",
	}

	base := "/azure/communication-jobrouter/2023-11-01/routing"

	adminCases := []struct {
		family      string
		id          string
		createBody  string
		updateBody  string
		expectedKey string
	}{
		{
			family:      "classificationPolicies",
			id:          "class-a",
			createBody:  `{"name":"classification-a","fallbackQueueId":"queue-a"}`,
			updateBody:  `{"name":"classification-a-updated"}`,
			expectedKey: "name",
		},
		{
			family:      "distributionPolicies",
			id:          "dist-a",
			createBody:  `{"name":"distribution-a","mode":{"kind":"roundRobin"}}`,
			updateBody:  `{"name":"distribution-a-updated","mode":{"kind":"longestIdle"}}`,
			expectedKey: "name",
		},
		{
			family:      "exceptionPolicies",
			id:          "exception-a",
			createBody:  `{"name":"exception-a","exceptionRules":[{"id":"rule-a"}]}`,
			updateBody:  `{"name":"exception-a-updated","exceptionRules":[{"id":"rule-b"}]}`,
			expectedKey: "name",
		},
		{
			family:      "queues",
			id:          "queue-a",
			createBody:  `{"name":"queue-a","distributionPolicyId":"dist-a","exceptionPolicyId":"exception-a","labels":{"env":"test"}}`,
			updateBody:  `{"name":"queue-a-updated","distributionPolicyId":"dist-a"}`,
			expectedKey: "name",
		},
	}

	for _, tc := range adminCases {
		t.Run("admin_"+tc.family, func(t *testing.T) {
			path := fmt.Sprintf("%s/%s/%s?api-version=%s", base, tc.family, tc.id, azureCommunicationJobRouterAPIVersion)
			resp := providerContractRequest(t, ts, http.MethodPatch, path, []byte(tc.createBody), jsonHeaders)
			if resp.StatusCode != http.StatusCreated {
				t.Fatalf("expected 201 for create %s, got %d body=%s", tc.family, resp.StatusCode, string(providerContractBody(t, resp)))
			}
			if resp.Header.Get("ETag") == "" || resp.Header.Get("Last-Modified") == "" {
				t.Fatalf("expected ETag and Last-Modified on create %s", tc.family)
			}
			createPayload := providerContractJSONMap(t, resp)
			if createPayload["id"] != tc.id {
				t.Fatalf("expected id %q, got %#v", tc.id, createPayload)
			}

			resp = providerContractRequest(t, ts, http.MethodPatch, path, []byte(tc.updateBody), jsonHeaders)
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("expected 200 for update %s, got %d body=%s", tc.family, resp.StatusCode, string(providerContractBody(t, resp)))
			}

			resp = providerContractRequest(t, ts, http.MethodGet, path, nil, headers)
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("expected 200 for get %s, got %d body=%s", tc.family, resp.StatusCode, string(providerContractBody(t, resp)))
			}
			getPayload := providerContractJSONMap(t, resp)
			if getPayload[tc.expectedKey] == "" {
				t.Fatalf("expected %s field for %s, got %#v", tc.expectedKey, tc.family, getPayload)
			}

			listPath := fmt.Sprintf("%s/%s?api-version=%s&maxpagesize=10", base, tc.family, azureCommunicationJobRouterAPIVersion)
			resp = providerContractRequest(t, ts, http.MethodGet, listPath, nil, headers)
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("expected 200 for list %s, got %d body=%s", tc.family, resp.StatusCode, string(providerContractBody(t, resp)))
			}
			listPayload := providerContractJSONMap(t, resp)
			if len(listPayload["value"].([]any)) == 0 {
				t.Fatalf("expected list values for %s, got %#v", tc.family, listPayload)
			}
		})
	}

	resp := providerContractRequest(t, ts, http.MethodPatch, base+"/workers/worker-a?api-version="+azureCommunicationJobRouterAPIVersion, []byte(`{
		"queueIds":["queue-a"],
		"capacity":2,
		"labels":{"lang":"en"}
	}`), jsonHeaders)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 for create worker-a, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPatch, base+"/workers/worker-b?api-version="+azureCommunicationJobRouterAPIVersion, []byte(`{
		"queueIds":["queue-a"],
		"capacity":1,
		"labels":{"lang":"fr"}
	}`), jsonHeaders)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 for create worker-b, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/workers/worker-a?api-version="+azureCommunicationJobRouterAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for get worker-a, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/workers?api-version="+azureCommunicationJobRouterAPIVersion+"&maxpagesize=10", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for list workers, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPatch, base+"/jobs/job-1?api-version="+azureCommunicationJobRouterAPIVersion, []byte(`{
		"channelId":"chat",
		"queueId":"queue-a",
		"priority":10,
		"labels":{"topic":"sales"}
	}`), jsonHeaders)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 for create job-1, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPatch, base+"/jobs/job-2?api-version="+azureCommunicationJobRouterAPIVersion, []byte(`{
		"channelId":"voice",
		"queueId":"queue-a",
		"priority":5
	}`), jsonHeaders)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 for create job-2, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPatch, base+"/jobs/job-3?api-version="+azureCommunicationJobRouterAPIVersion, []byte(`{
		"channelId":"chat",
		"queueId":"queue-a",
		"priority":1
	}`), jsonHeaders)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 for create job-3, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/jobs/job-1?api-version="+azureCommunicationJobRouterAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for get job-1, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	jobOnePayload := providerContractJSONMap(t, resp)
	offers := jobOnePayload["offers"].([]any)
	if len(offers) < 2 {
		t.Fatalf("expected offers for job-1, got %#v", jobOnePayload)
	}
	var offerA, offerB string
	for _, raw := range offers {
		offer := providerContractJSONMapFromAny(t, raw)
		switch offer["workerId"] {
		case "worker-a":
			offerA = offer["id"].(string)
		case "worker-b":
			offerB = offer["id"].(string)
		}
	}
	if offerA == "" || offerB == "" {
		t.Fatalf("expected offers for both workers, got %#v", jobOnePayload)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/jobs?api-version="+azureCommunicationJobRouterAPIVersion+"&maxpagesize=10", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for list jobs, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/queues/queue-a/statistics?api-version="+azureCommunicationJobRouterAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for queue statistics, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/jobs/job-2/position?api-version="+azureCommunicationJobRouterAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for position, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"/workers/worker-a/offers/"+offerA+":accept?api-version="+azureCommunicationJobRouterAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for accept offer, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	acceptPayload := providerContractJSONMap(t, resp)
	if acceptPayload["assignmentId"] == "" {
		t.Fatalf("expected assignment id, got %#v", acceptPayload)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"/workers/worker-b/offers/"+offerB+":decline?api-version="+azureCommunicationJobRouterAPIVersion, []byte(`{
		"retryOfferAt":"2026-04-12T15:00:00Z"
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for decline offer, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"/jobs/job-1:unassign?api-version="+azureCommunicationJobRouterAPIVersion, []byte(`{}`), jsonHeaders)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for unassign job, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/jobs/job-1?api-version="+azureCommunicationJobRouterAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for get job-1 after unassign, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	jobOneAfterUnassign := providerContractJSONMap(t, resp)
	offerA = ""
	for _, raw := range jobOneAfterUnassign["offers"].([]any) {
		offer := providerContractJSONMapFromAny(t, raw)
		if offer["workerId"] == "worker-a" && offer["status"] == "pending" {
			offerA = offer["id"].(string)
		}
	}
	if offerA == "" {
		t.Fatalf("expected renewed pending offer for worker-a, got %#v", jobOneAfterUnassign)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"/workers/worker-a/offers/"+offerA+":accept?api-version="+azureCommunicationJobRouterAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for accept offer second time, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"/jobs/job-1:complete?api-version="+azureCommunicationJobRouterAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for complete job, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"/jobs/job-1:close?api-version="+azureCommunicationJobRouterAPIVersion, []byte(`{
		"dispositionCode":"closed-by-test"
	}`), jsonHeaders)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for close job, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"/jobs/job-2:cancel?api-version="+azureCommunicationJobRouterAPIVersion, []byte(`{
		"dispositionCode":"cancelled-by-test"
	}`), jsonHeaders)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for cancel job, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"/jobs/job-3:reclassify?api-version="+azureCommunicationJobRouterAPIVersion, []byte(`{
		"queueId":"queue-a",
		"requestedWorkerSelectors":[{"key":"lang","labelOperator":"equal","value":"en"}]
	}`), jsonHeaders)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for reclassify job, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodDelete, base+"/jobs/job-3?api-version="+azureCommunicationJobRouterAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204 for delete job-3, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodDelete, base+"/workers/worker-b?api-version="+azureCommunicationJobRouterAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204 for delete worker-b, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodDelete, base+"/workers/worker-a?api-version="+azureCommunicationJobRouterAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204 for delete worker-a, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	for i := len(adminCases) - 1; i >= 0; i-- {
		tc := adminCases[i]
		path := fmt.Sprintf("%s/%s/%s?api-version=%s", base, tc.family, tc.id, azureCommunicationJobRouterAPIVersion)
		resp = providerContractRequest(t, ts, http.MethodDelete, path, nil, headers)
		if resp.StatusCode != http.StatusNoContent {
			t.Fatalf("expected 204 for delete %s, got %d body=%s", tc.family, resp.StatusCode, string(providerContractBody(t, resp)))
		}
	}
}

func TestAzureCommunicationJobRouterValidation(t *testing.T) {
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
	}
	jsonHeaders := map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
		"Content-Type":              "application/merge-patch+json",
	}

	tests := []struct {
		name       string
		method     string
		path       string
		body       []byte
		headers    map[string]string
		wantStatus int
		wantCode   string
	}{
		{
			name:       "invalid api version",
			method:     http.MethodGet,
			path:       "/azure/communication-jobrouter/2023-11-01/routing/jobs?api-version=2023-01-01",
			headers:    headers,
			wantStatus: http.StatusBadRequest,
			wantCode:   "InvalidApiVersion",
		},
		{
			name:       "invalid json",
			method:     http.MethodPatch,
			path:       "/azure/communication-jobrouter/2023-11-01/routing/queues/queue-a?api-version=2023-11-01",
			body:       []byte(`{`),
			headers:    jsonHeaders,
			wantStatus: http.StatusBadRequest,
			wantCode:   "InvalidRequest",
		},
		{
			name:       "invalid queue payload",
			method:     http.MethodPatch,
			path:       "/azure/communication-jobrouter/2023-11-01/routing/queues/queue-a?api-version=2023-11-01",
			body:       []byte(`{"labels":{}}`),
			headers:    jsonHeaders,
			wantStatus: http.StatusBadRequest,
			wantCode:   "InvalidRequest",
		},
		{
			name:       "invalid distribution policy payload",
			method:     http.MethodPatch,
			path:       "/azure/communication-jobrouter/2023-11-01/routing/distributionPolicies/dist-a?api-version=2023-11-01",
			body:       []byte(`{"name":"dist-a"}`),
			headers:    jsonHeaders,
			wantStatus: http.StatusBadRequest,
			wantCode:   "InvalidRequest",
		},
		{
			name:       "invalid exception policy payload",
			method:     http.MethodPatch,
			path:       "/azure/communication-jobrouter/2023-11-01/routing/exceptionPolicies/exception-a?api-version=2023-11-01",
			body:       []byte(`{"name":"exception-a"}`),
			headers:    jsonHeaders,
			wantStatus: http.StatusBadRequest,
			wantCode:   "InvalidRequest",
		},
		{
			name:       "invalid job payload",
			method:     http.MethodPatch,
			path:       "/azure/communication-jobrouter/2023-11-01/routing/jobs/job-a?api-version=2023-11-01",
			body:       []byte(`{"labels":{"topic":"sales"}}`),
			headers:    jsonHeaders,
			wantStatus: http.StatusBadRequest,
			wantCode:   "InvalidRequest",
		},
		{
			name:       "invalid worker payload",
			method:     http.MethodPatch,
			path:       "/azure/communication-jobrouter/2023-11-01/routing/workers/worker-a?api-version=2023-11-01",
			body:       []byte(`{"tags":{"team":"sales"}}`),
			headers:    jsonHeaders,
			wantStatus: http.StatusBadRequest,
			wantCode:   "InvalidRequest",
		},
		{
			name:       "unknown job",
			method:     http.MethodGet,
			path:       "/azure/communication-jobrouter/2023-11-01/routing/jobs/missing?api-version=2023-11-01",
			headers:    headers,
			wantStatus: http.StatusNotFound,
			wantCode:   "JobNotFound",
		},
		{
			name:       "unknown worker",
			method:     http.MethodGet,
			path:       "/azure/communication-jobrouter/2023-11-01/routing/workers/missing?api-version=2023-11-01",
			headers:    headers,
			wantStatus: http.StatusNotFound,
			wantCode:   "WorkerNotFound",
		},
		{
			name:       "unknown queue statistics",
			method:     http.MethodGet,
			path:       "/azure/communication-jobrouter/2023-11-01/routing/queues/missing/statistics?api-version=2023-11-01",
			headers:    headers,
			wantStatus: http.StatusNotFound,
			wantCode:   "QueueNotFound",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp := providerContractRequest(t, ts, tc.method, tc.path, tc.body, tc.headers)
			if resp.StatusCode != tc.wantStatus {
				t.Fatalf("expected %d, got %d body=%s", tc.wantStatus, resp.StatusCode, string(providerContractBody(t, resp)))
			}
			payload := providerContractJSONMap(t, resp)
			errPayload := providerContractJSONMapFromAny(t, payload["error"])
			if errPayload["code"] != tc.wantCode {
				t.Fatalf("expected error code %q, got %#v", tc.wantCode, payload)
			}
		})
	}

	resp := providerContractRequest(t, ts, http.MethodPatch, "/azure/communication-jobrouter/2023-11-01/routing/queues/queue-a?api-version=2023-11-01", []byte(`{
		"name":"queue-a",
		"distributionPolicyId":"dist-a"
	}`), jsonHeaders)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 for queue setup, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	resp = providerContractRequest(t, ts, http.MethodPatch, "/azure/communication-jobrouter/2023-11-01/routing/workers/worker-a?api-version=2023-11-01", []byte(`{
		"queueIds":["queue-a"],
		"capacity":1
	}`), jsonHeaders)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 for worker setup, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	resp = providerContractRequest(t, ts, http.MethodPatch, "/azure/communication-jobrouter/2023-11-01/routing/jobs/job-a?api-version=2023-11-01", []byte(`{
		"channelId":"chat",
		"queueId":"queue-a"
	}`), jsonHeaders)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 for job setup, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	jobPayload := providerContractJSONMap(t, providerContractRequest(t, ts, http.MethodGet, "/azure/communication-jobrouter/2023-11-01/routing/jobs/job-a?api-version=2023-11-01", nil, headers))
	offer := providerContractJSONMapFromAny(t, jobPayload["offers"].([]any)[0])
	offerID := offer["id"].(string)

	resp = providerContractRequest(t, ts, http.MethodPost, "/azure/communication-jobrouter/2023-11-01/routing/workers/worker-a/offers/"+offerID+":accept?api-version=2023-11-01", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for initial accept, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	resp = providerContractRequest(t, ts, http.MethodPost, "/azure/communication-jobrouter/2023-11-01/routing/workers/worker-a/offers/"+offerID+":accept?api-version=2023-11-01", nil, headers)
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409 for duplicate accept, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
}
