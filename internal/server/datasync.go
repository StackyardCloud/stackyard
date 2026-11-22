package server

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

// AWS DataSync: JSON 1.1 under X-Amz-Target FmrsService.*, signed as datasync.
// Requests are validated and responses shaped against the embedded botocore
// model (internal/awsmodels/datasync/model.json); handlers hold the store
// lock, and responses are shaped before it is released.

const (
	dsyTargetPrefix = "FmrsService."
	dsyRegion       = "us-east-1"
	dsyAccountID    = "123456789012"
	dsyMaxBody      = 1 << 20
	dsyMaxTags      = 50
	dsyMaxResults   = 100
)

type dsyHandler func(s *Server, in map[string]any) (map[string]any, *dsyErr)

// dsyHandlers is filled by the per-area files' init functions.
var dsyHandlers = map[string]dsyHandler{}

type dsyErr struct {
	status int
	code   string
	msg    string
}

func dsyInvalid(format string, args ...any) *dsyErr {
	return &dsyErr{status: http.StatusBadRequest, code: "InvalidRequestException", msg: fmt.Sprintf(format, args...)}
}

func dsyNotFound(kind, arn string) *dsyErr {
	return dsyInvalid("%s %s does not exist.", kind, arn)
}

func (s *Server) handleDataSyncJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	target := strings.TrimSpace(r.Header.Get("X-Amz-Target"))
	if r.Method != http.MethodPost || !strings.HasPrefix(target, dsyTargetPrefix) {
		return false
	}
	// A declared length bounds the body (net/http stops reading there), so
	// require one; chunked bodies of unknown length are refused.
	if r.ContentLength < 0 || r.ContentLength > dsyMaxBody {
		respondDataSyncError(w, dsyInvalid("request body must declare a Content-Length of at most 1 MiB"))
		return true
	}
	if ok, status, code, msg, _ := s.validateSigV4WithService(r, "datasync"); !ok {
		respondDataSyncError(w, &dsyErr{status: status, code: code, msg: msg})
		return true
	}
	op := strings.TrimPrefix(target, dsyTargetPrefix)
	model, known := awsmodels.DataSyncModel().Operations[op]
	handler := dsyHandlers[op]
	if _, listed := datasyncOperationByName[op]; !known || !listed || handler == nil {
		respondDataSyncError(w, &dsyErr{status: http.StatusBadRequest, code: "UnknownOperationException", msg: "unknown DataSync operation " + op})
		return true
	}
	body, err := readBodyBytes(r)
	if err != nil {
		respondDataSyncError(w, dsyInvalid("unable to read request body"))
		return true
	}
	in := map[string]any{}
	if len(strings.TrimSpace(string(body))) > 0 {
		if err := json.Unmarshal(body, &in); err != nil {
			respondDataSyncError(w, dsyInvalid("request body is not a JSON object"))
			return true
		}
	}
	dm := awsmodels.DataSyncModel()
	if msg := modelValidate(dm, model.Input, in, ""); msg != "" {
		respondDataSyncError(w, dsyInvalid("%s", msg))
		return true
	}
	// Conforming the input keeps model members only, so callers can't reach
	// fields the store keeps for itself.
	in, _ = modelConform(dm, model.Input, in).(map[string]any)
	out, derr := s.datasync.handle(s, handler, model.Output, in)
	if derr != nil {
		respondDataSyncError(w, derr)
		return true
	}
	payload, _ := json.Marshal(out)
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(payload)
	return true
}

// handle runs a handler under the store lock and shapes its output there too,
// since outputs can alias stored records.
func (st *dsyStore) handle(s *Server, h dsyHandler, outShape string, in map[string]any) (any, *dsyErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	out, err := h(s, in)
	if err != nil {
		return nil, err
	}
	if outShape == "" {
		return map[string]any{}, nil
	}
	return modelConform(awsmodels.DataSyncModel(), outShape, out), nil
}

func respondDataSyncError(w http.ResponseWriter, e *dsyErr) {
	payload, _ := json.Marshal(map[string]string{"__type": e.code, "message": e.msg})
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.Header().Set("X-Amzn-ErrorType", e.code)
	w.WriteHeader(e.status)
	_, _ = w.Write(payload)
}

// --- shared helpers -------------------------------------------------------

func dsyID(prefix string) string {
	b := make([]byte, 9)
	_, _ = rand.Read(b)
	return prefix + hex.EncodeToString(b)[:17]
}

func dsyARN(resource string) string {
	return fmt.Sprintf("arn:aws:datasync:%s:%s:%s", dsyRegion, dsyAccountID, resource)
}

func dsyStr(in map[string]any, key string) string {
	v, _ := in[key].(string)
	return v
}

func dsyMap(in map[string]any, key string) map[string]any {
	v, _ := in[key].(map[string]any)
	return v
}

func dsyStrings(in map[string]any, key string) []string {
	list, _ := in[key].([]any)
	out := make([]string, 0, len(list))
	for _, v := range list {
		if s, ok := v.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func dsyCopyMap(in map[string]any) map[string]any {
	b, _ := json.Marshal(in)
	out := map[string]any{}
	_ = json.Unmarshal(b, &out)
	return out
}

// dsyTagsFrom validates a Tags list into a key/value map.
func dsyTagsFrom(in map[string]any, key string, into map[string]string) *dsyErr {
	list, _ := in[key].([]any)
	for _, raw := range list {
		tag, _ := raw.(map[string]any)
		k := dsyStr(tag, "Key")
		if strings.HasPrefix(strings.ToLower(k), "aws:") {
			return dsyInvalid("tag keys can't start with aws:")
		}
		into[k] = dsyStr(tag, "Value")
	}
	if len(into) > dsyMaxTags {
		return dsyInvalid("a resource can have at most %d tags", dsyMaxTags)
	}
	return nil
}

func dsyTagList(tags map[string]string) []any {
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

// dsyPage pages n items (already in order) by MaxResults/NextToken; the token
// is the decimal offset of the next item.
func dsyPage(in map[string]any, n int) (start, end int, next string, err *dsyErr) {
	limit := dsyMaxResults
	if v, ok := in["MaxResults"].(float64); ok && v > 0 && v < dsyMaxResults {
		limit = int(v)
	}
	if tok := dsyStr(in, "NextToken"); tok != "" {
		off, perr := strconv.ParseInt(tok, 10, 32)
		if perr != nil || off < 0 || int(off) > n {
			return 0, 0, "", dsyInvalid("NextToken %q is not valid", tok)
		}
		start = int(off)
	}
	end = min(start+limit, n)
	if end < n {
		next = strconv.Itoa(end)
	}
	return start, end, next, nil
}

// dsyFilterMatch applies a ListLocations/ListTasks filter operator to one value.
func dsyFilterMatch(op, value string, values []string) bool {
	num := func(s string) (float64, bool) {
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return f, true
		}
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			return float64(t.Unix()), true
		}
		return 0, false
	}
	for _, want := range values {
		switch op {
		case "Equals", "In":
			if value == want {
				return true
			}
		case "NotEquals":
			if value != want {
				return true
			}
		case "Contains":
			if strings.Contains(value, want) {
				return true
			}
		case "NotContains":
			if !strings.Contains(value, want) {
				return true
			}
		case "BeginsWith":
			if strings.HasPrefix(value, want) {
				return true
			}
		case "LessThan", "LessThanOrEqual", "GreaterThan", "GreaterThanOrEqual":
			a, ok1 := num(value)
			b, ok2 := num(want)
			if ok1 && ok2 && ((op == "LessThan" && a < b) || (op == "LessThanOrEqual" && a <= b) ||
				(op == "GreaterThan" && a > b) || (op == "GreaterThanOrEqual" && a >= b)) {
				return true
			}
		}
	}
	return false
}
