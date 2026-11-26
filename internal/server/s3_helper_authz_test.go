package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestS3HelperRoutesApplyS3Authorization checks that the /s3 JSON helper
// routes give the same allow/deny outcome as the S3 API for the same caller.
func TestS3HelperRoutesApplyS3Authorization(t *testing.T) {
	srv := New(Config{Addr: ":0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "info"})
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)

	mustOK := func(resp *http.Response) {
		t.Helper()
		if resp.StatusCode >= 300 {
			t.Fatalf("setup request failed: %d %s", resp.StatusCode, mustBody(t, resp))
		}
		_ = resp.Body.Close()
	}
	// open: no policy, private ACL, one private and one public-read object.
	mustOK(signedRequest(t, "PUT", ts.URL+"/open", nil, nil))
	mustOK(signedRequest(t, "PUT", ts.URL+"/open/private.txt", []byte("p"), nil))
	mustOK(signedRequest(t, "PUT", ts.URL+"/open/public.txt", []byte("p"), map[string]string{"x-amz-acl": "public-read"}))
	// denied: a bucket policy denying the signing caller everything.
	mustOK(signedRequest(t, "PUT", ts.URL+"/denied", nil, nil))
	mustOK(signedRequest(t, "PUT", ts.URL+"/denied/obj.txt", []byte("p"), map[string]string{"x-amz-acl": "public-read"}))
	policy := `{"Version":"2012-10-17","Statement":[{"Effect":"Deny","Principal":{"AWS":"` + testAccessKey +
		`"},"Action":"s3:*","Resource":["arn:aws:s3:::denied","arn:aws:s3:::denied/*"]}]}`
	mustOK(signedRequest(t, "PUT", ts.URL+"/denied?policy", []byte(policy), nil))

	type call int
	const (
		unsigned call = iota
		signed
		badSig
	)
	do := func(method, path string, body []byte, how call) int {
		t.Helper()
		var resp *http.Response
		switch how {
		case signed:
			resp = signedRequest(t, method, ts.URL+path, body, nil)
		default:
			req, err := http.NewRequest(method, ts.URL+path, bytes.NewReader(body))
			if err != nil {
				t.Fatal(err)
			}
			if how == badSig {
				req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential="+testAccessKey+"/20240101/us-east-1/s3/aws4_request, SignedHeaders=host, Signature=bad")
				req.Header.Set("x-amz-date", "20240101T000000Z")
			}
			r, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			resp = r
		}
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusForbidden && strings.HasPrefix(path, "/s3/") {
			if b := string(mustBody(t, resp)); !strings.Contains(b, `"error"`) {
				t.Fatalf("%s %s: expected JSON error body, got %q", method, path, b)
			}
		}
		return resp.StatusCode
	}

	cases := []struct {
		name       string
		method     string
		helper, s3 string // helper path and the equivalent S3 API path
		how        call
		allowed    bool
	}{
		{"list objects: deny policy", "GET", "/s3/objects/denied", "/denied", signed, false},
		{"list objects: no policy signed", "GET", "/s3/objects/open", "/open", signed, true},
		{"list objects: unsigned private bucket", "GET", "/s3/objects/open", "/open", unsigned, false},
		{"list objects: bad signature", "GET", "/s3/objects/open", "/open", badSig, false},

		{"get object: deny policy", "GET", "/s3/objects/denied/obj.txt", "/denied/obj.txt", signed, false},
		{"get object: no policy signed", "GET", "/s3/objects/open/private.txt", "/open/private.txt", signed, true},
		{"get object: unsigned private object", "GET", "/s3/objects/open/private.txt", "/open/private.txt", unsigned, false},
		{"get object: unsigned public-read object", "GET", "/s3/objects/open/public.txt", "/open/public.txt", unsigned, true},
		{"get object: bad signature", "GET", "/s3/objects/open/public.txt", "/open/public.txt", badSig, false},

		{"put object: deny policy", "PUT", "/s3/objects/denied/new.txt", "/denied/new.txt", signed, false},
		{"put object: no policy signed", "PUT", "/s3/objects/open/new.txt", "/open/new.txt", signed, true},
		{"put object: unsigned private bucket", "PUT", "/s3/objects/open/new2.txt", "/open/new2.txt", unsigned, false},
		{"put object: bad signature", "PUT", "/s3/objects/open/new3.txt", "/open/new3.txt", badSig, false},

		{"create bucket: signed", "PUT", "/s3/buckets/made-a", "/made-b", signed, true},
		{"create bucket: unsigned", "PUT", "/s3/buckets/made-c", "/made-d", unsigned, false},
		{"create bucket: bad signature", "PUT", "/s3/buckets/made-e", "/made-f", badSig, false},

		{"list buckets: signed", "GET", "/s3/buckets", "/", signed, true},
		{"list buckets: unsigned", "GET", "/s3/buckets", "/", unsigned, true},
		{"list buckets: bad signature", "GET", "/s3/buckets", "/", badSig, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body := []byte(nil)
			if tc.method == "PUT" && strings.Contains(tc.helper, "/objects/") {
				body = []byte("data")
			}
			got := do(tc.method, tc.helper, body, tc.how)
			if tc.allowed && got >= 300 {
				t.Fatalf("helper %s %s: want allowed, got %d", tc.method, tc.helper, got)
			}
			if !tc.allowed && got != http.StatusForbidden {
				t.Fatalf("helper %s %s: want 403, got %d", tc.method, tc.helper, got)
			}
			if s3Got := do(tc.method, tc.s3, body, tc.how); (s3Got < 300) != (got < 300) {
				t.Fatalf("parity: helper %s returned %d but S3 API %s returned %d", tc.helper, got, tc.s3, s3Got)
			}
		})
	}
}
