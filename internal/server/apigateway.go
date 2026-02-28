package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

type apiGatewayError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleAPIGatewayRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isAPIGatewayRESTCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "apigateway")
	if !ok {
		respondAPIGatewayError(w, status, code, msg)
		return true
	}

	action := parseAPIGatewayRoute(r.Method, rawRequestPath(r))
	if action == "" {
		respondAPIGatewayError(w, http.StatusBadRequest, "BadRequestException", "unknown action")
		return true
	}
	if _, known := apiGatewayOperationByName[action]; !known {
		respondAPIGatewayError(w, http.StatusBadRequest, "BadRequestException", "unknown action")
		return true
	}

	payload, err := parseAPIGatewayPayload(r)
	if err != nil {
		respondAPIGatewayError(w, http.StatusBadRequest, "BadRequestException", "invalid JSON body")
		return true
	}

	response := s.apigateway.Handle(action, payload)
	if response == nil {
		response = map[string]any{}
	}
	respondAPIGatewayJSON(w, http.StatusOK, response)
	return true
}

func isAPIGatewayRESTCandidate(r *http.Request) bool {
	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service != "" && service != "apigateway" {
		return false
	}
	if service == "apigateway" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	if strings.Contains(host, ".apigateway.") || strings.HasPrefix(host, "apigateway.") {
		return true
	}

	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	if strings.Contains(userAgent, "command#apigateway") || strings.Contains(userAgent, " apigateway/") {
		return true
	}

	path := strings.TrimSpace(strings.SplitN(rawRequestPath(r), "?", 2)[0])
	return strings.HasPrefix(path, "/apigateway/")
}

func parseAPIGatewayRoute(method, requestPath string) string {
	path := strings.TrimSpace(strings.SplitN(requestPath, "?", 2)[0])
	if path == "" {
		path = "/"
	}
	for _, op := range apiGatewayOperations {
		if !strings.EqualFold(op.Method, method) {
			continue
		}
		if op.URI == path {
			return op.Name
		}
	}
	return ""
}

func parseAPIGatewayPayload(r *http.Request) (map[string]any, error) {
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

func respondAPIGatewayJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondAPIGatewayError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondAPIGatewayJSON(w, status, apiGatewayError{Type: code, Message: msg})
}
