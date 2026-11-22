package server

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/stackyard/stackyard/internal/services/ec2"
)

// The rules packages AWS publishes in us-east-1. Each produces one fixed,
// deterministic sample finding per scanned instance.
type iscRulesPackage struct {
	arn, name, version, description string
	rule                            iscRule
}

type iscRule struct {
	id, title, description, recommendation, severity string
	numeric                                          float64
	attrs                                            []iscKV
}

var iscRulesPackages = []iscRulesPackage{
	{arn: "arn:aws:inspector:us-east-1:316112463485:rulespackage/0-gEjTy7T7", name: "Common Vulnerabilities and Exposures", version: "1.1",
		description: "The rules in this package help verify whether the EC2 instances in your assessment targets are exposed to common vulnerabilities and exposures (CVEs).",
		rule: iscRule{id: "CVE-2021-3156", title: "Instance %s is vulnerable to CVE-2021-3156", severity: "High", numeric: 9,
			description:    "Sudo before 1.9.5p2 contains an off-by-one error that can result in a heap-based buffer overflow (Stackyard sample finding).",
			recommendation: "Use your Operating System's update feature to update package sudo.",
			attrs:          []iscKV{{"CVE_ID", "CVE-2021-3156"}, {"package_name", "sudo"}}}},
	{arn: "arn:aws:inspector:us-east-1:316112463485:rulespackage/0-rExsr2X8", name: "CIS Operating System Security Configuration Benchmarks", version: "1.0",
		description: "The CIS Security Benchmarks program provides well-defined, unbiased and consensus-based industry best practices to help organizations assess and improve their security.",
		rule: iscRule{id: "1.1.1.1", title: "Instance %s is not compliant with rule 1.1.1.1 Ensure mounting of cramfs filesystems is disabled", severity: "High", numeric: 9,
			description:    "The cramfs filesystem type is a compressed read-only Linux filesystem; removing support reduces the attack surface (Stackyard sample finding).",
			recommendation: "Edit or create /etc/modprobe.d/CIS.conf and add: install cramfs /bin/true.",
			attrs:          []iscKV{{"BENCHMARK_ID", "CIS Amazon Linux 2 Benchmark-Level 1"}}}},
	{arn: "arn:aws:inspector:us-east-1:316112463485:rulespackage/0-R01qwB5Q", name: "Security Best Practices", version: "1.0",
		description: "The rules in this package help determine whether your systems are configured securely.",
		rule: iscRule{id: "SSH_ROOT_LOGIN", title: "Instance %s is configured to allow users to log in with root credentials over SSH", severity: "Medium", numeric: 6,
			description:    "Allowing root logins over SSH makes brute force attacks against the root account possible (Stackyard sample finding).",
			recommendation: "Set PermitRootLogin no in /etc/ssh/sshd_config and restart sshd.",
			attrs:          []iscKV{{"RULE_NAME", "Disable root login over SSH"}}}},
	{arn: "arn:aws:inspector:us-east-1:316112463485:rulespackage/0-PmNV0Tcd", name: "Network Reachability", version: "1.1",
		description: "The rules in this package help you analyze the network configurations of your EC2 instances.",
		rule: iscRule{id: "RECOGNIZED_PORT_WITH_LISTENER", title: "On instance %s, TCP port 22 which is associated with 'SSH' is reachable from the internet", severity: "Medium", numeric: 6,
			description:    "A recognized port is reachable from the internet through an internet gateway (Stackyard sample finding).",
			recommendation: "Restrict access to the instance with security groups or network ACLs if SSH access from the internet is not needed.",
			attrs:          []iscKV{{"PORT", "22"}, {"PROTOCOL", "TCP"}}}},
}

func iscRulesPackageByARN(arn string) *iscRulesPackage {
	for i := range iscRulesPackages {
		if iscRulesPackages[i].arn == arn {
			return &iscRulesPackages[i]
		}
	}
	return nil
}

type iscStateChange struct {
	state string
	at    time.Time
}

type iscNotification struct {
	at           time.Time
	event, topic string
}

type iscRun struct {
	arn, name, templateArn             string
	state                              string
	duration                           int
	packages                           []string
	userAttrs                          []iscKV
	created, started, completed, moved time.Time
	changes                            []iscStateChange
	notifications                      []iscNotification
	agents                             []ec2.Instance // running targeted instances
	exclusions                         []string
	findings                           []string
}

type iscFinding struct {
	arn, runArn, packageArn string
	instance                ec2.Instance
	userAttrs               []iscKV
	created, updated        time.Time
}

type iscExclusion struct {
	arn, instanceID string
}

type iscPreview struct {
	template string
	ids      []string // excluded instance ids
}

type iscReport struct {
	runArn, format, kind string
}

func iscRunActive(state string) bool {
	switch state {
	case "COMPLETED", "COMPLETED_WITH_ERRORS", "FAILED", "ERROR", "CANCELED":
		return false
	}
	return true
}

// activeRuns returns in-progress runs of the given templates.
func (st *iscStore) activeRuns(templates ...string) []string {
	var out []string
	for _, arn := range st.runOrder {
		if r := st.runs[arn]; iscContains(templates, r.templateArn) && iscRunActive(r.state) {
			out = append(out, arn)
		}
	}
	return out
}

// targetInstances splits a target's instances into running (scanned) and
// others (excluded). Without a resource group a target covers every
// instance; with one, instances carrying any of the group's tags.
func (st *iscStore) targetInstances(t *iscTarget, all []ec2.Instance) (running, other []ec2.Instance) {
	var tags []iscKV
	if g := st.groups[t.groupArn]; g != nil {
		tags = g.tags
	}
	for _, inst := range all {
		if inst.StateName == "terminated" {
			continue
		}
		match := len(tags) == 0
		for _, kv := range tags {
			if v, ok := inst.Tags[kv.key]; ok && (kv.value == "" || v == kv.value) {
				match = true
				break
			}
		}
		if !match {
			continue
		}
		if inst.StateName == "running" {
			running = append(running, inst)
		} else {
			other = append(other, inst)
		}
	}
	// ponytail: a run scans at most iscMaxRunInstances instances; that cap
	// is what keeps the findings store bounded.
	if len(running) > iscMaxRunInstances {
		running = running[:iscMaxRunInstances]
	}
	return running, other
}

func (st *iscStore) startRun(c *iscCall) (map[string]any, *iscError) {
	tpl, err := st.template(iscStr(c.in, "assessmentTemplateArn"))
	if err != nil {
		return nil, err
	}
	if len(st.runs) >= iscMaxRuns {
		return nil, iscLimit("ASSESSMENT_RUN_LIMIT_EXCEEDED", "an account can have at most %d assessment runs", iscMaxRuns)
	}
	running, other := st.targetInstances(st.targets[tpl.targetArn], c.instances)
	var busy []any
	for _, arn := range st.runOrder {
		r := st.runs[arn]
		if !iscRunActive(r.state) {
			continue
		}
		for _, a := range r.agents {
			for _, mine := range running {
				if a.ID == mine.ID {
					busy = append(busy, map[string]any{"agentId": a.ID, "assessmentRunArn": r.arn})
				}
			}
		}
	}
	if len(busy) > 0 {
		return nil, &iscError{code: "AgentsAlreadyRunningAssessmentException", msg: "agents are already running an assessment",
			extra: map[string]any{"agents": busy, "agentsTruncated": false}}
	}
	name := iscStr(c.in, "assessmentRunName")
	if name == "" {
		name = "Run - " + tpl.name + " - " + fmt.Sprint(tpl.runCount+1)
	}
	now := time.Now().UTC()
	run := &iscRun{
		arn: tpl.arn + "/run/" + iscID(), name: name, templateArn: tpl.arn, duration: tpl.duration,
		packages: append([]string{}, tpl.packages...), userAttrs: append([]iscKV{}, tpl.userAttrs...), agents: running,
		created: now,
	}
	for _, inst := range other {
		ex := &iscExclusion{arn: run.arn + "/exclusion/" + iscID(), instanceID: inst.ID}
		st.exclusions[ex.arn] = ex
		run.exclusions = append(run.exclusions, ex.arn)
	}
	st.runs[run.arn] = run
	st.runOrder = append(st.runOrder, run.arn)
	st.moveRun(run, "CREATED", now)
	tpl.lastRun, tpl.runCount = run.arn, tpl.runCount+1
	return map[string]any{"assessmentRunArn": run.arn}, nil
}

// moveRun records a state change, its notifications and, on completion, the
// run's findings.
func (st *iscStore) moveRun(run *iscRun, state string, now time.Time) {
	run.state, run.moved = state, now
	run.changes = append(run.changes, iscStateChange{state: state, at: now})
	notify := func(event string) {
		for _, topic := range st.subscribedTopics(run.templateArn, event) {
			if len(run.notifications) < iscMaxNotifications {
				run.notifications = append(run.notifications, iscNotification{at: now, event: event, topic: topic})
			}
		}
	}
	notify("ASSESSMENT_RUN_STATE_CHANGED")
	switch state {
	case "COLLECTING_DATA":
		run.started = now
		notify("ASSESSMENT_RUN_STARTED")
	case "COMPLETED", "CANCELED":
		run.completed = now
		if state == "COMPLETED" {
			st.reportFindings(run, now)
			if len(run.findings) > 0 {
				notify("FINDING_REPORTED")
			}
		}
		notify("ASSESSMENT_RUN_COMPLETED")
	}
}

func (st *iscStore) reportFindings(run *iscRun, now time.Time) {
	for _, inst := range run.agents {
		for _, pkg := range run.packages {
			f := &iscFinding{arn: run.arn + "/finding/" + iscID(), runArn: run.arn, packageArn: pkg, instance: inst,
				userAttrs: append([]iscKV{}, run.userAttrs...), created: now, updated: now}
			st.findings[f.arn] = f
			run.findings = append(run.findings, f.arn)
		}
	}
}

// advance moves an active run one phase on; reads settle runs, nothing
// runs in the background.
func (st *iscStore) advance(run *iscRun) {
	next := map[string]string{"CREATED": "COLLECTING_DATA", "COLLECTING_DATA": "EVALUATING_RULES", "EVALUATING_RULES": "COMPLETED"}[run.state]
	if next != "" {
		st.moveRun(run, next, time.Now().UTC())
	}
}

func (st *iscStore) run(arn string) (*iscRun, *iscError) {
	if !iscRunARN.MatchString(arn) {
		return nil, iscInvalid("INVALID_ASSESSMENT_RUN_ARN", "invalid assessment run ARN %s", arn)
	}
	r := st.runs[arn]
	if r == nil {
		return nil, iscNoSuch("ASSESSMENT_RUN_DOES_NOT_EXIST", "assessment run %s does not exist", arn)
	}
	return r, nil
}

func (st *iscStore) stopRun(c *iscCall) (map[string]any, *iscError) {
	run, err := st.run(iscStr(c.in, "assessmentRunArn"))
	if err != nil {
		return nil, err
	}
	if !iscRunActive(run.state) {
		return map[string]any{}, nil
	}
	now := time.Now().UTC()
	if iscStr(c.in, "stopAction") == "SKIP_EVALUATION" {
		st.moveRun(run, "CANCELED", now)
		return map[string]any{}, nil
	}
	if run.state == "CREATED" || run.state == "COLLECTING_DATA" {
		st.moveRun(run, "EVALUATING_RULES", now)
	}
	st.moveRun(run, "COMPLETED", now)
	return map[string]any{}, nil
}

func (st *iscStore) deleteRun(c *iscCall) (map[string]any, *iscError) {
	run, err := st.run(iscStr(c.in, "assessmentRunArn"))
	if err != nil {
		return nil, err
	}
	if iscRunActive(run.state) {
		return nil, iscInProgress([]string{run.arn})
	}
	st.dropRun(run.arn)
	return map[string]any{}, nil
}

func (st *iscStore) dropRun(arn string) {
	run := st.runs[arn]
	for _, f := range run.findings {
		delete(st.findings, f)
	}
	for _, e := range run.exclusions {
		delete(st.exclusions, e)
	}
	for token, rep := range st.reports {
		if rep.runArn == arn {
			delete(st.reports, token)
			st.reportOrder = iscRemove(st.reportOrder, token)
		}
	}
	delete(st.runs, arn)
	st.runOrder = iscRemove(st.runOrder, arn)
}

func (st *iscStore) findingCounts(run *iscRun) map[string]any {
	counts := map[string]any{"Low": 0, "Medium": 0, "High": 0, "Informational": 0, "Undefined": 0}
	for _, f := range run.findings {
		sev := iscRulesPackageByARN(st.findings[f].packageArn).rule.severity
		counts[sev] = counts[sev].(int) + 1
	}
	return counts
}

func (st *iscStore) runOut(run *iscRun) map[string]any {
	changes := []any{}
	for _, ch := range run.changes {
		changes = append(changes, map[string]any{"state": ch.state, "stateChangedAt": iscTime(ch.at)})
	}
	notes := []any{}
	for _, n := range run.notifications {
		// ponytail: notifications are recorded, not published to SNS.
		notes = append(notes, map[string]any{"date": iscTime(n.at), "event": n.event, "error": false, "snsTopicArn": n.topic, "snsPublishStatusCode": "SUCCESS"})
	}
	out := map[string]any{
		"arn": run.arn, "name": run.name, "assessmentTemplateArn": run.templateArn, "state": run.state,
		"durationInSeconds": run.duration, "rulesPackageArns": iscNonNil(run.packages), "userAttributesForFindings": iscKVOut(run.userAttrs),
		"createdAt": iscTime(run.created), "stateChangedAt": iscTime(run.moved), "dataCollected": run.state == "COMPLETED" || run.state == "EVALUATING_RULES",
		"stateChanges": changes, "notifications": notes, "findingCounts": st.findingCounts(run),
	}
	if !run.started.IsZero() {
		out["startedAt"] = iscTime(run.started)
	}
	if !run.completed.IsZero() {
		out["completedAt"] = iscTime(run.completed)
	}
	return out
}

func (st *iscStore) describeRuns(c *iscCall) (map[string]any, *iscError) {
	found, failed := iscDescribe(iscStrList(c.in, "assessmentRunArns"), iscRunARN, func(arn string) (map[string]any, bool) {
		run := st.runs[arn]
		if run == nil {
			return nil, false
		}
		st.advance(run)
		return st.runOut(run), true
	})
	return map[string]any{"assessmentRuns": found, "failedItems": failed}, nil
}

func (st *iscStore) listRuns(c *iscCall) (map[string]any, *iscError) {
	templates := iscStrList(c.in, "assessmentTemplateArns")
	for _, t := range templates {
		if _, err := st.template(t); err != nil {
			return nil, err
		}
	}
	filter, _ := c.in["filter"].(map[string]any)
	states := iscStrList(filter, "states")
	var arns []string
	for _, arn := range st.runOrder {
		r := st.runs[arn]
		switch {
		case len(templates) > 0 && !iscContains(templates, r.templateArn):
		case iscStr(filter, "namePattern") != "" && !iscGlob(iscStr(filter, "namePattern"), r.name):
		case len(states) > 0 && !iscContains(states, r.state):
		case !iscDurationOK(filter["durationRange"], r.duration):
		case !iscAnyIn(iscStrList(filter, "rulesPackageArns"), r.packages):
		case filter["startTimeRange"] != nil && (r.started.IsZero() || !iscInRange(filter["startTimeRange"], r.started)):
		case filter["completionTimeRange"] != nil && (r.completed.IsZero() || !iscInRange(filter["completionTimeRange"], r.completed)):
		case !iscInRange(filter["stateChangeTimeRange"], r.moved):
		default:
			arns = append(arns, arn)
		}
	}
	page, next, err := iscPage(arns, c.in, 500)
	if err != nil {
		return nil, err
	}
	return iscWithNext(map[string]any{"assessmentRunArns": iscNonNil(page)}, next), nil
}

func iscTelemetry(agents int) []any {
	n := max(agents, 1)
	return []any{
		map[string]any{"messageType": "InspectorDuplicateProcess", "count": 0, "dataSize": 0},
		map[string]any{"messageType": "InspectorTimeEventMsg", "count": 12 * n, "dataSize": 2048 * n},
		map[string]any{"messageType": "InspectorProcessEvent", "count": 40 * n, "dataSize": 8192 * n},
	}
}

func (st *iscStore) listRunAgents(c *iscCall) (map[string]any, *iscError) {
	run, err := st.run(iscStr(c.in, "assessmentRunArn"))
	if err != nil {
		return nil, err
	}
	code := "IDLE"
	if iscRunActive(run.state) {
		code = "RUNNING"
	}
	filter, _ := c.in["filter"].(map[string]any)
	var ids []string
	for _, a := range run.agents {
		if filter != nil && (!iscContains(iscStrList(filter, "agentHealths"), "HEALTHY") || !iscContains(iscStrList(filter, "agentHealthCodes"), code)) {
			continue
		}
		ids = append(ids, a.ID)
	}
	page, next, perr := iscPage(ids, c.in, 500)
	if perr != nil {
		return nil, perr
	}
	agents := []any{}
	for _, id := range page {
		agents = append(agents, map[string]any{"agentId": id, "assessmentRunArn": run.arn, "agentHealth": "HEALTHY",
			"agentHealthCode": code, "telemetryMetadata": iscTelemetry(1)})
	}
	return iscWithNext(map[string]any{"assessmentRunAgents": agents}, next), nil
}

func (st *iscStore) telemetry(c *iscCall) (map[string]any, *iscError) {
	run, err := st.run(iscStr(c.in, "assessmentRunArn"))
	if err != nil {
		return nil, err
	}
	return map[string]any{"telemetryMetadata": iscTelemetry(len(run.agents))}, nil
}

func (st *iscStore) previewAgents(c *iscCall) (map[string]any, *iscError) {
	t, err := st.target(iscStr(c.in, "previewAgentsArn"))
	if err != nil {
		return nil, err
	}
	running, other := st.targetInstances(t, c.instances)
	all := append(append([]ec2.Instance{}, running...), other...)
	byID := map[string]ec2.Instance{}
	var ids []string
	for _, inst := range all {
		byID[inst.ID] = inst
		ids = append(ids, inst.ID)
	}
	page, next, perr := iscPage(ids, c.in, 500)
	if perr != nil {
		return nil, perr
	}
	previews := []any{}
	for _, id := range page {
		inst := byID[id]
		health := "UNKNOWN"
		if inst.StateName == "running" {
			health = "HEALTHY"
		}
		p := map[string]any{"agentId": id, "agentHealth": health, "agentVersion": "1.1.1", "operatingSystem": "Amazon Linux 2",
			"kernelVersion": "5.10.0", "hostname": "ip-" + strings.ReplaceAll(inst.PrivateIP, ".", "-") + ".ec2.internal"}
		if inst.PrivateIP != "" {
			p["ipv4Address"] = inst.PrivateIP
		}
		previews = append(previews, p)
	}
	return iscWithNext(map[string]any{"agentPreviews": previews}, next), nil
}

// --- findings ---

func (st *iscStore) findingOut(f *iscFinding) map[string]any {
	pkg := iscRulesPackageByARN(f.packageArn)
	attrs := append([]iscKV{{"INSTANCE_ID", f.instance.ID}}, pkg.rule.attrs...)
	var tags []iscKV
	for k, v := range f.instance.Tags {
		tags = append(tags, iscKV{k, v})
	}
	sort.Slice(tags, func(i, j int) bool { return tags[i].key < tags[j].key })
	asset := map[string]any{"schemaVersion": 1, "agentId": f.instance.ID, "tags": iscKVOut(tags), "networkInterfaces": []any{}}
	if f.instance.ImageID != "" {
		asset["amiId"] = f.instance.ImageID
	}
	if f.instance.PrivateIP != "" {
		asset["ipv4Addresses"] = []any{f.instance.PrivateIP}
		asset["hostname"] = "ip-" + strings.ReplaceAll(f.instance.PrivateIP, ".", "-") + ".ec2.internal"
	}
	return map[string]any{
		"arn": f.arn, "schemaVersion": 1, "service": "Inspector",
		"serviceAttributes": map[string]any{"schemaVersion": 1, "assessmentRunArn": f.runArn, "rulesPackageArn": f.packageArn},
		"assetType":         "ec2-instance", "assetAttributes": asset, "id": pkg.rule.id,
		"title": fmt.Sprintf(pkg.rule.title, f.instance.ID), "description": pkg.rule.description, "recommendation": pkg.rule.recommendation,
		"severity": pkg.rule.severity, "numericSeverity": pkg.rule.numeric, "confidence": 10, "indicatorOfCompromise": false,
		"attributes": iscKVOut(attrs), "userAttributes": iscKVOut(f.userAttrs), "createdAt": iscTime(f.created), "updatedAt": iscTime(f.updated),
	}
}

func (st *iscStore) describeFindings(c *iscCall) (map[string]any, *iscError) {
	found, failed := iscDescribe(iscStrList(c.in, "findingArns"), iscFindingARN, func(arn string) (map[string]any, bool) {
		f := st.findings[arn]
		if f == nil {
			return nil, false
		}
		return st.findingOut(f), true
	})
	return map[string]any{"findings": found, "failedItems": failed}, nil
}

// iscKVsMatch reports whether every wanted key/value is present in have.
func iscKVsMatch(want, have []iscKV) bool {
	for _, w := range want {
		ok := false
		for _, h := range have {
			if h.key == w.key && (w.value == "" || h.value == w.value) {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}
	return true
}

func (st *iscStore) findingMatches(f *iscFinding, filter map[string]any) bool {
	if filter == nil {
		return true
	}
	pkg := iscRulesPackageByARN(f.packageArn)
	attrs := append([]iscKV{{"INSTANCE_ID", f.instance.ID}}, pkg.rule.attrs...)
	switch {
	case len(iscStrList(filter, "agentIds")) > 0 && !iscContains(iscStrList(filter, "agentIds"), f.instance.ID):
	case len(iscStrList(filter, "autoScalingGroups")) > 0: // instances here have no Auto Scaling group
	case len(iscStrList(filter, "ruleNames")) > 0 && !iscContains(iscStrList(filter, "ruleNames"), pkg.rule.id):
	case len(iscStrList(filter, "severities")) > 0 && !iscContains(iscStrList(filter, "severities"), pkg.rule.severity):
	case len(iscStrList(filter, "rulesPackageArns")) > 0 && !iscContains(iscStrList(filter, "rulesPackageArns"), f.packageArn):
	case !iscKVsMatch(iscKVs(filter, "attributes"), attrs):
	case !iscKVsMatch(iscKVs(filter, "userAttributes"), f.userAttrs):
	case !iscInRange(filter["creationTimeRange"], f.created):
	default:
		return true
	}
	return false
}

func (st *iscStore) listFindings(c *iscCall) (map[string]any, *iscError) {
	runs := iscStrList(c.in, "assessmentRunArns")
	for _, r := range runs {
		if _, err := st.run(r); err != nil {
			return nil, err
		}
	}
	filter, _ := c.in["filter"].(map[string]any)
	var arns []string
	for _, runArn := range st.runOrder {
		if len(runs) > 0 && !iscContains(runs, runArn) {
			continue
		}
		for _, f := range st.runs[runArn].findings {
			if st.findingMatches(st.findings[f], filter) {
				arns = append(arns, f)
			}
		}
	}
	page, next, err := iscPage(arns, c.in, 500)
	if err != nil {
		return nil, err
	}
	return iscWithNext(map[string]any{"findingArns": iscNonNil(page)}, next), nil
}

// editFindings applies edit to each finding, collecting failedItems.
func (st *iscStore) editFindings(arns []string, edit func(f *iscFinding) bool) map[string]any {
	failed := map[string]any{}
	for _, arn := range arns {
		f := st.findings[arn]
		switch {
		case !iscFindingARN.MatchString(arn):
			failed[arn] = map[string]any{"failureCode": "INVALID_ARN", "retryable": false}
		case f == nil:
			failed[arn] = map[string]any{"failureCode": "ITEM_DOES_NOT_EXIST", "retryable": false}
		case !edit(f):
			failed[arn] = map[string]any{"failureCode": "LIMIT_EXCEEDED", "retryable": false}
		default:
			f.updated = time.Now().UTC()
		}
	}
	return failed
}

func (st *iscStore) addAttributes(c *iscCall) (map[string]any, *iscError) {
	attrs := iscKVs(c.in, "attributes")
	failed := st.editFindings(iscStrList(c.in, "findingArns"), func(f *iscFinding) bool {
		next := append([]iscKV{}, f.userAttrs...)
		for _, a := range attrs {
			replaced := false
			for i := range next {
				if next[i].key == a.key {
					next[i], replaced = a, true
				}
			}
			if !replaced {
				next = append(next, a)
			}
		}
		if len(next) > 10 {
			return false
		}
		f.userAttrs = next
		return true
	})
	return map[string]any{"failedItems": failed}, nil
}

func (st *iscStore) removeAttributes(c *iscCall) (map[string]any, *iscError) {
	keys := iscStrList(c.in, "attributeKeys")
	failed := st.editFindings(iscStrList(c.in, "findingArns"), func(f *iscFinding) bool {
		kept := []iscKV{}
		for _, a := range f.userAttrs {
			if !iscContains(keys, a.key) {
				kept = append(kept, a)
			}
		}
		f.userAttrs = kept
		return true
	})
	return map[string]any{"failedItems": failed}, nil
}

// --- reports ---

func (st *iscStore) report(c *iscCall) (map[string]any, *iscError) {
	run, err := st.run(iscStr(c.in, "assessmentRunArn"))
	if err != nil {
		return nil, err
	}
	if iscRunActive(run.state) {
		return nil, iscInProgress([]string{run.arn})
	}
	if run.state != "COMPLETED" {
		return nil, &iscError{code: "UnsupportedFeatureException", msg: "reports are available only for completed assessment runs"}
	}
	if len(st.reportOrder) >= iscMaxReports {
		delete(st.reports, st.reportOrder[0])
		st.reportOrder = st.reportOrder[1:]
	}
	token := strings.ToLower(strings.TrimPrefix(iscID(), "0-")) + strings.ToLower(strings.TrimPrefix(iscID(), "0-"))
	format := strings.ToLower(iscStr(c.in, "reportFileFormat"))
	st.reports[token] = iscReport{runArn: run.arn, format: format, kind: iscStr(c.in, "reportType")}
	st.reportOrder = append(st.reportOrder, token)
	return map[string]any{"status": "COMPLETED", "url": c.baseURL + iscReportPath + token + "." + format}, nil
}

// --- exclusions ---

func iscExclusionOut(arn, instanceID string) map[string]any {
	out := map[string]any{
		"title":          "Agent not found",
		"description":    "Instance " + instanceID + " is not running, so no Inspector agent was found to assess it.",
		"recommendation": "Start the instance and make sure the Inspector agent is installed and running.",
		"scopes":         []any{map[string]any{"key": "INSTANCE_ID", "value": instanceID}},
		"attributes":     []any{},
	}
	if arn != "" {
		out["arn"] = arn
	}
	return out
}

func (st *iscStore) createPreview(c *iscCall) (map[string]any, *iscError) {
	tpl, err := st.template(iscStr(c.in, "assessmentTemplateArn"))
	if err != nil {
		return nil, err
	}
	_, other := st.targetInstances(st.targets[tpl.targetArn], c.instances)
	if len(st.previewOrder) >= iscMaxPreviews {
		delete(st.previews, st.previewOrder[0])
		st.previewOrder = st.previewOrder[1:]
	}
	token := uuid.NewString()
	p := &iscPreview{template: tpl.arn}
	for _, inst := range other {
		p.ids = append(p.ids, inst.ID)
	}
	st.previews[token] = p
	st.previewOrder = append(st.previewOrder, token)
	return map[string]any{"previewToken": token}, nil
}

func (st *iscStore) getPreview(c *iscCall) (map[string]any, *iscError) {
	tpl, err := st.template(iscStr(c.in, "assessmentTemplateArn"))
	if err != nil {
		return nil, err
	}
	p := st.previews[iscStr(c.in, "previewToken")]
	if p == nil || p.template != tpl.arn {
		return nil, iscNoSuch("ASSESSMENT_TEMPLATE_DOES_NOT_EXIST", "no exclusions preview %s for template %s", iscStr(c.in, "previewToken"), tpl.arn)
	}
	page, next, perr := iscPage(p.ids, c.in, 500)
	if perr != nil {
		return nil, perr
	}
	previews := []any{}
	for _, id := range page {
		previews = append(previews, iscExclusionOut("", id))
	}
	return iscWithNext(map[string]any{"previewStatus": "COMPLETED", "exclusionPreviews": previews}, next), nil
}

func (st *iscStore) listExclusions(c *iscCall) (map[string]any, *iscError) {
	run, err := st.run(iscStr(c.in, "assessmentRunArn"))
	if err != nil {
		return nil, err
	}
	page, next, perr := iscPage(run.exclusions, c.in, 500)
	if perr != nil {
		return nil, perr
	}
	return iscWithNext(map[string]any{"exclusionArns": iscNonNil(page)}, next), nil
}

func (st *iscStore) describeExclusions(c *iscCall) (map[string]any, *iscError) {
	found, failed := iscDescribe(iscStrList(c.in, "exclusionArns"), iscExclusionARN, func(arn string) (map[string]any, bool) {
		ex := st.exclusions[arn]
		if ex == nil {
			return nil, false
		}
		return iscExclusionOut(ex.arn, ex.instanceID), true
	})
	exclusions := map[string]any{}
	for _, item := range found {
		m := item.(map[string]any)
		exclusions[m["arn"].(string)] = m
	}
	return map[string]any{"exclusions": exclusions, "failedItems": failed}, nil
}

// --- rules packages ---

func (st *iscStore) listRulesPackages(c *iscCall) (map[string]any, *iscError) {
	arns := make([]string, len(iscRulesPackages))
	for i, p := range iscRulesPackages {
		arns[i] = p.arn
	}
	page, next, err := iscPage(arns, c.in, 500)
	if err != nil {
		return nil, err
	}
	return iscWithNext(map[string]any{"rulesPackageArns": iscNonNil(page)}, next), nil
}

func (st *iscStore) describeRulesPackages(c *iscCall) (map[string]any, *iscError) {
	found, failed := iscDescribe(iscStrList(c.in, "rulesPackageArns"), iscPackageARN, func(arn string) (map[string]any, bool) {
		p := iscRulesPackageByARN(arn)
		if p == nil {
			return nil, false
		}
		return map[string]any{"arn": p.arn, "name": p.name, "version": p.version, "provider": "Amazon Web Services, Inc.", "description": p.description}, true
	})
	return map[string]any{"rulesPackages": found, "failedItems": failed}, nil
}
