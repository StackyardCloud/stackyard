package server

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Interactive sessions and statements, Data Quality, column-statistics and
// materialized-view task runs, table optimizers, integrations, glossaries and
// assets, connection types, tags, resource policies and the code-generation
// helpers.

func init() {
	glueKinds["dqannotation"] = glueKind{fields: []string{"ProfileId", "StatisticId"}}
	registerGlueHandlers(map[string]glueHandler{
		"CreateSession":                           glueCreateSession,
		"StopSession":                             glueStopSession,
		"ListSessions":                            glueListSessions,
		"RunStatement":                            glueRunStatement,
		"CancelStatement":                         glueCancelStatement,
		"GetStatement":                            glueAdvanceThen("GetStatement", func(st *glueStore, in glueRecord) { st.advanceStatements(in["SessionId"]) }),
		"ListStatements":                          glueAdvanceThen("ListStatements", func(st *glueStore, in glueRecord) { st.advanceStatements(in["SessionId"]) }),
		"GetSessionEndpoint":                      glueGetSessionEndpoint,
		"GetSystemLogsForSession":                 glueGetSystemLogsForSession,
		"StartDataQualityRulesetEvaluationRun":    glueStartDQEvaluation,
		"StartDataQualityRuleRecommendationRun":   glueStartDQRecommendation,
		"CancelDataQualityRulesetEvaluationRun":   glueCancelRun("dqevalrun"),
		"CancelDataQualityRuleRecommendationRun":  glueCancelRun("dqrecrun"),
		"GetDataQualityRulesetEvaluationRun":      glueAdvanceThen("GetDataQualityRulesetEvaluationRun", (*glueStore).advanceDQ),
		"ListDataQualityRulesetEvaluationRuns":    glueAdvanceThen("ListDataQualityRulesetEvaluationRuns", (*glueStore).advanceDQ),
		"BatchGetDataQualityRulesetEvaluationRun": glueAdvanceThen("BatchGetDataQualityRulesetEvaluationRun", (*glueStore).advanceDQ),
		"GetDataQualityRuleRecommendationRun":     glueAdvanceThen("GetDataQualityRuleRecommendationRun", (*glueStore).advanceDQ),
		"ListDataQualityRuleRecommendationRuns":   glueAdvanceThen("ListDataQualityRuleRecommendationRuns", (*glueStore).advanceDQ),
		"PublishDataQualityResultV2":              gluePublishDQResult,
		"CreateDataQualityRuleset":                glueCreateDQRuleset,
		"GetDataQualityModel":                     glueGetDQModel,
		"GetDataQualityModelResult":               glueGetDQModelResult,
		"PutDataQualityProfileAnnotation":         gluePutDQAnnotation,
		"BatchPutDataQualityStatisticAnnotation":  glueBatchPutDQAnnotation,
		"ListDataQualityStatisticAnnotations":     glueListDQAnnotations,
		"ListDataQualityStatistics": func(st *glueStore, in glueRecord) (glueRecord, *glueError) {
			// ponytail: evaluation runs record no column statistics; add when profiles are modelled.
			return glueRecord{"Statistics": []any{}}, nil
		},
		"CreateDataQualityEvaluationConfig":    glueCreateDQEvaluationConfig,
		"StartColumnStatisticsTaskRun":         glueStartColumnStatsRun,
		"StopColumnStatisticsTaskRun":          glueStopColumnStatsRun,
		"GetColumnStatisticsTaskRun":           glueAdvanceThen("GetColumnStatisticsTaskRun", (*glueStore).advanceColumnStats),
		"GetColumnStatisticsTaskRuns":          glueAdvanceThen("GetColumnStatisticsTaskRuns", (*glueStore).advanceColumnStats),
		"StartColumnStatisticsTaskRunSchedule": glueColumnStatsSchedule(true),
		"StopColumnStatisticsTaskRunSchedule":  glueColumnStatsSchedule(false),
		"StartMaterializedViewRefreshTaskRun":  glueStartMVRefresh,
		"StopMaterializedViewRefreshTaskRun":   glueStopMVRefresh,
		"GetMaterializedViewRefreshTaskRun":    glueAdvanceThen("GetMaterializedViewRefreshTaskRun", (*glueStore).advanceMVRefresh),
		"ListMaterializedViewRefreshTaskRuns":  glueAdvanceThen("ListMaterializedViewRefreshTaskRuns", (*glueStore).advanceMVRefresh),
		"GetTableOptimizer":                    glueGetTableOptimizer,
		"BatchGetTableOptimizer":               glueBatchGetTableOptimizer,
		"ListTableOptimizerRuns":               glueListTableOptimizerRuns,
		"CreateIntegration":                    glueCreateIntegration,
		"ModifyIntegration":                    glueModifyIntegration,
		"DeleteIntegration":                    glueDeleteIntegration,
		"DescribeIntegrations":                 glueDescribeIntegrations,
		"DescribeInboundIntegrations":          glueDescribeInboundIntegrations,
		"CreateIntegrationResourceProperty":    glueCreateIntegrationResourceProperty,
		"CreateGlossaryTerm":                   glueCreateGlossaryTerm,
		"ListGlossaryTerms":                    glueListGlossaryTerms,
		"DeleteGlossary":                       glueDeleteGlossary,
		"AssociateGlossaryTerms":               glueGlossaryTerms(true),
		"DisassociateGlossaryTerms":            glueGlossaryTerms(false),
		"PutAsset":                             gluePutAsset,
		"PutAssetType":                         gluePutNamed("assettype"),
		"PutFormType":                          gluePutNamed("formtype"),
		"PutAttachment":                        gluePutAttachment,
		"ListIterableForms":                    glueListIterableForms,
		"BatchGetIterableForms":                glueBatchGetIterableForms,
		"RegisterConnectionType":               glueRegisterConnectionType,
		"DeleteConnectionType": func(st *glueStore, in glueRecord) (glueRecord, *glueError) {
			if glueIsBuiltinConnectionType(in["ConnectionType"]) {
				return nil, glueInvalid("connection type %v is built in and can't be deleted", in["ConnectionType"])
			}
			return st.generic(glueOps["DeleteConnectionType"], in)
		},
		"ListEntities":                          glueListEntities,
		"DescribeEntity":                        glueConnectionScoped("Fields"),
		"GetEntityRecords":                      glueConnectionScoped("Records"),
		"CreateGlueIdentityCenterConfiguration": glueCreateIdentityCenter,
		"GetGlueIdentityCenterConfiguration":    glueSingleGet("identitycenter", ""),
		"UpdateGlueIdentityCenterConfiguration": glueUpdateIdentityCenter,
		"DeleteGlueIdentityCenterConfiguration": glueDeleteIdentityCenter,
		"TagResource":                           glueTagResource,
		"UntagResource":                         glueUntagResource,
		"GetTags":                               glueGetTags,
		"PutResourcePolicy":                     gluePutResourcePolicy,
		"GetResourcePolicy":                     glueGetResourcePolicy,
		"DeleteResourcePolicy":                  glueDeleteResourcePolicy,
		"GetResourcePolicies":                   glueGetResourcePolicies,
		"CreateScript":                          glueCreateScript,
		"GetPlan":                               glueGetPlan,
		"GetDataflowGraph":                      glueGetDataflowGraph,
		"GetMapping":                            glueGetMapping,
		"GetDashboardUrl":                       glueGetDashboardURL,
	})
}

// glueBuiltinConnectionTypes are seeded into every store and can't be
// registered or deleted.
var glueBuiltinConnectionTypes = []glueRecord{
	{"ConnectionType": "JDBC", "DisplayName": "JDBC", "Vendor": "AWS", "Description": "Java Database Connectivity data sources", "Categories": []any{"Database"}},
	{"ConnectionType": "KAFKA", "DisplayName": "Apache Kafka", "Vendor": "AWS", "Description": "Apache Kafka and Amazon MSK clusters", "Categories": []any{"Streaming"}},
	{"ConnectionType": "MONGODB", "DisplayName": "MongoDB", "Vendor": "AWS", "Description": "MongoDB and Amazon DocumentDB", "Categories": []any{"Database"}},
	{"ConnectionType": "NETWORK", "DisplayName": "Network", "Vendor": "AWS", "Description": "A VPC network connection", "Categories": []any{"Network"}},
}

func glueIsBuiltinConnectionType(name any) bool {
	for _, b := range glueBuiltinConnectionTypes {
		if b["ConnectionType"] == name {
			return true
		}
	}
	return false
}

// Interactive sessions. A session is PROVISIONING in the create response and
// READY afterwards; statements report RUNNING once, then AVAILABLE.

const glueMaxStatementsPerSession = 1000

func glueCreateSession(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	rec, err := st.createFrom(glueOp{kind: "session"}, in)
	if err != nil {
		return nil, err
	}
	rec["Status"], rec["_next"] = "READY", float64(0)
	view := glueWrap(rec, "")
	view["Status"] = "PROVISIONING"
	return glueRecord{"Session": view}, nil
}

func glueStopSession(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	rec, err := st.find("session", []any{in["Id"]})
	if err != nil {
		return nil, err
	}
	if rec["Status"] == "STOPPED" || rec["Status"] == "STOPPING" {
		return nil, glueErr("IllegalSessionStateException", "session %v is already %v", in["Id"], rec["Status"])
	}
	rec["Status"], rec["CompletedOn"] = "STOPPED", glueEpoch(time.Now())
	return glueRecord{"Id": in["Id"]}, nil
}

func glueListSessions(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	page, err := st.list(glueOp{kind: "session", verb: "list", out: "Sessions"}, in)
	if err != nil {
		return nil, err
	}
	var ids []any
	for _, s := range glueList(page["Sessions"]) {
		ids = append(ids, s.(glueRecord)["Id"])
	}
	page["Ids"] = glueNonNil(ids)
	return page, nil
}

func glueRunStatement(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	session, err := st.find("session", []any{in["SessionId"]})
	if err != nil {
		return nil, err
	}
	if session["Status"] != "READY" {
		return nil, glueErr("IllegalSessionStateException", "session %v is %v, not READY", in["SessionId"], session["Status"])
	}
	next, _ := session["_next"].(float64)
	if next >= glueMaxStatementsPerSession {
		return nil, glueErr("ResourceNumberLimitExceededException", "session %v has run %d statements", in["SessionId"], glueMaxStatementsPerSession)
	}
	session["_next"] = next + 1
	_, err = st.create("statement", glueRecord{"SessionId": in["SessionId"], "Id": next, "Code": in["Code"], "State": "RUNNING", "Progress": 0.0}, nil)
	if err != nil {
		return nil, err
	}
	return glueRecord{"Id": next}, nil
}

func (st *glueStore) advanceStatements(sessionID any) {
	for _, s := range st.all("statement") {
		if s["SessionId"] != sessionID || !glueProgress(s, "State", map[string]string{"RUNNING": "AVAILABLE", "CANCELLING": "CANCELLED"}) {
			continue
		}
		s["CompletedOn"], s["Progress"] = glueEpoch(time.Now()), 1.0
		if s["State"] == "AVAILABLE" {
			s["Output"] = glueRecord{"Status": "ok", "ExecutionCount": s["Id"].(float64) + 1, "Data": glueRecord{"TextPlain": ""}}
		}
	}
}

func glueCancelStatement(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	s, err := st.find("statement", []any{in["SessionId"], in["Id"]})
	if err != nil {
		return nil, err
	}
	if s["State"] != "RUNNING" && s["State"] != "WAITING" {
		return nil, glueErr("IllegalSessionStateException", "statement %v is %v", in["Id"], s["State"])
	}
	s["State"], s["_reads"] = "CANCELLING", 0
	return glueRecord{}, nil
}

func glueGetSessionEndpoint(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	s, err := st.find("session", []any{in["SessionId"]})
	if err != nil {
		return nil, err
	}
	if s["Status"] != "READY" {
		return nil, glueErr("IllegalSessionStateException", "session %v is %v, not READY", in["SessionId"], s["Status"])
	}
	return glueRecord{"SparkConnect": glueRecord{
		"Url": fmt.Sprintf("sc://glue-%v.%s.example.internal:443", in["SessionId"], glueRegion), "AuthToken": st.newID("tok-"),
		"AuthTokenExpirationTime": glueEpoch(time.Now().Add(time.Hour)),
	}}, nil
}

func glueGetSystemLogsForSession(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	if _, err := st.find("session", []any{in["Id"]}); err != nil {
		return nil, err
	}
	return glueRecord{"SystemLogsUrl": fmt.Sprintf("https://console.aws.amazon.com/cloudwatch/home?region=%s#logsV2:log-groups/log-group/$252Faws-glue$252Fsessions/log-events/%v", glueRegion, in["Id"])}, nil
}

// Data Quality.

var glueDQRule = regexp.MustCompile(`(?m)^\s*([A-Za-z][A-Za-z0-9]*)\b[^,\]]*`)

func glueDQRules(ruleset string) []string {
	body := ruleset
	if i := strings.Index(body, "["); i >= 0 {
		body = body[i+1:]
	}
	if i := strings.LastIndex(body, "]"); i >= 0 {
		body = body[:i]
	}
	var rules []string
	for _, part := range strings.Split(body, ",") {
		if r := strings.TrimSpace(part); r != "" {
			rules = append(rules, r)
		}
	}
	return rules
}

func glueCreateDQRuleset(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	if !strings.Contains(glueStr(in, "Ruleset"), "Rules") || !glueDQRule.MatchString(glueStr(in, "Ruleset")) {
		return nil, glueInvalid("Ruleset must be DQDL, e.g. Rules = [ RowCount > 0 ]")
	}
	rec, err := st.createFrom(glueOps["CreateDataQualityRuleset"], in)
	if err != nil {
		return nil, err
	}
	rec["RuleCount"] = float64(len(glueDQRules(glueStr(in, "Ruleset"))))
	return glueRecord{"Name": rec["Name"]}, nil
}

func glueCreateDQEvaluationConfig(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	if rs := in["RulesetName"]; rs != nil {
		if _, err := st.find("dqruleset", []any{rs}); err != nil {
			return nil, err
		}
	}
	rec, err := st.createFrom(glueOps["CreateDataQualityEvaluationConfig"], in)
	if err != nil {
		return nil, err
	}
	rec["EvaluationRunMode"], rec["PauseRuns"] = "SCHEDULED", false
	if in["ScheduleExpression"] == nil {
		rec["EvaluationRunMode"] = "ON_DEMAND"
	}
	return glueRecord{"Name": rec["Name"], "EvaluationRunMode": rec["EvaluationRunMode"]}, nil
}

func glueStartDQEvaluation(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	for _, n := range glueList(in["RulesetNames"]) {
		if _, err := st.find("dqruleset", []any{n}); err != nil {
			return nil, err
		}
	}
	rec, err := st.create("dqevalrun", glueRecord{
		"DataSource": in["DataSource"], "Role": in["Role"], "RulesetNames": in["RulesetNames"], "Status": "RUNNING",
		"NumberOfWorkers": in["NumberOfWorkers"], "Timeout": in["Timeout"], "AdditionalRunOptions": in["AdditionalRunOptions"],
		"AdditionalDataSources": in["AdditionalDataSources"], "ResultIds": []any{},
	}, nil)
	if err != nil {
		return nil, err
	}
	return glueRecord{"RunId": rec["RunId"]}, nil
}

func glueStartDQRecommendation(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	if n := in["CreatedRulesetName"]; n != nil {
		if _, err := st.find("dqruleset", []any{n}); err == nil {
			return nil, glueErr("AlreadyExistsException", "ruleset %v already exists", n)
		}
	}
	rec, err := st.create("dqrecrun", glueRecord{
		"DataSource": in["DataSource"], "Role": in["Role"], "Status": "RUNNING", "CreatedRulesetName": in["CreatedRulesetName"],
		"RecommendationMode": glueFirst(glueStr(in, "RecommendationMode"), "DEFAULT"), "NumberOfWorkers": in["NumberOfWorkers"], "Timeout": in["Timeout"],
	}, nil)
	if err != nil {
		return nil, err
	}
	return glueRecord{"RunId": rec["RunId"]}, nil
}

// advanceDQ settles evaluation runs (writing one result per ruleset) and
// recommendation runs (optionally creating the recommended ruleset).
func (st *glueStore) advanceDQ(glueRecord) {
	for _, r := range st.all("dqevalrun") {
		if !glueProgress(r, "Status", map[string]string{"RUNNING": "SUCCEEDED", "STOPPING": "STOPPED"}) {
			continue
		}
		r["CompletedOn"], r["ExecutionTime"] = glueEpoch(time.Now()), glueSince(r, "StartedOn")
		if r["Status"] != "SUCCEEDED" {
			continue
		}
		var ids []any
		for _, name := range glueList(r["RulesetNames"]) {
			rs, err := st.find("dqruleset", []any{name})
			if err != nil {
				continue
			}
			var results []any
			for i, rule := range glueDQRules(glueStr(rs, "Ruleset")) {
				results = append(results, glueRecord{"Name": fmt.Sprintf("Rule_%d", i+1), "Description": rule, "EvaluatedRule": rule, "Result": "PASS"})
			}
			res, err := st.create("dqresult", glueRecord{
				"RulesetName": name, "DataSource": r["DataSource"], "RulesetEvaluationRunId": r["RunId"], "Score": 1.0,
				"RuleResults": glueNonNil(results), "EvaluationContext": name, "CompletedOn": r["CompletedOn"],
			}, nil)
			if err == nil {
				ids = append(ids, res["ResultId"])
			}
		}
		r["ResultIds"] = glueNonNil(ids)
	}
	for _, r := range st.all("dqrecrun") {
		if !glueProgress(r, "Status", map[string]string{"RUNNING": "SUCCEEDED", "STOPPING": "STOPPED"}) {
			continue
		}
		r["CompletedOn"], r["ExecutionTime"] = glueEpoch(time.Now()), glueSince(r, "StartedOn")
		if r["Status"] != "SUCCEEDED" {
			continue
		}
		r["RecommendedRuleset"] = "Rules = [ RowCount > 0, IsComplete \"id\" ]"
		if name := r["CreatedRulesetName"]; name != nil {
			_, _ = st.create("dqruleset", glueRecord{"Name": name, "Ruleset": r["RecommendedRuleset"], "RecommendationRunId": r["RunId"], "RuleCount": float64(2)}, nil)
		}
	}
}

func glueCancelRun(kind string) glueHandler {
	return func(st *glueStore, in glueRecord) (glueRecord, *glueError) {
		r, err := st.find(kind, []any{in["RunId"]})
		if err != nil {
			return nil, err
		}
		if r["Status"] != "RUNNING" && r["Status"] != "STARTING" {
			return nil, glueErr("IllegalStateException", "run %v is %v", in["RunId"], r["Status"])
		}
		r["Status"], r["_reads"] = "STOPPING", 0
		return glueRecord{}, nil
	}
}

func gluePublishDQResult(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	rec := glueRecord{"ResultPublishedOn": glueEpoch(time.Now())}
	for k, v := range in {
		if k != "ClientToken" && k != "Tags" {
			rec[k] = v
		}
	}
	created, err := st.create("dqresult", rec, nil)
	if err != nil {
		return nil, err
	}
	return glueRecord{"ResultId": created["ResultId"]}, nil
}

func glueGetDQModel(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	now := glueEpoch(time.Now())
	return glueRecord{"Status": "SUCCEEDED", "StartedOn": now, "CompletedOn": now}, nil
}

func glueGetDQModelResult(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	return glueRecord{"CompletedOn": glueEpoch(time.Now()), "Model": []any{}}, nil
}

func gluePutDQAnnotation(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	return glueRecord{}, st.annotate(in["ProfileId"], "", in["InclusionAnnotation"])
}

func (st *glueStore) annotate(profile, statistic, annotation any) *glueError {
	bucket := st.records["dqannotation"]
	if bucket == nil {
		bucket = map[string]glueRecord{}
		st.records["dqannotation"] = bucket
	}
	if len(bucket) >= glueMaxRecordsPerKind {
		return glueErr("ResourceNumberLimitExceededException", "too many annotations")
	}
	bucket[glueKey([]any{profile, statistic})] = glueRecord{"ProfileId": profile, "StatisticId": statistic,
		"InclusionAnnotation": glueRecord{"Value": annotation, "LastModifiedOn": glueEpoch(time.Now())}, "StatisticRecordedOn": glueEpoch(time.Now())}
	return nil
}

func glueBatchPutDQAnnotation(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	var failed []any
	for _, raw := range glueList(in["InclusionAnnotations"]) {
		m, _ := raw.(map[string]any)
		if err := st.annotate(m["ProfileId"], m["StatisticId"], m["InclusionAnnotation"]); err != nil {
			failed = append(failed, glueRecord{"ProfileId": m["ProfileId"], "StatisticId": m["StatisticId"], "FailureReason": err.msg})
		}
	}
	return glueRecord{"FailedInclusionAnnotations": glueNonNil(failed)}, nil
}

func glueListDQAnnotations(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	var out []glueRecord
	for _, a := range st.all("dqannotation") {
		if (in["ProfileId"] == nil || a["ProfileId"] == in["ProfileId"]) && (in["StatisticId"] == nil || a["StatisticId"] == in["StatisticId"]) {
			out = append(out, a)
		}
	}
	return gluePageRecords(out, in, "Annotations")
}

// Column statistics task runs: one active run per table.

func glueStartColumnStatsRun(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	if _, err := st.find("table", []any{in["DatabaseName"], in["TableName"]}); err != nil {
		return nil, err
	}
	for _, r := range st.all("colstatsrun") {
		if r["DatabaseName"] == in["DatabaseName"] && r["TableName"] == in["TableName"] && (r["Status"] == "RUNNING" || r["Status"] == "STARTING") {
			return nil, glueErr("ColumnStatisticsTaskRunningException", "a column statistics task is already running for %v.%v", in["DatabaseName"], in["TableName"])
		}
	}
	rec, err := st.create("colstatsrun", glueRecord{
		"DatabaseName": in["DatabaseName"], "TableName": in["TableName"], "Role": in["Role"], "Status": "RUNNING",
		"ColumnNameList": in["ColumnNameList"], "SampleSize": in["SampleSize"], "SecurityConfiguration": in["SecurityConfiguration"],
		"CustomerId": glueAccount, "CatalogID": glueAccount, "ComputationType": "FULL", "StartTime": glueEpoch(time.Now()),
		"WorkerType": "G.1X", "NumberOfWorkers": float64(2),
	}, nil)
	if err != nil {
		return nil, err
	}
	return glueRecord{"ColumnStatisticsTaskRunId": rec["ColumnStatisticsTaskRunId"]}, nil
}

func (st *glueStore) advanceColumnStats(glueRecord) {
	for _, r := range st.all("colstatsrun") {
		if glueProgress(r, "Status", map[string]string{"RUNNING": "SUCCEEDED", "STARTING": "SUCCEEDED"}) {
			r["EndTime"] = glueEpoch(time.Now())
		}
	}
}

func glueStopColumnStatsRun(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	if _, err := st.find("table", []any{in["DatabaseName"], in["TableName"]}); err != nil {
		return nil, err
	}
	for _, r := range st.all("colstatsrun") {
		if r["DatabaseName"] == in["DatabaseName"] && r["TableName"] == in["TableName"] && (r["Status"] == "RUNNING" || r["Status"] == "STARTING") {
			r["Status"], r["EndTime"] = "STOPPED", glueEpoch(time.Now())
			return glueRecord{}, nil
		}
	}
	return nil, glueErr("ColumnStatisticsTaskNotRunningException", "no column statistics task is running for %v.%v", in["DatabaseName"], in["TableName"])
}

func glueColumnStatsSchedule(start bool) glueHandler {
	return func(st *glueStore, in glueRecord) (glueRecord, *glueError) {
		rec, err := st.find("colstatsettings", []any{in["DatabaseName"], in["TableName"]})
		if err != nil {
			return nil, err
		}
		if glueStr(rec, "Schedule") == "" {
			return nil, glueInvalid("the column statistics settings for %v.%v have no schedule", in["DatabaseName"], in["TableName"])
		}
		state := "NOT_SCHEDULED"
		if start {
			state = "SCHEDULED"
		}
		if rec["ScheduleStatus"] == state {
			return nil, glueErr("IllegalStateException", "the schedule is already %s", state)
		}
		rec["ScheduleStatus"] = state
		return glueRecord{}, nil
	}
}

// Materialized view refresh task runs.

func glueStartMVRefresh(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	if _, err := st.find("table", []any{in["DatabaseName"], in["TableName"]}); err != nil {
		return nil, err
	}
	refresh := "INCREMENTAL"
	if in["FullRefresh"] == true {
		refresh = "FULL"
	}
	rec, err := st.create("mvrefreshrun", glueRecord{
		"CatalogId": in["CatalogId"], "DatabaseName": in["DatabaseName"], "TableName": in["TableName"], "Status": "RUNNING",
		"RefreshType": refresh, "CustomerId": glueAccount, "StartTime": glueEpoch(time.Now()),
	}, nil)
	if err != nil {
		return nil, err
	}
	return glueRecord{"MaterializedViewRefreshTaskRunId": rec["MaterializedViewRefreshTaskRunId"]}, nil
}

func (st *glueStore) advanceMVRefresh(glueRecord) {
	for _, r := range st.all("mvrefreshrun") {
		if glueProgress(r, "Status", map[string]string{"RUNNING": "SUCCEEDED"}) {
			r["EndTime"] = glueEpoch(time.Now())
		}
	}
}

func glueStopMVRefresh(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	for _, r := range st.all("mvrefreshrun") {
		if r["DatabaseName"] == in["DatabaseName"] && r["TableName"] == in["TableName"] && r["Status"] == "RUNNING" {
			r["Status"], r["EndTime"] = "STOPPED", glueEpoch(time.Now())
			return glueRecord{}, nil
		}
	}
	return nil, glueErr("MaterializedViewTaskNotRunningException", "no refresh is running for %v.%v", in["DatabaseName"], in["TableName"])
}

// Table optimizers: the response shapes use lower-case member names.

func glueTableOptimizerView(rec glueRecord) glueRecord {
	return glueRecord{"type": rec["Type"], "configuration": rec["TableOptimizerConfiguration"], "configurationSource": "catalog"}
}

func glueGetTableOptimizer(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	rec, err := st.find("tableoptimizer", []any{in["DatabaseName"], in["TableName"], in["Type"]})
	if err != nil {
		return nil, err
	}
	return glueRecord{"CatalogId": in["CatalogId"], "DatabaseName": in["DatabaseName"], "TableName": in["TableName"], "TableOptimizer": glueTableOptimizerView(rec)}, nil
}

func glueBatchGetTableOptimizer(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	var found, failed []any
	for _, raw := range glueList(in["Entries"]) {
		e, _ := raw.(map[string]any)
		rec, err := st.find("tableoptimizer", []any{e["databaseName"], e["tableName"], e["type"]})
		if err != nil {
			failed = append(failed, glueRecord{"catalogId": e["catalogId"], "databaseName": e["databaseName"], "tableName": e["tableName"], "type": e["type"], "error": glueErrorDetail(err)})
			continue
		}
		found = append(found, glueRecord{"catalogId": e["catalogId"], "databaseName": e["databaseName"], "tableName": e["tableName"], "tableOptimizer": glueTableOptimizerView(rec)})
	}
	return glueRecord{"TableOptimizers": glueNonNil(found), "Failures": glueNonNil(failed)}, nil
}

func glueListTableOptimizerRuns(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	if _, err := st.find("tableoptimizer", []any{in["DatabaseName"], in["TableName"], in["Type"]}); err != nil {
		return nil, err
	}
	return glueRecord{"CatalogId": in["CatalogId"], "DatabaseName": in["DatabaseName"], "TableName": in["TableName"], "TableOptimizerRuns": []any{}}, nil
}

// Zero-ETL integrations.

func glueCreateIntegration(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	for _, existing := range st.all("integration") {
		if existing["IntegrationName"] == in["IntegrationName"] {
			return nil, glueErr("IntegrationConflictOperationFault", "integration %v already exists", in["IntegrationName"])
		}
	}
	rec := glueRecord{"Status": "ACTIVE", "Errors": []any{}}
	for k, v := range in {
		rec[k] = v
	}
	rec["IntegrationArn"] = fmt.Sprintf("arn:aws:glue:%s:%s:integration:%s", glueRegion, glueAccount, glueUUID(st.newID("")))
	if tags := glueList(in["Tags"]); tags != nil {
		rec["Tags"] = tags
	}
	return st.create("integration", rec, nil)
}

func (st *glueStore) integration(id any) (glueRecord, *glueError) {
	for _, r := range st.all("integration") {
		if r["IntegrationArn"] == id || r["IntegrationName"] == id {
			return r, nil
		}
	}
	return nil, glueErr("IntegrationNotFoundFault", "integration %v not found", id)
}

func glueModifyIntegration(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	rec, err := st.integration(in["IntegrationIdentifier"])
	if err != nil {
		return nil, err
	}
	for _, f := range []string{"DataFilter", "Description", "IntegrationConfig", "IntegrationName"} {
		if v, ok := in[f]; ok {
			rec[f] = v
		}
	}
	glueTouch(rec)
	return rec, nil
}

func glueDeleteIntegration(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	rec, err := st.integration(in["IntegrationIdentifier"])
	if err != nil {
		return nil, err
	}
	st.remove("integration", rec)
	out := glueWrap(rec, "")
	out["Status"] = "DELETING"
	return out, nil
}

func glueDescribeIntegrations(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	if id := in["IntegrationIdentifier"]; id != nil {
		rec, err := st.integration(id)
		if err != nil {
			return nil, err
		}
		return glueRecord{"Integrations": []any{rec}}, nil
	}
	return st.list(glueOps["DescribeIntegrations"], in)
}

func glueDescribeInboundIntegrations(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	var out []glueRecord
	for _, r := range st.all("integration") {
		if (in["IntegrationArn"] == nil || r["IntegrationArn"] == in["IntegrationArn"]) && (in["TargetArn"] == nil || r["TargetArn"] == in["TargetArn"]) {
			out = append(out, r)
		}
	}
	page, err := gluePageRecords(out, in, "InboundIntegrations")
	if err != nil {
		return nil, err
	}
	if next, ok := page["NextToken"]; ok {
		page["Marker"] = next
	}
	return page, nil
}

func glueCreateIntegrationResourceProperty(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	rec, err := st.createFrom(glueOps["CreateIntegrationResourceProperty"], in)
	if err != nil {
		return nil, err
	}
	rec["ResourcePropertyArn"] = fmt.Sprintf("arn:aws:glue:%s:%s:integration-resource-property/%s", glueRegion, glueAccount, st.newID(""))
	return rec, nil
}

// Glossaries, terms and assets.

func glueCreateGlossaryTerm(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	rec := glueRecord{"GlossaryId": in["GlossaryIdentifier"]}
	for k, v := range in {
		if k != "GlossaryIdentifier" && k != "ClientToken" {
			rec[k] = v
		}
	}
	return st.create("glossaryterm", rec, nil)
}

func glueListGlossaryTerms(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	if _, err := st.find("glossary", []any{in["GlossaryIdentifier"]}); err != nil {
		return nil, err
	}
	var out []glueRecord
	for _, t := range st.all("glossaryterm") {
		if t["GlossaryId"] == in["GlossaryIdentifier"] {
			out = append(out, t)
		}
	}
	return gluePageRecords(out, in, "Items")
}

func glueDeleteGlossary(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	rec, err := st.find("glossary", []any{in["Identifier"]})
	if err != nil {
		return nil, err
	}
	st.remove("glossary", rec)
	return glueRecord{}, nil
}

func glueGlossaryTerms(add bool) glueHandler {
	return func(st *glueStore, in glueRecord) (glueRecord, *glueError) {
		asset, err := st.find("asset", []any{in["AssetIdentifier"]})
		if err != nil {
			return nil, err
		}
		terms := glueList(asset["GlossaryTerms"])
		for _, id := range glueList(in["GlossaryTermIdentifiers"]) {
			if add {
				if _, err := st.find("glossaryterm", []any{id}); err != nil {
					return nil, err
				}
				if !glueContainsAny(terms, id) {
					terms = append(terms, id)
				}
				continue
			}
			var kept []any
			for _, t := range terms {
				if t != id {
					kept = append(kept, t)
				}
			}
			terms = kept
		}
		asset["GlossaryTerms"] = glueNonNil(terms)
		return glueRecord{"AssetIdentifier": in["AssetIdentifier"], "ItemIdentifier": in["ItemIdentifier"], "IterableFormName": in["IterableFormName"], "GlossaryTerms": asset["GlossaryTerms"]}, nil
	}
}

func gluePutAsset(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	// Older CLI models name it AssetTypeId (required); newer ones AssetTypeIdentifier (optional).
	if typeID := glueFirstPresent(in, "AssetTypeIdentifier", "AssetTypeId"); typeID != nil {
		if _, err := st.find("assettype", []any{typeID}); err != nil {
			return nil, err
		}
		delete(in, "AssetTypeIdentifier")
		in["AssetTypeId"] = typeID
	} else {
		in["AssetTypeId"] = glueDefaultAssetType // the output requires one
	}
	if prev, err := st.find("asset", []any{in["Identifier"]}); err == nil {
		st.remove("asset", prev)
	}
	rec := glueRecord{"Id": in["Identifier"], "AssetName": in["Name"], "AssetDescription": in["Description"], "GlossaryTerms": []any{}}
	for k, v := range in {
		if k != "Identifier" && k != "ClientToken" {
			rec[k] = v
		}
	}
	return st.create("asset", rec, nil)
}

func gluePutNamed(kind string) glueHandler {
	return func(st *glueStore, in glueRecord) (glueRecord, *glueError) {
		if prev, err := st.find(kind, []any{in["Name"]}); err == nil {
			st.remove(kind, prev)
		}
		rec := glueRecord{"Id": in["Name"]}
		for k, v := range in {
			if k != "ClientToken" {
				rec[k] = v
			}
		}
		return st.create(kind, rec, nil)
	}
}

func gluePutAttachment(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	// Older CLI models name it FormTypeId (required); newer ones FormTypeIdentifier (optional).
	if formID := glueFirstPresent(in, "FormTypeIdentifier", "FormTypeId"); formID != nil {
		if _, err := st.find("formtype", []any{formID}); err != nil {
			return nil, err
		}
		delete(in, "FormTypeIdentifier")
		in["FormTypeId"] = formID
	}
	if prev, err := st.find("attachment", []any{in["AssetIdentifier"], in["AttachmentName"]}); err == nil {
		st.remove("attachment", prev)
	}
	// Outputs name the asset and item AssetId/ItemId (current model) or echo the
	// *Identifier inputs (older CLI models); keep both, conform picks per model.
	rec := glueRecord{"AssetId": in["AssetIdentifier"], "ItemId": in["ItemIdentifier"]}
	for k, v := range in {
		if k != "ClientToken" {
			rec[k] = v
		}
	}
	return st.create("attachment", rec, nil)
}

func glueListIterableForms(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	if _, err := st.find("asset", []any{in["AssetIdentifier"]}); err != nil {
		return nil, err
	}
	return glueRecord{"Items": []any{}}, nil
}

func glueBatchGetIterableForms(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	if _, err := st.find("asset", []any{in["AssetIdentifier"]}); err != nil {
		return nil, err
	}
	var errs []any
	for _, id := range glueList(in["ItemIdentifiers"]) {
		errs = append(errs, glueRecord{"ItemIdentifier": id, "Code": "EntityNotFoundException", "Message": fmt.Sprintf("form item %v not found", id)})
	}
	return glueRecord{"Items": []any{}, "Errors": glueNonNil(errs)}, nil
}

// Connection types and entities.

func glueRegisterConnectionType(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	rec := glueRecord{"DisplayName": in["ConnectionType"], "Vendor": "Custom"}
	for k, v := range in {
		if k != "Tags" {
			rec[k] = v
		}
	}
	created, err := st.create("connectiontype", rec, nil)
	if err != nil {
		return nil, err
	}
	return glueRecord{"ConnectionTypeArn": fmt.Sprintf("arn:aws:glue:%s:%s:connectionType/%v", glueRegion, glueAccount, created["ConnectionType"])}, nil
}

func glueListEntities(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	if name := in["ConnectionName"]; name != nil {
		if _, err := st.find("connection", []any{name}); err != nil {
			return nil, err
		}
		return glueRecord{"Entities": []any{}}, nil
	}
	// Without a connection the entities are the catalog's databases.
	var out []any
	for _, db := range st.all("database") {
		out = append(out, glueRecord{"EntityName": db["Name"], "Label": db["Name"], "IsParentEntity": true, "Category": "Database", "Description": db["Description"]})
	}
	return glueRecord{"Entities": glueNonNil(out)}, nil
}

func glueConnectionScoped(member string) glueHandler {
	return func(st *glueStore, in glueRecord) (glueRecord, *glueError) {
		if name := in["ConnectionName"]; name != nil {
			if _, err := st.find("connection", []any{name}); err != nil {
				return nil, err
			}
		}
		return glueRecord{member: []any{}}, nil
	}
}

// IAM Identity Center configuration (one per account).

func glueCreateIdentityCenter(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	if _, ok := st.singles["identitycenter"]; ok {
		return nil, glueErr("AlreadyExistsException", "an Identity Center configuration already exists")
	}
	rec := glueRecord{"ApplicationArn": fmt.Sprintf("arn:aws:sso::%s:application/ssoins-stackyard/apl-%s", glueAccount, st.newID(""))}
	for k, v := range in {
		rec[k] = v
	}
	st.singles["identitycenter"] = rec
	return glueRecord{"ApplicationArn": rec["ApplicationArn"]}, nil
}

func glueUpdateIdentityCenter(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	rec, ok := st.singles["identitycenter"]
	if !ok {
		return nil, glueNotFound("no Identity Center configuration exists")
	}
	for k, v := range in {
		rec[k] = v
	}
	return glueRecord{}, nil
}

func glueDeleteIdentityCenter(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	if _, ok := st.singles["identitycenter"]; !ok {
		return nil, glueNotFound("no Identity Center configuration exists")
	}
	delete(st.singles, "identitycenter")
	return glueRecord{}, nil
}

// Tags, keyed by resource ARN; the resource must exist.

func (st *glueStore) resourceForARN(arn string) *glueError {
	const prefix = "arn:aws:glue:"
	rest, ok := strings.CutPrefix(arn, prefix)
	if !ok {
		return glueInvalid("%s is not a Glue resource ARN", arn)
	}
	parts := strings.SplitN(rest, ":", 3)
	if len(parts) != 3 {
		return glueInvalid("%s is not a Glue resource ARN", arn)
	}
	typ, path, _ := strings.Cut(parts[2], "/")
	for name, kind := range glueKinds {
		if kind.arnType != typ {
			continue
		}
		ids := strings.Split(path, "/")
		if len(ids) != len(kind.fields) {
			continue
		}
		anyIDs := make([]any, len(ids))
		for i, id := range ids {
			anyIDs[i] = id
		}
		if _, err := st.find(name, anyIDs); err == nil {
			return nil
		}
	}
	return glueNotFound("resource %s not found", arn)
}

func glueTagResource(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	arn := glueStr(in, "ResourceArn")
	if err := st.resourceForARN(arn); err != nil {
		return nil, err
	}
	tags := st.tags[arn]
	if tags == nil {
		tags = map[string]string{}
	}
	for k, v := range glueTags(in["TagsToAdd"]) {
		tags[k] = v
	}
	if len(tags) > 50 {
		return nil, glueInvalid("a resource can have at most 50 tags")
	}
	st.tags[arn] = tags
	return glueRecord{}, nil
}

func glueUntagResource(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	arn := glueStr(in, "ResourceArn")
	if err := st.resourceForARN(arn); err != nil {
		return nil, err
	}
	for _, k := range glueList(in["TagsToRemove"]) {
		delete(st.tags[arn], fmt.Sprint(k))
	}
	return glueRecord{}, nil
}

func glueGetTags(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	arn := glueStr(in, "ResourceArn")
	if err := st.resourceForARN(arn); err != nil {
		return nil, err
	}
	out := map[string]any{}
	for k, v := range st.tags[arn] {
		out[k] = v
	}
	return glueRecord{"Tags": out}, nil
}

// Resource policies, keyed by resource ARN ("" is the Data Catalog policy).

func gluePolicyHash(policy string) string {
	sum := sha256.Sum256([]byte(policy))
	return hex.EncodeToString(sum[:])[:32]
}

func gluePutResourcePolicy(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	policy := glueStr(in, "PolicyInJson")
	var doc map[string]any
	if err := json.Unmarshal([]byte(policy), &doc); err != nil || doc["Statement"] == nil {
		return nil, glueInvalid("PolicyInJson must be a JSON policy document with a Statement")
	}
	arn := glueStr(in, "ResourceArn")
	prev, exists := st.policies[arn]
	switch glueStr(in, "PolicyExistsCondition") {
	case "MUST_EXIST":
		if !exists {
			return nil, glueErr("ConditionCheckFailureException", "no policy exists")
		}
	case "NOT_EXIST":
		if exists {
			return nil, glueErr("ConditionCheckFailureException", "a policy already exists")
		}
	}
	if want := glueStr(in, "PolicyHashCondition"); want != "" && (!exists || prev["PolicyHash"] != want) {
		return nil, glueErr("ConditionCheckFailureException", "the policy hash does not match")
	}
	now := glueEpoch(time.Now())
	rec := glueRecord{"PolicyInJson": policy, "PolicyHash": gluePolicyHash(policy), "UpdateTime": now, "CreateTime": now, "_arn": arn}
	if exists {
		rec["CreateTime"] = prev["CreateTime"]
	}
	st.policies[arn] = rec
	return glueRecord{"PolicyHash": rec["PolicyHash"]}, nil
}

func glueGetResourcePolicy(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	rec, ok := st.policies[glueStr(in, "ResourceArn")]
	if !ok {
		return nil, glueNotFound("no resource policy is set")
	}
	return rec, nil
}

func glueDeleteResourcePolicy(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	arn := glueStr(in, "ResourceArn")
	rec, ok := st.policies[arn]
	if !ok {
		return nil, glueNotFound("no resource policy is set")
	}
	if want := glueStr(in, "PolicyHashCondition"); want != "" && rec["PolicyHash"] != want {
		return nil, glueErr("ConditionCheckFailureException", "the policy hash does not match")
	}
	delete(st.policies, arn)
	return glueRecord{}, nil
}

func glueGetResourcePolicies(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	var out []glueRecord
	for _, arn := range glueSortedKeys(st.policies) {
		out = append(out, st.policies[arn])
	}
	return gluePageRecords(out, in, "GetResourcePoliciesResponseList")
}

func glueSortedKeys(m map[string]glueRecord) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && keys[j] < keys[j-1]; j-- {
			keys[j], keys[j-1] = keys[j-1], keys[j]
		}
	}
	return keys
}

// Code generation: scripts are a readable PySpark skeleton of the DAG.

func glueDagScript(nodes []any) string {
	var b strings.Builder
	b.WriteString("import sys\nfrom awsglue.context import GlueContext\nfrom pyspark.context import SparkContext\n\nglueContext = GlueContext(SparkContext.getOrCreate())\n")
	for _, raw := range nodes {
		n, _ := raw.(map[string]any)
		fmt.Fprintf(&b, "# %v: %v\n", n["Id"], n["NodeType"])
		for _, a := range glueList(n["Args"]) {
			if m, ok := a.(map[string]any); ok {
				fmt.Fprintf(&b, "#   %v = %v\n", m["Name"], m["Value"])
			}
		}
	}
	return b.String()
}

func glueCreateScript(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	script := glueDagScript(glueList(in["DagNodes"]))
	if in["Language"] == "SCALA" {
		return glueRecord{"ScalaCode": "// generated by stackyard\n" + strings.ReplaceAll(script, "#", "//")}, nil
	}
	return glueRecord{"PythonScript": script}, nil
}

func glueGetPlan(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	var lines []string
	for _, raw := range glueList(in["Mapping"]) {
		m, _ := raw.(map[string]any)
		lines = append(lines, fmt.Sprintf("# %v.%v -> %v.%v", m["SourceTable"], m["SourcePath"], m["TargetTable"], m["TargetPath"]))
	}
	script := glueDagScript(nil) + strings.Join(lines, "\n")
	if in["Language"] == "SCALA" {
		return glueRecord{"ScalaCode": strings.ReplaceAll(script, "#", "//")}, nil
	}
	return glueRecord{"PythonScript": script}, nil
}

func glueGetDataflowGraph(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	var nodes []any
	for i, line := range strings.Split(glueStr(in, "PythonScript"), "\n") {
		if strings.Contains(line, "glueContext.") && strings.Contains(line, "=") {
			id := strings.TrimSpace(strings.SplitN(line, "=", 2)[0])
			nodes = append(nodes, glueRecord{"Id": id, "NodeType": "Transform", "Args": []any{}, "LineNumber": float64(i + 1)})
		}
	}
	return glueRecord{"DagNodes": glueNonNil(nodes), "DagEdges": []any{}}, nil
}

func glueGetMapping(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	src, _ := in["Source"].(map[string]any)
	table, err := st.find("table", []any{src["DatabaseName"], src["TableName"]})
	if err != nil {
		return nil, err
	}
	var mapping []any
	if sd, ok := table["StorageDescriptor"].(map[string]any); ok {
		for _, c := range glueList(sd["Columns"]) {
			m, _ := c.(map[string]any)
			mapping = append(mapping, glueRecord{"SourceTable": src["TableName"], "SourcePath": m["Name"], "SourceType": m["Type"], "TargetTable": src["TableName"], "TargetPath": m["Name"], "TargetType": m["Type"]})
		}
	}
	return glueRecord{"Mapping": glueNonNil(mapping)}, nil
}

func glueGetDashboardURL(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	return glueRecord{"Url": fmt.Sprintf("https://%s.console.aws.amazon.com/gluestudio/home?region=%s#/%v/%v", glueRegion, glueRegion, in["ResourceType"], in["ResourceId"])}, nil
}

// glueDefaultAssetType names the type of assets put without AssetTypeIdentifier.
const glueDefaultAssetType = "stackyard-default-asset-type"

// glueFirstPresent returns the first non-nil value among keys.
func glueFirstPresent(in glueRecord, keys ...string) any {
	for _, k := range keys {
		if v := in[k]; v != nil {
			return v
		}
	}
	return nil
}
