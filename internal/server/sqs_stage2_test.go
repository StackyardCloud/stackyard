package server

import (
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
)

type receiveMessageEnvelope struct {
	Messages []sqsReceivedMessage `xml:"ReceiveMessageResult>Message"`
}

func TestSQSStage2SendReceiveDelete(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	create := []byte("Action=CreateQueue&QueueName=jobs")
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/", create, map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "sqs")
	assertStatus(t, resp, http.StatusOK)

	queueURL := url.QueryEscape(ts.URL + "/123456789012/jobs")
	send := []byte("Action=SendMessage&QueueUrl=" + queueURL + "&MessageBody=hello")
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/", send, map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "sqs")
	assertStatus(t, resp, http.StatusOK)

	receive := []byte("Action=ReceiveMessage&QueueUrl=" + queueURL + "&MaxNumberOfMessages=1&VisibilityTimeout=5")
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/", receive, map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "sqs")
	assertStatus(t, resp, http.StatusOK)
	var env receiveMessageEnvelope
	if err := xml.Unmarshal(mustBody(t, resp), &env); err != nil {
		t.Fatalf("parse receive: %v", err)
	}
	if len(env.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(env.Messages))
	}

	del := []byte("Action=DeleteMessage&QueueUrl=" + queueURL + "&ReceiptHandle=" + url.QueryEscape(env.Messages[0].ReceiptHandle))
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/", del, map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "sqs")
	assertStatus(t, resp, http.StatusOK)
}

func TestSQSStage2ChangeVisibility(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	create := []byte("Action=CreateQueue&QueueName=vis")
	_ = signedRequestWithService(t, http.MethodPost, ts.URL+"/", create, map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "sqs")

	queueURL := url.QueryEscape(ts.URL + "/123456789012/vis")
	send := []byte("Action=SendMessage&QueueUrl=" + queueURL + "&MessageBody=hello")
	_ = signedRequestWithService(t, http.MethodPost, ts.URL+"/", send, map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "sqs")

	receive := []byte("Action=ReceiveMessage&QueueUrl=" + queueURL + "&MaxNumberOfMessages=1&VisibilityTimeout=10")
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/", receive, map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "sqs")
	var env receiveMessageEnvelope
	if err := xml.Unmarshal(mustBody(t, resp), &env); err != nil {
		t.Fatalf("parse receive: %v", err)
	}
	if len(env.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(env.Messages))
	}

	change := []byte("Action=ChangeMessageVisibility&QueueUrl=" + queueURL + "&ReceiptHandle=" + url.QueryEscape(env.Messages[0].ReceiptHandle) + "&VisibilityTimeout=0")
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/", change, map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "sqs")
	assertStatus(t, resp, http.StatusOK)

	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/", receive, map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "sqs")
	env = receiveMessageEnvelope{}
	if err := xml.Unmarshal(mustBody(t, resp), &env); err != nil {
		t.Fatalf("parse receive: %v", err)
	}
	if len(env.Messages) != 1 {
		t.Fatalf("expected 1 message after visibility reset, got %d", len(env.Messages))
	}
}

func TestSQSStage2BatchFlows(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	create := []byte("Action=CreateQueue&QueueName=batch")
	_ = signedRequestWithService(t, http.MethodPost, ts.URL+"/", create, map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "sqs")

	queueURL := url.QueryEscape(ts.URL + "/123456789012/batch")
	send := "Action=SendMessageBatch&QueueUrl=" + queueURL +
		"&SendMessageBatchRequestEntry.1.Id=a&SendMessageBatchRequestEntry.1.MessageBody=one" +
		"&SendMessageBatchRequestEntry.2.Id=b&SendMessageBatchRequestEntry.2.MessageBody=two"
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte(send), map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "sqs")
	assertStatus(t, resp, http.StatusOK)

	receive := []byte("Action=ReceiveMessage&QueueUrl=" + queueURL + "&MaxNumberOfMessages=2")
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/", receive, map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "sqs")
	var env receiveMessageEnvelope
	if err := xml.Unmarshal(mustBody(t, resp), &env); err != nil {
		t.Fatalf("parse receive: %v", err)
	}
	if len(env.Messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(env.Messages))
	}

	deleteReq := strings.Builder{}
	deleteReq.WriteString("Action=DeleteMessageBatch&QueueUrl=" + queueURL)
	for i, msg := range env.Messages {
		idx := i + 1
		deleteReq.WriteString("&DeleteMessageBatchRequestEntry." + strconv.Itoa(idx) + ".Id=" + url.QueryEscape(msg.MessageId))
		deleteReq.WriteString("&DeleteMessageBatchRequestEntry." + strconv.Itoa(idx) + ".ReceiptHandle=" + url.QueryEscape(msg.ReceiptHandle))
	}
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte(deleteReq.String()), map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "sqs")
	assertStatus(t, resp, http.StatusOK)
}
