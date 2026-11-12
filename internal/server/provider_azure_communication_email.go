package server

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const azureCommunicationEmailAPIVersion = "2025-09-01"
const azureCommunicationEmailPrefix = "/azure/communication-email/2025-09-01"

type azureCommunicationEmailOperation struct {
	OperationID          string
	RequestDigest        string
	SenderAddress        string
	Subject              string
	Recipients           map[string][]map[string]any
	Content              map[string]any
	Headers              map[string]any
	ReplyTo              []map[string]any
	Attachments          []map[string]any
	Status               string
	ForcedTerminalStatus string
	Error                map[string]any
	PollCount            int
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

func (s *Server) handleAzureCommunicationEmailRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, azureCommunicationEmailPrefix) {
		return false
	}
	if strings.TrimSpace(r.URL.Query().Get("api-version")) != azureCommunicationEmailAPIVersion {
		azureCommunicationEmailError(w, http.StatusBadRequest, "InvalidApiVersion", "api-version must be 2025-09-01")
		return true
	}

	relative := strings.TrimPrefix(path, azureCommunicationEmailPrefix)
	segments := splitPathSegments(strings.Trim(relative, "/"))
	if len(segments) == 0 {
		azureCommunicationEmailError(w, http.StatusNotFound, "PathNotFound", "route not found")
		return true
	}

	if len(segments) == 1 && strings.EqualFold(segments[0], "emails:send") {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return true
		}
		s.handleAzureCommunicationEmailSend(w, r)
		return true
	}

	if len(segments) == 3 && strings.EqualFold(segments[0], "emails") && strings.EqualFold(segments[1], "operations") {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return true
		}
		s.handleAzureCommunicationEmailGetSendResult(w, strings.TrimSpace(segments[2]))
		return true
	}

	azureCommunicationEmailError(w, http.StatusNotFound, "PathNotFound", "route not found")
	return true
}

func (s *Server) handleAzureCommunicationEmailSend(w http.ResponseWriter, r *http.Request) {
	payload, body, ok := azureCommunicationEmailReadObject(w, r)
	if !ok {
		return
	}

	senderAddress := strings.TrimSpace(azureCommunicationEmailString(payload["senderAddress"]))
	if senderAddress == "" {
		azureCommunicationEmailError(w, http.StatusBadRequest, "InvalidRequest", "senderAddress is required")
		return
	}

	content := azureCommunicationEmailObject(payload["content"])
	subject := strings.TrimSpace(azureCommunicationEmailString(content["subject"]))
	if subject == "" {
		azureCommunicationEmailError(w, http.StatusBadRequest, "InvalidRequest", "content.subject is required")
		return
	}

	recipientsPayload := azureCommunicationEmailObject(payload["recipients"])
	toRecipients, ok := azureCommunicationEmailRecipientList(recipientsPayload["to"])
	if !ok || len(toRecipients) == 0 {
		azureCommunicationEmailError(w, http.StatusBadRequest, "InvalidRequest", "recipients.to must contain at least one recipient")
		return
	}
	ccRecipients, ok := azureCommunicationEmailRecipientList(recipientsPayload["cc"])
	if !ok {
		azureCommunicationEmailError(w, http.StatusBadRequest, "InvalidRequest", "recipients.cc must be a valid recipient list")
		return
	}
	bccRecipients, ok := azureCommunicationEmailRecipientList(recipientsPayload["bcc"])
	if !ok {
		azureCommunicationEmailError(w, http.StatusBadRequest, "InvalidRequest", "recipients.bcc must be a valid recipient list")
		return
	}

	attachments, ok := azureCommunicationEmailAttachmentList(payload["attachments"])
	if !ok {
		azureCommunicationEmailError(w, http.StatusBadRequest, "InvalidRequest", "attachments must contain name, contentType, and contentInBase64")
		return
	}
	replyTo, ok := azureCommunicationEmailRecipientList(payload["replyTo"])
	if !ok {
		azureCommunicationEmailError(w, http.StatusBadRequest, "InvalidRequest", "replyTo must be a valid recipient list")
		return
	}

	headers := azureCommunicationEmailObject(payload["headers"])
	operationID := strings.TrimSpace(r.Header.Get("Operation-Id"))
	now := time.Now().UTC()
	if operationID == "" {
		operationID = azureCommunicationEmailDeterministicID("operation", string(body), now)
	}

	recipients := map[string][]map[string]any{
		"to":  toRecipients,
		"cc":  ccRecipients,
		"bcc": bccRecipients,
	}
	requestDigest := azureCommunicationEmailDigest(operationID + "|" + string(body))
	forcedTerminalStatus := azureCommunicationEmailNormalizeStatus(azureCommunicationEmailString(headers["x-stackyard-operation-status"]))

	s.providerStorageMu.Lock()
	operation := s.azureCommunicationEmailOperations[operationID]
	if operation == nil {
		operation = &azureCommunicationEmailOperation{
			OperationID:          operationID,
			RequestDigest:        requestDigest,
			SenderAddress:        senderAddress,
			Subject:              subject,
			Recipients:           azureCommunicationEmailCloneRecipientGroups(recipients),
			Content:              azureCommunicationEmailCloneObject(content),
			Headers:              azureCommunicationEmailCloneObject(headers),
			ReplyTo:              azureCommunicationEmailCloneRecipientSlice(replyTo),
			Attachments:          azureCommunicationEmailCloneObjectSlice(attachments),
			Status:               "Running",
			ForcedTerminalStatus: forcedTerminalStatus,
			CreatedAt:            now,
			UpdatedAt:            now,
		}
		s.azureCommunicationEmailOperations[operationID] = operation
	} else {
		operation.UpdatedAt = now
	}
	fixture := azureCommunicationEmailOperationFixture(operation)
	s.providerStorageMu.Unlock()

	w.Header().Set("Operation-Location", azureCommunicationEmailOperationLocation(operationID))
	w.Header().Set("Retry-After", "1")
	respondJSON(w, http.StatusAccepted, fixture)
}

func (s *Server) handleAzureCommunicationEmailGetSendResult(w http.ResponseWriter, operationID string) {
	if operationID == "" {
		azureCommunicationEmailError(w, http.StatusBadRequest, "InvalidRequest", "operationId is required")
		return
	}

	s.providerStorageMu.Lock()
	operation := s.azureCommunicationEmailOperations[operationID]
	if operation == nil {
		s.providerStorageMu.Unlock()
		azureCommunicationEmailError(w, http.StatusNotFound, "EmailOperationNotFound", "email operation not found")
		return
	}

	if strings.EqualFold(operation.Status, "Running") {
		operation.PollCount++
		if operation.PollCount >= 2 {
			if operation.ForcedTerminalStatus != "" && !strings.EqualFold(operation.ForcedTerminalStatus, "Running") {
				operation.Status = operation.ForcedTerminalStatus
				if strings.EqualFold(operation.Status, "Failed") && len(operation.Error) == 0 {
					operation.Error = map[string]any{
						"code":    "ServiceError",
						"message": "email send operation failed",
					}
				}
			} else {
				operation.Status = "Succeeded"
			}
		}
		operation.UpdatedAt = time.Now().UTC()
	}
	fixture := azureCommunicationEmailOperationFixture(operation)
	running := strings.EqualFold(operation.Status, "Running")
	s.providerStorageMu.Unlock()

	if running {
		w.Header().Set("Retry-After", "1")
	}
	respondJSON(w, http.StatusOK, fixture)
}

func azureCommunicationEmailOperationFixture(operation *azureCommunicationEmailOperation) map[string]any {
	result := map[string]any{
		"id":     operation.OperationID,
		"status": operation.Status,
	}
	if len(operation.Error) > 0 {
		result["error"] = azureCommunicationEmailCloneObject(operation.Error)
	}
	return result
}

func azureCommunicationEmailOperationLocation(operationID string) string {
	return fmt.Sprintf("%s/emails/operations/%s?api-version=%s", azureCommunicationEmailPrefix, operationID, azureCommunicationEmailAPIVersion)
}

func azureCommunicationEmailReadObject(w http.ResponseWriter, r *http.Request) (map[string]any, []byte, bool) {
	body, err := readBodyBytes(r)
	if err != nil {
		azureCommunicationEmailError(w, http.StatusBadRequest, "InvalidRequest", "unable to read request body")
		return nil, nil, false
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		azureCommunicationEmailError(w, http.StatusBadRequest, "InvalidRequest", "request body must be valid JSON")
		return nil, nil, false
	}
	if payload == nil {
		payload = map[string]any{}
	}
	return payload, body, true
}

func azureCommunicationEmailRecipientList(value any) ([]map[string]any, bool) {
	if value == nil {
		return []map[string]any{}, true
	}
	items, ok := value.([]any)
	if !ok {
		return nil, false
	}
	out := make([]map[string]any, 0, len(items))
	for _, raw := range items {
		item, ok := raw.(map[string]any)
		if !ok {
			return nil, false
		}
		address := strings.TrimSpace(azureCommunicationEmailString(item["address"]))
		if address == "" {
			return nil, false
		}
		out = append(out, map[string]any{
			"address":     address,
			"displayName": strings.TrimSpace(azureCommunicationEmailString(item["displayName"])),
		})
	}
	return out, true
}

func azureCommunicationEmailAttachmentList(value any) ([]map[string]any, bool) {
	if value == nil {
		return []map[string]any{}, true
	}
	items, ok := value.([]any)
	if !ok {
		return nil, false
	}
	out := make([]map[string]any, 0, len(items))
	for _, raw := range items {
		item, ok := raw.(map[string]any)
		if !ok {
			return nil, false
		}
		name := strings.TrimSpace(azureCommunicationEmailString(item["name"]))
		contentType := strings.TrimSpace(azureCommunicationEmailString(item["contentType"]))
		contentInBase64 := strings.TrimSpace(azureCommunicationEmailString(item["contentInBase64"]))
		if name == "" || contentType == "" || contentInBase64 == "" {
			return nil, false
		}
		out = append(out, map[string]any{
			"name":            name,
			"contentType":     contentType,
			"contentInBase64": contentInBase64,
			"contentId":       strings.TrimSpace(azureCommunicationEmailString(item["contentId"])),
		})
	}
	return out, true
}

func azureCommunicationEmailNormalizeStatus(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "notstarted":
		return "NotStarted"
	case "running":
		return "Running"
	case "succeeded":
		return "Succeeded"
	case "failed":
		return "Failed"
	case "canceled", "cancelled":
		return "Canceled"
	default:
		return ""
	}
}

func azureCommunicationEmailDeterministicID(prefix, seed string, now time.Time) string {
	return prefix + "-" + azureCommunicationEmailDigest(seed + "|" + now.Format(time.RFC3339Nano))[:24]
}

func azureCommunicationEmailObject(value any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	if typed, ok := value.(map[string]any); ok {
		return typed
	}
	return map[string]any{}
}

func azureCommunicationEmailCloneObject(input map[string]any) map[string]any {
	if input == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(input))
	for key, value := range input {
		switch typed := value.(type) {
		case map[string]any:
			out[key] = azureCommunicationEmailCloneObject(typed)
		case []any:
			cloned := make([]any, len(typed))
			for i, item := range typed {
				if object, ok := item.(map[string]any); ok {
					cloned[i] = azureCommunicationEmailCloneObject(object)
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

func azureCommunicationEmailCloneObjectSlice(input []map[string]any) []map[string]any {
	out := make([]map[string]any, 0, len(input))
	for _, item := range input {
		out = append(out, azureCommunicationEmailCloneObject(item))
	}
	return out
}

func azureCommunicationEmailCloneRecipientSlice(input []map[string]any) []map[string]any {
	return azureCommunicationEmailCloneObjectSlice(input)
}

func azureCommunicationEmailCloneRecipientGroups(input map[string][]map[string]any) map[string][]map[string]any {
	out := make(map[string][]map[string]any, len(input))
	for key, value := range input {
		out[key] = azureCommunicationEmailCloneRecipientSlice(value)
	}
	return out
}

func azureCommunicationEmailString(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	default:
		return ""
	}
}

func azureCommunicationEmailDigest(input string) string {
	sum := sha1.Sum([]byte(input))
	return hex.EncodeToString(sum[:])
}

func azureCommunicationEmailError(w http.ResponseWriter, status int, code, message string) {
	respondJSON(w, status, map[string]any{
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
	})
}
