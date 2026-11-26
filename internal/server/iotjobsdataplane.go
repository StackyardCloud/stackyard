package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

// AWS IoT Jobs Data Plane: REST JSON, signed iot-jobs-data. Its paths
// (/things/{thingName}/jobs/...) are the IoT control plane's too, so it is
// claimed only by signing name (or, unsigned, by its data.jobs.iot.* host)
// ahead of the IoT router. Executions belong to jobs and commands created
// through the IoT control plane (iot_registry.go); state lives in
// iotjobsdataplane_store.go.

const iotjMaxBody = 64 << 10

type iotjErr struct {
	status    int
	code, msg string
}

var iotjErrStatus = map[string]int{
	"ResourceNotFoundException": http.StatusNotFound, "InvalidStateTransitionException": http.StatusConflict,
	"ConflictException": http.StatusConflict, "TerminalStateException": http.StatusGone,
	"ServiceQuotaExceededException": http.StatusPaymentRequired, "ThrottlingException": http.StatusTooManyRequests,
	"InternalServerException": http.StatusInternalServerError,
}

func iotjErrorf(code, format string, args ...any) *iotjErr {
	status := iotjErrStatus[code]
	if status == 0 {
		status = http.StatusBadRequest
	}
	return &iotjErr{status: status, code: code, msg: fmt.Sprintf(format, args...)}
}

func iotjInvalid(format string, args ...any) *iotjErr {
	return iotjErrorf("InvalidRequestException", format, args...)
}

func isIoTJobsDataPlaneCandidate(r *http.Request) bool {
	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service == "iot-jobs-data" {
		return true
	}
	return service == "" && strings.HasPrefix(strings.ToLower(strings.TrimSpace(r.Host)), "data.jobs.iot.")
}

func (s *Server) handleIoTJobsDataPlaneRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isIoTJobsDataPlaneCandidate(r) {
		return false
	}
	op, labels, ok := iotdMatchRoute(r, iotjRoutes)
	if !ok {
		iotjRespondError(w, iotjInvalid("no IoT Jobs data plane operation matches %s %s", r.Method, rawRequestPath(r)))
		return true
	}
	// A declared length bounds what net/http reads; refuse chunked bodies.
	if r.ContentLength < 0 || r.ContentLength > iotjMaxBody {
		iotjRespondError(w, iotjInvalid("request body must declare a Content-Length of at most %d bytes", iotjMaxBody))
		return true
	}
	if ok, status, code, msg, _ := s.validateSigV4WithService(r, "iot-jobs-data"); !ok {
		iotjRespondError(w, &iotjErr{status: status, code: code, msg: msg})
		return true
	}
	in, err := iotjDecode(r, op, labels)
	if err != nil {
		iotjRespondError(w, err)
		return true
	}
	var out map[string]any
	switch op {
	case "GetPendingJobExecutions":
		out, err = s.iotjGetPending(in)
	case "StartNextPendingJobExecution":
		out, err = s.iotjStartNext(in)
	case "DescribeJobExecution":
		out, err = s.iotjDescribe(in)
	case "UpdateJobExecution":
		out, err = s.iotjUpdate(in)
	case "StartCommandExecution":
		out, err = s.iotjStartCommandExecution(in)
	}
	if err != nil {
		iotjRespondError(w, err)
		return true
	}
	m := awsmodels.IoTJobsDataPlaneModel()
	body, _ := json.Marshal(m.Conform(m.Operations[op].Output, out))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
	return true
}

var iotjRoutes = func() []iotdRoute {
	out := make([]iotdRoute, 0, len(iotJobsDataPlaneOperations))
	for _, op := range iotJobsDataPlaneOperations {
		out = append(out, iotdRoute{name: op.Name, method: op.Method, segments: splitPathSegments(op.URI)})
	}
	return out
}()

// iotjDecode validates the JSON body against the operation's input shape
// (unknown members are refused) and adds the URI labels and query values.
func iotjDecode(r *http.Request, op string, labels map[string]string) (map[string]any, *iotjErr) {
	m := awsmodels.IoTJobsDataPlaneModel()
	shapeName := m.Operations[op].Input
	in := map[string]any{}
	if r.ContentLength > 0 {
		raw, err := readBodyBytes(r)
		if err != nil {
			return nil, iotjInvalid("unable to read request body")
		}
		if len(bytes.TrimSpace(raw)) > 0 {
			if err := json.Unmarshal(raw, &in); err != nil {
				return nil, iotjInvalid("request body is not a JSON object")
			}
		}
	}
	if problems := m.Validate(shapeName, in); len(problems) > 0 {
		code := "InvalidRequestException"
		if op == "StartCommandExecution" {
			code = "ValidationException"
		}
		return nil, iotjErrorf(code, "%s", strings.Join(problems, "; "))
	}
	for name, v := range labels {
		in[name] = v
	}
	if t, ok := in["thingName"].(string); ok && !iotdThingNameRe.MatchString(t) {
		return nil, iotjInvalid("thingName must be 1-128 characters of [a-zA-Z0-9:_-]")
	}
	if j, ok := in["jobId"].(string); ok && j != "$next" && !iotjJobIDRe.MatchString(j) {
		return nil, iotjInvalid("jobId must be 1-64 characters of [a-zA-Z0-9_-]")
	}
	q := r.URL.Query()
	if v := q.Get("includeJobDocument"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return nil, iotjInvalid("includeJobDocument must be a boolean")
		}
		in["includeJobDocument"] = b
	}
	if v := q.Get("executionNumber"); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return nil, iotjInvalid("executionNumber must be an integer")
		}
		in["executionNumber"] = float64(n)
	}
	return in, nil
}

func iotjRespondError(w http.ResponseWriter, e *iotjErr) {
	body, _ := json.Marshal(map[string]string{"message": e.msg})
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Amzn-ErrorType", e.code)
	w.WriteHeader(e.status)
	_, _ = w.Write(body)
}
