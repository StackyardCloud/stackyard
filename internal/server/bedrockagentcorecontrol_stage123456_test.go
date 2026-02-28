package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestBedrockAgentCoreControlStage1RuntimeAndEndpointLifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := bedrockAgentCoreControlRequest(t, ts, http.MethodPut, "/runtimes/", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	createRuntime := decodeBedrockAgentCoreControlPayload(t, resp)
	runtimeEntity, ok := createRuntime["agentRuntime"].(map[string]any)
	if !ok {
		t.Fatalf("expected CreateAgentRuntime response to include agentRuntime")
	}
	runtimeID := bagccPayloadString(runtimeEntity, "agentRuntimeId")
	if runtimeID == "" {
		t.Fatalf("expected CreateAgentRuntime to return agentRuntimeId")
	}

	resp = bedrockAgentCoreControlRequest(t, ts, http.MethodPost, "/runtimes/?maxResults=10&nextToken=token-1", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	listRuntimes := decodeBedrockAgentCoreControlPayload(t, resp)
	if _, ok := listRuntimes["agentRuntimes"].([]any); !ok {
		t.Fatalf("expected ListAgentRuntimes response to include agentRuntimes")
	}

	resp = bedrockAgentCoreControlRequest(t, ts, http.MethodGet, "/runtimes/"+url.PathEscape(runtimeID)+"/?version=v1", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = bedrockAgentCoreControlRequest(t, ts, http.MethodPut, "/runtimes/"+url.PathEscape(runtimeID)+"/", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)

	resp = bedrockAgentCoreControlRequest(t, ts, http.MethodPut, "/runtimes/"+url.PathEscape(runtimeID)+"/runtime-endpoints/", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	createEndpoint := decodeBedrockAgentCoreControlPayload(t, resp)
	endpointEntity, ok := createEndpoint["agentRuntimeEndpoint"].(map[string]any)
	if !ok {
		t.Fatalf("expected CreateAgentRuntimeEndpoint response to include agentRuntimeEndpoint")
	}
	endpointName := bagccPayloadString(endpointEntity, "endpointName")
	if endpointName == "" {
		t.Fatalf("expected CreateAgentRuntimeEndpoint to return endpointName")
	}

	resp = bedrockAgentCoreControlRequest(t, ts, http.MethodPost, "/runtimes/"+url.PathEscape(runtimeID)+"/runtime-endpoints/?maxResults=10&nextToken=token-1", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)

	resp = bedrockAgentCoreControlRequest(t, ts, http.MethodPost, "/runtimes/"+url.PathEscape(runtimeID)+"/versions/?maxResults=10&nextToken=token-1", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)

	resp = bedrockAgentCoreControlRequest(t, ts, http.MethodGet, "/runtimes/"+url.PathEscape(runtimeID)+"/runtime-endpoints/"+url.PathEscape(endpointName)+"/", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = bedrockAgentCoreControlRequest(t, ts, http.MethodPut, "/runtimes/"+url.PathEscape(runtimeID)+"/runtime-endpoints/"+url.PathEscape(endpointName)+"/", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)

	resp = bedrockAgentCoreControlRequest(t, ts, http.MethodDelete, "/runtimes/"+url.PathEscape(runtimeID)+"/runtime-endpoints/"+url.PathEscape(endpointName)+"/?clientToken=stackyard-client-token", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = bedrockAgentCoreControlRequest(t, ts, http.MethodDelete, "/runtimes/"+url.PathEscape(runtimeID)+"/?clientToken=stackyard-client-token", nil)
	assertStatus(t, resp, http.StatusOK)
}

func TestBedrockAgentCoreControlStage2ResourceLifecycles(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	calls := []struct {
		name   string
		method string
		path   string
		body   []byte
		key    string
	}{
		{name: "CreateBrowser", method: http.MethodPut, path: "/browsers", body: []byte(`{}`), key: "browser"},
		{name: "GetBrowser", method: http.MethodGet, path: "/browsers/browser-000001", key: "browser"},
		{name: "ListBrowsers", method: http.MethodPost, path: "/browsers?maxResults=10&nextToken=token-1&type=SYSTEM", body: []byte(`{}`), key: "browsers"},
		{name: "DeleteBrowser", method: http.MethodDelete, path: "/browsers/browser-000001?clientToken=stackyard-client-token", key: ""},
		{name: "CreateBrowserProfile", method: http.MethodPut, path: "/browser-profiles", body: []byte(`{}`), key: "browserProfile"},
		{name: "GetBrowserProfile", method: http.MethodGet, path: "/browser-profiles/profile-000001", key: "browserProfile"},
		{name: "ListBrowserProfiles", method: http.MethodPost, path: "/browser-profiles?maxResults=10&nextToken=token-1", body: []byte(`{}`), key: "browserProfiles"},
		{name: "DeleteBrowserProfile", method: http.MethodDelete, path: "/browser-profiles/profile-000001?clientToken=stackyard-client-token", key: ""},
		{name: "CreateCodeInterpreter", method: http.MethodPut, path: "/code-interpreters", body: []byte(`{}`), key: "codeInterpreter"},
		{name: "GetCodeInterpreter", method: http.MethodGet, path: "/code-interpreters/code-000001", key: "codeInterpreter"},
		{name: "ListCodeInterpreters", method: http.MethodPost, path: "/code-interpreters?maxResults=10&nextToken=token-1&type=SYSTEM", body: []byte(`{}`), key: "codeInterpreters"},
		{name: "DeleteCodeInterpreter", method: http.MethodDelete, path: "/code-interpreters/code-000001?clientToken=stackyard-client-token", key: ""},
		{name: "CreateMemory", method: http.MethodPost, path: "/memories/create", body: []byte(`{}`), key: "memory"},
		{name: "GetMemory", method: http.MethodGet, path: "/memories/memory-000001/details?view=DETAIL", key: "memory"},
		{name: "ListMemories", method: http.MethodPost, path: "/memories/", body: []byte(`{}`), key: "memories"},
		{name: "UpdateMemory", method: http.MethodPut, path: "/memories/memory-000001/update", body: []byte(`{}`), key: "memory"},
		{name: "DeleteMemory", method: http.MethodDelete, path: "/memories/memory-000001/delete?clientToken=stackyard-client-token", key: ""},
		{name: "CreateEvaluator", method: http.MethodPost, path: "/evaluators/create", body: []byte(`{}`), key: "evaluator"},
		{name: "GetEvaluator", method: http.MethodGet, path: "/evaluators/evaluator-000001", key: "evaluator"},
		{name: "ListEvaluators", method: http.MethodPost, path: "/evaluators?maxResults=10&nextToken=token-1", body: []byte(`{}`), key: "evaluators"},
		{name: "UpdateEvaluator", method: http.MethodPut, path: "/evaluators/evaluator-000001", body: []byte(`{}`), key: "evaluator"},
		{name: "DeleteEvaluator", method: http.MethodDelete, path: "/evaluators/evaluator-000001", key: ""},
		{name: "CreateOnlineEvaluationConfig", method: http.MethodPost, path: "/online-evaluation-configs/create", body: []byte(`{}`), key: "onlineEvaluationConfig"},
		{name: "GetOnlineEvaluationConfig", method: http.MethodGet, path: "/online-evaluation-configs/online-eval-000001", key: "onlineEvaluationConfig"},
		{name: "ListOnlineEvaluationConfigs", method: http.MethodPost, path: "/online-evaluation-configs?maxResults=10&nextToken=token-1", body: []byte(`{}`), key: "onlineEvaluationConfigs"},
		{name: "UpdateOnlineEvaluationConfig", method: http.MethodPut, path: "/online-evaluation-configs/online-eval-000001", body: []byte(`{}`), key: "onlineEvaluationConfig"},
		{name: "DeleteOnlineEvaluationConfig", method: http.MethodDelete, path: "/online-evaluation-configs/online-eval-000001", key: ""},
	}

	for _, tc := range calls {
		resp := bedrockAgentCoreControlRequest(t, ts, tc.method, tc.path, tc.body)
		assertStatus(t, resp, http.StatusOK)
		if tc.key == "" {
			continue
		}
		payload := decodeBedrockAgentCoreControlPayload(t, resp)
		if _, ok := payload[tc.key]; !ok {
			t.Fatalf("%s expected response key %q, got %v", tc.name, tc.key, payload)
		}
	}
}

func TestBedrockAgentCoreControlStage3GatewayAndTargets(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := bedrockAgentCoreControlRequest(t, ts, http.MethodPost, "/gateways/", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)

	resp = bedrockAgentCoreControlRequest(t, ts, http.MethodGet, "/gateways/?maxResults=10&nextToken=token-1", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = bedrockAgentCoreControlRequest(t, ts, http.MethodGet, "/gateways/gateway-000001/", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = bedrockAgentCoreControlRequest(t, ts, http.MethodPut, "/gateways/gateway-000001/", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)

	resp = bedrockAgentCoreControlRequest(t, ts, http.MethodPost, "/gateways/gateway-000001/targets/", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)

	resp = bedrockAgentCoreControlRequest(t, ts, http.MethodGet, "/gateways/gateway-000001/targets/?maxResults=10&nextToken=token-1", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = bedrockAgentCoreControlRequest(t, ts, http.MethodGet, "/gateways/gateway-000001/targets/target-000001/", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = bedrockAgentCoreControlRequest(t, ts, http.MethodPut, "/gateways/gateway-000001/targets/target-000001/", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)

	resp = bedrockAgentCoreControlRequest(t, ts, http.MethodPut, "/gateways/gateway-000001/synchronizeTargets", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	payload := decodeBedrockAgentCoreControlPayload(t, resp)
	if got := bagccPayloadString(payload, "status"); got != "SYNCHRONIZED" {
		t.Fatalf("expected SynchronizeGatewayTargets status SYNCHRONIZED, got %q", got)
	}

	resp = bedrockAgentCoreControlRequest(t, ts, http.MethodDelete, "/gateways/gateway-000001/targets/target-000001/", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = bedrockAgentCoreControlRequest(t, ts, http.MethodDelete, "/gateways/gateway-000001/", nil)
	assertStatus(t, resp, http.StatusOK)
}

func TestBedrockAgentCoreControlStage4PolicyEngineAndGeneration(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := bedrockAgentCoreControlRequest(t, ts, http.MethodPost, "/policy-engines", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)

	resp = bedrockAgentCoreControlRequest(t, ts, http.MethodGet, "/policy-engines?maxResults=10&nextToken=token-1", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = bedrockAgentCoreControlRequest(t, ts, http.MethodGet, "/policy-engines/policy-engine-000001", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = bedrockAgentCoreControlRequest(t, ts, http.MethodPut, "/policy-engines/policy-engine-000001", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)

	resp = bedrockAgentCoreControlRequest(t, ts, http.MethodPost, "/policy-engines/policy-engine-000001/policies", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)

	resp = bedrockAgentCoreControlRequest(t, ts, http.MethodGet, "/policy-engines/policy-engine-000001/policies?maxResults=10&nextToken=token-1&targetResourceScope=ALL", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = bedrockAgentCoreControlRequest(t, ts, http.MethodGet, "/policy-engines/policy-engine-000001/policies/policy-000001", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = bedrockAgentCoreControlRequest(t, ts, http.MethodPut, "/policy-engines/policy-engine-000001/policies/policy-000001", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)

	resp = bedrockAgentCoreControlRequest(t, ts, http.MethodPost, "/policy-engines/policy-engine-000001/policy-generations", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	startPayload := decodeBedrockAgentCoreControlPayload(t, resp)
	policyGen, ok := startPayload["policyGeneration"].(map[string]any)
	if !ok {
		t.Fatalf("expected StartPolicyGeneration response to include policyGeneration")
	}
	policyGenerationID := bagccPayloadString(policyGen, "policyGenerationId")
	if policyGenerationID == "" {
		t.Fatalf("expected StartPolicyGeneration to return policyGenerationId")
	}

	resp = bedrockAgentCoreControlRequest(t, ts, http.MethodGet, "/policy-engines/policy-engine-000001/policy-generations?maxResults=10&nextToken=token-1", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = bedrockAgentCoreControlRequest(t, ts, http.MethodGet, "/policy-engines/policy-engine-000001/policy-generations/"+url.PathEscape(policyGenerationID), nil)
	assertStatus(t, resp, http.StatusOK)

	resp = bedrockAgentCoreControlRequest(t, ts, http.MethodGet, "/policy-engines/policy-engine-000001/policy-generations/"+url.PathEscape(policyGenerationID)+"/assets?maxResults=10&nextToken=token-1", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = bedrockAgentCoreControlRequest(t, ts, http.MethodDelete, "/policy-engines/policy-engine-000001/policies/policy-000001", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = bedrockAgentCoreControlRequest(t, ts, http.MethodDelete, "/policy-engines/policy-engine-000001", nil)
	assertStatus(t, resp, http.StatusOK)
}

func TestBedrockAgentCoreControlStage5IdentityPolicyAndTagging(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	identityCalls := []struct {
		method string
		path   string
		body   []byte
		key    string
	}{
		{method: http.MethodPost, path: "/identities/CreateApiKeyCredentialProvider", body: []byte(`{}`), key: "apiKeyCredentialProvider"},
		{method: http.MethodPost, path: "/identities/GetApiKeyCredentialProvider", body: []byte(`{}`), key: "apiKeyCredentialProvider"},
		{method: http.MethodPost, path: "/identities/ListApiKeyCredentialProviders", body: []byte(`{}`), key: "apiKeyCredentialProviders"},
		{method: http.MethodPost, path: "/identities/UpdateApiKeyCredentialProvider", body: []byte(`{}`), key: "apiKeyCredentialProvider"},
		{method: http.MethodPost, path: "/identities/DeleteApiKeyCredentialProvider", body: []byte(`{}`), key: ""},
		{method: http.MethodPost, path: "/identities/CreateOauth2CredentialProvider", body: []byte(`{}`), key: "oauth2CredentialProvider"},
		{method: http.MethodPost, path: "/identities/GetOauth2CredentialProvider", body: []byte(`{}`), key: "oauth2CredentialProvider"},
		{method: http.MethodPost, path: "/identities/ListOauth2CredentialProviders", body: []byte(`{}`), key: "oauth2CredentialProviders"},
		{method: http.MethodPost, path: "/identities/UpdateOauth2CredentialProvider", body: []byte(`{}`), key: "oauth2CredentialProvider"},
		{method: http.MethodPost, path: "/identities/DeleteOauth2CredentialProvider", body: []byte(`{}`), key: ""},
		{method: http.MethodPost, path: "/identities/CreateWorkloadIdentity", body: []byte(`{}`), key: "workloadIdentity"},
		{method: http.MethodPost, path: "/identities/GetWorkloadIdentity", body: []byte(`{}`), key: "workloadIdentity"},
		{method: http.MethodPost, path: "/identities/ListWorkloadIdentities", body: []byte(`{}`), key: "workloadIdentities"},
		{method: http.MethodPost, path: "/identities/UpdateWorkloadIdentity", body: []byte(`{}`), key: "workloadIdentity"},
		{method: http.MethodPost, path: "/identities/DeleteWorkloadIdentity", body: []byte(`{}`), key: ""},
		{method: http.MethodPost, path: "/identities/set-token-vault-cmk", body: []byte(`{}`), key: "kmsKeyArn"},
		{method: http.MethodPost, path: "/identities/get-token-vault", body: []byte(`{}`), key: "tokenVault"},
	}

	for _, tc := range identityCalls {
		resp := bedrockAgentCoreControlRequest(t, ts, tc.method, tc.path, tc.body)
		assertStatus(t, resp, http.StatusOK)
		if tc.key == "" {
			continue
		}
		payload := decodeBedrockAgentCoreControlPayload(t, resp)
		if _, ok := payload[tc.key]; !ok {
			t.Fatalf("expected response key %q for %s", tc.key, tc.path)
		}
	}

	resourceARN := url.PathEscape("arn:aws:bedrock-agentcore:us-east-1:123456789012:resource/stackyard-control")

	resp := bedrockAgentCoreControlRequest(t, ts, http.MethodPost, "/tags/"+resourceARN, []byte(`{"tags":{"team":"platform","env":"dev"}}`))
	assertStatus(t, resp, http.StatusOK)

	resp = bedrockAgentCoreControlRequest(t, ts, http.MethodGet, "/tags/"+resourceARN, nil)
	assertStatus(t, resp, http.StatusOK)
	tagPayload := decodeBedrockAgentCoreControlPayload(t, resp)
	tags, ok := tagPayload["tags"].(map[string]any)
	if !ok || len(tags) == 0 {
		t.Fatalf("expected ListTagsForResource to return tags")
	}

	resp = bedrockAgentCoreControlRequest(t, ts, http.MethodDelete, "/tags/"+resourceARN+"?tagKeys=team", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = bedrockAgentCoreControlRequest(t, ts, http.MethodPut, "/resourcepolicy/"+resourceARN, []byte(`{"policy":"{\"Version\":\"2012-10-17\",\"Statement\":[]}"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = bedrockAgentCoreControlRequest(t, ts, http.MethodGet, "/resourcepolicy/"+resourceARN, nil)
	assertStatus(t, resp, http.StatusOK)
	policyPayload := decodeBedrockAgentCoreControlPayload(t, resp)
	if got := bagccPayloadString(policyPayload, "policy"); got == "" {
		t.Fatalf("expected GetResourcePolicy to return policy")
	}

	resp = bedrockAgentCoreControlRequest(t, ts, http.MethodDelete, "/resourcepolicy/"+resourceARN, nil)
	assertStatus(t, resp, http.StatusOK)
}

func TestBedrockAgentCoreControlStage6ValidationAndIdempotency(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := bedrockAgentCoreControlRequest(t, ts, http.MethodGet, "/control-plane-unknown-route", nil)
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException for unknown route, got %q", body)
	}

	resp = signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/policy-engines",
		[]byte(`{"incomplete":`),
		map[string]string{"Content-Type": "application/json"},
		"bedrock-agentcore",
	)
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "invalid JSON body") {
		t.Fatalf("expected invalid JSON body error, got %q", body)
	}

	resp = bedrockAgentCoreControlRequest(t, ts, http.MethodDelete, "/runtimes/runtime-000001/?clientToken=stackyard-client-token", nil)
	assertStatus(t, resp, http.StatusOK)
	resp = bedrockAgentCoreControlRequest(t, ts, http.MethodDelete, "/runtimes/runtime-000001/?clientToken=stackyard-client-token", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = bedrockAgentCoreControlRequest(t, ts, http.MethodDelete, "/tags/"+url.PathEscape("arn:aws:bedrock-agentcore:us-east-1:123456789012:resource/stackyard-control")+"?tagKeys=missing", nil)
	assertStatus(t, resp, http.StatusOK)
}

func decodeBedrockAgentCoreControlPayload(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	payload := map[string]any{}
	if err := json.Unmarshal(mustBody(t, resp), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	return payload
}

func bagccPayloadString(payload map[string]any, key string) string {
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
