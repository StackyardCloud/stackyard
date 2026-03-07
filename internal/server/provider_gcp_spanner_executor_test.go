package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPSpannerExecutorRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPSpannerExecutorContractServer(t)

	notImplementedResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/google.spanner.executor.v1.SpannerExecutorProxy/Unknown", []byte(`{"actionId":1,"action":{"start":{}}}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "spanner-executor",
	})
	if notImplementedResp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp spanner executor unknown route, got %d body=%s", notImplementedResp.StatusCode, string(providerContractBody(t, notImplementedResp)))
	}

	assertGCPSpannerExecutorSuccess(t, ts, http.MethodPost, "/gcp/google.spanner.executor.v1.SpannerExecutorProxy/ExecuteActionAsync", []byte(`{
		"actionId": 1,
		"action": {
			"databasePath": "projects/stackyard/instances/stackyard-instance/databases/stackyard-db",
			"start": {}
		}
	}`), `"code":0`)

	assertGCPSpannerExecutorSuccess(t, ts, http.MethodPost, "/gcp/google.spanner.executor.v1.SpannerExecutorProxy/ExecuteActionAsync", []byte(`{
		"actionId": 2,
		"action": {
			"databasePath": "projects/stackyard/instances/stackyard-instance/databases/stackyard-db",
			"read": {
				"table": "Users",
				"column": ["id", "name"],
				"keys": {"all": true}
			}
		}
	}`), `"readResult"`)

	assertGCPSpannerExecutorSuccess(t, ts, http.MethodPost, "/gcp/google.spanner.executor.v1.SpannerExecutorProxy/ExecuteActionAsync", []byte(`{
		"actionId": 3,
		"action": {
			"query": {
				"sql": "SELECT 1"
			}
		}
	}`), `"queryResult"`)

	assertGCPSpannerExecutorSuccess(t, ts, http.MethodPost, "/gcp/google.spanner.executor.v1.SpannerExecutorProxy/ExecuteActionAsync", []byte(`{
		"actionId": 4,
		"action": {
			"admin": {
				"listCloudInstances": {}
			}
		}
	}`), `"adminResult"`)

	assertGCPSpannerExecutorSuccess(t, ts, http.MethodPost, "/gcp/google.spanner.executor.v1.SpannerExecutorProxy/ExecuteActionAsync", []byte(`{
		"actionId": 5,
		"action": {
			"generateDbPartitionsQuery": {
				"query": {
					"sql": "SELECT * FROM Users"
				}
			}
		}
	}`), `"dbPartition"`)

	assertGCPSpannerExecutorSuccess(t, ts, http.MethodPost, "/gcp/google.spanner.executor.v1.SpannerExecutorProxy/ExecuteActionAsync", []byte(`{
		"actionId": 6,
		"action": {
			"executePartition": {
				"partition": {
					"partitionToken": "dG9rZW4tMQ==",
					"table": "Users"
				}
			}
		}
	}`), `"readResult"`)

	assertGCPSpannerExecutorSuccess(t, ts, http.MethodPost, "/gcp/google.spanner.executor.v1.SpannerExecutorProxy/ExecuteActionAsync", []byte(`{
		"actionId": 7,
		"action": {
			"executeChangeStreamQuery": {
				"name": "projects/stackyard/instances/stackyard-instance/databases/stackyard-db/changeStreams/users",
				"startTime": "2026-01-01T00:00:00Z"
			}
		}
	}`), `"changeStreamRecords"`)

	assertGCPSpannerExecutorSuccess(t, ts, http.MethodPost, "/gcp/google.spanner.executor.v1.SpannerExecutorProxy/ExecuteActionAsync", []byte(`{
		"actionId": 8,
		"action": {
			"queryCancellation": {
				"longRunningSql": "SELECT * FROM Users",
				"cancelQuery": "CANCEL QUERY ?"
			}
		}
	}`), `"queryResult"`)
}

func TestGCPSpannerExecutorRouter_InvalidJSONReturnsBadRequest(t *testing.T) {
	t.Parallel()

	ts := newGCPSpannerExecutorContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/google.spanner.executor.v1.SpannerExecutorProxy/ExecuteActionAsync", []byte(`{"actionId"`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "spanner-executor",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp spanner executor router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSpannerExecutorRouter_ActionIDRequired(t *testing.T) {
	t.Parallel()

	ts := newGCPSpannerExecutorContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/google.spanner.executor.v1.SpannerExecutorProxy/ExecuteActionAsync", []byte(`{"action":{"start":{}}}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "spanner-executor",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp spanner executor router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSpannerExecutorRouter_ActionRequired(t *testing.T) {
	t.Parallel()

	ts := newGCPSpannerExecutorContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/google.spanner.executor.v1.SpannerExecutorProxy/ExecuteActionAsync", []byte(`{"actionId":1}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "spanner-executor",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp spanner executor router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSpannerExecutorRouter_ActionMustSetSingleBranch(t *testing.T) {
	t.Parallel()

	ts := newGCPSpannerExecutorContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/google.spanner.executor.v1.SpannerExecutorProxy/ExecuteActionAsync", []byte(`{"actionId":1,"action":{"start":{},"query":{"sql":"SELECT 1"}}}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "spanner-executor",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp spanner executor router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSpannerExecutorRouter_ReadRequiresTable(t *testing.T) {
	t.Parallel()

	ts := newGCPSpannerExecutorContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/google.spanner.executor.v1.SpannerExecutorProxy/ExecuteActionAsync", []byte(`{"actionId":2,"action":{"read":{"column":["id"],"keys":{"all":true}}}}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "spanner-executor",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp spanner executor router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSpannerExecutorRouter_StartBatchTxnRequiresParam(t *testing.T) {
	t.Parallel()

	ts := newGCPSpannerExecutorContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/google.spanner.executor.v1.SpannerExecutorProxy/ExecuteActionAsync", []byte(`{"actionId":3,"action":{"startBatchTxn":{}}}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "spanner-executor",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp spanner executor router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSpannerExecutorRouter_DatabaseNotFound(t *testing.T) {
	t.Parallel()

	ts := newGCPSpannerExecutorContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/google.spanner.executor.v1.SpannerExecutorProxy/ExecuteActionAsync", []byte(`{
		"actionId": 4,
		"action": {
			"databasePath": "projects/stackyard/instances/stackyard-instance/databases/missing-db",
			"query": {"sql":"SELECT 1"}
		}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "spanner-executor",
	})
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 from gcp spanner executor router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"NotFound"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSpannerExecutorRouter_QueryCancellationAlreadyCancelled(t *testing.T) {
	t.Parallel()

	ts := newGCPSpannerExecutorContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/google.spanner.executor.v1.SpannerExecutorProxy/ExecuteActionAsync", []byte(`{
		"actionId": 5,
		"action": {
			"queryCancellation": {
				"longRunningSql": "already-cancelled query",
				"cancelQuery": "CANCEL QUERY ?"
			}
		}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "spanner-executor",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp spanner executor router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"FailedPrecondition"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSpannerExecutorRouter_AdminCreateAlreadyExists(t *testing.T) {
	t.Parallel()

	ts := newGCPSpannerExecutorContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/google.spanner.executor.v1.SpannerExecutorProxy/ExecuteActionAsync", []byte(`{
		"actionId": 6,
		"action": {
			"admin": {
				"createCloudInstance": {
					"instance": {
						"name": "projects/stackyard/instances/existing-instance"
					}
				}
			}
		}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "spanner-executor",
	})
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409 from gcp spanner executor router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"AlreadyExists"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSpannerExecutorRouter_OutputShapes(t *testing.T) {
	t.Parallel()

	ts := newGCPSpannerExecutorContractServer(t)

	readResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/google.spanner.executor.v1.SpannerExecutorProxy/ExecuteActionAsync", []byte(`{
		"actionId": 10,
		"action": {
			"read": {
				"table": "Users",
				"column": ["id", "name"],
				"keys": {"all": true}
			}
		}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "spanner-executor",
	})
	if readResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from read action, got %d body=%s", readResp.StatusCode, string(providerContractBody(t, readResp)))
	}
	readBody := providerContractJSONMap(t, readResp)
	if _, ok := readBody["actionId"].(float64); !ok {
		t.Fatalf("expected actionId number, got %#v", readBody["actionId"])
	}
	readOutcome, _ := readBody["outcome"].(map[string]any)
	readStatus, _ := readOutcome["status"].(map[string]any)
	if _, ok := readStatus["code"].(float64); !ok {
		t.Fatalf("expected outcome.status.code number, got %#v", readStatus["code"])
	}
	readResult, _ := readOutcome["readResult"].(map[string]any)
	if _, ok := readResult["table"].(string); !ok {
		t.Fatalf("expected readResult.table string, got %#v", readResult["table"])
	}

	adminResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/google.spanner.executor.v1.SpannerExecutorProxy/ExecuteActionAsync", []byte(`{
		"actionId": 11,
		"action": {
			"admin": {
				"listCloudInstances": {}
			}
		}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "spanner-executor",
	})
	if adminResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from admin action, got %d body=%s", adminResp.StatusCode, string(providerContractBody(t, adminResp)))
	}
	adminBody := providerContractJSONMap(t, adminResp)
	adminOutcome, _ := adminBody["outcome"].(map[string]any)
	adminResult, _ := adminOutcome["adminResult"].(map[string]any)
	instanceResponse, _ := adminResult["instanceResponse"].(map[string]any)
	listedInstances, _ := instanceResponse["listedInstances"].([]any)
	if len(listedInstances) == 0 {
		t.Fatalf("expected listedInstances array, got %#v", instanceResponse["listedInstances"])
	}
	firstInstance, _ := listedInstances[0].(map[string]any)
	if _, ok := firstInstance["name"].(string); !ok {
		t.Fatalf("expected listedInstances[0].name string, got %#v", firstInstance["name"])
	}

	dmlResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/google.spanner.executor.v1.SpannerExecutorProxy/ExecuteActionAsync", []byte(`{
		"actionId": 12,
		"action": {
			"dml": {
				"update": {
					"sql": "UPDATE Users SET name = 'Stackyard' WHERE id = '1'"
				}
			}
		}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "spanner-executor",
	})
	if dmlResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from dml action, got %d body=%s", dmlResp.StatusCode, string(providerContractBody(t, dmlResp)))
	}
	dmlBody := providerContractJSONMap(t, dmlResp)
	dmlOutcome, _ := dmlBody["outcome"].(map[string]any)
	dmlRows, _ := dmlOutcome["dmlRowsModified"].([]any)
	if len(dmlRows) == 0 {
		t.Fatalf("expected dmlRowsModified array, got %#v", dmlOutcome["dmlRowsModified"])
	}

	partitionResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/google.spanner.executor.v1.SpannerExecutorProxy/ExecuteActionAsync", []byte(`{
		"actionId": 13,
		"action": {
			"generateDbPartitionsQuery": {
				"query": {
					"sql": "SELECT * FROM Users"
				}
			}
		}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "spanner-executor",
	})
	if partitionResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from generate db partitions action, got %d body=%s", partitionResp.StatusCode, string(providerContractBody(t, partitionResp)))
	}
	partitionBody := providerContractJSONMap(t, partitionResp)
	partitionOutcome, _ := partitionBody["outcome"].(map[string]any)
	dbPartitions, _ := partitionOutcome["dbPartition"].([]any)
	if len(dbPartitions) == 0 {
		t.Fatalf("expected dbPartition array, got %#v", partitionOutcome["dbPartition"])
	}
	firstPartition, _ := dbPartitions[0].(map[string]any)
	if _, ok := firstPartition["partitionToken"].(string); !ok {
		t.Fatalf("expected dbPartition[0].partitionToken string, got %#v", firstPartition["partitionToken"])
	}

	changeStreamResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/google.spanner.executor.v1.SpannerExecutorProxy/ExecuteActionAsync", []byte(`{
		"actionId": 14,
		"action": {
			"executeChangeStreamQuery": {
				"name": "projects/stackyard/instances/stackyard-instance/databases/stackyard-db/changeStreams/users",
				"startTime": "2026-01-01T00:00:00Z"
			}
		}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "spanner-executor",
	})
	if changeStreamResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from execute change stream action, got %d body=%s", changeStreamResp.StatusCode, string(providerContractBody(t, changeStreamResp)))
	}
	changeStreamBody := providerContractJSONMap(t, changeStreamResp)
	changeStreamOutcome, _ := changeStreamBody["outcome"].(map[string]any)
	changeStreamRecords, _ := changeStreamOutcome["changeStreamRecords"].([]any)
	if len(changeStreamRecords) == 0 {
		t.Fatalf("expected changeStreamRecords array, got %#v", changeStreamOutcome["changeStreamRecords"])
	}
}

func TestGCPSpannerExecutorRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/spanner_executor?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp spanner executor contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "spanner_executor" {
		t.Fatalf("expected service=spanner_executor, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}

func newGCPSpannerExecutorContractServer(t *testing.T) *httptest.Server {
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

func assertGCPSpannerExecutorSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{
		"X-Stackyard-GCP-Service": "spanner-executor",
	}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp spanner executor router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}
