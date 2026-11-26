package server

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Scans. CreateUploadUrl reserves a code artifact and returns a one-hour URL
// on this server that takes a zip (the stand-in for S3's pre-signed PUT); the
// upload is scanned as it arrives. CreateScan starts a run over an uploaded
// artifact: it reports InProgress for cgsScanDelay, then Successful (or
// Failed when nothing was uploaded). A run's findings carry over first-seen
// times from the scan's previous run, and findings that disappear are
// reported Closed.

const (
	cgsUploadPath     = "/_stackyard/codegurusecurity/uploads/"
	cgsUploadLifetime = time.Hour
	cgsZipType        = "application/zip"
	cgsMaxArtifacts   = 200 // the oldest artifact is dropped past this
	cgsMaxScans       = 100
	cgsMaxRuns        = 5 // runs kept per scan
	cgsMaxTags        = 50
	cgsMaxDays        = 400 // daily metric buckets kept
	cgsScanDelay      = 2 * time.Second
)

var cgsSeverities = []string{"Critical", "High", "Medium", "Low", "Info"}

func cgsSeverityIndex(sev string) int {
	for i, s := range cgsSeverities {
		if s == sev {
			return i
		}
	}
	return len(cgsSeverities) - 1
}

type cgsArtifact struct {
	id, scanName, key string
	created, expiry   time.Time
	uploaded          bool
	matches           []cgsMatch // replaced on re-upload, never mutated
}

type cgsFinding struct {
	id, path, status string
	rule, line       int
	snippet          []cgsLine
	created, updated time.Time
}

type cgsRun struct {
	id, state, analysis, scanType, artifact, token, errMsg string
	created, updated, due                                  time.Time
	matches                                                []cgsMatch
	findings                                               []*cgsFinding
	done                                                   bool
}

type cgsScan struct {
	name, arn string
	created   time.Time
	tags      map[string]string
	runs      []*cgsRun // oldest first
	revisions int
}

// cgsDay is one day of account-wide finding metrics, per severity.
type cgsDay struct {
	opened, closed, open, closeDays [5]float64
	closedN                         [5]int
}

type cgsStore struct {
	mu        sync.Mutex
	artifacts map[string]*cgsArtifact
	scans     map[string]*cgsScan
	days      map[string]*cgsDay // by yyyy-mm-dd
	kmsKeyArn string
}

func newCodeGuruSecurityStore() *cgsStore {
	return &cgsStore{artifacts: map[string]*cgsArtifact{}, scans: map[string]*cgsScan{}, days: map[string]*cgsDay{}}
}

func cgsToken() string {
	b := make([]byte, 24)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// ---- uploads ----

func (s *Server) cgsCreateUploadURL(req cgsRequest) (map[string]any, *cgsErr) {
	st := s.codeGuruSecurity
	st.mu.Lock()
	defer st.mu.Unlock()
	if len(st.artifacts) >= cgsMaxArtifacts {
		oldest := ""
		for id, a := range st.artifacts {
			if oldest == "" || a.created.Before(st.artifacts[oldest].created) {
				oldest = id
			}
		}
		delete(st.artifacts, oldest)
	}
	now := time.Now().UTC()
	a := &cgsArtifact{id: uuid.NewString(), scanName: c9Str(req.in, "scanName"), key: cgsToken(), created: now, expiry: now.Add(cgsUploadLifetime)}
	st.artifacts[a.id] = a
	return map[string]any{
		"s3Url":          req.base + cgsUploadPath + a.id + "?key=" + url.QueryEscape(a.key),
		"requestHeaders": map[string]any{"Content-Type": cgsZipType},
		"codeArtifactId": a.id,
	}, nil
}

// handleCodeGuruSecurityUpload takes the zip for a code artifact and scans it.
func (s *Server) handleCodeGuruSecurityUpload(w http.ResponseWriter, r *http.Request) {
	st := s.codeGuruSecurity
	id, key := r.PathValue("id"), r.URL.Query().Get("key")
	st.mu.Lock()
	a := st.artifacts[id]
	ok := a != nil && subtle.ConstantTimeCompare([]byte(a.key), []byte(key)) == 1 && time.Now().Before(a.expiry)
	st.mu.Unlock()
	if !ok {
		cgsPlainError(w, http.StatusForbidden, "the upload URL is not valid or has expired")
		return
	}
	if r.ContentLength <= 0 || r.ContentLength > cgsMaxUpload {
		cgsPlainError(w, http.StatusBadRequest, fmt.Sprintf("the body must declare a Content-Length from 1 to %d bytes", cgsMaxUpload))
		return
	}
	if ct, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type")); ct != cgsZipType {
		cgsPlainError(w, http.StatusBadRequest, "Content-Type must be "+cgsZipType)
		return
	}
	// The body is read and scanned outside the lock; both are bounded.
	data, err := io.ReadAll(io.LimitReader(r.Body, r.ContentLength))
	if err != nil || int64(len(data)) != r.ContentLength {
		cgsPlainError(w, http.StatusBadRequest, "the body ended early")
		return
	}
	matches, _, err := cgsScanZip(data)
	if err != nil {
		cgsPlainError(w, http.StatusBadRequest, "the body is not a valid zip archive")
		return
	}
	st.mu.Lock()
	stored := st.artifacts[id] == a
	if stored {
		a.uploaded, a.matches = true, matches
	}
	st.mu.Unlock()
	if !stored {
		cgsPlainError(w, http.StatusConflict, "the code artifact was removed")
		return
	}
	w.WriteHeader(http.StatusOK)
}

func cgsPlainError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_, _ = io.WriteString(w, msg+"\n")
}

// ---- scan runs ----

func cgsScanArn(name string) string {
	return "arn:aws:codeguru-security:" + cgsRegion + ":" + cgsAccount + ":scans/" + name
}

func (sc *cgsScan) latest() *cgsRun { return sc.runs[len(sc.runs)-1] }

// lastDone returns the newest finished run, or nil.
func (sc *cgsScan) lastDone() *cgsRun {
	for i := len(sc.runs) - 1; i >= 0; i-- {
		if sc.runs[i].done {
			return sc.runs[i]
		}
	}
	return nil
}

// advance finishes the runs that are due; callers hold st.mu.
func (st *cgsStore) advance(now time.Time) {
	names := make([]string, 0, len(st.scans))
	for name, sc := range st.scans {
		if run := sc.latest(); !run.done && !now.Before(run.due) {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	for _, name := range names {
		sc := st.scans[name]
		st.finish(sc, sc.latest().due) // finished when due, whenever it is observed
	}
}

func cgsFindingID(scan string, m cgsMatch) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s\x00%s\x00%s\x00%d", scan, cgsRules[m.rule].id, m.path, m.line)))
	return hex.EncodeToString(sum[:20])
}

// finish completes the scan's latest run at time now and records its
// metrics; callers hold st.mu.
func (st *cgsStore) finish(sc *cgsScan, now time.Time) {
	run := sc.latest()
	prev := sc.lastDone()
	run.done, run.updated = true, now
	if run.errMsg != "" {
		run.state = "Failed"
		return
	}
	run.state = "Successful"
	earlier := map[string]*cgsFinding{}
	if prev != nil {
		for _, f := range prev.findings {
			if f.status == "Open" {
				earlier[f.id] = f
			}
		}
	}
	day := st.day(now)
	seen := map[string]bool{}
	for _, m := range run.matches {
		rule := cgsRules[m.rule]
		if run.analysis == "Security" && !rule.security {
			continue
		}
		id := cgsFindingID(sc.name, m)
		if seen[id] {
			continue
		}
		seen[id] = true
		f := &cgsFinding{id: id, path: m.path, status: "Open", rule: m.rule, line: m.line, snippet: m.snippet, created: now, updated: now}
		if old := earlier[id]; old != nil {
			f.created = old.created
		} else {
			day.opened[cgsSeverityIndex(rule.severity)]++
		}
		run.findings = append(run.findings, f)
	}
	ids := make([]string, 0, len(earlier))
	for id := range earlier {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		if seen[id] {
			continue
		}
		old := earlier[id]
		closed := *old
		closed.status, closed.updated = "Closed", now
		run.findings = append(run.findings, &closed)
		sev := cgsSeverityIndex(cgsRules[old.rule].severity)
		day.closed[sev]++
		day.closeDays[sev] += now.Sub(old.created).Hours() / 24
		day.closedN[sev]++
	}
	run.matches = nil
	day.open = st.openBySeverity()
}

// openBySeverity counts the open findings of every scan's last finished run.
func (st *cgsStore) openBySeverity() [5]float64 {
	var out [5]float64
	for _, sc := range st.scans {
		if run := sc.lastDone(); run != nil {
			for _, f := range run.findings {
				if f.status == "Open" {
					out[cgsSeverityIndex(cgsRules[f.rule].severity)]++
				}
			}
		}
	}
	return out
}

// day returns today's metrics bucket, dropping the oldest past cgsMaxDays.
func (st *cgsStore) day(now time.Time) *cgsDay {
	key := now.Format("2006-01-02")
	if d := st.days[key]; d != nil {
		return d
	}
	if len(st.days) >= cgsMaxDays {
		oldest := ""
		for k := range st.days {
			if oldest == "" || k < oldest {
				oldest = k
			}
		}
		delete(st.days, oldest)
	}
	d := &cgsDay{}
	st.days[key] = d
	return d
}

func (s *Server) cgsCreateScan(req cgsRequest) (map[string]any, *cgsErr) {
	in := req.in
	name := c9Str(in, "scanName")
	artifactID := c9Str(c9Map(in, "resourceId"), "codeArtifactId")
	if artifactID == "" {
		return nil, cgsValidation("resourceId.codeArtifactId is required")
	}
	tags := map[string]string{}
	for k, v := range c9Map(in, "tags") {
		tags[k], _ = v.(string)
	}
	if len(tags) > cgsMaxTags {
		return nil, cgsValidation("a scan can have at most %d tags", cgsMaxTags)
	}
	analysis := c9Str(in, "analysisType")
	if analysis == "" {
		analysis = "Security"
	}
	scanType := c9Str(in, "scanType")
	if scanType == "" {
		scanType = "Standard"
	}
	token := c9Str(in, "clientToken")
	now := time.Now().UTC()

	st := s.codeGuruSecurity
	st.mu.Lock()
	defer st.mu.Unlock()
	st.advance(now)
	a := st.artifacts[artifactID]
	if a == nil {
		return nil, cgsNotFound(artifactID, "Code artifact %s doesn't exist.", artifactID)
	}
	if a.scanName != name {
		return nil, cgsValidation("code artifact %s was created for scan %s, not %s", artifactID, a.scanName, name)
	}
	sc := st.scans[name]
	if sc != nil {
		for _, run := range sc.runs {
			if token != "" && run.token == token {
				return sc.runView(run), nil
			}
		}
		if !sc.latest().done {
			return nil, cgsConflict(name, "Scan %s already has a run in progress.", name)
		}
	} else {
		if len(st.scans) >= cgsMaxScans {
			return nil, cgsValidation("the account already has %d scans", cgsMaxScans)
		}
		sc = &cgsScan{name: name, arn: cgsScanArn(name), created: now, tags: tags}
		st.scans[name] = sc
	}
	run := &cgsRun{
		id: uuid.NewString(), state: "InProgress", analysis: analysis, scanType: scanType, artifact: artifactID, token: token,
		created: now, updated: now, due: now.Add(cgsScanDelay), matches: a.matches,
	}
	if !a.uploaded {
		run.errMsg = "No code was uploaded to code artifact " + artifactID + "."
	}
	runs := append(append([]*cgsRun(nil), sc.runs...), run)
	if len(runs) > cgsMaxRuns {
		runs = runs[len(runs)-cgsMaxRuns:]
	}
	sc.runs = runs
	sc.revisions++
	return sc.runView(run), nil
}

func (sc *cgsScan) runView(run *cgsRun) map[string]any {
	return map[string]any{
		"scanName": sc.name, "runId": run.id, "resourceId": map[string]any{"codeArtifactId": run.artifact},
		"scanState": run.state, "scanNameArn": sc.arn,
	}
}

// scan finds a scan after finishing due runs; callers hold st.mu.
func (st *cgsStore) scan(name string) (*cgsScan, *cgsErr) {
	st.advance(time.Now().UTC())
	if sc := st.scans[name]; sc != nil {
		return sc, nil
	}
	return nil, cgsNotFound(name, "Scan %s doesn't exist.", name)
}

func (s *Server) cgsGetScan(req cgsRequest) (map[string]any, *cgsErr) {
	st := s.codeGuruSecurity
	st.mu.Lock()
	defer st.mu.Unlock()
	sc, err := st.scan(c9Str(req.in, "scanName"))
	if err != nil {
		return nil, err
	}
	run := sc.latest()
	if id := c9Str(req.in, "runId"); id != "" {
		run = nil
		for _, r := range sc.runs {
			if r.id == id {
				run = r
			}
		}
		if run == nil {
			return nil, cgsNotFound(id, "Scan %s has no run %s.", sc.name, id)
		}
	}
	out := map[string]any{
		"scanName": sc.name, "runId": run.id, "scanState": run.state, "createdAt": c9Epoch(run.created),
		"updatedAt": c9Epoch(run.updated), "analysisType": run.analysis, "numberOfRevisions": float64(sc.revisions), "scanNameArn": sc.arn,
	}
	if run.errMsg != "" && run.done {
		out["errorMessage"] = run.errMsg
	}
	return out, nil
}

func (s *Server) cgsListScans(req cgsRequest) (map[string]any, *cgsErr) {
	st := s.codeGuruSecurity
	st.mu.Lock()
	defer st.mu.Unlock()
	st.advance(time.Now().UTC())
	names := make([]string, 0, len(st.scans))
	for name := range st.scans {
		names = append(names, name)
	}
	sort.Strings(names)
	start, end, next, err := cgsPage(req.in, len(names), 100)
	if err != nil {
		return nil, err
	}
	out := make([]any, 0, end-start)
	for _, name := range names[start:end] {
		sc := st.scans[name]
		run := sc.latest()
		out = append(out, map[string]any{
			"scanState": run.state, "createdAt": c9Epoch(sc.created), "updatedAt": c9Epoch(run.updated),
			"scanName": sc.name, "runId": run.id, "scanNameArn": sc.arn,
		})
	}
	return map[string]any{"summaries": out, "nextToken": next}, nil
}

func cgsPage(in map[string]any, n, def int) (int, int, string, *cgsErr) {
	start, end, next, err := c9Page(c9Str(in, "nextToken"), n, c9Int(in, "maxResults", def), "ValidationException")
	if err != nil {
		return 0, 0, "", cgsValidation("%s", err.msg)
	}
	return start, end, next, nil
}

// ---- findings ----

func (sc *cgsScan) findingView(f *cgsFinding) map[string]any {
	rule := cgsRules[f.rule]
	snippet := make([]any, 0, len(f.snippet))
	for _, l := range f.snippet {
		snippet = append(snippet, map[string]any{"number": float64(l.n), "content": l.text})
	}
	base := f.path
	for i := len(base) - 1; i >= 0; i-- {
		if base[i] == '/' {
			base = base[i+1:]
			break
		}
	}
	refs := make([]any, 0, len(rule.cwe))
	for _, c := range rule.cwe {
		refs = append(refs, "https://cwe.mitre.org/data/definitions/"+c[len("CWE-"):]+".html")
	}
	findingType := "Software and Configuration Checks/Code Quality"
	if rule.security {
		findingType = "Software and Configuration Checks/Vulnerabilities/" + rule.category
	}
	return map[string]any{
		"createdAt": c9Epoch(f.created), "updatedAt": c9Epoch(f.updated), "description": rule.description,
		"generatorId": "codeguru-security", "id": f.id, "type": findingType, "status": f.status,
		"resource": map[string]any{"id": sc.name, "subResourceId": f.path},
		"vulnerability": map[string]any{
			"referenceUrls": refs, "relatedVulnerabilities": c9AnyStrings(rule.cwe), "id": f.id, "itemCount": float64(1),
			"filePath": map[string]any{"name": base, "path": f.path, "startLine": float64(f.line), "endLine": float64(f.line), "codeSnippet": snippet},
		},
		"severity": rule.severity, "title": rule.title, "detectorTags": c9AnyStrings([]string{rule.category, "codeguru-security"}),
		"detectorId": "codeguru-security/" + rule.detector, "detectorName": rule.name, "ruleId": rule.id,
		"remediation": map[string]any{
			"recommendation": map[string]any{"text": rule.recommendation, "url": rule.url}, "suggestedFixes": []any{},
		},
	}
}

func (s *Server) cgsGetFindings(req cgsRequest) (map[string]any, *cgsErr) {
	st := s.codeGuruSecurity
	st.mu.Lock()
	defer st.mu.Unlock()
	sc, err := st.scan(c9Str(req.in, "scanName"))
	if err != nil {
		return nil, err
	}
	run := sc.lastDone()
	if run == nil {
		return nil, cgsConflict(sc.name, "Scan %s is still in progress.", sc.name)
	}
	status := c9Str(req.in, "status")
	var picked []*cgsFinding
	for _, f := range run.findings {
		if status == "" || status == "All" || f.status == status {
			picked = append(picked, f)
		}
	}
	sort.SliceStable(picked, func(i, j int) bool {
		a, b := cgsSeverityIndex(cgsRules[picked[i].rule].severity), cgsSeverityIndex(cgsRules[picked[j].rule].severity)
		return a < b || a == b && picked[i].id < picked[j].id
	})
	start, end, next, perr := cgsPage(req.in, len(picked), 1000)
	if perr != nil {
		return nil, perr
	}
	out := make([]any, 0, end-start)
	for _, f := range picked[start:end] {
		out = append(out, sc.findingView(f))
	}
	return map[string]any{"findings": out, "nextToken": next}, nil
}

func (s *Server) cgsBatchGetFindings(req cgsRequest) (map[string]any, *cgsErr) {
	st := s.codeGuruSecurity
	st.mu.Lock()
	defer st.mu.Unlock()
	st.advance(time.Now().UTC())
	found, failed := []any{}, []any{}
	seen := map[string]bool{}
	fail := func(scan, id, code, msg string) {
		failed = append(failed, map[string]any{"scanName": scan, "findingId": id, "errorCode": code, "message": msg})
	}
	for _, item := range c9List(req.in, "findingIdentifiers") {
		ident, _ := item.(map[string]any)
		name, id := c9Str(ident, "scanName"), c9Str(ident, "findingId")
		key := name + "\x00" + id
		if seen[key] {
			fail(name, id, "DUPLICATE_IDENTIFIER", "The finding is listed more than once.")
			continue
		}
		seen[key] = true
		sc := st.scans[name]
		if sc == nil {
			fail(name, id, "INVALID_SCAN_NAME", "Scan "+name+" doesn't exist.")
			continue
		}
		var hit *cgsFinding
		if run := sc.lastDone(); run != nil {
			for _, f := range run.findings {
				if f.id == id {
					hit = f
				}
			}
		}
		if hit == nil {
			fail(name, id, "ITEM_DOES_NOT_EXIST", "Finding "+id+" doesn't exist in scan "+name+".")
			continue
		}
		found = append(found, sc.findingView(hit))
	}
	return map[string]any{"findings": found, "failedFindings": failed}, nil
}

// ---- tags ----

// byArn resolves a scan ARN; callers hold st.mu.
func (st *cgsStore) byArn(arn string) (*cgsScan, *cgsErr) {
	for _, sc := range st.scans {
		if sc.arn == arn {
			return sc, nil
		}
	}
	return nil, cgsNotFound(arn, "Resource %s doesn't exist.", arn)
}

func (s *Server) cgsTagResource(req cgsRequest) (map[string]any, *cgsErr) {
	st := s.codeGuruSecurity
	st.mu.Lock()
	defer st.mu.Unlock()
	sc, err := st.byArn(c9Str(req.in, "resourceArn"))
	if err != nil {
		return nil, err
	}
	next := make(map[string]string, len(sc.tags))
	for k, v := range sc.tags {
		next[k] = v
	}
	for k, v := range c9Map(req.in, "tags") {
		next[k], _ = v.(string)
	}
	if len(next) > cgsMaxTags {
		return nil, cgsValidation("a scan can have at most %d tags", cgsMaxTags)
	}
	sc.tags = next
	return map[string]any{}, nil
}

func (s *Server) cgsUntagResource(req cgsRequest) (map[string]any, *cgsErr) {
	st := s.codeGuruSecurity
	st.mu.Lock()
	defer st.mu.Unlock()
	sc, err := st.byArn(c9Str(req.in, "resourceArn"))
	if err != nil {
		return nil, err
	}
	next := make(map[string]string, len(sc.tags))
	for k, v := range sc.tags {
		next[k] = v
	}
	for _, k := range c9Strings(req.in, "tagKeys") {
		delete(next, k)
	}
	sc.tags = next
	return map[string]any{}, nil
}

func (s *Server) cgsListTags(req cgsRequest) (map[string]any, *cgsErr) {
	st := s.codeGuruSecurity
	st.mu.Lock()
	defer st.mu.Unlock()
	sc, err := st.byArn(c9Str(req.in, "resourceArn"))
	if err != nil {
		return nil, err
	}
	tags := make(map[string]any, len(sc.tags))
	for k, v := range sc.tags {
		tags[k] = v
	}
	return map[string]any{"tags": tags}, nil
}

// ---- account configuration ----

func (st *cgsStore) encryption() map[string]any {
	if st.kmsKeyArn == "" {
		return map[string]any{}
	}
	return map[string]any{"kmsKeyArn": st.kmsKeyArn}
}

func (s *Server) cgsGetAccountConfiguration(cgsRequest) (map[string]any, *cgsErr) {
	st := s.codeGuruSecurity
	st.mu.Lock()
	defer st.mu.Unlock()
	return map[string]any{"encryptionConfig": st.encryption()}, nil
}

// cgsUpdateAccountConfiguration sets the KMS key, which must exist in the
// KMS store (looked up before this store's lock is taken). An empty
// encryptionConfig goes back to an AWS-owned key.
func (s *Server) cgsUpdateAccountConfiguration(req cgsRequest) (map[string]any, *cgsErr) {
	arn := c9Str(c9Map(req.in, "encryptionConfig"), "kmsKeyArn")
	if arn != "" {
		key, err := s.kms.DescribeKey(arn)
		if err != nil {
			return nil, cgsNotFound(arn, "KMS key %s doesn't exist.", arn)
		}
		if !key.Enabled {
			return nil, cgsValidation("KMS key %s is not enabled", arn)
		}
	}
	st := s.codeGuruSecurity
	st.mu.Lock()
	defer st.mu.Unlock()
	st.kmsKeyArn = arn
	return map[string]any{"encryptionConfig": st.encryption()}, nil
}

// cgsDayStart parses a yyyy-mm-dd bucket key.
func cgsDayStart(key string) time.Time {
	t, _ := time.Parse("2006-01-02", key)
	return t
}
