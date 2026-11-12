package server

import (
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestS3ControlAccountPublicAccessBlock(t *testing.T) {
	srv := New(Config{
		Addr:      "127.0.0.1:0",
		AccessKey: testAccessKey,
		SecretKey: testSecretKey,
		LogLevel:  "error",
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	accountID := "123456789012"
	headers := map[string]string{
		"x-amz-account-id": accountID,
		"Content-Type":     "application/xml",
	}
	urlStr := ts.URL + "/v20180820/configuration/publicAccessBlock"

	resp := signedRequestWithService(t, http.MethodGet, urlStr, nil, headers, "s3-control")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for missing public access block, got %d", resp.StatusCode)
	}
	assertS3ErrorCode(t, resp, "NoSuchPublicAccessBlockConfiguration")

	resp = signedRequestWithService(t, http.MethodPost, urlStr, nil, headers, "s3-control")
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405 for unsupported method, got %d", resp.StatusCode)
	}

	badBody := `<PublicAccessBlockConfiguration><BlockPublicAcls>true</BlockPublicAcls></PublicAccessBlockConfiguration>`
	resp = signedRequestWithService(t, http.MethodPut, urlStr, []byte(badBody), headers, "s3-control")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing namespace, got %d", resp.StatusCode)
	}
	assertS3ErrorCode(t, resp, "MalformedXML")

	unknownBody := `<?xml version="1.0" encoding="UTF-8"?>` +
		`<PublicAccessBlockConfiguration xmlns="http://awss3control.amazonaws.com/doc/2018-08-20/">` +
		`<BlockPublicAcls>true</BlockPublicAcls>` +
		`<UnknownField>true</UnknownField>` +
		`</PublicAccessBlockConfiguration>`
	resp = signedRequestWithService(t, http.MethodPut, urlStr, []byte(unknownBody), headers, "s3-control")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for unknown element, got %d", resp.StatusCode)
	}
	assertS3ErrorCode(t, resp, "MalformedXML")

	missingFieldBody := `<?xml version="1.0" encoding="UTF-8"?>` +
		`<PublicAccessBlockConfiguration xmlns="http://awss3control.amazonaws.com/doc/2018-08-20/">` +
		`<BlockPublicAcls>true</BlockPublicAcls>` +
		`<IgnorePublicAcls>true</IgnorePublicAcls>` +
		`<BlockPublicPolicy>false</BlockPublicPolicy>` +
		`</PublicAccessBlockConfiguration>`
	resp = signedRequestWithService(t, http.MethodPut, urlStr, []byte(missingFieldBody), headers, "s3-control")
	assertStatus(t, resp, http.StatusOK)

	orderBody := `<?xml version="1.0" encoding="UTF-8"?>` +
		`<PublicAccessBlockConfiguration xmlns="http://awss3control.amazonaws.com/doc/2018-08-20/">` +
		`<BlockPublicAcls>true</BlockPublicAcls>` +
		`<BlockPublicPolicy>false</BlockPublicPolicy>` +
		`<IgnorePublicAcls>true</IgnorePublicAcls>` +
		`<RestrictPublicBuckets>false</RestrictPublicBuckets>` +
		`</PublicAccessBlockConfiguration>`
	resp = signedRequestWithService(t, http.MethodPut, urlStr, []byte(orderBody), headers, "s3-control")
	assertStatus(t, resp, http.StatusOK)

	invalidBoolBody := `<?xml version="1.0" encoding="UTF-8"?>` +
		`<PublicAccessBlockConfiguration xmlns="http://awss3control.amazonaws.com/doc/2018-08-20/">` +
		`<BlockPublicAcls>yes</BlockPublicAcls>` +
		`<IgnorePublicAcls>true</IgnorePublicAcls>` +
		`<BlockPublicPolicy>false</BlockPublicPolicy>` +
		`<RestrictPublicBuckets>false</RestrictPublicBuckets>` +
		`</PublicAccessBlockConfiguration>`
	resp = signedRequestWithService(t, http.MethodPut, urlStr, []byte(invalidBoolBody), headers, "s3-control")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid boolean value, got %d", resp.StatusCode)
	}
	assertS3ErrorCode(t, resp, "MalformedXML")

	putBody := `<?xml version="1.0" encoding="UTF-8"?>` +
		`<PublicAccessBlockConfiguration xmlns="http://awss3control.amazonaws.com/doc/2018-08-20/">` +
		`<BlockPublicAcls>true</BlockPublicAcls>` +
		`<IgnorePublicAcls>true</IgnorePublicAcls>` +
		`<BlockPublicPolicy>false</BlockPublicPolicy>` +
		`<RestrictPublicBuckets>false</RestrictPublicBuckets>` +
		`</PublicAccessBlockConfiguration>`
	resp = signedRequestWithService(t, http.MethodPut, urlStr, []byte(putBody), headers, "s3-control")
	assertStatus(t, resp, http.StatusOK)

	resp = signedRequestWithService(t, http.MethodGet, urlStr, nil, headers, "s3-control")
	assertStatus(t, resp, http.StatusOK)
	var cfg s3ControlAccountPublicAccessBlockConfiguration
	body := mustBody(t, resp)
	if err := xml.Unmarshal(body, &cfg); err != nil {
		t.Fatalf("parse public access block response: %v", err)
	}
	if cfg.Xmlns == "" || !strings.Contains(cfg.Xmlns, "awss3control") {
		t.Fatalf("expected s3 control namespace, got %q", cfg.Xmlns)
	}
	if !cfg.BlockPublicAcls || !cfg.IgnorePublicAcls {
		t.Fatalf("expected public access block values to be set")
	}

	resp = signedRequestWithService(t, http.MethodDelete, urlStr, nil, headers, "s3-control")
	assertStatus(t, resp, http.StatusOK)

	resp = signedRequestWithService(t, http.MethodGet, urlStr, nil, headers, "s3-control")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 after delete, got %d", resp.StatusCode)
	}
	assertS3ErrorCode(t, resp, "NoSuchPublicAccessBlockConfiguration")

	resp = signedRequestWithService(t, http.MethodDelete, urlStr, nil, headers, "s3-control")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for delete after missing config, got %d", resp.StatusCode)
	}
	assertS3ErrorCode(t, resp, "NoSuchPublicAccessBlockConfiguration")
}
