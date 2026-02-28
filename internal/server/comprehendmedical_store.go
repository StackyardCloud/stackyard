package server

import (
	"fmt"
	"strings"
	"sync"
)

type comprehendMedicalStore struct {
	mu     sync.Mutex
	nextID int64
}

func newComprehendMedicalStore() *comprehendMedicalStore {
	return &comprehendMedicalStore{nextID: 1}
}

func (s *comprehendMedicalStore) Handle(action string, payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch action {
	case "DetectEntitiesV2", "DetectPHI", "InferICD10CM", "InferRxNorm", "InferSNOMEDCT":
		return map[string]any{"Entities": []any{}}
	}

	if strings.HasPrefix(action, "Start") {
		return map[string]any{
			"JobId": comprehendMedicalPayloadString(payload, "JobId", fmt.Sprintf("job-%06d", s.nextLocked())),
		}
	}

	if strings.HasPrefix(action, "Stop") {
		return map[string]any{
			"JobId":     comprehendMedicalPayloadString(payload, "JobId", "job-000001"),
			"JobStatus": "STOP_REQUESTED",
		}
	}

	if strings.HasPrefix(action, "Describe") {
		return map[string]any{
			"ComprehendMedicalAsyncJobProperties": map[string]any{
				"JobId":             comprehendMedicalPayloadString(payload, "JobId", "job-000001"),
				"JobStatus":         "COMPLETED",
				"SubmitTime":        1.0,
				"EndTime":           2.0,
				"InputDataConfig":   map[string]any{"S3Bucket": "stackyard-input", "S3Key": "input.json"},
				"OutputDataConfig":  map[string]any{"S3Bucket": "stackyard-output", "S3Key": "output/"},
				"DataAccessRoleArn": "arn:aws:iam::123456789012:role/stackyard-comprehend-medical",
				"LanguageCode":      "en",
				"ModelVersion":      "1.0.0",
			},
		}
	}

	if strings.HasPrefix(action, "List") {
		return map[string]any{
			"ComprehendMedicalAsyncJobPropertiesList": []any{
				map[string]any{
					"JobId":             "job-000001",
					"JobStatus":         "COMPLETED",
					"SubmitTime":        1.0,
					"EndTime":           2.0,
					"InputDataConfig":   map[string]any{"S3Bucket": "stackyard-input", "S3Key": "input.json"},
					"OutputDataConfig":  map[string]any{"S3Bucket": "stackyard-output", "S3Key": "output/"},
					"DataAccessRoleArn": "arn:aws:iam::123456789012:role/stackyard-comprehend-medical",
					"LanguageCode":      "en",
					"ModelVersion":      "1.0.0",
				},
			},
			"NextToken": "",
		}
	}

	return map[string]any{}
}

func (s *comprehendMedicalStore) nextLocked() int64 {
	id := s.nextID
	s.nextID++
	return id
}

func comprehendMedicalPayloadString(payload map[string]any, key, fallback string) string {
	if payload == nil {
		return fallback
	}
	for k, v := range payload {
		if strings.EqualFold(k, key) {
			if s := strings.TrimSpace(fmt.Sprintf("%v", v)); s != "" {
				return s
			}
			break
		}
	}
	return fallback
}
