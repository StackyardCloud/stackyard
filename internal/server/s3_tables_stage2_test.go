package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestS3TablesStage2CreateGetDeleteTable(t *testing.T) {
	srv := New(Config{
		Addr:      "127.0.0.1:0",
		AccessKey: testAccessKey,
		SecretKey: testSecretKey,
		LogLevel:  "error",
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	arn := "arn:aws:s3tables:us-east-1:123456789012:bucket/demo-bucket"
	createURL := ts.URL + "/tables/" + url.PathEscape(arn)
	reqBody := []byte(`{"namespace":["analytics"],"name":"metrics"}`)
	resp := signedRequestWithService(t, http.MethodPost, createURL, reqBody, map[string]string{
		"Content-Type": "application/json",
	}, "s3tables")
	assertStatus(t, resp, http.StatusOK)

	var createResp s3TablesCreateTableResponse
	if err := json.Unmarshal(mustBody(t, resp), &createResp); err != nil {
		t.Fatalf("parse create response: %v", err)
	}
	if createResp.Name != "metrics" {
		t.Fatalf("unexpected table name %q", createResp.Name)
	}

	getURL := createURL + "?namespace=analytics&name=metrics"
	resp = signedRequestWithService(t, http.MethodGet, getURL, nil, nil, "s3tables")
	assertStatus(t, resp, http.StatusOK)
	var getResp s3TablesGetTableResponse
	if err := json.Unmarshal(mustBody(t, resp), &getResp); err != nil {
		t.Fatalf("parse get response: %v", err)
	}
	if getResp.Format != "ICEBERG" {
		t.Fatalf("unexpected table format %q", getResp.Format)
	}

	delURL := createURL + "?namespace=analytics&name=metrics"
	resp = signedRequestWithService(t, http.MethodDelete, delURL, nil, nil, "s3tables")
	assertStatus(t, resp, http.StatusOK)

	resp = signedRequestWithService(t, http.MethodGet, getURL, nil, nil, "s3tables")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 after delete, got %d", resp.StatusCode)
	}
}

func TestS3TablesStage2ListTablesPagination(t *testing.T) {
	srv := New(Config{
		Addr:      "127.0.0.1:0",
		AccessKey: testAccessKey,
		SecretKey: testSecretKey,
		LogLevel:  "error",
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	arn := "arn:aws:s3tables:us-east-1:123456789012:bucket/demo-bucket"
	createURL := ts.URL + "/tables/" + url.PathEscape(arn)
	for _, name := range []string{"alpha", "beta"} {
		body := []byte(`{"namespace":["analytics"],"name":"` + name + `"}`)
		resp := signedRequestWithService(t, http.MethodPost, createURL, body, map[string]string{
			"Content-Type": "application/json",
		}, "s3tables")
		assertStatus(t, resp, http.StatusOK)
	}

	listURL := createURL + "?namespace=analytics&maxTables=1"
	resp := signedRequestWithService(t, http.MethodGet, listURL, nil, nil, "s3tables")
	assertStatus(t, resp, http.StatusOK)
	var page1 s3TablesListTablesResponse
	if err := json.Unmarshal(mustBody(t, resp), &page1); err != nil {
		t.Fatalf("parse page1: %v", err)
	}
	if len(page1.Tables) != 1 || page1.ContinuationToken == "" {
		t.Fatalf("expected 1 table and continuation token, got %d token=%q", len(page1.Tables), page1.ContinuationToken)
	}

	listURL = createURL + "?namespace=analytics&maxTables=1&continuationToken=" + url.QueryEscape(page1.ContinuationToken)
	resp = signedRequestWithService(t, http.MethodGet, listURL, nil, nil, "s3tables")
	assertStatus(t, resp, http.StatusOK)
	var page2 s3TablesListTablesResponse
	if err := json.Unmarshal(mustBody(t, resp), &page2); err != nil {
		t.Fatalf("parse page2: %v", err)
	}
	if len(page2.Tables) != 1 {
		t.Fatalf("expected 1 table on page2, got %d", len(page2.Tables))
	}
}

func TestS3TablesStage2RenameAndMetadata(t *testing.T) {
	srv := New(Config{
		Addr:      "127.0.0.1:0",
		AccessKey: testAccessKey,
		SecretKey: testSecretKey,
		LogLevel:  "error",
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	arn := "arn:aws:s3tables:us-east-1:123456789012:bucket/demo-bucket"
	baseURL := ts.URL + "/tables/" + url.PathEscape(arn)
	createBody := []byte(`{"namespace":["analytics"],"name":"rename-me"}`)
	resp := signedRequestWithService(t, http.MethodPost, baseURL, createBody, map[string]string{
		"Content-Type": "application/json",
	}, "s3tables")
	assertStatus(t, resp, http.StatusOK)

	renameURL := baseURL + "/rename"
	renameBody := []byte(`{"namespace":["analytics"],"name":"rename-me","newName":"renamed"}`)
	resp = signedRequestWithService(t, http.MethodPost, renameURL, renameBody, map[string]string{
		"Content-Type": "application/json",
	}, "s3tables")
	assertStatus(t, resp, http.StatusOK)
	var renameResp s3TablesRenameTableResponse
	if err := json.Unmarshal(mustBody(t, resp), &renameResp); err != nil {
		t.Fatalf("parse rename response: %v", err)
	}
	if renameResp.Name != "renamed" {
		t.Fatalf("unexpected rename result %q", renameResp.Name)
	}

	metaURL := baseURL + "/metadata?namespace=analytics&name=renamed"
	resp = signedRequestWithService(t, http.MethodGet, metaURL, nil, nil, "s3tables")
	assertStatus(t, resp, http.StatusOK)
	var metaResp s3TablesGetTableMetadataLocationResponse
	if err := json.Unmarshal(mustBody(t, resp), &metaResp); err != nil {
		t.Fatalf("parse metadata response: %v", err)
	}
	if metaResp.MetadataLocation == "" {
		t.Fatalf("expected metadata location")
	}

	updateBody := []byte(`{"namespace":["analytics"],"name":"renamed","metadataLocation":"s3://demo-bucket/analytics/renamed/metadata-v2.json"}`)
	resp = signedRequestWithService(t, http.MethodPut, baseURL+"/metadata", updateBody, map[string]string{
		"Content-Type": "application/json",
	}, "s3tables")
	assertStatus(t, resp, http.StatusOK)

	resp = signedRequestWithService(t, http.MethodGet, metaURL, nil, nil, "s3tables")
	assertStatus(t, resp, http.StatusOK)
	if err := json.Unmarshal(mustBody(t, resp), &metaResp); err != nil {
		t.Fatalf("parse metadata response: %v", err)
	}
	if metaResp.MetadataLocation != "s3://demo-bucket/analytics/renamed/metadata-v2.json" {
		t.Fatalf("unexpected metadata location %q", metaResp.MetadataLocation)
	}
}
