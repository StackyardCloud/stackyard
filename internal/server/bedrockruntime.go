package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream"
)

// brtError is a Bedrock Runtime error answer.
type brtError struct {
	status int
	code   string
	msg    string
}

func brtValidation(format string, args ...any) *brtError {
	return &brtError{http.StatusBadRequest, "ValidationException", fmt.Sprintf(format, args...)}
}

func brtNotFound(format string, args ...any) *brtError {
	return &brtError{http.StatusNotFound, "ResourceNotFoundException", fmt.Sprintf(format, args...)}
}

type brtRequest struct {
	r      *http.Request
	body   []byte
	params map[string]string
}

func (s *Server) handleBedrockRuntimeRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	op, params, ok := bedrockRuntimeRoute(r)
	if !ok {
		return false
	}
	if ok, status, code, msg, _ := s.validateSigV4WithService(r, "bedrock"); !ok {
		respondBedrockRuntimeError(w, &brtError{status, code, msg})
		return true
	}
	body, err := readBodyBytes(r)
	if err != nil {
		respondBedrockRuntimeError(w, brtValidation("unable to read request body"))
		return true
	}
	req := &brtRequest{r: r, body: body, params: params}
	var e *brtError
	switch op {
	case "InvokeModel":
		e = s.brtInvokeModel(w, req, false)
	case "InvokeModelWithResponseStream":
		e = s.brtInvokeModel(w, req, true)
	case "InvokeModelWithBidirectionalStream":
		e = s.brtBidirectional(w, req)
	case "Converse":
		e = s.brtConverse(w, req, false)
	case "ConverseStream":
		e = s.brtConverse(w, req, true)
	case "CountTokens":
		e = s.brtCountTokens(w, req)
	case "ApplyGuardrail":
		e = s.brtApplyGuardrail(w, req)
	case "InvokeGuardrailChecks":
		e = brtGuardrailChecks(w, req)
	case "StartAsyncInvoke":
		e = s.brtStartAsyncInvoke(w, req)
	case "GetAsyncInvoke":
		e = s.brtGetAsyncInvoke(w, req)
	case "ListAsyncInvokes":
		e = s.brtListAsyncInvokes(w, req)
	}
	if e != nil {
		respondBedrockRuntimeError(w, e)
	}
	return true
}

var brtModelActions = map[string]string{
	"invoke":                           "InvokeModel",
	"invoke-with-response-stream":      "InvokeModelWithResponseStream",
	"invoke-with-bidirectional-stream": "InvokeModelWithBidirectionalStream",
	"converse":                         "Converse",
	"converse-stream":                  "ConverseStream",
	"count-tokens":                     "CountTokens",
}

// bedrockRuntimeRoute claims only Bedrock-signed (or bedrock-runtime host)
// requests on the runtime's paths; everything else signed "bedrock" falls
// through to the control-plane router.
func bedrockRuntimeRoute(r *http.Request) (string, map[string]string, bool) {
	host := strings.ToLower(r.Host)
	if strings.TrimSpace(sigV4ServiceHint(r)) != "bedrock" && !strings.Contains(host, "bedrock-runtime") {
		return "", nil, false
	}
	segs := strings.Split(strings.Trim(rawRequestPath(r), "/"), "/")
	unescape := func(parts []string) string {
		joined := strings.Join(parts, "/")
		if v, err := url.PathUnescape(joined); err == nil {
			return v
		}
		return joined
	}
	switch {
	case segs[0] == "model" && len(segs) >= 3 && r.Method == http.MethodPost:
		op, ok := brtModelActions[segs[len(segs)-1]]
		return op, map[string]string{"modelId": unescape(segs[1 : len(segs)-1])}, ok
	case segs[0] == "guardrail" && len(segs) == 5 && segs[2] == "version" && segs[4] == "apply" && r.Method == http.MethodPost:
		return "ApplyGuardrail", map[string]string{"guardrailIdentifier": unescape(segs[1:2]), "guardrailVersion": unescape(segs[3:4])}, true
	case segs[0] == "guardrail-checks" && len(segs) == 2 && segs[1] == "invoke" && r.Method == http.MethodPost:
		return "InvokeGuardrailChecks", nil, true
	case segs[0] == "async-invoke" && len(segs) == 1 && r.Method == http.MethodPost:
		return "StartAsyncInvoke", nil, true
	case segs[0] == "async-invoke" && len(segs) == 1 && r.Method == http.MethodGet:
		return "ListAsyncInvokes", nil, true
	case segs[0] == "async-invoke" && len(segs) > 1 && r.Method == http.MethodGet:
		return "GetAsyncInvoke", map[string]string{"invocationArn": unescape(segs[1:])}, true
	}
	return "", nil, false
}

func respondBedrockRuntimeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondBedrockRuntimeError(w http.ResponseWriter, e *brtError) {
	w.Header().Set("X-Amzn-ErrorType", e.code)
	respondBedrockRuntimeJSON(w, e.status, map[string]string{"__type": e.code, "message": e.msg})
}

// Model families decide the InvokeModel body format and which APIs a model
// supports.
const (
	brtAnthropic   = "anthropic"
	brtTitanText   = "titan-text"
	brtTitanEmbed  = "titan-embed"
	brtNova        = "nova"
	brtLlama       = "llama"
	brtCohereChat  = "cohere-chat"
	brtCohereEmbed = "cohere-embed"
	brtMistral     = "mistral"
	brtNovaReel    = "nova-reel"
	brtNovaSonic   = "nova-sonic"
)

type brtModel struct {
	id     string
	family string
	count  bool // supports CountTokens
	dims   int  // embedding size
}

func (m brtModel) text() bool {
	switch m.family {
	case brtAnthropic, brtTitanText, brtNova, brtLlama, brtCohereChat, brtMistral:
		return true
	}
	return false
}

// ponytail: a fixed catalog of well-known foundation models; extend it when a
// client needs another model ID.
var brtCatalog = func() map[string]brtModel {
	out := map[string]brtModel{}
	add := func(family string, count bool, dims int, ids ...string) {
		for _, id := range ids {
			out[id] = brtModel{id: id, family: family, count: count, dims: dims}
		}
	}
	add(brtAnthropic, false, 0, "anthropic.claude-3-haiku-20240307-v1:0", "anthropic.claude-3-5-sonnet-20240620-v1:0",
		"anthropic.claude-3-sonnet-20240229-v1:0", "anthropic.claude-3-opus-20240229-v1:0", "anthropic.claude-v2", "anthropic.claude-v2:1",
		"anthropic.claude-instant-v1")
	add(brtAnthropic, true, 0, "anthropic.claude-3-5-haiku-20241022-v1:0", "anthropic.claude-3-5-sonnet-20241022-v2:0",
		"anthropic.claude-3-7-sonnet-20250219-v1:0", "anthropic.claude-sonnet-4-20250514-v1:0", "anthropic.claude-opus-4-20250514-v1:0")
	add(brtTitanText, false, 0, "amazon.titan-text-express-v1", "amazon.titan-text-lite-v1", "amazon.titan-text-premier-v1:0")
	add(brtTitanEmbed, false, 1536, "amazon.titan-embed-text-v1")
	add(brtTitanEmbed, false, 1024, "amazon.titan-embed-text-v2:0")
	add(brtNova, false, 0, "amazon.nova-micro-v1:0", "amazon.nova-lite-v1:0", "amazon.nova-pro-v1:0", "amazon.nova-premier-v1:0")
	add(brtNovaReel, false, 0, "amazon.nova-reel-v1:0", "amazon.nova-reel-v1:1")
	add(brtNovaSonic, false, 0, "amazon.nova-sonic-v1:0")
	add(brtLlama, false, 0, "meta.llama3-8b-instruct-v1:0", "meta.llama3-70b-instruct-v1:0", "meta.llama3-1-8b-instruct-v1:0",
		"meta.llama3-1-70b-instruct-v1:0", "meta.llama3-2-1b-instruct-v1:0", "meta.llama3-3-70b-instruct-v1:0")
	add(brtCohereChat, false, 0, "cohere.command-r-v1:0", "cohere.command-r-plus-v1:0")
	add(brtCohereEmbed, false, 1024, "cohere.embed-english-v3", "cohere.embed-multilingual-v3")
	add(brtMistral, false, 0, "mistral.mistral-7b-instruct-v0:2", "mistral.mixtral-8x7b-instruct-v0:1",
		"mistral.mistral-large-2402-v1:0", "mistral.mistral-small-2402-v1:0")
	return out
}()

var brtProfilePrefixes = []string{"us.", "eu.", "apac.", "us-gov.", "global.", "jp.", "au.", "ca."}

// brtResolveModel maps a modelId (model ID, inference profile ID, or a
// foundation-model, inference-profile, application-inference-profile,
// provisioned-model, custom-model or imported-model ARN) to its model and ARN.
func (s *Server) brtResolveModel(id string) (brtModel, string, *brtError) {
	id = strings.TrimSpace(id)
	if id == "" {
		return brtModel{}, "", brtValidation("modelId is required.")
	}
	if strings.HasPrefix(id, "arn:") {
		parts := strings.SplitN(id, ":", 6)
		if len(parts) < 6 || parts[2] != "bedrock" {
			return brtModel{}, "", brtValidation("The provided model identifier is invalid.")
		}
		resource := parts[5]
		switch kind, rest, _ := strings.Cut(resource, "/"); kind {
		case "foundation-model", "inference-profile":
			m, _, e := s.brtResolveModel(rest)
			return m, id, e
		case "application-inference-profile", "provisioned-model":
			base, ok := s.bedrock.runtimeBaseModel(id)
			if !ok {
				return brtModel{}, "", brtNotFound("The model %s doesn't exist.", id)
			}
			m, _, e := s.brtResolveModel(base)
			return m, id, e
		case "custom-model":
			// arn:...:custom-model/<base model id>/<suffix>
			if i := strings.LastIndex(rest, "/"); i > 0 {
				if m, ok := brtCatalog[rest[:i]]; ok {
					return m, id, nil
				}
			}
			return brtModel{}, "", brtNotFound("The model %s doesn't exist.", id)
		case "imported-model":
			// ponytail: imported models are Llama-architecture; other
			// architectures would need their own body format.
			return brtModel{id: id, family: brtLlama}, id, nil
		}
		return brtModel{}, "", brtValidation("The provided model identifier is invalid.")
	}
	base := id
	for _, prefix := range brtProfilePrefixes {
		base = strings.TrimPrefix(base, prefix)
	}
	m, ok := brtCatalog[base]
	if !ok {
		return brtModel{}, "", brtValidation("The provided model identifier is invalid.")
	}
	return m, "arn:aws:bedrock:us-east-1::foundation-model/" + m.id, nil
}

// brtTokens is the emulator's token count: one token per whitespace-separated
// word.
func brtTokens(text string) int {
	return len(strings.Fields(text))
}

// brtGeneration is a deterministic completion: an echo of the prompt, cut at
// the first stop sequence or at maxTokens words.
type brtGeneration struct {
	text    string
	reason  string // "end", "max" or "stop"
	stopSeq string
}

func brtGenerate(prompt string, maxTokens int, stops []string) brtGeneration {
	words := strings.Fields(prompt)
	if len(words) > 32 {
		words = words[len(words)-32:]
	}
	text := "Stackyard response: " + strings.Join(words, " ")
	if len(words) == 0 {
		text = "Stackyard response."
	}
	g := brtGeneration{text: text, reason: "end"}
	cut := -1
	for _, stop := range stops {
		if i := strings.Index(text, stop); stop != "" && i >= 0 && (cut < 0 || i < cut) {
			cut, g.stopSeq = i, stop
		}
	}
	if cut >= 0 {
		g.text, g.reason = strings.TrimRight(text[:cut], " "), "stop"
	}
	if out := strings.Fields(g.text); maxTokens > 0 && len(out) > maxTokens {
		g.text, g.reason, g.stopSeq = strings.Join(out[:maxTokens], " "), "max", ""
	}
	return g
}

// brtChunks splits text into the deltas a stream sends, a few words at a time.
func brtChunks(text string) []string {
	words := strings.Fields(text)
	var out []string
	for i := 0; i < len(words); i += 4 {
		end := min(i+4, len(words))
		chunk := strings.Join(words[i:end], " ")
		if i > 0 {
			chunk = " " + chunk
		}
		out = append(out, chunk)
	}
	return out
}

// brtEmbedding is a deterministic unit-length vector derived from the text.
func brtEmbedding(text string, dims int) []float64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(text))
	seed := h.Sum64()
	out := make([]float64, dims)
	var norm float64
	for i := range out {
		seed = seed*6364136223846793005 + 1442695040888963407
		out[i] = float64(int64(seed>>11))/float64(1<<52) - 1
		norm += out[i] * out[i]
	}
	norm = math.Sqrt(norm)
	for i := range out {
		out[i] = math.Round(out[i]/norm*1e6) / 1e6
	}
	return out
}

func brtDecodeObject(body []byte, what string) (map[string]any, *brtError) {
	if len(bytes.TrimSpace(body)) == 0 {
		return map[string]any{}, nil
	}
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()
	var out map[string]any
	if err := dec.Decode(&out); err != nil || out == nil {
		return nil, brtValidation("Malformed input request: %s must be a JSON object, please reformat your input and try again.", what)
	}
	return out, nil
}

func brtNumber(v any) (float64, bool) {
	switch t := v.(type) {
	case json.Number:
		f, err := t.Float64()
		return f, err == nil
	case float64:
		return t, true
	case int:
		return float64(t), true
	}
	return 0, false
}

// brtRange checks an optional numeric field; whole requires an integer.
func brtRange(m map[string]any, key, path string, least, most float64, whole bool) (float64, bool, *brtError) {
	raw, ok := m[key]
	if !ok || raw == nil {
		return 0, false, nil
	}
	f, isNum := brtNumber(raw)
	if !isNum || (whole && f != math.Trunc(f)) {
		kind := "number"
		if whole {
			kind = "integer"
		}
		return 0, false, brtValidation("Malformed input request: %s must be an %s, please reformat your input and try again.", path, kind)
	}
	if f < least || f > most {
		return 0, false, brtValidation("Malformed input request: %s must be between %v and %v, please reformat your input and try again.", path, least, most)
	}
	return f, true, nil
}

func brtStrings(v any) ([]string, bool) {
	if v == nil {
		return nil, true
	}
	list, ok := v.([]any)
	if !ok {
		return nil, false
	}
	out := make([]string, 0, len(list))
	for _, item := range list {
		s, ok := item.(string)
		if !ok {
			return nil, false
		}
		out = append(out, s)
	}
	return out, true
}

func brtOnly(m map[string]any, allowed ...string) *brtError {
	for key := range m {
		found := false
		for _, a := range allowed {
			if key == a {
				found = true
				break
			}
		}
		if !found {
			return brtValidation("Malformed input request: #: extraneous key [%s] is not permitted, please reformat your input and try again.", key)
		}
	}
	return nil
}

// brtEnumHeader checks an optional enum request header.
func brtEnumHeader(r *http.Request, name string, allowed ...string) (string, *brtError) {
	v := strings.TrimSpace(r.Header.Get(name))
	if v == "" {
		return "", nil
	}
	for _, a := range allowed {
		if v == a {
			return v, nil
		}
	}
	return "", brtValidation("1 validation error detected: Value '%s' at '%s' failed to satisfy constraint: Member must satisfy enum value set: [%s]", v, name, strings.Join(allowed, ", "))
}

// brtStream writes AWS event-stream frames.
type brtStream struct {
	w   http.ResponseWriter
	enc *eventstream.Encoder
}

func brtStartStream(w http.ResponseWriter) *brtStream {
	w.Header().Set("Content-Type", "application/vnd.amazon.eventstream")
	w.WriteHeader(http.StatusOK)
	return &brtStream{w: w, enc: eventstream.NewEncoder()}
}

func (st *brtStream) event(eventType string, payload any) {
	body, _ := json.Marshal(payload)
	msg := eventstream.Message{Payload: body}
	msg.Headers.Set(":message-type", eventstream.StringValue("event"))
	msg.Headers.Set(":event-type", eventstream.StringValue(eventType))
	msg.Headers.Set(":content-type", eventstream.StringValue("application/json"))
	_ = st.enc.Encode(st.w, msg)
	if f, ok := st.w.(http.Flusher); ok {
		f.Flush()
	}
}

// brtReadEvents decodes an event-stream request body.
func brtReadEvents(body []byte) ([]eventstream.Message, error) {
	dec := eventstream.NewDecoder()
	r := bytes.NewReader(body)
	var out []eventstream.Message
	for r.Len() > 0 {
		msg, err := dec.Decode(r, nil)
		if err != nil {
			return nil, err
		}
		out = append(out, msg)
	}
	if len(out) == 0 {
		return nil, io.ErrUnexpectedEOF
	}
	return out, nil
}

func brtHeaderString(msg eventstream.Message, name string) string {
	if v := msg.Headers.Get(name); v != nil {
		return v.String()
	}
	return ""
}

func brtID(prefix string, parts ...string) string {
	h := fnv.New64a()
	for _, p := range parts {
		_, _ = h.Write([]byte(p))
	}
	return fmt.Sprintf("%s%016x", prefix, h.Sum64())
}
