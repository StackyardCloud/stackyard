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
	gcpSpeechV2CreateRecognizerPath   = "/gcp/google.cloud.speech.v2.Speech/CreateRecognizer"
	gcpSpeechV2ListRecognizersPath    = "/gcp/google.cloud.speech.v2.Speech/ListRecognizers"
	gcpSpeechV2GetRecognizerPath      = "/gcp/google.cloud.speech.v2.Speech/GetRecognizer"
	gcpSpeechV2UpdateRecognizerPath   = "/gcp/google.cloud.speech.v2.Speech/UpdateRecognizer"
	gcpSpeechV2DeleteRecognizerPath   = "/gcp/google.cloud.speech.v2.Speech/DeleteRecognizer"
	gcpSpeechV2UndeleteRecognizerPath = "/gcp/google.cloud.speech.v2.Speech/UndeleteRecognizer"

	gcpSpeechV2RecognizePath          = "/gcp/google.cloud.speech.v2.Speech/Recognize"
	gcpSpeechV2StreamingRecognizePath = "/gcp/google.cloud.speech.v2.Speech/StreamingRecognize"
	gcpSpeechV2BatchRecognizePath     = "/gcp/google.cloud.speech.v2.Speech/BatchRecognize"

	gcpSpeechV2GetConfigPath    = "/gcp/google.cloud.speech.v2.Speech/GetConfig"
	gcpSpeechV2UpdateConfigPath = "/gcp/google.cloud.speech.v2.Speech/UpdateConfig"

	gcpSpeechV2CreateCustomClassPath   = "/gcp/google.cloud.speech.v2.Speech/CreateCustomClass"
	gcpSpeechV2ListCustomClassesPath   = "/gcp/google.cloud.speech.v2.Speech/ListCustomClasses"
	gcpSpeechV2GetCustomClassPath      = "/gcp/google.cloud.speech.v2.Speech/GetCustomClass"
	gcpSpeechV2UpdateCustomClassPath   = "/gcp/google.cloud.speech.v2.Speech/UpdateCustomClass"
	gcpSpeechV2DeleteCustomClassPath   = "/gcp/google.cloud.speech.v2.Speech/DeleteCustomClass"
	gcpSpeechV2UndeleteCustomClassPath = "/gcp/google.cloud.speech.v2.Speech/UndeleteCustomClass"

	gcpSpeechV2CreatePhraseSetPath   = "/gcp/google.cloud.speech.v2.Speech/CreatePhraseSet"
	gcpSpeechV2ListPhraseSetsPath    = "/gcp/google.cloud.speech.v2.Speech/ListPhraseSets"
	gcpSpeechV2GetPhraseSetPath      = "/gcp/google.cloud.speech.v2.Speech/GetPhraseSet"
	gcpSpeechV2UpdatePhraseSetPath   = "/gcp/google.cloud.speech.v2.Speech/UpdatePhraseSet"
	gcpSpeechV2DeletePhraseSetPath   = "/gcp/google.cloud.speech.v2.Speech/DeletePhraseSet"
	gcpSpeechV2UndeletePhraseSetPath = "/gcp/google.cloud.speech.v2.Speech/UndeletePhraseSet"
)

var gcpSpeechV2ReferenceTime = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

func (s *Server) handleGCPSpeechV2Router(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_speech_v2(w, r) {
		return true
	}

	path := normalizeGCPSpeechV2Path(rawRequestPath(r))
	if isGCPSpeechV2LocationRequest(r, path) {
		if r.Method != http.MethodGet {
			return false
		}
		if handleGCPSpeechV2ListLocations(w, r, path) {
			return true
		}
		if handleGCPSpeechV2GetLocation(w, path) {
			return true
		}
		return false
	}

	if !isGCPSpeechV2Path(path, hasGCPSpeechV2Hint(r)) {
		return false
	}
	if r.Method != http.MethodPost {
		return false
	}

	body, valid := decodeGCPSpeechV2JSONBody(w, r, path)
	if !valid {
		return true
	}

	switch path {
	case gcpSpeechV2CreateRecognizerPath:
		return handleGCPSpeechV2CreateRecognizer(w, path, body)
	case gcpSpeechV2ListRecognizersPath:
		return handleGCPSpeechV2ListRecognizers(w, path, body)
	case gcpSpeechV2GetRecognizerPath:
		return handleGCPSpeechV2GetRecognizer(w, path, body)
	case gcpSpeechV2UpdateRecognizerPath:
		return handleGCPSpeechV2UpdateRecognizer(w, path, body)
	case gcpSpeechV2DeleteRecognizerPath:
		return handleGCPSpeechV2DeleteRecognizer(w, path, body)
	case gcpSpeechV2UndeleteRecognizerPath:
		return handleGCPSpeechV2UndeleteRecognizer(w, path, body)
	case gcpSpeechV2RecognizePath:
		return handleGCPSpeechV2Recognize(w, path, body)
	case gcpSpeechV2StreamingRecognizePath:
		return handleGCPSpeechV2StreamingRecognize(w, path, body)
	case gcpSpeechV2BatchRecognizePath:
		return handleGCPSpeechV2BatchRecognize(w, path, body)
	case gcpSpeechV2GetConfigPath:
		return handleGCPSpeechV2GetConfig(w, path, body)
	case gcpSpeechV2UpdateConfigPath:
		return handleGCPSpeechV2UpdateConfig(w, path, body)
	case gcpSpeechV2CreateCustomClassPath:
		return handleGCPSpeechV2CreateCustomClass(w, path, body)
	case gcpSpeechV2ListCustomClassesPath:
		return handleGCPSpeechV2ListCustomClasses(w, path, body)
	case gcpSpeechV2GetCustomClassPath:
		return handleGCPSpeechV2GetCustomClass(w, path, body)
	case gcpSpeechV2UpdateCustomClassPath:
		return handleGCPSpeechV2UpdateCustomClass(w, path, body)
	case gcpSpeechV2DeleteCustomClassPath:
		return handleGCPSpeechV2DeleteCustomClass(w, path, body)
	case gcpSpeechV2UndeleteCustomClassPath:
		return handleGCPSpeechV2UndeleteCustomClass(w, path, body)
	case gcpSpeechV2CreatePhraseSetPath:
		return handleGCPSpeechV2CreatePhraseSet(w, path, body)
	case gcpSpeechV2ListPhraseSetsPath:
		return handleGCPSpeechV2ListPhraseSets(w, path, body)
	case gcpSpeechV2GetPhraseSetPath:
		return handleGCPSpeechV2GetPhraseSet(w, path, body)
	case gcpSpeechV2UpdatePhraseSetPath:
		return handleGCPSpeechV2UpdatePhraseSet(w, path, body)
	case gcpSpeechV2DeletePhraseSetPath:
		return handleGCPSpeechV2DeletePhraseSet(w, path, body)
	case gcpSpeechV2UndeletePhraseSetPath:
		return handleGCPSpeechV2UndeletePhraseSet(w, path, body)
	default:
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	}
}

func normalizeGCPSpeechV2Path(path string) string {
	path = strings.TrimSpace(path)
	path = strings.ReplaceAll(path, "%3A", ":")
	path = strings.ReplaceAll(path, "%3a", ":")
	return path
}

func hasGCPSpeechV2Hint(r *http.Request) bool {
	service := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Stackyard-GCP-Service")))
	switch service {
	case "speech_v2", "speech-v2", "speech-apiv2", "speech_apiv2", "cloud-speech-v2", "cloud_speech_v2", "cloud-speech-to-text-v2", "cloud_speech_to_text_v2", "gcp-cloud-speech-v2":
		return true
	}
	ua := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	return strings.Contains(ua, "stackyard-speech-apiv2") || strings.Contains(ua, "cloud.google.com/go/speech/apiv2")
}

func isGCPSpeechV2LocationRequest(r *http.Request, path string) bool {
	if !hasGCPSpeechV2Hint(r) {
		return false
	}
	_, _, _, ok := parseGCPSpeechV2ProjectLocationPath(path)
	return ok
}

func isGCPSpeechV2Path(path string, includeHint bool) bool {
	switch path {
	case gcpSpeechV2CreateRecognizerPath,
		gcpSpeechV2ListRecognizersPath,
		gcpSpeechV2GetRecognizerPath,
		gcpSpeechV2UpdateRecognizerPath,
		gcpSpeechV2DeleteRecognizerPath,
		gcpSpeechV2UndeleteRecognizerPath,
		gcpSpeechV2RecognizePath,
		gcpSpeechV2StreamingRecognizePath,
		gcpSpeechV2BatchRecognizePath,
		gcpSpeechV2GetConfigPath,
		gcpSpeechV2UpdateConfigPath,
		gcpSpeechV2CreateCustomClassPath,
		gcpSpeechV2ListCustomClassesPath,
		gcpSpeechV2GetCustomClassPath,
		gcpSpeechV2UpdateCustomClassPath,
		gcpSpeechV2DeleteCustomClassPath,
		gcpSpeechV2UndeleteCustomClassPath,
		gcpSpeechV2CreatePhraseSetPath,
		gcpSpeechV2ListPhraseSetsPath,
		gcpSpeechV2GetPhraseSetPath,
		gcpSpeechV2UpdatePhraseSetPath,
		gcpSpeechV2DeletePhraseSetPath,
		gcpSpeechV2UndeletePhraseSetPath:
		return true
	}
	if !includeHint {
		return false
	}
	return strings.HasPrefix(path, "/gcp/google.cloud.speech.v2.Speech/")
}

func decodeGCPSpeechV2JSONBody(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
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
		respondGCPSpeechV2InvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if body == nil {
		body = map[string]any{}
	}
	return body, true
}

func handleGCPSpeechV2ListLocations(w http.ResponseWriter, r *http.Request, path string) bool {
	project, _, list, ok := parseGCPSpeechV2ProjectLocationPath(path)
	if !ok || !list {
		return false
	}
	pageSize, start, valid := parseGCPSpeechV2QueryPagination(w, r, path, 100, 1000)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpSpeechV2Location(project, "global"),
		gcpSpeechV2Location(project, "us-central1"),
	}
	return respondGCPSpeechV2List(w, "locations", items, pageSize, start, path)
}

func handleGCPSpeechV2GetLocation(w http.ResponseWriter, path string) bool {
	project, location, list, ok := parseGCPSpeechV2ProjectLocationPath(path)
	if !ok || list {
		return false
	}
	respondJSON(w, http.StatusOK, gcpSpeechV2Location(project, location))
	return true
}

func handleGCPSpeechV2CreateRecognizer(w http.ResponseWriter, path string, body map[string]any) bool {
	parent := strings.TrimSpace(gcpSpeechString(body, "parent"))
	project, location, ok := parseGCPSpeechParentName(parent)
	if !ok {
		respondGCPSpeechV2InvalidArgument(w, path, "parent is required")
		return true
	}
	recognizerID := strings.TrimSpace(gcpSpeechString(body, "recognizerId", "recognizer_id"))
	if !isGCPSpeechV2ResourceID(recognizerID) {
		respondGCPSpeechV2InvalidArgument(w, path, "recognizerId is required")
		return true
	}
	if strings.Contains(strings.ToLower(recognizerID), "existing") {
		respondGCPSpeechV2AlreadyExists(w, path, "recognizer already exists")
		return true
	}
	recognizerReq := gcpSpeechBodyMap(body, "recognizer")
	if len(recognizerReq) == 0 {
		respondGCPSpeechV2InvalidArgument(w, path, "recognizer is required")
		return true
	}
	expectedName := fmt.Sprintf("projects/%s/locations/%s/recognizers/%s", project, location, recognizerID)
	if providedName := strings.TrimSpace(gcpSpeechString(recognizerReq, "name")); providedName != "" && providedName != expectedName {
		respondGCPSpeechV2InvalidArgument(w, path, "recognizer.name must match parent and recognizerId")
		return true
	}

	recognizer := gcpSpeechV2Recognizer(project, location, recognizerID, false)
	gcpSpeechV2ApplyRecognizerOverrides(recognizer, recognizerReq)
	respondJSON(w, http.StatusOK, gcpSpeechV2Operation(project, location, "createRecognizer."+recognizerID, "google.cloud.speech.v2.Speech.CreateRecognizer", expectedName, "type.googleapis.com/google.cloud.speech.v2.Recognizer", recognizer))
	return true
}

func handleGCPSpeechV2ListRecognizers(w http.ResponseWriter, path string, body map[string]any) bool {
	parent := strings.TrimSpace(gcpSpeechString(body, "parent"))
	project, location, ok := parseGCPSpeechParentName(parent)
	if !ok {
		respondGCPSpeechV2InvalidArgument(w, path, "parent is required")
		return true
	}
	pageSize, start, valid := parseGCPSpeechV2BodyPagination(w, path, body, 5, 100)
	if !valid {
		return true
	}
	showDeleted, _ := gcpSpeechV2Bool(body, "showDeleted", "show_deleted")
	items := []map[string]any{
		gcpSpeechV2Recognizer(project, location, "recognizer-1", false),
		gcpSpeechV2Recognizer(project, location, "recognizer-deleted", true),
	}
	if !showDeleted {
		items = items[:1]
	}
	return respondGCPSpeechV2List(w, "recognizers", items, pageSize, start, path)
}

func handleGCPSpeechV2GetRecognizer(w http.ResponseWriter, path string, body map[string]any) bool {
	name := strings.TrimSpace(gcpSpeechString(body, "name"))
	project, location, recognizerID, ok := parseGCPSpeechV2RecognizerName(name)
	if !ok {
		respondGCPSpeechV2InvalidArgument(w, path, "name is required")
		return true
	}
	if strings.Contains(strings.ToLower(recognizerID), "missing") {
		respondGCPSpeechV2NotFound(w, path, "recognizer not found")
		return true
	}
	respondJSON(w, http.StatusOK, gcpSpeechV2Recognizer(project, location, recognizerID, strings.Contains(strings.ToLower(recognizerID), "deleted")))
	return true
}

func handleGCPSpeechV2UpdateRecognizer(w http.ResponseWriter, path string, body map[string]any) bool {
	recognizerReq := gcpSpeechBodyMap(body, "recognizer")
	if len(recognizerReq) == 0 {
		respondGCPSpeechV2InvalidArgument(w, path, "recognizer is required")
		return true
	}
	project, location, recognizerID, ok := parseGCPSpeechV2RecognizerName(strings.TrimSpace(gcpSpeechString(recognizerReq, "name")))
	if !ok {
		respondGCPSpeechV2InvalidArgument(w, path, "recognizer.name is required")
		return true
	}
	if strings.Contains(strings.ToLower(recognizerID), "missing") {
		respondGCPSpeechV2NotFound(w, path, "recognizer not found")
		return true
	}
	updateMask := gcpSpeechV2ExtractUpdateMask(body)
	if updateMask == "" {
		respondGCPSpeechV2InvalidArgument(w, path, "updateMask is required")
		return true
	}
	if gcpSpeechV2MaskContains(updateMask, "name") {
		respondGCPSpeechV2FailedPrecondition(w, path, "recognizer.name is immutable")
		return true
	}

	expectedEtag := gcpSpeechV2Etag(recognizerID)
	if providedEtag := strings.TrimSpace(gcpSpeechString(recognizerReq, "etag")); providedEtag != "" && providedEtag != expectedEtag {
		respondGCPSpeechV2FailedPrecondition(w, path, "etag mismatch")
		return true
	}

	recognizer := gcpSpeechV2Recognizer(project, location, recognizerID, false)
	gcpSpeechV2ApplyRecognizerOverrides(recognizer, recognizerReq)
	respondJSON(w, http.StatusOK, gcpSpeechV2Operation(project, location, "updateRecognizer."+recognizerID, "google.cloud.speech.v2.Speech.UpdateRecognizer", recognizer["name"].(string), "type.googleapis.com/google.cloud.speech.v2.Recognizer", recognizer))
	return true
}

func handleGCPSpeechV2DeleteRecognizer(w http.ResponseWriter, path string, body map[string]any) bool {
	name := strings.TrimSpace(gcpSpeechString(body, "name"))
	project, location, recognizerID, ok := parseGCPSpeechV2RecognizerName(name)
	if !ok {
		respondGCPSpeechV2InvalidArgument(w, path, "name is required")
		return true
	}
	allowMissing, _ := gcpSpeechV2Bool(body, "allowMissing", "allow_missing")
	if strings.Contains(strings.ToLower(recognizerID), "missing") && !allowMissing {
		respondGCPSpeechV2NotFound(w, path, "recognizer not found")
		return true
	}
	if providedEtag := strings.TrimSpace(gcpSpeechString(body, "etag")); providedEtag != "" && providedEtag != gcpSpeechV2Etag(recognizerID) {
		respondGCPSpeechV2FailedPrecondition(w, path, "etag mismatch")
		return true
	}

	recognizer := gcpSpeechV2Recognizer(project, location, recognizerID, true)
	respondJSON(w, http.StatusOK, gcpSpeechV2Operation(project, location, "deleteRecognizer."+recognizerID, "google.cloud.speech.v2.Speech.DeleteRecognizer", recognizer["name"].(string), "type.googleapis.com/google.cloud.speech.v2.Recognizer", recognizer))
	return true
}

func handleGCPSpeechV2UndeleteRecognizer(w http.ResponseWriter, path string, body map[string]any) bool {
	name := strings.TrimSpace(gcpSpeechString(body, "name"))
	project, location, recognizerID, ok := parseGCPSpeechV2RecognizerName(name)
	if !ok {
		respondGCPSpeechV2InvalidArgument(w, path, "name is required")
		return true
	}
	if strings.Contains(strings.ToLower(recognizerID), "missing") {
		respondGCPSpeechV2NotFound(w, path, "recognizer not found")
		return true
	}
	if providedEtag := strings.TrimSpace(gcpSpeechString(body, "etag")); providedEtag != "" && providedEtag != gcpSpeechV2Etag(recognizerID) {
		respondGCPSpeechV2FailedPrecondition(w, path, "etag mismatch")
		return true
	}
	recognizer := gcpSpeechV2Recognizer(project, location, recognizerID, false)
	respondJSON(w, http.StatusOK, gcpSpeechV2Operation(project, location, "undeleteRecognizer."+recognizerID, "google.cloud.speech.v2.Speech.UndeleteRecognizer", recognizer["name"].(string), "type.googleapis.com/google.cloud.speech.v2.Recognizer", recognizer))
	return true
}

func handleGCPSpeechV2Recognize(w http.ResponseWriter, path string, body map[string]any) bool {
	recognizer := strings.TrimSpace(gcpSpeechString(body, "recognizer"))
	project, location, _, ok := parseGCPSpeechV2RecognizerNameOrImplicit(recognizer)
	if !ok {
		respondGCPSpeechV2InvalidArgument(w, path, "recognizer is required")
		return true
	}
	content := strings.TrimSpace(gcpSpeechString(body, "content"))
	uri := strings.TrimSpace(gcpSpeechString(body, "uri"))
	if content == "" && uri == "" {
		respondGCPSpeechV2InvalidArgument(w, path, "content or uri is required")
		return true
	}
	if content != "" && uri != "" {
		respondGCPSpeechV2InvalidArgument(w, path, "content and uri are mutually exclusive")
		return true
	}
	if uri != "" && strings.Contains(strings.ToLower(uri), "missing") {
		respondGCPSpeechV2NotFound(w, path, "audio uri not found")
		return true
	}

	config := gcpSpeechBodyMap(body, "config")
	languageCode := gcpSpeechV2FirstLanguageCode(config)
	if strings.HasSuffix(recognizer, "/_") && languageCode == "" {
		respondGCPSpeechV2InvalidArgument(w, path, "config.languageCodes is required for implicit recognizer")
		return true
	}
	if languageCode == "" {
		languageCode = "en-US"
	}

	respondJSON(w, http.StatusOK, gcpSpeechV2RecognizeResponse(project, location, languageCode, "stackyard speech v2 recognized audio"))
	return true
}

func handleGCPSpeechV2StreamingRecognize(w http.ResponseWriter, path string, body map[string]any) bool {
	recognizer := strings.TrimSpace(gcpSpeechString(body, "recognizer"))
	project, location, _, ok := parseGCPSpeechV2RecognizerNameOrImplicit(recognizer)
	if !ok {
		respondGCPSpeechV2InvalidArgument(w, path, "recognizer is required")
		return true
	}

	streamingConfig := gcpSpeechBodyMap(body, "streamingConfig", "streaming_config")
	if len(streamingConfig) == 0 {
		respondGCPSpeechV2InvalidArgument(w, path, "streamingConfig is required")
		return true
	}
	config := gcpSpeechBodyMap(streamingConfig, "config")
	if len(config) == 0 {
		respondGCPSpeechV2InvalidArgument(w, path, "streamingConfig.config is required")
		return true
	}
	languageCode := gcpSpeechV2FirstLanguageCode(config)
	if languageCode == "" {
		respondGCPSpeechV2InvalidArgument(w, path, "streamingConfig.config.languageCodes is required")
		return true
	}
	if audio := strings.TrimSpace(gcpSpeechString(body, "audio")); audio != "" {
		respondGCPSpeechV2InvalidArgument(w, path, "audio is not supported in staged unary streaming emulation")
		return true
	}

	respondJSON(w, http.StatusOK, gcpSpeechV2StreamingRecognizeResponse(project, location, languageCode, "stackyard speech v2 streaming transcript"))
	return true
}

func handleGCPSpeechV2BatchRecognize(w http.ResponseWriter, path string, body map[string]any) bool {
	recognizer := strings.TrimSpace(gcpSpeechString(body, "recognizer"))
	project, location, _, ok := parseGCPSpeechV2RecognizerNameOrImplicit(recognizer)
	if !ok {
		respondGCPSpeechV2InvalidArgument(w, path, "recognizer is required")
		return true
	}

	filesAny, ok := body["files"].([]any)
	if !ok || len(filesAny) == 0 {
		respondGCPSpeechV2InvalidArgument(w, path, "files is required")
		return true
	}
	if len(filesAny) > 15 {
		respondGCPSpeechV2InvalidArgument(w, path, "files must contain at most 15 entries")
		return true
	}
	uris := make([]string, 0, len(filesAny))
	for _, raw := range filesAny {
		fileMap, _ := raw.(map[string]any)
		uri := strings.TrimSpace(gcpSpeechString(fileMap, "uri"))
		if uri == "" {
			respondGCPSpeechV2InvalidArgument(w, path, "files[].uri is required")
			return true
		}
		if strings.Contains(strings.ToLower(uri), "missing") {
			respondGCPSpeechV2NotFound(w, path, "audio uri not found")
			return true
		}
		uris = append(uris, uri)
	}

	outputConfig := gcpSpeechBodyMap(body, "recognitionOutputConfig", "recognition_output_config")
	if len(outputConfig) == 0 {
		respondGCPSpeechV2InvalidArgument(w, path, "recognitionOutputConfig is required")
		return true
	}
	if !gcpSpeechV2HasOutputTarget(outputConfig) {
		respondGCPSpeechV2InvalidArgument(w, path, "recognitionOutputConfig output target is required")
		return true
	}

	response := gcpSpeechV2BatchRecognizeResponse(uris)
	respondJSON(w, http.StatusOK, gcpSpeechV2Operation(project, location, "batchRecognize.stackyard", "google.cloud.speech.v2.Speech.BatchRecognize", fmt.Sprintf("projects/%s/locations/%s/recognizers/_", project, location), "type.googleapis.com/google.cloud.speech.v2.BatchRecognizeResponse", response))
	return true
}

func handleGCPSpeechV2GetConfig(w http.ResponseWriter, path string, body map[string]any) bool {
	name := strings.TrimSpace(gcpSpeechString(body, "name"))
	project, location, ok := parseGCPSpeechV2ConfigName(name)
	if !ok {
		respondGCPSpeechV2InvalidArgument(w, path, "name is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpSpeechV2Config(project, location))
	return true
}

func handleGCPSpeechV2UpdateConfig(w http.ResponseWriter, path string, body map[string]any) bool {
	configReq := gcpSpeechBodyMap(body, "config")
	if len(configReq) == 0 {
		respondGCPSpeechV2InvalidArgument(w, path, "config is required")
		return true
	}
	project, location, ok := parseGCPSpeechV2ConfigName(strings.TrimSpace(gcpSpeechString(configReq, "name")))
	if !ok {
		respondGCPSpeechV2InvalidArgument(w, path, "config.name is required")
		return true
	}
	updateMask := gcpSpeechV2ExtractUpdateMask(body)
	if updateMask == "" {
		respondGCPSpeechV2InvalidArgument(w, path, "updateMask is required")
		return true
	}
	if gcpSpeechV2MaskContains(updateMask, "name") {
		respondGCPSpeechV2FailedPrecondition(w, path, "config.name is immutable")
		return true
	}
	config := gcpSpeechV2Config(project, location)
	if kmsKeyName := strings.TrimSpace(gcpSpeechString(configReq, "kmsKeyName", "kms_key_name")); kmsKeyName != "" {
		config["kmsKeyName"] = kmsKeyName
	}
	respondJSON(w, http.StatusOK, config)
	return true
}

func handleGCPSpeechV2CreateCustomClass(w http.ResponseWriter, path string, body map[string]any) bool {
	parent := strings.TrimSpace(gcpSpeechString(body, "parent"))
	project, location, ok := parseGCPSpeechParentName(parent)
	if !ok {
		respondGCPSpeechV2InvalidArgument(w, path, "parent is required")
		return true
	}
	customClassID := strings.TrimSpace(gcpSpeechString(body, "customClassId", "custom_class_id"))
	if !isGCPSpeechV2ResourceID(customClassID) {
		respondGCPSpeechV2InvalidArgument(w, path, "customClassId is required")
		return true
	}
	if strings.Contains(strings.ToLower(customClassID), "existing") {
		respondGCPSpeechV2AlreadyExists(w, path, "custom class already exists")
		return true
	}
	customClassReq := gcpSpeechBodyMap(body, "customClass", "custom_class")
	if len(customClassReq) == 0 {
		respondGCPSpeechV2InvalidArgument(w, path, "customClass is required")
		return true
	}
	expectedName := fmt.Sprintf("projects/%s/locations/%s/customClasses/%s", project, location, customClassID)
	if providedName := strings.TrimSpace(gcpSpeechString(customClassReq, "name")); providedName != "" && providedName != expectedName {
		respondGCPSpeechV2InvalidArgument(w, path, "customClass.name must match parent and customClassId")
		return true
	}

	customClass := gcpSpeechV2CustomClass(project, location, customClassID, false)
	gcpSpeechV2ApplyCustomClassOverrides(customClass, customClassReq)
	respondJSON(w, http.StatusOK, gcpSpeechV2Operation(project, location, "createCustomClass."+customClassID, "google.cloud.speech.v2.Speech.CreateCustomClass", expectedName, "type.googleapis.com/google.cloud.speech.v2.CustomClass", customClass))
	return true
}

func handleGCPSpeechV2ListCustomClasses(w http.ResponseWriter, path string, body map[string]any) bool {
	parent := strings.TrimSpace(gcpSpeechString(body, "parent"))
	project, location, ok := parseGCPSpeechParentName(parent)
	if !ok {
		respondGCPSpeechV2InvalidArgument(w, path, "parent is required")
		return true
	}
	pageSize, start, valid := parseGCPSpeechV2BodyPagination(w, path, body, 5, 100)
	if !valid {
		return true
	}
	showDeleted, _ := gcpSpeechV2Bool(body, "showDeleted", "show_deleted")
	items := []map[string]any{
		gcpSpeechV2CustomClass(project, location, "custom-class-1", false),
		gcpSpeechV2CustomClass(project, location, "custom-class-deleted", true),
	}
	if !showDeleted {
		items = items[:1]
	}
	return respondGCPSpeechV2List(w, "customClasses", items, pageSize, start, path)
}

func handleGCPSpeechV2GetCustomClass(w http.ResponseWriter, path string, body map[string]any) bool {
	name := strings.TrimSpace(gcpSpeechString(body, "name"))
	project, location, customClassID, ok := parseGCPSpeechCustomClassName(name)
	if !ok {
		respondGCPSpeechV2InvalidArgument(w, path, "name is required")
		return true
	}
	if strings.Contains(strings.ToLower(customClassID), "missing") {
		respondGCPSpeechV2NotFound(w, path, "custom class not found")
		return true
	}
	respondJSON(w, http.StatusOK, gcpSpeechV2CustomClass(project, location, customClassID, strings.Contains(strings.ToLower(customClassID), "deleted")))
	return true
}

func handleGCPSpeechV2UpdateCustomClass(w http.ResponseWriter, path string, body map[string]any) bool {
	customClassReq := gcpSpeechBodyMap(body, "customClass", "custom_class")
	if len(customClassReq) == 0 {
		respondGCPSpeechV2InvalidArgument(w, path, "customClass is required")
		return true
	}
	project, location, customClassID, ok := parseGCPSpeechCustomClassName(strings.TrimSpace(gcpSpeechString(customClassReq, "name")))
	if !ok {
		respondGCPSpeechV2InvalidArgument(w, path, "customClass.name is required")
		return true
	}
	if strings.Contains(strings.ToLower(customClassID), "missing") {
		respondGCPSpeechV2NotFound(w, path, "custom class not found")
		return true
	}
	updateMask := gcpSpeechV2ExtractUpdateMask(body)
	if updateMask == "" {
		respondGCPSpeechV2InvalidArgument(w, path, "updateMask is required")
		return true
	}
	if gcpSpeechV2MaskContains(updateMask, "name") {
		respondGCPSpeechV2FailedPrecondition(w, path, "customClass.name is immutable")
		return true
	}
	if providedEtag := strings.TrimSpace(gcpSpeechString(customClassReq, "etag")); providedEtag != "" && providedEtag != gcpSpeechV2Etag(customClassID) {
		respondGCPSpeechV2FailedPrecondition(w, path, "etag mismatch")
		return true
	}
	customClass := gcpSpeechV2CustomClass(project, location, customClassID, false)
	gcpSpeechV2ApplyCustomClassOverrides(customClass, customClassReq)
	respondJSON(w, http.StatusOK, gcpSpeechV2Operation(project, location, "updateCustomClass."+customClassID, "google.cloud.speech.v2.Speech.UpdateCustomClass", customClass["name"].(string), "type.googleapis.com/google.cloud.speech.v2.CustomClass", customClass))
	return true
}

func handleGCPSpeechV2DeleteCustomClass(w http.ResponseWriter, path string, body map[string]any) bool {
	name := strings.TrimSpace(gcpSpeechString(body, "name"))
	project, location, customClassID, ok := parseGCPSpeechCustomClassName(name)
	if !ok {
		respondGCPSpeechV2InvalidArgument(w, path, "name is required")
		return true
	}
	allowMissing, _ := gcpSpeechV2Bool(body, "allowMissing", "allow_missing")
	if strings.Contains(strings.ToLower(customClassID), "missing") && !allowMissing {
		respondGCPSpeechV2NotFound(w, path, "custom class not found")
		return true
	}
	if providedEtag := strings.TrimSpace(gcpSpeechString(body, "etag")); providedEtag != "" && providedEtag != gcpSpeechV2Etag(customClassID) {
		respondGCPSpeechV2FailedPrecondition(w, path, "etag mismatch")
		return true
	}
	customClass := gcpSpeechV2CustomClass(project, location, customClassID, true)
	respondJSON(w, http.StatusOK, gcpSpeechV2Operation(project, location, "deleteCustomClass."+customClassID, "google.cloud.speech.v2.Speech.DeleteCustomClass", customClass["name"].(string), "type.googleapis.com/google.cloud.speech.v2.CustomClass", customClass))
	return true
}

func handleGCPSpeechV2UndeleteCustomClass(w http.ResponseWriter, path string, body map[string]any) bool {
	name := strings.TrimSpace(gcpSpeechString(body, "name"))
	project, location, customClassID, ok := parseGCPSpeechCustomClassName(name)
	if !ok {
		respondGCPSpeechV2InvalidArgument(w, path, "name is required")
		return true
	}
	if strings.Contains(strings.ToLower(customClassID), "missing") {
		respondGCPSpeechV2NotFound(w, path, "custom class not found")
		return true
	}
	if providedEtag := strings.TrimSpace(gcpSpeechString(body, "etag")); providedEtag != "" && providedEtag != gcpSpeechV2Etag(customClassID) {
		respondGCPSpeechV2FailedPrecondition(w, path, "etag mismatch")
		return true
	}
	customClass := gcpSpeechV2CustomClass(project, location, customClassID, false)
	respondJSON(w, http.StatusOK, gcpSpeechV2Operation(project, location, "undeleteCustomClass."+customClassID, "google.cloud.speech.v2.Speech.UndeleteCustomClass", customClass["name"].(string), "type.googleapis.com/google.cloud.speech.v2.CustomClass", customClass))
	return true
}

func handleGCPSpeechV2CreatePhraseSet(w http.ResponseWriter, path string, body map[string]any) bool {
	parent := strings.TrimSpace(gcpSpeechString(body, "parent"))
	project, location, ok := parseGCPSpeechParentName(parent)
	if !ok {
		respondGCPSpeechV2InvalidArgument(w, path, "parent is required")
		return true
	}
	phraseSetID := strings.TrimSpace(gcpSpeechString(body, "phraseSetId", "phrase_set_id"))
	if !isGCPSpeechV2ResourceID(phraseSetID) {
		respondGCPSpeechV2InvalidArgument(w, path, "phraseSetId is required")
		return true
	}
	if strings.Contains(strings.ToLower(phraseSetID), "existing") {
		respondGCPSpeechV2AlreadyExists(w, path, "phrase set already exists")
		return true
	}
	phraseSetReq := gcpSpeechBodyMap(body, "phraseSet", "phrase_set")
	if len(phraseSetReq) == 0 {
		respondGCPSpeechV2InvalidArgument(w, path, "phraseSet is required")
		return true
	}
	expectedName := fmt.Sprintf("projects/%s/locations/%s/phraseSets/%s", project, location, phraseSetID)
	if providedName := strings.TrimSpace(gcpSpeechString(phraseSetReq, "name")); providedName != "" && providedName != expectedName {
		respondGCPSpeechV2InvalidArgument(w, path, "phraseSet.name must match parent and phraseSetId")
		return true
	}

	phraseSet := gcpSpeechV2PhraseSet(project, location, phraseSetID, false)
	gcpSpeechV2ApplyPhraseSetOverrides(phraseSet, phraseSetReq)
	respondJSON(w, http.StatusOK, gcpSpeechV2Operation(project, location, "createPhraseSet."+phraseSetID, "google.cloud.speech.v2.Speech.CreatePhraseSet", expectedName, "type.googleapis.com/google.cloud.speech.v2.PhraseSet", phraseSet))
	return true
}

func handleGCPSpeechV2ListPhraseSets(w http.ResponseWriter, path string, body map[string]any) bool {
	parent := strings.TrimSpace(gcpSpeechString(body, "parent"))
	project, location, ok := parseGCPSpeechParentName(parent)
	if !ok {
		respondGCPSpeechV2InvalidArgument(w, path, "parent is required")
		return true
	}
	pageSize, start, valid := parseGCPSpeechV2BodyPagination(w, path, body, 5, 100)
	if !valid {
		return true
	}
	showDeleted, _ := gcpSpeechV2Bool(body, "showDeleted", "show_deleted")
	items := []map[string]any{
		gcpSpeechV2PhraseSet(project, location, "phrase-set-1", false),
		gcpSpeechV2PhraseSet(project, location, "phrase-set-deleted", true),
	}
	if !showDeleted {
		items = items[:1]
	}
	return respondGCPSpeechV2List(w, "phraseSets", items, pageSize, start, path)
}

func handleGCPSpeechV2GetPhraseSet(w http.ResponseWriter, path string, body map[string]any) bool {
	name := strings.TrimSpace(gcpSpeechString(body, "name"))
	project, location, phraseSetID, ok := parseGCPSpeechPhraseSetName(name)
	if !ok {
		respondGCPSpeechV2InvalidArgument(w, path, "name is required")
		return true
	}
	if strings.Contains(strings.ToLower(phraseSetID), "missing") {
		respondGCPSpeechV2NotFound(w, path, "phrase set not found")
		return true
	}
	respondJSON(w, http.StatusOK, gcpSpeechV2PhraseSet(project, location, phraseSetID, strings.Contains(strings.ToLower(phraseSetID), "deleted")))
	return true
}

func handleGCPSpeechV2UpdatePhraseSet(w http.ResponseWriter, path string, body map[string]any) bool {
	phraseSetReq := gcpSpeechBodyMap(body, "phraseSet", "phrase_set")
	if len(phraseSetReq) == 0 {
		respondGCPSpeechV2InvalidArgument(w, path, "phraseSet is required")
		return true
	}
	project, location, phraseSetID, ok := parseGCPSpeechPhraseSetName(strings.TrimSpace(gcpSpeechString(phraseSetReq, "name")))
	if !ok {
		respondGCPSpeechV2InvalidArgument(w, path, "phraseSet.name is required")
		return true
	}
	if strings.Contains(strings.ToLower(phraseSetID), "missing") {
		respondGCPSpeechV2NotFound(w, path, "phrase set not found")
		return true
	}
	updateMask := gcpSpeechV2ExtractUpdateMask(body)
	if updateMask == "" {
		respondGCPSpeechV2InvalidArgument(w, path, "updateMask is required")
		return true
	}
	if gcpSpeechV2MaskContains(updateMask, "name") {
		respondGCPSpeechV2FailedPrecondition(w, path, "phraseSet.name is immutable")
		return true
	}
	if providedEtag := strings.TrimSpace(gcpSpeechString(phraseSetReq, "etag")); providedEtag != "" && providedEtag != gcpSpeechV2Etag(phraseSetID) {
		respondGCPSpeechV2FailedPrecondition(w, path, "etag mismatch")
		return true
	}

	phraseSet := gcpSpeechV2PhraseSet(project, location, phraseSetID, false)
	gcpSpeechV2ApplyPhraseSetOverrides(phraseSet, phraseSetReq)
	respondJSON(w, http.StatusOK, gcpSpeechV2Operation(project, location, "updatePhraseSet."+phraseSetID, "google.cloud.speech.v2.Speech.UpdatePhraseSet", phraseSet["name"].(string), "type.googleapis.com/google.cloud.speech.v2.PhraseSet", phraseSet))
	return true
}

func handleGCPSpeechV2DeletePhraseSet(w http.ResponseWriter, path string, body map[string]any) bool {
	name := strings.TrimSpace(gcpSpeechString(body, "name"))
	project, location, phraseSetID, ok := parseGCPSpeechPhraseSetName(name)
	if !ok {
		respondGCPSpeechV2InvalidArgument(w, path, "name is required")
		return true
	}
	allowMissing, _ := gcpSpeechV2Bool(body, "allowMissing", "allow_missing")
	if strings.Contains(strings.ToLower(phraseSetID), "missing") && !allowMissing {
		respondGCPSpeechV2NotFound(w, path, "phrase set not found")
		return true
	}
	if providedEtag := strings.TrimSpace(gcpSpeechString(body, "etag")); providedEtag != "" && providedEtag != gcpSpeechV2Etag(phraseSetID) {
		respondGCPSpeechV2FailedPrecondition(w, path, "etag mismatch")
		return true
	}

	phraseSet := gcpSpeechV2PhraseSet(project, location, phraseSetID, true)
	respondJSON(w, http.StatusOK, gcpSpeechV2Operation(project, location, "deletePhraseSet."+phraseSetID, "google.cloud.speech.v2.Speech.DeletePhraseSet", phraseSet["name"].(string), "type.googleapis.com/google.cloud.speech.v2.PhraseSet", phraseSet))
	return true
}

func handleGCPSpeechV2UndeletePhraseSet(w http.ResponseWriter, path string, body map[string]any) bool {
	name := strings.TrimSpace(gcpSpeechString(body, "name"))
	project, location, phraseSetID, ok := parseGCPSpeechPhraseSetName(name)
	if !ok {
		respondGCPSpeechV2InvalidArgument(w, path, "name is required")
		return true
	}
	if strings.Contains(strings.ToLower(phraseSetID), "missing") {
		respondGCPSpeechV2NotFound(w, path, "phrase set not found")
		return true
	}
	if providedEtag := strings.TrimSpace(gcpSpeechString(body, "etag")); providedEtag != "" && providedEtag != gcpSpeechV2Etag(phraseSetID) {
		respondGCPSpeechV2FailedPrecondition(w, path, "etag mismatch")
		return true
	}
	phraseSet := gcpSpeechV2PhraseSet(project, location, phraseSetID, false)
	respondJSON(w, http.StatusOK, gcpSpeechV2Operation(project, location, "undeletePhraseSet."+phraseSetID, "google.cloud.speech.v2.Speech.UndeletePhraseSet", phraseSet["name"].(string), "type.googleapis.com/google.cloud.speech.v2.PhraseSet", phraseSet))
	return true
}

func parseGCPSpeechV2ProjectLocationPath(path string) (project, location string, list, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 5 && len(parts) != 6 {
		return "", "", false, false
	}
	if parts[0] != "gcp" || parts[1] != "v2" || parts[2] != "projects" || parts[4] != "locations" {
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

func parseGCPSpeechV2RecognizerName(name string) (project, location, recognizerID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 6 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "recognizers" {
		return "", "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	recognizerID = strings.TrimSpace(parts[5])
	if project == "" || location == "" || (!isGCPSpeechV2ResourceID(recognizerID) && recognizerID != "_") {
		return "", "", "", false
	}
	return project, location, recognizerID, true
}

func parseGCPSpeechV2RecognizerNameOrImplicit(name string) (project, location, recognizerID string, ok bool) {
	return parseGCPSpeechV2RecognizerName(name)
}

func parseGCPSpeechV2ConfigName(name string) (project, location string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 5 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "config" {
		return "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	if project == "" || location == "" {
		return "", "", false
	}
	return project, location, true
}

func isGCPSpeechV2ResourceID(id string) bool {
	id = strings.TrimSpace(id)
	if !isGCPSpeechResourceID(id) {
		return false
	}
	return len(id) >= 4 && len(id) <= 63
}

func parseGCPSpeechV2QueryPagination(w http.ResponseWriter, r *http.Request, path string, defaultPageSize, maxPageSize int) (pageSize, start int, ok bool) {
	pageSize = defaultPageSize
	if raw := strings.TrimSpace(r.URL.Query().Get("pageSize")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 0 {
			respondGCPSpeechV2InvalidArgument(w, path, "pageSize must be a non-negative integer")
			return 0, 0, false
		}
		if value > maxPageSize {
			respondGCPSpeechV2InvalidArgument(w, path, fmt.Sprintf("pageSize must be <= %d", maxPageSize))
			return 0, 0, false
		}
		pageSize = value
	}
	if raw := strings.TrimSpace(r.URL.Query().Get("pageToken")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 0 {
			respondGCPSpeechV2InvalidArgument(w, path, "pageToken must be a non-negative integer")
			return 0, 0, false
		}
		start = value
	}
	return pageSize, start, true
}

func parseGCPSpeechV2BodyPagination(w http.ResponseWriter, path string, body map[string]any, defaultPageSize, maxPageSize int) (pageSize, start int, ok bool) {
	pageSize = defaultPageSize
	if value, exists := gcpSpeechInt(body, "pageSize", "page_size"); exists {
		if value < 0 {
			respondGCPSpeechV2InvalidArgument(w, path, "pageSize must be a non-negative integer")
			return 0, 0, false
		}
		if value > maxPageSize {
			respondGCPSpeechV2InvalidArgument(w, path, fmt.Sprintf("pageSize must be <= %d", maxPageSize))
			return 0, 0, false
		}
		pageSize = value
	}
	if token := strings.TrimSpace(gcpSpeechString(body, "pageToken", "page_token")); token != "" {
		value, err := strconv.Atoi(token)
		if err != nil || value < 0 {
			respondGCPSpeechV2InvalidArgument(w, path, "pageToken must be a non-negative integer")
			return 0, 0, false
		}
		start = value
	}
	return pageSize, start, true
}

func respondGCPSpeechV2List(w http.ResponseWriter, field string, items []map[string]any, pageSize, start int, path string) bool {
	if pageSize < 0 || start < 0 {
		respondGCPSpeechV2InvalidArgument(w, path, "invalid pagination values")
		return false
	}
	if pageSize == 0 {
		pageSize = len(items)
	}
	if start > len(items) {
		respondGCPSpeechV2InvalidArgument(w, path, "pageToken is out of range")
		return true
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

func gcpSpeechV2Bool(body map[string]any, keys ...string) (bool, bool) {
	for _, key := range keys {
		raw, ok := body[key]
		if !ok {
			continue
		}
		switch value := raw.(type) {
		case bool:
			return value, true
		case string:
			switch strings.ToLower(strings.TrimSpace(value)) {
			case "true":
				return true, true
			case "false":
				return false, true
			}
		}
	}
	return false, false
}

func gcpSpeechV2ExtractUpdateMask(body map[string]any) string {
	mask := strings.TrimSpace(gcpSpeechString(body, "updateMask", "update_mask"))
	if mask != "" {
		return mask
	}
	maskMap := gcpSpeechBodyMap(body, "updateMask", "update_mask")
	if len(maskMap) == 0 {
		return ""
	}
	pathsAny, ok := maskMap["paths"].([]any)
	if !ok || len(pathsAny) == 0 {
		return ""
	}
	parts := make([]string, 0, len(pathsAny))
	for _, raw := range pathsAny {
		text := strings.TrimSpace(fmt.Sprint(raw))
		if text != "" {
			parts = append(parts, text)
		}
	}
	return strings.Join(parts, ",")
}

func gcpSpeechV2HasOutputTarget(outputConfig map[string]any) bool {
	for _, key := range []string{"gcsOutputConfig", "gcs_output_config", "inlineResponseConfig", "inline_response_config"} {
		if _, exists := outputConfig[key]; exists {
			return true
		}
	}
	return false
}

func gcpSpeechV2MaskContains(mask, field string) bool {
	field = strings.ToLower(strings.TrimSpace(field))
	for _, part := range strings.Split(strings.ToLower(mask), ",") {
		if strings.TrimSpace(part) == field {
			return true
		}
	}
	return false
}

func gcpSpeechV2FirstLanguageCode(config map[string]any) string {
	raw, ok := config["languageCodes"]
	if !ok {
		raw = config["language_codes"]
	}
	switch value := raw.(type) {
	case []string:
		if len(value) > 0 {
			return strings.TrimSpace(value[0])
		}
	case []any:
		for _, rawItem := range value {
			if text := strings.TrimSpace(fmt.Sprint(rawItem)); text != "" {
				return text
			}
		}
	}
	return ""
}

func gcpSpeechV2Location(project, location string) map[string]any {
	return map[string]any{
		"name":        fmt.Sprintf("projects/%s/locations/%s", project, location),
		"locationId":  location,
		"displayName": strings.ToUpper(location),
		"labels": map[string]any{
			"cloud.googleapis.com/region": location,
		},
	}
}

func gcpSpeechV2Recognizer(project, location, recognizerID string, deleted bool) map[string]any {
	resourceName := fmt.Sprintf("projects/%s/locations/%s/recognizers/%s", project, location, recognizerID)
	resp := map[string]any{
		"name":        resourceName,
		"uid":         "uid-" + recognizerID,
		"displayName": "Stackyard Recognizer " + recognizerID,
		"model":       "latest_long",
		"languageCodes": []any{
			"en-US",
		},
		"defaultRecognitionConfig": map[string]any{
			"languageCodes": []any{"en-US"},
			"model":         "latest_long",
			"features": map[string]any{
				"enableAutomaticPunctuation": true,
			},
		},
		"annotations": map[string]any{
			"env": "staged",
		},
		"state":       "ACTIVE",
		"createTime":  gcpSpeechV2ReferenceTime.Add(-30 * time.Minute).Format(time.RFC3339Nano),
		"updateTime":  gcpSpeechV2ReferenceTime.Format(time.RFC3339Nano),
		"etag":        gcpSpeechV2Etag(recognizerID),
		"reconciling": false,
	}
	if deleted {
		resp["state"] = "DELETED"
		resp["deleteTime"] = gcpSpeechV2ReferenceTime.Add(-5 * time.Minute).Format(time.RFC3339Nano)
		resp["expireTime"] = gcpSpeechV2ReferenceTime.Add(24 * time.Hour).Format(time.RFC3339Nano)
	}
	return resp
}

func gcpSpeechV2ApplyRecognizerOverrides(response, request map[string]any) {
	if response == nil || request == nil {
		return
	}
	if displayName := strings.TrimSpace(gcpSpeechString(request, "displayName", "display_name")); displayName != "" {
		response["displayName"] = displayName
	}
	if model := strings.TrimSpace(gcpSpeechString(request, "model")); model != "" {
		response["model"] = model
	}
	if languagesAny, ok := request["languageCodes"].([]any); ok && len(languagesAny) > 0 {
		languages := make([]any, 0, len(languagesAny))
		for _, raw := range languagesAny {
			if text := strings.TrimSpace(fmt.Sprint(raw)); text != "" {
				languages = append(languages, text)
			}
		}
		if len(languages) > 0 {
			response["languageCodes"] = languages
		}
	}
	if defaultCfg, ok := request["defaultRecognitionConfig"].(map[string]any); ok && len(defaultCfg) > 0 {
		response["defaultRecognitionConfig"] = defaultCfg
	}
	if annotations, ok := request["annotations"].(map[string]any); ok && len(annotations) > 0 {
		response["annotations"] = annotations
	}
}

func gcpSpeechV2RecognizeResponse(project, location, languageCode, transcript string) map[string]any {
	return map[string]any{
		"results": []any{
			map[string]any{
				"alternatives": []any{
					map[string]any{
						"transcript": transcript,
						"confidence": 0.98,
					},
				},
				"channelTag":      1,
				"resultEndOffset": "1.200s",
				"languageCode":    languageCode,
			},
		},
		"metadata": map[string]any{
			"requestId":           fmt.Sprintf("speech-v2-%s-%s-req-1", project, location),
			"totalBilledDuration": "1.200s",
		},
	}
}

func gcpSpeechV2StreamingRecognizeResponse(project, location, languageCode, transcript string) map[string]any {
	return map[string]any{
		"results": []any{
			map[string]any{
				"alternatives": []any{
					map[string]any{
						"transcript": transcript,
						"confidence": 0.93,
					},
				},
				"isFinal":         true,
				"stability":       0.91,
				"resultEndOffset": "0.900s",
				"channelTag":      1,
				"languageCode":    languageCode,
			},
		},
		"speechEventType":   "END_OF_SINGLE_UTTERANCE",
		"speechEventOffset": "0.900s",
		"metadata": map[string]any{
			"requestId":           fmt.Sprintf("speech-v2-%s-%s-stream-1", project, location),
			"totalBilledDuration": "0.900s",
		},
	}
}

func gcpSpeechV2BatchRecognizeResponse(uris []string) map[string]any {
	results := map[string]any{}
	for i, uri := range uris {
		results[uri] = map[string]any{
			"metadata": map[string]any{
				"requestId":           fmt.Sprintf("speech-v2-batch-%d", i+1),
				"totalBilledDuration": "1.200s",
			},
			"inlineResult": map[string]any{
				"transcript": map[string]any{
					"results": []any{
						map[string]any{
							"alternatives": []any{
								map[string]any{
									"transcript": fmt.Sprintf("stackyard transcript for %s", uri),
									"confidence": 0.95,
								},
							},
							"channelTag":      1,
							"resultEndOffset": "1.200s",
							"languageCode":    "en-US",
						},
					},
					"metadata": map[string]any{
						"requestId":           fmt.Sprintf("speech-v2-batch-inline-%d", i+1),
						"totalBilledDuration": "1.200s",
					},
				},
			},
		}
	}
	return map[string]any{
		"results":             results,
		"totalBilledDuration": "1.200s",
	}
}

func gcpSpeechV2Config(project, location string) map[string]any {
	return map[string]any{
		"name":       fmt.Sprintf("projects/%s/locations/%s/config", project, location),
		"kmsKeyName": fmt.Sprintf("projects/%s/locations/%s/keyRings/stackyard/cryptoKeys/speech-v2", project, location),
		"updateTime": gcpSpeechV2ReferenceTime.Format(time.RFC3339Nano),
	}
}

func gcpSpeechV2CustomClass(project, location, customClassID string, deleted bool) map[string]any {
	resourceName := fmt.Sprintf("projects/%s/locations/%s/customClasses/%s", project, location, customClassID)
	resp := map[string]any{
		"name":        resourceName,
		"uid":         "uid-" + customClassID,
		"displayName": "Stackyard CustomClass " + customClassID,
		"items": []any{
			map[string]any{"value": "stackyard"},
			map[string]any{"value": "speech"},
		},
		"annotations": map[string]any{
			"env": "staged",
		},
		"state":       "ACTIVE",
		"createTime":  gcpSpeechV2ReferenceTime.Add(-30 * time.Minute).Format(time.RFC3339Nano),
		"updateTime":  gcpSpeechV2ReferenceTime.Format(time.RFC3339Nano),
		"etag":        gcpSpeechV2Etag(customClassID),
		"reconciling": false,
	}
	if deleted {
		resp["state"] = "DELETED"
		resp["deleteTime"] = gcpSpeechV2ReferenceTime.Add(-5 * time.Minute).Format(time.RFC3339Nano)
		resp["expireTime"] = gcpSpeechV2ReferenceTime.Add(24 * time.Hour).Format(time.RFC3339Nano)
	}
	return resp
}

func gcpSpeechV2ApplyCustomClassOverrides(response, request map[string]any) {
	if response == nil || request == nil {
		return
	}
	if displayName := strings.TrimSpace(gcpSpeechString(request, "displayName", "display_name")); displayName != "" {
		response["displayName"] = displayName
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
	if annotations, ok := request["annotations"].(map[string]any); ok && len(annotations) > 0 {
		response["annotations"] = annotations
	}
}

func gcpSpeechV2PhraseSet(project, location, phraseSetID string, deleted bool) map[string]any {
	resourceName := fmt.Sprintf("projects/%s/locations/%s/phraseSets/%s", project, location, phraseSetID)
	resp := map[string]any{
		"name":        resourceName,
		"uid":         "uid-" + phraseSetID,
		"displayName": "Stackyard PhraseSet " + phraseSetID,
		"phrases": []any{
			map[string]any{"value": "stackyard"},
			map[string]any{"value": "cloud emulation"},
		},
		"boost": 12.5,
		"annotations": map[string]any{
			"env": "staged",
		},
		"state":       "ACTIVE",
		"createTime":  gcpSpeechV2ReferenceTime.Add(-30 * time.Minute).Format(time.RFC3339Nano),
		"updateTime":  gcpSpeechV2ReferenceTime.Format(time.RFC3339Nano),
		"etag":        gcpSpeechV2Etag(phraseSetID),
		"reconciling": false,
	}
	if deleted {
		resp["state"] = "DELETED"
		resp["deleteTime"] = gcpSpeechV2ReferenceTime.Add(-5 * time.Minute).Format(time.RFC3339Nano)
		resp["expireTime"] = gcpSpeechV2ReferenceTime.Add(24 * time.Hour).Format(time.RFC3339Nano)
	}
	return resp
}

func gcpSpeechV2ApplyPhraseSetOverrides(response, request map[string]any) {
	if response == nil || request == nil {
		return
	}
	if displayName := strings.TrimSpace(gcpSpeechString(request, "displayName", "display_name")); displayName != "" {
		response["displayName"] = displayName
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
	if annotations, ok := request["annotations"].(map[string]any); ok && len(annotations) > 0 {
		response["annotations"] = annotations
	}
}

func gcpSpeechV2Operation(project, location, operationID, method, resource, responseType string, response map[string]any) map[string]any {
	return map[string]any{
		"name": fmt.Sprintf("projects/%s/locations/%s/operations/%s", project, location, operationID),
		"metadata": map[string]any{
			"@type":           "type.googleapis.com/google.cloud.speech.v2.OperationMetadata",
			"createTime":      gcpSpeechV2ReferenceTime.Add(-2 * time.Second).Format(time.RFC3339Nano),
			"updateTime":      gcpSpeechV2ReferenceTime.Format(time.RFC3339Nano),
			"resource":        resource,
			"method":          method,
			"progressPercent": 100,
		},
		"done":     true,
		"response": gcpSpeechV2TypedAny(responseType, response),
	}
}

func gcpSpeechV2TypedAny(typeURL string, payload map[string]any) map[string]any {
	out := map[string]any{
		"@type": typeURL,
	}
	for k, v := range payload {
		out[k] = v
	}
	return out
}

func gcpSpeechV2Etag(resourceID string) string {
	return fmt.Sprintf(`W/"speech-v2-%s"`, resourceID)
}

func respondGCPSpeechV2InvalidArgument(w http.ResponseWriter, path, message string) {
	respondGCPSpeechV2Error(w, http.StatusBadRequest, "InvalidArgument", path, message)
}

func respondGCPSpeechV2NotFound(w http.ResponseWriter, path, message string) {
	respondGCPSpeechV2Error(w, http.StatusNotFound, "NotFound", path, message)
}

func respondGCPSpeechV2AlreadyExists(w http.ResponseWriter, path, message string) {
	respondGCPSpeechV2Error(w, http.StatusConflict, "AlreadyExists", path, message)
}

func respondGCPSpeechV2FailedPrecondition(w http.ResponseWriter, path, message string) {
	respondGCPSpeechV2Error(w, http.StatusBadRequest, "FailedPrecondition", path, message)
}

func respondGCPSpeechV2Error(w http.ResponseWriter, status int, errToken, path, message string) {
	respondJSON(w, status, map[string]any{
		"error":    errToken,
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func handleGCPContractProbe_speech_v2(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "speech_v2") {
		return false
	}
	if r.URL.Query().Get("pageSize") == "bad" {
		respondGCPSpeechV2InvalidArgument(w, path, "pageSize must be a non-negative integer")
		return true
	}
	if r.URL.Query().Get("alreadyExists") == "1" {
		respondGCPSpeechV2AlreadyExists(w, path, "resource already exists")
		return true
	}
	if r.URL.Query().Get("failedPrecondition") == "1" {
		respondGCPSpeechV2FailedPrecondition(w, path, "precondition failed")
		return true
	}
	if r.URL.Query().Get("typedSuccess") == "1" {
		respondJSON(w, http.StatusOK, map[string]any{
			"name":     "projects/stackyard/locations/us-central1/speech_v2/sample",
			"service":  "speech_v2",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}
	return false
}
