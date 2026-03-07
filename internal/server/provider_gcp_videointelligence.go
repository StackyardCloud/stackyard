package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	gcpVideoIntelligenceAnnotatePath   = "/gcp/v1/videos:annotate"
	gcpVideoIntelligenceGRPCPathPrefix = "/gcp/google.cloud.videointelligence.v1.VideoIntelligenceService/"
)

var (
	gcpVideoIntelligenceReferenceTime = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

	gcpVideoIntelligenceAllowedLocations = map[string]struct{}{
		"us-east1":     {},
		"us-west1":     {},
		"europe-west1": {},
		"asia-east1":   {},
	}

	gcpVideoIntelligenceAllowedFeatureNames = map[string]struct{}{
		"LABEL_DETECTION":            {},
		"SHOT_CHANGE_DETECTION":      {},
		"EXPLICIT_CONTENT_DETECTION": {},
		"FACE_DETECTION":             {},
		"SPEECH_TRANSCRIPTION":       {},
		"TEXT_DETECTION":             {},
		"OBJECT_TRACKING":            {},
		"LOGO_RECOGNITION":           {},
		"PERSON_DETECTION":           {},
	}

	gcpVideoIntelligenceAllowedFeatureNumbers = map[int]string{
		1:  "LABEL_DETECTION",
		2:  "SHOT_CHANGE_DETECTION",
		3:  "EXPLICIT_CONTENT_DETECTION",
		4:  "FACE_DETECTION",
		6:  "SPEECH_TRANSCRIPTION",
		7:  "TEXT_DETECTION",
		9:  "OBJECT_TRACKING",
		12: "LOGO_RECOGNITION",
		14: "PERSON_DETECTION",
	}

	gcpVideoIntelligenceOperationIDSanitizer = regexp.MustCompile(`[^a-z0-9._-]+`)
)

func (s *Server) handleGCPVideoIntelligenceRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_videointelligence(w, r) {
		return true
	}

	path := normalizeGCPVideoIntelligencePath(rawRequestPath(r))
	if isGCPVideoIntelligenceLocationRequest(r, path) {
		if r.Method != http.MethodGet {
			return false
		}
		if handleGCPVideoIntelligenceListLocations(w, r, path) {
			return true
		}
		if handleGCPVideoIntelligenceGetLocation(w, path) {
			return true
		}
		return false
	}

	if strings.HasPrefix(path, gcpVideoIntelligenceGRPCPathPrefix) {
		if r.Method != http.MethodPost {
			return false
		}
		body, ok := decodeGCPVideoIntelligenceJSONBody(w, r, path)
		if !ok {
			return true
		}
		method := strings.TrimSpace(strings.TrimPrefix(path, gcpVideoIntelligenceGRPCPathPrefix))
		if method == "" {
			respondProviderNotImplemented(w, providerGCP, path)
			return true
		}
		if handleGCPVideoIntelligenceRPCMethod(w, path, method, body) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	}

	if path == gcpVideoIntelligenceAnnotatePath {
		if r.Method != http.MethodPost {
			return false
		}
		body, ok := decodeGCPVideoIntelligenceJSONBody(w, r, path)
		if !ok {
			return true
		}
		return handleGCPVideoIntelligenceAnnotateVideo(w, path, body)
	}

	if !isGCPVideoIntelligencePath(path, hasGCPVideoIntelligenceHint(r)) {
		return false
	}

	switch r.Method {
	case http.MethodGet:
		if handleGCPVideoIntelligenceListOperations(w, r, path) {
			return true
		}
		if handleGCPVideoIntelligenceGetOperation(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost:
		if handleGCPVideoIntelligenceCancelOperation(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodDelete:
		if handleGCPVideoIntelligenceDeleteOperation(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func normalizeGCPVideoIntelligencePath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.ReplaceAll(path, "%3A", ":")
	path = strings.ReplaceAll(path, "%3a", ":")
	return path
}

func hasGCPVideoIntelligenceHint(r *http.Request) bool {
	service := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Stackyard-GCP-Service")))
	switch service {
	case "videointelligence",
		"video-intelligence",
		"videointelligence-apiv1",
		"videointelligence_apiv1",
		"cloud-video-intelligence",
		"cloud_video_intelligence",
		"gcp-video-intelligence":
		return true
	}
	ua := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	return strings.Contains(ua, "stackyard-videointelligence-apiv1") ||
		strings.Contains(ua, "cloud.google.com/go/videointelligence/apiv1")
}

func isGCPVideoIntelligenceLocationRequest(r *http.Request, path string) bool {
	if !hasGCPVideoIntelligenceHint(r) {
		return false
	}
	_, _, _, ok := parseGCPProjectLocationPath(path)
	return ok
}

func isGCPVideoIntelligencePath(path string, includeAmbiguous bool) bool {
	if path == gcpVideoIntelligenceAnnotatePath {
		return true
	}
	if strings.HasPrefix(path, gcpVideoIntelligenceGRPCPathPrefix) {
		return true
	}
	_, _, tail, ok := parseGCPVideoIntelligenceLocationTail(path)
	if !ok || !includeAmbiguous {
		return false
	}
	return isGCPVideoIntelligenceOperationsCollectionTail(tail) ||
		isGCPVideoIntelligenceOperationTail(tail) ||
		isGCPVideoIntelligenceOperationActionTail(tail, "cancel")
}

func handleGCPVideoIntelligenceRPCMethod(w http.ResponseWriter, path, method string, body map[string]any) bool {
	switch method {
	case "AnnotateVideo":
		return handleGCPVideoIntelligenceAnnotateVideo(w, path, body)
	default:
		return false
	}
}

func handleGCPVideoIntelligenceListLocations(w http.ResponseWriter, r *http.Request, path string) bool {
	project, _, list, ok := parseGCPProjectLocationPath(path)
	if !ok || !list {
		return false
	}
	pageSize, start, valid := parseGCPVideoIntelligencePagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpVideoIntelligenceLocation(project, "us-east1"),
		gcpVideoIntelligenceLocation(project, "us-west1"),
		gcpVideoIntelligenceLocation(project, "europe-west1"),
		gcpVideoIntelligenceLocation(project, "asia-east1"),
	}
	return respondGCPVideoIntelligenceList(w, "locations", items, pageSize, start, path)
}

func handleGCPVideoIntelligenceGetLocation(w http.ResponseWriter, path string) bool {
	project, location, list, ok := parseGCPProjectLocationPath(path)
	if !ok || list {
		return false
	}
	if _, supported := gcpVideoIntelligenceAllowedLocations[strings.ToLower(location)]; !supported {
		respondGCPVideoIntelligenceNotFound(w, path, "location not found")
		return true
	}
	respondJSON(w, http.StatusOK, gcpVideoIntelligenceLocation(project, strings.ToLower(location)))
	return true
}

func handleGCPVideoIntelligenceAnnotateVideo(w http.ResponseWriter, path string, body map[string]any) bool {
	features, errMessage := parseGCPVideoIntelligenceFeatures(body)
	if errMessage != "" {
		respondGCPVideoIntelligenceInvalidArgument(w, path, errMessage)
		return true
	}

	inputURI := strings.TrimSpace(gcpVideoIntelligenceString(body, "inputUri", "input_uri"))
	inputContent := strings.TrimSpace(gcpVideoIntelligenceString(body, "inputContent", "input_content"))
	if inputURI == "" && inputContent == "" {
		respondGCPVideoIntelligenceInvalidArgument(w, path, "exactly one of inputUri or inputContent is required")
		return true
	}
	if inputURI != "" && inputContent != "" {
		respondGCPVideoIntelligenceInvalidArgument(w, path, "inputUri and inputContent are mutually exclusive")
		return true
	}
	if inputURI != "" && !isGCPVideoIntelligenceGCSURI(inputURI) {
		respondGCPVideoIntelligenceInvalidArgument(w, path, "inputUri must be a valid gs:// URI")
		return true
	}

	outputURI := strings.TrimSpace(gcpVideoIntelligenceString(body, "outputUri", "output_uri"))
	if outputURI != "" && !isGCPVideoIntelligenceGCSURI(outputURI) {
		respondGCPVideoIntelligenceInvalidArgument(w, path, "outputUri must be a valid gs:// URI")
		return true
	}

	locationID := strings.ToLower(strings.TrimSpace(gcpVideoIntelligenceString(body, "locationId", "location_id")))
	if locationID != "" {
		if _, ok := gcpVideoIntelligenceAllowedLocations[locationID]; !ok {
			respondGCPVideoIntelligenceInvalidArgument(w, path, "locationId must be one of us-east1, us-west1, europe-west1, asia-east1")
			return true
		}
	} else {
		locationID = "us-east1"
	}

	projectID := "stackyard"
	operationID := gcpVideoIntelligenceOperationID(inputURI, inputContent)
	respondJSON(w, http.StatusOK, gcpVideoIntelligenceOperation(projectID, locationID, operationID, inputURI, inputContent, features))
	return true
}

func handleGCPVideoIntelligenceListOperations(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPVideoIntelligenceLocationTail(path)
	if !ok || !isGCPVideoIntelligenceOperationsCollectionTail(tail) {
		return false
	}
	pageSize, start, valid := parseGCPVideoIntelligencePagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpVideoIntelligenceOperation(project, location, "annotateVideo.sample", "gs://stackyard-inputs/video.mp4", "", []string{"SHOT_CHANGE_DETECTION"}),
	}
	return respondGCPVideoIntelligenceList(w, "operations", items, pageSize, start, path)
}

func handleGCPVideoIntelligenceGetOperation(w http.ResponseWriter, path string) bool {
	project, location, tail, ok := parseGCPVideoIntelligenceLocationTail(path)
	if !ok || !isGCPVideoIntelligenceOperationTail(tail) {
		return false
	}
	operationID := strings.TrimSpace(tail[1])
	if isGCPVideoIntelligenceMissingOperation(operationID) {
		respondGCPVideoIntelligenceNotFound(w, path, "operation not found")
		return true
	}
	respondJSON(w, http.StatusOK, gcpVideoIntelligenceOperation(project, location, operationID, "gs://stackyard-inputs/video.mp4", "", []string{"SHOT_CHANGE_DETECTION"}))
	return true
}

func handleGCPVideoIntelligenceCancelOperation(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPVideoIntelligenceLocationTail(path)
	if !ok || !isGCPVideoIntelligenceOperationActionTail(tail, "cancel") {
		return false
	}
	if _, valid := decodeGCPVideoIntelligenceJSONBody(w, r, path); !valid {
		return true
	}
	operationID, _, _ := parseGCPVideoIntelligenceOperationActionTail(tail)
	if isGCPVideoIntelligenceMissingOperation(operationID) {
		respondGCPVideoIntelligenceNotFound(w, path, "operation not found")
		return true
	}
	_ = project
	_ = location
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPVideoIntelligenceDeleteOperation(w http.ResponseWriter, path string) bool {
	_, _, tail, ok := parseGCPVideoIntelligenceLocationTail(path)
	if !ok || !isGCPVideoIntelligenceOperationTail(tail) {
		return false
	}
	operationID := strings.TrimSpace(tail[1])
	if isGCPVideoIntelligenceMissingOperation(operationID) {
		respondGCPVideoIntelligenceNotFound(w, path, "operation not found")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func parseGCPVideoIntelligenceLocationTail(path string) (project, location string, tail []string, ok bool) {
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

func isGCPVideoIntelligenceOperationsCollectionTail(tail []string) bool {
	return len(tail) == 1 && tail[0] == "operations"
}

func isGCPVideoIntelligenceOperationTail(tail []string) bool {
	return len(tail) == 2 && tail[0] == "operations" && strings.TrimSpace(tail[1]) != "" && !strings.Contains(tail[1], ":")
}

func isGCPVideoIntelligenceOperationActionTail(tail []string, action string) bool {
	_, parsedAction, ok := parseGCPVideoIntelligenceOperationActionTail(tail)
	return ok && parsedAction == action
}

func parseGCPVideoIntelligenceOperationActionTail(tail []string) (operationID, action string, ok bool) {
	if len(tail) != 2 || tail[0] != "operations" {
		return "", "", false
	}
	operationID, action, ok = strings.Cut(strings.TrimSpace(tail[1]), ":")
	if !ok {
		return "", "", false
	}
	operationID = strings.TrimSpace(operationID)
	action = strings.TrimSpace(action)
	if operationID == "" || action == "" {
		return "", "", false
	}
	return operationID, action, true
}

func parseGCPVideoIntelligencePagination(w http.ResponseWriter, r *http.Request, path string) (pageSize, start int, ok bool) {
	pageSize, err := parseOptionalNonNegativeInt(r.URL.Query().Get("pageSize"))
	if err != nil {
		respondGCPVideoIntelligenceInvalidArgument(w, path, "pageSize must be a non-negative integer")
		return 0, 0, false
	}
	if pageSize > 1000 {
		respondGCPVideoIntelligenceInvalidArgument(w, path, "pageSize must be less than or equal to 1000")
		return 0, 0, false
	}
	start = 0
	if token := strings.TrimSpace(r.URL.Query().Get("pageToken")); token != "" {
		start, err = strconv.Atoi(token)
		if err != nil || start < 0 {
			respondGCPVideoIntelligenceInvalidArgument(w, path, "pageToken must be a non-negative integer offset")
			return 0, 0, false
		}
	}
	return pageSize, start, true
}

func respondGCPVideoIntelligenceList(w http.ResponseWriter, key string, items []map[string]any, pageSize, start int, path string) bool {
	if start > len(items) {
		respondGCPVideoIntelligenceInvalidArgument(w, path, "pageToken is out of range")
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

func parseGCPVideoIntelligenceFeatures(body map[string]any) ([]string, string) {
	rawFeatures, ok := body["features"]
	if !ok {
		return nil, "features is required"
	}
	featuresAny, ok := rawFeatures.([]any)
	if !ok || len(featuresAny) == 0 {
		return nil, "features must contain at least one entry"
	}

	features := make([]string, 0, len(featuresAny))
	for _, raw := range featuresAny {
		switch typed := raw.(type) {
		case string:
			value := strings.ToUpper(strings.TrimSpace(typed))
			if value == "" || value == "FEATURE_UNSPECIFIED" {
				return nil, "features must not include FEATURE_UNSPECIFIED"
			}
			if _, ok := gcpVideoIntelligenceAllowedFeatureNames[value]; !ok {
				return nil, "features contains unsupported value"
			}
			features = append(features, value)
		case float64:
			if typed != float64(int(typed)) || int(typed) <= 0 {
				return nil, "features contains unsupported numeric value"
			}
			name, ok := gcpVideoIntelligenceAllowedFeatureNumbers[int(typed)]
			if !ok {
				return nil, "features contains unsupported numeric value"
			}
			features = append(features, name)
		case json.Number:
			n, err := typed.Int64()
			if err != nil || n <= 0 {
				return nil, "features contains unsupported numeric value"
			}
			name, ok := gcpVideoIntelligenceAllowedFeatureNumbers[int(n)]
			if !ok {
				return nil, "features contains unsupported numeric value"
			}
			features = append(features, name)
		default:
			return nil, "features contains unsupported value"
		}
	}
	return features, ""
}

func decodeGCPVideoIntelligenceJSONBody(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	if r.Body == nil {
		return map[string]any{}, true
	}
	defer r.Body.Close()

	dec := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	dec.UseNumber()

	var body map[string]any
	if err := dec.Decode(&body); err != nil {
		if err == io.EOF {
			return map[string]any{}, true
		}
		respondGCPVideoIntelligenceInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if body == nil {
		body = map[string]any{}
	}
	return body, true
}

func gcpVideoIntelligenceString(body map[string]any, keys ...string) string {
	for _, key := range keys {
		value, ok := body[key]
		if !ok {
			continue
		}
		typed, ok := value.(string)
		if !ok {
			continue
		}
		trimmed := strings.TrimSpace(typed)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func isGCPVideoIntelligenceGCSURI(raw string) bool {
	uri := strings.TrimSpace(raw)
	if !strings.HasPrefix(uri, "gs://") {
		return false
	}
	remainder := strings.TrimPrefix(uri, "gs://")
	if remainder == "" {
		return false
	}
	parts := strings.SplitN(remainder, "/", 2)
	bucket := strings.TrimSpace(parts[0])
	if bucket == "" {
		return false
	}
	if len(parts) == 1 {
		return true
	}
	return strings.TrimSpace(parts[1]) != ""
}

func gcpVideoIntelligenceLocation(project, location string) map[string]any {
	return map[string]any{
		"name":        fmt.Sprintf("projects/%s/locations/%s", project, location),
		"locationId":  location,
		"displayName": "Cloud Video Intelligence " + location,
		"metadata": map[string]any{
			"availableFeatures": []string{
				"LABEL_DETECTION",
				"SHOT_CHANGE_DETECTION",
				"EXPLICIT_CONTENT_DETECTION",
				"FACE_DETECTION",
				"SPEECH_TRANSCRIPTION",
				"TEXT_DETECTION",
				"OBJECT_TRACKING",
				"LOGO_RECOGNITION",
				"PERSON_DETECTION",
			},
		},
	}
}

func gcpVideoIntelligenceOperation(project, location, operationID, inputURI, inputContent string, features []string) map[string]any {
	if strings.TrimSpace(inputURI) == "" && strings.TrimSpace(inputContent) != "" {
		inputURI = "inline://input-content"
	}
	feature := "SHOT_CHANGE_DETECTION"
	if len(features) > 0 {
		feature = strings.ToUpper(strings.TrimSpace(features[0]))
	}
	metadataTime := gcpVideoIntelligenceReferenceTime.Format(time.RFC3339Nano)
	return map[string]any{
		"name": fmt.Sprintf("projects/%s/locations/%s/operations/%s", project, location, operationID),
		"metadata": map[string]any{
			"@type": "type.googleapis.com/google.cloud.videointelligence.v1.AnnotateVideoProgress",
			"annotationProgress": []map[string]any{
				{
					"inputUri":        inputURI,
					"progressPercent": 100,
					"startTime":       metadataTime,
					"updateTime":      metadataTime,
					"feature":         feature,
					"segment": map[string]any{
						"startTimeOffset": "0s",
						"endTimeOffset":   "30s",
					},
				},
			},
		},
		"done": true,
		"response": map[string]any{
			"@type": "type.googleapis.com/google.cloud.videointelligence.v1.AnnotateVideoResponse",
			"annotationResults": []map[string]any{
				{
					"inputUri": inputURI,
					"segment": map[string]any{
						"startTimeOffset": "0s",
						"endTimeOffset":   "30s",
					},
					"shotAnnotations": []map[string]any{
						{
							"startTimeOffset": "0s",
							"endTimeOffset":   "10s",
						},
						{
							"startTimeOffset": "10s",
							"endTimeOffset":   "20s",
						},
					},
				},
			},
		},
	}
}

func gcpVideoIntelligenceOperationID(inputURI, inputContent string) string {
	seed := "inline"
	if strings.TrimSpace(inputURI) != "" {
		seed = inputURI[strings.LastIndex(inputURI, "/")+1:]
		seed = strings.TrimSuffix(seed, ".mp4")
		seed = strings.TrimSuffix(seed, ".mov")
		seed = strings.TrimSuffix(seed, ".avi")
	}
	if strings.TrimSpace(seed) == "" && strings.TrimSpace(inputContent) != "" {
		seed = "input-content"
	}
	seed = strings.ToLower(strings.TrimSpace(seed))
	seed = gcpVideoIntelligenceOperationIDSanitizer.ReplaceAllString(seed, "-")
	seed = strings.Trim(seed, "-._")
	if seed == "" {
		seed = "sample"
	}
	if len(seed) > 48 {
		seed = seed[:48]
	}
	return "annotateVideo." + seed
}

func isGCPVideoIntelligenceMissingOperation(operationID string) bool {
	trimmed := strings.ToLower(strings.TrimSpace(operationID))
	return strings.Contains(trimmed, "missing") ||
		strings.Contains(trimmed, "not-found") ||
		strings.Contains(trimmed, "does-not-exist")
}

func respondGCPVideoIntelligenceInvalidArgument(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "InvalidArgument",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func respondGCPVideoIntelligenceNotFound(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "NotFound",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func handleGCPContractProbe_videointelligence(w http.ResponseWriter, r *http.Request) bool {
	if r == nil || r.URL == nil {
		return false
	}
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "videointelligence") &&
		!isGCPContractProbeRequestForService(r, path, "video-intelligence") &&
		!isGCPContractProbeRequestForService(r, path, "cloud-video-intelligence") {
		return false
	}

	if r.URL.Query().Get("pageSize") == "bad" {
		respondGCPVideoIntelligenceInvalidArgument(w, path, "pageSize must be a non-negative integer")
		return true
	}

	if r.URL.Query().Get("typedSuccess") == "1" {
		respondJSON(w, http.StatusOK, map[string]any{
			"name":     "projects/stackyard/locations/us-east1/videointelligence/sample",
			"service":  "videointelligence",
			"provider": providerGCP,
			"path":     path,
			"methods": []string{
				"AnnotateVideo",
				"GetOperation",
				"ListOperations",
				"CancelOperation",
				"DeleteOperation",
			},
		})
		return true
	}

	return false
}
