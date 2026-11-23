package server

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream"
	"github.com/stackyard/stackyard/internal/awsmodels"
)

const (
	pollyMaxStreamEvents = 1000
	pollyMaxStreamAudio  = 32 << 20 // audio bytes buffered across one stream
	pollyStreamChunk     = 32 << 10
)

// pollyStream answers StartSpeechSynthesisStream. Settings arrive as
// x-amzn-* headers and text as TextEvent frames ending with CloseStreamEvent.
// ponytail: the request stream is read whole (it must declare its length)
// before audio is sent back; full-duplex SDK streaming isn't modelled.
func (s *Server) pollyStream(w http.ResponseWriter, r *http.Request) *pollyError {
	settings := map[string]any{}
	for _, h := range []struct{ header, member, shape string }{
		{"x-amzn-Engine", "Engine", "Engine"},
		{"x-amzn-OutputFormat", "OutputFormat", "OutputFormat"},
		{"x-amzn-VoiceId", "VoiceId", "VoiceId"},
		{"x-amzn-LanguageCode", "LanguageCode", "LanguageCode"},
		{"x-amzn-SampleRate", "SampleRate", "SampleRate"},
	} {
		v := strings.TrimSpace(r.Header.Get(h.header))
		if v == "" {
			continue
		}
		if err := pollyCheck(h.shape, h.member, v); err != nil {
			return err
		}
		settings[h.member] = v
	}
	for _, req := range []string{"Engine", "OutputFormat", "VoiceId"} {
		if settings[req] == nil {
			return pollyValidation("%s is required.", req)
		}
	}
	if raw := r.Header.Get("x-amzn-LexiconNames"); raw != "" {
		var names []any
		for _, n := range strings.Split(raw, ",") {
			names = append(names, strings.TrimSpace(n))
		}
		settings["LexiconNames"] = names
	}
	body, err := readBodyBytes(r)
	if err != nil {
		return pollyValidation("unable to read request body")
	}
	events, err := brtReadEvents(body)
	if err != nil {
		return pollyValidation("the request body is not a valid event stream")
	}
	if len(events) > pollyMaxStreamEvents {
		return pollyValidation("A stream can carry at most %d events.", pollyMaxStreamEvents)
	}
	m := awsmodels.PollyModel()
	var results []*pollySynthesis
	chars, audioBytes := 0, 0
	for _, ev := range events {
		switch brtHeaderString(ev, ":event-type") {
		case "TextEvent":
			text := map[string]any{}
			if err := json.Unmarshal(ev.Payload, &text); err != nil {
				return pollyValidation("TextEvent payload is not a JSON object")
			}
			if problems := m.Validate("TextEvent", text); len(problems) > 0 {
				return pollyValidation("%s", strings.Join(problems, "; "))
			}
			in := map[string]any{"Text": text["Text"], "TextType": text["TextType"]}
			for k, v := range settings {
				in[k] = v
			}
			res, perr := s.pollySynthesize(in, pollySpeechLimits)
			if perr != nil {
				return perr
			}
			chars += res.chars
			// Every result is buffered until the stream ends, so bound the
			// whole stream, not just each event.
			if audioBytes += len(res.audio); audioBytes > pollyMaxStreamAudio {
				return pollyErr(http.StatusPaymentRequired, "ServiceQuotaExceededException", "A stream can produce at most %d bytes of audio.", pollyMaxStreamAudio)
			}
			results = append(results, res)
		case "CloseStreamEvent":
		default:
			e := pollyValidation("Unexpected inbound event %q.", brtHeaderString(ev, ":event-type"))
			e.extra["reason"] = "invalidInboundEvent"
			return e
		}
	}
	w.Header().Set("Content-Type", "application/vnd.amazon.eventstream")
	w.WriteHeader(http.StatusOK)
	enc := eventstream.NewEncoder()
	send := func(eventType, contentType string, payload []byte) {
		msg := eventstream.Message{Payload: payload}
		msg.Headers.Set(":message-type", eventstream.StringValue("event"))
		msg.Headers.Set(":event-type", eventstream.StringValue(eventType))
		msg.Headers.Set(":content-type", eventstream.StringValue(contentType))
		_ = enc.Encode(w, msg)
	}
	for _, res := range results {
		for off := 0; off < len(res.audio); off += pollyStreamChunk {
			send("AudioEvent", "application/octet-stream", res.audio[off:min(off+pollyStreamChunk, len(res.audio))])
		}
	}
	closed, _ := json.Marshal(map[string]any{"RequestCharacters": chars})
	send("StreamClosedEvent", "application/json", closed)
	return nil
}
