package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

// AWS Elemental MediaPackage VOD: REST JSON with camelCase members, signed
// mediapackage-vod. The slim model is keyed by wire names
// (gen-aws-model-slim.py --wire-names), so it validates bodies as sent.

const mpvodMaxBody = 256 << 10

var (
	mpvodPathRoots = map[string]bool{"packaging_groups": true, "packaging_configurations": true, "assets": true, "tags": true}
	mpvodPackages  = []string{"cmafPackage", "dashPackage", "hlsPackage", "mssPackage"}
	mpvodS3ARN     = regexp.MustCompile(`^arn:aws:s3:::([a-z0-9][a-z0-9.-]{1,61}[a-z0-9])/(.+\.(?:smil|m3u8))$`)
	mpvodRoleARN   = regexp.MustCompile(`^arn:aws:iam::\d{12}:role/[\w+=,.@/-]{1,512}$`)
	mpvodSecretARN = regexp.MustCompile(`^arn:aws:secretsmanager:[a-z0-9-]+:\d{12}:secret:[\w/+=.@-]{1,512}$`)
)

func isMediaPackageVODRESTCandidate(r *http.Request) bool {
	host := strings.ToLower(strings.TrimSpace(r.Host))
	if strings.TrimSpace(sigV4ServiceHint(r)) != "mediapackage-vod" && !strings.HasPrefix(host, "mediapackage-vod.") {
		return false
	}
	seg := splitPathSegments(rawRequestPath(r))
	return len(seg) > 0 && mpvodPathRoots[seg[0]]
}

func (s *Server) handleMediaPackageVODRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isMediaPackageVODRESTCandidate(r) {
		return false
	}
	// A declared length bounds what net/http reads, so require one.
	if r.ContentLength < 0 || r.ContentLength > mpvodMaxBody {
		respondMPVODError(w, mpvodInvalid("request body must declare a Content-Length of at most 256 KiB"))
		return true
	}
	if ok, status, code, msg, _ := s.validateSigV4WithService(r, "mediapackage-vod"); !ok {
		respondMPVODError(w, &mpvodErr{status, code, msg})
		return true
	}
	seg := splitPathSegments(rawRequestPath(r))
	for i, v := range seg {
		u, err := url.PathUnescape(v)
		if err != nil {
			respondMPVODError(w, mpvodInvalid("invalid path segment"))
			return true
		}
		seg[i] = u
	}
	status, out, err := s.mpvodRoute(r, seg)
	if err != nil {
		respondMPVODError(w, err)
		return true
	}
	w.Header().Set("Content-Type", "application/json")
	if status == http.StatusNoContent {
		w.WriteHeader(status)
		return true
	}
	raw, _ := json.Marshal(out)
	w.WriteHeader(status)
	_, _ = w.Write(raw)
	return true
}

func (s *Server) mpvodRoute(r *http.Request, seg []string) (int, map[string]any, *mpvodErr) {
	m, n, q := r.Method, len(seg), r.URL.Query()
	st := s.mediaPackageVOD
	ok := func(out map[string]any, err *mpvodErr) (int, map[string]any, *mpvodErr) {
		return http.StatusOK, out, err
	}
	accepted := func(err *mpvodErr) (int, map[string]any, *mpvodErr) {
		return http.StatusAccepted, map[string]any{}, err
	}
	list := func(op string) (int, map[string]any, *mpvodErr) {
		max := 50
		if v := q.Get("maxResults"); v != "" {
			p, err := strconv.ParseInt(v, 10, 32)
			if err != nil || p < 1 || p > 1000 {
				return 0, nil, mpvodInvalid("maxResults must be between 1 and 1000")
			}
			max = int(p)
		}
		return ok(st.list(op, q.Get("packagingGroupId"), q.Get("nextToken"), max))
	}
	id := ""
	if n >= 2 {
		id = seg[1]
	}
	switch {
	case seg[0] == "packaging_groups" && n == 1 && m == http.MethodPost:
		in, err := mpvodDecode(r, "CreatePackagingGroup")
		if err != nil {
			return 0, nil, err
		}
		auth, logs, err := mpvodGroupSettings(in)
		if err != nil {
			return 0, nil, err
		}
		return ok(st.createGroup(mpvodStr(in, "id"), auth, logs, mpvodTags(in)))
	case seg[0] == "packaging_groups" && n == 1 && m == http.MethodGet:
		return list("ListPackagingGroups")
	case seg[0] == "packaging_groups" && n == 2 && m == http.MethodGet:
		return ok(st.describeGroup(id))
	case seg[0] == "packaging_groups" && n == 2 && m == http.MethodPut:
		in, err := mpvodDecode(r, "UpdatePackagingGroup")
		if err != nil {
			return 0, nil, err
		}
		auth, _, err := mpvodGroupSettings(in)
		if err != nil {
			return 0, nil, err
		}
		return ok(st.updateGroup("UpdatePackagingGroup", id, auth, nil))
	case seg[0] == "packaging_groups" && n == 3 && seg[2] == "configure_logs" && m == http.MethodPut:
		in, err := mpvodDecode(r, "ConfigureLogs")
		if err != nil {
			return 0, nil, err
		}
		_, logs, err := mpvodGroupSettings(in)
		if err != nil {
			return 0, nil, err
		}
		return ok(st.updateGroup("ConfigureLogs", id, nil, logs))
	case seg[0] == "packaging_groups" && n == 2 && m == http.MethodDelete:
		return accepted(st.deleteGroup(id))

	case seg[0] == "packaging_configurations" && n == 1 && m == http.MethodPost:
		in, err := mpvodDecode(r, "CreatePackagingConfiguration")
		if err != nil {
			return 0, nil, err
		}
		kind, pkg, err := mpvodPackage(in)
		if err != nil {
			return 0, nil, err
		}
		return ok(st.createConfig(mpvodStr(in, "id"), mpvodStr(in, "packagingGroupId"), kind, pkg, mpvodTags(in)))
	case seg[0] == "packaging_configurations" && n == 1 && m == http.MethodGet:
		return list("ListPackagingConfigurations")
	case seg[0] == "packaging_configurations" && n == 2 && m == http.MethodGet:
		return ok(st.describeConfig(id))
	case seg[0] == "packaging_configurations" && n == 2 && m == http.MethodDelete:
		return accepted(st.deleteConfig(id))

	case seg[0] == "assets" && n == 1 && m == http.MethodPost:
		in, err := mpvodDecode(r, "CreateAsset")
		if err != nil {
			return 0, nil, err
		}
		if err := s.mpvodCheckSource(r, mpvodStr(in, "sourceArn"), mpvodStr(in, "sourceRoleArn")); err != nil {
			return 0, nil, err
		}
		return ok(st.createAsset(mpvodAssetInput{
			id: mpvodStr(in, "id"), group: mpvodStr(in, "packagingGroupId"), resourceID: mpvodStr(in, "resourceId"),
			sourceARN: mpvodStr(in, "sourceArn"), roleARN: mpvodStr(in, "sourceRoleArn"), tags: mpvodTags(in),
		}))
	case seg[0] == "assets" && n == 1 && m == http.MethodGet:
		return list("ListAssets")
	case seg[0] == "assets" && n == 2 && m == http.MethodGet:
		return ok(st.describeAsset(id))
	case seg[0] == "assets" && n == 2 && m == http.MethodDelete:
		return accepted(st.deleteAsset(id))

	case seg[0] == "tags" && n >= 2:
		// ARNs may arrive percent-encoded (one segment) or with raw slashes.
		arn := strings.Join(seg[1:], "/")
		switch m {
		case http.MethodGet:
			return ok(st.listTags(arn))
		case http.MethodPost:
			in, err := mpvodDecode(r, "TagResource")
			if err != nil {
				return 0, nil, err
			}
			return http.StatusNoContent, nil, st.retag(arn, mpvodTags(in), nil)
		case http.MethodDelete:
			keys := q["tagKeys"]
			if len(keys) == 0 || len(keys) > mpvodMaxTags {
				return 0, nil, mpvodInvalid("tagKeys must list 1 to %d keys", mpvodMaxTags)
			}
			return http.StatusNoContent, nil, st.retag(arn, nil, keys)
		}
	}
	return 0, nil, &mpvodErr{http.StatusNotFound, "NotFoundException", "no MediaPackage VOD operation matches " + m + " " + rawRequestPath(r)}
}

// mpvodDecode reads a JSON body, validates it against the operation's input
// shape and conforms it (unknown members dropped).
func mpvodDecode(r *http.Request, op string) (map[string]any, *mpvodErr) {
	raw, err := readBodyBytes(r)
	if err != nil {
		return nil, mpvodInvalid("unable to read request body")
	}
	in := map[string]any{}
	if len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, &in); err != nil {
			return nil, mpvodInvalid("request body is not a JSON object")
		}
	}
	md := awsmodels.MediaPackageVODModel()
	shape := md.Operations[op].Input
	if problems := md.Validate(shape, in); len(problems) > 0 {
		return nil, mpvodInvalid("%s", strings.Join(problems, "; "))
	}
	conformed, _ := md.Conform(shape, in).(map[string]any)
	return conformed, nil
}

// mpvodGroupSettings checks a group's authorization and egress access logs.
func mpvodGroupSettings(in map[string]any) (map[string]any, map[string]any, *mpvodErr) {
	auth, _ := in["authorization"].(map[string]any)
	if auth != nil {
		if !mpvodSecretARN.MatchString(mpvodStr(auth, "cdnIdentifierSecret")) {
			return nil, nil, mpvodInvalid("authorization.cdnIdentifierSecret must be a Secrets Manager secret ARN")
		}
		if !mpvodRoleARN.MatchString(mpvodStr(auth, "secretsRoleArn")) {
			return nil, nil, mpvodInvalid("authorization.secretsRoleArn must be an IAM role ARN")
		}
	}
	logs, _ := in["egressAccessLogs"].(map[string]any)
	if logs != nil {
		name := mpvodStr(logs, "logGroupName")
		if name == "" {
			name = mpvodDefaultLogs
		}
		if !strings.HasPrefix(name, "/aws/MediaPackage/") || len(name) > 512 {
			return nil, nil, mpvodInvalid("egressAccessLogs.logGroupName must start with /aws/MediaPackage/")
		}
		logs = map[string]any{"logGroupName": name}
	}
	return auth, logs, nil
}

// mpvodPackage requires exactly one package type with 1 to 20 manifests of
// distinct names, and positive segment durations.
func mpvodPackage(in map[string]any) (string, map[string]any, *mpvodErr) {
	kind := ""
	for _, k := range mpvodPackages {
		if in[k] != nil {
			if kind != "" {
				return "", nil, mpvodInvalid("Specify exactly one of cmafPackage, dashPackage, hlsPackage or mssPackage")
			}
			kind = k
		}
	}
	if kind == "" {
		return "", nil, mpvodInvalid("Specify exactly one of cmafPackage, dashPackage, hlsPackage or mssPackage")
	}
	pkg, _ := in[kind].(map[string]any)
	if d, ok := pkg["segmentDurationSeconds"].(float64); ok && (d < 1 || d > 30) {
		return "", nil, mpvodInvalid("%s.segmentDurationSeconds must be between 1 and 30", kind)
	}
	key := map[string]string{"cmafPackage": "hlsManifests", "hlsPackage": "hlsManifests", "dashPackage": "dashManifests", "mssPackage": "mssManifests"}[kind]
	manifests, _ := pkg[key].([]any)
	if len(manifests) == 0 || len(manifests) > mpvodMaxManifests {
		return "", nil, mpvodInvalid("%s.%s needs 1 to %d manifests", kind, key, mpvodMaxManifests)
	}
	seen := map[string]bool{}
	for _, item := range manifests {
		mf, _ := item.(map[string]any)
		name := mpvodStr(mf, "manifestName")
		if seen[name] {
			return "", nil, mpvodInvalid("%s.%s has two manifests named %q", kind, key, name)
		}
		seen[name] = true
	}
	return kind, pkg, nil
}

// mpvodCheckSource requires an IAM role ARN and an existing .smil or .m3u8
// object in Stackyard's S3. Runs before the VOD store lock.
func (s *Server) mpvodCheckSource(r *http.Request, source, role string) *mpvodErr {
	if !mpvodRoleARN.MatchString(role) {
		return mpvodInvalid("sourceRoleArn must be an IAM role ARN")
	}
	m := mpvodS3ARN.FindStringSubmatch(source)
	if m == nil {
		return mpvodInvalid("sourceArn must be an S3 object ARN ending in .smil or .m3u8")
	}
	if !s.authorizeServiceS3(r, "mediapackage.amazonaws.com", m[1], m[2], "s3:GetObject") {
		return &mpvodErr{http.StatusForbidden, "ForbiddenException", "Access denied reading " + source}
	}
	if _, err := s.s3.GetObject(m[1], m[2]); err != nil {
		return mpvodInvalid("sourceArn %s does not exist", source)
	}
	return nil
}

func mpvodStr(in map[string]any, key string) string {
	v, _ := in[key].(string)
	return v
}

func mpvodTags(in map[string]any) map[string]string {
	raw, _ := in["tags"].(map[string]any)
	if len(raw) == 0 {
		return nil
	}
	out := make(map[string]string, len(raw))
	for k, v := range raw {
		out[k], _ = v.(string)
	}
	return out
}

func respondMPVODError(w http.ResponseWriter, e *mpvodErr) {
	raw, _ := json.Marshal(map[string]string{"message": e.msg})
	w.Header().Set("X-Amzn-ErrorType", e.code)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(e.status)
	_, _ = w.Write(raw)
}
