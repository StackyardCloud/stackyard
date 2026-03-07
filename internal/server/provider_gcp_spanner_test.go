package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPSpannerRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPSpannerContractServer(t)

	database := "/gcp/v1/projects/stackyard/instances/stackyard-instance/databases/stackyard-db"
	session := database + "/sessions/s-1"

	assertGCPSpannerSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations?pageSize=1", nil, "locations")
	assertGCPSpannerSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1", nil, "us-central1")

	assertGCPSpannerSuccess(t, ts, http.MethodPost, database+"/sessions", []byte(`{"session":{"labels":{"env":"test"}}}`), "/sessions/s-1")
	assertGCPSpannerSuccess(t, ts, http.MethodPost, database+"/sessions:batchCreate", []byte(`{"sessionCount":2,"sessionTemplate":{"labels":{"env":"test"}}}`), `"session"`)
	assertGCPSpannerSuccess(t, ts, http.MethodGet, session, nil, "/sessions/s-1")
	assertGCPSpannerSuccess(t, ts, http.MethodGet, database+"/sessions?pageSize=1", nil, `"sessions"`)
	assertGCPSpannerSuccess(t, ts, http.MethodDelete, session, nil, "{}")

	assertGCPSpannerSuccess(t, ts, http.MethodPost, session+":executeSql", []byte(`{"sql":"SELECT 1","transaction":{"singleUse":{"readOnly":{"strong":true}}}}`), `"rows"`)
	assertGCPSpannerSuccess(t, ts, http.MethodPost, session+":executeStreamingSql", []byte(`{"sql":"SELECT 1","transaction":{"singleUse":{"readOnly":{"strong":true}}}}`), `"values"`)
	assertGCPSpannerSuccess(t, ts, http.MethodPost, session+":executeBatchDml", []byte(`{"transaction":{"begin":{"readWrite":{}}},"statements":[{"sql":"UPDATE Users SET value='ok' WHERE id=1"}]}`), `"resultSets"`)
	assertGCPSpannerSuccess(t, ts, http.MethodPost, session+":read", []byte(`{"table":"Users","columns":["UserId","Name"],"keySet":{"all":true},"transaction":{"singleUse":{"readOnly":{"strong":true}}}}`), `"rows"`)
	assertGCPSpannerSuccess(t, ts, http.MethodPost, session+":streamingRead", []byte(`{"table":"Users","columns":["UserId","Name"],"keySet":{"all":true},"transaction":{"singleUse":{"readOnly":{"strong":true}}}}`), `"values"`)
	assertGCPSpannerSuccess(t, ts, http.MethodPost, session+":beginTransaction", []byte(`{"options":{"readWrite":{}}}`), `"id"`)
	assertGCPSpannerSuccess(t, ts, http.MethodPost, session+":commit", []byte(`{"transactionId":"dHgtcy0x","mutations":[{"insert":{"table":"Users","columns":["UserId"],"values":[["1"]]}}]}`), `"commitTimestamp"`)
	assertGCPSpannerSuccess(t, ts, http.MethodPost, session+":rollback", []byte(`{"transactionId":"dHgtcy0x"}`), "{}")
	assertGCPSpannerSuccess(t, ts, http.MethodPost, session+":partitionQuery", []byte(`{"sql":"SELECT * FROM Users","transaction":{"singleUse":{"readOnly":{"strong":true}}}}`), `"partitions"`)
	assertGCPSpannerSuccess(t, ts, http.MethodPost, session+":partitionRead", []byte(`{"table":"Users","columns":["UserId"],"keySet":{"all":true},"transaction":{"singleUse":{"readOnly":{"strong":true}}}}`), `"partitions"`)
	assertGCPSpannerSuccess(t, ts, http.MethodPost, session+":batchWrite", []byte(`{"mutationGroups":[{"mutations":[{"insert":{"table":"Users","columns":["UserId"],"values":[["1"]]}}]}]}`), `"indexes"`)
}

func TestGCPSpannerRouter_ListSessionsInvalidPageSize(t *testing.T) {
	t.Parallel()

	ts := newGCPSpannerContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/instances/stackyard-instance/databases/stackyard-db/sessions?pageSize=bad", nil, map[string]string{
		"X-Stackyard-GCP-Service": "spanner",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp spanner router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSpannerRouter_CreateSessionNameMustMatchDatabase(t *testing.T) {
	t.Parallel()

	ts := newGCPSpannerContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/instances/stackyard-instance/databases/stackyard-db/sessions", []byte(`{"session":{"name":"projects/stackyard/instances/other/databases/stackyard-db/sessions/s-1"}}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "spanner",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp spanner router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSpannerRouter_GetSessionNotFound(t *testing.T) {
	t.Parallel()

	ts := newGCPSpannerContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/instances/stackyard-instance/databases/stackyard-db/sessions/missing-session", nil, map[string]string{
		"X-Stackyard-GCP-Service": "spanner",
	})
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 from gcp spanner router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"NotFound"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSpannerRouter_ExecuteSQLRequiresStatement(t *testing.T) {
	t.Parallel()

	ts := newGCPSpannerContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/instances/stackyard-instance/databases/stackyard-db/sessions/s-1:executeSql", []byte(`{"transaction":{"singleUse":{"readOnly":{"strong":true}}}}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "spanner",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp spanner router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSpannerRouter_CommitStaleTransactionFailsPrecondition(t *testing.T) {
	t.Parallel()

	ts := newGCPSpannerContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/instances/stackyard-instance/databases/stackyard-db/sessions/s-1:commit", []byte(`{"transactionId":"tx-stale"}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "spanner",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp spanner router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"FailedPrecondition"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSpannerRouter_CommitAbortTransactionReturnsAborted(t *testing.T) {
	t.Parallel()

	ts := newGCPSpannerContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/instances/stackyard-instance/databases/stackyard-db/sessions/s-1:commit", []byte(`{"transactionId":"tx-abort"}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "spanner",
	})
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409 from gcp spanner router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"Aborted"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSpannerRouter_BatchWriteRequiresMutationGroups(t *testing.T) {
	t.Parallel()

	ts := newGCPSpannerContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/instances/stackyard-instance/databases/stackyard-db/sessions/s-1:batchWrite", []byte(`{"mutationGroups":[]}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "spanner",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp spanner router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSpannerRouter_OutputShapes(t *testing.T) {
	t.Parallel()

	ts := newGCPSpannerContractServer(t)
	database := "/gcp/v1/projects/stackyard/instances/stackyard-instance/databases/stackyard-db"
	session := database + "/sessions/s-1"

	listResp := providerContractRequest(t, ts, http.MethodGet, database+"/sessions?pageSize=1", nil, map[string]string{
		"X-Stackyard-GCP-Service": "spanner",
	})
	if listResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from list sessions, got %d body=%s", listResp.StatusCode, string(providerContractBody(t, listResp)))
	}
	listBody := providerContractJSONMap(t, listResp)
	sessions, ok := listBody["sessions"].([]any)
	if !ok || len(sessions) == 0 {
		t.Fatalf("expected sessions array, got %#v", listBody["sessions"])
	}
	firstSession, _ := sessions[0].(map[string]any)
	if _, ok := firstSession["name"].(string); !ok {
		t.Fatalf("expected sessions[0].name string, got %#v", firstSession["name"])
	}

	execResp := providerContractRequest(t, ts, http.MethodPost, session+":executeSql", []byte(`{"sql":"SELECT 1","transaction":{"singleUse":{"readOnly":{"strong":true}}}}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "spanner",
	})
	if execResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from executeSql, got %d body=%s", execResp.StatusCode, string(providerContractBody(t, execResp)))
	}
	execBody := providerContractJSONMap(t, execResp)
	metadata, _ := execBody["metadata"].(map[string]any)
	rowType, _ := metadata["rowType"].(map[string]any)
	fields, _ := rowType["fields"].([]any)
	if len(fields) == 0 {
		t.Fatalf("expected metadata.rowType.fields in executeSql response, got %#v", rowType["fields"])
	}
	rows, _ := execBody["rows"].([]any)
	if len(rows) == 0 {
		t.Fatalf("expected rows in executeSql response, got %#v", execBody["rows"])
	}

	streamResp := providerContractRequest(t, ts, http.MethodPost, session+":executeStreamingSql", []byte(`{"sql":"SELECT 1","transaction":{"singleUse":{"readOnly":{"strong":true}}}}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "spanner",
	})
	if streamResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from executeStreamingSql, got %d body=%s", streamResp.StatusCode, string(providerContractBody(t, streamResp)))
	}
	streamBody := providerContractJSONMap(t, streamResp)
	if values, _ := streamBody["values"].([]any); len(values) == 0 {
		t.Fatalf("expected values in streaming response, got %#v", streamBody["values"])
	}

	beginResp := providerContractRequest(t, ts, http.MethodPost, session+":beginTransaction", []byte(`{"options":{"readWrite":{}}}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "spanner",
	})
	if beginResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from beginTransaction, got %d body=%s", beginResp.StatusCode, string(providerContractBody(t, beginResp)))
	}
	beginBody := providerContractJSONMap(t, beginResp)
	if _, ok := beginBody["id"].(string); !ok {
		t.Fatalf("expected transaction id string, got %#v", beginBody["id"])
	}

	commitResp := providerContractRequest(t, ts, http.MethodPost, session+":commit", []byte(`{"transactionId":"dHgtcy0x","mutations":[{"insert":{"table":"Users","columns":["UserId"],"values":[["1"]]}}]}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "spanner",
	})
	if commitResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from commit, got %d body=%s", commitResp.StatusCode, string(providerContractBody(t, commitResp)))
	}
	commitBody := providerContractJSONMap(t, commitResp)
	if _, ok := commitBody["commitTimestamp"].(string); !ok {
		t.Fatalf("expected commitTimestamp string, got %#v", commitBody["commitTimestamp"])
	}
	commitStats, _ := commitBody["commitStats"].(map[string]any)
	if _, ok := commitStats["mutationCount"].(string); !ok {
		t.Fatalf("expected commitStats.mutationCount string, got %#v", commitStats["mutationCount"])
	}

	partitionResp := providerContractRequest(t, ts, http.MethodPost, session+":partitionQuery", []byte(`{"sql":"SELECT * FROM Users","transaction":{"singleUse":{"readOnly":{"strong":true}}}}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "spanner",
	})
	if partitionResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from partitionQuery, got %d body=%s", partitionResp.StatusCode, string(providerContractBody(t, partitionResp)))
	}
	partitionBody := providerContractJSONMap(t, partitionResp)
	partitions, _ := partitionBody["partitions"].([]any)
	if len(partitions) == 0 {
		t.Fatalf("expected partitions in partitionQuery response, got %#v", partitionBody["partitions"])
	}
	firstPartition, _ := partitions[0].(map[string]any)
	if _, ok := firstPartition["partitionToken"].(string); !ok {
		t.Fatalf("expected partitions[0].partitionToken string, got %#v", firstPartition["partitionToken"])
	}

	batchWriteResp := providerContractRequest(t, ts, http.MethodPost, session+":batchWrite", []byte(`{"mutationGroups":[{"mutations":[{"insert":{"table":"Users","columns":["UserId"],"values":[["1"]]}}]}]}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "spanner",
	})
	if batchWriteResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from batchWrite, got %d body=%s", batchWriteResp.StatusCode, string(providerContractBody(t, batchWriteResp)))
	}
	batchWriteBody := providerContractJSONMap(t, batchWriteResp)
	if indexes, _ := batchWriteBody["indexes"].([]any); len(indexes) == 0 {
		t.Fatalf("expected indexes in batchWrite response, got %#v", batchWriteBody["indexes"])
	}
	status, _ := batchWriteBody["status"].(map[string]any)
	if code, ok := status["code"].(float64); !ok || int(code) != 0 {
		t.Fatalf("expected status.code=0, got %#v", status["code"])
	}
}

func TestGCPSpannerRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/spanner?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp spanner contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "spanner" {
		t.Fatalf("expected service=spanner, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}

func newGCPSpannerContractServer(t *testing.T) *httptest.Server {
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

func assertGCPSpannerSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{
		"X-Stackyard-GCP-Service": "spanner",
	}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp spanner router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}
