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

// r53pMaxBody bounds a request body; Profiles requests are small.
const r53pMaxBody = 256 << 10

var r53pPathRoots = map[string]bool{
	"profile": true, "profiles": true, "profileassociation": true, "profileassociations": true,
	"profileresourceassociation": true, "profileresourceassociations": true, "tags": true,
}

func isRoute53ProfilesRESTCandidate(r *http.Request) bool {
	service := strings.TrimSpace(sigV4ServiceHint(r))
	host := strings.ToLower(strings.TrimSpace(r.Host))
	if service != "route53profiles" && !strings.HasPrefix(host, "route53profiles.") {
		return false
	}
	segments := splitPathSegments(rawRequestPath(r))
	return len(segments) > 0 && r53pPathRoots[segments[0]]
}

func (s *Server) handleRoute53ProfilesRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isRoute53ProfilesRESTCandidate(r) {
		return false
	}
	// A declared length bounds what net/http reads, so require one.
	if r.ContentLength < 0 || r.ContentLength > r53pMaxBody {
		respondR53PError(w, r53pValidation("request body must declare a Content-Length of at most 256 KiB"))
		return true
	}
	if ok, status, code, msg, _ := s.validateSigV4WithService(r, "route53profiles"); !ok {
		respondR53PError(w, &r53pError{status: status, code: code, msg: msg})
		return true
	}
	seg := splitPathSegments(rawRequestPath(r))
	for i, v := range seg {
		u, err := url.PathUnescape(v)
		if err != nil {
			respondR53PError(w, r53pValidation("invalid path segment"))
			return true
		}
		seg[i] = u
	}
	status, out, err := s.r53pRoute(r, seg)
	if err != nil {
		respondR53PError(w, err)
		return true
	}
	respondR53PJSON(w, status, out)
	return true
}

func (s *Server) r53pRoute(r *http.Request, seg []string) (int, map[string]any, *r53pError) {
	q := r.URL.Query()
	m, n := r.Method, len(seg)
	ok := func(out map[string]any, err *r53pError) (int, map[string]any, *r53pError) {
		return http.StatusOK, out, err
	}
	switch {
	case seg[0] == "profile" && n == 1 && m == http.MethodPost:
		in, err := r53pDecode(r, "CreateProfile")
		if err != nil {
			return 0, nil, err
		}
		tags, _ := in["Tags"].([]any)
		return ok(s.route53Profiles.createProfile(r53pStr(in, "Name"), r53pStr(in, "ClientToken"), tags))
	case seg[0] == "profile" && n == 2 && m == http.MethodGet:
		if err := r53pCheckID("ProfileId", seg[1]); err != nil {
			return 0, nil, err
		}
		return ok(s.route53Profiles.getProfile(seg[1]))
	case seg[0] == "profile" && n == 2 && m == http.MethodDelete:
		if err := r53pCheckID("ProfileId", seg[1]); err != nil {
			return 0, nil, err
		}
		return ok(s.route53Profiles.deleteProfile(seg[1]))
	case seg[0] == "profiles" && n == 1 && m == http.MethodGet:
		token, max, err := r53pPaging(q)
		if err != nil {
			return 0, nil, err
		}
		return ok(s.route53Profiles.listProfiles(token, max))

	case seg[0] == "profileassociation" && n == 1 && m == http.MethodPost:
		in, err := r53pDecode(r, "AssociateProfile")
		if err != nil {
			return 0, nil, err
		}
		vpcID := r53pStr(in, "ResourceId")
		if err := s.r53pCheckVPC(vpcID); err != nil {
			return 0, nil, err
		}
		tags, _ := in["Tags"].([]any)
		return ok(s.route53Profiles.associateProfile(r53pStr(in, "ProfileId"), vpcID, r53pStr(in, "Name"), r53pInt(in, "Priority"), tags))
	case seg[0] == "profileassociation" && n == 2 && (m == http.MethodGet || m == http.MethodPatch):
		if err := r53pCheckID("ProfileAssociationId", seg[1]); err != nil {
			return 0, nil, err
		}
		if m == http.MethodGet {
			return ok(s.route53Profiles.getAssoc(seg[1]))
		}
		in, err := r53pDecode(r, "UpdateProfileAssociation")
		if err != nil {
			return 0, nil, err
		}
		return ok(s.route53Profiles.updateAssoc(seg[1], r53pOptStr(in, "Name"), r53pInt(in, "Priority")))
	case seg[0] == "profileassociation" && n == 5 && m == http.MethodDelete && seg[1] == "Profileid" && seg[3] == "resourceid":
		if err := r53pCheckID("ProfileId", seg[2]); err != nil {
			return 0, nil, err
		}
		if err := r53pCheckID("ResourceId", seg[4]); err != nil {
			return 0, nil, err
		}
		return ok(s.route53Profiles.disassociateProfile(seg[2], seg[4]))
	case seg[0] == "profileassociations" && n == 1 && m == http.MethodGet:
		token, max, err := r53pPaging(q)
		if err != nil {
			return 0, nil, err
		}
		for _, k := range []string{"profileId", "resourceId"} {
			if v := q.Get(k); v != "" {
				if err := r53pCheckID(k, v); err != nil {
					return 0, nil, err
				}
			}
		}
		return ok(s.route53Profiles.listAssocs(q.Get("profileId"), q.Get("resourceId"), token, max))

	case seg[0] == "profileresourceassociation" && n == 1 && m == http.MethodPost:
		in, err := r53pDecode(r, "AssociateResourceToProfile")
		if err != nil {
			return 0, nil, err
		}
		arn, props := r53pStr(in, "ResourceArn"), r53pStr(in, "ResourceProperties")
		typ := r53pResourceType(arn)
		if typ == "" {
			return 0, nil, r53pInvalid("ResourceArn", "ResourceArn %s isn't a private hosted zone, Resolver rule, query log configuration, DNS Firewall rule group or interface VPC endpoint.", arn)
		}
		if err := r53pCheckProperties(typ, props); err != nil {
			return 0, nil, err
		}
		return ok(s.route53Profiles.associateResource(r53pStr(in, "ProfileId"), arn, typ, r53pStr(in, "Name"), props))
	case seg[0] == "profileresourceassociation" && n == 2 && (m == http.MethodGet || m == http.MethodPatch):
		if err := r53pCheckID("ProfileResourceAssociationId", seg[1]); err != nil {
			return 0, nil, err
		}
		if m == http.MethodGet {
			return ok(s.route53Profiles.getResAssoc(seg[1]))
		}
		in, err := r53pDecode(r, "UpdateProfileResourceAssociation")
		if err != nil {
			return 0, nil, err
		}
		return ok(s.route53Profiles.updateResAssoc(seg[1], r53pOptStr(in, "Name"), r53pOptStr(in, "ResourceProperties")))
	case seg[0] == "profileresourceassociation" && n >= 5 && m == http.MethodDelete && seg[1] == "profileid" && seg[3] == "resourcearn":
		if err := r53pCheckID("ProfileId", seg[2]); err != nil {
			return 0, nil, err
		}
		// The ARN label may arrive percent-encoded or with raw slashes.
		return ok(s.route53Profiles.disassociateResource(seg[2], strings.Join(seg[4:], "/")))
	case seg[0] == "profileresourceassociations" && n == 3 && m == http.MethodGet && seg[1] == "profileid":
		if err := r53pCheckID("ProfileId", seg[2]); err != nil {
			return 0, nil, err
		}
		token, max, err := r53pPaging(q)
		if err != nil {
			return 0, nil, err
		}
		return ok(s.route53Profiles.listResAssocs(seg[2], q.Get("resourceType"), q.Get("resourceArn"), token, max))

	case seg[0] == "tags" && n >= 2:
		arn := strings.Join(seg[1:], "/")
		switch m {
		case http.MethodGet:
			return ok(s.route53Profiles.listTags(arn))
		case http.MethodPost:
			in, err := r53pDecode(r, "TagResource")
			if err != nil {
				return 0, nil, err
			}
			tags := map[string]string{}
			if raw, ok := in["Tags"].(map[string]any); ok {
				for k, v := range raw {
					tags[k], _ = v.(string)
				}
			}
			return http.StatusNoContent, nil, s.route53Profiles.tag(arn, tags)
		case http.MethodDelete:
			keys := q["tagKeys"]
			if len(keys) == 0 {
				return 0, nil, r53pValidation("tagKeys is required.")
			}
			if len(keys) > r53pMaxTags {
				return 0, nil, r53pValidation("At most %d tag keys can be removed at once.", r53pMaxTags)
			}
			return http.StatusNoContent, nil, s.route53Profiles.untag(arn, keys)
		}
	}
	return 0, nil, &r53pError{status: http.StatusNotFound, code: "UnknownOperationException", msg: "no Route 53 Profiles operation matches " + r.Method + " " + rawRequestPath(r)}
}

// r53pCheckVPC requires a VPC that exists in the EC2 store. It runs before
// the Profiles lock is taken, so the two stores never lock in turn.
func (s *Server) r53pCheckVPC(vpcID string) *r53pError {
	if !strings.HasPrefix(vpcID, "vpc-") {
		return r53pInvalid("ResourceId", "ResourceId %s isn't a VPC ID.", vpcID)
	}
	if len(s.ec2.DescribeVpcs([]string{vpcID})) == 0 {
		return r53pNotFound("VPC", "VPC %s does not exist.", vpcID)
	}
	return nil
}

// r53pDecode reads a JSON body and validates it against the operation's
// input shape (URI and query members are skipped by the validator).
func r53pDecode(r *http.Request, op string) (map[string]any, *r53pError) {
	raw, err := readBodyBytes(r)
	if err != nil {
		return nil, r53pValidation("unable to read request body")
	}
	in := map[string]any{}
	if len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, &in); err != nil {
			return nil, r53pValidation("request body is not a JSON object")
		}
	}
	m := awsmodels.Route53ProfilesModel()
	if problems := m.Validate(m.Operations[op].Input, in); len(problems) > 0 {
		return nil, r53pValidation("%s", strings.Join(problems, "; "))
	}
	return in, nil
}

func r53pCheckID(field, v string) *r53pError {
	if problems := awsmodels.Route53ProfilesModel().Validate("ResourceId", v); len(problems) > 0 {
		_, reason, _ := strings.Cut(problems[0], ": ")
		return r53pValidation("%s %s.", field, reason)
	}
	return nil
}

func r53pPaging(q url.Values) (string, int, *r53pError) {
	max := 100
	if raw := q.Get("maxResults"); raw != "" {
		n, err := strconv.ParseInt(raw, 10, 32)
		if err != nil || n < 1 || n > 100 {
			return "", 0, r53pValidation("maxResults must be between 1 and 100.")
		}
		max = int(n)
	}
	return q.Get("nextToken"), max, nil
}

func r53pStr(in map[string]any, key string) string {
	v, _ := in[key].(string)
	return v
}

func r53pOptStr(in map[string]any, key string) *string {
	if v, ok := in[key].(string); ok {
		return &v
	}
	return nil
}

// r53pInt reads an integer member the validator already range-checked.
func r53pInt(in map[string]any, key string) int {
	v, _ := in[key].(float64)
	return int(v)
}

func respondR53PJSON(w http.ResponseWriter, status int, body map[string]any) {
	w.Header().Set("Content-Type", "application/json")
	if status == http.StatusNoContent {
		w.WriteHeader(status)
		return
	}
	raw, _ := json.Marshal(body)
	w.WriteHeader(status)
	_, _ = w.Write(raw)
}

func respondR53PError(w http.ResponseWriter, e *r53pError) {
	body := map[string]any{"Message": e.msg}
	if e.fieldName != "" {
		body["FieldName"] = e.fieldName
	}
	if e.resourceType != "" {
		body["ResourceType"] = e.resourceType
	}
	raw, _ := json.Marshal(body)
	w.Header().Set("X-Amzn-ErrorType", e.code)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(e.status)
	_, _ = w.Write(raw)
}
