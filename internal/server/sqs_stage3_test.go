package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestSQSStage3AttributesAndTags(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	create := []byte("Action=CreateQueue&QueueName=attrs&Attribute.1.Name=VisibilityTimeout&Attribute.1.Value=45")
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/", create, map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "sqs")
	assertStatus(t, resp, http.StatusOK)

	queueURL := url.QueryEscape(ts.URL + "/123456789012/attrs")
	getAttrs := []byte("Action=GetQueueAttributes&QueueUrl=" + queueURL + "&AttributeName.1=All")
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/", getAttrs, map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "sqs")
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "<Name>VisibilityTimeout</Name>") || !strings.Contains(body, "<Value>45</Value>") {
		t.Fatalf("expected VisibilityTimeout attribute, got %s", body)
	}

	setAttrs := []byte("Action=SetQueueAttributes&QueueUrl=" + queueURL + "&Attribute.1.Name=VisibilityTimeout&Attribute.1.Value=60")
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/", setAttrs, map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "sqs")
	assertStatus(t, resp, http.StatusOK)

	getAttrs = []byte("Action=GetQueueAttributes&QueueUrl=" + queueURL + "&AttributeName.1=VisibilityTimeout")
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/", getAttrs, map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "sqs")
	assertStatus(t, resp, http.StatusOK)
	body = string(mustBody(t, resp))
	if !strings.Contains(body, "<Value>60</Value>") {
		t.Fatalf("expected updated VisibilityTimeout, got %s", body)
	}

	tag := "Action=TagQueue&QueueUrl=" + queueURL + "&Tag.1.Key=env&Tag.1.Value=dev&Tag.2.Key=team&Tag.2.Value=core"
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte(tag), map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "sqs")
	assertStatus(t, resp, http.StatusOK)

	listTags := []byte("Action=ListQueueTags&QueueUrl=" + queueURL)
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/", listTags, map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "sqs")
	assertStatus(t, resp, http.StatusOK)
	body = string(mustBody(t, resp))
	if !strings.Contains(body, "<Key>env</Key>") || !strings.Contains(body, "<Key>team</Key>") {
		t.Fatalf("expected tags, got %s", body)
	}

	untag := "Action=UntagQueue&QueueUrl=" + queueURL + "&TagKey.1=env"
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte(untag), map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "sqs")
	assertStatus(t, resp, http.StatusOK)

	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/", listTags, map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "sqs")
	assertStatus(t, resp, http.StatusOK)
	body = string(mustBody(t, resp))
	if strings.Contains(body, "<Key>env</Key>") {
		t.Fatalf("expected env tag removed, got %s", body)
	}
	if !strings.Contains(body, "<Key>team</Key>") {
		t.Fatalf("expected team tag to remain, got %s", body)
	}
}
