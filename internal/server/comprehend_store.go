package server

import (
	"fmt"
	"strings"
	"sync"
)

type comprehendStore struct {
	mu     sync.Mutex
	nextID int64
}

func newComprehendStore() *comprehendStore {
	return &comprehendStore{nextID: 1}
}

func (s *comprehendStore) Handle(action string, payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch action {
	case "BatchDetectDominantLanguage", "BatchDetectEntities", "BatchDetectKeyPhrases", "BatchDetectSentiment", "BatchDetectSyntax", "BatchDetectTargetedSentiment":
		return map[string]any{"ResultList": []any{}, "ErrorList": []any{}}
	case "DetectDominantLanguage":
		return map[string]any{"Languages": []any{map[string]any{"LanguageCode": "en", "Score": 0.99}}}
	case "DetectEntities":
		return map[string]any{"Entities": []any{}, "DocumentMetadata": map[string]any{"Pages": 1}}
	case "DetectKeyPhrases":
		return map[string]any{"KeyPhrases": []any{}}
	case "DetectPiiEntities":
		return map[string]any{"Entities": []any{}}
	case "DetectSentiment":
		return map[string]any{"Sentiment": "NEUTRAL", "SentimentScore": map[string]any{"Positive": 0.25, "Negative": 0.25, "Neutral": 0.5, "Mixed": 0.0}}
	case "DetectSyntax":
		return map[string]any{"SyntaxTokens": []any{}}
	case "DetectTargetedSentiment":
		return map[string]any{"Entities": []any{}}
	case "DetectToxicContent":
		return map[string]any{"ResultList": []any{}}
	case "ClassifyDocument":
		return map[string]any{"Classes": []any{}, "Labels": []any{}}
	case "ContainsPiiEntities":
		return map[string]any{"Labels": []any{}}
	case "PutResourcePolicy":
		return map[string]any{"PolicyRevisionId": "1"}
	case "DescribeResourcePolicy":
		return map[string]any{"ResourcePolicy": `{"Version":"2012-10-17","Statement":[]}`, "PolicyRevisionId": "1"}
	case "DeleteResourcePolicy", "TagResource", "UntagResource":
		return map[string]any{}
	case "ListTagsForResource":
		return map[string]any{"ResourceArn": comprehendDefaultString(payload, "ResourceArn", comprehendDefaultString(payload, "resourceArn", "arn:aws:comprehend:us-east-1:123456789012:document-classifier/stackyard-classifier")), "Tags": []any{}}
	case "ImportModel":
		return map[string]any{"ModelArn": fmt.Sprintf("arn:aws:comprehend:us-east-1:123456789012:model/stackyard-model-%06d", s.nextLocked())}
	}

	if strings.HasPrefix(action, "Create") {
		resource := strings.TrimPrefix(action, "Create")
		name := comprehendEntityName(payload, resource)
		if resource == "Dataset" {
			return map[string]any{"DatasetArn": comprehendARN("dataset", name)}
		}
		if resource == "DocumentClassifier" {
			return map[string]any{"DocumentClassifierArn": comprehendARN("document-classifier", name)}
		}
		if resource == "EntityRecognizer" {
			return map[string]any{"EntityRecognizerArn": comprehendARN("entity-recognizer", name)}
		}
		if resource == "Endpoint" {
			return map[string]any{"EndpointArn": comprehendARN("endpoint", name)}
		}
		if resource == "Flywheel" {
			return map[string]any{"FlywheelArn": comprehendARN("flywheel", name)}
		}
		return map[string]any{}
	}

	if strings.HasPrefix(action, "Update") {
		resource := strings.TrimPrefix(action, "Update")
		name := comprehendEntityName(payload, resource)
		if resource == "Endpoint" {
			return map[string]any{"EndpointArn": comprehendARN("endpoint", name)}
		}
		if resource == "Flywheel" {
			return map[string]any{"FlywheelArn": comprehendARN("flywheel", name)}
		}
		return map[string]any{}
	}

	if strings.HasPrefix(action, "Delete") || strings.HasPrefix(action, "Stop") {
		return map[string]any{}
	}

	if strings.HasPrefix(action, "Start") {
		resource := strings.TrimPrefix(strings.TrimSuffix(action, "Job"), "Start")
		if strings.HasSuffix(action, "Job") {
			return map[string]any{"JobId": fmt.Sprintf("%s-%06d", strings.ToLower(resource), s.nextLocked()), "JobStatus": "IN_PROGRESS"}
		}
		if action == "StartFlywheelIteration" {
			return map[string]any{"FlywheelArn": comprehendARN("flywheel", comprehendEntityName(payload, "Flywheel")), "FlywheelIterationId": fmt.Sprintf("iter-%06d", s.nextLocked())}
		}
		return map[string]any{}
	}

	if strings.HasPrefix(action, "Describe") {
		resource := strings.TrimPrefix(action, "Describe")
		summaryKey := resource
		if strings.HasSuffix(resource, "Job") {
			return map[string]any{resource + "Properties": map[string]any{"JobId": fmt.Sprintf("job-%06d", s.nextLocked()), "JobStatus": "COMPLETED"}}
		}
		if resource == "DocumentClassifier" {
			return map[string]any{"DocumentClassifierProperties": map[string]any{"DocumentClassifierArn": comprehendARN("document-classifier", comprehendEntityName(payload, "DocumentClassifier")), "Status": "TRAINED"}}
		}
		if resource == "EntityRecognizer" {
			return map[string]any{"EntityRecognizerProperties": map[string]any{"EntityRecognizerArn": comprehendARN("entity-recognizer", comprehendEntityName(payload, "EntityRecognizer")), "Status": "TRAINED"}}
		}
		if resource == "Endpoint" {
			return map[string]any{"EndpointProperties": map[string]any{"EndpointArn": comprehendARN("endpoint", comprehendEntityName(payload, "Endpoint")), "Status": "IN_SERVICE"}}
		}
		if resource == "Flywheel" {
			return map[string]any{"FlywheelProperties": map[string]any{"FlywheelArn": comprehendARN("flywheel", comprehendEntityName(payload, "Flywheel")), "Status": "ACTIVE"}}
		}
		if resource == "Dataset" {
			return map[string]any{"DatasetProperties": map[string]any{"DatasetArn": comprehendARN("dataset", comprehendEntityName(payload, "Dataset")), "Status": "COMPLETED"}}
		}
		return map[string]any{summaryKey: map[string]any{}}
	}

	if strings.HasPrefix(action, "List") {
		if action == "ListDocumentClassifiers" || action == "ListDocumentClassifierSummaries" {
			return map[string]any{"DocumentClassifierPropertiesList": []any{}, "DocumentClassifierSummariesList": []any{}, "NextToken": ""}
		}
		if action == "ListEntityRecognizers" || action == "ListEntityRecognizerSummaries" {
			return map[string]any{"EntityRecognizerPropertiesList": []any{}, "EntityRecognizerSummariesList": []any{}, "NextToken": ""}
		}
		if action == "ListEndpoints" {
			return map[string]any{"EndpointPropertiesList": []any{}, "NextToken": ""}
		}
		if action == "ListFlywheels" {
			return map[string]any{"FlywheelSummaryList": []any{}, "NextToken": ""}
		}
		if action == "ListFlywheelIterationHistory" {
			return map[string]any{"FlywheelIterationPropertiesList": []any{}, "NextToken": ""}
		}
		if action == "ListDatasets" {
			return map[string]any{"DatasetPropertiesList": []any{}, "NextToken": ""}
		}
		return map[string]any{"NextToken": ""}
	}

	return map[string]any{}
}

func (s *comprehendStore) nextLocked() int64 {
	id := s.nextID
	s.nextID++
	return id
}

func comprehendEntityName(payload map[string]any, resource string) string {
	if payload == nil {
		return "stackyard"
	}
	candidates := []string{
		resource + "Name",
		resource + "Arn",
		strings.ToLower(resource[:1]) + resource[1:] + "Name",
		strings.ToLower(resource[:1]) + resource[1:] + "Arn",
		"ResourceArn",
		"resourceArn",
	}
	for _, key := range candidates {
		if value := comprehendPayloadValue(payload, key); strings.TrimSpace(value) != "" {
			return comprehendNameFromARN(value)
		}
	}
	if resource == "DocumentClassifier" {
		return "stackyard-classifier"
	}
	if resource == "EntityRecognizer" {
		return "stackyard-recognizer"
	}
	if resource == "Endpoint" {
		return "stackyard-endpoint"
	}
	if resource == "Flywheel" {
		return "stackyard-flywheel"
	}
	if resource == "Dataset" {
		return "stackyard-dataset"
	}
	return "stackyard"
}

func comprehendPayloadValue(payload map[string]any, key string) string {
	for k, v := range payload {
		if strings.EqualFold(k, key) {
			return strings.TrimSpace(fmt.Sprintf("%v", v))
		}
	}
	return ""
}

func comprehendDefaultString(payload map[string]any, key, fallback string) string {
	if value := comprehendPayloadValue(payload, key); value != "" {
		return value
	}
	return fallback
}

func comprehendARN(resource, name string) string {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		trimmed = "stackyard"
	}
	if strings.HasPrefix(trimmed, "arn:") {
		return trimmed
	}
	return fmt.Sprintf("arn:aws:comprehend:us-east-1:123456789012:%s/%s", resource, trimmed)
}

func comprehendNameFromARN(value string) string {
	v := strings.TrimSpace(value)
	if v == "" {
		return "stackyard"
	}
	if !strings.HasPrefix(v, "arn:") {
		return v
	}
	if slash := strings.LastIndex(v, "/"); slash >= 0 && slash+1 < len(v) {
		return v[slash+1:]
	}
	if colon := strings.LastIndex(v, ":"); colon >= 0 && colon+1 < len(v) {
		return v[colon+1:]
	}
	return "stackyard"
}
