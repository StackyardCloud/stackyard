package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func comprehendRequest(t *testing.T, ts *httptest.Server, action string, payload string) *http.Response {
	t.Helper()
	return signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		[]byte(payload),
		map[string]string{
			"Content-Type": "application/x-amz-json-1.1",
			"X-Amz-Target": "Comprehend_20171127." + action,
		},
		"comprehend",
	)
}

func TestComprehendStage0CatalogCoverage(t *testing.T) {
	if len(comprehendOperations) != 85 {
		t.Fatalf("expected 85 Comprehend operations from docs, got %d", len(comprehendOperations))
	}
	if len(comprehendOperationByName) != len(comprehendOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"DetectSentiment",
		"BatchDetectEntities",
		"CreateEndpoint",
		"ListFlywheels",
		"StartDocumentClassificationJob",
		"StopTrainingDocumentClassifier",
	}
	for _, action := range requiredActions {
		if _, ok := comprehendOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(comprehendDataTypes) != 106 {
		t.Fatalf("expected 106 Comprehend data types from docs, got %d", len(comprehendDataTypes))
	}
	if len(comprehendDataTypeByName) != len(comprehendDataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"BatchItemError",
		"DocumentClassifierProperties",
		"EndpointProperties",
		"EntityRecognizerProperties",
		"FlywheelProperties",
		"SentimentScore",
	}
	for _, typeName := range requiredTypes {
		if _, ok := comprehendDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestComprehendStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := comprehendRequest(t, ts, "TotallyUnknownAction", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestComprehendKnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := comprehendRequest(t, ts, "DetectSentiment", `{"Text":"hello","LanguageCode":"en"}`)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplementedException") {
		t.Fatalf("did not expect NotImplementedException response body, got %q", body)
	}
	if !strings.Contains(body, "Sentiment") {
		t.Fatalf("expected DetectSentiment response body to include Sentiment, got %q", body)
	}
}
