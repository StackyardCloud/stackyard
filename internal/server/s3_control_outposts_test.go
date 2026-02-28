package server

import (
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestS3ControlOutpostsStage8(t *testing.T) {
	srv := New(Config{
		Addr:      "127.0.0.1:0",
		AccessKey: testAccessKey,
		SecretKey: testSecretKey,
		LogLevel:  "error",
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	accountID := "123456789012"
	outpostID := "op-1234"
	headers := map[string]string{
		"x-amz-account-id": accountID,
		"Content-Type":     "application/xml",
	}

	createBody := `<?xml version="1.0" encoding="UTF-8"?>` +
		`<CreateBucketRequest xmlns="http://awss3control.amazonaws.com/doc/2018-08-20/">` +
		`<Bucket>outposts-demo</Bucket>` +
		`<Tagging><Tag><Key>env</Key><Value>dev</Value></Tag></Tagging>` +
		`</CreateBucketRequest>`
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/v20180820/outposts/"+outpostID+"/bucket", []byte(createBody), headers, "s3-control")
	assertStatus(t, resp, http.StatusOK)
	var createResp s3ControlCreateOutpostsBucketResult
	if err := xml.Unmarshal(mustBody(t, resp), &createResp); err != nil {
		t.Fatalf("parse create outposts bucket response: %v", err)
	}
	if createResp.BucketName != "outposts-demo" {
		t.Fatalf("unexpected bucket name")
	}

	resp = signedRequestWithService(t, http.MethodGet, ts.URL+"/v20180820/outposts/"+outpostID+"/bucket/outposts-demo", nil, headers, "s3-control")
	assertStatus(t, resp, http.StatusOK)
	var getResp s3ControlGetOutpostsBucketResult
	if err := xml.Unmarshal(mustBody(t, resp), &getResp); err != nil {
		t.Fatalf("parse get outposts bucket response: %v", err)
	}
	if getResp.BucketName != "outposts-demo" {
		t.Fatalf("unexpected bucket name in get")
	}

	resp = signedRequestWithService(t, http.MethodGet, ts.URL+"/v20180820/outposts/"+outpostID+"/buckets", nil, headers, "s3-control")
	assertStatus(t, resp, http.StatusOK)
	var listResp s3ControlListRegionalBucketsResult
	if err := xml.Unmarshal(mustBody(t, resp), &listResp); err != nil {
		t.Fatalf("parse list regional buckets response: %v", err)
	}
	if len(listResp.RegionalBuckets) != 1 {
		t.Fatalf("expected 1 regional bucket, got %d", len(listResp.RegionalBuckets))
	}

	taggingBody := `<?xml version="1.0" encoding="UTF-8"?>` +
		`<Tagging xmlns="http://awss3control.amazonaws.com/doc/2018-08-20/">` +
		`<Tag><Key>team</Key><Value>core</Value></Tag>` +
		`</Tagging>`
	resp = signedRequestWithService(t, http.MethodPut, ts.URL+"/v20180820/outposts/"+outpostID+"/bucket/outposts-demo/tagging", []byte(taggingBody), headers, "s3-control")
	assertStatus(t, resp, http.StatusOK)

	resp = signedRequestWithService(t, http.MethodGet, ts.URL+"/v20180820/outposts/"+outpostID+"/bucket/outposts-demo/tagging", nil, headers, "s3-control")
	assertStatus(t, resp, http.StatusOK)
	var getTagResp s3ControlGetOutpostsBucketTaggingResult
	if err := xml.Unmarshal(mustBody(t, resp), &getTagResp); err != nil {
		t.Fatalf("parse get outposts bucket tagging response: %v", err)
	}
	if len(getTagResp.Tagging.Tags) == 0 {
		t.Fatalf("expected tags")
	}

	resp = signedRequestWithService(t, http.MethodDelete, ts.URL+"/v20180820/outposts/"+outpostID+"/bucket/outposts-demo/tagging", nil, headers, "s3-control")
	assertStatus(t, resp, http.StatusOK)

	resp = signedRequestWithService(t, http.MethodGet, ts.URL+"/v20180820/outposts/"+outpostID+"/bucket/outposts-demo/tagging", nil, headers, "s3-control")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 after delete tagging, got %d", resp.StatusCode)
	}

	lifecycleBody := `<?xml version="1.0" encoding="UTF-8"?>` +
		`<LifecycleConfiguration xmlns="http://awss3control.amazonaws.com/doc/2018-08-20/">` +
		`<Rule><ID>rule-1</ID><Status>Enabled</Status></Rule>` +
		`</LifecycleConfiguration>`
	resp = signedRequestWithService(t, http.MethodPut, ts.URL+"/v20180820/outposts/"+outpostID+"/bucket/outposts-demo/lifecycle", []byte(lifecycleBody), headers, "s3-control")
	assertStatus(t, resp, http.StatusOK)

	resp = signedRequestWithService(t, http.MethodGet, ts.URL+"/v20180820/outposts/"+outpostID+"/bucket/outposts-demo/lifecycle", nil, headers, "s3-control")
	assertStatus(t, resp, http.StatusOK)
	var getLifecycle s3ControlGetOutpostsBucketLifecycleResult
	if err := xml.Unmarshal(mustBody(t, resp), &getLifecycle); err != nil {
		t.Fatalf("parse get lifecycle response: %v", err)
	}

	resp = signedRequestWithService(t, http.MethodDelete, ts.URL+"/v20180820/outposts/"+outpostID+"/bucket/outposts-demo/lifecycle", nil, headers, "s3-control")
	assertStatus(t, resp, http.StatusOK)

	resp = signedRequestWithService(t, http.MethodGet, ts.URL+"/v20180820/outposts/"+outpostID+"/bucket/outposts-demo/lifecycle", nil, headers, "s3-control")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 after delete lifecycle, got %d", resp.StatusCode)
	}

	policyBody := `<?xml version="1.0" encoding="UTF-8"?>` +
		`<PutBucketPolicyRequest xmlns="http://awss3control.amazonaws.com/doc/2018-08-20/">` +
		`<Policy>{"Version":"2012-10-17","Statement":[]}</Policy>` +
		`</PutBucketPolicyRequest>`
	resp = signedRequestWithService(t, http.MethodPut, ts.URL+"/v20180820/outposts/"+outpostID+"/bucket/outposts-demo/policy", []byte(policyBody), headers, "s3-control")
	assertStatus(t, resp, http.StatusOK)

	resp = signedRequestWithService(t, http.MethodGet, ts.URL+"/v20180820/outposts/"+outpostID+"/bucket/outposts-demo/policy", nil, headers, "s3-control")
	assertStatus(t, resp, http.StatusOK)
	var getPolicy s3ControlGetOutpostsBucketPolicyResult
	if err := xml.Unmarshal(mustBody(t, resp), &getPolicy); err != nil {
		t.Fatalf("parse get policy response: %v", err)
	}
	if getPolicy.Policy == "" {
		t.Fatalf("expected policy body")
	}

	resp = signedRequestWithService(t, http.MethodDelete, ts.URL+"/v20180820/outposts/"+outpostID+"/bucket/outposts-demo/policy", nil, headers, "s3-control")
	assertStatus(t, resp, http.StatusOK)

	replicationBody := `<?xml version="1.0" encoding="UTF-8"?>` +
		`<ReplicationConfiguration xmlns="http://awss3control.amazonaws.com/doc/2018-08-20/">` +
		`<Role>arn:aws:iam::123456789012:role/replication</Role>` +
		`</ReplicationConfiguration>`
	resp = signedRequestWithService(t, http.MethodPut, ts.URL+"/v20180820/outposts/"+outpostID+"/bucket/outposts-demo/replication", []byte(replicationBody), headers, "s3-control")
	assertStatus(t, resp, http.StatusOK)

	resp = signedRequestWithService(t, http.MethodGet, ts.URL+"/v20180820/outposts/"+outpostID+"/bucket/outposts-demo/replication", nil, headers, "s3-control")
	assertStatus(t, resp, http.StatusOK)
	var getReplication s3ControlGetOutpostsBucketReplicationResult
	if err := xml.Unmarshal(mustBody(t, resp), &getReplication); err != nil {
		t.Fatalf("parse get replication response: %v", err)
	}

	resp = signedRequestWithService(t, http.MethodDelete, ts.URL+"/v20180820/outposts/"+outpostID+"/bucket/outposts-demo/replication", nil, headers, "s3-control")
	assertStatus(t, resp, http.StatusOK)

	resp = signedRequestWithService(t, http.MethodGet, ts.URL+"/v20180820/outposts/"+outpostID+"/bucket/outposts-demo/replication", nil, headers, "s3-control")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 after delete replication, got %d", resp.StatusCode)
	}

	resp = signedRequestWithService(t, http.MethodDelete, ts.URL+"/v20180820/outposts/"+outpostID+"/bucket/outposts-demo", nil, headers, "s3-control")
	assertStatus(t, resp, http.StatusOK)

	resp = signedRequestWithService(t, http.MethodGet, ts.URL+"/v20180820/outposts/"+outpostID+"/bucket/outposts-demo", nil, headers, "s3-control")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 after delete bucket, got %d", resp.StatusCode)
	}
}
