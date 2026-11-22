package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream"
)

// StartConversation is a bidirectional event stream: the client sends
// ConfigurationEvent / TextInputEvent / DTMFInputEvent / ... frames on the
// request body and reads TextResponseEvent / IntentResultEvent frames from the
// response while the request is still open. Each turn is answered as soon as
// its frame arrives, so a streaming SDK client and a client that sends the
// whole body at once (the coverage script) both work.
//
// SDK clients sign the request with STREAMING-AWS4-HMAC-SHA256-EVENTS and wrap
// each event in an envelope carrying :chunk-signature. The headers signature
// is checked; ponytail: the per-event chunk signatures are not, add that if a
// test depends on rejecting tampered frames.
//
// Audio input is accepted and ignored (no speech recognition).

const lexRTMaxEventPayload = 256 << 10

func (s *Server) lexRuntimeStartConversation(w http.ResponseWriter, r *http.Request, t lexRTTarget) {
	if mode := r.Header.Get("x-amz-lex-conversation-mode"); mode != "" && mode != "AUDIO" && mode != "TEXT" {
		respondLexV2Error(w, http.StatusBadRequest, "ValidationException", "x-amz-lex-conversation-mode must be AUDIO or TEXT")
		return
	}
	if fault := s.lexv2.ConversationTarget(t); fault != nil {
		respondLexV2Error(w, fault.status, fault.code, fault.msg)
		return
	}

	rc := http.NewResponseController(w)
	_ = rc.EnableFullDuplex() // HTTP/1.1 only; HTTP/2 is already full duplex
	w.Header().Set("Content-Type", "application/vnd.amazon.eventstream")
	w.WriteHeader(http.StatusOK)
	_ = rc.Flush()

	conv := &lexRTConversation{s: s, t: t, w: w, rc: rc, enc: eventstream.NewEncoder()}
	dec := eventstream.NewDecoder()
	buf := make([]byte, 0, 4096)
	for {
		msg, err := dec.Decode(r.Body, buf)
		if err != nil {
			if !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
				conv.exception("ValidationException", "malformed event stream frame: "+err.Error())
			}
			return
		}
		// A signed envelope carries the real event as its payload; an empty
		// signed envelope marks the end of the client's stream.
		if msg.Headers.Get(":chunk-signature") != nil {
			if len(msg.Payload) == 0 {
				return
			}
			inner, err := dec.Decode(bytes.NewReader(msg.Payload), nil)
			if err != nil {
				conv.exception("ValidationException", "malformed signed event: "+err.Error())
				return
			}
			msg = inner
		}
		if len(msg.Payload) > lexRTMaxEventPayload {
			conv.exception("ValidationException", "event payload is too large")
			return
		}
		if !conv.handle(lexRTHeaderString(msg.Headers, ":event-type"), msg.Payload) {
			return
		}
	}
}

type lexRTConversation struct {
	s      *Server
	t      lexRTTarget
	w      http.ResponseWriter
	rc     *http.ResponseController
	enc    *eventstream.Encoder
	attrs  map[string]string
	nextID int
}

// handle processes one client event and reports whether to keep reading.
func (c *lexRTConversation) handle(eventType string, payload []byte) bool {
	switch eventType {
	case "ConfigurationEvent":
		var ev struct {
			ResponseContentType string             `json:"responseContentType"`
			SessionState        *lexRTSessionState `json:"sessionState"`
			RequestAttributes   map[string]string  `json:"requestAttributes"`
			WelcomeMessages     []lexRTMessage     `json:"welcomeMessages"`
		}
		if json.Unmarshal(payload, &ev) != nil {
			return c.exception("ValidationException", "ConfigurationEvent is not valid JSON")
		}
		if ev.ResponseContentType == "" || !lexRTResponseTypeValid(ev.ResponseContentType) {
			return c.exception("ValidationException", "ConfigurationEvent.responseContentType must be text/plain; charset=utf-8, audio/mpeg, audio/ogg or audio/pcm")
		}
		if msg := validateLexRTState(ev.SessionState); msg != "" {
			return c.exception("ValidationException", msg)
		}
		if msg := validateLexRTMessages(ev.WelcomeMessages); msg != "" {
			return c.exception("ValidationException", msg)
		}
		c.attrs = ev.RequestAttributes
		if ev.SessionState != nil {
			if _, fault := c.s.lexv2.PutSession(c.t, *ev.SessionState, ev.WelcomeMessages); fault != nil {
				return c.exception(fault.code, fault.msg)
			}
		}
		if len(ev.WelcomeMessages) > 0 {
			c.event("TextResponseEvent", map[string]any{"messages": ev.WelcomeMessages})
		}
		return true
	case "TextInputEvent":
		var ev struct {
			Text string `json:"text"`
		}
		if json.Unmarshal(payload, &ev) != nil || ev.Text == "" {
			return c.exception("ValidationException", "TextInputEvent.text is required")
		}
		return c.turn(ev.Text, "Text")
	case "DTMFInputEvent":
		var ev struct {
			InputCharacter string `json:"inputCharacter"`
		}
		if json.Unmarshal(payload, &ev) != nil || ev.InputCharacter == "" {
			return c.exception("ValidationException", "DTMFInputEvent.inputCharacter is required")
		}
		return c.turn(ev.InputCharacter, "DTMF")
	case "DisconnectionEvent":
		return false
	default: // AudioInputEvent, PlaybackCompletionEvent and unknown events
		return true
	}
}

func (c *lexRTConversation) turn(text, inputMode string) bool {
	turn, fault := c.s.lexv2.Recognize(c.t, text, nil)
	if fault != nil {
		return c.exception(fault.code, fault.msg)
	}
	result := map[string]any{
		"inputMode":       inputMode,
		"interpretations": turn.Interpretations,
		"sessionState":    turn.State,
		"sessionId":       c.t.session,
	}
	if len(c.attrs) > 0 {
		result["requestAttributes"] = c.attrs
	}
	c.event("IntentResultEvent", result)
	if len(turn.Messages) > 0 {
		c.event("TextResponseEvent", map[string]any{"messages": turn.Messages})
	}
	return true
}

func (c *lexRTConversation) event(eventType string, body map[string]any) {
	c.nextID++
	body["eventId"] = fmt.Sprintf("RESPONSE-%d", c.nextID)
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

// exception writes a modeled exception frame and ends the conversation.
func (c *lexRTConversation) exception(code, message string) bool {
	payload, _ := json.Marshal(map[string]string{"message": message})
	msg := eventstream.Message{Payload: payload}
	msg.Headers.Set(":message-type", eventstream.StringValue("exception"))
	msg.Headers.Set(":exception-type", eventstream.StringValue(code))
	msg.Headers.Set(":content-type", eventstream.StringValue("application/json"))
	c.write(msg)
	return false
}

func (c *lexRTConversation) write(msg eventstream.Message) {
	if err := c.enc.Encode(c.w, msg); err == nil {
		_ = c.rc.Flush()
	}
}

func lexRTHeaderString(h eventstream.Headers, name string) string {
	if v := h.Get(name); v != nil {
		return strings.TrimSpace(v.String())
	}
	return ""
}

// ConversationTarget checks that a conversation can start against the target.
func (s *lexV2Store) ConversationTarget(t lexRTTarget) *lexRTFault {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, _, fault := s.runtimeLocaleLocked(t)
	return fault
}
