package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

func (s *Server) handleGCPChatRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if strings.HasPrefix(path, "/gcp/google.chat.v1.ChatService/") {
		switch r.Method {
		case http.MethodGet, http.MethodPost, http.MethodPatch, http.MethodDelete:
			respondProviderNotImplemented(w, providerGCP, path)
			return true
		default:
			return false
		}
	}
	if !isGCPChatPath(path) {
		return false
	}

	switch r.Method {
	case http.MethodGet:
		if handleGCPChatListSpaces(w, r, path) {
			return true
		}
		if handleGCPChatGetSpace(w, path) {
			return true
		}
		if handleGCPChatListMessages(w, r, path) {
			return true
		}
		if handleGCPChatGetMessage(w, path) {
			return true
		}
		if handleGCPChatListMemberships(w, r, path) {
			return true
		}
		if handleGCPChatGetMembership(w, path) {
			return true
		}
		if handleGCPChatListReactions(w, r, path) {
			return true
		}
		if handleGCPChatGetSpaceReadState(w, path) {
			return true
		}
		if handleGCPChatGetThreadReadState(w, path) {
			return true
		}
		if handleGCPChatGetSpaceNotificationSetting(w, path) {
			return true
		}
		if handleGCPChatListSpaceEvents(w, r, path) {
			return true
		}
		if handleGCPChatGetSpaceEvent(w, path) {
			return true
		}
		if handleGCPChatListCustomEmojis(w, r, path) {
			return true
		}
		if handleGCPChatGetCustomEmoji(w, path) {
			return true
		}
		if handleGCPChatGetAttachment(w, path) {
			return true
		}
		if handleGCPChatFindDirectMessage(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost:
		if handleGCPChatCreateMessage(w, r, path) {
			return true
		}
		if handleGCPChatCreateReaction(w, r, path) {
			return true
		}
		if handleGCPChatUploadAttachment(w, r, path) {
			return true
		}
		if handleGCPChatSearchSpaces(w, r, path) {
			return true
		}
		if handleGCPChatSetUpSpace(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPatch:
		if handleGCPChatUpdateMessage(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodDelete:
		if handleGCPChatDeleteReaction(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func isGCPChatPath(path string) bool {
	normalized := normalizeGCPChatPath(path)
	if normalized == "/gcp/v1/spaces" || strings.HasPrefix(normalized, "/gcp/v1/spaces/") {
		return true
	}
	if normalized == "/gcp/v1/spaces:search" || normalized == "/gcp/v1/spaces:setup" || normalized == "/gcp/v1/spaces:findDirectMessage" {
		return true
	}
	if normalized == "/gcp/v1/customEmojis" || strings.HasPrefix(normalized, "/gcp/v1/customEmojis/") {
		return true
	}
	if strings.HasPrefix(normalized, "/gcp/v1/users/") {
		return strings.Contains(normalized, "/spaceReadState") ||
			strings.Contains(normalized, "/threadReadState") ||
			strings.Contains(normalized, "/spaceNotificationSetting")
	}
	return strings.Contains(normalized, "/messages") ||
		strings.Contains(normalized, "/members") ||
		strings.Contains(normalized, "/reactions") ||
		strings.Contains(normalized, "/spaceEvents") ||
		strings.Contains(normalized, "/attachments") ||
		strings.Contains(normalized, ":completeImport")
}

func handleGCPChatListSpaces(w http.ResponseWriter, r *http.Request, path string) bool {
	if normalizeGCPChatPath(path) != "/gcp/v1/spaces" {
		return false
	}
	pageSize, start, valid := parseGCPChatPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpChatSpace("team-space")}
	return respondGCPChatList(w, "spaces", items, pageSize, start, path)
}

func handleGCPChatGetSpace(w http.ResponseWriter, path string) bool {
	spaceID, ok := parseGCPChatSpaceID(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpChatSpace(spaceID))
	return true
}

func handleGCPChatCreateMessage(w http.ResponseWriter, r *http.Request, path string) bool {
	spaceID, ok := parseGCPChatMessagesCollectionSpaceID(path)
	if !ok {
		return false
	}
	body, valid := decodeGCPChatJSONBody(w, r, path)
	if !valid {
		return true
	}
	message := gcpChatBodyMap(body, "message")
	text := strings.TrimSpace(gcpChatString(message, "text"))
	if text == "" {
		respondGCPChatInvalidArgument(w, path, "message.text is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpChatMessage(spaceID, "message-1", text))
	return true
}

func handleGCPChatListMessages(w http.ResponseWriter, r *http.Request, path string) bool {
	spaceID, ok := parseGCPChatMessagesCollectionSpaceID(path)
	if !ok {
		return false
	}
	pageSize, start, valid := parseGCPChatPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpChatMessage(spaceID, "message-1", "hello from stackyard")}
	return respondGCPChatList(w, "messages", items, pageSize, start, path)
}

func handleGCPChatGetMessage(w http.ResponseWriter, path string) bool {
	spaceID, messageID, ok := parseGCPChatMessagePath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpChatMessage(spaceID, messageID, "hello from stackyard"))
	return true
}

func handleGCPChatUpdateMessage(w http.ResponseWriter, r *http.Request, path string) bool {
	spaceID, messageID, ok := parseGCPChatMessagePath(path)
	if !ok {
		return false
	}
	body, valid := decodeGCPChatJSONBody(w, r, path)
	if !valid {
		return true
	}
	message := gcpChatBodyMap(body, "message")
	text := strings.TrimSpace(gcpChatString(message, "text"))
	if text == "" {
		respondGCPChatInvalidArgument(w, path, "message.text is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpChatMessage(spaceID, messageID, text))
	return true
}

func handleGCPChatListMemberships(w http.ResponseWriter, r *http.Request, path string) bool {
	spaceID, ok := parseGCPChatMembershipsCollectionSpaceID(path)
	if !ok {
		return false
	}
	pageSize, start, valid := parseGCPChatPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpChatMembership(spaceID, "user-1")}
	return respondGCPChatList(w, "memberships", items, pageSize, start, path)
}

func handleGCPChatGetMembership(w http.ResponseWriter, path string) bool {
	spaceID, memberID, ok := parseGCPChatMembershipPath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpChatMembership(spaceID, memberID))
	return true
}

func handleGCPChatListReactions(w http.ResponseWriter, r *http.Request, path string) bool {
	spaceID, messageID, ok := parseGCPChatReactionsCollectionPath(path)
	if !ok {
		return false
	}
	pageSize, start, valid := parseGCPChatPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpChatReaction(spaceID, messageID, "reaction-1")}
	return respondGCPChatList(w, "reactions", items, pageSize, start, path)
}

func handleGCPChatCreateReaction(w http.ResponseWriter, r *http.Request, path string) bool {
	spaceID, messageID, ok := parseGCPChatReactionsCollectionPath(path)
	if !ok {
		return false
	}
	body, valid := decodeGCPChatJSONBody(w, r, path)
	if !valid {
		return true
	}
	reaction := gcpChatBodyMap(body, "reaction")
	if len(reaction) == 0 {
		respondGCPChatInvalidArgument(w, path, "reaction is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpChatReaction(spaceID, messageID, "reaction-1"))
	return true
}

func handleGCPChatDeleteReaction(w http.ResponseWriter, path string) bool {
	_, _, _, ok := parseGCPChatReactionPath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPChatGetSpaceReadState(w http.ResponseWriter, path string) bool {
	userID, spaceID, ok := parseGCPChatSpaceReadStatePath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"name":         fmt.Sprintf("users/%s/spaces/%s/spaceReadState", userID, spaceID),
		"lastReadTime": "2026-01-01T00:00:00Z",
	})
	return true
}

func handleGCPChatGetThreadReadState(w http.ResponseWriter, path string) bool {
	userID, spaceID, threadID, ok := parseGCPChatThreadReadStatePath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"name":         fmt.Sprintf("users/%s/spaces/%s/threads/%s/threadReadState", userID, spaceID, threadID),
		"lastReadTime": "2026-01-01T00:00:00Z",
	})
	return true
}

func handleGCPChatGetSpaceNotificationSetting(w http.ResponseWriter, path string) bool {
	userID, spaceID, ok := parseGCPChatSpaceNotificationSettingPath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"name":                fmt.Sprintf("users/%s/spaces/%s/spaceNotificationSetting", userID, spaceID),
		"notificationSetting": "ALL",
	})
	return true
}

func handleGCPChatListSpaceEvents(w http.ResponseWriter, r *http.Request, path string) bool {
	spaceID, ok := parseGCPChatSpaceEventsCollectionSpaceID(path)
	if !ok {
		return false
	}
	pageSize, start, valid := parseGCPChatPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpChatSpaceEvent(spaceID, "event-1")}
	return respondGCPChatList(w, "spaceEvents", items, pageSize, start, path)
}

func handleGCPChatGetSpaceEvent(w http.ResponseWriter, path string) bool {
	spaceID, eventID, ok := parseGCPChatSpaceEventPath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpChatSpaceEvent(spaceID, eventID))
	return true
}

func handleGCPChatListCustomEmojis(w http.ResponseWriter, r *http.Request, path string) bool {
	if normalizeGCPChatPath(path) != "/gcp/v1/customEmojis" {
		return false
	}
	pageSize, start, valid := parseGCPChatPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpChatCustomEmoji("emoji-1")}
	return respondGCPChatList(w, "customEmojis", items, pageSize, start, path)
}

func handleGCPChatGetCustomEmoji(w http.ResponseWriter, path string) bool {
	emojiID, ok := parseGCPChatCustomEmojiID(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpChatCustomEmoji(emojiID))
	return true
}

func handleGCPChatUploadAttachment(w http.ResponseWriter, r *http.Request, path string) bool {
	spaceID, ok := parseGCPChatAttachmentUploadSpaceID(path)
	if !ok {
		return false
	}
	body, valid := decodeGCPChatJSONBody(w, r, path)
	if !valid {
		return true
	}
	filename := strings.TrimSpace(gcpChatString(body, "filename"))
	if filename == "" {
		filename = strings.TrimSpace(r.URL.Query().Get("filename"))
	}
	if filename == "" {
		respondGCPChatInvalidArgument(w, path, "filename is required")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"attachmentDataRef": map[string]any{
			"resourceName": fmt.Sprintf("spaces/%s/messages/message-1/attachments/attachment-1", spaceID),
		},
		"filename": filename,
	})
	return true
}

func handleGCPChatGetAttachment(w http.ResponseWriter, path string) bool {
	spaceID, messageID, attachmentID, ok := parseGCPChatAttachmentPath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"name":        fmt.Sprintf("spaces/%s/messages/%s/attachments/%s", spaceID, messageID, attachmentID),
		"downloadUri": "https://chat.googleapis.com/download/attachment-1",
	})
	return true
}

func handleGCPChatSearchSpaces(w http.ResponseWriter, r *http.Request, path string) bool {
	if normalizeGCPChatPath(path) != "/gcp/v1/spaces:search" {
		return false
	}
	if _, _, valid := parseGCPChatPagination(w, r, path); !valid {
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"spaces": []any{gcpChatSpace("team-space")},
	})
	return true
}

func handleGCPChatFindDirectMessage(w http.ResponseWriter, r *http.Request, path string) bool {
	if normalizeGCPChatPath(path) != "/gcp/v1/spaces:findDirectMessage" {
		return false
	}
	name := strings.TrimSpace(r.URL.Query().Get("name"))
	if name == "" {
		respondGCPChatInvalidArgument(w, path, "name is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpChatSpace("dm-space"))
	return true
}

func handleGCPChatSetUpSpace(w http.ResponseWriter, r *http.Request, path string) bool {
	if normalizeGCPChatPath(path) != "/gcp/v1/spaces:setup" {
		return false
	}
	body, valid := decodeGCPChatJSONBody(w, r, path)
	if !valid {
		return true
	}
	space := gcpChatBodyMap(body, "space")
	if len(space) == 0 {
		respondGCPChatInvalidArgument(w, path, "space is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpChatSpace("team-space"))
	return true
}

func parseGCPChatPagination(w http.ResponseWriter, r *http.Request, path string) (pageSize, start int, ok bool) {
	pageSize, err := parseOptionalNonNegativeInt(r.URL.Query().Get("pageSize"))
	if err != nil {
		respondGCPChatInvalidArgument(w, path, "pageSize must be a non-negative integer")
		return 0, 0, false
	}
	start = 0
	pageToken := strings.TrimSpace(r.URL.Query().Get("pageToken"))
	if pageToken != "" {
		start, err = strconv.Atoi(pageToken)
		if err != nil || start < 0 {
			respondGCPChatInvalidArgument(w, path, "pageToken must be a non-negative integer offset")
			return 0, 0, false
		}
	}
	return pageSize, start, true
}

func respondGCPChatList(w http.ResponseWriter, key string, items []map[string]any, pageSize, start int, path string) bool {
	if start > len(items) {
		respondGCPChatInvalidArgument(w, path, "pageToken is out of range")
		return false
	}
	end := len(items)
	if pageSize > 0 && start+pageSize < end {
		end = start + pageSize
	}
	nextPageToken := ""
	if end < len(items) {
		nextPageToken = strconv.Itoa(end)
	}
	respondJSON(w, http.StatusOK, map[string]any{
		key:             items[start:end],
		"nextPageToken": nextPageToken,
	})
	return true
}

func decodeGCPChatJSONBody(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	if r.Body == nil {
		return map[string]any{}, true
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	var body map[string]any
	if err := decoder.Decode(&body); err != nil && err.Error() != "EOF" {
		respondGCPChatInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if body == nil {
		body = map[string]any{}
	}
	return body, true
}

func gcpChatBodyMap(body map[string]any, key string) map[string]any {
	nested, _ := body[key].(map[string]any)
	if len(nested) > 0 {
		return nested
	}
	return body
}

func gcpChatString(body map[string]any, key string) string {
	value, _ := body[key].(string)
	return value
}

func normalizeGCPChatPath(path string) string {
	normalized := strings.TrimSpace(path)
	normalized = strings.ReplaceAll(normalized, "%3A", ":")
	normalized = strings.ReplaceAll(normalized, "%3a", ":")
	return normalized
}

func parseGCPChatSpaceID(path string) (string, bool) {
	parts := strings.Split(strings.Trim(normalizeGCPChatPath(path), "/"), "/")
	if len(parts) != 4 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "spaces" || strings.TrimSpace(parts[3]) == "" {
		return "", false
	}
	return strings.TrimSpace(parts[3]), true
}

func parseGCPChatMessagesCollectionSpaceID(path string) (string, bool) {
	parts := strings.Split(strings.Trim(normalizeGCPChatPath(path), "/"), "/")
	if len(parts) != 5 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "spaces" || parts[4] != "messages" || strings.TrimSpace(parts[3]) == "" {
		return "", false
	}
	return strings.TrimSpace(parts[3]), true
}

func parseGCPChatMessagePath(path string) (spaceID, messageID string, ok bool) {
	parts := strings.Split(strings.Trim(normalizeGCPChatPath(path), "/"), "/")
	if len(parts) != 6 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "spaces" || parts[4] != "messages" {
		return "", "", false
	}
	spaceID = strings.TrimSpace(parts[3])
	messageID = strings.TrimSpace(parts[5])
	if spaceID == "" || messageID == "" {
		return "", "", false
	}
	return spaceID, messageID, true
}

func parseGCPChatMembershipsCollectionSpaceID(path string) (string, bool) {
	parts := strings.Split(strings.Trim(normalizeGCPChatPath(path), "/"), "/")
	if len(parts) != 5 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "spaces" || parts[4] != "members" || strings.TrimSpace(parts[3]) == "" {
		return "", false
	}
	return strings.TrimSpace(parts[3]), true
}

func parseGCPChatMembershipPath(path string) (spaceID, memberID string, ok bool) {
	parts := strings.Split(strings.Trim(normalizeGCPChatPath(path), "/"), "/")
	if len(parts) != 6 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "spaces" || parts[4] != "members" {
		return "", "", false
	}
	spaceID = strings.TrimSpace(parts[3])
	memberID = strings.TrimSpace(parts[5])
	if spaceID == "" || memberID == "" {
		return "", "", false
	}
	return spaceID, memberID, true
}

func parseGCPChatReactionsCollectionPath(path string) (spaceID, messageID string, ok bool) {
	parts := strings.Split(strings.Trim(normalizeGCPChatPath(path), "/"), "/")
	if len(parts) != 7 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "spaces" || parts[4] != "messages" || parts[6] != "reactions" {
		return "", "", false
	}
	spaceID = strings.TrimSpace(parts[3])
	messageID = strings.TrimSpace(parts[5])
	if spaceID == "" || messageID == "" {
		return "", "", false
	}
	return spaceID, messageID, true
}

func parseGCPChatReactionPath(path string) (spaceID, messageID, reactionID string, ok bool) {
	parts := strings.Split(strings.Trim(normalizeGCPChatPath(path), "/"), "/")
	if len(parts) != 8 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "spaces" || parts[4] != "messages" || parts[6] != "reactions" {
		return "", "", "", false
	}
	spaceID = strings.TrimSpace(parts[3])
	messageID = strings.TrimSpace(parts[5])
	reactionID = strings.TrimSpace(parts[7])
	if spaceID == "" || messageID == "" || reactionID == "" {
		return "", "", "", false
	}
	return spaceID, messageID, reactionID, true
}

func parseGCPChatSpaceReadStatePath(path string) (userID, spaceID string, ok bool) {
	parts := strings.Split(strings.Trim(normalizeGCPChatPath(path), "/"), "/")
	if len(parts) != 7 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "users" || parts[4] != "spaces" || parts[6] != "spaceReadState" {
		return "", "", false
	}
	userID = strings.TrimSpace(parts[3])
	spaceID = strings.TrimSpace(parts[5])
	if userID == "" || spaceID == "" {
		return "", "", false
	}
	return userID, spaceID, true
}

func parseGCPChatThreadReadStatePath(path string) (userID, spaceID, threadID string, ok bool) {
	parts := strings.Split(strings.Trim(normalizeGCPChatPath(path), "/"), "/")
	if len(parts) != 9 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "users" || parts[4] != "spaces" || parts[6] != "threads" || parts[8] != "threadReadState" {
		return "", "", "", false
	}
	userID = strings.TrimSpace(parts[3])
	spaceID = strings.TrimSpace(parts[5])
	threadID = strings.TrimSpace(parts[7])
	if userID == "" || spaceID == "" || threadID == "" {
		return "", "", "", false
	}
	return userID, spaceID, threadID, true
}

func parseGCPChatSpaceNotificationSettingPath(path string) (userID, spaceID string, ok bool) {
	parts := strings.Split(strings.Trim(normalizeGCPChatPath(path), "/"), "/")
	if len(parts) != 7 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "users" || parts[4] != "spaces" || parts[6] != "spaceNotificationSetting" {
		return "", "", false
	}
	userID = strings.TrimSpace(parts[3])
	spaceID = strings.TrimSpace(parts[5])
	if userID == "" || spaceID == "" {
		return "", "", false
	}
	return userID, spaceID, true
}

func parseGCPChatSpaceEventsCollectionSpaceID(path string) (string, bool) {
	parts := strings.Split(strings.Trim(normalizeGCPChatPath(path), "/"), "/")
	if len(parts) != 5 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "spaces" || parts[4] != "spaceEvents" {
		return "", false
	}
	if strings.TrimSpace(parts[3]) == "" {
		return "", false
	}
	return strings.TrimSpace(parts[3]), true
}

func parseGCPChatSpaceEventPath(path string) (spaceID, eventID string, ok bool) {
	parts := strings.Split(strings.Trim(normalizeGCPChatPath(path), "/"), "/")
	if len(parts) != 6 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "spaces" || parts[4] != "spaceEvents" {
		return "", "", false
	}
	spaceID = strings.TrimSpace(parts[3])
	eventID = strings.TrimSpace(parts[5])
	if spaceID == "" || eventID == "" {
		return "", "", false
	}
	return spaceID, eventID, true
}

func parseGCPChatCustomEmojiID(path string) (string, bool) {
	parts := strings.Split(strings.Trim(normalizeGCPChatPath(path), "/"), "/")
	if len(parts) != 4 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "customEmojis" || strings.TrimSpace(parts[3]) == "" {
		return "", false
	}
	return strings.TrimSpace(parts[3]), true
}

func parseGCPChatAttachmentUploadSpaceID(path string) (string, bool) {
	parts := strings.Split(strings.Trim(normalizeGCPChatPath(path), "/"), "/")
	if len(parts) != 5 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "spaces" || strings.TrimSpace(parts[3]) == "" {
		return "", false
	}
	action := normalizeGCPChatPath(parts[4])
	if action != "attachments:upload" {
		return "", false
	}
	return strings.TrimSpace(parts[3]), true
}

func parseGCPChatAttachmentPath(path string) (spaceID, messageID, attachmentID string, ok bool) {
	parts := strings.Split(strings.Trim(normalizeGCPChatPath(path), "/"), "/")
	if len(parts) != 8 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "spaces" || parts[4] != "messages" || parts[6] != "attachments" {
		return "", "", "", false
	}
	spaceID = strings.TrimSpace(parts[3])
	messageID = strings.TrimSpace(parts[5])
	attachmentID = strings.TrimSpace(parts[7])
	if spaceID == "" || messageID == "" || attachmentID == "" {
		return "", "", "", false
	}
	return spaceID, messageID, attachmentID, true
}

func gcpChatSpace(spaceID string) map[string]any {
	return map[string]any{
		"name":        fmt.Sprintf("spaces/%s", spaceID),
		"displayName": "Team Space",
		"spaceType":   "SPACE",
	}
}

func gcpChatMessage(spaceID, messageID, text string) map[string]any {
	return map[string]any{
		"name": fmt.Sprintf("spaces/%s/messages/%s", spaceID, messageID),
		"sender": map[string]any{
			"name":        "users/me",
			"displayName": "Stackyard Bot",
		},
		"text": text,
	}
}

func gcpChatMembership(spaceID, memberID string) map[string]any {
	return map[string]any{
		"name": fmt.Sprintf("spaces/%s/members/%s", spaceID, memberID),
		"member": map[string]any{
			"name": fmt.Sprintf("users/%s", memberID),
		},
	}
}

func gcpChatReaction(spaceID, messageID, reactionID string) map[string]any {
	return map[string]any{
		"name": fmt.Sprintf("spaces/%s/messages/%s/reactions/%s", spaceID, messageID, reactionID),
		"emoji": map[string]any{
			"unicode": "👍",
		},
	}
}

func gcpChatSpaceEvent(spaceID, eventID string) map[string]any {
	return map[string]any{
		"name":      fmt.Sprintf("spaces/%s/spaceEvents/%s", spaceID, eventID),
		"eventType": "MESSAGE_POSTED",
		"payload": map[string]any{
			"message": gcpChatMessage(spaceID, "message-1", "hello from stackyard"),
		},
	}
}

func gcpChatCustomEmoji(emojiID string) map[string]any {
	return map[string]any{
		"name":       fmt.Sprintf("customEmojis/%s", emojiID),
		"emojiName":  "stackyard",
		"uid":        emojiID,
		"createTime": "2026-01-01T00:00:00Z",
	}
}

func respondGCPChatInvalidArgument(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "InvalidArgument",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}
