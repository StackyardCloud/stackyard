package server

import (
	"net/http"
	"testing"
)

// ETL: job runs and bookmarks, crawlers, triggers, workflows, blueprints and
// ML transforms. Runs report their in-flight state once, then settle.

func TestGlueStage2JobRunsAndBookmarks(t *testing.T) {
	_, ts := newGlueTestServer(t)
	if code := glueErrorType(t, ts, "StartJobRun", `{"JobName":"nope"}`); code != "EntityNotFoundException" {
		t.Fatalf("run of a missing job: %s", code)
	}
	out := glueCall(t, ts, http.StatusOK, "CreateJob", `{"Name":"etl","Role":"arn:aws:iam::123456789012:role/glue","Command":{"Name":"glueetl","ScriptLocation":"s3://b/etl.py"},
	  "DefaultArguments":{"--job-bookmark-option":"job-bookmark-enable"},"GlueVersion":"4.0","Tags":{"env":"dev"}}`)
	if out["Name"] != "etl" {
		t.Fatalf("create job: %v", out)
	}
	if code := glueErrorType(t, ts, "CreateJob", `{"Name":"etl","Role":"r","Command":{"Name":"glueetl"}}`); code != "AlreadyExistsException" {
		t.Fatalf("duplicate job: %s", code)
	}

	runID := glueCall(t, ts, http.StatusOK, "StartJobRun", `{"JobName":"etl","Arguments":{"--day":"1"}}`)["JobRunId"].(string)
	// The default MaxConcurrentRuns is 1.
	if code := glueErrorType(t, ts, "StartJobRun", `{"JobName":"etl"}`); code != "ConcurrentRunsExceededException" {
		t.Fatalf("second concurrent run: %s", code)
	}
	run := glueCall(t, ts, http.StatusOK, "GetJobRun", `{"JobName":"etl","RunId":"`+runID+`"}`)["JobRun"].(map[string]any)
	args := run["Arguments"].(map[string]any)
	if run["JobRunState"] != "RUNNING" || args["--day"] != "1" || args["--job-bookmark-option"] != "job-bookmark-enable" || run["GlueVersion"] != "4.0" {
		t.Fatalf("first read: %v", run)
	}
	run = glueCall(t, ts, http.StatusOK, "GetJobRun", `{"JobName":"etl","RunId":"`+runID+`"}`)["JobRun"].(map[string]any)
	if run["JobRunState"] != "SUCCEEDED" || run["CompletedOn"] == nil {
		t.Fatalf("second read: %v", run)
	}
	bm := glueCall(t, ts, http.StatusOK, "GetJobBookmark", `{"JobName":"etl"}`)["JobBookmarkEntry"].(map[string]any)
	if bm["Run"] != 1.0 || bm["RunId"] != runID {
		t.Fatalf("bookmark after a run: %v", bm)
	}
	bm = glueCall(t, ts, http.StatusOK, "ResetJobBookmark", `{"JobName":"etl"}`)["JobBookmarkEntry"].(map[string]any)
	if bm["Run"] != 0.0 {
		t.Fatalf("reset bookmark: %v", bm)
	}

	// Stopping: running runs stop, finished ones are reported as errors.
	second := glueCall(t, ts, http.StatusOK, "StartJobRun", `{"JobName":"etl"}`)["JobRunId"].(string)
	res := glueCall(t, ts, http.StatusOK, "BatchStopJobRun", `{"JobName":"etl","JobRunIds":["`+second+`","`+runID+`"]}`)
	if len(res["SuccessfulSubmissions"].([]any)) != 1 || len(res["Errors"].([]any)) != 1 {
		t.Fatalf("batch stop: %v", res)
	}
	glueCall(t, ts, http.StatusOK, "GetJobRuns", `{"JobName":"etl"}`)
	runs := glueCall(t, ts, http.StatusOK, "GetJobRuns", `{"JobName":"etl"}`)["JobRuns"].([]any)
	states := map[any]int{}
	for _, r := range runs {
		states[r.(map[string]any)["JobRunState"]]++
	}
	if states["STOPPED"] != 1 || states["SUCCEEDED"] != 1 {
		t.Fatalf("run states: %v", states)
	}

	glueCall(t, ts, http.StatusOK, "UpdateJob", `{"JobName":"etl","JobUpdate":{"Role":"r2","Command":{"Name":"glueetl"},"ExecutionProperty":{"MaxConcurrentRuns":2}}}`)
	job := glueCall(t, ts, http.StatusOK, "GetJob", `{"JobName":"etl"}`)["Job"].(map[string]any)
	if job["Role"] != "r2" {
		t.Fatalf("updated job: %v", job)
	}
	glueCall(t, ts, http.StatusOK, "StartJobRun", `{"JobName":"etl"}`)
	glueCall(t, ts, http.StatusOK, "StartJobRun", `{"JobName":"etl"}`)
	if names := glueCall(t, ts, http.StatusOK, "ListJobs", `{"Tags":{"env":"dev"}}`)["JobNames"].([]any); len(names) != 1 {
		t.Fatalf("list by tag: %v", names)
	}
	got := glueCall(t, ts, http.StatusOK, "BatchGetJobs", `{"JobNames":["etl","ghost"]}`)
	if len(got["Jobs"].([]any)) != 1 || got["JobsNotFound"].([]any)[0] != "ghost" {
		t.Fatalf("batch get: %v", got)
	}
	glueCall(t, ts, http.StatusOK, "DeleteJob", `{"JobName":"etl"}`)
	if code := glueErrorType(t, ts, "GetJobRuns", `{"JobName":"etl"}`); code != "EntityNotFoundException" {
		t.Fatalf("runs of a deleted job: %s", code)
	}
}

func TestGlueStage2CrawlersAndSchedules(t *testing.T) {
	_, ts := newGlueTestServer(t)
	glueCall(t, ts, http.StatusOK, "CreateDatabase", `{"DatabaseInput":{"Name":"lake"}}`)
	if code := glueErrorType(t, ts, "CreateCrawler", `{"Name":"c","Role":"r","Targets":{}}`); code != "InvalidInputException" {
		t.Fatalf("crawler without targets: %s", code)
	}
	glueCall(t, ts, http.StatusOK, "CreateCrawler", `{"Name":"c","Role":"r","DatabaseName":"lake","TablePrefix":"raw_","Targets":{"S3Targets":[{"Path":"s3://bucket/clicks/"}]},"Schedule":"cron(0 1 * * ? *)"}`)
	if code := glueErrorType(t, ts, "StopCrawler", `{"Name":"c"}`); code != "CrawlerNotRunningException" {
		t.Fatalf("stopping an idle crawler: %s", code)
	}
	glueCall(t, ts, http.StatusOK, "StartCrawler", `{"Name":"c"}`)
	if code := glueErrorType(t, ts, "StartCrawler", `{"Name":"c"}`); code != "CrawlerRunningException" {
		t.Fatalf("starting a running crawler: %s", code)
	}
	if code := glueErrorType(t, ts, "DeleteCrawler", `{"Name":"c"}`); code != "CrawlerRunningException" {
		t.Fatalf("deleting a running crawler: %s", code)
	}
	if c := glueCall(t, ts, http.StatusOK, "GetCrawler", `{"Name":"c"}`)["Crawler"].(map[string]any); c["State"] != "RUNNING" {
		t.Fatalf("first read: %v", c)
	}
	c := glueCall(t, ts, http.StatusOK, "GetCrawler", `{"Name":"c"}`)["Crawler"].(map[string]any)
	if c["State"] != "READY" || c["LastCrawl"].(map[string]any)["Status"] != "SUCCEEDED" {
		t.Fatalf("settled crawler: %v", c)
	}
	// The crawl registered a table for its S3 target.
	table := glueCall(t, ts, http.StatusOK, "GetTable", `{"DatabaseName":"lake","Name":"raw_clicks"}`)["Table"].(map[string]any)
	if table["StorageDescriptor"].(map[string]any)["Location"] != "s3://bucket/clicks/" {
		t.Fatalf("crawled table: %v", table)
	}
	crawls := glueCall(t, ts, http.StatusOK, "ListCrawls", `{"CrawlerName":"c"}`)["Crawls"].([]any)
	if len(crawls) != 1 || crawls[0].(map[string]any)["State"] != "COMPLETED" {
		t.Fatalf("crawl history: %v", crawls)
	}

	// Stopping a crawl cancels it.
	glueCall(t, ts, http.StatusOK, "StartCrawler", `{"Name":"c"}`)
	glueCall(t, ts, http.StatusOK, "StopCrawler", `{"Name":"c"}`)
	if code := glueErrorType(t, ts, "StopCrawler", `{"Name":"c"}`); code != "CrawlerStoppingException" {
		t.Fatalf("stopping twice: %s", code)
	}
	glueCall(t, ts, http.StatusOK, "GetCrawlers", `{}`)
	c = glueCall(t, ts, http.StatusOK, "GetCrawlers", `{}`)["Crawlers"].([]any)[0].(map[string]any)
	if c["State"] != "READY" || c["LastCrawl"].(map[string]any)["Status"] != "CANCELLED" {
		t.Fatalf("cancelled crawl: %v", c)
	}

	// The create-time schedule is running; stop, restart, update it.
	sched := c["Schedule"].(map[string]any)
	if sched["State"] != "SCHEDULED" || sched["ScheduleExpression"] != "cron(0 1 * * ? *)" {
		t.Fatalf("schedule: %v", sched)
	}
	if code := glueErrorType(t, ts, "StartCrawlerSchedule", `{"CrawlerName":"c"}`); code != "SchedulerRunningException" {
		t.Fatalf("starting a running schedule: %s", code)
	}
	glueCall(t, ts, http.StatusOK, "StopCrawlerSchedule", `{"CrawlerName":"c"}`)
	if code := glueErrorType(t, ts, "StopCrawlerSchedule", `{"CrawlerName":"c"}`); code != "SchedulerNotRunningException" {
		t.Fatalf("stopping a stopped schedule: %s", code)
	}
	glueCall(t, ts, http.StatusOK, "UpdateCrawlerSchedule", `{"CrawlerName":"c","Schedule":"cron(0 2 * * ? *)"}`)
	glueCall(t, ts, http.StatusOK, "CreateCrawler", `{"Name":"adhoc","Role":"r","Targets":{"S3Targets":[{"Path":"s3://b/x"}]}}`)
	if code := glueErrorType(t, ts, "StartCrawlerSchedule", `{"CrawlerName":"adhoc"}`); code != "NoScheduleException" {
		t.Fatalf("schedule for an unscheduled crawler: %s", code)
	}
	metrics := glueCall(t, ts, http.StatusOK, "GetCrawlerMetrics", `{"CrawlerNameList":["c"]}`)["CrawlerMetricsList"].([]any)
	if len(metrics) != 1 || metrics[0].(map[string]any)["TablesCreated"] != 1.0 {
		t.Fatalf("metrics: %v", metrics)
	}
}

func TestGlueStage2TriggersWorkflowsAndBlueprints(t *testing.T) {
	_, ts := newGlueTestServer(t)
	glueCall(t, ts, http.StatusOK, "CreateJob", `{"Name":"load","Role":"r","Command":{"Name":"glueetl"}}`)
	glueCall(t, ts, http.StatusOK, "CreateWorkflow", `{"Name":"nightly","DefaultRunProperties":{"env":"prod"},"MaxConcurrentRuns":1}`)

	if code := glueErrorType(t, ts, "CreateTrigger", `{"Name":"t","Type":"SCHEDULED","Actions":[{"JobName":"load"}]}`); code != "InvalidInputException" {
		t.Fatalf("scheduled trigger without a schedule: %s", code)
	}
	if code := glueErrorType(t, ts, "CreateTrigger", `{"Name":"t","Type":"ON_DEMAND","Actions":[{"JobName":"ghost"}]}`); code != "EntityNotFoundException" {
		t.Fatalf("trigger for a missing job: %s", code)
	}
	glueCall(t, ts, http.StatusOK, "CreateTrigger", `{"Name":"kick","Type":"ON_DEMAND","WorkflowName":"nightly","Actions":[{"JobName":"load"}]}`)
	glueCall(t, ts, http.StatusOK, "CreateTrigger", `{"Name":"cron","Type":"SCHEDULED","Schedule":"cron(0 3 * * ? *)","StartOnCreation":true,"Actions":[{"JobName":"load"}]}`)
	if tr := glueCall(t, ts, http.StatusOK, "GetTrigger", `{"Name":"cron"}`)["Trigger"].(map[string]any); tr["State"] != "ACTIVATED" {
		t.Fatalf("StartOnCreation: %v", tr)
	}
	glueCall(t, ts, http.StatusOK, "StopTrigger", `{"Name":"cron"}`)
	if code := glueErrorType(t, ts, "StopTrigger", `{"Name":"kick"}`); code != "InvalidInputException" {
		t.Fatalf("stopping an on-demand trigger: %s", code)
	}
	if names := glueCall(t, ts, http.StatusOK, "ListTriggers", `{"DependentJobName":"load"}`)["TriggerNames"].([]any); len(names) != 2 {
		t.Fatalf("dependent triggers: %v", names)
	}

	// The workflow graph links the trigger to its job.
	wf := glueCall(t, ts, http.StatusOK, "GetWorkflow", `{"Name":"nightly","IncludeGraph":true}`)["Workflow"].(map[string]any)
	if g := wf["Graph"].(map[string]any); len(g["Nodes"].([]any)) != 2 || len(g["Edges"].([]any)) != 1 {
		t.Fatalf("graph: %v", wf)
	}
	runID := glueCall(t, ts, http.StatusOK, "StartWorkflowRun", `{"Name":"nightly","RunProperties":{"day":"mon"}}`)["RunId"].(string)
	if code := glueErrorType(t, ts, "StartWorkflowRun", `{"Name":"nightly"}`); code != "ConcurrentRunsExceededException" {
		t.Fatalf("MaxConcurrentRuns: %s", code)
	}
	// Starting the run fired the workflow's on-demand trigger.
	if runs := glueCall(t, ts, http.StatusOK, "GetJobRuns", `{"JobName":"load"}`)["JobRuns"].([]any); len(runs) != 1 || runs[0].(map[string]any)["TriggerName"] != "kick" {
		t.Fatalf("job runs from the trigger: %v", runs)
	}
	props := glueCall(t, ts, http.StatusOK, "GetWorkflowRunProperties", `{"Name":"nightly","RunId":"`+runID+`"}`)["RunProperties"].(map[string]any)
	if props["env"] != "prod" || props["day"] != "mon" {
		t.Fatalf("run properties: %v", props)
	}
	glueCall(t, ts, http.StatusOK, "PutWorkflowRunProperties", `{"Name":"nightly","RunId":"`+runID+`","RunProperties":{"step":"2"}}`)
	glueCall(t, ts, http.StatusOK, "StopWorkflowRun", `{"Name":"nightly","RunId":"`+runID+`"}`)
	glueCall(t, ts, http.StatusOK, "GetWorkflowRun", `{"Name":"nightly","RunId":"`+runID+`"}`)
	run := glueCall(t, ts, http.StatusOK, "GetWorkflowRun", `{"Name":"nightly","RunId":"`+runID+`"}`)["Run"].(map[string]any)
	if run["Status"] != "STOPPED" || run["WorkflowRunProperties"].(map[string]any)["step"] != "2" {
		t.Fatalf("stopped run: %v", run)
	}
	if code := glueErrorType(t, ts, "StopWorkflowRun", `{"Name":"nightly","RunId":"`+runID+`"}`); code != "IllegalWorkflowStateException" {
		t.Fatalf("stopping a stopped run: %s", code)
	}
	resumed := glueCall(t, ts, http.StatusOK, "ResumeWorkflowRun", `{"Name":"nightly","RunId":"`+runID+`","NodeIds":["JOB_load"]}`)
	if resumed["RunId"] == runID || resumed["RunId"] == nil {
		t.Fatalf("resume: %v", resumed)
	}
	glueCall(t, ts, http.StatusOK, "DeleteWorkflow", `{"Name":"nightly"}`)
	if code := glueErrorType(t, ts, "GetTrigger", `{"Name":"kick"}`); code != "EntityNotFoundException" {
		t.Fatalf("workflow trigger after delete: %s", code)
	}

	// Blueprint runs create their workflow.
	glueCall(t, ts, http.StatusOK, "CreateBlueprint", `{"Name":"bp","BlueprintLocation":"s3://b/bp.zip"}`)
	bpRun := glueCall(t, ts, http.StatusOK, "StartBlueprintRun", `{"BlueprintName":"bp","RoleArn":"arn:aws:iam::123456789012:role/r"}`)["RunId"].(string)
	glueCall(t, ts, http.StatusOK, "GetBlueprintRun", `{"BlueprintName":"bp","RunId":"`+bpRun+`"}`)
	br := glueCall(t, ts, http.StatusOK, "GetBlueprintRun", `{"BlueprintName":"bp","RunId":"`+bpRun+`"}`)["BlueprintRun"].(map[string]any)
	if br["State"] != "SUCCEEDED" {
		t.Fatalf("blueprint run: %v", br)
	}
	glueCall(t, ts, http.StatusOK, "GetWorkflow", `{"Name":"`+br["WorkflowName"].(string)+`"}`)
}

func TestGlueStage2MLTransformsAndDevEndpoints(t *testing.T) {
	_, ts := newGlueTestServer(t)
	id := glueCall(t, ts, http.StatusOK, "CreateMLTransform", `{"Name":"dedupe","Role":"r","InputRecordTables":[{"DatabaseName":"d","TableName":"t"}],"Parameters":{"TransformType":"FIND_MATCHES","FindMatchesParameters":{"PrimaryKeyColumnName":"id"}}}`)["TransformId"].(string)
	if tf := glueCall(t, ts, http.StatusOK, "GetMLTransform", `{"TransformId":"`+id+`"}`); tf["Status"] != "NOT_READY" || tf["Name"] != "dedupe" {
		t.Fatalf("transform: %v", tf)
	}
	task := glueCall(t, ts, http.StatusOK, "StartMLEvaluationTaskRun", `{"TransformId":"`+id+`"}`)["TaskRunId"].(string)
	glueCall(t, ts, http.StatusOK, "GetMLTaskRun", `{"TransformId":"`+id+`","TaskRunId":"`+task+`"}`)
	if tr := glueCall(t, ts, http.StatusOK, "GetMLTaskRun", `{"TransformId":"`+id+`","TaskRunId":"`+task+`"}`); tr["Status"] != "SUCCEEDED" {
		t.Fatalf("task run: %v", tr)
	}
	if tf := glueCall(t, ts, http.StatusOK, "GetMLTransform", `{"TransformId":"`+id+`"}`); tf["Status"] != "READY" || tf["EvaluationMetrics"] == nil {
		t.Fatalf("evaluated transform: %v", tf)
	}
	if code := glueErrorType(t, ts, "StartMLEvaluationTaskRun", `{"TransformId":"tfm-missing"}`); code != "EntityNotFoundException" {
		t.Fatalf("task for a missing transform: %s", code)
	}

	ep := glueCall(t, ts, http.StatusOK, "CreateDevEndpoint", `{"EndpointName":"dev","RoleArn":"arn:aws:iam::123456789012:role/r","Arguments":{"a":"1"}}`)
	if ep["Status"] != "READY" || ep["EndpointName"] != "dev" {
		t.Fatalf("dev endpoint: %v", ep)
	}
	glueCall(t, ts, http.StatusOK, "UpdateDevEndpoint", `{"EndpointName":"dev","AddArguments":{"b":"2"},"DeleteArguments":["a"],"AddPublicKeys":["ssh-rsa AAA"]}`)
	got := glueCall(t, ts, http.StatusOK, "GetDevEndpoint", `{"EndpointName":"dev"}`)["DevEndpoint"].(map[string]any)
	if args := got["Arguments"].(map[string]any); args["b"] != "2" || args["a"] != nil || len(got["PublicKeys"].([]any)) != 1 {
		t.Fatalf("updated dev endpoint: %v", got)
	}
}

func TestGlueStage2JobUpgradeAnalysis(t *testing.T) {
	_, ts := newGlueTestServer(t)
	glueCall(t, ts, http.StatusOK, "CreateJob", `{"Name":"old","Role":"r","Command":{"Name":"glueetl"},"GlueVersion":"3.0"}`)
	id := glueCall(t, ts, http.StatusOK, "StartJobUpgradeAnalysis", `{"jobName":"old"}`)["analysisId"].(string)
	if len(id) != 39 {
		t.Fatalf("analysisId %q must be 39 characters", id)
	}
	a := glueCall(t, ts, http.StatusOK, "GetJobUpgradeAnalysis", `{"jobName":"old","analysisId":"`+id+`"}`)
	if a["status"] != "OBSERVING" || a["sourceVersion"] != "3.0" {
		t.Fatalf("analysis: %v", a)
	}
	glueCall(t, ts, http.StatusOK, "StopJobUpgradeAnalysis", `{"jobName":"old","analysisId":"`+id+`"}`)
	if code := glueErrorType(t, ts, "StopJobUpgradeAnalysis", `{"jobName":"old","analysisId":"`+id+`"}`); code != "IllegalSessionStateException" {
		t.Fatalf("stopping a stopped analysis: %s", code)
	}
}
