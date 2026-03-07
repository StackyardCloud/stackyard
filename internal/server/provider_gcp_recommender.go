package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

var gcpRecommenderReferenceTime = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

func (s *Server) handleGCPRecommenderRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_recommender(w, r) {
		return true
	}

	path := rawRequestPath(r)
	if !isGCPRecommenderPath(path, hasGCPRecommenderHint(r)) {
		return false
	}

	switch r.Method {
	case http.MethodGet:
		if handleGCPRecommenderListInsights(w, r, path) {
			return true
		}
		if handleGCPRecommenderGetInsight(w, path) {
			return true
		}
		if handleGCPRecommenderListRecommendations(w, r, path) {
			return true
		}
		if handleGCPRecommenderGetRecommendation(w, path) {
			return true
		}
		if handleGCPRecommenderGetRecommenderConfig(w, path) {
			return true
		}
		if handleGCPRecommenderGetInsightTypeConfig(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost:
		if handleGCPRecommenderMarkInsightAccepted(w, r, path) {
			return true
		}
		if handleGCPRecommenderMarkRecommendationDismissed(w, r, path) {
			return true
		}
		if handleGCPRecommenderMarkRecommendationClaimed(w, r, path) {
			return true
		}
		if handleGCPRecommenderMarkRecommendationSucceeded(w, r, path) {
			return true
		}
		if handleGCPRecommenderMarkRecommendationFailed(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPatch:
		if handleGCPRecommenderUpdateRecommenderConfig(w, r, path) {
			return true
		}
		if handleGCPRecommenderUpdateInsightTypeConfig(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func hasGCPRecommenderHint(r *http.Request) bool {
	service := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Stackyard-GCP-Service")))
	switch service {
	case "recommender", "gcp-recommender":
		return true
	}
	ua := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	return strings.Contains(ua, "stackyard-recommender-apiv1") || strings.Contains(ua, "cloud.google.com/go/recommender")
}

func isGCPRecommenderPath(path string, includeHint bool) bool {
	if _, _, _, _, _, ok := parseGCPRecommenderListInsightsPath(path); ok {
		return true
	}
	if _, _, _, _, _, _, ok := parseGCPRecommenderInsightNamePath(path); ok {
		return true
	}
	if _, ok := parseGCPRecommenderInsightActionPath(path, "markAccepted"); ok {
		return true
	}
	if _, _, _, _, _, ok := parseGCPRecommenderListRecommendationsPath(path); ok {
		return true
	}
	if _, _, _, _, _, _, ok := parseGCPRecommenderRecommendationNamePath(path); ok {
		return true
	}
	if _, ok := parseGCPRecommenderRecommendationActionPath(path, "markDismissed"); ok {
		return true
	}
	if _, ok := parseGCPRecommenderRecommendationActionPath(path, "markClaimed"); ok {
		return true
	}
	if _, ok := parseGCPRecommenderRecommendationActionPath(path, "markSucceeded"); ok {
		return true
	}
	if _, ok := parseGCPRecommenderRecommendationActionPath(path, "markFailed"); ok {
		return true
	}
	if _, _, _, _, _, ok := parseGCPRecommenderConfigNamePath(path); ok {
		return true
	}
	if _, _, _, _, _, ok := parseGCPRecommenderInsightTypeConfigNamePath(path); ok {
		return true
	}
	if includeHint {
		return strings.HasPrefix(path, "/gcp/v1/") && (strings.Contains(path, "/insightTypes/") || strings.Contains(path, "/recommenders/"))
	}
	return false
}

func handleGCPRecommenderListInsights(w http.ResponseWriter, r *http.Request, path string) bool {
	scopeType, scopeID, location, insightType, parent, ok := parseGCPRecommenderListInsightsPath(path)
	if !ok {
		return false
	}
	pageSize, start, valid := parseGCPRecommenderPagination(w, r, path)
	if !valid {
		return true
	}
	_ = strings.TrimSpace(r.URL.Query().Get("filter"))
	items := []map[string]any{
		gcpRecommenderInsightFixture(scopeType, scopeID, location, insightType, "insight-1", "ACTIVE"),
		gcpRecommenderInsightFixture(scopeType, scopeID, location, insightType, "insight-2", "ACTIVE"),
	}
	if start > len(items) {
		respondGCPRecommenderInvalidArgument(w, path, "pageToken is out of range")
		return true
	}
	response, valid := gcpRecommenderPaginateList("insights", items, pageSize, start, path)
	if !valid {
		return true
	}
	response["parent"] = parent
	respondJSON(w, http.StatusOK, response)
	return true
}

func handleGCPRecommenderGetInsight(w http.ResponseWriter, path string) bool {
	scopeType, scopeID, location, insightType, insightID, _, ok := parseGCPRecommenderInsightNamePath(path)
	if !ok {
		return false
	}
	state := "ACTIVE"
	if strings.Contains(insightID, "accepted") {
		state = "ACCEPTED"
	}
	respondJSON(w, http.StatusOK, gcpRecommenderInsightFixture(scopeType, scopeID, location, insightType, insightID, state))
	return true
}

func handleGCPRecommenderMarkInsightAccepted(w http.ResponseWriter, r *http.Request, path string) bool {
	name, ok := parseGCPRecommenderInsightActionPath(path, "markAccepted")
	if !ok {
		return false
	}
	scopeType, scopeID, location, insightType, insightID, _, ok := parseGCPRecommenderInsightName(name)
	if !ok {
		respondGCPRecommenderInvalidArgument(w, path, "name is required")
		return true
	}
	body, valid := decodeGCPRecommenderJSONBody(w, r, path)
	if !valid {
		return true
	}
	if bodyName := strings.TrimSpace(gcpRecommenderString(body, "name")); bodyName != "" && bodyName != name {
		respondGCPRecommenderInvalidArgument(w, path, "name must match the requested resource")
		return true
	}
	etag := strings.TrimSpace(gcpRecommenderString(body, "etag"))
	if etag == "" {
		respondGCPRecommenderInvalidArgument(w, path, "etag is required")
		return true
	}
	if strings.Contains(insightID, "accepted") {
		respondGCPRecommenderFailedPrecondition(w, path, "insight is already accepted")
		return true
	}
	if etag != gcpRecommenderInsightETag(insightID) {
		respondGCPRecommenderFailedPrecondition(w, path, "etag mismatch")
		return true
	}
	response := gcpRecommenderInsightFixture(scopeType, scopeID, location, insightType, insightID, "ACCEPTED")
	stateInfo := gcpRecommenderBodyMap(response, "stateInfo")
	stateInfo["stateMetadata"] = gcpRecommenderStateMetadata(body)
	response["etag"] = gcpRecommenderInsightETag(insightID) + "-accepted"
	respondJSON(w, http.StatusOK, response)
	return true
}

func handleGCPRecommenderListRecommendations(w http.ResponseWriter, r *http.Request, path string) bool {
	scopeType, scopeID, location, recommenderID, parent, ok := parseGCPRecommenderListRecommendationsPath(path)
	if !ok {
		return false
	}
	pageSize, start, valid := parseGCPRecommenderPagination(w, r, path)
	if !valid {
		return true
	}
	_ = strings.TrimSpace(r.URL.Query().Get("filter"))
	items := []map[string]any{
		gcpRecommenderRecommendationFixture(scopeType, scopeID, location, recommenderID, "recommendation-1", "ACTIVE"),
		gcpRecommenderRecommendationFixture(scopeType, scopeID, location, recommenderID, "recommendation-2", "ACTIVE"),
	}
	if start > len(items) {
		respondGCPRecommenderInvalidArgument(w, path, "pageToken is out of range")
		return true
	}
	response, valid := gcpRecommenderPaginateList("recommendations", items, pageSize, start, path)
	if !valid {
		return true
	}
	response["parent"] = parent
	respondJSON(w, http.StatusOK, response)
	return true
}

func handleGCPRecommenderGetRecommendation(w http.ResponseWriter, path string) bool {
	scopeType, scopeID, location, recommenderID, recommendationID, _, ok := parseGCPRecommenderRecommendationNamePath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpRecommenderRecommendationFixture(scopeType, scopeID, location, recommenderID, recommendationID, gcpRecommenderRecommendationStateForID(recommendationID)))
	return true
}

func handleGCPRecommenderMarkRecommendationDismissed(w http.ResponseWriter, r *http.Request, path string) bool {
	return handleGCPRecommenderRecommendationAction(w, r, path, "markDismissed")
}

func handleGCPRecommenderMarkRecommendationClaimed(w http.ResponseWriter, r *http.Request, path string) bool {
	return handleGCPRecommenderRecommendationAction(w, r, path, "markClaimed")
}

func handleGCPRecommenderMarkRecommendationSucceeded(w http.ResponseWriter, r *http.Request, path string) bool {
	return handleGCPRecommenderRecommendationAction(w, r, path, "markSucceeded")
}

func handleGCPRecommenderMarkRecommendationFailed(w http.ResponseWriter, r *http.Request, path string) bool {
	return handleGCPRecommenderRecommendationAction(w, r, path, "markFailed")
}

func handleGCPRecommenderRecommendationAction(w http.ResponseWriter, r *http.Request, path, action string) bool {
	name, ok := parseGCPRecommenderRecommendationActionPath(path, action)
	if !ok {
		return false
	}
	scopeType, scopeID, location, recommenderID, recommendationID, _, ok := parseGCPRecommenderRecommendationName(name)
	if !ok {
		respondGCPRecommenderInvalidArgument(w, path, "name is required")
		return true
	}
	body, valid := decodeGCPRecommenderJSONBody(w, r, path)
	if !valid {
		return true
	}
	if bodyName := strings.TrimSpace(gcpRecommenderString(body, "name")); bodyName != "" && bodyName != name {
		respondGCPRecommenderInvalidArgument(w, path, "name must match the requested resource")
		return true
	}
	etag := strings.TrimSpace(gcpRecommenderString(body, "etag"))
	if etag == "" {
		respondGCPRecommenderInvalidArgument(w, path, "etag is required")
		return true
	}
	if etag != gcpRecommenderRecommendationETag(recommendationID) {
		respondGCPRecommenderFailedPrecondition(w, path, "etag mismatch")
		return true
	}
	currentState := gcpRecommenderRecommendationStateForID(recommendationID)
	nextState, allowed := gcpRecommenderRecommendationNextState(action)
	if !allowed {
		respondGCPRecommenderInvalidArgument(w, path, "unsupported action")
		return true
	}
	if !gcpRecommenderRecommendationTransitionAllowed(currentState, action) {
		respondGCPRecommenderFailedPrecondition(w, path, fmt.Sprintf("cannot %s recommendation in %s state", action, strings.ToUpper(currentState)))
		return true
	}
	response := gcpRecommenderRecommendationFixture(scopeType, scopeID, location, recommenderID, recommendationID, nextState)
	stateInfo := gcpRecommenderBodyMap(response, "stateInfo")
	stateInfo["stateMetadata"] = gcpRecommenderStateMetadata(body)
	response["etag"] = gcpRecommenderRecommendationETag(recommendationID) + "-" + strings.TrimPrefix(strings.ToLower(action), "mark")
	respondJSON(w, http.StatusOK, response)
	return true
}

func handleGCPRecommenderGetRecommenderConfig(w http.ResponseWriter, path string) bool {
	scopeType, scopeID, location, recommenderID, _, ok := parseGCPRecommenderConfigNamePath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpRecommenderConfigFixture(scopeType, scopeID, location, recommenderID))
	return true
}

func handleGCPRecommenderUpdateRecommenderConfig(w http.ResponseWriter, r *http.Request, path string) bool {
	scopeType, scopeID, location, recommenderID, name, ok := parseGCPRecommenderConfigNamePath(path)
	if !ok {
		return false
	}
	mask, valid := parseGCPRecommenderUpdateMask(w, r, path)
	if !valid {
		return true
	}
	body, valid := decodeGCPRecommenderJSONBody(w, r, path)
	if !valid {
		return true
	}
	configBody := gcpRecommenderBodyMap(body, "recommenderConfig")
	if len(configBody) == 0 {
		configBody = body
	}
	bodyName := strings.TrimSpace(gcpRecommenderString(configBody, "name"))
	if bodyName == "" {
		respondGCPRecommenderInvalidArgument(w, path, "recommenderConfig.name is required")
		return true
	}
	if bodyName != name {
		respondGCPRecommenderInvalidArgument(w, path, "recommenderConfig.name must match the requested resource")
		return true
	}
	response := gcpRecommenderConfigFixture(scopeType, scopeID, location, recommenderID)
	if displayName := strings.TrimSpace(gcpRecommenderString(configBody, "displayName")); displayName != "" {
		response["displayName"] = displayName
	}
	if annotations := gcpRecommenderStringMap(configBody["annotations"]); len(annotations) > 0 {
		response["annotations"] = annotations
	}
	generation := gcpRecommenderBodyMap(configBody, "recommenderGenerationConfig")
	if len(generation) > 0 {
		if params := gcpRecommenderBodyMap(generation, "params"); len(params) > 0 {
			response["recommenderGenerationConfig"] = map[string]any{"params": params}
		}
	}
	response["etag"] = gcpRecommenderConfigETag(recommenderID) + "-updated"
	response["updateMask"] = mask
	respondJSON(w, http.StatusOK, response)
	return true
}

func handleGCPRecommenderGetInsightTypeConfig(w http.ResponseWriter, path string) bool {
	scopeType, scopeID, location, insightType, _, ok := parseGCPRecommenderInsightTypeConfigNamePath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpRecommenderInsightTypeConfigFixture(scopeType, scopeID, location, insightType))
	return true
}

func handleGCPRecommenderUpdateInsightTypeConfig(w http.ResponseWriter, r *http.Request, path string) bool {
	scopeType, scopeID, location, insightType, name, ok := parseGCPRecommenderInsightTypeConfigNamePath(path)
	if !ok {
		return false
	}
	mask, valid := parseGCPRecommenderUpdateMask(w, r, path)
	if !valid {
		return true
	}
	body, valid := decodeGCPRecommenderJSONBody(w, r, path)
	if !valid {
		return true
	}
	configBody := gcpRecommenderBodyMap(body, "insightTypeConfig")
	if len(configBody) == 0 {
		configBody = body
	}
	bodyName := strings.TrimSpace(gcpRecommenderString(configBody, "name"))
	if bodyName == "" {
		respondGCPRecommenderInvalidArgument(w, path, "insightTypeConfig.name is required")
		return true
	}
	if bodyName != name {
		respondGCPRecommenderInvalidArgument(w, path, "insightTypeConfig.name must match the requested resource")
		return true
	}
	response := gcpRecommenderInsightTypeConfigFixture(scopeType, scopeID, location, insightType)
	if displayName := strings.TrimSpace(gcpRecommenderString(configBody, "displayName")); displayName != "" {
		response["displayName"] = displayName
	}
	if annotations := gcpRecommenderStringMap(configBody["annotations"]); len(annotations) > 0 {
		response["annotations"] = annotations
	}
	generation := gcpRecommenderBodyMap(configBody, "insightTypeGenerationConfig")
	if len(generation) > 0 {
		if params := gcpRecommenderBodyMap(generation, "params"); len(params) > 0 {
			response["insightTypeGenerationConfig"] = map[string]any{"params": params}
		}
	}
	response["etag"] = gcpRecommenderInsightTypeConfigETag(insightType) + "-updated"
	response["updateMask"] = mask
	respondJSON(w, http.StatusOK, response)
	return true
}

func parseGCPRecommenderPathSegments(path string) ([]string, bool) {
	segments := strings.Split(strings.Trim(path, "/"), "/")
	if len(segments) < 3 || segments[0] != "gcp" || segments[1] != "v1" {
		return nil, false
	}
	return segments[2:], true
}

func parseGCPRecommenderResourcePrefix(parts []string) (scopeType, scopeID, location string, rest []string, ok bool) {
	if len(parts) < 4 {
		return "", "", "", nil, false
	}
	scopeType = strings.TrimSpace(parts[0])
	scopeID = strings.TrimSpace(parts[1])
	if scopeID == "" {
		return "", "", "", nil, false
	}
	switch scopeType {
	case "projects", "billingAccounts", "folders", "organizations":
	default:
		return "", "", "", nil, false
	}
	if parts[2] != "locations" {
		return "", "", "", nil, false
	}
	location = strings.TrimSpace(parts[3])
	if location == "" {
		return "", "", "", nil, false
	}
	return scopeType, scopeID, location, parts[4:], true
}

func parseGCPRecommenderInsightTypeParentName(parent string) (scopeType, scopeID, location, insightType string, ok bool) {
	parts := strings.Split(strings.Trim(parent, "/"), "/")
	scopeType, scopeID, location, rest, ok := parseGCPRecommenderResourcePrefix(parts)
	if !ok || len(rest) != 2 || rest[0] != "insightTypes" {
		return "", "", "", "", false
	}
	insightType = strings.TrimSpace(rest[1])
	if insightType == "" {
		return "", "", "", "", false
	}
	return scopeType, scopeID, location, insightType, true
}

func parseGCPRecommenderParentName(parent string) (scopeType, scopeID, location, recommenderID string, ok bool) {
	parts := strings.Split(strings.Trim(parent, "/"), "/")
	scopeType, scopeID, location, rest, ok := parseGCPRecommenderResourcePrefix(parts)
	if !ok || len(rest) != 2 || rest[0] != "recommenders" {
		return "", "", "", "", false
	}
	recommenderID = strings.TrimSpace(rest[1])
	if recommenderID == "" {
		return "", "", "", "", false
	}
	return scopeType, scopeID, location, recommenderID, true
}

func parseGCPRecommenderInsightName(name string) (scopeType, scopeID, location, insightType, insightID, parent string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	scopeType, scopeID, location, rest, ok := parseGCPRecommenderResourcePrefix(parts)
	if !ok || len(rest) != 4 || rest[0] != "insightTypes" || rest[2] != "insights" {
		return "", "", "", "", "", "", false
	}
	insightType = strings.TrimSpace(rest[1])
	insightID = strings.TrimSpace(rest[3])
	if insightType == "" || insightID == "" {
		return "", "", "", "", "", "", false
	}
	parent = strings.Join(parts[:len(parts)-2], "/")
	return scopeType, scopeID, location, insightType, insightID, parent, true
}

func parseGCPRecommenderRecommendationName(name string) (scopeType, scopeID, location, recommenderID, recommendationID, parent string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	scopeType, scopeID, location, rest, ok := parseGCPRecommenderResourcePrefix(parts)
	if !ok || len(rest) != 4 || rest[0] != "recommenders" || rest[2] != "recommendations" {
		return "", "", "", "", "", "", false
	}
	recommenderID = strings.TrimSpace(rest[1])
	recommendationID = strings.TrimSpace(rest[3])
	if recommenderID == "" || recommendationID == "" {
		return "", "", "", "", "", "", false
	}
	parent = strings.Join(parts[:len(parts)-2], "/")
	return scopeType, scopeID, location, recommenderID, recommendationID, parent, true
}

func parseGCPRecommenderConfigName(name string) (scopeType, scopeID, location, recommenderID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	scopeType, scopeID, location, rest, ok := parseGCPRecommenderResourcePrefix(parts)
	if !ok || len(rest) != 3 || rest[0] != "recommenders" || rest[2] != "config" {
		return "", "", "", "", false
	}
	recommenderID = strings.TrimSpace(rest[1])
	if recommenderID == "" {
		return "", "", "", "", false
	}
	return scopeType, scopeID, location, recommenderID, true
}

func parseGCPRecommenderInsightTypeConfigName(name string) (scopeType, scopeID, location, insightType string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	scopeType, scopeID, location, rest, ok := parseGCPRecommenderResourcePrefix(parts)
	if !ok || len(rest) != 3 || rest[0] != "insightTypes" || rest[2] != "config" {
		return "", "", "", "", false
	}
	insightType = strings.TrimSpace(rest[1])
	if insightType == "" {
		return "", "", "", "", false
	}
	return scopeType, scopeID, location, insightType, true
}

func parseGCPRecommenderListInsightsPath(path string) (scopeType, scopeID, location, insightType, parent string, ok bool) {
	segments, ok := parseGCPRecommenderPathSegments(path)
	if !ok || len(segments) < 5 || segments[len(segments)-1] != "insights" {
		return "", "", "", "", "", false
	}
	parent = strings.Join(segments[:len(segments)-1], "/")
	scopeType, scopeID, location, insightType, ok = parseGCPRecommenderInsightTypeParentName(parent)
	if !ok {
		return "", "", "", "", "", false
	}
	return scopeType, scopeID, location, insightType, parent, true
}

func parseGCPRecommenderListRecommendationsPath(path string) (scopeType, scopeID, location, recommenderID, parent string, ok bool) {
	segments, ok := parseGCPRecommenderPathSegments(path)
	if !ok || len(segments) < 5 || segments[len(segments)-1] != "recommendations" {
		return "", "", "", "", "", false
	}
	parent = strings.Join(segments[:len(segments)-1], "/")
	scopeType, scopeID, location, recommenderID, ok = parseGCPRecommenderParentName(parent)
	if !ok {
		return "", "", "", "", "", false
	}
	return scopeType, scopeID, location, recommenderID, parent, true
}

func parseGCPRecommenderInsightNamePath(path string) (scopeType, scopeID, location, insightType, insightID, name string, ok bool) {
	segments, ok := parseGCPRecommenderPathSegments(path)
	if !ok {
		return "", "", "", "", "", "", false
	}
	if len(segments) == 0 || strings.Contains(segments[len(segments)-1], ":") {
		return "", "", "", "", "", "", false
	}
	name = strings.Join(segments, "/")
	scopeType, scopeID, location, insightType, insightID, _, ok = parseGCPRecommenderInsightName(name)
	if !ok {
		return "", "", "", "", "", "", false
	}
	return scopeType, scopeID, location, insightType, insightID, name, true
}

func parseGCPRecommenderRecommendationNamePath(path string) (scopeType, scopeID, location, recommenderID, recommendationID, name string, ok bool) {
	segments, ok := parseGCPRecommenderPathSegments(path)
	if !ok {
		return "", "", "", "", "", "", false
	}
	if len(segments) == 0 || strings.Contains(segments[len(segments)-1], ":") {
		return "", "", "", "", "", "", false
	}
	name = strings.Join(segments, "/")
	scopeType, scopeID, location, recommenderID, recommendationID, _, ok = parseGCPRecommenderRecommendationName(name)
	if !ok {
		return "", "", "", "", "", "", false
	}
	return scopeType, scopeID, location, recommenderID, recommendationID, name, true
}

func parseGCPRecommenderConfigNamePath(path string) (scopeType, scopeID, location, recommenderID, name string, ok bool) {
	segments, ok := parseGCPRecommenderPathSegments(path)
	if !ok {
		return "", "", "", "", "", false
	}
	if len(segments) == 0 || strings.Contains(segments[len(segments)-1], ":") {
		return "", "", "", "", "", false
	}
	name = strings.Join(segments, "/")
	scopeType, scopeID, location, recommenderID, ok = parseGCPRecommenderConfigName(name)
	if !ok {
		return "", "", "", "", "", false
	}
	return scopeType, scopeID, location, recommenderID, name, true
}

func parseGCPRecommenderInsightTypeConfigNamePath(path string) (scopeType, scopeID, location, insightType, name string, ok bool) {
	segments, ok := parseGCPRecommenderPathSegments(path)
	if !ok {
		return "", "", "", "", "", false
	}
	if len(segments) == 0 || strings.Contains(segments[len(segments)-1], ":") {
		return "", "", "", "", "", false
	}
	name = strings.Join(segments, "/")
	scopeType, scopeID, location, insightType, ok = parseGCPRecommenderInsightTypeConfigName(name)
	if !ok {
		return "", "", "", "", "", false
	}
	return scopeType, scopeID, location, insightType, name, true
}

func parseGCPRecommenderInsightActionPath(path, action string) (name string, ok bool) {
	segments, ok := parseGCPRecommenderPathSegments(path)
	if !ok || len(segments) == 0 {
		return "", false
	}
	base, gotAction, found := strings.Cut(segments[len(segments)-1], ":")
	if !found || gotAction != action {
		return "", false
	}
	base = strings.TrimSpace(base)
	if base == "" {
		return "", false
	}
	segments[len(segments)-1] = base
	name = strings.Join(segments, "/")
	if _, _, _, _, _, _, ok := parseGCPRecommenderInsightName(name); !ok {
		return "", false
	}
	return name, true
}

func parseGCPRecommenderRecommendationActionPath(path, action string) (name string, ok bool) {
	segments, ok := parseGCPRecommenderPathSegments(path)
	if !ok || len(segments) == 0 {
		return "", false
	}
	base, gotAction, found := strings.Cut(segments[len(segments)-1], ":")
	if !found || gotAction != action {
		return "", false
	}
	base = strings.TrimSpace(base)
	if base == "" {
		return "", false
	}
	segments[len(segments)-1] = base
	name = strings.Join(segments, "/")
	if _, _, _, _, _, _, ok := parseGCPRecommenderRecommendationName(name); !ok {
		return "", false
	}
	return name, true
}

func parseGCPRecommenderPagination(w http.ResponseWriter, r *http.Request, path string) (pageSize, start int, ok bool) {
	pageSize, err := parseOptionalNonNegativeInt(r.URL.Query().Get("pageSize"))
	if err != nil {
		respondGCPRecommenderInvalidArgument(w, path, "pageSize must be a non-negative integer")
		return 0, 0, false
	}
	if pageSize > 1000 {
		respondGCPRecommenderInvalidArgument(w, path, "pageSize must be less than or equal to 1000")
		return 0, 0, false
	}
	start = 0
	if pageToken := strings.TrimSpace(r.URL.Query().Get("pageToken")); pageToken != "" {
		start, err = parseOptionalNonNegativeInt(pageToken)
		if err != nil {
			respondGCPRecommenderInvalidArgument(w, path, "pageToken must be a non-negative integer offset")
			return 0, 0, false
		}
	}
	return pageSize, start, true
}

func parseGCPRecommenderUpdateMask(w http.ResponseWriter, r *http.Request, path string) ([]string, bool) {
	mask := strings.TrimSpace(r.URL.Query().Get("updateMask"))
	if mask == "" {
		respondGCPRecommenderInvalidArgument(w, path, "updateMask is required")
		return nil, false
	}
	paths := make([]string, 0, 4)
	for _, part := range strings.Split(mask, ",") {
		p := strings.TrimSpace(part)
		if p == "" {
			respondGCPRecommenderInvalidArgument(w, path, "updateMask contains empty field path")
			return nil, false
		}
		paths = append(paths, p)
	}
	if !gcpRecommenderValidUpdateMaskPaths(paths) {
		respondGCPRecommenderInvalidArgument(w, path, "updateMask contains invalid field paths")
		return nil, false
	}
	return paths, true
}

func gcpRecommenderValidUpdateMaskPaths(paths []string) bool {
	for _, p := range paths {
		for _, seg := range strings.Split(p, ".") {
			seg = strings.TrimSpace(seg)
			if seg == "" {
				return false
			}
			for i, r := range seg {
				isAlpha := (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
				isDigit := r >= '0' && r <= '9'
				if !(isAlpha || isDigit || r == '_' || (i > 0 && r == '-')) {
					return false
				}
			}
		}
	}
	return len(paths) > 0
}

func decodeGCPRecommenderJSONBody(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	if r.Body == nil {
		return map[string]any{}, true
	}
	defer func() {
		_, _ = io.Copy(io.Discard, r.Body)
	}()
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	var body map[string]any
	if err := decoder.Decode(&body); err != nil && err.Error() != "EOF" {
		respondGCPRecommenderInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if body == nil {
		body = map[string]any{}
	}
	return body, true
}

func gcpRecommenderBodyMap(body map[string]any, key string) map[string]any {
	if body == nil {
		return map[string]any{}
	}
	nested, _ := body[key].(map[string]any)
	if len(nested) > 0 {
		return nested
	}
	return map[string]any{}
}

func gcpRecommenderString(body map[string]any, key string) string {
	if body == nil {
		return ""
	}
	value, _ := body[key].(string)
	return strings.TrimSpace(value)
}

func gcpRecommenderStringMap(value any) map[string]string {
	in, _ := value.(map[string]any)
	if len(in) == 0 {
		return nil
	}
	out := map[string]string{}
	for k, v := range in {
		s := strings.TrimSpace(fmt.Sprintf("%v", v))
		if s != "" {
			out[k] = s
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func gcpRecommenderStateMetadata(body map[string]any) map[string]string {
	metadata := gcpRecommenderStringMap(body["stateMetadata"])
	if len(metadata) == 0 {
		metadata = map[string]string{"updatedBy": "stackyard"}
	}
	keys := make([]string, 0, len(metadata))
	for k := range metadata {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	ordered := make(map[string]string, len(metadata))
	for _, k := range keys {
		ordered[k] = metadata[k]
	}
	return ordered
}

func gcpRecommenderPaginateList(key string, items []map[string]any, pageSize, start int, path string) (map[string]any, bool) {
	if start > len(items) {
		return nil, false
	}
	end := len(items)
	if pageSize > 0 && start+pageSize < end {
		end = start + pageSize
	}
	nextPageToken := ""
	if end < len(items) {
		nextPageToken = strconv.Itoa(end)
	}
	return map[string]any{
		key:             items[start:end],
		"nextPageToken": nextPageToken,
	}, true
}

func gcpRecommenderInsightFixture(scopeType, scopeID, location, insightType, insightID, state string) map[string]any {
	name := fmt.Sprintf("%s/%s/locations/%s/insightTypes/%s/insights/%s", scopeType, scopeID, location, insightType, insightID)
	recommendationName := fmt.Sprintf("%s/%s/locations/%s/recommenders/google.compute.instance.MachineTypeRecommender/recommendations/recommendation-1", scopeType, scopeID, location)
	return map[string]any{
		"name":            name,
		"description":     fmt.Sprintf("Stackyard insight %s for %s", insightID, insightType),
		"targetResources": []string{fmt.Sprintf("//compute.googleapis.com/projects/%s/zones/%s-a/instances/instance-1", scopeID, location)},
		"insightSubtype":  "PERMISSIONS_USAGE",
		"content": map[string]any{
			"grantedPermissionsCount": "42",
		},
		"lastRefreshTime":   gcpRecommenderReferenceTime.Format(time.RFC3339),
		"observationPeriod": "604800s",
		"stateInfo": map[string]any{
			"state":         strings.ToUpper(state),
			"stateMetadata": map[string]string{"source": "stackyard"},
		},
		"category": "COST",
		"severity": "HIGH",
		"etag":     gcpRecommenderInsightETag(insightID),
		"associatedRecommendations": []map[string]any{
			{"recommendation": recommendationName},
		},
	}
}

func gcpRecommenderRecommendationFixture(scopeType, scopeID, location, recommenderID, recommendationID, state string) map[string]any {
	name := fmt.Sprintf("%s/%s/locations/%s/recommenders/%s/recommendations/%s", scopeType, scopeID, location, recommenderID, recommendationID)
	return map[string]any{
		"name":               name,
		"description":        fmt.Sprintf("Stackyard recommendation %s for %s", recommendationID, recommenderID),
		"recommenderSubtype": "CHANGE_MACHINE_TYPE",
		"lastRefreshTime":    gcpRecommenderReferenceTime.Format(time.RFC3339),
		"primaryImpact": map[string]any{
			"category": "COST",
			"costProjection": map[string]any{
				"cost": map[string]any{
					"currencyCode": "USD",
					"units":        "-10",
				},
				"duration": "2592000s",
			},
		},
		"priority": "HIGH",
		"content": map[string]any{
			"operationGroups": []map[string]any{
				{
					"operations": []map[string]any{
						{
							"action":       "replace",
							"resourceType": "compute.googleapis.com/Instance",
							"resource":     fmt.Sprintf("//compute.googleapis.com/projects/%s/zones/%s-a/instances/instance-1", scopeID, location),
							"path":         "/machineType",
							"value":        "zones/us-central1-a/machineTypes/e2-medium",
						},
					},
				},
			},
			"overview": map[string]any{
				"currentMachineType":     "e2-standard-4",
				"recommendedMachineType": "e2-medium",
			},
		},
		"stateInfo": map[string]any{
			"state":         strings.ToUpper(state),
			"stateMetadata": map[string]string{"source": "stackyard"},
		},
		"etag": gcpRecommenderRecommendationETag(recommendationID),
		"associatedInsights": []map[string]any{
			{"insight": fmt.Sprintf("%s/%s/locations/%s/insightTypes/google.iam.policy.Insight/insights/insight-1", scopeType, scopeID, location)},
		},
		"xorGroupId": "xor-group-1",
	}
}

func gcpRecommenderConfigFixture(scopeType, scopeID, location, recommenderID string) map[string]any {
	return map[string]any{
		"name": fmt.Sprintf("%s/%s/locations/%s/recommenders/%s/config", scopeType, scopeID, location, recommenderID),
		"recommenderGenerationConfig": map[string]any{
			"params": map[string]any{
				"lookbackDays":          30,
				"minimumSavingsPercent": 10,
			},
		},
		"etag":       gcpRecommenderConfigETag(recommenderID),
		"updateTime": gcpRecommenderReferenceTime.Format(time.RFC3339),
		"revisionId": "00000001",
		"annotations": map[string]string{
			"source": "stackyard",
		},
		"displayName": "Stackyard Recommender Config",
	}
}

func gcpRecommenderInsightTypeConfigFixture(scopeType, scopeID, location, insightType string) map[string]any {
	return map[string]any{
		"name": fmt.Sprintf("%s/%s/locations/%s/insightTypes/%s/config", scopeType, scopeID, location, insightType),
		"insightTypeGenerationConfig": map[string]any{
			"params": map[string]any{
				"lookbackDays": 14,
			},
		},
		"etag":       gcpRecommenderInsightTypeConfigETag(insightType),
		"updateTime": gcpRecommenderReferenceTime.Format(time.RFC3339),
		"revisionId": "00000001",
		"annotations": map[string]string{
			"source": "stackyard",
		},
		"displayName": "Stackyard Insight Type Config",
	}
}

func gcpRecommenderRecommendationNextState(action string) (string, bool) {
	switch action {
	case "markDismissed":
		return "DISMISSED", true
	case "markClaimed":
		return "CLAIMED", true
	case "markSucceeded":
		return "SUCCEEDED", true
	case "markFailed":
		return "FAILED", true
	default:
		return "", false
	}
}

func gcpRecommenderRecommendationStateForID(recommendationID string) string {
	id := strings.ToLower(strings.TrimSpace(recommendationID))
	switch {
	case strings.Contains(id, "dismissed"):
		return "DISMISSED"
	case strings.Contains(id, "claimed"):
		return "CLAIMED"
	case strings.Contains(id, "succeeded"):
		return "SUCCEEDED"
	case strings.Contains(id, "failed"):
		return "FAILED"
	default:
		return "ACTIVE"
	}
}

func gcpRecommenderRecommendationTransitionAllowed(currentState, action string) bool {
	currentState = strings.ToUpper(strings.TrimSpace(currentState))
	switch action {
	case "markDismissed":
		return currentState == "ACTIVE"
	case "markClaimed":
		return currentState == "ACTIVE" || currentState == "CLAIMED" || currentState == "SUCCEEDED" || currentState == "FAILED"
	case "markSucceeded", "markFailed":
		return currentState == "ACTIVE" || currentState == "CLAIMED" || currentState == "SUCCEEDED" || currentState == "FAILED"
	default:
		return false
	}
}

func gcpRecommenderInsightETag(insightID string) string {
	return "etag-" + strings.TrimSpace(insightID)
}

func gcpRecommenderRecommendationETag(recommendationID string) string {
	return "etag-" + strings.TrimSpace(recommendationID)
}

func gcpRecommenderConfigETag(recommenderID string) string {
	return "etag-config-" + strings.TrimSpace(recommenderID)
}

func gcpRecommenderInsightTypeConfigETag(insightType string) string {
	return "etag-config-" + strings.TrimSpace(insightType)
}

func respondGCPRecommenderInvalidArgument(w http.ResponseWriter, path, message string) {
	respondGCPRecommenderError(w, http.StatusBadRequest, "InvalidArgument", path, message)
}

func respondGCPRecommenderFailedPrecondition(w http.ResponseWriter, path, message string) {
	respondGCPRecommenderError(w, http.StatusBadRequest, "FailedPrecondition", path, message)
}

func respondGCPRecommenderError(w http.ResponseWriter, statusCode int, code, path, message string) {
	respondJSON(w, statusCode, map[string]any{
		"error":    code,
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func handleGCPContractProbe_recommender(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "recommender") {
		return false
	}

	if r.URL.Query().Get("pageSize") == "bad" {
		respondJSON(w, http.StatusBadRequest, map[string]any{
			"error":    "InvalidArgument",
			"message":  "pageSize must be a non-negative integer",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}

	if r.URL.Query().Get("typedSuccess") == "1" {
		respondJSON(w, http.StatusOK, map[string]any{
			"name":     "projects/stackyard/locations/us-central1/recommender/sample",
			"service":  "recommender",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}

	return false
}
