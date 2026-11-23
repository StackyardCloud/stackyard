package server

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

// Amazon OpenSearch Serverless control plane: awsJson1_0, X-Amz-Target
// OpenSearchServerless.<Op>, signed "aoss". Requests are validated against a
// slim copy of the botocore model and responses are conformed to it under the
// store lock. The data plane (the OpenSearch REST API served on each
// collection endpoint) is not emulated.

const (
	aossTargetPrefix = "OpenSearchServerless."
	aossMaxBody      = 1 << 20
)

var aossKnownOps = func() map[string]bool {
	out := map[string]bool{}
	for _, op := range openSearchServerlessOperations {
		out[op.Name] = true
	}
	return out
}()

func (s *Server) handleOpenSearchServerlessJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}
	op, ok := strings.CutPrefix(strings.TrimSpace(r.Header.Get("X-Amz-Target")), aossTargetPrefix)
	if !ok {
		return false
	}
	if hint := strings.TrimSpace(sigV4ServiceHint(r)); hint != "" && hint != "aoss" {
		return false
	}
	// A declared length bounds what net/http reads; refuse chunked bodies.
	// r.Body is deliberately not reassigned.
	if r.ContentLength < 0 || r.ContentLength > aossMaxBody {
		respondOpenSearchServerlessError(w, aossValidation("request body must declare a Content-Length of at most 1 MiB"))
		return true
	}
	if ok, status, code, msg, _ := s.validateSigV4WithService(r, "aoss"); !ok {
		respondOpenSearchServerlessError(w, &aossErr{status: status, code: code, msg: msg})
		return true
	}
	if !aossKnownOps[op] {
		respondOpenSearchServerlessError(w, &aossErr{status: http.StatusBadRequest, code: "UnknownOperationException", msg: "unknown operation " + op})
		return true
	}
	raw, err := readBodyBytes(r)
	if err != nil {
		respondOpenSearchServerlessError(w, aossValidation("unable to read request body"))
		return true
	}
	in := map[string]any{}
	if strings.TrimSpace(string(raw)) != "" {
		if err := json.Unmarshal(raw, &in); err != nil {
			respondOpenSearchServerlessError(w, &aossErr{status: http.StatusBadRequest, code: "SerializationException", msg: "invalid JSON body"})
			return true
		}
	}
	model := awsmodels.OpenSearchServerlessModel()
	shapes := model.Operations[op]
	if problems := model.Validate(shapes.Input, in); len(problems) > 0 {
		respondOpenSearchServerlessError(w, aossValidation("%s", strings.Join(problems, "; ")))
		return true
	}
	// Only modelled members reach the store, so callers can't set internal state.
	in, _ = model.Conform(shapes.Input, in).(map[string]any)
	out, aerr := s.openSearchServerless.handle(op, shapes.Output, in, "http://"+r.Host)
	if aerr != nil {
		respondOpenSearchServerlessError(w, aerr)
		return true
	}
	body, _ := json.Marshal(out)
	w.Header().Set("Content-Type", "application/x-amz-json-1.0")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
	return true
}

func respondOpenSearchServerlessError(w http.ResponseWriter, e *aossErr) {
	body, _ := json.Marshal(map[string]string{"__type": e.code, "message": e.msg})
	w.Header().Set("Content-Type", "application/x-amz-json-1.0")
	w.Header().Set("X-Amzn-ErrorType", e.code)
	w.WriteHeader(e.status)
	_, _ = w.Write(body)
}
