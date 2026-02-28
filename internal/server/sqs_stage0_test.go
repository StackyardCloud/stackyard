package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func readSQSFixture(t *testing.T, name string) []byte {
	t.Helper()
	path := filepath.Join("testdata", "sqs", name)
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return body
}

func replaceSQSBaseURL(body []byte, baseURL string) []byte {
	return []byte(strings.ReplaceAll(string(body), "{{BASE_URL}}", baseURL))
}

func assertSQSXML(t *testing.T, got, want []byte) {
	t.Helper()
	trimGot := strings.TrimSpace(string(got))
	trimWant := strings.TrimSpace(string(want))
	if trimGot != trimWant {
		t.Fatalf("unexpected XML response:\n got=%s\nwant=%s", trimGot, trimWant)
	}
}

func TestSQSStage0QueueFixtures(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	createBody := []byte("Action=CreateQueue&QueueName=demo&Version=2012-11-05")
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/", createBody, map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "sqs")
	assertStatus(t, resp, http.StatusOK)
	createResp := replaceSQSBaseURL(readSQSFixture(t, "create-queue-response.xml"), ts.URL)
	assertSQSXML(t, mustBody(t, resp), createResp)

	getBody := []byte("Action=GetQueueUrl&QueueName=demo&Version=2012-11-05")
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/", getBody, map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "sqs")
	assertStatus(t, resp, http.StatusOK)
	getResp := replaceSQSBaseURL(readSQSFixture(t, "get-queue-url-response.xml"), ts.URL)
	assertSQSXML(t, mustBody(t, resp), getResp)

	listBody := []byte("Action=ListQueues&Version=2012-11-05")
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/", listBody, map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "sqs")
	assertStatus(t, resp, http.StatusOK)
	listResp := replaceSQSBaseURL(readSQSFixture(t, "list-queues-response.xml"), ts.URL)
	assertSQSXML(t, mustBody(t, resp), listResp)

	deleteURL := url.QueryEscape(ts.URL + "/123456789012/demo")
	deleteBody := []byte("Action=DeleteQueue&QueueUrl=" + deleteURL + "&Version=2012-11-05")
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/", deleteBody, map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "sqs")
	assertStatus(t, resp, http.StatusOK)
	deleteResp := readSQSFixture(t, "delete-queue-response.xml")
	assertSQSXML(t, mustBody(t, resp), deleteResp)
}

func TestSQSStage0Errors(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	missingName := []byte("Action=CreateQueue&Version=2012-11-05")
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/", missingName, map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "sqs")
	assertStatus(t, resp, http.StatusBadRequest)
	missingResp := readSQSFixture(t, "error-missing-queue-name.xml")
	assertSQSXML(t, mustBody(t, resp), missingResp)

	invalidAction := []byte("Action=Nope&Version=2012-11-05")
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/", invalidAction, map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "sqs")
	assertStatus(t, resp, http.StatusBadRequest)
	invalidResp := readSQSFixture(t, "error-invalid-action.xml")
	assertSQSXML(t, mustBody(t, resp), invalidResp)
}
