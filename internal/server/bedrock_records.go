package server

import (
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"
)

// handleRuntimeRecordLocked keeps the control-plane records the Bedrock
// Runtime depends on. It returns (nil, nil) for actions the generic
// fabricated handler should answer instead.
func (s *bedrockStore) handleRuntimeRecordLocked(action string, payload map[string]any) (map[string]any, *brtError) {
	now := time.Now().UTC().Format(time.RFC3339)
	switch action {
	case "CreateGuardrail":
		version, e := bedrockCompileGuardrail(payload, now)
		if e != nil {
			return nil, e
		}
		id := fmt.Sprintf("gr%010d", s.nextLocked())
		name := bedrockString(payload, "name")
		if name == "" {
			name = "stackyard"
		}
		g := &bedrockGuardrail{id: id, arn: bedrockARN("guardrail", id), name: name, created: now,
			versions: map[string]*bedrockGuardrailVersion{"DRAFT": version}, next: 1}
		s.guardrails[id] = g
		return map[string]any{"guardrailId": id, "guardrailArn": g.arn, "version": "DRAFT", "createdAt": now}, nil
	case "UpdateGuardrail", "CreateGuardrailVersion", "DeleteGuardrail", "GetGuardrail":
		g, e := s.requireGuardrailLocked(payload)
		if e != nil {
			return nil, e
		}
		switch action {
		case "UpdateGuardrail":
			version, e := bedrockCompileGuardrail(payload, now)
			if e != nil {
				return nil, e
			}
			g.versions["DRAFT"] = version
			if name := bedrockString(payload, "name"); name != "" {
				g.name = name
			}
			return map[string]any{"guardrailId": g.id, "guardrailArn": g.arn, "version": "DRAFT", "updatedAt": now}, nil
		case "CreateGuardrailVersion":
			version := fmt.Sprintf("%d", g.next)
			g.next++
			draft := g.versions["DRAFT"]
			g.versions[version] = &bedrockGuardrailVersion{cfg: bedrockCloneMap(draft.cfg), regexes: draft.regexes, updated: now}
			return map[string]any{"guardrailId": g.id, "version": version}, nil
		case "DeleteGuardrail":
			if version := bedrockString(payload, "guardrailVersion"); version != "" && version != "DRAFT" {
				if _, ok := g.versions[version]; !ok {
					return nil, brtNotFound("The guardrail version %s doesn't exist.", version)
				}
				delete(g.versions, version)
			} else {
				delete(s.guardrails, g.id)
			}
			return map[string]any{}, nil
		default:
			version := bedrockString(payload, "guardrailVersion")
			if version == "" {
				version = "DRAFT"
			}
			v, ok := g.versions[version]
			if !ok {
				return nil, brtNotFound("The guardrail version %s doesn't exist.", version)
			}
			return g.describe(version, v), nil
		}
	case "ListGuardrails":
		var out []any
		if bedrockString(payload, "guardrailIdentifier") != "" {
			g, e := s.requireGuardrailLocked(payload)
			if e != nil {
				return nil, e
			}
			for _, version := range g.sortedVersions() {
				out = append(out, g.summary(version, g.versions[version]))
			}
		} else {
			ids := make([]string, 0, len(s.guardrails))
			for id := range s.guardrails {
				ids = append(ids, id)
			}
			sort.Strings(ids)
			for _, id := range ids {
				g := s.guardrails[id]
				out = append(out, g.summary("DRAFT", g.versions["DRAFT"]))
			}
		}
		// ponytail: one page; maxResults/nextToken are ignored.
		if out == nil {
			out = []any{}
		}
		return map[string]any{"guardrails": out}, nil
	case "CreateProvisionedModelThroughput":
		arn := bedrockARN("provisioned-model", fmt.Sprintf("pm%010d", s.nextLocked()))
		s.models[arn] = bedrockModelIDFromARN(bedrockString(payload, "modelId"))
		return map[string]any{"provisionedModelArn": arn}, nil
	case "CreateInferenceProfile":
		source, _ := payload["modelSource"].(map[string]any)
		arn := bedrockARN("application-inference-profile", fmt.Sprintf("aip%010d", s.nextLocked()))
		s.models[arn] = bedrockModelIDFromARN(bedrockString(source, "copyFrom"))
		return map[string]any{"inferenceProfileArn": arn, "status": "ACTIVE"}, nil
	}
	return nil, nil
}

// bedrockCompileGuardrail validates a guardrail configuration's regexes and
// compiles them once.
func bedrockCompileGuardrail(cfg map[string]any, now string) (*bedrockGuardrailVersion, *brtError) {
	v := &bedrockGuardrailVersion{cfg: bedrockCloneMap(cfg), updated: now}
	for i, entry := range brtPolicyList(cfg, "sensitiveInformationPolicyConfig", "regexesConfig") {
		pattern := bedrockString(entry, "pattern")
		re, err := regexp.Compile(pattern)
		if pattern == "" || err != nil {
			return nil, brtValidation("sensitiveInformationPolicyConfig.regexesConfig.%d.pattern is not a valid regular expression.", i)
		}
		if re.MatchString("") {
			return nil, brtValidation("sensitiveInformationPolicyConfig.regexesConfig.%d.pattern must not match an empty string.", i)
		}
		v.regexes = append(v.regexes, re)
	}
	return v, nil
}

func (s *bedrockStore) requireGuardrailLocked(payload map[string]any) (*bedrockGuardrail, *brtError) {
	identifier := bedrockString(payload, "guardrailIdentifier")
	if identifier == "" {
		return nil, brtValidation("guardrailIdentifier is required.")
	}
	g := s.guardrailLocked(identifier)
	if g == nil {
		return nil, &brtError{http.StatusNotFound, "ResourceNotFoundException", fmt.Sprintf("The guardrail %s doesn't exist.", identifier)}
	}
	return g, nil
}

func (s *bedrockStore) guardrailLocked(identifier string) *bedrockGuardrail {
	if g, ok := s.guardrails[identifier]; ok {
		return g
	}
	for _, g := range s.guardrails {
		if g.arn == identifier {
			return g
		}
	}
	return nil
}

func (g *bedrockGuardrail) sortedVersions() []string {
	out := make([]string, 0, len(g.versions))
	for v := range g.versions {
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i] == "DRAFT" || out[j] == "DRAFT" {
			return out[i] == "DRAFT"
		}
		if len(out[i]) != len(out[j]) {
			return len(out[i]) < len(out[j])
		}
		return out[i] < out[j]
	})
	return out
}

func (g *bedrockGuardrail) summary(version string, v *bedrockGuardrailVersion) map[string]any {
	return map[string]any{"id": g.id, "arn": g.arn, "name": g.name, "status": "READY", "version": version, "createdAt": g.created, "updatedAt": v.updated}
}

// describe renders GetGuardrail: the stored *Config policy lists under their
// Get names.
func (g *bedrockGuardrail) describe(version string, v *bedrockGuardrailVersion) map[string]any {
	out := map[string]any{
		"name": g.name, "guardrailId": g.id, "guardrailArn": g.arn, "version": version, "status": "READY",
		"createdAt": g.created, "updatedAt": v.updated,
		"blockedInputMessaging":   bedrockString(v.cfg, "blockedInputMessaging"),
		"blockedOutputsMessaging": bedrockString(v.cfg, "blockedOutputsMessaging"),
	}
	for _, p := range []struct{ policy, list, outPolicy, outList string }{
		{"topicPolicyConfig", "topicsConfig", "topicPolicy", "topics"},
		{"contentPolicyConfig", "filtersConfig", "contentPolicy", "filters"},
		{"wordPolicyConfig", "wordsConfig", "wordPolicy", "words"},
		{"wordPolicyConfig", "managedWordListsConfig", "wordPolicy", "managedWordLists"},
		{"sensitiveInformationPolicyConfig", "piiEntitiesConfig", "sensitiveInformationPolicy", "piiEntities"},
		{"sensitiveInformationPolicyConfig", "regexesConfig", "sensitiveInformationPolicy", "regexes"},
	} {
		items := brtPolicyList(v.cfg, p.policy, p.list)
		if items == nil {
			continue
		}
		section, _ := out[p.outPolicy].(map[string]any)
		if section == nil {
			section = map[string]any{}
			out[p.outPolicy] = section
		}
		list := make([]any, len(items))
		for i, item := range items {
			list[i] = bedrockCloneMap(item)
		}
		section[p.outList] = list
	}
	if d := bedrockString(v.cfg, "description"); d != "" {
		out["description"] = d
	}
	return out
}

// guardrailPolicy returns a guardrail version's configuration (a copy) and
// its compiled regexes.
func (s *bedrockStore) guardrailPolicy(identifier, version string) (map[string]any, []*regexp.Regexp, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	g := s.guardrailLocked(strings.TrimSpace(identifier))
	if g == nil {
		return nil, nil, false
	}
	v, ok := g.versions[strings.TrimSpace(version)]
	if !ok {
		return nil, nil, false
	}
	return bedrockCloneMap(v.cfg), v.regexes, true
}

// runtimeBaseModel returns the model behind a provisioned throughput or
// application inference profile ARN.
func (s *bedrockStore) runtimeBaseModel(arn string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id, ok := s.models[arn]
	return id, ok
}

// bedrockModelIDFromARN turns a foundation-model or inference-profile ARN into
// its model ID; anything else is returned unchanged.
func bedrockModelIDFromARN(value string) string {
	value = strings.TrimSpace(value)
	for _, marker := range []string{":foundation-model/", ":inference-profile/"} {
		if i := strings.Index(value, marker); i >= 0 {
			return value[i+len(marker):]
		}
	}
	return value
}

func bedrockString(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	if v, ok := m[key]; ok && v != nil {
		return strings.TrimSpace(fmt.Sprint(v))
	}
	return ""
}

func bedrockCloneMap(m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = bedrockCloneValue(v)
	}
	return out
}

func bedrockCloneValue(v any) any {
	switch t := v.(type) {
	case map[string]any:
		return bedrockCloneMap(t)
	case []any:
		out := make([]any, len(t))
		for i, item := range t {
			out[i] = bedrockCloneValue(item)
		}
		return out
	default:
		return v
	}
}
