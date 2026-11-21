package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream"
)

func sagemakerRuntimeCreateEndpoint(t *testing.T, ts *httptest.Server, name, config string) {
	t.Helper()
	resp := sagemakerRequest(t, ts, "CreateEndpointConfig", `{"EndpointConfigName":"`+name+`-config",`+config+`}`)
	assertStatus(t, resp, http.StatusOK)
	_ = mustBody(t, resp)
	resp = sagemakerRequest(t, ts, "CreateEndpoint", `{"EndpointName":"`+name+`","EndpointConfigName":"`+name+`-config"}`)
	assertStatus(t, resp, http.StatusOK)
	_ = mustBody(t, resp)
}

func sagemakerRuntimeExpectError(t *testing.T, resp *http.Response, status int, code, fragment string) {
	t.Helper()
	assertStatus(t, resp, status)
	var body struct {
		Type    string `json:"__type"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &body); err != nil {
		t.Fatalf("decode error body: %v", err)
	}
	if body.Type != code || resp.Header.Get("X-Amzn-ErrorType") != code || !strings.Contains(body.Message, fragment) {
		t.Fatalf("error = %+v (header %q), want %s containing %q", body, resp.Header.Get("X-Amzn-ErrorType"), code, fragment)
	}
}

func TestSageMakerRuntimeStage1InvokeEndpoint(t *testing.T) {
	_, ts := newSageMakerRuntimeTestServer(t)
	sagemakerRuntimeCreateEndpoint(t, ts, "multi", `"ProductionVariants":[{"VariantName":"blue","ModelName":"stackyard-model"},{"VariantName":"green","ModelName":"stackyard-model"}]`)

	// The seeded endpoint echoes the body with the Accept type and passes attributes through.
	resp := sagemakerRuntimeRequest(t, ts, "stackyard-endpoint", "invocations", []byte(`{"inputs":"hi"}`), map[string]string{
		"Content-Type":                       "application/json",
		"Accept":                             "application/jsonlines",
		"X-Amzn-SageMaker-Custom-Attributes": "trace=1",
	})
	assertStatus(t, resp, http.StatusOK)
	if got := string(mustBody(t, resp)); got != `{"inputs":"hi"}` {
		t.Fatalf("body = %q", got)
	}
	if resp.Header.Get("Content-Type") != "application/jsonlines" || resp.Header.Get("x-Amzn-Invoked-Production-Variant") != "AllTraffic" || resp.Header.Get("X-Amzn-SageMaker-Custom-Attributes") != "trace=1" {
		t.Fatalf("headers = %v", resp.Header)
	}

	// Routing follows the endpoint config: first variant by default, TargetVariant when sent.
	resp = sagemakerRuntimeRequest(t, ts, "multi", "invocations", []byte("x"), nil)
	assertStatus(t, resp, http.StatusOK)
	if v := resp.Header.Get("x-Amzn-Invoked-Production-Variant"); v != "blue" {
		t.Fatalf("default variant = %q", v)
	}
	_ = mustBody(t, resp)
	resp = sagemakerRuntimeRequest(t, ts, "multi", "invocations", []byte("x"), map[string]string{"X-Amzn-SageMaker-Target-Variant": "green"})
	assertStatus(t, resp, http.StatusOK)
	if v := resp.Header.Get("x-Amzn-Invoked-Production-Variant"); v != "green" {
		t.Fatalf("target variant = %q", v)
	}
	_ = mustBody(t, resp)

	// DescribeEndpointConfig reports the stored variants.
	resp = sagemakerRequest(t, ts, "DescribeEndpointConfig", `{"EndpointConfigName":"multi-config"}`)
	if body := string(mustBody(t, resp)); !strings.Contains(body, `"green"`) {
		t.Fatalf("DescribeEndpointConfig = %s", body)
	}

	// Stateful sessions: NEW_SESSION opens one; an unknown ID is rejected.
	resp = sagemakerRuntimeRequest(t, ts, "multi", "invocations", []byte("x"), map[string]string{"X-Amzn-SageMaker-Session-Id": "NEW_SESSION"})
	assertStatus(t, resp, http.StatusOK)
	session, _, _ := strings.Cut(resp.Header.Get("X-Amzn-SageMaker-New-Session-Id"), ";")
	_ = mustBody(t, resp)
	if session == "" {
		t.Fatalf("no new session id")
	}
	resp = sagemakerRuntimeRequest(t, ts, "multi", "invocations", []byte("x"), map[string]string{"X-Amzn-SageMaker-Session-Id": session})
	assertStatus(t, resp, http.StatusOK)
	_ = mustBody(t, resp)
	resp = sagemakerRuntimeRequest(t, ts, "stackyard-endpoint", "invocations", []byte("x"), map[string]string{"X-Amzn-SageMaker-Session-Id": session})
	sagemakerRuntimeExpectError(t, resp, http.StatusBadRequest, "ValidationError", "Session")
}

func TestSageMakerRuntimeStage1Validation(t *testing.T) {
	srv, ts := newSageMakerRuntimeTestServer(t)
	sagemakerRuntimeCreateEndpoint(t, ts, "async-ep", `"AsyncInferenceConfig":{"OutputConfig":{"S3OutputPath":"s3://out/results"}}`)

	cases := []struct {
		name, endpoint, suffix string
		body                   []byte
		headers                map[string]string
		fragment               string
	}{
		{"missing endpoint", "nope", "invocations", []byte("x"), nil, "Endpoint nope of account"},
		{"bad endpoint name", "-bad", "invocations", []byte("x"), nil, "endpointName"},
		{"unknown variant", "stackyard-endpoint", "invocations", []byte("x"), map[string]string{"X-Amzn-SageMaker-Target-Variant": "red"}, "Variant red"},
		{"bad variant pattern", "stackyard-endpoint", "invocations", []byte("x"), map[string]string{"X-Amzn-SageMaker-Target-Variant": "a_b"}, "Target-Variant"},
		{"bad inference component", "stackyard-endpoint", "invocations", []byte("x"), map[string]string{"X-Amzn-SageMaker-Inference-Component": "-x"}, "Inference-Component"},
		{"bad session", "stackyard-endpoint", "invocations", []byte("x"), map[string]string{"X-Amzn-SageMaker-Session-Id": "a_b"}, "sessionId"},
		{"body too large", "stackyard-endpoint", "invocations", bytes.Repeat([]byte("a"), sagemakerRuntimeMaxBody+1), nil, "6 MB"},
		{"realtime on async endpoint", "async-ep", "invocations", []byte("x"), nil, "use InvokeEndpointAsync"},
		{"async on realtime endpoint", "stackyard-endpoint", "async-invocations", nil, map[string]string{"X-Amzn-SageMaker-InputLocation": "s3://in/a.json"}, "not configured for asynchronous"},
		{"async needs input", "async-ep", "async-invocations", nil, nil, "InputLocation is required"},
		{"async bad location", "async-ep", "async-invocations", nil, map[string]string{"X-Amzn-SageMaker-InputLocation": "ftp://x/y"}, "InputLocation"},
		{"async inline too large", "async-ep", "async-invocations", bytes.Repeat([]byte("a"), sagemakerRuntimeMaxAsyncBody+1), nil, "128000"},
		{"async ttl", "async-ep", "async-invocations", []byte("x"), map[string]string{"X-Amzn-SageMaker-RequestTTLSeconds": "59"}, "RequestTTLSeconds"},
		{"async timeout", "async-ep", "async-invocations", []byte("x"), map[string]string{"X-Amzn-SageMaker-InvocationTimeoutSeconds": "3601"}, "InvocationTimeoutSeconds"},
		{"async filename", "async-ep", "async-invocations", []byte("x"), map[string]string{"X-Amzn-SageMaker-Filename": "a.b.c"}, "Filename"},
		{"async extension", "async-ep", "async-invocations", []byte("x"), map[string]string{"X-Amzn-SageMaker-S3OutputPathExtension": "s3:x"}, "S3OutputPathExtension"},
		{"stream unknown endpoint", "nope", "invocations-response-stream", []byte("x"), nil, "not found"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp := sagemakerRuntimeRequest(t, ts, tc.endpoint, tc.suffix, tc.body, tc.headers)
			sagemakerRuntimeExpectError(t, resp, http.StatusBadRequest, "ValidationError", tc.fragment)
		})
	}

	// An endpoint that is not InService cannot be invoked.
	srv.sagemaker.mu.Lock()
	srv.sagemaker.endpoints["stackyard-endpoint"].Status = "Failed"
	srv.sagemaker.mu.Unlock()
	resp := sagemakerRuntimeRequest(t, ts, "stackyard-endpoint", "invocations", []byte("x"), nil)
	sagemakerRuntimeExpectError(t, resp, http.StatusBadRequest, "ValidationError", "not in service")
}

func TestSageMakerRuntimeStage1Async(t *testing.T) {
	srv, ts := newSageMakerRuntimeTestServer(t)
	sagemakerRuntimeCreateEndpoint(t, ts, "async-ep", `"AsyncInferenceConfig":{"OutputConfig":{"S3OutputPath":"s3://out/results/","S3FailurePath":"s3://out/failures"}}`)
	for _, b := range []string{"in", "out"} {
		if _, err := srv.s3.CreateBucket(b); err != nil {
			t.Fatalf("create bucket %s: %v", b, err)
		}
	}
	if _, err := srv.s3.PutObject("in", "req.json", "application/json", []byte(`{"q":1}`), nil, "private", "STANDARD"); err != nil {
		t.Fatalf("put input: %v", err)
	}

	resp := sagemakerRuntimeRequest(t, ts, "async-ep", "async-invocations", nil, map[string]string{
		"X-Amzn-SageMaker-InputLocation":         "s3://in/req.json",
		"X-Amzn-SageMaker-Inference-Id":          "inf-1",
		"X-Amzn-SageMaker-S3OutputPathExtension": "batch",
		"X-Amzn-SageMaker-Accept":                "application/json",
	})
	assertStatus(t, resp, http.StatusAccepted)
	var out struct{ InferenceId string }
	if err := json.Unmarshal(mustBody(t, resp), &out); err != nil || out.InferenceId != "inf-1" {
		t.Fatalf("InferenceId = %q, err %v", out.InferenceId, err)
	}
	if got := resp.Header.Get("X-Amzn-SageMaker-OutputLocation"); got != "s3://out/results/batch/inf-1.out" {
		t.Fatalf("OutputLocation = %q", got)
	}
	if got := resp.Header.Get("X-Amzn-SageMaker-FailureLocation"); got != "s3://out/failures/batch/inf-1-error.out" {
		t.Fatalf("FailureLocation = %q", got)
	}
	obj, err := srv.s3.GetObject("out", "results/batch/inf-1.out")
	if err != nil || string(obj.Body) != `{"q":1}` || obj.ContentType != "application/json" {
		t.Fatalf("output object = %q (%s), err %v", obj.Body, obj.ContentType, err)
	}

	// Inline body plus Filename; a generated InferenceId when none is sent.
	resp = sagemakerRuntimeRequest(t, ts, "async-ep", "async-invocations", []byte("inline"), map[string]string{"X-Amzn-SageMaker-Filename": "r.txt"})
	assertStatus(t, resp, http.StatusAccepted)
	if err := json.Unmarshal(mustBody(t, resp), &out); err != nil || out.InferenceId == "" {
		t.Fatalf("generated InferenceId missing: %v", err)
	}
	if obj, err := srv.s3.GetObject("out", "results/r.txt"); err != nil || string(obj.Body) != "inline" {
		t.Fatalf("inline output = %q, err %v", obj.Body, err)
	}

	// A missing input object produces a failure object, not an output.
	resp = sagemakerRuntimeRequest(t, ts, "async-ep", "async-invocations", nil, map[string]string{
		"X-Amzn-SageMaker-InputLocation": "s3://in/missing.json",
		"X-Amzn-SageMaker-Inference-Id":  "inf-2",
	})
	assertStatus(t, resp, http.StatusAccepted)
	_ = mustBody(t, resp)
	if _, err := srv.s3.GetObject("out", "results/inf-2.out"); err == nil {
		t.Fatalf("output written for missing input")
	}
	if obj, err := srv.s3.GetObject("out", "failures/inf-2-error.out"); err != nil || !strings.Contains(string(obj.Body), "missing.json") {
		t.Fatalf("failure object = %q, err %v", obj.Body, err)
	}
}

func TestSageMakerRuntimeStage1ResponseStream(t *testing.T) {
	_, ts := newSageMakerRuntimeTestServer(t)
	payload := bytes.Repeat([]byte("0123456789"), 60) // 600 bytes → 3 parts
	resp := sagemakerRuntimeRequest(t, ts, "stackyard-endpoint", "invocations-response-stream", payload, map[string]string{
		"Content-Type":            "text/plain",
		"X-Amzn-SageMaker-Accept": "application/x-text",
	})
	assertStatus(t, resp, http.StatusOK)
	if resp.Header.Get("Content-Type") != "application/vnd.amazon.eventstream" || resp.Header.Get("X-Amzn-SageMaker-Content-Type") != "application/x-text" || resp.Header.Get("x-Amzn-Invoked-Production-Variant") != "AllTraffic" {
		t.Fatalf("headers = %v", resp.Header)
	}
	raw := mustBody(t, resp)
	dec := eventstream.NewDecoder()
	reader := bytes.NewReader(raw)
	var joined []byte
	parts := 0
	for {
		msg, err := dec.Decode(reader, nil)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("decode event %d: %v", parts, err)
		}
		if et := msg.Headers.Get(":event-type"); et == nil || et.String() != "PayloadPart" {
			t.Fatalf("event-type = %v", et)
		}
		joined = append(joined, msg.Payload...)
		parts++
	}
	if parts != 3 || !bytes.Equal(joined, payload) {
		t.Fatalf("got %d parts, payload match %v", parts, bytes.Equal(joined, payload))
	}

	// An empty body still yields one (empty) PayloadPart.
	resp = sagemakerRuntimeRequest(t, ts, "stackyard-endpoint", "invocations-response-stream", nil, nil)
	assertStatus(t, resp, http.StatusOK)
	msg, err := dec.Decode(bytes.NewReader(mustBody(t, resp)), nil)
	if err != nil || len(msg.Payload) != 0 {
		t.Fatalf("empty stream: %v %q", err, msg.Payload)
	}
}
