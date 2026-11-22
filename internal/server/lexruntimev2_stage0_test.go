package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const lexRTSeedPath = "/bots/A1B2C3D4E5/botAliases/TSTALIASID/botLocales/en_US/sessions/"

func lexRTRequest(t *testing.T, ts *httptest.Server, method, path string, body []byte, headers map[string]string) *http.Response {
	t.Helper()
	h := map[string]string{}
	if body != nil {
		h["Content-Type"] = "application/json"
	}
	for k, v := range headers {
		h[k] = v
	}
	return signedRequestWithService(t, method, ts.URL+path, body, h, "lex")
}

func newLexRTTestServer(t *testing.T) (*Server, *httptest.Server) {
	t.Helper()
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return srv, ts
}

func lexRTErrorCode(t *testing.T, resp *http.Response) string {
	t.Helper()
	var body struct {
		Type string `json:"__type"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &body); err != nil {
		t.Fatalf("decode error body: %v", err)
	}
	return body.Type
}

func TestLexRuntimeV2Stage0Operations(t *testing.T) {
	if len(lexruntimev2Operations) != 6 || len(lexruntimev2OperationByName) != 6 {
		t.Fatalf("operations = %d (by name %d), want 6", len(lexruntimev2Operations), len(lexruntimev2OperationByName))
	}
	for _, name := range []string{"RecognizeText", "RecognizeUtterance", "PutSession", "GetSession", "DeleteSession", "StartConversation"} {
		if _, ok := lexruntimev2OperationByName[name]; !ok {
			t.Fatalf("missing operation %s", name)
		}
	}
}

func TestLexRuntimeV2Stage0RoutingAndValidation(t *testing.T) {
	_, ts := newLexRTTestServer(t)

	// The models API on the same signing name is untouched.
	resp := lexRTRequest(t, ts, http.MethodPost, "/bots/", []byte(`{}`), nil)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "botSummaries") {
		t.Fatalf("ListBots body = %s", body)
	}

	// A runtime path signed for another service is not claimed.
	req := httptest.NewRequest(http.MethodGet, lexRTSeedPath+"s1", nil)
	req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential=stackyard/20260101/us-east-1/s3/aws4_request, SignedHeaders=host, Signature=00")
	if _, _, ok := parseLexRuntimeV2Path(req); ok {
		t.Fatal("s3-signed request claimed by the Lex runtime router")
	}

	cases := []struct {
		name, method, path string
		body               string
		status             int
		code               string
	}{
		{"bad bot id", http.MethodPost, "/bots/short/botAliases/TSTALIASID/botLocales/en_US/sessions/s1/text", `{"text":"hi"}`, http.StatusBadRequest, "ValidationException"},
		{"bad session id", http.MethodPost, lexRTSeedPath + "x/text", `{"text":"hi"}`, http.StatusBadRequest, "ValidationException"},
		{"unknown bot", http.MethodPost, "/bots/ZZZZZZZZZZ/botAliases/TSTALIASID/botLocales/en_US/sessions/s1/text", `{"text":"hi"}`, http.StatusNotFound, "ResourceNotFoundException"},
		{"unknown alias", http.MethodPost, "/bots/A1B2C3D4E5/botAliases/NOPE/botLocales/en_US/sessions/s1/text", `{"text":"hi"}`, http.StatusNotFound, "ResourceNotFoundException"},
		{"unknown locale", http.MethodPost, "/bots/A1B2C3D4E5/botAliases/TSTALIASID/botLocales/fr_FR/sessions/s1/text", `{"text":"hi"}`, http.StatusNotFound, "ResourceNotFoundException"},
		{"missing text", http.MethodPost, lexRTSeedPath + "s1/text", `{}`, http.StatusBadRequest, "ValidationException"},
		{"text too long", http.MethodPost, lexRTSeedPath + "s1/text", `{"text":"` + strings.Repeat("a", 1025) + `"}`, http.StatusBadRequest, "ValidationException"},
		{"bad json", http.MethodPost, lexRTSeedPath + "s1/text", `{`, http.StatusBadRequest, "ValidationException"},
		{"bad dialog action", http.MethodPost, lexRTSeedPath + "s1", `{"sessionState":{"dialogAction":{"type":"Dance"}}}`, http.StatusBadRequest, "ValidationException"},
		{"put without state", http.MethodPost, lexRTSeedPath + "s1", `{}`, http.StatusBadRequest, "ValidationException"},
		{"intent without name", http.MethodPost, lexRTSeedPath + "s1", `{"sessionState":{"intent":{"state":"InProgress"}}}`, http.StatusBadRequest, "ValidationException"},
		{"bad message type", http.MethodPost, lexRTSeedPath + "s1", `{"sessionState":{},"messages":[{"contentType":"HTML","content":"x"}]}`, http.StatusBadRequest, "ValidationException"},
		{"get missing session", http.MethodGet, lexRTSeedPath + "never", "", http.StatusNotFound, "ResourceNotFoundException"},
		{"delete missing session", http.MethodDelete, lexRTSeedPath + "never", "", http.StatusNotFound, "ResourceNotFoundException"},
		{"unsupported method", http.MethodPut, lexRTSeedPath + "s1/text", `{"text":"hi"}`, http.StatusMethodNotAllowed, "ValidationException"},
	}
	for _, tc := range cases {
		var body []byte
		if tc.body != "" {
			body = []byte(tc.body)
		}
		resp := lexRTRequest(t, ts, tc.method, tc.path, body, nil)
		assertStatus(t, resp, tc.status)
		if code := lexRTErrorCode(t, resp); code != tc.code {
			t.Fatalf("%s: code = %q, want %q", tc.name, code, tc.code)
		}
	}

}
