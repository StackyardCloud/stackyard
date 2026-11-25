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

// AWS Cloud9: awsJson1_1, X-Amz-Target AWSCloud9WorkspaceManagementService.*,
// signed cloud9. Environments run on instances in the EC2 store
// (cloud9_store.go). The awsJson plumbing here (c9JSONAPI, c9Serve) is shared
// with AWS Invoicing.

const (
	c9TargetPrefix = "AWSCloud9WorkspaceManagementService."
	// ponytail: 64 KiB covers every Cloud9 and Invoicing request (the largest,
	// 1,000 account IDs in a rule, is ~16 KiB) and bounds what one call stores.
	c9MaxBody = 64 << 10
)

type c9Err struct {
	status            int
	code, msg, reason string
}

func c9Errorf(status int, code, format string, args ...any) *c9Err {
	return &c9Err{status: status, code: code, msg: fmt.Sprintf(format, args...)}
}

func c9BadRequest(format string, args ...any) *c9Err {
	return c9Errorf(http.StatusBadRequest, "BadRequestException", format, args...)
}

func c9NotFound(format string, args ...any) *c9Err {
	return c9Errorf(http.StatusNotFound, "NotFoundException", format, args...)
}

func c9Conflict(format string, args ...any) *c9Err {
	return c9Errorf(http.StatusConflict, "ConflictException", format, args...)
}

func c9Limit(format string, args ...any) *c9Err {
	return c9Errorf(http.StatusTooManyRequests, "LimitExceededException", format, args...)
}

// c9Request is one decoded call: the model-conformed input and this server's
// scheme://host, for URLs handed back to the caller.
type c9Request struct {
	in   map[string]any
	base string
}

type c9Handler func(s *Server, req c9Request) (map[string]any, *c9Err)

// c9JSONAPI describes one awsJson service served by c9Serve.
type c9JSONAPI struct {
	prefix, signing, host, contentType, invalidCode string
	model                                           func() *awsmodels.Model
	handlers                                        map[string]c9Handler
}

var c9API = &c9JSONAPI{
	prefix: c9TargetPrefix, signing: "cloud9", host: "cloud9.", contentType: "application/x-amz-json-1.1",
	invalidCode: "BadRequestException", model: awsmodels.Cloud9Model,
}

func init() {
	c9API.handlers = map[string]c9Handler{
		"CreateEnvironmentEC2":           (*Server).c9CreateEnvironment,
		"CreateEnvironmentMembership":    (*Server).c9CreateMembership,
		"DeleteEnvironment":              (*Server).c9DeleteEnvironment,
		"DeleteEnvironmentMembership":    (*Server).c9DeleteMembership,
		"DescribeEnvironmentMemberships": (*Server).c9DescribeMemberships,
		"DescribeEnvironmentStatus":      (*Server).c9DescribeStatus,
		"DescribeEnvironments":           (*Server).c9DescribeEnvironments,
		"ListEnvironments":               (*Server).c9ListEnvironments,
		"ListTagsForResource":            (*Server).c9ListTags,
		"TagResource":                    (*Server).c9TagResource,
		"UntagResource":                  (*Server).c9UntagResource,
		"UpdateEnvironment":              (*Server).c9UpdateEnvironment,
		"UpdateEnvironmentMembership":    (*Server).c9UpdateMembership,
	}
}

func (s *Server) handleCloud9JSONRouter(w http.ResponseWriter, r *http.Request) bool {
	return s.c9Serve(w, r, c9API, c9OperationByName)
}

// c9Claims reports whether r is api's traffic: a POST whose X-Amz-Target has
// api.prefix, signed for api.signing (or unsigned to api's host). Requests
// signed for any other service are left to their own routers.
func c9Claims(r *http.Request, api *c9JSONAPI) (string, bool) {
	if r.Method != http.MethodPost {
		return "", false
	}
	op, ok := strings.CutPrefix(strings.TrimSpace(r.Header.Get("X-Amz-Target")), api.prefix)
	if !ok {
		return "", false
	}
	switch hint := sigV4ServiceHint(r); hint {
	case api.signing:
		return op, true
	case "":
		return op, strings.HasPrefix(strings.ToLower(r.Host), api.host)
	}
	return "", false
}

// c9Serve checks the body length and signature, validates and conforms the
// input against the CLI model, runs the handler and writes its output
// conformed to the output shape.
func (s *Server) c9Serve(w http.ResponseWriter, r *http.Request, api *c9JSONAPI, known map[string]bool) bool {
	op, ok := c9Claims(r, api)
	if !ok {
		return false
	}
	// A declared length bounds what is read; chunked bodies are refused.
	if r.ContentLength < 0 || r.ContentLength > c9MaxBody {
		c9RespondError(w, api, c9Errorf(http.StatusBadRequest, api.invalidCode, "request body must declare a Content-Length of at most %d bytes", c9MaxBody))
		return true
	}
	if ok, status, code, msg, _ := s.validateSigV4WithService(r, api.signing); !ok {
		c9RespondError(w, api, &c9Err{status: status, code: code, msg: msg})
		return true
	}
	h := api.handlers[op]
	if !known[op] || h == nil {
		c9RespondError(w, api, c9Errorf(http.StatusBadRequest, "UnknownOperationException", "unknown operation %s", op))
		return true
	}
	raw, err := io.ReadAll(io.LimitReader(r.Body, c9MaxBody))
	in := map[string]any{}
	if err == nil && strings.TrimSpace(string(raw)) != "" {
		err = json.Unmarshal(raw, &in)
	}
	if err != nil {
		c9RespondError(w, api, c9Errorf(http.StatusBadRequest, "SerializationException", "invalid JSON body"))
		return true
	}
	model := api.model()
	shapes := model.Operations[op]
	if problems := model.Validate(shapes.Input, in); len(problems) > 0 {
		e := c9Errorf(http.StatusBadRequest, api.invalidCode, "%s", strings.Join(problems, "; "))
		e.reason = "fieldValidationFailed"
		c9RespondError(w, api, e)
		return true
	}
	// Conform drops members the input shape doesn't define, so callers can't
	// reach the store's own fields.
	conformed, _ := model.Conform(shapes.Input, in).(map[string]any)
	if conformed == nil {
		conformed = map[string]any{}
	}
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	out, cerr := h(s, c9Request{in: conformed, base: scheme + "://" + r.Host})
	if cerr != nil {
		c9RespondError(w, api, cerr)
		return true
	}
	// Handlers build their output under the store lock; nothing in it is
	// shared with stored records.
	for _, k := range []string{"nextToken", "NextToken"} {
		if out[k] == "" {
			delete(out, k)
		}
	}
	body, _ := json.Marshal(model.Conform(shapes.Output, out))
	w.Header().Set("Content-Type", api.contentType)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
	return true
}

func c9RespondError(w http.ResponseWriter, api *c9JSONAPI, e *c9Err) {
	payload := map[string]string{"__type": e.code, "message": e.msg}
	if e.reason != "" && api.invalidCode == "ValidationException" && e.code == "ValidationException" {
		payload["reason"] = e.reason
	}
	body, _ := json.Marshal(payload)
	w.Header().Set("Content-Type", api.contentType)
	w.Header().Set("X-Amzn-ErrorType", e.code)
	w.WriteHeader(e.status)
	_, _ = w.Write(body)
}

// ---- request helpers (shared with Invoicing) ----

func c9Str(in map[string]any, key string) string {
	v, _ := in[key].(string)
	return v
}

func c9Map(in map[string]any, key string) map[string]any {
	v, _ := in[key].(map[string]any)
	return v
}

func c9List(in map[string]any, key string) []any {
	v, _ := in[key].([]any)
	return v
}

func c9Strings(in map[string]any, key string) []string {
	var out []string
	for _, v := range c9List(in, key) {
		if s, ok := v.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func c9Bool(in map[string]any, key string) (bool, bool) {
	v, ok := in[key].(bool)
	return v, ok
}

// c9Int reads an optional integer member; the model has checked its range.
func c9Int(in map[string]any, key string, def int) int {
	if v, ok := in[key].(float64); ok {
		return int(v)
	}
	return def
}

// c9Time reads a timestamp member sent as epoch seconds or RFC 3339.
func c9Time(in map[string]any, key string) (time.Time, bool, *c9Err) {
	switch v := in[key].(type) {
	case float64:
		sec := int64(v)
		return time.Unix(sec, int64((v-float64(sec))*1e9)).UTC(), true, nil
	case string:
		t, err := time.Parse(time.RFC3339Nano, v)
		if err != nil {
			return time.Time{}, false, c9Errorf(http.StatusBadRequest, "ValidationException", "%s is not a valid timestamp", key)
		}
		return t.UTC(), true, nil
	}
	return time.Time{}, false, nil
}

func c9Epoch(t time.Time) float64 { return float64(t.UnixMilli()) / 1000 }

// c9Page slices n items by an offset token and a page size.
func c9Page(token string, n, limit int, invalidCode string) (int, int, string, *c9Err) {
	start := 0
	if token != "" {
		v, err := strconv.ParseInt(token, 10, 32)
		if err != nil || v < 0 || int(v) > n {
			return 0, 0, "", c9Errorf(http.StatusBadRequest, invalidCode, "the pagination token is not valid")
		}
		start = int(v)
	}
	end := min(start+max(limit, 1), n)
	next := ""
	if end < n {
		next = strconv.Itoa(end)
	}
	return start, end, next, nil
}

func c9AnyStrings(items []string) []any {
	out := make([]any, len(items))
	for i, v := range items {
		out[i] = v
	}
	return out
}
