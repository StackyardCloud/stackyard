package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestSQSStage5AddRemovePermission(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	create := []byte("Action=CreateQueue&QueueName=perm-queue")
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/", create, map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "sqs")
	assertStatus(t, resp, http.StatusOK)

	queueURL := url.QueryEscape(ts.URL + "/123456789012/perm-queue")
	add := []byte("Action=AddPermission&QueueUrl=" + queueURL + "&Label=Stmt1&AWSAccountId.1=111122223333&ActionName.1=SendMessage")
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/", add, map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "sqs")
	assertStatus(t, resp, http.StatusOK)

	getAttrs := []byte("Action=GetQueueAttributes&QueueUrl=" + queueURL + "&AttributeName.1=Policy")
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/", getAttrs, map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "sqs")
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "Stmt1") || !strings.Contains(body, "111122223333") {
		t.Fatalf("expected policy to include statement, got %s", body)
	}

	remove := []byte("Action=RemovePermission&QueueUrl=" + queueURL + "&Label=Stmt1")
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/", remove, map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "sqs")
	assertStatus(t, resp, http.StatusOK)

	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/", getAttrs, map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "sqs")
	assertStatus(t, resp, http.StatusOK)
	body = string(mustBody(t, resp))
	if strings.Contains(body, "Stmt1") {
		t.Fatalf("expected statement removed, got %s", body)
	}
}

func TestSQSStage5InvalidLabel(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	create := []byte("Action=CreateQueue&QueueName=perm-invalid")
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/", create, map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "sqs")
	assertStatus(t, resp, http.StatusOK)

	queueURL := url.QueryEscape(ts.URL + "/123456789012/perm-invalid")
	add := []byte("Action=AddPermission&QueueUrl=" + queueURL + "&Label=bad label&AWSAccountId.1=111122223333&ActionName.1=SendMessage")
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/", add, map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "sqs")
	assertStatus(t, resp, http.StatusBadRequest)
}
