package server

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	ekssvc "github.com/stackyard/stackyard/internal/services/eks"
)

// EKS Auth (signing name eks-auth) has one operation,
// AssumeRoleForPodIdentity, answered from the EKS store's pod identity
// associations and the STS store's temporary credentials.
const (
	ekaAudience      = "pods.eks.amazonaws.com"
	ekaMaxBody       = 32 << 10
	ekaMaxTokenBytes = 16 << 10
	ekaCredentialTTL = 6 * time.Hour
	ekaSessionMaxLen = 64
	ekaDefaultAcct   = "123456789012"
)

var (
	ekaClusterNamePattern = regexp.MustCompile(`^[0-9A-Za-z][A-Za-z0-9\-_]*$`)
	ekaTokenPattern       = regexp.MustCompile(`^[A-Za-z0-9\-_=]+\.[A-Za-z0-9\-_=]+\.[A-Za-z0-9\-_=]+$`)
	ekaSessionUnsafe      = regexp.MustCompile(`[^\w+=,.@-]`)
)

type ekaError struct {
	status int
	code   string
	msg    string
}

func ekaErr(status int, code, format string, args ...any) *ekaError {
	return &ekaError{status: status, code: code, msg: fmt.Sprintf(format, args...)}
}

type ekaRequest struct {
	Token       string `json:"token"`
	EksNodeName string `json:"eksNodeName"`
	InstanceID  string `json:"instanceId"`
	Zone        string `json:"zone"`
}

// ekaClaims are the projected service-account token claims EKS Auth reads.
type ekaClaims struct {
	Aud        json.RawMessage `json:"aud"`
	Exp        *float64        `json:"exp"`
	Sub        string          `json:"sub"`
	Kubernetes struct {
		Pod struct {
			Name string `json:"name"`
		} `json:"pod"`
	} `json:"kubernetes.io"`
}

func (s *Server) handleEKSAuthRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isEKSAuthCandidate(r) {
		return false
	}
	cluster, ok := ekaRoute(r)
	if !ok {
		respondEKAError(w, ekaErr(http.StatusNotFound, "UnknownOperationException", "no EKS Auth operation matches %s %s", r.Method, rawRequestPath(r)))
		return true
	}
	// A declared length bounds the body (net/http stops reading there), so
	// require one; chunked bodies of unknown length are refused. r.Body is
	// not reassigned (CodeQL treats that as a fresh request source).
	if r.ContentLength < 0 || r.ContentLength > ekaMaxBody {
		respondEKAError(w, ekaErr(http.StatusBadRequest, "InvalidRequestException", "request body must declare a Content-Length of at most %d bytes", ekaMaxBody))
		return true
	}
	if ok, status, code, msg, _ := s.validateSigV4WithService(r, "eks-auth"); !ok {
		respondEKAError(w, &ekaError{status: status, code: code, msg: msg})
		return true
	}
	body, err := readBodyBytes(r)
	if err != nil {
		respondEKAError(w, ekaErr(http.StatusBadRequest, "InvalidRequestException", "unable to read request body"))
		return true
	}
	var in ekaRequest
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&in); err != nil {
		respondEKAError(w, ekaErr(http.StatusBadRequest, "InvalidRequestException", "invalid JSON body"))
		return true
	}
	out, apiErr := s.ekaAssumeRole(cluster, in, time.Now().UTC())
	if apiErr != nil {
		respondEKAError(w, apiErr)
		return true
	}
	respondEKAJSON(w, http.StatusOK, out)
	return true
}

// isEKSAuthCandidate claims requests signed for eks-auth, or unsigned ones
// sent to an eks-auth host. The EKS router (signing name eks) never sees them.
func isEKSAuthCandidate(r *http.Request) bool {
	hint := strings.TrimSpace(sigV4ServiceHint(r))
	if hint != "" {
		return hint == "eks-auth"
	}
	host := strings.ToLower(r.Host)
	return strings.HasPrefix(host, "eks-auth.") || strings.Contains(host, ".eks-auth.")
}

// ekaRoute matches POST /clusters/{clusterName}/assume-role-for-pod-identity.
func ekaRoute(r *http.Request) (string, bool) {
	if r.Method != http.MethodPost {
		return "", false
	}
	parts := strings.Split(strings.Trim(rawRequestPath(r), "/"), "/")
	if len(parts) != 3 || parts[0] != "clusters" || parts[2] != "assume-role-for-pod-identity" || parts[1] == "" {
		return "", false
	}
	return parts[1], true
}

func (s *Server) ekaAssumeRole(cluster string, in ekaRequest, now time.Time) (map[string]any, *ekaError) {
	if len(cluster) > 100 || !ekaClusterNamePattern.MatchString(cluster) {
		return nil, ekaErr(http.StatusBadRequest, "InvalidParameterException", "clusterName %q is not valid", cluster)
	}
	if in.Token == "" {
		return nil, ekaErr(http.StatusBadRequest, "InvalidParameterException", "token is required")
	}
	namespace, serviceAccount, pod, apiErr := ekaParseToken(in.Token, now)
	if apiErr != nil {
		return nil, apiErr
	}
	matches, err := s.eks.ListPodIdentityAssociations(cluster, namespace, serviceAccount)
	if errors.Is(err, ekssvc.ErrClusterNotFound) {
		return nil, ekaErr(http.StatusNotFound, "ResourceNotFoundException", "No cluster found for name: %s.", cluster)
	}
	if err != nil {
		return nil, ekaErr(http.StatusInternalServerError, "InternalServerException", "unable to read pod identity associations")
	}
	// ponytail: AWS's error for a token with no matching association isn't
	// documented beyond the modeled faults; AccessDenied is the closest fit.
	if len(matches) == 0 {
		return nil, ekaErr(http.StatusBadRequest, "AccessDeniedException", "No Pod Identity Association found for service account %s/%s in cluster %s", namespace, serviceAccount, cluster)
	}
	assoc, err := s.eks.DescribePodIdentityAssociation(cluster, matches[0].AssociationID)
	if err != nil {
		return nil, ekaErr(http.StatusNotFound, "ResourceNotFoundException", "pod identity association %s was not found", matches[0].AssociationID)
	}

	session := ekaSessionName(cluster, pod, serviceAccount)
	account, roleName := ekaRoleParts(assoc.RoleArn)
	creds := s.sts.IssueCredentials(session)
	return map[string]any{
		"subject":  map[string]any{"namespace": namespace, "serviceAccount": serviceAccount},
		"audience": ekaAudience,
		"podIdentityAssociation": map[string]any{
			"associationArn": assoc.AssociationArn,
			"associationId":  assoc.AssociationID,
		},
		"assumedRoleUser": map[string]any{
			"arn":          fmt.Sprintf("arn:aws:sts::%s:assumed-role/%s/%s", account, roleName, session),
			"assumeRoleId": "AROASTACKYARDPODIDENTITY:" + session,
		},
		// ponytail: Stackyard's SigV4 check accepts only its configured key,
		// so these temporary credentials are shaped like STS ones but aren't
		// accepted for signing; callers keep using the configured key.
		"credentials": map[string]any{
			"accessKeyId":     creds["AccessKeyId"],
			"secretAccessKey": creds["SecretAccessKey"],
			"sessionToken":    creds["SessionToken"],
			"expiration":      float64(now.Add(ekaCredentialTTL).Unix()),
		},
	}, nil
}

// ekaParseToken reads a projected service-account JWT. The signature can't be
// checked (Stackyard holds no cluster signing keys), so it validates shape and
// claims: aud must include pods.eks.amazonaws.com, exp must be in the future,
// and sub must be system:serviceaccount:<namespace>:<name>.
func ekaParseToken(token string, now time.Time) (namespace, serviceAccount, pod string, apiErr *ekaError) {
	invalid := func(msg string) (string, string, string, *ekaError) {
		return "", "", "", ekaErr(http.StatusBadRequest, "InvalidTokenException", "%s", msg)
	}
	if len(token) > ekaMaxTokenBytes || !ekaTokenPattern.MatchString(token) {
		return invalid("The token is not a valid JWT")
	}
	payload, err := base64.RawURLEncoding.DecodeString(strings.TrimRight(strings.Split(token, ".")[1], "="))
	if err != nil {
		return invalid("The token payload is not valid base64url")
	}
	var claims ekaClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return invalid("The token payload is not a JSON object")
	}
	if !ekaHasAudience(claims.Aud) {
		return invalid("The token audience must include " + ekaAudience)
	}
	if claims.Exp == nil {
		return invalid("The token has no expiry")
	}
	if float64(now.Unix()) >= *claims.Exp {
		return "", "", "", ekaErr(http.StatusBadRequest, "ExpiredTokenException", "The token has expired")
	}
	rest, ok := strings.CutPrefix(claims.Sub, "system:serviceaccount:")
	ns, sa, found := strings.Cut(rest, ":")
	if !ok || !found || ns == "" || sa == "" || strings.Contains(sa, ":") {
		return invalid("The token subject must be system:serviceaccount:<namespace>:<name>")
	}
	return ns, sa, claims.Kubernetes.Pod.Name, nil
}

func ekaHasAudience(raw json.RawMessage) bool {
	var one string
	if json.Unmarshal(raw, &one) == nil {
		return one == ekaAudience
	}
	var many []string
	if json.Unmarshal(raw, &many) != nil {
		return false
	}
	for _, a := range many {
		if a == ekaAudience {
			return true
		}
	}
	return false
}

// ekaSessionName mirrors EKS's eks-<cluster>-<pod> session names, kept to the
// IAM session-name alphabet and 64 characters.
func ekaSessionName(cluster, pod, serviceAccount string) string {
	who := pod
	if who == "" {
		who = serviceAccount
	}
	name := ekaSessionUnsafe.ReplaceAllString("eks-"+cluster+"-"+who, "-")
	if len(name) > ekaSessionMaxLen {
		name = name[:ekaSessionMaxLen]
	}
	return name
}

// ekaRoleParts splits arn:aws:iam::<account>:role/<path/name> into the account
// and the bare role name.
func ekaRoleParts(roleArn string) (account, name string) {
	account, name = ekaDefaultAcct, "stackyard-role"
	parts := strings.SplitN(roleArn, ":", 6)
	if len(parts) == 6 {
		if parts[4] != "" {
			account = parts[4]
		}
		if i := strings.LastIndex(parts[5], "/"); i >= 0 && i < len(parts[5])-1 {
			name = parts[5][i+1:]
		}
	}
	return account, name
}

// respondEKAJSON writes a JSON body; the content type is fixed here, where the
// body is written.
func respondEKAJSON(w http.ResponseWriter, status int, v any) {
	body, err := json.Marshal(v)
	if err != nil {
		status, body = http.StatusInternalServerError, []byte(`{"message":"internal error"}`)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

func respondEKAError(w http.ResponseWriter, e *ekaError) {
	w.Header().Set("X-Amzn-ErrorType", e.code)
	respondEKAJSON(w, e.status, map[string]string{"__type": e.code, "message": e.msg})
}
