package server

import (
	"strings"
	"sync"
)

type ssmGUIConnectStore struct {
	mu sync.Mutex

	connectionRecordingPreferences map[string]any
}

func newSSMGUIConnectStore() *ssmGUIConnectStore {
	return &ssmGUIConnectStore{
		connectionRecordingPreferences: map[string]any{
			"KmsKeyArn": "arn:aws:kms:us-east-1:123456789012:key/stackyard-ssm-guiconnect-key",
			"RecordingDestinations": map[string]any{
				"S3Buckets": []any{
					map[string]any{
						"BucketName":  "stackyard-ssm-guiconnect-recordings",
						"BucketOwner": "123456789012",
						"KeyPrefix":   "session-recordings/",
					},
				},
			},
		},
	}
}

func (s *ssmGUIConnectStore) Handle(action string, payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch action {
	case "GetConnectionRecordingPreferences":
		return map[string]any{
			"ConnectionRecordingPreferences": ssmGUIConnectCloneMap(s.connectionRecordingPreferences),
		}

	case "UpdateConnectionRecordingPreferences":
		prefs := ssmGUIConnectMapAny(payload, "ConnectionRecordingPreferences")
		if len(prefs) == 0 {
			prefs = ssmGUIConnectCloneMap(payload)
		}
		s.connectionRecordingPreferences = prefs
		return map[string]any{}

	case "DeleteConnectionRecordingPreferences":
		s.connectionRecordingPreferences = map[string]any{}
		return map[string]any{}
	}

	return map[string]any{}
}

func ssmGUIConnectMapAny(values map[string]any, key string) map[string]any {
	for k, v := range values {
		if strings.EqualFold(strings.TrimSpace(k), strings.TrimSpace(key)) {
			if m, ok := v.(map[string]any); ok {
				return ssmGUIConnectCloneMap(m)
			}
		}
	}
	return map[string]any{}
}

func ssmGUIConnectCloneMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		switch tv := v.(type) {
		case map[string]any:
			out[k] = ssmGUIConnectCloneMap(tv)
		case []any:
			out[k] = ssmGUIConnectCloneSlice(tv)
		default:
			out[k] = tv
		}
	}
	return out
}

func ssmGUIConnectCloneSlice(in []any) []any {
	out := make([]any, 0, len(in))
	for _, v := range in {
		switch tv := v.(type) {
		case map[string]any:
			out = append(out, ssmGUIConnectCloneMap(tv))
		case []any:
			out = append(out, ssmGUIConnectCloneSlice(tv))
		default:
			out = append(out, tv)
		}
	}
	return out
}
