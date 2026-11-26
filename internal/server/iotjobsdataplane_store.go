package server

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

// Job executions: one per (job, thing) for every thing a recorded IoT job
// targets by thing ARN (thing-group targets are not expanded: the IoT
// control plane keeps no group membership). An execution starts QUEUED and
// is stored only once a device changes it. Devices move it to IN_PROGRESS,
// SUCCEEDED, FAILED or REJECTED; every change bumps versionNumber and can be
// guarded by expectedVersion. Canceling the job cancels queued executions
// (in-progress ones too with force); an in-progress execution past its step
// or job timeout becomes TIMED_OUT. Deleting or re-creating the job drops
// its executions.

const (
	iotjMaxStoredExecutions = 50000
	iotjMaxCommandExecs     = 10000
	iotjMaxDetails          = 10
	iotjMaxDetailValue      = 1024
	iotjMaxCommandParams    = 50
	iotjMaxStepTimeout      = 10080 // minutes (7 days)
)

var (
	iotjJobIDRe      = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,64}$`)
	iotjCommandArnRe = regexp.MustCompile(`^arn:aws[a-z-]*:iot:[a-z0-9-]+:[0-9]{12}:command/([a-zA-Z0-9_-]{1,64})$`)
	iotjTargetArnRe  = regexp.MustCompile(`^arn:aws[a-z-]*:iot:[a-z0-9-]+:[0-9]{12}:(thing|client)/[^/]+$`)
)

type iotjExec struct {
	jobSeq       int64
	status       string
	details      map[string]string
	queuedAt     time.Time
	startedAt    time.Time
	updatedAt    time.Time
	version      int64
	jobDeadline  time.Time // from the job's inProgressTimeout
	stepDeadline time.Time // from stepTimeoutInMinutes
}

type iotjCommandExec struct {
	id, commandID, target, fingerprint string
	created                            time.Time
}

type iotjStore struct {
	mu       sync.Mutex
	execs    map[string]*iotjExec // thing + "\x00" + jobId
	commands map[string]*iotjCommandExec
	tokens   map[string]string // clientToken -> command execution id
}

func newIoTJobsDataPlaneStore() *iotjStore {
	return &iotjStore{execs: map[string]*iotjExec{}, commands: map[string]*iotjCommandExec{}, tokens: map[string]string{}}
}

var iotjTerminal = map[string]bool{
	"SUCCEEDED": true, "FAILED": true, "REJECTED": true, "TIMED_OUT": true, "REMOVED": true, "CANCELED": true,
}

func iotjKey(thing, job string) string { return thing + "\x00" + job }

// iotjPending is one execution (stored or still implicitly QUEUED) with its job.
type iotjPending struct {
	job  iotJobView
	exec *iotjExec
}

// settleLocked returns the thing's executions in queue order. Stored
// executions of deleted or re-created jobs are dropped; cancellation and
// timeouts are applied. Implicit executions are fresh QUEUED copies.
func (st *iotjStore) settleLocked(thing string, jobs []iotJobView, now time.Time) []iotjPending {
	live := map[string]int64{}
	out := make([]iotjPending, 0, len(jobs))
	for _, job := range jobs {
		live[job.ID] = job.Seq
		e := st.execs[iotjKey(thing, job.ID)]
		if e == nil || e.jobSeq != job.Seq {
			delete(st.execs, iotjKey(thing, job.ID))
			e = &iotjExec{jobSeq: job.Seq, status: "QUEUED", queuedAt: job.Created, updatedAt: job.Created, version: 1}
		}
		iotjApplyJobState(e, job, now)
		out = append(out, iotjPending{job: job, exec: e})
	}
	// ponytail: a scan of every stored execution (at most
	// iotjMaxStoredExecutions); index by thing if that ever shows up.
	prefix := thing + "\x00"
	for key, e := range st.execs {
		if id, ok := strings.CutPrefix(key, prefix); ok {
			if seq, alive := live[id]; !alive || seq != e.jobSeq {
				delete(st.execs, key)
			}
		}
	}
	return out
}

func iotjApplyJobState(e *iotjExec, job iotJobView, now time.Time) {
	switch {
	case job.Status == "CANCELED" && (e.status == "QUEUED" || (e.status == "IN_PROGRESS" && job.Force)):
		e.status, e.updatedAt = "CANCELED", now
		e.version++
	case e.status == "IN_PROGRESS":
		if d := iotjDeadline(e); !d.IsZero() && now.After(d) {
			e.status, e.updatedAt = "TIMED_OUT", d
			e.version++
		}
	}
}

func iotjDeadline(e *iotjExec) time.Time {
	d := e.jobDeadline
	if !e.stepDeadline.IsZero() && (d.IsZero() || e.stepDeadline.Before(d)) {
		d = e.stepDeadline
	}
	return d
}

// storeLocked keeps e for (thing, job), refusing new entries past the cap.
func (st *iotjStore) storeLocked(thing, jobID string, e *iotjExec) *iotjErr {
	key := iotjKey(thing, jobID)
	if _, ok := st.execs[key]; !ok && len(st.execs) >= iotjMaxStoredExecutions {
		return iotjErrorf("ThrottlingException", "at most %d job executions can be tracked", iotjMaxStoredExecutions)
	}
	st.execs[key] = e
	return nil
}

func iotjSeconds(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return float64(t.Unix())
}

func iotjSummary(p iotjPending) map[string]any {
	return map[string]any{
		"jobId": p.job.ID, "queuedAt": iotjSeconds(p.exec.queuedAt), "startedAt": iotjSeconds(p.exec.startedAt),
		"lastUpdatedAt": iotjSeconds(p.exec.updatedAt), "versionNumber": float64(p.exec.version), "executionNumber": float64(1),
	}
}

func iotjDetails(e *iotjExec) map[string]any {
	out := make(map[string]any, len(e.details))
	for k, v := range e.details {
		out[k] = v
	}
	return out
}

func iotjExecution(thing string, p iotjPending, withDoc bool, now time.Time) map[string]any {
	out := iotjSummary(p)
	out["thingName"] = thing
	out["status"] = p.exec.status
	out["statusDetails"] = iotjDetails(p.exec)
	if d := iotjDeadline(p.exec); p.exec.status == "IN_PROGRESS" && !d.IsZero() {
		out["approximateSecondsBeforeTimedOut"] = float64(max(int64(d.Sub(now).Seconds()), 0))
	}
	if withDoc {
		out["jobDocument"] = p.job.Document
	}
	return out
}

// iotjNext picks the in-progress execution, else the oldest queued one.
func iotjNext(all []iotjPending) (iotjPending, bool) {
	for _, status := range []string{"IN_PROGRESS", "QUEUED"} {
		for _, p := range all {
			if p.exec.status == status {
				return p, true
			}
		}
	}
	return iotjPending{}, false
}

// iotjParseDetails reads a validated statusDetails map, bounding its size.
func iotjParseDetails(in map[string]any) (map[string]string, bool, *iotjErr) {
	raw, ok := in["statusDetails"].(map[string]any)
	if !ok {
		return nil, false, nil
	}
	if len(raw) > iotjMaxDetails {
		return nil, true, iotjInvalid("statusDetails can have at most %d entries", iotjMaxDetails)
	}
	out := make(map[string]string, len(raw))
	for k, v := range raw {
		str, _ := v.(string)
		if len(str) > iotjMaxDetailValue {
			return nil, true, iotjInvalid("statusDetails values can be at most %d characters", iotjMaxDetailValue)
		}
		out[k] = str
	}
	return out, true, nil
}

// iotjStepTimeout reads stepTimeoutInMinutes (0 when absent).
func iotjStepTimeout(in map[string]any) (int64, *iotjErr) {
	n, ok := in["stepTimeoutInMinutes"].(float64)
	if !ok {
		return 0, nil
	}
	if n < 1 || n > iotjMaxStepTimeout {
		return 0, iotjInvalid("stepTimeoutInMinutes must be between 1 and %d", iotjMaxStepTimeout)
	}
	return int64(n), nil
}

// start moves e to IN_PROGRESS.
func iotjStart(e *iotjExec, job iotJobView, now time.Time) {
	e.status, e.startedAt = "IN_PROGRESS", now
	if job.InProgressTimeoutMinutes > 0 {
		e.jobDeadline = now.Add(time.Duration(job.InProgressTimeoutMinutes) * time.Minute)
	}
}

func iotjSetStep(e *iotjExec, step int64, now time.Time) {
	if step > 0 {
		e.stepDeadline = now.Add(time.Duration(step) * time.Minute)
	}
}

func iotjThing(in map[string]any) string {
	t, _ := in["thingName"].(string)
	return t
}

func (s *Server) iotjGetPending(in map[string]any) (map[string]any, *iotjErr) {
	thing := iotjThing(in)
	jobs := s.iot.jobsForThing(thing) // iot lock taken and released before ours
	st := s.iotJobsData
	st.mu.Lock()
	defer st.mu.Unlock()
	inProgress, queued := []any{}, []any{}
	for _, p := range st.settleLocked(thing, jobs, time.Now().UTC()) {
		switch p.exec.status {
		case "IN_PROGRESS":
			inProgress = append(inProgress, iotjSummary(p))
		case "QUEUED":
			queued = append(queued, iotjSummary(p))
		}
	}
	return map[string]any{"inProgressJobs": inProgress, "queuedJobs": queued}, nil
}

func (s *Server) iotjStartNext(in map[string]any) (map[string]any, *iotjErr) {
	thing := iotjThing(in)
	details, hasDetails, err := iotjParseDetails(in)
	if err != nil {
		return nil, err
	}
	step, err := iotjStepTimeout(in)
	if err != nil {
		return nil, err
	}
	jobs := s.iot.jobsForThing(thing)
	st := s.iotJobsData
	st.mu.Lock()
	defer st.mu.Unlock()
	now := time.Now().UTC()
	p, ok := iotjNext(st.settleLocked(thing, jobs, now))
	if !ok {
		return map[string]any{}, nil
	}
	if p.exec.status == "QUEUED" {
		next := *p.exec
		iotjStart(&next, p.job, now)
		iotjSetStep(&next, step, now)
		if hasDetails {
			next.details = details
		}
		next.updatedAt = now
		next.version++
		if err := st.storeLocked(thing, p.job.ID, &next); err != nil {
			return nil, err
		}
		p.exec = &next
	}
	return map[string]any{"execution": iotjExecution(thing, p, true, now)}, nil
}

func (s *Server) iotjFind(all []iotjPending, jobID string, in map[string]any) (iotjPending, *iotjErr) {
	var p iotjPending
	found := false
	if jobID == "$next" {
		p, found = iotjNext(all)
	} else {
		for _, c := range all {
			if c.job.ID == jobID {
				p, found = c, true
			}
		}
	}
	if !found {
		return p, iotjErrorf("ResourceNotFoundException", "no job execution %s for thing %s", jobID, iotjThing(in))
	}
	if n, ok := in["executionNumber"].(float64); ok && n != 1 {
		return p, iotjErrorf("ResourceNotFoundException", "job execution number %v not found", n)
	}
	return p, nil
}

func (s *Server) iotjDescribe(in map[string]any) (map[string]any, *iotjErr) {
	thing := iotjThing(in)
	jobID, _ := in["jobId"].(string)
	jobs := s.iot.jobsForThing(thing)
	st := s.iotJobsData
	st.mu.Lock()
	defer st.mu.Unlock()
	now := time.Now().UTC()
	p, err := s.iotjFind(st.settleLocked(thing, jobs, now), jobID, in)
	if err != nil {
		return nil, err
	}
	withDoc, _ := in["includeJobDocument"].(bool)
	return map[string]any{"execution": iotjExecution(thing, p, withDoc, now)}, nil
}

var iotjDeviceStatuses = map[string]bool{"IN_PROGRESS": true, "SUCCEEDED": true, "FAILED": true, "REJECTED": true}

func (s *Server) iotjUpdate(in map[string]any) (map[string]any, *iotjErr) {
	thing := iotjThing(in)
	jobID, _ := in["jobId"].(string)
	status, _ := in["status"].(string)
	if !iotjDeviceStatuses[status] {
		return nil, iotjInvalid("status must be IN_PROGRESS, SUCCEEDED, FAILED or REJECTED")
	}
	details, hasDetails, err := iotjParseDetails(in)
	if err != nil {
		return nil, err
	}
	step, err := iotjStepTimeout(in)
	if err != nil {
		return nil, err
	}
	jobs := s.iot.jobsForThing(thing)
	st := s.iotJobsData
	st.mu.Lock()
	defer st.mu.Unlock()
	now := time.Now().UTC()
	p, err := s.iotjFind(st.settleLocked(thing, jobs, now), jobID, in)
	if err != nil {
		return nil, err
	}
	if iotjTerminal[p.exec.status] {
		return nil, iotjErrorf("InvalidStateTransitionException", "job execution %s is already %s", jobID, p.exec.status)
	}
	if v, ok := in["expectedVersion"].(float64); ok && int64(v) != p.exec.version {
		return nil, iotjErrorf("InvalidStateTransitionException", "version mismatch: expected %d, current %d", int64(v), p.exec.version)
	}
	next := *p.exec
	if next.status == "QUEUED" && status == "IN_PROGRESS" {
		iotjStart(&next, p.job, now)
	}
	next.status = status
	if status == "IN_PROGRESS" {
		iotjSetStep(&next, step, now)
	}
	if hasDetails {
		next.details = details
	}
	next.updatedAt = now
	next.version++
	if err := st.storeLocked(thing, p.job.ID, &next); err != nil {
		return nil, err
	}
	out := map[string]any{}
	if b, _ := in["includeJobExecutionState"].(bool); b {
		out["executionState"] = map[string]any{"status": next.status, "statusDetails": iotjDetails(&next), "versionNumber": float64(next.version)}
	}
	if b, _ := in["includeJobDocument"].(bool); b {
		out["jobDocument"] = p.job.Document
	}
	return out, nil
}

func (s *Server) iotjStartCommandExecution(in map[string]any) (map[string]any, *iotjErr) {
	commandArn, _ := in["commandArn"].(string)
	target, _ := in["targetArn"].(string)
	m := iotjCommandArnRe.FindStringSubmatch(commandArn)
	if m == nil {
		return nil, iotjErrorf("ValidationException", "commandArn must be an IoT command ARN")
	}
	if !iotjTargetArnRe.MatchString(target) {
		return nil, iotjErrorf("ValidationException", "targetArn must be an IoT thing or client ARN")
	}
	if params, ok := in["parameters"].(map[string]any); ok {
		if len(params) > iotjMaxCommandParams {
			return nil, iotjErrorf("ValidationException", "at most %d parameters are allowed", iotjMaxCommandParams)
		}
		for name, v := range params {
			if obj, _ := v.(map[string]any); len(obj) != 1 {
				return nil, iotjErrorf("ValidationException", "parameter %s must set exactly one value type", name)
			}
		}
	}
	if !s.iot.commandExists(m[1]) { // iot lock taken and released before ours
		return nil, iotjErrorf("ResourceNotFoundException", "command %s not found", m[1])
	}
	fp, _ := json.Marshal([]any{commandArn, target, in["parameters"], in["executionTimeoutSeconds"]})
	st := s.iotJobsData
	st.mu.Lock()
	defer st.mu.Unlock()
	token, _ := in["clientToken"].(string)
	if id, ok := st.tokens[token]; ok && token != "" {
		if prev := st.commands[id]; prev != nil {
			if prev.fingerprint != string(fp) {
				return nil, iotjErrorf("ConflictException", "clientToken %s was used with different parameters", token)
			}
			return map[string]any{"executionId": id}, nil
		}
	}
	if len(st.commands) >= iotjMaxCommandExecs {
		return nil, iotjErrorf("ServiceQuotaExceededException", "at most %d command executions can exist", iotjMaxCommandExecs)
	}
	buf := make([]byte, 16)
	_, _ = rand.Read(buf)
	id := hex.EncodeToString(buf)
	st.commands[id] = &iotjCommandExec{id: id, commandID: m[1], target: target, fingerprint: string(fp), created: time.Now().UTC()}
	if token != "" {
		st.tokens[token] = id
	}
	return map[string]any{"executionId": id}, nil
}

// storedKeys lists the stored executions (thing NUL jobId), sorted.
func (st *iotjStore) storedKeys() []string {
	st.mu.Lock()
	defer st.mu.Unlock()
	out := make([]string, 0, len(st.execs))
	for k := range st.execs {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
