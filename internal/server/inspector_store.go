package server

import (
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

// Resource limits. AWS allows 50 targets and 500 templates per account; the
// run cap keeps the findings store bounded (runs x instances x packages).
const (
	iscMaxGroups        = 100
	iscMaxTargets       = 50
	iscMaxTemplates     = 500
	iscMaxRuns          = 200
	iscMaxRunInstances  = 25
	iscMaxSubscriptions = 50
	iscMaxReports       = 1000
	iscMaxPreviews      = 500
	iscMaxNotifications = 100
)

var (
	iscArnPrefix    = `^arn:aws:inspector:[a-z0-9-]+:\d{12}:`
	iscGroupARN     = regexp.MustCompile(iscArnPrefix + `resourcegroup/0-[A-Za-z0-9]{8}$`)
	iscTargetARN    = regexp.MustCompile(iscArnPrefix + `target/0-[A-Za-z0-9]{8}$`)
	iscTemplateARN  = regexp.MustCompile(iscArnPrefix + `target/0-[A-Za-z0-9]{8}/template/0-[A-Za-z0-9]{8}$`)
	iscRunARN       = regexp.MustCompile(iscArnPrefix + `target/0-[A-Za-z0-9]{8}/template/0-[A-Za-z0-9]{8}/run/0-[A-Za-z0-9]{8}$`)
	iscFindingARN   = regexp.MustCompile(iscArnPrefix + `target/0-[A-Za-z0-9]{8}/template/0-[A-Za-z0-9]{8}/run/0-[A-Za-z0-9]{8}/finding/0-[A-Za-z0-9]{8}$`)
	iscExclusionARN = regexp.MustCompile(iscArnPrefix + `target/0-[A-Za-z0-9]{8}/template/0-[A-Za-z0-9]{8}/run/0-[A-Za-z0-9]{8}/exclusion/0-[A-Za-z0-9]{8}$`)
	iscPackageARN   = regexp.MustCompile(iscArnPrefix + `rulespackage/0-[A-Za-z0-9]{8}$`)
	iscRoleARN      = regexp.MustCompile(`^arn:aws:iam::\d{12}:role/[\w+=,.@/-]{1,512}$`)
	iscTopicARN     = regexp.MustCompile(`^arn:aws:sns:[a-z0-9-]+:\d{12}:[A-Za-z0-9_-]{1,256}$`)
)

type iscKV struct{ key, value string }

type iscGroup struct {
	arn     string
	tags    []iscKV
	created time.Time
}

type iscTarget struct {
	arn, name, groupArn string
	created, updated    time.Time
}

type iscTemplate struct {
	arn, name, targetArn string
	duration             int
	packages             []string
	userAttrs            []iscKV
	created              time.Time
	lastRun              string
	runCount             int
}

type iscStore struct {
	mu            sync.Mutex
	groups        map[string]*iscGroup
	groupOrder    []string
	targets       map[string]*iscTarget
	targetOrder   []string
	templates     map[string]*iscTemplate
	templateOrder []string
	runs          map[string]*iscRun
	runOrder      []string
	findings      map[string]*iscFinding
	exclusions    map[string]*iscExclusion
	previews      map[string]*iscPreview
	previewOrder  []string
	reports       map[string]iscReport
	reportOrder   []string
	tags          map[string][]iscKV
	subs          map[string]map[string]map[string]time.Time // resource -> topic -> event -> subscribedAt
	roleArn       string
	roleAt        time.Time
	roleSet       bool
	created       time.Time
}

func newInspectorStore() *iscStore {
	return &iscStore{
		groups: map[string]*iscGroup{}, targets: map[string]*iscTarget{}, templates: map[string]*iscTemplate{},
		runs: map[string]*iscRun{}, findings: map[string]*iscFinding{}, exclusions: map[string]*iscExclusion{},
		previews: map[string]*iscPreview{}, reports: map[string]iscReport{}, tags: map[string][]iscKV{},
		subs: map[string]map[string]map[string]time.Time{}, created: time.Now().UTC(),
	}
}

type iscHandler func(st *iscStore, c *iscCall) (map[string]any, *iscError)

var iscHandlers map[string]iscHandler

func init() {
	iscHandlers = map[string]iscHandler{
		"CreateResourceGroup": (*iscStore).createResourceGroup, "DescribeResourceGroups": (*iscStore).describeResourceGroups,
		"CreateAssessmentTarget": (*iscStore).createTarget, "UpdateAssessmentTarget": (*iscStore).updateTarget,
		"DeleteAssessmentTarget": (*iscStore).deleteTarget, "DescribeAssessmentTargets": (*iscStore).describeTargets,
		"ListAssessmentTargets": (*iscStore).listTargets, "CreateAssessmentTemplate": (*iscStore).createTemplate,
		"DeleteAssessmentTemplate": (*iscStore).deleteTemplate, "DescribeAssessmentTemplates": (*iscStore).describeTemplates,
		"ListAssessmentTemplates": (*iscStore).listTemplates, "ListRulesPackages": (*iscStore).listRulesPackages,
		"DescribeRulesPackages": (*iscStore).describeRulesPackages, "SetTagsForResource": (*iscStore).setTags,
		"ListTagsForResource": (*iscStore).listTags, "SubscribeToEvent": (*iscStore).subscribe,
		"UnsubscribeFromEvent": (*iscStore).unsubscribe, "ListEventSubscriptions": (*iscStore).listSubscriptions,
		"RegisterCrossAccountAccessRole": (*iscStore).registerRole, "DescribeCrossAccountAccessRole": (*iscStore).describeRole,
		"StartAssessmentRun": (*iscStore).startRun, "StopAssessmentRun": (*iscStore).stopRun,
		"DeleteAssessmentRun": (*iscStore).deleteRun, "DescribeAssessmentRuns": (*iscStore).describeRuns,
		"ListAssessmentRuns": (*iscStore).listRuns, "ListAssessmentRunAgents": (*iscStore).listRunAgents,
		"GetTelemetryMetadata": (*iscStore).telemetry, "PreviewAgents": (*iscStore).previewAgents,
		"DescribeFindings": (*iscStore).describeFindings, "ListFindings": (*iscStore).listFindings,
		"AddAttributesToFindings": (*iscStore).addAttributes, "RemoveAttributesFromFindings": (*iscStore).removeAttributes,
		"GetAssessmentReport": (*iscStore).report, "CreateExclusionsPreview": (*iscStore).createPreview,
		"GetExclusionsPreview": (*iscStore).getPreview, "ListExclusions": (*iscStore).listExclusions,
		"DescribeExclusions": (*iscStore).describeExclusions,
	}
}

// dispatch runs op under the store lock and shapes the result to the output
// model while still holding it, so nothing reads stored state after unlock.
func (st *iscStore) dispatch(op, outShape string, c *iscCall) (any, *iscError) {
	h := iscHandlers[op]
	if h == nil {
		return nil, &iscError{code: "InternalException", msg: "operation " + op + " has no handler"}
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	out, err := h(st, c)
	if err != nil {
		return nil, err
	}
	if outShape == "" {
		return map[string]any{}, nil
	}
	return awsmodels.InspectorModel().Conform(outShape, out), nil
}

func iscKVs(in map[string]any, key string) []iscKV {
	var out []iscKV
	items, _ := in[key].([]any)
	for _, it := range items {
		m, _ := it.(map[string]any)
		out = append(out, iscKV{key: iscStr(m, "key"), value: iscStr(m, "value")})
	}
	return out
}

func iscKVOut(kvs []iscKV) []any {
	out := make([]any, 0, len(kvs))
	for _, kv := range kvs {
		m := map[string]any{"key": kv.key}
		if kv.value != "" {
			m["value"] = kv.value
		}
		out = append(out, m)
	}
	return out
}

func iscARN(kind string) string {
	return "arn:aws:inspector:" + iscRegion + ":" + iscAccount + ":" + kind
}

// --- resource groups ---

func (st *iscStore) createResourceGroup(c *iscCall) (map[string]any, *iscError) {
	if len(st.groups) >= iscMaxGroups {
		return nil, iscLimit("RESOURCE_GROUP_LIMIT_EXCEEDED", "an account can have at most %d resource groups", iscMaxGroups)
	}
	g := &iscGroup{arn: iscARN("resourcegroup/" + iscID()), tags: iscKVs(c.in, "resourceGroupTags"), created: time.Now().UTC()}
	st.groups[g.arn] = g
	st.groupOrder = append(st.groupOrder, g.arn)
	return map[string]any{"resourceGroupArn": g.arn}, nil
}

// iscDescribe runs one lookup per requested ARN and collects failedItems.
func iscDescribe(arns []string, pattern *regexp.Regexp, lookup func(string) (map[string]any, bool)) ([]any, map[string]any) {
	found, failed := []any{}, map[string]any{}
	seen := map[string]bool{}
	for _, arn := range arns {
		switch {
		case seen[arn]:
			failed[arn] = map[string]any{"failureCode": "DUPLICATE_ARN", "retryable": false}
		case !pattern.MatchString(arn):
			failed[arn] = map[string]any{"failureCode": "INVALID_ARN", "retryable": false}
		default:
			if item, ok := lookup(arn); ok {
				found = append(found, item)
			} else {
				failed[arn] = map[string]any{"failureCode": "ITEM_DOES_NOT_EXIST", "retryable": false}
			}
		}
		seen[arn] = true
	}
	return found, failed
}

func (st *iscStore) describeResourceGroups(c *iscCall) (map[string]any, *iscError) {
	found, failed := iscDescribe(iscStrList(c.in, "resourceGroupArns"), iscGroupARN, func(arn string) (map[string]any, bool) {
		g := st.groups[arn]
		if g == nil {
			return nil, false
		}
		return map[string]any{"arn": g.arn, "tags": iscKVOut(g.tags), "createdAt": iscTime(g.created)}, true
	})
	return map[string]any{"resourceGroups": found, "failedItems": failed}, nil
}

// --- assessment targets ---

func (st *iscStore) checkGroup(arn string) *iscError {
	if arn == "" {
		return nil
	}
	if !iscGroupARN.MatchString(arn) {
		return iscInvalid("INVALID_RESOURCE_GROUP_ARN", "invalid resource group ARN %s", arn)
	}
	if st.groups[arn] == nil {
		return iscNoSuch("RESOURCE_GROUP_DOES_NOT_EXIST", "resource group %s does not exist", arn)
	}
	return nil
}

func (st *iscStore) targetNameTaken(name, except string) bool {
	for _, t := range st.targets {
		if t.name == name && t.arn != except {
			return true
		}
	}
	return false
}

func (st *iscStore) createTarget(c *iscCall) (map[string]any, *iscError) {
	name, group := iscStr(c.in, "assessmentTargetName"), iscStr(c.in, "resourceGroupArn")
	if err := st.checkGroup(group); err != nil {
		return nil, err
	}
	if st.targetNameTaken(name, "") {
		return nil, iscInvalid("ASSESSMENT_TARGET_NAME_ALREADY_TAKEN", "assessment target name %s is already taken", name)
	}
	if len(st.targets) >= iscMaxTargets {
		return nil, iscLimit("ASSESSMENT_TARGET_LIMIT_EXCEEDED", "an account can have at most %d assessment targets", iscMaxTargets)
	}
	now := time.Now().UTC()
	t := &iscTarget{arn: iscARN("target/" + iscID()), name: name, groupArn: group, created: now, updated: now}
	st.targets[t.arn] = t
	st.targetOrder = append(st.targetOrder, t.arn)
	return map[string]any{"assessmentTargetArn": t.arn}, nil
}

func (st *iscStore) target(arn string) (*iscTarget, *iscError) {
	if !iscTargetARN.MatchString(arn) {
		return nil, iscInvalid("INVALID_ASSESSMENT_TARGET_ARN", "invalid assessment target ARN %s", arn)
	}
	t := st.targets[arn]
	if t == nil {
		return nil, iscNoSuch("ASSESSMENT_TARGET_DOES_NOT_EXIST", "assessment target %s does not exist", arn)
	}
	return t, nil
}

func (st *iscStore) updateTarget(c *iscCall) (map[string]any, *iscError) {
	t, err := st.target(iscStr(c.in, "assessmentTargetArn"))
	if err != nil {
		return nil, err
	}
	name, group := iscStr(c.in, "assessmentTargetName"), iscStr(c.in, "resourceGroupArn")
	if err := st.checkGroup(group); err != nil {
		return nil, err
	}
	if st.targetNameTaken(name, t.arn) {
		return nil, iscInvalid("ASSESSMENT_TARGET_NAME_ALREADY_TAKEN", "assessment target name %s is already taken", name)
	}
	t.name, t.groupArn, t.updated = name, group, time.Now().UTC()
	return map[string]any{}, nil
}

func (st *iscStore) deleteTarget(c *iscCall) (map[string]any, *iscError) {
	t, err := st.target(iscStr(c.in, "assessmentTargetArn"))
	if err != nil {
		return nil, err
	}
	var templates []string
	for _, tpl := range st.templateOrder {
		if st.templates[tpl].targetArn == t.arn {
			templates = append(templates, tpl)
		}
	}
	if busy := st.activeRuns(templates...); len(busy) > 0 {
		return nil, iscInProgress(busy)
	}
	for _, tpl := range templates {
		st.dropTemplate(tpl)
	}
	delete(st.targets, t.arn)
	st.targetOrder = iscRemove(st.targetOrder, t.arn)
	return map[string]any{}, nil
}

func (st *iscStore) describeTargets(c *iscCall) (map[string]any, *iscError) {
	found, failed := iscDescribe(iscStrList(c.in, "assessmentTargetArns"), iscTargetARN, func(arn string) (map[string]any, bool) {
		t := st.targets[arn]
		if t == nil {
			return nil, false
		}
		out := map[string]any{"arn": t.arn, "name": t.name, "createdAt": iscTime(t.created), "updatedAt": iscTime(t.updated)}
		if t.groupArn != "" {
			out["resourceGroupArn"] = t.groupArn
		}
		return out, true
	})
	return map[string]any{"assessmentTargets": found, "failedItems": failed}, nil
}

func (st *iscStore) listTargets(c *iscCall) (map[string]any, *iscError) {
	filter, _ := c.in["filter"].(map[string]any)
	pattern := iscStr(filter, "assessmentTargetNamePattern")
	var arns []string
	for _, arn := range st.targetOrder {
		if pattern == "" || iscGlob(pattern, st.targets[arn].name) {
			arns = append(arns, arn)
		}
	}
	page, next, err := iscPage(arns, c.in, 500)
	if err != nil {
		return nil, err
	}
	return iscWithNext(map[string]any{"assessmentTargetArns": iscNonNil(page)}, next), nil
}

// --- assessment templates ---

func (st *iscStore) createTemplate(c *iscCall) (map[string]any, *iscError) {
	t, err := st.target(iscStr(c.in, "assessmentTargetArn"))
	if err != nil {
		return nil, err
	}
	name := iscStr(c.in, "assessmentTemplateName")
	for _, tpl := range st.templates {
		if tpl.targetArn == t.arn && tpl.name == name {
			return nil, iscInvalid("ASSESSMENT_TEMPLATE_NAME_ALREADY_TAKEN", "assessment template name %s is already taken", name)
		}
	}
	var packages []string
	for _, p := range iscStrList(c.in, "rulesPackageArns") {
		if !iscPackageARN.MatchString(p) {
			return nil, iscInvalid("INVALID_RULES_PACKAGE_ARN", "invalid rules package ARN %s", p)
		}
		if iscRulesPackageByARN(p) == nil {
			return nil, iscNoSuch("RULES_PACKAGE_DOES_NOT_EXIST", "rules package %s does not exist", p)
		}
		if !iscContains(packages, p) {
			packages = append(packages, p)
		}
	}
	if len(st.templates) >= iscMaxTemplates {
		return nil, iscLimit("ASSESSMENT_TEMPLATE_LIMIT_EXCEEDED", "an account can have at most %d assessment templates", iscMaxTemplates)
	}
	tpl := &iscTemplate{
		arn: t.arn + "/template/" + iscID(), name: name, targetArn: t.arn, duration: iscInt(c.in, "durationInSeconds", 3600),
		packages: packages, userAttrs: iscKVs(c.in, "userAttributesForFindings"), created: time.Now().UTC(),
	}
	st.templates[tpl.arn] = tpl
	st.templateOrder = append(st.templateOrder, tpl.arn)
	return map[string]any{"assessmentTemplateArn": tpl.arn}, nil
}

func (st *iscStore) template(arn string) (*iscTemplate, *iscError) {
	if !iscTemplateARN.MatchString(arn) {
		return nil, iscInvalid("INVALID_ASSESSMENT_TEMPLATE_ARN", "invalid assessment template ARN %s", arn)
	}
	tpl := st.templates[arn]
	if tpl == nil {
		return nil, iscNoSuch("ASSESSMENT_TEMPLATE_DOES_NOT_EXIST", "assessment template %s does not exist", arn)
	}
	return tpl, nil
}

func (st *iscStore) deleteTemplate(c *iscCall) (map[string]any, *iscError) {
	tpl, err := st.template(iscStr(c.in, "assessmentTemplateArn"))
	if err != nil {
		return nil, err
	}
	if busy := st.activeRuns(tpl.arn); len(busy) > 0 {
		return nil, iscInProgress(busy)
	}
	st.dropTemplate(tpl.arn)
	return map[string]any{}, nil
}

// dropTemplate deletes a template with its runs, tags and subscriptions.
func (st *iscStore) dropTemplate(arn string) {
	for _, run := range append([]string{}, st.runOrder...) {
		if st.runs[run].templateArn == arn {
			st.dropRun(run)
		}
	}
	delete(st.templates, arn)
	delete(st.tags, arn)
	delete(st.subs, arn)
	st.templateOrder = iscRemove(st.templateOrder, arn)
}

func (st *iscStore) describeTemplates(c *iscCall) (map[string]any, *iscError) {
	found, failed := iscDescribe(iscStrList(c.in, "assessmentTemplateArns"), iscTemplateARN, func(arn string) (map[string]any, bool) {
		tpl := st.templates[arn]
		if tpl == nil {
			return nil, false
		}
		out := map[string]any{
			"arn": tpl.arn, "name": tpl.name, "assessmentTargetArn": tpl.targetArn, "durationInSeconds": tpl.duration,
			"rulesPackageArns": iscNonNil(tpl.packages), "userAttributesForFindings": iscKVOut(tpl.userAttrs),
			"assessmentRunCount": tpl.runCount, "createdAt": iscTime(tpl.created),
		}
		if tpl.lastRun != "" {
			out["lastAssessmentRunArn"] = tpl.lastRun
		}
		return out, true
	})
	return map[string]any{"assessmentTemplates": found, "failedItems": failed}, nil
}

func iscDurationOK(rng any, d int) bool {
	m, ok := rng.(map[string]any)
	if !ok {
		return true
	}
	if lo := iscInt(m, "minSeconds", -1); lo >= 0 && d < lo {
		return false
	}
	if hi := iscInt(m, "maxSeconds", -1); hi >= 0 && d > hi {
		return false
	}
	return true
}

func iscAnyIn(want, have []string) bool {
	if len(want) == 0 {
		return true
	}
	for _, w := range want {
		if iscContains(have, w) {
			return true
		}
	}
	return false
}

func (st *iscStore) listTemplates(c *iscCall) (map[string]any, *iscError) {
	targets := iscStrList(c.in, "assessmentTargetArns")
	for _, t := range targets {
		if _, err := st.target(t); err != nil {
			return nil, err
		}
	}
	filter, _ := c.in["filter"].(map[string]any)
	var arns []string
	for _, arn := range st.templateOrder {
		tpl := st.templates[arn]
		if len(targets) > 0 && !iscContains(targets, tpl.targetArn) {
			continue
		}
		if p := iscStr(filter, "namePattern"); p != "" && !iscGlob(p, tpl.name) {
			continue
		}
		if !iscDurationOK(filter["durationRange"], tpl.duration) || !iscAnyIn(iscStrList(filter, "rulesPackageArns"), tpl.packages) {
			continue
		}
		arns = append(arns, arn)
	}
	page, next, err := iscPage(arns, c.in, 500)
	if err != nil {
		return nil, err
	}
	return iscWithNext(map[string]any{"assessmentTemplateArns": iscNonNil(page)}, next), nil
}

// --- tags (assessment templates only, as in AWS) ---

func (st *iscStore) setTags(c *iscCall) (map[string]any, *iscError) {
	tpl, err := st.template(iscStr(c.in, "resourceArn"))
	if err != nil {
		err.errorCode = "INVALID_RESOURCE_ARN"
		if err.code == "NoSuchEntityException" {
			err.errorCode = "ASSESSMENT_TEMPLATE_DOES_NOT_EXIST"
		}
		return nil, err
	}
	tags := iscKVs(c.in, "tags")
	seen := map[string]bool{}
	for _, t := range tags {
		if seen[t.key] {
			return nil, iscInvalid("INVALID_TAG_KEY", "duplicate tag key %s", t.key)
		}
		seen[t.key] = true
	}
	st.tags[tpl.arn] = tags
	return map[string]any{}, nil
}

func (st *iscStore) listTags(c *iscCall) (map[string]any, *iscError) {
	tpl, err := st.template(iscStr(c.in, "resourceArn"))
	if err != nil {
		return nil, err
	}
	return map[string]any{"tags": iscKVOut(st.tags[tpl.arn])}, nil
}

// --- event subscriptions ---

func (st *iscStore) subscribe(c *iscCall) (map[string]any, *iscError) {
	tpl, err := st.template(iscStr(c.in, "resourceArn"))
	if err != nil {
		return nil, err
	}
	topic := iscStr(c.in, "topicArn")
	if !iscTopicARN.MatchString(topic) {
		return nil, iscInvalid("INVALID_SNS_TOPIC_ARN", "invalid SNS topic ARN %s", topic)
	}
	if !c.topicExists {
		return nil, iscNoSuch("SNS_TOPIC_DOES_NOT_EXIST", "SNS topic %s does not exist", topic)
	}
	byTopic := st.subs[tpl.arn]
	if byTopic == nil {
		byTopic = map[string]map[string]time.Time{}
		st.subs[tpl.arn] = byTopic
	}
	if byTopic[topic] == nil {
		if len(byTopic) >= iscMaxSubscriptions {
			return nil, iscLimit("EVENT_SUBSCRIPTION_LIMIT_EXCEEDED", "a resource can have at most %d topic subscriptions", iscMaxSubscriptions)
		}
		byTopic[topic] = map[string]time.Time{}
	}
	event := iscStr(c.in, "event")
	if _, ok := byTopic[topic][event]; !ok {
		byTopic[topic][event] = time.Now().UTC()
	}
	return map[string]any{}, nil
}

func (st *iscStore) unsubscribe(c *iscCall) (map[string]any, *iscError) {
	tpl, err := st.template(iscStr(c.in, "resourceArn"))
	if err != nil {
		return nil, err
	}
	topic, event := iscStr(c.in, "topicArn"), iscStr(c.in, "event")
	if events := st.subs[tpl.arn][topic]; events != nil {
		delete(events, event)
		if len(events) == 0 {
			delete(st.subs[tpl.arn], topic)
		}
	}
	return map[string]any{}, nil
}

func (st *iscStore) listSubscriptions(c *iscCall) (map[string]any, *iscError) {
	resource := iscStr(c.in, "resourceArn")
	if resource != "" {
		if _, err := st.template(resource); err != nil {
			return nil, err
		}
	}
	type key struct{ resource, topic string }
	var keys []key
	for res, byTopic := range st.subs {
		if resource != "" && res != resource {
			continue
		}
		for topic := range byTopic {
			keys = append(keys, key{res, topic})
		}
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].resource != keys[j].resource {
			return keys[i].resource < keys[j].resource
		}
		return keys[i].topic < keys[j].topic
	})
	ids := make([]string, len(keys))
	for i := range keys {
		ids[i] = keys[i].resource + "\x00" + keys[i].topic
	}
	page, next, err := iscPage(ids, c.in, 500)
	if err != nil {
		return nil, err
	}
	subs := []any{}
	for _, id := range page {
		res, topic, _ := strings.Cut(id, "\x00")
		var events []any
		names := make([]string, 0, len(st.subs[res][topic]))
		for ev := range st.subs[res][topic] {
			names = append(names, ev)
		}
		sort.Strings(names)
		for _, ev := range names {
			events = append(events, map[string]any{"event": ev, "subscribedAt": iscTime(st.subs[res][topic][ev])})
		}
		subs = append(subs, map[string]any{"resourceArn": res, "topicArn": topic, "eventSubscriptions": events})
	}
	return iscWithNext(map[string]any{"subscriptions": subs}, next), nil
}

// subscribedTopics returns topics subscribed to event on a template.
func (st *iscStore) subscribedTopics(template, event string) []string {
	var out []string
	for topic, events := range st.subs[template] {
		if _, ok := events[event]; ok {
			out = append(out, topic)
		}
	}
	sort.Strings(out)
	return out
}

// --- cross-account role ---

func (st *iscStore) registerRole(c *iscCall) (map[string]any, *iscError) {
	arn := iscStr(c.in, "roleArn")
	if !iscRoleARN.MatchString(arn) {
		return nil, iscInvalid("INVALID_IAM_ROLE_ARN", "invalid IAM role ARN %s", arn)
	}
	st.roleArn, st.roleAt, st.roleSet = arn, time.Now().UTC(), true
	return map[string]any{}, nil
}

func (st *iscStore) describeRole(*iscCall) (map[string]any, *iscError) {
	if !st.roleSet {
		// ponytail: before registration AWS reports the account's default
		// role as not valid; Stackyard never requires a registered role.
		return map[string]any{"roleArn": "arn:aws:iam::" + iscAccount + ":role/inspector", "valid": false, "registeredAt": iscTime(st.created)}, nil
	}
	return map[string]any{"roleArn": st.roleArn, "valid": true, "registeredAt": iscTime(st.roleAt)}, nil
}

// --- shared helpers ---

func iscContains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

func iscRemove(list []string, v string) []string {
	out := list[:0]
	for _, x := range list {
		if x != v {
			out = append(out, x)
		}
	}
	return out
}

// iscNonNil returns list as []any, the form the model's Conform walks.
func iscNonNil(list []string) []any {
	out := make([]any, len(list))
	for i, v := range list {
		out[i] = v
	}
	return out
}

func iscWithNext(out map[string]any, next string) map[string]any {
	if next != "" {
		out["nextToken"] = next
	}
	return out
}
