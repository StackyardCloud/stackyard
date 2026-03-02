package server

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestEndpointForDebugLogPrefersTargetHeader(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "http://stackyard.local/", nil)
	req.Header.Set("X-Amz-Target", "TrentService.CreateKey")

	got := endpointForDebugLog(req)
	if got != "TrentService.CreateKey" {
		t.Fatalf("expected X-Amz-Target endpoint, got %q", got)
	}
}

func TestEndpointForDebugLogReadsActionFromFormBody(t *testing.T) {
	body := "Action=CreateTopic&Version=2010-03-31"
	req := httptest.NewRequest(http.MethodPost, "http://stackyard.local/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	got := endpointForDebugLog(req)
	if got != "Action:CreateTopic" {
		t.Fatalf("expected Action endpoint from form body, got %q", got)
	}

	// Middleware helpers must preserve request body readability.
	replay, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatalf("read request body after endpoint extraction: %v", err)
	}
	if string(replay) != body {
		t.Fatalf("expected request body to be preserved, got %q", string(replay))
	}
}

func TestDebugLoggingIncludesEndpoint(t *testing.T) {
	srv := New(Config{
		Addr:      "127.0.0.1:0",
		AccessKey: testAccessKey,
		SecretKey: testSecretKey,
		LogLevel:  "debug",
	})
	req := httptest.NewRequest(http.MethodPost, "http://stackyard.local/", nil)
	req.Header.Set("X-Amz-Target", "TrentService.CreateKey")
	rec := httptest.NewRecorder()

	var logs bytes.Buffer
	originalWriter := log.Writer()
	originalFlags := log.Flags()
	originalPrefix := log.Prefix()
	log.SetOutput(&logs)
	log.SetFlags(0)
	log.SetPrefix("")
	t.Cleanup(func() {
		log.SetOutput(originalWriter)
		log.SetFlags(originalFlags)
		log.SetPrefix(originalPrefix)
	})

	handler := srv.loggingMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	handler.ServeHTTP(rec, req)

	if got := logs.String(); !strings.Contains(got, `endpoint="TrentService.CreateKey"`) {
		t.Fatalf("expected debug logs to include endpoint label, got %q", got)
	}
}
