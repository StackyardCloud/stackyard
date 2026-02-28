package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

type cloudWatchSyntheticsError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleCloudWatchSyntheticsRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isCloudWatchSyntheticsRESTRouterCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "synthetics")
	if !ok {
		respondCloudWatchSyntheticsError(w, status, code, msg)
		return true
	}

	action, pathParams := parseCloudWatchSyntheticsRoute(r.Method, r.URL.Path)
	if action == "" {
		respondCloudWatchSyntheticsError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	payload, err := parseCloudWatchSyntheticsPayload(r)
	if err != nil {
		respondCloudWatchSyntheticsError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.cloudwatchsynthetics.Handle(action, payload, pathParams, r.URL.Query())
	if response == nil {
		response = map[string]any{}
	}
	respondCloudWatchSyntheticsJSON(w, http.StatusOK, response)
	return true
}

func isCloudWatchSyntheticsRESTRouterCandidate(r *http.Request) bool {
	method := strings.ToUpper(strings.TrimSpace(r.Method))
	if method != http.MethodPost && method != http.MethodGet && method != http.MethodDelete && method != http.MethodPatch {
		return false
	}

	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service != "" && service != "synthetics" {
		return false
	}
	if service == "synthetics" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	if strings.Contains(host, ".synthetics.") || strings.HasPrefix(host, "synthetics.") || strings.Contains(host, "cloudwatchsynthetics") {
		return true
	}

	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	if strings.Contains(userAgent, "command#synthetics") || strings.Contains(userAgent, " synthetics/") {
		return true
	}

	path := strings.TrimSpace(r.URL.Path)
	return strings.HasPrefix(path, "/canary") ||
		strings.HasPrefix(path, "/canaries") ||
		strings.HasPrefix(path, "/group") ||
		strings.HasPrefix(path, "/groups") ||
		strings.HasPrefix(path, "/runtime-versions")
}

func parseCloudWatchSyntheticsRoute(method, rawPath string) (string, map[string]string) {
	for _, op := range cloudWatchSyntheticsOperations {
		if !strings.EqualFold(op.Method, method) {
			continue
		}
		if params, ok := matchCloudWatchSyntheticsPath(op.URI, rawPath); ok {
			return op.Name, params
		}
	}
	return "", nil
}

func matchCloudWatchSyntheticsPath(templatePath, rawPath string) (map[string]string, bool) {
	templateSeg := cloudWatchSyntheticsSplitPath(templatePath)
	pathSeg := cloudWatchSyntheticsSplitPath(rawPath)
	if len(templateSeg) != len(pathSeg) {
		return nil, false
	}

	params := map[string]string{}
	for i := range templateSeg {
		tseg := templateSeg[i]
		pseg := pathSeg[i]
		if strings.HasPrefix(tseg, "{") && strings.HasSuffix(tseg, "}") {
			name := strings.TrimSuffix(strings.TrimPrefix(tseg, "{"), "}")
			unescaped, err := url.PathUnescape(pseg)
			if err != nil {
				return nil, false
			}
			params[name] = unescaped
			continue
		}
		if tseg != pseg {
			return nil, false
		}
	}

	return params, true
}

func cloudWatchSyntheticsSplitPath(raw string) []string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return []string{}
	}
	trimmed = strings.TrimSuffix(trimmed, "/")
	if trimmed == "" {
		return []string{}
	}
	trimmed = strings.TrimPrefix(trimmed, "/")
	if trimmed == "" {
		return []string{}
	}
	return strings.Split(trimmed, "/")
}

func parseCloudWatchSyntheticsPayload(r *http.Request) (map[string]any, error) {
	body, err := readBodyBytes(r)
	if err != nil {
		return nil, err
	}
	if len(bytes.TrimSpace(body)) == 0 {
		return map[string]any{}, nil
	}

	var payload map[string]any
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()
	if err := dec.Decode(&payload); err != nil {
		return nil, err
	}
	if payload == nil {
		payload = map[string]any{}
	}
	return payload, nil
}

func respondCloudWatchSyntheticsJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondCloudWatchSyntheticsError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondCloudWatchSyntheticsJSON(w, status, cloudWatchSyntheticsError{Type: code, Message: msg})
}
