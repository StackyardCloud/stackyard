package server

import (
	"fmt"
	"strings"
	"sync"
)

type bedrockStore struct {
	mu     sync.Mutex
	nextID int64
}

func newBedrockStore() *bedrockStore {
	return &bedrockStore{nextID: 1}
}

func (s *bedrockStore) Handle(action string, payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch action {
	case "ListFoundationModels":
		return map[string]any{"modelSummaries": []any{}, "nextToken": ""}
	case "GetFoundationModel":
		return map[string]any{
			"modelDetails": map[string]any{
				"modelArn":     "arn:aws:bedrock:us-east-1::foundation-model/stackyard.fm-v1",
				"modelId":      "stackyard.fm-v1",
				"modelName":    "Stackyard Foundation Model",
				"providerName": "Stackyard",
			},
		}
	case "ListTagsForResource":
		return map[string]any{"tags": []any{}}
	case "TagResource", "UntagResource":
		return map[string]any{}
	}

	if strings.HasPrefix(action, "Create") {
		resource := strings.TrimPrefix(action, "Create")
		name := bedrockEntityName(payload, resource)
		key := bedrockLowerFirst(resource) + "Arn"
		if strings.HasSuffix(resource, "Job") {
			key = bedrockLowerFirst(resource) + "Arn"
		}
		if resource == "FoundationModelAgreement" {
			return map[string]any{"modelAgreementArn": bedrockARN("model-agreement", name)}
		}
		return map[string]any{key: bedrockARN(bedrockResourceSlug(resource), name)}
	}

	if strings.HasPrefix(action, "Start") {
		resource := strings.TrimPrefix(action, "Start")
		if strings.HasSuffix(resource, "Workflow") {
			idKey := bedrockLowerFirst(resource) + "Id"
			return map[string]any{idKey: fmt.Sprintf("%s-%06d", bedrockResourceSlug(resource), s.nextLocked())}
		}
		return map[string]any{"status": "IN_PROGRESS"}
	}

	if strings.HasPrefix(action, "Stop") || strings.HasPrefix(action, "Cancel") || strings.HasPrefix(action, "Delete") || strings.HasPrefix(action, "Deregister") || strings.HasPrefix(action, "Put") || strings.HasPrefix(action, "Update") || strings.HasPrefix(action, "Register") || strings.HasPrefix(action, "BatchDelete") {
		return map[string]any{}
	}

	if strings.HasPrefix(action, "Get") {
		resource := strings.TrimPrefix(action, "Get")
		key := bedrockLowerFirst(resource)
		return map[string]any{
			key: map[string]any{
				"arn":    bedrockARN(bedrockResourceSlug(resource), bedrockEntityName(payload, resource)),
				"status": "ACTIVE",
			},
		}
	}

	if strings.HasPrefix(action, "List") {
		resource := strings.TrimPrefix(action, "List")
		key := bedrockLowerFirst(resource)
		if key == "tagsForResource" {
			key = "tags"
		}
		return map[string]any{key: []any{}, "nextToken": ""}
	}

	if strings.HasPrefix(action, "Export") {
		return map[string]any{"exportArn": bedrockARN("export", fmt.Sprintf("export-%06d", s.nextLocked()))}
	}

	return map[string]any{}
}

func (s *bedrockStore) nextLocked() int64 {
	id := s.nextID
	s.nextID++
	return id
}

func bedrockLowerFirst(value string) string {
	v := strings.TrimSpace(value)
	if v == "" {
		return "value"
	}
	if len(v) == 1 {
		return strings.ToLower(v)
	}
	return strings.ToLower(v[:1]) + v[1:]
}

func bedrockResourceSlug(value string) string {
	v := strings.TrimSpace(value)
	if v == "" {
		return "resource"
	}
	var out []rune
	for i, r := range v {
		if i > 0 && r >= 'A' && r <= 'Z' {
			out = append(out, '-')
		}
		if r >= 'A' && r <= 'Z' {
			out = append(out, r+32)
		} else {
			out = append(out, r)
		}
	}
	return string(out)
}

func bedrockARN(resource, name string) string {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		trimmed = "stackyard"
	}
	if strings.HasPrefix(trimmed, "arn:") {
		return trimmed
	}
	return fmt.Sprintf("arn:aws:bedrock:us-east-1:123456789012:%s/%s", resource, trimmed)
}

func bedrockEntityName(payload map[string]any, resource string) string {
	if payload != nil {
		candidates := []string{
			resource + "Name",
			resource + "Arn",
			"resourceName",
			"resourceArn",
			"ResourceArn",
			"jobIdentifier",
			"jobArn",
			"modelIdentifier",
			"modelArn",
		}
		for _, candidate := range candidates {
			for k, raw := range payload {
				if strings.EqualFold(k, candidate) {
					value := strings.TrimSpace(fmt.Sprintf("%v", raw))
					if value != "" {
						if strings.HasPrefix(value, "arn:") {
							if slash := strings.LastIndex(value, "/"); slash >= 0 && slash+1 < len(value) {
								return value[slash+1:]
							}
						}
						return value
					}
				}
			}
		}
	}
	return "stackyard"
}
