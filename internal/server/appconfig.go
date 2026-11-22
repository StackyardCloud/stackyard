package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type appConfigError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleAppConfigRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isAppConfigRESTCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "appconfig")
	if !ok {
		respondAppConfigError(w, status, code, msg)
		return true
	}

	action, pathParams := parseAppConfigRoute(r.Method, rawRequestPath(r), r.Header.Get("X-Amz-Target"))
	if action == "" {
		respondAppConfigError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}
	if _, known := appConfigOperationByName[action]; !known {
		respondAppConfigError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	var payload map[string]any
	if action == "CreateHostedConfigurationVersion" {
		// The body is the configuration itself (any bytes, not a JSON
		// request); the metadata travels in headers.
		body, err := readBodyBytes(r)
		if err != nil {
			respondAppConfigError(w, http.StatusBadRequest, "BadRequestException", "unable to read request body")
			return true
		}
		payload = map[string]any{
			appConfigDataContentKey: body,
			"ContentType":           r.Header.Get("Content-Type"),
			"VersionLabel":          r.Header.Get("VersionLabel"),
			"Description":           r.Header.Get("Description"),
		}
	} else {
		var err error
		if payload, err = parseAppConfigPayload(r); err != nil {
			respondAppConfigError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
			return true
		}
	}

	response, fault := s.appconfig.Handle(action, payload, pathParams, r.URL.Query())
	if fault != nil {
		respondAppConfigError(w, fault.Status, fault.Code, fault.Message)
		return true
	}
	if response == nil {
		response = map[string]any{}
	}
	if action == "CreateHostedConfigurationVersion" || action == "GetHostedConfigurationVersion" {
		status := http.StatusOK
		if action == "CreateHostedConfigurationVersion" {
			status = http.StatusCreated
		}
		respondAppConfigHostedVersion(w, status, response)
		return true
	}
	respondAppConfigJSON(w, http.StatusOK, response)
	return true
}

// respondAppConfigHostedVersion writes a HostedConfigurationVersion the way
// the API does: the content is the body and everything else is a header.
func respondAppConfigHostedVersion(w http.ResponseWriter, status int, version map[string]any) {
	h := w.Header()
	headerValue := func(key string) string {
		if version[key] == nil {
			return ""
		}
		return fmt.Sprint(version[key])
	}
	if v := headerValue("ApplicationId"); v != "" {
		h.Set("Application-Id", v)
	}
	if v := headerValue("ConfigurationProfileId"); v != "" {
		h.Set("Configuration-Profile-Id", v)
	}
	if v := headerValue("VersionNumber"); v != "" {
		h.Set("Version-Number", v)
	}
	if v := headerValue("Description"); v != "" {
		h.Set("Description", v)
	}
	if v := headerValue("VersionLabel"); v != "" {
		h.Set("VersionLabel", v)
	}
	contentType, _ := version["ContentType"].(string)
	content, _ := version["Content"].([]byte)
	writeAppConfigContent(w, status, contentType, content)
}

func isAppConfigRESTCandidate(r *http.Request) bool {
	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service != "" && service != "appconfig" {
		return false
	}
	if service == "appconfig" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	if strings.Contains(host, ".appconfig.") || strings.HasPrefix(host, "appconfig.") {
		return true
	}

	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	if strings.Contains(userAgent, "command#appconfig") || strings.Contains(userAgent, " appconfig/") {
		return true
	}

	path := strings.TrimSpace(rawRequestPath(r))
	return strings.HasPrefix(path, "/applications") ||
		strings.HasPrefix(path, "/deploymentstrategies") ||
		strings.HasPrefix(path, "/deployementstrategies") ||
		strings.HasPrefix(path, "/extensions") ||
		strings.HasPrefix(path, "/extensionassociations") ||
		strings.HasPrefix(path, "/settings") ||
		strings.HasPrefix(path, "/tags")
}

func parseAppConfigRoute(method, requestPath, target string) (string, map[string]string) {
	target = strings.TrimSpace(target)
	if target != "" {
		action := parseAppConfigTarget(target)
		if action != "" {
			return action, map[string]string{}
		}
	}

	path := strings.TrimSpace(strings.SplitN(requestPath, "?", 2)[0])
	if path == "" {
		path = "/"
	}
	for _, op := range appConfigOperations {
		if !strings.EqualFold(op.Method, method) {
			continue
		}
		params, ok := appConfigPathParams(op.URI, path)
		if ok {
			return op.Name, params
		}
	}
	return "", nil
}

func parseAppConfigTarget(target string) string {
	if target == "" {
		return ""
	}
	if strings.HasPrefix(target, "AppConfig_20191009.") {
		return strings.TrimPrefix(target, "AppConfig_20191009.")
	}
	if dot := strings.LastIndex(target, "."); dot >= 0 && dot+1 < len(target) {
		return target[dot+1:]
	}
	return target
}

func appConfigPathParams(template, actual string) (map[string]string, bool) {
	templatePath := strings.TrimSpace(strings.SplitN(template, "?", 2)[0])
	actualPath := strings.TrimSpace(strings.SplitN(actual, "?", 2)[0])
	tSegs := appConfigSplitPathSegments(templatePath)
	aSegs := appConfigSplitPathSegments(actualPath)
	if len(tSegs) != len(aSegs) {
		return nil, false
	}

	params := map[string]string{}
	for i := range tSegs {
		t := tSegs[i]
		a := aSegs[i]
		if strings.HasPrefix(t, "{") && strings.HasSuffix(t, "}") {
			name := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(t, "{"), "}"))
			if name == "" {
				return nil, false
			}
			value, err := url.PathUnescape(a)
			if err != nil {
				value = a
			}
			params[name] = value
			continue
		}
		if t != a {
			return nil, false
		}
	}

	return params, true
}

func appConfigSplitPathSegments(path string) []string {
	path = strings.TrimSpace(strings.SplitN(path, "?", 2)[0])
	if path == "" || path == "/" {
		return nil
	}
	parts := strings.Split(strings.Trim(path, "/"), "/")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func parseAppConfigPayload(r *http.Request) (map[string]any, error) {
	body, err := readBodyBytes(r)
	if err != nil {
		return nil, err
	}
	if len(bytes.TrimSpace(body)) == 0 {
		return map[string]any{}, nil
	}

	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if contentType != "" && !strings.Contains(contentType, "json") {
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

func respondAppConfigJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondAppConfigError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondAppConfigJSON(w, status, appConfigError{Type: code, Message: msg})
}

// writeAppConfigContent writes caller-supplied configuration bytes. Only known
// non-HTML media types are echoed, each as a constant; anything else, including
// text/html, is served as application/octet-stream, always with nosniff. The
// header and the body write stay in one function so the content type is
// visibly safe at the point the bytes are written.
func writeAppConfigContent(w http.ResponseWriter, status int, contentType string, content []byte) {
	h := w.Header()
	h.Set("X-Content-Type-Options", "nosniff")
	switch strings.ToLower(strings.TrimSpace(strings.SplitN(contentType, ";", 2)[0])) {
	case "application/json":
		h.Set("Content-Type", "application/json")
	case "application/x-yaml":
		h.Set("Content-Type", "application/x-yaml")
	case "application/yaml":
		h.Set("Content-Type", "application/yaml")
	case "text/yaml":
		h.Set("Content-Type", "text/yaml")
	case "text/plain":
		h.Set("Content-Type", "text/plain")
	default:
		h.Set("Content-Type", "application/octet-stream")
	}
	w.WriteHeader(status)
	_, _ = w.Write(content)
}
