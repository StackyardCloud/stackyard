package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func assertSQSContains(t *testing.T, body []byte, substr string) {
	t.Helper()
	if !strings.Contains(string(body), substr) {
		t.Fatalf("expected response to contain %q, got %s", substr, string(body))
	}
}

func TestSQSStage1FifoQueueIdempotency(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	body := []byte("Action=CreateQueue&QueueName=jobs.fifo&Attribute.1.Name=FifoQueue&Attribute.1.Value=true")
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/", body, map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "sqs")
	assertStatus(t, resp, http.StatusOK)
	assertSQSContains(t, mustBody(t, resp), "<QueueUrl>"+ts.URL+"/123456789012/jobs.fifo</QueueUrl>")

	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/", body, map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "sqs")
	assertStatus(t, resp, http.StatusOK)

	diff := []byte("Action=CreateQueue&QueueName=jobs.fifo&Attribute.1.Name=FifoQueue&Attribute.1.Value=false")
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/", diff, map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "sqs")
	assertStatus(t, resp, http.StatusBadRequest)
	assertSQSContains(t, mustBody(t, resp), "<Code>QueueAlreadyExists</Code>")
}

func TestSQSStage1InvalidNames(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	body := []byte("Action=CreateQueue&QueueName=bad.name")
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/", body, map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "sqs")
	assertStatus(t, resp, http.StatusBadRequest)
	assertSQSContains(t, mustBody(t, resp), "<Code>InvalidParameterValue</Code>")

	body = []byte("Action=CreateQueue&QueueName=jobs.fifo")
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/", body, map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "sqs")
	assertStatus(t, resp, http.StatusBadRequest)
	assertSQSContains(t, mustBody(t, resp), "<Code>InvalidParameterValue</Code>")

	badURL := url.QueryEscape(ts.URL + "/123456789012/")
	body = []byte("Action=DeleteQueue&QueueUrl=" + badURL)
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/", body, map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "sqs")
	assertStatus(t, resp, http.StatusBadRequest)
	assertSQSContains(t, mustBody(t, resp), "<Code>InvalidParameterValue</Code>")
}
