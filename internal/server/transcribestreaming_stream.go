package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream"
	"github.com/google/uuid"
)

// transcribeStreamingRun answers one start operation: headers and 200 first,
// then a transcript frame for every audio frame as it arrives, then the final
// results when the client ends its stream.
//
// SDK clients sign with STREAMING-AWS4-HMAC-SHA256-EVENTS and wrap each event in
// an envelope carrying :chunk-signature; the header signature is checked,
// ponytail: the per-event chunk signatures are not (same as Lex Runtime V2).
func (s *Server) transcribeStreamingRun(w http.ResponseWriter, r *http.Request, p tssParams) {
	rc := http.NewResponseController(w)
	_ = rc.EnableFullDuplex() // HTTP/1.1 only; HTTP/2 is already full duplex
	for name, value := range p.echo {
		w.Header().Set("x-amzn-transcribe-"+name, value)
	}
	w.Header().Set("x-amzn-request-id", uuid.NewString())
	w.Header().Set("Content-Type", "application/vnd.amazon.eventstream")
	w.WriteHeader(http.StatusOK)
	_ = rc.Flush()

	c := &tssSession{s: s, p: p, w: w, rc: rc, enc: eventstream.NewEncoder()}
	defer c.close()
	body := &tssCountingReader{r: io.LimitReader(r.Body, tssMaxStreamBytes+1)}
	dec := eventstream.NewDecoder()
	buf := make([]byte, 0, 4096)
	for {
		msg, err := dec.Decode(body, buf)
		if err != nil {
			switch {
			case body.n > tssMaxStreamBytes:
				c.exception("LimitExceededException", "the audio stream exceeds the maximum length")
			case errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF):
				c.finish()
			default:
				c.exception("BadRequestException", "malformed event stream frame")
			}
			return
		}
		if msg.Headers.Get(":chunk-signature") != nil {
			if len(msg.Payload) == 0 { // empty signed envelope: end of the client stream
				c.finish()
				return
			}
			inner, err := dec.Decode(bytes.NewReader(msg.Payload), nil)
			if err != nil {
				c.exception("BadRequestException", "malformed signed event")
				return
			}
			msg = inner
		}
		if len(msg.Payload) > tssMaxEventBytes {
			c.exception("BadRequestException", "an event payload exceeds 256 KiB")
			return
		}
		if !c.handle(lexRTHeaderString(msg.Headers, ":event-type"), msg.Payload) {
			return
		}
	}
}

type tssCountingReader struct {
	r io.Reader
	n int64
}

func (c *tssCountingReader) Read(b []byte) (int, error) {
	n, err := c.r.Read(b)
	c.n += int64(n)
	return n, err
}

type tssSession struct {
	s   *Server
	p   tssParams
	w   http.ResponseWriter
	rc  *http.ResponseController
	enc *eventstream.Encoder

	configured bool             // a ConfigurationEvent was accepted
	sawAudio   bool             // audio arrived (configuration must come first)
	roles      map[int]string   // call analytics / scribe channel roles
	seconds    float64          // audio received so far
	segStart   float64          // start of the open result
	segment    int              // results finalised so far
	ended      bool             // finish or exception ran
	scribeDone bool             // END_OF_SESSION received
	scribe     *tssScribeConfig // accepted scribe configuration
}

// handle processes one client event and reports whether to keep reading.
func (c *tssSession) handle(eventType string, payload []byte) bool {
	switch eventType {
	case "AudioEvent":
		if c.p.op == "StartMedicalScribeStream" && !c.configured && !c.s.transcribeStreaming.resumable(c.p.sessionID) {
			return c.exception("BadRequestException", "the first event of a new Medical Scribe stream must be a ConfigurationEvent")
		}
		if len(payload) == 0 { // an empty audio event also ends the stream
			c.finish()
			return false
		}
		c.sawAudio = true
		return c.audio(len(payload))
	case "ConfigurationEvent":
		switch c.p.op {
		case "StartCallAnalyticsStreamTranscription":
			return c.callAnalyticsConfig(payload)
		case "StartMedicalScribeStream":
			return c.scribeConfig(payload)
		}
		return c.exception("BadRequestException", "this stream doesn't accept a ConfigurationEvent")
	case "SessionControlEvent":
		if c.p.op != "StartMedicalScribeStream" {
			return c.exception("BadRequestException", "this stream doesn't accept a SessionControlEvent")
		}
		var ev struct{ Type string }
		if json.Unmarshal(payload, &ev) != nil || ev.Type != "END_OF_SESSION" {
			return c.exception("BadRequestException", "SessionControlEvent.Type must be END_OF_SESSION")
		}
		c.scribeDone = true
		c.finish()
		return false
	}
	return c.exception("BadRequestException", "unknown event type")
}

func (c *tssSession) callAnalyticsConfig(payload []byte) bool {
	if c.configured || c.sawAudio {
		return c.exception("BadRequestException", "the ConfigurationEvent must be the first event and sent once")
	}
	var ev struct {
		ChannelDefinitions []struct {
			ChannelId       int
			ParticipantRole string
		}
		PostCallAnalyticsSettings *struct {
			OutputLocation           string
			DataAccessRoleArn        string
			ContentRedactionOutput   string
			OutputEncryptionKMSKeyId string
		}
	}
	dec := json.NewDecoder(bytes.NewReader(payload))
	dec.DisallowUnknownFields()
	if dec.Decode(&ev) != nil {
		return c.exception("BadRequestException", "ConfigurationEvent isn't valid")
	}
	roles, ok := tssChannelRoles(len(ev.ChannelDefinitions), func(i int) (int, string) {
		return ev.ChannelDefinitions[i].ChannelId, ev.ChannelDefinitions[i].ParticipantRole
	}, "AGENT", "CUSTOMER")
	if !ok {
		return c.exception("BadRequestException", "ChannelDefinitions must map channels 0 and 1 to AGENT and CUSTOMER")
	}
	if pc := ev.PostCallAnalyticsSettings; pc != nil {
		if !strings.HasPrefix(pc.OutputLocation, "s3://") || !tssRoleARNPattern.MatchString(pc.DataAccessRoleArn) {
			return c.exception("BadRequestException", "PostCallAnalyticsSettings needs an s3:// OutputLocation and a DataAccessRoleArn")
		}
		if pc.ContentRedactionOutput != "" && pc.ContentRedactionOutput != "redacted" && pc.ContentRedactionOutput != "redacted_and_unredacted" {
			return c.exception("BadRequestException", "ContentRedactionOutput must be redacted or redacted_and_unredacted")
		}
	}
	c.roles, c.configured = roles, true
	return true
}

// tssScribeConfig is the Medical Scribe ConfigurationEvent.
type tssScribeConfig struct {
	VocabularyName         string `json:",omitempty"`
	VocabularyFilterName   string `json:",omitempty"`
	VocabularyFilterMethod string `json:",omitempty"`
	ResourceAccessRoleArn  string
	ChannelDefinitions     []struct {
		ChannelId       int
		ParticipantRole string
	} `json:",omitempty"`
	EncryptionSettings *struct {
		KmsKeyId             string
		KmsEncryptionContext map[string]string `json:",omitempty"`
	} `json:",omitempty"`
	PostStreamAnalyticsSettings struct {
		ClinicalNoteGenerationSettings struct {
			OutputBucketName string
			NoteTemplate     string `json:",omitempty"`
		}
	}
	MedicalScribeContext json.RawMessage `json:",omitempty"`
}

func (c *tssSession) scribeConfig(payload []byte) bool {
	if c.configured || c.sawAudio || c.s.transcribeStreaming.resumable(c.p.sessionID) {
		return c.exception("BadRequestException", "the ConfigurationEvent must be the first event of a new session and sent once")
	}
	var cfg tssScribeConfig
	dec := json.NewDecoder(bytes.NewReader(payload))
	dec.DisallowUnknownFields()
	if dec.Decode(&cfg) != nil {
		return c.exception("BadRequestException", "ConfigurationEvent isn't valid")
	}
	if !tssRoleARNPattern.MatchString(cfg.ResourceAccessRoleArn) {
		return c.exception("BadRequestException", "ResourceAccessRoleArn must be an IAM role ARN")
	}
	if cfg.PostStreamAnalyticsSettings.ClinicalNoteGenerationSettings.OutputBucketName == "" {
		return c.exception("BadRequestException", "PostStreamAnalyticsSettings.ClinicalNoteGenerationSettings.OutputBucketName is required")
	}
	if m := cfg.VocabularyFilterMethod; m != "" && (cfg.VocabularyFilterName == "" || (m != "remove" && m != "mask" && m != "tag")) {
		return c.exception("BadRequestException", "VocabularyFilterMethod needs a filter and must be remove, mask or tag")
	}
	if len(cfg.MedicalScribeContext) > 64<<10 {
		return c.exception("BadRequestException", "MedicalScribeContext is too large")
	}
	roles, ok := tssChannelRoles(len(cfg.ChannelDefinitions), func(i int) (int, string) {
		return cfg.ChannelDefinitions[i].ChannelId, cfg.ChannelDefinitions[i].ParticipantRole
	}, "PATIENT", "CLINICIAN")
	if !ok {
		return c.exception("BadRequestException", "ChannelDefinitions must map channels 0 and 1 to PATIENT and CLINICIAN")
	}
	check := tssParams{op: "StartMedicalStreamTranscription", language: c.p.language, vocabulary: cfg.VocabularyName}
	if cfg.VocabularyFilterName != "" {
		check.filter = cfg.VocabularyFilterName
	}
	if msg := c.s.transcribe.checkStreamingResources(check); msg != "" {
		return c.exception("BadRequestException", msg)
	}
	c.roles, c.scribe, c.configured = roles, &cfg, true
	c.s.transcribeStreaming.configure(c.p, &cfg)
	return true
}

// tssChannelRoles validates optional two-channel role definitions.
func tssChannelRoles(n int, at func(int) (int, string), a, b string) (map[int]string, bool) {
	if n == 0 {
		return nil, true
	}
	if n != 2 {
		return nil, false
	}
	roles := map[int]string{}
	for i := range n {
		id, role := at(i)
		if (id != 0 && id != 1) || (role != a && role != b) || roles[id] != "" {
			return nil, false
		}
		roles[id] = role
	}
	return roles, roles[0] != roles[1]
}

// audio advances the received duration by an estimate for the encoding and
// sends a partial result, finalising one every tssSegmentSeconds.
func (c *tssSession) audio(n int) bool {
	c.seconds += float64(n) / tssBytesPerSecond(c.p)
	for c.seconds-c.segStart >= tssSegmentSeconds {
		end := c.segStart + tssSegmentSeconds
		c.result(c.segStart, end, false)
		c.segStart, c.segment = end, c.segment+1
	}
	if c.seconds > c.segStart {
		c.result(c.segStart, c.seconds, true)
	}
	return true
}

// ponytail: compressed encodings are assumed to run at a fixed bitrate; decode
// them if a test needs exact timings.
func tssBytesPerSecond(p tssParams) float64 {
	switch p.encoding {
	case "pcm":
		return float64(p.rate * 2 * p.channels)
	case "g711-alaw", "g711-ulaw":
		return float64(p.rate * p.channels)
	case "g729":
		return 1000 * float64(p.channels)
	}
	return 4000 * float64(p.channels) // ogg-opus, flac: ~32 kbit/s per channel
}

// finish finalises the open result once, when the client ends its stream.
func (c *tssSession) finish() {
	if c.ended {
		return
	}
	c.ended = true
	if c.seconds > c.segStart {
		c.result(c.segStart, c.seconds, false)
	}
	if c.p.op == "StartCallAnalyticsStreamTranscription" {
		// ponytail: category rules aren't evaluated; every real-time category
		// defined in Transcribe is reported as matched.
		c.event("CategoryEvent", map[string]any{"MatchedCategories": c.s.transcribe.callAnalyticsCategoryNames(), "MatchedDetails": map[string]any{}})
	}
}

// result sends one transcript frame in the shape of the stream's operation.
// There's no speech engine: the text describes the audio window it covers.
func (c *tssSession) result(start, end float64, partial bool) {
	id := fmt.Sprintf("%s-%d", c.p.sessionID[:8], c.segment)
	text := fmt.Sprintf("[audio %.2fs-%.2fs]", start, end)
	item := map[string]any{"StartTime": start, "EndTime": end, "Type": "pronunciation", "Content": text}
	if c.p.speakerLabels {
		item["Speaker"] = "spk_0"
	}
	if c.p.stabilize {
		item["Stable"] = !partial
	}
	switch c.p.op {
	case "StartCallAnalyticsStreamTranscription":
		role := "CUSTOMER"
		if c.roles != nil {
			role = c.roles[0]
		}
		ev := map[string]any{
			"UtteranceId": id, "IsPartial": partial, "ParticipantRole": role,
			"BeginOffsetMillis": int64(start * 1000), "EndOffsetMillis": int64(end * 1000), "Transcript": text,
			"Items": []any{map[string]any{"BeginOffsetMillis": int64(start * 1000), "EndOffsetMillis": int64(end * 1000), "Type": "pronunciation", "Content": text, "Confidence": 1.0, "VocabularyFilterMatch": false, "Stable": !partial}},
		}
		if !partial {
			ev["Sentiment"] = "NEUTRAL"
		}
		if c.p.identify {
			ev["LanguageCode"] = c.p.language
		}
		c.event("UtteranceEvent", ev)
	case "StartMedicalScribeStream":
		c.event("TranscriptEvent", map[string]any{"TranscriptSegment": map[string]any{
			"SegmentId": id, "BeginAudioTime": start, "EndAudioTime": end, "Content": text, "IsPartial": partial, "ChannelId": "0",
			"Items": []any{map[string]any{"BeginAudioTime": start, "EndAudioTime": end, "Type": "pronunciation", "Content": text, "Confidence": 1.0, "VocabularyFilterMatch": false}},
		}})
	default: // StartStreamTranscription, StartMedicalStreamTranscription
		res := map[string]any{
			"ResultId": id, "StartTime": start, "EndTime": end, "IsPartial": partial,
			"Alternatives": []any{map[string]any{"Transcript": text, "Items": []any{item}}},
		}
		if c.p.channelID {
			res["ChannelId"] = "ch_0"
		}
		if c.p.identify && c.p.op == "StartStreamTranscription" {
			res["LanguageCode"] = c.p.language
		}
		c.event("TranscriptEvent", map[string]any{"Transcript": map[string]any{"Results": []any{res}}})
	}
}

func (c *tssSession) event(eventType string, body map[string]any) {
	payload, err := json.Marshal(body)
	if err != nil {
		return
	}
	msg := eventstream.Message{Payload: payload}
	msg.Headers.Set(":message-type", eventstream.StringValue("event"))
	msg.Headers.Set(":event-type", eventstream.StringValue(eventType))
	msg.Headers.Set(":content-type", eventstream.StringValue("application/json"))
	c.write(msg)
}

// exception writes a modeled exception frame and ends the stream.
func (c *tssSession) exception(code, message string) bool {
	c.ended = true
	payload, _ := json.Marshal(map[string]string{"Message": message})
	msg := eventstream.Message{Payload: payload}
	msg.Headers.Set(":message-type", eventstream.StringValue("exception"))
	msg.Headers.Set(":exception-type", eventstream.StringValue(code))
	msg.Headers.Set(":content-type", eventstream.StringValue("application/json"))
	c.write(msg)
	if c.p.op == "StartMedicalScribeStream" {
		c.s.transcribeStreaming.fail(c.p.sessionID)
	}
	return false
}

func (c *tssSession) write(msg eventstream.Message) {
	if err := c.enc.Encode(c.w, msg); err == nil {
		_ = c.rc.Flush()
	}
}

// close records the outcome of a Medical Scribe stream.
func (c *tssSession) close() {
	if c.p.op == "StartMedicalScribeStream" {
		c.s.transcribeStreaming.end(c.p.sessionID, c.scribeDone, c.seconds)
	}
}

// callAnalyticsCategoryNames lists the Call Analytics categories defined in
// batch Transcribe.
func (s *transcribeStore) callAnalyticsCategoryNames() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]string, 0, len(s.callAnalyticsCategories))
	for name := range s.callAnalyticsCategories {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}
