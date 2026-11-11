package server

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const azureCommunicationAdvancedMessagingAPIVersion = "2024-08-30"
const azureCommunicationAdvancedMessagingPrefix = "/azure/communication-advancedmessaging/2024-08-30"

var azureCommunicationAdvancedMessagingUUIDPattern = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

type azureCommunicationAdvancedMessagingMessage struct {
	Key       string
	Kind      string
	Payload   map[string]any
	Receipts  []map[string]any
	CreatedAt time.Time
	UpdatedAt time.Time
}

type azureCommunicationAdvancedMessagingTemplate struct {
	ChannelID string
	Name      string
	Language  string
	Status    string
	Kind      string
	Content   map[string]any
	CreatedAt time.Time
	UpdatedAt time.Time
}

type azureCommunicationAdvancedMessagingStream struct {
	ID          string
	ContentType string
	Content     []byte
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (s *Server) handleAzureCommunicationAdvancedMessagingRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, azureCommunicationAdvancedMessagingPrefix) {
		return false
	}
	if strings.TrimSpace(r.URL.Query().Get("api-version")) != azureCommunicationAdvancedMessagingAPIVersion {
		azureCommunicationAdvancedMessagingError(w, http.StatusBadRequest, "InvalidApiVersion", "api-version must be 2024-08-30")
		return true
	}

	relative := strings.TrimPrefix(path, azureCommunicationAdvancedMessagingPrefix)
	segments := splitPathSegments(strings.Trim(relative, "/"))
	if len(segments) < 2 || !strings.EqualFold(segments[0], "messages") {
		azureCommunicationAdvancedMessagingError(w, http.StatusNotFound, "PathNotFound", "route not found")
		return true
	}

	switch strings.ToLower(strings.TrimSpace(segments[1])) {
	case "notifications:send":
		if len(segments) != 2 {
			azureCommunicationAdvancedMessagingError(w, http.StatusNotFound, "PathNotFound", "route not found")
			return true
		}
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return true
		}
		s.handleAzureCommunicationAdvancedMessagingSend(w, r)
		return true
	case "channels":
		s.handleAzureCommunicationAdvancedMessagingTemplates(w, r, segments[2:])
		return true
	case "streams":
		s.handleAzureCommunicationAdvancedMessagingStreams(w, r, segments[2:])
		return true
	default:
		azureCommunicationAdvancedMessagingError(w, http.StatusNotFound, "PathNotFound", "route not found")
		return true
	}
}

func (s *Server) handleAzureCommunicationAdvancedMessagingTemplates(w http.ResponseWriter, r *http.Request, tail []string) {
	if len(tail) != 2 || !strings.EqualFold(tail[1], "templates") {
		azureCommunicationAdvancedMessagingError(w, http.StatusNotFound, "PathNotFound", "route not found")
		return
	}
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	channelID := strings.TrimSpace(tail[0])
	if !azureCommunicationAdvancedMessagingUUIDPattern.MatchString(channelID) {
		azureCommunicationAdvancedMessagingError(w, http.StatusBadRequest, "InvalidRequest", "channelId must be a valid UUID")
		return
	}

	templates := s.azureCommunicationAdvancedMessagingTemplatesForChannel(channelID)
	maxPageSize := len(templates)
	if raw := strings.TrimSpace(r.URL.Query().Get("maxpagesize")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value <= 0 {
			azureCommunicationAdvancedMessagingError(w, http.StatusBadRequest, "InvalidRequest", "maxpagesize must be a positive integer")
			return
		}
		maxPageSize = value
	}

	if maxPageSize > len(templates) {
		maxPageSize = len(templates)
	}
	page := templates[:maxPageSize]
	value := make([]any, 0, len(page))
	for _, item := range page {
		value = append(value, azureCommunicationAdvancedMessagingTemplateFixture(item))
	}

	var nextLink any
	if maxPageSize < len(templates) {
		nextLink = azureCommunicationAdvancedMessagingPrefix + "/messages/channels/" + channelID + "/templates?api-version=" + azureCommunicationAdvancedMessagingAPIVersion + "&continuationToken=" + strconv.Itoa(maxPageSize)
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"value":    value,
		"nextLink": nextLink,
	})
}

func (s *Server) handleAzureCommunicationAdvancedMessagingStreams(w http.ResponseWriter, r *http.Request, tail []string) {
	if len(tail) != 1 {
		azureCommunicationAdvancedMessagingError(w, http.StatusNotFound, "PathNotFound", "route not found")
		return
	}
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	streamID := strings.TrimSpace(tail[0])
	if streamID == "" {
		azureCommunicationAdvancedMessagingError(w, http.StatusBadRequest, "InvalidRequest", "id is required")
		return
	}

	s.providerStorageMu.Lock()
	stream, ok := s.azureCommunicationAdvancedMessagingStreams[streamID]
	s.providerStorageMu.Unlock()
	if !ok {
		azureCommunicationAdvancedMessagingError(w, http.StatusNotFound, "StreamNotFound", "stream not found")
		return
	}

	w.Header().Set("Content-Type", stream.ContentType)
	w.Header().Set("Content-Length", strconv.Itoa(len(stream.Content)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(stream.Content)
}

func (s *Server) handleAzureCommunicationAdvancedMessagingSend(w http.ResponseWriter, r *http.Request) {
	body, err := readBodyBytes(r)
	if err != nil {
		azureCommunicationAdvancedMessagingError(w, http.StatusBadRequest, "InvalidRequest", "unable to read request body")
		return
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		azureCommunicationAdvancedMessagingError(w, http.StatusBadRequest, "InvalidRequest", "request body must be valid JSON")
		return
	}

	kind := strings.TrimSpace(azureCommunicationAdvancedMessagingString(payload["kind"]))
	channelRegistrationID := strings.TrimSpace(azureCommunicationAdvancedMessagingString(payload["channelRegistrationId"]))
	if kind == "" {
		azureCommunicationAdvancedMessagingError(w, http.StatusBadRequest, "InvalidRequest", "kind is required")
		return
	}
	if channelRegistrationID == "" {
		azureCommunicationAdvancedMessagingError(w, http.StatusBadRequest, "InvalidRequest", "channelRegistrationId is required")
		return
	}

	recipients, ok := azureCommunicationAdvancedMessagingRecipients(payload["to"])
	if !ok || len(recipients) == 0 {
		azureCommunicationAdvancedMessagingError(w, http.StatusBadRequest, "InvalidRequest", "to must contain at least one recipient")
		return
	}

	if err := azureCommunicationAdvancedMessagingValidateKind(kind, payload); err != nil {
		azureCommunicationAdvancedMessagingError(w, http.StatusBadRequest, "InvalidRequest", err.Error())
		return
	}

	repeatabilityID := strings.TrimSpace(r.Header.Get("Repeatability-Request-ID"))
	repeatabilityFirstSent := strings.TrimSpace(r.Header.Get("Repeatability-First-Sent"))
	key := azureCommunicationAdvancedMessagingMessageKey(kind, channelRegistrationID, recipients, payload, repeatabilityID, repeatabilityFirstSent)

	now := time.Now().UTC()
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	if existing, ok := s.azureCommunicationAdvancedMessagingMessages[key]; ok && repeatabilityID != "" && repeatabilityFirstSent != "" {
		respondJSON(w, http.StatusAccepted, map[string]any{
			"receipts": azureCommunicationAdvancedMessagingCloneReceiptSlice(existing.Receipts),
		})
		return
	}

	receipts := make([]map[string]any, 0, len(recipients))
	for idx, recipient := range recipients {
		seed := fmt.Sprintf("%s|%s|%d|%s", key, recipient, idx, now.Format(time.RFC3339Nano))
		receipts = append(receipts, map[string]any{
			"messageId": "message-" + azureCommunicationAdvancedMessagingDigest(seed)[:12],
			"to":        recipient,
		})
	}

	s.azureCommunicationAdvancedMessagingMessages[key] = &azureCommunicationAdvancedMessagingMessage{
		Key:       key,
		Kind:      kind,
		Payload:   azureCommunicationAdvancedMessagingCloneObject(payload),
		Receipts:  azureCommunicationAdvancedMessagingCloneReceiptSlice(receipts),
		CreatedAt: now,
		UpdatedAt: now,
	}

	respondJSON(w, http.StatusAccepted, map[string]any{
		"receipts": receipts,
	})
}

func (s *Server) azureCommunicationAdvancedMessagingTemplatesForChannel(channelID string) []*azureCommunicationAdvancedMessagingTemplate {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	var items []*azureCommunicationAdvancedMessagingTemplate
	for _, item := range s.azureCommunicationAdvancedMessagingTemplates {
		if strings.EqualFold(item.ChannelID, channelID) {
			items = append(items, item)
		}
	}
	if len(items) == 0 {
		now := time.Now().UTC()
		seeded := []*azureCommunicationAdvancedMessagingTemplate{
			{
				ChannelID: channelID,
				Name:      "order_update",
				Language:  "en_US",
				Status:    "approved",
				Kind:      "whatsApp",
				Content: map[string]any{
					"body": "Your order {{1}} is ready for pickup.",
				},
				CreatedAt: now,
				UpdatedAt: now,
			},
			{
				ChannelID: channelID,
				Name:      "welcome_message",
				Language:  "en_US",
				Status:    "pending",
				Kind:      "whatsApp",
				Content: map[string]any{
					"body": "Welcome to Stackyard, {{1}}.",
				},
				CreatedAt: now,
				UpdatedAt: now,
			},
		}
		for _, item := range seeded {
			s.azureCommunicationAdvancedMessagingTemplates[azureCommunicationAdvancedMessagingTemplateKey(item.ChannelID, item.Name)] = item
			items = append(items, item)
		}
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Name == items[j].Name {
			return items[i].Language < items[j].Language
		}
		return items[i].Name < items[j].Name
	})
	return append([]*azureCommunicationAdvancedMessagingTemplate(nil), items...)
}

func azureCommunicationAdvancedMessagingValidateKind(kind string, payload map[string]any) error {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "text":
		if strings.TrimSpace(azureCommunicationAdvancedMessagingString(payload["content"])) == "" {
			return fmt.Errorf("content is required for text messages")
		}
	case "template":
		template := azureCommunicationAdvancedMessagingObject(payload["template"])
		if strings.TrimSpace(azureCommunicationAdvancedMessagingString(template["name"])) == "" {
			return fmt.Errorf("template.name is required for template messages")
		}
		language := azureCommunicationAdvancedMessagingObject(template["language"])
		if strings.TrimSpace(azureCommunicationAdvancedMessagingString(language["code"])) == "" {
			return fmt.Errorf("template.language.code is required for template messages")
		}
	case "image", "audio", "video", "document", "image_v0":
		if strings.TrimSpace(azureCommunicationAdvancedMessagingString(payload["mediaUri"])) == "" {
			return fmt.Errorf("mediaUri is required for %s messages", kind)
		}
	default:
		return fmt.Errorf("kind must be one of text, template, image, audio, video, document, image_v0")
	}
	return nil
}

func azureCommunicationAdvancedMessagingRecipients(value any) ([]string, bool) {
	switch typed := value.(type) {
	case string:
		if strings.TrimSpace(typed) == "" {
			return nil, false
		}
		return []string{strings.TrimSpace(typed)}, true
	case []any:
		out := make([]string, 0, len(typed))
		for _, raw := range typed {
			item := strings.TrimSpace(azureCommunicationAdvancedMessagingString(raw))
			if item == "" {
				return nil, false
			}
			out = append(out, item)
		}
		return out, len(out) > 0
	default:
		return nil, false
	}
}

func azureCommunicationAdvancedMessagingTemplateFixture(item *azureCommunicationAdvancedMessagingTemplate) map[string]any {
	return map[string]any{
		"name":     item.Name,
		"language": item.Language,
		"status":   item.Status,
		"kind":     item.Kind,
		"content":  azureCommunicationAdvancedMessagingCloneObject(item.Content),
	}
}

func azureCommunicationAdvancedMessagingMessageKey(kind, channelRegistrationID string, recipients []string, payload map[string]any, repeatabilityID, repeatabilityFirstSent string) string {
	if repeatabilityID != "" && repeatabilityFirstSent != "" {
		return strings.ToLower(repeatabilityID + "|" + repeatabilityFirstSent)
	}
	normalizedRecipients := append([]string(nil), recipients...)
	sort.Strings(normalizedRecipients)
	return azureCommunicationAdvancedMessagingDigest(kind + "|" + channelRegistrationID + "|" + strings.Join(normalizedRecipients, ",") + "|" + fmt.Sprintf("%v", payload))
}

func azureCommunicationAdvancedMessagingTemplateKey(channelID, name string) string {
	return strings.ToLower(strings.TrimSpace(channelID) + "|" + strings.TrimSpace(name))
}

func azureCommunicationAdvancedMessagingDigest(seed string) string {
	sum := sha1.Sum([]byte(seed))
	return hex.EncodeToString(sum[:])
}

func azureCommunicationAdvancedMessagingError(w http.ResponseWriter, status int, code, message string) {
	respondJSON(w, status, map[string]any{
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
	})
}

func azureCommunicationAdvancedMessagingString(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	default:
		return ""
	}
}

func azureCommunicationAdvancedMessagingObject(value any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	if typed, ok := value.(map[string]any); ok {
		return typed
	}
	return map[string]any{}
}

func azureCommunicationAdvancedMessagingCloneObject(input map[string]any) map[string]any {
	if input == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(input))
	for key, value := range input {
		switch typed := value.(type) {
		case map[string]any:
			out[key] = azureCommunicationAdvancedMessagingCloneObject(typed)
		case []any:
			cloned := make([]any, len(typed))
			for i, item := range typed {
				if object, ok := item.(map[string]any); ok {
					cloned[i] = azureCommunicationAdvancedMessagingCloneObject(object)
				} else {
					cloned[i] = item
				}
			}
			out[key] = cloned
		default:
			out[key] = value
		}
	}
	return out
}

func azureCommunicationAdvancedMessagingCloneReceiptSlice(input []map[string]any) []map[string]any {
	out := make([]map[string]any, 0, len(input))
	for _, item := range input {
		out = append(out, azureCommunicationAdvancedMessagingCloneObject(item))
	}
	return out
}
