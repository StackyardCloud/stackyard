package server

// Shared plumbing for the Amazon Location Service standalone (V2) APIs:
// geo-maps, geo-places and geo-routes. They are stateless REST-JSON services
// authenticated by SigV4 or by an API key in the "key" query parameter, so
// each request is bound to its model input shape, validated, and answered
// from deterministic built-in data (see geomaps.go, geoplaces.go, georoutes.go).

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

// geosaMaxBody bounds a request body. The largest legitimate bodies are
// SnapToRoads traces (5000 points) and route matrices with avoidance areas.
const geosaMaxBody = 2 << 20

// geosaPricingBucket is reported in x-amz-geo-pricing-bucket on every reply.
const geosaPricingBucket = "Core"

type geosaOp struct {
	Name   string
	Method string
	URI    string
}

// geosaCall is one bound, validated request.
type geosaCall struct {
	in   map[string]any
	base string // scheme://host the caller used, for URLs inside style descriptors
}

// geosaBlobKind selects one of a fixed set of binary content types.
type geosaBlobKind int

const (
	geosaBlobJSON geosaBlobKind = iota + 1
	geosaBlobPNG
	geosaBlobJPEG
	geosaBlobProtobuf
	geosaBlobVectorTile
)

// geosaReply is a JSON document or a binary payload plus header members.
type geosaReply struct {
	body    map[string]any
	blob    []byte
	kind    geosaBlobKind
	pricing bool // send x-amz-geo-pricing-bucket
	maxAge  int  // Cache-Control max-age for binary payloads
}

type geosaError struct {
	status int
	code   string
	msg    string
	reason string
	fields []map[string]any
}

func geosaValidation(reason, field, format string, args ...any) *geosaError {
	msg := fmt.Sprintf(format, args...)
	e := &geosaError{status: http.StatusBadRequest, code: "ValidationException", msg: msg, reason: reason}
	if field != "" {
		e.fields = []map[string]any{{"name": field, "message": msg}}
	}
	return e
}

type geosaService struct {
	signingName string
	hosts       []string // endpoint host prefixes, e.g. "geo-maps." and "maps.geo."
	model       func() *awsmodels.Model
	ops         []geosaOp
	handlers    map[string]func(geosaCall) (*geosaReply, *geosaError)
	write       func(http.ResponseWriter, *geosaReply) // sets a constant Content-Type and writes the reply
}

// claims reports whether the request is for this service: signed for its
// signing name, or unsigned and sent to one of its endpoint hosts.
func (svc *geosaService) claims(r *http.Request) bool {
	if hint := strings.TrimSpace(sigV4ServiceHint(r)); hint != "" {
		return hint == svc.signingName
	}
	host := strings.ToLower(strings.TrimSpace(r.Host))
	for _, prefix := range svc.hosts {
		if strings.HasPrefix(host, prefix) {
			return true
		}
	}
	return false
}

func (s *Server) geosaServe(w http.ResponseWriter, r *http.Request, svc *geosaService) bool {
	if !svc.claims(r) {
		return false
	}
	// A declared length bounds what net/http reads, so require one.
	if r.ContentLength < 0 || r.ContentLength > geosaMaxBody {
		geosaWriteError(w, geosaValidation("Other", "", "request body must declare a Content-Length of at most %d bytes", geosaMaxBody))
		return true
	}
	if err := s.geosaAuthorize(r, svc.signingName); err != nil {
		geosaWriteError(w, err)
		return true
	}
	op, params := geosaMatch(svc.ops, r.Method, rawRequestPath(r))
	if op == "" {
		geosaWriteError(w, geosaValidation("UnknownOperation", "", "no %s operation matches %s %s", svc.signingName, r.Method, rawRequestPath(r)))
		return true
	}
	md := svc.model()
	in, verr := geosaBind(md, md.Operations[op].Input, r, params)
	if verr != nil {
		geosaWriteError(w, verr)
		return true
	}
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	rep, herr := svc.handlers[op](geosaCall{in: in, base: scheme + "://" + r.Host})
	if herr != nil {
		geosaWriteError(w, herr)
		return true
	}
	if rep.body != nil {
		conformed, _ := md.Conform(md.Operations[op].Output, rep.body).(map[string]any)
		rep.body = conformed
	}
	svc.write(w, rep)
	return true
}

// geosaAuthorize accepts a valid SigV4 signature for the service, or an
// unsigned request that carries an API key.
func (s *Server) geosaAuthorize(r *http.Request, signingName string) *geosaError {
	q := r.URL.Query()
	signed := strings.TrimSpace(r.Header.Get("Authorization")) != "" || q.Get("X-Amz-Algorithm") != "" || q.Get("X-Amz-Credential") != ""
	if !signed && skipSigV4Validation(r) {
		return nil
	}
	if !signed {
		// ponytail: any non-empty key is accepted; Location API keys are not
		// checked against the location store (its key values are placeholders).
		if key := q.Get("key"); key != "" && len(key) <= 1000 {
			return nil
		}
		return &geosaError{status: http.StatusForbidden, code: "AccessDeniedException", msg: "request must be signed with SigV4 or carry an API key"}
	}
	if ok, status, code, msg, _ := s.validateSigV4WithService(r, signingName); !ok {
		if status == http.StatusForbidden {
			code = "AccessDeniedException"
		}
		return &geosaError{status: status, code: code, msg: msg}
	}
	return nil
}

func geosaMatch(ops []geosaOp, method, rawPath string) (string, map[string]string) {
	path := strings.SplitN(rawPath, "?", 2)[0]
	for _, op := range ops {
		if op.Method != method {
			continue
		}
		if params, ok := locationPathParams(op.URI, path); ok {
			return op.Name, params
		}
	}
	return "", nil
}

// geosaBind decodes the JSON body and the URI and query-string members of
// shapeName into one input document, validating each against the model.
func geosaBind(md *awsmodels.Model, shapeName string, r *http.Request, params map[string]string) (map[string]any, *geosaError) {
	in := map[string]any{}
	raw, err := readBodyBytes(r)
	if err != nil {
		return nil, geosaValidation("CannotParse", "", "unable to read request body")
	}
	if len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, &in); err != nil || in == nil {
			return nil, geosaValidation("CannotParse", "", "request body is not a JSON object")
		}
	}
	if problems := md.Validate(shapeName, in); len(problems) > 0 {
		return nil, geosaProblems(problems)
	}
	shape := md.Shapes[shapeName]
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
			if slices.Contains(shape.Required, name) {
				return nil, geosaValidation("Missing", name, "%s is required", name)
			}
			continue
		}
		v, perr := geosaParseParam(md, ref.Shape, values)
		if perr != "" {
			return nil, geosaValidation("CannotParse", name, "%s %s", name, perr)
		}
		if problems := md.Validate(ref.Shape, v); len(problems) > 0 {
			_, reason, _ := strings.Cut(problems[0], ": ")
			return nil, geosaValidation("FieldValidationFailed", name, "%s %s", name, reason)
		}
		in[name] = v
	}
	return in, nil
}

// geosaParseParam converts URI or query-string text to the member's JSON type.
func geosaParseParam(md *awsmodels.Model, shapeName string, values []string) (any, string) {
	s := md.Shapes[shapeName]
	if s == nil {
		return values[0], ""
	}
	switch s.Type {
	case "list":
		out := []any{}
		for _, v := range values {
			for _, part := range strings.Split(v, ",") {
				if len(out) >= 100 {
					return nil, "has too many values"
				}
				item, perr := geosaParseParam(md, s.Member.Shape, []string{part})
				if perr != "" {
					return nil, perr
				}
				out = append(out, item)
			}
		}
		return out, ""
	case "integer", "long":
		n, err := strconv.ParseInt(values[0], 10, 64)
		if err != nil {
			return nil, "must be an integer"
		}
		return float64(n), ""
	case "float", "double":
		f, err := strconv.ParseFloat(values[0], 64)
		if err != nil || math.IsNaN(f) || math.IsInf(f, 0) {
			return nil, "must be a number"
		}
		return f, ""
	case "boolean":
		b, err := strconv.ParseBool(values[0])
		if err != nil {
			return nil, "must be a boolean"
		}
		return b, ""
	}
	return values[0], ""
}

func geosaProblems(problems []string) *geosaError {
	reason := "FieldValidationFailed"
	fields := make([]map[string]any, 0, len(problems))
	for _, p := range problems {
		name, msg, _ := strings.Cut(p, ": ")
		if strings.HasPrefix(msg, "missing required member") {
			reason = "Missing"
		} else if strings.HasPrefix(msg, "unknown member") {
			reason = "UnknownField"
		}
		fields = append(fields, map[string]any{"name": name, "message": msg})
	}
	return &geosaError{status: http.StatusBadRequest, code: "ValidationException", msg: strings.Join(problems, "; "), reason: reason, fields: fields}
}

func geosaWriteError(w http.ResponseWriter, e *geosaError) {
	body := map[string]any{"message": e.msg}
	if e.code == "ValidationException" {
		fields := e.fields
		if fields == nil {
			fields = []map[string]any{}
		}
		reason := e.reason
		if reason == "" {
			reason = "Other"
		}
		body["reason"], body["fieldList"] = reason, fields
	}
	raw, _ := json.Marshal(body)
	w.Header().Set("X-Amzn-ErrorType", e.code)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(e.status)
	_, _ = w.Write(raw)
}

// --- shared geometry -------------------------------------------------------

const geosaEarthRadius = 6371008.8 // mean Earth radius, metres

// geosaPos reads a validated [longitude, latitude] list and range-checks it.
func geosaPos(v any, field string) ([2]float64, *geosaError) {
	items, _ := v.([]any)
	if len(items) != 2 {
		return [2]float64{}, geosaValidation("FieldValidationFailed", field, "%s must be [longitude, latitude]", field)
	}
	lon, _ := items[0].(float64)
	lat, _ := items[1].(float64)
	if lon < -180 || lon > 180 || lat < -90 || lat > 90 {
		return [2]float64{}, geosaValidation("FieldValidationFailed", field, "%s has an invalid position: longitude must be within [-180, 180] and latitude within [-90, 90]", field)
	}
	return [2]float64{lon, lat}, nil
}

// geosaBBox reads [west, south, east, north].
func geosaBBox(v any, field string) ([4]float64, *geosaError) {
	items, _ := v.([]any)
	if len(items) != 4 {
		return [4]float64{}, geosaValidation("FieldValidationFailed", field, "%s must be [west, south, east, north]", field)
	}
	sw, err := geosaPos(items[:2], field)
	if err != nil {
		return [4]float64{}, err
	}
	ne, err := geosaPos(items[2:], field)
	if err != nil {
		return [4]float64{}, err
	}
	if sw[1] > ne[1] {
		return [4]float64{}, geosaValidation("FieldValidationFailed", field, "%s south latitude must not exceed north latitude", field)
	}
	return [4]float64{sw[0], sw[1], ne[0], ne[1]}, nil
}

// geosaInBBox handles boxes that cross the antimeridian (west > east).
func geosaInBBox(b [4]float64, p [2]float64) bool {
	if p[1] < b[1] || p[1] > b[3] {
		return false
	}
	if b[0] <= b[2] {
		return p[0] >= b[0] && p[0] <= b[2]
	}
	return p[0] >= b[0] || p[0] <= b[2]
}

// geosaDistance is the great-circle (haversine) distance in metres.
func geosaDistance(a, b [2]float64) float64 {
	rad := math.Pi / 180
	dLat := (b[1] - a[1]) * rad
	dLon := (b[0] - a[0]) * rad
	h := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(a[1]*rad)*math.Cos(b[1]*rad)*math.Sin(dLon/2)*math.Sin(dLon/2)
	return 2 * geosaEarthRadius * math.Asin(math.Min(1, math.Sqrt(h)))
}

// geosaOffset moves p by distance metres along bearing degrees (great circle).
func geosaOffset(p [2]float64, bearing, distance float64) [2]float64 {
	rad := math.Pi / 180
	lat1, lon1, brg := p[1]*rad, p[0]*rad, bearing*rad
	d := distance / geosaEarthRadius
	lat2 := math.Asin(math.Sin(lat1)*math.Cos(d) + math.Cos(lat1)*math.Sin(d)*math.Cos(brg))
	lon2 := lon1 + math.Atan2(math.Sin(brg)*math.Sin(d)*math.Cos(lat1), math.Cos(d)-math.Sin(lat1)*math.Sin(lat2))
	lon := math.Mod(lon2/rad+540, 360) - 180
	return [2]float64{geosaRound(lon, 7), geosaRound(lat2/rad, 7)}
}

func geosaRound(v float64, places int) float64 {
	m := math.Pow(10, float64(places))
	return math.Round(v*m) / m
}

func geosaPosJSON(p [2]float64) []any { return []any{p[0], p[1]} }

func geosaStr(in map[string]any, key string) string {
	v, _ := in[key].(string)
	return v
}

func geosaObj(in map[string]any, key string) map[string]any {
	v, _ := in[key].(map[string]any)
	return v
}

func geosaList(in map[string]any, key string) []any {
	v, _ := in[key].([]any)
	return v
}

// geosaInt reads a validated integer member, or def when absent.
func geosaInt(in map[string]any, key string, def int64) int64 {
	if v, ok := in[key].(float64); ok {
		return int64(v)
	}
	return def
}

func geosaHas(list []any, want string) bool {
	for _, v := range list {
		if s, _ := v.(string); s == want {
			return true
		}
	}
	return false
}
