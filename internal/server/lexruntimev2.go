package server

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"unicode/utf8"
)

// Amazon Lex Runtime V2 (lexruntimev2). It shares the `lex` signing name with
// the Lex V2 models API, so requests are told apart by path: the runtime uses
// /bots/{botId}/botAliases/{botAliasId}/botLocales/{localeId}/sessions/{id},
// which no models operation does (those use lowercase botaliases/botlocales).

var lexRuntimePath = regexp.MustCompile(`^/bots/([^/]+)/botAliases/([^/]+)/botLocales/([^/]+)/sessions/([^/]+)(?:/(text|utterance|conversation))?/?$`)

const lexRTMaxBody = 1 << 20

func (s *Server) handleLexRuntimeV2RESTRouter(w http.ResponseWriter, r *http.Request) bool {
	target, suffix, ok := parseLexRuntimeV2Path(r)
	if !ok {
		return false
	}
	if ok, status, code, msg, _ := s.validateSigV4WithService(r, "lex"); !ok {
		respondLexV2Error(w, status, code, msg)
		return true
	}
	if fault := target.validate(); fault != nil {
		respondLexV2Error(w, fault.status, fault.code, fault.msg)
		return true
	}
	switch {
	case suffix == "" && r.Method == http.MethodGet:
		s.lexRuntimeGetSession(w, target)
	case suffix == "" && r.Method == http.MethodPost:
		s.lexRuntimePutSession(w, r, target)
	case suffix == "" && r.Method == http.MethodDelete:
		s.lexRuntimeDeleteSession(w, target)
	case suffix == "text" && r.Method == http.MethodPost:
		s.lexRuntimeRecognizeText(w, r, target)
	case suffix == "utterance" && r.Method == http.MethodPost:
		s.lexRuntimeRecognizeUtterance(w, r, target)
	case suffix == "conversation" && r.Method == http.MethodPost:
		s.lexRuntimeStartConversation(w, r, target)
	default:
		respondLexV2Error(w, http.StatusMethodNotAllowed, "ValidationException", "unsupported method "+r.Method+" for this Lex runtime path")
	}
	return true
}

// parseLexRuntimeV2Path claims runtime paths that are signed for `lex` (or
// unsigned, or sent to a runtime-v2-lex host).
func parseLexRuntimeV2Path(r *http.Request) (lexRTTarget, string, bool) {
	m := lexRuntimePath.FindStringSubmatch(rawRequestPath(r))
	if m == nil {
		return lexRTTarget{}, "", false
	}
	if hint := strings.TrimSpace(sigV4ServiceHint(r)); hint != "" && hint != "lex" {
		return lexRTTarget{}, "", false
	}
	unescape := func(v string) string {
		if u, err := url.PathUnescape(v); err == nil {
			return u
		}
		return v
	}
	return lexRTTarget{bot: unescape(m[1]), alias: unescape(m[2]), locale: unescape(m[3]), session: unescape(m[4])}, m[5], true
}

func decodeLexRTBody(r *http.Request, into any) error {
	body, err := readBodyBytes(r)
	if err != nil {
		return err
	}
	if len(bytes.TrimSpace(body)) == 0 {
		return nil
	}
	dec := json.NewDecoder(bytes.NewReader(body))
	return dec.Decode(into)
}

func (s *Server) lexRuntimeRecognizeText(w http.ResponseWriter, r *http.Request, t lexRTTarget) {
	var req struct {
		Text              *string            `json:"text"`
		SessionState      *lexRTSessionState `json:"sessionState"`
		RequestAttributes map[string]string  `json:"requestAttributes"`
	}
	if err := decodeLexRTBody(r, &req); err != nil {
		respondLexV2Error(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return
	}
	if req.Text == nil {
		respondLexV2Error(w, http.StatusBadRequest, "ValidationException", "text is required")
		return
	}
	if n := utf8.RuneCountInString(*req.Text); n < 1 || n > 1024 {
		respondLexV2Error(w, http.StatusBadRequest, "ValidationException", "text must be 1-1024 characters")
		return
	}
	if msg := validateLexRTState(req.SessionState); msg != "" {
		respondLexV2Error(w, http.StatusBadRequest, "ValidationException", msg)
		return
	}
	turn, fault := s.lexv2.Recognize(t, *req.Text, req.SessionState)
	if fault != nil {
		respondLexV2Error(w, fault.status, fault.code, fault.msg)
		return
	}
	out := map[string]any{
		"messages":        nonNilLexRT(turn.Messages),
		"sessionState":    turn.State,
		"interpretations": turn.Interpretations,
		"sessionId":       t.session,
	}
	if len(req.RequestAttributes) > 0 {
		out["requestAttributes"] = req.RequestAttributes
	}
	respondLexV2JSON(w, http.StatusOK, out)
}

// lexRTRequestContentTypes are the input formats RecognizeUtterance accepts.
// Speech is accepted but not transcribed: ponytail, it is recognized as an
// empty utterance (FallbackIntent); plug in a transcriber if tests send audio.
var lexRTRequestContentTypes = []string{"text/plain", "audio/l16", "audio/x-l16", "audio/lpcm", "audio/x-cbr-opus-with-preamble"}

func (s *Server) lexRuntimeRecognizeUtterance(w http.ResponseWriter, r *http.Request, t lexRTTarget) {
	requestType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	isText := strings.HasPrefix(requestType, "text/plain")
	known := false
	for _, prefix := range lexRTRequestContentTypes {
		known = known || strings.HasPrefix(requestType, prefix)
	}
	if !known {
		respondLexV2Error(w, http.StatusBadRequest, "ValidationException", "Content-Type must be text/plain; charset=utf-8 or a supported audio format")
		return
	}
	if !lexRTResponseTypeValid(r.Header.Get("Response-Content-Type")) {
		respondLexV2Error(w, http.StatusBadRequest, "ValidationException", "Response-Content-Type must be text/plain; charset=utf-8, audio/mpeg, audio/ogg or audio/pcm")
		return
	}
	var state *lexRTSessionState
	if raw := r.Header.Get("x-amz-lex-session-state"); raw != "" {
		state = &lexRTSessionState{}
		if err := decodeLexRTHeader(raw, state); err != nil {
			respondLexV2Error(w, http.StatusBadRequest, "ValidationException", "x-amz-lex-session-state must be base64-encoded JSON")
			return
		}
		if msg := validateLexRTState(state); msg != "" {
			respondLexV2Error(w, http.StatusBadRequest, "ValidationException", msg)
			return
		}
	}
	var attrs map[string]string
	if raw := r.Header.Get("x-amz-lex-request-attributes"); raw != "" {
		if err := decodeLexRTHeader(raw, &attrs); err != nil {
			respondLexV2Error(w, http.StatusBadRequest, "ValidationException", "x-amz-lex-request-attributes must be base64-encoded JSON")
			return
		}
	}
	body, err := readBodyBytes(r)
	if err != nil || len(body) > lexRTMaxBody {
		respondLexV2Error(w, http.StatusBadRequest, "ValidationException", "inputStream is too large or unreadable")
		return
	}
	transcript, inputMode := "", "Speech"
	if isText {
		transcript, inputMode = strings.TrimSpace(string(body)), "Text"
		if !utf8.ValidString(transcript) || utf8.RuneCountInString(transcript) > 1024 {
			respondLexV2Error(w, http.StatusBadRequest, "ValidationException", "text input must be UTF-8 and at most 1024 characters")
			return
		}
	}
	turn, fault := s.lexv2.Recognize(t, transcript, state)
	if fault != nil {
		respondLexV2Error(w, fault.status, fault.code, fault.msg)
		return
	}
	h := w.Header()
	h.Set("x-amz-lex-input-mode", inputMode)
	h.Set("x-amz-lex-session-id", t.session)
	setLexRTHeader(h, "x-amz-lex-session-state", turn.State)
	setLexRTHeader(h, "x-amz-lex-interpretations", turn.Interpretations)
	if len(turn.Messages) > 0 {
		setLexRTHeader(h, "x-amz-lex-messages", turn.Messages)
	}
	if len(attrs) > 0 {
		setLexRTHeader(h, "x-amz-lex-request-attributes", attrs)
	}
	if transcript != "" {
		h.Set("x-amz-lex-input-transcript", gzipBase64LexRT([]byte(transcript)))
	}
	writeLexRTAudioStream(w, r.Header.Get("Response-Content-Type"), turn.Messages)
}

func (s *Server) lexRuntimePutSession(w http.ResponseWriter, r *http.Request, t lexRTTarget) {
	var req struct {
		SessionState      *lexRTSessionState `json:"sessionState"`
		Messages          []lexRTMessage     `json:"messages"`
		RequestAttributes map[string]string  `json:"requestAttributes"`
	}
	if err := decodeLexRTBody(r, &req); err != nil {
		respondLexV2Error(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return
	}
	if req.SessionState == nil {
		respondLexV2Error(w, http.StatusBadRequest, "ValidationException", "sessionState is required")
		return
	}
	if msg := validateLexRTState(req.SessionState); msg != "" {
		respondLexV2Error(w, http.StatusBadRequest, "ValidationException", msg)
		return
	}
	if msg := validateLexRTMessages(req.Messages); msg != "" {
		respondLexV2Error(w, http.StatusBadRequest, "ValidationException", msg)
		return
	}
	if !lexRTResponseTypeValid(r.Header.Get("ResponseContentType")) {
		respondLexV2Error(w, http.StatusBadRequest, "ValidationException", "ResponseContentType must be text/plain; charset=utf-8, audio/mpeg, audio/ogg or audio/pcm")
		return
	}
	turn, fault := s.lexv2.PutSession(t, *req.SessionState, req.Messages)
	if fault != nil {
		respondLexV2Error(w, fault.status, fault.code, fault.msg)
		return
	}
	h := w.Header()
	h.Set("x-amz-lex-session-id", t.session)
	setLexRTHeader(h, "x-amz-lex-session-state", turn.State)
	if len(turn.Messages) > 0 {
		setLexRTHeader(h, "x-amz-lex-messages", turn.Messages)
	}
	if len(req.RequestAttributes) > 0 {
		setLexRTHeader(h, "x-amz-lex-request-attributes", req.RequestAttributes)
	}
	writeLexRTAudioStream(w, r.Header.Get("ResponseContentType"), turn.Messages)
}

func (s *Server) lexRuntimeGetSession(w http.ResponseWriter, t lexRTTarget) {
	turn, fault := s.lexv2.GetSession(t)
	if fault != nil {
		respondLexV2Error(w, fault.status, fault.code, fault.msg)
		return
	}
	respondLexV2JSON(w, http.StatusOK, map[string]any{
		"sessionId":       t.session,
		"messages":        nonNilLexRT(turn.Messages),
		"interpretations": nonNilLexRT(turn.Interpretations),
		"sessionState":    turn.State,
	})
}

func (s *Server) lexRuntimeDeleteSession(w http.ResponseWriter, t lexRTTarget) {
	if fault := s.lexv2.DeleteSession(t); fault != nil {
		respondLexV2Error(w, fault.status, fault.code, fault.msg)
		return
	}
	respondLexV2JSON(w, http.StatusOK, map[string]any{"botId": t.bot, "botAliasId": t.alias, "localeId": t.locale, "sessionId": t.session})
}

func lexRTResponseTypeValid(v string) bool {
	v = strings.ToLower(strings.TrimSpace(v))
	if v == "" {
		return true
	}
	for _, prefix := range []string{"text/plain", "audio/mpeg", "audio/ogg", "audio/pcm"} {
		if strings.HasPrefix(v, prefix) {
			return true
		}
	}
	return false
}

// writeLexRTAudioStream writes the audioStream body of RecognizeUtterance and
// PutSession. For text/plain (the default) it is the message text; audio
// formats get an empty stream, since nothing is synthesized. The body holds
// model-supplied prompt text, so each content type is a constant set here,
// next to the write, with nosniff.
func writeLexRTAudioStream(w http.ResponseWriter, responseType string, messages []lexRTMessage) {
	h := w.Header()
	h.Set("X-Content-Type-Options", "nosniff")
	var body []byte
	switch v := strings.ToLower(strings.TrimSpace(responseType)); {
	case strings.HasPrefix(v, "audio/mpeg"):
		h.Set("Content-Type", "audio/mpeg")
	case strings.HasPrefix(v, "audio/ogg"):
		h.Set("Content-Type", "audio/ogg")
	case strings.HasPrefix(v, "audio/pcm"):
		h.Set("Content-Type", "audio/pcm")
	default:
		h.Set("Content-Type", "text/plain; charset=utf-8")
		parts := make([]string, 0, len(messages))
		for _, m := range messages {
			if m.Content != "" {
				parts = append(parts, m.Content)
			}
		}
		body = []byte(strings.Join(parts, "\n"))
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}

// setLexRTHeader writes a JSON value the way Lex does in response headers:
// gzip-compressed, then base64-encoded.
func setLexRTHeader(h http.Header, name string, v any) {
	raw, err := json.Marshal(v)
	if err != nil {
		return
	}
	h.Set(name, gzipBase64LexRT(raw))
}

func gzipBase64LexRT(raw []byte) string {
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	_, _ = zw.Write(raw)
	_ = zw.Close()
	return base64.StdEncoding.EncodeToString(buf.Bytes())
}

// decodeLexRTHeader reads a request header that holds base64-encoded JSON.
// SDKs send it uncompressed; a gzip stream is accepted too.
func decodeLexRTHeader(raw string, into any) error {
	data, err := base64.StdEncoding.DecodeString(strings.TrimSpace(raw))
	if err != nil {
		return err
	}
	if len(data) > 2 && data[0] == 0x1f && data[1] == 0x8b {
		zr, err := gzip.NewReader(bytes.NewReader(data))
		if err != nil {
			return err
		}
		if data, err = io.ReadAll(io.LimitReader(zr, lexRTMaxBody)); err != nil {
			return err
		}
	}
	return json.Unmarshal(data, into)
}

func nonNilLexRT[T any](v []T) []T {
	if v == nil {
		return []T{}
	}
	return v
}
