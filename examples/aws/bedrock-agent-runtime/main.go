// Command bedrock-agent-runtime exercises Agents for Amazon Bedrock Runtime
// against Stackyard: it builds a knowledge base and an agent with the
// bedrock-agent control plane, then retrieves, generates, invokes the agent
// (an event stream) and runs a Bedrock session.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

const roleArn = "arn:aws:iam::123456789012:role/bedrock-agent-runtime-example"

type client struct {
	endpoint, region string
	creds            aws.CredentialsProvider
	signer           *v4.Signer
}

func main() {
	c := client{
		endpoint: strings.TrimRight(getenv("STACKYARD_ENDPOINT", "http://localhost:4566"), "/"),
		region:   getenv("AWS_REGION", "us-east-1"),
		creds:    credentials.NewStaticCredentialsProvider(getenv("AWS_ACCESS_KEY_ID", "stackyard"), getenv("AWS_SECRET_ACCESS_KEY", "stackyard"), ""),
		signer:   v4.NewSigner(),
	}
	suffix := fmt.Sprint(time.Now().UnixNano())
	fmt.Printf("Agents for Amazon Bedrock Runtime example using %s\n", c.endpoint)

	// Control plane: a knowledge base with one document and an agent that uses it.
	kbID := c.call("CreateKnowledgeBase", http.MethodPut, "/knowledgebases/", map[string]any{
		"name": "docs" + suffix, "roleArn": roleArn,
		"knowledgeBaseConfiguration": map[string]any{
			"type":                             "VECTOR",
			"vectorKnowledgeBaseConfiguration": map[string]any{"embeddingModelArn": "arn:aws:bedrock:us-east-1::foundation-model/amazon.titan-embed-text-v2:0"},
		},
		"storageConfiguration": map[string]any{"type": "S3_VECTORS"},
	})["knowledgeBase"].(map[string]any)["knowledgeBaseId"].(string)
	dsID := c.call("CreateDataSource", http.MethodPut, "/knowledgebases/"+kbID+"/datasources/", map[string]any{
		"name": "inline", "dataSourceConfiguration": map[string]any{"type": "CUSTOM"},
	})["dataSource"].(map[string]any)["dataSourceId"].(string)
	c.call("IngestKnowledgeBaseDocuments", http.MethodPut, "/knowledgebases/"+kbID+"/datasources/"+dsID+"/documents", map[string]any{
		"documents": []any{map[string]any{"content": map[string]any{
			"dataSourceType": "CUSTOM",
			"custom": map[string]any{
				"customDocumentIdentifier": map[string]any{"id": "faq"},
				"sourceType":               "IN_LINE",
				"inlineContent":            map[string]any{"type": "TEXT", "textContent": map[string]any{"data": "Stackyard runs AWS APIs locally for tests."}},
			},
		}}},
	})
	agentID := c.call("CreateAgent", http.MethodPut, "/agents/", map[string]any{
		"agentName": "helper" + suffix, "foundationModel": "anthropic.claude-3-haiku-20240307-v1:0",
		"agentResourceRoleArn": roleArn, "instruction": "You answer questions about Stackyard using the knowledge base.",
	})["agent"].(map[string]any)["agentId"].(string)
	c.call("AssociateAgentKnowledgeBase", http.MethodPut, "/agents/"+agentID+"/agentversions/DRAFT/knowledgebases/", map[string]any{
		"knowledgeBaseId": kbID, "description": "Stackyard FAQ",
	})
	c.call("PrepareAgent", http.MethodPost, "/agents/"+agentID+"/", nil)
	aliasID := c.call("CreateAgentAlias", http.MethodPut, "/agents/"+agentID+"/agentaliases/", map[string]any{"agentAliasName": "live"})["agentAlias"].(map[string]any)["agentAliasId"].(string)

	// Runtime: retrieve, retrieve-and-generate, then invoke the agent.
	results := c.call("Retrieve", http.MethodPost, "/knowledgebases/"+kbID+"/retrieve", map[string]any{
		"retrievalQuery": map[string]any{"text": "What does Stackyard run?"},
	})["retrievalResults"].([]any)
	if len(results) != 1 {
		exitf("expected one retrieval result, got %d", len(results))
	}
	rag := c.call("RetrieveAndGenerate", http.MethodPost, "/retrieveAndGenerate", map[string]any{
		"input": map[string]any{"text": "What does Stackyard run?"},
		"retrieveAndGenerateConfiguration": map[string]any{"type": "KNOWLEDGE_BASE", "knowledgeBaseConfiguration": map[string]any{
			"knowledgeBaseId": kbID, "modelArn": "arn:aws:bedrock:us-east-1::foundation-model/anthropic.claude-3-haiku-20240307-v1:0",
		}},
	})
	fmt.Printf("generated: %v\n", rag["output"].(map[string]any)["text"])

	stream := c.send("InvokeAgent", http.MethodPost, "/agents/"+agentID+"/agentAliases/"+aliasID+"/sessions/example-session/text", map[string]any{
		"inputText": "What does Stackyard run?",
	})
	answer := ""
	dec := eventstream.NewDecoder()
	for r := bytes.NewReader(stream); r.Len() > 0; {
		msg, err := dec.Decode(r, nil)
		if err != nil {
			exitf("InvokeAgent: decode event: %v", err)
		}
		if msg.Headers.Get(":event-type").String() == "chunk" {
			var chunk struct{ Bytes []byte }
			_ = json.Unmarshal(msg.Payload, &chunk)
			answer += string(chunk.Bytes)
		}
	}
	if !strings.Contains(answer, "AWS APIs") {
		exitf("the agent should answer from the knowledge base, got %q", answer)
	}
	fmt.Printf("agent answered: %s\n", answer)

	// A Bedrock session with one invocation and step.
	session := c.call("CreateSession", http.MethodPut, "/sessions/", map[string]any{})["sessionId"].(string)
	invocation := c.call("CreateInvocation", http.MethodPut, "/sessions/"+session+"/invocations/", map[string]any{})["invocationId"].(string)
	c.call("PutInvocationStep", http.MethodPut, "/sessions/"+session+"/invocationSteps/", map[string]any{
		"invocationIdentifier": invocation, "invocationStepTime": time.Now().UTC().Format(time.RFC3339),
		"payload": map[string]any{"contentBlocks": []any{map[string]any{"text": answer}}},
	})
	c.call("EndSession", http.MethodPatch, "/sessions/"+session, nil)
	c.call("DeleteSession", http.MethodDelete, "/sessions/"+session+"/", nil)

	c.call("DeleteAgent", http.MethodDelete, "/agents/"+agentID+"/?skipResourceInUseCheck=true", nil)
	c.call("DeleteKnowledgeBase", http.MethodDelete, "/knowledgebases/"+kbID, nil)
	fmt.Println("done")
}

// call sends one signed REST JSON request and exits unless it succeeds.
func (c client) call(op, method, path string, body any) map[string]any {
	data := c.send(op, method, path, body)
	out := map[string]any{}
	if len(data) > 0 {
		if err := json.Unmarshal(data, &out); err != nil {
			exitf("%s: decode: %v", op, err)
		}
	}
	return out
}

// send signs and sends one request and returns the raw body of a 2xx reply.
func (c client) send(op, method, path string, body any) []byte {
	var raw []byte
	if body != nil {
		raw, _ = json.Marshal(body)
	}
	req, err := http.NewRequest(method, c.endpoint+path, bytes.NewReader(raw))
	if err != nil {
		exitf("%s: %v", op, err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	sum := sha256.Sum256(raw)
	hash := hex.EncodeToString(sum[:])
	req.Header.Set("X-Amz-Content-Sha256", hash)
	creds, err := c.creds.Retrieve(context.Background())
	if err != nil {
		exitf("%s: credentials: %v", op, err)
	}
	if err := c.signer.SignHTTP(context.Background(), creds, req, hash, "bedrock", c.region, time.Now()); err != nil {
		exitf("%s: sign: %v", op, err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		exitf("%s: %v", op, err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		exitf("%s: HTTP %d: %s", op, resp.StatusCode, data)
	}
	fmt.Printf("%s: HTTP %d\n", op, resp.StatusCode)
	return data
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
