package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

// AWS IoT Data Plane: REST, signed iotdata. Its /things/{thingName}/shadow
// paths overlap the IoT control plane's /things prefix, so it is claimed only
// by signing name (or, unsigned, by its data-ats.iot.* / data.iot.* host)
// ahead of the IoT router. Shadows live in iotdataplane_shadow.go; Publish,
// retained messages and client connections in iotdataplane_mqtt.go.

const iotdMaxBody = 128 << 10 // the MQTT message size limit

var (
	iotdThingNameRe  = regexp.MustCompile(`^[a-zA-Z0-9:_-]{1,128}$`)
	iotdShadowNameRe = regexp.MustCompile(`^[$a-zA-Z0-9:_-]{1,64}$`)
)

type iotdErr struct {
	status    int
	code, msg string
}

var iotdErrStatus = map[string]int{
	"ResourceNotFoundException": http.StatusNotFound, "ConflictException": http.StatusConflict,
	"RequestEntityTooLargeException": http.StatusRequestEntityTooLarge, "ForbiddenException": http.StatusForbidden,
	"UnsupportedDocumentEncodingException": http.StatusUnsupportedMediaType, "ThrottlingException": http.StatusTooManyRequests,
	"MethodNotAllowedException": http.StatusMethodNotAllowed,
}

func iotdErrorf(code, format string, args ...any) *iotdErr {
	status := iotdErrStatus[code]
	if status == 0 {
		status = http.StatusBadRequest
	}
	return &iotdErr{status: status, code: code, msg: fmt.Sprintf(format, args...)}
}

func iotdInvalid(format string, args ...any) *iotdErr {
	return iotdErrorf("InvalidRequestException", format, args...)
}

// iotdRoute is one REST operation; iotjobsdataplane.go shares the matcher.
type iotdRoute struct {
	name, method string
	segments     []string
}

var iotdRoutes = func() []iotdRoute {
	out := make([]iotdRoute, 0, len(iotDataPlaneOperations))
	for _, op := range iotDataPlaneOperations {
		out = append(out, iotdRoute{name: op.Name, method: op.Method, segments: splitPathSegments(op.URI)})
	}
	return out
}()

// iotdMatchRoute matches the request's raw path segment by segment; a
// {label} takes exactly one (percent-decoded) segment, so a topic with
// encoded slashes stays one label.
func iotdMatchRoute(r *http.Request, routes []iotdRoute) (string, map[string]string, bool) {
	raw := splitPathSegments(rawRequestPath(r))
	if len(raw) == 0 || len(raw) > 8 {
		return "", nil, false
	}
	path := make([]string, len(raw))
	for i, seg := range raw {
		v, err := url.PathUnescape(seg)
		if err != nil {
			return "", nil, false
		}
		path[i] = v
	}
	for _, rt := range routes {
		if rt.method != r.Method || len(rt.segments) != len(path) {
			continue
		}
		labels := map[string]string{}
		ok := true
		for i, seg := range rt.segments {
			if strings.HasPrefix(seg, "{") && strings.HasSuffix(seg, "}") {
				labels[seg[1:len(seg)-1]] = path[i]
			} else if seg != path[i] {
				ok = false
				break
			}
		}
		if ok {
			return rt.name, labels, true
		}
	}
	return "", nil, false
}

func isIoTDataPlaneCandidate(r *http.Request) bool {
	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service == "iotdata" {
		return true
	}
	host := strings.ToLower(strings.TrimSpace(r.Host))
	return service == "" && (strings.HasPrefix(host, "data-ats.iot.") || strings.HasPrefix(host, "data.iot."))
}

func (s *Server) handleIoTDataPlaneRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isIoTDataPlaneCandidate(r) {
		return false
	}
	op, labels, ok := iotdMatchRoute(r, iotdRoutes)
	if !ok {
		iotdRespondError(w, iotdErrorf("MethodNotAllowedException", "no IoT data plane operation matches %s %s", r.Method, rawRequestPath(r)))
		return true
	}
	// A declared length bounds what net/http reads; refuse chunked bodies.
	if r.ContentLength < 0 || r.ContentLength > iotdMaxBody {
		iotdRespondError(w, iotdErrorf("RequestEntityTooLargeException", "request body must declare a Content-Length of at most %d bytes", iotdMaxBody))
		return true
	}
	if ok, status, code, msg, _ := s.validateSigV4WithService(r, "iotdata"); !ok {
		iotdRespondError(w, &iotdErr{status: status, code: code, msg: msg})
		return true
	}
	var body []byte
	if r.ContentLength > 0 {
		raw, err := readBodyBytes(r)
		if err != nil {
			iotdRespondError(w, iotdInvalid("unable to read request body"))
			return true
		}
		body = raw
	}
	out, err := s.iotdDispatch(op, labels, r, body)
	if err != nil {
		iotdRespondError(w, err)
		return true
	}
	raw, _ := json.Marshal(out)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(raw)
	return true
}

func (s *Server) iotdDispatch(op string, labels map[string]string, r *http.Request, body []byte) (map[string]any, *iotdErr) {
	q := r.URL.Query()
	st := s.iotData
	switch op {
	case "GetThingShadow", "UpdateThingShadow", "DeleteThingShadow":
		thing, shadow, err := iotdShadowTarget(labels["thingName"], q)
		if err != nil {
			return nil, err
		}
		switch op {
		case "GetThingShadow":
			return st.getShadow(thing, shadow)
		case "DeleteThingShadow":
			return st.deleteShadow(thing, shadow)
		}
		return st.updateShadow(thing, shadow, body)
	case "ListNamedShadowsForThing":
		thing := labels["thingName"]
		if !iotdThingNameRe.MatchString(thing) {
			return nil, iotdInvalid("thingName must be 1-128 characters of [a-zA-Z0-9:_-]")
		}
		start, size, err := iotdPage(q, "pageSize", 100)
		if err != nil {
			return nil, err
		}
		return st.listNamedShadows(thing, start, size), nil
	case "Publish":
		msg, err := iotdParseMessage(labels["topic"], r, body, true)
		if err != nil {
			return nil, err
		}
		return map[string]any{}, st.publish(msg)
	case "GetRetainedMessage":
		return st.getRetained(labels["topic"])
	case "ListRetainedMessages":
		start, size, err := iotdPage(q, "maxResults", 200)
		if err != nil {
			return nil, err
		}
		return st.listRetained(start, size), nil
	}
	// The rest address a client connection.
	clientID := labels["clientId"]
	if clientID == "" || len(clientID) > 128 || strings.HasPrefix(clientID, "$") {
		return nil, iotdInvalid("clientId must be 1-128 characters and not start with $")
	}
	switch op {
	case "GetConnection":
		sock, err := iotdBool(q, "includeSocketInformation")
		if err != nil {
			return nil, err
		}
		return st.getConnection(clientID, sock)
	case "DeleteConnection":
		clean, err := iotdBool(q, "cleanSession")
		if err != nil {
			return nil, err
		}
		if _, err := iotdBool(q, "preventWillMessage"); err != nil {
			return nil, err
		}
		return map[string]any{}, st.deleteConnection(clientID, clean)
	case "ListSubscriptions":
		start, size, err := iotdPage(q, "maxResults", 200)
		if err != nil {
			return nil, err
		}
		return st.listSubscriptions(clientID, start, size)
	}
	// SendDirectMessage
	msg, err := iotdParseMessage(q.Get("topic"), r, body, false)
	if err != nil {
		return nil, err
	}
	if v := q.Get("timeout"); v != "" {
		if n, perr := strconv.ParseInt(v, 10, 32); perr != nil || n < 1 || n > 30 {
			return nil, iotdInvalid("timeout must be between 1 and 30 seconds")
		}
	}
	if _, err := iotdBool(q, "confirmation"); err != nil {
		return nil, err
	}
	return st.sendDirect(clientID, msg)
}

// iotdShadowTarget validates a thing name and the optional ?name= shadow
// name ("" is the classic shadow).
func iotdShadowTarget(thing string, q url.Values) (string, string, *iotdErr) {
	if !iotdThingNameRe.MatchString(thing) {
		return "", "", iotdInvalid("thingName must be 1-128 characters of [a-zA-Z0-9:_-]")
	}
	name, given := q["name"]
	if !given {
		return thing, "", nil
	}
	if len(name) != 1 || !iotdShadowNameRe.MatchString(name[0]) {
		return "", "", iotdInvalid("shadowName must be 1-64 characters of [$a-zA-Z0-9:_-]")
	}
	return thing, name[0], nil
}

func iotdBool(q url.Values, key string) (bool, *iotdErr) {
	v := q.Get(key)
	if v == "" {
		return false, nil
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return false, iotdInvalid("%s must be a boolean", key)
	}
	return b, nil
}

// iotdPage reads an offset nextToken and a page size (1..limit, default limit).
func iotdPage(q url.Values, sizeKey string, limit int) (int, int, *iotdErr) {
	start, size := 0, limit
	if v := q.Get("nextToken"); v != "" {
		n, err := strconv.ParseInt(v, 10, 32)
		if err != nil || n < 0 {
			return 0, 0, iotdInvalid("invalid nextToken")
		}
		start = int(n)
	}
	if v := q.Get(sizeKey); v != "" {
		n, err := strconv.ParseInt(v, 10, 32)
		if err != nil || n < 1 || int(n) > limit {
			return 0, 0, iotdInvalid("%s must be between 1 and %d", sizeKey, limit)
		}
		size = int(n)
	}
	return start, size, nil
}

// iotdSlice clamps a page window to n items and returns the next token.
func iotdSlice(n, start, size int) (int, int, string) {
	start = min(start, n)
	end := min(start+size, n)
	if end < n {
		return start, end, strconv.Itoa(end)
	}
	return start, end, ""
}

func iotdRespondError(w http.ResponseWriter, e *iotdErr) {
	body, _ := json.Marshal(map[string]string{"message": e.msg})
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Amzn-ErrorType", e.code)
	w.WriteHeader(e.status)
	_, _ = w.Write(body)
}
