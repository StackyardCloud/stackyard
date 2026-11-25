package server

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/stackyard/stackyard/internal/awsmodels"
)

// AWS Budgets: awsJson1_1, X-Amz-Target AWSBudgetServiceGateway.*, signed
// budgets. The JSON plumbing here (budJSONAPI, budServe, budPage, budClone) is
// shared with Cost Optimization Hub and BCM Data Exports.

const (
	budTargetPrefix = "AWSBudgetServiceGateway."
	// ponytail: 64 KiB covers the largest real budget (filter expressions
	// included) and, with the per-store counts, bounds stored bytes.
	budMaxBody = 64 << 10
)

type budErr struct {
	status    int
	code, msg string
	extra     map[string]any // model-specific error members (reason, errorCode)
}

func budErrorf(code, format string, args ...any) *budErr {
	status := http.StatusBadRequest
	switch code {
	case "ResourceNotFoundException":
		status = http.StatusNotFound
	case "AccessDeniedException":
		status = http.StatusForbidden
	case "ConflictException":
		status = http.StatusConflict
	}
	return &budErr{status: status, code: code, msg: fmt.Sprintf(format, args...)}
}

func budInvalid(format string, args ...any) *budErr {
	return budErrorf("InvalidParameterException", format, args...)
}

func budNotFound(format string, args ...any) *budErr {
	return budErrorf("NotFoundException", format, args...)
}

type budHandler func(s *Server, in map[string]any) (map[string]any, *budErr)

// budJSONAPI describes one awsJson service served by budServe.
type budJSONAPI struct {
	prefix, signing, contentType, invalidCode string
	model                                     func() *awsmodels.Model
	handlers                                  map[string]budHandler
	maxBody                                   int64 // 0: budMaxBody
	// respondError writes errors in the service's own exception shape
	// (nil: budRespondError).
	respondError func(http.ResponseWriter, *budErr)
	// s3Caller passes the request's caller to handlers (s3CallerIn) that
	// touch S3 on the caller's behalf.
	s3Caller bool
}

func (api *budJSONAPI) fail(w http.ResponseWriter, e *budErr) {
	if api.respondError != nil {
		api.respondError(w, e)
		return
	}
	budRespondError(w, api, e)
}

var budAPI = &budJSONAPI{
	prefix: budTargetPrefix, signing: "budgets", contentType: "application/x-amz-json-1.1",
	invalidCode: "InvalidParameterException", model: awsmodels.BudgetsModel,
}

func init() {
	budAPI.handlers = map[string]budHandler{
		"CreateBudget":                          (*Server).budCreateBudget,
		"DescribeBudget":                        (*Server).budDescribeBudget,
		"DescribeBudgets":                       (*Server).budDescribeBudgets,
		"UpdateBudget":                          (*Server).budUpdateBudget,
		"DeleteBudget":                          (*Server).budDeleteBudget,
		"DescribeBudgetPerformanceHistory":      (*Server).budPerformanceHistory,
		"CreateNotification":                    (*Server).budCreateNotification,
		"UpdateNotification":                    (*Server).budUpdateNotification,
		"DeleteNotification":                    (*Server).budDeleteNotification,
		"DescribeNotificationsForBudget":        (*Server).budNotificationsForBudget,
		"DescribeBudgetNotificationsForAccount": (*Server).budNotificationsForAccount,
		"CreateSubscriber":                      (*Server).budCreateSubscriber,
		"UpdateSubscriber":                      (*Server).budUpdateSubscriber,
		"DeleteSubscriber":                      (*Server).budDeleteSubscriber,
		"DescribeSubscribersForNotification":    (*Server).budSubscribersForNotification,
		"CreateBudgetAction":                    (*Server).budCreateAction,
		"DescribeBudgetAction":                  (*Server).budDescribeAction,
		"UpdateBudgetAction":                    (*Server).budUpdateAction,
		"DeleteBudgetAction":                    (*Server).budDeleteAction,
		"ExecuteBudgetAction":                   (*Server).budExecuteAction,
		"DescribeBudgetActionsForBudget":        (*Server).budActionsForBudget,
		"DescribeBudgetActionsForAccount":       (*Server).budActionsForAccount,
		"DescribeBudgetActionHistories":         (*Server).budActionHistories,
		"TagResource":                           (*Server).budTagResource,
		"UntagResource":                         (*Server).budUntagResource,
		"ListTagsForResource":                   (*Server).budListTags,
	}
}

func (s *Server) handleBudgetsJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	return s.budServe(w, r, budAPI, budOperationByName)
}

// budServe claims POSTs whose X-Amz-Target carries api.prefix, checks the
// body length and the signature, validates and conforms the input against the
// CLI model, and writes the handler's output conformed to the output shape.
func (s *Server) budServe(w http.ResponseWriter, r *http.Request, api *budJSONAPI, known map[string]bool) bool {
	if r.Method != http.MethodPost {
		return false
	}
	op, ok := strings.CutPrefix(strings.TrimSpace(r.Header.Get("X-Amz-Target")), api.prefix)
	if !ok {
		return false
	}
	s.budRun(w, r, api, op, known[op])
	return true
}

// budRun serves one claimed request for op: REST JSON routers that resolve op
// from the path (Account, Tax Settings) share it with budServe.
func (s *Server) budRun(w http.ResponseWriter, r *http.Request, api *budJSONAPI, op string, known bool) {
	// A declared length bounds what net/http reads; refuse chunked bodies.
	limit := api.maxBody
	if limit == 0 {
		limit = budMaxBody
	}
	if r.ContentLength < 0 || r.ContentLength > limit {
		api.fail(w, budErrorf(api.invalidCode, "request body must declare a Content-Length of at most %d bytes", limit))
		return
	}
	if ok, status, code, msg, _ := s.validateSigV4WithService(r, api.signing); !ok {
		api.fail(w, &budErr{status: status, code: code, msg: msg})
		return
	}
	h := api.handlers[op]
	if !known || h == nil {
		api.fail(w, budErrorf("UnknownOperationException", "unknown operation %s", op))
		return
	}
	raw, err := readBodyBytes(r)
	in := map[string]any{}
	if err == nil && strings.TrimSpace(string(raw)) != "" {
		err = json.Unmarshal(raw, &in)
	}
	if err != nil {
		api.fail(w, budErrorf("SerializationException", "invalid JSON body"))
		return
	}
	model := api.model()
	shapes := model.Operations[op]
	if problems := model.Validate(shapes.Input, in); len(problems) > 0 {
		api.fail(w, budErrorf(api.invalidCode, "%s", strings.Join(problems, "; ")))
		return
	}
	// Conform drops members the input shape doesn't define, so callers can't
	// reach the store's own fields.
	conformed, _ := model.Conform(shapes.Input, in).(map[string]any)
	if api.s3Caller {
		conformed = s.withS3Caller(conformed, r)
	}
	out, berr := h(s, conformed)
	if berr != nil {
		api.fail(w, berr)
		return
	}
	// Handlers return values built (or cloned) under their store lock, so
	// nothing here is shared with stored records.
	for _, k := range []string{"NextToken", "nextToken"} {
		if out[k] == "" {
			delete(out, k) // last page
		}
	}
	body, _ := json.Marshal(model.Conform(shapes.Output, out))
	w.Header().Set("Content-Type", api.contentType)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}

func budRespondError(w http.ResponseWriter, api *budJSONAPI, e *budErr) {
	body, _ := json.Marshal(map[string]string{"__type": e.code, "Message": e.msg})
	w.Header().Set("Content-Type", api.contentType)
	w.Header().Set("X-Amzn-ErrorType", e.code)
	w.WriteHeader(e.status)
	_, _ = w.Write(body)
}

// ---- shared request helpers ----

func budStr(in map[string]any, key string) string {
	v, _ := in[key].(string)
	return v
}

func budMap(in map[string]any, key string) map[string]any {
	m, _ := in[key].(map[string]any)
	return m
}

func budList(in map[string]any, key string) []any {
	l, _ := in[key].([]any)
	return l
}

func budStrings(in map[string]any, key string) []string {
	var out []string
	for _, v := range budList(in, key) {
		if s, ok := v.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

// budInt reads an optional integer member; the model has already checked its range.
func budInt(in map[string]any, key string, def int) int {
	if f, ok := in[key].(float64); ok && f > 0 {
		return int(f)
	}
	return def
}

func budBool(in map[string]any, key string) bool {
	b, _ := in[key].(bool)
	return b
}

// budTime reads a timestamp sent as epoch seconds or an ISO 8601 string.
func budTime(v any) (time.Time, bool) {
	switch t := v.(type) {
	case float64:
		if math.IsNaN(t) || math.IsInf(t, 0) || t < 0 || t > 1e11 {
			return time.Time{}, false
		}
		sec, frac := math.Modf(t)
		return time.Unix(int64(sec), int64(frac*1e9)).UTC(), true
	case string:
		for _, layout := range []string{time.RFC3339Nano, "2006-01-02T15:04:05", "2006-01-02"} {
			if p, err := time.Parse(layout, t); err == nil {
				return p.UTC(), true
			}
		}
		if f, err := strconv.ParseFloat(t, 64); err == nil {
			return budTime(f)
		}
	}
	return time.Time{}, false
}

func budEpoch(t time.Time) float64 { return float64(t.UnixMilli()) / 1000 }

// budClone deep-copies a decoded JSON value so responses never share maps or
// slices with stored records.
func budClone(v any) any {
	switch t := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, item := range t {
			out[k] = budClone(item)
		}
		return out
	case []any:
		out := make([]any, len(t))
		for i, item := range t {
			out[i] = budClone(item)
		}
		return out
	}
	return v
}

func budCloneMap(m map[string]any) map[string]any {
	out, _ := budClone(m).(map[string]any)
	return out
}

// ---- pagination ----

// budPage returns the [start, end) window over total items and the next token.
// Tokens are opaque offsets; a token past the end is refused with invalidCode.
func budPage(token string, total, size int, invalidCode string) (int, int, string, *budErr) {
	start := 0
	if token != "" {
		raw, err := base64.RawURLEncoding.DecodeString(token)
		rest, hasPrefix := strings.CutPrefix(string(raw), "o:")
		n, perr := strconv.ParseInt(rest, 10, 32)
		if err != nil || perr != nil || !hasPrefix || n < 0 || int(n) > total {
			return 0, 0, "", budErrorf(invalidCode, "The NextToken is not valid.")
		}
		start = int(n)
	}
	end := min(total, start+size)
	next := ""
	if end < total {
		next = base64.RawURLEncoding.EncodeToString([]byte("o:" + strconv.Itoa(end)))
	}
	return start, end, next, nil
}

// ---- tags ----

// budTagsIn reads a [{Key, Value}] list, refusing the reserved aws: prefix.
func budTagsIn(list []any, invalidCode string) (map[string]string, *budErr) {
	out := map[string]string{}
	for _, item := range list {
		m, _ := item.(map[string]any)
		k, v := budStr(m, "Key"), budStr(m, "Value")
		if strings.HasPrefix(strings.ToLower(k), "aws:") {
			return nil, budErrorf(invalidCode, "tag keys starting with aws: are reserved")
		}
		out[k] = v
	}
	return out, nil
}

func budTagsOut(tags map[string]string) []any {
	keys := make([]string, 0, len(tags))
	for k := range tags {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, k := range keys {
		out = append(out, map[string]any{"Key": k, "Value": tags[k]})
	}
	return out
}

// budMergeTags adds tags to a copy of current, refusing more than limit keys.
func budMergeTags(current, add map[string]string, limit int, code string) (map[string]string, *budErr) {
	out := make(map[string]string, len(current)+len(add))
	for k, v := range current {
		out[k] = v
	}
	for k, v := range add {
		out[k] = v
	}
	if len(out) > limit {
		return nil, budErrorf(code, "a resource can have at most %d tags", limit)
	}
	return out, nil
}

func budDropTags(current map[string]string, keys []string) map[string]string {
	out := make(map[string]string, len(current))
	for k, v := range current {
		out[k] = v
	}
	for _, k := range keys {
		delete(out, k)
	}
	return out
}

func budNewID() string { return uuid.NewString() }
