package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
)

// CloudTrail Data (signing name "cloudtrail-data") is a REST-JSON data plane with a
// single operation, PutAuditEvents, that ingests events into channels owned by the
// CloudTrail control-plane store (s.cloudtrail).

const cloudtraildataMaxBodyBytes = 1 << 20 // PutAuditEvents accepts up to 1 MB per request.

var (
	cloudtraildataEventIDPattern    = regexp.MustCompile(`^[-_A-Za-z0-9]{1,128}$`)
	cloudtraildataExternalIDPattern = regexp.MustCompile(`^[\w+=,.@:/-]+$`)
)

type cloudtraildataAuditEvent struct {
	EventData         *string `json:"eventData"`
	EventDataChecksum string  `json:"eventDataChecksum"`
	ID                string  `json:"id"`
}

type cloudtraildataPutRequest struct {
	AuditEvents []cloudtraildataAuditEvent `json:"auditEvents"`
}

type cloudtraildataSuccess struct {
	EventID string `json:"eventID"`
	ID      string `json:"id"`
}

type cloudtraildataFailure struct {
	ErrorCode    string `json:"errorCode"`
	ErrorMessage string `json:"errorMessage"`
	ID           string `json:"id"`
}

type cloudtraildataPutResponse struct {
	Failed     []cloudtraildataFailure `json:"failed"`
	Successful []cloudtraildataSuccess `json:"successful"`
}

// cloudtraildataFault is a request-level error; per-event problems go in the failed list.
type cloudtraildataFault struct {
	Code    string
	Message string
}

func (s *Server) handleCloudTrailDataRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isCloudTrailDataCandidate(r) {
		return false
	}
	if ok, status, code, msg, _ := s.validateSigV4WithService(r, "cloudtrail-data"); !ok {
		respondCloudTrailDataError(w, status, code, msg)
		return true
	}
	if r.Method != http.MethodPost || rawRequestPath(r) != "/PutAuditEvents" {
		respondCloudTrailDataError(w, http.StatusNotImplemented, "NotImplementedException", "unsupported CloudTrail Data route")
		return true
	}
	s.handleCloudTrailDataPutAuditEvents(w, r)
	return true
}

func isCloudTrailDataCandidate(r *http.Request) bool {
	if sigV4ServiceHint(r) == "cloudtrail-data" {
		return true
	}
	host := strings.ToLower(strings.TrimSpace(r.Host))
	return strings.HasPrefix(host, "cloudtrail-data.") || strings.Contains(host, ".cloudtrail-data.")
}

func (s *Server) handleCloudTrailDataPutAuditEvents(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	channelArn := strings.TrimSpace(query.Get("channelArn"))
	if channelArn == "" {
		respondCloudTrailDataError(w, http.StatusBadRequest, "ValidationException", "channelArn is required")
		return
	}
	externalID := query.Get("externalId")
	if _, present := query["externalId"]; present && (len(externalID) < 2 || len(externalID) > 1224 || !cloudtraildataExternalIDPattern.MatchString(externalID)) {
		respondCloudTrailDataError(w, http.StatusBadRequest, "ValidationException", "invalid externalId")
		return
	}

	body, err := readBodyBytes(r)
	if err != nil {
		respondCloudTrailDataError(w, http.StatusBadRequest, "ValidationException", "unable to read request body")
		return
	}
	if len(body) > cloudtraildataMaxBodyBytes {
		respondCloudTrailDataError(w, http.StatusBadRequest, "ValidationException", "request payload exceeds 1 MB")
		return
	}
	var req cloudtraildataPutRequest
	if err := json.Unmarshal(bytes.TrimSpace(body), &req); err != nil {
		respondCloudTrailDataError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return
	}
	if fault := validateCloudTrailDataEvents(req.AuditEvents); fault != nil {
		respondCloudTrailDataError(w, http.StatusBadRequest, fault.Code, fault.Message)
		return
	}

	resp, fault := s.cloudtrail.PutAuditEvents(channelArn, externalID, req.AuditEvents)
	if fault != nil {
		respondCloudTrailDataError(w, http.StatusBadRequest, fault.Code, fault.Message)
		return
	}
	respondCloudTrailDataJSON(w, http.StatusOK, resp)
}

func validateCloudTrailDataEvents(events []cloudtraildataAuditEvent) *cloudtraildataFault {
	if len(events) < 1 || len(events) > 100 {
		return &cloudtraildataFault{"ValidationException", "auditEvents must contain between 1 and 100 events"}
	}
	seen := make(map[string]struct{}, len(events))
	for _, event := range events {
		if !cloudtraildataEventIDPattern.MatchString(event.ID) {
			return &cloudtraildataFault{"ValidationException", "each audit event requires an id matching [-_A-Za-z0-9]{1,128}"}
		}
		if _, dup := seen[event.ID]; dup {
			return &cloudtraildataFault{"DuplicatedAuditEventId", "duplicate audit event id " + event.ID}
		}
		seen[event.ID] = struct{}{}
	}
	return nil
}

func respondCloudTrailDataJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondCloudTrailDataError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondCloudTrailDataJSON(w, status, cloudTrailError{Type: code, Message: msg})
}
