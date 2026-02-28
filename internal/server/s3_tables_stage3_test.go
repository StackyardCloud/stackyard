package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestS3TablesStage3TaggingTableBucket(t *testing.T) {
	srv := New(Config{
		Addr:      "127.0.0.1:0",
		AccessKey: testAccessKey,
		SecretKey: testSecretKey,
		LogLevel:  "error",
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	bucketArn := "arn:aws:s3tables:us-east-1:123456789012:bucket/demo-bucket"
	tagsURL := ts.URL + "/tags/" + url.PathEscape(bucketArn)
	body := []byte(`{"tags":[{"key":"env","value":"dev"},{"key":"team","value":"core"}]}`)
	resp := signedRequestWithService(t, http.MethodPost, tagsURL, body, map[string]string{
		"Content-Type": "application/json",
	}, "s3tables")
	assertStatus(t, resp, http.StatusOK)

	resp = signedRequestWithService(t, http.MethodGet, tagsURL, nil, nil, "s3tables")
	assertStatus(t, resp, http.StatusOK)
	var list s3TablesListTagsForResourceResponse
	if err := json.Unmarshal(mustBody(t, resp), &list); err != nil {
		t.Fatalf("parse list tags: %v", err)
	}
	if len(list.Tags) != 2 {
		t.Fatalf("expected 2 tags, got %d", len(list.Tags))
	}

	resp = signedRequestWithService(t, http.MethodDelete, tagsURL+"?tagKeys=env", nil, nil, "s3tables")
	assertStatus(t, resp, http.StatusOK)

	resp = signedRequestWithService(t, http.MethodGet, tagsURL, nil, nil, "s3tables")
	assertStatus(t, resp, http.StatusOK)
	if err := json.Unmarshal(mustBody(t, resp), &list); err != nil {
		t.Fatalf("parse list tags: %v", err)
	}
	if len(list.Tags) != 1 || list.Tags[0].Key != "team" {
		t.Fatalf("unexpected tags after delete: %+v", list.Tags)
	}
}

func TestS3TablesStage3TaggingTable(t *testing.T) {
	srv := New(Config{
		Addr:      "127.0.0.1:0",
		AccessKey: testAccessKey,
		SecretKey: testSecretKey,
		LogLevel:  "error",
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	bucketArn := "arn:aws:s3tables:us-east-1:123456789012:bucket/demo-bucket"
	createURL := ts.URL + "/tables/" + url.PathEscape(bucketArn)
	resp := signedRequestWithService(t, http.MethodPost, createURL, []byte(`{"namespace":["analytics"],"name":"tagged"}`), map[string]string{
		"Content-Type": "application/json",
	}, "s3tables")
	assertStatus(t, resp, http.StatusOK)
	var createResp s3TablesCreateTableResponse
	if err := json.Unmarshal(mustBody(t, resp), &createResp); err != nil {
		t.Fatalf("parse create response: %v", err)
	}

	tagsURL := ts.URL + "/tags/" + url.PathEscape(createResp.TableArn)
	resp = signedRequestWithService(t, http.MethodPost, tagsURL, []byte(`{"tags":[{"key":"owner","value":"data"}]}`), map[string]string{
		"Content-Type": "application/json",
	}, "s3tables")
	assertStatus(t, resp, http.StatusOK)

	resp = signedRequestWithService(t, http.MethodGet, tagsURL, nil, nil, "s3tables")
	assertStatus(t, resp, http.StatusOK)
	var list s3TablesListTagsForResourceResponse
	if err := json.Unmarshal(mustBody(t, resp), &list); err != nil {
		t.Fatalf("parse list tags: %v", err)
	}
	if len(list.Tags) != 1 || list.Tags[0].Key != "owner" {
		t.Fatalf("unexpected tags: %+v", list.Tags)
	}
}

func TestS3TablesStage3TablePolicy(t *testing.T) {
	srv := New(Config{
		Addr:      "127.0.0.1:0",
		AccessKey: testAccessKey,
		SecretKey: testSecretKey,
		LogLevel:  "error",
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	bucketArn := "arn:aws:s3tables:us-east-1:123456789012:bucket/demo-bucket"
	createURL := ts.URL + "/tables/" + url.PathEscape(bucketArn)
	resp := signedRequestWithService(t, http.MethodPost, createURL, []byte(`{"namespace":["analytics"],"name":"policy"}`), map[string]string{
		"Content-Type": "application/json",
	}, "s3tables")
	assertStatus(t, resp, http.StatusOK)
	var createResp s3TablesCreateTableResponse
	if err := json.Unmarshal(mustBody(t, resp), &createResp); err != nil {
		t.Fatalf("parse create response: %v", err)
	}

	policyURL := ts.URL + "/tables/" + url.PathEscape(createResp.TableArn) + "/policy"
	resp = signedRequestWithService(t, http.MethodPut, policyURL, []byte(`{"policy":"{}"}`), map[string]string{
		"Content-Type": "application/json",
	}, "s3tables")
	assertStatus(t, resp, http.StatusOK)

	resp = signedRequestWithService(t, http.MethodGet, policyURL, nil, nil, "s3tables")
	assertStatus(t, resp, http.StatusOK)
	var getResp s3TablesGetPolicyResponse
	if err := json.Unmarshal(mustBody(t, resp), &getResp); err != nil {
		t.Fatalf("parse get policy: %v", err)
	}
	if getResp.Policy != "{}" {
		t.Fatalf("unexpected policy %q", getResp.Policy)
	}

	resp = signedRequestWithService(t, http.MethodDelete, policyURL, nil, nil, "s3tables")
	assertStatus(t, resp, http.StatusOK)

	resp = signedRequestWithService(t, http.MethodGet, policyURL, nil, nil, "s3tables")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 after delete, got %d", resp.StatusCode)
	}
}

func TestS3TablesStage3TableBucketPolicy(t *testing.T) {
	srv := New(Config{
		Addr:      "127.0.0.1:0",
		AccessKey: testAccessKey,
		SecretKey: testSecretKey,
		LogLevel:  "error",
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	bucketArn := "arn:aws:s3tables:us-east-1:123456789012:bucket/demo-bucket"
	policyURL := ts.URL + "/buckets/" + url.PathEscape(bucketArn) + "/policy"
	resp := signedRequestWithService(t, http.MethodPut, policyURL, []byte(`{"policy":"{}"}`), map[string]string{
		"Content-Type": "application/json",
	}, "s3tables")
	assertStatus(t, resp, http.StatusOK)

	resp = signedRequestWithService(t, http.MethodGet, policyURL, nil, nil, "s3tables")
	assertStatus(t, resp, http.StatusOK)
	var getResp s3TablesGetPolicyResponse
	if err := json.Unmarshal(mustBody(t, resp), &getResp); err != nil {
		t.Fatalf("parse get policy: %v", err)
	}
	if getResp.Policy != "{}" {
		t.Fatalf("unexpected policy %q", getResp.Policy)
	}

	resp = signedRequestWithService(t, http.MethodDelete, policyURL, nil, nil, "s3tables")
	assertStatus(t, resp, http.StatusOK)
}
