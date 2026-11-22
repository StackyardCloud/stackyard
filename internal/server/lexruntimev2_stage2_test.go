package server

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream"
)

func TestLexRuntimeV2Stage2RecognizeUtterance(t *testing.T) {
	_, ts := newLexRTTestServer(t)
	botID, aliasID := lexRTHotelBot(t, ts)
	path := "/bots/" + botID + "/botAliases/" + aliasID + "/botLocales/en_US/sessions/u1/utterance"

	state := base64.StdEncoding.EncodeToString([]byte(`{"sessionAttributes":{"k":"v"}}`))
	attrs := base64.StdEncoding.EncodeToString([]byte(`{"channel":"phone"}`))
	resp := lexRTRequest(t, ts, http.MethodPost, path, []byte("book a hotel"), map[string]string{
		"Content-Type":                 "text/plain; charset=utf-8",
		"Response-Content-Type":        "text/plain; charset=utf-8",
		"x-amz-lex-session-state":      state,
		"x-amz-lex-request-attributes": attrs,
	})
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); body != "Which room type?" {
		t.Fatalf("audioStream text = %q", body)
	}
	if resp.Header.Get("x-amz-lex-input-mode") != "Text" || resp.Header.Get("x-amz-lex-session-id") != "u1" || resp.Header.Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("headers = %v", resp.Header)
	}
	var got lexRTSessionState
	if err := decodeLexRTHeader(resp.Header.Get("x-amz-lex-session-state"), &got); err != nil ||
		got.Intent.Name != "BookHotel" || got.SessionAttributes["k"] != "v" {
		t.Fatalf("session-state header: %v %+v", err, got)
	}
	var interps []lexRTInterpretation
	if err := decodeLexRTHeader(resp.Header.Get("x-amz-lex-interpretations"), &interps); err != nil || interps[0].Intent.Name != "BookHotel" {
		t.Fatalf("interpretations header: %v %+v", err, interps)
	}
	var transcript []byte
	if raw, err := base64.StdEncoding.DecodeString(resp.Header.Get("x-amz-lex-input-transcript")); err == nil {
		transcript = raw
	}
	if len(transcript) < 2 || transcript[0] != 0x1f || transcript[1] != 0x8b {
		t.Fatalf("input transcript should be gzip+base64")
	}
	var echoed map[string]string
	if err := decodeLexRTHeader(resp.Header.Get("x-amz-lex-request-attributes"), &echoed); err != nil || echoed["channel"] != "phone" {
		t.Fatalf("request attributes header: %v %v", err, echoed)
	}

	// Audio in: accepted, not transcribed, so it falls back; audio out is empty.
	resp = lexRTRequest(t, ts, http.MethodPost, path, []byte{0, 1, 2, 3}, map[string]string{
		"Content-Type":          "audio/l16; rate=16000; channels=1",
		"Response-Content-Type": "audio/mpeg",
	})
	assertStatus(t, resp, http.StatusOK)
	if resp.Header.Get("Content-Type") != "audio/mpeg" || resp.Header.Get("x-amz-lex-input-mode") != "Speech" || len(mustBody(t, resp)) != 0 {
		t.Fatalf("audio response = %v", resp.Header)
	}

	for name, headers := range map[string]map[string]string{
		"html request":  {"Content-Type": "text/html"},
		"html response": {"Content-Type": "text/plain; charset=utf-8", "Response-Content-Type": "text/html"},
		"bad state":     {"Content-Type": "text/plain; charset=utf-8", "x-amz-lex-session-state": "%%%"},
	} {
		resp := lexRTRequest(t, ts, http.MethodPost, path, []byte("hi"), headers)
		assertStatus(t, resp, http.StatusBadRequest)
		if code := lexRTErrorCode(t, resp); code != "ValidationException" {
			t.Fatalf("%s: code %q", name, code)
		}
	}
}

func lexRTEvent(t *testing.T, eventType string, payload any) []byte {
	t.Helper()
	raw, _ := json.Marshal(payload)
	msg := eventstream.Message{Payload: raw}
	msg.Headers.Set(":message-type", eventstream.StringValue("event"))
	msg.Headers.Set(":event-type", eventstream.StringValue(eventType))
	msg.Headers.Set(":content-type", eventstream.StringValue("application/json"))
	var buf bytes.Buffer
	if err := eventstream.NewEncoder().Encode(&buf, msg); err != nil {
		t.Fatalf("encode %s: %v", eventType, err)
	}
	return buf.Bytes()
}

// lexRTSigned wraps an event the way SDKs do on a signed event stream.
func lexRTSigned(t *testing.T, inner []byte) []byte {
	t.Helper()
	msg := eventstream.Message{Payload: inner}
	msg.Headers.Set(":date", eventstream.TimestampValue(time.Now()))
	msg.Headers.Set(":chunk-signature", eventstream.BytesValue([]byte{1, 2, 3}))
	var buf bytes.Buffer
	if err := eventstream.NewEncoder().Encode(&buf, msg); err != nil {
		t.Fatalf("encode envelope: %v", err)
	}
	return buf.Bytes()
}

type lexRTFrame struct {
	kind, eventType string
	body            map[string]any
}

func readLexRTFrame(t *testing.T, r io.Reader) (lexRTFrame, error) {
	t.Helper()
	msg, err := eventstream.NewDecoder().Decode(r, nil)
	if err != nil {
		return lexRTFrame{}, err
	}
	f := lexRTFrame{kind: lexRTHeaderString(msg.Headers, ":message-type"), eventType: lexRTHeaderString(msg.Headers, ":event-type")}
	if f.kind == "exception" {
		f.eventType = lexRTHeaderString(msg.Headers, ":exception-type")
	}
	if err := json.Unmarshal(msg.Payload, &f.body); err != nil {
		t.Fatalf("frame payload: %v", err)
	}
	return f, nil
}

func TestLexRuntimeV2Stage2StartConversationBuffered(t *testing.T) {
	_, ts := newLexRTTestServer(t)
	botID, aliasID := lexRTHotelBot(t, ts)
	path := "/bots/" + botID + "/botAliases/" + aliasID + "/botLocales/en_US/sessions/c1/conversation"

	var body []byte
	body = append(body, lexRTEvent(t, "ConfigurationEvent", map[string]any{
		"responseContentType": "text/plain; charset=utf-8",
		"requestAttributes":   map[string]string{"a": "b"},
		"welcomeMessages":     []map[string]string{{"contentType": "PlainText", "content": "Hello!"}},
		"sessionState":        map[string]any{"sessionAttributes": map[string]string{"x": "y"}},
	})...)
	body = append(body, lexRTEvent(t, "TextInputEvent", map[string]string{"text": "book a hotel"})...)
	body = append(body, lexRTEvent(t, "AudioInputEvent", map[string]string{"contentType": "audio/lpcm"})...)
	body = append(body, lexRTEvent(t, "DTMFInputEvent", map[string]string{"inputCharacter": "1"})...)
	body = append(body, lexRTEvent(t, "DisconnectionEvent", map[string]string{})...)
	resp := lexRTRequest(t, ts, http.MethodPost, path, body, map[string]string{"Content-Type": "application/vnd.amazon.eventstream", "x-amz-lex-conversation-mode": "TEXT"})
	assertStatus(t, resp, http.StatusOK)
	defer resp.Body.Close()

	var frames []lexRTFrame
	for {
		f, err := readLexRTFrame(t, resp.Body)
		if err != nil {
			break
		}
		frames = append(frames, f)
	}
	want := []string{"TextResponseEvent", "IntentResultEvent", "TextResponseEvent", "IntentResultEvent", "TextResponseEvent"}
	if len(frames) != len(want) {
		t.Fatalf("frames = %+v", frames)
	}
	for i, f := range frames {
		if f.kind != "event" || f.eventType != want[i] {
			t.Fatalf("frame %d = %s/%s, want event/%s", i, f.kind, f.eventType, want[i])
		}
		if id, _ := f.body["eventId"].(string); !strings.HasPrefix(id, "RESPONSE-") {
			t.Fatalf("frame %d eventId = %v", i, f.body["eventId"])
		}
	}
	result := frames[1].body
	state, _ := result["sessionState"].(map[string]any)
	intent, _ := state["intent"].(map[string]any)
	attrs, _ := state["sessionAttributes"].(map[string]any)
	if intent["name"] != "BookHotel" || attrs["x"] != "y" || result["inputMode"] != "Text" || result["sessionId"] != "c1" {
		t.Fatalf("IntentResultEvent = %v", result)
	}
	if mode := frames[3].body["inputMode"]; mode != "DTMF" {
		t.Fatalf("DTMF inputMode = %v", mode)
	}

	// A bad configuration ends the stream with a modeled exception frame.
	bad := lexRTEvent(t, "ConfigurationEvent", map[string]any{"responseContentType": "text/html"})
	resp = lexRTRequest(t, ts, http.MethodPost, path, bad, nil)
	assertStatus(t, resp, http.StatusOK)
	f, err := readLexRTFrame(t, resp.Body)
	resp.Body.Close()
	if err != nil || f.kind != "exception" || f.eventType != "ValidationException" {
		t.Fatalf("exception frame = %+v %v", f, err)
	}

	// Target errors come back as plain HTTP errors before the stream opens.
	resp = lexRTRequest(t, ts, http.MethodPost, "/bots/"+botID+"/botAliases/"+aliasID+"/botLocales/de_DE/sessions/c1/conversation", body, nil)
	assertStatus(t, resp, http.StatusNotFound)
	resp.Body.Close()
	resp = lexRTRequest(t, ts, http.MethodPost, path, body, map[string]string{"x-amz-lex-conversation-mode": "VIDEO"})
	assertStatus(t, resp, http.StatusBadRequest)
	resp.Body.Close()
}

// TestLexRuntimeV2Stage2StartConversationStreaming drives a real full-duplex
// conversation: each turn's answer is read while the request is still open,
// with the SDK's STREAMING-AWS4-HMAC-SHA256-EVENTS signing and signed frames.
func TestLexRuntimeV2Stage2StartConversationStreaming(t *testing.T) {
	_, ts := newLexRTTestServer(t)
	botID, aliasID := lexRTHotelBot(t, ts)

	pr, pw := io.Pipe()
	req, err := http.NewRequest(http.MethodPost, ts.URL+"/bots/"+botID+"/botAliases/"+aliasID+"/botLocales/en_US/sessions/live/conversation", pr)
	if err != nil {
		t.Fatal(err)
	}
	req.Host = req.URL.Host
	amzDate := time.Now().UTC().Format("20060102T150405Z")
	req.Header.Set("x-amz-date", amzDate)
	req.Header.Set("x-amz-content-sha256", sigV4StreamingEventsPayload)
	req.Header.Set("Content-Type", "application/vnd.amazon.eventstream")
	signed := buildSignedHeaders(req)
	canonical, _ := buildCanonicalRequest(req, signed, sigV4StreamingEventsPayload, false)
	signature := signString(buildStringToSign(amzDate, testRegion, "lex", canonical), testRegion, "lex", testSecretKey)
	req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential="+testAccessKey+"/"+amzDate[:8]+"/"+testRegion+"/lex/aws4_request, SignedHeaders="+signed+", Signature="+signature)

	// One writer goroutine keeps frames in order; the pipe blocks until the
	// server reads, so writes can't happen on the test goroutine.
	frames := make(chan []byte, 8)
	go func() {
		for f := range frames {
			_, _ = pw.Write(f)
		}
		_ = pw.Close()
	}()
	send := func(frame []byte) { frames <- frame }
	send(lexRTSigned(t, lexRTEvent(t, "ConfigurationEvent", map[string]string{"responseContentType": "text/plain; charset=utf-8"})))
	send(lexRTSigned(t, lexRTEvent(t, "TextInputEvent", map[string]string{"text": "book a hotel"})))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("start conversation: %v", err)
	}
	defer resp.Body.Close()
	assertStatus(t, resp, http.StatusOK)

	expect := func(eventType string) lexRTFrame {
		t.Helper()
		f, err := readLexRTFrame(t, resp.Body)
		if err != nil || f.eventType != eventType {
			t.Fatalf("want %s, got %+v %v", eventType, f, err)
		}
		return f
	}
	expect("IntentResultEvent")
	if msgs := expect("TextResponseEvent").body["messages"].([]any); msgs[0].(map[string]any)["content"] != "Which room type?" {
		t.Fatalf("prompt = %v", msgs)
	}
	// The request body is still open; the next turn is answered in place.
	send(lexRTSigned(t, lexRTEvent(t, "TextInputEvent", map[string]string{"text": "twin"})))
	slots := expect("IntentResultEvent").body["sessionState"].(map[string]any)["intent"].(map[string]any)["slots"].(map[string]any)
	if slots["RoomType"].(map[string]any)["value"].(map[string]any)["interpretedValue"] != "twin" {
		t.Fatalf("slots = %v", slots)
	}
	expect("TextResponseEvent")
	send(lexRTSigned(t, nil)) // empty signed frame: end of stream
	close(frames)
	if _, err := readLexRTFrame(t, resp.Body); err == nil {
		t.Fatal("expected the stream to end")
	}
}
