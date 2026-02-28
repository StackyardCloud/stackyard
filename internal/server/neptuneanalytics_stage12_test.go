package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func neptuneAnalyticsRequest(t *testing.T, ts *httptest.Server, method, path string, body []byte) *http.Response {
	t.Helper()
	headers := map[string]string{
		"Accept": "application/json",
	}
	if len(body) > 0 {
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "neptune-graph")
}

func neptuneAnalyticsCall(t *testing.T, ts *httptest.Server, method, path string, body []byte) (int, string) {
	t.Helper()
	resp := neptuneAnalyticsRequest(t, ts, method, path, body)
	return resp.StatusCode, string(mustBody(t, resp))
}

func neptuneAnalyticsJSONField(payload, key string) string {
	marker := `"` + key + `":"`
	start := strings.Index(payload, marker)
	if start == -1 {
		return ""
	}
	start += len(marker)
	end := strings.Index(payload[start:], `"`)
	if end == -1 {
		return ""
	}
	return payload[start : start+end]
}

func TestNeptuneAnalyticsStage1GraphLifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	status, body := neptuneAnalyticsCall(
		t,
		ts,
		http.MethodPost,
		"/graphs",
		mustJSON(t, map[string]any{
			"graphName":          "stackyard-neptuneanalytics-stage1",
			"provisionedMemory":  128,
			"publicConnectivity": true,
		}),
	)
	if status != http.StatusOK {
		t.Fatalf("expected CreateGraph 200, got %d: %s", status, body)
	}
	graphID := neptuneAnalyticsJSONField(body, "id")
	if graphID == "" {
		t.Fatalf("expected graph id in create response: %s", body)
	}

	status, body = neptuneAnalyticsCall(t, ts, http.MethodGet, "/graphs/"+graphID, nil)
	if status != http.StatusOK {
		t.Fatalf("expected GetGraph 200, got %d: %s", status, body)
	}
	if !strings.Contains(body, `"name":"stackyard-neptuneanalytics-stage1"`) {
		t.Fatalf("expected created graph name in get response: %s", body)
	}

	status, body = neptuneAnalyticsCall(t, ts, http.MethodGet, "/graphs?maxResults=10", nil)
	if status != http.StatusOK {
		t.Fatalf("expected ListGraphs 200, got %d: %s", status, body)
	}
	if !strings.Contains(body, `"graphs"`) || !strings.Contains(body, graphID) {
		t.Fatalf("expected graph in list response: %s", body)
	}

	status, body = neptuneAnalyticsCall(
		t,
		ts,
		http.MethodPatch,
		"/graphs/"+graphID,
		mustJSON(t, map[string]any{
			"publicConnectivity": false,
			"deletionProtection": true,
			"provisionedMemory":  256,
		}),
	)
	if status != http.StatusOK {
		t.Fatalf("expected UpdateGraph 200, got %d: %s", status, body)
	}
	if !strings.Contains(body, `"provisionedMemory":256`) || !strings.Contains(body, `"deletionProtection":true`) {
		t.Fatalf("expected updated graph attributes in response: %s", body)
	}

	status, body = neptuneAnalyticsCall(t, ts, http.MethodPost, "/graphs/"+graphID+"/stop", nil)
	if status != http.StatusOK {
		t.Fatalf("expected StopGraph 200, got %d: %s", status, body)
	}
	if !strings.Contains(body, `"status":"STOPPED"`) {
		t.Fatalf("expected STOPPED graph status: %s", body)
	}

	status, body = neptuneAnalyticsCall(t, ts, http.MethodPost, "/graphs/"+graphID+"/start", nil)
	if status != http.StatusOK {
		t.Fatalf("expected StartGraph 200, got %d: %s", status, body)
	}
	if !strings.Contains(body, `"status":"AVAILABLE"`) {
		t.Fatalf("expected AVAILABLE graph status: %s", body)
	}

	status, body = neptuneAnalyticsCall(t, ts, http.MethodDelete, "/graphs/"+graphID+"?skipSnapshot=true", nil)
	if status != http.StatusConflict {
		t.Fatalf("expected DeleteGraph conflict with deletionProtection, got %d: %s", status, body)
	}

	status, body = neptuneAnalyticsCall(
		t,
		ts,
		http.MethodPatch,
		"/graphs/"+graphID,
		mustJSON(t, map[string]any{
			"deletionProtection": false,
		}),
	)
	if status != http.StatusOK {
		t.Fatalf("expected UpdateGraph(deletionProtection=false) 200, got %d: %s", status, body)
	}

	status, body = neptuneAnalyticsCall(t, ts, http.MethodDelete, "/graphs/"+graphID+"?skipSnapshot=true", nil)
	if status != http.StatusOK {
		t.Fatalf("expected DeleteGraph 200, got %d: %s", status, body)
	}

	status, body = neptuneAnalyticsCall(t, ts, http.MethodGet, "/graphs/"+graphID, nil)
	if status != http.StatusNotFound {
		t.Fatalf("expected GetGraph after delete 404, got %d: %s", status, body)
	}
}

func TestNeptuneAnalyticsStage2SnapshotAndRestore(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	status, body := neptuneAnalyticsCall(
		t,
		ts,
		http.MethodPost,
		"/graphs",
		mustJSON(t, map[string]any{
			"graphName":         "stackyard-neptuneanalytics-stage2-source",
			"provisionedMemory": 128,
		}),
	)
	if status != http.StatusOK {
		t.Fatalf("expected CreateGraph 200, got %d: %s", status, body)
	}
	graphID := neptuneAnalyticsJSONField(body, "id")
	if graphID == "" {
		t.Fatalf("expected graph id in create response: %s", body)
	}

	status, body = neptuneAnalyticsCall(
		t,
		ts,
		http.MethodPost,
		"/snapshots",
		mustJSON(t, map[string]any{
			"graphIdentifier": graphID,
			"snapshotName":    "stackyard-neptuneanalytics-stage2-snapshot",
		}),
	)
	if status != http.StatusOK {
		t.Fatalf("expected CreateGraphSnapshot 200, got %d: %s", status, body)
	}
	snapshotID := neptuneAnalyticsJSONField(body, "id")
	if snapshotID == "" {
		t.Fatalf("expected snapshot id in create response: %s", body)
	}

	status, body = neptuneAnalyticsCall(t, ts, http.MethodGet, "/snapshots/"+snapshotID, nil)
	if status != http.StatusOK {
		t.Fatalf("expected GetGraphSnapshot 200, got %d: %s", status, body)
	}
	if !strings.Contains(body, `"sourceGraphId":"`+graphID+`"`) {
		t.Fatalf("expected source graph id in snapshot response: %s", body)
	}

	status, body = neptuneAnalyticsCall(t, ts, http.MethodGet, "/snapshots?graphIdentifier="+graphID+"&maxResults=10", nil)
	if status != http.StatusOK {
		t.Fatalf("expected ListGraphSnapshots 200, got %d: %s", status, body)
	}
	if !strings.Contains(body, `"graphSnapshots"`) || !strings.Contains(body, snapshotID) {
		t.Fatalf("expected snapshot in list response: %s", body)
	}

	status, body = neptuneAnalyticsCall(
		t,
		ts,
		http.MethodPost,
		"/snapshots/"+snapshotID+"/restore",
		mustJSON(t, map[string]any{
			"graphName":         "stackyard-neptuneanalytics-stage2-restored",
			"provisionedMemory": 128,
		}),
	)
	if status != http.StatusOK {
		t.Fatalf("expected RestoreGraphFromSnapshot 200, got %d: %s", status, body)
	}
	if !strings.Contains(body, `"sourceSnapshotId":"`+snapshotID+`"`) {
		t.Fatalf("expected restored graph source snapshot id: %s", body)
	}
	restoredGraphID := neptuneAnalyticsJSONField(body, "id")
	if restoredGraphID == "" {
		t.Fatalf("expected restored graph id in response: %s", body)
	}

	status, body = neptuneAnalyticsCall(t, ts, http.MethodGet, "/graphs/"+restoredGraphID, nil)
	if status != http.StatusOK {
		t.Fatalf("expected restored graph get 200, got %d: %s", status, body)
	}

	status, body = neptuneAnalyticsCall(t, ts, http.MethodDelete, "/snapshots/"+snapshotID, nil)
	if status != http.StatusOK {
		t.Fatalf("expected DeleteGraphSnapshot 200, got %d: %s", status, body)
	}

	status, body = neptuneAnalyticsCall(t, ts, http.MethodGet, "/snapshots/"+snapshotID, nil)
	if status != http.StatusNotFound {
		t.Fatalf("expected GetGraphSnapshot after delete 404, got %d: %s", status, body)
	}
}

func TestNeptuneAnalyticsStage12ImplementedRoutesDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	createStatus, createBody := neptuneAnalyticsCall(
		t,
		ts,
		http.MethodPost,
		"/graphs",
		mustJSON(t, map[string]any{
			"graphName":         "stackyard-neptuneanalytics-stage12",
			"provisionedMemory": 128,
		}),
	)
	if createStatus != http.StatusOK {
		t.Fatalf("expected CreateGraph 200, got %d: %s", createStatus, createBody)
	}
	graphID := neptuneAnalyticsJSONField(createBody, "id")
	if graphID == "" {
		t.Fatalf("expected graph id from create response: %s", createBody)
	}

	snapshotStatus, snapshotBody := neptuneAnalyticsCall(
		t,
		ts,
		http.MethodPost,
		"/snapshots",
		mustJSON(t, map[string]any{
			"graphIdentifier": graphID,
			"snapshotName":    "stackyard-neptuneanalytics-stage12-snapshot",
		}),
	)
	if snapshotStatus != http.StatusOK {
		t.Fatalf("expected CreateGraphSnapshot 200, got %d: %s", snapshotStatus, snapshotBody)
	}
	snapshotID := neptuneAnalyticsJSONField(snapshotBody, "id")
	if snapshotID == "" {
		t.Fatalf("expected snapshot id from create response: %s", snapshotBody)
	}

	cases := []struct {
		method string
		path   string
		body   []byte
	}{
		{method: http.MethodGet, path: "/graphs"},
		{method: http.MethodGet, path: "/graphs/" + graphID},
		{method: http.MethodPatch, path: "/graphs/" + graphID, body: mustJSON(t, map[string]any{"publicConnectivity": true})},
		{method: http.MethodPost, path: "/graphs/" + graphID + "/stop"},
		{method: http.MethodPost, path: "/graphs/" + graphID + "/start"},
		{method: http.MethodGet, path: "/snapshots"},
		{method: http.MethodGet, path: "/snapshots/" + snapshotID},
		{
			method: http.MethodPost,
			path:   "/snapshots/" + snapshotID + "/restore",
			body:   mustJSON(t, map[string]any{"graphName": "stackyard-neptuneanalytics-stage12-restored", "provisionedMemory": 128}),
		},
		{method: http.MethodDelete, path: "/snapshots/" + snapshotID},
	}

	for _, tc := range cases {
		status, body := neptuneAnalyticsCall(t, ts, tc.method, tc.path, tc.body)
		if status == http.StatusNotImplemented || strings.Contains(body, "NotImplementedException") {
			t.Fatalf("%s %s unexpectedly returned not implemented: status=%d body=%s", tc.method, tc.path, status, body)
		}
	}

	// Ensure JSON body is parseable for one representative response shape.
	var listOut map[string]any
	listStatus, listBody := neptuneAnalyticsCall(t, ts, http.MethodGet, "/graphs", nil)
	if listStatus != http.StatusOK {
		t.Fatalf("expected ListGraphs 200, got %d: %s", listStatus, listBody)
	}
	if err := json.Unmarshal([]byte(listBody), &listOut); err != nil {
		t.Fatalf("expected valid JSON list response: %v; body=%s", err, listBody)
	}
}
