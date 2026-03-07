package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPChatRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPChatContractServer(t)

	assertGCPChatSuccess(t, ts, http.MethodGet, "/gcp/v1/spaces?pageSize=1", nil, "spaces")
	assertGCPChatSuccess(t, ts, http.MethodGet, "/gcp/v1/spaces/team-space", nil, "team-space")
	assertGCPChatSuccess(t, ts, http.MethodPost, "/gcp/v1/spaces/team-space/messages", []byte(`{"message":{"text":"hello from stackyard"}}`), "messages/message-1")
	assertGCPChatSuccess(t, ts, http.MethodGet, "/gcp/v1/spaces/team-space/messages?pageSize=1", nil, "messages")
	assertGCPChatSuccess(t, ts, http.MethodGet, "/gcp/v1/spaces/team-space/messages/message-1", nil, "message-1")
	assertGCPChatSuccess(t, ts, http.MethodPatch, "/gcp/v1/spaces/team-space/messages/message-1", []byte(`{"message":{"text":"updated by stackyard"}}`), "updated by stackyard")

	assertGCPChatSuccess(t, ts, http.MethodGet, "/gcp/v1/spaces/team-space/members?pageSize=1", nil, "memberships")
	assertGCPChatSuccess(t, ts, http.MethodGet, "/gcp/v1/spaces/team-space/members/user-1", nil, "user-1")

	assertGCPChatSuccess(t, ts, http.MethodGet, "/gcp/v1/spaces/team-space/messages/message-1/reactions?pageSize=1", nil, "reactions")
	assertGCPChatSuccess(t, ts, http.MethodPost, "/gcp/v1/spaces/team-space/messages/message-1/reactions", []byte(`{"reaction":{"emoji":{"unicode":"👍"}}}`), "reaction-1")
	assertGCPChatSuccess(t, ts, http.MethodDelete, "/gcp/v1/spaces/team-space/messages/message-1/reactions/reaction-1", nil, "{}")

	assertGCPChatSuccess(t, ts, http.MethodGet, "/gcp/v1/users/me/spaces/team-space/spaceReadState", nil, "spaceReadState")
	assertGCPChatSuccess(t, ts, http.MethodGet, "/gcp/v1/users/me/spaces/team-space/threads/thread-1/threadReadState", nil, "threadReadState")
	assertGCPChatSuccess(t, ts, http.MethodGet, "/gcp/v1/users/me/spaces/team-space/spaceNotificationSetting", nil, "spaceNotificationSetting")

	assertGCPChatSuccess(t, ts, http.MethodGet, "/gcp/v1/spaces/team-space/spaceEvents?pageSize=1", nil, "spaceEvents")
	assertGCPChatSuccess(t, ts, http.MethodGet, "/gcp/v1/spaces/team-space/spaceEvents/event-1", nil, "event-1")

	assertGCPChatSuccess(t, ts, http.MethodGet, "/gcp/v1/customEmojis?pageSize=1", nil, "customEmojis")
	assertGCPChatSuccess(t, ts, http.MethodGet, "/gcp/v1/customEmojis/emoji-1", nil, "emoji-1")

	assertGCPChatSuccess(t, ts, http.MethodPost, "/gcp/v1/spaces/team-space/attachments:upload", []byte(`{"filename":"evidence.json"}`), "attachmentDataRef")
	assertGCPChatSuccess(t, ts, http.MethodGet, "/gcp/v1/spaces/team-space/messages/message-1/attachments/attachment-1", nil, "attachment-1")

	assertGCPChatSuccess(t, ts, http.MethodPost, "/gcp/v1/spaces:search?pageSize=1", []byte(`{}`), "spaces")
	assertGCPChatSuccess(t, ts, http.MethodGet, "/gcp/v1/spaces:findDirectMessage?name=users/me", nil, "dm-space")
	assertGCPChatSuccess(t, ts, http.MethodPost, "/gcp/v1/spaces:setup", []byte(`{"space":{"displayName":"Team Space"}}`), "team-space")
}

func TestGCPChatRouter_GrpcRoutesStillNotImplemented(t *testing.T) {
	t.Parallel()

	ts := newGCPChatContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/google.chat.v1.ChatService/ListSpaces", []byte(`{}`), map[string]string{"Content-Type": "application/json"})
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp chat router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"gcp"`) || !strings.Contains(body, "ChatService/ListSpaces") {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPChatRouter_ListSpacesInvalidPageSize(t *testing.T) {
	t.Parallel()

	ts := newGCPChatContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/spaces?pageSize=bad", nil, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp chat router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPChatRouter_CreateMessageRequiresText(t *testing.T) {
	t.Parallel()

	ts := newGCPChatContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/spaces/team-space/messages", []byte(`{"message":{}}`), map[string]string{"Content-Type": "application/json"})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp chat router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPChatRouter_UploadAttachmentRequiresFilename(t *testing.T) {
	t.Parallel()

	ts := newGCPChatContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/spaces/team-space/attachments:upload", []byte(`{}`), map[string]string{"Content-Type": "application/json"})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp chat router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func newGCPChatContractServer(t *testing.T) *httptest.Server {
	t.Helper()

	return newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})
}

func assertGCPChatSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp chat router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}
