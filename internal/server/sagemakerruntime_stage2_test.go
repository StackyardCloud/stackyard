package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream"
)

// sagemakerRuntimeChunkedPost sends a body with no Content-Length. The body
// limit is enforced before SigV4, so a sagemaker credential scope is enough.
func sagemakerRuntimeChunkedPost(t *testing.T, url string, size int) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, url, io.MultiReader(bytes.NewReader(bytes.Repeat([]byte("a"), size))))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.ContentLength = -1
	req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential="+testAccessKey+"/20250101/us-east-1/sagemaker/aws4_request, SignedHeaders=host, Signature=x")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	return resp
}

func TestSageMakerRuntimeStage2BodyLimits(t *testing.T) {
	_, ts := newSageMakerRuntimeTestServer(t)
	sagemakerRuntimeCreateEndpoint(t, ts, "async-ep", `"AsyncInferenceConfig":{"OutputConfig":{"S3OutputPath":"s3://out/results"}}`)

	// Declared length over the limit is refused before the body is read.
	resp := sagemakerRuntimeRequest(t, ts, "stackyard-endpoint", "invocations-response-stream", bytes.Repeat([]byte("a"), sagemakerRuntimeMaxBody+1), nil)
	sagemakerRuntimeExpectError(t, resp, http.StatusBadRequest, "ValidationError", "6 MB")

	// Chunked bodies with no length are bounded too.
	resp = sagemakerRuntimeChunkedPost(t, ts.URL+"/endpoints/stackyard-endpoint/invocations", sagemakerRuntimeMaxBody+1)
	sagemakerRuntimeExpectError(t, resp, http.StatusBadRequest, "ValidationError", "6 MB")
	resp = sagemakerRuntimeChunkedPost(t, ts.URL+"/endpoints/async-ep/async-invocations", sagemakerRuntimeMaxAsyncBody+1)
	sagemakerRuntimeExpectError(t, resp, http.StatusBadRequest, "ValidationError", "128000")

	// A body exactly at the limit is accepted.
	resp = sagemakerRuntimeRequest(t, ts, "async-ep", "async-invocations", bytes.Repeat([]byte("a"), sagemakerRuntimeMaxAsyncBody), nil)
	assertStatus(t, resp, http.StatusAccepted)
	_ = mustBody(t, resp)
}

func TestSageMakerRuntimeStage2Sessions(t *testing.T) {
	srv, ts := newSageMakerRuntimeTestServer(t)
	sagemakerRuntimeCreateEndpoint(t, ts, "stateful", `"ProductionVariants":[{"VariantName":"v1"}]`)
	open := func() string {
		t.Helper()
		resp := sagemakerRuntimeRequest(t, ts, "stateful", "invocations", []byte("x"), map[string]string{"X-Amzn-SageMaker-Session-Id": "NEW_SESSION"})
		assertStatus(t, resp, http.StatusOK)
		_ = mustBody(t, resp)
		header := resp.Header.Get("X-Amzn-SageMaker-New-Session-Id")
		id, expiresAttr, _ := strings.Cut(header, "; Expires=")
		expires, err := time.Parse(time.RFC3339, expiresAttr)
		if err != nil || time.Until(expires) < 19*time.Minute || time.Until(expires) > 21*time.Minute {
			t.Fatalf("session header %q: expiry %v, err %v", header, expires, err)
		}
		return id
	}

	first := open()
	resp := sagemakerRuntimeRequest(t, ts, "stateful", "invocations-response-stream", []byte("x"), map[string]string{"X-Amzn-SageMaker-Session-Id": first})
	assertStatus(t, resp, http.StatusOK)
	_ = mustBody(t, resp)
	resp = sagemakerRuntimeRequest(t, ts, "stateful", "invocations-response-stream", []byte("x"), map[string]string{"X-Amzn-SageMaker-Session-Id": "NEW_SESSION"})
	sagemakerRuntimeExpectError(t, resp, http.StatusBadRequest, "ValidationError", "cannot open a session")

	// An expired session is rejected and swept when the next one opens.
	srv.sagemaker.mu.Lock()
	srv.sagemaker.runtimeSessions[first] = sagemakerRuntimeSession{Endpoint: "stateful", Expires: time.Now().Add(-time.Second)}
	srv.sagemaker.runtimeSessions["stale"] = sagemakerRuntimeSession{Endpoint: "stateful", Expires: time.Now().Add(-time.Second)}
	srv.sagemaker.mu.Unlock()
	resp = sagemakerRuntimeRequest(t, ts, "stateful", "invocations", []byte("x"), map[string]string{"X-Amzn-SageMaker-Session-Id": first})
	sagemakerRuntimeExpectError(t, resp, http.StatusBadRequest, "ValidationError", "expired")
	second := open()
	srv.sagemaker.mu.Lock()
	_, staleKept := srv.sagemaker.runtimeSessions["stale"]
	srv.sagemaker.mu.Unlock()
	if staleKept {
		t.Fatalf("expired session not swept")
	}

	// Deleting and recreating the endpoint invalidates its sessions.
	resp = sagemakerRequest(t, ts, "DeleteEndpoint", `{"EndpointName":"stateful"}`)
	assertStatus(t, resp, http.StatusOK)
	_ = mustBody(t, resp)
	resp = sagemakerRequest(t, ts, "CreateEndpoint", `{"EndpointName":"stateful","EndpointConfigName":"stateful-config"}`)
	assertStatus(t, resp, http.StatusOK)
	_ = mustBody(t, resp)
	resp = sagemakerRuntimeRequest(t, ts, "stateful", "invocations", []byte("x"), map[string]string{"X-Amzn-SageMaker-Session-Id": second})
	sagemakerRuntimeExpectError(t, resp, http.StatusBadRequest, "ValidationError", "not found")
}

func TestSageMakerRuntimeStage2AsyncConfig(t *testing.T) {
	_, ts := newSageMakerRuntimeTestServer(t)
	for body, fragment := range map[string]string{
		`{"EndpointConfigName":"c1","AsyncInferenceConfig":{}}`:                                          "outputConfig",
		`{"EndpointConfigName":"c2","AsyncInferenceConfig":{"OutputConfig":{"S3OutputPath":"ftp://x"}}}`: "S3OutputPath",
		`{"EndpointConfigName":"c3","AsyncInferenceConfig":{"OutputConfig":{"S3FailurePath":7}}}`:        "S3FailurePath",
		`{"EndpointConfigName":"c4","AsyncInferenceConfig":"yes"}`:                                       "must be an object",
	} {
		resp := sagemakerRequest(t, ts, "CreateEndpointConfig", body)
		assertStatus(t, resp, http.StatusBadRequest)
		if got := string(mustBody(t, resp)); !strings.Contains(got, fragment) || !strings.Contains(got, "ValidationException") {
			t.Fatalf("%s: got %s, want %q", body, got, fragment)
		}
	}

	// OutputConfig without S3OutputPath is valid, but async calls have nowhere to write.
	sagemakerRuntimeCreateEndpoint(t, ts, "no-output", `"AsyncInferenceConfig":{"OutputConfig":{"KmsKeyId":"k"}}`)
	resp := sagemakerRuntimeRequest(t, ts, "no-output", "async-invocations", []byte("x"), nil)
	sagemakerRuntimeExpectError(t, resp, http.StatusBadRequest, "ValidationError", "S3OutputPath")

	// https:// input locations are refused clearly, even alongside an inline body.
	sagemakerRuntimeCreateEndpoint(t, ts, "async-ep", `"AsyncInferenceConfig":{"OutputConfig":{"S3OutputPath":"s3://out/r"}}`)
	resp = sagemakerRuntimeRequest(t, ts, "async-ep", "async-invocations", []byte("x"), map[string]string{"X-Amzn-SageMaker-InputLocation": "https://in.s3.amazonaws.com/a"})
	sagemakerRuntimeExpectError(t, resp, http.StatusBadRequest, "ValidationError", "s3:// URI")
}

func TestSageMakerRuntimeStage2StreamChunks(t *testing.T) {
	_, ts := newSageMakerRuntimeTestServer(t)
	decode := func(raw []byte) [][]byte {
		t.Helper()
		var parts [][]byte
		reader, dec := bytes.NewReader(raw), eventstream.NewDecoder()
		for {
			msg, err := dec.Decode(reader, nil) // verifies prelude and message CRCs
			if err == io.EOF {
				return parts
			}
			if err != nil {
				t.Fatalf("decode part %d: %v", len(parts), err)
			}
			parts = append(parts, msg.Payload)
		}
	}
	stream := func(body []byte) [][]byte {
		t.Helper()
		resp := sagemakerRuntimeRequest(t, ts, "stackyard-endpoint", "invocations-response-stream", body, nil)
		assertStatus(t, resp, http.StatusOK)
		return decode(mustBody(t, resp))
	}

	// Multi-byte runes are never split, even where the size limit falls inside one.
	runes := []byte("a" + strings.Repeat("é", 300))
	parts := stream(runes)
	if len(parts) < 3 || !bytes.Equal(bytes.Join(parts, nil), runes) {
		t.Fatalf("got %d parts, reassembled ok %v", len(parts), bytes.Equal(bytes.Join(parts, nil), runes))
	}
	for i, p := range parts {
		if !utf8.Valid(p) || len(p) > sagemakerRuntimeStreamChunk {
			t.Fatalf("part %d invalid (len %d)", i, len(p))
		}
	}

	// A part ends after its last newline when one falls inside the limit.
	lines := []byte(strings.Repeat("x", 100) + "\n" + strings.Repeat("y", 300))
	parts = stream(lines)
	if len(parts[0]) != 101 || !bytes.Equal(bytes.Join(parts, nil), lines) {
		t.Fatalf("first part len %d", len(parts[0]))
	}

	// An empty body still sends one empty PayloadPart.
	if parts = stream(nil); len(parts) != 1 || len(parts[0]) != 0 {
		t.Fatalf("empty body parts = %q", parts)
	}
}

func TestSageMakerRuntimeStage2ResponsesAndRouting(t *testing.T) {
	srv, ts := newSageMakerRuntimeTestServer(t)

	// Wildcard or listed Accept values fall back to the request Content-Type.
	for accept, want := range map[string]string{"*/*": "text/csv", "application/json, text/csv": "text/csv", "application/json": "application/json"} {
		resp := sagemakerRuntimeRequest(t, ts, "stackyard-endpoint", "invocations", []byte("1,2"), map[string]string{"Content-Type": "text/csv", "Accept": accept})
		assertStatus(t, resp, http.StatusOK)
		_ = mustBody(t, resp)
		if got := resp.Header.Get("Content-Type"); got != want {
			t.Fatalf("Accept %q: Content-Type %q, want %q", accept, got, want)
		}
	}

	// A runtime call promotes a Creating endpoint without touching LastModifiedTime.
	sagemakerRuntimeCreateEndpoint(t, ts, "fresh", `"ProductionVariants":[{"VariantName":"v"}]`)
	srv.sagemaker.mu.Lock()
	srv.sagemaker.endpoints["fresh"].LastModifiedTime = "2020-01-01T00:00:00Z"
	srv.sagemaker.mu.Unlock()
	resp := sagemakerRuntimeRequest(t, ts, "fresh", "invocations", []byte("x"), nil)
	assertStatus(t, resp, http.StatusOK)
	_ = mustBody(t, resp)
	srv.sagemaker.mu.Lock()
	ep := *srv.sagemaker.endpoints["fresh"]
	srv.sagemaker.mu.Unlock()
	if ep.Status != "InService" || ep.LastModifiedTime != "2020-01-01T00:00:00Z" {
		t.Fatalf("endpoint after invoke = %+v", ep)
	}

	// InferenceId characters are escaped into the S3 key.
	sagemakerRuntimeCreateEndpoint(t, ts, "async-ep", `"AsyncInferenceConfig":{"OutputConfig":{"S3OutputPath":"s3://out/r"}}`)
	if _, err := srv.s3.CreateBucket("out"); err != nil {
		t.Fatalf("bucket: %v", err)
	}
	resp = sagemakerRuntimeRequest(t, ts, "async-ep", "async-invocations", []byte("x"), map[string]string{"X-Amzn-SageMaker-Inference-Id": "a/b?c"})
	assertStatus(t, resp, http.StatusAccepted)
	var out struct{ InferenceId string }
	if err := json.Unmarshal(mustBody(t, resp), &out); err != nil || out.InferenceId != "a/b?c" {
		t.Fatalf("InferenceId = %q, err %v", out.InferenceId, err)
	}
	if loc := resp.Header.Get("X-Amzn-SageMaker-OutputLocation"); loc != "s3://out/r/a%2Fb%3Fc.out" {
		t.Fatalf("OutputLocation = %q", loc)
	}
	if _, err := srv.s3.GetObject("out", "r/a%2Fb%3Fc.out"); err != nil {
		t.Fatalf("escaped output object: %v", err)
	}

	// A sagemaker-signed request with X-Amz-Target stays with the control plane.
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/endpoints/stackyard-endpoint/invocations", []byte(`{}`), map[string]string{
		"Content-Type": "application/x-amz-json-1.1",
		"X-Amz-Target": "SageMaker.ListEndpoints",
	}, "sagemaker")
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, `"Endpoints"`) {
		t.Fatalf("control plane body = %s", body)
	}

	// Route matching: host prefixes, other sagemaker-signed paths and targets.
	route := func(host, path, target, auth string) bool {
		req, _ := http.NewRequest(http.MethodPost, "http://"+host+path, nil)
		if target != "" {
			req.Header.Set("X-Amz-Target", target)
		}
		if auth != "" {
			req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential="+testAccessKey+"/20250101/us-east-1/"+auth+"/aws4_request, SignedHeaders=host, Signature=x")
		}
		_, _, ok := sagemakerRuntimeRoute(req)
		return ok
	}
	for _, tc := range []struct {
		host, path, target, auth string
		want                     bool
	}{
		{"runtime.sagemaker.us-east-1.amazonaws.com", "/endpoints/e/invocations", "", "", true},
		{"runtime-fips.sagemaker.us-east-1.amazonaws.com", "/endpoints/e/invocations", "", "", true},
		{"not-runtime.sagemaker.example.com", "/endpoints/e/invocations", "", "", false},
		{"localhost", "/endpoints/e/invocations", "", "sagemaker", true},
		{"localhost", "/endpoints/e/invocations", "SageMaker.ListEndpoints", "sagemaker", false},
		{"localhost", "/human-loops", "", "sagemaker", false},
		{"localhost", "/FeatureGroup/fg", "", "sagemaker", false},
		{"localhost", "/endpoints/e/other", "", "sagemaker", false},
	} {
		if got := route(tc.host, tc.path, tc.target, tc.auth); got != tc.want {
			t.Fatalf("route(%s %s target=%q auth=%q) = %v, want %v", tc.host, tc.path, tc.target, tc.auth, got, tc.want)
		}
	}
}
