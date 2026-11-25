package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

// Amazon Personalize control plane: awsJson1_1, X-Amz-Target
// AmazonPersonalize.*, signed personalize. Personalize Events and Personalize
// Runtime share the signing name but are REST JSON (personalizeevents.go,
// personalizeruntime.go) and act on the same store.

const (
	pszTargetPrefix = "AmazonPersonalize."
	pszMaxBody      = 1 << 20
)

type pszHandler func(s *Server, in map[string]any) (map[string]any, *pszErr)

var pszHandlers map[string]pszHandler

func init() {
	pszHandlers = map[string]pszHandler{
		"CreateDatasetGroup": func(s *Server, in map[string]any) (map[string]any, *pszErr) {
			return s.personalize.createDatasetGroup(in)
		},
		"CreateSchema":  func(s *Server, in map[string]any) (map[string]any, *pszErr) { return s.personalize.createSchema(in) },
		"CreateDataset": func(s *Server, in map[string]any) (map[string]any, *pszErr) { return s.personalize.createDataset(in) },
		"UpdateDataset": func(s *Server, in map[string]any) (map[string]any, *pszErr) { return s.personalize.updateDataset(in) },
		"CreateEventTracker": func(s *Server, in map[string]any) (map[string]any, *pszErr) {
			return s.personalize.createEventTracker(in)
		},
		"CreateFilter":   func(s *Server, in map[string]any) (map[string]any, *pszErr) { return s.personalize.createFilter(in) },
		"CreateSolution": func(s *Server, in map[string]any) (map[string]any, *pszErr) { return s.personalize.createSolution(in) },
		"UpdateSolution": func(s *Server, in map[string]any) (map[string]any, *pszErr) { return s.personalize.updateSolution(in) },
		"CreateSolutionVersion": func(s *Server, in map[string]any) (map[string]any, *pszErr) {
			return s.personalize.createSolutionVersion(in)
		},
		"StopSolutionVersionCreation": func(s *Server, in map[string]any) (map[string]any, *pszErr) {
			return s.personalize.stopSolutionVersion(in)
		},
		"GetSolutionMetrics": func(s *Server, in map[string]any) (map[string]any, *pszErr) { return s.personalize.solutionMetrics(in) },
		"CreateCampaign":     func(s *Server, in map[string]any) (map[string]any, *pszErr) { return s.personalize.createCampaign(in) },
		"UpdateCampaign":     func(s *Server, in map[string]any) (map[string]any, *pszErr) { return s.personalize.updateCampaign(in) },
		"CreateRecommender": func(s *Server, in map[string]any) (map[string]any, *pszErr) {
			return s.personalize.createRecommender(in)
		},
		"UpdateRecommender": func(s *Server, in map[string]any) (map[string]any, *pszErr) {
			return s.personalize.updateRecommender(in)
		},
		"StartRecommender": func(s *Server, in map[string]any) (map[string]any, *pszErr) {
			return s.personalize.startStopRecommender(pszStr(in, "recommenderArn"), true)
		},
		"StopRecommender": func(s *Server, in map[string]any) (map[string]any, *pszErr) {
			return s.personalize.startStopRecommender(pszStr(in, "recommenderArn"), false)
		},
		"CreateMetricAttribution": func(s *Server, in map[string]any) (map[string]any, *pszErr) {
			return s.personalize.createMetricAttribution(in)
		},
		"UpdateMetricAttribution": func(s *Server, in map[string]any) (map[string]any, *pszErr) {
			return s.personalize.updateMetricAttribution(in)
		},
		"ListMetricAttributionMetrics": func(s *Server, in map[string]any) (map[string]any, *pszErr) {
			return s.personalize.listMetricAttributionMetrics(in)
		},
		"CreateDatasetImportJob":  (*Server).pszCreateImportJob,
		"CreateDatasetExportJob":  (*Server).pszCreateExportJob,
		"CreateDataDeletionJob":   (*Server).pszCreateDeletionJob,
		"CreateBatchInferenceJob": (*Server).pszCreateBatchInferenceJob,
		"CreateBatchSegmentJob":   (*Server).pszCreateBatchSegmentJob,
		"DescribeAlgorithm": func(_ *Server, in map[string]any) (map[string]any, *pszErr) {
			return pszDescribeAlgorithm(pszStr(in, "algorithmArn"))
		},
		"DescribeRecipe": func(_ *Server, in map[string]any) (map[string]any, *pszErr) {
			return pszDescribeRecipe(pszStr(in, "recipeArn"))
		},
		"DescribeFeatureTransformation": func(_ *Server, in map[string]any) (map[string]any, *pszErr) {
			return pszDescribeFeatureTransformation(pszStr(in, "featureTransformationArn"))
		},
		"ListRecipes": func(_ *Server, in map[string]any) (map[string]any, *pszErr) { return pszListRecipes(in) },
		"ListTagsForResource": func(s *Server, in map[string]any) (map[string]any, *pszErr) {
			return s.personalize.listTags(pszStr(in, "resourceArn"))
		},
		"TagResource": func(s *Server, in map[string]any) (map[string]any, *pszErr) {
			return nil, s.personalize.tagResource(pszStr(in, "resourceArn"), pszTags(in))
		},
		"UntagResource": func(s *Server, in map[string]any) (map[string]any, *pszErr) {
			return nil, s.personalize.untagResource(pszStr(in, "resourceArn"), pszStrings(in["tagKeys"]))
		},
	}
	// Describe, List and Delete share one implementation per resource kind.
	for op, kind := range pszDescribeOps {
		k := pszKinds[kind]
		pszHandlers[op] = func(s *Server, in map[string]any) (map[string]any, *pszErr) {
			return s.personalize.describe(k, pszStr(in, k.arnKey))
		}
	}
	for op, l := range pszListOps {
		l := l
		pszHandlers[op] = func(s *Server, in map[string]any) (map[string]any, *pszErr) {
			return s.personalize.list(pszKinds[l.kind], l.filterKey, pszStr(in, l.filterKey), pszStr(in, "nextToken"), pszMax(in))
		}
	}
	for op, kind := range pszDeleteOps {
		k := pszKinds[kind]
		pszHandlers[op] = func(s *Server, in map[string]any) (map[string]any, *pszErr) {
			return nil, s.personalize.delete(k, pszStr(in, k.arnKey))
		}
	}
}

var pszDescribeOps = map[string]string{
	"DescribeDatasetGroup": "datasetGroup", "DescribeSchema": "schema", "DescribeDataset": "dataset",
	"DescribeDatasetImportJob": "importJob", "DescribeDatasetExportJob": "exportJob", "DescribeDataDeletionJob": "deletionJob",
	"DescribeEventTracker": "eventTracker", "DescribeFilter": "filter", "DescribeSolution": "solution",
	"DescribeSolutionVersion": "solutionVersion", "DescribeCampaign": "campaign", "DescribeRecommender": "recommender",
	"DescribeMetricAttribution": "metricAttribution", "DescribeBatchInferenceJob": "batchInferenceJob",
	"DescribeBatchSegmentJob": "batchSegmentJob",
}

type pszListOp struct{ kind, filterKey string }

var pszListOps = map[string]pszListOp{
	"ListDatasetGroups": {"datasetGroup", ""}, "ListSchemas": {"schema", ""},
	"ListDatasets": {"dataset", "datasetGroupArn"}, "ListDatasetImportJobs": {"importJob", "datasetArn"},
	"ListDatasetExportJobs": {"exportJob", "datasetArn"}, "ListDataDeletionJobs": {"deletionJob", "datasetGroupArn"},
	"ListEventTrackers": {"eventTracker", "datasetGroupArn"}, "ListFilters": {"filter", "datasetGroupArn"},
	"ListSolutions": {"solution", "datasetGroupArn"}, "ListSolutionVersions": {"solutionVersion", "solutionArn"},
	"ListCampaigns": {"campaign", "solutionArn"}, "ListRecommenders": {"recommender", "datasetGroupArn"},
	"ListMetricAttributions": {"metricAttribution", "datasetGroupArn"},
	"ListBatchInferenceJobs": {"batchInferenceJob", "solutionVersionArn"},
	"ListBatchSegmentJobs":   {"batchSegmentJob", "solutionVersionArn"},
}

var pszDeleteOps = map[string]string{
	"DeleteDatasetGroup": "datasetGroup", "DeleteSchema": "schema", "DeleteDataset": "dataset",
	"DeleteEventTracker": "eventTracker", "DeleteFilter": "filter", "DeleteSolution": "solution",
	"DeleteCampaign": "campaign", "DeleteRecommender": "recommender", "DeleteMetricAttribution": "metricAttribution",
}

func (s *Server) handlePersonalizeJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}
	op, ok := strings.CutPrefix(strings.TrimSpace(r.Header.Get("X-Amz-Target")), pszTargetPrefix)
	if !ok {
		return false
	}
	// A declared length bounds what net/http reads; refuse chunked bodies.
	if r.ContentLength < 0 || r.ContentLength > pszMaxBody {
		respondPersonalizeError(w, http.StatusBadRequest, "InvalidInputException", "request body must declare a Content-Length of at most 1 MiB")
		return true
	}
	if ok, status, code, msg, _ := s.validateSigV4WithService(r, "personalize"); !ok {
		respondPersonalizeError(w, status, code, msg)
		return true
	}
	h := pszHandlers[op]
	if _, known := personalizeOperationByName[op]; !known || h == nil {
		respondPersonalizeError(w, http.StatusBadRequest, "UnknownOperationException", "unknown operation "+op)
		return true
	}
	in, perr := pszDecode(r, awsmodels.PersonalizeModel(), op)
	if perr != nil {
		respondPersonalizeError(w, http.StatusBadRequest, perr.code, perr.msg)
		return true
	}
	out, perr := h(s, in)
	if perr != nil {
		respondPersonalizeError(w, http.StatusBadRequest, perr.code, perr.msg)
		return true
	}
	if out == nil {
		out = map[string]any{}
	}
	model := awsmodels.PersonalizeModel()
	// out is built fresh under the store lock; nothing in it is shared.
	body, _ := json.Marshal(model.Conform(model.Operations[op].Output, out))
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
	return true
}

// pszDecode reads a JSON body, validates it against op's input shape and
// conforms it (members the shape doesn't define are dropped).
func pszDecode(r *http.Request, model *awsmodels.Model, op string) (map[string]any, *pszErr) {
	raw, err := readBodyBytes(r)
	in := map[string]any{}
	if err == nil && strings.TrimSpace(string(raw)) != "" {
		err = json.Unmarshal(raw, &in)
	}
	if err != nil {
		return nil, pszError("SerializationException", "invalid JSON body")
	}
	shapes := model.Operations[op]
	if problems := model.Validate(shapes.Input, in); len(problems) > 0 {
		return nil, pszError("InvalidInputException", "%s", strings.Join(problems, "; "))
	}
	conformed, _ := model.Conform(shapes.Input, in).(map[string]any)
	if conformed == nil {
		conformed = map[string]any{}
	}
	return conformed, nil
}

func respondPersonalizeError(w http.ResponseWriter, status int, code, msg string) {
	body, _ := json.Marshal(map[string]string{"__type": code, "message": msg})
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.Header().Set("X-Amzn-ErrorType", code)
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

// ---- request helpers ----

func pszStr(m map[string]any, k string) string { v, _ := m[k].(string); return v }

func pszMap(m map[string]any, k string) map[string]any { v, _ := m[k].(map[string]any); return v }

func pszList(m map[string]any, k string) []any { v, _ := m[k].([]any); return v }

// pszBool returns the boolean m[k], or def when it isn't set.
func pszBool(m map[string]any, k string, def bool) bool {
	if v, ok := m[k].(bool); ok {
		return v
	}
	return def
}

// pszInt returns the integer m[k] (range-checked by the model), or def.
func pszInt(m map[string]any, k string, def int) int {
	if v, ok := m[k].(float64); ok && v >= -1e9 && v <= 1e9 {
		return int(v)
	}
	return def
}

func pszMax(m map[string]any) int { return pszInt(m, "maxResults", pszDefaultPage) }

func pszStrings(v any) []string {
	list, _ := v.([]any)
	out := make([]string, 0, len(list))
	for _, item := range list {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func pszTags(in map[string]any) [][2]string {
	var out [][2]string
	for _, item := range pszList(in, "tags") {
		t, _ := item.(map[string]any)
		out = append(out, [2]string{pszStr(t, "tagKey"), pszStr(t, "tagValue")})
	}
	return out
}

// pszTime parses a JSON timestamp: epoch seconds or an RFC 3339 string.
func pszTime(v any) (time.Time, bool) {
	switch t := v.(type) {
	case float64:
		if t < 0 || t > 1e11 { // beyond year ~5000: not a real timestamp
			return time.Time{}, false
		}
		sec := int64(t)
		return time.Unix(sec, int64((t-float64(sec))*1e9)).UTC(), true
	case string:
		if parsed, err := time.Parse(time.RFC3339Nano, t); err == nil {
			return parsed.UTC(), true
		}
	}
	return time.Time{}, false
}

func pszEpoch(t time.Time) float64 { return float64(t.UnixMilli()) / 1000 }
