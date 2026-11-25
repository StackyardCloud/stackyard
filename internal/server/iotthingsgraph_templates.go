package server

import (
	"encoding/json"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// IoT Things Graph flow templates (@workflowType), system templates
// (@systemType), system instances (@deployment) and flow executions.

type itgRevision struct {
	num, nsVersion int
	text           string
	refs           map[string]bool
	at             time.Time
}

type itgTemplate struct {
	id, arn, kind string // kind: Workflow or System
	revisions     []itgRevision
	next          int
	deprecated    bool
	created       time.Time
}

func (t *itgTemplate) latest() itgRevision { return t.revisions[len(t.revisions)-1] }

type itgInstance struct {
	id, arn, target, systemID, text   string
	ggGroupName, ggGroupID, ggVersion string
	s3Bucket, flowRole                string
	metrics                           map[string]any
	deps                              []any
	nsVersion                         int
	status                            string
	tags                              map[string]string
	created, updated                  time.Time
}

type itgMessage struct {
	id, typ, payload string
	at               time.Time
}

type itgExecution struct {
	id, instanceID, flowID, status string
	messages                       []itgMessage
	created, updated               time.Time
}

var (
	itgWorkflowRe   = regexp.MustCompile(`@workflowType\s*\(\s*id\s*:\s*"([^"]*)"`)
	itgSystemRe     = regexp.MustCompile(`@systemType\s*\(\s*id\s*:\s*"([^"]*)"`)
	itgDeploymentRe = regexp.MustCompile(`@deployment\s*\(\s*id\s*:\s*"([^"]*)"`)
	itgSystemIDRe   = regexp.MustCompile(`systemId\s*:\s*"([^"]*)"`)
)

// itgDefinedID reads the id a definition declares and checks that it names
// a kind (Workflow, System, Deployment) in the account's namespace.
func itgDefinedID(text string, re *regexp.Regexp, kind string) (string, *budErr) {
	m := re.FindStringSubmatch(text)
	if m == nil {
		return "", itgErr("The definition has no annotation declaring a %s id.", kind)
	}
	id := m[1]
	rest, ok := strings.CutPrefix(id, itgNSPrefix)
	k, name, _ := strings.Cut(rest, ":")
	if !itgIDPattern.MatchString(id) || !ok || !strings.EqualFold(k, kind) || name == "" {
		return "", itgErr("%s is not a %s id in the namespace %s", id, kind, itgNSName)
	}
	return id, nil
}

func itgARN(kind, id string) string { return itgARNBase + kind + "/default/" + itgEntityName(id) }

// checkRefs requires every referenced id to exist: namespace ids in this
// store, public (aws) ids only once the namespace tracks the public one.
func (st *itgStore) checkRefs(refs map[string]bool) *budErr {
	for ref := range refs {
		if !strings.HasPrefix(ref, itgNSPrefix) {
			if !st.synced {
				return itgErr("%s is outside the namespace and the namespace doesn't track the public namespace", ref)
			}
			continue
		}
		if st.entities[ref] != nil || st.systems[ref] != nil {
			continue
		}
		if f := st.flows[ref]; f != nil {
			if f.deprecated {
				return itgErr("flow template %s is deprecated", ref)
			}
			continue
		}
		return itgErr("%s does not exist in the namespace", ref)
	}
	return nil
}

func (st *itgStore) templates(kind string) map[string]*itgTemplate {
	if kind == "Workflow" {
		return st.flows
	}
	return st.systems
}

func itgTemplateRe(kind string) *regexp.Regexp {
	if kind == "Workflow" {
		return itgWorkflowRe
	}
	return itgSystemRe
}

func (st *itgStore) newRevision(in map[string]any, kind, id string, num int, now time.Time) (itgRevision, *budErr) {
	text := budStr(budMap(in, "definition"), "text")
	if v, ok := in["compatibleNamespaceVersion"].(float64); ok && (v < 1 || int(v) > st.nsVersion) {
		return itgRevision{}, itgErr("compatibleNamespaceVersion %v does not exist; the namespace is at version %d", v, st.nsVersion)
	}
	refs := itgRefs(text, id)
	if err := st.checkRefs(refs); err != nil {
		return itgRevision{}, err
	}
	if kind == "System" {
		for ref := range refs {
			if st.systems[ref] != nil {
				return itgRevision{}, itgErr("a system template cannot reference another system template (%s)", ref)
			}
		}
	}
	if err := st.reserve(len(text)); err != nil {
		return itgRevision{}, err
	}
	return itgRevision{num: num, nsVersion: st.nsVersion, text: text, refs: refs, at: now}, nil
}

func (t *itgTemplate) summary(rev itgRevision) map[string]any {
	return map[string]any{"id": t.id, "arn": t.arn, "revisionNumber": rev.num, "createdAt": budEpoch(rev.at)}
}

func (st *itgStore) createTemplate(kind string, in map[string]any, now time.Time) (map[string]any, *budErr) {
	id, err := itgDefinedID(budStr(budMap(in, "definition"), "text"), itgTemplateRe(kind), kind)
	if err != nil {
		return nil, err
	}
	all := st.templates(kind)
	if all[id] != nil {
		return nil, budErrorf("ResourceAlreadyExistsException", "%s template %s already exists", kind, id)
	}
	if len(all) >= itgMaxTemplates {
		return nil, budErrorf("LimitExceededException", "at most %d %s templates can exist", itgMaxTemplates, kind)
	}
	rev, err := st.newRevision(in, kind, id, 1, now)
	if err != nil {
		return nil, err
	}
	t := &itgTemplate{id: id, arn: itgARN(kind, id), kind: kind, revisions: []itgRevision{rev}, next: 2, created: now}
	all[id] = t
	return map[string]any{"summary": t.summary(rev)}, nil
}

func (st *itgStore) template(kind, id string) (*itgTemplate, *budErr) {
	if !itgIDPattern.MatchString(id) {
		return nil, itgErr("%s is not a valid id", id)
	}
	t := st.templates(kind)[id]
	if t == nil {
		return nil, itgNotFound("%s template %s not found", kind, id)
	}
	return t, nil
}

func (st *itgStore) updateTemplate(kind string, in map[string]any, now time.Time) (map[string]any, *budErr) {
	t, err := st.template(kind, budStr(in, "id"))
	if err != nil {
		return nil, err
	}
	defined, err := itgDefinedID(budStr(budMap(in, "definition"), "text"), itgTemplateRe(kind), kind)
	if err != nil {
		return nil, err
	}
	if defined != t.id {
		return nil, itgErr("the definition declares %s, not %s", defined, t.id)
	}
	rev, err := st.newRevision(in, kind, t.id, t.next, now)
	if err != nil {
		return nil, err
	}
	revs := append(append([]itgRevision{}, t.revisions...), rev)
	for len(revs) > itgMaxRevisions {
		st.bytes -= len(revs[0].text)
		revs = revs[1:]
	}
	t.revisions, t.next = revs, t.next+1
	return map[string]any{"summary": t.summary(rev)}, nil
}

func (st *itgStore) getTemplate(kind string, in map[string]any) (map[string]any, *budErr) {
	t, err := st.template(kind, budStr(in, "id"))
	if err != nil {
		return nil, err
	}
	rev := t.latest()
	if n, ok := in["revisionNumber"].(float64); ok {
		found := false
		for _, r := range t.revisions {
			if float64(r.num) == n {
				rev, found = r, true
			}
		}
		if !found {
			return nil, itgNotFound("%s template %s has no revision %v", kind, t.id, n)
		}
	}
	return map[string]any{"description": map[string]any{
		"summary": t.summary(rev), "definition": map[string]any{"language": "GRAPHQL", "text": rev.text},
		"validatedNamespaceVersion": rev.nsVersion,
	}}, nil
}

func (st *itgStore) templateRevisions(kind string, in map[string]any) (map[string]any, *budErr) {
	t, err := st.template(kind, budStr(in, "id"))
	if err != nil {
		return nil, err
	}
	start, end, next, err := budPage(budStr(in, "nextToken"), len(t.revisions), budInt(in, "maxResults", 100), itgInvalid)
	if err != nil {
		return nil, err
	}
	out := make([]any, 0, end-start)
	for i := start; i < end; i++ {
		out = append(out, t.summary(t.revisions[len(t.revisions)-1-i])) // newest first
	}
	return map[string]any{"summaries": out, "nextToken": next}, nil
}

func (st *itgStore) deprecateTemplate(kind string, in map[string]any) (map[string]any, *budErr) {
	t, err := st.template(kind, budStr(in, "id"))
	if err != nil {
		return nil, err
	}
	t.deprecated = true
	return map[string]any{}, nil
}

// deleteTemplate refuses while a system template (for flows) or a system
// instance (for systems) still references it.
func (st *itgStore) deleteTemplate(kind string, in map[string]any) (map[string]any, *budErr) {
	id := budStr(in, "id")
	t := st.templates(kind)[id]
	if t == nil {
		return nil, itgErr("%s template %s does not exist", kind, id)
	}
	if kind == "Workflow" {
		for _, s := range st.systems {
			if s.latest().refs[id] {
				return nil, budErrorf("ResourceInUseException", "system template %s uses flow template %s", s.id, id)
			}
		}
	} else {
		for _, inst := range st.instances {
			if inst.systemID == id {
				return nil, budErrorf("ResourceInUseException", "system instance %s uses system template %s", inst.id, id)
			}
		}
	}
	for _, r := range t.revisions {
		st.bytes -= len(r.text)
	}
	delete(st.templates(kind), id)
	return map[string]any{}, nil
}

// searchTemplates lists templates (sorted by id) referencing any value of
// every filter.
func (st *itgStore) searchTemplates(kind string, in map[string]any) (map[string]any, *budErr) {
	var match []*itgTemplate
	for _, t := range st.templates(kind) {
		ok := true
		for _, f := range budList(in, "filters") {
			fm, _ := f.(map[string]any)
			hit := false
			for _, v := range budStrings(fm, "value") {
				hit = hit || t.latest().refs[v]
			}
			ok = ok && hit
		}
		if ok {
			match = append(match, t)
		}
	}
	sort.Slice(match, func(i, j int) bool { return match[i].id < match[j].id })
	start, end, next, err := budPage(budStr(in, "nextToken"), len(match), budInt(in, "maxResults", 100), itgInvalid)
	if err != nil {
		return nil, err
	}
	out := make([]any, 0, end-start)
	for _, t := range match[start:end] {
		out = append(out, t.summary(t.latest()))
	}
	return map[string]any{"summaries": out, "nextToken": next}, nil
}

func (st *itgStore) createFlow(in map[string]any, now time.Time) (map[string]any, *budErr) {
	return st.createTemplate("Workflow", in, now)
}

func (st *itgStore) updateFlow(in map[string]any, now time.Time) (map[string]any, *budErr) {
	return st.updateTemplate("Workflow", in, now)
}

func (st *itgStore) getFlow(in map[string]any, _ time.Time) (map[string]any, *budErr) {
	return st.getTemplate("Workflow", in)
}

func (st *itgStore) flowRevisions(in map[string]any, _ time.Time) (map[string]any, *budErr) {
	return st.templateRevisions("Workflow", in)
}

func (st *itgStore) deprecateFlow(in map[string]any, _ time.Time) (map[string]any, *budErr) {
	return st.deprecateTemplate("Workflow", in)
}

func (st *itgStore) deleteFlow(in map[string]any, _ time.Time) (map[string]any, *budErr) {
	return st.deleteTemplate("Workflow", in)
}

func (st *itgStore) searchFlows(in map[string]any, _ time.Time) (map[string]any, *budErr) {
	return st.searchTemplates("Workflow", in)
}

func (st *itgStore) createSystem(in map[string]any, now time.Time) (map[string]any, *budErr) {
	return st.createTemplate("System", in, now)
}

func (st *itgStore) updateSystem(in map[string]any, now time.Time) (map[string]any, *budErr) {
	return st.updateTemplate("System", in, now)
}

func (st *itgStore) getSystem(in map[string]any, _ time.Time) (map[string]any, *budErr) {
	return st.getTemplate("System", in)
}

func (st *itgStore) systemRevisions(in map[string]any, _ time.Time) (map[string]any, *budErr) {
	return st.templateRevisions("System", in)
}

func (st *itgStore) deprecateSystem(in map[string]any, _ time.Time) (map[string]any, *budErr) {
	return st.deprecateTemplate("System", in)
}

func (st *itgStore) deleteSystem(in map[string]any, _ time.Time) (map[string]any, *budErr) {
	return st.deleteTemplate("System", in)
}

func (st *itgStore) searchSystems(in map[string]any, _ time.Time) (map[string]any, *budErr) {
	return st.searchTemplates("System", in)
}

// ---- system instances ----

func (st *itgStore) createInstance(in map[string]any, now time.Time) (map[string]any, *budErr) {
	text := budStr(budMap(in, "definition"), "text")
	id, err := itgDefinedID(text, itgDeploymentRe, "Deployment")
	if err != nil {
		return nil, err
	}
	target := budStr(in, "target")
	inst := &itgInstance{
		id: id, arn: itgARN("Deployment", id), target: target, text: text, status: "NOT_DEPLOYED",
		ggGroupName: budStr(in, "greengrassGroupName"), s3Bucket: budStr(in, "s3BucketName"),
		flowRole: budStr(in, "flowActionsRoleArn"), created: now, updated: now,
	}
	switch {
	case target == "GREENGRASS" && (inst.ggGroupName == "" || inst.s3Bucket == ""):
		return nil, itgErr("GREENGRASS deployments need greengrassGroupName and s3BucketName")
	case target == "CLOUD" && inst.flowRole == "":
		return nil, itgErr("CLOUD deployments need flowActionsRoleArn")
	}
	if m := budMap(in, "metricsConfiguration"); m != nil {
		if budBool(m, "cloudMetricEnabled") && budStr(m, "metricRuleRoleArn") == "" {
			return nil, itgErr("cloudMetricEnabled needs metricRuleRoleArn")
		}
		inst.metrics = budCloneMap(m)
	}
	sys := itgSystemIDRe.FindStringSubmatch(text)
	if sys == nil {
		return nil, itgErr("the @deployment annotation needs a systemId")
	}
	t := st.systems[sys[1]]
	if t == nil || t.deprecated {
		return nil, itgErr("system template %s does not exist or is deprecated", sys[1])
	}
	inst.systemID = t.id
	inst.deps = []any{map[string]any{"id": t.id, "revisionNumber": t.latest().num}}
	for ref := range t.latest().refs {
		if f := st.flows[ref]; f != nil {
			inst.deps = append(inst.deps, map[string]any{"id": f.id, "revisionNumber": f.latest().num})
		}
	}
	tags, err := itgTagsIn(budList(in, "tags"))
	if err != nil {
		return nil, err
	}
	inst.tags, inst.nsVersion = tags, st.nsVersion
	if st.instances[id] != nil {
		return nil, budErrorf("ResourceAlreadyExistsException", "system instance %s already exists", id)
	}
	if len(st.instances) >= itgMaxInstances {
		return nil, budErrorf("LimitExceededException", "at most %d system instances can exist", itgMaxInstances)
	}
	if err := st.reserve(len(text)); err != nil {
		return nil, err
	}
	st.instances[id] = inst
	return map[string]any{"summary": inst.summary()}, nil
}

func (inst *itgInstance) summary() map[string]any {
	out := map[string]any{"id": inst.id, "arn": inst.arn, "status": inst.status, "target": inst.target,
		"createdAt": budEpoch(inst.created), "updatedAt": budEpoch(inst.updated)}
	for k, v := range map[string]string{"greengrassGroupName": inst.ggGroupName, "greengrassGroupId": inst.ggGroupID, "greengrassGroupVersionId": inst.ggVersion} {
		if v != "" {
			out[k] = v
		}
	}
	return out
}

// settle finishes an in-progress deployment or undeployment.
func (st *itgStore) settle(inst *itgInstance, now time.Time) {
	switch inst.status {
	case "DEPLOY_IN_PROGRESS":
		inst.status, inst.updated = "DEPLOYED_IN_TARGET", now
		if inst.target == "CLOUD" {
			st.startExecutions(inst, now)
		}
	case "UNDEPLOY_IN_PROGRESS":
		inst.status, inst.updated = "NOT_DEPLOYED", now
	}
}

func (st *itgStore) instance(id string, now time.Time) (*itgInstance, *budErr) {
	if !itgIDPattern.MatchString(id) {
		return nil, itgErr("%s is not a valid id", id)
	}
	inst := st.instances[id]
	if inst == nil {
		return nil, itgNotFound("system instance %s not found", id)
	}
	st.settle(inst, now)
	return inst, nil
}

func (st *itgStore) getInstance(in map[string]any, now time.Time) (map[string]any, *budErr) {
	inst, err := st.instance(budStr(in, "id"), now)
	if err != nil {
		return nil, err
	}
	desc := map[string]any{
		"summary": inst.summary(), "definition": map[string]any{"language": "GRAPHQL", "text": inst.text},
		"validatedNamespaceVersion": inst.nsVersion, "validatedDependencyRevisions": budClone(inst.deps),
	}
	for k, v := range map[string]string{"s3BucketName": inst.s3Bucket, "flowActionsRoleArn": inst.flowRole} {
		if v != "" {
			desc[k] = v
		}
	}
	if inst.metrics != nil {
		desc["metricsConfiguration"] = budCloneMap(inst.metrics)
	}
	return map[string]any{"description": desc}, nil
}

func (st *itgStore) requireID(in map[string]any) (string, *budErr) {
	id := budStr(in, "id")
	if id == "" {
		return "", itgErr("id is required")
	}
	return id, nil
}

func (st *itgStore) deployInstance(in map[string]any, now time.Time) (map[string]any, *budErr) {
	id, err := st.requireID(in)
	if err != nil {
		return nil, err
	}
	inst, err := st.instance(id, now)
	if err != nil {
		return nil, err
	}
	if inst.status == "DEPLOY_IN_PROGRESS" || inst.status == "UNDEPLOY_IN_PROGRESS" {
		return nil, budErrorf("ResourceInUseException", "system instance %s is %s", id, inst.status)
	}
	inst.status, inst.updated = "DEPLOY_IN_PROGRESS", now
	out := map[string]any{}
	if inst.target == "GREENGRASS" {
		if inst.ggGroupID == "" {
			inst.ggGroupID = uuid.NewString()
		}
		inst.ggVersion = uuid.NewString()
		out["greengrassDeploymentId"] = uuid.NewString()
	}
	out["summary"] = inst.summary()
	return out, nil
}

func (st *itgStore) undeployInstance(in map[string]any, now time.Time) (map[string]any, *budErr) {
	id, err := st.requireID(in)
	if err != nil {
		return nil, err
	}
	inst, err := st.instance(id, now)
	if err != nil {
		return nil, err
	}
	switch inst.status {
	case "DEPLOYED_IN_TARGET", "FAILED":
	case "UNDEPLOY_IN_PROGRESS":
		return nil, budErrorf("ResourceInUseException", "system instance %s is already being undeployed", id)
	default:
		return nil, itgErr("system instance %s is %s, not deployed", id, inst.status)
	}
	inst.status, inst.updated = "UNDEPLOY_IN_PROGRESS", now
	return map[string]any{"summary": inst.summary()}, nil
}

func (st *itgStore) deleteInstance(in map[string]any, now time.Time) (map[string]any, *budErr) {
	id, err := st.requireID(in)
	if err != nil {
		return nil, err
	}
	inst := st.instances[id]
	if inst == nil {
		return nil, itgErr("system instance %s does not exist", id)
	}
	st.settle(inst, now)
	if inst.status != "NOT_DEPLOYED" && inst.status != "FAILED" {
		return nil, budErrorf("ResourceInUseException", "system instance %s is %s; undeploy it first", id, inst.status)
	}
	st.bytes -= len(inst.text)
	delete(st.instances, id)
	st.execOrder = idaWithout(st.execOrder, func(e string) bool {
		if st.executions[e].instanceID == id {
			delete(st.executions, e)
			return true
		}
		return false
	})
	return map[string]any{}, nil
}

func (st *itgStore) searchInstances(in map[string]any, now time.Time) (map[string]any, *budErr) {
	var match []*itgInstance
	for _, inst := range st.instances {
		st.settle(inst, now)
		ok := true
		for _, f := range budList(in, "filters") {
			fm, _ := f.(map[string]any)
			field := map[string]string{"SYSTEM_TEMPLATE_ID": inst.systemID, "STATUS": inst.status, "GREENGRASS_GROUP_NAME": inst.ggGroupName}[budStr(fm, "name")]
			values := budStrings(fm, "value")
			hit := len(values) == 0
			for _, v := range values {
				hit = hit || v == field
			}
			ok = ok && hit
		}
		if ok {
			match = append(match, inst)
		}
	}
	sort.Slice(match, func(i, j int) bool { return match[i].id < match[j].id })
	start, end, next, err := budPage(budStr(in, "nextToken"), len(match), budInt(in, "maxResults", 100), itgInvalid)
	if err != nil {
		return nil, err
	}
	out := make([]any, 0, end-start)
	for _, inst := range match[start:end] {
		out = append(out, inst.summary())
	}
	return map[string]any{"summaries": out, "nextToken": next}, nil
}

// ---- flow executions ----

// startExecutions records one completed execution per flow the deployed
// system uses; the oldest executions are dropped past the cap.
func (st *itgStore) startExecutions(inst *itgInstance, now time.Time) {
	t := st.systems[inst.systemID]
	if t == nil {
		return
	}
	var flows []string
	for ref := range t.latest().refs {
		if st.flows[ref] != nil {
			flows = append(flows, ref)
		}
	}
	sort.Strings(flows)
	for _, flow := range flows {
		id := uuid.NewString()
		var msgs []itgMessage
		for i, typ := range []string{"EXECUTION_STARTED", "STEP_STARTED", "STEP_SUCCEEDED", "EXECUTION_SUCCEEDED"} {
			payload, _ := json.Marshal(map[string]any{"flowExecutionId": id, "flowTemplateId": flow, "systemInstanceId": inst.id, "step": i})
			msgs = append(msgs, itgMessage{id: id + "-" + strconv.Itoa(i), typ: typ, payload: string(payload), at: now})
		}
		st.executions[id] = &itgExecution{id: id, instanceID: inst.id, flowID: flow, status: "SUCCEEDED", messages: msgs, created: now, updated: now}
		st.execOrder = append(st.execOrder, id)
	}
	if over := len(st.execOrder) - itgMaxExecutions; over > 0 {
		for _, id := range st.execOrder[:over] {
			delete(st.executions, id)
		}
		st.execOrder = append([]string{}, st.execOrder[over:]...)
	}
}

func (st *itgStore) searchExecutions(in map[string]any, now time.Time) (map[string]any, *budErr) {
	inst, err := st.instance(budStr(in, "systemInstanceId"), now)
	if err != nil {
		return nil, err
	}
	from, hasFrom := budTime(in["startTime"])
	to, hasTo := budTime(in["endTime"])
	want := budStr(in, "flowExecutionId")
	var match []*itgExecution
	for _, id := range st.execOrder {
		e := st.executions[id]
		if e.instanceID != inst.id || (want != "" && e.id != want) ||
			(hasFrom && e.created.Before(from)) || (hasTo && e.created.After(to)) {
			continue
		}
		match = append(match, e)
	}
	start, end, next, err := budPage(budStr(in, "nextToken"), len(match), budInt(in, "maxResults", 100), itgInvalid)
	if err != nil {
		return nil, err
	}
	out := make([]any, 0, end-start)
	for _, e := range match[start:end] {
		out = append(out, map[string]any{"flowExecutionId": e.id, "status": e.status, "systemInstanceId": e.instanceID,
			"flowTemplateId": e.flowID, "createdAt": budEpoch(e.created), "updatedAt": budEpoch(e.updated)})
	}
	return map[string]any{"summaries": out, "nextToken": next}, nil
}

func (st *itgStore) executionMessages(in map[string]any, _ time.Time) (map[string]any, *budErr) {
	e := st.executions[budStr(in, "flowExecutionId")]
	if e == nil {
		return nil, itgNotFound("flow execution %s not found", budStr(in, "flowExecutionId"))
	}
	start, end, next, err := budPage(budStr(in, "nextToken"), len(e.messages), budInt(in, "maxResults", 100), itgInvalid)
	if err != nil {
		return nil, err
	}
	out := make([]any, 0, end-start)
	for _, m := range e.messages[start:end] {
		out = append(out, map[string]any{"messageId": m.id, "eventType": m.typ, "timestamp": budEpoch(m.at), "payload": m.payload})
	}
	return map[string]any{"messages": out, "nextToken": next}, nil
}

// ---- tags (system instances) ----

func itgTagsIn(list []any) (map[string]string, *budErr) {
	out := map[string]string{}
	for _, item := range list {
		m, _ := item.(map[string]any)
		k := budStr(m, "key")
		if strings.HasPrefix(strings.ToLower(k), "aws:") {
			return nil, itgErr("tag keys starting with aws: are reserved")
		}
		out[k] = budStr(m, "value")
	}
	if len(out) > itgMaxTags {
		return nil, itgErr("a resource can have at most %d tags", itgMaxTags)
	}
	return out, nil
}

func (st *itgStore) tagged(arn string) (*itgInstance, *budErr) {
	for _, inst := range st.instances {
		if inst.arn == arn {
			return inst, nil
		}
	}
	return nil, itgErr("resource %s does not exist", arn)
}

func (st *itgStore) tagResource(in map[string]any, _ time.Time) (map[string]any, *budErr) {
	add, err := itgTagsIn(budList(in, "tags"))
	if err != nil {
		return nil, err
	}
	inst, err := st.tagged(budStr(in, "resourceArn"))
	if err != nil {
		return nil, err
	}
	merged, err := budMergeTags(inst.tags, add, itgMaxTags, itgInvalid)
	if err != nil {
		return nil, err
	}
	inst.tags = merged
	return map[string]any{}, nil
}

func (st *itgStore) untagResource(in map[string]any, _ time.Time) (map[string]any, *budErr) {
	inst, err := st.tagged(budStr(in, "resourceArn"))
	if err != nil {
		return nil, err
	}
	inst.tags = budDropTags(inst.tags, budStrings(in, "tagKeys"))
	return map[string]any{}, nil
}

func (st *itgStore) listTags(in map[string]any, _ time.Time) (map[string]any, *budErr) {
	inst, err := st.tagged(budStr(in, "resourceArn"))
	if err != nil {
		return nil, err
	}
	tags := istTagsOut(inst.tags)
	start, end, next, err := budPage(budStr(in, "nextToken"), len(tags), budInt(in, "maxResults", 100), itgInvalid)
	if err != nil {
		return nil, err
	}
	return map[string]any{"tags": tags[start:end], "nextToken": next}, nil
}
