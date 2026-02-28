package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"
	"time"
)

func signedRequestWithService(t *testing.T, method, urlStr string, body []byte, headers map[string]string, service string) *http.Response {
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
	signedHeaders := buildSignedHeaders(req)
	canonicalRequest, _ := buildCanonicalRequest(req, signedHeaders, payloadHash, false)
	stringToSign := buildStringToSign(amzDate, testRegion, service, canonicalRequest)
	signature := signString(stringToSign, testRegion, service, testSecretKey)
	credentialScope := amzDate[:8] + "/" + testRegion + "/" + service + "/aws4_request"
	auth := "AWS4-HMAC-SHA256 Credential=" + testAccessKey + "/" + credentialScope + ", SignedHeaders=" + signedHeaders + ", Signature=" + signature
	req.Header.Set("Authorization", auth)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	return resp
}

func buildSignedHeaders(req *http.Request) string {
	base := []string{"host", "x-amz-content-sha256", "x-amz-date"}
	seen := map[string]bool{
		"host":                 true,
		"x-amz-content-sha256": true,
		"x-amz-date":           true,
	}
	for key := range req.Header {
		lower := strings.ToLower(key)
		if strings.HasPrefix(lower, "x-amz-") && !seen[lower] {
			seen[lower] = true
			base = append(base, lower)
		}
	}
	sort.Strings(base)
	return strings.Join(base, ";")
}

func TestS3ControlStage0Validation(t *testing.T) {
	srv := New(Config{
		Addr:      "127.0.0.1:0",
		AccessKey: testAccessKey,
		SecretKey: testSecretKey,
		LogLevel:  "error",
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	urlStr := ts.URL + "/v20180820/accesspoint"

	resp := signedRequestWithService(t, http.MethodGet, urlStr, nil, map[string]string{}, "s3-control")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing account id, got %d", resp.StatusCode)
	}

	resp = signedRequestWithService(t, http.MethodGet, urlStr, nil, map[string]string{"x-amz-account-id": "bad"}, "s3-control")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid account id, got %d", resp.StatusCode)
	}

	resp = signedRequestWithService(t, http.MethodGet, urlStr, nil, map[string]string{"x-amz-account-id": "123456789012"}, "sqs")
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 for wrong service signing, got %d", resp.StatusCode)
	}

	resp = signedRequestWithService(t, http.MethodGet, urlStr, nil, map[string]string{"x-amz-account-id": "123456789012"}, "s3")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for s3 signing, got %d", resp.StatusCode)
	}

	resp = signedRequestWithService(t, http.MethodGet, urlStr, nil, map[string]string{"x-amz-account-id": "123456789012"}, "s3-outposts")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for s3-outposts signing, got %d", resp.StatusCode)
	}

	resp = signedRequestWithService(t, http.MethodGet, urlStr, nil, map[string]string{"x-amz-account-id": "123456789012"}, "s3-control")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for list access points, got %d", resp.StatusCode)
	}
}
