package server

import (
	"net/http"
	"strings"
)

// backupMaxBody bounds a request body; the largest inputs (plan JSON,
// framework controls, restore metadata) are far below this.
const backupMaxBody = 1 << 20

// handleBackupRESTRouter serves AWS Backup (rest-json, signing name "backup").
// Its paths (/tags/, /resources/, /audit/ ...) are generic, so a request is
// only claimed when it is signed for backup or sent to a backup host.
func (s *Server) handleBackupRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isBackupRESTCandidate(r) {
		return false
	}
	op, params := matchBackupRoute(r.Method, rawRequestPath(r))
	if op == nil {
		respondBedrockAgentError(w, &bedrockAgentError{status: http.StatusNotFound, code: "UnknownOperationException", msg: "no AWS Backup operation matches " + r.Method + " " + rawRequestPath(r)})
		return true
	}
	// A declared length bounds the body (net/http stops reading there), so
	// require one; chunked bodies of unknown length are refused. r.Body is
	// not reassigned (CodeQL treats that as a fresh request source).
	if r.ContentLength < 0 || r.ContentLength > backupMaxBody {
		respondBedrockAgentError(w, bkInvalid("request body must declare a Content-Length of at most 1 MiB"))
		return true
	}
	if ok, status, code, msg, _ := s.validateSigV4WithService(r, "backup"); !ok {
		respondBedrockAgentError(w, &bedrockAgentError{status: status, code: code, msg: msg})
		return true
	}
	in, err := decodeBedrockAgentBody(r)
	if err != nil {
		respondBedrockAgentError(w, bkInvalid("invalid JSON body"))
		return true
	}
	for k, v := range params {
		in[k] = v
	}
	q := r.URL.Query()
	for _, pair := range strings.Split(op.Query, ",") {
		name, member, ok := strings.Cut(pair, "=")
		if ok && q.Has(name) {
			in[member] = q.Get(name)
		}
	}
	out, apiErr := s.backup.handle(s, op.Name, in)
	if apiErr != nil {
		respondBedrockAgentError(w, apiErr)
		return true
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(op.Status)
	if op.Status != http.StatusNoContent {
		_, _ = w.Write(out)
	}
	return true
}

func isBackupRESTCandidate(r *http.Request) bool {
	switch r.Method {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete:
	default:
		return false
	}
	if sigV4ServiceHint(r) == "backup" {
		return true
	}
	host := strings.ToLower(r.Host)
	return strings.HasPrefix(host, "backup.") && strings.Contains(host, ".amazonaws.")
}

// matchBackupRoute finds the operation whose URI template matches; fewer
// variables win, so a literal segment beats a variable one.
func matchBackupRoute(method, path string) (*backupOperation, map[string]string) {
	var best *backupOperation
	var bestParams map[string]string
	for i := range backupOperations {
		op := &backupOperations[i]
		if op.Method != method {
			continue
		}
		params, ok := bedrockAgentCoreControlPathParams(op.URI, path)
		if !ok {
			continue
		}
		if best == nil || len(params) < len(bestParams) {
			best, bestParams = op, params
		}
	}
	return best, bestParams
}

// All AWS Backup errors are HTTP 400 except ServiceUnavailableException.
func bkErr(code, msg string) *bedrockAgentError {
	return &bedrockAgentError{status: http.StatusBadRequest, code: code, msg: msg}
}

func bkInvalid(msg string) *bedrockAgentError { return bkErr("InvalidParameterValueException", msg) }
func bkMissing(msg string) *bedrockAgentError { return bkErr("MissingParameterValueException", msg) }
func bkRequest(msg string) *bedrockAgentError { return bkErr("InvalidRequestException", msg) }
func bkNotFound(kind, id string) *bedrockAgentError {
	return bkErr("ResourceNotFoundException", kind+" "+id+" does not exist")
}
func bkExists(kind, id string) *bedrockAgentError {
	return bkErr("AlreadyExistsException", kind+" "+id+" already exists")
}
