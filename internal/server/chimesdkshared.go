package server

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

// Shared plumbing for the Amazon Chime SDK Meetings, Messaging and Media
// Pipelines REST JSON APIs. All three sign as chime (like the legacy Chime
// API in chime.go) and share /tags, so each router claims only its own
// operation templates, and /tags requests by the resource ARN they name.

const (
	chimeSDKRegion  = "us-east-1"
	chimeSDKAccount = "123456789012"
	chimeSDKMaxBody = 256 << 10
)

type chimeSDKErr struct {
	status    int
	code, msg string
}

func chimeSDKError(status int, code, format string, args ...any) *chimeSDKErr {
	return &chimeSDKErr{status: status, code: code, msg: fmt.Sprintf(format, args...)}
}

func chimeSDKBadRequest(format string, args ...any) *chimeSDKErr {
	return chimeSDKError(http.StatusBadRequest, "BadRequestException", format, args...)
}

func chimeSDKNotFound(format string, args ...any) *chimeSDKErr {
	return chimeSDKError(http.StatusNotFound, "NotFoundException", format, args...)
}

func chimeSDKConflict(format string, args ...any) *chimeSDKErr {
	return chimeSDKError(http.StatusConflict, "ConflictException", format, args...)
}

func chimeSDKForbidden(format string, args ...any) *chimeSDKErr {
	return chimeSDKError(http.StatusForbidden, "ForbiddenException", format, args...)
}

// chimeSDKLimit is the Messaging and Media Pipelines quota error.
func chimeSDKLimit(format string, args ...any) *chimeSDKErr {
	return chimeSDKError(http.StatusBadRequest, "ResourceLimitExceededException", format, args...)
}

// chimeSDKErrorCodes maps exceptions to the ErrorCode enum in their Code member.
var chimeSDKErrorCodes = map[string]string{
	"BadRequestException": "BadRequest", "ConflictException": "Conflict", "ForbiddenException": "Forbidden",
	"NotFoundException": "NotFound", "ResourceNotFoundException": "NotFound", "LimitExceededException": "ResourceLimitExceeded",
	"ResourceLimitExceededException": "ResourceLimitExceeded", "TooManyTagsException": "ResourceLimitExceeded",
	"UnauthorizedClientException": "Unauthorized", "ServiceFailureException": "ServiceFailure",
}

func chimeSDKWriteError(w http.ResponseWriter, e *chimeSDKErr) {
	body, _ := json.Marshal(map[string]string{"Code": chimeSDKErrorCodes[e.code], "Message": e.msg})
	w.Header().Set("X-Amzn-ErrorType", e.code)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(e.status)
	_, _ = w.Write(body)
}

// chimeSDKMatch returns the operation in ops that r addresses when r is
// signed for chime (or unsigned to a host starting with hostPrefix).
func chimeSDKMatch(r *http.Request, hostPrefix string, ops []chimeOperation) (chimeOperation, map[string]string, bool) {
	hint := sigV4ServiceHint(r)
	if hint != "chime" && (hint != "" || !strings.HasPrefix(strings.ToLower(r.Host), hostPrefix)) {
		return chimeOperation{}, nil, false
	}
	op, params := chimeRouteIn(ops, r.Method, rawRequestPath(r), r.URL.Query())
	return op, params, op.Name != ""
}

// chimeSDKDecode authenticates r and decodes its body, URI and query members
// against the operation's input shape. On failure it writes the error and
// returns false.
func (s *Server) chimeSDKDecode(w http.ResponseWriter, r *http.Request, model *awsmodels.Model, op chimeOperation, params map[string]string) (map[string]any, bool) {
	// A declared length bounds what net/http reads; refuse chunked bodies.
	if r.ContentLength < 0 || r.ContentLength > chimeSDKMaxBody {
		chimeSDKWriteError(w, chimeSDKBadRequest("Request body must declare a Content-Length of at most %d bytes.", chimeSDKMaxBody))
		return nil, false
	}
	if ok, status, code, msg, _ := s.validateSigV4WithService(r, "chime"); !ok {
		chimeSDKWriteError(w, chimeSDKError(status, code, "%s", msg))
		return nil, false
	}
	in, problem := awsModelDecodeREST(r, model, model.Operations[op.Name].Input, params)
	if problem != "" {
		chimeSDKWriteError(w, chimeSDKBadRequest("%s", problem))
		return nil, false
	}
	return in, true
}

// chimeSDKTagOwner names the API that owns the resource a /tags request
// addresses: "messaging" for channels and channel flows, "pipelines" for
// media pipeline resources and "meetings" otherwise.
func chimeSDKTagOwner(r *http.Request) string {
	arn := r.URL.Query().Get("arn")
	if r.Method == http.MethodPost && r.ContentLength > 0 && r.ContentLength <= chimeSDKMaxBody {
		var body struct {
			ResourceARN string
		}
		if raw, err := readBodyBytes(r); err == nil && json.Unmarshal(raw, &body) == nil {
			arn = body.ResourceARN
		}
	}
	switch {
	case strings.Contains(arn, "/channel/") || strings.Contains(arn, "/channel-flow/"):
		return "messaging"
	case strings.Contains(arn, ":media-"):
		return "pipelines"
	}
	return "meetings"
}

func chimeSDKIsTagOp(op chimeOperation) bool { return strings.HasPrefix(op.URI, "/tags") }

// ---- tags ----

func chimeSDKTagList(tags map[string]string) []any {
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

// chimeSDKMergeTags returns a copy of cur with the request's Tags applied,
// or an error when a key is reserved or the result exceeds max tags.
func chimeSDKMergeTags(cur map[string]string, in []any, max int, tooMany func() *chimeSDKErr) (map[string]string, *chimeSDKErr) {
	out := make(map[string]string, len(cur)+len(in))
	for k, v := range cur {
		out[k] = v
	}
	for _, item := range in {
		t, _ := item.(map[string]any)
		key := mmgrStr(t, "Key")
		if strings.HasPrefix(strings.ToLower(key), "aws:") {
			return nil, chimeSDKBadRequest("Tag keys can't start with aws:.")
		}
		out[key] = mmgrStr(t, "Value")
	}
	if len(out) > max {
		return nil, tooMany()
	}
	return out, nil
}

func chimeSDKUntag(cur map[string]string, keys []any) map[string]string {
	out := make(map[string]string, len(cur))
	for k, v := range cur {
		out[k] = v
	}
	for _, k := range keys {
		if key, ok := k.(string); ok {
			delete(out, key)
		}
	}
	return out
}

// ---- helpers ----

// chimeSDKPage sorts keys and returns the page after token (the last key of
// the previous page) holding at most max keys (def when max is unset), and
// the token for the next page ("" on the last page).
func chimeSDKPage(keys []string, token string, max, def int) ([]string, string) {
	sort.Strings(keys)
	start := 0
	if token != "" {
		start = sort.SearchStrings(keys, token)
		if start < len(keys) && keys[start] == token {
			start++
		}
	}
	if max <= 0 {
		max = def
	}
	end := min(start+max, len(keys))
	next := ""
	if end < len(keys) {
		next = keys[end-1]
	}
	return keys[start:end], next
}

func chimeSDKMapOf(v any) map[string]any { m, _ := v.(map[string]any); return m }

func chimeSDKTime(t time.Time) float64 { return float64(t.UnixMilli()) / 1000 }

// chimeSDKCopy deep-copies a stored JSON document for a response.
func chimeSDKCopy(v map[string]any) map[string]any {
	if v == nil {
		return nil
	}
	raw, _ := json.Marshal(v)
	out := map[string]any{}
	_ = json.Unmarshal(raw, &out)
	return out
}

// chimeSDKSecret returns n random bytes, base64url encoded.
func chimeSDKSecret(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

// chimeSDKPut adds key to the set stored under owner, creating it.
func chimeSDKPut(index map[string]map[string]struct{}, owner, key string) {
	set := index[owner]
	if set == nil {
		set = map[string]struct{}{}
		index[owner] = set
	}
	set[key] = struct{}{}
}

// chimeSDKDrop removes key from the set stored under owner.
func chimeSDKDrop(index map[string]map[string]struct{}, owner, key string) {
	if set := index[owner]; set != nil {
		delete(set, key)
		if len(set) == 0 {
			delete(index, owner)
		}
	}
}

func chimeSDKKeys(set map[string]struct{}) []string {
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	return out
}
