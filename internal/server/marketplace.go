package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

type marketplaceError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleMarketplaceRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isMarketplaceRESTCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "aws-marketplace")
	if !ok {
		respondMarketplaceError(w, status, code, msg)
		return true
	}

	action := parseMarketplaceRoute(r.Method, rawRequestPath(r))
	if action == "" {
		respondMarketplaceError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}
	if _, known := marketplaceOperationByName[action]; !known {
		respondMarketplaceError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	payload, err := parseMarketplacePayload(r)
	if err != nil {
		respondMarketplaceError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.marketplace.Handle(action, payload)
	if response == nil {
		response = map[string]any{}
	}
	respondMarketplaceJSON(w, http.StatusOK, response)
	return true
}

func isMarketplaceRESTCandidate(r *http.Request) bool {
	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service != "" && service != "aws-marketplace" {
		return false
	}
	if service == "aws-marketplace" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	if strings.Contains(host, ".marketplace.") || strings.HasPrefix(host, "marketplace.") {
		return true
	}

	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	if strings.Contains(userAgent, "marketplace") {
		return true
	}

	path := strings.TrimSpace(strings.SplitN(rawRequestPath(r), "?", 2)[0])
	return strings.HasPrefix(path, "/marketplace/")
}

func parseMarketplaceRoute(method, requestPath string) string {
	path := strings.TrimSpace(strings.SplitN(requestPath, "?", 2)[0])
	if path == "" {
		path = "/"
	}
	for _, op := range marketplaceOperations {
		if !strings.EqualFold(op.Method, method) {
			continue
		}
		if op.URI == path {
			return op.Name
		}
	}
	return ""
}

func parseMarketplacePayload(r *http.Request) (map[string]any, error) {
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

func respondMarketplaceJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondMarketplaceError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondMarketplaceJSON(w, status, marketplaceError{Type: code, Message: msg})
}
