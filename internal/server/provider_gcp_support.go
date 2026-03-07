package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
)

var gcpSupportReferenceTime = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

func (s *Server) handleGCPSupportRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_support(w, r) {
		return true
	}

	path := normalizeGCPSupportPath(rawRequestPath(r))
	if !isGCPSupportPath(path, hasGCPSupportHint(r)) {
		return false
	}

	switch r.Method {
	case http.MethodGet:
		if handleGCPSupportGetCase(w, path) {
			return true
		}
		if handleGCPSupportListCases(w, r, path) {
			return true
		}
		if handleGCPSupportSearchCases(w, r, path) {
			return true
		}
		if handleGCPSupportSearchCaseClassifications(w, r, path) {
			return true
		}
		if handleGCPSupportListComments(w, r, path) {
			return true
		}
		if handleGCPSupportListAttachments(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost:
		if handleGCPSupportCreateCase(w, r, path) {
			return true
		}
		if handleGCPSupportEscalateCase(w, r, path) {
			return true
		}
		if handleGCPSupportCloseCase(w, r, path) {
			return true
		}
		if handleGCPSupportCreateComment(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPatch:
		if handleGCPSupportUpdateCase(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func normalizeGCPSupportPath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.ReplaceAll(path, "%3A", ":")
	path = strings.ReplaceAll(path, "%3a", ":")
	return path
}

func hasGCPSupportHint(r *http.Request) bool {
	service := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Stackyard-GCP-Service")))
	switch service {
	case "support",
		"support-apiv2",
		"support_apiv2",
		"cloud-support",
		"cloud_support",
		"cloudsupport",
		"gcp-cloud-support":
		return true
	}
	ua := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	return strings.Contains(ua, "stackyard-support-apiv2") || strings.Contains(ua, "cloud.google.com/go/support/apiv2")
}

func isGCPSupportPath(path string, includeHint bool) bool {
	if isGCPSupportGRPCPath(path) {
		return true
	}
	if _, ok := parseGCPSupportCasePath(path); ok {
		return true
	}
	if _, ok := parseGCPSupportCasesCollectionPath(path); ok {
		return true
	}
	if _, ok := parseGCPSupportSearchCasesPath(path); ok {
		return true
	}
	if _, action, ok := parseGCPSupportCaseActionPath(path); ok {
		return action == "escalate" || action == "close"
	}
	if parseGCPSupportCaseClassificationsPath(path) {
		return true
	}
	if _, ok := parseGCPSupportCommentsCollectionPath(path); ok {
		return true
	}
	if _, ok := parseGCPSupportAttachmentsCollectionPath(path); ok {
		return true
	}
	return includeHint && strings.HasPrefix(path, "/gcp/v2/")
}

func isGCPSupportGRPCPath(path string) bool {
	trimmed := strings.TrimSpace(path)
	return strings.HasPrefix(trimmed, "/gcp/google.cloud.support.v2.CaseService/") ||
		strings.HasPrefix(trimmed, "/gcp/google.cloud.support.v2.CommentService/") ||
		strings.HasPrefix(trimmed, "/gcp/google.cloud.support.v2.CaseAttachmentService/")
}

func handleGCPSupportGetCase(w http.ResponseWriter, path string) bool {
	name, ok := parseGCPSupportCasePath(path)
	if !ok {
		return false
	}
	if !isGCPSupportCaseName(name) {
		respondGCPSupportInvalidArgument(w, path, "name is invalid")
		return true
	}
	if strings.Contains(strings.ToLower(name), "missing") {
		respondGCPSupportNotFound(w, path, "case not found")
		return true
	}
	respondJSON(w, http.StatusOK, gcpSupportCase(name, strings.Contains(strings.ToLower(name), "closed"), strings.Contains(strings.ToLower(name), "escalated")))
	return true
}

func handleGCPSupportListCases(w http.ResponseWriter, r *http.Request, path string) bool {
	parent, ok := parseGCPSupportCasesCollectionPath(path)
	if !ok {
		return false
	}
	if !isGCPSupportCaseParent(parent) {
		respondGCPSupportInvalidArgument(w, path, "parent is invalid")
		return true
	}
	pageSize, start, ok := parseGCPSupportPagination(w, r, path, 100)
	if !ok {
		return true
	}
	filter := strings.TrimSpace(r.URL.Query().Get("filter"))
	if filter != "" && !isGCPSupportCaseFilter(filter) {
		respondGCPSupportInvalidArgument(w, path, "filter is invalid")
		return true
	}

	items := []map[string]any{
		gcpSupportCase(fmt.Sprintf("%s/cases/case-open-1", parent), false, false),
		gcpSupportCase(fmt.Sprintf("%s/cases/case-closed-1", parent), true, false),
	}
	items = gcpSupportFilterCases(items, filter)
	return respondGCPSupportList(w, "cases", items, pageSize, start, path)
}

func handleGCPSupportSearchCases(w http.ResponseWriter, r *http.Request, path string) bool {
	parent, ok := parseGCPSupportSearchCasesPath(path)
	if !ok {
		return false
	}
	if !isGCPSupportCaseParent(parent) {
		respondGCPSupportInvalidArgument(w, path, "parent is invalid")
		return true
	}
	pageSize, start, ok := parseGCPSupportPagination(w, r, path, 100)
	if !ok {
		return true
	}
	query := strings.TrimSpace(r.URL.Query().Get("query"))
	if query != "" && !isGCPSupportSearchQuery(query) {
		respondGCPSupportInvalidArgument(w, path, "query is invalid")
		return true
	}

	items := []map[string]any{
		gcpSupportCase(fmt.Sprintf("%s/cases/case-open-1", parent), false, false),
		gcpSupportCase(fmt.Sprintf("%s/cases/case-open-2", parent), false, true),
		gcpSupportCase(fmt.Sprintf("%s/cases/case-closed-1", parent), true, false),
	}
	items = gcpSupportFilterSearchCases(items, query)
	return respondGCPSupportList(w, "cases", items, pageSize, start, path)
}

func handleGCPSupportCreateCase(w http.ResponseWriter, r *http.Request, path string) bool {
	parent, ok := parseGCPSupportCasesCollectionPath(path)
	if !ok {
		return false
	}
	if !isGCPSupportCaseParent(parent) {
		respondGCPSupportInvalidArgument(w, path, "parent is invalid")
		return true
	}
	body, ok := decodeGCPSupportJSONBodyRequired(w, r, path)
	if !ok {
		return true
	}
	if strings.TrimSpace(gcpSupportString(body, "displayName")) == "" {
		respondGCPSupportInvalidArgument(w, path, "case.displayName is required")
		return true
	}
	if strings.TrimSpace(gcpSupportString(body, "description")) == "" {
		respondGCPSupportInvalidArgument(w, path, "case.description is required")
		return true
	}
	classification := gcpSupportBodyMap(body, "classification")
	if strings.TrimSpace(gcpSupportString(classification, "id")) == "" {
		respondGCPSupportInvalidArgument(w, path, "case.classification.id is required")
		return true
	}
	if !isGCPSupportPriority(body["priority"]) {
		respondGCPSupportInvalidArgument(w, path, "case.priority is required")
		return true
	}

	caseID := strings.TrimSpace(gcpSupportString(body, "name"))
	if caseID != "" {
		if !strings.HasPrefix(caseID, parent+"/cases/") {
			respondGCPSupportInvalidArgument(w, path, "case.name must match parent")
			return true
		}
		parts := strings.Split(strings.Trim(caseID, "/"), "/")
		caseID = parts[len(parts)-1]
	} else {
		caseID = "case-created-1"
	}
	if !isGCPSupportCaseID(caseID) {
		respondGCPSupportInvalidArgument(w, path, "case.name is invalid")
		return true
	}

	created := gcpSupportCase(fmt.Sprintf("%s/cases/%s", parent, caseID), false, false)
	created["displayName"] = strings.TrimSpace(gcpSupportString(body, "displayName"))
	created["description"] = strings.TrimSpace(gcpSupportString(body, "description"))
	created["classification"] = map[string]any{
		"id":          strings.TrimSpace(gcpSupportString(classification, "id")),
		"displayName": gcpSupportString(classification, "displayName"),
	}
	created["priority"] = gcpSupportPriorityString(body["priority"])
	if tz := strings.TrimSpace(gcpSupportString(body, "timeZone")); tz != "" {
		created["timeZone"] = tz
	}
	if languageCode := strings.TrimSpace(gcpSupportString(body, "languageCode")); languageCode != "" {
		created["languageCode"] = languageCode
	}
	if testCase, ok := body["testCase"].(bool); ok {
		created["testCase"] = testCase
	}
	if subs, ok := body["subscriberEmailAddresses"].([]any); ok {
		emails := make([]string, 0, len(subs))
		for _, raw := range subs {
			email, _ := raw.(string)
			email = strings.TrimSpace(email)
			if email != "" {
				emails = append(emails, email)
			}
		}
		if len(emails) > 0 {
			created["subscriberEmailAddresses"] = emails
		}
	}

	respondJSON(w, http.StatusOK, created)
	return true
}

func handleGCPSupportUpdateCase(w http.ResponseWriter, r *http.Request, path string) bool {
	name, ok := parseGCPSupportCasePath(path)
	if !ok {
		return false
	}
	if !isGCPSupportCaseName(name) {
		respondGCPSupportInvalidArgument(w, path, "name is invalid")
		return true
	}
	body, ok := decodeGCPSupportJSONBodyRequired(w, r, path)
	if !ok {
		return true
	}
	bodyName := strings.TrimSpace(gcpSupportString(body, "name"))
	if bodyName == "" {
		respondGCPSupportInvalidArgument(w, path, "case.name is required")
		return true
	}
	if bodyName != name {
		respondGCPSupportInvalidArgument(w, path, "case.name must match requested resource")
		return true
	}

	mask := strings.TrimSpace(r.URL.Query().Get("updateMask"))
	if mask == "" {
		respondGCPSupportInvalidArgument(w, path, "updateMask is required")
		return true
	}
	allowed := map[string]struct{}{
		"priority":                   {},
		"display_name":               {},
		"subscriber_email_addresses": {},
	}
	maskParts := splitGCPSupportCSV(mask)
	if len(maskParts) == 0 {
		respondGCPSupportInvalidArgument(w, path, "updateMask is required")
		return true
	}
	for _, field := range maskParts {
		if _, ok := allowed[field]; !ok {
			respondGCPSupportInvalidArgument(w, path, "updateMask has unsupported fields")
			return true
		}
	}

	closed := strings.Contains(strings.ToLower(name), "closed")
	escalated := strings.Contains(strings.ToLower(name), "escalated")
	updated := gcpSupportCase(name, closed, escalated)
	for _, field := range maskParts {
		switch field {
		case "priority":
			if !isGCPSupportPriority(body["priority"]) {
				respondGCPSupportInvalidArgument(w, path, "case.priority is required")
				return true
			}
			updated["priority"] = gcpSupportPriorityString(body["priority"])
		case "display_name":
			displayName := strings.TrimSpace(gcpSupportString(body, "displayName"))
			if displayName == "" {
				respondGCPSupportInvalidArgument(w, path, "case.displayName is required")
				return true
			}
			updated["displayName"] = displayName
		case "subscriber_email_addresses":
			rawSubs, ok := body["subscriberEmailAddresses"].([]any)
			if !ok {
				respondGCPSupportInvalidArgument(w, path, "case.subscriberEmailAddresses is required")
				return true
			}
			emails := make([]string, 0, len(rawSubs))
			for _, raw := range rawSubs {
				email, _ := raw.(string)
				email = strings.TrimSpace(email)
				if email != "" {
					emails = append(emails, email)
				}
			}
			updated["subscriberEmailAddresses"] = emails
		}
	}

	respondJSON(w, http.StatusOK, updated)
	return true
}

func handleGCPSupportEscalateCase(w http.ResponseWriter, r *http.Request, path string) bool {
	name, action, ok := parseGCPSupportCaseActionPath(path)
	if !ok || action != "escalate" {
		return false
	}
	if !isGCPSupportCaseName(name) {
		respondGCPSupportInvalidArgument(w, path, "name is invalid")
		return true
	}
	body, ok := decodeGCPSupportJSONBodyOptional(w, r, path)
	if !ok {
		return true
	}
	if bodyName := strings.TrimSpace(gcpSupportString(body, "name")); bodyName != "" && bodyName != name {
		respondGCPSupportInvalidArgument(w, path, "name must match requested resource")
		return true
	}

	escalation := gcpSupportBodyMap(body, "escalation")
	if len(escalation) == 0 {
		respondGCPSupportInvalidArgument(w, path, "escalation is required")
		return true
	}
	if !isGCPSupportEscalationReason(escalation["reason"]) {
		respondGCPSupportInvalidArgument(w, path, "escalation.reason is required")
		return true
	}
	if strings.TrimSpace(gcpSupportString(escalation, "justification")) == "" {
		respondGCPSupportInvalidArgument(w, path, "escalation.justification is required")
		return true
	}
	if strings.Contains(strings.ToLower(name), "closed") {
		respondGCPSupportFailedPrecondition(w, path, "cannot escalate a closed case")
		return true
	}

	respondJSON(w, http.StatusOK, gcpSupportCase(name, false, true))
	return true
}

func handleGCPSupportCloseCase(w http.ResponseWriter, r *http.Request, path string) bool {
	name, action, ok := parseGCPSupportCaseActionPath(path)
	if !ok || action != "close" {
		return false
	}
	if !isGCPSupportCaseName(name) {
		respondGCPSupportInvalidArgument(w, path, "name is invalid")
		return true
	}
	body, ok := decodeGCPSupportJSONBodyOptional(w, r, path)
	if !ok {
		return true
	}
	if bodyName := strings.TrimSpace(gcpSupportString(body, "name")); bodyName != "" && bodyName != name {
		respondGCPSupportInvalidArgument(w, path, "name must match requested resource")
		return true
	}
	if strings.Contains(strings.ToLower(name), "closed") {
		respondGCPSupportFailedPrecondition(w, path, "case is already closed")
		return true
	}

	respondJSON(w, http.StatusOK, gcpSupportCase(name, true, strings.Contains(strings.ToLower(name), "escalated")))
	return true
}

func handleGCPSupportSearchCaseClassifications(w http.ResponseWriter, r *http.Request, path string) bool {
	if !parseGCPSupportCaseClassificationsPath(path) {
		return false
	}
	pageSize, start, ok := parseGCPSupportPagination(w, r, path, 100)
	if !ok {
		return true
	}
	query := strings.TrimSpace(r.URL.Query().Get("query"))
	if query != "" && !isGCPSupportClassificationQuery(query) {
		respondGCPSupportInvalidArgument(w, path, "query is invalid")
		return true
	}

	items := []map[string]any{
		{"id": "technical-issue/compute-engine", "displayName": "Technical Issue > Compute > Compute Engine"},
		{"id": "technical-issue/storage", "displayName": "Technical Issue > Storage"},
		{"id": "billing-issue/invoice", "displayName": "Billing Issue > Invoice"},
	}
	if query != "" {
		filtered := make([]map[string]any, 0, len(items))
		q := strings.ToLower(query)
		for _, item := range items {
			id := strings.ToLower(gcpSupportString(item, "id"))
			display := strings.ToLower(gcpSupportString(item, "displayName"))
			if strings.Contains(id, q) || strings.Contains(display, q) {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}

	return respondGCPSupportList(w, "caseClassifications", items, pageSize, start, path)
}

func handleGCPSupportListComments(w http.ResponseWriter, r *http.Request, path string) bool {
	parent, ok := parseGCPSupportCommentsCollectionPath(path)
	if !ok {
		return false
	}
	if !isGCPSupportCaseName(parent) {
		respondGCPSupportInvalidArgument(w, path, "parent is invalid")
		return true
	}
	pageSize, start, ok := parseGCPSupportPagination(w, r, path, 100)
	if !ok {
		return true
	}

	items := []map[string]any{
		gcpSupportComment(parent, "comment-1", "Initial case triage completed."),
		gcpSupportComment(parent, "comment-2", "Please provide additional logs."),
	}
	return respondGCPSupportList(w, "comments", items, pageSize, start, path)
}

func handleGCPSupportCreateComment(w http.ResponseWriter, r *http.Request, path string) bool {
	parent, ok := parseGCPSupportCommentsCollectionPath(path)
	if !ok {
		return false
	}
	if !isGCPSupportCaseName(parent) {
		respondGCPSupportInvalidArgument(w, path, "parent is invalid")
		return true
	}
	body, ok := decodeGCPSupportJSONBodyRequired(w, r, path)
	if !ok {
		return true
	}
	commentBody := strings.TrimSpace(gcpSupportString(body, "body"))
	if commentBody == "" {
		respondGCPSupportInvalidArgument(w, path, "comment.body is required")
		return true
	}

	commentID := strings.TrimSpace(gcpSupportString(body, "name"))
	if commentID != "" {
		if !strings.HasPrefix(commentID, parent+"/comments/") {
			respondGCPSupportInvalidArgument(w, path, "comment.name must match parent")
			return true
		}
		parts := strings.Split(strings.Trim(commentID, "/"), "/")
		commentID = parts[len(parts)-1]
	} else {
		commentID = "comment-created-1"
	}
	if !isGCPSupportCommentID(commentID) {
		respondGCPSupportInvalidArgument(w, path, "comment.name is invalid")
		return true
	}
	respondJSON(w, http.StatusOK, gcpSupportComment(parent, commentID, commentBody))
	return true
}

func handleGCPSupportListAttachments(w http.ResponseWriter, r *http.Request, path string) bool {
	parent, ok := parseGCPSupportAttachmentsCollectionPath(path)
	if !ok {
		return false
	}
	if !isGCPSupportCaseName(parent) {
		respondGCPSupportInvalidArgument(w, path, "parent is invalid")
		return true
	}
	pageSize, start, ok := parseGCPSupportPagination(w, r, path, 100)
	if !ok {
		return true
	}

	items := []map[string]any{
		gcpSupportAttachment(parent, "attachment-1", "stacktrace.txt", "text/plain", 2048),
		gcpSupportAttachment(parent, "attachment-2", "screenshot.png", "image/png", 65536),
	}
	return respondGCPSupportList(w, "attachments", items, pageSize, start, path)
}

func parseGCPSupportCasePath(path string) (name string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 6 || parts[0] != "gcp" || parts[1] != "v2" {
		return "", false
	}
	if parts[2] != "organizations" && parts[2] != "projects" {
		return "", false
	}
	if strings.TrimSpace(parts[3]) == "" || parts[4] != "cases" || strings.TrimSpace(parts[5]) == "" {
		return "", false
	}
	name = fmt.Sprintf("%s/%s/cases/%s", parts[2], strings.TrimSpace(parts[3]), strings.TrimSpace(parts[5]))
	return name, true
}

func parseGCPSupportCasesCollectionPath(path string) (parent string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 5 || parts[0] != "gcp" || parts[1] != "v2" {
		return "", false
	}
	if parts[2] != "organizations" && parts[2] != "projects" {
		return "", false
	}
	if strings.TrimSpace(parts[3]) == "" || parts[4] != "cases" {
		return "", false
	}
	parent = fmt.Sprintf("%s/%s", parts[2], strings.TrimSpace(parts[3]))
	return parent, true
}

func parseGCPSupportSearchCasesPath(path string) (parent string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 5 || parts[0] != "gcp" || parts[1] != "v2" {
		return "", false
	}
	if parts[2] != "organizations" && parts[2] != "projects" {
		return "", false
	}
	name, action, found := strings.Cut(strings.TrimSpace(parts[4]), ":")
	if !found || name != "cases" || action != "search" {
		return "", false
	}
	if strings.TrimSpace(parts[3]) == "" {
		return "", false
	}
	parent = fmt.Sprintf("%s/%s", parts[2], strings.TrimSpace(parts[3]))
	return parent, true
}

func parseGCPSupportCaseActionPath(path string) (name, action string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 6 || parts[0] != "gcp" || parts[1] != "v2" {
		return "", "", false
	}
	if parts[2] != "organizations" && parts[2] != "projects" {
		return "", "", false
	}
	if strings.TrimSpace(parts[3]) == "" || parts[4] != "cases" {
		return "", "", false
	}
	caseID, action, found := strings.Cut(strings.TrimSpace(parts[5]), ":")
	if !found || caseID == "" || action == "" {
		return "", "", false
	}
	name = fmt.Sprintf("%s/%s/cases/%s", parts[2], strings.TrimSpace(parts[3]), caseID)
	return name, action, true
}

func parseGCPSupportCaseClassificationsPath(path string) bool {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 3 || parts[0] != "gcp" || parts[1] != "v2" {
		return false
	}
	name, action, found := strings.Cut(strings.TrimSpace(parts[2]), ":")
	return found && name == "caseClassifications" && action == "search"
}

func parseGCPSupportCommentsCollectionPath(path string) (parent string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 7 || parts[0] != "gcp" || parts[1] != "v2" {
		return "", false
	}
	if parts[2] != "organizations" && parts[2] != "projects" {
		return "", false
	}
	if strings.TrimSpace(parts[3]) == "" || parts[4] != "cases" || strings.TrimSpace(parts[5]) == "" || parts[6] != "comments" {
		return "", false
	}
	parent = fmt.Sprintf("%s/%s/cases/%s", parts[2], strings.TrimSpace(parts[3]), strings.TrimSpace(parts[5]))
	return parent, true
}

func parseGCPSupportAttachmentsCollectionPath(path string) (parent string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 7 || parts[0] != "gcp" || parts[1] != "v2" {
		return "", false
	}
	if parts[2] != "organizations" && parts[2] != "projects" {
		return "", false
	}
	if strings.TrimSpace(parts[3]) == "" || parts[4] != "cases" || strings.TrimSpace(parts[5]) == "" || parts[6] != "attachments" {
		return "", false
	}
	parent = fmt.Sprintf("%s/%s/cases/%s", parts[2], strings.TrimSpace(parts[3]), strings.TrimSpace(parts[5]))
	return parent, true
}

func parseGCPSupportPagination(w http.ResponseWriter, r *http.Request, path string, maxPageSize int) (pageSize, start int, ok bool) {
	pageSize, err := parseOptionalNonNegativeInt(r.URL.Query().Get("pageSize"))
	if err != nil {
		respondGCPSupportInvalidArgument(w, path, "pageSize must be a non-negative integer")
		return 0, 0, false
	}
	if pageSize > maxPageSize {
		respondGCPSupportOutOfRange(w, path, fmt.Sprintf("pageSize cannot exceed %d", maxPageSize))
		return 0, 0, false
	}

	start = 0
	if token := strings.TrimSpace(r.URL.Query().Get("pageToken")); token != "" {
		start, err = parseOptionalNonNegativeInt(token)
		if err != nil {
			respondGCPSupportInvalidArgument(w, path, "pageToken must be a non-negative integer offset")
			return 0, 0, false
		}
	}
	return pageSize, start, true
}

func respondGCPSupportList(w http.ResponseWriter, key string, items []map[string]any, pageSize, start int, path string) bool {
	if start > len(items) {
		respondGCPSupportInvalidArgument(w, path, "pageToken is out of range")
		return false
	}
	end := len(items)
	if pageSize > 0 && start+pageSize < end {
		end = start + pageSize
	}
	nextPageToken := ""
	if end < len(items) {
		nextPageToken = fmt.Sprintf("%d", end)
	}
	respondJSON(w, http.StatusOK, map[string]any{
		key:             items[start:end],
		"nextPageToken": nextPageToken,
	})
	return true
}

func decodeGCPSupportJSONBodyOptional(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	if r.Body == nil {
		return map[string]any{}, true
	}
	data, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		respondGCPSupportInvalidArgument(w, path, "request body could not be read")
		return nil, false
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return map[string]any{}, true
	}
	body := map[string]any{}
	if err := json.Unmarshal(data, &body); err != nil {
		respondGCPSupportInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	return body, true
}

func decodeGCPSupportJSONBodyRequired(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	body, ok := decodeGCPSupportJSONBodyOptional(w, r, path)
	if !ok {
		return nil, false
	}
	if len(body) == 0 {
		respondGCPSupportInvalidArgument(w, path, "request body is required")
		return nil, false
	}
	return body, true
}

func gcpSupportCase(name string, closed, escalated bool) map[string]any {
	state := "OPEN"
	if closed {
		state = "CLOSED"
	}
	caseID := "case"
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) > 0 {
		caseID = parts[len(parts)-1]
	}
	created := gcpSupportReferenceTime.Add(5 * time.Minute)
	updated := created.Add(30 * time.Minute)
	if closed {
		updated = updated.Add(30 * time.Minute)
	}
	return map[string]any{
		"name":        name,
		"displayName": "Stackyard Support Case " + caseID,
		"description": "Deterministic staged support case fixture",
		"classification": map[string]any{
			"id":          "technical-issue/compute-engine",
			"displayName": "Technical Issue > Compute > Compute Engine",
		},
		"timeZone":                 "America/New_York",
		"subscriberEmailAddresses": []string{"ops@example.com"},
		"state":                    state,
		"createTime":               created.Format(time.RFC3339),
		"updateTime":               updated.Format(time.RFC3339),
		"creator": map[string]any{
			"displayName": "Stackyard Operator",
			"email":       "operator@example.com",
		},
		"escalated":    escalated,
		"testCase":     true,
		"languageCode": "en",
		"priority":     "P2",
	}
}

func gcpSupportComment(parent, commentID, body string) map[string]any {
	return map[string]any{
		"name":       fmt.Sprintf("%s/comments/%s", parent, commentID),
		"createTime": gcpSupportReferenceTime.Add(20 * time.Minute).Format(time.RFC3339),
		"creator": map[string]any{
			"displayName": "Stackyard Support",
			"email":       "support@example.com",
		},
		"body": body,
	}
}

func gcpSupportAttachment(parent, attachmentID, filename, mimeType string, sizeBytes int64) map[string]any {
	return map[string]any{
		"name":       fmt.Sprintf("%s/attachments/%s", parent, attachmentID),
		"createTime": gcpSupportReferenceTime.Add(15 * time.Minute).Format(time.RFC3339),
		"creator": map[string]any{
			"displayName": "Stackyard Support",
			"email":       "support@example.com",
		},
		"filename":  filename,
		"mimeType":  mimeType,
		"sizeBytes": sizeBytes,
	}
}

func gcpSupportFilterCases(items []map[string]any, filter string) []map[string]any {
	filter = strings.TrimSpace(filter)
	if filter == "" {
		return items
	}
	parts := splitGCPSupportFilterByAnd(filter)
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		if gcpSupportCaseMatchesFilter(item, parts) {
			out = append(out, item)
		}
	}
	return out
}

func gcpSupportFilterSearchCases(items []map[string]any, query string) []map[string]any {
	query = strings.TrimSpace(query)
	if query == "" {
		return items
	}
	queryLower := strings.ToLower(query)
	if strings.HasPrefix(queryLower, "state=") || strings.HasPrefix(queryLower, "priority=") {
		return gcpSupportFilterCases(items, query)
	}
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		display := strings.ToLower(gcpSupportString(item, "displayName"))
		description := strings.ToLower(gcpSupportString(item, "description"))
		if strings.Contains(display, queryLower) || strings.Contains(description, queryLower) {
			out = append(out, item)
		}
	}
	return out
}

func gcpSupportCaseMatchesFilter(item map[string]any, parts []string) bool {
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if strings.HasPrefix(part, "state=") {
			expected := strings.TrimSpace(strings.TrimPrefix(part, "state="))
			if gcpSupportString(item, "state") != expected {
				return false
			}
			continue
		}
		if strings.HasPrefix(part, "priority=") {
			expr := strings.TrimSpace(strings.TrimPrefix(part, "priority="))
			if strings.Contains(expr, " OR ") {
				tokens := strings.Split(expr, " OR ")
				matched := false
				for _, token := range tokens {
					token = strings.TrimSpace(token)
					if token == gcpSupportString(item, "priority") {
						matched = true
						break
					}
				}
				if !matched {
					return false
				}
				continue
			}
			if gcpSupportString(item, "priority") != expr {
				return false
			}
			continue
		}
		if strings.HasPrefix(part, "creator.email=") {
			expected := strings.TrimSpace(strings.TrimPrefix(part, "creator.email="))
			expected = strings.Trim(expected, `"`)
			creator := gcpSupportBodyMap(item, "creator")
			if strings.TrimSpace(gcpSupportString(creator, "email")) != expected {
				return false
			}
			continue
		}
		return false
	}
	return true
}

func isGCPSupportCaseFilter(filter string) bool {
	parts := splitGCPSupportFilterByAnd(filter)
	if len(parts) == 0 {
		return false
	}
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			return false
		}
		if strings.HasPrefix(part, "state=") {
			state := strings.TrimSpace(strings.TrimPrefix(part, "state="))
			if state != "OPEN" && state != "CLOSED" {
				return false
			}
			continue
		}
		if strings.HasPrefix(part, "priority=") {
			expr := strings.TrimSpace(strings.TrimPrefix(part, "priority="))
			if expr == "" {
				return false
			}
			for _, p := range strings.Split(expr, " OR ") {
				if !isGCPSupportPriorityString(strings.TrimSpace(p)) {
					return false
				}
			}
			continue
		}
		if strings.HasPrefix(part, "creator.email=") {
			email := strings.TrimSpace(strings.TrimPrefix(part, "creator.email="))
			email = strings.Trim(email, `"`)
			if !strings.Contains(email, "@") {
				return false
			}
			continue
		}
		return false
	}
	return true
}

func isGCPSupportSearchQuery(query string) bool {
	query = strings.TrimSpace(query)
	if query == "" {
		return true
	}
	if strings.HasPrefix(query, "state=") || strings.HasPrefix(query, "priority=") {
		return isGCPSupportCaseFilter(query)
	}
	if strings.Contains(strings.ToLower(query), "organization=") || strings.Contains(strings.ToLower(query), "project=") {
		return true
	}
	return true
}

func isGCPSupportClassificationQuery(query string) bool {
	query = strings.TrimSpace(query)
	if query == "" {
		return true
	}
	for _, r := range query {
		if r == '\n' || r == '\r' {
			return false
		}
	}
	return true
}

func splitGCPSupportFilterByAnd(filter string) []string {
	raw := strings.Split(filter, "AND")
	parts := make([]string, 0, len(raw))
	for _, p := range raw {
		p = strings.TrimSpace(p)
		if p != "" {
			parts = append(parts, p)
		}
	}
	return parts
}

func splitGCPSupportCSV(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func isGCPSupportCaseName(name string) bool {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 4 {
		return false
	}
	if parts[0] != "organizations" && parts[0] != "projects" {
		return false
	}
	if strings.TrimSpace(parts[1]) == "" || parts[2] != "cases" {
		return false
	}
	return isGCPSupportCaseID(parts[3])
}

func isGCPSupportCaseParent(parent string) bool {
	parts := strings.Split(strings.Trim(parent, "/"), "/")
	if len(parts) != 2 {
		return false
	}
	if parts[0] != "organizations" && parts[0] != "projects" {
		return false
	}
	return strings.TrimSpace(parts[1]) != ""
}

func isGCPSupportCaseID(caseID string) bool {
	caseID = strings.TrimSpace(caseID)
	if caseID == "" {
		return false
	}
	for _, r := range caseID {
		if !(r == '-' || r == '_' || r == '.' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')) {
			return false
		}
	}
	return true
}

func isGCPSupportCommentID(commentID string) bool {
	return isGCPSupportCaseID(commentID)
}

func isGCPSupportPriority(raw any) bool {
	return isGCPSupportPriorityString(gcpSupportPriorityString(raw))
}

func isGCPSupportPriorityString(priority string) bool {
	switch strings.TrimSpace(priority) {
	case "P0", "P1", "P2", "P3", "P4", "1", "2", "3", "4", "5":
		return true
	default:
		return false
	}
}

func gcpSupportPriorityString(raw any) string {
	switch value := raw.(type) {
	case string:
		value = strings.TrimSpace(value)
		switch value {
		case "P0", "P1", "P2", "P3", "P4":
			return value
		case "1":
			return "P0"
		case "2":
			return "P1"
		case "3":
			return "P2"
		case "4":
			return "P3"
		case "5":
			return "P4"
		}
	case float64:
		switch int(value) {
		case 1:
			return "P0"
		case 2:
			return "P1"
		case 3:
			return "P2"
		case 4:
			return "P3"
		case 5:
			return "P4"
		}
	}
	return ""
}

func isGCPSupportEscalationReason(raw any) bool {
	switch value := raw.(type) {
	case string:
		switch strings.TrimSpace(value) {
		case "RESOLUTION_TIME", "TECHNICAL_EXPERTISE", "BUSINESS_IMPACT", "TECHNICAL_ISSUE", "1", "2", "3":
			return true
		}
	case float64:
		return int(value) == 1 || int(value) == 2 || int(value) == 3
	}
	return false
}

func gcpSupportBodyMap(body map[string]any, key string) map[string]any {
	nested, _ := body[key].(map[string]any)
	if len(nested) > 0 {
		return nested
	}
	return map[string]any{}
}

func gcpSupportString(body map[string]any, key string) string {
	value, _ := body[key].(string)
	return strings.TrimSpace(value)
}

func respondGCPSupportInvalidArgument(w http.ResponseWriter, path, message string) {
	respondGCPSupportError(w, http.StatusBadRequest, "InvalidArgument", path, message)
}

func respondGCPSupportFailedPrecondition(w http.ResponseWriter, path, message string) {
	respondGCPSupportError(w, http.StatusBadRequest, "FailedPrecondition", path, message)
}

func respondGCPSupportOutOfRange(w http.ResponseWriter, path, message string) {
	respondGCPSupportError(w, http.StatusBadRequest, "OutOfRange", path, message)
}

func respondGCPSupportNotFound(w http.ResponseWriter, path, message string) {
	respondGCPSupportError(w, http.StatusNotFound, "NotFound", path, message)
}

func respondGCPSupportError(w http.ResponseWriter, statusCode int, code, path, message string) {
	respondJSON(w, statusCode, map[string]any{
		"error":    code,
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func handleGCPContractProbe_support(w http.ResponseWriter, r *http.Request) bool {
	if r == nil || r.URL == nil {
		return false
	}
	if r.URL.Query().Get("stackyard_contract_probe") != "1" {
		return false
	}

	path := normalizeGCPSupportPath(rawRequestPath(r))
	if !isGCPSupportPath(path, true) {
		return false
	}

	if r.URL.Query().Get("pageSize") == "bad" {
		respondGCPSupportInvalidArgument(w, path, "pageSize must be a non-negative integer")
		return true
	}
	if r.URL.Query().Get("typedSuccess") != "1" {
		return false
	}

	payload := gcpSupportCase("projects/stackyard/cases/case-probe-1", false, false)
	payload["service"] = "support"
	payload["provider"] = providerGCP
	payload["path"] = path
	respondJSON(w, http.StatusOK, payload)
	return true
}

func gcpSupportSortCasesByName(items []map[string]any) {
	sort.SliceStable(items, func(i, j int) bool {
		return gcpSupportString(items[i], "name") < gcpSupportString(items[j], "name")
	})
}
