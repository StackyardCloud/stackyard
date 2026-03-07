package server

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func (s *Server) handleGCPConfidentialComputingRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if isGCPConfidentialComputingLocationRequest(r, path) {
		if r.Method != http.MethodGet {
			return false
		}
		if handleGCPConfidentialComputingListLocations(w, r, path) {
			return true
		}
		if handleGCPConfidentialComputingGetLocation(w, path) {
			return true
		}
		return false
	}

	if !isGCPConfidentialComputingPath(path) {
		return false
	}

	switch r.Method {
	case http.MethodGet:
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost:
		if handleGCPConfidentialComputingCreateChallenge(w, r, path) {
			return true
		}
		if handleGCPConfidentialComputingVerify(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func isGCPConfidentialComputingPath(path string) bool {
	if !strings.HasPrefix(path, "/gcp/v1/projects/") {
		return false
	}

	if strings.Contains(path, "/challenges") ||
		strings.Contains(path, ":verifyAttestation") ||
		strings.Contains(path, ":verifyConfidentialSpace") ||
		strings.Contains(path, ":verifyConfidentialGke") {
		return true
	}

	return false
}

func isGCPConfidentialComputingLocationRequest(r *http.Request, path string) bool {
	return isGCPProjectLocationDiscoveryPath(path) && hasGCPConfidentialComputingHint(r)
}

func hasGCPConfidentialComputingHint(r *http.Request) bool {
	if strings.EqualFold(strings.TrimSpace(r.Header.Get("X-Stackyard-GCP-Service")), "confidentialcomputing") {
		return true
	}
	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	return strings.Contains(userAgent, "stackyard-confidentialcomputing")
}

func isGCPProjectLocationDiscoveryPath(path string) bool {
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return false
	}
	parts := strings.Split(trimmed, "/")
	if len(parts) != 5 && len(parts) != 6 {
		return false
	}
	if parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" {
		return false
	}
	if strings.TrimSpace(parts[3]) == "" || parts[4] != "locations" {
		return false
	}
	if len(parts) == 6 && strings.TrimSpace(parts[5]) == "" {
		return false
	}
	return true
}

var gcpConfidentialComputingChallengeCreateTime = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

func handleGCPConfidentialComputingListLocations(w http.ResponseWriter, r *http.Request, path string) bool {
	project, _, list, ok := parseGCPProjectLocationPath(path)
	if !ok || !list {
		return false
	}

	pageSize, err := parseOptionalNonNegativeInt(r.URL.Query().Get("pageSize"))
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]any{
			"error":    "InvalidArgument",
			"message":  "pageSize must be a non-negative integer",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}

	start := 0
	pageToken := strings.TrimSpace(r.URL.Query().Get("pageToken"))
	if pageToken != "" {
		start, err = parseOptionalNonNegativeInt(pageToken)
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]any{
				"error":    "InvalidArgument",
				"message":  "pageToken must be a non-negative integer offset",
				"provider": providerGCP,
				"path":     path,
			})
			return true
		}
	}

	locations := []map[string]any{
		gcpConfidentialComputingLocation(project, "us-central1"),
		gcpConfidentialComputingLocation(project, "global"),
	}
	if start > len(locations) {
		respondJSON(w, http.StatusBadRequest, map[string]any{
			"error":    "InvalidArgument",
			"message":  "pageToken is out of range",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}

	end := len(locations)
	if pageSize > 0 && start+pageSize < end {
		end = start + pageSize
	}

	nextPageToken := ""
	if end < len(locations) {
		nextPageToken = strconv.Itoa(end)
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"locations":     locations[start:end],
		"nextPageToken": nextPageToken,
	})
	return true
}

func handleGCPConfidentialComputingGetLocation(w http.ResponseWriter, path string) bool {
	project, location, list, ok := parseGCPProjectLocationPath(path)
	if !ok || list {
		return false
	}

	respondJSON(w, http.StatusOK, gcpConfidentialComputingLocation(project, location))
	return true
}

func gcpConfidentialComputingLocation(project, location string) map[string]any {
	display := strings.ReplaceAll(location, "-", " ")
	if display == "" {
		display = "Unknown"
	} else {
		display = strings.ToUpper(display[:1]) + display[1:]
	}
	return map[string]any{
		"name":        fmt.Sprintf("projects/%s/locations/%s", project, location),
		"locationId":  location,
		"displayName": fmt.Sprintf("%s (Stackyard)", display),
		"labels": map[string]string{
			"service": "confidentialcomputing",
			"stage":   "emulated",
		},
	}
}

func handleGCPConfidentialComputingCreateChallenge(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, ok := parseGCPConfidentialComputingCreateChallengePath(path)
	if !ok {
		return false
	}

	challengeID := strings.TrimSpace(r.URL.Query().Get("challengeId"))
	if challengeID == "" {
		var req struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err == nil {
			trimmed := strings.Trim(req.Name, "/")
			if idx := strings.LastIndex(trimmed, "/challenges/"); idx >= 0 {
				challengeID = strings.TrimSpace(trimmed[idx+len("/challenges/"):])
			}
		}
	}
	if challengeID == "" {
		challengeID = "ch-1"
	}

	createTime := gcpConfidentialComputingChallengeCreateTime
	expireTime := createTime.Add(10 * time.Minute)
	respondJSON(w, http.StatusOK, map[string]any{
		"name":       fmt.Sprintf("projects/%s/locations/%s/challenges/%s", project, location, challengeID),
		"createTime": createTime.Format(time.RFC3339Nano),
		"expireTime": expireTime.Format(time.RFC3339Nano),
		"used":       false,
		"tpmNonce":   "nonce-" + challengeID,
	})
	return true
}

func handleGCPConfidentialComputingVerify(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, challengeID, action, ok := parseGCPConfidentialComputingVerifyPath(path)
	if !ok {
		return false
	}

	var req struct {
		Attester string `json:"attester"`
	}
	_ = json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req)

	attester := gcpConfidentialComputingAttester(action, req.Attester)
	tokenBody := fmt.Sprintf("project=%s,location=%s,challenge=%s,attester=%s", project, location, challengeID, attester)
	token := "stackyard." + base64.RawURLEncoding.EncodeToString([]byte(tokenBody)) + ".sig"

	respondJSON(w, http.StatusOK, map[string]any{
		"oidcClaimsToken": token,
		"partialErrors":   []any{},
	})
	return true
}

func gcpConfidentialComputingAttester(action, attester string) string {
	switch action {
	case "verifyConfidentialSpace":
		return "confidential-space"
	case "verifyConfidentialGke":
		return "confidential-gke"
	case "verifyAttestation":
		trimmed := strings.TrimSpace(attester)
		if trimmed != "" {
			return trimmed
		}
		return "default"
	default:
		return "default"
	}
}

func parseGCPProjectLocationPath(path string) (project, location string, list, ok bool) {
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return "", "", false, false
	}
	parts := strings.Split(trimmed, "/")
	if len(parts) != 5 && len(parts) != 6 {
		return "", "", false, false
	}
	if parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "locations" {
		return "", "", false, false
	}
	project = strings.TrimSpace(parts[3])
	if project == "" {
		return "", "", false, false
	}
	if len(parts) == 5 {
		return project, "", true, true
	}
	location = strings.TrimSpace(parts[5])
	if location == "" {
		return "", "", false, false
	}
	return project, location, false, true
}

func parseGCPConfidentialComputingCreateChallengePath(path string) (project, location string, ok bool) {
	trimmed := strings.Trim(path, "/")
	parts := strings.Split(trimmed, "/")
	if len(parts) != 7 {
		return "", "", false
	}
	if parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "locations" || parts[6] != "challenges" {
		return "", "", false
	}
	project = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	if project == "" || location == "" {
		return "", "", false
	}
	return project, location, true
}

func parseGCPConfidentialComputingVerifyPath(path string) (project, location, challengeID, action string, ok bool) {
	trimmed := strings.Trim(path, "/")
	parts := strings.Split(trimmed, "/")
	if len(parts) != 8 {
		return "", "", "", "", false
	}
	if parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "locations" || parts[6] != "challenges" {
		return "", "", "", "", false
	}
	project = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	if project == "" || location == "" {
		return "", "", "", "", false
	}
	challengeAndAction := strings.TrimSpace(parts[7])
	challengeID, action, ok = strings.Cut(challengeAndAction, ":")
	if !ok {
		return "", "", "", "", false
	}
	challengeID = strings.TrimSpace(challengeID)
	action = strings.TrimSpace(action)
	if challengeID == "" || action == "" {
		return "", "", "", "", false
	}
	if action != "verifyAttestation" && action != "verifyConfidentialSpace" && action != "verifyConfidentialGke" {
		return "", "", "", "", false
	}
	return project, location, challengeID, action, true
}

func parseOptionalNonNegativeInt(raw string) (int, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return 0, nil
	}
	n, err := strconv.Atoi(value)
	if err != nil || n < 0 {
		return 0, fmt.Errorf("expected non-negative integer")
	}
	return n, nil
}
