package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

// pipesMaxBody bounds a request body; pipe definitions are a few KiB.
const pipesMaxBody = 1 << 20

func (s *Server) handlePipesRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isPipesRESTCandidate(r) {
		return false
	}
	// A declared length bounds what net/http reads, so require one.
	if r.ContentLength < 0 || r.ContentLength > pipesMaxBody {
		respondPipesError(w, pipesValidation("request body must declare a Content-Length of at most 1 MiB"))
		return true
	}
	if ok, status, code, msg, _ := s.validateSigV4WithService(r, "pipes"); !ok {
		respondPipesError(w, &pipesError{status: status, code: code, msg: msg})
		return true
	}
	segments := splitPathSegments(rawRequestPath(r))
	var out map[string]any
	var perr *pipesError
	switch {
	case len(segments) == 2 && segments[0] == "v1" && segments[1] == "pipes" && r.Method == http.MethodGet:
		out, perr = s.pipesList(r.URL.Query())
	case len(segments) == 3 && segments[0] == "v1" && segments[1] == "pipes":
		name, err := url.PathUnescape(segments[2])
		if err != nil {
			perr = pipesValidation("invalid pipe name")
			break
		}
		out, perr = s.pipesPipeRoute(r, name)
	case len(segments) == 4 && segments[0] == "v1" && segments[1] == "pipes" && r.Method == http.MethodPost && (segments[3] == "start" || segments[3] == "stop"):
		name, err := url.PathUnescape(segments[2])
		if err != nil {
			perr = pipesValidation("invalid pipe name")
			break
		}
		if perr = pipesCheckName(name); perr == nil {
			out, perr = s.pipes.setDesired(name, map[string]string{"start": "RUNNING", "stop": "STOPPED"}[segments[3]])
		}
	case len(segments) >= 2 && segments[0] == "tags":
		arn, err := url.PathUnescape(strings.Join(segments[1:], "/"))
		if err != nil {
			perr = pipesValidation("invalid resource ARN")
			break
		}
		out, perr = s.pipesTagsRoute(r, arn)
	default:
		perr = &pipesError{status: http.StatusNotFound, code: "UnknownOperationException", msg: "no Pipes operation matches " + r.Method + " " + rawRequestPath(r)}
	}
	if perr != nil {
		respondPipesError(w, perr)
		return true
	}
	respondPipesJSON(w, out)
	return true
}

func isPipesRESTCandidate(r *http.Request) bool {
	service := strings.TrimSpace(sigV4ServiceHint(r))
	host := strings.ToLower(strings.TrimSpace(r.Host))
	if service != "pipes" && !strings.HasPrefix(host, "pipes.") && !strings.Contains(host, ".pipes.") {
		return false
	}
	path := rawRequestPath(r)
	return path == "/v1/pipes" || strings.HasPrefix(path, "/v1/pipes/") || strings.HasPrefix(path, "/tags/")
}

func respondPipesJSON(w http.ResponseWriter, body map[string]any) {
	raw, _ := json.Marshal(body)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(raw)
}

func respondPipesError(w http.ResponseWriter, e *pipesError) {
	body := map[string]any{"message": e.msg}
	switch e.code {
	case "ValidationException":
		if len(e.fields) > 0 {
			body["fieldList"] = e.fields
		}
	case "ConflictException":
		if len(e.fields) > 0 {
			body["resourceId"], body["resourceType"] = e.fields[0]["resourceId"], e.fields[0]["resourceType"]
		}
	}
	raw, _ := json.Marshal(body)
	w.Header().Set("X-Amzn-ErrorType", e.code)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(e.status)
	_, _ = w.Write(raw)
}

func pipesDecodeBody(r *http.Request) (map[string]any, *pipesError) {
	raw, err := readBodyBytes(r)
	if err != nil {
		return nil, pipesValidation("unable to read request body")
	}
	in := map[string]any{}
	if len(bytes.TrimSpace(raw)) == 0 {
		return in, nil
	}
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, pipesValidation("request body is not a JSON object")
	}
	return in, nil
}

// pipesCheckValue validates one URI or query value against its model shape.
func pipesCheckValue(member, shape, value string) *pipesError {
	if problems := awsmodels.PipesModel().Validate(shape, value); len(problems) > 0 {
		_, reason, _ := strings.Cut(problems[0], ": ")
		return pipesValidation(member + " " + reason + ".")
	}
	return nil
}

func pipesCheckName(name string) *pipesError { return pipesCheckValue("Name", "PipeName", name) }

func (s *Server) pipesPipeRoute(r *http.Request, name string) (map[string]any, *pipesError) {
	if err := pipesCheckName(name); err != nil {
		return nil, err
	}
	switch r.Method {
	case http.MethodGet:
		return s.pipes.describe(name)
	case http.MethodDelete:
		return s.pipes.delete(name)
	case http.MethodPost:
		in, err := pipesDecodeBody(r)
		if err != nil {
			return nil, err
		}
		if err := pipesValidateShape("CreatePipeRequest", in); err != nil {
			return nil, err
		}
		if err := pipesCheckDefinition(pipesStr(in, "Source"), in, pipesStr(in, "Target")); err != nil {
			return nil, err
		}
		return s.pipes.create(name, in, s.pipesMissingResource(pipesStr(in, "Source"), pipesStr(in, "Enrichment"), pipesStr(in, "Target")))
	case http.MethodPut:
		in, err := pipesDecodeBody(r)
		if err != nil {
			return nil, err
		}
		if err := pipesValidateShape("UpdatePipeRequest", in); err != nil {
			return nil, err
		}
		return s.pipes.update(name, in, s.pipesMissingResource)
	}
	return nil, &pipesError{status: http.StatusNotFound, code: "UnknownOperationException", msg: "no Pipes operation matches " + r.Method + " " + rawRequestPath(r)}
}

func (s *Server) pipesTagsRoute(r *http.Request, arn string) (map[string]any, *pipesError) {
	switch r.Method {
	case http.MethodGet:
		return s.pipes.tags(arn)
	case http.MethodPost:
		in, err := pipesDecodeBody(r)
		if err != nil {
			return nil, err
		}
		if err := pipesValidateShape("TagResourceRequest", in); err != nil {
			return nil, err
		}
		return map[string]any{}, s.pipes.tag(arn, pipesStringMap(in["tags"]))
	case http.MethodDelete:
		keys := r.URL.Query()["tagKeys"]
		if len(keys) == 0 {
			return nil, pipesValidation("tagKeys is required.")
		}
		return map[string]any{}, s.pipes.untag(arn, keys)
	}
	return nil, &pipesError{status: http.StatusNotFound, code: "UnknownOperationException", msg: "no Pipes operation matches " + r.Method + " " + rawRequestPath(r)}
}

func (s *Server) pipesList(q url.Values) (map[string]any, *pipesError) {
	f := pipesListFilter{
		namePrefix: q.Get("NamePrefix"), desired: q.Get("DesiredState"), current: q.Get("CurrentState"),
		sourcePrefix: q.Get("SourcePrefix"), targetPrefix: q.Get("TargetPrefix"), nextToken: q.Get("NextToken"), limit: 100,
	}
	for _, c := range [][3]string{{"NamePrefix", "PipeName", f.namePrefix}, {"DesiredState", "RequestedPipeState", f.desired}, {"CurrentState", "PipeState", f.current}, {"NextToken", "NextToken", f.nextToken}} {
		if c[2] == "" {
			continue
		}
		if err := pipesCheckValue(c[0], c[1], c[2]); err != nil {
			return nil, err
		}
	}
	if raw := q.Get("Limit"); raw != "" {
		n, err := strconv.ParseInt(raw, 10, 32)
		if err != nil || n < 1 || n > 100 {
			return nil, pipesValidation("Limit must be between 1 and 100.")
		}
		f.limit = int(n)
	}
	return s.pipes.list(f)
}
