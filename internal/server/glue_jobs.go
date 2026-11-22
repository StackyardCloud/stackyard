package server

import (
	"fmt"
	"path"
	"strings"
	"time"
)

// ETL behaviour: job runs and bookmarks, crawler and schedule states, triggers,
// workflow runs, blueprint runs, dev endpoints and ML transform task runs.
//
// Runs don't execute anything. A run reports its in-flight state (RUNNING,
// STOPPING, ...) on the first read after it changes and settles on the next
// read, so callers see the transition the way they would poll for it.

const (
	glueDefaultJobTimeoutMinutes = 2880
	// glueMaxCrawlHistory bounds the crawl history kept per crawler.
	glueMaxCrawlHistory = 100
)

func init() {
	registerGlueHandlers(map[string]glueHandler{
		"StartJobRun":                         glueStartJobRun,
		"GetJobRun":                           glueAdvanceThen("GetJobRun", func(st *glueStore, in glueRecord) { st.advanceJobRuns(in["JobName"]) }),
		"GetJobRuns":                          glueAdvanceThen("GetJobRuns", func(st *glueStore, in glueRecord) { st.advanceJobRuns(in["JobName"]) }),
		"BatchStopJobRun":                     glueBatchStopJobRun,
		"GetJobBookmark":                      glueGetJobBookmark,
		"ResetJobBookmark":                    glueResetJobBookmark,
		"GetSystemLogsForJobRun":              glueGetSystemLogsForJobRun,
		"UpdateJobFromSourceControl":          glueSourceControlSync,
		"UpdateSourceControlFromJob":          glueSourceControlSync,
		"StartJobUpgradeAnalysis":             glueStartJobUpgradeAnalysis,
		"StopJobUpgradeAnalysis":              glueStopJobUpgradeAnalysis,
		"CreateCrawler":                       glueCreateCrawler,
		"UpdateCrawler":                       glueUpdateCrawler,
		"DeleteCrawler":                       glueDeleteCrawler,
		"StartCrawler":                        glueStartCrawler,
		"StopCrawler":                         glueStopCrawler,
		"GetCrawler":                          glueAdvanceThen("GetCrawler", (*glueStore).advanceCrawlersAny),
		"GetCrawlers":                         glueAdvanceThen("GetCrawlers", (*glueStore).advanceCrawlersAny),
		"BatchGetCrawlers":                    glueAdvanceThen("BatchGetCrawlers", (*glueStore).advanceCrawlersAny),
		"ListCrawls":                          glueListCrawls,
		"GetCrawlerMetrics":                   glueGetCrawlerMetrics,
		"StartCrawlerSchedule":                glueCrawlerSchedule(true),
		"StopCrawlerSchedule":                 glueCrawlerSchedule(false),
		"UpdateCrawlerSchedule":               glueUpdateCrawlerSchedule,
		"CreateTrigger":                       glueCreateTrigger,
		"StartTrigger":                        glueStartTrigger,
		"StopTrigger":                         glueStopTrigger,
		"GetTriggers":                         glueDependentTriggers(false),
		"ListTriggers":                        glueDependentTriggers(true),
		"GetWorkflow":                         glueGetWorkflow,
		"BatchGetWorkflows":                   glueBatchGetWorkflows,
		"DeleteWorkflow":                      glueDeleteWorkflow,
		"StartWorkflowRun":                    glueStartWorkflowRun,
		"GetWorkflowRun":                      glueAdvanceThen("GetWorkflowRun", func(st *glueStore, in glueRecord) { st.advanceWorkflowRuns(in["Name"]) }),
		"GetWorkflowRuns":                     glueAdvanceThen("GetWorkflowRuns", func(st *glueStore, in glueRecord) { st.advanceWorkflowRuns(in["Name"]) }),
		"StopWorkflowRun":                     glueStopWorkflowRun,
		"ResumeWorkflowRun":                   glueResumeWorkflowRun,
		"GetWorkflowRunProperties":            glueGetWorkflowRunProperties,
		"PutWorkflowRunProperties":            gluePutWorkflowRunProperties,
		"CreateBlueprint":                     glueCreateBlueprint,
		"StartBlueprintRun":                   glueStartBlueprintRun,
		"GetBlueprintRun":                     glueAdvanceThen("GetBlueprintRun", func(st *glueStore, in glueRecord) { st.advanceBlueprintRuns(in["BlueprintName"]) }),
		"GetBlueprintRuns":                    glueAdvanceThen("GetBlueprintRuns", func(st *glueStore, in glueRecord) { st.advanceBlueprintRuns(in["BlueprintName"]) }),
		"CreateDevEndpoint":                   glueCreateDevEndpoint,
		"UpdateDevEndpoint":                   glueUpdateDevEndpoint,
		"CreateMLTransform":                   glueCreateMLTransform,
		"StartMLEvaluationTaskRun":            glueStartMLTaskRun("EVALUATION"),
		"StartMLLabelingSetGenerationTaskRun": glueStartMLTaskRun("LABELING_SET_GENERATION"),
		"StartImportLabelsTaskRun":            glueStartMLTaskRun("IMPORT_LABELS"),
		"StartExportLabelsTaskRun":            glueStartMLTaskRun("EXPORT_LABELS"),
		"GetMLTaskRun":                        glueAdvanceThen("GetMLTaskRun", func(st *glueStore, in glueRecord) { st.advanceMLTaskRuns(in["TransformId"]) }),
		"GetMLTaskRuns":                       glueAdvanceThen("GetMLTaskRuns", func(st *glueStore, in glueRecord) { st.advanceMLTaskRuns(in["TransformId"]) }),
		"GetMLTransform":                      glueAdvanceThen("GetMLTransform", func(st *glueStore, in glueRecord) { st.advanceMLTaskRuns(in["TransformId"]) }),
		"CancelMLTaskRun":                     glueCancelMLTaskRun,
		"GetJobUpgradeAnalysis":               glueAdvanceThen("GetJobUpgradeAnalysis", (*glueStore).advanceUpgradeAnalyses),
	})
}

// glueAdvanceThen moves runs along, then serves op from the generic table.
func glueAdvanceThen(op string, advance func(*glueStore, glueRecord)) glueHandler {
	spec := glueOps[op]
	return func(st *glueStore, in glueRecord) (glueRecord, *glueError) {
		advance(st, in)
		return st.generic(spec, in)
	}
}

// glueProgress applies one read to a run: an in-flight state is reported
// once, then the next read moves it to next[state]. It reports whether the
// state changed.
func glueProgress(rec glueRecord, field string, next map[string]string) bool {
	to, ok := next[glueStr(rec, field)]
	if !ok {
		return false
	}
	reads, _ := rec["_reads"].(int)
	if reads < 1 {
		rec["_reads"] = reads + 1
		return false
	}
	rec["_reads"] = 0
	rec[field] = to
	glueTouch(rec)
	return true
}

func glueSince(rec glueRecord, field string) float64 {
	start, _ := rec[field].(float64)
	return float64(int64(glueEpoch(time.Now()) - start))
}

// Jobs.

var glueRunFinish = map[string]string{"STARTING": "SUCCEEDED", "RUNNING": "SUCCEEDED", "STOPPING": "STOPPED", "WAITING": "SUCCEEDED"}

func glueJobMaxConcurrent(job glueRecord) int {
	if ep, ok := job["ExecutionProperty"].(map[string]any); ok {
		if n, ok := ep["MaxConcurrentRuns"].(float64); ok && n > 0 {
			return int(n)
		}
	}
	return 1
}

func glueRunActive(state string) bool {
	return state == "STARTING" || state == "RUNNING" || state == "STOPPING" || state == "WAITING"
}

func (st *glueStore) startJobRun(jobName any, args map[string]any, overrides glueRecord) (glueRecord, *glueError) {
	job, err := st.find("job", []any{jobName})
	if err != nil {
		return nil, err
	}
	active := 0
	for _, r := range st.all("jobrun") {
		if r["JobName"] == jobName && glueRunActive(glueStr(r, "JobRunState")) {
			active++
		}
	}
	if active >= glueJobMaxConcurrent(job) {
		return nil, glueErr("ConcurrentRunsExceededException", "job %v already has %d running run(s), the limit for the job", jobName, active)
	}
	arguments := map[string]any{}
	for k, v := range glueTags(job["DefaultArguments"]) {
		arguments[k] = v
	}
	for k, v := range args {
		arguments[k] = v
	}
	rec := glueRecord{
		"JobName": jobName, "Id": "jr_" + st.newID(""), "Attempt": float64(0), "JobRunState": "RUNNING",
		"Arguments": arguments, "PredecessorRuns": []any{}, "LogGroupName": "/aws-glue/jobs",
		"Timeout": glueFirstValue(overrides, "Timeout"), "ExecutionTime": float64(0),
	}
	for _, f := range []string{"AllocatedCapacity", "MaxCapacity", "WorkerType", "NumberOfWorkers", "GlueVersion", "ExecutionClass", "JobMode", "SecurityConfiguration", "NotificationProperty", "Timeout", "MaintenanceWindow", "JobRunQueuingEnabled"} {
		if v := glueFirstValue(overrides, f); v != nil {
			rec[f] = v
		} else if v := job[f]; v != nil {
			rec[f] = v
		}
	}
	if rec["Timeout"] == nil {
		rec["Timeout"] = float64(glueDefaultJobTimeoutMinutes)
	}
	if name := glueStr(overrides, "TriggerName"); name != "" {
		rec["TriggerName"] = name
	}
	return st.create("jobrun", rec, nil)
}

func glueStartJobRun(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	args, _ := in["Arguments"].(map[string]any)
	rec, err := st.startJobRun(in["JobName"], args, in)
	if err != nil {
		return nil, err
	}
	return glueRecord{"JobRunId": rec["Id"]}, nil
}

func (st *glueStore) advanceJobRuns(jobName any) {
	for _, r := range st.all("jobrun") {
		if r["JobName"] != jobName || !glueProgress(r, "JobRunState", glueRunFinish) {
			continue
		}
		r["CompletedOn"] = glueEpoch(time.Now())
		r["ExecutionTime"] = glueSince(r, "StartedOn")
		if r["JobRunState"] == "SUCCEEDED" {
			st.advanceBookmark(r)
		}
	}
}

func (st *glueStore) advanceBookmark(run glueRecord) {
	job, err := st.find("job", []any{run["JobName"]})
	if err != nil {
		return
	}
	opt := glueTags(run["Arguments"])["--job-bookmark-option"]
	if opt != "job-bookmark-enable" {
		return
	}
	bm := st.bookmark(job)
	bm["Version"] = bm["Version"].(float64) + 1
	bm["Run"] = bm["Run"].(float64) + 1
	bm["PreviousRunId"], bm["RunId"] = bm["RunId"], run["Id"]
	bm["JobBookmark"] = fmt.Sprintf(`{"run":%v,"runId":"%v"}`, bm["Run"], run["Id"])
}

func (st *glueStore) bookmark(job glueRecord) glueRecord {
	bm, ok := job["_bookmark"].(glueRecord)
	if !ok {
		bm = glueRecord{"JobName": job["Name"], "Version": float64(0), "Run": float64(0), "Attempt": float64(0), "JobBookmark": "{}"}
		job["_bookmark"] = bm
	}
	return bm
}

func glueGetJobBookmark(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	job, err := st.find("job", []any{in["JobName"]})
	if err != nil {
		return nil, err
	}
	return glueRecord{"JobBookmarkEntry": st.bookmark(job)}, nil
}

func glueResetJobBookmark(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	job, err := st.find("job", []any{in["JobName"]})
	if err != nil {
		return nil, err
	}
	if runID := in["RunId"]; runID != nil {
		if _, err := st.find("jobrun", []any{in["JobName"], runID}); err != nil {
			return nil, err
		}
	}
	bm := st.bookmark(job)
	bm["Version"] = bm["Version"].(float64) + 1
	bm["Run"], bm["Attempt"], bm["JobBookmark"] = float64(0), float64(0), "{}"
	bm["RunId"] = in["RunId"]
	return glueRecord{"JobBookmarkEntry": bm}, nil
}

func glueBatchStopJobRun(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	if _, err := st.find("job", []any{in["JobName"]}); err != nil {
		return nil, err
	}
	var ok, errs []any
	for _, id := range glueList(in["JobRunIds"]) {
		run, err := st.find("jobrun", []any{in["JobName"], id})
		if err == nil && !glueRunActive(glueStr(run, "JobRunState")) {
			err = glueInvalid("job run %v is %s and can't be stopped", id, glueStr(run, "JobRunState"))
		}
		if err != nil {
			errs = append(errs, glueRecord{"JobName": in["JobName"], "JobRunId": id, "ErrorDetail": glueErrorDetail(err)})
			continue
		}
		run["JobRunState"], run["_reads"] = "STOPPING", 0
		ok = append(ok, glueRecord{"JobName": in["JobName"], "JobRunId": id})
	}
	return glueRecord{"SuccessfulSubmissions": glueNonNil(ok), "Errors": glueNonNil(errs)}, nil
}

func glueGetSystemLogsForJobRun(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	run, err := st.find("jobrun", []any{in["JobName"], in["RunId"]})
	if err != nil {
		return nil, err
	}
	return glueRecord{"SystemLogsUrl": fmt.Sprintf("https://console.aws.amazon.com/cloudwatch/home?region=%s#logsV2:log-groups/log-group/$252Faws-glue$252Fjobs$252Foutput/log-events/%v", glueRegion, run["Id"])}, nil
}

func glueSourceControlSync(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	job, err := st.find("job", []any{in["JobName"]})
	if err != nil {
		return nil, err
	}
	details, _ := job["SourceControlDetails"].(map[string]any)
	if details == nil {
		details = map[string]any{}
	}
	for _, f := range []string{"Provider", "RepositoryName", "RepositoryOwner", "BranchName", "Folder", "AuthStrategy"} {
		if v, ok := in[f]; ok {
			details[f] = v
		}
	}
	if c, ok := in["CommitId"]; ok {
		details["LastCommitId"] = c
	}
	job["SourceControlDetails"] = details
	glueTouch(job)
	return glueRecord{"JobName": in["JobName"]}, nil
}

func glueStartJobUpgradeAnalysis(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	job, err := st.find("job", []any{in["jobName"]})
	if err != nil {
		return nil, err
	}
	id := in["analysisId"]
	if id == nil {
		id = "jua" + glueUUID(st.newID("")) // analysis ids are exactly 39 characters
	}
	rec, err := st.create("upgradeanalysis", glueRecord{
		"jobName": in["jobName"], "analysisId": id, "analysisType": "AUTO_UPGRADE", "status": "OBSERVING",
		"sourceVersion": glueFirst(glueStr(job, "GlueVersion"), "4.0"), "targetVersion": "5.0",
	}, nil)
	if err != nil {
		return nil, err
	}
	return glueRecord{"jobName": rec["jobName"], "analysisId": rec["analysisId"]}, nil
}

func (st *glueStore) advanceUpgradeAnalyses(in glueRecord) {
	for _, a := range st.all("upgradeanalysis") {
		if a["jobName"] == in["jobName"] && glueProgress(a, "status", map[string]string{"OBSERVING": "SUCCEEDED"}) {
			a["autoUpgradeEvaluationSummary"] = "The job runs on the target version without changes."
		}
	}
}

func glueStopJobUpgradeAnalysis(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	a, err := st.find("upgradeanalysis", []any{in["jobName"], in["analysisId"]})
	if err != nil {
		return nil, err
	}
	if s := glueStr(a, "status"); s != "OBSERVING" && s != "WAITING_FOR_CONSENT" && s != "UPGRADING" {
		return nil, glueErr("IllegalSessionStateException", "upgrade analysis %v is %s", in["analysisId"], s)
	}
	a["status"] = "STOPPED"
	return glueRecord{}, nil
}

// Crawlers.

func glueCrawlerScheduleValue(v any, state string) any {
	expr, _ := v.(string)
	if expr == "" {
		return nil
	}
	return glueRecord{"ScheduleExpression": expr, "State": state}
}

func glueCheckCrawlerTargets(st *glueStore, in glueRecord) *glueError {
	if targets, ok := in["Targets"].(map[string]any); ok {
		n := 0
		for _, v := range targets {
			n += len(glueList(v))
		}
		if n == 0 {
			return glueInvalid("Targets must include at least one target")
		}
	}
	for _, c := range glueList(in["Classifiers"]) {
		if _, err := st.find("classifier", []any{c}); err != nil {
			return err
		}
	}
	return nil
}

func glueCreateCrawler(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	if err := glueCheckCrawlerTargets(st, in); err != nil {
		return nil, err
	}
	rec, err := st.createFrom(glueOps["CreateCrawler"], in)
	if err != nil {
		return nil, err
	}
	rec["State"], rec["Version"], rec["CrawlElapsedTime"] = "READY", float64(1), float64(0)
	rec["Schedule"] = glueCrawlerScheduleValue(in["Schedule"], "SCHEDULED")
	return glueRecord{}, nil
}

func glueUpdateCrawler(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	rec, err := st.find("crawler", []any{in["Name"]})
	if err != nil {
		return nil, err
	}
	if glueStr(rec, "State") != "READY" {
		return nil, glueErr("CrawlerRunningException", "crawler %v is %s", in["Name"], glueStr(rec, "State"))
	}
	if err := glueCheckCrawlerTargets(st, in); err != nil {
		return nil, err
	}
	schedule, hasSchedule := in["Schedule"]
	st.merge("crawler", rec, in, "", []string{"Name"})
	if hasSchedule {
		rec["Schedule"] = glueCrawlerScheduleValue(schedule, "SCHEDULED")
	}
	rec["Version"] = rec["Version"].(float64) + 1
	return glueRecord{}, nil
}

func glueDeleteCrawler(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	rec, err := st.find("crawler", []any{in["Name"]})
	if err != nil {
		return nil, err
	}
	if glueStr(rec, "State") != "READY" {
		return nil, glueErr("CrawlerRunningException", "crawler %v is %s", in["Name"], glueStr(rec, "State"))
	}
	st.remove("crawler", rec)
	return glueRecord{}, nil
}

func (st *glueStore) startCrawler(name any) *glueError {
	rec, err := st.find("crawler", []any{name})
	if err != nil {
		return err
	}
	if glueStr(rec, "State") != "READY" {
		return glueErr("CrawlerRunningException", "crawler %v is already %s", name, glueStr(rec, "State"))
	}
	now := glueEpoch(time.Now())
	rec["State"], rec["_reads"], rec["_started"] = "RUNNING", 0, now
	crawls, _ := rec["_crawls"].([]any)
	if len(crawls) >= glueMaxCrawlHistory {
		crawls = crawls[len(crawls)-glueMaxCrawlHistory+1:]
	}
	rec["_crawls"] = append(crawls, glueRecord{"CrawlId": st.newID(""), "State": "RUNNING", "StartTime": now, "LogGroup": "/aws-glue/crawlers", "LogStream": fmt.Sprint(name)})
	return nil
}

func glueStartCrawler(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	return glueRecord{}, st.startCrawler(in["Name"])
}

func glueStopCrawler(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	rec, err := st.find("crawler", []any{in["Name"]})
	if err != nil {
		return nil, err
	}
	switch glueStr(rec, "State") {
	case "RUNNING":
		rec["State"], rec["_reads"] = "STOPPING", 0
		return glueRecord{}, nil
	case "STOPPING":
		return nil, glueErr("CrawlerStoppingException", "crawler %v is already stopping", in["Name"])
	}
	return nil, glueErr("CrawlerNotRunningException", "crawler %v is not running", in["Name"])
}

func (st *glueStore) advanceCrawlersAny(glueRecord) {
	for _, c := range st.all("crawler") {
		stopping := glueStr(c, "State") == "STOPPING"
		if !glueProgress(c, "State", map[string]string{"RUNNING": "READY", "STOPPING": "READY"}) {
			continue
		}
		now := glueEpoch(time.Now())
		status, history := "SUCCEEDED", "COMPLETED"
		if stopping {
			status, history = "CANCELLED", "STOPPED"
		} else {
			st.crawlTargets(c)
		}
		c["CrawlElapsedTime"] = float64(0)
		c["LastCrawl"] = glueRecord{"Status": status, "StartTime": c["_started"], "LogGroup": "/aws-glue/crawlers", "LogStream": c["Name"], "MessagePrefix": st.newID("")}
		if crawls, _ := c["_crawls"].([]any); len(crawls) > 0 {
			last := crawls[len(crawls)-1].(glueRecord)
			last["State"], last["EndTime"] = history, now
		}
		started, _ := c["_started"].(float64)
		runtime := now - started
		c["_lastRuntime"] = runtime
	}
}

// crawlTargets registers a table for each S3 target the way a crawl would.
func (st *glueStore) crawlTargets(c glueRecord) {
	db := glueStr(c, "DatabaseName")
	if _, err := st.find("database", []any{db}); db == "" || err != nil {
		return
	}
	targets, _ := c["Targets"].(map[string]any)
	created := 0
	for _, t := range glueList(targets["S3Targets"]) {
		m, _ := t.(map[string]any)
		loc := strings.TrimRight(glueStr(m, "Path"), "/")
		if loc == "" {
			continue
		}
		name := strings.ToLower(glueStr(c, "TablePrefix") + strings.ReplaceAll(path.Base(loc), "-", "_"))
		if _, err := st.find("table", []any{db, name}); err == nil {
			continue
		}
		if _, err := glueCreateTable(st, glueRecord{"DatabaseName": db, "TableInput": map[string]any{
			"Name": name, "TableType": "EXTERNAL_TABLE", "Parameters": map[string]any{"classification": "unknown", "UPDATED_BY_CRAWLER": c["Name"]},
			"StorageDescriptor": map[string]any{"Location": loc + "/", "Columns": []any{}},
		}}); err == nil {
			created++
		}
	}
	c["_tablesCreated"] = float64(created)
}

func glueListCrawls(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	st.advanceCrawlersAny(in)
	rec, err := st.find("crawler", []any{in["CrawlerName"]})
	if err != nil {
		return nil, err
	}
	var crawls []glueRecord
	for _, c := range glueList(rec["_crawls"]) {
		crawls = append(crawls, c.(glueRecord))
	}
	return gluePageRecords(crawls, in, "Crawls")
}

func glueGetCrawlerMetrics(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	st.advanceCrawlersAny(in)
	names := glueList(in["CrawlerNameList"])
	var out []glueRecord
	for _, c := range st.all("crawler") {
		if len(names) > 0 && !glueContainsAny(names, c["Name"]) {
			continue
		}
		last, _ := c["_lastRuntime"].(float64)
		created, _ := c["_tablesCreated"].(float64)
		out = append(out, glueRecord{
			"CrawlerName": c["Name"], "TimeLeftSeconds": float64(0), "StillEstimating": glueStr(c, "State") == "RUNNING",
			"LastRuntimeSeconds": last, "MedianRuntimeSeconds": last, "TablesCreated": created, "TablesUpdated": float64(0), "TablesDeleted": float64(0),
		})
	}
	return gluePageRecords(out, in, "CrawlerMetricsList")
}

func glueCrawlerSchedule(start bool) glueHandler {
	return func(st *glueStore, in glueRecord) (glueRecord, *glueError) {
		rec, err := st.find("crawler", []any{in["CrawlerName"]})
		if err != nil {
			return nil, err
		}
		sched, _ := rec["Schedule"].(glueRecord)
		if sched == nil {
			return nil, glueErr("NoScheduleException", "crawler %v has no schedule", in["CrawlerName"])
		}
		if start {
			if sched["State"] == "SCHEDULED" {
				return nil, glueErr("SchedulerRunningException", "the schedule for crawler %v is already running", in["CrawlerName"])
			}
			sched["State"] = "SCHEDULED"
		} else {
			if sched["State"] != "SCHEDULED" {
				return nil, glueErr("SchedulerNotRunningException", "the schedule for crawler %v is not running", in["CrawlerName"])
			}
			sched["State"] = "NOT_SCHEDULED"
		}
		return glueRecord{}, nil
	}
}

func glueUpdateCrawlerSchedule(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	rec, err := st.find("crawler", []any{in["CrawlerName"]})
	if err != nil {
		return nil, err
	}
	state := "SCHEDULED"
	if sched, ok := rec["Schedule"].(glueRecord); ok {
		state = glueStr(sched, "State")
	}
	rec["Schedule"] = glueCrawlerScheduleValue(in["Schedule"], state)
	glueTouch(rec)
	return glueRecord{}, nil
}

// Triggers.

func glueCheckTriggerActions(st *glueStore, actions []any) *glueError {
	if len(actions) == 0 {
		return glueInvalid("Actions must include at least one action")
	}
	for _, a := range actions {
		m, _ := a.(map[string]any)
		switch {
		case glueStr(m, "JobName") != "":
			if _, err := st.find("job", []any{m["JobName"]}); err != nil {
				return err
			}
		case glueStr(m, "CrawlerName") != "":
			if _, err := st.find("crawler", []any{m["CrawlerName"]}); err != nil {
				return err
			}
		default:
			return glueInvalid("each action needs a JobName or a CrawlerName")
		}
	}
	return nil
}

func glueCreateTrigger(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	typ := glueStr(in, "Type")
	switch {
	case typ == "SCHEDULED" && glueStr(in, "Schedule") == "":
		return nil, glueInvalid("a SCHEDULED trigger needs a Schedule")
	case typ == "CONDITIONAL" && in["Predicate"] == nil:
		return nil, glueInvalid("a CONDITIONAL trigger needs a Predicate")
	case typ == "ON_DEMAND" && in["StartOnCreation"] == true:
		return nil, glueInvalid("StartOnCreation is not supported for ON_DEMAND triggers")
	}
	if err := glueCheckTriggerActions(st, glueList(in["Actions"])); err != nil {
		return nil, err
	}
	if wf := in["WorkflowName"]; wf != nil {
		if _, err := st.find("workflow", []any{wf}); err != nil {
			return nil, err
		}
	}
	rec, err := st.createFrom(glueOps["CreateTrigger"], in)
	if err != nil {
		return nil, err
	}
	rec["Id"], rec["State"] = rec["Name"], "CREATED"
	if in["StartOnCreation"] == true {
		rec["State"] = "ACTIVATED"
	}
	delete(rec, "StartOnCreation")
	return glueRecord{"Name": rec["Name"]}, nil
}

func (st *glueStore) fireTrigger(t glueRecord) {
	for _, a := range glueList(t["Actions"]) {
		m, _ := a.(map[string]any)
		if job := m["JobName"]; job != nil {
			args, _ := m["Arguments"].(map[string]any)
			_, _ = st.startJobRun(job, args, glueRecord{"TriggerName": t["Name"], "Timeout": m["Timeout"]})
		} else if crawler := m["CrawlerName"]; crawler != nil {
			_ = st.startCrawler(crawler)
		}
	}
}

func glueStartTrigger(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	rec, err := st.find("trigger", []any{in["Name"]})
	if err != nil {
		return nil, err
	}
	if glueStr(rec, "Type") == "ON_DEMAND" {
		st.fireTrigger(rec)
	} else {
		rec["State"] = "ACTIVATED"
	}
	return glueRecord{"Name": rec["Name"]}, nil
}

func glueStopTrigger(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	rec, err := st.find("trigger", []any{in["Name"]})
	if err != nil {
		return nil, err
	}
	if glueStr(rec, "Type") == "ON_DEMAND" {
		return nil, glueInvalid("ON_DEMAND triggers can't be stopped")
	}
	rec["State"] = "DEACTIVATED"
	return glueRecord{"Name": rec["Name"]}, nil
}

func glueDependentTriggers(names bool) glueHandler {
	return func(st *glueStore, in glueRecord) (glueRecord, *glueError) {
		dependent := in["DependentJobName"]
		if dependent != nil {
			if _, err := st.find("job", []any{dependent}); err != nil {
				return nil, err
			}
		}
		wantTags := glueTags(in["Tags"])
		var out []glueRecord
		for _, t := range st.all("trigger") {
			if dependent != nil && !glueTriggerRunsJob(t, dependent) {
				continue
			}
			if !glueHasTags(st.tags[st.arn("trigger", t)], wantTags) {
				continue
			}
			out = append(out, t)
		}
		if !names {
			return gluePageRecords(out, in, "Triggers")
		}
		page, err := gluePageRecords(out, in, "TriggerNames")
		if err != nil {
			return nil, err
		}
		var list []any
		for _, t := range glueList(page["TriggerNames"]) {
			list = append(list, t.(glueRecord)["Name"])
		}
		page["TriggerNames"] = glueNonNil(list)
		return page, nil
	}
}

func glueTriggerRunsJob(t glueRecord, job any) bool {
	for _, a := range glueList(t["Actions"]) {
		if m, ok := a.(map[string]any); ok && m["JobName"] == job {
			return true
		}
	}
	return false
}

// Workflows.

func (st *glueStore) workflowGraph(name any) glueRecord {
	var nodes, edges []any
	seen := map[string]bool{}
	addNode := func(typ string, n any) string {
		id := typ + "_" + fmt.Sprint(n)
		if !seen[id] {
			seen[id] = true
			nodes = append(nodes, glueRecord{"Type": typ, "Name": n, "UniqueId": id})
		}
		return id
	}
	for _, t := range st.all("trigger") {
		if t["WorkflowName"] != name {
			continue
		}
		tid := addNode("TRIGGER", t["Name"])
		for _, a := range glueList(t["Actions"]) {
			m, _ := a.(map[string]any)
			if m["JobName"] != nil {
				edges = append(edges, glueRecord{"SourceId": tid, "DestinationId": addNode("JOB", m["JobName"])})
			} else if m["CrawlerName"] != nil {
				edges = append(edges, glueRecord{"SourceId": tid, "DestinationId": addNode("CRAWLER", m["CrawlerName"])})
			}
		}
	}
	return glueRecord{"Nodes": glueNonNil(nodes), "Edges": glueNonNil(edges)}
}

func (st *glueStore) workflowView(wf glueRecord, graph bool) glueRecord {
	out := glueWrap(wf, "")
	var last glueRecord
	for _, r := range st.all("workflowrun") {
		if r["Name"] == wf["Name"] {
			if s, _ := r["StartedOn"].(float64); last == nil || s >= last["StartedOn"].(float64) {
				last = r
			}
		}
	}
	if last != nil {
		out["LastRun"] = last
	}
	if graph {
		out["Graph"] = st.workflowGraph(wf["Name"])
	}
	return out
}

func glueGetWorkflow(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	st.advanceWorkflowRuns(in["Name"])
	wf, err := st.find("workflow", []any{in["Name"]})
	if err != nil {
		return nil, err
	}
	return glueRecord{"Workflow": st.workflowView(wf, in["IncludeGraph"] == true)}, nil
}

func glueBatchGetWorkflows(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	var found, missing []any
	for _, n := range glueList(in["Names"]) {
		st.advanceWorkflowRuns(n)
		if wf, err := st.find("workflow", []any{n}); err == nil {
			found = append(found, st.workflowView(wf, in["IncludeGraph"] == true))
		} else {
			missing = append(missing, n)
		}
	}
	return glueRecord{"Workflows": glueNonNil(found), "MissingWorkflows": glueNonNil(missing)}, nil
}

func glueDeleteWorkflow(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	wf, err := st.find("workflow", []any{in["Name"]})
	if err != nil {
		return nil, err
	}
	for _, t := range st.all("trigger") {
		if t["WorkflowName"] == in["Name"] {
			st.remove("trigger", t)
		}
	}
	st.remove("workflow", wf)
	return glueRecord{"Name": in["Name"]}, nil
}

func (st *glueStore) startWorkflowRun(wf glueRecord, props map[string]any, previous any) (glueRecord, *glueError) {
	if max, ok := wf["MaxConcurrentRuns"].(float64); ok && max > 0 {
		active := 0
		for _, r := range st.all("workflowrun") {
			if r["Name"] == wf["Name"] && (r["Status"] == "RUNNING" || r["Status"] == "STOPPING") {
				active++
			}
		}
		if active >= int(max) {
			return nil, glueErr("ConcurrentRunsExceededException", "workflow %v already has %d running run(s)", wf["Name"], active)
		}
	}
	merged := map[string]any{}
	for k, v := range glueTags(wf["DefaultRunProperties"]) {
		merged[k] = v
	}
	for k, v := range props {
		merged[k] = v
	}
	graph := st.workflowGraph(wf["Name"])
	actions := float64(len(glueList(graph["Edges"])))
	rec, err := st.create("workflowrun", glueRecord{
		"Name": wf["Name"], "WorkflowRunId": "wr_" + st.newID(""), "Status": "RUNNING", "WorkflowRunProperties": merged,
		"PreviousRunId": previous, "Graph": graph,
		"Statistics": glueRecord{"TotalActions": actions, "RunningActions": actions, "SucceededActions": float64(0), "FailedActions": float64(0), "StoppedActions": float64(0), "TimeoutActions": float64(0), "ErroredActions": float64(0), "WaitingActions": float64(0)},
	}, nil)
	if err != nil {
		return nil, err
	}
	for _, t := range st.all("trigger") {
		if t["WorkflowName"] == wf["Name"] && t["Type"] == "ON_DEMAND" {
			st.fireTrigger(t)
		}
	}
	return rec, nil
}

func glueStartWorkflowRun(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	wf, err := st.find("workflow", []any{in["Name"]})
	if err != nil {
		return nil, err
	}
	props, _ := in["RunProperties"].(map[string]any)
	rec, err := st.startWorkflowRun(wf, props, nil)
	if err != nil {
		return nil, err
	}
	return glueRecord{"RunId": rec["WorkflowRunId"]}, nil
}

func (st *glueStore) advanceWorkflowRuns(name any) {
	for _, r := range st.all("workflowrun") {
		if r["Name"] != name || !glueProgress(r, "Status", map[string]string{"RUNNING": "COMPLETED", "STOPPING": "STOPPED"}) {
			continue
		}
		r["CompletedOn"] = glueEpoch(time.Now())
		if stats, ok := r["Statistics"].(glueRecord); ok {
			total := stats["TotalActions"]
			stats["RunningActions"] = float64(0)
			if r["Status"] == "COMPLETED" {
				stats["SucceededActions"] = total
			} else {
				stats["StoppedActions"] = total
			}
		}
	}
}

func glueStopWorkflowRun(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	run, err := st.find("workflowrun", []any{in["Name"], in["RunId"]})
	if err != nil {
		return nil, err
	}
	if run["Status"] != "RUNNING" {
		return nil, glueErr("IllegalWorkflowStateException", "workflow run %v is %v", in["RunId"], run["Status"])
	}
	run["Status"], run["_reads"] = "STOPPING", 0
	return glueRecord{}, nil
}

func glueResumeWorkflowRun(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	wf, err := st.find("workflow", []any{in["Name"]})
	if err != nil {
		return nil, err
	}
	run, err := st.find("workflowrun", []any{in["Name"], in["RunId"]})
	if err != nil {
		return nil, err
	}
	if run["Status"] == "RUNNING" || run["Status"] == "STOPPING" {
		return nil, glueErr("IllegalWorkflowStateException", "workflow run %v is still %v", in["RunId"], run["Status"])
	}
	props, _ := run["WorkflowRunProperties"].(map[string]any)
	next, err := st.startWorkflowRun(wf, props, run["WorkflowRunId"])
	if err != nil {
		return nil, err
	}
	return glueRecord{"RunId": next["WorkflowRunId"], "NodeIds": in["NodeIds"]}, nil
}

func glueGetWorkflowRunProperties(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	run, err := st.find("workflowrun", []any{in["Name"], in["RunId"]})
	if err != nil {
		return nil, err
	}
	return glueRecord{"RunProperties": run["WorkflowRunProperties"]}, nil
}

func gluePutWorkflowRunProperties(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	run, err := st.find("workflowrun", []any{in["Name"], in["RunId"]})
	if err != nil {
		return nil, err
	}
	props, _ := run["WorkflowRunProperties"].(map[string]any)
	if props == nil {
		props = map[string]any{}
	}
	for k, v := range glueTags(in["RunProperties"]) {
		props[k] = v
	}
	run["WorkflowRunProperties"] = props
	return glueRecord{}, nil
}

// Blueprints.

func glueCreateBlueprint(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	rec, err := st.createFrom(glueOps["CreateBlueprint"], in)
	if err != nil {
		return nil, err
	}
	rec["Status"], rec["ParameterSpec"] = "ACTIVE", "{}"
	rec["BlueprintServiceLocation"] = rec["BlueprintLocation"]
	return glueRecord{"Name": rec["Name"]}, nil
}

func glueStartBlueprintRun(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	bp, err := st.find("blueprint", []any{in["BlueprintName"]})
	if err != nil {
		return nil, err
	}
	if bp["Status"] != "ACTIVE" {
		return nil, glueErr("IllegalBlueprintStateException", "blueprint %v is %v", in["BlueprintName"], bp["Status"])
	}
	id := st.newID("")
	rec, err := st.create("blueprintrun", glueRecord{
		"BlueprintName": in["BlueprintName"], "RunId": id, "State": "RUNNING", "RoleArn": in["RoleArn"],
		"Parameters": in["Parameters"], "WorkflowName": fmt.Sprintf("%v-%s", in["BlueprintName"], id[len(id)-6:]),
	}, nil)
	if err != nil {
		return nil, err
	}
	return glueRecord{"RunId": rec["RunId"]}, nil
}

func (st *glueStore) advanceBlueprintRuns(name any) {
	for _, r := range st.all("blueprintrun") {
		if r["BlueprintName"] == name && glueProgress(r, "State", map[string]string{"RUNNING": "SUCCEEDED"}) {
			r["CompletedOn"] = glueEpoch(time.Now())
			_, _ = st.create("workflow", glueRecord{"Name": r["WorkflowName"], "Description": fmt.Sprintf("Created by blueprint %v", name),
				"BlueprintDetails": glueRecord{"BlueprintName": name, "RunId": r["RunId"]}}, nil)
		}
	}
}

// Dev endpoints.

func glueCreateDevEndpoint(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	rec, err := st.createFrom(glueOps["CreateDevEndpoint"], in)
	if err != nil {
		return nil, err
	}
	rec["Status"], rec["AvailabilityZone"], rec["ZeppelinRemoteSparkInterpreterPort"] = "READY", glueRegion+"a", float64(9007)
	rec["YarnEndpointAddress"] = fmt.Sprintf("ip-10-0-0-%d.ec2.internal", st.seq%250+1)
	rec["PrivateAddress"] = fmt.Sprintf("10.0.0.%d", st.seq%250+1)
	if rec["NumberOfNodes"] == nil && rec["WorkerType"] == nil {
		rec["NumberOfNodes"] = float64(5)
	}
	return rec, nil
}

func glueUpdateDevEndpoint(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	rec, err := st.find("devendpoint", []any{in["EndpointName"]})
	if err != nil {
		return nil, err
	}
	args := glueTags(rec["Arguments"])
	for k, v := range glueTags(in["AddArguments"]) {
		args[k] = v
	}
	for _, k := range glueList(in["DeleteArguments"]) {
		delete(args, fmt.Sprint(k))
	}
	argMap := map[string]any{}
	for k, v := range args {
		argMap[k] = v
	}
	rec["Arguments"] = argMap
	keys := glueList(rec["PublicKeys"])
	keys = append(keys, glueList(in["AddPublicKeys"])...)
	var kept []any
	for _, k := range keys {
		if !glueContainsAny(glueList(in["DeletePublicKeys"]), k) {
			kept = append(kept, k)
		}
	}
	rec["PublicKeys"] = glueNonNil(kept)
	if pk := in["PublicKey"]; pk != nil {
		rec["PublicKey"] = pk
	}
	if libs, ok := in["CustomLibraries"].(map[string]any); ok {
		for k, v := range libs {
			rec[k] = v
		}
	}
	rec["LastUpdateStatus"] = "COMPLETED"
	glueTouch(rec)
	return glueRecord{}, nil
}

// ML transforms and their task runs.

func glueCreateMLTransform(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	rec, err := st.createFrom(glueOps["CreateMLTransform"], in)
	if err != nil {
		return nil, err
	}
	rec["Status"], rec["LabelCount"] = "NOT_READY", float64(0)
	return glueRecord{"TransformId": rec["TransformId"]}, nil
}

func glueStartMLTaskRun(taskType string) glueHandler {
	return func(st *glueStore, in glueRecord) (glueRecord, *glueError) {
		props := glueRecord{"TaskType": taskType}
		switch taskType {
		case "IMPORT_LABELS":
			props["ImportLabelsTaskRunProperties"] = glueRecord{"InputS3Path": in["InputS3Path"], "Replace": in["ReplaceAllLabels"] == true}
		case "EXPORT_LABELS":
			props["ExportLabelsTaskRunProperties"] = glueRecord{"OutputS3Path": in["OutputS3Path"]}
		case "LABELING_SET_GENERATION":
			props["LabelingSetGenerationTaskRunProperties"] = glueRecord{"OutputS3Path": in["OutputS3Path"]}
		}
		rec, err := st.create("mltaskrun", glueRecord{"TransformId": in["TransformId"], "Status": "RUNNING", "Properties": props, "LogGroupName": "/aws-glue/ml"}, nil)
		if err != nil {
			return nil, err
		}
		return glueRecord{"TaskRunId": rec["TaskRunId"]}, nil
	}
}

func (st *glueStore) advanceMLTaskRuns(transformID any) {
	tf, err := st.find("mltransform", []any{transformID})
	if err != nil {
		return
	}
	for _, r := range st.all("mltaskrun") {
		if r["TransformId"] != transformID || !glueProgress(r, "Status", map[string]string{"RUNNING": "SUCCEEDED", "STOPPING": "STOPPED"}) {
			continue
		}
		r["CompletedOn"], r["ExecutionTime"] = glueEpoch(time.Now()), glueSince(r, "StartedOn")
		if r["Status"] != "SUCCEEDED" {
			continue
		}
		props, _ := r["Properties"].(glueRecord)
		switch props["TaskType"] {
		case "IMPORT_LABELS":
			n, _ := tf["LabelCount"].(float64)
			tf["LabelCount"] = n + 10
		case "EVALUATION":
			tf["Status"] = "READY"
			tf["EvaluationMetrics"] = glueRecord{"TransformType": "FIND_MATCHES", "FindMatchesMetrics": glueRecord{
				"AreaUnderPRCurve": 1.0, "Precision": 1.0, "Recall": 1.0, "F1": 1.0,
			}}
		}
	}
}

func glueCancelMLTaskRun(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	run, err := st.find("mltaskrun", []any{in["TransformId"], in["TaskRunId"]})
	if err != nil {
		return nil, err
	}
	if run["Status"] == "RUNNING" || run["Status"] == "STARTING" {
		run["Status"], run["_reads"] = "STOPPING", 0
	}
	return glueRecord{"TransformId": run["TransformId"], "TaskRunId": run["TaskRunId"], "Status": run["Status"]}, nil
}

// Shared helpers.

func glueContainsAny(list []any, v any) bool {
	for _, item := range list {
		if item == v {
			return true
		}
	}
	return false
}

func glueHasTags(have, want map[string]string) bool {
	for k, v := range want {
		if have[k] != v {
			return false
		}
	}
	return true
}
