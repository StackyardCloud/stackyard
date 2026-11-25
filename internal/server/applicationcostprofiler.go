package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

// AWS Application Cost Profiler: REST JSON, signed application-cost-profiler.
// Report definitions name a destination in Stackyard's S3; each
// ImportApplicationUsage reads a usage file from S3 and writes every
// definition's tenant-share report next to its destination prefix. State
// lives in applicationcostprofiler_store.go.

// ponytail: 16 KiB is generous; the largest request is a report definition
// (description 1 KiB, bucket and prefix 575 characters).
const acpMaxBody = 16 << 10

// acpErrorf builds an error with the status the model gives its shape.
func acpErrorf(code, format string, args ...any) *budErr {
	status := map[string]int{
		"AccessDeniedException": http.StatusForbidden, "ServiceQuotaExceededException": http.StatusPaymentRequired,
		"ThrottlingException": http.StatusTooManyRequests, "InternalServerException": http.StatusInternalServerError,
	}[code]
	if status == 0 {
		status = http.StatusBadRequest
	}
	return &budErr{status: status, code: code, msg: fmt.Sprintf(format, args...)}
}

func acpInvalid(format string, args ...any) *budErr {
	return acpErrorf("ValidationException", format, args...)
}

type acpRoute struct {
	op       acpOperation
	segments []string
}

var acpRoutes = func() []acpRoute {
	out := make([]acpRoute, 0, len(acpOperations))
	for _, op := range acpOperations {
		out = append(out, acpRoute{op: op, segments: splitPathSegments(op.URI)})
	}
	return out
}()

func acpMatch(method string, path []string) (string, map[string]string, bool) {
	for _, rt := range acpRoutes {
		labels := map[string]string{}
		if rt.op.Method == method && nmMatchSegments(rt.segments, path, labels, false) {
			return rt.op.Name, labels, true
		}
	}
	return "", nil, false
}

// isACPRESTCandidate claims requests signed for application-cost-profiler,
// or unsigned requests sent to its host.
func isACPRESTCandidate(r *http.Request) bool {
	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service == "application-cost-profiler" {
		return true
	}
	return service == "" && strings.HasPrefix(strings.ToLower(strings.TrimSpace(r.Host)), "application-cost-profiler.")
}

func (s *Server) handleApplicationCostProfilerRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isACPRESTCandidate(r) {
		return false
	}
	raw := splitPathSegments(rawRequestPath(r))
	if len(raw) > nmMaxPathSegments {
		acpRespondError(w, acpErrorf("UnknownOperationException", "no Application Cost Profiler operation matches this path"))
		return true
	}
	path := make([]string, len(raw))
	for i, v := range raw {
		u, err := url.PathUnescape(v)
		if err != nil {
			acpRespondError(w, acpInvalid("invalid path segment"))
			return true
		}
		path[i] = u
	}
	op, labels, ok := acpMatch(r.Method, path)
	if !ok {
		acpRespondError(w, acpErrorf("UnknownOperationException", "no Application Cost Profiler operation matches %s %s", r.Method, rawRequestPath(r)))
		return true
	}
	// A declared length bounds what net/http reads; refuse chunked bodies.
	if r.ContentLength < 0 || r.ContentLength > acpMaxBody {
		acpRespondError(w, acpInvalid("request body must declare a Content-Length of at most %d bytes", acpMaxBody))
		return true
	}
	if ok, status, code, msg, _ := s.validateSigV4WithService(r, "application-cost-profiler"); !ok {
		acpRespondError(w, &budErr{status: status, code: code, msg: msg})
		return true
	}
	in, err := acpDecode(r, op, labels)
	if err != nil {
		acpRespondError(w, err)
		return true
	}
	out, err := s.acpDispatch(op, in)
	if err != nil {
		acpRespondError(w, err)
		return true
	}
	m := awsmodels.ApplicationCostProfilerModel()
	body, _ := json.Marshal(m.Conform(m.Operations[op].Output, out))
	acpRespond(w, body)
	return true
}

// acpDecode builds the operation input from the JSON body (validated and
// conformed against the input shape) plus URI labels and query parameters.
func acpDecode(r *http.Request, op string, labels map[string]string) (map[string]any, *budErr) {
	m := awsmodels.ApplicationCostProfilerModel()
	shapeName := m.Operations[op].Input
	in := map[string]any{}
	if r.ContentLength > 0 {
		raw, err := readBodyBytes(r)
		if err != nil {
			return nil, acpInvalid("unable to read request body")
		}
		if len(bytes.TrimSpace(raw)) > 0 {
			if err := json.Unmarshal(raw, &in); err != nil || in == nil {
				return nil, acpInvalid("request body is not a JSON object")
			}
		}
	}
	if problems := m.Validate(shapeName, in); len(problems) > 0 {
		return nil, acpInvalid("%s", strings.Join(problems, "; "))
	}
	in, _ = m.Conform(shapeName, in).(map[string]any)
	if in == nil {
		in = map[string]any{}
	}
	shape := m.Shapes[shapeName]
	q := r.URL.Query()
	for member, ref := range shape.Members {
		var value any
		switch ref.Location {
		case "uri":
			v, ok := labels[ref.LocationName]
			if !ok {
				continue
			}
			value = v
		case "querystring":
			vals, ok := q[ref.LocationName]
			if !ok {
				continue
			}
			value = vals
		default:
			continue
		}
		conv, err := nmConvert(m, ref.Shape, value)
		if err != nil {
			return nil, acpInvalid("%s %s", member, err.Error())
		}
		if problems := m.Validate(ref.Shape, conv); len(problems) > 0 {
			_, reason, _ := strings.Cut(problems[0], ": ")
			return nil, acpInvalid("%s %s", member, reason)
		}
		in[member] = conv
	}
	for _, req := range shape.Required {
		if _, ok := in[req]; !ok {
			return nil, acpInvalid("missing required member %s", req)
		}
	}
	return in, nil
}

func acpRespond(w http.ResponseWriter, body []byte) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}

// acpRespondError writes a REST JSON error: the code travels in
// X-Amzn-ErrorType, the body carries the shape's lowercase message.
func acpRespondError(w http.ResponseWriter, e *budErr) {
	body, _ := json.Marshal(map[string]string{"message": e.msg})
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Amzn-ErrorType", e.code)
	w.WriteHeader(e.status)
	_, _ = w.Write(body)
}
