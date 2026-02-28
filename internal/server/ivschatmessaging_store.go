package server

import (
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"
)

type ivsChatMessagingStore struct {
	mu sync.Mutex

	nextMessageID int64
	nextEventID   int64

	rooms map[string]map[string]any
}

func newIVSChatMessagingStore() *ivsChatMessagingStore {
	s := &ivsChatMessagingStore{
		nextMessageID: 2,
		nextEventID:   2,
		rooms:         map[string]map[string]any{},
	}

	s.rooms["room-00000001"] = map[string]any{
		"roomIdentifier": "room-00000001",
		"messages": []any{
			map[string]any{
				"id":       "msg-00000001",
				"content":  "seed message",
				"sendTime": time.Now().UTC().Format(time.RFC3339),
			},
		},
		"events": []any{},
	}
	return s
}

func (s *ivsChatMessagingStore) Handle(action string, payload map[string]any, _ map[string]string, _ url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	roomIdentifier := ivsChatMessagingFirstNonEmpty(
		ivsChatMessagingStringAny(payload, "roomIdentifier", "RoomIdentifier"),
		"room-00000001",
	)
	room := s.ensureRoomLocked(roomIdentifier)
	now := time.Now().UTC().Format(time.RFC3339)

	switch action {
	case "SendMessage":
		messageID := fmt.Sprintf("msg-%08d", s.nextMessageIDLocked())
		content := ivsChatMessagingFirstNonEmpty(ivsChatMessagingStringAny(payload, "content", "Content"), "stackyard message")
		attrs := ivsChatMessagingMapAny(payload, "attributes", "Attributes")
		messages := ivsChatMessagingSliceAny(room, "messages")
		messages = append(messages, map[string]any{
			"id":         messageID,
			"content":    content,
			"attributes": attrs,
			"sender":     ivsChatMessagingFirstNonEmpty(ivsChatMessagingStringAny(payload, "senderId", "SenderId"), "stackyard-user"),
			"sendTime":   now,
		})
		room["messages"] = messages
		return map[string]any{
			"id":       messageID,
			"sendTime": now,
		}

	case "DeleteMessage":
		messageID := ivsChatMessagingFirstNonEmpty(ivsChatMessagingStringAny(payload, "id", "messageId"), "msg-00000001")
		room["messages"] = s.deleteMessageLocked(room, messageID)
		room["lastDeletedMessageId"] = messageID
		room["lastUpdatedTime"] = now
		return map[string]any{}

	case "DisconnectUser":
		userID := ivsChatMessagingFirstNonEmpty(ivsChatMessagingStringAny(payload, "userId", "UserId"), "stackyard-user")
		eventID := fmt.Sprintf("evt-%08d", s.nextEventIDLocked())
		events := ivsChatMessagingSliceAny(room, "events")
		events = append(events, map[string]any{
			"id":        eventID,
			"type":      "DISCONNECT_USER",
			"userId":    userID,
			"eventTime": now,
		})
		room["events"] = events
		room["lastDisconnectedUserId"] = userID
		room["lastUpdatedTime"] = now
		return map[string]any{}
	}

	return map[string]any{}
}

func (s *ivsChatMessagingStore) nextMessageIDLocked() int64 {
	id := s.nextMessageID
	s.nextMessageID++
	return id
}

func (s *ivsChatMessagingStore) nextEventIDLocked() int64 {
	id := s.nextEventID
	s.nextEventID++
	return id
}

func (s *ivsChatMessagingStore) ensureRoomLocked(roomIdentifier string) map[string]any {
	roomIdentifier = strings.TrimSpace(roomIdentifier)
	if roomIdentifier == "" {
		roomIdentifier = "room-00000001"
	}
	if room, ok := s.rooms[roomIdentifier]; ok {
		return room
	}
	room := map[string]any{
		"roomIdentifier": roomIdentifier,
		"messages":       []any{},
		"events":         []any{},
	}
	s.rooms[roomIdentifier] = room
	return room
}

func (s *ivsChatMessagingStore) deleteMessageLocked(room map[string]any, messageID string) []any {
	items := ivsChatMessagingSliceAny(room, "messages")
	out := make([]any, 0, len(items))
	for _, raw := range items {
		msg, ok := raw.(map[string]any)
		if !ok {
			out = append(out, raw)
			continue
		}
		if ivsChatMessagingStringAny(msg, "id") == messageID {
			continue
		}
		out = append(out, ivsChatMessagingCloneAny(raw))
	}
	return out
}

func ivsChatMessagingStringAny(payload map[string]any, keys ...string) string {
	if payload == nil {
		return ""
	}
	for _, key := range keys {
		if key == "" {
			continue
		}
		if raw, ok := payload[key]; ok {
			value := strings.TrimSpace(fmt.Sprintf("%v", raw))
			if value != "" && value != "<nil>" {
				return value
			}
		}
	}
	return ""
}

func ivsChatMessagingFirstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func ivsChatMessagingMapAny(payload map[string]any, keys ...string) map[string]any {
	if payload == nil {
		return map[string]any{}
	}
	for _, key := range keys {
		if raw, ok := payload[key]; ok {
			if m, ok := raw.(map[string]any); ok {
				return ivsChatMessagingCloneMap(m)
			}
		}
	}
	return map[string]any{}
}

func ivsChatMessagingSliceAny(payload map[string]any, key string) []any {
	if payload == nil {
		return []any{}
	}
	raw, ok := payload[key]
	if !ok {
		return []any{}
	}
	items, ok := raw.([]any)
	if !ok {
		return []any{}
	}
	out := make([]any, 0, len(items))
	for _, item := range items {
		out = append(out, ivsChatMessagingCloneAny(item))
	}
	return out
}

func ivsChatMessagingCloneMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = ivsChatMessagingCloneAny(value)
	}
	return out
}

func ivsChatMessagingCloneAny(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return ivsChatMessagingCloneMap(typed)
	case []any:
		out := make([]any, 0, len(typed))
		for _, item := range typed {
			out = append(out, ivsChatMessagingCloneAny(item))
		}
		return out
	default:
		return typed
	}
}
