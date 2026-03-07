package server

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	gcpTextToSpeechRESTListVoicesPath      = "/gcp/v1/voices"
	gcpTextToSpeechRESTSynthesizePath      = "/gcp/v1/text:synthesize"
	gcpTextToSpeechRESTLongAudioMethodName = ":synthesizeLongAudio"

	gcpTextToSpeechGRPCListVoicesPath          = "/gcp/google.cloud.texttospeech.v1.TextToSpeech/ListVoices"
	gcpTextToSpeechGRPCSynthesizeSpeechPath    = "/gcp/google.cloud.texttospeech.v1.TextToSpeech/SynthesizeSpeech"
	gcpTextToSpeechGRPCStreamingSynthesizePath = "/gcp/google.cloud.texttospeech.v1.TextToSpeech/StreamingSynthesize"
	gcpTextToSpeechGRPCSynthesizeLongAudioPath = "/gcp/google.cloud.texttospeech.v1.TextToSpeechLongAudioSynthesize/SynthesizeLongAudio"
	gcpTextToSpeechGRPCGetOperationPath        = "/gcp/google.cloud.texttospeech.v1.TextToSpeechLongAudioSynthesize/GetOperation"
	gcpTextToSpeechGRPCListOperationsPath      = "/gcp/google.cloud.texttospeech.v1.TextToSpeechLongAudioSynthesize/ListOperations"
)

var gcpTextToSpeechReferenceTime = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

func (s *Server) handleGCPTextToSpeechRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_texttospeech(w, r) {
		return true
	}

	path := normalizeGCPTextToSpeechPath(rawRequestPath(r))
	if isGCPTextToSpeechLocationRequest(r, path) {
		if r.Method != http.MethodGet {
			return false
		}
		if handleGCPTextToSpeechListLocations(w, r, path) {
			return true
		}
		if handleGCPTextToSpeechGetLocation(w, path) {
			return true
		}
		return false
	}

	if !isGCPTextToSpeechPath(path, hasGCPTextToSpeechHint(r)) {
		return false
	}

	switch r.Method {
	case http.MethodGet:
		if handleGCPTextToSpeechListVoicesGET(w, r, path) {
			return true
		}
		if handleGCPTextToSpeechListOperations(w, r, path) {
			return true
		}
		if handleGCPTextToSpeechGetOperation(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost:
		body, valid := decodeGCPTextToSpeechJSONBody(w, r, path)
		if !valid {
			return true
		}
		switch path {
		case gcpTextToSpeechGRPCListVoicesPath:
			return handleGCPTextToSpeechListVoicesPOST(w, path, body)
		case gcpTextToSpeechRESTSynthesizePath, gcpTextToSpeechGRPCSynthesizeSpeechPath:
			return handleGCPTextToSpeechSynthesizeSpeech(w, path, body)
		case gcpTextToSpeechGRPCStreamingSynthesizePath:
			respondProviderNotImplemented(w, providerGCP, path)
			return true
		case gcpTextToSpeechGRPCSynthesizeLongAudioPath:
			return handleGCPTextToSpeechSynthesizeLongAudioGRPC(w, path, body)
		case gcpTextToSpeechGRPCGetOperationPath:
			return handleGCPTextToSpeechGetOperationPOST(w, path, body)
		case gcpTextToSpeechGRPCListOperationsPath:
			return handleGCPTextToSpeechListOperationsPOST(w, path, body)
		default:
			if handleGCPTextToSpeechSynthesizeLongAudioREST(w, path, body) {
				return true
			}
			respondProviderNotImplemented(w, providerGCP, path)
			return true
		}
	default:
		return false
	}
}

func normalizeGCPTextToSpeechPath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.ReplaceAll(path, "%3A", ":")
	path = strings.ReplaceAll(path, "%3a", ":")
	return path
}

func hasGCPTextToSpeechHint(r *http.Request) bool {
	service := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Stackyard-GCP-Service")))
	switch service {
	case "texttospeech",
		"texttospeech-apiv1",
		"texttospeech_apiv1",
		"text-to-speech",
		"text_to_speech",
		"tts",
		"cloud-texttospeech",
		"gcp-texttospeech":
		return true
	}
	ua := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	return strings.Contains(ua, "stackyard-texttospeech-apiv1") ||
		strings.Contains(ua, "cloud.google.com/go/texttospeech/apiv1")
}

func isGCPTextToSpeechLocationRequest(r *http.Request, path string) bool {
	if !hasGCPTextToSpeechHint(r) {
		return false
	}
	_, location, list, ok := parseGCPProjectLocationPath(path)
	if !ok {
		return false
	}
	if !list && strings.Contains(location, ":") {
		return false
	}
	return true
}

func isGCPTextToSpeechPath(path string, includeHint bool) bool {
	switch path {
	case gcpTextToSpeechRESTListVoicesPath,
		gcpTextToSpeechRESTSynthesizePath,
		gcpTextToSpeechGRPCListVoicesPath,
		gcpTextToSpeechGRPCSynthesizeSpeechPath,
		gcpTextToSpeechGRPCStreamingSynthesizePath,
		gcpTextToSpeechGRPCSynthesizeLongAudioPath,
		gcpTextToSpeechGRPCGetOperationPath,
		gcpTextToSpeechGRPCListOperationsPath:
		return true
	}
	if _, _, ok := parseGCPTextToSpeechSynthesizeLongAudioPath(path); ok {
		return true
	}
	if _, _, ok := parseGCPTextToSpeechOperationsCollectionPath(path); ok {
		return includeHint
	}
	if _, _, _, ok := parseGCPTextToSpeechOperationPath(path); ok {
		return includeHint
	}
	if isGCPTextToSpeechGRPCPath(path) {
		return true
	}
	return false
}

func isGCPTextToSpeechGRPCPath(path string) bool {
	trimmed := strings.TrimSpace(path)
	return strings.HasPrefix(trimmed, "/gcp/google.cloud.texttospeech.v1.TextToSpeech/") ||
		strings.HasPrefix(trimmed, "/gcp/google.cloud.texttospeech.v1.TextToSpeechLongAudioSynthesize/")
}

func handleGCPTextToSpeechListLocations(w http.ResponseWriter, r *http.Request, path string) bool {
	project, _, list, ok := parseGCPProjectLocationPath(path)
	if !ok || !list {
		return false
	}
	pageSize, start, ok := parseGCPTextToSpeechQueryPagination(w, r, path)
	if !ok {
		return true
	}
	items := []map[string]any{
		gcpTextToSpeechLocation(project, "us-central1"),
		gcpTextToSpeechLocation(project, "global"),
	}
	return respondGCPTextToSpeechList(w, "locations", items, pageSize, start, path)
}

func handleGCPTextToSpeechGetLocation(w http.ResponseWriter, path string) bool {
	project, location, list, ok := parseGCPProjectLocationPath(path)
	if !ok || list {
		return false
	}
	respondJSON(w, http.StatusOK, gcpTextToSpeechLocation(project, location))
	return true
}

func handleGCPTextToSpeechListVoicesGET(w http.ResponseWriter, r *http.Request, path string) bool {
	if path != gcpTextToSpeechRESTListVoicesPath {
		return false
	}
	languageCode := strings.TrimSpace(r.URL.Query().Get("languageCode"))
	if languageCode == "" {
		languageCode = strings.TrimSpace(r.URL.Query().Get("language_code"))
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"voices": gcpTextToSpeechFilterVoices(gcpTextToSpeechVoices(), languageCode),
	})
	return true
}

func handleGCPTextToSpeechListVoicesPOST(w http.ResponseWriter, path string, body map[string]any) bool {
	if path != gcpTextToSpeechGRPCListVoicesPath {
		return false
	}
	languageCode := strings.TrimSpace(gcpTextToSpeechString(body, "languageCode", "language_code"))
	respondJSON(w, http.StatusOK, map[string]any{
		"voices": gcpTextToSpeechFilterVoices(gcpTextToSpeechVoices(), languageCode),
	})
	return true
}

func handleGCPTextToSpeechSynthesizeSpeech(w http.ResponseWriter, path string, body map[string]any) bool {
	if len(body) == 0 {
		respondGCPTextToSpeechInvalidArgument(w, path, "request is required")
		return true
	}
	input := gcpTextToSpeechBodyMap(body, "input")
	if len(input) == 0 {
		respondGCPTextToSpeechInvalidArgument(w, path, "input is required")
		return true
	}
	if !validateGCPTextToSpeechSynthesisInput(w, path, input) {
		return true
	}
	audioConfig := gcpTextToSpeechBodyMap(body, "audioConfig", "audio_config")
	if len(audioConfig) == 0 {
		respondGCPTextToSpeechInvalidArgument(w, path, "audio_config is required")
		return true
	}
	if !validateGCPTextToSpeechAudioConfig(w, path, audioConfig) {
		return true
	}
	voice := gcpTextToSpeechBodyMap(body, "voice")
	if len(voice) > 0 {
		if !validateGCPTextToSpeechVoice(w, path, voice) {
			return true
		}
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"audioContent": base64.StdEncoding.EncodeToString([]byte("stackyard-audio")),
	})
	return true
}

func handleGCPTextToSpeechSynthesizeLongAudioREST(w http.ResponseWriter, path string, body map[string]any) bool {
	project, location, ok := parseGCPTextToSpeechSynthesizeLongAudioPath(path)
	if !ok {
		return false
	}
	return handleGCPTextToSpeechSynthesizeLongAudioCore(w, path, body, project, location, true)
}

func handleGCPTextToSpeechSynthesizeLongAudioGRPC(w http.ResponseWriter, path string, body map[string]any) bool {
	if path != gcpTextToSpeechGRPCSynthesizeLongAudioPath {
		return false
	}
	parent := strings.TrimSpace(gcpTextToSpeechString(body, "parent"))
	project, location, ok := parseGCPTextToSpeechLocationName(parent)
	if !ok {
		respondGCPTextToSpeechInvalidArgument(w, path, "parent is required")
		return true
	}
	return handleGCPTextToSpeechSynthesizeLongAudioCore(w, path, body, project, location, false)
}

func handleGCPTextToSpeechSynthesizeLongAudioCore(w http.ResponseWriter, path string, body map[string]any, project, location string, enforceParentMatch bool) bool {
	if len(body) == 0 {
		respondGCPTextToSpeechInvalidArgument(w, path, "request is required")
		return true
	}
	if enforceParentMatch {
		if parent := strings.TrimSpace(gcpTextToSpeechString(body, "parent")); parent != "" {
			parsedProject, parsedLocation, ok := parseGCPTextToSpeechLocationName(parent)
			if !ok {
				respondGCPTextToSpeechInvalidArgument(w, path, "parent is invalid")
				return true
			}
			if parsedProject != project || parsedLocation != location {
				respondGCPTextToSpeechInvalidArgument(w, path, "parent must match requested resource")
				return true
			}
		}
	}
	input := gcpTextToSpeechBodyMap(body, "input")
	if len(input) == 0 {
		respondGCPTextToSpeechInvalidArgument(w, path, "input is required")
		return true
	}
	if !validateGCPTextToSpeechSynthesisInput(w, path, input) {
		return true
	}
	audioConfig := gcpTextToSpeechBodyMap(body, "audioConfig", "audio_config")
	if len(audioConfig) == 0 {
		respondGCPTextToSpeechInvalidArgument(w, path, "audio_config is required")
		return true
	}
	if !validateGCPTextToSpeechAudioConfig(w, path, audioConfig) {
		return true
	}

	outputGCSURI := strings.TrimSpace(gcpTextToSpeechString(body, "outputGcsUri", "output_gcs_uri"))
	if outputGCSURI == "" {
		respondGCPTextToSpeechInvalidArgument(w, path, "output_gcs_uri is required")
		return true
	}
	if !strings.HasPrefix(outputGCSURI, "gs://") {
		respondGCPTextToSpeechInvalidArgument(w, path, "output_gcs_uri must start with gs://")
		return true
	}

	respondJSON(w, http.StatusOK, gcpTextToSpeechOperation(project, location, "synthesizeLongAudio.op-1"))
	return true
}

func handleGCPTextToSpeechListOperations(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, ok := parseGCPTextToSpeechOperationsCollectionPath(path)
	if !ok {
		return false
	}
	pageSize, start, ok := parseGCPTextToSpeechQueryPagination(w, r, path)
	if !ok {
		return true
	}
	return respondGCPTextToSpeechList(w, "operations", gcpTextToSpeechOperationFixtures(project, location), pageSize, start, path)
}

func handleGCPTextToSpeechListOperationsPOST(w http.ResponseWriter, path string, body map[string]any) bool {
	if path != gcpTextToSpeechGRPCListOperationsPath {
		return false
	}
	parent := strings.TrimSpace(gcpTextToSpeechString(body, "name"))
	project, location, ok := parseGCPTextToSpeechLocationName(parent)
	if !ok {
		respondGCPTextToSpeechInvalidArgument(w, path, "name is required")
		return true
	}
	pageSize, start, ok := parseGCPTextToSpeechBodyPagination(w, path, body)
	if !ok {
		return true
	}
	return respondGCPTextToSpeechList(w, "operations", gcpTextToSpeechOperationFixtures(project, location), pageSize, start, path)
}

func handleGCPTextToSpeechGetOperation(w http.ResponseWriter, path string) bool {
	project, location, operationID, ok := parseGCPTextToSpeechOperationPath(path)
	if !ok {
		return false
	}
	if isGCPTextToSpeechMissingID(operationID) {
		respondGCPTextToSpeechNotFound(w, path, "operation not found")
		return true
	}
	respondJSON(w, http.StatusOK, gcpTextToSpeechOperation(project, location, operationID))
	return true
}

func handleGCPTextToSpeechGetOperationPOST(w http.ResponseWriter, path string, body map[string]any) bool {
	if path != gcpTextToSpeechGRPCGetOperationPath {
		return false
	}
	name := strings.TrimSpace(gcpTextToSpeechString(body, "name"))
	project, location, operationID, ok := parseGCPTextToSpeechOperationName(name)
	if !ok {
		respondGCPTextToSpeechInvalidArgument(w, path, "name is required")
		return true
	}
	if isGCPTextToSpeechMissingID(operationID) {
		respondGCPTextToSpeechNotFound(w, path, "operation not found")
		return true
	}
	respondJSON(w, http.StatusOK, gcpTextToSpeechOperation(project, location, operationID))
	return true
}

func parseGCPTextToSpeechSynthesizeLongAudioPath(path string) (project, location string, ok bool) {
	basePath, found := strings.CutSuffix(strings.TrimSpace(path), gcpTextToSpeechRESTLongAudioMethodName)
	if !found {
		return "", "", false
	}
	parts := strings.Split(strings.Trim(basePath, "/"), "/")
	if len(parts) != 6 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "locations" {
		return "", "", false
	}
	project = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	if project == "" || location == "" {
		return "", "", false
	}
	return project, location, true
}

func parseGCPTextToSpeechOperationsCollectionPath(path string) (project, location string, ok bool) {
	parts := strings.Split(strings.Trim(strings.TrimSpace(path), "/"), "/")
	if len(parts) != 7 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "locations" || parts[6] != "operations" {
		return "", "", false
	}
	project = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	if project == "" || location == "" {
		return "", "", false
	}
	return project, location, true
}

func parseGCPTextToSpeechOperationPath(path string) (project, location, operationID string, ok bool) {
	parts := strings.Split(strings.Trim(strings.TrimSpace(path), "/"), "/")
	if len(parts) != 8 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "locations" || parts[6] != "operations" {
		return "", "", "", false
	}
	project = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	operationID = strings.TrimSpace(parts[7])
	if project == "" || location == "" || operationID == "" {
		return "", "", "", false
	}
	return project, location, operationID, true
}

func parseGCPTextToSpeechLocationName(name string) (project, location string, ok bool) {
	parts := strings.Split(strings.Trim(strings.TrimSpace(name), "/"), "/")
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

func parseGCPTextToSpeechOperationName(name string) (project, location, operationID string, ok bool) {
	parts := strings.Split(strings.Trim(strings.TrimSpace(name), "/"), "/")
	if len(parts) != 6 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "operations" {
		return "", "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	operationID = strings.TrimSpace(parts[5])
	if project == "" || location == "" || operationID == "" {
		return "", "", "", false
	}
	return project, location, operationID, true
}

func parseGCPTextToSpeechQueryPagination(w http.ResponseWriter, r *http.Request, path string) (pageSize, start int, ok bool) {
	pageSize, err := parseOptionalNonNegativeInt(r.URL.Query().Get("pageSize"))
	if err != nil {
		respondGCPTextToSpeechInvalidArgument(w, path, "pageSize must be a non-negative integer")
		return 0, 0, false
	}
	if pageSize > 1000 {
		respondGCPTextToSpeechInvalidArgument(w, path, "pageSize must be <= 1000")
		return 0, 0, false
	}
	start = 0
	pageToken := strings.TrimSpace(r.URL.Query().Get("pageToken"))
	if pageToken != "" {
		start, err = strconv.Atoi(pageToken)
		if err != nil || start < 0 {
			respondGCPTextToSpeechInvalidArgument(w, path, "pageToken must be a non-negative integer offset")
			return 0, 0, false
		}
	}
	return pageSize, start, true
}

func parseGCPTextToSpeechBodyPagination(w http.ResponseWriter, path string, body map[string]any) (pageSize, start int, ok bool) {
	pageSize = 0
	if parsed, exists := gcpTextToSpeechInt(body, "pageSize", "page_size"); exists {
		if parsed < 0 {
			respondGCPTextToSpeechInvalidArgument(w, path, "pageSize must be a non-negative integer")
			return 0, 0, false
		}
		if parsed > 1000 {
			respondGCPTextToSpeechInvalidArgument(w, path, "pageSize must be <= 1000")
			return 0, 0, false
		}
		pageSize = parsed
	}
	start = 0
	pageToken := strings.TrimSpace(gcpTextToSpeechString(body, "pageToken", "page_token"))
	if pageToken != "" {
		parsed, err := strconv.Atoi(pageToken)
		if err != nil || parsed < 0 {
			respondGCPTextToSpeechInvalidArgument(w, path, "pageToken must be a non-negative integer offset")
			return 0, 0, false
		}
		start = parsed
	}
	return pageSize, start, true
}

func respondGCPTextToSpeechList(w http.ResponseWriter, key string, items []map[string]any, pageSize, start int, path string) bool {
	if start > len(items) {
		respondGCPTextToSpeechInvalidArgument(w, path, "pageToken is out of range")
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

func decodeGCPTextToSpeechJSONBody(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
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
		respondGCPTextToSpeechInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if body == nil {
		body = map[string]any{}
	}
	return body, true
}

func validateGCPTextToSpeechSynthesisInput(w http.ResponseWriter, path string, input map[string]any) bool {
	text := strings.TrimSpace(gcpTextToSpeechString(input, "text"))
	ssml := strings.TrimSpace(gcpTextToSpeechString(input, "ssml"))
	if text == "" && ssml == "" {
		respondGCPTextToSpeechInvalidArgument(w, path, "input.text or input.ssml is required")
		return false
	}
	if text != "" && ssml != "" {
		respondGCPTextToSpeechInvalidArgument(w, path, "input must provide exactly one of text or ssml")
		return false
	}
	return true
}

func validateGCPTextToSpeechAudioConfig(w http.ResponseWriter, path string, audioConfig map[string]any) bool {
	rawEncoding, ok := gcpTextToSpeechLookup(audioConfig, "audioEncoding", "audio_encoding")
	if !ok {
		respondGCPTextToSpeechInvalidArgument(w, path, "audio_config.audio_encoding is required")
		return false
	}
	if !isGCPTextToSpeechEnumValueSet(rawEncoding, "AUDIO_ENCODING_UNSPECIFIED") {
		respondGCPTextToSpeechInvalidArgument(w, path, "audio_config.audio_encoding is invalid")
		return false
	}
	if sampleRate, ok := gcpTextToSpeechInt(audioConfig, "sampleRateHertz", "sample_rate_hertz"); ok && sampleRate <= 0 {
		respondGCPTextToSpeechInvalidArgument(w, path, "audio_config.sample_rate_hertz must be positive")
		return false
	}
	return true
}

func validateGCPTextToSpeechVoice(w http.ResponseWriter, path string, voice map[string]any) bool {
	if rawGender, ok := gcpTextToSpeechLookup(voice, "ssmlGender", "ssml_gender"); ok && !isGCPTextToSpeechEnumValueSet(rawGender, "SSML_VOICE_GENDER_UNSPECIFIED") {
		respondGCPTextToSpeechInvalidArgument(w, path, "voice.ssml_gender is invalid")
		return false
	}
	if rawLanguage, ok := gcpTextToSpeechLookup(voice, "languageCode", "language_code"); ok && strings.TrimSpace(fmt.Sprint(rawLanguage)) == "" {
		respondGCPTextToSpeechInvalidArgument(w, path, "voice.language_code is invalid")
		return false
	}
	return true
}

func isGCPTextToSpeechEnumValueSet(raw any, unspecifiedToken string) bool {
	switch v := raw.(type) {
	case json.Number:
		parsed, err := v.Int64()
		return err == nil && parsed > 0
	case float64:
		return int64(v) > 0
	case int:
		return v > 0
	case int32:
		return v > 0
	case int64:
		return v > 0
	case string:
		token := strings.TrimSpace(v)
		return token != "" && token != unspecifiedToken
	default:
		return false
	}
}

func gcpTextToSpeechLocation(project, location string) map[string]any {
	return map[string]any{
		"name":        fmt.Sprintf("projects/%s/locations/%s", project, location),
		"locationId":  location,
		"displayName": "Text-to-Speech " + location,
	}
}

func gcpTextToSpeechOperationFixtures(project, location string) []map[string]any {
	return []map[string]any{
		gcpTextToSpeechOperation(project, location, "synthesizeLongAudio.op-1"),
		gcpTextToSpeechOperation(project, location, "synthesizeLongAudio.op-2"),
	}
}

func gcpTextToSpeechOperation(project, location, operationID string) map[string]any {
	name := fmt.Sprintf("projects/%s/locations/%s/operations/%s", project, location, operationID)
	return map[string]any{
		"name": name,
		"metadata": map[string]any{
			"@type":          "type.googleapis.com/google.cloud.texttospeech.v1.SynthesizeLongAudioMetadata",
			"startTime":      gcpTextToSpeechReferenceTime.Format(time.RFC3339),
			"lastUpdateTime": gcpTextToSpeechReferenceTime.Add(2 * time.Second).Format(time.RFC3339),
		},
		"done": true,
		"response": map[string]any{
			"@type": "type.googleapis.com/google.cloud.texttospeech.v1.SynthesizeLongAudioResponse",
		},
	}
}

func gcpTextToSpeechVoices() []map[string]any {
	return []map[string]any{
		{
			"languageCodes":          []string{"en-US", "en"},
			"name":                   "en-US-Standard-A",
			"ssmlGender":             "FEMALE",
			"naturalSampleRateHertz": 24000,
		},
		{
			"languageCodes":          []string{"en-GB", "en"},
			"name":                   "en-GB-Standard-B",
			"ssmlGender":             "MALE",
			"naturalSampleRateHertz": 22050,
		},
		{
			"languageCodes":          []string{"es-ES"},
			"name":                   "es-ES-Standard-A",
			"ssmlGender":             "FEMALE",
			"naturalSampleRateHertz": 24000,
		},
	}
}

func gcpTextToSpeechFilterVoices(voices []map[string]any, languageCode string) []map[string]any {
	languageCode = strings.ToLower(strings.TrimSpace(languageCode))
	if languageCode == "" {
		return voices
	}
	filtered := make([]map[string]any, 0, len(voices))
	for _, voice := range voices {
		rawCodes, _ := voice["languageCodes"].([]string)
		matched := false
		for _, raw := range rawCodes {
			candidate := strings.ToLower(strings.TrimSpace(raw))
			if candidate == languageCode || strings.HasPrefix(candidate, languageCode+"-") || strings.HasPrefix(languageCode, candidate+"-") {
				matched = true
				break
			}
		}
		if matched {
			filtered = append(filtered, voice)
		}
	}
	return filtered
}

func gcpTextToSpeechBodyMap(body map[string]any, keys ...string) map[string]any {
	for _, key := range keys {
		if nested, ok := body[key].(map[string]any); ok && len(nested) > 0 {
			return nested
		}
	}
	return map[string]any{}
}

func gcpTextToSpeechString(body map[string]any, keys ...string) string {
	for _, key := range keys {
		value, ok := body[key]
		if !ok {
			continue
		}
		if str, ok := value.(string); ok {
			return strings.TrimSpace(str)
		}
	}
	return ""
}

func gcpTextToSpeechLookup(body map[string]any, keys ...string) (any, bool) {
	for _, key := range keys {
		if value, ok := body[key]; ok {
			return value, true
		}
	}
	return nil, false
}

func gcpTextToSpeechInt(body map[string]any, keys ...string) (int, bool) {
	for _, key := range keys {
		raw, ok := body[key]
		if !ok {
			continue
		}
		switch typed := raw.(type) {
		case int:
			return typed, true
		case int32:
			return int(typed), true
		case int64:
			return int(typed), true
		case float64:
			return int(typed), true
		case json.Number:
			parsed, err := typed.Int64()
			if err == nil {
				return int(parsed), true
			}
		case string:
			parsed, err := strconv.Atoi(strings.TrimSpace(typed))
			if err == nil {
				return parsed, true
			}
		}
	}
	return 0, false
}

func isGCPTextToSpeechMissingID(id string) bool {
	return strings.Contains(strings.ToLower(strings.TrimSpace(id)), "missing")
}

func respondGCPTextToSpeechInvalidArgument(w http.ResponseWriter, path, message string) {
	respondGCPTextToSpeechError(w, http.StatusBadRequest, "InvalidArgument", path, message)
}

func respondGCPTextToSpeechNotFound(w http.ResponseWriter, path, message string) {
	respondGCPTextToSpeechError(w, http.StatusNotFound, "NotFound", path, message)
}

func respondGCPTextToSpeechError(w http.ResponseWriter, status int, err, path, message string) {
	respondJSON(w, status, map[string]any{
		"error":    err,
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func handleGCPContractProbe_texttospeech(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "texttospeech") {
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
			"name":     "projects/stackyard/locations/us-central1/operations/synthesizeLongAudio.sample",
			"service":  "texttospeech",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}
	return false
}
