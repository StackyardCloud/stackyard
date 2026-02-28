package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func signRequestWithHeaders(t *testing.T, method, urlStr string, body []byte, headers map[string]string, service, region, signedHeaders string) *http.Response {
	t.Helper()
	var reader *bytes.Reader
	if body == nil {
		reader = bytes.NewReader(nil)
	} else {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, urlStr, reader)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	req.Host = req.URL.Host
	amzDate := time.Now().UTC().Format("20060102T150405Z")
	req.Header.Set("x-amz-date", amzDate)
	payloadHash := sha256Hex(body)
	req.Header.Set("x-amz-content-sha256", payloadHash)
	if signedHeaders == "" {
		signedHeaders = buildSignedHeaders(req)
	}
	canonicalRequest, _ := buildCanonicalRequest(req, signedHeaders, payloadHash, false)
	stringToSign := buildStringToSign(amzDate, region, service, canonicalRequest)
	signature := signString(stringToSign, region, service, testSecretKey)
	credentialScope := amzDate[:8] + "/" + region + "/" + service + "/aws4_request"
	auth := "AWS4-HMAC-SHA256 Credential=" + testAccessKey + "/" + credentialScope + ", SignedHeaders=" + signedHeaders + ", Signature=" + signature
	req.Header.Set("Authorization", auth)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	return resp
}

func TestS3ControlStage5RequiresSignedAccountHeader(t *testing.T) {
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
	resp := signRequestWithHeaders(t, http.MethodGet, ts.URL+"/v20180820/accesspoint", nil, headers, "s3-control", testRegion, "host;x-amz-content-sha256;x-amz-date")
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 for unsigned account header, got %d", resp.StatusCode)
	}
}

func TestS3ControlStage5AcceptsSignedAccountHeader(t *testing.T) {
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
	resp := signRequestWithHeaders(t, http.MethodGet, ts.URL+"/v20180820/accesspoint", nil, headers, "s3-control", testRegion, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 with signed account header, got %d", resp.StatusCode)
	}
}

func TestS3ControlStage5MissingRegionInScope(t *testing.T) {
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
	resp := signRequestWithHeaders(t, http.MethodGet, ts.URL+"/v20180820/accesspoint", nil, headers, "s3-control", "", "")
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 for missing region, got %d", resp.StatusCode)
	}
}
