package server

import (
	"net/http"
	"regexp"
	"strings"
	"unicode/utf8"
)

// ponytail: built-in detectors for a subset of PII types; other types are
// accepted but never detected.
var brtPIIPatterns = map[string]*regexp.Regexp{
	"EMAIL":                     regexp.MustCompile(`[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}`),
	"PHONE":                     regexp.MustCompile(`\+?\d{1,2}[ .-]?\(?\d{3}\)?[ .-]?\d{3}[ .-]?\d{4}\b`),
	"US_SOCIAL_SECURITY_NUMBER": regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`),
	"CREDIT_DEBIT_CARD_NUMBER":  regexp.MustCompile(`\b(?:\d[ -]?){13,16}\b`),
	"IP_ADDRESS":                regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}\b`),
	"URL":                       regexp.MustCompile(`https?://[^\s]+`),
	"AWS_ACCESS_KEY":            regexp.MustCompile(`\b(?:AKIA|ASIA)[A-Z0-9]{16}\b`),
}

// brtGuardrailResult is one guardrail evaluation.
type brtGuardrailResult struct {
	intervened bool
	outputs    []string
	assessment map[string]any
	usage      map[string]any
	guarded    int
}

func brtActionFor(entry map[string]any, source, fallback string) string {
	key := "inputAction"
	if source == "OUTPUT" {
		key = "outputAction"
	}
	if v := bedrockString(entry, key); v != "" {
		return v
	}
	if v := bedrockString(entry, "action"); v != "" {
		return v
	}
	return fallback
}

func brtPolicyList(cfg map[string]any, policy, list string) []map[string]any {
	p, _ := cfg[policy].(map[string]any)
	items, _ := p[list].([]any)
	var out []map[string]any
	for _, item := range items {
		if m, ok := item.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out
}

// brtEvaluateGuardrail applies a guardrail's denied topics, custom words, PII
// entities and regexes to the texts. Content filters, managed word lists and
// contextual grounding are not evaluated.
func brtEvaluateGuardrail(cfg map[string]any, texts []string, source string) brtGuardrailResult {
	joined := strings.Join(texts, "\n")
	lower := strings.ToLower(joined)
	blocked, masked := false, false
	assessment := map[string]any{}

	var topics []any
	for _, t := range brtPolicyList(cfg, "topicPolicyConfig", "topicsConfig") {
		name := bedrockString(t, "name")
		hit := name != "" && strings.Contains(lower, strings.ToLower(name))
		examples, _ := brtStrings(t["examples"])
		for _, ex := range examples {
			hit = hit || (ex != "" && strings.Contains(lower, strings.ToLower(ex)))
		}
		if hit {
			action := brtWordAction(brtActionFor(t, source, "BLOCK"))
			blocked = blocked || action == "BLOCKED"
			topics = append(topics, map[string]any{"name": name, "type": "DENY", "action": action, "detected": true})
		}
	}
	if topics != nil {
		assessment["topicPolicy"] = map[string]any{"topics": topics}
	}

	var words []any
	for _, wd := range brtPolicyList(cfg, "wordPolicyConfig", "wordsConfig") {
		text := bedrockString(wd, "text")
		if text != "" && strings.Contains(lower, strings.ToLower(text)) {
			action := brtWordAction(brtActionFor(wd, source, "BLOCK"))
			blocked = blocked || action == "BLOCKED"
			words = append(words, map[string]any{"match": text, "action": action, "detected": true})
		}
	}
	if words != nil {
		assessment["wordPolicy"] = map[string]any{"customWords": words, "managedWordLists": []any{}}
	}

	outputs := append([]string(nil), texts...)
	var pii, regexes []any
	apply := func(re *regexp.Regexp, label, action string, record func(match, act string)) {
		for _, match := range re.FindAllString(joined, -1) {
			act := brtPIIAction(action)
			record(match, act)
			switch act {
			case "BLOCKED":
				blocked = true
			case "ANONYMIZED":
				masked = true
				for i := range outputs {
					outputs[i] = strings.ReplaceAll(outputs[i], match, "{"+label+"}")
				}
			}
		}
	}
	for _, e := range brtPolicyList(cfg, "sensitiveInformationPolicyConfig", "piiEntitiesConfig") {
		typ := bedrockString(e, "type")
		if re, ok := brtPIIPatterns[typ]; ok {
			apply(re, typ, brtActionFor(e, source, "BLOCK"), func(match, act string) {
				pii = append(pii, map[string]any{"match": match, "type": typ, "action": act, "detected": true})
			})
		}
	}
	for _, e := range brtPolicyList(cfg, "sensitiveInformationPolicyConfig", "regexesConfig") {
		re, err := regexp.Compile(bedrockString(e, "pattern"))
		if err != nil {
			continue
		}
		name := bedrockString(e, "name")
		apply(re, name, brtActionFor(e, source, "BLOCK"), func(match, act string) {
			regexes = append(regexes, map[string]any{"name": name, "match": match, "regex": re.String(), "action": act, "detected": true})
		})
	}
	if pii != nil || regexes != nil {
		if pii == nil {
			pii = []any{}
		}
		if regexes == nil {
			regexes = []any{}
		}
		assessment["sensitiveInformationPolicy"] = map[string]any{"piiEntities": pii, "regexes": regexes}
	}

	res := brtGuardrailResult{assessment: assessment, guarded: utf8.RuneCountInString(joined), outputs: []string{}}
	switch {
	case blocked:
		res.intervened = true
		msgKey := "blockedInputMessaging"
		if source == "OUTPUT" {
			msgKey = "blockedOutputsMessaging"
		}
		msg := bedrockString(cfg, msgKey)
		if msg == "" {
			msg = "Sorry, the model cannot answer this question."
		}
		res.outputs = []string{msg}
	case masked:
		res.intervened = true
		res.outputs = outputs
	}
	units := (res.guarded + 999) / 1000
	policyUnits := func(policy string) int {
		if _, ok := cfg[policy]; ok {
			return units
		}
		return 0
	}
	res.usage = map[string]any{
		"topicPolicyUnits":                    policyUnits("topicPolicyConfig"),
		"contentPolicyUnits":                  policyUnits("contentPolicyConfig"),
		"wordPolicyUnits":                     policyUnits("wordPolicyConfig"),
		"sensitiveInformationPolicyUnits":     policyUnits("sensitiveInformationPolicyConfig"),
		"sensitiveInformationPolicyFreeUnits": 0,
		"contextualGroundingPolicyUnits":      policyUnits("contextualGroundingPolicyConfig"),
	}
	return res
}

func brtWordAction(action string) string {
	if strings.EqualFold(action, "NONE") {
		return "NONE"
	}
	return "BLOCKED"
}

func brtPIIAction(action string) string {
	switch strings.ToUpper(action) {
	case "NONE":
		return "NONE"
	case "ANONYMIZE":
		return "ANONYMIZED"
	}
	return "BLOCKED"
}

// brtGuardrail looks up a guardrail version for a runtime call.
func (s *Server) brtGuardrail(identifier, version string) (map[string]any, *brtError) {
	if strings.TrimSpace(identifier) == "" || strings.TrimSpace(version) == "" {
		return nil, brtValidation("Guardrail identifier and version must both be provided.")
	}
	cfg, ok := s.bedrock.guardrailPolicy(identifier, version)
	if !ok {
		return nil, brtNotFound("The guardrail identifier or version provided in the request does not exist.")
	}
	return cfg, nil
}

func (s *Server) brtApplyGuardrail(w http.ResponseWriter, req *brtRequest) *brtError {
	payload, e := brtDecodeObject(req.body, "body")
	if e != nil {
		return e
	}
	source := bedrockString(payload, "source")
	if source != "INPUT" && source != "OUTPUT" {
		return brtValidation("1 validation error detected: Value '%s' at 'source' failed to satisfy constraint: Member must satisfy enum value set: [INPUT, OUTPUT]", source)
	}
	scope := bedrockString(payload, "outputScope")
	if scope != "" && scope != "INTERVENTIONS" && scope != "FULL" {
		return brtValidation("1 validation error detected: Value '%s' at 'outputScope' failed to satisfy constraint: Member must satisfy enum value set: [INTERVENTIONS, FULL]", scope)
	}
	content, _ := payload["content"].([]any)
	if len(content) == 0 {
		return brtValidation("1 validation error detected: Value at 'content' failed to satisfy constraint: Member must not be null")
	}
	var texts []string
	for i, block := range content {
		b, _ := block.(map[string]any)
		text, _ := b["text"].(map[string]any)
		image, hasImage := b["image"]
		switch {
		case text != nil:
			t, ok := text["text"].(string)
			if !ok {
				return brtValidation("content.%d.text.text is required.", i)
			}
			texts = append(texts, t)
		case hasImage && image != nil:
			// ponytail: image content is accepted but not evaluated.
		default:
			return brtValidation("content.%d must contain exactly one of text or image.", i)
		}
	}
	cfg, e := s.brtGuardrail(req.params["guardrailIdentifier"], req.params["guardrailVersion"])
	if e != nil {
		return e
	}
	res := brtEvaluateGuardrail(cfg, texts, source)
	action := "NONE"
	if res.intervened {
		action = "GUARDRAIL_INTERVENED"
	}
	outputs := []any{}
	for _, t := range res.outputs {
		outputs = append(outputs, map[string]any{"text": t})
	}
	if scope == "FULL" && !res.intervened {
		for _, t := range texts {
			outputs = append(outputs, map[string]any{"text": t})
		}
	}
	coverage := map[string]any{"textCharacters": map[string]any{"guarded": res.guarded, "total": res.guarded}}
	assessment := res.assessment
	assessment["invocationMetrics"] = map[string]any{"guardrailProcessingLatency": 1, "usage": res.usage, "guardrailCoverage": coverage}
	respondBedrockRuntimeJSON(w, http.StatusOK, map[string]any{
		"usage":             res.usage,
		"action":            action,
		"outputs":           outputs,
		"assessments":       []any{assessment},
		"guardrailCoverage": coverage,
	})
	return nil
}

// Keyword cues for InvokeGuardrailChecks scores; ponytail: a word list, not a
// classifier.
var brtCheckCues = map[string][]string{
	"VIOLENCE":         {"kill", "attack", "weapon", "bomb"},
	"HATE":             {"hate"},
	"SEXUAL":           {"sexual", "explicit"},
	"MISCONDUCT":       {"steal", "illegal", "hack"},
	"INSULTS":          {"stupid", "idiot"},
	"JAILBREAK":        {"jailbreak", "ignore previous instructions", "do anything now"},
	"PROMPT_INJECTION": {"ignore all previous", "disregard the above"},
	"PROMPT_LEAKAGE":   {"system prompt", "your instructions"},
}

func brtCategories(checks map[string]any, check, key, item string) ([]string, bool) {
	c, ok := checks[check].(map[string]any)
	if !ok {
		return nil, false
	}
	list, _ := c[key].([]any)
	var out []string
	for _, entry := range list {
		if m, ok := entry.(map[string]any); ok {
			out = append(out, bedrockString(m, item))
		}
	}
	return out, true
}

func brtGuardrailChecks(w http.ResponseWriter, req *brtRequest) *brtError {
	payload, e := brtDecodeObject(req.body, "body")
	if e != nil {
		return e
	}
	messages, _ := payload["messages"].([]any)
	if len(messages) == 0 {
		return brtValidation("1 validation error detected: Value at 'messages' failed to satisfy constraint: Member must have length greater than or equal to 1")
	}
	type text struct {
		msg, block int
		value      string
	}
	var texts []text
	for i, raw := range messages {
		m, _ := raw.(map[string]any)
		switch role := bedrockString(m, "role"); role {
		case "user", "assistant", "system":
		default:
			return brtValidation("1 validation error detected: Value '%s' at 'messages.%d.role' failed to satisfy constraint: Member must satisfy enum value set: [user, assistant, system]", role, i)
		}
		content, _ := m["content"].([]any)
		if len(content) == 0 {
			return brtValidation("1 validation error detected: Value at 'messages.%d.content' failed to satisfy constraint: Member must have length greater than or equal to 1", i)
		}
		for j, block := range content {
			b, _ := block.(map[string]any)
			v, ok := b["text"].(string)
			if !ok {
				return brtValidation("messages.%d.content.%d.text is required.", i, j)
			}
			texts = append(texts, text{i, j, v})
		}
	}
	checks, _ := payload["checks"].(map[string]any)
	all := ""
	chars := 0
	for _, t := range texts {
		all += strings.ToLower(t.value) + "\n"
		chars += utf8.RuneCountInString(t.value)
	}
	units := map[string]any{"textUnits": (chars + 999) / 1000}
	results, usage := map[string]any{}, map[string]any{}
	score := func(category string) float64 {
		for _, cue := range brtCheckCues[category] {
			if strings.Contains(all, cue) {
				return 0.9
			}
		}
		return 0
	}
	for _, check := range []struct{ name, key, item string }{{"contentFilter", "categories", "category"}, {"promptAttack", "categories", "category"}} {
		cats, ok := brtCategories(checks, check.name, check.key, check.item)
		if !ok {
			continue
		}
		if len(cats) == 0 {
			return brtValidation("checks.%s.%s must contain at least one entry.", check.name, check.key)
		}
		var out []any
		for _, cat := range cats {
			if _, known := brtCheckCues[cat]; !known {
				return brtValidation("Value '%s' at 'checks.%s.%s' is not a valid category.", cat, check.name, check.key)
			}
			out = append(out, map[string]any{"category": cat, "severityScore": score(cat)})
		}
		results[check.name] = map[string]any{"results": out}
		usage[check.name] = units
	}
	if types, ok := brtCategories(checks, "sensitiveInformation", "entities", "type"); ok {
		if len(types) == 0 {
			return brtValidation("checks.sensitiveInformation.entities must contain at least one entry.")
		}
		out := []any{}
		for _, typ := range types {
			re := brtPIIPatterns[typ]
			if re == nil {
				continue
			}
			for _, t := range texts {
				for _, loc := range re.FindAllStringIndex(t.value, -1) {
					out = append(out, map[string]any{
						"type": typ, "confidenceScore": 0.99,
						"beginOffset": utf8.RuneCountInString(t.value[:loc[0]]), "endOffset": utf8.RuneCountInString(t.value[:loc[1]]),
						"messageIndex": t.msg, "contentIndex": t.block,
					})
				}
			}
		}
		results["sensitiveInformation"] = map[string]any{"results": out, "truncated": false}
		usage["sensitiveInformation"] = units
	}
	if len(results) == 0 {
		return brtValidation("At least one of contentFilter, promptAttack or sensitiveInformation must be specified in checks.")
	}
	respondBedrockRuntimeJSON(w, http.StatusOK, map[string]any{"results": results, "usage": usage})
	return nil
}
