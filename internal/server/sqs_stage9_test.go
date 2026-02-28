package server

import (
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestSQSStage9PurgeQueueQuery(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	create := []byte("Action=CreateQueue&QueueName=purge-q")
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/", create, map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "sqs")
	assertStatus(t, resp, http.StatusOK)

	queueURL := url.QueryEscape(ts.URL + "/123456789012/purge-q")
	send := []byte("Action=SendMessage&QueueUrl=" + queueURL + "&MessageBody=one")
	_ = signedRequestWithService(t, http.MethodPost, ts.URL+"/", send, map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "sqs")
	send = []byte("Action=SendMessage&QueueUrl=" + queueURL + "&MessageBody=two")
	_ = signedRequestWithService(t, http.MethodPost, ts.URL+"/", send, map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "sqs")

	purge := []byte("Action=PurgeQueue&QueueUrl=" + queueURL)
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/", purge, map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "sqs")
	assertStatus(t, resp, http.StatusOK)

	receive := []byte("Action=ReceiveMessage&QueueUrl=" + queueURL + "&MaxNumberOfMessages=10")
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/", receive, map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "sqs")
	assertStatus(t, resp, http.StatusOK)
	var env receiveMessageEnvelope
	if err := xml.Unmarshal(mustBody(t, resp), &env); err != nil {
		t.Fatalf("parse receive: %v", err)
	}
	if len(env.Messages) != 0 {
		t.Fatalf("expected 0 messages after purge, got %d", len(env.Messages))
	}
}

func TestSQSStage9PurgeQueueJSON(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	create := []byte("Action=CreateQueue&QueueName=purge-json")
	_ = signedRequestWithService(t, http.MethodPost, ts.URL+"/", create, map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "sqs")

	queueURL := ts.URL + "/123456789012/purge-json"
	send := []byte("Action=SendMessage&QueueUrl=" + url.QueryEscape(queueURL) + "&MessageBody=payload")
	_ = signedRequestWithService(t, http.MethodPost, ts.URL+"/", send, map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "sqs")

	headers := map[string]string{
		"Content-Type": "application/x-amz-json-1.0",
		"X-Amz-Target": "AmazonSQS.PurgeQueue",
	}
	body := []byte(`{"QueueUrl":"` + queueURL + `"}`)
	resp := signRequestWithHeaders(t, http.MethodPost, ts.URL+"/", body, headers, "sqs", testRegion, "")
	assertStatus(t, resp, http.StatusOK)

	receive := []byte("Action=ReceiveMessage&QueueUrl=" + url.QueryEscape(queueURL) + "&MaxNumberOfMessages=10")
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/", receive, map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "sqs")
	assertStatus(t, resp, http.StatusOK)
	var env receiveMessageEnvelope
	if err := xml.Unmarshal(mustBody(t, resp), &env); err != nil {
		t.Fatalf("parse receive: %v", err)
	}
	if len(env.Messages) != 0 {
		t.Fatalf("expected 0 messages after purge, got %d", len(env.Messages))
	}
}
