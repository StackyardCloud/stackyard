package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestGameLiftStreamsStage123456LifecycleAndTagging(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := gameliftStreamsRequest(t, ts, http.MethodPost, "/applications", []byte(`{"description":"stage-app"}`))
	assertStatus(t, resp, http.StatusOK)
	app := decodeGameLiftStreamsPayload(t, resp)
	appID := gameLiftStreamsPayloadString(app, "identifier")
	if appID == "" {
		t.Fatalf("expected CreateApplication to return identifier")
	}

	resp = gameliftStreamsRequest(t, ts, http.MethodPost, "/streamgroups", []byte(`{"description":"stage-group"}`))
	assertStatus(t, resp, http.StatusOK)
	group := decodeGameLiftStreamsPayload(t, resp)
	groupID := gameLiftStreamsPayloadString(group, "identifier")
	if groupID == "" {
		t.Fatalf("expected CreateStreamGroup to return identifier")
	}

	resp = gameliftStreamsRequest(t, ts, http.MethodPost, "/streamgroups/"+url.PathEscape(groupID)+"/associations", []byte(`{"applicationIdentifiers":["`+appID+`"]}`))
	assertStatus(t, resp, http.StatusOK)

	resp = gameliftStreamsRequest(t, ts, http.MethodPost, "/streamgroups/"+url.PathEscape(groupID)+"/streamsessions", []byte(`{"applicationIdentifier":"`+appID+`","description":"stage-session"}`))
	assertStatus(t, resp, http.StatusOK)
	session := decodeGameLiftStreamsPayload(t, resp)
	sessionID := gameLiftStreamsPayloadString(session, "streamSessionIdentifier")
	if sessionID == "" {
		t.Fatalf("expected StartStreamSession to return streamSessionIdentifier")
	}

	resp = gameliftStreamsRequest(t, ts, http.MethodPost, "/streamgroups/"+url.PathEscape(groupID)+"/streamsessions/"+url.PathEscape(sessionID)+"/connections", []byte(`{"signalRequest":"OFFER"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = gameliftStreamsRequest(t, ts, http.MethodPut, "/streamgroups/"+url.PathEscape(groupID)+"/streamsessions/"+url.PathEscape(sessionID)+"/exportfiles", []byte(`{"outputUri":"s3://stackyard-gameliftstreams/exports/"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = gameliftStreamsRequest(t, ts, http.MethodGet, "/streamgroups/"+url.PathEscape(groupID)+"/streamsessions?MaxResults=10", nil)
	assertStatus(t, resp, http.StatusOK)
	list := decodeGameLiftStreamsPayload(t, resp)
	items, ok := list["items"].([]any)
	if !ok || len(items) == 0 {
		t.Fatalf("expected ListStreamSessions to return items")
	}

	resourceARN := url.PathEscape("arn:aws:gameliftstreams:us-east-1:123456789012:streamgroup/" + groupID)
	resp = gameliftStreamsRequest(t, ts, http.MethodPost, "/tags/"+resourceARN, []byte(`{"tags":{"team":"platform","env":"dev"}}`))
	assertStatus(t, resp, http.StatusOK)

	resp = gameliftStreamsRequest(t, ts, http.MethodGet, "/tags/"+resourceARN, nil)
	assertStatus(t, resp, http.StatusOK)
	tagPayload := decodeGameLiftStreamsPayload(t, resp)
	tags, ok := tagPayload["tags"].(map[string]any)
	if !ok || len(tags) == 0 {
		t.Fatalf("expected ListTagsForResource to return tags")
	}

	resp = gameliftStreamsRequest(t, ts, http.MethodDelete, "/tags/"+resourceARN+"?tagKeys=team", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = gameliftStreamsRequest(t, ts, http.MethodDelete, "/streamgroups/"+url.PathEscape(groupID)+"/streamsessions/"+url.PathEscape(sessionID), nil)
	assertStatus(t, resp, http.StatusOK)
}

func decodeGameLiftStreamsPayload(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	payload := map[string]any{}
	if err := json.Unmarshal(mustBody(t, resp), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	return payload
}

func gameLiftStreamsPayloadString(payload map[string]any, key string) string {
	raw, ok := payload[key]
	if !ok || raw == nil {
		return ""
	}
	value, ok := raw.(string)
	if !ok {
		return ""
	}
	return value
}
