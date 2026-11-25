package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

// SageMaker control-plane operations backed by real state: feature groups
// (sagemaker_featuregroups.go, whose records Feature Store Runtime reads and
// writes), device fleets, devices and edge deployment plans
// (sagemaker_edge.go, which Edge Manager resolves) and trial components
// (sagemaker_trials.go, which SageMaker Metrics writes to). Their input is
// checked against a slim model of just these operations; every other
// operation keeps the lenient behaviour of sagemaker_store.go.
//
// Handlers run under sagemakerStore.mu and their output is marshalled before
// the lock is released, so responses never share stored maps or slices.

const (
	sagemakerErrNotFound   = "ResourceNotFound"
	sagemakerErrInUse      = "ResourceInUse"
	sagemakerErrLimit      = "ResourceLimitExceeded"
	sagemakerErrValidation = "ValidationException"

	sagemakerDefaultPage = 50
)

type sagemakerResourceError struct{ code, msg string }

func sagemakerResErr(code, format string, args ...any) *sagemakerResourceError {
	return &sagemakerResourceError{code: code, msg: fmt.Sprintf(format, args...)}
}

type sagemakerResourceHandler func(st *sagemakerStore, raw []byte, now time.Time) (any, *sagemakerResourceError)

// sagemakerTyped decodes the (already model-validated) request into T.
func sagemakerTyped[T any](f func(st *sagemakerStore, req T, now time.Time) (any, *sagemakerResourceError)) sagemakerResourceHandler {
	return func(st *sagemakerStore, raw []byte, now time.Time) (any, *sagemakerResourceError) {
		var req T
		if err := json.Unmarshal(raw, &req); err != nil {
			return nil, sagemakerResErr(sagemakerErrValidation, "invalid request: %v", err)
		}
		return f(st, req, now)
	}
}

var sagemakerResourceHandlers map[string]sagemakerResourceHandler

func init() {
	sagemakerResourceHandlers = map[string]sagemakerResourceHandler{
		"CreateFeatureGroup":         sagemakerTyped(smfsCreateFeatureGroup),
		"DescribeFeatureGroup":       sagemakerTyped(smfsDescribeFeatureGroup),
		"UpdateFeatureGroup":         sagemakerTyped(smfsUpdateFeatureGroup),
		"DeleteFeatureGroup":         sagemakerTyped(smfsDeleteFeatureGroup),
		"ListFeatureGroups":          sagemakerTyped(smfsListFeatureGroups),
		"CreateDeviceFleet":          sagemakerTyped(smeCreateDeviceFleet),
		"DescribeDeviceFleet":        sagemakerTyped(smeDescribeDeviceFleet),
		"UpdateDeviceFleet":          sagemakerTyped(smeUpdateDeviceFleet),
		"DeleteDeviceFleet":          sagemakerTyped(smeDeleteDeviceFleet),
		"ListDeviceFleets":           sagemakerTyped(smeListDeviceFleets),
		"RegisterDevices":            sagemakerTyped(smeRegisterDevices),
		"DeregisterDevices":          sagemakerTyped(smeDeregisterDevices),
		"UpdateDevices":              sagemakerTyped(smeUpdateDevices),
		"DescribeDevice":             sagemakerTyped(smeDescribeDevice),
		"ListDevices":                sagemakerTyped(smeListDevices),
		"CreateEdgeDeploymentPlan":   sagemakerTyped(smeCreatePlan),
		"DescribeEdgeDeploymentPlan": sagemakerTyped(smeDescribePlan),
		"DeleteEdgeDeploymentPlan":   sagemakerTyped(smeDeletePlan),
		"StartEdgeDeploymentStage":   sagemakerTyped(smeStartStage),
		"StopEdgeDeploymentStage":    sagemakerTyped(smeStopStage),
		"CreateTrialComponent":       sagemakerTyped(smmCreateTrialComponent),
		"DescribeTrialComponent":     sagemakerTyped(smmDescribeTrialComponent),
		"DeleteTrialComponent":       sagemakerTyped(smmDeleteTrialComponent),
		"ListTrialComponents":        sagemakerTyped(smmListTrialComponents),
	}
}

// serveSageMakerResource answers action when it is one of the stateful
// operations above and reports whether it did.
func (s *Server) serveSageMakerResource(w http.ResponseWriter, action string, payload map[string]any) bool {
	h := sagemakerResourceHandlers[action]
	if h == nil {
		return false
	}
	// The router decodes with UseNumber; re-decode so numbers are float64 for
	// the model check.
	raw, _ := json.Marshal(payload)
	in := map[string]any{}
	_ = json.Unmarshal(raw, &in)
	model := awsmodels.SageMakerResourceModel()
	if problems := model.Validate(model.Operations[action].Input, in); len(problems) > 0 {
		respondSageMakerError(w, http.StatusBadRequest, sagemakerErrValidation,
			fmt.Sprintf("%d validation error(s) detected: %s", len(problems), strings.Join(problems, "; ")))
		return true
	}
	st := s.sagemaker
	st.mu.Lock()
	st.initResourcesLocked()
	out, rerr := h(st, raw, time.Now().UTC())
	var body []byte
	if rerr == nil {
		if out == nil {
			out = map[string]any{}
		}
		body, _ = json.Marshal(out)
	}
	st.mu.Unlock()
	if rerr != nil {
		respondSageMakerError(w, http.StatusBadRequest, rerr.code, rerr.msg)
		return true
	}
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
	return true
}

// initResourcesLocked creates the maps of the stateful resources on first use.
func (s *sagemakerStore) initResourcesLocked() {
	if s.featureGroups != nil {
		return
	}
	s.featureGroups = map[string]*smfsGroup{}
	s.deviceFleets = map[string]*smeFleet{}
	s.devices = map[string]*smeDevice{}
	s.edgePlans = map[string]*smePlan{}
	s.trialComponents = map[string]*smmComponent{}
}

type sagemakerTag struct {
	Key   string `json:"Key"`
	Value string `json:"Value"`
}

func (s *sagemakerStore) setTagsLocked(arn string, tags []sagemakerTag) {
	if len(tags) == 0 {
		return
	}
	stored := s.ensureTagsLocked(arn)
	for _, t := range tags {
		stored[t.Key] = t.Value
	}
}

// sagemakerResourceARN is the ARN of a resource of kind (e.g. "feature-group").
func sagemakerResourceARN(kind, name string) string {
	return "arn:aws:sagemaker:us-east-1:" + sagemakerAccountID + ":" + kind + "/" + name
}

// sagemakerNameFromARN returns name, or the resource name when name is an ARN
// of one of kinds.
func sagemakerNameFromARN(name string, kinds ...string) string {
	if !strings.HasPrefix(name, "arn:") {
		return name
	}
	for _, kind := range kinds {
		if i := strings.Index(name, ":"+kind+"/"); i >= 0 {
			return name[i+len(kind)+2:]
		}
	}
	return name
}

func sagemakerEpoch(t time.Time) float64 { return float64(t.UnixMilli()) / 1000 }

// sagemakerTimestamp is a request timestamp: epoch seconds or an ISO 8601 string.
type sagemakerTimestamp struct{ time.Time }

func (t *sagemakerTimestamp) UnmarshalJSON(b []byte) error {
	var v any
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	switch x := v.(type) {
	case float64:
		// Beyond year ~5000 is not a real timestamp.
		if x < 0 || x > 1e11 {
			return fmt.Errorf("timestamp %v is out of range", x)
		}
		sec := int64(x)
		t.Time = time.Unix(sec, int64((x-float64(sec))*1e9)).UTC()
	case string:
		parsed, err := time.Parse(time.RFC3339Nano, x)
		if err != nil {
			return fmt.Errorf("timestamp %q is not ISO 8601", x)
		}
		t.Time = parsed.UTC()
	default:
		return fmt.Errorf("timestamp must be a number or a string")
	}
	return nil
}

// sagemakerPage returns the [start, end) window of total items for a
// NextToken (a decimal offset) and MaxResults, and the next token.
func sagemakerPage(total int, token string, max int) (int, int, string, *sagemakerResourceError) {
	start := 0
	if token != "" {
		n, err := strconv.ParseInt(token, 10, 32)
		if err != nil || n < 0 || int(n) > total {
			return 0, 0, "", sagemakerResErr(sagemakerErrValidation, "Invalid NextToken %q.", token)
		}
		start = int(n)
	}
	if max <= 0 {
		max = sagemakerDefaultPage
	}
	end := min(total, start+max)
	next := ""
	if end < total {
		next = strconv.Itoa(end)
	}
	return start, end, next, nil
}

// sagemakerSortNames returns the keys of m sorted by name, or by the time
// created reports when byTime, descending when desc.
func sagemakerSortNames[V any](m map[string]V, byTime, desc bool, created func(V) time.Time) []string {
	names := make([]string, 0, len(m))
	for name := range m {
		names = append(names, name)
	}
	sort.Slice(names, func(i, j int) bool {
		a, b := names[i], names[j]
		if desc {
			a, b = b, a
		}
		if byTime {
			if ta, tb := created(m[a]), created(m[b]); !ta.Equal(tb) {
				return ta.Before(tb)
			}
		}
		return a < b
	})
	return names
}
