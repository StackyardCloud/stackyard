package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func neptuneAnalyticsCallWithHeaders(
	t *testing.T,
	ts *httptest.Server,
	method, path string,
	body []byte,
	extraHeaders map[string]string,
) (int, string) {
	t.Helper()
	headers := map[string]string{
		"Accept": "application/json",
	}
	if len(body) > 0 {
		headers["Content-Type"] = "application/json"
	}
	for k, v := range extraHeaders {
		headers[k] = v
	}
	resp := signedRequestWithService(t, method, ts.URL+path, body, headers, "neptune-graph")
	return resp.StatusCode, string(mustBody(t, resp))
}

func neptuneAnalyticsDecodeJSONMap(t *testing.T, body string) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.Unmarshal([]byte(body), &out); err != nil {
		t.Fatalf("expected JSON object, got error %v: %s", err, body)
	}
	return out
}

func neptuneAnalyticsRequireStringField(t *testing.T, payload map[string]any, key string) string {
	t.Helper()
	raw, ok := payload[key]
	if !ok {
		t.Fatalf("missing field %q in payload: %+v", key, payload)
	}
	value, ok := raw.(string)
	if !ok || strings.TrimSpace(value) == "" {
		t.Fatalf("field %q is not a non-empty string: %#v", key, raw)
	}
	return value
}

func TestNeptuneAnalyticsStage3ImportExportLifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	status, body := neptuneAnalyticsCall(
		t,
		ts,
		http.MethodPost,
		"/graphs",
		mustJSON(t, map[string]any{
			"graphName":         "stackyard-neptuneanalytics-stage3",
			"provisionedMemory": 128,
		}),
	)
	if status != http.StatusOK {
		t.Fatalf("expected CreateGraph 200, got %d: %s", status, body)
	}
	createGraph := neptuneAnalyticsDecodeJSONMap(t, body)
	graphID := neptuneAnalyticsRequireStringField(t, createGraph, "id")

	status, body = neptuneAnalyticsCall(
		t,
		ts,
		http.MethodPost,
		"/graphs/"+graphID+"/importtasks",
		mustJSON(t, map[string]any{
			"source":  "s3://stackyard-neptuneanalytics-stage3/import",
			"roleArn": "arn:aws:iam::123456789012:role/stackyard-neptuneanalytics",
			"format":  "CSV",
		}),
	)
	if status != http.StatusOK {
		t.Fatalf("expected StartImportTask 200, got %d: %s", status, body)
	}
	startImport := neptuneAnalyticsDecodeJSONMap(t, body)
	importTaskID := neptuneAnalyticsRequireStringField(t, startImport, "taskId")

	status, body = neptuneAnalyticsCall(t, ts, http.MethodGet, "/importtasks/"+importTaskID, nil)
	if status != http.StatusOK {
		t.Fatalf("expected GetImportTask 200, got %d: %s", status, body)
	}
	if !strings.Contains(body, `"taskId":"`+importTaskID+`"`) {
		t.Fatalf("expected import task in GetImportTask response: %s", body)
	}

	status, body = neptuneAnalyticsCall(t, ts, http.MethodGet, "/importtasks?maxResults=10", nil)
	if status != http.StatusOK {
		t.Fatalf("expected ListImportTasks 200, got %d: %s", status, body)
	}
	if !strings.Contains(body, importTaskID) {
		t.Fatalf("expected import task id in ListImportTasks response: %s", body)
	}

	status, body = neptuneAnalyticsCall(t, ts, http.MethodDelete, "/importtasks/"+importTaskID, nil)
	if status != http.StatusOK {
		t.Fatalf("expected CancelImportTask 200, got %d: %s", status, body)
	}
	if !strings.Contains(body, `"status":"CANCELLED"`) {
		t.Fatalf("expected CANCELLED import task state: %s", body)
	}

	status, body = neptuneAnalyticsCall(
		t,
		ts,
		http.MethodPost,
		"/exporttasks",
		mustJSON(t, map[string]any{
			"graphIdentifier":  graphID,
			"roleArn":          "arn:aws:iam::123456789012:role/stackyard-neptuneanalytics",
			"format":           "CSV",
			"destination":      "s3://stackyard-neptuneanalytics-stage3/export",
			"kmsKeyIdentifier": "alias/aws/neptune-graph",
			"parquetType":      "COLUMNAR",
		}),
	)
	if status != http.StatusOK {
		t.Fatalf("expected StartExportTask 200, got %d: %s", status, body)
	}
	startExport := neptuneAnalyticsDecodeJSONMap(t, body)
	exportTaskID := neptuneAnalyticsRequireStringField(t, startExport, "taskId")

	status, body = neptuneAnalyticsCall(t, ts, http.MethodGet, "/exporttasks/"+exportTaskID, nil)
	if status != http.StatusOK {
		t.Fatalf("expected GetExportTask 200, got %d: %s", status, body)
	}
	if !strings.Contains(body, `"taskId":"`+exportTaskID+`"`) {
		t.Fatalf("expected export task in GetExportTask response: %s", body)
	}

	status, body = neptuneAnalyticsCall(t, ts, http.MethodGet, "/exporttasks?maxResults=10", nil)
	if status != http.StatusOK {
		t.Fatalf("expected ListExportTasks 200, got %d: %s", status, body)
	}
	if !strings.Contains(body, exportTaskID) {
		t.Fatalf("expected export task id in ListExportTasks response: %s", body)
	}

	status, body = neptuneAnalyticsCall(t, ts, http.MethodDelete, "/exporttasks/"+exportTaskID, nil)
	if status != http.StatusOK {
		t.Fatalf("expected CancelExportTask 200, got %d: %s", status, body)
	}
	if !strings.Contains(body, `"status":"CANCELLED"`) {
		t.Fatalf("expected CANCELLED export task state: %s", body)
	}

	status, body = neptuneAnalyticsCall(
		t,
		ts,
		http.MethodPost,
		"/importtasks",
		mustJSON(t, map[string]any{
			"graphName": "stackyard-neptuneanalytics-stage3-import-graph",
			"source":    "s3://stackyard-neptuneanalytics-stage3/import-graph",
			"roleArn":   "arn:aws:iam::123456789012:role/stackyard-neptuneanalytics",
			"format":    "CSV",
		}),
	)
	if status != http.StatusOK {
		t.Fatalf("expected CreateGraphUsingImportTask 200, got %d: %s", status, body)
	}
	createGraphUsingImportTask := neptuneAnalyticsDecodeJSONMap(t, body)
	_ = neptuneAnalyticsRequireStringField(t, createGraphUsingImportTask, "graphId")
	_ = neptuneAnalyticsRequireStringField(t, createGraphUsingImportTask, "taskId")
}

func TestNeptuneAnalyticsStage4QueryAndSummary(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	status, body := neptuneAnalyticsCall(
		t,
		ts,
		http.MethodPost,
		"/graphs",
		mustJSON(t, map[string]any{
			"graphName":         "stackyard-neptuneanalytics-stage4",
			"provisionedMemory": 128,
		}),
	)
	if status != http.StatusOK {
		t.Fatalf("expected CreateGraph 200, got %d: %s", status, body)
	}
	graphID := neptuneAnalyticsRequireStringField(t, neptuneAnalyticsDecodeJSONMap(t, body), "id")

	status, body = neptuneAnalyticsCallWithHeaders(
		t,
		ts,
		http.MethodPost,
		"/queries",
		mustJSON(t, map[string]any{
			"query":    "MATCH (n) RETURN n LIMIT 1",
			"language": "OPEN_CYPHER",
		}),
		map[string]string{"graphIdentifier": graphID},
	)
	if status != http.StatusOK {
		t.Fatalf("expected ExecuteQuery 200, got %d: %s", status, body)
	}
	executeQuery := neptuneAnalyticsDecodeJSONMap(t, body)
	queryID := neptuneAnalyticsRequireStringField(t, executeQuery, "queryId")

	status, body = neptuneAnalyticsCallWithHeaders(
		t,
		ts,
		http.MethodGet,
		"/queries/"+queryID,
		nil,
		map[string]string{"graphIdentifier": graphID},
	)
	if status != http.StatusOK {
		t.Fatalf("expected GetQuery 200, got %d: %s", status, body)
	}
	if !strings.Contains(body, `"id":"`+queryID+`"`) {
		t.Fatalf("expected query in GetQuery response: %s", body)
	}

	status, body = neptuneAnalyticsCallWithHeaders(
		t,
		ts,
		http.MethodGet,
		"/queries?maxResults=10&state=ALL",
		nil,
		map[string]string{"graphIdentifier": graphID},
	)
	if status != http.StatusOK {
		t.Fatalf("expected ListQueries 200, got %d: %s", status, body)
	}
	if !strings.Contains(body, queryID) {
		t.Fatalf("expected query in ListQueries response: %s", body)
	}

	status, body = neptuneAnalyticsCallWithHeaders(
		t,
		ts,
		http.MethodDelete,
		"/queries/"+queryID,
		nil,
		map[string]string{"graphIdentifier": graphID},
	)
	if status != http.StatusOK {
		t.Fatalf("expected CancelQuery 200, got %d: %s", status, body)
	}

	status, body = neptuneAnalyticsCallWithHeaders(
		t,
		ts,
		http.MethodGet,
		"/summary?mode=DETAILED",
		nil,
		map[string]string{"graphIdentifier": graphID},
	)
	if status != http.StatusOK {
		t.Fatalf("expected GetGraphSummary 200, got %d: %s", status, body)
	}
	if !strings.Contains(body, `"graphSummary"`) || !strings.Contains(body, `"graph"`) {
		t.Fatalf("expected graph summary payload: %s", body)
	}
}

func TestNeptuneAnalyticsStage5PrivateEndpointsAndTags(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	status, body := neptuneAnalyticsCall(
		t,
		ts,
		http.MethodPost,
		"/graphs",
		mustJSON(t, map[string]any{
			"graphName":         "stackyard-neptuneanalytics-stage5",
			"provisionedMemory": 128,
		}),
	)
	if status != http.StatusOK {
		t.Fatalf("expected CreateGraph 200, got %d: %s", status, body)
	}
	createGraph := neptuneAnalyticsDecodeJSONMap(t, body)
	graphID := neptuneAnalyticsRequireStringField(t, createGraph, "id")
	graphARN := neptuneAnalyticsRequireStringField(t, createGraph, "arn")

	status, body = neptuneAnalyticsCall(
		t,
		ts,
		http.MethodPost,
		"/graphs/"+graphID+"/endpoints",
		mustJSON(t, map[string]any{
			"vpcId":               "vpc-0123456789abcdef0",
			"subnetIds":           []string{"subnet-11111111", "subnet-22222222"},
			"vpcSecurityGroupIds": []string{"sg-12345678"},
		}),
	)
	if status != http.StatusOK {
		t.Fatalf("expected CreatePrivateGraphEndpoint 200, got %d: %s", status, body)
	}
	createEndpoint := neptuneAnalyticsDecodeJSONMap(t, body)
	vpcID := neptuneAnalyticsRequireStringField(t, createEndpoint, "vpcId")

	status, body = neptuneAnalyticsCall(t, ts, http.MethodGet, "/graphs/"+graphID+"/endpoints/"+vpcID, nil)
	if status != http.StatusOK {
		t.Fatalf("expected GetPrivateGraphEndpoint 200, got %d: %s", status, body)
	}

	status, body = neptuneAnalyticsCall(t, ts, http.MethodGet, "/graphs/"+graphID+"/endpoints?maxResults=10", nil)
	if status != http.StatusOK {
		t.Fatalf("expected ListPrivateGraphEndpoints 200, got %d: %s", status, body)
	}
	if !strings.Contains(body, vpcID) {
		t.Fatalf("expected endpoint in list response: %s", body)
	}

	escapedARN := url.PathEscape(graphARN)
	status, body = neptuneAnalyticsCall(
		t,
		ts,
		http.MethodPost,
		"/tags/"+escapedARN,
		mustJSON(t, map[string]any{
			"tags": map[string]string{
				"env":   "stage5",
				"owner": "stackyard",
			},
		}),
	)
	if status != http.StatusOK {
		t.Fatalf("expected TagResource 200, got %d: %s", status, body)
	}

	status, body = neptuneAnalyticsCall(t, ts, http.MethodGet, "/tags/"+escapedARN, nil)
	if status != http.StatusOK {
		t.Fatalf("expected ListTagsForResource 200, got %d: %s", status, body)
	}
	if !strings.Contains(body, `"env":"stage5"`) {
		t.Fatalf("expected env tag in list response: %s", body)
	}

	status, body = neptuneAnalyticsCall(t, ts, http.MethodDelete, "/tags/"+escapedARN+"?tagKeys=owner", nil)
	if status != http.StatusOK {
		t.Fatalf("expected UntagResource 200, got %d: %s", status, body)
	}

	status, body = neptuneAnalyticsCall(t, ts, http.MethodGet, "/tags/"+escapedARN, nil)
	if status != http.StatusOK {
		t.Fatalf("expected ListTagsForResource after untag 200, got %d: %s", status, body)
	}
	if strings.Contains(body, `"owner":"stackyard"`) {
		t.Fatalf("expected owner tag to be removed: %s", body)
	}

	status, body = neptuneAnalyticsCall(t, ts, http.MethodDelete, "/graphs/"+graphID+"/endpoints/"+vpcID, nil)
	if status != http.StatusOK {
		t.Fatalf("expected DeletePrivateGraphEndpoint 200, got %d: %s", status, body)
	}
}

func TestNeptuneAnalyticsStage6ValidationAndErrorModel(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	status, body := neptuneAnalyticsCall(
		t,
		ts,
		http.MethodPost,
		"/graphs",
		mustJSON(t, map[string]any{
			"graphName":         "stackyard-neptuneanalytics-stage6",
			"provisionedMemory": 128,
		}),
	)
	if status != http.StatusOK {
		t.Fatalf("expected CreateGraph 200, got %d: %s", status, body)
	}
	createGraph := neptuneAnalyticsDecodeJSONMap(t, body)
	graphID := neptuneAnalyticsRequireStringField(t, createGraph, "id")
	graphARN := neptuneAnalyticsRequireStringField(t, createGraph, "arn")

	status, body = neptuneAnalyticsCallWithHeaders(
		t,
		ts,
		http.MethodGet,
		"/queries",
		nil,
		map[string]string{"graphIdentifier": graphID},
	)
	if status != http.StatusBadRequest || !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ListQueries missing maxResults validation error, got %d: %s", status, body)
	}

	status, body = neptuneAnalyticsCallWithHeaders(
		t,
		ts,
		http.MethodPost,
		"/queries",
		mustJSON(t, map[string]any{
			"query":    "MATCH (n) RETURN n LIMIT 1",
			"language": "SPARQL",
		}),
		map[string]string{"graphIdentifier": graphID},
	)
	if status != http.StatusBadRequest || !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ExecuteQuery validation error for language, got %d: %s", status, body)
	}

	status, body = neptuneAnalyticsCall(t, ts, http.MethodPut, "/graphs/"+graphID, mustJSON(t, map[string]any{}))
	if status != http.StatusBadRequest || !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ResetGraph missing skipSnapshot validation error, got %d: %s", status, body)
	}

	status, body = neptuneAnalyticsCall(
		t,
		ts,
		http.MethodPost,
		"/graphs/"+graphID+"/endpoints",
		mustJSON(t, map[string]any{
			"vpcId":               "vpc-00000000000000001",
			"subnetIds":           []string{"subnet-11111111"},
			"vpcSecurityGroupIds": []string{"sg-11111111"},
		}),
	)
	if status != http.StatusOK {
		t.Fatalf("expected CreatePrivateGraphEndpoint 200, got %d: %s", status, body)
	}

	status, body = neptuneAnalyticsCall(
		t,
		ts,
		http.MethodPost,
		"/graphs/"+graphID+"/endpoints",
		mustJSON(t, map[string]any{
			"vpcId":               "vpc-00000000000000001",
			"subnetIds":           []string{"subnet-11111111"},
			"vpcSecurityGroupIds": []string{"sg-11111111"},
		}),
	)
	if status != http.StatusConflict || !strings.Contains(body, "ConflictException") {
		t.Fatalf("expected duplicate private endpoint conflict, got %d: %s", status, body)
	}

	escapedARN := url.PathEscape(graphARN)
	status, body = neptuneAnalyticsCall(
		t,
		ts,
		http.MethodPost,
		"/tags/"+escapedARN,
		mustJSON(t, map[string]any{
			"tags": map[string]string{},
		}),
	)
	if status != http.StatusBadRequest || !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected TagResource empty-tags validation error, got %d: %s", status, body)
	}

	status, body = neptuneAnalyticsCall(t, ts, http.MethodGet, "/importtasks?nextToken=bad&maxResults=1", nil)
	if status != http.StatusBadRequest || !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ListImportTasks invalid nextToken validation error, got %d: %s", status, body)
	}

	status, body = neptuneAnalyticsCallWithHeaders(
		t,
		ts,
		http.MethodGet,
		"/queries?maxResults=1&state=BAD_STATE",
		nil,
		map[string]string{"graphIdentifier": graphID},
	)
	if status != http.StatusBadRequest || !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ListQueries invalid state validation error, got %d: %s", status, body)
	}
}
