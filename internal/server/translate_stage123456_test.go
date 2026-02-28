package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTranslateStage12TranslationAndLanguageReadSurface(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := translateRequest(t, ts, "ListLanguages", `{}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "English") {
		t.Fatalf("expected ListLanguages to include English, got %q", body)
	}

	resp = translateRequest(t, ts, "TranslateText", `{"Text":"hello world","SourceLanguageCode":"en","TargetLanguageCode":"es"}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "TranslatedText") {
		t.Fatalf("expected TranslateText to include TranslatedText, got %q", body)
	}

	resp = translateRequest(t, ts, "TranslateDocument", `{"Document":{"Content":"aGVsbG8=","ContentType":"text/plain"},"SourceLanguageCode":"en","TargetLanguageCode":"fr"}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "TranslatedDocument") {
		t.Fatalf("expected TranslateDocument to include TranslatedDocument, got %q", body)
	}
}

func TestTranslateStage34TerminologyAndParallelDataLifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := translateRequest(t, ts, "ImportTerminology", `{"Name":"stage-terminology","Description":"stage description","MergeStrategy":"OVERWRITE"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = translateRequest(t, ts, "GetTerminology", `{"Name":"stage-terminology"}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "stage-terminology") {
		t.Fatalf("expected GetTerminology to include stage-terminology, got %q", body)
	}

	resp = translateRequest(t, ts, "ListTerminologies", `{}`)
	assertStatus(t, resp, http.StatusOK)

	resp = translateRequest(t, ts, "CreateParallelData", `{"Name":"stage-parallel-data","Description":"stage pd"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = translateRequest(t, ts, "GetParallelData", `{"Name":"stage-parallel-data"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = translateRequest(t, ts, "ListParallelData", `{}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "stage-parallel-data") {
		t.Fatalf("expected ListParallelData to include stage-parallel-data, got %q", body)
	}

	resp = translateRequest(t, ts, "UpdateParallelData", `{"Name":"stage-parallel-data","Description":"updated pd"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = translateRequest(t, ts, "DeleteParallelData", `{"Name":"stage-parallel-data"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = translateRequest(t, ts, "DeleteTerminology", `{"Name":"stage-terminology"}`)
	assertStatus(t, resp, http.StatusOK)
}

func TestTranslateStage56TextTranslationJobTaggingValidationAndIdempotency(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := translateRequest(t, ts, "StartTextTranslationJob", `{"JobName":"stage-job","ClientToken":"stage-token","SourceLanguageCode":"en","TargetLanguageCode":"de","InputDataConfig":{"S3Uri":"s3://stackyard/input/"},"OutputDataConfig":{"S3Uri":"s3://stackyard/output/"}}`)
	assertStatus(t, resp, http.StatusOK)
	first := string(mustBody(t, resp))

	resp = translateRequest(t, ts, "StartTextTranslationJob", `{"JobName":"stage-job","ClientToken":"stage-token"}`)
	assertStatus(t, resp, http.StatusOK)
	second := string(mustBody(t, resp))
	if first != second {
		t.Fatalf("expected idempotent StartTextTranslationJob response, got %q and %q", first, second)
	}

	resp = translateRequest(t, ts, "ListTextTranslationJobs", `{}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "stage-job") {
		t.Fatalf("expected ListTextTranslationJobs to include stage-job, got %q", body)
	}

	resp = translateRequest(t, ts, "DescribeTextTranslationJob", `{}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "TextTranslationJobProperties") {
		t.Fatalf("expected DescribeTextTranslationJob payload, got %q", body)
	}

	resp = translateRequest(t, ts, "StopTextTranslationJob", `{}`)
	assertStatus(t, resp, http.StatusOK)

	resourceARN := "arn:aws:translate:us-east-1:123456789012:terminology/stage-terminology"
	resp = translateRequest(t, ts, "TagResource", `{"ResourceArn":"`+resourceARN+`","Tags":[{"Key":"env","Value":"stage"},{"Key":"owner","Value":"qa"}]}`)
	assertStatus(t, resp, http.StatusOK)

	resp = translateRequest(t, ts, "ListTagsForResource", `{"ResourceArn":"`+resourceARN+`"}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "owner") {
		t.Fatalf("expected ListTagsForResource to include owner tag, got %q", body)
	}

	resp = translateRequest(t, ts, "UntagResource", `{"ResourceArn":"`+resourceARN+`","TagKeys":["owner"]}`)
	assertStatus(t, resp, http.StatusOK)

	resp = translateRequest(t, ts, "TotallyUnknownAction", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException for unknown action, got %q", body)
	}

	resp = signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		[]byte(`{"broken":`),
		map[string]string{
			"Content-Type": "application/x-amz-json-1.1",
			"X-Amz-Target": "AWSShineFrontendService_20170701.ListLanguages",
		},
		"translate",
	)
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "invalid JSON body") {
		t.Fatalf("expected invalid JSON body error, got %q", body)
	}
}
