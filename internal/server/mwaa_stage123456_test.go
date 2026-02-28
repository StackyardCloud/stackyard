package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestMWAAStage12EnvironmentLifecycleAndReadSurface(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	environmentName := "stage-mwaa-environment"

	resp := mwaaRequest(t, ts, http.MethodPut, "/environments/"+url.PathEscape(environmentName), []byte(`{
		"AirflowVersion":"2.10.3",
		"EnvironmentClass":"mw1.small",
		"ExecutionRoleArn":"arn:aws:iam::123456789012:role/stackyard-mwaa-role",
		"SourceBucketArn":"arn:aws:s3:::stackyard-mwaa",
		"DagS3Path":"dags"
	}`))
	assertStatus(t, resp, http.StatusOK)
	createPayload := decodeMWAAPayload(t, resp)
	envARN := mwaaStringFromMap(createPayload, "Arn")
	if envARN == "" {
		t.Fatalf("expected CreateEnvironment to return Arn")
	}

	resp = mwaaRequest(t, ts, http.MethodGet, "/environments/"+url.PathEscape(environmentName), nil)
	assertStatus(t, resp, http.StatusOK)
	getPayload := decodeMWAAPayload(t, resp)
	envObj, ok := getPayload["Environment"].(map[string]any)
	if !ok {
		t.Fatalf("expected GetEnvironment to return Environment object")
	}
	if got := mwaaStringFromMap(envObj, "Name"); got != environmentName {
		t.Fatalf("expected environment name %q, got %q", environmentName, got)
	}

	resp = mwaaRequest(t, ts, http.MethodGet, "/environments?MaxResults=10&NextToken=token-000001", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = mwaaRequest(t, ts, http.MethodPatch, "/environments/"+url.PathEscape(environmentName), []byte(`{
		"EnvironmentClass":"mw1.medium",
		"MaxWorkers":8,
		"MinWorkers":2
	}`))
	assertStatus(t, resp, http.StatusOK)
}

func TestMWAAStage34TokenInvokeAndMetricsSurfaces(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	environmentName := "stackyard-environment"

	resp := mwaaRequest(t, ts, http.MethodPost, "/clitoken/"+url.PathEscape(environmentName), []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	cliPayload := decodeMWAAPayload(t, resp)
	if mwaaStringFromMap(cliPayload, "CliToken") == "" {
		t.Fatalf("expected CreateCliToken to return CliToken")
	}

	resp = mwaaRequest(t, ts, http.MethodPost, "/webtoken/"+url.PathEscape(environmentName), []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	webPayload := decodeMWAAPayload(t, resp)
	if mwaaStringFromMap(webPayload, "WebToken") == "" {
		t.Fatalf("expected CreateWebLoginToken to return WebToken")
	}

	resp = mwaaRequest(t, ts, http.MethodPost, "/restapi/"+url.PathEscape(environmentName), []byte(`{"Method":"GET","Path":"/health"}`))
	assertStatus(t, resp, http.StatusOK)
	invokePayload := decodeMWAAPayload(t, resp)
	if statusCode, ok := invokePayload["RestApiStatusCode"].(float64); !ok || int(statusCode) != 200 {
		t.Fatalf("expected InvokeRestApi to return RestApiStatusCode=200")
	}

	resp = mwaaRequest(t, ts, http.MethodPost, "/metrics/environments/"+url.PathEscape(environmentName), []byte(`{
		"MetricData":[{"MetricName":"SchedulerHeartbeat","Unit":"Count","Value":1.0}]
	}`))
	assertStatus(t, resp, http.StatusOK)
}

func TestMWAAStage56TaggingValidationAndIdempotency(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	envName := "stackyard-environment"
	resourceARN := "arn:aws:airflow:us-east-1:123456789012:environment/" + envName
	escapedARN := url.PathEscape(resourceARN)

	resp := mwaaRequest(t, ts, http.MethodPost, "/tags/"+escapedARN, []byte(`{"Tags":{"owner":"qa","env":"stage"}}`))
	assertStatus(t, resp, http.StatusOK)

	resp = mwaaRequest(t, ts, http.MethodGet, "/tags/"+escapedARN, nil)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "owner") {
		t.Fatalf("expected ListTagsForResource to include owner tag, got %q", body)
	}

	resp = mwaaRequest(t, ts, http.MethodDelete, "/tags/"+escapedARN+"?tagKeys=owner", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = mwaaRequest(t, ts, http.MethodGet, "/tags/"+escapedARN, nil)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); strings.Contains(body, `"owner"`) {
		t.Fatalf("expected owner tag to be removed, got %q", body)
	}

	resp = mwaaRequest(t, ts, http.MethodGet, "/unknown-mwaa-route", nil)
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException for unknown route, got %q", body)
	}

	resp = signedRequestWithService(
		t,
		http.MethodPatch,
		ts.URL+"/environments/"+url.PathEscape(envName),
		[]byte(`{"broken":`),
		map[string]string{"Content-Type": "application/json"},
		"airflow",
	)
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "invalid JSON body") {
		t.Fatalf("expected invalid JSON body error, got %q", body)
	}

	resp = mwaaRequest(t, ts, http.MethodDelete, "/environments/"+url.PathEscape(envName), nil)
	assertStatus(t, resp, http.StatusOK)
	resp = mwaaRequest(t, ts, http.MethodDelete, "/environments/"+url.PathEscape(envName), nil)
	assertStatus(t, resp, http.StatusOK)
}

func decodeMWAAPayload(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	payload := map[string]any{}
	if err := json.Unmarshal(mustBody(t, resp), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	return payload
}

func mwaaStringFromMap(payload map[string]any, key string) string {
	raw, ok := payload[key]
	if !ok || raw == nil {
		return ""
	}
	value, ok := raw.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(value)
}
