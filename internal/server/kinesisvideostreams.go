package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

type kinesisVideoStreamsError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleKinesisVideoStreamsRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isKinesisVideoStreamsRESTCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "kinesisvideo")
	if !ok {
		respondKinesisVideoStreamsError(w, status, code, msg)
		return true
	}

	action := parseKinesisVideoStreamsRoute(r.Method, rawRequestPath(r))
	if action == "" {
		respondKinesisVideoStreamsError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}
	if _, known := kinesisVideoStreamsOperationByName[action]; !known {
		respondKinesisVideoStreamsError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	payload, err := parseKinesisVideoStreamsPayload(r)
	if err != nil {
		respondKinesisVideoStreamsError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.kinesisvideostreams.Handle(action, payload)
	if response == nil {
		response = map[string]any{}
	}
	respondKinesisVideoStreamsJSON(w, http.StatusOK, response)
	return true
}

func isKinesisVideoStreamsRESTCandidate(r *http.Request) bool {
	if strings.ToUpper(strings.TrimSpace(r.Method)) != http.MethodPost {
		return false
	}

	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service != "" && service != "kinesisvideo" {
		return false
	}
	if service == "kinesisvideo" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	if strings.Contains(host, "kinesisvideo") || strings.Contains(host, "kinesis-video") {
		return true
	}

	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	if strings.Contains(userAgent, "command#kinesisvideo") || strings.Contains(userAgent, " kinesisvideo/") {
		return true
	}

	action := parseKinesisVideoStreamsRoute(r.Method, rawRequestPath(r))
	return action != ""
}

func parseKinesisVideoStreamsRoute(method, requestPath string) string {
	path := strings.TrimSpace(strings.SplitN(requestPath, "?", 2)[0])
	if path == "" {
		path = "/"
	}

	for _, op := range kinesisVideoStreamsOperations {
		if !strings.EqualFold(op.Method, method) {
			continue
		}
		if strings.EqualFold(op.URI, path) {
			return op.Name
		}
	}
	return ""
}

func parseKinesisVideoStreamsPayload(r *http.Request) (map[string]any, error) {
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

func respondKinesisVideoStreamsJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondKinesisVideoStreamsError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondKinesisVideoStreamsJSON(w, status, kinesisVideoStreamsError{Type: code, Message: msg})
}
