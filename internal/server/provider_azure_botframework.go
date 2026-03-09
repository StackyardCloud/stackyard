package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type azureBotConversation struct {
	ID             string
	Members        []azureBotMember
	Activities     map[string]map[string]any
	ActivityOrder  []string
	NextActivityID int64
	CreatedAt      time.Time
}

type azureBotMember struct {
	ID   string
	Name string
	Role string
}

func (s *Server) handleAzureBotFrameworkRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, "/azure/botframework/") {
		return false
	}

	relative := "/" + strings.TrimPrefix(path, "/azure/botframework/")
	segments := splitPathSegments(strings.TrimPrefix(relative, "/"))
	if len(segments) < 2 || segments[0] != "v3" || segments[1] != "conversations" {
		respondAzureImplemented(w, path)
		return true
	}

	if len(segments) == 2 && r.Method == http.MethodPost {
		s.handleAzureBotFrameworkCreateConversation(w, r, "")
		return true
	}
	if len(segments) == 3 && r.Method == http.MethodPost {
		s.handleAzureBotFrameworkCreateConversation(w, r, segments[2])
		return true
	}
	if len(segments) == 4 && segments[3] == "activities" && r.Method == http.MethodPost {
		s.handleAzureBotFrameworkCreateActivity(w, r, segments[2], "")
		return true
	}
	if len(segments) == 5 && segments[3] == "activities" {
		conversationID := segments[2]
		activityID := segments[4]
		switch r.Method {
		case http.MethodPost:
			s.handleAzureBotFrameworkCreateActivity(w, r, conversationID, activityID)
		case http.MethodPut:
			s.handleAzureBotFrameworkUpdateActivity(w, r, conversationID, activityID)
		case http.MethodDelete:
			s.handleAzureBotFrameworkDeleteActivity(w, conversationID, activityID)
		default:
			respondAzureImplemented(w, path)
		}
		return true
	}
	if len(segments) == 4 && segments[3] == "members" && r.Method == http.MethodGet {
		s.handleAzureBotFrameworkListMembers(w, segments[2])
		return true
	}
	if len(segments) == 4 && segments[3] == "pagedmembers" && r.Method == http.MethodGet {
		s.handleAzureBotFrameworkListPagedMembers(w, r, segments[2], path)
		return true
	}
	if len(segments) == 6 && segments[3] == "activities" && segments[5] == "members" && r.Method == http.MethodGet {
		s.handleAzureBotFrameworkListActivityMembers(w, segments[2], segments[4])
		return true
	}

	respondAzureImplemented(w, path)
	return true
}

func (s *Server) handleAzureBotFrameworkCreateConversation(w http.ResponseWriter, r *http.Request, requestedConversationID string) {
	payload, ok := decodeAzureBotFrameworkJSONBody(w, r)
	if !ok {
		return
	}

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	conversationID := strings.TrimSpace(requestedConversationID)
	if conversationID == "" {
		conversationID = s.nextAzureBotConversationIDLocked()
	}
	if _, exists := s.azureBotConversations[conversationID]; exists {
		respondJSON(w, http.StatusConflict, map[string]any{"error": "ConversationAlreadyExists", "message": "conversation already exists"})
		return
	}

	conversation := &azureBotConversation{
		ID:            conversationID,
		Members:       azureBotMembersFromPayload(payload),
		Activities:    map[string]map[string]any{},
		ActivityOrder: []string{},
		CreatedAt:     time.Now().UTC(),
	}

	initialActivityID := ""
	if activity, ok := payload["activity"].(map[string]any); ok {
		initialActivityID = s.azureBotAppendActivityLocked(conversation, activity, "")
	}

	s.azureBotConversations[conversationID] = conversation

	response := map[string]any{
		"id":         conversationID,
		"serviceUrl": "https://stackyard.local/azure/botframework",
	}
	if initialActivityID != "" {
		response["activityId"] = initialActivityID
	}
	respondJSON(w, http.StatusCreated, response)
}

func (s *Server) handleAzureBotFrameworkCreateActivity(w http.ResponseWriter, r *http.Request, conversationID, replyToActivityID string) {
	payload, ok := decodeAzureBotFrameworkJSONBody(w, r)
	if !ok {
		return
	}
	if !hasAzureBotActivityContent(payload) {
		respondJSON(w, http.StatusBadRequest, map[string]any{"error": "InvalidRequest", "message": "activity payload must include text or type"})
		return
	}

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	conversation := s.azureBotConversations[conversationID]
	if conversation == nil {
		respondJSON(w, http.StatusNotFound, map[string]any{"error": "ConversationNotFound", "message": "conversation not found"})
		return
	}
	if replyToActivityID != "" {
		if _, exists := conversation.Activities[replyToActivityID]; !exists {
			respondJSON(w, http.StatusNotFound, map[string]any{"error": "ActivityNotFound", "message": "reply target activity not found"})
			return
		}
	}

	activityID := s.azureBotAppendActivityLocked(conversation, payload, replyToActivityID)
	respondJSON(w, http.StatusOK, map[string]any{"id": activityID})
}

func (s *Server) handleAzureBotFrameworkUpdateActivity(w http.ResponseWriter, r *http.Request, conversationID, activityID string) {
	payload, ok := decodeAzureBotFrameworkJSONBody(w, r)
	if !ok {
		return
	}
	if !hasAzureBotActivityContent(payload) {
		respondJSON(w, http.StatusBadRequest, map[string]any{"error": "InvalidRequest", "message": "activity payload must include text or type"})
		return
	}

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	conversation := s.azureBotConversations[conversationID]
	if conversation == nil {
		respondJSON(w, http.StatusNotFound, map[string]any{"error": "ConversationNotFound", "message": "conversation not found"})
		return
	}
	if _, exists := conversation.Activities[activityID]; !exists {
		respondJSON(w, http.StatusNotFound, map[string]any{"error": "ActivityNotFound", "message": "activity not found"})
		return
	}

	conversation.Activities[activityID] = azureBotNormalizeActivityPayload(conversationID, activityID, payload, "")
	respondJSON(w, http.StatusOK, map[string]any{"id": activityID})
}

func (s *Server) handleAzureBotFrameworkDeleteActivity(w http.ResponseWriter, conversationID, activityID string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	conversation := s.azureBotConversations[conversationID]
	if conversation == nil {
		respondJSON(w, http.StatusNotFound, map[string]any{"error": "ConversationNotFound", "message": "conversation not found"})
		return
	}
	if _, exists := conversation.Activities[activityID]; !exists {
		respondJSON(w, http.StatusNotFound, map[string]any{"error": "ActivityNotFound", "message": "activity not found"})
		return
	}

	delete(conversation.Activities, activityID)
	for i, id := range conversation.ActivityOrder {
		if id == activityID {
			conversation.ActivityOrder = append(conversation.ActivityOrder[:i], conversation.ActivityOrder[i+1:]...)
			break
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAzureBotFrameworkListMembers(w http.ResponseWriter, conversationID string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	conversation := s.azureBotConversations[conversationID]
	if conversation == nil {
		respondJSON(w, http.StatusNotFound, map[string]any{"error": "ConversationNotFound", "message": "conversation not found"})
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{"members": azureBotMembersToMap(conversation.Members)})
}

func (s *Server) handleAzureBotFrameworkListPagedMembers(w http.ResponseWriter, r *http.Request, conversationID, path string) {
	s.providerStorageMu.Lock()
	conversation := s.azureBotConversations[conversationID]
	s.providerStorageMu.Unlock()
	if conversation == nil {
		respondJSON(w, http.StatusNotFound, map[string]any{"error": "ConversationNotFound", "message": "conversation not found"})
		return
	}

	pageSize := 100
	if raw := strings.TrimSpace(r.URL.Query().Get("pageSize")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value <= 0 {
			respondJSON(w, http.StatusBadRequest, map[string]any{"error": "InvalidRequest", "message": "pageSize must be a positive integer", "path": path})
			return
		}
		pageSize = value
	}

	start := 0
	if raw := strings.TrimSpace(r.URL.Query().Get("continuationToken")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 0 {
			respondJSON(w, http.StatusBadRequest, map[string]any{"error": "InvalidRequest", "message": "continuationToken must be a non-negative integer", "path": path})
			return
		}
		start = value
	}

	members := azureBotMembersToMap(conversation.Members)
	if start > len(members) {
		respondJSON(w, http.StatusBadRequest, map[string]any{"error": "InvalidRequest", "message": "continuationToken is out of range", "path": path})
		return
	}
	end := start + pageSize
	if end > len(members) {
		end = len(members)
	}
	nextToken := ""
	if end < len(members) {
		nextToken = strconv.Itoa(end)
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"members":           members[start:end],
		"continuationToken": nextToken,
	})
}

func (s *Server) handleAzureBotFrameworkListActivityMembers(w http.ResponseWriter, conversationID, activityID string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	conversation := s.azureBotConversations[conversationID]
	if conversation == nil {
		respondJSON(w, http.StatusNotFound, map[string]any{"error": "ConversationNotFound", "message": "conversation not found"})
		return
	}
	if _, exists := conversation.Activities[activityID]; !exists {
		respondJSON(w, http.StatusNotFound, map[string]any{"error": "ActivityNotFound", "message": "activity not found"})
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{"members": azureBotMembersToMap(conversation.Members)})
}

func decodeAzureBotFrameworkJSONBody(w http.ResponseWriter, r *http.Request) (map[string]any, bool) {
	body, err := readBodyBytes(r)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]any{"error": "InvalidRequest", "message": "unable to read request body"})
		return nil, false
	}
	if len(strings.TrimSpace(string(body))) == 0 {
		return map[string]any{}, true
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]any{"error": "InvalidRequest", "message": "request body must be valid JSON"})
		return nil, false
	}
	if payload == nil {
		payload = map[string]any{}
	}
	return payload, true
}

func hasAzureBotActivityContent(payload map[string]any) bool {
	if payload == nil {
		return false
	}
	if strings.TrimSpace(azureBotString(payload["text"])) != "" {
		return true
	}
	if strings.TrimSpace(azureBotString(payload["type"])) != "" {
		return true
	}
	return false
}

func azureBotMembersFromPayload(payload map[string]any) []azureBotMember {
	seen := map[string]struct{}{}
	members := make([]azureBotMember, 0, 4)
	appendMember := func(id, name, role string) {
		id = strings.TrimSpace(id)
		if id == "" {
			return
		}
		if _, exists := seen[id]; exists {
			return
		}
		if strings.TrimSpace(name) == "" {
			name = id
		}
		if strings.TrimSpace(role) == "" {
			role = "user"
		}
		members = append(members, azureBotMember{ID: id, Name: name, Role: role})
		seen[id] = struct{}{}
	}

	if botRaw, ok := payload["bot"].(map[string]any); ok {
		appendMember(azureBotString(botRaw["id"]), azureBotString(botRaw["name"]), "bot")
	}
	if rawMembers, ok := payload["members"].([]any); ok {
		for _, raw := range rawMembers {
			member, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			appendMember(
				azureBotString(member["id"]),
				azureBotString(member["name"]),
				azureBotString(member["role"]),
			)
		}
	}

	if len(members) == 0 {
		appendMember("bot-1", "Stackyard Bot", "bot")
		appendMember("user-1", "Stackyard User", "user")
	}
	return members
}

func azureBotMembersToMap(members []azureBotMember) []map[string]any {
	out := make([]map[string]any, 0, len(members))
	for _, member := range members {
		out = append(out, map[string]any{
			"id":   member.ID,
			"name": member.Name,
			"role": member.Role,
		})
	}
	return out
}

func (s *Server) nextAzureBotConversationIDLocked() string {
	s.azureBotNextConversationID++
	return fmt.Sprintf("conversation-%d", s.azureBotNextConversationID)
}

func (s *Server) azureBotAppendActivityLocked(conversation *azureBotConversation, payload map[string]any, replyToActivityID string) string {
	conversation.NextActivityID++
	activityID := fmt.Sprintf("activity-%d", conversation.NextActivityID)
	conversation.Activities[activityID] = azureBotNormalizeActivityPayload(conversation.ID, activityID, payload, replyToActivityID)
	conversation.ActivityOrder = append(conversation.ActivityOrder, activityID)
	return activityID
}

func azureBotNormalizeActivityPayload(conversationID, activityID string, payload map[string]any, replyToActivityID string) map[string]any {
	out := map[string]any{}
	for key, value := range payload {
		out[key] = value
	}
	if strings.TrimSpace(azureBotString(out["type"])) == "" {
		out["type"] = "message"
	}
	if strings.TrimSpace(azureBotString(out["timestamp"])) == "" {
		out["timestamp"] = time.Now().UTC().Format(time.RFC3339Nano)
	}
	out["id"] = activityID
	out["conversation"] = map[string]any{"id": conversationID}
	if strings.TrimSpace(replyToActivityID) != "" {
		out["replyToId"] = replyToActivityID
	}
	return out
}

func azureBotString(value any) string {
	text, _ := value.(string)
	return strings.TrimSpace(text)
}
