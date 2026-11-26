package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/stackyard/stackyard/internal/awsmodels"
	"github.com/stackyard/stackyard/internal/services/sns"
)

// Amazon Timestream for LiveAnalytics: the Write and Query APIs (JSON 1.0,
// X-Amz-Target Timestream_20181101.*, SigV4 name "timestream") share one store.
// Timestream for InfluxDB is a separate service (AmazonTimestreamInfluxDB.*).

const (
	tsmTargetPrefix = "Timestream_20181101."
	tsmMaxBody      = 4 << 20
	tsmAccount      = "123456789012"
	tsmRegion       = "us-east-1"
)

var (
	timestreamWriteOperationByName = tsmOpSet(timestreamWriteOperations)
	timestreamQueryOperationByName = tsmOpSet(timestreamQueryOperations)
)

func tsmOpSet(ops []tsmOperation) map[string]bool {
	out := make(map[string]bool, len(ops))
	for _, op := range ops {
		out[op.Name] = true
	}
	return out
}

type tsmError struct {
	code  string
	msg   string
	extra map[string]any
}

func (e *tsmError) status() int {
	switch e.code {
	case "InternalServerException":
		return http.StatusInternalServerError
	case "AccessDeniedException":
		return http.StatusForbidden
	}
	return http.StatusBadRequest
}

func tsmValidation(format string, args ...any) *tsmError {
	return &tsmError{code: "ValidationException", msg: fmt.Sprintf(format, args...)}
}

func tsmNotFound(format string, args ...any) *tsmError {
	return &tsmError{code: "ResourceNotFoundException", msg: fmt.Sprintf(format, args...)}
}

func tsmConflict(format string, args ...any) *tsmError {
	return &tsmError{code: "ConflictException", msg: fmt.Sprintf(format, args...)}
}

func tsmQuota(format string, args ...any) *tsmError {
	return &tsmError{code: "ServiceQuotaExceededException", msg: fmt.Sprintf(format, args...)}
}

// tsmModelFor picks the model that defines op. Shared operations (endpoints,
// tags) use the Query model, whose ListTagsForResource adds paging.
func tsmModelFor(op string) *awsmodels.Model {
	if timestreamQueryOperationByName[op] {
		return awsmodels.TimestreamQueryModel()
	}
	return awsmodels.TimestreamWriteModel()
}

// tsmCall carries one request into the store. Anything read from other stores
// (S3 objects, SNS topics) is gathered before the store lock is taken.
type tsmCall struct {
	in      map[string]any
	host    string
	now     time.Time
	buckets map[string]bool   // S3 bucket existence
	objects map[string][]byte // batch load source objects by key
	model   []byte            // batch load data model object
	topicOK bool
	denied  string // why a batch load source couldn't be read, if access was denied
	putS3   func(bucket, key string, body []byte)
	publish func(topic, message string)
	after   []func() // writes to other stores, run once the Timestream lock is released
}

func (s *Server) handleTimestreamJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	target := strings.TrimSpace(r.Header.Get("X-Amz-Target"))
	if r.Method != http.MethodPost || !strings.HasPrefix(target, tsmTargetPrefix) {
		return false
	}
	if hint := strings.TrimSpace(sigV4ServiceHint(r)); hint != "" && hint != "timestream" {
		return false
	}
	// A declared length bounds what net/http reads, so require one.
	if r.ContentLength < 0 || r.ContentLength > tsmMaxBody {
		respondTimestreamError(w, tsmValidation("request body must declare a Content-Length of at most 4 MiB"))
		return true
	}
	if ok, status, code, msg, _ := s.validateSigV4WithService(r, "timestream"); !ok {
		respondTimestreamJSON(w, status, map[string]any{"__type": code, "message": msg})
		return true
	}
	op := strings.TrimPrefix(target, tsmTargetPrefix)
	if !timestreamWriteOperationByName[op] && !timestreamQueryOperationByName[op] {
		respondTimestreamJSON(w, http.StatusBadRequest, map[string]any{"__type": "UnknownOperationException", "message": "unknown operation " + op})
		return true
	}
	raw, err := readBodyBytes(r)
	in := map[string]any{}
	if err == nil && len(bytes.TrimSpace(raw)) > 0 {
		err = json.Unmarshal(raw, &in)
	}
	if err != nil {
		respondTimestreamJSON(w, http.StatusBadRequest, map[string]any{"__type": "SerializationException", "message": "request body is not a JSON object"})
		return true
	}
	model := tsmModelFor(op)
	shapes := model.Operations[op]
	if problems := model.Validate(shapes.Input, in); len(problems) > 0 {
		respondTimestreamError(w, tsmValidation("%s", problems[0]))
		return true
	}
	// Inputs are conformed to the model, so callers can't smuggle extra fields.
	in, _ = model.Conform(shapes.Input, in).(map[string]any)
	if in == nil {
		in = map[string]any{}
	}
	call := &tsmCall{in: in, host: r.Host, now: time.Now().UTC(), buckets: map[string]bool{}}
	s.tsmGather(r, op, call)
	out, terr := s.tsdb.dispatch(op, call)
	for _, fn := range call.after {
		fn()
	}
	if terr != nil {
		respondTimestreamError(w, terr)
		return true
	}
	respondTimestreamJSON(w, http.StatusOK, out)
	return true
}

// tsmGather reads what an operation needs from the S3 and SNS stores before
// the Timestream lock is taken.
const tsmServicePrincipal = "timestream.amazonaws.com"

func (s *Server) tsmGather(r *http.Request, op string, call *tsmCall) {
	caller := s.s3CallerOf(r)
	allowed := func(action, bucket, key string) bool {
		if s.authorizeServiceS3As(caller, tsmServicePrincipal, bucket, key, action) {
			return true
		}
		if call.denied == "" {
			call.denied = "Access denied: " + action + " on s3://" + bucket + "/" + key
		}
		return false
	}
	bucket := func(name string) {
		if name == "" {
			return
		}
		if _, ok := call.buckets[name]; !ok {
			_, err := s.s3.GetBucket(name)
			call.buckets[name] = err == nil
		}
	}
	call.putS3 = func(b, key string, body []byte) {
		// ponytail: a denied report write is dropped, like a failed one; the
		// check runs here, outside the Timestream lock, as the write does.
		call.after = append(call.after, func() {
			if s.authorizeServiceS3As(caller, tsmServicePrincipal, b, key, "s3:PutObject") {
				_, _ = s.s3.PutObject(b, key, "text/csv", body, nil, "", "")
			}
		})
	}
	call.publish = func(topic, message string) {
		call.after = append(call.after, func() { _, _ = s.sns.Publish(sns.PublishInput{TopicARN: topic, Message: message}) })
	}
	in := call.in
	switch op {
	case "CreateBatchLoadTask":
		src := tsmPath(in, "DataSourceConfiguration", "DataSourceS3Configuration")
		srcBucket := tsmStr(src, "BucketName")
		bucket(srcBucket)
		bucket(tsmStr(tsmPath(in, "ReportConfiguration", "ReportS3Configuration"), "BucketName"))
		if call.buckets[srcBucket] && allowed("s3:ListBucket", srcBucket, "") {
			call.objects = s.tsmReadObjects(srcBucket, tsmStr(src, "ObjectKeyPrefix"), func(key string) bool {
				return allowed("s3:GetObject", srcBucket, key)
			})
		}
		if dm := tsmPath(in, "DataModelConfiguration", "DataModelS3Configuration"); dm != nil &&
			allowed("s3:GetObject", tsmStr(dm, "BucketName"), tsmStr(dm, "ObjectKey")) {
			if obj, err := s.s3.GetObject(tsmStr(dm, "BucketName"), tsmStr(dm, "ObjectKey")); err == nil && len(obj.Body) <= 1<<20 {
				call.model = obj.Body
			}
		}
	case "CreateTable", "UpdateTable":
		bucket(tsmStr(tsmPath(in, "MagneticStoreWriteProperties", "MagneticStoreRejectedDataLocation", "S3Configuration"), "BucketName"))
	case "CreateScheduledQuery":
		bucket(tsmStr(tsmPath(in, "ErrorReportConfiguration", "S3Configuration"), "BucketName"))
		_, err := s.sns.GetTopic(tsmStr(tsmPath(in, "NotificationConfiguration", "SnsConfiguration"), "TopicArn"))
		call.topicOK = err == nil
	}
}

// tsmReadObjects loads the CSV objects under prefix, bounded in count and bytes.
func (s *Server) tsmReadObjects(bucket, prefix string, allowed func(key string) bool) map[string][]byte {
	objs, err := s.s3.ListObjects(bucket, prefix)
	if err != nil {
		return nil
	}
	out := map[string][]byte{}
	total := 0
	for _, o := range objs {
		if len(out) >= tsmMaxBatchFiles || strings.HasSuffix(o.Key, "/") {
			continue
		}
		if !allowed(o.Key) {
			return nil
		}
		full, err := s.s3.GetObject(bucket, o.Key)
		if err != nil || total+len(full.Body) > tsmMaxBatchBytes {
			continue
		}
		total += len(full.Body)
		out[o.Key] = full.Body
	}
	return out
}

func respondTimestreamJSON(w http.ResponseWriter, status int, body any) {
	raw, _ := json.Marshal(body)
	w.Header().Set("Content-Type", "application/x-amz-json-1.0")
	w.WriteHeader(status)
	_, _ = w.Write(raw)
}

func respondTimestreamError(w http.ResponseWriter, e *tsmError) {
	body := map[string]any{"__type": e.code, "Message": e.msg}
	for k, v := range e.extra {
		body[k] = v
	}
	w.Header().Set("X-Amzn-ErrorType", e.code)
	respondTimestreamJSON(w, e.status(), body)
}

// --- small helpers over decoded JSON ---

func tsmStr(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	s, _ := m[key].(string)
	return s
}

func tsmMap(m map[string]any, key string) map[string]any {
	if m == nil {
		return nil
	}
	v, _ := m[key].(map[string]any)
	return v
}

func tsmPath(m map[string]any, keys ...string) map[string]any {
	for _, k := range keys {
		m = tsmMap(m, k)
	}
	return m
}

func tsmList(m map[string]any, key string) []any {
	if m == nil {
		return nil
	}
	v, _ := m[key].([]any)
	return v
}

func tsmInt(m map[string]any, key string, def int64) int64 {
	if m == nil {
		return def
	}
	if f, ok := m[key].(float64); ok {
		return int64(f)
	}
	return def
}

func tsmBool(m map[string]any, key string) (bool, bool) {
	if m == nil {
		return false, false
	}
	b, ok := m[key].(bool)
	return b, ok
}

func tsmEpoch(t time.Time) float64 { return float64(t.UnixNano()) / 1e9 }

// tsmPage slices items by an offset NextToken; the token is parsed at 32 bits.
func tsmPage[T any](items []T, token string, max int64, def int) ([]T, string, *tsmError) {
	start := 0
	if token != "" {
		n, err := strconv.ParseInt(token, 10, 32)
		if err != nil || n < 0 || int(n) > len(items) {
			return nil, "", tsmValidation("Invalid NextToken")
		}
		start = int(n)
	}
	size := def
	if max > 0 {
		size = int(max)
	}
	end := min(start+size, len(items))
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return items[start:end], next, nil
}
