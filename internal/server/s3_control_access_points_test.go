package server

import (
	"bytes"
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestS3ControlAccessPointsStage1(t *testing.T) {
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

	resp := signedRequest(t, http.MethodPut, ts.URL+"/ap-bucket", nil, nil)
	assertStatus(t, resp, http.StatusOK)

	badBody := `<CreateAccessPointRequest><Bucket>ap-bucket</Bucket></CreateAccessPointRequest>`
	resp = signedRequestWithService(t, http.MethodPut, ts.URL+"/v20180820/accesspoint/ap-one", []byte(badBody), headers, "s3-control")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing namespace, got %d", resp.StatusCode)
	}

	orderBody := `<?xml version="1.0" encoding="UTF-8"?>` +
		`<CreateAccessPointRequest xmlns="http://awss3control.amazonaws.com/doc/2018-08-20/">` +
		`<Bucket>ap-bucket</Bucket>` +
		`<PublicAccessBlockConfiguration>` +
		`<BlockPublicAcls>false</BlockPublicAcls>` +
		`<BlockPublicPolicy>false</BlockPublicPolicy>` +
		`<IgnorePublicAcls>true</IgnorePublicAcls>` +
		`<RestrictPublicBuckets>false</RestrictPublicBuckets>` +
		`</PublicAccessBlockConfiguration>` +
		`</CreateAccessPointRequest>`
	resp = signedRequestWithService(t, http.MethodPut, ts.URL+"/v20180820/accesspoint/ap-one", []byte(orderBody), headers, "s3-control")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid PAB order, got %d", resp.StatusCode)
	}

	invalidBoolBody := `<?xml version="1.0" encoding="UTF-8"?>` +
		`<CreateAccessPointRequest xmlns="http://awss3control.amazonaws.com/doc/2018-08-20/">` +
		`<Bucket>ap-bucket</Bucket>` +
		`<PublicAccessBlockConfiguration>` +
		`<BlockPublicAcls>yes</BlockPublicAcls>` +
		`<IgnorePublicAcls>true</IgnorePublicAcls>` +
		`<BlockPublicPolicy>false</BlockPublicPolicy>` +
		`<RestrictPublicBuckets>false</RestrictPublicBuckets>` +
		`</PublicAccessBlockConfiguration>` +
		`</CreateAccessPointRequest>`
	resp = signedRequestWithService(t, http.MethodPut, ts.URL+"/v20180820/accesspoint/ap-one", []byte(invalidBoolBody), headers, "s3-control")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid PAB bool, got %d", resp.StatusCode)
	}

	missingFieldBody := `<?xml version="1.0" encoding="UTF-8"?>` +
		`<CreateAccessPointRequest xmlns="http://awss3control.amazonaws.com/doc/2018-08-20/">` +
		`<Bucket>ap-bucket</Bucket>` +
		`<PublicAccessBlockConfiguration>` +
		`<BlockPublicAcls>false</BlockPublicAcls>` +
		`<IgnorePublicAcls>true</IgnorePublicAcls>` +
		`<BlockPublicPolicy>false</BlockPublicPolicy>` +
		`</PublicAccessBlockConfiguration>` +
		`</CreateAccessPointRequest>`
	resp = signedRequestWithService(t, http.MethodPut, ts.URL+"/v20180820/accesspoint/ap-one", []byte(missingFieldBody), headers, "s3-control")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing PAB field, got %d", resp.StatusCode)
	}

	createBody := `<?xml version="1.0" encoding="UTF-8"?>` +
		`<CreateAccessPointRequest xmlns="http://awss3control.amazonaws.com/doc/2018-08-20/">` +
		`<Bucket>ap-bucket</Bucket>` +
		`<PublicAccessBlockConfiguration>` +
		`<BlockPublicAcls>false</BlockPublicAcls>` +
		`<IgnorePublicAcls>true</IgnorePublicAcls>` +
		`<BlockPublicPolicy>false</BlockPublicPolicy>` +
		`<RestrictPublicBuckets>false</RestrictPublicBuckets>` +
		`</PublicAccessBlockConfiguration>` +
		`</CreateAccessPointRequest>`
	resp = signedRequestWithService(t, http.MethodPut, ts.URL+"/v20180820/accesspoint/ap-one", []byte(createBody), headers, "s3-control")
	assertStatus(t, resp, http.StatusOK)
	var createResp s3ControlCreateAccessPointResult
	if err := xml.Unmarshal(mustBody(t, resp), &createResp); err != nil {
		t.Fatalf("parse create access point response: %v", err)
	}
	if !strings.Contains(createResp.AccessPointArn, "accesspoint/ap-one") {
		t.Fatalf("expected access point arn, got %q", createResp.AccessPointArn)
	}
	if createResp.Alias == "" {
		t.Fatalf("expected access point alias")
	}

	resp = signedRequestWithService(t, http.MethodPut, ts.URL+"/v20180820/accesspoint/ap-two", []byte(createBody), headers, "s3-control")
	assertStatus(t, resp, http.StatusOK)

	resp = signedRequestWithService(t, http.MethodGet, ts.URL+"/v20180820/accesspoint/ap-one", nil, headers, "s3-control")
	assertStatus(t, resp, http.StatusOK)
	var getResp s3ControlAccessPointResult
	if err := xml.Unmarshal(mustBody(t, resp), &getResp); err != nil {
		t.Fatalf("parse get access point response: %v", err)
	}
	if getResp.Name != "ap-one" || getResp.Bucket != "ap-bucket" {
		t.Fatalf("unexpected access point response: %+v", getResp)
	}
	if getResp.NetworkOrigin != "Internet" {
		t.Fatalf("expected Internet network origin, got %q", getResp.NetworkOrigin)
	}
	if getResp.PublicAccessBlockConfiguration == nil || !getResp.PublicAccessBlockConfiguration.IgnorePublicAcls {
		t.Fatalf("expected public access block configuration")
	}

	scopeBody := `<?xml version="1.0" encoding="UTF-8"?>` +
		`<Scope xmlns="http://awss3control.amazonaws.com/doc/2018-08-20/">` +
		`<Permissions><Permission>GetObject</Permission></Permissions>` +
		`<Prefixes><Prefix>docs/</Prefix></Prefixes>` +
		`</Scope>`
	resp = signedRequestWithService(t, http.MethodPut, ts.URL+"/v20180820/accesspoint/ap-one/scope", []byte(scopeBody), headers, "s3-control")
	assertStatus(t, resp, http.StatusOK)

	resp = signedRequestWithService(t, http.MethodGet, ts.URL+"/v20180820/accesspoint/ap-one/scope", nil, headers, "s3-control")
	assertStatus(t, resp, http.StatusOK)
	var scopeResp s3ControlGetAccessPointScopeResult
	if err := xml.Unmarshal(mustBody(t, resp), &scopeResp); err != nil {
		t.Fatalf("parse get access point scope response: %v", err)
	}
	if len(scopeResp.Scope.Permissions) != 1 || scopeResp.Scope.Permissions[0] != "GetObject" {
		t.Fatalf("unexpected scope permissions: %+v", scopeResp.Scope.Permissions)
	}
	if len(scopeResp.Scope.Prefixes) != 1 || scopeResp.Scope.Prefixes[0] != "docs/" {
		t.Fatalf("unexpected scope prefixes: %+v", scopeResp.Scope.Prefixes)
	}

	resp = signedRequestWithService(t, http.MethodGet, ts.URL+"/v20180820/accesspointfordirectory?directoryBucket=ap-bucket", nil, headers, "s3-control")
	assertStatus(t, resp, http.StatusOK)
	var listDirectoryResp s3ControlListAccessPointsForDirectoryBucketsResult
	if err := xml.Unmarshal(mustBody(t, resp), &listDirectoryResp); err != nil {
		t.Fatalf("parse list access points for directory buckets response: %v", err)
	}
	if len(listDirectoryResp.AccessPointList) != 2 {
		t.Fatalf("expected 2 directory bucket access points, got %d", len(listDirectoryResp.AccessPointList))
	}

	resp = signedRequestWithService(t, http.MethodGet, ts.URL+"/v20180820/accesspoint?bucket=ap-bucket&maxResults=1", nil, headers, "s3-control")
	assertStatus(t, resp, http.StatusOK)
	var listResp s3ControlAccessPointListResult
	if err := xml.Unmarshal(mustBody(t, resp), &listResp); err != nil {
		t.Fatalf("parse list access points response: %v", err)
	}
	if len(listResp.AccessPointList) != 1 {
		t.Fatalf("expected 1 access point, got %d", len(listResp.AccessPointList))
	}
	if listResp.NextToken == "" {
		t.Fatalf("expected next token for truncated response")
	}

	resp = signedRequestWithService(t, http.MethodGet, ts.URL+"/v20180820/accesspoint?bucket=ap-bucket&maxResults=1&nextToken="+listResp.NextToken, nil, headers, "s3-control")
	assertStatus(t, resp, http.StatusOK)
	var listResp2 s3ControlAccessPointListResult
	if err := xml.Unmarshal(mustBody(t, resp), &listResp2); err != nil {
		t.Fatalf("parse list access points response: %v", err)
	}
	if len(listResp2.AccessPointList) != 1 {
		t.Fatalf("expected 1 access point on second page, got %d", len(listResp2.AccessPointList))
	}
	if listResp2.NextToken != "" {
		t.Fatalf("expected no next token on final page")
	}

	policyJSON := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":"*","Action":"s3:GetObject","Resource":"*"}]}`
	var policyBuf bytes.Buffer
	policyBuf.WriteString(`<PutAccessPointPolicyRequest xmlns="http://awss3control.amazonaws.com/doc/2018-08-20/"><Policy>`)
	if err := xml.EscapeText(&policyBuf, []byte(policyJSON)); err != nil {
		t.Fatalf("escape policy: %v", err)
	}
	policyBuf.WriteString(`</Policy></PutAccessPointPolicyRequest>`)
	resp = signedRequestWithService(t, http.MethodPut, ts.URL+"/v20180820/accesspoint/ap-one/policy", policyBuf.Bytes(), headers, "s3-control")
	assertStatus(t, resp, http.StatusOK)

	resp = signedRequestWithService(t, http.MethodGet, ts.URL+"/v20180820/accesspoint/ap-one/policy", nil, headers, "s3-control")
	assertStatus(t, resp, http.StatusOK)
	var policyResp s3ControlGetAccessPointPolicyResult
	if err := xml.Unmarshal(mustBody(t, resp), &policyResp); err != nil {
		t.Fatalf("parse get policy response: %v", err)
	}
	if strings.TrimSpace(policyResp.Policy) != policyJSON {
		t.Fatalf("unexpected policy response: %q", policyResp.Policy)
	}

	resp = signedRequestWithService(t, http.MethodGet, ts.URL+"/v20180820/accesspoint/ap-one/policyStatus", nil, headers, "s3-control")
	assertStatus(t, resp, http.StatusOK)
	var statusResp s3ControlGetAccessPointPolicyStatusResult
	if err := xml.Unmarshal(mustBody(t, resp), &statusResp); err != nil {
		t.Fatalf("parse policy status response: %v", err)
	}
	if !statusResp.PolicyStatus.IsPublic {
		t.Fatalf("expected public policy status")
	}

	resp = signedRequestWithService(t, http.MethodDelete, ts.URL+"/v20180820/accesspoint/ap-one/policy", nil, headers, "s3-control")
	assertStatus(t, resp, http.StatusOK)

	resp = signedRequestWithService(t, http.MethodDelete, ts.URL+"/v20180820/accesspoint/ap-one/scope", nil, headers, "s3-control")
	assertStatus(t, resp, http.StatusOK)

	resp = signedRequestWithService(t, http.MethodGet, ts.URL+"/v20180820/accesspoint/ap-one/scope", nil, headers, "s3-control")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for missing scope, got %d", resp.StatusCode)
	}
	assertS3ErrorCode(t, resp, "NoSuchAccessPointScope")

	resp = signedRequestWithService(t, http.MethodGet, ts.URL+"/v20180820/accesspoint/ap-one/policy", nil, headers, "s3-control")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for missing policy, got %d", resp.StatusCode)
	}
	assertS3ErrorCode(t, resp, "NoSuchAccessPointPolicy")
}
