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

// AWS IoT Core Device Advisor: REST JSON, signed iotdeviceadvisor. Suite
// definitions are versioned; suite runs derive their test cases from the
// definition's root group and advance PENDING -> RUNNING -> PASS (or
// STOPPING -> STOPPED) one step per read. State lives in
// iotdeviceadvisor_store.go.

// ponytail: 64 KiB covers any real suite definition (rootGroup is at most
// 2048 characters by the model).
const idaMaxBody = 64 << 10

type idaErr struct {
	status    int
	code, msg string
}

func idaErrorf(code, format string, args ...any) *idaErr {
	status := map[string]int{
		"ResourceNotFoundException": http.StatusNotFound, "ConflictException": http.StatusConflict,
		"InternalServerException": http.StatusInternalServerError,
	}[code]
	if status == 0 {
		status = http.StatusBadRequest
	}
	return &idaErr{status: status, code: code, msg: fmt.Sprintf(format, args...)}
}

func idaInvalid(format string, args ...any) *idaErr {
	return idaErrorf("ValidationException", format, args...)
}

type idaRoute struct {
	op       iotDeviceAdvisorOperation
	segments []string
}

var idaRoutes = func() []idaRoute {
	out := make([]idaRoute, 0, len(iotDeviceAdvisorOperations))
	for _, op := range iotDeviceAdvisorOperations {
		out = append(out, idaRoute{op: op, segments: splitPathSegments(op.URI)})
	}
	return out
}()

// idaMatch finds the operation for a method and decoded path; a second pass
// lets a label (an ARN sent with raw slashes) absorb several segments.
func idaMatch(method string, path []string) (string, map[string]string, bool) {
	for _, greedy := range []bool{false, true} {
		for _, rt := range idaRoutes {
			if rt.op.Method != method {
				continue
			}
			labels := map[string]string{}
			if nmMatchSegments(rt.segments, path, labels, greedy) {
				return rt.op.Name, labels, true
			}
		}
	}
	return "", nil, false
}

func isIoTDeviceAdvisorRESTCandidate(r *http.Request) bool {
	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service == "iotdeviceadvisor" {
		return true
	}
	return service == "" && strings.HasPrefix(strings.ToLower(strings.TrimSpace(r.Host)), "api.iotdeviceadvisor.")
}

func (s *Server) handleIoTDeviceAdvisorRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isIoTDeviceAdvisorRESTCandidate(r) {
		return false
	}
	raw := splitPathSegments(rawRequestPath(r))
	if len(raw) == 0 || len(raw) > nmMaxPathSegments {
		respondIDAError(w, idaErrorf("UnknownOperationException", "no Device Advisor operation matches %s %s", r.Method, rawRequestPath(r)))
		return true
	}
	path := make([]string, len(raw))
	for i, v := range raw {
		u, err := url.PathUnescape(v)
		if err != nil {
			respondIDAError(w, idaInvalid("invalid path segment"))
			return true
		}
		path[i] = u
	}
	op, labels, ok := idaMatch(r.Method, path)
	if !ok {
		respondIDAError(w, idaErrorf("UnknownOperationException", "no Device Advisor operation matches %s %s", r.Method, rawRequestPath(r)))
		return true
	}
	// A declared length bounds what net/http reads; refuse chunked bodies.
	if r.ContentLength < 0 || r.ContentLength > idaMaxBody {
		respondIDAError(w, idaInvalid("request body must declare a Content-Length of at most %d bytes", idaMaxBody))
		return true
	}
	if ok, status, code, msg, _ := s.validateSigV4WithService(r, "iotdeviceadvisor"); !ok {
		respondIDAError(w, &idaErr{status: status, code: code, msg: msg})
		return true
	}
	in, err := idaDecode(r, op, labels)
	if err != nil {
		respondIDAError(w, err)
		return true
	}
	out, err := s.iotDeviceAdvisor.dispatch(op, in)
	if err != nil {
		respondIDAError(w, err)
		return true
	}
	m := awsmodels.IoTDeviceAdvisorModel()
	body, _ := json.Marshal(m.Conform(m.Operations[op].Output, out))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
	return true
}

// idaDecode builds the operation input from the JSON body (validated against
// the input shape, which refuses unknown members) plus URI labels and query
// parameters converted to their member shapes.
func idaDecode(r *http.Request, op string, labels map[string]string) (map[string]any, *idaErr) {
	m := awsmodels.IoTDeviceAdvisorModel()
	shapeName := m.Operations[op].Input
	in := map[string]any{}
	if r.ContentLength > 0 {
		raw, err := readBodyBytes(r)
		if err != nil {
			return nil, idaInvalid("unable to read request body")
		}
		if len(bytes.TrimSpace(raw)) > 0 {
			if err := json.Unmarshal(raw, &in); err != nil {
				return nil, idaInvalid("request body is not a JSON object")
			}
		}
	}
	if problems := m.Validate(shapeName, in); len(problems) > 0 {
		return nil, idaInvalid("%s", strings.Join(problems, "; "))
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
			return nil, idaInvalid("%s %s", member, err.Error())
		}
		if problems := m.Validate(ref.Shape, conv); len(problems) > 0 {
			_, reason, _ := strings.Cut(problems[0], ": ")
			return nil, idaInvalid("%s %s", member, reason)
		}
		in[member] = conv
	}
	for _, req := range shape.Required {
		if _, ok := in[req]; !ok {
			return nil, idaInvalid("missing required member %s", req)
		}
	}
	return in, nil
}

func respondIDAError(w http.ResponseWriter, e *idaErr) {
	body, _ := json.Marshal(map[string]string{"message": e.msg})
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Amzn-ErrorType", e.code)
	w.WriteHeader(e.status)
	_, _ = w.Write(body)
}
