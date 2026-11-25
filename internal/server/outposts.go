package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

// AWS Outposts: REST JSON, signed outposts. Sites, outposts, quotes, orders
// (which advance one step per read until FULFILLED and then install assets),
// capacity tasks, private connectivity, connections, billing and renewals.
// State lives in outposts_store.go and outposts_orders.go.
//
// opstRESTAPI is the REST JSON plumbing shared with Snow Device Management:
// route by method and path template, check the declared body length and the
// signature, validate and conform the input against the CLI model, and write
// the handler's output conformed to the output shape.

// ponytail: 64 KiB covers any real request (the largest is a quote with a
// handful of capacities and constraints).
const opstMaxBody = 64 << 10

type opstRoute struct {
	Name, Method, URI string
	segments          []string
}

type opstRESTAPI struct {
	signing, hostPrefix string
	respondError        func(w http.ResponseWriter, e *budErr)
	model               func() *awsmodels.Model
	routes              func() []opstRoute
	dispatch            func(s *Server, op string, in map[string]any) (map[string]any, *budErr)
}

// opstErrorf builds an error with the status its code carries in the models.
func opstErrorf(code, format string, args ...any) *budErr {
	status := map[string]int{
		"ResourceNotFoundException": http.StatusNotFound, "NotFoundException": http.StatusNotFound,
		"ConflictException": http.StatusConflict, "AccessDeniedException": http.StatusForbidden,
		"ServiceQuotaExceededException": http.StatusPaymentRequired, "InternalServerException": http.StatusInternalServerError,
		"ThrottlingException": http.StatusTooManyRequests,
	}[code]
	if status == 0 {
		status = http.StatusBadRequest
	}
	return &budErr{status: status, code: code, msg: fmt.Sprintf(format, args...)}
}

func opstInvalid(format string, args ...any) *budErr {
	return opstErrorf("ValidationException", format, args...)
}

func opstCompileRoutes(names []opstRoute) func() []opstRoute {
	return sync.OnceValue(func() []opstRoute {
		out := make([]opstRoute, len(names))
		for i, rt := range names {
			rt.segments = splitPathSegments(rt.URI)
			out[i] = rt
		}
		return out
	})
}

func (api *opstRESTAPI) candidate(r *http.Request) bool {
	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service == api.signing {
		return true
	}
	return service == "" && strings.HasPrefix(strings.ToLower(strings.TrimSpace(r.Host)), api.hostPrefix)
}

// match finds the operation for a method and decoded path; a second pass lets
// a label (an ARN sent with raw slashes) absorb several segments.
func (api *opstRESTAPI) match(method string, path []string) (string, map[string]string, bool) {
	for _, greedy := range []bool{false, true} {
		for _, rt := range api.routes() {
			if rt.Method != method {
				continue
			}
			labels := map[string]string{}
			if nmMatchSegments(rt.segments, path, labels, greedy) {
				return rt.Name, labels, true
			}
		}
	}
	return "", nil, false
}

func (s *Server) opstServe(w http.ResponseWriter, r *http.Request, api *opstRESTAPI) bool {
	if !api.candidate(r) {
		return false
	}
	raw := splitPathSegments(rawRequestPath(r))
	unknown := func() bool {
		opstRespondError(w, api, opstErrorf("UnknownOperationException", "no operation matches %s %s", r.Method, rawRequestPath(r)))
		return true
	}
	if len(raw) == 0 || len(raw) > nmMaxPathSegments {
		return unknown()
	}
	path := make([]string, len(raw))
	for i, v := range raw {
		u, err := url.PathUnescape(v)
		if err != nil {
			opstRespondError(w, api, opstInvalid("invalid path segment"))
			return true
		}
		path[i] = u
	}
	op, labels, ok := api.match(r.Method, path)
	if !ok {
		return unknown()
	}
	// A declared length bounds what net/http reads; refuse chunked bodies.
	if r.ContentLength < 0 || r.ContentLength > opstMaxBody {
		opstRespondError(w, api, opstInvalid("request body must declare a Content-Length of at most %d bytes", opstMaxBody))
		return true
	}
	if ok, status, code, msg, _ := s.validateSigV4WithService(r, api.signing); !ok {
		opstRespondError(w, api, &budErr{status: status, code: code, msg: msg})
		return true
	}
	in, err := opstDecode(r, api.model(), op, labels)
	if err != nil {
		opstRespondError(w, api, err)
		return true
	}
	out, err := api.dispatch(s, op, in)
	if err != nil {
		opstRespondError(w, api, err)
		return true
	}
	// Handlers return values built under their store lock, so nothing here
	// is shared with stored records.
	for _, k := range []string{"NextToken", "nextToken"} {
		if out[k] == "" {
			delete(out, k)
		}
	}
	m := api.model()
	body, _ := json.Marshal(m.Conform(m.Operations[op].Output, out))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
	return true
}

// opstDecode builds the operation input from the JSON body (validated against
// the input shape, which refuses unknown members) plus URI labels and query
// parameters converted to their member shapes.
func opstDecode(r *http.Request, m *awsmodels.Model, op string, labels map[string]string) (map[string]any, *budErr) {
	shapeName := m.Operations[op].Input
	in := map[string]any{}
	if r.ContentLength > 0 {
		// The caller checked Content-Length against opstMaxBody.
		raw, err := io.ReadAll(io.LimitReader(r.Body, opstMaxBody+1))
		if err != nil || len(raw) > opstMaxBody {
			return nil, opstInvalid("unable to read request body")
		}
		if len(bytes.TrimSpace(raw)) > 0 {
			if err := json.Unmarshal(raw, &in); err != nil {
				return nil, opstInvalid("request body is not a JSON object")
			}
		}
	}
	if problems := m.Validate(shapeName, in); len(problems) > 0 {
		return nil, opstInvalid("%s", strings.Join(problems, "; "))
	}
	in, _ = m.Conform(shapeName, in).(map[string]any)
	if in == nil {
		in = map[string]any{}
	}
	shape := m.Shapes[shapeName]
	if shape == nil {
		return in, nil
	}
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
			return nil, opstInvalid("%s %s", member, err.Error())
		}
		if problems := m.Validate(ref.Shape, conv); len(problems) > 0 {
			_, reason, _ := strings.Cut(problems[0], ": ")
			return nil, opstInvalid("%s %s", member, reason)
		}
		in[member] = conv
	}
	for _, req := range shape.Required {
		if _, ok := in[req]; !ok {
			return nil, opstInvalid("missing required member %s", req)
		}
	}
	return in, nil
}

func opstRespondError(w http.ResponseWriter, api *opstRESTAPI, e *budErr) {
	api.respondError(w, e)
}

// opstWriteError writes Outposts' error shape (a Message member).
func opstWriteError(w http.ResponseWriter, e *budErr) {
	body, _ := json.Marshal(map[string]string{"Message": e.msg})
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Amzn-ErrorType", e.code)
	w.WriteHeader(e.status)
	_, _ = w.Write(body)
}

// ---- pagination ----

// opstPage returns the [start, end) window over total items and the next
// token, formatted by format (Outposts tokens must look like "N##x").
func opstPage(token string, total, size int, parse *regexp.Regexp, format string) (int, int, string, *budErr) {
	start := 0
	if token != "" {
		m := parse.FindStringSubmatch(token)
		if m == nil {
			return 0, 0, "", opstInvalid("The NextToken is not valid.")
		}
		n, err := strconv.ParseInt(m[1], 10, 32)
		if err != nil || n < 0 || int(n) > total {
			return 0, 0, "", opstInvalid("The NextToken is not valid.")
		}
		start = int(n)
	}
	end := min(total, start+size)
	next := ""
	if end < total {
		next = fmt.Sprintf(format, end)
	}
	return start, end, next, nil
}

// ---- Outposts ----

var opstTokenRE = regexp.MustCompile(`^(\d+)##page$`)

var opstAPI = &opstRESTAPI{
	signing: "outposts", hostPrefix: "outposts.", respondError: opstWriteError,
	model:  awsmodels.OutpostsModel,
	routes: opstCompileRoutes(outpostsOperations),
	dispatch: func(s *Server, op string, in map[string]any) (map[string]any, *budErr) {
		return s.outposts.dispatch(op, in)
	},
}

func (s *Server) handleOutpostsRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	return s.opstServe(w, r, opstAPI)
}

// opstPageOf pages a list for Outposts: MaxResults defaults to 100.
func opstPageOf(in map[string]any, total int) (int, int, string, *budErr) {
	return opstPage(budStr(in, "NextToken"), total, budInt(in, "MaxResults", 100), opstTokenRE, "%d##page")
}
