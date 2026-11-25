package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

// Amazon Kendra Intelligent Ranking: awsJson1_0, X-Amz-Target
// AWSKendraRerankingFrontendService.*, signed kendra-ranking. The CLI 2.42.1
// model lists smithy-rpc-v2-cbor first, but the CLI sends JSON 1.0. Kendra's
// own router claims any target containing "Kendra", so this one runs first.

const (
	kdrTargetPrefix = "AWSKendraRerankingFrontendService."
	kdrMaxBody      = 1 << 20
	kdrAccountID    = "123456789012"
	kdrRegion       = "us-east-1"
	kdrContentType  = "application/x-amz-json-1.0"
)

type kdrErr struct {
	status    int
	code, msg string
}

func kdrValidation(format string, args ...any) *kdrErr {
	return &kdrErr{http.StatusBadRequest, "ValidationException", fmt.Sprintf(format, args...)}
}

func kdrNotFound(format string, args ...any) *kdrErr {
	return &kdrErr{http.StatusNotFound, "ResourceNotFoundException", fmt.Sprintf(format, args...)}
}

type kdrHandler func(st *kdrStore, in map[string]any) (map[string]any, *kdrErr)

var kdrHandlers = map[string]kdrHandler{
	"CreateRescoreExecutionPlan":   (*kdrStore).createPlan,
	"DescribeRescoreExecutionPlan": (*kdrStore).describePlan,
	"ListRescoreExecutionPlans":    (*kdrStore).listPlans,
	"UpdateRescoreExecutionPlan":   (*kdrStore).updatePlan,
	"DeleteRescoreExecutionPlan":   (*kdrStore).deletePlan,
	"Rescore":                      (*kdrStore).rescore,
	"TagResource":                  (*kdrStore).tagResource,
	"UntagResource":                (*kdrStore).untagResource,
	"ListTagsForResource":          (*kdrStore).listTags,
}

func (s *Server) handleKendraRankingJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}
	op, ok := strings.CutPrefix(strings.TrimSpace(r.Header.Get("X-Amz-Target")), kdrTargetPrefix)
	if !ok {
		return false
	}
	// A declared length bounds what is read; refuse chunked bodies.
	if r.ContentLength < 0 || r.ContentLength > kdrMaxBody {
		respondKdrError(w, kdrValidation("request body must declare a Content-Length of at most 1 MiB"))
		return true
	}
	if ok, status, code, msg, _ := s.validateSigV4WithService(r, "kendra-ranking"); !ok {
		respondKdrError(w, &kdrErr{status, code, msg})
		return true
	}
	h := kdrHandlers[op]
	if !kendraRankingOperationByName[op] || h == nil {
		respondKdrError(w, &kdrErr{http.StatusBadRequest, "UnknownOperationException", "unknown operation " + op})
		return true
	}
	raw, err := io.ReadAll(io.LimitReader(r.Body, kdrMaxBody))
	in := map[string]any{}
	if err == nil && strings.TrimSpace(string(raw)) != "" {
		err = json.Unmarshal(raw, &in)
	}
	if err != nil {
		respondKdrError(w, &kdrErr{http.StatusBadRequest, "SerializationException", "invalid JSON body"})
		return true
	}
	model := awsmodels.KendraRankingModel()
	shapes := model.Operations[op]
	if problems := model.Validate(shapes.Input, in); len(problems) > 0 {
		respondKdrError(w, kdrValidation("%s", strings.Join(problems, "; ")))
		return true
	}
	// Conform drops members the input shape doesn't define, so callers can't
	// reach the store's own fields.
	conformed, _ := model.Conform(shapes.Input, in).(map[string]any)
	out, kerr := h(s.kendraRanking, conformed)
	if kerr != nil {
		respondKdrError(w, kerr)
		return true
	}
	if out == nil {
		out = map[string]any{}
	}
	// out is built fresh by every handler; nothing in it is shared with the store.
	body, _ := json.Marshal(model.Conform(shapes.Output, out))
	w.Header().Set("Content-Type", kdrContentType)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
	return true
}

func respondKdrError(w http.ResponseWriter, e *kdrErr) {
	body, _ := json.Marshal(map[string]string{"__type": e.code, "Message": e.msg})
	w.Header().Set("Content-Type", kdrContentType)
	w.Header().Set("X-Amzn-ErrorType", e.code)
	w.WriteHeader(e.status)
	_, _ = w.Write(body)
}

func kdrStr(in map[string]any, key string) string {
	v, _ := in[key].(string)
	return v
}

func kdrList(in map[string]any, key string) []any {
	l, _ := in[key].([]any)
	return l
}
