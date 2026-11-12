package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestS3UpdateObjectEncryptionTargetsObjectRoute(t *testing.T) {
	srv := New(Config{
		Addr:      ":0",
		AccessKey: testAccessKey,
		SecretKey: testSecretKey,
		LogLevel:  "error",
	})
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)

	bucket := "update-object-encryption"
	resp := signedRequest(t, http.MethodPut, ts.URL+"/"+bucket, nil, nil)
	assertStatus(t, resp, http.StatusOK)

	resp = signedRequest(t, http.MethodPut, ts.URL+"/"+bucket+"/notes/encrypted.txt", []byte("payload"), nil)
	assertStatus(t, resp, http.StatusOK)

	body := []byte(
		`<ObjectEncryption xmlns="http://s3.amazonaws.com/doc/2006-03-01/"><SSE-KMS><KMSKeyArn>` +
			`arn:aws:kms:us-east-1:123456789012:key/stackyard-test` +
			`</KMSKeyArn><BucketKeyEnabled>false</BucketKeyEnabled></SSE-KMS></ObjectEncryption>`,
	)
	req, _ := http.NewRequest(http.MethodPut, ts.URL+"/"+bucket+"/notes/encrypted.txt?encryption", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/xml")
	resp = signedRequestWithRequest(t, req, body)
	assertStatus(t, resp, http.StatusOK)

	req, _ = http.NewRequest(http.MethodGet, ts.URL+"/"+bucket+"/notes/encrypted.txt", nil)
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusOK)
	if got := resp.Header.Get("x-amz-server-side-encryption"); got != "aws:kms" {
		t.Fatalf("expected aws:kms SSE header after UpdateObjectEncryption, got %q", got)
	}
}
