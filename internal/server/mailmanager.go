package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

// Amazon SES Mail Manager: X-Amz-Target MailManagerSvc.*, signed ses. The
// CLI 2.42.1 model lists smithy-rpc-v2-cbor first, but the CLI sends JSON 1.0.
// Archives are filled by SES (SESv2 and Pinpoint Email) sends through a
// configuration set whose archiving options name the archive (see
// mailmanager_archive.go).

const (
	mmgrTargetPrefix = "MailManagerSvc."
	mmgrMaxBody      = 1 << 20
	mmgrContentType  = "application/x-amz-json-1.0"
)

type mmgrErr struct{ code, msg string }

func mmgrError(code, format string, args ...any) *mmgrErr {
	return &mmgrErr{code: code, msg: fmt.Sprintf(format, args...)}
}

func mmgrInvalid(format string, args ...any) *mmgrErr {
	return mmgrError("ValidationException", format, args...)
}

var mmgrStatus = map[string]int{
	"ValidationException": http.StatusBadRequest, "ResourceNotFoundException": http.StatusNotFound,
	"ConflictException": http.StatusConflict, "ServiceQuotaExceededException": http.StatusPaymentRequired,
	"AccessDeniedException": http.StatusForbidden, "UnknownOperationException": http.StatusBadRequest,
	"SerializationException": http.StatusBadRequest,
}

// mmgrReq is one decoded request; base is the server's own URL, used for
// pre-signed import and download links.
type mmgrReq struct {
	in     map[string]any
	base   string
	caller s3Caller // authorizes the S3 writes an export makes for the caller
}

type mmgrHandler func(s *Server, q mmgrReq) (map[string]any, *mmgrErr)

func (s *Server) handleMailManagerJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if s.handleMailManagerLinks(w, r) {
		return true
	}
	if r.Method != http.MethodPost {
		return false
	}
	op, ok := strings.CutPrefix(strings.TrimSpace(r.Header.Get("X-Amz-Target")), mmgrTargetPrefix)
	if !ok {
		return false
	}
	// A declared length bounds what net/http reads; refuse chunked bodies.
	if r.ContentLength < 0 || r.ContentLength > mmgrMaxBody {
		respondMailManagerError(w, mmgrInvalid("request body must declare a Content-Length of at most 1 MiB"))
		return true
	}
	if ok, status, code, msg, _ := s.validateSigV4WithService(r, "ses"); !ok {
		respondMailManagerErrorStatus(w, status, code, msg)
		return true
	}
	h := mmgrHandlers[op]
	if !mailManagerOperationByName[op] || h == nil {
		respondMailManagerError(w, mmgrError("UnknownOperationException", "unknown operation %s", op))
		return true
	}
	model := awsmodels.MailManagerModel()
	raw, err := readBodyBytes(r)
	if err != nil {
		respondMailManagerError(w, mmgrError("SerializationException", "invalid JSON body"))
		return true
	}
	in, problem := awsModelDecode(raw, model, model.Operations[op].Input)
	if problem != "" {
		code := "ValidationException"
		if problem == awsModelBadJSON {
			code = "SerializationException"
		}
		respondMailManagerError(w, mmgrError(code, "%s", problem))
		return true
	}
	out, merr := h(s, mmgrReq{in: in, base: mmgrBaseURL(r), caller: s.s3CallerOf(r)})
	if merr != nil {
		respondMailManagerError(w, merr)
		return true
	}
	if out == nil {
		out = map[string]any{}
	}
	// out is built fresh under the store lock; nothing in it is shared.
	body, _ := json.Marshal(model.Conform(model.Operations[op].Output, out))
	w.Header().Set("Content-Type", mmgrContentType)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
	return true
}

func respondMailManagerError(w http.ResponseWriter, e *mmgrErr) {
	status := mmgrStatus[e.code]
	if status == 0 {
		status = http.StatusBadRequest
	}
	respondMailManagerErrorStatus(w, status, e.code, e.msg)
}

func respondMailManagerErrorStatus(w http.ResponseWriter, status int, code, msg string) {
	body, _ := json.Marshal(map[string]string{"__type": code, "Message": msg})
	w.Header().Set("Content-Type", mmgrContentType)
	w.Header().Set("X-Amzn-ErrorType", code)
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

func mmgrBaseURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return scheme + "://" + r.Host
}

// awsModelBadJSON is the problem awsModelDecode reports for a body that isn't
// a JSON object.
const awsModelBadJSON = "invalid JSON body"

// awsModelDecode parses a JSON request body, validates it against shape and
// conforms it (members the shape doesn't define are dropped). It returns a
// problem description instead of the input when either step fails. Shared by
// Mail Manager, Pinpoint Email and Pinpoint SMS Voice.
func awsModelDecode(raw []byte, model *awsmodels.Model, shape string) (map[string]any, string) {
	in := map[string]any{}
	if strings.TrimSpace(string(raw)) != "" {
		if err := json.Unmarshal(raw, &in); err != nil || in == nil {
			return nil, awsModelBadJSON
		}
	}
	if problems := model.Validate(shape, in); len(problems) > 0 {
		return nil, strings.Join(problems, "; ")
	}
	conformed, _ := model.Conform(shape, in).(map[string]any)
	if conformed == nil {
		conformed = map[string]any{}
	}
	return conformed, ""
}

// ---- request helpers ----

func mmgrStr(m map[string]any, k string) string { v, _ := m[k].(string); return v }

func mmgrMap(m map[string]any, k string) map[string]any { v, _ := m[k].(map[string]any); return v }

func mmgrList(m map[string]any, k string) []any { v, _ := m[k].([]any); return v }

// mmgrInt returns the integer m[k] (range-checked by the model), or def.
func mmgrInt(m map[string]any, k string, def int) int {
	if v, ok := m[k].(float64); ok && v >= -1e9 && v <= 1e9 {
		return int(v)
	}
	return def
}

func mmgrStrings(v any) []string {
	list, _ := v.([]any)
	out := make([]string, 0, len(list))
	for _, item := range list {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

// mmgrTime parses a JSON timestamp: epoch seconds or an RFC 3339 string.
func mmgrTime(v any) (time.Time, bool) {
	switch t := v.(type) {
	case float64:
		if t < 0 || t > 1e11 {
			return time.Time{}, false
		}
		sec := int64(t)
		return time.Unix(sec, int64((t-float64(sec))*1e9)).UTC(), true
	case string:
		if parsed, err := time.Parse(time.RFC3339Nano, t); err == nil {
			return parsed.UTC(), true
		}
	}
	return time.Time{}, false
}

func mmgrEpoch(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return float64(t.UnixMilli()) / 1000
}

// mmgrPage slices n items by an offset token.
func mmgrPage(n int, token string, size, def int) (int, int, string, bool) {
	if size <= 0 {
		size = def
	}
	start := 0
	if token != "" {
		if _, err := fmt.Sscanf(token, "%d", &start); err != nil || start < 0 || start > n || fmt.Sprint(start) != token {
			return 0, 0, "", false
		}
	}
	end := min(start+size, n)
	next := ""
	if end < n {
		next = fmt.Sprint(end)
	}
	return start, end, next, true
}
