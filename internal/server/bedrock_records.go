package server

import (
	"fmt"
	"strings"
	"time"
)

// handleRuntimeRecordLocked keeps the control-plane records the Bedrock
// Runtime depends on. It returns nil for actions (or unknown identifiers) the
// generic fabricated handler should answer instead.
func (s *bedrockStore) handleRuntimeRecordLocked(action string, payload map[string]any) map[string]any {
	switch action {
	case "CreateGuardrail":
		id := fmt.Sprintf("gr%010d", s.nextLocked())
		g := &bedrockGuardrail{
			id:       id,
			arn:      bedrockARN("guardrail", id),
			versions: map[string]map[string]any{"DRAFT": bedrockCloneMap(payload)},
			next:     1,
		}
		s.guardrails[id] = g
		return map[string]any{"guardrailId": id, "guardrailArn": g.arn, "version": "DRAFT", "createdAt": time.Now().UTC().Format(time.RFC3339)}
	case "UpdateGuardrail":
		g := s.guardrailLocked(bedrockString(payload, "guardrailIdentifier"))
		if g == nil {
			return nil
		}
		g.versions["DRAFT"] = bedrockCloneMap(payload)
		return map[string]any{"guardrailId": g.id, "guardrailArn": g.arn, "version": "DRAFT", "updatedAt": time.Now().UTC().Format(time.RFC3339)}
	case "CreateGuardrailVersion":
		g := s.guardrailLocked(bedrockString(payload, "guardrailIdentifier"))
		if g == nil {
			return nil
		}
		version := fmt.Sprintf("%d", g.next)
		g.next++
		g.versions[version] = bedrockCloneMap(g.versions["DRAFT"])
		return map[string]any{"guardrailId": g.id, "version": version}
	case "DeleteGuardrail":
		g := s.guardrailLocked(bedrockString(payload, "guardrailIdentifier"))
		if g == nil {
			return nil
		}
		if version := bedrockString(payload, "guardrailVersion"); version != "" && version != "DRAFT" {
			delete(g.versions, version)
		} else {
			delete(s.guardrails, g.id)
		}
		return map[string]any{}
	case "CreateProvisionedModelThroughput":
		arn := bedrockARN("provisioned-model", fmt.Sprintf("pm%010d", s.nextLocked()))
		s.models[arn] = bedrockModelIDFromARN(bedrockString(payload, "modelId"))
		return map[string]any{"provisionedModelArn": arn}
	case "CreateInferenceProfile":
		source, _ := payload["modelSource"].(map[string]any)
		arn := bedrockARN("application-inference-profile", fmt.Sprintf("aip%010d", s.nextLocked()))
		s.models[arn] = bedrockModelIDFromARN(bedrockString(source, "copyFrom"))
		return map[string]any{"inferenceProfileArn": arn, "status": "ACTIVE"}
	}
	return nil
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

// guardrailPolicy returns a copy of a guardrail version's configuration.
func (s *bedrockStore) guardrailPolicy(identifier, version string) (map[string]any, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	g := s.guardrailLocked(strings.TrimSpace(identifier))
	if g == nil {
		return nil, false
	}
	cfg, ok := g.versions[strings.TrimSpace(version)]
	if !ok {
		return nil, false
	}
	return bedrockCloneMap(cfg), true
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
