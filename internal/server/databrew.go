package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

// dataBrewRoots are the first path segments DataBrew owns.
var dataBrewRoots = map[string]bool{
	"datasets": true, "recipes": true, "recipeVersions": true, "projects": true, "profileJobs": true,
	"recipeJobs": true, "jobs": true, "schedules": true, "rulesets": true, "tags": true,
}

type dbwError struct {
	status int
	code   string
	msg    string
}

func dbwValidation(format string, args ...any) *dbwError {
	return &dbwError{http.StatusBadRequest, "ValidationException", fmt.Sprintf(format, args...)}
}

func dbwNotFound(format string, args ...any) *dbwError {
	return &dbwError{http.StatusNotFound, "ResourceNotFoundException", fmt.Sprintf(format, args...)}
}

func dbwConflict(format string, args ...any) *dbwError {
	return &dbwError{http.StatusConflict, "ConflictException", fmt.Sprintf(format, args...)}
}

func dbwQuota(format string, args ...any) *dbwError {
	return &dbwError{http.StatusPaymentRequired, "ServiceQuotaExceededException", fmt.Sprintf(format, args...)}
}

// handleDataBrewRESTRouter serves AWS Glue DataBrew (REST-JSON, signing name
// databrew). Recipe and profile jobs run against CSV/JSON data in the S3 store.
func (s *Server) handleDataBrewRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isDataBrewCandidate(r) {
		return false
	}
	// A declared length bounds what net/http reads, so require one.
	if r.ContentLength < 0 || r.ContentLength > dbwMaxBody {
		respondDataBrewError(w, dbwValidation("request body must declare a Content-Length of at most 1 MiB"))
		return true
	}
	if ok, status, code, msg, _ := s.validateSigV4WithService(r, "databrew"); !ok {
		respondDataBrewError(w, &dbwError{status, code, msg})
		return true
	}
	out, err := s.dataBrewRoute(r)
	if err != nil {
		respondDataBrewError(w, err)
		return true
	}
	raw, _ := json.Marshal(out)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(raw)
	return true
}

func isDataBrewCandidate(r *http.Request) bool {
	root, _, _ := strings.Cut(strings.TrimPrefix(rawRequestPath(r), "/"), "/")
	if !dataBrewRoots[root] {
		return false
	}
	service := strings.TrimSpace(sigV4ServiceHint(r))
	host := strings.ToLower(strings.TrimSpace(r.Host))
	return service == "databrew" || (service == "" && (strings.HasPrefix(host, "databrew.") || strings.Contains(host, ".databrew.")))
}

func respondDataBrewError(w http.ResponseWriter, e *dbwError) {
	raw, _ := json.Marshal(map[string]any{"Message": e.msg})
	w.Header().Set("X-Amzn-ErrorType", e.code)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(e.status)
	_, _ = w.Write(raw)
}

// dbwDecode reads the JSON body and returns it conformed to the operation's
// input shape (unknown members are rejected, so callers can't set internal
// fields).
func dbwDecode(r *http.Request, op string) (map[string]any, *dbwError) {
	raw, err := readBodyBytes(r)
	if err != nil {
		return nil, dbwValidation("unable to read request body")
	}
	in := map[string]any{}
	if len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, &in); err != nil {
			return nil, dbwValidation("request body is not a JSON object")
		}
	}
	m := awsmodels.DataBrewModel()
	shape := m.Operations[op].Input
	if problems := m.Validate(shape, in); len(problems) > 0 {
		return nil, dbwValidation("%s", problems[0])
	}
	conformed, _ := m.Conform(shape, in).(map[string]any)
	if conformed == nil {
		conformed = map[string]any{}
	}
	return conformed, nil
}

func (s *Server) dataBrewRoute(r *http.Request) (any, *dbwError) {
	path := strings.TrimPrefix(rawRequestPath(r), "/")
	q := r.URL.Query()
	m := r.Method
	st := s.dataBrew
	if rest, ok := strings.CutPrefix(path, "tags/"); ok {
		arn, err := url.PathUnescape(rest)
		if err != nil || arn == "" {
			return nil, dbwValidation("ResourceArn is not valid.")
		}
		switch m {
		case http.MethodGet:
			return st.listTags(arn)
		case http.MethodPost:
			in, err := dbwDecode(r, "TagResource")
			if err != nil {
				return nil, err
			}
			return map[string]any{}, st.tag(arn, dbwTagMap(in["Tags"]))
		case http.MethodDelete:
			return map[string]any{}, st.untag(arn, q["tagKeys"])
		}
		return nil, dbwUnknown(r)
	}
	segs := strings.Split(path, "/")
	for i, seg := range segs {
		v, err := url.PathUnescape(seg)
		if err != nil {
			return nil, dbwValidation("the request path is not valid")
		}
		segs[i] = v
	}
	n := len(segs)
	name := ""
	if n > 1 {
		name = segs[1]
	}
	switch segs[0] {
	case "datasets":
		switch {
		case n == 1 && m == http.MethodPost:
			return s.dbwPutDataset("", r)
		case n == 1 && m == http.MethodGet:
			return st.list("dataset", q, nil)
		case n == 2 && m == http.MethodGet:
			return st.describe("dataset", name)
		case n == 2 && m == http.MethodPut:
			return s.dbwPutDataset(name, r)
		case n == 2 && m == http.MethodDelete:
			return st.deleteDataset(name)
		}
	case "recipes":
		switch {
		case n == 1 && m == http.MethodPost:
			in, err := dbwDecode(r, "CreateRecipe")
			if err != nil {
				return nil, err
			}
			return st.createRecipe(in)
		case n == 1 && m == http.MethodGet:
			return st.listRecipes(q)
		case n == 2 && m == http.MethodGet:
			return st.describeRecipe(name, q.Get("recipeVersion"))
		case n == 2 && m == http.MethodPut:
			in, err := dbwDecode(r, "UpdateRecipe")
			if err != nil {
				return nil, err
			}
			return st.updateRecipe(name, in)
		case n == 3 && segs[2] == "publishRecipe" && m == http.MethodPost:
			in, err := dbwDecode(r, "PublishRecipe")
			if err != nil {
				return nil, err
			}
			return st.publishRecipe(name, dbwStr(in, "Description"))
		case n == 3 && segs[2] == "batchDeleteRecipeVersion" && m == http.MethodPost:
			in, err := dbwDecode(r, "BatchDeleteRecipeVersion")
			if err != nil {
				return nil, err
			}
			return st.batchDeleteRecipeVersions(name, dbwStrings(in["RecipeVersions"]))
		case n == 4 && segs[2] == "recipeVersion" && m == http.MethodDelete:
			return st.deleteRecipeVersion(name, segs[3])
		}
	case "recipeVersions":
		if n == 1 && m == http.MethodGet {
			return st.listRecipeVersions(q)
		}
	case "projects":
		switch {
		case n == 1 && m == http.MethodPost:
			in, err := dbwDecode(r, "CreateProject")
			if err != nil {
				return nil, err
			}
			return st.createProject(in)
		case n == 1 && m == http.MethodGet:
			return st.list("project", q, nil)
		case n == 2 && m == http.MethodGet:
			return st.describe("project", name)
		case n == 2 && m == http.MethodPut:
			in, err := dbwDecode(r, "UpdateProject")
			if err != nil {
				return nil, err
			}
			return st.updateProject(name, in)
		case n == 2 && m == http.MethodDelete:
			return st.deleteProject(name)
		case n == 3 && segs[2] == "startProjectSession" && m == http.MethodPut:
			in, err := dbwDecode(r, "StartProjectSession")
			if err != nil {
				return nil, err
			}
			return st.startSession(name, dbwBool(in, "AssumeControl"))
		case n == 3 && segs[2] == "sendProjectSessionAction" && m == http.MethodPut:
			in, err := dbwDecode(r, "SendProjectSessionAction")
			if err != nil {
				return nil, err
			}
			return s.dbwSessionAction(r, name, in)
		}
	case "profileJobs", "recipeJobs":
		kind := "PROFILE"
		if segs[0] == "recipeJobs" {
			kind = "RECIPE"
		}
		op := map[bool]string{true: "Create", false: "Update"}[n == 1] + map[string]string{"PROFILE": "ProfileJob", "RECIPE": "RecipeJob"}[kind]
		if (n == 1 && m == http.MethodPost) || (n == 2 && m == http.MethodPut) {
			in, err := dbwDecode(r, op)
			if err != nil {
				return nil, err
			}
			return s.dbwPutJob(kind, name, in)
		}
	case "jobs":
		switch {
		case n == 1 && m == http.MethodGet:
			return st.list("job", q, st.jobFilter(q))
		case n == 2 && m == http.MethodGet:
			return st.describe("job", name)
		case n == 2 && m == http.MethodDelete:
			return st.deleteJob(name)
		case n == 3 && segs[2] == "startJobRun" && m == http.MethodPost:
			return s.dbwStartJobRun(r, name)
		case n == 3 && segs[2] == "jobRuns" && m == http.MethodGet:
			return st.listRuns(name, q)
		case n == 4 && segs[2] == "jobRun" && m == http.MethodGet:
			return st.describeRun(name, segs[3])
		case n == 5 && segs[2] == "jobRun" && segs[4] == "stopJobRun" && m == http.MethodPost:
			return st.stopRun(name, segs[3])
		}
	case "schedules":
		switch {
		case n == 1 && m == http.MethodPost:
			in, err := dbwDecode(r, "CreateSchedule")
			if err != nil {
				return nil, err
			}
			return st.putSchedule("", in)
		case n == 1 && m == http.MethodGet:
			return st.list("schedule", q, st.scheduleFilter(q))
		case n == 2 && m == http.MethodGet:
			return st.describe("schedule", name)
		case n == 2 && m == http.MethodPut:
			in, err := dbwDecode(r, "UpdateSchedule")
			if err != nil {
				return nil, err
			}
			return st.putSchedule(name, in)
		case n == 2 && m == http.MethodDelete:
			return st.remove("schedule", name)
		}
	case "rulesets":
		switch {
		case n == 1 && m == http.MethodPost:
			in, err := dbwDecode(r, "CreateRuleset")
			if err != nil {
				return nil, err
			}
			return st.putRuleset("", in)
		case n == 1 && m == http.MethodGet:
			return st.list("ruleset", q, st.rulesetFilter(q))
		case n == 2 && m == http.MethodGet:
			return st.describe("ruleset", name)
		case n == 2 && m == http.MethodPut:
			in, err := dbwDecode(r, "UpdateRuleset")
			if err != nil {
				return nil, err
			}
			return st.putRuleset(name, in)
		case n == 2 && m == http.MethodDelete:
			return st.deleteRuleset(name)
		}
	}
	return nil, dbwUnknown(r)
}

func dbwUnknown(r *http.Request) *dbwError {
	return &dbwError{http.StatusNotFound, "UnknownOperationException", "no DataBrew operation matches " + r.Method + " " + rawRequestPath(r)}
}

func dbwMaxResults(q url.Values) (int, *dbwError) {
	raw := q.Get("maxResults")
	if raw == "" {
		return 100, nil
	}
	v, err := strconv.ParseInt(raw, 10, 32)
	if err != nil || v < 1 || v > 100 {
		return 0, dbwValidation("maxResults must be between 1 and 100.")
	}
	return int(v), nil
}

func dbwStr(m map[string]any, key string) string {
	v, _ := m[key].(string)
	return v
}

func dbwBool(m map[string]any, key string) bool {
	v, _ := m[key].(bool)
	return v
}

func dbwMap(m map[string]any, key string) map[string]any {
	v, _ := m[key].(map[string]any)
	return v
}

func dbwStrings(v any) []string {
	items, _ := v.([]any)
	out := make([]string, 0, len(items))
	for _, item := range items {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func dbwTagMap(v any) map[string]string {
	obj, _ := v.(map[string]any)
	out := make(map[string]string, len(obj))
	for k, val := range obj {
		if s, ok := val.(string); ok {
			out[k] = s
		}
	}
	return out
}
