package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

// Amazon CodeGuru Security: REST JSON, signed codeguru-security. Only
// requests signed for codeguru-security (or unsigned to a codeguru-security.*
// host) are claimed, so CodeGuru Reviewer and Profiler keep /tags, /metrics
// and the rest of their paths. Code is uploaded to a URL on this server
// (codegurusecurity_scan.go) and scanned by a small built-in rule set.

const (
	cgsMaxBody = 64 << 10
	cgsAccount = "123456789012"
	cgsRegion  = "us-east-1"
)

type cgsErr struct {
	status                   int
	code, msg, reason, resID string
}

func cgsValidation(format string, args ...any) *cgsErr {
	return &cgsErr{status: http.StatusBadRequest, code: "ValidationException", msg: fmt.Sprintf(format, args...), reason: "fieldValidationFailed"}
}

func cgsNotFound(id, format string, args ...any) *cgsErr {
	return &cgsErr{status: http.StatusNotFound, code: "ResourceNotFoundException", msg: fmt.Sprintf(format, args...), resID: id}
}

func cgsConflict(id, format string, args ...any) *cgsErr {
	return &cgsErr{status: http.StatusConflict, code: "ConflictException", msg: fmt.Sprintf(format, args...), resID: id}
}

// cgsRequest is one decoded call: the model-checked input (body, URI and
// query members) and this server's scheme://host.
type cgsRequest struct {
	in   map[string]any
	base string
}

type cgsHandler func(s *Server, req cgsRequest) (map[string]any, *cgsErr)

var cgsHandlers = map[string]cgsHandler{
	"BatchGetFindings":           (*Server).cgsBatchGetFindings,
	"CreateScan":                 (*Server).cgsCreateScan,
	"CreateUploadUrl":            (*Server).cgsCreateUploadURL,
	"GetAccountConfiguration":    (*Server).cgsGetAccountConfiguration,
	"GetFindings":                (*Server).cgsGetFindings,
	"GetMetricsSummary":          (*Server).cgsGetMetricsSummary,
	"GetScan":                    (*Server).cgsGetScan,
	"ListFindingsMetrics":        (*Server).cgsListFindingsMetrics,
	"ListScans":                  (*Server).cgsListScans,
	"ListTagsForResource":        (*Server).cgsListTags,
	"TagResource":                (*Server).cgsTagResource,
	"UntagResource":              (*Server).cgsUntagResource,
	"UpdateAccountConfiguration": (*Server).cgsUpdateAccountConfiguration,
}

func cgsRoute(method, path string) (string, map[string]string) {
	for _, op := range cgsOperations {
		if op.Method != method {
			continue
		}
		if params, ok := connectPathParams(op.URI, path); ok {
			return op.Name, params
		}
	}
	return "", nil
}

func (s *Server) handleCodeGuruSecurityRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	hint := sigV4ServiceHint(r)
	if hint != "codeguru-security" && !(hint == "" && strings.HasPrefix(strings.ToLower(r.Host), "codeguru-security.")) {
		return false
	}
	if r.ContentLength < 0 || r.ContentLength > cgsMaxBody {
		respondCgsError(w, cgsValidation("request body must declare a Content-Length of at most %d bytes", cgsMaxBody))
		return true
	}
	if ok, status, code, msg, _ := s.validateSigV4WithService(r, "codeguru-security"); !ok {
		respondCgsError(w, &cgsErr{status: status, code: code, msg: msg})
		return true
	}
	op, params := cgsRoute(r.Method, rawRequestPath(r))
	if op == "" {
		e := cgsValidation("unknown operation %s %s", r.Method, r.URL.Path)
		e.reason = "unknownOperation"
		respondCgsError(w, e)
		return true
	}
	raw, err := io.ReadAll(io.LimitReader(r.Body, cgsMaxBody))
	in := map[string]any{}
	if err == nil && strings.TrimSpace(string(raw)) != "" {
		err = json.Unmarshal(raw, &in)
	}
	if err != nil {
		e := cgsValidation("invalid JSON body")
		e.reason = "cannotParse"
		respondCgsError(w, e)
		return true
	}
	model := awsmodels.CodeGuruSecurityModel()
	shapes := model.Operations[op]
	if problems := model.Validate(shapes.Input, in); len(problems) > 0 {
		respondCgsError(w, cgsValidation("%s", strings.Join(problems, "; ")))
		return true
	}
	conformed, _ := model.Conform(shapes.Input, in).(map[string]any)
	if conformed == nil {
		conformed = map[string]any{}
	}
	// URI and query members are bound after Conform so a body can't set them.
	if cerr := cgsBindLocated(r, model, shapes.Input, params, conformed); cerr != nil {
		respondCgsError(w, cerr)
		return true
	}
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	out, cerr := cgsHandlers[op](s, cgsRequest{in: conformed, base: scheme + "://" + r.Host})
	if cerr != nil {
		respondCgsError(w, cerr)
		return true
	}
	if out["nextToken"] == "" {
		delete(out, "nextToken")
	}
	// Handlers build their output under the store lock.
	body, _ := json.Marshal(model.Conform(shapes.Output, out))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
	return true
}

// cgsBindLocated binds the URI and query-string members of the input shape,
// each decoded to its JSON type and checked against its model shape.
func cgsBindLocated(r *http.Request, model *awsmodels.Model, input string, params map[string]string, in map[string]any) *cgsErr {
	shape := model.Shapes[input]
	if shape == nil {
		return nil
	}
	required := map[string]bool{}
	for _, name := range shape.Required {
		required[name] = true
	}
	q := r.URL.Query()
	for name, ref := range shape.Members {
		var values []string
		switch ref.Location {
		case "uri":
			if v, ok := params[ref.LocationName]; ok {
				values = []string{v}
			}
		case "querystring":
			values = q[ref.LocationName]
		default:
			continue
		}
		if len(values) == 0 {
			if required[name] {
				return cgsValidation("missing required parameter %s", name)
			}
			continue
		}
		member := model.Shapes[ref.Shape]
		if member == nil {
			continue
		}
		var v any = values[0]
		switch member.Type {
		case "list":
			if len(values) > 200 {
				return cgsValidation("%s takes at most 200 values", name)
			}
			v = c9AnyStrings(values)
		case "integer", "long":
			n, err := strconv.ParseInt(values[0], 10, 32)
			if err != nil {
				return cgsValidation("%s must be an integer", name)
			}
			v = float64(n)
		case "timestamp":
			t, err := cgsParseTime(values[0])
			if err != nil {
				return cgsValidation("%s must be an ISO 8601 timestamp or epoch seconds", name)
			}
			v = c9Epoch(t)
		}
		if problems := model.Validate(ref.Shape, v); len(problems) > 0 {
			return cgsValidation("%s: %s", name, strings.Join(problems, "; "))
		}
		in[name] = v
	}
	return nil
}

func cgsParseTime(v string) (time.Time, error) {
	if f, err := strconv.ParseFloat(v, 64); err == nil {
		sec := int64(f)
		return time.Unix(sec, 0).UTC(), nil
	}
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02T15:04:05", "2006-01-02"} {
		if t, err := time.Parse(layout, v); err == nil {
			return t.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid timestamp %q", v)
}

func respondCgsError(w http.ResponseWriter, e *cgsErr) {
	payload := map[string]string{"message": e.msg, "errorCode": e.code}
	switch e.code {
	case "ValidationException":
		payload["reason"] = e.reason
	case "ResourceNotFoundException", "ConflictException":
		payload["resourceId"], payload["resourceType"] = e.resID, "scan"
	}
	body, _ := json.Marshal(payload)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Amzn-ErrorType", e.code)
	w.WriteHeader(e.status)
	_, _ = w.Write(body)
}

// cgsTime reads a timestamp member bound by cgsBindLocated (epoch seconds).
func cgsTime(in map[string]any, key string) time.Time {
	v, _ := in[key].(float64)
	return time.Unix(int64(v), 0).UTC()
}
