package server

import (
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	sarAccountID      = "123456789012"
	sarRegion         = "us-east-1"
	sarBucket         = "stackyard-serverlessrepo"
	sarMaxApps        = 1000
	sarMaxVersions    = 100 // per application
	sarMaxTemplates   = 100 // generated templates per application
	sarMaxStatements  = 50
	sarMaxLabels      = 10
	sarMaxChangeSets  = 1000 // remembered client tokens
	sarTemplateExpiry = 6 * time.Hour
	sarMaxStoredBytes = 256 << 20 // template text held across all applications
)

// sarAllowedActions are the actions an application policy may grant.
var sarAllowedActions = map[string]bool{
	"GetApplication": true, "CreateCloudFormationChangeSet": true, "CreateCloudFormationTemplate": true,
	"ListApplicationVersions": true, "SearchApplications": true, "Deploy": true, "UnshareApplication": true,
}

var (
	sarNameRe    = regexp.MustCompile(`^[a-zA-Z0-9\-]{1,140}$`)
	sarLabelRe   = regexp.MustCompile(`^[a-zA-Z0-9+\-_:/@]{1,127}$`)
	sarSemverRe  = regexp.MustCompile(`^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$`)
	sarAccountRe = regexp.MustCompile(`^\d{12}$`)
	sarOrgRe     = regexp.MustCompile(`^o-[a-z0-9]{10,32}$`)
	sarStackRe   = regexp.MustCompile(`^[a-zA-Z][-a-zA-Z0-9]{0,113}$`)
)

type sarError struct {
	status int
	code   string
	msg    string
}

func sarBadRequest(format string, args ...any) *sarError {
	return &sarError{http.StatusBadRequest, "BadRequestException", fmt.Sprintf(format, args...)}
}

func sarNotFound(format string, args ...any) *sarError {
	return &sarError{http.StatusNotFound, "NotFoundException", fmt.Sprintf(format, args...)}
}

func sarConflict(format string, args ...any) *sarError {
	return &sarError{http.StatusConflict, "ConflictException", fmt.Sprintf(format, args...)}
}

type sarVersion struct {
	semver, created                 string
	templateURL, templateBody       string
	sourceCodeURL, sourceArchiveURL string
	tpl                             *sarTemplate
}

type sarGenerated struct {
	id, semver, created, expires, url string
	reads                             int
}

type sarApp struct {
	id, name, author, description            string
	homePageURL, spdx, licenseURL, readmeURL string
	labels                                   []string
	created                                  string
	versions                                 map[string]*sarVersion
	order                                    []string // version creation order
	policy                                   []map[string]any
	templates                                map[string]*sarGenerated
}

type sarStore struct {
	mu         sync.Mutex
	apps       map[string]*sarApp // by ARN
	changeSets map[string]map[string]any
	tokens     []string // changeSet client tokens, oldest first
	bytes      int      // template text held across all versions
}

func newServerlessRepoStore() *sarStore {
	return &sarStore{apps: map[string]*sarApp{}, changeSets: map[string]map[string]any{}}
}

func sarNow() string { return time.Now().UTC().Format("2006-01-02T15:04:05.000Z") }

func sarAppARN(name string) string {
	return "arn:aws:serverlessrepo:" + sarRegion + ":" + sarAccountID + ":applications/" + name
}

// app resolves an application by ARN or bare name.
func (st *sarStore) appLocked(id string) (*sarApp, *sarError) {
	if a := st.apps[id]; a != nil {
		return a, nil
	}
	if a := st.apps[sarAppARN(id)]; a != nil {
		return a, nil
	}
	return nil, sarNotFound("Application %s not found.", id)
}

func (a *sarApp) latest() *sarVersion {
	if len(a.order) == 0 {
		return nil
	}
	return a.versions[a.order[len(a.order)-1]]
}

func (a *sarApp) version(semver string) (*sarVersion, *sarError) {
	if semver == "" {
		if v := a.latest(); v != nil {
			return v, nil
		}
		return nil, sarNotFound("Application %s has no versions.", a.id)
	}
	if v := a.versions[semver]; v != nil {
		return v, nil
	}
	return nil, sarNotFound("Version %s of application %s not found.", semver, a.id)
}

func (v *sarVersion) out(appID string) map[string]any {
	out := map[string]any{
		"applicationId": appID, "creationTime": v.created, "semanticVersion": v.semver,
		"templateUrl": v.templateURL, "parameterDefinitions": v.tpl.params,
		"requiredCapabilities": v.tpl.capabilities, "resourcesSupported": true,
	}
	if v.sourceCodeURL != "" {
		out["sourceCodeUrl"] = v.sourceCodeURL
	}
	if v.sourceArchiveURL != "" {
		out["sourceCodeArchiveUrl"] = v.sourceArchiveURL
	}
	return out
}

func (a *sarApp) out(v *sarVersion) map[string]any {
	out := a.summary()
	out["isVerifiedAuthor"] = false
	for k, val := range map[string]string{"licenseUrl": a.licenseURL, "readmeUrl": a.readmeURL} {
		if val != "" {
			out[k] = val
		}
	}
	if v != nil {
		out["version"] = v.out(a.id)
	}
	return out
}

func (a *sarApp) summary() map[string]any {
	out := map[string]any{
		"applicationId": a.id, "name": a.name, "author": a.author, "description": a.description,
		"creationTime": a.created, "labels": append([]string{}, a.labels...),
	}
	if a.homePageURL != "" {
		out["homePageUrl"] = a.homePageURL
	}
	if a.spdx != "" {
		out["spdxLicenseId"] = a.spdx
	}
	return out
}

// sarAppMeta holds the validated, already-fetched pieces of a create call.
type sarAppMeta struct {
	in                    sarCreateAppInput
	licenseURL, readmeURL string
	version               *sarVersion
}

func (st *sarStore) create(m *sarAppMeta) (map[string]any, *sarError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	id := sarAppARN(m.in.Name)
	if st.apps[id] != nil {
		return nil, sarConflict("An application named %s already exists.", m.in.Name)
	}
	if len(st.apps) >= sarMaxApps {
		return nil, &sarError{http.StatusTooManyRequests, "TooManyRequestsException", fmt.Sprintf("At most %d applications can exist.", sarMaxApps)}
	}
	a := &sarApp{
		id: id, name: m.in.Name, author: m.in.Author, description: m.in.Description,
		homePageURL: m.in.HomePageURL, spdx: m.in.SpdxLicenseID, licenseURL: m.licenseURL, readmeURL: m.readmeURL,
		labels: append([]string{}, m.in.Labels...), created: sarNow(),
		versions: map[string]*sarVersion{}, templates: map[string]*sarGenerated{},
	}
	if m.version != nil {
		if st.bytes+len(m.version.templateBody) > sarMaxStoredBytes {
			return nil, &sarError{http.StatusTooManyRequests, "TooManyRequestsException", "The repository's template storage is full."}
		}
		st.bytes += len(m.version.templateBody)
		a.versions[m.version.semver] = m.version
		a.order = append(a.order, m.version.semver)
	}
	st.apps[id] = a
	return a.out(m.version), nil
}

func (st *sarStore) get(id, semver string) (map[string]any, *sarError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	a, err := st.appLocked(id)
	if err != nil {
		return nil, err
	}
	var v *sarVersion
	if semver != "" || len(a.order) > 0 {
		if v, err = a.version(semver); err != nil {
			return nil, err
		}
	}
	return a.out(v), nil
}

func (st *sarStore) update(id string, in sarUpdateAppInput, readmeURL string) (map[string]any, *sarError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	a, err := st.appLocked(id)
	if err != nil {
		return nil, err
	}
	if in.Author != nil {
		a.author = *in.Author
	}
	if in.Description != nil {
		a.description = *in.Description
	}
	if in.HomePageURL != nil {
		a.homePageURL = *in.HomePageURL
	}
	if in.Labels != nil {
		a.labels = append([]string{}, in.Labels...)
	}
	if readmeURL != "" {
		a.readmeURL = readmeURL
	}
	return a.out(a.latest()), nil
}

func (st *sarStore) delete(id string) *sarError {
	st.mu.Lock()
	defer st.mu.Unlock()
	a, err := st.appLocked(id)
	if err != nil {
		return err
	}
	for _, v := range a.versions {
		st.bytes -= len(v.templateBody)
	}
	delete(st.apps, a.id)
	return nil
}

func (st *sarStore) list(token string, max int) (map[string]any, *sarError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	apps := make([]*sarApp, 0, len(st.apps))
	for _, a := range st.apps {
		apps = append(apps, a)
	}
	sort.Slice(apps, func(i, j int) bool {
		if apps[i].created != apps[j].created {
			return apps[i].created < apps[j].created
		}
		return apps[i].id < apps[j].id
	})
	start, end, next, err := sarPage(token, max, len(apps))
	if err != nil {
		return nil, err
	}
	out := []any{}
	for _, a := range apps[start:end] {
		out = append(out, a.summary())
	}
	return sarWithToken(map[string]any{"applications": out}, next), nil
}

func (st *sarStore) addVersion(id string, v *sarVersion) (map[string]any, *sarError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	a, err := st.appLocked(id)
	if err != nil {
		return nil, err
	}
	if a.versions[v.semver] != nil {
		return nil, sarConflict("Version %s of application %s already exists.", v.semver, a.id)
	}
	if len(a.order) >= sarMaxVersions {
		return nil, sarBadRequest("An application can have at most %d versions.", sarMaxVersions)
	}
	if st.bytes+len(v.templateBody) > sarMaxStoredBytes {
		return nil, &sarError{http.StatusTooManyRequests, "TooManyRequestsException", "The repository's template storage is full."}
	}
	st.bytes += len(v.templateBody)
	a.versions[v.semver] = v
	a.order = append(a.order, v.semver)
	return v.out(a.id), nil
}

func (st *sarStore) listVersions(id, token string, max int) (map[string]any, *sarError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	a, err := st.appLocked(id)
	if err != nil {
		return nil, err
	}
	start, end, next, perr := sarPage(token, max, len(a.order))
	if perr != nil {
		return nil, perr
	}
	out := []any{}
	for _, s := range a.order[start:end] {
		v := a.versions[s]
		m := map[string]any{"applicationId": a.id, "creationTime": v.created, "semanticVersion": v.semver}
		if v.sourceCodeURL != "" {
			m["sourceCodeUrl"] = v.sourceCodeURL
		}
		out = append(out, m)
	}
	return sarWithToken(map[string]any{"versions": out}, next), nil
}

func (st *sarStore) listDependencies(id, semver, token string, max int) (map[string]any, *sarError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	a, err := st.appLocked(id)
	if err != nil {
		return nil, err
	}
	v, err := a.version(semver)
	if err != nil {
		return nil, err
	}
	deps := v.tpl.dependencies
	start, end, next, perr := sarPage(token, max, len(deps))
	if perr != nil {
		return nil, perr
	}
	out := []any{}
	for _, d := range deps[start:end] {
		out = append(out, map[string]any{"applicationId": d.applicationID, "semanticVersion": d.semanticVersion})
	}
	return sarWithToken(map[string]any{"dependencies": out}, next), nil
}

func (st *sarStore) putPolicy(id string, statements []map[string]any) (map[string]any, *sarError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	a, err := st.appLocked(id)
	if err != nil {
		return nil, err
	}
	a.policy = statements
	return map[string]any{"statements": sarCopyStatements(a.policy)}, nil
}

func (st *sarStore) getPolicy(id string) (map[string]any, *sarError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	a, err := st.appLocked(id)
	if err != nil {
		return nil, err
	}
	return map[string]any{"statements": sarCopyStatements(a.policy)}, nil
}

// unshare removes an organization from every statement that names it, and
// drops statements left without any principal organization.
func (st *sarStore) unshare(id, org string) *sarError {
	st.mu.Lock()
	defer st.mu.Unlock()
	a, err := st.appLocked(id)
	if err != nil {
		return err
	}
	found := false
	var kept []map[string]any
	for _, s := range a.policy {
		orgs, _ := s["principalOrgIDs"].([]string)
		var rest []string
		for _, o := range orgs {
			if o == org {
				found = true
				continue
			}
			rest = append(rest, o)
		}
		if len(orgs) > 0 && len(rest) == 0 {
			continue
		}
		// Stored statements are replaced, never changed in place, so a response
		// still being encoded from an earlier copy is never raced.
		ns := map[string]any{}
		for k, v := range s {
			ns[k] = v
		}
		if len(orgs) > 0 {
			ns["principalOrgIDs"] = rest
		}
		kept = append(kept, ns)
	}
	if !found {
		return sarNotFound("Application %s isn't shared with organization %s.", a.id, org)
	}
	a.policy = kept
	return nil
}

// templateSource returns the version whose template a generated template or
// change set uses.
func (st *sarStore) templateSource(id, semver string) (string, *sarVersion, *sarError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	a, err := st.appLocked(id)
	if err != nil {
		return "", nil, err
	}
	v, err := a.version(semver)
	if err != nil {
		return "", nil, err
	}
	return a.id, v, nil
}

func (st *sarStore) addTemplate(appID string, g *sarGenerated) (map[string]any, *sarError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	a, err := st.appLocked(appID)
	if err != nil {
		return nil, err
	}
	if len(a.templates) >= sarMaxTemplates {
		// Drop the oldest; generated templates expire anyway.
		var oldest *sarGenerated
		for _, t := range a.templates {
			if oldest == nil || t.created < oldest.created {
				oldest = t
			}
		}
		delete(a.templates, oldest.id)
	}
	a.templates[g.id] = g
	return g.out(a.id, "PREPARING"), nil
}

func (g *sarGenerated) out(appID, status string) map[string]any {
	return map[string]any{
		"applicationId": appID, "creationTime": g.created, "expirationTime": g.expires,
		"semanticVersion": g.semver, "status": status, "templateId": g.id, "templateUrl": g.url,
	}
}

// getTemplate reports PREPARING on the first read and ACTIVE afterwards, or
// EXPIRED once the expiration time has passed.
func (st *sarStore) getTemplate(appID, templateID string) (map[string]any, *sarGenerated, *sarError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	a, err := st.appLocked(appID)
	if err != nil {
		return nil, nil, err
	}
	g := a.templates[templateID]
	if g == nil {
		return nil, nil, sarNotFound("Template %s of application %s not found.", templateID, a.id)
	}
	status := "ACTIVE"
	if g.reads == 0 {
		status = "PREPARING"
	}
	g.reads++
	if g.expires < sarNow() {
		status = "EXPIRED"
	}
	copied := *g
	return g.out(a.id, status), &copied, nil
}

// rememberChangeSet stores a change set response under its client token, so a
// retried request gets the same answer; the oldest tokens are forgotten first.
func (st *sarStore) changeSetForToken(token string) map[string]any {
	st.mu.Lock()
	defer st.mu.Unlock()
	if token == "" {
		return nil
	}
	return st.changeSets[token]
}

func (st *sarStore) rememberChangeSet(token string, out map[string]any) {
	if token == "" {
		return
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	if _, ok := st.changeSets[token]; !ok {
		st.tokens = append(st.tokens, token)
		if len(st.tokens) > sarMaxChangeSets {
			delete(st.changeSets, st.tokens[0])
			st.tokens = st.tokens[1:]
		}
	}
	st.changeSets[token] = out
}

// sarPage turns a decimal offset token into slice bounds.
func sarPage(token string, max, n int) (int, int, string, *sarError) {
	start := 0
	if token != "" {
		v, err := strconv.ParseInt(token, 10, 32)
		if err != nil || v < 0 || int(v) > n {
			return 0, 0, "", sarBadRequest("The nextToken is invalid.")
		}
		start = int(v)
	}
	end := min(start+max, n)
	next := ""
	if end < n {
		next = strconv.Itoa(end)
	}
	return start, end, next, nil
}

func sarWithToken(out map[string]any, next string) map[string]any {
	if next != "" {
		out["nextToken"] = next
	}
	return out
}

func sarNewID() string { return uuid.NewString() }

// sarValidAuthor checks the author pattern without its lookahead:
// lowercase letters, digits and single hyphens, not at either end.
func sarValidAuthor(s string) bool {
	if s == "" || len(s) > 127 || strings.HasPrefix(s, "-") || strings.HasSuffix(s, "-") || strings.Contains(s, "--") {
		return false
	}
	for _, r := range s {
		if !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-') {
			return false
		}
	}
	return true
}

// sarCopyStatements copies policy statements (and their string lists) for a
// response, so nothing returned aliases stored state.
func sarCopyStatements(in []map[string]any) []any {
	out := []any{}
	for _, s := range in {
		c := map[string]any{}
		for k, v := range s {
			if list, ok := v.([]string); ok {
				v = append([]string{}, list...)
			}
			c[k] = v
		}
		out = append(out, c)
	}
	return out
}
