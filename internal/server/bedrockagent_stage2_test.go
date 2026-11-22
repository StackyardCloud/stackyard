package server

import (
	"net/http"
	"strings"
	"testing"
)

func TestBedrockAgentStage2Errors(t *testing.T) {
	ts := newBedrockAgentTestServer(t)
	id := baObj(t, baWant(t, ts, http.StatusAccepted, http.MethodPut, "/agents/", `{"agentName":"one"}`), "agent")["agentId"].(string)

	cases := []struct {
		name, method, path, body string
		status                   int
		code                     string
	}{
		{"invalid JSON", http.MethodPut, "/agents/", `{"agentName":`, http.StatusBadRequest, "ValidationException"},
		{"missing name", http.MethodPut, "/agents/", `{}`, http.StatusBadRequest, "ValidationException"},
		{"bad name", http.MethodPut, "/agents/", `{"agentName":"no spaces"}`, http.StatusBadRequest, "ValidationException"},
		{"duplicate name", http.MethodPut, "/agents/", `{"agentName":"one"}`, http.StatusConflict, "ConflictException"},
		{"TTL out of range", http.MethodPut, "/agents/", `{"agentName":"two","idleSessionTTLInSeconds":5}`, http.StatusBadRequest, "ValidationException"},
		{"short instruction", http.MethodPut, "/agents/", `{"agentName":"three","instruction":"too short"}`, http.StatusBadRequest, "ValidationException"},
		{"bad agent id", http.MethodGet, "/agents/short/", "", http.StatusBadRequest, "ValidationException"},
		{"unknown agent", http.MethodGet, "/agents/ZZZZZZZZZZ/", "", http.StatusNotFound, "ResourceNotFoundException"},
		{"bad page token", http.MethodPost, "/agents/", `{"nextToken":"x"}`, http.StatusBadRequest, "ValidationException"},
		{"bad page size", http.MethodPost, "/agents/", `{"maxResults":0}`, http.StatusBadRequest, "ValidationException"},
		{"test alias is fixed", http.MethodDelete, "/agents/" + id + "/agentaliases/TSTALIASID/", "", http.StatusBadRequest, "ValidationException"},
		{"alias needs a prepared draft", http.MethodPut, "/agents/" + id + "/agentaliases/", `{"agentAliasName":"live"}`, http.StatusBadRequest, "ValidationException"},
		{"draft can't be deleted", http.MethodDelete, "/agents/" + id + "/agentversions/DRAFT/", "", http.StatusBadRequest, "ValidationException"},
		{"collaborators need a supervisor", http.MethodPut, "/agents/" + id + "/agentversions/DRAFT/agentcollaborators/", `{"agentDescriptor":{"aliasArn":"x"},"collaboratorName":"c","collaborationInstruction":"i"}`, http.StatusBadRequest, "ValidationException"},
		{"unknown knowledge base", http.MethodPut, "/agents/" + id + "/agentversions/DRAFT/knowledgebases/", `{"knowledgeBaseId":"ZZZZZZZZZZ","description":"d"}`, http.StatusNotFound, "ResourceNotFoundException"},
		{"vector KB needs storage", http.MethodPut, "/knowledgebases/", `{"name":"kb","roleArn":"r","knowledgeBaseConfiguration":{"type":"VECTOR","vectorKnowledgeBaseConfiguration":{"embeddingModelArn":"m"}}}`, http.StatusBadRequest, "ValidationException"},
		{"unknown KB type", http.MethodPut, "/knowledgebases/", `{"name":"kb","roleArn":"r","knowledgeBaseConfiguration":{"type":"GRAPH"}}`, http.StatusBadRequest, "ValidationException"},
		{"flow needs a role", http.MethodPost, "/flows/", `{"name":"f"}`, http.StatusBadRequest, "ValidationException"},
		{"two prompt variants", http.MethodPost, "/prompts/", `{"name":"p","variants":[{"name":"a","templateType":"TEXT","templateConfiguration":{}},{"name":"b","templateType":"TEXT","templateConfiguration":{}}]}`, http.StatusBadRequest, "ValidationException"},
		{"empty instruction", http.MethodPost, "/retrievalinstructions/", `{"name":"r","instruction":{}}`, http.StatusBadRequest, "ValidationException"},
		{"tag unknown resource", http.MethodPost, "/tags/arn:aws:bedrock:us-east-1:123456789012:agent%2FZZZZZZZZZZ", `{"tags":{"a":"b"}}`, http.StatusNotFound, "ResourceNotFoundException"},
		{"policy must be JSON", http.MethodPut, "/resourcepolicy/arn:aws:bedrock:us-east-1:123456789012:agent%2F" + id, `{"policy":"nope"}`, http.StatusBadRequest, "ValidationException"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			status, out := bedrockAgentCall(t, ts, c.method, c.path, c.body)
			if status != c.status || out["__type"] != c.code {
				t.Fatalf("expected %d %s, got %d %v", c.status, c.code, status, out)
			}
			if msg, _ := out["message"].(string); strings.TrimSpace(msg) == "" {
				t.Fatalf("error without a message: %v", out)
			}
		})
	}
}

func TestBedrockAgentStage2LookupsForRuntime(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	st := srv.bedrockagent
	in := map[string]any{"agentName": "rt", "foundationModel": "m", "agentResourceRoleArn": "arn:aws:iam::123456789012:role/r"}
	raw, err := st.handle("CreateAgent", in, nil)
	if err != nil {
		t.Fatal(err)
	}
	id := strings.Split(strings.Split(string(raw), `"agentId":"`)[1], `"`)[0]

	snap, ok := st.LookupAgentAlias(id, bedrockAgentTestAlias)
	if !ok || snap.Agent["agentVersion"] != bedrockAgentDraft {
		t.Fatalf("the test alias resolves to DRAFT: %v %v", ok, snap.Agent)
	}
	snap.Agent["agentName"] = "changed"
	if again, _ := st.LookupAgentAlias(id, bedrockAgentTestAlias); again.Agent["agentName"] != "rt" {
		t.Fatal("lookups must return copies")
	}
	if _, ok := st.LookupAgentAlias(id, "NOSUCHALIA"); ok {
		t.Fatal("unknown alias must not resolve")
	}
	if _, _, ok := st.LookupKnowledgeBase("ZZZZZZZZZZ"); ok {
		t.Fatal("unknown knowledge base must not resolve")
	}
	if _, ok := st.LookupFlowAlias("ZZZZZZZZZZ", bedrockAgentTestAlias); ok {
		t.Fatal("unknown flow must not resolve")
	}
	if _, ok := st.LookupPrompt("ZZZZZZZZZZ"); ok {
		t.Fatal("unknown prompt must not resolve")
	}
}
