package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func apiGatewayManagementConnect(t *testing.T, ts *httptest.Server, body string) *http.Response {
	t.Helper()
	resp, err := http.Post(ts.URL+"/_stackyard/apigatewaymanagementapi/connections", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func apiGatewayManagementConnectID(t *testing.T, ts *httptest.Server, body string) string {
	t.Helper()
	resp := apiGatewayManagementConnect(t, ts, body)
	assertStatus(t, resp, http.StatusCreated)
	var out struct {
		ConnectionID string `json:"connectionId"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &out); err != nil || len(out.ConnectionID) != 16 {
		t.Fatalf("bad connect response: %v %q", err, out.ConnectionID)
	}
	return out.ConnectionID
}

func TestAPIGatewayManagementStage1Lifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	id := apiGatewayManagementConnectID(t, ts, `{"sourceIp":"203.0.113.7","userAgent":"wscat"}`)
	path := "/@connections/" + url.PathEscape(id)

	resp := apiGatewayManagementRequest(t, ts, http.MethodGet, path, nil)
	assertStatus(t, resp, http.StatusOK)
	var got struct {
		ConnectedAt  string `json:"connectedAt"`
		LastActiveAt string `json:"lastActiveAt"`
		Identity     struct {
			SourceIP  string `json:"sourceIp"`
			UserAgent string `json:"userAgent"`
		} `json:"identity"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &got); err != nil {
		t.Fatal(err)
	}
	if got.Identity.SourceIP != "203.0.113.7" || got.Identity.UserAgent != "wscat" || got.ConnectedAt == "" || got.LastActiveAt == "" {
		t.Fatalf("unexpected connection %+v", got)
	}

	for _, msg := range []string{"hello", "world"} {
		resp = apiGatewayManagementRequest(t, ts, http.MethodPost, path, []byte(msg))
		assertStatus(t, resp, http.StatusOK)
		_ = mustBody(t, resp)
	}
	resp, err := http.Get(ts.URL + "/_stackyard/apigatewaymanagementapi/connections/" + url.PathEscape(id) + "/messages")
	if err != nil {
		t.Fatal(err)
	}
	assertStatus(t, resp, http.StatusOK)
	var inbox struct {
		Messages []struct {
			Data []byte `json:"data"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &inbox); err != nil {
		t.Fatal(err)
	}
	if len(inbox.Messages) != 2 || string(inbox.Messages[0].Data) != "hello" || string(inbox.Messages[1].Data) != "world" {
		t.Fatalf("unexpected inbox %+v", inbox)
	}

	resp = apiGatewayManagementRequest(t, ts, http.MethodPost, path, bytes.Repeat([]byte("x"), apiGatewayManagementMaxData+1))
	assertStatus(t, resp, http.StatusRequestEntityTooLarge)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "PayloadTooLargeException") {
		t.Fatalf("unexpected body %s", body)
	}

	resp = apiGatewayManagementRequest(t, ts, http.MethodDelete, path, nil)
	assertStatus(t, resp, http.StatusNoContent)
	_ = mustBody(t, resp)

	for _, method := range []string{http.MethodGet, http.MethodPost, http.MethodDelete} {
		resp = apiGatewayManagementRequest(t, ts, method, path, []byte("late"))
		assertStatus(t, resp, http.StatusGone)
		if body := string(mustBody(t, resp)); !strings.Contains(body, "GoneException") {
			t.Fatalf("%s: unexpected body %s", method, body)
		}
	}
}

func TestAPIGatewayManagementStage1ScopedToWebSocketAPIAndStage(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := apiGatewayManagementConnect(t, ts, `{"apiId":"api-000001","stage":"prod"}`)
	assertStatus(t, resp, http.StatusBadRequest) // the default API is HTTP, not WEBSOCKET
	_ = mustBody(t, resp)
	resp = apiGatewayManagementConnect(t, ts, `{not json`)
	assertStatus(t, resp, http.StatusBadRequest)
	_ = mustBody(t, resp)

	resp = apiGatewayV2Request(t, ts, http.MethodPost, "/apigatewayv2/create-api", `{"Name":"chat","ProtocolType":"WEBSOCKET"}`)
	assertStatus(t, resp, http.StatusOK)
	var api struct {
		ApiID string `json:"ApiId"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &api); err != nil || api.ApiID == "" {
		t.Fatalf("create api: %v", err)
	}
	id := apiGatewayManagementConnectID(t, ts, `{"apiId":"`+api.ApiID+`","stage":"prod"}`)
	escaped := url.PathEscape(id)

	resp = apiGatewayManagementRequest(t, ts, http.MethodGet, "/prod/@connections/"+escaped, nil)
	assertStatus(t, resp, http.StatusOK)
	_ = mustBody(t, resp)

	resp = apiGatewayManagementRequest(t, ts, http.MethodGet, "/dev/@connections/"+escaped, nil)
	assertStatus(t, resp, http.StatusGone)
	_ = mustBody(t, resp)

	// The request host carries the API ID; a connection is invisible to other APIs.
	if _, found := srv.apigatewaymanagementapi.get(id, "other-api", "prod"); found {
		t.Fatalf("connection must not be reachable through another API")
	}
	if _, found := srv.apigatewaymanagementapi.get(id, api.ApiID, ""); !found {
		t.Fatalf("connection must be reachable through its own API")
	}
}

func TestAPIGatewayManagementStage1DoesNotStealS3ObjectPaths(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := signedRequest(t, http.MethodPut, ts.URL+"/prod", nil, nil)
	assertStatus(t, resp, http.StatusOK)
	_ = mustBody(t, resp)
	resp = signedRequest(t, http.MethodPut, ts.URL+"/prod/@connections/key", []byte("s3 body"), nil)
	assertStatus(t, resp, http.StatusOK)
	_ = mustBody(t, resp)
	resp = signedRequest(t, http.MethodGet, ts.URL+"/prod/@connections/key", nil, nil)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); body != "s3 body" {
		t.Fatalf("expected S3 to serve the object, got %q", body)
	}
}
