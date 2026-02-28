package server

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

type ivsChatStore struct {
	mu sync.Mutex

	nextRoomID      int64
	nextLoggingID   int64
	nextChatTokenID int64
	nextEventID     int64

	rooms                 map[string]map[string]any
	loggingConfigurations map[string]map[string]any
	tags                  map[string]map[string]string
}

func newIVSChatStore() *ivsChatStore {
	s := &ivsChatStore{
		nextRoomID:            2,
		nextLoggingID:         2,
		nextChatTokenID:       2,
		nextEventID:           2,
		rooms:                 map[string]map[string]any{},
		loggingConfigurations: map[string]map[string]any{},
		tags:                  map[string]map[string]string{},
	}

	room := s.ensureRoomByARNLocked(ivsChatArn("room", "room-00000001"))
	cfg := s.ensureLoggingByARNLocked(ivsChatArn("logging-configuration", "logging-00000001"))
	room["loggingConfigurationIdentifiers"] = []any{ivsChatStringAny(cfg, "id")}
	s.tags[ivsChatStringAny(room, "arn")] = map[string]string{"seed": "true"}
	s.tags[ivsChatStringAny(cfg, "arn")] = map[string]string{"seed": "true"}
	return s
}

func (s *ivsChatStore) Handle(action string, payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	nowStr := now.Format(time.RFC3339)

	roomIdentifier := ivsChatFirstNonEmpty(
		ivsChatStringAny(payload, "identifier", "roomIdentifier", "id", "arn"),
		ivsChatPath(pathParams, "roomIdentifier"),
		"room-00000001",
	)
	loggingIdentifier := ivsChatFirstNonEmpty(
		ivsChatStringAny(payload, "identifier", "loggingConfigurationIdentifier", "id", "arn"),
		"logging-00000001",
	)
	resourceARN := ivsChatFirstNonEmpty(
		ivsChatPath(pathParams, "resourceArn"),
		ivsChatStringAny(payload, "resourceArn", "arn"),
		ivsChatStringAny(s.ensureRoomByIdentifierLocked(roomIdentifier), "arn"),
	)

	switch action {
	case "CreateRoom":
		id := fmt.Sprintf("room-%08d", s.nextRoomIDLocked())
		arn := ivsChatArn("room", id)
		room := s.ensureRoomByARNLocked(arn)
		for k, v := range payload {
			room[k] = v
		}
		room["arn"] = arn
		room["id"] = id
		room["name"] = ivsChatFirstNonEmpty(ivsChatStringAny(payload, "name"), id)
		if _, ok := room["maximumMessageLength"]; !ok {
			room["maximumMessageLength"] = 500
		}
		if _, ok := room["maximumMessageRatePerSecond"]; !ok {
			room["maximumMessageRatePerSecond"] = 10
		}
		if _, ok := room["createdTime"]; !ok {
			room["createdTime"] = nowStr
		}
		room["updatedTime"] = nowStr
		return map[string]any{"room": ivsChatCloneMap(room)}

	case "GetRoom":
		room := s.ensureRoomByIdentifierLocked(roomIdentifier)
		return map[string]any{"room": ivsChatCloneMap(room)}

	case "ListRooms":
		return map[string]any{
			"rooms":     s.listRoomsLocked(),
			"nextToken": "",
		}

	case "UpdateRoom":
		room := s.ensureRoomByIdentifierLocked(roomIdentifier)
		for k, v := range payload {
			room[k] = v
		}
		room["updatedTime"] = nowStr
		return map[string]any{"room": ivsChatCloneMap(room)}

	case "DeleteRoom":
		room := s.ensureRoomByIdentifierLocked(roomIdentifier)
		delete(s.rooms, ivsChatStringAny(room, "arn"))
		return map[string]any{}

	case "CreateLoggingConfiguration":
		id := fmt.Sprintf("logging-%08d", s.nextLoggingIDLocked())
		arn := ivsChatArn("logging-configuration", id)
		cfg := s.ensureLoggingByARNLocked(arn)
		for k, v := range payload {
			cfg[k] = v
		}
		cfg["arn"] = arn
		cfg["id"] = id
		cfg["name"] = ivsChatFirstNonEmpty(ivsChatStringAny(payload, "name"), id)
		if _, ok := cfg["state"]; !ok {
			cfg["state"] = "ACTIVE"
		}
		if _, ok := cfg["createdTime"]; !ok {
			cfg["createdTime"] = nowStr
		}
		cfg["updatedTime"] = nowStr
		return map[string]any{"loggingConfiguration": ivsChatCloneMap(cfg)}

	case "GetLoggingConfiguration":
		cfg := s.ensureLoggingByIdentifierLocked(loggingIdentifier)
		return map[string]any{"loggingConfiguration": ivsChatCloneMap(cfg)}

	case "ListLoggingConfigurations":
		return map[string]any{
			"loggingConfigurations": s.listLoggingConfigurationsLocked(),
			"nextToken":             "",
		}

	case "UpdateLoggingConfiguration":
		cfg := s.ensureLoggingByIdentifierLocked(loggingIdentifier)
		for k, v := range payload {
			cfg[k] = v
		}
		cfg["updatedTime"] = nowStr
		return map[string]any{"loggingConfiguration": ivsChatCloneMap(cfg)}

	case "DeleteLoggingConfiguration":
		cfg := s.ensureLoggingByIdentifierLocked(loggingIdentifier)
		delete(s.loggingConfigurations, ivsChatStringAny(cfg, "arn"))
		return map[string]any{}

	case "CreateChatToken":
		room := s.ensureRoomByIdentifierLocked(roomIdentifier)
		tokenID := s.nextChatTokenIDLocked()
		userID := ivsChatFirstNonEmpty(ivsChatStringAny(payload, "userId"), fmt.Sprintf("user-%08d", tokenID))
		return map[string]any{
			"token":                 fmt.Sprintf("chat-token-%08d", tokenID),
			"sessionExpirationTime": now.Add(1 * time.Hour).Format(time.RFC3339),
			"tokenExpirationTime":   now.Add(15 * time.Minute).Format(time.RFC3339),
			"roomIdentifier":        ivsChatStringAny(room, "id"),
			"userId":                userID,
		}

	case "SendEvent":
		room := s.ensureRoomByIdentifierLocked(roomIdentifier)
		eventID := s.nextEventIDLocked()
		events := ivsChatSliceAny(room, "events")
		event := map[string]any{
			"id":         fmt.Sprintf("evt-%08d", eventID),
			"eventName":  ivsChatFirstNonEmpty(ivsChatStringAny(payload, "eventName"), "stackyard.event"),
			"attributes": ivsChatMapAny(payload, "attributes"),
			"createdAt":  nowStr,
		}
		events = append(events, event)
		room["events"] = events
		room["updatedTime"] = nowStr
		return map[string]any{"id": event["id"]}

	case "DisconnectUser":
		room := s.ensureRoomByIdentifierLocked(roomIdentifier)
		room["lastDisconnectedUserId"] = ivsChatFirstNonEmpty(ivsChatStringAny(payload, "userId"), "user-00000001")
		room["updatedTime"] = nowStr
		return map[string]any{}

	case "DeleteMessage":
		room := s.ensureRoomByIdentifierLocked(roomIdentifier)
		room["lastDeletedMessageId"] = ivsChatFirstNonEmpty(ivsChatStringAny(payload, "id", "messageId"), "msg-00000001")
		room["updatedTime"] = nowStr
		return map[string]any{}

	case "TagResource":
		tags := s.ensureTagsLocked(resourceARN)
		for key, value := range ivsChatReadTags(payload) {
			tags[key] = value
		}
		return map[string]any{}

	case "UntagResource":
		tags := s.ensureTagsLocked(resourceARN)
		for _, key := range ivsChatReadTagKeys(payload, query) {
			delete(tags, key)
		}
		return map[string]any{}

	case "ListTagsForResource":
		return map[string]any{"tags": ivsChatCloneTags(s.ensureTagsLocked(resourceARN))}
	}

	return map[string]any{}
}

func (s *ivsChatStore) nextRoomIDLocked() int64 {
	id := s.nextRoomID
	s.nextRoomID++
	return id
}

func (s *ivsChatStore) nextLoggingIDLocked() int64 {
	id := s.nextLoggingID
	s.nextLoggingID++
	return id
}

func (s *ivsChatStore) nextChatTokenIDLocked() int64 {
	id := s.nextChatTokenID
	s.nextChatTokenID++
	return id
}

func (s *ivsChatStore) nextEventIDLocked() int64 {
	id := s.nextEventID
	s.nextEventID++
	return id
}

func (s *ivsChatStore) firstRoomARNLocked() string {
	keys := make([]string, 0, len(s.rooms))
	for arn := range s.rooms {
		keys = append(keys, arn)
	}
	sort.Strings(keys)
	if len(keys) == 0 {
		return ""
	}
	return keys[0]
}

func (s *ivsChatStore) firstLoggingARNLocked() string {
	keys := make([]string, 0, len(s.loggingConfigurations))
	for arn := range s.loggingConfigurations {
		keys = append(keys, arn)
	}
	sort.Strings(keys)
	if len(keys) == 0 {
		return ""
	}
	return keys[0]
}

func (s *ivsChatStore) ensureRoomByIdentifierLocked(identifier string) map[string]any {
	identifier = strings.TrimSpace(identifier)
	if identifier == "" {
		identifier = "room-00000001"
	}
	if strings.HasPrefix(identifier, "arn:") {
		return s.ensureRoomByARNLocked(identifier)
	}
	for arn, room := range s.rooms {
		if strings.EqualFold(ivsChatStringAny(room, "id"), identifier) || strings.EqualFold(ivsChatStringAny(room, "name"), identifier) {
			return s.ensureRoomByARNLocked(arn)
		}
	}
	return s.ensureRoomByARNLocked(ivsChatArn("room", identifier))
}

func (s *ivsChatStore) ensureRoomByARNLocked(arn string) map[string]any {
	arn = strings.TrimSpace(arn)
	if arn == "" {
		arn = ivsChatFirstNonEmpty(s.firstRoomARNLocked(), ivsChatArn("room", "room-00000001"))
	}
	if room, ok := s.rooms[arn]; ok {
		return room
	}
	id := ivsChatResourceID(arn)
	room := map[string]any{
		"arn":                         arn,
		"id":                          id,
		"name":                        id,
		"maximumMessageLength":        500,
		"maximumMessageRatePerSecond": 10,
		"createdTime":                 time.Now().UTC().Format(time.RFC3339),
		"updatedTime":                 time.Now().UTC().Format(time.RFC3339),
	}
	s.rooms[arn] = room
	return room
}

func (s *ivsChatStore) ensureLoggingByIdentifierLocked(identifier string) map[string]any {
	identifier = strings.TrimSpace(identifier)
	if identifier == "" {
		identifier = "logging-00000001"
	}
	if strings.HasPrefix(identifier, "arn:") {
		return s.ensureLoggingByARNLocked(identifier)
	}
	for arn, cfg := range s.loggingConfigurations {
		if strings.EqualFold(ivsChatStringAny(cfg, "id"), identifier) || strings.EqualFold(ivsChatStringAny(cfg, "name"), identifier) {
			return s.ensureLoggingByARNLocked(arn)
		}
	}
	return s.ensureLoggingByARNLocked(ivsChatArn("logging-configuration", identifier))
}

func (s *ivsChatStore) ensureLoggingByARNLocked(arn string) map[string]any {
	arn = strings.TrimSpace(arn)
	if arn == "" {
		arn = ivsChatFirstNonEmpty(s.firstLoggingARNLocked(), ivsChatArn("logging-configuration", "logging-00000001"))
	}
	if cfg, ok := s.loggingConfigurations[arn]; ok {
		return cfg
	}
	id := ivsChatResourceID(arn)
	cfg := map[string]any{
		"arn":   arn,
		"id":    id,
		"name":  id,
		"state": "ACTIVE",
		"destinationConfiguration": map[string]any{
			"s3": map[string]any{
				"bucketName": "stackyard-chat-logs",
			},
		},
		"createdTime": time.Now().UTC().Format(time.RFC3339),
		"updatedTime": time.Now().UTC().Format(time.RFC3339),
	}
	s.loggingConfigurations[arn] = cfg
	return cfg
}

func (s *ivsChatStore) ensureTagsLocked(resourceARN string) map[string]string {
	resourceARN = strings.TrimSpace(resourceARN)
	if resourceARN == "" {
		resourceARN = ivsChatFirstNonEmpty(s.firstRoomARNLocked(), ivsChatArn("room", "room-00000001"))
	}
	if tags, ok := s.tags[resourceARN]; ok {
		return tags
	}
	s.tags[resourceARN] = map[string]string{}
	return s.tags[resourceARN]
}

func (s *ivsChatStore) listRoomsLocked() []map[string]any {
	keys := make([]string, 0, len(s.rooms))
	for arn := range s.rooms {
		keys = append(keys, arn)
	}
	sort.Strings(keys)
	out := make([]map[string]any, 0, len(keys))
	for _, arn := range keys {
		out = append(out, ivsChatCloneMap(s.rooms[arn]))
	}
	return out
}

func (s *ivsChatStore) listLoggingConfigurationsLocked() []map[string]any {
	keys := make([]string, 0, len(s.loggingConfigurations))
	for arn := range s.loggingConfigurations {
		keys = append(keys, arn)
	}
	sort.Strings(keys)
	out := make([]map[string]any, 0, len(keys))
	for _, arn := range keys {
		out = append(out, ivsChatCloneMap(s.loggingConfigurations[arn]))
	}
	return out
}

func ivsChatArn(resource, id string) string {
	return fmt.Sprintf(
		"arn:aws:ivs-chat:us-east-1:123456789012:%s/%s",
		strings.TrimSpace(resource),
		strings.TrimSpace(id),
	)
}

func ivsChatResourceID(arn string) string {
	parts := strings.Split(strings.TrimSpace(arn), "/")
	if len(parts) == 0 {
		return ""
	}
	return strings.TrimSpace(parts[len(parts)-1])
}

func ivsChatPath(pathParams map[string]string, key string) string {
	if pathParams == nil {
		return ""
	}
	return strings.TrimSpace(pathParams[key])
}

func ivsChatStringAny(payload map[string]any, keys ...string) string {
	if payload == nil {
		return ""
	}
	for _, key := range keys {
		if key == "" {
			continue
		}
		if value, ok := payload[key]; ok {
			if text := strings.TrimSpace(fmt.Sprintf("%v", value)); text != "" && text != "<nil>" {
				return text
			}
		}
	}
	return ""
}

func ivsChatMapAny(payload map[string]any, key string) map[string]any {
	if payload == nil {
		return map[string]any{}
	}
	if raw, ok := payload[key]; ok {
		if m, ok := raw.(map[string]any); ok {
			return ivsChatCloneMap(m)
		}
	}
	return map[string]any{}
}

func ivsChatSliceAny(payload map[string]any, key string) []any {
	if payload == nil {
		return []any{}
	}
	raw, ok := payload[key]
	if !ok {
		return []any{}
	}
	values, ok := raw.([]any)
	if !ok {
		return []any{}
	}
	out := make([]any, 0, len(values))
	for _, v := range values {
		out = append(out, ivsChatCloneAny(v))
	}
	return out
}

func ivsChatFirstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func ivsChatReadTags(payload map[string]any) map[string]string {
	out := map[string]string{}
	if payload == nil {
		return out
	}

	raw, ok := payload["tags"]
	if !ok {
		raw = payload["Tags"]
	}
	switch typed := raw.(type) {
	case map[string]any:
		for key, value := range typed {
			key = strings.TrimSpace(key)
			valueText := strings.TrimSpace(fmt.Sprintf("%v", value))
			if key != "" && valueText != "" && valueText != "<nil>" {
				out[key] = valueText
			}
		}
	case map[string]string:
		for key, value := range typed {
			key = strings.TrimSpace(key)
			value = strings.TrimSpace(value)
			if key != "" && value != "" {
				out[key] = value
			}
		}
	}
	return out
}

func ivsChatReadTagKeys(payload map[string]any, query url.Values) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0)
	appendKey := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		if _, ok := seen[value]; ok {
			return
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}

	readRawKeys := func(raw any) {
		switch typed := raw.(type) {
		case []any:
			for _, item := range typed {
				appendKey(fmt.Sprintf("%v", item))
			}
		case []string:
			for _, item := range typed {
				appendKey(item)
			}
		case string:
			for _, item := range strings.Split(typed, ",") {
				appendKey(item)
			}
		}
	}

	if payload != nil {
		readRawKeys(payload["tagKeys"])
		readRawKeys(payload["TagKeys"])
	}
	if query != nil {
		for _, raw := range query["tagKeys"] {
			for _, item := range strings.Split(raw, ",") {
				appendKey(item)
			}
		}
	}

	return out
}

func ivsChatCloneMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = ivsChatCloneAny(value)
	}
	return out
}

func ivsChatCloneAny(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return ivsChatCloneMap(typed)
	case []any:
		out := make([]any, 0, len(typed))
		for _, item := range typed {
			out = append(out, ivsChatCloneAny(item))
		}
		return out
	default:
		return typed
	}
}

func ivsChatCloneTags(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
