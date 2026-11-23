// Command transcribe-streaming streams audio to Stackyard's Amazon Transcribe
// Streaming emulator: a live transcription with partial and final results, a
// Medical Scribe session read back with GetMedicalScribeStream.
//
// The AWS CLI has no transcribe-streaming commands, so requests are signed
// with SigV4 and event-stream frames are encoded with the SDK's eventstream
// package. Stackyard has no speech engine: transcripts describe the audio
// window they cover.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream"
	"github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/google/uuid"
)

func main() {
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	region := getenv("AWS_REGION", "us-east-1")
	creds := credentials.NewStaticCredentialsProvider(getenv("AWS_ACCESS_KEY_ID", "stackyard"), getenv("AWS_SECRET_ACCESS_KEY", "stackyard"), "")
	fmt.Println("Stackyard Transcribe Streaming client using", endpoint)

	headers := map[string]string{
		"x-amzn-transcribe-language-code":  "en-US",
		"x-amzn-transcribe-sample-rate":    "16000",
		"x-amzn-transcribe-media-encoding": "pcm",
	}
	second := make([]byte, 32000) // 1 s of 16 kHz 16-bit mono PCM (silence)

	// Live transcription: three seconds of audio, then an empty AudioEvent.
	body := frame("AudioEvent", second) + frame("AudioEvent", second) + frame("AudioEvent", second) + frame("AudioEvent", nil)
	for _, ev := range stream(endpoint, region, creds, http.MethodPost, "/stream-transcription", body, headers) {
		r := ev["Transcript"].(map[string]any)["Results"].([]any)[0].(map[string]any)
		alt := r["Alternatives"].([]any)[0].(map[string]any)
		fmt.Printf("TranscriptEvent partial=%v %v\n", r["IsPartial"], alt["Transcript"])
	}

	// Medical Scribe: configure, stream, end the session, read it back.
	session := uuid.NewString()
	headers["x-amzn-transcribe-session-id"] = session
	config, _ := json.Marshal(map[string]any{
		"ResourceAccessRoleArn":       "arn:aws:iam::123456789012:role/stackyard-scribe",
		"PostStreamAnalyticsSettings": map[string]any{"ClinicalNoteGenerationSettings": map[string]any{"OutputBucketName": "stackyard-notes"}},
	})
	end, _ := json.Marshal(map[string]string{"Type": "END_OF_SESSION"})
	body = frame("ConfigurationEvent", config) + frame("AudioEvent", second) + frame("SessionControlEvent", end)
	fmt.Println("Medical Scribe segments:", len(stream(endpoint, region, creds, http.MethodPost, "/medical-scribe-stream", body, headers)))
	raw := send(endpoint, region, creds, http.MethodGet, "/medical-scribe-stream/"+session, "", nil)
	var details struct{ MedicalScribeStreamDetails map[string]any }
	if err := json.Unmarshal(raw, &details); err != nil {
		exitf("GetMedicalScribeStream: %v", err)
	}
	fmt.Println("Scribe session", session, "status:", details.MedicalScribeStreamDetails["StreamStatus"])
	if details.MedicalScribeStreamDetails["StreamStatus"] != "COMPLETED" {
		exitf("scribe session did not complete")
	}
}

// frame encodes one event; JSON events carry a JSON payload, audio is raw.
func frame(eventType string, payload []byte) string {
	msg := eventstream.Message{Payload: payload}
	msg.Headers.Set(":message-type", eventstream.StringValue("event"))
	msg.Headers.Set(":event-type", eventstream.StringValue(eventType))
	contentType := "application/json"
	if eventType == "AudioEvent" {
		contentType = "application/octet-stream"
	}
	msg.Headers.Set(":content-type", eventstream.StringValue(contentType))
	var buf bytes.Buffer
	if err := eventstream.NewEncoder().Encode(&buf, msg); err != nil {
		exitf("encode %s: %v", eventType, err)
	}
	return buf.String()
}

// stream sends a whole event stream and returns the events of the response,
// failing on an exception frame.
func stream(endpoint, region string, creds credentials.StaticCredentialsProvider, method, path, body string, headers map[string]string) []map[string]any {
	raw := send(endpoint, region, creds, method, path, body, headers)
	var out []map[string]any
	r := bytes.NewReader(raw)
	for {
		msg, err := eventstream.NewDecoder().Decode(r, nil)
		if errors.Is(err, io.EOF) {
			return out
		}
		if err != nil {
			exitf("%s: decode frame: %v", path, err)
		}
		if v := msg.Headers.Get(":message-type"); v != nil && v.String() == "exception" {
			exitf("%s: %s %s", path, msg.Headers.Get(":exception-type"), msg.Payload)
		}
		var ev map[string]any
		if err := json.Unmarshal(msg.Payload, &ev); err != nil {
			exitf("%s: event payload: %v", path, err)
		}
		out = append(out, ev)
	}
}

func send(endpoint, region string, creds credentials.StaticCredentialsProvider, method, path, body string, headers map[string]string) []byte {
	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, method, endpoint+path, bytes.NewReader([]byte(body)))
	if err != nil {
		exitf("%s: %v", path, err)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/vnd.amazon.eventstream")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	value, err := creds.Retrieve(ctx)
	if err != nil {
		exitf("credentials: %v", err)
	}
	sum := sha256.Sum256([]byte(body))
	if err := v4.NewSigner().SignHTTP(ctx, value, req, hex.EncodeToString(sum[:]), "transcribe", region, time.Now()); err != nil {
		exitf("sign: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		exitf("%s: %v", path, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode/100 != 2 {
		exitf("%s: HTTP %d: %s", path, resp.StatusCode, raw)
	}
	return raw
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
