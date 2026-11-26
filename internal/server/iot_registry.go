package server

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"
)

// The IoT control plane's responses are generic, but jobs and commands it
// creates are recorded here so the IoT Jobs data plane
// (iotjobsdataplane*.go) can run their executions: CreateJob records the
// targets and document, CancelJob marks the job canceled, DeleteJob removes
// it; CreateCommand and DeleteCommand do the same for commands.

const (
	iotMaxRegisteredJobs     = 10000
	iotMaxRegisteredCommands = 10000
	iotMaxJobTargets         = 100
	iotMaxJobDocument        = 32768
)

// iotJobView is a copy of a recorded job, safe to use without the iot lock.
type iotJobView struct {
	ID       string
	Seq      int64 // distinguishes a re-created job from its predecessor
	Targets  []string
	Document string
	Status   string // IN_PROGRESS or CANCELED
	Force    bool   // canceled with force: in-progress executions cancel too
	Created  time.Time
	// InProgressTimeoutMinutes comes from timeoutConfig (0 = none).
	InProgressTimeoutMinutes int64
}

type iotRegistry struct {
	jobs     map[string]*iotJobView
	commands map[string]time.Time
	seq      int64
}

// record applies CreateJob, CancelJob, DeleteJob, CreateCommand and
// DeleteCommand to the registry. It returns an error code and message when
// the request must be refused.
func (s *iotStore) record(action string, payload map[string]any, pathParams map[string]string, query url.Values) (string, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	reg := &s.registry
	if reg.jobs == nil {
		reg.jobs, reg.commands = map[string]*iotJobView{}, map[string]time.Time{}
	}
	now := time.Now().UTC()
	switch action {
	case "CreateJob":
		id := pathParams["jobId"]
		if id == "" {
			return "", ""
		}
		// The IoT control plane answers every CreateJob with success, so a
		// repeated id re-creates the job (new Seq: its executions restart).
		if _, exists := reg.jobs[id]; !exists && len(reg.jobs) >= iotMaxRegisteredJobs {
			return "LimitExceededException", fmt.Sprintf("at most %d jobs can exist", iotMaxRegisteredJobs)
		}
		rawTargets, _ := payload["targets"].([]any)
		if len(rawTargets) > iotMaxJobTargets {
			return "InvalidRequestException", fmt.Sprintf("a job can have at most %d targets", iotMaxJobTargets)
		}
		targets := make([]string, 0, len(rawTargets))
		for _, t := range rawTargets {
			if str, ok := t.(string); ok && str != "" {
				targets = append(targets, str)
			}
		}
		doc, _ := payload["document"].(string)
		if len(doc) > iotMaxJobDocument {
			return "InvalidRequestException", fmt.Sprintf("the job document can be at most %d bytes", iotMaxJobDocument)
		}
		var timeout int64
		if tc, ok := payload["timeoutConfig"].(map[string]any); ok {
			if n, ok := tc["inProgressTimeoutInMinutes"].(float64); ok && n >= 1 && n <= 10080 {
				timeout = int64(n)
			}
		}
		reg.seq++
		reg.jobs[id] = &iotJobView{ID: id, Seq: reg.seq, Targets: targets, Document: doc, Status: "IN_PROGRESS",
			Created: now, InProgressTimeoutMinutes: timeout}
	case "CancelJob":
		if job, ok := reg.jobs[pathParams["jobId"]]; ok {
			job.Status = "CANCELED"
			job.Force = job.Force || strings.EqualFold(query.Get("force"), "true")
		}
	case "DeleteJob":
		delete(reg.jobs, pathParams["jobId"])
	case "CreateCommand":
		id := pathParams["commandId"]
		if id == "" {
			return "", ""
		}
		if _, exists := reg.commands[id]; !exists && len(reg.commands) >= iotMaxRegisteredCommands {
			return "ServiceQuotaExceededException", fmt.Sprintf("at most %d commands can exist", iotMaxRegisteredCommands)
		}
		reg.commands[id] = now
	case "DeleteCommand":
		delete(reg.commands, pathParams["commandId"])
	}
	return "", ""
}

func (j *iotJobView) clone() iotJobView {
	out := *j
	out.Targets = append([]string(nil), j.Targets...)
	return out
}

// iotJobTargetsThing reports whether a job targets thingName directly.
func iotJobTargetsThing(job iotJobView, thingName string) bool {
	for _, t := range job.Targets {
		if strings.HasSuffix(t, ":thing/"+thingName) {
			return true
		}
	}
	return false
}

// jobsForThing returns the recorded jobs that target thingName, oldest first.
func (s *iotStore) jobsForThing(thingName string) []iotJobView {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []iotJobView
	for _, job := range s.registry.jobs {
		if iotJobTargetsThing(*job, thingName) {
			out = append(out, job.clone())
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Seq < out[j].Seq })
	return out
}

// commandExists reports whether CreateCommand recorded commandID.
func (s *iotStore) commandExists(commandID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.registry.commands[commandID]
	return ok
}
