package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func readOutpostsFixture(t *testing.T, name string) []byte {
	t.Helper()
	path := filepath.Join("testdata", "s3outposts", name)
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return body
}

func assertJSONEqual(t *testing.T, got, want []byte) {
	t.Helper()
	var gotVal any
	var wantVal any
	if err := json.Unmarshal(bytes.TrimSpace(got), &gotVal); err != nil {
		t.Fatalf("decode got JSON: %v", err)
	}
	if err := json.Unmarshal(bytes.TrimSpace(want), &wantVal); err != nil {
		t.Fatalf("decode want JSON: %v", err)
	}
	if !reflect.DeepEqual(gotVal, wantVal) {
		gotPretty, _ := json.Marshal(gotVal)
		wantPretty, _ := json.Marshal(wantVal)
		t.Fatalf("unexpected JSON response: got=%s want=%s", string(gotPretty), string(wantPretty))
	}
}

func TestS3OutpostsStage0CreateEndpointFixtures(t *testing.T) {
	srv := New(Config{
		Addr:      "127.0.0.1:0",
		AccessKey: testAccessKey,
		SecretKey: testSecretKey,
		LogLevel:  "error",
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	reqBody := readOutpostsFixture(t, "create-endpoint-request.json")
	expected := readOutpostsFixture(t, "create-endpoint-response.json")
	headers := map[string]string{
		"Content-Type":     "application/json",
		"x-amz-account-id": "123456789012",
	}
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/S3Outposts/CreateEndpoint", reqBody, headers, "s3-outposts")
	assertStatus(t, resp, http.StatusOK)
	body := mustBody(t, resp)
	assertJSONEqual(t, body, expected)
}

func TestS3OutpostsStage0CreateEndpointUnknownField(t *testing.T) {
	srv := New(Config{
		Addr:      "127.0.0.1:0",
		AccessKey: testAccessKey,
		SecretKey: testSecretKey,
		LogLevel:  "error",
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	reqBody := []byte(`{"OutpostId":"op-0123456789abcdef0","SecurityGroupId":"sg-0123456789abcdef0","SubnetId":"subnet-0123456789abcdef0","Unknown":"x"}`)
	headers := map[string]string{
		"Content-Type":     "application/json",
		"x-amz-account-id": "123456789012",
	}
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/S3Outposts/CreateEndpoint", reqBody, headers, "s3-outposts")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid JSON, got %d", resp.StatusCode)
	}
}

func TestS3OutpostsStage0DeleteEndpointErrors(t *testing.T) {
	srv := New(Config{
		Addr:      "127.0.0.1:0",
		AccessKey: testAccessKey,
		SecretKey: testSecretKey,
		LogLevel:  "error",
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	headers := map[string]string{
		"x-amz-account-id": "123456789012",
	}
	resp := signedRequestWithService(t, http.MethodDelete, ts.URL+"/S3Outposts/DeleteEndpoint?endpointId=1234567890123456789", nil, headers, "s3-outposts")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing outpostId, got %d", resp.StatusCode)
	}
	assertJSONEqual(t, mustBody(t, resp), readOutpostsFixture(t, "error-validation.json"))

	resp = signedRequestWithService(t, http.MethodDelete, ts.URL+"/S3Outposts/DeleteEndpoint?outpostId=op-0123456789abcdef0&endpointId=0000000000000000000", nil, headers, "s3-outposts")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for missing endpoint, got %d", resp.StatusCode)
	}
	assertJSONEqual(t, mustBody(t, resp), readOutpostsFixture(t, "error-not-found.json"))
}

func TestS3OutpostsStage0DeleteEndpointFixture(t *testing.T) {
	srv := New(Config{
		Addr:      "127.0.0.1:0",
		AccessKey: testAccessKey,
		SecretKey: testSecretKey,
		LogLevel:  "error",
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	headers := map[string]string{
		"x-amz-account-id": "123456789012",
	}
	resp := signedRequestWithService(t, http.MethodDelete, ts.URL+"/S3Outposts/DeleteEndpoint?outpostId=op-0123456789abcdef0&endpointId=1234567890123456789", nil, headers, "s3-outposts")
	assertStatus(t, resp, http.StatusOK)
	assertJSONEqual(t, mustBody(t, resp), readOutpostsFixture(t, "delete-endpoint-response.json"))
}

func TestS3OutpostsStage0ListEndpointsFixture(t *testing.T) {
	srv := New(Config{
		Addr:      "127.0.0.1:0",
		AccessKey: testAccessKey,
		SecretKey: testSecretKey,
		LogLevel:  "error",
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	headers := map[string]string{
		"x-amz-account-id": "123456789012",
	}
	resp := signedRequestWithService(t, http.MethodGet, ts.URL+"/S3Outposts/ListEndpoints", nil, headers, "s3-outposts")
	assertStatus(t, resp, http.StatusOK)
	assertJSONEqual(t, mustBody(t, resp), readOutpostsFixture(t, "list-endpoints-response.json"))
}

func TestS3OutpostsStage0ListOutpostsWithS3Fixture(t *testing.T) {
	srv := New(Config{
		Addr:      "127.0.0.1:0",
		AccessKey: testAccessKey,
		SecretKey: testSecretKey,
		LogLevel:  "error",
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	headers := map[string]string{
		"x-amz-account-id": "123456789012",
	}
	resp := signedRequestWithService(t, http.MethodGet, ts.URL+"/S3Outposts/ListOutpostsWithS3", nil, headers, "s3-outposts")
	assertStatus(t, resp, http.StatusOK)
	assertJSONEqual(t, mustBody(t, resp), readOutpostsFixture(t, "list-outposts-with-s3-response.json"))
}

func TestS3OutpostsStage0ListSharedEndpointsFixture(t *testing.T) {
	srv := New(Config{
		Addr:      "127.0.0.1:0",
		AccessKey: testAccessKey,
		SecretKey: testSecretKey,
		LogLevel:  "error",
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	headers := map[string]string{
		"x-amz-account-id": "123456789012",
	}
	resp := signedRequestWithService(t, http.MethodGet, ts.URL+"/S3Outposts/ListSharedEndpoints", nil, headers, "s3-outposts")
	assertStatus(t, resp, http.StatusOK)
	assertJSONEqual(t, mustBody(t, resp), readOutpostsFixture(t, "list-shared-endpoints-response.json"))

	resp = signedRequestWithService(t, http.MethodGet, ts.URL+"/S3Outposts/ListSharedEndpoints?outpostId=op-0123456789abcdef0", nil, headers, "s3-outposts")
	assertStatus(t, resp, http.StatusOK)
	assertJSONEqual(t, mustBody(t, resp), readOutpostsFixture(t, "list-shared-endpoints-response.json"))
}

func TestS3OutpostsStage0RequiresS3OutpostsSigning(t *testing.T) {
	srv := New(Config{
		Addr:      "127.0.0.1:0",
		AccessKey: testAccessKey,
		SecretKey: testSecretKey,
		LogLevel:  "error",
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	headers := map[string]string{
		"x-amz-account-id": "123456789012",
	}
	resp := signedRequestWithService(t, http.MethodGet, ts.URL+"/S3Outposts/ListEndpoints", nil, headers, "s3-control")
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 for wrong signing service, got %d", resp.StatusCode)
	}
}

func TestS3OutpostsStage1Validation(t *testing.T) {
	srv := New(Config{
		Addr:      "127.0.0.1:0",
		AccessKey: testAccessKey,
		SecretKey: testSecretKey,
		LogLevel:  "error",
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	headers := map[string]string{
		"Content-Type":     "application/json",
		"x-amz-account-id": "123456789012",
	}

	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/S3Outposts/CreateEndpoint", []byte(`{"OutpostId":"op-0123456789abcdef0"}`), headers, "s3-outposts")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing fields, got %d", resp.StatusCode)
	}

	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/S3Outposts/CreateEndpoint", []byte(`{"OutpostId":"op-0123456789abcdef0","SecurityGroupId":"sg-0123456789abcdef0","SubnetId":"subnet-0123456789abcdef0","AccessType":"CustomerOwnedIp"}`), headers, "s3-outposts")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing CustomerOwnedIpv4Pool, got %d", resp.StatusCode)
	}

	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/S3Outposts/CreateEndpoint", []byte(`<CreateEndpoint/>`), map[string]string{
		"Content-Type":     "application/xml",
		"x-amz-account-id": "123456789012",
	}, "s3-outposts")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for malformed JSON/XML, got %d", resp.StatusCode)
	}
}
