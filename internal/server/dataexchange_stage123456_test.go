package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestDataExchangeStage123456LifecycleAndReadSurface(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	grantARN := url.QueryEscape("arn:aws:dataexchange:us-east-1:123456789012:data-grants/dg-000001")
	resourceARN := url.QueryEscape("arn:aws:dataexchange:us-east-1:123456789012:data-sets/ds-000001")

	cases := []struct {
		name    string
		method  string
		path    string
		payload string
	}{
		{name: "CreateDataSet", method: http.MethodPost, path: "/v1/data-sets", payload: `{"Name":"stage-data-set"}`},
		{name: "UpdateDataSet", method: http.MethodPatch, path: "/v1/data-sets/ds-000001", payload: `{"Description":"updated"}`},
		{name: "GetDataSet", method: http.MethodGet, path: "/v1/data-sets/ds-000001", payload: ``},
		{name: "CreateRevision", method: http.MethodPost, path: "/v1/data-sets/ds-000001/revisions", payload: `{"Comment":"stage-revision"}`},
		{name: "UpdateRevision", method: http.MethodPatch, path: "/v1/data-sets/ds-000001/revisions/rev-000001", payload: `{"Comment":"updated"}`},
		{name: "GetRevision", method: http.MethodGet, path: "/v1/data-sets/ds-000001/revisions/rev-000001", payload: ``},
		{name: "GetAsset", method: http.MethodGet, path: "/v1/data-sets/ds-000001/revisions/rev-000001/assets/asset-000001", payload: ``},
		{name: "UpdateAsset", method: http.MethodPatch, path: "/v1/data-sets/ds-000001/revisions/rev-000001/assets/asset-000001", payload: `{"Name":"updated-asset"}`},
		{name: "CreateJob", method: http.MethodPost, path: "/v1/jobs", payload: `{"Type":"IMPORT_ASSETS_FROM_S3"}`},
		{name: "StartJob", method: http.MethodPatch, path: "/v1/jobs/job-000001", payload: `{"State":"IN_PROGRESS"}`},
		{name: "GetJob", method: http.MethodGet, path: "/v1/jobs/job-000001", payload: ``},
		{name: "CancelJob", method: http.MethodDelete, path: "/v1/jobs/job-000001", payload: ``},
		{name: "CreateEventAction", method: http.MethodPost, path: "/v1/event-actions", payload: `{"Name":"stage-action"}`},
		{name: "UpdateEventAction", method: http.MethodPatch, path: "/v1/event-actions/ea-000001", payload: `{"Name":"updated-action"}`},
		{name: "GetEventAction", method: http.MethodGet, path: "/v1/event-actions/ea-000001", payload: ``},
		{name: "CreateDataGrant", method: http.MethodPost, path: "/v1/data-grants", payload: `{}`},
		{name: "GetDataGrant", method: http.MethodGet, path: "/v1/data-grants/dg-000001", payload: ``},
		{name: "AcceptDataGrant", method: http.MethodPost, path: "/v1/data-grants/" + grantARN + "/accept", payload: `{}`},
		{name: "GetReceivedDataGrant", method: http.MethodGet, path: "/v1/received-data-grants/" + grantARN, payload: ``},
		{name: "ListDataSets", method: http.MethodGet, path: "/v1/data-sets?maxResults=10&origin=OWNED", payload: ``},
		{name: "ListDataSetRevisions", method: http.MethodGet, path: "/v1/data-sets/ds-000001/revisions?maxResults=10", payload: ``},
		{name: "ListRevisionAssets", method: http.MethodGet, path: "/v1/data-sets/ds-000001/revisions/rev-000001/assets?maxResults=10", payload: ``},
		{name: "ListJobs", method: http.MethodGet, path: "/v1/jobs?dataSetId=ds-000001&revisionId=rev-000001", payload: ``},
		{name: "ListEventActions", method: http.MethodGet, path: "/v1/event-actions?eventSourceId=evt-src-000001", payload: ``},
		{name: "ListDataGrants", method: http.MethodGet, path: "/v1/data-grants?maxResults=10", payload: ``},
		{name: "ListReceivedDataGrants", method: http.MethodGet, path: "/v1/received-data-grants?acceptanceState=ACCEPTED", payload: ``},
		{name: "SendDataSetNotification", method: http.MethodPost, path: "/v1/data-sets/ds-000001/notification", payload: `{"Comment":"notify"}`},
		{name: "SendApiAsset", method: http.MethodPost, path: "/v1?assetId=asset-000001", payload: `{}`},
		{name: "TagResource", method: http.MethodPost, path: "/tags/" + resourceARN, payload: `{"Tags":{"env":"stage","owner":"tests"}}`},
		{name: "ListTagsForResource", method: http.MethodGet, path: "/tags/" + resourceARN, payload: ``},
		{name: "UntagResource", method: http.MethodDelete, path: "/tags/" + resourceARN + "?tagKeys=env", payload: ``},
	}

	for _, tc := range cases {
		resp := dataExchangeRequest(t, ts, tc.method, tc.path, tc.payload)
		body := string(mustBody(t, resp))
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("%s: expected 200, got %d: %s", tc.name, resp.StatusCode, body)
		}
		if strings.Contains(body, "NotImplemented") {
			t.Fatalf("%s: expected non-NotImplemented response, got: %s", tc.name, body)
		}
	}
}

func TestDataExchangeStage6ValidationAndIdempotency(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := dataExchangeRequest(t, ts, http.MethodPost, "/unknown-dataexchange-route", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException for unknown route, got %q", body)
	}

	resp = signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/v1/data-sets",
		[]byte(`{"broken":`),
		map[string]string{"Content-Type": "application/json"},
		"dataexchange",
	)
	assertStatus(t, resp, http.StatusBadRequest)
	body = string(mustBody(t, resp))
	if !strings.Contains(body, "invalid JSON body") {
		t.Fatalf("expected invalid JSON body validation error, got %q", body)
	}

	resp = dataExchangeRequest(t, ts, http.MethodDelete, "/v1/jobs/job-000001", "")
	assertStatus(t, resp, http.StatusOK)
	resp = dataExchangeRequest(t, ts, http.MethodDelete, "/v1/jobs/job-000001", "")
	assertStatus(t, resp, http.StatusOK)
}
