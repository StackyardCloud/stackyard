package server

import (
	"net/url"
	"strings"
	"sync"
	"time"
)

type ivsMultitrackStore struct {
	mu sync.Mutex

	ingest                map[string]any
	ingestEndpoint        map[string]any
	configurationMetadata map[string]any
	capabilities          map[string]any
	preferences           map[string]any
	system                map[string]any
	encoderConfiguration  map[string]any
	audioConfiguration    map[string]any
	audioTrackConfig      map[string]any
	audioTrackSettings    map[string]any
	videoTrackSettings    map[string]any
	clients               map[string]map[string]any
}

func newIVSMultitrackStore() *ivsMultitrackStore {
	return &ivsMultitrackStore{
		ingest: map[string]any{
			"name":   "stackyard-default",
			"region": "us-east-1",
		},
		ingestEndpoint: map[string]any{
			"protocol": "RTMPS",
			"url":      "rtmps://ingest.stackyard.local/app",
			"port":     443,
		},
		configurationMetadata: map[string]any{
			"version":   "1.0",
			"source":    "stackyard",
			"updatedAt": time.Now().UTC().Format(time.RFC3339),
		},
		capabilities: map[string]any{
			"cpu":            map[string]any{"architecture": "x86_64", "cores": 8},
			"gpu":            map[string]any{"vendor": "NVIDIA", "model": "T4"},
			"memory":         map[string]any{"sizeMb": 16384},
			"gamingFeatures": map[string]any{"supportsVariableFrameRate": true},
		},
		preferences: map[string]any{
			"maxBitrateKbps": 4500,
			"latencyMode":    "LOW",
		},
		system: map[string]any{
			"platform":  "linux",
			"osVersion": "6.1",
			"hostname":  "stackyard-client",
		},
		encoderConfiguration: map[string]any{
			"videoCodec": "h264",
			"audioCodec": "aac",
			"profile":    "main",
		},
		audioConfiguration: map[string]any{
			"sampleRateHz": 48000,
			"channels":     2,
			"bitrateKbps":  128,
		},
		audioTrackConfig: map[string]any{
			"name":            "main",
			"languageCode":    "en-US",
			"trackIndex":      0,
			"isDefault":       true,
			"isInterleavable": true,
		},
		audioTrackSettings: map[string]any{
			"name":        "main",
			"bitrateKbps": 128,
			"enabled":     true,
		},
		videoTrackSettings: map[string]any{
			"resolution":  "1920x1080",
			"framerate":   map[string]any{"numerator": 60, "denominator": 1},
			"bitrateKbps": 4500,
		},
		clients: map[string]map[string]any{
			"client-00000001": {
				"id":        "client-00000001",
				"name":      "stackyard-broadcast-client",
				"version":   "1.0.0",
				"platform":  "macOS",
				"updatedAt": time.Now().UTC().Format(time.RFC3339),
			},
		},
	}
}

func (s *ivsMultitrackStore) Handle(action string, payload map[string]any, _ map[string]string, query url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC().Format(time.RFC3339)
	clientID := ivsMultitrackFirstNonEmpty(
		strings.TrimSpace(query.Get("clientId")),
		ivsMultitrackStringAny(payload, "clientId", "clientID"),
		ivsMultitrackNestedString(payload, "client", "id"),
		"client-00000001",
	)
	client := s.ensureClientLocked(clientID)

	s.configurationMetadata["updatedAt"] = now
	client["updatedAt"] = now
	client["lastSeenAt"] = now

	switch action {
	case "FindIngest":
		return map[string]any{
			"client":          ivsMultitrackCloneMap(client),
			"ingest":          ivsMultitrackCloneMap(s.ingest),
			"ingestEndpoint":  ivsMultitrackCloneMap(s.ingestEndpoint),
			"ingestEndpoints": []any{ivsMultitrackCloneMap(s.ingestEndpoint)},
		}
	case "GetClientConfiguration":
		if requestClient, ok := payload["client"].(map[string]any); ok {
			for key, value := range requestClient {
				client[key] = value
			}
			client["id"] = clientID
		}
		return map[string]any{
			"client":                    ivsMultitrackCloneMap(client),
			"clientConfigurationStatus": map[string]any{"state": "READY", "message": "configuration generated", "updatedAt": now},
			"configurationMetadata":     ivsMultitrackCloneMap(s.configurationMetadata),
			"capabilities":              ivsMultitrackCloneMap(s.capabilities),
			"preferences":               ivsMultitrackCloneMap(s.preferences),
			"system":                    ivsMultitrackCloneMap(s.system),
			"encoderConfiguration":      ivsMultitrackCloneMap(s.encoderConfiguration),
			"audioConfiguration":        ivsMultitrackCloneMap(s.audioConfiguration),
			"audioTrackConfiguration":   ivsMultitrackCloneMap(s.audioTrackConfig),
			"audioTrackSettings":        []any{ivsMultitrackCloneMap(s.audioTrackSettings)},
			"videoTrackSettings":        ivsMultitrackCloneMap(s.videoTrackSettings),
			"ingest":                    ivsMultitrackCloneMap(s.ingest),
			"ingestEndpoint":            ivsMultitrackCloneMap(s.ingestEndpoint),
			"ingestEndpoints":           []any{ivsMultitrackCloneMap(s.ingestEndpoint)},
		}
	default:
		return map[string]any{}
	}
}

func (s *ivsMultitrackStore) ensureClientLocked(clientID string) map[string]any {
	if existing, ok := s.clients[clientID]; ok {
		return existing
	}
	client := map[string]any{
		"id":       clientID,
		"name":     "stackyard-broadcast-client",
		"version":  "1.0.0",
		"platform": "linux",
	}
	s.clients[clientID] = client
	return client
}

func ivsMultitrackNestedString(payload map[string]any, key, nestedKey string) string {
	value, ok := payload[key]
	if !ok {
		return ""
	}
	inner, ok := value.(map[string]any)
	if !ok {
		return ""
	}
	return ivsMultitrackStringAny(inner, nestedKey)
}

func ivsMultitrackStringAny(payload map[string]any, keys ...string) string {
	for _, key := range keys {
		value, ok := payload[key]
		if !ok || value == nil {
			continue
		}
		switch typed := value.(type) {
		case string:
			trimmed := strings.TrimSpace(typed)
			if trimmed != "" {
				return trimmed
			}
		}
	}
	return ""
}

func ivsMultitrackFirstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func ivsMultitrackCloneMap(input map[string]any) map[string]any {
	out := make(map[string]any, len(input))
	for key, value := range input {
		out[key] = ivsMultitrackCloneAny(value)
	}
	return out
}

func ivsMultitrackCloneAny(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return ivsMultitrackCloneMap(typed)
	case []any:
		out := make([]any, 0, len(typed))
		for _, item := range typed {
			out = append(out, ivsMultitrackCloneAny(item))
		}
		return out
	default:
		return typed
	}
}
