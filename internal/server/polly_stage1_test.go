package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream"
)

const pollyTestLexicon = `<?xml version="1.0" encoding="UTF-8"?>
<lexicon version="1.0" xmlns="http://www.w3.org/2005/01/pronunciation-lexicon" alphabet="ipa" xml:lang="en-US">
  <lexeme><grapheme>W3C</grapheme><alias>World Wide Web Consortium</alias></lexeme>
</lexicon>`

func TestPollyStage1VoicesAndSynthesis(t *testing.T) {
	_, ts := newPollyTestServer(t)

	all := pollyCall(t, ts, http.StatusOK, http.MethodGet, "/v1/voices", "")
	if n := len(all["Voices"].([]any)); n != 100 || all["NextToken"] != "100" {
		t.Fatalf("first page: %d voices, token %v", n, all["NextToken"])
	}
	gen := pollyCall(t, ts, http.StatusOK, http.MethodGet, "/v1/voices?Engine=generative&LanguageCode=en-US", "")
	for _, v := range gen["Voices"].([]any) {
		if v.(map[string]any)["LanguageCode"] != "en-US" {
			t.Fatalf("language filter: %v", v)
		}
	}
	hindi := pollyCall(t, ts, http.StatusOK, http.MethodGet, "/v1/voices?LanguageCode=hi-IN&IncludeAdditionalLanguageCodes=true", "")
	if len(hindi["Voices"].([]any)) != 2 {
		t.Fatalf("additional languages: %v", hindi)
	}
	if code := pollyErrCode(t, ts, http.MethodGet, "/v1/voices?Engine=robot", ""); code != "ValidationException" {
		t.Fatalf("bad engine: %s", code)
	}

	// Each audio format comes back with its fixed content type and a valid header.
	for format, want := range map[string]struct {
		ctype string
		magic []byte
	}{
		"mp3": {"audio/mpeg", []byte{0xFF, 0xF3}}, "ogg_vorbis": {"audio/ogg", []byte("OggS")},
		"ogg_opus": {"audio/ogg", []byte("OggS")}, "pcm": {"audio/pcm", []byte{0, 0}},
	} {
		status, hdr, data := pollyRaw(t, ts, http.MethodPost, "/v1/speech", `{"OutputFormat":"`+format+`","Text":"Hello there","VoiceId":"Joanna"}`, nil)
		if status != http.StatusOK || hdr.Get("Content-Type") != want.ctype || !bytes.HasPrefix(data, want.magic) {
			t.Fatalf("%s: %d %s %x", format, status, hdr.Get("Content-Type"), data[:min(4, len(data))])
		}
		if hdr.Get("x-amzn-RequestCharacters") != "11" {
			t.Fatalf("%s: billed characters %s", format, hdr.Get("x-amzn-RequestCharacters"))
		}
	}
	// PCM length follows the text: 11 chars * 60 ms at 16 kHz, 16-bit.
	_, _, pcm := pollyRaw(t, ts, http.MethodPost, "/v1/speech", `{"OutputFormat":"pcm","SampleRate":"16000","Text":"Hello there","VoiceId":"Joanna"}`, nil)
	if len(pcm) != 16000*2*660/1000 {
		t.Fatalf("pcm length %d", len(pcm))
	}

	// Speech marks.
	status, hdr, data := pollyRaw(t, ts, http.MethodPost, "/v1/speech", `{"OutputFormat":"json","SpeechMarkTypes":["sentence","word"],"Text":"Mary had a lamb. It was white.","VoiceId":"Joanna"}`, nil)
	if status != http.StatusOK || hdr.Get("Content-Type") != "application/x-json-stream" {
		t.Fatalf("marks: %d %s", status, hdr.Get("Content-Type"))
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	var first, second map[string]any
	_ = json.Unmarshal([]byte(lines[0]), &first)
	_ = json.Unmarshal([]byte(lines[1]), &second)
	if first["type"] != "sentence" || first["value"] != "Mary had a lamb." || second["type"] != "word" || second["value"] != "Mary" {
		t.Fatalf("marks: %s", data)
	}
	if n := strings.Count(string(data), `"type":"sentence"`); n != 2 {
		t.Fatalf("want 2 sentences, got %d: %s", n, data)
	}

	// SSML: marks, well-formedness and the <speak> root.
	ssml := `{"OutputFormat":"json","SpeechMarkTypes":["ssml","word"],"TextType":"ssml","Text":"<speak>Hi <mark name=\"m1\"/>you</speak>","VoiceId":"Joanna"}`
	_, _, data = pollyRaw(t, ts, http.MethodPost, "/v1/speech", ssml, nil)
	if !strings.Contains(string(data), `"type":"ssml"`) || !strings.Contains(string(data), `"value":"m1"`) {
		t.Fatalf("ssml marks: %s", data)
	}
	for name, body := range map[string]string{
		"InvalidSsmlException":                      `{"OutputFormat":"mp3","TextType":"ssml","Text":"<speak>open","VoiceId":"Joanna"}`,
		"MarksNotSupportedForFormatException":       `{"OutputFormat":"mp3","SpeechMarkTypes":["word"],"Text":"hi","VoiceId":"Joanna"}`,
		"SsmlMarksNotSupportedForTextTypeException": `{"OutputFormat":"json","SpeechMarkTypes":["ssml"],"Text":"hi","VoiceId":"Joanna"}`,
		"EngineNotSupportedException":               `{"OutputFormat":"mp3","Engine":"neural","Text":"hi","VoiceId":"Aditi"}`,
		"InvalidSampleRateException":                `{"OutputFormat":"pcm","SampleRate":"22050","Text":"hi","VoiceId":"Joanna"}`,
		"LexiconNotFoundException":                  `{"OutputFormat":"mp3","LexiconNames":["missing"],"Text":"hi","VoiceId":"Joanna"}`,
		"LanguageNotSupportedException":             `{"OutputFormat":"mp3","LanguageCode":"fr-FR","Text":"hi","VoiceId":"Joanna"}`,
		"TextLengthExceededException":               `{"OutputFormat":"mp3","Text":"` + strings.Repeat("a", 3001) + `","VoiceId":"Joanna"}`,
		"ValidationException":                       `{"OutputFormat":"wav","Text":"hi","VoiceId":"Joanna"}`,
	} {
		if code := pollyErrCode(t, ts, http.MethodPost, "/v1/speech", body); code != name {
			t.Fatalf("want %s, got %s", name, code)
		}
	}
	if code := pollyErrCode(t, ts, http.MethodPost, "/v1/speech", `{"OutputFormat":"mp3","TextType":"ssml","Text":"<other>x</other>","VoiceId":"Joanna"}`); code != "InvalidSsmlException" {
		t.Fatalf("non-speak root: %s", code)
	}
}

func TestPollyStage1LexiconsTasksVoicesAndTags(t *testing.T) {
	srv, ts := newPollyTestServer(t)

	put := func(name, content string) (int, []byte) {
		body, _ := json.Marshal(map[string]any{"Content": content})
		status, _, data := pollyRaw(t, ts, http.MethodPut, "/v1/lexicons/"+name, string(body), nil)
		return status, data
	}
	if status, data := put("W3C", pollyTestLexicon); status != http.StatusOK {
		t.Fatalf("PutLexicon: %d %s", status, data)
	}
	for want, content := range map[string]string{
		"InvalidLexiconException":         "<lexicon",
		"UnsupportedPlsAlphabetException": strings.Replace(pollyTestLexicon, `alphabet="ipa"`, `alphabet="morse"`, 1),
		"UnsupportedPlsLanguageException": strings.Replace(pollyTestLexicon, `xml:lang="en-US"`, `xml:lang="xx-XX"`, 1),
		"LexiconSizeExceededException":    strings.Replace(pollyTestLexicon, "W3C", strings.Repeat("x", 4000), 1),
	} {
		body, _ := json.Marshal(map[string]any{"Content": content})
		if code := pollyErrCode(t, ts, http.MethodPut, "/v1/lexicons/Bad", string(body)); code != want {
			t.Fatalf("want %s, got %s", want, code)
		}
	}
	got := pollyCall(t, ts, http.StatusOK, http.MethodGet, "/v1/lexicons/W3C", "")
	attrs := got["LexiconAttributes"].(map[string]any)
	if attrs["Alphabet"] != "ipa" || attrs["LanguageCode"] != "en-US" || attrs["LexemesCount"] != float64(1) {
		t.Fatalf("lexicon attributes: %v", attrs)
	}
	lexARN := attrs["LexiconArn"].(string)
	if status, _, data := pollyRaw(t, ts, http.MethodPost, "/v1/speech", `{"OutputFormat":"mp3","LexiconNames":["W3C"],"Text":"W3C","VoiceId":"Joanna"}`, nil); status != http.StatusOK {
		t.Fatalf("synthesis with lexicon: %d %s", status, data)
	}

	// Tags.
	pollyCall(t, ts, http.StatusOK, http.MethodPost, "/v1/tagResource", `{"ResourceArn":"`+lexARN+`","Tags":{"env":"dev"}}`)
	tags := pollyCall(t, ts, http.StatusOK, http.MethodGet, "/v1/listTagsForResource/?arn="+lexARN, "")
	if tags["Tags"].(map[string]any)["env"] != "dev" {
		t.Fatalf("tags: %v", tags)
	}
	if code := pollyErrCode(t, ts, http.MethodPost, "/v1/tagResource", `{"ResourceArn":"`+lexARN+`","Tags":{"aws:x":"y"}}`); code != "InvalidTagParameterException" {
		t.Fatalf("aws: tag: %s", code)
	}
	if code := pollyErrCode(t, ts, http.MethodPost, "/v1/tagResource", `{"ResourceArn":"arn:aws:polly:us-east-1:123456789012:lexicon/nope","Tags":{"a":"b"}}`); code != "ResourceNotFoundException" {
		t.Fatalf("tag missing resource: %s", code)
	}
	pollyCall(t, ts, http.StatusOK, http.MethodPost, "/v1/untagResource", `{"ResourceArn":"`+lexARN+`","TagKeys":["env"]}`)

	// Synthesis tasks write to S3 and advance on read.
	if _, err := srv.s3.CreateBucket("polly-out"); err != nil {
		t.Fatal(err)
	}
	if code := pollyErrCode(t, ts, http.MethodPost, "/v1/synthesisTasks", `{"OutputFormat":"mp3","OutputS3BucketName":"no-such-bucket","Text":"hi","VoiceId":"Joanna"}`); code != "InvalidS3BucketException" {
		t.Fatalf("missing bucket: %s", code)
	}
	if code := pollyErrCode(t, ts, http.MethodPost, "/v1/synthesisTasks", `{"OutputFormat":"mp3","OutputS3BucketName":"polly-out","SnsTopicArn":"arn:aws:sns:us-east-1:123456789012:nope","Text":"hi","VoiceId":"Joanna"}`); code != "InvalidSnsTopicArnException" {
		t.Fatalf("missing topic: %s", code)
	}
	task := pollyCall(t, ts, http.StatusOK, http.MethodPost, "/v1/synthesisTasks", `{"OutputFormat":"mp3","OutputS3BucketName":"polly-out","OutputS3KeyPrefix":"out/","Text":"hello","VoiceId":"Joanna"}`)["SynthesisTask"].(map[string]any)
	id := task["TaskId"].(string)
	if task["TaskStatus"] != "scheduled" || !strings.HasSuffix(task["OutputUri"].(string), "/polly-out/out/"+id+".mp3") {
		t.Fatalf("task: %v", task)
	}
	if obj, err := srv.s3.GetObject("polly-out", "out/"+id+".mp3"); err != nil || len(obj.Body) == 0 {
		t.Fatalf("task output object: %v", err)
	}
	for _, want := range []string{"inProgress", "completed", "completed"} {
		if got := pollyCall(t, ts, http.StatusOK, http.MethodGet, "/v1/synthesisTasks/"+id, "")["SynthesisTask"].(map[string]any)["TaskStatus"]; got != want {
			t.Fatalf("task status %v, want %s", got, want)
		}
	}
	listed := pollyCall(t, ts, http.StatusOK, http.MethodGet, "/v1/synthesisTasks?Status=completed&MaxResults=1", "")
	if len(listed["SynthesisTasks"].([]any)) != 1 {
		t.Fatalf("list tasks: %v", listed)
	}
	if code := pollyErrCode(t, ts, http.MethodGet, "/v1/synthesisTasks/unknown-task", ""); code != "SynthesisTaskNotFoundException" {
		t.Fatalf("unknown task: %s", code)
	}

	// Custom voices.
	voiceBody := `{"Name":"mine","LanguageCode":"en-US","TermsAccepted":true,"ClientToken":"tok-1","VoiceCreationData":{"Recordings":[{"Format":"wav","Data":{"Bytes":"AAAA"}}]}}`
	if code := pollyErrCode(t, ts, http.MethodPost, "/v1/voices", strings.Replace(voiceBody, "true", "false", 1)); code != "ValidationException" {
		t.Fatalf("terms not accepted: %s", code)
	}
	voice := pollyCall(t, ts, http.StatusOK, http.MethodPost, "/v1/voices", voiceBody)["Voice"].(map[string]any)
	vid := voice["Id"].(string)
	if voice["Status"] != "pending" || voice["Origin"] != "customer" {
		t.Fatalf("voice: %v", voice)
	}
	// Recording.Data is required in responses, but Polly never returns the bytes.
	rec := voice["VoiceCreationData"].(map[string]any)["Recordings"].([]any)[0].(map[string]any)
	if data, ok := rec["Data"].(map[string]any); !ok || len(data) != 0 {
		t.Fatalf("byte recording Data should be present and empty: %v", rec)
	}
	if again := pollyCall(t, ts, http.StatusOK, http.MethodPost, "/v1/voices", voiceBody)["Voice"].(map[string]any); again["Id"] != vid {
		t.Fatalf("ClientToken isn't idempotent: %v", again)
	}
	if code := pollyErrCode(t, ts, http.MethodDelete, "/v1/voices/"+vid, ""); code != "ConflictException" {
		t.Fatalf("delete pending voice: %s", code)
	}
	pollyCall(t, ts, http.StatusOK, http.MethodGet, "/v1/voices/"+vid, "")
	if got := pollyCall(t, ts, http.StatusOK, http.MethodGet, "/v1/voices/"+vid, "")["Voice"].(map[string]any); got["Status"] != "active" {
		t.Fatalf("voice after read: %v", got)
	}
	if got := pollyCall(t, ts, http.StatusOK, http.MethodGet, "/v1/voices/Joanna", "")["Voice"].(map[string]any); got["Origin"] != "aws" {
		t.Fatalf("built-in GetVoice: %v", got)
	}
	pollyCall(t, ts, http.StatusOK, http.MethodDelete, "/v1/voices/"+vid, "")
	if code := pollyErrCode(t, ts, http.MethodGet, "/v1/voices/"+vid, ""); code != "ResourceNotFoundException" {
		t.Fatalf("deleted voice: %s", code)
	}

	pollyCall(t, ts, http.StatusOK, http.MethodDelete, "/v1/lexicons/W3C", "")
	if code := pollyErrCode(t, ts, http.MethodGet, "/v1/lexicons/W3C", ""); code != "LexiconNotFoundException" {
		t.Fatalf("deleted lexicon: %s", code)
	}
}

func TestPollyStage1Stream(t *testing.T) {
	_, ts := newPollyTestServer(t)
	var body bytes.Buffer
	enc := eventstream.NewEncoder()
	for _, ev := range []struct{ typ, payload string }{
		{"TextEvent", `{"Text":"Hello"}`}, {"TextEvent", `{"Text":" world"}`}, {"CloseStreamEvent", `{}`},
	} {
		msg := eventstream.Message{Payload: []byte(ev.payload)}
		msg.Headers.Set(":message-type", eventstream.StringValue("event"))
		msg.Headers.Set(":event-type", eventstream.StringValue(ev.typ))
		_ = enc.Encode(&body, msg)
	}
	headers := map[string]string{
		"Content-Type": "application/vnd.amazon.eventstream", "x-amzn-Engine": "neural",
		"x-amzn-OutputFormat": "pcm", "x-amzn-VoiceId": "Joanna",
	}
	status, hdr, data := pollyRaw(t, ts, http.MethodPost, "/v1/synthesisStream", body.String(), headers)
	if status != http.StatusOK || hdr.Get("Content-Type") != "application/vnd.amazon.eventstream" {
		t.Fatalf("stream: %d %s %s", status, hdr.Get("Content-Type"), data)
	}
	events, err := brtReadEvents(data)
	if err != nil {
		t.Fatal(err)
	}
	last := events[len(events)-1]
	if brtHeaderString(events[0], ":event-type") != "AudioEvent" || brtHeaderString(last, ":event-type") != "StreamClosedEvent" || !strings.Contains(string(last.Payload), `"RequestCharacters":11`) {
		t.Fatalf("stream events: %d, last %s", len(events), last.Payload)
	}
	delete(headers, "x-amzn-VoiceId")
	if status, _, _ := pollyRaw(t, ts, http.MethodPost, "/v1/synthesisStream", body.String(), headers); status != http.StatusBadRequest {
		t.Fatalf("stream without VoiceId: %d", status)
	}
}

// Each TextEvent is capped on its own, but every result is buffered until the
// stream ends, so the whole stream is bounded too.
func TestPollyStage1StreamAudioIsBoundedAcrossEvents(t *testing.T) {
	_, ts := newPollyTestServer(t)
	var body bytes.Buffer
	enc := eventstream.NewEncoder()
	text, _ := json.Marshal(map[string]string{"Text": strings.Repeat("word ", 590)})
	for i := 0; i < 40; i++ { // ~1.9 MB of 16 kHz PCM per event, ~77 MB in total
		msg := eventstream.Message{Payload: text}
		msg.Headers.Set(":message-type", eventstream.StringValue("event"))
		msg.Headers.Set(":event-type", eventstream.StringValue("TextEvent"))
		_ = enc.Encode(&body, msg)
	}
	headers := map[string]string{
		"Content-Type": "application/vnd.amazon.eventstream", "x-amzn-Engine": "neural",
		"x-amzn-OutputFormat": "pcm", "x-amzn-SampleRate": "16000", "x-amzn-VoiceId": "Joanna",
	}
	status, _, data := pollyRaw(t, ts, http.MethodPost, "/v1/synthesisStream", body.String(), headers)
	if status == http.StatusOK && len(data) > pollyMaxStreamAudio+(1<<20) {
		t.Fatalf("stream returned %d bytes, over the %d-byte cap", len(data), pollyMaxStreamAudio)
	}
	if status == http.StatusOK && !strings.Contains(string(data), "ServiceQuotaExceededException") {
		t.Fatalf("expected ServiceQuotaExceededException, got %d bytes", len(data))
	}
}
