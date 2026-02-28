package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

func readS3TablesFixture(t *testing.T, name string) []byte {
	t.Helper()
	path := filepath.Join("testdata", "s3tables", name)
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return body
}

func TestS3TablesStage0CreateTableBucketFixture(t *testing.T) {
	srv := New(Config{
		Addr:      "127.0.0.1:0",
		AccessKey: testAccessKey,
		SecretKey: testSecretKey,
		LogLevel:  "error",
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	reqBody := readS3TablesFixture(t, "create-table-bucket-request.json")
	want := readS3TablesFixture(t, "create-table-bucket-response.json")

	resp := signedRequestWithService(t, http.MethodPut, ts.URL+"/buckets", reqBody, map[string]string{
		"Content-Type": "application/json",
	}, "s3tables")
	assertStatus(t, resp, http.StatusOK)
	assertJSONEqual(t, mustBody(t, resp), want)
}

func TestS3TablesStage0GetTableBucketFixture(t *testing.T) {
	srv := New(Config{
		Addr:      "127.0.0.1:0",
		AccessKey: testAccessKey,
		SecretKey: testSecretKey,
		LogLevel:  "error",
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	want := readS3TablesFixture(t, "get-table-bucket-response.json")
	arn := "arn:aws:s3tables:us-east-1:123456789012:bucket/demo-bucket"
	resp := signedRequestWithService(t, http.MethodGet, ts.URL+"/buckets/"+url.PathEscape(arn), nil, nil, "s3tables")
	assertStatus(t, resp, http.StatusOK)
	assertJSONEqual(t, mustBody(t, resp), want)
}

func TestS3TablesStage0ListTableBucketsFixture(t *testing.T) {
	srv := New(Config{
		Addr:      "127.0.0.1:0",
		AccessKey: testAccessKey,
		SecretKey: testSecretKey,
		LogLevel:  "error",
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	want := readS3TablesFixture(t, "list-table-buckets-response.json")
	resp := signedRequestWithService(t, http.MethodGet, ts.URL+"/buckets", nil, nil, "s3tables")
	assertStatus(t, resp, http.StatusOK)
	assertJSONEqual(t, mustBody(t, resp), want)
}

func TestS3TablesStage0CreateNamespaceFixture(t *testing.T) {
	srv := New(Config{
		Addr:      "127.0.0.1:0",
		AccessKey: testAccessKey,
		SecretKey: testSecretKey,
		LogLevel:  "error",
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	reqBody := readS3TablesFixture(t, "create-namespace-request.json")
	want := readS3TablesFixture(t, "create-namespace-response.json")
	arn := "arn:aws:s3tables:us-east-1:123456789012:bucket/demo-bucket"
	resp := signedRequestWithService(t, http.MethodPut, ts.URL+"/namespaces/"+url.PathEscape(arn), reqBody, map[string]string{
		"Content-Type": "application/json",
	}, "s3tables")
	assertStatus(t, resp, http.StatusOK)
	assertJSONEqual(t, mustBody(t, resp), want)
}

func TestS3TablesStage0ListNamespacesFixture(t *testing.T) {
	srv := New(Config{
		Addr:      "127.0.0.1:0",
		AccessKey: testAccessKey,
		SecretKey: testSecretKey,
		LogLevel:  "error",
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	want := readS3TablesFixture(t, "list-namespaces-response.json")
	arn := "arn:aws:s3tables:us-east-1:123456789012:bucket/demo-bucket"
	resp := signedRequestWithService(t, http.MethodGet, ts.URL+"/namespaces/"+url.PathEscape(arn), nil, nil, "s3tables")
	assertStatus(t, resp, http.StatusOK)
	assertJSONEqual(t, mustBody(t, resp), want)
}

func TestS3TablesStage0InvalidJSON(t *testing.T) {
	srv := New(Config{
		Addr:      "127.0.0.1:0",
		AccessKey: testAccessKey,
		SecretKey: testSecretKey,
		LogLevel:  "error",
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	want := readS3TablesFixture(t, "error-invalid-json.json")
	resp := signedRequestWithService(t, http.MethodPut, ts.URL+"/buckets", []byte("{"), map[string]string{
		"Content-Type": "application/json",
	}, "s3tables")
	assertStatus(t, resp, http.StatusBadRequest)
	assertJSONEqual(t, mustBody(t, resp), want)
}

func TestS3TablesStage0MissingTableBucketARN(t *testing.T) {
	srv := New(Config{
		Addr:      "127.0.0.1:0",
		AccessKey: testAccessKey,
		SecretKey: testSecretKey,
		LogLevel:  "error",
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	want := readS3TablesFixture(t, "error-missing-table-bucket-arn.json")
	resp := signedRequestWithService(t, http.MethodPut, ts.URL+"/namespaces/", []byte(`{"namespace":["analytics"]}`), map[string]string{
		"Content-Type": "application/json",
	}, "s3tables")
	assertStatus(t, resp, http.StatusBadRequest)
	assertJSONEqual(t, mustBody(t, resp), want)
}
