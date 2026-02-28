package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestConnectStage12InstanceLifecycleAndReadSurface(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := connectRequest(t, ts, http.MethodPut, "/instance", []byte(`{"InstanceAlias":"stage-instance"}`))
	assertStatus(t, resp, http.StatusOK)
	createPayload := decodeConnectPayload(t, resp)
	instanceID := connectPayloadString(createPayload, "Id")
	if instanceID == "" {
		t.Fatalf("expected CreateInstance to return Id")
	}

	resp = connectRequest(t, ts, http.MethodGet, "/instance/"+url.PathEscape(instanceID), nil)
	assertStatus(t, resp, http.StatusOK)

	resp = connectRequest(t, ts, http.MethodGet, "/instance", nil)
	assertStatus(t, resp, http.StatusOK)
	listPayload := decodeConnectPayload(t, resp)
	items, ok := listPayload["InstanceSummaryList"].([]any)
	if !ok || len(items) == 0 {
		t.Fatalf("expected ListInstances to return InstanceSummaryList")
	}

	resp = connectRequest(t, ts, http.MethodDelete, "/instance/"+url.PathEscape(instanceID), nil)
	assertStatus(t, resp, http.StatusOK)
}

func TestConnectStage34UserQueueAndContactFlowLifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	instanceID := "instance-000001"

	resp := connectRequest(t, ts, http.MethodPut, "/users/"+url.PathEscape(instanceID), []byte(`{"Username":"stage-user"}`))
	assertStatus(t, resp, http.StatusOK)
	createUserPayload := decodeConnectPayload(t, resp)
	userID := connectPayloadString(createUserPayload, "UserId")
	if userID == "" {
		t.Fatalf("expected CreateUser to return UserId")
	}

	resp = connectRequest(t, ts, http.MethodGet, "/users/"+url.PathEscape(instanceID)+"/"+url.PathEscape(userID), nil)
	assertStatus(t, resp, http.StatusOK)

	resp = connectRequest(t, ts, http.MethodGet, "/users-summary/"+url.PathEscape(instanceID), nil)
	assertStatus(t, resp, http.StatusOK)

	resp = connectRequest(t, ts, http.MethodPut, "/queues/"+url.PathEscape(instanceID), []byte(`{"Name":"stage-queue"}`))
	assertStatus(t, resp, http.StatusOK)
	createQueuePayload := decodeConnectPayload(t, resp)
	queueID := connectPayloadString(createQueuePayload, "QueueId")
	if queueID == "" {
		t.Fatalf("expected CreateQueue to return QueueId")
	}

	resp = connectRequest(t, ts, http.MethodGet, "/queues/"+url.PathEscape(instanceID)+"/"+url.PathEscape(queueID), nil)
	assertStatus(t, resp, http.StatusOK)

	resp = connectRequest(t, ts, http.MethodGet, "/queues-summary/"+url.PathEscape(instanceID), nil)
	assertStatus(t, resp, http.StatusOK)

	resp = connectRequest(t, ts, http.MethodPut, "/contact-flows/"+url.PathEscape(instanceID), []byte(`{"Name":"stage-flow"}`))
	assertStatus(t, resp, http.StatusOK)
	createFlowPayload := decodeConnectPayload(t, resp)
	contactFlowID := connectPayloadString(createFlowPayload, "ContactFlowId")
	if contactFlowID == "" {
		t.Fatalf("expected CreateContactFlow to return ContactFlowId")
	}

	resp = connectRequest(
		t,
		ts,
		http.MethodGet,
		"/contact-flows/"+url.PathEscape(instanceID)+"/"+url.PathEscape(contactFlowID),
		nil,
	)
	assertStatus(t, resp, http.StatusOK)

	resp = connectRequest(t, ts, http.MethodGet, "/contact-flows-summary/"+url.PathEscape(instanceID), nil)
	assertStatus(t, resp, http.StatusOK)
	listFlowPayload := decodeConnectPayload(t, resp)
	flows, ok := listFlowPayload["ContactFlowSummaryList"].([]any)
	if !ok || len(flows) == 0 {
		t.Fatalf("expected ListContactFlows to return ContactFlowSummaryList")
	}

	resp = connectRequest(t, ts, http.MethodDelete, "/users/"+url.PathEscape(instanceID)+"/"+url.PathEscape(userID), nil)
	assertStatus(t, resp, http.StatusOK)

	resp = connectRequest(t, ts, http.MethodDelete, "/queues/"+url.PathEscape(instanceID)+"/"+url.PathEscape(queueID), nil)
	assertStatus(t, resp, http.StatusOK)

	resp = connectRequest(t, ts, http.MethodDelete, "/contact-flows/"+url.PathEscape(instanceID)+"/"+url.PathEscape(contactFlowID), nil)
	assertStatus(t, resp, http.StatusOK)
}

func TestConnectStage56TaggingValidationAndIdempotency(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resourceARN := "arn:aws:connect:us-east-1:123456789012:instance/instance-000001/contact-flow/contact-flow-000001"
	escapedARN := url.PathEscape(resourceARN)

	resp := connectRequest(t, ts, http.MethodPost, "/tags/"+escapedARN, []byte(`{"tags":{"owner":"qa","env":"dev"}}`))
	assertStatus(t, resp, http.StatusOK)

	resp = connectRequest(t, ts, http.MethodGet, "/tags/"+escapedARN, nil)
	assertStatus(t, resp, http.StatusOK)
	tagPayload := decodeConnectPayload(t, resp)
	tags, ok := tagPayload["tags"].(map[string]any)
	if !ok {
		t.Fatalf("expected ListTagsForResource to return tags map")
	}
	if got := connectPayloadString(tags, "owner"); got != "qa" {
		t.Fatalf("expected owner tag qa, got %q", got)
	}

	resp = connectRequest(t, ts, http.MethodDelete, "/tags/"+escapedARN+"?tagKeys=owner", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = connectRequest(t, ts, http.MethodGet, "/tags/"+escapedARN, nil)
	assertStatus(t, resp, http.StatusOK)
	tagPayload = decodeConnectPayload(t, resp)
	tags, _ = tagPayload["tags"].(map[string]any)
	if _, exists := tags["owner"]; exists {
		t.Fatalf("expected owner tag to be removed")
	}

	resp = connectRequest(t, ts, http.MethodPost, "/unknown-connect-route", []byte(`{}`))
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException for unknown route, got %q", body)
	}

	resp = signedRequestWithService(
		t,
		http.MethodPut,
		ts.URL+"/instance",
		[]byte(`{"broken":`),
		map[string]string{"Content-Type": "application/json"},
		"connect",
	)
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "invalid JSON body") {
		t.Fatalf("expected invalid JSON body error, got %q", body)
	}

	resp = connectRequest(t, ts, http.MethodDelete, "/queues/instance-000001/queue-000001", nil)
	assertStatus(t, resp, http.StatusOK)
	resp = connectRequest(t, ts, http.MethodDelete, "/queues/instance-000001/queue-000001", nil)
	assertStatus(t, resp, http.StatusOK)
}

func decodeConnectPayload(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	payload := map[string]any{}
	if err := json.Unmarshal(mustBody(t, resp), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	return payload
}

func connectPayloadString(payload map[string]any, key string) string {
	raw, ok := payload[key]
	if !ok || raw == nil {
		return ""
	}
	value, ok := raw.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(value)
}
