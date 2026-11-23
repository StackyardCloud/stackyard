package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestS3ObjectAnnotations(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()
	obj := ts.URL + "/ann-bucket/doc.txt"

	assertStatus(t, signedRequest(t, http.MethodPut, ts.URL+"/ann-bucket", nil, nil), http.StatusOK)
	assertStatus(t, signedRequest(t, http.MethodPut, obj+"?annotation&annotationName=a1", []byte("x"), nil), http.StatusNotFound)
	assertStatus(t, signedRequest(t, http.MethodPut, obj, []byte("body"), nil), http.StatusOK)

	resp := signedRequest(t, http.MethodPut, obj+"?annotation&annotationName=a1", []byte(`{"k":"v"}`), nil)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "<AnnotationName>a1</AnnotationName>") {
		t.Fatalf("put body: %s", body)
	}
	resp = signedRequest(t, http.MethodGet, obj+"?annotation&annotationName=a1", nil, nil)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); body != `{"k":"v"}` || resp.Header.Get("Content-Type") != "application/octet-stream" {
		t.Fatalf("get: %q %q", body, resp.Header.Get("Content-Type"))
	}
	resp = signedRequest(t, http.MethodGet, obj+"?annotation", nil, nil)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "<Annotations><AnnotationEntry><AnnotationName>a1</AnnotationName>") ||
		!strings.Contains(body, "<AnnotationCount>1</AnnotationCount>") {
		t.Fatalf("list: %s", body)
	}
	assertStatus(t, signedRequest(t, http.MethodDelete, obj+"?annotation&annotationName=a1", nil, nil), http.StatusNoContent)
	assertStatus(t, signedRequest(t, http.MethodDelete, obj+"?annotation&annotationName=a1", nil, nil), http.StatusNotFound)

	cfg := []byte(`<AnnotationTableConfiguration><ConfigurationState>ENABLED</ConfigurationState></AnnotationTableConfiguration>`)
	assertStatus(t, signedRequest(t, http.MethodPut, ts.URL+"/ann-bucket?metadataAnnotationTable", cfg, nil), http.StatusOK)
	assertStatus(t, signedRequest(t, http.MethodPut, ts.URL+"/ann-bucket?metadataAnnotationTable", []byte(`<x/>`), nil), http.StatusBadRequest)
}
