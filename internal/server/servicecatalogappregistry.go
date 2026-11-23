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

// aprMaxBody bounds a request body; attributes are at most 8,000 characters.
const aprMaxBody = 64 << 10

var aprPathRoots = map[string]bool{
	"applications": true, "attribute-groups": true, "configuration": true, "sync": true, "tags": true,
}

// isAppRegistryRESTCandidate claims REST paths signed for servicecatalog (or
// sent to an AppRegistry host). Service Catalog shares the signing name but is
// JSON 1.1 with X-Amz-Target, so those requests are never claimed.
func isAppRegistryRESTCandidate(r *http.Request) bool {
	if strings.TrimSpace(r.Header.Get("X-Amz-Target")) != "" {
		return false
	}
	service := strings.TrimSpace(sigV4ServiceHint(r))
	host := strings.ToLower(strings.TrimSpace(r.Host))
	if service != "servicecatalog" && !strings.HasPrefix(host, "servicecatalog-appregistry.") {
		return false
	}
	segments := splitPathSegments(rawRequestPath(r))
	return len(segments) > 0 && aprPathRoots[segments[0]]
}

func (s *Server) handleAppRegistryRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isAppRegistryRESTCandidate(r) {
		return false
	}
	// A declared length bounds what net/http reads, so require one.
	if r.ContentLength < 0 || r.ContentLength > aprMaxBody {
		respondAPRError(w, aprValidation("request body must declare a Content-Length of at most 64 KiB"))
		return true
	}
	if ok, status, code, msg, _ := s.validateSigV4WithService(r, "servicecatalog"); !ok {
		respondAPRError(w, &aprError{status: status, code: code, msg: msg})
		return true
	}
	seg := splitPathSegments(rawRequestPath(r))
	for i, v := range seg {
		u, err := url.PathUnescape(v)
		if err != nil {
			respondAPRError(w, aprValidation("invalid path segment"))
			return true
		}
		seg[i] = u
	}
	status, out, err := s.aprRoute(r, seg)
	if err != nil {
		respondAPRError(w, err)
		return true
	}
	respondAPRJSON(w, status, out)
	return true
}

func (s *Server) aprRoute(r *http.Request, seg []string) (int, map[string]any, *aprError) {
	q := r.URL.Query()
	m, n := r.Method, len(seg)
	ok := func(out map[string]any, err *aprError) (int, map[string]any, *aprError) {
		return http.StatusOK, out, err
	}
	switch {
	case seg[0] == "applications" && n == 1 && m == http.MethodPost:
		in, err := aprDecode(r, "CreateApplication")
		if err != nil {
			return 0, nil, err
		}
		name := aprStr(in, "name")
		out, created, err := s.appRegistry.createApplication(name, aprStr(in, "description"), aprStr(in, "clientToken"), aprTags(in))
		if err != nil {
			return 0, nil, err
		}
		if created {
			s.aprCreateResourceGroups(name)
		}
		return http.StatusCreated, out, nil
	case seg[0] == "applications" && n == 1 && m == http.MethodGet:
		token, max, err := aprPaging(q)
		if err != nil {
			return 0, nil, err
		}
		return ok(s.appRegistry.listApplications(token, max))
	case seg[0] == "applications" && n == 2:
		if err := aprCheckSpec("ApplicationSpecifier", seg[1]); err != nil {
			return 0, nil, err
		}
		switch m {
		case http.MethodGet:
			return ok(s.appRegistry.getApplication(seg[1]))
		case http.MethodPatch:
			in, err := aprDecode(r, "UpdateApplication")
			if err != nil {
				return 0, nil, err
			}
			return ok(s.appRegistry.updateApplication(seg[1], aprOptStr(in, "name"), aprOptStr(in, "description")))
		case http.MethodDelete:
			out, name, err := s.appRegistry.deleteApplication(seg[1])
			if err != nil {
				return 0, nil, err
			}
			s.aprDeleteResourceGroups(name)
			return http.StatusOK, out, nil
		}
	case seg[0] == "applications" && n == 3 && m == http.MethodGet && (seg[2] == "attribute-groups" || seg[2] == "attribute-group-details" || seg[2] == "resources"):
		if err := aprCheckSpec("ApplicationSpecifier", seg[1]); err != nil {
			return 0, nil, err
		}
		token, max, err := aprPaging(q)
		if err != nil {
			return 0, nil, err
		}
		if seg[2] == "resources" {
			return ok(s.appRegistry.listAssociatedResources(seg[1], token, max))
		}
		return ok(s.appRegistry.listAssociatedAttributeGroups(seg[1], token, max, seg[2] == "attribute-group-details"))
	case seg[0] == "applications" && n == 4 && seg[2] == "attribute-groups" && (m == http.MethodPut || m == http.MethodDelete):
		if err := aprCheckSpec("ApplicationSpecifier", seg[1]); err != nil {
			return 0, nil, err
		}
		if err := aprCheckSpec("AttributeGroupSpecifier", seg[3]); err != nil {
			return 0, nil, err
		}
		if m == http.MethodPut {
			return ok(s.appRegistry.associateAttributeGroup(seg[1], seg[3]))
		}
		return ok(s.appRegistry.disassociateAttributeGroup(seg[1], seg[3]))
	case seg[0] == "applications" && n >= 5 && seg[2] == "resources":
		if err := aprCheckSpec("ApplicationSpecifier", seg[1]); err != nil {
			return 0, nil, err
		}
		// A stack ARN may arrive percent-encoded or with raw slashes.
		typ, res := seg[3], strings.Join(seg[4:], "/")
		if err := aprCheckResource(typ, res); err != nil {
			return 0, nil, err
		}
		switch m {
		case http.MethodPut:
			in, err := aprDecode(r, "AssociateResource")
			if err != nil {
				return 0, nil, err
			}
			ref, err := s.aprResolveResource(typ, res, seg[1])
			if err != nil {
				return 0, nil, err
			}
			return ok(s.appRegistry.associateResource(seg[1], ref, aprList(in, "options")))
		case http.MethodDelete:
			return ok(s.appRegistry.disassociateResource(seg[1], typ, s.aprResourceKey(typ, res, seg[1])))
		case http.MethodGet:
			if _, _, err := aprPaging(q); err != nil {
				return 0, nil, err
			}
			for _, v := range q["resourceTagStatus"] {
				if !aprEnum("ResourceItemStatus", v) {
					return 0, nil, aprValidation("resourceTagStatus %q isn't a valid status.", v)
				}
			}
			return ok(s.appRegistry.getAssociatedResource(seg[1], typ, s.aprResourceKey(typ, res, seg[1])))
		}

	case seg[0] == "attribute-groups" && n == 1 && m == http.MethodPost:
		in, err := aprDecode(r, "CreateAttributeGroup")
		if err != nil {
			return 0, nil, err
		}
		out, err := s.appRegistry.createAttributeGroup(aprStr(in, "name"), aprStr(in, "description"), aprStr(in, "attributes"), aprStr(in, "clientToken"), aprTags(in))
		if err != nil {
			return 0, nil, err
		}
		return http.StatusCreated, out, nil
	case seg[0] == "attribute-groups" && n == 1 && m == http.MethodGet:
		token, max, err := aprPaging(q)
		if err != nil {
			return 0, nil, err
		}
		return ok(s.appRegistry.listAttributeGroups(token, max))
	case seg[0] == "attribute-groups" && n == 2:
		if err := aprCheckSpec("AttributeGroupSpecifier", seg[1]); err != nil {
			return 0, nil, err
		}
		switch m {
		case http.MethodGet:
			return ok(s.appRegistry.getAttributeGroup(seg[1]))
		case http.MethodPatch:
			in, err := aprDecode(r, "UpdateAttributeGroup")
			if err != nil {
				return 0, nil, err
			}
			return ok(s.appRegistry.updateAttributeGroup(seg[1], aprOptStr(in, "name"), aprOptStr(in, "description"), aprOptStr(in, "attributes")))
		case http.MethodDelete:
			return ok(s.appRegistry.deleteAttributeGroup(seg[1]))
		}

	case seg[0] == "configuration" && n == 1 && m == http.MethodGet:
		return http.StatusOK, s.appRegistry.getConfiguration(), nil
	case seg[0] == "configuration" && n == 1 && m == http.MethodPut:
		in, err := aprDecode(r, "PutConfiguration")
		if err != nil {
			return 0, nil, err
		}
		cfg, _ := in["configuration"].(map[string]any)
		tq, _ := cfg["tagQueryConfiguration"].(map[string]any)
		return http.StatusOK, map[string]any{}, s.appRegistry.putConfiguration(aprStr(tq, "tagKey"))

	case seg[0] == "sync" && n >= 3 && m == http.MethodPost:
		typ, res := seg[1], strings.Join(seg[2:], "/")
		if err := aprCheckResource(typ, res); err != nil {
			return 0, nil, err
		}
		ref, err := s.aprResolveResource(typ, res, "")
		if err != nil {
			return 0, nil, err
		}
		return http.StatusOK, s.appRegistry.syncResource(ref), nil

	case seg[0] == "tags" && n >= 2:
		arn := strings.Join(seg[1:], "/")
		if err := aprCheckSpec("Arn", arn); err != nil {
			return 0, nil, err
		}
		switch m {
		case http.MethodGet:
			return ok(s.appRegistry.listTags(arn))
		case http.MethodPost:
			in, err := aprDecode(r, "TagResource")
			if err != nil {
				return 0, nil, err
			}
			return http.StatusOK, map[string]any{}, s.appRegistry.tagResource(arn, aprTags(in))
		case http.MethodDelete:
			keys := q["tagKeys"]
			if len(keys) == 0 {
				return 0, nil, aprValidation("tagKeys is required.")
			}
			if len(keys) > aprMaxTags {
				return 0, nil, aprValidation("At most %d tag keys can be removed at once.", aprMaxTags)
			}
			return http.StatusOK, map[string]any{}, s.appRegistry.untagResource(arn, keys)
		}
	}
	return 0, nil, aprErr(http.StatusNotFound, "UnknownOperationException", "no AppRegistry operation matches %s %s", r.Method, rawRequestPath(r))
}

// aprResolveResource turns an AssociateResource/SyncResource target into its
// name and ARN. CloudFormation stacks must exist; the lookup takes the
// CloudFormation lock before (never inside) the AppRegistry lock.
func (s *Server) aprResolveResource(typ, res, appSpec string) (aprResourceRef, *aprError) {
	if typ == "CFN_STACK" {
		name, arn, found := s.cloudformation.lookupStack(res)
		if !found {
			return aprResourceRef{}, aprNotFound("Stack %s does not exist.", res)
		}
		return aprResourceRef{typ: typ, name: name, arn: arn}, nil
	}
	ref := aprResourceRef{typ: typ, name: res}
	if appSpec != "" {
		ref.arn = aprTagValueARN(s.aprAppID(appSpec), res)
	} else {
		ref.arn = res // SyncResource matches tag values by name
	}
	return ref, nil
}

// aprResourceKey is what disassociate/get match on: the stack ARN (so a name
// or an ARN both work) or the tag value.
func (s *Server) aprResourceKey(typ, res, appSpec string) string {
	if typ == "CFN_STACK" {
		if _, arn, found := s.cloudformation.lookupStack(res); found {
			return arn
		}
	}
	return res
}

func (s *Server) aprAppID(spec string) string {
	s.appRegistry.mu.Lock()
	defer s.appRegistry.mu.Unlock()
	if a, err := s.appRegistry.appLocked(spec); err == nil {
		return a.id
	}
	return spec
}

// aprCreateResourceGroups creates the two resource groups AppRegistry keeps
// for each application. It runs after the AppRegistry lock is released.
func (s *Server) aprCreateResourceGroups(app string) {
	for _, name := range []string{aprAppGroupName(app), aprAppTagGroupName(app)} {
		s.resourcegroups.Handle("CreateGroup", map[string]any{"Name": name, "Description": "Created by AWS Service Catalog AppRegistry"}, nil, nil)
	}
}

func (s *Server) aprDeleteResourceGroups(app string) {
	for _, name := range []string{aprAppGroupName(app), aprAppTagGroupName(app)} {
		s.resourcegroups.Handle("DeleteGroup", map[string]any{"Group": name}, nil, nil)
	}
}

// lookupStack resolves a stack name or stack ARN to both.
func (s *cloudFormationStore) lookupStack(nameOrARN string) (string, string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if arn, ok := s.stacks[nameOrARN]; ok {
		return nameOrARN, arn, true
	}
	for name, arn := range s.stacks {
		if arn == nameOrARN {
			return name, arn, true
		}
	}
	return "", "", false
}

// aprDecode reads a JSON body and validates it against the operation's input
// shape (URI and query members are skipped by the validator).
func aprDecode(r *http.Request, op string) (map[string]any, *aprError) {
	raw, err := readBodyBytes(r)
	if err != nil {
		return nil, aprValidation("unable to read request body")
	}
	in := map[string]any{}
	if len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, &in); err != nil {
			return nil, aprValidation("request body is not a JSON object")
		}
	}
	m := awsmodels.ServiceCatalogAppRegistryModel()
	if problems := m.Validate(m.Operations[op].Input, in); len(problems) > 0 {
		return nil, aprValidation("%s", strings.Join(problems, "; "))
	}
	return in, nil
}

func aprCheckSpec(shape, v string) *aprError {
	if problems := awsmodels.ServiceCatalogAppRegistryModel().Validate(shape, v); len(problems) > 0 {
		_, reason, _ := strings.Cut(problems[0], ": ")
		return aprValidation("%s %s.", shape, reason)
	}
	return nil
}

func aprCheckResource(typ, res string) *aprError {
	if !aprEnum("ResourceType", typ) {
		return aprValidation("resourceType must be CFN_STACK or RESOURCE_TAG_VALUE.")
	}
	return aprCheckSpec("ResourceSpecifier", res)
}

func aprEnum(shape, v string) bool {
	s := awsmodels.ServiceCatalogAppRegistryModel().Shapes[shape]
	if s == nil {
		return false
	}
	for _, e := range s.Enum {
		if e == v {
			return true
		}
	}
	return false
}

func aprPaging(q url.Values) (string, int, *aprError) {
	max := 100
	if raw := q.Get("maxResults"); raw != "" {
		n, err := strconv.ParseInt(raw, 10, 32)
		if err != nil || n < 1 || n > 100 {
			return "", 0, aprValidation("maxResults must be between 1 and 100.")
		}
		max = int(n)
	}
	token := q.Get("nextToken")
	if token != "" {
		if err := aprCheckSpec("NextToken", token); err != nil {
			return "", 0, err
		}
	}
	return token, max, nil
}

func aprStr(in map[string]any, key string) string {
	v, _ := in[key].(string)
	return v
}

func aprOptStr(in map[string]any, key string) *string {
	if v, ok := in[key].(string); ok {
		return &v
	}
	return nil
}

func aprList(in map[string]any, key string) []string {
	raw, _ := in[key].([]any)
	out := make([]string, 0, len(raw))
	for _, v := range raw {
		if s, ok := v.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func aprTags(in map[string]any) map[string]string {
	raw, _ := in["tags"].(map[string]any)
	out := make(map[string]string, len(raw))
	for k, v := range raw {
		out[k], _ = v.(string)
	}
	return out
}

func respondAPRJSON(w http.ResponseWriter, status int, body map[string]any) {
	raw, _ := json.Marshal(body)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(raw)
}

func respondAPRError(w http.ResponseWriter, e *aprError) {
	raw, _ := json.Marshal(map[string]any{"message": e.msg})
	w.Header().Set("X-Amzn-ErrorType", e.code)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(e.status)
	_, _ = w.Write(raw)
}
