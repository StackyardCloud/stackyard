package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestEntityResolutionStage12NamespaceSchemaAndWorkflowLifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	namespaceName := "stage-id-namespace"
	schemaName := "stage-schema"
	matchingWorkflow := "stage-matching-workflow"
	idMappingWorkflow := "stage-idmapping-workflow"

	resp := entityResolutionRequest(
		t,
		ts,
		http.MethodPost,
		"/idnamespaces",
		[]byte(`{"idNamespaceName":"`+namespaceName+`"}`),
	)
	assertStatus(t, resp, http.StatusOK)
	resp = entityResolutionRequest(t, ts, http.MethodGet, "/idnamespaces/"+url.PathEscape(namespaceName), nil)
	assertStatus(t, resp, http.StatusOK)
	resp = entityResolutionRequest(t, ts, http.MethodGet, "/idnamespaces", nil)
	assertStatus(t, resp, http.StatusOK)
	resp = entityResolutionRequest(t, ts, http.MethodPut, "/idnamespaces/"+url.PathEscape(namespaceName), []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)

	resp = entityResolutionRequest(
		t,
		ts,
		http.MethodPost,
		"/schemas",
		[]byte(`{"schemaName":"`+schemaName+`"}`),
	)
	assertStatus(t, resp, http.StatusOK)
	resp = entityResolutionRequest(t, ts, http.MethodGet, "/schemas/"+url.PathEscape(schemaName), nil)
	assertStatus(t, resp, http.StatusOK)
	resp = entityResolutionRequest(t, ts, http.MethodGet, "/schemas", nil)
	assertStatus(t, resp, http.StatusOK)
	resp = entityResolutionRequest(t, ts, http.MethodPut, "/schemas/"+url.PathEscape(schemaName), []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)

	resp = entityResolutionRequest(
		t,
		ts,
		http.MethodPost,
		"/matchingworkflows",
		[]byte(`{"workflowName":"`+matchingWorkflow+`"}`),
	)
	assertStatus(t, resp, http.StatusOK)
	resp = entityResolutionRequest(t, ts, http.MethodGet, "/matchingworkflows/"+url.PathEscape(matchingWorkflow), nil)
	assertStatus(t, resp, http.StatusOK)
	resp = entityResolutionRequest(t, ts, http.MethodGet, "/matchingworkflows", nil)
	assertStatus(t, resp, http.StatusOK)
	resp = entityResolutionRequest(t, ts, http.MethodPut, "/matchingworkflows/"+url.PathEscape(matchingWorkflow), []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)

	resp = entityResolutionRequest(
		t,
		ts,
		http.MethodPost,
		"/idmappingworkflows",
		[]byte(`{"workflowName":"`+idMappingWorkflow+`"}`),
	)
	assertStatus(t, resp, http.StatusOK)
	resp = entityResolutionRequest(t, ts, http.MethodGet, "/idmappingworkflows/"+url.PathEscape(idMappingWorkflow), nil)
	assertStatus(t, resp, http.StatusOK)
	resp = entityResolutionRequest(t, ts, http.MethodGet, "/idmappingworkflows", nil)
	assertStatus(t, resp, http.StatusOK)
	resp = entityResolutionRequest(t, ts, http.MethodPut, "/idmappingworkflows/"+url.PathEscape(idMappingWorkflow), []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)

	resp = entityResolutionRequest(t, ts, http.MethodDelete, "/schemas/"+url.PathEscape(schemaName), nil)
	assertStatus(t, resp, http.StatusOK)
	resp = entityResolutionRequest(t, ts, http.MethodDelete, "/idmappingworkflows/"+url.PathEscape(idMappingWorkflow), nil)
	assertStatus(t, resp, http.StatusOK)
	resp = entityResolutionRequest(t, ts, http.MethodDelete, "/matchingworkflows/"+url.PathEscape(matchingWorkflow), nil)
	assertStatus(t, resp, http.StatusOK)
	resp = entityResolutionRequest(t, ts, http.MethodDelete, "/idnamespaces/"+url.PathEscape(namespaceName), nil)
	assertStatus(t, resp, http.StatusOK)
}

func TestEntityResolutionStage34JobsPolicyAndTagging(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	matchingWorkflow := "stackyard-matching-workflow"
	idMappingWorkflow := "stackyard-idmapping-workflow"
	arn := "arn:aws:entityresolution:us-east-1:123456789012:idnamespace/stackyard-id-namespace"

	resp := entityResolutionRequest(t, ts, http.MethodPost, "/matchingworkflows/"+url.PathEscape(matchingWorkflow)+"/jobs", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	resp = entityResolutionRequest(t, ts, http.MethodGet, "/matchingworkflows/"+url.PathEscape(matchingWorkflow)+"/jobs/job-000001", nil)
	assertStatus(t, resp, http.StatusOK)
	resp = entityResolutionRequest(t, ts, http.MethodGet, "/matchingworkflows/"+url.PathEscape(matchingWorkflow)+"/jobs", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = entityResolutionRequest(t, ts, http.MethodPost, "/idmappingworkflows/"+url.PathEscape(idMappingWorkflow)+"/jobs", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	resp = entityResolutionRequest(t, ts, http.MethodGet, "/idmappingworkflows/"+url.PathEscape(idMappingWorkflow)+"/jobs/job-000001", nil)
	assertStatus(t, resp, http.StatusOK)
	resp = entityResolutionRequest(t, ts, http.MethodGet, "/idmappingworkflows/"+url.PathEscape(idMappingWorkflow)+"/jobs", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = entityResolutionRequest(t, ts, http.MethodPost, "/matchingworkflows/"+url.PathEscape(matchingWorkflow)+"/generateMatches", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	resp = entityResolutionRequest(t, ts, http.MethodPost, "/matchingworkflows/"+url.PathEscape(matchingWorkflow)+"/matches", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	resp = entityResolutionRequest(t, ts, http.MethodDelete, "/matchingworkflows/"+url.PathEscape(matchingWorkflow)+"/uniqueids", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = entityResolutionRequest(t, ts, http.MethodPut, "/policies/"+url.PathEscape(arn), []byte(`{"policy":"{\"Version\":\"2012-10-17\",\"Statement\":[]}"}`))
	assertStatus(t, resp, http.StatusOK)
	resp = entityResolutionRequest(t, ts, http.MethodGet, "/policies/"+url.PathEscape(arn), nil)
	assertStatus(t, resp, http.StatusOK)
	resp = entityResolutionRequest(t, ts, http.MethodPost, "/policies/"+url.PathEscape(arn)+"/stage-statement", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	resp = entityResolutionRequest(t, ts, http.MethodDelete, "/policies/"+url.PathEscape(arn)+"/stage-statement", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = entityResolutionRequest(t, ts, http.MethodPost, "/tags/"+url.PathEscape(arn), []byte(`{"tags":{"env":"stage","owner":"qa"}}`))
	assertStatus(t, resp, http.StatusOK)
	resp = entityResolutionRequest(t, ts, http.MethodGet, "/tags/"+url.PathEscape(arn), nil)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "owner") {
		t.Fatalf("expected ListTagsForResource to include owner tag, got %q", body)
	}
	resp = entityResolutionRequest(t, ts, http.MethodDelete, "/tags/"+url.PathEscape(arn)+"?tagKeys=owner", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = entityResolutionRequest(t, ts, http.MethodGet, "/providerservices/default-provider/default-service", nil)
	assertStatus(t, resp, http.StatusOK)
	resp = entityResolutionRequest(t, ts, http.MethodGet, "/providerservices", nil)
	assertStatus(t, resp, http.StatusOK)
}

func TestEntityResolutionStage56ValidationAndIdempotency(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := entityResolutionRequest(t, ts, http.MethodGet, "/entityresolution/unknown", nil)
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException for unknown route, got %q", body)
	}

	resp = signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/idnamespaces",
		[]byte(`{"broken":`),
		map[string]string{"Content-Type": "application/json"},
		"entityresolution",
	)
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "invalid JSON body") {
		t.Fatalf("expected invalid JSON body error, got %q", body)
	}

	namespaceName := "stage-idempotent-namespace"
	resp = entityResolutionRequest(t, ts, http.MethodPost, "/idnamespaces", []byte(`{"idNamespaceName":"`+namespaceName+`"}`))
	assertStatus(t, resp, http.StatusOK)
	resp = entityResolutionRequest(t, ts, http.MethodPost, "/idnamespaces", []byte(`{"idNamespaceName":"`+namespaceName+`"}`))
	assertStatus(t, resp, http.StatusOK)
	resp = entityResolutionRequest(t, ts, http.MethodGet, "/idnamespaces/"+url.PathEscape(namespaceName), nil)
	assertStatus(t, resp, http.StatusOK)
}
