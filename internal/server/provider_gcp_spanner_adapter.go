package server

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const gcpSpannerAdapterDefaultSessionID = "as-1"

func (s *Server) handleGCPSpannerAdapterRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_spanner_adapter(w, r) {
		return true
	}

	path := normalizeGCPSpannerPath(rawRequestPath(r))
	if !isGCPSpannerAdapterPath(path, hasGCPSpannerAdapterHint(r)) {
		return false
	}

	if r.Method != http.MethodPost {
		return false
	}

	if handleGCPSpannerAdapterCreateSession(w, r, path) {
		return true
	}
	if handleGCPSpannerAdapterAdaptMessage(w, r, path) {
		return true
	}

	respondProviderNotImplemented(w, providerGCP, path)
	return true
}

func hasGCPSpannerAdapterHint(r *http.Request) bool {
	service := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Stackyard-GCP-Service")))
	switch service {
	case "spanner_adapter", "spanner-adapter", "spanner-adapter-apiv1", "spanner_adapter_apiv1", "cloud-spanner-adapter", "cloud_spanner_adapter", "cloudspanneradapter", "gcp-cloud-spanner-adapter":
		return true
	}
	ua := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	return strings.Contains(ua, "stackyard-spanner-adapter-apiv1") || strings.Contains(ua, "cloud.google.com/go/spanner/adapter")
}

func isGCPSpannerAdapterPath(path string, includeHint bool) bool {
	if _, _, _, ok := parseGCPSpannerAdapterCreateSessionPath(path); ok {
		return true
	}
	if _, _, _, _, ok := parseGCPSpannerAdapterAdaptMessagePath(path); ok {
		return true
	}
	if includeHint {
		return strings.HasPrefix(path, "/gcp/v1/projects/") &&
			strings.Contains(path, "/instances/") &&
			strings.Contains(path, "/databases/") &&
			strings.Contains(path, "/sessions")
	}
	return false
}

func handleGCPSpannerAdapterCreateSession(w http.ResponseWriter, r *http.Request, path string) bool {
	project, instance, database, ok := parseGCPSpannerAdapterCreateSessionPath(path)
	if !ok {
		return false
	}
	if isGCPSpannerMissingResource(project, instance, database) {
		respondGCPSpannerAdapterNotFound(w, path, "database not found")
		return true
	}

	body, valid := decodeGCPSpannerAdapterJSONBody(w, r, path)
	if !valid {
		return true
	}
	if len(body) == 0 {
		respondGCPSpannerAdapterInvalidArgument(w, path, "session is required")
		return true
	}

	sessionID := gcpSpannerAdapterDefaultSessionID
	if name := strings.TrimSpace(gcpSpannerAdapterString(body, "name")); name != "" {
		p, i, d, s, validName := parseGCPSpannerSessionName(name)
		if !validName {
			respondGCPSpannerAdapterInvalidArgument(w, path, "session.name is invalid")
			return true
		}
		if p != project || i != instance || d != database {
			respondGCPSpannerAdapterInvalidArgument(w, path, "session.name must match parent")
			return true
		}
		sessionID = s
	}

	respondJSON(w, http.StatusOK, gcpSpannerAdapterSessionFixture(project, instance, database, sessionID))
	return true
}

func handleGCPSpannerAdapterAdaptMessage(w http.ResponseWriter, r *http.Request, path string) bool {
	project, instance, database, sessionID, ok := parseGCPSpannerAdapterAdaptMessagePath(path)
	if !ok {
		return false
	}
	if isGCPSpannerMissingResource(project, instance, database, sessionID) {
		respondGCPSpannerAdapterNotFound(w, path, "session not found")
		return true
	}

	body, valid := decodeGCPSpannerAdapterJSONBody(w, r, path)
	if !valid {
		return true
	}

	name := strings.TrimSpace(gcpSpannerAdapterString(body, "name"))
	if name == "" {
		respondGCPSpannerAdapterInvalidArgument(w, path, "name is required")
		return true
	}
	p, i, d, s, validName := parseGCPSpannerSessionName(name)
	if !validName {
		respondGCPSpannerAdapterInvalidArgument(w, path, "name is invalid")
		return true
	}
	if p != project || i != instance || d != database || s != sessionID {
		respondGCPSpannerAdapterInvalidArgument(w, path, "name must match request path")
		return true
	}

	protocol := strings.TrimSpace(gcpSpannerAdapterString(body, "protocol"))
	if protocol == "" {
		respondGCPSpannerAdapterInvalidArgument(w, path, "protocol is required")
		return true
	}
	if strings.Contains(strings.ToLower(protocol), "unsupported") {
		respondGCPSpannerAdapterFailedPrecondition(w, path, "protocol is not supported")
		return true
	}

	if attachmentsRaw, ok := body["attachments"]; ok {
		attachments, ok := attachmentsRaw.(map[string]any)
		if !ok {
			respondGCPSpannerAdapterInvalidArgument(w, path, "attachments must be an object")
			return true
		}
		for key, value := range attachments {
			if strings.TrimSpace(key) == "" {
				respondGCPSpannerAdapterInvalidArgument(w, path, "attachments keys must be non-empty")
				return true
			}
			if _, ok := value.(string); !ok {
				respondGCPSpannerAdapterInvalidArgument(w, path, fmt.Sprintf("attachments[%q] must be a string", key))
				return true
			}
		}
	}

	payload, err := gcpSpannerAdapterPayloadFromAny(body["payload"])
	if err != nil {
		respondGCPSpannerAdapterInvalidArgument(w, path, "payload must be base64-encoded bytes")
		return true
	}

	respondJSON(w, http.StatusOK, gcpSpannerAdapterAdaptMessageFixture(protocol, sessionID, payload))
	return true
}

func parseGCPSpannerAdapterCreateSessionPath(path string) (project, instance, database string, ok bool) {
	p, i, d, tail, ok := parseGCPSpannerDatabaseTail(path)
	if !ok || len(tail) != 1 || tail[0] != "sessions:adapter" {
		return "", "", "", false
	}
	return p, i, d, true
}

func parseGCPSpannerAdapterAdaptMessagePath(path string) (project, instance, database, sessionID string, ok bool) {
	return parseGCPSpannerSessionActionPath(path, "adaptMessage")
}

func parseGCPSpannerAdapterDatabaseName(name string) (project, instance, database string, ok bool) {
	return parseGCPSpannerDatabaseName(name)
}

func decodeGCPSpannerAdapterJSONBody(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	if r.Body == nil {
		return map[string]any{}, true
	}
	defer r.Body.Close()

	limited := io.LimitReader(r.Body, 1<<20)
	dec := json.NewDecoder(limited)
	dec.UseNumber()

	var body map[string]any
	if err := dec.Decode(&body); err != nil {
		if err == io.EOF {
			return map[string]any{}, true
		}
		respondGCPSpannerAdapterInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if body == nil {
		body = map[string]any{}
	}
	return body, true
}

func gcpSpannerAdapterPayloadFromAny(value any) ([]byte, error) {
	if value == nil {
		return nil, nil
	}
	encoded, ok := value.(string)
	if !ok {
		return nil, fmt.Errorf("payload type")
	}
	if strings.TrimSpace(encoded) == "" {
		return []byte{}, nil
	}
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, err
	}
	return decoded, nil
}

func gcpSpannerAdapterSessionFixture(project, instance, database, sessionID string) map[string]any {
	return map[string]any{
		"name": fmt.Sprintf("projects/%s/instances/%s/databases/%s/sessions/%s", project, instance, database, sessionID),
	}
}

func gcpSpannerAdapterAdaptMessageFixture(protocol, sessionID string, payload []byte) map[string]any {
	adaptedPayload := payload
	if len(adaptedPayload) == 0 {
		adaptedPayload = []byte("stackyard-adapted-" + strings.ToLower(protocol))
	}
	return map[string]any{
		"payload": adaptedPayload,
		"stateUpdates": map[string]any{
			"adapter":  "stackyard",
			"protocol": protocol,
			"session":  sessionID,
		},
		"last": true,
	}
}

func gcpSpannerAdapterString(body map[string]any, key string) string {
	if body == nil {
		return ""
	}
	value, ok := body[key]
	if !ok || value == nil {
		return ""
	}
	text, _ := value.(string)
	return text
}

func respondGCPSpannerAdapterInvalidArgument(w http.ResponseWriter, path, message string) {
	respondGCPSpannerAdapterError(w, http.StatusBadRequest, "InvalidArgument", path, message)
}

func respondGCPSpannerAdapterNotFound(w http.ResponseWriter, path, message string) {
	respondGCPSpannerAdapterError(w, http.StatusNotFound, "NotFound", path, message)
}

func respondGCPSpannerAdapterFailedPrecondition(w http.ResponseWriter, path, message string) {
	respondGCPSpannerAdapterError(w, http.StatusBadRequest, "FailedPrecondition", path, message)
}

func respondGCPSpannerAdapterError(w http.ResponseWriter, status int, err, path, message string) {
	respondJSON(w, status, map[string]any{
		"error":    err,
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func handleGCPContractProbe_spanner_adapter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "spanner_adapter") {
		return false
	}
	if r.URL.Query().Get("pageSize") == "bad" {
		respondGCPSpannerAdapterInvalidArgument(w, path, "pageSize must be a non-negative integer")
		return true
	}
	if r.URL.Query().Get("typedSuccess") == "1" {
		respondJSON(w, http.StatusOK, map[string]any{
			"name":     "projects/stackyard/locations/us-central1/spanner_adapter/sample",
			"service":  "spanner_adapter",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}
	return false
}
