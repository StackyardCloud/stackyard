package server

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"
)

const (
	testAccessKey = "stackyard"
	testSecretKey = "stackyard"
	testRegion    = "us-east-1"
)

func TestS3CoreDataPlaneCompatibility(t *testing.T) {
	srv := New(Config{
		Addr:      ":0",
		AccessKey: testAccessKey,
		SecretKey: testSecretKey,
		LogLevel:  "info",
	})
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)

	bucket := "compat-bucket"

	// Create bucket.
	resp := signedRequest(t, "PUT", ts.URL+"/"+bucket, nil, nil)
	assertStatus(t, resp, http.StatusOK)

	// Access point ARN path support.
	apBucket := "ap-demo"
	resp = signedRequest(t, "PUT", ts.URL+"/"+apBucket, nil, nil)
	assertStatus(t, resp, http.StatusOK)
	apArnPath := ts.URL + "/arn:aws:s3:us-east-1:123456789012:accesspoint/" + apBucket + "/notes/ap.txt"
	resp = signedRequest(t, "PUT", apArnPath, []byte("ap-object"), nil)
	assertStatus(t, resp, http.StatusOK)
	resp = signedRequest(t, "GET", apArnPath, nil, nil)
	assertStatus(t, resp, http.StatusOK)

	// Access point host-style support.
	hostReq, _ := http.NewRequest("PUT", ts.URL+"/notes/host.txt", bytes.NewReader([]byte("host-style")))
	hostReq.Host = "ap-demo-123456789012.s3-accesspoint.us-east-1.amazonaws.com"
	resp = signedRequestWithRequest(t, hostReq, []byte("host-style"))
	assertStatus(t, resp, http.StatusOK)
	hostReq, _ = http.NewRequest("GET", ts.URL+"/notes/host.txt", nil)
	hostReq.Host = "ap-demo-123456789012.s3-accesspoint.us-east-1.amazonaws.com"
	resp = signedRequestWithRequest(t, hostReq, nil)
	assertStatus(t, resp, http.StatusOK)

	// Put object with metadata + Content-MD5.
	body := []byte("hello stackyard")
	sum := md5.Sum(body)
	md5b64 := base64.StdEncoding.EncodeToString(sum[:])
	headers := map[string]string{
		"Content-Type":     "text/plain",
		"Content-MD5":      md5b64,
		"x-amz-meta-owner": "compat",
	}
	resp = signedRequest(t, "PUT", ts.URL+"/"+bucket+"/notes/hello.txt", body, headers)
	assertStatus(t, resp, http.StatusOK)

	versionID := resp.Header.Get("x-amz-version-id")
	if versionID != "" {
		// versioning might be enabled later; allow empty here
	}

	// SSE-S3.
	resp = signedRequest(t, "PUT", ts.URL+"/"+bucket+"/notes/sse.txt", []byte("encrypted"), map[string]string{
		"x-amz-server-side-encryption": "AES256",
	})
	assertStatus(t, resp, http.StatusOK)
	if resp.Header.Get("x-amz-server-side-encryption") != "AES256" {
		t.Fatalf("expected SSE-S3 header on put")
	}
	sseReq, _ := http.NewRequest("GET", ts.URL+"/"+bucket+"/notes/sse.txt", nil)
	resp = signedRequestWithRequest(t, sseReq, nil)
	assertStatus(t, resp, http.StatusOK)
	if resp.Header.Get("x-amz-server-side-encryption") != "AES256" {
		t.Fatalf("expected SSE-S3 header on get")
	}

	// SSE-C.
	customerKey := base64.StdEncoding.EncodeToString([]byte("customer-key-1"))
	customerSum := md5.Sum([]byte("customer-key-1"))
	customerMD5 := base64.StdEncoding.EncodeToString(customerSum[:])
	sseCHeaders := map[string]string{
		"x-amz-server-side-encryption-customer-algorithm": "AES256",
		"x-amz-server-side-encryption-customer-key":       customerKey,
		"x-amz-server-side-encryption-customer-key-md5":   customerMD5,
	}
	resp = signedRequest(t, "PUT", ts.URL+"/"+bucket+"/notes/sse-c.txt", []byte("secret"), sseCHeaders)
	assertStatus(t, resp, http.StatusOK)
	if resp.Header.Get("x-amz-server-side-encryption-customer-algorithm") != "AES256" {
		t.Fatalf("expected SSE-C header on put")
	}
	if resp.Header.Get("x-amz-server-side-encryption-customer-key-md5") != customerMD5 {
		t.Fatalf("expected SSE-C md5 on put")
	}
	sseReq, _ = http.NewRequest("GET", ts.URL+"/"+bucket+"/notes/sse-c.txt", nil)
	resp = signedRequestWithRequest(t, sseReq, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing SSE-C headers, got %d", resp.StatusCode)
	}
	assertS3ErrorCode(t, resp, "InvalidRequest")
	sseReq, _ = http.NewRequest("GET", ts.URL+"/"+bucket+"/notes/sse-c.txt", nil)
	sseReq.Header.Set("x-amz-server-side-encryption-customer-algorithm", "AES256")
	sseReq.Header.Set("x-amz-server-side-encryption-customer-key", customerKey)
	sseReq.Header.Set("x-amz-server-side-encryption-customer-key-md5", "deadbeef")
	resp = signedRequestWithRequest(t, sseReq, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid SSE-C md5, got %d", resp.StatusCode)
	}
	assertS3ErrorCode(t, resp, "InvalidDigest")
	sseReq, _ = http.NewRequest("GET", ts.URL+"/"+bucket+"/notes/sse-c.txt", nil)
	for k, v := range sseCHeaders {
		sseReq.Header.Set(k, v)
	}
	resp = signedRequestWithRequest(t, sseReq, nil)
	assertStatus(t, resp, http.StatusOK)
	if resp.Header.Get("x-amz-server-side-encryption-customer-algorithm") != "AES256" {
		t.Fatalf("expected SSE-C header on get")
	}
	if resp.Header.Get("x-amz-server-side-encryption-customer-key-md5") != customerMD5 {
		t.Fatalf("expected SSE-C md5 on get")
	}
	sseReq, _ = http.NewRequest("HEAD", ts.URL+"/"+bucket+"/notes/sse-c.txt", nil)
	for k, v := range sseCHeaders {
		sseReq.Header.Set(k, v)
	}
	resp = signedRequestWithRequest(t, sseReq, nil)
	assertStatus(t, resp, http.StatusOK)
	if resp.Header.Get("x-amz-server-side-encryption-customer-algorithm") != "AES256" {
		t.Fatalf("expected SSE-C header on head")
	}

	// Content-MD5 mismatch.
	badHeaders := map[string]string{
		"Content-Type": "text/plain",
		"Content-MD5":  base64.StdEncoding.EncodeToString([]byte("badbadbadbadbadb")),
	}
	resp = signedRequest(t, "PUT", ts.URL+"/"+bucket+"/notes/bad.txt", body, badHeaders)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 on invalid digest, got %d", resp.StatusCode)
	}
	assertS3ErrorCode(t, resp, "InvalidDigest")

	// Head object.
	req, _ := http.NewRequest("HEAD", ts.URL+"/"+bucket+"/notes/hello.txt", nil)
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusOK)
	if got := resp.Header.Get("ETag"); got == "" {
		t.Fatalf("expected ETag header")
	}

	// Conditional: If-None-Match => 304.
	req, _ = http.NewRequest("GET", ts.URL+"/"+bucket+"/notes/hello.txt", nil)
	req.Header.Set("If-None-Match", resp.Header.Get("ETag"))
	resp = signedRequestWithRequest(t, req, nil)
	if resp.StatusCode != http.StatusNotModified {
		t.Fatalf("expected 304, got %d", resp.StatusCode)
	}

	// Conditional: If-Match wrong => 412.
	req, _ = http.NewRequest("GET", ts.URL+"/"+bucket+"/notes/hello.txt", nil)
	req.Header.Set("If-Match", `"deadbeef"`)
	resp = signedRequestWithRequest(t, req, nil)
	if resp.StatusCode != http.StatusPreconditionFailed {
		t.Fatalf("expected 412, got %d", resp.StatusCode)
	}
	assertS3ErrorCode(t, resp, "PreconditionFailed")

	// Range read.
	req, _ = http.NewRequest("GET", ts.URL+"/"+bucket+"/notes/hello.txt", nil)
	req.Header.Set("Range", "bytes=0-4")
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusPartialContent)

	// Pre-signed URL + range.
	presigned := presignURL(t, ts.URL+"/"+bucket+"/notes/hello.txt", 300)
	req, _ = http.NewRequest("GET", presigned, nil)
	req.Header.Set("Range", "bytes=0-4")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("presigned request: %v", err)
	}
	if resp.StatusCode != http.StatusPartialContent {
		body := mustBody(t, resp)
		t.Fatalf("expected 206, got %d: %s", resp.StatusCode, string(body))
	}

	// S3 Select (minimal CSV).
	resp = signedRequest(t, "PUT", ts.URL+"/"+bucket+"/notes/select.csv", []byte("a,b\nc,d\n"), map[string]string{
		"Content-Type": "text/csv",
	})
	assertStatus(t, resp, http.StatusOK)
	selectPayload := `<?xml version="1.0" encoding="UTF-8"?>
<SelectObjectContentRequest xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <Expression>SELECT * FROM S3Object</Expression>
  <ExpressionType>SQL</ExpressionType>
  <InputSerialization>
    <CSV>
      <FileHeaderInfo>NONE</FileHeaderInfo>
    </CSV>
  </InputSerialization>
  <OutputSerialization>
    <CSV/>
  </OutputSerialization>
</SelectObjectContentRequest>`
	req, _ = http.NewRequest("POST", ts.URL+"/"+bucket+"/notes/select.csv?select&select-type=2", strings.NewReader(selectPayload))
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusOK)
	selectBody := mustBody(t, resp)
	if string(selectBody) != "a,b\nc,d\n" {
		t.Fatalf("unexpected select output: %q", string(selectBody))
	}

	// S3 Select with WHERE (CSV headers).
	resp = signedRequest(t, "PUT", ts.URL+"/"+bucket+"/notes/select-where.csv", []byte("name,age\nalice,30\nbob,40\n"), map[string]string{
		"Content-Type": "text/csv",
	})
	assertStatus(t, resp, http.StatusOK)
	selectWherePayload := `<?xml version="1.0" encoding="UTF-8"?>
<SelectObjectContentRequest xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <Expression>SELECT * FROM S3Object WHERE name = 'bob'</Expression>
  <ExpressionType>SQL</ExpressionType>
  <InputSerialization>
    <CSV>
      <FileHeaderInfo>USE</FileHeaderInfo>
    </CSV>
  </InputSerialization>
  <OutputSerialization>
    <CSV/>
  </OutputSerialization>
</SelectObjectContentRequest>`
	req, _ = http.NewRequest("POST", ts.URL+"/"+bucket+"/notes/select-where.csv?select&select-type=2", strings.NewReader(selectWherePayload))
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusOK)
	selectBody = mustBody(t, resp)
	if string(selectBody) != "bob,40\n" {
		t.Fatalf("unexpected select where output: %q", string(selectBody))
	}

	// S3 Select JSON + WHERE.
	jsonLines := `{"name":"alpha","count":1}
{"name":"beta","count":2}
`
	resp = signedRequest(t, "PUT", ts.URL+"/"+bucket+"/notes/select.json", []byte(jsonLines), map[string]string{
		"Content-Type": "application/json",
	})
	assertStatus(t, resp, http.StatusOK)
	selectJSONPayload := `<?xml version="1.0" encoding="UTF-8"?>
<SelectObjectContentRequest xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <Expression>SELECT * FROM S3Object WHERE name = 'beta'</Expression>
  <ExpressionType>SQL</ExpressionType>
  <InputSerialization>
    <JSON>
      <Type>LINES</Type>
    </JSON>
  </InputSerialization>
  <OutputSerialization>
    <JSON/>
  </OutputSerialization>
</SelectObjectContentRequest>`
	req, _ = http.NewRequest("POST", ts.URL+"/"+bucket+"/notes/select.json?select&select-type=2", strings.NewReader(selectJSONPayload))
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusOK)
	selectBody = mustBody(t, resp)
	line := strings.TrimSpace(string(selectBody))
	var selectJSON map[string]any
	if err := json.Unmarshal([]byte(line), &selectJSON); err != nil {
		t.Fatalf("select JSON parse: %v (%q)", err, line)
	}
	if selectJSON["name"] != "beta" || selectJSON["count"] != float64(2) {
		t.Fatalf("unexpected select JSON output: %v", selectJSON)
	}

	// S3 Select JSON with AND + nested path + numeric comparisons.
	jsonNested := `{"user":{"name":"alice","age":30},"score":10}
{"user":{"name":"bob","age":40},"score":20}
`
	resp = signedRequest(t, "PUT", ts.URL+"/"+bucket+"/notes/select-nested.json", []byte(jsonNested), map[string]string{
		"Content-Type": "application/json",
	})
	assertStatus(t, resp, http.StatusOK)
	selectNestedPayload := `<?xml version="1.0" encoding="UTF-8"?>
<SelectObjectContentRequest xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <Expression>SELECT * FROM S3Object WHERE user.name = 'bob' AND score >= 20</Expression>
  <ExpressionType>SQL</ExpressionType>
  <InputSerialization>
    <JSON>
      <Type>LINES</Type>
    </JSON>
  </InputSerialization>
  <OutputSerialization>
    <JSON/>
  </OutputSerialization>
</SelectObjectContentRequest>`
	req, _ = http.NewRequest("POST", ts.URL+"/"+bucket+"/notes/select-nested.json?select&select-type=2", strings.NewReader(selectNestedPayload))
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusOK)
	selectBody = mustBody(t, resp)
	line = strings.TrimSpace(string(selectBody))
	var nested map[string]any
	if err := json.Unmarshal([]byte(line), &nested); err != nil {
		t.Fatalf("select nested JSON parse: %v (%q)", err, line)
	}
	userVal, ok := nested["user"].(map[string]any)
	if !ok {
		t.Fatalf("missing user in nested select output: %v", nested)
	}
	if nested["score"] != float64(20) || userVal["name"] != "bob" || userVal["age"] != float64(40) {
		t.Fatalf("unexpected select nested output: %v", nested)
	}

	// S3 Select CSV with AND + numeric comparison.
	resp = signedRequest(t, "PUT", ts.URL+"/"+bucket+"/notes/select-num.csv", []byte("name,score\nalpha,10\nbeta,25\n"), map[string]string{
		"Content-Type": "text/csv",
	})
	assertStatus(t, resp, http.StatusOK)
	selectNumPayload := `<?xml version="1.0" encoding="UTF-8"?>
<SelectObjectContentRequest xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <Expression>SELECT * FROM S3Object WHERE name = 'beta' AND score > 20</Expression>
  <ExpressionType>SQL</ExpressionType>
  <InputSerialization>
    <CSV>
      <FileHeaderInfo>USE</FileHeaderInfo>
    </CSV>
  </InputSerialization>
  <OutputSerialization>
    <CSV/>
  </OutputSerialization>
</SelectObjectContentRequest>`
	req, _ = http.NewRequest("POST", ts.URL+"/"+bucket+"/notes/select-num.csv?select&select-type=2", strings.NewReader(selectNumPayload))
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusOK)
	selectBody = mustBody(t, resp)
	if string(selectBody) != "beta,25\n" {
		t.Fatalf("unexpected select numeric output: %q", string(selectBody))
	}

	// Copy object with precondition failure.
	req, _ = http.NewRequest("PUT", ts.URL+"/"+bucket+"/notes/copied.txt", nil)
	req.Header.Set("x-amz-copy-source", bucket+"/notes/hello.txt")
	req.Header.Set("x-amz-copy-source-if-match", `"deadbeef"`)
	resp = signedRequestWithRequest(t, req, nil)
	if resp.StatusCode != http.StatusPreconditionFailed {
		t.Fatalf("expected 412, got %d", resp.StatusCode)
	}
	assertS3ErrorCode(t, resp, "PreconditionFailed")

	// Copy object success.
	req, _ = http.NewRequest("PUT", ts.URL+"/"+bucket+"/notes/copied.txt", nil)
	req.Header.Set("x-amz-copy-source", bucket+"/notes/hello.txt")
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusOK)

	// Copy object with access point ARN source.
	apSrc := "ap-copy"
	resp = signedRequest(t, "PUT", ts.URL+"/"+apSrc, nil, nil)
	assertStatus(t, resp, http.StatusOK)
	resp = signedRequest(t, "PUT", ts.URL+"/"+apSrc+"/notes/src.txt", []byte("source"), nil)
	assertStatus(t, resp, http.StatusOK)
	req, _ = http.NewRequest("PUT", ts.URL+"/"+bucket+"/notes/copied-ap.txt", nil)
	req.Header.Set("x-amz-copy-source", "arn:aws:s3:us-east-1:123456789012:accesspoint/"+apSrc+"/notes/src.txt")
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusOK)

	// Copy object missing source key.
	req, _ = http.NewRequest("PUT", ts.URL+"/"+bucket+"/notes/missing-copy.txt", nil)
	req.Header.Set("x-amz-copy-source", bucket+"/notes/missing.txt")
	resp = signedRequestWithRequest(t, req, nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for missing copy source key, got %d", resp.StatusCode)
	}
	assertS3ErrorCode(t, resp, "NoSuchKey")

	// Copy object missing destination bucket.
	req, _ = http.NewRequest("PUT", ts.URL+"/missing-bucket/notes/copied.txt", nil)
	req.Header.Set("x-amz-copy-source", bucket+"/notes/hello.txt")
	resp = signedRequestWithRequest(t, req, nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for missing destination bucket, got %d", resp.StatusCode)
	}
	assertS3ErrorCode(t, resp, "NoSuchBucket")

	// Multipart invalid request (uploadId without partNumber).
	req, _ = http.NewRequest("PUT", ts.URL+"/"+bucket+"/notes/multipart.txt?uploadId=invalid", nil)
	resp = signedRequestWithRequest(t, req, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid multipart request, got %d", resp.StatusCode)
	}
	assertS3ErrorCode(t, resp, "InvalidRequest")

	// List pagination.
	signedRequest(t, "PUT", ts.URL+"/"+bucket+"/notes/a.txt", []byte("a"), nil)
	signedRequest(t, "PUT", ts.URL+"/"+bucket+"/notes/b.txt", []byte("b"), nil)
	signedRequest(t, "PUT", ts.URL+"/"+bucket+"/notes/c.txt", []byte("c"), nil)

	respBody := mustBody(t, signedRequest(t, "GET", ts.URL+"/"+bucket+"?list-type=2&max-keys=2", nil, nil))
	var listV2 s3ListBucketResultV2
	if err := xml.Unmarshal(respBody, &listV2); err != nil {
		t.Fatalf("list v2 parse: %v", err)
	}
	if !listV2.IsTruncated || listV2.NextContinuationToken == "" {
		t.Fatalf("expected truncated list with continuation token")
	}

	// Delete object.
	resp = signedRequest(t, "DELETE", ts.URL+"/"+bucket+"/notes/copied.txt", nil, nil)
	assertStatus(t, resp, http.StatusNoContent)

	// Delete objects (multi).
	deletePayload := `<?xml version="1.0" encoding="UTF-8"?>
<Delete>
  <Quiet>true</Quiet>
  <Object><Key>notes/hello.txt</Key></Object>
</Delete>`
	req, _ = http.NewRequest("POST", ts.URL+"/"+bucket+"?delete", strings.NewReader(deletePayload))
	req.Header.Set("Content-Type", "application/xml")
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusOK)

	// Enable versioning.
	versioningPayload := `<?xml version="1.0" encoding="UTF-8"?>
<VersioningConfiguration xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <Status>Enabled</Status>
</VersioningConfiguration>`
	req, _ = http.NewRequest("PUT", ts.URL+"/"+bucket+"?versioning", strings.NewReader(versioningPayload))
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusOK)

	// Put object and expect version id.
	resp = signedRequest(t, "PUT", ts.URL+"/"+bucket+"/notes/versioned.txt", []byte("v1"), nil)
	assertStatus(t, resp, http.StatusOK)
	versionID = resp.Header.Get("x-amz-version-id")
	if versionID == "" {
		t.Fatalf("expected version id on put")
	}

	// Get by version id.
	req, _ = http.NewRequest("GET", ts.URL+"/"+bucket+"/notes/versioned.txt?versionId="+url.QueryEscape(versionID), nil)
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusOK)

	// Delete without version id should create delete marker.
	req, _ = http.NewRequest("DELETE", ts.URL+"/"+bucket+"/notes/versioned.txt", nil)
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusNoContent)
	if resp.Header.Get("x-amz-delete-marker") != "true" {
		t.Fatalf("expected delete marker header")
	}
	deleteMarkerID := resp.Header.Get("x-amz-version-id")
	if deleteMarkerID == "" {
		t.Fatalf("expected delete marker version id")
	}

	// GET/HEAD on delete marker should return 404 with delete marker headers.
	req, _ = http.NewRequest("GET", ts.URL+"/"+bucket+"/notes/versioned.txt", nil)
	resp = signedRequestWithRequest(t, req, nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for delete marker GET, got %d", resp.StatusCode)
	}
	if resp.Header.Get("x-amz-delete-marker") != "true" {
		t.Fatalf("expected delete marker header on GET")
	}
	if resp.Header.Get("x-amz-version-id") != deleteMarkerID {
		t.Fatalf("expected delete marker version id %q, got %q", deleteMarkerID, resp.Header.Get("x-amz-version-id"))
	}
	assertS3ErrorCode(t, resp, "NoSuchKey")

	req, _ = http.NewRequest("HEAD", ts.URL+"/"+bucket+"/notes/versioned.txt", nil)
	resp = signedRequestWithRequest(t, req, nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for delete marker HEAD, got %d", resp.StatusCode)
	}
	if resp.Header.Get("x-amz-delete-marker") != "true" {
		t.Fatalf("expected delete marker header on HEAD")
	}
	if resp.Header.Get("x-amz-version-id") != deleteMarkerID {
		t.Fatalf("expected delete marker version id %q, got %q", deleteMarkerID, resp.Header.Get("x-amz-version-id"))
	}

	req, _ = http.NewRequest("GET", ts.URL+"/"+bucket+"/notes/versioned.txt?versionId="+url.QueryEscape(deleteMarkerID), nil)
	resp = signedRequestWithRequest(t, req, nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for delete marker version GET, got %d", resp.StatusCode)
	}
	if resp.Header.Get("x-amz-delete-marker") != "true" {
		t.Fatalf("expected delete marker header on version GET")
	}
	if resp.Header.Get("x-amz-version-id") != deleteMarkerID {
		t.Fatalf("expected delete marker version id %q, got %q", deleteMarkerID, resp.Header.Get("x-amz-version-id"))
	}
	assertS3ErrorCode(t, resp, "NoSuchKey")

	req, _ = http.NewRequest("HEAD", ts.URL+"/"+bucket+"/notes/versioned.txt?versionId="+url.QueryEscape(deleteMarkerID), nil)
	resp = signedRequestWithRequest(t, req, nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for delete marker version HEAD, got %d", resp.StatusCode)
	}
	if resp.Header.Get("x-amz-delete-marker") != "true" {
		t.Fatalf("expected delete marker header on version HEAD")
	}
	if resp.Header.Get("x-amz-version-id") != deleteMarkerID {
		t.Fatalf("expected delete marker version id %q, got %q", deleteMarkerID, resp.Header.Get("x-amz-version-id"))
	}

	// List versions should include delete marker.
	resp = signedRequest(t, "GET", ts.URL+"/"+bucket+"?versions", nil, nil)
	assertStatus(t, resp, http.StatusOK)
	body = mustBody(t, resp)
	var versions s3ListObjectVersionsResult
	if err := xml.Unmarshal(body, &versions); err != nil {
		t.Fatalf("list versions parse: %v", err)
	}
	if len(versions.DeleteMarkers) == 0 {
		t.Fatalf("expected delete markers in list versions")
	}

	// Put ACL and get ACL.
	resp = signedRequest(t, "PUT", ts.URL+"/"+bucket+"/notes/hello.txt", []byte("hello-again"), nil)
	assertStatus(t, resp, http.StatusOK)
	req, _ = http.NewRequest("PUT", ts.URL+"/"+bucket+"/notes/hello.txt?acl", nil)
	req.Header.Set("x-amz-acl", "public-read")
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusOK)
	req, _ = http.NewRequest("GET", ts.URL+"/"+bucket+"/notes/hello.txt?acl", nil)
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusOK)

	// Public-read object should be accessible without auth.
	unsignedReq, _ := http.NewRequest("GET", ts.URL+"/"+bucket+"/notes/hello.txt", nil)
	unsignedResp, err := http.DefaultClient.Do(unsignedReq)
	if err != nil {
		t.Fatalf("unsigned get: %v", err)
	}
	assertStatus(t, unsignedResp, http.StatusOK)
	unsignedReq, _ = http.NewRequest("HEAD", ts.URL+"/"+bucket+"/notes/hello.txt", nil)
	unsignedResp, err = http.DefaultClient.Do(unsignedReq)
	if err != nil {
		t.Fatalf("unsigned head: %v", err)
	}
	assertStatus(t, unsignedResp, http.StatusOK)

	// Private object should be AccessDenied without auth.
	resp = signedRequest(t, "PUT", ts.URL+"/"+bucket+"/notes/private.txt", []byte("private"), nil)
	assertStatus(t, resp, http.StatusOK)
	unsignedReq, _ = http.NewRequest("GET", ts.URL+"/"+bucket+"/notes/private.txt", nil)
	unsignedResp, err = http.DefaultClient.Do(unsignedReq)
	if err != nil {
		t.Fatalf("unsigned private get: %v", err)
	}
	if unsignedResp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 for private object, got %d", unsignedResp.StatusCode)
	}
	assertS3ErrorCode(t, unsignedResp, "AccessDenied")

	// Private bucket list should be denied without auth.
	unsignedReq, _ = http.NewRequest("GET", ts.URL+"/"+bucket, nil)
	unsignedResp, err = http.DefaultClient.Do(unsignedReq)
	if err != nil {
		t.Fatalf("unsigned list bucket: %v", err)
	}
	if unsignedResp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 for private bucket list, got %d", unsignedResp.StatusCode)
	}
	assertS3ErrorCode(t, unsignedResp, "AccessDenied")

	// Public bucket list should be allowed without auth.
	req, _ = http.NewRequest("PUT", ts.URL+"/"+bucket+"?acl", nil)
	req.Header.Set("x-amz-acl", "public-read")
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusOK)
	unsignedReq, _ = http.NewRequest("GET", ts.URL+"/"+bucket, nil)
	unsignedResp, err = http.DefaultClient.Do(unsignedReq)
	if err != nil {
		t.Fatalf("unsigned list bucket public: %v", err)
	}
	assertStatus(t, unsignedResp, http.StatusOK)

	// Bucket encryption config.
	encryptionPayload := `<?xml version="1.0" encoding="UTF-8"?>
<ServerSideEncryptionConfiguration xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <Rule>
    <ApplyServerSideEncryptionByDefault>
      <SSEAlgorithm>AES256</SSEAlgorithm>
    </ApplyServerSideEncryptionByDefault>
  </Rule>
</ServerSideEncryptionConfiguration>`
	req, _ = http.NewRequest("PUT", ts.URL+"/"+bucket+"?encryption", strings.NewReader(encryptionPayload))
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusOK)
	req, _ = http.NewRequest("GET", ts.URL+"/"+bucket+"?encryption", nil)
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusOK)
	body = mustBody(t, resp)
	var encCfg s3BucketEncryptionConfiguration
	if err := xml.Unmarshal(body, &encCfg); err != nil {
		t.Fatalf("encryption parse: %v", err)
	}
	if len(encCfg.Rules) == 0 || encCfg.Rules[0].ApplyServerSideEncryptionByDefault.SSEAlgorithm != "AES256" {
		t.Fatalf("unexpected encryption config: %+v", encCfg)
	}
	req, _ = http.NewRequest("DELETE", ts.URL+"/"+bucket+"?encryption", nil)
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusNoContent)
	req, _ = http.NewRequest("GET", ts.URL+"/"+bucket+"?encryption", nil)
	resp = signedRequestWithRequest(t, req, nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for missing encryption, got %d", resp.StatusCode)
	}
	assertS3ErrorCode(t, resp, "ServerSideEncryptionConfigurationNotFoundError")

	// Bucket notifications.
	if _, err := srv.sqs.CreateQueue("events", nil); err != nil {
		t.Fatalf("create queue: %v", err)
	}
	notificationPayload := `<?xml version="1.0" encoding="UTF-8"?>
<NotificationConfiguration xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <QueueConfiguration>
    <Id>queue-1</Id>
    <Queue>arn:aws:sqs:us-east-1:123456789012:events</Queue>
    <Event>s3:ObjectCreated:Put</Event>
    <Event>s3:ObjectRemoved:Delete</Event>
    <Event>s3:ObjectRemoved:DeleteMarkerCreated</Event>
  </QueueConfiguration>
</NotificationConfiguration>`
	req, _ = http.NewRequest("PUT", ts.URL+"/"+bucket+"?notification", strings.NewReader(notificationPayload))
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusOK)
	req, _ = http.NewRequest("GET", ts.URL+"/"+bucket+"?notification", nil)
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusOK)
	body = mustBody(t, resp)
	var notifCfg s3NotificationConfiguration
	if err := xml.Unmarshal(body, &notifCfg); err != nil {
		t.Fatalf("notification parse: %v", err)
	}
	if len(notifCfg.QueueConfigurations) != 1 {
		t.Fatalf("expected queue configuration, got %+v", notifCfg)
	}

	resp = signedRequest(t, "PUT", ts.URL+"/"+bucket+"/notes/notify.txt", []byte("notify"), nil)
	assertStatus(t, resp, http.StatusOK)
	messages, err := srv.sqs.ReceiveMessages("events", 1)
	if err != nil {
		t.Fatalf("receive messages: %v", err)
	}
	if len(messages) == 0 {
		t.Fatalf("expected notification message")
	}
	var event struct {
		Records []struct {
			EventName string `json:"eventName"`
			S3        struct {
				Object struct {
					Key string `json:"key"`
				} `json:"object"`
			} `json:"s3"`
		} `json:"Records"`
	}
	if err := json.Unmarshal([]byte(messages[0].Body), &event); err != nil {
		t.Fatalf("notification JSON parse: %v", err)
	}
	if len(event.Records) == 0 || event.Records[0].EventName != "ObjectCreated:Put" {
		t.Fatalf("unexpected notification event: %+v", event)
	}
	if event.Records[0].S3.Object.Key != "notes/notify.txt" {
		t.Fatalf("unexpected notification key: %+v", event)
	}
	resp = signedRequest(t, "DELETE", ts.URL+"/"+bucket+"/notes/notify.txt", nil, nil)
	assertStatus(t, resp, http.StatusNoContent)
	messages, err = srv.sqs.ReceiveMessages("events", 1)
	if err != nil {
		t.Fatalf("receive delete messages: %v", err)
	}
	if len(messages) == 0 {
		t.Fatalf("expected delete notification message")
	}
	if err := json.Unmarshal([]byte(messages[0].Body), &event); err != nil {
		t.Fatalf("notification JSON parse: %v", err)
	}
	if len(event.Records) == 0 || (event.Records[0].EventName != "ObjectRemoved:Delete" && event.Records[0].EventName != "ObjectRemoved:DeleteMarkerCreated") {
		t.Fatalf("unexpected delete notification event: %+v", event)
	}

	// Analytics configuration.
	analyticsPayload := `<?xml version="1.0" encoding="UTF-8"?>
<AnalyticsConfiguration xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <Id>analytics-1</Id>
  <Filter>
    <And>
      <Prefix>logs/</Prefix>
      <Tag>
        <Key>env</Key>
        <Value>prod</Value>
      </Tag>
    </And>
  </Filter>
  <StorageClassAnalysis/>
</AnalyticsConfiguration>`
	req, _ = http.NewRequest("PUT", ts.URL+"/"+bucket+"?analytics", strings.NewReader(analyticsPayload))
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusOK)
	analyticsPayload2 := strings.Replace(analyticsPayload, "<Id>analytics-1</Id>", "<Id>analytics-2</Id>", 1)
	req, _ = http.NewRequest("PUT", ts.URL+"/"+bucket+"?analytics", strings.NewReader(analyticsPayload2))
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusOK)
	req, _ = http.NewRequest("GET", ts.URL+"/"+bucket+"?analytics&id=analytics-1", nil)
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusOK)
	req, _ = http.NewRequest("GET", ts.URL+"/"+bucket+"?analytics", nil)
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusOK)
	body = mustBody(t, resp)
	assertXMLRoot(t, body, "ListBucketAnalyticsConfigurationResult")
	assertXMLOrder(t, body, []string{"<IsTruncated>", "<AnalyticsConfiguration>"})
	req, _ = http.NewRequest("GET", ts.URL+"/"+bucket+"?analytics&max-keys=1", nil)
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusOK)
	body = mustBody(t, resp)
	assertXMLOrder(t, body, []string{"<IsTruncated>", "<NextContinuationToken>", "<AnalyticsConfiguration>"})
	if strings.Count(string(body), "xmlns=") != 1 {
		t.Fatalf("expected single xmlns on analytics list response")
	}
	req, _ = http.NewRequest("GET", ts.URL+"/"+bucket+"?analytics&max-keys=1&continuation-token=analytics-1", nil)
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusOK)
	body = mustBody(t, resp)
	assertXMLOrder(t, body, []string{"<IsTruncated>", "<ContinuationToken>", "<AnalyticsConfiguration>"})
	req, _ = http.NewRequest("DELETE", ts.URL+"/"+bucket+"?analytics&id=analytics-1", nil)
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusNoContent)
	req, _ = http.NewRequest("DELETE", ts.URL+"/"+bucket+"?analytics&id=analytics-2", nil)
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusNoContent)
	req, _ = http.NewRequest("GET", ts.URL+"/"+bucket+"?analytics&id=analytics-1", nil)
	resp = signedRequestWithRequest(t, req, nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for analytics config, got %d", resp.StatusCode)
	}
	assertS3ErrorCode(t, resp, "NoSuchConfiguration")

	// Metrics configuration.
	metricsPayload := `<?xml version="1.0" encoding="UTF-8"?>
<MetricsConfiguration xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <Id>metrics-1</Id>
  <Filter>
    <Tag>
      <Key>team</Key>
      <Value>stackyard</Value>
    </Tag>
  </Filter>
</MetricsConfiguration>`
	req, _ = http.NewRequest("PUT", ts.URL+"/"+bucket+"?metrics", strings.NewReader(metricsPayload))
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusOK)
	metricsPayload2 := strings.Replace(metricsPayload, "<Id>metrics-1</Id>", "<Id>metrics-2</Id>", 1)
	req, _ = http.NewRequest("PUT", ts.URL+"/"+bucket+"?metrics", strings.NewReader(metricsPayload2))
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusOK)
	req, _ = http.NewRequest("GET", ts.URL+"/"+bucket+"?metrics&id=metrics-1", nil)
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusOK)
	req, _ = http.NewRequest("GET", ts.URL+"/"+bucket+"?metrics", nil)
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusOK)
	body = mustBody(t, resp)
	assertXMLRoot(t, body, "ListMetricsConfigurationsResult")
	assertXMLOrder(t, body, []string{"<IsTruncated>", "<MetricsConfiguration>"})
	if strings.Count(string(body), "xmlns=") != 1 {
		t.Fatalf("expected single xmlns on metrics list response")
	}
	req, _ = http.NewRequest("GET", ts.URL+"/"+bucket+"?metrics&max-keys=1", nil)
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusOK)
	body = mustBody(t, resp)
	assertXMLOrder(t, body, []string{"<IsTruncated>", "<NextContinuationToken>", "<MetricsConfiguration>"})
	req, _ = http.NewRequest("DELETE", ts.URL+"/"+bucket+"?metrics&id=metrics-1", nil)
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusNoContent)
	req, _ = http.NewRequest("DELETE", ts.URL+"/"+bucket+"?metrics&id=metrics-2", nil)
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusNoContent)
	req, _ = http.NewRequest("GET", ts.URL+"/"+bucket+"?metrics&id=metrics-1", nil)
	resp = signedRequestWithRequest(t, req, nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for metrics config, got %d", resp.StatusCode)
	}
	assertS3ErrorCode(t, resp, "NoSuchConfiguration")

	// Inventory configuration.
	inventoryPayload := `<?xml version="1.0" encoding="UTF-8"?>
<InventoryConfiguration xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <Id>inventory-1</Id>
  <IsEnabled>true</IsEnabled>
  <IncludedObjectVersions>All</IncludedObjectVersions>
  <Schedule>
    <Frequency>Daily</Frequency>
  </Schedule>
  <Filter>
    <Prefix>logs/</Prefix>
  </Filter>
  <Destination>
    <S3BucketDestination>
      <Bucket>arn:aws:s3:::demo</Bucket>
      <Format>CSV</Format>
    </S3BucketDestination>
  </Destination>
</InventoryConfiguration>`
	req, _ = http.NewRequest("PUT", ts.URL+"/"+bucket+"?inventory", strings.NewReader(inventoryPayload))
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusOK)
	inventoryPayload2 := strings.Replace(inventoryPayload, "<Id>inventory-1</Id>", "<Id>inventory-2</Id>", 1)
	req, _ = http.NewRequest("PUT", ts.URL+"/"+bucket+"?inventory", strings.NewReader(inventoryPayload2))
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusOK)
	req, _ = http.NewRequest("GET", ts.URL+"/"+bucket+"?inventory&id=inventory-1", nil)
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusOK)
	req, _ = http.NewRequest("GET", ts.URL+"/"+bucket+"?inventory", nil)
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusOK)
	body = mustBody(t, resp)
	assertXMLRoot(t, body, "ListInventoryConfigurationsResult")
	assertXMLOrder(t, body, []string{"<InventoryConfiguration>", "<IsTruncated>"})
	if strings.Count(string(body), "xmlns=") != 1 {
		t.Fatalf("expected single xmlns on inventory list response")
	}
	req, _ = http.NewRequest("GET", ts.URL+"/"+bucket+"?inventory&max-keys=1", nil)
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusOK)
	body = mustBody(t, resp)
	assertXMLOrder(t, body, []string{"<InventoryConfiguration>", "<IsTruncated>", "<NextContinuationToken>"})
	req, _ = http.NewRequest("GET", ts.URL+"/"+bucket+"?inventory&max-keys=1&continuation-token=inventory-1", nil)
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusOK)
	body = mustBody(t, resp)
	assertXMLOrder(t, body, []string{"<ContinuationToken>", "<InventoryConfiguration>", "<IsTruncated>"})
	req, _ = http.NewRequest("DELETE", ts.URL+"/"+bucket+"?inventory&id=inventory-1", nil)
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusNoContent)
	req, _ = http.NewRequest("DELETE", ts.URL+"/"+bucket+"?inventory&id=inventory-2", nil)
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusNoContent)
	req, _ = http.NewRequest("GET", ts.URL+"/"+bucket+"?inventory&id=inventory-1", nil)
	resp = signedRequestWithRequest(t, req, nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for inventory config, got %d", resp.StatusCode)
	}
	assertS3ErrorCode(t, resp, "NoSuchConfiguration")

	// Intelligent tiering configuration.
	itPayload := `<?xml version="1.0" encoding="UTF-8"?>
<IntelligentTieringConfiguration xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <Id>tiering-1</Id>
  <Filter>
    <Tag>
      <Key>class</Key>
      <Value>cold</Value>
    </Tag>
  </Filter>
  <Status>Enabled</Status>
  <Tiering>
    <AccessTier>ARCHIVE_ACCESS</AccessTier>
    <Days>30</Days>
  </Tiering>
</IntelligentTieringConfiguration>`
	req, _ = http.NewRequest("PUT", ts.URL+"/"+bucket+"?intelligent-tiering", strings.NewReader(itPayload))
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusOK)
	itPayload2 := strings.Replace(itPayload, "<Id>tiering-1</Id>", "<Id>tiering-2</Id>", 1)
	req, _ = http.NewRequest("PUT", ts.URL+"/"+bucket+"?intelligent-tiering", strings.NewReader(itPayload2))
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusOK)
	req, _ = http.NewRequest("GET", ts.URL+"/"+bucket+"?intelligent-tiering&id=tiering-1", nil)
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusOK)
	req, _ = http.NewRequest("GET", ts.URL+"/"+bucket+"?intelligent-tiering", nil)
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusOK)
	body = mustBody(t, resp)
	assertXMLRoot(t, body, "ListBucketIntelligentTieringConfigurationsOutput")
	assertXMLOrder(t, body, []string{"<IsTruncated>", "<IntelligentTieringConfiguration>"})
	if strings.Count(string(body), "xmlns=") != 1 {
		t.Fatalf("expected single xmlns on intelligent tiering list response")
	}
	req, _ = http.NewRequest("GET", ts.URL+"/"+bucket+"?intelligent-tiering&max-keys=1", nil)
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusOK)
	body = mustBody(t, resp)
	assertXMLOrder(t, body, []string{"<IsTruncated>", "<NextContinuationToken>", "<IntelligentTieringConfiguration>"})
	req, _ = http.NewRequest("DELETE", ts.URL+"/"+bucket+"?intelligent-tiering&id=tiering-1", nil)
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusNoContent)
	req, _ = http.NewRequest("DELETE", ts.URL+"/"+bucket+"?intelligent-tiering&id=tiering-2", nil)
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusNoContent)
	req, _ = http.NewRequest("GET", ts.URL+"/"+bucket+"?intelligent-tiering&id=tiering-1", nil)
	resp = signedRequestWithRequest(t, req, nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for intelligent tiering config, got %d", resp.StatusCode)
	}
	assertS3ErrorCode(t, resp, "NoSuchConfiguration")

	// Stage 6: RestoreObject / GetObjectTorrent / RenameObject / CreateSession / Metadata configuration / ListDirectoryBuckets / WriteGetObjectResponse.
	restorePayload := `<?xml version="1.0" encoding="UTF-8"?>
<RestoreRequest xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <Days>1</Days>
</RestoreRequest>`
	restoreHeaders := map[string]string{"x-amz-storage-class": "GLACIER"}
	resp = signedRequest(t, "PUT", ts.URL+"/"+bucket+"/notes/glacier.txt", []byte("cold"), restoreHeaders)
	assertStatus(t, resp, http.StatusOK)
	req, _ = http.NewRequest("POST", ts.URL+"/"+bucket+"/notes/glacier.txt?restore", strings.NewReader(restorePayload))
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusAccepted)
	req, _ = http.NewRequest("HEAD", ts.URL+"/"+bucket+"/notes/glacier.txt", nil)
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusOK)
	if resp.Header.Get("x-amz-restore") == "" {
		t.Fatalf("expected x-amz-restore header on restored object")
	}
	req, _ = http.NewRequest("GET", ts.URL+"/"+bucket+"/notes/glacier.txt?torrent", nil)
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusOK)
	if ct := resp.Header.Get("Content-Type"); ct != "application/x-bittorrent" {
		t.Fatalf("expected torrent content type, got %q", ct)
	}

	dirBucket := "dir--usw2-az1--x-s3"
	resp = signedRequest(t, "PUT", ts.URL+"/"+dirBucket, nil, nil)
	assertStatus(t, resp, http.StatusOK)
	resp = signedRequest(t, "PUT", ts.URL+"/"+dirBucket+"/notes/rename-src.txt", []byte("rename"), nil)
	assertStatus(t, resp, http.StatusOK)
	req, _ = http.NewRequest("PUT", ts.URL+"/"+dirBucket+"/notes/rename-dst.txt?renameObject", nil)
	req.Header.Set("x-amz-rename-source", "/"+dirBucket+"/notes/rename-src.txt")
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusOK)
	resp = signedRequest(t, "GET", ts.URL+"/"+dirBucket+"/notes/rename-dst.txt", nil, nil)
	assertStatus(t, resp, http.StatusOK)
	resp = signedRequest(t, "GET", ts.URL+"/"+dirBucket+"/notes/rename-src.txt", nil, nil)
	assertS3ErrorCode(t, resp, "NoSuchKey")

	req, _ = http.NewRequest("GET", ts.URL+"/"+dirBucket+"?session", nil)
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusOK)
	body = mustBody(t, resp)
	var sessionOut struct {
		XMLName     xml.Name `xml:"CreateSessionResult"`
		Credentials struct {
			AccessKeyID     string `xml:"AccessKeyId"`
			SecretAccessKey string `xml:"SecretAccessKey"`
			SessionToken    string `xml:"SessionToken"`
			Expiration      string `xml:"Expiration"`
		} `xml:"Credentials"`
	}
	if err := xml.Unmarshal(body, &sessionOut); err != nil {
		t.Fatalf("parse create session: %v", err)
	}
	if sessionOut.Credentials.AccessKeyID == "" || sessionOut.Credentials.SessionToken == "" {
		t.Fatalf("expected session credentials")
	}

	metaBucket := "meta-bucket"
	resp = signedRequest(t, "PUT", ts.URL+"/"+metaBucket, nil, nil)
	assertStatus(t, resp, http.StatusOK)
	metaPayload := `<?xml version="1.0" encoding="UTF-8"?>
<MetadataConfiguration xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <JournalTableConfiguration>
    <RecordExpiration>
      <Days>7</Days>
    </RecordExpiration>
  </JournalTableConfiguration>
  <InventoryTableConfiguration>
    <ConfigurationState>ENABLED</ConfigurationState>
  </InventoryTableConfiguration>
</MetadataConfiguration>`
	req, _ = http.NewRequest("POST", ts.URL+"/"+metaBucket+"?metadataConfiguration", strings.NewReader(metaPayload))
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusOK)
	req, _ = http.NewRequest("GET", ts.URL+"/"+metaBucket+"?metadataConfiguration", nil)
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusOK)
	body = mustBody(t, resp)
	assertXMLRoot(t, body, "GetBucketMetadataConfigurationResult")
	if !strings.Contains(string(body), "<InventoryTableConfigurationResult>") {
		t.Fatalf("expected inventory table configuration result")
	}
	updateInventory := `<?xml version="1.0" encoding="UTF-8"?>
<InventoryTableConfiguration xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <ConfigurationState>DISABLED</ConfigurationState>
</InventoryTableConfiguration>`
	req, _ = http.NewRequest("PUT", ts.URL+"/"+metaBucket+"?metadataInventoryTable", strings.NewReader(updateInventory))
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusOK)
	updateJournal := `<?xml version="1.0" encoding="UTF-8"?>
<JournalTableConfiguration xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <RecordExpiration>
    <Days>10</Days>
  </RecordExpiration>
</JournalTableConfiguration>`
	req, _ = http.NewRequest("PUT", ts.URL+"/"+metaBucket+"?metadataJournalTable", strings.NewReader(updateJournal))
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusOK)
	req, _ = http.NewRequest("GET", ts.URL+"/"+metaBucket+"?metadataConfiguration", nil)
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusOK)
	body = mustBody(t, resp)
	if !strings.Contains(string(body), "<ConfigurationState>DISABLED</ConfigurationState>") {
		t.Fatalf("expected inventory configuration state update")
	}
	if !strings.Contains(string(body), "<Days>10</Days>") {
		t.Fatalf("expected journal record expiration update")
	}
	req, _ = http.NewRequest("DELETE", ts.URL+"/"+metaBucket+"?metadataConfiguration", nil)
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusNoContent)
	req, _ = http.NewRequest("GET", ts.URL+"/"+metaBucket+"?metadataConfiguration", nil)
	resp = signedRequestWithRequest(t, req, nil)
	assertS3ErrorCode(t, resp, "NoSuchConfiguration")

	metaTablePayload := `<?xml version="1.0" encoding="UTF-8"?>
<MetadataTableConfiguration xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <S3TablesDestination>
    <TableBucketArn>arn:aws:s3:::meta-bucket-metadata</TableBucketArn>
    <TableName>meta-table</TableName>
  </S3TablesDestination>
</MetadataTableConfiguration>`
	req, _ = http.NewRequest("POST", ts.URL+"/"+metaBucket+"?metadataTable", strings.NewReader(metaTablePayload))
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusOK)
	req, _ = http.NewRequest("GET", ts.URL+"/"+metaBucket+"?metadataTable", nil)
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusOK)
	body = mustBody(t, resp)
	assertXMLRoot(t, body, "GetBucketMetadataTableConfigurationResult")
	req, _ = http.NewRequest("DELETE", ts.URL+"/"+metaBucket+"?metadataTable", nil)
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusNoContent)
	req, _ = http.NewRequest("GET", ts.URL+"/"+metaBucket+"?metadataTable", nil)
	resp = signedRequestWithRequest(t, req, nil)
	assertS3ErrorCode(t, resp, "NoSuchConfiguration")

	resp = signedRequest(t, "PUT", ts.URL+"/dir2--usw2-az1--x-s3", nil, nil)
	assertStatus(t, resp, http.StatusOK)
	req, _ = http.NewRequest("GET", ts.URL+"/?max-directory-buckets=1", nil)
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusOK)
	body = mustBody(t, resp)
	var dirList struct {
		XMLName           xml.Name `xml:"ListAllMyDirectoryBucketsResult"`
		ContinuationToken string   `xml:"ContinuationToken"`
		Buckets           struct {
			Buckets []struct {
				Name string `xml:"Name"`
			} `xml:"Bucket"`
		} `xml:"Buckets"`
	}
	if err := xml.Unmarshal(body, &dirList); err != nil {
		t.Fatalf("parse directory buckets: %v", err)
	}
	if len(dirList.Buckets.Buckets) != 1 {
		t.Fatalf("expected 1 directory bucket, got %d", len(dirList.Buckets.Buckets))
	}
	if !strings.Contains(dirList.Buckets.Buckets[0].Name, "--x-s3") {
		t.Fatalf("expected directory bucket name, got %q", dirList.Buckets.Buckets[0].Name)
	}
	if dirList.ContinuationToken == "" {
		t.Fatalf("expected continuation token for directory buckets")
	}

	req, _ = http.NewRequest("POST", ts.URL+"/WriteGetObjectResponse", strings.NewReader("ok"))
	req.Header.Set("x-amz-request-route", "route")
	req.Header.Set("x-amz-request-token", "token")
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusOK)

	// Bucket notifications with prefix/suffix filters.
	if _, err := srv.sqs.CreateQueue("events-filter", nil); err != nil {
		t.Fatalf("create filter queue: %v", err)
	}
	filterPayload := `<?xml version="1.0" encoding="UTF-8"?>
<NotificationConfiguration xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <QueueConfiguration>
    <Id>queue-filter</Id>
    <Queue>arn:aws:sqs:us-east-1:123456789012:events-filter</Queue>
    <Event>s3:ObjectCreated:Put</Event>
    <Filter>
      <S3Key>
        <FilterRule>
          <Name>prefix</Name>
          <Value>notes/</Value>
        </FilterRule>
        <FilterRule>
          <Name>suffix</Name>
          <Value>.txt</Value>
        </FilterRule>
      </S3Key>
    </Filter>
  </QueueConfiguration>
</NotificationConfiguration>`
	req, _ = http.NewRequest("PUT", ts.URL+"/"+bucket+"?notification", strings.NewReader(filterPayload))
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusOK)

	resp = signedRequest(t, "PUT", ts.URL+"/"+bucket+"/notes/filtered.txt", []byte("match"), nil)
	assertStatus(t, resp, http.StatusOK)
	resp = signedRequest(t, "PUT", ts.URL+"/"+bucket+"/notes/filtered.bin", []byte("nope"), nil)
	assertStatus(t, resp, http.StatusOK)
	resp = signedRequest(t, "PUT", ts.URL+"/"+bucket+"/other/filtered.txt", []byte("nope"), nil)
	assertStatus(t, resp, http.StatusOK)

	filterMessages, err := srv.sqs.ReceiveMessages("events-filter", 10)
	if err != nil {
		t.Fatalf("receive filter messages: %v", err)
	}
	if len(filterMessages) != 1 {
		t.Fatalf("expected 1 filtered notification, got %d", len(filterMessages))
	}
	if err := json.Unmarshal([]byte(filterMessages[0].Body), &event); err != nil {
		t.Fatalf("filter notification JSON parse: %v", err)
	}
	if len(event.Records) == 0 || event.Records[0].S3.Object.Key != "notes/filtered.txt" {
		t.Fatalf("unexpected filtered notification: %+v", event)
	}
	filterMessages, err = srv.sqs.ReceiveMessages("events-filter", 1)
	if err != nil {
		t.Fatalf("receive filter messages empty: %v", err)
	}
	if len(filterMessages) != 0 {
		t.Fatalf("expected no additional filtered notifications")
	}

	// Multiple queues with distinct filters.
	if _, err := srv.sqs.CreateQueue("events-notes", nil); err != nil {
		t.Fatalf("create notes queue: %v", err)
	}
	if _, err := srv.sqs.CreateQueue("events-reports", nil); err != nil {
		t.Fatalf("create reports queue: %v", err)
	}
	multiQueuePayload := `<?xml version="1.0" encoding="UTF-8"?>
<NotificationConfiguration xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <QueueConfiguration>
    <Id>queue-notes</Id>
    <Queue>arn:aws:sqs:us-east-1:123456789012:events-notes</Queue>
    <Event>s3:ObjectCreated:Put</Event>
    <Filter>
      <S3Key>
        <FilterRule><Name>prefix</Name><Value>notes/</Value></FilterRule>
      </S3Key>
    </Filter>
  </QueueConfiguration>
  <QueueConfiguration>
    <Id>queue-reports</Id>
    <Queue>arn:aws:sqs:us-east-1:123456789012:events-reports</Queue>
    <Event>s3:ObjectCreated:Put</Event>
    <Filter>
      <S3Key>
        <FilterRule><Name>prefix</Name><Value>reports/</Value></FilterRule>
      </S3Key>
    </Filter>
  </QueueConfiguration>
</NotificationConfiguration>`
	req, _ = http.NewRequest("PUT", ts.URL+"/"+bucket+"?notification", strings.NewReader(multiQueuePayload))
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusOK)

	resp = signedRequest(t, "PUT", ts.URL+"/"+bucket+"/notes/mq.txt", []byte("notes"), nil)
	assertStatus(t, resp, http.StatusOK)
	resp = signedRequest(t, "PUT", ts.URL+"/"+bucket+"/reports/r1.txt", []byte("reports"), nil)
	assertStatus(t, resp, http.StatusOK)

	notesMsgs, err := srv.sqs.ReceiveMessages("events-notes", 10)
	if err != nil {
		t.Fatalf("receive notes messages: %v", err)
	}
	if len(notesMsgs) != 1 {
		t.Fatalf("expected 1 notes message, got %d", len(notesMsgs))
	}
	if err := json.Unmarshal([]byte(notesMsgs[0].Body), &event); err != nil {
		t.Fatalf("notes message parse: %v", err)
	}
	if event.Records[0].S3.Object.Key != "notes/mq.txt" {
		t.Fatalf("unexpected notes message key: %+v", event)
	}
	reportsMsgs, err := srv.sqs.ReceiveMessages("events-reports", 10)
	if err != nil {
		t.Fatalf("receive reports messages: %v", err)
	}
	if len(reportsMsgs) != 1 {
		t.Fatalf("expected 1 reports message, got %d", len(reportsMsgs))
	}
	if err := json.Unmarshal([]byte(reportsMsgs[0].Body), &event); err != nil {
		t.Fatalf("reports message parse: %v", err)
	}
	if event.Records[0].S3.Object.Key != "reports/r1.txt" {
		t.Fatalf("unexpected reports message key: %+v", event)
	}

	// Overlapping filters should deliver to both queues.
	if _, err := srv.sqs.CreateQueue("events-overlap-1", nil); err != nil {
		t.Fatalf("create overlap queue 1: %v", err)
	}
	if _, err := srv.sqs.CreateQueue("events-overlap-2", nil); err != nil {
		t.Fatalf("create overlap queue 2: %v", err)
	}
	overlapPayload := `<?xml version="1.0" encoding="UTF-8"?>
<NotificationConfiguration xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <QueueConfiguration>
    <Id>queue-overlap-1</Id>
    <Queue>arn:aws:sqs:us-east-1:123456789012:events-overlap-1</Queue>
    <Event>s3:ObjectCreated:Put</Event>
    <Filter>
      <S3Key>
        <FilterRule><Name>prefix</Name><Value>overlap/</Value></FilterRule>
      </S3Key>
    </Filter>
  </QueueConfiguration>
  <QueueConfiguration>
    <Id>queue-overlap-2</Id>
    <Queue>arn:aws:sqs:us-east-1:123456789012:events-overlap-2</Queue>
    <Event>s3:ObjectCreated:Put</Event>
    <Filter>
      <S3Key>
        <FilterRule><Name>prefix</Name><Value>overlap/</Value></FilterRule>
        <FilterRule><Name>suffix</Name><Value>.txt</Value></FilterRule>
      </S3Key>
    </Filter>
  </QueueConfiguration>
</NotificationConfiguration>`
	req, _ = http.NewRequest("PUT", ts.URL+"/"+bucket+"?notification", strings.NewReader(overlapPayload))
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusOK)

	resp = signedRequest(t, "PUT", ts.URL+"/"+bucket+"/overlap/item.txt", []byte("overlap"), nil)
	assertStatus(t, resp, http.StatusOK)

	overlap1, err := srv.sqs.ReceiveMessages("events-overlap-1", 10)
	if err != nil {
		t.Fatalf("receive overlap-1: %v", err)
	}
	if len(overlap1) != 1 {
		t.Fatalf("expected 1 overlap-1 message, got %d", len(overlap1))
	}
	overlap2, err := srv.sqs.ReceiveMessages("events-overlap-2", 10)
	if err != nil {
		t.Fatalf("receive overlap-2: %v", err)
	}
	if len(overlap2) != 1 {
		t.Fatalf("expected 1 overlap-2 message, got %d", len(overlap2))
	}

	// Invalid filter rules.
	invalidPayload := `<?xml version="1.0" encoding="UTF-8"?>
<NotificationConfiguration xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <QueueConfiguration>
    <Id>queue-invalid</Id>
    <Queue>arn:aws:sqs:us-east-1:123456789012:events</Queue>
    <Event>s3:ObjectCreated:Put</Event>
    <Filter>
      <S3Key>
        <FilterRule><Name>invalid</Name><Value>x</Value></FilterRule>
      </S3Key>
    </Filter>
  </QueueConfiguration>
</NotificationConfiguration>`
	req, _ = http.NewRequest("PUT", ts.URL+"/"+bucket+"?notification", strings.NewReader(invalidPayload))
	resp = signedRequestWithRequest(t, req, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid filter rule, got %d", resp.StatusCode)
	}
	assertS3ErrorCode(t, resp, "InvalidArgument")

	duplicatePayload := `<?xml version="1.0" encoding="UTF-8"?>
<NotificationConfiguration xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <QueueConfiguration>
    <Id>queue-dup</Id>
    <Queue>arn:aws:sqs:us-east-1:123456789012:events</Queue>
    <Event>s3:ObjectCreated:Put</Event>
    <Filter>
      <S3Key>
        <FilterRule><Name>prefix</Name><Value>a/</Value></FilterRule>
        <FilterRule><Name>prefix</Name><Value>b/</Value></FilterRule>
      </S3Key>
    </Filter>
  </QueueConfiguration>
</NotificationConfiguration>`
	req, _ = http.NewRequest("PUT", ts.URL+"/"+bucket+"?notification", strings.NewReader(duplicatePayload))
	resp = signedRequestWithRequest(t, req, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for duplicate prefix filters, got %d", resp.StatusCode)
	}
	assertS3ErrorCode(t, resp, "InvalidArgument")

	// Object tagging.
	objectTagPayload := `<?xml version="1.0" encoding="UTF-8"?>
<Tagging xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <TagSet>
    <Tag>
      <Key>team</Key>
      <Value>stackyard</Value>
    </Tag>
  </TagSet>
</Tagging>`
	req, _ = http.NewRequest("PUT", ts.URL+"/"+bucket+"/notes/hello.txt?tagging", strings.NewReader(objectTagPayload))
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusOK)
	req, _ = http.NewRequest("GET", ts.URL+"/"+bucket+"/notes/hello.txt?tagging", nil)
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusOK)
	req, _ = http.NewRequest("DELETE", ts.URL+"/"+bucket+"/notes/hello.txt?tagging", nil)
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusNoContent)

	// Object attributes.
	req, _ = http.NewRequest("GET", ts.URL+"/"+bucket+"/notes/hello.txt?attributes", nil)
	req.Header.Set("x-amz-object-attributes", "ETag,ObjectSize,StorageClass,LastModified,VersionId")
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusOK)
	body = mustBody(t, resp)
	var attrs s3ObjectAttributesResponse
	if err := xml.Unmarshal(body, &attrs); err != nil {
		t.Fatalf("object attributes parse: %v", err)
	}
	if attrs.ETag == "" || attrs.ObjectSize == nil || attrs.StorageClass == "" || attrs.LastModified == "" {
		t.Fatalf("unexpected attributes response: %+v", attrs)
	}

	// Bucket object lock configuration.
	lockPayload := `<?xml version="1.0" encoding="UTF-8"?>
<ObjectLockConfiguration xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <ObjectLockEnabled>Enabled</ObjectLockEnabled>
  <Rule>
    <DefaultRetention>
      <Mode>GOVERNANCE</Mode>
      <Days>1</Days>
    </DefaultRetention>
  </Rule>
</ObjectLockConfiguration>`
	req, _ = http.NewRequest("PUT", ts.URL+"/"+bucket+"?object-lock", strings.NewReader(lockPayload))
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusOK)
	req, _ = http.NewRequest("GET", ts.URL+"/"+bucket+"?object-lock", nil)
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusOK)

	// Object retention.
	retainUntil := time.Now().UTC().Add(24 * time.Hour).Format(time.RFC3339)
	retentionPayload := `<?xml version="1.0" encoding="UTF-8"?>
<Retention xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <Mode>GOVERNANCE</Mode>
  <RetainUntilDate>` + retainUntil + `</RetainUntilDate>
</Retention>`
	req, _ = http.NewRequest("PUT", ts.URL+"/"+bucket+"/notes/hello.txt?retention", strings.NewReader(retentionPayload))
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusOK)
	req, _ = http.NewRequest("GET", ts.URL+"/"+bucket+"/notes/hello.txt?retention", nil)
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusOK)

	// Retention with past date should fail.
	pastRetain := time.Now().UTC().Add(-1 * time.Hour).Format(time.RFC3339)
	pastPayload := `<?xml version="1.0" encoding="UTF-8"?>
<Retention xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <Mode>GOVERNANCE</Mode>
  <RetainUntilDate>` + pastRetain + `</RetainUntilDate>
</Retention>`
	req, _ = http.NewRequest("PUT", ts.URL+"/"+bucket+"/notes/hello.txt?retention", strings.NewReader(pastPayload))
	resp = signedRequestWithRequest(t, req, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for past retention date, got %d", resp.StatusCode)
	}
	assertS3ErrorCode(t, resp, "InvalidRequest")

	// Object legal hold.
	legalHoldPayload := `<?xml version="1.0" encoding="UTF-8"?>
<LegalHold xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <Status>ON</Status>
</LegalHold>`
	req, _ = http.NewRequest("PUT", ts.URL+"/"+bucket+"/notes/hello.txt?legal-hold", strings.NewReader(legalHoldPayload))
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusOK)
	req, _ = http.NewRequest("GET", ts.URL+"/"+bucket+"/notes/hello.txt?legal-hold", nil)
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusOK)

	invalidHold := `<?xml version="1.0" encoding="UTF-8"?>
<LegalHold xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <Status>MAYBE</Status>
</LegalHold>`
	req, _ = http.NewRequest("PUT", ts.URL+"/"+bucket+"/notes/hello.txt?legal-hold", strings.NewReader(invalidHold))
	resp = signedRequestWithRequest(t, req, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid legal hold status, got %d", resp.StatusCode)
	}
	assertS3ErrorCode(t, resp, "InvalidRequest")

	// Legal hold blocks delete.
	signedRequest(t, "PUT", ts.URL+"/"+bucket+"/notes/hold.txt", []byte("hold"), nil)
	req, _ = http.NewRequest("PUT", ts.URL+"/"+bucket+"/notes/hold.txt?legal-hold", strings.NewReader(legalHoldPayload))
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusOK)
	req, _ = http.NewRequest("DELETE", ts.URL+"/"+bucket+"/notes/hold.txt", nil)
	resp = signedRequestWithRequest(t, req, nil)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 for legal hold delete, got %d", resp.StatusCode)
	}
	assertS3ErrorCode(t, resp, "AccessDenied")

	// Governance retention blocks delete unless bypassed.
	signedRequest(t, "PUT", ts.URL+"/"+bucket+"/notes/locked.txt", []byte("locked"), nil)
	req, _ = http.NewRequest("PUT", ts.URL+"/"+bucket+"/notes/locked.txt?retention", strings.NewReader(retentionPayload))
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusOK)
	req, _ = http.NewRequest("DELETE", ts.URL+"/"+bucket+"/notes/locked.txt", nil)
	resp = signedRequestWithRequest(t, req, nil)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 for governance retention delete, got %d", resp.StatusCode)
	}
	assertS3ErrorCode(t, resp, "AccessDenied")
	req, _ = http.NewRequest("DELETE", ts.URL+"/"+bucket+"/notes/locked.txt?x-amz-bypass-governance-retention=true", nil)
	req.Header.Set("x-amz-bypass-governance-retention", "true")
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusNoContent)

	// Compliance retention never bypasses.
	compliancePayload := `<?xml version="1.0" encoding="UTF-8"?>
<Retention xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <Mode>COMPLIANCE</Mode>
  <RetainUntilDate>` + retainUntil + `</RetainUntilDate>
</Retention>`
	signedRequest(t, "PUT", ts.URL+"/"+bucket+"/notes/compliance.txt", []byte("comp"), nil)
	req, _ = http.NewRequest("PUT", ts.URL+"/"+bucket+"/notes/compliance.txt?retention", strings.NewReader(compliancePayload))
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusOK)
	req, _ = http.NewRequest("DELETE", ts.URL+"/"+bucket+"/notes/compliance.txt", nil)
	req.Header.Set("x-amz-bypass-governance-retention", "true")
	resp = signedRequestWithRequest(t, req, nil)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 for compliance retention delete, got %d", resp.StatusCode)
	}
	assertS3ErrorCode(t, resp, "AccessDenied")

	// CORS configuration.
	corsPayload := `<?xml version="1.0" encoding="UTF-8"?>
<CORSConfiguration xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <CORSRule>
    <AllowedOrigin>http://example.com</AllowedOrigin>
    <AllowedMethod>GET</AllowedMethod>
    <AllowedHeader>*</AllowedHeader>
    <MaxAgeSeconds>300</MaxAgeSeconds>
  </CORSRule>
</CORSConfiguration>`
	req, _ = http.NewRequest("PUT", ts.URL+"/"+bucket+"?cors", strings.NewReader(corsPayload))
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusOK)
	req, _ = http.NewRequest("GET", ts.URL+"/"+bucket+"?cors", nil)
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusOK)

	// Lifecycle configuration.
	lifecyclePayload := `<?xml version="1.0" encoding="UTF-8"?>
<LifecycleConfiguration xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <Rule>
    <ID>rule-1</ID>
    <Prefix>logs/</Prefix>
    <Status>Enabled</Status>
  </Rule>
</LifecycleConfiguration>`
	req, _ = http.NewRequest("PUT", ts.URL+"/"+bucket+"?lifecycle", strings.NewReader(lifecyclePayload))
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusOK)
	req, _ = http.NewRequest("GET", ts.URL+"/"+bucket+"?lifecycle", nil)
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusOK)

	// Tagging configuration.
	taggingPayload := `<?xml version="1.0" encoding="UTF-8"?>
<Tagging xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <TagSet>
    <Tag>
      <Key>env</Key>
      <Value>dev</Value>
    </Tag>
  </TagSet>
</Tagging>`
	req, _ = http.NewRequest("PUT", ts.URL+"/"+bucket+"?tagging", strings.NewReader(taggingPayload))
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusOK)
	req, _ = http.NewRequest("GET", ts.URL+"/"+bucket+"?tagging", nil)
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusOK)

	// Website configuration.
	websitePayload := `<?xml version="1.0" encoding="UTF-8"?>
<WebsiteConfiguration xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <IndexDocument>
    <Suffix>index.html</Suffix>
  </IndexDocument>
  <ErrorDocument>
    <Key>error.html</Key>
  </ErrorDocument>
</WebsiteConfiguration>`
	req, _ = http.NewRequest("PUT", ts.URL+"/"+bucket+"?website", strings.NewReader(websitePayload))
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusOK)
	req, _ = http.NewRequest("GET", ts.URL+"/"+bucket+"?website", nil)
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusOK)

	// Logging configuration.
	loggingPayload := `<?xml version="1.0" encoding="UTF-8"?>
<BucketLoggingStatus xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <LoggingEnabled>
    <TargetBucket>` + bucket + `</TargetBucket>
    <TargetPrefix>logs/</TargetPrefix>
  </LoggingEnabled>
</BucketLoggingStatus>`
	req, _ = http.NewRequest("PUT", ts.URL+"/"+bucket+"?logging", strings.NewReader(loggingPayload))
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusOK)
	req, _ = http.NewRequest("GET", ts.URL+"/"+bucket+"?logging", nil)
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusOK)

	// Verify access log delivery.
	respBody = mustBody(t, signedRequest(t, "GET", ts.URL+"/"+bucket+"?list-type=2&prefix=logs/", nil, nil))
	var logList s3ListBucketResultV2
	if err := xml.Unmarshal(respBody, &logList); err != nil {
		t.Fatalf("log list parse: %v", err)
	}
	if len(logList.Contents) == 0 || !strings.HasPrefix(logList.Contents[0].Key, "logs/") {
		t.Fatalf("expected access log objects with logs/ prefix")
	}

	// Replication configuration and delivery.
	replicaBucket := "replica-bucket"
	resp = signedRequest(t, "PUT", ts.URL+"/"+replicaBucket, nil, nil)
	assertStatus(t, resp, http.StatusOK)
	req, _ = http.NewRequest("PUT", ts.URL+"/"+replicaBucket+"?versioning", strings.NewReader(versioningPayload))
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusOK)
	replicationPayload := `<?xml version="1.0" encoding="UTF-8"?>
<ReplicationConfiguration xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <Role>arn:aws:iam::123456789012:role/replication</Role>
  <Rule>
    <ID>replicate-all</ID>
    <Status>Enabled</Status>
    <Prefix></Prefix>
    <DeleteMarkerReplication>
      <Status>Enabled</Status>
    </DeleteMarkerReplication>
    <Destination>
      <Bucket>` + replicaBucket + `</Bucket>
    </Destination>
  </Rule>
</ReplicationConfiguration>`
	req, _ = http.NewRequest("PUT", ts.URL+"/"+bucket+"?replication", strings.NewReader(replicationPayload))
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusOK)
	replicationHeaders := map[string]string{
		"x-amz-storage-class": "STANDARD_IA",
		"x-amz-meta-origin":   "primary",
	}
	resp = signedRequest(t, "PUT", ts.URL+"/"+bucket+"/notes/replicated.txt", []byte("replicated"), replicationHeaders)
	assertStatus(t, resp, http.StatusOK)
	replicatedVersion := resp.Header.Get("x-amz-version-id")
	resp = signedRequest(t, "GET", ts.URL+"/"+replicaBucket+"/notes/replicated.txt", nil, nil)
	assertStatus(t, resp, http.StatusOK)
	if resp.Header.Get("x-amz-meta-origin") != "primary" {
		t.Fatalf("expected replica metadata origin=primary, got %q", resp.Header.Get("x-amz-meta-origin"))
	}
	if replicatedVersion != "" && resp.Header.Get("x-amz-version-id") != replicatedVersion {
		t.Fatalf("expected replica version id %q, got %q", replicatedVersion, resp.Header.Get("x-amz-version-id"))
	}
	respBody = mustBody(t, signedRequest(t, "GET", ts.URL+"/"+replicaBucket+"?list-type=2&prefix=notes/replicated.txt", nil, nil))
	var replicaList s3ListBucketResultV2
	if err := xml.Unmarshal(respBody, &replicaList); err != nil {
		t.Fatalf("replica list parse: %v", err)
	}
	if len(replicaList.Contents) == 0 || replicaList.Contents[0].StorageClass != "STANDARD_IA" {
		t.Fatalf("expected replica storage class STANDARD_IA, got %#v", replicaList.Contents)
	}
	req, _ = http.NewRequest("DELETE", ts.URL+"/"+bucket+"/notes/replicated.txt?x-amz-bypass-governance-retention=true", nil)
	req.Header.Set("x-amz-bypass-governance-retention", "true")
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusNoContent)
	deleteVersion := resp.Header.Get("x-amz-version-id")
	resp = signedRequest(t, "GET", ts.URL+"/"+replicaBucket+"/notes/replicated.txt", nil, nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
	if deleteVersion != "" {
		resp = signedRequest(t, "GET", ts.URL+"/"+replicaBucket+"?versions&prefix=notes/replicated.txt", nil, nil)
		assertStatus(t, resp, http.StatusOK)
		body = mustBody(t, resp)
		var versionList s3ListObjectVersionsResult
		if err := xml.Unmarshal(body, &versionList); err != nil {
			t.Fatalf("replica list versions parse: %v", err)
		}
		found := false
		for _, marker := range versionList.DeleteMarkers {
			if marker.VersionID == deleteVersion {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected delete marker version %q on replica", deleteVersion)
		}
	}

	// Multipart upload with invalid part order.
	uploadID := initiateMultipart(t, ts.URL, bucket, "notes/multipart.txt")
	_ = initiateMultipart(t, ts.URL, bucket, "notes/sub/multipart.txt")
	// List multipart uploads.
	req, _ = http.NewRequest("GET", ts.URL+"/"+bucket+"?uploads&prefix=notes/", nil)
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusOK)
	body = mustBody(t, resp)
	var uploadsResult s3ListMultipartUploadsResult
	if err := xml.Unmarshal(body, &uploadsResult); err != nil {
		t.Fatalf("list uploads parse: %v", err)
	}
	if len(uploadsResult.Uploads) == 0 {
		t.Fatalf("expected multipart uploads")
	}
	req, _ = http.NewRequest("GET", ts.URL+"/"+bucket+"?uploads&prefix=notes/&delimiter=/", nil)
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusOK)
	body = mustBody(t, resp)
	var uploadsDelimited s3ListMultipartUploadsResult
	if err := xml.Unmarshal(body, &uploadsDelimited); err != nil {
		t.Fatalf("list uploads delimiter parse: %v", err)
	}
	if len(uploadsDelimited.CommonPrefixes) == 0 {
		t.Fatalf("expected common prefixes for delimiter listing")
	}
	req, _ = http.NewRequest("GET", ts.URL+"/"+bucket+"?uploads&prefix=notes/&encoding-type=url", nil)
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusOK)
	body = mustBody(t, resp)
	var uploadsEncoded s3ListMultipartUploadsResult
	if err := xml.Unmarshal(body, &uploadsEncoded); err != nil {
		t.Fatalf("list uploads encoding parse: %v", err)
	}
	if uploadsEncoded.EncodingType != "url" {
		t.Fatalf("expected encoding type url, got %q", uploadsEncoded.EncodingType)
	}
	req, _ = http.NewRequest("GET", ts.URL+"/"+bucket+"?uploads&prefix=notes/&max-uploads=1", nil)
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusOK)
	body = mustBody(t, resp)
	var uploadsPage s3ListMultipartUploadsResult
	if err := xml.Unmarshal(body, &uploadsPage); err != nil {
		t.Fatalf("list uploads page parse: %v", err)
	}
	if !uploadsPage.IsTruncated || uploadsPage.NextKeyMarker == "" {
		t.Fatalf("expected truncated uploads with next marker")
	}
	part1ETag := uploadPart(t, ts.URL, bucket, "notes/multipart.txt", uploadID, 1, []byte("small-1"))
	part2ETag := uploadPart(t, ts.URL, bucket, "notes/multipart.txt", uploadID, 2, []byte("small-2"))
	// List multipart parts.
	req, _ = http.NewRequest("GET", ts.URL+"/"+bucket+"/notes/multipart.txt?uploadId="+url.QueryEscape(uploadID), nil)
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusOK)
	body = mustBody(t, resp)
	var partsResult s3ListPartsResult
	if err := xml.Unmarshal(body, &partsResult); err != nil {
		t.Fatalf("list parts parse: %v", err)
	}
	if len(partsResult.Parts) < 2 {
		t.Fatalf("expected parts, got %d", len(partsResult.Parts))
	}
	req, _ = http.NewRequest("GET", ts.URL+"/"+bucket+"/notes/multipart.txt?uploadId="+url.QueryEscape(uploadID)+"&encoding-type=url", nil)
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusOK)
	body = mustBody(t, resp)
	var partsEncoded s3ListPartsResult
	if err := xml.Unmarshal(body, &partsEncoded); err != nil {
		t.Fatalf("list parts encoding parse: %v", err)
	}
	if partsEncoded.EncodingType != "url" {
		t.Fatalf("expected parts encoding type url, got %q", partsEncoded.EncodingType)
	}
	req, _ = http.NewRequest("GET", ts.URL+"/"+bucket+"/notes/multipart.txt?uploadId="+url.QueryEscape(uploadID)+"&max-parts=1", nil)
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusOK)
	body = mustBody(t, resp)
	var partsPage s3ListPartsResult
	if err := xml.Unmarshal(body, &partsPage); err != nil {
		t.Fatalf("list parts page parse: %v", err)
	}
	if !partsPage.IsTruncated || partsPage.NextPartNumberMarker == 0 {
		t.Fatalf("expected truncated parts with next marker")
	}
	completePayload := completeMultipartPayload([]multipartPart{
		{Number: 2, ETag: part2ETag},
		{Number: 1, ETag: part1ETag},
	})
	req, _ = http.NewRequest("POST", ts.URL+"/"+bucket+"/notes/multipart.txt?uploadId="+url.QueryEscape(uploadID), strings.NewReader(completePayload))
	resp = signedRequestWithRequest(t, req, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
	assertS3ErrorCode(t, resp, "InvalidPartOrder")

	// Complete multipart upload with no parts.
	req, _ = http.NewRequest("POST", ts.URL+"/"+bucket+"/notes/multipart.txt?uploadId="+url.QueryEscape(uploadID), strings.NewReader("<CompleteMultipartUpload></CompleteMultipartUpload>"))
	resp = signedRequestWithRequest(t, req, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty parts, got %d", resp.StatusCode)
	}
	assertS3ErrorCode(t, resp, "InvalidRequest")

	// Complete multipart upload with mismatched key.
	req, _ = http.NewRequest("POST", ts.URL+"/"+bucket+"/notes/multipart-other.txt?uploadId="+url.QueryEscape(uploadID), strings.NewReader(completeMultipartPayload([]multipartPart{
		{Number: 1, ETag: part1ETag},
	})))
	resp = signedRequestWithRequest(t, req, nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for key mismatch, got %d", resp.StatusCode)
	}
	assertS3ErrorCode(t, resp, "NoSuchUpload")

	// Complete multipart upload with mismatched ETag.
	req, _ = http.NewRequest("POST", ts.URL+"/"+bucket+"/notes/multipart.txt?uploadId="+url.QueryEscape(uploadID), strings.NewReader(completeMultipartPayload([]multipartPart{
		{Number: 1, ETag: "deadbeef"},
		{Number: 2, ETag: part2ETag},
	})))
	resp = signedRequestWithRequest(t, req, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid part etag, got %d", resp.StatusCode)
	}
	assertS3ErrorCode(t, resp, "InvalidPart")

	// Complete multipart upload with missing part.
	req, _ = http.NewRequest("POST", ts.URL+"/"+bucket+"/notes/multipart.txt?uploadId="+url.QueryEscape(uploadID), strings.NewReader(completeMultipartPayload([]multipartPart{
		{Number: 1, ETag: part1ETag},
		{Number: 3, ETag: "missing"},
	})))
	resp = signedRequestWithRequest(t, req, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing part, got %d", resp.StatusCode)
	}
	assertS3ErrorCode(t, resp, "InvalidPart")

	// Complete multipart upload with duplicate part numbers.
	req, _ = http.NewRequest("POST", ts.URL+"/"+bucket+"/notes/multipart.txt?uploadId="+url.QueryEscape(uploadID), strings.NewReader(completeMultipartPayload([]multipartPart{
		{Number: 1, ETag: part1ETag},
		{Number: 1, ETag: part1ETag},
	})))
	resp = signedRequestWithRequest(t, req, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for duplicate parts, got %d", resp.StatusCode)
	}
	assertS3ErrorCode(t, resp, "InvalidPartOrder")

	// Complete multipart upload with too many parts.
	tooMany := make([]multipartPart, 0, 10001)
	for i := 1; i <= 10001; i++ {
		tooMany = append(tooMany, multipartPart{Number: i, ETag: "etag"})
	}
	req, _ = http.NewRequest("POST", ts.URL+"/"+bucket+"/notes/multipart.txt?uploadId="+url.QueryEscape(uploadID), strings.NewReader(completeMultipartPayload(tooMany)))
	resp = signedRequestWithRequest(t, req, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for too many parts, got %d", resp.StatusCode)
	}
	assertS3ErrorCode(t, resp, "InvalidRequest")

	// Multipart upload valid single part (last part can be small).
	uploadID = initiateMultipart(t, ts.URL, bucket, "notes/multipart-ok.txt")
	part1ETag = uploadPart(t, ts.URL, bucket, "notes/multipart-ok.txt", uploadID, 1, []byte("small-1"))
	completePayload = completeMultipartPayload([]multipartPart{
		{Number: 1, ETag: part1ETag},
	})
	req, _ = http.NewRequest("POST", ts.URL+"/"+bucket+"/notes/multipart-ok.txt?uploadId="+url.QueryEscape(uploadID), strings.NewReader(completePayload))
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusOK)

	// Upload part copy.
	resp = signedRequest(t, "PUT", ts.URL+"/"+bucket+"/notes/copy-source.txt", []byte("copy-source"), nil)
	assertStatus(t, resp, http.StatusOK)
	copyUploadID := initiateMultipart(t, ts.URL, bucket, "notes/multipart-copy.txt")
	req, _ = http.NewRequest("PUT", ts.URL+"/"+bucket+"/notes/multipart-copy.txt?uploadId="+url.QueryEscape(copyUploadID)+"&partNumber=1", nil)
	req.Header.Set("x-amz-copy-source", bucket+"/notes/copy-source.txt")
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusOK)

	// Delete bucket behaviors.
	deleteBucket := "delete-me"
	resp = signedRequest(t, "PUT", ts.URL+"/"+deleteBucket, nil, nil)
	assertStatus(t, resp, http.StatusOK)
	resp = signedRequest(t, "DELETE", ts.URL+"/"+deleteBucket, nil, nil)
	assertStatus(t, resp, http.StatusNoContent)

	notEmpty := "not-empty"
	resp = signedRequest(t, "PUT", ts.URL+"/"+notEmpty, nil, nil)
	assertStatus(t, resp, http.StatusOK)
	resp = signedRequest(t, "PUT", ts.URL+"/"+notEmpty+"/keep.txt", []byte("data"), nil)
	assertStatus(t, resp, http.StatusOK)
	resp = signedRequest(t, "DELETE", ts.URL+"/"+notEmpty, nil, nil)
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409 for non-empty bucket delete, got %d", resp.StatusCode)
	}

	// Get bucket location.
	req, _ = http.NewRequest("GET", ts.URL+"/"+bucket+"?location", nil)
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusOK)
	body = mustBody(t, resp)
	var location s3BucketLocationResult
	if err := xml.Unmarshal(body, &location); err != nil {
		t.Fatalf("location parse: %v", err)
	}

	// Bucket policy.
	policyPayload := `{
  "Version": "2012-10-17",
  "Statement": [{
    "Sid": "PublicRead",
    "Effect": "Allow",
    "Principal": "*",
    "Action": "s3:GetObject",
    "Resource": "arn:aws:s3:::` + bucket + `/*"
  }]
}`
	req, _ = http.NewRequest("PUT", ts.URL+"/"+bucket+"?policy", strings.NewReader(policyPayload))
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusOK)
	req, _ = http.NewRequest("GET", ts.URL+"/"+bucket+"?policy", nil)
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusOK)
	body = mustBody(t, resp)
	if !strings.Contains(string(body), `"Statement"`) {
		t.Fatalf("expected policy body, got %s", string(body))
	}
	// Bucket policy status.
	req, _ = http.NewRequest("GET", ts.URL+"/"+bucket+"?policyStatus", nil)
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusOK)
	body = mustBody(t, resp)
	var policyStatus s3BucketPolicyStatus
	if err := xml.Unmarshal(body, &policyStatus); err != nil {
		t.Fatalf("policy status parse: %v", err)
	}
	if !policyStatus.IsPublic {
		t.Fatalf("expected policy status to be public")
	}

	req, _ = http.NewRequest("DELETE", ts.URL+"/"+bucket+"?policy", nil)
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusNoContent)

	// Public access block.
	publicAccessPayload := `<?xml version="1.0" encoding="UTF-8"?>
<PublicAccessBlockConfiguration xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <BlockPublicAcls>true</BlockPublicAcls>
  <IgnorePublicAcls>true</IgnorePublicAcls>
  <BlockPublicPolicy>true</BlockPublicPolicy>
  <RestrictPublicBuckets>true</RestrictPublicBuckets>
</PublicAccessBlockConfiguration>`
	req, _ = http.NewRequest("PUT", ts.URL+"/"+bucket+"?publicAccessBlock", strings.NewReader(publicAccessPayload))
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusOK)
	req, _ = http.NewRequest("GET", ts.URL+"/"+bucket+"?publicAccessBlock", nil)
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusOK)
	body = mustBody(t, resp)
	var publicAccess s3PublicAccessBlockConfiguration
	if err := xml.Unmarshal(body, &publicAccess); err != nil {
		t.Fatalf("public access block parse: %v", err)
	}
	if !publicAccess.BlockPublicAcls || !publicAccess.IgnorePublicAcls || !publicAccess.BlockPublicPolicy || !publicAccess.RestrictPublicBuckets {
		t.Fatalf("unexpected public access block config: %+v", publicAccess)
	}
	req, _ = http.NewRequest("DELETE", ts.URL+"/"+bucket+"?publicAccessBlock", nil)
	resp = signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusNoContent)
}

func signedRequest(t *testing.T, method, urlStr string, body []byte, headers map[string]string) *http.Response {
	t.Helper()
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, urlStr, reader)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	return signedRequestWithRequest(t, req, body)
}

func signedRequestWithRequest(t *testing.T, req *http.Request, body []byte) *http.Response {
	t.Helper()
	if req.Host == "" {
		req.Host = req.URL.Host
	}
	if body == nil && req.Body != nil {
		read, err := io.ReadAll(req.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		body = read
		req.Body = io.NopCloser(bytes.NewReader(read))
	}
	amzDate := time.Now().UTC().Format("20060102T150405Z")
	req.Header.Set("x-amz-date", amzDate)
	payloadHash := sha256Hex(body)
	req.Header.Set("x-amz-content-sha256", payloadHash)
	signRequest(req, payloadHash, amzDate)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	return resp
}

func signRequest(req *http.Request, payloadHash, amzDate string) {
	signedHeaders := "host;x-amz-content-sha256;x-amz-date"
	canonicalRequest, _ := buildCanonicalRequest(req, signedHeaders, payloadHash, false)
	stringToSign := buildStringToSign(amzDate, testRegion, "s3", canonicalRequest)
	signature := signString(stringToSign, testRegion, "s3", testSecretKey)
	credentialScope := amzDate[:8] + "/" + testRegion + "/s3/aws4_request"
	auth := "AWS4-HMAC-SHA256 Credential=" + testAccessKey + "/" + credentialScope + ", SignedHeaders=" + signedHeaders + ", Signature=" + signature
	req.Header.Set("Authorization", auth)
}

// canonical helpers use server implementations

func assertStatus(t *testing.T, resp *http.Response, want int) {
	t.Helper()
	if resp.StatusCode != want {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected %d, got %d: %s", want, resp.StatusCode, string(body))
	}
}

func mustBody(t *testing.T, resp *http.Response) []byte {
	t.Helper()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return body
}

func assertS3ErrorCode(t *testing.T, resp *http.Response, want string) {
	t.Helper()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read error body: %v", err)
	}
	_ = resp.Body.Close()
	if len(body) == 0 {
		t.Fatalf("expected XML error body, got empty response")
	}
	var errResp s3ErrorResponse
	if err := xml.Unmarshal(body, &errResp); err != nil {
		t.Fatalf("parse error XML: %v (%s)", err, string(body))
	}
	if errResp.Code != want {
		t.Fatalf("expected error code %q, got %q", want, errResp.Code)
	}
}

func assertXMLRoot(t *testing.T, body []byte, root string) {
	t.Helper()
	trimmed := strings.TrimSpace(string(body))
	if !strings.HasPrefix(trimmed, "<"+root) {
		t.Fatalf("expected XML root %q, got %q", root, trimmed)
	}
	if !strings.Contains(trimmed, `xmlns="http://s3.amazonaws.com/doc/2006-03-01/"`) {
		t.Fatalf("expected S3 namespace on %q response", root)
	}
}

func assertXMLOrder(t *testing.T, body []byte, elements []string) {
	t.Helper()
	text := string(body)
	last := -1
	for _, element := range elements {
		idx := strings.Index(text, element)
		if idx == -1 {
			t.Fatalf("missing XML element %q", element)
		}
		if idx < last {
			t.Fatalf("unexpected XML element order for %q", element)
		}
		last = idx
	}
}

type multipartPart struct {
	Number int
	ETag   string
}

func initiateMultipart(t *testing.T, baseURL, bucket, key string) string {
	req, _ := http.NewRequest("POST", baseURL+"/"+bucket+"/"+key+"?uploads", nil)
	resp := signedRequestWithRequest(t, req, nil)
	assertStatus(t, resp, http.StatusOK)
	body := mustBody(t, resp)
	var out s3InitiateMultipartUploadResult
	if err := xml.Unmarshal(body, &out); err != nil {
		t.Fatalf("init mpu parse: %v", err)
	}
	if out.UploadID == "" {
		t.Fatalf("missing upload id")
	}
	return out.UploadID
}

func uploadPart(t *testing.T, baseURL, bucket, key, uploadID string, partNumber int, body []byte) string {
	req, _ := http.NewRequest("PUT", baseURL+"/"+bucket+"/"+key+"?uploadId="+url.QueryEscape(uploadID)+"&partNumber="+strconv.Itoa(partNumber), bytes.NewReader(body))
	resp := signedRequestWithRequest(t, req, body)
	assertStatus(t, resp, http.StatusOK)
	etag := strings.Trim(resp.Header.Get("ETag"), "\"")
	if etag == "" {
		t.Fatalf("missing part etag")
	}
	return etag
}

func completeMultipartPayload(parts []multipartPart) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?><CompleteMultipartUpload>`)
	for _, part := range parts {
		b.WriteString("<Part><PartNumber>")
		b.WriteString(strconv.Itoa(part.Number))
		b.WriteString("</PartNumber><ETag>")
		b.WriteString(part.ETag)
		b.WriteString("</ETag></Part>")
	}
	b.WriteString(`</CompleteMultipartUpload>`)
	return b.String()
}

func presignURL(t *testing.T, urlStr string, expires int) string {
	t.Helper()
	parsed, err := url.Parse(urlStr)
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}
	amzDate := time.Now().UTC().Format("20060102T150405Z")
	credentialScope := amzDate[:8] + "/" + testRegion + "/s3/aws4_request"
	query := parsed.Query()
	query.Set("X-Amz-Algorithm", "AWS4-HMAC-SHA256")
	query.Set("X-Amz-Credential", testAccessKey+"/"+credentialScope)
	query.Set("X-Amz-Date", amzDate)
	query.Set("X-Amz-Expires", strconv.Itoa(expires))
	query.Set("X-Amz-SignedHeaders", "host")
	parsed.RawQuery = query.Encode()

	canonicalRequest, _ := buildCanonicalRequest(&http.Request{
		Method: "GET",
		URL:    parsed,
		Host:   parsed.Host,
		Header: http.Header{"Host": []string{parsed.Host}},
	}, "host", "UNSIGNED-PAYLOAD", true)
	stringToSign := buildStringToSign(amzDate, testRegion, "s3", canonicalRequest)
	signature := signString(stringToSign, testRegion, "s3", testSecretKey)
	query.Set("X-Amz-Signature", signature)
	parsed.RawQuery = query.Encode()
	return parsed.String()
}
