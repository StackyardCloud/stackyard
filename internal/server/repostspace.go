package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

// AWS re:Post Private: REST JSON under /spaces and /tags, signed repostspace.
// Spaces provision (CREATING -> CREATED) and delete (DELETING -> gone) on a
// timer settled on read; channels provision the same way. See
// repostspace_store.go for the state.

const (
	rpsSigningName = "repostspace"
	rpsMaxBody     = 1 << 20
	rpsContentType = "application/json"
)

type rpsOperation = sesv2Operation

// rpsErr is an AWS error: code is the exception shape, status its HTTP status.
type rpsErr struct {
	status       int
	code, msg    string
	resourceID   string
	resourceType string
}

func (s *Server) handleRepostSpaceRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if r.Header.Get("X-Amz-Target") != "" {
		return false
	}
	hint := sigV4ServiceHint(r)
	if hint != rpsSigningName && (hint != "" || !strings.HasPrefix(strings.ToLower(r.Host), rpsSigningName+".")) {
		return false
	}
	op, params, ok := matchSESV2OperationIn(repostSpaceOperations, r.Method, r.URL.EscapedPath())
	if !ok {
		writeRPSError(w, &rpsErr{status: http.StatusNotFound, code: "ResourceNotFoundException", msg: "no such operation"})
		return true
	}
	// A declared length bounds what net/http reads; refuse chunked bodies.
	if r.ContentLength < 0 || r.ContentLength > rpsMaxBody {
		writeRPSError(w, rpsValidation("request body must declare a Content-Length of at most 1 MiB"))
		return true
	}
	if ok, status, code, msg, _ := s.validateSigV4WithService(r, rpsSigningName); !ok {
		writeRPSError(w, &rpsErr{status: status, code: code, msg: msg})
		return true
	}
	model := awsmodels.RepostSpaceModel()
	in, problem := awsModelDecodeREST(r, model, model.Operations[op.Name].Input, params)
	if problem != "" {
		writeRPSError(w, rpsValidation("%s", problem))
		return true
	}
	body, rerr := s.repostSpace.run(op.Name, in)
	if rerr != nil {
		writeRPSError(w, rerr)
		return true
	}
	w.Header().Set("Content-Type", rpsContentType)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
	return true
}

func writeRPSError(w http.ResponseWriter, e *rpsErr) {
	payload := map[string]any{"message": e.msg}
	switch e.code {
	case "ValidationException":
		payload["reason"] = "fieldValidationFailed"
	case "ResourceNotFoundException", "ConflictException", "ServiceQuotaExceededException":
		payload["resourceId"], payload["resourceType"] = e.resourceID, e.resourceType
		if e.code == "ServiceQuotaExceededException" {
			payload["serviceCode"], payload["quotaCode"] = rpsSigningName, "L-"+strings.ToUpper(e.resourceType)
		}
	}
	body, _ := json.Marshal(payload)
	w.Header().Set("Content-Type", rpsContentType)
	w.Header().Set("X-Amzn-ErrorType", e.code)
	w.WriteHeader(e.status)
	_, _ = w.Write(body)
}

func rpsValidation(format string, args ...any) *rpsErr {
	return &rpsErr{status: http.StatusBadRequest, code: "ValidationException", msg: fmt.Sprintf(format, args...)}
}

func rpsNotFound(kind, id string) *rpsErr {
	return &rpsErr{status: http.StatusNotFound, code: "ResourceNotFoundException", msg: kind + " " + id + " was not found.", resourceID: id, resourceType: kind}
}

func rpsConflict(kind, id, msg string) *rpsErr {
	return &rpsErr{status: http.StatusConflict, code: "ConflictException", msg: msg, resourceID: id, resourceType: kind}
}

func rpsQuota(kind string, max int) *rpsErr {
	return &rpsErr{status: http.StatusPaymentRequired, code: "ServiceQuotaExceededException", msg: fmt.Sprintf("Stackyard keeps at most %d %ss.", max, kind), resourceType: kind}
}
