// Command bedrock-agent walks the Agents for Amazon Bedrock lifecycle against
// Stackyard: a knowledge base with an ingested document, an agent that uses
// it, PrepareAgent, and an alias that snapshots version 1.
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
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

const roleArn = "arn:aws:iam::123456789012:role/bedrock-agent-example"

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
	fmt.Printf("Agents for Amazon Bedrock example using %s\n", c.endpoint)

	kb := c.call("CreateKnowledgeBase", http.MethodPut, "/knowledgebases/", map[string]any{
		"name":    "docs" + suffix,
		"roleArn": roleArn,
		"knowledgeBaseConfiguration": map[string]any{
			"type":                             "VECTOR",
			"vectorKnowledgeBaseConfiguration": map[string]any{"embeddingModelArn": "arn:aws:bedrock:us-east-1::foundation-model/amazon.titan-embed-text-v2:0"},
		},
		"storageConfiguration": map[string]any{"type": "S3_VECTORS"},
	})["knowledgeBase"].(map[string]any)
	kbID := kb["knowledgeBaseId"].(string)

	ds := c.call("CreateDataSource", http.MethodPut, "/knowledgebases/"+kbID+"/datasources/", map[string]any{
		"name": "inline", "dataSourceConfiguration": map[string]any{"type": "CUSTOM"},
	})["dataSource"].(map[string]any)
	dsPath := "/knowledgebases/" + kbID + "/datasources/" + ds["dataSourceId"].(string)
	c.call("IngestKnowledgeBaseDocuments", http.MethodPut, dsPath+"/documents", map[string]any{
		"documents": []any{map[string]any{"content": map[string]any{
			"dataSourceType": "CUSTOM",
			"custom": map[string]any{
				"customDocumentIdentifier": map[string]any{"id": "faq"},
				"sourceType":               "IN_LINE",
				"inlineContent":            map[string]any{"type": "TEXT", "textContent": map[string]any{"data": "Stackyard runs AWS APIs locally."}},
			},
		}}},
	})

	agent := c.call("CreateAgent", http.MethodPut, "/agents/", map[string]any{
		"agentName":            "helper" + suffix,
		"foundationModel":      "anthropic.claude-3-haiku-20240307-v1:0",
		"agentResourceRoleArn": roleArn,
		"instruction":          "You answer questions about Stackyard using the knowledge base.",
	})["agent"].(map[string]any)
	agentID := agent["agentId"].(string)
	c.call("AssociateAgentKnowledgeBase", http.MethodPut, "/agents/"+agentID+"/agentversions/DRAFT/knowledgebases/", map[string]any{
		"knowledgeBaseId": kbID, "description": "Stackyard FAQ",
	})

	prepared := c.call("PrepareAgent", http.MethodPost, "/agents/"+agentID+"/", nil)
	fmt.Printf("prepared agent %s: %v\n", agentID, prepared["agentStatus"])
	alias := c.call("CreateAgentAlias", http.MethodPut, "/agents/"+agentID+"/agentaliases/", map[string]any{"agentAliasName": "live"})["agentAlias"].(map[string]any)
	routes := alias["routingConfiguration"].([]any)
	fmt.Printf("alias %s routes to version %v\n", alias["agentAliasId"], routes[0].(map[string]any)["agentVersion"])

	version := c.call("GetAgentVersion", http.MethodGet, "/agents/"+agentID+"/agentversions/1/", nil)["agentVersion"].(map[string]any)
	if version["agentStatus"] != "PREPARED" {
		exitf("version 1 should be PREPARED, got %v", version["agentStatus"])
	}

	c.call("DeleteAgentAlias", http.MethodDelete, "/agents/"+agentID+"/agentaliases/"+alias["agentAliasId"].(string)+"/", nil)
	c.call("DeleteAgent", http.MethodDelete, "/agents/"+agentID+"/?skipResourceInUseCheck=true", nil)
	c.call("DeleteKnowledgeBase", http.MethodDelete, "/knowledgebases/"+kbID, nil)
	fmt.Println("done")
}

// call sends one signed REST JSON request and exits unless it succeeds.
func (c client) call(op, method, path string, body any) map[string]any {
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
	out := map[string]any{}
	if len(data) > 0 {
		if err := json.Unmarshal(data, &out); err != nil {
			exitf("%s: decode: %v", op, err)
		}
	}
	fmt.Printf("%s: HTTP %d\n", op, resp.StatusCode)
	return out
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
