package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
)

var identityStoreResourceIDPattern = regexp.MustCompile(`^[0-9a-f]{10}-[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

func TestDirectConnectShard11RootShapes(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := directConnectRequest(t, ts, "CreateConnection", `{}`)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, `"connectionId"`) || strings.Contains(body, `"connection":`) {
		t.Fatalf("expected CreateConnection to return a root connection shape, got %s", body)
	}

	resp = directConnectRequest(t, ts, "CreatePrivateVirtualInterface", `{}`)
	assertStatus(t, resp, http.StatusOK)
	body = string(mustBody(t, resp))
	if !strings.Contains(body, `"virtualInterfaceId"`) || strings.Contains(body, `"virtualInterface":`) {
		t.Fatalf("expected CreatePrivateVirtualInterface to return a root virtual interface shape, got %s", body)
	}

	resp = directConnectRequest(t, ts, "CreateTransitVirtualInterface", `{}`)
	assertStatus(t, resp, http.StatusOK)
	body = string(mustBody(t, resp))
	var createTransit map[string]any
	if err := json.Unmarshal([]byte(body), &createTransit); err != nil {
		t.Fatalf("failed to decode CreateTransitVirtualInterface body: %v", err)
	}
	createTransitVIF, _ := createTransit["virtualInterface"].(map[string]any)
	if len(createTransitVIF) == 0 {
		t.Fatalf("expected CreateTransitVirtualInterface to wrap the virtual interface shape, got %s", body)
	}
	if _, ok := createTransit["virtualInterfaceId"]; ok {
		t.Fatalf("expected CreateTransitVirtualInterface to omit root virtualInterfaceId, got %s", body)
	}
	if got, _ := createTransitVIF["virtualInterfaceType"].(string); got != "transit" {
		t.Fatalf("expected CreateTransitVirtualInterface to return transit type, got %s", body)
	}

	resp = directConnectRequest(t, ts, "AllocateTransitVirtualInterface", `{}`)
	assertStatus(t, resp, http.StatusOK)
	body = string(mustBody(t, resp))
	var allocateTransit map[string]any
	if err := json.Unmarshal([]byte(body), &allocateTransit); err != nil {
		t.Fatalf("failed to decode AllocateTransitVirtualInterface body: %v", err)
	}
	allocateTransitVIF, _ := allocateTransit["virtualInterface"].(map[string]any)
	if len(allocateTransitVIF) == 0 {
		t.Fatalf("expected AllocateTransitVirtualInterface to wrap the virtual interface shape, got %s", body)
	}
	if _, ok := allocateTransit["virtualInterfaceId"]; ok {
		t.Fatalf("expected AllocateTransitVirtualInterface to omit root virtualInterfaceId, got %s", body)
	}
	if got, _ := allocateTransitVIF["virtualInterfaceType"].(string); got != "transit" {
		t.Fatalf("expected AllocateTransitVirtualInterface to return transit type, got %s", body)
	}

	resp = directConnectRequest(t, ts, "DescribeLoa", `{}`)
	assertStatus(t, resp, http.StatusOK)
	body = string(mustBody(t, resp))
	if strings.Contains(body, `"loa":`) || !strings.Contains(body, `"loaContent"`) {
		t.Fatalf("expected DescribeLoa to return the flat LOA shape, got %s", body)
	}

	resp = directConnectRequest(t, ts, "ConfirmCustomerAgreement", `{}`)
	assertStatus(t, resp, http.StatusOK)
	body = string(mustBody(t, resp))
	if !strings.Contains(body, `"status":"signed"`) {
		t.Fatalf("expected ConfirmCustomerAgreement to return signed status, got %s", body)
	}
}

func TestIdentityStoreShard11ResourceIDsAndModeledEmptyMutations(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := identityStoreRequest(t, ts, "CreateUser", `{
		"IdentityStoreId":"d-1234567890",
		"UserName":"shard11.user",
		"DisplayName":"Shard 11 User"
	}`)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	userID := jsonTagValue(body, "UserId")
	if !identityStoreResourceIDPattern.MatchString(userID) {
		t.Fatalf("expected CreateUser to return a modeled ResourceId, got %q in %s", userID, body)
	}

	resp = identityStoreRequest(t, ts, "UpdateUser", `{
		"IdentityStoreId":"d-1234567890",
		"UserId":"`+userID+`",
		"Operations":[{"AttributePath":"DisplayName","AttributeValue":"Updated User"}]
	}`)
	assertStatus(t, resp, http.StatusOK)
	body = strings.TrimSpace(string(mustBody(t, resp)))
	if body != "{}" {
		t.Fatalf("expected UpdateUser modeled-empty body, got %s", body)
	}

	resp = identityStoreRequest(t, ts, "DeleteUser", `{
		"IdentityStoreId":"d-1234567890",
		"UserId":"`+userID+`"
	}`)
	assertStatus(t, resp, http.StatusOK)
	body = strings.TrimSpace(string(mustBody(t, resp)))
	if body != "{}" {
		t.Fatalf("expected DeleteUser modeled-empty body, got %s", body)
	}
}

func TestNeptuneDataShard11ModeledShapes(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	status, body := neptuneDataCall(t, ts, http.MethodPost, "/loader", []byte(`{
		"source":"s3://stackyard-neptunedata/shard11/input.csv",
		"format":"csv",
		"s3BucketRegion":"us-east-1",
		"iamRoleArn":"arn:aws:iam::123456789012:role/stackyard-neptunedata"
	}`))
	if status != http.StatusOK {
		t.Fatalf("expected StartLoaderJob 200, got %d: %s", status, body)
	}
	var startLoader map[string]any
	if err := json.Unmarshal([]byte(body), &startLoader); err != nil {
		t.Fatalf("failed to decode StartLoaderJob body: %v", err)
	}
	startPayload, _ := startLoader["payload"].(map[string]any)
	loadID, _ := startPayload["loadId"].(string)
	if strings.TrimSpace(loadID) == "" {
		t.Fatalf("expected StartLoaderJob payload.loadId, got %s", body)
	}
	if _, ok := startPayload["runNumber"].(string); !ok {
		t.Fatalf("expected StartLoaderJob payload.runNumber to be a string, got %T in %s", startPayload["runNumber"], body)
	}

	status, body = neptuneDataCall(t, ts, http.MethodGet, "/loader/"+loadID+"?details=true&errors=true&page=1&errorsPerPage=10", nil)
	if status != http.StatusOK {
		t.Fatalf("expected GetLoaderJobStatus 200, got %d: %s", status, body)
	}
	var loaderStatus map[string]any
	if err := json.Unmarshal([]byte(body), &loaderStatus); err != nil {
		t.Fatalf("failed to decode GetLoaderJobStatus body: %v", err)
	}
	if got, _ := loaderStatus["loadId"].(string); got != loadID {
		t.Fatalf("expected GetLoaderJobStatus top-level loadId %q, got %q in %s", loadID, got, body)
	}
	if payload, ok := loaderStatus["payload"].(map[string]any); !ok || len(payload) != 0 {
		t.Fatalf("expected GetLoaderJobStatus payload to be an empty document, got %s", body)
	}

	status, body = neptuneDataCall(t, ts, http.MethodPost, "/gremlin", []byte(`{"gremlin":"g.V().limit(1)"}`))
	if status != http.StatusOK {
		t.Fatalf("expected ExecuteGremlinQuery 200, got %d: %s", status, body)
	}
	var gremlinExec map[string]any
	if err := json.Unmarshal([]byte(body), &gremlinExec); err != nil {
		t.Fatalf("failed to decode ExecuteGremlinQuery body: %v", err)
	}
	if _, ok := gremlinExec["result"].(map[string]any); !ok {
		t.Fatalf("expected ExecuteGremlinQuery result to be an object, got %s", body)
	}
	if strings.Contains(body, `"queryStatus"`) {
		t.Fatalf("expected ExecuteGremlinQuery to omit legacy meta.queryStatus, got %s", body)
	}
	gremlinQueryID, _ := gremlinExec["queryId"].(string)

	status, body = neptuneDataCall(t, ts, http.MethodGet, "/gremlin/status/"+gremlinQueryID, nil)
	if status != http.StatusOK {
		t.Fatalf("expected GetGremlinQueryStatus 200, got %d: %s", status, body)
	}
	if strings.Contains(body, `"subqueries"`) {
		t.Fatalf("expected GetGremlinQueryStatus to omit document-typed subqueries members, got %s", body)
	}

	status, body = neptuneDataCall(
		t,
		ts,
		http.MethodGet,
		"/propertygraph/stream?limit=2&iteratorType=TRIM_HORIZON&commitNum=1&opNum=1",
		nil,
	)
	if status != http.StatusOK {
		t.Fatalf("expected GetPropertygraphStream 200, got %d: %s", status, body)
	}
	if !strings.Contains(body, `"lastTrxTimestamp"`) || !strings.Contains(body, `"commitTimestamp"`) {
		t.Fatalf("expected GetPropertygraphStream to use modeled wire names, got %s", body)
	}
	if !strings.Contains(body, `"value":"node"`) {
		t.Fatalf("expected GetPropertygraphStream to return document-valued propertygraph data, got %s", body)
	}

	status, body = neptuneDataCall(
		t,
		ts,
		http.MethodGet,
		"/sparql/stream?limit=2&iteratorType=LATEST&commitNum=1&opNum=1",
		nil,
	)
	if status != http.StatusOK {
		t.Fatalf("expected GetSparqlStream 200, got %d: %s", status, body)
	}
	if !strings.Contains(body, `"format":"NQUADS"`) || !strings.Contains(body, `"lastTrxTimestamp"`) {
		t.Fatalf("expected GetSparqlStream to use modeled wire names and format, got %s", body)
	}

	status, body = neptuneDataCall(t, ts, http.MethodPost, "/opencypher", []byte(`{"query":"MATCH (n) RETURN n LIMIT 1"}`))
	if status != http.StatusOK {
		t.Fatalf("expected ExecuteOpenCypherQuery 200, got %d: %s", status, body)
	}
	var openCypherExec map[string]any
	if err := json.Unmarshal([]byte(body), &openCypherExec); err != nil {
		t.Fatalf("failed to decode ExecuteOpenCypherQuery body: %v", err)
	}
	if _, ok := openCypherExec["results"].(map[string]any); !ok {
		t.Fatalf("expected ExecuteOpenCypherQuery results to be an object, got %s", body)
	}
}
