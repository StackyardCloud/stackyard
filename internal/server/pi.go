package server

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/stackyard/stackyard/internal/awsmodels"
	rdssvc "github.com/stackyard/stackyard/internal/services/rds"
)

// Amazon RDS Performance Insights: awsJson1_1, X-Amz-Target
// PerformanceInsightsv20180227.*, signed pi. The CLI 2.41 model lists
// smithy-rpc-v2-cbor first, but the CLI sends JSON 1.1.

const (
	piTargetPrefix = "PerformanceInsightsv20180227."
	piMaxBody      = 1 << 20
	piAccountID    = "123456789012"
	piRegion       = "us-east-1"
)

type piErr struct{ code, msg string }

func piInvalid(format string, args ...any) *piErr {
	return &piErr{code: "InvalidArgumentException", msg: fmt.Sprintf(format, args...)}
}

func piNotAuthorized(format string, args ...any) *piErr {
	return &piErr{code: "NotAuthorizedException", msg: fmt.Sprintf(format, args...)}
}

type piHandler func(s *Server, in map[string]any) (map[string]any, *piErr)

var piHandlers map[string]piHandler

func init() {
	piHandlers = map[string]piHandler{
		"DescribeDimensionKeys":                        (*Server).piDescribeDimensionKeys,
		"GetResourceMetrics":                           (*Server).piGetResourceMetrics,
		"GetDimensionKeyDetails":                       (*Server).piGetDimensionKeyDetails,
		"GetResourceMetadata":                          (*Server).piGetResourceMetadata,
		"ListAvailableResourceDimensions":              (*Server).piListAvailableResourceDimensions,
		"ListAvailableResourceMetrics":                 (*Server).piListAvailableResourceMetrics,
		"CreatePerformanceAnalysisReport":              (*Server).piCreateReport,
		"GetPerformanceAnalysisReport":                 (*Server).piGetReport,
		"ListPerformanceAnalysisReports":               (*Server).piListReports,
		"ListPerformanceAnalysisReportRecommendations": (*Server).piListRecommendations,
		"DeletePerformanceAnalysisReport":              (*Server).piDeleteReport,
		"TagResource":                                  func(s *Server, in map[string]any) (map[string]any, *piErr) { return s.pi.tagResource(in) },
		"UntagResource":                                func(s *Server, in map[string]any) (map[string]any, *piErr) { return s.pi.untagResource(in) },
		"ListTagsForResource":                          func(s *Server, in map[string]any) (map[string]any, *piErr) { return s.pi.listTags(in) },
	}
}

func (s *Server) handlePIJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}
	op, ok := strings.CutPrefix(strings.TrimSpace(r.Header.Get("X-Amz-Target")), piTargetPrefix)
	if !ok {
		return false
	}
	// A declared length bounds what net/http reads; refuse chunked bodies.
	if r.ContentLength < 0 || r.ContentLength > piMaxBody {
		respondPIError(w, http.StatusBadRequest, "InvalidArgumentException", "request body must declare a Content-Length of at most 1 MiB")
		return true
	}
	if ok, status, code, msg, _ := s.validateSigV4WithService(r, "pi"); !ok {
		respondPIError(w, status, code, msg)
		return true
	}
	h := piHandlers[op]
	if !piOperationByName[op] || h == nil {
		respondPIError(w, http.StatusBadRequest, "UnknownOperationException", "unknown operation "+op)
		return true
	}
	raw, err := readBodyBytes(r)
	in := map[string]any{}
	if err == nil && strings.TrimSpace(string(raw)) != "" {
		err = json.Unmarshal(raw, &in)
	}
	if err != nil {
		respondPIError(w, http.StatusBadRequest, "SerializationException", "invalid JSON body")
		return true
	}
	model := awsmodels.PIModel()
	shapes := model.Operations[op]
	if problems := model.Validate(shapes.Input, in); len(problems) > 0 {
		respondPIError(w, http.StatusBadRequest, "InvalidArgumentException", strings.Join(problems, "; "))
		return true
	}
	// Conform drops members the input shape doesn't define, so callers can't
	// reach the store's own fields.
	conformed, _ := model.Conform(shapes.Input, in).(map[string]any)
	out, perr := h(s, conformed)
	if perr != nil {
		respondPIError(w, http.StatusBadRequest, perr.code, perr.msg)
		return true
	}
	// out is built fresh by every handler; nothing in it is shared with the store.
	body, _ := json.Marshal(model.Conform(shapes.Output, out))
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
	return true
}

func respondPIError(w http.ResponseWriter, status int, code, msg string) {
	body, _ := json.Marshal(map[string]string{"__type": code, "Message": msg})
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.Header().Set("X-Amzn-ErrorType", code)
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

// ---- request helpers ----

func piStr(in map[string]any, key string) string {
	v, _ := in[key].(string)
	return v
}

func piList(in map[string]any, key string) []string {
	var out []string
	for _, v := range piAnyList(in[key]) {
		if s, ok := v.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func piAnyList(v any) []any {
	l, _ := v.([]any)
	return l
}

func piMap(in map[string]any, key string) map[string]any {
	m, _ := in[key].(map[string]any)
	return m
}

// piInt reads an optional integer member; the model has already checked its range.
func piInt(in map[string]any, key string, def int) int {
	if f, ok := in[key].(float64); ok {
		return int(f)
	}
	return def
}

// piTime reads a timestamp sent as epoch seconds or an ISO 8601 string.
func piTime(in map[string]any, key string) (time.Time, bool, *piErr) {
	switch v := in[key].(type) {
	case nil:
		return time.Time{}, false, nil
	case float64:
		if math.IsNaN(v) || math.IsInf(v, 0) || v < 0 || v > 1e11 {
			return time.Time{}, false, piInvalid("%s is out of range", key)
		}
		sec, frac := math.Modf(v)
		return time.Unix(int64(sec), int64(frac*1e9)).UTC(), true, nil
	case string:
		for _, layout := range []string{time.RFC3339Nano, "2006-01-02T15:04:05", "2006-01-02"} {
			if t, err := time.Parse(layout, v); err == nil {
				return t.UTC(), true, nil
			}
		}
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return piTime(map[string]any{key: f}, key)
		}
	}
	return time.Time{}, false, piInvalid("%s is not a valid timestamp", key)
}

func piEpoch(t time.Time) float64 { return float64(t.UnixMilli()) / 1000 }

// piResolve checks ServiceType and Identifier against the RDS store: the
// identifier is an instance's DbiResourceId, and DOCDB instances are the ones
// with the docdb engine. Unknown identifiers get NotAuthorizedException, as AWS
// reports for resources the caller can't see.
// ponytail: the RDS store doesn't record whether Performance Insights is
// enabled, so every instance is treated as enabled.
func (s *Server) piResolve(in map[string]any) (string, string, *piErr) {
	serviceType, id := piStr(in, "ServiceType"), piStr(in, "Identifier")
	inst, ok := s.rds.InstanceByResourceID(id)
	if !ok || (serviceType == "DOCDB") != (inst.Engine == "docdb") {
		return "", "", piNotAuthorized("The user is not authorized to perform this request on %s %s.", serviceType, id)
	}
	return serviceType, rdssvc.ResourceID(inst.Identifier), nil
}

// ---- pagination ----

func piToken(offset int) string {
	return base64.RawURLEncoding.EncodeToString([]byte("o:" + strconv.Itoa(offset)))
}

func piOffset(in map[string]any, total int) (int, *piErr) {
	tok := piStr(in, "NextToken")
	if tok == "" {
		return 0, nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(tok)
	n, perr := strconv.ParseInt(strings.TrimPrefix(string(raw), "o:"), 10, 32)
	if err != nil || perr != nil || !strings.HasPrefix(string(raw), "o:") || n < 0 || int(n) > total {
		return 0, piInvalid("The NextToken is not valid.")
	}
	return int(n), nil
}

// piPage returns the [start, end) window and the next token.
func piPage(in map[string]any, total, def int) (int, int, string, *piErr) {
	start, err := piOffset(in, total)
	if err != nil {
		return 0, 0, "", err
	}
	size := piInt(in, "MaxResults", def)
	if size <= 0 {
		size = def
	}
	end := min(total, start+size)
	next := ""
	if end < total {
		next = piToken(end)
	}
	return start, end, next, nil
}
