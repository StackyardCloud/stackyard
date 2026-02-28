package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTranscribeStage12TranscriptionAndVocabularyLifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := transcribeRequest(t, ts, "StartTranscriptionJob", `{"TranscriptionJobName":"stage-transcription-job","LanguageCode":"en-US"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = transcribeRequest(t, ts, "GetTranscriptionJob", `{"TranscriptionJobName":"stage-transcription-job"}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "stage-transcription-job") {
		t.Fatalf("expected GetTranscriptionJob to include stage-transcription-job, got %q", body)
	}

	resp = transcribeRequest(t, ts, "ListTranscriptionJobs", `{}`)
	assertStatus(t, resp, http.StatusOK)

	resp = transcribeRequest(t, ts, "DeleteTranscriptionJob", `{"TranscriptionJobName":"stage-transcription-job"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = transcribeRequest(t, ts, "CreateVocabulary", `{"VocabularyName":"stage-vocabulary","LanguageCode":"en-US"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = transcribeRequest(t, ts, "GetVocabulary", `{"VocabularyName":"stage-vocabulary"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = transcribeRequest(t, ts, "ListVocabularies", `{}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "stage-vocabulary") {
		t.Fatalf("expected ListVocabularies to include stage-vocabulary, got %q", body)
	}

	resp = transcribeRequest(t, ts, "UpdateVocabulary", `{"VocabularyName":"stage-vocabulary","LanguageCode":"en-US"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = transcribeRequest(t, ts, "DeleteVocabulary", `{"VocabularyName":"stage-vocabulary"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = transcribeRequest(t, ts, "CreateVocabularyFilter", `{"VocabularyFilterName":"stage-vocabulary-filter","LanguageCode":"en-US"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = transcribeRequest(t, ts, "GetVocabularyFilter", `{"VocabularyFilterName":"stage-vocabulary-filter"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = transcribeRequest(t, ts, "ListVocabularyFilters", `{}`)
	assertStatus(t, resp, http.StatusOK)

	resp = transcribeRequest(t, ts, "UpdateVocabularyFilter", `{"VocabularyFilterName":"stage-vocabulary-filter","LanguageCode":"en-US"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = transcribeRequest(t, ts, "DeleteVocabularyFilter", `{"VocabularyFilterName":"stage-vocabulary-filter"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = transcribeRequest(t, ts, "CreateMedicalVocabulary", `{"VocabularyName":"stage-medical-vocabulary","LanguageCode":"en-US"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = transcribeRequest(t, ts, "GetMedicalVocabulary", `{"VocabularyName":"stage-medical-vocabulary"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = transcribeRequest(t, ts, "ListMedicalVocabularies", `{}`)
	assertStatus(t, resp, http.StatusOK)

	resp = transcribeRequest(t, ts, "UpdateMedicalVocabulary", `{"VocabularyName":"stage-medical-vocabulary","LanguageCode":"en-US"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = transcribeRequest(t, ts, "DeleteMedicalVocabulary", `{"VocabularyName":"stage-medical-vocabulary"}`)
	assertStatus(t, resp, http.StatusOK)
}

func TestTranscribeStage34MedicalAnalyticsAndLanguageModelLifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := transcribeRequest(t, ts, "StartMedicalTranscriptionJob", `{"MedicalTranscriptionJobName":"stage-medical-transcription-job","LanguageCode":"en-US"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = transcribeRequest(t, ts, "GetMedicalTranscriptionJob", `{"MedicalTranscriptionJobName":"stage-medical-transcription-job"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = transcribeRequest(t, ts, "ListMedicalTranscriptionJobs", `{}`)
	assertStatus(t, resp, http.StatusOK)
	resp = transcribeRequest(t, ts, "DeleteMedicalTranscriptionJob", `{"MedicalTranscriptionJobName":"stage-medical-transcription-job"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = transcribeRequest(t, ts, "StartMedicalScribeJob", `{"MedicalScribeJobName":"stage-medical-scribe-job","LanguageCode":"en-US"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = transcribeRequest(t, ts, "GetMedicalScribeJob", `{"MedicalScribeJobName":"stage-medical-scribe-job"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = transcribeRequest(t, ts, "ListMedicalScribeJobs", `{}`)
	assertStatus(t, resp, http.StatusOK)
	resp = transcribeRequest(t, ts, "DeleteMedicalScribeJob", `{"MedicalScribeJobName":"stage-medical-scribe-job"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = transcribeRequest(t, ts, "CreateCallAnalyticsCategory", `{"CategoryName":"stage-call-analytics-category"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = transcribeRequest(t, ts, "UpdateCallAnalyticsCategory", `{"CategoryName":"stage-call-analytics-category"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = transcribeRequest(t, ts, "GetCallAnalyticsCategory", `{"CategoryName":"stage-call-analytics-category"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = transcribeRequest(t, ts, "ListCallAnalyticsCategories", `{}`)
	assertStatus(t, resp, http.StatusOK)
	resp = transcribeRequest(t, ts, "DeleteCallAnalyticsCategory", `{"CategoryName":"stage-call-analytics-category"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = transcribeRequest(t, ts, "StartCallAnalyticsJob", `{"CallAnalyticsJobName":"stage-call-analytics-job","LanguageCode":"en-US"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = transcribeRequest(t, ts, "GetCallAnalyticsJob", `{"CallAnalyticsJobName":"stage-call-analytics-job"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = transcribeRequest(t, ts, "ListCallAnalyticsJobs", `{}`)
	assertStatus(t, resp, http.StatusOK)
	resp = transcribeRequest(t, ts, "DeleteCallAnalyticsJob", `{"CallAnalyticsJobName":"stage-call-analytics-job"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = transcribeRequest(t, ts, "CreateLanguageModel", `{"ModelName":"stage-language-model"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = transcribeRequest(t, ts, "DescribeLanguageModel", `{"ModelName":"stage-language-model"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = transcribeRequest(t, ts, "ListLanguageModels", `{}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "stage-language-model") {
		t.Fatalf("expected ListLanguageModels to include stage-language-model, got %q", body)
	}
	resp = transcribeRequest(t, ts, "DeleteLanguageModel", `{"ModelName":"stage-language-model"}`)
	assertStatus(t, resp, http.StatusOK)
}

func TestTranscribeStage56TaggingValidationAndIdempotency(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resourceARN := "arn:aws:transcribe:us-east-1:123456789012:transcription-job/stage-transcription-job"
	resp := transcribeRequest(t, ts, "TagResource", `{"ResourceArn":"`+resourceARN+`","Tags":[{"Key":"env","Value":"stage"},{"Key":"owner","Value":"qa"}]}`)
	assertStatus(t, resp, http.StatusOK)

	resp = transcribeRequest(t, ts, "ListTagsForResource", `{"ResourceArn":"`+resourceARN+`"}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "owner") {
		t.Fatalf("expected ListTagsForResource to include owner tag, got %q", body)
	}

	resp = transcribeRequest(t, ts, "UntagResource", `{"ResourceArn":"`+resourceARN+`","TagKeys":["owner"]}`)
	assertStatus(t, resp, http.StatusOK)

	resp = transcribeRequest(t, ts, "CreateVocabulary", `{"VocabularyName":"stage-idempotent-vocabulary","LanguageCode":"en-US"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = transcribeRequest(t, ts, "CreateVocabulary", `{"VocabularyName":"stage-idempotent-vocabulary","LanguageCode":"en-US"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = transcribeRequest(t, ts, "TotallyUnknownAction", `{}`)
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
			"X-Amz-Target": "Transcribe.ListTranscriptionJobs",
		},
		"transcribe",
	)
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "invalid JSON body") {
		t.Fatalf("expected invalid JSON body error, got %q", body)
	}
}
