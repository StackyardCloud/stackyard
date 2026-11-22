package server

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// barKBFixture is a knowledge base with one inline data source and two documents.
type barKBFixture struct{ kb, ds string }

func barNewKB(t *testing.T, ts *httptest.Server) barKBFixture {
	t.Helper()
	kb := createKnowledgeBase(t, ts, "runtimekb")
	ds := baObj(t, baWant(t, ts, http.StatusOK, http.MethodPut, "/knowledgebases/"+kb+"/datasources/", `{"name":"inline","dataSourceConfiguration":{"type":"CUSTOM"}}`), "dataSource")["dataSourceId"].(string)
	doc := func(id, text, team string) string {
		return fmt.Sprintf(`{"content":{"dataSourceType":"CUSTOM","custom":{"customDocumentIdentifier":{"id":%q},"sourceType":"IN_LINE","inlineContent":{"type":"TEXT","textContent":{"data":%q}}}},"metadata":{"type":"IN_LINE_ATTRIBUTE","inlineAttributes":[{"key":"team","value":{"type":"STRING","stringValue":%q}}]}}`, id, text, team)
	}
	baWant(t, ts, http.StatusAccepted, http.MethodPut, "/knowledgebases/"+kb+"/datasources/"+ds+"/documents",
		`{"documents":[`+doc("refunds", "Refunds are issued within 14 days of purchase.", "billing")+`,`+doc("shipping", "Shipping takes 3 business days within the country.", "logistics")+`]}`)
	return barKBFixture{kb: kb, ds: ds}
}

func barStream(t *testing.T, ts *httptest.Server, method, path string, body any) (http.Header, []string, []map[string]any) {
	t.Helper()
	status, hdr, raw := brtCall(t, ts, method, path, body, nil)
	if status != http.StatusOK {
		t.Fatalf("%s %s: %d %s", method, path, status, raw)
	}
	if ct := hdr.Get("Content-Type"); ct != "application/vnd.amazon.eventstream" {
		t.Fatalf("%s: content type %q", path, ct)
	}
	types, payloads := brtFrames(t, raw)
	return hdr, types, payloads
}

func barChunkText(t *testing.T, payload map[string]any) string {
	t.Helper()
	raw, err := base64.StdEncoding.DecodeString(payload["bytes"].(string))
	if err != nil {
		t.Fatalf("chunk bytes: %v", err)
	}
	return string(raw)
}

func barJSON(t *testing.T, ts *httptest.Server, want int, method, path string, body any) map[string]any {
	t.Helper()
	status, _, raw := brtCall(t, ts, method, path, body, nil)
	if status != want {
		t.Fatalf("%s %s: %d (want %d) %s", method, path, status, want, raw)
	}
	out := map[string]any{}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			t.Fatalf("decode %s: %v", raw, err)
		}
	}
	return out
}

func TestBedrockAgentRuntimeStage1InvokeAgent(t *testing.T) {
	ts := newBedrockAgentTestServer(t)
	fx := barNewKB(t, ts)

	id := createPreparedAgent(t, ts, "helpdesk", "DISABLED")
	draft := "/agents/" + id + "/agentversions/DRAFT"
	baWant(t, ts, http.StatusOK, http.MethodPut, draft+"/knowledgebases/", fmt.Sprintf(`{"knowledgeBaseId":%q,"description":"policies"}`, fx.kb))
	baWant(t, ts, http.StatusOK, http.MethodPut, draft+"/actiongroups/", `{"actionGroupName":"orders","actionGroupExecutor":{"customControl":"RETURN_CONTROL"},"functionSchema":{"functions":[{"name":"cancel_order","parameters":{"orderId":{"type":"string","required":true}}}]}}`)
	baWant(t, ts, http.StatusAccepted, http.MethodPost, "/agents/"+id+"/", "")
	alias := baObj(t, baWant(t, ts, http.StatusAccepted, http.MethodPut, "/agents/"+id+"/agentaliases/", `{"agentAliasName":"live"}`), "agentAlias")["agentAliasId"].(string)
	invoke := "/agents/" + id + "/agentAliases/" + alias + "/sessions/s-1/text"

	// The knowledge base answers, with a citation and a trace.
	hdr, types, payloads := barStream(t, ts, http.MethodPost, invoke, map[string]any{"inputText": "How long do refunds take?", "enableTrace": true})
	if hdr.Get("x-amz-bedrock-agent-session-id") != "s-1" || hdr.Get("x-amzn-bedrock-agent-content-type") != "application/json" {
		t.Fatalf("missing agent headers: %v", hdr)
	}
	last := len(types) - 1
	if types[last] != "chunk" || types[0] != "trace" {
		t.Fatalf("expected traces then a chunk, got %v", types)
	}
	if text := barChunkText(t, payloads[last]); !strings.Contains(text, "14 days") {
		t.Fatalf("answer should come from the refunds document: %q", text)
	}
	citations := payloads[last]["attribution"].(map[string]any)["citations"].([]any)
	ref := citations[0].(map[string]any)["retrievedReferences"].([]any)[0].(map[string]any)
	if ref["location"].(map[string]any)["customDocumentLocation"].(map[string]any)["id"] != "refunds" {
		t.Fatalf("citation should point at the refunds document: %v", ref)
	}

	// A RETURN_CONTROL function named in the input hands control back.
	_, types, payloads = barStream(t, ts, http.MethodPost, invoke, map[string]any{"inputText": "please cancel_order orderId=A-17"})
	if len(types) != 1 || types[0] != "returnControl" {
		t.Fatalf("expected returnControl, got %v", types)
	}
	call := payloads[0]["invocationInputs"].([]any)[0].(map[string]any)["functionInvocationInput"].(map[string]any)
	params := call["parameters"].([]any)
	if call["function"] != "cancel_order" || len(params) != 1 || params[0].(map[string]any)["value"] != "A-17" {
		t.Fatalf("unexpected function call: %v", call)
	}
	invocationID := payloads[0]["invocationId"].(string)

	// Results for a different invocation are rejected; the right one is answered.
	results := []any{map[string]any{"functionResult": map[string]any{"actionGroup": "orders", "function": "cancel_order", "responseBody": map[string]any{"TEXT": map[string]any{"body": "cancelled"}}}}}
	barJSON(t, ts, http.StatusBadRequest, http.MethodPost, invoke, map[string]any{"sessionState": map[string]any{"invocationId": "other", "returnControlInvocationResults": results}})
	_, _, payloads = barStream(t, ts, http.MethodPost, invoke, map[string]any{"sessionState": map[string]any{"invocationId": invocationID, "returnControlInvocationResults": results}})
	if text := barChunkText(t, payloads[0]); !strings.Contains(text, "cancelled") {
		t.Fatalf("results should be acknowledged: %q", text)
	}
	// Nothing is pending any more.
	barJSON(t, ts, http.StatusBadRequest, http.MethodPost, invoke, map[string]any{"sessionState": map[string]any{"returnControlInvocationResults": results}})

	// Errors: missing input, bad session ID, unknown alias, unprepared agent.
	barJSON(t, ts, http.StatusBadRequest, http.MethodPost, invoke, map[string]any{})
	barJSON(t, ts, http.StatusBadRequest, http.MethodPost, "/agents/"+id+"/agentAliases/"+alias+"/sessions/x/text", map[string]any{"inputText": "hi"})
	barJSON(t, ts, http.StatusNotFound, http.MethodPost, "/agents/"+id+"/agentAliases/NOPE000000/sessions/s-1/text", map[string]any{"inputText": "hi"})
	fresh := baObj(t, baWant(t, ts, http.StatusAccepted, http.MethodPut, "/agents/", `{"agentName":"unprepared"}`), "agent")["agentId"].(string)
	out := barJSON(t, ts, http.StatusConflict, http.MethodPost, "/agents/"+fresh+"/agentAliases/TSTALIASID/sessions/s-1/text", map[string]any{"inputText": "hi"})
	if out["__type"] != "ConflictException" {
		t.Fatalf("expected ConflictException, got %v", out)
	}
}

func TestBedrockAgentRuntimeStage1AgentMemory(t *testing.T) {
	ts := newBedrockAgentTestServer(t)
	out := baWant(t, ts, http.StatusAccepted, http.MethodPut, "/agents/", fmt.Sprintf(`{"agentName":"memo","foundationModel":"anthropic.claude-v2","agentResourceRoleArn":"arn:aws:iam::123456789012:role/agent","instruction":%q,"memoryConfiguration":{"enabledMemoryTypes":["SESSION_SUMMARY"],"storageDays":30}}`, bedrockAgentTestInstruction))
	id := baObj(t, out, "agent")["agentId"].(string)
	baWant(t, ts, http.StatusAccepted, http.MethodPost, "/agents/"+id+"/", "")
	base := "/agents/" + id + "/agentAliases/TSTALIASID"

	barStream(t, ts, http.MethodPost, base+"/sessions/m-1/text", map[string]any{"inputText": "remember my favourite colour is teal", "memoryId": "user-1"})
	barStream(t, ts, http.MethodPost, base+"/sessions/m-1/text", map[string]any{"inputText": "thanks", "memoryId": "user-1", "endSession": true})
	mem := barJSON(t, ts, http.StatusOK, http.MethodGet, base+"/memories?memoryId=user-1&memoryType=SESSION_SUMMARY", nil)["memoryContents"].([]any)
	if len(mem) != 1 || !strings.Contains(mem[0].(map[string]any)["sessionSummary"].(map[string]any)["summaryText"].(string), "teal") {
		t.Fatalf("expected one session summary, got %v", mem)
	}
	barJSON(t, ts, http.StatusBadRequest, http.MethodGet, base+"/memories?memoryId=user-1&memoryType=OTHER", nil)
	barJSON(t, ts, http.StatusNotFound, http.MethodGet, "/agents/"+id+"/agentAliases/NOPE000000/memories?memoryId=user-1&memoryType=SESSION_SUMMARY", nil)
	barJSON(t, ts, http.StatusAccepted, http.MethodDelete, base+"/memories?memoryId=user-1", nil)
	if mem := barJSON(t, ts, http.StatusOK, http.MethodGet, base+"/memories?memoryId=user-1&memoryType=SESSION_SUMMARY", nil)["memoryContents"].([]any); len(mem) != 0 {
		t.Fatalf("memory should be deleted, got %v", mem)
	}
}

func TestBedrockAgentRuntimeStage1RetrieveAndGenerate(t *testing.T) {
	ts := newBedrockAgentTestServer(t)
	fx := barNewKB(t, ts)

	res := barJSON(t, ts, http.StatusOK, http.MethodPost, "/knowledgebases/"+fx.kb+"/retrieve", map[string]any{"retrievalQuery": map[string]any{"text": "shipping country"}})["retrievalResults"].([]any)
	if len(res) != 1 || res[0].(map[string]any)["documentId"] != "shipping" {
		t.Fatalf("expected the shipping document, got %v", res)
	}
	// A metadata filter that excludes the only match.
	filtered := barJSON(t, ts, http.StatusOK, http.MethodPost, "/knowledgebases/"+fx.kb+"/retrieve", map[string]any{
		"retrievalQuery":         map[string]any{"text": "shipping country"},
		"retrievalConfiguration": map[string]any{"vectorSearchConfiguration": map[string]any{"filter": map[string]any{"equals": map[string]any{"key": "team", "value": "billing"}}}},
	})["retrievalResults"].([]any)
	if len(filtered) != 0 {
		t.Fatalf("filter should exclude logistics documents, got %v", filtered)
	}
	barJSON(t, ts, http.StatusBadRequest, http.MethodPost, "/knowledgebases/"+fx.kb+"/retrieve", map[string]any{"retrievalQuery": map[string]any{"text": "x"}, "retrievalConfiguration": map[string]any{"vectorSearchConfiguration": map[string]any{"numberOfResults": 101}}})
	barJSON(t, ts, http.StatusNotFound, http.MethodPost, "/knowledgebases/NOPE000000/retrieve", map[string]any{"retrievalQuery": map[string]any{"text": "x"}})

	cfg := map[string]any{"type": "KNOWLEDGE_BASE", "knowledgeBaseConfiguration": map[string]any{"knowledgeBaseId": fx.kb, "modelArn": "anthropic.claude-v2"}}
	rag := barJSON(t, ts, http.StatusOK, http.MethodPost, "/retrieveAndGenerate", map[string]any{"input": map[string]any{"text": "When are refunds issued?"}, "retrieveAndGenerateConfiguration": cfg})
	session := rag["sessionId"].(string)
	if !strings.Contains(rag["output"].(map[string]any)["text"].(string), "14 days") || len(rag["citations"].([]any)) != 1 {
		t.Fatalf("unexpected RAG reply: %v", rag)
	}
	// The session keeps its configuration.
	again := barJSON(t, ts, http.StatusOK, http.MethodPost, "/retrieveAndGenerate", map[string]any{"input": map[string]any{"text": "zebra"}, "sessionId": session})
	if again["sessionId"] != session || again["output"].(map[string]any)["text"] != barNoAnswer {
		t.Fatalf("expected the no-answer reply in the same session: %v", again)
	}
	barJSON(t, ts, http.StatusNotFound, http.MethodPost, "/retrieveAndGenerate", map[string]any{"input": map[string]any{"text": "x"}, "sessionId": "missing"})
	barJSON(t, ts, http.StatusBadRequest, http.MethodPost, "/retrieveAndGenerate", map[string]any{"input": map[string]any{"text": "x"}})
	barJSON(t, ts, http.StatusBadRequest, http.MethodPost, "/retrieveAndGenerate", map[string]any{"input": map[string]any{"text": strings.Repeat("a", 1001)}, "retrieveAndGenerateConfiguration": cfg})

	hdr, types, _ := barStream(t, ts, http.MethodPost, "/retrieveAndGenerateStream", map[string]any{"input": map[string]any{"text": "refunds"}, "retrieveAndGenerateConfiguration": cfg})
	if hdr.Get("x-amzn-bedrock-knowledge-base-session-id") == "" || strings.Join(types, ",") != "output,citation" {
		t.Fatalf("unexpected stream: %v %v", hdr, types)
	}

	ext := map[string]any{"type": "EXTERNAL_SOURCES", "externalSourcesConfiguration": map[string]any{"modelArn": "anthropic.claude-v2", "sources": []any{
		map[string]any{"sourceType": "BYTE_CONTENT", "byteContent": map[string]any{"identifier": "faq", "contentType": "text/plain", "data": base64.StdEncoding.EncodeToString([]byte("Our office opens at nine."))}},
	}}}
	out := barJSON(t, ts, http.StatusOK, http.MethodPost, "/retrieveAndGenerate", map[string]any{"input": map[string]any{"text": "When does the office open?"}, "retrieveAndGenerateConfiguration": ext})
	if !strings.Contains(out["output"].(map[string]any)["text"].(string), "nine") {
		t.Fatalf("external source should answer: %v", out)
	}
}

func TestBedrockAgentRuntimeStage1RerankQueryPromptDocuments(t *testing.T) {
	ts := newBedrockAgentTestServer(t)
	fx := barNewKB(t, ts)

	rr := barJSON(t, ts, http.StatusOK, http.MethodPost, "/rerank", map[string]any{
		"queries": []any{map[string]any{"type": "TEXT", "textQuery": map[string]any{"text": "refund policy"}}},
		"sources": []any{
			map[string]any{"type": "INLINE", "inlineDocumentSource": map[string]any{"type": "TEXT", "textDocument": map[string]any{"text": "shipping times"}}},
			map[string]any{"type": "INLINE", "inlineDocumentSource": map[string]any{"type": "TEXT", "textDocument": map[string]any{"text": "our refund policy"}}},
		},
		"rerankingConfiguration": map[string]any{"type": "BEDROCK_RERANKING_MODEL", "bedrockRerankingConfiguration": map[string]any{"numberOfResults": 1, "modelConfiguration": map[string]any{"modelArn": "arn:aws:bedrock:us-east-1::foundation-model/amazon.rerank-v1:0"}}},
	})["results"].([]any)
	if len(rr) != 1 || rr[0].(map[string]any)["index"] != float64(1) {
		t.Fatalf("the refund source should rank first: %v", rr)
	}
	barJSON(t, ts, http.StatusBadRequest, http.MethodPost, "/rerank", map[string]any{"queries": []any{}, "sources": []any{}})

	kbARN := "arn:aws:bedrock:us-east-1:123456789012:knowledge-base/" + fx.kb
	gq := barJSON(t, ts, http.StatusOK, http.MethodPost, "/generateQuery", map[string]any{
		"queryGenerationInput":        map[string]any{"type": "TEXT", "text": "How many orders shipped?"},
		"transformationConfiguration": map[string]any{"mode": "TEXT_TO_SQL", "textToSqlConfiguration": map[string]any{"type": "KNOWLEDGE_BASE", "knowledgeBaseConfiguration": map[string]any{"knowledgeBaseArn": kbARN}}},
	})["queries"].([]any)
	if gq[0].(map[string]any)["type"] != "REDSHIFT_SQL" {
		t.Fatalf("unexpected generated query: %v", gq)
	}

	_, types, payloads := barStream(t, ts, http.MethodPost, "/optimize-prompt", map[string]any{"input": map[string]any{"textPrompt": map[string]any{"text": "summarize this"}}, "targetModelId": "anthropic.claude-v2"})
	if strings.Join(types, ",") != "analyzePromptEvent,optimizedPromptEvent" || !strings.Contains(fmt.Sprint(payloads[1]), "summarize this") {
		t.Fatalf("unexpected optimize stream: %v %v", types, payloads)
	}

	_, types, payloads = barStream(t, ts, http.MethodPost, "/agenticRetrieveStream", map[string]any{
		"messages":                     []any{map[string]any{"role": "user", "content": map[string]any{"text": "refunds"}}},
		"retrievers":                   []any{map[string]any{"configuration": map[string]any{"knowledgeBase": map[string]any{"knowledgeBaseId": fx.kb}}}},
		"agenticRetrieveConfiguration": map[string]any{},
		"generateResponse":             true,
	})
	if types[len(types)-1] != "result" || len(payloads[len(payloads)-1]["results"].([]any)) != 1 {
		t.Fatalf("unexpected agentic retrieve stream: %v %v", types, payloads)
	}

	// Document content links serve the text with a safe content type.
	doc := "/knowledgebases/" + fx.kb + "/datasources/" + fx.ds + "/documents/refunds/content"
	content := barJSON(t, ts, http.StatusOK, http.MethodPost, doc, map[string]any{})
	resp, err := http.Get(content["presignedUrl"].(string))
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), "14 days") || resp.Header.Get("Content-Type") != "text/plain; charset=utf-8" || resp.Header.Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("document link: %d %v %q", resp.StatusCode, resp.Header, body)
	}
	if resp, _ := http.Get(ts.URL + barDocumentPath + "bogus"); resp.StatusCode != http.StatusForbidden {
		t.Fatalf("a bad link must be refused, got %d", resp.StatusCode)
	}
	barJSON(t, ts, http.StatusNotFound, http.MethodPost, "/knowledgebases/"+fx.kb+"/datasources/"+fx.ds+"/documents/missing/content", map[string]any{})

	acl := "/knowledgebases/" + fx.kb + "/datasources/" + fx.ds
	if out := barJSON(t, ts, http.StatusOK, http.MethodPost, acl+"/check-ingested-document-acl", map[string]any{"documentId": "refunds", "userContext": map[string]any{"userId": "u1"}}); out["hasAccess"] != true {
		t.Fatalf("expected access: %v", out)
	}
	barJSON(t, ts, http.StatusBadRequest, http.MethodPost, acl+"/check-ingested-document-acl", map[string]any{"documentId": "refunds"})
	barJSON(t, ts, http.StatusOK, http.MethodPost, acl+"/get-ingested-document-acl", map[string]any{"documentId": "refunds"})
}
