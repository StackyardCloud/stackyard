package server

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

type augmentedAIHumanLoop struct {
	Name             string
	Arn              string
	FlowDefinition   string
	Status           string
	FailureReason    string
	CreationTime     time.Time
	FailureCode      string
	OutputS3Location string
}

type augmentedAIStore struct {
	mu         sync.Mutex
	next       int64
	humanLoops map[string]*augmentedAIHumanLoop
}

func newAugmentedAIStore() *augmentedAIStore {
	now := time.Now().UTC()
	seedName := "stackyard-human-loop"
	seed := &augmentedAIHumanLoop{
		Name:             seedName,
		Arn:              augmentedAIHumanLoopARN(seedName),
		FlowDefinition:   "arn:aws:sagemaker:us-east-1:123456789012:flow-definition/stackyard-flow-definition",
		Status:           "InProgress",
		FailureReason:    "",
		CreationTime:     now,
		FailureCode:      "",
		OutputS3Location: "s3://stackyard-a2i/output/stackyard-human-loop.json",
	}
	return &augmentedAIStore{
		next: 1,
		humanLoops: map[string]*augmentedAIHumanLoop{
			seedName: seed,
		},
	}
}

func (s *augmentedAIStore) Handle(action string, payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()

	switch action {
	case "StartHumanLoop":
		name := augmentedAIResolveName(payload, pathParams)
		if name == "" {
			name = s.nextID("human-loop")
		}
		flowArn := augmentedAIDefaultString(payload, "FlowDefinitionArn", "arn:aws:sagemaker:us-east-1:123456789012:flow-definition/stackyard-flow-definition")
		loop := &augmentedAIHumanLoop{
			Name:             name,
			Arn:              augmentedAIHumanLoopARN(name),
			FlowDefinition:   flowArn,
			Status:           "InProgress",
			FailureReason:    "",
			CreationTime:     now,
			FailureCode:      "",
			OutputS3Location: fmt.Sprintf("s3://stackyard-a2i/output/%s.json", name),
		}
		s.humanLoops[name] = loop
		return map[string]any{"HumanLoopArn": loop.Arn}

	case "StopHumanLoop":
		name := augmentedAIResolveName(payload, pathParams)
		if name == "" {
			name = "stackyard-human-loop"
		}
		loop, ok := s.humanLoops[name]
		if !ok {
			loop = &augmentedAIHumanLoop{
				Name:             name,
				Arn:              augmentedAIHumanLoopARN(name),
				FlowDefinition:   "arn:aws:sagemaker:us-east-1:123456789012:flow-definition/stackyard-flow-definition",
				Status:           "InProgress",
				CreationTime:     now,
				OutputS3Location: fmt.Sprintf("s3://stackyard-a2i/output/%s.json", name),
			}
			s.humanLoops[name] = loop
		}
		loop.Status = "Stopped"
		return map[string]any{}

	case "DeleteHumanLoop":
		name := augmentedAIResolveName(payload, pathParams)
		if name == "" {
			name = "stackyard-human-loop"
		}
		delete(s.humanLoops, name)
		return map[string]any{}

	case "DescribeHumanLoop":
		name := augmentedAIResolveName(payload, pathParams)
		if name == "" {
			name = "stackyard-human-loop"
		}
		loop, ok := s.humanLoops[name]
		if !ok {
			loop = &augmentedAIHumanLoop{
				Name:             name,
				Arn:              augmentedAIHumanLoopARN(name),
				FlowDefinition:   "arn:aws:sagemaker:us-east-1:123456789012:flow-definition/stackyard-flow-definition",
				Status:           "Completed",
				CreationTime:     now,
				OutputS3Location: fmt.Sprintf("s3://stackyard-a2i/output/%s.json", name),
			}
			s.humanLoops[name] = loop
		}
		return map[string]any{
			"CreationTime":      loop.CreationTime,
			"FailureCode":       loop.FailureCode,
			"FailureReason":     loop.FailureReason,
			"FlowDefinitionArn": loop.FlowDefinition,
			"HumanLoopArn":      loop.Arn,
			"HumanLoopName":     loop.Name,
			"HumanLoopOutput": map[string]any{
				"OutputS3Uri": loop.OutputS3Location,
			},
			"HumanLoopStatus": loop.Status,
		}

	case "ListHumanLoops":
		flowFilter := strings.TrimSpace(query.Get("FlowDefinitionArn"))
		if flowFilter == "" {
			flowFilter = strings.TrimSpace(augmentedAIDefaultString(payload, "FlowDefinitionArn", ""))
		}

		statusFilter := strings.TrimSpace(query.Get("HumanLoopStatus"))
		if statusFilter == "" {
			statusFilter = strings.TrimSpace(augmentedAIDefaultString(payload, "HumanLoopStatus", ""))
		}

		names := make([]string, 0, len(s.humanLoops))
		for name := range s.humanLoops {
			names = append(names, name)
		}
		sort.Strings(names)

		summaries := make([]any, 0, len(names))
		for _, name := range names {
			loop := s.humanLoops[name]
			if flowFilter != "" && !strings.EqualFold(loop.FlowDefinition, flowFilter) {
				continue
			}
			if statusFilter != "" && !strings.EqualFold(loop.Status, statusFilter) {
				continue
			}
			summaries = append(summaries, map[string]any{
				"CreationTime":      loop.CreationTime,
				"FailureReason":     loop.FailureReason,
				"FlowDefinitionArn": loop.FlowDefinition,
				"HumanLoopName":     loop.Name,
				"HumanLoopStatus":   loop.Status,
			})
		}

		return map[string]any{
			"HumanLoopSummaries": summaries,
			"NextToken":          "",
		}
	}

	return map[string]any{}
}

func (s *augmentedAIStore) nextID(prefix string) string {
	s.next++
	return fmt.Sprintf("%s-%06d", prefix, s.next)
}

func augmentedAIResolveName(payload map[string]any, pathParams map[string]string) string {
	if value := augmentedAIPathParam(pathParams, "HumanLoopName", ""); value != "" {
		return value
	}
	if value := augmentedAIDefaultString(payload, "HumanLoopName", ""); value != "" {
		return value
	}
	return ""
}

func augmentedAIHumanLoopARN(name string) string {
	return fmt.Sprintf("arn:aws:sagemaker:us-east-1:123456789012:human-loop/%s", name)
}

func augmentedAIDefaultString(payload map[string]any, key, fallback string) string {
	value := augmentedAIValue(payload, key)
	text := strings.TrimSpace(augmentedAIToString(value))
	if text == "" || text == "<nil>" {
		return fallback
	}
	return text
}

func augmentedAIPathParam(pathParams map[string]string, key, fallback string) string {
	if pathParams == nil {
		return fallback
	}
	if value, ok := pathParams[key]; ok {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	for k, value := range pathParams {
		if strings.EqualFold(k, key) {
			value = strings.TrimSpace(value)
			if value != "" {
				return value
			}
		}
	}
	return fallback
}

func augmentedAIValue(payload map[string]any, key string) any {
	if payload == nil {
		return nil
	}
	if value, ok := payload[key]; ok {
		return value
	}
	for k, value := range payload {
		if strings.EqualFold(k, key) {
			return value
		}
	}
	return nil
}

func augmentedAIToString(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	default:
		return fmt.Sprintf("%v", value)
	}
}
