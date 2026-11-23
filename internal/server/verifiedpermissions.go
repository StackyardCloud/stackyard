package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

// Amazon Verified Permissions: awsJson1_0, X-Amz-Target VerifiedPermissions.<Op>,
// signed "verifiedpermissions". Policies are parsed as Cedar when created and
// evaluated by a Cedar subset evaluator (verifiedpermissions_cedar_*.go).
// Requests are validated against a slim copy of the botocore model, and
// responses are conformed to it under the store lock.

const (
	vpTargetPrefix = "VerifiedPermissions."
	vpMaxBody      = 1 << 20
)

func (s *Server) handleVerifiedPermissionsJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}
	op, ok := strings.CutPrefix(strings.TrimSpace(r.Header.Get("X-Amz-Target")), vpTargetPrefix)
	if !ok {
		return false
	}
	if hint := strings.TrimSpace(sigV4ServiceHint(r)); hint != "" && hint != "verifiedpermissions" {
		return false
	}
	// A declared length bounds what net/http reads; refuse chunked bodies.
	// r.Body is deliberately not reassigned.
	if r.ContentLength < 0 || r.ContentLength > vpMaxBody {
		respondVerifiedPermissionsError(w, vpValidation("request body must declare a Content-Length of at most 1 MiB"))
		return true
	}
	if ok, status, code, msg, _ := s.validateSigV4WithService(r, "verifiedpermissions"); !ok {
		respondVerifiedPermissionsError(w, &vpErr{status: status, code: code, msg: msg})
		return true
	}
	model := awsmodels.VerifiedPermissionsModel()
	shapes, known := model.Operations[op]
	if !known || vpHandlers[op] == nil {
		respondVerifiedPermissionsError(w, &vpErr{status: http.StatusBadRequest, code: "UnknownOperationException", msg: "unknown operation " + op})
		return true
	}
	raw, err := readBodyBytes(r)
	if err != nil {
		respondVerifiedPermissionsError(w, vpValidation("unable to read request body"))
		return true
	}
	in := map[string]any{}
	if strings.TrimSpace(string(raw)) != "" {
		if err := json.Unmarshal(raw, &in); err != nil {
			respondVerifiedPermissionsError(w, &vpErr{status: http.StatusBadRequest, code: "SerializationException", msg: "invalid JSON body"})
			return true
		}
	}
	if problems := model.Validate(shapes.Input, in); len(problems) > 0 {
		respondVerifiedPermissionsError(w, vpValidation("%s", strings.Join(problems, "; ")))
		return true
	}
	// Only modelled members reach the store, so callers can't set internal state.
	in, _ = model.Conform(shapes.Input, in).(map[string]any)
	x := vpExtra{now: time.Now().UTC()}
	// Look the Cognito user pool up before taking this store's lock.
	if cfg := vpMap(in, "configuration"); cfg != nil || vpMap(in, "updateConfiguration") != nil {
		if cfg == nil {
			cfg = vpMap(in, "updateConfiguration")
		}
		if cog := vpMap(cfg, "cognitoUserPoolConfiguration"); cog != nil {
			_, pool := vpCognitoPoolID(vpStr(cog, "userPoolArn"))
			_, err := s.cognitouserpools.DescribeUserPool(pool)
			x.userPoolExists = err == nil
		}
	}
	out, verr := s.verifiedPermissions.handle(op, in, x)
	if verr != nil {
		respondVerifiedPermissionsError(w, verr)
		return true
	}
	body, _ := json.Marshal(out)
	w.Header().Set("Content-Type", "application/x-amz-json-1.0")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
	return true
}

func respondVerifiedPermissionsError(w http.ResponseWriter, e *vpErr) {
	payload := map[string]any{"__type": e.code, "message": e.msg}
	for k, v := range e.extra {
		payload[k] = v
	}
	body, _ := json.Marshal(payload)
	w.Header().Set("Content-Type", "application/x-amz-json-1.0")
	w.Header().Set("X-Amzn-ErrorType", e.code)
	w.WriteHeader(e.status)
	_, _ = w.Write(body)
}
