package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

// nmMaxBody bounds a request body. Core network policy documents are the
// largest inputs; 1 MiB is plenty for any realistic policy.
const nmMaxBody = 1 << 20

// nmMaxPathSegments bounds the backtracking path matcher.
const nmMaxPathSegments = 32

// nmErr is a modelled Network Manager error.
type nmErr struct {
	status int
	code   string
	msg    string
	extra  map[string]any
}

func nmValidation(format string, args ...any) *nmErr {
	return &nmErr{status: http.StatusBadRequest, code: "ValidationException", msg: fmt.Sprintf(format, args...)}
}

func nmNotFound(resourceType, id string) *nmErr {
	return &nmErr{status: http.StatusNotFound, code: "ResourceNotFoundException",
		msg:   fmt.Sprintf("The %s %s does not exist.", resourceType, id),
		extra: map[string]any{"ResourceId": id, "ResourceType": resourceType}}
}

func nmConflict(resourceType, id, format string, args ...any) *nmErr {
	return &nmErr{status: http.StatusConflict, code: "ConflictException", msg: fmt.Sprintf(format, args...),
		extra: map[string]any{"ResourceId": id, "ResourceType": resourceType}}
}

func nmQuota(resourceType, format string, args ...any) *nmErr {
	return &nmErr{status: http.StatusPaymentRequired, code: "ServiceQuotaExceededException", msg: fmt.Sprintf(format, args...),
		extra: map[string]any{"ResourceType": resourceType, "ServiceCode": "networkmanager", "LimitCode": "L-" + strings.ToUpper(resourceType)}}
}

// nmRoute is one operation's compiled URI template.
type nmRoute struct {
	op       networkmanagerOperation
	segments []string // literal segments, or "{label}" placeholders
}

var nmRoutes = func() []nmRoute {
	out := make([]nmRoute, 0, len(networkmanagerOperations))
	for _, op := range networkmanagerOperations {
		out = append(out, nmRoute{op: op, segments: splitPathSegments(op.URI)})
	}
	return out
}()

// nmMatch finds the operation for a method and path. Labels normally hold one
// (percent-encoded) segment; a second pass lets a label absorb several raw
// segments, so ARNs sent with unescaped slashes still route.
func nmMatch(method string, path []string) (networkmanagerOperation, map[string]string, bool) {
	for _, greedy := range []bool{false, true} {
		for _, rt := range nmRoutes {
			if rt.op.Method != method {
				continue
			}
			labels := map[string]string{}
			if nmMatchSegments(rt.segments, path, labels, greedy) {
				return rt.op, labels, true
			}
		}
	}
	return networkmanagerOperation{}, nil, false
}

func nmMatchSegments(tmpl, path []string, labels map[string]string, greedy bool) bool {
	if len(tmpl) == 0 {
		return len(path) == 0
	}
	if len(path) == 0 {
		return false
	}
	t := tmpl[0]
	if !strings.HasPrefix(t, "{") {
		return t == path[0] && nmMatchSegments(tmpl[1:], path[1:], labels, greedy)
	}
	name := strings.Trim(t, "{}")
	maxTake := 1
	if greedy {
		maxTake = len(path) - (len(tmpl) - 1)
	}
	for take := 1; take <= maxTake; take++ {
		labels[name] = strings.Join(path[:take], "/")
		if nmMatchSegments(tmpl[1:], path[take:], labels, greedy) {
			return true
		}
	}
	delete(labels, name)
	return false
}

func isNetworkManagerRESTCandidate(r *http.Request) bool {
	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service == "networkmanager" {
		return true
	}
	return service == "" && strings.HasPrefix(strings.ToLower(strings.TrimSpace(r.Host)), "networkmanager.")
}

func (s *Server) handleNetworkManagerRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isNetworkManagerRESTCandidate(r) {
		return false
	}
	raw := splitPathSegments(rawRequestPath(r))
	if len(raw) == 0 || len(raw) > nmMaxPathSegments {
		respondNMError(w, &nmErr{status: http.StatusNotFound, code: "UnknownOperationException", msg: "no Network Manager operation matches " + r.Method + " " + rawRequestPath(r)})
		return true
	}
	path := make([]string, len(raw))
	for i, v := range raw {
		u, err := url.PathUnescape(v)
		if err != nil {
			respondNMError(w, nmValidation("invalid path segment"))
			return true
		}
		path[i] = u
	}
	op, labels, ok := nmMatch(r.Method, path)
	if !ok {
		respondNMError(w, &nmErr{status: http.StatusNotFound, code: "UnknownOperationException", msg: "no Network Manager operation matches " + r.Method + " " + rawRequestPath(r)})
		return true
	}
	// A declared length bounds what net/http reads, so require one; r.Body is
	// never replaced.
	if r.ContentLength < 0 || r.ContentLength > nmMaxBody {
		respondNMError(w, nmValidation("request body must declare a Content-Length of at most 1 MiB"))
		return true
	}
	if ok, status, code, msg, _ := s.validateSigV4WithService(r, "networkmanager"); !ok {
		respondNMError(w, &nmErr{status: status, code: code, msg: msg})
		return true
	}
	in, err := nmDecode(r, op.Name, labels)
	if err != nil {
		respondNMError(w, err)
		return true
	}
	out, err := s.nmDispatch(op.Name, in)
	if err != nil {
		respondNMError(w, err)
		return true
	}
	respondNMJSON(w, out)
	return true
}

// nmDecode builds the operation input: the JSON body (validated against the
// input shape, which rejects unknown members) plus URI labels and query
// parameters converted to their member shapes and validated one by one.
func nmDecode(r *http.Request, op string, labels map[string]string) (map[string]any, *nmErr) {
	m := awsmodels.NetworkManagerModel()
	shapeName := m.Operations[op].Input
	in := map[string]any{}
	if r.ContentLength > 0 {
		raw, err := readBodyBytes(r)
		if err != nil {
			return nil, nmValidation("unable to read request body")
		}
		if len(bytes.TrimSpace(raw)) > 0 {
			if err := json.Unmarshal(raw, &in); err != nil {
				return nil, nmValidation("request body is not a JSON object")
			}
		}
	}
	if shapeName == "" {
		return in, nil
	}
	if problems := m.Validate(shapeName, in); len(problems) > 0 {
		return nil, nmValidation("%s", strings.Join(problems, "; "))
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
			return nil, nmValidation("%s %s", member, err.Error())
		}
		if problems := m.Validate(ref.Shape, conv); len(problems) > 0 {
			_, reason, _ := strings.Cut(problems[0], ": ")
			return nil, nmValidation("%s %s", member, reason)
		}
		in[member] = conv
	}
	for _, req := range shape.Required {
		if _, ok := in[req]; !ok {
			return nil, nmValidation("missing required member %s", req)
		}
	}
	return in, nil
}

// nmConvert turns a URI label (string) or query values ([]string) into the
// JSON-decoded form of shapeName so the model can validate it.
func nmConvert(m *awsmodels.Model, shapeName string, value any) (any, error) {
	s := m.Shapes[shapeName]
	first := func() string {
		if vals, ok := value.([]string); ok {
			if len(vals) == 0 {
				return ""
			}
			return vals[0]
		}
		return value.(string)
	}
	switch s.Type {
	case "list":
		vals, ok := value.([]string)
		if !ok {
			vals = []string{value.(string)}
		}
		out := make([]any, 0, len(vals))
		for _, v := range vals {
			// Query lists arrive repeated or comma-joined.
			for _, part := range strings.Split(v, ",") {
				if part != "" {
					out = append(out, part)
				}
			}
		}
		return out, nil
	case "integer", "long":
		n, err := strconv.ParseInt(first(), 10, 32)
		if err != nil {
			return nil, fmt.Errorf("must be an integer")
		}
		return float64(n), nil
	case "boolean":
		b, err := strconv.ParseBool(first())
		if err != nil {
			return nil, fmt.Errorf("must be a boolean")
		}
		return b, nil
	}
	return first(), nil
}

func respondNMJSON(w http.ResponseWriter, body map[string]any) {
	raw, _ := json.Marshal(body)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(raw)
}

func respondNMError(w http.ResponseWriter, e *nmErr) {
	body := map[string]any{"Message": e.msg}
	for k, v := range e.extra {
		body[k] = v
	}
	raw, _ := json.Marshal(body)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Amzn-ErrorType", e.code)
	w.WriteHeader(e.status)
	_, _ = w.Write(raw)
}

func nmStr(in map[string]any, key string) string {
	v, _ := in[key].(string)
	return v
}

func nmOptStr(in map[string]any, key string) (string, bool) {
	v, ok := in[key].(string)
	return v, ok
}

func nmInt(in map[string]any, key string) (int, bool) {
	v, ok := in[key].(float64)
	return int(v), ok
}

func nmList(in map[string]any, key string) []any {
	v, _ := in[key].([]any)
	return v
}

func nmStrings(in map[string]any, key string) []string {
	var out []string
	for _, v := range nmList(in, key) {
		if s, ok := v.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func nmMap(in map[string]any, key string) map[string]any {
	v, _ := in[key].(map[string]any)
	return v
}

// nmPage pages items by an offset token; maxResults defaults to every item
// up to the model maximum (500).
func nmPage(in map[string]any, items []any) ([]any, string, *nmErr) {
	start := 0
	if tok := nmStr(in, "NextToken"); tok != "" {
		n, err := strconv.ParseInt(tok, 10, 32)
		if err != nil || n < 0 || int(n) > len(items) {
			return nil, "", nmValidation("The NextToken is invalid.")
		}
		start = int(n)
	}
	size := 500
	if n, ok := nmInt(in, "MaxResults"); ok {
		size = n
	}
	end := min(start+size, len(items))
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return items[start:end], next, nil
}
