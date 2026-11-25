package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

// Amazon SageMaker Feature Store Runtime (featurestore-runtime.sagemaker,
// REST JSON, signed "sagemaker"). Records are kept in the online store of
// feature groups created with the SageMaker control plane
// (sagemaker_featuregroups.go), under the SageMaker store's lock.
//
// API reference:
// https://docs.aws.amazon.com/sagemaker/latest/APIReference/API_Operations_Amazon_SageMaker_Feature_Store_Runtime.html

const (
	smfsMaxBody = 4 << 20
	smfsHost    = "featurestore-runtime.sagemaker."
)

var (
	smfsGroupPath   = regexp.MustCompile(`^/FeatureGroup/([^/]+)(/ListRecords|/Record)?$`)
	smfsGroupRefRE  = regexp.MustCompile(`^(arn:aws[a-z\-]*:sagemaker:[a-z0-9\-]*:[0-9]{12}:feature-group/)?([a-zA-Z0-9]([-_]*[a-zA-Z0-9]){0,63})$`)
	smfsFeatureName = regexp.MustCompile(`^[a-zA-Z0-9]([-_]*[a-zA-Z0-9]){0,63}$`)
)

// sagemakerDataError is an error of the REST JSON SageMaker data planes
// (Feature Store Runtime, Metrics, Edge).
type sagemakerDataError struct {
	status    int
	code, msg string
}

func smfsInvalid(format string, args ...any) *sagemakerDataError {
	return &sagemakerDataError{http.StatusBadRequest, "ValidationError", fmt.Sprintf(format, args...)}
}

// sagemakerDataPlaneClaims reports whether r belongs to a SageMaker REST data
// plane: no JSON target (the control plane uses one), and signed for
// "sagemaker" or sent to the data plane's host.
func sagemakerDataPlaneClaims(r *http.Request, hostPrefix string) bool {
	if strings.TrimSpace(r.Header.Get("X-Amz-Target")) != "" {
		return false
	}
	return strings.TrimSpace(sigV4ServiceHint(r)) == "sagemaker" || strings.HasPrefix(strings.ToLower(r.Host), hostPrefix)
}

// runSageMakerDataPlane checks the body length and signature, validates the
// JSON body against op's input shape and runs h on the raw (validated) body.
// It returns the marshalled result, or writes the error and returns false.
func (s *Server) runSageMakerDataPlane(w http.ResponseWriter, r *http.Request, model *awsmodels.Model, op string, maxBody int64, h func(raw []byte) (any, *sagemakerDataError)) ([]byte, bool) {
	// A declared length bounds what net/http reads; refuse chunked bodies.
	if r.ContentLength < 0 || r.ContentLength > maxBody {
		respondSageMakerDataError(w, smfsInvalid("The request body must declare a Content-Length of at most %d bytes.", maxBody))
		return nil, false
	}
	if ok, status, code, msg, _ := s.validateSigV4WithService(r, "sagemaker"); !ok {
		respondSageMakerDataError(w, &sagemakerDataError{status, code, msg})
		return nil, false
	}
	raw, err := readBodyBytes(r)
	in := map[string]any{}
	if err == nil && strings.TrimSpace(string(raw)) != "" {
		err = json.Unmarshal(raw, &in)
	}
	if err != nil {
		respondSageMakerDataError(w, smfsInvalid("The request body is not valid JSON."))
		return nil, false
	}
	if problems := model.Validate(model.Operations[op].Input, in); len(problems) > 0 {
		respondSageMakerDataError(w, smfsInvalid("%d validation error(s) detected: %s", len(problems), strings.Join(problems, "; ")))
		return nil, false
	}
	if len(raw) == 0 {
		raw = []byte("{}")
	}
	out, derr := h(raw)
	if derr != nil {
		respondSageMakerDataError(w, derr)
		return nil, false
	}
	if out == nil {
		out = map[string]any{}
	}
	body, _ := json.Marshal(out)
	return body, true
}

func respondSageMakerDataError(w http.ResponseWriter, e *sagemakerDataError) {
	body, _ := json.Marshal(map[string]string{"__type": e.code, "message": e.msg})
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Amzn-ErrorType", e.code)
	w.WriteHeader(e.status)
	_, _ = w.Write(body)
}

// smfsRoute returns the operation and the feature group reference of a
// Feature Store Runtime request.
func smfsRoute(r *http.Request) (string, string, bool) {
	path := rawRequestPath(r)
	switch {
	case r.Method == http.MethodPost && path == "/BatchGetRecord":
		return "BatchGetRecord", "", true
	case r.Method == http.MethodPost && path == "/BatchWriteRecord":
		return "BatchWriteRecord", "", true
	}
	m := smfsGroupPath.FindStringSubmatch(path)
	if m == nil {
		return "", "", false
	}
	group, err := url.PathUnescape(m[1])
	if err != nil {
		return "", "", false
	}
	op := map[string]string{
		http.MethodPut + " ":              "PutRecord",
		http.MethodGet + " ":              "GetRecord",
		http.MethodDelete + " ":           "DeleteRecord",
		http.MethodPost + " /ListRecords": "ListRecords",
		http.MethodPost + " /Record":      "UpdateRecord",
	}[r.Method+" "+m[2]]
	return op, group, op != ""
}

func (s *Server) handleSageMakerFeatureStoreRuntimeRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !sagemakerDataPlaneClaims(r, smfsHost) {
		return false
	}
	op, group, ok := smfsRoute(r)
	if !ok {
		return false
	}
	if _, known := sagemakerfeaturestoreruntimeOperationByName[op]; !known {
		return false
	}
	body, ok := s.runSageMakerDataPlane(w, r, awsmodels.SageMakerFeatureStoreRuntimeModel(), op, smfsMaxBody, func(raw []byte) (any, *sagemakerDataError) {
		if op != "BatchGetRecord" && op != "BatchWriteRecord" && !smfsGroupRefRE.MatchString(group) {
			return nil, smfsInvalid("FeatureGroupName %q does not match the feature group name or ARN pattern.", group)
		}
		return s.smfsDispatch(op, group, r.URL.Query(), raw)
	})
	if ok {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}
	return true
}

func (s *Server) smfsDispatch(op, group string, q url.Values, raw []byte) (any, *sagemakerDataError) {
	st := s.sagemaker
	now := time.Now().UTC()
	switch op {
	case "GetRecord":
		names := q["FeatureName"]
		for _, n := range names {
			if !smfsFeatureName.MatchString(n) {
				return nil, smfsInvalid("FeatureName %q does not match the feature name pattern.", n)
			}
		}
		if _, ok := q["RecordIdentifierValueAsString"]; !ok {
			return nil, smfsInvalid("RecordIdentifierValueAsString is required.")
		}
		exp := q.Get("ExpirationTimeResponse")
		if exp != "" && exp != "Enabled" && exp != "Disabled" {
			return nil, smfsInvalid("ExpirationTimeResponse must be Enabled or Disabled.")
		}
		return st.smfsLocked(func() (any, *sagemakerDataError) {
			return st.smfsGetRecordLocked(group, q.Get("RecordIdentifierValueAsString"), names, exp == "Enabled", now)
		})
	case "DeleteRecord":
		for _, key := range []string{"RecordIdentifierValueAsString", "EventTime"} {
			if _, ok := q[key]; !ok {
				return nil, smfsInvalid("%s is required.", key)
			}
		}
		mode := q.Get("DeletionMode")
		if mode == "" {
			mode = "SoftDelete"
		}
		if mode != "SoftDelete" && mode != "HardDelete" {
			return nil, smfsInvalid("DeletionMode must be SoftDelete or HardDelete.")
		}
		targets := q["TargetStores"]
		if err := smfsCheckTargetList(targets); err != nil {
			return nil, err
		}
		return st.smfsLocked(func() (any, *sagemakerDataError) {
			return nil, st.smfsDeleteRecordLocked(group, q.Get("RecordIdentifierValueAsString"), q.Get("EventTime"), mode, targets)
		})
	case "PutRecord":
		var req smfsWriteReq
		if err := json.Unmarshal(raw, &req); err != nil {
			return nil, smfsInvalid("invalid request: %v", err)
		}
		return st.smfsLocked(func() (any, *sagemakerDataError) {
			return nil, st.smfsPutLocked(group, req.Record, req.TargetStores, req.TtlDuration, now)
		})
	case "UpdateRecord":
		var req smfsUpdateReq
		if err := json.Unmarshal(raw, &req); err != nil {
			return nil, smfsInvalid("invalid request: %v", err)
		}
		return st.smfsLocked(func() (any, *sagemakerDataError) { return nil, st.smfsUpdateLocked(group, req, now) })
	case "ListRecords":
		var req smfsListReq
		if err := json.Unmarshal(raw, &req); err != nil {
			return nil, smfsInvalid("invalid request: %v", err)
		}
		return st.smfsListRecords(group, req, now)
	case "BatchGetRecord":
		var req smfsBatchGetReq
		if err := json.Unmarshal(raw, &req); err != nil {
			return nil, smfsInvalid("invalid request: %v", err)
		}
		return st.smfsLocked(func() (any, *sagemakerDataError) { return st.smfsBatchGetLocked(req, now), nil })
	case "BatchWriteRecord":
		var req smfsBatchWriteReq
		if err := json.Unmarshal(raw, &req); err != nil {
			return nil, smfsInvalid("invalid request: %v", err)
		}
		return st.smfsLocked(func() (any, *sagemakerDataError) { return st.smfsBatchWriteLocked(req, now), nil })
	}
	return nil, smfsInvalid("unknown operation %s", op)
}

// smfsLocked runs f under the SageMaker store's lock and marshals its result
// before unlocking, so responses never share stored slices.
func (s *sagemakerStore) smfsLocked(f func() (any, *sagemakerDataError)) (any, *sagemakerDataError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.initResourcesLocked()
	out, err := f()
	if err != nil || out == nil {
		return nil, err
	}
	body, _ := json.Marshal(out)
	return json.RawMessage(body), nil
}

func smfsCheckTargetList(targets []string) *sagemakerDataError {
	if len(targets) > 2 {
		return smfsInvalid("TargetStores has at most 2 items.")
	}
	for _, t := range targets {
		if t != "OnlineStore" && t != "OfflineStore" {
			return smfsInvalid("TargetStores items must be OnlineStore or OfflineStore.")
		}
	}
	return nil
}
