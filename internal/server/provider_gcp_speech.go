package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	gcpSpeechRecognizePath            = "/gcp/google.cloud.speech.v1.Speech/Recognize"
	gcpSpeechLongRunningRecognizePath = "/gcp/google.cloud.speech.v1.Speech/LongRunningRecognize"
	gcpSpeechStreamingRecognizePath   = "/gcp/google.cloud.speech.v1.Speech/StreamingRecognize"

	gcpSpeechCreatePhraseSetPath   = "/gcp/google.cloud.speech.v1.Adaptation/CreatePhraseSet"
	gcpSpeechGetPhraseSetPath      = "/gcp/google.cloud.speech.v1.Adaptation/GetPhraseSet"
	gcpSpeechListPhraseSetPath     = "/gcp/google.cloud.speech.v1.Adaptation/ListPhraseSet"
	gcpSpeechUpdatePhraseSetPath   = "/gcp/google.cloud.speech.v1.Adaptation/UpdatePhraseSet"
	gcpSpeechDeletePhraseSetPath   = "/gcp/google.cloud.speech.v1.Adaptation/DeletePhraseSet"
	gcpSpeechCreateCustomClassPath = "/gcp/google.cloud.speech.v1.Adaptation/CreateCustomClass"
	gcpSpeechGetCustomClassPath    = "/gcp/google.cloud.speech.v1.Adaptation/GetCustomClass"
	gcpSpeechListCustomClassesPath = "/gcp/google.cloud.speech.v1.Adaptation/ListCustomClasses"
	gcpSpeechUpdateCustomClassPath = "/gcp/google.cloud.speech.v1.Adaptation/UpdateCustomClass"
	gcpSpeechDeleteCustomClassPath = "/gcp/google.cloud.speech.v1.Adaptation/DeleteCustomClass"
)

var gcpSpeechReferenceTime = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

func (s *Server) handleGCPSpeechRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_speech(w, r) {
		return true
	}

	path := normalizeGCPSpeechPath(rawRequestPath(r))
	if isGCPSpeechLocationRequest(r, path) {
		if r.Method != http.MethodGet {
			return false
		}
		if handleGCPSpeechListLocations(w, r, path) {
			return true
		}
		if handleGCPSpeechGetLocation(w, path) {
			return true
		}
		return false
	}

	if !isGCPSpeechPath(path, hasGCPSpeechHint(r)) {
		return false
	}

	if r.Method != http.MethodPost {
		return false
	}

	body, valid := decodeGCPSpeechJSONBody(w, r, path)
	if !valid {
		return true
	}

	switch path {
	case gcpSpeechRecognizePath:
		return handleGCPSpeechRecognize(w, path, body)
	case gcpSpeechLongRunningRecognizePath:
		return handleGCPSpeechLongRunningRecognize(w, path, body)
	case gcpSpeechStreamingRecognizePath:
		return handleGCPSpeechStreamingRecognize(w, path, body)
	case gcpSpeechCreatePhraseSetPath:
		return handleGCPSpeechCreatePhraseSet(w, path, body)
	case gcpSpeechGetPhraseSetPath:
		return handleGCPSpeechGetPhraseSet(w, path, body)
	case gcpSpeechListPhraseSetPath:
		return handleGCPSpeechListPhraseSet(w, path, body)
	case gcpSpeechUpdatePhraseSetPath:
		return handleGCPSpeechUpdatePhraseSet(w, path, body)
	case gcpSpeechDeletePhraseSetPath:
		return handleGCPSpeechDeletePhraseSet(w, path, body)
	case gcpSpeechCreateCustomClassPath:
		return handleGCPSpeechCreateCustomClass(w, path, body)
	case gcpSpeechGetCustomClassPath:
		return handleGCPSpeechGetCustomClass(w, path, body)
	case gcpSpeechListCustomClassesPath:
		return handleGCPSpeechListCustomClasses(w, path, body)
	case gcpSpeechUpdateCustomClassPath:
		return handleGCPSpeechUpdateCustomClass(w, path, body)
	case gcpSpeechDeleteCustomClassPath:
		return handleGCPSpeechDeleteCustomClass(w, path, body)
	default:
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	}
}

func normalizeGCPSpeechPath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.ReplaceAll(path, "%3A", ":")
	path = strings.ReplaceAll(path, "%3a", ":")
	return path
}

func hasGCPSpeechHint(r *http.Request) bool {
	service := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Stackyard-GCP-Service")))
	switch service {
	case "speech", "speech-apiv1", "speech_apiv1", "cloud-speech", "cloud_speech", "cloudspeech", "speech-to-text", "speech_to_text", "cloud-speech-to-text", "gcp-cloud-speech":
		return true
	}
	ua := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	return strings.Contains(ua, "stackyard-speech-apiv1") || strings.Contains(ua, "cloud.google.com/go/speech")
}

func isGCPSpeechLocationRequest(r *http.Request, path string) bool {
	if !hasGCPSpeechHint(r) {
		return false
	}
	_, _, _, ok := parseGCPProjectLocationPath(path)
	return ok
}

func isGCPSpeechPath(path string, includeHint bool) bool {
	switch path {
	case gcpSpeechRecognizePath,
		gcpSpeechLongRunningRecognizePath,
		gcpSpeechStreamingRecognizePath,
		gcpSpeechCreatePhraseSetPath,
		gcpSpeechGetPhraseSetPath,
		gcpSpeechListPhraseSetPath,
		gcpSpeechUpdatePhraseSetPath,
		gcpSpeechDeletePhraseSetPath,
		gcpSpeechCreateCustomClassPath,
		gcpSpeechGetCustomClassPath,
		gcpSpeechListCustomClassesPath,
		gcpSpeechUpdateCustomClassPath,
		gcpSpeechDeleteCustomClassPath:
		return true
	}
	if !includeHint {
		return false
	}
	return strings.HasPrefix(path, "/gcp/google.cloud.speech.v1.Speech/") || strings.HasPrefix(path, "/gcp/google.cloud.speech.v1.Adaptation/")
}

func decodeGCPSpeechJSONBody(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
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
		respondGCPSpeechInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if body == nil {
		body = map[string]any{}
	}
	return body, true
}

func handleGCPSpeechListLocations(w http.ResponseWriter, r *http.Request, path string) bool {
	project, _, list, ok := parseGCPProjectLocationPath(path)
	if !ok || !list {
		return false
	}

	pageSize, start, valid := parseGCPSpeechQueryPagination(w, r, path, 100)
	if !valid {
		return true
	}

	items := []map[string]any{
		gcpSpeechLocation(project, "global"),
		gcpSpeechLocation(project, "us-central1"),
	}
	return respondGCPSpeechList(w, "locations", items, pageSize, start, path)
}

func handleGCPSpeechGetLocation(w http.ResponseWriter, path string) bool {
	project, location, list, ok := parseGCPProjectLocationPath(path)
	if !ok || list {
		return false
	}
	respondJSON(w, http.StatusOK, gcpSpeechLocation(project, location))
	return true
}

func handleGCPSpeechRecognize(w http.ResponseWriter, path string, body map[string]any) bool {
	if len(body) == 0 {
		respondGCPSpeechInvalidArgument(w, path, "recognize request is required")
		return true
	}

	config := gcpSpeechBodyMap(body, "config")
	if len(config) == 0 {
		respondGCPSpeechInvalidArgument(w, path, "config is required")
		return true
	}
	audio := gcpSpeechBodyMap(body, "audio")
	if len(audio) == 0 {
		respondGCPSpeechInvalidArgument(w, path, "audio is required")
		return true
	}

	languageCode := strings.TrimSpace(gcpSpeechString(config, "languageCode", "language_code"))
	if languageCode == "" {
		respondGCPSpeechInvalidArgument(w, path, "config.languageCode is required")
		return true
	}
	if strings.EqualFold(strings.TrimSpace(gcpSpeechString(config, "encoding")), "ENCODING_UNSPECIFIED") {
		respondGCPSpeechInvalidArgument(w, path, "config.encoding is invalid")
		return true
	}
	if sampleRate, ok := gcpSpeechInt(config, "sampleRateHertz", "sample_rate_hertz"); ok && sampleRate <= 0 {
		respondGCPSpeechInvalidArgument(w, path, "config.sampleRateHertz must be positive")
		return true
	}

	content := strings.TrimSpace(gcpSpeechString(audio, "content"))
	uri := strings.TrimSpace(gcpSpeechString(audio, "uri"))
	if content == "" && uri == "" {
		respondGCPSpeechInvalidArgument(w, path, "audio.content or audio.uri is required")
		return true
	}
	if content != "" && uri != "" {
		respondGCPSpeechInvalidArgument(w, path, "audio must provide only one source")
		return true
	}
	if uri != "" && strings.Contains(strings.ToLower(uri), "missing") {
		respondGCPSpeechNotFound(w, path, "audio uri not found")
		return true
	}

	respondJSON(w, http.StatusOK, gcpSpeechRecognizeResponse(languageCode, "stackyard recognized speech"))
	return true
}

func handleGCPSpeechLongRunningRecognize(w http.ResponseWriter, path string, body map[string]any) bool {
	if len(body) == 0 {
		respondGCPSpeechInvalidArgument(w, path, "long running recognize request is required")
		return true
	}

	config := gcpSpeechBodyMap(body, "config")
	if len(config) == 0 {
		respondGCPSpeechInvalidArgument(w, path, "config is required")
		return true
	}
	audio := gcpSpeechBodyMap(body, "audio")
	if len(audio) == 0 {
		respondGCPSpeechInvalidArgument(w, path, "audio is required")
		return true
	}

	languageCode := strings.TrimSpace(gcpSpeechString(config, "languageCode", "language_code"))
	if languageCode == "" {
		respondGCPSpeechInvalidArgument(w, path, "config.languageCode is required")
		return true
	}
	content := strings.TrimSpace(gcpSpeechString(audio, "content"))
	uri := strings.TrimSpace(gcpSpeechString(audio, "uri"))
	if content == "" && uri == "" {
		respondGCPSpeechInvalidArgument(w, path, "audio.content or audio.uri is required")
		return true
	}
	if content != "" && uri != "" {
		respondGCPSpeechInvalidArgument(w, path, "audio must provide only one source")
		return true
	}
	if uri != "" && strings.Contains(strings.ToLower(uri), "missing") {
		respondGCPSpeechNotFound(w, path, "audio uri not found")
		return true
	}

	respondJSON(w, http.StatusOK, gcpSpeechLongRunningOperation("stackyard", "global", "speech.longRunningRecognize.op-1", languageCode, "stackyard long running recognized speech"))
	return true
}

func handleGCPSpeechStreamingRecognize(w http.ResponseWriter, path string, body map[string]any) bool {
	if len(body) == 0 {
		respondGCPSpeechInvalidArgument(w, path, "streaming request payload is required")
		return true
	}

	streamingConfig := gcpSpeechBodyMap(body, "streamingConfig", "streaming_config")
	if len(streamingConfig) == 0 {
		respondGCPSpeechInvalidArgument(w, path, "streamingConfig is required before audio content")
		return true
	}

	config := gcpSpeechBodyMap(streamingConfig, "config")
	if len(config) == 0 {
		respondGCPSpeechInvalidArgument(w, path, "streamingConfig.config is required")
		return true
	}
	languageCode := strings.TrimSpace(gcpSpeechString(config, "languageCode", "language_code"))
	if languageCode == "" {
		respondGCPSpeechInvalidArgument(w, path, "streamingConfig.config.languageCode is required")
		return true
	}

	if sampleRate, ok := gcpSpeechInt(config, "sampleRateHertz", "sample_rate_hertz"); ok && sampleRate <= 0 {
		respondGCPSpeechInvalidArgument(w, path, "streamingConfig.config.sampleRateHertz must be positive")
		return true
	}

	respondJSON(w, http.StatusOK, gcpSpeechStreamingRecognizeResponse(languageCode, "stackyard streaming recognized speech"))
	return true
}

func handleGCPSpeechCreatePhraseSet(w http.ResponseWriter, path string, body map[string]any) bool {
	parent := strings.TrimSpace(gcpSpeechString(body, "parent"))
	project, location, ok := parseGCPSpeechParentName(parent)
	if !ok {
		respondGCPSpeechInvalidArgument(w, path, "parent is required")
		return true
	}

	phraseSetID := strings.TrimSpace(gcpSpeechString(body, "phraseSetId", "phrase_set_id"))
	if !isGCPSpeechResourceID(phraseSetID) {
		respondGCPSpeechInvalidArgument(w, path, "phraseSetId is required")
		return true
	}
	if strings.Contains(strings.ToLower(phraseSetID), "existing") {
		respondGCPSpeechAlreadyExists(w, path, "phrase set already exists")
		return true
	}

	phraseSetReq := gcpSpeechBodyMap(body, "phraseSet", "phrase_set")
	if len(phraseSetReq) == 0 {
		respondGCPSpeechInvalidArgument(w, path, "phraseSet is required")
		return true
	}

	expectedName := fmt.Sprintf("projects/%s/locations/%s/phraseSets/%s", project, location, phraseSetID)
	if providedName := strings.TrimSpace(gcpSpeechString(phraseSetReq, "name")); providedName != "" && providedName != expectedName {
		respondGCPSpeechInvalidArgument(w, path, "phraseSet.name must match parent and phraseSetId")
		return true
	}

	response := gcpSpeechPhraseSet(project, location, phraseSetID)
	gcpSpeechApplyPhraseSetOverrides(response, phraseSetReq)
	respondJSON(w, http.StatusOK, response)
	return true
}

func handleGCPSpeechGetPhraseSet(w http.ResponseWriter, path string, body map[string]any) bool {
	name := strings.TrimSpace(gcpSpeechString(body, "name"))
	project, location, phraseSetID, ok := parseGCPSpeechPhraseSetName(name)
	if !ok {
		respondGCPSpeechInvalidArgument(w, path, "name is required")
		return true
	}
	if strings.Contains(strings.ToLower(phraseSetID), "missing") {
		respondGCPSpeechNotFound(w, path, "phrase set not found")
		return true
	}

	respondJSON(w, http.StatusOK, gcpSpeechPhraseSet(project, location, phraseSetID))
	return true
}

func handleGCPSpeechListPhraseSet(w http.ResponseWriter, path string, body map[string]any) bool {
	parent := strings.TrimSpace(gcpSpeechString(body, "parent"))
	project, location, ok := parseGCPSpeechParentName(parent)
	if !ok {
		respondGCPSpeechInvalidArgument(w, path, "parent is required")
		return true
	}

	pageSize, start, valid := parseGCPSpeechBodyPagination(w, path, body, 50, 1000)
	if !valid {
		return true
	}

	items := []map[string]any{
		gcpSpeechPhraseSet(project, location, "phrase-set-1"),
		gcpSpeechPhraseSet(project, location, "phrase-set-2"),
	}
	return respondGCPSpeechList(w, "phraseSets", items, pageSize, start, path)
}

func handleGCPSpeechUpdatePhraseSet(w http.ResponseWriter, path string, body map[string]any) bool {
	phraseSetReq := gcpSpeechBodyMap(body, "phraseSet", "phrase_set")
	if len(phraseSetReq) == 0 {
		respondGCPSpeechInvalidArgument(w, path, "phraseSet is required")
		return true
	}
	name := strings.TrimSpace(gcpSpeechString(phraseSetReq, "name"))
	project, location, phraseSetID, ok := parseGCPSpeechPhraseSetName(name)
	if !ok {
		respondGCPSpeechInvalidArgument(w, path, "phraseSet.name is required")
		return true
	}
	if strings.Contains(strings.ToLower(phraseSetID), "missing") {
		respondGCPSpeechNotFound(w, path, "phrase set not found")
		return true
	}

	updateMask := strings.TrimSpace(gcpSpeechString(body, "updateMask", "update_mask"))
	if updateMask == "" {
		updateMaskMap := gcpSpeechBodyMap(body, "updateMask", "update_mask")
		if len(updateMaskMap) > 0 {
			if paths, ok := updateMaskMap["paths"].([]any); ok && len(paths) > 0 {
				var pathStrings []string
				for _, p := range paths {
					text := strings.TrimSpace(fmt.Sprint(p))
					if text != "" {
						pathStrings = append(pathStrings, text)
					}
				}
				updateMask = strings.Join(pathStrings, ",")
			}
		}
	}
	if updateMask == "" {
		respondGCPSpeechInvalidArgument(w, path, "updateMask is required")
		return true
	}
	if strings.Contains(strings.ToLower(updateMask), "name") {
		respondGCPSpeechFailedPrecondition(w, path, "name is immutable")
		return true
	}

	response := gcpSpeechPhraseSet(project, location, phraseSetID)
	gcpSpeechApplyPhraseSetOverrides(response, phraseSetReq)
	respondJSON(w, http.StatusOK, response)
	return true
}

func handleGCPSpeechDeletePhraseSet(w http.ResponseWriter, path string, body map[string]any) bool {
	name := strings.TrimSpace(gcpSpeechString(body, "name"))
	_, _, phraseSetID, ok := parseGCPSpeechPhraseSetName(name)
	if !ok {
		respondGCPSpeechInvalidArgument(w, path, "name is required")
		return true
	}
	if strings.Contains(strings.ToLower(phraseSetID), "missing") {
		respondGCPSpeechNotFound(w, path, "phrase set not found")
		return true
	}

	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPSpeechCreateCustomClass(w http.ResponseWriter, path string, body map[string]any) bool {
	parent := strings.TrimSpace(gcpSpeechString(body, "parent"))
	project, location, ok := parseGCPSpeechParentName(parent)
	if !ok {
		respondGCPSpeechInvalidArgument(w, path, "parent is required")
		return true
	}

	customClassID := strings.TrimSpace(gcpSpeechString(body, "customClassId", "custom_class_id"))
	if !isGCPSpeechResourceID(customClassID) {
		respondGCPSpeechInvalidArgument(w, path, "customClassId is required")
		return true
	}
	if strings.Contains(strings.ToLower(customClassID), "existing") {
		respondGCPSpeechAlreadyExists(w, path, "custom class already exists")
		return true
	}

	customClassReq := gcpSpeechBodyMap(body, "customClass", "custom_class")
	if len(customClassReq) == 0 {
		respondGCPSpeechInvalidArgument(w, path, "customClass is required")
		return true
	}

	expectedName := fmt.Sprintf("projects/%s/locations/%s/customClasses/%s", project, location, customClassID)
	if providedName := strings.TrimSpace(gcpSpeechString(customClassReq, "name")); providedName != "" && providedName != expectedName {
		respondGCPSpeechInvalidArgument(w, path, "customClass.name must match parent and customClassId")
		return true
	}

	response := gcpSpeechCustomClass(project, location, customClassID)
	gcpSpeechApplyCustomClassOverrides(response, customClassReq)
	respondJSON(w, http.StatusOK, response)
	return true
}

func handleGCPSpeechGetCustomClass(w http.ResponseWriter, path string, body map[string]any) bool {
	name := strings.TrimSpace(gcpSpeechString(body, "name"))
	project, location, customClassID, ok := parseGCPSpeechCustomClassName(name)
	if !ok {
		respondGCPSpeechInvalidArgument(w, path, "name is required")
		return true
	}
	if strings.Contains(strings.ToLower(customClassID), "missing") {
		respondGCPSpeechNotFound(w, path, "custom class not found")
		return true
	}

	respondJSON(w, http.StatusOK, gcpSpeechCustomClass(project, location, customClassID))
	return true
}

func handleGCPSpeechListCustomClasses(w http.ResponseWriter, path string, body map[string]any) bool {
	parent := strings.TrimSpace(gcpSpeechString(body, "parent"))
	project, location, ok := parseGCPSpeechParentName(parent)
	if !ok {
		respondGCPSpeechInvalidArgument(w, path, "parent is required")
		return true
	}

	pageSize, start, valid := parseGCPSpeechBodyPagination(w, path, body, 50, 1000)
	if !valid {
		return true
	}

	items := []map[string]any{
		gcpSpeechCustomClass(project, location, "custom-class-1"),
		gcpSpeechCustomClass(project, location, "custom-class-2"),
	}
	return respondGCPSpeechList(w, "customClasses", items, pageSize, start, path)
}

func handleGCPSpeechUpdateCustomClass(w http.ResponseWriter, path string, body map[string]any) bool {
	customClassReq := gcpSpeechBodyMap(body, "customClass", "custom_class")
	if len(customClassReq) == 0 {
		respondGCPSpeechInvalidArgument(w, path, "customClass is required")
		return true
	}
	name := strings.TrimSpace(gcpSpeechString(customClassReq, "name"))
	project, location, customClassID, ok := parseGCPSpeechCustomClassName(name)
	if !ok {
		respondGCPSpeechInvalidArgument(w, path, "customClass.name is required")
		return true
	}
	if strings.Contains(strings.ToLower(customClassID), "missing") {
		respondGCPSpeechNotFound(w, path, "custom class not found")
		return true
	}

	updateMask := strings.TrimSpace(gcpSpeechString(body, "updateMask", "update_mask"))
	if updateMask == "" {
		updateMaskMap := gcpSpeechBodyMap(body, "updateMask", "update_mask")
		if len(updateMaskMap) > 0 {
			if paths, ok := updateMaskMap["paths"].([]any); ok && len(paths) > 0 {
				var pathStrings []string
				for _, p := range paths {
					text := strings.TrimSpace(fmt.Sprint(p))
					if text != "" {
						pathStrings = append(pathStrings, text)
					}
				}
				updateMask = strings.Join(pathStrings, ",")
			}
		}
	}
	if updateMask == "" {
		respondGCPSpeechInvalidArgument(w, path, "updateMask is required")
		return true
	}
	if strings.Contains(strings.ToLower(updateMask), "name") {
		respondGCPSpeechFailedPrecondition(w, path, "name is immutable")
		return true
	}

	response := gcpSpeechCustomClass(project, location, customClassID)
	gcpSpeechApplyCustomClassOverrides(response, customClassReq)
	respondJSON(w, http.StatusOK, response)
	return true
}

func handleGCPSpeechDeleteCustomClass(w http.ResponseWriter, path string, body map[string]any) bool {
	name := strings.TrimSpace(gcpSpeechString(body, "name"))
	_, _, customClassID, ok := parseGCPSpeechCustomClassName(name)
	if !ok {
		respondGCPSpeechInvalidArgument(w, path, "name is required")
		return true
	}
	if strings.Contains(strings.ToLower(customClassID), "missing") {
		respondGCPSpeechNotFound(w, path, "custom class not found")
		return true
	}

	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func gcpSpeechRecognizeResponse(languageCode, transcript string) map[string]any {
	return map[string]any{
		"results": []any{
			map[string]any{
				"alternatives": []any{
					map[string]any{
						"transcript": transcript,
						"confidence": 0.98,
					},
				},
				"channelTag":    1,
				"languageCode":  languageCode,
				"resultEndTime": "1.200s",
			},
		},
		"totalBilledTime": "1.200s",
		"requestId":       "speech-req-1",
	}
}

func gcpSpeechLongRunningOperation(project, location, operationID, languageCode, transcript string) map[string]any {
	return map[string]any{
		"name": fmt.Sprintf("projects/%s/locations/%s/operations/%s", project, location, operationID),
		"metadata": map[string]any{
			"@type":           "type.googleapis.com/google.cloud.speech.v1.LongRunningRecognizeMetadata",
			"progressPercent": 100,
			"startTime":       gcpSpeechReferenceTime.Add(-2 * time.Second).Format(time.RFC3339Nano),
			"lastUpdateTime":  gcpSpeechReferenceTime.Format(time.RFC3339Nano),
		},
		"done": true,
		"response": map[string]any{
			"@type":           "type.googleapis.com/google.cloud.speech.v1.LongRunningRecognizeResponse",
			"results":         gcpSpeechRecognizeResponse(languageCode, transcript)["results"],
			"totalBilledTime": "1.200s",
		},
	}
}

func gcpSpeechStreamingRecognizeResponse(languageCode, transcript string) map[string]any {
	return map[string]any{
		"results": []any{
			map[string]any{
				"alternatives": []any{
					map[string]any{
						"transcript": transcript,
						"confidence": 0.93,
					},
				},
				"isFinal":       true,
				"stability":     0.91,
				"resultEndTime": "0.900s",
				"languageCode":  languageCode,
			},
		},
		"speechEventType": "END_OF_SINGLE_UTTERANCE",
	}
}

func gcpSpeechLocation(project, location string) map[string]any {
	return map[string]any{
		"name":        fmt.Sprintf("projects/%s/locations/%s", project, location),
		"locationId":  location,
		"displayName": strings.ToUpper(location),
		"labels": map[string]string{
			"cloud.googleapis.com/region": location,
		},
	}
}

func gcpSpeechPhraseSet(project, location, phraseSetID string) map[string]any {
	return map[string]any{
		"name": fmt.Sprintf("projects/%s/locations/%s/phraseSets/%s", project, location, phraseSetID),
		"phrases": []any{
			map[string]any{"value": "stackyard"},
			map[string]any{"value": "cloud emulation"},
		},
		"boost": 12.5,
	}
}

func gcpSpeechCustomClass(project, location, customClassID string) map[string]any {
	return map[string]any{
		"name":          fmt.Sprintf("projects/%s/locations/%s/customClasses/%s", project, location, customClassID),
		"customClassId": customClassID,
		"items": []any{
			map[string]any{"value": "stackyard"},
			map[string]any{"value": "speech"},
		},
	}
}

func gcpSpeechApplyPhraseSetOverrides(response, request map[string]any) {
	if response == nil || request == nil {
		return
	}

	if boost, ok := gcpSpeechFloat(request, "boost"); ok {
		response["boost"] = boost
	}
	if rawPhrases, ok := request["phrases"].([]any); ok && len(rawPhrases) > 0 {
		phrases := make([]any, 0, len(rawPhrases))
		for _, rawPhrase := range rawPhrases {
			phrase, _ := rawPhrase.(map[string]any)
			value := strings.TrimSpace(gcpSpeechString(phrase, "value"))
			if value == "" {
				continue
			}
			entry := map[string]any{"value": value}
			if boost, ok := gcpSpeechFloat(phrase, "boost"); ok {
				entry["boost"] = boost
			}
			phrases = append(phrases, entry)
		}
		if len(phrases) > 0 {
			response["phrases"] = phrases
		}
	}
}

func gcpSpeechApplyCustomClassOverrides(response, request map[string]any) {
	if response == nil || request == nil {
		return
	}

	if customClassID := strings.TrimSpace(gcpSpeechString(request, "customClassId", "custom_class_id")); customClassID != "" {
		response["customClassId"] = customClassID
	}
	if rawItems, ok := request["items"].([]any); ok && len(rawItems) > 0 {
		items := make([]any, 0, len(rawItems))
		for _, rawItem := range rawItems {
			item, _ := rawItem.(map[string]any)
			value := strings.TrimSpace(gcpSpeechString(item, "value"))
			if value == "" {
				continue
			}
			items = append(items, map[string]any{"value": value})
		}
		if len(items) > 0 {
			response["items"] = items
		}
	}
}

func parseGCPSpeechQueryPagination(w http.ResponseWriter, r *http.Request, path string, defaultPageSize int) (pageSize, start int, valid bool) {
	pageSize = defaultPageSize
	if raw := strings.TrimSpace(r.URL.Query().Get("pageSize")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 0 {
			respondGCPSpeechInvalidArgument(w, path, "pageSize must be a non-negative integer")
			return 0, 0, false
		}
		if value > 1000 {
			respondGCPSpeechInvalidArgument(w, path, "pageSize must be <= 1000")
			return 0, 0, false
		}
		pageSize = value
	}
	if raw := strings.TrimSpace(r.URL.Query().Get("pageToken")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 0 {
			respondGCPSpeechInvalidArgument(w, path, "pageToken must be a non-negative integer")
			return 0, 0, false
		}
		start = value
	}
	return pageSize, start, true
}

func parseGCPSpeechBodyPagination(w http.ResponseWriter, path string, body map[string]any, defaultPageSize, maxPageSize int) (pageSize, start int, valid bool) {
	pageSize = defaultPageSize
	if value, ok := gcpSpeechInt(body, "pageSize", "page_size"); ok {
		if value < 0 {
			respondGCPSpeechInvalidArgument(w, path, "pageSize must be a non-negative integer")
			return 0, 0, false
		}
		if value > maxPageSize {
			respondGCPSpeechInvalidArgument(w, path, fmt.Sprintf("pageSize must be <= %d", maxPageSize))
			return 0, 0, false
		}
		pageSize = value
	}

	token := strings.TrimSpace(gcpSpeechString(body, "pageToken", "page_token"))
	if token != "" {
		value, err := strconv.Atoi(token)
		if err != nil || value < 0 {
			respondGCPSpeechInvalidArgument(w, path, "pageToken must be a non-negative integer")
			return 0, 0, false
		}
		start = value
	}
	return pageSize, start, true
}

func respondGCPSpeechList(w http.ResponseWriter, field string, items []map[string]any, pageSize, start int, path string) bool {
	if pageSize < 0 || start < 0 {
		respondGCPSpeechInvalidArgument(w, path, "invalid pagination values")
		return false
	}
	if pageSize == 0 {
		pageSize = len(items)
	}
	if start > len(items) {
		start = len(items)
	}
	end := start + pageSize
	if end > len(items) {
		end = len(items)
	}

	pageItems := make([]any, 0, end-start)
	for _, item := range items[start:end] {
		pageItems = append(pageItems, item)
	}

	nextToken := ""
	if end < len(items) {
		nextToken = strconv.Itoa(end)
	}

	respondJSON(w, http.StatusOK, map[string]any{
		field:           pageItems,
		"nextPageToken": nextToken,
	})
	return true
}

func parseGCPSpeechParentName(name string) (project, location string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 4 || parts[0] != "projects" || parts[2] != "locations" {
		return "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	if project == "" || location == "" {
		return "", "", false
	}
	return project, location, true
}

func parseGCPSpeechPhraseSetName(name string) (project, location, phraseSetID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 6 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "phraseSets" {
		return "", "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	phraseSetID = strings.TrimSpace(parts[5])
	if project == "" || location == "" || !isGCPSpeechResourceID(phraseSetID) {
		return "", "", "", false
	}
	return project, location, phraseSetID, true
}

func parseGCPSpeechCustomClassName(name string) (project, location, customClassID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 6 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "customClasses" {
		return "", "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	customClassID = strings.TrimSpace(parts[5])
	if project == "" || location == "" || !isGCPSpeechResourceID(customClassID) {
		return "", "", "", false
	}
	return project, location, customClassID, true
}

func isGCPSpeechResourceID(id string) bool {
	id = strings.TrimSpace(id)
	if id == "" {
		return false
	}
	for _, ch := range id {
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '-' || ch == '_' || ch == '.' {
			continue
		}
		return false
	}
	return true
}

func gcpSpeechBodyMap(body map[string]any, keys ...string) map[string]any {
	for _, key := range keys {
		if value, ok := body[key].(map[string]any); ok {
			return value
		}
	}
	return map[string]any{}
}

func gcpSpeechString(body map[string]any, keys ...string) string {
	for _, key := range keys {
		raw, ok := body[key]
		if !ok {
			continue
		}
		switch value := raw.(type) {
		case string:
			return value
		case json.Number:
			return value.String()
		case float64:
			return strconv.FormatFloat(value, 'f', -1, 64)
		default:
			return fmt.Sprint(value)
		}
	}
	return ""
}

func gcpSpeechInt(body map[string]any, keys ...string) (int, bool) {
	for _, key := range keys {
		raw, ok := body[key]
		if !ok {
			continue
		}
		switch value := raw.(type) {
		case int:
			return value, true
		case int32:
			return int(value), true
		case int64:
			return int(value), true
		case float64:
			if value != float64(int(value)) {
				return 0, false
			}
			return int(value), true
		case json.Number:
			i, err := value.Int64()
			if err != nil {
				return 0, false
			}
			return int(i), true
		case string:
			i, err := strconv.Atoi(strings.TrimSpace(value))
			if err != nil {
				return 0, false
			}
			return i, true
		}
	}
	return 0, false
}

func gcpSpeechFloat(body map[string]any, keys ...string) (float64, bool) {
	for _, key := range keys {
		raw, ok := body[key]
		if !ok {
			continue
		}
		switch value := raw.(type) {
		case float64:
			return value, true
		case float32:
			return float64(value), true
		case json.Number:
			f, err := value.Float64()
			if err != nil {
				return 0, false
			}
			return f, true
		case string:
			f, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
			if err != nil {
				return 0, false
			}
			return f, true
		case int:
			return float64(value), true
		case int32:
			return float64(value), true
		case int64:
			return float64(value), true
		}
	}
	return 0, false
}

func respondGCPSpeechInvalidArgument(w http.ResponseWriter, path, message string) {
	respondGCPSpeechError(w, http.StatusBadRequest, "InvalidArgument", path, message)
}

func respondGCPSpeechNotFound(w http.ResponseWriter, path, message string) {
	respondGCPSpeechError(w, http.StatusNotFound, "NotFound", path, message)
}

func respondGCPSpeechFailedPrecondition(w http.ResponseWriter, path, message string) {
	respondGCPSpeechError(w, http.StatusBadRequest, "FailedPrecondition", path, message)
}

func respondGCPSpeechAlreadyExists(w http.ResponseWriter, path, message string) {
	respondGCPSpeechError(w, http.StatusConflict, "AlreadyExists", path, message)
}

func respondGCPSpeechError(w http.ResponseWriter, status int, errToken, path, message string) {
	respondJSON(w, status, map[string]any{
		"error":    errToken,
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func handleGCPContractProbe_speech(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "speech") {
		return false
	}
	if r.URL.Query().Get("pageSize") == "bad" {
		respondGCPSpeechInvalidArgument(w, path, "pageSize must be a non-negative integer")
		return true
	}
	if r.URL.Query().Get("alreadyExists") == "1" {
		respondGCPSpeechAlreadyExists(w, path, "resource already exists")
		return true
	}
	if r.URL.Query().Get("failedPrecondition") == "1" {
		respondGCPSpeechFailedPrecondition(w, path, "precondition failed")
		return true
	}
	if r.URL.Query().Get("typedSuccess") == "1" {
		respondJSON(w, http.StatusOK, map[string]any{
			"name":     "projects/stackyard/locations/us-central1/speech/sample",
			"service":  "speech",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}
	return false
}
