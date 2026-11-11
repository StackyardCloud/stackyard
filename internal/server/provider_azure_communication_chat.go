package server

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

const azureCommunicationChatAPIVersion = "2025-03-15"
const azureCommunicationChatPrefix = "/azure/communication-chat/2025-03-15"

type azureCommunicationChatThread struct {
	ThreadID         string
	Topic            string
	Metadata         map[string]any
	RetentionPolicy  map[string]any
	CreatedBy        map[string]any
	CreatedAt        time.Time
	UpdatedAt        time.Time
	DeletedAt        *time.Time
	RepeatabilityKey string
}

type azureCommunicationChatParticipant struct {
	ThreadID         string
	RawID            string
	Identifier       map[string]any
	DisplayName      string
	ShareHistoryTime string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type azureCommunicationChatMessage struct {
	ThreadID          string
	MessageID         string
	Type              string
	Content           string
	Metadata          map[string]any
	SenderDisplayName string
	SenderIdentifier  map[string]any
	CreatedAt         time.Time
	EditedAt          *time.Time
	DeletedAt         *time.Time
}

type azureCommunicationChatReadReceipt struct {
	ThreadID         string
	ChatMessageID    string
	SenderIdentifier map[string]any
	CreatedAt        time.Time
}

func (s *Server) handleAzureCommunicationChatRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, azureCommunicationChatPrefix) {
		return false
	}
	if strings.TrimSpace(r.URL.Query().Get("api-version")) != azureCommunicationChatAPIVersion {
		azureCommunicationChatError(w, http.StatusBadRequest, "InvalidApiVersion", "api-version must be 2025-03-15")
		return true
	}

	relative := strings.TrimPrefix(path, azureCommunicationChatPrefix)
	segments := splitPathSegments(strings.Trim(relative, "/"))
	if len(segments) == 0 || !strings.EqualFold(segments[0], "chat") {
		azureCommunicationChatError(w, http.StatusNotFound, "PathNotFound", "route not found")
		return true
	}
	if len(segments) < 2 || !strings.EqualFold(segments[1], "threads") {
		azureCommunicationChatError(w, http.StatusNotFound, "PathNotFound", "route not found")
		return true
	}

	s.handleAzureCommunicationChatThreads(w, r, segments[2:])
	return true
}

func (s *Server) handleAzureCommunicationChatThreads(w http.ResponseWriter, r *http.Request, tail []string) {
	if len(tail) == 0 {
		switch r.Method {
		case http.MethodPost:
			s.handleAzureCommunicationChatCreateThread(w, r)
		case http.MethodGet:
			s.handleAzureCommunicationChatListThreads(w, r)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

	threadID := strings.TrimSpace(tail[0])
	if threadID == "" {
		azureCommunicationChatError(w, http.StatusBadRequest, "InvalidRequest", "chatThreadId is required")
		return
	}

	if len(tail) == 1 {
		switch r.Method {
		case http.MethodGet:
			s.handleAzureCommunicationChatGetThread(w, threadID)
		case http.MethodPatch:
			s.handleAzureCommunicationChatUpdateThread(w, r, threadID)
		case http.MethodDelete:
			s.handleAzureCommunicationChatDeleteThread(w, threadID)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

	switch strings.ToLower(strings.TrimSpace(tail[1])) {
	case "participants":
		s.handleAzureCommunicationChatParticipants(w, r, threadID, tail[2:])
	case "messages":
		s.handleAzureCommunicationChatMessages(w, r, threadID, tail[2:])
	case "readreceipts":
		s.handleAzureCommunicationChatReadReceipts(w, r, threadID, tail[2:])
	case "typing":
		if len(tail) != 2 {
			azureCommunicationChatError(w, http.StatusNotFound, "PathNotFound", "route not found")
			return
		}
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleAzureCommunicationChatSendTyping(w, threadID)
	default:
		azureCommunicationChatError(w, http.StatusNotFound, "PathNotFound", "route not found")
	}
}

func (s *Server) handleAzureCommunicationChatParticipants(w http.ResponseWriter, r *http.Request, threadID string, tail []string) {
	switch {
	case len(tail) == 0 && r.Method == http.MethodGet:
		s.handleAzureCommunicationChatListParticipants(w, r, threadID)
	case len(tail) == 1 && strings.EqualFold(tail[0], ":add") && r.Method == http.MethodPost:
		s.handleAzureCommunicationChatAddParticipants(w, r, threadID)
	case len(tail) == 1 && strings.EqualFold(tail[0], ":remove") && r.Method == http.MethodPost:
		s.handleAzureCommunicationChatRemoveParticipant(w, r, threadID)
	case len(tail) <= 1:
		w.WriteHeader(http.StatusMethodNotAllowed)
	default:
		azureCommunicationChatError(w, http.StatusNotFound, "PathNotFound", "route not found")
	}
}

func (s *Server) handleAzureCommunicationChatMessages(w http.ResponseWriter, r *http.Request, threadID string, tail []string) {
	switch {
	case len(tail) == 0 && r.Method == http.MethodPost:
		s.handleAzureCommunicationChatSendMessage(w, r, threadID)
	case len(tail) == 0 && r.Method == http.MethodGet:
		s.handleAzureCommunicationChatListMessages(w, r, threadID)
	case len(tail) == 1 && r.Method == http.MethodGet:
		s.handleAzureCommunicationChatGetMessage(w, threadID, tail[0])
	case len(tail) == 1 && r.Method == http.MethodPatch:
		s.handleAzureCommunicationChatUpdateMessage(w, r, threadID, tail[0])
	case len(tail) == 1 && r.Method == http.MethodDelete:
		s.handleAzureCommunicationChatDeleteMessage(w, threadID, tail[0])
	case len(tail) <= 1:
		w.WriteHeader(http.StatusMethodNotAllowed)
	default:
		azureCommunicationChatError(w, http.StatusNotFound, "PathNotFound", "route not found")
	}
}

func (s *Server) handleAzureCommunicationChatReadReceipts(w http.ResponseWriter, r *http.Request, threadID string, tail []string) {
	if len(tail) != 0 {
		azureCommunicationChatError(w, http.StatusNotFound, "PathNotFound", "route not found")
		return
	}
	switch r.Method {
	case http.MethodPost:
		s.handleAzureCommunicationChatSendReadReceipt(w, r, threadID)
	case http.MethodGet:
		s.handleAzureCommunicationChatListReadReceipts(w, r, threadID)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleAzureCommunicationChatCreateThread(w http.ResponseWriter, r *http.Request) {
	payload, body, ok := azureCommunicationChatReadObject(w, r)
	if !ok {
		return
	}
	topic := strings.TrimSpace(azureCommunicationChatString(payload["topic"]))
	if topic == "" {
		azureCommunicationChatError(w, http.StatusBadRequest, "InvalidRequest", "topic is required")
		return
	}

	now := time.Now().UTC()
	repeatabilityKey := azureCommunicationChatRepeatabilityKey(r)
	threadID := azureCommunicationChatDeterministicID("thread", string(body), r, now)

	metadata, _ := payload["metadata"].(map[string]any)
	retentionPolicy, _ := payload["retentionPolicy"].(map[string]any)
	participantsInput, _ := payload["participants"].([]any)

	s.providerStorageMu.Lock()
	if repeatabilityKey != "" {
		for _, thread := range s.azureCommunicationChatThreads {
			if thread.RepeatabilityKey == repeatabilityKey {
				fixture := azureCommunicationChatCreateThreadResultFixture(thread, nil)
				s.providerStorageMu.Unlock()
				respondJSON(w, http.StatusCreated, fixture)
				return
			}
		}
	}

	thread := &azureCommunicationChatThread{
		ThreadID:         threadID,
		Topic:            topic,
		Metadata:         azureCommunicationChatCloneObject(metadata),
		RetentionPolicy:  azureCommunicationChatCloneObject(retentionPolicy),
		CreatedBy:        azureCommunicationChatDefaultIdentifier("8:acs:thread-creator"),
		CreatedAt:        now,
		UpdatedAt:        now,
		RepeatabilityKey: repeatabilityKey,
	}
	s.azureCommunicationChatThreads[threadID] = thread

	invalidParticipants := make([]any, 0)
	for _, raw := range participantsInput {
		object, _ := raw.(map[string]any)
		identifierValue := object["communicationIdentifier"]
		if identifierValue == nil {
			invalidParticipants = append(invalidParticipants, map[string]any{
				"rawId":  "",
				"reason": "missing communicationIdentifier",
			})
			continue
		}
		rawID, identifier := azureCommunicationChatNormalizeIdentifier(identifierValue)
		key := azureCommunicationChatParticipantKey(threadID, rawID)
		s.azureCommunicationChatParticipants[key] = &azureCommunicationChatParticipant{
			ThreadID:         threadID,
			RawID:            rawID,
			Identifier:       identifier,
			DisplayName:      strings.TrimSpace(azureCommunicationChatString(object["displayName"])),
			ShareHistoryTime: strings.TrimSpace(azureCommunicationChatString(object["shareHistoryTime"])),
			CreatedAt:        now,
			UpdatedAt:        now,
		}
	}
	fixture := azureCommunicationChatCreateThreadResultFixture(thread, invalidParticipants)
	s.providerStorageMu.Unlock()

	respondJSON(w, http.StatusCreated, fixture)
}

func (s *Server) handleAzureCommunicationChatListThreads(w http.ResponseWriter, r *http.Request) {
	maxPageSize, ok := azureCommunicationChatParsePositiveIntQuery(w, r, "maxPageSize", true)
	if !ok {
		return
	}
	startTime, ok := azureCommunicationChatParseTimeQuery(w, r, "startTime")
	if !ok {
		return
	}

	s.providerStorageMu.Lock()
	items := make([]*azureCommunicationChatThread, 0, len(s.azureCommunicationChatThreads))
	for _, thread := range s.azureCommunicationChatThreads {
		if thread.DeletedAt != nil {
			continue
		}
		if startTime != nil && thread.CreatedAt.Before(*startTime) {
			continue
		}
		items = append(items, thread)
	}
	s.providerStorageMu.Unlock()

	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.Before(items[j].CreatedAt) })
	start := 0
	if raw := strings.TrimSpace(r.URL.Query().Get("skip")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 0 {
			azureCommunicationChatError(w, http.StatusBadRequest, "InvalidRequest", "skip must be a non-negative integer")
			return
		}
		start = value
	}
	if start > len(items) {
		start = len(items)
	}
	items = items[start:]
	pageSize := len(items)
	if maxPageSize > 0 && maxPageSize < pageSize {
		pageSize = maxPageSize
	}

	value := make([]any, 0, pageSize)
	for _, thread := range items[:pageSize] {
		value = append(value, azureCommunicationChatThreadFixture(thread))
	}

	var nextLink any
	if start+pageSize < len(items)+start {
		nextLink = fmt.Sprintf("%s/chat/threads?api-version=%s&skip=%d", azureCommunicationChatPrefix, azureCommunicationChatAPIVersion, start+pageSize)
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"value":    value,
		"nextLink": nextLink,
	})
}

func (s *Server) handleAzureCommunicationChatGetThread(w http.ResponseWriter, threadID string) {
	s.providerStorageMu.Lock()
	thread := s.azureCommunicationChatThreads[threadID]
	s.providerStorageMu.Unlock()
	if thread == nil || thread.DeletedAt != nil {
		azureCommunicationChatError(w, http.StatusNotFound, "ChatThreadNotFound", "chat thread not found")
		return
	}
	respondJSON(w, http.StatusOK, azureCommunicationChatThreadFixture(thread))
}

func (s *Server) handleAzureCommunicationChatUpdateThread(w http.ResponseWriter, r *http.Request, threadID string) {
	payload, _, ok := azureCommunicationChatReadObject(w, r)
	if !ok {
		return
	}

	s.providerStorageMu.Lock()
	thread := s.azureCommunicationChatThreads[threadID]
	if thread == nil || thread.DeletedAt != nil {
		s.providerStorageMu.Unlock()
		azureCommunicationChatError(w, http.StatusNotFound, "ChatThreadNotFound", "chat thread not found")
		return
	}
	if topic := strings.TrimSpace(azureCommunicationChatString(payload["topic"])); topic != "" {
		thread.Topic = topic
	}
	if metadata, ok := payload["metadata"].(map[string]any); ok {
		thread.Metadata = azureCommunicationChatCloneObject(metadata)
	}
	thread.UpdatedAt = time.Now().UTC()
	s.providerStorageMu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAzureCommunicationChatDeleteThread(w http.ResponseWriter, threadID string) {
	s.providerStorageMu.Lock()
	thread := s.azureCommunicationChatThreads[threadID]
	if thread != nil && thread.DeletedAt == nil {
		now := time.Now().UTC()
		thread.DeletedAt = &now
		thread.UpdatedAt = now
	}
	s.providerStorageMu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAzureCommunicationChatAddParticipants(w http.ResponseWriter, r *http.Request, threadID string) {
	payload, _, ok := azureCommunicationChatReadObject(w, r)
	if !ok {
		return
	}
	participants, ok := payload["participants"].([]any)
	if !ok || len(participants) == 0 {
		azureCommunicationChatError(w, http.StatusBadRequest, "InvalidRequest", "participants must contain at least one participant")
		return
	}

	s.providerStorageMu.Lock()
	thread := s.azureCommunicationChatThreads[threadID]
	if thread == nil || thread.DeletedAt != nil {
		s.providerStorageMu.Unlock()
		azureCommunicationChatError(w, http.StatusNotFound, "ChatThreadNotFound", "chat thread not found")
		return
	}
	now := time.Now().UTC()
	invalidParticipants := make([]any, 0)
	for _, raw := range participants {
		object, _ := raw.(map[string]any)
		identifierValue := object["communicationIdentifier"]
		if identifierValue == nil {
			invalidParticipants = append(invalidParticipants, map[string]any{
				"rawId":  "",
				"reason": "missing communicationIdentifier",
			})
			continue
		}
		rawID, identifier := azureCommunicationChatNormalizeIdentifier(identifierValue)
		key := azureCommunicationChatParticipantKey(threadID, rawID)
		if existing := s.azureCommunicationChatParticipants[key]; existing != nil {
			existing.DisplayName = strings.TrimSpace(azureCommunicationChatString(object["displayName"]))
			existing.ShareHistoryTime = strings.TrimSpace(azureCommunicationChatString(object["shareHistoryTime"]))
			existing.UpdatedAt = now
			continue
		}
		s.azureCommunicationChatParticipants[key] = &azureCommunicationChatParticipant{
			ThreadID:         threadID,
			RawID:            rawID,
			Identifier:       identifier,
			DisplayName:      strings.TrimSpace(azureCommunicationChatString(object["displayName"])),
			ShareHistoryTime: strings.TrimSpace(azureCommunicationChatString(object["shareHistoryTime"])),
			CreatedAt:        now,
			UpdatedAt:        now,
		}
	}
	thread.UpdatedAt = now
	s.providerStorageMu.Unlock()

	respondJSON(w, http.StatusCreated, map[string]any{
		"invalidParticipants": invalidParticipants,
	})
}

func (s *Server) handleAzureCommunicationChatRemoveParticipant(w http.ResponseWriter, r *http.Request, threadID string) {
	payload, _, ok := azureCommunicationChatReadObject(w, r)
	if !ok {
		return
	}
	identifierValue := payload["communicationIdentifier"]
	if identifierValue == nil {
		azureCommunicationChatError(w, http.StatusBadRequest, "InvalidRequest", "communicationIdentifier is required")
		return
	}
	rawID, _ := azureCommunicationChatNormalizeIdentifier(identifierValue)

	s.providerStorageMu.Lock()
	thread := s.azureCommunicationChatThreads[threadID]
	if thread == nil || thread.DeletedAt != nil {
		s.providerStorageMu.Unlock()
		azureCommunicationChatError(w, http.StatusNotFound, "ChatThreadNotFound", "chat thread not found")
		return
	}
	delete(s.azureCommunicationChatParticipants, azureCommunicationChatParticipantKey(threadID, rawID))
	thread.UpdatedAt = time.Now().UTC()
	s.providerStorageMu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAzureCommunicationChatListParticipants(w http.ResponseWriter, r *http.Request, threadID string) {
	maxPageSize, ok := azureCommunicationChatParsePositiveIntQuery(w, r, "maxPageSize", true)
	if !ok {
		return
	}
	skip, ok := azureCommunicationChatParsePositiveIntQuery(w, r, "skip", false)
	if !ok {
		return
	}

	s.providerStorageMu.Lock()
	thread := s.azureCommunicationChatThreads[threadID]
	if thread == nil || thread.DeletedAt != nil {
		s.providerStorageMu.Unlock()
		azureCommunicationChatError(w, http.StatusNotFound, "ChatThreadNotFound", "chat thread not found")
		return
	}
	items := make([]*azureCommunicationChatParticipant, 0)
	for _, participant := range s.azureCommunicationChatParticipants {
		if participant.ThreadID == threadID {
			items = append(items, participant)
		}
	}
	s.providerStorageMu.Unlock()

	sort.Slice(items, func(i, j int) bool { return items[i].RawID < items[j].RawID })
	if skip > len(items) {
		skip = len(items)
	}
	items = items[skip:]
	pageSize := len(items)
	if maxPageSize > 0 && maxPageSize < pageSize {
		pageSize = maxPageSize
	}
	value := make([]any, 0, pageSize)
	for _, participant := range items[:pageSize] {
		value = append(value, azureCommunicationChatParticipantFixture(participant))
	}

	var nextLink any
	if skip+pageSize < len(items)+skip {
		nextLink = fmt.Sprintf("%s/chat/threads/%s/participants?api-version=%s&skip=%d", azureCommunicationChatPrefix, threadID, azureCommunicationChatAPIVersion, skip+pageSize)
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"value":    value,
		"nextLink": nextLink,
	})
}

func (s *Server) handleAzureCommunicationChatSendMessage(w http.ResponseWriter, r *http.Request, threadID string) {
	payload, body, ok := azureCommunicationChatReadObject(w, r)
	if !ok {
		return
	}
	content := strings.TrimSpace(azureCommunicationChatString(payload["content"]))
	if content == "" {
		azureCommunicationChatError(w, http.StatusBadRequest, "InvalidRequest", "content is required")
		return
	}

	s.providerStorageMu.Lock()
	thread := s.azureCommunicationChatThreads[threadID]
	if thread == nil || thread.DeletedAt != nil {
		s.providerStorageMu.Unlock()
		azureCommunicationChatError(w, http.StatusNotFound, "ChatThreadNotFound", "chat thread not found")
		return
	}
	now := time.Now().UTC()
	messageID := azureCommunicationChatDeterministicID("message", threadID+"|"+string(body), r, now)
	metadata, _ := payload["metadata"].(map[string]any)
	message := &azureCommunicationChatMessage{
		ThreadID:          threadID,
		MessageID:         messageID,
		Type:              strings.TrimSpace(azureCommunicationChatString(payload["type"])),
		Content:           content,
		Metadata:          azureCommunicationChatCloneObject(metadata),
		SenderDisplayName: strings.TrimSpace(azureCommunicationChatString(payload["senderDisplayName"])),
		SenderIdentifier:  azureCommunicationChatDefaultIdentifier("8:acs:stackyard-user"),
		CreatedAt:         now,
	}
	if message.Type == "" {
		message.Type = "text"
	}
	s.azureCommunicationChatMessages[azureCommunicationChatMessageKey(threadID, messageID)] = message
	thread.UpdatedAt = now
	s.providerStorageMu.Unlock()

	w.Header().Set("Location", fmt.Sprintf("%s/chat/threads/%s/messages/%s?api-version=%s", azureCommunicationChatPrefix, threadID, messageID, azureCommunicationChatAPIVersion))
	respondJSON(w, http.StatusCreated, map[string]any{
		"id":        messageID,
		"type":      message.Type,
		"createdOn": now.Format(time.RFC3339Nano),
	})
}

func (s *Server) handleAzureCommunicationChatGetMessage(w http.ResponseWriter, threadID, messageID string) {
	s.providerStorageMu.Lock()
	thread := s.azureCommunicationChatThreads[threadID]
	message := s.azureCommunicationChatMessages[azureCommunicationChatMessageKey(threadID, messageID)]
	s.providerStorageMu.Unlock()
	if thread == nil || thread.DeletedAt != nil {
		azureCommunicationChatError(w, http.StatusNotFound, "ChatThreadNotFound", "chat thread not found")
		return
	}
	if message == nil {
		azureCommunicationChatError(w, http.StatusNotFound, "ChatMessageNotFound", "chat message not found")
		return
	}
	respondJSON(w, http.StatusOK, azureCommunicationChatMessageFixture(message))
}

func (s *Server) handleAzureCommunicationChatListMessages(w http.ResponseWriter, r *http.Request, threadID string) {
	maxPageSize, ok := azureCommunicationChatParsePositiveIntQuery(w, r, "maxPageSize", true)
	if !ok {
		return
	}
	startTime, ok := azureCommunicationChatParseTimeQuery(w, r, "startTime")
	if !ok {
		return
	}

	s.providerStorageMu.Lock()
	thread := s.azureCommunicationChatThreads[threadID]
	if thread == nil || thread.DeletedAt != nil {
		s.providerStorageMu.Unlock()
		azureCommunicationChatError(w, http.StatusNotFound, "ChatThreadNotFound", "chat thread not found")
		return
	}
	items := make([]*azureCommunicationChatMessage, 0)
	for _, message := range s.azureCommunicationChatMessages {
		if message.ThreadID != threadID {
			continue
		}
		if startTime != nil && message.CreatedAt.Before(*startTime) {
			continue
		}
		items = append(items, message)
	}
	s.providerStorageMu.Unlock()

	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.Before(items[j].CreatedAt) })
	pageSize := len(items)
	if maxPageSize > 0 && maxPageSize < pageSize {
		pageSize = maxPageSize
	}
	value := make([]any, 0, pageSize)
	for _, message := range items[:pageSize] {
		value = append(value, azureCommunicationChatMessageFixture(message))
	}
	var nextLink any
	if pageSize < len(items) {
		nextLink = fmt.Sprintf("%s/chat/threads/%s/messages?api-version=%s&skip=%d", azureCommunicationChatPrefix, threadID, azureCommunicationChatAPIVersion, pageSize)
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"value":    value,
		"nextLink": nextLink,
	})
}

func (s *Server) handleAzureCommunicationChatUpdateMessage(w http.ResponseWriter, r *http.Request, threadID, messageID string) {
	payload, _, ok := azureCommunicationChatReadObject(w, r)
	if !ok {
		return
	}
	if payload["content"] == nil && payload["metadata"] == nil {
		azureCommunicationChatError(w, http.StatusBadRequest, "InvalidRequest", "content or metadata is required")
		return
	}

	s.providerStorageMu.Lock()
	thread := s.azureCommunicationChatThreads[threadID]
	message := s.azureCommunicationChatMessages[azureCommunicationChatMessageKey(threadID, messageID)]
	if thread == nil || thread.DeletedAt != nil {
		s.providerStorageMu.Unlock()
		azureCommunicationChatError(w, http.StatusNotFound, "ChatThreadNotFound", "chat thread not found")
		return
	}
	if message == nil {
		s.providerStorageMu.Unlock()
		azureCommunicationChatError(w, http.StatusNotFound, "ChatMessageNotFound", "chat message not found")
		return
	}
	if content := strings.TrimSpace(azureCommunicationChatString(payload["content"])); content != "" {
		message.Content = content
	}
	if metadata, ok := payload["metadata"].(map[string]any); ok {
		message.Metadata = azureCommunicationChatCloneObject(metadata)
	}
	now := time.Now().UTC()
	message.EditedAt = &now
	thread.UpdatedAt = now
	s.providerStorageMu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAzureCommunicationChatDeleteMessage(w http.ResponseWriter, threadID, messageID string) {
	s.providerStorageMu.Lock()
	thread := s.azureCommunicationChatThreads[threadID]
	message := s.azureCommunicationChatMessages[azureCommunicationChatMessageKey(threadID, messageID)]
	if thread == nil || thread.DeletedAt != nil {
		s.providerStorageMu.Unlock()
		azureCommunicationChatError(w, http.StatusNotFound, "ChatThreadNotFound", "chat thread not found")
		return
	}
	if message == nil {
		s.providerStorageMu.Unlock()
		azureCommunicationChatError(w, http.StatusNotFound, "ChatMessageNotFound", "chat message not found")
		return
	}
	now := time.Now().UTC()
	message.DeletedAt = &now
	thread.UpdatedAt = now
	s.providerStorageMu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAzureCommunicationChatSendReadReceipt(w http.ResponseWriter, r *http.Request, threadID string) {
	payload, _, ok := azureCommunicationChatReadObject(w, r)
	if !ok {
		return
	}
	chatMessageID := strings.TrimSpace(azureCommunicationChatString(payload["chatMessageId"]))
	if chatMessageID == "" {
		azureCommunicationChatError(w, http.StatusBadRequest, "InvalidRequest", "chatMessageId is required")
		return
	}

	s.providerStorageMu.Lock()
	thread := s.azureCommunicationChatThreads[threadID]
	if thread == nil || thread.DeletedAt != nil {
		s.providerStorageMu.Unlock()
		azureCommunicationChatError(w, http.StatusNotFound, "ChatThreadNotFound", "chat thread not found")
		return
	}
	receipt := &azureCommunicationChatReadReceipt{
		ThreadID:         threadID,
		ChatMessageID:    chatMessageID,
		SenderIdentifier: azureCommunicationChatDefaultIdentifier("8:acs:stackyard-user"),
		CreatedAt:        time.Now().UTC(),
	}
	s.azureCommunicationChatReadReceipts = append(s.azureCommunicationChatReadReceipts, receipt)
	thread.UpdatedAt = receipt.CreatedAt
	s.providerStorageMu.Unlock()
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleAzureCommunicationChatListReadReceipts(w http.ResponseWriter, r *http.Request, threadID string) {
	maxPageSize, ok := azureCommunicationChatParsePositiveIntQuery(w, r, "maxPageSize", true)
	if !ok {
		return
	}
	skip, ok := azureCommunicationChatParsePositiveIntQuery(w, r, "skip", false)
	if !ok {
		return
	}

	s.providerStorageMu.Lock()
	thread := s.azureCommunicationChatThreads[threadID]
	if thread == nil || thread.DeletedAt != nil {
		s.providerStorageMu.Unlock()
		azureCommunicationChatError(w, http.StatusNotFound, "ChatThreadNotFound", "chat thread not found")
		return
	}
	items := make([]*azureCommunicationChatReadReceipt, 0)
	for _, receipt := range s.azureCommunicationChatReadReceipts {
		if receipt.ThreadID == threadID {
			items = append(items, receipt)
		}
	}
	s.providerStorageMu.Unlock()

	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.Before(items[j].CreatedAt) })
	if skip > len(items) {
		skip = len(items)
	}
	items = items[skip:]
	pageSize := len(items)
	if maxPageSize > 0 && maxPageSize < pageSize {
		pageSize = maxPageSize
	}
	value := make([]any, 0, pageSize)
	for _, receipt := range items[:pageSize] {
		value = append(value, azureCommunicationChatReadReceiptFixture(receipt))
	}
	var nextLink any
	if skip+pageSize < len(items)+skip {
		nextLink = fmt.Sprintf("%s/chat/threads/%s/readReceipts?api-version=%s&skip=%d", azureCommunicationChatPrefix, threadID, azureCommunicationChatAPIVersion, skip+pageSize)
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"value":    value,
		"nextLink": nextLink,
	})
}

func (s *Server) handleAzureCommunicationChatSendTyping(w http.ResponseWriter, threadID string) {
	s.providerStorageMu.Lock()
	thread := s.azureCommunicationChatThreads[threadID]
	if thread == nil || thread.DeletedAt != nil {
		s.providerStorageMu.Unlock()
		azureCommunicationChatError(w, http.StatusNotFound, "ChatThreadNotFound", "chat thread not found")
		return
	}
	thread.UpdatedAt = time.Now().UTC()
	s.providerStorageMu.Unlock()
	w.WriteHeader(http.StatusOK)
}

func azureCommunicationChatCreateThreadResultFixture(thread *azureCommunicationChatThread, invalidParticipants []any) map[string]any {
	result := map[string]any{
		"chatThread":          azureCommunicationChatThreadFixture(thread),
		"invalidParticipants": invalidParticipants,
	}
	if invalidParticipants == nil {
		result["invalidParticipants"] = []any{}
	}
	return result
}

func azureCommunicationChatThreadFixture(thread *azureCommunicationChatThread) map[string]any {
	fixture := map[string]any{
		"id":                               thread.ThreadID,
		"topic":                            thread.Topic,
		"metadata":                         azureCommunicationChatCloneObject(thread.Metadata),
		"retentionPolicy":                  azureCommunicationChatCloneObject(thread.RetentionPolicy),
		"createdOn":                        thread.CreatedAt.Format(time.RFC3339Nano),
		"createdByCommunicationIdentifier": azureCommunicationChatCloneObject(thread.CreatedBy),
	}
	if thread.DeletedAt != nil {
		fixture["deletedOn"] = thread.DeletedAt.Format(time.RFC3339Nano)
	}
	return fixture
}

func azureCommunicationChatParticipantFixture(participant *azureCommunicationChatParticipant) map[string]any {
	fixture := map[string]any{
		"communicationIdentifier": azureCommunicationChatCloneObject(participant.Identifier),
		"displayName":             participant.DisplayName,
		"shareHistoryTime":        participant.ShareHistoryTime,
	}
	return fixture
}

func azureCommunicationChatMessageFixture(message *azureCommunicationChatMessage) map[string]any {
	fixture := map[string]any{
		"id":                            message.MessageID,
		"type":                          message.Type,
		"content":                       message.Content,
		"metadata":                      azureCommunicationChatCloneObject(message.Metadata),
		"senderDisplayName":             message.SenderDisplayName,
		"senderCommunicationIdentifier": azureCommunicationChatCloneObject(message.SenderIdentifier),
		"createdOn":                     message.CreatedAt.Format(time.RFC3339Nano),
	}
	if message.EditedAt != nil {
		fixture["editedOn"] = message.EditedAt.Format(time.RFC3339Nano)
	}
	if message.DeletedAt != nil {
		fixture["deletedOn"] = message.DeletedAt.Format(time.RFC3339Nano)
	}
	return fixture
}

func azureCommunicationChatReadReceiptFixture(receipt *azureCommunicationChatReadReceipt) map[string]any {
	return map[string]any{
		"chatMessageId":                 receipt.ChatMessageID,
		"senderCommunicationIdentifier": azureCommunicationChatCloneObject(receipt.SenderIdentifier),
		"readOn":                        receipt.CreatedAt.Format(time.RFC3339Nano),
	}
}

func azureCommunicationChatReadObject(w http.ResponseWriter, r *http.Request) (map[string]any, []byte, bool) {
	body, err := readBodyBytes(r)
	if err != nil {
		azureCommunicationChatError(w, http.StatusBadRequest, "InvalidRequest", "unable to read request body")
		return nil, nil, false
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		azureCommunicationChatError(w, http.StatusBadRequest, "InvalidRequest", "request body must be valid JSON")
		return nil, nil, false
	}
	if payload == nil {
		payload = map[string]any{}
	}
	return payload, body, true
}

func azureCommunicationChatParsePositiveIntQuery(w http.ResponseWriter, r *http.Request, key string, allowEmpty bool) (int, bool) {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		if allowEmpty {
			return 0, true
		}
		return 0, true
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 0 || (key == "maxPageSize" && value == 0) {
		azureCommunicationChatError(w, http.StatusBadRequest, "InvalidRequest", key+" must be a positive integer")
		return 0, false
	}
	return value, true
}

func azureCommunicationChatParseTimeQuery(w http.ResponseWriter, r *http.Request, key string) (*time.Time, bool) {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return nil, true
	}
	value, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		azureCommunicationChatError(w, http.StatusBadRequest, "InvalidRequest", key+" must be a valid RFC3339 timestamp")
		return nil, false
	}
	return &value, true
}

func azureCommunicationChatParticipantKey(threadID, rawID string) string {
	return threadID + "|" + rawID
}

func azureCommunicationChatMessageKey(threadID, messageID string) string {
	return threadID + "|" + messageID
}

func azureCommunicationChatRepeatabilityKey(r *http.Request) string {
	requestID := strings.TrimSpace(r.Header.Get("Repeatability-Request-ID"))
	if requestID == "" {
		requestID = strings.TrimSpace(r.Header.Get("repeatability-request-id"))
	}
	firstSent := strings.TrimSpace(r.Header.Get("Repeatability-First-Sent"))
	if firstSent == "" {
		firstSent = strings.TrimSpace(r.Header.Get("repeatability-first-sent"))
	}
	if requestID == "" || firstSent == "" {
		return ""
	}
	return requestID + "|" + firstSent
}

func azureCommunicationChatDeterministicID(prefix, seed string, r *http.Request, now time.Time) string {
	material := prefix + "|" + seed
	if repeatabilityKey := azureCommunicationChatRepeatabilityKey(r); repeatabilityKey != "" {
		material += "|" + repeatabilityKey
	} else {
		material += "|" + now.Format(time.RFC3339Nano)
	}
	return prefix + "-" + azureCommunicationChatDigest(material)[:18]
}

func azureCommunicationChatNormalizeIdentifier(value any) (string, map[string]any) {
	if value == nil {
		return "", nil
	}
	switch typed := value.(type) {
	case string:
		rawID := strings.TrimSpace(typed)
		return rawID, map[string]any{"rawId": rawID}
	case map[string]any:
		if rawID := strings.TrimSpace(azureCommunicationChatString(typed["rawId"])); rawID != "" {
			return rawID, azureCommunicationChatCloneObject(typed)
		}
		if id := strings.TrimSpace(azureCommunicationChatString(typed["id"])); id != "" {
			clone := azureCommunicationChatCloneObject(typed)
			if clone["rawId"] == nil {
				clone["rawId"] = id
			}
			return id, clone
		}
		for _, key := range []string{"communicationUser", "phoneNumber", "microsoftTeamsUser", "microsoftTeamsApp", "unknownIdentifier"} {
			nested, _ := typed[key].(map[string]any)
			if len(nested) == 0 {
				continue
			}
			if id := strings.TrimSpace(azureCommunicationChatString(nested["id"])); id != "" {
				return id, azureCommunicationChatCloneObject(typed)
			}
			if phone := strings.TrimSpace(azureCommunicationChatString(nested["phoneNumber"])); phone != "" {
				return phone, azureCommunicationChatCloneObject(typed)
			}
			if rawID := strings.TrimSpace(azureCommunicationChatString(nested["rawId"])); rawID != "" {
				return rawID, azureCommunicationChatCloneObject(typed)
			}
		}
		rawID := "participant-" + azureCommunicationChatDigest(fmt.Sprintf("%v", typed))[:12]
		clone := azureCommunicationChatCloneObject(typed)
		clone["rawId"] = rawID
		return rawID, clone
	default:
		rawID := "participant-" + azureCommunicationChatDigest(fmt.Sprintf("%v", value))[:12]
		return rawID, map[string]any{"rawId": rawID}
	}
}

func azureCommunicationChatDefaultIdentifier(rawID string) map[string]any {
	return map[string]any{
		"rawId": rawID,
	}
}

func azureCommunicationChatCloneObject(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	raw, err := json.Marshal(in)
	if err != nil {
		out := make(map[string]any, len(in))
		for key, value := range in {
			out[key] = value
		}
		return out
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		out = make(map[string]any, len(in))
		for key, value := range in {
			out[key] = value
		}
	}
	return out
}

func azureCommunicationChatString(value any) string {
	if value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	default:
		return fmt.Sprintf("%v", value)
	}
}

func azureCommunicationChatDigest(input string) string {
	sum := sha1.Sum([]byte(input))
	return hex.EncodeToString(sum[:])
}

func azureCommunicationChatError(w http.ResponseWriter, status int, code, message string) {
	respondJSON(w, status, map[string]any{
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
	})
}
