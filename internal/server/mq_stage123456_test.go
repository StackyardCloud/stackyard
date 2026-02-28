package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestMQStage12BrokerLifecycleAndReadSurface(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := mqRequest(t, ts, http.MethodPost, "/v1/brokers", []byte(`{"BrokerName":"stage-mq-broker"}`))
	assertStatus(t, resp, http.StatusOK)
	createPayload := decodeMQPayload(t, resp)
	brokerID := mqPayloadString(createPayload, "BrokerId")
	if brokerID == "" {
		t.Fatalf("expected CreateBroker to return BrokerId")
	}

	resp = mqRequest(t, ts, http.MethodGet, "/v1/brokers/"+url.PathEscape(brokerID), nil)
	assertStatus(t, resp, http.StatusOK)

	resp = mqRequest(t, ts, http.MethodGet, "/v1/brokers", nil)
	assertStatus(t, resp, http.StatusOK)
	listPayload := decodeMQPayload(t, resp)
	items, ok := listPayload["BrokerSummaries"].([]any)
	if !ok || len(items) == 0 {
		t.Fatalf("expected ListBrokers to return BrokerSummaries")
	}

	resp = mqRequest(t, ts, http.MethodPut, "/v1/brokers/"+url.PathEscape(brokerID), []byte(`{"HostInstanceType":"mq.t3.small"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = mqRequest(t, ts, http.MethodPost, "/v1/brokers/"+url.PathEscape(brokerID)+"/reboot", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)

	resp = mqRequest(t, ts, http.MethodPost, "/v1/brokers/"+url.PathEscape(brokerID)+"/promote", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)

	resp = mqRequest(t, ts, http.MethodDelete, "/v1/brokers/"+url.PathEscape(brokerID), nil)
	assertStatus(t, resp, http.StatusOK)
}

func TestMQStage34ConfigurationLifecycleAndRevisions(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := mqRequest(t, ts, http.MethodPost, "/v1/configurations", []byte(`{"Name":"stage-mq-configuration","EngineType":"ACTIVEMQ"}`))
	assertStatus(t, resp, http.StatusOK)
	createPayload := decodeMQPayload(t, resp)
	configID := mqPayloadString(createPayload, "Id")
	if configID == "" {
		t.Fatalf("expected CreateConfiguration to return Id")
	}

	resp = mqRequest(t, ts, http.MethodGet, "/v1/configurations/"+url.PathEscape(configID), nil)
	assertStatus(t, resp, http.StatusOK)

	resp = mqRequest(t, ts, http.MethodGet, "/v1/configurations", nil)
	assertStatus(t, resp, http.StatusOK)
	listPayload := decodeMQPayload(t, resp)
	configs, ok := listPayload["Configurations"].([]any)
	if !ok || len(configs) == 0 {
		t.Fatalf("expected ListConfigurations to return Configurations")
	}

	resp = mqRequest(
		t,
		ts,
		http.MethodPut,
		"/v1/configurations/"+url.PathEscape(configID),
		[]byte(`{"Description":"updated revision","Data":"<broker updated='true'/>"}`),
	)
	assertStatus(t, resp, http.StatusOK)

	resp = mqRequest(t, ts, http.MethodGet, "/v1/configurations/"+url.PathEscape(configID)+"/revisions", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = mqRequest(t, ts, http.MethodGet, "/v1/configurations/"+url.PathEscape(configID)+"/revisions/1", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = mqRequest(t, ts, http.MethodDelete, "/v1/configurations/"+url.PathEscape(configID), nil)
	assertStatus(t, resp, http.StatusOK)
}

func TestMQStage5UsersAndTaggingLifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	brokerID := "b-000001"
	username := "stage-user"

	resp := mqRequest(
		t,
		ts,
		http.MethodPost,
		"/v1/brokers/"+url.PathEscape(brokerID)+"/users/"+url.PathEscape(username),
		[]byte(`{"ConsoleAccess":true,"Groups":["admins","developers"]}`),
	)
	assertStatus(t, resp, http.StatusOK)

	resp = mqRequest(t, ts, http.MethodGet, "/v1/brokers/"+url.PathEscape(brokerID)+"/users/"+url.PathEscape(username), nil)
	assertStatus(t, resp, http.StatusOK)

	resp = mqRequest(t, ts, http.MethodGet, "/v1/brokers/"+url.PathEscape(brokerID)+"/users", nil)
	assertStatus(t, resp, http.StatusOK)
	usersPayload := decodeMQPayload(t, resp)
	users, ok := usersPayload["Users"].([]any)
	if !ok || len(users) == 0 {
		t.Fatalf("expected ListUsers to return Users")
	}

	resp = mqRequest(
		t,
		ts,
		http.MethodPut,
		"/v1/brokers/"+url.PathEscape(brokerID)+"/users/"+url.PathEscape(username),
		[]byte(`{"ConsoleAccess":false,"Groups":["readers"]}`),
	)
	assertStatus(t, resp, http.StatusOK)

	resourceARN := "arn:aws:mq:us-east-1:123456789012:broker:stackyard-broker:b-000001"
	escapedARN := url.PathEscape(resourceARN)

	resp = mqRequest(t, ts, http.MethodPost, "/v1/tags/"+escapedARN, []byte(`{"Tags":{"owner":"qa","env":"dev"}}`))
	assertStatus(t, resp, http.StatusOK)

	resp = mqRequest(t, ts, http.MethodGet, "/v1/tags/"+escapedARN, nil)
	assertStatus(t, resp, http.StatusOK)
	tagPayload := decodeMQPayload(t, resp)
	tags, ok := tagPayload["Tags"].(map[string]any)
	if !ok {
		t.Fatalf("expected ListTags to return Tags map")
	}
	if got := mqPayloadString(tags, "owner"); got != "qa" {
		t.Fatalf("expected owner tag qa, got %q", got)
	}

	resp = mqRequest(t, ts, http.MethodDelete, "/v1/tags/"+escapedARN+"?tagKeys=owner", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = mqRequest(t, ts, http.MethodDelete, "/v1/brokers/"+url.PathEscape(brokerID)+"/users/"+url.PathEscape(username), nil)
	assertStatus(t, resp, http.StatusOK)
}

func TestMQStage6ValidationAndIdempotency(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := mqRequest(t, ts, http.MethodPost, "/v1/mq-unknown-route", []byte(`{}`))
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException for unknown route, got %q", body)
	}

	resp = signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/v1/brokers",
		[]byte(`{"broken":`),
		map[string]string{"Content-Type": "application/json"},
		"mq",
	)
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "invalid JSON body") {
		t.Fatalf("expected invalid JSON body error, got %q", body)
	}

	resp = mqRequest(t, ts, http.MethodDelete, "/v1/brokers/b-000001/users/admin", nil)
	assertStatus(t, resp, http.StatusOK)
	resp = mqRequest(t, ts, http.MethodDelete, "/v1/brokers/b-000001/users/admin", nil)
	assertStatus(t, resp, http.StatusOK)
}

func decodeMQPayload(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	payload := map[string]any{}
	if err := json.Unmarshal(mustBody(t, resp), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	return payload
}

func mqPayloadString(payload map[string]any, key string) string {
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
