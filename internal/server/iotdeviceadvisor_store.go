package server

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	idaAccount  = "123456789012"
	idaRegion   = "us-east-1"
	idaARNBase  = "arn:aws:iotdeviceadvisor:" + idaRegion + ":" + idaAccount + ":"
	idaEndpoint = "a1b2c3d4e5f6g7-ats.deviceadvisor.iot." + idaRegion + ".amazonaws.com"
	// ponytail: fixed caps keep stored counts and bytes bounded; AWS's own
	// quotas are in the same range.
	idaMaxSuites       = 200
	idaMaxVersions     = 20 // kept per suite; older versions are dropped
	idaMaxRuns         = 1000
	idaMaxTags         = 50
	idaMaxGroups       = 50
	idaMaxTestsPerRun  = 500
	idaDefaultPageSize = 25
)

type idaVersion struct {
	name   string // v1, v2, ...
	config map[string]any
	at     time.Time
}

type idaSuite struct {
	id, arn, clientToken string
	versions             []idaVersion // oldest first
	next                 int          // number of the next version
	tags                 map[string]string
	created, modified    time.Time
}

func (s *idaSuite) latest() idaVersion { return s.versions[len(s.versions)-1] }

func (s *idaSuite) version(name string) (idaVersion, bool) {
	if name == "" {
		return s.latest(), true
	}
	for _, v := range s.versions {
		if v.name == name {
			return v, true
		}
	}
	return idaVersion{}, false
}

type idaTest struct{ id, name string }

type idaGroup struct {
	id, name string
	tests    []idaTest
}

type idaRun struct {
	id, arn, suiteID, version, suiteName string
	qualification                        bool
	config                               map[string]any
	groups                               []idaGroup
	status                               string
	tags                                 map[string]string
	created, started, ended              time.Time
}

type idaStore struct {
	mu         sync.Mutex
	suites     map[string]*idaSuite
	suiteOrder []string // oldest first
	runs       map[string]*idaRun
	runOrder   []string // oldest first
}

func newIoTDeviceAdvisorStore() *idaStore {
	return &idaStore{suites: map[string]*idaSuite{}, runs: map[string]*idaRun{}}
}

func idaNewID() string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b) // 12 characters, the model's minimum
}

func idaStr(m map[string]any, k string) string { v, _ := m[k].(string); return v }

func idaMap(m map[string]any, k string) map[string]any { v, _ := m[k].(map[string]any); return v }

func idaBool(m map[string]any, k string) bool { v, _ := m[k].(bool); return v }

func idaTags(m map[string]any) map[string]string {
	out := map[string]string{}
	for k, v := range idaMap(m, "tags") {
		out[k], _ = v.(string)
	}
	return out
}

func idaTagsOut(tags map[string]string) map[string]any {
	out := make(map[string]any, len(tags))
	for k, v := range tags {
		out[k] = v
	}
	return out
}

func idaCheckTags(tags map[string]string) *idaErr {
	if len(tags) > idaMaxTags {
		return idaInvalid("a resource can have at most %d tags", idaMaxTags)
	}
	for k := range tags {
		if strings.HasPrefix(strings.ToLower(k), "aws:") {
			return idaInvalid("tag keys starting with aws: are reserved")
		}
	}
	return nil
}

func (st *idaStore) dispatch(op string, in map[string]any) (map[string]any, *idaErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	now := time.Now().UTC()
	switch op {
	case "CreateSuiteDefinition":
		return st.createSuite(in, now)
	case "UpdateSuiteDefinition":
		return st.updateSuite(in, now)
	case "GetSuiteDefinition":
		return st.getSuite(in)
	case "ListSuiteDefinitions":
		return st.listSuites(in)
	case "DeleteSuiteDefinition":
		return st.deleteSuite(in)
	case "StartSuiteRun":
		return st.startRun(in, now)
	case "GetSuiteRun":
		return st.getRun(in, now)
	case "StopSuiteRun":
		return st.stopRun(in, now)
	case "ListSuiteRuns":
		return st.listRuns(in, now)
	case "GetSuiteRunReport":
		return st.runReport(in, now)
	case "GetEndpoint":
		if idaStr(in, "authenticationMethod") == "SignatureVersion4" && idaStr(in, "deviceRoleArn") == "" {
			return nil, idaInvalid("deviceRoleArn is required for SignatureVersion4 authentication")
		}
		return map[string]any{"endpoint": idaEndpoint}, nil
	case "TagResource", "UntagResource", "ListTagsForResource":
		return st.tagging(op, in)
	}
	return nil, idaErrorf("UnknownOperationException", "unknown operation %s", op)
}

// ---- suite definitions ----

// idaCheckConfig validates a suiteDefinitionConfiguration beyond the model.
func idaCheckConfig(cfg map[string]any) *idaErr {
	root := idaStr(cfg, "rootGroup")
	if root == "" {
		if !idaBool(cfg, "intendedForQualification") {
			return idaInvalid("rootGroup is required unless the suite is intended for qualification")
		}
		return nil
	}
	_, err := idaParseGroups(root)
	return err
}

// idaParseGroups reads the test groups of a rootGroup document:
// {"tests":[{"name":"MQTT Connect","tests":[{"name":"MQTT_Connect","test":{"id":"MQTT_Connect"}}]}]}.
// An empty root group (qualification suites) gets the qualification tests.
func idaParseGroups(root string) ([]idaGroup, *idaErr) {
	if root == "" {
		return []idaGroup{{id: idaStableID("group:Qualification"), name: "Qualification", tests: []idaTest{
			{id: idaStableID("test:TLS_Connect"), name: "TLS Connect"},
			{id: idaStableID("test:MQTT_Connect"), name: "MQTT Connect"},
			{id: idaStableID("test:MQTT_Publish"), name: "MQTT Publish"},
		}}}, nil
	}
	var doc struct {
		Tests []struct {
			Name  string `json:"name"`
			Tests []struct {
				Name string `json:"name"`
				Test struct {
					ID string `json:"id"`
				} `json:"test"`
			} `json:"tests"`
		} `json:"tests"`
	}
	if err := json.Unmarshal([]byte(root), &doc); err != nil {
		return nil, idaInvalid("rootGroup is not a valid test group document")
	}
	if len(doc.Tests) == 0 || len(doc.Tests) > idaMaxGroups {
		return nil, idaInvalid("rootGroup must contain between 1 and %d test groups", idaMaxGroups)
	}
	groups := make([]idaGroup, 0, len(doc.Tests))
	total := 0
	for _, g := range doc.Tests {
		if g.Name == "" || len(g.Tests) == 0 {
			return nil, idaInvalid("each test group needs a name and at least one test")
		}
		group := idaGroup{id: idaStableID("group:" + g.Name), name: g.Name}
		for _, t := range g.Tests {
			id := t.Test.ID
			if id == "" {
				id = t.Name
			}
			if id == "" {
				return nil, idaInvalid("each test needs a name or a test id")
			}
			name := t.Name
			if name == "" {
				name = id
			}
			group.tests = append(group.tests, idaTest{id: idaStableID("test:" + id), name: name})
		}
		total += len(group.tests)
		groups = append(groups, group)
	}
	if total > idaMaxTestsPerRun {
		return nil, idaInvalid("rootGroup can hold at most %d tests", idaMaxTestsPerRun)
	}
	return groups, nil
}

// idaStableID gives a test or group a stable 16-character id.
func idaStableID(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:8])
}

func (st *idaStore) suite(id string) (*idaSuite, *idaErr) {
	s := st.suites[id]
	if s == nil {
		return nil, idaErrorf("ResourceNotFoundException", "Suite definition %s not found", id)
	}
	return s, nil
}

func (st *idaStore) createSuite(in map[string]any, now time.Time) (map[string]any, *idaErr) {
	cfg := idaMap(in, "suiteDefinitionConfiguration")
	if err := idaCheckConfig(cfg); err != nil {
		return nil, err
	}
	tags := idaTags(in)
	if err := idaCheckTags(tags); err != nil {
		return nil, err
	}
	token := idaStr(in, "clientToken")
	if token != "" {
		for _, s := range st.suites {
			if s.clientToken == token {
				return idaSuiteSummary(s), nil
			}
		}
	}
	if len(st.suites) >= idaMaxSuites {
		return nil, idaInvalid("at most %d suite definitions can exist", idaMaxSuites)
	}
	id := idaNewID()
	s := &idaSuite{
		id: id, arn: idaARNBase + "suitedefinition/" + id, clientToken: token, next: 2, tags: tags,
		versions: []idaVersion{{name: "v1", config: budCloneMap(cfg), at: now}}, created: now, modified: now,
	}
	st.suites[id] = s
	st.suiteOrder = append(st.suiteOrder, id)
	return idaSuiteSummary(s), nil
}

func idaSuiteSummary(s *idaSuite) map[string]any {
	return map[string]any{"suiteDefinitionId": s.id, "suiteDefinitionArn": s.arn,
		"suiteDefinitionName": idaStr(s.latest().config, "suiteDefinitionName"), "createdAt": budEpoch(s.created)}
}

func (st *idaStore) updateSuite(in map[string]any, now time.Time) (map[string]any, *idaErr) {
	s, err := st.suite(idaStr(in, "suiteDefinitionId"))
	if err != nil {
		return nil, err
	}
	cfg := idaMap(in, "suiteDefinitionConfiguration")
	if err := idaCheckConfig(cfg); err != nil {
		return nil, err
	}
	v := idaVersion{name: "v" + strconv.Itoa(s.next), config: budCloneMap(cfg), at: now}
	// Copy-on-write so earlier snapshots of the slice never change.
	versions := append(append([]idaVersion{}, s.versions...), v)
	if len(versions) > idaMaxVersions {
		versions = versions[len(versions)-idaMaxVersions:]
	}
	s.versions, s.next, s.modified = versions, s.next+1, now
	return map[string]any{"suiteDefinitionId": s.id, "suiteDefinitionArn": s.arn,
		"suiteDefinitionName": idaStr(cfg, "suiteDefinitionName"), "suiteDefinitionVersion": v.name,
		"createdAt": budEpoch(s.created), "lastUpdatedAt": budEpoch(now)}, nil
}

func (st *idaStore) getSuite(in map[string]any) (map[string]any, *idaErr) {
	s, err := st.suite(idaStr(in, "suiteDefinitionId"))
	if err != nil {
		return nil, err
	}
	v, ok := s.version(idaStr(in, "suiteDefinitionVersion"))
	if !ok {
		return nil, idaErrorf("ResourceNotFoundException", "Suite definition %s has no version %s", s.id, idaStr(in, "suiteDefinitionVersion"))
	}
	return map[string]any{
		"suiteDefinitionId": s.id, "suiteDefinitionArn": s.arn, "suiteDefinitionVersion": v.name,
		"latestVersion": s.latest().name, "suiteDefinitionConfiguration": budCloneMap(v.config),
		"createdAt": budEpoch(s.created), "lastModifiedAt": budEpoch(s.modified), "tags": idaTagsOut(s.tags),
	}, nil
}

func idaPageSize(in map[string]any) int {
	if n, ok := in["maxResults"].(float64); ok && n > 0 {
		return int(n)
	}
	return idaDefaultPageSize
}

func idaPage(in map[string]any, total int) (int, int, string, *idaErr) {
	start, end, next, err := budPage(idaStr(in, "nextToken"), total, idaPageSize(in), "ValidationException")
	if err != nil {
		return 0, 0, "", idaInvalid("%s", err.msg)
	}
	return start, end, next, nil
}

func (st *idaStore) listSuites(in map[string]any) (map[string]any, *idaErr) {
	start, end, next, err := idaPage(in, len(st.suiteOrder))
	if err != nil {
		return nil, err
	}
	list := make([]any, 0, end-start)
	for i := start; i < end; i++ {
		s := st.suites[st.suiteOrder[len(st.suiteOrder)-1-i]] // newest first
		cfg := s.latest().config
		item := map[string]any{
			"suiteDefinitionId": s.id, "suiteDefinitionName": idaStr(cfg, "suiteDefinitionName"),
			"intendedForQualification": idaBool(cfg, "intendedForQualification"),
			"isLongDurationTest":       idaBool(cfg, "isLongDurationTest"), "createdAt": budEpoch(s.created),
		}
		if d, ok := cfg["devices"]; ok {
			item["defaultDevices"] = budClone(d)
		}
		if p := idaStr(cfg, "protocol"); p != "" {
			item["protocol"] = p
		}
		list = append(list, item)
	}
	out := map[string]any{"suiteDefinitionInformationList": list}
	if next != "" {
		out["nextToken"] = next
	}
	return out, nil
}

// deleteSuite removes a definition and its runs.
func (st *idaStore) deleteSuite(in map[string]any) (map[string]any, *idaErr) {
	s, err := st.suite(idaStr(in, "suiteDefinitionId"))
	if err != nil {
		return nil, err
	}
	delete(st.suites, s.id)
	st.suiteOrder = idaWithout(st.suiteOrder, func(id string) bool { return id == s.id })
	st.runOrder = idaWithout(st.runOrder, func(id string) bool {
		if st.runs[id].suiteID == s.id {
			delete(st.runs, id)
			return true
		}
		return false
	})
	return map[string]any{}, nil
}

// idaWithout returns a new slice holding the ids drop rejects.
func idaWithout(ids []string, drop func(string) bool) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if !drop(id) {
			out = append(out, id)
		}
	}
	return out
}

// ---- suite runs ----

func idaActive(status string) bool {
	return status == "PENDING" || status == "RUNNING" || status == "STOPPING"
}

func (st *idaStore) startRun(in map[string]any, now time.Time) (map[string]any, *idaErr) {
	s, err := st.suite(idaStr(in, "suiteDefinitionId"))
	if err != nil {
		return nil, err
	}
	v, ok := s.version(idaStr(in, "suiteDefinitionVersion"))
	if !ok {
		return nil, idaErrorf("ResourceNotFoundException", "Suite definition %s has no version %s", s.id, idaStr(in, "suiteDefinitionVersion"))
	}
	cfg := idaMap(in, "suiteRunConfiguration")
	dev := idaMap(cfg, "primaryDevice")
	if idaStr(dev, "thingArn") == "" && idaStr(dev, "certificateArn") == "" {
		return nil, idaInvalid("primaryDevice needs a thingArn or a certificateArn")
	}
	tags := idaTags(in)
	if err := idaCheckTags(tags); err != nil {
		return nil, err
	}
	groups, err := idaParseGroups(idaStr(v.config, "rootGroup"))
	if err != nil {
		return nil, err
	}
	if sel := budStrings(cfg, "selectedTestList"); len(sel) > 0 {
		if groups = idaSelect(groups, sel); len(groups) == 0 {
			return nil, idaInvalid("selectedTestList matches no test in the suite definition")
		}
	}
	if !idaBool(cfg, "parallelRun") {
		for _, r := range st.runs {
			if r.suiteID == s.id && idaActive(r.status) {
				return nil, idaErrorf("ConflictException", "Suite run %s is still in progress for suite definition %s", r.id, s.id)
			}
		}
	}
	if len(st.runs) >= idaMaxRuns {
		return nil, idaInvalid("at most %d suite runs can be stored; delete old suite definitions first", idaMaxRuns)
	}
	id := idaNewID()
	r := &idaRun{
		id: id, arn: idaARNBase + "suiterun/" + s.id + "/" + id, suiteID: s.id, version: v.name,
		suiteName: idaStr(v.config, "suiteDefinitionName"), qualification: idaBool(v.config, "intendedForQualification"),
		config: budCloneMap(cfg), groups: groups, status: "PENDING", tags: tags, created: now,
	}
	st.runs[id] = r
	st.runOrder = append(st.runOrder, id)
	return map[string]any{"suiteRunId": id, "suiteRunArn": r.arn, "createdAt": budEpoch(now), "endpoint": idaEndpoint}, nil
}

// idaSelect keeps only the selected tests (by test case definition id).
func idaSelect(groups []idaGroup, selected []string) []idaGroup {
	want := map[string]bool{}
	for _, id := range selected {
		want[id] = true
	}
	var out []idaGroup
	for _, g := range groups {
		var tests []idaTest
		for _, t := range g.tests {
			if want[t.id] {
				tests = append(tests, t)
			}
		}
		if len(tests) > 0 {
			out = append(out, idaGroup{id: g.id, name: g.name, tests: tests})
		}
	}
	return out
}

// settle advances a run one step per read: PENDING -> RUNNING -> PASS, and
// STOPPING -> STOPPED.
func (r *idaRun) settle(now time.Time) {
	switch r.status {
	case "PENDING":
		r.status, r.started = "RUNNING", now
	case "RUNNING":
		r.status, r.ended = "PASS", now
	case "STOPPING":
		r.status, r.ended = "STOPPED", now
	}
}

func (st *idaStore) run(in map[string]any) (*idaRun, *idaErr) {
	r := st.runs[idaStr(in, "suiteRunId")]
	if r == nil || r.suiteID != idaStr(in, "suiteDefinitionId") {
		return nil, idaErrorf("ResourceNotFoundException", "Suite run %s not found for suite definition %s", idaStr(in, "suiteRunId"), idaStr(in, "suiteDefinitionId"))
	}
	return r, nil
}

func (st *idaStore) getRun(in map[string]any, now time.Time) (map[string]any, *idaErr) {
	r, err := st.run(in)
	if err != nil {
		return nil, err
	}
	r.settle(now)
	groups := make([]any, 0, len(r.groups))
	for _, g := range r.groups {
		tests := make([]any, 0, len(g.tests))
		for _, t := range g.tests {
			tc := map[string]any{
				"testCaseRunId": idaStableID(r.id + ":" + t.id), "testCaseDefinitionId": t.id,
				"testCaseDefinitionName": t.name, "status": r.status,
			}
			if !r.started.IsZero() {
				tc["startTime"] = budEpoch(r.started)
			}
			if !r.ended.IsZero() {
				tc["endTime"] = budEpoch(r.ended)
			}
			tests = append(tests, tc)
		}
		groups = append(groups, map[string]any{"groupId": g.id, "groupName": g.name, "tests": tests})
	}
	out := map[string]any{
		"suiteDefinitionId": r.suiteID, "suiteDefinitionVersion": r.version, "suiteRunId": r.id, "suiteRunArn": r.arn,
		"suiteRunConfiguration": budCloneMap(r.config), "testResult": map[string]any{"groups": groups},
		"status": r.status, "tags": idaTagsOut(r.tags),
	}
	if !r.started.IsZero() {
		out["startTime"] = budEpoch(r.started)
	}
	if !r.ended.IsZero() {
		out["endTime"] = budEpoch(r.ended)
	}
	return out, nil
}

func (st *idaStore) stopRun(in map[string]any, now time.Time) (map[string]any, *idaErr) {
	r, err := st.run(in)
	if err != nil {
		return nil, err
	}
	switch r.status {
	case "PENDING", "RUNNING":
		r.status = "STOPPING"
	case "STOPPING":
	default:
		return nil, idaInvalid("Suite run %s already finished with status %s", r.id, r.status)
	}
	return map[string]any{}, nil
}

func (st *idaStore) listRuns(in map[string]any, now time.Time) (map[string]any, *idaErr) {
	suiteID, version := idaStr(in, "suiteDefinitionId"), idaStr(in, "suiteDefinitionVersion")
	if version != "" && suiteID == "" {
		return nil, idaInvalid("suiteDefinitionVersion requires suiteDefinitionId")
	}
	var match []*idaRun
	for i := len(st.runOrder) - 1; i >= 0; i-- { // newest first
		r := st.runs[st.runOrder[i]]
		if (suiteID == "" || r.suiteID == suiteID) && (version == "" || r.version == version) {
			match = append(match, r)
		}
	}
	start, end, next, err := idaPage(in, len(match))
	if err != nil {
		return nil, err
	}
	list := make([]any, 0, end-start)
	for _, r := range match[start:end] {
		r.settle(now)
		passed, failed := 0, 0
		for _, g := range r.groups {
			switch r.status {
			case "PASS":
				passed += len(g.tests)
			case "FAIL":
				failed += len(g.tests)
			}
		}
		item := map[string]any{
			"suiteDefinitionId": r.suiteID, "suiteDefinitionVersion": r.version, "suiteDefinitionName": r.suiteName,
			"suiteRunId": r.id, "createdAt": budEpoch(r.created), "status": r.status, "passed": passed, "failed": failed,
		}
		if !r.started.IsZero() {
			item["startedAt"] = budEpoch(r.started)
		}
		if !r.ended.IsZero() {
			item["endAt"] = budEpoch(r.ended)
		}
		list = append(list, item)
	}
	out := map[string]any{"suiteRunsList": list}
	if next != "" {
		out["nextToken"] = next
	}
	return out, nil
}

// runReport links the qualification report of a passed qualification run.
func (st *idaStore) runReport(in map[string]any, now time.Time) (map[string]any, *idaErr) {
	r, err := st.run(in)
	if err != nil {
		return nil, err
	}
	if !r.qualification {
		return nil, idaInvalid("Suite run %s is not a qualification run", r.id)
	}
	if r.status != "PASS" && r.status != "PASS_WITH_WARNINGS" {
		return nil, idaInvalid("Suite run %s has status %s; a report exists only for passed runs", r.id, r.status)
	}
	return map[string]any{"qualificationReportDownloadUrl": "https://deviceadvisor-reports." + idaRegion + ".amazonaws.com/" +
		idaAccount + "/" + r.suiteID + "/" + r.id + "/qualification-report.pdf"}, nil
}

// ---- tags ----

func (st *idaStore) tagging(op string, in map[string]any) (map[string]any, *idaErr) {
	arn := idaStr(in, "resourceArn")
	var tags *map[string]string
	rest, _ := strings.CutPrefix(arn, idaARNBase)
	if id, ok := strings.CutPrefix(rest, "suitedefinition/"); ok && st.suites[id] != nil {
		tags = &st.suites[id].tags
	} else if ids, ok := strings.CutPrefix(rest, "suiterun/"); ok {
		suiteID, runID, _ := strings.Cut(ids, "/")
		if r := st.runs[runID]; r != nil && r.suiteID == suiteID {
			tags = &r.tags
		}
	}
	if tags == nil {
		return nil, idaErrorf("ResourceNotFoundException", "Resource %s not found", arn)
	}
	switch op {
	case "TagResource":
		merged := map[string]string{}
		for k, v := range *tags {
			merged[k] = v
		}
		for k, v := range idaTags(in) {
			merged[k] = v
		}
		if err := idaCheckTags(merged); err != nil {
			return nil, err
		}
		*tags = merged
	case "UntagResource":
		*tags = budDropTags(*tags, budStrings(in, "tagKeys"))
	default:
		return map[string]any{"tags": idaTagsOut(*tags)}, nil
	}
	return map[string]any{}, nil
}
