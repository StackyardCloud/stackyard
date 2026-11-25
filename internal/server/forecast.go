package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

// Amazon Forecast: awsJson1_1, X-Amz-Target AmazonForecast.*, signed forecast.
// Forecast Query (forecastquery.go) shares the signing name but uses the
// AmazonForecastRuntime target prefix.

const (
	fcTargetPrefix = "AmazonForecast."
	fcMaxBody      = 1 << 20
)

type fcErr struct{ code, msg string }

func fcError(code, format string, args ...any) *fcErr {
	return &fcErr{code: code, msg: fmt.Sprintf(format, args...)}
}

func fcInvalid(format string, args ...any) *fcErr {
	return fcError("InvalidInputException", format, args...)
}

func fcNotFound(format string, args ...any) *fcErr {
	return fcError("ResourceNotFoundException", format, args...)
}

func fcInUse(format string, args ...any) *fcErr {
	return fcError("ResourceInUseException", format, args...)
}

func (s *Server) handleForecastJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}
	op, ok := strings.CutPrefix(strings.TrimSpace(r.Header.Get("X-Amz-Target")), fcTargetPrefix)
	if !ok {
		return false
	}
	model := awsmodels.ForecastModel()
	in, ferr := fcDecode(s, w, r, "forecast", model, op)
	if ferr != nil {
		respondForecastError(w, ferr)
		return true
	}
	if in == nil { // signature failure already written
		return true
	}
	out, ferr := s.forecast.run(op, in)
	if ferr != nil {
		respondForecastError(w, ferr)
		return true
	}
	writeForecastJSON(w, http.StatusOK, out, "")
	return true
}

// fcDecode checks the body length and signature, then validates the body
// against the operation's input shape and conforms it (unknown members are
// dropped, so callers never reach stored bookkeeping fields). A nil map with
// a nil error means the signature failure has been written.
func fcDecode(s *Server, w http.ResponseWriter, r *http.Request, service string, model *awsmodels.Model, op string) (map[string]any, *fcErr) {
	// A declared length bounds what is read; refuse chunked bodies.
	if r.ContentLength < 0 || r.ContentLength > fcMaxBody {
		return nil, fcInvalid("request body must declare a Content-Length of at most 1 MiB")
	}
	if ok, status, code, msg, _ := s.validateSigV4WithService(r, service); !ok {
		writeForecastJSON(w, status, map[string]any{"__type": code, "message": msg}, code)
		return nil, nil
	}
	shapes, known := model.Operations[op]
	if !known {
		return nil, fcError("UnknownOperationException", "unknown operation %s", op)
	}
	raw, err := io.ReadAll(io.LimitReader(r.Body, fcMaxBody+1))
	in := map[string]any{}
	if err == nil && strings.TrimSpace(string(raw)) != "" {
		err = json.Unmarshal(raw, &in)
	}
	if err != nil || len(raw) > fcMaxBody {
		return nil, fcError("SerializationException", "invalid JSON body")
	}
	if problems := fcRealProblems(model.Validate(shapes.Input, in)); len(problems) > 0 {
		return nil, fcInvalid("%s", strings.Join(problems, "; "))
	}
	conformed, _ := model.Conform(shapes.Input, in).(map[string]any)
	if conformed == nil {
		conformed = map[string]any{}
	}
	return conformed, nil
}

// fcRealProblems drops pattern failures the service doesn't make: the
// Forecast model types RoleArn (an IAM role) and Filter.Value (an ARN or a
// status) as Forecast ARNs.
func fcRealProblems(problems []string) []string {
	out := problems[:0:0]
	for _, p := range problems {
		path, ok := strings.CutSuffix(p, ": does not match the required pattern")
		if ok && (strings.HasSuffix(path, "RoleArn") || strings.HasPrefix(path, "Filters[") && strings.HasSuffix(path, "].Value")) {
			continue
		}
		out = append(out, p)
	}
	return out
}

func respondForecastError(w http.ResponseWriter, e *fcErr) {
	writeForecastJSON(w, http.StatusBadRequest, map[string]any{"__type": e.code, "Message": e.msg}, e.code)
}

func writeForecastJSON(w http.ResponseWriter, status int, v any, errorType string) {
	body, _ := json.Marshal(v)
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	if errorType != "" {
		w.Header().Set("X-Amzn-ErrorType", errorType)
	}
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

// Small accessors for conformed JSON maps, shared by the Forecast, Forecast
// Query and Fraud Detector handlers.

func fcStr(m map[string]any, key string) string {
	s, _ := m[key].(string)
	return s
}

func fcObj(m map[string]any, key string) map[string]any {
	o, _ := m[key].(map[string]any)
	return o
}

func fcStrs(m map[string]any, key string) []string {
	items, _ := m[key].([]any)
	out := make([]string, 0, len(items))
	for _, item := range items {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func fcNum(m map[string]any, key string) (float64, bool) {
	n, ok := m[key].(float64)
	return n, ok
}

func fcBool(m map[string]any, key string) bool {
	b, _ := m[key].(bool)
	return b
}

func fcAnyList(items []string) []any {
	out := make([]any, len(items))
	for i, s := range items {
		out[i] = s
	}
	return out
}
