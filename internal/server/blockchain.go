package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

type blockchainError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleBlockchainRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isBlockchainRESTRouterCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "managedblockchain-query")
	if !ok {
		respondBlockchainError(w, status, code, msg)
		return true
	}

	action, pathParams := parseBlockchainRoute(r.Method, rawRequestPath(r))
	if action == "" {
		respondBlockchainError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}
	if _, known := blockchainOperationByName[action]; !known {
		respondBlockchainError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	payload, err := parseBlockchainPayload(r)
	if err != nil {
		respondBlockchainError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.blockchain.Handle(action, payload, pathParams, r.URL.Query())
	if response == nil {
		response = map[string]any{}
	}
	respondBlockchainJSON(w, http.StatusOK, response)
	return true
}

func isBlockchainRESTRouterCandidate(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}

	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service != "" && service != "managedblockchain-query" {
		return false
	}
	if service == "managedblockchain-query" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	if strings.Contains(host, ".managedblockchain-query.") ||
		strings.HasPrefix(host, "managedblockchain-query.") ||
		strings.Contains(host, ".managedblockchain.") ||
		strings.HasPrefix(host, "managedblockchain.") {
		return true
	}

	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	if strings.Contains(userAgent, "command#managedblockchain-query") || strings.Contains(userAgent, " managedblockchain-query/") {
		return true
	}

	action, _ := parseBlockchainRoute(r.Method, rawRequestPath(r))
	return action != ""
}

func parseBlockchainRoute(method, requestPath string) (string, map[string]string) {
	path := blockchainNormalizePath(requestPath)
	for _, op := range blockchainOperations {
		if !strings.EqualFold(op.Method, method) {
			continue
		}
		if blockchainNormalizePath(op.URI) == path {
			return op.Name, map[string]string{}
		}
	}
	return "", nil
}

func blockchainNormalizePath(raw string) string {
	path := strings.TrimSpace(strings.SplitN(raw, "?", 2)[0])
	if path == "" {
		return "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if path != "/" {
		path = strings.TrimSuffix(path, "/")
	}
	return path
}

func parseBlockchainPayload(r *http.Request) (map[string]any, error) {
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

func respondBlockchainJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondBlockchainError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondBlockchainJSON(w, status, blockchainError{Type: code, Message: msg})
}
