package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestS3TablesStage9IcebergFlow(t *testing.T) {
	srv := New(Config{
		Addr:      "127.0.0.1:0",
		AccessKey: testAccessKey,
		SecretKey: testSecretKey,
		LogLevel:  "error",
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	configURL := ts.URL + "/iceberg/v1/config"
	resp := signedRequestWithService(t, http.MethodGet, configURL, nil, nil, "s3tables")
	assertStatus(t, resp, http.StatusOK)
	var cfg icebergConfigResponse
	if err := json.Unmarshal(mustBody(t, resp), &cfg); err != nil {
		t.Fatalf("parse config: %v", err)
	}
	if cfg.Defaults == nil || cfg.Defaults["catalog"] == "" {
		t.Fatalf("expected defaults catalog")
	}

	bucketArn := "arn:aws:s3tables:us-east-1:123456789012:bucket/demo-bucket"
	prefix := url.PathEscape(bucketArn)
	base := ts.URL + "/iceberg/v1/" + prefix

	listNS := base + "/namespaces"
	resp = signedRequestWithService(t, http.MethodGet, listNS, nil, nil, "s3tables")
	assertStatus(t, resp, http.StatusOK)
	var listResp icebergListNamespacesResponse
	if err := json.Unmarshal(mustBody(t, resp), &listResp); err != nil {
		t.Fatalf("parse list namespaces: %v", err)
	}
	if len(listResp.Namespaces) == 0 {
		t.Fatalf("expected namespaces")
	}

	resp = signedRequestWithService(t, http.MethodPost, listNS, []byte(`{"namespace":["iceberg"]}`), map[string]string{
		"Content-Type": "application/json",
	}, "s3tables")
	assertStatus(t, resp, http.StatusOK)

	nsURL := base + "/namespaces/iceberg"
	resp = signedRequestWithService(t, http.MethodHead, nsURL, nil, nil, "s3tables")
	assertStatus(t, resp, http.StatusOK)

	tablesURL := base + "/namespaces/iceberg/tables"
	resp = signedRequestWithService(t, http.MethodPost, tablesURL, []byte(`{"name":"events"}`), map[string]string{
		"Content-Type": "application/json",
	}, "s3tables")
	assertStatus(t, resp, http.StatusOK)
	var createResp icebergCreateTableResponse
	if err := json.Unmarshal(mustBody(t, resp), &createResp); err != nil {
		t.Fatalf("parse create table: %v", err)
	}
	if createResp.TableIdentifier.Name != "events" || createResp.MetadataLocation == "" {
		t.Fatalf("unexpected create response: %#v", createResp)
	}

	resp = signedRequestWithService(t, http.MethodGet, tablesURL, nil, nil, "s3tables")
	assertStatus(t, resp, http.StatusOK)
	var listTables icebergListTablesResponse
	if err := json.Unmarshal(mustBody(t, resp), &listTables); err != nil {
		t.Fatalf("parse list tables: %v", err)
	}
	if len(listTables.Identifiers) == 0 {
		t.Fatalf("expected tables")
	}

	tableURL := base + "/namespaces/iceberg/tables/events"
	resp = signedRequestWithService(t, http.MethodGet, tableURL, nil, nil, "s3tables")
	assertStatus(t, resp, http.StatusOK)
	var getResp icebergCreateTableResponse
	if err := json.Unmarshal(mustBody(t, resp), &getResp); err != nil {
		t.Fatalf("parse get table: %v", err)
	}
	if getResp.MetadataLocation == "" {
		t.Fatalf("expected metadata location")
	}

	updateLoc := "s3://demo-bucket/iceberg/events/metadata-v2.json"
	resp = signedRequestWithService(t, http.MethodPost, tableURL, []byte(`{"metadataLocation":"`+updateLoc+`"}`), map[string]string{
		"Content-Type": "application/json",
	}, "s3tables")
	assertStatus(t, resp, http.StatusOK)
	if err := json.Unmarshal(mustBody(t, resp), &getResp); err != nil {
		t.Fatalf("parse update response: %v", err)
	}
	if getResp.MetadataLocation != updateLoc {
		t.Fatalf("unexpected metadata location %q", getResp.MetadataLocation)
	}

	renameURL := base + "/tables/rename"
	resp = signedRequestWithService(t, http.MethodPost, renameURL, []byte(`{"source":{"namespace":["iceberg"],"name":"events"},"destination":{"namespace":["iceberg"],"name":"renamed"}}`), map[string]string{
		"Content-Type": "application/json",
	}, "s3tables")
	assertStatus(t, resp, http.StatusOK)
	if err := json.Unmarshal(mustBody(t, resp), &getResp); err != nil {
		t.Fatalf("parse rename response: %v", err)
	}
	if getResp.TableIdentifier.Name != "renamed" {
		t.Fatalf("expected renamed table, got %q", getResp.TableIdentifier.Name)
	}

	resp = signedRequestWithService(t, http.MethodDelete, base+"/namespaces/iceberg/tables/renamed?purge=true", nil, nil, "s3tables")
	assertStatus(t, resp, http.StatusOK)

	resp = signedRequestWithService(t, http.MethodDelete, nsURL, nil, nil, "s3tables")
	assertStatus(t, resp, http.StatusOK)
}

func TestS3TablesStage9IcebergDeleteRequiresPurge(t *testing.T) {
	srv := New(Config{
		Addr:      "127.0.0.1:0",
		AccessKey: testAccessKey,
		SecretKey: testSecretKey,
		LogLevel:  "error",
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	bucketArn := "arn:aws:s3tables:us-east-1:123456789012:bucket/demo-bucket"
	base := ts.URL + "/iceberg/v1/" + url.PathEscape(bucketArn)

	_ = signedRequestWithService(t, http.MethodPost, base+"/namespaces", []byte(`{"namespace":["purge"]}`), map[string]string{
		"Content-Type": "application/json",
	}, "s3tables")
	_ = signedRequestWithService(t, http.MethodPost, base+"/namespaces/purge/tables", []byte(`{"name":"t1"}`), map[string]string{
		"Content-Type": "application/json",
	}, "s3tables")

	resp := signedRequestWithService(t, http.MethodDelete, base+"/namespaces/purge/tables/t1?purge=false", nil, nil, "s3tables")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid purge, got %d", resp.StatusCode)
	}
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "purge") {
		t.Fatalf("expected purge error, got %s", body)
	}
}
