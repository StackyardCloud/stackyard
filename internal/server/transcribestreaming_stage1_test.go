package server

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"
)

// 16 kHz 16-bit mono PCM is 32000 bytes a second.
func tssSeconds(s float64) []byte { return make([]byte, int(s*32000)) }

func TestTranscribeStreamingStage1StartStreamTranscription(t *testing.T) {
	_, ts := newTSSTestServer(t)
	resp := tssPost(t, ts, "/stream-transcription", tssHeaders(map[string]string{
		"x-amzn-transcribe-enable-partial-results-stabilization": "true",
		"x-amzn-transcribe-show-speaker-label":                   "true",
	}),
		tssFrame(t, "AudioEvent", tssSeconds(2)),
		tssFrame(t, "AudioEvent", tssSeconds(2)),
		tssFrame(t, "AudioEvent", tssSeconds(2)),
		tssFrame(t, "AudioEvent", []byte{}), // end of stream
	)
	assertStatus(t, resp, http.StatusOK)
	if resp.Header.Get("x-amzn-transcribe-session-id") == "" || resp.Header.Get("x-amzn-transcribe-sample-rate") != "16000" {
		t.Fatalf("response headers: %v", resp.Header)
	}
	frames := tssFrames(t, resp)
	type result struct {
		start, end float64
		partial    bool
	}
	var got []result
	for _, f := range frames {
		if f.kind != "event" || f.eventType != "TranscriptEvent" {
			t.Fatalf("unexpected frame %+v", f)
		}
		r := f.body["Transcript"].(map[string]any)["Results"].([]any)[0].(map[string]any)
		item := r["Alternatives"].([]any)[0].(map[string]any)["Items"].([]any)[0].(map[string]any)
		if item["Speaker"] != "spk_0" || item["Stable"] != !r["IsPartial"].(bool) {
			t.Fatalf("item = %v", item)
		}
		got = append(got, result{r["StartTime"].(float64), r["EndTime"].(float64), r["IsPartial"].(bool)})
	}
	// 2 s, 4 s partials; at 6 s the first 5 s segment is final and a new one opens.
	want := []result{{0, 2, true}, {0, 4, true}, {0, 5, false}, {5, 6, true}, {5, 6, false}}
	if len(got) != len(want) {
		t.Fatalf("results = %+v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("result %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestTranscribeStreamingStage1IdentifyLanguageAndLimits(t *testing.T) {
	_, ts := newTSSTestServer(t)
	resp := tssPost(t, ts, "/stream-transcription", tssHeaders(map[string]string{
		"x-amzn-transcribe-language-code":      "",
		"x-amzn-transcribe-identify-language":  "true",
		"x-amzn-transcribe-language-options":   "en-US,es-US",
		"x-amzn-transcribe-preferred-language": "es-US",
	}), tssFrame(t, "AudioEvent", tssSeconds(1)))
	frames := tssFrames(t, resp)
	last := frames[len(frames)-1].body["Transcript"].(map[string]any)["Results"].([]any)[0].(map[string]any)
	if last["LanguageCode"] != "es-US" || last["IsPartial"] != false {
		t.Fatalf("final result = %v", last)
	}
	// An event payload over 256 KiB ends the stream with an exception frame.
	resp = tssPost(t, ts, "/stream-transcription", tssHeaders(nil), tssFrame(t, "AudioEvent", make([]byte, 300<<10)))
	frames = tssFrames(t, resp)
	if f := frames[len(frames)-1]; f.kind != "exception" || f.eventType != "BadRequestException" {
		t.Fatalf("oversized event: %+v", frames)
	}
	// A ConfigurationEvent isn't part of StartStreamTranscription.
	resp = tssPost(t, ts, "/stream-transcription", tssHeaders(nil), tssFrame(t, "ConfigurationEvent", map[string]any{}))
	if f := tssFrames(t, resp); len(f) != 1 || f[0].eventType != "BadRequestException" {
		t.Fatalf("configuration on a plain stream: %+v", f)
	}
}

func TestTranscribeStreamingStage1MedicalAndCallAnalytics(t *testing.T) {
	_, ts := newTSSTestServer(t)
	resp := tssPost(t, ts, "/medical-stream-transcription", tssHeaders(map[string]string{
		"x-amzn-transcribe-specialty": "CARDIOLOGY", "x-amzn-transcribe-type": "CONVERSATION",
		"x-amzn-transcribe-vocabulary-name": "stackyard-medical-vocabulary",
	}), tssFrame(t, "AudioEvent", tssSeconds(1)))
	assertStatus(t, resp, http.StatusOK)
	frames := tssFrames(t, resp)
	if len(frames) != 2 || frames[1].body["Transcript"].(map[string]any)["Results"].([]any)[0].(map[string]any)["IsPartial"] != false {
		t.Fatalf("medical frames: %+v", frames)
	}

	ca := tssHeaders(nil)
	config := tssFrame(t, "ConfigurationEvent", map[string]any{
		"ChannelDefinitions":        []any{map[string]any{"ChannelId": 0, "ParticipantRole": "AGENT"}, map[string]any{"ChannelId": 1, "ParticipantRole": "CUSTOMER"}},
		"PostCallAnalyticsSettings": map[string]any{"OutputLocation": "s3://bucket/out", "DataAccessRoleArn": "arn:aws:iam::123456789012:role/ca"},
	})
	frames = tssFrames(t, tssPost(t, ts, "/call-analytics-stream-transcription", ca, config, tssFrame(t, "AudioEvent", tssSeconds(1))))
	if len(frames) != 3 || frames[0].eventType != "UtteranceEvent" || frames[2].eventType != "CategoryEvent" {
		t.Fatalf("call analytics frames: %+v", frames)
	}
	final := frames[1].body
	if final["ParticipantRole"] != "AGENT" || final["IsPartial"] != false || final["Sentiment"] != "NEUTRAL" || final["EndOffsetMillis"] != float64(1000) {
		t.Fatalf("final utterance = %v", final)
	}
	// The configuration must come first.
	frames = tssFrames(t, tssPost(t, ts, "/call-analytics-stream-transcription", ca, tssFrame(t, "AudioEvent", tssSeconds(1)), config))
	if f := frames[len(frames)-1]; f.kind != "exception" || f.eventType != "BadRequestException" {
		t.Fatalf("late configuration: %+v", frames)
	}
}

func TestTranscribeStreamingStage1MedicalScribeSession(t *testing.T) {
	_, ts := newTSSTestServer(t)
	const id = "0f8fad5b-d9cb-469f-a165-70867728950e"
	headers := tssHeaders(map[string]string{"x-amzn-transcribe-session-id": id})
	config := tssFrame(t, "ConfigurationEvent", map[string]any{
		"ResourceAccessRoleArn":       "arn:aws:iam::123456789012:role/scribe",
		"ChannelDefinitions":          []any{map[string]any{"ChannelId": 0, "ParticipantRole": "CLINICIAN"}, map[string]any{"ChannelId": 1, "ParticipantRole": "PATIENT"}},
		"PostStreamAnalyticsSettings": map[string]any{"ClinicalNoteGenerationSettings": map[string]any{"OutputBucketName": "notes"}},
	})
	get := func() map[string]any {
		t.Helper()
		resp := signedRequestWithService(t, http.MethodGet, ts.URL+"/medical-scribe-stream/"+id, nil, nil, "transcribe")
		assertStatus(t, resp, http.StatusOK)
		var out struct{ MedicalScribeStreamDetails map[string]any }
		if err := json.Unmarshal(mustBody(t, resp), &out); err != nil {
			t.Fatal(err)
		}
		return out.MedicalScribeStreamDetails
	}

	// A new session must start with its configuration.
	frames := tssFrames(t, tssPost(t, ts, "/medical-scribe-stream", headers, tssFrame(t, "AudioEvent", tssSeconds(1))))
	if len(frames) != 1 || frames[0].eventType != "BadRequestException" {
		t.Fatalf("audio before configuration: %+v", frames)
	}
	// Configured, then dropped without END_OF_SESSION: the session pauses.
	frames = tssFrames(t, tssPost(t, ts, "/medical-scribe-stream", headers, config, tssFrame(t, "AudioEvent", tssSeconds(1))))
	if frames[0].eventType != "TranscriptEvent" || frames[0].body["TranscriptSegment"].(map[string]any)["IsPartial"] != true {
		t.Fatalf("scribe frames: %+v", frames)
	}
	if d := get(); d["StreamStatus"] != "PAUSED" || d["ResourceAccessRoleArn"] != "arn:aws:iam::123456789012:role/scribe" {
		t.Fatalf("paused details = %v", d)
	}
	// Resuming takes no configuration; END_OF_SESSION completes it.
	frames = tssFrames(t, tssPost(t, ts, "/medical-scribe-stream", headers,
		tssFrame(t, "AudioEvent", tssSeconds(1)), tssFrame(t, "SessionControlEvent", map[string]string{"Type": "END_OF_SESSION"})))
	if f := frames[len(frames)-1]; f.kind != "event" {
		t.Fatalf("resume frames: %+v", frames)
	}
	d := get()
	result := d["PostStreamAnalyticsResult"].(map[string]any)["ClinicalNoteGenerationResult"].(map[string]any)
	if d["StreamStatus"] != "COMPLETED" || d["StreamEndedAt"] == nil || result["ClinicalNoteOutputLocation"] != "s3://notes/"+id+"/summary.json" {
		t.Fatalf("completed details = %v", d)
	}
	// A completed session can't stream again.
	resp := tssPost(t, ts, "/medical-scribe-stream", headers, config)
	if resp.StatusCode != http.StatusBadRequest || tssErrorType(t, resp) != "BadRequestException" {
		t.Fatalf("restart completed session: %d", resp.StatusCode)
	}
}

// The response is a live stream: each audio frame is answered before the
// client sends the next one.
func TestTranscribeStreamingStage1Duplex(t *testing.T) {
	_, ts := newTSSTestServer(t)
	pr, pw := io.Pipe()
	req, err := http.NewRequest(http.MethodPost, ts.URL+"/stream-transcription", pr)
	if err != nil {
		t.Fatal(err)
	}
	for k, v := range tssHeaders(nil) {
		req.Header.Set(k, v)
	}
	req.Host = req.URL.Host
	amzDate := time.Now().UTC().Format("20060102T150405Z")
	req.Header.Set("x-amz-date", amzDate)
	req.Header.Set("x-amz-content-sha256", sigV4StreamingEventsPayload)
	signed := buildSignedHeaders(req)
	canonical, _ := buildCanonicalRequest(req, signed, sigV4StreamingEventsPayload, false)
	signature := signString(buildStringToSign(amzDate, testRegion, "transcribe", canonical), testRegion, "transcribe", testSecretKey)
	req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential="+testAccessKey+"/"+amzDate[:8]+"/"+testRegion+"/transcribe/aws4_request, SignedHeaders="+signed+", Signature="+signature)

	frames := make(chan []byte, 4)
	go func() {
		for f := range frames {
			_, _ = pw.Write(f)
		}
		_ = pw.Close()
	}()
	frames <- lexRTSigned(t, tssFrame(t, "AudioEvent", tssSeconds(1)))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	read := func() map[string]any {
		t.Helper()
		f, err := readLexRTFrame(t, resp.Body)
		if err != nil || f.eventType != "TranscriptEvent" {
			t.Fatalf("frame %+v %v", f, err)
		}
		return f.body["Transcript"].(map[string]any)["Results"].([]any)[0].(map[string]any)
	}
	if r := read(); r["EndTime"] != float64(1) || r["IsPartial"] != true {
		t.Fatalf("first partial = %v", r)
	}
	frames <- lexRTSigned(t, tssFrame(t, "AudioEvent", tssSeconds(1)))
	if r := read(); r["EndTime"] != float64(2) {
		t.Fatalf("second partial = %v", r)
	}
	frames <- lexRTSigned(t, nil) // empty signed envelope: end of stream
	close(frames)
	if r := read(); r["IsPartial"] != false || r["EndTime"] != float64(2) {
		t.Fatalf("final = %v", r)
	}
}
