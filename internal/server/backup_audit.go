package server

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	bkAuditNamePattern   = regexp.MustCompile(`^[a-zA-Z][_a-zA-Z0-9]{0,255}$`)
	bkTestingNamePattern = regexp.MustCompile(`^[a-zA-Z0-9_]{1,50}$`)
	bkTieringNamePattern = regexp.MustCompile(`^[a-zA-Z0-9_]{1,200}$`)
	// bkFrameworkControls are the controls AWS Backup Audit Manager offers.
	bkFrameworkControls = []string{
		"BACKUP_RECOVERY_POINT_MINIMUM_RETENTION_CHECK", "BACKUP_PLAN_MIN_FREQUENCY_AND_MIN_RETENTION_CHECK",
		"BACKUP_RECOVERY_POINT_ENCRYPTED", "BACKUP_RESOURCES_PROTECTED_BY_BACKUP_PLAN",
		"BACKUP_RECOVERY_POINT_MANUAL_DELETION_DISABLED", "BACKUP_RESOURCES_PROTECTED_BY_BACKUP_VAULT_LOCK",
		"BACKUP_LAST_RECOVERY_POINT_CREATED", "RESOURCES_IN_LOGICALLY_AIR_GAPPED_VAULT",
		"BACKUP_RESOURCES_PROTECTED_BY_CROSS_REGION", "BACKUP_RESOURCES_PROTECTED_BY_CROSS_ACCOUNT",
		"RESTORE_TIME_FOR_RESOURCES_MEET_TARGET",
	}
	bkReportTemplates = []string{"RESOURCE_COMPLIANCE_REPORT", "CONTROL_COMPLIANCE_REPORT", "BACKUP_JOB_REPORT", "COPY_JOB_REPORT", "RESTORE_JOB_REPORT", "SCAN_JOB_REPORT"}
)

// bkTestingPlan is a restore testing plan and its selections.
type bkTestingPlan struct {
	rec        map[string]any
	selections map[string]map[string]any
}

func bkAuditHandlers() map[string]bkHandler {
	return map[string]bkHandler{
		"CreateFramework":               (*backupStore).createFramework,
		"DescribeFramework":             (*backupStore).describeFramework,
		"UpdateFramework":               (*backupStore).updateFramework,
		"DeleteFramework":               (*backupStore).deleteFramework,
		"ListFrameworks":                (*backupStore).listFrameworks,
		"CreateReportPlan":              (*backupStore).createReportPlan,
		"DescribeReportPlan":            (*backupStore).describeReportPlan,
		"UpdateReportPlan":              (*backupStore).updateReportPlan,
		"DeleteReportPlan":              (*backupStore).deleteReportPlan,
		"ListReportPlans":               (*backupStore).listReportPlans,
		"StartReportJob":                (*backupStore).startReportJob,
		"DescribeReportJob":             (*backupStore).describeReportJob,
		"ListReportJobs":                (*backupStore).listReportJobs,
		"CreateRestoreTestingPlan":      (*backupStore).createTestingPlan,
		"GetRestoreTestingPlan":         (*backupStore).getTestingPlan,
		"UpdateRestoreTestingPlan":      (*backupStore).updateTestingPlan,
		"DeleteRestoreTestingPlan":      (*backupStore).deleteTestingPlan,
		"ListRestoreTestingPlans":       (*backupStore).listTestingPlans,
		"CreateRestoreTestingSelection": (*backupStore).createTestingSelection,
		"GetRestoreTestingSelection":    (*backupStore).getTestingSelection,
		"UpdateRestoreTestingSelection": (*backupStore).updateTestingSelection,
		"DeleteRestoreTestingSelection": (*backupStore).deleteTestingSelection,
		"ListRestoreTestingSelections":  (*backupStore).listTestingSelections,
		"CreateTieringConfiguration":    (*backupStore).createTiering,
		"GetTieringConfiguration":       (*backupStore).getTiering,
		"UpdateTieringConfiguration":    (*backupStore).updateTiering,
		"DeleteTieringConfiguration":    (*backupStore).deleteTiering,
		"ListTieringConfigurations":     (*backupStore).listTiering,
	}
}

// bkUnique is a short suffix that keeps ARNs of recreated names distinct.
func bkUnique() string { return strings.ReplaceAll(bkNewID(), "-", "")[:12] }

// --- frameworks ---

func bkCheckControls(in map[string]any) *bedrockAgentError {
	controls := bkList(in, "FrameworkControls")
	if len(controls) == 0 || len(controls) > 50 {
		return bkInvalid("FrameworkControls must list between 1 and 50 controls")
	}
	for _, c := range controls {
		control, _ := c.(map[string]any)
		if !containsString(bkFrameworkControls, bkStr(control, "ControlName")) {
			return bkInvalid("unknown framework control " + bkStr(control, "ControlName"))
		}
	}
	if len(bkStr(in, "FrameworkDescription")) > 1024 {
		return bkInvalid("FrameworkDescription must be at most 1024 characters")
	}
	return nil
}

func (st *backupStore) createFramework(_ *Server, in map[string]any) (any, *bedrockAgentError) {
	if err := bkRequire(in, "FrameworkName", "FrameworkControls"); err != nil {
		return nil, err
	}
	name := bkStr(in, "FrameworkName")
	if !bkAuditNamePattern.MatchString(name) {
		return nil, bkInvalid("FrameworkName must start with a letter and contain only letters, numbers and underscores")
	}
	if err := bkCheckControls(in); err != nil {
		return nil, err
	}
	if f := st.frameworks[name]; f != nil {
		if tok := bkStr(in, "IdempotencyToken"); tok != "" && f["IdempotencyToken"] == tok {
			return f, nil
		}
		return nil, bkExists("Framework", name)
	}
	if len(st.frameworks) >= backupMaxItems {
		return nil, bkErr("LimitExceededException", "too many frameworks")
	}
	arn := bkARN("framework", name+"-"+bkUnique())
	f := map[string]any{
		"FrameworkName": name, "FrameworkArn": arn, "FrameworkDescription": in["FrameworkDescription"],
		"FrameworkControls": in["FrameworkControls"], "CreationTime": bkNow(), "DeploymentStatus": "COMPLETED",
		"FrameworkStatus": "ACTIVE", "IdempotencyToken": in["IdempotencyToken"],
	}
	if err := st.setTags(arn, bkMap(in, "FrameworkTags")); err != nil {
		return nil, err
	}
	st.frameworks[name] = f
	return f, nil
}

func (st *backupStore) framework(name string) (map[string]any, *bedrockAgentError) {
	f := st.frameworks[name]
	if f == nil {
		return nil, bkNotFound("Framework", name)
	}
	return f, nil
}

func (st *backupStore) describeFramework(_ *Server, in map[string]any) (any, *bedrockAgentError) {
	return st.framework(bkStr(in, "FrameworkName"))
}

func (st *backupStore) updateFramework(_ *Server, in map[string]any) (any, *bedrockAgentError) {
	f, err := st.framework(bkStr(in, "FrameworkName"))
	if err != nil {
		return nil, err
	}
	if in["FrameworkControls"] != nil {
		if err := bkCheckControls(in); err != nil {
			return nil, err
		}
		f["FrameworkControls"] = in["FrameworkControls"]
	}
	if in["FrameworkDescription"] != nil {
		f["FrameworkDescription"] = in["FrameworkDescription"]
	}
	return f, nil
}

func (st *backupStore) deleteFramework(_ *Server, in map[string]any) (any, *bedrockAgentError) {
	f, err := st.framework(bkStr(in, "FrameworkName"))
	if err != nil {
		return nil, err
	}
	for _, rp := range st.reportPlans {
		if containsAny(bkList(bkMap(rp, "ReportSetting"), "FrameworkArns"), f["FrameworkArn"]) {
			return nil, bkErr("ConflictException", "framework "+bkStr(in, "FrameworkName")+" is used by report plan "+bkStr(rp, "ReportPlanName"))
		}
	}
	delete(st.tags, bkStr(f, "FrameworkArn"))
	delete(st.frameworks, bkStr(in, "FrameworkName"))
	return nil, nil
}

func (st *backupStore) listFrameworks(_ *Server, in map[string]any) (any, *bedrockAgentError) {
	items := []map[string]any{}
	for _, f := range bkSorted(st.frameworks, bkSelf, "CreationTime") {
		item := bkShape(f, "Framework")
		item["NumberOfControls"] = len(bkList(f, "FrameworkControls"))
		items = append(items, item)
	}
	return bkPage(items, in, "Frameworks")
}

// --- report plans and jobs ---

func (st *backupStore) checkReportPlan(in map[string]any) *bedrockAgentError {
	if ch := bkMap(in, "ReportDeliveryChannel"); ch != nil {
		if err := bkRequire(ch, "S3BucketName"); err != nil {
			return err
		}
		for _, f := range bkList(ch, "Formats") {
			if s, _ := f.(string); s != "CSV" && s != "JSON" {
				return bkInvalid("report Formats must be CSV or JSON")
			}
		}
	}
	setting := bkMap(in, "ReportSetting")
	if setting == nil {
		return nil
	}
	if !containsString(bkReportTemplates, bkStr(setting, "ReportTemplate")) {
		return bkInvalid("ReportTemplate must be one of " + strings.Join(bkReportTemplates, ", "))
	}
	frameworks := bkList(setting, "FrameworkArns")
	if t := bkStr(setting, "ReportTemplate"); (t == "CONTROL_COMPLIANCE_REPORT" || t == "RESOURCE_COMPLIANCE_REPORT") && len(frameworks) == 0 {
		return bkInvalid(t + " needs FrameworkArns")
	}
	for _, arn := range frameworks {
		found := false
		for _, f := range st.frameworks {
			found = found || f["FrameworkArn"] == arn
		}
		if !found {
			return bkNotFound("Framework", fmt.Sprint(arn))
		}
	}
	if len(bkStr(in, "ReportPlanDescription")) > 1024 {
		return bkInvalid("ReportPlanDescription must be at most 1024 characters")
	}
	return nil
}

func (st *backupStore) createReportPlan(_ *Server, in map[string]any) (any, *bedrockAgentError) {
	if err := bkRequire(in, "ReportPlanName", "ReportDeliveryChannel", "ReportSetting"); err != nil {
		return nil, err
	}
	name := bkStr(in, "ReportPlanName")
	if !bkAuditNamePattern.MatchString(name) {
		return nil, bkInvalid("ReportPlanName must start with a letter and contain only letters, numbers and underscores")
	}
	if err := st.checkReportPlan(in); err != nil {
		return nil, err
	}
	if p := st.reportPlans[name]; p != nil {
		if tok := bkStr(in, "IdempotencyToken"); tok != "" && p["IdempotencyToken"] == tok {
			return p, nil
		}
		return nil, bkExists("Report plan", name)
	}
	if len(st.reportPlans) >= backupMaxItems {
		return nil, bkErr("LimitExceededException", "too many report plans")
	}
	arn := bkARN("report-plan", name+"-"+bkUnique())
	setting := baClone(bkMap(in, "ReportSetting"))
	if n := len(bkList(setting, "FrameworkArns")); n > 0 {
		setting["NumberOfFrameworks"] = n
	}
	p := map[string]any{
		"ReportPlanName": name, "ReportPlanArn": arn, "ReportPlanDescription": in["ReportPlanDescription"],
		"ReportSetting": setting, "ReportDeliveryChannel": in["ReportDeliveryChannel"], "DeploymentStatus": "COMPLETED",
		"CreationTime": bkNow(), "IdempotencyToken": in["IdempotencyToken"],
	}
	if err := st.setTags(arn, bkMap(in, "ReportPlanTags")); err != nil {
		return nil, err
	}
	st.reportPlans[name] = p
	return p, nil
}

func (st *backupStore) reportPlan(name string) (map[string]any, *bedrockAgentError) {
	p := st.reportPlans[name]
	if p == nil {
		return nil, bkNotFound("Report plan", name)
	}
	return p, nil
}

func (st *backupStore) describeReportPlan(_ *Server, in map[string]any) (any, *bedrockAgentError) {
	p, err := st.reportPlan(bkStr(in, "ReportPlanName"))
	if err != nil {
		return nil, err
	}
	return map[string]any{"ReportPlan": bkShape(p, "ReportPlan")}, nil
}

func (st *backupStore) updateReportPlan(_ *Server, in map[string]any) (any, *bedrockAgentError) {
	p, err := st.reportPlan(bkStr(in, "ReportPlanName"))
	if err != nil {
		return nil, err
	}
	if err := st.checkReportPlan(in); err != nil {
		return nil, err
	}
	for _, k := range []string{"ReportPlanDescription", "ReportDeliveryChannel", "ReportSetting"} {
		if in[k] != nil {
			p[k] = in[k]
		}
	}
	return p, nil
}

func (st *backupStore) deleteReportPlan(_ *Server, in map[string]any) (any, *bedrockAgentError) {
	p, err := st.reportPlan(bkStr(in, "ReportPlanName"))
	if err != nil {
		return nil, err
	}
	delete(st.tags, bkStr(p, "ReportPlanArn"))
	delete(st.reportPlans, bkStr(in, "ReportPlanName"))
	return nil, nil
}

func (st *backupStore) listReportPlans(_ *Server, in map[string]any) (any, *bedrockAgentError) {
	items := []map[string]any{}
	for _, p := range bkSorted(st.reportPlans, bkSelf, "CreationTime") {
		items = append(items, bkShape(p, "ReportPlan"))
	}
	return bkPage(items, in, "ReportPlans")
}

func (st *backupStore) startReportJob(_ *Server, in map[string]any) (any, *bedrockAgentError) {
	p, err := st.reportPlan(bkStr(in, "ReportPlanName"))
	if err != nil {
		return nil, err
	}
	if j := st.idempotent(bkJobReport, in); j != nil {
		return j.rec, nil
	}
	now := bkNow()
	ch := bkMap(p, "ReportDeliveryChannel")
	prefix := strings.Trim(bkStr(ch, "S3KeyPrefix"), "/")
	if prefix != "" {
		prefix += "/"
	}
	format := "csv"
	if f := bkList(ch, "Formats"); len(f) > 0 {
		format = strings.ToLower(f[0].(string))
	}
	j := &bkJob{kind: bkJobReport, stateKey: "Status", steps: []string{"CREATED", "RUNNING", "COMPLETED"}, rec: map[string]any{
		"ReportPlanArn": p["ReportPlanArn"], "ReportTemplate": bkMap(p, "ReportSetting")["ReportTemplate"],
		"CreationTime": now, "Status": "CREATED", "ReportPlanName": p["ReportPlanName"],
	}}
	id, err := st.addJob(j, "ReportJobId")
	if err != nil {
		return nil, err
	}
	j.rec["ReportDestination"] = map[string]any{"S3BucketName": ch["S3BucketName"], "S3Keys": []any{prefix + "Backup/" + backupAccount + "/" + backupRegion + "/" + id + "." + format}}
	p["LastAttemptedExecutionTime"] = now
	p["LastSuccessfulExecutionTime"] = now
	st.remember(bkJobReport, in, id)
	return j.rec, nil
}

func (st *backupStore) describeReportJob(s *Server, in map[string]any) (any, *bedrockAgentError) {
	j, err := st.job(bkJobReport, bkStr(in, "ReportJobId"))
	if err != nil {
		return nil, err
	}
	st.tick(s, j)
	return map[string]any{"ReportJob": bkShape(j.rec, "ReportJob")}, nil
}

func (st *backupStore) listReportJobs(s *Server, in map[string]any) (any, *bedrockAgentError) {
	if err := bkEnum(in, "ByStatus", "CREATED", "RUNNING", "COMPLETED", "FAILED"); err != nil {
		return nil, err
	}
	filters := []bkFilter{
		{"ByReportPlanName", "ReportPlanName", '='}, {"ByStatus", "Status", '='},
		{"ByCreationBefore", "CreationTime", '<'}, {"ByCreationAfter", "CreationTime", '>'},
	}
	return st.listJobs(s, bkJobReport, in, filters, nil, "ReportJob", "ReportJobs")
}

// --- restore testing ---

func bkCheckTestingPlan(plan map[string]any, creating bool) *bedrockAgentError {
	if creating {
		if err := bkRequire(plan, "RestoreTestingPlanName", "ScheduleExpression", "RecoveryPointSelection"); err != nil {
			return err
		}
		if !bkTestingNamePattern.MatchString(bkStr(plan, "RestoreTestingPlanName")) {
			return bkInvalid("RestoreTestingPlanName must be 1-50 letters, numbers or underscores")
		}
	}
	if sched := bkStr(plan, "ScheduleExpression"); sched != "" && !bkSchedulePattern.MatchString(sched) {
		return bkInvalid("ScheduleExpression must be a cron() or rate() expression")
	}
	if n, absent, err := bkInt(plan, "StartWindowHours"); err != nil {
		return err
	} else if !absent && (n < 1 || n > 168) {
		return bkInvalid("StartWindowHours must be between 1 and 168")
	}
	sel := bkMap(plan, "RecoveryPointSelection")
	if sel == nil {
		return nil
	}
	if err := bkRequire(sel, "Algorithm", "IncludeVaults", "RecoveryPointTypes"); err != nil {
		return err
	}
	if err := bkEnum(sel, "Algorithm", "LATEST_WITHIN_WINDOW", "RANDOM_WITHIN_WINDOW"); err != nil {
		return err
	}
	types := bkList(sel, "RecoveryPointTypes")
	if len(types) == 0 || len(types) > 2 {
		return bkInvalid("RecoveryPointTypes must list CONTINUOUS and/or SNAPSHOT")
	}
	for _, t := range types {
		if s, _ := t.(string); s != "CONTINUOUS" && s != "SNAPSHOT" {
			return bkInvalid("RecoveryPointTypes must list CONTINUOUS and/or SNAPSHOT")
		}
	}
	if len(bkList(sel, "IncludeVaults")) == 0 {
		return bkInvalid("IncludeVaults must list at least one vault ARN or \"*\"")
	}
	if n, absent, err := bkInt(sel, "SelectionWindowDays"); err != nil {
		return err
	} else if !absent && (n < 1 || n > 365) {
		return bkInvalid("SelectionWindowDays must be between 1 and 365")
	}
	return nil
}

func (st *backupStore) testingPlan(name string) (*bkTestingPlan, *bedrockAgentError) {
	p := st.testingPlans[name]
	if p == nil {
		return nil, bkNotFound("Restore testing plan", name)
	}
	return p, nil
}

func (st *backupStore) createTestingPlan(_ *Server, in map[string]any) (any, *bedrockAgentError) {
	if err := bkRequire(in, "RestoreTestingPlan"); err != nil {
		return nil, err
	}
	plan := bkMap(in, "RestoreTestingPlan")
	if err := bkCheckTestingPlan(plan, true); err != nil {
		return nil, err
	}
	name := bkStr(plan, "RestoreTestingPlanName")
	if p := st.testingPlans[name]; p != nil {
		if tok := bkStr(in, "CreatorRequestId"); tok != "" && p.rec["CreatorRequestId"] == tok {
			return p.rec, nil
		}
		return nil, bkExists("Restore testing plan", name)
	}
	if len(st.testingPlans) >= backupMaxItems {
		return nil, bkErr("LimitExceededException", "too many restore testing plans")
	}
	rec := baClone(plan)
	rec["RestoreTestingPlanArn"] = bkARN("restore-testing-plan", name+"-"+bkUnique())
	rec["CreationTime"] = bkNow()
	rec["CreatorRequestId"] = in["CreatorRequestId"]
	if bkStr(rec, "ScheduleExpressionTimezone") == "" {
		rec["ScheduleExpressionTimezone"] = "Etc/UTC"
	}
	if rec["StartWindowHours"] == nil {
		rec["StartWindowHours"] = 24
	}
	if err := st.setTags(bkStr(rec, "RestoreTestingPlanArn"), bkMap(in, "Tags")); err != nil {
		return nil, err
	}
	st.testingPlans[name] = &bkTestingPlan{rec: rec, selections: map[string]map[string]any{}}
	return rec, nil
}

func (st *backupStore) getTestingPlan(_ *Server, in map[string]any) (any, *bedrockAgentError) {
	p, err := st.testingPlan(bkStr(in, "RestoreTestingPlanName"))
	if err != nil {
		return nil, err
	}
	return map[string]any{"RestoreTestingPlan": bkShape(p.rec, "RestoreTestingPlanForGet")}, nil
}

func (st *backupStore) updateTestingPlan(_ *Server, in map[string]any) (any, *bedrockAgentError) {
	p, err := st.testingPlan(bkStr(in, "RestoreTestingPlanName"))
	if err != nil {
		return nil, err
	}
	if err := bkRequire(in, "RestoreTestingPlan"); err != nil {
		return nil, err
	}
	plan := bkMap(in, "RestoreTestingPlan")
	if err := bkCheckTestingPlan(plan, false); err != nil {
		return nil, err
	}
	for _, k := range []string{"ScheduleExpression", "ScheduleExpressionTimezone", "StartWindowHours", "RecoveryPointSelection"} {
		if plan[k] != nil {
			p.rec[k] = plan[k]
		}
	}
	p.rec["LastUpdateTime"] = bkNow()
	return map[string]any{"CreationTime": p.rec["CreationTime"], "RestoreTestingPlanArn": p.rec["RestoreTestingPlanArn"], "RestoreTestingPlanName": p.rec["RestoreTestingPlanName"], "UpdateTime": p.rec["LastUpdateTime"]}, nil
}

func (st *backupStore) deleteTestingPlan(_ *Server, in map[string]any) (any, *bedrockAgentError) {
	p, err := st.testingPlan(bkStr(in, "RestoreTestingPlanName"))
	if err != nil {
		return nil, err
	}
	if len(p.selections) > 0 {
		return nil, bkRequest("delete the restore testing selections of plan " + bkStr(in, "RestoreTestingPlanName") + " first")
	}
	delete(st.tags, bkStr(p.rec, "RestoreTestingPlanArn"))
	delete(st.testingPlans, bkStr(in, "RestoreTestingPlanName"))
	return nil, nil
}

func (st *backupStore) listTestingPlans(_ *Server, in map[string]any) (any, *bedrockAgentError) {
	items := []map[string]any{}
	for _, p := range bkSorted(st.testingPlans, func(p *bkTestingPlan) map[string]any { return p.rec }, "CreationTime") {
		items = append(items, bkShape(p.rec, "RestoreTestingPlanForList"))
	}
	return bkPage(items, in, "RestoreTestingPlans")
}

func bkCheckTestingSelection(sel map[string]any, creating bool) *bedrockAgentError {
	if creating {
		if err := bkRequire(sel, "RestoreTestingSelectionName", "ProtectedResourceType", "IamRoleArn"); err != nil {
			return err
		}
		if !bkTestingNamePattern.MatchString(bkStr(sel, "RestoreTestingSelectionName")) {
			return bkInvalid("RestoreTestingSelectionName must be 1-50 letters, numbers or underscores")
		}
		if !containsString(bkResourceTypes, bkStr(sel, "ProtectedResourceType")) {
			return bkInvalid("unsupported ProtectedResourceType " + bkStr(sel, "ProtectedResourceType"))
		}
	}
	if err := bkCheckRole(sel, "IamRoleArn"); err != nil {
		return err
	}
	arns, conds := bkList(sel, "ProtectedResourceArns"), bkMap(sel, "ProtectedResourceConditions")
	if len(arns) > 0 && conds != nil {
		return bkInvalid("specify ProtectedResourceArns or ProtectedResourceConditions, not both")
	}
	if len(arns) > 30 {
		return bkInvalid("ProtectedResourceArns can list at most 30 resources")
	}
	for _, a := range arns {
		if s, _ := a.(string); s != "*" && !strings.HasPrefix(s, "arn:") {
			return bkInvalid("ProtectedResourceArns entries must be ARNs or \"*\"")
		}
	}
	if n, absent, err := bkInt(sel, "ValidationWindowHours"); err != nil {
		return err
	} else if !absent && (n < 0 || n > 168) {
		return bkInvalid("ValidationWindowHours must be between 0 and 168")
	}
	return nil
}

func (st *backupStore) createTestingSelection(_ *Server, in map[string]any) (any, *bedrockAgentError) {
	p, err := st.testingPlan(bkStr(in, "RestoreTestingPlanName"))
	if err != nil {
		return nil, err
	}
	if err := bkRequire(in, "RestoreTestingSelection"); err != nil {
		return nil, err
	}
	sel := bkMap(in, "RestoreTestingSelection")
	if err := bkCheckTestingSelection(sel, true); err != nil {
		return nil, err
	}
	name := bkStr(sel, "RestoreTestingSelectionName")
	if cur := p.selections[name]; cur != nil {
		if tok := bkStr(in, "CreatorRequestId"); tok != "" && cur["CreatorRequestId"] == tok {
			return st.testingSelectionOutput(p, cur), nil
		}
		return nil, bkExists("Restore testing selection", name)
	}
	if len(p.selections) >= 30 {
		return nil, bkErr("LimitExceededException", "a restore testing plan can have at most 30 selections")
	}
	rec := baClone(sel)
	rec["RestoreTestingPlanName"] = p.rec["RestoreTestingPlanName"]
	rec["CreationTime"] = bkNow()
	rec["CreatorRequestId"] = in["CreatorRequestId"]
	if rec["ValidationWindowHours"] == nil {
		rec["ValidationWindowHours"] = 0
	}
	p.selections[name] = rec
	return st.testingSelectionOutput(p, rec), nil
}

func (st *backupStore) testingSelectionOutput(p *bkTestingPlan, sel map[string]any) map[string]any {
	return map[string]any{
		"CreationTime": sel["CreationTime"], "RestoreTestingPlanArn": p.rec["RestoreTestingPlanArn"],
		"RestoreTestingPlanName": p.rec["RestoreTestingPlanName"], "RestoreTestingSelectionName": sel["RestoreTestingSelectionName"],
		"UpdateTime": sel["UpdateTime"],
	}
}

func (st *backupStore) testingSelection(in map[string]any) (*bkTestingPlan, map[string]any, *bedrockAgentError) {
	p, err := st.testingPlan(bkStr(in, "RestoreTestingPlanName"))
	if err != nil {
		return nil, nil, err
	}
	sel := p.selections[bkStr(in, "RestoreTestingSelectionName")]
	if sel == nil {
		return nil, nil, bkNotFound("Restore testing selection", bkStr(in, "RestoreTestingSelectionName"))
	}
	return p, sel, nil
}

func (st *backupStore) getTestingSelection(_ *Server, in map[string]any) (any, *bedrockAgentError) {
	_, sel, err := st.testingSelection(in)
	if err != nil {
		return nil, err
	}
	return map[string]any{"RestoreTestingSelection": bkShape(sel, "RestoreTestingSelectionForGet")}, nil
}

func (st *backupStore) updateTestingSelection(_ *Server, in map[string]any) (any, *bedrockAgentError) {
	p, sel, err := st.testingSelection(in)
	if err != nil {
		return nil, err
	}
	if err := bkRequire(in, "RestoreTestingSelection"); err != nil {
		return nil, err
	}
	upd := bkMap(in, "RestoreTestingSelection")
	merged := baClone(sel)
	for _, k := range []string{"IamRoleArn", "ProtectedResourceArns", "ProtectedResourceConditions", "RestoreMetadataOverrides", "ValidationWindowHours"} {
		if upd[k] != nil {
			merged[k] = upd[k]
		}
	}
	if upd["ProtectedResourceArns"] != nil {
		delete(merged, "ProtectedResourceConditions")
	}
	if upd["ProtectedResourceConditions"] != nil {
		delete(merged, "ProtectedResourceArns")
	}
	if err := bkCheckTestingSelection(merged, false); err != nil {
		return nil, err
	}
	merged["CreationTime"] = sel["CreationTime"]
	merged["UpdateTime"] = bkNow()
	p.selections[bkStr(sel, "RestoreTestingSelectionName")] = merged
	return st.testingSelectionOutput(p, merged), nil
}

func (st *backupStore) deleteTestingSelection(_ *Server, in map[string]any) (any, *bedrockAgentError) {
	p, sel, err := st.testingSelection(in)
	if err != nil {
		return nil, err
	}
	delete(p.selections, bkStr(sel, "RestoreTestingSelectionName"))
	return nil, nil
}

func (st *backupStore) listTestingSelections(_ *Server, in map[string]any) (any, *bedrockAgentError) {
	p, err := st.testingPlan(bkStr(in, "RestoreTestingPlanName"))
	if err != nil {
		return nil, err
	}
	items := []map[string]any{}
	for _, sel := range bkSorted(p.selections, func(m map[string]any) map[string]any { return m }, "CreationTime") {
		items = append(items, bkShape(sel, "RestoreTestingSelectionForList"))
	}
	return bkPage(items, in, "RestoreTestingSelections")
}

// --- tiering configurations ---

func (st *backupStore) checkTiering(cfg map[string]any) *bedrockAgentError {
	if err := bkRequire(cfg, "BackupVaultName", "ResourceSelection"); err != nil {
		return err
	}
	if v := bkStr(cfg, "BackupVaultName"); v != "*" && st.vaults[v] == nil {
		return bkNotFound("Backup vault", v)
	}
	selections := bkList(cfg, "ResourceSelection")
	if len(selections) == 0 || len(selections) > 5 {
		return bkInvalid("ResourceSelection must list between 1 and 5 selections")
	}
	for _, r := range selections {
		sel, _ := r.(map[string]any)
		if err := bkRequire(sel, "Resources", "TieringDownSettingsInDays", "ResourceType"); err != nil {
			return err
		}
		if bkStr(sel, "ResourceType") != "S3" {
			return bkInvalid("tiering only supports ResourceType S3")
		}
		if days, _, err := bkInt(sel, "TieringDownSettingsInDays"); err != nil {
			return err
		} else if days < 60 || days > bkMaxRetentionDays {
			return bkInvalid("TieringDownSettingsInDays must be between 60 and 36500")
		}
		if n := len(bkList(sel, "Resources")); n == 0 || n > 100 {
			return bkInvalid("Resources must list between 1 and 100 resources")
		}
	}
	return nil
}

func (st *backupStore) createTiering(_ *Server, in map[string]any) (any, *bedrockAgentError) {
	if err := bkRequire(in, "TieringConfiguration"); err != nil {
		return nil, err
	}
	cfg := bkMap(in, "TieringConfiguration")
	if err := bkRequire(cfg, "TieringConfigurationName"); err != nil {
		return nil, err
	}
	name := bkStr(cfg, "TieringConfigurationName")
	if !bkTieringNamePattern.MatchString(name) {
		return nil, bkInvalid("TieringConfigurationName must be 1-200 letters, numbers or underscores")
	}
	if err := st.checkTiering(cfg); err != nil {
		return nil, err
	}
	if cur := st.tiering[name]; cur != nil {
		if tok := bkStr(in, "CreatorRequestId"); tok != "" && cur["CreatorRequestId"] == tok {
			return cur, nil
		}
		return nil, bkExists("Tiering configuration", name)
	}
	if len(st.tiering) >= backupMaxItems {
		return nil, bkErr("LimitExceededException", "too many tiering configurations")
	}
	now := bkNow()
	rec := baClone(cfg)
	rec["TieringConfigurationArn"] = bkARN("tiering-configuration", name+"-"+bkUnique())
	rec["CreationTime"], rec["LastUpdatedTime"] = now, now
	rec["CreatorRequestId"] = in["CreatorRequestId"]
	if err := st.setTags(bkStr(rec, "TieringConfigurationArn"), bkMap(in, "TieringConfigurationTags")); err != nil {
		return nil, err
	}
	st.tiering[name] = rec
	return rec, nil
}

func (st *backupStore) tieringConfig(name string) (map[string]any, *bedrockAgentError) {
	t := st.tiering[name]
	if t == nil {
		return nil, bkNotFound("Tiering configuration", name)
	}
	return t, nil
}

func (st *backupStore) getTiering(_ *Server, in map[string]any) (any, *bedrockAgentError) {
	t, err := st.tieringConfig(bkStr(in, "TieringConfigurationName"))
	if err != nil {
		return nil, err
	}
	return map[string]any{"TieringConfiguration": bkShape(t, "TieringConfiguration")}, nil
}

func (st *backupStore) updateTiering(_ *Server, in map[string]any) (any, *bedrockAgentError) {
	t, err := st.tieringConfig(bkStr(in, "TieringConfigurationName"))
	if err != nil {
		return nil, err
	}
	if err := bkRequire(in, "TieringConfiguration"); err != nil {
		return nil, err
	}
	cfg := bkMap(in, "TieringConfiguration")
	if err := st.checkTiering(cfg); err != nil {
		return nil, err
	}
	t["BackupVaultName"] = cfg["BackupVaultName"]
	t["ResourceSelection"] = cfg["ResourceSelection"]
	t["LastUpdatedTime"] = bkNow()
	return t, nil
}

func (st *backupStore) deleteTiering(_ *Server, in map[string]any) (any, *bedrockAgentError) {
	t, err := st.tieringConfig(bkStr(in, "TieringConfigurationName"))
	if err != nil {
		return nil, err
	}
	delete(st.tags, bkStr(t, "TieringConfigurationArn"))
	delete(st.tiering, bkStr(in, "TieringConfigurationName"))
	return nil, nil
}

func (st *backupStore) listTiering(_ *Server, in map[string]any) (any, *bedrockAgentError) {
	items := []map[string]any{}
	for _, t := range bkSorted(st.tiering, bkSelf, "CreationTime") {
		items = append(items, bkShape(t, "TieringConfigurationsListMember"))
	}
	return bkPage(items, in, "TieringConfigurations")
}
