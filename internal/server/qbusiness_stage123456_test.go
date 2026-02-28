package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestQBusinessStage123456CoreLifecycleAndReadSurface(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	cases := []struct {
		name    string
		method  string
		path    string
		payload string
	}{
		{name: "CreateApplication", method: http.MethodPost, path: "/applications", payload: `{"displayName":"stage-qbusiness-app"}`},
		{name: "CreateIndex", method: http.MethodPost, path: "/applications/app-000001/indices", payload: `{"displayName":"stage-index"}`},
		{name: "CreateDataSource", method: http.MethodPost, path: "/applications/app-000001/indices/idx-000001/datasources", payload: `{"displayName":"stage-ds"}`},
		{name: "BatchPutDocument", method: http.MethodPost, path: "/applications/app-000001/indices/idx-000001/documents", payload: `{"documents":[{"documentId":"doc-stage-1"}]}`},
		{name: "StartDataSourceSyncJob", method: http.MethodPost, path: "/applications/app-000001/indices/idx-000001/datasources/ds-000001/startsync", payload: `{}`},
		{name: "ListDataSourceSyncJobs", method: http.MethodGet, path: "/applications/app-000001/indices/idx-000001/datasources/ds-000001/syncjobs", payload: ``},
		{name: "Chat", method: http.MethodPost, path: "/applications/app-000001/conversations?userId=user-000001", payload: `{"userMessage":"hello stackyard qbusiness"}`},
		{name: "ListConversations", method: http.MethodGet, path: "/applications/app-000001/conversations?userId=user-000001", payload: ``},
		{name: "ListMessages", method: http.MethodGet, path: "/applications/app-000001/conversations/conv-000001?userId=user-000001", payload: ``},
		{name: "CreateChatResponseConfiguration", method: http.MethodPost, path: "/applications/app-000001/chatresponseconfigurations", payload: `{"displayName":"stage-cfg"}`},
		{name: "ListChatResponseConfigurations", method: http.MethodGet, path: "/applications/app-000001/chatresponseconfigurations", payload: ``},
		{name: "CreateSubscription", method: http.MethodPost, path: "/applications/app-000001/subscriptions", payload: `{"type":"Q_BUSINESS"}`},
		{name: "ListSubscriptions", method: http.MethodGet, path: "/applications/app-000001/subscriptions", payload: ``},
		{name: "TagResource", method: http.MethodPost, path: "/v1/tags/arn%3Aaws%3Aqbusiness%3Aus-east-1%3A123456789012%3Aapplication%2Fapp-000001", payload: `{"tags":{"env":"stage","owner":"tests"}}`},
		{name: "ListTagsForResource", method: http.MethodGet, path: "/v1/tags/arn%3Aaws%3Aqbusiness%3Aus-east-1%3A123456789012%3Aapplication%2Fapp-000001", payload: ``},
	}

	for _, tc := range cases {
		resp := qBusinessRequest(t, ts, tc.method, tc.path, tc.payload)
		body := string(mustBody(t, resp))
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("%s: expected 200, got %d: %s", tc.name, resp.StatusCode, body)
		}
		if strings.Contains(body, "NotImplemented") {
			t.Fatalf("%s: expected non-NotImplemented response, got: %s", tc.name, body)
		}
	}
}
