package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestBedrockAgentCoreDataStage12IdentityAndMemoryLifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := bedrockAgentCoreDataRequest(t, ts, http.MethodPost, "/identities/api-key", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	apiKeyPayload := decodeBedrockAgentCoreDataPayload(t, resp)
	if got := bagcdPayloadString(apiKeyPayload, "apiKey"); got == "" {
		t.Fatalf("expected apiKey from GetResourceApiKey")
	}

	resp = bedrockAgentCoreDataRequest(t, ts, http.MethodPost, "/identities/oauth2/token", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	oauthPayload := decodeBedrockAgentCoreDataPayload(t, resp)
	if got := bagcdPayloadString(oauthPayload, "accessToken"); got == "" {
		t.Fatalf("expected accessToken from GetResourceOauth2Token")
	}

	resp = bedrockAgentCoreDataRequest(t, ts, http.MethodPost, "/identities/GetWorkloadAccessToken", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	workloadPayload := decodeBedrockAgentCoreDataPayload(t, resp)
	if got := bagcdPayloadString(workloadPayload, "accessToken"); got == "" {
		t.Fatalf("expected accessToken from GetWorkloadAccessToken")
	}

	resp = bedrockAgentCoreDataRequest(t, ts, http.MethodPost, "/identities/CompleteResourceTokenAuth", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	completePayload := decodeBedrockAgentCoreDataPayload(t, resp)
	if got := bagcdPayloadString(completePayload, "status"); got != "COMPLETED" {
		t.Fatalf("expected CompleteResourceTokenAuth status COMPLETED, got %q", got)
	}

	resp = bedrockAgentCoreDataRequest(t, ts, http.MethodPost, "/memories/memory-000001/memoryRecords/batchCreate", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	createPayload := decodeBedrockAgentCoreDataPayload(t, resp)
	records, _ := createPayload["memoryRecords"].([]any)
	if len(records) == 0 {
		t.Fatalf("expected BatchCreateMemoryRecords to return created records")
	}

	resp = bedrockAgentCoreDataRequest(t, ts, http.MethodGet, "/memories/memory-000001/memoryRecord/record-000001", nil)
	assertStatus(t, resp, http.StatusOK)
	getPayload := decodeBedrockAgentCoreDataPayload(t, resp)
	if _, ok := getPayload["memoryRecord"].(map[string]any); !ok {
		t.Fatalf("expected GetMemoryRecord response to include memoryRecord")
	}

	resp = bedrockAgentCoreDataRequest(t, ts, http.MethodPost, "/memories/memory-000001/memoryRecords/batchUpdate", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)

	resp = bedrockAgentCoreDataRequest(t, ts, http.MethodPost, "/memories/memory-000001/memoryRecords", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	listPayload := decodeBedrockAgentCoreDataPayload(t, resp)
	if _, ok := listPayload["memoryRecords"].([]any); !ok {
		t.Fatalf("expected ListMemoryRecords response to include memoryRecords")
	}

	resp = bedrockAgentCoreDataRequest(t, ts, http.MethodPost, "/memories/memory-000001/retrieve", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	retrievePayload := decodeBedrockAgentCoreDataPayload(t, resp)
	if _, ok := retrievePayload["memoryRecords"].([]any); !ok {
		t.Fatalf("expected RetrieveMemoryRecords response to include memoryRecords")
	}

	resp = bedrockAgentCoreDataRequest(t, ts, http.MethodPost, "/memories/memory-000001/memoryRecords/batchDelete", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)

	resp = bedrockAgentCoreDataRequest(t, ts, http.MethodDelete, "/memories/memory-000001/memoryRecords/record-000001", nil)
	assertStatus(t, resp, http.StatusOK)
}

func TestBedrockAgentCoreDataStage3ActorSessionEventAndExtraction(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := bedrockAgentCoreDataRequest(t, ts, http.MethodPost, "/memories/memory-000001/actors", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	actorsPayload := decodeBedrockAgentCoreDataPayload(t, resp)
	if _, ok := actorsPayload["actors"].([]any); !ok {
		t.Fatalf("expected ListActors response to include actors")
	}

	resp = bedrockAgentCoreDataRequest(t, ts, http.MethodPost, "/memories/memory-000001/actor/actor-000001/sessions", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	sessionsPayload := decodeBedrockAgentCoreDataPayload(t, resp)
	if _, ok := sessionsPayload["sessions"].([]any); !ok {
		t.Fatalf("expected ListSessions response to include sessions")
	}

	resp = bedrockAgentCoreDataRequest(t, ts, http.MethodPost, "/memories/memory-000001/events", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	createEventPayload := decodeBedrockAgentCoreDataPayload(t, resp)
	event, ok := createEventPayload["event"].(map[string]any)
	if !ok {
		t.Fatalf("expected CreateEvent response to include event")
	}
	eventID := bagcdPayloadString(event, "eventId")
	if eventID == "" {
		t.Fatalf("expected eventId from CreateEvent")
	}

	getEventPath := "/memories/memory-000001/actor/actor-000001/sessions/session-000001/events/" + url.PathEscape(eventID)
	resp = bedrockAgentCoreDataRequest(t, ts, http.MethodGet, getEventPath, nil)
	assertStatus(t, resp, http.StatusOK)

	resp = bedrockAgentCoreDataRequest(t, ts, http.MethodPost, "/memories/memory-000001/actor/actor-000001/sessions/session-000001", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	listEventsPayload := decodeBedrockAgentCoreDataPayload(t, resp)
	if _, ok := listEventsPayload["events"].([]any); !ok {
		t.Fatalf("expected ListEvents response to include events")
	}

	resp = bedrockAgentCoreDataRequest(t, ts, http.MethodPost, "/memories/memory-000001/extractionJobs/start", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	startExtractionPayload := decodeBedrockAgentCoreDataPayload(t, resp)
	if _, ok := startExtractionPayload["extractionJob"].(map[string]any); !ok {
		t.Fatalf("expected StartMemoryExtractionJob response to include extractionJob")
	}

	resp = bedrockAgentCoreDataRequest(t, ts, http.MethodPost, "/memories/memory-000001/extractionJobs", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	listExtractionPayload := decodeBedrockAgentCoreDataPayload(t, resp)
	if _, ok := listExtractionPayload["extractionJobs"].([]any); !ok {
		t.Fatalf("expected ListMemoryExtractionJobs response to include extractionJobs")
	}

	deleteEventPath := "/memories/memory-000001/actor/actor-000001/sessions/session-000001/events/" + url.PathEscape(eventID)
	resp = bedrockAgentCoreDataRequest(t, ts, http.MethodDelete, deleteEventPath, nil)
	assertStatus(t, resp, http.StatusOK)
}

func TestBedrockAgentCoreDataStage45BrowserAndCodeInterpreterSessions(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := bedrockAgentCoreDataRequest(t, ts, http.MethodPut, "/browsers/browser-000001/sessions/start", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	startBrowserPayload := decodeBedrockAgentCoreDataPayload(t, resp)
	browserSession, ok := startBrowserPayload["session"].(map[string]any)
	if !ok {
		t.Fatalf("expected StartBrowserSession response to include session")
	}
	browserSessionID := bagcdPayloadString(browserSession, "sessionId")
	if browserSessionID == "" {
		t.Fatalf("expected browser sessionId")
	}

	resp = bedrockAgentCoreDataRequest(t, ts, http.MethodGet, "/browsers/browser-000001/sessions/get?sessionId="+url.QueryEscape(browserSessionID), nil)
	assertStatus(t, resp, http.StatusOK)

	resp = bedrockAgentCoreDataRequest(t, ts, http.MethodPost, "/browsers/browser-000001/sessions/list", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)

	resp = bedrockAgentCoreDataRequest(t, ts, http.MethodPut, "/browsers/browser-000001/sessions/streams/update?sessionId="+url.QueryEscape(browserSessionID), []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	updatePayload := decodeBedrockAgentCoreDataPayload(t, resp)
	if got := bagcdPayloadString(updatePayload, "status"); got != "UPDATED" {
		t.Fatalf("expected UpdateBrowserStream status UPDATED, got %q", got)
	}

	resp = bedrockAgentCoreDataRequest(t, ts, http.MethodPut, "/browser-profiles/profile-000001/save", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)

	resp = bedrockAgentCoreDataRequest(t, ts, http.MethodPut, "/browsers/browser-000001/sessions/stop?sessionId="+url.QueryEscape(browserSessionID), []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)

	resp = bedrockAgentCoreDataRequest(t, ts, http.MethodPut, "/code-interpreters/code-000001/sessions/start", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	startCodePayload := decodeBedrockAgentCoreDataPayload(t, resp)
	codeSession, ok := startCodePayload["session"].(map[string]any)
	if !ok {
		t.Fatalf("expected StartCodeInterpreterSession response to include session")
	}
	codeSessionID := bagcdPayloadString(codeSession, "sessionId")
	if codeSessionID == "" {
		t.Fatalf("expected code interpreter sessionId")
	}

	resp = bedrockAgentCoreDataRequest(t, ts, http.MethodGet, "/code-interpreters/code-000001/sessions/get?sessionId="+url.QueryEscape(codeSessionID), nil)
	assertStatus(t, resp, http.StatusOK)

	resp = bedrockAgentCoreDataRequest(t, ts, http.MethodPost, "/code-interpreters/code-000001/sessions/list", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)

	resp = bedrockAgentCoreDataRequest(t, ts, http.MethodPost, "/code-interpreters/code-000001/tools/invoke", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)

	resp = bedrockAgentCoreDataRequest(t, ts, http.MethodPut, "/code-interpreters/code-000001/sessions/stop?sessionId="+url.QueryEscape(codeSessionID), []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
}

func TestBedrockAgentCoreDataStage6RuntimeAndEvaluation(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	runtimeARN := "arn:aws:bedrock-agentcore:us-east-1:123456789012:runtime/stackyard-runtime"
	runtimePath := url.PathEscape(runtimeARN)

	resp := bedrockAgentCoreDataRequest(
		t,
		ts,
		http.MethodGet,
		"/runtimes/"+runtimePath+"/invocations/.well-known/agent-card.json?qualifier=LATEST",
		nil,
	)
	assertStatus(t, resp, http.StatusOK)
	cardPayload := decodeBedrockAgentCoreDataPayload(t, resp)
	if got := bagcdPayloadString(cardPayload, "name"); got == "" {
		t.Fatalf("expected GetAgentCard response to include name")
	}

	resp = bedrockAgentCoreDataRequest(
		t,
		ts,
		http.MethodPost,
		"/runtimes/"+runtimePath+"/invocations?accountId=123456789012&qualifier=LATEST",
		[]byte(`{}`),
	)
	assertStatus(t, resp, http.StatusOK)
	invokePayload := decodeBedrockAgentCoreDataPayload(t, resp)
	if got := bagcdPayloadString(invokePayload, "sessionId"); got == "" {
		t.Fatalf("expected InvokeAgentRuntime response to include sessionId")
	}

	resp = bedrockAgentCoreDataRequest(
		t,
		ts,
		http.MethodPost,
		"/runtimes/"+runtimePath+"/stopruntimesession?qualifier=LATEST",
		[]byte(`{}`),
	)
	assertStatus(t, resp, http.StatusOK)

	resp = bedrockAgentCoreDataRequest(t, ts, http.MethodPost, "/evaluations/evaluate/evaluator-000001", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	evalPayload := decodeBedrockAgentCoreDataPayload(t, resp)
	if _, ok := evalPayload["result"].(map[string]any); !ok {
		t.Fatalf("expected Evaluate response to include result")
	}
}

func decodeBedrockAgentCoreDataPayload(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	payload := map[string]any{}
	if err := json.Unmarshal(mustBody(t, resp), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	return payload
}

func bagcdPayloadString(payload map[string]any, key string) string {
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
