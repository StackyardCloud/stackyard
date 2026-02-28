package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSQSStage6MessageMoveTasks(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	create := []byte("Action=CreateQueue&QueueName=source")
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/", create, map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "sqs")
	assertStatus(t, resp, http.StatusOK)

	sourceArn := "arn:aws:sqs:us-east-1:123456789012:source"
	start := []byte("Action=StartMessageMoveTask&SourceArn=" + sourceArn)
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/", start, map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "sqs")
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "<TaskHandle>") {
		t.Fatalf("expected task handle, got %s", body)
	}
	handle := between(body, "<TaskHandle>", "</TaskHandle>")
	if handle == "" {
		t.Fatalf("expected task handle value, got %s", body)
	}

	list := []byte("Action=ListMessageMoveTasks&SourceArn=" + sourceArn)
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/", list, map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "sqs")
	assertStatus(t, resp, http.StatusOK)
	body = string(mustBody(t, resp))
	if !strings.Contains(body, "<TaskHandle>") {
		t.Fatalf("expected list to include task handle, got %s", body)
	}

	cancel := []byte("Action=CancelMessageMoveTask&TaskHandle=" + handle)
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/", cancel, map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "sqs")
	assertStatus(t, resp, http.StatusOK)
}

func TestSQSStage6InvalidSourceQueue(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	start := []byte("Action=StartMessageMoveTask&SourceArn=arn:aws:sqs:us-east-1:123456789012:missing")
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/", start, map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "sqs")
	assertStatus(t, resp, http.StatusNotFound)
}

func between(value, start, end string) string {
	s := strings.Index(value, start)
	if s == -1 {
		return ""
	}
	s += len(start)
	e := strings.Index(value[s:], end)
	if e == -1 {
		return ""
	}
	return value[s : s+e]
}
