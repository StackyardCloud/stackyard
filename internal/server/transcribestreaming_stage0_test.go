package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream"
)

func newTSSTestServer(t *testing.T) (*Server, *httptest.Server) {
	t.Helper()
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return srv, ts
}

// tssFrame encodes one input event; payload is raw bytes for AudioEvent and
// JSON otherwise.
func tssFrame(t *testing.T, eventType string, payload any) []byte {
	t.Helper()
	msg := eventstream.Message{}
	msg.Headers.Set(":message-type", eventstream.StringValue("event"))
	msg.Headers.Set(":event-type", eventstream.StringValue(eventType))
	if raw, ok := payload.([]byte); ok {
		msg.Payload = raw
		msg.Headers.Set(":content-type", eventstream.StringValue("application/octet-stream"))
	} else {
		b, err := json.Marshal(payload)
		if err != nil {
			t.Fatal(err)
		}
		msg.Payload = b
		msg.Headers.Set(":content-type", eventstream.StringValue("application/json"))
	}
	var buf bytes.Buffer
	if err := eventstream.NewEncoder().Encode(&buf, msg); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func tssHeaders(extra map[string]string) map[string]string {
	h := map[string]string{
		"Content-Type":                     "application/vnd.amazon.eventstream",
		"x-amzn-transcribe-language-code":  "en-US",
		"x-amzn-transcribe-sample-rate":    "16000",
		"x-amzn-transcribe-media-encoding": "pcm",
	}
	for k, v := range extra {
		if v == "" {
			delete(h, k)
		} else {
			h[k] = v
		}
	}
	return h
}

func tssPost(t *testing.T, ts *httptest.Server, path string, headers map[string]string, frames ...[]byte) *http.Response {
	t.Helper()
	return signedRequestWithService(t, http.MethodPost, ts.URL+path, bytes.Join(frames, nil), headers, "transcribe")
}

type tssOut struct {
	kind, eventType string
	body            map[string]any
}

// tssFrames reads every response frame until the server closes the stream.
func tssFrames(t *testing.T, resp *http.Response) []tssOut {
	t.Helper()
	defer resp.Body.Close()
	var out []tssOut
	for {
		msg, err := eventstream.NewDecoder().Decode(resp.Body, nil)
		if errors.Is(err, io.EOF) {
			return out
		}
		if err != nil {
			t.Fatalf("decode frame: %v", err)
		}
		f := tssOut{kind: lexRTHeaderString(msg.Headers, ":message-type"), eventType: lexRTHeaderString(msg.Headers, ":event-type")}
		if f.kind == "exception" {
			f.eventType = lexRTHeaderString(msg.Headers, ":exception-type")
		}
		if err := json.Unmarshal(msg.Payload, &f.body); err != nil {
			t.Fatalf("frame payload: %v", err)
		}
		out = append(out, f)
	}
}

func tssErrorType(t *testing.T, resp *http.Response) string {
	t.Helper()
	defer resp.Body.Close()
	var body struct{ Message string }
	if err := json.Unmarshal(mustBody(t, resp), &body); err != nil || body.Message == "" {
		t.Fatalf("error body: %v", err)
	}
	return resp.Header.Get("X-Amzn-ErrorType")
}

func TestTranscribeStreamingStage0Operations(t *testing.T) {
	for _, op := range transcribestreamingOperations {
		got, _, ok := transcribeStreamingRoute(strings.Replace(op.URI, "{SessionId}", "x", 1))
		if !ok || got != op.Name {
			t.Fatalf("%s %s routes to %q", op.Method, op.URI, got)
		}
	}
	if _, _, ok := transcribeStreamingRoute("/medical-scribe-stream/a/b"); ok {
		t.Fatal("nested scribe path must not route")
	}
}

func TestTranscribeStreamingStage0RoutingKeepsBatchTranscribe(t *testing.T) {
	_, ts := newTSSTestServer(t)
	// Batch Transcribe (JSON 1.1 on "/") still answers.
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte(`{}`), map[string]string{
		"Content-Type": "application/x-amz-json-1.1", "X-Amz-Target": "Transcribe.ListVocabularies",
	}, "transcribe")
	assertStatus(t, resp, http.StatusOK)
	resp.Body.Close()
	// A request signed for another service isn't claimed.
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/stream-transcription", nil, tssHeaders(nil), "s3")
	if resp.Header.Get("X-Amzn-ErrorType") == "BadRequestException" {
		t.Fatal("an s3-signed request was claimed by Transcribe Streaming")
	}
	resp.Body.Close()
	// A body of unknown length that isn't a signed event stream is refused.
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/stream-transcription", io.MultiReader(strings.NewReader("x")))
	req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential="+testAccessKey+"/20260101/"+testRegion+"/transcribe/aws4_request, SignedHeaders=host, Signature=00")
	raw, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if raw.StatusCode != http.StatusBadRequest || tssErrorType(t, raw) != "BadRequestException" {
		t.Fatalf("chunked body: %d", raw.StatusCode)
	}
}

func TestTranscribeStreamingStage0HeaderValidation(t *testing.T) {
	_, ts := newTSSTestServer(t)
	for name, c := range map[string]struct {
		path    string
		headers map[string]string
	}{
		"no sample rate":         {"/stream-transcription", map[string]string{"x-amzn-transcribe-sample-rate": ""}},
		"rate out of range":      {"/stream-transcription", map[string]string{"x-amzn-transcribe-sample-rate": "96000"}},
		"bad encoding":           {"/stream-transcription", map[string]string{"x-amzn-transcribe-media-encoding": "mp3"}},
		"language and identify":  {"/stream-transcription", map[string]string{"x-amzn-transcribe-identify-language": "true"}},
		"identify one option":    {"/stream-transcription", map[string]string{"x-amzn-transcribe-language-code": "", "x-amzn-transcribe-identify-language": "true", "x-amzn-transcribe-language-options": "en-US"}},
		"bad session id":         {"/stream-transcription", map[string]string{"x-amzn-transcribe-session-id": "nope"}},
		"unknown vocabulary":     {"/stream-transcription", map[string]string{"x-amzn-transcribe-vocabulary-name": "missing-vocab"}},
		"filter method alone":    {"/stream-transcription", map[string]string{"x-amzn-transcribe-vocabulary-filter-method": "mask"}},
		"stability alone":        {"/stream-transcription", map[string]string{"x-amzn-transcribe-partial-results-stability": "high"}},
		"channels without flag":  {"/stream-transcription", map[string]string{"x-amzn-transcribe-number-of-channels": "2"}},
		"medical non en-US":      {"/medical-stream-transcription", map[string]string{"x-amzn-transcribe-language-code": "en-GB", "x-amzn-transcribe-specialty": "PRIMARYCARE", "x-amzn-transcribe-type": "DICTATION"}},
		"medical no specialty":   {"/medical-stream-transcription", map[string]string{"x-amzn-transcribe-type": "DICTATION"}},
		"call analytics ja-JP":   {"/call-analytics-stream-transcription", map[string]string{"x-amzn-transcribe-language-code": "ja-JP"}},
		"scribe low sample rate": {"/medical-scribe-stream", map[string]string{"x-amzn-transcribe-sample-rate": "8000"}},
	} {
		resp := tssPost(t, ts, c.path, tssHeaders(c.headers), tssFrame(t, "AudioEvent", []byte{1, 2}))
		if resp.StatusCode != http.StatusBadRequest || tssErrorType(t, resp) != "BadRequestException" {
			t.Fatalf("%s: %d", name, resp.StatusCode)
		}
	}
	// A vocabulary defined in batch Transcribe is accepted.
	resp := tssPost(t, ts, "/stream-transcription", tssHeaders(map[string]string{"x-amzn-transcribe-vocabulary-name": "stackyard-vocabulary"}), tssFrame(t, "AudioEvent", []byte{}))
	assertStatus(t, resp, http.StatusOK)
	if resp.Header.Get("x-amzn-transcribe-vocabulary-name") != "stackyard-vocabulary" {
		t.Fatalf("echoed headers: %v", resp.Header)
	}
	tssFrames(t, resp)
	// GetMedicalScribeStream: bad id, then unknown session.
	for id, want := range map[string]int{"nope": http.StatusBadRequest, "11111111-2222-3333-4444-555555555555": http.StatusNotFound} {
		resp := signedRequestWithService(t, http.MethodGet, ts.URL+"/medical-scribe-stream/"+id, nil, nil, "transcribe")
		if resp.StatusCode != want {
			t.Fatalf("GetMedicalScribeStream %s: %d", id, resp.StatusCode)
		}
		tssErrorType(t, resp)
	}
}
