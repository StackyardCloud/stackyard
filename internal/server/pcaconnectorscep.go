package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/stackyard/stackyard/internal/awsmodels"
	"github.com/stackyard/stackyard/internal/services/privateca"
)

// pscMaxBody bounds a request body; SCEP connector requests are small.
const pscMaxBody = 64 << 10

var pscPathRoots = map[string]bool{
	"connectors": true, "challenges": true, "challengeMetadata": true, "challengePasswords": true, "tags": true,
}

func isPCAConnectorSCEPRESTCandidate(r *http.Request) bool {
	service := strings.TrimSpace(sigV4ServiceHint(r))
	host := strings.ToLower(strings.TrimSpace(r.Host))
	if service != "pca-connector-scep" && !strings.HasPrefix(host, "pca-connector-scep.") {
		return false
	}
	seg := splitPathSegments(rawRequestPath(r))
	return len(seg) > 0 && pscPathRoots[seg[0]]
}

func (s *Server) handlePCAConnectorSCEPRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isPCAConnectorSCEPRESTCandidate(r) {
		return false
	}
	// A declared length bounds what net/http reads, so require one.
	if r.ContentLength < 0 || r.ContentLength > pscMaxBody {
		respondPSCError(w, pscValidation("OTHER", "request body must declare a Content-Length of at most 64 KiB"))
		return true
	}
	if ok, status, code, msg, _ := s.validateSigV4WithService(r, "pca-connector-scep"); !ok {
		respondPSCError(w, &pscError{status: status, code: code, msg: msg})
		return true
	}
	seg := splitPathSegments(rawRequestPath(r))
	for i, v := range seg {
		u, err := url.PathUnescape(v)
		if err != nil {
			respondPSCError(w, pscValidation("OTHER", "invalid path segment"))
			return true
		}
		seg[i] = u
	}
	status, out, err := s.pscRoute(r, seg)
	if err != nil {
		respondPSCError(w, err)
		return true
	}
	respondPSCJSON(w, status, out)
	return true
}

func (s *Server) pscRoute(r *http.Request, seg []string) (int, map[string]any, *pscError) {
	q, m, n := r.URL.Query(), r.Method, len(seg)
	// ARN labels may arrive percent-encoded (one segment) or with raw slashes.
	label := strings.Join(seg[min(1, n):], "/")
	ok := func(out map[string]any, err *pscError) (int, map[string]any, *pscError) {
		return http.StatusOK, out, err
	}
	accepted := func(out map[string]any, err *pscError) (int, map[string]any, *pscError) {
		return http.StatusAccepted, out, err
	}
	switch {
	case seg[0] == "connectors" && n == 1 && m == http.MethodPost:
		in, err := pscDecode(r, "CreateConnector", "CertificateAuthorityArn")
		if err != nil {
			return 0, nil, err
		}
		caARN := pscStr(in, "CertificateAuthorityArn")
		if err := s.pscCheckCA(caARN); err != nil {
			return 0, nil, err
		}
		intune, _ := in["MobileDeviceManagement"].(map[string]any)
		intune, _ = intune["Intune"].(map[string]any)
		scheme := "http"
		if r.TLS != nil {
			scheme = "https"
		}
		return accepted(s.pcaConnectorSCEP.createConnector(pscConnectorInput{
			caARN: caARN, azureAppID: pscStr(intune, "AzureApplicationId"), domain: pscStr(intune, "Domain"),
			vpce: pscStr(in, "VpcEndpointId"), token: pscStr(in, "ClientToken"), tags: pscTags(in),
			endpointBase: scheme + "://" + r.Host,
		}))
	case seg[0] == "connectors" && n == 1 && m == http.MethodGet:
		token, max, err := pscPaging(q)
		if err != nil {
			return 0, nil, err
		}
		return ok(s.pcaConnectorSCEP.listConnectors(token, max))
	case seg[0] == "connectors" && n >= 2 && (m == http.MethodGet || m == http.MethodDelete):
		if err := pscCheckShape("ConnectorArn", label); err != nil {
			return 0, nil, err
		}
		if m == http.MethodGet {
			return ok(s.pcaConnectorSCEP.getConnector(label))
		}
		return http.StatusAccepted, nil, s.pcaConnectorSCEP.deleteConnector(label)

	case seg[0] == "challenges" && n == 1 && m == http.MethodPost:
		in, err := pscDecode(r, "CreateChallenge")
		if err != nil {
			return 0, nil, err
		}
		return accepted(s.pcaConnectorSCEP.createChallenge(pscStr(in, "ConnectorArn"), pscStr(in, "ClientToken"), pscTags(in)))
	case seg[0] == "challenges" && n >= 2 && m == http.MethodDelete:
		if err := pscCheckShape("ChallengeArn", label); err != nil {
			return 0, nil, err
		}
		return http.StatusAccepted, nil, s.pcaConnectorSCEP.deleteChallenge(label)
	case seg[0] == "challengeMetadata" && n == 1 && m == http.MethodGet:
		arn := q.Get("ConnectorArn")
		if arn == "" {
			return 0, nil, pscValidation("OTHER", "ConnectorArn is required.")
		}
		if err := pscCheckShape("ConnectorArn", arn); err != nil {
			return 0, nil, err
		}
		token, max, err := pscPaging(q)
		if err != nil {
			return 0, nil, err
		}
		return ok(s.pcaConnectorSCEP.listChallengeMetadata(arn, token, max))
	case seg[0] == "challengeMetadata" && n >= 2 && m == http.MethodGet:
		if err := pscCheckShape("ChallengeArn", label); err != nil {
			return 0, nil, err
		}
		return ok(s.pcaConnectorSCEP.getChallengeMetadata(label))
	case seg[0] == "challengePasswords" && n >= 2 && m == http.MethodGet:
		if err := pscCheckShape("ChallengeArn", label); err != nil {
			return 0, nil, err
		}
		return ok(s.pcaConnectorSCEP.getChallengePassword(label))

	case seg[0] == "tags" && n >= 2:
		switch m {
		case http.MethodGet:
			return ok(s.pcaConnectorSCEP.listTags(label))
		case http.MethodPost:
			in, err := pscDecode(r, "TagResource")
			if err != nil {
				return 0, nil, err
			}
			return http.StatusNoContent, nil, s.pcaConnectorSCEP.tag(label, pscTags(in))
		case http.MethodDelete:
			keys := q["tagKeys"]
			if len(keys) == 0 || len(keys) > pscMaxTags {
				return 0, nil, pscValidation("OTHER", "tagKeys must list 1 to %d keys.", pscMaxTags)
			}
			return http.StatusNoContent, nil, s.pcaConnectorSCEP.untag(label, keys)
		}
	}
	return 0, nil, pscValidation("UNKNOWN_OPERATION", "no Private CA Connector for SCEP operation matches %s %s", r.Method, rawRequestPath(r))
}

// pscCheckCA requires an ACTIVE, general-purpose CA in the Private CA store.
// The model's ARN pattern expects a UUID, but Stackyard's CA ids are hex, so
// the CA is checked by lookup instead. Runs before the connector lock.
func (s *Server) pscCheckCA(arn string) *pscError {
	if !strings.HasPrefix(arn, "arn:aws:acm-pca:") || len(arn) < 5 || len(arn) > 200 {
		return pscValidation("OTHER", "CertificateAuthorityArn %q isn't a Private CA ARN.", arn)
	}
	ca, err := s.privateca.DescribeCertificateAuthority(arn)
	if errors.Is(err, privateca.ErrNotFound) {
		return pscNotFound("CertificateAuthority", arn)
	}
	if err != nil {
		return pscValidation("OTHER", "CertificateAuthorityArn %q isn't valid.", arn)
	}
	if ca.Status != "ACTIVE" {
		return pscValidation("INVALID_STATE", "Certificate authority %s is %s; it must be ACTIVE.", arn, ca.Status)
	}
	if ca.UsageMode != "" && ca.UsageMode != "GENERAL_PURPOSE" {
		return pscValidation("INVALID_CA_USAGE_MODE", "Certificate authority %s has usage mode %s; SCEP needs GENERAL_PURPOSE.", arn, ca.UsageMode)
	}
	return nil
}

// pscDecode reads a JSON body and validates it against the operation's input
// shape. Members named in skip are checked by the caller instead.
func pscDecode(r *http.Request, op string, skip ...string) (map[string]any, *pscError) {
	raw, err := readBodyBytes(r)
	if err != nil {
		return nil, pscValidation("OTHER", "unable to read request body")
	}
	in := map[string]any{}
	if len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, &in); err != nil {
			return nil, pscValidation("OTHER", "request body is not a JSON object")
		}
	}
	check := make(map[string]any, len(in))
	for k, v := range in {
		check[k] = v
	}
	for _, k := range skip {
		if _, ok := in[k].(string); !ok {
			return nil, pscValidation("OTHER", "input: missing required member %s", k)
		}
		delete(check, k)
	}
	md := awsmodels.PCAConnectorSCEPModel()
	var problems []string
	for _, p := range md.Validate(md.Operations[op].Input, check) {
		if !pscSkipped(p, skip) {
			problems = append(problems, p)
		}
	}
	if len(problems) > 0 {
		return nil, pscValidation("OTHER", "%s", strings.Join(problems, "; "))
	}
	return in, nil
}

// pscSkipped reports a "missing required member" problem for a member the
// caller checks itself.
func pscSkipped(problem string, skip []string) bool {
	for _, k := range skip {
		if strings.HasSuffix(problem, "missing required member "+k) {
			return true
		}
	}
	return false
}

func pscCheckShape(shape, v string) *pscError {
	if problems := awsmodels.PCAConnectorSCEPModel().Validate(shape, v); len(problems) > 0 {
		_, reason, _ := strings.Cut(problems[0], ": ")
		return pscValidation("OTHER", "%s %s.", shape, reason)
	}
	return nil
}

func pscPaging(q url.Values) (string, int, *pscError) {
	max := 20
	if raw := q.Get("MaxResults"); raw != "" {
		n, err := strconv.ParseInt(raw, 10, 32)
		if err != nil || n < 1 || n > 1000 {
			return "", 0, pscValidation("OTHER", "MaxResults must be between 1 and 1000.")
		}
		max = int(n)
	}
	return q.Get("NextToken"), max, nil
}

func pscStr(in map[string]any, key string) string {
	v, _ := in[key].(string)
	return v
}

func pscTags(in map[string]any) map[string]string {
	raw, _ := in["Tags"].(map[string]any)
	if len(raw) == 0 {
		return nil
	}
	out := make(map[string]string, len(raw))
	for k, v := range raw {
		out[k], _ = v.(string)
	}
	return out
}

func respondPSCJSON(w http.ResponseWriter, status int, body map[string]any) {
	w.Header().Set("Content-Type", "application/json")
	if status == http.StatusNoContent || body == nil {
		w.WriteHeader(status)
		return
	}
	raw, _ := json.Marshal(body)
	w.WriteHeader(status)
	_, _ = w.Write(raw)
}

func respondPSCError(w http.ResponseWriter, e *pscError) {
	body := map[string]any{"Message": e.msg}
	if e.reason != "" {
		body["Reason"] = e.reason
	}
	if e.resourceID != "" {
		body["ResourceId"] = e.resourceID
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
