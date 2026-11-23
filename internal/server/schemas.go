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

// schMaxBody bounds a request body: GetDiscoveredSchema carries up to ten
// 100,000-character events.
const schMaxBody = 2 << 20

func isSchemasRESTCandidate(r *http.Request) bool {
	service := strings.TrimSpace(sigV4ServiceHint(r))
	host := strings.ToLower(strings.TrimSpace(r.Host))
	if service != "schemas" && !strings.HasPrefix(host, "schemas.") {
		return false
	}
	seg := splitPathSegments(rawRequestPath(r))
	if len(seg) == 0 {
		return false
	}
	if seg[0] == "tags" {
		return true
	}
	if seg[0] != "v1" || len(seg) < 2 {
		return false
	}
	switch seg[1] {
	case "registries", "discoverers", "discover", "policy":
		return true
	}
	return false
}

func (s *Server) handleSchemasRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isSchemasRESTCandidate(r) {
		return false
	}
	// A declared length bounds what net/http reads, so require one.
	if r.ContentLength < 0 || r.ContentLength > schMaxBody {
		respondSchError(w, schBadRequest("request body must declare a Content-Length of at most 2 MiB"))
		return true
	}
	if ok, status, code, msg, _ := s.validateSigV4WithService(r, "schemas"); !ok {
		respondSchError(w, &schError{status: status, code: code, msg: msg})
		return true
	}
	seg := splitPathSegments(rawRequestPath(r))
	for i, v := range seg {
		u, err := url.PathUnescape(v)
		if err != nil {
			respondSchError(w, schBadRequest("invalid path segment"))
			return true
		}
		seg[i] = u
	}
	if s.schServeSource(w, r, seg) {
		return true
	}
	status, out, err := s.schRoute(r, seg)
	if err != nil {
		respondSchError(w, err)
		return true
	}
	respondSchJSON(w, status, out)
	return true
}

// schServeSource answers GetCodeBindingSource, whose response body is the
// generated zip archive itself.
func (s *Server) schServeSource(w http.ResponseWriter, r *http.Request, seg []string) bool {
	// v1/registries/name/{registry}/schemas/name/{schema}/language/{language}/source
	if r.Method != http.MethodGet || len(seg) != 10 || seg[0] != "v1" || seg[1] != "registries" || seg[2] != "name" ||
		seg[4] != "schemas" || seg[5] != "name" || seg[7] != "language" || seg[9] != "source" {
		return false
	}
	body, err := s.schemas.codeBindingSource(seg[3], seg[6], seg[8], r.URL.Query().Get("schemaVersion"))
	if err != nil {
		respondSchError(w, err)
		return true
	}
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
	return true
}

func (s *Server) schRoute(r *http.Request, seg []string) (int, map[string]any, *schError) {
	q := r.URL.Query()
	m, n := r.Method, len(seg)
	ok := func(out map[string]any, err *schError) (int, map[string]any, *schError) {
		return http.StatusOK, out, err
	}
	created := func(out map[string]any, err *schError) (int, map[string]any, *schError) {
		return http.StatusCreated, out, err
	}
	noContent := func(err *schError) (int, map[string]any, *schError) { return http.StatusNoContent, nil, err }
	limit, token, perr := schPaging(q)

	if seg[0] == "tags" && n >= 2 {
		arn := strings.Join(seg[1:], "/")
		switch m {
		case http.MethodGet:
			return ok(s.schemas.listTags(arn))
		case http.MethodPost:
			in, err := schDecode(r, "TagResource")
			if err != nil {
				return 0, nil, err
			}
			return noContent(s.schemas.tag(arn, schTags(in)))
		case http.MethodDelete:
			keys := q["tagKeys"]
			if len(keys) == 0 {
				return 0, nil, schBadRequest("tagKeys is required.")
			}
			if len(keys) > schMaxTags {
				return 0, nil, schBadRequest("At most %d tag keys can be removed at once.", schMaxTags)
			}
			return noContent(s.schemas.untag(arn, keys))
		}
		return 0, nil, schUnknown(r)
	}

	switch {
	case seg[1] == "discover" && n == 2 && m == http.MethodPost:
		in, err := schDecode(r, "GetDiscoveredSchema")
		if err != nil {
			return 0, nil, err
		}
		content, derr := schDiscover(schStringList(in["Events"]), schStr(in, "Type"))
		if derr != nil {
			return 0, nil, derr
		}
		return ok(map[string]any{"Content": content}, nil)

	case seg[1] == "policy" && n == 2:
		registry := q.Get("registryName")
		switch m {
		case http.MethodGet:
			return ok(s.schemas.getPolicy(registry))
		case http.MethodPut:
			in, err := schDecode(r, "PutResourcePolicy")
			if err != nil {
				return 0, nil, err
			}
			return ok(s.schemas.putPolicy(registry, schStr(in, "Policy"), schStr(in, "RevisionId")))
		case http.MethodDelete:
			return noContent(s.schemas.deletePolicy(registry))
		}

	case seg[1] == "discoverers" && n == 2:
		switch m {
		case http.MethodGet:
			if perr != nil {
				return 0, nil, perr
			}
			return ok(s.schemas.listDiscoverers(q.Get("discovererIdPrefix"), q.Get("sourceArnPrefix"), token, limit))
		case http.MethodPost:
			in, err := schDecode(r, "CreateDiscoverer")
			if err != nil {
				return 0, nil, err
			}
			source := schStr(in, "SourceArn")
			bus, berr := s.eventbridge.DescribeEventBus(source)
			if !strings.HasPrefix(source, "arn:aws:events:") || berr != nil {
				return 0, nil, schBadRequest("SourceArn %s isn't an existing EventBridge event bus.", source)
			}
			cross := true
			if v, ok := in["CrossAccount"].(bool); ok {
				cross = v
			}
			return created(s.schemas.createDiscoverer(source, bus.Name, schStr(in, "Description"), cross, schTags(in)))
		}
	case seg[1] == "discoverers" && n >= 4 && seg[2] == "id":
		id := seg[3]
		switch {
		case n == 4 && m == http.MethodGet:
			return ok(s.schemas.describeDiscoverer(id))
		case n == 4 && m == http.MethodPut:
			in, err := schDecode(r, "UpdateDiscoverer")
			if err != nil {
				return 0, nil, err
			}
			var cross *bool
			if v, ok := in["CrossAccount"].(bool); ok {
				cross = &v
			}
			return ok(s.schemas.updateDiscoverer(id, schOptStr(in, "Description"), cross))
		case n == 4 && m == http.MethodDelete:
			return noContent(s.schemas.deleteDiscoverer(id))
		case n == 5 && m == http.MethodPost && seg[4] == "start":
			return ok(s.schemas.setDiscovererState(id, "STARTED"))
		case n == 5 && m == http.MethodPost && seg[4] == "stop":
			return ok(s.schemas.setDiscovererState(id, "STOPPED"))
		}

	case seg[1] == "registries" && n == 2 && m == http.MethodGet:
		if perr != nil {
			return 0, nil, perr
		}
		return ok(s.schemas.listRegistries(q.Get("registryNamePrefix"), q.Get("scope"), token, limit))
	case seg[1] == "registries" && n >= 4 && seg[2] == "name":
		return s.schRegistryRoute(r, seg[3], seg[4:], limit, token, perr)
	}
	return 0, nil, schUnknown(r)
}

// schRegistryRoute serves /v1/registries/name/{registryName}/...; rest is the
// path after the registry name.
func (s *Server) schRegistryRoute(r *http.Request, registry string, rest []string, limit int, token string, perr *schError) (int, map[string]any, *schError) {
	q := r.URL.Query()
	m, n := r.Method, len(rest)
	ok := func(out map[string]any, err *schError) (int, map[string]any, *schError) {
		return http.StatusOK, out, err
	}
	version := q.Get("schemaVersion")
	switch {
	case n == 0 && m == http.MethodPost:
		in, err := schDecode(r, "CreateRegistry")
		if err != nil {
			return 0, nil, err
		}
		out, cerr := s.schemas.createRegistry(registry, schStr(in, "Description"), schTags(in))
		return http.StatusCreated, out, cerr
	case n == 0 && m == http.MethodGet:
		return ok(s.schemas.describeRegistry(registry))
	case n == 0 && m == http.MethodPut:
		in, err := schDecode(r, "UpdateRegistry")
		if err != nil {
			return 0, nil, err
		}
		return ok(s.schemas.updateRegistry(registry, schOptStr(in, "Description")))
	case n == 0 && m == http.MethodDelete:
		return http.StatusNoContent, nil, s.schemas.deleteRegistry(registry)
	case n == 1 && rest[0] == "schemas" && m == http.MethodGet:
		if perr != nil {
			return 0, nil, perr
		}
		return ok(s.schemas.listSchemas(registry, q.Get("schemaNamePrefix"), token, limit))
	case n == 2 && rest[0] == "schemas" && rest[1] == "search" && m == http.MethodGet:
		if perr != nil {
			return 0, nil, perr
		}
		return ok(s.schemas.searchSchemas(registry, q.Get("keywords"), token, limit))
	case n < 3 || rest[0] != "schemas" || rest[1] != "name":
		return 0, nil, schUnknown(r)
	}
	name, tail := rest[2], rest[3:]
	switch {
	case len(tail) == 0 && m == http.MethodPost:
		in, err := schDecode(r, "CreateSchema")
		if err != nil {
			return 0, nil, err
		}
		out, cerr := s.schemas.createSchema(registry, name, schStr(in, "Type"), schStr(in, "Content"), schStr(in, "Description"), schTags(in))
		return http.StatusCreated, out, cerr
	case len(tail) == 0 && m == http.MethodGet:
		return ok(s.schemas.describeSchema(registry, name, version))
	case len(tail) == 0 && m == http.MethodPut:
		in, err := schDecode(r, "UpdateSchema")
		if err != nil {
			return 0, nil, err
		}
		return ok(s.schemas.updateSchema(registry, name, schOptStr(in, "Type"), schOptStr(in, "Content"), schOptStr(in, "Description")))
	case len(tail) == 0 && m == http.MethodDelete:
		return http.StatusNoContent, nil, s.schemas.deleteSchema(registry, name)
	case len(tail) == 1 && tail[0] == "versions" && m == http.MethodGet:
		if perr != nil {
			return 0, nil, perr
		}
		return ok(s.schemas.listSchemaVersions(registry, name, token, limit))
	case len(tail) == 1 && tail[0] == "export" && m == http.MethodGet:
		return ok(s.schemas.exportSchema(registry, name, version, q.Get("type")))
	case len(tail) == 2 && tail[0] == "version" && m == http.MethodDelete:
		return http.StatusNoContent, nil, s.schemas.deleteSchemaVersion(registry, name, tail[1])
	case len(tail) == 2 && tail[0] == "language" && m == http.MethodPost:
		out, err := s.schemas.putCodeBinding(registry, name, tail[1], version)
		return http.StatusAccepted, out, err
	case len(tail) == 2 && tail[0] == "language" && m == http.MethodGet:
		return ok(s.schemas.describeCodeBinding(registry, name, tail[1], version))
	}
	return 0, nil, schUnknown(r)
}

func schUnknown(r *http.Request) *schError {
	return schNotFound("no EventBridge Schemas operation matches %s %s", r.Method, rawRequestPath(r))
}

// schDecode reads a JSON body and validates it against the operation's input
// shape (URI and query members are skipped by the validator).
func schDecode(r *http.Request, op string) (map[string]any, *schError) {
	raw, err := readBodyBytes(r)
	if err != nil {
		return nil, schBadRequest("unable to read request body")
	}
	in := map[string]any{}
	if len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, &in); err != nil {
			return nil, schBadRequest("request body is not a JSON object")
		}
	}
	// Tags travel as "tags" on the wire; the model names the member Tags.
	if v, ok := in["tags"]; ok {
		delete(in, "tags")
		in["Tags"] = v
	}
	m := awsmodels.SchemasModel()
	if problems := m.Validate(m.Operations[op].Input, in); len(problems) > 0 {
		return nil, schBadRequest("%s", strings.Join(problems, "; "))
	}
	return in, nil
}

func schPaging(q url.Values) (int, string, *schError) {
	limit := 100
	if raw := q.Get("limit"); raw != "" {
		n, err := strconv.ParseInt(raw, 10, 32)
		if err != nil || n < 1 || n > 100 {
			return 0, "", schBadRequest("Limit must be between 1 and 100.")
		}
		limit = int(n)
	}
	return limit, q.Get("nextToken"), nil
}

func schStr(in map[string]any, key string) string {
	v, _ := in[key].(string)
	return v
}

func schOptStr(in map[string]any, key string) *string {
	if v, ok := in[key].(string); ok {
		return &v
	}
	return nil
}

func schStringList(v any) []string {
	list, _ := v.([]any)
	out := make([]string, 0, len(list))
	for _, e := range list {
		s, _ := e.(string)
		out = append(out, s)
	}
	return out
}

func schTags(in map[string]any) map[string]string {
	out := map[string]string{}
	if raw, ok := in["Tags"].(map[string]any); ok {
		for k, v := range raw {
			out[k], _ = v.(string)
		}
	}
	return out
}

func respondSchJSON(w http.ResponseWriter, status int, body map[string]any) {
	w.Header().Set("Content-Type", "application/json")
	if status == http.StatusNoContent {
		w.WriteHeader(status)
		return
	}
	raw, _ := json.Marshal(body)
	w.WriteHeader(status)
	_, _ = w.Write(raw)
}

func respondSchError(w http.ResponseWriter, e *schError) {
	raw, _ := json.Marshal(map[string]any{"Code": e.code, "Message": e.msg})
	w.Header().Set("X-Amzn-ErrorType", e.code)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(e.status)
	_, _ = w.Write(raw)
}
