package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestSQSStage4RedriveAndListSources(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	dlqCreate := []byte("Action=CreateQueue&QueueName=dlq")
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/", dlqCreate, map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "sqs")
	assertStatus(t, resp, http.StatusOK)

	policy := `{"deadLetterTargetArn":"arn:aws:sqs:us-east-1:123456789012:dlq","maxReceiveCount":"1"}`
	policy = url.QueryEscape(policy)
	create := []byte("Action=CreateQueue&QueueName=source&Attribute.1.Name=RedrivePolicy&Attribute.1.Value=" + policy)
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/", create, map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "sqs")
	assertStatus(t, resp, http.StatusOK)

	sourceURL := url.QueryEscape(ts.URL + "/123456789012/source")
	dlqURL := url.QueryEscape(ts.URL + "/123456789012/dlq")

	send := []byte("Action=SendMessage&QueueUrl=" + sourceURL + "&MessageBody=hello")
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/", send, map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "sqs")
	assertStatus(t, resp, http.StatusOK)

	recv := []byte("Action=ReceiveMessage&QueueUrl=" + sourceURL + "&MaxNumberOfMessages=1&VisibilityTimeout=0")
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/", recv, map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "sqs")
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "<Message>") {
		t.Fatalf("expected message on first receive, got %s", body)
	}

	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/", recv, map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "sqs")
	assertStatus(t, resp, http.StatusOK)
	body = string(mustBody(t, resp))
	if strings.Contains(body, "<Message>") {
		t.Fatalf("expected no message after redrive, got %s", body)
	}

	recvDLQ := []byte("Action=ReceiveMessage&QueueUrl=" + dlqURL + "&MaxNumberOfMessages=1")
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/", recvDLQ, map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "sqs")
	assertStatus(t, resp, http.StatusOK)
	body = string(mustBody(t, resp))
	if !strings.Contains(body, "<Message>") {
		t.Fatalf("expected message on dlq, got %s", body)
	}

	list := []byte("Action=ListDeadLetterSourceQueues&QueueUrl=" + dlqURL)
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/", list, map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "sqs")
	assertStatus(t, resp, http.StatusOK)
	body = string(mustBody(t, resp))
	if !strings.Contains(body, ts.URL+"/123456789012/source") {
		t.Fatalf("expected source queue url, got %s", body)
	}
}
