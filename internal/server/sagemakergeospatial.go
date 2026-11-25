package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

// Amazon SageMaker geospatial capabilities: REST JSON, signed
// sagemaker-geospatial. The router claims only requests signed for that name
// (or sent unsigned to a sagemaker-geospatial.* host), so SageMaker and
// SageMaker Runtime, which sign as sagemaker, keep their traffic. State lives
// in sagemakergeospatial_store.go; the raster catalog and search in
// sagemakergeospatial_catalog.go.

const (
	sgeoSigningName = "sagemaker-geospatial"
	// ponytail: 1 MiB covers any real request; area-of-interest polygons are
	// the only large members.
	sgeoMaxBody  = 1 << 20
	sgeoTileSize = 256
)

type sgeoRoute struct {
	op       sageMakerGeospatialOperation
	segments []string
}

var sgeoRoutes = func() []sgeoRoute {
	out := make([]sgeoRoute, 0, len(sageMakerGeospatialOperations))
	for _, op := range sageMakerGeospatialOperations {
		out = append(out, sgeoRoute{op: op, segments: splitPathSegments(op.URI)})
	}
	return out
}()

// sgeoMatch finds the operation for a method and decoded path; a second pass
// lets a label (an ARN sent with raw slashes) absorb several segments.
func sgeoMatch(method string, path []string) (string, map[string]string, bool) {
	for _, greedy := range []bool{false, true} {
		for _, rt := range sgeoRoutes {
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

func isSageMakerGeospatialCandidate(r *http.Request) bool {
	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service == sgeoSigningName {
		return true
	}
	return service == "" && strings.HasPrefix(strings.ToLower(strings.TrimSpace(r.Host)), sgeoSigningName+".")
}

var sgeoStatus = map[string]int{
	"ValidationException": http.StatusBadRequest, "ResourceNotFoundException": http.StatusNotFound,
	"ConflictException": http.StatusConflict, "ServiceQuotaExceededException": http.StatusPaymentRequired,
	"AccessDeniedException": http.StatusForbidden, "InternalServerException": http.StatusInternalServerError,
}

func sgeoInvalid(format string, args ...any) *fcErr {
	return fcError("ValidationException", format, args...)
}

func sgeoConflict(format string, args ...any) *fcErr {
	return fcError("ConflictException", format, args...)
}

func (s *Server) handleSageMakerGeospatialRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isSageMakerGeospatialCandidate(r) {
		return false
	}
	op, in, ferr := s.sgeoDecode(w, r)
	if ferr == nil && in == nil {
		return true // signature failure already written
	}
	if ferr != nil {
		respondSGeoError(w, ferr)
		return true
	}
	if op == "GetTile" {
		seed, x, y, z, ferr := s.sageMakerGeospatial.tileSeed(in)
		if ferr != nil {
			respondSGeoError(w, ferr)
			return true
		}
		tile := sgeoTile(seed, x, y, z) // rendered outside the store lock
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(tile)
		return true
	}
	out, ferr := s.sgeoRun(op, s.withS3Caller(in, r))
	if ferr != nil {
		respondSGeoError(w, ferr)
		return true
	}
	body, _ := json.Marshal(out)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
	return true
}

// sgeoDecode matches the route, checks length and signature, and builds the
// input from the JSON body (validated and conformed against the input shape)
// plus URI labels and query parameters converted to their member shapes. A
// nil input with a nil error means the signature failure has been written.
func (s *Server) sgeoDecode(w http.ResponseWriter, r *http.Request) (string, map[string]any, *fcErr) {
	raw := splitPathSegments(rawRequestPath(r))
	if len(raw) == 0 || len(raw) > nmMaxPathSegments {
		return "", nil, fcError("UnknownOperationException", "no SageMaker geospatial operation matches %s %s", r.Method, rawRequestPath(r))
	}
	path := make([]string, len(raw))
	for i, v := range raw {
		u, err := url.PathUnescape(v)
		if err != nil {
			return "", nil, sgeoInvalid("invalid path segment")
		}
		path[i] = u
	}
	op, labels, ok := sgeoMatch(r.Method, path)
	if !ok {
		return "", nil, fcError("UnknownOperationException", "no SageMaker geospatial operation matches %s %s", r.Method, rawRequestPath(r))
	}
	// A declared length bounds what is read; refuse chunked bodies.
	if r.ContentLength < 0 || r.ContentLength > sgeoMaxBody {
		return "", nil, sgeoInvalid("request body must declare a Content-Length of at most %d bytes", sgeoMaxBody)
	}
	if ok, status, code, msg, _ := s.validateSigV4WithService(r, sgeoSigningName); !ok {
		respondSGeoError(w, &fcErr{code: code, msg: msg}, status)
		return "", nil, nil
	}
	m := awsmodels.SageMakerGeospatialModel()
	shapeName := m.Operations[op].Input
	in := map[string]any{}
	body, err := io.ReadAll(io.LimitReader(r.Body, sgeoMaxBody+1))
	if err != nil || len(body) > sgeoMaxBody {
		return "", nil, sgeoInvalid("unable to read request body")
	}
	if len(bytes.TrimSpace(body)) > 0 {
		if err := json.Unmarshal(body, &in); err != nil {
			return "", nil, sgeoInvalid("request body is not a JSON object")
		}
	}
	if problems := m.Validate(shapeName, in); len(problems) > 0 {
		return "", nil, sgeoInvalid("%s", strings.Join(problems, "; "))
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
			return "", nil, sgeoInvalid("%s %s", member, err.Error())
		}
		if problems := m.Validate(ref.Shape, conv); len(problems) > 0 {
			_, reason, _ := strings.Cut(problems[0], ": ")
			return "", nil, sgeoInvalid("%s %s", member, reason)
		}
		in[member] = conv
	}
	for _, req := range shape.Required {
		if _, ok := in[req]; !ok {
			return "", nil, sgeoInvalid("missing required member %s", req)
		}
	}
	return op, in, nil
}

func respondSGeoError(w http.ResponseWriter, e *fcErr, status ...int) {
	code := http.StatusBadRequest
	if len(status) > 0 {
		code = status[0]
	} else if c, ok := sgeoStatus[e.code]; ok {
		code = c
	}
	body, _ := json.Marshal(map[string]string{"Message": e.msg})
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Amzn-ErrorType", e.code)
	w.WriteHeader(code)
	_, _ = w.Write(body)
}

// sgeoTile renders a deterministic 256x256 grayscale PNG for one tile.
// ponytail: placeholder imagery — the value pattern depends only on the job,
// assets and tile coordinates.
func sgeoTile(seed string, x, y, z int) []byte {
	img := image.NewGray(image.Rect(0, 0, sgeoTileSize, sgeoTileSize))
	base := uint8(amlScore(fmt.Sprintf("%s|%d|%d|%d", seed, z, x, y)) * 255)
	for py := 0; py < sgeoTileSize; py++ {
		for px := 0; px < sgeoTileSize; px++ {
			img.SetGray(px, py, color.Gray{Y: base + uint8((px+py)/4)})
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}
