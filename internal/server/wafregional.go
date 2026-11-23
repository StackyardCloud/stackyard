package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

// AWS WAF Classic Regional: awsJson1_1, X-Amz-Target AWSWAF_Regional_20161128.*,
// signed waf-regional. Every create, update and delete of a WAF entity needs an
// unused change token from GetChangeToken.

const (
	wrTargetPrefix = "AWSWAF_Regional_20161128."
	wrMaxBody      = 2 << 20 // an IPSet update can carry thousands of descriptors
)

type wrErr struct {
	code, msg string
	extra     map[string]any
}

func wrError(code, format string, args ...any) *wrErr {
	return &wrErr{code: code, msg: fmt.Sprintf(format, args...)}
}

// wrParam is WAFInvalidParameterException with the modelled field/parameter/reason.
func wrParam(field, parameter, reason, format string, args ...any) *wrErr {
	e := wrError("WAFInvalidParameterException", format, args...)
	e.extra = map[string]any{"field": field, "parameter": parameter, "reason": reason}
	return e
}

func wrMissing(what, id string) *wrErr {
	return wrError("WAFNonexistentItemException", "The referenced %s %s doesn't exist.", what, id)
}

// wrHandler runs with the store lock held; its output is conformed to the
// operation's output shape before the lock is released.
type wrHandler func(st *wafRegionalStore, in map[string]any) (map[string]any, *wrErr)

func (s *Server) handleWAFRegionalJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}
	op, ok := strings.CutPrefix(strings.TrimSpace(r.Header.Get("X-Amz-Target")), wrTargetPrefix)
	if !ok {
		return false
	}
	// A declared length bounds what net/http reads; refuse chunked bodies.
	if r.ContentLength < 0 || r.ContentLength > wrMaxBody {
		respondWAFRegionalError(w, wrError("WAFBadRequestException", "request body must declare a Content-Length of at most 2 MiB"))
		return true
	}
	if ok, status, code, msg, _ := s.validateSigV4WithService(r, "waf-regional"); !ok {
		writeWAFRegionalJSON(w, status, map[string]any{"__type": code, "message": msg}, code)
		return true
	}
	h := wafRegionalHandlers[op]
	if _, known := wafRegionalOperationByName[op]; !known || (h == nil && op != "CreateWebACLMigrationStack") {
		respondWAFRegionalError(w, wrError("UnknownOperationException", "unknown operation %s", op))
		return true
	}
	raw, err := readBodyBytes(r)
	in := map[string]any{}
	if err == nil && strings.TrimSpace(string(raw)) != "" {
		err = json.Unmarshal(raw, &in)
	}
	if err != nil {
		respondWAFRegionalError(w, wrError("SerializationException", "invalid JSON body"))
		return true
	}
	model := awsmodels.WAFRegionalModel()
	shapes := model.Operations[op]
	if problems := model.Validate(shapes.Input, in); len(problems) > 0 {
		respondWAFRegionalError(w, wrError("WAFInvalidParameterException", "%s", strings.Join(problems, "; ")))
		return true
	}
	// Conform drops members the input shape doesn't define, so callers can't
	// reach the store's own fields.
	conformed, _ := model.Conform(shapes.Input, in).(map[string]any)
	var out any
	var werr *wrErr
	if op == "CreateWebACLMigrationStack" {
		out, werr = s.wrMigrationStack(conformed)
	} else {
		out, werr = s.wafRegional.run(h, conformed, shapes.Output)
	}
	if werr != nil {
		respondWAFRegionalError(w, werr)
		return true
	}
	writeWAFRegionalJSON(w, http.StatusOK, out, "")
	return true
}

// run executes h under the store lock and conforms its output there, so the
// response never shares maps with stored state.
func (st *wafRegionalStore) run(h wrHandler, in map[string]any, output string) (any, *wrErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	out, err := h(st, in)
	if err != nil {
		return nil, err
	}
	if out == nil {
		out = map[string]any{}
	}
	return awsmodels.WAFRegionalModel().Conform(output, out), nil
}

func respondWAFRegionalError(w http.ResponseWriter, e *wrErr) {
	body := map[string]any{"__type": e.code, "message": e.msg}
	for k, v := range e.extra {
		body[k] = v
	}
	writeWAFRegionalJSON(w, http.StatusBadRequest, body, e.code)
}

func writeWAFRegionalJSON(w http.ResponseWriter, status int, v any, errorType string) {
	body, _ := json.Marshal(v)
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	if errorType != "" {
		w.Header().Set("X-Amzn-ErrorType", errorType)
	}
	w.WriteHeader(status)
	_, _ = w.Write(body)
}
