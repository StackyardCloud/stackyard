package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestElementalInferenceStage12FeedLifecycleAndReadSurface(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := elementalInferenceRequest(
		t,
		ts,
		http.MethodPost,
		"/v1/feed",
		[]byte(`{"name":"stage-elemental-feed","clientToken":"elemental-stage12-token"}`),
	)
	assertStatus(t, resp, http.StatusOK)
	createPayload := decodeElementalInferencePayload(t, resp)
	feedID := elementalInferencePayloadStringValue(createPayload, "feed", "id")
	if feedID == "" {
		feedID = elementalInferencePayloadStringValue(createPayload, "id")
	}
	if feedID == "" {
		t.Fatalf("expected CreateFeed response to include feed id")
	}

	resp = elementalInferenceRequest(t, ts, http.MethodGet, "/v1/feed/"+url.PathEscape(feedID), nil)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, feedID) {
		t.Fatalf("expected GetFeed response to include %q, got %q", feedID, body)
	}

	resp = elementalInferenceRequest(t, ts, http.MethodGet, "/v1/feeds", nil)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, feedID) {
		t.Fatalf("expected ListFeeds response to include %q, got %q", feedID, body)
	}

	resp = elementalInferenceRequest(
		t,
		ts,
		http.MethodPut,
		"/v1/feed/"+url.PathEscape(feedID),
		[]byte(`{"name":"stage-elemental-feed-updated"}`),
	)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "stage-elemental-feed-updated") {
		t.Fatalf("expected UpdateFeed response to include updated name, got %q", body)
	}

	resp = elementalInferenceRequest(t, ts, http.MethodDelete, "/v1/feed/"+url.PathEscape(feedID), nil)
	assertStatus(t, resp, http.StatusOK)
}

func TestElementalInferenceStage34AssociationAndTagging(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	feedID := "feed-000001"
	flowARN := "arn:aws:mediaconnect:us-east-1:123456789012:flow/flow-000001"
	resourceARN := "arn:aws:elemental-inference:us-east-1:123456789012:feed/" + feedID

	resp := elementalInferenceRequest(
		t,
		ts,
		http.MethodPost,
		"/v1/feed/"+url.PathEscape(feedID),
		[]byte(`{"flowArn":"`+flowARN+`"}`),
	)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "ASSOCIATED") || !strings.Contains(body, "flow-000001") {
		t.Fatalf("expected AssociateFeed response to include association details, got %q", body)
	}

	resp = elementalInferenceRequest(t, ts, http.MethodPost, "/v1/feed/"+url.PathEscape(feedID), []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "DISASSOCIATED") {
		t.Fatalf("expected DisassociateFeed response to include DISASSOCIATED, got %q", body)
	}

	resp = elementalInferenceRequest(
		t,
		ts,
		http.MethodPost,
		"/v1/tags/"+url.PathEscape(resourceARN),
		[]byte(`{"tags":{"env":"stage","owner":"qa"}}`),
	)
	assertStatus(t, resp, http.StatusOK)

	resp = elementalInferenceRequest(t, ts, http.MethodGet, "/v1/tags/"+url.PathEscape(resourceARN), nil)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "owner") || !strings.Contains(body, "qa") {
		t.Fatalf("expected ListTagsForResource to include owner tag, got %q", body)
	}

	resp = elementalInferenceRequest(t, ts, http.MethodDelete, "/v1/tags/"+url.PathEscape(resourceARN)+"?tagKeys=owner", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = elementalInferenceRequest(t, ts, http.MethodGet, "/v1/tags/"+url.PathEscape(resourceARN), nil)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); strings.Contains(body, "owner") {
		t.Fatalf("expected owner tag to be removed, got %q", body)
	}
}

func TestElementalInferenceStage56ValidationAndIdempotency(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := elementalInferenceRequest(t, ts, http.MethodPost, "/v1/unknown", []byte(`{}`))
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException for unknown route, got %q", body)
	}

	resp = signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/v1/feed",
		[]byte(`{"broken":`),
		map[string]string{"Content-Type": "application/json"},
		"elemental-inference",
	)
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "invalid JSON body") {
		t.Fatalf("expected invalid JSON body error, got %q", body)
	}

	resp = elementalInferenceRequest(
		t,
		ts,
		http.MethodPost,
		"/v1/feed",
		[]byte(`{"name":"idem-one","clientToken":"elemental-idempotent-token"}`),
	)
	assertStatus(t, resp, http.StatusOK)
	firstID := elementalInferencePayloadStringValue(decodeElementalInferencePayload(t, resp), "feed", "id")

	resp = elementalInferenceRequest(
		t,
		ts,
		http.MethodPost,
		"/v1/feed",
		[]byte(`{"name":"idem-two","clientToken":"elemental-idempotent-token"}`),
	)
	assertStatus(t, resp, http.StatusOK)
	secondID := elementalInferencePayloadStringValue(decodeElementalInferencePayload(t, resp), "feed", "id")

	if firstID == "" || secondID == "" {
		t.Fatalf("expected idempotent CreateFeed calls to return feed ids")
	}
	if firstID != secondID {
		t.Fatalf("expected idempotent CreateFeed calls to return same id, got %q and %q", firstID, secondID)
	}
}

func decodeElementalInferencePayload(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	payload := map[string]any{}
	if err := json.Unmarshal(mustBody(t, resp), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	return payload
}

func elementalInferencePayloadStringValue(payload map[string]any, keys ...string) string {
	if payload == nil {
		return ""
	}
	for _, key := range keys {
		raw, ok := payload[key]
		if !ok || raw == nil {
			continue
		}
		switch typed := raw.(type) {
		case string:
			if value := strings.TrimSpace(typed); value != "" {
				return value
			}
		case map[string]any:
			for _, nestedKey := range []string{"id", "feedId", "feedID"} {
				if nestedRaw, ok := typed[nestedKey]; ok {
					if nestedValue, ok := nestedRaw.(string); ok && strings.TrimSpace(nestedValue) != "" {
						return strings.TrimSpace(nestedValue)
					}
				}
			}
		}
	}
	return ""
}
