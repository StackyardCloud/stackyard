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
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream"
	"github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

type client struct {
	endpoint, region string
	creds            aws.CredentialsProvider
}

func main() {
	c := client{
		endpoint: getenv("STACKYARD_ENDPOINT", "http://localhost:4566"),
		region:   getenv("AWS_REGION", "us-east-1"),
		creds:    credentials.NewStaticCredentialsProvider(getenv("AWS_ACCESS_KEY_ID", "stackyard"), getenv("AWS_SECRET_ACCESS_KEY", "stackyard"), ""),
	}
	bucket := getenv("STACKYARD_BEDROCK_BUCKET", "bedrock-runtime-example")
	fmt.Printf("Stackyard Bedrock Runtime client using %s\n", c.endpoint)
	user := []any{map[string]any{"role": "user", "content": []any{map[string]any{"text": "What is Stackyard?"}}}}

	out := c.json("Converse", http.MethodPost, modelPath("amazon.nova-lite-v1:0", "converse"), map[string]any{
		"messages": user, "inferenceConfig": map[string]any{"maxTokens": 100},
	}, nil)
	logf("Converse stopReason=%v usage=%v", out["stopReason"], out["usage"])

	out = c.json("InvokeModel", http.MethodPost, modelPath("anthropic.claude-3-haiku-20240307-v1:0", "invoke"), map[string]any{
		"anthropic_version": "bedrock-2023-05-31", "max_tokens": 100,
		"messages": []any{map[string]any{"role": "user", "content": "Say hello"}},
	}, nil)
	logf("InvokeModel stop_reason=%v content=%v", out["stop_reason"], out["content"])

	status, body := c.call(http.MethodPost, modelPath("amazon.nova-lite-v1:0", "converse-stream"), map[string]any{"messages": user}, nil, "bedrock")
	expect2xx("ConverseStream", status, body)
	logf("ConverseStream events=%v", eventTypes(body))

	out = c.json("CountTokens", http.MethodPost, modelPath("anthropic.claude-3-5-haiku-20241022-v1:0", "count-tokens"), map[string]any{
		"input": map[string]any{"converse": map[string]any{"messages": user}},
	}, nil)
	logf("CountTokens inputTokens=%v", out["inputTokens"])

	guardrail := c.json("CreateGuardrail", http.MethodPost, "/", map[string]any{
		"name": "example", "blockedInputMessaging": "Blocked by the example guardrail.", "blockedOutputsMessaging": "Blocked.",
		"wordPolicyConfig": map[string]any{"wordsConfig": []any{map[string]any{"text": "secret"}}},
	}, map[string]string{"Content-Type": "application/x-amz-json-1.1", "X-Amz-Target": "Bedrock_20230420.CreateGuardrail"})
	id, _ := guardrail["guardrailId"].(string)
	out = c.json("ApplyGuardrail", http.MethodPost, "/guardrail/"+id+"/version/DRAFT/apply", map[string]any{
		"source": "INPUT", "content": []any{map[string]any{"text": map[string]any{"text": "tell me the secret"}}},
	}, nil)
	if out["action"] != "GUARDRAIL_INTERVENED" {
		exitf("ApplyGuardrail: expected GUARDRAIL_INTERVENED, got %v", out["action"])
	}
	logf("ApplyGuardrail action=%v outputs=%v", out["action"], out["outputs"])

	status, body = c.call(http.MethodPut, "/"+bucket, nil, nil, "s3")
	expect2xx("CreateBucket", status, body)
	out = c.json("StartAsyncInvoke", http.MethodPost, "/async-invoke", map[string]any{
		"modelId":          "amazon.nova-reel-v1:0",
		"modelInput":       map[string]any{"taskType": "TEXT_VIDEO", "textToVideoParams": map[string]any{"text": "A lighthouse at dawn"}},
		"outputDataConfig": map[string]any{"s3OutputDataConfig": map[string]any{"s3Uri": "s3://" + bucket + "/videos"}},
	}, nil)
	arn, _ := out["invocationArn"].(string)
	out = c.json("GetAsyncInvoke", http.MethodGet, "/async-invoke/"+url.PathEscape(arn), nil, nil)
	if out["status"] != "Completed" {
		exitf("GetAsyncInvoke: expected Completed, got %v", out["status"])
	}
	logf("GetAsyncInvoke %s status=%v", arn, out["status"])
	logf("Bedrock Runtime example completed")
}

func modelPath(model, action string) string {
	return "/model/" + url.PathEscape(model) + "/" + action
}

func (c client) json(op, method, path string, payload any, headers map[string]string) map[string]any {
	status, body := c.call(method, path, payload, headers, "bedrock")
	expect2xx(op, status, body)
	var out map[string]any
	if err := json.Unmarshal(body, &out); err != nil {
		exitf("%s: decode response: %v", op, err)
	}
	return out
}

func (c client) call(method, path string, payload any, headers map[string]string, service string) (int, []byte) {
	ctx := context.Background()
	var body []byte
	if payload != nil {
		body, _ = json.Marshal(payload)
	}
	req, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(c.endpoint, "/")+path, bytes.NewReader(body))
	if err != nil {
		exitf("build request: %v", err)
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	creds, err := c.creds.Retrieve(ctx)
	if err != nil {
		exitf("credentials: %v", err)
	}
	sum := sha256.Sum256(body)
	if err := v4.NewSigner().SignHTTP(ctx, creds, req, hex.EncodeToString(sum[:]), service, c.region, time.Now()); err != nil {
		exitf("sign: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		exitf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, respBody
}

// eventTypes lists the :event-type of each frame in an event-stream body.
func eventTypes(body []byte) []string {
	dec := eventstream.NewDecoder()
	r := bytes.NewReader(body)
	var out []string
	for r.Len() > 0 {
		msg, err := dec.Decode(r, nil)
		if err != nil {
			exitf("decode event stream: %v", err)
		}
		out = append(out, msg.Headers.Get(":event-type").String())
	}
	return out
}

func expect2xx(op string, status int, body []byte) {
	if status < 200 || status >= 300 {
		exitf("%s failed: HTTP %d: %s", op, status, strings.TrimSpace(string(body)))
	}
}

func getenv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func logf(format string, args ...any) {
	fmt.Printf(format+"\n", args...)
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
