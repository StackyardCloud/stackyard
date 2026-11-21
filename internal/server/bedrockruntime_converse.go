package server

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"
)

// brtConversation is a validated Converse / CountTokens conversation.
type brtConversation struct {
	prompt     string   // every text, for token counts
	last       string   // last user text, which the reply echoes
	guard      []string // guardContent texts; when empty the whole prompt is guarded
	lastRole   string
	toolResult bool // the last user turn returned tool results
	tools      []string
	choice     string
	choiceTool string
	maxTokens  int
	stops      []string
}

var (
	brtContentKeys    = []string{"text", "image", "document", "video", "audio", "toolUse", "toolResult", "guardContent", "cachePoint", "reasoningContent", "citationsContent", "searchResult", "toolAddition", "toolRemoval"}
	brtToolResultKeys = []string{"json", "text", "image", "document", "video", "searchResult"}
	brtToolNameRE     = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,64}$`)
	brtToolUseIDRE    = regexp.MustCompile(`^[a-zA-Z0-9_.:-]{1,64}$`)
	brtImageFormats   = []string{"png", "jpeg", "gif", "webp"}
	brtDocFormats     = []string{"pdf", "csv", "doc", "docx", "xls", "xlsx", "html", "txt", "md"}
)

func brtConstraint(value any, path, constraint string) *brtError {
	if value == nil {
		return brtValidation("1 validation error detected: Value at '%s' failed to satisfy constraint: %s", path, constraint)
	}
	return brtValidation("1 validation error detected: Value '%v' at '%s' failed to satisfy constraint: %s", value, path, constraint)
}

// brtOneOf returns the single member set in a union value.
func brtOneOf(raw any, keys []string, path string) (string, map[string]any, *brtError) {
	m, ok := raw.(map[string]any)
	if !ok {
		return "", nil, brtValidation("%s must be an object.", path)
	}
	found := ""
	for _, k := range keys {
		if v, ok := m[k]; ok && v != nil {
			if found != "" {
				return "", nil, brtValidation("%s must set exactly one member, but both %s and %s are set.", path, found, k)
			}
			found = k
		}
	}
	if found == "" {
		return "", nil, brtValidation("%s must set exactly one of %s.", path, strings.Join(keys, ", "))
	}
	return found, m, nil
}

func brtParseConversation(p map[string]any, inference bool) (*brtConversation, *brtError) {
	c := &brtConversation{maxTokens: 512}
	var texts []string
	if raw, ok := p["system"]; ok && raw != nil {
		list, ok := raw.([]any)
		if !ok {
			return nil, brtValidation("system must be a list of system content blocks.")
		}
		for i, block := range list {
			key, m, e := brtOneOf(block, []string{"text", "guardContent", "cachePoint"}, fmt.Sprintf("system.%d", i))
			if e != nil {
				return nil, e
			}
			switch key {
			case "text":
				texts = append(texts, bedrockString(m, "text"))
			case "guardContent":
				t := brtGuardText(m["guardContent"])
				c.guard, texts = append(c.guard, t), append(texts, t)
			}
		}
	}
	messages, _ := p["messages"].([]any)
	if len(messages) == 0 {
		return nil, brtValidation("A conversation must start with a user message. Try again with a conversation that starts with a user message.")
	}
	usesTools := false
	pending := map[string]bool{} // toolUse IDs the next user turn must answer
	for i, raw := range messages {
		msg, _ := raw.(map[string]any)
		role := bedrockString(msg, "role")
		switch {
		case role != "user" && role != "assistant":
			return nil, brtConstraint(role, fmt.Sprintf("messages.%d.role", i), "Member must satisfy enum value set: [user, assistant]")
		case i == 0 && role != "user":
			return nil, brtValidation("A conversation must start with a user message. Try again with a conversation that starts with a user message.")
		case i > 0 && role == c.lastRole:
			return nil, brtValidation("A conversation must alternate between user and assistant roles. Make sure the conversation alternates between user and assistant roles and try again.")
		}
		content, _ := msg["content"].([]any)
		if len(content) == 0 {
			return nil, brtConstraint(nil, fmt.Sprintf("messages.%d.content", i), "Member must have length greater than or equal to 1")
		}
		uses, answered := map[string]bool{}, map[string]bool{}
		for j, block := range content {
			path := fmt.Sprintf("messages.%d.content.%d", i, j)
			key, m, e := brtOneOf(block, brtContentKeys, path)
			if e != nil {
				return nil, e
			}
			switch key {
			case "text":
				t, ok := m["text"].(string)
				if !ok || strings.TrimSpace(t) == "" {
					return nil, brtValidation("The text field in the ContentBlock object at %s is blank. Add text to the text field, and try again.", path)
				}
				texts = append(texts, t)
				if role == "user" {
					c.last = t
				}
			case "image", "document":
				if e := brtCheckMedia(key, m[key], path); e != nil {
					return nil, e
				}
			case "guardContent":
				t := brtGuardText(m["guardContent"])
				c.guard, texts = append(c.guard, t), append(texts, t)
				if role == "user" {
					c.last = t
				}
			case "toolUse":
				usesTools = true
				id, e := brtCheckToolUse(m["toolUse"], role, path)
				if e != nil {
					return nil, e
				}
				uses[id] = true
			case "toolResult":
				usesTools = true
				id, text, e := brtCheckToolResult(m["toolResult"], role, path)
				if e != nil {
					return nil, e
				}
				if !pending[id] {
					return nil, brtValidation("The toolUseId %s at %s.toolResult doesn't match any toolUse block of the previous turn.", id, path)
				}
				answered[id] = true
				texts = append(texts, text)
			}
		}
		if role == "user" {
			var missing []string
			for id := range pending {
				if !answered[id] {
					missing = append(missing, id)
				}
			}
			if len(missing) > 0 {
				sort.Strings(missing)
				return nil, brtValidation("Expected toolResult blocks at messages.%d.content for the following Ids: %s", i, strings.Join(missing, ", "))
			}
			c.toolResult = len(answered) > 0
		}
		pending, c.lastRole = uses, role
	}
	if e := brtParseToolConfig(c, p, usesTools); e != nil {
		return nil, e
	}
	if inference {
		if e := brtParseInference(c, p); e != nil {
			return nil, e
		}
	}
	c.prompt = strings.Join(texts, "\n")
	return c, nil
}

func brtGuardText(raw any) string {
	g, _ := raw.(map[string]any)
	t, _ := g["text"].(map[string]any)
	return bedrockString(t, "text")
}

func brtCheckMedia(kind string, raw any, path string) *brtError {
	m, _ := raw.(map[string]any)
	formats := brtImageFormats
	if kind == "document" {
		formats = brtDocFormats
		if bedrockString(m, "name") == "" {
			return brtConstraint(nil, path+".document.name", "Member must not be null")
		}
	}
	if f := bedrockString(m, "format"); !brtIn(f, formats) {
		return brtConstraint(f, path+"."+kind+".format", "Member must satisfy enum value set: ["+strings.Join(formats, ", ")+"]")
	}
	if src, _ := m["source"].(map[string]any); len(src) == 0 {
		return brtConstraint(nil, path+"."+kind+".source", "Member must not be null")
	}
	return nil
}

func brtCheckToolUse(raw any, role, path string) (string, *brtError) {
	if role != "assistant" {
		return "", brtValidation("toolUse content blocks are only allowed in assistant messages (%s).", path)
	}
	m, _ := raw.(map[string]any)
	id, name := bedrockString(m, "toolUseId"), bedrockString(m, "name")
	if !brtToolUseIDRE.MatchString(id) {
		return "", brtConstraint(id, path+".toolUse.toolUseId", "Member must satisfy regular expression pattern: [a-zA-Z0-9_.:-]+")
	}
	if !brtToolNameRE.MatchString(name) {
		return "", brtConstraint(name, path+".toolUse.name", "Member must satisfy regular expression pattern: [a-zA-Z0-9_-]+")
	}
	if _, ok := m["input"]; !ok {
		return "", brtConstraint(nil, path+".toolUse.input", "Member must not be null")
	}
	return id, nil
}

func brtCheckToolResult(raw any, role, path string) (string, string, *brtError) {
	if role != "user" {
		return "", "", brtValidation("toolResult content blocks are only allowed in user messages (%s).", path)
	}
	m, _ := raw.(map[string]any)
	id := bedrockString(m, "toolUseId")
	if !brtToolUseIDRE.MatchString(id) {
		return "", "", brtConstraint(id, path+".toolResult.toolUseId", "Member must satisfy regular expression pattern: [a-zA-Z0-9_.:-]+")
	}
	if s := bedrockString(m, "status"); s != "" && s != "success" && s != "error" {
		return "", "", brtConstraint(s, path+".toolResult.status", "Member must satisfy enum value set: [success, error]")
	}
	content, _ := m["content"].([]any)
	if len(content) == 0 {
		return "", "", brtConstraint(nil, path+".toolResult.content", "Member must not be null")
	}
	var texts []string
	for k, item := range content {
		key, im, e := brtOneOf(item, brtToolResultKeys, fmt.Sprintf("%s.toolResult.content.%d", path, k))
		if e != nil {
			return "", "", e
		}
		if key == "text" {
			texts = append(texts, bedrockString(im, "text"))
		}
	}
	return id, strings.Join(texts, " "), nil
}

func brtParseToolConfig(c *brtConversation, p map[string]any, usesTools bool) *brtError {
	cfg, ok := p["toolConfig"].(map[string]any)
	if !ok {
		if usesTools {
			return brtValidation("The toolConfig field must be defined when using toolUse and toolResult content blocks.")
		}
		return nil
	}
	tools, _ := cfg["tools"].([]any)
	if len(tools) == 0 {
		return brtConstraint(nil, "toolConfig.tools", "Member must have length greater than or equal to 1")
	}
	seen := map[string]bool{}
	for i, raw := range tools {
		key, m, e := brtOneOf(raw, []string{"toolSpec", "systemTool", "cachePoint"}, fmt.Sprintf("toolConfig.tools.%d", i))
		if e != nil {
			return e
		}
		if key != "toolSpec" {
			continue
		}
		spec, _ := m["toolSpec"].(map[string]any)
		name := bedrockString(spec, "name")
		if !brtToolNameRE.MatchString(name) {
			return brtConstraint(name, fmt.Sprintf("toolConfig.tools.%d.toolSpec.name", i), "Member must satisfy regular expression pattern: [a-zA-Z0-9_-]+")
		}
		if seen[name] {
			return brtValidation("The toolConfig.tools list contains the tool name %s more than once. Tool names must be unique.", name)
		}
		seen[name] = true
		if _, _, e := brtOneOf(spec["inputSchema"], []string{"json"}, fmt.Sprintf("toolConfig.tools.%d.toolSpec.inputSchema", i)); e != nil {
			return e
		}
		c.tools = append(c.tools, name)
	}
	if raw, ok := cfg["toolChoice"]; ok && raw != nil {
		key, m, e := brtOneOf(raw, []string{"auto", "any", "tool"}, "toolConfig.toolChoice")
		if e != nil {
			return e
		}
		c.choice = key
		if key == "tool" {
			t, _ := m["tool"].(map[string]any)
			c.choiceTool = bedrockString(t, "name")
			if !seen[c.choiceTool] {
				return brtValidation("The tool %s specified in toolConfig.toolChoice isn't defined in toolConfig.tools.", c.choiceTool)
			}
		}
	}
	return nil
}

func brtParseInference(c *brtConversation, p map[string]any) *brtError {
	cfg, ok := p["inferenceConfig"].(map[string]any)
	if !ok {
		return nil
	}
	if v, set, _ := brtRange(cfg, "maxTokens", "", 1, 1e9, true); set {
		c.maxTokens = int(v)
	} else if raw, present := cfg["maxTokens"]; present {
		return brtConstraint(raw, "inferenceConfig.maxTokens", "Member must have value greater than or equal to 1")
	}
	for _, key := range []string{"temperature", "topP"} {
		if _, _, e := brtRange(cfg, key, "", 0, 1, false); e != nil {
			return brtConstraint(cfg[key], "inferenceConfig."+key, "Member must have value between 0 and 1")
		}
	}
	stops, ok := brtStrings(cfg["stopSequences"])
	if !ok || len(stops) > 2500 {
		return brtConstraint(nil, "inferenceConfig.stopSequences", "Member must be a list of at most 2500 strings")
	}
	for i, s := range stops {
		if s == "" {
			return brtConstraint(nil, fmt.Sprintf("inferenceConfig.stopSequences.%d", i), "Member must have length greater than or equal to 1")
		}
	}
	c.stops = stops
	return nil
}

// brtToolToCall picks the tool a reply calls: the forced tool, the first tool
// for toolChoice any, or (auto) a tool the user names.
func (c *brtConversation) brtToolToCall() string {
	if c.lastRole != "user" || len(c.tools) == 0 {
		return ""
	}
	switch c.choice {
	case "tool":
		return c.choiceTool
	case "any":
		return c.tools[0]
	}
	if c.toolResult {
		return ""
	}
	lower := strings.ToLower(c.last)
	for _, t := range c.tools {
		if strings.Contains(lower, strings.ToLower(t)) {
			return t
		}
	}
	return ""
}

func brtEnumField(m map[string]any, key, path string, allowed ...string) (string, *brtError) {
	v := bedrockString(m, key)
	if v != "" && !brtIn(v, allowed) {
		return "", brtConstraint(v, path, "Member must satisfy enum value set: ["+strings.Join(allowed, ", ")+"]")
	}
	return v, nil
}

func (s *Server) brtConverse(w http.ResponseWriter, req *brtRequest, stream bool) *brtError {
	start := time.Now()
	model, _, e := s.brtResolveModel(req.params["modelId"])
	if e != nil {
		return e
	}
	if !model.text() {
		return brtValidation("This action doesn't support the model that you provided. Try again with a supported text or chat model.")
	}
	p, e := brtDecodeObject(req.body, "body")
	if e != nil {
		return e
	}
	if _, ok := p["promptVariables"]; ok {
		return brtValidation("promptVariables can only be used when modelId is a prompt resource ARN.")
	}
	conv, e := brtParseConversation(p, true)
	if e != nil {
		return e
	}
	perf, _ := p["performanceConfig"].(map[string]any)
	latency, e := brtEnumField(perf, "latency", "performanceConfig.latency", "standard", "optimized")
	if e != nil {
		return e
	}
	tierCfg, _ := p["serviceTier"].(map[string]any)
	tier, e := brtEnumField(tierCfg, "type", "serviceTier.type", "priority", "default", "flex", "reserved")
	if e != nil {
		return e
	}
	paths, ok := brtStrings(p["additionalModelResponseFieldPaths"])
	if !ok || len(paths) > 10 {
		return brtConstraint(nil, "additionalModelResponseFieldPaths", "Member must be a list of at most 10 JSON pointers")
	}
	for _, path := range paths {
		if !strings.HasPrefix(path, "/") {
			return brtValidation("additionalModelResponseFieldPaths entry %q must be a JSON pointer starting with /.", path)
		}
	}
	var guard *brtGuardrailResult
	var trace map[string]any
	if gc, ok := p["guardrailConfig"].(map[string]any); ok {
		mode, e := brtEnumField(gc, "trace", "guardrailConfig.trace", "enabled", "disabled", "enabled_full")
		if e != nil {
			return e
		}
		if _, e := brtEnumField(gc, "streamProcessingMode", "guardrailConfig.streamProcessingMode", "sync", "async"); e != nil {
			return e
		}
		gid := bedrockString(gc, "guardrailIdentifier")
		cfg, e := s.brtGuardrail(gid, bedrockString(gc, "guardrailVersion"))
		if e != nil {
			return e
		}
		texts := conv.guard
		if len(texts) == 0 {
			texts = []string{conv.prompt}
		}
		res := brtEvaluateGuardrail(cfg, texts, "INPUT")
		guard = &res
		if mode == "enabled" || mode == "enabled_full" {
			trace = map[string]any{"guardrail": map[string]any{"inputAssessment": map[string]any{gid: res.assessment}}}
		}
	}

	var content map[string]any
	reason, stopSeq := "end_turn", ""
	tool := conv.brtToolToCall()
	switch {
	case guard != nil && guard.intervened:
		content, reason = map[string]any{"text": strings.Join(guard.outputs, "\n")}, "guardrail_intervened"
	case tool != "":
		content = map[string]any{"toolUse": map[string]any{"toolUseId": brtID("tooluse_", conv.prompt, tool), "name": tool, "input": map[string]any{}}}
		reason = "tool_use"
	default:
		g := brtGenerate(conv.last, conv.maxTokens, conv.stops)
		content, stopSeq = map[string]any{"text": g.text}, g.stopSeq
		reason = brtReason(g, "end_turn", "max_tokens", "stop_sequence")
	}
	outText := bedrockString(content, "text")
	inTok, outTok := brtTokens(conv.prompt), max(brtTokens(outText), 1)
	usage := map[string]any{"inputTokens": inTok, "outputTokens": outTok, "totalTokens": inTok + outTok}
	metrics := map[string]any{"latencyMs": time.Since(start).Milliseconds()}
	var extra map[string]any
	if len(paths) > 0 {
		extra = map[string]any{}
		for _, path := range paths {
			if path == "/stop_sequence" {
				extra["stop_sequence"] = brtNullable(stopSeq)
			}
		}
	}

	if !stream {
		out := map[string]any{"output": map[string]any{"message": map[string]any{"role": "assistant", "content": []any{content}}},
			"stopReason": reason, "usage": usage, "metrics": metrics}
		brtSetOptional(out, "additionalModelResponseFields", extra)
		brtSetOptional(out, "trace", trace)
		if latency != "" {
			out["performanceConfig"] = map[string]any{"latency": latency}
		}
		if tier != "" {
			out["serviceTier"] = map[string]any{"type": tier}
		}
		respondBedrockRuntimeJSON(w, http.StatusOK, out)
		return nil
	}
	st := brtStartStream(w)
	st.event("messageStart", map[string]any{"role": "assistant"})
	if use, ok := content["toolUse"].(map[string]any); ok {
		st.event("contentBlockStart", map[string]any{"contentBlockIndex": 0, "start": map[string]any{"toolUse": map[string]any{"toolUseId": use["toolUseId"], "name": use["name"]}}})
		st.event("contentBlockDelta", map[string]any{"contentBlockIndex": 0, "delta": map[string]any{"toolUse": map[string]any{"input": "{}"}}})
	} else {
		for _, d := range brtChunks(outText) {
			st.event("contentBlockDelta", map[string]any{"contentBlockIndex": 0, "delta": map[string]any{"text": d}})
		}
	}
	st.event("contentBlockStop", map[string]any{"contentBlockIndex": 0})
	stop := map[string]any{"stopReason": reason}
	brtSetOptional(stop, "additionalModelResponseFields", extra)
	st.event("messageStop", stop)
	meta := map[string]any{"usage": usage, "metrics": metrics}
	brtSetOptional(meta, "trace", trace)
	if latency != "" {
		meta["performanceConfig"] = map[string]any{"latency": latency}
	}
	st.event("metadata", meta)
	return nil
}

func brtSetOptional(out map[string]any, key string, value map[string]any) {
	if value != nil {
		out[key] = value
	}
}

func (s *Server) brtCountTokens(w http.ResponseWriter, req *brtRequest) *brtError {
	model, _, e := s.brtResolveModel(req.params["modelId"])
	if e != nil {
		return e
	}
	if !model.count {
		return brtValidation("The provided model doesn't support counting tokens.")
	}
	p, e := brtDecodeObject(req.body, "body")
	if e != nil {
		return e
	}
	key, input, e := brtOneOf(p["input"], []string{"invokeModel", "converse"}, "input")
	if e != nil {
		return e
	}
	tokens := 0
	if key == "invokeModel" {
		im, _ := input["invokeModel"].(map[string]any)
		raw, err := base64.StdEncoding.DecodeString(bedrockString(im, "body"))
		if err != nil || len(raw) == 0 {
			return brtValidation("input.invokeModel.body must be a base64-encoded JSON request body.")
		}
		body, e := brtDecodeObject(raw, "input.invokeModel.body")
		if e != nil {
			return e
		}
		inv, e := brtParseInvoke(model, body)
		if e != nil {
			return e
		}
		tokens = brtTokens(inv.prompt)
	} else {
		conv, _ := input["converse"].(map[string]any)
		c, e := brtParseConversation(conv, false)
		if e != nil {
			return e
		}
		tokens = brtTokens(c.prompt)
	}
	respondBedrockRuntimeJSON(w, http.StatusOK, map[string]any{"inputTokens": tokens})
	return nil
}
