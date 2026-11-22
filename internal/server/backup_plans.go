package server

import (
	"encoding/json"
	"regexp"
	"strings"
)

var (
	bkPlanNamePattern = regexp.MustCompile(`^[a-zA-Z0-9\-_.]{1,50}$`)
	bkSchedulePattern = regexp.MustCompile(`^(cron|rate|at)\(.+\)$`)
)

const bkMaxRules = 50

// bkPlan keeps every version of a plan; the last one is current.
type bkPlan struct {
	versions   []map[string]any
	selections map[string]map[string]any
	deleted    float64
}

func (p *bkPlan) current() map[string]any { return p.versions[len(p.versions)-1] }

func bkPlanHandlers() map[string]bkHandler {
	return map[string]bkHandler{
		"CreateBackupPlan":          (*backupStore).createBackupPlan,
		"UpdateBackupPlan":          (*backupStore).updateBackupPlan,
		"GetBackupPlan":             (*backupStore).getBackupPlan,
		"DeleteBackupPlan":          (*backupStore).deleteBackupPlan,
		"ListBackupPlans":           (*backupStore).listBackupPlans,
		"ListBackupPlanVersions":    (*backupStore).listBackupPlanVersions,
		"ExportBackupPlanTemplate":  (*backupStore).exportBackupPlanTemplate,
		"GetBackupPlanFromJSON":     (*backupStore).getBackupPlanFromJSON,
		"ListBackupPlanTemplates":   (*backupStore).listBackupPlanTemplates,
		"GetBackupPlanFromTemplate": (*backupStore).getBackupPlanFromTemplate,
		"CreateBackupSelection":     (*backupStore).createBackupSelection,
		"GetBackupSelection":        (*backupStore).getBackupSelection,
		"DeleteBackupSelection":     (*backupStore).deleteBackupSelection,
		"ListBackupSelections":      (*backupStore).listBackupSelections,
	}
}

// bkCheckLifecycle applies AWS Backup's lifecycle rules: cold storage needs
// DeleteAfterDays at least 90 days later.
func bkCheckLifecycle(lc map[string]any) *bedrockAgentError {
	if lc == nil {
		return nil
	}
	del, delAbsent, err := bkInt(lc, "DeleteAfterDays")
	if err != nil {
		return err
	}
	cold, coldAbsent, err := bkInt(lc, "MoveToColdStorageAfterDays")
	if err != nil {
		return err
	}
	if (!delAbsent && del < 1) || (!coldAbsent && cold < 1) {
		return bkInvalid("lifecycle days must be positive")
	}
	if !delAbsent && !coldAbsent && del < cold+90 {
		return bkInvalid("DeleteAfterDays must be at least 90 days greater than MoveToColdStorageAfterDays")
	}
	return nil
}

// bkNormalizePlan validates a BackupPlanInput and returns the stored
// BackupPlan, keeping RuleIds of rules that keep their name.
func bkNormalizePlan(plan map[string]any, previous map[string]any) (map[string]any, *bedrockAgentError) {
	if plan == nil {
		return nil, bkMissing("BackupPlan is required")
	}
	name := bkStr(plan, "BackupPlanName")
	if !bkPlanNamePattern.MatchString(name) {
		return nil, bkInvalid("BackupPlanName must be 1-50 letters, numbers, '-', '_' or '.'")
	}
	rules := bkList(plan, "Rules")
	if len(rules) == 0 || len(rules) > bkMaxRules {
		return nil, bkInvalid("BackupPlan must have between 1 and 50 rules")
	}
	oldIDs := map[string]any{}
	for _, r := range bkList(previous, "Rules") {
		rule, _ := r.(map[string]any)
		oldIDs[bkStr(rule, "RuleName")] = rule["RuleId"]
	}
	seen := map[string]bool{}
	out := make([]any, 0, len(rules))
	for _, r := range rules {
		rule, ok := r.(map[string]any)
		if !ok {
			return nil, bkInvalid("each rule must be an object")
		}
		if err := bkRequire(rule, "RuleName"); err != nil {
			return nil, err
		}
		ruleName := bkStr(rule, "RuleName")
		if !bkPlanNamePattern.MatchString(ruleName) || seen[ruleName] {
			return nil, bkInvalid("rule names must be unique and 1-50 letters, numbers, '-', '_' or '.'")
		}
		seen[ruleName] = true
		if bkStr(rule, "TargetBackupVaultName") == "" {
			return nil, bkMissing("rule " + ruleName + " needs TargetBackupVaultName")
		}
		stored := baClone(rule)
		if sched := bkStr(rule, "ScheduleExpression"); sched == "" {
			stored["ScheduleExpression"] = "cron(0 5 ? * * *)"
		} else if !bkSchedulePattern.MatchString(sched) {
			return nil, bkInvalid("ScheduleExpression must be a cron() or rate() expression")
		}
		if bkStr(rule, "ScheduleExpressionTimezone") == "" {
			stored["ScheduleExpressionTimezone"] = "Etc/UTC"
		}
		start, startAbsent, err := bkInt(rule, "StartWindowMinutes")
		if err != nil {
			return nil, err
		}
		if startAbsent {
			start = 480
		}
		complete, completeAbsent, err := bkInt(rule, "CompletionWindowMinutes")
		if err != nil {
			return nil, err
		}
		if completeAbsent {
			complete = 10080
		}
		if start < 60 || complete < start+60 {
			return nil, bkInvalid("StartWindowMinutes must be at least 60 and CompletionWindowMinutes at least 60 more")
		}
		stored["StartWindowMinutes"], stored["CompletionWindowMinutes"] = start, complete
		if err := bkCheckLifecycle(bkMap(rule, "Lifecycle")); err != nil {
			return nil, err
		}
		for _, c := range bkList(rule, "CopyActions") {
			action, _ := c.(map[string]any)
			if !strings.HasPrefix(bkStr(action, "DestinationBackupVaultArn"), "arn:aws") {
				return nil, bkInvalid("CopyActions need a DestinationBackupVaultArn")
			}
			if err := bkCheckLifecycle(bkMap(action, "Lifecycle")); err != nil {
				return nil, err
			}
		}
		if id := oldIDs[ruleName]; id != nil {
			stored["RuleId"] = id
		} else {
			stored["RuleId"] = bkNewID()
		}
		out = append(out, stored)
	}
	normalized := baClone(plan)
	normalized["Rules"] = out
	return normalized, nil
}

func bkVersionID() string { return strings.ReplaceAll(bkNewID(), "-", "") }

func (st *backupStore) plan(id string) (*bkPlan, *bedrockAgentError) {
	p := st.plans[id]
	if p == nil {
		return nil, bkNotFound("Backup plan", id)
	}
	return p, nil
}

func (st *backupStore) livePlan(id string) (*bkPlan, *bedrockAgentError) {
	p, err := st.plan(id)
	if err != nil {
		return nil, err
	}
	if p.deleted > 0 {
		return nil, bkNotFound("Backup plan", id)
	}
	return p, nil
}

func (st *backupStore) createBackupPlan(_ *Server, in map[string]any) (any, *bedrockAgentError) {
	if err := bkRequire(in, "BackupPlan"); err != nil {
		return nil, err
	}
	if err := bkCheckToken(in, "CreatorRequestId"); err != nil {
		return nil, err
	}
	plan, err := bkNormalizePlan(bkMap(in, "BackupPlan"), nil)
	if err != nil {
		return nil, err
	}
	for _, p := range st.plans {
		cur := p.current()
		if tok := bkStr(in, "CreatorRequestId"); tok != "" && cur["CreatorRequestId"] == tok {
			return cur, nil // idempotent retry
		}
		if p.deleted == 0 && bkStr(bkMap(cur, "BackupPlan"), "BackupPlanName") == bkStr(plan, "BackupPlanName") {
			return nil, bkExists("Backup plan", bkStr(plan, "BackupPlanName"))
		}
	}
	if len(st.plans) >= backupMaxItems {
		return nil, bkErr("LimitExceededException", "too many backup plans")
	}
	id := bkNewID()
	arn := bkARN("backup-plan", id)
	version := map[string]any{
		"BackupPlan": plan, "BackupPlanId": id, "BackupPlanArn": arn, "VersionId": bkVersionID(),
		"CreationDate": bkNow(), "CreatorRequestId": in["CreatorRequestId"],
		"AdvancedBackupSettings": plan["AdvancedBackupSettings"],
	}
	if err := st.setTags(arn, bkMap(in, "BackupPlanTags")); err != nil {
		return nil, err
	}
	st.plans[id] = &bkPlan{versions: []map[string]any{version}, selections: map[string]map[string]any{}}
	return version, nil
}

func (st *backupStore) updateBackupPlan(_ *Server, in map[string]any) (any, *bedrockAgentError) {
	p, err := st.livePlan(bkStr(in, "BackupPlanId"))
	if err != nil {
		return nil, err
	}
	if err := bkRequire(in, "BackupPlan"); err != nil {
		return nil, err
	}
	cur := p.current()
	plan, err := bkNormalizePlan(bkMap(in, "BackupPlan"), bkMap(cur, "BackupPlan"))
	if err != nil {
		return nil, err
	}
	for id, other := range st.plans {
		if id != cur["BackupPlanId"] && other.deleted == 0 && bkStr(bkMap(other.current(), "BackupPlan"), "BackupPlanName") == bkStr(plan, "BackupPlanName") {
			return nil, bkExists("Backup plan", bkStr(plan, "BackupPlanName"))
		}
	}
	if len(p.versions) >= backupMaxItems {
		return nil, bkErr("LimitExceededException", "too many versions of this backup plan")
	}
	version := baClone(cur)
	version["BackupPlan"] = plan
	version["VersionId"] = bkVersionID()
	version["CreationDate"] = bkNow()
	version["AdvancedBackupSettings"] = plan["AdvancedBackupSettings"]
	p.versions = append(p.versions, version)
	return version, nil
}

func (st *backupStore) getBackupPlan(_ *Server, in map[string]any) (any, *bedrockAgentError) {
	p, err := st.plan(bkStr(in, "BackupPlanId"))
	if err != nil {
		return nil, err
	}
	version := p.current()
	if want := bkStr(in, "VersionId"); want != "" {
		version = nil
		for _, v := range p.versions {
			if v["VersionId"] == want {
				version = v
			}
		}
		if version == nil {
			return nil, bkNotFound("Backup plan version", want)
		}
	}
	if n, absent, err := bkInt(in, "MaxScheduledRunsPreview"); err != nil {
		return nil, err
	} else if !absent && (n < 0 || n > 10) {
		return nil, bkInvalid("MaxScheduledRunsPreview must be between 0 and 10")
	}
	out := baClone(version)
	if p.deleted > 0 {
		out["DeletionDate"] = p.deleted
	}
	return out, nil
}

func (st *backupStore) deleteBackupPlan(_ *Server, in map[string]any) (any, *bedrockAgentError) {
	p, err := st.livePlan(bkStr(in, "BackupPlanId"))
	if err != nil {
		return nil, err
	}
	if len(p.selections) > 0 {
		return nil, bkRequest("Related backup plan selections must be deleted prior to backup plan deletion")
	}
	p.deleted = bkNow()
	cur := p.current()
	return map[string]any{"BackupPlanId": cur["BackupPlanId"], "BackupPlanArn": cur["BackupPlanArn"], "VersionId": cur["VersionId"], "DeletionDate": p.deleted}, nil
}

func bkPlanListMember(p *bkPlan, version map[string]any) map[string]any {
	item := bkShape(version, "BackupPlansListMember")
	item["BackupPlanName"] = bkStr(bkMap(version, "BackupPlan"), "BackupPlanName")
	if p.deleted > 0 {
		item["DeletionDate"] = p.deleted
	}
	return item
}

func (st *backupStore) listBackupPlans(_ *Server, in map[string]any) (any, *bedrockAgentError) {
	includeDeleted := baBool(in["IncludeDeleted"])
	items := []map[string]any{}
	for _, p := range bkSorted(st.plans, func(p *bkPlan) map[string]any { return p.versions[0] }, "CreationDate") {
		if p.deleted > 0 && !includeDeleted {
			continue
		}
		items = append(items, bkPlanListMember(p, p.current()))
	}
	return bkPage(items, in, "BackupPlansList")
}

func (st *backupStore) listBackupPlanVersions(_ *Server, in map[string]any) (any, *bedrockAgentError) {
	p, err := st.plan(bkStr(in, "BackupPlanId"))
	if err != nil {
		return nil, err
	}
	items := []map[string]any{}
	for i := len(p.versions) - 1; i >= 0; i-- {
		items = append(items, bkPlanListMember(p, p.versions[i]))
	}
	return bkPage(items, in, "BackupPlanVersionsList")
}

// bkPlanDocument is a stored plan in template form (no RuleIds).
func bkPlanDocument(plan map[string]any) map[string]any {
	doc := baClone(plan)
	for _, r := range bkList(doc, "Rules") {
		if rule, ok := r.(map[string]any); ok {
			delete(rule, "RuleId")
		}
	}
	return doc
}

func (st *backupStore) exportBackupPlanTemplate(_ *Server, in map[string]any) (any, *bedrockAgentError) {
	p, err := st.livePlan(bkStr(in, "BackupPlanId"))
	if err != nil {
		return nil, err
	}
	raw, _ := json.Marshal(map[string]any{"BackupPlan": bkPlanDocument(bkMap(p.current(), "BackupPlan"))})
	return map[string]any{"BackupPlanTemplateJson": string(raw)}, nil
}

func (st *backupStore) getBackupPlanFromJSON(_ *Server, in map[string]any) (any, *bedrockAgentError) {
	if err := bkRequire(in, "BackupPlanTemplateJson"); err != nil {
		return nil, err
	}
	var doc map[string]any
	if raw := bkStr(in, "BackupPlanTemplateJson"); len(raw) > backupMaxBody || json.Unmarshal([]byte(raw), &doc) != nil {
		return nil, bkInvalid("BackupPlanTemplateJson must be a JSON backup plan document")
	}
	if inner := bkMap(doc, "BackupPlan"); inner != nil {
		doc = inner
	}
	plan, err := bkNormalizePlan(doc, nil)
	if err != nil {
		return nil, err
	}
	return map[string]any{"BackupPlan": plan}, nil
}

// bkPlanTemplates are the built-in plan templates AWS Backup offers.
var bkPlanTemplates = []struct{ id, name string }{
	{"87c0c1ef-254d-4180-8fef-2e76a2c38aaa", "Daily-35day-Retention"},
	{"af38a7d0-8c4a-4b8e-9c51-8b8d6a7c1d01", "Daily-Monthly-1yr-Retention"},
	{"5a6f0b2e-3c8d-4e7f-a1b2-c3d4e5f60718", "Daily-Weekly-Monthly-5YR-Retention"},
	{"0d1e2f3a-4b5c-4d6e-8f70-81920a3b4c5d", "Daily-Weekly-Monthly-7YR-Retention"},
}

func (st *backupStore) listBackupPlanTemplates(_ *Server, in map[string]any) (any, *bedrockAgentError) {
	items := []map[string]any{}
	for _, t := range bkPlanTemplates {
		items = append(items, map[string]any{"BackupPlanTemplateId": t.id, "BackupPlanTemplateName": t.name})
	}
	return bkPage(items, in, "BackupPlanTemplatesList")
}

func (st *backupStore) getBackupPlanFromTemplate(_ *Server, in map[string]any) (any, *bedrockAgentError) {
	id := bkStr(in, "BackupPlanTemplateId")
	for _, t := range bkPlanTemplates {
		if t.id != id {
			continue
		}
		rule := map[string]any{
			"RuleName": "DailyBackups", "TargetBackupVaultName": "Default", "ScheduleExpression": "cron(0 5 ? * * *)",
			"StartWindowMinutes": 480, "CompletionWindowMinutes": 10080, "Lifecycle": map[string]any{"DeleteAfterDays": 35},
		}
		return map[string]any{"BackupPlanDocument": map[string]any{"BackupPlanName": t.name, "Rules": []any{rule}}}, nil
	}
	return nil, bkNotFound("Backup plan template", id)
}

func (st *backupStore) createBackupSelection(_ *Server, in map[string]any) (any, *bedrockAgentError) {
	p, err := st.livePlan(bkStr(in, "BackupPlanId"))
	if err != nil {
		return nil, err
	}
	if err := bkRequire(in, "BackupSelection"); err != nil {
		return nil, err
	}
	if err := bkCheckToken(in, "CreatorRequestId"); err != nil {
		return nil, err
	}
	sel := bkMap(in, "BackupSelection")
	if err := bkRequire(sel, "SelectionName", "IamRoleArn"); err != nil {
		return nil, err
	}
	if !bkPlanNamePattern.MatchString(bkStr(sel, "SelectionName")) {
		return nil, bkInvalid("SelectionName must be 1-50 letters, numbers, '-', '_' or '.'")
	}
	if err := bkCheckRole(sel, "IamRoleArn"); err != nil {
		return nil, err
	}
	if len(bkList(sel, "Resources")) == 0 && len(bkList(sel, "ListOfTags")) == 0 && bkMap(sel, "Conditions") == nil {
		return nil, bkInvalid("BackupSelection needs Resources, ListOfTags or Conditions")
	}
	for _, key := range []string{"Resources", "NotResources"} {
		list := bkList(sel, key)
		if len(list) > 500 {
			return nil, bkInvalid(key + " can list at most 500 resources")
		}
		for _, r := range list {
			if s, _ := r.(string); s != "*" && !strings.HasPrefix(s, "arn:") {
				return nil, bkInvalid(key + " entries must be resource ARNs or \"*\"")
			}
		}
	}
	for _, c := range bkList(sel, "ListOfTags") {
		cond, _ := c.(map[string]any)
		if bkStr(cond, "ConditionType") != "STRINGEQUALS" || bkStr(cond, "ConditionKey") == "" {
			return nil, bkInvalid("ListOfTags conditions need ConditionType STRINGEQUALS and a ConditionKey")
		}
	}
	for _, s := range p.selections {
		if tok := bkStr(in, "CreatorRequestId"); tok != "" && s["CreatorRequestId"] == tok {
			return s, nil
		}
		if bkStr(bkMap(s, "BackupSelection"), "SelectionName") == bkStr(sel, "SelectionName") {
			return nil, bkExists("Backup selection", bkStr(sel, "SelectionName"))
		}
	}
	if len(p.selections) >= backupMaxItems {
		return nil, bkErr("LimitExceededException", "too many selections for this backup plan")
	}
	id := bkNewID()
	rec := map[string]any{
		"SelectionId": id, "BackupPlanId": p.current()["BackupPlanId"], "CreationDate": bkNow(),
		"CreatorRequestId": in["CreatorRequestId"], "BackupSelection": baClone(sel),
	}
	p.selections[id] = rec
	return rec, nil
}

func (st *backupStore) selection(in map[string]any) (*bkPlan, map[string]any, *bedrockAgentError) {
	p, err := st.livePlan(bkStr(in, "BackupPlanId"))
	if err != nil {
		return nil, nil, err
	}
	sel := p.selections[bkStr(in, "SelectionId")]
	if sel == nil {
		return nil, nil, bkNotFound("Backup selection", bkStr(in, "SelectionId"))
	}
	return p, sel, nil
}

func (st *backupStore) getBackupSelection(_ *Server, in map[string]any) (any, *bedrockAgentError) {
	_, sel, err := st.selection(in)
	if err != nil {
		return nil, err
	}
	return sel, nil
}

func (st *backupStore) deleteBackupSelection(_ *Server, in map[string]any) (any, *bedrockAgentError) {
	p, sel, err := st.selection(in)
	if err != nil {
		return nil, err
	}
	delete(p.selections, sel["SelectionId"].(string))
	return nil, nil
}

func (st *backupStore) listBackupSelections(_ *Server, in map[string]any) (any, *bedrockAgentError) {
	p, err := st.livePlan(bkStr(in, "BackupPlanId"))
	if err != nil {
		return nil, err
	}
	items := []map[string]any{}
	for _, s := range bkSorted(p.selections, bkSelf, "CreationDate") {
		item := bkShape(s, "BackupSelectionsListMember")
		item["SelectionName"] = bkMap(s, "BackupSelection")["SelectionName"]
		item["IamRoleArn"] = bkMap(s, "BackupSelection")["IamRoleArn"]
		items = append(items, item)
	}
	return bkPage(items, in, "BackupSelectionsList")
}
