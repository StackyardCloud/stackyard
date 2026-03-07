package server

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var gcpServiceHealthReferenceTime = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

func (s *Server) handleGCPServiceHealthRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_servicehealth(w, r) {
		return true
	}

	path := normalizeGCPServiceHealthPath(rawRequestPath(r))
	if isGCPServiceHealthLocationRequest(r, path) {
		if r.Method != http.MethodGet {
			return false
		}
		if handleGCPServiceHealthListLocations(w, r, path) {
			return true
		}
		if handleGCPServiceHealthGetLocation(w, path) {
			return true
		}
		return false
	}

	if !isGCPServiceHealthPath(path, hasGCPServiceHealthHint(r)) {
		return false
	}
	if r.Method != http.MethodGet {
		return false
	}

	if handleGCPServiceHealthListEvents(w, r, path) {
		return true
	}
	if handleGCPServiceHealthGetEvent(w, path) {
		return true
	}
	if handleGCPServiceHealthListOrganizationEvents(w, r, path) {
		return true
	}
	if handleGCPServiceHealthGetOrganizationEvent(w, path) {
		return true
	}
	if handleGCPServiceHealthListOrganizationImpacts(w, r, path) {
		return true
	}
	if handleGCPServiceHealthGetOrganizationImpact(w, path) {
		return true
	}

	respondProviderNotImplemented(w, providerGCP, path)
	return true
}

func normalizeGCPServiceHealthPath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.ReplaceAll(path, "%3A", ":")
	path = strings.ReplaceAll(path, "%3a", ":")
	return path
}

func hasGCPServiceHealthHint(r *http.Request) bool {
	service := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Stackyard-GCP-Service")))
	switch service {
	case "servicehealth",
		"servicehealth-apiv1",
		"servicehealth_apiv1",
		"service-health",
		"service_health",
		"gcp-service-health":
		return true
	}
	ua := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	return strings.Contains(ua, "stackyard-servicehealth-apiv1") || strings.Contains(ua, "cloud.google.com/go/servicehealth")
}

func isGCPServiceHealthLocationRequest(r *http.Request, path string) bool {
	if !hasGCPServiceHealthHint(r) {
		return false
	}
	if _, _, _, ok := parseGCPProjectLocationPath(path); ok {
		return true
	}
	_, _, _, ok := parseGCPServiceHealthOrganizationLocationPath(path)
	return ok
}

func isGCPServiceHealthPath(path string, includeHint bool) bool {
	if _, _, tail, ok := parseGCPServiceHealthProjectLocationTail(path); ok {
		if isGCPServiceHealthEventsCollectionTail(tail) || isGCPServiceHealthEventTail(tail) {
			return true
		}
	}
	if _, _, tail, ok := parseGCPServiceHealthOrganizationLocationTail(path); ok {
		if isGCPServiceHealthOrganizationEventsCollectionTail(tail) ||
			isGCPServiceHealthOrganizationEventTail(tail) ||
			isGCPServiceHealthOrganizationImpactsCollectionTail(tail) ||
			isGCPServiceHealthOrganizationImpactTail(tail) {
			return true
		}
	}
	if isGCPServiceHealthGRPCPath(path) {
		return true
	}
	if includeHint && strings.HasPrefix(path, "/gcp/v1/projects/") && strings.Contains(path, "/events") {
		return true
	}
	return false
}

func isGCPServiceHealthGRPCPath(path string) bool {
	return strings.HasPrefix(path, "/gcp/google.cloud.servicehealth.v1.ServiceHealth/")
}

func handleGCPServiceHealthListLocations(w http.ResponseWriter, r *http.Request, path string) bool {
	if project, _, list, ok := parseGCPProjectLocationPath(path); ok && list {
		pageSize, start, valid := parseGCPServiceHealthPagination(w, r, path)
		if !valid {
			return true
		}
		items := []map[string]any{
			gcpServiceHealthLocation("projects", project, "us-central1"),
			gcpServiceHealthLocation("projects", project, "global"),
		}
		return respondGCPServiceHealthList(w, "locations", items, pageSize, start, path)
	}
	orgID, _, list, ok := parseGCPServiceHealthOrganizationLocationPath(path)
	if !ok || !list {
		return false
	}
	pageSize, start, valid := parseGCPServiceHealthPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpServiceHealthLocation("organizations", orgID, "global"),
		gcpServiceHealthLocation("organizations", orgID, "us-central1"),
	}
	return respondGCPServiceHealthList(w, "locations", items, pageSize, start, path)
}

func handleGCPServiceHealthGetLocation(w http.ResponseWriter, path string) bool {
	if project, location, list, ok := parseGCPProjectLocationPath(path); ok && !list {
		respondJSON(w, http.StatusOK, gcpServiceHealthLocation("projects", project, location))
		return true
	}
	orgID, location, list, ok := parseGCPServiceHealthOrganizationLocationPath(path)
	if !ok || list {
		return false
	}
	respondJSON(w, http.StatusOK, gcpServiceHealthLocation("organizations", orgID, location))
	return true
}

func handleGCPServiceHealthListEvents(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPServiceHealthProjectLocationTail(path)
	if !ok || !isGCPServiceHealthEventsCollectionTail(tail) {
		return false
	}
	pageSize, start, valid := parseGCPServiceHealthPagination(w, r, path)
	if !valid {
		return true
	}
	view, valid := parseGCPServiceHealthEventView(r.URL.Query().Get("view"))
	if !valid {
		respondGCPServiceHealthInvalidArgument(w, path, "view must be EVENT_VIEW_UNSPECIFIED, EVENT_VIEW_BASIC, EVENT_VIEW_FULL, or 0..2")
		return true
	}
	if isGCPServiceHealthMalformedFilter(r.URL.Query().Get("filter")) {
		respondGCPServiceHealthInvalidArgument(w, path, "filter is malformed")
		return true
	}
	full := view != "EVENT_VIEW_BASIC"
	items := []map[string]any{
		gcpServiceHealthEvent(project, location, "event-1", full),
		gcpServiceHealthEvent(project, location, "event-2", full),
	}
	return respondGCPServiceHealthList(w, "events", items, pageSize, start, path)
}

func handleGCPServiceHealthGetEvent(w http.ResponseWriter, path string) bool {
	project, location, tail, ok := parseGCPServiceHealthProjectLocationTail(path)
	if !ok || !isGCPServiceHealthEventTail(tail) {
		return false
	}
	eventID := strings.TrimSpace(tail[1])
	respondJSON(w, http.StatusOK, gcpServiceHealthEvent(project, location, eventID, true))
	return true
}

func handleGCPServiceHealthListOrganizationEvents(w http.ResponseWriter, r *http.Request, path string) bool {
	orgID, location, tail, ok := parseGCPServiceHealthOrganizationLocationTail(path)
	if !ok || !isGCPServiceHealthOrganizationEventsCollectionTail(tail) {
		return false
	}
	pageSize, start, valid := parseGCPServiceHealthPagination(w, r, path)
	if !valid {
		return true
	}
	view, valid := parseGCPServiceHealthOrganizationEventView(r.URL.Query().Get("view"))
	if !valid {
		respondGCPServiceHealthInvalidArgument(w, path, "view must be ORGANIZATION_EVENT_VIEW_UNSPECIFIED, ORGANIZATION_EVENT_VIEW_BASIC, ORGANIZATION_EVENT_VIEW_FULL, or 0..2")
		return true
	}
	if isGCPServiceHealthMalformedFilter(r.URL.Query().Get("filter")) {
		respondGCPServiceHealthInvalidArgument(w, path, "filter is malformed")
		return true
	}
	full := view != "ORGANIZATION_EVENT_VIEW_BASIC"
	items := []map[string]any{
		gcpServiceHealthOrganizationEvent(orgID, location, "org-event-1", full),
		gcpServiceHealthOrganizationEvent(orgID, location, "org-event-2", full),
	}
	return respondGCPServiceHealthList(w, "organizationEvents", items, pageSize, start, path)
}

func handleGCPServiceHealthGetOrganizationEvent(w http.ResponseWriter, path string) bool {
	orgID, location, tail, ok := parseGCPServiceHealthOrganizationLocationTail(path)
	if !ok || !isGCPServiceHealthOrganizationEventTail(tail) {
		return false
	}
	eventID := strings.TrimSpace(tail[1])
	respondJSON(w, http.StatusOK, gcpServiceHealthOrganizationEvent(orgID, location, eventID, true))
	return true
}

func handleGCPServiceHealthListOrganizationImpacts(w http.ResponseWriter, r *http.Request, path string) bool {
	orgID, location, tail, ok := parseGCPServiceHealthOrganizationLocationTail(path)
	if !ok || !isGCPServiceHealthOrganizationImpactsCollectionTail(tail) {
		return false
	}
	pageSize, start, valid := parseGCPServiceHealthPagination(w, r, path)
	if !valid {
		return true
	}
	if isGCPServiceHealthMalformedFilter(r.URL.Query().Get("filter")) {
		respondGCPServiceHealthInvalidArgument(w, path, "filter is malformed")
		return true
	}
	items := []map[string]any{
		gcpServiceHealthOrganizationImpact(orgID, location, "impact-1"),
		gcpServiceHealthOrganizationImpact(orgID, location, "impact-2"),
	}
	return respondGCPServiceHealthList(w, "organizationImpacts", items, pageSize, start, path)
}

func handleGCPServiceHealthGetOrganizationImpact(w http.ResponseWriter, path string) bool {
	orgID, location, tail, ok := parseGCPServiceHealthOrganizationLocationTail(path)
	if !ok || !isGCPServiceHealthOrganizationImpactTail(tail) {
		return false
	}
	impactID := strings.TrimSpace(tail[1])
	respondJSON(w, http.StatusOK, gcpServiceHealthOrganizationImpact(orgID, location, impactID))
	return true
}

func parseGCPServiceHealthProjectLocationTail(path string) (project, location string, tail []string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 7 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "locations" {
		return "", "", nil, false
	}
	project = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	if project == "" || location == "" {
		return "", "", nil, false
	}
	return project, location, parts[6:], true
}

func parseGCPServiceHealthOrganizationLocationPath(path string) (orgID, location string, list, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 5 && len(parts) != 6 {
		return "", "", false, false
	}
	if parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "organizations" || parts[4] != "locations" {
		return "", "", false, false
	}
	orgID = strings.TrimSpace(parts[3])
	if orgID == "" {
		return "", "", false, false
	}
	if len(parts) == 5 {
		return orgID, "", true, true
	}
	location = strings.TrimSpace(parts[5])
	if location == "" {
		return "", "", false, false
	}
	return orgID, location, false, true
}

func parseGCPServiceHealthOrganizationLocationTail(path string) (orgID, location string, tail []string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 7 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "organizations" || parts[4] != "locations" {
		return "", "", nil, false
	}
	orgID = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	if orgID == "" || location == "" {
		return "", "", nil, false
	}
	return orgID, location, parts[6:], true
}

func isGCPServiceHealthEventsCollectionTail(tail []string) bool {
	return len(tail) == 1 && tail[0] == "events"
}

func isGCPServiceHealthEventTail(tail []string) bool {
	return len(tail) == 2 && tail[0] == "events" && strings.TrimSpace(tail[1]) != ""
}

func isGCPServiceHealthOrganizationEventsCollectionTail(tail []string) bool {
	return len(tail) == 1 && tail[0] == "organizationEvents"
}

func isGCPServiceHealthOrganizationEventTail(tail []string) bool {
	return len(tail) == 2 && tail[0] == "organizationEvents" && strings.TrimSpace(tail[1]) != ""
}

func isGCPServiceHealthOrganizationImpactsCollectionTail(tail []string) bool {
	return len(tail) == 1 && tail[0] == "organizationImpacts"
}

func isGCPServiceHealthOrganizationImpactTail(tail []string) bool {
	return len(tail) == 2 && tail[0] == "organizationImpacts" && strings.TrimSpace(tail[1]) != ""
}

func parseGCPServiceHealthPagination(w http.ResponseWriter, r *http.Request, path string) (pageSize, start int, ok bool) {
	rawPageSize := strings.TrimSpace(r.URL.Query().Get("pageSize"))
	if rawPageSize != "" {
		parsedPageSize, err := strconv.Atoi(rawPageSize)
		if err != nil {
			respondGCPServiceHealthInvalidArgument(w, path, "pageSize must be an integer")
			return 0, 0, false
		}
		if parsedPageSize < 1 || parsedPageSize > 100 {
			respondGCPServiceHealthOutOfRange(w, path, "pageSize must be between 1 and 100")
			return 0, 0, false
		}
		pageSize = parsedPageSize
	}

	rawPageToken := strings.TrimSpace(r.URL.Query().Get("pageToken"))
	if rawPageToken != "" {
		parsedOffset, err := strconv.Atoi(rawPageToken)
		if err != nil || parsedOffset < 0 {
			respondGCPServiceHealthInvalidArgument(w, path, "pageToken must be a non-negative integer offset")
			return 0, 0, false
		}
		start = parsedOffset
	}
	return pageSize, start, true
}

func parseGCPServiceHealthEventView(raw string) (string, bool) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", true
	}
	switch value {
	case "0", "EVENT_VIEW_UNSPECIFIED":
		return "EVENT_VIEW_UNSPECIFIED", true
	case "1", "EVENT_VIEW_BASIC":
		return "EVENT_VIEW_BASIC", true
	case "2", "EVENT_VIEW_FULL":
		return "EVENT_VIEW_FULL", true
	default:
		return "", false
	}
}

func parseGCPServiceHealthOrganizationEventView(raw string) (string, bool) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", true
	}
	switch value {
	case "0", "ORGANIZATION_EVENT_VIEW_UNSPECIFIED":
		return "ORGANIZATION_EVENT_VIEW_UNSPECIFIED", true
	case "1", "ORGANIZATION_EVENT_VIEW_BASIC":
		return "ORGANIZATION_EVENT_VIEW_BASIC", true
	case "2", "ORGANIZATION_EVENT_VIEW_FULL":
		return "ORGANIZATION_EVENT_VIEW_FULL", true
	default:
		return "", false
	}
}

func isGCPServiceHealthMalformedFilter(raw string) bool {
	filter := strings.TrimSpace(raw)
	if filter == "" {
		return false
	}
	if strings.ContainsAny(filter, "\r\n") {
		return true
	}
	if strings.Contains(filter, "!!") || strings.Contains(filter, "==") || strings.Contains(filter, ";;") {
		return true
	}
	if strings.Count(filter, "\"")%2 != 0 {
		return true
	}
	return false
}

func respondGCPServiceHealthList(w http.ResponseWriter, key string, items []map[string]any, pageSize, start int, path string) bool {
	if start > len(items) {
		respondGCPServiceHealthOutOfRange(w, path, "pageToken is out of range")
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
		"unreachable":   []string{},
	})
	return true
}

func gcpServiceHealthLocation(scope, scopeID, location string) map[string]any {
	return map[string]any{
		"name":        fmt.Sprintf("%s/%s/locations/%s", scope, scopeID, location),
		"locationId":  location,
		"displayName": "Service Health " + location,
		"labels": map[string]string{
			"service": "servicehealth",
			"scope":   scope,
		},
	}
}

func gcpServiceHealthEvent(project, location, eventID string, includeFull bool) map[string]any {
	eventName := fmt.Sprintf("projects/%s/locations/%s/events/%s", project, location, eventID)
	event := map[string]any{
		"name":             eventName,
		"title":            "Service disruption " + eventID,
		"description":      "Stackyard staged project service health event",
		"category":         "INCIDENT",
		"detailedCategory": "CONFIRMED_INCIDENT",
		"state":            "ACTIVE",
		"detailedState":    "CONFIRMED",
		"eventImpacts": []any{
			map[string]any{
				"product": map[string]any{
					"productName": "Google Kubernetes Engine",
					"id":          "gke",
				},
				"location": map[string]any{
					"locationName": location,
				},
			},
		},
		"relevance":      "RELATED",
		"updateTime":     gcpServiceHealthReferenceTime.Add(20 * time.Minute).Format(time.RFC3339),
		"startTime":      gcpServiceHealthReferenceTime.Format(time.RFC3339),
		"endTime":        gcpServiceHealthReferenceTime.Add(2 * time.Hour).Format(time.RFC3339),
		"nextUpdateTime": gcpServiceHealthReferenceTime.Add(30 * time.Minute).Format(time.RFC3339),
	}
	if includeFull {
		event["updates"] = []any{
			map[string]any{
				"updateTime":  gcpServiceHealthReferenceTime.Add(10 * time.Minute).Format(time.RFC3339),
				"title":       "Investigating",
				"description": "Google is investigating elevated error rates.",
				"symptom":     "Intermittent request failures",
				"workaround":  "Retry with exponential backoff.",
			},
		}
	}
	return event
}

func gcpServiceHealthOrganizationEvent(orgID, location, eventID string, includeFull bool) map[string]any {
	eventName := fmt.Sprintf("organizations/%s/locations/%s/organizationEvents/%s", orgID, location, eventID)
	event := map[string]any{
		"name":             eventName,
		"title":            "Organization incident " + eventID,
		"description":      "Stackyard staged organization service health event",
		"category":         "INCIDENT",
		"detailedCategory": "CONFIRMED_INCIDENT",
		"state":            "ACTIVE",
		"detailedState":    "CONFIRMED",
		"eventImpacts": []any{
			map[string]any{
				"product": map[string]any{
					"productName": "Cloud Storage",
					"id":          "storage",
				},
				"location": map[string]any{
					"locationName": location,
				},
			},
		},
		"updateTime":     gcpServiceHealthReferenceTime.Add(25 * time.Minute).Format(time.RFC3339),
		"startTime":      gcpServiceHealthReferenceTime.Format(time.RFC3339),
		"endTime":        gcpServiceHealthReferenceTime.Add(2 * time.Hour).Format(time.RFC3339),
		"nextUpdateTime": gcpServiceHealthReferenceTime.Add(35 * time.Minute).Format(time.RFC3339),
	}
	if includeFull {
		event["updates"] = []any{
			map[string]any{
				"updateTime":  gcpServiceHealthReferenceTime.Add(12 * time.Minute).Format(time.RFC3339),
				"title":       "Mitigation in progress",
				"description": "Traffic engineering changes are being rolled out.",
				"symptom":     "Increased latency in regional endpoints",
				"workaround":  "Use alternate region where possible.",
			},
		}
	}
	return event
}

func gcpServiceHealthOrganizationImpact(orgID, location, impactID string) map[string]any {
	eventName := fmt.Sprintf("organizations/%s/locations/%s/organizationEvents/org-event-1", orgID, location)
	return map[string]any{
		"name":   fmt.Sprintf("organizations/%s/locations/%s/organizationImpacts/%s", orgID, location, impactID),
		"events": []string{eventName},
		"asset": map[string]any{
			"assetName": fmt.Sprintf("//cloudresourcemanager.googleapis.com/projects/%s", "123456789"),
			"assetType": "cloudresourcemanager.googleapis.com/Project",
		},
		"updateTime": gcpServiceHealthReferenceTime.Add(40 * time.Minute).Format(time.RFC3339),
	}
}

func respondGCPServiceHealthInvalidArgument(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "InvalidArgument",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func respondGCPServiceHealthOutOfRange(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "OutOfRange",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func handleGCPContractProbe_servicehealth(w http.ResponseWriter, r *http.Request) bool {
	if r == nil || r.URL == nil {
		return false
	}
	path := normalizeGCPServiceHealthPath(rawRequestPath(r))
	if !isGCPContractProbeRequestForService(r, path, "servicehealth") {
		if r.URL.Query().Get("stackyard_contract_probe") != "1" {
			return false
		}
		parts := strings.Split(strings.Trim(path, "/"), "/")
		if len(parts) != 7 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "organizations" || parts[4] != "locations" || parts[6] != "servicehealth" {
			return false
		}
	}
	if r.URL.Query().Get("pageSize") == "bad" {
		respondGCPServiceHealthInvalidArgument(w, path, "pageSize must be a non-negative integer")
		return true
	}
	if r.URL.Query().Get("typedSuccess") != "1" {
		return false
	}

	payload := gcpServiceHealthEvent("stackyard", "global", "event-1", true)
	payload["service"] = "servicehealth"
	payload["provider"] = providerGCP
	payload["path"] = path
	respondJSON(w, http.StatusOK, payload)
	return true
}
