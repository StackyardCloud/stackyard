package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

type apiGatewayV2Error struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleAPIGatewayV2RESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isAPIGatewayV2RESTCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "apigateway")
	if !ok {
		respondAPIGatewayV2Error(w, status, code, msg)
		return true
	}

	action := parseAPIGatewayV2Route(r.Method, rawRequestPath(r))
	if action == "" {
		respondAPIGatewayV2Error(w, http.StatusBadRequest, "BadRequestException", "unknown action")
		return true
	}
	if _, known := apiGatewayV2OperationByName[action]; !known {
		respondAPIGatewayV2Error(w, http.StatusBadRequest, "BadRequestException", "unknown action")
		return true
	}

	payload, err := parseAPIGatewayV2Payload(r)
	if err != nil {
		respondAPIGatewayV2Error(w, http.StatusBadRequest, "BadRequestException", "invalid JSON body")
		return true
	}

	response := s.apigatewayv2.Handle(action, payload)
	if response == nil {
		response = map[string]any{}
	}
	respondAPIGatewayV2JSON(w, http.StatusOK, response)
	return true
}

func isAPIGatewayV2RESTCandidate(r *http.Request) bool {
	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service != "" && service != "apigateway" && service != "apigatewayv2" {
		return false
	}
	if service == "apigatewayv2" {
		return true
	}
	path := strings.TrimSpace(strings.SplitN(rawRequestPath(r), "?", 2)[0])
	if service == "apigateway" {
		if strings.HasPrefix(path, "/apigatewayv2/") {
			return true
		}
		userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
		return strings.Contains(userAgent, "command#apigatewayv2") || strings.Contains(userAgent, " apigatewayv2/")
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	if strings.Contains(host, ".apigateway.") || strings.HasPrefix(host, "apigateway.") {
		return true
	}

	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	if strings.Contains(userAgent, "command#apigatewayv2") || strings.Contains(userAgent, " apigatewayv2/") {
		return true
	}

	return strings.HasPrefix(path, "/apigatewayv2/")
}

func parseAPIGatewayV2Route(method, requestPath string) string {
	path := strings.TrimSpace(strings.SplitN(requestPath, "?", 2)[0])
	if path == "" {
		path = "/"
	}
	for _, op := range apiGatewayV2Operations {
		if !strings.EqualFold(op.Method, method) {
			continue
		}
		if op.URI == path {
			return op.Name
		}
	}
	return ""
}

func parseAPIGatewayV2Payload(r *http.Request) (map[string]any, error) {
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

func respondAPIGatewayV2JSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondAPIGatewayV2Error(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondAPIGatewayV2JSON(w, status, apiGatewayV2Error{Type: code, Message: msg})
}
