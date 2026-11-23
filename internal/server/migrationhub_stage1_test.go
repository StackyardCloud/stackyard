package server

import (
	"fmt"
	"testing"
)

const mghTestTask = `"ProgressUpdateStream":"tool","MigrationTaskName":"task-1"`

func TestMigrationHubStage1StreamsAndTasks(t *testing.T) {
	_, ts := newMigrationHubTestServer(t)
	if code := mghErrCode(t, ts, "ImportMigrationTask", `{`+mghTestTask+`}`); code != "ResourceNotFoundException" {
		t.Fatalf("task on missing stream: %s", code)
	}
	if code := mghErrCode(t, ts, "CreateProgressUpdateStream", `{"ProgressUpdateStreamName":"tool","DryRun":true}`); code != "DryRunOperation" {
		t.Fatalf("dry run: %s", code)
	}
	if out := mghOK(t, ts, "ListProgressUpdateStreams", `{}`); len(out["ProgressUpdateStreamSummaryList"].([]any)) != 0 {
		t.Fatalf("dry run created a stream: %v", out)
	}
	mghOK(t, ts, "CreateProgressUpdateStream", `{"ProgressUpdateStreamName":"tool"}`)
	mghOK(t, ts, "CreateProgressUpdateStream", `{"ProgressUpdateStreamName":"tool"}`) // idempotent
	if code := mghErrCode(t, ts, "CreateProgressUpdateStream", `{"ProgressUpdateStreamName":"bad/name"}`); code != "InvalidInputException" {
		t.Fatalf("bad stream name: %s", code)
	}
	mghOK(t, ts, "ImportMigrationTask", `{`+mghTestTask+`}`)
	task := mghOK(t, ts, "DescribeMigrationTask", `{`+mghTestTask+`}`)["MigrationTask"].(map[string]any)
	if task["Task"].(map[string]any)["Status"] != "NOT_STARTED" {
		t.Fatalf("imported task: %v", task)
	}

	notify := func(at, status string, pct int) string {
		return fmt.Sprintf(`{%s,"Task":{"Status":%q,"ProgressPercent":%d,"StatusDetail":"copying"},"UpdateDateTime":%s,"NextUpdateSeconds":60}`, mghTestTask, status, pct, at)
	}
	mghOK(t, ts, "NotifyMigrationTaskState", notify("1700000100", "IN_PROGRESS", 40))
	mghOK(t, ts, "NotifyMigrationTaskState", notify("1700000200", "COMPLETED", 100))
	// A stale update can't roll the task back.
	if code := mghErrCode(t, ts, "NotifyMigrationTaskState", notify("1700000150", "FAILED", 50)); code != "InvalidInputException" {
		t.Fatalf("stale update: %s", code)
	}
	if code := mghErrCode(t, ts, "NotifyMigrationTaskState", notify("1700000300", "IN_PROGRESS", 101)); code != "InvalidInputException" {
		t.Fatalf("percent over 100: %s", code)
	}
	task = mghOK(t, ts, "DescribeMigrationTask", `{`+mghTestTask+`}`)["MigrationTask"].(map[string]any)
	state := task["Task"].(map[string]any)
	if state["Status"] != "COMPLETED" || state["ProgressPercent"] != float64(100) || task["UpdateDateTime"] != float64(1700000200) {
		t.Fatalf("task after updates: %v", task)
	}
	updates := mghOK(t, ts, "ListMigrationTaskUpdates", `{`+mghTestTask+`,"MaxResults":1}`)
	if len(updates["MigrationTaskUpdateList"].([]any)) != 1 || updates["NextToken"] != "1" {
		t.Fatalf("updates page: %v", updates)
	}
	updates = mghOK(t, ts, "ListMigrationTaskUpdates", `{`+mghTestTask+`,"NextToken":"1"}`)
	if u := updates["MigrationTaskUpdateList"].([]any); len(u) != 1 || u[0].(map[string]any)["UpdateType"] != "MIGRATION_TASK_STATE_UPDATED" {
		t.Fatalf("second updates page: %v", updates)
	}
	if code := mghErrCode(t, ts, "ListMigrationTaskUpdates", `{`+mghTestTask+`,"NextToken":"99"}`); code != "InvalidInputException" {
		t.Fatalf("bad token: %s", code)
	}

	// Resource attributes are validated per type and feed the ResourceName filter.
	if code := mghErrCode(t, ts, "PutResourceAttributes", `{`+mghTestTask+`,"ResourceAttributeList":[{"Type":"IPV4_ADDRESS","Value":"not-an-ip"}]}`); code != "InvalidInputException" {
		t.Fatalf("bad IPv4: %s", code)
	}
	mghOK(t, ts, "PutResourceAttributes", `{`+mghTestTask+`,"ResourceAttributeList":[{"Type":"IPV4_ADDRESS","Value":"10.0.0.5"},{"Type":"FQDN","Value":"web.example.com"}]}`)
	if tasks := mghOK(t, ts, "ListMigrationTasks", `{"ResourceName":"web.example.com"}`)["MigrationTaskSummaryList"].([]any); len(tasks) != 1 {
		t.Fatalf("ResourceName filter: %v", tasks)
	}
	if tasks := mghOK(t, ts, "ListMigrationTasks", `{"ResourceName":"other"}`)["MigrationTaskSummaryList"].([]any); len(tasks) != 0 {
		t.Fatalf("ResourceName filter matched other: %v", tasks)
	}

	// Deleting a stream deletes its tasks.
	mghOK(t, ts, "DeleteProgressUpdateStream", `{"ProgressUpdateStreamName":"tool"}`)
	if code := mghErrCode(t, ts, "DescribeMigrationTask", `{`+mghTestTask+`}`); code != "ResourceNotFoundException" {
		t.Fatalf("task after stream delete: %s", code)
	}
	if code := mghErrCode(t, ts, "DeleteProgressUpdateStream", `{"ProgressUpdateStreamName":"tool"}`); code != "ResourceNotFoundException" {
		t.Fatalf("delete missing stream: %s", code)
	}
}

func TestMigrationHubStage1AssociationsAndApplications(t *testing.T) {
	_, ts := newMigrationHubTestServer(t)
	mghOK(t, ts, "CreateProgressUpdateStream", `{"ProgressUpdateStreamName":"tool"}`)
	mghOK(t, ts, "ImportMigrationTask", `{`+mghTestTask+`}`)

	artifact := `"arn:aws:ec2:us-east-1:123456789012:instance/i-0123456789abcdef0"`
	if code := mghErrCode(t, ts, "AssociateCreatedArtifact", `{`+mghTestTask+`,"CreatedArtifact":{"Name":"not-an-arn"}}`); code != "InvalidInputException" {
		t.Fatalf("non-ARN artifact: %s", code)
	}
	mghOK(t, ts, "AssociateCreatedArtifact", `{`+mghTestTask+`,"CreatedArtifact":{"Name":`+artifact+`,"Description":"web"}}`)
	mghOK(t, ts, "AssociateCreatedArtifact", `{`+mghTestTask+`,"CreatedArtifact":{"Name":`+artifact+`,"Description":"web v2"}}`)
	arts := mghOK(t, ts, "ListCreatedArtifacts", `{`+mghTestTask+`}`)["CreatedArtifactList"].([]any)
	if len(arts) != 1 || arts[0].(map[string]any)["Description"] != "web v2" {
		t.Fatalf("artifacts: %v", arts)
	}
	if code := mghErrCode(t, ts, "DisassociateCreatedArtifact", `{`+mghTestTask+`,"CreatedArtifactName":`+artifact+`,"DryRun":true}`); code != "DryRunOperation" {
		t.Fatalf("dry-run disassociate: %s", code)
	}
	mghOK(t, ts, "DisassociateCreatedArtifact", `{`+mghTestTask+`,"CreatedArtifactName":`+artifact+`}`)
	if code := mghErrCode(t, ts, "DisassociateCreatedArtifact", `{`+mghTestTask+`,"CreatedArtifactName":`+artifact+`}`); code != "ResourceNotFoundException" {
		t.Fatalf("disassociate missing artifact: %s", code)
	}

	// Discovered resources must exist in Application Discovery Service.
	if code := mghErrCode(t, ts, "AssociateDiscoveredResource", `{`+mghTestTask+`,"DiscoveredResource":{"ConfigurationId":"srv-missing"}}`); code != "ResourceNotFoundException" {
		t.Fatalf("unknown configuration: %s", code)
	}
	mghOK(t, ts, "AssociateDiscoveredResource", `{`+mghTestTask+`,"DiscoveredResource":{"ConfigurationId":"srv-000001","Description":"db"}}`)
	if res := mghOK(t, ts, "ListDiscoveredResources", `{`+mghTestTask+`}`)["DiscoveredResourceList"].([]any); len(res) != 1 || res[0].(map[string]any)["ConfigurationId"] != "srv-000001" {
		t.Fatalf("discovered: %v", res)
	}
	if tasks := mghOK(t, ts, "ListMigrationTasks", `{"ResourceName":"srv-000001"}`)["MigrationTaskSummaryList"].([]any); len(tasks) != 1 {
		t.Fatalf("ResourceName by configuration id: %v", tasks)
	}
	mghOK(t, ts, "DisassociateDiscoveredResource", `{`+mghTestTask+`,"ConfigurationId":"srv-000001"}`)

	mghOK(t, ts, "AssociateSourceResource", `{`+mghTestTask+`,"SourceResource":{"Name":"vm-42","StatusDetail":"replicating"}}`)
	if src := mghOK(t, ts, "ListSourceResources", `{`+mghTestTask+`}`)["SourceResourceList"].([]any); len(src) != 1 || src[0].(map[string]any)["StatusDetail"] != "replicating" {
		t.Fatalf("sources: %v", src)
	}
	mghOK(t, ts, "DisassociateSourceResource", `{`+mghTestTask+`,"SourceResourceName":"vm-42"}`)

	// Application states are keyed by Application Discovery Service applications.
	if code := mghErrCode(t, ts, "NotifyApplicationState", `{"ApplicationId":"app-missing","Status":"IN_PROGRESS"}`); code != "ResourceNotFoundException" {
		t.Fatalf("unknown application: %s", code)
	}
	if code := mghErrCode(t, ts, "DescribeApplicationState", `{"ApplicationId":"app-000001"}`); code != "ResourceNotFoundException" {
		t.Fatalf("application with no state: %s", code)
	}
	mghOK(t, ts, "NotifyApplicationState", `{"ApplicationId":"app-000001","Status":"IN_PROGRESS","UpdateDateTime":1700000000}`)
	if out := mghOK(t, ts, "DescribeApplicationState", `{"ApplicationId":"app-000001"}`); out["ApplicationStatus"] != "IN_PROGRESS" || out["LastUpdatedTime"] != float64(1700000000) {
		t.Fatalf("application state: %v", out)
	}
	if states := mghOK(t, ts, "ListApplicationStates", `{"ApplicationIds":["app-000001","app-nope"]}`)["ApplicationStateList"].([]any); len(states) != 1 {
		t.Fatalf("application states: %v", states)
	}
}
