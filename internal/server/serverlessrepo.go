package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// sarMaxBody bounds a request body: a template body travels inline.
const sarMaxBody = 2 << 20

type sarCreateAppInput struct {
	Author               string   `json:"author"`
	Description          string   `json:"description"`
	HomePageURL          string   `json:"homePageUrl"`
	Labels               []string `json:"labels"`
	LicenseBody          string   `json:"licenseBody"`
	LicenseURL           string   `json:"licenseUrl"`
	Name                 string   `json:"name"`
	ReadmeBody           string   `json:"readmeBody"`
	ReadmeURL            string   `json:"readmeUrl"`
	SemanticVersion      string   `json:"semanticVersion"`
	SourceCodeArchiveURL string   `json:"sourceCodeArchiveUrl"`
	SourceCodeURL        string   `json:"sourceCodeUrl"`
	SpdxLicenseID        string   `json:"spdxLicenseId"`
	TemplateBody         string   `json:"templateBody"`
	TemplateURL          string   `json:"templateUrl"`
}

type sarVersionInput struct {
	SourceCodeArchiveURL string `json:"sourceCodeArchiveUrl"`
	SourceCodeURL        string `json:"sourceCodeUrl"`
	TemplateBody         string `json:"templateBody"`
	TemplateURL          string `json:"templateUrl"`
}

type sarUpdateAppInput struct {
	Author      *string  `json:"author"`
	Description *string  `json:"description"`
	HomePageURL *string  `json:"homePageUrl"`
	Labels      []string `json:"labels"`
	ReadmeBody  *string  `json:"readmeBody"`
	ReadmeURL   *string  `json:"readmeUrl"`
}

type sarStatementInput struct {
	Actions         []string `json:"actions"`
	PrincipalOrgIDs []string `json:"principalOrgIDs"`
	Principals      []string `json:"principals"`
	StatementID     string   `json:"statementId"`
}

type sarChangeSetInput struct {
	Capabilities       []string `json:"capabilities"`
	ChangeSetName      string   `json:"changeSetName"`
	ClientToken        string   `json:"clientToken"`
	Description        string   `json:"description"`
	NotificationArns   []string `json:"notificationArns"`
	ParameterOverrides []struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	} `json:"parameterOverrides"`
	ResourceTypes         []string `json:"resourceTypes"`
	RollbackConfiguration *struct {
		MonitoringTimeInMinutes *int64 `json:"monitoringTimeInMinutes"`
		RollbackTriggers        []struct {
			Arn  string `json:"arn"`
			Type string `json:"type"`
		} `json:"rollbackTriggers"`
	} `json:"rollbackConfiguration"`
	SemanticVersion string `json:"semanticVersion"`
	StackName       string `json:"stackName"`
	Tags            []struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	} `json:"tags"`
	TemplateID string `json:"templateId"`
}

func isServerlessRepoRESTCandidate(r *http.Request) bool {
	service := strings.TrimSpace(sigV4ServiceHint(r))
	host := strings.ToLower(strings.TrimSpace(r.Host))
	if service != "serverlessrepo" && !strings.HasPrefix(host, "serverlessrepo.") {
		return false
	}
	seg := splitPathSegments(rawRequestPath(r))
	return len(seg) > 0 && seg[0] == "applications"
}

func (s *Server) handleServerlessRepoRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isServerlessRepoRESTCandidate(r) {
		return false
	}
	// A declared length bounds what net/http reads, so require one.
	if r.ContentLength < 0 || r.ContentLength > sarMaxBody {
		respondSARError(w, sarBadRequest("The request body must declare a Content-Length of at most 2 MiB."))
		return true
	}
	if ok, status, code, msg, _ := s.validateSigV4WithService(r, "serverlessrepo"); !ok {
		respondSARError(w, &sarError{status, code, msg})
		return true
	}
	seg := splitPathSegments(rawRequestPath(r))
	for i, v := range seg {
		u, err := url.PathUnescape(v)
		if err != nil {
			respondSARError(w, sarBadRequest("Invalid path segment."))
			return true
		}
		seg[i] = u
	}
	status, out, err := s.sarRoute(r, seg)
	if err != nil {
		respondSARError(w, err)
		return true
	}
	respondSARJSON(w, status, out)
	return true
}

func (s *Server) sarRoute(r *http.Request, seg []string) (int, map[string]any, *sarError) {
	m, q, n := r.Method, r.URL.Query(), len(seg)
	if n == 1 {
		switch m {
		case http.MethodPost:
			return s.sarCreateApplication(r)
		case http.MethodGet:
			max, err := sarMaxItems(q)
			if err != nil {
				return 0, nil, err
			}
			out, err := s.serverlessRepo.list(q.Get("nextToken"), max)
			return http.StatusOK, out, err
		}
	}
	if n < 2 {
		return 0, nil, sarNotFound("No Serverless Application Repository operation matches %s %s.", m, rawRequestPath(r))
	}
	app := seg[1]
	switch {
	case n == 2 && m == http.MethodGet:
		out, err := s.serverlessRepo.get(app, q.Get("semanticVersion"))
		return http.StatusOK, out, err
	case n == 2 && m == http.MethodPatch:
		return s.sarUpdateApplication(r, app)
	case n == 2 && m == http.MethodDelete:
		return http.StatusNoContent, nil, s.serverlessRepo.delete(app)
	case n == 4 && seg[2] == "versions" && m == http.MethodPut:
		return s.sarCreateVersion(r, app, seg[3])
	case n == 3 && seg[2] == "versions" && m == http.MethodGet:
		max, err := sarMaxItems(q)
		if err != nil {
			return 0, nil, err
		}
		out, err := s.serverlessRepo.listVersions(app, q.Get("nextToken"), max)
		return http.StatusOK, out, err
	case n == 3 && seg[2] == "dependencies" && m == http.MethodGet:
		max, err := sarMaxItems(q)
		if err != nil {
			return 0, nil, err
		}
		out, err := s.serverlessRepo.listDependencies(app, q.Get("semanticVersion"), q.Get("nextToken"), max)
		return http.StatusOK, out, err
	case n == 3 && seg[2] == "policy" && m == http.MethodPut:
		return s.sarPutPolicy(r, app)
	case n == 3 && seg[2] == "policy" && m == http.MethodGet:
		out, err := s.serverlessRepo.getPolicy(app)
		return http.StatusOK, out, err
	case n == 3 && seg[2] == "unshare" && m == http.MethodPost:
		var in struct {
			OrganizationID string `json:"organizationId"`
		}
		if err := sarDecode(r, &in); err != nil {
			return 0, nil, err
		}
		if !sarOrgRe.MatchString(in.OrganizationID) {
			return 0, nil, sarBadRequest("organizationId must be an AWS Organizations ID such as o-a1b2c3d4e5.")
		}
		return http.StatusNoContent, nil, s.serverlessRepo.unshare(app, in.OrganizationID)
	case n == 3 && seg[2] == "templates" && m == http.MethodPost:
		return s.sarCreateTemplate(r, app)
	case n == 4 && seg[2] == "templates" && m == http.MethodGet:
		out, _, err := s.serverlessRepo.getTemplate(app, seg[3])
		return http.StatusOK, out, err
	case n == 3 && seg[2] == "changesets" && m == http.MethodPost:
		return s.sarCreateChangeSet(r, app)
	}
	return 0, nil, sarNotFound("No Serverless Application Repository operation matches %s %s.", m, rawRequestPath(r))
}

func (s *Server) sarCreateApplication(r *http.Request) (int, map[string]any, *sarError) {
	var in sarCreateAppInput
	if err := sarDecode(r, &in); err != nil {
		return 0, nil, err
	}
	switch {
	case !sarNameRe.MatchString(in.Name):
		return 0, nil, sarBadRequest("name must be 1-140 letters, digits or hyphens.")
	case !sarValidAuthor(in.Author):
		return 0, nil, sarBadRequest("author must be 1-127 lowercase letters, digits and single hyphens, not starting or ending with a hyphen.")
	case in.Description == "" || len(in.Description) > 256:
		return 0, nil, sarBadRequest("description must be 1-256 characters.")
	case in.LicenseBody != "" && in.LicenseURL != "":
		return 0, nil, sarBadRequest("Specify licenseBody or licenseUrl, not both.")
	case in.ReadmeBody != "" && in.ReadmeURL != "":
		return 0, nil, sarBadRequest("Specify readmeBody or readmeUrl, not both.")
	case in.TemplateBody != "" && in.TemplateURL != "":
		return 0, nil, sarBadRequest("Specify templateBody or templateUrl, not both.")
	case (in.TemplateBody != "" || in.TemplateURL != "") && in.SemanticVersion == "":
		return 0, nil, sarBadRequest("semanticVersion is required with a template.")
	case in.SemanticVersion != "" && in.TemplateBody == "" && in.TemplateURL == "":
		return 0, nil, sarBadRequest("A template (templateBody or templateUrl) is required with semanticVersion.")
	}
	if err := sarCheckLabels(in.Labels); err != nil {
		return 0, nil, err
	}
	meta := &sarAppMeta{in: in}
	var err *sarError
	if meta.licenseURL, err = s.sarDocument(r, "licenseUrl", in.LicenseURL, in.LicenseBody, in.Name, "LICENSE.txt"); err != nil {
		return 0, nil, err
	}
	if meta.readmeURL, err = s.sarDocument(r, "readmeUrl", in.ReadmeURL, in.ReadmeBody, in.Name, "README.md"); err != nil {
		return 0, nil, err
	}
	if in.SemanticVersion != "" {
		if meta.version, err = s.sarNewVersion(r, in.Name, in.SemanticVersion, sarVersionInput{
			SourceCodeArchiveURL: in.SourceCodeArchiveURL, SourceCodeURL: in.SourceCodeURL,
			TemplateBody: in.TemplateBody, TemplateURL: in.TemplateURL,
		}); err != nil {
			return 0, nil, err
		}
	}
	out, err := s.serverlessRepo.create(meta)
	return http.StatusCreated, out, err
}

func (s *Server) sarUpdateApplication(r *http.Request, app string) (int, map[string]any, *sarError) {
	var in sarUpdateAppInput
	if err := sarDecode(r, &in); err != nil {
		return 0, nil, err
	}
	if in.Author != nil && !sarValidAuthor(*in.Author) {
		return 0, nil, sarBadRequest("author must be 1-127 lowercase letters, digits and single hyphens.")
	}
	if in.Description != nil && (*in.Description == "" || len(*in.Description) > 256) {
		return 0, nil, sarBadRequest("description must be 1-256 characters.")
	}
	if in.ReadmeBody != nil && in.ReadmeURL != nil {
		return 0, nil, sarBadRequest("Specify readmeBody or readmeUrl, not both.")
	}
	if err := sarCheckLabels(in.Labels); err != nil {
		return 0, nil, err
	}
	// Check the application first, so a missing one doesn't leave a stored
	// readme behind.
	if _, err := s.serverlessRepo.get(app, ""); err != nil {
		return 0, nil, err
	}
	readme := ""
	if in.ReadmeBody != nil || in.ReadmeURL != nil {
		var body, u string
		if in.ReadmeBody != nil {
			body = *in.ReadmeBody
		}
		if in.ReadmeURL != nil {
			u = *in.ReadmeURL
		}
		name := app[strings.LastIndex(app, "/")+1:]
		var err *sarError
		if readme, err = s.sarDocument(r, "readmeUrl", u, body, name, "README.md"); err != nil {
			return 0, nil, err
		}
	}
	out, err := s.serverlessRepo.update(app, in, readme)
	return http.StatusOK, out, err
}

func (s *Server) sarCreateVersion(r *http.Request, app, semver string) (int, map[string]any, *sarError) {
	var in sarVersionInput
	if err := sarDecode(r, &in); err != nil {
		return 0, nil, err
	}
	if in.TemplateBody == "" && in.TemplateURL == "" {
		return 0, nil, sarBadRequest("templateBody or templateUrl is required.")
	}
	if in.TemplateBody != "" && in.TemplateURL != "" {
		return 0, nil, sarBadRequest("Specify templateBody or templateUrl, not both.")
	}
	// Check the application first, so a missing one doesn't leave a stored
	// template behind.
	if _, err := s.serverlessRepo.get(app, ""); err != nil {
		return 0, nil, err
	}
	v, err := s.sarNewVersion(r, app[strings.LastIndex(app, "/")+1:], semver, in)
	if err != nil {
		return 0, nil, err
	}
	out, err := s.serverlessRepo.addVersion(app, v)
	return http.StatusCreated, out, err
}

// sarNewVersion validates a version and loads its template. S3 is read here,
// before the repository lock is taken.
func (s *Server) sarNewVersion(r *http.Request, appName, semver string, in sarVersionInput) (*sarVersion, *sarError) {
	if !sarSemverRe.MatchString(semver) || len(semver) > 64 {
		return nil, sarBadRequest("semanticVersion %q isn't a valid Semantic Version 2.0.0 string.", semver)
	}
	body, templateURL := in.TemplateBody, in.TemplateURL
	if templateURL != "" {
		raw, err := s.sarFetchS3(r, "templateUrl", templateURL)
		if err != nil {
			return nil, err
		}
		body = string(raw)
	}
	tpl, err := sarParseTemplate(body)
	if err != nil {
		return nil, err
	}
	if templateURL == "" {
		if templateURL, err = s.sarPutDoc(appName+"/"+semver+"/template.yaml", "application/x-yaml", []byte(body)); err != nil {
			return nil, err
		}
	}
	if in.SourceCodeArchiveURL != "" {
		if _, err := s.sarFetchS3(r, "sourceCodeArchiveUrl", in.SourceCodeArchiveURL); err != nil {
			return nil, err
		}
	}
	return &sarVersion{
		semver: semver, created: sarNow(), templateURL: templateURL, templateBody: body,
		sourceCodeURL: in.SourceCodeURL, sourceArchiveURL: in.SourceCodeArchiveURL, tpl: tpl,
	}, nil
}

func (s *Server) sarPutPolicy(r *http.Request, app string) (int, map[string]any, *sarError) {
	var in struct {
		Statements []sarStatementInput `json:"statements"`
	}
	if err := sarDecode(r, &in); err != nil {
		return 0, nil, err
	}
	if in.Statements == nil {
		return 0, nil, sarBadRequest("statements is required.")
	}
	if len(in.Statements) > sarMaxStatements {
		return 0, nil, sarBadRequest("A policy can have at most %d statements.", sarMaxStatements)
	}
	var stored []map[string]any
	for i, st := range in.Statements {
		if len(st.Actions) == 0 || len(st.Principals) == 0 {
			return 0, nil, sarBadRequest("statements[%d] needs actions and principals.", i)
		}
		for _, a := range st.Actions {
			if !sarAllowedActions[a] {
				return 0, nil, sarBadRequest("statements[%d]: action %q isn't supported.", i, a)
			}
		}
		for _, p := range st.Principals {
			if p != "*" && !sarAccountRe.MatchString(p) {
				return 0, nil, sarBadRequest("statements[%d]: principal %q must be an AWS account ID or *.", i, p)
			}
		}
		if len(st.PrincipalOrgIDs) > 0 && !(len(st.Principals) == 1 && st.Principals[0] == "*") {
			return 0, nil, sarBadRequest("statements[%d]: principalOrgIDs requires principals to be [\"*\"].", i)
		}
		for _, o := range st.PrincipalOrgIDs {
			if !sarOrgRe.MatchString(o) {
				return 0, nil, sarBadRequest("statements[%d]: %q isn't an AWS Organizations ID.", i, o)
			}
		}
		id := st.StatementID
		if id == "" {
			id = sarNewID()
		}
		m := map[string]any{"statementId": id, "actions": append([]string{}, st.Actions...), "principals": append([]string{}, st.Principals...)}
		if len(st.PrincipalOrgIDs) > 0 {
			m["principalOrgIDs"] = append([]string{}, st.PrincipalOrgIDs...)
		}
		stored = append(stored, m)
	}
	out, err := s.serverlessRepo.putPolicy(app, stored)
	return http.StatusOK, out, err
}

func (s *Server) sarCreateTemplate(r *http.Request, app string) (int, map[string]any, *sarError) {
	var in struct {
		SemanticVersion string `json:"semanticVersion"`
	}
	if err := sarDecode(r, &in); err != nil {
		return 0, nil, err
	}
	appID, v, err := s.serverlessRepo.templateSource(app, in.SemanticVersion)
	if err != nil {
		return 0, nil, err
	}
	id := sarNewID()
	name := appID[strings.LastIndex(appID, "/")+1:]
	// One object per version: repeated calls overwrite rather than grow S3.
	u, err := s.sarPutDoc(name+"/templates/"+v.semver+".yaml", "application/x-yaml", []byte(v.templateBody))
	if err != nil {
		return 0, nil, err
	}
	now := time.Now().UTC()
	g := &sarGenerated{
		id: id, semver: v.semver, created: now.Format("2006-01-02T15:04:05.000Z"),
		expires: now.Add(sarTemplateExpiry).Format("2006-01-02T15:04:05.000Z"), url: u,
	}
	out, err := s.serverlessRepo.addTemplate(appID, g)
	return http.StatusCreated, out, err
}

func (s *Server) sarCreateChangeSet(r *http.Request, app string) (int, map[string]any, *sarError) {
	var in sarChangeSetInput
	if err := sarDecode(r, &in); err != nil {
		return 0, nil, err
	}
	if prev := s.serverlessRepo.changeSetForToken(in.ClientToken); prev != nil {
		return http.StatusCreated, prev, nil
	}
	if !sarStackRe.MatchString(in.StackName) {
		return 0, nil, sarBadRequest("stackName must start with a letter and contain only letters, digits and hyphens (at most 114 characters).")
	}
	if in.ChangeSetName != "" && !sarStackRe.MatchString(in.ChangeSetName) {
		return 0, nil, sarBadRequest("changeSetName must start with a letter and contain only letters, digits and hyphens.")
	}
	for _, t := range in.Tags {
		if t.Key == "" || len(t.Key) > 127 || len(t.Value) > 255 || strings.HasPrefix(t.Key, "aws:") {
			return 0, nil, sarBadRequest("Tag keys must be 1-127 characters without the aws: prefix, and values at most 255.")
		}
	}
	if len(in.Tags) > 50 {
		return 0, nil, sarBadRequest("At most 50 tags are allowed.")
	}
	appID, v, err := s.serverlessRepo.templateSource(app, in.SemanticVersion)
	if err != nil {
		return 0, nil, err
	}
	if in.TemplateID != "" {
		_, g, err := s.serverlessRepo.getTemplate(appID, in.TemplateID)
		if err != nil {
			return 0, nil, err
		}
		if g.semver != v.semver {
			return 0, nil, sarBadRequest("Template %s was generated for version %s, not %s.", in.TemplateID, g.semver, v.semver)
		}
	}
	if err := sarCheckOverrides(v.tpl, in); err != nil {
		return 0, nil, err
	}
	given := map[string]bool{}
	for _, c := range in.Capabilities {
		given[c] = true
	}
	var missing []string
	for _, c := range v.tpl.capabilities {
		if !given[c] {
			missing = append(missing, c)
		}
	}
	if len(missing) > 0 {
		return 0, nil, sarBadRequest("Requires capabilities: [%s]", strings.Join(missing, ", "))
	}
	stack := "serverlessrepo-" + in.StackName
	csName := in.ChangeSetName
	if csName == "" {
		csName = "a" + strings.ReplaceAll(sarNewID(), "-", "")
	}
	stackID, csID := s.sarRegisterChangeSet(stack, csName)
	out := map[string]any{"applicationId": appID, "changeSetId": csID, "semanticVersion": v.semver, "stackId": stackID}
	s.serverlessRepo.rememberChangeSet(in.ClientToken, out)
	return http.StatusCreated, out, nil
}

// sarCheckOverrides validates parameter overrides against the version's
// parameter definitions: every name must be defined, required parameters
// (no default) must be given, and allowed values and patterns hold.
func sarCheckOverrides(tpl *sarTemplate, in sarChangeSetInput) *sarError {
	defs := map[string]map[string]any{}
	for _, d := range tpl.params {
		defs[d["name"].(string)] = d
	}
	given := map[string]string{}
	for _, o := range in.ParameterOverrides {
		d, ok := defs[o.Name]
		if !ok {
			return sarBadRequest("Parameter %s isn't defined by the application.", o.Name)
		}
		if allowed, ok := d["allowedValues"].([]string); ok && len(allowed) > 0 {
			found := false
			for _, a := range allowed {
				found = found || a == o.Value
			}
			if !found {
				return sarBadRequest("Parameter %s must be one of [%s].", o.Name, strings.Join(allowed, ", "))
			}
		}
		given[o.Name] = o.Value
	}
	for name, d := range defs {
		if _, hasDefault := d["defaultValue"]; !hasDefault {
			if _, ok := given[name]; !ok {
				return sarBadRequest("Parameter %s is required.", name)
			}
		}
	}
	return nil
}

// sarRegisterChangeSet records the stack and change set in the CloudFormation
// store, after the repository lock has been released.
func (s *Server) sarRegisterChangeSet(stack, changeSet string) (string, string) {
	s.cloudformation.mu.Lock()
	defer s.cloudformation.mu.Unlock()
	stackID, ok := s.cloudformation.stacks[stack]
	if !ok {
		stackID = "arn:aws:cloudformation:" + sarRegion + ":" + sarAccountID + ":stack/" + stack + "/" + sarNewID()
		s.cloudformation.stacks[stack] = stackID
	}
	csID := "arn:aws:cloudformation:" + sarRegion + ":" + sarAccountID + ":changeSet/" + changeSet + "/" + sarNewID()
	s.cloudformation.changeSets[changeSet] = csID
	return stackID, csID
}

// sarDocument returns the URL of a readme or license: a given URL must name an
// S3 object; a given body is stored in Stackyard's S3.
func (s *Server) sarDocument(r *http.Request, field, u, body, appName, file string) (string, *sarError) {
	if u != "" {
		if _, err := s.sarFetchS3(r, field, u); err != nil {
			return "", err
		}
		return u, nil
	}
	if body == "" {
		return "", nil
	}
	if len(body) > 1<<20 {
		return "", sarBadRequest("%s body exceeds 1 MiB.", strings.TrimSuffix(field, "Url"))
	}
	return s.sarPutDoc(appName+"/"+file, "text/plain", []byte(body))
}

// sarFetchS3 reads an object named by a virtual-hosted, path-style or s3://
// URL from Stackyard's S3 store.
func (s *Server) sarFetchS3(r *http.Request, field, raw string) ([]byte, *sarError) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, sarBadRequest("%s %q isn't a URL.", field, raw)
	}
	host, path := u.Hostname(), strings.TrimPrefix(u.Path, "/")
	var bucket, key string
	switch {
	case u.Scheme == "s3":
		bucket, key = host, path
	case strings.Contains(host, ".s3") && strings.HasSuffix(host, ".amazonaws.com"):
		bucket, key = host[:strings.Index(host, ".s3")], path
	default:
		bucket, key, _ = strings.Cut(path, "/")
	}
	if bucket == "" || key == "" {
		return nil, sarBadRequest("%s %q doesn't name an S3 object.", field, raw)
	}
	if !s.authorizeServiceS3(r, "serverlessrepo.amazonaws.com", bucket, key, "s3:GetObject") {
		return nil, &sarError{http.StatusForbidden, "ForbiddenException", field + ": access denied reading s3://" + bucket + "/" + key + "."}
	}
	obj, err := s.s3.GetObject(bucket, key)
	if err != nil {
		return nil, sarBadRequest("%s: s3://%s/%s could not be read.", field, bucket, key)
	}
	if len(obj.Body) > sarMaxTemplateBytes {
		return nil, sarBadRequest("%s: the object exceeds %d bytes.", field, sarMaxTemplateBytes)
	}
	return obj.Body, nil
}

// sarPutDoc writes a document to the repository's bucket in Stackyard's S3 and
// returns its URL.
func (s *Server) sarPutDoc(key, contentType string, body []byte) (string, *sarError) {
	_, _ = s.s3.CreateBucket(sarBucket) // already existing is fine
	if _, err := s.s3.PutObject(sarBucket, key, contentType, body, nil, "", ""); err != nil {
		return "", &sarError{http.StatusInternalServerError, "InternalServerErrorException", "could not store " + key}
	}
	return "https://" + sarBucket + ".s3.amazonaws.com/" + key, nil
}

func sarCheckLabels(labels []string) *sarError {
	if len(labels) > sarMaxLabels {
		return sarBadRequest("At most %d labels are allowed.", sarMaxLabels)
	}
	for _, l := range labels {
		if !sarLabelRe.MatchString(l) {
			return sarBadRequest("Label %q must be 1-127 characters of letters, digits and +-_:/@.", l)
		}
	}
	return nil
}

func sarMaxItems(q url.Values) (int, *sarError) {
	raw := q.Get("maxItems")
	if raw == "" {
		return 100, nil
	}
	n, err := strconv.ParseInt(raw, 10, 32)
	if err != nil || n < 1 || n > 100 {
		return 0, sarBadRequest("maxItems must be between 1 and 100.")
	}
	return int(n), nil
}

// sarDecode reads a JSON body into a typed struct, rejecting unknown members.
func sarDecode(r *http.Request, v any) *sarError {
	raw, err := readBodyBytes(r)
	if err != nil {
		return sarBadRequest("Unable to read the request body.")
	}
	if len(bytes.TrimSpace(raw)) == 0 {
		return nil
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return sarBadRequest("The request body is invalid: %v", err)
	}
	return nil
}

func respondSARJSON(w http.ResponseWriter, status int, body map[string]any) {
	w.Header().Set("Content-Type", "application/json")
	if status == http.StatusNoContent {
		w.WriteHeader(status)
		return
	}
	raw, _ := json.Marshal(body)
	w.WriteHeader(status)
	_, _ = w.Write(raw)
}

func respondSARError(w http.ResponseWriter, e *sarError) {
	raw, _ := json.Marshal(map[string]any{"errorCode": e.code, "message": e.msg})
	w.Header().Set("X-Amzn-ErrorType", e.code)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(e.status)
	_, _ = w.Write(raw)
}
