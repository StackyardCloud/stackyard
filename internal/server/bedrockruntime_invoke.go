package server

import (
	"encoding/base64"
	"encoding/json"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// brtInvocation is a validated InvokeModel body.
type brtInvocation struct {
	model     brtModel
	body      map[string]any
	prompt    string // every input text, for token counts and guardrails
	last      string // the text the completion echoes
	maxTokens int
	stops     []string
	textMode  bool   // Anthropic Text Completions (prompt / max_tokens_to_sample)
	tool      string // Anthropic tool forced by tool_choice
}

func (s *Server) brtInvokeModel(w http.ResponseWriter, req *brtRequest, stream bool) *brtError {
	start := time.Now()
	model, _, e := s.brtResolveModel(req.params["modelId"])
	if e != nil {
		return e
	}
	switch {
	case model.family == brtNovaReel:
		return brtValidation("The provided model doesn't support on-demand invocation. Use StartAsyncInvoke instead.")
	case model.family == brtNovaSonic:
		return brtValidation("The provided model only supports InvokeModelWithBidirectionalStream.")
	case stream && !model.text():
		return brtValidation("The model is unsupported for streaming.")
	}
	if ct := req.r.Header.Get("Content-Type"); ct != "" && !strings.HasPrefix(strings.ToLower(ct), "application/json") {
		return brtValidation("The provided Content-Type %s is not supported; use application/json.", ct)
	}
	acceptHeader := "Accept"
	if stream {
		acceptHeader = "X-Amzn-Bedrock-Accept"
	}
	if accept := strings.TrimSpace(req.r.Header.Get(acceptHeader)); accept != "" && accept != "*/*" && !strings.HasPrefix(accept, "application/json") {
		return brtValidation("The provided Accept type %s is not supported; use application/json.", accept)
	}
	trace, e := brtEnumHeader(req.r, "X-Amzn-Bedrock-Trace", "ENABLED", "DISABLED", "ENABLED_FULL")
	if e != nil {
		return e
	}
	latency, e := brtEnumHeader(req.r, "X-Amzn-Bedrock-PerformanceConfig-Latency", "standard", "optimized")
	if e != nil {
		return e
	}
	tier, e := brtEnumHeader(req.r, "X-Amzn-Bedrock-Service-Tier", "priority", "default", "flex", "reserved")
	if e != nil {
		return e
	}
	body, e := brtDecodeObject(req.body, "body")
	if e != nil {
		return e
	}
	inv, e := brtParseInvoke(model, body)
	if e != nil {
		return e
	}
	var guard *brtGuardrailResult
	gid, gver := req.r.Header.Get("X-Amzn-Bedrock-GuardrailIdentifier"), req.r.Header.Get("X-Amzn-Bedrock-GuardrailVersion")
	if gid != "" || gver != "" {
		cfg, e := s.brtGuardrail(gid, gver)
		if e != nil {
			return e
		}
		res := brtEvaluateGuardrail(cfg, []string{inv.prompt}, "INPUT")
		guard = &res
	}
	gen := brtGenerate(inv.last, inv.maxTokens, inv.stops)
	if guard != nil && guard.intervened {
		gen = brtGeneration{text: strings.Join(guard.outputs, "\n"), reason: "end"}
		inv.tool = ""
	}
	inTok, outTok := brtTokens(inv.prompt), brtTokens(gen.text)
	metrics := map[string]any{"inputTokenCount": inTok, "outputTokenCount": outTok, "invocationLatency": time.Since(start).Milliseconds(), "firstByteLatency": 0}
	extra := map[string]any{}
	if guard != nil {
		extra["amazon-bedrock-guardrailAction"] = "NONE"
		if guard.intervened {
			extra["amazon-bedrock-guardrailAction"] = "INTERVENED"
		}
		if trace == "ENABLED" || trace == "ENABLED_FULL" {
			extra["amazon-bedrock-trace"] = map[string]any{"guardrail": map[string]any{"input": map[string]any{gid: guard.assessment}}}
		}
	}
	if latency != "" {
		w.Header().Set("X-Amzn-Bedrock-PerformanceConfig-Latency", latency)
	}
	if tier != "" {
		w.Header().Set("X-Amzn-Bedrock-Service-Tier", tier)
	}
	if !stream {
		w.Header().Set("X-Amzn-Bedrock-Input-Token-Count", strconv.Itoa(inTok))
		w.Header().Set("X-Amzn-Bedrock-Output-Token-Count", strconv.Itoa(outTok))
		w.Header().Set("X-Amzn-Bedrock-Invocation-Latency", strconv.FormatInt(time.Since(start).Milliseconds(), 10))
		out := brtRender(inv, gen, inTok, outTok)
		for k, v := range extra {
			out[k] = v
		}
		respondBedrockRuntimeJSON(w, http.StatusOK, out)
		return nil
	}
	chunks := brtRenderChunks(inv, gen, inTok, outTok)
	last := chunks[len(chunks)-1]
	last["amazon-bedrock-invocationMetrics"] = metrics
	for k, v := range extra {
		last[k] = v
	}
	w.Header().Set("X-Amzn-Bedrock-Content-Type", "application/json")
	st := brtStartStream(w)
	for _, chunk := range chunks {
		raw, _ := json.Marshal(chunk)
		st.event("chunk", map[string]any{"bytes": base64.StdEncoding.EncodeToString(raw)})
	}
	return nil
}

// brtParseInvoke validates a model-family request body.
func brtParseInvoke(model brtModel, body map[string]any) (*brtInvocation, *brtError) {
	inv := &brtInvocation{model: model, body: body, maxTokens: 512}
	var e *brtError
	switch model.family {
	case brtAnthropic:
		e = brtParseAnthropic(inv)
	case brtTitanText:
		e = brtParseTitanText(inv)
	case brtTitanEmbed:
		e = brtParseTitanEmbed(inv)
	case brtNova:
		e = brtParseNova(inv)
	case brtLlama:
		e = brtParsePrompt(inv, "max_gen_len", 2048, "prompt", "temperature", "top_p", "max_gen_len", "images")
	case brtMistral:
		e = brtParsePrompt(inv, "max_tokens", 8192, "prompt", "max_tokens", "stop", "temperature", "top_p", "top_k")
	case brtCohereChat:
		e = brtParseCohereChat(inv)
	case brtCohereEmbed:
		e = brtParseCohereEmbed(inv)
	}
	return inv, e
}

func brtRequireString(m map[string]any, key string) (string, *brtError) {
	v, ok := m[key].(string)
	if !ok || strings.TrimSpace(v) == "" {
		return "", brtValidation("Malformed input request: #: required key [%s] not found, please reformat your input and try again.", key)
	}
	return v, nil
}

func brtCommonLimits(m map[string]any, prefix string, keys map[string][2]float64) *brtError {
	for key, bounds := range keys {
		if _, _, e := brtRange(m, key, prefix+key, bounds[0], bounds[1], false); e != nil {
			return e
		}
	}
	return nil
}

func brtSetMax(inv *brtInvocation, m map[string]any, key, path string, most float64) *brtError {
	v, ok, e := brtRange(m, key, path, 1, most, true)
	if e != nil {
		return e
	}
	if ok {
		inv.maxTokens = int(v)
	}
	return nil
}

func brtSetStops(inv *brtInvocation, m map[string]any, key string) *brtError {
	stops, ok := brtStrings(m[key])
	if !ok {
		return brtValidation("Malformed input request: %s must be an array of strings, please reformat your input and try again.", key)
	}
	inv.stops = stops
	return nil
}

func brtParseAnthropic(inv *brtInvocation) *brtError {
	b := inv.body
	if _, ok := b["prompt"]; ok {
		if e := brtOnly(b, "prompt", "max_tokens_to_sample", "temperature", "top_p", "top_k", "stop_sequences", "anthropic_version"); e != nil {
			return e
		}
		prompt, e := brtRequireString(b, "prompt")
		if e != nil {
			return e
		}
		if !strings.HasPrefix(prompt, "\n\nHuman:") || !strings.Contains(prompt, "\n\nAssistant:") {
			return brtValidation("Invalid prompt: prompt must start with \"\\n\\nHuman:\" turn and end with \"\\n\\nAssistant:\"")
		}
		if _, ok := b["max_tokens_to_sample"]; !ok {
			return brtValidation("Malformed input request: #: required key [max_tokens_to_sample] not found, please reformat your input and try again.")
		}
		inv.textMode, inv.prompt, inv.last = true, prompt, strings.ReplaceAll(strings.ReplaceAll(prompt, "Human:", ""), "Assistant:", "")
		if e := brtSetMax(inv, b, "max_tokens_to_sample", "max_tokens_to_sample", 100000); e != nil {
			return e
		}
		if e := brtCommonLimits(b, "", map[string][2]float64{"temperature": {0, 1}, "top_p": {0, 1}, "top_k": {0, 500}}); e != nil {
			return e
		}
		return brtSetStops(inv, b, "stop_sequences")
	}
	if e := brtOnly(b, "anthropic_version", "anthropic_beta", "max_tokens", "messages", "system", "temperature", "top_p", "top_k", "stop_sequences", "tools", "tool_choice", "thinking", "metadata"); e != nil {
		return e
	}
	if v, _ := b["anthropic_version"].(string); v != "bedrock-2023-05-31" {
		return brtValidation("Malformed input request: #: required key [anthropic_version] not found or invalid; use \"bedrock-2023-05-31\", please reformat your input and try again.")
	}
	if _, ok := b["max_tokens"]; !ok {
		return brtValidation("Malformed input request: #: required key [max_tokens] not found, please reformat your input and try again.")
	}
	if e := brtSetMax(inv, b, "max_tokens", "max_tokens", 200000); e != nil {
		return e
	}
	if e := brtCommonLimits(b, "", map[string][2]float64{"temperature": {0, 1}, "top_p": {0, 1}, "top_k": {0, 500}}); e != nil {
		return e
	}
	if e := brtSetStops(inv, b, "stop_sequences"); e != nil {
		return e
	}
	var texts []string
	switch sys := b["system"].(type) {
	case nil:
	case string:
		texts = append(texts, sys)
	case []any:
		for _, block := range sys {
			bm, _ := block.(map[string]any)
			texts = append(texts, bedrockString(bm, "text"))
		}
	default:
		return brtValidation("Malformed input request: system must be a string or an array of text blocks, please reformat your input and try again.")
	}
	messages, _ := b["messages"].([]any)
	if len(messages) == 0 {
		return brtValidation("Malformed input request: #: required key [messages] not found, please reformat your input and try again.")
	}
	for i, raw := range messages {
		m, _ := raw.(map[string]any)
		role := bedrockString(m, "role")
		if role != "user" && role != "assistant" {
			return brtValidation("Malformed input request: messages.%d.role must be user or assistant, please reformat your input and try again.", i)
		}
		if i == 0 && role != "user" {
			return brtValidation("messages: the first message must use the \"user\" role")
		}
		text, e := brtAnthropicContent(m["content"], i)
		if e != nil {
			return e
		}
		texts = append(texts, text)
		if role == "user" {
			inv.last = text
		}
	}
	inv.prompt = strings.Join(texts, "\n")
	if choice, ok := b["tool_choice"].(map[string]any); ok {
		tools, _ := b["tools"].([]any)
		if len(tools) == 0 {
			return brtValidation("tool_choice may only be specified while providing tools.")
		}
		switch bedrockString(choice, "type") {
		case "any":
			first, _ := tools[0].(map[string]any)
			inv.tool = bedrockString(first, "name")
		case "tool":
			inv.tool = bedrockString(choice, "name")
		}
	}
	return nil
}

func brtAnthropicContent(content any, i int) (string, *brtError) {
	switch c := content.(type) {
	case string:
		return c, nil
	case []any:
		var parts []string
		for j, raw := range c {
			block, _ := raw.(map[string]any)
			switch typ := bedrockString(block, "type"); typ {
			case "text":
				t, ok := block["text"].(string)
				if !ok {
					return "", brtValidation("messages.%d.content.%d.text: Field required", i, j)
				}
				parts = append(parts, t)
			case "tool_result":
				if s, ok := block["content"].(string); ok {
					parts = append(parts, s)
				}
			case "image", "document", "tool_use", "thinking", "redacted_thinking":
			default:
				return "", brtValidation("messages.%d.content.%d.type: Input tag '%s' found using 'type' does not match any of the expected tags", i, j, typ)
			}
		}
		return strings.Join(parts, " "), nil
	}
	return "", brtValidation("messages.%d.content: Input should be a valid list or string", i)
}

func brtParseTitanText(inv *brtInvocation) *brtError {
	b := inv.body
	if e := brtOnly(b, "inputText", "textGenerationConfig"); e != nil {
		return e
	}
	text, e := brtRequireString(b, "inputText")
	if e != nil {
		return e
	}
	inv.prompt, inv.last = text, text
	cfg, _ := b["textGenerationConfig"].(map[string]any)
	if cfg == nil {
		return nil
	}
	if e := brtSetMax(inv, cfg, "maxTokenCount", "textGenerationConfig.maxTokenCount", 8192); e != nil {
		return e
	}
	if e := brtCommonLimits(cfg, "textGenerationConfig.", map[string][2]float64{"temperature": {0, 1}, "topP": {0, 1}}); e != nil {
		return e
	}
	return brtSetStops(inv, cfg, "stopSequences")
}

func brtParseTitanEmbed(inv *brtInvocation) *brtError {
	b := inv.body
	allowed := []string{"inputText"}
	if inv.model.id == "amazon.titan-embed-text-v2:0" {
		allowed = append(allowed, "dimensions", "normalize", "embeddingTypes")
	}
	if e := brtOnly(b, allowed...); e != nil {
		return e
	}
	text, e := brtRequireString(b, "inputText")
	if e != nil {
		return e
	}
	inv.prompt = text
	types, ok := brtStrings(b["embeddingTypes"])
	for _, t := range types {
		ok = ok && (t == "float" || t == "binary")
	}
	if !ok {
		return brtValidation("Malformed input request: embeddingTypes entries must be float or binary, please reformat your input and try again.")
	}
	if raw, ok := b["dimensions"]; ok {
		d, _ := brtNumber(raw)
		if d != 256 && d != 512 && d != 1024 {
			return brtValidation("Malformed input request: dimensions must be one of 256, 512 or 1024, please reformat your input and try again.")
		}
		inv.model.dims = int(d)
	}
	return nil
}

func brtParseNova(inv *brtInvocation) *brtError {
	b := inv.body
	if e := brtOnly(b, "schemaVersion", "messages", "system", "inferenceConfig", "toolConfig"); e != nil {
		return e
	}
	var texts []string
	if sys, ok := b["system"].([]any); ok {
		for _, block := range sys {
			bm, _ := block.(map[string]any)
			texts = append(texts, bedrockString(bm, "text"))
		}
	}
	messages, _ := b["messages"].([]any)
	if len(messages) == 0 {
		return brtValidation("Malformed input request: #: required key [messages] not found, please reformat your input and try again.")
	}
	for i, raw := range messages {
		m, _ := raw.(map[string]any)
		role := bedrockString(m, "role")
		if want := []string{"user", "assistant"}[i%2]; role != want {
			return brtValidation("Malformed input request: messages must alternate between user and assistant roles, starting with user, please reformat your input and try again.")
		}
		content, _ := m["content"].([]any)
		if len(content) == 0 {
			return brtValidation("Malformed input request: messages.%d.content must not be empty, please reformat your input and try again.", i)
		}
		for _, block := range content {
			bm, _ := block.(map[string]any)
			if t, ok := bm["text"].(string); ok {
				texts = append(texts, t)
				if role == "user" {
					inv.last = t
				}
			}
		}
	}
	inv.prompt = strings.Join(texts, "\n")
	cfg, _ := b["inferenceConfig"].(map[string]any)
	if cfg == nil {
		return nil
	}
	for _, key := range []string{"max_new_tokens", "maxTokens"} {
		if e := brtSetMax(inv, cfg, key, "inferenceConfig."+key, 10000); e != nil {
			return e
		}
	}
	if e := brtCommonLimits(cfg, "inferenceConfig.", map[string][2]float64{"temperature": {0, 1}, "topP": {0, 1}, "top_p": {0, 1}, "topK": {0, 128}, "top_k": {0, 128}}); e != nil {
		return e
	}
	return brtSetStops(inv, cfg, "stopSequences")
}

// brtParsePrompt covers the prompt-string families (Meta Llama, Mistral).
func brtParsePrompt(inv *brtInvocation, maxKey string, most float64, allowed ...string) *brtError {
	b := inv.body
	if e := brtOnly(b, allowed...); e != nil {
		return e
	}
	prompt, e := brtRequireString(b, "prompt")
	if e != nil {
		return e
	}
	inv.prompt, inv.last = prompt, prompt
	if e := brtSetMax(inv, b, maxKey, maxKey, most); e != nil {
		return e
	}
	if e := brtCommonLimits(b, "", map[string][2]float64{"temperature": {0, 1}, "top_p": {0, 1}, "top_k": {0, 200}}); e != nil {
		return e
	}
	return brtSetStops(inv, b, "stop")
}

func brtParseCohereChat(inv *brtInvocation) *brtError {
	b := inv.body
	if e := brtOnly(b, "message", "chat_history", "documents", "preamble", "max_tokens", "temperature", "p", "k", "stop_sequences",
		"search_queries_only", "prompt_truncation", "frequency_penalty", "presence_penalty", "seed", "return_prompt", "tools",
		"tool_results", "raw_prompting"); e != nil {
		return e
	}
	msg, e := brtRequireString(b, "message")
	if e != nil {
		return e
	}
	texts := []string{bedrockString(b, "preamble")}
	history, _ := b["chat_history"].([]any)
	for i, raw := range history {
		h, _ := raw.(map[string]any)
		switch bedrockString(h, "role") {
		case "USER", "CHATBOT", "SYSTEM", "TOOL":
		default:
			return brtValidation("Malformed input request: chat_history.%d.role must be USER, CHATBOT, SYSTEM or TOOL, please reformat your input and try again.", i)
		}
		texts = append(texts, bedrockString(h, "message"))
	}
	inv.prompt, inv.last = strings.TrimSpace(strings.Join(append(texts, msg), "\n")), msg
	if e := brtSetMax(inv, b, "max_tokens", "max_tokens", 4000); e != nil {
		return e
	}
	if e := brtCommonLimits(b, "", map[string][2]float64{"temperature": {0, 1}, "p": {0.01, 0.99}, "k": {0, 500}}); e != nil {
		return e
	}
	if e := brtSetStops(inv, b, "stop_sequences"); e != nil {
		return e
	}
	if len(inv.stops) > 4 {
		return brtValidation("Malformed input request: stop_sequences accepts at most 4 entries, please reformat your input and try again.")
	}
	return nil
}

var brtCohereInputTypes = []string{"search_document", "search_query", "classification", "clustering", "image"}
var brtCohereEmbeddingTypes = []string{"float", "int8", "uint8", "binary", "ubinary"}

func brtParseCohereEmbed(inv *brtInvocation) *brtError {
	b := inv.body
	if e := brtOnly(b, "texts", "images", "input_type", "truncate", "embedding_types"); e != nil {
		return e
	}
	texts, ok := brtStrings(b["texts"])
	if !ok || (len(texts) == 0 && b["images"] == nil) {
		return brtValidation("Malformed input request: #: required key [texts] not found, please reformat your input and try again.")
	}
	if len(texts) > 96 {
		return brtValidation("Malformed input request: texts accepts at most 96 entries, please reformat your input and try again.")
	}
	if !brtIn(bedrockString(b, "input_type"), brtCohereInputTypes) {
		return brtValidation("Malformed input request: input_type must be one of %s, please reformat your input and try again.", strings.Join(brtCohereInputTypes, ", "))
	}
	if t := bedrockString(b, "truncate"); t != "" && !brtIn(t, []string{"NONE", "START", "END"}) {
		return brtValidation("Malformed input request: truncate must be NONE, START or END, please reformat your input and try again.")
	}
	types, ok := brtStrings(b["embedding_types"])
	for _, t := range types {
		ok = ok && brtIn(t, brtCohereEmbeddingTypes)
	}
	if !ok {
		return brtValidation("Malformed input request: embedding_types entries must be one of %s, please reformat your input and try again.", strings.Join(brtCohereEmbeddingTypes, ", "))
	}
	inv.prompt = strings.Join(texts, "\n")
	return nil
}

func brtIn(v string, list []string) bool {
	for _, item := range list {
		if v == item {
			return true
		}
	}
	return false
}

func brtReason(g brtGeneration, end, max, stop string) string {
	switch g.reason {
	case "max":
		return max
	case "stop":
		return stop
	}
	return end
}

// brtRender builds the non-streaming InvokeModel response body.
func brtRender(inv *brtInvocation, g brtGeneration, inTok, outTok int) map[string]any {
	switch inv.model.family {
	case brtAnthropic:
		if inv.textMode {
			return map[string]any{"type": "completion", "completion": " " + g.text, "stop_reason": brtReason(g, "stop_sequence", "max_tokens", "stop_sequence"), "stop": brtNullable(g.stopSeq)}
		}
		content := []any{map[string]any{"type": "text", "text": g.text}}
		reason := brtReason(g, "end_turn", "max_tokens", "stop_sequence")
		if inv.tool != "" {
			content = []any{map[string]any{"type": "tool_use", "id": brtID("toolu_bdrk_", inv.prompt, inv.tool), "name": inv.tool, "input": map[string]any{}}}
			reason = "tool_use"
		}
		return map[string]any{"id": brtID("msg_bdrk_", inv.prompt), "type": "message", "role": "assistant", "model": inv.model.id,
			"content": content, "stop_reason": reason, "stop_sequence": brtNullable(g.stopSeq),
			"usage": map[string]any{"input_tokens": inTok, "output_tokens": outTok}}
	case brtTitanText:
		return map[string]any{"inputTextTokenCount": inTok, "results": []any{map[string]any{"tokenCount": outTok, "outputText": g.text,
			"completionReason": brtReason(g, "FINISH", "LENGTH", "STOP_CRITERIA_MET")}}}
	case brtTitanEmbed:
		vec := brtEmbedding(inv.prompt, inv.model.dims)
		out := map[string]any{"embedding": vec, "inputTextTokenCount": inTok}
		if types, _ := brtStrings(inv.body["embeddingTypes"]); len(types) > 0 {
			byType := map[string]any{}
			for _, t := range types {
				switch t {
				case "float":
					byType[t] = vec
				case "binary":
					bits := make([]int, len(vec))
					for i, v := range vec {
						if v > 0 {
							bits[i] = 1
						}
					}
					byType[t] = bits
				}
			}
			out["embeddingsByType"] = byType
		}
		return out
	case brtNova:
		return map[string]any{"output": map[string]any{"message": map[string]any{"role": "assistant", "content": []any{map[string]any{"text": g.text}}}},
			"stopReason": brtReason(g, "end_turn", "max_tokens", "stop_sequence"),
			"usage":      map[string]any{"inputTokens": inTok, "outputTokens": outTok, "totalTokens": inTok + outTok}}
	case brtLlama:
		return map[string]any{"generation": g.text, "prompt_token_count": inTok, "generation_token_count": outTok, "stop_reason": brtReason(g, "stop", "length", "stop")}
	case brtMistral:
		return map[string]any{"outputs": []any{map[string]any{"text": g.text, "stop_reason": brtReason(g, "stop", "length", "stop")}}}
	case brtCohereChat:
		return map[string]any{"response_id": brtID("", inv.prompt), "generation_id": brtID("", inv.prompt, "gen"), "text": g.text,
			"finish_reason": brtReason(g, "COMPLETE", "MAX_TOKENS", "COMPLETE"),
			"chat_history":  []any{map[string]any{"role": "USER", "message": inv.last}, map[string]any{"role": "CHATBOT", "message": g.text}}}
	case brtCohereEmbed:
		texts, _ := brtStrings(inv.body["texts"])
		vectors := make([][]float64, len(texts))
		for i, t := range texts {
			vectors[i] = brtEmbedding(t, inv.model.dims)
		}
		out := map[string]any{"id": brtID("", inv.prompt), "texts": texts, "response_type": "embeddings_floats", "embeddings": vectors}
		if types, _ := brtStrings(inv.body["embedding_types"]); len(types) > 0 {
			byType := map[string]any{}
			for _, t := range types {
				list := make([]any, len(vectors))
				for i, v := range vectors {
					list[i] = brtQuantize(v, t)
				}
				byType[t] = list
			}
			out["response_type"], out["embeddings"] = "embeddings_by_type", byType
		}
		return out
	}
	return map[string]any{}
}

// brtRenderChunks builds the InvokeModelWithResponseStream chunk bodies.
func brtRenderChunks(inv *brtInvocation, g brtGeneration, inTok, outTok int) []map[string]any {
	deltas := brtChunks(g.text)
	var out []map[string]any
	switch inv.model.family {
	case brtAnthropic:
		if inv.textMode {
			for _, d := range deltas {
				out = append(out, map[string]any{"type": "completion", "completion": d, "stop_reason": nil, "stop": nil})
			}
			return append(out, map[string]any{"type": "completion", "completion": "", "stop_reason": brtReason(g, "stop_sequence", "max_tokens", "stop_sequence"), "stop": brtNullable(g.stopSeq)})
		}
		msg := brtRender(inv, g, inTok, 1)
		msg["content"], msg["stop_reason"], msg["stop_sequence"] = []any{}, nil, nil
		out = append(out, map[string]any{"type": "message_start", "message": msg},
			map[string]any{"type": "content_block_start", "index": 0, "content_block": map[string]any{"type": "text", "text": ""}})
		for _, d := range deltas {
			out = append(out, map[string]any{"type": "content_block_delta", "index": 0, "delta": map[string]any{"type": "text_delta", "text": d}})
		}
		return append(out, map[string]any{"type": "content_block_stop", "index": 0},
			map[string]any{"type": "message_delta", "delta": map[string]any{"stop_reason": brtReason(g, "end_turn", "max_tokens", "stop_sequence"), "stop_sequence": brtNullable(g.stopSeq)}, "usage": map[string]any{"output_tokens": outTok}},
			map[string]any{"type": "message_stop"})
	case brtTitanText:
		for i, d := range deltas {
			chunk := map[string]any{"outputText": d, "index": 0, "totalOutputTextTokenCount": brtTokens(strings.Join(deltas[:i+1], "")), "completionReason": nil}
			if i == 0 {
				chunk["inputTextTokenCount"] = inTok
			}
			out = append(out, chunk)
		}
		return append(out, map[string]any{"outputText": "", "index": 0, "totalOutputTextTokenCount": outTok, "completionReason": brtReason(g, "FINISH", "LENGTH", "STOP_CRITERIA_MET")})
	case brtNova:
		out = append(out, map[string]any{"messageStart": map[string]any{"role": "assistant"}})
		for _, d := range deltas {
			out = append(out, map[string]any{"contentBlockDelta": map[string]any{"delta": map[string]any{"text": d}, "contentBlockIndex": 0}})
		}
		return append(out, map[string]any{"contentBlockStop": map[string]any{"contentBlockIndex": 0}},
			map[string]any{"messageStop": map[string]any{"stopReason": brtReason(g, "end_turn", "max_tokens", "stop_sequence")}},
			map[string]any{"metadata": map[string]any{"usage": map[string]any{"inputTokens": inTok, "outputTokens": outTok}, "metrics": map[string]any{}, "trace": map[string]any{}}})
	case brtLlama:
		for i, d := range deltas {
			chunk := map[string]any{"generation": d, "generation_token_count": brtTokens(strings.Join(deltas[:i+1], "")), "stop_reason": nil}
			if i == 0 {
				chunk["prompt_token_count"] = inTok
			}
			out = append(out, chunk)
		}
		return append(out, map[string]any{"generation": "", "generation_token_count": outTok, "stop_reason": brtReason(g, "stop", "length", "stop")})
	case brtMistral:
		for _, d := range deltas {
			out = append(out, map[string]any{"outputs": []any{map[string]any{"text": d, "stop_reason": nil}}})
		}
		return append(out, map[string]any{"outputs": []any{map[string]any{"text": "", "stop_reason": brtReason(g, "stop", "length", "stop")}}})
	case brtCohereChat:
		full := brtRender(inv, g, inTok, outTok)
		out = append(out, map[string]any{"is_finished": false, "event_type": "stream-start", "generation_id": full["generation_id"]})
		for _, d := range deltas {
			out = append(out, map[string]any{"is_finished": false, "event_type": "text-generation", "text": d})
		}
		return append(out, map[string]any{"is_finished": true, "event_type": "stream-end", "finish_reason": full["finish_reason"], "response": full})
	}
	return []map[string]any{{}}
}

func brtNullable(v string) any {
	if v == "" {
		return nil
	}
	return v
}

// brtQuantize converts a float embedding to one of the Cohere/Titan
// embedding types; binary types pack eight sign bits per byte.
func brtQuantize(vec []float64, typ string) any {
	switch typ {
	case "int8":
		out := make([]int, len(vec))
		for i, v := range vec {
			out[i] = int(math.Round(v * 127))
		}
		return out
	case "uint8":
		out := make([]int, len(vec))
		for i, v := range vec {
			out[i] = int(math.Round((v + 1) * 127.5))
		}
		return out
	case "binary", "ubinary":
		out := make([]int, (len(vec)+7)/8)
		for i, v := range vec {
			if v > 0 {
				out[i/8] |= 1 << (7 - i%8)
			}
		}
		if typ == "binary" {
			for i := range out {
				out[i] -= 128
			}
		}
		return out
	}
	return vec
}
