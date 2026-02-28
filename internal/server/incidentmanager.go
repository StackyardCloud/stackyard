package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

type incidentManagerError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

var incidentManagerContactOperationByName = map[string]struct{}{
	"AcceptPage":                {},
	"ActivateContactChannel":    {},
	"CreateContact":             {},
	"CreateContactChannel":      {},
	"CreateRotation":            {},
	"CreateRotationOverride":    {},
	"DeactivateContactChannel":  {},
	"DeleteContact":             {},
	"DeleteContactChannel":      {},
	"DeleteRotation":            {},
	"DeleteRotationOverride":    {},
	"DescribeEngagement":        {},
	"DescribePage":              {},
	"GetContact":                {},
	"GetContactChannel":         {},
	"GetContactPolicy":          {},
	"GetRotation":               {},
	"GetRotationOverride":       {},
	"ListContactChannels":       {},
	"ListContacts":              {},
	"ListEngagements":           {},
	"ListPageReceipts":          {},
	"ListPageResolutions":       {},
	"ListPagesByContact":        {},
	"ListPagesByEngagement":     {},
	"ListPreviewRotationShifts": {},
	"ListRotationOverrides":     {},
	"ListRotations":             {},
	"ListRotationShifts":        {},
	"PutContactPolicy":          {},
	"SendActivationCode":        {},
	"StartEngagement":           {},
	"StopEngagement":            {},
	"UpdateContact":             {},
	"UpdateContactChannel":      {},
	"UpdateRotation":            {},
}

func incidentManagerIsContactAction(action string) bool {
	_, ok := incidentManagerContactOperationByName[action]
	return ok
}

func (s *Server) handleIncidentManagerRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isIncidentManagerCandidate(r) {
		return false
	}

	action, signingService, usesJSONRPC := resolveIncidentManagerAction(r)
	if action == "" {
		respondIncidentManagerError(w, http.StatusBadRequest, "ValidationException", "missing action", usesJSONRPC)
		return true
	}
	if _, known := incidentManagerOperationByName[action]; !known {
		respondIncidentManagerError(w, http.StatusBadRequest, "ValidationException", "unknown action", usesJSONRPC)
		return true
	}
	if signingService == "" {
		if incidentManagerIsContactAction(action) {
			signingService = "ssm-contacts"
		} else {
			signingService = "ssm-incidents"
		}
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, signingService)
	if !ok {
		respondIncidentManagerError(w, status, code, msg, usesJSONRPC)
		return true
	}

	payload, err := parseIncidentManagerPayload(r)
	if err != nil {
		respondIncidentManagerError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body", usesJSONRPC)
		return true
	}

	response := s.incidentmanager.Handle(action, payload)
	if response == nil {
		response = map[string]any{}
	}
	respondIncidentManagerJSON(w, http.StatusOK, response, usesJSONRPC)
	return true
}

func isIncidentManagerCandidate(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}

	target := strings.TrimSpace(r.Header.Get("X-Amz-Target"))
	if strings.HasPrefix(target, "SSMContacts.") || strings.HasPrefix(target, "SSMIncidents.") {
		return true
	}

	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service == "ssm-incidents" || service == "ssm-contacts" {
		return true
	}
	if service != "" {
		return false
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	if strings.Contains(host, "ssm-incidents") ||
		strings.Contains(host, "ssm-contacts") ||
		strings.Contains(host, "incident-manager") ||
		strings.Contains(host, "incidentmanager") {
		return true
	}

	action := incidentManagerActionFromPath(rawRequestPath(r))
	if action == "" {
		return false
	}
	_, known := incidentManagerOperationByName[action]
	return known
}

func resolveIncidentManagerAction(r *http.Request) (action string, signingService string, usesJSONRPC bool) {
	target := strings.TrimSpace(r.Header.Get("X-Amz-Target"))
	if strings.HasPrefix(target, "SSMContacts.") {
		return strings.TrimPrefix(target, "SSMContacts."), "ssm-contacts", true
	}
	if strings.HasPrefix(target, "SSMIncidents.") {
		return strings.TrimPrefix(target, "SSMIncidents."), "ssm-incidents", true
	}

	action = incidentManagerActionFromPath(rawRequestPath(r))
	if action == "" {
		return "", "", false
	}
	if incidentManagerIsContactAction(action) {
		return action, "ssm-contacts", false
	}
	return action, "ssm-incidents", false
}

func incidentManagerActionFromPath(requestPath string) string {
	path := strings.TrimSpace(strings.SplitN(requestPath, "?", 2)[0])
	if path == "" || path == "/" {
		return ""
	}
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 || strings.TrimSpace(parts[0]) == "" {
		return ""
	}
	token := strings.TrimSpace(parts[0])
	return strings.ToUpper(token[:1]) + token[1:]
}

func parseIncidentManagerPayload(r *http.Request) (map[string]any, error) {
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

func respondIncidentManagerJSON(w http.ResponseWriter, status int, body any, usesJSONRPC bool) {
	if usesJSONRPC {
		w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	} else {
		w.Header().Set("Content-Type", "application/json")
	}
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondIncidentManagerError(w http.ResponseWriter, status int, code, msg string, usesJSONRPC bool) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondIncidentManagerJSON(w, status, incidentManagerError{Type: code, Message: msg}, usesJSONRPC)
}
