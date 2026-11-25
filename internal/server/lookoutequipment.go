package server

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

// Amazon Lookout for Equipment: awsJson1_0, X-Amz-Target
// AWSLookoutEquipmentFrontendService.*, signed lookoutequipment. Input is
// validated and conformed against the slim CLI model by fcDecode (shared with
// Forecast). State: lookoutequipment_store.go (datasets, ingestion, labels,
// tags, policies) and lookoutequipment_models.go (models, versions and
// schedulers); S3 reads in lookoutequipment_data.go.

const lkeTargetPrefix = "AWSLookoutEquipmentFrontendService."

func (s *Server) handleLookoutEquipmentJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}
	op, ok := strings.CutPrefix(strings.TrimSpace(r.Header.Get("X-Amz-Target")), lkeTargetPrefix)
	if !ok {
		return false
	}
	in, ferr := fcDecode(s, w, r, "lookoutequipment", awsmodels.LookoutEquipmentModel(), op)
	if ferr == nil && in == nil { // signature failure already written
		return true
	}
	var out any
	if ferr == nil {
		out, ferr = s.lkeRun(op, s.withS3Caller(in, r))
	}
	if ferr != nil {
		code := ferr.code
		if code == "InvalidInputException" { // fcDecode's model-validation code
			code = "ValidationException"
		}
		writeLKEJSON(w, http.StatusBadRequest, map[string]any{"__type": code, "Message": ferr.msg}, code)
		return true
	}
	writeLKEJSON(w, http.StatusOK, out, "")
	return true
}

func writeLKEJSON(w http.ResponseWriter, status int, v any, errorType string) {
	body, _ := json.Marshal(v)
	w.Header().Set("Content-Type", "application/x-amz-json-1.0")
	if errorType != "" {
		w.Header().Set("X-Amzn-ErrorType", errorType)
	}
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

func lkeInvalid(format string, args ...any) *fcErr {
	return fcError("ValidationException", format, args...)
}

func lkeConflict(format string, args ...any) *fcErr {
	return fcError("ConflictException", format, args...)
}

func lkeQuota(format string, args ...any) *fcErr {
	return fcError("ServiceQuotaExceededException", format, args...)
}

// lkeHex returns n random bytes as lowercase hex.
func lkeHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func lkeArn(kind, name string) string {
	return "arn:aws:lookoutequipment:" + fcRegion + ":" + fcAccount + ":" + kind + "/" + name + "/" + lkeHex(16)
}

// lkePage sorts summaries by key and returns one page of them.
func lkePage(items []map[string]any, in map[string]any, key, listKey string) (map[string]any, *fcErr) {
	sort.Slice(items, func(i, j int) bool { return fcStr(items[i], key) < fcStr(items[j], key) })
	start := 0
	if tok := fcStr(in, "NextToken"); tok != "" {
		n, err := strconv.ParseInt(tok, 10, 32)
		if err != nil || n < 0 || int(n) > len(items) {
			return nil, lkeInvalid("invalid NextToken")
		}
		start = int(n)
	}
	limit := 50
	if v, ok := fcNum(in, "MaxResults"); ok {
		limit = int(v) // the model bounds MaxResults to 1..500
	}
	end := min(start+limit, len(items))
	page := make([]any, 0, end-start)
	for _, item := range items[start:end] {
		page = append(page, item)
	}
	out := map[string]any{listKey: page}
	if end < len(items) {
		out["NextToken"] = strconv.Itoa(end)
	}
	return out, nil
}

// lkePick copies the named fields of doc.
func lkePick(doc map[string]any, keys ...string) map[string]any {
	out := make(map[string]any, len(keys))
	for _, k := range keys {
		if v, ok := doc[k]; ok {
			out[k] = v
		}
	}
	return out
}

func lkeCopy(doc map[string]any) map[string]any {
	out := make(map[string]any, len(doc))
	for k, v := range doc {
		out[k] = v
	}
	return out
}
