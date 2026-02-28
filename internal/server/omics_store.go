package server

import (
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"
)

type omicsStore struct {
	mu   sync.Mutex
	next int64
	tags map[string]map[string]string
}

func newOmicsStore() *omicsStore {
	return &omicsStore{
		next: 1,
		tags: map[string]map[string]string{
			"arn:aws:omics:us-east-1:123456789012:resource/stackyard": {
				"seed": "true",
			},
		},
	}
}

func (s *omicsStore) Handle(action string, payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()

	switch action {
	case "TagResource":
		arn := omicsResolveResourceARN(payload, pathParams, query)
		if arn == "" {
			arn = "arn:aws:omics:us-east-1:123456789012:resource/stackyard"
		}
		incoming := omicsExtractTagMap(omicsValue(payload, "tags"))
		current := s.tags[arn]
		if current == nil {
			current = map[string]string{}
		}
		for k, v := range incoming {
			current[k] = v
		}
		s.tags[arn] = current
		return map[string]any{}

	case "UntagResource":
		arn := omicsResolveResourceARN(payload, pathParams, query)
		if arn == "" {
			arn = "arn:aws:omics:us-east-1:123456789012:resource/stackyard"
		}
		current := s.tags[arn]
		if current == nil {
			return map[string]any{}
		}
		for _, key := range omicsStringSlice(omicsValue(payload, "tagKeys")) {
			delete(current, key)
		}
		for _, key := range query["tagKeys"] {
			key = strings.TrimSpace(key)
			if key != "" {
				delete(current, key)
			}
		}
		s.tags[arn] = current
		return map[string]any{}

	case "ListTagsForResource":
		arn := omicsResolveResourceARN(payload, pathParams, query)
		if arn == "" {
			arn = "arn:aws:omics:us-east-1:123456789012:resource/stackyard"
		}
		return map[string]any{"tags": omicsCloneStringMap(s.tags[arn])}

	case "BatchDeleteReadSet":
		return map[string]any{"errors": []any{}}
	case "CompleteMultipartReadSetUpload":
		return map[string]any{"status": "COMPLETED"}
	case "UploadReadSetPart":
		return map[string]any{"eTag": "stackyard-etag"}
	case "PutS3AccessPolicy":
		return map[string]any{"s3AccessPointArn": omicsDefaultString(payload, "s3AccessPointArn", "arn:aws:s3:us-east-1:123456789012:accesspoint/stackyard-omics"), "s3AccessPolicy": omicsValue(payload, "s3AccessPolicy")}
	case "GetS3AccessPolicy":
		return map[string]any{"s3AccessPointArn": omicsDefaultString(payload, "s3AccessPointArn", "arn:aws:s3:us-east-1:123456789012:accesspoint/stackyard-omics"), "s3AccessPolicy": "{\"Version\":\"2012-10-17\",\"Statement\":[]}"}
	case "DeleteS3AccessPolicy":
		return map[string]any{}
	}

	if strings.HasPrefix(action, "List") {
		key := omicsListKey(action)
		entry := map[string]any{
			"id":           omicsResolveEntityID(action, payload, pathParams),
			"name":         "stackyard",
			"arn":          omicsARNFor(action, omicsResolveEntityID(action, payload, pathParams)),
			"status":       "ACTIVE",
			"creationTime": now,
			"updateTime":   now,
		}
		return map[string]any{key: []any{entry}, "nextToken": ""}
	}

	if strings.HasPrefix(action, "Get") || strings.HasPrefix(action, "Describe") {
		id := omicsResolveEntityID(action, payload, pathParams)
		return map[string]any{
			"id":            id,
			"name":          "stackyard",
			"arn":           omicsARNFor(action, id),
			"status":        "ACTIVE",
			"statusMessage": "",
			"creationTime":  now,
			"updateTime":    now,
		}
	}

	if strings.HasPrefix(action, "Create") {
		id := omicsResolveEntityID(action, payload, pathParams)
		if id == "" {
			id = s.nextID("resource")
		}
		return map[string]any{
			"id":     id,
			"arn":    omicsARNFor(action, id),
			"status": "ACTIVE",
		}
	}

	if strings.HasPrefix(action, "Start") {
		id := omicsResolveEntityID(action, payload, pathParams)
		if id == "" {
			id = s.nextID("job")
		}
		return map[string]any{
			"id":     id,
			"arn":    omicsARNFor(action, id),
			"status": "SUBMITTED",
		}
	}

	if strings.HasPrefix(action, "Update") || strings.HasPrefix(action, "Patch") {
		id := omicsResolveEntityID(action, payload, pathParams)
		return map[string]any{
			"id":         id,
			"arn":        omicsARNFor(action, id),
			"status":     "UPDATED",
			"updateTime": now,
		}
	}

	if strings.HasPrefix(action, "Delete") || strings.HasPrefix(action, "Cancel") || strings.HasPrefix(action, "Accept") {
		return map[string]any{}
	}

	return map[string]any{"operation": action, "status": "SUCCESS", "timestamp": now}
}

func (s *omicsStore) nextID(prefix string) string {
	s.next++
	return fmt.Sprintf("%s-%06d", prefix, s.next)
}

func omicsListKey(action string) string {
	keys := map[string]string{
		"ListAnnotationImportJobs":    "annotationImportJobs",
		"ListAnnotationStores":        "annotationStores",
		"ListAnnotationStoreVersions": "annotationStoreVersions",
		"ListMultipartReadSetUploads": "multipartReadSetUploads",
		"ListReadSetActivationJobs":   "readSetActivationJobs",
		"ListReadSetExportJobs":       "readSetExportJobs",
		"ListReadSetImportJobs":       "readSetImportJobs",
		"ListReadSets":                "readSets",
		"ListReadSetUploadParts":      "parts",
		"ListReferenceImportJobs":     "referenceImportJobs",
		"ListReferences":              "references",
		"ListReferenceStores":         "referenceStores",
		"ListRunCaches":               "runCaches",
		"ListRunGroups":               "runGroups",
		"ListRuns":                    "runs",
		"ListRunTasks":                "tasks",
		"ListSequenceStores":          "sequenceStores",
		"ListShares":                  "shares",
		"ListTagsForResource":         "tags",
		"ListVariantImportJobs":       "variantImportJobs",
		"ListVariantStores":           "variantStores",
		"ListWorkflows":               "workflows",
		"ListWorkflowVersions":        "workflowVersions",
	}
	if key := keys[action]; key != "" {
		return key
	}
	return "items"
}

func omicsResolveEntityID(action string, payload map[string]any, pathParams map[string]string) string {
	keys := []string{
		"id",
		"name",
		"workflowId",
		"workflowOwnerId",
		"runGroupId",
		"runId",
		"taskId",
		"sequenceStoreId",
		"referenceStoreId",
		"variantStoreId",
		"annotationStoreName",
		"jobId",
		"shareId",
		"s3AccessPointArn",
	}
	for _, key := range keys {
		if v := strings.TrimSpace(omicsPathParam(pathParams, key, "")); v != "" {
			return v
		}
	}
	for _, key := range keys {
		if v := strings.TrimSpace(omicsDefaultString(payload, key, "")); v != "" {
			return v
		}
	}

	switch {
	case strings.Contains(action, "Sequence"):
		return "stackyard-sequence-store"
	case strings.Contains(action, "Reference"):
		return "stackyard-reference-store"
	case strings.Contains(action, "Variant"):
		return "stackyard-variant-store"
	case strings.Contains(action, "Annotation"):
		return "stackyard-annotation-store"
	case strings.Contains(action, "Workflow"):
		return "stackyard-workflow"
	case strings.Contains(action, "RunGroup"):
		return "stackyard-run-group"
	case strings.Contains(action, "Run"):
		return "stackyard-run"
	case strings.Contains(action, "Share"):
		return "stackyard-share"
	default:
		return "stackyard"
	}
}

func omicsARNFor(action, id string) string {
	resourceType := "resource"
	switch {
	case strings.Contains(action, "Sequence"):
		resourceType = "sequence-store"
	case strings.Contains(action, "Reference"):
		resourceType = "reference-store"
	case strings.Contains(action, "Variant"):
		resourceType = "variant-store"
	case strings.Contains(action, "Annotation"):
		resourceType = "annotation-store"
	case strings.Contains(action, "Workflow"):
		resourceType = "workflow"
	case strings.Contains(action, "RunGroup"):
		resourceType = "run-group"
	case strings.Contains(action, "Run"):
		resourceType = "run"
	case strings.Contains(action, "Share"):
		resourceType = "share"
	}
	return fmt.Sprintf("arn:aws:omics:us-east-1:123456789012:%s/%s", resourceType, id)
}

func omicsResolveResourceARN(payload map[string]any, pathParams map[string]string, query url.Values) string {
	if v := strings.TrimSpace(omicsDefaultString(payload, "resourceArn", "")); v != "" {
		return v
	}
	if v := strings.TrimSpace(omicsPathParam(pathParams, "resourceArn", "")); v != "" {
		return v
	}
	if v := strings.TrimSpace(query.Get("resourceArn")); v != "" {
		return v
	}
	return ""
}

func omicsPathParam(pathParams map[string]string, key, fallback string) string {
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

func omicsValue(payload map[string]any, key string) any {
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

func omicsDefaultString(payload map[string]any, key, fallback string) string {
	value := omicsValue(payload, key)
	text := strings.TrimSpace(omicsToString(value))
	if text == "" {
		return fallback
	}
	return text
}

func omicsToString(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	case []byte:
		return string(v)
	case bool:
		if v {
			return "true"
		}
		return "false"
	default:
		return fmt.Sprintf("%v", value)
	}
}

func omicsStringSlice(value any) []string {
	switch v := value.(type) {
	case []string:
		out := make([]string, 0, len(v))
		for _, item := range v {
			item = strings.TrimSpace(item)
			if item != "" {
				out = append(out, item)
			}
		}
		return out
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			text := strings.TrimSpace(omicsToString(item))
			if text != "" {
				out = append(out, text)
			}
		}
		return out
	default:
		text := strings.TrimSpace(omicsToString(value))
		if text == "" {
			return nil
		}
		return []string{text}
	}
}

func omicsExtractTagMap(value any) map[string]string {
	tags := map[string]string{}
	switch typed := value.(type) {
	case map[string]string:
		for k, v := range typed {
			k = strings.TrimSpace(k)
			if k == "" {
				continue
			}
			tags[k] = strings.TrimSpace(v)
		}
	case map[string]any:
		for rawKey, rawValue := range typed {
			key := strings.TrimSpace(rawKey)
			if key == "" {
				continue
			}
			tags[key] = strings.TrimSpace(omicsToString(rawValue))
		}
	}
	return tags
}

func omicsCloneStringMap(src map[string]string) map[string]string {
	if len(src) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}
