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

// padMaxBody bounds a request body; template definitions are the largest.
const padMaxBody = 256 << 10

var padPathRoots = map[string]bool{"connectors": true, "directoryRegistrations": true, "templates": true, "tags": true}

func isPCAConnectorADRESTCandidate(r *http.Request) bool {
	service := strings.TrimSpace(sigV4ServiceHint(r))
	host := strings.ToLower(strings.TrimSpace(r.Host))
	if service != "pca-connector-ad" && !strings.HasPrefix(host, "pca-connector-ad.") {
		return false
	}
	seg := splitPathSegments(rawRequestPath(r))
	return len(seg) > 0 && padPathRoots[seg[0]]
}

func (s *Server) handlePCAConnectorADRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isPCAConnectorADRESTCandidate(r) {
		return false
	}
	// A declared length bounds what net/http reads, so require one.
	if r.ContentLength < 0 || r.ContentLength > padMaxBody {
		respondPADError(w, padValidation("FIELD_VALIDATION_FAILED", "request body must declare a Content-Length of at most 256 KiB"))
		return true
	}
	if ok, status, code, msg, _ := s.validateSigV4WithService(r, "pca-connector-ad"); !ok {
		respondPADError(w, &padError{status: status, code: code, msg: msg})
		return true
	}
	seg := splitPathSegments(rawRequestPath(r))
	for i, v := range seg {
		u, err := url.PathUnescape(v)
		if err != nil {
			respondPADError(w, padValidation("FIELD_VALIDATION_FAILED", "invalid path segment"))
			return true
		}
		seg[i] = u
	}
	status, out, err := s.padRoute(r, seg)
	if err != nil {
		respondPADError(w, err)
		return true
	}
	respondPADJSON(w, status, out)
	return true
}

func (s *Server) padRoute(r *http.Request, seg []string) (int, map[string]any, *padError) {
	st := s.pcaConnectorAD
	m, n := r.Method, len(seg)
	q := r.URL.Query()
	done := func(status int, out map[string]any, err *padError) (int, map[string]any, *padError) {
		return status, out, err
	}
	switch seg[0] {
	case "connectors":
		switch {
		case n == 1 && m == http.MethodPost:
			in, err := padDecode(r, "CreateConnector")
			if err != nil {
				return 0, nil, err
			}
			vpc, _ := in["VpcInformation"].(map[string]any)
			if err := s.padCheckConnectorTargets(padStr(in, "CertificateAuthorityArn"), padStr(in, "DirectoryId"), vpc); err != nil {
				return 0, nil, err
			}
			tags, _ := in["Tags"].(map[string]any)
			out, err := st.createConnector(padStr(in, "ClientToken"), padStr(in, "CertificateAuthorityArn"), padStr(in, "DirectoryId"), vpc, tags)
			return done(http.StatusAccepted, out, err)
		case n == 1 && m == http.MethodGet:
			token, max, err := padPaging(q)
			if err != nil {
				return 0, nil, err
			}
			out, err := st.listConnectors(token, max)
			return done(http.StatusOK, out, err)
		case n == 2 && (m == http.MethodGet || m == http.MethodDelete):
			if err := padCheck("ConnectorArn", "ConnectorArn", seg[1]); err != nil {
				return 0, nil, err
			}
			if m == http.MethodGet {
				out, err := st.getConnector(seg[1])
				return done(http.StatusOK, out, err)
			}
			return done(http.StatusAccepted, nil, st.deleteConnector(seg[1]))
		}
	case "directoryRegistrations":
		switch {
		case n == 1 && m == http.MethodPost:
			in, err := padDecode(r, "CreateDirectoryRegistration")
			if err != nil {
				return 0, nil, err
			}
			dirID := padStr(in, "DirectoryId")
			stage, ok := s.directoryservice.directoryStage(dirID)
			if !ok {
				return 0, nil, padNotFound("DIRECTORY", dirID)
			}
			tags, _ := in["Tags"].(map[string]any)
			out, err := st.createDirectoryRegistration(padStr(in, "ClientToken"), dirID, stage == "Active", tags)
			return done(http.StatusAccepted, out, err)
		case n == 1 && m == http.MethodGet:
			token, max, err := padPaging(q)
			if err != nil {
				return 0, nil, err
			}
			out, err := st.listDirectoryRegistrations(token, max)
			return done(http.StatusOK, out, err)
		}
		if n >= 2 {
			if err := padCheck("DirectoryRegistrationArn", "DirectoryRegistrationArn", seg[1]); err != nil {
				return 0, nil, err
			}
		}
		switch {
		case n == 2 && m == http.MethodGet:
			out, err := st.getDirectoryRegistration(seg[1])
			return done(http.StatusOK, out, err)
		case n == 2 && m == http.MethodDelete:
			return done(http.StatusAccepted, nil, st.deleteDirectoryRegistration(seg[1]))
		case n == 3 && seg[2] == "servicePrincipalNames" && m == http.MethodGet:
			token, max, err := padPaging(q)
			if err != nil {
				return 0, nil, err
			}
			out, err := st.listSPNs(seg[1], token, max)
			return done(http.StatusOK, out, err)
		case n == 4 && seg[2] == "servicePrincipalNames":
			if err := padCheck("ConnectorArn", "ConnectorArn", seg[3]); err != nil {
				return 0, nil, err
			}
			switch m {
			case http.MethodPost:
				if _, err := padDecode(r, "CreateServicePrincipalName"); err != nil {
					return 0, nil, err
				}
				return done(http.StatusAccepted, nil, st.createSPN(seg[1], seg[3]))
			case http.MethodGet:
				out, err := st.getSPN(seg[1], seg[3])
				return done(http.StatusOK, out, err)
			case http.MethodDelete:
				st.deleteSPN(seg[1], seg[3])
				return done(http.StatusAccepted, nil, nil)
			}
		}
	case "templates":
		switch {
		case n == 1 && m == http.MethodPost:
			in, err := padDecode(r, "CreateTemplate")
			if err != nil {
				return 0, nil, err
			}
			name := padStr(in, "Name")
			if err := padCheckTemplateName("Name", name); err != nil {
				return 0, nil, err
			}
			def, _ := in["Definition"].(map[string]any)
			schema, err := padCheckDefinition(def)
			if err != nil {
				return 0, nil, err
			}
			tags, _ := in["Tags"].(map[string]any)
			out, err := st.createTemplate(padStr(in, "ClientToken"), padStr(in, "ConnectorArn"), name, def, schema, tags)
			return done(http.StatusOK, out, err)
		case n == 1 && m == http.MethodGet:
			conn := q.Get("ConnectorArn")
			if err := padCheck("ConnectorArn", "ConnectorArn", conn); err != nil {
				return 0, nil, err
			}
			token, max, err := padPaging(q)
			if err != nil {
				return 0, nil, err
			}
			out, err := st.listTemplates(conn, token, max)
			return done(http.StatusOK, out, err)
		}
		if n >= 2 {
			if err := padCheck("TemplateArn", "TemplateArn", seg[1]); err != nil {
				return 0, nil, err
			}
		}
		switch {
		case n == 2 && m == http.MethodGet:
			out, err := st.getTemplate(seg[1])
			return done(http.StatusOK, out, err)
		case n == 2 && m == http.MethodDelete:
			return done(http.StatusAccepted, nil, st.deleteTemplate(seg[1]))
		case n == 2 && m == http.MethodPatch:
			in, err := padDecode(r, "UpdateTemplate")
			if err != nil {
				return 0, nil, err
			}
			def, _ := in["Definition"].(map[string]any)
			schema := 0
			if def != nil {
				if schema, err = padCheckDefinition(def); err != nil {
					return 0, nil, err
				}
			}
			reenroll, _ := in["ReenrollAllCertificateHolders"].(bool)
			return done(http.StatusOK, nil, st.updateTemplate(seg[1], def, schema, reenroll))
		case n == 3 && seg[2] == "accessControlEntries" && m == http.MethodPost:
			in, err := padDecode(r, "CreateTemplateGroupAccessControlEntry")
			if err != nil {
				return 0, nil, err
			}
			rights, _ := in["AccessRights"].(map[string]any)
			return done(http.StatusOK, nil, st.createACE(padStr(in, "ClientToken"), seg[1], padStr(in, "GroupSecurityIdentifier"), padStr(in, "GroupDisplayName"), rights))
		case n == 3 && seg[2] == "accessControlEntries" && m == http.MethodGet:
			token, max, err := padPaging(q)
			if err != nil {
				return 0, nil, err
			}
			out, err := st.listACEs(seg[1], token, max)
			return done(http.StatusOK, out, err)
		case n == 4 && seg[2] == "accessControlEntries":
			if err := padCheck("GroupSecurityIdentifier", "GroupSecurityIdentifier", seg[3]); err != nil {
				return 0, nil, err
			}
			switch m {
			case http.MethodGet:
				out, err := st.getACE(seg[1], seg[3])
				return done(http.StatusOK, out, err)
			case http.MethodPatch:
				in, err := padDecode(r, "UpdateTemplateGroupAccessControlEntry")
				if err != nil {
					return 0, nil, err
				}
				rights, _ := in["AccessRights"].(map[string]any)
				var display *string
				if v, ok := in["GroupDisplayName"].(string); ok {
					display = &v
				}
				return done(http.StatusOK, nil, st.updateACE(seg[1], seg[3], display, rights))
			case http.MethodDelete:
				return done(http.StatusOK, nil, st.deleteACE(seg[1], seg[3]))
			}
		}
	case "tags":
		if n != 2 {
			break
		}
		arn := seg[1]
		switch m {
		case http.MethodGet:
			out, err := st.listTags(arn)
			return done(http.StatusOK, out, err)
		case http.MethodPost:
			in, err := padDecode(r, "TagResource")
			if err != nil {
				return 0, nil, err
			}
			tags, _ := in["Tags"].(map[string]any)
			return done(http.StatusNoContent, nil, st.tag(arn, tags))
		case http.MethodDelete:
			keys := q["tagKeys"]
			if len(keys) == 0 || len(keys) > padMaxTags {
				return 0, nil, padValidation("FIELD_VALIDATION_FAILED", "tagKeys must name 1-%d keys.", padMaxTags)
			}
			return done(http.StatusNoContent, nil, st.untag(arn, keys))
		}
	}
	return 0, nil, &padError{status: http.StatusNotFound, code: "UnknownOperationException", msg: "no pca-connector-ad operation matches " + r.Method + " " + rawRequestPath(r)}
}

// padCheckConnectorTargets checks a new connector's CA, directory and security
// groups against their stores. It runs before the connector store's lock, so
// stores never lock in turn.
func (s *Server) padCheckConnectorTargets(caARN, dirID string, vpc map[string]any) *padError {
	ca, err := s.privateca.DescribeCertificateAuthority(caARN)
	if err != nil {
		return padNotFound("CERTIFICATE_AUTHORITY", caARN)
	}
	if ca.Status != "ACTIVE" {
		return padValidation("INVALID_STATE", "Certificate authority %s is %s, not ACTIVE.", caARN, ca.Status)
	}
	if _, ok := s.directoryservice.directoryStage(dirID); !ok {
		return padNotFound("DIRECTORY", dirID)
	}
	raw, _ := vpc["SecurityGroupIds"].([]any)
	ids := make([]string, 0, len(raw))
	for _, v := range raw {
		id, _ := v.(string)
		ids = append(ids, id)
	}
	groups := s.ec2.DescribeSecurityGroups(ids, nil, "")
	if len(groups) != len(ids) {
		return padValidation("FIELD_VALIDATION_FAILED", "One or more security groups in VpcInformation.SecurityGroupIds do not exist.")
	}
	for _, g := range groups {
		if g.VpcID != groups[0].VpcID {
			return padValidation("MISMATCHED_VPC", "All security groups must belong to the same VPC.")
		}
	}
	return nil
}

// directoryStage reports a directory's Stage if it exists.
func (s *directoryServiceStore) directoryStage(id string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	d, ok := s.directories[id]
	if !ok {
		return "", false
	}
	stage, _ := d["Stage"].(string)
	return stage, true
}

// padDecode reads a JSON body and validates it against the operation's input
// shape (URI and query members are skipped by the validator).
func padDecode(r *http.Request, op string) (map[string]any, *padError) {
	raw, err := readBodyBytes(r)
	if err != nil {
		return nil, padValidation("FIELD_VALIDATION_FAILED", "unable to read request body")
	}
	in := map[string]any{}
	if len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, &in); err != nil {
			return nil, padValidation("FIELD_VALIDATION_FAILED", "request body is not a JSON object")
		}
	}
	model := awsmodels.PCAConnectorADModel()
	if problems := model.Validate(model.Operations[op].Input, in); len(problems) > 0 {
		return nil, padValidation("FIELD_VALIDATION_FAILED", "%s", strings.Join(problems, "; "))
	}
	return in, nil
}

// padCheck validates a URI or query value against a model shape.
func padCheck(field, shape, v string) *padError {
	if problems := awsmodels.PCAConnectorADModel().Validate(shape, v); len(problems) > 0 || v == "" {
		return padValidation("FIELD_VALIDATION_FAILED", "%s is not valid.", field)
	}
	return nil
}

func padPaging(q url.Values) (string, int, *padError) {
	max := 100
	if raw := q.Get("MaxResults"); raw != "" {
		n, err := strconv.ParseInt(raw, 10, 32)
		if err != nil || n < 1 || n > 1000 {
			return "", 0, padValidation("FIELD_VALIDATION_FAILED", "MaxResults must be between 1 and 1000.")
		}
		max = int(n)
	}
	return q.Get("NextToken"), max, nil
}

func padStr(in map[string]any, key string) string {
	v, _ := in[key].(string)
	return v
}

func respondPADJSON(w http.ResponseWriter, status int, body map[string]any) {
	w.Header().Set("Content-Type", "application/json")
	if body == nil {
		w.WriteHeader(status)
		return
	}
	raw, _ := json.Marshal(body)
	w.WriteHeader(status)
	_, _ = w.Write(raw)
}

func respondPADError(w http.ResponseWriter, e *padError) {
	body := map[string]any{"Message": e.msg}
	if e.reason != "" {
		body["Reason"] = e.reason
	}
	if e.resourceType != "" {
		body["ResourceId"], body["ResourceType"] = e.resourceID, e.resourceType
	}
	if e.code == "ServiceQuotaExceededException" {
		body["ServiceCode"], body["QuotaCode"] = "pca-connector-ad", "L-"+e.resourceType
	}
	raw, _ := json.Marshal(body)
	w.Header().Set("X-Amzn-ErrorType", e.code)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(e.status)
	_, _ = w.Write(raw)
}
